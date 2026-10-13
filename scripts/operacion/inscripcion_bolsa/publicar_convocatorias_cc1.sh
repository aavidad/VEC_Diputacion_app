#!/usr/bin/env bash
set -euo pipefail

# Genera una carga SQL revisable. La conexión y la ejecución corresponden a Dirección.
if [[ $# -lt 1 || -z ${1:-} ]]; then
  echo 'Uso: publicar_convocatorias_cc1.sh <directorio_salida> [--unidad-ref REF --ambito-ref REF]' >&2
  exit 64
fi
raiz=$(git rev-parse --show-toplevel)
origen="$raiz/deploy/principal/inscripcion_recorrido_20261013/catalogos"
python3 "$origen/generar_carga_cc1.py" "$origen/cc1_inscripcion_sintetica.json" "$@"
echo "Carga preparada: $1/cargar_cc1_y_convocatorias.sql"
echo 'Dirección ejecuta el SQL en la base autorizada y coteja manifiesto.json con las lecturas BC9/CC11.'
