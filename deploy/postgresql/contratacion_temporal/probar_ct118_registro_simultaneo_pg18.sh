#!/usr/bin/env bash
# Ensayo de CT118 con dos registros de firma simultáneos en PostgreSQL 18.4
# desechable sobre la estructura real restaurada (volcado sintético). Instala
# AD3-85 real y CT118 (UP, DOWN sin historia, UP) y sustituye la fachada
# AD3-85 por un doble (ct118_simultaneo_fixture.sql) tras pasar las pruebas
# negativas y el recorrido existentes de CT118, que se revierten. En SERIALIZABLE el
# cerrojo del documento no protege: la transacción que pierde la carrera
# debe terminar con 40001 o con 23505 de la restricción nombrada (nunca con
# P1185 ni con una segunda fila), para que el adaptador la repita:
#   1. misma clave: el reintento recupera el mismo recibo (YaRegistrada);
#   2. otra clave con la misma secuencia: el reintento da P1183 (conflicto).
# Uso: probar_ct118_registro_simultaneo_pg18.sh GLOBALS_SQL VOLCADO_PG_DUMP
# El contenedor usa --rm, sin red ni volúmenes anónimos; sus datos viven en
# /dev/shm/vec-pg-ct118s-<pid> y se borran al terminar.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta el volcado de roles (pg_dumpall --globals-only)}
volcado=${2:?falta el volcado de la base (pg_dump -Fc)}
[[ -s $globales && -s $volcado ]] || { echo 'Faltan los volcados' >&2; exit 2; }

nombre="vec-pg-ct118s-$$"
datos="/dev/shm/$nombre"
salida_a="$datos.a"
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  docker run --rm -v "/dev/shm:/limpiar" postgres:18.4 rm -rf "/limpiar/$nombre" >/dev/null 2>&1 || rm -rf "$datos" 2>/dev/null || true
  rm -f "$salida_a"
}
trap limpiar EXIT
mkdir -p "$datos"
docker run -d --rm --network none --name "$nombre" -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$datos:/var/lib/postgresql" postgres:18.4 >/dev/null
for _ in $(seq 1 240); do
  if docker logs "$nombre" 2>&1 | grep -q 'PostgreSQL init process complete' && docker exec "$nombre" pg_isready -q -U postgres; then break; fi
  sleep 0.5
done
if [[ -n $(docker inspect --format '{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{end}}{{end}}' "$nombre") ]]; then
  echo 'volumen anónimo inesperado' >&2; exit 65
fi
run() { docker exec -i "$nombre" psql -X -q -o /dev/null -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
escalar() { docker exec "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
ejecutor() { docker exec -i "$nombre" psql -X -At -v ON_ERROR_STOP=1 -v VERBOSITY=verbose -U vec_ct118s_runtime -d postgres; }
[[ $(escalar 'SHOW server_version') == 18.4* ]] || { echo 'PostgreSQL 18.4 no disponible' >&2; exit 2; }
ok() { printf 'OK %s\n' "$1"; }
falla() { echo "FALLO: $1" >&2; exit 1; }

echo '== Restaurando roles y estructura real'
docker exec -i "$nombre" psql -X -q -U postgres -d postgres <"$globales" >/dev/null 2>&1 || true
docker exec -i "$nombre" pg_restore -U postgres -d postgres <"$volcado" >/dev/null 2>&1 || true
[[ $(escalar "SELECT to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NOT NULL") == t ]] || { echo 'Restauración incompleta' >&2; exit 2; }

base=$repo/deploy/postgresql
m=$base/contratacion_temporal/migraciones/000118_registro_firmas_documento
echo '== AD3-85 y CT118: UP, DOWN sin historia, UP'
run <"$base/autorizacion_atestada_v3/migraciones/000085_consumidor_firma_documento_ct.up.sql"
run <"$m.up.sql"; run <"$m.down.sql"; run <"$m.up.sql"
[[ $(escalar "SELECT count(*) FROM pg_constraint WHERE conrelid='vec_contratacion_temporal.firma_documento_v1'::regclass AND conname IN ('firma_documento_v1_clave_unica','firma_documento_v1_secuencia_unica')") == 2 ]] \
  || falla 'restricciones únicas con nombre fijo'
ok 'CT118 instalada con las restricciones de clave y secuencia nombradas'
echo '== Pruebas de CT118 existentes: negativas, historia y recorrido (se revierten)'
run <"$base/contratacion_temporal/pruebas_sql/ct118_registro_firmas_documento.sql"
run <"$base/contratacion_temporal/pruebas_sql/ct118_recorrido_con_fachada_de_prueba.sql"
ok 'pruebas negativas y recorrido de CT118'
run <"$base/contratacion_temporal/pruebas_sql/ct118_simultaneo_fixture.sql"

# Dos transacciones SERIALIZABLE: A registra y espera antes de confirmar; B
# empieza mientras A sigue abierta. Devuelve la salida de B.
carrera() {
  local clave_a=$1 clave_b=$2 paso=$3 secuencia=$4
  printf "BEGIN ISOLATION LEVEL SERIALIZABLE;\nSELECT prueba_ct118s.registrar('%s',%s,%s)->>'FirmaRef';\nSELECT pg_sleep(2);\nCOMMIT;\n" \
    "$clave_a" "$paso" "$secuencia" | ejecutor >"$salida_a" 2>&1 &
  local pid=$!
  sleep 0.7
  printf "BEGIN ISOLATION LEVEL SERIALIZABLE;\nSELECT prueba_ct118s.registrar('%s',%s,%s)->>'FirmaRef';\nCOMMIT;\n" \
    "$clave_b" "$paso" "$secuencia" | ejecutor 2>&1 || true
  wait "$pid" || falla "la transacción A no confirmó: $(cat "$salida_a")"
}
perdedora_reintentable() {
  local salida=$1 restriccion=$2
  grep -q 'P1185' <<<"$salida" && falla "la carrera se convirtió en P1185: $salida"
  if grep -q '40001' <<<"$salida"; then return 0; fi
  grep -q '23505' <<<"$salida" && grep -q "$restriccion" <<<"$salida" && return 0
  falla "la transacción B no terminó con 40001 ni con 23505 de $restriccion: $salida"
}
reintento() {
  printf "BEGIN ISOLATION LEVEL SERIALIZABLE;\nSELECT prueba_ct118s.registrar('%s',%s,%s);\nCOMMIT;\n" "$1" "$2" "$3" | ejecutor 2>&1 || true
}

echo '== 1. Misma clave a la vez'
salida_b=$(carrera clave-simultanea-000001 clave-simultanea-000001 1 1)
perdedora_reintentable "$salida_b" firma_documento_v1_clave_unica; ok "B pierde con error reintentable: $(grep -o -m1 '40001\|23505' <<<"$salida_b")"
firma_a=$(grep -m1 '^firma-ct:' "$salida_a") || falla "A sin recibo: $(cat "$salida_a")"
salida_r=$(reintento clave-simultanea-000001 1 1)
grep -q '"YaRegistrada": true' <<<"$salida_r" && grep -q "\"FirmaRef\": \"$firma_a\"" <<<"$salida_r" \
  || falla "el reintento no recupera el recibo de A: $salida_r"
ok 'el reintento recupera el mismo recibo'

echo '== 2. Otra clave con la misma secuencia a la vez'
salida_b=$(carrera clave-simultanea-000002 clave-simultanea-000003 2 2)
perdedora_reintentable "$salida_b" firma_documento_v1_secuencia_unica; ok "B pierde con error reintentable: $(grep -o -m1 '40001\|23505' <<<"$salida_b")"
salida_r=$(reintento clave-simultanea-000003 2 2)
grep -q 'P1183' <<<"$salida_r" || falla "el reintento no informa del conflicto de secuencia: $salida_r"
ok 'el reintento informa del conflicto de secuencia (P1183)'

[[ $(escalar "SELECT (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_v1)||'/'||(SELECT count(*) FROM vec_contratacion_temporal.firma_documento_auditoria_v1)||'/'||(SELECT count(*) FROM vec_contratacion_temporal.firma_documento_outbox_v1)") == 2/2/2 ]] \
  || falla 'filas de firma, auditoría y outbox'
ok 'dos firmas, dos auditorías y dos eventos: ninguna fila de las transacciones perdidas'
echo 'CT118 registro simultáneo verificado'
