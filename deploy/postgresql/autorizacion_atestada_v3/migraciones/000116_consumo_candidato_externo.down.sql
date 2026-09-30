\set ON_ERROR_STOP on
-- Solo reversible sin historia externa. No ejecutar DOWN sobre recibos reales.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000116',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3_externa,
 vec_autorizacion_atestada_v3.consumo_decision_v3_externa,
 vec_autorizacion_atestada_v3.auditoria_consumo_v3_externa IN ACCESS EXCLUSIVE MODE;
DO $historia$
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3_externa)
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.consumo_decision_v3_externa)
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3_externa)
    OR NOT EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria_externa
                    WHERE control_id AND secuencia=0 AND cabeza_sha256=repeat('0',64))
 THEN RAISE EXCEPTION 'AD3-116: historia externa impide DOWN' USING ERRCODE='55000'; END IF;
END $historia$;
DO $nucleo$
DECLARE f regprocedure := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; anterior text;
 despacho text := $d$BEGIN
    IF session_user = 'vec_externo_bolsa_desarrollo' THEN
        IF p_perfil_mutacion NOT IN ('consulta_participaciones_propias_bolsa','portal_candidato_bolsa')
           OR p_perfil_mutacion IS NULL THEN
            RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='perfil externo VEC-AD-3 rechazado';
        END IF;
        RETURN QUERY SELECT q.* FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(
            p_perfil_mutacion,p_capacidad_canonica,p_decision_canonica,p_motivo_canonico,
            p_contexto_actor_canonico,p_persona_version,p_perfil_version,p_payload_vec_ad_3,
            p_sobre_cose_sign1,p_evidencia_verificacion,p_raiz_publica_spki) q;
        RETURN;
    END IF;
$d$;
BEGIN
 SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT original;
 IF pg_catalog.strpos(original,despacho)=0
    OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,despacho,''))<>pg_catalog.length(despacho)
 THEN RAISE EXCEPTION 'AD3-116: núcleo posterior incompatible' USING ERRCODE='55000'; END IF;
 anterior:=pg_catalog.replace(original,despacho,E'BEGIN\n');
 EXECUTE anterior;
END $nucleo$;
DROP FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3_externa;
DROP TABLE vec_autorizacion_atestada_v3.consumo_decision_v3_externa;
DROP TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3_externa;
DROP TABLE vec_autorizacion_atestada_v3.control_cadena_auditoria_externa;
COMMIT;
