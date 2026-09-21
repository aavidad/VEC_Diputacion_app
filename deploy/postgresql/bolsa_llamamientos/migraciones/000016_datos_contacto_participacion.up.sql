\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000016', 0));

-- B4 (Peticion.pdf p.1: «correo electrónico y dos teléfonos» del candidato;
-- pliego SE15/2020 §1 c.2): datos de contacto de una participación, versionados
-- append-only. El agregado (correo, teléfono 1, teléfono 2) se guarda como un
-- único sobre AEAD cifrado por la aplicación con clave del KMS y ligado a la
-- participación y a la versión; la base nunca ve el claro ni ninguna huella de
-- él. Mismo circuito que B2: concesión V3 consumida, auditoría y alta en una
-- transacción; idempotencia por clave y recibo determinista.
CREATE TABLE vec_bolsa_llamamientos.datos_contacto_participacion (
    participacion_ref text NOT NULL,
    version bigint NOT NULL CHECK (version > 0),
    clave_ref text NOT NULL CHECK (octet_length(clave_ref) BETWEEN 1 AND 256),
    nonce bytea NOT NULL CHECK (octet_length(nonce) BETWEEN 12 AND 32),
    cifrado bytea NOT NULL CHECK (octet_length(cifrado) BETWEEN 16 AND 4096),
    motivo text NOT NULL CHECK (octet_length(motivo) BETWEEN 1 AND 1000 AND motivo = btrim(motivo)),
    actor text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    clave_idempotencia text NOT NULL CHECK (octet_length(clave_idempotencia) BETWEEN 1 AND 256 AND clave_idempotencia = btrim(clave_idempotencia)),
    recibo_ref text NOT NULL CHECK (octet_length(recibo_ref) BETWEEN 1 AND 256 AND recibo_ref = btrim(recibo_ref)),
    PRIMARY KEY (participacion_ref, version),
    UNIQUE (participacion_ref, clave_idempotencia),
    UNIQUE (recibo_ref)
);
ALTER TABLE vec_bolsa_llamamientos.datos_contacto_participacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.datos_contacto_participacion FORCE ROW LEVEL SECURITY;
CREATE POLICY datos_contacto_participacion_solo_propietario ON vec_bolsa_llamamientos.datos_contacto_participacion TO vec_bolsa_llamamientos_propietario USING (current_user = 'vec_bolsa_llamamientos_propietario') WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.datos_contacto_participacion FROM PUBLIC;
CREATE TRIGGER datos_contacto_participacion_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.datos_contacto_participacion FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- La bitácora de frontera compartida admite también los fallos de B4.
DO $bitacora$
DECLARE accion text; ruta text;
BEGIN
 SELECT pg_get_constraintdef(oid,true) INTO STRICT accion FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass AND conname='bitacora_intento_borrador_llamamiento_accion_check' AND contype='c';
 SELECT pg_get_constraintdef(oid,true) INTO STRICT ruta FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass AND conname='bitacora_intento_borrador_llamamiento_ruta_clase_check' AND contype='c';
 IF strpos(accion,'''registrar_datos_contacto''')<>0 OR strpos(ruta,'''datos_contacto''')<>0 OR strpos(accion,'CHECK (accion = ANY (ARRAY[')<>1 OR right(accion,3)<>']))' OR strpos(ruta,'CHECK (ruta_clase = ANY (ARRAY[')<>1 OR right(ruta,3)<>']))' THEN
  RAISE EXCEPTION 'bitácora de frontera incompatible con B4' USING ERRCODE='55000';
 END IF;
 ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento DROP CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check;
 EXECUTE 'ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento ADD CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check '||left(accion,length(accion)-3)||', ''registrar_datos_contacto''::text]))';
 ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento DROP CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check;
 EXECUTE 'ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento ADD CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check '||left(ruta,length(ruta)-3)||', ''datos_contacto''::text]))';
END $bitacora$;

CREATE FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1(
    p_bolsa_ref text, p_participacion_ref text, p_version bigint, p_clave_ref text, p_nonce bytea, p_cifrado bytea,
    p_motivo text, p_actor text, p_registrada_en timestamptz, p_clave_idempotencia text, p_recibo_ref text,
    p_capacidad bytea, p_decision bytea, p_motivo_autorizacion bytea, p_contexto bytea, p_persona_version numeric,
    p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, recibo_ref text, version bigint, registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE consumo record; decision jsonb; v_vigente bigint;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_bolsa_ref IS NULL OR p_participacion_ref IS NULL OR p_version IS NULL OR p_version < 1
    OR p_clave_ref IS NULL OR octet_length(p_clave_ref) NOT BETWEEN 1 AND 256 OR p_nonce IS NULL OR octet_length(p_nonce) NOT BETWEEN 12 AND 32
    OR p_cifrado IS NULL OR octet_length(p_cifrado) NOT BETWEEN 16 AND 4096
    OR p_motivo IS NULL OR p_motivo <> btrim(p_motivo) OR octet_length(p_motivo) NOT BETWEEN 1 AND 1000 OR p_actor IS NULL OR p_registrada_en IS NULL
    OR p_clave_idempotencia IS NULL OR p_clave_idempotencia <> btrim(p_clave_idempotencia) OR octet_length(p_clave_idempotencia) NOT BETWEEN 1 AND 256
    OR p_recibo_ref IS NULL OR p_recibo_ref <> btrim(p_recibo_ref) OR octet_length(p_recibo_ref) NOT BETWEEN 1 AND 256 THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='datos de contacto invalidos';
 END IF;
 IF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e JOIN vec_bolsa_llamamientos.constitucion c USING (instantanea_ref, version_instantanea) WHERE e.participacion_ref = p_participacion_ref AND c.bolsa_ref = p_bolsa_ref) THEN
  RAISE EXCEPTION USING ERRCODE='23503', MESSAGE='participacion ajena a la bolsa';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:datos_contacto:' || p_participacion_ref, 0));
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_datos_contacto_participacion_v3_atestada(p_capacidad, p_decision, p_motivo_autorizacion, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 BEGIN decision := convert_from(p_decision, 'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='datos de contacto no autorizados'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_participacion_ref OR consumo.consumo_nuevo IS NOT TRUE
    OR decision->>'principal_id' IS DISTINCT FROM p_actor OR decision->>'accion' IS DISTINCT FROM 'bolsa.datos_contacto_participacion.registrar'
    OR decision->>'modulo_id' IS DISTINCT FROM 'bolsa' OR decision->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa'
    OR decision->>'finalidad' IS DISTINCT FROM 'gestion_datos_contacto_participacion' OR decision->>'recurso_ref' IS DISTINCT FROM p_participacion_ref
    OR decision->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM decision->>'contexto_recurso_huella_sha256' THEN
  RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='datos de contacto no autorizados';
 END IF;
 -- Replay: la aplicación ya comparó el claro; aquí basta devolver el recibo original.
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.datos_contacto_participacion d WHERE d.participacion_ref = p_participacion_ref AND d.clave_idempotencia = p_clave_idempotencia) THEN
  RETURN QUERY SELECT true, d.recibo_ref, d.version, d.registrada_en FROM vec_bolsa_llamamientos.datos_contacto_participacion d WHERE d.participacion_ref = p_participacion_ref AND d.clave_idempotencia = p_clave_idempotencia;
  RETURN;
 END IF;
 SELECT coalesce(max(d.version), 0) INTO v_vigente FROM vec_bolsa_llamamientos.datos_contacto_participacion d WHERE d.participacion_ref = p_participacion_ref;
 IF p_version <> v_vigente + 1 THEN
  RAISE EXCEPTION USING ERRCODE='VBS02', MESSAGE='version de datos de contacto no consecutiva';
 END IF;
 INSERT INTO vec_bolsa_llamamientos.datos_contacto_participacion (participacion_ref, version, clave_ref, nonce, cifrado, motivo, actor, registrada_en, clave_idempotencia, recibo_ref)
 VALUES (p_participacion_ref, p_version, p_clave_ref, p_nonce, p_cifrado, p_motivo, p_actor, p_registrada_en, p_clave_idempotencia, p_recibo_ref);
 RETURN QUERY SELECT false, p_recibo_ref, p_version, p_registrada_en;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.leer_datos_contacto_participacion_v1(p_participacion_ref text)
RETURNS TABLE(version bigint, clave_ref text, nonce bytea, cifrado bytea, motivo text, registrada_en timestamptz, recibo_ref text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT d.version, d.clave_ref, d.nonce, d.cifrado, d.motivo, d.registrada_en, d.recibo_ref
   FROM vec_bolsa_llamamientos.datos_contacto_participacion d
  WHERE d.participacion_ref = p_participacion_ref
  ORDER BY d.version DESC LIMIT 1
$f$;

CREATE FUNCTION vec_bolsa_llamamientos.recuperar_datos_contacto_participacion_v1(p_participacion_ref text, p_clave_idempotencia text)
RETURNS TABLE(version bigint, clave_ref text, nonce bytea, cifrado bytea, motivo text, registrada_en timestamptz, recibo_ref text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT d.version, d.clave_ref, d.nonce, d.cifrado, d.motivo, d.registrada_en, d.recibo_ref
   FROM vec_bolsa_llamamientos.datos_contacto_participacion d
  WHERE d.participacion_ref = p_participacion_ref AND d.clave_idempotencia = p_clave_idempotencia
$f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_datos_contacto_participacion_v1(text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.recuperar_datos_contacto_participacion_v1(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_datos_contacto_participacion_v1(text) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.recuperar_datos_contacto_participacion_v1(text,text) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
