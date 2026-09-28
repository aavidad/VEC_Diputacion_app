#!/usr/bin/env bash
# Proyección B56 en PostgreSQL 18.4 desechable. No instala SQL sobre bases conservadas.
set -Eeuo pipefail
directorio=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo=$(git -C "$directorio" rev-parse --show-toplevel)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
command -v docker >/dev/null
docker info >/dev/null
temporal=$(mktemp -d /tmp/vec-b56.XXXXXXXX)
chmod 1777 "$temporal"
nombre=vec-b56-${temporal##*.}
contenedor=
limpiar() {
  if [[ -n $contenedor ]]; then docker rm -f "$contenedor" >/dev/null 2>&1 || true; fi
  if [[ -d $temporal ]]; then
    docker run --rm --network none --pull never -v "$temporal:/limpiar" --entrypoint rm "$imagen" -rf /limpiar/18 >/dev/null 2>&1 || true
    rm -f -- "$temporal/carrera.sql" "$temporal/carrera.log" "$temporal/prealter.sql" "$temporal/prealter.log"
    rmdir "$temporal" 2>/dev/null || true
  fi
}
trap limpiar EXIT
contenedor=$(docker run -d --rm --pull never --network none --name "$nombre" \
  --env POSTGRES_HOST_AUTH_METHOD=trust -v "$temporal:/var/lib/postgresql" "$imagen")
for _ in $(seq 1 120); do
  if docker logs "$contenedor" 2>&1 | grep -q 'PostgreSQL init process complete' &&
     docker exec "$contenedor" pg_isready -q -U postgres -d postgres; then break; fi
  sleep 0.5
done
docker exec "$contenedor" pg_isready -q -U postgres -d postgres
admin() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres; }
scalar() { docker exec "$contenedor" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
ejecutor() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_b56_rrhh -d postgres; }
[[ $(scalar 'SHOW server_version') == 18.4* ]]
[[ -z $(docker inspect --format '{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{end}}{{end}}' "$contenedor") ]]
admin < "$directorio/preimagen_sintetica.sql" >/dev/null
b48=$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000048_consulta_auditoria_participacion.up.sql
b56=$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000056_motivo_traza_auditoria_participacion
firma='vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
admin < "$b48" >/dev/null
anterior=$(scalar "SELECT md5(pg_get_functiondef('$firma'::regprocedure))")
[[ $(scalar "SELECT encode(sha256(convert_to(pg_get_functiondef('$firma'::regprocedure),'UTF8')),'hex')") == '363c4dfa08690195a47ad8ea90478ddf8edd12b0dee8a64b48180571536c3d91' ]]
[[ $(scalar "SELECT pg_get_userbyid(p.proowner)||':'||p.proacl::text FROM pg_proc p WHERE p.oid='$firma'::regprocedure") == 'vec_bolsa_llamamientos_propietario:{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}' ]]
rechazar_up() {
  local salida_up
  if salida_up=$(admin < "$b56.up.sql" 2>&1); then echo 'B56: UP aceptó preimagen alterada' >&2; exit 1; fi
  [[ $salida_up == *'estado incompatible para consulta de auditoria Bolsa'* ]]
}
admin <<'SQL' >/dev/null
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $hotfix$
BEGIN
 EXECUTE replace(pg_get_functiondef('vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),
                 'Motivo reservado en Bolsa','Motivo alterado en Bolsa');
END $hotfix$;
COMMIT;
SQL
rechazar_up
admin <<'SQL' >/dev/null
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $restaurar$
BEGIN
 EXECUTE replace(pg_get_functiondef('vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),
                 'Motivo alterado en Bolsa','Motivo reservado en Bolsa');
END $restaurar$;
COMMIT;
SQL
[[ $(scalar "SELECT md5(pg_get_functiondef('$firma'::regprocedure))") == "$anterior" ]]
admin <<'SQL' >/dev/null
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
ALTER FUNCTION vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) SET search_path=pg_catalog,pg_temp;
COMMIT;
SQL
rechazar_up
[[ $(scalar "SELECT md5(pg_get_functiondef('$firma'::regprocedure))") != "$anterior" ]]
admin <<'SQL' >/dev/null
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
ALTER FUNCTION vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) SET search_path=pg_catalog;
COMMIT;
SQL
[[ $(scalar "SELECT md5(pg_get_functiondef('$firma'::regprocedure))") == "$anterior" ]]
admin <<'SQL' >/dev/null
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_b56_sin_permiso;
COMMIT;
SQL
rechazar_up
admin <<'SQL' >/dev/null
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
REVOKE EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_b56_sin_permiso;
COMMIT;
SQL
[[ $(scalar "SELECT pg_get_userbyid(p.proowner)||':'||p.proacl::text FROM pg_proc p WHERE p.oid='$firma'::regprocedure") == 'vec_bolsa_llamamientos_propietario:{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}' ]]
echo 'OK preimagen B48: hotfix de cuerpo/configuración y GRANT extra rechazados'
# Intercalación antes del ALTER: el primer DO leyó B48, después otro DDL
# confirma un hotfix. SERIALIZABLE debe abortar la escritura de pg_proc y
# conservar el hotfix, sin sustituirlo por B56.
sed '/^ALTER FUNCTION vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(/i SELECT pg_sleep(4);' "$b56.up.sql" > "$temporal/prealter.sql"
admin < "$temporal/prealter.sql" > "$temporal/prealter.log" 2>&1 &
pid_prealter=$!
en_barrera=false
for _ in $(seq 1 100); do
  if [[ $(scalar "SELECT count(*) FROM pg_stat_activity WHERE query='SELECT pg_sleep(4);' AND state='active'") == 1 ]]; then en_barrera=true; break; fi
  sleep 0.05
done
[[ $en_barrera == true ]] || { echo 'B56: no alcanzó barrera previa al ALTER' >&2; exit 1; }
admin <<'SQL' >/dev/null
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $hotfix$
BEGIN
 EXECUTE replace(pg_get_functiondef('vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),
                 'Motivo reservado en Bolsa','Motivo alterado en Bolsa');
END $hotfix$;
COMMIT;
SQL
if wait "$pid_prealter"; then echo 'B56: UP sustituyó hotfix confirmado tras primera sonda' >&2; exit 1; fi
[[ $(scalar "SELECT md5(pg_get_functiondef('$firma'::regprocedure))") != "$anterior" ]]
admin <<'SQL' >/dev/null
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $restaurar$
BEGIN
 EXECUTE replace(pg_get_functiondef('vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),
                 'Motivo alterado en Bolsa','Motivo reservado en Bolsa');
END $restaurar$;
COMMIT;
SQL
[[ $(scalar "SELECT md5(pg_get_functiondef('$firma'::regprocedure))") == "$anterior" ]]
echo 'OK carrera previa al ALTER: hotfix confirmado conservado, UP abortado'
sed 's/^COMMIT;$/ROLLBACK;/' "$b56.up.sql" | admin >/dev/null
[[ $(scalar "SELECT md5(pg_get_functiondef('$firma'::regprocedure))") == "$anterior" ]]
# Intercalación adversarial: la migración ya verificó B48 y mantiene el objeto
# función bloqueado mientras duerme. GRANT y hotfix con la misma firma deben
# fallar por lock_timeout; ninguno puede cambiar la preimagen antes del COMMIT.
sed '/^CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(/i SELECT pg_sleep(4);' "$b56.up.sql" > "$temporal/carrera.sql"
admin < "$temporal/carrera.sql" > "$temporal/carrera.log" 2>&1 &
pid_carrera=$!
en_barrera=false
for _ in $(seq 1 100); do
  if [[ $(scalar "SELECT count(*) FROM pg_stat_activity WHERE query='SELECT pg_sleep(4);' AND state='active'") == 1 ]]; then en_barrera=true; break; fi
  sleep 0.05
done
[[ $en_barrera == true ]] || { echo 'B56: no alcanzó la barrera de carrera' >&2; exit 1; }
if admin <<'SQL' >/dev/null 2>&1
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL lock_timeout='700ms';
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_b56_sin_permiso;
COMMIT;
SQL
then echo 'B56: GRANT concurrente confirmó durante UP' >&2; exit 1; fi
if admin <<'SQL' >/dev/null 2>&1
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL lock_timeout='700ms';
DO $hotfix$
BEGIN
 EXECUTE replace(pg_get_functiondef('vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),
                 'Motivo reservado en Bolsa','Motivo alterado en Bolsa');
END $hotfix$;
COMMIT;
SQL
then echo 'B56: hotfix concurrente confirmó durante UP' >&2; exit 1; fi
wait "$pid_carrera" || { cat "$temporal/carrera.log" >&2; exit 1; }
nuevo=$(scalar "SELECT md5(pg_get_functiondef('$firma'::regprocedure))")
[[ $nuevo != "$anterior" ]]
[[ $(scalar "SELECT pg_get_userbyid(p.proowner)||':'||p.proacl::text FROM pg_proc p WHERE p.oid='$firma'::regprocedure") == 'vec_bolsa_llamamientos_propietario:{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}' ]]
echo 'OK carrera B56: hotfix y GRANT concurrentes no sustituyen preimagen'
if admin < "$b56.up.sql" >/dev/null 2>&1; then echo 'B56: doble UP aceptado' >&2; exit 1; fi
sed 's/^COMMIT;$/ROLLBACK;/' "$b56.down.sql" | admin >/dev/null
[[ $(scalar "SELECT md5(pg_get_functiondef('$firma'::regprocedure))") == "$nuevo" ]]
echo 'OK B56: UP/DOWN en ROLLBACK, UP, doble UP denegado'
admin < "$directorio/fixture_y_consulta.sql" >/dev/null
historia=$(scalar "SELECT (SELECT count(*) FROM vec_bolsa_llamamientos.situacion_participacion)::text || ':' || (SELECT count(*) FROM vec_bolsa_llamamientos.datos_contacto_participacion)::text || ':' || (SELECT count(*) FROM vec_bolsa_llamamientos.traza_valor_participacion)::text")
if salida_down=$(admin < "$b56.down.sql" 2>&1); then
  echo 'B56: DOWN aceptado con historia' >&2; exit 1
fi
[[ $salida_down == *'B56: DOWN denegado con historia de Bolsa'* ]]
[[ $(scalar "SELECT md5(pg_get_functiondef('$firma'::regprocedure))") == "$nuevo" ]]
[[ $(scalar "SELECT has_function_privilege('vec_b56_rrhh','$firma','EXECUTE') AND NOT has_function_privilege('vec_b56_sin_permiso','$firma','EXECUTE') AND NOT has_table_privilege('vec_b56_rrhh','vec_bolsa_llamamientos.traza_valor_participacion','SELECT') AND NOT has_table_privilege('vec_b56_rrhh','vec_bolsa_llamamientos.datos_contacto_participacion','SELECT')") == t ]]
ejecutor < "$directorio/comprobar_ejecutor.sql" >/dev/null
echo 'OK rol nominal: dos situaciones, dos contactos, motivo por cambio, cursor límite 1, ACL y denegación'
docker restart "$contenedor" >/dev/null
for _ in $(seq 1 120); do docker exec "$contenedor" pg_isready -q -U postgres -d postgres && break; sleep 0.5; done
ejecutor < "$directorio/comprobar_ejecutor.sql" >/dev/null
[[ $(scalar "SELECT md5(pg_get_functiondef('$firma'::regprocedure))") == "$nuevo" ]]
if salida_down=$(admin < "$b56.down.sql" 2>&1); then
  echo 'B56: DOWN aceptado tras reinicio con historia' >&2; exit 1
fi
[[ $salida_down == *'B56: DOWN denegado con historia de Bolsa'* ]]
[[ $(scalar "SELECT md5(pg_get_functiondef('$firma'::regprocedure))") == "$nuevo" ]]
[[ $(scalar "SELECT (SELECT count(*) FROM vec_bolsa_llamamientos.situacion_participacion)::text || ':' || (SELECT count(*) FROM vec_bolsa_llamamientos.datos_contacto_participacion)::text || ':' || (SELECT count(*) FROM vec_bolsa_llamamientos.traza_valor_participacion)::text") == "$historia" ]]
echo 'OK reinicio PG18: mismas filas y definición; DOWN denegado con historia'
