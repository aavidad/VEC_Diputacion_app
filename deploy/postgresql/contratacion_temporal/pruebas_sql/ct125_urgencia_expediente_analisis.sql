\set ON_ERROR_STOP on
-- CT125: urgencia declarada en el análisis y fachada v4 del cuadro RRHH.
-- Se ejecuta como superusuario del ensayo desechable, con los roles reales
-- (SET ROLE) para las llamadas y las comprobaciones de ACL. Cada «ok» es una
-- comprobación superada; cualquier fallo aborta el guion.

-- Un análisis confirmado del volcado y otro distinto.
SELECT r.recibo_ref AS recibo, r.ambito_raiz_hmac AS ambito,
       r.expediente_ref AS expediente, (r.version_expediente + 1)::text AS version
  FROM vec_contratacion_temporal.reserva_operacion_analisis r
  JOIN vec_contratacion_temporal.confirmacion_operacion_analisis c USING (ambito_raiz_hmac)
 ORDER BY r.recibo_ref LIMIT 1 \gset
SELECT r.recibo_ref AS recibo_ajeno
  FROM vec_contratacion_temporal.reserva_operacion_analisis r
  JOIN vec_contratacion_temporal.confirmacion_operacion_analisis c USING (ambito_raiz_hmac)
 WHERE r.recibo_ref <> :'recibo' ORDER BY r.recibo_ref LIMIT 1 \gset
SELECT pg_catalog.set_config('ct125.recibo', :'recibo', false),
       pg_catalog.set_config('ct125.ajeno', :'recibo_ajeno', false),
       pg_catalog.set_config('ct125.expediente', :'expediente', false),
       pg_catalog.set_config('ct125.version', :'version', false) \gset

-- 1. Un análisis confirmado en otra transacción no admite urgencia.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_ejecutor;
DO $$ BEGIN
    PERFORM vec_contratacion_temporal.registrar_urgencia_analisis_v1(
        current_setting('ct125.ajeno'), 'Motivo fuera de su confirmación');
    RAISE EXCEPTION 'FALLO: urgencia fuera de la confirmación admitida';
EXCEPTION WHEN insufficient_privilege THEN NULL; END $$;
ROLLBACK;
SELECT 'ok';

-- 2. Recibo inexistente y motivos mal formados.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_ejecutor;
DO $$ BEGIN
    PERFORM vec_contratacion_temporal.registrar_urgencia_analisis_v1('rec_ct_an_inexistente', 'Motivo');
    RAISE EXCEPTION 'FALLO: recibo inexistente admitido';
EXCEPTION WHEN insufficient_privilege THEN NULL; END $$;
DO $$
DECLARE v_motivo text;
BEGIN
    FOREACH v_motivo IN ARRAY ARRAY['', ' con espacios ', pg_catalog.repeat('x', 1001), 'control' || pg_catalog.chr(7)] LOOP
        BEGIN
            PERFORM vec_contratacion_temporal.registrar_urgencia_analisis_v1(current_setting('ct125.recibo'), v_motivo);
            RAISE EXCEPTION 'FALLO: motivo inválido admitido: %', v_motivo;
        EXCEPTION WHEN invalid_parameter_value THEN NULL;
        END;
    END LOOP;
END $$;
ROLLBACK;
SELECT 'ok';

-- 3. ACL: la tabla no se lee ni se escribe directamente; solo el ejecutor
--    registra y solo el consultor RRHH lee la fachada v4.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_ejecutor;
DO $$ BEGIN
    PERFORM count(*) FROM vec_contratacion_temporal.urgencia_expediente_analisis;
    RAISE EXCEPTION 'FALLO: el ejecutor lee la tabla';
EXCEPTION WHEN insufficient_privilege THEN NULL; END $$;
DO $$ BEGIN
    INSERT INTO vec_contratacion_temporal.urgencia_expediente_analisis
    VALUES (current_setting('ct125.expediente'), current_setting('ct125.version')::numeric,
            current_setting('ct125.recibo'), 'Directo', pg_catalog.date_trunc('microseconds', pg_catalog.now()));
    RAISE EXCEPTION 'FALLO: el ejecutor escribe la tabla';
EXCEPTION WHEN insufficient_privilege THEN NULL; END $$;
ROLLBACK;
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_consultor_rrhh;
DO $$ BEGIN
    PERFORM vec_contratacion_temporal.registrar_urgencia_analisis_v1(current_setting('ct125.recibo'), 'Motivo');
    RAISE EXCEPTION 'FALLO: el consultor registra urgencias';
EXCEPTION WHEN insufficient_privilege THEN NULL; END $$;
ROLLBACK;
SELECT CASE WHEN has_function_privilege('vec_contratacion_temporal_consultor_rrhh',
         'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'EXECUTE')
       AND NOT has_function_privilege('vec_contratacion_temporal_ejecutor',
         'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'EXECUTE')
       AND NOT EXISTS (SELECT 1 FROM pg_proc p, aclexplode(p.proacl) a
                        WHERE p.pronamespace = 'vec_contratacion_temporal'::regnamespace
                          AND p.proname IN ('registrar_urgencia_analisis_v1', 'consultar_cuadro_rrhh_atestado_v4')
                          AND a.grantee = 0)
       THEN 'ok' ELSE 'FALLO: ACL de CT125' END;

-- 4. En la transacción que confirma el análisis se registra. Se reproduce
--    la confirmación reescribiendo su fila idéntica en esta transacción.
BEGIN;
SET LOCAL session_replication_role = replica;
CREATE TEMP TABLE ct125_confirmacion ON COMMIT DROP AS
SELECT * FROM vec_contratacion_temporal.confirmacion_operacion_analisis
 WHERE ambito_raiz_hmac = :'ambito';
DELETE FROM vec_contratacion_temporal.confirmacion_operacion_analisis
 WHERE ambito_raiz_hmac = :'ambito';
INSERT INTO vec_contratacion_temporal.confirmacion_operacion_analisis
SELECT * FROM ct125_confirmacion;
SET LOCAL session_replication_role = origin;
SET LOCAL ROLE vec_contratacion_temporal_ejecutor;
DO $$ BEGIN
    IF vec_contratacion_temporal.registrar_urgencia_analisis_v1(
           current_setting('ct125.recibo'), 'Cierre del servicio de ayuda a domicilio de Loja')
       IS DISTINCT FROM 'registrada' THEN
        RAISE EXCEPTION 'FALLO: urgencia no registrada';
    END IF;
    IF vec_contratacion_temporal.registrar_urgencia_analisis_v1(
           current_setting('ct125.recibo'), 'Cierre del servicio de ayuda a domicilio de Loja')
       IS DISTINCT FROM 'repetida' THEN
        RAISE EXCEPTION 'FALLO: la repetición no devuelve repetida';
    END IF;
END $$;
COMMIT;
SELECT CASE WHEN (SELECT count(*) FROM vec_contratacion_temporal.urgencia_expediente_analisis
                   WHERE recibo_ref = :'recibo' AND expediente_ref = :'expediente'
                     AND version = :'version'::numeric) = 1
       THEN 'ok' ELSE 'FALLO: fila de urgencia' END;

-- 5. Otra transacción: el mismo motivo es «repetida»; otro motivo, conflicto.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_ejecutor;
DO $$ BEGIN
    IF vec_contratacion_temporal.registrar_urgencia_analisis_v1(
           current_setting('ct125.recibo'), 'Cierre del servicio de ayuda a domicilio de Loja')
       IS DISTINCT FROM 'repetida' THEN
        RAISE EXCEPTION 'FALLO: repetición posterior';
    END IF;
    BEGIN
        PERFORM vec_contratacion_temporal.registrar_urgencia_analisis_v1(
            current_setting('ct125.recibo'), 'Otro motivo distinto');
        RAISE EXCEPTION 'FALLO: otro motivo admitido';
    EXCEPTION WHEN unique_violation THEN NULL;
    END;
END $$;
ROLLBACK;
SELECT 'ok';

-- 6. Solo adición: ni el propietario la modifica ni la borra.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $$ BEGIN
    UPDATE vec_contratacion_temporal.urgencia_expediente_analisis SET motivo = 'Cambiado';
    RAISE EXCEPTION 'FALLO: la urgencia se modifica';
EXCEPTION WHEN OTHERS THEN
    IF SQLERRM LIKE 'FALLO%' THEN RAISE; END IF;
END $$;
DO $$ BEGIN
    DELETE FROM vec_contratacion_temporal.urgencia_expediente_analisis;
    RAISE EXCEPTION 'FALLO: la urgencia se borra';
EXCEPTION WHEN OTHERS THEN
    IF SQLERRM LIKE 'FALLO%' THEN RAISE; END IF;
END $$;
ROLLBACK;
SELECT 'ok';

-- 7. Fachada v4 con un doble explícito de la v3 (su autorización atestada
--    tiene su propio ensayo): la urgencia se alinea con el contenido
--    canónico, vale desde la versión que la declaró y no para otras.
SELECT pg_catalog.set_config('ct125.contenido',
    'VEC-CT-CONTENIDO-CUADRO-RRHH-V1' || chr(10)
    || (SELECT string_agg(octet_length(convert_to(v, 'UTF8'))::text || ':' || v || chr(10), '' ORDER BY n)
          FROM unnest(ARRAY['2026-09-26T08:00:00Z', '3']
                || ARRAY[current_setting('ct125.expediente'), 'x', 'x', current_setting('ct125.version')] || array_fill('x'::text, ARRAY[11])
                || ARRAY[current_setting('ct125.expediente'), 'x', 'x', (current_setting('ct125.version')::numeric - 1)::text] || array_fill('x'::text, ARRAY[11])
                || ARRAY['expediente:ct:sin-urgencia', 'x', 'x', '7'] || array_fill('x'::text, ARRAY[11])
               ) WITH ORDINALITY AS m(v, n)),
    false) \gset
BEGIN;
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v3(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    p_capacidad_canonica bytea, p_decision_canonica bytea,
    p_motivo_canonico bytea, p_contexto_actor_canonico bytea,
    p_persona_version numeric, p_perfil_version numeric,
    p_payload_vec_ad_3 bytea, p_sobre_cose_sign_1 bytea,
    p_evidencia_verificacion bytea, p_raiz_publica_spki bytea
)
RETURNS TABLE(
    contenido_canonico bytea, cursor_siguiente text, esquema text,
    acceso_ref text, secuencia numeric, anterior_sha256 text,
    huella_sha256 text, vinculo_identidad_huella_sha256 text,
    alcance_huella_sha256 text, registrada_en timestamptz,
    auditoria_vec_ref text, auditoria_vec_huella_sha256 text,
    consumo_vec_huella_sha256 text, contenido_huella_sha256 text,
    resultado_huella_sha256 text, cursor_huella_sha256 text,
    generada_en timestamptz, expediente_ref text,
    version_expediente numeric, total smallint, recibo_sello_sha256 text,
    total_filtrado numeric, en_tramitacion numeric,
    con_incidencia numeric, en_llamamiento numeric,
    fase_desde_expedientes text[], fase_desde_instantes timestamptz[]
)
LANGUAGE sql AS $doble$
    SELECT convert_to(current_setting('ct125.contenido'), 'UTF8'), NULL::text, 'doble'::text,
           'acceso'::text, 1::numeric, NULL::text, 'h'::text, 'v'::text, 'a'::text, now(),
           'aud'::text, 'ah'::text, 'ch'::text, 'coh'::text, 'rh'::text, 'cuh'::text, now(),
           NULL::text, NULL::numeric, 3::smallint, 's'::text, 3::numeric, 3::numeric,
           0::numeric, 0::numeric,
           ARRAY[current_setting('ct125.expediente'), current_setting('ct125.expediente'), 'expediente:ct:sin-urgencia'],
           ARRAY[now(), now(), now()]
$doble$;
SET LOCAL ROLE vec_contratacion_temporal_consultor_rrhh;
SELECT CASE WHEN urgente_expedientes = ARRAY[true, false, false]
             AND fase_desde_expedientes[3] = 'expediente:ct:sin-urgencia'
       THEN 'ok' ELSE 'FALLO: urgencias de v4 ' || urgente_expedientes::text END
  FROM vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(
      NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL);
RESET ROLE;
-- Un total que no cuadra con el contenido se rechaza sin detalles.
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(p_contenido bytea)
RETURNS TABLE(orden integer, expediente_ref text, version numeric)
LANGUAGE sql IMMUTABLE STRICT AS $doble$ SELECT 1, 'expediente:ct:sin-urgencia'::text, 1::numeric $doble$;
SET LOCAL ROLE vec_contratacion_temporal_consultor_rrhh;
DO $$ BEGIN
    PERFORM * FROM vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(
        NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL);
    RAISE EXCEPTION 'FALLO: cuadro desalineado admitido';
EXCEPTION WHEN insufficient_privilege THEN NULL; END $$;
ROLLBACK;
SELECT 'ok';
SELECT 'CT125 OK';
