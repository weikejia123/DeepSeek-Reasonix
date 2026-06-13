# Desktop UI — 数据模型：Trash / Changes / Referenced Files

<!--
版本: v1.0
创建: 2026-06-14 01:15:00
更新: 2026-06-14 01:15:00
-->

**归流自**: `02-dev-faq/Desktop-UI面板FAQ.md` Q1-Q3

## 架构

Desktop 右侧面板展示三种运行时数据，各自有独立的数据源：

```
┌─ ProjectTree ──── 项目文件树
├─ Session Changes ─ AI 写入过的文件（来自 Checkpoints）
├─ Referenced Files ─ AI 读取过的文件（来自 telemetry）
└─ Trash ─────────── 已删除会话（.trash/ 目录）
```

## 一、Session Changes（本会话变更）

**数据来源**: `ctrl.Checkpoints()` → AI 工具调用记录

```
AI 调用 write_file / edit_file / multi_edit / delete_range / delete_symbol
  → Controller 记录文件路径到 checkpoint
  → Checkpoints() 返回 []CheckpointMeta { Files: ["desktop/app.go", ...] }
  → 前端渲染文件路径 + 操作时间
```

| 特性 | 说明 |
|------|------|
| 数据来源 | AI 操作日志（工具调用记录），非 `git diff` |
| 展示内容 | 文件路径列表 + 操作时间 |
| 是否可回退 | 否（仅展示，回退用 `/rewind`） |
| Git 状态 | 可附带 `GitStatus` 字段（如 modified） |

## 二、Referenced Files（引用的文件 / 依赖文件）

**数据来源**: `telemetry.ReadFiles` → AI 文件读取记录

```
AI 调用 read_file / grep / glob / lsp_*
  → telemetry 记录 { path, turn, lineRange }
  → 前端渲染文件路径 + 读取回合 + 读取范围
```

| 特性 | 说明 |
|------|------|
| 数据来源 | 文件读取记录 |
| "依赖文件" 由来 | UI 占位符 `"Filter dependency files…"` 的别名 |
| 展示内容 | 路径 + 回合编号 + 读取范围（如 L10-50） |

### 两个面板对比

| 维度 | Session Changes | Referenced Files |
|------|:---:|:---:|
| 含义 | AI **写入过**的文件 | AI **读取过**的文件 |
| 数据源 | `ctrl.Checkpoints()` | `telemetry.ReadFiles` |
| 比喻 | "AI 的产出物" | "AI 的输入材料" |

## 三、Trash（回收站）

**数据来源**: `desktop/sessions.go` → `.trash/` 目录

### 三级删除机制

| 操作 | 实现 | 可恢复 |
|------|------|:---:|
| **Trash Topic** | `os.Rename` 移动到 `.trash/{key}/` | ✅ |
| **Remove Project** | 仅从 `desktop-projects.json` 移除此项目记录 | ✅ |
| **Purge** | `os.RemoveAll(itemDir)` 物理删除 | ❌ |

### 文件结构

```
reasonix/sessions/
  .trash/
    {uuid}/
      _trashed.json   ← 元数据（原名、删除时间）
      session.jsonl    ← 原会话文件
      session.meta
      session.ckpt
```

恢复操作：从 `.trash/` 移回 `sessions/` 目录。

## 四、代码索引

| 文件 | 内容 |
|------|------|
| `desktop/tabs.go:2976-2988` | Checkpoints() 数据源 |
| `desktop/tabs.go:2964` | telemetry.ReadFiles |
| `desktop/sessions.go:122` | trashSessionArtifacts() |
| `desktop/sessions.go:236` | purgeTrashedSessionFile() |
| `desktop/app.go:1410` | RemoveWorkspace() |
| `desktop/frontend/src/components/WorkspacePanel.tsx:680` | "Filter dependency files" |
