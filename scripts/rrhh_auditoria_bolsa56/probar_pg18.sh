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
sed 's/^COMMIT;$/ROLLBACK;/' "$b56.up.sql" | admin >/dev/null
[[ $(scalar "SELECT md5(pg_get_functiondef('$firma'::regprocedure))") == "$anterior" ]]
admin < "$b56.up.sql" >/dev/null
nuevo=$(scalar "SELECT md5(pg_get_functiondef('$firma'::regprocedure))")
[[ $nuevo != "$anterior" ]]
if admin < "$b56.up.sql" >/dev/null 2>&1; then echo 'B56: doble UP aceptado' >&2; exit 1; fi
sed 's/^COMMIT;$/ROLLBACK;/' "$b56.down.sql" | admin >/dev/null
[[ $(scalar "SELECT md5(pg_get_functiondef('$firma'::regprocedure))") == "$nuevo" ]]
echo 'OK B56: UP/DOWN en ROLLBACK, UP, doble UP denegado'
admin < "$directorio/fixture_y_consulta.sql" >/dev/null
[[ $(scalar "SELECT has_function_privilege('vec_b56_rrhh','$firma','EXECUTE') AND NOT has_function_privilege('vec_b56_sin_permiso','$firma','EXECUTE') AND NOT has_table_privilege('vec_b56_rrhh','vec_bolsa_llamamientos.traza_valor_participacion','SELECT') AND NOT has_table_privilege('vec_b56_rrhh','vec_bolsa_llamamientos.datos_contacto_participacion','SELECT')") == t ]]
ejecutor < "$directorio/comprobar_ejecutor.sql" >/dev/null
echo 'OK rol nominal: dos situaciones, dos contactos, motivo por cambio, cursor límite 1, ACL y denegación'
docker restart "$contenedor" >/dev/null
for _ in $(seq 1 120); do docker exec "$contenedor" pg_isready -q -U postgres -d postgres && break; sleep 0.5; done
ejecutor < "$directorio/comprobar_ejecutor.sql" >/dev/null
[[ $(scalar "SELECT md5(pg_get_functiondef('$firma'::regprocedure))") == "$nuevo" ]]
echo 'OK reinicio PG18: mismas filas y definición; sin DOWN con historia'
