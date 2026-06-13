# MCP System — 插件连接、传输与工具代理

<!--
版本: v1.0
创建: 2026-06-13 23:20:00
更新: 2026-06-13 23:20:00
-->

**能力类型**: 系统架构文档  
**涉及包**: `internal/plugin/`、`internal/boot/boot.go`、`internal/builtinmcp/`、`internal/config/`

## 架构概览

MCP (Model Context Protocol) 系统将外部工具服务器无缝接入 Agent 的工具表：

```
配置层         reasonix.toml [[plugins]] / .mcp.json
                  ↓
Spec 描述       plugin.Spec{Name, Type, Command/URL, Env, Headers}
                  ↓
连接层         plugin.Host → Client per server
  ├─ stdio   → 子进程 + JSON-RPC over stdin/stdout
  ├─ http    → Streamable HTTP (2025 spec)
  └─ sse     → 旧 HTTP+SSE 传输
                  ↓
工具代理层     LazyTool (plugin/lazy.go)
  ├─ 懒启动  → 首次调用时 handshake
  ├─ 命名    → mcp__<server>__<tool>
  └─ 故障    → 优雅降级错误消息
                  ↓
注册层         tool.Registry.Add(t)
                  ↓
模型可见       Schemas() 导出 mcp__* 工具
```

---

## 一、配置定义

### reasonix.toml

```toml
[[plugins]]
name = "github"
type = "http"
url = "https://api.github.com/mcp"
headers = { Authorization = "${GITHUB_TOKEN}" }

[[plugins]]
name = "fs"
command = "npx"
args = ["-y", "@modelcontextprotocol/server-filesystem", "."]
```

### .mcp.json（Claude Code 兼容）

`config/mcpjson.go` 自动加载项目根目录的 `.mcp.json`，与 TOML 配置合并。同名时 TOML 优先。

---

## 二、Spec 结构

```go
// plugin/plugin.go:32-51
type Spec struct {
    Name    string            // 服务器标识
    Type    string            // "stdio" | "http" | "sse"
    Command string            // stdio 命令
    Args    []string
    Env     map[string]string
    URL     string            // http/sse 地址
    Headers map[string]string
    Dir     string            // stdio 工作目录
    Stderr  io.Writer         // 子进程 stderr
}
```

---

## 三、三层启动策略

`plugin.Host` 支持三个启动层（`plugin.go:157-169`）：

| 层 | 配置 | 行为 |
|-----|------|------|
| `eager` | `tier = "eager"` | Build 时立即启动（阻塞 session 就绪） |
| `background` | `tier = "background"`（默认） | Build 后异步启动 |
| `lazy` | `tier = "lazy"` | 首次工具调用时启动 |

启动并发限制：同时最多 4 个 stdio 子进程启动（防 fork-bomb）。

---

## 四、LazyTool — 懒加载工具代理

`plugin/lazy.go` 为每个 MCP 工具创建代理，模型看到的工具名是 `mcp__<server>__<tool>`。

### 执行流程

```
模型调用 mcp__github__search_code(args)
  → Registry.Get("mcp__github__search_code")
  → LazyTool.Execute(ctx, args)
      ├─ 服务器未启动? → 启动 + initialize handshake
      ├─ 服务器故障?   → 返回友好错误
      ├─ 启动中?       → "still initializing — retry next turn"
      └─ 就绪          → tools/call → 返回结果
```

### 首次调用 connect 工具

`mcp__<server>__connect` 是一个特殊工具（`lazy.go:258`），调用一次完成 handshake，服务器真实工具在下一 turn 的 `Schemas()` 中可见。

---

## 五、内置 MCP 服务器

`internal/builtinmcp/builtinmcp.go` 提供代码智能服务器：

| 服务器 | 工具 | 说明 |
|--------|------|------|
| `codegraph` | `codegraph_*` (9 tools) | 符号搜索、调用图、影响分析 |
| `lsp` | `lsp_*` (4 tools) | LSP 定义、悬停、引用、诊断 |

这些作为 MCP 服务器运行但内置在 Reasonix 中，通过 `codegraph` Go 模块 + gopls LSP 后端实现。

---

## 六、配置加载优先级

```
项目 reasonix.toml > 全局 ~/.reasonix/reasonix.toml > .mcp.json
```

`config/edit.go` 提供 `UpsertPlugin` / `RemovePlugin` 运行时编辑。

---

## 七、代码索引

| 文件 | 内容 |
|------|------|
| `internal/plugin/plugin.go` | Spec、Host、Client、transport 接口 |
| `internal/plugin/lazy.go` | LazyTool 代理 + 懒启动 |
| `internal/plugin/transport_stdio.go` | stdio 子进程传输 |
| `internal/plugin/transport_http.go` | HTTP/SSE 传输 |
| `internal/builtinmcp/builtinmcp.go` | 内置 CodeGraph + LSP 服务器 |
| `internal/config/config.go:968` | PluginEntry 配置结构 |
| `internal/config/mcpjson.go` | .mcp.json 解析 |
| `internal/boot/boot.go:237-280` | Build 阶段 MCP 启动 |
