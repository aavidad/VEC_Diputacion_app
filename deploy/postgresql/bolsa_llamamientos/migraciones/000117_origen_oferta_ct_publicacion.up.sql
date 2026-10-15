\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000117',0));
DO $pre$
DECLARE h_pol text;h_trig text;h_v1 text;h_v4 text;
BEGIN
 SELECT encode(sha256(convert_to(pg_get_functiondef(to_regprocedure('vec_bolsa_llamamientos.publicar_politica_ofertas_v1(text,bigint,jsonb,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')),'UTF8')),'hex') INTO h_pol;
 SELECT encode(sha256(convert_to(pg_get_functiondef(to_regprocedure('vec_bolsa_llamamientos.verificar_politica_oferta_b47()')),'UTF8')),'hex') INTO h_trig;
 SELECT encode(sha256(convert_to(pg_get_functiondef(to_regprocedure('vec_bolsa_llamamientos.publicar_oferta_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')),'UTF8')),'hex') INTO h_v1;
 SELECT encode(sha256(convert_to(pg_get_functiondef(to_regprocedure('vec_bolsa_llamamientos.publicar_oferta_v4(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer)')),'UTF8')),'hex') INTO h_v4;
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR current_setting('server_version_num')::integer<180000
    OR h_pol IS DISTINCT FROM '6477623babb3a8afab2d70a3936455a317198a7742c798571d29c627a3137387'
    OR h_trig IS DISTINCT FROM '7e5c6e748f3e9ff5ec43e4a45e8451d64f6003c812a789bf11b71d390468e12f'
    OR h_v1 IS DISTINCT FROM 'a86bf11335cb23bbc20835a5d280f906c1ce1538334b5e064e654098ae306211'
    OR h_v4 IS DISTINCT FROM '94f0a5c9cd173d4d4869f196fce7d4cc08b16b553dd08ef327fc7a71cec47436'
    OR to_regclass('vec_bolsa_llamamientos.oferta_publicada') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.plazas_oferta') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.politica_ofertas_version') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.constitucion_entrada') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.instantanea_orden_bolsa') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.origen_oferta_ct_v1') IS NOT NULL
    OR to_regprocedure('vec_bolsa_llamamientos.publicar_oferta_v5(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer,jsonb)') IS NOT NULL
 THEN RAISE EXCEPTION 'B117: PARO preimagen B71/B110 incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Se reconstruyen desde pg_get_functiondef de la principal: misma firma y OID,
-- ACL y propietario; solo admiten la nueva variante de inicio de plazo.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.publicar_politica_ofertas_v1(p_bolsa text, p_version_esperada bigint, p_politica jsonb, p_actor text, p_clave text, p_recibo text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS TABLE(politica jsonb, reutilizada boolean)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO pg_catalog, pg_temp
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE consumo record; d jsonb; previa record; v_version bigint; v_huella text; v_ahora timestamptz:=clock_timestamp();
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_bolsa IS NULL OR p_bolsa !~ '^bolsa:[A-Za-z0-9:_-]{1,250}$'
    OR p_version_esperada IS NULL OR p_version_esperada<0 OR p_version_esperada>2147483646
    OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR p_clave !~ '^[A-Za-z0-9:_-]+$' OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR p_recibo IS NULL OR p_recibo !~ '^recibo:politica-ofertas:[0-9a-f]{64}$'
    OR jsonb_typeof(p_politica) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica)) NOT IN (3,4)
    OR NOT (p_politica ?& ARRAY['plazo','adjudicacion','no_cubierta'])
    OR ((SELECT count(*) FROM jsonb_object_keys(p_politica))=4 AND NOT (p_politica ? 'plazas'))
    -- B58: apartado opcional de plazas. Solo valores que ejecuta la proyección.
    OR (p_politica ? 'plazas' AND (
        jsonb_typeof(p_politica->'plazas') IS DISTINCT FROM 'object'
        OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'plazas'))<>3
        OR NOT (p_politica->'plazas' ?& ARRAY['llamada','respuesta_horas','tras_renuncia'])
        OR coalesce(p_politica#>>'{plazas,llamada}','') NOT IN ('simultanea','sucesiva')
        OR jsonb_typeof(p_politica#>'{plazas,respuesta_horas}') IS DISTINCT FROM 'number'
        OR coalesce(p_politica#>>'{plazas,respuesta_horas}','') !~ '^([1-9][0-9]?|[1-6][0-9]{2}|7[01][0-9]|720)$'
        OR coalesce(p_politica#>>'{plazas,tras_renuncia}','') NOT IN ('siguiente_en_orden','llamamiento_directo')))
    OR jsonb_typeof(p_politica->'plazo') IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'plazo')) NOT IN (4,5)
    OR ((SELECT count(*) FROM jsonb_object_keys(p_politica->'plazo'))=5
        AND (p_politica#>>'{plazo,inicio}' IS DISTINCT FROM 'notificacion' AND p_politica#>>'{plazo,inicio}' IS DISTINCT FROM 'publicacion'))
    OR NOT (p_politica->'plazo' ?& ARRAY['unidad','cantidad','computo','municipio_sede'])
    OR coalesce(p_politica#>>'{plazo,unidad}','') NOT IN ('dias_habiles','dias_naturales','horas_naturales')
    OR jsonb_typeof(p_politica#>'{plazo,cantidad}') IS DISTINCT FROM 'number'
    OR coalesce(p_politica#>>'{plazo,cantidad}','') !~ '^[0-9]{1,3}$'
    OR ((p_politica#>>'{plazo,unidad}' IN ('dias_habiles','dias_naturales')
             AND (p_politica#>>'{plazo,cantidad}')::int BETWEEN 1 AND 30
             AND p_politica#>>'{plazo,computo}'='administrativo')
         OR (p_politica#>>'{plazo,unidad}'='horas_naturales'
             AND (p_politica#>>'{plazo,cantidad}')::int BETWEEN 1 AND 720
             AND p_politica#>>'{plazo,computo}'='continuo_utc')) IS NOT TRUE
    OR coalesce(p_politica#>>'{plazo,municipio_sede}','') !~ '^[0-9]{5}$'
    OR jsonb_typeof(p_politica->'adjudicacion') IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'adjudicacion')) NOT IN (2,3)
    OR ((SELECT count(*) FROM jsonb_object_keys(p_politica->'adjudicacion'))=3
        AND p_politica#>>'{adjudicacion,confirmacion}' IS DISTINCT FROM 'aceptacion_previa')
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
 -- Las versiones anteriores pueden repetirse, pero toda política nueva declara
 -- en su propia versión que el plazo empieza en la notificación.
 IF (p_politica#>>'{plazo,inicio}' IS DISTINCT FROM 'notificacion' AND p_politica#>>'{plazo,inicio}' IS DISTINCT FROM 'publicacion') THEN
  RAISE EXCEPTION 'B71: inicio de plazo no configurado' USING ERRCODE='22023'; END IF;
 -- El replay histórico ya devolvió arriba el contenido y recibo originales.
 -- Toda versión nueva requiere la aceptación telemática previa.
 IF p_politica#>>'{adjudicacion,confirmacion}' IS DISTINCT FROM 'aceptacion_previa' THEN
  RAISE EXCEPTION 'B70: confirmación telemática requerida para la versión nueva' USING ERRCODE='22023';
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
END $function$;


CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.verificar_politica_oferta_b47()
 RETURNS trigger
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO pg_catalog, pg_temp
AS $function$
DECLARE v record;
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:politica-ofertas:'||NEW.bolsa_ref,0));
 SELECT * INTO v FROM vec_bolsa_llamamientos.politica_ofertas_version
  WHERE bolsa_ref=NEW.bolsa_ref ORDER BY version DESC LIMIT 1;
 IF NOT FOUND OR jsonb_typeof(NEW.plazo) IS DISTINCT FROM 'object'
    OR (v.politica#>>'{plazo,unidad}'='horas_naturales' AND
        (SELECT count(*) FROM jsonb_object_keys(NEW.plazo))<>13)
    OR (v.politica#>>'{plazo,unidad}'<>'horas_naturales' AND
        (SELECT count(*) FROM jsonb_object_keys(NEW.plazo))<>11)
    OR NOT (NEW.plazo ?& ARRAY['regla_ref','huella_catalogo','unidad','cantidad','computo',
                                'ultimo_dia','ejemplo','calendarios','politica_version','municipio_sede'])
    OR NEW.plazo->>'politica_version' IS DISTINCT FROM v.version::text
    OR NEW.plazo->>'huella_catalogo' IS DISTINCT FROM v.huella_sha256
    OR NEW.plazo->>'regla_ref' IS DISTINCT FROM 'politica-ofertas:'||NEW.bolsa_ref||':'||v.version::text
    OR (v.politica#>>'{plazo,inicio}' IS DISTINCT FROM 'notificacion' AND v.politica#>>'{plazo,inicio}' IS DISTINCT FROM 'publicacion')
    OR NEW.plazo->>'unidad' IS DISTINCT FROM v.politica#>>'{plazo,unidad}'
    OR NEW.plazo->>'cantidad' IS DISTINCT FROM v.politica#>>'{plazo,cantidad}'
    OR NEW.plazo->>'computo' IS DISTINCT FROM v.politica#>>'{plazo,computo}'
    OR NEW.plazo->>'municipio_sede' IS DISTINCT FROM v.politica#>>'{plazo,municipio_sede}'
    OR NEW.plazo->>'ejemplo' IS DISTINCT FROM 'true'
    OR (v.politica#>>'{plazo,unidad}'='horas_naturales' AND v.politica#>>'{plazo,inicio}'='notificacion' AND (
      NOT (NEW.plazo ?& ARRAY['apertura_en','vence_en'])
      OR coalesce(NEW.plazo->>'apertura_en','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
      OR coalesce(NEW.plazo->>'vence_en','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
      OR jsonb_typeof(NEW.plazo->'notificacion') IS DISTINCT FROM 'object'
      OR (SELECT count(*) FROM jsonb_object_keys(NEW.plazo->'notificacion'))<>4
      OR NOT (NEW.plazo->'notificacion' ?& ARRAY['notificada_en','referencia_correo','huella_correo_sha256','fuente'])
      OR coalesce(NEW.plazo#>>'{notificacion,notificada_en}','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
      OR coalesce(NEW.plazo#>>'{notificacion,referencia_correo}','') !~ '^[A-Za-z0-9][A-Za-z0-9:_.-]{0,255}$'
      OR (NEW.plazo#>>'{notificacion,referencia_correo}' !~ '^[a-z_]+(:[a-z_]+)*:[0-9a-f]{64}$'
          AND NEW.plazo#>>'{notificacion,referencia_correo}' ~* '(([0-9][._:/#-]?){8}|[XYZ][._:/#-]?([0-9][._:/#-]?){7})[A-Z]')
      OR NEW.plazo#>>'{notificacion,referencia_correo}' ~* '(^|[._:/#-])(dni|nie|nif|pasaporte|passport)([._:/#-]|$)'
      OR coalesce(NEW.plazo#>>'{notificacion,huella_correo_sha256}','') !~ '^[0-9a-f]{64}$'
      OR NEW.plazo#>>'{notificacion,fuente}' IS DISTINCT FROM 'correo_externo_declarado_rrhh'
      OR NEW.plazo->>'apertura_en' IS DISTINCT FROM NEW.plazo#>>'{notificacion,notificada_en}'
      OR NEW.plazo->>'vence_en' IS DISTINCT FROM to_char(NEW.vence_antes_de AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
      OR NEW.vence_antes_de IS DISTINCT FROM (NEW.plazo#>>'{notificacion,notificada_en}')::timestamptz +
         ((v.politica#>>'{plazo,cantidad}')::int * interval '1 hour')
      OR NEW.plazo->>'ultimo_dia' IS DISTINCT FROM
         ((NEW.vence_antes_de - interval '1 microsecond') AT TIME ZONE 'Europe/Madrid')::date::text
      OR NEW.plazo->'calendarios' IS DISTINCT FROM '["calendario:utc-continuo:v1"]'::jsonb))
    OR (v.politica#>>'{plazo,unidad}'<>'horas_naturales' AND v.politica#>>'{plazo,inicio}'='notificacion' AND (
        jsonb_typeof(NEW.plazo->'notificacion') IS DISTINCT FROM 'object'
        OR (SELECT count(*) FROM jsonb_object_keys(NEW.plazo->'notificacion'))<>4
        OR NOT (NEW.plazo->'notificacion' ?& ARRAY['notificada_en','referencia_correo','huella_correo_sha256','fuente'])
        OR coalesce(NEW.plazo#>>'{notificacion,notificada_en}','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
        OR coalesce(NEW.plazo#>>'{notificacion,referencia_correo}','') !~ '^[A-Za-z0-9][A-Za-z0-9:_.-]{0,255}$'
      OR (NEW.plazo#>>'{notificacion,referencia_correo}' !~ '^[a-z_]+(:[a-z_]+)*:[0-9a-f]{64}$'
          AND NEW.plazo#>>'{notificacion,referencia_correo}' ~* '(([0-9][._:/#-]?){8}|[XYZ][._:/#-]?([0-9][._:/#-]?){7})[A-Z]')
      OR NEW.plazo#>>'{notificacion,referencia_correo}' ~* '(^|[._:/#-])(dni|nie|nif|pasaporte|passport)([._:/#-]|$)'
        OR coalesce(NEW.plazo#>>'{notificacion,huella_correo_sha256}','') !~ '^[0-9a-f]{64}$'
        OR NEW.plazo#>>'{notificacion,fuente}' IS DISTINCT FROM 'correo_externo_declarado_rrhh'
        OR (NEW.vence_antes_de AT TIME ZONE 'Europe/Madrid') IS DISTINCT FROM
        ((NEW.plazo->>'ultimo_dia')::date + 1)::timestamp))
    OR (v.politica#>>'{plazo,inicio}'='publicacion' AND (
      NEW.plazo->>'inicio' IS DISTINCT FROM 'publicacion'
      OR NEW.plazo ? 'notificacion'
      OR NEW.publicada_en<>date_trunc('microseconds',NEW.publicada_en)
      OR (v.politica#>>'{plazo,unidad}'='horas_naturales' AND (
        NOT (NEW.plazo ?& ARRAY['apertura_en','vence_en'])
        OR NEW.plazo->>'apertura_en' IS DISTINCT FROM to_char(NEW.publicada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        OR NEW.plazo->>'vence_en' IS DISTINCT FROM to_char(NEW.vence_antes_de AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        OR NEW.vence_antes_de IS DISTINCT FROM NEW.publicada_en +
          ((v.politica#>>'{plazo,cantidad}')::int * interval '1 hour')
        OR NEW.plazo->>'ultimo_dia' IS DISTINCT FROM
          ((NEW.vence_antes_de - interval '1 microsecond') AT TIME ZONE 'Europe/Madrid')::date::text
        OR NEW.plazo->'calendarios' IS DISTINCT FROM '["calendario:utc-continuo:v1"]'::jsonb))
      OR (v.politica#>>'{plazo,unidad}'<>'horas_naturales' AND
        (NEW.vence_antes_de AT TIME ZONE 'Europe/Madrid') IS DISTINCT FROM
          ((NEW.plazo->>'ultimo_dia')::date + 1)::timestamp)))
    OR jsonb_typeof(NEW.plazo->'calendarios') IS DISTINCT FROM 'array'
    OR jsonb_array_length(NEW.plazo->'calendarios') NOT BETWEEN 1 AND 16
    OR EXISTS(SELECT 1 FROM jsonb_array_elements(NEW.plazo->'calendarios') c
              WHERE jsonb_typeof(c) IS DISTINCT FROM 'string' OR octet_length(c#>>'{}') NOT BETWEEN 1 AND 256
                OR c#>>'{}' !~ '^[A-Za-z0-9:_-]+$')
 THEN RAISE EXCEPTION 'B47: oferta sin política de ejemplo vigente' USING ERRCODE='VBP02'; END IF;
 RETURN NEW;
END $function$;


-- El origen CT queda unido en la misma transacción a la oferta B28/B58.
-- La procedencia no reinterpreta ofertas históricas sin esta fila.
CREATE TABLE vec_bolsa_llamamientos.origen_oferta_ct_v1(
 oferta_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.oferta_publicada(oferta_ref),
 recibo_ref text NOT NULL UNIQUE,
 organizacion_ref text NOT NULL,
 unidad_ref text NOT NULL,
 expediente_ref text NOT NULL,
 version_expediente bigint NOT NULL CHECK(version_expediente>0),
 clave_ct text NOT NULL,
 necesidad_ref text NOT NULL,
 acta_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.constitucion(acta_ref),
 categoria_ref text NOT NULL,
 bolsa_ref text NOT NULL,
 version_bolsa bigint NOT NULL,
 huella_bolsa_sha256 text NOT NULL CHECK(huella_bolsa_sha256~'^[0-9a-f]{64}$'),
 instantanea_ref text NOT NULL,
 version_instantanea bigint NOT NULL,
 huella_instantanea_sha256 text NOT NULL CHECK(huella_instantanea_sha256~'^[0-9a-f]{64}$'),
 politica_ref text NOT NULL,
 version_politica bigint NOT NULL,
 huella_politica_sha256 text NOT NULL CHECK(huella_politica_sha256~'^[0-9a-f]{64}$'),
 numero_plazas integer NOT NULL CHECK(numero_plazas BETWEEN 1 AND 100),
 elegibles integer NOT NULL CHECK(elegibles BETWEEN 1 AND 1000000 AND numero_plazas<=elegibles),
 origen_canonico jsonb NOT NULL CHECK(jsonb_typeof(origen_canonico)='object'),
 origen_huella_sha256 text NOT NULL CHECK(origen_huella_sha256~'^[0-9a-f]{64}$'
  AND origen_huella_sha256=encode(sha256(convert_to(origen_canonico::text,'UTF8')),'hex')),
 decision_ref text NOT NULL UNIQUE,
 auditoria_ref text NOT NULL UNIQUE,
 evento_ref text NOT NULL UNIQUE,
 publicada_en timestamptz(6) NOT NULL,
 UNIQUE(expediente_ref,clave_ct),
 FOREIGN KEY(bolsa_ref,version_bolsa,huella_bolsa_sha256)
  REFERENCES vec_bolsa_llamamientos.bolsa_constituida(bolsa_ref,version,huella_bolsa_sha256),
 FOREIGN KEY(instantanea_ref,version_instantanea,huella_instantanea_sha256)
  REFERENCES vec_bolsa_llamamientos.instantanea_orden_bolsa(instantanea_ref,version,huella_instantanea_sha256),
 FOREIGN KEY(politica_ref,version_politica)
  REFERENCES vec_bolsa_llamamientos.politica_orden_bolsa(politica_ref,version)
);
CREATE TABLE vec_bolsa_llamamientos.origen_oferta_ct_outbox_v1(
 evento_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.origen_oferta_ct_v1(evento_ref),
 oferta_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.origen_oferta_ct_v1(oferta_ref),
 tipo text NOT NULL CHECK(tipo='bolsa.oferta.ct.origen_confirmado'),
 estado text NOT NULL CHECK(estado='pendiente'),
 creada_en timestamptz(6) NOT NULL
);
CREATE FUNCTION vec_bolsa_llamamientos.rechazar_truncado_origen_oferta_ct_v1()
RETURNS trigger LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
BEGIN
 RAISE EXCEPTION 'B117: historia de origen inmutable' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.rechazar_truncado_origen_oferta_ct_v1() FROM PUBLIC;

ALTER TABLE vec_bolsa_llamamientos.origen_oferta_ct_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.origen_oferta_ct_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY solo_propietario ON vec_bolsa_llamamientos.origen_oferta_ct_v1 TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.origen_oferta_ct_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.origen_oferta_ct_v1 FROM PUBLIC;
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.origen_oferta_ct_v1
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_bolsa_llamamientos.origen_oferta_ct_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.rechazar_truncado_origen_oferta_ct_v1();

ALTER TABLE vec_bolsa_llamamientos.origen_oferta_ct_outbox_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.origen_oferta_ct_outbox_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY solo_propietario ON vec_bolsa_llamamientos.origen_oferta_ct_outbox_v1 TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.origen_oferta_ct_outbox_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.origen_oferta_ct_outbox_v1 FROM PUBLIC;
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.origen_oferta_ct_outbox_v1
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_bolsa_llamamientos.origen_oferta_ct_outbox_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.rechazar_truncado_origen_oferta_ct_v1();
-- Canon de intención compartido con Go: cada campo lleva longitud UTF-8 decimal.
CREATE FUNCTION vec_bolsa_llamamientos.huella_operacion_oferta_v5(
 p_prefijo text,p_bolsa text,p_clave text,p_actor text,p_numero integer,
 p_datos jsonb,p_origen jsonb,p_notificacion jsonb,p_publicada timestamptz,p_vence timestamptz,p_plazo jsonb)
RETURNS text LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE campos text[];valor text;canon text;calendarios text;elegibles integer;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_prefijo NOT IN ('vec.bolsa.oferta.recuperar.v1','vec.bolsa.oferta.publicar.v1')
    OR p_bolsa IS NULL OR p_clave IS NULL OR p_actor IS NULL OR p_numero NOT BETWEEN 1 AND 100
    OR jsonb_typeof(p_datos) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'B117: canon de oferta inválido' USING ERRCODE='22023';
 END IF;
 campos:=ARRAY[p_prefijo,p_bolsa,p_clave,p_actor,p_numero::text,
  p_datos->>'categoria',p_datos->>'centro',p_datos->>'fecha_inicio',
  coalesce(p_datos->>'fecha_fin',''),p_datos->>'descripcion'];
 IF p_origen IS NULL THEN
  campos:=campos||ARRAY['sin_origen'];
 ELSE
  IF jsonb_typeof(p_origen) IS DISTINCT FROM 'object'
     OR jsonb_typeof(p_origen->'lectura') IS DISTINCT FROM 'object'
     OR jsonb_typeof(p_origen#>'{lectura,entradas}') IS DISTINCT FROM 'array'
     OR coalesce(p_origen#>>'{lectura,corte}','') !~
       '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$' THEN
   RAISE EXCEPTION 'B117: origen CT ilegible' USING ERRCODE='22023';
  END IF;
  SELECT count(*) INTO elegibles FROM jsonb_array_elements(p_origen#>'{lectura,entradas}') e
   WHERE e->>'orden_vigente' IS NOT NULL;
  campos:=campos||ARRAY['con_origen',
   p_origen->>'organizacion_ref',p_origen->>'unidad_ref',p_origen->>'expediente_ref',
   p_origen->>'version_expediente',p_origen->>'clave_ct',p_origen->>'necesidad_ref',
   p_origen->>'acta_ref',p_origen->>'categoria_ref',p_origen->>'bolsa_ref',
   p_origen->>'version_bolsa',p_origen->>'huella_bolsa_sha256',
   p_origen->>'instantanea_ref',p_origen->>'version_instantanea',
   p_origen->>'huella_instantanea_sha256',p_origen->>'politica_ref',
   p_origen->>'version_politica',p_origen->>'huella_politica_sha256',
   p_origen#>>'{lectura,corte}',elegibles::text];
 END IF;
 IF p_notificacion IS NULL THEN
  campos:=campos||ARRAY['sin_notificacion'];
 ELSE
  IF jsonb_typeof(p_notificacion) IS DISTINCT FROM 'object'
     OR (SELECT count(*) FROM jsonb_object_keys(p_notificacion))<>4
     OR NOT (p_notificacion ?& ARRAY['notificada_en','referencia_correo','huella_correo_sha256','fuente'])
     OR coalesce(p_notificacion->>'notificada_en','') !~
       '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$' THEN
   RAISE EXCEPTION 'B117: notificación de recuperación inválida' USING ERRCODE='22023';
  END IF;
  campos:=campos||ARRAY['con_notificacion',p_notificacion->>'notificada_en',
    p_notificacion->>'referencia_correo',p_notificacion->>'huella_correo_sha256',
    p_notificacion->>'fuente'];
 END IF;
 IF p_prefijo='vec.bolsa.oferta.publicar.v1' THEN
  IF p_notificacion IS NOT NULL THEN
   RAISE EXCEPTION 'B117: publicación no usa notificación' USING ERRCODE='22023';
  END IF;
  IF p_publicada IS NULL OR p_vence IS NULL OR jsonb_typeof(p_plazo) IS DISTINCT FROM 'object'
     OR jsonb_typeof(p_plazo->'calendarios') IS DISTINCT FROM 'array' THEN
   RAISE EXCEPTION 'B117: plazo de publicación incompleto' USING ERRCODE='22023';
  END IF;
  IF jsonb_array_length(p_plazo->'calendarios') NOT BETWEEN 1 AND 16
     OR EXISTS(SELECT 1 FROM jsonb_array_elements(p_plazo->'calendarios') c
       WHERE jsonb_typeof(c) IS DISTINCT FROM 'string'
          OR octet_length(c#>>'{}') NOT BETWEEN 1 AND 256
          OR c#>>'{}' !~ '^[A-Za-z0-9:_-]+$') THEN
   RAISE EXCEPTION 'B117: calendario de plazo inválido' USING ERRCODE='22023';
  END IF;
  SELECT string_agg(x.valor,chr(30) ORDER BY x.ord) INTO calendarios
  FROM jsonb_array_elements_text(p_plazo->'calendarios') WITH ORDINALITY x(valor,ord);
  campos:=campos||ARRAY[
   to_char(p_publicada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   to_char(p_vence AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   p_plazo->>'regla_ref',p_plazo->>'huella_catalogo',p_plazo->>'unidad',
   p_plazo->>'cantidad',p_plazo->>'computo',p_plazo->>'municipio_sede',
   p_plazo->>'ultimo_dia',p_plazo->>'politica_version',calendarios,
   coalesce(p_plazo->>'apertura_en',''),coalesce(p_plazo->>'vence_en','')];
 END IF;
 IF array_position(campos,NULL) IS NOT NULL THEN
  RAISE EXCEPTION 'B117: canon de oferta incompleto' USING ERRCODE='22023';
 END IF;
 SELECT string_agg(octet_length(x.valor)::text||':'||x.valor,'' ORDER BY x.ord)
  INTO canon FROM unnest(campos) WITH ORDINALITY x(valor,ord);
 RETURN encode(sha256(convert_to(canon,'UTF8')),'hex');
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.huella_operacion_oferta_v5(text,text,text,text,integer,jsonb,jsonb,jsonb,timestamptz,timestamptz,jsonb) FROM PUBLIC;
CREATE FUNCTION vec_bolsa_llamamientos.validar_origen_oferta_ct_v1(
 p_origen jsonb,p_bolsa text,p_numero integer,p_publicada timestamptz)
RETURNS integer LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET statement_timeout='15s' AS $f$
DECLARE l jsonb; corte timestamptz;cuantos bigint;elegibles integer;actuales integer;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR jsonb_typeof(p_origen) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_origen))<>18
    OR NOT (p_origen ?& ARRAY['organizacion_ref','unidad_ref','expediente_ref',
      'version_expediente','clave_ct','necesidad_ref','acta_ref','categoria_ref',
      'bolsa_ref','version_bolsa','huella_bolsa_sha256','instantanea_ref',
      'version_instantanea','huella_instantanea_sha256','politica_ref',
      'version_politica','huella_politica_sha256','lectura'])
    OR p_origen->>'bolsa_ref' IS DISTINCT FROM p_bolsa
    OR p_origen->>'version_expediente' !~ '^[1-9][0-9]{0,15}$'
    OR p_origen->>'version_bolsa' !~ '^[1-9][0-9]{0,15}$'
    OR p_origen->>'version_instantanea' !~ '^[1-9][0-9]{0,15}$'
    OR p_origen->>'version_politica' !~ '^[1-9][0-9]{0,15}$'
    OR p_origen->>'expediente_ref' IS NULL OR p_origen->>'clave_ct' IS NULL
    OR position(chr(31) in (p_origen->>'expediente_ref'))>0
    OR position(chr(31) in (p_origen->>'clave_ct'))>0
    OR p_origen->>'organizacion_ref' IS NULL OR p_origen->>'necesidad_ref' IS NULL
    OR p_origen->>'unidad_ref' !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    OR p_numero NOT BETWEEN 1 AND 100 OR p_publicada IS NULL THEN
  RAISE EXCEPTION 'B117: origen CT inválido' USING ERRCODE='22023';
 END IF;
 l:=p_origen->'lectura';
 IF jsonb_typeof(l) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(l))<>18
    OR NOT (l ?& ARRAY['acta_ref','categoria_ref','bolsa_ref','version_bolsa',
      'huella_bolsa_sha256','bolsa_canonica','instantanea_ref',
      'version_instantanea','huella_instantanea_sha256','instantanea_canonica',
      'politica_ref','version_politica','huella_politica_sha256',
      'politica_provisional','corte','bolsa_vigente_hasta',
      'politica_vigente_hasta','entradas'])
    -- AD261 publica solo 16 campos; el serializador Go añade estas dos
    -- claves opcionales como null. Nunca se acepta una vigencia del cliente.
    OR jsonb_typeof(l->'bolsa_vigente_hasta') IS DISTINCT FROM 'null'
    OR jsonb_typeof(l->'politica_vigente_hasta') IS DISTINCT FROM 'null'
    OR jsonb_typeof(l->'entradas') IS DISTINCT FROM 'array'
    OR jsonb_array_length(l->'entradas')<1
    OR EXISTS(SELECT 1 FROM jsonb_array_elements(l->'entradas') AS entrada(valor)
      WHERE jsonb_typeof(entrada.valor) IS DISTINCT FROM 'object'
        OR (SELECT count(*) FROM jsonb_object_keys(entrada.valor))<>7
        OR NOT (entrada.valor ?& ARRAY['orden_acta','orden_vigente',
          'participacion_ref','candidato_ref','situacion','razon','elegible'])
        OR jsonb_typeof(entrada.valor->'elegible') IS DISTINCT FROM 'boolean')
    OR coalesce(l->>'corte','') !~
      '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
    OR l->>'acta_ref' IS DISTINCT FROM p_origen->>'acta_ref'
    OR l->>'categoria_ref' IS DISTINCT FROM p_origen->>'categoria_ref'
    OR l->>'bolsa_ref' IS DISTINCT FROM p_bolsa
    OR l->>'version_bolsa' IS DISTINCT FROM p_origen->>'version_bolsa'
    OR l->>'huella_bolsa_sha256' IS DISTINCT FROM p_origen->>'huella_bolsa_sha256'
    OR l->>'instantanea_ref' IS DISTINCT FROM p_origen->>'instantanea_ref'
    OR l->>'version_instantanea' IS DISTINCT FROM p_origen->>'version_instantanea'
    OR l->>'huella_instantanea_sha256' IS DISTINCT FROM p_origen->>'huella_instantanea_sha256'
    OR l->>'politica_ref' IS DISTINCT FROM p_origen->>'politica_ref'
    OR l->>'version_politica' IS DISTINCT FROM p_origen->>'version_politica'
    OR l->>'huella_politica_sha256' IS DISTINCT FROM p_origen->>'huella_politica_sha256'
    OR encode(sha256(decode(l->>'bolsa_canonica','base64')),'hex')
       IS DISTINCT FROM p_origen->>'huella_bolsa_sha256'
    OR encode(sha256(decode(l->>'instantanea_canonica','base64')),'hex')
       IS DISTINCT FROM p_origen->>'huella_instantanea_sha256' THEN
  RAISE EXCEPTION 'B117: captura CT divergente' USING ERRCODE='22023';
 END IF;
 corte:=(l->>'corte')::timestamptz;
 IF corte>p_publicada OR p_publicada>clock_timestamp()+interval '1 minute'
    OR p_publicada>corte+interval '5 minutes' THEN
  RAISE EXCEPTION 'B117: captura CT fuera de vigencia' USING ERRCODE='23503';
 END IF;
 SELECT count(*) INTO cuantos
 FROM vec_bolsa_llamamientos.constitucion c
 JOIN vec_bolsa_llamamientos.bolsa_constituida b ON b.bolsa_ref=c.bolsa_ref
  AND b.version=c.version_bolsa AND b.huella_bolsa_sha256=c.huella_bolsa_sha256
 JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa i ON i.instantanea_ref=c.instantanea_ref
  AND i.version=c.version_instantanea AND i.huella_instantanea_sha256=c.huella_instantanea_sha256
 JOIN vec_bolsa_llamamientos.politica_orden_bolsa pol ON pol.politica_ref=p_origen->>'politica_ref'
  AND pol.version=(p_origen->>'version_politica')::bigint AND pol.bolsa_ref=c.bolsa_ref
 WHERE c.acta_ref=p_origen->>'acta_ref' AND c.categoria_ref=p_origen->>'categoria_ref'
   AND c.bolsa_ref=p_bolsa AND c.version_bolsa=(p_origen->>'version_bolsa')::bigint
   AND c.huella_bolsa_sha256=p_origen->>'huella_bolsa_sha256'
   AND c.instantanea_ref=p_origen->>'instantanea_ref'
   AND c.version_instantanea=(p_origen->>'version_instantanea')::bigint
   AND c.huella_instantanea_sha256=p_origen->>'huella_instantanea_sha256'
   AND pol.vigente_desde<=corte AND (pol.vigente_hasta IS NULL OR p_publicada<pol.vigente_hasta)
   AND pol.provisional=(l->>'politica_provisional')::boolean
   AND encode(sha256(convert_to(to_jsonb(pol)::text,'UTF8')),'hex')=p_origen->>'huella_politica_sha256'
   AND b.estado='vigente' AND b.vigente_desde<=corte AND b.vigente_desde<=p_publicada
   AND (b.vigente_hasta IS NULL OR p_publicada<b.vigente_hasta)
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.sustitucion_bolsa s
     WHERE s.bolsa_ref_sustituida=c.bolsa_ref AND s.version_sustituida=c.version_bolsa
       AND s.huella_sustituida_sha256=c.huella_bolsa_sha256);
 IF cuantos<>1 THEN RAISE EXCEPTION 'B117: constitución CT no vigente' USING ERRCODE='23503'; END IF;
 SELECT count(*) INTO cuantos FROM jsonb_array_elements(l->'entradas') WITH ORDINALITY x(valor,n)
 JOIN vec_bolsa_llamamientos.constitucion_entrada e
  ON e.instantanea_ref=p_origen->>'instantanea_ref'
  AND e.version_instantanea=(p_origen->>'version_instantanea')::bigint
  AND e.orden=x.n AND e.participacion_ref=x.valor->>'participacion_ref'
 JOIN vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(p_bolsa,corte) o
  ON o.participacion_ref=e.participacion_ref AND o.orden_acta=e.orden
 LEFT JOIN vec_bolsa_llamamientos.vinculo_candidato vc
  ON vc.participacion_ref=e.participacion_ref AND vc.acta_ref=p_origen->>'acta_ref'
 WHERE x.valor->>'orden_acta'=e.orden::text
   AND x.valor->>'orden_vigente' IS NOT DISTINCT FROM o.orden_vigente::text
   AND x.valor->>'situacion' IS NOT DISTINCT FROM o.situacion
   AND x.valor->>'razon' IS NOT DISTINCT FROM o.razon
   AND x.valor->>'candidato_ref' IS NOT DISTINCT FROM coalesce(vc.candidato_ref,'')
   AND x.valor->>'elegible'=(o.orden_vigente IS NOT NULL)::text
   AND o.politica_ref=p_origen->>'politica_ref'
   AND o.version_politica=(p_origen->>'version_politica')::bigint
   AND o.provisional=(l->>'politica_provisional')::boolean;
 IF cuantos<>jsonb_array_length(l->'entradas') THEN
  RAISE EXCEPTION 'B117: orden CT divergente' USING ERRCODE='23503';
 END IF;
 SELECT count(*) INTO elegibles FROM jsonb_array_elements(l->'entradas') e
 WHERE e->>'orden_vigente' IS NOT NULL;
 IF elegibles<p_numero OR elegibles<1 THEN
  RAISE EXCEPTION 'B117: plazas superiores a posiciones elegibles en la captura' USING ERRCODE='23505';
 END IF;
 -- La captura conserva la decisión histórica; para una oferta nueva también
 -- se exige cobertura suficiente en el instante firmado de publicación.
 SELECT count(*) INTO actuales
 FROM vec_bolsa_llamamientos.constitucion_entrada e
 JOIN vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(p_bolsa,p_publicada) o
   ON o.participacion_ref=e.participacion_ref AND o.orden_acta=e.orden
 WHERE e.instantanea_ref=p_origen->>'instantanea_ref'
   AND e.version_instantanea=(p_origen->>'version_instantanea')::bigint
   AND o.orden_vigente IS NOT NULL
   AND o.politica_ref=p_origen->>'politica_ref'
   AND o.version_politica=(p_origen->>'version_politica')::bigint
   AND o.provisional=(l->>'politica_provisional')::boolean;
 IF actuales<p_numero THEN
  RAISE EXCEPTION 'B117: plazas superiores a posiciones elegibles al publicar' USING ERRCODE='23505';
 END IF;
 RETURN elegibles;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.validar_origen_oferta_ct_v1(jsonb,text,integer,timestamptz) FROM PUBLIC;
-- La consulta de una sola operación consume V3 fresca antes de leer filas.
-- No depende de la política actual: permite recuperar la versión histórica.
CREATE FUNCTION vec_bolsa_llamamientos.recuperar_oferta_operacion_v1(
 p_bolsa text,p_actor text,p_clave text,p_datos jsonb,p_numero_plazas integer,p_origen jsonb,p_notificacion jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
 p_unidad text,p_ambito text)
RETURNS TABLE(oferta jsonb,encontrada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE h text;ctx text;d jsonb;c record;o record;origen record;plazas record;recibo text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER') IS TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_bolsa IS NULL OR position(chr(31) in p_bolsa)>0 OR position(chr(31) in p_clave)>0 OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR p_unidad IS NULL OR p_unidad !~ '^[A-Za-z0-9:_-]+$'
    OR p_ambito IS NULL OR p_ambito !~ '^[A-Za-z0-9:_-]+$' THEN
  RAISE EXCEPTION 'B117: recuperación denegada' USING ERRCODE='42501';
 END IF;
 h:=vec_bolsa_llamamientos.huella_operacion_oferta_v5(
  'vec.bolsa.oferta.recuperar.v1',p_bolsa,p_clave,p_actor,p_numero_plazas,p_datos,p_origen,p_notificacion,NULL,NULL,NULL);
 ctx:=encode(sha256(convert_to(
  '{"ambitos":{"ambito_ref":"'||p_ambito||'","unidad_ref":"'||p_unidad||
  '"},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B117: decisión ilegible' USING ERRCODE='42501'; END;
 IF d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM ctx
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_llamamientos_bolsa'
    OR d->>'recurso_ref' IS DISTINCT FROM p_bolsa
    OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida' THEN
  RAISE EXCEPTION 'B117: recuperación no ligada a la intención' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT c FROM vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF c.efecto_ref IS DISTINCT FROM p_bolsa OR c.huella_efecto_sha256 IS DISTINCT FROM ctx
    OR c.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'B117: consumo de recuperación divergente' USING ERRCODE='42501';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:oferta:'||p_bolsa||':'||p_clave,0));
 SELECT * INTO o FROM vec_bolsa_llamamientos.oferta_publicada x
  WHERE x.bolsa_ref=p_bolsa AND x.clave_idempotencia=p_clave;
 IF NOT FOUND THEN
  RETURN QUERY SELECT NULL::jsonb,false;
  RETURN;
 END IF;
 recibo:='recibo:oferta:'||encode(sha256(convert_to(p_bolsa||chr(31)||p_clave,'UTF8')),'hex');
 SELECT * INTO plazas FROM vec_bolsa_llamamientos.plazas_oferta p WHERE p.oferta_ref=o.oferta_ref;
 SELECT * INTO origen FROM vec_bolsa_llamamientos.origen_oferta_ct_v1 x WHERE x.oferta_ref=o.oferta_ref;
 IF o.actor_ref IS DISTINCT FROM p_actor OR o.recibo_ref IS DISTINCT FROM recibo
    OR o.datos IS DISTINCT FROM p_datos
    OR (p_notificacion IS NULL AND (o.plazo->>'inicio' IS DISTINCT FROM 'publicacion'
       OR o.plazo ? 'notificacion'))
    OR (p_notificacion IS NOT NULL AND o.plazo->'notificacion' IS DISTINCT FROM p_notificacion)
    OR plazas.numero_plazas IS DISTINCT FROM p_numero_plazas
    OR (p_origen IS NULL) IS DISTINCT FROM (origen.oferta_ref IS NULL)
    OR (p_origen IS NOT NULL AND (origen.origen_canonico IS DISTINCT FROM p_origen
      OR origen.recibo_ref IS DISTINCT FROM o.recibo_ref
      OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.origen_oferta_ct_outbox_v1 ev
        WHERE ev.evento_ref=origen.evento_ref AND ev.oferta_ref=o.oferta_ref))) THEN
  RAISE EXCEPTION 'B117: clave reutilizada con otra oferta' USING ERRCODE='VBO01';
 END IF;
 RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(o.oferta_ref,clock_timestamp()),true;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.recuperar_oferta_operacion_v1(text,text,text,jsonb,integer,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.recuperar_oferta_operacion_v1(text,text,text,jsonb,integer,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text) TO vec_bolsa_llamamientos_ejecutor;
CREATE FUNCTION vec_bolsa_llamamientos.publicar_oferta_v5(
 p_oferta text,p_recibo text,p_bolsa text,p_actor text,p_clave text,p_datos jsonb,p_plazo jsonb,
 p_publicada timestamptz,p_vence timestamptz,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_unidad text,p_ambito text,p_numero_plazas integer,
 p_origen jsonb)
RETURNS TABLE(oferta jsonb,reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE h text;ctx text;d jsonb;c record;previa record;plazas record;origen_previo record;
 v_ahora timestamptz;v_recibo text;v_evento text;v_elegibles integer;v_comando text;v_version bigint;
BEGIN
 -- El contrato histórico B71 mantiene los mismos bytes, columnas y recibos.
 IF p_plazo->>'inicio'='notificacion' AND p_origen IS NULL THEN
  RETURN QUERY SELECT x.oferta,x.reutilizada FROM vec_bolsa_llamamientos.publicar_oferta_v4(
   p_oferta,p_recibo,p_bolsa,p_actor,p_clave,p_datos,p_plazo-'inicio',p_publicada,p_vence,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
   p_payload,p_sobre,p_evidencia,p_raiz,p_unidad,p_ambito,p_numero_plazas) x;
  RETURN;
 END IF;
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER') IS TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_plazo->>'inicio' IS DISTINCT FROM 'publicacion'
    OR p_plazo ? 'notificacion'
    OR p_oferta !~ '^oferta:[0-9a-f]{64}$'
    OR p_recibo !~ '^recibo:oferta:[0-9a-f]{64}$'
    OR p_bolsa IS NULL OR octet_length(p_bolsa) NOT BETWEEN 1 AND 256
    OR position(chr(31) in p_bolsa)>0 OR position(chr(31) in p_clave)>0
    OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR p_datos IS NULL OR jsonb_typeof(p_datos) IS DISTINCT FROM 'object'
    OR NOT (p_datos ?& ARRAY['categoria','centro','fecha_inicio','descripcion'])
    OR EXISTS(SELECT 1 FROM jsonb_object_keys(p_datos) k
       WHERE k NOT IN ('categoria','centro','fecha_inicio','fecha_fin','descripcion'))
    OR EXISTS(SELECT 1 FROM jsonb_each(p_datos) e
       WHERE jsonb_typeof(e.value)<>'string' OR octet_length(e.value#>>'{}') NOT BETWEEN 2 AND 2000
         OR e.value#>>'{}'<>btrim(e.value#>>'{}'))
    OR p_datos->>'fecha_inicio' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
    OR (p_datos ? 'fecha_fin' AND (p_datos->>'fecha_fin' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
       OR p_datos->>'fecha_fin'<p_datos->>'fecha_inicio'))
    OR p_numero_plazas NOT BETWEEN 1 AND 100
    OR p_unidad !~ '^[A-Za-z0-9:_-]+$' OR p_ambito !~ '^[A-Za-z0-9:_-]+$'
    OR p_publicada IS NULL OR p_vence IS NULL OR p_vence<=p_publicada
    OR p_vence>p_publicada+interval '120 days' THEN
  RAISE EXCEPTION 'B117: publicación inválida' USING ERRCODE='22023';
 END IF;
 v_recibo:='recibo:oferta:'||encode(sha256(convert_to(p_bolsa||chr(31)||p_clave,'UTF8')),'hex');
 IF p_recibo IS DISTINCT FROM v_recibo OR p_oferta IS DISTINCT FROM replace(v_recibo,'recibo:oferta:','oferta:') THEN
  RAISE EXCEPTION 'B117: referencias no ligadas a la clave' USING ERRCODE='22023';
 END IF;
 h:=vec_bolsa_llamamientos.huella_operacion_oferta_v5(
  'vec.bolsa.oferta.publicar.v1',p_bolsa,p_clave,p_actor,p_numero_plazas,
  p_datos,p_origen,NULL,p_publicada,p_vence,p_plazo);
 ctx:=encode(sha256(convert_to(
  '{"ambitos":{"ambito_ref":"'||p_ambito||'","unidad_ref":"'||p_unidad||
  '"},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B117: decisión ilegible' USING ERRCODE='42501'; END;
 IF d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM ctx
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_llamamientos_bolsa'
    OR d->>'recurso_ref' IS DISTINCT FROM p_bolsa
    OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida' THEN
  RAISE EXCEPTION 'B117: decisión no ligada a la publicación' USING ERRCODE='42501';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:oferta:'||p_bolsa||':'||p_clave,0));
 SELECT * INTO STRICT c FROM vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF c.efecto_ref IS DISTINCT FROM p_bolsa OR c.huella_efecto_sha256 IS DISTINCT FROM ctx
    OR c.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'B117: consumo de publicación divergente' USING ERRCODE='42501';
 END IF;
 SELECT * INTO previa FROM vec_bolsa_llamamientos.oferta_publicada o
 WHERE o.bolsa_ref=p_bolsa AND o.clave_idempotencia=p_clave;
 IF FOUND THEN
  SELECT * INTO plazas FROM vec_bolsa_llamamientos.plazas_oferta p WHERE p.oferta_ref=previa.oferta_ref;
  SELECT * INTO origen_previo FROM vec_bolsa_llamamientos.origen_oferta_ct_v1 x
   WHERE x.oferta_ref=previa.oferta_ref;
  IF previa.oferta_ref IS DISTINCT FROM p_oferta OR previa.recibo_ref IS DISTINCT FROM p_recibo
     OR previa.actor_ref IS DISTINCT FROM p_actor OR previa.datos IS DISTINCT FROM p_datos
     OR previa.plazo->>'inicio' IS DISTINCT FROM 'publicacion'
     OR plazas.numero_plazas IS DISTINCT FROM p_numero_plazas
     OR (p_origen IS NULL) IS DISTINCT FROM (origen_previo.oferta_ref IS NULL)
     OR (p_origen IS NOT NULL AND (origen_previo.origen_canonico IS DISTINCT FROM p_origen
       OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.origen_oferta_ct_outbox_v1 ev
         WHERE ev.evento_ref=origen_previo.evento_ref AND ev.oferta_ref=previa.oferta_ref))) THEN
   RAISE EXCEPTION 'B117: clave reutilizada con otro cuerpo u origen' USING ERRCODE='VBO01';
  END IF;
  RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(previa.oferta_ref,clock_timestamp()),true;
  RETURN;
 END IF;
 v_ahora:=clock_timestamp();
 IF p_publicada>v_ahora+interval '1 minute' OR p_publicada<v_ahora-interval '5 minutes'
    OR p_vence<=v_ahora OR p_publicada<>date_trunc('microseconds',p_publicada) THEN
  RAISE EXCEPTION 'B117: publicación fuera de plazo' USING ERRCODE='22023';
 END IF;
 IF jsonb_typeof(p_plazo) IS DISTINCT FROM 'object'
    OR (p_plazo->>'unidad'='horas_naturales' AND
      (SELECT count(*) FROM jsonb_object_keys(p_plazo))<>13)
    OR (p_plazo->>'unidad'<>'horas_naturales' AND
      (SELECT count(*) FROM jsonb_object_keys(p_plazo))<>11)
    OR NOT (p_plazo ?& ARRAY['inicio','regla_ref','huella_catalogo','unidad','cantidad',
      'computo','ultimo_dia','ejemplo','calendarios','politica_version','municipio_sede'])
    OR coalesce(p_plazo->>'politica_version','') !~ '^[1-9][0-9]{0,9}$' THEN
  RAISE EXCEPTION 'B117: material de plazo incompleto' USING ERRCODE='22023';
 END IF;
 v_version:=(p_plazo->>'politica_version')::bigint;
 IF p_numero_plazas>1 AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.politica_ofertas_version pv
     WHERE pv.bolsa_ref=p_bolsa AND pv.version=v_version AND jsonb_typeof(pv.politica->'plazas')='object') THEN
  RAISE EXCEPTION 'B117: política sin varias plazas' USING ERRCODE='VBO08';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion co
   JOIN vec_bolsa_llamamientos.bolsa_constituida b
     ON b.bolsa_ref=co.bolsa_ref AND b.version=co.version_bolsa
   WHERE co.bolsa_ref=p_bolsa AND b.estado='vigente' AND b.vigente_desde<=p_publicada
     AND (b.vigente_hasta IS NULL OR p_publicada<b.vigente_hasta)) THEN
  RAISE EXCEPTION 'B117: bolsa no vigente' USING ERRCODE='23503';
 END IF;
 IF p_origen IS NOT NULL THEN
  IF p_origen->>'unidad_ref' IS DISTINCT FROM p_unidad THEN
   RAISE EXCEPTION 'B117: unidad del origen distinta' USING ERRCODE='42501';
  END IF;
  v_elegibles:=vec_bolsa_llamamientos.validar_origen_oferta_ct_v1(p_origen,p_bolsa,p_numero_plazas,p_publicada);
 END IF;
 v_comando:=encode(sha256(convert_to(p_bolsa||chr(31)||p_datos::text,'UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.oferta_publicada(
  oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,
  publicada_en,vence_antes_de,huella_comando_sha256,decision_ref)
 VALUES(p_oferta,p_recibo,p_bolsa,p_actor,p_clave,p_datos,p_plazo,
  p_publicada,p_vence,v_comando,c.decision_ref);
 INSERT INTO vec_bolsa_llamamientos.plazas_oferta(
  oferta_ref,bolsa_ref,numero_plazas,politica_version,registrada_en)
 VALUES(p_oferta,p_bolsa,p_numero_plazas,v_version,clock_timestamp());
 IF p_origen IS NOT NULL THEN
  v_evento:='evento:bolsa:oferta-ct:'||encode(sha256(convert_to(
   (p_origen->>'expediente_ref')||chr(31)||(p_origen->>'clave_ct'),'UTF8')),'hex');
  INSERT INTO vec_bolsa_llamamientos.origen_oferta_ct_v1(
   oferta_ref,recibo_ref,organizacion_ref,unidad_ref,expediente_ref,version_expediente,
   clave_ct,necesidad_ref,acta_ref,categoria_ref,bolsa_ref,version_bolsa,
   huella_bolsa_sha256,instantanea_ref,version_instantanea,huella_instantanea_sha256,
   politica_ref,version_politica,huella_politica_sha256,numero_plazas,elegibles,
   origen_canonico,origen_huella_sha256,decision_ref,auditoria_ref,evento_ref,publicada_en)
  VALUES(p_oferta,p_recibo,p_origen->>'organizacion_ref',p_origen->>'unidad_ref',
   p_origen->>'expediente_ref',(p_origen->>'version_expediente')::bigint,
   p_origen->>'clave_ct',p_origen->>'necesidad_ref',p_origen->>'acta_ref',
   p_origen->>'categoria_ref',p_bolsa,(p_origen->>'version_bolsa')::bigint,
   p_origen->>'huella_bolsa_sha256',p_origen->>'instantanea_ref',
   (p_origen->>'version_instantanea')::bigint,p_origen->>'huella_instantanea_sha256',
   p_origen->>'politica_ref',(p_origen->>'version_politica')::bigint,
   p_origen->>'huella_politica_sha256',p_numero_plazas,v_elegibles,p_origen,
   encode(sha256(convert_to(p_origen::text,'UTF8')),'hex'),
   c.decision_ref,c.auditoria_ref,v_evento,p_publicada);
  INSERT INTO vec_bolsa_llamamientos.origen_oferta_ct_outbox_v1(
   evento_ref,oferta_ref,tipo,estado,creada_en)
  VALUES(v_evento,p_oferta,'bolsa.oferta.ct.origen_confirmado','pendiente',p_publicada);
 END IF;
 RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,clock_timestamp()),false;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.publicar_oferta_v5(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.publicar_oferta_v5(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer,jsonb) TO vec_bolsa_llamamientos_ejecutor;
DO $post$
DECLARE p record;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_trigger t
    WHERE t.tgrelid='vec_bolsa_llamamientos.oferta_publicada'::regclass
      AND t.tgname='oferta_politica_b47' AND NOT t.tgisinternal AND t.tgenabled='O')
    OR EXISTS(SELECT 1 FROM pg_class c
      WHERE c.oid IN ('vec_bolsa_llamamientos.origen_oferta_ct_v1'::regclass,
        'vec_bolsa_llamamientos.origen_oferta_ct_outbox_v1'::regclass)
        AND (c.relowner<>'vec_bolsa_llamamientos_propietario'::regrole
          OR NOT c.relrowsecurity OR NOT c.relforcerowsecurity))
    OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',
      'vec_bolsa_llamamientos.publicar_oferta_v5(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer,jsonb)','EXECUTE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',
      'vec_bolsa_llamamientos.recuperar_oferta_operacion_v1(text,text,text,jsonb,integer,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)','EXECUTE') THEN
  RAISE EXCEPTION 'B117: postimagen de permisos o tablas inválida' USING ERRCODE='55000';
 END IF;
 FOR p IN SELECT oid,proowner,prosecdef,proconfig FROM pg_proc WHERE oid IN (
    'vec_bolsa_llamamientos.publicar_politica_ofertas_v1(text,bigint,jsonb,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
    'vec_bolsa_llamamientos.verificar_politica_oferta_b47()'::regprocedure,
    'vec_bolsa_llamamientos.rechazar_truncado_origen_oferta_ct_v1()'::regprocedure,
    'vec_bolsa_llamamientos.huella_operacion_oferta_v5(text,text,text,text,integer,jsonb,jsonb,jsonb,timestamptz,timestamptz,jsonb)'::regprocedure,
    'vec_bolsa_llamamientos.validar_origen_oferta_ct_v1(jsonb,text,integer,timestamptz)'::regprocedure,
    'vec_bolsa_llamamientos.recuperar_oferta_operacion_v1(text,text,text,jsonb,integer,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)'::regprocedure,
    'vec_bolsa_llamamientos.publicar_oferta_v5(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer,jsonb)'::regprocedure)
 LOOP
  IF p.proowner<>'vec_bolsa_llamamientos_propietario'::regrole OR NOT p.prosecdef
     OR has_function_privilege('public',p.oid,'EXECUTE')
     OR NOT EXISTS(SELECT 1 FROM unnest(p.proconfig) c
        WHERE replace(c,' ','')='search_path=pg_catalog,pg_temp') THEN
   RAISE EXCEPTION 'B117: postimagen de función inválida' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $post$;
COMMIT;
