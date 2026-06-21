# 三、Token Economy 模式

## 原理

在对话的初始阶段，减少注入系统提示词的 tools 数量，从而减少每次请求的 prefix token 数量。虽然这会降低模型的工具能力，但对于简单任务可以显著节约 token。

## 实现

**文件：** `internal/boot/token_profile.go:15-68`

启动时指定 `--token-mode economy`，效果：

| 配置项 | 正常模式 | Economy 模式 |
|--------|---------|-------------|
| 内置工具 | 全部（~30+） | 仅 15 个核心内置工具 |
| 核心工具 | bash, read_file, edit_file, glob, grep, ... | 同上（保留） |
| 非核心工具 | web_fetch, LSP, MCP, task, install_source, ... | **移除**，通过 `connect_tool_source` 按需激活 |
| 系统提示词 | 包含完整的技能索引（Skills index） | **不包含**技能索引 |

## 适用场景

- 简单问答、文件读取等不需要复杂工具链的任务
- 上下文窗口较小或对 token 成本敏感的模型
- 对话初期，工具支持尚未成为瓶颈时

## 代价

- 模型无法直接使用 MCP、web_fetch 等工具
- 需要通过 `/connect_tool_source` 命令手动激活所需工具
- 技能索引不可见，无法按名调用技能

## 代码位置

- 提示词注入：`internal/boot/token_profile.go:20` — 在 system prompt 中添加说明
- 工具过滤：`internal/boot/boot.go:137-138` — 根据 `TokenEconomy` 标志筛选内置工具
- 按需激活：通过 `connect_tool_source` 工具（由 `internal/tool/builtin/` 实现）

## 通用性

完全通用，不依赖特定模型。任何提供工具的 API 都可以使用此模式。
