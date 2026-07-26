#!/usr/bin/env bash
# shellcheck disable=SC2154

# Cargado por probar_integracion_o2_05.sh tras instalar la migración 000016.

probar_lectura_analisis_durable_o4() {
    local ausencias resultado version_actual version_incorrecta
    version_actual="$(
        valor "SELECT version::text
          FROM vec_contratacion_temporal.expediente_integral_actual
         WHERE expediente_ref='expediente:ct:o205:alta_valida'"
    )"
    [[ "${version_actual}" =~ ^[2-9][0-9]*$ ]]
    version_incorrecta="$((version_actual + 1))"
    resultado="$(
        docker exec "${contenedor}" psql -XAtq --set ON_ERROR_STOP=1 \
            --username vec_ct_o205_runtime --dbname postgres --command \
            "BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ ONLY;
             SET LOCAL search_path='pg_catalog';
             SET LOCAL timezone='UTC';
             SET LOCAL statement_timeout='15s';
             SET LOCAL idle_in_transaction_session_timeout='20s';
             SELECT pg_catalog.jsonb_build_object(
               'referencia', expediente_json::jsonb->>'referencia',
               'organizacion', expediente_json::jsonb->>'organizacion_ref',
               'version', expediente_json::jsonb->>'version',
               'recibo',
                 expediente_json::jsonb
                   #>>'{analisis,actuacion_registro,recibo_ref}',
               'huella', analisis_huella_sha256
             )::text
               FROM vec_contratacion_temporal
                 .leer_expediente_analisis_durable_o3_v1(
                   'organizacion:dipgra',
                   'expediente:ct:o205:alta_valida',
                   ${version_actual}
                 );
             COMMIT"
    )"
    [[ "${resultado}" == *"\"version\": \"${version_actual}\""* &&
       "${resultado}" == *'"organizacion": "organizacion:dipgra"'* &&
       "${resultado}" == *'"referencia": "expediente:ct:o205:alta_valida"'* &&
       "${resultado}" =~ \"recibo\":\ \"rec_ct_an_[0-9a-f]{32}\" &&
       "${resultado}" =~ \"huella\":\ \"[0-9a-f]{64}\" ]]

    ausencias="$(
        docker exec "${contenedor}" psql -XAtq --set ON_ERROR_STOP=1 \
            --username vec_ct_o205_runtime --dbname postgres --command \
            "BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ ONLY;
             SET LOCAL statement_timeout='15s';
             SET LOCAL idle_in_transaction_session_timeout='20s';
             SELECT (
               (SELECT count(*)
                  FROM vec_contratacion_temporal
                    .leer_expediente_analisis_durable_o3_v1(
                      'organizacion:otra',
                      'expediente:ct:o205:alta_valida',
                      ${version_actual}
                    )) = 0
               AND
               (SELECT count(*)
                  FROM vec_contratacion_temporal
                    .leer_expediente_analisis_durable_o3_v1(
                      'organizacion:dipgra',
                      'expediente:ct:o205:alta_valida',
                      ${version_incorrecta}
                    )) = 0
             )::text;
             COMMIT"
    )"
    [[ "${ausencias}" == 'true' ]]

    probar_recorrido_go_lectura_analisis_durable_o4 "${version_actual}"
}

probar_recorrido_go_lectura_analisis_durable_o4() {
    local version="$1"
    local binario="${directorio_temporal}/lector-analisis-o4.test"
    local diagnostico="${directorio_temporal}/lector-analisis-o4.log"
    go test -c ./internal/modules/contrataciontemporal/adapters/postgres \
        -o "${binario}"
    docker cp "${binario}" "${contenedor}:/tmp/lector-analisis-o4.test" \
        >/dev/null
    if ! docker exec \
        --env 'VEC_O4_LECTOR_DSN=postgres://vec_ct_o205_runtime@/postgres?host=/var/run/postgresql&sslmode=disable' \
        --env "VEC_O4_LECTOR_VERSION=${version}" \
        "${contenedor}" /tmp/lector-analisis-o4.test \
        -test.run \
        '^TestLectorExpedienteAnalisisDurableO3PostgreSQLReal$' \
        -test.count=1 -test.v >"${diagnostico}" 2>&1; then
        sed -n '1,200p' "${diagnostico}" >&2
        return 1
    fi
}

probar_acl_lectura_analisis_durable_o4() {
    [[ "$(valor "SELECT (
      p.proowner = 'vec_contratacion_temporal_propietario'::regrole
      AND p.prosecdef
      AND p.provolatile = 's'
      AND p.proconfig @> ARRAY[
        'search_path=pg_catalog',
        'row_security=on',
        'TimeZone=UTC',
        'lock_timeout=2s'
      ]::text[]
    )::text
      FROM pg_catalog.pg_proc p
     WHERE p.oid =
       'vec_contratacion_temporal.leer_expediente_analisis_durable_o3_v1(text,text,numeric)'::regprocedure")" == 'true' ]]
    [[ "$(valor "SELECT has_function_privilege(
      'vec_ct_o205_runtime',
      'vec_contratacion_temporal.leer_expediente_analisis_durable_o3_v1(text,text,numeric)',
      'EXECUTE'
    )::text")" == 'true' ]]
    [[ "$(valor "SELECT has_function_privilege(
      'public',
      'vec_contratacion_temporal.leer_expediente_analisis_durable_o3_v1(text,text,numeric)',
      'EXECUTE'
    )::text")" == 'false' ]]
    esperar_fallo 'lector O3 fuera de transacción de solo lectura' \
        sql vec_ct_o205_runtime \
        "BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
         SET LOCAL statement_timeout='15s';
         SET LOCAL idle_in_transaction_session_timeout='20s';
         SELECT * FROM vec_contratacion_temporal
           .leer_expediente_analisis_durable_o3_v1(
             'organizacion:dipgra',
             'expediente:ct:o205:alta_valida',
             2
           );
         COMMIT"
}
