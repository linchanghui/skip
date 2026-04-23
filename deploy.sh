#!/usr/bin/env bash

# Skip 部署脚本（AWS EC2）
# 默认场景：与 dota2master 同域名，通过 Caddy 子路径 /skip* 反代到本机 18181。
# 说明：deploy/nginx-dota2master-skip.conf.example 仅作 Nginx 备用示例，当前线上入口以 Caddy 为准。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

EC2_HOST="${EC2_HOST:-ec2-user@15.164.92.89}"
REMOTE_DIR="${REMOTE_DIR:-~/skip}"
LOCAL_PROJECT="${LOCAL_PROJECT:-$SCRIPT_DIR}"
SERVICE_PORT="${SERVICE_PORT:-18181}"
MOUNT_BASE="${MOUNT_BASE:-/skip}"
VITE_BASE_PATH="${VITE_BASE_PATH:-/skip/}"
STOP_DOTA2="${STOP_DOTA2:-0}"
SKIP_WEB_BUILD="${SKIP_WEB_BUILD:-0}" # 1 跳过前端构建
RELOAD_CADDY="${RELOAD_CADDY:-0}"     # 1 远端重载 Caddy
CADDY_CONTAINER="${CADDY_CONTAINER:-caddy}"
BINARY_NAME="${BINARY_NAME:-skip-server}"
GO_BIN="${GO_BIN:-go}"
NPM_BIN="${NPM_BIN:-npm}"

SSH_BASE_OPTS=(-o BatchMode=yes -o ConnectTimeout=20 -o ConnectionAttempts=1 -o GSSAPIAuthentication=no)
RSYNC_SSH="ssh ${SSH_BASE_OPTS[*]}"

echo "开始部署 Skip（端口 ${SERVICE_PORT}，子路径 ${MOUNT_BASE}）..."
echo "目标主机: ${EC2_HOST}"
echo "远程目录: ${REMOTE_DIR}"
echo "本地项目: ${LOCAL_PROJECT}"
echo "Go 命令: ${GO_BIN}"
echo "NPM 命令: ${NPM_BIN}"

cd "${LOCAL_PROJECT}"
[[ -f go.mod ]] || { echo "错误: 在 ${LOCAL_PROJECT} 未找到 go.mod"; exit 1; }
[[ -d cmd/server ]] || { echo "错误: 未找到 cmd/server"; exit 1; }

GO_VER_STR="$(${GO_BIN} version 2>/dev/null || true)"
if [[ -z "${GO_VER_STR}" ]]; then
  echo "错误: 无法执行 ${GO_BIN}，请通过 GO_BIN 指定可用 Go（需 >= 1.22）"
  exit 1
fi
if ! awk -v s="${GO_VER_STR}" 'BEGIN {
  n=split(s,a," ");
  v=a[3];
  sub(/^go/,"",v);
  split(v,p,".");
  major=p[1]+0; minor=p[2]+0;
  if (major>1 || (major==1 && minor>=22)) exit 0;
  exit 1
}'; then
  echo "错误: 当前 Go 版本过低：${GO_VER_STR}（要求 >= 1.22）"
  exit 1
fi

if [[ -d web && "${SKIP_WEB_BUILD}" != "1" ]]; then
  echo "正在构建前端（VITE_BASE_PATH=${VITE_BASE_PATH}）..."
  (
    cd web
    ${NPM_BIN} ci
    VITE_BASE_PATH="${VITE_BASE_PATH}" ${NPM_BIN} run build
  )
fi

echo "正在本地交叉编译 Linux amd64 二进制..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 "${GO_BIN}" build -ldflags="-s -w" -o "${BINARY_NAME}" ./cmd/server

echo "正在同步代码与二进制到 EC2..."
rsync -avz -e "${RSYNC_SSH}" \
  --exclude='.git/' \
  --exclude='node_modules/' \
  --exclude='.idea/' \
  --exclude='.vscode/' \
  --exclude='data/' \
  "${LOCAL_PROJECT}/" "${EC2_HOST}:${REMOTE_DIR}/"

if [[ "${STOP_DOTA2}" == "1" ]]; then
  echo "远程停止 dota2-web 容器（STOP_DOTA2=1）..."
  ssh "${SSH_BASE_OPTS[@]}" -n "${EC2_HOST}" "docker stop dota2-web || true"
else
  echo "保留 dota2-web（主站不受影响）。"
fi

echo "正在远程启动 Skip..."
ssh "${SSH_BASE_OPTS[@]}" "${EC2_HOST}" env \
  REMOTE_DIR="${REMOTE_DIR}" \
  SERVICE_PORT="${SERVICE_PORT}" \
  MOUNT_BASE="${MOUNT_BASE}" \
  BINARY_NAME="${BINARY_NAME}" \
  RELOAD_CADDY="${RELOAD_CADDY}" \
  CADDY_CONTAINER="${CADDY_CONTAINER}" \
  bash -s <<'REMOTE_SCRIPT'
set -euo pipefail
case "${REMOTE_DIR}" in
  "~/"*) REMOTE_DIR="${HOME}/${REMOTE_DIR#~/}" ;;
  "~") REMOTE_DIR="${HOME}" ;;
esac

BASE_NO_SLASH="${MOUNT_BASE%/}"
if [[ -z "${BASE_NO_SLASH}" ]]; then
  BASE_NO_SLASH="/"
fi

mkdir -p "${REMOTE_DIR}/data"
chmod +x "${REMOTE_DIR}/${BINARY_NAME}"
cd "${REMOTE_DIR}"

if [[ -f "${REMOTE_DIR}/env" ]]; then
  set -a
  # shellcheck disable=SC1091
  . "${REMOTE_DIR}/env"
  set +a
fi

if command -v timeout >/dev/null 2>&1; then
  timeout 15 fuser -k "${SERVICE_PORT}/tcp" >/dev/null 2>&1 || true
else
  fuser -k "${SERVICE_PORT}/tcp" >/dev/null 2>&1 || true
fi
sleep 1

nohup "./${BINARY_NAME}" \
  -addr ":${SERVICE_PORT}" \
  -base "${BASE_NO_SLASH}" \
  -static "./web/dist" \
  -db "./data/app.db" \
  >> "./skip.log" 2>&1 </dev/null &
disown || true

if [[ "${RELOAD_CADDY}" == "1" ]]; then
  docker exec "${CADDY_CONTAINER}" caddy reload --config /etc/caddy/Caddyfile \
    || docker restart "${CADDY_CONTAINER}" \
    || true
fi

for i in $(seq 1 20); do
  if curl -fsS "http://127.0.0.1:${SERVICE_PORT}${BASE_NO_SLASH}/healthz" >/dev/null; then
    echo "healthz ok"
    exit 0
  fi
  sleep 1
done

echo "错误: Skip 启动后 healthz 检查失败，请查看 ${REMOTE_DIR}/skip.log"
exit 1
REMOTE_SCRIPT

echo "清理本地交叉编译产物..."
rm -f "${LOCAL_PROJECT}/${BINARY_NAME}"

echo "部署完成。"
echo "日志查看: ssh ${EC2_HOST} 'tail -n 200 ${REMOTE_DIR}/skip.log'"
echo "线上验证: https://dota2master.com${MOUNT_BASE}/"
echo "Nginx 示例（备用）: deploy/nginx-dota2master-skip.conf.example"
