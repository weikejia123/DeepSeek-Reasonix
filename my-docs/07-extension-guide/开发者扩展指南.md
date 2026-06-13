# L4 Extension Guide — 开发者扩展指南（7篇合集）

<!--
版本: v1.0
创建: 2026-06-14 00:25:00
更新: 2026-06-14 00:25:00
-->

## 如何新增一个内置 Tool

### Step 1: 实现 Tool 接口

```go
// internal/tool/builtin/my_tool.go
package builtin

import (
    "context"
    "encoding/json"
    "reasonix/internal/tool"
)

type myTool struct{}

func (myTool) Name() string        { return "my_tool" }
func (myTool) ReadOnly() bool      { return true }
func (myTool) Description() string { return "My custom tool description" }
func (myTool) Schema() json.RawMessage {
    return json.RawMessage(`{"type":"object","properties":{"input":{"type":"string"}},"required":["input"]}`)
}
func (t myTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
    var p struct{ Input string ` + "`" + `json:"input"` + "`" + ` }
    if err := json.Unmarshal(args, &p); err != nil {
        return "", err
    }
    return "result: " + p.Input, nil
}

func init() {
    tool.RegisterBuiltin(myTool{})  // 编译时自动注册
}
```

### Step 2: 重启，工具自动出现在 Registry

## 如何编写一个 Skill

### inline skill 示例

```markdown
---
name: my-skill
description: 我的第一个 skill
---
# 我的 Skill

当被调用时，请执行以下步骤:
1. 使用 grep 搜索相关代码
2. 使用 read_file 读取关键文件
3. 汇总发现并给出建议
```

保存到: `.reasonix/skills/my-skill/SKILL.md`

### subagent skill 示例

```markdown
---
name: code-analyzer
description: 深度代码分析子代理
runAs: subagent
allowed-tools: read_file, grep, glob, ls
model: deepseek/v4
effort: high
---
你是代码分析专家。收到任务后:
1. 先用 glob/grep 定位相关文件
2. 深度阅读关键代码
3. 返回蒸馏后的分析结论，附 file:line 引用
```

## 如何对接一个 MCP 服务器

```toml
# reasonix.toml
[[plugins]]
name = "my-server"
command = "node"
args = ["/path/to/my-mcp-server.js"]
tier = "lazy"    # eager | background | lazy

# 或 HTTP 方式:
# [[plugins]]
# name = "my-api"
# type = "http"
# url = "http://localhost:3000/mcp"
```

工具自动以 `mcp__my-server__<tool名>` 注册。

## 如何新增一个 Hook

```jsonc
// .reasonix/hooks/settings.json
{
  "hooks": {
    "PreToolUse": [{
      "matcher": "bash",      // 仅匹配 bash 工具
      "command": "./hooks/audit.sh",
      "timeout": 3000
    }],
    "PostLLMCall": [{
      "command": "./hooks/log-tokens.sh"
    }]
  }
}
```

Hook 脚本通过 stdin 接收 JSON payload，stdout 返回 JSON 结果。

## 如何编写自定义命令

```markdown
<!-- .reasonix/commands/review.md -->
---
description: Review changes with focus
argument-hint: [focus]
---
请审查当前的 git diff。$ARGUMENTS

重点检查:
- 潜在的 bug
- 缺失的错误处理
- 安全问题
```

调用: `/review "关注认证逻辑"`

## 如何新增 LLM 提供商

```go
// 实现 Provider 接口
type MyProvider struct{}

func (p *MyProvider) Name() string { return "my-provider" }
func (p *MyProvider) Chat(ctx context.Context, req provider.Request) (*provider.Response, provider.Stream, error) {
    // 调用你的 API
}
```

在 `boot.go` 中注册: `providers["my-provider"] = &MyProvider{}`

## 如何扩展 Desktop 前端

### 1. 添加 Reducer Action

```typescript
// useController.ts
case "my_custom_event": {
    return { ...s, /* 更新状态 */ };
}
```

### 2. 添加 Bridge 方法

```typescript
// bridge.ts
MyCustomAPI(): Promise<string>;

// app.go
func (a *App) MyCustomAPI() string { return "hello"; }
```

### 3. 添加 UI 组件

```tsx
// components/MyPanel.tsx
export function MyPanel({ data }: { data: string }) {
    return <div>{data}</div>;
}
```
