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
 OR to_regprocedure('vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text)') IS NULL
 OR to_regclass('vec_autorizacion_atestada_v3.auditoria_consumo_v3') IS NULL
 OR encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.cotejar_lectura_bolsa_actual_interna_v1(jsonb,text,text)'::regprocedure),'UTF8')),'hex')<>'1d063fe250f27707496dd9869419a7aac785e41cc35af4c569e6ef986041ea39'
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
        lectura record;asiento record;v_ct record;v_contextual boolean;
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
   SELECT * INTO STRICT lectura FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
    p_categoria,v_ct.modalidad_clave,v_ct.causa_clave);
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



SET LOCAL ROLE vec_autorizacion_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion.cotejar_lectura_bolsa_actual_interna_v1(p_orden jsonb, p_principal text, p_perfil text)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
AS $function$
DECLARE a record;r record;c record;catalogo record;politicas jsonb;ahora timestamptz(6);
 accion text;finalidad text;tipo text;campos jsonb;patron text;
BEGIN
 accion:=p_orden->>'accion';
 CASE accion
 WHEN 'bolsa.rrhh.exclusiones_llamamiento.consultar' THEN
  finalidad:='seguimiento_contacto_rrhh';tipo:='llamamiento_bolsa';
  campos:='["bolsa_ref","criterios_ofrecimiento","exclusiones[].eventos_relacion","exclusiones[].hechos_relacion","exclusiones[].motivo_clave","exclusiones[].orden_acta","exclusiones[].participacion_ref","exclusiones[].situacion_en_origen","hay_mas","llamamiento_ref","siguiente_orden_acta","total_exclusiones"]'::jsonb;patron:='^llamamiento:[0-9a-f]{64}$';
 WHEN 'bolsa.rrhh.respuestas_llamamiento.consultar' THEN
  finalidad:='seguimiento_contacto_rrhh';tipo:='llamamiento_bolsa';
  campos:='["llamamiento_ref","respuestas[].participacion_ref","respuestas[].respuesta"]'::jsonb;patron:='^llamamiento:[0-9a-f]{64}$';
 WHEN 'bolsa.origen_ct.consultar' THEN
  finalidad:='gestionar_contratacion_temporal';tipo:='bolsa_constituida';
  -- Dos proyecciones cerradas. La versión del rol debe conceder exactamente
  -- los campos solicitados; el catálogo anterior no abre el snapshot nuevo.
  IF p_orden->'campos_permitidos'='["acta_ref","bolsa_canonica","bolsa_ref","categoria_ref","corte","criterios_ofrecimiento","entradas[].candidato_ref","entradas[].elegible","entradas[].eventos_relacion","entradas[].orden_acta","entradas[].orden_vigente","entradas[].participacion_ref","entradas[].razon","entradas[].situacion","huella_bolsa_sha256","huella_instantanea_sha256","huella_politica_sha256","instantanea_canonica","instantanea_ref","politica_provisional","politica_ref","reserva.candidato_ref","reserva.clave_ct","reserva.configuracion.asunto","reserva.configuracion.categoria","reserva.configuracion.centro","reserva.configuracion.cuerpo","reserva.configuracion.descripcion","reserva.configuracion.fecha_inicio","reserva.configuracion.modalidad","reserva.configuracion.plantilla_version","reserva.configuracion.plazo","reserva.configuracion.referencia","reserva.expediente_ref","reserva.orden_seleccionado","reserva.participacion_ref","reserva.unidad_ref","version_bolsa","version_instantanea","version_politica"]'::jsonb THEN
   campos:='["acta_ref","bolsa_canonica","bolsa_ref","categoria_ref","corte","criterios_ofrecimiento","entradas[].candidato_ref","entradas[].elegible","entradas[].eventos_relacion","entradas[].orden_acta","entradas[].orden_vigente","entradas[].participacion_ref","entradas[].razon","entradas[].situacion","huella_bolsa_sha256","huella_instantanea_sha256","huella_politica_sha256","instantanea_canonica","instantanea_ref","politica_provisional","politica_ref","reserva.candidato_ref","reserva.clave_ct","reserva.configuracion.asunto","reserva.configuracion.categoria","reserva.configuracion.centro","reserva.configuracion.cuerpo","reserva.configuracion.descripcion","reserva.configuracion.fecha_inicio","reserva.configuracion.modalidad","reserva.configuracion.plantilla_version","reserva.configuracion.plazo","reserva.configuracion.referencia","reserva.expediente_ref","reserva.orden_seleccionado","reserva.participacion_ref","reserva.unidad_ref","version_bolsa","version_instantanea","version_politica"]'::jsonb;
  ELSE
   campos:='["acta_ref","bolsa_canonica","bolsa_ref","categoria_ref","corte","entradas[].candidato_ref","entradas[].elegible","entradas[].orden_acta","entradas[].orden_vigente","entradas[].participacion_ref","entradas[].razon","entradas[].situacion","huella_bolsa_sha256","huella_instantanea_sha256","huella_politica_sha256","instantanea_canonica","instantanea_ref","politica_provisional","politica_ref","reserva.candidato_ref","reserva.clave_ct","reserva.configuracion.asunto","reserva.configuracion.categoria","reserva.configuracion.centro","reserva.configuracion.cuerpo","reserva.configuracion.descripcion","reserva.configuracion.fecha_inicio","reserva.configuracion.modalidad","reserva.configuracion.plantilla_version","reserva.configuracion.plazo","reserva.configuracion.referencia","reserva.expediente_ref","reserva.orden_seleccionado","reserva.participacion_ref","reserva.unidad_ref","version_bolsa","version_instantanea","version_politica"]'::jsonb;
  END IF;
  patron:='^bolsa:[a-z0-9][a-z0-9._:-]{0,193}$';
 WHEN 'bolsa.datos_contacto_participacion.origen.consultar' THEN
  finalidad:='consulta_origen_datos_contacto_participacion';tipo:='participacion_bolsa';
  campos:='["enmascarados","origen","participacion_ref","registrada_en","sin_contacto","version"]'::jsonb;patron:='^[A-Za-z0-9][A-Za-z0-9:._-]{0,255}[A-Za-z0-9:._-]{0,255}[A-Za-z0-9:._-]?$';
 ELSE RETURN false; END CASE;
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR p_orden->>'accion' IS DISTINCT FROM accion
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM finalidad
 OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_orden->>'recurso_ref' !~ patron
 OR jsonb_typeof(p_orden->'unidad_ref') IS DISTINCT FROM 'string'
 OR jsonb_typeof(p_orden->'ambito_ref') IS DISTINCT FROM 'string'
 OR jsonb_typeof(p_orden->'bolsa_ref') IS DISTINCT FROM 'string'
 OR vec_autorizacion.texto_positivo_valido(p_orden->>'unidad_ref',512) IS NOT TRUE
 OR vec_autorizacion.texto_positivo_valido(p_orden->>'ambito_ref',512) IS NOT TRUE
 OR vec_autorizacion.texto_positivo_valido(p_orden->>'bolsa_ref',512) IS NOT TRUE

 OR p_orden->'campos_permitidos' IS DISTINCT FROM
    campos
 OR jsonb_typeof(p_orden->'politicas_evaluadas') IS DISTINCT FROM 'array'
 THEN RETURN false; END IF;
 ahora:=clock_timestamp();
 IF (p_orden->>'emitida_en')::timestamptz>ahora
 OR (p_orden->>'valida_hasta')::timestamptz<=ahora
 OR (p_orden->>'valida_hasta')::timestamptz>ahora+interval '30 seconds'
 THEN RETURN false; END IF;
 SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil_actual ap
 JOIN vec_autorizacion.asignacion_perfil x ON x.perfil_activo_ref=ap.perfil_activo_ref
  AND x.asignacion_ref=ap.asignacion_ref
 WHERE ap.perfil_activo_ref=p_perfil FOR SHARE OF ap,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO r FROM vec_autorizacion.version_rol x
 WHERE x.version_rol_ref=a.version_rol_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO c FROM vec_autorizacion.control_vigencia_version_rol_actual cp
 JOIN vec_autorizacion.control_vigencia_version_rol x
  ON x.version_rol_ref=cp.version_rol_ref AND x.revision=cp.revision
 WHERE cp.version_rol_ref=r.version_rol_ref FOR SHARE OF cp,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT * INTO catalogo FROM vec_autorizacion.control_catalogo_politicas
 WHERE control_id IS TRUE FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT COALESCE(jsonb_agg(jsonb_build_object('referencia',p.politica_ref,
  'huella_sha256',p.huella_sha256) ORDER BY p.politica_ref),'[]'::jsonb)
 INTO politicas FROM vec_autorizacion.politica_restrictiva_actual pa
 JOIN vec_autorizacion.politica_restrictiva p
  ON p.politica_id=pa.politica_id AND p.politica_ref=pa.politica_ref;
 RETURN a.principal_id IS NOT DISTINCT FROM p_principal
  AND a.perfil_activo_ref IS NOT DISTINCT FROM p_perfil
  AND a.asignacion_ref IS NOT DISTINCT FROM p_orden->>'asignacion_ref'
  AND a.version IS NOT DISTINCT FROM (p_orden->>'asignacion_version')::numeric
  AND a.huella_sha256 IS NOT DISTINCT FROM p_orden->>'asignacion_huella_sha256'
  AND a.documento->>'estado' IS NOT DISTINCT FROM 'activa'
  AND ahora>=(a.documento->>'vigente_desde')::timestamptz
  AND ahora<(a.documento->>'vigente_hasta')::timestamptz
  AND r.version_rol_ref IS NOT DISTINCT FROM p_orden->>'version_rol_ref'
  AND r.version IS NOT DISTINCT FROM (p_orden->>'version_rol_version')::numeric
  AND r.huella_sha256 IS NOT DISTINCT FROM p_orden->>'version_rol_huella_sha256'
  AND r.documento->>'estado' IS NOT DISTINCT FROM 'publicada'
  AND c.version_rol_ref IS NOT DISTINCT FROM p_orden->>'control_version_rol_ref'
  AND c.revision IS NOT DISTINCT FROM (p_orden->>'revision_control_version_rol')::numeric
  AND c.huella_sha256 IS NOT DISTINCT FROM p_orden->>'control_version_rol_huella_sha256'
  AND c.estado IS NOT DISTINCT FROM 'habilitada'
  AND catalogo.revision IS NOT DISTINCT FROM (p_orden->>'revision_catalogo_politicas')::numeric
  AND catalogo.huella_sha256 IS NOT DISTINCT FROM p_orden->>'catalogo_politicas_huella_sha256'
  AND politicas IS NOT DISTINCT FROM p_orden->'politicas_evaluadas'
  AND CASE WHEN jsonb_typeof(a.documento->'ambitos')='array' THEN
   jsonb_array_length(a.documento->'ambitos')=2
   AND vec_autorizacion.ambitos_positivos_validos(a.documento) IS TRUE
   AND a.documento->'ambitos' @> jsonb_build_array(
    jsonb_build_object('clave','unidad_ref',
     'valores',jsonb_build_array(p_orden->>'unidad_ref')),
    jsonb_build_object('clave','ambito_ref',
     'valores',jsonb_build_array(p_orden->>'ambito_ref')))
   ELSE false END
  AND (SELECT count(*) FROM jsonb_array_elements(r.documento->'concesiones') concesion
   WHERE concesion->>'accion'=accion)=1
  AND EXISTS(SELECT 1 FROM jsonb_array_elements(r.documento->'concesiones') concesion
   WHERE concesion->>'accion'=accion
    AND concesion->>'modulo_id'='bolsa'
    AND concesion->>'tipo_recurso'=tipo
    AND concesion->'finalidades'=jsonb_build_array(finalidad)
    AND concesion->'campos_permitidos'=p_orden->'campos_permitidos'
    AND concesion->>'garantia_minima'='alto'
    AND concesion->'obligaciones' IN ('[]'::jsonb,'["auditar"]'::jsonb));
EXCEPTION WHEN data_exception OR invalid_text_representation OR numeric_value_out_of_range THEN
 RETURN false;
END $function$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_bolsa_interna_v1(p_contexto_canonico bytea, p_vinculo_canonico bytea, p_orden jsonb)
 RETURNS TABLE(auditoria_ref text, secuencia numeric, huella_sha256 text, correlacion_ref text, registrada_en timestamp with time zone)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '10s'
AS $function$
DECLARE
 v_contexto jsonb;v_vinculo jsonb;v_sesion record;
 v_claves text[]:=ARRAY[
  'intento_ref','registro_contexto_ref','contexto_sha256','procedencia_sha256',
  'autenticacion_ref','sesion_ref','autenticacion_sha256','accion','modulo_id',
  'recurso_ref','bolsa_ref','unidad_ref','ambito_ref','finalidad_ref',
  'campos_permitidos','resultado','motivo_ref',
  'proceso','canal','correlacion_ref','lectura_revision_permisos',
  'lectura_instantanea_sha256','asignacion_ref','asignacion_version',
  'asignacion_huella_sha256','version_rol_ref','version_rol_version',
  'version_rol_huella_sha256','control_version_rol_ref','revision_control_version_rol',
  'control_version_rol_huella_sha256','revision_catalogo_politicas',
  'catalogo_politicas_huella_sha256','politicas_evaluadas','emitida_en','valida_hasta'];
 v_clave text;v_material bytea;v_material_sha text;v_anterior text;v_secuencia numeric;
 v_instante timestamptz(6);v_huella text;v_ref text;
 v_tipo text;v_dominio text;v_dominio_material text;v_prefijo text;v_accion text;v_finalidad text;v_campos jsonb;v_patron text;v_motivo text;
BEGIN
 v_accion:=p_orden->>'accion';
 CASE v_accion
 WHEN 'bolsa.rrhh.exclusiones_llamamiento.consultar' THEN
  v_tipo:='lectura_exclusiones_llamamiento';v_dominio_material:='vec.auditoria.lectura_exclusiones_llamamiento.v1';v_dominio:='vec.auditoria.eslabon.lectura_exclusiones_llamamiento.v1';v_prefijo:='aud_v3_le_';
  v_finalidad:='seguimiento_contacto_rrhh';v_motivo:='lectura_exclusiones_llamamiento_correcta';
  v_campos:='["bolsa_ref","criterios_ofrecimiento","exclusiones[].eventos_relacion","exclusiones[].hechos_relacion","exclusiones[].motivo_clave","exclusiones[].orden_acta","exclusiones[].participacion_ref","exclusiones[].situacion_en_origen","hay_mas","llamamiento_ref","siguiente_orden_acta","total_exclusiones"]'::jsonb;v_patron:='^llamamiento:[0-9a-f]{64}$';
 WHEN 'bolsa.rrhh.respuestas_llamamiento.consultar' THEN
  v_tipo:='lectura_respuestas_llamamiento';v_dominio_material:='vec.auditoria.lectura_respuestas_llamamiento.v1';v_dominio:='vec.auditoria.eslabon.lectura_respuestas_llamamiento.v1';v_prefijo:='aud_v3_lr_';
  v_finalidad:='seguimiento_contacto_rrhh';v_motivo:='lectura_respuestas_llamamiento_correcta';
  v_campos:='["llamamiento_ref","respuestas[].participacion_ref","respuestas[].respuesta"]'::jsonb;v_patron:='^llamamiento:[0-9a-f]{64}$';
 WHEN 'bolsa.datos_contacto_participacion.origen.consultar' THEN
  v_tipo:='lectura_origen_datos_contacto';v_dominio_material:='vec.auditoria.lectura_origen_datos_contacto.v1';v_dominio:='vec.auditoria.eslabon.lectura_origen_datos_contacto.v1';v_prefijo:='aud_v3_lc_';
  v_finalidad:='consulta_origen_datos_contacto_participacion';v_motivo:='lectura_origen_datos_contacto_correcta';
  v_campos:='["enmascarados","origen","participacion_ref","registrada_en","sin_contacto","version"]'::jsonb;v_patron:='^[A-Za-z0-9][A-Za-z0-9:._-]{0,255}[A-Za-z0-9:._-]{0,255}[A-Za-z0-9:._-]?$';
 ELSE RAISE EXCEPTION 'AD249: operación de lectura no admitida' USING ERRCODE='42501'; END CASE;
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 OR vec_autorizacion_atestada_v3.login_lectura_respuestas_llamamiento_valido_v1() IS NOT TRUE
 OR p_contexto_canonico IS NULL OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
 OR p_vinculo_canonico IS NULL OR octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_orden))<>array_length(v_claves,1)
 OR (p_orden ?& v_claves) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD237: lectura sin emisor acreditado' USING ERRCODE='42501'; END IF;
 FOREACH v_clave IN ARRAY v_claves LOOP
  IF v_clave IN ('campos_permitidos','politicas_evaluadas') THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'array'
   THEN RAISE EXCEPTION 'AD237: orden de lectura inválida' USING ERRCODE='22023'; END IF;
  ELSIF v_clave IN ('asignacion_version','version_rol_version',
                    'revision_control_version_rol','revision_catalogo_politicas') THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'number'
   OR (p_orden->>v_clave) !~ '^[1-9][0-9]{0,19}$'
   THEN RAISE EXCEPTION 'AD237: versión de lectura inválida' USING ERRCODE='22023'; END IF;
  ELSIF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'string'
   OR octet_length(p_orden->>v_clave) NOT BETWEEN 1 AND 512
  THEN RAISE EXCEPTION 'AD237: orden de lectura inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_orden->>'intento_ref' !~ '^lectura_[0-9a-f]{32}$'
 OR p_orden->>'recurso_ref' !~ v_patron
 OR p_orden->>'contexto_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'procedencia_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'autenticacion_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'lectura_revision_permisos' !~ '^[1-9][0-9]{0,19}$'
 OR p_orden->>'lectura_instantanea_sha256' !~ '^[0-9a-f]{64}$'
 OR (p_orden->>'lectura_revision_permisos')::numeric>18446744073709551615::numeric
 OR p_orden->>'accion' IS DISTINCT FROM v_accion
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM v_finalidad
 OR p_orden->>'resultado' IS DISTINCT FROM 'obtenida'
 OR p_orden->>'motivo_ref' IS DISTINCT FROM v_motivo
 OR p_orden->>'proceso' IS DISTINCT FROM 'vec-server'
 OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR EXISTS(SELECT 1 FROM unnest(ARRAY[p_orden->>'bolsa_ref',p_orden->>'unidad_ref',
   p_orden->>'ambito_ref']) ref
   WHERE ref ~ '[[:space:][:cntrl:]]' OR strpos(ref,'*')>0)
 OR p_orden->>'correlacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_orden->'campos_permitidos' IS DISTINCT FROM
    v_campos
 THEN RAISE EXCEPTION 'AD237: contrato de lectura inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  v_contexto:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
  v_vinculo:=convert_from(p_vinculo_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AD237: evidencia de lectura inválida' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(v_contexto) IS DISTINCT FROM 'object'
 OR jsonb_typeof(v_vinculo) IS DISTINCT FROM 'object'
 OR v_contexto->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
 OR v_vinculo->>'esquema' IS DISTINCT FROM 'vec.autenticacion-actor.vinculo.v2.contexto-registrado'
 OR v_vinculo->>'bloque_version' IS DISTINCT FROM '2'
 OR v_vinculo->>'principal_id' IS DISTINCT FROM v_contexto->>'principal_ref'
 OR v_vinculo->>'perfil_activo_ref' IS DISTINCT FROM v_contexto->>'perfil_activo_ref'
 OR v_vinculo->>'cuenta_ref' IS DISTINCT FROM v_contexto->>'cuenta_ref'
 OR v_vinculo->>'metodo_observado' IS DISTINCT FROM v_contexto->>'metodo'
 OR v_vinculo->>'garantia_observada' IS DISTINCT FROM v_contexto->>'garantia'
 OR v_vinculo->>'contexto_actor_ref' IS DISTINCT FROM v_contexto->>'contexto_actor_ref'
 OR v_vinculo->>'contexto_actor_version' IS DISTINCT FROM v_contexto->>'contexto_version'
 OR v_vinculo->>'contexto_actor_cuenta_version' IS DISTINCT FROM v_contexto->>'cuenta_version'
 OR v_vinculo->>'registro_contexto_ref' IS DISTINCT FROM p_orden->>'registro_contexto_ref'
 OR v_vinculo->>'contexto_actor_huella_sha256' IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR v_vinculo->>'manifiesto_procedencia_huella_sha256' IS DISTINCT FROM p_orden->>'procedencia_sha256'
 OR v_vinculo->>'autenticacion_ref' IS DISTINCT FROM p_orden->>'autenticacion_ref'
 OR v_vinculo->>'sesion_ref' IS DISTINCT FROM p_orden->>'sesion_ref'
 OR v_vinculo->>'autenticacion_huella_sha256' IS DISTINCT FROM p_orden->>'autenticacion_sha256'
 OR v_vinculo->>'autoridad_efectiva' IS DISTINCT FROM 'autoridad_maestra_acreditada'
 OR v_vinculo->>'superficie' IS DISTINCT FROM 'interna_corporativa'
 OR v_vinculo->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
 OR encode(sha256(p_contexto_canonico),'hex') IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(
  p_orden->>'registro_contexto_ref',p_orden->>'contexto_sha256',
  p_orden->>'procedencia_sha256',p_contexto_canonico) IS NOT TRUE
 OR vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(
  p_contexto_canonico) IS NOT TRUE
 OR vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(
  p_vinculo_canonico) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD237: vínculo histórico de lectura inválido' USING ERRCODE='42501'; END IF;
 SELECT * INTO v_sesion FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
  p_orden->>'autenticacion_ref',p_orden->>'sesion_ref');
 IF NOT FOUND THEN RAISE EXCEPTION 'AD237: sesión actual ausente' USING ERRCODE='42501'; END IF;
 IF v_sesion.autenticacion_huella_sha256 IS DISTINCT FROM p_orden->>'autenticacion_sha256'
 OR v_sesion.cuenta_ref IS DISTINCT FROM v_contexto->>'cuenta_ref'
 OR v_sesion.cuenta_ordinaria_ref IS DISTINCT FROM v_vinculo->>'cuenta_ordinaria_ref'
 OR v_sesion.control_sesion_ref IS DISTINCT FROM v_vinculo->>'control_sesion_ref'
 OR v_sesion.control_sesion_revision IS DISTINCT FROM v_vinculo->>'control_sesion_revision'
 OR v_sesion.control_sesion_huella_sha256 IS DISTINCT FROM v_vinculo->>'control_sesion_huella_sha256'
 OR v_sesion.superficie IS DISTINCT FROM 'interna_corporativa'
 OR v_sesion.cuenta_privilegiada IS NOT FALSE
 OR v_sesion.garantia_observada IS DISTINCT FROM 'alto'
 OR v_sesion.garantia_observada IS DISTINCT FROM v_vinculo->>'garantia_observada'
 OR v_sesion.sesion_valida_hasta IS DISTINCT FROM (v_vinculo->>'sesion_valida_hasta')::timestamptz
 OR v_sesion.sesion_revalidada_en IS DISTINCT FROM (v_vinculo->>'sesion_revalidada_en')::timestamptz
 OR vec_autorizacion.cotejar_lectura_bolsa_actual_interna_v1(
  p_orden,v_contexto->>'principal_ref',v_contexto->>'perfil_activo_ref') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD237: autoridad actual de lectura denegada' USING ERRCODE='42501'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac(v_dominio_material)||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_contexto_canonico),'hex'))||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_vinculo_canonico),'hex'));
 FOREACH v_clave IN ARRAY v_claves LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_orden->>v_clave);
 END LOOP;
 v_material_sha:=encode(sha256(v_material),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended(
  'vec_autorizacion_atestada_v3:lectura:'||(p_orden->>'intento_ref'),0));
 PERFORM 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
 WHERE intento_ref=p_orden->>'intento_ref';
 IF FOUND THEN RAISE EXCEPTION 'AD237: lectura ya asentada' USING ERRCODE='23505'; END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 v_secuencia:=v_secuencia+1;v_instante:=clock_timestamp();
 v_ref:=v_prefijo||substr(p_orden->>'intento_ref',9,32);
 v_huella:=encode(sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac(v_dominio)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(to_char(v_instante AT TIME ZONE 'UTC',
   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  intento_ref,intento_material_sha256,actor_ref,perfil_activo_ref,registro_contexto_ref,
  contexto_sha256,procedencia_sha256,autenticacion_ref,sesion_ref,autenticacion_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,
  correlacion_ref,vinculo_sha256,transaccion_origen,
  lectura_revision_permisos,lectura_instantanea_sha256)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,v_tipo,
  p_orden->>'intento_ref',v_material_sha,v_contexto->>'principal_ref',
  v_contexto->>'perfil_activo_ref',p_orden->>'registro_contexto_ref',
  p_orden->>'contexto_sha256',p_orden->>'procedencia_sha256',
  p_orden->>'autenticacion_ref',p_orden->>'sesion_ref',p_orden->>'autenticacion_sha256',
  p_orden->>'accion',p_orden->>'modulo_id',p_orden->>'recurso_ref',
  p_orden->>'finalidad_ref',p_orden->>'resultado',p_orden->>'motivo_ref',
  p_orden->>'proceso',p_orden->>'canal',p_orden->>'correlacion_ref',
  encode(sha256(p_vinculo_canonico),'hex'),pg_current_xact_id(),
  (p_orden->>'lectura_revision_permisos')::numeric,
  p_orden->>'lectura_instantanea_sha256');
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_orden->>'correlacion_ref',v_instante;
END $function$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_exclusiones_llamamiento_v1(
 p_contexto_canonico bytea,p_vinculo_canonico bytea,p_orden jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,
 correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
BEGIN
 IF p_orden->>'accion' IS DISTINCT FROM 'bolsa.rrhh.exclusiones_llamamiento.consultar'
 THEN RAISE EXCEPTION 'AD276: lectura de motivos no autorizada' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT * FROM vec_autorizacion_atestada_v3.registrar_lectura_bolsa_interna_v1(
  p_contexto_canonico,p_vinculo_canonico,p_orden);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_exclusiones_llamamiento_v1(bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_exclusiones_llamamiento_v1(bytea,bytea,jsonb)
 TO vec_bolsa_llamamientos_propietario;
DO $post$ DECLARE f record; BEGIN
 FOR f IN SELECT p.oid,p.prosecdef,p.proowner,p.proconfig FROM pg_proc p
 WHERE p.oid IN (
  'vec_autorizacion.cotejar_lectura_bolsa_actual_interna_v1(jsonb,text,text)'::regprocedure,
  'vec_autorizacion_atestada_v3.registrar_lectura_bolsa_interna_v1(bytea,bytea,jsonb)'::regprocedure,
  'vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(bytea,bytea,jsonb,jsonb,text,text,text)'::regprocedure,
  'vec_autorizacion_atestada_v3.registrar_lectura_exclusiones_llamamiento_v1(bytea,bytea,jsonb)'::regprocedure)
 LOOP
  IF f.prosecdef IS NOT TRUE OR f.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[] IS NOT TRUE
  OR has_function_privilege('public',f.oid,'EXECUTE') THEN
   RAISE EXCEPTION 'AD276: función no cerrada %',f.oid::regprocedure USING ERRCODE='55000';
  END IF;
 END LOOP;
 IF has_function_privilege('vec_bolsa_llamamientos_propietario',
  'vec_autorizacion_atestada_v3.registrar_lectura_exclusiones_llamamiento_v1(bytea,bytea,jsonb)','EXECUTE') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD276: ACL nominal ausente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
