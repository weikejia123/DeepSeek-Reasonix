# 顶部 Git 分支显示 Bug — 下拉列表跨项目未刷新

**日期**: 2025-07-14
**状态**: 已修复（改为纯文本展示）
**严重度**: 中高（显示错误，但不影响数据安全）

## 现象

在 Desktop 中新建项目、或切换到另一个项目的 tab 后：

- 顶部栏的 git 分支**名称**正确更新为新项目的当前分支
- 但点击下拉后，列表里的分支**仍然来自上一个项目**，列表未随项目切换刷新

## 根因

`desktop/frontend/src/App.tsx` 中：

```
useEffect → dockRefreshKey 变化 → app.WorkspaceChanges() → setGitBranch / setGitAvailable
```

- `gitBranch` 和 `gitAvailable` 通过 `useEffect` 随 `dockRefreshKey` 自动刷新 → **按钮文字总是正确**
- `branches`（下拉列表数据）只在**用户首次点击按钮、且 `branches.length === 0` 时**通过 `app.GitBranches()` 拉取一次，之后永远不更新
- 这意味着：第一次在项目 A 打开下拉 → 拉取项目 A 的分支列表并缓存到 state → 切换到项目 B → 按钮显示项目 B 的分支名 → 点击下拉 → 列表**仍是项目 A 的**（`branches.length > 0`，不再触发拉取）

### 数据结构层面的缺陷

```typescript
const [gitBranch, setGitBranch] = useState("");       // 随 useEffect 自动刷新 ✓
const [branches, setBranches] = useState<string[]>([]); // 只在点击时拉取一次，从不随项目切换清空 ✗
```

两个状态的生命周期不同步：`gitBranch` 跟着 `dockRefreshKey` 走，`branches` 跟着用户点击走。切换项目后没有 `setBranches([])` 的清理逻辑。

## 更深的隐患

即使修复了列表刷新，这个下拉菜单本质上也是在**非当前 workspace** 的 git repo 上做 `GitCheckout`：

- `app.GitBranches()` 和 `app.GitCheckout(b)` 的后端实现（`desktop/workspace_changes.go:217,233`）读取的是"active tab 的 workspace root"
- 下拉菜单点击切换 git 分支的行为对用户来说是隐式的、危险的——可能在不经意间改变了错误项目的工作区 git 状态

## 决议

**将顶部 git 分支显示从下拉菜单改为纯文本展示**，不再支持：

- 下拉列出所有分支
- 点击切换 git 分支

保留 `gitBranch` 状态和 `WorkspaceChanges` 的拉取逻辑不变，仅移除 `<AnchoredPopover>` 下拉菜单及相关的 `branches`/`branchMenuOpen`/`branchLoading`/`branchAnchorRef` state 和 `GitBranches`/`GitCheckout` 调用。

如果未来需要 Reasonix 分支切换功能（区别于 git 分支），将在独立的 feature 中实现。

## 涉及文件

| 文件 | 变更 |
|------|------|
| `desktop/frontend/src/App.tsx` | 删除下拉菜单 JSX、删除相关 state、删除无用 import，改为纯文本 `<span>` |
| `desktop/frontend/src/styles.css` | 调整 `.workspace-branch-name` 样式从 button 变为纯文本 |
| `desktop/frontend/src/lib/bridge.ts` | 保留 `GitBranches`/`GitCheckout` 不动（向后兼容） |
| `desktop/workspace_changes.go` | 保留 `GitBranches`/`GitCheckout` 不动（可能被其他路径使用） |
