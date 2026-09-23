\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000019', 0));

-- B8, manual de Mérida §6-7 y Petición RRHH p.2: pausar, reactivar y excluir.
-- No son situaciones nuevas. Cada operación produce exactamente una de las de
-- B2, de modo que el histórico sigue siendo uno solo; esta migración añade
-- quién la registra, con qué justificante y quién la valida. El documento vive
-- en su custodia: aquí solo quedan su referencia y su huella.
DO $precondicion$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_bolsa_llamamientos.registrar_situacion_participacion_v1(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.situacion_participacion') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.operacion_situacion_participacion') IS NOT NULL THEN
  RAISE EXCEPTION 'estado incompatible para las operaciones B8' USING ERRCODE='55000';
 END IF;
END $precondicion$;

CREATE TABLE vec_bolsa_llamamientos.operacion_situacion_participacion (
    participacion_ref text NOT NULL,
    desde timestamptz(6) NOT NULL,
    operacion text NOT NULL CHECK (operacion IN ('pausar','reactivar','excluir')),
    justificante_tipo text NOT NULL CHECK (justificante_tipo IN ('solicitud_candidato','informe_medico','resolucion','correo','acta_bolsa','otro')),
    justificante_ref text NOT NULL CHECK (octet_length(justificante_ref) BETWEEN 1 AND 256 AND justificante_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]*$'),
    justificante_sha256 text NOT NULL CHECK (justificante_sha256 ~ '^[a-f0-9]{64}$'),
    actor text NOT NULL CHECK (octet_length(actor) BETWEEN 1 AND 256 AND actor = btrim(actor)),
    validador text NOT NULL CHECK (octet_length(validador) BETWEEN 1 AND 256 AND validador = btrim(validador)),
    validada_en timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    clave_idempotencia text NOT NULL CHECK (octet_length(clave_idempotencia) BETWEEN 1 AND 256 AND clave_idempotencia = btrim(clave_idempotencia)),
    PRIMARY KEY (participacion_ref, desde),
    UNIQUE (participacion_ref, clave_idempotencia),
    FOREIGN KEY (participacion_ref, desde) REFERENCES vec_bolsa_llamamientos.situacion_participacion(participacion_ref, desde),
    CHECK (validada_en <= registrada_en),
    -- Regla provisional de dirección mientras RRHH no responda la duda 6: la
    -- exclusión, que B2 no permite deshacer, exige una segunda persona.
    CHECK (operacion <> 'excluir' OR validador <> actor)
);
ALTER TABLE vec_bolsa_llamamientos.operacion_situacion_participacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.operacion_situacion_participacion FORCE ROW LEVEL SECURITY;
CREATE POLICY operacion_situacion_participacion_solo_propietario ON vec_bolsa_llamamientos.operacion_situacion_participacion TO vec_bolsa_llamamientos_propietario USING (current_user = 'vec_bolsa_llamamientos_propietario') WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.operacion_situacion_participacion FROM PUBLIC;
CREATE TRIGGER operacion_situacion_participacion_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.operacion_situacion_participacion FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

CREATE FUNCTION vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v1(
 p_bolsa_ref text, p_participacion_ref text, p_operacion text, p_desde timestamptz, p_fecha_disponible timestamptz,
 p_motivo text, p_actor text, p_clave_idempotencia text, p_recibo_ref text, p_registrada_en timestamptz,
 p_justificante_tipo text, p_justificante_ref text, p_justificante_sha256 text, p_validador text, p_validada_en timestamptz,
 p_capacidad bytea, p_decision bytea, p_motivo_autorizacion bytea, p_contexto bytea, p_persona_version numeric,
 p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, recibo_ref text, situacion text, desde timestamptz, fecha_disponible timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE v_situacion text; v_cambio record; v_previa record;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='operacion no autorizada'; END IF;
 v_situacion := CASE p_operacion WHEN 'pausar' THEN 'no_disponible' WHEN 'reactivar' THEN 'disponible' WHEN 'excluir' THEN 'excluido' END;
 IF v_situacion IS NULL
    OR p_justificante_tipo NOT IN ('solicitud_candidato','informe_medico','resolucion','correo','acta_bolsa','otro')
    OR p_justificante_ref IS NULL OR p_justificante_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]*$' OR octet_length(p_justificante_ref) NOT BETWEEN 1 AND 256
    OR p_justificante_sha256 IS NULL OR p_justificante_sha256 !~ '^[a-f0-9]{64}$'
    OR p_validador IS NULL OR p_validador <> btrim(p_validador) OR octet_length(p_validador) NOT BETWEEN 1 AND 256
    OR p_validada_en IS NULL OR p_registrada_en IS NULL OR p_validada_en > p_registrada_en
    OR (p_operacion = 'excluir' AND p_validador = p_actor)
    OR p_fecha_disponible IS NOT NULL THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='operacion invalida';
 END IF;

 -- B2 consume primero la misma autorización positiva también en replay.
 -- Su comprobación de clave impide adoptar un cambio B2 ajeno a B8.
 SELECT o.*, s.situacion AS situacion_registrada, s.recibo_ref AS recibo_registrado, s.fecha_disponible AS fecha_registrada
   INTO v_previa
   FROM vec_bolsa_llamamientos.operacion_situacion_participacion o
   JOIN vec_bolsa_llamamientos.situacion_participacion s USING (participacion_ref, desde)
  WHERE o.participacion_ref = p_participacion_ref AND o.clave_idempotencia = p_clave_idempotencia;
 SELECT * INTO STRICT v_cambio FROM vec_bolsa_llamamientos.registrar_situacion_participacion_v1(
   p_bolsa_ref, p_participacion_ref, v_situacion, p_desde, NULL, p_motivo, p_actor, p_clave_idempotencia,
   p_recibo_ref, p_registrada_en, p_capacidad, p_decision, p_motivo_autorizacion, p_contexto, p_persona_version,
   p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 IF v_cambio.reutilizada THEN
  IF v_previa.participacion_ref IS NULL OR v_previa.operacion <> p_operacion OR v_previa.justificante_tipo <> p_justificante_tipo
     OR v_previa.justificante_ref <> p_justificante_ref OR v_previa.justificante_sha256 <> p_justificante_sha256
     OR v_previa.validador <> p_validador OR v_previa.actor <> p_actor THEN
   RAISE EXCEPTION USING ERRCODE='VBS01', MESSAGE='clave idempotente reutilizada con otra operacion';
  END IF;
  RETURN QUERY SELECT true, v_previa.recibo_registrado, v_previa.situacion_registrada, v_previa.desde, v_previa.fecha_registrada;
  RETURN;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(
   participacion_ref, desde, operacion, justificante_tipo, justificante_ref, justificante_sha256,
   actor, validador, validada_en, registrada_en, clave_idempotencia)
 VALUES (p_participacion_ref, v_cambio.desde, p_operacion, p_justificante_tipo, p_justificante_ref,
   p_justificante_sha256, p_actor, p_validador, p_validada_en, p_registrada_en, p_clave_idempotencia);
 RETURN QUERY SELECT false, v_cambio.recibo_ref, v_cambio.situacion, v_cambio.desde, v_cambio.fecha_disponible;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.listar_operaciones_situacion_participacion_v1(
 p_participacion_ref text,p_actor text,p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(desde timestamptz, operacion text, situacion text, justificante_tipo text, justificante_ref text, justificante_sha256 text, actor text, validador text, validada_en timestamptz, motivo text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE v_consumo record; v_decision jsonb;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_participacion_ref IS NULL OR p_actor IS NULL THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de operaciones no autorizada';
 END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(
  p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN v_decision:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de operaciones no autorizada'; END;
 IF v_consumo.efecto_ref IS DISTINCT FROM p_participacion_ref OR v_consumo.consumo_nuevo IS NOT TRUE
    OR v_decision->>'principal_id' IS DISTINCT FROM p_actor
    OR v_decision->>'accion' IS DISTINCT FROM 'bolsa.situacion_participacion.cambiar'
    OR v_decision->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa'
    OR v_decision->>'finalidad' IS DISTINCT FROM 'gestion_situacion_participacion'
    OR v_decision->>'recurso_ref' IS DISTINCT FROM p_participacion_ref
    OR v_decision->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR v_decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_decision->>'contexto_recurso_huella_sha256' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de operaciones no autorizada';
 END IF;
 RETURN QUERY SELECT o.desde, o.operacion, s.situacion, o.justificante_tipo, o.justificante_ref, o.justificante_sha256, o.actor, o.validador, o.validada_en, s.motivo
   FROM vec_bolsa_llamamientos.operacion_situacion_participacion o
   JOIN vec_bolsa_llamamientos.situacion_participacion s USING (participacion_ref, desde)
  WHERE o.participacion_ref = p_participacion_ref
  ORDER BY o.desde DESC;
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v1(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_operaciones_situacion_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v1(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_operaciones_situacion_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
