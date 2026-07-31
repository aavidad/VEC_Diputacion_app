#!/usr/bin/env bash
set -euo pipefail

export LC_ALL=C
export TZ=UTC

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"

imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-contexto-actor-s02b-pg-${USER:-usuario}-$$"
base=ct132_s02b
clave_admin=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
directorio=deploy/postgresql/contexto_actor_v1
roles="$directorio/roles_up.sql"
up_base="$directorio/migraciones/000001_contexto_actor_v1.up.sql"
up_acreditacion="$directorio/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql"
down_acreditacion="$directorio/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.down.sql"
firma_acreditacion='vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)'

limpiar() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

esperar_postgres() {
  local consecutivas=0 respuesta
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
  echo 'PostgreSQL 18.4 primario no quedó disponible' >&2
  return 1
}

psql_archivo() {
  local archivo=$1
  docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/$archivo"
}

psql_admin() {
  docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base"
}

ejecutar_sql() {
  local sql=$1
  docker exec "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command "$sql"
}

consulta() {
  local sql=$1
  docker exec "$contenedor" psql -XAt --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command "$sql"
}

retirar_acreditacion() {
  docker exec --interactive \
    --env PGOPTIONS="-c vec.confirmar_retirada_acreditacion_contexto_actor_v2=RETIRAR_ACREDITACION_CONTEXTO_ACTOR_V2" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/$down_acreditacion"
}

retirar_sin_confirmacion() {
  psql_archivo "$down_acreditacion"
}

retirar_como_consumidor() {
  docker exec --interactive \
    --env PGOPTIONS="-c role=ct132_consumidor -c vec.confirmar_retirada_acreditacion_contexto_actor_v2=RETIRAR_ACREDITACION_CONTEXTO_ACTOR_V2" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/$down_acreditacion"
}

# pg_dump acredita estructura, ACL y datos. La segunda mitad añade catálogos
# que un volcado lógico no representa de forma exhaustiva. Se eliminan solo
# las guardas aleatorias que pg_dump genera en cada invocación.
huella_estado() {
  {
    docker exec "$contenedor" pg_dump --format=plain \
      --username postgres --dbname "$base" |
      sed -E '/^\\(un)?restrict /d'
    docker exec --interactive "$contenedor" psql -XAt \
      --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
WITH espacios AS (
  SELECT oid
    FROM pg_catalog.pg_namespace
   WHERE nspname IN ('vec_contexto_actor_v1', 'ct132_consumidor')
), clases AS (
  SELECT oid FROM pg_catalog.pg_class WHERE relnamespace IN (SELECT oid FROM espacios)
), procesos AS (
  SELECT oid FROM pg_catalog.pg_proc WHERE pronamespace IN (SELECT oid FROM espacios)
), tipos AS (
  SELECT oid FROM pg_catalog.pg_type WHERE typnamespace IN (SELECT oid FROM espacios)
), roles_objetivo AS (
  SELECT oid
    FROM pg_catalog.pg_roles
   WHERE rolname IN (
     'vec_contexto_actor_v1_propietario',
     'vec_contexto_actor_v1_migrador',
     'vec_contexto_actor_v1_runtime',
     'ct132_consumidor',
     'ct132_login_runtime'
   )
), objetos AS (
  SELECT 'pg_catalog.pg_class'::regclass AS clase, oid FROM clases
  UNION ALL SELECT 'pg_catalog.pg_proc'::regclass, oid FROM procesos
  UNION ALL SELECT 'pg_catalog.pg_type'::regclass, oid FROM tipos
), filas AS (
  SELECT 'namespace|' || pg_catalog.to_jsonb(n)::text AS fila
    FROM pg_catalog.pg_namespace AS n WHERE n.oid IN (SELECT oid FROM espacios)
  UNION ALL
  SELECT 'class|' || pg_catalog.to_jsonb(c)::text
    FROM pg_catalog.pg_class AS c WHERE c.oid IN (SELECT oid FROM clases)
  UNION ALL
  SELECT 'attribute|' || pg_catalog.to_jsonb(a)::text
    FROM pg_catalog.pg_attribute AS a WHERE a.attrelid IN (SELECT oid FROM clases)
  UNION ALL
  SELECT 'attrdef|' || pg_catalog.to_jsonb(a)::text
    FROM pg_catalog.pg_attrdef AS a WHERE a.adrelid IN (SELECT oid FROM clases)
  UNION ALL
  SELECT 'index|' || pg_catalog.to_jsonb(i)::text
    FROM pg_catalog.pg_index AS i
   WHERE i.indrelid IN (SELECT oid FROM clases)
      OR i.indexrelid IN (SELECT oid FROM clases)
  UNION ALL
  SELECT 'constraint|' || pg_catalog.to_jsonb(c)::text
    FROM pg_catalog.pg_constraint AS c
   WHERE c.connamespace IN (SELECT oid FROM espacios)
      OR c.conrelid IN (SELECT oid FROM clases)
      OR c.confrelid IN (SELECT oid FROM clases)
  UNION ALL
  SELECT 'trigger|' || pg_catalog.to_jsonb(t)::text
    FROM pg_catalog.pg_trigger AS t WHERE t.tgrelid IN (SELECT oid FROM clases)
  UNION ALL
  SELECT 'proc|' || pg_catalog.to_jsonb(p)::text
    FROM pg_catalog.pg_proc AS p WHERE p.oid IN (SELECT oid FROM procesos)
  UNION ALL
  SELECT 'type|' || pg_catalog.to_jsonb(t)::text
    FROM pg_catalog.pg_type AS t WHERE t.oid IN (SELECT oid FROM tipos)
  UNION ALL
  SELECT 'policy|' || pg_catalog.to_jsonb(p)::text
    FROM pg_catalog.pg_policy AS p WHERE p.polrelid IN (SELECT oid FROM clases)
  UNION ALL
  SELECT 'rewrite|' || pg_catalog.to_jsonb(r)::text
    FROM pg_catalog.pg_rewrite AS r WHERE r.ev_class IN (SELECT oid FROM clases)
  UNION ALL
  SELECT 'default_acl|' || pg_catalog.to_jsonb(d)::text
    FROM pg_catalog.pg_default_acl AS d
   WHERE d.defaclrole = 'vec_contexto_actor_v1_propietario'::regrole
  UNION ALL
  SELECT 'role|' || pg_catalog.to_jsonb(r)::text
    FROM pg_catalog.pg_roles AS r
   WHERE r.oid IN (SELECT oid FROM roles_objetivo)
  UNION ALL
  SELECT 'auth_member|' || pg_catalog.to_jsonb(m)::text
    FROM pg_catalog.pg_auth_members AS m
   WHERE m.roleid IN (SELECT oid FROM roles_objetivo)
      OR m.member IN (SELECT oid FROM roles_objetivo)
  UNION ALL
  SELECT 'role_setting|' || pg_catalog.to_jsonb(s)::text
    FROM pg_catalog.pg_db_role_setting AS s
   WHERE s.setrole IN (SELECT oid FROM roles_objetivo)
  UNION ALL
  SELECT 'depend|' || pg_catalog.to_jsonb(d)::text
    FROM pg_catalog.pg_depend AS d
   WHERE EXISTS (
     SELECT 1 FROM objetos AS o
      WHERE (d.classid, d.objid) = (o.clase, o.oid)
         OR (d.refclassid, d.refobjid) = (o.clase, o.oid)
   )
  UNION ALL
  SELECT 'description|' || pg_catalog.to_jsonb(d)::text
    FROM pg_catalog.pg_description AS d
   WHERE EXISTS (
     SELECT 1 FROM objetos AS o
      WHERE (d.classoid, d.objoid) = (o.clase, o.oid)
   )
  UNION ALL
  SELECT 'publication_rel|' || pg_catalog.to_jsonb(p)::text
    FROM pg_catalog.pg_publication_rel AS p
   WHERE p.prrelid IN (SELECT oid FROM clases)
  UNION ALL
  SELECT 'statistic_ext|' || pg_catalog.to_jsonb(s)::text
    FROM pg_catalog.pg_statistic_ext AS s
   WHERE s.stxnamespace IN (SELECT oid FROM espacios)
      OR s.stxrelid IN (SELECT oid FROM clases)
)
SELECT fila FROM filas ORDER BY fila COLLATE "C";
SQL
  } | sha256sum | cut -d' ' -f1
}

exigir_rechazo_sin_cambio() {
  local descripcion=$1 ejecutor=$2
  local antes despues salida estado
  antes=$(huella_estado)
  set +e
  salida=$($ejecutor 2>&1)
  estado=$?
  set -e
  if [[ $estado -eq 0 ]]; then
    echo "retirada aceptada indebidamente: $descripcion" >&2
    return 1
  fi
  despues=$(huella_estado)
  if [[ $antes != "$despues" ]]; then
    echo "rollback de catálogo o datos incompleto: $descripcion" >&2
    echo "$salida" >&2
    return 1
  fi
}

probar_deriva() {
  local descripcion=$1 preparar=$2 restaurar=$3
  local base_exacta hostil restaurada
  base_exacta=$(huella_estado)
  ejecutar_sql "$preparar"
  hostil=$(huella_estado)
  if [[ $hostil == "$base_exacta" ]]; then
    echo "la perturbación no cambió la huella: $descripcion" >&2
    return 1
  fi
  exigir_rechazo_sin_cambio "$descripcion" retirar_acreditacion
  ejecutar_sql "$restaurar"
  restaurada=$(huella_estado)
  if [[ $restaurada != "$base_exacta" ]]; then
    echo "la restauración no fue exacta: $descripcion" >&2
    return 1
  fi
}

docker run --detach --rm --name "$contenedor" \
  --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave_admin" \
  "$imagen" >/dev/null
esperar_postgres

psql_admin <<'SQL'
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
psql_archivo "$roles"
psql_archivo "$up_base"
psql_archivo "$up_acreditacion"

psql_admin <<'SQL'
CREATE ROLE ct132_consumidor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE ct132_login_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contexto_actor_v1_runtime TO ct132_login_runtime
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL

# Línea base exacta de los objetos aportados por 000002 y denegación por
# defecto para PUBLIC, runtime y un consumidor posterior sin privilegios.
psql_admin <<'SQL'
DO $linea_base$
DECLARE
  esquema oid := 'vec_contexto_actor_v1'::regnamespace;
  propietario oid := 'vec_contexto_actor_v1_propietario'::regrole;
  runtime oid := 'vec_contexto_actor_v1_runtime'::regrole;
  consumidor oid := 'ct132_consumidor'::regrole;
  control oid :=
    'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass;
  tipo_control oid :=
    'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regtype;
  tipo_array_control oid :=
    'vec_contexto_actor_v1._control_generacion_punteros_actuales_v2'::regtype;
  acreditar oid := pg_catalog.to_regprocedure(
    'vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)'
  );
BEGIN
  IF (
    SELECT pg_catalog.count(*)
      FROM pg_catalog.pg_roles
     WHERE rolname IN (
       'vec_contexto_actor_v1_propietario',
       'vec_contexto_actor_v1_runtime'
     )
       AND NOT rolsuper AND NOT rolinherit AND NOT rolcreaterole
       AND NOT rolcreatedb AND NOT rolcanlogin AND NOT rolreplication
       AND NOT rolbypassrls
  ) <> 2
  OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_auth_members
     WHERE member = runtime OR grantor = runtime
  )
  OR EXISTS (
    SELECT 1
      FROM pg_catalog.pg_auth_members AS m
      JOIN pg_catalog.pg_roles AS r ON r.oid = m.member
     WHERE m.roleid = runtime
       AND (
         m.admin_option OR NOT m.inherit_option OR m.set_option
         OR NOT r.rolcanlogin OR r.rolsuper OR NOT r.rolinherit
         OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication
         OR r.rolbypassrls
         OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_auth_members AS adicional
            WHERE adicional.member = m.member
         ) <> 1
         OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_db_role_setting AS s
            WHERE s.setrole = m.member
         )
       )
  )
  OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_db_role_setting
     WHERE setrole = runtime
       AND setdatabase IN (
         0, (SELECT oid FROM pg_catalog.pg_database
              WHERE datname = pg_catalog.current_database())
       )
  )
  OR (
    SELECT nspowner FROM pg_catalog.pg_namespace WHERE oid = esquema
  ) IS DISTINCT FROM propietario
  OR (
    SELECT pg_catalog.format(
      '%s|%s|%s|%s|%s|%s|%s',
      relkind, relpersistence, relowner = propietario, NOT relrowsecurity,
      NOT relforcerowsecurity, NOT relhasrules, NOT relhastriggers
    ) FROM pg_catalog.pg_class WHERE oid = control
  ) IS DISTINCT FROM 'r|p|t|t|t|t|t'
  OR (
    SELECT pg_catalog.string_agg(
      pg_catalog.format('%s:%s:%s:%s', attname,
        pg_catalog.format_type(atttypid, atttypmod), attnotnull, attisdropped),
      ',' ORDER BY attnum
    )
      FROM pg_catalog.pg_attribute
     WHERE attrelid = control AND attnum > 0
  ) IS DISTINCT FROM
    'control_id:boolean:t:f,generacion:numeric:t:f,actualizada_en:timestamp with time zone:t:f'
  OR (
    SELECT pg_catalog.count(*) FROM pg_catalog.pg_constraint
     WHERE conrelid = control
  ) <> 7
  OR (
    SELECT pg_catalog.count(*) FROM pg_catalog.pg_index
     WHERE indrelid = control AND indisprimary AND indisunique
       AND indisvalid AND indisready AND indislive
  ) <> 1
  OR (
    SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc
     WHERE pronamespace = esquema
       AND proname IN (
         'serializar_mutacion_punteros_actuales_v2',
         'avanzar_generacion_punteros_actuales_v2',
         'acreditar_uso_registro_contexto_actor_v2'
       )
       AND proowner = propietario AND prosecdef AND provolatile = 'v'
       AND proconfig = ARRAY['search_path=pg_catalog']
  ) <> 3
  OR (
    SELECT pg_catalog.count(*) FROM pg_catalog.pg_trigger
     WHERE NOT tgisinternal
       AND tgname IN (
         'puntero_actual_no_truncable_v2',
         'serializar_mutacion_punteros_actuales_v2',
         'avanzar_generacion_punteros_actuales_v2'
       )
  ) <> 15
  OR (
    SELECT pg_catalog.count(DISTINCT tgrelid) FROM pg_catalog.pg_trigger
     WHERE NOT tgisinternal
       AND tgname IN (
         'puntero_actual_no_truncable_v2',
         'serializar_mutacion_punteros_actuales_v2',
         'avanzar_generacion_punteros_actuales_v2'
       )
  ) <> 5
  OR (
    SELECT pg_catalog.count(*) FROM pg_catalog.pg_class
     WHERE relnamespace = esquema AND relkind = 'S'
  ) <> 0
  OR (
    SELECT pg_catalog.count(*) FROM pg_catalog.pg_type
     WHERE typnamespace = esquema
       AND typname IN (
         'control_generacion_punteros_actuales_v2',
         '_control_generacion_punteros_actuales_v2'
       )
  ) <> 2
  OR pg_catalog.obj_description(acreditar, 'pg_proc') IS DISTINCT FROM
    'Acredita un rca_ V2 contra bytes, procedencia y punteros actuales; devuelve solo el instante autoritativo y no concede acceso a tablas.'
  OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_namespace AS n
    CROSS JOIN LATERAL pg_catalog.aclexplode(
      coalesce(n.nspacl, pg_catalog.acldefault('n', n.nspowner))
    ) AS a
     WHERE n.oid = esquema AND a.grantee = 0
  )
  OR pg_catalog.has_schema_privilege(consumidor, esquema, 'USAGE')
  OR pg_catalog.has_table_privilege(runtime, control, 'SELECT')
  OR pg_catalog.has_table_privilege(consumidor, control, 'SELECT')
  OR pg_catalog.has_column_privilege(runtime, control, 2::smallint, 'SELECT')
  OR pg_catalog.has_column_privilege(
    consumidor, control, 2::smallint, 'SELECT'
  )
  OR pg_catalog.has_function_privilege(runtime, acreditar, 'EXECUTE')
  OR pg_catalog.has_function_privilege(consumidor, acreditar, 'EXECUTE')
  OR pg_catalog.has_type_privilege(runtime, tipo_control, 'USAGE')
  OR pg_catalog.has_type_privilege(runtime, tipo_array_control, 'USAGE')
  OR pg_catalog.has_type_privilege(consumidor, tipo_control, 'USAGE')
  OR pg_catalog.has_type_privilege(consumidor, tipo_array_control, 'USAGE')
  OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_proc AS p
    CROSS JOIN LATERAL pg_catalog.aclexplode(
      coalesce(p.proacl, pg_catalog.acldefault('f', p.proowner))
    ) AS a
     WHERE p.oid = acreditar AND a.grantee = 0
  )
  OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_type AS t
    CROSS JOIN LATERAL pg_catalog.aclexplode(
      coalesce(t.typacl, pg_catalog.acldefault('T', t.typowner))
    ) AS a
     WHERE t.oid = tipo_control
       AND a.grantee IN (0, runtime, consumidor)
  )
  OR (
    SELECT pg_catalog.count(*) = 1
       AND pg_catalog.bool_and(
         control_id AND generacion = 0
         AND vec_contexto_actor_v1.instante_valido(actualizada_en)
       )
      FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
  ) IS NOT TRUE THEN
    RAISE EXCEPTION USING ERRCODE = '55000',
      MESSAGE = 'línea base 000002 o denegación predeterminada no acreditada';
  END IF;
END
$linea_base$;
SQL

huella_base=$(huella_estado)
exigir_rechazo_sin_cambio 'confirmación explícita ausente' retirar_sin_confirmacion
exigir_rechazo_sin_cambio 'ejecutor no superusuario' retirar_como_consumidor

probar_deriva 'tabla de control con opciones y comentario' \
  "ALTER TABLE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 SET (fillfactor=70);
   COMMENT ON TABLE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 IS 'ct132 deriva sintética';" \
  "COMMENT ON TABLE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 IS NULL;
   ALTER TABLE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 RESET (fillfactor);"

probar_deriva 'columna de control con estadística y comentario' \
  "ALTER TABLE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
     ALTER COLUMN generacion SET STATISTICS 321;
   COMMENT ON COLUMN vec_contexto_actor_v1.control_generacion_punteros_actuales_v2.generacion IS 'ct132 deriva sintética';" \
  "COMMENT ON COLUMN vec_contexto_actor_v1.control_generacion_punteros_actuales_v2.generacion IS NULL;
   ALTER TABLE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
     ALTER COLUMN generacion SET STATISTICS -1;"

probar_deriva 'restricción posterior en la tabla de control' \
  "ALTER TABLE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
     ADD CONSTRAINT ct132_generacion_acotada CHECK (generacion < 999999);" \
  "ALTER TABLE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
     DROP CONSTRAINT ct132_generacion_acotada;"

probar_deriva 'índice base con opción y comentario' \
  "ALTER INDEX vec_contexto_actor_v1.control_generacion_punteros_actuales_v2_pkey
     SET (fillfactor=70);
   COMMENT ON INDEX vec_contexto_actor_v1.control_generacion_punteros_actuales_v2_pkey IS 'ct132 deriva sintética';" \
  "COMMENT ON INDEX vec_contexto_actor_v1.control_generacion_punteros_actuales_v2_pkey IS NULL;
   ALTER INDEX vec_contexto_actor_v1.control_generacion_punteros_actuales_v2_pkey
     RESET (fillfactor);"

probar_deriva 'función gestionada con coste y comentario' \
  "ALTER FUNCTION vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2() COST 321;
   COMMENT ON FUNCTION vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2() IS 'ct132 deriva sintética';" \
  "COMMENT ON FUNCTION vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2() IS NULL;
   ALTER FUNCTION vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2() COST 100;"

probar_deriva 'trigger gestionado deshabilitado y comentado' \
  "ALTER TABLE vec_contexto_actor_v1.proyeccion_cuenta_actual
     DISABLE TRIGGER serializar_mutacion_punteros_actuales_v2;
   COMMENT ON TRIGGER serializar_mutacion_punteros_actuales_v2
     ON vec_contexto_actor_v1.proyeccion_cuenta_actual IS 'ct132 deriva sintética';" \
  "COMMENT ON TRIGGER serializar_mutacion_punteros_actuales_v2
     ON vec_contexto_actor_v1.proyeccion_cuenta_actual IS NULL;
   ALTER TABLE vec_contexto_actor_v1.proyeccion_cuenta_actual
     ENABLE TRIGGER serializar_mutacion_punteros_actuales_v2;"

probar_deriva 'estadística extendida posterior' \
  "CREATE STATISTICS vec_contexto_actor_v1.ct132_control_estadistica
     ON generacion, actualizada_en
     FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2;" \
  "DROP STATISTICS vec_contexto_actor_v1.ct132_control_estadistica;"

probar_deriva 'ACL de esquema concedida a PUBLIC' \
  "GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO PUBLIC;" \
  "REVOKE USAGE ON SCHEMA vec_contexto_actor_v1 FROM PUBLIC;"

probar_deriva 'ACL de tabla y columna concedida a rol posterior' \
  "GRANT SELECT ON vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
     TO ct132_consumidor;
   GRANT UPDATE(generacion)
     ON vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
     TO ct132_consumidor;" \
  "REVOKE ALL ON vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
     FROM ct132_consumidor;
   REVOKE UPDATE(generacion)
     ON vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
     FROM ct132_consumidor;"

probar_deriva 'secuencia posterior con ACL de PUBLIC y rol' \
  "BEGIN;
   SET LOCAL ROLE vec_contexto_actor_v1_propietario;
   CREATE SEQUENCE vec_contexto_actor_v1.ct132_secuencia;
   GRANT USAGE ON SEQUENCE vec_contexto_actor_v1.ct132_secuencia
     TO PUBLIC, ct132_consumidor;
   COMMIT;" \
  "BEGIN;
   SET LOCAL ROLE vec_contexto_actor_v1_propietario;
   DROP SEQUENCE vec_contexto_actor_v1.ct132_secuencia;
   COMMIT;"

probar_deriva 'ACL de función concedida a PUBLIC y rol' \
  "GRANT EXECUTE ON FUNCTION $firma_acreditacion
     TO PUBLIC, ct132_consumidor;" \
  "REVOKE EXECUTE ON FUNCTION $firma_acreditacion
     FROM PUBLIC, ct132_consumidor;"

probar_deriva 'ACL y comentario de tipo compuesto' \
  "GRANT USAGE ON TYPE
     vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
     TO PUBLIC, ct132_consumidor;
   COMMENT ON TYPE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
     IS 'ct132 deriva sintética';" \
  "COMMENT ON TYPE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
     IS NULL;
   REVOKE USAGE ON TYPE
     vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
     FROM PUBLIC, ct132_consumidor;"

probar_deriva 'ACL predeterminada para tablas posteriores' \
  "ALTER DEFAULT PRIVILEGES FOR ROLE vec_contexto_actor_v1_propietario
     IN SCHEMA vec_contexto_actor_v1
     GRANT SELECT ON TABLES TO ct132_consumidor;" \
  "ALTER DEFAULT PRIVILEGES FOR ROLE vec_contexto_actor_v1_propietario
     IN SCHEMA vec_contexto_actor_v1
     REVOKE SELECT ON TABLES FROM ct132_consumidor;"

probar_deriva 'runtime miembro efectivo del propietario' \
  "GRANT vec_contexto_actor_v1_propietario
     TO vec_contexto_actor_v1_runtime
     WITH ADMIN FALSE, INHERIT TRUE, SET TRUE;" \
  "REVOKE vec_contexto_actor_v1_propietario
     FROM vec_contexto_actor_v1_runtime;"

probar_deriva 'runtime habilitado para LOGIN y BYPASSRLS' \
  "ALTER ROLE vec_contexto_actor_v1_runtime LOGIN BYPASSRLS;" \
  "ALTER ROLE vec_contexto_actor_v1_runtime NOLOGIN NOBYPASSRLS;"

probar_deriva 'runtime con search_path persistente en la base' \
  "ALTER ROLE vec_contexto_actor_v1_runtime
     IN DATABASE $base SET search_path = 'public';" \
  "ALTER ROLE vec_contexto_actor_v1_runtime
     IN DATABASE $base RESET search_path;"

# Un puntero posterior se descubre por su trío, sin añadir su nombre a una
# lista del runner ni del down.
probar_deriva 'puntero posterior con trío gestionado' \
  "BEGIN;
   SET LOCAL ROLE vec_contexto_actor_v1_propietario;
   CREATE TABLE vec_contexto_actor_v1.ct132_puntero_posterior(
     referencia text PRIMARY KEY
   );
   CREATE TRIGGER puntero_actual_no_truncable_v2 BEFORE TRUNCATE
     ON vec_contexto_actor_v1.ct132_puntero_posterior FOR EACH STATEMENT
     EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();
   CREATE TRIGGER serializar_mutacion_punteros_actuales_v2
     BEFORE INSERT OR UPDATE OR DELETE
     ON vec_contexto_actor_v1.ct132_puntero_posterior FOR EACH STATEMENT
     EXECUTE FUNCTION vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2();
   CREATE TRIGGER avanzar_generacion_punteros_actuales_v2
     AFTER INSERT OR UPDATE OR DELETE
     ON vec_contexto_actor_v1.ct132_puntero_posterior FOR EACH STATEMENT
     EXECUTE FUNCTION vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2();
   COMMIT;" \
  "BEGIN;
   SET LOCAL ROLE vec_contexto_actor_v1_propietario;
   DROP TABLE vec_contexto_actor_v1.ct132_puntero_posterior;
   COMMIT;"

# Este consumidor usa cuerpo textual, por lo que el inventario nominal debe
# hallarlo sin conocer su nombre.
probar_deriva 'consumidor nominal posterior de acreditación' \
  "CREATE SCHEMA ct132_consumidor AUTHORIZATION postgres;
   CREATE FUNCTION ct132_consumidor.usar_acreditacion()
   RETURNS timestamptz LANGUAGE sql
   AS \$cuerpo\$
     SELECT vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
       NULL::text,NULL::text,NULL::text,NULL::text,NULL::text,NULL::text,
       NULL::numeric,NULL::text,NULL::numeric,NULL::text,NULL::numeric,
       NULL::text,NULL::numeric,NULL::text,NULL::text,
       NULL::timestamptz,NULL::timestamptz)
   \$cuerpo\$;" \
  "DROP SCHEMA ct132_consumidor CASCADE;"

# La vista no coincide con el detector nominal. DROP ... RESTRICT debe
# conservarla y el rollback debe restaurar también los objetos ya retirados.
probar_deriva 'dependencia externa conservada por RESTRICT' \
  "CREATE SCHEMA ct132_consumidor AUTHORIZATION postgres;
   CREATE VIEW ct132_consumidor.control_visible AS
     SELECT control_id, generacion, actualizada_en
       FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2;" \
  "DROP SCHEMA ct132_consumidor CASCADE;"

probar_deriva 'publicación posterior de la tabla de control' \
  "CREATE PUBLICATION ct132_publicacion
     FOR TABLE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
     WITH (publish = 'insert');" \
  "DROP PUBLICATION ct132_publicacion;"

[[ $(huella_estado) == "$huella_base" ]] || {
  echo 'la matriz no restauró la instalación 000002 exacta' >&2
  exit 1
}

retirar_acreditacion

resultado=$(consulta "
  SELECT
    pg_catalog.to_regclass(
      'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'
    ) IS NULL
    AND pg_catalog.to_regprocedure(
      '$firma_acreditacion'
    ) IS NULL
    AND pg_catalog.to_regclass(
      'vec_contexto_actor_v1.registros_contexto'
    ) IS NOT NULL
    AND (
      SELECT pg_catalog.count(*) FROM pg_catalog.pg_trigger
       WHERE NOT tgisinternal
         AND tgname IN (
           'puntero_actual_no_truncable_v2',
           'serializar_mutacion_punteros_actuales_v2',
           'avanzar_generacion_punteros_actuales_v2'
         )
    ) = 0")
[[ $resultado == t ]] || {
  echo 'la retirada final no dejó la postcondición exacta' >&2
  exit 1
}

echo 'S0.2b: estructura, ACL, consumidores, rollback y retirada PostgreSQL 18.4 superados'
