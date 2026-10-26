#!/usr/bin/env bash
#
# FusionMail CI 质量门禁
# ============================================================
# 在任何构建/部署之前阻断回归。这是 2.0 重构的"安全网"：
# 没有它，后续所有结构性改动都无法验证。
#
# 检查项：
#   后端  go vet ./...            静态检查
#         go test ./... -race     竞态检测下的全量测试
#   前端  tsc --noEmit            类型检查（��一的类型门禁）
#         vitest run              单元测试
#         eslint .                代码规范
#
# 用法：
#   ./scripts/ci-quality-gate.sh              # 全部检查
#   ./scripts/ci-quality-gate.sh --backend    # 仅后端
#   ./scripts/ci-quality-gate.sh --frontend   # 仅前端
#
# 环境变量：
#   SKIP_INSTALL=1   跳过前端 npm ci（CI 中由镜像层缓存处理时使用）
# ============================================================
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_BACKEND=true
RUN_FRONTEND=true

RED='\033[0;31m'; GREEN='\033[0;32m'; BLUE='\033[0;34m'; NC='\033[0m'
info()  { echo -e "${BLUE}[gate]${NC} $*"; }
ok()    { echo -e "${GREEN}[pass]${NC} $*"; }
fail()  { echo -e "${RED}[FAIL]${NC} $*" >&2; }

while [ $# -gt 0 ]; do
  case "$1" in
    --backend)  RUN_FRONTEND=false ;;
    --frontend) RUN_BACKEND=false ;;
    -h|--help)  sed -n '2,25p' "$0"; exit 0 ;;
    *) fail "未知选项: $1"; exit 1 ;;
  esac
  shift
done

FAILED=()

run() {
  local label="$1"; shift
  info "$label"
  if "$@"; then
    ok "$label"
  else
    fail "$label"
    FAILED+=("$label")
  fi
}

# ---------------------------------------------------------------- 后端
if [ "$RUN_BACKEND" = true ]; then
  command -v go >/dev/null 2>&1 || { fail "未找到 go"; exit 1; }
  cd "$ROOT_DIR/backend"

  run "后端静态检查: go vet ./..."        go vet ./...

  # 生成 API 文档：分层边界守卫测试（internal/dto/response/boundary_test.go）
  # 会解析 docs/swagger.json 检查 GORM 模型是否泄露进 API 契约。
  # 不生成的话该测试会跳过，等于没有门禁。
  if command -v swag >/dev/null 2>&1; then
    run "生成 API 文档: swag init" swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
  else
    info "未找到 swag，跳过 API 文档生成（边界��卫测试将被跳过）"
    info "安装：go install github.com/swaggo/swag/cmd/swag@v1.16.6"
  fi

  run "后端测试: go test ./... -race"     go test ./... -race
fi

# ---------------------------------------------------------------- 前端
if [ "$RUN_FRONTEND" = true ]; then
  command -v npm >/dev/null 2>&1 || { fail "未找到 npm"; exit 1; }
  cd "$ROOT_DIR/frontend"

  if [ "${SKIP_INSTALL:-0}" != "1" ] && [ ! -d node_modules ]; then
    run "前端依赖安装: npm ci" npm ci --no-audit --no-fund
  fi

  run "前端类型检查: tsc --noEmit"  npx tsc --noEmit
  run "前端单元测试: vitest run"    npx vitest run
  run "前端代码规范: eslint"        npx eslint .
fi

# ---------------------------------------------------------------- 汇总
echo
if [ ${#FAILED[@]} -gt 0 ]; then
  fail "质量门禁未通过，共 ${#FAILED[@]} 项失败："
  for f in "${FAILED[@]}"; do echo "  - $f"; done
  exit 1
fi
ok "质量门禁全部通过"
