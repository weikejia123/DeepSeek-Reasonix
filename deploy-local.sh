#!/usr/bin/env bash
# ============================================================================
# deploy-local.sh — DeepSeek-Reasonix 本地部署
#
# 将 wkj-dev 二开分支的 Go 源码构建并安装到本地环境。
# 部署后 `reasonix-go` 命令指向本地二开分支编译的二进制。
# （与 npm 安装的 `reasonix` 区分，避免冲突）
#
# 用法:
#   ./deploy-local.sh                          # 完整部署（构建 + 安装到 PATH）
#   ./deploy-local.sh /custom/path/reasonix-go  # 安装到自定义路径
#   ./deploy-local.sh build                    # 仅构建（不安装）
#   ./deploy-local.sh verify                   # 仅验证
#   ./deploy-local.sh help                     # 显示帮助
#
# 前置条件:
#   - Go 工具链
#   - 当前在 wkj-dev 分支（或设置 REASONIX_ALLOW_BRANCH=1 跳过检查）
#
# 效果:
#   CGO_ENABLED=0 go build 编译单二进制
#   安装到自动检测的 PATH 目录，命名为 reasonix-go
# ============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 委托给已有的 my-scripts/deploy.sh
exec "$SCRIPT_DIR/my-scripts/deploy.sh" "$@"
