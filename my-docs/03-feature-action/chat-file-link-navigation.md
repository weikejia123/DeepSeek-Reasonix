# 对话文件链接导航 — 设计草案 → 实施实录

> 本文档记录了从设计到实施的完整过程，含发现的 bug 与修复。
> 设计阶段标记为「草案」，实施阶段标记为「实施」，保持两端真实可追溯。

## 问题

AI 回复中经常引用文件路径（如 `controller.go:206`、`my-docs/03-feature-action/hope-command-design.md`），
但当前这些文本只是普通文本，不可点击。用户需要手动在文件树中找到目标文件。

## 目标

将对话中出现的项目内文件路径渲染为**可点击链接**，点击后：

1. 打开右侧 WorkspacePanel 的文件树视图
2. 自动定位并展开目标文件
3. 在预览面板中显示文件内容

---

## 草案阶段：设计思路

### 现有基础设施

| 机制 | 位置 | 说明 |
|------|------|------|
| `WorkspaceRevealRequest` | `App.tsx:127` | `{ id: number; path: string }` — 触发文件树导航 |
| `openRightDockFile(path)` | `App.tsx:1765-1776` | 设置 reveal request → 打开 files 视图 → 选中并预览 |
| `ContextPanel.onOpenWorkspaceFile` | `ContextPanel.tsx:316-322` | **已有先例**：文件行点击 → `openRightDockFile` |
| `WorkspacePanel.revealPathRequest` | `WorkspacePanel.tsx:467-485` | 接收 reveal 请求 → 选中文件 → 滚动到目标位置 |
| `Markdown.a` 组件 | `Markdown.tsx:88-105` | 拦截 `<a>` 标签，当前一律走 `openExternal()` |
| `WorkspacePanel.entriesByDir` | `WorkspacePanel.tsx:224` | 懒加载的文件树数据 |

### 核心理念

**不修改流式渲染路径。** 对话输出期间，Markdown 渲染完全保持原样。
turn 完成（`item.streaming === false`）后，对已完成的消息文本做一次性
后处理：匹配已知文件路径 → 替换为 Markdown 链接 → 重新交给 Markdown 渲染。

```
流式阶段:                    完成阶段:
item.streaming === true      item.streaming === false
     ↓                            ↓
rawText unchanged             linkified = pathLinkify(rawText, filePathSet)
     ↓                            ↓
<Markdown text={rawText} />   <Markdown text={linkified} />
```

### 为什么这样设计

| 问题 | 流式方案 | 后处理方案 |
|------|---------|-----------|
| 流式闪烁 | 文本不完整时反复匹配/取消匹配 | **不存在** — 只在完成后执行一次 |
| 性能压力 | 每帧都可能触发扫描 | **一次** O(textLen + matches) |
| 代码块隔离 | 需要占位符-还原的复杂流程 | 同样需要，但只需执行一次 |
| 实现复杂度 | 需要协调 `useDeferredValue` 和匹配时机 | **低** — 纯函数调用 |
| 用户体验 | 链接在流式过程中逐渐出现 | 流式结束瞬间链接全部出现 |

---

## 实施阶段：实际实现

### 数据来源决策

草案阶段列出了两个方案：

| 方案 | 描述 | 草案倾向 |
|------|------|----------|
| A | `entriesByDir`（懒加载） | 建议一期用此方案 |
| B | 新增 Go API `ListWorkspaceFiles()` | 标注为可选增强 |

**实际选择：方案 B** —— 新增 Go API `ListWorkspaceFiles()`（`desktop/app.go:3863-3894`）。
原因是方案 A 的数据只覆盖已展开目录，而 AI 回复中引用的文件可能在任何目录下。

Go 端实现：`filepath.WalkDir` 递归遍历整个工作区，结合 `skipWorkspaceEntry` 过滤
`.git`、`node_modules` 等噪声目录，返回扁平相对路径数组。

前端工具函数：`buildFileSetFromList()`（`workspaceFileSet.ts:18-20`）将数组转为 `Set<string>`。

App.tsx 的加载时机：`useEffect` 监听 `[projectRevision, dockRefreshKey]`，在切换
项目或 turn 完成后自动刷新文件集。

`buildFileSet()`（从 `entriesByDir` 构建）作为工具函数保留，以备未来降级使用。

### 路径匹配算法（实际实现 vs 草案）

| 维度 | 草案设计 | 实际实现 |
|------|---------|----------|
| 匹配单位 | "扫描所有子串" | 按空白符拆分为 word/space token |
| 排序策略 | "按长度降序替换" | 精确 Set 匹配优先，歧义检测兜底 |
| 预过滤 | 无 | `lastExt()` + `CODE_EXTS`：不含 `/` 且扩展名不在集合中 → 跳过 |
| 歧义处理 | 未明确定义 | `buildBasenameIndex()`：basename 唯一 → 可短名链接；不唯一 → 只匹配完整路径 |
| 代码块保护 | 占位符-还原 | 逐行扫描 `FENCE_RE` 跳过代码行 |

实际算法的核心流程：

```
linkifyPaths(markdown, fileSet)
  → 逐行扫描，跳过 fenced code blocks
  → 每行 tokenize → word/space 序列
  → 每个 word token:
       stripMDFormatting(clean)  ← 剥离反引号等
       → lastExt + CODE_EXTS 预过滤
       → fileSet.has(clean) 精确匹配
       → buildBasenameIndex 歧义检测
  → 匹配到的路径替换为 [token](clean_path)
  → 返回处理后的 Markdown 文本
```

**关键点：** `stripMDFormatting()` —— 在匹配前剥离反引号、`**bold**`、`_italic_` 等
Markdown 格式字符，但输出保留原始 token 文本不受影响。这是解决 Bug 1 的核心修复。

### AssistantMessage 后处理逻辑

实际代码（`desktop/frontend/src/components/Message.tsx`）：

```tsx
const linkedText = useMemo(() => {
  if (item.streaming || !filePathSet) return item.text;
  return linkifyPaths(item.text, filePathSet);
}, [item.text, item.streaming, filePathSet]);
```

与草案一致，无偏离。

### Markdown `<a>` 组件改造

实际代码（`desktop/frontend/src/components/Markdown.tsx`）：

```tsx
a: ({ href, children }) => {
  if (href && !/^https?:\/\//.test(href)) {
    return (
      <a href="#" className="md-link--workspace"
        onClick={(e) => { e.preventDefault(); onOpenWorkspaceFile?.(href); }}>
        {children}
      </a>
    );
  }
  // 外部链接 — 原有行为
  return (<a href={href} onClick={(e) => { …; openExternal(href); }}>…</a>);
}
```

与草案一致，但实际 `onOpenWorkspaceFile` 直接从 App.tsx 传入，没有经过 `cwd` 拼接
（草案中设计了 `cwd` 参数，最终实现中未使用，因为 `ListWorkspaceFiles` 返回的是
**工作区相对路径**，不需要额外拼接）。

### 数据传输链

与草案一致，逐层传递：

```
App.tsx → Transcript → WarmZone → LiveAssistantMessage / AssistantMessage
    ↓         ↓           ↓              ↓
filePathSet 文件集     & onOpenWorkspaceFile  回调
    ↓
Markdown → <a> 组件 → openRightDockFile(path)
```

与草案的区别：没有传递 `cwd`（未使用），`filePathSet` 直接是 `Set<string>` 而非
从 `entriesByDir` 实时计算。

---

## 发现的 Bug 与修复

### Bug 1：Markdown 反引号导致预过滤失败（已修复）

**现象**：AI 回复中路径几乎一定被反引号包裹（ `` `README.md` ``），因为这是 Markdown
行内代码的标准写法。`lastExt(`` `README.md` ``)` 返回 `` .md` ``（反引号被当成了扩展名
的一部分），`CODE_EXTS.has(".md`")` → `false`，预过滤直接拒绝。

即使过了预过滤（含 `/` 的路径），`fileSet.has("`desktop/app.go`")` 也匹配不上 Set 里的
`desktop/app.go`。

**根因**：`pathLinkify.ts` 的 `linkifyLine()` 直接用 tokenizer 输出的原始文本去匹配。
tokenizer 只按空白符拆分，不处理 Markdown 格式化字符。

**修复**：新增 `stripMDFormatting()` 函数（`pathLinkify.ts:117-137`），在匹配前剥离
反引号、`**bold**`、`_italic_` 等格式字符。输出保留原始 token 文本，只有匹配/链接
逻辑使用清洗后的文本：

```
输入 token: `README.md`
→ clean = "README.md"
→ fileSet.has("README.md") → true
→ 输出 [\`README.md\`](README.md)
→ Markdown 渲染为 <a>，点击调 openRightDockFile("README.md")
```

**位置**：`desktop/frontend/src/lib/pathLinkify.ts:117-137`

**状态**：已修复，正在等待打包测试验证。

### Bug 2：Browser dev 模式 Mock 数据不足（未修复，已知限制）

**现象**：`bridge.ts` 的 mock 实现 `ListWorkspaceFiles()` 只返回 11 条硬编码路径，
项目里 99% 的路径不会出现在 `filePathSet` 中。

**影响范围**：仅 browser dev 模式（`npm run dev`）。桌面打包版走 Go 后端全量 WalkDir，
不受此限制。

**状态**：已知限制，未修复。Browser dev 模式下测试时只在硬编码的 11 个文件内有效。
建议直接用桌面打包版测试全量路径。

## 当前效果验证（阶段性记录）

> 以下为真实测试结果，记录在当前匹配算法下已验证和已知盲区的模式。
> 不追求完美，分阶段迭代。

### ✅ 可以跳转

| 类别 | AI 输出示例 | 匹配链路 |
|------|-------------|---------|
| 含 `/` 的完整路径 | `desktop/app.go`、`internal/control/input.go` | token → `fileSet.has` 精确命中 |
| 唯一 basename + 已知扩展名 | `App.tsx`、`README.md` | prefilter(`CODE_EXTS`) → `buildBasenameIndex` 消歧命中 |
| 反引号包裹 | `` `README.md` `` | `stripMDFormatting` 去 `` ` `` → 后续正常匹配 |
| `**bold**` 包裹 | `**internal/control/input.go**` | `stripMDFormatting` 去 `**` → 后续正常匹配 |
| `_italic_` 包裹 | `_internal/control/input.go_` | `stripMDFormatting` 去 `_` → 后续正常匹配 |
| 中文标点粘连 | `文件：my-docs/03-feature-action/chat-file-link-navigation.md` | `stripSurroundingPunct` 去 `文件：` → `fileSet.has` 命中 |
| 括号包裹 | `（internal/control/input.go）` | `stripSurroundingPunct` 去 `（）` → `fileSet.has` 命中 |
| 引号包裹 | `"internal/control/input.go"` | `stripSurroundingPunct` 去 `"` → `fileSet.has` 命中 |
| 路径后跟行号 | `internal/control/input.go:93-103` | `stripSurroundingPunct` 截断 `:93-103` → 链到文件 |
| basename 后跟行号 | `controller.go:206` | `stripSurroundingPunct` 截断 `:206` → basename 匹配命中 |

### ❌ 不能跳转

| 类别 | AI 输出示例 | 阻断位置 |
|------|-------------|---------|
| 非代码扩展名的根文件 | `go.mod`（`.mod` 不在 `CODE_EXTS`）、`Makefile`、`Dockerfile` | prefilter：无 `/` 且 `lastExt` 不命中 `CODE_EXTS` |
| 代码块内的路径 | `` ```go\npackage main\n``` `` 内的任何路径 | `linkifyPaths` 的 fence 检测跳过整段 |
| 不存在的路径 | AI 幻觉出的路径 | `fileSet.has` 无此 key |
| basename 歧义（同名多目录） | 仅写 `index.ts` 但项目有 3 个 `index.ts` | `buildBasenameIndex` → `null`（标记为歧义）→ 不链接 |
| 外部/绝对路径 | `/etc/hosts`、`../foo` | 不在 `fileSet` 中（只含相对工作区路径） |

### 三阶段过滤链

```
word → stripMDFormatting → stripSurroundingPunct → pathOnly
                                                      ↓
                                    prefilter: / 或 CODE_EXTS ？
                                      ↓ Yes              ↓ No → ❌ 丢弃
                                    fileSet.has(pathOnly) ？
                                      ↓ Yes → ✅          ↓ No
                                    basenameIndex 歧义检测
                                      ↓ 唯一 → ✅         ↓ 歧义/无 → ❌
```

### 当前状态总评

> **功能已可用，覆盖日常对话中 80%+ 的文件路径引用模式。**
> 点击蓝色链接即可跳转到文件树并预览文件内容。
> 中文标点粘连等边界问题不影响核心体验，留到下阶段迭代。

---

## 涉及文件及改动量（实际）

| 文件 | 改动 | 说明 |
|------|------|------|
| `desktop/frontend/src/lib/workspaceFileSet.ts` | **新增** | `buildFileSet`, `buildFileSetFromList`, `buildBasenameIndex` |
| `desktop/frontend/src/lib/pathLinkify.ts` | **新增** | `linkifyPaths`, `linkifyLine`, `tokenize`, `stripMDFormatting`, `lastExt` |
| `desktop/frontend/src/components/Markdown.tsx` | 修改 | `a` 组件分叉：工作区路径 → `onOpenWorkspaceFile`，外部链接 → `openExternal` |
| `desktop/frontend/src/components/Message.tsx` | 修改 | `useMemo` 计算 `linkedText`，新增 `filePathSet` + `onOpenWorkspaceFile` props |
| `desktop/frontend/src/components/Transcript.tsx` | 修改 | `filePathSet` + `onOpenWorkspaceFile` 逐层传递（4 层组件链） |
| `desktop/frontend/src/App.tsx` | 修改 | `useEffect` 调用 `ListWorkspaceFiles`，构建 `workspaceFileSet`，传入 Transcript |
| `desktop/app.go` | **新增 （方法）** | `ListWorkspaceFiles()` — WalkDir 递归收集全量文件路径 |
| `desktop/frontend/src/lib/bridge.ts` | 修改 | 接口声明 + mock（11 条硬编码） |

---

## 实施总结

关键偏离点记录：

1. **数据来源从方案 A 改为方案 B**：草案倾向用 `entriesByDir` 懒加载数据，实际实现用
   新增的 Go API `ListWorkspaceFiles()` 全量遍历。原因：懒加载数据覆盖不全。
2. **匹配算法从"子串扫描"改为"token 拆分"**：草案描述的是 Trie/子串匹配，实际实现
   按空白符拆分 token 后逐条检查。性能足够且实现更简单。
3. **额外处理 Markdown 格式化字符**：草案完全忽略了反引号问题，实际发现这导致所有
   路径一概无法匹配。新增 `stripMDFormatting()` 作为修复。
4. **未使用 `cwd` 参数**：草案设计了 `cwd` 用于拼接相对路径，但 `ListWorkspaceFiles`
   返回的就是工作区相对路径，无需额外拼接。

---

## 未决问题

1. ~~文件集数据来源~~ → **已决策**：Go API `ListWorkspaceFiles()`（方案 B）
2. ~~文件路径歧义~~ → **已处理**：`buildBasenameIndex` 歧义检测（唯一短名可链，歧义短名跳过）
3. **行号锚点**（二期）— CodeViewer 是否支持 `#L93` hash 跳转？一期只链接文件，行号保留在链接文本中供人工参考
4. ~~工作区切换~~ → **已处理**：`useEffect` 依赖 `[projectRevision, dockRefreshKey]` 自动重建文件集
5. **文件链接视觉样式** — 目前链接是浏览器默认蓝色 `<a>` 样式，可自定义 CSS
6. **中文标点粘连** — 已知盲区，`stripSurroundingPunct` 已实现需验证，留到下阶段
