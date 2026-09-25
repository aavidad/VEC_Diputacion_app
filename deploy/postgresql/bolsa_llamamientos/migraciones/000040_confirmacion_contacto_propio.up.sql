\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000040', 0));

-- Duda 45 (continuación de 000035): la persona confirma desde «Mi bolsa» que
-- el contacto vigente de su participación sigue siendo el suyo. Consume la
-- acción propia AD3-86 'bolsa.participaciones_propias.confirmar_contacto'
-- sobre 'mi-bolsa:<candidato>' y coteja al candidato con el único vínculo
-- candidato activo del contexto atestado. Confirma exactamente la versión
-- que la persona vio: si RRHH registró otra entre medias, se rechaza.
--
-- La confirmación no crea una versión nueva ni toca el sobre cifrado: queda
-- en una tabla propia, de solo adición, ligada a la versión. Una versión
-- confirmada deja de contar como «contacto de origen CONVOCA sin confirmar».
DO $precondicion$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.datos_contacto_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.origen_datos_contacto_participacion') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.listar_participaciones_candidato_v1(text)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(text)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.exigir_consumo_candidato_v1(text[],text)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_propio_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'dependencias de la confirmación de contacto ausentes (000016, 000029, 000035 y AD3-86)' USING ERRCODE='55000';
 END IF;
 IF to_regclass('vec_bolsa_llamamientos.confirmacion_contacto_participacion') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion 000040 ya aplicada' USING ERRCODE='55000';
 END IF;
END $precondicion$;

CREATE TABLE vec_bolsa_llamamientos.confirmacion_contacto_participacion(
 participacion_ref text NOT NULL,
 version bigint NOT NULL,
 candidato_ref text NOT NULL CHECK (candidato_ref ~ '^can_[A-Za-z0-9_-]{22,128}$'),
 bolsa_ref text NOT NULL CHECK (octet_length(bolsa_ref) BETWEEN 1 AND 512 AND bolsa_ref = btrim(bolsa_ref)),
 clave_idempotencia text NOT NULL CHECK (octet_length(clave_idempotencia) BETWEEN 8 AND 256 AND clave_idempotencia = btrim(clave_idempotencia)),
 recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:confirmacion-contacto:[0-9a-f]{64}$'),
 decision_ref text NOT NULL UNIQUE CHECK (octet_length(decision_ref) BETWEEN 1 AND 256),
 confirmada_en timestamptz(6) NOT NULL,
 PRIMARY KEY (participacion_ref, version),
 UNIQUE (participacion_ref, clave_idempotencia),
 FOREIGN KEY (participacion_ref, version) REFERENCES vec_bolsa_llamamientos.datos_contacto_participacion(participacion_ref, version)
);
ALTER TABLE vec_bolsa_llamamientos.confirmacion_contacto_participacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.confirmacion_contacto_participacion FORCE ROW LEVEL SECURITY;
CREATE POLICY confirmacion_contacto_participacion_solo_propietario ON vec_bolsa_llamamientos.confirmacion_contacto_participacion
 TO vec_bolsa_llamamientos_propietario
 USING (current_user = 'vec_bolsa_llamamientos_propietario') WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.confirmacion_contacto_participacion FROM PUBLIC;
CREATE TRIGGER confirmacion_contacto_participacion_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.confirmacion_contacto_participacion
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Participación vigente del candidato en la bolsa, con el mismo criterio que
-- la lectura del contacto (000029). Sin vínculo: 42501.
CREATE FUNCTION vec_bolsa_llamamientos.participacion_contacto_candidato_v1(p_candidato_ref text, p_bolsa_ref text)
RETURNS text LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE v text;
BEGIN
 SELECT p.participacion_ref INTO v FROM vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(p_candidato_ref) p
  WHERE p.bolsa_ref = p_bolsa_ref;
 IF v IS NULL THEN RAISE EXCEPTION 'la bolsa no es del candidato' USING ERRCODE='42501'; END IF;
 RETURN v;
END $f$;

-- Núcleo sin material: solo lo alcanza la función pública tras consumir la
-- decisión (y las pruebas como propietario).
CREATE FUNCTION vec_bolsa_llamamientos.registrar_confirmacion_contacto_interna_v1(
 p_candidato_ref text, p_bolsa_ref text, p_participacion_ref text, p_version bigint, p_clave text, p_recibo_ref text,
 p_confirmada_en timestamptz, p_decision_ref text)
RETURNS TABLE(reutilizada boolean, recibo_ref text, version bigint, confirmada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE v_vigente bigint;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_version IS NULL OR p_version < 1 OR p_confirmada_en IS NULL OR p_decision_ref IS NULL
    OR p_recibo_ref IS NULL OR p_recibo_ref !~ '^recibo:confirmacion-contacto:[0-9a-f]{64}$'
    OR p_clave IS NULL OR p_clave <> btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256 THEN
  RAISE EXCEPTION 'confirmación de contacto inválida' USING ERRCODE='22023';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:datos_contacto:' || p_participacion_ref, 0));
 SELECT max(x.version) INTO v_vigente FROM vec_bolsa_llamamientos.datos_contacto_participacion x WHERE x.participacion_ref = p_participacion_ref;
 IF v_vigente IS NULL THEN RAISE EXCEPTION 'sin contacto que confirmar' USING ERRCODE='VBC03'; END IF;
 -- Se confirma la versión que la persona vio: otra más reciente la invalida.
 IF v_vigente <> p_version THEN RAISE EXCEPTION 'el contacto ha cambiado' USING ERRCODE='VBC02'; END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.confirmacion_contacto_participacion x
             WHERE x.participacion_ref = p_participacion_ref AND x.version = p_version) THEN
  RAISE EXCEPTION 'contacto ya confirmado' USING ERRCODE='VBC04';
 END IF;
 INSERT INTO vec_bolsa_llamamientos.confirmacion_contacto_participacion(participacion_ref, version, candidato_ref, bolsa_ref, clave_idempotencia,
   recibo_ref, decision_ref, confirmada_en)
 VALUES (p_participacion_ref, p_version, p_candidato_ref, p_bolsa_ref, p_clave, p_recibo_ref, p_decision_ref, p_confirmada_en);
 RETURN QUERY SELECT false, p_recibo_ref, p_version, p_confirmada_en;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.confirmar_contacto_propio_v1(
 p_candidato_ref text, p_bolsa_ref text, p_version bigint, p_clave text, p_recibo_ref text, p_confirmada_en timestamptz,
 p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric,
 p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, recibo_ref text, version bigint, confirmada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' SET lock_timeout = '2s' AS $f$
DECLARE c jsonb; d jsonb; x jsonb; n integer; v_participacion text; v_consumo record;
 v_previa vec_bolsa_llamamientos.confirmacion_contacto_participacion%ROWTYPE;
 v_accion constant text := 'bolsa.participaciones_propias.confirmar_contacto';
 v_ahora timestamptz := clock_timestamp();
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_bolsa_ref IS NULL OR octet_length(p_bolsa_ref) NOT BETWEEN 1 AND 512 OR p_bolsa_ref <> btrim(p_bolsa_ref)
    OR p_version IS NULL OR p_version < 1
    OR p_clave IS NULL OR p_clave <> btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR p_recibo_ref IS NULL OR p_recibo_ref !~ '^recibo:confirmacion-contacto:[0-9a-f]{64}$'
    OR p_confirmada_en IS NULL OR p_confirmada_en > v_ahora + interval '1 minute' OR p_confirmada_en < v_ahora - interval '5 minutes' THEN
  RAISE EXCEPTION 'confirmación de contacto inválida' USING ERRCODE='22023';
 END IF;
 BEGIN c := convert_from(p_capacidad,'UTF8')::jsonb; d := convert_from(p_decision,'UTF8')::jsonb; x := convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de confirmación inválido' USING ERRCODE='22023'; END;
 IF c->>'efecto_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref OR d->>'recurso_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
    OR c->>'operacion' IS DISTINCT FROM v_accion OR d->>'accion' IS DISTINCT FROM v_accion
    OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array' THEN
  RAISE EXCEPTION 'confirmación de contacto denegada' USING ERRCODE='42501';
 END IF;
 SELECT count(*) INTO n FROM jsonb_array_elements(x->'vinculos') e WHERE e->>'tipo'='candidato' AND e->>'estado'='activo';
 IF n <> 1 OR NOT EXISTS (SELECT 1 FROM jsonb_array_elements(x->'vinculos') e
                           WHERE e->>'tipo'='candidato' AND e->>'estado'='activo' AND e->>'referencia'=p_candidato_ref) THEN
  RAISE EXCEPTION 'confirmación de contacto denegada' USING ERRCODE='42501';
 END IF;
 v_participacion := vec_bolsa_llamamientos.participacion_contacto_candidato_v1(p_candidato_ref, p_bolsa_ref);
 -- Como B2 y el portal (000030), la decisión viva se consume antes de
 -- resolver el replay: un reintento no devuelve el recibo sin una
 -- autorización nueva y verificada.
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_propio_bolsa_v3_atestada(
  p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref THEN
  RAISE EXCEPTION 'confirmación de contacto denegada' USING ERRCODE='42501';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:datos_contacto:' || v_participacion, 0));
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.confirmacion_contacto_participacion cp
  WHERE cp.participacion_ref = v_participacion AND cp.clave_idempotencia = p_clave;
 IF FOUND THEN
  IF v_previa.version <> p_version OR v_previa.candidato_ref <> p_candidato_ref THEN
   RAISE EXCEPTION 'clave reutilizada con otra confirmación' USING ERRCODE='VBC01';
  END IF;
  RETURN QUERY SELECT true, v_previa.recibo_ref, v_previa.version, v_previa.confirmada_en;
  RETURN;
 END IF;
 RETURN QUERY SELECT * FROM vec_bolsa_llamamientos.registrar_confirmacion_contacto_interna_v1(
  p_candidato_ref, p_bolsa_ref, v_participacion, p_version, p_clave, p_recibo_ref, p_confirmada_en, v_consumo.decision_ref);
END $f$;

-- Estado del contacto por bolsa del candidato (participación vigente, el
-- mismo criterio que confirmar): versión vigente, marca de origen CONVOCA (si
-- la hay) y confirmación de esa versión. Nunca el claro. Solo responde dentro
-- de la transacción que ya consumió la consulta propia (Mi bolsa) del mismo
-- candidato: sin esa marca, 42501.
CREATE FUNCTION vec_bolsa_llamamientos.leer_contacto_candidato_v1(p_candidato_ref text, p_corte timestamptz)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' AS $f$
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_consumo_candidato_v1(ARRAY['consulta'], p_candidato_ref);
 RETURN (
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'bolsa', p.bolsa_ref, 'version', v.version,
   'origen', CASE WHEN o.participacion_ref IS NULL THEN NULL ELSE jsonb_build_object('origen', o.origen,
      'vigente_hasta', to_char(o.vigente_hasta,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), 'ultimo_dia', to_char(o.ultimo_dia,'YYYY-MM-DD')) END,
   'confirmada_en', CASE WHEN cf.participacion_ref IS NULL THEN NULL ELSE to_char(cf.confirmada_en,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END)
   ORDER BY p.bolsa_ref), '[]'::jsonb)
 FROM vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(p_candidato_ref) p
 JOIN LATERAL (SELECT max(dc.version) AS version FROM vec_bolsa_llamamientos.datos_contacto_participacion dc
                WHERE dc.participacion_ref = p.participacion_ref AND dc.registrada_en <= p_corte) v ON v.version IS NOT NULL
 LEFT JOIN vec_bolsa_llamamientos.origen_datos_contacto_participacion o ON o.participacion_ref = p.participacion_ref AND o.version = v.version
 LEFT JOIN vec_bolsa_llamamientos.confirmacion_contacto_participacion cf
   ON cf.participacion_ref = p.participacion_ref AND cf.version = v.version AND cf.confirmada_en <= p_corte
 WHERE p_corte IS NOT NULL);
END $f$;

-- Confirmación de una versión concreta, para RRHH y para los avisos del
-- llamamiento: una versión confirmada por la persona cuenta como propia.
CREATE FUNCTION vec_bolsa_llamamientos.leer_confirmacion_contacto_participacion_v1(p_participacion_ref text, p_version bigint)
RETURNS timestamptz LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT cf.confirmada_en FROM vec_bolsa_llamamientos.confirmacion_contacto_participacion cf
  WHERE cf.participacion_ref = p_participacion_ref AND cf.version = p_version
$f$;

DO $acl$
DECLARE f regprocedure; publicas regprocedure[] := ARRAY[
  'vec_bolsa_llamamientos.confirmar_contacto_propio_v1(text,text,bigint,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_bolsa_llamamientos.leer_contacto_candidato_v1(text,timestamptz)'::regprocedure,
  'vec_bolsa_llamamientos.leer_confirmacion_contacto_participacion_v1(text,bigint)'::regprocedure];
 internas regprocedure[] := ARRAY[
  'vec_bolsa_llamamientos.participacion_contacto_candidato_v1(text,text)'::regprocedure,
  'vec_bolsa_llamamientos.registrar_confirmacion_contacto_interna_v1(text,text,text,bigint,text,text,timestamptz,text)'::regprocedure];
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
     OR (SELECT proowner FROM pg_proc WHERE oid = f) <> 'vec_bolsa_llamamientos_propietario'::regrole
     OR NOT (SELECT prosecdef FROM pg_proc WHERE oid = f) THEN
   RAISE EXCEPTION 'ACL de la confirmación de contacto abierta' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $acl$;
COMMIT;
