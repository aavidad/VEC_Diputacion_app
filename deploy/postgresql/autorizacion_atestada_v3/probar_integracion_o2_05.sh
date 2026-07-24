#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd -P)"
imagen="${VEC_POSTGRES_TEST_IMAGE:-postgres@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}"
contenedor="vec-o205-pg-${PPID}-${RANDOM}"

verificar_nombres_versiones_migracion_unicas() {
    local ruta nombre version
    local -A archivo_por_version=()

    for ruta in "$@"; do
        nombre="${ruta##*/}"
        if [[ ! "${nombre}" =~ ^([0-9]{6})_ ]]; then
            continue
        fi
        version="${BASH_REMATCH[1]}"
        if [[ -n "${archivo_por_version[${version}]:-}" ]]; then
            printf 'Prefijo de migración duplicado %s: %s y %s\n' \
                "${version}" "${archivo_por_version[${version}]}" \
                "${nombre}" >&2
            return 65
        fi
        archivo_por_version["${version}"]="${nombre}"
    done
}

verificar_versiones_migracion_unicas() {
    local directorio="${raiz}/deploy/postgresql/contratacion_temporal/migraciones"

    verificar_nombres_versiones_migracion_unicas \
        "${directorio}"/*.up.sql
}

verificar_versiones_migracion_unicas
estado_prefijo=0
verificar_nombres_versiones_migracion_unicas \
    /prueba/000001_primera.up.sql /prueba/000001_segunda.up.sql \
    >/dev/null 2>&1 || estado_prefijo=$?
[[ "${estado_prefijo}" -eq 65 ]]
unset estado_prefijo

if [[ ! "${imagen}" =~ ^[^[:space:]@]+@sha256:[0-9a-f]{64}$ ]]; then
    printf 'VEC_POSTGRES_TEST_IMAGE debe fijarse por digest sha256\n' >&2
    exit 64
fi
directorio_temporal="$(mktemp -d -t vec-o205.XXXXXXXX)"
chmod 700 "${directorio_temporal}"

limpiar() {
    rm -f "/tmp/o205-${contenedor}-"*.log
    rm -rf "${directorio_temporal}"
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
        psql -X --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username vec_ct_o205_runtime \
        --dbname postgres <<SQL
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT * FROM public.invocar_vector_o2_05('${caso}');
COMMIT;
SQL
}

recibo_con_estilo_fecha() {
    local caso="$1"
    local estilo="$2"
    docker exec --interactive "${contenedor}" \
        psql -XAtq --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username vec_ct_o205_runtime --dbname postgres <<SQL
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SET LOCAL DateStyle='${estilo}';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT recibo_huella_sha256
  FROM public.invocar_vector_o2_05('${caso}');
COMMIT;
SQL
}

firmar_capacidad_con_go() {
    local caso="$1"
    local entrada="${directorio_temporal}/entrada-go-o205.json"
    local salida="${directorio_temporal}/bundle-go-o205.json"
    valor "SELECT public.exportar_entrada_go_o2_05('${caso}')" \
        >"${entrada}"
    chmod 600 "${entrada}"
    local diagnostico="${directorio_temporal}/go-test-vector.log"
    if ! VEC_O205_VECTOR_ENTRADA="${entrada}" \
        VEC_O205_VECTOR_SALIDA="${salida}" \
            go test \
              ./internal/vec/adapters/seguridad/confianzaatestacion \
              -run '^TestGenerarVectorO205ParaSQL$' -count=1 \
              >"${diagnostico}" 2>&1; then
        printf 'falló generación Go de capacidad O2-05:\n' >&2
        sed -n '1,240p' "${diagnostico}" >&2
        return 1
    fi
    rm -f "${diagnostico}"
    docker cp "${salida}" "${contenedor}:/tmp/capacidad-go-o205.json"
    docker exec "${contenedor}" \
        chmod 644 /tmp/capacidad-go-o205.json
    sql postgres \
        "SELECT public.aplicar_bundle_go_o2_05('${caso}',pg_catalog.pg_read_file('/tmp/capacidad-go-o205.json')::jsonb)" \
        >/dev/null
    rm -f "${entrada}" "${salida}"
}

source "${raiz}/deploy/postgresql/autorizacion_atestada_v3/pruebas_replay_o2_05.sh"
source "${raiz}/deploy/postgresql/autorizacion_atestada_v3/pruebas_acl_o2_05.sh"
source "${raiz}/deploy/postgresql/autorizacion_atestada_v3/pruebas_atomicidad_o2_05.sh"

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
archivo postgres \
    deploy/postgresql/autorizacion/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.up.sql \
    >/dev/null

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
    deploy/postgresql/contratacion_temporal/migraciones/000004_integridad_agregado_alta.up.sql \
    >/dev/null
archivo vec_ct_o205_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000005_funcion_confirmar_alta_atestada.up.sql \
    >/dev/null
archivo vec_ct_o205_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000006_expediente_integral_versionado.up.sql \
    >/dev/null
afirmar_sin_referencias_o2_05

paso 'frontera intercambiable sin FK a tablas de otra autoridad'
[[ "$(valor "SELECT count(*) FROM pg_catalog.pg_constraint c JOIN pg_catalog.pg_class origen ON origen.oid=c.conrelid JOIN pg_catalog.pg_namespace esquema_origen ON esquema_origen.oid=origen.relnamespace JOIN pg_catalog.pg_class destino ON destino.oid=c.confrelid JOIN pg_catalog.pg_namespace esquema_destino ON esquema_destino.oid=destino.relnamespace WHERE c.contype='f' AND esquema_origen.nspname='vec_contratacion_temporal' AND esquema_destino.nspname<>esquema_origen.nspname")" == '0' ]]

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
afirmar_agregado_completo_o2_05 alta_valida
[[ "$(valor "SELECT ((v.agregado_json->>'referencia')='expediente:ct:o205:alta_valida' AND (v.agregado_json->>'version')::numeric=1 AND jsonb_array_length(v.agregado_json->'actuaciones')=1 AND v.agregado_json->'analisis' IS NULL AND a.version=1 AND a.operacion_ref=v.operacion_ref AND encode(sha256(v.prueba_canonica),'hex')=v.prueba_huella_sha256)::text FROM vec_contratacion_temporal.expediente_version_integral v JOIN vec_contratacion_temporal.expediente_integral_actual a USING (expediente_ref) WHERE v.expediente_ref='expediente:ct:o205:alta_valida'")" == 'true' ]]
recibo_iso="$(recibo_con_estilo_fecha alta_valida 'ISO, YMD')"
recibo_aleman="$(recibo_con_estilo_fecha alta_valida 'German, DMY')"
[[ "${recibo_iso}" == "${recibo_aleman}" ]]

probar_canon_cerrado_o2_05

paso 'rechazos cruzados, caducidad y alias incoherentes sin efecto parcial'
for variante in efecto_cruzado decision_cruzada expirada alias_cruzado; do
    caso="rechazo_${variante}"
    preparar "${caso}" "${variante}"
    esperar_fallo "${variante}" invocar "${caso}"
    [[ "$(valor "SELECT (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3 WHERE decision_ref='decision:ct:o205:${caso}') + (SELECT count(*) FROM vec_contratacion_temporal.expediente_alta WHERE expediente_ref='expediente:ct:o205:${caso}')")" == '0' ]]
done
preparar clave_no_activada valido 99
esperar_fallo 'clave provisionada que nunca fue activada' \
    invocar clave_no_activada
[[ "$(valor "SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3 WHERE decision_ref='decision:ct:o205:clave_no_activada'")" == '0' ]]

paso 'gobierno futuro no avanza checkpoint ni bloquea el vigente'
checkpoint_antes="$(valor "SELECT revision::text||':'||configuracion_secuencia_minima::text||':'||raiz_version_minima::text FROM vec_autorizacion_atestada_v3.checkpoint_gobierno WHERE control_id")"
docker exec --interactive "${contenedor}" \
    psql -X --set ON_ERROR_STOP=1 --username postgres \
    --dbname postgres >/dev/null <<'SQL'
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version
VALUES (
  'configuracion-o205-futura',10,repeat('a',64),
  clock_timestamp()+interval '1 day',
  clock_timestamp()+interval '2 days',
  'acto:configuracion:o205:futura',clock_timestamp()
);
WITH raiz AS (
  SELECT decode(
    '302a300506032b6570032100' || repeat('44',32),'hex'
  ) spki
)
INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version
SELECT 'raiz-o205-futura',10,spki,encode(sha256(spki),'hex'),
       clock_timestamp()+interval '1 day',
       clock_timestamp()+interval '2 days',
       'VEC-AD-3-COSE-EDDSA-1',
       'vec-diputacion/pruebas/o205/consumidor',
       'acto:raiz:o205:futura',clock_timestamp()
  FROM raiz;
INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz
VALUES ('configuracion-o205-futura','raiz-o205-futura',10);
INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual
VALUES (
  10,'configuracion-o205-futura',clock_timestamp()+interval '1 day',
  'acto:puntero-configuracion:o205:futura',clock_timestamp()
);
COMMIT;
SQL
[[ "$(valor "SELECT revision::text||':'||configuracion_secuencia_minima::text||':'||raiz_version_minima::text FROM vec_autorizacion_atestada_v3.checkpoint_gobierno WHERE control_id")" == "${checkpoint_antes}" ]]
preparar gobierno_vigente
invocar gobierno_vigente >/dev/null

probar_atomicidad_y_reconciliacion_o2_05

paso 'concurrencia real sobre la misma capacidad'
preparar carrera
cadena_antes="$(valor "SELECT secuencia_auditoria::text||':'||secuencia_outbox::text FROM vec_contratacion_temporal.control_cadenas_alta WHERE control_id")"
for indice in 1 2 3 4; do
    invocar carrera >"/tmp/o205-${contenedor}-${indice}.log" 2>&1 &
    eval "pid_${indice}=$!"
done
for indice in 1 2 3 4; do
    pid="pid_${indice}"
    if ! wait "${!pid}"; then
        if ! grep -Eq 'ERROR:  (40001|40P01):' \
            "/tmp/o205-${contenedor}-${indice}.log"; then
            printf 'fallo concurrente no reintentable:\n' >&2
            sed -n '1,120p' \
                "/tmp/o205-${contenedor}-${indice}.log" >&2
            exit 1
        fi
        invocar carrera \
            >"/tmp/o205-${contenedor}-${indice}-reintento.log" 2>&1
    fi
done
afirmar_agregado_completo_o2_05 carrera
cadena_despues="$(valor "SELECT secuencia_auditoria::text||':'||secuencia_outbox::text FROM vec_contratacion_temporal.control_cadenas_alta WHERE control_id")"
IFS=':' read -r audit_antes outbox_antes <<<"${cadena_antes}"
IFS=':' read -r audit_despues outbox_despues <<<"${cadena_despues}"
[[ "${audit_despues}" -eq $((audit_antes + 1)) ]]
[[ "${outbox_despues}" -eq $((outbox_antes + 1)) ]]
[[ "$(valor "SELECT ((SELECT cabeza_auditoria_sha256 FROM vec_contratacion_temporal.control_cadenas_alta WHERE control_id)=(SELECT huella_sha256 FROM vec_contratacion_temporal.auditoria_alta ORDER BY secuencia DESC LIMIT 1) AND (SELECT cabeza_outbox_sha256 FROM vec_contratacion_temporal.control_cadenas_alta WHERE control_id)=(SELECT huella_sha256 FROM vec_contratacion_temporal.outbox_alta ORDER BY secuencia DESC LIMIT 1))::text")" == 'true' ]]
rm -f "/tmp/o205-${contenedor}-"*.log

probar_integridad_replay_o2_05

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
sql postgres \
    "BEGIN; SET LOCAL session_replication_role='replica'; UPDATE vec_autorizacion_atestada_v3.configuracion_confianza_version SET secuencia=100 WHERE revision='configuracion-o205-1'; UPDATE vec_autorizacion_atestada_v3.configuracion_confianza_version SET secuencia=1 WHERE revision='configuracion-o205-2'; COMMIT" \
    >/dev/null
preparar rollback_config valido 2
[[ "$(valor "SELECT (convert_from(capacidad,'UTF8')::jsonb->>'configuracion_secuencia')::numeric < (SELECT configuracion_secuencia_minima FROM vec_autorizacion_atestada_v3.checkpoint_gobierno WHERE control_id) FROM public.vectores_o2_05 WHERE caso='rollback_config'")" == 't' ]]
esperar_fallo 'rollback de secuencia de configuración' \
    invocar rollback_config
sql postgres \
    "BEGIN; SET LOCAL session_replication_role='replica'; UPDATE vec_autorizacion_atestada_v3.configuracion_confianza_version SET secuencia=2 WHERE revision='configuracion-o205-2'; UPDATE vec_autorizacion_atestada_v3.configuracion_confianza_version SET secuencia=1 WHERE revision='configuracion-o205-1'; COMMIT" \
    >/dev/null
sql postgres \
    "BEGIN; SET LOCAL session_replication_role='replica'; UPDATE vec_autorizacion_atestada_v3.raiz_confianza_version SET version=1 WHERE clave_id IN ('raiz-o205-2') AND version=2; UPDATE vec_autorizacion_atestada_v3.configuracion_raiz SET raiz_version=1 WHERE configuracion_revision='configuracion-o205-2' AND raiz_clave_id IN ('raiz-o205-2'); COMMIT" \
    >/dev/null
preparar rollback_raiz valido 2
[[ "$(valor "SELECT (convert_from(capacidad,'UTF8')::jsonb->>'raiz_version')::numeric < (SELECT raiz_version_minima FROM vec_autorizacion_atestada_v3.checkpoint_gobierno WHERE control_id) FROM public.vectores_o2_05 WHERE caso='rollback_raiz'")" == 't' ]]
esperar_fallo 'rollback de versión de raíz' invocar rollback_raiz
sql postgres \
    "BEGIN; SET LOCAL session_replication_role='replica'; UPDATE vec_autorizacion_atestada_v3.raiz_confianza_version SET version=2 WHERE clave_id IN ('raiz-o205-2') AND version=1; UPDATE vec_autorizacion_atestada_v3.configuracion_raiz SET raiz_version=2 WHERE configuracion_revision='configuracion-o205-2' AND raiz_clave_id IN ('raiz-o205-2'); COMMIT" \
    >/dev/null
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
  'configuracion-o205-3',4,repeat('5',64),
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
  23,'configuracion-o205-3',clock_timestamp(),
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

paso 'capacidad completa emitida por Go y consumida por SQL'
preparar mac_go_real valido 2
firmar_capacidad_con_go mac_go_real
invocar mac_go_real >/dev/null
preparar mac_go_alterado valido 2
firmar_capacidad_con_go mac_go_alterado
sql postgres \
    "UPDATE public.vectores_o2_05 SET capacidad=pg_catalog.set_byte(capacidad,pg_catalog.octet_length(capacidad)-3,CASE pg_catalog.get_byte(capacidad,pg_catalog.octet_length(capacidad)-3) WHEN 48 THEN 49 ELSE 48 END) WHERE caso='mac_go_alterado'" \
    >/dev/null
esperar_fallo 'MAC Go adulterado' invocar mac_go_alterado
[[ "$(valor "SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3 WHERE decision_ref='decision:ct:o205:mac_go_alterado'")" == '0' ]]

paso 'decisión ya durable no elude revocación viva de sesión'
docker exec --interactive "${contenedor}" \
    psql -X --set ON_ERROR_STOP=1 --username postgres \
    --dbname postgres >/dev/null <<'SQL'
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version
VALUES (
  'configuracion-o205-4',7,repeat('6',64),
  clock_timestamp()-interval '1 minute',
  clock_timestamp()+interval '2 hours',
  'acto:configuracion:o205:4',clock_timestamp()
);
WITH raiz AS (
  SELECT decode(
    '302a300506032b6570032100' || repeat('55',32),'hex'
  ) spki
)
INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version
SELECT 'raiz-o205-4',6,spki,encode(sha256(spki),'hex'),
       clock_timestamp()-interval '1 minute',
       clock_timestamp()+interval '2 hours',
       'VEC-AD-3-COSE-EDDSA-1',
       'vec-diputacion/pruebas/o205/consumidor',
       'acto:raiz:o205:4',clock_timestamp()
  FROM raiz;
INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz
VALUES ('configuracion-o205-4','raiz-o205-4',6);
INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual
VALUES (
  (SELECT max(orden)+1
     FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual),
  'configuracion-o205-4',clock_timestamp(),
  'acto:puntero-configuracion:o205:4',clock_timestamp()
);
COMMIT;
SQL
preparar sesion_revocada valido 2
sql postgres \
    "BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE; SELECT public.durabilizar_decision_o2_05('sesion_revocada'); COMMIT" \
    >/dev/null
sql postgres \
    "BEGIN; SET LOCAL ROLE vec_autorizacion_propietario; INSERT INTO vec_autorizacion.control_sesion_v1(control_sesion_ref,revision,sesion_ref,estado,huella_sha256,sesion_revalidada_en,sesion_valida_hasta) VALUES ('cse_registro_v3_0000000000000000000000',2,'ses_registro_v3_0000000000000000000000','revocada',repeat('a',64),clock_timestamp(),clock_timestamp()+interval '2 hours'); UPDATE vec_autorizacion.control_sesion_actual_v1 SET revision=2,actualizada_en=clock_timestamp(),acto_ref='acto:sesion:registro-v3:revocada' WHERE sesion_ref='ses_registro_v3_0000000000000000000000'; COMMIT" \
    >/dev/null
esperar_fallo 'sesión revocada después de decisión durable' \
    invocar sesion_revocada
[[ "$(valor "SELECT (SELECT count(*) FROM vec_autorizacion.decision_concedida_contexto_actor_v3 WHERE decision_ref='decision:ct:o205:sesion_revocada')::text||':'||(SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3 WHERE decision_ref='decision:ct:o205:sesion_revocada')||':'||(SELECT count(*) FROM vec_contratacion_temporal.expediente_alta WHERE expediente_ref='expediente:ct:o205:sesion_revocada')")" == '1:0:0' ]]

paso 'ACL: único mando runtime y denegación por defecto'
esperar_fallo 'lectura directa CT' sql vec_ct_o205_runtime \
    'TABLE vec_contratacion_temporal.expediente_alta'
esperar_fallo 'lectura directa de historia integral' \
    sql vec_ct_o205_runtime \
    'TABLE vec_contratacion_temporal.expediente_version_integral'
esperar_fallo 'lectura directa del puntero integral' \
    sql vec_ct_o205_runtime \
    'TABLE vec_contratacion_temporal.expediente_integral_actual'
esperar_fallo 'lectura directa atestación' sql vec_ct_o205_runtime \
    'TABLE vec_autorizacion_atestada_v3.consumo_decision_v3'
esperar_fallo 'preparación histórica abierta' sql vec_ct_o205_runtime \
    "SELECT * FROM vec_contratacion_temporal.preparar_alta_v2('{}'::jsonb)"
esperar_fallo 'consumidor genérico abierto' sql vec_ct_o205_runtime \
    "SELECT * FROM vec_autorizacion_atestada_v3.registrar_y_consumir_decision_v3_atestada(''::bytea,''::bytea,''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea)"
[[ "$(valor "SELECT has_function_privilege('vec_ct_o205_runtime','vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE')")" == 't' ]]
[[ "$(valor "SELECT has_function_privilege('public','vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE')")" == 'f' ]]

paso 'rollback ordinario protegido'
esperar_fallo 'down historia integral con datos' \
    archivo vec_ct_o205_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000006_expediente_integral_versionado.down.sql
esperar_fallo 'down CT con historia' archivo vec_ct_o205_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000005_funcion_confirmar_alta_atestada.down.sql
esperar_fallo 'down integridad CT con historia' \
    archivo vec_ct_o205_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000004_integridad_agregado_alta.down.sql
esperar_fallo 'down esquema CT con historia' archivo vec_ct_o205_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000003_expediente_confirmacion_atestada.down.sql
esperar_fallo 'down consumidor con historia' archivo vec_ad3_o205_migrador \
    deploy/postgresql/autorizacion_atestada_v3/migraciones/000002_consumidor_capacidad_v3.down.sql
esperar_fallo 'down gobierno provisionado' archivo vec_ad3_o205_migrador \
    deploy/postgresql/autorizacion_atestada_v3/migraciones/000001_gobierno_y_registro_v3.down.sql
esperar_fallo 'down revalidación viva con consumidor instalado' \
    archivo postgres \
    deploy/postgresql/autorizacion/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.down.sql

paso 'retirada de la superficie de prueba y destrucción explícita ensayada'
sql postgres \
    'REVOKE USAGE ON SCHEMA public FROM vec_ct_o205_runtime; DROP FUNCTION public.aplicar_bundle_go_o2_05(text,jsonb); DROP FUNCTION public.exportar_entrada_go_o2_05(text); DROP FUNCTION public.durabilizar_decision_o2_05(text); DROP FUNCTION public.mutar_tipo_capacidad_o2_05(text,text); DROP FUNCTION public.mutar_efecto_o2_05(text,text,jsonb); DROP FUNCTION public.invocar_vector_o2_05(text); DROP FUNCTION public.preparar_vector_o2_05(text,text,numeric); DROP TABLE public.vectores_o2_05' \
    >/dev/null
docker exec \
    --env PGOPTIONS='-c vec.confirmar_destruccion_contratacion_temporal=DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' \
    "${contenedor}" psql -X --set ON_ERROR_STOP=1 \
    --username vec_ct_o205_migrador --dbname postgres \
    --file /repo/deploy/postgresql/contratacion_temporal/migraciones/000006_expediente_integral_versionado.down.sql \
    >/dev/null
docker exec \
    --env PGOPTIONS='-c vec.confirmar_destruccion_contratacion_temporal=DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' \
    "${contenedor}" psql -X --set ON_ERROR_STOP=1 \
    --username vec_ct_o205_migrador --dbname postgres \
    --file /repo/deploy/postgresql/contratacion_temporal/migraciones/000005_funcion_confirmar_alta_atestada.down.sql \
    >/dev/null
docker exec \
    --env PGOPTIONS='-c vec.confirmar_destruccion_contratacion_temporal=DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' \
    "${contenedor}" psql -X --set ON_ERROR_STOP=1 \
    --username vec_ct_o205_migrador --dbname postgres \
    --file /repo/deploy/postgresql/contratacion_temporal/migraciones/000004_integridad_agregado_alta.down.sql \
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

paso 'reinstalación limpia y segunda retirada completa'
sql postgres \
    'REVOKE vec_autorizacion_atestada_v3_migrador FROM vec_ad3_o205_migrador; REVOKE CONNECT ON DATABASE postgres FROM vec_ad3_o205_migrador' \
    >/dev/null
archivo postgres deploy/postgresql/autorizacion_atestada_v3/roles_down.sql \
    >/dev/null
archivo postgres deploy/postgresql/autorizacion_atestada_v3/roles_up.sql \
    >/dev/null
sql postgres \
    'GRANT CONNECT ON DATABASE postgres TO vec_ad3_o205_migrador; GRANT vec_autorizacion_atestada_v3_migrador TO vec_ad3_o205_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE' \
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
    deploy/postgresql/contratacion_temporal/migraciones/000004_integridad_agregado_alta.up.sql \
    >/dev/null
archivo vec_ct_o205_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000005_funcion_confirmar_alta_atestada.up.sql \
    >/dev/null
archivo vec_ct_o205_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000006_expediente_integral_versionado.up.sql \
    >/dev/null
afirmar_sin_referencias_o2_05
archivo vec_ct_o205_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000006_expediente_integral_versionado.down.sql \
    >/dev/null
archivo vec_ct_o205_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000005_funcion_confirmar_alta_atestada.down.sql \
    >/dev/null
archivo vec_ct_o205_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000004_integridad_agregado_alta.down.sql \
    >/dev/null
archivo vec_ct_o205_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000003_expediente_confirmacion_atestada.down.sql \
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
archivo postgres \
    deploy/postgresql/autorizacion/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.down.sql \
    >/dev/null
[[ "$(valor "SELECT count(*) FROM pg_roles WHERE rolname LIKE 'vec_autorizacion_atestada_v3_%'")" == '0' ]]
[[ "$(valor "SELECT (to_regnamespace('vec_autorizacion_atestada_v3') IS NULL)::text")" == 'true' ]]
[[ "$(valor "SELECT (to_regprocedure('vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)') IS NULL)::text")" == 'true' ]]
[[ "$(valor "SELECT count(*) FROM pg_default_acl d JOIN pg_roles r ON r.oid=d.defaclrole WHERE r.rolname LIKE 'vec_autorizacion_atestada_v3_%'")" == '0' ]]
[[ "$(valor "SELECT count(*) FROM pg_auth_members m JOIN pg_roles g ON g.oid=m.roleid JOIN pg_roles u ON u.oid=m.member WHERE g.rolname LIKE 'vec_autorizacion_atestada_v3_%' OR u.rolname LIKE 'vec_autorizacion_atestada_v3_%'")" == '0' ]]

paso 'OK: autorización atestada, efecto único, atomicidad, ACL y rollback'
