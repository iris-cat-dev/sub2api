#!/usr/bin/env bash
# 构建前端并部署到 Cloudflare Pages。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
FRONTEND_DIR="${REPO_ROOT}/frontend"
PROJECT_NAME="${CF_PAGES_PROJECT:-sub2api-admin-frontend}"
BRANCH="${CF_PAGES_BRANCH:-main}"

if ! command -v pnpm >/dev/null 2>&1; then
    echo "错误: 未找到 pnpm，请先安装 pnpm 11.9.0。" >&2
    exit 1
fi

cd "${FRONTEND_DIR}"

echo "安装前端依赖..."
pnpm install --frozen-lockfile

echo "构建前端..."
pnpm run build

echo "检查 Cloudflare 登录状态..."
pnpm exec wrangler whoami

echo "部署到 Cloudflare Pages: ${PROJECT_NAME} (${BRANCH})..."
pnpm exec wrangler pages deploy dist \
    --project-name "${PROJECT_NAME}" \
    --branch "${BRANCH}" \
    --commit-dirty=true
