# sendtab：跨 Tab 消息投递

> Desktop 专享。源 Tab 向另一个已打开的 Tab 发送消息，目标 Tab 的聊天历史会显示 `MessageFromTab[源Tab标题]: 消息内容`，并在时间线标记为蓝色。

## 用户用法：斜杠指令

```text
/sendtab <目标tab标题> <消息内容>
```

示例：

```text
/sendtab DERE-1.7.0 这是一条测试记录，不用过度思考。
```

目标 Tab 会收到：

```text
MessageFromTab[当前Tab标题]: 这是一条测试记录，不用过度思考。
```

### 标题含空格

用双引号包裹标题：

```text
/sendtab "Bug 修复" 请检查第 42 行的越界问题
```

### 常见错误

| 错误输入 | 结果 |
|----------|------|
| `/sendtab` | 缺少目标标题和消息 |
| `/sendtab DERE-1.7.0` | 缺少消息内容 |
| `/sendtab "DERE-1.7.0` | 引号未闭合 |
| `/sendtab 不存在的Tab 你好` | 提示“no open tab with title” |

## Agent 用法：工具调用

当用户说“把这句话发到 DERE-1.7.0 Tab”或“告诉另一个 Tab ……”时，使用 `sendtab` 工具。

```json
{
  "tabName": "DERE-1.7.0",
  "message": "这是一条测试记录，不用过度思考。"
}
```

### 何时使用工具

- 用户明确要求向另一个已打开 Tab 发送消息
- 用户说“在 xxx Tab 里执行/询问/记录 ……”且当前不在该 Tab

### 何时不要使用

- 目标 Tab 未打开 —— 先提示用户打开
- 当前已经在目标 Tab —— 直接回复即可，不必跨 Tab 发送
- 消息为空或只有空白 —— 拒绝并说明需要内容

## 匹配规则

- Tab 标题**精确匹配**，区分大小写
- 只匹配当前已打开的 Tab
- 多个 Tab 不会同时命中；未命中会明确报错

## 显示效果

目标 Tab 的聊天记录会新增一条用户气泡：

```text
MessageFromTab[源Tab标题]: 消息内容
```

时间线导航条对应圆点显示为蓝色，便于与正常用户消息、A-Loop 消息区分。
