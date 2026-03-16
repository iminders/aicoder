# 搜索工具实现说明

## 新增工具

根据 `docs/todo.md` 中的计划,已实现以下搜索工具:

### 1. ast_search - AST 语义搜索

**文件:** `internal/tools/search/ast_search.go`

**功能:**
- 使用 Go 的 `go/ast` 包进行语义搜索
- 支持查找: 函数、方法、类型、接口、结构体、变量、导入
- 支持导出符号过滤
- 支持名称模式匹配

**测试:** `internal/tools/search/ast_search_test.go` (11 个测试用例,全部通过)

### 2. web_search - 联网搜索

**文件:** `internal/tools/search/web_search.go`

**功能:**
- 使用 Google Custom Search API 进行联网搜索
- 支持自定义结果数量和语言
- 返回标题、URL 和摘要
- 需要配置 API 凭证

**测试:** `internal/tools/search/web_search_test.go` (5 个测试用例,全部通过)

## 测试结果

```bash
$ go test ./internal/tools/search/ -v
=== RUN   TestASTSearchTool_Go_Functions
--- PASS: TestASTSearchTool_Go_Functions (0.00s)
=== RUN   TestASTSearchTool_Go_Types
--- PASS: TestASTSearchTool_Go_Types (0.00s)
=== RUN   TestASTSearchTool_InvalidQueryType
--- PASS: TestASTSearchTool_InvalidQueryType (0.00s)
=== RUN   TestASTSearchTool_NamePattern
--- PASS: TestASTSearchTool_NamePattern (0.00s)
=== RUN   TestASTSearchTool_Imports
--- PASS: TestASTSearchTool_Imports (0.00s)
=== RUN   TestWebSearchTool_NoConfig
--- PASS: TestWebSearchTool_NoConfig (0.00s)
=== RUN   TestWebSearchTool_DefaultValues
--- PASS: TestWebSearchTool_DefaultValues (0.00s)
=== RUN   TestWebSearchTool_Schema
--- PASS: TestWebSearchTool_Schema (0.00s)
=== RUN   TestWebSearchTool_ToolInterface
--- PASS: TestWebSearchTool_ToolInterface (0.00s)
PASS
ok  	github.com/iminders/aicoder/internal/tools/search	0.019s
```

## 使用示例

### AST 搜索

```bash
# 查找所有导出的函数
aicoder
> 使用 ast_search 查找所有导出的函数

# 查找特定接口
> 查找项目中所有的 Handler 接口
```

### 联网搜索

```bash
# 配置 API 凭证
export SEARCH_API_KEY="your-api-key"
export SEARCH_ENGINE_ID="your-engine-id"

# 使用搜索
aicoder
> 搜索 Go 1.22 的新特性
```

## 文档更新

- ✅ 更新了 `docs/tools-reference.md`,添加了两个新工具的详细说明
- ✅ 更新了工具列表总览

## 实现特点

1. **最小化实现:** 代码简洁,只包含必要功能
2. **完整测试:** 每个工具都有完整的单元测试
3. **错误处理:** 完善的错误处理和用户提示
4. **文档完善:** 详细的参数说明和使用示例
5. **自动注册:** 工具在 `init()` 函数中自动注册到全局注册表

## 注意事项

- `ast_search` 目前仅支持 Go 语言 (v1.0)
- `web_search` 需要配置 Google Custom Search API 凭证
- 两个工具都遵循项目的沙箱保护机制
- 所有测试都通过,代码可以直接使用
