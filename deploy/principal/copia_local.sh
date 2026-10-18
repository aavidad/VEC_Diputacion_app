#!/usr/bin/env bash
set -Eeuo pipefail
umask 077
exec python3 "$(dirname -- "${BASH_SOURCE[0]}")/copia_local.py" "$@"
