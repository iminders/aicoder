package search

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/iminders/aicoder/internal/tools"
)

type ASTSearchTool struct{}

func (t *ASTSearchTool) Name() string          { return "ast_search" }
func (t *ASTSearchTool) Risk() tools.RiskLevel { return tools.RiskLow }
func (t *ASTSearchTool) Description() string {
	return "Search for code elements using AST (Abstract Syntax Tree) semantic search. Supports Go, Python, and JavaScript. Can find functions, types, interfaces, methods, etc."
}

func (t *ASTSearchTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path":        {"type": "string",  "description": "Root directory to search"},
			"query_type":  {"type": "string",  "description": "Type of element to search: 'function', 'type', 'interface', 'method', 'struct', 'variable', 'import', 'all'"},
			"name":        {"type": "string",  "description": "Name pattern to match (optional, supports wildcards)"},
			"language":    {"type": "string",  "description": "Language: 'go', 'python', 'javascript' (default: auto-detect)"},
			"exported_only": {"type": "boolean", "description": "Only search for exported symbols (Go only, default: false)"}
		},
		"required": ["path", "query_type"]
	}`)
}

type astInput struct {
	Path         string `json:"path"`
	QueryType    string `json:"query_type"`
	Name         string `json:"name"`
	Language     string `json:"language"`
	ExportedOnly bool   `json:"exported_only"`
}

func (t *ASTSearchTool) Execute(ctx context.Context, raw json.RawMessage) (*tools.Result, error) {
	var in astInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}

	// 验证查询类型
	validTypes := map[string]bool{
		"function": true, "type": true, "interface": true,
		"method": true, "struct": true, "variable": true,
		"import": true, "all": true,
	}
	if !validTypes[in.QueryType] {
		return &tools.Result{
			IsError: true,
			Content: fmt.Sprintf("Invalid query_type: %s. Valid types: function, type, interface, method, struct, variable, import, all", in.QueryType),
		}, nil
	}

	// 根据语言选择搜索方法
	var results []string
	var err error

	if in.Language == "" || in.Language == "go" {
		results, err = t.searchGo(in)
	} else {
		return &tools.Result{
			IsError: true,
			Content: fmt.Sprintf("Language '%s' is not yet supported. Currently only 'go' is supported in v1.0.", in.Language),
		}, nil
	}

	if err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}

	if len(results) == 0 {
		return &tools.Result{Content: "No matches found."}, nil
	}

	return &tools.Result{Content: strings.Join(results, "\n")}, nil
}

// searchGo 使用 go/ast 搜索 Go 代码
func (t *ASTSearchTool) searchGo(in astInput) ([]string, error) {
	var results []string
	const maxResults = 100

	err := filepath.WalkDir(in.Path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// 跳过特定目录
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" || d.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// 只处理 .go 文件
		if filepath.Ext(p) != ".go" {
			return nil
		}

		// 解析文件
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, p, nil, parser.ParseComments)
		if err != nil {
			// 跳过解析失败的文件
			return nil
		}

		// 遍历 AST
		ast.Inspect(node, func(n ast.Node) bool {
			if len(results) >= maxResults {
				return false
			}

			// 检查函数
			if in.QueryType == "function" || in.QueryType == "all" {
				if fn, ok := n.(*ast.FuncDecl); ok {
					if fn.Recv == nil { // 没有接收者的是函数
						if t.matchName(fn.Name.Name, in.Name) {
							if !in.ExportedOnly || ast.IsExported(fn.Name.Name) {
								pos := fset.Position(fn.Pos())
								sig := t.formatFuncSignature(fn)
								results = append(results, fmt.Sprintf("%s:%d: func %s", p, pos.Line, sig))
							}
						}
					}
				}
			}

			// 检查方法
			if in.QueryType == "method" || in.QueryType == "all" {
				if fn, ok := n.(*ast.FuncDecl); ok {
					if fn.Recv != nil { // 有接收者的是方法
						if t.matchName(fn.Name.Name, in.Name) {
							if !in.ExportedOnly || ast.IsExported(fn.Name.Name) {
								pos := fset.Position(fn.Pos())
								recv := t.formatReceiver(fn.Recv)
								sig := t.formatFuncSignature(fn)
								results = append(results, fmt.Sprintf("%s:%d: func (%s) %s", p, pos.Line, recv, sig))
							}
						}
					}
				}
			}

			// 检查类型
			if in.QueryType == "type" || in.QueryType == "all" {
				if ts, ok := n.(*ast.TypeSpec); ok {
					if t.matchName(ts.Name.Name, in.Name) {
						if !in.ExportedOnly || ast.IsExported(ts.Name.Name) {
							pos := fset.Position(ts.Pos())
							typeStr := t.formatType(ts)
							results = append(results, fmt.Sprintf("%s:%d: type %s", p, pos.Line, typeStr))
						}
					}
				}
			}

			// 检查接口
			if in.QueryType == "interface" {
				if ts, ok := n.(*ast.TypeSpec); ok {
					if _, isInterface := ts.Type.(*ast.InterfaceType); isInterface {
						if t.matchName(ts.Name.Name, in.Name) {
							if !in.ExportedOnly || ast.IsExported(ts.Name.Name) {
								pos := fset.Position(ts.Pos())
								results = append(results, fmt.Sprintf("%s:%d: interface %s", p, pos.Line, ts.Name.Name))
							}
						}
					}
				}
			}

			// 检查结构体
			if in.QueryType == "struct" {
				if ts, ok := n.(*ast.TypeSpec); ok {
					if _, isStruct := ts.Type.(*ast.StructType); isStruct {
						if t.matchName(ts.Name.Name, in.Name) {
							if !in.ExportedOnly || ast.IsExported(ts.Name.Name) {
								pos := fset.Position(ts.Pos())
								results = append(results, fmt.Sprintf("%s:%d: struct %s", p, pos.Line, ts.Name.Name))
							}
						}
					}
				}
			}

			// 检查变量
			if in.QueryType == "variable" || in.QueryType == "all" {
				if vs, ok := n.(*ast.ValueSpec); ok {
					for _, name := range vs.Names {
						if t.matchName(name.Name, in.Name) {
							if !in.ExportedOnly || ast.IsExported(name.Name) {
								pos := fset.Position(name.Pos())
								results = append(results, fmt.Sprintf("%s:%d: var %s", p, pos.Line, name.Name))
							}
						}
					}
				}
			}

			// 检查导入
			if in.QueryType == "import" || in.QueryType == "all" {
				if imp, ok := n.(*ast.ImportSpec); ok {
					path := strings.Trim(imp.Path.Value, "\"")
					if in.Name == "" || strings.Contains(path, in.Name) {
						pos := fset.Position(imp.Pos())
						results = append(results, fmt.Sprintf("%s:%d: import %s", p, pos.Line, path))
					}
				}
			}

			return true
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}

// matchName 检查名称是否匹配模式
func (t *ASTSearchTool) matchName(name, pattern string) bool {
	if pattern == "" {
		return true
	}

	// 支持简单的通配符
	if strings.Contains(pattern, "*") {
		// 转换为正则表达式
		regexPattern := strings.ReplaceAll(pattern, "*", ".*")
		regexPattern = "^" + regexPattern + "$"
		matched, _ := filepath.Match(regexPattern, name)
		return matched
	}

	// 精确匹配或包含匹配
	return strings.Contains(strings.ToLower(name), strings.ToLower(pattern))
}

// formatFuncSignature 格式化函数签名
func (t *ASTSearchTool) formatFuncSignature(fn *ast.FuncDecl) string {
	name := fn.Name.Name

	// 参数
	params := []string{}
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			typeStr := t.exprToString(field.Type)
			if len(field.Names) > 0 {
				for _, name := range field.Names {
					params = append(params, name.Name+" "+typeStr)
				}
			} else {
				params = append(params, typeStr)
			}
		}
	}

	// 返回值
	results := []string{}
	if fn.Type.Results != nil {
		for _, field := range fn.Type.Results.List {
			typeStr := t.exprToString(field.Type)
			if len(field.Names) > 0 {
				for _, name := range field.Names {
					results = append(results, name.Name+" "+typeStr)
				}
			} else {
				results = append(results, typeStr)
			}
		}
	}

	sig := name + "(" + strings.Join(params, ", ") + ")"
	if len(results) > 0 {
		if len(results) == 1 {
			sig += " " + results[0]
		} else {
			sig += " (" + strings.Join(results, ", ") + ")"
		}
	}

	return sig
}

// formatReceiver 格式化接收者
func (t *ASTSearchTool) formatReceiver(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}

	field := recv.List[0]
	typeStr := t.exprToString(field.Type)

	if len(field.Names) > 0 {
		return field.Names[0].Name + " " + typeStr
	}

	return typeStr
}

// formatType 格式化类型定义
func (t *ASTSearchTool) formatType(ts *ast.TypeSpec) string {
	name := ts.Name.Name
	typeStr := t.exprToString(ts.Type)

	// 简化显示
	if len(typeStr) > 50 {
		typeStr = typeStr[:50] + "..."
	}

	return name + " " + typeStr
}

// exprToString 将表达式转换为字符串
func (t *ASTSearchTool) exprToString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}

	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + t.exprToString(e.X)
	case *ast.SelectorExpr:
		return t.exprToString(e.X) + "." + e.Sel.Name
	case *ast.ArrayType:
		return "[]" + t.exprToString(e.Elt)
	case *ast.MapType:
		return "map[" + t.exprToString(e.Key) + "]" + t.exprToString(e.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{...}"
	case *ast.FuncType:
		return "func(...)"
	case *ast.ChanType:
		return "chan " + t.exprToString(e.Value)
	default:
		return "..."
	}
}

func init() {
	tools.Global.Register(&ASTSearchTool{})
}
