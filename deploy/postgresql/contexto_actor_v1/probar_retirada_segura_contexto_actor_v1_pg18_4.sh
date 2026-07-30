#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-contexto-actor-safe-down-pg-${USER:-usuario}-$$"
base_control=ct95_control
clave_admin=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
down=deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.down.sql
up=deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql
down_000002=deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.down.sql
up_000002=deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql
down_roles=deploy/postgresql/contexto_actor_v1/roles_down.sql

limpiar() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM
esperar_postgres() {
  local base=$1 consecutivas=0 respuesta
  for _ in $(seq 1 200); do
    if respuesta=$(docker exec "$contenedor" psql -XAt \
      --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
      --command \
      "SELECT current_setting('server_version_num') || '|' ||
              pg_catalog.pg_is_in_recovery()" 2>/dev/null) &&
      [[ $respuesta == '180004|false' ]]; then
      consecutivas=$((consecutivas + 1))
      [[ $consecutivas -eq 3 ]] && return 0
    else
      consecutivas=0
    fi
    sleep 0.05
  done
  echo "PostgreSQL 18.4 primario no quedó disponible para $base" >&2
  return 1
}
docker run --detach --rm --name "$contenedor" \
  --env POSTGRES_DB="$base_control" \
  --env POSTGRES_PASSWORD="$clave_admin" \
  "$imagen" >/dev/null
esperar_postgres "$base_control"

version=$(docker exec "$contenedor" psql -XAt \
  --set ON_ERROR_STOP=1 --username postgres --dbname "$base_control" \
  --command "SELECT current_setting('server_version_num')")
[[ $version == 180004 ]] || {
  echo "se requiere PostgreSQL 18.4 exacto; se obtuvo $version" >&2
  exit 1
}
recuperacion=$(docker exec "$contenedor" psql -XAt \
  --set ON_ERROR_STOP=1 --username postgres --dbname "$base_control" \
  --command 'SELECT pg_catalog.pg_is_in_recovery()')
[[ $recuperacion == f ]] || {
  echo 'la retirada focal requiere un primario real' >&2
  exit 1
}

psql_archivo() {
  local base=$1 archivo=$2
  shift 2
  docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" "$@" \
    < "$raiz/$archivo"
}
retirar_roles() {
  local base=$1 usuario=${2:-postgres}
  docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 \
    --set confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1 \
    --username "$usuario" --dbname "$base" < "$raiz/$down_roles"
}

rechazar_retirada_roles() {
  local base=$1 descripcion=$2 estado
  set +e
  retirar_roles "$base" >/dev/null 2>&1
  estado=$?
  set -e
  if [[ $estado -eq 0 ]]; then
    echo "retirada de roles aceptada indebidamente: $descripcion" >&2
    exit 1
  fi
}
psql_admin() {
  local base=$1
  docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base"
}
consulta() {
  local base=$1 sql=$2 usuario=${3:-postgres}
  docker exec "$contenedor" psql -XAt --set ON_ERROR_STOP=1 \
    --username "$usuario" --dbname "$base" --command "$sql"
}

huella_esquema() {
  local base=$1
  docker exec "$contenedor" pg_dump -s --username postgres --dbname "$base" |
    sed -E '/^\\(un)?restrict /d' |
    sha256sum | cut -d' ' -f1
}

preparar_base() {
  local base=$1
  docker exec "$contenedor" createdb -U postgres "$base"
  psql_admin "$base" <<'SQL'
DO $base$
BEGIN
  EXECUTE pg_catalog.format(
    'REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',
    pg_catalog.current_database()
  );
  EXECUTE pg_catalog.format(
    'GRANT CONNECT ON DATABASE %I TO vec_contexto_actor_v1_propietario, vec_contexto_actor_v1_migrador, vec_contexto_actor_v1_runtime',
    pg_catalog.current_database()
  );
END
$base$;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
}

crear_base() {
  local base=$1
  preparar_base "$base"
  psql_archivo "$base" "$up"
}

esperar_fallo_sin_cambio() {
  local base=$1 descripcion=$2 modo=${3:-correcta}
  local antes despues salida estado
  antes=$(huella_esquema "$base")
  set +e
  case "$modo" in
    ausente)
      salida=$(psql_archivo "$base" "$down" 2>&1)
      estado=$?
      ;;
    incorrecta)
      salida=$(psql_archivo "$base" "$down" \
        --set confirmar_destruccion_contexto_actor_v1=NO_AUTORIZADA 2>&1)
      estado=$?
      ;;
    correcta)
      salida=$(psql_archivo "$base" "$down" \
        --set confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1 2>&1)
      estado=$?
      ;;
    *)
      echo "modo de prueba desconocido: $modo" >&2
      exit 1
      ;;
  esac
  set -e
  if [[ $estado -eq 0 ]]; then
    echo "retirada aceptada indebidamente: $descripcion" >&2
    exit 1
  fi
  despues=$(huella_esquema "$base")
  if [[ $antes != "$despues" ]]; then
    echo "rollback incompleto tras rechazo: $descripcion" >&2
    echo "$salida" >&2
    exit 1
  fi
}

# Los roles se crean una sola vez en el clúster efímero. Las demás bases
# reciben la misma ACL explícita que el bootstrap acredita.
psql_admin "$base_control" <<'SQL'
DO $base$
BEGIN
  EXECUTE pg_catalog.format(
    'REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',
    pg_catalog.current_database()
  );
END
$base$;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
psql_archivo "$base_control" deploy/postgresql/contexto_actor_v1/roles_up.sql
psql_admin "$base_control" <<'SQL'
DO $base$
BEGIN
  EXECUTE pg_catalog.format(
    'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_contexto_actor_v1_propietario, vec_contexto_actor_v1_migrador, vec_contexto_actor_v1_runtime',
    pg_catalog.current_database()
  );
END
$base$;
SQL

base_exacta=ct95_exacta
preparar_base "$base_exacta"
psql_admin "$base_exacta" <<'SQL'
CREATE PUBLICATION ct113_global_previa
  FOR ALL TABLES
  WITH (publish = 'insert, truncate', publish_via_partition_root = true);
SQL
psql_archivo "$base_exacta" "$up"
esperar_fallo_sin_cambio "$base_exacta" \
  'publicación global anterior al módulo'
[[ $(consulta "$base_exacta" \
  "SELECT puballtables AND pubinsert AND pubtruncate
          AND pubviaroot AND NOT pubupdate AND NOT pubdelete
     FROM pg_catalog.pg_publication
    WHERE pubname='ct113_global_previa'") == t ]]
psql_admin "$base_exacta" <<'SQL'
DROP PUBLICATION ct113_global_previa;
SQL
esperar_fallo_sin_cambio "$base_exacta" \
  'confirmación ausente' ausente
esperar_fallo_sin_cambio "$base_exacta" \
  'confirmación incorrecta' incorrecta

# Un objeto posterior, incluso vacío, no pertenece al manifiesto de 000001.
psql_admin "$base_exacta" <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_contexto_actor_v1.consumidor_futuro_v1()
RETURNS integer LANGUAGE sql BEGIN ATOMIC SELECT 1; END;
COMMIT;
SQL
esperar_fallo_sin_cambio "$base_exacta" 'función futura'
psql_admin "$base_exacta" <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
DROP FUNCTION vec_contexto_actor_v1.consumidor_futuro_v1();
RESET ROLE;
SQL

# Toda concesión externa y toda ACL predeterminada añadida fallan cerradas.
psql_admin "$base_exacta" <<'SQL'
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO PUBLIC;
SQL
esperar_fallo_sin_cambio "$base_exacta" 'ACL de esquema para PUBLIC'
psql_admin "$base_exacta" <<'SQL'
REVOKE USAGE ON SCHEMA vec_contexto_actor_v1 FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_contexto_actor_v1_propietario
  GRANT SELECT ON TABLES TO PUBLIC;
SQL
esperar_fallo_sin_cambio "$base_exacta" 'ACL predeterminada hostil'
psql_admin "$base_exacta" <<'SQL'
ALTER DEFAULT PRIVILEGES FOR ROLE vec_contexto_actor_v1_propietario
  REVOKE SELECT ON TABLES FROM PUBLIC;
SQL

# Las ACL de columna y sus propiedades físicas forman parte del contrato; no
# basta con acreditar la relación que las contiene.
psql_admin "$base_exacta" <<'SQL'
CREATE ROLE ct105_lector_columna NOLOGIN;
GRANT SELECT (procedencia_ref)
  ON vec_contexto_actor_v1.procedencias
  TO ct105_lector_columna;
SQL
esperar_fallo_sin_cambio "$base_exacta" 'ACL de columna hostil'
[[ $(consulta "$base_exacta" \
  "SELECT pg_catalog.has_column_privilege(
     'ct105_lector_columna',
     'vec_contexto_actor_v1.procedencias',
     'procedencia_ref','SELECT')") == t ]]
psql_admin "$base_exacta" <<'SQL'
REVOKE SELECT (procedencia_ref)
  ON vec_contexto_actor_v1.procedencias
  FROM ct105_lector_columna;
DROP ROLE ct105_lector_columna;
ALTER TABLE vec_contexto_actor_v1.procedencias
  ALTER COLUMN procedencia_ref SET STATISTICS 777;
SQL
esperar_fallo_sin_cambio "$base_exacta" 'estadística de columna hostil'
[[ $(consulta "$base_exacta" \
  "SELECT attstattarget
     FROM pg_catalog.pg_attribute
    WHERE attrelid='vec_contexto_actor_v1.procedencias'::regclass
      AND attname='procedencia_ref'") == 777 ]]
psql_admin "$base_exacta" <<'SQL'
ALTER TABLE vec_contexto_actor_v1.procedencias
  ALTER COLUMN procedencia_ref SET STATISTICS -1;
SQL

# Tanto los triggers propios como los internos de las FK se acreditan por su
# definición semántica y estado exactos, sin depender del OID de su nombre.
psql_admin "$base_exacta" <<'SQL'
ALTER TABLE vec_contexto_actor_v1.procedencias
  DISABLE TRIGGER procedencia_monotona;
SQL
esperar_fallo_sin_cambio "$base_exacta" 'trigger deshabilitado'
psql_admin "$base_exacta" <<'SQL'
ALTER TABLE vec_contexto_actor_v1.procedencias
  ENABLE TRIGGER procedencia_monotona;
DO $deshabilitar_fk$
DECLARE nombre text;
BEGIN
  SELECT t.tgname INTO STRICT nombre
    FROM pg_catalog.pg_trigger AS t
   WHERE t.tgisinternal
     AND t.tgrelid='vec_contexto_actor_v1.perfil_actual'::regclass
   ORDER BY t.oid
   LIMIT 1;
  EXECUTE pg_catalog.format(
    'ALTER TABLE vec_contexto_actor_v1.perfil_actual DISABLE TRIGGER %I',
    nombre
  );
END
$deshabilitar_fk$;
SQL
esperar_fallo_sin_cambio "$base_exacta" 'trigger FK interno deshabilitado'
[[ $(consulta "$base_exacta" \
  "SELECT count(*)
     FROM pg_catalog.pg_trigger
    WHERE tgisinternal
      AND tgrelid='vec_contexto_actor_v1.perfil_actual'::regclass
      AND tgenabled='D'") == 1 ]]
psql_admin "$base_exacta" <<'SQL'
DO $habilitar_fk$
DECLARE nombre text;
BEGIN
  SELECT t.tgname INTO STRICT nombre
    FROM pg_catalog.pg_trigger AS t
   WHERE t.tgisinternal
     AND t.tgrelid='vec_contexto_actor_v1.perfil_actual'::regclass
     AND t.tgenabled='D'
   ORDER BY t.oid
   LIMIT 1;
  EXECUTE pg_catalog.format(
    'ALTER TABLE vec_contexto_actor_v1.perfil_actual ENABLE TRIGGER %I',
    nombre
  );
END
$habilitar_fk$;
SQL

# DROP TABLE eliminaría silenciosamente una asociación de publicación. El
# manifiesto debe detectarla antes de ejecutar un solo DROP.
psql_admin "$base_exacta" <<'SQL'
CREATE PUBLICATION ct105_publicacion_externa
  FOR TABLE vec_contexto_actor_v1.procedencias;
SQL
esperar_fallo_sin_cambio "$base_exacta" 'publicación externa'
[[ $(consulta "$base_exacta" \
  "SELECT count(*)
     FROM pg_catalog.pg_publication_rel AS pr
     JOIN pg_catalog.pg_publication AS p ON p.oid=pr.prpubid
    WHERE p.pubname='ct105_publicacion_externa'
      AND pr.prrelid='vec_contexto_actor_v1.procedencias'::regclass") == 1 ]]
psql_admin "$base_exacta" <<'SQL'
DROP PUBLICATION ct105_publicacion_externa;
CREATE PUBLICATION ct113_global_posterior
  FOR ALL TABLES
  WITH (publish = 'update, delete', publish_via_partition_root = false);
SQL
esperar_fallo_sin_cambio "$base_exacta" \
  'publicación global posterior al módulo'
[[ $(consulta "$base_exacta" \
  "SELECT puballtables AND pubupdate AND pubdelete
          AND NOT pubinsert AND NOT pubtruncate AND NOT pubviaroot
     FROM pg_catalog.pg_publication
    WHERE pubname='ct113_global_posterior'") == t ]]
psql_admin "$base_exacta" <<'SQL'
DROP PUBLICATION ct113_global_posterior;
ALTER ROLE vec_contexto_actor_v1_runtime CONNECTION LIMIT 7;
SQL
esperar_fallo_sin_cambio "$base_exacta" 'límite de conexiones hostil'
[[ $(consulta "$base_exacta" \
  "SELECT rolconnlimit
     FROM pg_catalog.pg_roles
    WHERE rolname='vec_contexto_actor_v1_runtime'") == 7 ]]
psql_admin "$base_exacta" <<'SQL'
ALTER ROLE vec_contexto_actor_v1_runtime CONNECTION LIMIT -1;
ALTER ROLE vec_contexto_actor_v1_runtime
  IN DATABASE ct95_exacta SET application_name = 'ct113_sintetica';
SQL
esperar_fallo_sin_cambio "$base_exacta" \
  'ajuste del rol específico para la base'
[[ $(consulta "$base_exacta" \
  "SELECT setconfig @> ARRAY['application_name=ct113_sintetica']
     FROM pg_catalog.pg_db_role_setting
    WHERE setrole='vec_contexto_actor_v1_runtime'::regrole
      AND setdatabase=(
          SELECT oid FROM pg_catalog.pg_database WHERE datname='ct95_exacta'
      )") == t ]]
psql_admin "$base_exacta" <<'SQL'
ALTER ROLE vec_contexto_actor_v1_runtime
  IN DATABASE ct95_exacta RESET application_name;
ALTER ROLE ALL IN DATABASE ct95_exacta
  SET statement_timeout = '7s';
SQL
esperar_fallo_sin_cambio "$base_exacta" \
  'ajuste de todos los roles para la base'
[[ $(consulta "$base_exacta" \
  "SELECT setconfig @> ARRAY['statement_timeout=7s']
     FROM pg_catalog.pg_db_role_setting
    WHERE setrole=0
      AND setdatabase=(
          SELECT oid FROM pg_catalog.pg_database WHERE datname='ct95_exacta'
      )") == t ]]
psql_admin "$base_exacta" <<'SQL'
ALTER ROLE ALL IN DATABASE ct95_exacta RESET statement_timeout;
SQL

# La huella de dependencias compartidas rechaza propiedad y ACL ajenas antes
# del primer DROP, incluso si viven en otra base del clúster.
psql_admin "$base_exacta" <<'SQL'
CREATE SCHEMA ct113_externo AUTHORIZATION postgres;
CREATE TABLE ct113_externo.recurso (id integer);
GRANT SELECT ON ct113_externo.recurso
  TO vec_contexto_actor_v1_runtime;
SQL
esperar_fallo_sin_cambio "$base_exacta" 'ACL sobre objeto externo'
[[ $(consulta "$base_exacta" \
  "SELECT pg_catalog.has_table_privilege(
      'vec_contexto_actor_v1_runtime',
      'ct113_externo.recurso','SELECT')") == t ]]
psql_admin "$base_exacta" <<'SQL'
REVOKE SELECT ON ct113_externo.recurso
  FROM vec_contexto_actor_v1_runtime;
DROP TABLE ct113_externo.recurso;
DROP SCHEMA ct113_externo RESTRICT;
SQL

base_externa=ct113_externa
docker exec "$contenedor" createdb -U postgres "$base_externa"
psql_admin "$base_externa" <<'SQL'
CREATE SCHEMA ct113_propiedad_externa
  AUTHORIZATION vec_contexto_actor_v1_runtime;
SQL
esperar_fallo_sin_cambio "$base_exacta" \
  'propiedad del rol en otra base'
[[ $(consulta "$base_externa" \
  "SELECT pg_catalog.pg_get_userbyid(nspowner)
     FROM pg_catalog.pg_namespace
    WHERE nspname='ct113_propiedad_externa'") == \
  vec_contexto_actor_v1_runtime ]]
docker exec "$contenedor" dropdb --force \
  --username postgres "$base_externa"

# Un consumidor exterior no aparece en el esquema, pero RESTRICT conserva la
# dependencia y PostgreSQL revierte también los DROP anteriores.
psql_admin "$base_exacta" <<'SQL'
CREATE SCHEMA ct95_consumidor AUTHORIZATION postgres;
CREATE VIEW ct95_consumidor.procedencias AS
  SELECT procedencia_ref
    FROM vec_contexto_actor_v1.procedencias;
SQL
esperar_fallo_sin_cambio "$base_exacta" 'dependencia exterior real'
psql_admin "$base_exacta" <<'SQL'
DROP VIEW ct95_consumidor.procedencias;
DROP SCHEMA ct95_consumidor RESTRICT;
SQL

# Una instalación 000001 exacta y vacía sí puede retirarse. Tras reconectar y
# reiniciar, el ciclo up -> down -> up produce un OID nuevo.
oid_antes=$(consulta "$base_exacta" \
  "SELECT 'vec_contexto_actor_v1'::regnamespace::oid")
docker restart "$contenedor" >/dev/null
esperar_postgres "$base_exacta"
psql_archivo "$base_exacta" "$down" \
  --set confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1
[[ $(consulta "$base_exacta" \
  "SELECT pg_catalog.to_regnamespace('vec_contexto_actor_v1') IS NULL") == t ]]
set +e
psql_archivo "$base_exacta" "$down" \
  --set confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1 \
  >/dev/null 2>&1
reentrada=$?
set -e
[[ $reentrada -ne 0 ]] || {
  echo 'la retirada ausente se presentó como éxito' >&2
  exit 1
}
psql_archivo "$base_exacta" "$up"
oid_despues=$(consulta "$base_exacta" \
  "SELECT 'vec_contexto_actor_v1'::regnamespace::oid")
[[ $oid_antes != "$oid_despues" ]] || {
  echo 'up -> down -> up conservó el OID del esquema' >&2
  exit 1
}
docker exec "$contenedor" dropdb --force \
  --username postgres "$base_exacta"

# Cualquier evidencia bloquea la retirada aun con el literal correcto.
base_datos=ct95_datos
crear_base "$base_datos"
psql_admin "$base_datos" <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.procedencias VALUES (
  'prc_evidencia_sintetica_ct95_000001',
  1, repeat('a', 64), 'no_autoritativa'
);
RESET ROLE;
SQL
esperar_fallo_sin_cambio "$base_datos" 'evidencia base'
[[ $(consulta "$base_datos" \
  'SELECT count(*) FROM vec_contexto_actor_v1.procedencias') == 1 ]]
docker exec "$contenedor" dropdb --force \
  --username postgres "$base_datos"

# 000002 aporta objetos y su fila de generación: debe retirarse primero por su
# propio down seguro, nunca arrastrarse desde la base.
base_000002=ct95_000002
crear_base "$base_000002"
psql_archivo "$base_000002" "$up_000002"
esperar_fallo_sin_cambio "$base_000002" 'migración 000002 instalada'
psql_archivo "$base_000002" "$down_000002" \
  --set confirmar_retirada_acreditacion_contexto_actor_v2=RETIRAR_ACREDITACION_CONTEXTO_ACTOR_V2
psql_archivo "$base_000002" "$down" \
  --set confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1
docker exec "$contenedor" dropdb --force \
  --username postgres "$base_000002"

# Un propietario alterado se rechaza y no se repara dentro del down.
base_propietario=ct95_propietario
crear_base "$base_propietario"
psql_admin "$base_propietario" <<'SQL'
CREATE ROLE ct95_propietario_hostil NOLOGIN;
ALTER FUNCTION vec_contexto_actor_v1.instante_valido(timestamptz)
  OWNER TO ct95_propietario_hostil;
SQL
esperar_fallo_sin_cambio "$base_propietario" 'propietario hostil'
docker exec "$contenedor" dropdb --force \
  --username postgres "$base_propietario"

# Carrera determinista: se observa el advisory no concedido antes de liberar
# al bloqueador. pg_sleep solo estaciona la sesión; no acredita el resultado.
base_carrera=ct95_carrera
crear_base "$base_carrera"
docker exec --env PGAPPNAME=ct95_bloqueador "$contenedor" \
  psql -Xq --set ON_ERROR_STOP=1 --username postgres \
  --dbname "$base_carrera" --command \
  "BEGIN;
   SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
     'vec_contexto_actor_v1:migracion:base:v1',0));
   SELECT pg_catalog.pg_sleep(300);
   COMMIT" >/dev/null 2>&1 &
proceso_bloqueador=$!
for _ in $(seq 1 200); do
  pid_bloqueador=$(consulta "$base_carrera" \
    "SELECT pid FROM pg_catalog.pg_stat_activity
      WHERE application_name='ct95_bloqueador'
        AND state='active'
        AND query LIKE '%pg_sleep(300)%'
      LIMIT 1")
  [[ -n $pid_bloqueador ]] && break
  sleep 0.02
done
[[ -n ${pid_bloqueador:-} ]] || {
  echo 'no se observó el bloqueador de la carrera' >&2
  exit 1
}

docker exec --env PGAPPNAME=ct95_retirada --interactive "$contenedor" \
  psql -Xq --set ON_ERROR_STOP=1 \
  --set confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1 \
  --username postgres --dbname "$base_carrera" \
  < "$raiz/$down" >/dev/null 2>&1 &
proceso_retirada=$!
for _ in $(seq 1 200); do
  espera=$(consulta "$base_carrera" \
    "SELECT count(*) FROM pg_catalog.pg_stat_activity
      WHERE application_name='ct95_retirada'
        AND wait_event_type='Lock' AND wait_event='advisory'")
  [[ $espera == 1 ]] && break
  sleep 0.02
done
[[ ${espera:-0} == 1 ]] || {
  echo 'la retirada concurrente no esperó la guarda base' >&2
  exit 1
}
[[ $(consulta "$base_carrera" \
  "SELECT pg_catalog.pg_terminate_backend($pid_bloqueador)") == t ]]
wait "$proceso_bloqueador" 2>/dev/null || true
wait "$proceso_retirada"
[[ $(consulta "$base_carrera" \
  "SELECT pg_catalog.to_regnamespace('vec_contexto_actor_v1') IS NULL") == t ]]
docker exec "$contenedor" dropdb --force \
  --username postgres "$base_carrera"

# El down de roles solo puede retirar los tres nombres exactos. Se borran las
# bases efímeras para liberar sus dependencias y se conserva un rol derivado
# deliberadamente parecido.
psql_admin "$base_control" <<'SQL'
DO $base$
BEGIN
  EXECUTE pg_catalog.format(
    'GRANT CONNECT ON DATABASE %I TO vec_contexto_actor_v1_propietario, vec_contexto_actor_v1_migrador, vec_contexto_actor_v1_runtime',
    pg_catalog.current_database()
  );
END
$base$;
CREATE ROLE vec_contexto_actor_v1_runtime_derivado NOLOGIN
  NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT
  NOREPLICATION NOBYPASSRLS CONNECTION LIMIT 7;
SQL

# La invocación desnuda debe terminar con estado no cero sin abrir una
# transacción destructiva.
set +e
psql_archivo "$base_control" \
  deploy/postgresql/contexto_actor_v1/roles_down.sql >/dev/null 2>&1
estado_roles=$?
set -e
[[ $estado_roles -ne 0 ]]
[[ $(consulta "$base_control" \
  "SELECT count(*) FROM pg_catalog.pg_roles
    WHERE rolname LIKE 'vec_contexto_actor_v1_%'
      AND rolname <> 'vec_contexto_actor_v1_runtime_derivado'") == 3 ]]

# Los ajustes por base y los atributos hostiles se revalidan dentro de la
# misma transacción que ejecutaría DROP ROLE.
psql_admin "$base_control" <<'SQL'
ALTER ROLE ALL IN DATABASE ct95_control
  SET statement_timeout = '11s';
SQL
rechazar_retirada_roles "$base_control" \
  'ajuste común específico de base'
[[ $(consulta "$base_control" \
  "SELECT setconfig @> ARRAY['statement_timeout=11s']
     FROM pg_catalog.pg_db_role_setting
    WHERE setrole=0
      AND setdatabase=(
          SELECT oid FROM pg_catalog.pg_database
           WHERE datname=current_database()
      )") == t ]]
psql_admin "$base_control" <<'SQL'
ALTER ROLE ALL IN DATABASE ct95_control RESET statement_timeout;
ALTER ROLE vec_contexto_actor_v1_runtime CONNECTION LIMIT 13;
SQL
rechazar_retirada_roles "$base_control" \
  'atributo hostil del rol runtime'
[[ $(consulta "$base_control" \
  "SELECT rolconnlimit=13 FROM pg_catalog.pg_roles
    WHERE rolname='vec_contexto_actor_v1_runtime'") == t ]]
psql_admin "$base_control" <<'SQL'
ALTER ROLE vec_contexto_actor_v1_runtime CONNECTION LIMIT -1;
SQL

# Simula una dependencia nacida entre el down de base y el de roles. La
# revalidación autónoma la conserva y no revoca siquiera el CONNECT esperado.
base_roles_externa=ct113_roles_externa
docker exec "$contenedor" createdb -U postgres "$base_roles_externa"
psql_admin "$base_roles_externa" <<'SQL'
CREATE SCHEMA ct113_roles_propiedad
  AUTHORIZATION vec_contexto_actor_v1_runtime;
SQL
rechazar_retirada_roles "$base_control" \
  'propiedad exterior nacida después del down base'
[[ $(consulta "$base_roles_externa" \
  "SELECT pg_catalog.pg_get_userbyid(nspowner)
     FROM pg_catalog.pg_namespace
    WHERE nspname='ct113_roles_propiedad'") == \
  vec_contexto_actor_v1_runtime ]]
[[ $(consulta "$base_control" \
  "SELECT pg_catalog.has_database_privilege(
      'vec_contexto_actor_v1_runtime',
      current_database(),'CONNECT')") == t ]]
docker exec "$contenedor" dropdb --force \
  --username postgres "$base_roles_externa"

psql_admin "$base_control" <<'SQL'
CREATE ROLE ct117_superusuario_retirada LOGIN SUPERUSER;
SQL

# Carrera final: roles_down adquiere primero pg_authid y queda detenido al
# pedir pg_shseclabel. Un ALTER ROLE posterior debe esperar sus bloqueos y no
# puede colarse entre la revalidación y DROP ROLE.
docker exec --env PGAPPNAME=ct113_bloqueo_catalogo "$contenedor" \
  psql -Xq --set ON_ERROR_STOP=1 --username postgres \
  --dbname "$base_control" --command \
  "BEGIN;
   LOCK TABLE pg_catalog.pg_shseclabel IN ACCESS EXCLUSIVE MODE;
   SELECT pg_catalog.pg_sleep(300);
   COMMIT" >/dev/null 2>&1 &
proceso_bloqueo_catalogo=$!
for _ in $(seq 1 200); do
  pid_bloqueo_catalogo=$(consulta "$base_control" \
    "SELECT pid FROM pg_catalog.pg_stat_activity
      WHERE application_name='ct113_bloqueo_catalogo'
        AND state='active' AND query LIKE '%pg_sleep(300)%'
      LIMIT 1")
  [[ -n $pid_bloqueo_catalogo ]] && break
  sleep 0.02
done
[[ -n ${pid_bloqueo_catalogo:-} ]]

docker exec --env PGAPPNAME=ct113_retirada_roles \
  --interactive "$contenedor" psql -Xq --set ON_ERROR_STOP=1 \
  --set confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1 \
  --username ct117_superusuario_retirada --dbname "$base_control" \
  < "$raiz/deploy/postgresql/contexto_actor_v1/roles_down.sql" \
  >/dev/null 2>&1 &
proceso_retirada_roles=$!
for _ in $(seq 1 200); do
  espera_roles=$(consulta "$base_control" \
    "SELECT count(*) FROM pg_catalog.pg_stat_activity
      WHERE application_name='ct113_retirada_roles'
        AND wait_event_type='Lock'")
  [[ $espera_roles == 1 ]] && break
  sleep 0.02
done
[[ ${espera_roles:-0} == 1 ]]

docker exec --env PGAPPNAME=ct113_mutador_roles "$contenedor" \
  psql -Xq --set ON_ERROR_STOP=1 --username postgres \
  --dbname "$base_control" --command \
  'ALTER ROLE vec_contexto_actor_v1_runtime CONNECTION LIMIT 17' \
  >/dev/null 2>&1 &
proceso_mutador_roles=$!
for _ in $(seq 1 200); do
  espera_mutador=$(consulta "$base_control" \
    "SELECT count(*) FROM pg_catalog.pg_stat_activity
      WHERE application_name='ct113_mutador_roles'
        AND wait_event_type='Lock'")
  [[ $espera_mutador == 1 ]] && break
  sleep 0.02
done
[[ ${espera_mutador:-0} == 1 ]]
[[ $(consulta "$base_control" \
  "SELECT pg_catalog.pg_terminate_backend($pid_bloqueo_catalogo)") == t ]]
wait "$proceso_bloqueo_catalogo" 2>/dev/null || true
wait "$proceso_retirada_roles"
set +e
wait "$proceso_mutador_roles"
estado_mutador=$?
set -e
[[ $estado_mutador -ne 0 ]]

[[ $(consulta "$base_control" \
  "SELECT count(*) FROM pg_catalog.pg_roles
    WHERE rolname IN (
      'vec_contexto_actor_v1_propietario',
      'vec_contexto_actor_v1_migrador',
      'vec_contexto_actor_v1_runtime'
    )") == 0 ]]
[[ $(consulta "$base_control" \
  "SELECT count(*) FROM pg_catalog.pg_roles
    WHERE rolname='vec_contexto_actor_v1_runtime_derivado'
      AND rolconnlimit=7") == 1 ]]
psql_admin "$base_control" <<'SQL'
DROP ROLE vec_contexto_actor_v1_runtime_derivado;
DROP ROLE ct95_propietario_hostil;
DROP ROLE ct117_superusuario_retirada;
SQL

if rg -n --ignore-case \
  'drop[[:space:]]+owned|drop[[:space:]]+schema[^;]*cascade' "$down"; then
  echo 'la retirada contiene una primitiva destructiva prohibida' >&2
  exit 1
fi

echo 'ContextoActor V1: retirada base segura PostgreSQL 18.4 superada'
