\set ON_ERROR_STOP on
-- Bolsa 000042: no incorporación tras una aceptación (duda 12 de RRHH,
-- respuesta de ejemplo: «no incorporación = baja y siguiente»).
--  * Bandeja: Contratación temporal (CT 000124) publica cada no incorporación
--    registrada por RRHH (`leer_no_incorporaciones_bolsa_v1`); un relevo Go
--    con su propio rol (`vec_bolsa_llamamientos_relevo_no_incorporacion`,
--    creado aquí sin LOGIN; la identidad LOGIN se aprovisiona fuera de Git) la
--    entrega aquí sin modificarla. Solo él puede entregar: el ejecutor
--    general de Bolsa no tiene EXECUTE sobre la bandeja.
--  * Origen verificable: antes de registrar nada, la bandeja pregunta a CT
--    (`no_incorporacion_publicada_bolsa_v1`, solo existencia: referencia de
--    origen, huella exacta del cuerpo y posición de publicación) si CT
--    publicó ese evento. Si no, la entrega queda en cuarentena sin efecto.
--    Bolsa no lee tablas de CT: llama a una función que CT le concede.
--  * Consecuencia: la resuelve esta base con la política de no incorporación
--    que la aplicación publica al arrancar desde el catálogo de reglas de
--    Bolsa (b24.sancion.* y el plazo del recurso b24.consecuencias), como
--    000033 y 000041. El relevo solo aporta las fechas del calendario (fin
--    del recurso y, si la regla lo pide, de la suspensión) y la política
--    comprueba la regla con la que se calcularon. Una clave que la política
--    no recoge no tiene efecto.
--  * Efecto: como una sanción B8 de solo adición (sanción, situación y
--    operación con la resolución y la segunda persona que la resolvió en
--    CT). No se aplica si Bolsa ya tiene la incorporación de ese llamamiento
--    en el histórico de contratos (000024).
--  * Evaluación: cada evento se evalúa al recibirlo y, mientras no se aplique,
--    en cada pasada del relevo (la aceptación puede llegar después, la
--    política puede cambiar o RRHH puede corregir la situación). Cada cambio
--    de resultado es una fila nueva de solo adición. Lo no aplicado aparece
--    en los avisos de Bolsa para RRHH (`consultar_avisos_rrhh_v3`).
--  * Siguiente llamamiento: la continuación (000006/000039) admite como
--    antecedente la aceptación de RRHH solo cuando su no incorporación quedó
--    APLICADA. Nada más cambia en el guardado.
--  * Otro evento distinto para el mismo llamamiento no sustituye al primero:
--    queda en la cuarentena y no tiene efecto.
-- Requiere 000019, 000024, 000026, 000032, 000033, 000039, 000041 y CT 000124.
-- Crea un rol de grupo: se instala con una sesión DBA, como Personal 000012.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000042',0));

DO $rol$
BEGIN
 IF to_regclass('vec_bolsa_llamamientos.no_incorporacion_bolsa') IS NOT NULL
    OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'vec_bolsa_llamamientos_relevo_no_incorporacion') THEN
  RAISE EXCEPTION 'Bolsa 000042 ya instalada: no se reaplica' USING ERRCODE='55000';
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = session_user AND rolsuper) THEN
  RAISE EXCEPTION 'Bolsa 000042: se instala con una sesión DBA' USING ERRCODE='55000';
 END IF;
 CREATE ROLE vec_bolsa_llamamientos_relevo_no_incorporacion
  NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_bolsa_llamamientos_relevo_no_incorporacion', current_database());
END $rol$;
COMMENT ON ROLE vec_bolsa_llamamientos_relevo_no_incorporacion IS
    'Bolsa 000042: grupo del relevo de no incorporaciones de CT; su LOGIN nominal se aprovisiona fuera de Git.';

SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
LOCK TABLE vec_bolsa_llamamientos.integracion_desarrollo IN SHARE ROW EXCLUSIVE MODE;

DO $precondicion$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN
  RAISE EXCEPTION 'Bolsa 000042: rol de migración incompatible' USING ERRCODE='55000';
 END IF;
 IF to_regclass('vec_bolsa_llamamientos.llamamiento_integracion_desarrollo') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.sancion_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.operacion_situacion_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.politica_transiciones_situacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.politica_segregacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.constitucion_entrada') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.contrato_participacion') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.instante_contrato_valido(jsonb,boolean)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_avisos_rrhh_v2(timestamptz)') IS NULL
    OR strpos((SELECT pg_get_constraintdef(oid,true) FROM pg_constraint
        WHERE conrelid='vec_bolsa_llamamientos.integracion_desarrollo'::regclass
          AND conname='integracion_desarrollo_tipo_check'),'expiracion_rrhh')=0 THEN
  RAISE EXCEPTION 'Bolsa 000042: dependencias incompatibles (000019, 000024, 000026, 000032, 000033, 000039 y 000041)' USING ERRCODE='55000';
 END IF;
 IF to_regprocedure('vec_contratacion_temporal.no_incorporacion_publicada_bolsa_v1(text,text,bigint)') IS NULL
    OR NOT has_function_privilege('vec_contratacion_temporal.no_incorporacion_publicada_bolsa_v1(text,text,bigint)','EXECUTE') THEN
  RAISE EXCEPTION 'Bolsa 000042: falta la comprobación de origen de CT 000124' USING ERRCODE='55000';
 END IF;
END $precondicion$;

GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_bolsa_llamamientos_relevo_no_incorporacion;

-- Política de no incorporación: consecuencias admitidas (b24.sancion.*) y la
-- regla del plazo del recurso. Versiones de solo adición; la vigente es la
-- de mayor versión. Sin ninguna publicada no hay consecuencia aplicable.
CREATE TABLE vec_bolsa_llamamientos.politica_no_incorporacion_bolsa (
    version bigint PRIMARY KEY CHECK (version >= 1),
    catalogo_ref text NOT NULL CHECK (octet_length(catalogo_ref) BETWEEN 1 AND 512 AND catalogo_ref = btrim(catalogo_ref)),
    catalogo_sha256 text NOT NULL CHECK (catalogo_sha256 ~ '^[a-f0-9]{64}$'),
    -- {clave: {etiqueta, efecto, regla_ref, regla_huella_sha256, con_plazo, orden_final, fin_automatico}}
    consecuencias jsonb NOT NULL CHECK (jsonb_typeof(consecuencias) = 'object' AND octet_length(consecuencias::text) <= 65536),
    recurso_regla_ref text NOT NULL CHECK (octet_length(recurso_regla_ref) BETWEEN 1 AND 300),
    recurso_regla_huella_sha256 text NOT NULL CHECK (recurso_regla_huella_sha256 ~ '^[a-f0-9]{64}$'),
    publicada_en timestamptz(6) NOT NULL,
    -- Constancia de quién publicó: la cuenta de conexión (session_user).
    publicada_por text NOT NULL DEFAULT session_user CHECK (octet_length(publicada_por) BETWEEN 1 AND 128)
);

-- Bandeja: eventos cuyo origen CT ha confirmado. Uno por llamamiento.
CREATE TABLE vec_bolsa_llamamientos.no_incorporacion_bolsa (
    evento_ref text PRIMARY KEY CHECK (evento_ref ~ '^evento:ct:no-incorporacion-bolsa:[0-9a-f]{64}$'),
    huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
    evento jsonb NOT NULL CHECK (jsonb_typeof(evento) = 'object' AND octet_length(evento::text) <= 16384),
    origen_ref text NOT NULL UNIQUE CHECK (octet_length(origen_ref) <= 512 AND origen_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]*$'),
    origen_creada_en timestamptz(6) NOT NULL CHECK (isfinite(origen_creada_en)),
    origen_posicion bigint NOT NULL CHECK (origen_posicion >= 0),
    llamamiento_ref text NOT NULL UNIQUE,
    recibido_en timestamptz(6) NOT NULL,
    recibido_por text NOT NULL DEFAULT session_user CHECK (octet_length(recibido_por) BETWEEN 1 AND 128),
    CHECK (huella_sha256 = encode(sha256(convert_to(evento::text, 'UTF8')), 'hex')),
    CHECK ((evento->>'evento_ref' = evento_ref AND evento->>'origen_ref' = origen_ref
        AND evento->>'llamamiento_ref' = llamamiento_ref) IS TRUE)
);
-- Evaluaciones de cada evento, de solo adición: la vigente es la de mayor
-- secuencia. Solo «aplicada» tiene efecto y habilita el siguiente llamamiento;
-- «llamamiento_ajeno» (el llamamiento no es de Bolsa) es definitiva; las demás
-- se reevalúan y aparecen en los avisos de RRHH.
CREATE TABLE vec_bolsa_llamamientos.no_incorporacion_bolsa_evaluacion (
    evento_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.no_incorporacion_bolsa(evento_ref),
    secuencia integer NOT NULL CHECK (secuencia BETWEEN 1 AND 100000),
    estado text NOT NULL CHECK (estado IN ('aplicada','llamamiento_ajeno','sin_aceptacion','participacion_no_constituida',
        'incorporacion_registrada','consecuencia_no_admitida','segunda_persona_ausente','transicion_no_admitida')),
    -- Aceptación de RRHH de la apertura del llamamiento (terminal).
    terminal_operacion_ref text REFERENCES vec_bolsa_llamamientos.integracion_desarrollo(operacion_ref),
    participacion_ref text,
    bolsa_ref text,
    politica_version bigint REFERENCES vec_bolsa_llamamientos.politica_no_incorporacion_bolsa(version),
    -- La consecuencia exacta que resolvió la política, con sus fechas.
    consecuencia jsonb CHECK (consecuencia IS NULL OR jsonb_typeof(consecuencia) = 'object'),
    sancion_ref text UNIQUE REFERENCES vec_bolsa_llamamientos.sancion_participacion(sancion_ref),
    evaluada_en timestamptz(6) NOT NULL,
    evaluada_por text NOT NULL DEFAULT session_user CHECK (octet_length(evaluada_por) BETWEEN 1 AND 128),
    PRIMARY KEY (evento_ref, secuencia),
    CHECK ((estado = 'aplicada') = (sancion_ref IS NOT NULL)),
    CHECK (estado <> 'aplicada' OR (terminal_operacion_ref IS NOT NULL AND participacion_ref IS NOT NULL
        AND consecuencia IS NOT NULL AND politica_version IS NOT NULL)),
    CHECK (participacion_ref IS NULL OR bolsa_ref IS NOT NULL)
);
CREATE UNIQUE INDEX no_incorporacion_bolsa_aplicada_evento ON vec_bolsa_llamamientos.no_incorporacion_bolsa_evaluacion (evento_ref) WHERE estado = 'aplicada';
CREATE UNIQUE INDEX no_incorporacion_bolsa_aplicada_terminal ON vec_bolsa_llamamientos.no_incorporacion_bolsa_evaluacion (terminal_operacion_ref) WHERE estado = 'aplicada';
CREATE INDEX no_incorporacion_bolsa_evaluacion_terminal ON vec_bolsa_llamamientos.no_incorporacion_bolsa_evaluacion (terminal_operacion_ref);
-- Entregas sin efecto, íntegras y de solo adición, para revisión:
--  * origen_no_verificado: CT no publicó ese evento exacto (forjado,
--    divergente o de otra posición);
--  * llamamiento_repetido: CT publicó otro evento distinto para un llamamiento
--    que ya tiene el suyo en la bandeja.
CREATE TABLE vec_bolsa_llamamientos.no_incorporacion_bolsa_cuarentena (
    evento_ref text NOT NULL CHECK (octet_length(evento_ref) <= 512),
    huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
    evento jsonb NOT NULL CHECK (jsonb_typeof(evento) = 'object' AND octet_length(evento::text) <= 16384),
    origen_ref text NOT NULL CHECK (octet_length(origen_ref) <= 512 AND origen_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]*$'),
    origen_creada_en timestamptz(6) NOT NULL CHECK (isfinite(origen_creada_en)),
    origen_posicion bigint NOT NULL CHECK (origen_posicion >= 0),
    motivo text NOT NULL CHECK (motivo IN ('origen_no_verificado','llamamiento_repetido')),
    recibido_en timestamptz(6) NOT NULL,
    recibido_por text NOT NULL DEFAULT session_user CHECK (octet_length(recibido_por) BETWEEN 1 AND 128),
    PRIMARY KEY (evento_ref, huella_sha256, origen_posicion),
    CHECK (huella_sha256 = encode(sha256(convert_to(evento::text, 'UTF8')), 'hex')),
    CHECK ((evento->>'evento_ref' = evento_ref AND evento->>'origen_ref' = origen_ref) IS TRUE)
);
CREATE INDEX no_incorporacion_bolsa_cursor ON vec_bolsa_llamamientos.no_incorporacion_bolsa (origen_posicion DESC, origen_ref DESC);
DO $seguridad$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['politica_no_incorporacion_bolsa','no_incorporacion_bolsa','no_incorporacion_bolsa_evaluacion',
                          'no_incorporacion_bolsa_cuarentena'] LOOP
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY %I ON vec_bolsa_llamamientos.%I TO vec_bolsa_llamamientos_propietario '
    'USING (current_user = ''vec_bolsa_llamamientos_propietario'') WITH CHECK (current_user = ''vec_bolsa_llamamientos_propietario'')',t||'_solo_propietario',t);
  EXECUTE format('REVOKE ALL ON vec_bolsa_llamamientos.%I FROM PUBLIC',t);
  EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.%I '
    'FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion()',t||'_inmutable',t);
 END LOOP;
END $seguridad$;

-- Forma de una consecuencia publicada desde el catálogo.
CREATE FUNCTION vec_bolsa_llamamientos.consecuencia_politica_valida_b42(p_clave text, c jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path = pg_catalog AS $f$
BEGIN
 RETURN coalesce(p_clave ~ '^b24[.]sancion[.][a-z0-9][a-z0-9._-]{0,110}$'
   AND jsonb_typeof(c) = 'object'
   AND (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(c) k) =
       ARRAY['con_plazo','efecto','etiqueta','fin_automatico','orden_final','regla_huella_sha256','regla_ref']
   AND c->>'efecto' IN ('ninguna','pausar','excluir')
   AND jsonb_typeof(c->'etiqueta') = 'string' AND octet_length(c->>'etiqueta') BETWEEN 1 AND 600 AND c->>'etiqueta' = btrim(c->>'etiqueta')
   AND jsonb_typeof(c->'regla_ref') = 'string' AND octet_length(c->>'regla_ref') BETWEEN 1 AND 300
   AND jsonb_typeof(c->'regla_huella_sha256') = 'string' AND c->>'regla_huella_sha256' ~ '^[a-f0-9]{64}$'
   AND jsonb_typeof(c->'con_plazo') = 'boolean' AND jsonb_typeof(c->'orden_final') = 'boolean'
   AND jsonb_typeof(c->'fin_automatico') = 'boolean'
   AND (NOT (c->'con_plazo')::boolean OR c->>'efecto' = 'pausar'), false);
EXCEPTION WHEN others THEN RETURN false;
END $f$;

-- Publica la política del catálogo. Solo crea versión si difiere de la
-- vigente. La publica la aplicación al arrancar con la cuenta de ejecución
-- de Bolsa, como 000033 y 000041; cada versión guarda la cuenta que la publicó.
CREATE FUNCTION vec_bolsa_llamamientos.publicar_politica_no_incorporacion_bolsa_v1(
 p_catalogo_ref text, p_catalogo_sha256 text, p_consecuencias jsonb, p_recurso_regla_ref text, p_recurso_regla_huella_sha256 text)
RETURNS TABLE(version bigint, reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET lock_timeout = '2s' SET statement_timeout = '5s' AS $f$
DECLARE v_vigente vec_bolsa_llamamientos.politica_no_incorporacion_bolsa;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='operacion no autorizada'; END IF;
 IF p_catalogo_ref IS NULL OR p_catalogo_sha256 IS NULL OR p_consecuencias IS NULL OR jsonb_typeof(p_consecuencias) <> 'object'
    OR p_recurso_regla_ref IS NULL OR p_recurso_regla_huella_sha256 IS NULL
    OR (SELECT count(*) FROM jsonb_object_keys(p_consecuencias)) > 64
    OR EXISTS (SELECT 1 FROM jsonb_each(p_consecuencias) e
                WHERE vec_bolsa_llamamientos.consecuencia_politica_valida_b42(e.key, e.value) IS NOT TRUE) THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='politica de no incorporacion invalida';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:politica_no_incorporacion_bolsa', 0));
 SELECT p.* INTO v_vigente FROM vec_bolsa_llamamientos.politica_no_incorporacion_bolsa p ORDER BY p.version DESC LIMIT 1;
 IF v_vigente.version IS NOT NULL AND v_vigente.catalogo_ref = p_catalogo_ref AND v_vigente.catalogo_sha256 = p_catalogo_sha256
    AND v_vigente.consecuencias = p_consecuencias AND v_vigente.recurso_regla_ref = p_recurso_regla_ref
    AND v_vigente.recurso_regla_huella_sha256 = p_recurso_regla_huella_sha256 THEN
  RETURN QUERY SELECT v_vigente.version, true;
  RETURN;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.politica_no_incorporacion_bolsa(version, catalogo_ref, catalogo_sha256, consecuencias,
   recurso_regla_ref, recurso_regla_huella_sha256, publicada_en)
 VALUES (coalesce(v_vigente.version, 0) + 1, p_catalogo_ref, p_catalogo_sha256, p_consecuencias, p_recurso_regla_ref,
   p_recurso_regla_huella_sha256, clock_timestamp());
 RETURN QUERY SELECT coalesce(v_vigente.version, 0) + 1, false;
EXCEPTION WHEN check_violation THEN
 RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='politica de no incorporacion invalida';
END $f$;

-- Consecuencia de la política vigente para la clave del evento, con las
-- fechas que calculó el calendario del catálogo (p_plazos: recurso_vence,
-- recurso_regla_ref, recurso_regla_huella_sha256 y suspension_hasta). Sin
-- consecuencia si la clave no está en la política, si las fechas no se
-- calcularon con su regla o si la consecuencia no es aplicable aquí (orden
-- final o fin automático, que exigen el circuito B8 completo).
CREATE FUNCTION vec_bolsa_llamamientos.consecuencia_no_incorporacion_b42(p_clave text, p_fecha date, p_plazos jsonb,
 OUT consecuencia jsonb, OUT politica_version bigint)
LANGUAGE plpgsql VOLATILE SET search_path = pg_catalog AS $f$
DECLARE v_politica vec_bolsa_llamamientos.politica_no_incorporacion_bolsa; c jsonb;
BEGIN
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_bolsa_llamamientos:politica_no_incorporacion_bolsa', 0));
 SELECT p.* INTO v_politica FROM vec_bolsa_llamamientos.politica_no_incorporacion_bolsa p ORDER BY p.version DESC LIMIT 1;
 IF v_politica.version IS NULL OR p_clave IS NULL OR NOT (v_politica.consecuencias ? p_clave) THEN RETURN; END IF;
 politica_version := v_politica.version;
 c := v_politica.consecuencias->p_clave;
 IF (c->'orden_final')::boolean OR (c->'fin_automatico')::boolean
    OR p_plazos IS NULL OR jsonb_typeof(p_plazos) <> 'object'
    OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(p_plazos) k) IS DISTINCT FROM
       ARRAY['recurso_regla_huella_sha256','recurso_regla_ref','recurso_vence','suspension_hasta']
    OR p_plazos->>'recurso_regla_ref' IS DISTINCT FROM v_politica.recurso_regla_ref
    OR p_plazos->>'recurso_regla_huella_sha256' IS DISTINCT FROM v_politica.recurso_regla_huella_sha256
    OR jsonb_typeof(p_plazos->'recurso_vence') <> 'string' OR p_plazos->>'recurso_vence' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
    OR (p_plazos->>'recurso_vence')::date <= p_fecha OR (p_plazos->>'recurso_vence')::date > p_fecha + 3660
    OR ((c->'con_plazo')::boolean AND NOT (jsonb_typeof(p_plazos->'suspension_hasta') = 'string'
        AND p_plazos->>'suspension_hasta' ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' AND (p_plazos->>'suspension_hasta')::date > p_fecha))
    OR (NOT (c->'con_plazo')::boolean AND jsonb_typeof(p_plazos->'suspension_hasta') <> 'null') THEN
  RETURN;
 END IF;
 consecuencia := jsonb_build_object('clave', p_clave, 'etiqueta', c->'etiqueta', 'efecto', c->'efecto',
   'regla_ref', c->'regla_ref', 'regla_huella_sha256', c->'regla_huella_sha256', 'orden_final', false, 'fin_automatico', false,
   'suspension_hasta', p_plazos->'suspension_hasta', 'recurso_vence', p_plazos->'recurso_vence',
   'recurso_regla_ref', v_politica.recurso_regla_ref, 'recurso_regla_huella_sha256', v_politica.recurso_regla_huella_sha256);
EXCEPTION WHEN invalid_datetime_format OR datetime_field_overflow OR invalid_text_representation THEN
 consecuencia := NULL;
END $f$;

-- Evalúa (o reevalúa) un evento de la bandeja y añade una evaluación solo si
-- el resultado cambia. Una evaluación aplicada o de llamamiento ajeno es
-- definitiva. Se llama con el cerrojo del evento tomado.
CREATE FUNCTION vec_bolsa_llamamientos.evaluar_no_incorporacion_b42(p_ref text, p_plazos jsonb)
RETURNS TABLE(estado text, participacion_ref text)
LANGUAGE plpgsql VOLATILE SET search_path = pg_catalog SET timezone = 'UTC' AS $f$
DECLARE n vec_bolsa_llamamientos.no_incorporacion_bolsa; v_ultima vec_bolsa_llamamientos.no_incorporacion_bolsa_evaluacion;
 v_terminal record; v_llamamiento record; v_estado text; v_participacion text; v_bolsa text; v_terminal_ref text;
 v_fecha date; v_ahora timestamptz(6); v_anterior record; v_situacion text; v_operacion text; v_politica record;
 v_segregacion text[]; v_sancion text; v_recibo text; v_motivo text; v_consecuencia jsonb; v_politica_version bigint;
BEGIN
 SELECT * INTO STRICT n FROM vec_bolsa_llamamientos.no_incorporacion_bolsa b WHERE b.evento_ref = p_ref;
 SELECT * INTO v_ultima FROM vec_bolsa_llamamientos.no_incorporacion_bolsa_evaluacion e
  WHERE e.evento_ref = p_ref ORDER BY e.secuencia DESC LIMIT 1;
 IF v_ultima.estado IN ('aplicada','llamamiento_ajeno') THEN
  RETURN QUERY SELECT v_ultima.estado, v_ultima.participacion_ref;
  RETURN;
 END IF;
 v_ahora := date_trunc('microseconds', clock_timestamp());
 v_fecha := (n.evento->>'fecha_notificacion')::date;
 SELECT l.bolsa_ref, l.operacion_ref INTO v_llamamiento FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo l
  WHERE l.llamamiento_ref = n.llamamiento_ref;
 IF NOT FOUND THEN
  v_estado := 'llamamiento_ajeno';
 ELSE
  v_bolsa := v_llamamiento.bolsa_ref;
  -- Aceptación de RRHH de la apertura de ese llamamiento, con su participación.
  SELECT i.operacion_ref, convert_from(a.registro_canonico, 'UTF8')::jsonb #>> '{propuesta,participacion_seleccionada_ref}' AS participacion
    INTO v_terminal
    FROM vec_bolsa_llamamientos.integracion_desarrollo a
    JOIN vec_bolsa_llamamientos.integracion_desarrollo i ON i.apertura_operacion_ref = a.operacion_ref AND i.tipo = 'aceptacion_rrhh'
   WHERE a.operacion_ref = v_llamamiento.operacion_ref AND a.tipo = 'propuesta'
   FOR SHARE OF i;
  IF FOUND AND v_terminal.participacion IS NOT NULL THEN
   v_terminal_ref := v_terminal.operacion_ref; v_participacion := v_terminal.participacion;
  END IF;
  IF v_terminal_ref IS NULL THEN
   v_estado := 'sin_aceptacion';
  ELSIF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.contrato_participacion c
                 WHERE c.llamamiento_ref = n.llamamiento_ref AND c.tipo = 'incorporacion') THEN
   -- Bolsa ya recibió la incorporación de ese llamamiento: no hay baja.
   v_estado := 'incorporacion_registrada';
  ELSIF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e
                      JOIN vec_bolsa_llamamientos.constitucion c USING (instantanea_ref, version_instantanea)
                     WHERE e.participacion_ref = v_participacion AND c.bolsa_ref = v_bolsa) THEN
   -- Participación de una fuente que Bolsa no ha constituido.
   v_estado := 'participacion_no_constituida';
  ELSE
   SELECT x.consecuencia, x.politica_version INTO v_consecuencia, v_politica_version
     FROM vec_bolsa_llamamientos.consecuencia_no_incorporacion_b42(n.evento->>'consecuencia_clave', v_fecha, p_plazos) x;
   IF v_consecuencia IS NULL OR v_fecha > (v_ahora AT TIME ZONE 'Europe/Madrid')::date THEN
    v_estado := 'consecuencia_no_admitida';
    v_consecuencia := NULL;
   ELSE
    v_operacion := v_consecuencia->>'efecto';
    PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_bolsa_llamamientos:politica_segregacion', 0));
    SELECT operaciones INTO STRICT v_segregacion FROM vec_bolsa_llamamientos.politica_segregacion ORDER BY version DESC LIMIT 1;
    IF (v_operacion = 'excluir' OR v_operacion = ANY (v_segregacion)) AND n.evento->>'resuelta_por' = n.evento->>'actor_ref' THEN
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
     v_sancion := 'sancion:' || encode(sha256(convert_to(v_participacion || chr(31) || p_ref, 'UTF8')), 'hex');
     v_motivo := v_consecuencia->>'etiqueta';
     IF v_operacion <> 'ninguna' THEN
      v_recibo := 'recibo:situacion:' || substr(v_sancion, 9);
      INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref, situacion, desde, hasta, fecha_disponible, motivo,
        actor, registrada_en, clave_idempotencia, recibo_ref, politica_transiciones_version)
      VALUES (v_participacion, v_situacion, v_ahora, NULL, NULL, v_motivo, n.evento->>'actor_ref', v_ahora, p_ref, v_recibo, v_politica.version);
      INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(participacion_ref, desde, operacion, justificante_tipo,
        justificante_ref, justificante_sha256, actor, validador, validada_en, registrada_en, clave_idempotencia)
      VALUES (v_participacion, v_ahora, v_operacion, 'resolucion', n.evento->>'resolucion_ref', n.evento->>'resolucion_sha256',
        n.evento->>'actor_ref', n.evento->>'resuelta_por', v_ahora, v_ahora, p_ref);
     ELSE
      v_recibo := 'recibo:sancion:' || substr(v_sancion, 9);
     END IF;
     INSERT INTO vec_bolsa_llamamientos.sancion_participacion(
       sancion_ref, participacion_ref, bolsa_ref, consecuencia, consecuencia_etiqueta, efecto, causa, fecha_notificacion,
       resolucion_ref, resolucion_sha256, resuelta_por, regla_ref, regla_huella_sha256, suspension_hasta, recurso_vence,
       recurso_regla_ref, recurso_regla_huella_sha256, situacion_desde, recibo_ref, actor, registrada_en, clave_idempotencia)
     VALUES (v_sancion, v_participacion, v_bolsa, v_consecuencia->>'clave', v_motivo, v_operacion, v_motivo, v_fecha,
       n.evento->>'resolucion_ref', n.evento->>'resolucion_sha256', n.evento->>'resuelta_por', v_consecuencia->>'regla_ref',
       v_consecuencia->>'regla_huella_sha256', (v_consecuencia->>'suspension_hasta')::date, (v_consecuencia->>'recurso_vence')::date,
       v_consecuencia->>'recurso_regla_ref', v_consecuencia->>'recurso_regla_huella_sha256',
       CASE WHEN v_operacion <> 'ninguna' THEN v_ahora END, v_recibo, n.evento->>'actor_ref', v_ahora, p_ref);
     v_estado := 'aplicada';
    END IF;
   END IF;
  END IF;
 END IF;
 IF v_ultima.estado IS NOT DISTINCT FROM v_estado AND v_ultima.terminal_operacion_ref IS NOT DISTINCT FROM v_terminal_ref
    AND v_ultima.participacion_ref IS NOT DISTINCT FROM v_participacion AND v_ultima.bolsa_ref IS NOT DISTINCT FROM v_bolsa THEN
  RETURN QUERY SELECT v_ultima.estado, v_ultima.participacion_ref;
  RETURN;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.no_incorporacion_bolsa_evaluacion(evento_ref, secuencia, estado, terminal_operacion_ref,
   participacion_ref, bolsa_ref, politica_version, consecuencia, sancion_ref, evaluada_en)
 VALUES (p_ref, coalesce(v_ultima.secuencia, 0) + 1, v_estado, v_terminal_ref, v_participacion, v_bolsa,
   CASE WHEN v_consecuencia IS NOT NULL THEN v_politica_version END, v_consecuencia, v_sancion, v_ahora);
 RETURN QUERY SELECT v_estado, v_participacion;
END $f$;

-- Bandeja: valida el evento completo, comprueba con CT que lo publicó y lo
-- registra una sola vez; después lo evalúa. Una reentrega idéntica lo
-- reevalúa si aún no está aplicado. Lo que CT no publicó, o un segundo evento
-- para el mismo llamamiento, queda en cuarentena y devuelve en_cuarentena,
-- sin error, para que el relevo continúe.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_no_incorporacion_bolsa_v1(
 p_evento jsonb, p_huella_sha256 text, p_origen_creada_en timestamptz, p_origen_posicion bigint, p_plazos jsonb)
RETURNS TABLE(reutilizado boolean, estado text, participacion_ref text, en_cuarentena boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' SET lock_timeout = '2s' AS $f$
DECLARE v_ref text; v_eval record; v_opaca text := '^[A-Za-z0-9][A-Za-z0-9:._/#-]*$'; v_motivo text;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN
  RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='registro de no incorporación no autorizado';
 END IF;
 IF p_evento IS NULL OR jsonb_typeof(p_evento) <> 'object' OR octet_length(p_evento::text) > 16384
    OR EXISTS (SELECT 1 FROM jsonb_each(p_evento) e WHERE jsonb_typeof(e.value) <> 'string' OR octet_length(e.value #>> '{}') > 512)
    OR p_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(p_evento::text, 'UTF8')), 'hex')
    OR p_origen_creada_en IS NULL OR NOT isfinite(p_origen_creada_en)
    OR p_origen_posicion IS NULL OR p_origen_posicion < 0
    OR (p_plazos IS NOT NULL AND (jsonb_typeof(p_plazos) <> 'object' OR octet_length(p_plazos::text) > 2048))
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
  PERFORM (p_evento->>'fecha_notificacion')::date;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='evento de no incorporación inválido';
 END;
 v_ref := p_evento->>'evento_ref';
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:no-incorporacion:llamamiento:' || (p_evento->>'llamamiento_ref'), 0));
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:no-incorporacion:' || v_ref, 0));
 -- Origen: CT dice si publicó exactamente este evento en esta posición.
 IF vec_contratacion_temporal.no_incorporacion_publicada_bolsa_v1(p_evento->>'origen_ref', p_huella_sha256, p_origen_posicion) IS NOT TRUE THEN
  v_motivo := 'origen_no_verificado';
 ELSIF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.no_incorporacion_bolsa n WHERE n.evento_ref = v_ref) THEN
  SELECT * INTO STRICT v_eval FROM vec_bolsa_llamamientos.evaluar_no_incorporacion_b42(v_ref, p_plazos);
  RETURN QUERY SELECT true, v_eval.estado, v_eval.participacion_ref, false;
  RETURN;
 ELSIF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.no_incorporacion_bolsa n WHERE n.llamamiento_ref = p_evento->>'llamamiento_ref') THEN
  v_motivo := 'llamamiento_repetido';
 END IF;
 IF v_motivo IS NOT NULL THEN
  INSERT INTO vec_bolsa_llamamientos.no_incorporacion_bolsa_cuarentena(evento_ref, huella_sha256, evento, origen_ref,
    origen_creada_en, origen_posicion, motivo, recibido_en)
  VALUES (v_ref, p_huella_sha256, p_evento, p_evento->>'origen_ref', date_trunc('microseconds', p_origen_creada_en),
    p_origen_posicion, v_motivo, date_trunc('microseconds', clock_timestamp()))
  ON CONFLICT DO NOTHING;
  RETURN QUERY SELECT false, NULL::text, NULL::text, true;
  RETURN;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.no_incorporacion_bolsa(evento_ref, huella_sha256, evento, origen_ref, origen_creada_en,
   origen_posicion, llamamiento_ref, recibido_en)
 VALUES (v_ref, p_huella_sha256, p_evento, p_evento->>'origen_ref', date_trunc('microseconds', p_origen_creada_en), p_origen_posicion,
   p_evento->>'llamamiento_ref', date_trunc('microseconds', clock_timestamp()));
 SELECT * INTO STRICT v_eval FROM vec_bolsa_llamamientos.evaluar_no_incorporacion_b42(v_ref, p_plazos);
 RETURN QUERY SELECT false, v_eval.estado, v_eval.participacion_ref, false;
END $f$;

-- Pendientes de reevaluar (lo no aplicado), en orden de publicación, con lo
-- que el relevo necesita para calcular las fechas del catálogo.
CREATE FUNCTION vec_bolsa_llamamientos.pendientes_no_incorporacion_bolsa_v1(p_limite integer)
RETURNS TABLE(evento_ref text, consecuencia_clave text, fecha_notificacion text)
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN
  RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='consulta de pendientes no autorizada';
 END IF;
 IF p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 100 THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='consulta de pendientes invalida';
 END IF;
 RETURN QUERY
 SELECT n.evento_ref, n.evento->>'consecuencia_clave', n.evento->>'fecha_notificacion'
   FROM vec_bolsa_llamamientos.no_incorporacion_bolsa n
   CROSS JOIN LATERAL (SELECT e.estado FROM vec_bolsa_llamamientos.no_incorporacion_bolsa_evaluacion e
                        WHERE e.evento_ref = n.evento_ref ORDER BY e.secuencia DESC LIMIT 1) u
  WHERE u.estado NOT IN ('aplicada','llamamiento_ajeno')
  ORDER BY n.origen_posicion, n.origen_ref
  LIMIT p_limite;
END $f$;

-- Reevaluación de un pendiente con las fechas que calcula el relevo.
CREATE FUNCTION vec_bolsa_llamamientos.reevaluar_no_incorporacion_bolsa_v1(p_evento_ref text, p_plazos jsonb)
RETURNS TABLE(estado text, participacion_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' SET lock_timeout = '2s' AS $f$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN
  RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='reevaluación de no incorporación no autorizada';
 END IF;
 IF p_evento_ref IS NULL OR p_evento_ref !~ '^evento:ct:no-incorporacion-bolsa:[0-9a-f]{64}$'
    OR (p_plazos IS NOT NULL AND (jsonb_typeof(p_plazos) <> 'object' OR octet_length(p_plazos::text) > 2048)) THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='reevaluación de no incorporación inválida';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:no-incorporacion:' || p_evento_ref, 0));
 IF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.no_incorporacion_bolsa n WHERE n.evento_ref = p_evento_ref) THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='reevaluación de no incorporación inválida';
 END IF;
 RETURN QUERY SELECT * FROM vec_bolsa_llamamientos.evaluar_no_incorporacion_b42(p_evento_ref, p_plazos);
END $f$;

-- Cursor del consumidor, como 000024: solo lo que CT publicó (bandeja y
-- segundos eventos de un mismo llamamiento); lo no verificado no lo mueve.
CREATE FUNCTION vec_bolsa_llamamientos.cursor_no_incorporaciones_bolsa_v1()
RETURNS TABLE(origen_posicion bigint, origen_ref text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT x.origen_posicion, x.origen_ref FROM (
   SELECT n.origen_posicion, n.origen_ref FROM vec_bolsa_llamamientos.no_incorporacion_bolsa n
   UNION ALL
   SELECT q.origen_posicion, q.origen_ref FROM vec_bolsa_llamamientos.no_incorporacion_bolsa_cuarentena q WHERE q.motivo = 'llamamiento_repetido') x
  ORDER BY x.origen_posicion DESC, x.origen_ref DESC LIMIT 1
$f$;

-- Antecedente del siguiente llamamiento: la aceptación cuya no incorporación
-- quedó aplicada. Cualquier otro resultado espera a RRHH.
CREATE FUNCTION vec_bolsa_llamamientos.no_incorporacion_registrada_b42(p_terminal text)
RETURNS boolean LANGUAGE sql STABLE SET search_path = pg_catalog AS $f$
 SELECT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.no_incorporacion_bolsa_evaluacion e
                 WHERE e.terminal_operacion_ref = p_terminal AND e.estado = 'aplicada')
$f$;

-- Avisos de RRHH: los de 000041 más las no incorporaciones sin aplicar de
-- llamamientos de Bolsa (revisión de RRHH; sin continuación hasta entonces).
CREATE FUNCTION vec_bolsa_llamamientos.consultar_avisos_rrhh_v3(p_corte timestamptz)
RETURNS TABLE(tipo text, bolsa_ref text, referencia text, detalle jsonb, fecha timestamptz)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' AS $f$
 SELECT v.tipo, v.bolsa_ref, v.referencia, v.detalle, v.fecha FROM vec_bolsa_llamamientos.consultar_avisos_rrhh_v2(p_corte) v
 UNION ALL
 SELECT 'no_incorporacion_revision'::text, u.bolsa_ref,
        'aviso:no-incorporacion:' || substr(n.evento_ref, 34),
        jsonb_strip_nulls(jsonb_build_object('estado', u.estado, 'llamamiento_ref', n.llamamiento_ref,
          'participacion_ref', u.participacion_ref, 'motivo_clave', n.evento->>'motivo_clave',
          'fecha_notificacion', n.evento->>'fecha_notificacion')),
        u.evaluada_en
   FROM vec_bolsa_llamamientos.no_incorporacion_bolsa n
   CROSS JOIN LATERAL (SELECT e.estado, e.bolsa_ref, e.participacion_ref, e.evaluada_en
                         FROM vec_bolsa_llamamientos.no_incorporacion_bolsa_evaluacion e
                        WHERE e.evento_ref = n.evento_ref AND e.evaluada_en <= p_corte
                        ORDER BY e.secuencia DESC LIMIT 1) u
  WHERE u.estado NOT IN ('aplicada','llamamiento_ajeno') AND u.bolsa_ref IS NOT NULL
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

-- ACL: el relevo solo entrega, reevalúa y lee su cursor; el ejecutor general
-- solo publica la política y consulta los avisos. Nada más.
DO $acl$
DECLARE f text;
 relevo text[] := ARRAY[
  'vec_bolsa_llamamientos.registrar_no_incorporacion_bolsa_v1(jsonb,text,timestamptz,bigint,jsonb)',
  'vec_bolsa_llamamientos.cursor_no_incorporaciones_bolsa_v1()',
  'vec_bolsa_llamamientos.pendientes_no_incorporacion_bolsa_v1(integer)',
  'vec_bolsa_llamamientos.reevaluar_no_incorporacion_bolsa_v1(text,jsonb)'];
 ejecutor text[] := ARRAY[
  'vec_bolsa_llamamientos.publicar_politica_no_incorporacion_bolsa_v1(text,text,jsonb,text,text)',
  'vec_bolsa_llamamientos.consultar_avisos_rrhh_v3(timestamptz)'];
 internas text[] := ARRAY[
  'vec_bolsa_llamamientos.consecuencia_politica_valida_b42(text,jsonb)',
  'vec_bolsa_llamamientos.consecuencia_no_incorporacion_b42(text,date,jsonb)',
  'vec_bolsa_llamamientos.evaluar_no_incorporacion_b42(text,jsonb)',
  'vec_bolsa_llamamientos.no_incorporacion_registrada_b42(text)'];
BEGIN
 FOREACH f IN ARRAY relevo || ejecutor || internas LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC', f);
 END LOOP;
 FOREACH f IN ARRAY relevo LOOP
  EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_bolsa_llamamientos_relevo_no_incorporacion', f);
 END LOOP;
 FOREACH f IN ARRAY ejecutor LOOP
  EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_bolsa_llamamientos_ejecutor', f);
 END LOOP;
 IF EXISTS (SELECT 1 FROM unnest(relevo) x WHERE has_function_privilege('vec_bolsa_llamamientos_ejecutor', x, 'EXECUTE'))
    OR EXISTS (SELECT 1 FROM unnest(ejecutor || internas) x WHERE has_function_privilege('vec_bolsa_llamamientos_relevo_no_incorporacion', x, 'EXECUTE'))
    OR EXISTS (SELECT 1 FROM unnest(internas) x WHERE has_function_privilege('vec_bolsa_llamamientos_ejecutor', x, 'EXECUTE'))
    OR EXISTS (SELECT 1 FROM unnest(ARRAY['politica_no_incorporacion_bolsa','no_incorporacion_bolsa','no_incorporacion_bolsa_evaluacion',
                 'no_incorporacion_bolsa_cuarentena']) t
                WHERE has_table_privilege('vec_bolsa_llamamientos_relevo_no_incorporacion','vec_bolsa_llamamientos.'||t,'SELECT,INSERT,UPDATE,DELETE')
                   OR has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.'||t,'SELECT,INSERT,UPDATE,DELETE')) THEN
  RAISE EXCEPTION 'Bolsa 000042: ACL efectiva incompatible' USING ERRCODE='42501';
 END IF;
END $acl$;
COMMENT ON TABLE vec_bolsa_llamamientos.no_incorporacion_bolsa IS
    'Bolsa 000042: bandeja de no incorporaciones publicadas por CT 000124 y verificadas contra su origen; una por llamamiento.';
COMMENT ON TABLE vec_bolsa_llamamientos.no_incorporacion_bolsa_evaluacion IS
    'Bolsa 000042: evaluaciones de solo adición; solo «aplicada» tiene efecto y habilita el siguiente llamamiento.';
COMMIT;
