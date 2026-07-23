#!/usr/bin/env bash

# Cargado por probar_integracion_o2_05.sh; reutiliza sus ayudantes y contenedor.
afirmar_sin_referencias_o2_05() {
    local tabla='vec_autorizacion_atestada_v3.consumo_decision_v3'
    local rol='vec_contratacion_temporal_propietario'

    [[ "$(valor "SELECT has_table_privilege('${rol}','${tabla}','REFERENCES')::text")" == 'false' ]]
    [[ "$(valor "SELECT bool_and(NOT has_column_privilege('${rol}','${tabla}',columna,'REFERENCES'))::text FROM unnest(ARRAY['decision_ref','efecto_ref','huella_efecto_sha256']) columna")" == 'true' ]]
    [[ "$(valor "SELECT count(*) FROM pg_catalog.pg_default_acl d CROSS JOIN LATERAL pg_catalog.aclexplode(d.defaclacl) permiso JOIN pg_catalog.pg_roles concedido ON concedido.oid=permiso.grantee WHERE concedido.rolname='${rol}' AND permiso.privilege_type='REFERENCES'")" == '0' ]]

    esperar_fallo \
        'acoplamiento FK desde contratación temporal a autorización' \
        sql postgres \
        "BEGIN; SET LOCAL ROLE ${rol}; CREATE TABLE vec_contratacion_temporal.prueba_acoplamiento_o205(decision_ref text REFERENCES ${tabla}(decision_ref)); ROLLBACK"
    [[ "$(valor "SELECT (to_regclass('vec_contratacion_temporal.prueba_acoplamiento_o205') IS NULL)::text")" == 'true' ]]
}
