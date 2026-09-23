#!/usr/bin/env bash
# Ensambla solo las dependencias nuevas de Dietas R1 desde fuentes canonicas.
# Requiere el nucleo AD3-48 y las migraciones de main ya instalados.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)

migraciones=(
  dietas_borradores/roles_up.sql
  personal/migraciones/000007_relacion_empleado_dietas.up.sql
  autorizacion_atestada_v3/migraciones/000049_consumidor_personal_dietas.up.sql
  personal/migraciones/000008_consulta_relaciones_propias_dietas.up.sql
  dietas_borradores/migraciones/000001_borrador_comision_durable.up.sql
)

printf '%s\n' '\set ON_ERROR_STOP on' 'BEGIN;'
for relativa in "${migraciones[@]}"; do
  fichero=$repo/deploy/postgresql/$relativa
  printf -- '-- INICIO deploy/postgresql/%s\n' "$relativa"
  grep -vE '^\\set ON_ERROR_STOP on$|^BEGIN;$|^COMMIT;$' -- "$fichero"
  printf -- '-- FIN deploy/postgresql/%s\n' "$relativa"
done
printf '%s\n' ':finalizar;'
