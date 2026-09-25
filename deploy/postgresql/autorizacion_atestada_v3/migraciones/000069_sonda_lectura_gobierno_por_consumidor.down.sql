\set ON_ERROR_STOP on
-- AD3-69 DOWN: retira solo las dos funciones de lectura por consumidor. No
-- guardan estado ni historia; AD3-50a y AD3-53a quedan intactas. Antes de
-- ejecutarlo, ningún binario desplegado debe esperar estas firmas en su
-- manifiesto de ACL del perfil de preflight (arrancaría fallando cerrado).
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion_atestada_v3:migracion:000069', 0));

DO $preimagen$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2(text,jsonb)') IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.leer_configuracion_interna_v2(text,jsonb)') IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(jsonb)') IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(jsonb)') IS NULL
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_proc AS p
             JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
             JOIN pg_catalog.pg_roles AS r ON r.oid = p.proowner
            WHERE n.nspname = 'vec_autorizacion_atestada_v3'
              AND p.proname IN ('comprobar_material_emision_interna_v2',
                                'leer_configuracion_interna_v2')
              AND r.rolname <> 'vec_autorizacion_atestada_v3_propietario')
       -- Ningún objeto puede depender de estas funciones.
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_depend AS d
            WHERE d.refclassid = 'pg_catalog.pg_proc'::regclass
              AND d.refobjid IN (
                  pg_catalog.to_regprocedure(
                      'vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2(text,jsonb)'),
                  pg_catalog.to_regprocedure(
                      'vec_autorizacion_atestada_v3.leer_configuracion_interna_v2(text,jsonb)'))
              AND d.deptype <> 'i')
    THEN
        RAISE EXCEPTION 'AD3-69: preimagen de retirada incompatible' USING ERRCODE = '55000';
    END IF;
END $preimagen$;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DROP FUNCTION vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2(text,jsonb) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.leer_configuracion_interna_v2(text,jsonb) RESTRICT;
COMMIT;
