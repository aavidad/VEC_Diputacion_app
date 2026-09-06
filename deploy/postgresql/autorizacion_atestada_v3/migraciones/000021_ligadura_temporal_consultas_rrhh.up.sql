\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000021', 0));

-- Solo las dos fronteras nominales de consultas RRHH. La decisión escribe
-- seis decimales y la capacidad RFC3339Nano omite ceros finales: se compara
-- el instante sin modificar sus bytes firmados, hashes, MAC ni vigencias.
-- Los hashes anteriores proceden de los cuerpos canónicos de 000003/000005.
-- No reinstalar migraciones históricas, añadir consumidores ni tocar datos.
DO $fechas_consultas$
DECLARE
    v_objetivo record;
    v_antes record;
    v_despues record;
    v_caso record;
    v_divergente boolean;
    v_fecha_textual text := $textual$d ->> 'valida_hasta' <> c ->> 'decision_valida_hasta'$textual$;
    v_fecha_instante text := $instante$(d ->> 'valida_hasta')::timestamptz <> (c ->> 'decision_valida_hasta')::timestamptz$instante$;
BEGIN
    -- Regresión del fragmento exacto a instalar, sin datos de negocio.
    FOR v_caso IN
        SELECT * FROM (VALUES
            ('2026-09-06T02:07:40.999340Z','2026-09-06T02:07:40.99934Z',false),
            ('2026-09-06T02:07:40.000000Z','2026-09-06T02:07:40Z',false),
            ('2026-09-06T02:07:40.999340Z','2026-09-06T02:07:40.999341Z',true)
        ) AS casos(decision_hasta,capacidad_hasta,divergente)
    LOOP
        EXECUTE 'SELECT ' || v_fecha_instante ||
            ' FROM (SELECT $1::jsonb AS d,$2::jsonb AS c) AS material'
            INTO v_divergente
            USING jsonb_build_object('valida_hasta',v_caso.decision_hasta),
                  jsonb_build_object('decision_valida_hasta',v_caso.capacidad_hasta);
        IF v_divergente IS DISTINCT FROM v_caso.divergente THEN
            RAISE EXCEPTION 'regresión de ligadura temporal de consultas RRHH' USING ERRCODE = '55000';
        END IF;
    END LOOP;

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
           OR encode(sha256(convert_to(v_antes.cuerpo,'UTF8')),'hex') IS DISTINCT FROM v_objetivo.sha_anterior
           OR length(v_antes.definicion)-length(replace(v_antes.definicion,v_fecha_textual,'')) <> length(v_fecha_textual)
           OR strpos(v_antes.definicion,v_fecha_instante) <> 0 THEN
            RAISE EXCEPTION 'definición anterior incompatible para ligadura temporal de consultas RRHH' USING ERRCODE = '55000';
        END IF;

        EXECUTE replace(v_antes.definicion,v_fecha_textual,v_fecha_instante);
        SELECT p.prosrc AS cuerpo,p.proacl AS acl,p.proowner AS propietario,
               p.proconfig AS configuracion,p.prosecdef AS definidor
          INTO STRICT v_despues FROM pg_proc p WHERE p.oid = v_antes.oid;
        IF encode(sha256(convert_to(v_despues.cuerpo,'UTF8')),'hex') IS DISTINCT FROM v_objetivo.sha_corregido
           OR v_despues.acl IS DISTINCT FROM v_antes.acl
           OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
           OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
           OR v_despues.definidor IS NOT TRUE THEN
            RAISE EXCEPTION 'ligadura temporal alteró definición o permisos de consultas RRHH' USING ERRCODE = '55000';
        END IF;
    END LOOP;
END
$fechas_consultas$;
COMMIT;
