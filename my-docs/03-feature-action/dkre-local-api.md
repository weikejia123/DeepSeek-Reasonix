# dkre 本地 API

Reasonix Desktop 启动后会在 `127.0.0.1:17532` 监听一个轻量 HTTP API，允许外部进程与打开的 agent tab 交互。仅本机可访问。

## 查询所有 tab 状态

```
GET http://127.0.0.1:17532/dkre/api/tabs
```

返回当前所有 tab 的名称和工作状态。

### 示例

```bash
curl http://127.0.0.1:17532/dkre/api/tabs
```

### 响应

```json
[
  {
    "name": "DERE-1.7.0",
    "running": false,
    "pendingPrompt": false,
    "backgroundJobs": 0,
    "cancelRequested": false,
    "cancellable": false,
    "loopActive": false,
    "ready": true
  },
  {
    "name": "Global",
    "running": true,
    "pendingPrompt": false,
    "backgroundJobs": 1,
    "cancelRequested": false,
    "cancellable": true,
    "loopActive": false,
    "ready": true
  }
]
```

### 字段说明

| 字段 | 说明 |
|------|------|
| `name` | tab 标题，与桌面端 tab 标签一致 |
| `running` | **是否正在工作中** — agent 在思考、输出或等待确认工具调用时为 `true` |
| `pendingPrompt` | 是否有待处理的 prompt（如等待用户确认工具调用） |
| `backgroundJobs` | 后台运行的任务数量（如文件读取、MCP 调用） |
| `cancelRequested` | 是否已请求取消当前运行 |
| `cancellable` | 当前运行能否被取消 |
| `loopActive` | 是否处于 A-Loop 循环模式 |
| `ready` | tab 是否已初始化完成，可以接收消息 |

---

## 给指定 tab 发送消息

```
POST http://127.0.0.1:17532/dkre/api/tabs/{tabName}/send
Content-Type: application/json

{"message": "你的消息"}
```

将消息注入到指定 tab。消息在目标 tab 中显示为带蓝色前缀的用户气泡：
`MessageFromTab[dkre-api]: 你的消息`

发送后目标 tab 的 agent 会正常处理该消息并生成响应。

### 示例

```bash
curl -X POST http://127.0.0.1:17532/dkre/api/tabs/DERE-1.7.0/send \
  -H 'Content-Type: application/json' \
  -d '{"message":"帮我看看这个版本有什么更新"}'
```

### 响应

成功：

```json
{"status": "sent"}
```

tab 正忙（已有对话在进行中）：

```json
{"error": "tab \"DERE-1.7.0\" is currently busy; try again later"}
```

返回 HTTP 409，稍后重试即可。

tab 不存在：

```json
{"error": "no open tab with title \"NonExistentTab\""}
```

返回 HTTP 404。

请求体为空或缺少 `message` 字段：

```json
{"error": "message is required"}
```

返回 HTTP 400。

### URL 编码注意事项

Tab 名称中包含空格、中文、特殊字符时需要 URL 编码后再放入路径：

```bash
# tab 名称为 "my project"
curl -X POST http://127.0.0.1:17532/dkre/api/tabs/my%20project/send \
  -H 'Content-Type: application/json' \
  -d '{"message":"hello"}'

# tab 名称为 "测试"
curl -X POST http://127.0.0.1:17532/dkre/api/tabs/%E6%B5%8B%E8%AF%95/send \
  -H 'Content-Type: application/json' \
  -d '{"message":"你好"}'
```

---

## 端口配置

默认端口 `17532`，可通过环境变量覆盖：

```bash
export REASONIX_DESKTOP_API_PORT=17533
```

需在启动 Reasonix Desktop **之前**设置。

---

## 典型场景

### 1. 检查 tab 是否空闲再发送消息

```bash
# 先查状态
STATUS=$(curl -s http://127.0.0.1:17532/dkre/api/tabs | jq -r '.[] | select(.name=="DERE-1.7.0") | .running')
if [ "$STATUS" = "false" ]; then
  curl -X POST http://127.0.0.1:17532/dkre/api/tabs/DERE-1.7.0/send \
    -H 'Content-Type: application/json' \
    -d '{"message":"开始分析"}'
else
  echo "tab 正忙，稍后再试"
fi
```

### 2. 轮询等待 tab 完成

```bash
# 发送消息
curl -X POST http://127.0.0.1:17532/dkre/api/tabs/DERE-1.7.0/send \
  -H 'Content-Type: application/json' \
  -d '{"message":"帮我做个代码审查"}'

# 等待 running 变为 false
while true; do
  RUNNING=$(curl -s http://127.0.0.1:17532/dkre/api/tabs | jq -r '.[] | select(.name=="DERE-1.7.0") | .running')
  if [ "$RUNNING" = "false" ]; then
    echo "完成"
    break
  fi
  sleep 2
done
```
