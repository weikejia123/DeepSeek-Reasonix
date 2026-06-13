# Session & Persistence — 会话生命周期与持久化

<!--
版本: v1.0
创建: 2026-06-13 23:40:00
更新: 2026-06-13 23:40:00
-->

**能力类型**: 系统架构文档
**涉及包**: `internal/agent/session.go`、`internal/agent/subagent_store.go`、`internal/control/controller.go`

## 架构概览

```
Session (内存)
  Messages []provider.Message  ← user/assistant/tool/steer/compaction
  rewriteVersion int            ← compact/fold 时递增
        │
        ▼
持久化 (JSONL, controller.go)
  SessionPath → <sessionDir>/<sessionID>.jsonl
  每消息一行 JSON，追加写入
        │
        ▼
恢复 (controller resume/new)
  Load session → Snapshot() → 重建 history
        │
        ▼
检查点 (checkpoint)
  beginCheckpoint(input) → 每 turn 前快照
  用于 Rewind 回退
```

## 一、Session 结构

```go
// agent/session.go:16-20
type Session struct {
    mu             sync.RWMutex
    Messages       []provider.Message  // 消息数组
    rewriteVersion int                 // compact/fold 时递增
}
```

### 关键方法

| 方法 | 线程安全 | 说明 |
|------|:---:|------|
| `NewSession(system)` | ✅ | 创建，可选系统提示词 |
| `Add(msg)` | ✅ | 追加一条消息 |
| `Replace(msgs)` | ✅ | 替换全部消息（compact 用） |
| `Snapshot()` | ✅ | 返回消息副本（前端读取） |

---

## 二、持久化格式

JSONL — 每行一个 JSON 对象，对应一个 `provider.Message`：

```jsonl
{"role":"system","content":"You are..."}
{"role":"user","content":"analyze the project"}
{"role":"assistant","content":"I'll start by...","toolCalls":[...]}
{"role":"tool","toolCallId":"call-1","name":"read_file","content":"..."}
```

### 自动保存 (autosave)

`controller.autosaveWG` — 每 turn 异步保存，不阻塞 UI。

---

## 三、会话生命周期

```
创建: NewSession(system) → Session.Messages = [system msg]
      ↓
运行: Run(ctx, input) → Add(user) → stream → Add(assistant) → Add(tool) → loop
      ↓ 每次 turn 后 autosave
      ↓
压缩: compact() → Replace(compacted messages)
      ↓
分叉: ForkNamed(turn, name) → 从 turn 索引处复制 messages 到新 session
      ↓
恢复: Load session → 通过 historyMessagesToItems 重建前端 items
```

---

## 四、Tab/Session 管理

Desktop 每个 Tab 对应一个 Controller + Session：

```
Tab { id, sessionPath, cwd, goal, ... }
  └─ Controller { session, runner, sink, ... }
       └─ Agent { session, prov, tools, ... }
```

Tab 切换 → 切换 active controller → 不同 session 独立运行。

---

## 五、子代理 Session

Subagent 有独立 Session（`subagent_store.go`），支持：

- `PrepareFresh(spec)` — 新建子代理 session
- `PrepareContinue(ref, spec)` — 同一子代理继续
- `PrepareFork(ref, spec)` — 从某子代理分叉
- `SaveCompleted(run)` / `SaveFailed(run)` — 持久化结果

---

## 六、代码索引

| 文件 | 内容 |
|------|------|
| `internal/agent/session.go` | Session 结构、Add/Replace/Snapshot |
| `internal/agent/subagent_store.go` | 子代理 session 管理 |
| `internal/control/controller.go:418-422` | autosave 机制 |
| `internal/control/controller.go:1912-1935` | SessionPath / parentSessionID |
