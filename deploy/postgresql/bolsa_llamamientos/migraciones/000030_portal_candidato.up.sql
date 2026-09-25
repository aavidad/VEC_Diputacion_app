\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000030', 0));

-- Portal del candidato (petición RRHH p. 1 punto 5 y p. 2; dudas 3, 17 y 18).
-- La persona, con la autorización propia AD3-84, puede:
--  * solicitar una pausa voluntaria o la reactivación. Es una solicitud: no
--    cambia la situación B2. Queda pendiente hasta que RRHH registra la
--    operación B8 con justificante 'solicitud_candidato' y la referencia de
--    la solicitud; así el histórico sigue siendo uno solo;
--  * responder a su llamamiento abierto dentro de plazo (aceptar, renunciar o
--    renunciar con causa justificada y justificante). El modo (respuesta
--    firme o propuesta que RRHH confirma), el plazo y las situaciones que
--    admiten cada solicitud llegan del catálogo de reglas: aquí solo se
--    comprueban y se conservan con la referencia de la regla.
-- Todo es de solo adición; RRHH lo ve en su bandeja de avisos.
DO $precondicion$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.operacion_situacion_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.llamamiento_emitido') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.contacto_participacion') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.listar_participaciones_candidato_v1(text)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.solicitud_portal_candidato') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.respuesta_portal_llamamiento') IS NOT NULL THEN
  RAISE EXCEPTION 'estado incompatible para el portal del candidato' USING ERRCODE='55000';
 END IF;
END $precondicion$;

CREATE TABLE vec_bolsa_llamamientos.solicitud_portal_candidato (
    solicitud_ref text PRIMARY KEY CHECK (solicitud_ref ~ '^solicitud-portal:[0-9a-f]{64}$'),
    recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:solicitud-portal:[0-9a-f]{64}$'),
    bolsa_ref text NOT NULL CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(bolsa_ref, 512)),
    participacion_ref text NOT NULL CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(participacion_ref, 512)),
    candidato_ref text NOT NULL CHECK (candidato_ref ~ '^can_[A-Za-z0-9_-]{22,128}$'),
    tipo text NOT NULL CHECK (tipo IN ('pausa','reactivacion')),
    pausa_hasta timestamptz(6),
    situacion_previa text NOT NULL,
    regla_ref text NOT NULL CHECK (octet_length(regla_ref) BETWEEN 1 AND 512 AND regla_ref = btrim(regla_ref)),
    clave_idempotencia text NOT NULL CHECK (octet_length(clave_idempotencia) BETWEEN 8 AND 256 AND clave_idempotencia = btrim(clave_idempotencia)),
    decision_ref text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL,
    UNIQUE (participacion_ref, clave_idempotencia),
    CHECK ((tipo = 'pausa' AND pausa_hasta IS NOT NULL AND pausa_hasta > registrada_en) OR (tipo = 'reactivacion' AND pausa_hasta IS NULL))
);
CREATE INDEX solicitud_portal_candidato_participacion ON vec_bolsa_llamamientos.solicitud_portal_candidato(participacion_ref, registrada_en DESC);

CREATE TABLE vec_bolsa_llamamientos.respuesta_portal_llamamiento (
    respuesta_ref text PRIMARY KEY CHECK (respuesta_ref ~ '^respuesta-portal:[0-9a-f]{64}$'),
    recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:respuesta-portal:[0-9a-f]{64}$'),
    bolsa_ref text NOT NULL,
    participacion_ref text NOT NULL,
    candidato_ref text NOT NULL CHECK (candidato_ref ~ '^can_[A-Za-z0-9_-]{22,128}$'),
    llamamiento_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.llamamiento_emitido(llamamiento_ref),
    respuesta text NOT NULL CHECK (respuesta IN ('acepta','renuncia','renuncia_justificada')),
    causa text CHECK (causa ~ '^[a-z][a-z0-9_]{0,63}$'),
    justificante_ref text CHECK (octet_length(justificante_ref) BETWEEN 1 AND 256 AND justificante_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]*$'),
    justificante_sha256 text CHECK (justificante_sha256 ~ '^[a-f0-9]{64}$'),
    modo text NOT NULL CHECK (modo IN ('firme','propuesta_rrhh')),
    contacto_en timestamptz(6) NOT NULL,
    vence_antes_de timestamptz(6) NOT NULL,
    regla_ref text NOT NULL CHECK (octet_length(regla_ref) BETWEEN 1 AND 512 AND regla_ref = btrim(regla_ref)),
    clave_idempotencia text NOT NULL CHECK (octet_length(clave_idempotencia) BETWEEN 8 AND 256 AND clave_idempotencia = btrim(clave_idempotencia)),
    decision_ref text NOT NULL UNIQUE,
    respondida_en timestamptz(6) NOT NULL,
    UNIQUE (llamamiento_ref, participacion_ref),
    UNIQUE (participacion_ref, clave_idempotencia),
    CHECK (contacto_en <= respondida_en AND respondida_en < vence_antes_de),
    CHECK ((respuesta = 'renuncia_justificada') = (causa IS NOT NULL AND justificante_ref IS NOT NULL AND justificante_sha256 IS NOT NULL)),
    CHECK (respuesta = 'renuncia_justificada' OR (causa IS NULL AND justificante_ref IS NULL AND justificante_sha256 IS NULL))
);

DO $tablas$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['solicitud_portal_candidato','respuesta_portal_llamamiento'] LOOP
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I ENABLE ROW LEVEL SECURITY', t);
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I FORCE ROW LEVEL SECURITY', t);
  EXECUTE format('CREATE POLICY %I ON vec_bolsa_llamamientos.%I TO vec_bolsa_llamamientos_propietario USING (current_user = ''vec_bolsa_llamamientos_propietario'') WITH CHECK (current_user = ''vec_bolsa_llamamientos_propietario'')', t||'_solo_propietario', t);
  EXECUTE format('REVOKE ALL ON vec_bolsa_llamamientos.%I FROM PUBLIC', t);
  EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion()', t||'_inmutable', t);
 END LOOP;
END $tablas$;

-- Participación única del candidato en la bolsa. Sin vínculo, 42501.
CREATE FUNCTION vec_bolsa_llamamientos.participacion_portal_candidato_v1(p_candidato_ref text, p_bolsa_ref text)
RETURNS text LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE v text; n integer;
BEGIN
 SELECT min(p.participacion_ref), count(DISTINCT p.participacion_ref) INTO v, n
   FROM vec_bolsa_llamamientos.listar_participaciones_candidato_v1(p_candidato_ref) p
  WHERE p.bolsa_ref = p_bolsa_ref;
 IF n <> 1 THEN RAISE EXCEPTION 'participación del portal ajena o ambigua' USING ERRCODE='42501'; END IF;
 RETURN v;
END $f$;

-- Llamamiento abierto: el último emitido que incluye la participación, sin
-- respuesta del portal, con un contacto efectivo posterior a la emisión. El
-- plazo empieza en el primer contacto efectivo; qué resultados lo son
-- (regla del catálogo) llega del servidor.
CREATE FUNCTION vec_bolsa_llamamientos.llamamiento_abierto_portal_v1(p_participacion_ref text, p_corte timestamptz, p_resultados_efectivos text[])
RETURNS TABLE(llamamiento_ref text, contacto_en timestamptz)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT l.llamamiento_ref, ct.instante
   FROM (SELECT e.llamamiento_ref, e.emitido_en
           FROM vec_bolsa_llamamientos.llamamiento_emitido e
          WHERE e.participaciones ? p_participacion_ref AND e.emitido_en <= p_corte
          ORDER BY e.emitido_en DESC, e.llamamiento_ref DESC LIMIT 1) l
   CROSS JOIN LATERAL (
     SELECT min(c.instante) AS instante
       FROM vec_bolsa_llamamientos.contacto_participacion c
      WHERE c.participacion_ref = p_participacion_ref AND c.resultado = ANY (p_resultados_efectivos)
        AND c.instante >= l.emitido_en AND c.instante <= p_corte
   ) ct
  WHERE ct.instante IS NOT NULL AND cardinality(p_resultados_efectivos) > 0
    AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento r
                     WHERE r.llamamiento_ref = l.llamamiento_ref AND r.participacion_ref = p_participacion_ref)
$f$;

-- Una solicitud está pendiente hasta que RRHH registra la operación B8 que la
-- cita como justificante.
CREATE FUNCTION vec_bolsa_llamamientos.solicitud_portal_pendiente_v1(p_solicitud_ref text)
RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.operacion_situacion_participacion o
                     WHERE o.justificante_tipo = 'solicitud_candidato' AND o.justificante_ref = p_solicitud_ref)
$f$;

-- Comprobación común del material: recurso propio, vínculo candidato único en
-- el contexto y la acción esperada. Devuelve la participación.
CREATE FUNCTION vec_bolsa_llamamientos.exigir_portal_candidato_v1(p_candidato_ref text, p_bolsa_ref text, p_accion text, p_capacidad bytea, p_decision bytea, p_contexto bytea)
RETURNS text LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE c jsonb; d jsonb; x jsonb; n integer;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$' THEN
  RAISE EXCEPTION 'portal del candidato no autorizado' USING ERRCODE='42501';
 END IF;
 BEGIN c := convert_from(p_capacidad,'UTF8')::jsonb; d := convert_from(p_decision,'UTF8')::jsonb; x := convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'portal del candidato inválido' USING ERRCODE='22023'; END;
 IF c->>'efecto_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref OR d->>'recurso_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
    OR c->>'operacion' IS DISTINCT FROM p_accion OR d->>'accion' IS DISTINCT FROM p_accion
    OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array' THEN
  RAISE EXCEPTION 'portal del candidato denegado' USING ERRCODE='42501';
 END IF;
 SELECT count(*) INTO n FROM jsonb_array_elements(x->'vinculos') e WHERE e->>'tipo'='candidato' AND e->>'estado'='activo';
 IF n <> 1 OR NOT EXISTS (SELECT 1 FROM jsonb_array_elements(x->'vinculos') e WHERE e->>'tipo'='candidato' AND e->>'estado'='activo' AND e->>'referencia'=p_candidato_ref) THEN
  RAISE EXCEPTION 'portal del candidato denegado' USING ERRCODE='42501';
 END IF;
 RETURN vec_bolsa_llamamientos.participacion_portal_candidato_v1(p_candidato_ref, p_bolsa_ref);
END $f$;

-- Núcleo de la solicitud, sin material: solo lo alcanza la función pública
-- tras consumir la decisión (y las pruebas como propietario).
CREATE FUNCTION vec_bolsa_llamamientos.registrar_solicitud_portal_interna_v1(
 p_solicitud_ref text, p_recibo_ref text, p_candidato_ref text, p_bolsa_ref text, p_participacion_ref text, p_tipo text,
 p_pausa_hasta timestamptz, p_pausa_maxima timestamptz, p_situaciones_admitidas text[], p_regla_ref text, p_clave text,
 p_registrada_en timestamptz, p_decision_ref text)
RETURNS TABLE(reutilizada boolean, solicitud_ref text, recibo_ref text, registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE v_situacion text;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_tipo NOT IN ('pausa','reactivacion') OR p_registrada_en IS NULL
    OR cardinality(p_situaciones_admitidas) IS NULL OR cardinality(p_situaciones_admitidas) = 0
    OR (p_tipo = 'pausa' AND (p_pausa_hasta IS NULL OR p_pausa_maxima IS NULL OR p_pausa_hasta <= p_registrada_en OR p_pausa_hasta > p_pausa_maxima))
    OR (p_tipo = 'reactivacion' AND (p_pausa_hasta IS NOT NULL OR p_pausa_maxima IS NOT NULL)) THEN
  RAISE EXCEPTION 'solicitud del portal inválida' USING ERRCODE='22023';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:portal:'||p_participacion_ref, 0));
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.solicitud_portal_candidato s
             WHERE s.participacion_ref = p_participacion_ref AND vec_bolsa_llamamientos.solicitud_portal_pendiente_v1(s.solicitud_ref)) THEN
  RAISE EXCEPTION 'ya hay una solicitud pendiente de RRHH' USING ERRCODE='VBP02';
 END IF;
 SELECT s.situacion INTO v_situacion FROM vec_bolsa_llamamientos.situacion_participacion s
  WHERE s.participacion_ref = p_participacion_ref AND s.desde <= p_registrada_en ORDER BY s.desde DESC LIMIT 1;
 -- Sin hecho B2 la participación está en su situación inicial, disponible.
 v_situacion := coalesce(v_situacion, 'disponible');
 IF NOT v_situacion = ANY (p_situaciones_admitidas) THEN
  RAISE EXCEPTION 'la situación actual no admite esta solicitud' USING ERRCODE='VBP03';
 END IF;
 INSERT INTO vec_bolsa_llamamientos.solicitud_portal_candidato(solicitud_ref, recibo_ref, bolsa_ref, participacion_ref, candidato_ref, tipo,
   pausa_hasta, situacion_previa, regla_ref, clave_idempotencia, decision_ref, registrada_en)
 VALUES (p_solicitud_ref, p_recibo_ref, p_bolsa_ref, p_participacion_ref, p_candidato_ref, p_tipo,
   p_pausa_hasta, v_situacion, p_regla_ref, p_clave, p_decision_ref, p_registrada_en);
 RETURN QUERY SELECT false, p_solicitud_ref, p_recibo_ref, p_registrada_en;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.solicitar_portal_candidato_v1(
 p_solicitud_ref text, p_recibo_ref text, p_candidato_ref text, p_bolsa_ref text, p_tipo text,
 p_pausa_hasta timestamptz, p_pausa_maxima timestamptz, p_situaciones_admitidas text[], p_regla_ref text, p_clave text, p_registrada_en timestamptz,
 p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric,
 p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, solicitud_ref text, recibo_ref text, registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET lock_timeout = '2s' AS $f$
DECLARE v_participacion text; v_previa vec_bolsa_llamamientos.solicitud_portal_candidato%ROWTYPE; v_consumo record; v_accion text;
BEGIN
 v_accion := CASE p_tipo WHEN 'pausa' THEN 'bolsa.participaciones_propias.solicitar_pausa'
                         WHEN 'reactivacion' THEN 'bolsa.participaciones_propias.solicitar_reactivacion' END;
 IF v_accion IS NULL THEN RAISE EXCEPTION 'solicitud del portal inválida' USING ERRCODE='22023'; END IF;
 v_participacion := vec_bolsa_llamamientos.exigir_portal_candidato_v1(p_candidato_ref, p_bolsa_ref, v_accion, p_capacidad, p_decision, p_contexto);
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.solicitud_portal_candidato s
  WHERE s.participacion_ref = v_participacion AND s.clave_idempotencia = p_clave;
 IF FOUND THEN
  IF v_previa.tipo <> p_tipo OR v_previa.pausa_hasta IS DISTINCT FROM p_pausa_hasta OR v_previa.candidato_ref <> p_candidato_ref THEN
   RAISE EXCEPTION 'clave idempotente reutilizada con otra solicitud' USING ERRCODE='VBP01';
  END IF;
  RETURN QUERY SELECT true, v_previa.solicitud_ref, v_previa.recibo_ref, v_previa.registrada_en;
  RETURN;
 END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(
  p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref THEN
  RAISE EXCEPTION 'portal del candidato denegado' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT * FROM vec_bolsa_llamamientos.registrar_solicitud_portal_interna_v1(
  p_solicitud_ref, p_recibo_ref, p_candidato_ref, p_bolsa_ref, v_participacion, p_tipo, p_pausa_hasta, p_pausa_maxima,
  p_situaciones_admitidas, p_regla_ref, p_clave, p_registrada_en, v_consumo.decision_ref);
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.registrar_respuesta_portal_interna_v1(
 p_respuesta_ref text, p_recibo_ref text, p_candidato_ref text, p_bolsa_ref text, p_participacion_ref text, p_respuesta text,
 p_causa text, p_justificante_ref text, p_justificante_sha256 text, p_modo text, p_contacto_en timestamptz, p_vence_antes_de timestamptz,
 p_resultados_efectivos text[], p_regla_ref text, p_clave text, p_respondida_en timestamptz, p_decision_ref text)
RETURNS TABLE(reutilizada boolean, respuesta_ref text, recibo_ref text, respondida_en timestamptz, modo text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE v_abierto record;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_respuesta NOT IN ('acepta','renuncia','renuncia_justificada')
    OR p_modo NOT IN ('firme','propuesta_rrhh') OR p_respondida_en IS NULL OR p_vence_antes_de <= p_contacto_en THEN
  RAISE EXCEPTION 'respuesta del portal inválida' USING ERRCODE='22023';
 END IF;
 -- Sin contacto leído por el servidor no hay llamamiento que responder.
 IF p_contacto_en IS NULL OR p_vence_antes_de IS NULL THEN
  RAISE EXCEPTION 'no hay un llamamiento abierto' USING ERRCODE='VBP04';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:portal:'||p_participacion_ref, 0));
 SELECT * INTO v_abierto FROM vec_bolsa_llamamientos.llamamiento_abierto_portal_v1(p_participacion_ref, p_respondida_en, p_resultados_efectivos);
 IF v_abierto.llamamiento_ref IS NULL OR v_abierto.contacto_en IS DISTINCT FROM p_contacto_en THEN
  RAISE EXCEPTION 'no hay un llamamiento abierto con ese contacto' USING ERRCODE='VBP04';
 END IF;
 -- Fuera de plazo no se admite: queda como «sin respuesta» para RRHH.
 IF p_respondida_en >= p_vence_antes_de THEN
  RAISE EXCEPTION 'respuesta fuera de plazo' USING ERRCODE='VBP05';
 END IF;
 INSERT INTO vec_bolsa_llamamientos.respuesta_portal_llamamiento(respuesta_ref, recibo_ref, bolsa_ref, participacion_ref, candidato_ref,
   llamamiento_ref, respuesta, causa, justificante_ref, justificante_sha256, modo, contacto_en, vence_antes_de, regla_ref,
   clave_idempotencia, decision_ref, respondida_en)
 VALUES (p_respuesta_ref, p_recibo_ref, p_bolsa_ref, p_participacion_ref, p_candidato_ref, v_abierto.llamamiento_ref, p_respuesta,
   p_causa, p_justificante_ref, p_justificante_sha256, p_modo, p_contacto_en, p_vence_antes_de, p_regla_ref, p_clave,
   p_decision_ref, p_respondida_en);
 RETURN QUERY SELECT false, p_respuesta_ref, p_recibo_ref, p_respondida_en, p_modo;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.responder_llamamiento_portal_v1(
 p_respuesta_ref text, p_recibo_ref text, p_candidato_ref text, p_bolsa_ref text, p_respuesta text,
 p_causa text, p_justificante_ref text, p_justificante_sha256 text, p_modo text, p_contacto_en timestamptz, p_vence_antes_de timestamptz,
 p_resultados_efectivos text[], p_regla_ref text, p_clave text, p_respondida_en timestamptz,
 p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric,
 p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, respuesta_ref text, recibo_ref text, respondida_en timestamptz, modo text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET lock_timeout = '2s' AS $f$
DECLARE v_participacion text; v_previa vec_bolsa_llamamientos.respuesta_portal_llamamiento%ROWTYPE; v_consumo record;
BEGIN
 v_participacion := vec_bolsa_llamamientos.exigir_portal_candidato_v1(p_candidato_ref, p_bolsa_ref,
   'bolsa.participaciones_propias.responder_llamamiento', p_capacidad, p_decision, p_contexto);
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento r
  WHERE r.participacion_ref = v_participacion AND r.clave_idempotencia = p_clave;
 IF FOUND THEN
  IF v_previa.respuesta <> p_respuesta OR v_previa.causa IS DISTINCT FROM p_causa OR v_previa.justificante_ref IS DISTINCT FROM p_justificante_ref
     OR v_previa.justificante_sha256 IS DISTINCT FROM p_justificante_sha256 OR v_previa.candidato_ref <> p_candidato_ref THEN
   RAISE EXCEPTION 'clave idempotente reutilizada con otra respuesta' USING ERRCODE='VBP01';
  END IF;
  RETURN QUERY SELECT true, v_previa.respuesta_ref, v_previa.recibo_ref, v_previa.respondida_en, v_previa.modo;
  RETURN;
 END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(
  p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref THEN
  RAISE EXCEPTION 'portal del candidato denegado' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT * FROM vec_bolsa_llamamientos.registrar_respuesta_portal_interna_v1(
  p_respuesta_ref, p_recibo_ref, p_candidato_ref, p_bolsa_ref, v_participacion, p_respuesta, p_causa, p_justificante_ref,
  p_justificante_sha256, p_modo, p_contacto_en, p_vence_antes_de, p_resultados_efectivos, p_regla_ref, p_clave,
  p_respondida_en, v_consumo.decision_ref);
END $f$;

-- Lectura del portal por participación del candidato. Solo se usa dentro de
-- la transacción que ya consumió la consulta propia (Mi bolsa) o la acción
-- propia del candidato; no expone referencias de participación.
CREATE FUNCTION vec_bolsa_llamamientos.leer_portal_candidato_v1(p_candidato_ref text, p_corte timestamptz, p_resultados_efectivos text[])
RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'bolsa', p.bolsa_ref,
   'llamamiento_abierto', CASE WHEN a.contacto_en IS NULL THEN NULL ELSE jsonb_build_object(
      'contacto_en', to_char(a.contacto_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) END,
   'solicitud_pendiente', (SELECT jsonb_build_object('tipo', s.tipo, 'recibo', s.recibo_ref,
        'registrada_en', to_char(s.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'pausa_hasta', CASE WHEN s.pausa_hasta IS NULL THEN NULL ELSE to_char(s.pausa_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END)
      FROM vec_bolsa_llamamientos.solicitud_portal_candidato s
     WHERE s.participacion_ref = p.participacion_ref AND s.registrada_en <= p_corte
       AND vec_bolsa_llamamientos.solicitud_portal_pendiente_v1(s.solicitud_ref)
     ORDER BY s.registrada_en DESC LIMIT 1),
   'ultima_respuesta', (SELECT jsonb_build_object('respuesta', r.respuesta, 'modo', r.modo, 'recibo', r.recibo_ref,
        'respondida_en', to_char(r.respondida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))
      FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento r
     WHERE r.participacion_ref = p.participacion_ref AND r.respondida_en <= p_corte
     ORDER BY r.respondida_en DESC LIMIT 1)
 ) ORDER BY p.bolsa_ref), '[]'::jsonb)
 FROM (SELECT DISTINCT ON (x.bolsa_ref) x.bolsa_ref, x.participacion_ref
         FROM vec_bolsa_llamamientos.listar_participaciones_candidato_v1(p_candidato_ref) x
        ORDER BY x.bolsa_ref, x.confirmada_en DESC) p
 LEFT JOIN LATERAL vec_bolsa_llamamientos.llamamiento_abierto_portal_v1(p.participacion_ref, p_corte, p_resultados_efectivos) a ON true
$f$;

-- Bandeja de RRHH: solicitudes pendientes y respuestas del portal, con el
-- mismo contrato que consultar_avisos_rrhh_v1.
CREATE FUNCTION vec_bolsa_llamamientos.consultar_avisos_portal_rrhh_v1(p_corte timestamptz)
RETURNS TABLE(tipo text, bolsa_ref text, referencia text, detalle jsonb, fecha timestamptz)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' AS $f$
 SELECT 'solicitud_portal'::text, s.bolsa_ref, s.solicitud_ref,
        jsonb_build_object('participacion_ref', s.participacion_ref, 'solicitud', s.tipo, 'pausa_hasta', s.pausa_hasta,
          'situacion_previa', s.situacion_previa, 'recibo_ref', s.recibo_ref, 'regla_ref', s.regla_ref),
        s.registrada_en
   FROM vec_bolsa_llamamientos.solicitud_portal_candidato s
  WHERE s.registrada_en <= p_corte AND vec_bolsa_llamamientos.solicitud_portal_pendiente_v1(s.solicitud_ref)
 UNION ALL
 SELECT 'respuesta_portal'::text, r.bolsa_ref, r.respuesta_ref,
        jsonb_build_object('participacion_ref', r.participacion_ref, 'llamamiento_ref', r.llamamiento_ref, 'respuesta', r.respuesta,
          'modo', r.modo, 'causa', r.causa, 'justificante_ref', r.justificante_ref, 'justificante_sha256', r.justificante_sha256,
          'contacto_en', r.contacto_en, 'vence_antes_de', r.vence_antes_de, 'recibo_ref', r.recibo_ref, 'regla_ref', r.regla_ref),
        r.respondida_en
   FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento r
  WHERE r.respondida_en <= p_corte
$f$;

DO $acl$
DECLARE f regprocedure; publicas regprocedure[] := ARRAY[
  'vec_bolsa_llamamientos.solicitar_portal_candidato_v1(text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_bolsa_llamamientos.responder_llamamiento_portal_v1(text,text,text,text,text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_bolsa_llamamientos.leer_portal_candidato_v1(text,timestamptz,text[])'::regprocedure,
  'vec_bolsa_llamamientos.consultar_avisos_portal_rrhh_v1(timestamptz)'::regprocedure];
 internas regprocedure[] := ARRAY[
  'vec_bolsa_llamamientos.participacion_portal_candidato_v1(text,text)'::regprocedure,
  'vec_bolsa_llamamientos.llamamiento_abierto_portal_v1(text,timestamptz,text[])'::regprocedure,
  'vec_bolsa_llamamientos.solicitud_portal_pendiente_v1(text)'::regprocedure,
  'vec_bolsa_llamamientos.exigir_portal_candidato_v1(text,text,text,bytea,bytea,bytea)'::regprocedure,
  'vec_bolsa_llamamientos.registrar_solicitud_portal_interna_v1(text,text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,text)'::regprocedure,
  'vec_bolsa_llamamientos.registrar_respuesta_portal_interna_v1(text,text,text,text,text,text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,text)'::regprocedure];
BEGIN
 FOREACH f IN ARRAY publicas || internas LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC', f::text);
 END LOOP;
 FOREACH f IN ARRAY publicas LOOP
  EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_bolsa_llamamientos_ejecutor', f::text);
 END LOOP;
 FOREACH f IN ARRAY publicas || internas LOOP
  IF EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl, acldefault('f', p.proowner))) a
              WHERE p.oid = f AND a.grantee <> p.proowner
                AND (a.grantee <> 'vec_bolsa_llamamientos_ejecutor'::regrole OR a.is_grantable OR NOT f = ANY (publicas)))
     OR (SELECT proowner FROM pg_proc WHERE oid = f) <> 'vec_bolsa_llamamientos_propietario'::regrole THEN
   RAISE EXCEPTION 'ACL del portal del candidato abierta' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $acl$;
COMMIT;
