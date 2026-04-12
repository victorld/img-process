#!/bin/sh

set -eu

mkdir -p /app/log

exec /app/img-process
