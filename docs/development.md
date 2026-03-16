# 开发者指南

**产品名称:** aicoder
**版本:** v1.0
**文档状态:** 正式版
**创建日期:** 2026-03-16
**关联文档:** [arch.md](./arch.md) · [code-guide.md](./code-guide.md) · [tools-reference.md](./tools-reference.md)

---

## 目录

- [1. 开发环境搭建](#1-开发环境搭建)
- [2. 项目结构](#2-项目结构)
- [3. 开发工作流](#3-开发工作流)
- [4. 添加新功能](#4-添加新功能)
- [5. 测试指南](#5-测试指南)
- [6. 调试技巧](#6-调试技巧)
- [7. 代码规范](#7-代码规范)
- [8. 发布流程](#8-发布流程)

---

## 1. 开发环境搭建

### 1.1 前置要求

- **Go 1.22+**
- **Git**
- **Make** (可选,用于构建脚本)

### 1.2 克隆项目

```bash
git clone https://github.com/iminders/aicoder.git
cd aicoder
```

### 1.3 安装依赖

```bash
# 下载 Go 模块依赖
go mod download

# 验证依赖
go mod verify
```

### 1.4 配置 API Key

```bash
# Anthropic (默认)
export ANTHROPIC_API_KEY="sk-ant-..."

# 或使用 OpenAI
export OPENAI_API_KEY="sk-..."
```

### 1.5 构建项目

```bash
# 使用 Makefile
make build

# 或直接使用 go build
go build -o aicoder .
```

### 1.6 运行项目

```bash
# 运行编译后的二进制
./aicoder

# 或直接运行
go run main.go
```

---

## 2. 项目结构

### 2.1 目录结构

```
aicoder/
├── main.go                    # 程序入口
├── cmd/                       # CLI 命令层
│   └── root.go               # 根命令、全局 flags
├── internal/                  # 核心业务逻辑
│   ├── agent/                # Agent 编排核心
│   │   ├── agent.go          # Agent 主循环
│   │   └── permission.go     # 权限管理
│   ├── llm/                  # LLM 提供商适配层
│   │   ├── interface.go      # Provider 接口定义
│   │   ├── registry.go       # 提供商注册表
│   │   ├── anthropic/        # Anthropic Claude 实现
│   │   └── openai/           # OpenAI / 兼容端点实现
│   ├── tools/                # 内置工具系统
│   │   ├── interface.go      # Tool 接口定义
│   │   ├── filesystem/       # 文件操作工具
│   │   ├── shell/            # Shell 命令执行工具
│   │   └── search/           # 代码搜索工具
│   ├── session/              # 会话管理 + 快照 + Token 统计
│   │   └── session.go
│   ├── config/               # 多级配置加载
│   │   └── config.go
│   ├── context/              # 项目上下文收集
│   │   └── collector.go
│   ├── slash/                # 斜杠命令处理
│   │   └── commands.go
│   ├── ui/                   # 终端渲染
│   │   └── renderer.go
│   └── logger/               # 日志系统
│       └── logger.go
├── pkg/                       # 可复用的公共库
│   ├── diff/                 # Unified diff 生成与应用
│   └── version/              # 版本信息
├── docs/                      # 文档
│   ├── README.md
│   ├── prd.md
│   ├── arch.md
│   ├── development.md
│   ├── code-guide.md
│   └── tools-reference.md
├── Makefile                   # 构建脚本
├── go.mod                     # Go 模块定义
└── go.sum                     # 依赖校验文件
```

### 2.2 核心模块说明

| 模块 | 职责 | 关键文件 |
|------|------|---------|
| **agent** | Agent 循环、工具调用编排、权限管理 | agent.go, permission.go |
| **llm** | LLM 提供商抽象和实现 | interface.go, anthropic/, openai/ |
| **tools** | 工具系统、工具注册表 | interface.go, filesystem/, shell/, search/ |
| **session** | 会话管理、对话历史、文件快照 | session.go |
| **config** | 配置加载、API Key 管理 | config.go |
| **context** | 项目上下文收集、系统提示构建 | collector.go |
| **slash** | 斜杠命令路由和处理 | commands.go |
| **ui** | 终端渲染、流式输出 | renderer.go |
| **logger** | 日志系统 | logger.go |

---

## 3. 开发工作流

### 3.1 创建新分支

```bash
# 从 main 分支创建功能分支
git checkout main
git pull origin main
git checkout -b feature/your-feature-name
```

### 3.2 开发和测试

```bash
# 运行测试
make test

# 或
go test ./...

# 运行特定包的测试
go test ./internal/agent/

# 查看测试覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 3.3 代码检查

```bash
# 运行 go vet
make vet

# 或
go vet ./...

# 格式化代码
go fmt ./...

# 运行 golangci-lint (如果已安装)
golangci-lint run
```

### 3.4 提交代码

```bash
# 添加修改
git add .

# 提交 (使用有意义的提交信息)
git commit -m "feat: add new feature"

# 推送到远程
git push origin feature/your-feature-name
```

### 3.5 创建 Pull Request

1. 在 GitHub 上创建 Pull Request
2. 填写 PR 描述,说明修改内容
3. 等待 CI 检查通过
4. 等待代码审查

---

## 4. 添加新功能

### 4.1 添加新的 LLM 提供商

#### 步骤 1: 创建提供商目录

```bash
mkdir -p internal/llm/yourprovider
```

#### 步骤 2: 实现 Provider 接口

创建 `internal/llm/yourprovider/provider.go`:

```go
package yourprovider

import (
    "context"
    "github.com/iminders/aicoder/internal/llm"
)

type Provider struct {
    apiKey  string
    baseURL string
    model   string
}

func New(apiKey, baseURL, model string) *Provider {
    return &Provider{
        apiKey:  apiKey,
        baseURL: baseURL,
        model:   model,
    }
}

func (p *Provider) Stream(ctx context.Context, req *llm.Request) (<-chan llm.StreamEvent, error) {
    // 实现流式调用逻辑
    ch := make(chan llm.StreamEvent)

    go func() {
        defer close(ch)
        // 发送流事件
        ch <- llm.StreamEvent{
            Type:  "text_delta",
            Delta: "Hello",
        }
        // ...
    }()

    return ch, nil
}

func (p *Provider) Name() string {
    return "yourprovider"
}

func (p *Provider) CurrentModel() string {
    return p.model
}
```

#### 步骤 3: 注册提供商

在 `internal/llm/registry.go` 的 `init()` 函数中注册:

```go
func init() {
    Register("yourprovider", func(apiKey, baseURL, model string) Provider {
        return yourprovider.New(apiKey, baseURL, model)
    })
}
```

#### 步骤 4: 添加配置支持

在 `internal/config/config.go` 中添加 API Key 支持:

```go
func APIKey(provider string) string {
    switch provider {
    case "yourprovider":
        return os.Getenv("YOURPROVIDER_API_KEY")
    // ...
    }
}
```

#### 步骤 5: 测试

```bash
export YOURPROVIDER_API_KEY="your-key"
./aicoder --provider yourprovider --model your-model
```

### 4.2 添加新的工具

#### 步骤 1: 创建工具文件

在 `internal/tools/` 下创建工具文件,例如 `internal/tools/filesystem/copy_file.go`:

```go
package filesystem

import (
    "context"
    "encoding/json"
    "io"
    "os"

    "github.com/iminders/aicoder/internal/tools"
)

type CopyFileTool struct{}

func (t *CopyFileTool) Name() string {
    return "copy_file"
}

func (t *CopyFileTool) Description() string {
    return "复制文件到新位置"
}

func (t *CopyFileTool) Schema() json.RawMessage {
    return json.RawMessage(`{
        "type": "object",
        "properties": {
            "source": {
                "type": "string",
                "description": "源文件路径"
            },
            "destination": {
                "type": "string",
                "description": "目标文件路径"
            }
        },
        "required": ["source", "destination"]
    }`)
}

func (t *CopyFileTool) Execute(ctx context.Context, input json.RawMessage) (*tools.Result, error) {
    var params struct {
        Source      string `json:"source"`
        Destination string `json:"destination"`
    }

    if err := json.Unmarshal(input, &params); err != nil {
        return &tools.Result{
            Content: "参数解析失败: " + err.Error(),
            IsError: true,
        }, nil
    }

    // 沙箱检查
    if err := checkSandbox(params.Source); err != nil {
        return &tools.Result{
            Content: err.Error(),
            IsError: true,
        }, nil
    }

    if err := checkSandbox(params.Destination); err != nil {
        return &tools.Result{
            Content: err.Error(),
            IsError: true,
        }, nil
    }

    // 复制文件
    src, err := os.Open(params.Source)
    if err != nil {
        return &tools.Result{
            Content: "打开源文件失败: " + err.Error(),
            IsError: true,
        }, nil
    }
    defer src.Close()

    dst, err := os.Create(params.Destination)
    if err != nil {
        return &tools.Result{
            Content: "创建目标文件失败: " + err.Error(),
            IsError: true,
        }, nil
    }
    defer dst.Close()

    if _, err := io.Copy(dst, src); err != nil {
        return &tools.Result{
            Content: "复制文件失败: " + err.Error(),
            IsError: true,
        }, nil
    }

    return &tools.Result{
        Content: "文件复制成功",
        IsError: false,
    }, nil
}

func (t *CopyFileTool) Risk() tools.RiskLevel {
    return tools.RiskMedium
}
```

#### 步骤 2: 注册工具

在 `cmd/root.go` 或工具初始化代码中注册:

```go
tools.Global.Register(&filesystem.CopyFileTool{})
```

#### 步骤 3: 测试工具

```bash
# 运行测试
go test ./internal/tools/filesystem/

# 手动测试
./aicoder
> 帮我复制 test.txt 到 test_copy.txt
```

### 4.3 添加新的斜杠命令

#### 步骤 1: 在 `internal/slash/commands.go` 中添加命令处理

```go
func HandleCommand(cmd string, sess *session.Session, cfg *config.Config) (string, error) {
    parts := strings.Fields(cmd)
    if len(parts) == 0 {
        return "", nil
    }

    switch parts[0] {
    // ... 现有命令

    case "/stats":
        // 显示会话统计信息
        return formatStats(sess), nil

    // ...
    }
}

func formatStats(sess *session.Session) string {
    return fmt.Sprintf(`会话统计:
- 消息数: %d
- 文件快照: %d
- Token 使用: %d (输入) + %d (输出)
- 预估费用: $%.4f`,
        len(sess.Messages),
        len(sess.Snapshots),
        sess.Usage.InputTokens,
        sess.Usage.OutputTokens,
        sess.Usage.EstimatedCost,
    )
}
```

#### 步骤 2: 更新帮助信息

在 `/help` 命令中添加新命令的说明。

#### 步骤 3: 测试命令

```bash
./aicoder
> /stats
```

---

## 5. 测试指南

### 5.1 单元测试

#### 编写测试

创建 `*_test.go` 文件:

```go
package agent

import (
    "testing"
)

func TestPermissionCheck(t *testing.T) {
    guard := NewPermissionGuard(&config.Config{
        AutoApprove: false,
        ForbiddenCommands: []string{"rm -rf /"},
    })

    // 测试禁止命令
    perm := guard.Check("run_command", map[string]interface{}{
        "command": "rm -rf /",
    })

    if perm != PermDeny {
        t.Errorf("expected PermDeny, got %v", perm)
    }
}
```

#### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/agent/

# 运行特定测试
go test -run TestPermissionCheck ./internal/agent/

# 显示详细输出
go test -v ./...

# 查看覆盖率
go test -cover ./...
```

### 5.2 集成测试

创建集成测试文件 `internal/agent/agent_integration_test.go`:

```go
// +build integration

package agent

import (
    "context"
    "testing"
)

func TestAgentFullCycle(t *testing.T) {
    // 跳过如果没有 API Key
    if os.Getenv("ANTHROPIC_API_KEY") == "" {
        t.Skip("ANTHROPIC_API_KEY not set")
    }

    // 创建 Agent
    cfg := &config.Config{
        Provider:  "anthropic",
        Model:     "claude-sonnet-4-5",
        AutoApprove: true,
    }

    agent, err := New(cfg)
    if err != nil {
        t.Fatal(err)
    }

    // 运行测试任务
    err = agent.Run(context.Background(), "读取 README.md 文件")
    if err != nil {
        t.Fatal(err)
    }

    // 验证结果
    if len(agent.sess.Messages) == 0 {
        t.Error("expected messages in session")
    }
}
```

运行集成测试:

```bash
go test -tags=integration ./...
```

### 5.3 性能测试

创建性能测试:

```go
func BenchmarkAgentRun(b *testing.B) {
    cfg := &config.Config{
        Provider: "anthropic",
        Model:    "claude-sonnet-4-5",
    }

    agent, _ := New(cfg)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        agent.Run(context.Background(), "hello")
    }
}
```

运行性能测试:

```bash
go test -bench=. ./...
```

---

## 6. 调试技巧

### 6.1 启用详细日志

```bash
# 启用 verbose 模式
./aicoder --verbose

# 查看日志文件
tail -f ~/.aicoder/logs/aicoder-$(date +%Y%m%d).log
```

### 6.2 使用 Delve 调试器

```bash
# 安装 Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 启动调试
dlv debug . -- --verbose

# 在代码中设置断点
(dlv) break internal/agent/agent.go:123
(dlv) continue
```

### 6.3 打印调试信息

```go
import "github.com/iminders/aicoder/internal/logger"

func someFunction() {
    logger.Debug("Debug info: %v", someValue)
    logger.Info("Info: %s", someString)
}
```

### 6.4 模拟 LLM 响应

创建 mock Provider 用于测试:

```go
type MockProvider struct {
    responses []string
}

func (m *MockProvider) Stream(ctx context.Context, req *llm.Request) (<-chan llm.StreamEvent, error) {
    ch := make(chan llm.StreamEvent)

    go func() {
        defer close(ch)
        for _, resp := range m.responses {
            ch <- llm.StreamEvent{
                Type:  "text_delta",
                Delta: resp,
            }
        }
        ch <- llm.StreamEvent{Type: "done"}
    }()

    return ch, nil
}
```

---

## 7. 代码规范

### 7.1 命名规范

- **包名**: 小写,单数形式,简短有意义 (例如: `agent`, `config`, `tools`)
- **文件名**: 小写,下划线分隔 (例如: `agent.go`, `permission.go`)
- **类型名**: 大驼峰 (例如: `Agent`, `Provider`, `ToolResult`)
- **函数名**: 大驼峰(导出) 或 小驼峰(私有) (例如: `New()`, `checkSandbox()`)
- **常量**: 大驼峰或全大写 (例如: `RiskLow`, `MAX_ITERATIONS`)

### 7.2 注释规范

```go
// Package agent 提供 AI Agent 的核心循环和工具调用编排功能
package agent

// Agent 表示一个 AI 编程助手实例
// 它管理与 LLM 的交互、工具调用和权限控制
type Agent struct {
    // cfg 是 Agent 的配置
    cfg *config.Config
    // provider 是 LLM 提供商
    provider llm.Provider
}

// New 创建一个新的 Agent 实例
// 它会初始化会话、收集项目上下文并构建系统提示词
func New(cfg *config.Config) (*Agent, error) {
    // ...
}
```

### 7.3 错误处理

```go
// 好的错误处理
result, err := someFunction()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// 避免忽略错误
result, _ := someFunction() // 不好!
```

### 7.4 代码格式化

```bash
# 使用 gofmt 格式化代码
go fmt ./...

# 使用 goimports (自动管理 import)
goimports -w .
```

---

## 8. 发布流程

### 8.1 版本号规范

遵循语义化版本 (Semantic Versioning):

- **主版本号**: 不兼容的 API 变更
- **次版本号**: 向后兼容的功能新增
- **修订号**: 向后兼容的问题修正

例如: `v1.2.3`

### 8.2 发布步骤

#### 步骤 1: 更新版本号

在 `pkg/version/version.go` 中更新版本号:

```go
const Version = "1.1.0"
```

#### 步骤 2: 更新 CHANGELOG

创建或更新 `CHANGELOG.md`:

```markdown
## [1.1.0] - 2026-03-20

### Added
- 新增 MCP 客户端支持
- 新增 Ollama 本地模型支持

### Changed
- 改进权限确认 UI

### Fixed
- 修复文件快照回滚问题
```

#### 步骤 3: 提交变更

```bash
git add .
git commit -m "chore: bump version to v1.1.0"
git push origin main
```

#### 步骤 4: 创建 Git Tag

```bash
git tag -a v1.1.0 -m "Release v1.1.0"
git push origin v1.1.0
```

#### 步骤 5: 触发 CI/CD

GitHub Actions 会自动:
1. 运行测试
2. 构建多平台二进制
3. 创建 GitHub Release
4. 上传构建产物

#### 步骤 6: 发布到包管理器

```bash
# 更新 Homebrew formula
# 更新 npm 包
# 更新安装脚本
```

---

## 附录

### A. 常用命令

```bash
# 构建
make build

# 测试
make test

# 代码检查
make vet

# 清理
make clean

# 交叉编译
make cross

# 查看帮助
make help
```

### B. 环境变量

| 变量名 | 说明 | 示例 |
|--------|------|------|
| `ANTHROPIC_API_KEY` | Anthropic API Key | `sk-ant-...` |
| `OPENAI_API_KEY` | OpenAI API Key | `sk-...` |
| `AICODER_MODEL` | 覆盖模型 | `claude-opus-4-5` |
| `AICODER_PROVIDER` | 覆盖提供商 | `openai` |
| `AICODER_BASE_URL` | 自定义 API 端点 | `http://localhost:8080` |
| `HTTPS_PROXY` | 代理设置 | `http://127.0.0.1:7890` |

### C. 有用的资源

- [Go 官方文档](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Anthropic API 文档](https://docs.anthropic.com/)
- [OpenAI API 文档](https://platform.openai.com/docs/)

---

*本文档持续更新中,如有问题请提交 Issue 或 PR。*
