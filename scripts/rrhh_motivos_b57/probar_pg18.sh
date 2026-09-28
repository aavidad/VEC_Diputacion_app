#!/usr/bin/env bash
set -Eeuo pipefail
dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd); repo=$(git -C "$dir" rev-parse --show-toplevel)
image=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}; tmp=$(mktemp -d /dev/shm/vec-b57.XXXXXXXX); chmod 1777 "$tmp"; name=vec-b57-${tmp##*.}; c=
clean(){ [[ -n $c ]] && docker rm -f "$c" >/dev/null 2>&1 || true; [[ -d $tmp ]] && docker run --rm --network none --pull never -v "$tmp:/x" --entrypoint rm "$image" -rf /x/18 >/dev/null 2>&1 || true; rmdir "$tmp" 2>/dev/null || true; }; trap clean EXIT
c=$(docker run -d --rm --pull never --network none --name "$name" -e POSTGRES_HOST_AUTH_METHOD=trust -v "$tmp:/var/lib/postgresql" "$image")
for _ in $(seq 1 120); do
  if docker logs "$c" 2>&1 | grep -q 'PostgreSQL init process complete' &&
     docker exec "$c" pg_isready -q -U postgres -d postgres; then break; fi
  sleep .5
done
sleep 2
psql(){ docker exec -i "$c" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres; }; q(){ docker exec "$c" psql -X -At -q -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
[[ $(q 'SHOW server_version') == 18.4* ]]
ad3=$repo/deploy/postgresql/autorizacion_atestada_v3
psql < "$ad3/pruebas_sql/ad3_101_preimagen_minima.sql" >/dev/null
psql < "$ad3/migraciones/000101_consumidor_consulta_reincorporacion_titular_bolsa.up.sql" >/dev/null
psql < "$ad3/migraciones/000102_consumidor_catalogo_causas_bolsa.up.sql" >/dev/null
psql < "$ad3/migraciones/000103_consumidor_consulta_catalogo_causas_bolsa.up.sql" >/dev/null
psql < "$ad3/migraciones/000104_consumidor_propuesta_catalogo_causas_bolsa.up.sql" >/dev/null
psql < "$ad3/migraciones/000105_consumidor_consulta_propuesta_causas_bolsa.up.sql" >/dev/null
psql < "$dir/preimagen_sintetica.sql" >/dev/null
b48=$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000048_consulta_auditoria_participacion.up.sql; b56=${B56_UP_SQL:-$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000056_motivo_traza_auditoria_participacion.up.sql}; b57=$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000057_causas_catalogadas_participacion
[[ -r $b56 ]] || { echo 'falta B56 final; use B56_UP_SQL para la fuente b28' >&2; exit 2; }
psql < "$b48" >/dev/null; psql < "$b56" >/dev/null; before=$(q "SELECT md5(pg_get_functiondef('vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))")
sed 's/^COMMIT;$/ROLLBACK;/' "$b57.up.sql" | psql >/dev/null; [[ $(q "SELECT to_regclass('vec_bolsa_llamamientos.causa_participacion_catalogo') IS NULL") == t ]]
psql < "$b57.up.sql" >/dev/null; if psql < "$b57.up.sql" >/dev/null 2>&1; then echo 'B57 doble UP aceptado' >&2; exit 1; fi
sed 's/^COMMIT;$/ROLLBACK;/' "$b57.down.sql" | psql >/dev/null; [[ $(q "SELECT md5(pg_get_functiondef('vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))") != "$before" ]]
echo 'OK B57: UP/DOWN rollback, UP y doble UP denegado'; psql < "$dir/doble_publicacion.sql" >/dev/null; psql < "$dir/fixture_y_comprobar.sql" >/dev/null
docker restart "$c" >/dev/null; for _ in $(seq 1 120); do docker exec "$c" pg_isready -q -U postgres -d postgres && break; sleep .5; done
psql < "$dir/comprobar_reinicio.sql" >/dev/null
[[ $(q "SELECT count(*) FROM vec_bolsa_llamamientos.propuesta_causa_participacion") == 1 ]]; [[ $(q "SELECT count(*) FROM vec_bolsa_llamamientos.causa_participacion_catalogo WHERE propuesta_ref IS NOT NULL") == 1 ]]
echo 'OK B57: propuesta/GET/publicación por dos RRHH, B2/B4, B56, ACL, replay y reinicio PG18'
