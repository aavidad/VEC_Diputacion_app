#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-autorizacion-motivos-rrhh-000008-${USER:-usuario}-$$"
base=vec_autorizacion_motivos_rrhh_000008
clave=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')

limpiar() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

docker run --detach --rm --name "$contenedor" \
  --publish 127.0.0.1::5432 \
  --env POSTGRES_DB="$base" \
  --env POSTGRES_PASSWORD="$clave" \
  "$imagen" >/dev/null

postgresql_definitivo_disponible=false
for _ in $(seq 1 120); do
  if ! docker inspect --format '{{.State.Running}}' "$contenedor" \
    2>/dev/null | grep -Fxq true; then
    break
  fi
  if docker logs "$contenedor" 2>&1 |
      LC_ALL=C grep -Fq \
        'PostgreSQL init process complete; ready for start up.' &&
    docker exec "$contenedor" psql -XAt --set ON_ERROR_STOP=1 \
      -U postgres -d "$base" -c 'SELECT 1' 2>/dev/null |
      grep -Fxq 1; then
    postgresql_definitivo_disponible=true
    break
  fi
  sleep 1
done
if [[ $postgresql_definitivo_disponible != true ]]; then
  docker logs "$contenedor" >&2 || true
  echo 'PostgreSQL definitivo no quedó disponible' >&2
  exit 1
fi

version_mayor=$(docker exec "$contenedor" psql -XAt \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" \
  -c "SELECT current_setting('server_version_num')::integer / 10000")
[[ $version_mayor == 18 ]] || {
  echo "se requiere PostgreSQL 18; inició PostgreSQL $version_mayor" >&2
  exit 1
}

psql_archivo() {
  docker exec --interactive "$contenedor" psql -X \
    --set ON_ERROR_STOP=1 -U postgres -d "$base" < "$raiz/$1"
}

psql_valor() {
  docker exec "$contenedor" psql -XAt --set ON_ERROR_STOP=1 \
    -U postgres -d "$base" -c "$1"
}

exigir_fallo_reentrada() {
  local caso=$1
  if psql_archivo \
    deploy/postgresql/autorizacion/migraciones/000008_vinculaciones_motivo_consultas_rrhh.up.sql \
    >/dev/null 2>&1; then
    echo "000008 adoptó o reparó el estado hostil: $caso" >&2
    exit 1
  fi
}

exigir_fallo_down() {
  local caso=$1
  if psql_archivo \
    deploy/postgresql/autorizacion/migraciones/000008_vinculaciones_motivo_consultas_rrhh.down.sql \
    >/dev/null 2>&1; then
    echo "000008 down retiró un objeto ajeno o alterado: $caso" >&2
    exit 1
  fi
}

restablecer_000008() {
  psql_archivo \
    deploy/postgresql/autorizacion/migraciones/000008_vinculaciones_motivo_consultas_rrhh.down.sql
  psql_archivo \
    deploy/postgresql/autorizacion/migraciones/000008_vinculaciones_motivo_consultas_rrhh.up.sql
}

docker exec --interactive "$contenedor" psql -X \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
DO $bloque$ BEGIN
  EXECUTE pg_catalog.format(
    'REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',
    pg_catalog.current_database()
  );
END $bloque$;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL

psql_archivo deploy/postgresql/contexto_actor_v1/roles_up.sql
psql_archivo \
  deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql
psql_archivo \
  deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql

for archivo in \
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
  psql_archivo "$archivo"
done

# Reentrada exacta: no crea filas, políticas, funciones ni privilegios extra.
psql_archivo \
  deploy/postgresql/autorizacion/migraciones/000008_vinculaciones_motivo_consultas_rrhh.up.sql

[[ $(psql_valor \
  "SELECT count(*) FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1") == 2 ]]
[[ $(psql_valor \
  "SELECT count(*) FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1") == 0 ]]
[[ $(psql_valor \
  "SELECT (count(*) = 12 AND bool_and(t.tgenabled = 'O' AND t.tgisinternal))::text FROM pg_catalog.pg_trigger AS t JOIN pg_catalog.pg_constraint AS c ON c.oid = t.tgconstraint WHERE c.conrelid IN ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass) AND c.contype = 'f'") == true ]]
[[ $(psql_valor \
  "SELECT (count(*) = 5 AND bool_and(i.indisvalid AND i.indisready AND i.indislive AND i.indimmediate))::text FROM pg_catalog.pg_index AS i WHERE i.indrelid IN ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass) AND (i.indisprimary OR i.indisunique OR i.indisexclusion)") == true ]]
[[ $(psql_valor \
  "SELECT (count(*) = 0)::text FROM pg_catalog.pg_rewrite WHERE ev_class IN ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass)") == true ]]
[[ $(psql_valor \
  "SELECT bool_and(a.attcollation = t.typcollation)::text FROM pg_catalog.pg_attribute AS a JOIN pg_catalog.pg_type AS t ON t.oid = a.atttypid WHERE a.attrelid IN ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass) AND a.attnum > 0") == true ]]

# Ninguna de las dos tablas UNLOGGED se adopta ni vuelve a LOGGED.
docker exec --interactive "$contenedor" psql -X \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
ALTER TABLE
  vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
  SET UNLOGGED;
ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
  SET UNLOGGED;
SQL
exigir_fallo_reentrada 'tablas UNLOGGED'
[[ $(psql_valor \
  "SELECT (count(*) = 2 AND bool_and(relpersistence = 'u'))::text FROM pg_catalog.pg_class WHERE oid IN ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass)") == true ]]
docker exec --interactive "$contenedor" psql -X \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
  SET LOGGED;
ALTER TABLE
  vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
  SET LOGGED;
SQL
psql_archivo \
  deploy/postgresql/autorizacion/migraciones/000008_vinculaciones_motivo_consultas_rrhh.up.sql

# Un disparador homónimo con otro evento no se adopta ni se repara.
docker exec --interactive "$contenedor" psql -X \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
DROP TRIGGER vinculacion_motivo_rrhh_inmutable ON
  vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1;
CREATE TRIGGER vinculacion_motivo_rrhh_inmutable
  BEFORE INSERT ON vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
  FOR EACH ROW EXECUTE FUNCTION
    vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1();
SQL
exigir_fallo_reentrada 'disparador BEFORE INSERT homónimo'
[[ $(psql_valor \
  "SELECT (tgtype = 7 AND tgenabled = 'O')::text FROM pg_catalog.pg_trigger WHERE tgrelid = 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass AND tgname = 'vinculacion_motivo_rrhh_inmutable' AND NOT tgisinternal") == true ]]
restablecer_000008

# Un WHEN(false) conserva evento y función, pero anula el contrato completo.
docker exec --interactive "$contenedor" psql -X \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
DROP TRIGGER vinculacion_motivo_rrhh_inmutable ON
  vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1;
CREATE TRIGGER vinculacion_motivo_rrhh_inmutable
  BEFORE UPDATE OR DELETE ON
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
  FOR EACH ROW WHEN (false) EXECUTE FUNCTION
    vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1();
SQL
exigir_fallo_reentrada 'disparador con WHEN(false)'
[[ $(psql_valor \
  "SELECT (tgtype = 27 AND tgenabled = 'O' AND tgqual IS NOT NULL)::text FROM pg_catalog.pg_trigger WHERE tgrelid = 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass AND tgname = 'vinculacion_motivo_rrhh_inmutable' AND NOT tgisinternal") == true ]]
restablecer_000008

# Una ACL de tabla para un BYPASSRLS ajeno no se revoca ni adopta.
docker exec --interactive "$contenedor" psql -X \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
CREATE ROLE vec_motivos_rrhh_bypass NOLOGIN BYPASSRLS;
SET ROLE vec_autorizacion_propietario;
GRANT SELECT, DELETE ON
  vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
  TO vec_motivos_rrhh_bypass;
SQL
exigir_fallo_reentrada 'ACL de tabla para BYPASSRLS'
[[ $(psql_valor \
  "SELECT pg_catalog.has_table_privilege('vec_motivos_rrhh_bypass', 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1', 'DELETE')::text") == true ]]
restablecer_000008

# Una ACL de columna es independiente de relacl y también falla cerrada.
docker exec --interactive "$contenedor" psql -X \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
GRANT SELECT (entrada_clave) ON
  vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
  TO vec_motivos_rrhh_bypass;
SQL
exigir_fallo_reentrada 'ACL de columna para BYPASSRLS'
[[ $(psql_valor \
  "SELECT EXISTS (SELECT 1 FROM pg_catalog.pg_attribute AS a CROSS JOIN LATERAL pg_catalog.aclexplode(a.attacl) AS acl WHERE a.attrelid = 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass AND a.attname = 'entrada_clave' AND acl.grantee = 'vec_motivos_rrhh_bypass'::regrole)::text") == true ]]
restablecer_000008
psql_valor "DROP ROLE vec_motivos_rrhh_bypass" >/dev/null

# Una regla de reescritura hostil permanece instalada tras el rollback.
docker exec --interactive "$contenedor" psql -X \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
CREATE RULE vinculacion_motivo_rrhh_borrado_hostil AS
  ON DELETE TO vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
  DO INSTEAD NOTHING;
SQL
exigir_fallo_reentrada 'regla ON DELETE DO INSTEAD'
[[ $(psql_valor \
  "SELECT (count(*) = 1 AND bool_and(is_instead))::text FROM pg_catalog.pg_rewrite WHERE ev_class = 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass AND rulename = 'vinculacion_motivo_rrhh_borrado_hostil'") == true ]]
restablecer_000008

# Una colación no determinista no se oculta recreando la FK textual exacta.
docker exec --interactive "$contenedor" psql -X \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
CREATE COLLATION vec_autorizacion.colacion_motivo_rrhh_hostil (
  provider = icu, locale = 'und-u-ks-level2', deterministic = false
);
ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
  DROP CONSTRAINT vinculacion_motivo_rrhh_checkpoint_historia_fk;
ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
  ALTER COLUMN clase_consulta TYPE text
  COLLATE vec_autorizacion.colacion_motivo_rrhh_hostil;
ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
  ALTER COLUMN clase_consulta TYPE text
  COLLATE vec_autorizacion.colacion_motivo_rrhh_hostil;
ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
  ADD CONSTRAINT vinculacion_motivo_rrhh_checkpoint_historia_fk
  FOREIGN KEY (
    clase_consulta, ultima_publicacion_version,
    ultima_publicacion_ref, ultima_publicacion_huella_sha256
  ) REFERENCES vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 (
    clase_consulta, publicacion_version,
    publicacion_ref, publicacion_huella_sha256
  );
SQL
exigir_fallo_reentrada 'colación no determinista con FK textual exacta'
[[ $(psql_valor \
  "SELECT (count(*) = 2 AND bool_and(a.attcollation = 'vec_autorizacion.colacion_motivo_rrhh_hostil'::regcollation))::text FROM pg_catalog.pg_attribute AS a WHERE a.attrelid IN ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass) AND a.attname = 'clase_consulta'") == true ]]
[[ $(psql_valor \
  "SELECT ('CUADRO' COLLATE vec_autorizacion.colacion_motivo_rrhh_hostil IN ('cuadro', 'detalle'))::text") == true ]]
restablecer_000008
psql_valor \
  "DROP COLLATION vec_autorizacion.colacion_motivo_rrhh_hostil" >/dev/null

# Un índice UNIQUE adicional no forma parte del conjunto semántico nominal.
docker exec --interactive "$contenedor" psql -X \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
CREATE UNIQUE INDEX vinculacion_motivo_rrhh_indice_hostil
  ON vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 (catalogo_id);
SQL
exigir_fallo_reentrada 'índice UNIQUE adicional'
[[ $(psql_valor \
  "SELECT (indisunique AND indisvalid AND indisready AND indislive)::text FROM pg_catalog.pg_index WHERE indexrelid = 'vec_autorizacion.vinculacion_motivo_rrhh_indice_hostil'::regclass") == true ]]
restablecer_000008

# Los RI deshabilitados en la hija no se reparan durante la reentrada.
docker exec --interactive "$contenedor" psql -X \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
  DISABLE TRIGGER ALL;
ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
  ENABLE TRIGGER vinculacion_motivo_rrhh_inmutable;
ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
  ENABLE TRIGGER vinculacion_motivo_rrhh_no_truncar;
SQL
exigir_fallo_reentrada 'disparadores RI deshabilitados en tabla hija'
[[ $(psql_valor \
  "SELECT (count(*) = 6 AND bool_and(t.tgenabled = 'D'))::text FROM pg_catalog.pg_trigger AS t JOIN pg_catalog.pg_constraint AS c ON c.oid = t.tgconstraint WHERE c.conrelid IN ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass) AND c.contype = 'f' AND t.tgrelid = 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass") == true ]]
restablecer_000008

# También se vigilan los RI instalados en el extremo referenciado externo.
docker exec --interactive "$contenedor" psql -X \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
DO $deshabilitar$
DECLARE disparador record;
BEGIN
  FOR disparador IN
    SELECT t.tgrelid, t.tgname
      FROM pg_catalog.pg_trigger AS t
      JOIN pg_catalog.pg_constraint AS c ON c.oid = t.tgconstraint
     WHERE c.conname = 'vinculacion_motivo_rrhh_entrada_fk'
       AND t.tgrelid = 'vec_autorizacion.motivo_v2_entrada'::regclass
  LOOP
    EXECUTE pg_catalog.format(
      'ALTER TABLE %s DISABLE TRIGGER %I',
      disparador.tgrelid::regclass, disparador.tgname
    );
  END LOOP;
END
$deshabilitar$;
SQL
exigir_fallo_reentrada 'disparadores RI deshabilitados en tabla referenciada'
[[ $(psql_valor \
  "SELECT (count(*) = 2 AND bool_and(t.tgenabled = 'D'))::text FROM pg_catalog.pg_trigger AS t JOIN pg_catalog.pg_constraint AS c ON c.oid = t.tgconstraint WHERE c.conname = 'vinculacion_motivo_rrhh_entrada_fk' AND t.tgrelid = 'vec_autorizacion.motivo_v2_entrada'::regclass") == true ]]
docker exec --interactive "$contenedor" psql -X \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
DO $habilitar$
DECLARE disparador record;
BEGIN
  FOR disparador IN
    SELECT t.tgrelid, t.tgname
      FROM pg_catalog.pg_trigger AS t
      JOIN pg_catalog.pg_constraint AS c ON c.oid = t.tgconstraint
     WHERE c.conname = 'vinculacion_motivo_rrhh_entrada_fk'
       AND t.tgrelid = 'vec_autorizacion.motivo_v2_entrada'::regclass
  LOOP
    EXECUTE pg_catalog.format(
      'ALTER TABLE %s ENABLE TRIGGER %I',
      disparador.tgrelid::regclass, disparador.tgname
    );
  END LOOP;
END
$habilitar$;
SQL
psql_archivo \
  deploy/postgresql/autorizacion/migraciones/000008_vinculaciones_motivo_consultas_rrhh.up.sql

# Una política homónima SELECT TO PUBLIC permanece hostil tras el rollback.
docker exec --interactive "$contenedor" psql -X \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
DROP POLICY acceso_propietario_exacto ON
  vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1;
CREATE POLICY acceso_propietario_exacto ON
  vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
  FOR SELECT TO PUBLIC USING (true);
SQL
exigir_fallo_reentrada 'política SELECT TO PUBLIC homónima'
[[ $(psql_valor \
  "SELECT (polcmd = 'r' AND polroles = ARRAY[0::oid] AND pg_catalog.pg_get_expr(polqual, polrelid) = 'true' AND polwithcheck IS NULL)::text FROM pg_catalog.pg_policy WHERE polrelid = 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass AND polname = 'acceso_propietario_exacto'") == true ]]
restablecer_000008

# La ausencia de una FK requerida aborta sin reconstruirla implícitamente.
docker exec --interactive "$contenedor" psql -X \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
  DROP CONSTRAINT vinculacion_motivo_rrhh_entrada_fk;
SQL
exigir_fallo_reentrada 'clave foránea eliminada'
[[ $(psql_valor \
  "SELECT (count(*) = 0)::text FROM pg_catalog.pg_constraint WHERE conrelid = 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass AND conname = 'vinculacion_motivo_rrhh_entrada_fk'") == true ]]
restablecer_000008

# Una UNIQUE previa con la misma definición, pero sin marca 000008, es ajena.
psql_archivo \
  deploy/postgresql/autorizacion/migraciones/000008_vinculaciones_motivo_consultas_rrhh.down.sql
docker exec --interactive "$contenedor" psql -X \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
ALTER TABLE vec_autorizacion.motivo_v2_catalogo_publicado
  ADD CONSTRAINT motivo_v2_catalogo_referencia_completa_unica
  UNIQUE (
    catalogo_id, catalogo_version, catalogo_huella_publicada_sha256
  );
SQL
exigir_fallo_reentrada 'UNIQUE exacta sin procedencia 000008'
[[ $(psql_valor \
  "SELECT (pg_catalog.obj_description(oid, 'pg_constraint') IS NULL)::text FROM pg_catalog.pg_constraint WHERE conrelid = 'vec_autorizacion.motivo_v2_catalogo_publicado'::regclass AND conname = 'motivo_v2_catalogo_referencia_completa_unica'") == true ]]
[[ $(psql_valor \
  "SELECT (pg_catalog.to_regclass('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1') IS NULL AND pg_catalog.to_regclass('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1') IS NULL)::text") == true ]]
docker exec --interactive "$contenedor" psql -X \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
ALTER TABLE vec_autorizacion.motivo_v2_catalogo_publicado
  DROP CONSTRAINT motivo_v2_catalogo_referencia_completa_unica;
SQL
psql_archivo \
  deploy/postgresql/autorizacion/migraciones/000008_vinculaciones_motivo_consultas_rrhh.up.sql

# El down tampoco retira la UNIQUE si pierde su marca de procedencia.
docker exec --interactive "$contenedor" psql -X \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
COMMENT ON CONSTRAINT motivo_v2_catalogo_referencia_completa_unica
  ON vec_autorizacion.motivo_v2_catalogo_publicado IS NULL;
SQL
exigir_fallo_down 'UNIQUE sin procedencia 000008'
[[ $(psql_valor \
  "SELECT (pg_catalog.to_regclass('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1') IS NOT NULL AND pg_catalog.obj_description(oid, 'pg_constraint') IS NULL)::text FROM pg_catalog.pg_constraint WHERE conrelid = 'vec_autorizacion.motivo_v2_catalogo_publicado'::regclass AND conname = 'motivo_v2_catalogo_referencia_completa_unica'") == true ]]
docker exec --interactive "$contenedor" psql -X \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
SET ROLE vec_autorizacion_propietario;
COMMENT ON CONSTRAINT motivo_v2_catalogo_referencia_completa_unica
  ON vec_autorizacion.motivo_v2_catalogo_publicado IS
  'vec_autorizacion:vinculacion-motivo-consulta-rrhh:referencia-completa:v1:000008';
SQL

# Sin evidencia, la retirada es completa y permite instalar de nuevo.
psql_archivo \
  deploy/postgresql/autorizacion/migraciones/000008_vinculaciones_motivo_consultas_rrhh.down.sql
[[ $(psql_valor \
  "SELECT (to_regclass('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1') IS NULL)::text") == true ]]
[[ $(psql_valor \
  "SELECT (to_regclass('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1') IS NULL)::text") == true ]]
psql_archivo \
  deploy/postgresql/autorizacion/migraciones/000008_vinculaciones_motivo_consultas_rrhh.up.sql

docker exec --interactive --env CLAVE="$clave" "$contenedor" \
  psql -X --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
\getenv clave CLAVE
CREATE ROLE vec_motivos_rrhh_login_ajeno LOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS PASSWORD :'clave';
CREATE ROLE vec_motivos_rrhh_login_proyector LOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS PASSWORD :'clave';
GRANT CONNECT ON DATABASE vec_autorizacion_motivos_rrhh_000008
  TO vec_motivos_rrhh_login_ajeno, vec_motivos_rrhh_login_proyector;
GRANT vec_autorizacion_motivos_proyector
  TO vec_motivos_rrhh_login_proyector
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL

for usuario in vec_motivos_rrhh_login_ajeno vec_motivos_rrhh_login_proyector
do
  for consulta in \
    'SELECT * FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1' \
    "INSERT INTO vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1(clase_consulta,publicacion_version,publicacion_ref,publicacion_huella_sha256,catalogo_id,catalogo_version,catalogo_huella_sha256,entrada_clave,publicada_en) VALUES ('cuadro',1,'publicacion_motivo_rrhh_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',repeat('a',64),'x',1,repeat('b',64),'motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',clock_timestamp())" \
    'SET ROLE vec_autorizacion_propietario'
  do
    if docker exec --env PGPASSWORD="$clave" "$contenedor" \
      psql -X --set ON_ERROR_STOP=1 -h 127.0.0.1 \
        -U "$usuario" -d "$base" -c "$consulta" >/dev/null 2>&1; then
      echo "$usuario obtuvo una capacidad privada: $consulta" >&2
      exit 1
    fi
  done
done

psql_archivo \
  deploy/postgresql/autorizacion/pruebas_sql/vinculaciones_motivo_consultas_rrhh_000008.sql

filas_antes=$(psql_valor \
  "SELECT count(*) FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1")
if psql_archivo \
  deploy/postgresql/autorizacion/migraciones/000008_vinculaciones_motivo_consultas_rrhh.down.sql \
  >/dev/null 2>&1; then
  echo '000008 down eliminó evidencia durable' >&2
  exit 1
fi
[[ $(psql_valor \
  "SELECT count(*) FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1") == "$filas_antes" ]]
[[ $(psql_valor \
  "SELECT count(*) FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1 WHERE ultima_publicacion_version=1") == 2 ]]

echo 'OK: fundamento privado de motivos RRHH 000008 en PostgreSQL 18.4'
