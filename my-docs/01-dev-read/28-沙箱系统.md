# Sandbox — Bash 安全隔离

<!--
版本: v1.0
创建: 2026-06-13 23:58:00
更新: 2026-06-13 23:58:00
-->

**能力类型**: 系统架构文档
**涉及包**: `internal/sandbox/`、`internal/tool/builtin/bash.go`

## 架构

```
bash(command)
  ├─ 沙箱模式? → 包装命令
  │     ├─ macOS Seatbelt → sandbox-exec profile
  │     └─ Linux (预留)
  └─ 非沙箱 → 直接执行
```

## 一、沙箱模式

```toml
[tools.bash]
sandbox = "off"  # off | macos
```

### macOS Seatbelt

使用 Apple 的 `sandbox-exec`（Seatbelt）机制：

- **写入限制**: 仅允许写入项目目录（`WriteRoots`）和临时目录
- **网络限制**: 可配置出站连接（`ProxySpec`）
- **文件读取**: 限制在项目范围内

沙箱 profile 格式（`.sb` 文件）定义允许的路径和操作。

## 二、非沙箱模式 (off)

直接执行 shell 命令，依赖：
- **权限门控** — bash 工具需用户批准（除非 YOLO）
- **Hook 拦截** — PreToolUse hook 可否决
- **受保护目录** — macOS 系统目录写保护（需额外确认）

## 三、后台任务

```go
// bash.go + bgjobs.go
bash(command, run_in_background=true)
  → jobID (bash-1)
bash_output(job_id="bash-1")  → 增量读取后台输出
kill_shell(job_id="bash-1")   → 终止后台任务
wait()                         → 等待所有后台任务完成
```

## 四、代码索引

| 文件 | 内容 |
|------|------|
| `internal/sandbox/seatbelt_darwin.go` | macOS Seatbelt 沙箱 |
| `internal/sandbox/sandbox.go` | 沙箱 Spec 定义 |
| `internal/tool/builtin/bash.go` | bash 工具实现 |
| `internal/tool/builtin/bgjobs.go` | 后台任务管理 |
