#!/usr/bin/env bash
set -Eeuo pipefail

REPO="${NPANEL_REPO:-npanel-dev/NPanel-backend}"
SOURCE_REF="${NPANEL_SOURCE_REF:-${NPANEL_BRANCH:-main}}"
MODE="${NPANEL_INSTALL_MODE:-}"
SERVICE_NAME="${NPANEL_SERVICE_NAME:-npanel}"
INSTALL_DIR="${NPANEL_INSTALL_DIR:-/opt/npanel-backend}"
CONFIG_DIR="${NPANEL_CONFIG_DIR:-/etc/npanel}"
CONFIG_FILE="${CONFIG_DIR}/config.yaml"
ENV_FILE="${CONFIG_DIR}/.env"
COMPOSE_FILE="${CONFIG_DIR}/docker-compose.yml"
MANAGE_BIN="${NPANEL_MANAGE_BIN:-/usr/local/bin/npanel-backend}"
HTTP_PORT="${NPANEL_HTTP_PORT:-8081}"
GRPC_PORT="${NPANEL_GRPC_PORT:-9012}"
SITE_HOST="${NPANEL_SITE_HOST:-localhost:${HTTP_PORT}}"
ADMIN_EMAIL="${NPANEL_ADMIN_EMAIL:-admin@npanel.dev}"
FORCE_CONFIG="${NPANEL_FORCE_CONFIG:-0}"
BIN_NAME="npanel"

MYSQL_HOST="${NPANEL_MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${NPANEL_MYSQL_PORT:-3306}"
MYSQL_DATABASE="${NPANEL_MYSQL_DATABASE:-npanel}"
MYSQL_USER="${NPANEL_MYSQL_USER:-npanel}"
MYSQL_PASSWORD="${NPANEL_MYSQL_PASSWORD:-}"
REDIS_HOST="${NPANEL_REDIS_HOST:-127.0.0.1}"
REDIS_PORT="${NPANEL_REDIS_PORT:-6379}"
REDIS_PASSWORD="${NPANEL_REDIS_PASSWORD:-}"
JWT_SECRET="${NPANEL_JWT_SECRET:-}"
ADMIN_PASSWORD="${NPANEL_ADMIN_PASSWORD:-}"

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
plain='\033[0m'

info() {
    echo -e "${green}==>${plain} $*"
}

warn() {
    echo -e "${yellow}WARN:${plain} $*"
}

error() {
    echo -e "${red}ERROR:${plain} $*" >&2
    exit 1
}

has_tty() {
    [[ -r /dev/tty && -w /dev/tty ]]
}

prompt_read() {
    local prompt="$1"
    local default="${2:-}"
    local secret="${3:-0}"
    local value

    if ! has_tty; then
        error "当前环境没有可交互终端；请使用 --mode 参数和 NPANEL_* 环境变量执行非交互安装"
    fi

    if [[ "${secret}" == "1" && -n "${default}" ]]; then
        prompt="${prompt} [已设置，回车保留]"
    elif [[ -n "${default}" ]]; then
        prompt="${prompt} [${default}]"
    fi
    prompt="${prompt}: "

    if [[ "${secret}" == "1" ]]; then
        printf "%s" "${prompt}" >/dev/tty
        IFS= read -r -s value </dev/tty || true
        printf "\n" >/dev/tty
    else
        printf "%s" "${prompt}" >/dev/tty
        IFS= read -r value </dev/tty || true
    fi

    if [[ -z "${value}" ]]; then
        value="${default}"
    fi
    printf "%s" "${value}"
}

prompt_yes_no() {
    local prompt="$1"
    local default="${2:-n}"
    local answer

    answer="$(prompt_read "${prompt}" "${default}")"
    case "${answer}" in
        y | Y | yes | YES) return 0 ;;
        *) return 1 ;;
    esac
}

yaml_escape() {
    printf "%s" "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

usage() {
    cat <<EOF
NPanel Backend one-click installer

Usage:
  bash install.sh [version]
  bash install.sh --version v1.0.10
  bash install.sh --mode docker
  bash install.sh --mode binary

Interactive menu:
  1) Docker all-in-one: installs Docker when needed, starts MySQL 8, Redis, and NPanel with Docker Compose.
  2) Local binary: installs NPanel on this server and asks for external/local MySQL and Redis connection info.

Options:
  -v, --version VERSION       NPanel build version, default: latest GitHub release
      --mode MODE             docker or binary; omit it to show the interactive menu
      --source-ref REF        Git branch/tag for Docker source checkout, default: ${SOURCE_REF}
      --http-port PORT        Host HTTP port, default: ${HTTP_PORT}
      --grpc-port PORT        Host gRPC port, default: ${GRPC_PORT}
      --site-host HOST        app.site.host value, default: ${SITE_HOST}
      --admin-email EMAIL     Default admin email, default: ${ADMIN_EMAIL}
      --force-config          Regenerate ${CONFIG_FILE} and ${ENV_FILE}
  -h, --help                  Show this help

Environment:
  NPANEL_REPO                 GitHub repo, default: ${REPO}
  NPANEL_SOURCE_REF           Git branch/tag for Docker source checkout
  NPANEL_INSTALL_DIR          Source/install directory, default: ${INSTALL_DIR}
  NPANEL_CONFIG_DIR           Runtime config directory, default: ${CONFIG_DIR}
  NPANEL_INSTALL_MODE         docker or binary
  NPANEL_MYSQL_HOST           MySQL host for binary mode, default: ${MYSQL_HOST}
  NPANEL_MYSQL_PORT           MySQL port for binary mode, default: ${MYSQL_PORT}
  NPANEL_MYSQL_DATABASE       MySQL database for binary mode, default: ${MYSQL_DATABASE}
  NPANEL_MYSQL_USER           MySQL user for binary mode, default: ${MYSQL_USER}
  NPANEL_MYSQL_PASSWORD       MySQL password for binary mode
  NPANEL_REDIS_HOST           Redis host for binary mode, default: ${REDIS_HOST}
  NPANEL_REDIS_PORT           Redis port for binary mode, default: ${REDIS_PORT}
  NPANEL_REDIS_PASSWORD       Redis password for binary mode; empty is allowed

Examples:
  curl -fsSL https://raw.githubusercontent.com/${REPO}/dev/scripts/install.sh | sudo bash
  curl -fsSL https://raw.githubusercontent.com/${REPO}/dev/scripts/install.sh | sudo bash -s -- v1.0.10
  curl -fsSL https://raw.githubusercontent.com/${REPO}/dev/scripts/install.sh | sudo bash -s -- --source-ref dev
EOF
}

require_root() {
    if [[ "$(id -u)" -ne 0 ]]; then
        error "请以 root 用户运行此脚本，例如：curl -fsSL ... | sudo bash"
    fi
}

detect_os() {
    if [[ "$(uname -s)" != "Linux" ]]; then
        error "当前脚本仅支持 Linux"
    fi

    if ! command -v systemctl >/dev/null 2>&1; then
        error "未检测到 systemd/systemctl，暂不支持当前系统的服务管理器"
    fi
}

detect_arch() {
    local machine
    machine="$(uname -m)"
    case "${machine}" in
        x86_64 | amd64)
            ARCH="amd64"
            ;;
        aarch64 | arm64)
            ARCH="arm64"
            ;;
        *)
            error "不支持的 CPU 架构：${machine}"
            ;;
    esac
}

detect_pkg_manager() {
    if command -v apt-get >/dev/null 2>&1; then
        PKG_MANAGER="apt-get"
    elif command -v dnf >/dev/null 2>&1; then
        PKG_MANAGER="dnf"
    elif command -v yum >/dev/null 2>&1; then
        PKG_MANAGER="yum"
    elif command -v apk >/dev/null 2>&1; then
        PKG_MANAGER="apk"
    else
        PKG_MANAGER=""
    fi
}

install_packages() {
    local missing=()
    local package

    for package in curl git jq openssl tar; do
        if ! command -v "${package}" >/dev/null 2>&1; then
            missing+=("${package}")
        fi
    done

    if [[ "${#missing[@]}" -eq 0 ]]; then
        return
    fi

    if [[ -z "${PKG_MANAGER}" ]]; then
        error "缺少依赖：${missing[*]}，且无法识别包管理器，请先手动安装"
    fi

    info "安装依赖：${missing[*]}"
    case "${PKG_MANAGER}" in
        apt-get)
            apt-get update
            apt-get install -y "${missing[@]}"
            ;;
        dnf)
            dnf install -y "${missing[@]}"
            ;;
        yum)
            yum install -y "${missing[@]}"
            ;;
        apk)
            apk add --no-cache "${missing[@]}"
            ;;
        *)
            error "不支持的包管理器：${PKG_MANAGER}"
            ;;
    esac
}

parse_args() {
    VERSION="${NPANEL_VERSION:-}"
    while [[ "$#" -gt 0 ]]; do
        case "$1" in
            -h | --help)
                usage
                exit 0
                ;;
            -v | --version)
                [[ "$#" -ge 2 ]] || error "--version 需要版本号"
                VERSION="$2"
                shift 2
                ;;
            --version=*)
                VERSION="${1#*=}"
                shift
                ;;
            --mode)
                [[ "$#" -ge 2 ]] || error "--mode 需要 docker 或 binary"
                MODE="$2"
                shift 2
                ;;
            --mode=*)
                MODE="${1#*=}"
                shift
                ;;
            --source-ref)
                [[ "$#" -ge 2 ]] || error "--source-ref 需要分支或标签"
                SOURCE_REF="$2"
                shift 2
                ;;
            --source-ref=*)
                SOURCE_REF="${1#*=}"
                shift
                ;;
            --http-port)
                [[ "$#" -ge 2 ]] || error "--http-port 需要端口"
                HTTP_PORT="$2"
                shift 2
                ;;
            --http-port=*)
                HTTP_PORT="${1#*=}"
                shift
                ;;
            --grpc-port)
                [[ "$#" -ge 2 ]] || error "--grpc-port 需要端口"
                GRPC_PORT="$2"
                shift 2
                ;;
            --grpc-port=*)
                GRPC_PORT="${1#*=}"
                shift
                ;;
            --site-host)
                [[ "$#" -ge 2 ]] || error "--site-host 需要域名或 host:port"
                SITE_HOST="$2"
                shift 2
                ;;
            --site-host=*)
                SITE_HOST="${1#*=}"
                shift
                ;;
            --admin-email)
                [[ "$#" -ge 2 ]] || error "--admin-email 需要邮箱"
                ADMIN_EMAIL="$2"
                shift 2
                ;;
            --admin-email=*)
                ADMIN_EMAIL="${1#*=}"
                shift
                ;;
            --force-config)
                FORCE_CONFIG=1
                shift
                ;;
            v* | [0-9]*)
                VERSION="$1"
                shift
                ;;
            *)
                error "未知参数：$1"
                ;;
        esac
    done

    if [[ -n "${MODE}" ]]; then
        case "${MODE}" in
            docker | binary) ;;
            *) error "--mode 仅支持 docker 或 binary" ;;
        esac
    fi

    if [[ -n "${VERSION}" && "${VERSION}" != v* ]]; then
        VERSION="v${VERSION}"
    fi

    if [[ "${SITE_HOST}" == "localhost:8081" && "${HTTP_PORT}" != "8081" ]]; then
        SITE_HOST="localhost:${HTTP_PORT}"
    fi
}

prompt_install_mode() {
    local choice

    if [[ -n "${MODE}" ]]; then
        return
    fi

    if ! has_tty; then
        error "当前环境没有可交互终端；请使用 --mode docker 或 --mode binary"
    fi

    cat >/dev/tty <<EOF

请选择 NPanel Backend 部署方式：
  1) Docker 一键部署（自动安装 Docker、MySQL 8、Redis、NPanel）
  2) 本机安装 NPanel（二进制 + systemd，手动输入 MySQL/Redis 信息）

EOF
    choice="$(prompt_read "请输入选项" "1")"
    case "${choice}" in
        1 | docker | Docker | DOCKER)
            MODE="docker"
            ;;
        2 | binary | local | Local | LOCAL)
            MODE="binary"
            ;;
        *)
            error "无效选项：${choice}"
            ;;
    esac
}

fetch_latest_version() {
    if [[ -n "${VERSION}" ]]; then
        return
    fi

    info "获取 ${REPO} 最新版本"
    VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | jq -r '.tag_name // empty')"
    if [[ -z "${VERSION}" || "${VERSION}" == "null" ]]; then
        error "无法获取最新版本号，请检查网络或 GitHub Release 状态"
    fi
}

random_hex() {
    local bytes="${1:-24}"
    openssl rand -hex "${bytes}"
}

install_docker() {
    if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
        info "Docker 与 Docker Compose 已安装"
    else
        info "安装 Docker Engine 与 Compose 插件"
        curl -fsSL https://get.docker.com -o /tmp/get-docker.sh
        sh /tmp/get-docker.sh
        rm -f /tmp/get-docker.sh
    fi

    systemctl enable --now docker >/dev/null 2>&1 || systemctl start docker

    if ! docker info >/dev/null 2>&1; then
        error "Docker 未能正常启动，请检查 systemctl status docker"
    fi

    if ! docker compose version >/dev/null 2>&1; then
        error "未检测到 docker compose 插件，请检查 Docker 安装结果"
    fi
}

checkout_source() {
    local repo_url="https://github.com/${REPO}.git"

    if [[ -d "${INSTALL_DIR}/.git" ]]; then
        info "更新源码目录：${INSTALL_DIR}"
        git -C "${INSTALL_DIR}" fetch --tags origin
        git -C "${INSTALL_DIR}" checkout "${SOURCE_REF}"
        git -C "${INSTALL_DIR}" pull --ff-only origin "${SOURCE_REF}" || warn "当前引用可能是标签或不可快进引用，已跳过 pull"
        return
    fi

    if [[ -e "${INSTALL_DIR}" && -n "$(find "${INSTALL_DIR}" -mindepth 1 -maxdepth 1 2>/dev/null)" ]]; then
        error "${INSTALL_DIR} 已存在且不是 Git 仓库，请先备份或设置 NPANEL_INSTALL_DIR"
    fi

    info "克隆源码 ${REPO}@${SOURCE_REF} 到 ${INSTALL_DIR}"
    mkdir -p "$(dirname "${INSTALL_DIR}")"
    git clone --depth 1 --branch "${SOURCE_REF}" "${repo_url}" "${INSTALL_DIR}"
}

write_runtime_env() {
    mkdir -p "${CONFIG_DIR}"

    if [[ -f "${ENV_FILE}" && "${FORCE_CONFIG}" != "1" ]]; then
        info "保留已有运行环境变量：${ENV_FILE}"
        # shellcheck disable=SC1090
        source "${ENV_FILE}"
        HTTP_PORT="${NPANEL_HTTP_PORT:-${HTTP_PORT}}"
        GRPC_PORT="${NPANEL_GRPC_PORT:-${GRPC_PORT}}"
        ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"
        return
    fi

    MYSQL_ROOT_PASSWORD="$(random_hex 24)"
    MYSQL_PASSWORD="$(random_hex 24)"
    REDIS_PASSWORD="$(random_hex 24)"
    JWT_SECRET="$(random_hex 32)"
    ADMIN_PASSWORD="$(random_hex 8)"

    info "生成运行环境变量：${ENV_FILE}"
    cat >"${ENV_FILE}" <<EOF
NPANEL_VERSION=${VERSION}
NPANEL_HTTP_PORT=${HTTP_PORT}
NPANEL_GRPC_PORT=${GRPC_PORT}
MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD}
MYSQL_DATABASE=npanel
MYSQL_USER=npanel
MYSQL_PASSWORD=${MYSQL_PASSWORD}
REDIS_PASSWORD=${REDIS_PASSWORD}
ADMIN_EMAIL=${ADMIN_EMAIL}
ADMIN_PASSWORD=${ADMIN_PASSWORD}
JWT_SECRET=${JWT_SECRET}
NPANEL_UPLOAD_DIR=/app/uploads
SITE_HOST=${SITE_HOST}
EOF
    chmod 0600 "${ENV_FILE}"
}

write_docker_config() {
    if [[ -f "${CONFIG_FILE}" && "${FORCE_CONFIG}" != "1" ]]; then
        info "保留已有配置：${CONFIG_FILE}"
        return
    fi

    info "写入 NPanel 配置：${CONFIG_FILE}"
    cat >"${CONFIG_FILE}" <<EOF
server:
  http:
    addr: 0.0.0.0:8081
    timeout: 55s
  grpc:
    addr: 0.0.0.0:9012
    timeout: 55s
  auth:
    enable_jwt: true
    jwt_secret: "${JWT_SECRET}"
    no_auth_paths:
      - "/api.public.auth.v1."
      - "/api.public.common.v1."
      - "/api/public/auth/"
      - "/api/public/common/"
      - "/api.server."
  cors:
    enable: true
    allowed_origins:
      - "*"
    allowed_methods:
      - "GET"
      - "POST"
      - "PUT"
      - "DELETE"
      - "OPTIONS"
    allowed_headers:
      - "*"
    exposed_headers:
      - "Content-Length"
      - "Authorization"
    allow_credentials: true
    max_age: 86400

log:
  level: info
  format: json
  disable_console: false
  path: logs
  max_size_mb: 100
  max_backups: 30
  max_age_days: 7
  compress: true

data:
  database:
    driver: mysql
    source: npanel:${MYSQL_PASSWORD}@tcp(mysql:3306)/npanel?parseTime=True&loc=Local
  redis:
    addr: redis:6379
    password: "${REDIS_PASSWORD}"
    read_timeout: 0.2s
    write_timeout: 0.2s
    db: 0
    pool_size: 10
    min_idle_conns: 5

app:
  site:
    host: "${SITE_HOST}"
    site_name: "NPanel"
    site_desc: "Professional Panel Management System"
    site_logo: "/assets/logo.png"
    keywords: "panel,management,npanel"
    custom_html: ""
    custom_data: ""

  verify:
    turnstile_site_key: ""
    enable_login_verify: false
    enable_register_verify: false
    enable_reset_password_verify: false

  mobile:
    enable: false
    enable_whitelist: false
    whitelist: []

  email:
    enable: false
    enable_verify: false
    enable_domain_suffix: false
    domain_suffix_list: ""

  register:
    stop_register: false
    enable_ip_register_limit: false
    ip_register_limit: 5
    ip_register_limit_duration: 3600
    enable_trial: false
    trial_subscribe: 7
    trial_time_unit: "day"
    trial_time: 7

  invite:
    forced_invite: false
    referral_percentage: 10
    only_first_purchase: true

  subscribe:
    single_model: false
    subscribe_path: "/subscribe"
    subscribe_domain: ""
    pan_domain: false
    user_agent_limit: false
    user_agent_list: ""

  admin:
    email: "${ADMIN_EMAIL}"
    password: "${ADMIN_PASSWORD}"
    algo: "default"
EOF
    chmod 0600 "${CONFIG_FILE}"
}

write_compose_file() {
    info "写入 Docker Compose 配置：${COMPOSE_FILE}"
    cat >"${COMPOSE_FILE}" <<EOF
services:
  mysql:
    image: mysql:8.4
    restart: unless-stopped
    environment:
      MYSQL_ROOT_PASSWORD: \${MYSQL_ROOT_PASSWORD}
      MYSQL_DATABASE: \${MYSQL_DATABASE}
      MYSQL_USER: \${MYSQL_USER}
      MYSQL_PASSWORD: \${MYSQL_PASSWORD}
      TZ: UTC
    command:
      - --character-set-server=utf8mb4
      - --collation-server=utf8mb4_unicode_ci
    volumes:
      - npanel_mysql:/var/lib/mysql
    healthcheck:
      test: ['CMD-SHELL', 'mysqladmin ping -h 127.0.0.1 -u\$\${MYSQL_USER} -p"\$\${MYSQL_PASSWORD}"']
      interval: 10s
      timeout: 5s
      retries: 20

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    command: ["redis-server", "--appendonly", "yes", "--requirepass", "\${REDIS_PASSWORD}"]
    volumes:
      - npanel_redis:/data
    healthcheck:
      test: ['CMD-SHELL', 'redis-cli -a "\$\${REDIS_PASSWORD}" ping | grep PONG']
      interval: 10s
      timeout: 5s
      retries: 20

  npanel:
    build:
      context: ${INSTALL_DIR}
      args:
        VERSION: \${NPANEL_VERSION:-${VERSION}}
    image: npanel:latest
    restart: unless-stopped
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy
    ports:
      - "\${NPANEL_HTTP_PORT:-8081}:8081"
      - "\${NPANEL_GRPC_PORT:-9012}:9012"
    volumes:
      - ${CONFIG_FILE}:/data/conf/config.yaml:ro
      - npanel_logs:/app/logs
      - npanel_uploads:/app/uploads
    environment:
      NPANEL_UPLOAD_DIR: \${NPANEL_UPLOAD_DIR:-/app/uploads}

volumes:
  npanel_mysql:
  npanel_redis:
  npanel_logs:
  npanel_uploads:
EOF
}

docker_compose() {
    docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" "$@"
}

write_manage_script() {
    info "写入管理命令：${MANAGE_BIN}"
    cat >"${MANAGE_BIN}" <<EOF
#!/usr/bin/env bash
set -Eeuo pipefail

INSTALL_DIR="${INSTALL_DIR}"
ENV_FILE="${ENV_FILE}"
COMPOSE_FILE="${COMPOSE_FILE}"

compose() {
    docker compose --env-file "\${ENV_FILE}" -f "\${COMPOSE_FILE}" "\$@"
}

case "\${1:-status}" in
    status)
        compose ps
        ;;
    logs)
        compose logs -f npanel
        ;;
    restart)
        compose restart npanel
        ;;
    stop)
        compose down
        ;;
    start)
        compose up -d
        ;;
    upgrade)
        git -C "\${INSTALL_DIR}" pull --ff-only
        compose up -d --build
        ;;
    *)
        echo "Usage: npanel-backend {status|logs|restart|start|stop|upgrade}"
        exit 1
        ;;
esac
EOF
    chmod 0755 "${MANAGE_BIN}"
}

start_docker_stack() {
    info "启动 NPanel Backend Docker Compose 服务"
    docker_compose up -d --build
    docker_compose ps
}

print_docker_summary() {
    local admin_password="${ADMIN_PASSWORD:-请查看 ${ENV_FILE}}"
    cat <<EOF

${green}NPanel Backend ${VERSION} Docker 模式安装完成${plain}

源码目录: ${INSTALL_DIR}
运行配置: ${CONFIG_FILE}
环境变量: ${ENV_FILE}
Compose 文件: ${COMPOSE_FILE}
管理命令: ${MANAGE_BIN}

HTTP API: http://服务器IP:${HTTP_PORT}
gRPC: 服务器IP:${GRPC_PORT}
默认管理员: ${ADMIN_EMAIL}
默认密码: ${admin_password}

常用命令:
  npanel-backend status
  npanel-backend logs
  npanel-backend restart
  npanel-backend upgrade

生产环境请首次登录后立即修改管理员密码。
EOF
}

install_docker_mode() {
    install_docker
    checkout_source
    write_runtime_env
    write_docker_config
    write_compose_file
    write_manage_script
    start_docker_stack
    print_docker_summary
}

collect_binary_config() {
    if ! has_tty; then
        JWT_SECRET="${JWT_SECRET:-$(random_hex 32)}"
        ADMIN_PASSWORD="${ADMIN_PASSWORD:-$(random_hex 8)}"
        return
    fi

    cat >/dev/tty <<EOF

请输入 MySQL 连接信息：
EOF
    MYSQL_HOST="$(prompt_read "MySQL 地址" "${MYSQL_HOST}")"
    MYSQL_PORT="$(prompt_read "MySQL 端口" "${MYSQL_PORT}")"
    MYSQL_DATABASE="$(prompt_read "MySQL 数据库名" "${MYSQL_DATABASE}")"
    MYSQL_USER="$(prompt_read "MySQL 账号" "${MYSQL_USER}")"
    MYSQL_PASSWORD="$(prompt_read "MySQL 密码" "${MYSQL_PASSWORD}" 1)"

    cat >/dev/tty <<EOF

请输入 Redis 连接信息：
EOF
    REDIS_HOST="$(prompt_read "Redis 地址" "${REDIS_HOST}")"
    REDIS_PORT="$(prompt_read "Redis 端口" "${REDIS_PORT}")"
    REDIS_PASSWORD="$(prompt_read "Redis 密码，留空则无密码" "${REDIS_PASSWORD}" 1)"

    ADMIN_EMAIL="$(prompt_read "默认管理员邮箱" "${ADMIN_EMAIL}")"
    ADMIN_PASSWORD="${ADMIN_PASSWORD:-$(random_hex 8)}"
    JWT_SECRET="${JWT_SECRET:-$(random_hex 32)}"
}

write_binary_config() {
    local mysql_host mysql_port mysql_database mysql_user mysql_password
    local redis_host redis_port redis_password jwt_secret admin_email admin_password site_host

    if [[ -f "${CONFIG_FILE}" && "${FORCE_CONFIG}" != "1" ]]; then
        if has_tty && prompt_yes_no "${CONFIG_FILE} 已存在，是否覆盖" "n"; then
            FORCE_CONFIG=1
        else
            info "保留已有配置：${CONFIG_FILE}"
            return
        fi
    fi

    mkdir -p "${CONFIG_DIR}"

    mysql_host="$(yaml_escape "${MYSQL_HOST}")"
    mysql_port="$(yaml_escape "${MYSQL_PORT}")"
    mysql_database="$(yaml_escape "${MYSQL_DATABASE}")"
    mysql_user="$(yaml_escape "${MYSQL_USER}")"
    mysql_password="$(yaml_escape "${MYSQL_PASSWORD}")"
    redis_host="$(yaml_escape "${REDIS_HOST}")"
    redis_port="$(yaml_escape "${REDIS_PORT}")"
    redis_password="$(yaml_escape "${REDIS_PASSWORD}")"
    jwt_secret="$(yaml_escape "${JWT_SECRET}")"
    admin_email="$(yaml_escape "${ADMIN_EMAIL}")"
    admin_password="$(yaml_escape "${ADMIN_PASSWORD}")"
    site_host="$(yaml_escape "${SITE_HOST}")"

    info "写入 NPanel 本机配置：${CONFIG_FILE}"
    cat >"${CONFIG_FILE}" <<EOF
server:
  http:
    addr: 0.0.0.0:8081
    timeout: 55s
  grpc:
    addr: 0.0.0.0:9012
    timeout: 55s
  auth:
    enable_jwt: true
    jwt_secret: "${jwt_secret}"
    no_auth_paths:
      - "/api.public.auth.v1."
      - "/api.public.common.v1."
      - "/api/public/auth/"
      - "/api/public/common/"
      - "/api.server."
  cors:
    enable: true
    allowed_origins:
      - "*"
    allowed_methods:
      - "GET"
      - "POST"
      - "PUT"
      - "DELETE"
      - "OPTIONS"
    allowed_headers:
      - "*"
    exposed_headers:
      - "Content-Length"
      - "Authorization"
    allow_credentials: true
    max_age: 86400

log:
  level: info
  format: json
  disable_console: false
  path: logs
  max_size_mb: 100
  max_backups: 30
  max_age_days: 7
  compress: true

data:
  database:
    driver: mysql
    source: "${mysql_user}:${mysql_password}@tcp(${mysql_host}:${mysql_port})/${mysql_database}?parseTime=True&loc=Local"
  redis:
    addr: "${redis_host}:${redis_port}"
    password: "${redis_password}"
    read_timeout: 0.2s
    write_timeout: 0.2s
    db: 0
    pool_size: 10
    min_idle_conns: 5

app:
  site:
    host: "${site_host}"
    site_name: "NPanel"
    site_desc: "Professional Panel Management System"
    site_logo: "/assets/logo.png"
    keywords: "panel,management,npanel"
    custom_html: ""
    custom_data: ""

  verify:
    turnstile_site_key: ""
    enable_login_verify: false
    enable_register_verify: false
    enable_reset_password_verify: false

  mobile:
    enable: false
    enable_whitelist: false
    whitelist: []

  email:
    enable: false
    enable_verify: false
    enable_domain_suffix: false
    domain_suffix_list: ""

  register:
    stop_register: false
    enable_ip_register_limit: false
    ip_register_limit: 5
    ip_register_limit_duration: 3600
    enable_trial: false
    trial_subscribe: 7
    trial_time_unit: "day"
    trial_time: 7

  invite:
    forced_invite: false
    referral_percentage: 10
    only_first_purchase: true

  subscribe:
    single_model: false
    subscribe_path: "/subscribe"
    subscribe_domain: ""
    pan_domain: false
    user_agent_limit: false
    user_agent_list: ""

  admin:
    email: "${admin_email}"
    password: "${admin_password}"
    algo: "default"
EOF
    chmod 0600 "${CONFIG_FILE}"
}

download_release() {
    CLEAN_VERSION="${VERSION#v}"
    PACKAGE="npanel-backend-${CLEAN_VERSION}-linux-${ARCH}"
    ARCHIVE="${PACKAGE}.tar.gz"
    BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
    TMP_DIR="$(mktemp -d)"

    trap 'rm -rf "${TMP_DIR:-}"' EXIT

    info "下载 NPanel Backend ${VERSION} (${ARCH})"
    curl -fL --retry 3 --connect-timeout 15 \
        "${BASE_URL}/${ARCHIVE}" \
        -o "${TMP_DIR}/${ARCHIVE}"

    if curl -fsSL "${BASE_URL}/SHA256SUMS" -o "${TMP_DIR}/SHA256SUMS"; then
        info "校验 SHA256SUMS"
        (
            cd "${TMP_DIR}"
            grep "${ARCHIVE}$" SHA256SUMS | sha256sum -c -
        )
    else
        warn "未找到 SHA256SUMS，跳过校验"
    fi
}

install_binary_files() {
    local extracted_dir

    info "安装到 ${INSTALL_DIR}"
    mkdir -p "${INSTALL_DIR}" "${CONFIG_DIR}"
    tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "${TMP_DIR}"

    extracted_dir="${TMP_DIR}/${PACKAGE}"
    [[ -x "${extracted_dir}/${BIN_NAME}" ]] || error "发布包中未找到可执行文件：${BIN_NAME}"

    install -m 0755 "${extracted_dir}/${BIN_NAME}" "${INSTALL_DIR}/${BIN_NAME}"
    cp -f "${extracted_dir}/README.md" "${INSTALL_DIR}/README.md" 2>/dev/null || true
    cp -f "${extracted_dir}/README.zh-CN.md" "${INSTALL_DIR}/README.zh-CN.md" 2>/dev/null || true
    cp -f "${extracted_dir}/LICENSE" "${INSTALL_DIR}/LICENSE" 2>/dev/null || true
    cp -f "${extracted_dir}/openapi.yaml" "${INSTALL_DIR}/openapi.yaml" 2>/dev/null || true
}

write_systemd_service() {
    info "写入 systemd 服务：${SERVICE_NAME}.service"
    cat >"/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=NPanel Backend Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/${BIN_NAME} -conf ${CONFIG_DIR}
Restart=always
RestartSec=5
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF
}

start_systemd_service() {
    info "重载 systemd 并启动服务"
    systemctl daemon-reload
    systemctl enable "${SERVICE_NAME}"
    systemctl restart "${SERVICE_NAME}"

    info "服务状态"
    systemctl --no-pager --full status "${SERVICE_NAME}" || true
}

print_binary_summary() {
    cat <<EOF

${green}NPanel Backend ${VERSION} 二进制模式安装完成${plain}

安装目录: ${INSTALL_DIR}
配置文件: ${CONFIG_FILE}
服务名称: ${SERVICE_NAME}
HTTP API: http://服务器IP:8081
gRPC: 服务器IP:9012

常用命令:
  systemctl status ${SERVICE_NAME}
  journalctl -u ${SERVICE_NAME} -f
  systemctl restart ${SERVICE_NAME}

MySQL: ${MYSQL_HOST}:${MYSQL_PORT}/${MYSQL_DATABASE}
Redis: ${REDIS_HOST}:${REDIS_PORT}
默认管理员: ${ADMIN_EMAIL}
默认密码: ${ADMIN_PASSWORD}

二进制模式不会安装 Docker，也不会安装 MySQL/Redis；请确认上面的数据库服务可从本机访问。
EOF
}

install_binary_mode() {
    download_release
    install_binary_files
    collect_binary_config
    write_binary_config
    write_systemd_service
    start_systemd_service
    print_binary_summary
}

main() {
    parse_args "$@"
    require_root
    detect_os
    detect_arch
    detect_pkg_manager
    prompt_install_mode
    install_packages
    fetch_latest_version

    if [[ "${MODE}" == "docker" ]]; then
        install_docker_mode
    else
        install_binary_mode
    fi
}

main "$@"
