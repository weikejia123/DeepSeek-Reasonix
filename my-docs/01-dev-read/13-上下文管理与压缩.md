# Context & Compaction — 上下文管理与压缩

<!--
版本: v1.0
创建: 2026-06-13 23:25:00
更新: 2026-06-13 23:25:00
-->

**能力类型**: 系统架构文档  
**涉及包**: `internal/agent/agent.go`、`internal/control/input.go`、`internal/agent/compaction.go`

## 架构概览

Reasonix 的上下文管理分两层：**Compose 注入**（每 turn 动态）和 **Compaction 压缩**（触发时重写历史）。

```
每 Turn:
  controller.Compose(input)
    ├─ activeGoalBlock?      → <active-goal>...</active-goal>
    ├─ PlanModeMarker?        → [Plan mode — read-only...]
    ├─ <memory-update>        → 最近记忆变更
    └─ <background-jobs>      → 完成的后台任务

  Agent.Run(ctx, composed)
    ├─ session.Add(user msg)  → 记录用户消息
    ├─ stream(ctx)            → provider 自动构建请求:
    │   ├─ system prompt      ← 缓存稳定的前缀 (prefix)
    │   ├─ session messages   ← user/assistant/tool/steer
    │   └─ tool schemas       ← 当前注册工具的 JSON Schema
    └─ maybeCompact(ctx)      → 接近窗口时压缩

压缩触发:
  maybeCompact(ctx, usage)
    ├─ promptTokens >= window × softCompactRatio?  → 警告
    ├─ promptTokens >= window × compactRatio?      → compact()
    └─ promptTokens >= window × compactForceRatio? → 强制 compact
```

---

## 一、系统前缀（System Prefix / System Prompt）

系统提示词是**缓存稳定的**——内容在 session 期间不变，DeepSeek 的自动前缀缓存可以复用，大幅降低 token 成本。

### 构建时机（`boot.go`）

```
Build()
  ├─ 基础提示词 (i18n 消息)
  ├─ 工具描述 (schemas — 在 Run 中动态追加)
  ├─ skill.ApplyIndex(system prompt, skills)  — # Skills 索引
  └─ 其他静态块
```

### 缓存稳定性保证

- **skill bodies 不进 prefix** — 只有 name+description 在 `# Skills` 索引中
- **memory-update 不进 prefix** — 通过 `Compose()` 注入后 riding turn tail
- **工具 schemas** — 在 `Run` 循环中动态追加（不在 prefix 中），但 schemas 变化频率低

---

## 二、Compose 注入（`input.go:93-132`）

每 turn 的 `Compose()` 在用户输入前注入动态块：

```go
func (c *Controller) Compose(text string) string {
    if goal active { text = activeGoalBlock(goal) + text }
    if plan { text = PlanModeMarker + text }
    if memory notes { text = memoryUpdateBlock + text }
    if jobs completed { text = jobsBlock + text }
    return text
}
```

### 注入块

| 块 | 触发条件 | 格式 |
|-----|---------|------|
| `<active-goal>` | goal 运行中 | 目标文本 + 提示词指令 |
| `PlanModeMarker` | plan 模式 | 长 system prompt 级指令 |
| `<memory-update>` | 本 turn 有记忆变更 | "\n- Saved memory ..." |
| `<background-jobs>` | 有完成的后台任务 | 任务完成摘要 |

这些注入块**不进入**缓存稳定的 prefix——它们 ride the turn tail，所以下一 turn 的 prefix 依然 cacheable。

---

## 三、Compaction 压缩（`agent.go:675` 附近）

### 三阈值体系

| 阈值 | 默认值 | 行为 |
|------|--------|------|
| `softCompactRatio` | 0.7 | 首次接近时 emit warning notice（仅一次） |
| `compactRatio` | 0.8 | 自动触发压缩 |
| `compactForceRatio` | 0.95 | 强制压缩，忽略"刚压缩过"的保护 |

### 压缩过程

```
compact(ctx, trigger, instructions, force)
  ├─ 1. 保留: session 首部 (system prefix 等效) + 最近 recentKeep 条消息
  ├─ 2. 中间部分 → 发送给 summarizer 模型
  │     ├─ <skill-pin> 块保留原样
  │     └─ 其他压缩为结构化摘要
  ├─ 3. 摘要 → 插入到保留部分的 gap 中
  ├─ 4. 原始消息 → 归档到 archiveDir
  └─ 5. session = 压缩后的版本
```

### 保护机制

- `compactStuck` — 连续压缩 3 次仍超窗口 → 暂停自动压缩，防止死循环
- `softCompactNoticed` — 软阈值仅通知一次，不反复骚扰

---

## 四、手动压缩

`/compact [focus]` 命令触发手动压缩，`focus` 参数指导 summarizer 保留哪些内容：

```
/compact "keep the auth changes we just discussed"
```

---

## 五、关键参数

```go
// agent.go:260-275
contextWindow       int     // 模型最大 context，如 128000 (DeepSeek v3)
softCompactRatio    float64 // 0.7
compactRatio        float64 // 0.8 (Configurable via reasonix.toml)
compactForceRatio   float64 // 0.95
recentKeep          int     // 压缩后保留的最近消息数
```

---

## 六、代码索引

| 文件 | 内容 |
|------|------|
| `internal/control/input.go:93-132` | `Compose()` 动态注入 |
| `internal/control/input.go:134-146` | `activeGoalBlock()` |
| `internal/agent/agent.go:260-275` | 压缩阈值配置字段 |
| `internal/agent/agent.go:~675` | `maybeCompact()` |
| `internal/agent/compaction.go` | `compact()` 实现 |
| `internal/boot/boot.go:190-206` | `skill.ApplyIndex` 注入 prefix |
