#!/usr/bin/env bash
set -euo pipefail
umask 077
carpeta=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
exec python3 "$carpeta/certificados.py" "$@"
