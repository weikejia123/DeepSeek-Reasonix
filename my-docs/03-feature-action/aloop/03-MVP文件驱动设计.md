# aloop MVP — 文件驱动设计

<!--
版本: v1.0
创建: 2026-06-14 05:07:20
更新: 2026-06-14 05:07:20
-->

**前置说明**：本设计**推翻了** [02-Desktop设计.md](02-Desktop设计.md) 中的全局配置方案。aloop 不需要 TOML 配置、不需要退出信号、不需要特殊的注入行为——它就是「定时重复发送的普通 turn」，和手动输入没有实质性区别。

---

## 一、设计原则

1. **aloop = 定时重复的普通 turn**。plan mode、tool approval 等全部现有机制照常运作，不需要特殊处理
2. **文件驱动**。每个实例是一个 JSON 文件，放在项目根目录的 `.aloop/` 下。Controller 只负责按实例文件启动/停止 goroutine
3. **最小改动**。目标：只改 3 个文件即可运行
4. **不过度设计**。不做退出信号、不做退出正则、不做 Compose 注入、不碰 TOML

---

## 二、文件结构

### 2.1 方案 A（推荐）：每个实例独立文件

```
<project-root>/.aloop/
  a1b2c3d4.json     # 一个实例
  e5f6g7h8.json     # 另一个实例
```

**优点**：
- 并发安全——创建/删除/修改一个实例不影响其他
- 文件系统天然支持增删，无需锁
- 用户可直接在项目目录中 `cat .aloop/*.json` 查看

**缺点**：
- 读取所有实例需遍历目录

### 2.2 方案 B：单文件 `instances.json`

```
<project-root>/.aloop/
  instances.json    # 一个文件管所有实例
```

**优点**：
- 一次读一个文件即可拿到全部状态

**缺点**：
- 写回需对 `instances.json` 加锁
- 文件合并冲突风险

> **设计选择**：方案 A（独立文件）。简单，安全，用户可直接编辑单个文件。

### 2.3 实例 JSON 结构

```json
// .aloop/a1b2c3d4.json
{
  "id": "a1b2c3d4",
  "prompt": "每30秒检查构建状态",
  "interval_seconds": 30,
  "remaining_iterations": 50,
  "enabled": true
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 唯一标识，创建时生成（短 UUID 或 时间戳+随机后缀） |
| `prompt` | string | 每次迭代发送的提示词 |
| `interval_seconds` | int | 迭代间隔，最小 5 秒 |
| `remaining_iterations` | int | 剩余次数，每轮 -1；≤0 自动停止不再启动 |
| `enabled` | bool | 启动时是否自动运行；用户可手动暂停设为 false |

### 2.4 运行时状态（Controller 内存，不落盘）

```json
// 不在 JSON 文件中存储，仅在 Controller 内存中
{
  "active": true,
  "current_iteration": 7,
  "cancel": "context.CancelFunc"
}
```

**为什么 `remaining_iterations` 落盘但 `current_iteration` 不落盘？**
- `remaining_iterations` 是「实例的」持久化属性——用户配置它，每次迭代减少。写回后崩溃恢复时可从正确次数继续
- `current_iteration` 是「当前运行的」状态——重启后归零，不需要恢复

---

## 三、Controller 层设计

### 3.1 新增类型

```go
// internal/control/controller.go 新增

type AloopConfig struct {
    ID                  string `json:"id"`
    Prompt              string `json:"prompt"`
    IntervalSeconds     int    `json:"interval_seconds"`
    RemainingIterations int    `json:"remaining_iterations"`
    Enabled             bool   `json:"enabled"`
}

type aloopInstance struct {
    cfg    AloopConfig
    active bool
    iter   int               // 当前已执行次数（内存中）
    cancel context.CancelFunc
}
```

### 3.2 Controller 新增字段

```go
// Controller struct
loops   map[string]*aloopInstance  // id → 实例
loopsMu sync.Mutex                 // 保护 loops map
```

### 3.3 新增方法

#### `LoadLoops(projectRoot string) ([]AloopConfig, error)`

```text
1. 读取 projectRoot + "/.aloop/" 目录下所有 *.json 文件
2. 逐个 json.Unmarshal → AloopConfig
3. 存入 c.loops map（id → aloopInstance{cfg: config}）
4. 若 cfg.Enabled == true 且 cfg.RemainingIterations > 0 → 自动调用 StartLoop(id)
5. 返回全部 AloopConfig 列表
```

#### `StartLoop(id string) error`

```text
1. c.loopsMu.Lock()；从 c.loops[id] 取实例
2. 若 inst.active → Unlock; return error("already running")
3. 若 inst.cfg.RemainingIterations ≤ 0 → Unlock; return error("no iterations left")
4. ctx, cancel := context.WithCancel(c.baseCtx)
5. inst.active, inst.iter, inst.cancel = true, 0, cancel
6. c.loopsMu.Unlock()
7. c.runGuarded 启动 goroutine（内部循环）：
   for {
       inst.cfg.RemainingIterations--  // 减到负值就停了
       inst.iter++
       // 写入文件（每次迭代后持久化 remaining_iterations）
       writeInstanceConfig(c.loopsDir, inst.cfg)
       
       c.runTurnWithRawDisplay(ctx, inst.cfg.Prompt, inst.cfg.Prompt, inst.cfg.Prompt)
       // ↑ 完全等同普通 turn：Compose、plan mode、tool approval 全部正常运作
       
       if ctx.Err() != nil || inst.cfg.RemainingIterations <= 0 {
           break
       }
       // 等待间隔，支持 ctx 取消
       select {
       case <-ctx.Done(): break
       case <-time.After(time.Duration(inst.cfg.IntervalSeconds) * time.Second):
       }
   }
   inst.active = false
   c.notice("▸ loop stopped: " + id)
```

#### `StopLoop(id string) error`

```text
1. 从 loops[id] 取实例
2. 若 !inst.active → return error("not running")
3. inst.cancel()  → goroutine 中的 ctx.Done() 收到信号，循环退出
4. inst.active = false
```

#### `LoopStatus(id string) (AloopStatus, error)`

```go
type AloopStatus struct {
    Config              AloopConfig `json:"config"`
    Active              bool        `json:"active"`
    CurrentIteration    int         `json:"current_iteration"`
}
```

#### `ListLoops() []AloopStatus`

遍历 `c.loops` map，对每个实例返回状态。

### 3.4 关键行为

| 场景 | 行为 |
|------|------|
| 循环中用户手动发消息 | 走 `runGuarded` 互斥锁，loop goroutine 等待，用户 turn 完成后继续 |
| remaining_iterations 归零 | 自动停止，发 Notice("loop complete") |
| 达到剩余次数后 `StartLoop` | 返回错误 "no iterations left"（用户需手动修改 JSON 增加次数再 Start） |
| 修改 JSON 文件后 | Controller 不会自动感知。需用户调用 `StopLoop(id)` → 改 JSON → `StartLoop(id)` |
| Plan mode 开启时循环 | 每轮迭代都走 plan 流程——先出 plan，批准后执行。和普通 turn 完全一致 |

---

## 四、Desktop 后端设计

### 4.1 `desktop/app.go` — Wails 绑定

```go
// 从 workspace root 推导 .aloop/ 路径
func (a *App) aloopDir() string {
    return filepath.Join(a.workspaceRoot(), ".aloop")
}

// 调用 Controller.LoadLoops()
func (a *App) LoadLoops() ([]control.AloopConfig, error) {
    tab := a.currentTab()
    if tab == nil { return nil, nil }
    return tab.Ctrl.LoadLoops(a.aloopDir())
}

// 启动
func (a *App) StartLoop(id string) error {
    tab := a.currentTab()
    if tab == nil { return errors.New("no active tab") }
    return tab.Ctrl.StartLoop(id)
}

// 停止
func (a *App) StopLoop(id string) error {
    tab := a.currentTab()
    if tab == nil { return errors.New("no active tab") }
    return tab.Ctrl.StopLoop(id)
}

// 状态
func (a *App) LoopStatus(id string) (control.AloopStatus, error) {
    tab := a.currentTab()
    if tab == nil { return control.AloopStatus{}, errors.New("no active tab") }
    return tab.Ctrl.LoopStatus(id)
}
```

### 4.2 实例创建

MVP 不提供前端 UI 来创建实例文件。用户直接在终端创建：

```bash
mkdir -p .aloop
cat > .aloop/monitor.json << 'EOF'
{
  "id": "monitor",
  "prompt": "检查构建状态",
  "interval_seconds": 30,
  "remaining_iterations": 50,
  "enabled": true
}
EOF
```

然后在 Desktop 中调用 `LoadLoops()` 或重启程序即可自动启动。

> **后续**：可在 Desktop 的「设置/工具」面板添加创建/编辑 UI，但 MVP 不做。

---

## 五、前端设计（MVP 跳过）

MVP 阶段**不写前端 UI**。用户通过以下方式与 aloop 交互：

| 操作 | 方式 |
|------|------|
| 创建实例 | 手动创建 `.aloop/<id>.json` |
| 查看状态 | 后续可加简单 panel 或命令行 `cat .aloop/*.json` |
| 启动/停止 | Desktop 自动处理（`enabled: true` 时） |

前端 UI 在后续版本中根据使用反馈决定是否增加。

---

## 六、改动清单

仅在分析，不改代码。

| # | 文件 | 操作 | 复杂度 |
|:-:|------|------|:------:|
| 1 | `internal/control/controller.go` | +AloopConfig 结构体 + aloopInstance + loops map + LoadLoops/StartLoop/StopLoop/LoopStatus/ListLoops | 🟡 中 |
| 2 | `desktop/app.go` | +4 Wails 绑定方法 + aloopDir() | 🟢 低 |
| 3 | `desktop/frontend/src/lib/bridge.ts` | +4 API 声明（mock 可选） | 🟢 低 |

**共 3 个文件**，~200 行新增代码。不改：
- ❌ 不改 `config.go` / `reasonix.toml`
- ❌ 不改 `boot.go`
- ❌ 不改 `Compose()` / `input.go`
- ❌ 不改 TUI
- ❌ 不改前端 UI

---

## 七、设计决策

**决策 1：单文件 vs 独立文件**

| 方案 | 并发安全 | 用户可编辑 | 遍历开销 |
|:----:|:--------:|:----------:|:--------:|
| 独立文件 | ✅ 天然安全 | ✅ `cat/echo/vim` 直接操作 | 一次 `os.ReadDir` |
| instances.json | ⚠️ 需加锁 | ❌ 编辑有冲突风险 | 一个 `os.ReadFile` |

→ **推荐独立文件**。简单就是最好的。

**决策 2：remaining_iterations 写入时机**

| 方案 | 崩溃安全 | 复杂度 |
|:----:|:--------:|:------:|
| 每次迭代后写回 | ✅ 保证次数准确 | 增一次 fsync |
| 仅在 stop 时写回 | ❌ 崩溃后次数丢失 | 简单但不可靠 |

→ **推荐每次迭代后写回**。`remaining_iterations` 是用户配置的"预算"，不能丢失。

---

## 八、不需要的

和之前的设计方案（02-Desktop设计.md）相比，本次 MVP **明确不做**：

| 之前的设计 | MVP 不做 |
|-----------|----------|
| TOML 全局配置 `loop_interval / loop_max_iterations / loop_exit_signal` | ❌ 每个实例自己配置 |
| 退出信号（`[loop:complete]` 子串检测） | ❌ `remaining_iterations ≤ 0` 自动停止 |
| Compose 注入 / `<active-goal>` 修改 | ❌ aloop 走普通路径 |
| Plan mode 互斥 | ❌ plan 正常运作 |
| 前端 UI（Composer / StatusBar / TabBar） | ❌ 手动创建 JSON |
| StatusBar 状态标识 | ❌ 后续版本 |
| styles.css 新增样式 | ❌ 后续版本 |

---

**本文件仅作设计讨论，不涉及任何代码修改。确认方向后可按清单逐文件实现。**
