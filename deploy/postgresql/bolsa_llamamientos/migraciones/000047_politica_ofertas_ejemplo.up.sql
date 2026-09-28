\set ON_ERROR_STOP on
-- B47. Política de EJEMPLO por bolsa para 3.06/3.07. Cada edición publica
-- versión nueva; las ofertas guardan la versión/huella y las versiones de
-- Calendarios usadas. La propuesta de B-OF sigue requiriendo confirmación RRHH.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000047',0));

DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.oferta_publicada') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.politica_ofertas_version') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_politica_ofertas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'B47: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_bolsa_llamamientos.politica_ofertas_version(
 bolsa_ref text NOT NULL CHECK (bolsa_ref ~ '^bolsa:[A-Za-z0-9:_-]{1,250}$'),
 version bigint NOT NULL CHECK (version>=1),
 politica jsonb NOT NULL,
 huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
 ejemplo boolean NOT NULL CHECK (ejemplo),
 actor_ref text NOT NULL CHECK (actor_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 clave_idempotencia text NOT NULL CHECK (clave_idempotencia ~ '^[A-Za-z0-9:_-]+$' AND octet_length(clave_idempotencia) BETWEEN 8 AND 256),
 version_esperada bigint NOT NULL CHECK (version_esperada>=0),
 recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:politica-ofertas:[0-9a-f]{64}$'),
 publicada_en timestamptz(6) NOT NULL,
 decision_ref text NOT NULL UNIQUE,
 auditoria_ref text NOT NULL UNIQUE,
 PRIMARY KEY (bolsa_ref,version),
 UNIQUE (bolsa_ref,clave_idempotencia),
 CHECK (version=version_esperada+1)
);
CREATE TRIGGER politica_ofertas_inmutable
 BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.politica_ofertas_version
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

CREATE TABLE vec_bolsa_llamamientos.politica_ofertas_outbox(
 recibo_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.politica_ofertas_version(recibo_ref),
 bolsa_ref text NOT NULL,
 version bigint NOT NULL,
 huella_sha256 text NOT NULL,
 creada_en timestamptz(6) NOT NULL,
 entregada_en timestamptz(6),
 FOREIGN KEY (bolsa_ref,version) REFERENCES vec_bolsa_llamamientos.politica_ofertas_version(bolsa_ref,version)
);

CREATE FUNCTION vec_bolsa_llamamientos.leer_politica_ofertas_v1(p_bolsa text)
RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT coalesce((SELECT jsonb_build_object(
   'bolsa_ref',p.bolsa_ref,'version',p.version,'huella_sha256',p.huella_sha256,
   'ejemplo',true,'configurada',true,'politica',p.politica,'recibo_ref',p.recibo_ref,
   'publicada_en',to_char(p.publicada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))
  FROM vec_bolsa_llamamientos.politica_ofertas_version p WHERE p.bolsa_ref=p_bolsa
  ORDER BY p.version DESC LIMIT 1),
  jsonb_build_object('bolsa_ref',p_bolsa,'version',0,'ejemplo',true,'configurada',false,'politica',NULL))
$f$;

CREATE FUNCTION vec_bolsa_llamamientos.publicar_politica_ofertas_v1(
 p_bolsa text,p_version_esperada bigint,p_politica jsonb,p_actor text,p_clave text,p_recibo text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(politica jsonb,reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE consumo record; d jsonb; previa record; v_version bigint; v_huella text; v_ahora timestamptz:=clock_timestamp();
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_bolsa IS NULL OR p_bolsa !~ '^bolsa:[A-Za-z0-9:_-]{1,250}$'
    OR p_version_esperada IS NULL OR p_version_esperada<0 OR p_version_esperada>2147483646
    OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR p_clave !~ '^[A-Za-z0-9:_-]+$' OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR p_recibo IS NULL OR p_recibo !~ '^recibo:politica-ofertas:[0-9a-f]{64}$'
    OR jsonb_typeof(p_politica) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica))<>3
    OR NOT (p_politica ?& ARRAY['plazo','adjudicacion','no_cubierta'])
    OR jsonb_typeof(p_politica->'plazo') IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'plazo'))<>4
    OR NOT (p_politica->'plazo' ?& ARRAY['unidad','cantidad','computo','municipio_sede'])
    OR coalesce(p_politica#>>'{plazo,unidad}','') NOT IN ('dias_habiles','dias_naturales')
    OR jsonb_typeof(p_politica#>'{plazo,cantidad}') IS DISTINCT FROM 'number'
    OR coalesce(p_politica#>>'{plazo,cantidad}','') !~ '^[0-9]{1,2}$'
    OR (p_politica#>'{plazo,cantidad}') NOT BETWEEN '1'::jsonb AND '30'::jsonb
    OR p_politica#>>'{plazo,computo}' IS DISTINCT FROM 'administrativo'
    OR coalesce(p_politica#>>'{plazo,municipio_sede}','') !~ '^[0-9]{5}$'
    OR jsonb_typeof(p_politica->'adjudicacion') IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'adjudicacion'))<>2
    OR p_politica#>>'{adjudicacion,criterio}' IS DISTINCT FROM 'orden_vigente'
    OR p_politica#>>'{adjudicacion,elegibilidad}' IS DISTINCT FROM 'disposicion_en_plazo'
    OR jsonb_typeof(p_politica->'no_cubierta') IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'no_cubierta'))<>2
    OR p_politica#>>'{no_cubierta,accion}' IS DISTINCT FROM 'llamamiento_directo'
    OR p_politica#>>'{no_cubierta,condicion}' IS DISTINCT FROM 'sin_disposiciones_elegibles'
 THEN RAISE EXCEPTION 'B47: política de ejemplo inválida' USING ERRCODE='22023'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_politica_ofertas_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B47: autorización inválida' USING ERRCODE='42501'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_bolsa OR consumo.consumo_nuevo IS NOT TRUE
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'bolsa.politica_ofertas.publicar'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida'
    OR d->>'finalidad' IS DISTINCT FROM 'gobierno_politica_ofertas_bolsa'
    OR d->>'recurso_ref' IS DISTINCT FROM p_bolsa
 THEN RAISE EXCEPTION 'B47: publicación no autorizada' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:politica-ofertas:'||p_bolsa,0));
 v_huella:=encode(sha256(convert_to(p_politica::text,'UTF8')),'hex');
 SELECT * INTO previa FROM vec_bolsa_llamamientos.politica_ofertas_version
  WHERE bolsa_ref=p_bolsa AND clave_idempotencia=p_clave;
 IF FOUND THEN
  IF previa.actor_ref<>p_actor OR previa.version_esperada<>p_version_esperada
     OR previa.huella_sha256<>v_huella OR previa.recibo_ref<>p_recibo
  THEN RAISE EXCEPTION 'B47: clave reutilizada' USING ERRCODE='VBP01'; END IF;
  RETURN QUERY SELECT jsonb_build_object('bolsa_ref',previa.bolsa_ref,'version',previa.version,
   'huella_sha256',previa.huella_sha256,'ejemplo',true,'configurada',true,'politica',previa.politica,
   'recibo_ref',previa.recibo_ref,
   'publicada_en',to_char(previa.publicada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),true;
  RETURN;
 END IF;
 SELECT coalesce(max(version),0) INTO v_version FROM vec_bolsa_llamamientos.politica_ofertas_version WHERE bolsa_ref=p_bolsa;
 IF v_version<>p_version_esperada THEN
  RAISE EXCEPTION 'B47: versión esperada obsoleta' USING ERRCODE='VBP01';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion WHERE bolsa_ref=p_bolsa) THEN
  RAISE EXCEPTION 'B47: bolsa no constituida' USING ERRCODE='23503'; END IF;
 INSERT INTO vec_bolsa_llamamientos.politica_ofertas_version(
  bolsa_ref,version,politica,huella_sha256,ejemplo,actor_ref,clave_idempotencia,version_esperada,
  recibo_ref,publicada_en,decision_ref,auditoria_ref)
 VALUES(p_bolsa,v_version+1,p_politica,v_huella,true,p_actor,p_clave,p_version_esperada,
  p_recibo,v_ahora,consumo.decision_ref,consumo.auditoria_ref);
 INSERT INTO vec_bolsa_llamamientos.politica_ofertas_outbox(recibo_ref,bolsa_ref,version,huella_sha256,creada_en)
 VALUES(p_recibo,p_bolsa,v_version+1,v_huella,v_ahora);
 RETURN QUERY SELECT vec_bolsa_llamamientos.leer_politica_ofertas_v1(p_bolsa),false;
END $f$;

-- Solo ofertas nuevas: una edición posterior no altera los plazos publicados.
CREATE FUNCTION vec_bolsa_llamamientos.verificar_politica_oferta_b47()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE v record;
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:politica-ofertas:'||NEW.bolsa_ref,0));
 SELECT * INTO v FROM vec_bolsa_llamamientos.politica_ofertas_version
  WHERE bolsa_ref=NEW.bolsa_ref ORDER BY version DESC LIMIT 1;
 IF NOT FOUND OR jsonb_typeof(NEW.plazo) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(NEW.plazo))<>10
    OR NOT (NEW.plazo ?& ARRAY['regla_ref','huella_catalogo','unidad','cantidad','computo',
                                'ultimo_dia','ejemplo','calendarios','politica_version','municipio_sede'])
    OR NEW.plazo->>'politica_version' IS DISTINCT FROM v.version::text
    OR NEW.plazo->>'huella_catalogo' IS DISTINCT FROM v.huella_sha256
    OR NEW.plazo->>'regla_ref' IS DISTINCT FROM 'politica-ofertas:'||NEW.bolsa_ref||':'||v.version::text
    OR NEW.plazo->>'unidad' IS DISTINCT FROM v.politica#>>'{plazo,unidad}'
    OR NEW.plazo->>'cantidad' IS DISTINCT FROM v.politica#>>'{plazo,cantidad}'
    OR NEW.plazo->>'computo' IS DISTINCT FROM v.politica#>>'{plazo,computo}'
    OR NEW.plazo->>'municipio_sede' IS DISTINCT FROM v.politica#>>'{plazo,municipio_sede}'
    OR NEW.plazo->>'ejemplo' IS DISTINCT FROM 'true'
    OR (NEW.vence_antes_de AT TIME ZONE 'Europe/Madrid') IS DISTINCT FROM
       ((NEW.plazo->>'ultimo_dia')::date + 1)::timestamp
    OR jsonb_typeof(NEW.plazo->'calendarios') IS DISTINCT FROM 'array'
    OR jsonb_array_length(NEW.plazo->'calendarios') NOT BETWEEN 1 AND 16
    OR EXISTS(SELECT 1 FROM jsonb_array_elements(NEW.plazo->'calendarios') c
              WHERE jsonb_typeof(c) IS DISTINCT FROM 'string' OR octet_length(c#>>'{}') NOT BETWEEN 1 AND 256)
 THEN RAISE EXCEPTION 'B47: oferta sin política de ejemplo vigente' USING ERRCODE='VBP02'; END IF;
 RETURN NEW;
END $f$;
CREATE TRIGGER oferta_politica_b47 BEFORE INSERT ON vec_bolsa_llamamientos.oferta_publicada
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.verificar_politica_oferta_b47();

-- La fachada B28 ya instalada conserva su definición e historia. B47 retira
-- su EXECUTE técnico y expone V2: la decisión V3 emitida por la aplicación
-- tras consultar Calendarios liga bolsa, fechas, política y calendarios
-- utilizados. El material no puede cambiar entre autorización e INSERT.
CREATE FUNCTION vec_bolsa_llamamientos.publicar_oferta_v2(
 p_oferta text,p_recibo text,p_bolsa text,p_actor text,p_clave text,p_datos jsonb,p_plazo jsonb,
 p_publicada timestamptz,p_vence timestamptz,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_unidad text,p_ambito text)
RETURNS TABLE(oferta jsonb,reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE d jsonb; v_calendarios text; v_material text; v_contexto text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR p_publicada IS NULL OR p_vence IS NULL
    OR p_bolsa IS NULL OR p_plazo IS NULL
    OR p_unidad IS NULL OR p_unidad !~ '^[A-Za-z0-9:_-]+$' OR octet_length(p_unidad) NOT BETWEEN 1 AND 256
    OR p_ambito IS NULL OR p_ambito !~ '^[A-Za-z0-9:_-]+$' OR octet_length(p_ambito) NOT BETWEEN 1 AND 256
    OR jsonb_typeof(p_plazo) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_plazo))<>10
    OR NOT (p_plazo ?& ARRAY['regla_ref','huella_catalogo','unidad','cantidad','computo',
                              'ultimo_dia','ejemplo','calendarios','politica_version','municipio_sede'])
    OR jsonb_typeof(p_plazo->'calendarios') IS DISTINCT FROM 'array'
    OR jsonb_array_length(p_plazo->'calendarios') NOT BETWEEN 1 AND 16
    OR EXISTS(SELECT 1 FROM jsonb_array_elements(p_plazo->'calendarios') c
              WHERE jsonb_typeof(c) IS DISTINCT FROM 'string' OR c#>>'{}' !~ '^[A-Za-z0-9:_-]+$')
    OR p_plazo->>'regla_ref' IS NULL OR p_plazo->>'huella_catalogo' IS NULL
    OR p_plazo->>'unidad' IS NULL OR p_plazo->>'cantidad' IS NULL
    OR p_plazo->>'computo' IS NULL OR p_plazo->>'municipio_sede' IS NULL
    OR p_plazo->>'ultimo_dia' IS NULL OR p_plazo->>'politica_version' IS NULL
 THEN RAISE EXCEPTION 'B47: material de plazo incompleto' USING ERRCODE='22023'; END IF;
 SELECT string_agg(c#>>'{}',chr(30) ORDER BY n) INTO v_calendarios
 FROM jsonb_array_elements(p_plazo->'calendarios') WITH ORDINALITY a(c,n);
 v_material:=encode(sha256(convert_to(array_to_string(ARRAY[
  p_bolsa,
  to_char(p_publicada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  to_char(p_vence AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  p_plazo->>'regla_ref',p_plazo->>'huella_catalogo',p_plazo->>'unidad',
  p_plazo->>'cantidad',p_plazo->>'computo',p_plazo->>'municipio_sede',
  p_plazo->>'ultimo_dia',p_plazo->>'politica_version',v_calendarios
 ],chr(31)),'UTF8')),'hex');
 v_contexto:=encode(sha256(convert_to(
  '{"ambitos":{"ambito_ref":"'||p_ambito||'","unidad_ref":"'||p_unidad||
  '"},"atributos":{"material_sha256":"'||v_material||'"}}','UTF8')),'hex');
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B47: decisión de oferta inválida' USING ERRCODE='42501'; END;
 IF d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto THEN
  RAISE EXCEPTION 'B47: plazo distinto del autorizado' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT * FROM vec_bolsa_llamamientos.publicar_oferta_v1(
  p_oferta,p_recibo,p_bolsa,p_actor,p_clave,p_datos,p_plazo,p_publicada,p_vence,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
END $f$;

REVOKE ALL ON TABLE vec_bolsa_llamamientos.politica_ofertas_version,
 vec_bolsa_llamamientos.politica_ofertas_outbox FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_politica_ofertas_v1(text),
 vec_bolsa_llamamientos.publicar_politica_ofertas_v1(text,bigint,jsonb,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.publicar_oferta_v2(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text),
 vec_bolsa_llamamientos.verificar_politica_oferta_b47() FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION vec_bolsa_llamamientos.publicar_oferta_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_politica_ofertas_v1(text),
 vec_bolsa_llamamientos.publicar_politica_ofertas_v1(text,bigint,jsonb,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.publicar_oferta_v2(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)
 TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
