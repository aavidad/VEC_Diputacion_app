\set ON_ERROR_STOP on
-- B55: lectura de la ficha de reincorporación con acción V3 propia AD3-101.
-- B46 v1 conserva definición e historia, pero el ejecutor deja de poder usarla.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000055',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_reincorporacion_titular_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.reincorporacion_titular_lectura_v3') IS NOT NULL
 THEN RAISE EXCEPTION 'B55: preimagen incompatible (B46 y AD3-101 requeridas)' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE TABLE vec_bolsa_llamamientos.reincorporacion_titular_lectura_v3(
 decision_ref text PRIMARY KEY,
 auditoria_ref text NOT NULL,
 participacion_ref text NOT NULL CHECK(octet_length(participacion_ref) BETWEEN 1 AND 512),
 actor_ref text NOT NULL CHECK(actor_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 filas_devueltas integer NOT NULL CHECK(filas_devueltas BETWEEN 0 AND 100),
 consultada_en timestamptz(6) NOT NULL
);
ALTER TABLE vec_bolsa_llamamientos.reincorporacion_titular_lectura_v3 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.reincorporacion_titular_lectura_v3 FORCE ROW LEVEL SECURITY;
CREATE POLICY reincorporacion_titular_lectura_v3_solo_propietario
 ON vec_bolsa_llamamientos.reincorporacion_titular_lectura_v3
 TO vec_bolsa_llamamientos_propietario
 USING (current_user = 'vec_bolsa_llamamientos_propietario')
 WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
CREATE TRIGGER reincorporacion_titular_lectura_inmutable
 BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.reincorporacion_titular_lectura_v3
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE FUNCTION vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v2(
 p_participacion_ref text, p_actor text, p_capacidad bytea, p_decision bytea, p_motivo_autorizacion bytea,
 p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea,
 p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(evento_ref text, expediente_ref text, relacion_ref text, fecha_efectiva date,
 recibo_ct_ref text, cese_evento_ref text, estado text, disponible_desde date,
 regla_version bigint, regla_huella_sha256 text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' AS $f$
DECLARE v_consumo record; v_decision jsonb; v_capacidad jsonb; v_filas integer;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR session_user = current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR current_setting('transaction_isolation') <> 'serializable'
    OR current_setting('transaction_read_only') <> 'off'
    OR current_setting('TimeZone') <> 'UTC'
    OR p_participacion_ref IS NULL OR p_participacion_ref = '' OR octet_length(p_participacion_ref) > 512
    OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$' THEN
  RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='consulta de reincorporaciones no autorizada';
 END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.consumir_consulta_reincorporacion_titular_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN v_decision := convert_from(p_decision, 'UTF8')::jsonb;
       v_capacidad := convert_from(p_capacidad, 'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='consulta de reincorporaciones no autorizada'; END;
 IF v_consumo.efecto_ref IS DISTINCT FROM p_participacion_ref OR v_consumo.consumo_nuevo IS NOT TRUE
    OR v_capacidad->>'operacion' IS DISTINCT FROM 'bolsa.reincorporacion_titular.consultar'
    OR v_capacidad->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.reincorporacion_titular.consultar.v1'
    OR v_capacidad->>'efecto_ref' IS DISTINCT FROM p_participacion_ref
    OR v_decision->>'principal_id' IS DISTINCT FROM p_actor
    OR v_decision->>'accion' IS DISTINCT FROM 'bolsa.reincorporacion_titular.consultar'
    OR v_decision->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa'
    OR v_decision->>'finalidad' IS DISTINCT FROM 'consulta_reincorporacion_titular'
    OR v_decision->>'recurso_ref' IS DISTINCT FROM p_participacion_ref
    OR v_decision->'campos_permitidos' IS DISTINCT FROM '["reincorporaciones_titular"]'::jsonb
    OR v_decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_decision->>'contexto_recurso_huella_sha256' THEN
  RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='consulta de reincorporaciones no autorizada';
 END IF;
 RETURN QUERY
 SELECT r.evento_ref, r.expediente_ref, r.relacion_ref, r.fecha_efectiva, r.recibo_ct_ref, r.cese_evento_ref,
  'cese_aplicado'::text, c.disponible_desde, c.politica_version, p.catalogo_sha256
 FROM vec_bolsa_llamamientos.restriccion_cese_bolsa c
 JOIN vec_bolsa_llamamientos.llamamiento_integracion_desarrollo l ON l.llamamiento_ref = c.llamamiento_ref
 JOIN vec_bolsa_llamamientos.integracion_desarrollo i ON i.operacion_ref = l.operacion_ref AND i.tipo = 'propuesta'
 JOIN vec_bolsa_llamamientos.reincorporacion_titular_ct r ON r.cese_evento_ref = c.origen_ref
  AND r.relacion_ref = c.relacion_ref AND r.cese_recibo_ref = c.recibo_ct_ref AND r.fecha_efectiva = c.fecha_efecto
 LEFT JOIN vec_bolsa_llamamientos.politica_cese_bolsa p ON p.version = c.politica_version
 WHERE convert_from(i.registro_canonico, 'UTF8')::jsonb #>> '{propuesta,participacion_seleccionada_ref}' = p_participacion_ref
 ORDER BY r.fecha_efectiva DESC, r.evento_ref DESC LIMIT 100;
 GET DIAGNOSTICS v_filas = ROW_COUNT;
 INSERT INTO vec_bolsa_llamamientos.reincorporacion_titular_lectura_v3(
  decision_ref,auditoria_ref,participacion_ref,actor_ref,filas_devueltas,consultada_en)
 VALUES(v_consumo.decision_ref,v_consumo.auditoria_ref,p_participacion_ref,p_actor,v_filas,clock_timestamp());
END $f$;

REVOKE ALL ON TABLE vec_bolsa_llamamientos.reincorporacion_titular_lectura_v3 FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
