\set ON_ERROR_STOP on
-- Solo para un clon desechable sin logins externos ni historia de uso.
-- Nunca ejecutar sobre una base que haya atendido el portal del candidato.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000059', 0));
DO $guardia$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
    OR to_regrole('vec_bolsa_llamamientos_portal_externo') IS NULL
    OR EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid='vec_bolsa_llamamientos_portal_externo'::regrole)
    OR position('vec_bolsa_llamamientos_portal_externo' in pg_get_functiondef(
        'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))>0
 THEN RAISE EXCEPTION 'Bolsa 000059: retirada denegada' USING ERRCODE='55000'; END IF;
END $guardia$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
REVOKE ALL ON FUNCTION
 vec_bolsa_llamamientos.consultar_mi_bolsa_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.consultar_mi_bolsa_portal_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.consultar_historial_mi_bolsa_v1(text,timestamptz,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.manifestar_disposicion_oferta_v1(text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.listar_ofertas_candidato_v1(text,timestamptz),
 vec_bolsa_llamamientos.solicitar_portal_candidato_v1(text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.responder_llamamiento_portal_v1(text,text,text,text,text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.preparar_respuesta_portal_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.leer_portal_candidato_v1(text,timestamptz,text[]),
 vec_bolsa_llamamientos.confirmar_contacto_propio_v1(text,text,bigint,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.leer_contacto_candidato_v1(text,timestamptz)
 FROM vec_bolsa_llamamientos_portal_externo;
REVOKE USAGE ON SCHEMA vec_bolsa_llamamientos FROM vec_bolsa_llamamientos_portal_externo;
RESET ROLE;
DO $conexion$ BEGIN
 EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM vec_bolsa_llamamientos_portal_externo',current_database());
END $conexion$;
DROP ROLE vec_bolsa_llamamientos_portal_externo;
COMMIT;
