\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000028', 0));

-- B-OF: publicación de ofertas (Petición RRHH p. 3; Reglamento de bolsas,
-- art. 8.1). RRHH publica una oferta de una bolsa constituida; las personas de
-- esa bolsa manifiestan disposición dentro del plazo que fija la regla b10 del
-- catálogo (calculado fuera, con Calendarios, y guardado aquí con su
-- referencia); al vencer, VEC propone la mejor posición del orden vigente
-- entre quienes se ofrecieron y RRHH confirma, o se pasa a llamamiento
-- directo si nadie lo hizo.
--
-- La publicación y la resolución son dos modalidades del llamamiento del
-- art. 8.1: consumen la misma autorización atestada que la emisión B7
-- (`llamamiento.emitir.v1` sobre `bolsa_constituida`), sin consumidor nuevo.
-- La tabla de disposiciones solo la leen aquí la consulta y la resolución; su
-- escritura por la propia persona exige un consumidor de autorización del
-- candidato que todavía no existe y llegará en una migración posterior.
DO $precondicion$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.bolsa_constituida') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.constitucion') IS NULL THEN
  RAISE EXCEPTION 'dependencias B-OF ausentes' USING ERRCODE='55000';
 END IF;
 IF to_regclass('vec_bolsa_llamamientos.oferta_publicada') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion 000028 ya aplicada' USING ERRCODE='55000';
 END IF;
END $precondicion$;

CREATE TABLE vec_bolsa_llamamientos.oferta_publicada(
 oferta_ref text PRIMARY KEY CHECK (oferta_ref ~ '^oferta:[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:oferta:[0-9a-f]{64}$'),
 bolsa_ref text NOT NULL CHECK (octet_length(bolsa_ref) BETWEEN 1 AND 256 AND bolsa_ref = btrim(bolsa_ref)),
 actor_ref text NOT NULL CHECK (actor_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 clave_idempotencia text NOT NULL CHECK (octet_length(clave_idempotencia) BETWEEN 8 AND 256 AND clave_idempotencia = btrim(clave_idempotencia)),
 -- Datos visibles de la oferta: categoría, centro, fechas y descripción.
 datos jsonb NOT NULL CHECK (jsonb_typeof(datos) = 'object'),
 -- Plazo resuelto con la regla del catálogo: referencia y huella, unidad,
 -- cantidad, cómputo, último día y calendarios usados.
 plazo jsonb NOT NULL CHECK (jsonb_typeof(plazo) = 'object'),
 publicada_en timestamptz(6) NOT NULL,
 vence_antes_de timestamptz(6) NOT NULL,
 huella_comando_sha256 text NOT NULL CHECK (huella_comando_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL UNIQUE,
 UNIQUE (bolsa_ref, clave_idempotencia),
 CHECK (vence_antes_de > publicada_en AND vence_antes_de <= publicada_en + interval '120 days')
);

CREATE TABLE vec_bolsa_llamamientos.disposicion_oferta(
 oferta_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.oferta_publicada(oferta_ref),
 participacion_ref text NOT NULL CHECK (octet_length(participacion_ref) BETWEEN 1 AND 256 AND participacion_ref = btrim(participacion_ref)),
 manifestada_en timestamptz(6) NOT NULL,
 clave_idempotencia text NOT NULL CHECK (octet_length(clave_idempotencia) BETWEEN 8 AND 256 AND clave_idempotencia = btrim(clave_idempotencia)),
 recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:disposicion:[0-9a-f]{64}$'),
 PRIMARY KEY (oferta_ref, participacion_ref)
);

CREATE TABLE vec_bolsa_llamamientos.resolucion_oferta(
 oferta_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.oferta_publicada(oferta_ref),
 recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:resolucion-oferta:[0-9a-f]{64}$'),
 tipo text NOT NULL CHECK (tipo IN ('adjudicada','llamamiento_directo')),
 participacion_ref text,
 orden_vigente bigint,
 disposiciones_total integer NOT NULL CHECK (disposiciones_total >= 0),
 actor_ref text NOT NULL CHECK (actor_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 clave_idempotencia text NOT NULL CHECK (octet_length(clave_idempotencia) BETWEEN 8 AND 256 AND clave_idempotencia = btrim(clave_idempotencia)),
 resuelta_en timestamptz(6) NOT NULL,
 decision_ref text NOT NULL UNIQUE,
 CHECK ((tipo = 'adjudicada' AND participacion_ref IS NOT NULL AND orden_vigente IS NOT NULL AND disposiciones_total >= 1)
     OR (tipo = 'llamamiento_directo' AND participacion_ref IS NULL AND orden_vigente IS NULL))
);

DO $acl$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['oferta_publicada','disposicion_oferta','resolucion_oferta'] LOOP
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I ENABLE ROW LEVEL SECURITY', t);
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I FORCE ROW LEVEL SECURITY', t);
  EXECUTE format('CREATE POLICY %I ON vec_bolsa_llamamientos.%I TO vec_bolsa_llamamientos_propietario USING (current_user = ''vec_bolsa_llamamientos_propietario'') WITH CHECK (current_user = ''vec_bolsa_llamamientos_propietario'')', t || '_solo_propietario', t);
  EXECUTE format('REVOKE ALL ON vec_bolsa_llamamientos.%I FROM PUBLIC', t);
  EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion()', t || '_inmutable', t);
 END LOOP;
END $acl$;

-- Proyección única de una oferta en un instante. El estado se deduce: la
-- resolución confirmada manda; sin ella, abierta hasta el vencimiento y
-- pendiente de resolución después. La propuesta usa el orden vigente B6 en
-- ese mismo instante: solo cuenta quien ocupa turno.
CREATE FUNCTION vec_bolsa_llamamientos.proyectar_oferta_v1(p_oferta_ref text, p_corte timestamptz)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE o record; r record; v_disp jsonb; v_propuesta jsonb; v_estado text; v_mejor record; v_total integer;
BEGIN
 SELECT * INTO o FROM vec_bolsa_llamamientos.oferta_publicada WHERE oferta_ref = p_oferta_ref;
 IF NOT FOUND OR p_corte IS NULL THEN RETURN NULL; END IF;
 SELECT * INTO r FROM vec_bolsa_llamamientos.resolucion_oferta WHERE oferta_ref = p_oferta_ref;
 SELECT coalesce(jsonb_agg(jsonb_build_object(
          'participacion_ref', d.participacion_ref,
          'manifestada_en', to_char(d.manifestada_en,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
          'orden_vigente', ov.orden_vigente,
          'situacion', ov.situacion)
        ORDER BY ov.orden_vigente NULLS LAST, d.manifestada_en, d.participacion_ref),'[]'::jsonb), count(*)
   INTO v_disp, v_total
   FROM vec_bolsa_llamamientos.disposicion_oferta d
   LEFT JOIN vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(o.bolsa_ref, p_corte) ov ON ov.participacion_ref = d.participacion_ref
  WHERE d.oferta_ref = p_oferta_ref AND d.manifestada_en <= p_corte;
 IF r.oferta_ref IS NOT NULL AND r.resuelta_en <= p_corte THEN
  v_estado := r.tipo;
 ELSIF p_corte < o.vence_antes_de THEN
  v_estado := 'abierta';
 ELSE
  v_estado := 'pendiente_resolucion';
  SELECT d.participacion_ref, ov.orden_vigente INTO v_mejor
    FROM vec_bolsa_llamamientos.disposicion_oferta d
    JOIN vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(o.bolsa_ref, p_corte) ov ON ov.participacion_ref = d.participacion_ref
   WHERE d.oferta_ref = p_oferta_ref AND d.manifestada_en < o.vence_antes_de AND ov.orden_vigente IS NOT NULL
   ORDER BY ov.orden_vigente, d.participacion_ref LIMIT 1;
  IF v_mejor.participacion_ref IS NULL THEN
   v_propuesta := jsonb_build_object('tipo','llamamiento_directo');
  ELSE
   v_propuesta := jsonb_build_object('tipo','adjudicar','participacion_ref',v_mejor.participacion_ref,'orden_vigente',v_mejor.orden_vigente);
  END IF;
 END IF;
 RETURN jsonb_build_object(
  'oferta_ref', o.oferta_ref, 'recibo_ref', o.recibo_ref, 'bolsa_ref', o.bolsa_ref,
  'datos', o.datos, 'plazo', o.plazo,
  'publicada_en', to_char(o.publicada_en,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'vence_antes_de', to_char(o.vence_antes_de,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'estado', v_estado, 'disposiciones', v_disp, 'disposiciones_total', v_total,
  'propuesta', v_propuesta,
  'resolucion', CASE WHEN r.oferta_ref IS NULL OR r.resuelta_en > p_corte THEN NULL ELSE jsonb_build_object(
     'recibo_ref', r.recibo_ref, 'tipo', r.tipo, 'participacion_ref', r.participacion_ref,
     'orden_vigente', r.orden_vigente, 'disposiciones_total', r.disposiciones_total,
     'resuelta_en', to_char(r.resuelta_en,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) END);
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.publicar_oferta_v1(
 p_oferta text, p_recibo text, p_bolsa text, p_actor text, p_clave text, p_datos jsonb, p_plazo jsonb,
 p_publicada timestamptz, p_vence timestamptz,
 p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric,
 p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(oferta jsonb, reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE previo record; consumo record; d jsonb; v_huella text; v_ahora timestamptz := clock_timestamp();
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR p_oferta IS NULL OR p_oferta !~ '^oferta:[0-9a-f]{64}$'
    OR p_recibo IS NULL OR p_recibo !~ '^recibo:oferta:[0-9a-f]{64}$'
    OR p_bolsa IS NULL OR octet_length(p_bolsa) NOT BETWEEN 1 AND 256 OR p_bolsa <> btrim(p_bolsa)
    OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR p_clave <> btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    -- Datos: cuatro textos obligatorios y fecha de fin opcional, sin claves ajenas.
    OR jsonb_typeof(p_datos) IS DISTINCT FROM 'object'
    OR NOT (p_datos ?& ARRAY['categoria','centro','fecha_inicio','descripcion'])
    OR EXISTS (SELECT 1 FROM jsonb_object_keys(p_datos) k WHERE k NOT IN ('categoria','centro','fecha_inicio','fecha_fin','descripcion'))
    OR EXISTS (SELECT 1 FROM jsonb_each(p_datos) e WHERE jsonb_typeof(e.value) <> 'string' OR octet_length(e.value#>>'{}') NOT BETWEEN 2 AND 2000 OR e.value#>>'{}' <> btrim(e.value#>>'{}'))
    OR p_datos->>'fecha_inicio' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
    OR (p_datos ? 'fecha_fin' AND (p_datos->>'fecha_fin' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' OR p_datos->>'fecha_fin' < p_datos->>'fecha_inicio'))
    -- Plazo: la regla del catálogo que lo fijó queda con su referencia exacta.
    OR jsonb_typeof(p_plazo) IS DISTINCT FROM 'object'
    OR NOT (p_plazo ?& ARRAY['regla_ref','huella_catalogo','unidad','cantidad','computo','ultimo_dia','ejemplo'])
    OR jsonb_typeof(p_plazo->'regla_ref') <> 'string' OR octet_length(p_plazo->>'regla_ref') NOT BETWEEN 3 AND 512
    OR p_plazo->>'huella_catalogo' !~ '^[0-9a-f]{64}$'
    OR jsonb_typeof(p_plazo->'cantidad') <> 'number' OR jsonb_typeof(p_plazo->'ejemplo') <> 'boolean'
    OR p_plazo->>'ultimo_dia' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
    OR p_publicada IS NULL OR p_vence IS NULL OR p_vence <= p_publicada OR p_vence > p_publicada + interval '120 days'
    OR p_publicada > v_ahora + interval '1 minute' OR p_publicada < v_ahora - interval '5 minutes' THEN
  RAISE EXCEPTION 'publicacion de oferta invalida' USING ERRCODE='22023';
 END IF;
 -- La decisión viva se consume antes de resolver el replay: un reintento no
 -- devuelve el recibo sin una autorización nueva y verificada.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(
  p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 BEGIN d := convert_from(p_decision, 'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'publicacion de oferta no autorizada' USING ERRCODE='42501'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_bolsa OR consumo.consumo_nuevo IS NOT TRUE
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_llamamientos_bolsa'
    OR d->>'recurso_ref' IS DISTINCT FROM p_bolsa
    OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida' THEN
  RAISE EXCEPTION 'publicacion de oferta no autorizada' USING ERRCODE='42501';
 END IF;
 v_huella := encode(sha256(convert_to(p_bolsa || chr(31) || p_datos::text, 'UTF8')), 'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:oferta:' || p_bolsa || ':' || p_clave, 0));
 SELECT * INTO previo FROM vec_bolsa_llamamientos.oferta_publicada WHERE bolsa_ref = p_bolsa AND clave_idempotencia = p_clave;
 IF FOUND THEN
  IF previo.actor_ref <> p_actor OR previo.huella_comando_sha256 <> v_huella THEN
   RAISE EXCEPTION 'clave de oferta reutilizada con otro comando' USING ERRCODE='VBO01';
  END IF;
  RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v1(previo.oferta_ref, v_ahora), true;
  RETURN;
 END IF;
 IF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.constitucion c
                  JOIN vec_bolsa_llamamientos.bolsa_constituida b ON b.bolsa_ref = c.bolsa_ref AND b.version = c.version_bolsa
                 WHERE c.bolsa_ref = p_bolsa AND b.estado = 'vigente' AND b.vigente_desde <= p_publicada
                   AND (b.vigente_hasta IS NULL OR p_publicada < b.vigente_hasta)) THEN
  RAISE EXCEPTION 'bolsa no vigente' USING ERRCODE='23503';
 END IF;
 INSERT INTO vec_bolsa_llamamientos.oferta_publicada(oferta_ref, recibo_ref, bolsa_ref, actor_ref, clave_idempotencia, datos, plazo, publicada_en, vence_antes_de, huella_comando_sha256, decision_ref)
 VALUES (p_oferta, p_recibo, p_bolsa, p_actor, p_clave, p_datos, p_plazo, p_publicada, p_vence, v_huella, consumo.decision_ref);
 RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v1(p_oferta, greatest(v_ahora, p_publicada)), false;
END $f$;

-- RRHH confirma la propuesta que VEC calcula en ese instante: debe coincidir
-- exactamente (misma persona o, sin disposiciones útiles, llamamiento
-- directo). Si el orden cambió entre la consulta y la confirmación, se
-- rechaza para que RRHH vea la propuesta nueva.
CREATE FUNCTION vec_bolsa_llamamientos.resolver_oferta_v1(
 p_oferta text, p_recibo text, p_bolsa text, p_participacion text, p_actor text, p_clave text,
 p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric,
 p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(oferta jsonb, reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE o record; previa record; consumo record; d jsonb; v_ahora timestamptz := clock_timestamp(); v_proyeccion jsonb; v_tipo text;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR p_oferta IS NULL OR p_oferta !~ '^oferta:[0-9a-f]{64}$'
    OR p_recibo IS NULL OR p_recibo !~ '^recibo:resolucion-oferta:[0-9a-f]{64}$'
    OR p_bolsa IS NULL OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR p_clave <> btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR (p_participacion IS NOT NULL AND (p_participacion <> btrim(p_participacion) OR octet_length(p_participacion) NOT BETWEEN 1 AND 256)) THEN
  RAISE EXCEPTION 'resolucion de oferta invalida' USING ERRCODE='22023';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:resolucion-oferta:' || p_oferta, 0));
 SELECT * INTO o FROM vec_bolsa_llamamientos.oferta_publicada WHERE oferta_ref = p_oferta;
 IF NOT FOUND OR o.bolsa_ref <> p_bolsa THEN
  RAISE EXCEPTION 'oferta inexistente en la bolsa' USING ERRCODE='23503';
 END IF;
 -- La decisión viva se consume antes de resolver el replay: un reintento no
 -- devuelve el recibo sin una autorización nueva y verificada.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(
  p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 BEGIN d := convert_from(p_decision, 'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'resolucion de oferta no autorizada' USING ERRCODE='42501'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_bolsa OR consumo.consumo_nuevo IS NOT TRUE
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_llamamientos_bolsa'
    OR d->>'recurso_ref' IS DISTINCT FROM p_bolsa
    OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida' THEN
  RAISE EXCEPTION 'resolucion de oferta no autorizada' USING ERRCODE='42501';
 END IF;
 SELECT * INTO previa FROM vec_bolsa_llamamientos.resolucion_oferta WHERE oferta_ref = p_oferta;
 IF FOUND THEN
  IF previa.clave_idempotencia <> p_clave OR previa.actor_ref <> p_actor OR previa.participacion_ref IS DISTINCT FROM p_participacion THEN
   RAISE EXCEPTION 'oferta ya resuelta' USING ERRCODE='VBO02';
  END IF;
  RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v1(p_oferta, v_ahora), true;
  RETURN;
 END IF;
 IF v_ahora < o.vence_antes_de THEN
  RAISE EXCEPTION 'plazo de disposicion abierto' USING ERRCODE='VBO03';
 END IF;
 v_proyeccion := vec_bolsa_llamamientos.proyectar_oferta_v1(p_oferta, v_ahora);
 IF v_proyeccion->'propuesta'->>'participacion_ref' IS DISTINCT FROM p_participacion THEN
  RAISE EXCEPTION 'la propuesta ha cambiado' USING ERRCODE='VBO04';
 END IF;
 v_tipo := CASE WHEN p_participacion IS NULL THEN 'llamamiento_directo' ELSE 'adjudicada' END;
 INSERT INTO vec_bolsa_llamamientos.resolucion_oferta(oferta_ref, recibo_ref, tipo, participacion_ref, orden_vigente, disposiciones_total, actor_ref, clave_idempotencia, resuelta_en, decision_ref)
 VALUES (p_oferta, p_recibo, v_tipo, p_participacion,
         CASE WHEN p_participacion IS NULL THEN NULL ELSE (v_proyeccion->'propuesta'->>'orden_vigente')::bigint END,
         (v_proyeccion->>'disposiciones_total')::integer, p_actor, p_clave, v_ahora, consumo.decision_ref);
 RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v1(p_oferta, v_ahora), false;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.listar_ofertas_bolsa_v1(p_bolsa text, p_corte timestamptz, p_limite integer)
RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
 SELECT coalesce(jsonb_agg(vec_bolsa_llamamientos.proyectar_oferta_v1(x.oferta_ref, p_corte) ORDER BY x.publicada_en DESC, x.oferta_ref), '[]'::jsonb)
   FROM (SELECT o.oferta_ref, o.publicada_en FROM vec_bolsa_llamamientos.oferta_publicada o
          WHERE o.bolsa_ref = p_bolsa AND o.publicada_en <= p_corte AND p_limite BETWEEN 1 AND 100
          ORDER BY o.publicada_en DESC, o.oferta_ref LIMIT greatest(least(p_limite, 100), 1)) x
$f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.proyectar_oferta_v1(text,timestamptz) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.publicar_oferta_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.resolver_oferta_v1(text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_ofertas_bolsa_v1(text,timestamptz,integer) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.publicar_oferta_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.resolver_oferta_v1(text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_ofertas_bolsa_v1(text,timestamptz,integer) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
