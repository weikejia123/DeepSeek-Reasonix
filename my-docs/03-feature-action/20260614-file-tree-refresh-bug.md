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

### 根本原因总结

经过完整的代码审阅，核心问题**不是**刷新事件未触发，而是工作区文件树的刷新机制有天然的设计局限：

1. **`ListWorkspaceFiles()` 仅在 `projectRevision` 或 `dockRefreshKey` 变化时重载**（`App.tsx:860`）—— 这些都是基于事件的被动刷新
2. **没有主动的文件系统监听（fsnotify/inotify）** —— 文件变更不主动推送
3. **WorkspacePanel 的 `refreshKey` 机制只重载已打开的目录**（`WorkspacePanel.tsx:532`）—— 如果新文件创建在**未展开**的目录下，目录树不会显示
4. **特定场景下 Go 后端没有调用 `emitProjectTreeChanged`** —— 当 `ToolResult` 事件不改变活动状态时（`thinking`→`thinking`），不触发树刷新

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

### 方案 A：在 ToolResult 时主动触发（轻量）

在 `tabEventSink.emit()` 中，对 writer 工具（`write_file`、`edit_file`、`multi_edit`、`delete_range`、`delete_symbol`）的 `ToolResult`，无条件调用 `emitProjectTreeChanged()`：

```go
// tabs.go:237 附近
if e.Kind == event.ToolResult && isFileWriter(e.Tool.Name) && e.Tool.Err == "" {
    s.app.emitProjectTreeChanged()
}
```

优点：改动最小，仅在文件变更时刷新
缺点：高频文件操作可能触发多次刷新

### 方案 B：添加文件系统监听（完整）

在 Go 端使用 `fsnotify` 监听工作区变更，文件变更时自动 `emitProjectTreeChanged`。

优点：彻底解决，不依赖工具事件
缺点：改动较大，需要处理大量 OS 文件变更噪声

### 方案 C：WorkspacePanel 收到 refreshKey 时强制重置（防御性）

在 WorkspacePanel 的 `refreshKey` useEffect 中，不仅重载已打开目录，也**重置根目录**：

```typescript
// WorkspacePanel.tsx:526
useEffect(() => {
    if (!open || !refreshKey) return;
    setEntriesByDir({});   // 清空缓存 ← 新增
    loadDir("");            // 重载根 ← 新增
    openDirsRef.current.forEach((dir) => void loadDir(dir));
}, [...]);
```

优点：确保每次 refreshKey 变化都全量刷新
缺点：刷新所有展开的目录可能较重

### 推荐组合：A + C

方案 A 确保文件变更立即推送，方案 C 确保 UI 侧可靠刷新。
