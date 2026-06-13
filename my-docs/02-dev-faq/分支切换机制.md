# 分支切换机制

## 问题

对话分叉后切换分支，上下文是叠加的还是独立的？

## 回答

**完全独立。** 每个分支是一个独立的 `.jsonl` 会话文件。切换分支 = 加载那个分支的完整文件内容作为上下文，与当前会话完全替换，不存在"合并"或"叠加"。

### 代码依据

- **`internal/control/controller.go:1661-1694`** `SwitchBranch()` — 分支切换入口
  - 第 1672 行：`c.Branches()` 列出所有分支
  - 第 1680 行：`agent.LoadSession(match.Path)` **把目标分支的 `.jsonl` 完整加载到内存**
  - 第 1685 行：`c.executor.SetSession(loaded)` **替换当前会话，完全替换**
- **`internal/control/controller.go:1536-1591`** `forkNamed()` — 分叉创建逻辑
  - 第 1569 行：创建新的 `.jsonl` 文件（`agent.NewSessionPath()`）
  - 第 1573 行：写入 `BranchMeta{ParentID, ForkTurn, ForkMessageIndex}`
- **`internal/agent/branch.go:18-30`** — `BranchMeta` 结构体定义（`parent_id` 仅用于树形展示，不影响上下文）

### 行为示意

```
         turn 1  turn 2  turn 3 (共享历史)
              /                \
    A.jsonl                      B.jsonl (fork 于此，各自独立追加)
    - 你追加问题X                - 你追加问题Y
    - AI回答X                    - AI回答Y

/switch A → 上下文 = A.jsonl 全部内容（共享历史 + 问题X + 回答X）
/switch B → 上下文 = B.jsonl 全部内容（共享历史 + 问题Y + 回答Y）
```

两个分支互不可见，是**独立的平行宇宙**。

### 关键设计

| 方面 | 机制 |
|------|------|
| 存储 | 每个分支是一个独立的 `.jsonl` 文件（完整消息副本） |
| 切换 | `LoadSession()` 完整加载目标文件 → `SetSession()` 替换当前会话 |
| 元数据 | `BranchMeta.parent_id` 仅用于 `/tree` 命令的树形展示，不影响对话上下文 |
| 隔离 | 分支之间完全隔离，A 做的操作不会出现在 B 的上下文中 |

### 关联文件

- `internal/control/controller.go` — `SwitchBranch()` / `forkNamed()` / `ForkSession()`
- `internal/agent/branch.go` — BranchMeta 定义、BranchID 解析
- `internal/control/branches.go` — `FormatBranchTree()`（树形展示）
- `desktop/frontend/src/components/Message.tsx:277-286` — 前端分叉按钮
- `desktop/frontend/src/lib/useController.ts:988-1001` — 前端分叉处理

---

## Q8: 为什么有些 turn 的分叉按钮可用，有些被禁用？

**判定标准：该 turn 是否在 `cpBound` 映射中有记录。**

`cpBound` 是 Controller 内部的一个 `map[int]int`，记录每个 turn 开始时消息日志的长度（即"会话边界"）。分叉需要知道"从哪一刀切开"，cpBound[turn] 就是切分位置。

### 前端判定链路（`Message.tsx:178-194`）

```
actionDisabled("fork") 返回禁用原因，依次检查：

① rewindDisabled 或 actionPending（有 turn 正在运行）
   → "disabledRunning"

② checkpoint 为 null（该 turn 根本没有 checkpoint）
   → "disabledNoCheckpoint"

③ checkpoint.canConversation === false（核心判定）
   → "disabledNoBoundary" ← 最常见原因

④ 全通过
   → 空字符串 → 按钮可用
```

`canConversation` 来自 Go 后端（`desktop/app.go:920`）：

```go
CanConversation: ctrl.CheckpointHasBoundary(m.Turn),
```

即 `controller.go:1593-1598`：

```go
func (c *Controller) CheckpointHasBoundary(turn int) bool {
    _, ok := c.cpBound[turn]
    return ok
}
```

### cpBound 的生命周期

| 时机 | 操作 | 代码位置 |
|------|------|----------|
| **建立** | 每个新 turn 开始时调用 `beginCheckpoint()` | `controller.go:396` → `c.cpBound[turn] = msgIndex` |
| **恢复会话** | 重新打开已保存的会话时从磁盘 checkpoint 文件重建 | `controller.go:379-381` → `c.cpBound = c.cp.Bounds()` |
| **全部清空** | 执行 **Summarize（压缩对话）** 之后 | `controller.go:1771` → `c.cpBound = map[int]int{}` |
| **部分删除** | 执行 **Conversation Rewind（会话回退）** 后，该 turn 及之后的边界被删 | `controller.go:1501-1504` → `delete(c.cpBound, k)` for `k >= turn` |

### 导致"禁用"的常见场景

| 场景 | 原因 |
|------|------|
| 执行过 `/summarize` 压缩对话 | 所有旧 turn 的 `cpBound` 被一次性清空 → 所有旧消息的 fork 按钮都灰掉 |
| 执行过 conversation rewind 回退 | 被回退掉的 turns 的边界被删除 |
| 会话正在跑新一轮 turn | 运行中禁止任何 fork 操作 |
| 会话没有持久化存储 | 纯内存模式、某些 CLI 一次性模式 → `sessionDir == ""` |
| 重开已保存的会话中的旧 turn | turn 虽然在历史里，但 checkpoint 里没有边界记录（`fork unavailable for turn %d (resumed session)`） |

### 代码索引

- 前端判定逻辑：`desktop/frontend/src/components/Message.tsx:178-194`
- 后端 cpBound 定义：`internal/control/controller.go:114`
- 边界建立：`internal/control/controller.go:396`
- 边界恢复：`internal/control/controller.go:379-381`
- 边界查询：`internal/control/controller.go:1593-1598`
- 压缩后清空：`internal/control/controller.go:1771`
- 回退后清理：`internal/control/controller.go:1501-1504`
- 分叉守卫：`internal/control/controller.go:1536-1552`
