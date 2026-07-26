#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

if (($# != 1)) || [[ -z "$1" ]]; then
	printf 'Uso: %s OPCIONES_DE_GIT_LOG\n' "$0" >&2
	exit 2
fi

if ! command -v gitleaks >/dev/null 2>&1; then
	printf '%s\n' \
		'Envio bloqueado: gitleaks no esta instalado en el equipo.' >&2
	exit 2
fi

gitleaks git . \
	--log-opts="$1" \
	--redact \
	--no-banner \
	--no-color
