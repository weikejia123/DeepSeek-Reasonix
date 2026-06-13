# Memory System — 项目知识持久化

<!--
版本: v1.0
创建: 2026-06-13 23:46:00
更新: 2026-06-13 23:46:00
-->

**能力类型**: 系统架构文档
**涉及包**: `internal/memory/`、`internal/control/controller.go`

## 架构

```
记忆源（文件系统）
  ├─ REASONIX.md (项目级，提交到 git)
  ├─ REASONIX.local.md (本地，git-ignored)
  ├─ ~/.config/reasonix/REASONIX.md (用户全局)
  └─ REASONIX.md in ancestor dirs (祖先目录)
        │
Store (memory.New)
  ├─ Load() all files → []Memory
  └─ 注入到 system prefix（缓存稳定部分）
        │
运行时
  ├─ QuickAdd(memory, note)    → 追加记忆
  ├─ QueueMemory(note)          → 本 turn riding tail
  └─ Session.Add(memory-update) → Compose 注入
        │
工具
  ├─ memory → 搜索/列出/读取
  ├─ remember → 保存/更新
  └─ forget → 删除
```

## 一、Memory 结构

```go
type Memory struct {
    Name        string  // kebab-case slug
    Title       string  // 人类可读标签
    Description string  // 一行摘要
    Type        Type    // user | feedback | project | reference
    Body        string  // Markdown 内容
}
```

## 二、四种类型

| Type | 用途 | 示例 |
|------|------|------|
| `user` | 用户偏好 | "使用 pnpm" |
| `feedback` | 工作指导 | "FAQ 存入 02-dev-faq" |
| `project` | 项目事实 | "端口 8080" |
| `reference` | 外部资源 | "API 文档 URL" |

## 三、加载层级

```
项目 .reasonix/memory/ (如果存在)
  └─ *.md 文件 → Memory 对象
全局 ~/.reasonix/memory/
  └─ *.md 文件 → Memory 对象
```

## 四、AGENTS.md / REASONIX.md

项目根和用户 home 目录的 `AGENTS.md` / `REASONIX.md` 作为**系统提示词的一部分**直接注入，每次 session 加载。支持：

- `@path` 导入其他文件
- `#<note>` 快捷添加（chat 中输入 `# 保存这个事实`）

## 五、Runtime 记忆更新

```
remember("prefers-pnpm", {body, type, description})
  → 写入 .reasonix/memory/prefers-pnpm.md
  → QueueMemory → 下 turn Compose 注入 <memory-update>
  → 下 session 自然进入 system prefix
```

## 六、代码索引

| 文件 | 内容 |
|------|------|
| `internal/memory/store.go` | Store、Memory 结构、加载 |
| `internal/memory/remember.go` | remember 工具 |
| `internal/memory/forget.go` | forget 工具 |
| `internal/memory/recall.go` | memory 搜索工具 |
| `internal/control/input.go:109-120` | Compose 注入 <memory-update> |
