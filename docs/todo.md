# 开发计划（TODO）

**产品名称：** aicoder  
**总工期：** v1.0 = 12 周 | v1.1 = +6 周 | v2.0 = +8 周  
**创建日期：** 2026-03-13  
**关联文档：** [prd.md](./prd.md) · [arch.md](./arch.md)

> **状态说明：** `[ ]` 待开始 · `[~]` 进行中 · `[x]` 已完成 · `[-]` 已跳过/推迟

---

## Phase 0：项目初始化（第 1 周）

### 环境搭建

- [ ] 初始化 Go 模块：`go mod init github.com/yourorg/aicoder`
- [ ] 配置 `.golangci.yml` 代码检查规则
- [ ] 配置 `Makefile`（build / test / lint / release 目标）
- [ ] 搭建 GitHub Actions CI 流水线（lint + test + 多平台 build 矩阵）
- [ ] 配置 GoReleaser（`goreleaser.yml`）：darwin arm64/amd64、linux arm64/amd64
- [ ] 创建 `docs/` 目录，放入 prd.md / arch.md / todo.md

### 脚手架

- [ ] 集成 cobra，创建根命令（`cmd/root.go`）
- [ ] 添加全局 flags：`--model`, `--verbose`, `--debug`, `--dangerously-skip-permissions`
- [ ] 集成 viper，打通 cobra flags > 环境变量 > 配置文件优先级链
- [ ] 集成 zap logger，封装 `pkg/logger/`，`--debug` 切换日志级别
- [ ] 实现 `cmd/version.go`，输出版本号 / 构建时间 / Go 版本

**验收标准：** `aicoder --help` 和 `aicoder version` 可正常运行，CI 绿色通过。

---

## Phase 1：配置与安全基础（第 2 周）

### 配置系统（`internal/config/`）

- [ ] 定义 `Config` 结构体（model / maxTokens / autoApprove / autoApproveCommands / forbiddenCommands / backupOnWrite / theme / language / proxy 等）
- [ ] 实现 `loader.go`：四层配置合并（CLI > 环境变量 > 项目级 > 用户级）
- [ ] 实现 `validator.go`：必填项检查、模型名合法性校验、代理 URL 格式校验
- [ ] 实现 `keychain.go`：
  - [ ] macOS：`go-keyring` 封装
  - [ ] Linux：`libsecret` 或加密文件（AES-256-GCM，密钥由机器 ID 派生）
  - [ ] 环境变量兜底（`ANTHROPIC_API_KEY` / `OPENAI_API_KEY` 等）
- [ ] 编写 config 模块单元测试（覆盖率 > 80%）

### 权限沙箱（`internal/permission/`）

- [ ] 实现 `sandbox.go`：硬编码禁止路径列表，路径前缀匹配检查
- [ ] 实现 `risk_classifier.go`：命令字符串 → RiskLevel（Low/Medium/High/Critical）映射
- [ ] 实现 `policy.go`：ALLOW / DENY / ASK 三态决策逻辑
- [ ] 实现 `allowlist.go`：会话级白名单的增删查
- [ ] 编写 permission 模块单元测试

**验收标准：** 配置四层合并逻辑测试通过，敏感路径访问被正确拦截。

---

## Phase 2：LLM 适配层（第 3 周）

### 接口定义（`internal/llm/`）

- [ ] 定义 `Provider` 接口（`Chat` / `Models` / `Name`）
- [ ] 定义 `Message` / `StreamChunk` / `ChatRequest` / `ToolDefinition` 等公共类型
- [ ] 实现 `stream.go`：流式 channel 消费工具函数，处理 done/error 信号
- [ ] 实现 `factory.go`：按 config.Model 前缀路由到对应 Provider

### Anthropic Provider（`internal/llm/anthropic/`）

- [ ] 集成 `anthropic-sdk-go`
- [ ] 实现流式 Chat，正确处理 `content_block_delta` / `tool_use` 事件
- [ ] 适配工具定义格式（JSON Schema → Anthropic tool_schema）
- [ ] 处理 API 错误（429 限速退避、5xx 重试、认证失败友好提示）
- [ ] 编写 mock 测试（离线，不消耗 API 额度）

### OpenAI Provider（`internal/llm/openai/`）

- [ ] 集成 `openai-go`
- [ ] 实现流式 Chat + function calling 格式适配
- [ ] 兼容任意 OpenAI API 格式端点（自定义 baseURL）
- [ ] 编写 mock 测试

### Google Provider（`internal/llm/google/`）

- [ ] 实现 Gemini REST API 调用 + 流式响应
- [ ] 适配 Google 工具定义格式
- [ ] 编写 mock 测试

**验收标准：** 三个 Provider 均通过 mock 测试；真实 API 集成测试在 CI（secrets 注入）中通过。

---

## Phase 3：工具执行引擎（第 4 周）

### 工具基础设施（`internal/tools/`）

- [ ] 定义 `Tool` 接口和 `ToolResult` 类型
- [ ] 实现 `registry.go`：工具注册、按名查找、列出所有工具（含 JSON Schema 导出）
- [ ] 实现 `risk.go`：工具级和命令级风险自动分级

### 文件系统工具（`internal/tools/fs/`）

- [ ] `read_file.go`：支持全文读取和行范围读取（`startLine` / `endLine` 参数）
- [ ] `write_file.go`：写入前触发 `snapshot.Save()`，支持自动创建父目录
- [ ] `edit_file.go`：diff/patch 模式精确修改（基于 `go-diff`），失败时自动回滚
- [ ] `list_dir.go`：递归目录结构，支持深度限制和 `.gitignore` 规则过滤
- [ ] `search_files.go`：正则全文搜索，返回文件名 + 行号 + 上下文行
- [ ] `delete_file.go`：删除前强制保存快照，触发 Critical 级别确认

### 命令执行工具（`internal/tools/exec/`）

- [ ] `run_command.go`：`exec.CommandContext`，捕获 stdout/stderr，支持超时配置
- [ ] `run_background.go`：启动后台进程，返回 PID，会话结束时自动清理
- [ ] 集成 sandbox 路径检查 + risk_classifier 双重校验

### 搜索工具（`internal/tools/search/`）

- [ ] `grep_search.go`：调用系统 grep 或纯 Go 实现，支持正则和大小写选项
- [ ] `ast_search.go`：基于 `go/ast`（Go）+ `tree-sitter`（多语言）语义搜索，v1.0 先支持 Go/Python/JS
- [ ] `web_search.go`：可选工具，通过配置启用，接入搜索 API

### 工具集成

- [ ] 在程序启动时统一注册所有内置工具到 Registry
- [ ] 编写全部工具的单元测试（含沙箱拦截、快照回滚、风险分级验证）

**验收标准：** 所有工具可被 Agent 正确调用；沙箱拦截、风险确认流程测试全部通过。

---

## Phase 4：Agent 核心循环（第 5 周）

### 会话管理（`internal/session/`）

- [ ] `session.go`：Session 结构体，生命周期管理（创建/持久化/加载）
- [ ] `history.go`：Messages 内存存储 + JSONL 持久化（`~/.aicoder/sessions/`）
- [ ] `cost.go`：累计 input/output tokens，按模型单价估算费用
- [ ] `snapshot.go`：文件快照栈（push/pop），支持多步 undo

### Agent 主循环（`internal/agent/`）

- [ ] `message.go`：构建完整 Messages（系统提示 + 项目上下文 + 历史 + 用户输入）
- [ ] `loop.go`：实现 `executeTools()`，依次执行工具调用列表，汇总结果
- [ ] `agent.go`：实现完整 `Run()` 循环（参考 arch.md § 3.3 伪代码）
- [ ] `interrupt.go`：`context.WithCancel` 绑定，处理 Ctrl+C 优雅退出
- [ ] 编写 Agent 集成测试（mock Provider + mock Tools）

### 项目上下文收集（`internal/context/`）

- [ ] `git.go`：采集 `git status` / `git diff HEAD` / `git log --oneline -5`
- [ ] `project.go`：检测语言和依赖文件（`go.mod`, `package.json`, `pyproject.toml` 等）
- [ ] `aicoder_md.go`：向上遍历目录树查找并加载 `AICODER.md`
- [ ] `summarizer.go`：生成目录结构摘要（深度 ≤ 3，过滤 `node_modules` / `.git`）
- [ ] `collector.go`：组合以上模块，输出格式化的系统提示片段

**验收标准：** 接入真实 LLM 后，Agent 可完成「读文件 → 修改 → 运行测试」完整工具调用链。

---

## Phase 5：终端 UI（第 6-7 周）

### TUI 基础组件（`internal/ui/`）

- [ ] `theme.go`：定义 dark / light 颜色 token（lipgloss CSS 变量）
- [ ] `spinner.go`：LLM 等待动画（多种样式可配置）
- [ ] `markdown.go`：集成 glamour，代码块语法高亮（256 色终端）
- [ ] `diff_view.go`：彩色 diff 渲染（绿色新增 / 红色删除）

### TUI 主模型（`internal/ui/app.go`）

- [ ] 定义 `Model` 结构体（inputBox / outputView / confirmModal / statusBar）
- [ ] 实现 `Update(msg tea.Msg) (tea.Model, tea.Cmd)` 事件分发
- [ ] 实现 `View() string` 布局渲染（响应式终端宽度适配）
- [ ] 通过 `tea.Cmd` 将 LLM StreamChunk 异步注入 Model（非阻塞渲染）

### 输入组件（`internal/ui/input.go`）

- [ ] 多行文本输入框
- [ ] `↑/↓` 方向键浏览输入历史（环形缓冲区，最多 100 条）
- [ ] Tab 键补全斜杠命令候选列表

### 输出组件（`internal/ui/output.go`）

- [ ] 流式文本追加，触发局部重渲染
- [ ] 工具调用折叠块（默认收起，Enter 展开）
- [ ] 输出区域 `PgUp/PgDn` 滚动

### 确认弹窗（`internal/ui/confirm.go`）

- [ ] Modal 组件：展示工具名 / 参数预览 / 风险等级
- [ ] 三个选项：Y 允许 / N 拒绝 / A 始终允许（快捷键触发）
- [ ] Critical 风险时红色警告边框

### 状态栏

- [ ] 显示：当前模型名 / 累计 Token 数 / 当前工作目录
- [ ] LLM 响应中时实时显示流速（tokens/s）

### 管道/单次模式

- [ ] 检测 `!isTerminal(os.Stdout)` 时退化为纯文本输出（无颜色、无 TUI）
- [ ] `--dangerously-skip-permissions` 下跳过所有确认提示

**验收标准：** 流式渲染流畅（目标 60fps），确认弹窗 UX 符合 PRD § 5.2 设计，管道模式可正常工作。

---

## Phase 6：斜杠命令（第 8 周）

### 路由器（`internal/slash/router.go`）

- [ ] 命令注册表（`map[name]Command`）
- [ ] 解析 `/` 前缀输入，路由到对应命令
- [ ] 向 UI inputBox 提供 Tab 补全候选列表

### 各命令实现（`internal/slash/commands/`）

- [ ] `/help`：列出所有可用命令及说明
- [ ] `/clear`：清空 session.Messages，保留系统提示
- [ ] `/history`：分页展示对话历史（含时间戳）
- [ ] `/undo`：调用 `snapshot.Pop()`，恢复文件，截断 Messages
- [ ] `/diff`：展示本次会话所有文件变更的 unified diff
- [ ] `/commit`：执行 `git add -A && git commit -m "<AI 生成 message>"`
- [ ] `/cost`：展示当前 Session Token 消耗和费用估算
- [ ] `/model`：列出可用模型，支持热切换（不重启 Session）
- [ ] `/config`：展示配置项，支持 `set key value` 修改并持久化
- [ ] `/init`：在当前目录生成 `.aicoder/` 和 `AICODER.md` 模板
- [ ] `/exit`：优雅退出，持久化 Session 历史

**验收标准：** 所有斜杠命令可正常触发，Tab 补全可用。

---

## Phase 7：安装与分发（第 9 周）

- [ ] 完善 GoReleaser 配置，生成 SHA256 checksums
- [ ] 设置 GitHub Release 自动触发（tag push）
- [ ] 编写 `install.sh` 安装脚本（检测平台，下载对应二进制）
- [ ] 发布 Homebrew Tap：`homebrew-tap` 仓库，自动更新 formula
- [ ] 发布 npm 包（`@yourorg/aicoder`）：postinstall 脚本下载对应平台二进制
- [ ] 完善 `README.md`：安装说明 / 快速上手 / 配置参考 / GIF 演示

**验收标准：** 三种安装方式（brew / npm / install.sh）均可成功安装并运行。

---

## Phase 8：测试与质量（第 10 周）

### 测试补全

- [ ] 检查各模块覆盖率，补齐缺口至 > 80%
- [ ] 编写 Agent 端到端集成测试场景：
  - [ ] 场景 A：读文件 → 修改 → 运行测试
  - [ ] 场景 B：用户拒绝工具调用，AI 改变策略继续完成任务
  - [ ] 场景 C：文件写入失败，自动回滚并提示
  - [ ] 场景 D：管道模式单次任务
- [ ] 性能基准测试：启动时间 < 500ms，工具执行延迟 < 100ms

### 安全审查

- [ ] 审查所有工具的沙箱路径检查完备性
- [ ] 确认 API Key 不会出现在任何日志文件中
- [ ] 验证 `--dangerously-skip-permissions` 无法通过环境变量意外激活

### 文档

- [ ] `docs/config-reference.md`：全量配置项说明
- [ ] `docs/tools-reference.md`：工具参数和返回格式
- [ ] `CONTRIBUTING.md`：开发环境搭建 / PR 规范 / 发布流程

**验收标准：** CI 全绿，覆盖率达标，安全审查无高危问题。

---

## Phase 9：Beta 测试与正式发布（第 11-12 周）

### Beta 发布

- [ ] 发布 `v1.0.0-beta.1` 到 GitHub Releases
- [ ] 招募 10-20 名 Beta 测试者（内部 + 目标用户社区）
- [ ] 建立反馈收集渠道（GitHub Issues 模板 / Discord）

### 问题修复

- [ ] 整理 Beta 反馈，按 P0/P1/P2 优先级分类
- [ ] 修复所有 P0 问题（崩溃、数据丢失）
- [ ] 修复主要 P1 问题（核心功能不可用）
- [ ] P2 问题记录至 v1.1 Backlog

### 正式发布

- [ ] 更新 `CHANGELOG.md`
- [ ] 打 `v1.0.0` tag，触发 GoReleaser 正式发布
- [ ] 更新 Homebrew formula 和 npm 包版本
- [ ] 发布 GitHub Release Notes

**验收标准：** 无 P0 问题，核心路径（交互式对话 + 工具调用 + 流式渲染）体验流畅。

---

## v1.1 Backlog（+6 周，目标 2026 Q3）

### MCP 客户端（`internal/mcp/`）

- [ ] 实现 stdio transport（fork 子进程，双向通信）
- [ ] 实现 SSE transport（HTTP 长连接）
- [ ] 实现 `initialize` + `tools/list` 握手协议
- [ ] 实现 `tool_bridge.go`：MCP 工具 → `Tool` 接口适配
- [ ] 配置文件支持 `mcpServers` 字段
- [ ] 编写 mock MCP Server 集成测试

### Ollama 本地模型支持

- [ ] 实现 `internal/llm/ollama/client.go`（`/api/chat` 流式接口）
- [ ] 工具调用兼容性适配（针对支持 function calling 的模型）

### 多步 `/undo`

- [ ] 支持 `/undo 3` 语法，回退最近 N 次文件操作
- [ ] TUI 中展示可撤销操作历史列表

### Windows 原生支持

- [ ] 替换 POSIX 相关 API 为跨平台写法
- [ ] 测试 Windows Terminal / PowerShell TUI 渲染
- [ ] CI 添加 `windows/amd64` 构建矩阵
- [ ] Chocolatey / Scoop 包发布

### 插件系统

- [ ] 设计插件接口规范（子进程 + JSON-RPC 通信）
- [ ] 支持用户自定义工具插件
- [ ] 插件加载与版本管理

---

## v2.0 Backlog（+8 周，目标 2026 Q4）

- [ ] 多 Agent 并行任务（任务拆解 + 子 Agent 协同）
- [ ] Web Dashboard（会话历史可视化、Token 用量分析）
- [ ] `AICODER.md` 模板市场（团队共享最佳实践）
- [ ] VS Code 插件（内嵌本 CLI，提供侧边栏 UI）
- [ ] 企业版：SSO 集成、操作审计日志导出、私有部署

---

## 模块依赖关系

```
Phase 0（脚手架）
    │
    ▼
Phase 1（配置 + 安全基础）
    │
    ├──→ Phase 2（LLM 适配层）──┐
    │                           │
    └──→ Phase 3（工具引擎）────→ Phase 4（Agent 核心循环）
                                          │
                               ┌──────────┴──────────┐
                               ▼                     ▼
                          Phase 5（TUI）      Phase 6（斜杠命令）
                               │
                               ▼
                          Phase 7（分发）
                               │
                               ▼
                          Phase 8（测试与质量）
                               │
                               ▼
                          Phase 9（Beta → 正式发布）
```

---

*本文档为滚动更新的开发计划，每周同步进度：完成项标记 `[x]`，新发现任务追加至对应 Phase。*
