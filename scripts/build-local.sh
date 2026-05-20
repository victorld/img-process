#!/bin/sh

set -eu

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

cd "$ROOT_DIR"
mkdir -p bin log

npm --prefix web ci
npm --prefix web run build

go build -mod=vendor -o "$ROOT_DIR/bin/img-process-web" ./main/webserver/webserver_main.go
