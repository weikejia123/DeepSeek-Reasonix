# Goal 模式完整分析

## 快速摘要

Goal 模式是 Reasonix 的自主多轮执行机制。用户设定目标后，系统在后台循环
推进直到完成、阻塞或超限。但在**双模型（planner + executor）架构**下，存在
一个固有矛盾：planner 只能读不能写，但 goal 块强制它"执行下一步"——导致
分析瘫痪。本文档完整剖析 Goal 模式的机制、风险、使用边界和反面案例。

| 场景 | 适合 Goal 模式？ |
|------|------------------|
| 实现明确功能（加接口/改页面/写测试） | ✅ 适合 |
| 「研究一下 / 分析一下 / 看看能不能」 | ❌ 不适合 |
| 单模型（无独立 planner） | ⚠️ 风险较低但仍需注意 |
| 双模型（planner_model 已配置） | ⚠️ 高风险，必须有明确目标 |

## 问题现象

在 goal 模式 + 双模型（planner + executor）架构下，planner 可能在单轮中发起 40+
次工具调用（`read_file` / `grep` / `glob` / `ls`），不断阅读更多文件，却**从未产出一行
新代码或任何执行指令**。它陷入"分析瘫痪"（analysis paralysis）——
以"追求全面"为名义过度研究，实际 5-8 次读取已足够产出计划。

## 根因分析

**核心矛盾：`<active-goal>` 块与 planner 系统提示的指令冲突。**

### 1. Goal 块的注入（每次 turn 都会注入）

`Controller.Compose()`（`internal/control/input.go:93-103`）在**每一次** turn
的 user 消息前注入 `<active-goal>` 块：

```go
// input.go:102-103
if strings.TrimSpace(goal) != "" && goalStatus == GoalStatusRunning {
    text = activeGoalBlock(goal) + "\n\n" + text
}
```

注入的内容由 `activeGoalBlock()`（`input.go:134-146`）生成，包含以下指令：

> Goal mode: pursue this goal autonomously. Keep working across turns until the
> goal is complete. Prefer sensible defaults over asking the user; use ask only
> when you are truly blocked on a user-owned decision. **Do not stop after
> describing a plan; execute the next useful step.** End every goal-mode assistant
> reply with exactly one status marker on its own line: [goal:continue],
> [goal:complete], or [goal:blocked:<short reason>].

同样，目标循环中的后续 turn 使用常量 `goalContinueTurn`（`controller.go:207`）：

> Continue pursuing the active goal. If it is complete, provide the concise final
> result and end with [goal:complete]. If it is truly blocked on a user-owned
> decision after trying sensible defaults, end with [goal:blocked:<short reason>].
> **Otherwise do the next useful work and end with [goal:continue].**

### 2. Planner 的系统提示

Planner 的系统提示（`internal/agent/coordinator.go:21-29` `DefaultPlannerPrompt`）
明确要求：

> You are the planner in a two-model coding agent. Given a task, produce a
> **concise, ordered plan** for the executor model to carry out. Use the read-only
> tools available to you when the task needs context from the workspace, user
> rules, or docs; **keep that research targeted and stop once you have
> enough evidence.** Do not write full implementations or attempt side effects.
> … Keep it **short and actionable.**

### 3. 指令冲突

| 来源 | 指令 |
|------|------|
| Planner 系统提示 | "stop once you have enough evidence" → 产出计划 |
| Goal 块（注入到 user 消息） | "do not stop after describing a plan; **execute** the next useful step" |
| Planner 能力 | **只读工具**（glob/grep/read_file/ls/web_fetch/lsp_*/memory/history），**没有** bash/edit_file/write_file |

Planner 收到矛盾的指令：系统提示让它产出简洁计划、goal 块让它自主执行。它无法执行（无写工具），
但又被告知"不要停在计划描述阶段"——于是它用自己能做的唯一方式"继续工作"：不断读更多文件。

### 4. 完整调用链

```
用户设置 goal
  → Controller.SetGoal()                              // controller.go:1293
  → applyGoalCommand()                                // controller.go:896-910
  → runGoalLoopWithRawDisplay()                       // controller.go:514-522
  → runTurnWithRawDisplay()                           // controller.go:524
  → Compose(input)  ← 注入 <active-goal> 块          // input.go:93
  → Coordinator.Run(ctx, composedInput)               // coordinator.go:91
  → c.plan(ctx, input)                                // coordinator.go:98
  → planner 收到: system="产出简洁计划" + user="<active-goal>…执行下一步…"
  → 冲突 → 分析瘫痪
```

**关键点：** `Compose()` 是无条件注入——它不区分"这个 turn 是否经过 planner"。
Planner 和 Executor 看到的是**同一份**被 goal 块污染的 user 消息。

### 5. 双模型架构的额外问题

当设置了独立规划模型（planner_model）时，`Coordinator.Run()`（`coordinator.go:91-104`）
先调 `plan()` 再调 `executor.Run()`。Planner 的步数由 `PlannerMaxSteps` 控制
（`boot.go:861`），但 planner 本身没有 `maxGoalAutoTurns` 那样的硬限制——
它受限于 `PlannerMaxSteps` 配置（默认值取决于配置）。

如果 `PlannerMaxSteps` 设得过大（如 50+），planner 可能在一个 turn 内消耗
大量 token 做无意义的轮询式阅读。

## 内置保护机制

系统有兜底保护，但不能防止初期的分析瘫痪：

| 机制 | 行为 | 代码位置 |
|------|------|----------|
| `maxGoalAutoTurns = 50` | 最多 50 个自动 turn 后强制 `GoalStatusBlocked` | `controller.go:206,645-648` |
| Blocked 检测 | 连续 3 次相同 `[goal:blocked:<reason>]` → 标记为 blocked | `controller.go:631-639` |
| Context 取消 | 用户取消（如前端 Esc）→ `stopGoal(GoalStatusStopped)` | `controller.go:596-598` |
| PlannerMaxSteps | 限制 planner 单轮的最大步数 | `boot.go:861` |

但这些保护只在**检测到明确的状态标记**或**超过次数限制**时触发。如果 planner
每次都返回 `[goal:continue]` 但在工具调用中消耗大量 token，系统会默认这是合法的
"继续推进"行为。

## 对用户/开发者的实用建议

### 设定 goal 时的注意事项

1. **设定具体、可执行的目标**，避免开放式的"研究/审查/了解"类任务。
   - ❌ "review the codebase and understand how authentication works"
   - ✅ "add OAuth2 login support to the API, following the pattern in internal/auth/basic.go"

2. **避免将"计划/研究"本身设为 goal**。Goal 模式假设目标是"完成某事"而非"了解某事"。
   如果确实需要全面代码审查，使用普通（normal）模式或 plan 模式，手动引导。

3. **如果发现 planner 开始循环读取**，前端手动取消（Esc），用更窄的目标重新设定
   goal，或切换到 normal 模式手动指导。

4. **设置合理的 `PlannerMaxSteps`**。如果使用双模型架构，将 planner 的最大步数
   限制在较小的值（如 10-15），防止 planner 在单轮中做过多工具调用。

5. **goal 模式最适合**：实现明确功能、修复已知 bug、编写测试——这类任务可以自然
   分解为"读 → 写 → 验证"的循环，planner 能快速定位上下文后产出计划。

### 潜在改进方向（未实现）

- 在 `Compose()` 或 `Coordinator.Run()` 中添加"此 turn 经过 planner"的信号，
  让 goal 块对 planner 仅注入目标描述、不注入"执行下一步"的指令。
- 为 planner 设定更严格的最大步数限制（当前 `PlannerMaxSteps` 配置可用）。

### 自己使用 goal 时的注意事项

如果你是**人类用户正在向 AI 下达 goal 指令**，需要注意：

1. **AI（planner）没有写工具**：当配置了独立规划模型时，planner 只有只读工具。
   它的任务本质是"调研并产出计划"，不是"执行"。
   如果 goal 指令模糊（如"研究一下这个模块"），planner 会一直读到 max steps 为止。

2. **区分"一次性 goal"和"迭代 goal"**：
   - 一次性 goal（如"修复 #123 号 bug"）：AI 应该快速读上下文 → 计划 → 执行
   - 迭代 goal（如"做三次性能优化"）：AI 可能在前几轮就完成最优解，后续轮次变差
   - 建议：**把大目标拆成多个小 goal 分步执行**，每个 goal 聚焦一件事

3. **注意 `[goal:continue]` 的隐形成本**：AI 每次标记 `[goal:continue]` 都会消耗
   一次 model API 调用。如果 planner 陷入分析瘫痪且每次都返回 `[goal:continue]`，
   它会在 50 轮内持续产生 API 费用而不产出代码。**发现这种情况应手动取消。**

## 相关源码索引

- `internal/control/input.go:93-146` — `Compose()` 注入 goal 块 + `activeGoalBlock()` 生成内容
- `internal/control/controller.go:206-208` — `maxGoalAutoTurns` + `goalContinueTurn` 常量
- `internal/control/controller.go:514-522` — `runGoalLoopWithRawDisplay()` 入口
- `internal/control/controller.go:524-531` — `runTurnWithRawDisplay()` 调用 `Compose()`
- `internal/control/controller.go:590-656` — `continueGoal()` + `advanceGoalAfterTurn()` 循环与控制
- `internal/control/controller.go:658-679` — `parseGoalStatusMarker()` 解析 `[goal:*]` 标记
- `internal/control/controller.go:1293-1314` — `SetGoal()` / `ClearGoal()`
- `internal/agent/coordinator.go:21-29` — `DefaultPlannerPrompt`（planner 系统提示）
- `internal/agent/coordinator.go:91-104` — `Coordinator.Run()`（plan → execute 流程）
- `internal/agent/task.go:378-418` — `PlannerToolRegistry()`（planner 只读工具集）
- `my-docs/02-dev-faq/planner-role-tool-limits.md` — planner 工具限制（前置知识）

---

## 反面案例：一个不适合 goal 模式的提示词

以下是一个来自实际使用的提示词，它完美展示了"什么不该用 goal 模式"：

### 原始提示词

```
继续优化代码，基于对数据库中数据样本的分析，结合软件公司如何寻找商机、
寻找靠谱的代理商与合作伙伴、如何发掘新产品方向（例如发达地区的新颖项目），
对程序进行迭代，注意：不要对生产环境的已运行服务干扰。具体测试完全可以在
本机运行，上面这些需求没有必须修改已有数据的，主要围绕web-app进行优化，
如果有新功能或新页面，必要的情况下可以新增数据表（原表尽可能不要动，生产
记录很多，非常容易锁表），如果原表加字段没有风险，也可以加字段；本地也有
ollama模型可以调用，提供llm相关的功能测试支撑。
```

### 被 Goal 块注入后的实际输入

Planner 实际收到的不是原始文本，而是 `Compose()` 注入后的产物：

```
<active-goal>
继续优化代码，基于对数据库中数据样本的分析，结合软件公司如何寻找商机…
Goal mode: pursue this goal autonomously. Do not stop after describing a plan;
execute the next useful step.
</active-goal>

Start pursuing the active goal now.
```

Planner 看到的是： **"分析数据 → 想出商机 → 执行迭代"** + **"不要停在计划阶段，执行下一步"**。

### 四个致命结构缺陷

| 问题 | 具体表现 |
|------|----------|
| **目标是"研究"而非"执行"** | "基于对数据库中数据样本的分析"、"结合…如何寻找商机"——任务本质是**认知/判断**，不是编码。Planner 只能用只读工具，所以它的唯一"执行"方式就是不断读 |
| **"做什么"未定义** | "对程序进行迭代"是空洞的——迭代什么？哪个模块？什么方向？Planner 不得不先"搞清楚要做什么"，这本身就需要大量分析 |
| **条件性许可需要判断** | "必要的情况下可以新增数据表"、"如果原表加字段没有风险"——planner 需要先判断"是否必要"和"是否有风险"，这两个判断都依赖深度分析 |
| **多目标混杂** | "商机发现" + "代理商合作" + "新产品方向" + "LLM 功能" + "不动原表" + "本地测试"——六个维度的高层决策浓缩在一段话里 |

### 矛盾链

```
Planner 系统提示: "stop once you have enough evidence → 产出简洁计划"
                       ↕ 直接冲突
Goal 块注入:        "execute the next useful step → 不要停在计划"
                       ↕ 能力缺口
Planner 实际能力:   只有 glob / grep / read_file / ls / lsp_*
                       ↕ 逻辑死结
任务本身:           "分析数据样本 → 理解商业模式 → 决定做什么"
                       ↓
              唯一"执行" = 读更多文件 → 分析瘫痪
```

### 什么提示词适合 goal 模式？

适合 goal 模式的提示词特征——**"做什么"是明确的**，需要的是执行，不需要"先研究再决定"：

```
✅ "在 web-app/src/api/ 下添加 POST /leads/import 接口，支持 CSV 上传"
✅ "修复 invoices 表的 tax_rate 默认值 bug，加单元测试"
✅ "把 Dashboard 页面的查询从 N+1 改成批量 JOIN"
```

### 正确的用法

这类"先分析再决策"的任务**不适合 goal 模式**，应该拆成多步人工引导：

```
📋 Normal/Plan 模式逐步引导：
  Turn 1: "先分析数据样本，看看有哪些潜在商机方向"
          [AI 输出分析结果]
  Turn 2: [人工评估 AI 输出，选一个方向]
  Turn 3: "好，方向 A，实现功能 X"
```

**Goal 模式 ≠ 把大型模糊任务扔给 AI 自动跑。** Goal 模式的前提是"知道要做什么，只需执行"。

---

## 单模型 vs 双模型的 Goal 行为差异

Goal 模式的瘫痪风险**高度依赖于是否配置了独立 planner**：

| 维度 | 单模型（无 planner_model） | 双模型（有 planner_model） |
|------|---------------------------|---------------------------|
| 每次 turn 的角色 | 同一个模型既规划又执行，有完整工具集 | Planner 只读调研 → Executor 执行 |
| Goal 块的影响 | 「执行下一步」是合理指令——模型可以调用 bash/write_file | 「执行下一步」与 planner 系统提示直接冲突 |
| 分析瘫痪风险 | 低——模型可以边读边写，自然推进 | **高**——planner 只有一个动作可做：读更多文件 |
| `[goal:continue]` 含义 | 持续执行，每轮都有产出 | Planner 可能纯消耗 token 无产出 |

### 判断自己是否处于双模型模式

查看 TOML 配置文件（`~/.reasonix/reasonix.toml` 或项目级配置）：

```toml
[agent]
planner_model = "deepseek-v4-pro"  # ← 有这行 = 双模型，高风险
```

如果没有 `planner_model` 字段，则为单模型——Goal 模式的瘫痪风险显著降低。
