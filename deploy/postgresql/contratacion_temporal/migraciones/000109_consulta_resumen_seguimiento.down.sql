\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000109', 0)
);

-- Solo sirve para una instalación desechable sin historia CT publicada.
-- No ejecutar en principal ni retirar una guarda de un circuito con datos.
DO $prevalidacion$
DECLARE
    v_detalle oid :=
        'vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    v_resumen oid :=
        'vec_contratacion_temporal.consultar_resumen_seguimiento_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.publicacion_version_rrhh)
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_proc p
            WHERE p.oid = v_detalle
              AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole
              AND pg_catalog.octet_length(p.prosrc) = 11840
              AND pg_catalog.encode(pg_catalog.sha256(
                  pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') =
                  '79ce26f53f4ac7d08c5f93405f6d8e266f40f41f5662064edd48b237728890fa'
       )
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_proc p
            WHERE p.oid = v_resumen
              AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole
              AND pg_catalog.octet_length(p.prosrc) = 11443
              AND pg_catalog.encode(pg_catalog.sha256(
                  pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') =
                  '73ae49630a485ebfb93677ca0636ff7166e9d69650c13b042bf2de22e889f1fb'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'historia o preimagen impide retirar consulta de seguimiento';
    END IF;
END
$prevalidacion$;

DROP FUNCTION vec_contratacion_temporal.consultar_resumen_seguimiento_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    text, bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) RESTRICT;

DO $restaurar_detalle$
DECLARE
    v_funcion oid :=
        'vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    v_definicion text;
    v_guardia text :=
        '    IF v_decision ->> ''garantia_minima'' IS DISTINCT FROM ''alto'''
        || E'\n' || '       OR v_decision -> ''campos_permitidos'' IS DISTINCT FROM ''[]''::jsonb'
        || E'\n' || '       OR pg_catalog.left(COALESCE(v_decision ->> ''version_rol_ref'', ''''), 44) ='
        || E'\n' || '          ''rol:rrhh_interno_certificado_seguimiento_ct_'' THEN'
        || E'\n' || '        RAISE EXCEPTION USING ERRCODE = ''42501'','
        || E'\n' || '            MESSAGE = ''consulta RRHH rechazada'';'
        || E'\n' || '    END IF;'
        || E'\n\n';
BEGIN
    SELECT pg_catalog.pg_get_functiondef(v_funcion) INTO STRICT v_definicion;
    IF pg_catalog.length(v_definicion) -
       pg_catalog.length(pg_catalog.replace(v_definicion,v_guardia,''))
       <> pg_catalog.length(v_guardia) THEN
        RAISE EXCEPTION USING ERRCODE='55000',
            MESSAGE='guarda CT109 inesperada';
    END IF;
    EXECUTE pg_catalog.replace(v_definicion,v_guardia,'');
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=v_funcion
          AND pg_catalog.encode(pg_catalog.sha256(
              pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') =
              '7b7d6c4a419262d54ddb7f2a096e5a4d6e1c962bc58e3027546e221717ed1814'
    ) THEN
        RAISE EXCEPTION USING ERRCODE='55000',
            MESSAGE='restauración CT45 no exacta';
    END IF;
END
$restaurar_detalle$;
COMMIT;
