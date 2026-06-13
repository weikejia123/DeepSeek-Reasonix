# Checkpoint & Rewind — 快照与回退

<!--
版本: v1.0
创建: 2026-06-13 23:51:00
更新: 2026-06-13 23:51:00
-->

**能力类型**: 系统架构文档

## 一、Checkpoint 创建

每 turn 开始前自动创建检查点：

```go
// controller.go:535
c.beginCheckpoint(input)  // 记录 turn 边界
```

```
Checkpoint { turn, prompt, files[], time, canCode, canConversation }
```

## 二、Rewind 回退

```
/rewind [turn] [scope]
  ├─ turn: 目标 turn 编号
  ├─ scope: code | conversation | both (default)
  │
  ├─ code: 回退文件变更到 turn 时的 git 状态
  ├─ conversation: 回退 session messages 到 turn 前
  └─ both: 同时回退代码和对话
```

## 三、CheckpointMeta

```go
type CheckpointMeta struct {
    Turn           int
    Prompt         string
    Files          []string   // 变更的文件列表
    Time           int64      // unix ms
    CanCode        bool       // 可回退代码
    CanConversation bool      // 可回退对话
}
```

## 四、Rewind 实现

1. `Git stash` + `checkout` 恢复文件到指定 turn 的 commit
2. `Session.Messages` 截断到 turn 索引
3. `rewriteVersion++` 标记 session 被修改

## 五、与 Branch 的关系

Rewind 不会创建分支——直接修改当前 session。如需保留当前状态，先用 `/branch` 分叉。

## 六、代码索引

| 文件 | 内容 |
|------|------|
| `internal/control/controller.go:535` | beginCheckpoint |
| `internal/control/controller.go:849-859` | /rewind 路由 |
| `internal/control/checkpoint.go` | Checkpoint/Rewind 实现 |
