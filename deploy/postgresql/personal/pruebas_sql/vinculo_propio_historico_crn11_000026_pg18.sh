#!/usr/bin/env bash
# Preparación CRN11. No abre conexiones ni aplica SQL.
# La candidata ejecutable exige AD148 causal + AD149 matriz exacta + Personal26,
# doble revisión y ensayo completo sobre el clon autorizado por Dirección.
set -euo pipefail
printf '%s\n' 'PARO: CRN11 es borrador no instalable. Falta AD149 nominal sobre la postimagen causal AD148; no se ha ejecutado ensayo SQL.' >&2
exit 78
