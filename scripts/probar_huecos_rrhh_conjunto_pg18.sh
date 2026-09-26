#!/usr/bin/env bash
# Ensayo conjunto de las migraciones de los huecos de RRHH (26/09/2026) en
# PostgreSQL 18.4 desechable sobre la estructura real restaurada de la
# principal (volcado con datos sintéticos). Primero lleva el volcado al estado
# de la principal (AD3-82…86, CT110…121 salvo CT117 y Bolsa 010, 019 y
# 021…040, que el volcado no trae); después instala, en el orden de despliegue,
# AD3-87, AD3-88, Bolsa 000041, CT122, CT123, CT124, CT125 y CT126: cada una
# con ROLLBACK sin rastro, UP, detección instalada y doble UP rechazado.
# Con todas juntas ejecuta las pruebas SQL de cada una, reinicia PostgreSQL,
# comprueba que siguen detectándose y que los DOWN con historia se niegan.
# Uso: probar_huecos_rrhh_conjunto_pg18.sh GLOBALS_SQL VOLCADO_PG_DUMP
# El contenedor usa --rm, sin red ni volúmenes anónimos; sus datos viven en
# /dev/shm/vec-pg-huecos-<pid> y se borran al terminar.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta el volcado de roles (pg_dumpall --globals-only)}
volcado=${2:?falta el volcado de la base (pg_dump -Fc)}
[[ -s $globales && -s $volcado ]] || { echo 'Faltan los volcados' >&2; exit 2; }

nombre="vec-pg-huecos-$$"
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
run() { docker exec -i "$nombre" psql -X -q -o /dev/null -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
escalar() { docker exec "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
[[ $(escalar 'SHOW server_version') == 18.4* ]] || { echo 'PostgreSQL 18.4 no disponible' >&2; exit 2; }
ok() { printf 'OK %s\n' "$1"; }
falla_con() { # $1 fichero, $2 texto esperado en el error
  if salida=$(run <"$1" 2>&1); then echo "FALLO: se esperaba rechazo de $(basename "$1")" >&2; exit 1; fi
  grep -q "$2" <<<"$salida" || { echo "FALLO: rechazo inesperado de $(basename "$1"): $salida" >&2; exit 1; }
}
prueba() { # $1 marca final, $2... ficheros en una sola sesión
  local marca=$1 salida; shift
  salida=$(cat "$@" | docker exec -i "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres "${variables[@]}" 2>&1) || { printf '%s\n' "$salida" | tail -5 >&2; exit 1; }
  grep -q "$marca" <<<"$salida" || { echo "FALLO: falta $marca" >&2; exit 1; }
  ! grep -q '^FALLO' <<<"$salida" || { grep '^FALLO' <<<"$salida" >&2; exit 1; }
  ok "$(grep -c '^ok$' <<<"$salida") comprobaciones hasta $marca"
}
reiniciar() {
  docker restart "$nombre" >/dev/null
  for _ in $(seq 1 120); do docker exec "$nombre" pg_isready -q -U postgres && break; sleep 0.5; done
  sleep 1
}

echo '== Restaurando roles y estructura real'
docker exec -i "$nombre" psql -X -q -U postgres -d postgres <"$globales" >/dev/null 2>&1 || true
docker exec -i "$nombre" pg_restore -U postgres -d postgres <"$volcado" >/dev/null 2>&1 || true
[[ $(escalar "SELECT to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NOT NULL") == t ]] || { echo 'Restauración incompleta' >&2; exit 2; }

ad3=$repo/deploy/postgresql/autorizacion_atestada_v3/migraciones
ct=$repo/deploy/postgresql/contratacion_temporal/migraciones
bolsa=$repo/deploy/postgresql/bolsa_llamamientos/migraciones
pruebas=$repo/deploy/postgresql/contratacion_temporal/pruebas_sql

echo '== Estado de la principal: migraciones que el volcado no trae'
for m in 000082 000083 000084 000085 000086; do run <"$(ls "$ad3"/${m}_*.up.sql)"; done; ok 'AD3-82…86'
for m in 000110 000111 000113 000115 000116 000118 000119 000120 000121; do run <"$(ls "$ct"/${m}_*.up.sql)"; done; ok 'CT110…121 (sin CT117)'
for m in 000010 000019 000021 000022 000023 000024 000025 000026 000028 000029 000030 000031 000032 000033 000034 000035 000037 000039 000040; do
  run <"$(ls "$bolsa"/${m}_*.up.sql)"
done; ok 'Bolsa 010, 019 y 021…040'

# Detección de cada migración nueva (la misma expresión que se usa para
# saber si está instalada en la principal).
declare -A detecta=(
  [ad3_87]="SELECT to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cancelacion_expediente_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL"
  [ad3_88]="SELECT to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_confirmacion_ginpix_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL"
  [bolsa_41]="SELECT to_regclass('vec_bolsa_llamamientos.politica_avisos_bolsa') IS NOT NULL"
  [ct_122]="SELECT to_regclass('vec_contratacion_temporal.cancelacion_expediente_v1') IS NOT NULL"
  [ct_123]="SELECT to_regprocedure('vec_contratacion_temporal.preparar_informe_juridico_tras_subsanacion_v1(jsonb)') IS NOT NULL"
  [ct_124]="SELECT to_regclass('vec_contratacion_temporal.confirmacion_ginpix_v1') IS NOT NULL"
  [ct_125]="SELECT to_regclass('vec_contratacion_temporal.urgencia_expediente_analisis') IS NOT NULL"
  [ct_126]="SELECT to_regclass('vec_contratacion_temporal.numeracion_parametros') IS NOT NULL"
)
orden=(ad3_87 ad3_88 bolsa_41 ct_122 ct_123 ct_124 ct_125 ct_126)
declare -A fichero=(
  [ad3_87]="$ad3/000087_consumidor_cancelacion_expediente_ct"
  [ad3_88]="$ad3/000088_consumidor_incorporacion_acreditada_ct"
  [bolsa_41]="$bolsa/000041_parametros_avisos_y_marcas"
  [ct_122]="$ct/000122_cancelacion_expediente"
  [ct_123]="$ct/000123_informe_nuevo_tras_subsanacion"
  [ct_124]="$ct/000124_incorporacion_acreditada"
  [ct_125]="$ct/000125_urgencia_expediente_analisis"
  [ct_126]="$ct/000126_parametros_numeracion_expedientes"
)
for m in "${orden[@]}"; do
  [[ $(escalar "${detecta[$m]}") == f ]] || { echo "FALLO: $m ya detectada antes de instalar" >&2; exit 1; }
done
ok 'ninguna migración nueva está presente al empezar'

echo '== Instalación en el orden de despliegue'
for m in "${orden[@]}"; do
  f=${fichero[$m]}
  sed 's/^COMMIT;$/ROLLBACK;/' "$f.up.sql" | run
  [[ $(escalar "${detecta[$m]}") == f ]] || { echo "FALLO: ROLLBACK de $m dejó rastro" >&2; exit 1; }
  run <"$f.up.sql"
  [[ $(escalar "${detecta[$m]}") == t ]] || { echo "FALLO: $m no se detecta tras UP" >&2; exit 1; }
  if run <"$f.up.sql" >/dev/null 2>&1; then echo "FALLO: doble UP de $m aceptado" >&2; exit 1; fi
  ok "$m: ROLLBACK sin rastro, UP detectado y doble UP rechazado"
done

echo '== Pruebas SQL de cada una con todas instaladas'
exp_de() { escalar "SELECT a.expediente_ref FROM vec_contratacion_temporal.expediente_integral_actual a JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version) WHERE a.version=$1 AND v.agregado_json->>'fase_actual'='$2' ORDER BY a.expediente_ref OFFSET ${3:-0} LIMIT 1"; }
exp_asignacion=$(exp_de 3 asignacion_unidad); exp_solicitud=$(exp_de 1 solicitud); exp_fiscalizado=$(exp_de 7 nombramiento)
[[ -n $exp_asignacion && -n $exp_solicitud && -n $exp_fiscalizado ]] || { echo 'Faltan expedientes sintéticos' >&2; exit 2; }
variables=()
run <"$pruebas/ct122_fixture_pg18.sql"
variables=(-v exp_asignacion="$exp_asignacion" -v exp_solicitud="$exp_solicitud" -v exp_fiscalizado="$exp_fiscalizado")
prueba 'CT122 OK' "$pruebas/ct122_cancelacion_expediente.sql"
variables=()
prueba 'CT123 OK' "$pruebas/ct123_informe_nuevo_tras_subsanacion.sql"
prueba 'fixture CT124 OK' "$pruebas/ct115_ct116_fixture_pg18.sql" "$pruebas/ct124_fixture_pg18.sql"
prueba 'CT124 OK' "$pruebas/ct124_utilidades.sql" "$pruebas/ct124_incorporacion_acreditada.sql"
prueba 'CT125 OK' "$pruebas/ct125_urgencia_expediente_analisis.sql"
prueba 'CT126 OK' "$pruebas/ct126_parametros_numeracion_expedientes.sql"
[[ $(escalar "SELECT count(*) FROM vec_bolsa_llamamientos.politica_avisos_bolsa") == 1 ]] || { echo 'FALLO: Bolsa 041 sin su versión 1' >&2; exit 1; }
ok 'Bolsa 041 con su versión 1 (conducta anterior)'

echo '== Reinicio de PostgreSQL'
reiniciar
for m in "${orden[@]}"; do
  [[ $(escalar "${detecta[$m]}") == t ]] || { echo "FALLO: $m no se detecta tras reiniciar" >&2; exit 1; }
done
ok 'las ocho siguen detectándose tras reiniciar'
prueba 'CT124 reinicio OK' "$pruebas/ct124_utilidades.sql" "$pruebas/ct124_reinicio.sql"

echo '== DOWN con historia o con dependientes instalados se niega'
for m in ct_122 ct_123 ct_124 ct_125 ct_126 ad3_87 ad3_88; do
  if run <"${fichero[$m]}.down.sql" >/dev/null 2>&1; then echo "FALLO: DOWN de $m aceptado" >&2; exit 1; fi
  [[ $(escalar "${detecta[$m]}") == t ]] || { echo "FALLO: DOWN de $m dejó rastro" >&2; exit 1; }
done
ok 'DOWN rechazados sin tocar nada'
echo 'ENSAYO CONJUNTO COMPLETO'
