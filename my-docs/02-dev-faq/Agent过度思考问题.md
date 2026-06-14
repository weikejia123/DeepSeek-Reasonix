# 开发 FAQ - Agent 过度思考问题

> **作者**: Wake.J  
> **团队**: DZSOFT  
> **更新时间**: 2026-06-14 15:30:00

---

## 问题描述

**现象**: 感觉 Agent 会过度思考、反复思考，在给出最终答案前经历多次循环。

**根本原因**: Agent 内部实现了多层防护机制，在特定条件下会阻止模型结束并强制继续思考。

---

## 防护机制详解

### 1. Storm Breaker（失败循环防护）

**位置**: `internal/agent/agent.go:1220-1240`

**逻辑**: 当同一个工具以相同方式失败 3 次时，阻止继续尝试。

```go
const stormBreakThreshold = 3
```

**设计意图**: 检测模型"参数换皮、错误依旧"的死循环模式。模型会表面修改参数（cosmetically），但失败本质不变。

**合理性**: ✅ **合理**
- 给模型 2 次自我纠正机会
- 符合"两次自然纠正健康，第三次是死亡螺旋"的观察

**潜在问题**:
- 如果错误信息包含随机元素（如时间戳），签名会不同，无法触发保护
- 3 次失败消耗 tokens 较多

---

### 2. Repeat Success Blocker（重复成功防护）

**位置**: `internal/agent/agent.go:1440-1460`

**逻辑**: 当同一个写操作成功 2 次后，第 3 次会被阻止。

```go
const repeatSuccessBreakThreshold = 2
```

**触发条件**: 仅限写操作（write_file/edit_file/multi_edit/notebook_edit/bash）且参数相同。

**合理性**: ✅ **合理且保守**
- 防止模型无意义重复相同的写操作

**潜在问题**:
- 阈值 2 可能过于激进，某些合法场景需要重复操作（如追加内容到同一文件）
- 只检查工具名和参数，不检查文件实际状态变化

---

### 3. Final Readiness Check（最终答案就绪检查）⭐

**位置**: `internal/agent/agent.go:617-660`

**逻辑**: 当模型尝试给出最终答案时，检查是否有未完成的 todo 或未执行的 project checks。

```go
const maxFinalReadinessBlocks = 3
```

**检查内容**:

| 检查项 | 条件 | 触发阻止 |
|--------|------|----------|
| 未完成 Todos | `!planMode && hasTodos && len(incomplete) > 0` | 是 |
| Project Checks | `hasProjectChecks && !HasSuccessfulCommandAfter(command, writer)` | 是 |

**Plan Mode 的影响**:
- Plan Mode 开启时，会跳过 todo 检查
- 但 project checks 仍然会进行

**合理性**: ⚠️ **可能过度干预**

**关键代码**:
```go
if readiness.reason != "" {
    finalReadinessBlocks++
    result := evidence.ReadinessBlocked
    // ... 阻止并继续循环
    continue
}
```

**潜在问题**:
- 强制要求模型执行特定的"项目检查"命令，增加额外步骤
- 如果 projectChecks 不为空，模型必须在最后一次写入后运行这些命令

**配置开关**: ❌ **目前没有直接的配置开关**

**间接禁用方法**:
1. 开启 Plan Mode（跳过 todo 检查）
2. 确保 `projectChecks` 为空且没有 todo

---

### 4. Empty Final Blocks（空答案检测）

**位置**: `internal/agent/agent.go:630-640`

**逻辑**: 如果模型输出空的最终答案，会被阻止并重试（最多 3 次）。

```go
if !hasVisibleFinalAnswer(text) {
    emptyFinalBlocks++
    if emptyFinalBlocks >= maxEmptyFinalBlocks {  // max=3
        return fmt.Errorf("model finished without a visible final answer")
    }
    // ... 重试
    continue
}
```

**合理性**: ✅ **合理**

**潜在问题**:
- 如果模型确实没有内容要输出（如只是确认收到），强制重试 3 次浪费 tokens

---

### 5. Executor Handoff Nudge（执行器无行动催促）

**位置**: `internal/agent/agent.go:640-650`

**逻辑**: Coordinator 模式下，如果执行器没有使用任何工具就回答，会被阻止并提示使用工具。

**阈值**: 仅 1 次，非常克制。

**合理性**: ✅ **合理且克制**

---

### 6. Stream Recovery（流恢复）

**位置**: `internal/agent/agent.go:580-590`

**逻辑**: 流被中断时自动重试（最多 1 次）。

```go
if interrupted && streamRecoveries < maxStreamRecoveries {  // max=1
    streamRecoveries++
    step-- // 不计入步数
    continue
}
```

**合理性**: ✅ **合理**

---

## 总体评估

### 设计优点

1. **分层防护**: 从软提示（nudge）到硬阻止（block）到错误返回，层次分明
2. **签名智能**: Storm Breaker 使用 `(tool, error)` 而非 `(tool, args)` 作为签名，避免被模型的"参数换皮"欺骗
3. **状态隔离**: 各计数器独立，互不干扰

### 可能的问题

| 问题 | 影响 | 建议 |
|------|------|------|
| Final Readiness Check 过于严格 | 强制模型执行额外的检查命令 | 考虑降低优先级或改为可选 |
| 阈值偏保守 | 3 次失败/空答案才阻止，消耗 tokens | 考虑根据模型能力动态调整 |
| 缺乏反馈学习 | 每次会话独立计数，不积累历史 | 可考虑记录常见死循环模式 |
| 用户感知"过度思考" | 多个机制叠加，模型可能在边界反复 | 优化提示词，减少不必要的重试 |

---

## 优化建议

### 方案 1: 添加 Final Readiness Check 配置开关

在 `Options` 结构体中添加：

```go
type Options struct {
    // ... 现有字段
    DisableFinalReadinessCheck bool  // 新增：禁用最终就绪检查
}
```

在 `finalReadinessCheck()` 函数开头添加：

```go
func (a *Agent) finalReadinessCheck() finalReadinessCheck {
    if a.opts.DisableFinalReadinessCheck {
        return finalReadinessCheck{}  // 直接跳过
    }
    // ... 原有逻辑
}
```

### 方案 2: 调整阈值

| 机制 | 当前阈值 | 建议阈值 | 理由 |
|------|----------|----------|------|
| Storm Breaker | 3 | 2 | 减少 tokens 消耗 |
| Empty Final Blocks | 3 | 2 | 更快失败 |

### 方案 3: 优化提示词

在系统提示词中明确告知模型：
- 如果工具调用失败，先分析原因再重试
- 避免无意义的重复操作
- 最终答案前确保所有检查通过

---

## 相关代码位置

| 机制 | 文件位置 | 关键函数/常量 |
|------|----------|---------------|
| Storm Breaker | `internal/agent/agent.go` | `applyStormBreaker()`, `stormBreakThreshold` |
| Repeat Success | `internal/agent/agent.go` | `applyRepeatSuccessBlocker()`, `repeatSuccessBreakThreshold` |
| Final Readiness | `internal/agent/agent.go` | `finalReadinessCheck()`, `maxFinalReadinessBlocks` |
| Empty Final | `internal/agent/agent.go` | `hasVisibleFinalAnswer()`, `maxEmptyFinalBlocks` |
| Executor Handoff | `internal/agent/agent.go` | `maxExecutorHandoffNudges` |
| Stream Recovery | `internal/agent/agent.go` | `maxStreamRecoveries` |

---

## 结论

这些机制**整体设计是合理的**，它们解决了真实的模型行为问题（死循环、空答案、重复操作）。用户感知的"过度思考"主要来源于：

1. **Final Readiness Check** 强制要求额外的验证步骤
2. **多个机制叠加**时，模型在边界条件反复试探
3. **提示词设计**可能鼓励模型"再试一次"而非"改变策略"

建议优先优化 Final Readiness Check 的可配置性，并考虑降低部分阈值。
