# Evidence & Todo — 任务验证系统

<!--
版本: v1.0
创建: 2026-06-13 23:50:00
更新: 2026-06-13 23:50:00
-->

**能力类型**: 系统架构文档
**涉及包**: `internal/evidence/`

## 一、Evidence Ledger

每 turn 维护一个 **evidence ledger**——记录本 turn 内工具执行的证据：

```go
// agent.go:234-236
evidence *evidence.Ledger  // 每 turn 重置
```

```
executeOne()
  ├─ 工具执行 → 记录 receipt (command+result)
  └─ evidence.Add(receipt)
        │
complete_step 调用
  ├─ 读取 evidence ledger
  ├─ 验证 cited evidence 存在且匹配
  └─ 签名完成步骤
```

## 二、complete_step 验证

```go
complete_step(step, result, evidence[])
  ├─ 1. 查找 evidence ledger 中匹配的 receipt
  │     ├─ verification → 验证 command 匹配
  │     ├─ diff → 验证 paths 匹配
  │     ├─ files → 验证 paths 匹配
  │     └─ manual → 人工确认
  ├─ 2. 无匹配 → REJECTED (no evidence)
  ├─ 3. 匹配 → 签名 step 完成
  └─ 4. advance todo list (mark step done, next in_progress)
```

## 三、Todo State

Host 维护 canonical task list，跨 turn 持久：

```go
// agent.go:243-244
todoState []evidence.TodoItem  // 跨 turn 存活，compaction 不压缩
```

- `todo_write` → 更新 todoState
- `complete_step` → 标记项完成、推进下一项
- Final answer gate → 检查所有项是否完成

## 四、Final Readiness Check

Agent 准备返回 final answer 时验证：
- Todo 列表是否所有项 completed
- 是否有 blocked 项需要用户介入
- 不合格 → retry message + continue

## 五、代码索引

| 文件 | 内容 |
|------|------|
| `internal/evidence/ledger.go` | Evidence Ledger 结构 |
| `internal/agent/agent.go:234-244` | evidence / todoState 字段 |
| `internal/agent/agent.go:616-625` | finalReadinessCheck |
| `internal/tool/builtin/completestep.go` | complete_step 实现 |
| `internal/tool/builtin/todo.go` | todo_write 实现 |
