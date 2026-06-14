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

---

## 附录 A: Project Checks 详解

### 什么是 Project Checks

Project Checks 是从项目文档中提取的**结构化检查命令**，要求模型在给出最终答案前必须执行。

### 提取来源

命令从项目文档中的 **"Reasonix host checks"** 章节提取，格式为：

```markdown
## Reasonix host checks
- verify: go test ./internal/...
- verify: git diff --check
- verify: go test ./...
```

### 提取逻辑

在 `internal/instruction/instruction.go:38-60` 中：

```go
func ExtractHostChecks(docs []memory.Source) []VerifyCheck {
    // 扫描文档中的 "## Reasonix host checks" 章节
    // 提取 "- verify: <command>" 格式的命令
}
```

### 检查时机

在 `internal/agent/agent.go:735-743` 中：

```go
for _, check := range a.projectChecks {
    command := strings.TrimSpace(check.Command)
    if command == "" {
        continue
    }
    if !a.evidence.HasSuccessfulCommandAfter(command, writer) {
        out.missingProjectChecks++
        missing = append(missing, fmt.Sprintf("run %q from %s after the latest write", 
            command, finalReadinessCheckSource(check)))
    }
}
```

### 关键条件

**必须在最后一次写入后运行这些命令**，否则会被阻止。

### 示例

如果项目文档 `AGENTS.md` 包含：

```markdown
## Reasonix host checks
- verify: go test ./...
- verify: git diff --check
```

那么模型在给出最终答案前，必须：
1. 执行 `go test ./...`
2. 执行 `git diff --check`

否则 Final Readiness Check 会失败，导致模型被阻止并需要重试。

---

## 附录 B: 功能加入时间线

### 项目迁移历史

| 时间 | 提交 | 事件 |
|------|------|------|
| **2026-05-29 17:53** | `32a4c02e` | **v2 初始化** —— ground-up rewrite（完全重写） |
| **2026-05-29 18:13** | `7de6a247` | 导入 Go 实现作为 v2 kernel（从 duo 改名为 reasonix） |
| **2026-06-02 07:26** | `686f0502` | **Project Checks 功能加入** |
| **2026-06-02 08:34** | `5b8d54fc` | **Final Readiness Check 功能加入** |

### 关键发现

1. **项目确实是 v2 版本完全重写**（2026-05-29），从之前的架构迁移到 Go
2. **Final Readiness Check 是在 v2 初始化后仅 4 天加入的**（2026-06-02）
3. **距离 v2 初始化只有 331 个提交**时就加入了这些检查机制

---

## 附录 C: 为什么 TUI 的 Agent 执行质量可能更高

### 可能的原因

#### 1. **TUI 和 Desktop 使用相同的内核**
从代码看，TUI 和 Desktop 都使用 `internal/agent` 包，理论上行为应该一致。但可能有以下差异：

#### 2. **可能的差异点**

| 方面 | TUI | Desktop |
|------|-----|---------|
| **交互模式** | 命令式、即时反馈 | 事件驱动、异步 |
| **上下文管理** | 简单的会话管理 | 复杂的 Tab/Workspace 管理 |
| **Project Checks** | 可能不加载项目文档 | 加载完整的项目文档 |
| **超时设置** | 可能更宽松 | 可能有更严格的超时 |
| **流式输出** | 直接终端输出 | 通过 SSE/WebSocket 转发 |

#### 3. **关键差异：Project Checks 的加载**

在 `internal/boot/boot.go:191` 中：
```go
mem := memory.Load(memory.Options{CWD: root, UserDir: config.MemoryUserDir()})
projectChecks := instruction.ExtractHostChecks(mem.Docs)
```

**如果 TUI 和 Desktop 的 `root` 或 `config.MemoryUserDir()` 不同**，可能导致：
- TUI 没有加载到包含 "Reasonix host checks" 的项目文档
- Desktop 加载了完整的项目文档，从而触发了更多的检查

### 验证建议

如果你想验证 TUI 和 Desktop 的差异，可以检查：

1. **TUI 是否加载了 Project Checks**
   ```bash
   # 在 TUI 运行时检查日志或添加调试输出
   ```

2. **比较两者的 `root` 目录**
   ```go
   // 在 boot.go 中添加日志
   fmt.Printf("DEBUG: root=%s, mem.Docs=%d, projectChecks=%d\n", 
       root, len(mem.Docs), len(projectChecks))
   ```

3. **临时禁用 Project Checks 对比**
   按照附录 D 的方法，将 `ProjectChecks: nil` 后，观察 Desktop 的行为是否更接近 TUI

---

## 附录 D: 最简单的关闭方法

### 方法：直接修改 boot.go

修改 `internal/boot/boot.go` 第 834 行，将 `projectChecks` 改为 `nil` 或空切片：

```go
// 修改前
ProjectChecks:     projectChecks,

// 修改后 - 方法1: 直接传 nil
ProjectChecks:     nil,

// 修改后 - 方法2: 传空切片
ProjectChecks:     []instruction.VerifyCheck{},
```

### 优点

1. **不需要修改配置窗口**（前端代码完全不用动）
2. **不需要新增配置项**
3. **只修改一行代码**
4. **立即生效** —— 所有项目的 "Reasonix host checks" 都会被忽略

### 修改位置

**文件**: `internal/boot/boot.go:834`

### 效果

- `agent.projectChecks` 为空
- `finalReadinessCheck()` 中的 `hasProjectChecks` 为 `false`
- Project Checks 检查被完全跳过
- 只有 Todo 检查（如果存在）会继续进行

### 环境变量方案（更灵活）

如果你想保留配置灵活性，可以添加一个简单的环境变量控制：

```go
// internal/boot/boot.go:834
projectChecksToUse := projectChecks
if os.Getenv("REASONIX_DISABLE_PROJECT_CHECKS") == "1" {
    projectChecksToUse = nil
}
// ... 然后在 Options 中使用 projectChecksToUse
```

这样用户可以通过设置环境变量 `REASONIX_DISABLE_PROJECT_CHECKS=1` 来临时关闭，无需重新编译。

---

## 附录 E: Desktop 配置窗口现状

### 当前 Agent 配置项

在 `desktop/settings_app.go:107-115` 中：

```go
type AgentView struct {
    Temperature       float64 `json:"temperature"`
    MaxSteps          int     `json:"maxSteps"`
    PlannerMaxSteps   int     `json:"plannerMaxSteps"`
    SystemPrompt      string  `json:"systemPrompt"`
    ColdResumePrune   bool    `json:"coldResumePrune"`
    ReasoningLanguage string  `json:"reasoningLanguage"`
}
```

### 缺失的配置项

❌ **没有 Final Readiness Check 的开关**
❌ **没有 Project Checks 的开关**

如果要通过配置窗口控制，需要：
1. 在 `AgentView` 中添加新字段
2. 在 `internal/agent` 中添加配置支持
3. 在前端 Settings 面板中添加 UI
4. 修改前后端绑定代码

**这就是为什么直接修改 `boot.go` 是最简单的方法** —— 它避免了繁琐的配置窗口修改。
