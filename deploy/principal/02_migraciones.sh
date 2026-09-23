#!/usr/bin/env bash
# Ensambla B4/B7/B11 directamente desde sus migraciones canónicas. La salida es
# una única transacción; `:finalizar` lo fija quien ejecuta el paquete.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)

migraciones=(
  autorizacion_atestada_v3/migraciones/000047_consumidor_datos_contacto_participacion
  bolsa_llamamientos/migraciones/000016_datos_contacto_participacion
  autorizacion_atestada_v3/migraciones/000048_consumidor_emision_llamamiento
  bolsa_llamamientos/migraciones/000017_emision_llamamiento
  bolsa_llamamientos/migraciones/000021_rellenar_vinculos_candidato
)

printf '%s\n' '\set ON_ERROR_STOP on' 'BEGIN;'
for relativa in "${migraciones[@]}"; do
  fichero=$repo/deploy/postgresql/$relativa.up.sql
  printf -- '-- INICIO deploy/postgresql/%s.up.sql\n' "$relativa"
  grep -vE '^\\set ON_ERROR_STOP on$|^BEGIN;$|^COMMIT;$' -- "$fichero"
  printf -- '-- FIN deploy/postgresql/%s.up.sql\n' "$relativa"
done
printf '%s\n' ':finalizar;'
