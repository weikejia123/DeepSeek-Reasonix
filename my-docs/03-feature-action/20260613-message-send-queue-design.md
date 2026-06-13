# 消息发送队列设计草案

## 问题

当前 Desktop 在 turn 运行期间，用户发送第二条消息的行为不可见、不可控：

- Composer 的 `submittingRef` 在 `onSend()` 返回后立即复位，不等待 turn 完成
- Go 后端的 `Controller.runGuarded()` 用单一 `c.running` bool 互斥，并发提交被**静默丢弃**（`ErrTurnRunning`）
- 前端虽然有 `pendingUser` 状态字段，但仅用于"发送中 → turn_started 之间"的过渡，**不支持多条排队**
- UI 上没有任何队列指示器，用户不知道消息是被丢弃还是在等待

## 现状全链路

### 后端互斥（Go）

```
Controller.runGuarded()                     // controller.go:407
  if c.running { return }                   // 第二条消息静默丢弃
  c.running = true
  defer c.running = false                   // turn_done 时复位
```

### 前端发送流（TS）

```
Composer.submit()                           // Composer.tsx:822
  submittingRef.current = true              // 本地防抖，onSend 返回后立即复位
  onSend(displayText, submitText)
  submittingRef.current = false             // ← 此时 turn 仍在运行！

useController.send()                        // useController.ts:766
  dispatchTo({ type: "user", text })       // 乐观 UI: 立即显示 user bubble
  app.SubmitToTab(tabId, submit)            // → Go 后端

Reducer: case "user"                        // useController.ts:460-472
  return {
    running: true,
    pendingUser: a.text,                    // 记录待确认的消息
    ...
  }

Reducer: case "turn_started"                // useController.ts:336-344
  flushPendingUser(s)                       // pendingUser → 正式 user item
  → running: true, turnActive: true

Reducer: case "turn_done"                   // useController.ts:394-400
  → running: false, turnActive: false

Reducer: case "unsend"                      // useController.ts:474
  → pendingUser: undefined, running: false  // Esc 撤回
```

### 关键断点

| 时刻 | `running` | `pendingUser` | Composer `submitting` | Go `c.running` |
|------|-----------|---------------|----------------------|----------------|
| 用户点发送 | `true` | `"用户消息"` | `true` | `true` |
| `onSend` 返回 | `true` | `"用户消息"` | **`false`** ⚠️ | `true` |
| `turn_started` | `true` | `undefined` | `false` | `true` |
| **此时用户再点发送** | `true` | `"第二条消息"` | `true→false` | **丢弃第二条** |
| `turn_done` | `false` | `undefined` | `false` | `false` |

问题时刻在第 3 行：`submitting=false` 但 turn 仍在跑，用户可以发第二条消息，
前端看似接受了（乐观显示 user bubble），但 Go 后端已丢弃。

## TUI 已有完整实现

CLI（`chat_tui.go`）已经有完整的 turn 运行中消息队列，Desktop 是缺失方。

### 数据结构

```go
// chat_tui.go:96-105
pendingInterject []string   // 运行中排队消息，FIFO
queueEditCursor  int         // 当前浏览的队列项索引（-1 = 不在浏览）
queueEditDraft   string      // 用户首次按 ↑ 时保存的输入框草稿
```

### Enter 发送逻辑

```go
// chat_tui.go:1088-1102
if m.queueEditCursor >= 0 {
    // 正在浏览队列 → 保存编辑回槽位
    m.pendingInterject[m.queueEditCursor] = line
    m.notice("queue [N] updated")
} else {
    // 新消息 → 追加到队尾
    m.pendingInterject = append(m.pendingInterject, line)
    m.notice("feedback queued — will send when the current turn finishes")
}
```

### 队列浏览（↑/↓ 键）

```go
// chat_tui.go:580-613  navigateQueue(delta)
// ↑ 第一次：保存当前输入框内容到 queueEditDraft，跳到队尾
// ↑ 继续：向上浏览更早的队列项，输入框显示该消息内容
// ↓：向下浏览，到底后恢复 queueEditDraft（退出浏览）
```

### turn_done 时自动出队

```go
// chat_tui.go:1236-1242
if len(m.pendingInterject) > 0 {
    interject := m.pendingInterject[0]
    m.pendingInterject = m.pendingInterject[1:]
    m.queueEditCursor = -1
    m.queueEditDraft = ""
    cmds = append(cmds, m.startTurn(interject, interject, interject))
}
```

### 状态栏指示

```go
// chat_tui.go:2278-2282
if n := len(m.pendingInterject); n > 0 {
    if n == 1 { working += " · ✎ feedback queued" }
    else      { working += fmt.Sprintf(" · ✎ %d queued", n) }
}
```

### 队列预览（输入框上方）

```go
// chat_tui.go:625-648  renderQueueIndicator()
// 显示格式: "  ▸ [1] 帮我优化这段代码…"
//           "    [2] 再确认一下测试是否通过"
// 消息预览截断到 50 字符
```

### TUI vs Desktop 差异

| 能力 | TUI | Desktop |
|------|-----|---------|
| 运行中发送 → 入队 | ✅ | ❌ 静默丢弃 |
| 队列可视化 | ✅ 输入框上方 + 状态栏 | ❌ |
| 浏览/编辑队列项 | ✅ ↑/↓ 键 | ❌ |
| turn_done 自动出队 | ✅ | ❌ |
| 编辑后保存回队列 | ✅ | ❌ |
| 队列长度显示 | ✅ "✎ N queued" | ❌ |

Desktop 的实现应**复刻 TUI 的全部行为**，仅在交互方式上适配 GUI（鼠标点击代替键盘 ↑/↓，弹出面板代替终端行内预览）。

## 目标

1. 在 turn 运行期间，用户发送的消息**进入可见队列**而非丢弃
2. 队列在 UI 中可视化——用户看到排队中的消息
3. 当前 turn 完成后自动发送队列中的下一条
4. 支持取消队列中的消息（单条撤销或全部清空）

## 设计方案

### 数据模型

```
State 新增字段:
  turnQueue: TurnQueueItem[]
    // 排队中的消息，FIFO

TurnQueueItem:
  id: string           // "q{seq}"
  displayText: string  // 用户在输入框中的可见文本
  submitText: string   // 实际提交给模型的内容（含 @-refs 展开等）
  createdAt: number    // Date.now()
```

### 核心流程

```
用户发送消息
  ├─ Go 后端空闲 (c.running == false)
  │    → 直接提交，现有行为不变
  └─ Go 后端忙碌 (c.running == true)
       → 追加到 turnQueue 队尾
       → UI 显示队列徽标 "队列中 (1)"
       → turn_done 事件到达时:
           如果 turnQueue 非空，dequeue 第一个，
           调用 app.SubmitToTab 提交
           重置 running=true
```

### 状态转换

```
Reducer: "queue_turn"
  → turnQueue: [...s.turnQueue, { id, displayText, submitText, createdAt }]

Reducer: "dequeue_turn"
  → 移除队首，dispatch "user" action 提交下一条

Reducer: "turn_done" (改造)
  → running: false
  → 如果 turnQueue.length > 0
       dispatch "dequeue_turn" (异步，不阻塞当前 reducer)
```

### Go 后端不改动

`runGuarded` 的 `c.running` 互斥机制保持不变。队列逻辑完全在前端实现——
前端在 `turn_done` 时判断队列是否非空，非空则用下一条消息调用 `SubmitToTab`。
Go 后端看到的是"一次接一次的正常 turn 提交"，感知不到队列存在。

### UI 设计

#### 队列指示器位置

在 Composer 底部或输入框右侧增加队列状态：

```
┌──────────────────────────────────────────────┐
│  输入框                                        │
│                                               │
│  📋 队列中 (2)  [查看] [清空]                   │
└──────────────────────────────────────────────┘
```

#### 队列详情（点击"查看"展开）

```
┌──────────────────────────────────────────────┐
│  排队消息                                      │
│  ┌──────────────────────────────────────────┐ │
│  │ 1. "帮我优化这段代码..."     [✕ 取消]     │ │
│  │    12 秒前                               │ │
│  ├──────────────────────────────────────────┤ │
│  │ 2. "再确认一下测试是否通过"   [✕ 取消]     │ │
│  │    刚刚                                  │ │
│  └──────────────────────────────────────────┘ │
│  [清空全部]                                    │
└──────────────────────────────────────────────┘
```

#### 状态栏指示

当队列非空时，状态栏模式指示器旁显示：

```
Goal  ▸ 队列 2
```

### 文件改动清单

| 文件 | 改动 |
|------|------|
| `desktop/frontend/src/lib/useController.ts` | State 新增 `turnQueue`；reducer 新增 `queue_turn` / `dequeue_turn` action；改造 `send` 函数（检测 running → 入队）；改造 `turn_done`（检测队列 → 出队提交）；`cancel` 增加取消队列项能力 |
| `desktop/frontend/src/components/Composer.tsx` | 新增队列指示器 UI 组件；`submittingRef` 逻辑调整（在 running 期间保持 disabled 或改为入队） |
| `desktop/frontend/src/components/StatusBar.tsx` | 显示队列计数 |
| `desktop/frontend/src/locales/*.ts` | 新增 i18n 条目（队列相关文案） |

### 不做的事

- **不做 Go 后端队列**：Go 端保持单线程互斥模型，队列逻辑完全在前端
- **不做跨 tab 队列**：每个 tab 独立队列，切换 tab 后当前 tab 队列保持不变
- **不做持久化**：关闭应用后队列消失（与未发送的 Composer 草稿行为一致）

### 潜在风险

| 风险 | 缓解 |
|------|------|
| 队列积压过多 | 限制最大队列长度（如 10），超过时提示"队列已满" |
| turn_done + dequeue 无限循环（goal 模式每轮自动续） | dequeue 时不触发 goal 续循环检查，仅提交用户消息 |
| 队列中消息引用的文件已被删除 | 提交时不做额外校验，由模型运行时处理（与正常发送行为一致） |

## 关联

- 文件链接导航：`chat-file-link-navigation.md`
- Hope 指令设计：`hope-command-design.md`
