\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000021', 0));

-- Reversión exacta de los cuerpos previos: no borra consumos, accesos,
-- auditorías ni ninguna historia. Restaura también el defecto textual;
-- solo para reversión controlada, nunca como recuperación de una consulta.
-- Si otro corte ha cambiado cualquiera de los cuerpos, rechazar sin pisarlo.
DO $fechas_consultas_reversa$
DECLARE
    v_objetivo record;
    v_antes record;
    v_despues record;
    v_fecha_textual text := $textual$d ->> 'valida_hasta' <> c ->> 'decision_valida_hasta'$textual$;
    v_fecha_instante text := $instante$(d ->> 'valida_hasta')::timestamptz <> (c ->> 'decision_valida_hasta')::timestamptz$instante$;
BEGIN
    FOR v_objetivo IN
        SELECT * FROM (VALUES
            ('vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
             '100eb91e9db337dd3c986e5d1137e9ef58105972c6d104ca28c919f6f8043b3f',
             '1cec6ba3faa9d25607273638e458d76dd5f7e1eca0373754d1a2e3f28c6fa137'),
            ('vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
             'd3f72e15374a572dd6004193fc136369d3d5a42ec05e38e8afcd1868d8d8c553',
             '7cfc002cff8878fc36288fa1200de1c51965e9ae4d84b4ffe6179bc62372b5ff')
        ) AS objetivos(firma,sha_anterior,sha_corregido)
    LOOP
        SELECT p.oid,pg_get_functiondef(p.oid) AS definicion,p.prosrc AS cuerpo,
               p.proacl AS acl,p.proowner AS propietario,p.proconfig AS configuracion
          INTO v_antes
          FROM pg_proc p
         WHERE p.oid = to_regprocedure(v_objetivo.firma)
           AND p.proowner = 'vec_autorizacion_atestada_v3_propietario'::regrole
           AND p.prosecdef;
        IF NOT FOUND
           OR encode(sha256(convert_to(v_antes.cuerpo,'UTF8')),'hex') IS DISTINCT FROM v_objetivo.sha_corregido
           OR length(v_antes.definicion)-length(replace(v_antes.definicion,v_fecha_instante,'')) <> length(v_fecha_instante)
           OR strpos(v_antes.definicion,v_fecha_textual) <> 0 THEN
            RAISE EXCEPTION 'definición corregida incompatible para revertir ligadura temporal de consultas RRHH' USING ERRCODE = '55000';
        END IF;

        EXECUTE replace(v_antes.definicion,v_fecha_instante,v_fecha_textual);
        SELECT p.prosrc AS cuerpo,p.proacl AS acl,p.proowner AS propietario,
               p.proconfig AS configuracion,p.prosecdef AS definidor
          INTO STRICT v_despues FROM pg_proc p WHERE p.oid = v_antes.oid;
        IF encode(sha256(convert_to(v_despues.cuerpo,'UTF8')),'hex') IS DISTINCT FROM v_objetivo.sha_anterior
           OR v_despues.acl IS DISTINCT FROM v_antes.acl
           OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
           OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
           OR v_despues.definidor IS NOT TRUE THEN
            RAISE EXCEPTION 'reversión temporal alteró definición o permisos de consultas RRHH' USING ERRCODE = '55000';
        END IF;
    END LOOP;
END
$fechas_consultas_reversa$;
COMMIT;
