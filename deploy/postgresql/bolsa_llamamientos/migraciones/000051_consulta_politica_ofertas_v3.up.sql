\set ON_ERROR_STOP on
-- B51: lectura RRHH de la política B47 mediante permiso propio AD3-97.
-- El cálculo interno de ofertas conserva leer_politica_ofertas_v1; la ruta
-- HTTP usa exclusivamente esta función con material atestado y traza local.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000051',0));

DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_bolsa_llamamientos.leer_politica_ofertas_v1(text)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_politica_ofertas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_politica_ofertas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'B51: preimagen incompatible (B47 y AD3-97 requeridas)' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_bolsa_llamamientos.politica_ofertas_lectura_v3(
 decision_ref text PRIMARY KEY,
 auditoria_ref text NOT NULL,
 bolsa_ref text NOT NULL CHECK (bolsa_ref ~ '^bolsa:[A-Za-z0-9:_-]{1,250}$'),
 actor_ref text NOT NULL CHECK (actor_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 version_leida bigint NOT NULL CHECK (version_leida>=0),
 consultada_en timestamptz(6) NOT NULL
);
CREATE TRIGGER politica_ofertas_lectura_inmutable
 BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.politica_ofertas_lectura_v3
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

CREATE FUNCTION vec_bolsa_llamamientos.consultar_politica_ofertas_v2(
 p_bolsa text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE d jsonb; consumo record; resultado jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_bolsa IS NULL OR p_bolsa !~ '^bolsa:[A-Za-z0-9:_-]{1,250}$'
 THEN RAISE EXCEPTION 'B51: consulta inválida' USING ERRCODE='22023'; END IF;
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B51: decisión inválida' USING ERRCODE='42501'; END;
 -- La autorización se consume ANTES de leer incluso si no hay política.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_consulta_politica_ofertas_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM p_bolsa
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM d->>'contexto_recurso_huella_sha256'
    OR d->>'accion' IS DISTINCT FROM 'bolsa.politica_ofertas.consultar'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida'
    OR d->>'finalidad' IS DISTINCT FROM 'consultar_politica_ofertas_bolsa'
    OR d->>'recurso_ref' IS DISTINCT FROM p_bolsa
    OR coalesce(d->>'principal_id','') !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR d->'campos_permitidos' IS DISTINCT FROM '["politica_ofertas"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'B51: lectura no autorizada' USING ERRCODE='42501'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion c WHERE c.bolsa_ref=p_bolsa) THEN
  RAISE EXCEPTION 'B51: bolsa no constituida' USING ERRCODE='23503'; END IF;
 resultado:=vec_bolsa_llamamientos.leer_politica_ofertas_v1(p_bolsa);
 IF resultado IS NULL OR resultado->>'bolsa_ref' IS DISTINCT FROM p_bolsa
    OR coalesce(resultado->>'version','') !~ '^[0-9]{1,16}$'
 THEN RAISE EXCEPTION 'B51: proyección inválida' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_bolsa_llamamientos.politica_ofertas_lectura_v3(
  decision_ref,auditoria_ref,bolsa_ref,actor_ref,version_leida,consultada_en)
 VALUES(consumo.decision_ref,consumo.auditoria_ref,p_bolsa,d->>'principal_id',
  (resultado->>'version')::bigint,clock_timestamp());
 RETURN resultado;
END $f$;

REVOKE ALL ON TABLE vec_bolsa_llamamientos.politica_ofertas_lectura_v3 FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_politica_ofertas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_politica_ofertas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
