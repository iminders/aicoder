# 产品需求文档（PRD）

**产品名称：** AI 编程助手 CLI（暂定名：`aicoder`）  
**版本：** v1.0  
**文档状态：** 草稿  
**创建日期：** 2026-03-13  
**作者：** [待填写]

---

## 1. 产品概述

### 1.1 背景与动机

随着大语言模型（LLM）能力的快速提升，AI 辅助编程已经从简单的代码补全演进为真正意义上的"结对编程"。Claude Code 等工具证明了在终端环境中直接与 AI 协同完成复杂软件工程任务的可行性。

本产品旨在构建一个**开源、可扩展、支持多模型**的 AI 编程 CLI 工具，帮助开发者在本地终端环境中高效完成代码编写、调试、重构、测试等全流程任务。

### 1.2 产品定位

面向专业开发者，提供一个**以终端为核心交互界面**的 AI 编程助手。与 IDE 插件不同，CLI 工具天然适合：

- 服务器端 / SSH 远程开发环境
- CI/CD 流水线中的自动化任务
- 对 IDE 无依赖的轻量化工作流
- 习惯终端操作的开发者群体

### 1.3 目标用户

| 用户类型 | 使用场景 |
|---|---|
| 后端/全栈开发者 | 日常编码、代码审查、文档生成 |
| DevOps/SRE 工程师 | 脚本编写、配置调试、日志分析 |
| 开源贡献者 | 快速理解陌生代码库、贡献代码 |
| AI 工具开发者 | 二次开发、集成到自定义工作流 |

---

## 2. 核心功能需求

### 2.1 交互模式

#### 2.1.1 交互式对话模式（Interactive Mode）

用户通过 `aicoder` 命令启动一个持久会话，在 REPL（Read-Eval-Print Loop）循环中与 AI 进行多轮对话。

```
$ aicoder
> 帮我重构 src/utils.py，提升可读性
> 给刚才的修改添加单元测试
> 提交这些变更，commit message 用英文
```

**需求要点：**
- 支持多轮上下文记忆，会话期间保持对话历史
- 支持 `/` 前缀的斜杠命令（见 2.3 节）
- 支持 `↑/↓` 方向键浏览历史输入
- 支持 `Ctrl+C` 中断当前 AI 任务，不退出会话
- 支持 `Ctrl+D` 或 `/exit` 退出

#### 2.1.2 单次执行模式（One-shot Mode）

通过参数直接传入指令，适用于脚本和自动化场景。

```bash
aicoder "解释 src/auth.go 中的 JWT 验证逻辑"
aicoder --file ./error.log "分析这个日志文件，找出根因"
```

#### 2.1.3 管道模式（Pipe Mode）

支持标准输入输出，与 Unix 管道无缝协作。

```bash
cat error.log | aicoder "分析这个报错"
git diff HEAD~1 | aicoder "为这次 diff 生成 commit message"
```

---

### 2.2 工具调用能力（Tool Use / Agent Loop）

AI 能够调用一组内置工具来自主完成任务，工具调用前须向用户确认（可配置）。

#### 2.2.1 文件系统工具

| 工具名 | 功能描述 |
|---|---|
| `read_file` | 读取文件内容（支持按行范围读取） |
| `write_file` | 写入或覆盖文件 |
| `edit_file` | 按 diff/patch 方式精确修改文件 |
| `list_dir` | 列出目录结构 |
| `search_files` | 在文件中搜索文本（支持正则） |
| `delete_file` | 删除文件（需二次确认） |

#### 2.2.2 命令执行工具

| 工具名 | 功能描述 |
|---|---|
| `run_command` | 在 Shell 中执行命令 |
| `run_background` | 在后台启动长时进程（如 dev server） |

**安全策略：**
- 危险命令（`rm -rf`、`sudo`、网络请求等）需用户明确授权
- 支持 `--dangerously-skip-permissions` 标志跳过所有确认（用于 CI 场景，需显式声明）
- 支持配置命令黑名单

#### 2.2.3 代码搜索工具

| 工具名 | 功能描述 |
|---|---|
| `grep_search` | 基于正则的全局代码搜索 |
| `ast_search` | 基于 AST 的语义搜索（如"找到所有导出函数"） |
| `web_search` | 联网搜索（可选，需配置） |

---

### 2.3 斜杠命令（Slash Commands）

在交互式对话中，用户可以通过 `/` 前缀触发内置命令：

| 命令 | 功能 |
|---|---|
| `/help` | 显示帮助信息 |
| `/clear` | 清空当前会话上下文 |
| `/history` | 查看对话历史 |
| `/undo` | 撤销上一次文件修改 |
| `/diff` | 查看本次会话中所有文件变更的 diff |
| `/commit` | 将本次会话的文件变更提交到 Git |
| `/cost` | 查看当前会话已消耗的 Token 及费用估算 |
| `/model` | 切换使用的 AI 模型 |
| `/config` | 查看或修改配置项 |
| `/init` | 在当前项目初始化 `aicoder` 配置（生成 `.AICODER.md`） |
| `/exit` | 退出程序 |

---

### 2.4 项目上下文感知（Project Context）

#### 2.4.1 .AICODER.md

项目根目录下的 `.AICODER.md` 文件作为**项目级系统提示词**，AI 在每次会话开始时自动加载：

```markdown
# 项目说明
这是一个基于 Go 的微服务项目，使用 Protobuf 定义接口。

# 代码规范
- 使用 gofmt 格式化
- 错误处理必须显式，不允许 panic
- 测试覆盖率要求 > 80%

# 常用命令
- 运行测试：make test
- 构建：make build
```

#### 2.4.2 自动上下文收集

AI 在处理任务时会自动探索相关文件，包括：
- 当前 Git 仓库信息（`git log`、`git status`、`git diff`）
- 项目语言和依赖（`package.json`、`go.mod`、`pyproject.toml` 等）
- 目录结构摘要

---

### 2.5 多模型支持

支持接入不同的 LLM 提供商，通过统一的适配层抽象差异：

| 提供商 | 支持的模型示例 |
|---|---|
| Anthropic | Claude Sonnet、Claude Opus |
| OpenAI | GPT-4o、o3 |
| Google | Gemini 2.0 Flash、Gemini 2.5 Pro |
| Ollama | 本地部署的任意模型 |
| 自定义 | 兼容 OpenAI API 格式的任意端点 |

配置方式：

```bash
# 通过环境变量
export AICODER_MODEL=claude-opus-4-5
export ANTHROPIC_API_KEY=sk-ant-...

# 通过 CLI 参数
aicoder --model gpt-4o "重构这个函数"
```

---

### 2.6 MCP（Model Context Protocol）支持

支持作为 **MCP 客户端**，连接外部 MCP 服务器以扩展 AI 能力：

```json
// ~/.aicoder/config.json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": { "GITHUB_TOKEN": "ghp_xxx" }
    },
    "postgres": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-postgres"],
      "env": { "DATABASE_URL": "postgresql://..." }
    }
  }
}
```

---

## 3. 非功能性需求

### 3.1 性能

- CLI 启动时间 < 500ms（冷启动）
- 工具调用延迟（不含 LLM 响应）< 100ms
- 支持流式输出（Streaming），AI 响应逐字显示，不等待完整结果

### 3.2 安全性

- API Key 存储在系统 Keychain 或加密配置文件，不明文写入磁盘
- 文件修改前生成 `.bak` 备份（可配置）
- 命令执行采用最小权限原则，默认沙箱化（禁止访问 `~/.ssh`、`/etc` 等敏感路径）
- 所有网络请求通过 HTTPS，支持代理配置

### 3.3 可观测性

- 本地日志文件记录所有工具调用及其输入/输出
- 支持 `--verbose` / `--debug` 标志输出详细调试信息
- Session 级 Token 用量统计

### 3.4 跨平台兼容性

| 平台 | 支持状态 |
|---|---|
| macOS（Apple Silicon / x86_64） | ✅ 完全支持 |
| Linux（x86_64 / ARM64） | ✅ 完全支持 |
| Windows（WSL2） | ✅ 支持 |
| Windows（原生） | 🔶 部分支持（v1.1 目标） |

---

## 4. 技术架构

### 4.1 技术栈建议

```
语言：        Go 1.22+（高性能、单二进制分发、跨平台）
终端 UI：     bubbletea + lipgloss（TUI 框架）
配置管理：    viper
HTTP 客户端：  resty
日志：        zap
测试：        testify
```

### 4.2 核心模块划分

```
aicoder/
├── cmd/                  # CLI 入口，命令定义（cobra）
├── internal/
│   ├── agent/            # Agent 循环，工具调用编排
│   ├── llm/              # LLM 提供商适配层
│   ├── tools/            # 内置工具实现
│   ├── mcp/              # MCP 客户端
│   ├── context/          # 项目上下文收集
│   ├── session/          # 会话管理、历史记录
│   ├── ui/               # 终端 UI 渲染
│   └── config/           # 配置读写
└── pkg/                  # 可供外部复用的公共库
```

### 4.3 Agent 执行循环

```
用户输入
   │
   ▼
构建 Messages（系统提示 + 历史 + 用户输入）
   │
   ▼
调用 LLM API（流式）
   │
   ├── 纯文本响应 ──→ 渲染输出，等待下一轮输入
   │
   └── 工具调用请求
          │
          ▼
       展示工具调用详情，请求用户确认
          │
          ├── 用户拒绝 ──→ 将拒绝结果返回 LLM，继续对话
          │
          └── 用户确认
                 │
                 ▼
              执行工具，捕获输出/错误
                 │
                 ▼
              将工具结果追加到 Messages，继续 Agent 循环
```

---

## 5. 用户体验设计

### 5.1 输出规范

- AI 响应以流式方式在终端渲染，使用 Markdown 格式化（代码块语法高亮）
- 工具调用以折叠块显示，默认收起，用户可展开查看详情
- 文件修改以彩色 diff 形式展示（绿色新增，红色删除）
- 错误信息以红色标注，并提供可能的修复建议

### 5.2 权限确认 UI 示例

```
┌─────────────────────────────────────────────────┐
│  🔧  工具调用请求                                │
│  工具：run_command                              │
│  命令：npm run test -- --coverage               │
│  风险：低                                       │
├─────────────────────────────────────────────────┤
│  [Y] 允许   [N] 拒绝   [A] 本次会话始终允许     │
└─────────────────────────────────────────────────┘
```

---

## 6. 配置系统

配置优先级（由高到低）：CLI 参数 > 环境变量 > 项目级配置（`.aicoder/config.json`）> 用户级配置（`~/.aicoder/config.json`）

**主要配置项：**

```json
{
  "model": "claude-sonnet-4-5",
  "maxTokens": 8192,
  "autoApprove": false,
  "autoApproveCommands": ["npm test", "go build", "git status"],
  "forbiddenCommands": ["rm -rf /", "sudo"],
  "backupOnWrite": true,
  "theme": "dark",
  "language": "zh-CN",
  "proxy": "http://127.0.0.1:7890"
}
```

---

## 7. 安装与分发

```bash
# Homebrew（macOS/Linux）
brew install aicoder

# npm（跨平台）
npm install -g @iminders/aicoder

# 直接下载二进制
curl -fsSL https://aicoder.dev/install.sh | sh

# Go 源码编译
go install github.com/iminders/aicoder@latest
```

---

## 8. 里程碑规划

### v1.0（MVP，目标工期：12 周）

- [x] 交互式对话模式
- [x] 文件系统工具（读/写/编辑/搜索）
- [x] 命令执行工具（含权限确认）
- [x] 流式输出 + Markdown 渲染
- [x] Anthropic / OpenAI 双提供商支持
- [x] `AICODER.md` 项目上下文
- [x] macOS / Linux 支持

### v1.1（目标工期：+6 周）

- [ ] MCP 客户端支持
- [ ] Ollama 本地模型支持
- [ ] `/undo` 撤销机制
- [ ] Windows 原生支持
- [ ] 插件系统（自定义工具）

### v2.0（目标工期：+8 周）

- [ ] 多 Agent 并行任务
- [ ] 可视化 Web Dashboard（会话历史、费用分析）
- [ ] 团队共享 `.AICODER.md` 模板市场
- [ ] IDE 插件（VS Code）集成本 CLI

---

## 9. 成功指标

| 指标 | v1.0 目标 |
|---|---|
| GitHub Star 数 | > 1,000 |
| 月活跃用户（MAU） | > 500 |
| 任务完成率（无需人工干预） | > 70% |
| 用户满意度（NPS） | > 40 |
| P50 启动时间 | < 300ms |

---

## 10. 待定事项（Open Questions）

1. **授权模型**：是否开源？采用何种 License（MIT / Apache 2.0）？
2. **遥测数据**：是否收集匿名使用数据以改进产品？如何做到透明可选？
3. **付费模式**：工具本身免费，还是提供托管的 LLM 代理服务？
4. **多语言支持**：CLI 界面国际化的优先级？
5. **企业版功能**：是否需要 SSO、审计日志、私有部署等企业级特性？

---

*文档持续更新中，如有问题请联系产品团队。*
