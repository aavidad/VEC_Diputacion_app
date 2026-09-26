\set ON_ERROR_STOP on
-- Bolsa 000042: no incorporación tras una aceptación (duda 12 de RRHH,
-- respuesta de ejemplo: «no incorporación = baja y siguiente»).
--  * Bandeja: Contratación temporal (CT 000124) publica cada no incorporación
--    registrada por RRHH (`leer_no_incorporaciones_bolsa_v1`); el relevo Go la
--    entrega aquí sin modificarla. Bolsa no lee tablas de CT: resuelve con su
--    propio llamamiento la participación aceptada y aplica la consecuencia que
--    fija SU catálogo para la clave que trae el evento (b24.sancion.*: baja,
--    suspensión o ninguna), como una sanción B8 de solo adición: sanción,
--    situación y operación con la resolución (referencia y huella) y la
--    segunda persona que la resolvió en CT. Idempotente por evento; una
--    entrega divergente queda en cuarentena, como en 000024.
--  * Siguiente llamamiento: la continuación (000006/000039) admite como
--    antecedente la aceptación de RRHH cuando la bandeja registró su no
--    incorporación. Nada más cambia en el guardado.
-- Confianza en el relevo (como 000024): Bolsa no puede comprobar aquí que el
-- evento exista en CT y el evento no lleva firma de origen. La garantía es el
-- relevo Go, único consumidor con EXECUTE (rol ejecutor), que lo lee con la
-- función autorizada de CT; aquí se exige su forma completa, la referencia
-- determinista y la huella exacta, y el efecto solo alcanza a la participación
-- aceptada en el llamamiento propio de Bolsa, con segunda persona distinta
-- del registrador cuando la política de segregación lo exige.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000042',0));
LOCK TABLE vec_bolsa_llamamientos.integracion_desarrollo IN SHARE ROW EXCLUSIVE MODE;

DO $precondicion$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN
  RAISE EXCEPTION 'Bolsa 000042: rol de migración incompatible' USING ERRCODE='55000';
 END IF;
 IF to_regclass('vec_bolsa_llamamientos.no_incorporacion_bolsa') IS NOT NULL THEN
  RAISE EXCEPTION 'Bolsa 000042 ya instalada: no se reaplica' USING ERRCODE='55000';
 END IF;
 IF to_regclass('vec_bolsa_llamamientos.llamamiento_integracion_desarrollo') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.sancion_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.operacion_situacion_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.politica_transiciones_situacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.politica_segregacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.constitucion_entrada') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.instante_contrato_valido(jsonb,boolean)') IS NULL
    OR strpos((SELECT pg_get_constraintdef(oid,true) FROM pg_constraint
        WHERE conrelid='vec_bolsa_llamamientos.integracion_desarrollo'::regclass
          AND conname='integracion_desarrollo_tipo_check'),'expiracion_rrhh')=0 THEN
  RAISE EXCEPTION 'Bolsa 000042: dependencias incompatibles (000019, 000024, 000026, 000032, 000033 y 000039)' USING ERRCODE='55000';
 END IF;
END $precondicion$;

CREATE TABLE vec_bolsa_llamamientos.no_incorporacion_bolsa (
    evento_ref text PRIMARY KEY CHECK (evento_ref ~ '^evento:ct:no-incorporacion-bolsa:[0-9a-f]{64}$'),
    huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
    evento jsonb NOT NULL CHECK (jsonb_typeof(evento) = 'object' AND octet_length(evento::text) <= 16384),
    origen_ref text NOT NULL UNIQUE CHECK (octet_length(origen_ref) <= 512 AND origen_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]*$'),
    origen_creada_en timestamptz(6) NOT NULL CHECK (isfinite(origen_creada_en)),
    origen_posicion bigint NOT NULL CHECK (origen_posicion >= 0),
    llamamiento_ref text NOT NULL,
    -- Aceptación de RRHH del llamamiento propio de Bolsa (terminal de la
    -- apertura). Sin ella el evento se conserva sin efecto.
    terminal_operacion_ref text UNIQUE REFERENCES vec_bolsa_llamamientos.integracion_desarrollo(operacion_ref),
    participacion_ref text,
    bolsa_ref text,
    -- Lo que fijó el catálogo de Bolsa para la clave del evento.
    consecuencia jsonb CHECK (consecuencia IS NULL OR jsonb_typeof(consecuencia) = 'object'),
    estado text NOT NULL CHECK (estado IN ('aplicada','sin_aceptacion','participacion_no_constituida',
        'consecuencia_no_admitida','segunda_persona_ausente','transicion_no_admitida')),
    sancion_ref text REFERENCES vec_bolsa_llamamientos.sancion_participacion(sancion_ref),
    recibido_en timestamptz(6) NOT NULL,
    CHECK ((estado = 'sin_aceptacion') = (terminal_operacion_ref IS NULL)),
    CHECK ((estado = 'aplicada') = (sancion_ref IS NOT NULL)),
    CHECK ((participacion_ref IS NULL) = (bolsa_ref IS NULL)),
    CHECK (huella_sha256 = encode(sha256(convert_to(evento::text, 'UTF8')), 'hex')),
    CHECK ((evento->>'evento_ref' = evento_ref AND evento->>'origen_ref' = origen_ref
        AND evento->>'llamamiento_ref' = llamamiento_ref) IS TRUE)
);
CREATE TABLE vec_bolsa_llamamientos.no_incorporacion_bolsa_cuarentena (
    evento_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.no_incorporacion_bolsa(evento_ref),
    huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
    evento jsonb NOT NULL CHECK (jsonb_typeof(evento) = 'object' AND octet_length(evento::text) <= 16384),
    origen_ref text NOT NULL CHECK (octet_length(origen_ref) <= 512 AND origen_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]*$'),
    origen_creada_en timestamptz(6) NOT NULL CHECK (isfinite(origen_creada_en)),
    origen_posicion bigint NOT NULL CHECK (origen_posicion >= 0),
    recibido_en timestamptz(6) NOT NULL,
    PRIMARY KEY (evento_ref, huella_sha256, origen_posicion),
    CHECK (huella_sha256 = encode(sha256(convert_to(evento::text, 'UTF8')), 'hex')),
    CHECK ((evento->>'evento_ref' = evento_ref AND evento->>'origen_ref' = origen_ref) IS TRUE)
);
CREATE INDEX no_incorporacion_bolsa_cursor ON vec_bolsa_llamamientos.no_incorporacion_bolsa (origen_posicion DESC, origen_ref DESC);
DO $seguridad$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['no_incorporacion_bolsa','no_incorporacion_bolsa_cuarentena'] LOOP
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY %I ON vec_bolsa_llamamientos.%I TO vec_bolsa_llamamientos_propietario '
    'USING (current_user = ''vec_bolsa_llamamientos_propietario'') WITH CHECK (current_user = ''vec_bolsa_llamamientos_propietario'')',t||'_solo_propietario',t);
  EXECUTE format('REVOKE ALL ON vec_bolsa_llamamientos.%I FROM PUBLIC',t);
  EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.%I '
    'FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion()',t||'_inmutable',t);
 END LOOP;
END $seguridad$;

-- Consecuencia resuelta por el catálogo de Bolsa: forma exacta. Devuelve si
-- es aplicable aquí (baja, suspensión sin fin automático o ninguna, sin
-- colocación al final del orden).
CREATE FUNCTION vec_bolsa_llamamientos.consecuencia_no_incorporacion_valida_b42(c jsonb, p_clave text, p_fecha date)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path = pg_catalog AS $f$
BEGIN
 IF c IS NULL OR jsonb_typeof(c) <> 'object'
    OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(c) k) IS DISTINCT FROM
       ARRAY['clave','efecto','etiqueta','fin_automatico','orden_final','recurso_regla_huella_sha256','recurso_regla_ref',
             'recurso_vence','regla_huella_sha256','regla_ref','suspension_hasta']
    OR c->>'clave' IS DISTINCT FROM p_clave
    OR c->>'efecto' NOT IN ('ninguna','pausar','excluir')
    OR jsonb_typeof(c->'etiqueta') <> 'string' OR octet_length(c->>'etiqueta') NOT BETWEEN 1 AND 600 OR c->>'etiqueta' <> btrim(c->>'etiqueta')
    OR c->'orden_final' <> 'false'::jsonb OR c->'fin_automatico' <> 'false'::jsonb
    OR c->>'regla_huella_sha256' !~ '^[a-f0-9]{64}$' OR c->>'recurso_regla_huella_sha256' !~ '^[a-f0-9]{64}$'
    OR octet_length(c->>'regla_ref') NOT BETWEEN 1 AND 300 OR octet_length(c->>'recurso_regla_ref') NOT BETWEEN 1 AND 300
    OR c->>'recurso_vence' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' OR (c->>'recurso_vence')::date <= p_fecha
    OR NOT (jsonb_typeof(c->'suspension_hasta') = 'null'
            OR (c->>'efecto' = 'pausar' AND c->>'suspension_hasta' ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
                AND (c->>'suspension_hasta')::date > p_fecha)) THEN
  RETURN false;
 END IF;
 RETURN true;
EXCEPTION WHEN others THEN RETURN false;
END $f$;

-- Bandeja: valida el evento completo, resuelve con el llamamiento propio la
-- participación aceptada y aplica la consecuencia una sola vez. Una
-- reentrega idéntica devuelve reutilizado; una divergente queda en cuarentena
-- y devuelve en_cuarentena, sin error, para que el relevo continúe.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_no_incorporacion_bolsa_v1(
 p_evento jsonb, p_huella_sha256 text, p_origen_creada_en timestamptz, p_origen_posicion bigint, p_consecuencia jsonb)
RETURNS TABLE(reutilizado boolean, estado text, participacion_ref text, en_cuarentena boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' SET lock_timeout = '2s' AS $f$
DECLARE v_previo record; v_terminal record; v_terminal_ref text; v_participacion_aceptada text; v_ref text; v_estado text; v_participacion text; v_bolsa text;
 v_opaca text := '^[A-Za-z0-9][A-Za-z0-9:._/#-]*$'; v_fecha date; v_ahora timestamptz(6); v_anterior record;
 v_situacion text; v_operacion text; v_politica record; v_segregacion text[]; v_sancion text; v_recibo text; v_motivo text;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN
  RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='registro de no incorporación no autorizado';
 END IF;
 IF p_evento IS NULL OR jsonb_typeof(p_evento) <> 'object' OR octet_length(p_evento::text) > 16384
    OR EXISTS (SELECT 1 FROM jsonb_each(p_evento) e WHERE jsonb_typeof(e.value) <> 'string' OR octet_length(e.value #>> '{}') > 512)
    OR p_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(p_evento::text, 'UTF8')), 'hex')
    OR p_origen_creada_en IS NULL OR NOT isfinite(p_origen_creada_en)
    OR p_origen_posicion IS NULL OR p_origen_posicion < 0
    OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(p_evento) k) IS DISTINCT FROM
       ARRAY['actor_ref','consecuencia_clave','esquema','evento_ref','expediente_ref','fecha_notificacion','llamamiento_ref',
             'motivo_clave','ocurrido_en','organizacion_ref','origen_ref','resolucion_ref','resolucion_sha256','resuelta_por','tipo']
    OR p_evento->>'esquema' IS DISTINCT FROM 'vec.contratacion-temporal.no-incorporacion-bolsa.v1'
    OR p_evento->>'tipo' IS DISTINCT FROM 'no_incorporacion'
    OR p_evento->>'origen_ref' !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]*$'
    OR p_evento->>'organizacion_ref' !~ v_opaca OR p_evento->>'expediente_ref' !~ v_opaca
    OR p_evento->>'llamamiento_ref' !~ v_opaca OR p_evento->>'actor_ref' !~ v_opaca
    OR p_evento->>'resuelta_por' !~ v_opaca OR octet_length(p_evento->>'resuelta_por') > 256
    OR p_evento->>'resolucion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]*$' OR octet_length(p_evento->>'resolucion_ref') > 256
    OR p_evento->>'resolucion_sha256' !~ '^[a-f0-9]{64}$'
    OR p_evento->>'motivo_clave' !~ '^[a-z][a-z0-9_]{1,63}$'
    OR p_evento->>'consecuencia_clave' !~ '^[a-z0-9][a-z0-9._-]{0,127}$'
    OR p_evento->>'fecha_notificacion' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
    OR NOT vec_bolsa_llamamientos.instante_contrato_valido(p_evento->'ocurrido_en', false)
    OR p_evento->>'evento_ref' IS DISTINCT FROM 'evento:ct:no-incorporacion-bolsa:' || encode(sha256(convert_to(
         'no_incorporacion' || chr(31) || (p_evento->>'origen_ref'), 'UTF8')), 'hex') THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='evento de no incorporación inválido';
 END IF;
 BEGIN
  v_fecha := (p_evento->>'fecha_notificacion')::date;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='evento de no incorporación inválido';
 END;
 v_ref := p_evento->>'evento_ref';
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:no-incorporacion:' || v_ref, 0));
 SELECT n.huella_sha256, n.origen_posicion, n.estado, n.participacion_ref INTO v_previo
   FROM vec_bolsa_llamamientos.no_incorporacion_bolsa n WHERE n.evento_ref = v_ref;
 IF FOUND THEN
  IF v_previo.huella_sha256 <> p_huella_sha256 OR v_previo.origen_posicion <> p_origen_posicion THEN
   INSERT INTO vec_bolsa_llamamientos.no_incorporacion_bolsa_cuarentena(evento_ref, huella_sha256, evento, origen_ref,
     origen_creada_en, origen_posicion, recibido_en)
   VALUES (v_ref, p_huella_sha256, p_evento, p_evento->>'origen_ref', date_trunc('microseconds', p_origen_creada_en),
     p_origen_posicion, date_trunc('microseconds', clock_timestamp()))
   ON CONFLICT DO NOTHING;
   RETURN QUERY SELECT false, NULL::text, NULL::text, true;
   RETURN;
  END IF;
  RETURN QUERY SELECT true, v_previo.estado, v_previo.participacion_ref, false;
  RETURN;
 END IF;
 v_ahora := date_trunc('microseconds', clock_timestamp());
 -- Aceptación de RRHH de la apertura de ese llamamiento, con su participación.
 SELECT i.operacion_ref, l.bolsa_ref, convert_from(a.registro_canonico, 'UTF8')::jsonb #>> '{propuesta,participacion_seleccionada_ref}' AS participacion
   INTO v_terminal
   FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo l
   JOIN vec_bolsa_llamamientos.integracion_desarrollo a ON a.operacion_ref = l.operacion_ref AND a.tipo = 'propuesta'
   JOIN vec_bolsa_llamamientos.integracion_desarrollo i ON i.apertura_operacion_ref = a.operacion_ref AND i.tipo = 'aceptacion_rrhh'
  WHERE l.llamamiento_ref = p_evento->>'llamamiento_ref'
  FOR SHARE OF i;
 IF FOUND AND v_terminal.participacion IS NOT NULL THEN
  v_terminal_ref := v_terminal.operacion_ref; v_participacion_aceptada := v_terminal.participacion;
 END IF;
 IF v_terminal_ref IS NULL THEN
  v_estado := 'sin_aceptacion';
 ELSIF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e
                     JOIN vec_bolsa_llamamientos.constitucion c USING (instantanea_ref, version_instantanea)
                    WHERE e.participacion_ref = v_terminal.participacion AND c.bolsa_ref = v_terminal.bolsa_ref) THEN
  -- Participación de una fuente que Bolsa no ha constituido: se registra la
  -- no incorporación, pero no hay situación propia que cambiar.
  v_estado := 'participacion_no_constituida';
 ELSIF NOT vec_bolsa_llamamientos.consecuencia_no_incorporacion_valida_b42(p_consecuencia, p_evento->>'consecuencia_clave', v_fecha)
       OR v_fecha > (v_ahora AT TIME ZONE 'Europe/Madrid')::date THEN
  v_estado := 'consecuencia_no_admitida';
 ELSE
  v_participacion := v_terminal.participacion; v_bolsa := v_terminal.bolsa_ref;
  v_operacion := p_consecuencia->>'efecto';
  PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_bolsa_llamamientos:politica_segregacion', 0));
  SELECT operaciones INTO STRICT v_segregacion FROM vec_bolsa_llamamientos.politica_segregacion ORDER BY version DESC LIMIT 1;
  IF (v_operacion = 'excluir' OR v_operacion = ANY (v_segregacion)) AND p_evento->>'resuelta_por' = p_evento->>'actor_ref' THEN
   v_estado := 'segunda_persona_ausente';
  END IF;
  IF v_estado IS NULL AND v_operacion <> 'ninguna' THEN
   v_situacion := CASE v_operacion WHEN 'pausar' THEN 'no_disponible' ELSE 'excluido' END;
   SELECT s.situacion, s.desde INTO v_anterior FROM vec_bolsa_llamamientos.situacion_participacion s
    WHERE s.participacion_ref = v_participacion ORDER BY s.desde DESC LIMIT 1 FOR SHARE;
   PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_bolsa_llamamientos:politica_transiciones_situacion', 0));
   SELECT p.version, p.transiciones INTO STRICT v_politica FROM vec_bolsa_llamamientos.politica_transiciones_situacion p ORDER BY p.version DESC LIMIT 1;
   IF v_anterior.situacion IS NULL OR v_ahora <= v_anterior.desde
      OR NOT ((v_anterior.situacion || '>' || v_situacion) = ANY (v_politica.transiciones)) THEN
    v_estado := 'transicion_no_admitida';
   END IF;
  END IF;
  IF v_estado IS NULL THEN
   v_sancion := 'sancion:' || encode(sha256(convert_to(v_participacion || chr(31) || v_ref, 'UTF8')), 'hex');
   v_motivo := p_consecuencia->>'etiqueta';
   IF v_operacion <> 'ninguna' THEN
    v_recibo := 'recibo:situacion:' || substr(v_sancion, 9);
    INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref, situacion, desde, hasta, fecha_disponible, motivo,
      actor, registrada_en, clave_idempotencia, recibo_ref, politica_transiciones_version)
    VALUES (v_participacion, v_situacion, v_ahora, NULL, NULL, v_motivo, p_evento->>'actor_ref', v_ahora, v_ref, v_recibo, v_politica.version);
    INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(participacion_ref, desde, operacion, justificante_tipo,
      justificante_ref, justificante_sha256, actor, validador, validada_en, registrada_en, clave_idempotencia)
    VALUES (v_participacion, v_ahora, v_operacion, 'resolucion', p_evento->>'resolucion_ref', p_evento->>'resolucion_sha256',
      p_evento->>'actor_ref', p_evento->>'resuelta_por', v_ahora, v_ahora, v_ref);
   ELSE
    v_recibo := 'recibo:sancion:' || substr(v_sancion, 9);
   END IF;
   INSERT INTO vec_bolsa_llamamientos.sancion_participacion(
     sancion_ref, participacion_ref, bolsa_ref, consecuencia, consecuencia_etiqueta, efecto, causa, fecha_notificacion,
     resolucion_ref, resolucion_sha256, resuelta_por, regla_ref, regla_huella_sha256, suspension_hasta, recurso_vence,
     recurso_regla_ref, recurso_regla_huella_sha256, situacion_desde, recibo_ref, actor, registrada_en, clave_idempotencia)
   VALUES (v_sancion, v_participacion, v_bolsa, p_consecuencia->>'clave', p_consecuencia->>'etiqueta', v_operacion, v_motivo, v_fecha,
     p_evento->>'resolucion_ref', p_evento->>'resolucion_sha256', p_evento->>'resuelta_por', p_consecuencia->>'regla_ref',
     p_consecuencia->>'regla_huella_sha256', (p_consecuencia->>'suspension_hasta')::date, (p_consecuencia->>'recurso_vence')::date,
     p_consecuencia->>'recurso_regla_ref', p_consecuencia->>'recurso_regla_huella_sha256',
     CASE WHEN v_operacion <> 'ninguna' THEN v_ahora END, v_recibo, p_evento->>'actor_ref', v_ahora, v_ref);
   v_estado := 'aplicada';
  END IF;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.no_incorporacion_bolsa(evento_ref, huella_sha256, evento, origen_ref, origen_creada_en,
   origen_posicion, llamamiento_ref, terminal_operacion_ref, participacion_ref, bolsa_ref, consecuencia, estado, sancion_ref, recibido_en)
 VALUES (v_ref, p_huella_sha256, p_evento, p_evento->>'origen_ref', date_trunc('microseconds', p_origen_creada_en), p_origen_posicion,
   p_evento->>'llamamiento_ref', v_terminal_ref, v_participacion_aceptada,
   CASE WHEN v_participacion_aceptada IS NOT NULL THEN v_terminal.bolsa_ref END,
   p_consecuencia, v_estado, v_sancion, v_ahora);
 RETURN QUERY SELECT false, v_estado, v_participacion_aceptada, false;
EXCEPTION WHEN unique_violation THEN
 RAISE EXCEPTION USING ERRCODE='VBC01', MESSAGE='no incorporación ya registrada con otro evento';
END $f$;

-- Cursor del consumidor, como 000024: las entregas en cuarentena no lo mueven.
CREATE FUNCTION vec_bolsa_llamamientos.cursor_no_incorporaciones_bolsa_v1()
RETURNS TABLE(origen_posicion bigint, origen_ref text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT n.origen_posicion, n.origen_ref FROM vec_bolsa_llamamientos.no_incorporacion_bolsa n
  ORDER BY n.origen_posicion DESC, n.origen_ref DESC LIMIT 1
$f$;

-- Antecedente del siguiente llamamiento: la aceptación con no incorporación.
CREATE FUNCTION vec_bolsa_llamamientos.no_incorporacion_registrada_b42(p_terminal text)
RETURNS boolean LANGUAGE sql STABLE SET search_path = pg_catalog AS $f$
 SELECT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.no_incorporacion_bolsa n WHERE n.terminal_operacion_ref = p_terminal)
$f$;

DO $continuacion$
DECLARE v_def text; v_acl aclitem[]; v_cambio record; v_oid oid:='vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl FROM pg_proc p
  WHERE p.oid=v_oid AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef;
 FOR v_cambio IN SELECT * FROM (VALUES
   ($antes$    WHERE operacion_ref=v_cont->>'terminal_operacion_ref' AND tipo IN ('renuncia_rrhh','expiracion_rrhh') FOR SHARE;$antes$,
    $despues$    WHERE operacion_ref=v_cont->>'terminal_operacion_ref' AND (tipo IN ('renuncia_rrhh','expiracion_rrhh')
      OR (tipo='aceptacion_rrhh' AND vec_bolsa_llamamientos.no_incorporacion_registrada_b42(operacion_ref))) FOR SHARE;$despues$),
   ($antes$       (CASE v_terminal_anterior.tipo WHEN 'expiracion_rrhh' THEN 'expiracion_gobernada' ELSE 'renuncia' END) OR$antes$,
    $despues$       (CASE v_terminal_anterior.tipo WHEN 'expiracion_rrhh' THEN 'expiracion_gobernada' WHEN 'aceptacion_rrhh' THEN 'aceptacion' ELSE 'renuncia' END) OR$despues$)
 ) AS cambios(anterior,nuevo) LOOP
  IF length(v_def)-length(replace(v_def,v_cambio.anterior,''))<>length(v_cambio.anterior) THEN
   RAISE EXCEPTION 'Bolsa 000042: guardado incompatible' USING ERRCODE='55000';
  END IF;
  v_def:=replace(v_def,v_cambio.anterior,v_cambio.nuevo);
 END LOOP;
 EXECUTE v_def;
 IF pg_get_functiondef(v_oid) IS DISTINCT FROM v_def OR (SELECT proacl FROM pg_proc WHERE oid=v_oid) IS DISTINCT FROM v_acl
    OR (SELECT proowner FROM pg_proc WHERE oid=v_oid) IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
    OR (SELECT NOT prosecdef FROM pg_proc WHERE oid=v_oid) THEN
  RAISE EXCEPTION 'Bolsa 000042: guardado alterado fuera de contrato' USING ERRCODE='55000';
 END IF;
END $continuacion$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consecuencia_no_incorporacion_valida_b42(jsonb,text,date) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_no_incorporacion_bolsa_v1(jsonb,text,timestamptz,bigint,jsonb) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.cursor_no_incorporaciones_bolsa_v1() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.no_incorporacion_registrada_b42(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_no_incorporacion_bolsa_v1(jsonb,text,timestamptz,bigint,jsonb) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.cursor_no_incorporaciones_bolsa_v1() TO vec_bolsa_llamamientos_ejecutor;
COMMENT ON TABLE vec_bolsa_llamamientos.no_incorporacion_bolsa IS
    'Bolsa 000042: bandeja de no incorporaciones publicadas por CT 000124; consecuencia del catálogo de Bolsa aplicada una vez y antecedente del siguiente llamamiento.';
COMMIT;
