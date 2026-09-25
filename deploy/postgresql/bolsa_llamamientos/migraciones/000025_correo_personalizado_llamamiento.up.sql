\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000025',0));

-- B7 personalizado (petición de RRHH p. 3): el cuerpo del correo es una
-- plantilla con marcadores que el servidor sustituye por persona.
-- 1) La reserva admite cualquier versión de plantilla con el formato
--    bolsa-llamamiento-vN; qué versiones existen y si interpretan marcadores lo
--    gobierna el catálogo de la aplicación. La huella del recibo no cambia: sigue
--    cubriendo bolsa, participaciones y configuración (plantilla incluida).
-- 2) Cada destinatario deja la huella SHA-256 del asunto y del cuerpo exactos
--    que se le envían, ligada a la reserva por el token de finalización, de
--    modo que cada correo personalizado es verificable junto al recibo.
DO $pre$ BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.llamamiento_emitido') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.reservar_llamamiento_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR strpos(pg_get_functiondef('vec_bolsa_llamamientos.reservar_llamamiento_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),$c$p_configuracion->>'plantilla_version'<>'bolsa-llamamiento-v1'$c$)=0
    OR to_regclass('vec_bolsa_llamamientos.cuerpo_llamamiento_contacto') IS NOT NULL
 THEN RAISE EXCEPTION 'estado incompatible con B7 personalizado (000025)' USING ERRCODE='55000'; END IF;
END $pre$;
DO $ajuste_b7$
DECLARE
 v_def text;
 v_ant text := $sql$p_configuracion->>'plantilla_version'<>'bolsa-llamamiento-v1'$sql$;
 v_nueva text := $sql$p_configuracion->>'plantilla_version' !~ '^bolsa-llamamiento-v[1-9][0-9]{0,3}$'$sql$;
BEGIN
 SELECT pg_get_functiondef('vec_bolsa_llamamientos.reservar_llamamiento_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) INTO STRICT v_def;
 IF strpos(v_def,v_ant)=0 OR strpos(substr(v_def,strpos(v_def,v_ant)+length(v_ant)),v_ant)<>0 THEN
  RAISE EXCEPTION 'reserva B7 incompatible con plantillas versionadas' USING ERRCODE='55000';
 END IF;
 EXECUTE replace(v_def,v_ant,v_nueva);
END $ajuste_b7$;

CREATE TABLE vec_bolsa_llamamientos.cuerpo_llamamiento_contacto(
 llamamiento_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.llamamiento_emitido(llamamiento_ref),
 ordinal integer NOT NULL CHECK(ordinal BETWEEN 1 AND 100),
 participacion_ref text NOT NULL CHECK(participacion_ref<>'' AND octet_length(participacion_ref)<=256),
 huella_asunto_sha256 text NOT NULL CHECK(huella_asunto_sha256 ~ '^[0-9a-f]{64}$'),
 huella_cuerpo_sha256 text NOT NULL CHECK(huella_cuerpo_sha256 ~ '^[0-9a-f]{64}$'),
 caracteres integer NOT NULL CHECK(caracteres BETWEEN 1 AND 20000),
 registrado_en timestamptz(6) NOT NULL,
 PRIMARY KEY(llamamiento_ref,ordinal),
 UNIQUE(llamamiento_ref,participacion_ref)
);
ALTER TABLE vec_bolsa_llamamientos.cuerpo_llamamiento_contacto ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.cuerpo_llamamiento_contacto FORCE ROW LEVEL SECURITY;
CREATE POLICY cuerpo_llamamiento_contacto_solo_propietario ON vec_bolsa_llamamientos.cuerpo_llamamiento_contacto TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.cuerpo_llamamiento_contacto FROM PUBLIC;
CREATE TRIGGER cuerpo_llamamiento_contacto_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.cuerpo_llamamiento_contacto FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE FUNCTION vec_bolsa_llamamientos.registrar_cuerpos_llamamiento_v1(p_bolsa text,p_clave text,p_actor text,p_token_finalizacion bytea,p_cuerpos jsonb) RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE previo record; v_existentes int; v_iguales int; BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$' OR p_cuerpos IS NULL OR jsonb_typeof(p_cuerpos)<>'array' OR jsonb_array_length(p_cuerpos) NOT BETWEEN 1 AND 100 THEN RAISE EXCEPTION 'huellas B7 inválidas' USING ERRCODE='22023'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:emision:'||p_bolsa||':'||p_clave,0));
 SELECT * INTO previo FROM vec_bolsa_llamamientos.llamamiento_emitido WHERE bolsa_ref=p_bolsa AND clave_idempotencia=p_clave;
 IF NOT FOUND OR previo.actor_ref<>p_actor OR octet_length(p_token_finalizacion) IS DISTINCT FROM 32 OR previo.huella_finalizacion IS DISTINCT FROM sha256(p_token_finalizacion) OR jsonb_array_length(p_cuerpos)<>jsonb_array_length(previo.participaciones)
    OR NOT(SELECT bool_and(jsonb_typeof(c.value)='object' AND (SELECT count(*)=4 FROM jsonb_object_keys(c.value)) AND c.value->>'participacion_ref' IS NOT DISTINCT FROM previo.participaciones->>(c.ordinality-1)::int AND coalesce(c.value->>'huella_asunto_sha256','') ~ '^[0-9a-f]{64}$' AND coalesce(c.value->>'huella_cuerpo_sha256','') ~ '^[0-9a-f]{64}$' AND jsonb_typeof(c.value->'caracteres')='number' AND (c.value->>'caracteres') ~ '^[1-9][0-9]{0,4}$' AND (c.value->>'caracteres')::int<=20000) FROM jsonb_array_elements(p_cuerpos) WITH ORDINALITY c(value,ordinality))
 THEN RAISE EXCEPTION 'huellas B7 no autorizadas' USING ERRCODE='42501'; END IF;
 SELECT count(*) INTO v_existentes FROM vec_bolsa_llamamientos.cuerpo_llamamiento_contacto WHERE llamamiento_ref=previo.llamamiento_ref;
 IF v_existentes>0 THEN
  SELECT count(*) INTO v_iguales FROM jsonb_array_elements(p_cuerpos) WITH ORDINALITY c(value,ordinality) JOIN vec_bolsa_llamamientos.cuerpo_llamamiento_contacto h ON h.llamamiento_ref=previo.llamamiento_ref AND h.ordinal=c.ordinality AND h.participacion_ref=c.value->>'participacion_ref' AND h.huella_asunto_sha256=c.value->>'huella_asunto_sha256' AND h.huella_cuerpo_sha256=c.value->>'huella_cuerpo_sha256' AND h.caracteres=(c.value->>'caracteres')::int;
  IF v_existentes<>jsonb_array_length(p_cuerpos) OR v_iguales<>v_existentes THEN RAISE EXCEPTION 'huellas B7 divergentes' USING ERRCODE='VBE01'; END IF;
  RETURN;
 END IF;
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contacto_participacion c WHERE c.llamamiento_ref=previo.llamamiento_ref) THEN RAISE EXCEPTION 'huellas B7 tras el envío' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_bolsa_llamamientos.cuerpo_llamamiento_contacto(llamamiento_ref,ordinal,participacion_ref,huella_asunto_sha256,huella_cuerpo_sha256,caracteres,registrado_en)
 SELECT previo.llamamiento_ref,c.ordinality,c.value->>'participacion_ref',c.value->>'huella_asunto_sha256',c.value->>'huella_cuerpo_sha256',(c.value->>'caracteres')::int,clock_timestamp() FROM jsonb_array_elements(p_cuerpos) WITH ORDINALITY c(value,ordinality);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_cuerpos_llamamiento_v1(text,text,text,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_cuerpos_llamamiento_v1(text,text,text,bytea,jsonb) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
