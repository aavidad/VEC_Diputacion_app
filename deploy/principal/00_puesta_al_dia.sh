#!/usr/bin/env bash
# Ensambla, en orden causal, las migraciones que llevan una réplica detenida en
# Bolsa `000008` / AD3 `000031` hasta la preimagen que exige `02_migraciones.sql`
# (AD3 `000046`, Bolsa `000014`). Imprime una sola transacción en la salida
# estándar; el `:finalizar` lo fija quien la ejecuta (`ROLLBACK` para ensayar,
# `COMMIT` para aplicar). No contiene secretos ni datos personales.
#
#   ./deploy/principal/00_puesta_al_dia.sh > /tmp/puesta_al_dia.sql
#
# La réplica principal de desarrollo tenía instalada AD3 `000038` pero no
# `000032`: el ensamblado la incluye porque AD3 `000044` la exige. Una réplica
# que ya las tenga rechaza el paquete en su propia precondición, sin efectos.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)

migraciones=(
  autorizacion_atestada_v3/migraciones/000032_consumidor_despacho_correo_llamamiento
  autorizacion_atestada_v3/migraciones/000043_consumidor_mi_bolsa
  bolsa_llamamientos/migraciones/000010_consulta_participaciones_propias
  autorizacion_atestada_v3/migraciones/000044_consumidor_borrador_llamamiento
  bolsa_llamamientos/migraciones/000011_borrador_llamamiento
  autorizacion_atestada_v3/migraciones/000045_consumidor_situacion_participacion
  bolsa_llamamientos/migraciones/000012_situacion_participacion
  autorizacion_atestada_v3/migraciones/000046_consumidor_contacto_participacion
  bolsa_llamamientos/migraciones/000013_contacto_participacion
  bolsa_llamamientos/migraciones/000014_sustitucion_bolsa
)

printf '%s\n' '\set ON_ERROR_STOP on' 'BEGIN;'
for relativa in "${migraciones[@]}"; do
  fichero=$repo/deploy/postgresql/$relativa.up.sql
  printf -- '-- INICIO deploy/postgresql/%s.up.sql\n' "$relativa"
  grep -vE '^\\set ON_ERROR_STOP on$|^BEGIN;$|^COMMIT;$' -- "$fichero"
  printf -- '-- FIN deploy/postgresql/%s.up.sql\n' "$relativa"
done
printf '%s\n' ':finalizar;'
