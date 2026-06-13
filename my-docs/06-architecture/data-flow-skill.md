# Skill 调用全链路

<!--
版本: v1.0
创建: 2026-06-14 00:20:00
更新: 2026-06-14 00:20:00
-->

## 两条调用路径

```
路径A — 用户直接调用 (CLI/TUI):
  /review "focus on auth"
    → chat_tui.startTurn → Controller.submit()
      → RunSkill("review") → render template
        → runGuarded → Agent.Run(template)
          → 模型看到 skill body 作为 user message

路径B — 模型通过工具调用:
  run_skill("review", "focus on auth")
    → runSkillTool.Execute()
      ├─ inline: renderInline(sk, args) → <skill-pin>...</skill-pin> → 工具结果
      └─ subagent: runner(ctx, sk, task, opts)
           → SubagentSpec{SystemPrompt: sk.Body, Registry: filtered}
             → Agent.RunSubAgentWithSession()
               → 返回 answer + ref="sa_..."
```

## 全链路时序

```
模型                  run_skill工具        Store           Runner         子Agent        前端
 │                       │                  │               │               │              │
 │─run_skill("review",  │                  │               │               │              │
 │  "focus on auth")──▶ │                  │               │               │              │
 │                       │─Read("review")─▶│               │               │              │
 │                       │◀─Skill{}────────│               │               │              │
 │                       │                  │               │               │              │
 │                       │ subagent分支:    │               │               │              │
 │                       │──runner(ctx,sk, │               │               │              │
 │                       │   task, opts)───┼──────────────▶│               │              │
 │                       │                  │               │─PrepareFresh()│              │
 │                       │                  │               │─RunSubAgent()─▶              │
 │                       │                  │               │               │─Run()        │
 │                       │                  │               │               │─stream()     │
 │                       │                  │               │               │─executeBatch │
 │                       │                  │               │◀─answer───────│              │
 │                       │                  │               │─SaveCompleted │              │
 │                       │◀─answer+ref──────│───────────────│               │              │
 │                       │                  │               │               │              │
 │─ToolResult(answer)◀───│                  │               │               │              │
 │  "Subagent ref: sa_001"                  │               │               │              │
```

## 关键决策点

| 决策 | 判定条件 | 分支 |
|------|---------|------|
| inline vs subagent | `sk.RunAs == RunSubagent` | inline→renderInline / subagent→runner |
| 子代理工具集 | `sk.AllowedTools` 非空? | 白名单 / 继承父工具排除meta-tools |
| 子代理模型 | `sk.Model` + config 覆盖 | per-skill config > frontmatter > 默认 |
| 继续/分叉 | `continue_from` / `fork_from` | PrepareContinue / PrepareFork |
