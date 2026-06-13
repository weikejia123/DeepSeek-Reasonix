# Agent Core Loop — Run 完整循环

<!--
版本: v1.0
创建: 2026-06-13 23:10:00
更新: 2026-06-13 23:10:00
-->

**能力类型**: 系统架构文档  
**代码位置**: `internal/agent/agent.go:535-680`

## 架构概览

Agent.Run 是系统的核心大脑——从接收用户输入到返回最终答案的单次 turn 完整执行循环。

```
Agent.Run(ctx, input)
  │
  ├─ 前置: clearSteerQueue / reset evidence / emit TurnStarted / add user msg
  │
  └─ for step := 0; step < maxSteps; step++:
       │
       ├─ 1. consumeSteer()               ← 消费队列中的 mid-turn steer
       │
       ├─ 2. capturePrefixShape()         ← 拍摄系统前缀形状快照
       │
       ├─ 3. stream(ctx, step+1)          ← ★ 调用模型 API
       │     ├─ 返回: text, reasoning, signature, calls, usage
       │     ├─ 中断恢复: 最多 3 次 streamRecoveries
       │     └─ 失败: return err
       │
       ├─ 4. CompareShape / emit Usage    ← 前缀缓存诊断
       │
       ├─ 5. session.Add(assistant msg)   ← 记录模型回复
       │
       ├─ 6. calls == 0?  (模型未调工具)
       │     ├─ finalReadinessCheck?      ← 最终答案就绪检查
       │     ├─ 无可见答案?               ← emptyFinal retry (最多3次)
       │     ├─ executor没干活?           ← handoff nudge (最多3次)
       │     ├─ steerQueueLen > 0?        ← 还有 steer → continue
       │     └─ return nil                ← ★★ 最终答案!
       │
       └─ 7. calls > 0  (模型调了工具)
             ├─ executeBatch(ctx, calls)  ← 并行/串行执行工具
             ├─ session.Add(tool msgs)    ← 记录工具结果
             └─ maybeCompact(ctx)         ← 必要时压缩上下文
             └─ continue ← 下一轮迭代
```

---

## 一、关键数据结构

```go
// agent.go:535-552
func (a *Agent) Run(ctx context.Context, input string) error {
    defer a.clearSteerQueue()     // turn 结束清理 steer 队列
    a.evidence.Reset()            // 重置 evidence ledger
    a.sink.Emit(TurnStarted)
    a.session.Add(user message)

    for step := 0; a.maxSteps <= 0 || step < a.maxSteps; step++ {
        // ... 循环体
    }
}
```

### 循环变量

| 变量 | 作用 |
|------|------|
| `step` | 当前迭代轮次（0-based），受 `maxSteps` 约束（≤0 无上限） |
| `streamRecoveries` | 流中断恢复计数（最多 `maxStreamRecoveries=3`） |
| `finalReadinessBlocks` | 最终答案就绪检查被拒次数（最多 `maxFinalReadinessBlocks=3`） |
| `emptyFinalBlocks` | 模型无可见文本返回计数（最多 `maxEmptyFinalBlocks=3`） |
| `handoffNudges` | executor 无行动催促进度（最多 `maxExecutorHandoffNudges=3`） |
| `usedAnyTool` | 本轮是否至少调用过一次工具 |

---

## 二、stream() — 模型调用

`stream(ctx, step+1)`（`agent.go:569`）：
- 构建请求：系统前缀 + session 消息 + 工具 schemas
- 调用 provider API（流式）
- 返回：文本、推理内容、工具调用列表、usage 统计

### 中断恢复

```go
if interrupted && streamRecoveries < maxStreamRecoveries {
    streamRecoveries++
    session.Add(assistant msg)
    session.Add(streamRecoveryMessage)
    sink.Emit(Retrying)
    step--  // 不消耗 maxSteps 配额
    continue
}
```

---

## 三、最终答案判定（calls == 0）

当模型不调用工具时，系统认为模型试图给出最终答案。但需经过四道检查：

### 3.1 finalReadinessCheck — 就绪检查

验证模型是否真正完成了预期工作（检查 todo 列表、plan 完成状态等）：

```
就绪 → 继续判定
未就绪 → notice + retry message + maybeCompact + continue (最多3次)
```

### 3.2 emptyFinal — 空答案检测

模型返回了无可见文本的 assistant 消息：

```
无可见文本 → notice + retry + continue (最多3次)
```

### 3.3 executorHandoff — 无行动催促

plan research 后 executor 收到 `planApprovedMessage` 但未调用任何工具：

```
无行动 → notice + nudge + continue (最多3次)
```

### 3.4 steerQueue — 队列未空

即使模型想返回 final answer，只要 steer 队列还有消息：

```go
// agent.go:652
if a.steerQueueLen() > 0 {
    continue  // 继续循环，不返回 final answer
}
```

---

## 四、工具执行分支（calls > 0）

```
executeBatch(ctx, calls)
  ├─ partitionToolCalls → 并行/串行分批
  ├─ executeOne per call → 权限门控 + 实际执行
  └─ emit ToolResult per call

session.Add(tool messages) → 记录每个工具的结果
maybeCompact(ctx)          → 上下文接近窗口时压缩
continue → 下一轮迭代
```

---

## 五、前置与后置

### 前置（Run 入口）

1. `clearSteerQueue()` — 清理上个 turn 的残留 steer
2. `evidence.Reset()` — 重置当前 turn 的 evidence ledger
3. `sink.Emit(TurnStarted)` — 通知前端
4. `session.Add(user message)` — 记录用户消息（含 possible images）

### 后置（Run 退出）

- `defer clearSteerQueue()` — 清空 steer 队列
- 正常返回 `nil` — 模型给出 final answer
- 错误返回 — 外部 `runGuarded` 捕获并 emit TurnDone

---

## 六、防护机制总结

| 机制 | 阈值 | 触发后行为 |
|------|------|----------|
| maxSteps | 可配置 | step ≥ maxSteps → error + 保存状态 |
| streamRecoveries | 3 | 中断 → retry message + 不消耗 step 配额 |
| finalReadinessBlocks | 3 | 未就绪 → retry message + continue |
| emptyFinalBlocks | 3 | 无可见答案 → retry + continue |
| handoffNudges | 3 | 无行动 → nudge + continue |
| steerQueue 非空 | 阻止退出 | 有 steer → continue 直到队列清空 |

---

## 七、与外部 Controller 的配合

```
controller.runGuarded()
  └─ controller.runTurnWithRawDisplay()
       └─ a.Run(ctx, composed_input)       ← Agent.Run
            └─ return nil / err

controller 负责:
  - Compose() 注入 goal/plan/memory/jobs 块
  - beginCheckpoint / snapshotActivity
  - Hook PromptSubmit/Stop
  - Plan approval 子 turn
  - Goal continueGoal 循环
```

---

## 八、代码索引

| 文件 | 行号 | 内容 |
|------|------|------|
| `internal/agent/agent.go` | 535-680 | `Run()` 完整循环 |
| `internal/agent/agent.go` | 558-561 | `consumeSteer()` 消费 |
| `internal/agent/agent.go` | 569 | `stream()` 模型调用 |
| `internal/agent/agent.go` | 616-660 | 最终答案四道检查 |
| `internal/agent/agent.go` | 664-672 | `executeBatch` + 记录 |
| `internal/agent/agent.go` | 1085-1141 | `executeBatch` 实现 |
| `internal/agent/agent.go` | 1155 | `partitionToolCalls` |
| `internal/control/controller.go` | 524-588 | `runTurnWithRawDisplay` |
