# 项目文档缺口分析报告 & L1-L4 补全计划

<!--
版本: v1.0
创建: 2026-06-14 00:00:00
更新: 2026-06-14 00:00:00
-->

> 基于 46 个 `internal/` 源码包 + Desktop 前端 vs 36 篇现有文档的交叉比对结果。

---

## 一、代码与文档一致性分析

### 1.1 完全覆盖（✅ 21 个包）

| 包 | 现有文档 | 覆盖度 |
|----|---------| :---: |
| `agent/` | 10-Agent Core Loop, 18-Subagent, 04-session, 05-loop, 06-assessment | 90% |
| `boot/` | 15-Boot Sequence | 85% |
| `checkpoint/` | 23-Checkpoint & Rewind | 80% |
| `config/` | 26-Config System | 80% |
| `control/` | 08-modes, 13-context, 20-plan, 21-gate, 24-autoplan | 85% |
| `event/` | 25-Event System | 85% |
| `evidence/` | 22-Evidence & Todo | 80% |
| `hook/` | 14-Hook System | 80% |
| `memory/` | 19-Memory System | 85% |
| `plugin/` | 12-MCP System | 85% |
| `provider/` | 17-Provider System | 75% |
| `sandbox/` | 28-Sandbox | 70% |
| `skill/` | 11-Skill System | 85% |
| `tool/` | 09-Tool System | 90% |

### 1.2 部分覆盖（⚠️ 6 个包）

| 包 | 现有提及 | 缺失内容 |
|----|---------|---------|
| `builtinmcp/` | 12-MCP 一笔带过 | codegraph/lsp 作为内置 MCP 的完整机制 |
| `cli/` | 03-TUI, 05-loop | chat_tui.go 作为完整应用的架构、渲染模型、bubbletea 集成 |
| `codegraph/` | 12-MCP 提及 | 符号索引构建、查询 API、9 个工具的实现 |
| `command/` | capabilities/slash-commands | 自定义命令引擎的加载/渲染/优先级算法 |
| `jobs/` | 13-context 提及 | 后台任务管理器完整的生命周期 |
| `lsp/` | 12-MCP 提及 | gopls 集成、支持的 14 种语言、诊断流程 |

### 1.3 完全空白（❌ 19 个包）

| 包 | 功能 | 影响度 |
|----|------|:---:|
| **`acp/`** | Agent Client Protocol — 标准 Agent 协议 | 🔴 高 |
| **`serve/`** | HTTP/SSE 服务器 — 第三个前端通道 | 🔴 高 |
| **`i18n/`** | 国际化的消息系统（140+ 键值） | 🟡 中 |
| **`diff/`** | diff 渲染引擎 — edit/write/create/delete 可视化 | 🟡 中 |
| **`fileref/`** | 文件引用解析 — `@path`, `/path:line` | 🟡 中 |
| **`history/`** | 会话历史搜索工具 | 🟡 中 |
| **`installsource/`** | install_source 工具 — Skill/MCP 安装引擎 | 🟡 中 |
| `billing/` | 计费/余额查询 | 🟢 低 |
| `bot/` | bot 集成 | 🟢 低 |
| `botruntime/` | bot 运行时 | 🟢 低 |
| `doctor/` | 诊断工具 | 🟢 低 |
| `fileutil/` | 文件工具函数 | 🟢 低 |
| `frontmatter/` | YAML-frontmatter 解析 | 🟢 低 |
| `inspect/` | 能力检视 API | 🟢 低 |
| `instruction/` | 结构化指令验证 | 🟢 低 |
| `mcpdiag/` | MCP 认证诊断 | 🟢 低 |
| `netclient/` | HTTP 客户端代理 | 🟢 低 |
| `notify/` | 桌面通知 | 🟢 低 |
| `outputstyle/` | 输出样式渲染 | 🟢 低 |
| `permission/` | 权限管理 | 🟢 低 |
| `proc/` | 进程管理 | 🟢 低 |
| `retrieval/` | 检索增强 | 🟢 低 |
| `sysproxy/` | 系统代理 | 🟢 低 |

### 1.4 关键空白总结

| 类别 | 缺失数 | 典型 |
|------|:---:|------|
| L1 Overview | **0 篇** | 没有"项目是什么/快速上手"文档 |
| L2 Architecture | **0 篇** | 没有系统架构总图、数据流图 |
| L4 Extension Guide | **0 篇** | 没有"如何新增 Tool/Skill/MCP/Provider"指南 |
| L3 高影响缺口 | **6 个包** | ACP、HTTP Server、i18n、Diff、FileRef、History |

---

## 二、L1-L4 认知金字塔文档树

### 当前 my-docs 在金字塔中的位置

```
        L1 (0%)  ← 完全缺失
       / \
      /   \
     / L2  \  (5%) ← 几乎缺失（仅 06-assessment 部分涉及）
    /-------\
   /   L3    \  (75%) ← 28 篇已覆盖（主要集中在此）
  /-----------\
 /     L4      \  (0%)  ← 完全缺失
/---------------\
```

---

## 三、补全计划 (Proposal)

### L1 — Overview & Concepts（新建目录 `my-docs/10-overview/`）

| # | 文件 | 标题 | 核心大纲 |
|---|------|------|---------|
| 01 | `what-is-reasonix.md` | Reasonix 是什么 | 项目定位、解决什么问题、与 Claude Code/Cursor 的差异、核心设计哲学 |
| 02 | `quick-start.md` | 快速上手 | 5 分钟体验：安装→配置 API key→发送第一条消息→看懂输出 |
| 03 | `core-concepts.md` | 核心概念词汇表 | Agent / Turn / Tool / Skill / MCP / Goal / Plan / Session / Context / Prefix / Steer / Hook / Subagent 等 15+ 核心概念的简明定义 |
| 04 | `project-structure.md` | 项目结构导览 | internal/ 46 个包的分类地图、desktop/cli/serve 三前端的职责边界 |

### L2 — Architecture & Data Flow（新建目录 `my-docs/20-architecture/`）

| # | 文件 | 标题 | 核心大纲 |
|---|------|------|---------|
| 01 | `system-architecture.md` | 系统架构总图 | 三前端(TUI/Desktop/HTTP)→Controller→Agent→Provider 的完整分层图；Boot 启动序列；组件交互矩阵 |
| 02 | `data-flow-turn.md` | 单 Turn 全链路数据流 | 用户输入→submit→Compose→Agent.Run→stream→executeBatch→events→frontend reducer 的完整数据流，含时序图 |
| 03 | `data-flow-skill.md` | Skill 调用全链路 | /skill-name→RunSkill→run_skill tool→inline vs subagent 两条路径的完整数据流 |
| 04 | `data-flow-mcp.md` | MCP 连接与调用全链路 | 配置加载→spec→transport→handshake→tools/list→lazy proxy→tools/call 的完整流程 |
| 05 | `component-interaction.md` | 核心组件交互矩阵 | Controller/Agent/Session/Registry/Provider/Plugin 之间的依赖关系和调用边界 |

### L3 — Mechanisms & Deep Dive（补全现有的 `01-dev-read/`）

#### 新建文档

| # | 文件 | 标题 | 优先级 |
|---|------|------|:---:|
| 29 | `29-acp-protocol.md` | ACP — Agent Client Protocol | 🔴 |
| 30 | `30-http-sse-server.md` | HTTP/SSE Server — 第三个前端通道 | 🔴 |
| 31 | `31-i18n-system.md` | i18n — 国际化消息系统 | 🟡 |
| 32 | `32-diff-engine.md` | Diff Engine — 文件变更渲染 | 🟡 |
| 33 | `33-file-reference.md` | File Reference — @path 与文件引用解析 | 🟡 |
| 34 | `34-history-tool.md` | History Tool — 会话历史搜索 | 🟡 |
| 35 | `35-install-source.md` | Install Source — Skill/MCP 安装引擎 | 🟡 |
| 36 | `36-command-engine.md` | Custom Command Engine — 加载/渲染/优先级 | 🟡 |
| 37 | `37-builtin-mcp.md` | Builtin MCP — CodeGraph + LSP 完整机制 | 🟡 |
| 38 | `38-jobs-manager.md` | Jobs Manager — 后台任务生命周期 | 🟢 |

#### 需补充/重构的现有文档

| 文件 | 问题 | 操作 |
|------|------|------|
| `02-Loop模式深度分析.md` | 与 05-loop-lifecycle 内容重叠 | 合并为 `02-loop-mode.md`，删除 05 |
| `04-Desktop会话持久化.md` | 未覆盖桌面完整架构 | 内容并入 27-frontend-architecture |

### L4 — Extension Guide（新建目录 `my-docs/30-extension-guide/`）

| # | 文件 | 标题 | 核心大纲 |
|---|------|------|---------|
| 01 | `add-builtin-tool.md` | 如何新增一个内置 Tool | 1. 实现 Tool 接口 2. init() 注册 3. 编写测试 4. 示例：新增 `git_status` 工具 |
| 02 | `add-skill.md` | 如何编写一个 Skill | 1. SKILL.md 结构 2. frontmatter 字段 3. inline vs subagent 选择 4. 示例：编写 `code-review` skill |
| 03 | `add-mcp-server.md` | 如何对接一个 MCP 服务器 | 1. reasonix.toml [[plugins]] 配置 2. stdio/http/sse 选择 3. 工具命名规范 4. 调试方法 |
| 04 | `add-provider.md` | 如何新增一个 LLM 提供商 | 1. 实现 Provider 接口 2. 注册到 config 3. 流式处理 4. 示例 |
| 05 | `add-hook.md` | 如何编写一个 Hook | 1. settings.json 结构 2. 10 事件点选择 3. stdin/stdout JSON 契约 4. 阻塞/非阻塞 5. 示例 |
| 06 | `add-custom-command.md` | 如何编写自定义命令 | 1. .reasonix/commands/ 目录 2. frontmatter 3. 模板变量 4. 示例 |
| 07 | `add-frontend-component.md` | 如何扩展 Desktop 前端 | 1. useController reducer 2. Bridge 接口 3. 组件注册 4. 示例 |

---

## 四、实施优先级

```
Phase 1 (立即):    L1 全部 4 篇       ← 没有 Overview 其他文档难以定位
Phase 2 (本周):    L4 Extension Guide  ← 二开者最需要的实践指南
Phase 3 (本周):    L2 Architecture 5 篇 ← 串起已有 L3 文档的骨架
Phase 4 (下周):    L3 补全 10 篇        ← 填补剩余缺口
Phase 5 (持续):    现有文档去重/合并     ← 02+05 合并等
```

### 预计工作量

| 层级 | 新建 | 重构 | 总计行数 |
|------|:---:|:---:|------|
| L1 Overview | 4 | 0 | ~3,000 |
| L2 Architecture | 5 | 0 | ~6,000 |
| L3 Deep Dive | 10 | 2 | ~12,000 |
| L4 Extension Guide | 7 | 0 | ~7,000 |
| **合计** | **26** | **2** | **~28,000** |
