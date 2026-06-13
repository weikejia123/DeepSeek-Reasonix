# Slash Commands — 指令系统完整参考

**能力类型**: 基础能力  
**代码位置**: `internal/control/controller.go:746-882`（submit 路由）、`internal/control/slash.go:337-406`（管理指令）、`internal/cli/chat_tui.go:3476-3612`（TUI 扩展）、`internal/command/command.go`（自定义命令）

## 架构概览

指令系统分三层：

```
用户输入 "/xxx ..."
        │
        ▼
controller.submit()  ── 第一层：核心指令（Go backend 处理）
  ├─ # 记忆快添      ── 输入处理阶段
  ├─ /goal          ── 命令解析
  ├─ !shell         ── shell 执行
  ├─ /compact       ── 会话管理
  ├─ /new /clear    ── 会话管理
  ├─ /mcp__         ── MCP 直接调用
  ├─ /tree /branch /switch /rewind  ── 分支/回退
  ├─ managementNotice ── 第二层：只读管理指令
  │   ├─ /model /provider
  │   ├─ /memory
  │   ├─ /skill /skills
  │   ├─ /hooks
  │   └─ /mcp
  ├─ CustomCommand  ── 第三层：用户自定义 .md 命令
  └─ RunSkill       ── 第四层：技能（/skill-name）

chat_tui (TUI)      ── TUI 专有指令（富交互 UI）
  ├─ /resume /rename /todo /verbose /sandbox
  ├─ /effort /auto-plan /paste-image
  ├─ /output-style /diff-fold /theme /language
  ├─ /loop /help /goal /remember /forget /quit
  └─ 同名的管理指令在 TUI 中有更丰富的 UI 实现
```

---

## 一、会话生命周期指令

### `/new`
- **作用**: 开启全新会话（fork 当前会话，创建独立分支）
- **参数**: 无
- **实现**: `controller.go:776-783` → `NewSession()`；TUI 额外重置上下文视图
- **场景**: 话题切换、实验性操作前备份、清理长对话

### `/clear`
- **作用**: 清空当前会话上下文（保留会话 ID，清空消息历史）
- **参数**: 无（TUI 有二次确认 `clearConfirm`）
- **实现**: `controller.go:784-791` → `ClearSession()`
- **场景**: 上下文溢出前手动清空、敏感信息残留清理

### `/compact [focus]`
- **作用**: 触发上下文压缩——将中间消息总结后只保留首尾
- **参数**: `[focus]` 可选，指导压缩器保留哪些内容
- **实现**: `controller.go:764-775` → `Compact(ctx, focus)`
- **场景**: 长对话接近 context window 时主动压缩；`/compact "keep the auth changes"`
- **注意**: 异步执行（`go func()`），完成时 emit CompactionStarted/Done 事件

---

## 二、分支与会话导航指令

### `/tree`
- **作用**: 显示会话分支树结构
- **实现**: `controller.go:826-828` → `BranchTreeText()`
- **场景**: 查看 fork 历史和分支关系

### `/branch [name]`
- **作用**: 创建命名分支（从当前 turn 分叉）
- **参数**: `name` 分支名；可选 `from=<turn>` 从指定 turn 分叉
- **实现**: `controller.go:829-842` → `Branch(name)` / `ForkNamed(turn, name)`
- **场景**: `/branch experiment` 在当前点创建实验分支；`/branch fix from=5` 从 turn 5 分叉

### `/switch <ref>`
- **作用**: 切换到指定分支或会话
- **参数**: `<ref>` 分支名或会话 ID
- **实现**: `controller.go:843-848` → `SwitchBranch(ref)`
- **场景**: 切回之前的分支继续工作

### `/rewind [turn] [scope]`
- **作用**: 回退到指定 turn，可选择回退范围
- **参数**: `turn` 目标 turn 编号；`scope` = `code`|`conversation`|`both`（默认 both）
- **实现**: `controller.go:849-859` → `Rewind(turn, scope)`
- **场景**: `/rewind 5 code` 只回退代码变更保留对话；`/rewind 3` 完全回退到 turn 3

---

## 三、自主执行指令

### `/goal [text]`
- **作用**: 设置/清除/查看自主执行目标。设置后模型在每个 turn 末尾输出 `[goal:continue|complete|blocked]` 标记，系统自动循环直到完成
- **参数**: 
  - `/goal <text>` — 设置新目标并启动
  - `/goal` — 查看当前目标状态
  - `/goal clear` — 清除目标
- **实现**: `controller.go:896-923` → `applyGoalCommand()`；循环 `continueGoal():590-607` + `advanceGoalAfterTurn():609-656`
- **关键行为**:
  - 最大 50 turns 自动停止（`maxGoalAutoTurns`）
  - 连续 3 次相同 `blocked` 原因 → 停止
  - 激活时跳过 auto-plan
- **场景**: "把整个项目的错误处理统一改为 structured errors" — 多步自主完成

### `/loop <interval> <prompt>`
- **作用**: 定时重复执行同一个 prompt（TUI 专有）
- **参数**: `interval` 间隔时间（如 `30s`、`5m`），`prompt` 重复执行的提示词
- **实现**: `chat_tui.go:3578-3580` → `runLoopCommand()`
- **场景**: `/loop 60s "检查服务是否存活"` — 定时健康检查
- **与 goal 的区别**: loop 是定时重复，goal 是模型驱动的自主推进

---

## 四、模型与提供商指令

### `/model [provider/model]`
- **作用**: 切换 AI 模型
- **参数**: `[provider/model]` 如 `deepseek/v4`；无参列出可用模型
- **实现**: `slash.go:343-344` → `modelListText()` / TUI `runModelSubcommand()`
- **场景**: `/model` 看有哪些模型；`/model deepseek/v4-flash` 切换

### `/provider [name]`
- **作用**: 切换 AI 提供商（及其默认模型）
- **参数**: `[name]` 提供商名（配置中的 `[providers.<name>]`）
- **实现**: `slash.go:345-350` → `providerSwitchText()`
- **场景**: `/provider deepseek` 切换到 DeepSeek 的默认模型

### `/effort [level]`
- **作用**: 设置推理深度（OpenAI o-series / 支持 reasoning_effort 的模型）
- **参数**: `[level]` = `auto`|`low`|`medium`|`high`|`xhigh`|`max`
- **实现**: TUI `chat_tui.go:3512-3513` → `runEffortCommand()`
- **场景**: `/effort high` 复杂重构时提高推理深度；`/effort low` 简单任务节省成本

---

## 五、扩展生态管理指令

### `/mcp [connect <name>]`
- **作用**: 管理 MCP（Model Context Protocol）服务器
- **参数**: `/mcp` 列出所有 MCP 服务器及状态；`/mcp connect <name>` 手动连接指定服务器
- **实现**: `slash.go:391-401` → `mcpListText()` / `ConnectConfiguredMCPServer()`
- **场景**: 查看 MCP 状态、手动触发懒加载服务器

### `/skill[s] [enable|disable <name>]`
- **作用**: 管理技能（Skills）
- **参数**: 
  - `/skills` 列出已加载技能
  - `/skills enable <name>` 启用
  - `/skills disable <name>` 禁用
- **实现**: `slash.go:353-369` → `skillListText()` / `SetSkillEnabled()`
- **场景**: 临时禁用不需要的技能减少 system prompt 体积

### `/hooks [trust|list]`
- **作用**: 管理 Hook 脚本
- **参数**: `/hooks` 列出 hooks；`/hooks trust` 信任当前项目 hooks
- **实现**: `slash.go:370-390` → `hookListText()` / `Trust()`
- **场景**: 新项目首次运行时 `/hooks trust` 确认安全

---

## 六、配置与 UI 指令（TUI）

### `/theme [name]`
- **作用**: 切换终端配色主题
- **参数**: `[name]` 主题名；无参列出可用
- **实现**: `chat_tui.go:3572-3574`
- **场景**: `/theme dark` `/theme light`

### `/language [lang]`
- **作用**: 切换 UI 语言
- **参数**: `[lang]` = `en`|`zh`|`zh-TW`
- **实现**: `chat_tui.go:3575-3577`
- **场景**: `/language zh` 切换到中文

### `/output-style [name]`
- **作用**: 切换输出渲染风格（工具调用展示格式）
- **参数**: `[name]` 风格名；无参列出
- **实现**: `chat_tui.go:3555-3562`
- **场景**: 调整工具调用的显示样式

### `/diff-fold`
- **作用**: 切换 diff 折叠模式（超过限制行数时折叠）
- **实现**: `chat_tui.go:3563-3571`
- **场景**: 长 diff 自动折叠，提高可读性

### `/verbose`
- **作用**: 切换详细推理显示（显示 thinking/reasoning 内容）
- **实现**: `chat_tui.go:3507-3508`
- **场景**: 想看到模型的思考过程时开启

### `/todo`
- **作用**: 清除当前钉住的任务列表
- **实现**: `chat_tui.go:3502-3506`
- **场景**: plan 阶段结束后清理残留的 task list

---

## 七、会话管理指令（TUI）

### `/resume`
- **作用**: 从历史会话列表中选择恢复（TUI 交互式选择器）
- **实现**: `chat_tui.go:3498-3499` → 打开 resume picker
- **场景**: 恢复之前中断的会话

### `/rename <name>`
- **作用**: 重命名当前会话
- **参数**: `<name>` 新名称
- **场景**: 给会话一个描述性名称便于日后识别

---

## 八、记忆管理指令

### `/memory`
- **作用**: 显示当前项目的记忆内容
- **实现**: `slash.go:351-352` → `memoryListText()`
- **场景**: 查看 agent 记住了哪些项目信息

### `/remember <note>`
- **作用**: 快速添加一条项目记忆
- **参数**: `<note>` 记忆内容
- **实现**: `chat_tui.go:3589-3597` → `QuickAdd(memory.ScopeProject, note)`；也支持 `# <note>` 快捷语法
- **场景**: `/remember "使用 pnpm 而非 npm"`

### `/forget <name>`
- **作用**: 删除一条记忆（TUI 专有）
- **参数**: `<name>` 记忆的名称/关键词
- **实现**: `chat_tui.go:3600-3601`
- **场景**: 清理过时或错误的记忆

---

## 九、Shell 执行

### `!<command>`
- **作用**: 直接执行 shell 命令
- **参数**: `<command>` 完整的 shell 命令
- **实现**: `controller.go:759-762` → `RunShell()`
- **场景**: `!git status` 快速检查状态；前缀 `!` 区别于普通对话

---

## 十、MCP 工具直接调用

### `/mcp__<server>__<tool> [args]`
- **作用**: 直接调用指定 MCP 服务器的工具
- **实现**: `controller.go:792-803` → `MCPPrompt()` → 展开为完整 prompt
- **场景**: `/mcp__github__search_repositories "reasonix"`

---

## 十一、自定义命令

### 定义方式

在 `.reasonix/commands/` 目录下放置 `.md` 文件：

```markdown
---
description: Review the current diff for bugs
argument-hint: [focus]
---
Review the following git diff $ARGUMENTS. Focus on:
- Bugs and correctness issues
- Missing error handling
- Security concerns
```

- 文件名即命令名：`review.md` → `/review`
- 子目录用 `:` 命名空间：`git/commit.md` → `/git:commit`
- 模板变量：`$ARGUMENTS`（所有参数）、`$1` `$2`…（位置参数）、`$$`（字面量 `$`）

### 优先级

自定义命令 > 同名 skill。两者都作为普通 turn 提交到模型。

---

## 十二、Skills 作为指令

所有已加载的 skill 可通过 `/<skill-name>` 直接调用。与 `run_skill` 工具等价：

- **Inline skills**: 模板展开后作为普通 turn 提交
- **Subagent skills**: 同 inline 方式提交，但 skill 内部以子代理运行

```go
// controller.go:872-877
if sent, ok := c.RunSkill(trimmed); ok {
    c.runGuarded(func(ctx context.Context) error {
        return c.runGoalLoopWithRawDisplay(ctx, sent, sent, display)
    })
    return
}
```

---

## 十三、辅助语法

### `# <note>` — 快捷记忆
输入以 `# ` 开头的消息自动转为记忆添加（不发送给模型）。

### `//text` — 双斜杠转义
以 `//` 开头的输入被视为普通消息（不解析为命令）。用于发送代码注释、file:// URL 等。

### 文件引用
- `/path/to/file:10` → 打开文件定位
- `/path/to/file` → 打开文件

---

## 优先级与路由一览

```
输入处理顺序:
  1. # <note>           → MemoryQuickAddNote
  2. /remember <note>   → RememberCommandNote
  3. /goal ...          → applyGoalCommand (启动自主循环)
  4. !<command>         → RunShell
  5. /compact           → Compact
  6. /new               → NewSession
  7. /clear             → ClearSession
  8. /mcp__...          → MCPPrompt
  9. //text             → 普通 turn（转义）
  10. /...              → 进入 slash 处理
      10a. /tree /branch /switch /rewind  → 分支操作
      10b. managementNotice              → /model /provider /memory /skills /hooks /mcp
      10c. CustomCommand                 → 用户 .md 命令
      10d. RunSkill                      → /<skill-name>
      10e. "unknown command"             → 错误提示
  11. 普通文本           → runRefTurn
```

---

## 代码索引

| 文件 | 行号 | 内容 |
|------|------|------|
| `internal/control/controller.go` | 746-882 | `submit()` 完整指令路由 |
| `internal/control/controller.go` | 896-923 | `applyGoalCommand()` |
| `internal/control/controller.go` | 590-656 | `continueGoal()` + `advanceGoalAfterTurn()` |
| `internal/control/slash.go` | 337-406 | `managementNotice()` 管理指令 |
| `internal/cli/chat_tui.go` | 3476-3612 | TUI 指令处理 |
| `internal/command/command.go` | 1-140 | 自定义命令加载与渲染 |
| `internal/i18n/messages_en.go` | 106 | 帮助文本（指令概览） |
