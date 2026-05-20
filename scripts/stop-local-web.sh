#!/bin/sh

set -eu

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
APP_BIN="$ROOT_DIR/bin/img-process-web"
PID_FILE="$ROOT_DIR/log/local-web.pid"
STARTED_FILE="$ROOT_DIR/log/local-web.started"

is_project_pid() {
  pid="$1"
  command="$(ps -p "$pid" -o command= 2>/dev/null || true)"
  case "$command" in
    *"$APP_BIN"*) return 0 ;;
    *) return 1 ;;
  esac
}

collect_pids() {
  {
    cat "$PID_FILE" 2>/dev/null || true
    pgrep -f "$APP_BIN" 2>/dev/null || true
  } | awk 'NF && !seen[$1]++ { print $1 }' | while read -r pid; do
    if kill -0 "$pid" >/dev/null 2>&1 && is_project_pid "$pid"; then
      echo "$pid"
    fi
  done
}

pids="$(collect_pids || true)"

for pid in $pids; do
  kill "$pid" >/dev/null 2>&1 || true
done

count=0
while [ -n "$(collect_pids || true)" ]; do
  count=$((count + 1))
  if [ "$count" -ge 20 ]; then
    for pid in $(collect_pids || true); do
      kill -9 "$pid" >/dev/null 2>&1 || true
    done
    break
  fi
  sleep 1
done

rm -f "$PID_FILE" "$STARTED_FILE"
