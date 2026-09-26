\set ON_ERROR_STOP on
-- Selección 000001. Solicitud de participación en una convocatoria
-- («Convoca integrado», fase 1). Esquema propio del módulo Selección; no lee
-- ni escribe tablas de otros módulos.
--
-- * Publicación mínima gobernada de la convocatoria: versiones de solo
--   adición (plazo, turnos, requisitos estructurados, baremo y formato del
--   justificante) que Selección publica al arrancar desde su catálogo. Una
--   publicación con el mismo contenido reutiliza la versión vigente.
-- * Solicitud: borrador versionado con control optimista, datos personales
--   cifrados fuera de la base (sobre) y documento de identidad reducido a
--   huella con clave y parcial; presentación idempotente con justificante
--   interno AAAA/SOL-NNNNNN (formato de la convocatoria), recibo, historia de
--   solo adición y outbox. El justificante no es un asiento de registro
--   administrativo ni una firma.
-- * El plazo se comprueba con el reloj de la base; un requisito obligatorio
--   que impide presentar declarado «no_cumple» impide presentar
--   («pendiente» no excluye).
-- * Toda lectura o escritura de datos de una solicitud consume en la misma
--   transacción una decisión V3: AD3-89 (la persona, sobre
--   'mis-solicitudes:<persona>') o AD3-90 (RRHH, con auditoría de acceso).
--
-- Orden: seleccion/roles_up.sql, AD3-89, AD3-90 y esta migración.
BEGIN;
SET LOCAL ROLE vec_seleccion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_seleccion:migracion:000001', 0));

DO $precondicion$
BEGIN
 IF current_user <> 'vec_seleccion_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_solicitud_propia_seleccion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_solicitudes_seleccion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'vec_seleccion_ejecutor' AND NOT rolcanlogin) THEN
  RAISE EXCEPTION 'dependencias de Selección 000001 ausentes (roles de Selección, AD3-89 y AD3-90)' USING ERRCODE = '55000';
 END IF;
 IF to_regnamespace('vec_seleccion') IS NOT NULL THEN
  RAISE EXCEPTION 'migración Selección 000001 ya aplicada' USING ERRCODE = '55000';
 END IF;
END $precondicion$;

CREATE SCHEMA vec_seleccion AUTHORIZATION vec_seleccion_propietario;
REVOKE ALL ON SCHEMA vec_seleccion FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_seleccion TO vec_seleccion_ejecutor;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_seleccion_propietario IN SCHEMA vec_seleccion REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_seleccion_propietario IN SCHEMA vec_seleccion REVOKE ALL ON FUNCTIONS FROM PUBLIC;

CREATE FUNCTION vec_seleccion.rechazar_mutacion() RETURNS trigger
LANGUAGE plpgsql SET search_path = pg_catalog AS $f$
BEGIN
 RAISE EXCEPTION 'historia de Selección de solo adición' USING ERRCODE = '55000';
END $f$;

-- Versiones publicadas de cada convocatoria. La vigente es la de versión
-- mayor. contenido: turnos, requisitos, baremo, numeracion, fecha_referencia.
CREATE TABLE vec_seleccion.convocatoria_publicada(
 convocatoria_ref text NOT NULL CHECK (convocatoria_ref ~ '^[a-z0-9][a-z0-9:._-]{2,190}$'),
 version integer NOT NULL CHECK (version >= 1),
 huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
 titulo text NOT NULL CHECK (octet_length(titulo) BETWEEN 1 AND 512 AND titulo = btrim(titulo)),
 abre_en timestamptz(6) NOT NULL,
 cierra_en timestamptz(6) NOT NULL,
 contenido jsonb NOT NULL,
 publicada_en timestamptz(6) NOT NULL,
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY (convocatoria_ref, version),
 CHECK (cierra_en > abre_en),
 CHECK (jsonb_typeof(contenido) = 'object' AND octet_length(contenido::text) <= 65536
        AND jsonb_typeof(contenido->'requisitos') = 'array' AND jsonb_typeof(contenido->'turnos') = 'array'
        AND jsonb_typeof(contenido->'baremo') = 'object' AND jsonb_typeof(contenido->'numeracion') = 'object'
        AND contenido->'numeracion'->>'patron' LIKE '%{anio}%' AND contenido->'numeracion'->>'patron' LIKE '%{numero}%'
        AND octet_length(contenido->'numeracion'->>'patron') BETWEEN 8 AND 40
        AND (contenido->'numeracion'->>'ancho') ~ '^[4-9]$')
);

CREATE TABLE vec_seleccion.solicitud(
 solicitud_ref text PRIMARY KEY CHECK (solicitud_ref ~ '^sol_[A-Za-z0-9_-]{22,64}$'),
 persona_ref text NOT NULL CHECK (persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 convocatoria_ref text NOT NULL,
 creada_en timestamptz(6) NOT NULL,
 UNIQUE (persona_ref, convocatoria_ref),
 UNIQUE (solicitud_ref, persona_ref, convocatoria_ref)
);

-- Cada versión del borrador. Los datos personales llegan cifrados (sobre
-- ligado a persona, convocatoria y versión); el documento de identidad solo
-- como huella con clave y parcial para listados.
CREATE TABLE vec_seleccion.solicitud_version(
 solicitud_ref text NOT NULL,
 persona_ref text NOT NULL,
 convocatoria_ref text NOT NULL,
 version integer NOT NULL CHECK (version >= 1),
 convocatoria_version integer NOT NULL,
 turno text NOT NULL CHECK (turno ~ '^[a-z0-9_]{1,64}$'),
 datos_clave_ref text NOT NULL CHECK (octet_length(datos_clave_ref) BETWEEN 1 AND 128 AND datos_clave_ref = btrim(datos_clave_ref)),
 datos_nonce bytea NOT NULL CHECK (octet_length(datos_nonce) BETWEEN 12 AND 32),
 datos_cifrado bytea NOT NULL CHECK (octet_length(datos_cifrado) BETWEEN 17 AND 16384),
 documento_huella text NOT NULL CHECK (documento_huella ~ '^[0-9a-f]{64}$'),
 documento_parcial text NOT NULL CHECK (documento_parcial ~ '^[*A-Z0-9]{4,24}$'),
 requisitos jsonb NOT NULL CHECK (jsonb_typeof(requisitos) = 'array' AND jsonb_array_length(requisitos) <= 64),
 meritos jsonb NOT NULL CHECK (jsonb_typeof(meritos) = 'array' AND jsonb_array_length(meritos) <= 200),
 puntuacion_micropuntos bigint NOT NULL CHECK (puntuacion_micropuntos BETWEEN 0 AND 1000000000000),
 clave_idempotencia text NOT NULL CHECK (clave_idempotencia ~ '^[A-Za-z0-9._:-]{8,128}$'),
 huella_material text NOT NULL CHECK (huella_material ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL UNIQUE CHECK (octet_length(decision_ref) BETWEEN 1 AND 256),
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY (solicitud_ref, version),
 FOREIGN KEY (solicitud_ref, persona_ref, convocatoria_ref) REFERENCES vec_seleccion.solicitud(solicitud_ref, persona_ref, convocatoria_ref),
 FOREIGN KEY (convocatoria_ref, convocatoria_version) REFERENCES vec_seleccion.convocatoria_publicada(convocatoria_ref, version)
);

-- Idempotencia semántica del guardado: la clave de la persona fija el
-- material. Repetirla con otro material es un conflicto.
CREATE TABLE vec_seleccion.operacion_borrador(
 persona_ref text NOT NULL,
 clave_idempotencia text NOT NULL,
 huella_material text NOT NULL CHECK (huella_material ~ '^[0-9a-f]{64}$'),
 solicitud_ref text NOT NULL,
 version integer NOT NULL,
 PRIMARY KEY (persona_ref, clave_idempotencia),
 FOREIGN KEY (solicitud_ref, version) REFERENCES vec_seleccion.solicitud_version(solicitud_ref, version)
);

CREATE TABLE vec_seleccion.presentacion(
 presentacion_id bigint GENERATED ALWAYS AS IDENTITY UNIQUE,
 solicitud_ref text PRIMARY KEY,
 version integer NOT NULL,
 convocatoria_ref text NOT NULL,
 numero_justificante text NOT NULL UNIQUE CHECK (octet_length(numero_justificante) BETWEEN 6 AND 64),
 anio integer NOT NULL CHECK (anio BETWEEN 2000 AND 2999),
 secuencia bigint NOT NULL CHECK (secuencia >= 1),
 presentada_en timestamptz(6) NOT NULL,
 recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:seleccion-presentacion:[0-9a-f]{64}$'),
 clave_idempotencia text NOT NULL CHECK (clave_idempotencia ~ '^[A-Za-z0-9._:-]{8,128}$'),
 huella_material text NOT NULL CHECK (huella_material ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL UNIQUE,
 UNIQUE (anio, secuencia),
 FOREIGN KEY (solicitud_ref, version) REFERENCES vec_seleccion.solicitud_version(solicitud_ref, version)
);

CREATE TABLE vec_seleccion.historia_solicitud(
 historia_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 solicitud_ref text NOT NULL REFERENCES vec_seleccion.solicitud(solicitud_ref),
 tipo text NOT NULL CHECK (tipo IN ('borrador_guardado', 'presentada')),
 version integer NOT NULL,
 decision_ref text NOT NULL,
 en timestamptz(6) NOT NULL
);

-- Accesos de RRHH: cada listado y cada ficha, con la decisión consumida.
CREATE TABLE vec_seleccion.acceso_rrhh(
 acceso_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 tipo text NOT NULL CHECK (tipo IN ('listado', 'detalle')),
 convocatoria_ref text NOT NULL,
 solicitud_ref text REFERENCES vec_seleccion.solicitud(solicitud_ref),
 actor_persona_ref text NOT NULL CHECK (actor_persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 decision_ref text NOT NULL UNIQUE,
 resultados integer NOT NULL CHECK (resultados >= 0),
 en timestamptz(6) NOT NULL,
 CHECK ((tipo = 'detalle') = (solicitud_ref IS NOT NULL))
);

-- Outbox de solo adición con referencias opacas (sin datos personales).
CREATE TABLE vec_seleccion.outbox(
 outbox_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 evento_ref text NOT NULL UNIQUE CHECK (evento_ref ~ '^evento:seleccion:[0-9a-f]{64}$'),
 tipo text NOT NULL CHECK (tipo = 'seleccion.solicitud.presentada'),
 carga jsonb NOT NULL CHECK (jsonb_typeof(carga) = 'object'),
 creado_en timestamptz(6) NOT NULL
);

DO $tablas$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['convocatoria_publicada','solicitud','solicitud_version','operacion_borrador','presentacion','historia_solicitud','acceso_rrhh','outbox'] LOOP
  EXECUTE format('ALTER TABLE vec_seleccion.%I ENABLE ROW LEVEL SECURITY', t);
  EXECUTE format('ALTER TABLE vec_seleccion.%I FORCE ROW LEVEL SECURITY', t);
  EXECUTE format('CREATE POLICY %I ON vec_seleccion.%I TO vec_seleccion_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
   t || '_solo_propietario', t, 'vec_seleccion_propietario', 'vec_seleccion_propietario');
  EXECUTE format('REVOKE ALL ON vec_seleccion.%I FROM PUBLIC', t);
  EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE OR DELETE ON vec_seleccion.%I FOR EACH ROW EXECUTE FUNCTION vec_seleccion.rechazar_mutacion()', t || '_inmutable', t);
  EXECUTE format('CREATE TRIGGER %I BEFORE TRUNCATE ON vec_seleccion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_seleccion.rechazar_mutacion()', t || '_sin_truncado', t);
 END LOOP;
END $tablas$;

-- Versión vigente (mayor) de una convocatoria.
CREATE FUNCTION vec_seleccion.convocatoria_vigente_interna_v1(p_convocatoria_ref text)
RETURNS vec_seleccion.convocatoria_publicada LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT c.* FROM vec_seleccion.convocatoria_publicada c
  WHERE c.convocatoria_ref = p_convocatoria_ref ORDER BY c.version DESC LIMIT 1
$f$;

-- Publicación al arrancar. Mismo contenido que la vigente: la reutiliza
-- (repetir el arranque no crea versiones). Contenido distinto: versión nueva.
-- publicada_en es el instante fijo del catálogo, no el reloj.
CREATE FUNCTION vec_seleccion.publicar_convocatoria_v1(
 p_convocatoria_ref text, p_titulo text, p_abre_en timestamptz, p_cierra_en timestamptz,
 p_contenido jsonb, p_huella text, p_publicada_en timestamptz)
RETURNS TABLE(version integer, nueva boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' SET lock_timeout = '2s' AS $f$
DECLARE v vec_seleccion.convocatoria_publicada;
BEGIN
 IF current_user <> 'vec_seleccion_propietario' OR p_convocatoria_ref IS NULL OR p_huella IS NULL OR p_huella !~ '^[0-9a-f]{64}$'
    OR p_publicada_en IS NULL OR p_abre_en IS NULL OR p_cierra_en IS NULL OR p_contenido IS NULL THEN
  RAISE EXCEPTION 'publicación de convocatoria inválida' USING ERRCODE = '22023';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_seleccion:convocatoria:' || p_convocatoria_ref, 0));
 v := vec_seleccion.convocatoria_vigente_interna_v1(p_convocatoria_ref);
 IF v.convocatoria_ref IS NOT NULL AND v.huella_sha256 = p_huella THEN
  IF v.titulo IS DISTINCT FROM p_titulo OR v.abre_en IS DISTINCT FROM p_abre_en OR v.cierra_en IS DISTINCT FROM p_cierra_en
     OR v.contenido IS DISTINCT FROM p_contenido OR v.publicada_en IS DISTINCT FROM p_publicada_en THEN
   RAISE EXCEPTION 'la huella de la convocatoria no corresponde a su contenido' USING ERRCODE = '22023';
  END IF;
  RETURN QUERY SELECT v.version, false;
  RETURN;
 END IF;
 INSERT INTO vec_seleccion.convocatoria_publicada(convocatoria_ref, version, huella_sha256, titulo, abre_en, cierra_en, contenido, publicada_en, registrada_en)
 VALUES (p_convocatoria_ref, coalesce(v.version, 0) + 1, p_huella, p_titulo, p_abre_en, p_cierra_en, p_contenido, p_publicada_en, clock_timestamp());
 RETURN QUERY SELECT coalesce(v.version, 0) + 1, true;
END $f$;

-- Convocatorias vigentes con su plazo evaluado con el reloj de la base.
-- Información pública de las bases: no lleva datos de ninguna solicitud.
CREATE FUNCTION vec_seleccion.convocatorias_publicadas_v1()
RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' AS $f$
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'convocatoria_ref', c.convocatoria_ref, 'version', c.version, 'titulo', c.titulo,
   'abre_en', to_char(c.abre_en, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), 'cierra_en', to_char(c.cierra_en, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'abierta', clock_timestamp() >= c.abre_en AND clock_timestamp() <= c.cierra_en,
   'huella_sha256', c.huella_sha256, 'contenido', c.contenido) ORDER BY c.convocatoria_ref), '[]'::jsonb)
 FROM (SELECT DISTINCT ON (x.convocatoria_ref) x.* FROM vec_seleccion.convocatoria_publicada x ORDER BY x.convocatoria_ref, x.version DESC) c
$f$;

-- Consumo de la decisión V3 de la persona (AD3-89) sobre su recurso propio.
-- La persona del contexto atestado debe ser p_persona_ref.
CREATE FUNCTION vec_seleccion.consumir_solicitud_propia_interna_v1(
 p_persona_ref text, p_operacion text,
 p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric,
 p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE c jsonb; d jsonb; x jsonb; r record; v_recurso text := 'mis-solicitudes:' || p_persona_ref;
BEGIN
 IF current_user <> 'vec_seleccion_propietario' OR p_persona_ref IS NULL OR p_persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$' THEN
  RAISE EXCEPTION 'solicitud propia inválida' USING ERRCODE = '22023';
 END IF;
 BEGIN c := convert_from(p_capacidad, 'UTF8')::jsonb; d := convert_from(p_decision, 'UTF8')::jsonb; x := convert_from(p_contexto, 'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de la solicitud propia inválido' USING ERRCODE = '22023'; END;
 IF c->>'operacion' IS DISTINCT FROM p_operacion OR d->>'accion' IS DISTINCT FROM p_operacion
    OR c->>'efecto_ref' IS DISTINCT FROM v_recurso OR d->>'recurso_ref' IS DISTINCT FROM v_recurso
    OR x->>'persona_ref' IS DISTINCT FROM p_persona_ref THEN
  RAISE EXCEPTION 'solicitud propia denegada' USING ERRCODE = '42501';
 END IF;
 SELECT * INTO STRICT r FROM vec_autorizacion_atestada_v3.registrar_y_consumir_solicitud_propia_seleccion_v3_atestada(
  p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 IF r.consumo_nuevo IS NOT TRUE OR r.efecto_ref IS DISTINCT FROM v_recurso OR r.decision_ref IS NULL THEN
  RAISE EXCEPTION 'solicitud propia denegada' USING ERRCODE = '42501';
 END IF;
 RETURN r.decision_ref;
END $f$;

-- Consumo de la decisión V3 de RRHH (AD3-90). Devuelve la decisión y la
-- persona que consulta, tomada del contexto atestado.
CREATE FUNCTION vec_seleccion.consumir_consulta_rrhh_interna_v1(
 p_operacion text, p_recurso text,
 p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric,
 p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(decision_ref text, actor_persona_ref text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE c jsonb; d jsonb; x jsonb; r record;
BEGIN
 IF current_user <> 'vec_seleccion_propietario' THEN RAISE EXCEPTION 'consulta inválida' USING ERRCODE = '22023'; END IF;
 BEGIN c := convert_from(p_capacidad, 'UTF8')::jsonb; d := convert_from(p_decision, 'UTF8')::jsonb; x := convert_from(p_contexto, 'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de la consulta inválido' USING ERRCODE = '22023'; END;
 IF c->>'operacion' IS DISTINCT FROM p_operacion OR d->>'accion' IS DISTINCT FROM p_operacion
    OR c->>'efecto_ref' IS DISTINCT FROM p_recurso OR d->>'recurso_ref' IS DISTINCT FROM p_recurso
    OR coalesce(x->>'persona_ref', '') !~ '^per_[A-Za-z0-9_-]{22,128}$' THEN
  RAISE EXCEPTION 'consulta de solicitudes denegada' USING ERRCODE = '42501';
 END IF;
 SELECT * INTO STRICT r FROM vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_solicitudes_seleccion_v3_atestada(
  p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 IF r.consumo_nuevo IS NOT TRUE OR r.efecto_ref IS DISTINCT FROM p_recurso OR r.decision_ref IS NULL THEN
  RAISE EXCEPTION 'consulta de solicitudes denegada' USING ERRCODE = '42501';
 END IF;
 RETURN QUERY SELECT r.decision_ref, x->>'persona_ref';
END $f$;

-- Guarda una versión nueva del borrador propio.
CREATE FUNCTION vec_seleccion.guardar_borrador_propio_v1(
 p_persona_ref text, p_convocatoria_ref text, p_convocatoria_version integer, p_version_esperada integer,
 p_clave text, p_huella_material text, p_solicitud_ref_nueva text, p_turno text,
 p_datos_clave_ref text, p_datos_nonce bytea, p_datos_cifrado bytea, p_documento_huella text, p_documento_parcial text,
 p_requisitos jsonb, p_meritos jsonb, p_puntuacion_micropuntos bigint,
 p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric,
 p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, solicitud_ref text, version integer, puntuacion_micropuntos bigint)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' SET lock_timeout = '2s' AS $f$
DECLARE v_decision text; v_previa vec_seleccion.operacion_borrador; v_conv vec_seleccion.convocatoria_publicada;
 v_solicitud vec_seleccion.solicitud; v_ultima integer; v_ahora timestamptz := clock_timestamp(); v_ref text; v_version integer; v_punt bigint;
BEGIN
 IF current_user <> 'vec_seleccion_propietario'
    OR p_convocatoria_ref IS NULL OR p_convocatoria_version IS NULL OR p_version_esperada IS NULL OR p_version_esperada < 0
    OR p_clave IS NULL OR p_clave !~ '^[A-Za-z0-9._:-]{8,128}$' OR p_huella_material IS NULL OR p_huella_material !~ '^[0-9a-f]{64}$'
    OR p_solicitud_ref_nueva IS NULL OR p_solicitud_ref_nueva !~ '^sol_[A-Za-z0-9_-]{22,64}$'
    OR p_puntuacion_micropuntos IS NULL OR p_puntuacion_micropuntos < 0 THEN
  RAISE EXCEPTION 'borrador de solicitud inválido' USING ERRCODE = '22023';
 END IF;
 -- Como el resto de consumidores, la decisión viva se consume antes de
 -- resolver la repetición: un reintento exige una autorización nueva.
 v_decision := vec_seleccion.consumir_solicitud_propia_interna_v1(p_persona_ref, 'seleccion.solicitudes_propias.guardar_borrador',
  p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_seleccion:persona:' || p_persona_ref, 0));
 SELECT * INTO v_previa FROM vec_seleccion.operacion_borrador o WHERE o.persona_ref = p_persona_ref AND o.clave_idempotencia = p_clave;
 IF FOUND THEN
  IF v_previa.huella_material <> p_huella_material THEN
   RAISE EXCEPTION 'clave reutilizada con otro borrador' USING ERRCODE = 'VSL02';
  END IF;
  RETURN QUERY SELECT true, v.solicitud_ref, v.version, v.puntuacion_micropuntos FROM vec_seleccion.solicitud_version v
   WHERE v.solicitud_ref = v_previa.solicitud_ref AND v.version = v_previa.version;
  RETURN;
 END IF;
 v_conv := vec_seleccion.convocatoria_vigente_interna_v1(p_convocatoria_ref);
 IF v_conv.convocatoria_ref IS NULL THEN RAISE EXCEPTION 'convocatoria no disponible' USING ERRCODE = 'VSL07'; END IF;
 IF v_conv.version <> p_convocatoria_version THEN RAISE EXCEPTION 'convocatoria actualizada' USING ERRCODE = 'VSL06'; END IF;
 IF v_ahora < v_conv.abre_en OR v_ahora > v_conv.cierra_en THEN RAISE EXCEPTION 'fuera de plazo' USING ERRCODE = 'VSL01'; END IF;
 SELECT * INTO v_solicitud FROM vec_seleccion.solicitud s WHERE s.persona_ref = p_persona_ref AND s.convocatoria_ref = p_convocatoria_ref;
 IF FOUND THEN
  IF EXISTS (SELECT 1 FROM vec_seleccion.presentacion p WHERE p.solicitud_ref = v_solicitud.solicitud_ref) THEN
   RAISE EXCEPTION 'solicitud ya presentada' USING ERRCODE = 'VSL04';
  END IF;
  SELECT max(v.version) INTO v_ultima FROM vec_seleccion.solicitud_version v WHERE v.solicitud_ref = v_solicitud.solicitud_ref;
  IF p_version_esperada <> v_ultima THEN RAISE EXCEPTION 'versión obsoleta' USING ERRCODE = 'VSL03'; END IF;
  v_ref := v_solicitud.solicitud_ref;
 ELSE
  IF p_version_esperada <> 0 THEN RAISE EXCEPTION 'versión obsoleta' USING ERRCODE = 'VSL03'; END IF;
  v_ref := p_solicitud_ref_nueva; v_ultima := 0;
  INSERT INTO vec_seleccion.solicitud(solicitud_ref, persona_ref, convocatoria_ref, creada_en) VALUES (v_ref, p_persona_ref, p_convocatoria_ref, v_ahora);
 END IF;
 v_version := v_ultima + 1;
 INSERT INTO vec_seleccion.solicitud_version(solicitud_ref, persona_ref, convocatoria_ref, version, convocatoria_version, turno,
   datos_clave_ref, datos_nonce, datos_cifrado, documento_huella, documento_parcial, requisitos, meritos, puntuacion_micropuntos,
   clave_idempotencia, huella_material, decision_ref, registrada_en)
 VALUES (v_ref, p_persona_ref, p_convocatoria_ref, v_version, v_conv.version, p_turno,
   p_datos_clave_ref, p_datos_nonce, p_datos_cifrado, p_documento_huella, p_documento_parcial, p_requisitos, p_meritos, p_puntuacion_micropuntos,
   p_clave, p_huella_material, v_decision, v_ahora);
 INSERT INTO vec_seleccion.operacion_borrador(persona_ref, clave_idempotencia, huella_material, solicitud_ref, version)
 VALUES (p_persona_ref, p_clave, p_huella_material, v_ref, v_version);
 INSERT INTO vec_seleccion.historia_solicitud(solicitud_ref, tipo, version, decision_ref, en) VALUES (v_ref, 'borrador_guardado', v_version, v_decision, v_ahora);
 RETURN QUERY SELECT false, v_ref, v_version, p_puntuacion_micropuntos;
END $f$;

-- Presenta la versión que la persona vio. Devuelve el justificante interno.
CREATE FUNCTION vec_seleccion.presentar_solicitud_propia_v1(
 p_persona_ref text, p_solicitud_ref text, p_version_esperada integer, p_clave text, p_huella_material text, p_recibo_ref text,
 p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric,
 p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, solicitud_ref text, numero_justificante text, presentada_en timestamptz, recibo_ref text, puntuacion_micropuntos bigint)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' SET lock_timeout = '2s' AS $f$
DECLARE v_decision text; v_solicitud vec_seleccion.solicitud; v_previa vec_seleccion.presentacion; v_version vec_seleccion.solicitud_version;
 v_conv vec_seleccion.convocatoria_publicada; v_ahora timestamptz := clock_timestamp(); v_anio integer; v_secuencia bigint; v_numero text;
 v_bloqueante text;
BEGIN
 IF current_user <> 'vec_seleccion_propietario'
    OR p_solicitud_ref IS NULL OR p_solicitud_ref !~ '^sol_[A-Za-z0-9_-]{22,64}$' OR p_version_esperada IS NULL OR p_version_esperada < 1
    OR p_clave IS NULL OR p_clave !~ '^[A-Za-z0-9._:-]{8,128}$' OR p_huella_material IS NULL OR p_huella_material !~ '^[0-9a-f]{64}$'
    OR p_recibo_ref IS NULL OR p_recibo_ref !~ '^recibo:seleccion-presentacion:[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'presentación inválida' USING ERRCODE = '22023';
 END IF;
 v_decision := vec_seleccion.consumir_solicitud_propia_interna_v1(p_persona_ref, 'seleccion.solicitudes_propias.presentar',
  p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_seleccion:persona:' || p_persona_ref, 0));
 -- Una solicitud ajena se trata como inexistente.
 SELECT * INTO v_solicitud FROM vec_seleccion.solicitud s WHERE s.solicitud_ref = p_solicitud_ref AND s.persona_ref = p_persona_ref;
 IF NOT FOUND THEN RAISE EXCEPTION 'solicitud no encontrada' USING ERRCODE = 'VSL08'; END IF;
 SELECT * INTO v_previa FROM vec_seleccion.presentacion p WHERE p.solicitud_ref = p_solicitud_ref;
 IF FOUND THEN
  IF v_previa.clave_idempotencia <> p_clave THEN RAISE EXCEPTION 'solicitud ya presentada' USING ERRCODE = 'VSL04'; END IF;
  IF v_previa.huella_material <> p_huella_material OR v_previa.version <> p_version_esperada THEN
   RAISE EXCEPTION 'clave reutilizada con otra presentación' USING ERRCODE = 'VSL02';
  END IF;
  RETURN QUERY SELECT true, v_previa.solicitud_ref, v_previa.numero_justificante, v_previa.presentada_en, v_previa.recibo_ref, v.puntuacion_micropuntos
   FROM vec_seleccion.solicitud_version v WHERE v.solicitud_ref = v_previa.solicitud_ref AND v.version = v_previa.version;
  RETURN;
 END IF;
 SELECT * INTO v_version FROM vec_seleccion.solicitud_version v WHERE v.solicitud_ref = p_solicitud_ref ORDER BY v.version DESC LIMIT 1;
 IF v_version.version <> p_version_esperada THEN RAISE EXCEPTION 'versión obsoleta' USING ERRCODE = 'VSL03'; END IF;
 v_conv := vec_seleccion.convocatoria_vigente_interna_v1(v_solicitud.convocatoria_ref);
 IF v_conv.convocatoria_ref IS NULL THEN RAISE EXCEPTION 'convocatoria no disponible' USING ERRCODE = 'VSL07'; END IF;
 IF v_ahora < v_conv.abre_en OR v_ahora > v_conv.cierra_en THEN RAISE EXCEPTION 'fuera de plazo' USING ERRCODE = 'VSL01'; END IF;
 IF v_conv.version <> v_version.convocatoria_version THEN RAISE EXCEPTION 'convocatoria actualizada' USING ERRCODE = 'VSL06'; END IF;
 -- Solo «no_cumple» en un requisito obligatorio que impide presentar
 -- detiene la presentación; «pendiente» no excluye.
 SELECT r->>'clave' INTO v_bloqueante FROM jsonb_array_elements(v_conv.contenido->'requisitos') r
  WHERE (r->>'obligatorio')::boolean AND coalesce((r->>'impide_presentar')::boolean, false)
    AND EXISTS (SELECT 1 FROM jsonb_array_elements(v_version.requisitos) q WHERE q->>'clave' = r->>'clave' AND q->>'estado' = 'no_cumple')
  LIMIT 1;
 IF v_bloqueante IS NOT NULL THEN RAISE EXCEPTION 'requisito no cumplido' USING ERRCODE = 'VSL05'; END IF;
 v_anio := extract(year FROM v_ahora AT TIME ZONE 'Europe/Madrid')::integer;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_seleccion:justificante:' || v_anio, 0));
 SELECT coalesce(max(p.secuencia), 0) + 1 INTO v_secuencia FROM vec_seleccion.presentacion p WHERE p.anio = v_anio;
 v_numero := replace(replace(v_conv.contenido->'numeracion'->>'patron', '{anio}', v_anio::text), '{numero}',
   lpad(v_secuencia::text, (v_conv.contenido->'numeracion'->>'ancho')::integer, '0'));
 INSERT INTO vec_seleccion.presentacion(solicitud_ref, version, convocatoria_ref, numero_justificante, anio, secuencia, presentada_en,
   recibo_ref, clave_idempotencia, huella_material, decision_ref)
 VALUES (p_solicitud_ref, v_version.version, v_solicitud.convocatoria_ref, v_numero, v_anio, v_secuencia, v_ahora,
   p_recibo_ref, p_clave, p_huella_material, v_decision);
 INSERT INTO vec_seleccion.historia_solicitud(solicitud_ref, tipo, version, decision_ref, en) VALUES (p_solicitud_ref, 'presentada', v_version.version, v_decision, v_ahora);
 INSERT INTO vec_seleccion.outbox(evento_ref, tipo, carga, creado_en)
 VALUES ('evento:seleccion:' || encode(sha256(convert_to('presentada' || chr(31) || p_solicitud_ref, 'UTF8')), 'hex'), 'seleccion.solicitud.presentada',
   jsonb_build_object('solicitud_ref', p_solicitud_ref, 'convocatoria_ref', v_solicitud.convocatoria_ref, 'convocatoria_version', v_conv.version,
     'version', v_version.version, 'numero_justificante', v_numero, 'recibo_ref', p_recibo_ref,
     'presentada_en', to_char(v_ahora, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')), v_ahora);
 RETURN QUERY SELECT false, p_solicitud_ref, v_numero, v_ahora, p_recibo_ref, v_version.puntuacion_micropuntos;
END $f$;

-- Solicitudes propias: la última versión de cada una y su presentación.
CREATE FUNCTION vec_seleccion.listar_solicitudes_propias_v1(
 p_persona_ref text,
 p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric,
 p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' SET lock_timeout = '2s' AS $f$
BEGIN
 PERFORM vec_seleccion.consumir_solicitud_propia_interna_v1(p_persona_ref, 'seleccion.solicitudes_propias.consultar',
  p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 RETURN (SELECT coalesce(jsonb_agg(jsonb_build_object(
   'solicitud_ref', s.solicitud_ref, 'convocatoria_ref', s.convocatoria_ref, 'convocatoria_titulo', c.titulo,
   'estado', CASE WHEN p.solicitud_ref IS NULL THEN 'borrador' ELSE 'presentada' END, 'version', v.version,
   'presentada_en', CASE WHEN p.solicitud_ref IS NULL THEN NULL ELSE to_char(p.presentada_en, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END,
   'numero_justificante', p.numero_justificante, 'puntuacion_micropuntos', v.puntuacion_micropuntos) ORDER BY s.creada_en, s.solicitud_ref), '[]'::jsonb)
  FROM vec_seleccion.solicitud s
  JOIN LATERAL (SELECT x.* FROM vec_seleccion.solicitud_version x WHERE x.solicitud_ref = s.solicitud_ref ORDER BY x.version DESC LIMIT 1) v ON true
  JOIN vec_seleccion.convocatoria_publicada c ON c.convocatoria_ref = v.convocatoria_ref AND c.version = v.convocatoria_version
  LEFT JOIN vec_seleccion.presentacion p ON p.solicitud_ref = s.solicitud_ref
  WHERE s.persona_ref = p_persona_ref);
END $f$;

-- Última versión de la solicitud propia en una convocatoria (con su sobre,
-- que solo la aplicación descifra); NULL si no hay.
CREATE FUNCTION vec_seleccion.leer_borrador_propio_v1(
 p_persona_ref text, p_convocatoria_ref text,
 p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric,
 p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' SET lock_timeout = '2s' AS $f$
BEGIN
 PERFORM vec_seleccion.consumir_solicitud_propia_interna_v1(p_persona_ref, 'seleccion.solicitudes_propias.consultar',
  p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 RETURN (SELECT jsonb_build_object(
   'solicitud_ref', v.solicitud_ref, 'persona_ref', v.persona_ref, 'convocatoria_ref', v.convocatoria_ref, 'convocatoria_version', v.convocatoria_version,
   'version', v.version, 'estado', CASE WHEN p.solicitud_ref IS NULL THEN 'borrador' ELSE 'presentada' END, 'turno', v.turno,
   'datos_clave_ref', v.datos_clave_ref, 'datos_nonce', encode(v.datos_nonce, 'base64'), 'datos_cifrado', encode(v.datos_cifrado, 'base64'),
   'requisitos', v.requisitos, 'meritos', v.meritos, 'puntuacion_micropuntos', v.puntuacion_micropuntos,
   'actualizada_en', to_char(v.registrada_en, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))
  FROM vec_seleccion.solicitud s
  JOIN LATERAL (SELECT x.* FROM vec_seleccion.solicitud_version x WHERE x.solicitud_ref = s.solicitud_ref ORDER BY x.version DESC LIMIT 1) v ON true
  LEFT JOIN vec_seleccion.presentacion p ON p.solicitud_ref = s.solicitud_ref
  WHERE s.persona_ref = p_persona_ref AND s.convocatoria_ref = p_convocatoria_ref);
END $f$;

-- Listado RRHH de las solicitudes presentadas en una convocatoria, paginado
-- por el orden de presentación. Queda auditado con la decisión consumida.
CREATE FUNCTION vec_seleccion.listar_solicitudes_convocatoria_v1(
 p_convocatoria_ref text, p_cursor bigint, p_limite integer,
 p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric,
 p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' SET lock_timeout = '2s' AS $f$
DECLARE r record; v_filas jsonb; v_ahora timestamptz := clock_timestamp();
BEGIN
 IF current_user <> 'vec_seleccion_propietario' OR p_convocatoria_ref IS NULL OR p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 100
    OR p_cursor IS NULL OR p_cursor < 0 THEN
  RAISE EXCEPTION 'consulta de solicitudes inválida' USING ERRCODE = '22023';
 END IF;
 SELECT * INTO STRICT r FROM vec_seleccion.consumir_consulta_rrhh_interna_v1('seleccion.solicitudes.consultar',
  'solicitudes-convocatoria:' || encode(sha256(convert_to(p_convocatoria_ref, 'UTF8')), 'hex'),
  p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'presentacion_id', q.presentacion_id, 'solicitud_ref', q.solicitud_ref, 'persona_ref', q.persona_ref, 'convocatoria_ref', q.convocatoria_ref,
   'version', q.version, 'numero_justificante', q.numero_justificante, 'turno', q.turno,
   'datos_clave_ref', q.datos_clave_ref, 'datos_nonce', encode(q.datos_nonce, 'base64'), 'datos_cifrado', encode(q.datos_cifrado, 'base64'),
   'documento_parcial', q.documento_parcial, 'presentada_en', to_char(q.presentada_en, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'puntuacion_micropuntos', q.puntuacion_micropuntos) ORDER BY q.presentacion_id), '[]'::jsonb)
 INTO v_filas
 FROM (SELECT p.presentacion_id, p.solicitud_ref, v.persona_ref, v.convocatoria_ref, v.version, p.numero_justificante, v.turno, v.datos_clave_ref,
         v.datos_nonce, v.datos_cifrado, v.documento_parcial, p.presentada_en, v.puntuacion_micropuntos
        FROM vec_seleccion.presentacion p
        JOIN vec_seleccion.solicitud_version v ON v.solicitud_ref = p.solicitud_ref AND v.version = p.version
       WHERE p.convocatoria_ref = p_convocatoria_ref AND p.presentacion_id > p_cursor
       ORDER BY p.presentacion_id LIMIT p_limite) q;
 INSERT INTO vec_seleccion.acceso_rrhh(tipo, convocatoria_ref, solicitud_ref, actor_persona_ref, decision_ref, resultados, en)
 VALUES ('listado', p_convocatoria_ref, NULL, r.actor_persona_ref, r.decision_ref, jsonb_array_length(v_filas), v_ahora);
 RETURN v_filas;
END $f$;

-- Ficha completa de una solicitud presentada para RRHH, con su historia y la
-- versión de la convocatoria con la que se presentó. Acceso auditado.
CREATE FUNCTION vec_seleccion.leer_detalle_solicitud_v1(
 p_solicitud_ref text,
 p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric,
 p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' SET lock_timeout = '2s' AS $f$
DECLARE r record; v_ficha jsonb; v_ahora timestamptz := clock_timestamp(); v_convocatoria text;
BEGIN
 IF current_user <> 'vec_seleccion_propietario' OR p_solicitud_ref IS NULL OR p_solicitud_ref !~ '^sol_[A-Za-z0-9_-]{22,64}$' THEN
  RAISE EXCEPTION 'consulta de solicitud inválida' USING ERRCODE = '22023';
 END IF;
 SELECT * INTO STRICT r FROM vec_seleccion.consumir_consulta_rrhh_interna_v1('seleccion.solicitudes.consultar_detalle',
  'solicitud-seleccion:' || p_solicitud_ref,
  p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 SELECT p.convocatoria_ref, jsonb_build_object(
   'solicitud_ref', p.solicitud_ref, 'persona_ref', v.persona_ref, 'convocatoria_ref', p.convocatoria_ref, 'convocatoria_titulo', c.titulo,
   'convocatoria_version', c.version, 'convocatoria_contenido', c.contenido,
   'version', v.version, 'numero_justificante', p.numero_justificante, 'estado', 'presentada',
   'presentada_en', to_char(p.presentada_en, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), 'turno', v.turno,
   'datos_clave_ref', v.datos_clave_ref, 'datos_nonce', encode(v.datos_nonce, 'base64'), 'datos_cifrado', encode(v.datos_cifrado, 'base64'),
   'documento_parcial', v.documento_parcial, 'requisitos', v.requisitos, 'meritos', v.meritos, 'puntuacion_micropuntos', v.puntuacion_micropuntos,
   'historia', (SELECT coalesce(jsonb_agg(jsonb_build_object('tipo', h.tipo, 'version', h.version, 'en', to_char(h.en, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) ORDER BY h.historia_id), '[]'::jsonb)
                FROM vec_seleccion.historia_solicitud h WHERE h.solicitud_ref = p.solicitud_ref))
 INTO v_convocatoria, v_ficha
 FROM vec_seleccion.presentacion p
 JOIN vec_seleccion.solicitud_version v ON v.solicitud_ref = p.solicitud_ref AND v.version = p.version
 JOIN vec_seleccion.convocatoria_publicada c ON c.convocatoria_ref = v.convocatoria_ref AND c.version = v.convocatoria_version
 WHERE p.solicitud_ref = p_solicitud_ref;
 IF v_ficha IS NULL THEN RAISE EXCEPTION 'solicitud no encontrada' USING ERRCODE = 'VSL08'; END IF;
 INSERT INTO vec_seleccion.acceso_rrhh(tipo, convocatoria_ref, solicitud_ref, actor_persona_ref, decision_ref, resultados, en)
 VALUES ('detalle', v_convocatoria, p_solicitud_ref, r.actor_persona_ref, r.decision_ref, 1, v_ahora);
 RETURN v_ficha;
END $f$;

DO $acl$
DECLARE f regprocedure; publicas regprocedure[] := ARRAY[
  'vec_seleccion.publicar_convocatoria_v1(text,text,timestamptz,timestamptz,jsonb,text,timestamptz)'::regprocedure,
  'vec_seleccion.convocatorias_publicadas_v1()'::regprocedure,
  'vec_seleccion.guardar_borrador_propio_v1(text,text,integer,integer,text,text,text,text,text,bytea,bytea,text,text,jsonb,jsonb,bigint,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_seleccion.presentar_solicitud_propia_v1(text,text,integer,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_seleccion.listar_solicitudes_propias_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_seleccion.leer_borrador_propio_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_seleccion.listar_solicitudes_convocatoria_v1(text,bigint,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_seleccion.leer_detalle_solicitud_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure];
 internas regprocedure[] := ARRAY[
  'vec_seleccion.rechazar_mutacion()'::regprocedure,
  'vec_seleccion.convocatoria_vigente_interna_v1(text)'::regprocedure,
  'vec_seleccion.consumir_solicitud_propia_interna_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_seleccion.consumir_consulta_rrhh_interna_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure];
BEGIN
 FOREACH f IN ARRAY publicas || internas LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC', f::text);
 END LOOP;
 FOREACH f IN ARRAY publicas LOOP
  EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_seleccion_ejecutor', f::text);
 END LOOP;
 FOREACH f IN ARRAY publicas || internas LOOP
  IF EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl, acldefault('f', p.proowner))) a
              WHERE p.oid = f AND a.grantee <> p.proowner
                AND (a.grantee <> 'vec_seleccion_ejecutor'::regrole OR a.is_grantable OR NOT f = ANY (publicas)))
     OR (SELECT proowner FROM pg_proc WHERE oid = f) <> 'vec_seleccion_propietario'::regrole
     OR (f <> 'vec_seleccion.rechazar_mutacion()'::regprocedure AND NOT (SELECT prosecdef FROM pg_proc WHERE oid = f)) THEN
   RAISE EXCEPTION 'ACL de Selección abierta' USING ERRCODE = '55000';
  END IF;
 END LOOP;
 IF EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace CROSS JOIN LATERAL aclexplode(coalesce(c.relacl, acldefault('r', c.relowner))) a
             WHERE n.nspname = 'vec_seleccion' AND a.grantee <> c.relowner) THEN
  RAISE EXCEPTION 'tablas de Selección abiertas' USING ERRCODE = '55000';
 END IF;
END $acl$;
COMMIT;
