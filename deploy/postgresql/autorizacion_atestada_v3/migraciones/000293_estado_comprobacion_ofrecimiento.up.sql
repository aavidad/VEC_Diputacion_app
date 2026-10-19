\set ON_ERROR_STOP on
-- AD293 / F1-09: misma captura auditada; estado mínimo solo en consulta actual.
-- Secuencia formal: AD288, B137, AD293; no reconstruye el núcleo ni amplía concesión.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000293',0));
DO $pre$ BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 OR current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR to_regprocedure('vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text,jsonb)') IS NULL
 OR encode(sha256(convert_to(pg_get_functiondef(
  'vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text,jsonb)'::regprocedure),'UTF8')),'hex')
  IS DISTINCT FROM '401b1e9430d2d7c5be462ba733abb6378277b45d2224d1a554768fde93b9afac'
 OR encode(sha256(convert_to(pg_get_functiondef(
  'vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(bytea,bytea,jsonb,jsonb,text,text,text)'::regprocedure),'UTF8')),'hex')
  IS DISTINCT FROM '5fc23b104e5dee751b4cb821919f9cab6ceb24842425bd8f210d0aab8bc85e55'
 OR (SELECT p.proacl::text FROM pg_proc p WHERE p.oid=
  'vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(bytea,bytea,jsonb,jsonb,text,text,text)'::regprocedure)
  IS DISTINCT FROM '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_autorizacion_atestada_v3_registrador_intentos=X/vec_autorizacion_atestada_v3_propietario}'
 THEN RAISE EXCEPTION 'AD293: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(p_contexto bytea, p_vinculo bytea, p_orden jsonb, p_auditoria jsonb, p_expediente text, p_clave text, p_categoria text)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE c jsonb;v jsonb;bolsa text;reserva jsonb;origen jsonb;lectura_raw jsonb;
        lectura record;asiento record;v_ct record;v_contextual boolean;v_estado boolean;v_proyectar boolean;v_estado_actual boolean;
BEGIN
 IF jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR p_orden->>'accion' IS DISTINCT FROM 'bolsa.origen_ct.consultar'
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'gestionar_contratacion_temporal'
 OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_orden->>'proceso' IS DISTINCT FROM 'vec-server'
 OR p_orden->>'resultado' IS DISTINCT FROM 'obtenida'
 OR p_orden->>'motivo_ref' IS DISTINCT FROM 'lectura_origen_ct_correcta'
 OR p_orden->>'unidad_ref' IS NULL OR p_orden->>'ambito_ref' IS NULL
 OR p_orden->>'recurso_ref' IS DISTINCT FROM p_orden->>'bolsa_ref'
 OR p_orden->>'contexto_sha256' IS DISTINCT FROM encode(sha256(p_contexto),'hex')
 THEN RAISE EXCEPTION 'AD261: captura de origen inválida' USING ERRCODE='42501'; END IF;
 v_contextual:=(p_orden->'campos_permitidos') ? 'criterios_ofrecimiento';
 v_estado:=p_orden ? 'comprobacion_contexto';
 bolsa:=vec_autorizacion_atestada_v3.resolver_recurso_origen_bolsa_ct_v1(
   p_contexto,p_vinculo,p_expediente,p_clave,p_categoria,p_orden->>'unidad_ref');
 c:=convert_from(p_contexto,'UTF8')::jsonb;
 v:=convert_from(p_vinculo,'UTF8')::jsonb;
 IF p_orden->>'recurso_ref' IS DISTINCT FROM bolsa
 OR p_orden->>'bolsa_ref' IS DISTINCT FROM bolsa
 OR p_orden->>'registro_contexto_ref' IS DISTINCT FROM v->>'registro_contexto_ref'
 OR p_orden->>'procedencia_sha256' IS DISTINCT FROM v->>'manifiesto_procedencia_huella_sha256'
 OR p_orden->>'autenticacion_ref' IS DISTINCT FROM v->>'autenticacion_ref'
 OR p_orden->>'sesion_ref' IS DISTINCT FROM v->>'sesion_ref'
 OR p_orden->>'autenticacion_sha256' IS DISTINCT FROM v->>'autenticacion_huella_sha256'
 OR vec_autorizacion.cotejar_lectura_bolsa_actual_interna_v1(
    p_orden,c->>'principal_ref',c->>'perfil_activo_ref') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD261: autoridad actual denegada' USING ERRCODE='42501'; END IF;
 reserva:=vec_bolsa_llamamientos.recuperar_reserva_llamamiento_ct_v1(p_expediente,p_clave);
 v_proyectar:=v_estado AND v_contextual AND reserva IS NULL;
 IF v_estado THEN
  IF jsonb_typeof(p_orden->'comprobacion_contexto') IS DISTINCT FROM 'object'
  THEN RAISE EXCEPTION 'AD293: contexto inválido' USING ERRCODE='42501'; END IF;
  IF (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(p_orden->'comprobacion_contexto') k)
    IS DISTINCT FROM ARRAY['ambito_ref','categoria_ref','causa_ofrecida','expediente_ref',
     'modalidad_ofrecida','necesidad_ref','organizacion_ref','unidad_ref','version_expediente']
  OR EXISTS(SELECT 1 FROM jsonb_each(p_orden->'comprobacion_contexto') x WHERE
    CASE WHEN x.key='version_expediente' THEN
      jsonb_typeof(x.value)<>'number' OR x.value#>>'{}' !~ '^[1-9][0-9]{0,8}$'
    WHEN x.key='causa_ofrecida' THEN
      jsonb_typeof(x.value)<>'string' OR octet_length(x.value#>>'{}')>256
    ELSE jsonb_typeof(x.value)<>'string' OR octet_length(x.value#>>'{}') NOT BETWEEN 2 AND 256 END)
  OR p_orden#>>'{comprobacion_contexto,expediente_ref}' IS DISTINCT FROM p_expediente
  OR p_orden#>>'{comprobacion_contexto,unidad_ref}' IS DISTINCT FROM p_orden->>'unidad_ref'
  OR p_orden#>>'{comprobacion_contexto,ambito_ref}' IS DISTINCT FROM p_orden->>'ambito_ref'
  OR p_orden#>>'{comprobacion_contexto,categoria_ref}' IS DISTINCT FROM p_categoria
  THEN RAISE EXCEPTION 'AD293: ámbito de comprobación ajeno' USING ERRCODE='42501'; END IF;
  IF reserva IS NOT NULL AND (
   p_orden#>>'{comprobacion_contexto,organizacion_ref}' IS DISTINCT FROM reserva#>>'{origen_ct,organizacion_ref}'
   OR p_orden#>>'{comprobacion_contexto,necesidad_ref}' IS DISTINCT FROM reserva#>>'{origen_ct,necesidad_ref}')
  THEN RAISE EXCEPTION 'AD293: origen histórico ajeno' USING ERRCODE='42501'; END IF;
  IF reserva IS NULL AND NOT v_contextual
  THEN RAISE EXCEPTION 'AD293: estado fuera de captura contextual' USING ERRCODE='42501'; END IF;
 END IF;
 -- La lectura histórica se proyecta una sola vez. Los campos de reserva
 -- se extraen después, sin devolver origen_ct ni contactos de B7.
 IF reserva IS NOT NULL THEN
  lectura_raw:=reserva->'origen_ct'->'lectura';
  IF NOT v_contextual AND EXISTS(
   SELECT 1 FROM jsonb_array_elements(lectura_raw->'entradas') e
    WHERE e ? 'comprobacion_ofrecimiento')
  THEN RAISE EXCEPTION 'AD293: snapshot sin concesión contextual' USING ERRCODE='42501'; END IF;
  IF lectura_raw ? 'criterios_ofrecimiento' AND NOT v_contextual THEN
   RAISE EXCEPTION 'AD276: snapshot sin concesión específica' USING ERRCODE='42501'; END IF;
  IF reserva->'origen_ct'->>'expediente_ref' IS DISTINCT FROM p_expediente
  OR reserva->'origen_ct'->>'clave_ct' IS DISTINCT FROM p_clave
  OR reserva->'origen_ct'->>'unidad_ref' IS DISTINCT FROM p_orden->>'unidad_ref'
  OR reserva->'origen_ct'->>'bolsa_ref' IS DISTINCT FROM bolsa
  OR reserva->>'bolsa_ref' IS DISTINCT FROM bolsa
  OR reserva->'origen_ct'->>'llamamiento_ref' IS DISTINCT FROM reserva->>'llamamiento_ref'
  OR reserva->'origen_ct'->>'recibo_ref' IS DISTINCT FROM reserva->>'recibo_ref'
  OR lectura_raw->>'categoria_ref' IS DISTINCT FROM p_categoria
  OR lectura_raw->>'bolsa_ref' IS DISTINCT FROM bolsa
  THEN RAISE EXCEPTION 'AD261: reserva de origen incompatible' USING ERRCODE='42501'; END IF;
 ELSE
  IF v_contextual THEN
   SELECT * INTO STRICT v_ct FROM vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(
    p_expediente,p_categoria,NULL,NULL);
   IF v_proyectar THEN
    IF p_orden#>>'{comprobacion_contexto,modalidad_ofrecida}' IS DISTINCT FROM v_ct.modalidad_clave
    OR p_orden#>>'{comprobacion_contexto,causa_ofrecida}' IS DISTINCT FROM v_ct.causa_clave
    OR p_orden#>>'{comprobacion_contexto,version_expediente}' IS DISTINCT FROM v_ct.version_expediente::text
    THEN RAISE EXCEPTION 'AD293: contexto ajeno' USING ERRCODE='42501'; END IF;
   END IF;
   SELECT * INTO STRICT lectura FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
    p_categoria,v_ct.modalidad_clave,v_ct.causa_clave,
    CASE WHEN v_proyectar THEN p_orden->'comprobacion_contexto' ELSE NULL::jsonb END);
  ELSE
   SELECT l.*,NULL::jsonb AS criterios_ofrecimiento INTO STRICT lectura
   FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_v1(p_categoria) l;
  END IF;
  IF lectura.bolsa_ref IS DISTINCT FROM bolsa
  THEN RAISE EXCEPTION 'AD261: fuente de origen cambió' USING ERRCODE='40001'; END IF;
  lectura_raw:=(to_jsonb(lectura)-'criterios_ofrecimiento')||jsonb_build_object(
    'huella_politica_sha256',encode(sha256(lectura.politica_canonica),'hex'));
  IF lectura.criterios_ofrecimiento IS NOT NULL THEN
   lectura_raw:=lectura_raw||jsonb_build_object(
    'criterios_ofrecimiento',lectura.criterios_ofrecimiento);
  END IF;
 END IF;
 -- Lista blanca estructural: una columna nueva de B110 o una propiedad nueva
 -- en la reserva no se publica de forma implícita.
 IF jsonb_typeof(lectura_raw) IS DISTINCT FROM 'object'
 OR lectura_raw ?& ARRAY['acta_ref','categoria_ref','bolsa_ref','version_bolsa',
   'huella_bolsa_sha256','bolsa_canonica','instantanea_ref','version_instantanea',
   'huella_instantanea_sha256','instantanea_canonica','politica_ref','version_politica',
   'huella_politica_sha256','politica_provisional','corte','entradas'] IS NOT TRUE
 OR jsonb_typeof(lectura_raw->'entradas') IS DISTINCT FROM 'array'
 OR (lectura_raw ? 'criterios_ofrecimiento' AND (
   jsonb_typeof(lectura_raw->'criterios_ofrecimiento') IS DISTINCT FROM 'object'
   OR (SELECT count(*) FROM jsonb_object_keys(lectura_raw->'criterios_ofrecimiento'))<>5
   OR lectura_raw->'criterios_ofrecimiento' ?& ARRAY[
      'version_politica_ofertas','huella_politica_ofertas_sha256',
      'modalidad_ofrecida','causa_ofrecida','configuracion'] IS NOT TRUE
   OR lectura_raw#>>'{criterios_ofrecimiento,huella_politica_ofertas_sha256}' !~ '^[0-9a-f]{64}$'
   OR vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(
       lectura_raw#>'{criterios_ofrecimiento,configuracion}') IS NOT TRUE))
 THEN RAISE EXCEPTION 'AD261: origen incompleto' USING ERRCODE='42501'; END IF;
 v_estado_actual:=v_proyectar AND (lectura_raw ? 'criterios_ofrecimiento');
 IF v_proyectar AND NOT v_estado_actual AND EXISTS(
  SELECT 1 FROM jsonb_array_elements(lectura_raw->'entradas') e
   WHERE e ? 'comprobacion_ofrecimiento')
 THEN RAISE EXCEPTION 'AD293: estado sin política publicada' USING ERRCODE='42501'; END IF;
 IF v_estado AND EXISTS (
  SELECT 1 FROM jsonb_array_elements(lectura_raw->'entradas') e
  WHERE (v_estado_actual OR e ? 'comprobacion_ofrecimiento') AND (
   jsonb_typeof(e->'comprobacion_ofrecimiento') IS DISTINCT FROM 'object'
   OR NOT (
    ((SELECT count(*) FROM jsonb_object_keys(e->'comprobacion_ofrecimiento'))=1
      AND e->'comprobacion_ofrecimiento'='{"version_esperada":0}'::jsonb)
    OR
    ((SELECT count(*) FROM jsonb_object_keys(e->'comprobacion_ofrecimiento'))=8
      AND e->'comprobacion_ofrecimiento' ?& ARRAY[
       'comprobacion_ref','resultado','motivo_clave','version_politica_ofertas',
       'version_esperada','registrada_en','recibo_ref','vigente']
      AND e#>>'{comprobacion_ofrecimiento,comprobacion_ref}' ~ '^comprobacion:[0-9a-f]{64}$'
      AND e#>>'{comprobacion_ofrecimiento,recibo_ref}' =
       'recibo:'||e#>>'{comprobacion_ofrecimiento,comprobacion_ref}'
      AND ((e#>>'{comprobacion_ofrecimiento,resultado}'='no_aplicable'
        AND e#>>'{comprobacion_ofrecimiento,motivo_clave}'='sin_supuesto_articulo_9')
       OR (e#>>'{comprobacion_ofrecimiento,resultado}'='pendiente'
        AND e#>>'{comprobacion_ofrecimiento,motivo_clave}' IN (
         'antecedentes_incompletos','criterios_invalidos',
         'guarda_contratacion_pendiente','descuento_pendiente_criterio_rrhh')))
      AND jsonb_typeof(e#>'{comprobacion_ofrecimiento,version_politica_ofertas}')='number'
      AND e#>>'{comprobacion_ofrecimiento,version_politica_ofertas}' ~ '^[1-9][0-9]{0,8}$'
      AND jsonb_typeof(e#>'{comprobacion_ofrecimiento,version_esperada}')='number'
      AND e#>>'{comprobacion_ofrecimiento,version_esperada}' ~ '^[1-9][0-9]{0,8}$'
      AND jsonb_typeof(e#>'{comprobacion_ofrecimiento,registrada_en}')='string'
      AND jsonb_typeof(e#>'{comprobacion_ofrecimiento,vigente}')='boolean')))
 ) THEN RAISE EXCEPTION 'AD293: estado incompatible' USING ERRCODE='42501'; END IF;
 IF NOT v_estado AND EXISTS (
  SELECT 1 FROM jsonb_array_elements(lectura_raw->'entradas') e
   WHERE e ? 'comprobacion_ofrecimiento')
 THEN RAISE EXCEPTION 'AD293: estado fuera de captura' USING ERRCODE='42501'; END IF;
 origen:=jsonb_build_object(
  'acta_ref',lectura_raw->'acta_ref',
  'categoria_ref',lectura_raw->'categoria_ref',
  'bolsa_ref',lectura_raw->'bolsa_ref',
  'version_bolsa',lectura_raw->'version_bolsa',
  'huella_bolsa_sha256',lectura_raw->'huella_bolsa_sha256',
  'bolsa_canonica',lectura_raw->'bolsa_canonica',
  'instantanea_ref',lectura_raw->'instantanea_ref',
  'version_instantanea',lectura_raw->'version_instantanea',
  'huella_instantanea_sha256',lectura_raw->'huella_instantanea_sha256',
  'instantanea_canonica',lectura_raw->'instantanea_canonica',
  'politica_ref',lectura_raw->'politica_ref',
  'version_politica',lectura_raw->'version_politica',
  'huella_politica_sha256',lectura_raw->'huella_politica_sha256',
  'politica_provisional',lectura_raw->'politica_provisional',
  'corte',lectura_raw->'corte',
  'entradas',(SELECT jsonb_agg(jsonb_build_object(
   'orden_acta',x.valor->'orden_acta',
   'orden_vigente',x.valor->'orden_vigente',
   'participacion_ref',x.valor->'participacion_ref',
   'candidato_ref',x.valor->'candidato_ref',
   'situacion',x.valor->'situacion',
   'razon',x.valor->'razon',
   'elegible',x.valor->'orden_vigente'<>'null'::jsonb)
   || CASE WHEN v_contextual AND x.valor ? 'eventos_relacion' THEN
      jsonb_build_object('eventos_relacion',x.valor->'eventos_relacion')
      ELSE '{}'::jsonb END
   || CASE WHEN v_estado_actual OR (v_estado AND reserva IS NOT NULL AND x.valor ? 'comprobacion_ofrecimiento') THEN
      jsonb_build_object('comprobacion_ofrecimiento',x.valor->'comprobacion_ofrecimiento')
      ELSE '{}'::jsonb END ORDER BY x.n)
   FROM jsonb_array_elements(lectura_raw->'entradas') WITH ORDINALITY x(valor,n)));
 IF lectura_raw ? 'criterios_ofrecimiento' THEN
  origen:=origen||jsonb_build_object('criterios_ofrecimiento',
   lectura_raw->'criterios_ofrecimiento');
 END IF;
 -- AD259 admite únicamente este éxito nominal. La proyección a la cadena
 -- común se hace aquí, tras el cotejo, y se revierte junto con la lectura.
 IF jsonb_typeof(p_auditoria) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_auditoria))<>16
 OR p_auditoria->>'intento_ref' !~ '^intento_[0-9a-f]{32}$'
 OR p_auditoria->>'registro_contexto_ref' IS DISTINCT FROM p_orden->>'registro_contexto_ref'
 OR p_auditoria->>'contexto_sha256' IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR p_auditoria->>'procedencia_sha256' IS DISTINCT FROM p_orden->>'procedencia_sha256'
 OR p_auditoria->>'autenticacion_ref' IS DISTINCT FROM p_orden->>'autenticacion_ref'
 OR p_auditoria->>'sesion_ref' IS DISTINCT FROM p_orden->>'sesion_ref'
 OR p_auditoria->>'autenticacion_sha256' IS DISTINCT FROM p_orden->>'autenticacion_sha256'
 OR p_auditoria->>'accion' IS DISTINCT FROM p_orden->>'accion'
 OR p_auditoria->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR p_auditoria->>'recurso_ref' IS DISTINCT FROM bolsa
 OR p_auditoria->>'finalidad_ref' IS DISTINCT FROM p_orden->>'finalidad_ref'
 OR p_auditoria->>'resultado' IS DISTINCT FROM 'exito'
 OR p_auditoria->>'proceso' IS DISTINCT FROM 'vec-rrhh'
 OR p_auditoria->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_auditoria->>'motivo_ref' IS NULL
 OR p_auditoria->>'correlacion_ref' IS NULL
 THEN RAISE EXCEPTION 'AD261: asiento de origen incompatible' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT asiento FROM vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(
  p_contexto,p_vinculo,p_auditoria);
 IF reserva IS NOT NULL THEN
  IF jsonb_typeof(reserva->'configuracion') IS DISTINCT FROM 'object'
   OR (SELECT count(*) FROM jsonb_object_keys(reserva->'configuracion'))<>10
   OR (reserva->'configuracion') ?& ARRAY['referencia','descripcion','categoria',
       'centro','modalidad','fecha_inicio','plazo','plantilla_version','asunto','cuerpo'] IS NOT TRUE
   OR EXISTS(SELECT 1 FROM jsonb_each(reserva->'configuracion') x
       WHERE jsonb_typeof(x.value) IS DISTINCT FROM 'string'
       OR octet_length(x.value#>>'{}') NOT BETWEEN 2 AND 4000)
  THEN RAISE EXCEPTION 'AD261: configuración de replay inválida' USING ERRCODE='42501'; END IF;
  reserva:=jsonb_build_object(
   'expediente_ref',reserva->'origen_ct'->>'expediente_ref',
   'clave_ct',reserva->'origen_ct'->>'clave_ct',
   'unidad_ref',reserva->'origen_ct'->>'unidad_ref',
   'participacion_ref',reserva->'origen_ct'->>'participacion_ref',
   'candidato_ref',reserva->'origen_ct'->>'candidato_ref',
   'orden_seleccionado',reserva->'origen_ct'->'orden_seleccionado',
   'configuracion',reserva->'configuracion');
 END IF;
 RETURN jsonb_build_object('origen',origen,'reserva',reserva,
  'acuse',jsonb_build_object('auditoria_ref',asiento.auditoria_ref,
   'secuencia',asiento.secuencia,'huella_sha256',asiento.huella_sha256,
   'correlacion_ref',asiento.correlacion_ref,'registrada_en',asiento.registrada_en));
EXCEPTION WHEN data_exception OR invalid_text_representation THEN
 RAISE EXCEPTION 'AD261: lectura no disponible' USING ERRCODE='42501';
END $function$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(bytea,bytea,jsonb,jsonb,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(bytea,bytea,jsonb,jsonb,text,text,text)
 TO vec_autorizacion_atestada_v3_registrador_intentos;
DO $post$ DECLARE p record; BEGIN
 SELECT pg_get_userbyid(x.proowner) AS dueno,x.prosecdef,x.provolatile,x.proacl::text AS acl,
   x.proconfig,encode(sha256(convert_to(pg_get_functiondef(x.oid),'UTF8')),'hex') AS huella
 INTO STRICT p FROM pg_proc x WHERE x.oid=
  'vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(bytea,bytea,jsonb,jsonb,text,text,text)'::regprocedure;
 IF p.dueno<>'vec_autorizacion_atestada_v3_propietario' OR p.prosecdef IS NOT TRUE OR p.provolatile<>'v'
 OR p.acl IS DISTINCT FROM '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_autorizacion_atestada_v3_registrador_intentos=X/vec_autorizacion_atestada_v3_propietario}'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']
 OR p.huella IS DISTINCT FROM '81549c426b997d792e0c3e5d56adf9720f25633ceb518a4d989ba120e210de32'
 THEN RAISE EXCEPTION 'AD293: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
