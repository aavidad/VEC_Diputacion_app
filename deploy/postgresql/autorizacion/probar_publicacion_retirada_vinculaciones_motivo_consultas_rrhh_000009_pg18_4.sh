#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-autorizacion-motivos-rrhh-000009-${USER:-usuario}-$$"
base=vec_autorizacion_motivos_rrhh_000009
clave=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
salidas=$(mktemp -d)
up=deploy/postgresql/autorizacion/migraciones/000009_publicacion_retirada_vinculaciones_motivo_consultas_rrhh.up.sql
down=deploy/postgresql/autorizacion/migraciones/000009_publicacion_retirada_vinculaciones_motivo_consultas_rrhh.down.sql
actor_a='Vec-Motivos.Proyector-A'
actor_b=vec_motivos_rrhh_proyector_b

limpiar() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  rm -rf "$salidas"
}
trap limpiar EXIT INT TERM

docker run --detach --rm --name "$contenedor" \
  --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave" \
  "$imagen" >/dev/null
disponible=false
for _ in $(seq 1 120); do
  if docker exec "$contenedor" pg_isready -U postgres -d "$base" \
      >/dev/null 2>&1; then
    disponible=true
    break
  fi
  sleep 1
done
[[ $disponible == true ]] || {
  docker logs "$contenedor" >&2 || true
  echo 'PostgreSQL 18.4 no quedó disponible' >&2
  exit 1
}
[[ $(docker exec "$contenedor" psql -XAtq -U postgres -d "$base" \
  -c "SELECT current_setting('server_version_num')") == 180004 ]]

psql_archivo() {
  docker exec --interactive "$contenedor" psql -Xq \
    --set ON_ERROR_STOP=1 -U postgres -d "$base" < "$raiz/$1"
}
psql_valor() {
  docker exec "$contenedor" psql -XAtq --set ON_ERROR_STOP=1 \
    -U postgres -d "$base" -c "$1"
}
actor_valor() {
  local actor=$1 consulta=$2
  docker exec --env PGPASSWORD="$clave" "$contenedor" psql -XAtq \
    --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
    -h 127.0.0.1 -U "$actor" -d "$base" -c "$consulta"
}
exigir_fallo_up() {
  if psql_archivo "$up" >/dev/null 2>&1; then
    echo "000009 up aceptó estado hostil: $1" >&2
    exit 1
  fi
}
exigir_fallo_down() {
  if psql_archivo "$down" >/dev/null 2>&1; then
    echo "000009 down aceptó estado hostil: $1" >&2
    exit 1
  fi
}
exigir_colision_opaca() {
  local actor=$1 consulta=$2 caso=$3 salida
  if salida=$(actor_valor "$actor" "$consulta" 2>&1); then
    echo "faltó 23505: $caso" >&2
    exit 1
  fi
  grep -Fq 'colision de identidad RRHH' <<<"$salida"
  grep -Fq '23505' <<<"$salida"
  if grep -Eiq 'DETAIL:|constraint|Key \(' <<<"$salida"; then
    echo "23505 filtró detalle: $caso" >&2
    exit 1
  fi
}
huella_fundamentos() {
  docker exec "$contenedor" pg_dump -U postgres -d "$base" --schema-only \
    -t vec_autorizacion.motivo_v2_evento_origen \
    -t vec_autorizacion.motivo_v2_catalogo_publicado \
    -t vec_autorizacion.motivo_v2_entrada \
    -t vec_autorizacion.motivo_v2_retirada \
    -t vec_autorizacion.motivo_v2_checkpoint_origen \
    -t vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 \
    -t vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1 |
    sed '/^\\restrict /d;/^\\unrestrict /d' | sha256sum | cut -d' ' -f1
}
huella_000009() {
  docker exec "$contenedor" pg_dump -U postgres -d "$base" --schema-only \
    -t vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 |
    sed '/^\\restrict /d;/^\\unrestrict /d' | sha256sum | cut -d' ' -f1
}

docker exec --interactive "$contenedor" psql -Xq \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
DO $b$ BEGIN
  EXECUTE format('REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',
                 current_database());
END $b$;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL

for archivo in \
  deploy/postgresql/contexto_actor_v1/roles_up.sql \
  deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql \
  deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql \
  deploy/postgresql/autorizacion/roles_up.sql \
  deploy/postgresql/autorizacion/roles_v2_up.sql \
  deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql \
  deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql \
  deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql \
  deploy/postgresql/autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql \
  deploy/postgresql/autorizacion/migraciones/000005_registro_decisiones_contexto_actor_v3.up.sql \
  deploy/postgresql/autorizacion/migraciones/000006_funcion_registro_decisiones_contexto_actor_v3.up.sql \
  deploy/postgresql/autorizacion/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.up.sql \
  deploy/postgresql/autorizacion/migraciones/000008_vinculaciones_motivo_consultas_rrhh.up.sql
do
  psql_archivo "$archivo" >/dev/null
done

huella_base=$(huella_fundamentos)
docker exec --interactive "$contenedor" psql -Xq \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
CREATE ROLE vec_motivos_rrhh_acl_hostil NOLOGIN BYPASSRLS;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_autorizacion_propietario
  IN SCHEMA vec_autorizacion
  GRANT SELECT, INSERT ON TABLES TO vec_motivos_rrhh_acl_hostil;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_autorizacion_propietario
  IN SCHEMA vec_autorizacion
  GRANT EXECUTE ON FUNCTIONS TO vec_motivos_rrhh_acl_hostil;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_autorizacion_propietario
  IN SCHEMA vec_autorizacion
  GRANT USAGE ON TYPES TO vec_motivos_rrhh_acl_hostil;
SQL
exigir_fallo_up 'default privileges de tabla, función y tipo'
[[ $(psql_valor "SELECT (to_regclass('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1') IS NULL)::text") == true ]]
docker exec --interactive "$contenedor" psql -Xq \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
ALTER DEFAULT PRIVILEGES FOR ROLE vec_autorizacion_propietario
  IN SCHEMA vec_autorizacion
  REVOKE SELECT, INSERT ON TABLES FROM vec_motivos_rrhh_acl_hostil;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_autorizacion_propietario
  IN SCHEMA vec_autorizacion
  REVOKE EXECUTE ON FUNCTIONS FROM vec_motivos_rrhh_acl_hostil;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_autorizacion_propietario
  IN SCHEMA vec_autorizacion
  REVOKE USAGE ON TYPES FROM vec_motivos_rrhh_acl_hostil;
DROP ROLE vec_motivos_rrhh_acl_hostil;
SQL

docker exec --interactive "$contenedor" psql -Xq \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
ALTER TABLE vec_autorizacion.motivo_v2_entrada
  DROP CONSTRAINT motivo_v2_entrada_catalogo_fk;
ALTER TABLE vec_autorizacion.motivo_v2_entrada
  ADD CONSTRAINT motivo_v2_entrada_catalogo_fk
  FOREIGN KEY (catalogo_id,catalogo_version)
  REFERENCES vec_autorizacion.motivo_v2_catalogo_publicado(
    catalogo_id,catalogo_version) ON DELETE CASCADE;
SQL
exigir_fallo_up 'FK V2 recompuesta con ON DELETE CASCADE'
[[ $(psql_valor "SELECT (to_regclass('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1') IS NULL)::text") == true ]]
fk_entrada=$(psql_valor "SELECT pg_get_constraintdef(oid,true) FROM pg_constraint WHERE conrelid='vec_autorizacion.motivo_v2_entrada'::regclass AND conname='motivo_v2_entrada_catalogo_fk'")
[[ $fk_entrada == *'ON DELETE CASCADE' ]]
docker exec --interactive "$contenedor" psql -Xq \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
ALTER TABLE vec_autorizacion.motivo_v2_entrada
  DROP CONSTRAINT motivo_v2_entrada_catalogo_fk;
ALTER TABLE vec_autorizacion.motivo_v2_entrada
  ADD CONSTRAINT motivo_v2_entrada_catalogo_fk
  FOREIGN KEY (catalogo_id,catalogo_version)
  REFERENCES vec_autorizacion.motivo_v2_catalogo_publicado(
    catalogo_id,catalogo_version);
SQL
[[ $(psql_valor "SELECT pg_get_constraintdef(oid,true) FROM pg_constraint WHERE conrelid='vec_autorizacion.motivo_v2_entrada'::regclass AND conname='motivo_v2_entrada_catalogo_fk'") == 'FOREIGN KEY (catalogo_id, catalogo_version) REFERENCES vec_autorizacion.motivo_v2_catalogo_publicado(catalogo_id, catalogo_version)' ]]

psql_archivo "$up" >/dev/null
psql_archivo \
  deploy/postgresql/autorizacion/pruebas_sql/publicacion_retirada_vinculaciones_motivo_consultas_rrhh_000009.sql \
  >/dev/null
huella_instalada=$(huella_000009)
exigir_fallo_up 'segunda ejecución'
[[ $(huella_000009) == "$huella_instalada" ]]

# Safe-down rechaza política, trigger, índice, ACL, marca y dependencia alterados.
docker exec --interactive "$contenedor" psql -Xq \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
CREATE POLICY acceso_hostil ON
  vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1
  FOR SELECT USING (true);
SQL
exigir_fallo_down 'política adicional'
psql_valor "SET ROLE vec_autorizacion_propietario; DROP POLICY acceso_hostil ON vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1" >/dev/null

psql_valor "SET ROLE vec_autorizacion_propietario; CREATE INDEX evento_indice_hostil ON vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1(actor_tecnico_ref)" >/dev/null
exigir_fallo_down 'índice ordinario adicional'
psql_valor "SET ROLE vec_autorizacion_propietario; DROP INDEX vec_autorizacion.evento_indice_hostil" >/dev/null

docker exec --interactive "$contenedor" psql -Xq \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
DROP TRIGGER vinculacion_motivo_rrhh_evento_inmutable ON
  vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1;
CREATE TRIGGER vinculacion_motivo_rrhh_evento_inmutable
  BEFORE UPDATE OR DELETE ON
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1
  FOR EACH ROW WHEN (false) EXECUTE FUNCTION
    vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1();
SQL
exigir_fallo_down 'trigger WHEN(false)'
docker exec --interactive "$contenedor" psql -Xq \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
DROP TRIGGER vinculacion_motivo_rrhh_evento_inmutable ON
  vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1;
CREATE TRIGGER vinculacion_motivo_rrhh_evento_inmutable
  BEFORE UPDATE OR DELETE ON
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1
  FOR EACH ROW EXECUTE FUNCTION
    vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1();
SQL

psql_valor "SET ROLE vec_autorizacion_propietario; COMMENT ON FUNCTION vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1(text,text,bigint,text,text,text,integer,text,text,timestamptz) IS 'hostil'" >/dev/null
exigir_fallo_down 'marca de fachada'
psql_valor "SET ROLE vec_autorizacion_propietario; COMMENT ON FUNCTION vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1(text,text,bigint,text,text,text,integer,text,text,timestamptz) IS 'vec_autorizacion:vinculacion-motivo-consulta-rrhh:publicar-cuadro-v1:000009'" >/dev/null

psql_valor "SET ROLE vec_autorizacion_propietario; CREATE FUNCTION vec_autorizacion.resolver_motivo_cuadro_rrhh_v1() RETURNS boolean LANGUAGE sql AS 'SELECT true'" >/dev/null
exigir_fallo_down 'dependencia M1.3'
psql_valor "SET ROLE vec_autorizacion_propietario; DROP FUNCTION vec_autorizacion.resolver_motivo_cuadro_rrhh_v1()" >/dev/null

psql_archivo "$down" >/dev/null
[[ $(huella_fundamentos) == "$huella_base" ]]
psql_archivo "$up" >/dev/null

docker exec --interactive --env CLAVE="$clave" "$contenedor" \
  psql -Xq --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
\getenv clave CLAVE
CREATE ROLE "Vec-Motivos.Proyector-A" LOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS PASSWORD :'clave';
CREATE ROLE vec_motivos_rrhh_proyector_b LOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS PASSWORD :'clave';
GRANT CONNECT ON DATABASE vec_autorizacion_motivos_rrhh_000009
  TO "Vec-Motivos.Proyector-A",vec_motivos_rrhh_proyector_b;
GRANT vec_autorizacion_motivos_proyector
  TO "Vec-Motivos.Proyector-A",vec_motivos_rrhh_proyector_b
  WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL

publicado=$(psql_valor "SET ROLE vec_autorizacion_motivos_proyector; SELECT vec_autorizacion.publicar_motivos_autorizacion_v2('evento_11111111111111111111111111111111',1,repeat('1',64),'motivos_rrhh_prueba',1,repeat('2',64),'2026-01-01T00:00:00Z','[{\"clave\":\"motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\",\"vigente_desde\":\"2026-01-01T00:00:00.000000Z\",\"vigente_hasta\":null},{\"clave\":\"motivo_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\",\"vigente_desde\":\"2026-01-01T00:00:00.000000Z\",\"vigente_hasta\":null}]'::jsonb)")
[[ $publicado == t ]]

pub_c="SELECT vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1('evento_vinculacion_motivo_rrhh_10000000000000000000000000000001',repeat('1',64),1,'publicacion_motivo_rrhh_10000000000000000000000000000001',repeat('3',64),'motivos_rrhh_prueba',1,repeat('2',64),'motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','2026-02-01T10:00:00Z')"
[[ $(actor_valor "$actor_a" "$pub_c") == t ]]
pub_d_mismo="SELECT vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1('evento_vinculacion_motivo_rrhh_20000000000000000000000000000001',repeat('4',64),1,'publicacion_motivo_rrhh_20000000000000000000000000000001',repeat('5',64),'motivos_rrhh_prueba',1,repeat('2',64),'motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','2026-02-01T11:00:00Z')"
[[ $(actor_valor "$actor_b" "$pub_d_mismo") == f ]]
pub_d="SELECT vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1('evento_vinculacion_motivo_rrhh_20000000000000000000000000000002',repeat('6',64),1,'publicacion_motivo_rrhh_20000000000000000000000000000002',repeat('7',64),'motivos_rrhh_prueba',1,repeat('2',64),'motivo_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb','2026-02-01T11:00:00Z')"

# Barrera observable: publicación obtiene V2 antes de esperar el checkpoint nominal.
docker exec --interactive "$contenedor" psql -Xq --set ON_ERROR_STOP=1 \
  -U postgres -d "$base" >"$salidas/bloqueo" <<'SQL' &
BEGIN;
SET application_name='ct51_bloqueo_nominal';
SELECT 1 FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
 WHERE clase_consulta='detalle' FOR UPDATE;
SELECT pg_sleep(4);
COMMIT;
SQL
pid_bloqueo=$!
for _ in $(seq 1 40); do
  [[ $(psql_valor "SELECT count(*) FROM pg_stat_activity AS a WHERE a.application_name='ct51_bloqueo_nominal' AND EXISTS (SELECT 1 FROM pg_locks AS l WHERE l.pid=a.pid AND l.relation='vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass AND l.mode='RowShareLock' AND l.granted)") == 1 ]] && break
  sleep 0.1
done
(actor_valor "$actor_b" "SET application_name='ct51_orden_v2_nominal'; $pub_d" \
  >"$salidas/orden" 2>"$salidas/orden.err") &
pid_orden=$!
orden_observado=false
for _ in $(seq 1 40); do
  if [[ $(psql_valor "SELECT count(*) FROM pg_stat_activity AS a WHERE a.application_name='ct51_orden_v2_nominal' AND a.wait_event_type='Lock' AND EXISTS (SELECT 1 FROM pg_locks AS l WHERE l.pid=a.pid AND l.relation='vec_autorizacion.motivo_v2_checkpoint_origen'::regclass AND l.mode='RowShareLock' AND l.granted)") == 1 ]]; then
    orden_observado=true
    break
  fi
  sleep 0.1
done
[[ $orden_observado == true ]]
wait "$pid_bloqueo"
wait "$pid_orden"
grep -Fxq t "$salidas/orden"

# Replay exacto después de rotar LOGIN: no cambia actor ni crea filas.
[[ $(actor_valor "$actor_b" "$pub_c") == t ]]
[[ $(psql_valor "SELECT count(*) FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1") == 2 ]]
[[ $(psql_valor "SELECT actor_tecnico_ref FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 WHERE clase_consulta='cuadro' AND operacion='publicacion'") == "$actor_a" ]]
exigir_colision_opaca "$actor_b" \
  "SELECT vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1('evento_vinculacion_motivo_rrhh_10000000000000000000000000000001',repeat('1',64),1,'publicacion_motivo_rrhh_10000000000000000000000000000001',repeat('3',64),'motivos_rrhh_prueba',1,repeat('2',64),'motivo_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb','2026-02-01T10:00:00Z')" \
  'replay divergente'

filas_pre_rollback=$(psql_valor "SELECT count(*) FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1")
actor_valor "$actor_a" "BEGIN; SELECT vec_autorizacion.retirar_vinculacion_motivo_cuadro_rrhh_v1('evento_vinculacion_motivo_rrhh_10000000000000000000000000000009',repeat('a',64),1,'publicacion_motivo_rrhh_10000000000000000000000000000001',repeat('3',64),'2026-03-05T12:00:00Z'); ROLLBACK" >/dev/null
[[ $(psql_valor "SELECT count(*) FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1") == "$filas_pre_rollback" ]]

ret_c="SELECT vec_autorizacion.retirar_vinculacion_motivo_cuadro_rrhh_v1('evento_vinculacion_motivo_rrhh_10000000000000000000000000000002',repeat('8',64),1,'publicacion_motivo_rrhh_10000000000000000000000000000001',repeat('3',64),'2026-03-10T12:00:00Z')"
ret_d="SELECT vec_autorizacion.retirar_vinculacion_motivo_detalle_rrhh_v1('evento_vinculacion_motivo_rrhh_20000000000000000000000000000003',repeat('9',64),1,'publicacion_motivo_rrhh_20000000000000000000000000000002',repeat('7',64),'2026-03-01T12:00:00Z')"
[[ $(actor_valor "$actor_a" "$ret_c") == t ]]
[[ $(actor_valor "$actor_b" "$ret_d") == t ]]
[[ $(actor_valor "$actor_b" "$ret_c") == t ]]
[[ $(psql_valor "SELECT count(*) FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1") == 4 ]]
docker exec --interactive "$contenedor" psql -Xq --set ON_ERROR_STOP=1 \
  -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
DO $inmutable$
BEGIN
  BEGIN
    UPDATE vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1
       SET actor_tecnico_ref=actor_tecnico_ref
     WHERE clase_consulta='cuadro' AND operacion='publicacion';
    RAISE EXCEPTION 'UPDATE directo aceptado';
  EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
  END;
  BEGIN
    DELETE FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1
     WHERE clase_consulta='detalle' AND operacion='publicacion';
    RAISE EXCEPTION 'DELETE directo aceptado';
  EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
  END;
END
$inmutable$;
SQL
[[ $(psql_valor "SELECT count(*) FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1") == 4 ]]

pub_c2_antes_pub="SELECT vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1('evento_vinculacion_motivo_rrhh_10000000000000000000000000000003',repeat('a',64),2,'publicacion_motivo_rrhh_10000000000000000000000000000002',repeat('b',64),'motivos_rrhh_prueba',1,repeat('2',64),'motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','2026-01-15T00:00:00Z')"
pub_c2_antes_ret="${pub_c2_antes_pub/2026-01-15T00:00:00Z/2026-02-15T00:00:00Z}"
[[ $(actor_valor "$actor_a" "$pub_c2_antes_pub") == f ]]
[[ $(actor_valor "$actor_a" "$pub_c2_antes_ret") == f ]]
pub_d2_solape="SELECT vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1('evento_vinculacion_motivo_rrhh_20000000000000000000000000000004',repeat('a',64),2,'publicacion_motivo_rrhh_20000000000000000000000000000003',repeat('b',64),'motivos_rrhh_prueba',1,repeat('2',64),'motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','2026-03-05T00:00:00Z')"
[[ $(actor_valor "$actor_b" "$pub_d2_solape") == f ]]

# Carrera transversal por identidad global: un éxito y un 23505 opaco.
pub_c2="SELECT vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1('evento_vinculacion_motivo_rrhh_ffffffffffffffffffffffffffffffff',repeat('c',64),2,'publicacion_motivo_rrhh_30000000000000000000000000000001',repeat('d',64),'motivos_rrhh_prueba',1,repeat('2',64),'motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','2026-04-01T00:00:00Z')"
pub_d2="SELECT vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1('evento_vinculacion_motivo_rrhh_ffffffffffffffffffffffffffffffff',repeat('e',64),2,'publicacion_motivo_rrhh_40000000000000000000000000000001',repeat('f',64),'motivos_rrhh_prueba',1,repeat('2',64),'motivo_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb','2026-04-01T00:00:00Z')"
set +e
(actor_valor "$actor_a" "$pub_c2" >"$salidas/carrera_c" 2>&1) & p1=$!
(actor_valor "$actor_b" "$pub_d2" >"$salidas/carrera_d" 2>&1) & p2=$!
wait "$p1"; e1=$?
wait "$p2"; e2=$?
set -e
[[ $((e1+e2)) -ne 0 ]]
[[ $(grep -lFx t "$salidas/carrera_c" "$salidas/carrera_d" | wc -l) == 1 ]]
fallo="$salidas/carrera_c"; [[ $e1 -eq 0 ]] && fallo="$salidas/carrera_d"
grep -Fq 'colision de identidad RRHH' "$fallo"
grep -Fq '23505' "$fallo"
if grep -Eiq 'DETAIL:|constraint|Key \(' "$fallo"; then
  echo 'la carrera filtró detalle de UNIQUE' >&2
  exit 1
fi
[[ $(psql_valor "SELECT count(*) FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 WHERE publicacion_version=2") == 1 ]]

clase_v2=$(psql_valor "SELECT clase_consulta FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 WHERE publicacion_version=2")
retirado_v2=$(psql_valor "SET ROLE vec_autorizacion_motivos_proyector; SELECT vec_autorizacion.retirar_motivos_autorizacion_v2('evento_22222222222222222222222222222222',2,repeat('3',64),'motivos_rrhh_prueba',1,repeat('2',64),repeat('4',64),'2026-05-01T00:00:00Z')")
[[ $retirado_v2 == t ]]
if [[ $clase_v2 == cuadro ]]; then
  ret_v2="SELECT vec_autorizacion.retirar_vinculacion_motivo_cuadro_rrhh_v1('evento_vinculacion_motivo_rrhh_50000000000000000000000000000001',repeat('5',64),2,'publicacion_motivo_rrhh_30000000000000000000000000000001',repeat('d',64),'2026-06-01T00:00:00Z')"
  intento_post="SELECT vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1('evento_vinculacion_motivo_rrhh_60000000000000000000000000000001',repeat('b',64),2,'publicacion_motivo_rrhh_60000000000000000000000000000001',repeat('6',64),'motivos_rrhh_prueba',1,repeat('2',64),'motivo_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb','2026-06-02T00:00:00Z')"
else
  ret_v2="SELECT vec_autorizacion.retirar_vinculacion_motivo_detalle_rrhh_v1('evento_vinculacion_motivo_rrhh_50000000000000000000000000000001',repeat('5',64),2,'publicacion_motivo_rrhh_40000000000000000000000000000001',repeat('f',64),'2026-06-01T00:00:00Z')"
  intento_post="SELECT vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1('evento_vinculacion_motivo_rrhh_60000000000000000000000000000001',repeat('b',64),2,'publicacion_motivo_rrhh_60000000000000000000000000000001',repeat('6',64),'motivos_rrhh_prueba',1,repeat('2',64),'motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','2026-06-02T00:00:00Z')"
fi
[[ $(actor_valor "$actor_a" "$ret_v2") == t ]]
[[ $(actor_valor "$actor_b" "$intento_post") == f ]]

# Rollback y DML directo no dejan historia huérfana; down con evidencia falla.
filas=$(psql_valor "SELECT count(*) FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1")
actor_valor "$actor_a" "BEGIN; $ret_v2; ROLLBACK" >/dev/null
[[ $(psql_valor "SELECT count(*) FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1") == "$filas" ]]
exigir_fallo_down 'evidencia durable'
[[ $(psql_valor "SELECT count(*) FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1") -ge 3 ]]

echo 'OK: publicación/retirada nominal RRHH 000009 en PostgreSQL 18.4'
