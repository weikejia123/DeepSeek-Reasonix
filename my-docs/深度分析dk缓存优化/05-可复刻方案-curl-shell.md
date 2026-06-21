# 五、可复刻方案：用 curl + shell 实现相同效果

本节展示如何只用 `curl` 和 `sh` 脚本调用 DeepSeek API，实现 Reasonix 的核心 token 节约优化。假设你有文件读写能力（相当于 curl 可以读写本地文件）。

## 核心原则

要利用 DeepSeek 的自动前缀缓存，必须遵守以下规则：

1. **System message 一旦确定就不再修改** — 所有对话中使用相同 system prompt
2. **工具定义列表一旦确定就不再修改** — 所有请求使用相同的 `tools` 数组
3. **只追加，不修改** — 每轮只在 messages 数组末尾追加新消息，不修改已有消息
4. **当上下文超限时压缩中间部分** — 重新构建 messages 数组

## 基础实现

### 第一步：初始化对话

```bash
#!/bin/bash
# 保存 system prompt 到文件，之后永不修改

cat > system_prompt.txt << 'PROMPT'
You are a helpful coding assistant. You can use the following tools to help users.

# Available Tools
- read_file: Read a file from disk
- edit_file: Edit a file by replacing exact text
- bash: Execute a shell command

# Rules
- Always reply in the same language the user uses
- Use tools when appropriate, don't just describe what to do
PROMPT

# 保存 tools 定义到文件，之后永不修改
cat > tools.json << 'TOOLS_EOF'
[
  {
    "type": "function",
    "function": {
      "name": "read_file",
      "description": "Read a file from disk",
      "parameters": {
        "type": "object",
        "properties": {
          "path": {"type": "string", "description": "File path"}
        },
        "required": ["path"]
      }
    }
  },
  {
    "type": "function",
    "function": {
      "name": "edit_file",
      "description": "Edit a file by replacing exact text",
      "parameters": {
        "type": "object",
        "properties": {
          "path": {"type": "string"},
          "old_string": {"type": "string"},
          "new_string": {"type": "string"}
        },
        "required": ["path", "old_string", "new_string"]
      }
    }
  },
  {
    "type": "function",
    "function": {
      "name": "bash",
      "description": "Execute a shell command",
      "parameters": {
        "type": "object",
        "properties": {
          "command": {"type": "string"}
        },
        "required": ["command"]
      }
    }
  }
]
TOOLS_EOF

# 初始化 messages 文件：只有 system message
echo '[{"role": "system", "content": "'$(cat system_prompt.txt | jq -sRr @json)'"}]' > messages.json
```

### 第二步：发送第一轮对话

```bash
#!/bin/bash
DEEPSEEK_API_KEY="sk-your-key-here"
MODEL="deepseek-v4-flash"

# 构建本轮完整的 messages 数组
# 注意：直接使用已有的 messages.json，在其末尾追加 user message
USER_INPUT="帮我看看当前目录有什么文件"

# 在 messages.json 末尾追加新的 user message
# jq 可以安全地操作 JSON
jq --arg content "$USER_INPUT" '. + [{"role": "user", "content": $content}]' messages.json > request_messages.json

# 发送请求
curl -s https://api.deepseek.com/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $DEEPSEEK_API_KEY" \
  -d "$(jq -n --arg model "$MODEL" \
    --slurpfile msgs <(cat request_messages.json) \
    --slurpfile tools <(cat tools.json) \
    '{
      model: $model,
      messages: $msgs[0],
      tools: $tools[0],
      stream: false,
      stream_options: {include_usage: true}
    }')" > response.json

# 提取 assistant 回复
ASSISTANT_CONTENT=$(jq -r '.choices[0].message.content // empty' response.json)
TOOL_CALLS=$(jq -c '.choices[0].message.tool_calls // []' response.json)

# 检查缓存命中率
CACHE_HIT=$(jq -r '.usage.prompt_cache_hit_tokens // 0' response.json)
CACHE_MISS=$(jq -r '.usage.prompt_cache_miss_tokens // 0' response.json)
echo "Cache hit: $CACHE_HIT, Cache miss: $CACHE_MISS"

# 将 assistant 回复追加到 messages.json
if [ "$TOOL_CALLS" != "[]" ]; then
  # 有工具调用
  jq --arg content "$ASSISTANT_CONTENT" \
    --argjson tool_calls "$TOOL_CALLS" \
    '. + [{"role": "assistant", "content": $content, "tool_calls": $tool_calls}]' \
    messages.json > messages_tmp.json && mv messages_tmp.json messages.json
else
  # 纯文本回复
  jq --arg content "$ASSISTANT_CONTENT" \
    '. + [{"role": "assistant", "content": $content}]' \
    messages.json > messages_tmp.json && mv messages_tmp.json messages.json
fi

# 保存本轮请求信息用于诊断
echo "{\"turn\": 1, \"cache_hit\": $CACHE_HIT, \"cache_miss\": $CACHE_MISS, \"prompt\": $((CACHE_HIT + CACHE_MISS))}" >> cache_stats.jsonl
```

### 第三步：处理工具调用

```bash
#!/bin/bash
# 接到工具调用后，执行工具，将结果追加到 messages

TOOL_CALLS=$(jq -c '.choices[0].message.tool_calls // []' response.json)

echo "$TOOL_CALLS" | jq -c '.[]' | while read -r tc; do
  FUNC_NAME=$(echo "$tc" | jq -r '.function.name')
  ARGS=$(echo "$tc" | jq -r '.function.arguments')
  TOOL_CALL_ID=$(echo "$tc" | jq -r '.id')
  
  # 执行工具
  case "$FUNC_NAME" in
    bash)
      CMD=$(echo "$ARGS" | jq -r '.command')
      RESULT=$(eval "$CMD" 2>&1)
      ;;
    read_file)
      PATH_ARG=$(echo "$ARGS" | jq -r '.path')
      RESULT=$(cat "$PATH_ARG" 2>&1)
      ;;
    *)
      RESULT="Error: unknown tool $FUNC_NAME"
      ;;
  esac
  
  # 将工具结果追加到 messages
  # 注意截断大输出（超过32KB的部分截断）
  RESULT_LEN=${#RESULT}
  if [ "$RESULT_LEN" -gt 32768 ]; then
    RESULT="${RESULT:0:16384}
[truncated $((RESULT_LEN - 32768)) bytes]
${RESULT: -16384}"
  fi
  
  jq --arg id "$TOOL_CALL_ID" \
    --arg content "$RESULT" \
    '. + [{"role": "tool", "tool_call_id": $id, "content": $content}]' \
    messages.json > messages_tmp.json && mv messages_tmp.json messages.json
done
```

### 第四步：继续对话（多轮）

```bash
#!/bin/bash
# 每轮只追加 user message → 发送 → 追加 assistant message → 处理工具调用

USER_INPUT="然后帮我创建一个新文件"

# 核心：复用已有的 messages.json，只 append
jq --arg content "$USER_INPUT" \
  '. + [{"role": "user", "content": $content}]' \
  messages.json > request_messages.json

# 发送请求（同第二步）
# ...（复用第二步的 curl 命令）

# 检查缓存命中率
echo "Turn N: cache_hit=$CACHE_HIT, cache_miss=$CACHE_MISS"
```

**关键：每轮只 append，messages.json 文件持续增长，但 system message 始终是第一条且内容不变。**

## 监控缓存命中率

```bash
#!/bin/bash
# 从 response.json 提取缓存数据
USAGE=$(jq '.usage' response.json)
echo "Debug: usage = $USAGE"
echo "Method 1 - DeepSeek top-level fields:"
echo "  prompt_cache_hit_tokens:  $(jq -r '.usage.prompt_cache_hit_tokens // 0' response.json)"
echo "  prompt_cache_miss_tokens: $(jq -r '.usage.prompt_cache_miss_tokens // 0' response.json)"

echo "Method 2 - OpenAI nested fields:"
echo "  prompt_tokens_details.cached_tokens: $(jq -r '.usage.prompt_tokens_details.cached_tokens // 0' response.json)"

# 计算命中率
HIT=$(jq -r '.usage.prompt_cache_hit_tokens // 0' response.json)
MISS=$(jq -r '.usage.prompt_cache_miss_tokens // 0' response.json)
TOTAL=$((HIT + MISS))
if [ "$TOTAL" -gt 0 ]; then
  PCT=$(echo "scale=2; $HIT * 100 / $TOTAL" | bc)
  echo "Cache hit rate: ${PCT}%"
fi

# 计算成本（以 deepseek-v4-flash 为例）
# Cache hit: ¥0.02/1M, Cache miss: ¥1/1M, Output: ¥2/1M
OUTPUT=$(jq -r '.usage.completion_tokens // 0' response.json)
COST=$(echo "scale=6; $HIT * 0.02 / 1000000 + $MISS * 1 / 1000000 + $OUTPUT * 2 / 1000000" | bc)
echo "Estimated cost: ¥$COST"
```

## 第五步：上下文压缩（messages 已满时）

当 messages 文件大小接近模型上下文窗口时，需要压缩：

```bash
#!/bin/bash
# 压缩中间部分

# 1. 确定要保留的头部和尾部
# 保留 system message + 前 2 条 user turns
# 保留最后 10 条消息（尾部）
# 中间部分让 LLM 总结

# 方案 A：用 LLM 总结中间部分
cat > /tmp/fold_prompt.txt << 'PROMPT'
Please summarize the conversation history below concisely. Extract:
1. Standing facts about the project
2. Files that have been created or modified
3. Commands that have been run
4. Decisions that were made
5. Any pending tasks

Keep it brief but informative. The summary should help continue the conversation efficiently.
PROMPT

# 从 messages.json 提取要总结的中间部分
jq '.[1:-10]' messages.json > /tmp/fold_messages.json

# 调用 LLM 总结
SUMMARY=$(curl -s https://api.deepseek.com/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $DEEPSEEK_API_KEY" \
  -d "$(jq -n --arg model "$MODEL" \
    --slurpfile msgs <(cat /tmp/fold_messages.json) \
    '{
      model: $model,
      messages: [{"role": "system", "content": "Summarize concisely."}] + $msgs[0],
      stream: false
    }')" | jq -r '.choices[0].message.content')

# 2. 重建 messages
# 保留: system + 前2条user turns + 总结 + 最后10条消息
SYSTEM=$(jq '.[0]' messages.json)
FIRST_USERS=$(jq '.[1:3]' messages.json)
TAIL=$(jq '.[-10:]' messages.json)

# 重建
echo "[$SYSTEM, $FIRST_USERS[0], $FIRST_USERS[1], {\"role\": \"user\", \"content\": \"<compaction-summary>$SUMMARY</compaction-summary>\"}, $TAIL]" | jq -s 'add' > messages.json
```

## 完整脚本示例

以下是一个完整的单文件脚本，实现了上述所有机制：

```bash
#!/bin/bash
# dk-chat.sh — 最小化复刻 DeepSeek prefix caching 优化

set -euo pipefail
DEEPSEEK_API_KEY="${DEEPSEEK_API_KEY:-}"
MODEL="${MODEL:-deepseek-v4-flash}"
SESSION_DIR="${SESSION_DIR:-./dk-session}"

mkdir -p "$SESSION_DIR"

# 初始化 system prompt 和 tools（只执行一次）
init_session() {
  cat > "$SESSION_DIR/system_prompt.txt" << 'EOSYSTEM'
You are a helpful assistant with file and shell access.
Always reply in the same language the user uses.
EOSYSTEM

  cat > "$SESSION_DIR/tools.json" << 'EOTOOLS'
[{"type":"function","function":{"name":"bash","description":"Run a shell command","parameters":{"type":"object","properties":{"command":{"type":"string"}},"required":["command"]}}},{"type":"function","function":{"name":"read_file","description":"Read a file","parameters":{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}}}]
EOTOOLS

  SYSTEM_CONTENT=$(cat "$SESSION_DIR/system_prompt.txt" | jq -sRr @json)
  echo '[{"role": "system", "content": '"$SYSTEM_CONTENT"'}]' > "$SESSION_DIR/messages.json"
  echo 0 > "$SESSION_DIR/turn"
}

# 发送一轮对话
send_turn() {
  local user_input="$1"
  local turn=$(($(cat "$SESSION_DIR/turn") + 1))
  
  # 1. 只追加 user message
  jq --arg content "$user_input" \
    '. + [{"role": "user", "content": $content}]' \
    "$SESSION_DIR/messages.json" > "$SESSION_DIR/request_messages.json"
  
  # 2. 发送请求
  curl -s "https://api.deepseek.com/chat/completions" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $DEEPSEEK_API_KEY" \
    -d "$(jq -n \
      --arg model "$MODEL" \
      --slurpfile msgs <(cat "$SESSION_DIR/request_messages.json") \
      --slurpfile tools <(cat "$SESSION_DIR/tools.json") \
      '{
        model: $model,
        messages: $msgs[0],
        tools: $tools[0],
        stream: false,
        stream_options: {include_usage: true}
      }')" > "$SESSION_DIR/response_${turn}.json"
  
  # 3. 提取缓存统计
  local hit miss
  hit=$(jq -r '.usage.prompt_cache_hit_tokens // 0' "$SESSION_DIR/response_${turn}.json")
  miss=$(jq -r '.usage.prompt_cache_miss_tokens // 0' "$SESSION_DIR/response_${turn}.json")
  echo "Turn $turn — cache hit: $hit, cache miss: $miss"
  echo "{\"turn\":$turn,\"hit\":$hit,\"miss\":$miss,\"total\":$((hit+miss))}" >> "$SESSION_DIR/cache_stats.jsonl"
  
  # 4. 追加 assistant 回复到 messages
  local content tool_calls
  content=$(jq -r '.choices[0].message.content // ""' "$SESSION_DIR/response_${turn}.json")
  tool_calls=$(jq -c '.choices[0].message.tool_calls // []' "$SESSION_DIR/response_${turn}.json")
  
  if [ "$tool_calls" != "[]" ]; then
    jq --arg content "$content" --argjson tc "$tool_calls" \
      '. + [{"role": "assistant", "content": $content, "tool_calls": $tc}]' \
      "$SESSION_DIR/messages.json" > "$SESSION_DIR/messages_tmp.json"
  else
    jq --arg content "$content" \
      '. + [{"role": "assistant", "content": $content}]' \
      "$SESSION_DIR/messages.json" > "$SESSION_DIR/messages_tmp.json"
  fi
  mv "$SESSION_DIR/messages_tmp.json" "$SESSION_DIR/messages.json"
  
  # 5. 处理工具调用
  echo "$tool_calls" | jq -c '.[]' | while read -r tc; do
    local func_name func_args tc_id result result_len
    
    tc_id=$(echo "$tc" | jq -r '.id')
    func_name=$(echo "$tc" | jq -r '.function.name')
    func_args=$(echo "$tc" | jq -r '.function.arguments')
    
    case "$func_name" in
      bash)
        result=$(eval "$(echo "$func_args" | jq -r '.command')" 2>&1 || true)
        ;;
      read_file)
        result=$(cat "$(echo "$func_args" | jq -r '.path')" 2>&1 || true)
        ;;
      *)
        result="Unknown tool: $func_name"
        ;;
    esac
    
    # 截断大输出
    result_len=${#result}
    if [ "$result_len" -gt 32768 ]; then
      result="${result:0:16384}"$'\n'"${result: -16384}"
    fi
    
    jq --arg id "$tc_id" --arg content "$result" \
      '. + [{"role": "tool", "tool_call_id": $id, "content": $content}]' \
      "$SESSION_DIR/messages.json" > "$SESSION_DIR/messages_tmp.json"
    mv "$SESSION_DIR/messages_tmp.json" "$SESSION_DIR/messages.json"
  done
  
  echo "$turn" > "$SESSION_DIR/turn"
}

# 使用方式
# init_session
# send_turn "你好，帮我看看当前目录"
# send_turn "然后创建一个文件 test.txt"
```

## 使用说明

1. 设置环境变量 `DEEPSEEK_API_KEY`
2. 调用 `init_session()` 初始化
3. 重复调用 `send_turn("你的问题")` 进行多轮对话
4. 查看 `$SESSION_DIR/cache_stats.jsonl` 监控缓存命中率

随着对话轮次增加，cache hit 占比会逐渐接近 90%+，因为 system message 和所有历史消息都被缓存了，只有最新追加的 user message 产生 cache miss。
