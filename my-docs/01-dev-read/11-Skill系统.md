# Skill System — 技能发现、加载与执行

<!--
版本: v1.0
创建: 2026-06-13 23:15:00
更新: 2026-06-13 23:15:00
-->

**能力类型**: 系统架构文档  
**涉及包**: `internal/skill/`、`internal/boot/boot.go:537-718`

## 架构概览

Skill 系统将可复用的 playbook 暴露为模型可调用的工具。分三层：

```
发现层  skill.go       — 扫描 .reasonix/skills/ 目录, 解析 frontmatter
索引层  index.go       — 仅 name+description 进入缓存稳定的 system prompt
执行层  tools.go       — run_skill/read_skill/install_skill 工具
运行层  boot.go        — SubagentRunner 连接 skill 与 agent 子代理
```

---

## 一、Skill 数据结构

```go
// skill.go:54-66
type Skill struct {
    Name         string    // 标识符: "review", "explore"
    Description  string    // 一行描述，显示在 # Skills 索引中
    Body         string    // 完整 Markdown playbook
    Scope        Scope     // project | custom | global | builtin
    Path         string    // 源文件绝对路径
    AllowedTools []string  // subagent 白名单（可选）
    RunAs        RunAs     // inline | subagent
    Model        string    // 可选模型覆盖
    Effort       string    // 可选 effort
}
```

### 优先级（同名冲突）

```
project (.reasonix/skills/) > custom paths > global (~/.reasonix/skills/) > builtin
```

---

## 二、Store 发现机制

`skill.New(opts)` 创建 `Store`，`List()` 扫描所有根目录：

```
扫描路径（优先级降序）:
  1. <project>/.reasonix/skills/      (project)
  2. <project>/.agents/skills/        (project)
  3. <project>/.agent/skills/         (project)
  4. <project>/.claude/skills/        (project)
  5. 自定义路径                        (custom)
  6. ~/.reasonix/skills/              (global)
  ...同上 .agents/.agent/.claude       (global)
  7. 内置 skills                       (builtin, 最后优先级)
```

### 目录布局

```
.reasonix/skills/
  review/
    SKILL.md              ← 目录式 skill
    references/           ← 可选引用（Anthropic Skills 兼容）
  my-skill.md             ← 扁平式 skill（单文件）
```

### 解析过程 (`parse()`)

1. 读取 `.md` 文件
2. 分离 YAML-like frontmatter
3. 提取 `name`、`description`、`runas`、`allowed-tools`、`model`、`effort`
4. 加载 `references/*.md` 追加到 body（目录式 skill 才有）
5. `runas` 判定：`runAs: subagent` 或 `context: fork` 或含 `agent:` 字段 → subagent

---

## 三、前缀索引

`index.go` 的 `ApplyIndex(sysPrompt, skills)` 将技能索引注入 system prompt：

```markdown
# Skills — playbooks you can invoke

One-liner index. Before non-trivial work, scan it: ...
- 1password — Set up and use 1Password CLI (op). Use when installing...
- Code — Coding workflow with planning, implementation, verification...
- Memory — Infinite organized memory that complements your agent's...
  ...
```

**关键设计**：只含 name + description + tag（`[🧬 subagent]`）。Body 按需加载（通过 `run_skill`），不进入缓存稳定的 system prefix。

---

## 四、执行模式

### 4.1 Inline Skill (`runAs: inline`)

```
run_skill("my-skill", "args")
  → store.Read("my-skill")
  → renderInline(sk, args)
      → <skill-pin name="my-skill">
          # Skill: my-skill
          > description
          (scope: project · /path/to/SKILL.md)
          
          <playbook body>
          
          Arguments: args
        </skill-pin>
  → 返回给模型作为工具结果
  → 模型在同 turn 内阅读并遵循
```

`<skill-pin>` 标签告诉 compaction 保留内容原样（不压缩摘要）。

### 4.2 Subagent Skill (`runAs: subagent`)

```
run_skill("review", "focus on auth changes")
  → store.Read("review")
  → runner(ctx, sk, "focus on auth changes", opts)
      → 构建 SubagentSpec
          SystemPrompt: sk.Body
          Registry: FilterRegistry(parent, sk.AllowedTools) // 排除 meta-tools
          Model/Effort: sk.Model/Effort 或配置覆盖
      → agent.RunSubAgentWithSession()
          Run(ctx, task)  ← 子代理在隔离 session 中完整执行
      → return answer + ref (sa_...)
  → 工具结果: "\n\nSubagent reference: sa_..."
```

子代理的 tool calls + reasoning 不进入父代理上下文，只返回**最终蒸馏答案**。

### 4.3 Continuation / Fork

```go
run_skill("review", "check the fix", continue_from="sa_001")
run_skill("review", "alternative approach", fork_from="sa_001")
```

- `continue_from` — 在同一条子代理 session 中继续（继承上下文）
- `fork_from` — 从某个子代理运行分叉，独立继续

---

## 五、内置 Subagent 技能

4 个内置 subagent 技能有独立顶层工具（`tools.go:266-303`）：

| 工具 | 技能 | 用途 |
|------|------|------|
| `explore` | explore | 只读代码探索，返回带 file:line 引用的蒸馏答案 |
| `research` | research | web_fetch + 代码阅读组合 |
| `review` | review | diff 审查（正确性/安全/测试缺失） |
| `security_review` | security-review | 专项安全审查 |

这些工具注册时检查技能是否存在（用户可能禁用了），不存在则跳过注册。

---

## 六、与 Controller 的集成

### `/skill-name` 命令路由

```go
// controller.go:872-877
if sent, ok := c.RunSkill(trimmed); ok {
    c.runGuarded(func(ctx context.Context) error {
        return c.runGoalLoopWithRawDisplay(ctx, sent, sent, display)
    })
    return
}
```

用户输入 `/<skill-name>` → 模板展开 → 作为普通 turn 提交（无论 inline 还是 subagent）。

### `/skills` 管理指令

- `/skills` — 列出现有技能
- `/skills enable <name>` — 启用
- `/skills disable <name>` — 禁用（需重启 session 生效）

---

## 七、代码索引

| 文件 | 内容 |
|------|------|
| `internal/skill/skill.go` | Skill 结构、Store、发现/解析/List/Read/Create |
| `internal/skill/index.go` | 前缀索引注入 |
| `internal/skill/tools.go` | run_skill/read_skill/install_skill + 子代理包装工具 |
| `internal/skill/builtins.go` | 内置技能定义 |
| `internal/boot/boot.go:543-618` | `skillRunner` — 子代理执行器 |
| `internal/boot/boot.go:708-718` | `addSkillTools` — 注册时机 |
| `internal/control/controller.go:872-877` | RunSkill 命令路由 |
