#!/usr/bin/env bash
# 本机启动 Sub2API（不用 Docker）：Homebrew PostgreSQL/Redis + 已编译的 backend/bin/server
set -euo pipefail

ROOT="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
BACKEND_DIR="${ROOT}/backend"
BIN="${BACKEND_DIR}/bin/server"
DATA_DIR="${BACKEND_DIR}/data"
ENV_FILE="${DATA_DIR}/.env"
PORT="${SERVER_PORT:-8080}"

# 与当前本机部署一致的默认值；可被 ${DATA_DIR}/.env 覆盖。
DEFAULT_JWT_SECRET="local-dev-jwt-secret-change-me-32b"
DEFAULT_TOTP_KEY="057fb7f7a8be34d482e4475acbf4941345b74f1ea72ef4d95f29c72c9cec9067"
DEFAULT_DB_HOST="127.0.0.1"
DEFAULT_DB_PORT="5432"
DEFAULT_DB_USER="sub2api"
DEFAULT_DB_PASSWORD="sub2api_local_dev"
DEFAULT_DB_NAME="sub2api"
DEFAULT_REDIS_HOST="127.0.0.1"
DEFAULT_REDIS_PORT="6379"
DEFAULT_ADMIN_EMAIL="admin@sub2api.local"
DEFAULT_ADMIN_PASSWORD="Admin123456!"

usage() {
  cat <<EOF
用法: $0 [--restart | --stop | -h]

  （无参数）  启动 PostgreSQL、Redis 和 Sub2API
  --restart   先停掉已有进程再启动
  --stop      只停止 Sub2API（不关数据库）
  -h          显示帮助

访问: http://127.0.0.1:${PORT}
账号: ${DEFAULT_ADMIN_EMAIL} / ${DEFAULT_ADMIN_PASSWORD}
EOF
}

log() { printf '%s\n' "$*"; }
err() { printf 'error: %s\n' "$*" >&2; }
die() { err "$*"; exit 1; }

have() { command -v "$1" >/dev/null 2>&1; }

prepend_path() {
  if [ -d "$1" ]; then
    PATH="$1:${PATH}"
  fi
}

setup_path() {
  if have brew; then
    local prefix
    prefix="$(brew --prefix 2>/dev/null || true)"
    if [ -n "${prefix}" ]; then
      prepend_path "${prefix}/bin"
      prepend_path "${prefix}/opt/postgresql@18/bin"
      prepend_path "${prefix}/opt/postgresql/bin"
      prepend_path "${prefix}/opt/redis/bin"
    fi
  fi
  prepend_path /opt/homebrew/opt/postgresql@18/bin
  prepend_path /opt/homebrew/opt/redis/bin
  prepend_path /opt/homebrew/bin
  prepend_path /usr/local/opt/postgresql@18/bin
  prepend_path /usr/local/opt/redis/bin
  export PATH
}

ensure_env_file() {
  mkdir -p "${DATA_DIR}"
  if [ -f "${ENV_FILE}" ]; then
    return 0
  fi

  local totp_key="${DEFAULT_TOTP_KEY}"
  if [ ! -f "${DATA_DIR}/config.yaml" ] && have openssl; then
    totp_key="$(openssl rand -hex 32)"
  fi

  cat > "${ENV_FILE}" <<EOF
# 本机启动环境（位于 gitignore 的 backend/data/）
JWT_SECRET=${DEFAULT_JWT_SECRET}
TOTP_ENCRYPTION_KEY=${totp_key}
DATABASE_HOST=${DEFAULT_DB_HOST}
DATABASE_PORT=${DEFAULT_DB_PORT}
DATABASE_USER=${DEFAULT_DB_USER}
DATABASE_PASSWORD=${DEFAULT_DB_PASSWORD}
DATABASE_DBNAME=${DEFAULT_DB_NAME}
REDIS_HOST=${DEFAULT_REDIS_HOST}
REDIS_PORT=${DEFAULT_REDIS_PORT}
ADMIN_EMAIL=${DEFAULT_ADMIN_EMAIL}
ADMIN_PASSWORD=${DEFAULT_ADMIN_PASSWORD}
SERVER_HOST=0.0.0.0
SERVER_PORT=${PORT}
EOF
  chmod 600 "${ENV_FILE}"
  log "已写入 ${ENV_FILE}"
}

load_env() {
  ensure_env_file
  set -a
  # shellcheck disable=SC1090
  . "${ENV_FILE}"
  set +a

  export DATA_DIR
  export JWT_SECRET="${JWT_SECRET:-${DEFAULT_JWT_SECRET}}"
  export TOTP_ENCRYPTION_KEY="${TOTP_ENCRYPTION_KEY:-${DEFAULT_TOTP_KEY}}"
  export DATABASE_HOST="${DATABASE_HOST:-${DEFAULT_DB_HOST}}"
  export DATABASE_PORT="${DATABASE_PORT:-${DEFAULT_DB_PORT}}"
  export DATABASE_USER="${DATABASE_USER:-${DEFAULT_DB_USER}}"
  export DATABASE_PASSWORD="${DATABASE_PASSWORD:-${DEFAULT_DB_PASSWORD}}"
  export DATABASE_DBNAME="${DATABASE_DBNAME:-${DEFAULT_DB_NAME}}"
  export REDIS_HOST="${REDIS_HOST:-${DEFAULT_REDIS_HOST}}"
  export REDIS_PORT="${REDIS_PORT:-${DEFAULT_REDIS_PORT}}"
  export ADMIN_EMAIL="${ADMIN_EMAIL:-${DEFAULT_ADMIN_EMAIL}}"
  export ADMIN_PASSWORD="${ADMIN_PASSWORD:-${DEFAULT_ADMIN_PASSWORD}}"
  export SERVER_HOST="${SERVER_HOST:-0.0.0.0}"
  export SERVER_PORT="${SERVER_PORT:-${PORT}}"
  PORT="${SERVER_PORT}"
}

start_brew_service() {
  have brew || return 0
  local name="$1"
  if brew services list 2>/dev/null | awk -v n="${name}" 'NR>0 && $1==n { found=1; if ($2=="started") running=1 } END { if (!found) exit 2; exit running ? 0 : 1 }'; then
    return 0
  fi
  local st=$?
  if [ "${st}" -eq 2 ]; then
    return 0
  fi
  log "启动 Homebrew 服务: ${name}"
  brew services start "${name}"
}

wait_for() {
  local desc="$1"
  shift
  local i=0
  while :; do
    if "$@" >/dev/null 2>&1; then
      return 0
    fi
    i=$((i + 1))
    if [ "${i}" -ge 30 ]; then
      die "${desc} 未就绪"
    fi
    sleep 1
  done
}

redis_ready() {
  [ "$(redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" ping 2>/dev/null || true)" = "PONG" ]
}

ensure_database() {
  have psql || return 0
  if ! psql -d postgres -Atqc "SELECT 1" >/dev/null 2>&1; then
    err "无法用当前用户连接本地 PostgreSQL，跳过自动建库"
    return 0
  fi

  psql -d postgres -v ON_ERROR_STOP=1 >/dev/null <<SQL
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '${DATABASE_USER}') THEN
    CREATE ROLE ${DATABASE_USER} LOGIN PASSWORD '${DATABASE_PASSWORD}' CREATEDB;
  ELSE
    ALTER ROLE ${DATABASE_USER} WITH LOGIN PASSWORD '${DATABASE_PASSWORD}' CREATEDB;
  END IF;
END
\$\$;
SQL

  if ! psql -d postgres -Atqc "SELECT 1 FROM pg_database WHERE datname = '${DATABASE_DBNAME}'" | grep -qx 1; then
    psql -d postgres -v ON_ERROR_STOP=1 -c "CREATE DATABASE ${DATABASE_DBNAME} OWNER ${DATABASE_USER};" >/dev/null
  fi
  psql -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE ${DATABASE_DBNAME} TO ${DATABASE_USER};" >/dev/null
  psql -d "${DATABASE_DBNAME}" -c "GRANT ALL ON SCHEMA public TO ${DATABASE_USER};" >/dev/null
}

server_pids() {
  pgrep -f "${BIN}" 2>/dev/null || true
}

health_ok() {
  have curl || return 1
  curl -fsS --max-time 2 "http://127.0.0.1:${PORT}/health" 2>/dev/null | grep -q '"status":"ok"'
}

stop_server() {
  local pids
  pids="$(server_pids)"
  if [ -z "${pids}" ]; then
    log "Sub2API 未在运行"
    return 0
  fi
  log "停止 Sub2API: ${pids}"
  # shellcheck disable=SC2086
  kill ${pids} 2>/dev/null || true
  local i=0
  while [ -n "$(server_pids)" ]; do
    i=$((i + 1))
    if [ "${i}" -ge 15 ]; then
      # shellcheck disable=SC2086
      kill -9 $(server_pids) 2>/dev/null || true
      break
    fi
    sleep 1
  done
}

start_deps() {
  start_brew_service postgresql@18
  start_brew_service redis

  have pg_isready || die "找不到 pg_isready，请确认已安装 postgresql@18"
  have redis-cli || die "找不到 redis-cli，请确认已安装 redis"

  wait_for "PostgreSQL" pg_isready -h "${DATABASE_HOST}" -p "${DATABASE_PORT}"
  wait_for "Redis" redis_ready
  ensure_database
}

start_server() {
  [ -x "${BIN}" ] || die "找不到可执行文件: ${BIN}
请先编译：
  cd frontend && npx --yes pnpm@9 install --frozen-lockfile && npx --yes pnpm@9 run build
  cd backend && CGO_ENABLED=0 go build -tags embed -ldflags='-s -w' -o bin/server ./cmd/server"

  local pids
  pids="$(server_pids)"
  if [ -n "${pids}" ] || health_ok; then
    if health_ok; then
      log "Sub2API 已在运行: http://127.0.0.1:${PORT}"
      return 0
    fi
    die "已有进程 ${pids}，但健康检查失败。可执行: $0 --restart"
  fi

  if have lsof && lsof -nP -iTCP:"${PORT}" -sTCP:LISTEN >/dev/null 2>&1; then
    die "端口 ${PORT} 已被占用"
  fi

  if [ ! -f "${DATA_DIR}/config.yaml" ]; then
    export AUTO_SETUP=true
    log "未检测到 config.yaml，将执行 AUTO_SETUP"
  fi

  log "启动 Sub2API: http://127.0.0.1:${PORT}"
  cd "${BACKEND_DIR}"
  exec "${BIN}"
}

ACTION="start"
case "${1:-}" in
  "" ) ACTION="start" ;;
  --restart ) ACTION="restart" ;;
  --stop ) ACTION="stop" ;;
  -h|--help ) usage; exit 0 ;;
  * ) usage >&2; exit 2 ;;
esac

setup_path
load_env

case "${ACTION}" in
  stop )
    stop_server
    ;;
  restart )
    stop_server
    start_deps
    start_server
    ;;
  start )
    start_deps
    start_server
    ;;
esac
