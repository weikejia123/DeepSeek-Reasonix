# Auto-Plan — 自动分类与计划触发

<!--
版本: v1.0
创建: 2026-06-13 23:52:00
更新: 2026-06-13 23:52:00
-->

**能力类型**: 系统架构文档
**代码位置**: `internal/control/auto_plan.go`

## 一、触发条件

```go
func (c *Controller) maybeAutoPlan(ctx, input) {
    if shouldAutoPlan(ctx, input) {
        SetPlanMode(true)
        notice("auto plan: task looks multi-step")
    }
}
```

## 二、shouldAutoPlan 判定

```go
func shouldAutoPlan(ctx, input) bool {
    if autoPlanOff || plan || goalActive { return false }
    score := autoPlanScore(input)
    return score > 0
}
```

互斥条件：
- `autoPlanOff` — 用户通过 `/auto-plan off` 关闭
- `plan` — 已在 plan 模式
- `goalActive` — Goal 运行中

## 三、autoPlanScore

使用**分类器模型**（轻量 cheap 模型）评分：

```
input → classifier.Chat(input) → score (0-10)
  ├─ 0     → 简单问题，不 plan
  ├─ 1-5   → 中等复杂，可选 plan
  └─ 6-10  → 复杂任务，自动 plan
```

分类器独立于主模型——通常是更便宜/更快的模型。

## 四、配置

```toml
[agent]
auto_plan = "auto"  # auto | off | always
```

- `auto` — 分类器评分决定
- `off` — 永不自动 plan
- `always` — 任何非空输入都 plan（开发调试用）

## 五、代码索引

| 文件 | 行号 | 内容 |
|------|------|------|
| `internal/control/auto_plan.go` | 33-37 | `maybeAutoPlan` |
| `internal/control/auto_plan.go` | 40-51 | `shouldAutoPlan` |
| `internal/control/auto_plan.go` | 53-70 | `autoPlanScore` |
