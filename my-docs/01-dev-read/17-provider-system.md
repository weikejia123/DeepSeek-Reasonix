# Provider System — LLM 调用与多提供商

<!--
版本: v1.0
创建: 2026-06-13 23:42:00
更新: 2026-06-13 23:42:00
-->

**能力类型**: 系统架构文档
**涉及包**: `internal/provider/`

## 架构

```
Provider 接口
  ├─ Chat(ctx, req) → (resp, stream)
  └─ 实现: openai / anthropic / deepseek / ...
        │
Agent.stream()
  ├─ 构建 Request{System, Messages, Tools, MaxTokens, Temperature}
  ├─ prov.Chat(ctx, req) → 流式 chunks
  └─ 解析: text, reasoning, tool calls, usage
```

## 一、Provider 接口

```go
type Provider interface {
    Name() string
    Chat(ctx context.Context, req Request) (*Response, Stream, error)
}

type Request struct {
    System      string
    Messages    []Message
    Tools       []ToolSchema
    MaxTokens   int
    Temperature float64
    Effort      string  // reasoning_effort for o-series
}

type Response struct {
    Text      string
    Reasoning string
    ToolCalls []ToolCall
    Usage     *Usage
}
```

## 二、多提供商支持

| Provider | Chat Model | 特色 |
|----------|-----------|------|
| DeepSeek | v3, v4, r1 | 自动前缀缓存、reasoning |
| OpenAI | gpt-4o, o3/o4 | reasoning_effort、结构化输出 |
| Anthropic | claude-* | 长上下文、扩展思考 |
| 其他 | 可插拔 | 通过配置文件扩展 |

## 三、前缀缓存

DeepSeek 自动检测 `System + Messages[:n]` 的稳定前缀并缓存——Reasonix 刻意保持 system prompt 在 session 期间不变以保持缓存温暖。`cacheHitTokens` / `cacheMissTokens` 在 Usage 事件中报告。

## 四、流式处理

```
Chat() → Stream
  └─ Next() → Chunk{Type, Text?, Reasoning?, ToolCall?, Usage?, Done?, Err?}
       │
Agent.stream() 消费:
  ├─ ChunkText → 追加到 text buffer → emit Text event
  ├─ ChunkReasoning → 追加到 reasoning buffer → emit Reasoning event
  ├─ ChunkToolCall → 累积 tool call 参数
  ├─ ChunkDone → 返回完整结果
  └─ ChunkErr → 中断恢复或返回错误
```

## 五、模型切换

`/model deepseek/v4-flash` → `SetModel("deepseek/v4-flash")` → 下次 turn 起效。`/provider` 切换提供商。

## 六、代码索引

| 文件 | 内容 |
|------|------|
| `internal/provider/provider.go` | Provider/Request/Response/ToolSchema 定义 |
| `internal/provider/openai/` | OpenAI 兼容实现 |
| `internal/agent/agent.go:569` | `stream()` 调用点 |
| `internal/agent/agent.go:592-599` | cache diagnostics + Usage 事件 |
