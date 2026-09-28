#!/usr/bin/env bash
# Ensayo aislado de AD3-94/CT131 y AD3-96/CT133 sobre una copia sintética.
# Uso: scripts/probar_rrhh_plantillas_recorrido_pg18.sh GLOBALS_SQL VOLCADO_FC
# El volcado debe contener la preimagen de CT131 y AD3-94. Nunca se conecta a
# servicios conservados: PostgreSQL 18.4 vive en /dev/shm, sin red, y se borra.
# El recorrido HTTP con V3 real se verifica aparte cuando la composición esté
# disponible; este ensayo no presenta una llamada SQL directa como V3 real.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta GLOBALS_SQL de una base sintética}
volcado=${2:?falta VOLCADO_FC de una base sintética}
[[ $# == 2 && -s $globales && -s $volcado ]] || { echo 'Uso: script GLOBALS_SQL VOLCADO_FC (sintéticos)' >&2; exit 2; }
for programa in docker go python3; do command -v "$programa" >/dev/null || { echo "Falta $programa" >&2; exit 2; }; done
docker info >/dev/null 2>&1 || { echo 'Docker no disponible para el usuario actual' >&2; exit 2; }

temporal=$(mktemp -d /dev/shm/vec-plantillas-pg18.XXXXXXXX)
nombre="vec-plantillas-pg18-$$"
socket="$temporal/socket"
datos="$temporal/data"
mkdir -p "$socket" "$datos" "$temporal/go-cache" "$temporal/go-tmp"
chmod 1777 "$socket"
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  # PostgreSQL escribe como su UID del contenedor; retirarlo desde Docker
  # evita dejar ficheros ajenos al usuario bajo /dev/shm tras el ensayo.
  docker run --rm --network none -v "$temporal:/limpiar" --entrypoint /bin/sh \
    postgres:18.4 -c 'rm -rf /limpiar/data /limpiar/socket' >/dev/null 2>&1 || true
  rm -rf -- "$temporal"
}
trap limpiar EXIT

docker run -d --rm --network none --name "$nombre" \
  -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$datos:/var/lib/postgresql" -v "$socket:/var/run/postgresql" \
  postgres:18.4 >/dev/null
for _ in $(seq 1 240); do
  docker exec "$nombre" pg_isready -q -U postgres -d postgres && break
  sleep 0.5
done
docker exec "$nombre" pg_isready -q -U postgres -d postgres || { echo 'PostgreSQL no arrancó' >&2; exit 1; }
[[ -z $(docker inspect --format '{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{end}}{{end}}' "$nombre") ]] || {
  echo 'Volumen anónimo inesperado' >&2; exit 1;
}
[[ $(docker exec "$nombre" psql -X -At -U postgres -d postgres -c 'SHOW server_version') == 18.4* ]] || {
  echo 'Se requiere PostgreSQL 18.4' >&2; exit 1;
}
psql_admin() { docker exec -i "$nombre" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres; }
escalar() { docker exec "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
echo 'Restaurando copia sintética en PostgreSQL desechable'
psql_admin <"$globales" >"$temporal/globals.log" 2>&1 || true
docker exec -i "$nombre" pg_restore -U postgres -d postgres <"$volcado" >"$temporal/restore.log" 2>&1 || true
[[ $(escalar "SELECT to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NOT NULL AND to_regprocedure('vec_contratacion_temporal.rechazar_mutacion_historia_v1()') IS NOT NULL AND to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL") == t ]] || {
  echo 'Volcado sin la preimagen CT/AD3 necesaria' >&2; exit 1;
}

ad3="$repo/deploy/postgresql/autorizacion_atestada_v3/migraciones"
ct="$repo/deploy/postgresql/contratacion_temporal/migraciones"
migraciones=(
  "$ad3/000094_consumidor_catalogo_plantillas_ct.up.sql"
  "$ct/000131_catalogo_plantillas_documentos.up.sql"
  "$ad3/000096_consumidor_catalogo_plantillas_documental_ct.up.sql"
  "$ct/000133_obtener_catalogo_plantillas_publicado_documental.up.sql"
)
sondas=(
  "to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')"
  "to_regclass('vec_contratacion_temporal.catalogo_plantillas_historia_v1')"
  "to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')"
  "to_regprocedure('vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')"
)
for i in "${!migraciones[@]}"; do
  [[ $(escalar "SELECT ${sondas[$i]} IS NULL") == t ]] || { echo "La migración ya está instalada: ${migraciones[$i]}" >&2; exit 1; }
done
for i in "${!migraciones[@]}"; do
  migracion=${migraciones[$i]}
  sed 's/^COMMIT;$/ROLLBACK;/' "$migracion" | psql_admin >/dev/null
  [[ $(escalar "SELECT ${sondas[$i]} IS NULL") == t ]] || { echo "ROLLBACK dejó rastro: $migracion" >&2; exit 1; }
  psql_admin <"$migracion" >/dev/null
  [[ $(escalar "SELECT ${sondas[$i]} IS NOT NULL") == t ]] || { echo "UP no se detecta: $migracion" >&2; exit 1; }
  if psql_admin <"$migracion" >/dev/null 2>&1; then echo "Doble UP aceptado: $migracion" >&2; exit 1; fi
  echo "OK $(basename "$migracion"): ROLLBACK, UP y doble UP"
done

# Usuarios de ensayo separados. Trust sólo existe dentro del contenedor sin red.
psql_admin >/dev/null <<'SQL'
DO $roles$
BEGIN
 IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN ('vec_plantillas_migrador_ensayo','vec_plantillas_ejecutor_ensayo')) THEN
  RAISE EXCEPTION 'identidades de ensayo ya presentes en el volcado';
 END IF;
END $roles$;
CREATE ROLE vec_plantillas_migrador_ensayo LOGIN INHERIT NOSUPERUSER NOBYPASSRLS;
CREATE ROLE vec_plantillas_ejecutor_ensayo LOGIN INHERIT NOSUPERUSER NOBYPASSRLS;
GRANT vec_contratacion_temporal_migrador TO vec_plantillas_migrador_ensayo WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_contratacion_temporal_ejecutor TO vec_plantillas_ejecutor_ensayo WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT CONNECT ON DATABASE postgres TO vec_plantillas_migrador_ensayo, vec_plantillas_ejecutor_ensayo;
SQL
export GOCACHE="$temporal/go-cache" GOTMPDIR="$temporal/go-tmp" GOTOOLCHAIN=auto GOPROXY=off GOFLAGS=-mod=readonly
export VEC_CT_PLANTILLAS_MIGRADOR_DATABASE_URL="host=$socket user=vec_plantillas_migrador_ensayo dbname=postgres sslmode=disable"
catalogo="$repo/data/demo/plantillas/ct_plantillas_documentos.ejemplo.demo.json"
aprobar='sin_aprobacion_rrhh:ejemplo_desarrollo'
echo 'Provisionando el catálogo de ejemplo por el CLI migrador'
(cd "$repo" && go run ./cmd/vec-provision-plantillas -catalogo "$catalogo" -aprobacion-ref "$aprobar") >"$temporal/recibo_1.json"
(cd "$repo" && go run ./cmd/vec-provision-plantillas -catalogo "$catalogo" -aprobacion-ref "$aprobar") >"$temporal/recibo_2.json"
python3 - "$temporal/recibo_1.json" "$temporal/recibo_2.json" <<'PY'
import json, sys
a, b = [json.load(open(p, encoding='utf-8')) for p in sys.argv[1:]]
assert a['resultado'] == 'registrado' and b['resultado'] == 'replay', (a, b)
assert a['recibo_ref'] == b['recibo_ref'] and a['registrada_en'] == b['registrada_en']
assert a['version'] == b['version'] == 1 and a['revision'] == b['revision'] == 1
assert a['catalogo_huella_sha256'] == b['catalogo_huella_sha256']
assert a['contenido_json_sha256'] == b['contenido_json_sha256']
print('OK provisión CLI y replay: mismo recibo, fecha, versión y huellas')
PY
[[ $(escalar "SELECT count(*) FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1") == 1 ]] || { echo 'Historia de provisión duplicada' >&2; exit 1; }
[[ $(escalar "SELECT count(*) FROM vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1") == 1 ]] || { echo 'Auditoría de provisión duplicada' >&2; exit 1; }
[[ $(escalar "SELECT count(*) FROM vec_contratacion_temporal.catalogo_plantillas_outbox_v1") == 0 ]] || { echo 'La provisión creó outbox de edición' >&2; exit 1; }

# La identidad migradora sólo provisiona; la ejecutora no obtiene tablas ni
# admite una llamada documental sin una decisión V3 emitida por el PDP.
[[ $(escalar "SELECT NOT has_table_privilege('vec_plantillas_ejecutor_ensayo','vec_contratacion_temporal.catalogo_plantillas_historia_v1','SELECT') AND NOT has_function_privilege('vec_plantillas_migrador_ensayo','vec_contratacion_temporal.operar_catalogo_plantillas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') AND NOT has_function_privilege('vec_plantillas_migrador_ensayo','vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')") == t ]] || {
  echo 'ACL de ejecución/provisión incompatible' >&2; exit 1;
}
echo 'OK ejecutor sin SELECT directo y migrador sin funciones V3'
for prueba in gobierno documental; do
  if [[ $prueba == gobierno ]]; then
    funcion=operar_catalogo_plantillas_v1
    material='{"operacion":"consultar"}'
  else
    funcion=obtener_catalogo_plantillas_publicado_documental_v1
    material='{"operacion":"listar","expediente_ref":"expediente:ct:ensayo","version_observada":1,"consulta_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}'
  fi
  argumentos="'\\x'::bytea,'\\x'::bytea,'\\x'::bytea,'\\x'::bytea,1,1,'\\x'::bytea,'\\x'::bytea,'\\x'::bytea,'\\x'::bytea"
  if salida=$(docker exec "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U vec_plantillas_ejecutor_ensayo -d postgres \
    -c "BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE; SELECT vec_contratacion_temporal.$funcion('$material'::jsonb,$argumentos); COMMIT;" 2>&1); then
    echo "La lectura directa sin decisión V3 fue aceptada: $funcion" >&2; exit 1
  fi
  grep -q "CT-131: decisión inválida\|CT-133: decisión inválida" <<<"$salida" || {
    echo "Rechazo inesperado de $funcion: $salida" >&2; exit 1;
  }
done
[[ $(escalar "SELECT count(*) FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1") == 1 ]] || { echo 'La llamada sin V3 alteró historia' >&2; exit 1; }
echo 'OK llamadas directas sin material V3 rechazadas sin historia nueva'

docker restart "$nombre" >/dev/null
for _ in $(seq 1 120); do docker exec "$nombre" pg_isready -q -U postgres -d postgres && break; sleep 0.5; done
docker exec "$nombre" pg_isready -q -U postgres -d postgres || { echo 'PostgreSQL no volvió del reinicio' >&2; exit 1; }
for i in "${!sondas[@]}"; do
  [[ $(escalar "SELECT ${sondas[$i]} IS NOT NULL") == t ]] || { echo "Migración perdida tras reinicio: ${migraciones[$i]}" >&2; exit 1; }
done
(cd "$repo" && go run ./cmd/vec-provision-plantillas -catalogo "$catalogo" -aprobacion-ref "$aprobar") >"$temporal/recibo_3.json"
python3 - "$temporal/recibo_1.json" "$temporal/recibo_3.json" <<'PY'
import json, sys
a, b = [json.load(open(p, encoding='utf-8')) for p in sys.argv[1:]]
assert b['resultado'] == 'replay'
for k in ('recibo_ref', 'registrada_en', 'version', 'revision', 'catalogo_huella_sha256', 'contenido_json_sha256'):
    assert a[k] == b[k], k
print('OK recuperación tras reinicio: misma provisión y sin duplicados')
PY
[[ $(escalar "SELECT count(*) FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1") == 1 ]] || { echo 'Historia duplicada tras reinicio' >&2; exit 1; }
echo 'ENSAYO PG18 DE PROVISIÓN Y ESTRUCTURA COMPLETO; edición, publicación y descarga HTTP con V3 real pendientes de composición y recorrido autenticado.'
