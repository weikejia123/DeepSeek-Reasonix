# OMP DeepSeek 缓存优化方案

## 前提：先搞清楚收益在哪

### 已确认的事实

1. **OMP 的缓存基础设施完整** — `appendOnlyContext` 已为 DeepSeek 自动启用，system prompt + tools 冻结、消息只追加，前缀字节稳定。
2. **工具 schema 中的 `i`（intent）字段是静态的** — schema 定义恒为 `{ type: "string" }`，不随轮次变化 → 不会导致缓存 miss。
3. **工具数量不影响缓存命中率** — 命中率只取决于前缀字节稳定性，工具多少不影响。当前 OMP 在 DeepSeek 上的命中率应该已经很高（~95%+）。

### 减少工具的真正收益

| 收益 | 是否节约 token | 是否提升质量 | 重要性 |
|------|---------------|------------|--------|
| 冷启动成本降低 | ✅ 少量 | ❌ | ⭐ |
| v4-flash 模型质量提升 | ❌ | ✅ | ⭐⭐⭐ |
| 减少模型选择工具的消耗 | ❌ | ✅ | ⭐⭐ |

**核心收益不是 token，是质量。** DeepSeek v4-flash 是入门推理模型，25+ 工具的 schema 占用大量 context 空间（约 3000-5000 token），模型选错工具的几率上升。减少到 15 个核心工具后，模型更聚焦。

## 最小实现路径

> 面向 DeepSeek v4-flash/v4-pro 减少工具注册数量，提升模型响应质量，附带减少冷启动 token 消耗。

### 文件 1：`packages/coding-agent/src/tools/index.ts`

新增核心工具集常量：

```typescript
/** DeepSeek Token Economy 模式下保留的核心内置工具。 */
export const TOKEN_ECONOMY_CORE_NAMES: readonly string[] = [
	"read",    // 文件读取
	"bash",    // 命令执行
	"edit",    // 文件编辑
	"write",   // 文件写入
	"ast_grep",// 代码搜索
	"ast_edit",// 代码编辑
	"find",    // 文件查找
	"search",  // 文本搜索
	"ask",     // 用户交互
	"eval",    // 代码执行
	"todo",    // 任务管理
	"task",    // 子代理
	"job",     // 后台任务
	"irc",     // 代理通信
	"web_search", // 网络搜索
	"inspect_image", // 图片查看
];
```

`createTools` 增加 `tokenEconomy` 参数：

```typescript
export async function createTools(
	session: ToolSession,
	toolNames?: string[],
	options?: { tokenEconomy?: boolean },
): Promise<Tool[]> {
```

在工具过滤逻辑的 `baseEntries` 构建处增加：

```typescript
// Token Economy 模式：只保留核心工具，减少 DeepSeek 模型决策负担
if (options?.tokenEconomy && !requestedTools) {
	baseEntries = baseEntries.filter(([name]) =>
		TOKEN_ECONOMY_CORE_NAMES.includes(name)
	);
}
```

注意：只有当 `toolNames` 未指定（全量工具模式）时才生效。用户显式指定 `toolNames` 时尊重用户配置。

### 文件 2：`packages/coding-agent/src/sdk.ts`

在调用 `createTools` 处传递 `tokenEconomy`：

```typescript
import { isDeepseekModelIdOrName } from "@oh-my-pi/pi-catalog/identity";

const tokenEconomy = isDeepseekModelIdOrName(model.id);
const builtinTools = await createTools(toolSession, options.toolNames, { tokenEconomy });
```

同时联动启用 `pruneToolDescriptions` 进一步压缩 schema：

```typescript
pruneToolDescriptions: tokenEconomy ? true : inlineToolDescriptors,
```

### 改动量

| 文件 | 改动 | 行数 |
|------|------|------|
| `tools/index.ts` | 新增常量 + `createTools` 参数 + 过滤逻辑 | ~25 |
| `sdk.ts` | 模型判断 + 参数传递 + `pruneToolDescriptions` 联动 | ~8 |
| 合计 | | **~33 行** |

### 保留的工具

排除以下非核心工具，它们依赖 `connect_tool_source` 按需激活或无直接替代：

- `ssh` — 远程操作，非核心编码场景
- `github` — 版本控制，非核心
- `lsp` — 自动附加（不占用工具 schema slot）
- `browser` — 浏览器操作，非核心
- `debug` — 调试器，非核心
- `checkpoint` / `rewind` — 检查点，非常用
- `memory_edit` / `retain` / `recall` / `reflect` / `learn` / `manage_skill` — 记忆系统，模块附加值
- `search_tool_bm25` — MCP 发现，默认关闭

## 验证方法

确认工具 schema 减少且前缀稳定：

```bash
# 启动 omp 并用 deepseek-v4-flash
omp --model deepseek-v4-flash

# 打开 cache_hit 状态行
# 观察：cache_hit: xx.xx% 在优化前后应无显著变化
# 观察：prompt_tokens 在冷启动时缩小约 2000-4000
```

直接验证：

```bash
# 对比两次相同请求的 usage
# 用 omp stats 或日志查看 cache_read_input_tokens
```

## 不做的扩展

- ❌ 缓存命中率监控面板 — 非最小实现
- ❌ CI 缓存影响检查 — 非最小实现
- ❌ `connect_tool_source` 工具 — 用户可通过配置白名单自行管理
- ❌ `PrefixShape` 诊断 — OMP 通过 `intentTracing`+`appendOnlyContext` 已保证前缀稳定
