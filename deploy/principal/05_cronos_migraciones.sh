#!/usr/bin/env bash
# Ensambla solo las dependencias nuevas de Cronos R1 (marcajes) desde fuentes canonicas.
# Requiere el nucleo AD3-48 y las migraciones de main ya instalados.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)

migraciones=(
  cronos_v1/migraciones/000001_esquema_marcajes.up.sql
  autorizacion_atestada_v3/migraciones/000051_consumidor_marcaje_propio_cronos.up.sql
  cronos_v1/migraciones/000002_registrar_marcaje_propio.up.sql
)

printf '%s\n' '\set ON_ERROR_STOP on' 'BEGIN;'
for relativa in "${migraciones[@]}"; do
  fichero=$repo/deploy/postgresql/$relativa
  printf -- '-- INICIO deploy/postgresql/%s\n' "$relativa"
  grep -vE '^\\set ON_ERROR_STOP on$|^BEGIN;$|^COMMIT;$' -- "$fichero"
  printf -- '-- FIN deploy/postgresql/%s\n' "$relativa"
done
printf '%s\n' ':finalizar;'
