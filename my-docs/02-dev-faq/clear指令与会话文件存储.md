# /clear 指令与会话文件存储

<!--版本:v1.1 更新:2025-07-14 21:49:00-->

## 问题

(1) `/clear` 指令清空上下文时，物理文件会被清除或删除吗？
(2) 会话的物理文件存储在哪里？

## 回答

### 一、`/clear` 确实会删除物理文件

**是的。** `/clear` 会删除当前会话的以下三类物理文件：

| 删除项 | 说明 |
|---|---|
| 当前会话 `.jsonl` 文件 | 该轮对话的完整 JSONL 日志（transcript） |
| 分支元数据文件 `.jsonl.meta` | 会话关联的分支 ID / 父子关系 |
| checkpoint 目录 | 所有 rewind 回退检查点数据 |

**不会删除的东西：**
- 项目源代码文件 ❌ 不动
- 其他历史会话的 transcript ❌ 不动
- 配置文件（API key、MCP 设置等） ❌ 不动
- `REASONIX.md` / memory 记忆文件 ❌ 不动
- 子 agent（subagent）的 transcript ❌ 不动

### `/clear` 与 `/new` 的关键区别

| 指令 | 行为 |
|---|---|
| `/new` | 先**保存**当前 transcript 到磁盘，再开新会话 |
| `/clear` | **丢弃/删除**当前 transcript 不保存，直接开新会话 |

界面上的提示也说明了这点（`clear_confirm.go:75`）：
> *"This deletes the current transcript from local history and keeps only the system prompt."*

---

### 二、会话文件存储路径

代码定义在 `internal/config/config.go:1773-1797` 和 `internal/agent/save.go:215-221`。

#### 全局路径（CLI 模式）

```
<os.UserConfigDir>/reasonix/sessions/
```

不同操作系统的实际路径：

| 系统 | 路径 |
|---|---|
| macOS | `~/Library/Application Support/reasonix/sessions/` |
| Linux | `~/.config/reasonix/sessions/` |
| Windows | `%AppData%/reasonix/sessions/` |

#### 项目级路径（Desktop 模式）

```
<config root>/projects/<workspace-slug>/sessions/
```

其中 `workspace-slug` 是将工作区绝对路径中的 `/` `\` `:` 替换为 `-` 的结果。例如工作区 `/Users/weikejia/CODE/my-project` 对应的 slug 为 `-Users-weikejia-CODE-my-project`，完整路径为：

```
~/Library/Application Support/reasonix/projects/-Users-weikejia-CODE-my-project/sessions/
```

#### 文件名格式

```
<UTC时间戳>-<模型名>.jsonl
```

例如：`20260115-143025.000000000-deepseek-chat.jsonl`

生成逻辑（`agent/save.go:215-221`）：
```go
func NewSessionPath(dir, model string) string {
    safe := strings.NewReplacer("/", "-", "\\", "-").Replace(model)
    return filepath.Join(dir,
        fmt.Sprintf("%s-%s.jsonl",
            time.Now().UTC().Format("20060102-150405.000000000"), safe))
}
```

#### 子agent的transcript存储

子 agent 的 transcript 存放在 session 目录下的 `subagents/` 子目录中（`agent/subagent_store.go:121`）。

---

### 三、代码依据

#### `/clear` 删除逻辑

- **`internal/control/controller.go:1387-1414`** `ClearSession()` — 入口，调用 `removeSessionArtifacts()` 删除旧文件，然后创建新 session path
- **`internal/control/controller.go:1416-1434`** `removeSessionArtifacts()` — 执行删除：`.jsonl` 文件、`.jsonl.meta` 分支元数据、`checkpoint/` 目录
- **`internal/cli/clear_confirm.go:37-46`** `confirmClearContext()` — CLI 确认后调用 `ctrl.ClearSession()`

#### 存储路径

- **`internal/config/config.go:1776-1782`** `SessionDir()` — 全局 session 目录
- **`internal/config/config.go:1787-1797`** `ProjectSessionDir()` — 项目级 session 目录
- **`internal/agent/save.go:215-221`** `NewSessionPath()` — 文件名生成规则

---

### 四、已修复：会话中文文件链接无法跳转

#### 现象

在 Desktop 会话消息中，点击含中文字符的文件链接会报错：
```
stat .../clear%E6%8C%87%E4%BB%A4%E4%B8%8E%E4%BC%9A%E8%AF%9D%E6%96%87%E4%BB%B6%E5%AD%98%E5%82%A8.md:
no such file or directory
```

#### 本质区别

| 场景 | 路径处理方式 | 中文文件名 |
|---|---|---|
| **文件树点击** | IDE 原生文件系统 API，路径以原始字节/字符串传递 | ✅ 正常 |
| **会话消息中点击** | Markdown 渲染 → React onClick → Wails bridge → Go 后端 | ✅ 已修复 |

#### 根因分析

文件链接的渲染与点击流程（Desktop 模式）：

```
pathLinkify.ts: linkifyPaths()
      ↓ 将路径包装为 Markdown 链接 [text](path)
react-markdown (remark/mdast-util-to-hast)
      ↓ 解析时对非 ASCII 字符做 percent-encoding（HTML href 规范要求）
      ↓ "my-docs/.../clear指令...md" → "my-docs/.../clear%E6%8C%87...md"
Markdown.tsx: <a> href 属性
      ↓ onClick → onOpenWorkspaceFile(href)  ← href 已被编码
App.tsx: openRightDockFile(path)
      ↓
WorkspacePanel.tsx → app.ReadFile(path)
      ↓
Go: os.Stat 看到 %E6%8C%87，文件系统不认识 → stat 报错
```

**关键：编码发生在 `react-markdown` 解析 markdown 链接时**，而不是 Wails bridge。证据是文件树点击中文文件不会报错（不经过 markdown 解析，直接拿原始 UTF-8 路径）。

#### 修复方案

在 `openRightDockFile`（`App.tsx:1782`）入口处对 path 做 `decodeURIComponent` 解码。这里是**会话链接的专属入口**，解码后走 `setWorkspaceRevealRequest → selectFile(path)`，后半段与文件树点击完全合并。

修复后链路：

```
会话链接:  Markdown.tsx → openRightDockFile(path)
                           ↓ decodeURIComponent ← 洗掉 react-markdown 编码
                           ↓ setWorkspaceRevealRequest
                           ↓ useEffect
                           ↓ selectFile(path)  ←──┐ 与文件树完全相同
文件树:    ProjectTree       → selectFile(path)  ←──┘
                                                   ↓ setSelectedPath
                                                   ↓ app.ReadFile ✅
```

#### 修改文件

| 文件 | 行号 | 改动 |
|---|---|---|
| `desktop/frontend/src/App.tsx` | `openRightDockFile` | 在 `path.trim()` 后增加 `try { nextPath = decodeURIComponent(nextPath); } catch {}` 解码非 ASCII 路径 |
