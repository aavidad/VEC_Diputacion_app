#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
sql="$raiz/scripts/operacion/inscripcion_bolsa/preparar_accesos.sql"

if [[ "${1:-}" != --aplicar || $# -ne 1 ]]; then
  echo 'Uso: PGSERVICE=servicio_privado preparar_accesos.sh --aplicar' >&2
  exit 2
fi
if [[ -z "${PGSERVICE:-}" || -n "${PGPASSWORD:-}" ]]; then
  echo 'Se exige PGSERVICE privado; usar PGPASSFILE seguro, no PGPASSWORD.' >&2
  exit 2
fi
command -v psql >/dev/null || { echo 'Falta psql.' >&2; exit 2; }
psql -X -w -q -v ON_ERROR_STOP=1 --single-transaction -f "$sql"
echo 'LOGIN y membresías mínimos comprobados; contraseñas, permisos V3 y despliegue pendientes.'
