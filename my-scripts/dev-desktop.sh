#!/usr/bin/env bash
# =============================================================================
# dev-desktop.sh
#
# 以开发模式启动 Reasonix Desktop，附带 Chrome DevTools。
# 用于调试前端事件流、状态管理、网络请求等。
#
# 用法:
#   ./dev-desktop.sh                         # 正常启动开发模式
#   ./dev-desktop.sh --skip-install           # 跳过依赖安装
#   ./dev-desktop.sh --help                   # 显示帮助
#
# 注意:
#   - wails dev 不会生成 .app 包，直接在终端中运行
#   - 前端代码修改后自动热重载
#   - DevTools 在应用启动后自动弹出
# =============================================================================

set -euo pipefail

# ---- 路径 ----
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DESKTOP_DIR="$PROJECT_DIR/desktop"
FRONTEND_DIR="$DESKTOP_DIR/frontend"

# ---- 颜色 ----
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_step()  { echo -e "\n${CYAN}━━━ $* ━━━${NC}"; }

# ----- 帮助 -----
if [[ "${1:-}" == "--help" ]]; then
    cat <<EOF
用法: $0 [--skip-install]

选项:
  --skip-install    跳过前端依赖安装，直接启动 dev 模式
  --help            显示此帮助

说明:
  wails dev 启动开发服务器，支持热重载 + Chrome DevTools
  按 Ctrl+C 停止
EOF
    exit 0
fi

SKIP_INSTALL=false
[[ "${1:-}" == "--skip-install" ]] && SKIP_INSTALL=true

# =============================================================================
# 阶段 1 — 前置检查
# =============================================================================
check_prerequisites() {
    log_step "检查前置环境"

    command -v go >/dev/null 2>&1 || { echo "需要 Go"; exit 1; }
    log_info "Go: $(go version | grep -oE 'go\S+')"

    command -v node >/dev/null 2>&1 || { echo "需要 Node.js"; exit 1; }
    log_info "Node.js: $(node --version)"

    if ! command -v pnpm >/dev/null 2>&1; then
        log_warn "未安装 pnpm，正在安装..."
        npm install -g pnpm
    fi
    log_info "pnpm: $(pnpm --version)"

    if ! xcode-select -p >/dev/null 2>&1; then
        echo "需要 Xcode CLT"; exit 1
    fi
    log_info "Xcode CLT: 已安装"

    if ! command -v wails >/dev/null 2>&1; then
        log_info "正在安装 Wails CLI..."
        go install github.com/wailsapp/wails/v2/cmd/wails@latest
    fi
    log_info "Wails: $(wails version 2>/dev/null)"

    log_info "环境检查通过"
}

# =============================================================================
# 阶段 2 — 安装前端依赖
# =============================================================================
install_deps() {
    log_step "安装前端依赖"
    [[ -d "$FRONTEND_DIR" ]] || { echo "前端目录不存在"; exit 1; }

    cd "$FRONTEND_DIR"
    if [[ -d "node_modules" ]]; then
        log_info "node_modules 已存在，跳过安装"
    else
        log_info "执行 pnpm install ..."
        pnpm install
    fi
}

# =============================================================================
# 阶段 3 — wails dev 启动
# =============================================================================
start_dev() {
    log_step "启动开发模式（wails dev）"

    cd "$DESKTOP_DIR"

    echo ""
    echo -e "${CYAN}  DevTools 将在应用启动后自动打开${NC}"
    echo -e "${CYAN}  按 Ctrl+C 停止${NC}"
    echo ""

    wails dev
}

# =============================================================================
# 主流程
# =============================================================================

echo ""
echo -e "${CYAN}  ╔═══════════════════════════════════════╗${NC}"
echo -e "${CYAN}  ║   Reasonix Desktop 开发模式          ║${NC}"
echo -e "${CYAN}  ╚═══════════════════════════════════════╝${NC}"
echo ""

check_prerequisites
$SKIP_INSTALL || install_deps
start_dev
