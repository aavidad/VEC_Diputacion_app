\set ON_ERROR_STOP on
-- AD276: proyección auditada del origen contextual y consulta nominal RRHH.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000276',0));
DO $pre$ BEGIN
 IF current_setting('server_version_num')::int<180000
 OR current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR to_regprocedure('vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(text,text,text,bigint)') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text)') IS NULL
 OR to_regclass('vec_autorizacion_atestada_v3.auditoria_consumo_v3') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_exclusiones_llamamiento_v1(bytea,bytea,jsonb)') IS NOT NULL
 OR encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion_atestada_v3.registrar_lectura_bolsa_interna_v1(bytea,bytea,jsonb)'::regprocedure),'UTF8')),'hex')
    <>'bb5536819bf643a67bce72311f3e0d7b55c2835ab4c6bfec128a060c218d2005'
 OR (SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid),'UTF8')),'hex')
     FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
      AND c.conname='auditoria_tipo_disjunto_v4')
    <>'df178e53fb5afa2e18efa30e8d535f5d2d1df1316afa9a51b814bd712190a0b3'
 OR (SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid),'UTF8')),'hex')
     FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
      AND c.conname='auditoria_lectura_respuestas_llamamiento_formato_v1')
    <>'5e1a2c36dc62efe3510042791eed08c65ec79b98cc597699cce1a2da43d829a6'
 OR encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(bytea,bytea,jsonb,jsonb,text,text,text)'::regprocedure),'UTF8')),'hex')
    <>'3b922884bde2cfc20d55a351117cc0bcd5fc03f677dc9596dacce85354bbec19'
 THEN RAISE EXCEPTION 'AD276: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- El tipo nuevo ocupa una rama positiva propia. Se conserva la expresión
-- instalada, de 80 KB, y se añade una sola alternativa de lectura nominal.
DO $guardas$ DECLARE anterior text;nueva text;formato text;
 marca text:='''lectura_respuestas_llamamiento''::text, ''lectura_origen_datos_contacto''::text';
BEGIN
 SELECT pg_get_constraintdef(c.oid) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
   AND c.conname='auditoria_tipo_disjunto_v4';
 IF (length(anterior)-length(replace(anterior,marca,'')))<>length(marca)
 THEN RAISE EXCEPTION 'AD276: marca disjunta ambigua' USING ERRCODE='55000'; END IF;
 nueva:=replace(anterior,marca,
  '''lectura_respuestas_llamamiento''::text, ''lectura_exclusiones_llamamiento''::text, ''lectura_origen_datos_contacto''::text');
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v4';
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||nueva;
 SELECT pg_get_constraintdef(c.oid) INTO STRICT formato FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
   AND c.conname='auditoria_lectura_respuestas_llamamiento_formato_v1';
 formato:=replace(formato,'lectura_respuestas_llamamiento','lectura_exclusiones_llamamiento');
 formato:=replace(formato,'bolsa.rrhh.respuestas_llamamiento.consultar',
  'bolsa.rrhh.exclusiones_llamamiento.consultar');
 formato:=replace(formato,'lectura_respuestas_llamamiento_correcta',
  'lectura_exclusiones_llamamiento_correcta');
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 '
  ||'ADD CONSTRAINT auditoria_lectura_exclusiones_llamamiento_formato_v1 '||formato;
END $guardas$;

-- AD261 conservada con lectura contextual y snapshot opcional.
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
        lectura record;asiento record;v_ct record;
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
 -- La lectura histórica se proyecta una sola vez. Los campos de reserva
 -- se extraen después, sin devolver origen_ct ni contactos de B7.
 IF reserva IS NOT NULL THEN
  lectura_raw:=reserva->'origen_ct'->'lectura';
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
  SELECT * INTO STRICT v_ct FROM vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(
   p_expediente,p_categoria,NULL,NULL);
  SELECT * INTO STRICT lectura FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
   p_categoria,v_ct.modalidad_clave);
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
   OR (SELECT count(*) FROM jsonb_object_keys(lectura_raw->'criterios_ofrecimiento'))<>4
   OR lectura_raw->'criterios_ofrecimiento' ?& ARRAY[
      'version_politica_ofertas','huella_politica_ofertas_sha256',
      'modalidad_ofrecida','configuracion'] IS NOT TRUE
   OR lectura_raw#>>'{criterios_ofrecimiento,huella_politica_ofertas_sha256}' !~ '^[0-9a-f]{64}$'
   OR vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(
       lectura_raw#>'{criterios_ofrecimiento,configuracion}') IS NOT TRUE))
 THEN RAISE EXCEPTION 'AD261: origen incompleto' USING ERRCODE='42501'; END IF;
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
   'elegible',x.valor->'orden_vigente'<>'null'::jsonb) ORDER BY x.n)
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
END $function$


