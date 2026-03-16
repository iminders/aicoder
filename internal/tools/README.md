# 工具执行引擎 (Phase 3)

## 📋 概述

本目录包含 aicoder 的工具执行引擎实现,提供了完整的工具系统基础设施、文件系统操作、命令执行和代码搜索功能。

## 🏗️ 架构

### 核心组件

```
internal/tools/
├── interface.go           # 工具接口定义、注册表、风险等级
├── risk.go                # 风险评估和分类系统
├── filesystem/            # 文件系统工具
│   ├── read_file.go      # 读取文件
│   └── tools.go          # write/edit/list/search/delete
├── shell/                 # Shell 命令工具
│   └── tools.go          # run_command/run_background
└── search/                # 搜索工具
    ├── grep.go           # 正则搜索
    ├── ast_search.go     # AST 语义搜索
    └── web_search.go     # 联网搜索
```

## ✅ 已实现功能 (Phase 3 完成度: 95%)

### 1. 工具基础设施

#### 工具接口 (`interface.go`)
- ✅ `Tool` 接口定义
  - `Name()` - 工具名称
  - `Description()` - 工具描述
  - `Schema()` - JSON Schema 参数定义
  - `Execute()` - 执行工具
  - `Risk()` - 风险等级

- ✅ `Registry` 工具注册表
  - `Register()` - 注册工具
  - `Get()` - 按名称查找
  - `All()` - 列出所有工具
  - 线程安全 (sync.RWMutex)

- ✅ `RiskLevel` 风险等级
  - `RiskLow` - 只读操作
  - `RiskMedium` - 写操作
  - `RiskHigh` - 危险操作

#### 风险评估系统 (`risk.go`)
- ✅ `ClassifyCommandRisk()` - 命令风险分类
  - 识别危险命令 (rm -rf, mkfs, sudo 等)
  - 三级风险分类 (Low/Medium/High)
  - 支持 50+ 常用命令模式

- ✅ `ClassifyToolRisk()` - 工具风险分类
  - 基于工具类型和参数评估
  - 智能路径危险检测
  - 动态风险调整

- ✅ `IsPathDangerous()` - 路径危险检测
  - 系统关键路径保护
  - 路径遍历攻击检测
  - 通配符检测

- ✅ `EvaluateOperationRisk()` - 综合风险评估
  - 批量操作风险提升
  - 递归操作风险提升
  - 管道和重定向检测

- ✅ `GetRiskDescription()` - 风险描述
- ✅ `ShouldAutoApprove()` - 自动批准决策

### 2. 文件系统工具 (6 个)

| 工具 | 状态 | 功能 | 风险等级 |
|------|------|------|---------|
| `read_file` | ✅ | 读取文件,支持行范围 | 低 |
| `write_file` | ✅ | 写入文件,自动创建目录 | 中 |
| `edit_file` | ✅ | 精确修改,diff/patch 模式 | 中 |
| `list_dir` | ✅ | 递归目录树,支持深度限制 | 低 |
| `search_files` | ✅ | 正则搜索,返回匹配行 | 低 |
| `delete_file` | ✅ | 删除文件,强制快照 | 高 |

**特性:**
- ✅ 沙箱路径保护 (`checkSandbox()`)
- ✅ 快照回调集成 (`SnapshotFunc`)
- ✅ 自动忽略列表 (.git, node_modules, vendor)
- ✅ .gitignore 规则支持
- ✅ 错误处理和回滚

### 3. Shell 命令工具 (2 个)

| 工具 | 状态 | 功能 | 风险等级 |
|------|------|------|---------|
| `run_command` | ✅ | 执行命令,60秒超时 | 中 |
| `run_background` | ✅ | 后台进程,返回 PID | 中 |

**特性:**
- ✅ 禁止命令黑名单 (`ForbiddenPatterns`)
- ✅ stdout/stderr 捕获
- ✅ 超时控制 (默认 60 秒)
- ✅ 工作目录支持
- ✅ 上下文取消支持

### 4. 搜索工具 (3 个)

| 工具 | 状态 | 功能 | 风险等级 |
|------|------|------|---------|
| `grep_search` | ✅ | 正则搜索,支持上下文行 | 低 |
| `ast_search` | ✅ | AST 语义搜索 (Go) | 低 |
| `web_search` | ✅ | Google Custom Search | 低 |

**特性:**
- ✅ 纯 Go 正则引擎
- ✅ 大小写不敏感
- ✅ Glob 文件过滤
- ✅ 上下文行显示
- ✅ AST 查询类型: function, method, type, interface, struct, variable, import, all
- ✅ 导出符号过滤
- ✅ Google Custom Search API 集成

### 5. 工具集成

- ✅ 自动注册 (5 个 `init()` 函数)
- ✅ 线程安全访问
- ✅ JSON Schema 导出
- ✅ 单元测试覆盖

## 📊 测试覆盖

### 测试统计

| 模块 | 测试文件 | 测试用例 | 覆盖场景 |
|------|---------|---------|---------|
| interface | - | - | (接口定义,无测试) |
| risk | risk_test.go | 6 个测试组 | 命令风险、工具风险、路径检测、综合评估 |
| filesystem | tools_test.go | 8 个 | 读写编辑删除、沙箱、行范围 |
| shell | tools_test.go | 5 个 | 命令执行、超时、禁止命令 |
| search/grep | grep_test.go | 4 个 | 正则搜索、大小写、Glob |
| search/ast | ast_search_test.go | 11 个 | 函数、方法、类型、接口、结构体、变量、导入 |
| search/web | web_search_test.go | 5 个 | 配置检查、参数验证、Schema |

**总计:** 39 个测试用例,全部通过 ✅

### 测试命令

```bash
# 运行所有工具测试
go test ./internal/tools/... -v

# 运行特定模块测试
go test ./internal/tools/ -v          # interface + risk
go test ./internal/tools/filesystem/ -v
go test ./internal/tools/shell/ -v
go test ./internal/tools/search/ -v

# 查看覆盖率
go test ./internal/tools/... -cover
```

## 🔒 安全机制

### 1. 沙箱保护

**禁止路径列表:**
```
/etc/shadow
/etc/passwd
/etc/sudoers
/.ssh
/.gnupg
```

**检查函数:** `filesystem/checkSandbox()`

### 2. 命令黑名单

**禁止命令模式:**
```
rm -rf /
mkfs
dd if=
:(){:|:&};:           # fork bomb
chmod -R 777 /
```

**检查函数:** `shell/isForbidden()`

### 3. 风险分级

**三级风险体系:**

| 等级 | 描述 | 示例 |
|------|------|------|
| **Low** | 只读操作,不修改系统 | read_file, ls, git status |
| **Medium** | 写操作,需用户确认 | write_file, npm install |
| **High** | 危险操作,强烈建议确认 | delete_file, rm -rf, sudo |

**评估函数:** `risk.go` 中的 5 个函数

### 4. 快照机制

**回调接口:**
```go
var SnapshotFunc func(toolName, callID, filePath string, before, after []byte)
```

**触发时机:**
- write_file: 写入前
- edit_file: 编辑前
- delete_file: 删除前

**用途:** 支持 `/undo` 命令回滚

## 📝 使用示例

### 注册自定义工具

```go
type MyTool struct{}

func (t *MyTool) Name() string          { return "my_tool" }
func (t *MyTool) Description() string   { return "My custom tool" }
func (t *MyTool) Risk() tools.RiskLevel { return tools.RiskLow }
func (t *MyTool) Schema() json.RawMessage {
    return json.RawMessage(`{"type": "object", "properties": {...}}`)
}
func (t *MyTool) Execute(ctx context.Context, input json.RawMessage) (*tools.Result, error) {
    // 实现工具逻辑
    return &tools.Result{Content: "success"}, nil
}

// 注册工具
func init() {
    tools.Global.Register(&MyTool{})
}
```

### 使用风险评估

```go
// 评估命令风险
risk := tools.ClassifyCommandRisk("rm -rf /tmp/test")
fmt.Println(risk.String()) // "高"

// 评估工具风险
risk = tools.ClassifyToolRisk("write_file", map[string]interface{}{
    "path": "/etc/hosts",
})
fmt.Println(risk) // RiskHigh

// 判断是否需要确认
needConfirm := !tools.ShouldAutoApprove(risk, true, false)
if needConfirm {
    // 显示确认对话框
}
```

### 访问工具注册表

```go
// 获取工具
tool, err := tools.Global.Get("read_file")
if err != nil {
    log.Fatal(err)
}

// 执行工具
result, err := tool.Execute(context.Background(), []byte(`{"path": "test.txt"}`))

// 列出所有工具
allTools := tools.Global.All()
for _, t := range allTools {
    fmt.Printf("%s: %s (Risk: %s)\n", t.Name(), t.Description(), t.Risk())
}
```

## 🎯 Phase 3 完成情况

### ✅ 已完成 (15/16 项)

- [x] Tool 接口定义
- [x] Registry 实现
- [x] RiskLevel 定义
- [x] risk.go 风险评估系统 ⭐ **新增**
- [x] read_file (行范围读取)
- [x] write_file (快照 + 自动创建目录)
- [x] edit_file (diff/patch + 回滚)
- [x] list_dir (递归 + .gitignore)
- [x] search_files (正则 + 上下文)
- [x] delete_file (快照 + 高风险)
- [x] run_command (超时 + 黑名单)
- [x] run_background (PID + 后台)
- [x] grep_search (纯 Go + 正则)
- [x] ast_search (Go AST) ⭐ **新增**
- [x] web_search (Google API) ⭐ **新增**

### ⚠️ 部分完成 (1/16 项)

- [~] 后台进程清理机制
  - ✅ 启动后台进程
  - ✅ 返回 PID 标签
  - ⏳ 会话结束时自动清理 (需要在 internal/session/ 中实现)

### 📌 验收标准

- [x] 所有工具可被 Agent 正确调用
- [x] 沙箱拦截测试全部通过
- [x] 快照回滚流程测试通过
- [x] 风险分级验证测试通过

**Phase 3 完成度: 95%** ✅

## 🔜 后续工作 (Phase 4 依赖)

### 必需 (与 Phase 4 协调)

1. **快照栈管理** (`internal/session/snapshot.go`)
   - 快照 push/pop
   - 快照持久化
   - `/undo` 命令支持

2. **后台进程清理** (`internal/session/session.go`)
   - 注册后台进程
   - 会话结束时清理
   - 进程状态跟踪

### 可选 (未来版本)

3. **AST 多语言支持** (v1.1)
   - Python: tree-sitter-python
   - JavaScript: tree-sitter-javascript

4. **性能优化**
   - 并发搜索
   - 结果分页
   - 大文件分块处理

5. **增强功能**
   - 符号链接攻击防护
   - 动态风险评分
   - 错误代码分类

## 📚 相关文档

- [工具参考文档](../../../docs/tools-reference.md) - 所有工具的详细说明
- [搜索工具说明](./search/README.md) - 搜索工具的使用指南
- [开发者指南](../../../docs/development.md) - 如何添加新工具
- [架构设计](../../../docs/arch.md) - 系统架构说明

## 📊 代码统计

- **实现代码:** ~1,800 行
- **测试代码:** ~1,200 行
- **文档:** ~500 行
- **工具数量:** 11 个
- **测试用例:** 39 个
- **测试通过率:** 100%

---

**最后更新:** 2026-03-16
**状态:** Phase 3 基本完成,可以进入 Phase 4
