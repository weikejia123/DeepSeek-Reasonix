# Config System — reasonix.toml 配置结构

<!--
版本: v1.0
创建: 2026-06-13 23:56:00
更新: 2026-06-13 23:56:00
-->

**能力类型**: 系统架构文档
**涉及包**: `internal/config/`

## 一、配置层级（加载优先级）

```
环境变量 RSNX_CONFIG          ← 最高优先
命令行 --config path          ←
项目 reasonix.toml            ← 覆盖全局
全局 ~/.reasonix/reasonix.toml ← 基础
.mcp.json (Claude Code兼容)   ← MCP服务器最低优先
```

## 二、主要配置节

```toml
[providers.deepseek]         # 模型提供商
  api_key = "${DEEPSEEK_KEY}"

[agent]                      # Agent 行为
  max_steps = 0              # 0=无限
  temperature = 0.0
  auto_plan = "auto"         # auto|off|always
  context_window = 128000

[agent.compaction]           # 压缩阈值
  ratio = 0.8
  soft_ratio = 0.7
  force_ratio = 0.95

[[plugins]]                  # MCP 服务器
  name = "github"
  type = "http"
  url = "https://..."
  tier = "lazy"              # eager|background|lazy

[skills]                     # 技能配置
  disabled_skills = ["review"]
  custom_paths = ["~/.my-skills"]
  excluded_paths = ["~/.agents/skills"]

[mcp]                        # MCP 配置
  disabled_servers = ["figma"]

[tools.bash]                 # Bash 工具配置
  sandbox = "off"            # off|macos

[memory]                     # 记忆配置
  disabled_types = []
```

## 三、特殊机制

### 环境变量替换
```
api_key = "${MY_KEY}"  → os.Getenv("MY_KEY")
```

### 项目级覆盖
项目 `reasonix.toml` 的 `providers` 会**替换**同名的全局 provider（不合并），方便项目锁定模型。

### .mcp.json 兼容
`config/mcpjson.go` 自动加载 Claude Code 格式的 `.mcp.json`，与 TOML [[plugins]] 并存，同名时 TOML 优先。

## 四、代码索引

| 文件 | 内容 |
|------|------|
| `internal/config/config.go` | Provider/Agent/Plugin 结构定义 |
| `internal/config/config.go:1125-` | Load() 合并逻辑 |
| `internal/config/mcpjson.go` | .mcp.json 解析与合并 |
| `internal/config/edit.go` | 运行时编辑 (UpsertPlugin/RemovePlugin) |
