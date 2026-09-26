#!/usr/bin/env bash
# Ensayo de CT126 (parámetros de numeración de expedientes) y de la
# publicación del gobierno de cobertura desde el catálogo de reglas, en
# PostgreSQL 18.4 desechable sobre la estructura real restaurada de la
# principal (volcado con datos sintéticos). CT126: ROLLBACK, UP/DOWN/UP,
# doble aplicación, formato por defecto idéntico al de hoy, ACL con los
# inicios de sesión reales, pruebas funcionales y negativas, reinicio y DOWN
# protegido por la historia. Después, dos arranques reales de la aplicación
# (prueba Go) con cambio de catálogo, reinicio de PostgreSQL y vuelta atrás.
# Uso: probar_ct126_numeracion_y_gobierno_cobertura_pg18.sh GLOBALS_SQL VOLCADO_PG_DUMP
# El contenedor usa --rm, sin red ni volúmenes anónimos; sus datos y su
# socket viven en /dev/shm/vec-pg-ct126-<pid> y se borran al terminar.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta el volcado de roles (pg_dumpall --globals-only)}
volcado=${2:?falta el volcado de la base (pg_dump -Fc)}
[[ -s $globales && -s $volcado ]] || { echo 'Faltan los volcados' >&2; exit 2; }

nombre="vec-pg-ct126-$$"
datos="/dev/shm/$nombre"
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  docker run --rm -v "/dev/shm:/limpiar" postgres:18.4 rm -rf "/limpiar/$nombre" >/dev/null 2>&1 || rm -rf "$datos" 2>/dev/null || true
}
trap limpiar EXIT
mkdir -p "$datos/datos" "$datos/socket"
chmod 1777 "$datos/socket"
docker run -d --rm --network none --name "$nombre" -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$datos/datos:/var/lib/postgresql" -v "$datos/socket:/var/run/postgresql" postgres:18.4 >/dev/null
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
como() { docker exec "$nombre" psql -X -q -At -v ON_ERROR_STOP=1 -U "$1" -d postgres -c "$2"; }
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
reiniciar() {
  docker restart "$nombre" >/dev/null
  for _ in $(seq 1 120); do docker exec "$nombre" pg_isready -q -U postgres && break; sleep 0.5; done
  sleep 1
}

echo '== Restaurando roles y estructura real'
docker exec -i "$nombre" psql -X -q -U postgres -d postgres <"$globales" >/dev/null 2>&1 || true
docker exec -i "$nombre" pg_restore -U postgres -d postgres <"$volcado" >/dev/null 2>&1 || true
[[ $(escalar "SELECT to_regprocedure('vec_contratacion_temporal.siguiente_numero_visible_v1(integer)') IS NOT NULL
  AND (SELECT ultima_secuencia >= 4 FROM vec_contratacion_temporal.gobi_o404b_checkpoint)") == t ]] \
  || { echo 'Restauración incompleta: falta CT103 o el gobierno de cobertura publicado' >&2; exit 2; }
for rol in vec_ad3_o207_gobierno vec_ct_o207_runtime; do
  [[ $(escalar "SELECT rolcanlogin FROM pg_roles WHERE rolname = '$rol'") == t ]] || { echo "falta el inicio de sesión $rol" >&2; exit 2; }
done

ct=$repo/deploy/postgresql/contratacion_temporal/migraciones
pruebas=$repo/deploy/postgresql/contratacion_temporal/pruebas_sql
m=$ct/000126_parametros_numeracion_expedientes
cuerpo="SELECT md5(prosrc) FROM pg_proc WHERE oid = 'vec_contratacion_temporal.siguiente_numero_visible_v1(integer)'::regprocedure"
objetos="SELECT (to_regclass('vec_contratacion_temporal.numeracion_parametros') IS NOT NULL)::int
  + (to_regprocedure('vec_contratacion_temporal.publicar_numeracion_parametros_v1(text,integer,text)') IS NOT NULL)::int"
numero_en_rollback="BEGIN; SELECT vec_contratacion_temporal.siguiente_numero_visible_v1(2030); ROLLBACK;"
original=$(escalar "$cuerpo")
formato_hoy=$(como vec_ct_o207_runtime "$numero_en_rollback")
[[ $formato_hoy =~ ^2030/CT-[0-9]{6}$ ]] || { echo "FALLO: formato previo inesperado $formato_hoy" >&2; exit 1; }

echo '== CT126: ROLLBACK, UP, doble UP, DOWN, doble DOWN, UP'
sed 's/^COMMIT;$/ROLLBACK;/' "$m.up.sql" | run
[[ $(escalar "$objetos") == 0 && $(escalar "$cuerpo") == "$original" ]] || { echo 'FALLO: ROLLBACK dejó rastro' >&2; exit 1; }; ok 'ROLLBACK sin rastro'
run <"$m.up.sql"; [[ $(escalar "$objetos") == 2 ]] || { echo 'FALLO: UP incompleto' >&2; exit 1; }; ok 'UP'
[[ $(como vec_ct_o207_runtime "$numero_en_rollback") == "$formato_hoy" ]] || { echo 'FALLO: el formato por defecto cambió' >&2; exit 1; }
ok "formato por defecto idéntico al de hoy ($formato_hoy)"
falla_con "$m.up.sql" 'CT-000126 ya instalada'; ok 'doble UP rechazado'
run <"$m.down.sql"
[[ $(escalar "$objetos") == 0 && $(escalar "$cuerpo") == "$original" ]] || { echo 'FALLO: DOWN incompleto' >&2; exit 1; }; ok 'DOWN sin historia devuelve la función de CT103'
falla_con "$m.down.sql" 'CT-000126 no instalada'; ok 'doble DOWN rechazado'
run <"$m.up.sql"; ok 'UP de nuevo'
[[ $(escalar "SELECT has_function_privilege('vec_contratacion_temporal_gobernador','vec_contratacion_temporal.publicar_numeracion_parametros_v1(text,integer,text)','EXECUTE')
  AND NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.publicar_numeracion_parametros_v1(text,integer,text)','EXECUTE')
  AND NOT has_function_privilege('public','vec_contratacion_temporal.publicar_numeracion_parametros_v1(text,integer,text)','EXECUTE')
  AND has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.siguiente_numero_visible_v1(integer)','EXECUTE')
  AND NOT has_table_privilege('vec_contratacion_temporal_gobernador','vec_contratacion_temporal.numeracion_parametros','SELECT')
  AND NOT has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.numeracion_parametros','SELECT')") == t ]] \
  || { echo 'FALLO: ACL de CT126' >&2; exit 1; }
ok 'ACL con roles reales'

echo '== Numeración: publicación, formato, negativos e historia'
prueba 'CT126 OK' "$pruebas/ct126_parametros_numeracion_expedientes.sql"
for prefijo in 'CT-1' '9' 'CTEMP2026'; do
  if salida=$(como vec_ad3_o207_gobierno "SELECT resultado FROM vec_contratacion_temporal.publicar_numeracion_parametros_v1('$prefijo', 1, 'configuracion:ct:numeracion:prueba')" 2>&1); then
    echo "FALLO: se publicó el prefijo terminado en cifra $prefijo" >&2; exit 1
  fi
  grep -q 'parámetros de numeración no válidos' <<<"$salida" || { echo "FALLO: rechazo inesperado de $prefijo: $salida" >&2; exit 1; }
  if salida=$(escalar "BEGIN; INSERT INTO vec_contratacion_temporal.numeracion_parametros(version,prefijo,digitos,huella_sha256,fuente_ref,publicado_por,publicado_en)
      VALUES (99,'$prefijo',1,encode(sha256(convert_to('$prefijo|1','UTF8')),'hex'),'configuracion:ct:numeracion:prueba','postgres',date_trunc('microseconds',clock_timestamp())); ROLLBACK;" 2>&1); then
    echo "FALLO: la tabla admitió el prefijo $prefijo" >&2; exit 1
  fi
  grep -q 'numeracion_parametros_prefijo_check' <<<"$salida" || { echo "FALLO: rechazo inesperado en la tabla de $prefijo: $salida" >&2; exit 1; }
done
ok 'prefijo terminado en cifra rechazado al publicar y en la tabla (evita números visibles repetidos)'

echo '== Reinicio de PostgreSQL'
reiniciar
[[ $(escalar "SELECT count(*)||'/'||max(version)||'/'||(SELECT prefijo FROM vec_contratacion_temporal.numeracion_parametros ORDER BY version DESC LIMIT 1) FROM vec_contratacion_temporal.numeracion_parametros") == '3/3/CT-' ]] \
  || { echo 'FALLO: historia tras reinicio' >&2; exit 1; }
[[ $(como vec_ct_o207_runtime "SELECT vec_contratacion_temporal.siguiente_numero_visible_v1(2032)") == '2032/CT-000003' ]] || { echo 'FALLO: numeración tras reinicio' >&2; exit 1; }
ok 'historia y formato conservados tras reinicio'
falla_con "$m.down.sql" 'reversión denegada'; ok 'DOWN con historia rechazado'

echo '== Arranques de la aplicación: gobierno de cobertura y numeración desde el catálogo'
socket=$datos/socket
export VEC_CT126_PG_DSN_GOBIERNO="host=$socket user=vec_ad3_o207_gobierno dbname=postgres sslmode=disable"
export VEC_CT126_PG_DSN_EJECUCION="host=$socket user=vec_ct_o207_runtime dbname=postgres sslmode=disable"
export VEC_CT126_PG_DSN_ADMIN="host=$socket user=postgres dbname=postgres sslmode=disable"
probar_go() {
  local salida
  salida=$(cd "$repo" && VEC_CT126_PG_FASE=$1 go test -count=1 -run '^TestArranqueCatalogoCoberturaYNumeracionPostgreSQL$' -v ./internal/app/bootstrap/ 2>&1) || true
  printf '%s\n' "$salida" | grep -E '^(---|    )' || true
  grep -q '^--- PASS: TestArranqueCatalogoCoberturaYNumeracionPostgreSQL' <<<"$salida" \
    || { printf '%s\n' "$salida" | tail -20 >&2; echo "FALLO: arranques Go, fase $1" >&2; exit 1; }
}
probar_go 1; ok 'dos arranques sin catálogo sin historia; cambio de catálogo publica una versión y no se repite'
reiniciar
probar_go 2; ok 'tras reiniciar PostgreSQL no se republica; volver a las vías de siempre publica y no se repite'
echo 'CT126 y gobierno de cobertura desde el catálogo verificados'
