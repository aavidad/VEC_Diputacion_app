\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000120',0));
DO $pre$
DECLARE h text;
BEGIN
 SELECT encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(
  'vec_bolsa_llamamientos.publicar_oferta_v5(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer,jsonb)')),'UTF8')),'hex') INTO h;
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR h IS DISTINCT FROM '3811f4fbb392f0b8c37f2583ae163e0aefafdf71fa34e2f7fc62454ef236d995'
    OR to_regclass('vec_bolsa_llamamientos.origen_oferta_ct_v1') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.plazas_oferta') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.verificar_oferta_ct_v1(text,text,bigint,text,text,text,text,text,text,integer,jsonb,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'B120: preimagen instalada incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Fachada del dueño de Bolsa. CT presenta el comando congelado; Bolsa coteja
-- su origen canónico y la publicación efectiva sin exponer sus tablas.
CREATE FUNCTION vec_bolsa_llamamientos.verificar_oferta_ct_v1(
 p_organizacion text,p_expediente text,p_version bigint,p_clave_ct text,p_actor text,
 p_bolsa text,p_clave_bolsa text,p_oferta text,p_recibo text,p_numero integer,
 p_origen jsonb,p_origen_huella text)
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER') IS TRUE
    OR p_organizacion IS NULL OR p_expediente IS NULL OR p_clave_ct IS NULL
    OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_bolsa IS NULL OR p_clave_bolsa IS NULL
    OR octet_length(p_clave_bolsa) NOT BETWEEN 8 AND 256
    OR p_oferta !~ '^oferta:[0-9a-f]{64}$'
    OR p_recibo !~ '^recibo:oferta:[0-9a-f]{64}$'
    OR p_version NOT BETWEEN 1 AND 9007199254740991
    OR p_numero NOT BETWEEN 2 AND 100
    OR jsonb_typeof(p_origen) IS DISTINCT FROM 'object'
    OR octet_length(p_origen::text)>1048576
    OR p_origen_huella !~ '^[0-9a-f]{64}$'
    OR p_origen_huella IS DISTINCT FROM encode(sha256(convert_to(p_origen::text,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'B120: verificación nominal denegada' USING ERRCODE='42501'; END IF;
 RETURN EXISTS(
  SELECT 1 FROM vec_bolsa_llamamientos.origen_oferta_ct_v1 origen
  JOIN vec_bolsa_llamamientos.oferta_publicada oferta
    ON oferta.oferta_ref=origen.oferta_ref
  JOIN vec_bolsa_llamamientos.plazas_oferta plazas
    ON plazas.oferta_ref=origen.oferta_ref
  JOIN vec_bolsa_llamamientos.origen_oferta_ct_outbox_v1 evento
    ON evento.evento_ref=origen.evento_ref AND evento.oferta_ref=origen.oferta_ref
  WHERE origen.organizacion_ref=p_organizacion
    AND origen.expediente_ref=p_expediente
    AND origen.version_expediente=p_version
    AND origen.clave_ct=p_clave_ct
    AND origen.bolsa_ref=p_bolsa
    AND origen.oferta_ref=p_oferta
    AND origen.recibo_ref=p_recibo
    AND origen.numero_plazas=p_numero
    AND origen.origen_canonico=p_origen
    AND origen.origen_huella_sha256=p_origen_huella
    AND oferta.bolsa_ref=p_bolsa
    AND oferta.actor_ref=p_actor
    AND oferta.recibo_ref=p_recibo
    AND oferta.clave_idempotencia=p_clave_bolsa
    AND plazas.bolsa_ref=p_bolsa
    AND plazas.numero_plazas=p_numero
    AND evento.tipo='bolsa.oferta.ct.origen_confirmado'
 );
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.verificar_oferta_ct_v1(text,text,bigint,text,text,text,text,text,text,integer,jsonb,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.verificar_oferta_ct_v1(text,text,bigint,text,text,text,text,text,text,integer,jsonb,text)
 TO vec_contratacion_temporal_propietario;
COMMIT;
