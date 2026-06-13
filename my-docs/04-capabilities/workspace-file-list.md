# ListWorkspaceFiles() — 全量工作区文件枚举

**能力类型**: 基础能力  
**提供方**: Go 后端 (`desktop/app.go`) → 前端 Bridge (`bridge.ts`)  
**前端入口**: `app.ListWorkspaceFiles(): Promise<string[]>`  
**代码位置**: `desktop/app.go:3867-3895`

## 能力概述

`ListWorkspaceFiles()` 递归遍历当前工作区（workspace root），返回所有非目录文件的**相对路径**扁平数组。已自动跳过 `.git`、`node_modules`、构建产物等噪声目录。

```
输入: 无参数（自动使用当前活动 workspace root）
输出: Promise<string[]>  — 扁平路径数组，如 ["README.md", "desktop/app.go", "internal/agent/agent.go", ...]
```

## 完整数据流

```
前端 App.tsx:853
  app.ListWorkspaceFiles()
        │
        ▼ Wails Bridge
desktop/app.go:3867  ListWorkspaceFiles()
  │
  ├─ activeWorkspaceBase()                          // app.go:3760
  │    → 取当前 workspace root，支持相对路径
  │    → 返回绝对路径
  │
  ├─ filepath.WalkDir(base, walkFn)                 // 递归遍历
  │    │
  │    ├─ skipWorkspaceEntry(rel, name, isDir)      // app.go:3753
  │    │    ├─ workspaceNoiseNames[name]            // 文件名黑名单
  │    │    └─ workspaceNoiseDirs[rel]              // 目录路径黑名单
  │    │
  │    └─ 非目录 → files = append(files, rel)       // 转斜杠、相对路径
  │
  └─ return files  — []string
        │
        ▼
前端 App.tsx:855
  buildFileSetFromList(files)                       // workspaceFileSet.ts:18
    → new Set(files)                                // O(n) Set 构造
    → setWorkspaceFileSet(fileSet)                  // React state
        │
        ▼
  workspaceFileSet 通过 props 下发给各组件
    → 文件链接化 (pathLinkify)
    → 扩展名枚举
    → 其他基于文件列表的功能
```

## 噪声过滤规则

### 文件名黑名单 (`workspaceNoiseNames`, app.go:3677-3686)

以下文件名（无论路径层级）会被跳过：

| 名称 | 来源 |
|------|------|
| `.codex` | Codex 本地状态 |
| `.codegraph` | 代码图谱索引 |
| `.DS_Store` | macOS Finder 元数据 |
| `.git` | Git 仓库 |
| `.npm` | npm 缓存 |
| `.pnpm-store` | pnpm 存储 |
| `node_modules` | Node 依赖 |
| `Thumbs.db` | Windows 缩略图缓存 |

遇到这些名称的目录时，`filepath.SkipDir` 会跳过整个子树。

### 目录路径黑名单 (`workspaceNoiseDirs`, app.go:3688-3699)

以下目录路径（需相对路径完全匹配）的子树会被跳过：

| 路径 | 来源 |
|------|------|
| `bin` | Go 构建产物 |
| `desktop/build` | Wails 构建输出 |
| `desktop/frontend/dist` | 前端打包产物 |
| `desktop/frontend/wailsjs` | Wails 生成代码 |
| `dist` | 通用构建产物 |
| `npm/.stage` | npm 暂存区 |
| `site/.astro` | Astro 缓存 |
| `site/dist` | 站点构建产物 |
| `stage` | 暂存目录 |
| `tmp` | 临时文件 |

## 触发时机

前端在以下条件变化时重新拉取（`App.tsx:860`）：

```typescript
useEffect(() => {
  app.ListWorkspaceFiles().then(files => setWorkspaceFileSet(buildFileSetFromList(files)));
}, [projectRevision, dockRefreshKey]);
```

- `projectRevision` 变化 — 文件系统变更事件（git checkout、文件增删）
- `dockRefreshKey` 变化 — 用户手动刷新

## 前端消费点

### 1. 文件链接化 (`pathLinkify`)

`workspaceFileSet` 传给消息渲染组件，用于识别聊天文本中的文件路径并转为可点击链接。`buildBasenameIndex()`（`workspaceFileSet.ts:25`）进一步构建 basename→path 索引，支持通过文件名（如 `agent.go`）直接链接到唯一文件。

### 2. 扩展名枚举

从 `workspaceFileSet`（内存中的 `Set<string>`）提取所有唯一扩展名，零 I/O 开销：

```typescript
function buildExtSet(fileSet: Set<string>): Set<string> {
  const exts = new Set<string>();
  for (const path of fileSet) {
    const dot = path.lastIndexOf(".");
    if (dot >= 0) exts.add(path.slice(dot).toLowerCase());
  }
  return exts;
}
```

时间复杂度 O(n)，50 万文件约 20ms。可用于替代硬编码的 `CODE_EXTS`，让文件链接化自动覆盖项目中的所有扩展名类型。

### 3. 文件树 UI

`workspaceFileSet` 可作为项目文件树的扁平数据源，用于构建侧边栏的文件浏览器。

## 使用前提

| 条件 | 说明 |
|------|------|
| 环境 | Desktop 端（Wails）运行 |
| 依赖 | `a.activeWorkspaceBase()` 需要当前存在有效的 workspace root |
| 权限 | 需要文件系统读权限遍历 workspace 目录 |

## 与前端的 bridge mock 差异

`bridge.ts:1971` 的 mock 实现返回一组硬编码文件路径（仅用于开发/测试环境，非 Wails 运行时）。生产环境走 Wails IPC 调用 Go 端的 `ListWorkspaceFiles()`。

## 关联能力

此能力是以下功能的**基础依赖**：

- `buildFileSetFromList()` — `workspaceFileSet.ts:18`
- `buildBasenameIndex()` — `workspaceFileSet.ts:25`，文件链接化核心
- `buildExtSet()` — 扩展名枚举（计划替代硬编码 `CODE_EXTS`）
- `pathLinkify` — 消息中的路径高亮与跳转
- 文件树 UI 组件 — 侧边栏项目文件浏览

## 代码索引

| 文件 | 行号 | 内容 |
|------|------|------|
| `desktop/app.go` | 3867-3895 | `ListWorkspaceFiles()` 主实现 |
| `desktop/app.go` | 3753-3759 | `skipWorkspaceEntry()` 噪声过滤 |
| `desktop/app.go` | 3760-3768 | `activeWorkspaceBase()` 路径解析 |
| `desktop/app.go` | 3675-3699 | 噪声名称/目录黑名单 |
| `desktop/frontend/src/App.tsx` | 848-860 | 前端触发点与 state 管理 |
| `desktop/frontend/src/lib/workspaceFileSet.ts` | 1-39 | `buildFileSet` / `buildFileSetFromList` / `buildBasenameIndex` |
| `desktop/frontend/src/lib/bridge.ts` | 171 | `ListWorkspaceFiles()` 接口声明 |
| `desktop/frontend/src/lib/bridge.ts` | 1971 | Mock 实现 |
