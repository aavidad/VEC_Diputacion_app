\set ON_ERROR_STOP on
-- Bolsa 000046, RRHH 2.05. El retorno del titular acredita la causa del cese
-- del sustituto; no produce una segunda restriccion de disponibilidad. La
-- restriccion y su regla versionada pertenecen a Bolsa 000045. CT 000130
-- acredita cada origen mediante una consulta estrecha, sin acceso a sus tablas.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000046', 0));
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;

DO $precondicion$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.reincorporacion_titular_ct') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.restriccion_cese_bolsa') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.vinculo_candidato') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.verificar_reincorporacion_publicada_bolsa_v1(text,text,bigint)') IS NULL
    OR NOT has_function_privilege('vec_contratacion_temporal.verificar_reincorporacion_publicada_bolsa_v1(text,text,bigint)', 'EXECUTE') THEN
  RAISE EXCEPTION 'Bolsa 000046: faltan 000045, CT 000130 o autorizacion V3, o ya esta instalada' USING ERRCODE='55000';
 END IF;
END $precondicion$;

CREATE TABLE vec_bolsa_llamamientos.reincorporacion_titular_ct (
 evento_ref text PRIMARY KEY CHECK (octet_length(evento_ref) BETWEEN 1 AND 512 AND evento_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]*$'),
 origen_huella_sha256 text NOT NULL CHECK (origen_huella_sha256 ~ '^[a-f0-9]{64}$'),
 origen_posicion bigint NOT NULL CHECK (origen_posicion >= 0),
 expediente_ref text NOT NULL CHECK (octet_length(expediente_ref) BETWEEN 1 AND 512 AND expediente_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]*$'),
 relacion_ref text NOT NULL CHECK (octet_length(relacion_ref) BETWEEN 1 AND 512 AND relacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]*$'),
 fecha_efectiva date NOT NULL CHECK (isfinite(fecha_efectiva)),
 recibo_ct_ref text NOT NULL UNIQUE CHECK (octet_length(recibo_ct_ref) BETWEEN 1 AND 512 AND recibo_ct_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]*$'),
 -- Referencia del acto CT115, que es origen_ref de 000045. La PK de 000045
 -- deriva otra referencia de publicación; no son intercambiables.
 cese_evento_ref text NOT NULL CHECK (octet_length(cese_evento_ref) BETWEEN 1 AND 512 AND cese_evento_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]*$'),
 cese_recibo_ref text NOT NULL CHECK (octet_length(cese_recibo_ref) BETWEEN 1 AND 512 AND cese_recibo_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]*$'),
 recibida_en timestamptz(6) NOT NULL CHECK (isfinite(recibida_en)),
 recibida_por text NOT NULL DEFAULT session_user CHECK (octet_length(recibida_por) BETWEEN 1 AND 128)
);
CREATE INDEX reincorporacion_titular_ct_cese ON vec_bolsa_llamamientos.reincorporacion_titular_ct(cese_evento_ref);
CREATE INDEX reincorporacion_titular_ct_cursor ON vec_bolsa_llamamientos.reincorporacion_titular_ct(origen_posicion DESC, evento_ref DESC);
ALTER TABLE vec_bolsa_llamamientos.reincorporacion_titular_ct ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.reincorporacion_titular_ct FORCE ROW LEVEL SECURITY;
CREATE POLICY reincorporacion_titular_ct_solo_propietario ON vec_bolsa_llamamientos.reincorporacion_titular_ct
 TO vec_bolsa_llamamientos_propietario
 USING (current_user = 'vec_bolsa_llamamientos_propietario')
 WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.reincorporacion_titular_ct FROM PUBLIC;
CREATE TRIGGER reincorporacion_titular_ct_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.reincorporacion_titular_ct
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- El relevo solo proporciona el cursor del outbox CT. Los campos se toman de
-- la respuesta acreditada por CT para esa huella y posicion exactas. El cese
-- puede llegar a Bolsa despues: la fila queda pendiente y se enlaza al leer.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_reincorporacion_titular_ct_v1(
 p_origen_ref text, p_huella_sha256 text, p_origen_posicion bigint)
RETURNS TABLE(reutilizada boolean, estado text, cese_evento_ref text, disponible_desde date)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '10s' AS $f$
DECLARE v_origen record; v_previa vec_bolsa_llamamientos.reincorporacion_titular_ct;
 v_cese record; v_estado text; v_disponible date;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_origen_ref IS NULL
    OR octet_length(p_origen_ref) NOT BETWEEN 1 AND 512 OR p_origen_ref !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]*$'
    OR p_huella_sha256 IS NULL OR p_huella_sha256 !~ '^[a-f0-9]{64}$'
    OR p_origen_posicion IS NULL OR p_origen_posicion < 0 THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='origen de reincorporacion invalido';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:reincorporacion-titular:' || p_origen_ref, 0));
 SELECT * INTO v_origen FROM vec_contratacion_temporal.verificar_reincorporacion_publicada_bolsa_v1(
   p_origen_ref, p_huella_sha256, p_origen_posicion);
 IF NOT FOUND OR v_origen.evento_ref IS DISTINCT FROM p_origen_ref
    OR v_origen.expediente_ref IS NULL OR v_origen.relacion_ref IS NULL
    OR v_origen.fecha_efectiva IS NULL OR NOT isfinite(v_origen.fecha_efectiva)
    OR v_origen.recibo_ct_ref IS NULL OR v_origen.cese_evento_ref IS NULL
    OR v_origen.cese_recibo_ref IS NULL THEN
  RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='origen de reincorporacion no acreditado';
 END IF;
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.reincorporacion_titular_ct r WHERE r.evento_ref = p_origen_ref;
 IF FOUND THEN
  IF v_previa.origen_huella_sha256 IS DISTINCT FROM p_huella_sha256
     OR v_previa.origen_posicion IS DISTINCT FROM p_origen_posicion
     OR v_previa.expediente_ref IS DISTINCT FROM v_origen.expediente_ref
     OR v_previa.relacion_ref IS DISTINCT FROM v_origen.relacion_ref
     OR v_previa.fecha_efectiva IS DISTINCT FROM v_origen.fecha_efectiva
     OR v_previa.recibo_ct_ref IS DISTINCT FROM v_origen.recibo_ct_ref
     OR v_previa.cese_evento_ref IS DISTINCT FROM v_origen.cese_evento_ref
     OR v_previa.cese_recibo_ref IS DISTINCT FROM v_origen.cese_recibo_ref THEN
   RAISE EXCEPTION USING ERRCODE='23505', MESSAGE='reincorporacion divergente';
  END IF;
 ELSE
  INSERT INTO vec_bolsa_llamamientos.reincorporacion_titular_ct(evento_ref, origen_huella_sha256, origen_posicion,
    expediente_ref, relacion_ref, fecha_efectiva, recibo_ct_ref, cese_evento_ref, cese_recibo_ref, recibida_en)
  VALUES (p_origen_ref, p_huella_sha256, p_origen_posicion, v_origen.expediente_ref, v_origen.relacion_ref,
    v_origen.fecha_efectiva, v_origen.recibo_ct_ref, v_origen.cese_evento_ref, v_origen.cese_recibo_ref,
    date_trunc('microseconds', clock_timestamp()));
 END IF;
 SELECT c.relacion_ref, c.recibo_ct_ref, c.fecha_efecto, c.disponible_desde
   INTO v_cese FROM vec_bolsa_llamamientos.restriccion_cese_bolsa c
  WHERE c.origen_ref = v_origen.cese_evento_ref;
 IF NOT FOUND THEN
  v_estado := 'pendiente_cese';
 ELSIF v_cese.relacion_ref IS DISTINCT FROM v_origen.relacion_ref
    OR v_cese.recibo_ct_ref IS DISTINCT FROM v_origen.cese_recibo_ref
    OR v_cese.fecha_efecto IS DISTINCT FROM v_origen.fecha_efectiva THEN
  v_estado := 'cese_incompatible';
 ELSE
  v_estado := 'cese_aplicado';
  v_disponible := v_cese.disponible_desde;
 END IF;
 RETURN QUERY SELECT v_previa.evento_ref IS NOT NULL, v_estado, v_origen.cese_evento_ref, v_disponible;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.cursor_reincorporaciones_titular_ct_v1()
RETURNS TABLE(origen_posicion bigint, origen_ref text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT r.origen_posicion, r.evento_ref FROM vec_bolsa_llamamientos.reincorporacion_titular_ct r
 ORDER BY r.origen_posicion DESC, r.evento_ref DESC LIMIT 1
$f$;

-- Lectura de la ficha Bolsa: el candidato se resuelve por el vinculo propio
-- de la participacion y la restriccion global de 000045. El consumo V3 se
-- produce incluso en una recuperacion. Ningun dato de otra persona sale.
CREATE FUNCTION vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v1(
 p_participacion_ref text, p_actor text, p_capacidad bytea, p_decision bytea, p_motivo_autorizacion bytea,
 p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea,
 p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(evento_ref text, expediente_ref text, relacion_ref text, fecha_efectiva date,
 recibo_ct_ref text, cese_evento_ref text, estado text, disponible_desde date,
 regla_version bigint, regla_huella_sha256 text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' AS $f$
DECLARE v_consumo record; v_decision jsonb;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_participacion_ref IS NULL OR p_actor IS NULL THEN
  RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='consulta de reincorporaciones no autorizada';
 END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(
  p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN v_decision := convert_from(p_decision, 'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='consulta de reincorporaciones no autorizada'; END;
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
  RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='consulta de reincorporaciones no autorizada';
 END IF;
 RETURN QUERY
 SELECT r.evento_ref, r.expediente_ref, r.relacion_ref, r.fecha_efectiva, r.recibo_ct_ref, r.cese_evento_ref,
  'cese_aplicado'::text, c.disponible_desde, c.politica_version, p.catalogo_sha256
 FROM vec_bolsa_llamamientos.vinculo_candidato v
 JOIN vec_bolsa_llamamientos.restriccion_cese_bolsa c ON c.candidato_ref = v.candidato_ref
 JOIN vec_bolsa_llamamientos.reincorporacion_titular_ct r ON r.cese_evento_ref = c.origen_ref
  AND r.relacion_ref = c.relacion_ref AND r.cese_recibo_ref = c.recibo_ct_ref AND r.fecha_efectiva = c.fecha_efecto
 LEFT JOIN vec_bolsa_llamamientos.politica_cese_bolsa p ON p.version = c.politica_version
 WHERE v.participacion_ref = p_participacion_ref
 ORDER BY r.fecha_efectiva DESC, r.evento_ref DESC LIMIT 100;
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_reincorporacion_titular_ct_v1(text,text,bigint) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.cursor_reincorporaciones_titular_ct_v1() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_reincorporacion_titular_ct_v1(text,text,bigint) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.cursor_reincorporaciones_titular_ct_v1() TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
COMMENT ON TABLE vec_bolsa_llamamientos.reincorporacion_titular_ct IS
 'Retornos de titular CT 000130 acreditados; no alteran disponibilidad, que deriva exclusivamente del cese Bolsa 000045.';
COMMIT;
