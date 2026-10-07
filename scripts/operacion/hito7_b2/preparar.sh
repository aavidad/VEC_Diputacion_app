#!/usr/bin/env bash
set -euo pipefail
umask 077
if [[ $# != 6 ]]; then
  printf '%s\n' 'Uso: preparar.sh FUENTE INVENTARIO_CT IDEMPOTENCIA CONFIG_B2 DSN_GOBIERNO SALIDA_NUEVA' >&2
  exit 2
fi
fuente=$1
inventario=$2
idempotencia=$3
configuracion=$4
gobierno=$5
salida=$6
# Las dos PR están integradas; no se cambia su publicador ni sus contratos.
git -C "$fuente" merge-base --is-ancestor e3481ca2a09aec5b8554093a64195b3788378a65 HEAD
git -C "$fuente" merge-base --is-ancestor 2d004e79376c3f3af379bb2264c1e1a742fb9c9d HEAD
binario=$(mktemp "${TMPDIR:-/var/tmp}/vec-preparar-b2.XXXXXX")
trap 'rm -f -- "$binario"' EXIT
(cd "$fuente" && GOMAXPROCS=8 GOPROXY=off go build -p 8 -o "$binario" ./cmd/vec-preparar-material-interno)
"$binario" -inventario-ct "$inventario" -material-idempotencia "$idempotencia" \
  -incorporacion-config "$configuracion" -dsn-archivo "$gobierno" -salida "$salida"
printf '%s\n' 'Preparación completa. La activación usa el servidor.json resultante y los dos selectores del LEEME.'
