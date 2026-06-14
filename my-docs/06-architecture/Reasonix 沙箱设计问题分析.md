# Reasonix 沙箱设计问题分析

- 日期：2026-06-15
- 分析对象：DeepSeek-Reasonix（当前 fork）
- 范围：桌面端 + 内核的沙箱相关代码
- 性质：仅分析，不修正

## 1. 「沙箱」一词承载了三种完全不同的机制

当前系统里至少有三层「沙箱」概念，但 UI、配置和报错信息没有明确区分：

| 层级 | 控制配置 | 作用对象 | 说明 |
|------|----------|----------|------|
| 内置文件写入边界 | `[sandbox].workspace_root` / `allow_write` | `write_file`、`edit_file`、`multi_edit`、`move_file`、`notebook_edit`、`delete_range`、`delete_symbol` | 进程内路径前缀检查 |
| bash OS 沙箱 | `[sandbox].bash` + `[sandbox].network` | `bash` 工具执行的命令 | macOS 通过 `sandbox-exec` 包装 |
| MCP 插件自带边界 | MCP Server 自己的参数/环境变量 | 如 `filesystem` MCP | 完全独立，Reasonix 不感知 |

问题：桌面端设置面板只有一个「Sandbox」区块。用户关掉后以为「所有沙箱都关了」，但 MCP 插件的限制、文件写入边界可能仍然生效。报错里也不说明是哪一层在拦截。

## 2. 桌面端设置写入全局用户配置，而不是当前项目

`applyConfigChange` 把设置写到 `config.UserConfigPath()`（`~/.config/reasonix/reasonix.toml` 这类路径），然后调用 `a.rebuild()` 重建当前 Tab。

问题：
- 多项目 Tab 时，改 sandbox 是全局生效的，但 UI 没有提示「这会改变所有项目」。
- 写入全局后，当前项目的 `reasonix.toml` 反而被忽略，用户可能在项目里配了 `workspace_root`，但桌面端优先用了全局配置。
- 命令行 `reasonix` 读取的是项目级 `reasonix.toml`，而桌面端可能跑的是全局配置，导致「命令行正常、桌面端报错」或反过来。

## 3. `WriteRoots` 的 fallback 行为容易把限制锁死在错误目录

`Config.WriteRootsForRoot(fallbackRoot)` 逻辑：

```go
root := ExpandVars(c.Sandbox.WorkspaceRoot)
if root == "" {
    root = fallbackRoot
    ...
}
```

问题：
- 如果全局配置里 `workspace_root` 被设成了 `/private/tmp/reasonix-sandbox`，那么所有项目的写入边界都会被锁到这个目录，除非每个项目单独覆盖。
- 桌面 Tab 传给 `boot.Build` 的 `WorkspaceRoot` 是真实项目路径，但 `WriteRootsForRoot(root)` 优先使用配置里的 `WorkspaceRoot`，所以真实项目路径被配置里的错误路径覆盖。
- 默认配置 `SandboxConfig{Bash: "enforce", Network: true}` 没有默认值 `WorkspaceRoot`，这本来是对的（空就用项目路径），但一旦某次设置写入了非空 `workspace_root`，就很难被发现。

## 4. Bash 沙箱默认开启，但只在 macOS 有效，其他平台静默降级

`BashMode()`：

```go
if c.Sandbox.Bash == "off" {
    return "off"
}
return "enforce"
```

`Command()`：

```go
if !spec.enforce() || !Available() {
    return sh.argv(command), false
}
```

问题：
- 默认是 `enforce`，但 `sandbox-exec` 只有 macOS 有。Windows/Linux 上 `Available()` 返回 false，于是**静默不包装**，bash 实际无限制。
- 同一个配置在不同平台安全性不一致，用户换系统后以为还有沙箱，其实已经裸奔。
- `doctor/report.go` 会提示 `(inactive: no OS sandbox on this host)`，但普通用户不一定看 doctor。

## 5. Seatbelt 配置允许了大量额外写入目录，削弱了边界意义

`seatbelt_darwin.go` 的 `writeAllowDirs` 会自动加入：

```go
"/dev", "/tmp", "/private/tmp", "/private/var/folders", os.TempDir()
home/{Library/Caches, .cache, .npm, .cargo, go}
```

问题：
- `/tmp` 和 `$TMPDIR` 是全局可写的，任何命令都可以把数据写到临时目录，再通过其他方式带走。
- 家目录缓存区（`.npm`、`.cargo`、`go`）虽然合理，但意味着沙箱内的命令可以修改这些缓存，存在投毒风险。
- 没有选项关闭这些额外目录，用户无法收紧。

## 6. 文件写入边界和 bash 写入边界是两套实现，规则可能不一致

- 文件写入边界：`internal/tool/builtin/confine.go` 用 `within()` 判断路径前缀。
- bash 写入边界：`internal/sandbox/seatbelt_darwin.go` 用 Seatbelt profile。

问题：
- `within()` 用 `filepath.Rel` 正确判断子目录，Seatbelt 的 `(subpath ...)` 也正确，两者算法一致。
- 但文件写入工具在 `resolveIn()` 里对相对路径的处理是：空 workDir 返回原路径，否则 join。如果 `workDir` 是真实项目路径，而 `roots` 被配置覆盖成 `/private/tmp/reasonix-sandbox`，那么相对路径会先被 join 到真实项目路径，再被 confine 拒绝——这就是出现「路径定位错乱」的原因。
- 也就是说，**工具看到的工作目录和写入边界可能指向不同根目录**，这是设计上的不一致。

## 7. MCP 插件没有任何与 Reasonix 沙箱的联动

`boot.go` 里：

```go
bashSpec := sandbox.Spec{Mode: cfg.BashMode(), WriteRoots: cfg.WriteRootsForRoot(root), Network: cfg.Sandbox.Network}
...
pluginHost := plugin.NewHost()
```

问题：
- `bashSpec` 只传给内置工具，插件 host 完全拿不到这些 roots。
- filesystem MCP 的 allowed directories 要么来自它自己的配置，要么来自导入的 Claude `mcp.json`，Reasonix 不会自动把当前项目路径注入进去。
- 用户在桌面端打开项目 A，但 filesystem MCP 可能只允许访问 `/private/tmp/reasonix-sandbox`，这种「项目上下文不传递给插件」是沙箱设计上的重大缺口。

## 8. 设置变更要求「无运行时工作」才能重建

`ensureActiveTabRebuildAllowed`：

```go
if controllerHasActiveRuntimeWork(tab.Ctrl) {
    return rebuildControllerActiveWorkError(setting)
}
```

问题：
- 修改 sandbox 后必须重建 controller 才生效。如果当前 Tab 正在运行（有 background job、待确认提示等），设置保存会被拒绝。
- 用户可能在设置里改了沙箱，以为生效了，但实际没重建，旧配置仍在运行。
- 没有提示用户「需要重建 Tab」或自动延后重建。

## 9. `allow_write` 是全局配置，没有按项目或按 Tab 的能力

问题：
- 某些项目需要访问兄弟目录或外部资源，但 `allow_write` 写入全局配置后会影响所有项目。
- 桌面端每个 Tab 是独立 controller，但共享同一份用户全局配置，无法为某个 Tab 单独放开路径。

## 10. 缺乏「当前生效边界」的可观测性

问题：
- UI 没有展示当前 Tab 实际生效的 `WriteRoots`、`BashMode`、MCP allowed directories。
- 用户只能等报错时才发现边界不对。doctor 报告虽然能看到 sandbox 配置，但不是每个用户都会运行。

## 总结

当前沙箱设计最大的几个痛点：

| 问题 | 表现 |
|------|------|
| 名称混淆 | 一个「沙箱」设置无法关掉 MCP 插件限制 |
| 全局配置覆盖项目路径 | `workspace_root` 全局生效，导致所有项目被锁到错误目录 |
| 桌面端与命令行配置来源不同 | 项目 `reasonix.toml` 可能被全局配置覆盖 |
| 文件工具的工作目录与写入边界分离 | 相对路径解析到项目，但 confine 用另一套 roots |
| 跨平台安全不一致 | macOS 有 seatbelt，其他平台静默无沙箱 |
| MCP 插件无项目感知 | filesystem MCP 不会自动允许当前项目路径 |
| 缺少生效边界展示 | 用户难以排查为什么路径被拒绝 |

这些都不是单个代码 bug，而是**架构层面把「内置工具边界」「OS 命令边界」「第三方插件边界」混在同一个配置命名空间下，同时桌面端又把它做成了全局配置**导致的设计张力。
