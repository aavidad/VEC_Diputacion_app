#!/usr/bin/env bash
# Reversión de CT-000142 en ROLLBACK sobre un clon desechable de la principal
# (datos sintéticos) SIN la migración aplicada. Uso:
#   ct142_reversion_rollback.sh <contenedor_postgres>
# 1) UP + DOWN en una transacción: el DOWN se deniega porque hay números
#    asignados, y la base queda intacta.
# 2) UP, vaciado sintético de la historia (solo dentro de la transacción) y
#    DOWN: las cuatro consultas recuperan cuerpo, propietario, ACL y
#    configuración exactos, y desaparecen los objetos nuevos. ROLLBACK final.
set -euo pipefail
contenedor=${1:?contenedor PostgreSQL con el clon}
raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
mig="$raiz/deploy/postgresql/contratacion_temporal/migraciones"
up=$(grep -vxE 'BEGIN;|COMMIT;' "$mig/000142_numeracion_anual_expedientes_anteriores.up.sql")
down=$(grep -vxE 'BEGIN;|COMMIT;' "$mig/000142_numeracion_anual_expedientes_anteriores.down.sql")
psql_super() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }

instantanea="SELECT string_agg(p.oid::regprocedure::text || '|' || encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')
  || '|' || coalesce(p.proacl::text,'') || '|' || coalesce(p.proconfig::text,'') || '|' || p.proowner::regrole::text
  || '|' || p.prosecdef::text, E'\n' ORDER BY p.oid::regprocedure::text)
  FROM pg_proc p WHERE p.oid IN (
   'vec_contratacion_temporal.materializar_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1)'::regprocedure,
   'vec_contratacion_temporal.contar_totales_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,text)'::regprocedure,
   'vec_contratacion_temporal.materializar_detalle_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,numeric)'::regprocedure,
   'vec_contratacion_temporal.expedientes_centro_ct124(jsonb,text)'::regprocedure)"
estado="SELECT (SELECT ultimo FROM vec_contratacion_temporal.numeracion_expedientes WHERE anio=2026)::text
  || '|' || (SELECT secuencia_outbox FROM vec_contratacion_temporal.control_cadenas_expediente_integral)::text
  || '|' || coalesce(to_regclass('vec_contratacion_temporal.numeracion_anual_asignada')::text,'-')"

antes=$(psql_super -At -c "$instantanea")
estado_antes=$(psql_super -At -c "$estado")

# 1) Con historia, el DOWN se deniega dentro de la misma transacción.
if salida=$(printf 'BEGIN;\n%s\n%s\nCOMMIT;\n' "$up" "$down" | psql_super 2>&1); then
  echo 'FALLO: DOWN admitido con números asignados' >&2; exit 1
fi
grep -q 'reversión denegada, hay números anuales asignados' <<<"$salida" \
  || { echo "FALLO: error inesperado: $salida" >&2; exit 1; }
[ "$(psql_super -At -c "$estado")" = "$estado_antes" ] || { echo 'FALLO: la base cambió' >&2; exit 1; }
[ "$(psql_super -At -c "$instantanea")" = "$antes" ] || { echo 'FALLO: consultas cambiadas' >&2; exit 1; }
echo 'ok DOWN denegado con historia; base intacta'

# 2) Sin historia, el DOWN restaura exactamente las consultas.
despues=$(printf '\\o /dev/null\nBEGIN;\n%s\nRESET ROLE;\nALTER TABLE vec_contratacion_temporal.numeracion_anual_asignada DISABLE TRIGGER USER;\nDELETE FROM vec_contratacion_temporal.numeracion_anual_asignada;\nALTER TABLE vec_contratacion_temporal.numeracion_anual_asignada ENABLE TRIGGER USER;\n%s\nRESET ROLE;\n\\o\n\\pset tuples_only on\n\\pset format unaligned\n%s;\n%s;\nROLLBACK;\n' \
  "$up" "$down" "$instantanea" "$estado" | psql_super)
consultas_tras_down=$(head -n 4 <<<"$despues")
[ "$consultas_tras_down" = "$antes" ] || { echo "FALLO: consultas no restauradas" >&2; diff <(echo "$antes") <(echo "$consultas_tras_down") >&2 || true; exit 1; }
tail -n 1 <<<"$despues" | grep -q '|-$' || { echo 'FALLO: la tabla sigue tras DOWN' >&2; exit 1; }
[ "$(psql_super -At -c "$estado")" = "$estado_antes" ] || { echo 'FALLO: la base cambió' >&2; exit 1; }
echo 'ok UP/DOWN exacto en ROLLBACK: consultas restauradas, objetos retirados, base intacta'
