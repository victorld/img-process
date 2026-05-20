#!/bin/sh

set -eu

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
MONIT_BIN="${MONIT_BIN:-$(command -v monit || true)}"
SOURCE_CONFIG="$ROOT_DIR/monit/img-process-web.monitrc"
RUNTIME_DIR="$ROOT_DIR/log/monit"
RUNTIME_CONFIG="$RUNTIME_DIR/img-process-web.monitrc"

usage() {
  cat <<EOF
Usage: $0 {start|stop|restart|reload|status|summary|validate|unmonitor}
EOF
}

require_monit() {
  if [ -z "$MONIT_BIN" ]; then
    echo "monit is not installed. Install it first, for example: brew install monit" >&2
    exit 1
  fi
}

prepare_config() {
  mkdir -p "$RUNTIME_DIR" "$ROOT_DIR/log"
  cp "$SOURCE_CONFIG" "$RUNTIME_CONFIG"
  chmod 700 "$RUNTIME_CONFIG"
}

build_local() {
  "$ROOT_DIR/scripts/build-local.sh"
}

cmd="${1:-status}"

case "$cmd" in
  start)
    require_monit
    prepare_config
    "$MONIT_BIN" -t -c "$RUNTIME_CONFIG"
    "$MONIT_BIN" -c "$RUNTIME_CONFIG"
    sleep 1
    "$MONIT_BIN" -c "$RUNTIME_CONFIG" monitor img-process-web >/dev/null 2>&1 || true
    "$MONIT_BIN" -c "$RUNTIME_CONFIG" start img-process-web
    ;;
  stop)
    require_monit
    prepare_config
    "$MONIT_BIN" -c "$RUNTIME_CONFIG" unmonitor img-process-web >/dev/null 2>&1 || true
    "$MONIT_BIN" -c "$RUNTIME_CONFIG" stop img-process-web >/dev/null 2>&1 || true
    "$ROOT_DIR/scripts/stop-local-web.sh" >/dev/null 2>&1 || true
    "$MONIT_BIN" -c "$RUNTIME_CONFIG" quit >/dev/null 2>&1 || true
    ;;
  restart)
    require_monit
    build_local
    prepare_config
    if "$MONIT_BIN" -c "$RUNTIME_CONFIG" summary >/dev/null 2>&1; then
      "$MONIT_BIN" -c "$RUNTIME_CONFIG" restart img-process-web
    else
      "$0" start
    fi
    ;;
  reload)
    require_monit
    prepare_config
    "$MONIT_BIN" -t -c "$RUNTIME_CONFIG"
    "$MONIT_BIN" -c "$RUNTIME_CONFIG" reload
    ;;
  status)
    require_monit
    prepare_config
    "$MONIT_BIN" -c "$RUNTIME_CONFIG" status
    ;;
  summary)
    require_monit
    prepare_config
    "$MONIT_BIN" -c "$RUNTIME_CONFIG" summary
    ;;
  validate)
    require_monit
    prepare_config
    "$MONIT_BIN" -t -c "$RUNTIME_CONFIG"
    ;;
  unmonitor)
    require_monit
    prepare_config
    "$MONIT_BIN" -c "$RUNTIME_CONFIG" unmonitor img-process-web
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac
