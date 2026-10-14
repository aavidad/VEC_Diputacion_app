#!/usr/bin/env bash
set -euo pipefail
umask 077
carpeta=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
raiz=$(cd -- "$carpeta/../../.." && pwd)
case "${1:-}" in
  --comprobar) [[ $# == 1 ]] || exit 2 ;;
  --aplicar) [[ $# == 2 && ${2:-} == PUBLICA-INFORME-PREVIO-V4 ]] || exit 2 ;;
  *) exit 2 ;;
esac
binario=${VEC_PLANTILLAS_V3_BIN:-$carpeta/publicar-plantillas}
[[ -x $binario ]] || { printf '%s\n' 'error=binario_publicacion_ausente' >&2; exit 2; }
exec "$binario" --catalogo "$raiz/data/catalogos/contratacion-temporal/plantillas_documentos.v4.json" \
  --operacion "$carpeta/operacion_informe_v4.json" "$@"
