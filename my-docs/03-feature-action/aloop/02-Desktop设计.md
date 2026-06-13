# aloop — 桌面端设计（仅 Desktop，配置驱动）

<!--
版本: v1.0
创建: 2026-06-14 04:54:29
更新: 2026-06-14 04:54:29
-->

**关联文档**:
- [01-Desktop-Loop-评估.md](01-Desktop-Loop-评估.md) — 前置评估，含背景与触达面扫描
- [00-Hope指令设计草案](../hope/00-设计草案.md)

**设计原则**：仅涉及 Desktop 改造，不改动 TUI `/loop`。只讨论设计，不改代码。

---

## 目录

1. [配置层设计](#一配置层设计)
2. [Controller 层设计](#二controller-层设计)
3. [Desktop 后端设计](#三desktop-后端设计)
4. [Desktop 前端设计](#四desktop-前端设计)
5. [向后兼容与过渡策略](#五向后兼容与过渡策略)
6. [实施清单与顺序](#六实施清单与顺序)
7. [关键决策点](#七关键决策点)

---

## 一、配置层设计

### 1.1 `reasonix.example.toml` — 新增配置项

在 `[agent]` 节新增三个字段：

```toml
[agent]
# -- 新增 --
loop_interval       = 30     # aloop 间隔秒数，≥5；0 表示不启用
loop_max_iterations = 50     # aloop 最大迭代数，达到后自动停止
loop_exit_signal    = ""     # 模型输出中触发退出的精确子串，空表示仅手动停止
```

**退出信号**设计为**精确子串匹配**（非正则），例如：

| 配置值 | 行为 |
|--------|------|
| `""`（默认） | 仅用户手动停止 / 达最大迭代数 |
| `"[loop:complete]"` | 模型输出含 `[loop:complete]` 时自动停止 |
| `"[DONE]"` | 模型输出含 `[DONE]` 时自动停止 |

子串匹配（`strings.Contains`）比正则更简单，配置门槛低，对用户友好。

### 1.2 `internal/config/config.go` — AgentConfig 扩展

```
AgentConfig struct (行 768-801 附近)
├── ...现有字段...
└── + LoopInterval      int    `toml:"loop_interval"`
    + LoopMaxIterations int    `toml:"loop_max_iterations"`
    + LoopExitSignal    string `toml:"loop_exit_signal"`
```

### 1.3 `internal/boot/boot.go` — 注入 Controller 选项

```
boot.go:888-919 构建 ctrlOpts 时
├── ctrlOpts := control.Options{...}
│   └── + LoopInterval:      time.Duration(cfg.Agent.LoopInterval) * time.Second
│       + LoopMaxIterations: cfg.Agent.LoopMaxIterations
│       + LoopExitSignal:    cfg.Agent.LoopExitSignal
```

---

## 二、Controller 层设计

### 2.1 `control.Options` 新增字段

```go
// controller.go:227-276 附近
type Options struct {
    // ... 现有字段 ...
    LoopInterval      time.Duration // aloop 默认间隔（0 = 未配置）
    LoopMaxIterations int           // aloop 默认最大迭代数（0 = 不限）
    LoopExitSignal    string        // 退出信号子串（空 = 不检测）
}
```

### 2.2 `Controller` struct 新增字段

```go
// controller.go:134 附近
type Controller struct {
    // ... 现有字段 ...
    aloopPrompt      string            // 当前 aloop prompt，空=未激活
    aloopInterval    time.Duration     // 触发间隔
    aloopMaxIter     int               // 最大迭代数
    aloopIter        int               // 已执行次数
    aloopActive      bool              // 是否运行中
    aloopExitPattern string            // 退出信号子串（从 Options 传入）
    aloopCancel      context.CancelFunc // 用于停止循环
}
```

### 2.3 新增方法

#### `StartLoop(prompt string, interval time.Duration)`

```
1. 若 aloopActive == true，先调用 StopLoop()
2. 存储 prompt / interval / maxIter / exitPattern
3. 设 aloopActive = true, aloopIter = 0
4. 若 interval == 0，使用 Options.LoopInterval 默认值
5. 创建带取消的 context: ctx, aloopCancel = context.WithCancel(c.baseCtx)
6. c.runGuarded(func(ctx) {
       for aloopActive {
           // 检查取消
           if ctx.Err() != nil { break }
           
           // 检查最大迭代数
           if aloopMaxIter > 0 && aloopIter >= aloopMaxIter {
               c.notice("▸ loop: max iterations reached (" + aloopMaxIter + ")")
               break
           }
           
           // 执行一轮
           aloopIter++
           c.runTurnWithRawDisplay(ctx, aloopPrompt, aloopPrompt, aloopPrompt)
           
           // 检查退出信号（若配置）
           if aloopExitPattern != "" {
               last := lastAssistantText(c.History())
               if strings.Contains(last, aloopExitPattern) {
                   c.notice("▸ loop: exit signal '" + aloopExitPattern + "' detected")
                   break
               }
           }
           
           // 等待间隔（非阻塞睡眠，支持 ctx 取消）
           select {
           case <-ctx.Done():
               break
           case <-time.After(aloopInterval):
               // 继续
           }
       }
       aloopActive = false
       c.notice("▸ loop stopped (after " + aloopIter + " iter)")
   })
```

**关键实现细节**：

- 使用 `time.After` 而非 `time.Ticker`，因为需要每次迭代后检查退出信号再决定是否继续
- `runGuarded` 的 body 内部自己循环——方案 A（推荐），body 内 `for` + `time.After`
- `runTurnWithRawDisplay` 在同一 goroutine 中同步阻塞，turn 完成后继续下一次迭代

#### `StopLoop()`

```
1. 若 aloopCancel != nil，调用 aloopCancel()
2. aloopCancel 取消后，aloopRunner 中的 select 会通过 ctx.Done() 收到信号退出循环
3. 清理字段：aloopPrompt = "", aloopActive = false
```

#### `LoopStatus() LoopStatusInfo`

```go
type LoopStatusInfo struct {
    Active   bool
    Prompt   string
    Interval time.Duration
    Iter     int
    MaxIter  int
}
```

### 2.4 退出信号处理流程

```
                    ┌──────────────────────────┐
                    │      aloop 运行中         │
                    └──────────┬───────────────┘
                               │
                    ┌──────────▼───────────────┐
                    │  runTurnWithRawDisplay()  │ ← 同步阻塞，执行一轮
                    └──────────┬───────────────┘
                               │
                    ┌──────────▼───────────────┐
                    │ 检查 aloopExitPattern?    │
                    │ strings.Contains(reply)   │
                    └──────────┬───────────────┘
                               │
                    ┌──────────▼───────────────┐
           ┌────────┤ 匹配 → c.notice → break  │
           │        └──────────────────────────┘
           │
    ┌──────▼──────┐
    │ 不匹配/未配置│
    └──────┬──────┘
           │
    ┌──────▼──────┐
    │ 检查 iter   │
    │ >= maxIter  │──→ 达到 → c.notice → break
    └──────┬──────┘
           │
    ┌──────▼──────┐
    │ time.After  │  ← 支持 ctx 取消
    │ 等待间隔    │
    └──────┬──────┘
           │
           ▼ 回到循环开始
```

### 2.5 与 Compose 的关系

aloop 发送的 prompt **仍然走 `Compose()`**——plan mode marker、reasoning language、memory update、background jobs 正常注入。

**关键解耦点**：不注入 `<active-goal>` 块。aloop 不依赖模型输出标记来驱动循环，而是依赖**定时器 + 退出信号**。

原 Compose 中的 goal 注入（`input.go:91-93`）：
```go
if strings.TrimSpace(goal) != "" && goalStatus == GoalStatusRunning {
    text = activeGoalBlock(goal) + "\n\n" + text
}
```

此段在 aloop 设计中**可整体删除**（或标记 deprecated）。

---

## 三、Desktop 后端设计

### 3.1 `desktop/app.go` — Wails 绑定

新增三个导出方法，对标现有的 `SetGoal`/`ClearGoal`：

```go
// StartLoop 启动 aloop。
// prompt: 循环发送的消息内容
// intervalSeconds: 间隔秒数（传 0 使用配置默认值）
func (a *App) StartLoop(prompt string, intervalSeconds int) {
    tab := a.currentTab()
    if tab == nil { return }
    interval := time.Duration(intervalSeconds) * time.Second
    if intervalSeconds <= 0 {
        interval = tab.Ctrl.Options().LoopInterval
    }
    tab.Ctrl.StartLoop(prompt, interval)
    a.updateTabMeta(tab.ID, map[string]any{
        "loopActive": true,
        "loopPrompt": prompt,
        "loopInterval": intervalSeconds,
        "loopIter": 0,
    })
}

// StopLoop 停止当前 tab 的 aloop
func (a *App) StopLoop() {
    tab := a.currentTab()
    if tab == nil { return }
    tab.Ctrl.StopLoop()
    a.updateTabMeta(tab.ID, map[string]any{
        "loopActive": false,
    })
}

// LoopStatus 返回当前 aloop 状态
func (a *App) LoopStatus() map[string]any {
    tab := a.currentTab()
    if tab == nil { return nil }
    info := tab.Ctrl.LoopStatus()
    return map[string]any{
        "active":   info.Active,
        "prompt":   info.Prompt,
        "interval": info.Interval.Seconds(),
        "iter":     info.Iter,
        "maxIter":  info.MaxIter,
    }
}
```

### 3.2 TabMeta 变更

`TabMeta` struct（`app.go:1900` 附近）变更：

```go
type TabMeta struct {
    // ... 现有字段 ...
    Goal       string `json:"goal,omitempty"`       // 保留兼容，标记 deprecated
    GoalStatus string `json:"goalStatus,omitempty"` // 保留兼容，标记 deprecated
    
    // -- 新增 --
    LoopActive   bool   `json:"loopActive,omitempty"`
    LoopPrompt   string `json:"loopPrompt,omitempty"`
    LoopInterval int    `json:"loopInterval,omitempty"`
    LoopIter     int    `json:"loopIter,omitempty"`
    LoopMaxIter  int    `json:"loopMaxIter,omitempty"`
}
```

### 3.3 Tab 状态恢复

Desktop 重启或切换 Tab 时，从 `TabMeta` 恢复 aloop 状态：

```go
// app.go Tab 恢复逻辑（~407 行附近）
if tabMeta.LoopActive && tabMeta.LoopPrompt != "" {
    tab.Ctrl.StartLoop(tabMeta.LoopPrompt, time.Duration(tabMeta.LoopInterval)*time.Second)
}
```

> **初版建议**：不持久化 loop 状态（`omitempty` 保证新格式向后兼容）。session 重启后不再自动恢复。

### 3.4 控制器恢复

`newCtrl` 初始化时（`app.go:3471/3567/3634`）注入 loop 配置：

```go
// 在 ctrlOpts 创建时注入
ctrlOpts.LoopInterval = time.Duration(cfg.Agent.LoopInterval) * time.Second
ctrlOpts.LoopMaxIterations = cfg.Agent.LoopMaxIterations
ctrlOpts.LoopExitSignal = cfg.Agent.LoopExitSignal
```

### 3.5 Help 命令列表

在命令帮助列表（`app.go:2051` 附近）新增：

```go
{Name: "aloop", Description: "start/stop/status a timed loop"},
```

---

## 四、Desktop 前端设计

### 4.1 `types.ts` — 类型定义

**CollaborationMode 扩展**：
```typescript
// 行 296 附近
export type CollaborationMode = "normal" | "plan" | "goal" | "loop";
// "goal" 保留为 "loop" 的向后兼容别名
```

**ControllerMeta 新增**：
```typescript
// 行 138-139 附近
export interface ControllerMeta {
    // ... 现有字段 ...
    loopActive?: boolean;
    loopPrompt?: string;
    loopInterval?: number;
    loopIter?: number;
    loopMaxIter?: number;
}
```

**TabMeta 新增**：
```typescript
// 行 292-293 附近
export interface TabMeta {
    // ... 现有字段 ...
    loopActive?: boolean;
    loopPrompt?: string;
    loopInterval?: number;
    loopIter?: number;
    loopMaxIter?: number;
}
```

**LoopStatusInfo 类型**：
```typescript
export interface LoopStatusInfo {
    active: boolean;
    prompt: string;
    interval: number;
    iter: number;
    maxIter: number;
}
```

### 4.2 `bridge.ts` — API 桥接

新增三个 bridge 方法声明：

```typescript
// bridge.ts 行 120-123 附近
StartLoop(prompt: string, intervalSeconds: number): Promise<void>;
StopLoop(): Promise<void>;
LoopStatus(): Promise<LoopStatusInfo>;
```

Mock 实现（参考行 1240-1263 的 goal mock）：

```typescript
// Mock 实现
private mockLoopActive = false;
private mockLoopPrompt = "";
private mockLoopInterval = 0;
private mockLoopIter = 0;
private mockLoopMaxIter = 50;

async StartLoop(prompt, intervalSeconds) {
    this.mockLoopActive = true;
    this.mockLoopPrompt = prompt;
    this.mockLoopInterval = intervalSeconds || 30;
    this.mockLoopIter = 0;
    // 模拟触发 bridge 推送状态
    this.pushTabUpdate({ loopActive: true, loopPrompt: prompt });
}

async StopLoop() {
    this.mockLoopActive = false;
    this.mockLoopPrompt = "";
    this.pushTabUpdate({ loopActive: false });
}

async LoopStatus() {
    return {
        active: this.mockLoopActive,
        prompt: this.mockLoopPrompt,
        interval: this.mockLoopInterval,
        iter: this.mockLoopIter,
        maxIter: this.mockLoopMaxIter,
    };
}
```

### 4.3 `composerProfile.ts` — Profile 字段

```typescript
// 行 18 附近 — ComposerProfileField 扩展
export type ComposerProfileField = "collaborationMode" | "toolApprovalMode" | "tokenMode" | "goal" | "loop";

// 行 24 附近 — ComposerProfile 接口
export interface ComposerProfile {
    // ... 现有字段 ...
    goalDraftMode: boolean;  // 保留，可复用为 loopDraftMode
    loopDraftMode: boolean;  // 或新增
    loopPrompt: string;
    loopInterval: number;
}
```

### 4.4 `App.tsx` — 协作模式切换

**loop 模式的 handleSend 分支**（行 1460 附近）：
```tsx
// 现有 goal 模式检测
if (collaborationMode === "goal" && !goal.trim()) { /* 显示提示 */ }

// 新增 loop 模式检测
if (collaborationMode === "loop") {
    // 如果用户输入了内容，作为 aloop 的 prompt
    if (text.trim()) {
        app.StartLoop(text.trim(), loopInterval);
    }
    return;
}
```

**applyCollaborationMode**（行 1137-1288）新增 loop 分支：
```tsx
// loop 模式：用户需提供 prompt + 间隔
case "loop":
    patchActiveComposerProfile({ collaborationMode: "loop", loopDraftMode: true }, ["collaborationMode", "loop"]);
    break;
```

### 4.5 `Composer.tsx` — UI 元素

**mode chip 新增 loop 菜单项**（行 1553 附近）：
```tsx
// loop 模式切换（参考 goal 的 mode chip）
<button onClick={() => setCollaborationMode("loop")}>
    <span>{t("composer.loopMode")}</span>
    <span>{t("composer.loopModeDesc")}</span>
</button>
```

**loop 激活时的状态指示**。当 `collaborationMode === "loop"` 且有活跃 loop 时：
```tsx
// 输入区下方显示
loopActive ? (
    <div className="composer-loop-bar">
        <span>▸ Loop: every {loopInterval}s · iter {loopIter}/{loopMaxIter}</span>
        <button onClick={app.StopLoop}>Stop</button>
    </div>
) : null
```

**输入 placeholder**：
```tsx
placeholder={
    loopModeOn && !activeLoop
        ? "Input loop prompt, then click start..."
        : t("composer.placeholder")
}
```

### 4.6 `StatusBar.tsx` — 状态栏

```tsx
// 行 260 附近
const loopMode = collaborationMode === "loop" && controllerLoopActive;
// ...
loopMode ? (
    <span className="statusbar__plan" key="loop">
        {t("composer.loopMode")} · {loopIter}/{loopMaxIter}
    </span>
) : null,
```

### 4.7 `styles.css` — 新增样式

```css
.composer-modebar[data-mode="loop"] {
    /* loop 模式的 modebar 样式（可复用 goal 的样式） */
}

.composer-loop-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 8px;
    background: var(--notice-bg);
    border-radius: 4px;
    font-size: 12px;
}
```

### 4.8 前端 locale 文案

```typescript
// 各语言文件新增
"composer.loopMode": "loop mode",
"composer.loopModeDesc": "Set a prompt and interval, then loop.",
"composer.loopModeActiveDesc": "Repeats on a timer until stopped or exit signal detected.",
"composer.loopInputPlaceholder": "Type loop prompt..."
```

---

## 五、向后兼容与过渡策略

### 5.1 兼容清单

| 现有功能 | 过渡策略 |
|----------|----------|
| `collaborationMode: "goal"` | 保留别名，串联到 `"loop"` 相同的 UI 路径 |
| `SetGoal(goal)` Wails API | 保留，内部调用 `StartLoop(goal, 0)`（使用配置默认间隔） |
| `ClearGoal()` | 保留，内部调用 `StopLoop()` |
| `/goal` slash 命令 | 保留，映射到 `/aloop` |
| `goal`/`goalStatus` TabMeta 字段 | 保留填充，从 loop 状态推导（仅填充 `"running"`/`"stopped"`） |
| `<active-goal>` 注入 | **移除**（不在 loop 模式下注入） |
| `[goal:complete/continue/blocked]` 标记 | **弃用**。仅通过 `loop_exit_signal` 配置可选支持 |
| `continueGoal()` 循环引擎 | **整体删除**（被 aloopRunner 替代） |

### 5.2 阶段过渡计划

| 阶段 | 操作 | 代码状态 |
|------|------|----------|
| **Phase 0**（本次） | 新增 aloop 完整链路 | Goal 代码保留，互不干扰 |
| **Phase 1**（下版本） | Desktop 前端 UI 中 goal → loop 映射 | Goal 标记 deprecated |
| **Phase 2**（下下版本） | 删除 Goal 系统（若已无人使用） | 清理代码 |

---

## 六、实施清单与顺序

### 仅 Desktop 视角

| 顺序 | 文件 | 操作 | 复杂度 |
|:----:|------|------|:------:|
| 1 | `internal/config/config.go` | `AgentConfig` +3 字段 | 🟢 低 |
| 2 | `reasonix.example.toml` | `[agent]` +3 配置项 | 🟢 低 |
| 3 | `internal/control/controller.go` | Options +3 字段 / Controller +7 字段 +3 方法 | 🟡 中 |
| 4 | `internal/control/input.go` | Compose 移除 `<active-goal>` 注入 | 🟢 低 |
| 5 | `internal/boot/boot.go` | ctrlOpts 传入 loop 配置 | 🟢 低 |
| 6 | `desktop/app.go` | +StartLoop/StopLoop/LoopStatus 绑定 + TabMeta 更新 | 🟡 中 |
| 7 | `desktop/frontend/src/lib/bridge.ts` | +3 API 声明 + mock | 🟡 中 |
| 8 | `desktop/frontend/src/lib/types.ts` | CollaborationMode 扩展 + 类型更新 | 🟢 低 |
| 9 | `desktop/frontend/src/lib/composerProfile.ts` | +loop 字段 | 🟢 低 |
| 10 | `desktop/frontend/src/App.tsx` | +loop 模式切换 + handleSend 分支 | 🟡 中 |
| 11 | `desktop/frontend/src/components/Composer.tsx` | +loop UI（状态指示 + 停止按钮） | 🟡 中 |
| 12 | `desktop/frontend/src/components/StatusBar.tsx` | +loop 状态标签 | 🟢 低 |
| 13 | `desktop/frontend/src/styles.css` | +loop 样式 | 🟢 低 |
| 14 | `desktop/frontend/src/locales/*.ts` | +loop 文案 | 🟢 低 |

**总计**：14 文件，~300 行新增/修改

### 验证点

| 阶段 | 验证 |
|:----:|------|
| 1-5 | `go build ./...` 编译通过 |
| 6 | Desktop 编译通过 |
| 7-9 | `npm run build` 前端编译通过 |
| 10-14 | Desktop 中 `/aloop 10 check status` 正常运行 |

---

## 七、关键决策点

| 决策 | 选项 | 推荐 |
|------|------|:----:|
| **退出信号匹配** | 「正则 vs 子串」 | **子串**（`strings.Contains`），门槛低 |
| **间隔覆盖** | 「允许运行时覆盖 vs 仅用配置文件」 | **允许**。`StartLoop(prompt, intervalSeconds)` 传 0 用配置文件默认值 |
| **Goal 系统** | 「立即删除 vs 保留 deprecated」 | **保留 deprecated**。先新增 aloop，验证稳定后再清理 |
| **Plan mode 互斥** | 「aloop 时退出 plan mode vs 不处理」 | **退出**。`StartLoop` 中 `c.SetPlanMode(false)` |
| **状态持久化** | 「写入 session tab vs 不持久化」 | **不持久化**。loop 随会话生命周期 |
| **前端 UI** | 「独立 mode chip vs 命令式」 | **先用 `/aloop` 命令**。mode chip 视使用反馈决定是否增加 |
| **前端循环状态轮询** | 「bridge 推送 vs 前端轮询」 | **bridge 推送**。turn 完成后通过 event 推送状态变更 |

---

**本文件仅作设计讨论，不涉及任何代码修改。确认方向后可按实施清单逐文件开发。**
