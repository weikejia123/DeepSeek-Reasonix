# 分支速查

> 团队内部使用的分支清单与速查表。完整说明见 [00-项目元信息/分支管理说明.md](00-项目元信息/分支管理说明.md)。

## 当前活跃分支

| 分支 | 基线 | 用途 | 推远程 | 备注 |
|------|------|------|:------:|------|
| `release/wkj-cli-stable` | `wkj-cli` | **最稳定的本地发布基线** | 仅局域网 | 只从 `wkj-cli` 接收已验证的稳定快照 |
| `wkj-cli` | `upstream/main-v2` | 本地集成二开基线 | 仅局域网 | 不上 GitHub，汇集已验证的本地功能 |
| `feature/sendtab` | `wkj-cli` | sendtab 跨 Tab 消息功能 | 仅局域网 | 个人使用习惯，不通用，有跨 Tab 上下文污染风险 |
| `release-1.7.0-ALOOP-V1` | `upstream/main-v2` | A-Loop 功能发布分支 | 否 | 临时发布验证分支 |

## 分支命名约定

| 前缀 | 含义 | 示例 |
|------|------|------|
| `release/` | 本地稳定发布分支 | `release/wkj-cli-stable` |
| `feature/` | 本地二开功能分支 | `feature/sendtab` |
| `fix/` | 本地修复分支 | `fix/desktop-crash` |
| `sync/` | 上游同步临时分支 | `sync/20260615` |
| `release-` | 上游版本的发布验证分支 | `release-1.7.0-ALOOP-V1` |
| `tmp-dev-YYYYMMDD-` | 个人临时探索/调试分支 | `tmp-dev-20260615-sendtab` |

## 常用命令

```bash
# 查看当前所有本地分支
git branch -vv

# 从 wkj-cli 切新功能分支
git checkout wkj-cli
git checkout -b feature/xxx

# 功能验证后合并回 wkj-cli
git checkout wkj-cli
git merge --no-ff feature/xxx

# 同步上游到 wkj-cli
git fetch upstream
git merge upstream/main-v2

# wkj-cli 验证稳定后，同步到 release 分支
git checkout release/wkj-cli-stable
git merge wkj-cli --no-ff -m "release: sync stable from wkj-cli"
```

## 注意事项

- `release/wkj-cli-stable` 是**最稳定的本地发布基线**，不直接合并上游
- `wkj-cli`、`feature/*`、`release/*` 分支**不会推送到 GitHub**，未来仅通过局域网 git 仓库共享
- 所有功能分支从 `wkj-cli` 切出，不要直接从 `tmp-dev-YYYYMMDD-xxx` 这类临时工作分支继续开发
- 功能验证完成、合并到 `wkj-cli` 后，临时工作分支应及时删除
- `wkj-cli` 验证稳定后，再同步到 `release/wkj-cli-stable`

---

**更新时间**：2026-06-15
