# Approval & Gate — 权限门控

<!--
版本: v1.0
创建: 2026-06-13 23:49:00
更新: 2026-06-13 23:49:00
-->

**能力类型**: 系统架构文档

## 架构

```
executeOne(ctx, call)
  ├─ Gate.PreToolUse (hook 阻塞点)
  ├─ Gate gate (内部权限门控)
  │     ├─ ReadOnly()? → 自动放行
  │     ├─ YOLO? → 自动放行 (approvedPlanAutoApproveTools)
  │     ├─ Session grant? → 自动放行
  │     └─ 否则 → ApprovalRequest → 等待用户
  └─ 执行工具
```

## 一、三层权限

| 层 | 机制 | 控制点 |
|-----|------|--------|
| Hook 阻塞 | `PreToolUse` hook → block=true | 外部脚本否决 |
| 权限门控 | `Gate` 接口 → `ApprovalRequest` | 用户交互确认 |
| 沙箱 | macOS Seatbelt / 仅 Linux | OS 级隔离 |

## 二、Gate 接口

```go
type Gate interface {
    Allow(ctx, toolName, args) (allow bool, session bool, persist bool, err error)
}
```

- `allow` — 本次放行
- `session` — 本次 session 内记住选择
- `persist` — 持久化到配置

## 三、YOLO 模式

```
ToolApprovalMode:
  "ask"  → 每次写操作询问
  "auto" → ApprovalRequest 自动批准（仍展示卡片）
  "yolo" → 完全跳过 Gate，工具直接执行
```

`ask` 工具不受 YOLO 影响——它请求用户决策，而非工具操作。

## 四、Approved Plan 自动批准

Plan 批准后的执行 turn 中 `approvedPlanAutoApproveTools=true`，所有写入工具自动批准（仅当前 turn）。

## 五、Session Grant

用户可选择 "Always allow for this session" — `SessionGate` 记住选择，同一 session 内不再询问。

## 六、代码索引

| 文件 | 内容 |
|------|------|
| `internal/control/approval.go` | Gate 接口、SessionGate |
| `internal/agent/agent.go:1291` | `executeOne` 门控调用点 |
| `internal/control/controller.go:578-582` | approvedPlanAutoApproveTools |
| `internal/hook/hook.go:34-35` | PreToolUse / PostToolUse |
