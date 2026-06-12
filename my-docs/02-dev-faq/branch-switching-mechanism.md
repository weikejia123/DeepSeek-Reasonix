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
