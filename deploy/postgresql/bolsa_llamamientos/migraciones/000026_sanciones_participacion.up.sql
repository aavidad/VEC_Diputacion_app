\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000026', 0));

-- Petición RRHH p. 2 y duda 62: histórico de sanciones de una participación.
-- Una sanción conserva la consecuencia resuelta (clave, etiqueta y referencia
-- exacta del catálogo de reglas con su huella), la causa, la fecha de
-- notificación, la resolución (referencia y huella; el documento sigue en su
-- custodia) y el vencimiento del recurso de reposición. Si la consecuencia
-- cambia la situación, se aplica con la operación B8 existente en la misma
-- transacción y bajo el mismo consumo de la autorización: esta migración no
-- añade transiciones de situación. Todo es de solo adición.
DO $precondicion$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v1(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.operacion_situacion_participacion') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.sancion_participacion') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.recurso_sancion_participacion') IS NOT NULL THEN
  RAISE EXCEPTION 'estado incompatible para las sanciones de Bolsa' USING ERRCODE='55000';
 END IF;
END $precondicion$;

CREATE TABLE vec_bolsa_llamamientos.sancion_participacion (
    sancion_ref text PRIMARY KEY CHECK (sancion_ref ~ '^sancion:[a-f0-9]{64}$'),
    participacion_ref text NOT NULL CHECK (octet_length(participacion_ref) BETWEEN 1 AND 512),
    bolsa_ref text NOT NULL CHECK (octet_length(bolsa_ref) BETWEEN 1 AND 512),
    consecuencia text NOT NULL CHECK (consecuencia ~ '^[a-z0-9][a-z0-9._-]{0,127}$'),
    consecuencia_etiqueta text NOT NULL CHECK (octet_length(consecuencia_etiqueta) BETWEEN 1 AND 600 AND consecuencia_etiqueta = btrim(consecuencia_etiqueta)),
    efecto text NOT NULL CHECK (efecto IN ('ninguna','pausar','excluir')),
    causa text NOT NULL CHECK (octet_length(causa) BETWEEN 1 AND 1000 AND causa = btrim(causa)),
    fecha_notificacion date NOT NULL,
    resolucion_ref text NOT NULL CHECK (octet_length(resolucion_ref) BETWEEN 1 AND 256 AND resolucion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]*$'),
    resolucion_sha256 text NOT NULL CHECK (resolucion_sha256 ~ '^[a-f0-9]{64}$'),
    resuelta_por text NOT NULL CHECK (octet_length(resuelta_por) BETWEEN 1 AND 256 AND resuelta_por = btrim(resuelta_por)),
    regla_ref text NOT NULL CHECK (octet_length(regla_ref) BETWEEN 1 AND 300),
    regla_huella_sha256 text NOT NULL CHECK (regla_huella_sha256 ~ '^[a-f0-9]{64}$'),
    suspension_hasta date,
    recurso_vence date NOT NULL,
    recurso_regla_ref text NOT NULL CHECK (octet_length(recurso_regla_ref) BETWEEN 1 AND 300),
    recurso_regla_huella_sha256 text NOT NULL CHECK (recurso_regla_huella_sha256 ~ '^[a-f0-9]{64}$'),
    situacion_desde timestamptz(6),
    recibo_ref text NOT NULL UNIQUE CHECK (octet_length(recibo_ref) BETWEEN 1 AND 256 AND recibo_ref = btrim(recibo_ref)),
    actor text NOT NULL CHECK (octet_length(actor) BETWEEN 1 AND 256 AND actor = btrim(actor)),
    registrada_en timestamptz(6) NOT NULL,
    clave_idempotencia text NOT NULL CHECK (octet_length(clave_idempotencia) BETWEEN 1 AND 256 AND clave_idempotencia = btrim(clave_idempotencia)),
    UNIQUE (participacion_ref, clave_idempotencia),
    FOREIGN KEY (participacion_ref, situacion_desde) REFERENCES vec_bolsa_llamamientos.operacion_situacion_participacion(participacion_ref, desde),
    CHECK ((efecto = 'ninguna') = (situacion_desde IS NULL)),
    CHECK (suspension_hasta IS NULL OR (efecto = 'pausar' AND suspension_hasta > fecha_notificacion)),
    CHECK (recurso_vence > fecha_notificacion),
    -- Como en la exclusión B8 (duda 6), una baja la resuelve otra persona.
    CHECK (efecto <> 'excluir' OR resuelta_por <> actor)
);
CREATE INDEX sancion_participacion_historial ON vec_bolsa_llamamientos.sancion_participacion(participacion_ref, registrada_en DESC);
ALTER TABLE vec_bolsa_llamamientos.sancion_participacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.sancion_participacion FORCE ROW LEVEL SECURITY;
CREATE POLICY sancion_participacion_solo_propietario ON vec_bolsa_llamamientos.sancion_participacion TO vec_bolsa_llamamientos_propietario USING (current_user = 'vec_bolsa_llamamientos_propietario') WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.sancion_participacion FROM PUBLIC;
CREATE TRIGGER sancion_participacion_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.sancion_participacion FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Estados anotados del recurso de reposición. Los valores admitidos los fija
-- el catálogo de reglas; aquí solo se exige su forma. El vigente es el último.
CREATE TABLE vec_bolsa_llamamientos.recurso_sancion_participacion (
    sancion_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.sancion_participacion(sancion_ref),
    estado text NOT NULL CHECK (estado ~ '^[a-z][a-z0-9_]{0,63}$'),
    fecha date NOT NULL,
    documento_ref text CHECK (documento_ref IS NULL OR (octet_length(documento_ref) BETWEEN 1 AND 256 AND documento_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]*$')),
    documento_sha256 text CHECK (documento_sha256 IS NULL OR documento_sha256 ~ '^[a-f0-9]{64}$'),
    actor text NOT NULL CHECK (octet_length(actor) BETWEEN 1 AND 256 AND actor = btrim(actor)),
    registrada_en timestamptz(6) NOT NULL,
    clave_idempotencia text NOT NULL CHECK (octet_length(clave_idempotencia) BETWEEN 1 AND 256 AND clave_idempotencia = btrim(clave_idempotencia)),
    PRIMARY KEY (sancion_ref, clave_idempotencia),
    CHECK ((documento_ref IS NULL) = (documento_sha256 IS NULL))
);
CREATE INDEX recurso_sancion_participacion_historial ON vec_bolsa_llamamientos.recurso_sancion_participacion(sancion_ref, registrada_en);
ALTER TABLE vec_bolsa_llamamientos.recurso_sancion_participacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.recurso_sancion_participacion FORCE ROW LEVEL SECURITY;
CREATE POLICY recurso_sancion_participacion_solo_propietario ON vec_bolsa_llamamientos.recurso_sancion_participacion TO vec_bolsa_llamamientos_propietario USING (current_user = 'vec_bolsa_llamamientos_propietario') WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.recurso_sancion_participacion FROM PUBLIC;
CREATE TRIGGER recurso_sancion_participacion_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.recurso_sancion_participacion FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Consumo de la autorización de las operaciones de situación, con las mismas
-- comprobaciones que el histórico B8. Uso interno: sin EXECUTE para nadie más.
CREATE FUNCTION vec_bolsa_llamamientos.consumir_autorizacion_sancion_v1(
 p_participacion_ref text,p_actor text,p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS void LANGUAGE plpgsql VOLATILE SET search_path = pg_catalog AS $f$
DECLARE v_consumo record; v_decision jsonb;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_participacion_ref IS NULL OR p_actor IS NULL THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='sancion no autorizada';
 END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(
  p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN v_decision:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='sancion no autorizada'; END;
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
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='sancion no autorizada';
 END IF;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.registrar_sancion_participacion_v1(
 p_bolsa_ref text, p_participacion_ref text, p_sancion_ref text, p_consecuencia text, p_consecuencia_etiqueta text,
 p_efecto text, p_causa text, p_fecha_notificacion date, p_resolucion_ref text, p_resolucion_sha256 text,
 p_resuelta_por text, p_regla_ref text, p_regla_huella text, p_suspension_hasta date, p_recurso_vence date,
 p_recurso_regla_ref text, p_recurso_regla_huella text, p_desde timestamptz, p_actor text, p_clave_idempotencia text,
 p_recibo_ref text, p_registrada_en timestamptz,
 p_capacidad bytea, p_decision bytea, p_motivo_autorizacion bytea, p_contexto bytea, p_persona_version numeric,
 p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, sancion_ref text, recibo_ref text, situacion text, desde timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE v_previa record; v_op record; v_reutilizada boolean := false; v_situacion text; v_desde timestamptz;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='sancion no autorizada'; END IF;
 IF p_bolsa_ref IS NULL OR p_participacion_ref IS NULL OR p_clave_idempotencia IS NULL OR p_actor IS NULL OR p_registrada_en IS NULL
    OR p_sancion_ref IS DISTINCT FROM 'sancion:' || encode(sha256(convert_to(p_participacion_ref || chr(31) || p_clave_idempotencia, 'UTF8')), 'hex')
    OR p_efecto IS NULL OR p_efecto NOT IN ('ninguna','pausar','excluir')
    OR p_fecha_notificacion IS NULL OR p_fecha_notificacion > (p_registrada_en AT TIME ZONE 'Europe/Madrid')::date
    OR p_recurso_vence IS NULL OR p_recurso_vence <= p_fecha_notificacion
    OR (p_suspension_hasta IS NOT NULL AND (p_efecto <> 'pausar' OR p_suspension_hasta <= p_fecha_notificacion))
    OR (p_efecto = 'excluir' AND p_resuelta_por = p_actor)
    OR (p_efecto <> 'ninguna' AND (p_desde IS NULL OR p_recibo_ref IS DISTINCT FROM 'recibo:situacion:' || substr(p_sancion_ref, 9)))
    OR (p_efecto = 'ninguna' AND (p_desde IS NOT NULL OR p_recibo_ref IS DISTINCT FROM 'recibo:sancion:' || substr(p_sancion_ref, 9))) THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='sancion invalida';
 END IF;

 SELECT s.* INTO v_previa FROM vec_bolsa_llamamientos.sancion_participacion s
  WHERE s.participacion_ref = p_participacion_ref AND s.clave_idempotencia = p_clave_idempotencia;

 IF p_efecto <> 'ninguna' THEN
  -- B8 consume la autorización, valida la transición y resuelve su replay.
  SELECT * INTO STRICT v_op FROM vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v1(
    p_bolsa_ref, p_participacion_ref, p_efecto, p_desde, NULL, p_causa, p_actor, p_clave_idempotencia,
    p_recibo_ref, p_registrada_en, 'resolucion', p_resolucion_ref, p_resolucion_sha256, p_resuelta_por, p_registrada_en,
    p_capacidad, p_decision, p_motivo_autorizacion, p_contexto, p_persona_version, p_perfil_version,
    p_payload, p_sobre, p_evidencia, p_raiz);
  v_reutilizada := v_op.reutilizada; v_situacion := v_op.situacion; v_desde := v_op.desde;
 ELSE
  IF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e
                   JOIN vec_bolsa_llamamientos.constitucion c USING (instantanea_ref, version_instantanea)
                  WHERE e.participacion_ref = p_participacion_ref AND c.bolsa_ref = p_bolsa_ref) THEN
   RAISE EXCEPTION USING ERRCODE='23503', MESSAGE='participacion ajena a la bolsa';
  END IF;
  PERFORM vec_bolsa_llamamientos.consumir_autorizacion_sancion_v1(p_participacion_ref, p_actor, p_capacidad, p_decision,
    p_motivo_autorizacion, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 END IF;

 IF v_previa.sancion_ref IS NOT NULL THEN
  IF (p_efecto <> 'ninguna' AND NOT v_reutilizada)
     OR v_previa.bolsa_ref <> p_bolsa_ref OR v_previa.consecuencia <> p_consecuencia OR v_previa.efecto <> p_efecto
     OR v_previa.causa <> p_causa OR v_previa.fecha_notificacion <> p_fecha_notificacion
     OR v_previa.resolucion_ref <> p_resolucion_ref OR v_previa.resolucion_sha256 <> p_resolucion_sha256
     OR v_previa.resuelta_por <> p_resuelta_por OR v_previa.actor <> p_actor THEN
   RAISE EXCEPTION USING ERRCODE='VBS01', MESSAGE='clave idempotente reutilizada con otra sancion';
  END IF;
  RETURN QUERY SELECT true, v_previa.sancion_ref, v_previa.recibo_ref, v_situacion, v_desde;
  RETURN;
 END IF;
 -- Una operación B8 ya registrada con esta clave que no es una sanción no se adopta.
 IF v_reutilizada THEN
  RAISE EXCEPTION USING ERRCODE='VBS01', MESSAGE='clave idempotente reutilizada con otra operacion';
 END IF;

 INSERT INTO vec_bolsa_llamamientos.sancion_participacion(
   sancion_ref, participacion_ref, bolsa_ref, consecuencia, consecuencia_etiqueta, efecto, causa, fecha_notificacion,
   resolucion_ref, resolucion_sha256, resuelta_por, regla_ref, regla_huella_sha256, suspension_hasta, recurso_vence,
   recurso_regla_ref, recurso_regla_huella_sha256, situacion_desde, recibo_ref, actor, registrada_en, clave_idempotencia)
 VALUES (p_sancion_ref, p_participacion_ref, p_bolsa_ref, p_consecuencia, p_consecuencia_etiqueta, p_efecto, p_causa,
   p_fecha_notificacion, p_resolucion_ref, p_resolucion_sha256, p_resuelta_por, p_regla_ref, p_regla_huella,
   p_suspension_hasta, p_recurso_vence, p_recurso_regla_ref, p_recurso_regla_huella, v_desde, p_recibo_ref,
   p_actor, p_registrada_en, p_clave_idempotencia);
 RETURN QUERY SELECT false, p_sancion_ref, p_recibo_ref, v_situacion, v_desde;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v1(
 p_participacion_ref text, p_sancion_ref text, p_estado text, p_fecha date, p_documento_ref text, p_documento_sha256 text,
 p_actor text, p_clave_idempotencia text, p_registrada_en timestamptz,
 p_capacidad bytea, p_decision bytea, p_motivo_autorizacion bytea, p_contexto bytea, p_persona_version numeric,
 p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, sancion_ref text, estado text, registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE v_sancion record; v_previa record;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='recurso no autorizado'; END IF;
 IF p_participacion_ref IS NULL OR p_sancion_ref IS NULL OR p_actor IS NULL OR p_clave_idempotencia IS NULL OR p_registrada_en IS NULL
    OR p_estado IS NULL OR p_estado !~ '^[a-z][a-z0-9_]{0,63}$'
    OR p_fecha IS NULL OR p_fecha > (p_registrada_en AT TIME ZONE 'Europe/Madrid')::date
    OR (p_documento_ref IS NULL) <> (p_documento_sha256 IS NULL) THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='recurso invalido';
 END IF;
 -- La autorización se consume antes de leer: una referencia conocida no
 -- permite saber si existe una sanción fuera de ámbito.
 PERFORM vec_bolsa_llamamientos.consumir_autorizacion_sancion_v1(p_participacion_ref, p_actor, p_capacidad, p_decision,
   p_motivo_autorizacion, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 SELECT s.* INTO v_sancion FROM vec_bolsa_llamamientos.sancion_participacion s
  WHERE s.sancion_ref = p_sancion_ref AND s.participacion_ref = p_participacion_ref FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='23503', MESSAGE='sancion inexistente'; END IF;
 SELECT r.* INTO v_previa FROM vec_bolsa_llamamientos.recurso_sancion_participacion r
  WHERE r.sancion_ref = p_sancion_ref AND r.clave_idempotencia = p_clave_idempotencia;
 IF FOUND THEN
  IF v_previa.estado <> p_estado OR v_previa.fecha <> p_fecha OR v_previa.actor <> p_actor
     OR v_previa.documento_ref IS DISTINCT FROM p_documento_ref OR v_previa.documento_sha256 IS DISTINCT FROM p_documento_sha256 THEN
   RAISE EXCEPTION USING ERRCODE='VBS01', MESSAGE='clave idempotente reutilizada con otro recurso';
  END IF;
  RETURN QUERY SELECT true, v_previa.sancion_ref, v_previa.estado, v_previa.registrada_en;
  RETURN;
 END IF;
 IF p_fecha < v_sancion.fecha_notificacion THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='recurso anterior a la notificacion';
 END IF;
 INSERT INTO vec_bolsa_llamamientos.recurso_sancion_participacion(sancion_ref, estado, fecha, documento_ref, documento_sha256, actor, registrada_en, clave_idempotencia)
 VALUES (p_sancion_ref, p_estado, p_fecha, p_documento_ref, p_documento_sha256, p_actor, p_registrada_en, p_clave_idempotencia);
 RETURN QUERY SELECT false, p_sancion_ref, p_estado, p_registrada_en;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.listar_sanciones_participacion_v1(
 p_participacion_ref text,p_actor text,p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(sancion_ref text, consecuencia text, consecuencia_etiqueta text, efecto text, causa text, fecha_notificacion date,
 resolucion_ref text, resolucion_sha256 text, resuelta_por text, regla_ref text, regla_huella_sha256 text,
 suspension_hasta date, recurso_vence date, recurso_regla_ref text, recurso_regla_huella_sha256 text,
 situacion_desde timestamptz, actor text, registrada_en timestamptz, recursos jsonb)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_participacion_ref IS NULL OR p_actor IS NULL THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de sanciones no autorizada';
 END IF;
 PERFORM vec_bolsa_llamamientos.consumir_autorizacion_sancion_v1(p_participacion_ref, p_actor, p_capacidad, p_decision,
   p_motivo_autorizacion, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 RETURN QUERY SELECT s.sancion_ref, s.consecuencia, s.consecuencia_etiqueta, s.efecto, s.causa, s.fecha_notificacion,
   s.resolucion_ref, s.resolucion_sha256, s.resuelta_por, s.regla_ref, s.regla_huella_sha256, s.suspension_hasta,
   s.recurso_vence, s.recurso_regla_ref, s.recurso_regla_huella_sha256, s.situacion_desde, s.actor, s.registrada_en,
   coalesce((SELECT jsonb_agg(jsonb_build_object('estado', r.estado, 'fecha', r.fecha, 'documento_ref', r.documento_ref,
       'documento_sha256', r.documento_sha256, 'actor', r.actor, 'registrada_en', r.registrada_en) ORDER BY r.registrada_en, r.clave_idempotencia)
     FROM vec_bolsa_llamamientos.recurso_sancion_participacion r WHERE r.sancion_ref = s.sancion_ref), '[]'::jsonb)
   FROM vec_bolsa_llamamientos.sancion_participacion s
  WHERE s.participacion_ref = p_participacion_ref
  ORDER BY s.registrada_en DESC, s.sancion_ref
  LIMIT 200;
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consumir_autorizacion_sancion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_sancion_participacion_v1(text,text,text,text,text,text,text,date,text,text,text,text,text,date,date,text,text,timestamptz,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v1(text,text,text,date,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_sanciones_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_sancion_participacion_v1(text,text,text,text,text,text,text,date,text,text,text,text,text,date,date,text,text,timestamptz,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v1(text,text,text,date,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_sanciones_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
