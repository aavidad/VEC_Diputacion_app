#!/usr/bin/env bash
# Ensayo de CT128 (propuesta de nombramiento del sucesor tras una no
# incorporación) en PostgreSQL 18.4 desechable sobre la estructura real
# restaurada de la principal (volcado con datos sintéticos). Lleva el volcado
# al estado de la integración de los huecos de RRHH (AD3-82…88, CT110…126 y
# Bolsa hasta 000042), comprueba en CT128 ROLLBACK sin rastro, UP, detección,
# doble UP rechazado, DOWN exacto (tres funciones, su ACL y la unicidad
# original) y UP otra vez; recorre con dobles explícitos de las fachadas AD3
# la cadena aceptación de A → no incorporación de A → baja en Bolsa →
# siguiente llamamiento → aviso, aceptación y resolución de B → propuesta de B
# (A queda sustituida) → incorporación de B → publicación a Bolsa → GINPIX →
# cese → cierre, con los negativos (segunda propuesta sin no incorporación,
# otra clave, versión anterior); reinicia PostgreSQL, repite los mismos
# materiales (mismos recibos, ninguna fila nueva) y comprueba que el DOWN se
# niega con historia.
# Uso: probar_ct128_propuesta_sucesor_pg18.sh GLOBALS_SQL VOLCADO_PG_DUMP
# El contenedor usa --rm, sin red ni volúmenes anónimos; sus datos viven en
# /dev/shm/vec-pg-ct128-<pid> y se borran al terminar.
# Con VEC_CT128_GO=1 publica el puerto solo en 127.0.0.1 y, tras la cadena
# SQL, ejecuta la prueba de contrato Go↔SQL de la consulta de propuestas.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta el volcado de roles (pg_dumpall --globals-only)}
volcado=${2:?falta el volcado de la base (pg_dump -Fc)}
[[ -s $globales && -s $volcado ]] || { echo 'Faltan los volcados' >&2; exit 2; }

nombre="vec-pg-ct128-$$"
datos="/dev/shm/$nombre"
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  docker run --rm -v "/dev/shm:/limpiar" postgres:18.4 rm -rf "/limpiar/$nombre" >/dev/null 2>&1 || rm -rf "$datos" 2>/dev/null || true
}
trap limpiar EXIT
mkdir -p "$datos"
red=(--network none)
[[ ${VEC_CT128_GO:-} == 1 ]] && red=(-p 127.0.0.1::5432)
docker run -d --rm "${red[@]}" --name "$nombre" -e POSTGRES_HOST_AUTH_METHOD=trust \
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
[[ $(escalar 'SHOW server_version') == 18.4* ]] || { echo 'PostgreSQL 18.4 no disponible' >&2; exit 2; }
ok() { printf 'OK %s\n' "$1"; }
igual() { [[ $1 == "$2" ]] || { echo "FALLO $3" >&2; exit 1; }; ok "$3"; }
falla_con() { # $1 fichero, $2 texto esperado en el error
  if salida=$(run <"$1" 2>&1); then echo "FALLO: se esperaba rechazo de $(basename "$1")" >&2; exit 1; fi
  grep -q "$2" <<<"$salida" || { echo "FALLO: rechazo inesperado de $(basename "$1"): $salida" >&2; exit 1; }
}
prueba() { # $1 marca final, $2... ficheros en una sola sesión
  local marca=$1 salida; shift
  salida=$(cat "$@" | docker exec -i "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres 2>&1) || { printf '%s\n' "$salida" | tail -8 >&2; exit 1; }
  grep -q "$marca" <<<"$salida" || { echo "FALLO: falta $marca" >&2; exit 1; }
  ok "$(grep -c '^ok$' <<<"$salida") comprobaciones hasta $marca"
}

echo '== Restaurando roles y estructura real'
docker exec -i "$nombre" psql -X -q -U postgres -d postgres <"$globales" >/dev/null 2>&1 || true
docker exec -i "$nombre" pg_restore -U postgres -d postgres <"$volcado" >/dev/null 2>&1 || true
[[ $(escalar "SELECT to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NOT NULL") == t ]] || { echo 'Restauración incompleta' >&2; exit 2; }

ad3=$repo/deploy/postgresql/autorizacion_atestada_v3/migraciones
ct=$repo/deploy/postgresql/contratacion_temporal/migraciones
bolsa=$repo/deploy/postgresql/bolsa_llamamientos/migraciones
pruebas=$repo/deploy/postgresql/contratacion_temporal/pruebas_sql
echo '== Estado de la integración: AD3-82…88, CT110…126 (sin CT117) y Bolsa hasta 000042'
for m in 000082 000083 000084 000085 000086 000087 000088; do run <"$(ls "$ad3"/${m}_*.up.sql)"; done
for m in 000110 000111 000113 000115 000116 000118 000119 000120 000121; do run <"$(ls "$ct"/${m}_*.up.sql)"; done
for m in 000010 000019 000021 000022 000023 000024 000025 000026 000028 000029 000030 000031 000032 000033 000034 000035 000037 000039 000040 000041 000042; do
  run <"$(ls "$bolsa"/${m}_*.up.sql)"
done
for m in 000122 000123 000124 000125 000126; do run <"$(ls "$ct"/${m}_*.up.sql)"; done
ok 'cadena previa instalada'

detecta="SELECT to_regclass('vec_contratacion_temporal.propuesta_sustitucion_v1') IS NOT NULL"
estado="SELECT md5(string_agg(pg_get_functiondef(p.oid)||coalesce(p.proacl::text,'')||coalesce(p.proconfig::text,''),'' ORDER BY p.oid::regprocedure::text))
  ||(SELECT md5(string_agg(conname||pg_get_constraintdef(oid),'' ORDER BY conname)) FROM pg_constraint
      WHERE conrelid='vec_contratacion_temporal.propuesta_formalizacion'::regclass)
  ||(SELECT count(*) FROM pg_trigger WHERE tgrelid='vec_contratacion_temporal.propuesta_formalizacion'::regclass)
  FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_contratacion_temporal'
   AND p.proname IN ('registrar_propuesta_formalizacion_v1','registrar_propuesta_formalizacion_v2','preparar_no_incorporacion_v1','leer_contratos_bolsa_v1')"
inicial=$(escalar "$estado")
up="$ct/000128_propuesta_sucesor_no_incorporacion.up.sql"
down="$ct/000128_propuesta_sucesor_no_incorporacion.down.sql"

echo '== CT128: ROLLBACK, UP, doble UP, DOWN exacto, UP'
igual "$(escalar "$detecta")" f 'CT128 no está instalada al empezar'
sed 's/^COMMIT;$/ROLLBACK;/' "$up" | run
igual "$(escalar "$estado")|$(escalar "$detecta")" "$inicial|f" 'ROLLBACK CT128 sin rastro'
run <"$up"; ok 'UP CT128'
igual "$(escalar "$detecta")" t 'CT128 detectada tras UP'
con_ct128=$(escalar "$estado")
[[ $con_ct128 != "$inicial" ]] || { echo 'FALLO: CT128 no amplió las funciones' >&2; exit 1; }
falla_con "$up" 'CT128 ya instalada'; ok 'doble UP CT128 rechazado'
run <"$down"
igual "$(escalar "$estado")|$(escalar "$detecta")" "$inicial|f" 'DOWN CT128 restaura exactamente funciones, ACL y unicidad'
run <"$up"
igual "$(escalar "$estado")" "$con_ct128" 'UP tras DOWN reproduce CT128'
igual "$(escalar "SELECT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.consultar_propuestas_expediente_v1(text,text)','EXECUTE')
  AND NOT has_function_privilege('public','vec_contratacion_temporal.consultar_propuestas_expediente_v1(text,text)','EXECUTE')
  AND NOT has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.propuesta_sustitucion_v1','SELECT')
  AND NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='propuesta_formalizacion_organizacion_ref_expediente_ref_key')")" t \
  'consulta solo para el ejecutor, tabla cerrada y unicidad sustituida por la de la propuesta vigente'

echo '== Cadena (dobles explícitos de las fachadas AD3)'
b='expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7'
salida=$(docker exec -i "$nombre" psql -X -q -At -v ON_ERROR_STOP=1 -v exp_a="$b" -U postgres -d postgres <"$pruebas/ct115_ct116_fixture_pg18.sql" 2>&1) || { echo "$salida" >&2; exit 1; }
prueba 'fixture no incorporación OK' "$pruebas/ct124_no_incorporacion_fixture.sql"
prueba 'cadena no incorporación OK' "$pruebas/ct124_utilidades.sql" "$pruebas/ct124_no_incorporacion_cadena.sql"
prueba 'fixture CT128 OK' "$pruebas/ct128_propuesta_sucesor_fixture.sql"
prueba 'cadena CT128 OK' "$pruebas/ct124_utilidades.sql" "$pruebas/ct128_propuesta_sucesor_cadena.sql"

if [[ ${VEC_CT128_GO:-} == 1 ]]; then
  echo '== Contrato Go↔SQL de la consulta de propuestas'
  puerto=$(docker port "$nombre" 5432/tcp | head -1 | sed 's/.*://')
  a='expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001'
  org=$(escalar "SELECT agregado_json->>'organizacion_ref' FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref='$a' AND version=7")
  (cd "$repo" && VEC_CT128_PG_DSN="postgres://vec_ct115_runtime@127.0.0.1:$puerto/postgres?sslmode=disable" VEC_CT128_ORG="$org" \
    VEC_CT128_EXP="$a" TMPDIR=${TMPDIR:-/dev/shm} go test -count=1 -run TestPropuestasExpedientePostgreSQLContratoGoSQL -v \
    ./internal/modules/contrataciontemporal/adapters/postgres/ 2>&1 | tail -8)
fi

echo '== Reinicio de PostgreSQL y repetición de los mismos materiales'
docker restart "$nombre" >/dev/null
for _ in $(seq 1 120); do docker exec "$nombre" pg_isready -q -U postgres && break; sleep 0.5; done
sleep 1
igual "$(escalar "$detecta")" t 'CT128 detectada tras reiniciar'
prueba 'reinicio CT128 OK' "$pruebas/ct124_utilidades.sql" "$pruebas/ct128_propuesta_sucesor_reinicio.sql"
falla_con "$down" 'no admitido con historia'; ok 'DOWN CT128 rechazado con historia'
igual "$(escalar "$detecta")" t 'el DOWN rechazado no deja rastro'
echo 'Propuesta del sucesor tras la no incorporación verificada'
