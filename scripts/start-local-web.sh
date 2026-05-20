#!/bin/sh

set -eu

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
APP_BIN="$ROOT_DIR/bin/img-process-web"
LOG_FILE="$ROOT_DIR/log/local-web.log"
PID_FILE="$ROOT_DIR/log/local-web.pid"
STARTED_FILE="$ROOT_DIR/log/local-web.started"

usage() {
  cat <<EOF
Usage: $0 [daemon]
EOF
}

if [ ! -x "$APP_BIN" ]; then
  echo "missing executable: $APP_BIN" >&2
  echo "run scripts/build-local.sh first" >&2
  exit 1
fi

cd "$ROOT_DIR"
mkdir -p log

export IMG_PROCESS_CONFIG="$ROOT_DIR/config.yaml"
export TZ="${TZ:-Asia/Shanghai}"

start_foreground() {
  exec "$APP_BIN"
}

listening_pid() {
  lsof -nP -tiTCP:8081 -sTCP:LISTEN 2>/dev/null | head -n 1
}

start_daemon() {
  if [ -f "$PID_FILE" ]; then
    pid="$(cat "$PID_FILE" 2>/dev/null || true)"
    if [ -n "$pid" ] && kill -0 "$pid" >/dev/null 2>&1; then
      exit 0
    fi
    rm -f "$PID_FILE" "$STARTED_FILE"
  fi

  listener_pid="$(listening_pid || true)"
  if [ -n "$listener_pid" ]; then
    listener_command="$(ps -p "$listener_pid" -o command= 2>/dev/null || true)"
    case "$listener_command" in
      *"$APP_BIN"*)
        echo "$listener_pid" >"$PID_FILE"
        date +%s >"$STARTED_FILE"
        exit 0
        ;;
    esac
    echo "port 8081 is already used by pid $listener_pid: $listener_command" >&2
    exit 1
  fi

  {
    echo "[$(date '+%Y-%m-%d %H:%M:%S %z')] starting local img-process web"
  } >>"$LOG_FILE" 2>&1

  nohup "$0" >>"$LOG_FILE" 2>&1 &
  echo $! >"$PID_FILE"
  date +%s >"$STARTED_FILE"
}

case "${1:-}" in
  "")
    start_foreground
    ;;
  daemon)
    start_daemon
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac
