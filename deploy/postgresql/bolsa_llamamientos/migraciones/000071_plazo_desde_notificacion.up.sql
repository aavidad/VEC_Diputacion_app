\set ON_ERROR_STOP on
-- B71: la notificacion externa declarada por RRHH inicia el plazo versionado.
-- La referencia opaca y la huella del correo se conservan en el plazo inmutable.
-- La publicacion sigue teniendo su propio instante. B28/B54/B58 quedan intactas.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000071',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_bolsa_llamamientos.publicar_oferta_v3(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.verificar_politica_oferta_b47()') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.plazas_oferta') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.oferta_publicada') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.publicar_politica_ofertas_v1(text,bigint,jsonb,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.publicar_oferta_v4(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer)') IS NOT NULL
 THEN RAISE EXCEPTION 'B71: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Mantiene la publicación B58, el consumo V3, CAS, historia y outbox.
-- Una nueva versión debe declarar el inicio; los replays históricos conservan
-- sus cuatro claves originales y su recibo.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.publicar_politica_ofertas_v1(
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
        AND p_politica#>>'{plazo,inicio}' IS DISTINCT FROM 'notificacion')
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
 -- Las versiones anteriores pueden repetirse, pero toda política nueva declara
 -- en su propia versión que el plazo empieza en la notificación.
 IF p_politica#>>'{plazo,inicio}' IS DISTINCT FROM 'notificacion' THEN
  RAISE EXCEPTION 'B71: inicio de plazo no configurado' USING ERRCODE='22023'; END IF;
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


CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.verificar_politica_oferta_b47()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
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
    OR v.politica#>>'{plazo,inicio}' IS DISTINCT FROM 'notificacion'
    OR NEW.plazo->>'unidad' IS DISTINCT FROM v.politica#>>'{plazo,unidad}'
    OR NEW.plazo->>'cantidad' IS DISTINCT FROM v.politica#>>'{plazo,cantidad}'
    OR NEW.plazo->>'computo' IS DISTINCT FROM v.politica#>>'{plazo,computo}'
    OR NEW.plazo->>'municipio_sede' IS DISTINCT FROM v.politica#>>'{plazo,municipio_sede}'
    OR NEW.plazo->>'ejemplo' IS DISTINCT FROM 'true'
    OR (v.politica#>>'{plazo,unidad}'='horas_naturales' AND (
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
    OR (v.politica#>>'{plazo,unidad}'<>'horas_naturales' AND (
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
    OR jsonb_typeof(NEW.plazo->'calendarios') IS DISTINCT FROM 'array'
    OR jsonb_array_length(NEW.plazo->'calendarios') NOT BETWEEN 1 AND 16
    OR EXISTS(SELECT 1 FROM jsonb_array_elements(NEW.plazo->'calendarios') c
              WHERE jsonb_typeof(c) IS DISTINCT FROM 'string' OR octet_length(c#>>'{}') NOT BETWEEN 1 AND 256)
 THEN RAISE EXCEPTION 'B47: oferta sin política de ejemplo vigente' USING ERRCODE='VBP02'; END IF;
 RETURN NEW;
END $f$;


CREATE FUNCTION vec_bolsa_llamamientos.publicar_oferta_v4(
 p_oferta text,p_recibo text,p_bolsa text,p_actor text,p_clave text,p_datos jsonb,p_plazo jsonb,
 p_publicada timestamptz,p_vence timestamptz,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_unidad text,p_ambito text,p_numero_plazas integer)
RETURNS TABLE(oferta jsonb,reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE d jsonb; v_calendarios text; v_material text; v_contexto text; x record; previa record; v_version bigint; consumo record; v_ahora timestamptz;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR p_publicada IS NULL OR p_vence IS NULL
    OR p_bolsa IS NULL OR p_plazo IS NULL
    OR p_numero_plazas IS NULL OR p_numero_plazas NOT BETWEEN 1 AND 100
    OR p_unidad IS NULL OR p_unidad !~ '^[A-Za-z0-9:_-]+$' OR octet_length(p_unidad) NOT BETWEEN 1 AND 256
    OR p_ambito IS NULL OR p_ambito !~ '^[A-Za-z0-9:_-]+$' OR octet_length(p_ambito) NOT BETWEEN 1 AND 256
    OR jsonb_typeof(p_plazo) IS DISTINCT FROM 'object'
    OR (p_plazo->>'unidad'='horas_naturales' AND
        (SELECT count(*) FROM jsonb_object_keys(p_plazo))<>13)
    OR (p_plazo->>'unidad'<>'horas_naturales' AND
        (SELECT count(*) FROM jsonb_object_keys(p_plazo))<>11)
    OR NOT (p_plazo ?& ARRAY['regla_ref','huella_catalogo','unidad','cantidad','computo',
                              'ultimo_dia','ejemplo','calendarios','politica_version','municipio_sede'])
    OR jsonb_typeof(p_plazo->'calendarios') IS DISTINCT FROM 'array'
    OR jsonb_array_length(p_plazo->'calendarios') NOT BETWEEN 1 AND 16
    OR EXISTS(SELECT 1 FROM jsonb_array_elements(p_plazo->'calendarios') c
              WHERE jsonb_typeof(c) IS DISTINCT FROM 'string' OR c#>>'{}' !~ '^[A-Za-z0-9:_-]+$')
    OR p_plazo->>'regla_ref' IS NULL OR p_plazo->>'huella_catalogo' IS NULL
    OR p_plazo->>'unidad' IS NULL OR p_plazo->>'cantidad' IS NULL
    OR p_plazo->>'computo' IS NULL OR p_plazo->>'municipio_sede' IS NULL
    OR p_plazo->>'ultimo_dia' IS NULL OR coalesce(p_plazo->>'politica_version','') !~ '^[1-9][0-9]{0,9}$'
    OR jsonb_typeof(p_plazo->'notificacion') IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_plazo->'notificacion'))<>4
    OR NOT (p_plazo->'notificacion' ?& ARRAY['notificada_en','referencia_correo','huella_correo_sha256','fuente'])
    OR coalesce(p_plazo#>>'{notificacion,notificada_en}','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
    OR coalesce(p_plazo#>>'{notificacion,referencia_correo}','') !~ '^[A-Za-z0-9][A-Za-z0-9:_.-]{0,255}$'
      OR (p_plazo#>>'{notificacion,referencia_correo}' !~ '^[a-z_]+(:[a-z_]+)*:[0-9a-f]{64}$'
          AND p_plazo#>>'{notificacion,referencia_correo}' ~* '(([0-9][._:/#-]?){8}|[XYZ][._:/#-]?([0-9][._:/#-]?){7})[A-Z]')
      OR p_plazo#>>'{notificacion,referencia_correo}' ~* '(^|[._:/#-])(dni|nie|nif|pasaporte|passport)([._:/#-]|$)'
    OR coalesce(p_plazo#>>'{notificacion,huella_correo_sha256}','') !~ '^[0-9a-f]{64}$'
    OR p_plazo#>>'{notificacion,fuente}' IS DISTINCT FROM 'correo_externo_declarado_rrhh'
    OR (p_plazo->>'unidad'='horas_naturales' AND
        (p_plazo->>'apertura_en' IS DISTINCT FROM p_plazo#>>'{notificacion,notificada_en}'
         OR p_plazo->>'vence_en' IS NULL))
 THEN RAISE EXCEPTION 'B71: material de plazo o notificacion incompleto' USING ERRCODE='22023'; END IF;
 SELECT string_agg(c#>>'{}',chr(30) ORDER BY n) INTO v_calendarios
 FROM jsonb_array_elements(p_plazo->'calendarios') WITH ORDINALITY a(c,n);
 v_material:=encode(sha256(convert_to(array_to_string(ARRAY[
  p_bolsa,
  to_char(p_publicada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  to_char(p_vence AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  p_plazo->>'regla_ref',p_plazo->>'huella_catalogo',p_plazo->>'unidad',
  p_plazo->>'cantidad',p_plazo->>'computo',p_plazo->>'municipio_sede',
  p_plazo->>'ultimo_dia',p_plazo->>'politica_version',v_calendarios
 ] || CASE WHEN p_plazo->>'unidad'='horas_naturales'
           THEN ARRAY[p_plazo->>'apertura_en',p_plazo->>'vence_en']
           ELSE ARRAY[]::text[] END || ARRAY[p_numero_plazas::text,
  p_plazo#>>'{notificacion,notificada_en}',p_plazo#>>'{notificacion,referencia_correo}',
  p_plazo#>>'{notificacion,huella_correo_sha256}',p_plazo#>>'{notificacion,fuente}'],chr(31)),'UTF8')),'hex');
 v_contexto:=encode(sha256(convert_to(
  '{"ambitos":{"ambito_ref":"'||p_ambito||'","unidad_ref":"'||p_unidad||
  '"},"atributos":{"material_sha256":"'||v_material||'"}}','UTF8')),'hex');
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B71: decisión de oferta inválida' USING ERRCODE='42501'; END;
 IF d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto THEN
  RAISE EXCEPTION 'B71: plazo o notificacion distintos de lo autorizado' USING ERRCODE='42501';
 END IF;
 -- El replay se resuelve antes del vencimiento: la preimagen B28 rechaza
 -- publicaciones antiguas incluso al reintentar una operación ya confirmada.
 -- El instante de publicación y la versión de política pueden variar en la nueva
 -- petición: se devuelve el registro original si el hecho notificado coincide.
 -- Se consume una decisión nueva y exacta antes de devolver el recibo.
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:oferta:'||p_bolsa||':'||p_clave,0));
 SELECT * INTO previa FROM vec_bolsa_llamamientos.oferta_publicada
  WHERE bolsa_ref=p_bolsa AND clave_idempotencia=p_clave;
 v_ahora:=clock_timestamp();
 IF FOUND THEN
  SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
  IF consumo.efecto_ref IS DISTINCT FROM p_bolsa OR consumo.consumo_nuevo IS NOT TRUE
     OR d->>'principal_id' IS DISTINCT FROM p_actor OR d->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1'
     OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'finalidad' IS DISTINCT FROM 'gestion_llamamientos_bolsa'
     OR d->>'recurso_ref' IS DISTINCT FROM p_bolsa OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida' THEN
   RAISE EXCEPTION 'B71: replay no autorizado' USING ERRCODE='42501'; END IF;
  IF previa.oferta_ref IS DISTINCT FROM p_oferta OR previa.recibo_ref IS DISTINCT FROM p_recibo
     OR previa.actor_ref IS DISTINCT FROM p_actor OR previa.datos IS DISTINCT FROM p_datos
     OR previa.plazo->'notificacion' IS DISTINCT FROM p_plazo->'notificacion'
     OR NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.plazas_oferta pp
                     WHERE pp.oferta_ref=previa.oferta_ref AND pp.numero_plazas=p_numero_plazas)
  THEN RAISE EXCEPTION 'B71: clave reutilizada con otra oferta' USING ERRCODE='VBO01'; END IF;
  RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(previa.oferta_ref,v_ahora),true;
  RETURN;
 END IF;
 v_version:=(p_plazo->>'politica_version')::bigint;
 IF p_numero_plazas>1 AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.politica_ofertas_version pv
     WHERE pv.bolsa_ref=p_bolsa AND pv.version=v_version AND jsonb_typeof(pv.politica->'plazas')='object') THEN
  RAISE EXCEPTION 'B71: la política de la bolsa no admite varias plazas' USING ERRCODE='VBO08';
 END IF;
 IF (p_plazo#>>'{notificacion,notificada_en}')::timestamptz>v_ahora
    OR p_vence<=v_ahora THEN
  RAISE EXCEPTION 'B71: notificacion futura o plazo vencido' USING ERRCODE='22023'; END IF;
 SELECT * INTO STRICT x FROM vec_bolsa_llamamientos.publicar_oferta_v1(
  p_oferta,p_recibo,p_bolsa,p_actor,p_clave,p_datos,p_plazo,p_publicada,p_vence,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO previa FROM vec_bolsa_llamamientos.plazas_oferta WHERE oferta_ref=x.oferta->>'oferta_ref';
 IF FOUND THEN
  IF previa.numero_plazas<>p_numero_plazas THEN
   RAISE EXCEPTION 'B58: clave de oferta con otro número de plazas' USING ERRCODE='VBO01'; END IF;
 ELSIF x.reutilizada THEN
  -- Oferta anterior publicada sin fila de plazas: solo equivale a una plaza.
  IF p_numero_plazas<>1 THEN
   RAISE EXCEPTION 'B58: clave de oferta con otro número de plazas' USING ERRCODE='VBO01'; END IF;
 ELSE
  INSERT INTO vec_bolsa_llamamientos.plazas_oferta(oferta_ref,bolsa_ref,numero_plazas,politica_version,registrada_en)
  VALUES(x.oferta->>'oferta_ref',p_bolsa,p_numero_plazas,v_version,clock_timestamp());
 END IF;
 RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(x.oferta->>'oferta_ref',greatest(clock_timestamp(),p_publicada)),x.reutilizada;
END $f$;


REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.publicar_oferta_v4(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.publicar_oferta_v4(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer) TO vec_bolsa_llamamientos_ejecutor;
REVOKE EXECUTE ON FUNCTION vec_bolsa_llamamientos.publicar_oferta_v3(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer) FROM vec_bolsa_llamamientos_ejecutor;
DO $acl$
BEGIN
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.publicar_oferta_v4(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer)','EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.publicar_oferta_v3(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer)','EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
                WHERE p.oid='vec_bolsa_llamamientos.publicar_oferta_v4(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer)'::regprocedure
                  AND a.grantee<>p.proowner AND (a.grantee<>'vec_bolsa_llamamientos_ejecutor'::regrole OR a.is_grantable))
 THEN RAISE EXCEPTION 'B71: ACL incorrecta' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
