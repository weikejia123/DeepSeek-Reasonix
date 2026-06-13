# Frontend Architecture — Desktop React 应用架构

<!--
版本: v1.0
创建: 2026-06-13 23:57:00
更新: 2026-06-13 23:57:00
-->

**能力类型**: 系统架构文档
**涉及文件**: `desktop/frontend/src/`

## 架构概览

```
Desktop (Wails)
  ├─ Go 后端 (desktop/app.go)
  │     └─ 暴露 Bridge API 给前端
  │
  └─ React 前端 (desktop/frontend/src/)
        ├─ App.tsx            ← 顶层组件 + handleSend 路由
        ├─ useController.ts   ← 状态管理 + reducer
        ├─ Composer.tsx       ← 输入框 + 模式选择
        ├─ StatusBar.tsx      ← 底部状态栏
        ├─ Sidebar.tsx        ← 标签管理
        └─ ChatView.tsx       ← 消息渲染
```

## 一、Bridge — Go ↔ React IPC

```typescript
// bridge.ts: Wails 绑定的 Go 方法
interface App {
    SubmitToTab(tabID, text): Promise<void>
    CancelTab(tabID): Promise<void>
    ListWorkspaceFiles(): Promise<string[]>
    SetGoalForTab(tabID, goal): void
    // ... 100+ 方法
}
```

## 二、useController — 核心状态管理

Reducer 模式（类似 Redux）：

```typescript
interface State {
    items: Item[]           // 消息列表
    running: boolean        // turn 运行中
    turnActive: boolean     // turn 激活
    pendingUser?: string    // 待确认的用户消息
    live?: LiveStream       // 实时流
    turnQueue?: TurnQueueItem[]
    context: ContextInfo
    approval?: WireApproval
    ask?: WireAsk
    // ...
}

function reducer(s: State, a: Action): State {
    switch (a.type) {
        case "user": ...
        case "turn_started": ...
        case "text": ...        // 流式文本追加到 live
        case "tool_dispatch": ...
        case "tool_result": ...
        case "usage": ...
        case "turn_done": ...
    }
}
```

## 三、事件→Action 映射

```
Go Event      →  前端 Action
────────────────────────────
TurnStarted   →  turn_started
Text          →  text (追加 live)
Reasoning     →  reasoning
Message       →  message (终结 live)
ToolDispatch  →  tool_dispatch
ToolResult    →  tool_result
ToolProgress  →  tool_progress
Usage         →  usage
Notice        →  notice
Steer         →  steer (↪ notice)
TurnDone      →  turn_done
ApprovalReq   →  approval_request
AskRequest    →  ask_request
Compaction*   →  compaction_*
```

## 四、handleSend 路由 (App.tsx)

```
用户输入 → handleSend()
  ├─ /goal → applyGoal + send
  ├─ !cmd  → runShell
  ├─ /theme → notice
  ├─ runningRef true? → steer()   ← mid-turn steer
  └─ 默认 → send()
       ├─ set goal → SetGoal
       └─ SubmitToTab → Go 后端
```

## 五、代码索引

| 文件 | 内容 |
|------|------|
| `desktop/frontend/src/App.tsx` | 顶层组件、handleSend、tab 管理 |
| `desktop/frontend/src/lib/useController.ts` | State、reducer、send/cancel/steer |
| `desktop/frontend/src/lib/bridge.ts` | Go↔React Bridge 接口 |
| `desktop/frontend/src/components/Composer.tsx` | 输入框 |
| `desktop/frontend/src/components/StatusBar.tsx` | 状态栏 |
| `desktop/app.go` | Go 后端 Bridge 方法 |
