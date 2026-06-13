# Hook System — 事件钩子与扩展点

<!--
版本: v1.0
创建: 2026-06-13 23:30:00
更新: 2026-06-13 23:30:00
-->

**能力类型**: 系统架构文档  
**涉及包**: `internal/hook/`、`internal/control/controller.go`（钩子触发点）

## 架构概览

Hook 系统允许外部脚本在 Agent 生命周期的 10 个事件点插入自定义逻辑：

```
配置             .reasonix/hooks/settings.json
                     ↓
Hook 定义        { event, matcher?, command, timeout }
                     ↓
加载             hook.Load(cwd) → []Hook
                     ↓
Runner           hook.Runner → 管理生命周期
                     ↓
触发点           controller / agent 各事件点
                   ├─ 阻塞式: PreToolUse, UserPromptSubmit
                   └─ 非阻塞: 其他 8 个
```

---

## 一、10 个事件点

| 事件 | 触发时机 | 阻塞? | 用途 |
|------|---------|:---:|------|
| `PreToolUse` | 工具执行前 | ✅ | 否决高风险操作、审计 |
| `PostToolUse` | 工具执行后 | ❌ | 日志、通知 |
| `UserPromptSubmit` | 用户消息提交前 | ✅ | 输入过滤、自定义路由 |
| `Stop` | Turn 结束时 | ❌ | 清理、保存状态 |
| `PostLLMCall` | 每次模型调用后 | ❌ | **改写推理内容**（独家） |
| `SessionStart` | 会话激活时 | ❌ | 环境准备、通知 |
| `SessionEnd` | 会话关闭时 | ❌ | 清理 |
| `SubagentStop` | 子代理完成时 | ❌ | 子代理结果处理 |
| `Notification` | 系统通知时 | ❌ | 桌面通知转发 |
| `PreCompact` | 压缩前 | ❌ | 自定义压缩策略 |

---

## 二、配置格式

```jsonc
// .reasonix/hooks/settings.json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "bash",
        "command": "~/.reasonix/hooks/audit-bash.sh",
        "timeout": 5000
      }
    ],
    "PostLLMCall": [
      {
        "command": "~/.reasonix/hooks/extract-todos.sh"
      }
    ]
  }
}
```

| 字段 | 说明 |
|------|------|
| `matcher` | 仅对匹配的工具名触发（支持 glob） |
| `command` | 执行的脚本路径 |
| `timeout` | 超时（ms），默认 10000 |

---

## 三、阻塞 vs 非阻塞

- **阻塞事件** (`PreToolUse`, `UserPromptSubmit`) — hook 可以返回 `block: true` 阻止操作
- **非阻塞事件** — 后台执行，不影响主流程

---

## 四、PostLLMCall 的特殊能力

唯一支持**修改推理结果**的事件：hook 脚本可以返回改写后的 `reasoning` / `text`，系统会使用改写后的版本。用于：
- 提取结构化信息（todo、commit message）
- 注入额外上下文
- 过滤敏感信息

---

## 五、运行时 API

每次 hook 调用接收 JSON stdin：
```json
{
  "cwd": "/path/to/project",
  "event": "PreToolUse",
  "tool_name": "bash",
  "tool_args": "{\"command\":\"rm -rf /\"}",
  "turn": 12,
  "session_id": "abc123"
}
```

返回 JSON stdout：
```json
{
  "block": false,
  "message": "approved",
  "reasoning": "rewritten reasoning text"
}
```

---

## 六、安全模型

`/hooks trust` 命令在项目首次加载 hooks 前需要用户明确信任。一次信任后该项目的 hooks 不再询问。

---

## 七、代码索引

| 文件 | 内容 |
|------|------|
| `internal/hook/hook.go` | Event 类型、Hook 结构、Run/Runner |
| `internal/hook/runner.go` | PromptSubmit/Stop/LLMCall 等触发方法 |
| `internal/control/controller.go:539-547` | UserPromptSubmit/Stop 触发点 |
| `internal/agent/agent.go` | PostLLMCall 触发点 |
