# 项目结构导览

<!--
版本: v1.0
创建: 2026-06-14 00:10:00
更新: 2026-06-14 00:10:00
-->

## 顶层目录

```
reasonix/
  cmd/           → 入口程序（CLI）
  internal/      → 46 个 Go 包（核心实现）
  desktop/       → Desktop 前端（Wails: Go + React/TypeScript）
  my-docs/       → 项目文档（你正在读的）
```

## internal/ 包分类地图

### 🔵 核心循环（3 个包）

| 包 | 职责 | 关键文件 |
|----|------|---------|
| `agent/` | Agent.Run() 循环、stream、executeBatch、session | agent.go (700+行) |
| `control/` | Controller：submit 路由、Compose、Approval、Goal/Plan 管理 | controller.go (2800+行) |
| `boot/` | Build() — 从零构建 Controller + Host | boot.go (1300+行) |

### 🟢 工具系统（3 个包）

| 包 | 职责 |
|----|------|
| `tool/` | Tool 接口、Registry、Builtin 全局集 |
| `tool/builtin/` | 16 个内置工具（bash, read_file, edit_file, grep, ...） |
| `command/` | 自定义命令：加载 .md 模板、变量替换 |

### 🟡 扩展系统（4 个包）

| 包 | 职责 |
|----|------|
| `skill/` | Skill Store：发现、解析、List、Read |
| `plugin/` | MCP 客户端：传输层、LazyTool 代理、三层启动 |
| `hook/` | Hook 引擎：10 事件点、阻塞/非阻塞执行 |
| `installsource/` | install_source 工具：Skill/MCP 安装规划与执行 |

### 🟠 LLM 集成（2 个包）

| 包 | 职责 |
|----|------|
| `provider/` | Provider 接口、Request/Response 结构、ToolSchema |
| `provider/openai/` | OpenAI 兼容实现（DeepSeek 走此路径） |

### 🔴 基础设施（6 个包）

| 包 | 职责 |
|----|------|
| `config/` | reasonix.toml 加载、合并、运行时编辑 |
| `event/` | Event 结构、19 种 Kind、Sink 接口 |
| `session/` → `agent/session.go` | Session 结构、消息管理 |
| `memory/` | Memory Store：层级加载、remember/forget/recall 工具 |
| `evidence/` | Evidence Ledger、complete_step 验证 |
| `diff/` | Diff 渲染引擎 |

### ⚪ 前端通道（4 个包）

| 包 | 职责 |
|----|------|
| `cli/` | TUI（bubbletea + lipgloss） |
| `desktop/` | Desktop（Wails: app.go + frontend/） |
| `serve/` | HTTP/SSE 服务器 |
| `acp/` | Agent Client Protocol |

### 🟣 辅助系统（10+ 个包）

| 包 | 职责 |
|----|------|
| `i18n/` | 国际化消息系统 |
| `history/` | 会话历史搜索 |
| `codegraph/` | 内置 MCP：符号搜索/调用图 |
| `lsp/` | 内置 MCP：LSP 定义/诊断 |
| `builtinmcp/` | 内置 MCP 服务器注册 |
| `checkpoint/` | 每 Turn 快照、Rewind |
| `sandbox/` | macOS Seatbelt bash 沙箱 |
| `jobs/` | 后台任务管理 |
| `billing/` | 计费/余额 |
| `doctor/` | 诊断工具 |
| `inspect/` | 能力检视 |
| `outputstyle/` | 输出样式 |
| `notify/` | 桌面通知 |
| `proc/` | 进程管理 |
| `fileref/` | 文件引用解析 |
| `frontmatter/` | YAML-frontmatter 解析 |
| `netclient/` | HTTP 代理客户端 |
| `sysproxy/` | 系统代理 |
| `mcpdiag/` | MCP 认证诊断 |
| `retrieval/` | 检索增强 |
| `permission/` | 权限管理 |
| `instruction/` | 结构化指令验证 |
| `nilutil/` | nil 检查工具 |
| `fileutil/` | 文件操作工具 |

## Desktop 前端结构

```
desktop/
  app.go                  → Go 后端：Bridge API
  tab*.go                 → Tab 管理
  frontend/
    src/
      App.tsx             → 顶层组件
      lib/
        useController.ts  → 状态管理（Reducer）
        bridge.ts         → Go↔React Bridge 接口
        types.ts          → TypeScript 类型定义
        i18n.tsx          → 国际化
        pathLinkify.ts    → 文件路径链接化
        workspaceFileSet.ts → 工作区文件集合
      components/
        Composer.tsx      → 输入框
        Transcript.tsx    → 消息列表
        StatusBar.tsx     → 底部状态栏
        ToolCard.tsx      → 工具调用卡片
        ... (40+ 组件)
```

## 数据流总览

```
用户输入
  │
  ├─ TUI: chat_tui key event → submit
  ├─ Desktop: Composer.submit() → handleSend() → Bridge → Go Submit()
  └─ HTTP: POST /submit
        │
        ▼
Controller.submit()   ← 命令/Goal/Skill/普通输入 路由
        │
        ▼
Controller.Compose()  ← 注入 <active-goal>/<memory-update>/plan marker
        │
        ▼
Agent.Run()           ← 核心循环 (stream → executeBatch → loop)
        │
        ▼
sink.Emit(Events)     ← 19 种事件流
        │
        ▼
前端 Reducer           ← 事件→Action→State 转换 → React 渲染
```
