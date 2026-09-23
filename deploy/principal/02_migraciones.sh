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
  dietas_borradores/roles_up.sql
  personal/migraciones/000007_relacion_empleado_dietas
  autorizacion_atestada_v3/migraciones/000049_consumidor_personal_dietas
  personal/migraciones/000008_consulta_relaciones_propias_dietas
  dietas_borradores/migraciones/000001_borrador_comision_durable
)

printf '%s\n' '\set ON_ERROR_STOP on' 'BEGIN;'
for relativa in "${migraciones[@]}"; do
  ruta=$relativa.up.sql
  [[ $relativa == *.sql ]] && ruta=$relativa
  fichero=$repo/deploy/postgresql/$ruta
  printf -- '-- INICIO deploy/postgresql/%s\n' "$ruta"
  grep -vE '^\\set ON_ERROR_STOP on$|^BEGIN;$|^COMMIT;$' -- "$fichero"
  printf -- '-- FIN deploy/postgresql/%s\n' "$ruta"
done
printf '%s\n' ':finalizar;'
