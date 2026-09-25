\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000024', 0));

-- B13, Petición RRHH p. 2: histórico de contratos de cada candidato.
-- Bolsa no lee tablas de Contratación temporal. Recibe el evento que CT113
-- publica desde su outbox y lo guarda aquí, que es a la vez su inbox
-- idempotente (clave: evento_ref; misma huella = reentrega, otra = error) y
-- el histórico de la participación. La participación la resuelve Bolsa con
-- su propio llamamiento: CT solo conoce la referencia opaca del llamamiento.
-- Si el llamamiento no es de Bolsa, el evento se conserva sin participación.
-- Confianza en el relevo: Bolsa no puede comprobar en su base que el evento
-- exista en Contratación temporal (no lee sus tablas) y el evento no lleva
-- firma de origen. La garantía es la del relevo Go, único consumidor con
-- EXECUTE (rol ejecutor): lee cada evento con la función autorizada de
-- lectura de CT (000113/000115), lo entrega sin modificarlo y Bolsa exige
-- aquí su forma completa, la referencia determinista (tipo y origen) y la
-- huella exacta del cuerpo. Un evento forjado con la credencial del ejecutor
-- quedaría como historia informativa: no cambia la situación, el orden ni el
-- llamamiento de nadie. Si algún día un consumidor decide con este histórico,
-- el evento deberá llevar una firma de origen verificable aquí.
DO $precondicion$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.llamamiento_integracion_desarrollo') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.integracion_desarrollo') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.contrato_participacion') IS NOT NULL THEN
  RAISE EXCEPTION 'estado incompatible para el histórico de contratos B13' USING ERRCODE='55000';
 END IF;
END $precondicion$;

CREATE TABLE vec_bolsa_llamamientos.contrato_participacion (
    evento_ref text PRIMARY KEY CHECK (evento_ref ~ '^evento:ct:contrato-bolsa:[0-9a-f]{64}$'),
    huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
    evento jsonb NOT NULL CHECK (jsonb_typeof(evento) = 'object' AND octet_length(evento::text) <= 16384),
    origen_ref text NOT NULL CHECK (octet_length(origen_ref) <= 512 AND origen_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]*$'),
    origen_creada_en timestamptz(6) NOT NULL CHECK (isfinite(origen_creada_en)),
    tipo text NOT NULL CHECK (tipo ~ '^[a-z][a-z0-9_]{1,39}$'),
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    llamamiento_ref text NOT NULL,
    participacion_ref text,
    bolsa_ref text,
    inicio timestamptz(6),
    fin_previsto timestamptz(6),
    modalidad_clave text,
    categoria_ref text,
    causa_clave text,
    ocurrido_en timestamptz(6) NOT NULL,
    recibido_en timestamptz(6) NOT NULL,
    UNIQUE (origen_ref, tipo),
    CHECK ((participacion_ref IS NULL) = (bolsa_ref IS NULL)),
    CHECK (fin_previsto IS NULL OR inicio IS NULL OR fin_previsto >= inicio),
    CHECK (huella_sha256 = encode(sha256(convert_to(evento::text, 'UTF8')), 'hex')),
    CHECK ((evento->>'evento_ref' = evento_ref AND evento->>'tipo' = tipo AND evento->>'origen_ref' = origen_ref
        AND evento->>'organizacion_ref' = organizacion_ref AND evento->>'expediente_ref' = expediente_ref
        AND evento->>'llamamiento_ref' = llamamiento_ref) IS TRUE)
);
CREATE INDEX contrato_participacion_por_participacion
    ON vec_bolsa_llamamientos.contrato_participacion (participacion_ref, inicio DESC)
    WHERE participacion_ref IS NOT NULL;
CREATE INDEX contrato_participacion_cursor
    ON vec_bolsa_llamamientos.contrato_participacion (origen_creada_en DESC, origen_ref DESC);
ALTER TABLE vec_bolsa_llamamientos.contrato_participacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.contrato_participacion FORCE ROW LEVEL SECURITY;
CREATE POLICY contrato_participacion_solo_propietario ON vec_bolsa_llamamientos.contrato_participacion
    TO vec_bolsa_llamamientos_propietario
    USING (current_user = 'vec_bolsa_llamamientos_propietario')
    WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.contrato_participacion FROM PUBLIC;
CREATE TRIGGER contrato_participacion_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.contrato_participacion
    FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

CREATE FUNCTION vec_bolsa_llamamientos.instante_contrato_valido(p jsonb, p_nulo boolean)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path = pg_catalog AS $f$
BEGIN
 IF p IS NULL OR jsonb_typeof(p) = 'null' THEN RETURN p_nulo; END IF;
 IF jsonb_typeof(p) <> 'string' OR p #>> '{}' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$' THEN RETURN false; END IF;
 RETURN isfinite((p #>> '{}')::timestamptz);
EXCEPTION WHEN others THEN RETURN false;
END $f$;

-- Inbox: la función recibe el evento publicado por CT113, lo valida
-- completo, resuelve la participación con el llamamiento propio de Bolsa y
-- lo registra una sola vez. Una reentrega idéntica devuelve reutilizado.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_contrato_participacion_v1(
 p_evento jsonb, p_huella_sha256 text, p_origen_creada_en timestamptz)
RETURNS TABLE(reutilizado boolean, participacion_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' SET lock_timeout = '2s' AS $f$
DECLARE v_previo record; v_llamamiento record; v_ref text; v_clave text := '^[a-z][a-z0-9._-]{1,79}$';
 v_opaca text := '^[A-Za-z0-9][A-Za-z0-9:._/-]*$';
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN
  RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='registro de contrato no autorizado';
 END IF;
 IF p_evento IS NULL OR jsonb_typeof(p_evento) <> 'object' OR octet_length(p_evento::text) > 16384
    OR EXISTS (SELECT 1 FROM jsonb_each(p_evento) e WHERE jsonb_typeof(e.value) = 'string' AND octet_length(e.value #>> '{}') > 512)
    OR p_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(p_evento::text, 'UTF8')), 'hex')
    OR p_origen_creada_en IS NULL OR NOT isfinite(p_origen_creada_en)
    OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(p_evento) k) IS DISTINCT FROM
       ARRAY['categoria_ref','causa_clave','esquema','evento_ref','expediente_ref','fin_previsto','inicio',
             'llamamiento_ref','modalidad_clave','ocurrido_en','organizacion_ref','origen_ref','tipo']
    OR p_evento->>'esquema' IS DISTINCT FROM 'vec.contratacion-temporal.contrato-bolsa.v1'
    OR jsonb_typeof(p_evento->'tipo') IS DISTINCT FROM 'string' OR p_evento->>'tipo' !~ '^[a-z][a-z0-9_]{1,39}$'
    OR jsonb_typeof(p_evento->'origen_ref') IS DISTINCT FROM 'string' OR p_evento->>'origen_ref' !~ v_opaca
    OR jsonb_typeof(p_evento->'organizacion_ref') IS DISTINCT FROM 'string' OR p_evento->>'organizacion_ref' !~ v_opaca
    OR jsonb_typeof(p_evento->'expediente_ref') IS DISTINCT FROM 'string' OR p_evento->>'expediente_ref' !~ v_opaca
    OR jsonb_typeof(p_evento->'llamamiento_ref') IS DISTINCT FROM 'string' OR p_evento->>'llamamiento_ref' !~ v_opaca
    OR p_evento->>'evento_ref' IS DISTINCT FROM 'evento:ct:contrato-bolsa:' || encode(sha256(convert_to(
         (p_evento->>'tipo') || chr(31) || (p_evento->>'origen_ref'), 'UTF8')), 'hex')
    OR NOT (jsonb_typeof(p_evento->'modalidad_clave') = 'null' OR (jsonb_typeof(p_evento->'modalidad_clave') = 'string' AND p_evento->>'modalidad_clave' ~ v_clave))
    OR NOT (jsonb_typeof(p_evento->'causa_clave') = 'null' OR (jsonb_typeof(p_evento->'causa_clave') = 'string' AND p_evento->>'causa_clave' ~ v_clave))
    OR NOT (jsonb_typeof(p_evento->'categoria_ref') = 'null' OR (jsonb_typeof(p_evento->'categoria_ref') = 'string' AND p_evento->>'categoria_ref' ~ v_opaca))
    OR NOT vec_bolsa_llamamientos.instante_contrato_valido(p_evento->'inicio', true)
    OR NOT vec_bolsa_llamamientos.instante_contrato_valido(p_evento->'fin_previsto', true)
    OR NOT vec_bolsa_llamamientos.instante_contrato_valido(p_evento->'ocurrido_en', false)
    OR (p_evento->>'fin_previsto')::timestamptz < (p_evento->>'inicio')::timestamptz THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='evento de contrato inválido';
 END IF;
 v_ref := p_evento->>'evento_ref';
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:contrato-participacion:' || v_ref, 0));
 SELECT c.huella_sha256, c.participacion_ref INTO v_previo
   FROM vec_bolsa_llamamientos.contrato_participacion c WHERE c.evento_ref = v_ref;
 IF FOUND THEN
  IF v_previo.huella_sha256 <> p_huella_sha256 THEN
   RAISE EXCEPTION USING ERRCODE='VBC01', MESSAGE='evento de contrato reentregado con otro contenido';
  END IF;
  RETURN QUERY SELECT true, v_previo.participacion_ref;
  RETURN;
 END IF;
 SELECT l.bolsa_ref, convert_from(i.registro_canonico, 'UTF8')::jsonb #>> '{propuesta,participacion_seleccionada_ref}' AS participacion
   INTO v_llamamiento
   FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo l
   JOIN vec_bolsa_llamamientos.integracion_desarrollo i ON i.operacion_ref = l.operacion_ref
  WHERE l.llamamiento_ref = p_evento->>'llamamiento_ref';
 IF v_llamamiento.participacion IS NULL THEN
  v_llamamiento.bolsa_ref := NULL;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.contrato_participacion(
   evento_ref, huella_sha256, evento, origen_ref, origen_creada_en, tipo, organizacion_ref, expediente_ref,
   llamamiento_ref, participacion_ref, bolsa_ref, inicio, fin_previsto, modalidad_clave, categoria_ref,
   causa_clave, ocurrido_en, recibido_en)
 VALUES (v_ref, p_huella_sha256, p_evento, p_evento->>'origen_ref', date_trunc('microseconds', p_origen_creada_en),
   p_evento->>'tipo', p_evento->>'organizacion_ref', p_evento->>'expediente_ref', p_evento->>'llamamiento_ref',
   v_llamamiento.participacion, v_llamamiento.bolsa_ref, (p_evento->>'inicio')::timestamptz,
   (p_evento->>'fin_previsto')::timestamptz, p_evento->>'modalidad_clave', p_evento->>'categoria_ref',
   p_evento->>'causa_clave', (p_evento->>'ocurrido_en')::timestamptz, date_trunc('microseconds', clock_timestamp()));
 RETURN QUERY SELECT false, v_llamamiento.participacion;
EXCEPTION WHEN unique_violation THEN
 RAISE EXCEPTION USING ERRCODE='VBC01', MESSAGE='origen de contrato ya registrado con otro evento';
END $f$;

-- Cursor del consumidor: el último origen recibido. El relevo relee desde
-- aquí con una ventana configurable; la idempotencia absorbe la relectura.
CREATE FUNCTION vec_bolsa_llamamientos.cursor_contratos_participacion_v1()
RETURNS TABLE(origen_creada_en timestamptz, origen_ref text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT c.origen_creada_en, c.origen_ref FROM vec_bolsa_llamamientos.contrato_participacion c
  ORDER BY c.origen_creada_en DESC, c.origen_ref DESC LIMIT 1
$f$;

-- Lectura RRHH en la ficha: mismo consumo V3 y mismas comprobaciones que el
-- historial B8 de la participación; sin material, ninguna fila.
CREATE FUNCTION vec_bolsa_llamamientos.listar_contratos_participacion_v1(
 p_participacion_ref text,p_actor text,p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(evento_ref text, tipo text, inicio timestamptz, fin_previsto timestamptz, modalidad_clave text,
 categoria_ref text, causa_clave text, expediente_ref text, llamamiento_ref text, ocurrido_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE v_consumo record; v_decision jsonb;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_participacion_ref IS NULL OR p_actor IS NULL THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de contratos no autorizada';
 END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(
  p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN v_decision:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de contratos no autorizada'; END;
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
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de contratos no autorizada';
 END IF;
 RETURN QUERY SELECT c.evento_ref, c.tipo, c.inicio, c.fin_previsto, c.modalidad_clave, c.categoria_ref, c.causa_clave,
   c.expediente_ref, c.llamamiento_ref, c.ocurrido_en
   FROM vec_bolsa_llamamientos.contrato_participacion c
  WHERE c.participacion_ref = p_participacion_ref
  ORDER BY coalesce(c.inicio, c.ocurrido_en) DESC, c.evento_ref DESC
  LIMIT 200;
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.instante_contrato_valido(jsonb,boolean) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_contrato_participacion_v1(jsonb,text,timestamptz) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.cursor_contratos_participacion_v1() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_contratos_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_contrato_participacion_v1(jsonb,text,timestamptz) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.cursor_contratos_participacion_v1() TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_contratos_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
