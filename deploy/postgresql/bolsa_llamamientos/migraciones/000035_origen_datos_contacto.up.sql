\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000035', 0));

-- Duda 45: los aspirantes importados de CONVOCA no tienen contacto propio en
-- VEC. El correo y los teléfonos que RRHH trae de CONVOCA se registran como una
-- versión más de B4 (000016, mismo sobre cifrado y misma concesión V3) y esa
-- versión queda marcada aquí como «contacto de origen CONVOCA», con la vigencia
-- que fija la regla b29 del catálogo de reglas (referencia y huella exactas).
-- La marca vale solo para su versión: cualquier versión posterior, registrada
-- por RRHH o confirmada por la persona, la sustituye. Nada se reescribe.
DO $precondicion$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.datos_contacto_participacion') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.origen_datos_contacto_participacion') IS NOT NULL
    OR to_regprocedure('vec_bolsa_llamamientos.leer_origen_datos_contacto_participacion_v1(text,bigint)') IS NOT NULL THEN
  RAISE EXCEPTION 'estado incompatible para el contacto de origen CONVOCA' USING ERRCODE='55000';
 END IF;
END $precondicion$;
CREATE TABLE vec_bolsa_llamamientos.origen_datos_contacto_participacion (
    participacion_ref text NOT NULL,
    version bigint NOT NULL,
    origen text NOT NULL CHECK (origen = 'convoca'),
    vigente_hasta timestamptz(6) NOT NULL,
    ultimo_dia date NOT NULL,
    regla_ref text NOT NULL CHECK (octet_length(regla_ref) BETWEEN 1 AND 512 AND regla_ref = btrim(regla_ref)),
    regla_huella_sha256 text NOT NULL CHECK (regla_huella_sha256 ~ '^[0-9a-f]{64}$'),
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (participacion_ref, version),
    FOREIGN KEY (participacion_ref, version) REFERENCES vec_bolsa_llamamientos.datos_contacto_participacion (participacion_ref, version),
    CHECK (vigente_hasta > registrada_en)
);
ALTER TABLE vec_bolsa_llamamientos.origen_datos_contacto_participacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.origen_datos_contacto_participacion FORCE ROW LEVEL SECURITY;
CREATE POLICY origen_datos_contacto_participacion_solo_propietario ON vec_bolsa_llamamientos.origen_datos_contacto_participacion TO vec_bolsa_llamamientos_propietario USING (current_user = 'vec_bolsa_llamamientos_propietario') WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.origen_datos_contacto_participacion FROM PUBLIC;
CREATE TRIGGER origen_datos_contacto_participacion_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.origen_datos_contacto_participacion FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Alta con origen: la misma transacción registra la versión B4 (y consume su
-- concesión V3) y la marca de origen. Un replay exige que la versión original
-- lleve la misma marca; si no, la clave se reutilizó con otros datos.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_origen_participacion_v1(
    p_bolsa_ref text, p_participacion_ref text, p_version bigint, p_clave_ref text, p_nonce bytea, p_cifrado bytea,
    p_motivo text, p_actor text, p_registrada_en timestamptz, p_clave_idempotencia text, p_recibo_ref text,
    p_capacidad bytea, p_decision bytea, p_motivo_autorizacion bytea, p_contexto bytea, p_persona_version numeric,
    p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea,
    p_origen text, p_vigente_hasta timestamptz, p_ultimo_dia date, p_regla_ref text, p_regla_huella_sha256 text)
RETURNS TABLE(o_reutilizada boolean, o_recibo_ref text, o_version bigint, o_registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET lock_timeout = '2s' AS $f$
DECLARE alta record;
BEGIN
 IF p_origen IS DISTINCT FROM 'convoca' OR p_vigente_hasta IS NULL OR p_ultimo_dia IS NULL OR p_registrada_en IS NULL
    OR p_vigente_hasta <= p_registrada_en OR p_ultimo_dia < (p_registrada_en AT TIME ZONE 'Europe/Madrid')::date
    OR p_regla_ref IS NULL OR p_regla_ref <> btrim(p_regla_ref) OR octet_length(p_regla_ref) NOT BETWEEN 1 AND 512
    OR p_regla_huella_sha256 IS NULL OR p_regla_huella_sha256 !~ '^[0-9a-f]{64}$' THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='origen de datos de contacto invalido';
 END IF;
 SELECT * INTO STRICT alta FROM vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1(
   p_bolsa_ref, p_participacion_ref, p_version, p_clave_ref, p_nonce, p_cifrado, p_motivo, p_actor, p_registrada_en,
   p_clave_idempotencia, p_recibo_ref, p_capacidad, p_decision, p_motivo_autorizacion, p_contexto, p_persona_version,
   p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 IF alta.reutilizada THEN
  IF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.origen_datos_contacto_participacion o
                  WHERE o.participacion_ref = p_participacion_ref AND o.version = alta.version AND o.origen = p_origen) THEN
   RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='origen de datos de contacto divergente';
  END IF;
 ELSE
  INSERT INTO vec_bolsa_llamamientos.origen_datos_contacto_participacion (participacion_ref, version, origen, vigente_hasta, ultimo_dia, regla_ref, regla_huella_sha256, registrada_en)
  VALUES (p_participacion_ref, alta.version, p_origen, p_vigente_hasta, p_ultimo_dia, p_regla_ref, p_regla_huella_sha256, alta.registrada_en);
 END IF;
 RETURN QUERY SELECT alta.reutilizada, alta.recibo_ref, alta.version, alta.registrada_en;
END $f$;

-- Lectura de la marca de una versión concreta. Sin fila: contacto sin origen
-- CONVOCA (propio o registrado por RRHH).
CREATE FUNCTION vec_bolsa_llamamientos.leer_origen_datos_contacto_participacion_v1(p_participacion_ref text, p_version bigint)
RETURNS TABLE(origen text, vigente_hasta timestamptz, ultimo_dia date, regla_ref text, regla_huella_sha256 text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT o.origen, o.vigente_hasta, o.ultimo_dia, o.regla_ref, o.regla_huella_sha256
   FROM vec_bolsa_llamamientos.origen_datos_contacto_participacion o
  WHERE o.participacion_ref = p_participacion_ref AND o.version = p_version
$f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_origen_participacion_v1(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,timestamptz,date,text,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_origen_datos_contacto_participacion_v1(text,bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_origen_participacion_v1(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,timestamptz,date,text,text) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_origen_datos_contacto_participacion_v1(text,bigint) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
