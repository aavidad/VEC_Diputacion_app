#!/usr/bin/env bash
set -euo pipefail
umask 077
carpeta=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
raiz=$(cd -- "$carpeta/../../.." && pwd)
case "${1:-}" in
  --comprobar) ;;
  --aplicar) [[ ${2:-} == PUBLICA-PLANTILLAS-V3 ]] || { echo 'Falta --aplicar PUBLICA-PLANTILLAS-V3' >&2; exit 2; } ;;
  *) echo 'Uso: publicar.sh --comprobar | --aplicar PUBLICA-PLANTILLAS-V3' >&2; exit 2 ;;
esac
binario=${VEC_PLANTILLAS_V3_BIN:-$carpeta/publicar-plantillas-v3}
if [[ ! -x $binario ]]; then
  echo 'Falta el ejecutable. Compílelo como indica LEEME.md y establezca VEC_PLANTILLAS_V3_BIN.' >&2
  exit 2
fi
exec "$binario" --catalogo "${VEC_PLANTILLAS_V3_CATALOGO:-$raiz/data/catalogos/contratacion-temporal/plantillas_documentos.v3.json}" "$@"
