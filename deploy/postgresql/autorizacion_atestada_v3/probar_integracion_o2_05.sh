#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd -P)"
imagen="${VEC_POSTGRES_TEST_IMAGE:-postgres@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}"
contenedor="vec-o205-pg-${PPID}-${RANDOM}"

limpiar() {
    rm -f "/tmp/o205-${contenedor}-"*.log
    if [[ "${VEC_CONSERVAR_CONTENEDOR_PRUEBA:-}" == '1' ]]; then
        printf 'contenedor conservado para diagnóstico: %s\n' \
            "${contenedor}" >&2
        return
    fi
    docker rm --force --volumes "${contenedor}" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

paso() {
    printf '[O2-05:PostgreSQL] %s\n' "$1"
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
    local caso="$1"
    local variante="${2:-valido}"
    local clave="${3:-1}"
    sql postgres \
        "SELECT public.preparar_vector_o2_05('${caso}','${variante}',${clave})" \
        >/dev/null
}

invocar() {
    local caso="$1"
    docker exec --interactive "${contenedor}" \
        psql -X --set ON_ERROR_STOP=1 --username vec_ct_o205_runtime \
        --dbname postgres <<SQL
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT * FROM public.invocar_vector_o2_05('${caso}');
COMMIT;
SQL
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

paso 'endurecimiento inicial'
docker exec --interactive "${contenedor}" \
    psql -X --set ON_ERROR_STOP=1 --username postgres --dbname postgres <<'SQL'
REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
DO $b$
BEGIN
  IF current_setting('server_version_num')::integer / 10000 <> 18 THEN
    RAISE EXCEPTION 'se exige PostgreSQL 18';
  END IF;
END
$b$;
SQL

paso 'dependencias de contexto de actor y autorización nominal V3'
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
    'CREATE ROLE vec_contexto_o205_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS; GRANT vec_contexto_actor_v1_runtime TO vec_contexto_o205_runtime WITH ADMIN FALSE, INHERIT TRUE, SET FALSE' \
    >/dev/null
docker exec --interactive "${contenedor}" \
    psql -X --set ON_ERROR_STOP=1 \
    --username vec_contexto_o205_runtime --dbname postgres <<'SQL' >/dev/null
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
docker exec --interactive "${contenedor}" \
    psql -X --set ON_ERROR_STOP=1 --username postgres --dbname postgres <<'SQL'
CREATE EXTENSION pgcrypto WITH SCHEMA public;
REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC;
SQL
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
    deploy/postgresql/autorizacion/pruebas_sql/integracion_contexto_actor_v3.sql
do
    archivo postgres "${ruta}" >/dev/null
done

paso 'instalación con identidades de migración separadas'
archivo postgres deploy/postgresql/contratacion_temporal/roles_up.sql >/dev/null
sql postgres \
    'CREATE ROLE vec_ct_o205_migrador LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; GRANT CONNECT ON DATABASE postgres TO vec_ct_o205_migrador; GRANT vec_contratacion_temporal_migrador TO vec_ct_o205_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE' \
    >/dev/null
archivo vec_ct_o205_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000001_preparacion_altas.up.sql \
    >/dev/null
archivo vec_ct_o205_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000002_rotacion_hmac.up.sql \
    >/dev/null
archivo postgres deploy/postgresql/autorizacion_atestada_v3/roles_up.sql \
    >/dev/null
sql postgres \
    'CREATE ROLE vec_ad3_o205_migrador LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; GRANT CONNECT ON DATABASE postgres TO vec_ad3_o205_migrador; GRANT vec_autorizacion_atestada_v3_migrador TO vec_ad3_o205_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE' \
    >/dev/null
archivo vec_ad3_o205_migrador \
    deploy/postgresql/autorizacion_atestada_v3/migraciones/000001_gobierno_y_registro_v3.up.sql \
    >/dev/null
archivo vec_ad3_o205_migrador \
    deploy/postgresql/autorizacion_atestada_v3/migraciones/000002_consumidor_capacidad_v3.up.sql \
    >/dev/null
archivo vec_ct_o205_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000003_expediente_confirmacion_atestada.up.sql \
    >/dev/null
archivo vec_ct_o205_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000004_funcion_confirmar_alta_atestada.up.sql \
    >/dev/null

paso 'vector byte a byte común a Go y PostgreSQL'
docker exec --interactive "${contenedor}" \
    psql -X --set ON_ERROR_STOP=1 --username postgres --dbname postgres <<'SQL'
DO $prueba$
DECLARE
  v bytea := convert_to(rtrim(pg_read_file(
    '/repo/internal/vec/adapters/seguridad/confianzaatestacion/testdata/capacidad_v3_canonica_o2_05.json'
  ), E'\n'), 'UTF8');
  j jsonb;
  v_orden_distinto bytea;
BEGIN
  IF NOT vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(v) THEN
    RAISE EXCEPTION 'PostgreSQL rechazó el vector compartido';
  END IF;
  j := convert_from(v, 'UTF8')::jsonb;
  IF vec_autorizacion_atestada_v3.capacidad_canonica(j) IS DISTINCT FROM v THEN
    RAISE EXCEPTION 'PostgreSQL no reconstruye exactamente el vector';
  END IF;
  IF vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(
       convert_to(left(convert_from(v,'UTF8'),-1) ||
         ',"desconocida":"x"}','UTF8')
     ) THEN
    RAISE EXCEPTION 'clave desconocida aceptada';
  END IF;
  IF vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(
       convert_to(left(convert_from(v,'UTF8'),-1) ||
         ',"esquema":"duplicado"}','UTF8')
     ) THEN
    RAISE EXCEPTION 'clave duplicada aceptada';
  END IF;
  v_orden_distinto := convert_to(j::text, 'UTF8');
  IF NOT vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(
       v_orden_distinto
     ) OR vec_autorizacion_atestada_v3.capacidad_canonica(j) =
          v_orden_distinto THEN
    RAISE EXCEPTION 'orden no canónico no detectado';
  END IF;
END
$prueba$;
SQL

paso 'gobierno sintético y superficie de prueba aislada'
sql postgres \
    'GRANT EXECUTE ON FUNCTION public.gen_random_bytes(integer) TO vec_autorizacion_atestada_v3_propietario' \
    >/dev/null
archivo postgres \
    deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/preparar_entorno_o2_05.sql \
    >/dev/null
archivo postgres \
    deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/ayudantes_o2_05.sql \
    >/dev/null
sql postgres \
    'CREATE ROLE vec_ct_o205_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS; GRANT vec_contratacion_temporal_ejecutor TO vec_ct_o205_runtime WITH ADMIN FALSE, INHERIT TRUE, SET FALSE; GRANT USAGE ON SCHEMA public TO vec_ct_o205_runtime; GRANT EXECUTE ON FUNCTION public.invocar_vector_o2_05(text) TO vec_ct_o205_runtime' \
    >/dev/null

paso 'alta completa, efecto único y replay exacto'
preparar alta_valida
primera="$(invocar alta_valida)"
segunda="$(invocar alta_valida)"
[[ "${primera}" == *'expediente:ct:o205:alta_valida'* ]]
[[ "${segunda}" == *'expediente:ct:o205:alta_valida'* ]]
[[ "$(valor "SELECT (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3 WHERE decision_ref='decision:ct:o205:alta_valida')::text || ':' || (SELECT count(*) FROM vec_contratacion_temporal.expediente_alta WHERE expediente_ref='expediente:ct:o205:alta_valida') || ':' || (SELECT count(*) FROM vec_contratacion_temporal.auditoria_alta WHERE expediente_ref='expediente:ct:o205:alta_valida') || ':' || (SELECT count(*) FROM vec_contratacion_temporal.outbox_alta WHERE expediente_ref='expediente:ct:o205:alta_valida')")" == '1:1:1:1' ]]

paso 'rechazos cruzados, caducidad y alias incoherentes sin efecto parcial'
for variante in efecto_cruzado decision_cruzada expirada alias_cruzado; do
    caso="rechazo_${variante}"
    preparar "${caso}" "${variante}"
    esperar_fallo "${variante}" invocar "${caso}"
    [[ "$(valor "SELECT (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3 WHERE decision_ref='decision:ct:o205:${caso}') + (SELECT count(*) FROM vec_contratacion_temporal.expediente_alta WHERE expediente_ref='expediente:ct:o205:${caso}')")" == '0' ]]
done

paso 'fallo inyectado posterior al consumo: rollback integral'
preparar fallo_atomico
docker exec --interactive "${contenedor}" \
    psql -X --set ON_ERROR_STOP=1 --username postgres \
    --dbname postgres >/dev/null <<'SQL'
CREATE FUNCTION public.fallo_o205()
RETURNS trigger LANGUAGE plpgsql AS $f$
BEGIN
    RAISE EXCEPTION 'fallo inyectado O2-05';
END
$f$;
CREATE TRIGGER fallo_o205
BEFORE INSERT ON vec_contratacion_temporal.outbox_alta
FOR EACH ROW EXECUTE FUNCTION public.fallo_o205();
SQL
esperar_fallo 'rollback después del consumo' invocar fallo_atomico
sql postgres \
    'DROP TRIGGER fallo_o205 ON vec_contratacion_temporal.outbox_alta; DROP FUNCTION public.fallo_o205()' \
    >/dev/null
[[ "$(valor "SELECT (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3 WHERE decision_ref='decision:ct:o205:fallo_atomico') + (SELECT count(*) FROM vec_contratacion_temporal.expediente_alta WHERE expediente_ref='expediente:ct:o205:fallo_atomico')")" == '0' ]]

paso 'concurrencia real sobre la misma capacidad'
preparar carrera
for indice in 1 2 3 4; do
    invocar carrera >"/tmp/o205-${contenedor}-${indice}.log" 2>&1 &
    eval "pid_${indice}=$!"
done
for indice in 1 2 3 4; do
    pid="pid_${indice}"
    if ! wait "${!pid}"; then
        # SERIALIZABLE permite 40001; el cliente abre una transacción nueva.
        invocar carrera >/dev/null
    fi
    rm -f "/tmp/o205-${contenedor}-${indice}.log"
done
[[ "$(valor "SELECT (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3 WHERE decision_ref='decision:ct:o205:carrera')::text || ':' || (SELECT count(*) FROM vec_contratacion_temporal.expediente_alta WHERE expediente_ref='expediente:ct:o205:carrera')")" == '1:1' ]]

paso 'rotación HMAC: retenida válida y revocación efectiva'
docker exec --interactive "${contenedor}" \
    psql -X --set ON_ERROR_STOP=1 --username postgres --dbname postgres <<'SQL'
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
WITH secreto AS (SELECT public.gen_random_bytes(32) valor)
INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version(
 clave_id,version,revision_gobierno,huella_gobierno_sha256,secreto_hmac,
 huella_secreto_sha256,emisor_id,audiencia_consumo,valida_desde,valida_hasta,
 acto_ref
) SELECT 'clave-capacidad-o205-2',2,2,repeat('3',64),valor,
 encode(sha256(valor),'hex'),'broker-o205-sintetico',
 'vec_contratacion_temporal.confirmar_alta_atestada.v1',
 clock_timestamp()-interval '1 minute',clock_timestamp()+interval '2 hours',
 'acto:clave-capacidad:o205:2' FROM secreto;
INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision
 VALUES (2,'clave-capacidad-o205-2',2,clock_timestamp(),
         'acto:puntero-clave:o205:2');
COMMIT;
SQL
preparar clave_retenida valido 1
invocar clave_retenida >/dev/null
preparar clave_activa valido 2
invocar clave_activa >/dev/null
sql postgres \
    "BEGIN; SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario; INSERT INTO vec_autorizacion_atestada_v3.revocacion_clave_capacidad VALUES ('clave-capacidad-o205-1',1,clock_timestamp(),'compromiso_prueba','acto:revocacion-clave:o205:1'); COMMIT" \
    >/dev/null
preparar clave_revocada valido 1
esperar_fallo 'clave HMAC revocada' invocar clave_revocada
sql postgres \
    'REVOKE EXECUTE ON FUNCTION public.gen_random_bytes(integer) FROM vec_autorizacion_atestada_v3_propietario' \
    >/dev/null

paso 'rotación de confianza: anti-rollback y revocaciones'
docker exec --interactive "${contenedor}" \
    psql -X --set ON_ERROR_STOP=1 --username postgres \
    --dbname postgres >/dev/null <<'SQL'
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version
VALUES (
  'configuracion-o205-2',2,repeat('4',64),
  clock_timestamp()-interval '1 minute',
  clock_timestamp()+interval '2 hours',
  'acto:configuracion:o205:2',clock_timestamp()
);
WITH raiz AS (
  SELECT decode(
    '302a300506032b6570032100' || repeat('22',32),'hex'
  ) spki
)
INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version
SELECT 'raiz-o205-2',2,spki,encode(sha256(spki),'hex'),
       clock_timestamp()-interval '1 minute',
       clock_timestamp()+interval '2 hours',
       'VEC-AD-3-COSE-EDDSA-1',
       'vec-diputacion/pruebas/o205/consumidor',
       'acto:raiz:o205:2',clock_timestamp()
  FROM raiz;
INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz
VALUES ('configuracion-o205-2','raiz-o205-2',2);
INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual
VALUES (
  2,'configuracion-o205-2',clock_timestamp(),
  'acto:puntero-configuracion:o205:2',clock_timestamp()
);
COMMIT;
SQL
preparar confianza_activa valido 2
invocar confianza_activa >/dev/null
preparar rollback_config rollback_configuracion 2
esperar_fallo 'rollback de secuencia de configuración' \
    invocar rollback_config
preparar rollback_raiz rollback_raiz 2
esperar_fallo 'rollback de versión de raíz' invocar rollback_raiz
sql postgres \
    "BEGIN; SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario; INSERT INTO vec_autorizacion_atestada_v3.revocacion_raiz VALUES ('raiz-o205-2',2,clock_timestamp(),'compromiso_prueba','acto:revocacion-raiz:o205:2',clock_timestamp()); COMMIT" \
    >/dev/null
preparar raiz_revocada valido 2
esperar_fallo 'raíz de confianza revocada' invocar raiz_revocada
docker exec --interactive "${contenedor}" \
    psql -X --set ON_ERROR_STOP=1 --username postgres \
    --dbname postgres >/dev/null <<'SQL'
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version
VALUES (
  'configuracion-o205-3',3,repeat('5',64),
  clock_timestamp()-interval '1 minute',
  clock_timestamp()+interval '2 hours',
  'acto:configuracion:o205:3',clock_timestamp()
);
WITH raiz AS (
  SELECT decode(
    '302a300506032b6570032100' || repeat('33',32),'hex'
  ) spki
)
INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version
SELECT 'raiz-o205-3',3,spki,encode(sha256(spki),'hex'),
       clock_timestamp()-interval '1 minute',
       clock_timestamp()+interval '2 hours',
       'VEC-AD-3-COSE-EDDSA-1',
       'vec-diputacion/pruebas/o205/consumidor',
       'acto:raiz:o205:3',clock_timestamp()
  FROM raiz;
INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz
VALUES ('configuracion-o205-3','raiz-o205-3',3);
INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual
VALUES (
  3,'configuracion-o205-3',clock_timestamp(),
  'acto:puntero-configuracion:o205:3',clock_timestamp()
);
COMMIT;
SQL
preparar configuracion_activa valido 2
invocar configuracion_activa >/dev/null
sql postgres \
    "BEGIN; SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario; INSERT INTO vec_autorizacion_atestada_v3.revocacion_configuracion VALUES ('configuracion-o205-3',clock_timestamp(),'compromiso_prueba','acto:revocacion-configuracion:o205:3',clock_timestamp()); COMMIT" \
    >/dev/null
preparar configuracion_revocada valido 2
esperar_fallo 'configuración de confianza revocada' \
    invocar configuracion_revocada

paso 'ACL: único mando runtime y denegación por defecto'
esperar_fallo 'lectura directa CT' sql vec_ct_o205_runtime \
    'TABLE vec_contratacion_temporal.expediente_alta'
esperar_fallo 'lectura directa atestación' sql vec_ct_o205_runtime \
    'TABLE vec_autorizacion_atestada_v3.consumo_decision_v3'
esperar_fallo 'preparación histórica abierta' sql vec_ct_o205_runtime \
    "SELECT * FROM vec_contratacion_temporal.preparar_alta_v2('{}'::jsonb)"
esperar_fallo 'consumidor genérico abierto' sql vec_ct_o205_runtime \
    "SELECT * FROM vec_autorizacion_atestada_v3.registrar_y_consumir_decision_v3_atestada(''::bytea,''::bytea,''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea)"
[[ "$(valor "SELECT has_function_privilege('vec_ct_o205_runtime','vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE')")" == 't' ]]
[[ "$(valor "SELECT has_function_privilege('public','vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE')")" == 'f' ]]

paso 'rollback ordinario protegido'
esperar_fallo 'down CT con historia' archivo vec_ct_o205_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000004_funcion_confirmar_alta_atestada.down.sql
esperar_fallo 'down consumidor con historia' archivo vec_ad3_o205_migrador \
    deploy/postgresql/autorizacion_atestada_v3/migraciones/000002_consumidor_capacidad_v3.down.sql

paso 'retirada de la superficie de prueba y destrucción explícita ensayada'
sql postgres \
    'REVOKE USAGE ON SCHEMA public FROM vec_ct_o205_runtime; DROP FUNCTION public.invocar_vector_o2_05(text); DROP FUNCTION public.preparar_vector_o2_05(text,text,numeric); DROP TABLE public.vectores_o2_05' \
    >/dev/null
docker exec \
    --env PGOPTIONS='-c vec.confirmar_destruccion_contratacion_temporal=DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' \
    "${contenedor}" psql -X --set ON_ERROR_STOP=1 \
    --username vec_ct_o205_migrador --dbname postgres \
    --file /repo/deploy/postgresql/contratacion_temporal/migraciones/000004_funcion_confirmar_alta_atestada.down.sql \
    >/dev/null
docker exec \
    --env PGOPTIONS='-c vec.confirmar_destruccion_contratacion_temporal=DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' \
    "${contenedor}" psql -X --set ON_ERROR_STOP=1 \
    --username vec_ct_o205_migrador --dbname postgres \
    --file /repo/deploy/postgresql/contratacion_temporal/migraciones/000003_expediente_confirmacion_atestada.down.sql \
    >/dev/null
docker exec \
    --env PGOPTIONS='-c vec.confirmar_destruccion_autorizacion_atestada_v3=DESTRUIR_AUTORIZACION_ATESTADA_V3_IRREVERSIBLE' \
    "${contenedor}" psql -X --set ON_ERROR_STOP=1 \
    --username vec_ad3_o205_migrador --dbname postgres \
    --file /repo/deploy/postgresql/autorizacion_atestada_v3/migraciones/000002_consumidor_capacidad_v3.down.sql \
    >/dev/null
docker exec \
    --env PGOPTIONS='-c vec.confirmar_destruccion_autorizacion_atestada_v3=DESTRUIR_AUTORIZACION_ATESTADA_V3_IRREVERSIBLE' \
    "${contenedor}" psql -X --set ON_ERROR_STOP=1 \
    --username vec_ad3_o205_migrador --dbname postgres \
    --file /repo/deploy/postgresql/autorizacion_atestada_v3/migraciones/000001_gobierno_y_registro_v3.down.sql \
    >/dev/null
sql postgres \
    'REVOKE CONNECT ON DATABASE postgres FROM vec_ct_o205_migrador, vec_ad3_o205_migrador; DROP ROLE vec_ct_o205_runtime; DROP ROLE vec_ct_o205_migrador; DROP ROLE vec_ad3_o205_migrador' \
    >/dev/null
archivo postgres deploy/postgresql/autorizacion_atestada_v3/roles_down.sql \
    >/dev/null
[[ "$(valor "SELECT count(*) FROM pg_roles WHERE rolname LIKE 'vec_autorizacion_atestada_v3_%'")" == '0' ]]
[[ "$(valor "SELECT (to_regnamespace('vec_autorizacion_atestada_v3') IS NULL)::text")" == 'true' ]]

paso 'OK: autorización atestada, efecto único, atomicidad, ACL y rollback'
