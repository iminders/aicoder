# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release features

## [1.0.0] - 2026-03-16

### Added
- Interactive REPL mode with AI agent loop
- One-shot execution mode
- Unix pipe mode support
- Built-in tools: file operations, shell execution, code search
- Permission confirmation system with risk classification
- Multi-LLM provider support (Anthropic Claude, OpenAI GPT)
- Project context awareness (AICODER.md, Git status, dependencies)
- 11 slash commands: /help, /clear, /history, /undo, /diff, /commit, /cost, /model, /config, /init, /exit
- Session management with snapshots and undo
- Token usage tracking and cost estimation
- Configuration system with multiple layers (CLI > env > project > user)
- Terminal UI with bubbletea framework
- Markdown rendering with syntax highlighting
- AST-based semantic code search
- Web search integration (optional)
- Risk assessment for commands and tools
- Directory tree summarization
- Comprehensive documentation

### Security
- Sandbox protection for sensitive paths
- Command blacklist for dangerous operations
- Permission confirmation for medium/high risk operations
- File snapshot system for rollback
- API key secure storage

### Documentation
- Complete README with installation and usage instructions
- Architecture documentation (arch.md)
- Product requirements document (prd.md)
- Development plan (todo.md)
- Tools reference guide
- Code contribution guide
- Development guide

[Unreleased]: https://github.com/iminders/aicoder/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/iminders/aicoder/releases/tag/v1.0.0
