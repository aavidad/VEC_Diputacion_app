\set ON_ERROR_STOP on
-- Fixture del ensayo de CT121 con la cadena real tras una expiración, sobre
-- la estructura real restaurada (volcado sintético) con CT110, CT111, CT119 y
-- CT121 instaladas. Base desechable: se confirma. SOLO PRUEBA.
--
-- 1. Dobles de las seis fachadas AD3 que consume la cadena (aviso, respuesta,
--    justificante, resolución, continuación y propuesta): devuelven un consumo
--    nuevo ligado al recurso y a la huella que declara la decisión. Prueban la
--    transacción CT, no la criptografía V3.
-- 2. Rebobinado del expediente B al estado real que dejó su primer aviso: el
--    volcado conserva, detrás de ese aviso, una renuncia, su continuación
--    (CT60), el circuito del sucesor y la propuesta (v7). Se retiran esas
--    filas posteriores (nada más) para rehacer el mismo tramo por el camino de
--    la expiración con las funciones reales: el plazo y la expiración de
--    CT111, la continuación de CT119 y los cinco escritores del sucesor. Lo
--    que queda (versiones 1-6, selección confirmada y primer aviso) lo
--    escribieron en su día las funciones reales.
\set exp_b 'expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7'
\set llam_1 'llamamiento:nccfkjnioljdeikkpipkcgcpilbogjnociankdfbapmnaekanagbiioaahphbmgj'
\set llam_2 'llamamiento:eipmgkkfjncbalgebihinpeeajhllfmkoikjhiodlbcdlhaohmgpojefdfilcmoe'
-- psql no sustituye variables dentro de bloques $$: los DO las leen de aquí.
SELECT set_config('prueba_ct121c.exp_b', :'exp_b', false), set_config('prueba_ct121c.llam_1', :'llam_1', false);

CREATE ROLE vec_ct121c_runtime LOGIN INHERIT IN ROLE vec_contratacion_temporal_ejecutor;
GRANT CONNECT ON DATABASE postgres TO vec_ct121c_runtime;

DO $dobles$
DECLARE v_nombre text;
BEGIN
    FOREACH v_nombre IN ARRAY ARRAY[
        'registrar_y_consumir_comunicacion_llamamiento_v3_atestada',
        'registrar_y_consumir_respuesta_recibida_rrhh_v3_atestada',
        'registrar_y_consumir_justificante_respuesta_ct_v3_atestada',
        'registrar_y_consumir_resolucion_manual_ct_v3_atestada',
        'registrar_y_consumir_continuacion_ct_v3_atestada',
        'registrar_y_consumir_propuesta_formalizacion_ct_v3_atestada'
    ] LOOP
        IF to_regprocedure('vec_autorizacion_atestada_v3.'||v_nombre||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
            RAISE EXCEPTION 'fachada AD3 ausente: %', v_nombre;
        END IF;
        EXECUTE format($f$CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.%I(
            p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
            p_persona_version numeric,p_perfil_version numeric,
            p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
        ) RETURNS TABLE (
            decision_ref text,efecto_ref text,huella_efecto_sha256 text,
            consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean
        ) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog
        AS $c$
        DECLARE d jsonb := convert_from(p_decision,'UTF8')::jsonb;
            h text := encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex');
        BEGIN
            -- Doble de prueba: eco del efecto de la decisión y consumo único.
            RETURN QUERY SELECT 'decision:'||substr(h,1,32), d->>'recurso_ref',
                d->>'contexto_recurso_huella_sha256', h, 'aud_v3_'||substr(h,33,32), clock_timestamp(), true;
        END
        $c$$f$, v_nombre);
    END LOOP;
END
$dobles$;

-- Rebobinado de B: todo lo posterior a su primer aviso.
DO $antes$
BEGIN
    IF (SELECT version FROM vec_contratacion_temporal.expediente_integral_actual WHERE expediente_ref=current_setting('prueba_ct121c.exp_b'))<>7
       OR (SELECT count(*) FROM vec_contratacion_temporal.comunicacion_llamamiento_local WHERE expediente_ref=current_setting('prueba_ct121c.exp_b'))<>2
       OR (SELECT count(*) FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh WHERE expediente_ref=current_setting('prueba_ct121c.exp_b'))<>2
       OR (SELECT count(*) FROM vec_contratacion_temporal.propuesta_formalizacion WHERE expediente_ref=current_setting('prueba_ct121c.exp_b'))<>1 THEN
        RAISE EXCEPTION 'el volcado no tiene la forma esperada del expediente B';
    END IF;
END
$antes$;
BEGIN;
SET LOCAL session_replication_role = replica;
DELETE FROM vec_contratacion_temporal.propuesta_formalizacion WHERE expediente_ref=:'exp_b';
DELETE FROM vec_contratacion_temporal.outbox_expediente_integral WHERE expediente_ref=:'exp_b' AND version_expediente=7;
DELETE FROM vec_contratacion_temporal.actuacion_expediente_integral WHERE expediente_ref=:'exp_b' AND version_expediente=7;
-- CT110 rellenó la entrada en fase de cada versión publicada.
DELETE FROM vec_contratacion_temporal.fase_entrada_publicacion_rrhh WHERE expediente_ref=:'exp_b' AND version=7;
DELETE FROM vec_contratacion_temporal.publicacion_version_rrhh WHERE expediente_ref=:'exp_b' AND version=7;
UPDATE vec_contratacion_temporal.expediente_integral_actual a
   SET version=6, actualizada_en=v.registrada_en, operacion_ref=v.operacion_ref
  FROM vec_contratacion_temporal.expediente_version_integral v
 WHERE a.expediente_ref=:'exp_b' AND v.expediente_ref=a.expediente_ref AND v.version=6;
DELETE FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_b' AND version=7;
DELETE FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh WHERE expediente_ref=:'exp_b';
DELETE FROM vec_contratacion_temporal.historia_respuesta_recibida_rrhh
 WHERE justificante_ref IN (SELECT justificante_ref FROM vec_contratacion_temporal.respuesta_recibida_rrhh WHERE expediente_ref=:'exp_b');
DELETE FROM vec_contratacion_temporal.outbox_respuesta_recibida_rrhh
 WHERE justificante_ref IN (SELECT justificante_ref FROM vec_contratacion_temporal.respuesta_recibida_rrhh WHERE expediente_ref=:'exp_b');
DELETE FROM vec_contratacion_temporal.respuesta_recibida_rrhh WHERE expediente_ref=:'exp_b';
DELETE FROM vec_contratacion_temporal.historia_comunicacion_llamamiento_local
 WHERE comunicacion_ref IN (SELECT comunicacion_ref FROM vec_contratacion_temporal.comunicacion_llamamiento_local
                             WHERE expediente_ref=:'exp_b' AND llamamiento_ref=:'llam_2');
DELETE FROM vec_contratacion_temporal.outbox_comunicacion_llamamiento_local
 WHERE comunicacion_ref IN (SELECT comunicacion_ref FROM vec_contratacion_temporal.comunicacion_llamamiento_local
                             WHERE expediente_ref=:'exp_b' AND llamamiento_ref=:'llam_2');
DELETE FROM vec_contratacion_temporal.comunicacion_llamamiento_local WHERE expediente_ref=:'exp_b' AND llamamiento_ref=:'llam_2';
COMMIT;
DO $despues$
BEGIN
    IF (SELECT version FROM vec_contratacion_temporal.expediente_integral_actual WHERE expediente_ref=current_setting('prueba_ct121c.exp_b'))<>6
       OR (SELECT count(*) FROM vec_contratacion_temporal.comunicacion_llamamiento_local
            WHERE expediente_ref=current_setting('prueba_ct121c.exp_b') AND llamamiento_ref=current_setting('prueba_ct121c.llam_1') AND estado='registrada_localmente' AND version_resultante=2)<>1
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.respuesta_recibida_rrhh WHERE expediente_ref=current_setting('prueba_ct121c.exp_b'))
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh WHERE expediente_ref=current_setting('prueba_ct121c.exp_b'))
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.evento_plazo_llamamiento_rrhh WHERE expediente_ref=current_setting('prueba_ct121c.exp_b')) THEN
        RAISE EXCEPTION 'rebobinado incompleto';
    END IF;
END
$despues$;
SELECT 'fixture CT121 OK' AS resultado;
