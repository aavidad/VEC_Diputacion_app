#!/usr/bin/env bash
set -Eeuo pipefail
repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
contenedor="vec-bolsa-relleno-pg18-$$"
limpiar() { docker rm -f "$contenedor" >/dev/null 2>&1 || true; }
trap limpiar EXIT
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm}
clave_prueba=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')
[[ ${#clave_prueba} -eq 64 ]] || { echo 'sin clave efimera de prueba' >&2; exit 1; }
docker run --detach --rm --name "$contenedor" -e POSTGRES_PASSWORD="$clave_prueba" "$imagen" >/dev/null
for _ in $(seq 1 60); do
  docker exec "$contenedor" pg_isready -U postgres >/dev/null 2>&1 && break
  sleep 1
done
psql_pg() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres; }
for ruta in \
  deploy/postgresql/autorizacion/roles_up.sql \
  deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql \
  deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql \
  deploy/postgresql/bolsa_llamamientos/roles_up.sql \
  deploy/postgresql/bolsa_llamamientos/migraciones_autorizacion/000001_revalidacion_llamamientos.up.sql \
  deploy/postgresql/bolsa_llamamientos/migraciones/000007_constitucion_bolsa.up.sql \
  deploy/postgresql/bolsa_llamamientos/migraciones/000008_vinculo_candidato.up.sql; do
  psql_pg < "$repo/$ruta" >/dev/null
done
# La instalación también se ensaya y no debe dejar la función creada.
sed '/^COMMIT;$/s//ROLLBACK;/' "$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000021_rellenar_vinculos_candidato.up.sql" | psql_pg >/dev/null
psql_pg <<'SQL' >/dev/null
DO $verificar$ BEGIN
 IF to_regprocedure('vec_bolsa_llamamientos.rellenar_vinculos_candidato_v1(text,jsonb,timestamptz)') IS NOT NULL THEN
  RAISE EXCEPTION 'ensayo de migracion persistio';
 END IF;
END $verificar$;
SQL
psql_pg < "$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000021_rellenar_vinculos_candidato.up.sql" >/dev/null
psql_pg <<'SQL' >/dev/null
SET ROLE vec_bolsa_llamamientos_ejecutor;
DO $acl$ BEGIN
  IF has_function_privilege(current_user,
    'vec_bolsa_llamamientos.rellenar_vinculos_candidato_v1(text,jsonb,timestamptz)',
    'EXECUTE') THEN
    RAISE EXCEPTION 'el ejecutor ordinario puede invocar el relleno';
  END IF;
END $acl$;
SQL
psql_pg <<'SQL'
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida
 (bolsa_ref,version,huella_bolsa_sha256,bolsa_canonica,categoria_ref,vigente_desde,estado,registrada_en)
VALUES ('bolsa:antigua',1,encode(sha256(convert_to('{}','UTF8')),'hex'),convert_to('{}','UTF8'),
        'categoria:prueba','2026-09-18T00:00:00Z','vigente','2026-09-18T00:00:00Z'),
       ('bolsa:nueva',1,encode(sha256(convert_to('[]','UTF8')),'hex'),convert_to('[]','UTF8'),
        'categoria:prueba','2026-09-22T00:00:00Z','vigente','2026-09-22T00:00:00Z');
INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa
 (instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,bolsa_ref,version_bolsa,
  huella_bolsa_sha256,total_participaciones,referida_en,generada_en,registrada_en)
SELECT 'instantanea:'||b.bolsa_ref,1,encode(sha256(convert_to('{}','UTF8')),'hex'),convert_to('{}','UTF8'),
       b.bolsa_ref,1,b.huella_bolsa_sha256,CASE WHEN b.bolsa_ref='bolsa:antigua' THEN 2 ELSE 1 END,
       b.registrada_en,b.registrada_en,b.registrada_en
 FROM vec_bolsa_llamamientos.bolsa_constituida b;
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada
 (instantanea_ref,version_instantanea,orden,participacion_ref,fila_numero)
VALUES ('instantanea:bolsa:antigua',1,1,'participacion:antigua:1',2),
       ('instantanea:bolsa:antigua',1,2,'participacion:antigua:2',3),
       ('instantanea:bolsa:nueva',1,1,'participacion:nueva:1',2);
INSERT INTO vec_bolsa_llamamientos.constitucion
 (acta_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,instantanea_ref,version_instantanea,
  huella_instantanea_sha256,categoria_ref,actor_ref,confirmada_en,registrada_en)
SELECT 'acta:'||b.bolsa_ref,b.bolsa_ref,1,b.huella_bolsa_sha256,'instantanea:'||b.bolsa_ref,1,
       i.huella_instantanea_sha256,b.categoria_ref,'actor:prueba',b.registrada_en,b.registrada_en
 FROM vec_bolsa_llamamientos.bolsa_constituida b
 JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa i ON i.bolsa_ref=b.bolsa_ref;
INSERT INTO vec_bolsa_llamamientos.vinculo_candidato
 (participacion_ref,candidato_ref,acta_ref,instantanea_ref,version_instantanea,registrada_en)
VALUES ('participacion:nueva:1','can_'||repeat('c',43),'acta:bolsa:nueva',
        'instantanea:bolsa:nueva',1,'2026-09-22T00:00:00Z');
COMMIT;
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $prueba$ DECLARE r jsonb; BEGIN
 r:=vec_bolsa_llamamientos.rellenar_vinculos_candidato_v1('acta:bolsa:antigua',
   '[{"fila_numero":2,"candidato_ref":"can_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
     {"fila_numero":3,"candidato_ref":"can_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}]'::jsonb,
   '2026-09-23T00:00:00Z');
 IF r <> '{"nuevos":2,"existentes":0}'::jsonb THEN RAISE EXCEPTION 'ensayo: %',r; END IF;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.vinculo_candidato) <> 3 THEN RAISE EXCEPTION 'ensayo incompleto'; END IF;
END $prueba$;
ROLLBACK;
DO $prueba$ DECLARE r jsonb; BEGIN
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.vinculo_candidato) <> 1 THEN RAISE EXCEPTION 'ROLLBACK dejo vinculos'; END IF;
 r:=vec_bolsa_llamamientos.rellenar_vinculos_candidato_v1('acta:bolsa:antigua',
   '[{"fila_numero":2,"candidato_ref":"can_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
     {"fila_numero":3,"candidato_ref":"can_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}]'::jsonb,
   '2026-09-23T00:00:00Z');
 IF r <> '{"nuevos":2,"existentes":0}'::jsonb THEN RAISE EXCEPTION 'primer relleno: %',r; END IF;
 r:=vec_bolsa_llamamientos.rellenar_vinculos_candidato_v1('acta:bolsa:antigua',
   '[{"fila_numero":2,"candidato_ref":"can_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
     {"fila_numero":3,"candidato_ref":"can_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}]'::jsonb,
   '2026-09-23T00:00:01Z');
 IF r <> '{"nuevos":0,"existentes":2}'::jsonb THEN RAISE EXCEPTION 'replay: %',r; END IF;
 r:=vec_bolsa_llamamientos.rellenar_vinculos_candidato_v1('acta:bolsa:nueva',
   '[{"fila_numero":2,"candidato_ref":"can_ccccccccccccccccccccccccccccccccccccccccccc"}]'::jsonb,
   '2026-09-23T00:00:00Z');
 IF r <> '{"nuevos":0,"existentes":1}'::jsonb THEN RAISE EXCEPTION 'nueva: %',r; END IF;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.vinculo_candidato) <> 3
    OR (SELECT registrada_en FROM vec_bolsa_llamamientos.vinculo_candidato WHERE participacion_ref='participacion:nueva:1') <> '2026-09-22T00:00:00Z' THEN
   RAISE EXCEPTION 'historia cambiada';
 END IF;
END $prueba$;
SQL
# DOWN debe fallar cuando ya existe historia.
if psql_pg < "$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000021_rellenar_vinculos_candidato.down.sql" >/dev/null 2>&1; then
  echo 'DOWN retiró función con historia' >&2
  exit 1
fi
printf 'PG18 relleno Bolsa 000021: ensayo, inserción, replay, constitución nueva y DOWN protegido OK\n'
