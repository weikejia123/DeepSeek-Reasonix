# Steer Queue — Mid-Turn 消息队列
<!--
版本: v1.0
创建: 2026-06-13 22:10:00
更新: 2026-06-13 22:10:00
-->

**能力类型**: 基础能力  
**提供方**: Go 后端 Agent (`internal/agent/agent.go`)  
**触发入口**: 用户在 turn 运行期间发送消息 → 前端自动路由到 `steer()`  
**代码位置**: `internal/agent/agent.go:225-232`（数据结构）、`391-415`（入队/出队）、`558-561`（消费）、`652`（阻止提前返回）

## 能力概述

在 turn 运行期间，用户发送的消息不会丢失，也不会被 Go 后端的 `c.running` 互斥丢弃。相反，消息进入 Agent 内部的 **Steer 队列**（`steerQueue`），在当前 turn 的每次循环迭代中被逐条消费，注入模型作为"继续当前任务的指导"。

## 功能边界

- ✅ **不会被丢弃**——消息进入 `steerQueue` FIFO 队列
- ❌ **没有显式条数限制**——队列是 Go slice，不设上限
- ❌ **没有显式字数限制**——每条消息完整存储，不做截断
- ⚠️ **实际约束**——仅受 context window（~128K tokens）和内存的隐式约束；足够多的长消息会挤满窗口

## 完整数据流

```
用户发送消息（turn 运行中）
        │
        ▼
App.tsx:1483  runningRef.current == true ?
        │ yes
        ▼
useController.steer(text)                    // useController.ts:794-800
        │
        ▼
app.SteerForTab(tabId, text)                 // desktop/app.go:620-624
        │
        ▼
controller.Steer(text)                       // controller.go:1166-1182
  if running { exec.Steer(text); return }    // → agent.Steer(text)
        │
        ▼
agent.Steer(text)                            // agent.go:391-396
  steerQueue = append(steerQueue, text)       // 追加到 FIFO 队列
        │
        ▼
[Agent loop 下一次迭代]
agent.consumeSteer()                         // agent.go:558-561
  text = steerQueue[0]; steerQueue = steerQueue[1:]  // 出队
  session.Add(MidTurnSteerPrefix + "\n" + text)       // 注入会话
  sink.Emit(event.Steer{Text: text})                   // 通知前端 "↪ text"
        │
        ▼
前端 reducer: "steer" case                   // useController.ts:438-439
  items.push({ kind:"notice", text:"↪ " + text })
        │
        ▼
[Agent 准备返回 final answer 时]
agent.go:652  if steerQueueLen() > 0 { continue }  // 还有 steer → 不返回，继续循环
        │
        ▼
下一个 loop 迭代 → consumeSteer() → 处理下一条 steer → …
        │
        ▼
steerQueue 清空 → 返回 final answer → turn_done
```

## 关键源码

| 文件 | 行号 | 作用 |
|------|------|------|
| `desktop/frontend/src/App.tsx` | 1483 | running 检测 → 路由到 steer() 而非 send() |
| `desktop/frontend/src/lib/useController.ts` | 794-800 | steer() 调用 app.SteerForTab |
| `desktop/app.go` | 619-624 | SteerForTab → controller.Steer |
| `internal/control/controller.go` | 1166-1182 | 根据 running 状态：运行中→exec.Steer，空闲→启动新 turn |
| `internal/agent/agent.go` | 225-232 | steerQueue 数据结构 |
| `internal/agent/agent.go` | 391-396 | Steer() 入队 |
| `internal/agent/agent.go` | 405-415 | consumeSteer() 出队（FIFO） |
| `internal/agent/agent.go` | 558-561 | 每步循环消费一条 steer |
| `internal/agent/agent.go` | 652 | final answer 前检查：队列非空→继续循环 |
| `internal/agent/agent.go` | 364-368 | MidTurnSteerPrefix 常量 |

## 条数限制分析

```go
// agent.go:391-396
func (a *Agent) Steer(text string) {
    a.steerMu.Lock()
    defer a.steerMu.Unlock()
    a.steerQueue = append(a.steerQueue, text)  // Go slice，无界
    a.steerConsumed = false
}
```

无 `if len(steerQueue) >= max { ... }` 检查。理论上可不断追加直到内存耗尽。

**隐式约束**：
- 每条 steer 注入当前 turn 的 session，占用 context tokens
- 默认 context window = ~128K tokens；prefix 约 100 字符（~25 tokens）+ 用户消息
- 粗略估算：每条 steer 平均 50 tokens，可排 ~2000+ 条才挤满窗口

## 字数限制分析

`steerQueue` 存储完整 `[]string`，不做截断。非常长的 steer（几千字）会被完整注入模型，消耗 context window、增加 token 成本，但不影响队列机制。

## 消费行为：同 Turn 内逐条消费

与 TUI 的 `pendingInterject`（当前 turn 结束后启动**新 turn**）不同，Agent Steer 队列的消费发生在**同一 turn 的循环迭代中**：

```go
// agent.go:553-561
for step := 0; ...; step++ {
    if text, ok := a.consumeSteer(); ok {
        a.session.Add(provider.Message{
            Role:    provider.RoleUser, 
            Content: midTurnSteerMessage(text),
        })
        a.sink.Emit(event.Event{Kind: event.Steer, Text: text})
    }
    // ... stream, execute tools ...
}
```

```go
// agent.go:652
if a.steerQueueLen() > 0 {
    continue  // 不返回，继续消费下一条 steer
}
return nil  // 队列清空后才返回 final answer
```

## 队列存活期

- **存活范围**：仅限当前 Agent 生命周期（当前 turn 内），turn_done 后 `clearSteerQueue()` 清空
- **边界情况**：若用户在 turn 快结束时发送，且 Agent 已通过 `steerQueueLen() > 0` 检查，则 `controller.Steer()` 检测到 running=false 后自动 `go func() { c.SubmitDisplay(text, text) }()` 启动新 turn，不丢消息

## Steer vs TUI pendingInterject

| 维度 | Agent Steer (Desktop) | TUI pendingInterject |
|------|----------------------|---------------------|
| 消费时机 | 同一 turn 的循环迭代中 | 当前 turn 结束后启动新 turn |
| 模型视角 | 带 prefix 标记的指导消息 | 普通用户消息，开启全新 turn |
| 前端展示 | `↪ 消息内容` notice | 普通 user bubble |
| 代码位置 | `agent.go:225-232` | `chat_tui.go:96-98` |
| Go 后端互斥 | 不经过 `runGuarded` | 新 turn 需 `Send()` → `runGuarded` |

## 关联能力

- `controller.Steer()` — 运行中入队 vs 空闲时启动新 turn 的路由逻辑
- `controller.SubmitToTab()` — 正常 send 路径（非 running 时使用）
- `App.tsx:1483` — 前端 running 检测，决定走 steer 还是 send
