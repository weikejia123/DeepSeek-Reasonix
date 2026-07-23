# AGENT-README: DeepSeek-Reasonix

> **分析版本**: V1-20260724
> **分析框架**: [ANALYSIS-DIR-CODER-AGENT.md](../../../ANALYSIS-DIR-CODER-AGENT.md) (PV1-20260724)
> **项目路径**: `/Users/weikejia/CODE/my-agent-group/projects/coder-agent/DeepSeek-Reasonix/`

---

## 1. 身份与定位

| 指标 | 内容 |
|------|------|
| **名称** | DeepSeek-Reasonix |
| **Tagline** | "A DeepSeek-native AI coding agent for your terminal." |
| **定位语** | DeepSeek 原生的终端 AI Coding Agent — 单一 Go 静态二进制, 配置+插件驱动 |
| **设计哲学** | DeepSeek 前缀缓存优先 (cache-stable 系统提示词, 降低成本); 配置驱动 (一切可配置, 无硬编码); CGO_ENABLED=0 单二进制 |
| **许可证** | MIT |
| **运行时语言** | Go |
| **构建系统** | Go toolchain + Makefile (`make build`, `make cross`) |
| **上游** | `github:esengine/DeepSeek-Reasonix` |
| **发布方式** | npm (`reasonix`) + Homebrew + GitHub Release (6 平台) + Desktop + VS Code 扩展 |
| **社区** | Discord, GitHub, AtomGit (中国镜像), XiaoHongShu |

**核心权衡**: Go 的简洁性与 DevOps 友好 vs TypeScript 生态; DeepSeek 前缀缓存优化 vs 通用 Provider 性能; 配置驱动灵活 vs 学习成本

---

## 2. 核心架构

| 指标 | 内容 |
|------|------|
| **运行时** | Go 原生 (CGO_ENABLED=0 单二进制) |
| **包结构** | `internal/` (核心), `cmd/` (CLI), `desktop/` (Wails 桌面), `npm/` (npm 发布); 每个 package 单一关注点 |
| **核心包** | `control.Controller` (传输无关控制器), `internal/tool/` (工具), `internal/provider/` (Provider), `internal/boot/` (启动), `internal/config/` (TOML 配置) |
| **Agent 主循环** | 控制器驱动: 任意 frontend (TUI/HTTP-SSE/Desktop) → `control.Controller` → 工具执行 → Provider 调用 → 循环 |
| **系统提示词构建** | 配置驱动的系统提示词; REASONIX.md 层级加载 (项目/用户/全局); 注意 cache-stable 前缀一致性 |
| **上下文注入文件** | `REASONIX.md` (项目级), `REASONIX.local.md` (个人, gitignored), `~/.config/reasonix/REASONIX.md` (用户级), 祖先目录遍历; `AGENTS.md` fallback |
| **状态管理** | 待确认 (Go 内部状态) |

**架构亮点**: 传输无关的 Controller 设计 — 同一控制器后端驱动所有前端 (TUI/HTTP-SSE/Desktop); 缓存稳定的系统提示词前缀设计

---

## 3. Provider 与模型支持

| 指标 | 内容 |
|------|------|
| **原生 Provider** | DeepSeek (预设); 通过 `reasonix.toml` 配置任意 OpenAI 兼容端点 |
| **双模型架构** | 可选 executor + planner 双模型组合 (独立缓存稳定的会话) |
| **鉴权方式** | API Key (通过 `reasonix.toml` 配置) |
| **自定义 Provider** | `reasonix.toml` 配置驱动 — 无硬编码模型; 任何 OpenAI 兼容端点 = 配置 entry |
| **本地模型** | 通过 OpenAI 兼容端点配置 (Ollama/vLLM 等) |
| **Provider 特定优化** | DeepSeek 自动前缀缓存优化 (系统提示词字节稳定); 无关工具输出剪裁/总结后压缩; 内建工具 schema 合约文档 |
| **模型选择** | `/model` 命令切换 |

**Provider 丰富度**: ★★★☆☆ (DeepSeek 原生, 其他通过 OpenAI 兼容配置; Provider 数量取决于外部配置)

---

## 4. 工具系统

| 指标 | 内容 |
|------|------|
| **内置工具** | 编译时自注册 (Go `init()` 模式) |
| **工具分类** | `internal/tool/builtin/` (内置), 工具合约文档 (`TOOL_CONTRACT.md`) |
| **工具注册机制** | 编译时 `init()` 自注册; 插件通过外部子进程 (stdio JSON-RPC) |
| **MCP 支持** | 通过 Plugin 系统: 外部工具作为子进程运行 (stdio JSON-RPC, MCP 兼容) |
| **工具权限模型** | 待确认 |
| **第三方工具** | Plugin 驱动 (子进程 stdio JSON-RPC); 可复用任何 MCP Server |
| **工具/插件合约** | `TOOL_CONTRACT.md` 文档化工具与建模的接口合约 |

**工具生态成熟度**: ★★★☆☆ (编译时注册 + MCP兼容插件, 但工具数量依赖于社区插件)

---

## 5. 用户界面与交互

| 指标 | 内容 |
|------|------|
| **TUI 技术栈** | Go TUI (具体库待确认) |
| **交互模式** | 交互式 TUI / Headless (`reasonix run`) / ACP (VS Code) / HTTP-SSE / Desktop (Wails) |
| **桌面应用** | Wails 桌面端 (macOS DMG, Windows exe, Linux deb/tar.gz) |
| **VS Code 扩展** | Visual Studio Marketplace + Open VSX Registry |
| **编辑器功能** | 标准 CLI 编辑; `/init` 自动创建项目指令 |
| **快捷键系统** | 待确认 |
| **主题系统** | 待确认 (配置驱动) |
| **ACP** | 支持 (VS Code 集成) |

**UI 亮点**: 最多前端形态 — TUI / Desktop / VS Code / HTTP-SSE; 但每个前端的 TUI 丰富度不如 Rust 或 Ink 方案

---

## 6. 会话管理

| 指标 | 内容 |
|------|------|
| **存储格式** | 待确认 (Go 内部格式) |
| **存储位置** | `~/.config/reasonix/` |
| **分支能力** | Checkpoints + Rewind (`CHECKPOINTS.md`) |
| **Fork/Clone** | 待确认 |
| **Compaction** | Cache-aware 上下文维护; 陈旧工具输出自动剪枝; 总结压缩 |
| **Session Resume** | 待确认 |
| **Bot Guide** | 支持无头机器人模式 (`BOT_GUIDE.md`) |

**会话管理成熟度**: ★★★☆☆ (Checkpoint/rewind 是亮点, 但树状分支/Fork 不如 pi/jcode 成熟)

---

## 7. 定制化与生态

| 指标 | 内容 |
|------|------|
| **Skills** | 配置驱动 (Skills 继承项目指令) |
| **Prompt Templates** | 通过 REASONIX.md 层级体系实现 |
| **Extensions/Plugins** | Plugin 系统: 外部子进程 stdio JSON-RPC (MCP 兼容) |
| **Themes** | 配置驱动 (TOML) |
| **包管理系统** | 无 — 通过 `reasonix.toml` 配置引用 |
| **自修改能力** | 待确认 |

**生态成熟度**: ★★★☆☆ (配置驱动但无扩展 API, Plugin 依赖外部进程)

---

## 8. 记忆与上下文

| 指标 | 内容 |
|------|------|
| **长期记忆** | `MEMORY.md` 索引 + 自动记忆存储 (frontmatter files); `#<note>` 快速添加; `remember` 工具写入持久化事实 |
| **项目上下文** | REASONIX.md 层级体系 (项目/个人/用户/祖先/全局); AGENTS.md fallback; `@path` 文件导入 |
| **会话间复用** | 自动记忆存储 (per-project), 下次会话前缀自动加载 |
| **上下文窗口管理** | Cache-stable 前缀 (系统提示词字节不变); 工具输出剪裁/总结压缩 |
| **记忆工具** | `remember` 工具用于存储持久化事实; `@path` 语法导入外部文件 |

**记忆能力**: ★★★☆☆ (无向量嵌入, 但自动记忆存储 + 前缀加载的设计高效实用, 且与 DeepSeek 缓存策略匹配)

---

## 9. 差异化功能

| 功能 | 支持情况 | 说明 |
|------|---------|------|
| **多 Agent 编排** | ✅ Subagent | Subagent profiles (`SUBAGENT_PROFILES.md`); 独立缓存稳定的会话 |
| **目标驱动工作流** | ⚠️ 基本 | 通过 Subagent 组合实现 |
| **Computer Use** | ❌ 无 | — |
| **浏览器自动化** | ❌ 无 | — |
| **语音模式** | ❌ 无 | — |
| **远程控制** | ❌ 无 | (但 Desktop 和 VS Code 可算本地远程) |
| **制品托管** | ❌ 无 | — |
| **监控** | ❌ 无 | — |
| **桌面应用** | ✅ Wails | 独立桌面 GUI 应用 |
| **VS Code 扩展** | ✅ 完整 | Marketplace + Open VSX; 原生编辑器集成 |
| **Subagent 双模型** | ✅ 独特 | Executor + Planner 独立模型组合 |
| **检查点回退** | ✅ Checkpoints | `CHECKPOINTS.md` 文档化 |
| **恢复与安全模式** | ✅ Recovery | `RECOVERY.md` 灾难恢复 + Safe Mode |

**差异化定位**: DeepSeek 原生优化 + 双模型 + 最多部署形态 (CLI + Desktop + VS Code); 成本优先 (前缀缓存)

---

## 10. 性能

| 指标 | 数据 | 来源 |
|------|------|------|
| **启动耗时** | 待测量 | 未在公开数据中找到基准 |
| **内存 (1 会话)** | 待测量 | 未在公开数据中找到基准 |
| **内存 (多会话)** | 待测量 | — |
| **构建方式** | Go build (`CGO_ENABLED=0`) | — |
| **二进制体积** | 单二进制 (Go 静态编译) | — |

**性能评级**: ★★★★☆ (Go 原生推测: 快于 Node/Bun, 慢于 Rust; 单二进制部署友好)

---

## 11. 安全

| 指标 | 内容 |
|------|------|
| **权限模型** | 待确认 |
| **沙箱支持** | 无文档化方案 |
| **供应链安全** | Go modules + go.sum; CGO_ENABLED=0 减少依赖 |
| **配置安全** | 通过 TOML 配置管理 API key |

**安全评级**: ★★★☆☆ (Go 标准安全, 无特别加强)

---

## 12. 开发与社区

| 指标 | 内容 |
|------|------|
| **测试框架** | Go test |
| **测试策略** | `gofmt -w`, `go vet ./...`, `go test ./internal/tool/builtin/ ./internal/boot/`; golangci-lint (CI) |
| **CI/CD** | GitHub Actions (ci.yml); 中国镜像 (AtomGit) |
| **社区模式** | 开放 — MIT, 多语言 (中/英), 活跃 Discord + XiaoHongShu |
| **代码质量** | Go 包注释规范 (单一关注点); import cycle 检测; PR metadata 规范 (Cache-impact 标记) |
| **发布工程** | GitHub Release + npm + Homebrew + Desktop; 6 平台交叉编译 |

**社区活跃度**: ★★★★☆ (中英文双语社区, 多部署渠道, 中国的 AI Agent 社区关注度高)

---

## 版本演进

### V1-20260724 — 初始分析
- **分析范围**: 基于 Reasonix 源码 + README + REASONIX.md + 文档目录的全面分析
- **数据来源**: README.md, REASONIX.md, 多篇 docs/ 文档, 公开资料
- **关键发现**: DeepSeek 原生缓存优化是整个设计的核心; 传输无关 Controller 是架构亮点; 部署形态最丰富
- **分析框架**: 12 维度, PV1

---

## 横向对比摘要

| 核心维度 | DeepSeek-Reasonix |
|---------|:-----------------:|
| 运行时 | Go (原生) |
| 哲学 | DeepSeek 原生 + 配置驱动 |
| Provider 数 | 1 原生 + 任意 OpenAI 兼容 |
| 内置工具类型 | 编译时注册 (数量靠配置) |
| MCP | 通过 Plugin (JSON-RPC) |
| 记忆系统 | ⚠️ 文件级记忆 (MEMORY.md) |
| Swarm | ✅ Subagent |
| 启动耗时 | 待测量 (Go, 推测快) |
| 内存(1会话) | 待测量 (Go, 推测中等) |
| 生态成熟度 | ★★★☆☆ |
| 安全性 | ★★★☆☆ |
| 代码质量 | ★★★★☆ |
