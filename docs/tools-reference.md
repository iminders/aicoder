# 工具参考文档

**产品名称:** aicoder
**版本:** v1.0
**文档状态:** 正式版
**创建日期:** 2026-03-16
**关联文档:** [development.md](./development.md) · [code-guide.md](./code-guide.md)

---

## 目录

- [1. 工具系统概述](#1-工具系统概述)
- [2. 文件系统工具](#2-文件系统工具)
- [3. Shell 工具](#3-shell-工具)
- [4. 搜索工具](#4-搜索工具)
- [5. 风险等级说明](#5-风险等级说明)
- [6. 沙箱保护](#6-沙箱保护)

---

## 1. 工具系统概述

### 1.1 工具接口

所有工具都实现以下接口:

```go
type Tool interface {
    Name() string                // 工具名称
    Description() string         // 工具描述
    Schema() json.RawMessage     // JSON Schema 参数定义
    Execute(ctx context.Context, input json.RawMessage) (*Result, error)
    Risk() RiskLevel            // 风险等级
}
```

### 1.2 工具结果

工具执行后返回 `Result` 结构:

```go
type Result struct {
    Content  string              // 结果内容
    IsError  bool                // 是否为错误
    Metadata map[string]any      // 元数据
}
```

### 1.3 工具注册

所有工具在程序启动时注册到全局注册表:

```go
tools.Global.Register(&ReadFileTool{})
tools.Global.Register(&WriteFileTool{})
// ...
```

---

## 2. 文件系统工具

### 2.1 read_file - 读取文件

**描述:** 读取文件内容,支持按行范围读取

**风险等级:** 低 (RiskLow)

**参数:**

| 参数名 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `path` | string | 是 | 文件路径 |
| `start_line` | integer | 否 | 起始行号 (从 1 开始) |
| `end_line` | integer | 否 | 结束行号 (包含) |

**返回:**

- 成功: 文件内容 (字符串)
- 失败: 错误信息

**示例:**

```json
// 读取整个文件
{
  "path": "src/main.go"
}

// 读取指定行范围
{
  "path": "src/main.go",
  "start_line": 10,
  "end_line": 20
}
```

**使用场景:**

```
用户: 读取 README.md 文件
AI: 调用 read_file 工具
{
  "path": "README.md"
}
```

**注意事项:**

- 路径可以是相对路径或绝对路径
- 相对路径相对于当前工作目录
- 受沙箱保护,无法读取敏感路径

---

### 2.2 write_file - 写入文件

**描述:** 写入或覆盖文件内容

**风险等级:** 中 (RiskMedium)

**参数:**

| 参数名 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `path` | string | 是 | 文件路径 |
| `content` | string | 是 | 文件内容 |

**返回:**

- 成功: "文件写入成功"
- 失败: 错误信息

**示例:**

```json
{
  "path": "test.txt",
  "content": "Hello, World!"
}
```

**使用场景:**

```
用户: 创建一个新文件 hello.txt,内容是 "Hello, World!"
AI: 调用 write_file 工具
{
  "path": "hello.txt",
  "content": "Hello, World!"
}
```

**注意事项:**

- 如果文件已存在,会被覆盖
- 如果父目录不存在,会自动创建
- 写入前会自动保存文件快照,支持 `/undo` 撤销
- 受沙箱保护,无法写入敏感路径
- 需要用户确认 (除非配置了自动批准)

---

### 2.3 edit_file - 编辑文件

**描述:** 精确替换文件中的内容

**风险等级:** 中 (RiskMedium)

**参数:**

| 参数名 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `path` | string | 是 | 文件路径 |
| `old_string` | string | 是 | 要替换的旧内容 |
| `new_string` | string | 是 | 新内容 |

**返回:**

- 成功: "文件编辑成功"
- 失败: 错误信息

**示例:**

```json
{
  "path": "src/main.go",
  "old_string": "fmt.Println(\"Hello\")",
  "new_string": "fmt.Println(\"Hello, World!\")"
}
```

**使用场景:**

```
用户: 把 main.go 中的 "Hello" 改成 "Hello, World!"
AI: 先读取文件,然后调用 edit_file 工具
{
  "path": "src/main.go",
  "old_string": "fmt.Println(\"Hello\")",
  "new_string": "fmt.Println(\"Hello, World!\")"
}
```

**注意事项:**

- `old_string` 必须在文件中精确匹配
- 如果 `old_string` 出现多次,只替换第一次出现的位置
- 编辑前会自动保存文件快照,支持 `/undo` 撤销
- 受沙箱保护,无法编辑敏感路径
- 需要用户确认 (除非配置了自动批准)

---

### 2.4 list_dir - 列出目录

**描述:** 列出目录结构,支持递归

**风险等级:** 低 (RiskLow)

**参数:**

| 参数名 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `path` | string | 是 | 目录路径 |
| `recursive` | boolean | 否 | 是否递归 (默认: false) |
| `max_depth` | integer | 否 | 最大递归深度 (默认: 3) |

**返回:**

- 成功: 目录树结构 (字符串)
- 失败: 错误信息

**示例:**

```json
// 列出当前目录
{
  "path": "."
}

// 递归列出目录
{
  "path": "src",
  "recursive": true,
  "max_depth": 2
}
```

**使用场景:**

```
用户: 列出 src 目录下的所有文件
AI: 调用 list_dir 工具
{
  "path": "src",
  "recursive": true
}
```

**注意事项:**

- 自动忽略 `.git`, `node_modules`, `.aicoder` 等目录
- 尊重 `.gitignore` 规则
- 受沙箱保护,无法列出敏感路径

---

### 2.5 search_files - 搜索文件

**描述:** 在文件中搜索文本,支持正则表达式

**风险等级:** 低 (RiskLow)

**参数:**

| 参数名 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `path` | string | 是 | 搜索根目录 |
| `pattern` | string | 是 | 搜索模式 (正则表达式) |
| `case_insensitive` | boolean | 否 | 忽略大小写 (默认: false) |
| `include` | string | 否 | 文件 glob 过滤 (例如: "*.go") |

**返回:**

- 成功: 匹配结果列表 (文件名:行号:内容)
- 失败: 错误信息

**示例:**

```json
// 搜索所有 Go 文件中的 "func main"
{
  "path": ".",
  "pattern": "func main",
  "include": "*.go"
}

// 忽略大小写搜索
{
  "path": "src",
  "pattern": "error",
  "case_insensitive": true
}
```

**使用场景:**

```
用户: 在项目中搜索所有的 TODO 注释
AI: 调用 search_files 工具
{
  "path": ".",
  "pattern": "TODO:",
  "case_insensitive": true
}
```

**注意事项:**

- 支持 Go 正则表达式语法
- 自动忽略二进制文件
- 受沙箱保护,无法搜索敏感路径

---

### 2.6 delete_file - 删除文件

**描述:** 删除文件或目录

**风险等级:** 高 (RiskHigh)

**参数:**

| 参数名 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `path` | string | 是 | 文件或目录路径 |

**返回:**

- 成功: "文件删除成功"
- 失败: 错误信息

**示例:**

```json
{
  "path": "temp.txt"
}
```

**使用场景:**

```
用户: 删除 temp.txt 文件
AI: 调用 delete_file 工具
{
  "path": "temp.txt"
}
```

**注意事项:**

- 删除前会自动保存文件快照,支持 `/undo` 撤销
- 删除目录时会递归删除所有内容
- 受沙箱保护,无法删除敏感路径
- 需要用户确认 (高风险操作)

---

## 3. Shell 工具

### 3.1 run_command - 执行命令

**描述:** 在 Shell 中执行命令

**风险等级:** 中 (RiskMedium)

**参数:**

| 参数名 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `command` | string | 是 | 要执行的命令 |
| `timeout` | integer | 否 | 超时时间 (秒,默认: 60) |

**返回:**

- 成功: 命令输出 (stdout + stderr)
- 失败: 错误信息

**示例:**

```json
// 运行测试
{
  "command": "go test ./..."
}

// 自定义超时
{
  "command": "npm run build",
  "timeout": 300
}
```

**使用场景:**

```
用户: 运行项目的测试
AI: 调用 run_command 工具
{
  "command": "go test ./..."
}
```

**注意事项:**

- 命令在当前工作目录执行
- 默认超时 60 秒
- 禁止执行危险命令 (例如: `rm -rf /`, `mkfs`, `dd if=`)
- 需要用户确认 (除非命令在白名单中)

**禁止命令列表:**

- `rm -rf /`
- `mkfs`
- `dd if=`
- `:(){:|:&};:` (fork bomb)
- `chmod -R 777 /`

---

### 3.2 run_background - 后台运行

**描述:** 在后台启动长时进程

**风险等级:** 中 (RiskMedium)

**参数:**

| 参数名 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `command` | string | 是 | 要执行的命令 |

**返回:**

- 成功: "进程已在后台启动,PID: {pid}"
- 失败: 错误信息

**示例:**

```json
{
  "command": "npm run dev"
}
```

**使用场景:**

```
用户: 启动开发服务器
AI: 调用 run_background 工具
{
  "command": "npm run dev"
}
```

**注意事项:**

- 进程在后台运行,不会阻塞 Agent
- 会话结束时,后台进程会被自动终止
- 需要用户确认

---

## 4. 搜索工具

### 4.1 grep_search - 全局搜索

**描述:** 在项目中进行全局正则搜索

**风险等级:** 低 (RiskLow)

**参数:**

| 参数名 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `path` | string | 是 | 搜索根目录 |
| `pattern` | string | 是 | 搜索模式 (正则表达式) |
| `case_insensitive` | boolean | 否 | 忽略大小写 (默认: false) |
| `include` | string | 否 | 文件 glob 过滤 |
| `context_lines` | integer | 否 | 上下文行数 (默认: 0) |

**返回:**

- 成功: 匹配结果列表
- 失败: 错误信息

**示例:**

```json
// 基本搜索
{
  "path": ".",
  "pattern": "func.*Error"
}

// 带上下文行
{
  "path": "src",
  "pattern": "TODO",
  "context_lines": 2
}

// 过滤文件类型
{
  "path": ".",
  "pattern": "import.*React",
  "include": "*.{js,jsx,ts,tsx}"
}
```

**使用场景:**

```
用户: 找出所有返回 error 的函数
AI: 调用 grep_search 工具
{
  "path": ".",
  "pattern": "func.*error",
  "include": "*.go"
}
```

**注意事项:**

- 支持 Go 正则表达式语法
- 自动忽略 `.git`, `node_modules` 等目录
- 自动忽略二进制文件
- 受沙箱保护

---

### 4.2 ast_search - AST 语义搜索

**描述:** 使用 AST (抽象语法树) 进行语义搜索,可以精确查找函数、类型、接口、方法等代码元素

**风险等级:** 低 (RiskLow)

**参数:**

| 参数名 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `path` | string | 是 | 搜索根目录 |
| `query_type` | string | 是 | 查询类型: 'function', 'type', 'interface', 'method', 'struct', 'variable', 'import', 'all' |
| `name` | string | 否 | 名称模式 (支持通配符) |
| `language` | string | 否 | 语言: 'go' (v1.0 仅支持 Go) |
| `exported_only` | boolean | 否 | 仅搜索导出的符号 (Go only, 默认: false) |

**返回:**

- 成功: 匹配的代码元素列表 (文件名:行号:元素信息)
- 失败: 错误信息

**示例:**

```json
// 查找所有函数
{
  "path": ".",
  "query_type": "function"
}

// 查找导出的函数
{
  "path": ".",
  "query_type": "function",
  "exported_only": true
}

// 查找特定名称的方法
{
  "path": ".",
  "query_type": "method",
  "name": "Handle"
}

// 查找所有接口
{
  "path": ".",
  "query_type": "interface"
}

// 查找所有结构体
{
  "path": ".",
  "query_type": "struct"
}

// 查找导入
{
  "path": ".",
  "query_type": "import",
  "name": "github.com"
}

// 查找所有代码元素
{
  "path": ".",
  "query_type": "all"
}
```

**使用场景:**

```
用户: 找出所有实现了 Handler 接口的类型
AI: 先调用 ast_search 查找 Handler 接口
{
  "path": ".",
  "query_type": "interface",
  "name": "Handler"
}
然后查找实现该接口的类型
```

**注意事项:**

- v1.0 仅支持 Go 语言
- 自动跳过 `.git`, `node_modules`, `vendor` 等目录
- 最多返回 100 个结果
- 名称匹配支持简单的通配符 (包含匹配)
- 受沙箱保护

**查询类型说明:**

| 查询类型 | 说明 | 示例 |
|---------|------|------|
| `function` | 查找函数 (不包括方法) | `func main()`, `func NewServer()` |
| `method` | 查找方法 (有接收者的函数) | `func (s *Server) Start()` |
| `type` | 查找所有类型定义 | `type User struct{}`, `type Handler interface{}` |
| `interface` | 仅查找接口类型 | `type Handler interface{}` |
| `struct` | 仅查找结构体类型 | `type User struct{}` |
| `variable` | 查找变量声明 | `var config Config` |
| `import` | 查找导入语句 | `import "fmt"` |
| `all` | 查找所有元素 | 所有上述类型 |

---

### 4.3 web_search - 联网搜索

**描述:** 使用搜索 API 进行联网搜索,返回搜索结果

**风险等级:** 低 (RiskLow)

**参数:**

| 参数名 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `query` | string | 是 | 搜索查询 |
| `num_results` | integer | 否 | 返回结果数量 (默认: 5, 最大: 10) |
| `language` | string | 否 | 语言代码 (例如: 'en', 'zh-CN', 默认: 'en') |

**返回:**

- 成功: 搜索结果列表 (标题、URL、摘要)
- 失败: 错误信息

**示例:**

```json
// 基本搜索
{
  "query": "golang tutorial"
}

// 自定义结果数量
{
  "query": "react hooks",
  "num_results": 10
}

// 中文搜索
{
  "query": "Go 语言教程",
  "language": "zh-CN"
}
```

**使用场景:**

```
用户: 搜索最新的 Go 1.22 新特性
AI: 调用 web_search 工具
{
  "query": "Go 1.22 new features",
  "num_results": 5
}
```

**配置要求:**

需要设置以下环境变量:

```bash
export SEARCH_API_KEY="your-api-key"
export SEARCH_ENGINE_ID="your-engine-id"
```

**获取 API 凭证:**

1. 获取 API Key: https://developers.google.com/custom-search/v1/overview
2. 创建搜索引擎: https://programmablesearchengine.google.com/
3. 设置环境变量

**注意事项:**

- 需要配置 Google Custom Search API
- 默认超时 30 秒
- 最多返回 10 个结果
- 结果包含标题、URL 和摘要
- 如果未配置 API 凭证,会返回配置说明

---

## 5. 风险等级说明

### 5.1 风险等级定义

| 等级 | 说明 | 示例操作 | 默认行为 |
|------|------|---------|---------|
| **低 (RiskLow)** | 只读操作,不会修改系统 | 读文件、列目录、搜索 | 可配置自动批准 |
| **中 (RiskMedium)** | 写操作,可能修改文件 | 写文件、编辑文件、运行命令 | 需要用户确认 |
| **高 (RiskHigh)** | 危险操作,可能造成数据丢失 | 删除文件、系统级命令 | 需要用户确认 + 警告 |

### 5.2 权限确认流程

```
1. 工具调用请求
   ↓
2. 检查禁止命令 → 硬拒绝
   ↓
3. 检查全局自动批准 → 允许
   ↓
4. 检查会话级允许 → 允许
   ↓
5. 检查读操作自动批准 → 允许
   ↓
6. 检查命令白名单 → 允许
   ↓
7. 显示确认对话框
   ├─ Y: 允许一次
   ├─ N: 拒绝
   └─ A: 始终允许 (本次会话)
```

### 5.3 配置自动批准

在 `~/.aicoder/config.json` 中配置:

```json
{
  "autoApprove": false,
  "autoApproveReads": true,
  "autoApproveCommands": [
    "go test",
    "npm test",
    "git status",
    "git diff"
  ]
}
```

---

## 6. 沙箱保护

### 6.1 禁止访问的路径

以下路径前缀被硬编码禁止访问:

```
/etc/shadow
/etc/passwd
/etc/sudoers
~/.ssh/
~/.gnupg/
```

### 6.2 禁止执行的命令

以下命令模式被硬编码禁止执行:

```
rm -rf /
mkfs
dd if=
:(){:|:&};:           # fork bomb
chmod -R 777 /
```

### 6.3 沙箱检查

所有文件操作和命令执行都会经过沙箱检查:

```go
func checkSandbox(path string) error {
    // 检查路径是否在禁止列表中
    for _, denied := range sandboxDenied {
        if strings.Contains(path, denied) {
            return fmt.Errorf("访问被拒绝: %s 是敏感路径", path)
        }
    }
    return nil
}
```

---

## 附录

### A. 工具列表总览

| 工具名 | 类别 | 风险等级 | 说明 |
|--------|------|---------|------|
| `read_file` | 文件系统 | 低 | 读取文件 |
| `write_file` | 文件系统 | 中 | 写入文件 |
| `edit_file` | 文件系统 | 中 | 编辑文件 |
| `list_dir` | 文件系统 | 低 | 列出目录 |
| `search_files` | 文件系统 | 低 | 搜索文件 |
| `delete_file` | 文件系统 | 高 | 删除文件 |
| `run_command` | Shell | 中 | 执行命令 |
| `run_background` | Shell | 中 | 后台运行 |
| `grep_search` | 搜索 | 低 | 全局正则搜索 |
| `ast_search` | 搜索 | 低 | AST 语义搜索 |
| `web_search` | 搜索 | 低 | 联网搜索 |

### B. JSON Schema 示例

工具的 JSON Schema 定义示例:

```json
{
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "description": "文件路径"
    },
    "content": {
      "type": "string",
      "description": "文件内容"
    }
  },
  "required": ["path", "content"]
}
```

### C. 错误处理

工具执行失败时,返回的 `Result` 结构:

```go
&tools.Result{
    Content: "错误信息: 文件不存在",
    IsError: true,
}
```

AI 会收到这个错误信息,并可以:
1. 向用户报告错误
2. 尝试其他方法
3. 请求用户提供更多信息

---

*本文档持续更新中,如有问题请提交 Issue 或 PR。*
