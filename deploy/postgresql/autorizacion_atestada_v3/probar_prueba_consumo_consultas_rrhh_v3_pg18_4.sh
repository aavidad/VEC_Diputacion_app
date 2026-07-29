#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd -P)"
imagen="${VEC_POSTGRES_TEST_IMAGE:-postgres@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}"
contenedor="vec-prueba-consumo-rrhh-v3-${PPID}-${RANDOM}"

if [[ ! "${imagen}" =~ ^[^[:space:]@]+@sha256:[0-9a-f]{64}$ ]]; then
    printf 'VEC_POSTGRES_TEST_IMAGE debe fijarse por digest sha256\n' >&2
    exit 64
fi
temporales="$(mktemp -d)"

limpiar() {
    docker rm --force --volumes "${contenedor}" >/dev/null 2>&1 || true
    rm -rf -- "${temporales}"
}
trap limpiar EXIT INT TERM

paso() {
    printf '[prueba consumo RRHH V3:PostgreSQL] %s\n' "$1"
}

archivo() {
    local usuario="$1"
    local ruta="$2"
    docker exec --interactive "${contenedor}" \
        psql -X --set ON_ERROR_STOP=1 --username "${usuario}" \
        --dbname postgres --file "/repo/${ruta}"
}

sql() {
    local usuario="$1"
    local consulta="$2"
    docker exec "${contenedor}" \
        psql -X --set ON_ERROR_STOP=1 --username "${usuario}" \
        --dbname postgres --command "${consulta}"
}

valor() {
    docker exec "${contenedor}" \
        psql -XAt --set ON_ERROR_STOP=1 --username postgres \
        --dbname postgres --command "$1"
}

esperar_sesion() {
    local aplicacion="$1"
    local estado="$2"
    local espera="${3:-}"
    local coincidencias
    for _ in {1..500}; do
        coincidencias="$(valor "SELECT count(*)
          FROM pg_stat_activity
         WHERE application_name='${aplicacion}'
           AND state='${estado}'
           AND ('${espera}'='' OR wait_event_type='${espera}')")"
        if [[ "${coincidencias}" == '1' ]]; then
            return 0
        fi
        sleep 0.01
    done
    valor "SELECT application_name,state,
        COALESCE(wait_event_type,''),COALESCE(wait_event,'')
      FROM pg_stat_activity
     WHERE application_name='${aplicacion}'" >&2
    printf 'sesión causal no alcanzó el estado esperado: %s/%s/%s\n' \
        "${aplicacion}" "${estado}" "${espera}" >&2
    return 1
}

esperar_fallo() {
    local descripcion="$1"
    shift
    local salida
    if salida="$("$@" 2>&1)"; then
        printf 'Se esperaba rechazo: %s\n%s\n' \
            "${descripcion}" "${salida}" >&2
        return 1
    fi
    paso "rechazo verificado: ${descripcion}"
}

preparar() {
    sql postgres \
        "SELECT public.preparar_vector_consulta_rrhh_v3('$1','$2')" \
        >/dev/null
}

consumir() {
    local caso="$1"
    local fachada="$2"
    docker exec --interactive "${contenedor}" \
        psql -XAtq --set ON_ERROR_STOP=1 \
        --username vec_consulta_rrhh_runtime --dbname postgres <<SQL
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT consumo_nuevo
  FROM vec_contratacion_temporal.prueba_consumir_consulta_rrhh_v3(
    '${caso}','${fachada}',''
  );
COMMIT;
SQL
}

probar() {
    local caso="$1"
    local fachada="$2"
    local pieza="${3:-}"
    docker exec --interactive "${contenedor}" \
        psql -XAtq --set ON_ERROR_STOP=1 \
        --username vec_consulta_rrhh_runtime --dbname postgres <<SQL
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT pg_catalog.concat_ws(
           '|', decision_ref, efecto_ref, huella_efecto_sha256,
           consumo_huella_sha256, auditoria_ref,
           auditoria_huella_sha256, consumida_en::text,
           revalidada_en::text
       )
  FROM vec_contratacion_temporal
       .prueba_evidencia_consumo_consulta_rrhh_v3(
    '${caso}','${fachada}','${pieza}'
  );
COMMIT;
SQL
}

estado_vec() {
    valor "SELECT pg_catalog.concat_ws('|',
        (SELECT pg_catalog.count(*)
           FROM vec_autorizacion_atestada_v3.atestacion_decision_v3),
        (SELECT pg_catalog.count(*)
           FROM vec_autorizacion_atestada_v3.consumo_decision_v3),
        (SELECT pg_catalog.count(*)
           FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3),
        (SELECT secuencia
           FROM vec_autorizacion_atestada_v3.control_cadena_auditoria
          WHERE control_id),
        (SELECT cabeza_sha256
           FROM vec_autorizacion_atestada_v3.control_cadena_auditoria
          WHERE control_id))"
}

paso "arranque efímero sin red: ${imagen}"
docker run --detach --name "${contenedor}" --network none \
    --env POSTGRES_PASSWORD="$(openssl rand -hex 24)" \
    --env POSTGRES_INITDB_ARGS='--auth-local=trust' \
    --tmpfs /var/lib/postgresql:rw,noexec,nosuid,size=768m \
    "${imagen}" >/dev/null
for _ in {1..60}; do
    docker exec "${contenedor}" pg_isready --quiet \
        --username postgres --dbname postgres && break
    sleep 1
done
docker exec "${contenedor}" pg_isready --quiet \
    --username postgres --dbname postgres
docker cp "${raiz}/." "${contenedor}:/repo"

paso 'endurecimiento y dependencias nominales'
docker exec --interactive "${contenedor}" \
    psql -X --set ON_ERROR_STOP=1 --username postgres \
    --dbname postgres <<'SQL'
REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
DO $version$
BEGIN
  IF current_setting('server_version_num')::integer <> 180004 THEN
    RAISE EXCEPTION 'se exige PostgreSQL 18.4 exacto';
  END IF;
END
$version$;
SQL

for ruta in \
    deploy/postgresql/contexto_actor_v1/roles_up.sql \
    deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql \
    deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql \
    deploy/postgresql/contexto_actor_v1/pruebas_sql/fixtures_sinteticos.sql \
    deploy/postgresql/autorizacion/pruebas_sql/fixture_contexto_actor_v3.sql
do
    archivo postgres "${ruta}" >/dev/null
done
sql postgres \
    "CREATE ROLE vec_contexto_consulta_rrhh LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS; GRANT vec_contexto_actor_v1_runtime TO vec_contexto_consulta_rrhh WITH ADMIN FALSE, INHERIT TRUE, SET FALSE" \
    >/dev/null
docker exec --interactive "${contenedor}" \
    psql -X --set ON_ERROR_STOP=1 \
    --username vec_contexto_consulta_rrhh --dbname postgres <<'SQL' \
    >/dev/null
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SELECT count(*)
  FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
    'oca_registro_v3_000000000000000000000000',
    'rca_registro_v3_000000000000000000000000',
    'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',
    'prf_sintetico_cccccccccccccccccccccccc',
    'certificado','alto',clock_timestamp()
  );
COMMIT;
SQL

sql postgres \
    'CREATE EXTENSION pgcrypto WITH SCHEMA public; REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC' \
    >/dev/null
for ruta in \
    deploy/postgresql/autorizacion/roles_up.sql \
    deploy/postgresql/autorizacion/roles_v2_up.sql \
    deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql \
    deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql \
    deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql \
    deploy/postgresql/autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql \
    deploy/postgresql/autorizacion/migraciones/000005_registro_decisiones_contexto_actor_v3.up.sql \
    deploy/postgresql/autorizacion/migraciones/000006_funcion_registro_decisiones_contexto_actor_v3.up.sql \
    deploy/postgresql/autorizacion/pruebas_sql/fixture_autorizacion_contexto_actor_v3.sql \
    deploy/postgresql/autorizacion/pruebas_sql/integracion_contexto_actor_v3.sql \
    deploy/postgresql/autorizacion/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.up.sql
do
    archivo postgres "${ruta}" >/dev/null
done

archivo postgres deploy/postgresql/contratacion_temporal/roles_up.sql \
    >/dev/null
sql postgres \
    "CREATE ROLE vec_contratacion_temporal_consultor_rrhh NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS; CREATE ROLE vec_consulta_rrhh_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS; GRANT CONNECT ON DATABASE postgres TO vec_contratacion_temporal_consultor_rrhh; GRANT vec_contratacion_temporal_consultor_rrhh TO vec_consulta_rrhh_runtime WITH ADMIN FALSE, INHERIT TRUE, SET FALSE" \
    >/dev/null
archivo postgres deploy/postgresql/autorizacion_atestada_v3/roles_up.sql \
    >/dev/null
sql postgres \
    "CREATE ROLE vec_ad3_consulta_migrador LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; GRANT CONNECT ON DATABASE postgres TO vec_ad3_consulta_migrador; GRANT vec_autorizacion_atestada_v3_migrador TO vec_ad3_consulta_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE" \
    >/dev/null

for migracion in \
    000001_gobierno_y_registro_v3 \
    000002_consumidor_capacidad_v3 \
    000003_consumidor_consulta_cuadro_rrhh_v3 \
    000004_consumidor_consulta_detalle_rrhh_v3 \
    000005_revalidacion_final_consultas_rrhh_v3
do
    archivo vec_ad3_consulta_migrador \
        "deploy/postgresql/autorizacion_atestada_v3/migraciones/${migracion}.up.sql" \
        >/dev/null
done

arriba=deploy/postgresql/autorizacion_atestada_v3/migraciones/000006_prueba_consumo_consultas_rrhh_v3.up.sql
abajo=deploy/postgresql/autorizacion_atestada_v3/migraciones/000006_prueba_consumo_consultas_rrhh_v3.down.sql

paso 'instalación, reentrada y dependencia CT protegida'
archivo vec_ad3_consulta_migrador "${arriba}" >/dev/null
esperar_fallo 'reentrada ascendente' \
    archivo vec_ad3_consulta_migrador "${arriba}"
[[ "$(valor "SELECT count(*) FROM pg_proc f
    JOIN pg_namespace n ON n.oid=f.pronamespace
    WHERE n.nspname='vec_autorizacion_atestada_v3'
      AND f.proname LIKE
          'revalidar_evidencia_consumo_consulta_%rrhh_v3%'")" == '3' ]]

docker exec --interactive "${contenedor}" \
    psql -X --set ON_ERROR_STOP=1 --username postgres \
    --dbname postgres <<'SQL' >/dev/null
CREATE FUNCTION
vec_contratacion_temporal.dependencia_futura_prueba_consumo_rrhh()
RETURNS text
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $$
DECLARE
    v text;
BEGIN
    SELECT decision_ref INTO v
      FROM vec_autorizacion_atestada_v3
           .revalidar_evidencia_consumo_consulta_cuadro_rrhh_v3_atestada(
          NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL
      );
    RETURN v;
END
$$;
ALTER FUNCTION
    vec_contratacion_temporal.dependencia_futura_prueba_consumo_rrhh()
OWNER TO vec_contratacion_temporal_propietario;
SQL
esperar_fallo 'down RESTRICT con dependencia CT simulada' \
    archivo vec_ad3_consulta_migrador "${abajo}"
sql postgres \
    'DROP FUNCTION vec_contratacion_temporal.dependencia_futura_prueba_consumo_rrhh()' \
    >/dev/null

paso 'ciclo up-down-up y reentrada descendente'
archivo vec_ad3_consulta_migrador "${abajo}" >/dev/null
esperar_fallo 'reentrada descendente' \
    archivo vec_ad3_consulta_migrador "${abajo}"
[[ "$(valor "SELECT count(*) FROM pg_proc f
    JOIN pg_namespace n ON n.oid=f.pronamespace
    WHERE n.nspname='vec_autorizacion_atestada_v3'
      AND f.proname LIKE
          'revalidar_evidencia_consumo_consulta_%rrhh_v3%'")" == '0' ]]
archivo vec_ad3_consulta_migrador "${arriba}" >/dev/null

paso 'catálogo, ACL exacta y ausencia de lectura cruzada'
[[ "$(valor "WITH f AS (
    SELECT p.*, n.nspname, l.lanname,
           pg_get_userbyid(p.proowner) AS propietario
      FROM pg_proc p
      JOIN pg_namespace n ON n.oid=p.pronamespace
      JOIN pg_language l ON l.oid=p.prolang
     WHERE n.nspname='vec_autorizacion_atestada_v3'
       AND p.proname LIKE
           'revalidar_evidencia_consumo_consulta_%rrhh_v3%'
) SELECT count(*)=3
    AND bool_and(propietario='vec_autorizacion_atestada_v3_propietario')
    AND bool_and(lanname='plpgsql' AND prosecdef AND provolatile='v')
    AND bool_and(NOT proisstrict AND NOT proleakproof)
    AND bool_and(proparallel='u' AND proretset)
    AND bool_and(proconfig=ARRAY[
        'search_path=pg_catalog','lock_timeout=1s'])
    AND bool_and(pg_get_function_result(oid)=
        'TABLE(decision_ref text, efecto_ref text, huella_efecto_sha256 text, consumo_huella_sha256 text, auditoria_ref text, auditoria_huella_sha256 text, consumida_en timestamp with time zone, revalidada_en timestamp with time zone)')
    FROM f")" == 't' ]]

[[ "$(valor "WITH f AS (
    SELECT p.*,l.lanname,pg_get_userbyid(p.proowner) AS propietario
      FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
      JOIN pg_language l ON l.oid=p.prolang
     WHERE n.nspname='vec_autorizacion_atestada_v3'
       AND p.proname='serializar_revocacion_consultas_rrhh_v3'
) SELECT count(*)=1
    AND bool_and(propietario='vec_autorizacion_atestada_v3_propietario')
    AND bool_and(lanname='plpgsql' AND prosecdef AND provolatile='v')
    AND bool_and(NOT proisstrict AND NOT proleakproof)
    AND bool_and(proparallel='u' AND NOT proretset)
    AND bool_and(prorettype='pg_catalog.trigger'::regtype)
    AND bool_and(proconfig=ARRAY['search_path=pg_catalog'])
    AND bool_and(encode(sha256(convert_to(prosrc,'UTF8')),'hex')=
        '296a4a80d06c0d1f4668601eaf8690131215f547986430ff1be01286ed0a95eb')
    FROM f")" == 't' ]]

[[ "$(valor "WITH g AS (
    SELECT t.*,c.relname
      FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid
      JOIN pg_namespace n ON n.oid=c.relnamespace
     WHERE n.nspname='vec_autorizacion_atestada_v3'
       AND t.tgname='serializar_revalidacion_rrhh_antes'
) SELECT count(*)=3
    AND count(DISTINCT relname)=3
    AND bool_and(relname IN (
        'revocacion_clave_capacidad','revocacion_configuracion',
        'revocacion_raiz'))
    AND bool_and(tgfoid=
        'vec_autorizacion_atestada_v3.serializar_revocacion_consultas_rrhh_v3()'
        ::regprocedure)
    AND bool_and(tgtype=7 AND tgenabled='O' AND NOT tgisinternal)
    AND bool_and(tgnargs=0 AND tgparentid=0 AND tgconstraint=0)
    AND bool_and(tgconstrrelid=0 AND tgconstrindid=0)
    AND bool_and(NOT tgdeferrable AND NOT tginitdeferred)
    AND bool_and(tgattr=''::int2vector AND tgargs='\\x'::bytea)
    AND bool_and(tgqual IS NULL AND tgoldtable IS NULL)
    AND bool_and(tgnewtable IS NULL)
    AND bool_and(obj_description(oid,'pg_trigger') IS NULL)
    AND bool_and(pg_get_triggerdef(oid,false)=format(
        'CREATE TRIGGER serializar_revalidacion_rrhh_antes BEFORE INSERT ON vec_autorizacion_atestada_v3.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.serializar_revocacion_consultas_rrhh_v3()',
        relname))
    AND NOT EXISTS (
        SELECT 1 FROM pg_seclabels s
         WHERE s.classoid='pg_trigger'::regclass
           AND s.objoid IN (SELECT oid FROM g))
    FROM g")" == 't' ]]

[[ "$(valor "WITH g AS (
    SELECT t.oid,t.tgrelid,t.tgfoid
      FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid
      JOIN pg_namespace n ON n.oid=c.relnamespace
     WHERE n.nspname='vec_autorizacion_atestada_v3'
       AND t.tgname='serializar_revalidacion_rrhh_antes'
), d AS (
    SELECT g.*,x.refclassid,x.refobjid,x.deptype
      FROM g JOIN pg_depend x
        ON x.classid='pg_trigger'::regclass AND x.objid=g.oid
) SELECT count(*)=6
    AND bool_and(
        (refclassid='pg_proc'::regclass
         AND refobjid=tgfoid AND deptype='n')
        OR
        (refclassid='pg_class'::regclass
         AND refobjid=tgrelid AND deptype='a'))
  FROM d")" == 't' ]]

[[ "$(valor "WITH esperadas(nombre,huella) AS (VALUES
    ('revalidar_consumo_consulta_rrhh_v3_interna',
     'd3f72e15374a572dd6004193fc136369d3d5a42ec05e38e8afcd1868d8d8c553'),
    ('revalidar_consumo_consulta_cuadro_rrhh_v3_atestada',
     '6f697bddff143180a6562cdacaec3a965e47720f5563731fc5c5011d9798a3e3'),
    ('revalidar_consumo_consulta_detalle_rrhh_v3_atestada',
     'c7d3f11fbbb08ffb5a75ce347f19124888af4022a74d4a3c79e5076420bd7c09'),
    ('revalidar_evidencia_consumo_consulta_rrhh_v3_interna',
     '4cc79858a566d9b31f31729ee48bbde7a11e962b73062f8db3beacf3e96d632f'),
    ('revalidar_evidencia_consumo_consulta_cuadro_rrhh_v3_atestada',
     '0a225931a597f0d84a65ee5e8f0920560dce7330d647091b6e246c485f5caf8a'),
    ('revalidar_evidencia_consumo_consulta_detalle_rrhh_v3_atestada',
     'fa5a9b2732a97a1afa579687108dedfafd64f86d95d0dfecd1b79a01fcff5e65'),
    ('serializar_revocacion_consultas_rrhh_v3',
     '296a4a80d06c0d1f4668601eaf8690131215f547986430ff1be01286ed0a95eb')
) SELECT count(*)=7 AND bool_and(
        encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')=e.huella)
  FROM esperadas e JOIN pg_proc p ON p.proname=e.nombre
  JOIN pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_autorizacion_atestada_v3'")" == 't' ]]

[[ "$(valor "WITH exteriores(oid) AS (
    SELECT to_regprocedure(
        'vec_autorizacion_atestada_v3.revalidar_evidencia_consumo_consulta_cuadro_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
    UNION ALL
    SELECT to_regprocedure(
        'vec_autorizacion_atestada_v3.revalidar_evidencia_consumo_consulta_detalle_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
), internas(oid) AS (
    SELECT to_regprocedure(
        'vec_autorizacion_atestada_v3.revalidar_evidencia_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
    UNION ALL
    SELECT to_regprocedure(
        'vec_autorizacion_atestada_v3.serializar_revocacion_consultas_rrhh_v3()')
), denegados(rol) AS (VALUES
    ('public'), ('vec_contratacion_temporal_consultor_rrhh'),
    ('vec_consulta_rrhh_runtime'),
    ('vec_autorizacion_atestada_v3_migrador'),
    ('vec_autorizacion_atestada_v3_emisor'),
    ('vec_autorizacion_atestada_v3_consumidor')
) SELECT
    NOT EXISTS (
        SELECT 1 FROM internas i
         WHERE has_function_privilege(
             'vec_contratacion_temporal_propietario',i.oid,'EXECUTE'))
    AND NOT EXISTS (
        SELECT 1 FROM exteriores e
         WHERE NOT has_function_privilege(
             'vec_contratacion_temporal_propietario',e.oid,'EXECUTE'))
    AND NOT EXISTS (
        SELECT 1 FROM exteriores e, denegados d
         WHERE has_function_privilege(d.rol,e.oid,'EXECUTE'))")" == 't' ]]

[[ "$(valor "SELECT NOT EXISTS (
    SELECT 1 FROM pg_proc f JOIN pg_namespace n ON n.oid=f.pronamespace
    WHERE n.nspname='vec_autorizacion_atestada_v3'
      AND f.proname LIKE
          'revalidar_evidencia_consumo_consulta_%rrhh_v3%'
      AND pg_get_functiondef(f.oid) ~
          'FROM[[:space:]]+vec_contratacion_temporal\\.'
    )")" == 't' ]]

esperar_fallo 'runtime no ejecuta directamente la función nominal' \
    sql vec_consulta_rrhh_runtime \
    "SELECT * FROM vec_autorizacion_atestada_v3.revalidar_evidencia_consumo_consulta_cuadro_rrhh_v3_atestada(NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL)"

paso 'fixtures sintéticos y frontera propietaria CT'
sql postgres \
    'GRANT EXECUTE ON FUNCTION public.gen_random_bytes(integer) TO vec_autorizacion_atestada_v3_propietario' \
    >/dev/null
archivo postgres \
    deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/preparar_entorno_o2_05.sql \
    >/dev/null
archivo postgres \
    deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/consultas_rrhh_v3.sql \
    >/dev/null
archivo postgres \
    deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/prueba_consumo_consultas_rrhh_v3.sql \
    >/dev/null

paso 'fail-closed sin consumo y rollback conjunto'
preparar sin_consumo cuadro
esperar_fallo 'no existe consumo durable' \
    probar sin_consumo cuadro
preparar rollback_conjunto cuadro
estado_antes="$(estado_vec)"
docker exec --interactive "${contenedor}" \
    psql -XAtq --set ON_ERROR_STOP=1 \
    --username vec_consulta_rrhh_runtime --dbname postgres <<'SQL' \
    >/dev/null
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT count(*)
  FROM vec_contratacion_temporal.prueba_consumir_consulta_rrhh_v3(
    'rollback_conjunto','cuadro',''
  );
SELECT count(*)
  FROM vec_contratacion_temporal
       .prueba_evidencia_consumo_consulta_rrhh_v3(
    'rollback_conjunto','cuadro',''
  );
ROLLBACK;
SQL
[[ "$(estado_vec)" == "${estado_antes}" ]]

paso 'cuadro, detalle, replay y ocho campos autoritativos'
preparar prueba_cuadro cuadro
[[ "$(consumir prueba_cuadro cuadro)" == 't' ]]
estado_antes="$(estado_vec)"
salida_cuadro="$(probar prueba_cuadro cuadro)"
[[ "$(awk -F'|' '{print NF}' <<<"${salida_cuadro}")" == '8' ]]
[[ "$(cut -d'|' -f1-7 <<<"${salida_cuadro}")" == \
    "$(valor "SELECT concat_ws('|',t.decision_ref,t.efecto_ref,
        t.huella_efecto_sha256,c.consumo_huella_sha256,
        a.auditoria_ref,a.huella_sha256,c.consumida_en::text)
       FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 t
       JOIN vec_autorizacion_atestada_v3.consumo_decision_v3 c
         USING(decision_ref,efecto_ref,huella_efecto_sha256)
       JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
         USING(decision_ref,efecto_ref,huella_efecto_sha256)
      WHERE t.decision_ref='decision:consulta-rrhh:prueba_cuadro'")" ]]
salida_replay="$(probar prueba_cuadro cuadro)"
[[ "$(cut -d'|' -f1-7 <<<"${salida_replay}")" == \
    "$(cut -d'|' -f1-7 <<<"${salida_cuadro}")" ]]
[[ "$(estado_vec)" == "${estado_antes}" ]]

preparar prueba_detalle detalle
[[ "$(consumir prueba_detalle detalle)" == 't' ]]
[[ "$(probar prueba_detalle detalle | awk -F'|' '{print NF}')" == '8' ]]
esperar_fallo 'detalle no cruza por prueba de cuadro' \
    probar prueba_detalle cuadro
esperar_fallo 'cuadro no cruza por prueba de detalle' \
    probar prueba_cuadro detalle

paso 'las diez piezas permanecen ligadas'
preparar piezas_exactas cuadro
consumir piezas_exactas cuadro >/dev/null
estado_antes="$(estado_vec)"
for pieza in \
    capacidad decision motivo contexto persona_version perfil_version \
    payload cose evidencia spki
do
    esperar_fallo "prueba rechaza pieza: ${pieza}" \
        probar piezas_exactas cuadro "${pieza}"
done
[[ "$(estado_vec)" == "${estado_antes}" ]]

paso 'los locks de evidencia viven hasta el fin de la transacción'
entrada_bloqueo="${temporales}/bloqueo-evidencia.in"
salida_bloqueo="${temporales}/bloqueo-evidencia.out"
mkfifo "${entrada_bloqueo}"
docker exec --interactive "${contenedor}" \
    psql -XAtq --set ON_ERROR_STOP=1 \
    --username vec_consulta_rrhh_runtime --dbname postgres \
    <"${entrada_bloqueo}" >"${salida_bloqueo}" 2>&1 &
pid_prueba=$!
exec 9>"${entrada_bloqueo}"
printf '%s\n' \
    "BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;" \
    "SET application_name='prueba_lock_evidencia';" \
    "SET LOCAL TimeZone='UTC';" \
    "SET LOCAL statement_timeout='15s';" \
    "SET LOCAL idle_in_transaction_session_timeout='20s';" \
    "SELECT count(*) FROM vec_contratacion_temporal.prueba_evidencia_consumo_consulta_rrhh_v3('piezas_exactas','cuadro','');" \
    >&9
esperar_sesion prueba_lock_evidencia 'idle in transaction'
esperar_fallo 'FOR SHARE protege la fila de consumo' \
    sql postgres \
    "BEGIN; SET LOCAL lock_timeout='200ms'; SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario; SELECT 1 FROM vec_autorizacion_atestada_v3.consumo_decision_v3 WHERE decision_ref='decision:consulta-rrhh:piezas_exactas' FOR UPDATE; COMMIT"
printf '%s\n' \
    "COMMIT;" \
    "SELECT 'prueba_lock_confirmada';" \
    >&9
exec 9>&-
wait "${pid_prueba}"
grep -qx 'prueba_lock_confirmada' "${salida_bloqueo}"

paso 'dos pruebas concurrentes se serializan sin denegación espuria'
entrada_primera="${temporales}/primera-prueba.in"
salida_primera="${temporales}/primera-prueba.out"
salida_segunda="${temporales}/segunda-prueba.out"
mkfifo "${entrada_primera}"
docker exec --interactive "${contenedor}" \
    psql -XAtq --set ON_ERROR_STOP=1 \
    --username vec_consulta_rrhh_runtime --dbname postgres \
    <"${entrada_primera}" >"${salida_primera}" 2>&1 &
pid_primera=$!
exec 9>"${entrada_primera}"
printf '%s\n' \
    "BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;" \
    "SET application_name='prueba_concurrente_primera';" \
    "SET LOCAL TimeZone='UTC';" \
    "SET LOCAL statement_timeout='15s';" \
    "SET LOCAL idle_in_transaction_session_timeout='20s';" \
    "SELECT count(*) FROM vec_contratacion_temporal.prueba_evidencia_consumo_consulta_rrhh_v3('prueba_cuadro','cuadro','');" \
    >&9
esperar_sesion prueba_concurrente_primera 'idle in transaction'
docker exec --interactive "${contenedor}" \
    psql -XAtq --set ON_ERROR_STOP=1 \
    --username vec_consulta_rrhh_runtime --dbname postgres \
    >"${salida_segunda}" 2>&1 <<'SQL' &
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET application_name='prueba_concurrente_segunda';
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT count(*)
  FROM vec_contratacion_temporal
       .prueba_evidencia_consumo_consulta_rrhh_v3(
    'prueba_cuadro','cuadro',''
  );
COMMIT;
SELECT 'segunda_prueba_confirmada';
SQL
pid_segunda=$!
esperar_sesion prueba_concurrente_segunda active Lock
printf '%s\n' \
    "COMMIT;" \
    "SELECT 'primera_prueba_confirmada';" \
    >&9
exec 9>&-
wait "${pid_primera}"
wait "${pid_segunda}"
grep -qx 'primera_prueba_confirmada' "${salida_primera}"
grep -qx 'segunda_prueba_confirmada' "${salida_segunda}"

paso 'cruce de auditoría válida de otra decisión/efecto'
preparar auditoria_a cuadro
preparar auditoria_b cuadro
consumir auditoria_a cuadro >/dev/null
consumir auditoria_b cuadro >/dev/null
esperar_fallo 'auditoría B no prueba el consumo A' \
    docker exec --interactive "${contenedor}" \
    psql -XAtq --set ON_ERROR_STOP=1 --username postgres \
    --dbname postgres <<'SQL'
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
    DISABLE TRIGGER inmutable;
DO $cruce$
DECLARE
    v_b record;
BEGIN
    SELECT auditoria_ref, secuencia, anterior_sha256, huella_sha256
      INTO STRICT v_b
      FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
     WHERE decision_ref = 'decision:consulta-rrhh:auditoria_b';
    UPDATE vec_autorizacion_atestada_v3.auditoria_consumo_v3
       SET auditoria_ref = 'aud_v3_temporal_cruce_prueba',
           secuencia = 9007199254740000,
           huella_sha256 = pg_catalog.repeat('e', 64)
     WHERE decision_ref = 'decision:consulta-rrhh:auditoria_b';
    UPDATE vec_autorizacion_atestada_v3.auditoria_consumo_v3
       SET auditoria_ref = v_b.auditoria_ref,
           secuencia = v_b.secuencia,
           anterior_sha256 = v_b.anterior_sha256,
           huella_sha256 = v_b.huella_sha256
     WHERE decision_ref = 'decision:consulta-rrhh:auditoria_a';
END
$cruce$;
SET SESSION AUTHORIZATION vec_consulta_rrhh_runtime;
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT count(*)
  FROM vec_contratacion_temporal
       .prueba_evidencia_consumo_consulta_rrhh_v3(
    'auditoria_a','cuadro',''
  );
COMMIT;
SQL

# shellcheck source=deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/probar_serializacion_revocaciones_rrhh_v3.sh
source "${raiz}/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/probar_serializacion_revocaciones_rrhh_v3.sh"
probar_serializacion_revocaciones_rrhh_v3

paso 'todas las puertas PostgreSQL 18.4 superadas'
