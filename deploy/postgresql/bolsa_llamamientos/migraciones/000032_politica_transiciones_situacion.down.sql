\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000032:down', 0));

-- Revierte exclusivamente 000032 y devuelve a registrar_situacion_participacion_v1
-- el cuerpo exacto de 000012. No ejecutar sobre bases con historia: se
-- rechaza si algún cambio ya aplicó la política o si se publicó otra versión
-- además de la inicial.
DO $precondicion$
DECLARE v_fuente text;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.politica_transiciones_situacion') IS NULL
    -- 000037 usa la readmisión: se retira antes.
    OR to_regclass('vec_bolsa_llamamientos.reversion_sancion_participacion') IS NOT NULL THEN
  RAISE EXCEPTION 'estado incompatible para revertir la politica de transiciones B2' USING ERRCODE='55000';
 END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion WHERE politica_transiciones_version IS NOT NULL)
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.politica_transiciones_situacion) <> 1 THEN
  RAISE EXCEPTION 'la politica de transiciones B2 tiene historia' USING ERRCODE='55000';
 END IF;
 SELECT p.prosrc INTO v_fuente FROM pg_catalog.pg_proc p
  WHERE p.oid = to_regprocedure('vec_bolsa_llamamientos.registrar_situacion_participacion_v1(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF v_fuente IS NULL OR md5(v_fuente) <> '72ef0cc858f11bbdd9065ffb88caf2c3' THEN
  RAISE EXCEPTION 'registrar_situacion_participacion_v1 no es la de 000032' USING ERRCODE='55000';
 END IF;
END $precondicion$;

DROP FUNCTION vec_bolsa_llamamientos.readmitir_participacion_por_recurso_v1(text,text,text,text,text,text,text,timestamptz);
DROP FUNCTION vec_bolsa_llamamientos.publicar_politica_transiciones_situacion_v1(text,text,text[]);
DROP FUNCTION vec_bolsa_llamamientos.consultar_politica_transiciones_situacion_v1();
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.registrar_situacion_participacion_v1(p_bolsa_ref text,p_participacion_ref text, p_situacion text, p_desde timestamptz, p_fecha_disponible timestamptz, p_motivo text, p_actor text, p_clave_idempotencia text, p_recibo_ref text, p_registrada_en timestamptz,p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(reutilizada boolean, recibo_ref text, situacion text, desde timestamptz, fecha_disponible timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE anterior record; consumo record; decision jsonb;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_bolsa_ref IS NULL OR p_participacion_ref IS NULL OR p_situacion NOT IN ('disponible','no_disponible','trabajando','pendiente_incorporacion','renuncia','excluido','disponible_desde') OR p_desde IS NULL OR p_motivo IS NULL OR p_motivo<>btrim(p_motivo) OR octet_length(p_motivo) NOT BETWEEN 1 AND 1000 OR p_actor IS NULL OR p_clave_idempotencia IS NULL OR p_clave_idempotencia<>btrim(p_clave_idempotencia) OR octet_length(p_clave_idempotencia) NOT BETWEEN 1 AND 256 OR p_recibo_ref IS NULL OR p_recibo_ref<>btrim(p_recibo_ref) OR octet_length(p_recibo_ref) NOT BETWEEN 1 AND 256 OR p_registrada_en IS NULL OR (p_situacion='disponible_desde' AND p_fecha_disponible IS NULL) OR (p_situacion<>'disponible_desde' AND p_fecha_disponible IS NOT NULL) THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='situacion invalida'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea) WHERE e.participacion_ref=p_participacion_ref AND c.bolsa_ref=p_bolsa_ref) THEN RAISE EXCEPTION USING ERRCODE='23503',MESSAGE='participacion ajena a la bolsa'; END IF;
 SELECT * INTO anterior FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p_participacion_ref ORDER BY desde DESC LIMIT 1 FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='23503', MESSAGE='participacion inexistente'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN decision:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='situacion no autorizada'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_participacion_ref OR consumo.consumo_nuevo IS NOT TRUE OR decision->>'principal_id' IS DISTINCT FROM p_actor OR decision->>'accion' IS DISTINCT FROM 'bolsa.situacion_participacion.cambiar' OR decision->>'modulo_id' IS DISTINCT FROM 'bolsa' OR decision->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa' OR decision->>'finalidad' IS DISTINCT FROM 'gestion_situacion_participacion' OR decision->>'recurso_ref' IS DISTINCT FROM p_participacion_ref OR decision->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb OR consumo.huella_efecto_sha256 IS DISTINCT FROM decision->>'contexto_recurso_huella_sha256' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='situacion no autorizada'; END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p_participacion_ref AND s.clave_idempotencia=p_clave_idempotencia AND (s.situacion<>p_situacion OR s.motivo<>p_motivo OR s.fecha_disponible IS DISTINCT FROM p_fecha_disponible)) THEN RAISE EXCEPTION USING ERRCODE='VBS01', MESSAGE='clave idempotente reutilizada con otro comando'; END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p_participacion_ref AND s.clave_idempotencia=p_clave_idempotencia) THEN RETURN QUERY SELECT true, s.recibo_ref, s.situacion, s.desde, s.fecha_disponible FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p_participacion_ref AND s.clave_idempotencia=p_clave_idempotencia; RETURN; END IF;
 IF p_desde < anterior.desde THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='desde anterior a la situacion vigente'; END IF;
 IF p_situacion='disponible_desde' AND p_fecha_disponible<=p_registrada_en THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='fecha disponible no futura'; END IF;
 IF NOT ((anterior.situacion='disponible' AND p_situacion IN ('no_disponible','pendiente_incorporacion','renuncia','excluido')) OR (anterior.situacion='no_disponible' AND p_situacion IN ('disponible','excluido')) OR (anterior.situacion='pendiente_incorporacion' AND p_situacion IN ('trabajando','disponible','renuncia','excluido')) OR (anterior.situacion='trabajando' AND p_situacion IN ('disponible','disponible_desde','excluido')) OR (anterior.situacion='disponible_desde' AND p_situacion IN ('disponible','excluido')) OR (anterior.situacion='renuncia' AND p_situacion IN ('disponible','excluido'))) THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='transicion de situacion invalida'; END IF;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref) VALUES(p_participacion_ref,p_situacion,p_desde,NULL,p_fecha_disponible,p_motivo,p_actor,p_registrada_en,p_clave_idempotencia,p_recibo_ref);
 RETURN QUERY SELECT false,p_recibo_ref,p_situacion,p_desde,p_fecha_disponible;
END $f$;

ALTER TABLE vec_bolsa_llamamientos.situacion_participacion DROP COLUMN politica_transiciones_version;
DROP TABLE vec_bolsa_llamamientos.politica_transiciones_situacion;
DROP FUNCTION vec_bolsa_llamamientos.transiciones_situacion_canonicas(text[]);
COMMIT;
