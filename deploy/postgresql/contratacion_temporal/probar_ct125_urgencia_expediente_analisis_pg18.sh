#!/usr/bin/env bash
# Ensayo de CT125 (urgencia declarada en el análisis y fachada v4 del cuadro
# RRHH) en PostgreSQL 18.4 desechable sobre la estructura real restaurada de
# la principal (volcado con datos sintéticos). Instala CT110 si el volcado no
# la tiene; después CT125: ROLLBACK, UP/DOWN/UP, doble aplicación, ACL con
# roles reales, pruebas funcionales y negativas (con un doble explícito de la
# fachada v3, cuya autorización atestada tiene su propio ensayo), reinicio y
# DOWN protegido por la historia.
# Uso: probar_ct125_urgencia_expediente_analisis_pg18.sh GLOBALS_SQL VOLCADO_PG_DUMP
# El contenedor usa --rm, sin red ni volúmenes anónimos; sus datos viven en
# /dev/shm/vec-pg-ct125-<pid> y se borran al terminar.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta el volcado de roles (pg_dumpall --globals-only)}
volcado=${2:?falta el volcado de la base (pg_dump -Fc)}
[[ -s $globales && -s $volcado ]] || { echo 'Faltan los volcados' >&2; exit 2; }

nombre="vec-pg-ct125-$$"
datos="/dev/shm/$nombre"
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  docker run --rm -v "/dev/shm:/limpiar" postgres:18.4 rm -rf "/limpiar/$nombre" >/dev/null 2>&1 || rm -rf "$datos" 2>/dev/null || true
}
trap limpiar EXIT
mkdir -p "$datos"
docker run -d --rm --network none --name "$nombre" -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$datos:/var/lib/postgresql" postgres:18.4 >/dev/null
esperar() {
  for _ in $(seq 1 240); do
    if docker logs "$nombre" 2>&1 | grep -q 'PostgreSQL init process complete' && docker exec "$nombre" pg_isready -q -U postgres; then return 0; fi
    sleep 0.5
  done
  return 1
}
esperar
if [[ -n $(docker inspect --format '{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{end}}{{end}}' "$nombre") ]]; then
  echo 'volumen anónimo inesperado' >&2; exit 65
fi
run() { docker exec -i "$nombre" psql -X -q -o /dev/null -v ON_ERROR_STOP=1 -U postgres -d postgres; }
escalar() { docker exec "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
[[ $(escalar 'SHOW server_version') == 18.4* ]] || { echo 'PostgreSQL 18.4 no disponible' >&2; exit 2; }
ok() { printf 'OK %s\n' "$1"; }
falla_con() {
  if salida=$(run <"$1" 2>&1); then echo "FALLO: se esperaba rechazo de $(basename "$1")" >&2; exit 1; fi
  grep -q "$2" <<<"$salida" || { echo "FALLO: rechazo inesperado de $(basename "$1"): $salida" >&2; exit 1; }
}
prueba() {
  local marca=$1 salida; shift
  salida=$(cat "$@" | docker exec -i "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres 2>&1) || { printf '%s\n' "$salida" | tail -5 >&2; exit 1; }
  grep -q "$marca" <<<"$salida" || { echo "FALLO: falta $marca" >&2; exit 1; }
  ! grep -q '^FALLO' <<<"$salida" || { grep '^FALLO' <<<"$salida" >&2; exit 1; }
  ok "$(grep -c '^ok$' <<<"$salida") comprobaciones hasta $marca"
}

echo '== Restaurando roles y estructura real'
docker exec -i "$nombre" psql -X -q -U postgres -d postgres <"$globales" >/dev/null 2>&1 || true
docker exec -i "$nombre" pg_restore -U postgres -d postgres <"$volcado" >/dev/null 2>&1 || true
[[ $(escalar "SELECT count(*) > 0 FROM vec_contratacion_temporal.confirmacion_operacion_analisis") == t ]] || { echo 'Restauración incompleta o sin análisis confirmados' >&2; exit 2; }

ct=$repo/deploy/postgresql/contratacion_temporal/migraciones
pruebas=$repo/deploy/postgresql/contratacion_temporal/pruebas_sql
v3="vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v3(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
if [[ $(escalar "SELECT to_regprocedure('$v3') IS NULL") == t ]]; then
  echo '== Cadena previa: CT110 (fachada v3 del cuadro)'
  run <"$ct/000110_fase_desde_cuadro_rrhh.up.sql"
fi

m=$ct/000125_urgencia_expediente_analisis
objetos="SELECT (to_regclass('vec_contratacion_temporal.urgencia_expediente_analisis') IS NOT NULL)::int
  + (to_regprocedure('vec_contratacion_temporal.registrar_urgencia_analisis_v1(text,text)') IS NOT NULL)::int
  + (to_regprocedure('${v3/_v3/_v4}') IS NOT NULL)::int"
echo '== CT125: ROLLBACK, UP, doble UP, DOWN, doble DOWN, UP'
sed 's/^COMMIT;$/ROLLBACK;/' "$m.up.sql" | run
[[ $(escalar "$objetos") == 0 ]] || { echo 'FALLO: ROLLBACK dejó objetos' >&2; exit 1; }; ok 'ROLLBACK sin rastro'
run <"$m.up.sql"; [[ $(escalar "$objetos") == 3 ]] || { echo 'FALLO: UP incompleto' >&2; exit 1; }; ok 'UP'
falla_con "$m.up.sql" 'CT-000125 ya instalada'; ok 'doble UP rechazado'
run <"$m.down.sql"; [[ $(escalar "$objetos") == 0 ]] || { echo 'FALLO: DOWN incompleto' >&2; exit 1; }; ok 'DOWN sin historia'
falla_con "$m.down.sql" 'CT-000125 no instalada'; ok 'doble DOWN rechazado'
run <"$m.up.sql"; ok 'UP de nuevo'

echo '== Urgencia en la confirmación del análisis y fachada v4'
prueba 'CT125 OK' "$pruebas/ct125_urgencia_expediente_analisis.sql"

echo '== Reinicio de PostgreSQL'
docker restart "$nombre" >/dev/null
for _ in $(seq 1 120); do docker exec "$nombre" pg_isready -q -U postgres && break; sleep 0.5; done
sleep 1
tras=$(escalar "SELECT count(*)||'/'||min(motivo) FROM vec_contratacion_temporal.urgencia_expediente_analisis")
[[ $tras == '1/Cierre del servicio de ayuda a domicilio de Loja' ]] || { echo "FALLO: urgencia tras reinicio: $tras" >&2; exit 1; }
[[ $(escalar "$objetos") == 3 ]] || { echo 'FALLO: objetos tras reinicio' >&2; exit 1; }; ok 'historia conservada tras reinicio'
falla_con "$m.down.sql" 'reversión denegada'; ok 'DOWN con historia rechazado'
echo 'CT125 verificada'
