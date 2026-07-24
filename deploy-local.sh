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
#   ./deploy-local.sh verify                   # 仅验证（检测运行的版本是否为本地的）
#   ./deploy-local.sh help                     # 显示帮助
#
# 验证机制:
#   构建时通过 -X ldflags 嵌入 git SHA 到二进制 version 字段
#   安装后通过 stat 比对 mtime/size 确认是刚刚构建的版本
#   运行 reasonix-go --version 查看嵌入的版本信息
#
# 前置条件:
#   - Go 工具链
#   - 当前在 wkj-dev 分支（或设置 REASONIX_ALLOW_BRANCH=1 跳过检查）
#
# 效果:
#   CGO_ENABLED=0 go build -ldflags "-X main.version=<git-SHA>"
#   安装到自动检测的 PATH 目录，命名为 reasonix-go
# ============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[✓]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[!]${NC} $1"; }
log_error() { echo -e "${RED}[✗]${NC} $1"; }
log_step()  { echo -e "${CYAN}━━━ $1 ━━━${NC}"; }

show_help() {
  sed -n '2,/^set -euo/p' "$0" | grep -E '^#' | sed 's/^# \?//'
  exit 0
}

# ─── 构建标记 ───
DEPLOY_MARKER=".deploy-marker"

write_marker() {
  local sha branch time
  sha="$(git rev-parse HEAD 2>/dev/null || echo 'unknown')"
  branch="$(git branch --show-current 2>/dev/null || echo 'unknown')"
  time="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

  local bin_path="bin/reasonix-go"
  local bin_stat=""
  if [ -f "$bin_path" ]; then
    if [ "$(uname)" = "Darwin" ]; then
      bin_stat="$(stat -f "%m|%z|%N" "$bin_path" 2>/dev/null || echo '')"
    else
      bin_stat="$(stat -c "%Y|%s|%n" "$bin_path" 2>/dev/null || echo '')"
    fi
  fi

  cat > "$DEPLOY_MARKER" <<EOF
SHA=$sha
BRANCH=$branch
BUILD_TIME=$time
BIN_PATH=$bin_path
BIN_STAT=$bin_stat
EOF
  log_info "构建标记已写入: $DEPLOY_MARKER"
  cat "$DEPLOY_MARKER" | sed 's/^/      /'
}

read_marker() {
  local key="$1"
  grep "^${key}=" "$DEPLOY_MARKER" 2>/dev/null | cut -d= -f2 || echo "unknown"
}

# ─── 前置检查 ───
check_prereqs() {
  log_step "检查前置条件"

  if ! command -v go &>/dev/null; then
    log_error "未找到 Go。请安装 Go 工具链"
    exit 1
  fi
  log_info "Go $(go version | grep -oP 'go\d+\.\d+\.\d+' || go version | awk '{print $3}')"

  local branch
  branch="$(git branch --show-current 2>/dev/null || echo '')"
  if [ "$branch" != "wkj-dev" ]; then
    log_warn "当前分支: $branch（期望: wkj-dev）"
    if [ "${REASONIX_ALLOW_BRANCH:-}" != "1" ]; then
      echo "   使用 git checkout wkj-dev 切换分支，或设置 REASONIX_ALLOW_BRANCH=1 跳过检查"
      exit 1
    fi
    log_warn "REASONIX_ALLOW_BRANCH=1 已设置，跳过分支检查"
  else
    log_info "当前分支: wkj-dev ✅"
  fi
}

# ─── 构建 ───
do_build() {
  log_step "构建 DeepSeek-Reasonix"
  local start
  start="$(date +%s)"

  local version
  version="$(git describe --tags --always 2>/dev/null || echo 'dev')-wkj-$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
  local ldflags="-s -w -X main.version=${version}"

  echo "  Version: $version"
  mkdir -p bin
  CGO_ENABLED=0 go build -ldflags "$ldflags" -o bin/reasonix-go ./cmd/reasonix 2>&1 | sed 's/^/      /'

  local end
  end="$(date +%s)"
  log_info "构建完成（$((end - start))s）"

  if [ -f bin/reasonix-go ]; then
    log_info "产物验证: bin/reasonix-go ✅"
    ls -lh bin/reasonix-go | awk '{print "      size:", $5}'
    # 验证 version 已嵌入
    local embedded
    embedded="$(bin/reasonix-go version 2>/dev/null || bin/reasonix-go --version 2>/dev/null || echo '')"
    if [ -n "$embedded" ]; then
      log_info "嵌入版本: $embedded"
    fi
  else
    log_error "构建失败: bin/reasonix-go 未生成"
    exit 1
  fi

  write_marker
}

# ─── 安装到 PATH ───
do_install() {
  local target="$1"
  local src="bin/reasonix-go"

  if [ ! -f "$src" ]; then
    log_error "二进制未找到: $src，请先构建"
    exit 1
  fi

  log_step "安装 reasonix-go → $target"

  local target_dir
  target_dir="$(dirname "$target")"
  mkdir -p "$target_dir"

  if [ ! -w "$target_dir" ]; then
    sudo cp -f "$src" "$target"
    sudo chmod +x "$target"
    log_info "已安装（sudo）: $target"
  else
    cp -f "$src" "$target"
    chmod +x "$target"
    log_info "已安装: $target"
  fi

  log_info "大小: $(ls -lh "$target" | awk '{print $5}')"
}

# ─── 确定安装目标 ───
resolve_target() {
  local custom="${1:-}"

  if [ -n "$custom" ] && [ "$custom" != "build" ] && [ "$custom" != "verify" ] && [ "$custom" != "help" ]; then
    echo "$custom"
    return
  fi

  if which reasonix-go &>/dev/null; then
    which reasonix-go
    return
  fi

  if [ -d "/usr/local/bin" ] && [ -w "/usr/local/bin" ]; then
    echo "/usr/local/bin/reasonix-go"
  else
    echo "$HOME/.local/bin/reasonix-go"
  fi
}

# ─── 验证部署 ───
verify_deployment() {
  log_step "验证部署"

  local errors=0

  if ! command -v reasonix-go &>/dev/null; then
    log_error "reasonix-go 命令不存在！"
    return 1
  fi
  log_info "reasonix-go 命令存在 ✅"

  local cmd_path
  cmd_path="$(which reasonix-go 2>/dev/null || true)"
  log_info "命令路径: $cmd_path"

  # 二进制 stat 对比
  if [ -f "$DEPLOY_MARKER" ]; then
    local marker_bin_path
    marker_bin_path="$(read_marker BIN_PATH)"

    if [ -n "$marker_bin_path" ] && [ -f "$marker_bin_path" ]; then
      local marker_stat
      marker_stat="$(read_marker BIN_STAT)"

      local actual_stat=""
      if [ -f "$cmd_path" ]; then
        if [ "$(uname)" = "Darwin" ]; then
          actual_stat="$(stat -f "%m|%z|%N" "$cmd_path" 2>/dev/null || echo '')"
        else
          actual_stat="$(stat -c "%Y|%s|%n" "$cmd_path" 2>/dev/null || echo '')"
        fi
      fi

      local marker_mtime="${marker_stat%%|*}"
      local actual_mtime="${actual_stat%%|*}"

      if [ "$marker_stat" = "$actual_stat" ]; then
        log_info "二进制 stat 匹配: 安装的正是刚刚构建的版本 ✅"
      elif [ "$marker_mtime" = "$actual_mtime" ]; then
        log_info "二进制 mtime 匹配 ✅"
      else
        log_warn "二进制 stat 不匹配 — 可能是旧版本"
        errors=$((errors + 1))
      fi
    fi
  fi

  # 运行版本（Go 二进制内嵌了 git SHA）
  log_info "执行 reasonix-go version..."
  local version_output
  version_output="$(reasonix-go version 2>&1 || reasonix-go --version 2>&1 || echo 'N/A')"
  log_info "版本输出: $version_output"

  # SHA 验证
  if [ -f "$DEPLOY_MARKER" ]; then
    local marker_sha current_sha
    marker_sha="$(read_marker SHA)"
    current_sha="$(git rev-parse HEAD 2>/dev/null || echo 'unknown')"

    if [ "$marker_sha" = "$current_sha" ]; then
      log_info "SHA 验证: $marker_sha ✅（与当前 HEAD 一致）"
    else
      log_warn "SHA 验证: 标记 $marker_sha ≠ 当前 $current_sha"
    fi
  fi

  echo ""
  if [ "$errors" -eq 0 ]; then
    echo -e "${GREEN}══════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}  ✅ 部署验证通过！运行的正是 wkj-dev 本地版本${NC}"
    echo -e "${GREEN}══════════════════════════════════════════════════${NC}"
  else
    echo -e "${YELLOW}⚠  部署完成但有 $errors 个警告${NC}"
  fi
  echo "  命令:  $(which reasonix-go)"
  echo "  版本:  $version_output"
  echo "  分支:  $(git branch --show-current)"
  echo "  SHA:   $(git rev-parse HEAD | head -c 12)"
}

# ─── 主流程 ───
main() {
  local cmd="${1:-full}"

  case "$cmd" in
    full)
      check_prereqs
      do_build
      local target
      target="$(resolve_target "${2:-}")"
      do_install "$target"
      verify_deployment
      ;;
    build)
      check_prereqs
      do_build
      ;;
    verify)
      verify_deployment
      ;;
    help|--help|-h)
      show_help
      ;;
    *)
      check_prereqs
      do_build
      do_install "$cmd"
      verify_deployment
      ;;
  esac
}

main "$@"
