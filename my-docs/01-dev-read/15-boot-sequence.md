# Boot Sequence — 从零到就绪

<!--
版本: v1.0
创建: 2026-06-13 23:35:00
更新: 2026-06-13 23:35:00
-->

**能力类型**: 系统架构文档  
**代码位置**: `internal/boot/boot.go:Build()`

## 完整启动流程

```
Build(opts)
  │
  ├─ 1. 配置加载
  │     ├─ config.Load(projectRoot)        → cfg
  │     ├─ resolve providers               → prov + price + ctxWin
  │     └─ resolve maxSteps, temperature
  │
  ├─ 2. Session 初始化
  │     ├─ session.Load() 或 session.New()
  │     └─ buildSession() → Session{message history}
  │
  ├─ 3. 内存系统
  │     ├─ memory.NewStore(root)           → mem.Store
  │     └─ 层级加载: project > user > global memory files
  │
  ├─ 4. 技能系统
  │     ├─ skill.New(opts)                 → skillStore
  │     ├─ skillStore.List()               → skills[]
  │     └─ skill.ApplyIndex(sysPrompt, skills) → # Skills 索引注入
  │
  ├─ 5. 自定义命令
  │     └─ command.Load(dirs...)           → cmds[]
  │
  ├─ 6. Hook 系统
  │     ├─ hook.LoadAll(root)              → hooks
  │     └─ hook.NewRunner(hooks)
  │
  ├─ 7. 工具 Registry 构建
  │     ├─ NewRegistry()
  │     ├─ 注册内置工具 (16 builtin)
  │     ├─ 注册 Agent 工具 (task, ask)
  │     ├─ 注册子系统工具 (history, memory, forget, remember)
  │     ├─ 注册命令工具 (slash_command)
  │     ├─ 安装源工具 (install_source)
  │     │
  │     └─ 按需激活 (token economy):
  │           ├─ codegraph → codegraph_* 工具
  │           ├─ lsp       → lsp_* 工具
  │           ├─ web_fetch 连接
  │           ├─ skill 工具 (run_skill, read_skill, install_skill)
  │           └─ skill 子代理包装 (explore, research, review, security_review)
  │
  ├─ 8. MCP 插件启动
  │     ├─ 按 tier 分类: eager / background / lazy
  │     ├─ eager 层: 同步启动 (阻塞)
  │     ├─ background 层: 异步启动 (gatherer)
  │     └─ lazy 层: 注册 mcp__*__connect 代理
  │
  ├─ 9. 系统前缀构建
  │     ├─ 基础提示词 + memory 文件内容
  │     ├─ skill index (name + description only)
  │     └─ 缓存稳定 — 整个 prefix 在 session 期间不变
  │
  ├─ 10. Controller 构建
  │     ├─ NewController(Options{...})
  │     │     ├─ Session, Registry, Provider
  │     │     ├─ Skills, Memory, Hooks
  │     │     └─ Config: contextWindow, compact, autoPlan
  │     │
  │     └─ controller.SetSession(session)
  │
  └─ 返回: Controller + Host (插件管理) + 诊断信息
```

---

## 关键时机

| 阶段 | 同步/异步 | 说明 |
|------|:---:|------|
| 配置解析 | 同步 | 阻塞直到 TOML 解析完成 |
| Session 加载 | 同步 | 恢复历史消息、skill 启用状态 |
| 技能发现 | 同步 | 扫描 .reasonix/skills/ 等目录 |
| MCP eager 启动 | 同步 | 关键服务器阻塞直到就绪 |
| MCP background 启动 | 异步 | 后台 warm-up，不阻塞 UI |
| MCP lazy 启动 | 懒加载 | 首次调用时才 connect |
| 工具注册 | 同步 | Agent 立即可用 |

---

## Token Economy 模式

`token_mode = "economy"` 时，可选工具源默认**不注册**。模型通过 `connect_tool_source` 按需激活：

```
connect_tool_source("skills")     → 注册 skill 工具
connect_tool_source("codegraph")  → 注册 codegraph_* 工具
connect_tool_source("mcp", "github") → 连接指定 MCP 服务器
```

---

## Controller 构建后的流程

```
Build() → Controller + Host
  ↓
Desktop/TUI/HTTP Server
  ↓
用户输入 → controller.Submit() → submit() 路由
  ├─ 命令 → 对应的处理函数
  ├─ Skill → RunSkill → runGuarded → runGoalLoopWithRawDisplay
  └─ 普通输入 → runRefTurn → runGoalLoopWithRawDisplay
       └─ Compose() → Agent.Run() → 事件流 → frontend
```

---

## 代码索引

| 文件 | 内容 |
|------|------|
| `internal/boot/boot.go:100-900` | `Build()` 完整启动函数 |
| `internal/boot/boot.go:360-380` | 内置工具注册 |
| `internal/boot/boot.go:511-530` | Agent 工具 + memory + history 注册 |
| `internal/boot/boot.go:537-718` | Skill 工具 + 子代理包装注册 |
| `internal/boot/boot.go:237-280` | MCP 插件分层启动 |
| `internal/boot/token_profile.go` | economy 模式 + connect_tool_source |
| `internal/boot/boot.go:190-206` | 系统前缀构建 |
