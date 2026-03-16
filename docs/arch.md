# 系统架构文档（Architecture Design）

**产品名称：** aicoder  
**版本：** v1.0  
**文档状态：** 草稿  
**创建日期：** 2026-03-13  
**关联文档：** [prd.md](./prd.md)

---

## 1. 架构总览

### 1.1 设计原则

- **单一职责**：每个模块只做一件事，边界清晰
- **依赖倒置**：上层模块依赖抽象接口，而非具体实现（尤其是 LLM 提供商、工具注册）
- **可替换性**：LLM 提供商、权限策略、工具集均可通过配置/插件替换
- **最小信任**：工具执行前默认需要用户确认，权限分级管理
- **零外部依赖启动**：主程序为单一静态二进制，无需运行时环境

### 1.2 整体分层

```
┌──────────────────────────────────────────────────────┐
│                    用户交互层                          │
│         CLI (cobra)  +  TUI (bubbletea)              │
├──────────────────────────────────────────────────────┤
│                    应用编排层                          │
│     Session 管理  │  Agent Loop  │  斜杠命令路由       │
├──────────────────────────────────────────────────────┤
│                    核心能力层                          │
│  LLM 适配层  │  工具执行引擎  │  MCP 客户端  │ 上下文  │
├──────────────────────────────────────────────────────┤
│                    基础设施层                          │
│      配置管理  │  日志  │  文件备份  │  安全沙箱        │
└──────────────────────────────────────────────────────┘
```

---

## 2. 目录结构

```
aicoder/
├── main.go                        # 程序入口
├── go.mod
├── go.sum
│
├── cmd/                           # CLI 命令层（cobra）
│   ├── root.go                    # 根命令，全局 flags
│   ├── interactive.go             # 交互式模式入口
│   ├── oneshot.go                 # 单次执行模式入口
│   └── version.go                 # version 子命令
│
├── internal/
│   │
│   ├── agent/                     # Agent 编排核心
│   │   ├── agent.go               # Agent 主循环
│   │   ├── loop.go                # 工具调用编排逻辑
│   │   ├── message.go             # Message 构建与管理
│   │   └── interrupt.go           # Ctrl+C 中断处理
│   │
│   ├── llm/                       # LLM 提供商适配层
│   │   ├── interface.go           # Provider 接口定义
│   │   ├── stream.go              # 流式响应处理
│   │   ├── anthropic/
│   │   │   └── client.go          # Anthropic SDK 封装
│   │   ├── openai/
│   │   │   └── client.go          # OpenAI SDK 封装
│   │   ├── google/
│   │   │   └── client.go          # Google Gemini 封装
│   │   ├── ollama/
│   │   │   └── client.go          # Ollama 本地模型封装（v1.1）
│   │   └── factory.go             # Provider 工厂，按配置实例化
│   │
│   ├── tools/                     # 内置工具实现
│   │   ├── registry.go            # 工具注册表
│   │   ├── interface.go           # Tool 接口定义
│   │   ├── risk.go                # 风险评级逻辑
│   │   ├── fs/
│   │   │   ├── read_file.go
│   │   │   ├── write_file.go
│   │   │   ├── edit_file.go       # diff/patch 模式编辑
│   │   │   ├── list_dir.go
│   │   │   ├── search_files.go    # 正则搜索
│   │   │   └── delete_file.go
│   │   ├── exec/
│   │   │   ├── run_command.go     # 同步命令执行
│   │   │   └── run_background.go  # 后台进程管理
│   │   └── search/
│   │       ├── grep_search.go
│   │       ├── ast_search.go      # AST 语义搜索
│   │       └── web_search.go      # 联网搜索（可选）
│   │
│   ├── mcp/                       # MCP 客户端（v1.1）
│   │   ├── client.go              # MCP 协议实现
│   │   ├── manager.go             # 多 Server 连接管理
│   │   ├── transport/
│   │   │   ├── stdio.go           # stdio transport
│   │   │   └── sse.go             # SSE transport
│   │   └── tool_bridge.go         # 将 MCP 工具适配为内置工具接口
│   │
│   ├── context/                   # 项目上下文收集
│   │   ├── collector.go           # 上下文收集主入口
│   │   ├── git.go                 # Git 信息（status/diff/log）
│   │   ├── project.go             # 项目语言/依赖检测
│   │   ├── aicoder_md.go          # AICODER.md 加载与解析
│   │   └── summarizer.go          # 目录结构摘要生成
│   │
│   ├── session/                   # 会话管理
│   │   ├── session.go             # Session 结构与生命周期
│   │   ├── history.go             # 对话历史存储（内存 + 磁盘持久化）
│   │   ├── cost.go                # Token 用量统计
│   │   └── snapshot.go            # 文件变更快照（用于 /undo）
│   │
│   ├── permission/                # 权限管理
│   │   ├── policy.go              # 权限策略定义与评估
│   │   ├── allowlist.go           # 会话级白名单
│   │   ├── sandbox.go             # 路径沙箱（禁止访问敏感目录）
│   │   └── risk_classifier.go     # 命令风险自动分级
│   │
│   ├── ui/                        # 终端 UI
│   │   ├── app.go                 # bubbletea App 主模型
│   │   ├── input.go               # 输入框组件（历史导航）
│   │   ├── output.go              # 流式输出渲染
│   │   ├── confirm.go             # 权限确认弹窗组件
│   │   ├── diff_view.go           # 彩色 diff 展示
│   │   ├── spinner.go             # 加载动画
│   │   ├── theme.go               # 颜色主题（dark/light）
│   │   └── markdown.go            # Markdown + 代码高亮渲染
│   │
│   ├── slash/                     # 斜杠命令路由
│   │   ├── router.go              # 命令注册与分发
│   │   └── commands/
│   │       ├── help.go
│   │       ├── clear.go
│   │       ├── history.go
│   │       ├── undo.go
│   │       ├── diff.go
│   │       ├── commit.go
│   │       ├── cost.go
│   │       ├── model.go
│   │       ├── config.go
│   │       └── init.go
│   │
│   └── config/                    # 配置管理
│       ├── config.go              # 配置结构体定义
│       ├── loader.go              # 多层配置加载（viper）
│       ├── keychain.go            # API Key 安全存储
│       └── validator.go           # 配置校验
│
├── pkg/                           # 可外部复用的公共库
│   ├── diff/                      # diff 算法封装
│   ├── backup/                    # 文件备份工具
│   └── logger/                    # zap 日志封装
│
├── testdata/                      # 测试数据
└── docs/                          # 文档目录
    ├── prd.md
    ├── arch.md
    └── todo.md
```

---

## 3. 核心模块详细设计

### 3.1 LLM 适配层

所有 LLM 提供商实现统一的 `Provider` 接口，上层 Agent 无需感知具体提供商。

```go
// internal/llm/interface.go

type Message struct {
    Role    string      // "user" | "assistant" | "tool"
    Content interface{} // string | []ContentBlock
}

type StreamChunk struct {
    Type     string // "text" | "tool_use" | "tool_result" | "done"
    Text     string
    ToolCall *ToolCall
    Usage    *Usage
}

type Provider interface {
    // Chat 发起流式对话，通过 channel 返回数据块
    Chat(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error)
    // Models 列出该提供商可用的模型
    Models(ctx context.Context) ([]ModelInfo, error)
    // Name 提供商标识符
    Name() string
}

type ChatRequest struct {
    Model        string
    Messages     []Message
    Tools        []ToolDefinition // 注入的工具描述
    MaxTokens    int
    SystemPrompt string
}
```

**提供商实现矩阵：**

| 提供商 | 协议 | 流式 | 工具调用 | 版本 |
|---|---|---|---|---|
| Anthropic | 原生 SDK | ✅ SSE | ✅ | v1.0 |
| OpenAI | REST | ✅ SSE | ✅ | v1.0 |
| Google | REST | ✅ | ✅ | v1.0 |
| Ollama | REST | ✅ | ✅（部分模型）| v1.1 |

---

### 3.2 工具注册与执行引擎

工具采用**注册表模式**，所有工具（含 MCP 桥接工具）在启动时统一注册。

```go
// internal/tools/interface.go

type Tool interface {
    Name() string
    Description() string
    Schema() json.RawMessage  // JSON Schema，用于传给 LLM
    Execute(ctx context.Context, params map[string]any) (ToolResult, error)
    RiskLevel() RiskLevel     // Low | Medium | High | Critical
}

type RiskLevel int
const (
    RiskLow      RiskLevel = iota // 只读操作
    RiskMedium                    // 写文件、低危命令
    RiskHigh                      // 删除文件、执行脚本
    RiskCritical                  // sudo、网络、rm -rf 等
)

type ToolResult struct {
    Content  string
    IsError  bool
    Metadata map[string]any
}
```

**工具执行决策流程：**

```
Agent 收到 LLM 工具调用请求
        │
        ▼
Registry.Get(toolName) → 查找工具实例
        │
        ▼
permission.Policy.Evaluate(tool, params)
        │
        ├── ALLOW（白名单 / autoApprove）─→ 直接执行
        ├── DENY（黑名单 / 沙箱）─────────→ 返回拒绝结果给 LLM
        └── ASK（需用户确认）
               │
               ▼
           UI.ShowConfirmDialog() [Y / N / A]
               │
               ├── Y 允许一次 ──────────→ 执行
               ├── A 始终允许 ──────────→ 加入白名单，执行
               └── N 拒绝 ──────────────→ 返回拒绝结果给 LLM
```

---

### 3.3 Agent 主循环

```go
// internal/agent/loop.go（伪代码）

func (a *Agent) Run(ctx context.Context, userInput string) error {
    a.session.Append(UserMessage(userInput))

    for {
        // 1. 构建完整请求（系统提示 + 项目上下文 + 历史）
        req := a.buildRequest()

        // 2. 调用 LLM，获取流式响应 channel
        stream, err := a.provider.Chat(ctx, req)
        if err != nil { return err }

        // 3. 消费流：渲染文本 / 收集工具调用
        response, toolCalls := a.consumeStream(ctx, stream)

        // 4. 追加 assistant 响应
        a.session.Append(AssistantMessage(response, toolCalls))

        // 5. 若无工具调用，本轮结束
        if len(toolCalls) == 0 {
            return nil
        }

        // 6. 依次执行工具（含权限确认）
        toolResults := a.executeTools(ctx, toolCalls)

        // 7. 追加工具结果，继续循环
        a.session.Append(ToolResultMessages(toolResults))
    }
}
```

> **Ctrl+C 中断：** 取消当前 ctx，Agent Loop 在流消费处退出，Messages 保留已完成部分，下次输入继续对话。

---

### 3.4 会话与快照管理

```
Session
  ├── ID：UUID，每次启动生成
  ├── Messages：[]Message（内存 + 持久化到 ~/.aicoder/sessions/）
  ├── FileSnapshots：map[filepath][]byte（写文件前保存原始内容）
  ├── CostTracker：InputTokens / OutputTokens / EstimatedCost
  └── AllowList：本次会话已授权的工具+命令集合
```

**`/undo` 实现机制：**
1. `write_file` / `edit_file` 执行前，将原始内容存入 `FileSnapshots` 栈
2. `/undo` 弹出栈顶快照，恢复文件到磁盘，并截断 `Messages` 中对应的工具记录

---

### 3.5 MCP 客户端架构（v1.1）

```
启动阶段
  │
  ▼
读取 config.mcpServers
  │
  ├── stdio transport → fork 子进程，stdin/stdout 双向通信
  └── SSE transport   → HTTP 长连接
  │
  ▼
MCP initialize + tools/list 握手
  │
  ▼
tool_bridge.go：适配为 Tool 接口
  │
  ▼
注册到全局 Registry（与内置工具统一调度）
```

---

### 3.6 配置加载优先级

```
CLI Flags（最高优先级）
    ↓
环境变量（AICODER_* 前缀）
    ↓
项目级配置：./.aicoder/config.json
    ↓
用户级配置：~/.aicoder/config.json（最低优先级）
```

**API Key 安全存储策略：**

| 平台 | 存储方案 |
|---|---|
| macOS | 系统 Keychain（`go-keyring`） |
| Linux | `libsecret` / 加密文件（AES-256-GCM） |
| CI 环境 | 环境变量（优先级最高，覆盖 Keychain） |

---

### 3.7 TUI 架构（bubbletea Elm 模式）

```
Model（全局状态）
  ├── inputBox     → 用户输入区（含历史导航）
  ├── outputView   → 流式输出区（支持滚动）
  ├── confirmModal → 权限确认弹窗（modal layer）
  ├── statusBar    → 底部状态栏（模型 / Token / 目录）
  └── spinner      → LLM 等待动画

Update（事件处理）
  ├── KeyMsg        → 键盘输入、方向键、快捷键
  ├── StreamChunkMsg → LLM 流式数据（通过 tea.Cmd 注入）
  ├── ToolResultMsg  → 工具执行完成事件
  └── ErrorMsg       → 错误展示

View（渲染）
  └── lipgloss 彩色布局 + glamour Markdown 渲染
```

> **非阻塞流式渲染：** LLM 响应通过 `tea.Cmd`（goroutine + channel）异步注入 Model，每个 chunk 触发一次 Update → View，实现逐字显示，不阻塞键盘事件循环。

---

## 4. 数据流图

### 4.1 交互式模式完整数据流

```
用户键盘输入
      │
      ▼
  TUI App (bubbletea) ── inputBox ── Enter
      │
      ▼
  slash.Router.Match(input)
      │
      ├── 斜杠命令 ─→ slash.Command.Execute() ─→ 更新 TUI
      │
      └── 普通对话
            │
            ▼
        agent.Run(ctx, input)
            │
            ├─[构建 Request]── context.Collector + session.Messages
            │
            ├─[LLM 流式响应]── llm.Provider.Chat()
            │       └── StreamChunkMsg ─→ outputView 逐字渲染
            │
            ├─[工具调用]── tools.Registry + permission.Policy
            │       └── confirmModal ←─ 等待用户 Y/N/A
            │
            └─[工具结果]── 追加 Messages，继续 Agent 循环
```

### 4.2 管道 / 单次执行模式数据流

```
stdin / CLI 参数
    │
    ▼
cmd.oneshot（检测 !isTerminal(os.Stdin)）
    │
    ▼
合并 stdin + 参数为 prompt
    │
    ▼
agent.Run()（无 TUI，直接写 stdout）
    │
    ├── 工具执行（autoApprove=true 或 --dangerously-skip-permissions）
    │
    └── stdout 输出，exit 0/1
```

---

## 5. 安全架构

### 5.1 路径沙箱（不可覆盖）

以下路径前缀硬编码禁止访问：

```
~/.ssh/
~/.gnupg/
/etc/shadow
/etc/passwd
/private/etc/  （macOS）
```

### 5.2 命令风险分级

| 级别 | 匹配示例 | 默认行为 |
|---|---|---|
| Low | `cat`, `ls`, `git status`, `go test` | 可配置 autoApprove |
| Medium | `npm install`, `go build`, 写文件 | 需确认 |
| High | `rm`, `mv`, 修改配置文件 | 需确认 + 风险提示 |
| Critical | `sudo`, `rm -rf /`, `curl \| sh` | 需确认 + 红色警告 |

### 5.3 审计日志格式

```jsonc
// ~/.aicoder/logs/2026-03-13.jsonl
{
  "ts": "2026-03-13T10:23:45Z",
  "session_id": "sess_abc123",
  "tool": "run_command",
  "input": { "command": "npm run test" },
  "risk": "Low",
  "approved_by": "user",  // "user" | "auto" | "denied"
  "duration_ms": 1234,
  "exit_code": 0
}
```

---

## 6. 关键依赖

| 依赖包 | 版本 | 用途 |
|---|---|---|
| `github.com/spf13/cobra` | v1.8+ | CLI 命令框架 |
| `github.com/charmbracelet/bubbletea` | v0.27+ | TUI 框架 |
| `github.com/charmbracelet/lipgloss` | v0.12+ | 终端样式渲染 |
| `github.com/charmbracelet/glamour` | v0.8+ | Markdown 渲染 |
| `github.com/spf13/viper` | v1.18+ | 多层配置管理 |
| `go.uber.org/zap` | v1.27+ | 结构化日志 |
| `github.com/anthropics/anthropic-sdk-go` | latest | Anthropic API |
| `github.com/openai/openai-go` | latest | OpenAI API |
| `github.com/stretchr/testify` | v1.9+ | 测试断言 |
| `github.com/zalando/go-keyring` | v0.2+ | 系统 Keychain |
| `github.com/sergi/go-diff` | v1.3+ | diff/patch 算法 |

---

## 7. 非功能性架构决策

### 7.1 性能保障

- 启动时仅初始化必要模块，LLM client 懒加载
- 流式 chunk 通过 buffered channel 传递，渲染延迟 < 16ms（60fps）
- 工具执行绑定 ctx，支持超时与取消（`exec.CommandContext`）

### 7.2 错误处理策略

- LLM API 错误：指数退避自动重试（最多 3 次），超出后展示友好提示
- 工具执行错误：捕获 stderr，格式化为 `ToolResult{IsError: true}`，返回给 LLM 自主决策
- 文件写入错误：从 FileSnapshot 自动回滚，提示用户

### 7.3 测试策略

| 层级 | 工具 | 覆盖率目标 |
|---|---|---|
| 单元测试 | testify | > 80% |
| 集成测试 | testify + mock LLM | Agent Loop 全路径覆盖 |
| E2E 测试 | 真实 API（CI 环境） | 主要用户路径 |
| 性能测试 | `go test -bench` | 启动时间 < 500ms |

---

*架构文档随代码演进持续更新，重大变更需同步修改本文档并通过 PR Review。*
