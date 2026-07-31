#!/usr/bin/env bash
set -euo pipefail
export LC_ALL=C TZ=UTC

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-vinculo-corporativo-${USER:-usuario}-$$"
base=c22b_vinculo_corporativo
clave=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
directorio=deploy/postgresql/contexto_actor_v1
roles="$directorio/roles_up.sql"
roles_selector="$directorio/roles_contexto_corporativo_rrhh_selector_v1_up.sql"
up_1="$directorio/migraciones/000001_contexto_actor_v1.up.sql"
up_2="$directorio/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql"
down_2="$directorio/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.down.sql"
up_3="$directorio/migraciones/000003_organizacion_corporativa_v1.up.sql"
down_3="$directorio/migraciones/000003_organizacion_corporativa_v1.down.sql"
up="$directorio/migraciones/000004_vinculo_corporativo_rrhh_v1.up.sql"
down="$directorio/migraciones/000004_vinculo_corporativo_rrhh_v1.down.sql"
estructura="$directorio/pruebas_sql/vinculo_corporativo_rrhh_v1_estructura_catalogal.sql"
integridad="$directorio/pruebas_sql/vinculo_corporativo_rrhh_v1_integridad_relacional.sql"
fixtures_base="$directorio/pruebas_sql/fixtures_sinteticos.sql"
temporales=()

limpiar() {
  local archivo
  for archivo in "${temporales[@]:-}"; do
    [[ ! -e $archivo ]] || rm -- "$archivo"
  done
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

fallar() {
  local archivo
  echo "$1" >&2
  for archivo in "${temporales[@]:-}"; do
    [[ ! -s $archivo ]] || { echo "diagnostico ${archivo##*/}:" >&2; tail -n 50 "$archivo" >&2; }
  done
  docker logs --tail 200 "$contenedor" >&2 2>/dev/null || true
  exit 1
}

esperar_postgres() {
  local consecutivas=0 respuesta
  for _ in $(seq 1 240); do
    if respuesta=$(docker exec --env PGPASSWORD="$clave" "$contenedor" \
      psql -XAt -h 127.0.0.1 -U postgres -d "$base" \
      -c "SELECT current_setting('server_version_num')||'|'||pg_is_in_recovery()" \
      2>/dev/null) && [[ $respuesta == '180004|false' ]]; then
      consecutivas=$((consecutivas + 1))
      [[ $consecutivas -eq 3 ]] && return 0
    else
      consecutivas=0
    fi
    sleep 0.25
  done
  fallar 'PostgreSQL 18.4 primario no quedo disponible tres veces consecutivas'
}

psql_archivo() {
  docker exec --interactive --env PGPASSWORD="$clave" "$contenedor" \
    psql -Xq -h 127.0.0.1 -U postgres -d "$base" -v ON_ERROR_STOP=1 \
    < "$raiz/$1"
}

psql_temporal() {
  docker exec --interactive --env PGPASSWORD="$clave" "$contenedor" \
    psql -Xq -h 127.0.0.1 -U postgres -d "$base" -v ON_ERROR_STOP=1 < "$1"
}

psql_sql() {
  docker exec --interactive --env PGPASSWORD="$clave" "$contenedor" \
    psql -Xq -h 127.0.0.1 -U postgres -d "$base" -v ON_ERROR_STOP=1
}

consulta() {
  docker exec --env PGPASSWORD="$clave" "$contenedor" \
    psql -XAt -h 127.0.0.1 -U postgres -d "$base" -v ON_ERROR_STOP=1 -c "$1"
}

retirar() {
  docker exec --interactive --env PGPASSWORD="$clave" \
    --env PGOPTIONS='-c vec.confirmar_retirada_vinculo_corporativo_rrhh_v1=RETIRAR_VINCULO_CORPORATIVO_RRHH_V1' \
    "$contenedor" psql -Xq -h 127.0.0.1 -U postgres -d "$base" \
    -v ON_ERROR_STOP=1 < "$raiz/$down"
}

retirar_mal() {
  docker exec --interactive --env PGPASSWORD="$clave" \
    --env PGOPTIONS='-c vec.confirmar_retirada_vinculo_corporativo_rrhh_v1=NO' \
    "$contenedor" psql -Xq -h 127.0.0.1 -U postgres -d "$base" \
    -v ON_ERROR_STOP=1 < "$raiz/$down"
}

retirar_000002() {
  docker exec --interactive --env PGPASSWORD="$clave" \
    --env PGOPTIONS='-c vec.confirmar_retirada_acreditacion_contexto_actor_v2=RETIRAR_ACREDITACION_CONTEXTO_ACTOR_V2' \
    "$contenedor" psql -Xq -h 127.0.0.1 -U postgres -d "$base" \
    -v ON_ERROR_STOP=1 < "$raiz/$down_2"
}

retirar_000003() {
  docker exec --interactive --env PGPASSWORD="$clave" \
    --env PGOPTIONS='-c vec.confirmar_retirada_organizacion_corporativa_v1=RETIRAR_ORGANIZACION_CORPORATIVA_V1' \
    "$contenedor" psql -Xq -h 127.0.0.1 -U postgres -d "$base" \
    -v ON_ERROR_STOP=1 < "$raiz/$down_3"
}

huella() {
  docker exec --env PGPASSWORD="$clave" "$contenedor" pg_dump \
    -h 127.0.0.1 -U postgres -d "$base" --format=plain |
    sed -E '/^\\(un)?restrict /d' | sha256sum | cut -d' ' -f1
}

huella_filas_base() {
  docker exec --env PGPASSWORD="$clave" "$contenedor" pg_dump \
    -h 127.0.0.1 -U postgres -d "$base" --data-only \
    --schema=vec_contexto_actor_v1 \
    --exclude-table=vec_contexto_actor_v1.vinculo_corporativo_versiones \
    --exclude-table=vec_contexto_actor_v1.vinculo_corporativo_actual |
    sed -E '/^\\(un)?restrict /d' | sha256sum | cut -d' ' -f1
}

exigir_fallo_intacto() {
  local descripcion=$1 funcion=$2 antes despues estado
  antes=$(huella)
  set +e
  "$funcion" >/dev/null 2>&1
  estado=$?
  set -e
  [[ $estado -ne 0 ]] || fallar "se acepto: $descripcion"
  despues=$(huella)
  [[ $antes == "$despues" ]] || fallar "el rechazo altero estado: $descripcion"
}

up_reentrada() { psql_archivo "$up"; }
down_sin_confirmacion() { psql_archivo "$down"; }

docker run -d --rm --name "$contenedor" --publish 127.0.0.1::5432 \
  -e POSTGRES_DB="$base" -e POSTGRES_PASSWORD="$clave" "$imagen" >/dev/null
esperar_postgres
psql_sql <<'SQL'
DO $base$
BEGIN
  CREATE ROLE c22b_dueno_base NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOINHERIT NOREPLICATION NOBYPASSRLS;
  EXECUTE format('ALTER DATABASE %I OWNER TO c22b_dueno_base',current_database());
  EXECUTE format('REVOKE ALL ON DATABASE %I FROM PUBLIC',current_database());
END
$base$;
REVOKE ALL ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
psql_archivo "$roles"
psql_archivo "$up_1"
psql_archivo "$up_2"
psql_archivo "$roles_selector"
psql_archivo "$up_3"
psql_sql <<'SQL'
CREATE ROLE c22b_consumidor NOLOGIN;
CREATE ROLE c22b_login LOGIN;
CREATE ROLE c22b_pdp NOLOGIN;
CREATE ROLE c22b_autorizacion NOLOGIN;
CREATE ROLE c22b_contratacion NOLOGIN;
CREATE ROLE c22b_bolsa NOLOGIN;
GRANT vec_contexto_actor_v1_runtime TO c22b_login
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL
huella_a=$(huella)

# Casos 1-3: atomicidad, instalacion literal y reentrada cerrada.
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
ALTER FUNCTION vec_contexto_actor_v1.organizacion_ref_valida(text) VOLATILE;
RESET ROLE;
SQL
exigir_fallo_intacto 'funcion predecesora hostil' up_reentrada
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
ALTER FUNCTION vec_contexto_actor_v1.organizacion_ref_valida(text) IMMUTABLE;
ALTER TABLE vec_contexto_actor_v1.perfil_versiones
  ALTER COLUMN persona_ref SET STORAGE MAIN;
RESET ROLE;
SQL
exigir_fallo_intacto 'almacenamiento de columna predecesora hostil' up_reentrada
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
ALTER TABLE vec_contexto_actor_v1.perfil_versiones
  ALTER COLUMN persona_ref SET STORAGE EXTENDED;
RESET ROLE;
SQL
[[ $(huella) == "$huella_a" ]] || fallar 'las regresiones predecesoras no restauraron A'
psql_sql <<'SQL'
CREATE FUNCTION public.c22b_fallo_tabla() RETURNS event_trigger LANGUAGE plpgsql AS $$
BEGIN
  IF tg_tag='CREATE TABLE' THEN RAISE EXCEPTION 'fallo atomico sintetico'; END IF;
END $$;
CREATE EVENT TRIGGER c22b_fallo_tabla ON ddl_command_start
  EXECUTE FUNCTION public.c22b_fallo_tabla();
SQL
exigir_fallo_intacto 'fallo atomico durante CREATE TABLE' up_reentrada
psql_sql <<'SQL'
DROP EVENT TRIGGER c22b_fallo_tabla;
DROP FUNCTION public.c22b_fallo_tabla();
SQL
[[ $(huella) == "$huella_a" ]] || fallar 'la prueba atomica no restauro A'
psql_archivo "$up"
exigir_fallo_intacto 'reentrada de 000004 up' up_reentrada

# Casos 4-5: forma catalogal y relaciones/datos adversariales focales.
psql_archivo "$estructura"
psql_archivo "$integridad"

# Casos 6-8: orden de migraciones y doble confirmacion de retirada.
exigir_fallo_intacto 'retirada 000002 con C2.2-B presente' retirar_000002
exigir_fallo_intacto 'retirada 000003 con C2.2-B presente' retirar_000003
exigir_fallo_intacto 'retirada B sin confirmacion' down_sin_confirmacion
exigir_fallo_intacto 'retirada B con confirmacion incorrecta' retirar_mal

# Casos 9-10: deriva catalogal hostil se rechaza y puede restaurarse.
psql_sql <<'SQL'
COMMENT ON TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
  IS 'metadato hostil sintetico';
SQL
exigir_fallo_intacto 'retirada con comentario hostil' retirar
psql_sql <<'SQL'
COMMENT ON TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones IS NULL;
CREATE INDEX c22b_indice_hostil
 ON vec_contexto_actor_v1.vinculo_corporativo_versiones(estado);
SQL
exigir_fallo_intacto 'retirada con indice hostil' retirar
psql_sql <<'SQL'
DROP INDEX vec_contexto_actor_v1.c22b_indice_hostil;
GRANT USAGE ON TYPE vec_contexto_actor_v1.vinculo_corporativo_versiones
  TO c22b_consumidor;
SQL
exigir_fallo_intacto 'retirada con ACL hostil de tipo' retirar
psql_sql <<'SQL'
REVOKE USAGE ON TYPE vec_contexto_actor_v1.vinculo_corporativo_versiones
  FROM c22b_consumidor;
GRANT SELECT (estado) ON vec_contexto_actor_v1.vinculo_corporativo_versiones
  TO c22b_consumidor;
SQL
exigir_fallo_intacto 'retirada con ACL hostil de columna' retirar
psql_sql <<'SQL'
REVOKE SELECT (estado) ON vec_contexto_actor_v1.vinculo_corporativo_versiones
  FROM c22b_consumidor;
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
  SET (toast.autovacuum_enabled=false);
SQL
exigir_fallo_intacto 'retirada con opcion TOAST hostil' retirar
docker exec --user root "$contenedor" mkdir -p /tmp/c22b_espacio_hostil
docker exec --user root "$contenedor" chown postgres:postgres /tmp/c22b_espacio_hostil
psql_sql <<'SQL'
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
  RESET (toast.autovacuum_enabled);
CREATE TABLESPACE c22b_espacio_hostil LOCATION '/tmp/c22b_espacio_hostil';
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
  SET TABLESPACE c22b_espacio_hostil;
SQL
exigir_fallo_intacto 'retirada con tablespace hostil' retirar
psql_sql <<'SQL'
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
  SET TABLESPACE pg_default;
DROP TABLESPACE c22b_espacio_hostil;
CREATE STATISTICS vec_contexto_actor_v1.c22b_estadistica_hostil
  ON estado, superficie
  FROM vec_contexto_actor_v1.vinculo_corporativo_versiones;
SQL
exigir_fallo_intacto 'retirada con estadistica hostil' retirar
psql_sql <<'SQL'
DROP STATISTICS vec_contexto_actor_v1.c22b_estadistica_hostil;
CREATE PUBLICATION c22b_publicacion_hostil
  FOR TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones;
SQL
exigir_fallo_intacto 'retirada con publicacion hostil' retirar
psql_sql <<'SQL'
DROP PUBLICATION c22b_publicacion_hostil;
SET ROLE vec_contexto_actor_v1_propietario;
CREATE VIEW vec_contexto_actor_v1.c22b_consumidor_hostil AS
 SELECT estado FROM vec_contexto_actor_v1.vinculo_corporativo_versiones;
RESET ROLE;
SQL
exigir_fallo_intacto 'retirada con vista consumidora' retirar
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
DROP VIEW vec_contexto_actor_v1.c22b_consumidor_hostil;
CREATE FUNCTION vec_contexto_actor_v1.c22b_consumidor_dinamico()
RETURNS bigint LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE resultado bigint;
BEGIN
  EXECUTE 'SELECT count(*) FROM vec_contexto_actor_v1.vinculo_corporativo_versiones'
    INTO resultado;
  RETURN resultado;
END $$;
RESET ROLE;
SQL
exigir_fallo_intacto 'retirada con consumidor dinamico' retirar
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
DROP FUNCTION vec_contexto_actor_v1.c22b_consumidor_dinamico();
RESET ROLE;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_contexto_actor_v1_propietario
 IN SCHEMA vec_contexto_actor_v1 GRANT SELECT ON TABLES TO c22b_consumidor;
SQL
exigir_fallo_intacto 'retirada con ACL predeterminada hostil' retirar
psql_sql <<'SQL'
ALTER DEFAULT PRIVILEGES FOR ROLE vec_contexto_actor_v1_propietario
 IN SCHEMA vec_contexto_actor_v1 REVOKE SELECT ON TABLES FROM c22b_consumidor;
SET ROLE vec_contexto_actor_v1_propietario;
CREATE TABLE vec_contexto_actor_v1.c22b_000005_sintetica(id integer);
RESET ROLE;
SQL
exigir_fallo_intacto 'retirada ante 000005 sintetica' retirar
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
DROP TABLE vec_contexto_actor_v1.c22b_000005_sintetica;
RESET ROLE;
SQL

# Caso 11: cualquier evidencia de B impide retirarla y no altera el estado.
restaurar_generacion=$(consulta "SELECT format(
 'UPDATE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 SET generacion=%s, actualizada_en=%L',
 generacion,actualizada_en) FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2")
evidencia=$(mktemp)
temporales+=("$evidencia")
sed 's/^ROLLBACK;$/COMMIT;/' "$integridad" > "$evidencia"
psql_temporal "$evidencia"
exigir_fallo_intacto 'retirada con historia y puntero persistentes' retirar
psql_sql <<SQL
SET session_replication_role=replica;
DELETE FROM vec_contexto_actor_v1.vinculo_corporativo_actual;
DELETE FROM vec_contexto_actor_v1.vinculo_corporativo_versiones;
DELETE FROM vec_contexto_actor_v1.organizacion_versiones
 WHERE organizacion_ref='org_diputaciondemo0001';
DELETE FROM vec_contexto_actor_v1.vinculo_contexto_versiones
 WHERE vinculo_ref='vca_corporativo_rrhh_000000000001';
DELETE FROM vec_contexto_actor_v1.perfil_versiones
 WHERE perfil_ref='prf_corporativo_rrhh_000000000001';
DELETE FROM vec_contexto_actor_v1.persona_versiones
 WHERE persona_ref='per_corporativa_rrhh_000000000001';
DELETE FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones
 WHERE cuenta_ref='cta_corporativa_rrhh_000000000001';
DELETE FROM vec_contexto_actor_v1.procedencias
 WHERE procedencia_ref IN ('prc_autoridad_corporativa_rrhh_0001',
                           'prc_vinculo_corporativo_rrhh_000001');
$restaurar_generacion;
SET session_replication_role=origin;
SQL
psql_archivo "$fixtures_base"
huella_base_poblada=$(huella_filas_base)
generacion_base_poblada=$(consulta 'SELECT generacion FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2')
retirar
[[ $(huella_filas_base) == "$huella_base_poblada" ]] ||
  fallar 'la retirada altero filas de A pobladas'
[[ $(consulta 'SELECT generacion FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2') == "$generacion_base_poblada" ]] ||
  fallar 'la retirada altero la generacion comun'

# Casos 12-13: reinstalacion desde A y segunda retirada exacta.
psql_archivo "$up"
psql_archivo "$estructura"
retirar
[[ $(huella_filas_base) == "$huella_base_poblada" ]] ||
  fallar 'el segundo ciclo altero filas de A pobladas'
[[ $(consulta 'SELECT generacion FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2') == "$generacion_base_poblada" ]] ||
  fallar 'el segundo ciclo altero la generacion comun'

echo 'OK: vinculo corporativo RRHH V1 supera estructura, integridad y retirada PG 18.4'
