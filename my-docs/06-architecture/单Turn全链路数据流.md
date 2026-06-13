# 单 Turn 全链路数据流

<!--
版本: v1.0
创建: 2026-06-14 00:18:00
更新: 2026-06-14 00:18:00
-->

## 时序图

```
用户          Controller       Agent           Provider        Tool           Frontend
 │               │               │               │               │               │
 │──输入────────▶│               │               │               │               │
 │               │─submit()      │               │               │               │
 │               │─Compose()     │               │               │               │
 │               │               │               │               │               │
 │               │──Run(input)──▶│               │               │               │
 │               │               │─TurnStarted──────────────events─────────────▶│
 │               │               │─session.Add() │               │               │
 │               │               │               │               │               │
 │               │               │──stream()────▶│               │               │
 │               │               │               │─Chat(req)     │               │
 │               │               │◀─chunks───────│               │               │
 │               │               │─Text/Reasoning──────────────────────────────▶│
 │               │               │─Usage──────────────────────────────────────▶│
 │               │               │               │               │               │
 │               │               │─executeBatch()│               │               │
 │               │               │─ToolDispatch────────────────────────────────▶│
 │               │               │               │──Execute()──▶│               │
 │               │               │               │◀─result──────│               │
 │               │               │─ToolResult──────────────────────────────────▶│
 │               │               │─session.Add() │               │               │
 │               │               │               │               │               │
 │               │               │──stream()────▶│ (第2轮)       │               │
 │               │               │◀─answer───────│               │               │
 │               │               │               │               │               │
 │               │               │─return nil    │               │               │
 │               │               │─TurnDone────────────────────────────────────▶│
 │               │◀──────────────│               │               │               │
 │               │               │               │               │               │
 │               │autosave       │               │               │               │
 │◀──done────────│               │               │               │               │
```

## 关键数据结构在流程中的变化

```
输入阶段:
  input = "分析项目结构"
  ↓ Compose()
  input = "<active-goal>...</active-goal>\n分析项目结构"  (如果goal激活)
  input = "[Plan mode...]\n<active-goal>...\n分析项目结构" (如果plan激活)

Agent.Run 阶段:
  session.Messages:
    [0] system: "You are..."
    [1] user:   "分析项目结构"
    [2] assistant: "I'll start by..." + toolCalls[{read_file, "go.mod"}]
    [3] tool:   "module github.com/..."   (read_file 结果)
    [4] assistant: "The project is..."    (final answer)

事件流:
  TurnStarted → Text("I'll") → Text(" start") → ToolDispatch(read_file)
  → ToolResult("module...") → Text("The") → Text(" project") → TurnDone
```

## 事件时序优化

```
只读工具批次 (并行):
  ToolDispatch(grep)  }
  ToolDispatch(glob)  }── 并行执行 ──→ ToolResult(grep)
                       }            → ToolResult(glob)

混合批次 (串行):
  ToolDispatch(edit_file) ─→ ToolResult ─→ ToolDispatch(read_file) ─→ ToolResult

complete_step 单列:
  ToolDispatch(complete_step) ─→ (等待前置 receipt 就绪) ─→ ToolResult
```
