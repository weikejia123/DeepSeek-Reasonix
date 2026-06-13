# Event System — 事件流与前端的通信骨架

<!--
版本: v1.0
创建: 2026-06-13 23:55:00
更新: 2026-06-13 23:55:00
-->

**能力类型**: 系统架构文档
**代码位置**: `internal/event/event.go`

## 架构

```
Agent.Run() / Controller 操作
  │
  └─ sink.Emit(Event{Kind, ...})
        │
        ▼
Sink 接口 → 分发给各前端
  ├─ TUI (bubbletea messages)
  ├─ Desktop (Wails events → React reducer)
  ├─ HTTP/SSE (server-sent events)
  └─ ACP (Agent Client Protocol)
```

## 一、Event 结构

```go
type Event struct {
    Kind             Kind
    Text             string            // Reasoning/Text/Notice/Phase/Steer
    Reasoning        string            // Message 中的完整推理链
    Tool             Tool              // ToolDispatch/ToolResult/ToolProgress
    Usage            *provider.Usage   // Usage
    Pricing          *provider.Pricing
    CacheDiagnostics *CacheDiagnostics
    Level            Level             // Notice: info/warn
    Approval         Approval          // ApprovalRequest
    Ask              Ask               // AskRequest
    Err              error             // TurnDone
    Compaction       Compaction        // CompactionStarted/Done
    RetryAttempt     int               // Retrying
    RetryMax         int
    SessionHit       int               // 会话累计缓存命中
    SessionMiss      int
    TabID            string            // 多 Tab 路由
}
```

## 二、完整事件目录 (19 kinds)

| 事件 | 携带字段 | 发射时机 |
|------|---------|---------|
| `TurnStarted` | — | Agent.Run() 入口 |
| `Reasoning` | Text | 推理流 delta |
| `Text` | Text | 回复文本流 delta |
| `Message` | Text, Reasoning | Assistant turn 完成 |
| `ToolDispatch` | Tool{ID,Name,Args,ReadOnly,FileDiff,Profile} | 工具调用前 |
| `ToolProgress` | Tool{ID,Output} | 长运行工具的输出流 |
| `ToolResult` | Tool{Output,Err,Truncated,DurationMs} | 工具调用完成 |
| `Usage` | Usage,Pricing,CacheDiagnostics | 每步模型调用后 |
| `Notice` | Level,Text | 警告/截断/阻塞通知 |
| `Phase` | Text | 协调器边界（planner→executor） |
| `ApprovalRequest` | Approval{Tool,Subject} | 需用户批准的工具 |
| `AskRequest` | Ask{Questions} | ask 工具的多选问题 |
| `TurnDone` | Err (nil=成功) | Run() 退出 |
| `CompactionStarted` | Compaction{Trigger} | 压缩开始 |
| `CompactionDone` | Compaction{Summary,Messages,Archive} | 压缩完成 |
| `Steer` | Text | mid-turn steer 被消费 |
| `Retrying` | RetryAttempt,RetryMax | 流恢复重试前 |
| `BackgroundJob` | Text | 后台 MCP 连接状态 |
| (`MCPServerReady`) | Text | MCP 服务器就绪（同 BackgroundJob） |

## 三、Sink 接口

```go
type Sink interface {
    Emit(Event)
}
```

多个实现：
- `event.Discard` — 空实现（headless 运行）
- TUI sink → bubbletea messages
- Desktop sink → Wails IPC → React reducer
- HTTP sink → SSE stream

## 四、多 Tab 路由

`Event.TabID` 字段让多 Tab 前端将事件路由到正确的 per-tab reducer。

## 五、代码索引

| 文件 | 内容 |
|------|------|
| `internal/event/event.go:24-90` | Kind 枚举定义 |
| `internal/event/event.go:107-228` | Event/Tool/Approval/Ask/Compaction 结构 |
