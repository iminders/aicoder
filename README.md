# aicoder

> 一个开源、可扩展、支持多模型的 AI 编程 CLI 工具

`aicoder` 让你在终端里直接与 AI 结对编程——读写文件、执行命令、搜索代码，驱动完整的 Agent 循环，无需离开命令行。

```
  ██████╗  ██╗ ██████╗ ██████╗ ██████╗ ███████╗██████╗
 ██╔══██╗ ██║██╔════╝██╔═══██╗██╔══██╗██╔════╝██╔══██╗
 ███████║ ██║██║     ██║   ██║██║  ██║█████╗  ██████╔╝
 ██╔══██║ ██║██║     ██║   ██║██║  ██║██╔══╝  ██╔══██╗
 ██║  ██║ ██║╚██████╗╚██████╔╝██████╔╝███████╗██║  ██║
 ╚═╝  ╚═╝ ╚═╝ ╚═════╝ ╚═════╝ ╚═════╝ ╚══════╝╚═╝  ╚═╝
```

---

## 特性

- **三种交互模式**：交互式 REPL、单次执行、Unix 管道
- **Agent Loop**：AI 可自主调用工具，直到任务完成
- **内置工具集**：文件读写编辑、Shell 命令执行、全局代码搜索
- **权限确认机制**：危险操作前要求用户确认，支持黑名单和白名单
- **多 LLM 提供商**：Anthropic Claude、OpenAI GPT（兼容任何 OpenAI 格式端点）
- **项目上下文感知**：自动读取 `.AICODER.md`、Git 状态、项目依赖
- **斜杠命令**：`/diff`、`/undo`、`/commit`、`/cost` 等 11 个内置命令
- **纯 Go 标准库**：无外部依赖，单二进制，跨平台

---

## 快速开始

### 安装

```bash
# 从源码编译（需要 Go 1.22+）
git clone https://github.com/iminders/aicoder
cd aicoder
make build
mv aicoder /usr/local/bin/
```

### 设置 API Key

```bash
# Anthropic（默认）
export ANTHROPIC_API_KEY="sk-ant-..."

# 或使用 OpenAI
export OPENAI_API_KEY="sk-..."

```

### 使用

```bash
# 交互式模式
aicoder

# 单次执行
aicoder "解释 src/auth.go 中的 JWT 验证逻辑"
aicoder --file ./error.log "分析这个日志，找出根因"

# 管道模式
cat error.log | aicoder "分析这个报错"
git diff HEAD~1 | aicoder "为这次变更生成 commit message"

# 指定模型
aicoder --model claude-opus-4-5 "重构这个服务"
aicoder --provider openai --model gpt-4o "帮我写单元测试"
```

---

## 交互式模式

启动后，输入自然语言指令，AI 会自动调用工具完成任务：

```
$ aicoder
> 帮我重构 src/utils.py，提升可读性
⚙  执行 read_file...
✅ read_file (12ms)
[AI 分析并展示重构方案]
⚙  执行 edit_file...
┌────────────────────────────────────────────────────────────┐
│  🔧 工具调用请求                                            │
│  工具：edit_file                                           │
│  path: src/utils.py                                        │
│  风险等级：中                                               │
├────────────────────────────────────────────────────────────┤
│  [Y] 允许   [N] 拒绝   [A] 本次会话始终允许                │
└────────────────────────────────────────────────────────────┘
请选择 [Y/n/a]: y
✅ edit_file (8ms)

> /diff
> /commit "refactor: improve utils.py readability"
```

---

## 斜杠命令

| 命令 | 功能 |
|---|---|
| `/help` | 显示帮助信息 |
| `/clear` | 清空会话上下文（保留系统提示词） |
| `/history` | 查看对话历史摘要 |
| `/undo` | 撤销上一次文件修改 |
| `/diff` | 查看本次会话所有文件变更 |
| `/commit [msg]` | Git 提交本次会话变更 |
| `/cost` | 查看 Token 用量和费用估算 |
| `/model [name]` | 查看或切换 AI 模型 |
| `/config` | 查看当前配置 |
| `/init` | 在当前目录生成 .AICODER.md 模板 |
| `/exit` | 退出程序 |

---

## 内置工具

### 文件系统
| 工具 | 说明 |
|---|---|
| `read_file` | 读取文件，支持按行范围读取 |
| `write_file` | 写入文件（自动创建目录） |
| `edit_file` | 精确替换——用 old_string/new_string 修改 |
| `list_dir` | 递归目录树（自动忽略 node_modules 等） |
| `search_files` | 正则搜索，返回文件:行号:内容 |
| `delete_file` | 删除文件（高风险，需确认） |

### Shell
| 工具 | 说明 |
|---|---|
| `run_command` | 执行 Shell 命令，60 秒超时 |
| `run_background` | 后台启动长时进程 |

### 搜索
| 工具 | 说明 |
|---|---|
| `grep_search` | 全目录正则搜索，支持 glob 过滤和上下文行 |

---

## .AICODER.md 项目配置

在项目根目录创建 `.AICODER.md`（或运行 `/init`），AI 会在每次会话开始时自动加载它作为项目级系统提示词：

```markdown
# 项目说明
这是一个基于 Go 的微服务项目，使用 gRPC 通信。

# 代码规范
- 使用 gofmt 格式化代码
- 错误必须显式处理，禁止 panic
- 测试覆盖率 > 80%

# 常用命令
- 运行测试：make test
- 构建：make build
- 启动服务：make run
```

---

## 配置

配置文件路径：`~/.aicoder/config.json`（用户级）、`.aicoder/config.json`（项目级）

```json
{
  "provider": "anthropic",
  "model": "claude-sonnet-4-5",
  "maxTokens": 8192,
  "autoApprove": false,
  "autoApproveReads": true,
  "autoApproveCommands": ["go test", "npm test", "git status"],
  "forbiddenCommands": ["rm -rf /", "mkfs", "dd if="],
  "backupOnWrite": true,
  "theme": "dark",
  "language": "zh-CN",
  "proxy": ""
}
```

本地部署DeepSeek R1配置

`export OPENAI_API_KEY=local`

`~/.aicoder/config.json`
```json
{
  "provider": "openai",
  "model": "DeepSeek-R1",
  "baseUrl": "http://127.0.0.1:10002",
  "maxTokens": 8192,
  "autoApprove": false,
  "autoApproveReads": true,
  "autoApproveCommands": ["go test", "npm test", "git status"],
  "forbiddenCommands": ["rm -rf /", "mkfs", "dd if="],
  "backupOnWrite": true,
  "theme": "dark",
  "language": "zh-CN",
  "proxy": ""
}
```

**配置优先级**：CLI 参数 > 环境变量 > 项目配置 > 用户配置 > 默认值

**环境变量**：
```bash
ANTHROPIC_API_KEY   # Anthropic API Key
OPENAI_API_KEY      # OpenAI API Key
AICODER_MODEL       # 覆盖模型
AICODER_PROVIDER    # 覆盖提供商
AICODER_BASE_URL    # 自定义 API 端点
HTTPS_PROXY         # 代理设置
```

---

## 安全

- **沙箱保护**：默认禁止访问 `~/.ssh`、`/etc/shadow`、`/etc/passwd` 等敏感路径
- **命令黑名单**：`rm -rf /`、`mkfs`、`dd if=` 等危险命令直接拒绝，不可绕过
- **权限确认**：中高风险操作默认要求用户确认（`Y/N/A`）
- **快照与撤销**：每次文件写操作自动保存快照，支持 `/undo` 即时回滚
- **CI 模式**：`--dangerously-skip-permissions` 跳过所有确认（仅用于自动化流水线）

---

## 开发

```bash
# 运行测试
make test

# 查看覆盖率
go test -cover ./...

# 交叉编译
make cross

# 代码检查
make vet
```

### 项目结构

```
aicoder/
├── main.go                  # 程序入口
├── cmd/root.go              # CLI 解析，三种运行模式
├── internal/
│   ├── agent/               # Agent Loop + 权限确认
│   ├── llm/                 # LLM 提供商接口和实现
│   │   ├── anthropic/       # Anthropic Claude
│   │   └── openai/          # OpenAI / 兼容端点
│   ├── tools/               # 内置工具系统
│   │   ├── filesystem/      # 文件操作工具
│   │   ├── shell/           # Shell 执行工具
│   │   └── search/          # 代码搜索工具
│   ├── session/             # 会话管理 + 快照 + Token 统计
│   ├── config/              # 多级配置加载
│   ├── context/             # 项目上下文收集
│   ├── slash/               # 斜杠命令处理
│   ├── ui/                  # 终端渲染
│   └── logger/              # 日志
└── pkg/
    ├── diff/                # Unified diff 生成与应用
    └── version/             # 版本信息
```

---

## 路线图

- **v1.1**：MCP（Model Context Protocol）客户端支持、Ollama 本地模型、多步撤销
- **v1.2**：插件系统（自定义工具）、Windows 原生支持
- **v2.0**：多 Agent 并行任务、Web Dashboard

---

## License

MIT License — 详见 [LICENSE](LICENSE)
