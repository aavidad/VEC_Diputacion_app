#!/usr/bin/env bash
set -euo pipefail
directorio=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
exec python3 "$directorio/runtime.py" check "$@"
