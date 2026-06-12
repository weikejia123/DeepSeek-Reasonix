# Desktop UI 面板常见问题

## Q1: 左侧项目工作区的内容删除后，系统中的真实对话历史文件会消失吗？

**不会立即消失，默认是移入回收站。**

左侧项目工作区（`ProjectTree`）显示的是 **项目 + 会话主题（Topic）的树形结构**。分为三类删除操作：

| 操作 | 实际行为 | 代码依据 |
|------|----------|----------|
| 右键 Topic → "Trash Topic" | 将会话的 `.jsonl`、`.meta`、`.ckpt` 等文件**移动**到 `reasonix/sessions/.trash/{key}/` 目录（`os.Rename`），并写入 `_trashed.json` 元数据。**文件仍存在磁盘上，可从回收站恢复。** | `desktop/app.go` → `tabs.go:2640` `TrashTopic()` → `sessions.go:122` `trashSessionArtifacts()` |
| 右键 Project → "Remove Project" | **只从 `desktop-projects.json` 中移除该项目记录**，绝不触碰文件系统中的任何项目文件。 | `desktop/app.go:1410` `RemoveWorkspace()` → `forgetWorkspace()` + `removeProject()` |
| 回收站中 → "Purge"（永久清除） | **这是唯一真正删除的操作**：调用 `os.RemoveAll(itemDir)` 从文件系统彻底删除。 | `desktop/sessions.go:236` `purgeTrashedSessionFile()` |

**结论：** 普通删除（Trash Topic / Move to Trash / Remove Project）**不会丢失文件**。只有回收站中的 "Purge" 才会永久删除。

---

## Q2: 右侧"本会话变更"的内容，相当于 git 变动吗？

**不完全等同。它更像是 AI 操作日志，而非 git diff。**

- **展示内容：** AI 在当前会话中**写入/修改过的文件列表**，每行显示文件路径和操作时间。
- **数据来源：** 来自 `ctrl.Checkpoints()`（`desktop/tabs.go:2976-2988`），即 AI 调用 `write_file`、`edit_file` 等工具时记录的文件路径。
- **与 git 的关系：** 核心变更列表来自 **AI 操作日志**，不是 `git diff` 结果。但每个条目可以附带 `GitStatus` 字段（如 `"modified"`），这个字段可能通过对比 git 工作树得出。

**结论：** 它是这段对话中 AI 写过哪些文件的摘要，不是 git diff。它告诉你"AI 改动了哪些文件"，但不展示具体改了什么内容。

| 对比维度 | 本会话变更 | git diff |
|----------|-----------|----------|
| 数据来源 | AI 工具调用记录 | git 版本对比 |
| 展示内容 | 文件路径列表 | 具体增删行 |
| 是否支持回退 | 否（仅列出） | 是 |

---

## Q3: 右侧"引用的文件"（依赖文件）是什么意思？

**"依赖文件"是"引用的文件（referenced files）"的别名。**

- **展示内容：** AI 在当前会话中**读取过的文件列表**，每行显示文件路径、读取时机（回合编号）和读取范围。
- **数据来源：** `telemetry.ReadFiles`（`desktop/tabs.go:2964`），即 AI 调用 `read_file`、`grep` 等工具时触发的文件读取记录。
- **"依赖文件"名称由来：** 在 `WorkspacePanel` 文件树视图中，当列表被限定为"引用的文件"时，搜索框占位符显示 `"Filter dependency files…"`（`WorkspacePanel.tsx:680`，i18n key `workspace.filterReferencedFiles` 在 `en.ts:146` 中定义为 `"Filter dependency files…"`）。

**结论：** "dependency files" 就是"引用的文件"（referenced files）的 UI 别名，比喻这些文件是当前会话所"依赖"的输入材料——AI 读了这些文件才有上下文回答问题。

---

### 右侧两个面板对比

| 面板 | UI 标题 | 数据含义 | 数据源头 |
|------|---------|----------|----------|
| **Session changes** | "本会话变更" | AI **写入过**的文件 | `ctrl.Checkpoints()`（AI 工具调用记录） |
| **Referenced files** | "引用的文件" / "依赖文件" | AI **读取过**的文件 | `telemetry.ReadFiles`（文件读取记录） |

两个面板均位于右侧 dock 的 **Overview（概览）** 标签页内。

---

## Q4: Goal（目标模式）的具体用途和使用方法？

### 用途

Goal 是 Reasonix 的 **"自主目标推进"模式**。设定一个目标后，AI 会自动连续工作——自己决策下一步做什么、调用工具执行、检查结果——直到目标完成、被阻塞（需要用户决策）或用户手动停止。适用于**一次性交代一个完整任务**，让 AI 自主干完，不需要每轮都输入指令。

### 如何开启和使用 Goal

**方法一：通过意图菜单（推荐）**

1. 在输入框左下角的意图菜单（intent menu），选择 **"目标模式"**（中文 UI 显示为"目标模式"，英文为 "Goal"）
2. 输入框中会出现 placeholder "请输入目标…"（英文 "Enter a goal…"）
3. 输入你的目标描述（如 "重构 internal/boot 包"），然后点击发送
4. AI 会自动开始推进，实时显示每步的思考与工具调用
5. AI 会在每轮末尾附加状态标记：`[goal:continue]`（继续）/ `[goal:complete]`（完成）/ `[goal:blocked:原因]`（阻塞）

**代码依据：** 前端 `Composer.tsx:L819` `goalModeOn = collaborationMode === "goal"`；`App.tsx:L1215-1226` `applyGoal()` 更新 goal 并同步到 controller。

**方法二：直接在聊天框输入 `/goal` 命令**

| 命令 | 作用 |
|------|------|
| `/goal 重构 internal/boot 包` | 设置目标并立即启动自主推进 |
| `/goal`（不带文本） | 查看当前目标是什么 |
| `/goal clear` 或 `/goal stop` 或 `/goal done` | 清除/停止当前目标 |

**代码依据：** `internal/control/input.go:L178-205` `ParseGoalCommand()` 解析 `/goal` 命令；`internal/control/controller.go:L896-922` `applyGoalCommand()` 处理命令并启动 goal loop。

### 底层工作原理

```
用户输入 goal → Go后端 SetGoalForTab()
  → controller.SetGoal(goal)  // 存入 goal 字符串，状态设为 running
  → controller.runGoalLoopWithRawDisplay()
    → 每次 Compose() 在用户消息前注入:

      <active-goal>
      你的目标描述

      Goal mode: pursue this goal autonomously...
      </active-goal>

    → AI 每轮回复末尾必须带状态标记
    → advanceGoalAfterTurn() 解析标记:
      [goal:complete] → 停止，完成任务
      [goal:continue] → 自动进入下一轮（最多50轮）
      [goal:blocked:xxx] → 相同原因≥3次则停止

```

**关键代码：**
- `internal/control/input.go:L134-146` `activeGoalBlock()` — 生成注入 AI 的 goal XML 块
- `internal/control/controller.go:L206-207` — `maxGoalAutoTurns = 50`（最多自动 50 轮）
- `internal/control/controller.go:L609-656` `advanceGoalAfterTurn()` — 解析 AI 回复的状态标记并决策是否继续

### 示例

**示例 1：代码重构**
```
目标：重构 internal/boot 包，将 Config 初始化逻辑抽离到独立的 internal/config 包中，保持向后兼容
```
→ AI 会自动：分析现有代码 → 创建新包 → 移动代码 → 更新引用 → 运行测试 → 输出 `[goal:complete]`

**示例 2：修复问题**
```
目标：修复 desktop 会话列表偶尔不刷新的问题
```
→ AI 会自动：排查代码 → 定位原因 → 修复 → 验证 → 输出 `[goal:complete]`

**示例 3：多步骤任务**
```
目标：为登录功能添加单元测试，至少要覆盖：成功登录、密码错误、用户不存在三种场景
```
→ AI 会自动：阅读现有代码 → 编写测试文件 → 运行测试 → 若失败则自动修复 → `[goal:complete]`

### 注意事项

1. **Goal 是"目标"而非"指令"**：描述"做什么"和"做到什么程度"，不要描述"怎么做"。AI 自主决策实现路径。
2. **最大 50 轮自动推进**：超过 50 轮会自动标记为 blocked 并停止（`maxGoalAutoTurns = 50`，`controller.go:L206`）。
3. **阻塞检测（3 次规则）**：如果 AI 连续 3 次因**相同原因**报告阻塞，会自动停止（`controller.go:L631-638`）。
4. **Goal 与 Plan 互斥**：设置 goal 时会自动关闭 plan mode（`internal/control/controller.go:L904` `c.SetPlanMode(false)`）。
5. **Goal 通过"骑在 turn 上"注入**：goal 内容被拼接到每次用户消息之前（`Compose()`），**不修改 system prompt**，因此不破坏 DeepSeek 的 prefix cache（`controller.go:L1290-1292` 注释明确说明）。
6. **Goal 只作用于 Controller 层**：goal 逻辑完全由 `internal/control` 管理，不触及 `internal/agent` 执行器（agent 只知道执行当前工具调用，不知道自己处于 goal 模式）。
7. **Goal 与 Loop 不同**：`/loop` 是定时重复同一 prompt，goal 是模型驱动的自主决策——AI 每轮自己决定下一步做什么（参见 `my-docs/02-Loop模式深度分析.md`）。

### 前端 UI 中文文案

| Key | 中文 |
|-----|------|
| `composer.goalMode` | "目标模式" |
| `composer.goalModeDesc` | "先输入目标，再点目标启动。" |
| `composer.goalModeActiveDesc` | "会自动推进，直到完成、阻塞或停止。" |
| `composer.goalInputPlaceholder` | "请输入目标…" |

（代码位置：`desktop/frontend/src/locales/zh.ts:L313-321`）

---

## Q5: 看到 "explore effort max 90012 ms" 是自动切换了模型吗？

**不是。"effort max" 是推理深度参数，不是模型名。**

| 字段 | 含义 |
|------|------|
| `explore` | 一个**内置 subagent skill**，在独立子代理中执行只读代码调查 |
| `effort max` | 推理深度拉到最高（对应 DeepSeek API 的 `reasoning_effort = "max"`），让模型花更多 tokens 做深度推理 |
| `90012 ms` | 该子任务的总耗时（约 90 秒） |

**模型选择逻辑**（`internal/boot/boot.go:1029-1044` `subagentModelRef()`）：

1. `[agent].subagent_models["explore"]` → 若有则用
2. Skill 文件自身的 `model:` frontmatter → 若有则用
3. `[agent].subagent_model` → 全局 subagent 默认
4. **以上都为空 → 使用主会话的模型**（用户设的 `deepseek-v4-flash`）

如果你没有配过 subagent 模型，explore 就是 **flash + max effort**——模型还是 flash，只是推理深度拉到了 max。

**effort 配置优先级**类似（`internal/boot/boot.go:1046-1061` `subagentEffortRef()`）：
- `[agent].subagent_efforts["explore"]` → Skill frontmatter `effort:` → `[agent].subagent_effort`

**代码位置：**
- explore skill 定义：`internal/skill/builtins.go:206-213`
- 顶层工具暴露：`internal/skill/tools.go:274-276`
- 模型/effort 解析：`internal/boot/boot.go:1029-1061`
- Effort 能力定义：`internal/config/effort.go:L32-34`（deepseek-v4-flash 支持 high\|max）

---

## Q6: 独立规划模型设为 pro，默认模型设为 flash，Goal 中就是 pro 规划 + flash 执行？

**是的，完全正确。** 这就是双模型协作（Coordinator）架构的工作方式。

### 配置后的流程

```
Goal 启动 → 每轮任务输入

  ┌─ 规划阶段 ─────────────────────────────┐
  │  模型：deepseek-v4-pro（独立规划模型）     │
  │  工具：仅只读（read_file, grep, glob 等）  │
  │  产出：一份简洁的执行计划                   │
  └──────────────────────────────────────────┘
                    ↓
  ┌─ 执行阶段 ─────────────────────────────┐
  │  模型：deepseek-v4-flash（默认主模型）     │
  │  工具：全部（write_file, bash 等）         │
  │  执行：按计划干活，可灵活调整               │
  └──────────────────────────────────────────┘
```

### 关键代码

**1. Coordinator 的创建**（`internal/boot/boot.go:848-871`）：
```go
if pm := cfg.Agent.PlannerModel; pm != "" && !tokenEconomy {
    pe, ok := cfg.ResolveModel(pm)
    // 创建规划器 provider（pro）
    plannerProv, _ := NewProviderWithProxy(pe, proxySpec)
    // 规划器用独立 session + 独立系统提示
    plannerSess := agent.NewSession(agent.PlannerPromptWithContext(mem.Block()))
    // 规划器只能用只读工具
    plannerTools := agent.PlannerToolRegistry(reg)
    // 将规划器和执行器包在一起
    runner = agent.NewCoordinator(plannerProv, plannerSess, pe.Price,
        plannerTools, agent.Options{...}, executor, ...)
    label = entry.Model + " + planner " + pe.Model  // 状态栏显示 "flash + planner pro"
}
```

**2. Coordinator 的运行逻辑**（`internal/agent/coordinator.go:91-104`）：
```go
func (c *Coordinator) Run(ctx context.Context, input string) error {
    // 阶段1：规划器做规划（只读）
    c.sink.Emit(event.Event{Kind: event.Phase, Text: c.planner.Name() + " · planning"})
    plan, err := c.plan(ctx, input)
    // 阶段2：执行器按计划执行
    c.sink.Emit(event.Event{Kind: event.Phase, Text: c.executor.prov.Name() + " · executing"})
    return c.executor.Run(ctx, formatHandoff(input, plan))
}
```

**3. 规划器的系统提示**（`coordinator.go:21-29`）：
> "You are the planner in a two-model coding agent. ... Do not write full implementations or attempt side effects. ... Output executor-ready instructions: what to do, which files or commands are relevant, expected blockers, and key decisions."

**4. 规划器可用的工具**（`internal/agent/task.go:378-385`）：
只读工具集中**排除**了以下元工具：
```go
var plannerNonResearchTools = []string{
    "ask", "bash_output", "complete_step",
    "slash_command", "todo_write", "wait",
}
```
其余的只读工具（`read_file`, `grep`, `glob`, `lsp_*`, `codegraph_*` 等）都可用。

### 适用场景

| 场景 | 效果 |
|------|------|
| **Goal 模式** | ✅ 走双模型：pro 规划 → flash 执行 |
| **普通对话（非 Goal）** | ✅ 也走双模型：每轮先 pro 规划再 flash 执行 |
| **简单问题（问候/问答）** | ⚡ 跳过规划器：`shouldPlan()` 判断为低风险则直走执行器（`coordinator.go:93-96`） |
| **未设 planner_model** | ❌ 单模型：flash 自己规划+执行 |

### 一句话总结

> **是的。pro 只做只读调研和出方案，flash 负责动手执行。** 两个模型有各自独立的 session（prefix cache 互不干扰），每次任务先调 pro 规划，再把 plan 拼到输入中交给 flash 执行。
