# 快速上手

<!--
版本: v1.0
创建: 2026-06-14 00:06:00
更新: 2026-06-14 00:06:00
-->

## 1. 安装

```bash
git clone https://github.com/deepseek-ai/DeepSeek-Reasonix.git
cd DeepSeek-Reasonix
go build -o reasonix ./cmd/reasonix
```

## 2. 配置 API Key

```bash
mkdir -p ~/.reasonix
cat > ~/.reasonix/reasonix.toml << 'EOF'
[providers.deepseek]
api_key = "${DEEPSEEK_API_KEY}"

[agent]
max_steps = 0        # 0 = 无上限
temperature = 0.0
EOF
```

环境变量方式：`export DEEPSEEK_API_KEY="sk-..."`

## 3. 发送第一条消息

```bash
cd your-project
./reasonix
```

进入 TUI 后：

```
> 分析这个项目的结构
```

你会看到：
1. `···` — 系统提示词加载
2. 模型返回第一条回复
3. 模型可能调用 `ls`、`read_file`、`grep` 等工具
4. 最终输出分析结果

## 4. 核心交互

| 操作 | 按键 |
|------|------|
| 发送消息 | Enter |
| 在 running 期间发 steer | 直接输入 + Enter |
| 取消当前 turn | Esc |
| 退出 | `/quit` 或 Ctrl+C |
| 查看命令列表 | `/help` |
| 切换模型 | `/model deepseek/v4-flash` |
| 自主执行目标 | `/goal 为所有 Go 文件添加错误处理` |
| 审查变更 | `/review`（调用 review skill） |

## 5. 三种模式

```
模式切换: 输入框左下角的 mode chip
  Normal  → 普通一问一答
  Plan    → 先计划→批准→执行（两轮）
  Goal    → 设定目标后自主多轮推进
```

## 6. 下一步

- 阅读 [core-concepts.md](core-concepts.md) 了解核心概念
- 阅读 [project-structure.md](project-structure.md) 了解代码结构
- 阅读 `01-dev-read/` 中的深度分析文档
