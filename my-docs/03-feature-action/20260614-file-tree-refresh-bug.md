# Bug 分析：Turn 完成后新增文件不显示在文件树中

<!--
版本: v1.0
创建: 2026-06-14 01:40:00
更新: 2026-06-14 01:40:00
-->

## 现象

Turn 完成后 AI 创建了新文件，文件树不显示，点击提示错误。需切换到其他菜单再切回文件树才能看到新文件。

## 刷新链路分析

```
AI write_file 完成
  → 发射 ToolResult 事件
  → TurnDone 事件

TurnDone 到达前端:
  ├─ App.tsx:811   setDockRefreshKey(v => v+1)
  │     ├─ WorkspacePanel.refreshKey 变化
  │     └─ useEffect (WorkspacePanel.tsx:526) → loadDir() 重载已打开目录
  │
  └─ tabs.go:232   emitProjectTreeChanged()
        ├─ Wails event "project-tree:changed"
        └─ App.tsx:1519  setProjectRevision(v => v+1)
              └─ ListWorkspaceFiles() 重载 (App.tsx:860)
```

## 根因：`emitProjectTreeChanged` 被活动状态节流抑制

关键代码 `desktop/tabs.go:231-234`：

```go
if s.app != nil {
    if status, update := topicActivityStatusFromEvent(e); update &&
       s.app.setTabActivityStatus(s.tabID, status) {    // ← 仅在状态变化时返回 true
        s.app.emitProjectTreeChanged()
    }
}
```

`setTabActivityStatus` 仅在新旧状态不同的返回 `true`（`tabs.go:2576-2578`）。

### 正常 Turn 的状态流转

```
TurnStarted → status = thinking → 旧 "" → 新 thinking → emit ✅
ToolDispatch → status = thinking → 旧 thinking → 新 thinking → 不 emit
ToolResult   → status = thinking → 不变 → 不 emit
TurnDone    → status = ""       → 旧 thinking → 新 "" → emit ✅
```

**在正常 Turn 中，TurnDone 会触发 `emitProjectTreeChanged`。** ✅

### 但有漏网场景

**场景：连续快速 Turn（Goal 模式或用户快速发 steer）**

```
Turn 1: TurnStarted → thinking → TurnDone → ""  (emit ✅)
  ↓ 极短间隙，下一个 TurnStarted 到达时，前端的 setProjectRevision 可能还未被 React 处理
Turn 2: TurnStarted → thinking → TurnDone → ""  (emit ✅)
```

这种场景下 React 可能会**批处理**两次 `setProjectRevision`，但由于值从 1→2 实际上只渲染一次，`ListWorkspaceFiles` 的 useEffect 只在最终渲染时触发（依赖 `projectRevision`），所以不是此问题的根因。

## 根本原因

**纯前端问题。** `WorkspacePanel` 的 `refreshKey` 回调没有清空目录缓存。

对比两个刷新路径：

```
open=false→true (切换面板回来)      refreshKey 变化 (TurnDone)
══════════════════════════════      ══════════════════════════
setEntriesByDir({})   ← 清空缓存   ❌ 缺失 — 不清缓存
loadDir("")           ← 重载根目录  loadDir(已展开目录) ← 只刷新已展开的
```

`refreshKey` 变化时不清空 `entriesByDir` 缓存，`loadDir` 用 `setEntriesByDir((prev) => ({ ...prev, [dir]: entries }))` 覆盖回去——但未展开的目录根本没被重载，其旧缓存直接保留。新文件如果在这些目录下，不可见。

**Go 后端不需要改动** — TurnDone 时 `emitProjectTreeChanged` 正常触发，`dockRefreshKey` 正常变化，前端数据源也是最新的。问题仅在前端缓存清理。

## 修复

`WorkspacePanel.tsx:532` — 加一行 `setEntriesByDir({})`：

```diff
  openDirsRef.current.forEach((dir) => void loadDir(dir));
+ setEntriesByDir({});
```

改动 1 行。Go 后端零改动。

## 影响范围

| 场景 | 是否重现 |
|------|:---:|
| 在已展开目录中创建文件 | ⚠️ 偶尔不刷新 |
| 在新目录中创建文件 | ❌ 新目录未展开，不可见 |
| 连续多 Turn 快速执行 | ⚠️ 可能的渲染延迟 |
| Goal 模式大量文件操作 | ⚠️ 高频 |

## 用户绕过方式

切换其他面板再切回（强制 `open` prop 从 `false→true`，触发 WorkspacePanel 的完全重置，`useEffect` line 344-360）。

## 推荐修复

**纯前端 1 行改动。** `WorkspacePanel.tsx:532` 加 `setEntriesByDir({})`，已在本文档更新同时应用。
