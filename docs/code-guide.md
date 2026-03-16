# 代码说明文档

**产品名称:** aicoder
**版本:** v1.0
**文档状态:** 正式版
**创建日期:** 2026-03-16
**关联文档:** [arch.md](./arch.md) · [development.md](./development.md) · [tools-reference.md](./tools-reference.md)

---

## 目录

- [1. 代码架构概览](#1-代码架构概览)
- [2. 核心模块详解](#2-核心模块详解)
- [3. 数据流分析](#3-数据流分析)
- [4. 关键算法](#4-关键算法)

---

## 1. 代码架构概览

### 1.1 分层架构

```
┌─────────────────────────────────────────┐
│         CLI 层 (cmd/)                    │
│  - 命令行参数解析                         │
│  - 全局 flags 定义                       │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│      应用层 (internal/agent/)            │
│  - Agent 主循环                          │
│  - 权限管理                              │
│  - 工具调用编排                          │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│      服务层 (internal/)                  │
│  - LLM 提供商 (llm/)                    │
│  - 工具系统 (tools/)                    │
│  - 会话管理 (session/)                  │
│  - 配置管理 (config/)                   │
│  - 上下文收集 (context/)                │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│      基础设施层 (internal/)              │
│  - UI 渲染 (ui/)                        │
│  - 日志系统 (logger/)                   │
│  - 斜杠命令 (slash/)                    │
└─────────────────────────────────────────┘
```

### 1.2 模块依赖关系

```
cmd/root.go
    ├─→ internal/config
    ├─→ internal/agent
    │       ├─→ internal/llm
    │       ├─→ internal/tools
    │       ├─→ internal/session
    │       ├─→ internal/context
    │       └─→ internal/ui
    ├─→ internal/slash
    └─→ internal/logger
```

---

## 2. 核心模块详解

### 2.1 Agent 模块 (internal/agent/)

#### agent.go - Agent 主循环

**核心结构:**

```go
type Agent struct {
    cfg      *config.Config      // 配置
    provider llm.Provider         // LLM 提供商
    sess     *session.Session     // 会话
    guard    *PermissionGuard     // 权限守卫
    renderer *ui.Renderer         // UI 渲染器
}
```

**关键方法:**

1. **New() - 创建 Agent**

```go
func New(cfg *config.Config) (*Agent, error)
```

流程:
- 创建会话
- 初始化权限守卫
- 收集项目上下文
- 构建系统提示词
- 创建 LLM Provider
- 初始化 UI 渲染器

2. **Run() - 执行 Agent 循环**

```go
func (a *Agent) Run(ctx context.Context, userInput string) error
```

流程:
```
1. 追加用户消息到会话
2. 循环 (最多 20 次):
   a. 构建 LLM 请求
   b. 调用 provider.Stream()
   c. 消费流事件
   d. 构建助手消息
   e. 如果无工具调用 → 结束
   f. 执行工具调用
   g. 追加工具结果 → 继续循环
```

3. **executeToolCall() - 执行工具调用**

```go
func (a *Agent) executeToolCall(ctx context.Context, tc session.Content) session.Content
```

流程:
- 权限检查
- 用户确认 (如需要)
- 执行工具
- 记录快照 (文件操作)
- 返回结果

#### permission.go - 权限管理

**权限等级:**

```go
const (
    PermAllow        = iota  // 直接允许
    PermNeedsConfirm         // 需要确认
    PermDeny                 // 硬拒绝
)
```

**PermissionGuard 结构:**

```go
type PermissionGuard struct {
    cfg           *config.Config
    sessionAllows map[string]bool  // 会话级白名单
}
```

**Check() - 权限检查**

```go
func (g *PermissionGuard) Check(toolName string, input map[string]interface{}) int
```

检查顺序:
1. 禁止命令检查 → PermDeny
2. 全局自动批准 → PermAllow
3. 会话级允许 → PermAllow
4. 读操作自动批准 → PermAllow
5. 命令白名单 → PermAllow
6. 默认 → PermNeedsConfirm

---

### 2.2 LLM 模块 (internal/llm/)

#### interface.go - 提供商接口

**Provider 接口:**

```go
type Provider interface {
    Stream(ctx context.Context, req *Request) (<-chan StreamEvent, error)
    Name() string
    CurrentModel() string
}
```

**Request 结构:**

```go
type Request struct {
    Model     string
    RawMsgs   interface{}      // []session.Message
    Tools     []ToolSchema
    System    string
    MaxTokens int
}
```

**StreamEvent 结构:**

```go
type StreamEvent struct {
    Type    string           // text_delta | tool_use_end | usage | error | done
    Delta   string
    ToolUse *ToolUseBlock
    Input   int              // input tokens
    Output  int              // output tokens
    Err     error
}
```

#### registry.go - 提供商注册

**注册机制:**

```go
var factories = map[string]ProviderFactory{}

func Register(name string, f ProviderFactory) {
    factories[name] = f
}

func New(name, apiKey, baseURL, model string) (Provider, error) {
    factory, ok := factories[name]
    if !ok {
        return nil, fmt.Errorf("unknown provider: %s", name)
    }
    return factory(apiKey, baseURL, model), nil
}
```

#### anthropic/provider.go - Anthropic 实现

**关键特性:**

- API 版本: `2023-06-01`
- Beta 特性: `interleaved-thinking-2025-05-14`
- 超时: 300 秒

**Stream() 实现:**

```go
func (p *Provider) Stream(ctx context.Context, req *llm.Request) (<-chan llm.StreamEvent, error)
```

流程:
1. 转换消息格式
2. 构建 API 请求
3. 发起 SSE 流式请求
4. 解析流事件
5. 转换为统一的 StreamEvent
6. 通过 channel 返回

**消息转换:**

- 系统消息 → 顶级 `system` 字段
- 工具调用 → `tool_use` 内容块
- 工具结果 → `tool_result` 内容块

---

### 2.3 工具模块 (internal/tools/)

#### interface.go - 工具接口

**Tool 接口:**

```go
type Tool interface {
    Name() string
    Description() string
    Schema() json.RawMessage
    Execute(ctx context.Context, input json.RawMessage) (*Result, error)
    Risk() RiskLevel
}
```

**Registry - 工具注册表:**

```go
type Registry struct {
    mu    sync.RWMutex
    tools map[string]Tool
}

var Global = &Registry{tools: map[string]Tool{}}
```

**关键方法:**

```go
func (r *Registry) Register(t Tool)
func (r *Registry) Get(name string) (Tool, bool)
func (r *Registry) All() []Tool
func (r *Registry) Schemas() []ToolSchema
```

#### filesystem/ - 文件系统工具

**沙箱检查:**

```go
var sandboxDenied = []string{
    "/etc/shadow", "/etc/passwd", "/etc/sudoers",
    "/.ssh", "/.gnupg",
}

func checkSandbox(path string) error {
    absPath, _ := filepath.Abs(path)
    for _, denied := range sandboxDenied {
        if strings.Contains(absPath, denied) {
            return fmt.Errorf("访问被拒绝: %s", denied)
        }
    }
    return nil
}
```

**文件快照机制:**

```go
var SnapshotFunc func(toolName, callID, filePath string, before, after []byte)
```

在 `write_file` 和 `edit_file` 执行前调用,保存文件原始内容。

---

### 2.4 会话模块 (internal/session/)

#### session.go - 会话管理

**Session 结构:**

```go
type Session struct {
    mu        sync.Mutex
    ID        string              // 时间戳 ID
    StartedAt time.Time
    Messages  []Message           // 对话历史
    Snapshots []FileSnapshot      // 文件快照
    Usage     TokenUsage          // Token 统计
    Model     string
}
```

**Message 结构:**

```go
type Message struct {
    Role    string    // system | user | assistant
    Content []Content
}

type Content struct {
    Type      string      // text | tool_use | tool_result | image
    Text      string
    ID        string      // 工具调用 ID
    Name      string      // 工具名称
    Input     interface{} // 工具输入
    ToolUseID string      // 关联的工具调用 ID
    IsError   bool
}
```

**关键方法:**

1. **AppendMessage() - 追加消息**

```go
func (s *Session) AppendMessage(msg Message)
```

线程安全,使用 mutex 保护。

2. **PushSnapshot() - 记录快照**

```go
func (s *Session) PushSnapshot(snap FileSnapshot)
```

保存文件修改前的状态。

3. **Undo() - 撤销修改**

```go
func (s *Session) Undo() error
```

流程:
- 弹出最后一个快照
- 恢复文件内容
- 截断相关消息

4. **RecordUsage() - 记录 Token 使用**

```go
func (s *Session) RecordUsage(input, output int)
```

累计 token 使用并估算费用。

---

### 2.5 配置模块 (internal/config/)

#### config.go - 配置管理

**Config 结构:**

```go
type Config struct {
    // LLM
    Provider  string
    Model     string
    MaxTokens int
    BaseURL   string

    // 安全
    AutoApprove         bool
    AutoApproveReads    bool
    AutoApproveCommands []string
    ForbiddenCommands   []string
    DangerouslySkip     bool

    // 文件
    BackupOnWrite bool

    // UI
    Theme    string
    Language string

    // 网络
    Proxy string

    // MCP
    MCPServers map[string]MCPServerConfig

    // 运行时
    Verbose bool
}
```

**Load() - 加载配置**

```go
func Load() (*Config, error)
```

加载顺序:
1. 默认配置
2. 用户级配置 (`~/.aicoder/config.json`)
3. 项目级配置 (`.aicoder/config.json`)
4. 环境变量

**APIKey() - 获取 API Key**

```go
func APIKey(provider string) string
```

根据提供商名称返回对应的环境变量。

---

### 2.6 上下文模块 (internal/context/)

#### collector.go - 上下文收集

**ProjectContext 结构:**

```go
type ProjectContext struct {
    RootDir     string
    AICoderMD   string
    GitInfo     string
    ProjectInfo string
}
```

**Collect() - 收集上下文**

```go
func Collect() (*ProjectContext, error)
```

收集内容:
1. 查找项目根目录
2. 读取 AICODER.md
3. 收集 Git 信息
4. 检测项目类型

**BuildSystemPrompt() - 构建系统提示**

```go
func (pc *ProjectContext) BuildSystemPrompt() string
```

组合:
- 基础角色定义
- AICODER.md 内容
- 项目环境信息
- Git 状态

---

## 3. 数据流分析

### 3.1 完整请求流程

```
用户输入
    ↓
cmd/root.go (CLI 解析)
    ↓
agent.Run()
    ↓
session.AppendMessage(user)
    ↓
构建 LLM Request
    ├─ 系统提示 (context.BuildSystemPrompt)
    ├─ 历史消息 (session.Messages)
    ├─ 工具列表 (tools.Global.Schemas)
    └─ 配置 (config)
    ↓
provider.Stream()
    ↓
消费 StreamEvent
    ├─ text_delta → renderer.Write()
    ├─ tool_use_end → 收集工具调用
    ├─ usage → session.RecordUsage()
    └─ done → 结束流
    ↓
session.AppendMessage(assistant)
    ↓
执行工具调用
    ├─ guard.Check() (权限检查)
    ├─ ui.Confirm() (用户确认)
    ├─ tool.Execute() (执行工具)
    └─ session.PushSnapshot() (保存快照)
    ↓
session.AppendMessage(tool_result)
    ↓
继续 Agent 循环 (最多 20 次)
```

### 3.2 消息转换流程

**会话消息 → LLM 请求:**

```
session.Message (统一格式)
    ↓
provider 转换
    ├─ Anthropic: 转换为 Anthropic 格式
    └─ OpenAI: 转换为 OpenAI 格式
    ↓
API 请求
```

**LLM 响应 → 会话消息:**

```
API 响应 (流式)
    ↓
provider 解析
    ↓
StreamEvent (统一格式)
    ↓
agent 消费
    ↓
session.Message (统一格式)
```

---

## 4. 关键算法

### 4.1 权限检查算法

```go
func (g *PermissionGuard) Check(toolName string, input map[string]interface{}) int {
    // 1. 检查禁止命令
    if toolName == "run_command" {
        cmd := input["command"].(string)
        for _, pattern := range forbiddenPatterns {
            if strings.Contains(cmd, pattern) {
                return PermDeny
            }
        }
    }

    // 2. 全局自动批准
    if g.cfg.AutoApprove {
        return PermAllow
    }

    // 3. 会话级允许
    key := fmt.Sprintf("%s:%v", toolName, input)
    if g.sessionAllows[key] {
        return PermAllow
    }

    // 4. 读操作自动批准
    tool, _ := tools.Global.Get(toolName)
    if tool.Risk() == tools.RiskLow && g.cfg.AutoApproveReads {
        return PermAllow
    }

    // 5. 命令白名单
    if toolName == "run_command" {
        cmd := input["command"].(string)
        for _, allowed := range g.cfg.AutoApproveCommands {
            if strings.HasPrefix(cmd, allowed) {
                return PermAllow
            }
        }
    }

    // 6. 默认需要确认
    return PermNeedsConfirm
}
```

### 4.2 文件快照算法

```go
func (s *Session) PushSnapshot(snap FileSnapshot) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.Snapshots = append(s.Snapshots, snap)
}

func (s *Session) Undo() error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if len(s.Snapshots) == 0 {
        return errors.New("没有可撤销的操作")
    }

    // 弹出最后一个快照
    snap := s.Snapshots[len(s.Snapshots)-1]
    s.Snapshots = s.Snapshots[:len(s.Snapshots)-1]

    // 恢复文件
    if err := os.WriteFile(snap.Path, snap.Before, 0644); err != nil {
        return err
    }

    // 截断消息 (移除相关的工具调用和结果)
    // ...

    return nil
}
```

### 4.3 Token 成本估算

```go
var prices = map[string][2]float64{
    "claude-opus-4-5":    {15.0, 75.0},   // $/M tokens
    "claude-sonnet-4-5":  {3.0, 15.0},
    "claude-haiku-4-5":   {0.25, 1.25},
    "gpt-4o":             {5.0, 15.0},
    "gpt-4o-mini":        {0.15, 0.60},
    "DeepSeek-R1":        {0.20, 3.00},
}

func (u *TokenUsage) EstimateCost(model string) float64 {
    price, ok := prices[model]
    if !ok {
        return 0
    }

    inputCost := float64(u.InputTokens) / 1_000_000 * price[0]
    outputCost := float64(u.OutputTokens) / 1_000_000 * price[1]

    return inputCost + outputCost
}
```

---

*本文档持续更新中,如有问题请提交 Issue 或 PR。*
