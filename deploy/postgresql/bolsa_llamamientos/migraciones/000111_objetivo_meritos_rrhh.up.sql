\set ON_ERROR_STOP on
-- B111: metadato técnico de persona sólo en el sobre SQL del detalle RRHH.
-- Conserva la proyección pública, listas y lector exterior de B96/B99.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp; SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000111',0));
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regprocedure('vec_bolsa_llamamientos.consultar_inscripcion_v1(text,jsonb,bytea,bytea,jsonb)') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.cotejar_objetivo_revision_meritos_v1(text,text,text,text,bytea,bytea,jsonb)') IS NOT NULL
 OR to_regrole('vec_meritos_propietario') IS NULL
 THEN RAISE EXCEPTION 'B111: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $detalle$
DECLARE f oid:=to_regprocedure('vec_bolsa_llamamientos.consultar_inscripcion_v1(text,jsonb,bytea,bytea,jsonb)');
 original text;nuevo text;actual text;meta jsonb;
 declaracion text:=$d$ v_conjunto jsonb; v_ambito_solicitud jsonb; v_unidad_ref text; v_ambito_ref text;$d$;
 declaracion_nueva text:=$d$ v_conjunto jsonb; v_ambito_solicitud jsonb; v_unidad_ref text; v_ambito_ref text;
 v_persona_objetivo text;$d$;
 seleccion text:=$s$'fuente_sha256',s.ambito_fuente_sha256) INTO STRICT v_ambito_solicitud$s$;
 seleccion_nueva text:=$s$'fuente_sha256',s.ambito_fuente_sha256),s.persona_ref
    INTO STRICT v_ambito_solicitud,v_persona_objetivo$s$;
 retorno text:=$r$ RETURN jsonb_build_object('resultado',v_resultado,'proyeccion',v_proyeccion,
  'auditoria_ref',v_auditoria.auditoria_ref,
  'correlacion_ref',v_auditoria.correlacion_ref,'consultada_en',v_auditoria.registrada_en);$r$;
 retorno_nuevo text:=$r$ IF p_accion='bolsa.inscripcion.rrhh.consultar' AND v_resultado='obtenida'
  AND (v_persona_objetivo IS NULL OR v_persona_objetivo !~ '^per_[A-Za-z0-9_-]{22,128}$')
 THEN RAISE EXCEPTION 'B111: persona objetivo incompatible' USING ERRCODE='55000'; END IF;
 RETURN jsonb_build_object('resultado',v_resultado,'proyeccion',v_proyeccion,
  'auditoria_ref',v_auditoria.auditoria_ref,
  'correlacion_ref',v_auditoria.correlacion_ref,'consultada_en',v_auditoria.registrada_en)
  || CASE WHEN p_accion='bolsa.inscripcion.rrhh.consultar' AND v_resultado='obtenida'
     THEN jsonb_build_object('persona_ref_interna',v_persona_objetivo)
     ELSE '{}'::jsonb END;$r$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO original,meta FROM pg_proc p WHERE p.oid=f;
 IF original IS NULL OR encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM
  'f30b4cba82e2e09ace0d6bb0cd4c4ec1bda06c82ce30b93a45910623e5c8b64b'
 OR (length(original)-length(replace(original,declaracion,'')))/length(declaracion)<>1
 OR (length(original)-length(replace(original,seleccion,'')))/length(seleccion)<>1
 OR (length(original)-length(replace(original,retorno,'')))/length(retorno)<>1
 OR strpos(original,'persona_ref_interna')<>0
 THEN RAISE EXCEPTION 'B111: lector instalado divergente' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,declaracion,declaracion_nueva);
 nuevo:=replace(nuevo,seleccion,seleccion_nueva);
 nuevo:=replace(nuevo,retorno,retorno_nuevo);
END $detalle$;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.consultar_inscripcion_v1(p_accion text, p_selector jsonb, p_contexto_canonico bytea, p_vinculo_canonico bytea, p_captura jsonb)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE
 v_contexto jsonb; v_vinculo jsonb; v_orden jsonb; v_proyeccion jsonb;
 v_resultado text:='obtenida'; v_persona text; v_canal text; v_idioma text;
 v_recurso text; v_recurso_esperado text; v_recurso_canonico text;
 v_filtro jsonb; v_f_estado text; v_f_convocatoria text; v_f_cursor text;
 v_f_limite integer; v_ref_selector text; v_prefijo text;
 v_revision_permisos numeric(20,0); v_huella_instantanea text;
 v_conjunto jsonb; v_ambito_solicitud jsonb; v_unidad_ref text; v_ambito_ref text;
 v_persona_objetivo text;
 v_motivos jsonb; v_motivos_proyeccion jsonb;
 v_ahora timestamptz(6); v_emitida_en timestamptz; v_valida_hasta timestamptz;
 v_auditoria record;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR session_user NOT IN('vec_bolsa_inscripciones_lector','vec_bolsa_inscripciones_rrhh_lector')
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR jsonb_typeof(p_selector) IS DISTINCT FROM 'object'
 OR jsonb_typeof(p_captura) IS DISTINCT FROM 'object'
 OR p_contexto_canonico IS NULL OR p_vinculo_canonico IS NULL
 OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
 OR octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384
 THEN RAISE EXCEPTION 'B96: consulta no disponible' USING ERRCODE='42501'; END IF;
 BEGIN
  v_contexto:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
  v_vinculo:=convert_from(p_vinculo_canonico,'UTF8')::jsonb;
  v_ahora:=date_trunc('microseconds',clock_timestamp());
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'B96: captura lectora inválida' USING ERRCODE='42501';
 END;
 v_persona:=p_captura->>'persona_ref';
 v_canal:=p_captura->>'canal';
 v_idioma:=p_captura->>'idioma';
 v_recurso:=p_captura->>'recurso_ref';
 v_huella_instantanea:=p_captura->>'huella_instantanea_sha256';
 IF jsonb_typeof(p_captura->'revision_permisos') IS DISTINCT FROM 'number'
 OR coalesce(p_captura->>'revision_permisos','') !~ '^[1-9][0-9]{0,19}$'
 OR coalesce(v_huella_instantanea,'') !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'B96: revisión de permisos inválida' USING ERRCODE='42501'; END IF;
 v_revision_permisos:=(p_captura->>'revision_permisos')::numeric(20,0);
 IF v_revision_permisos>18446744073709551615::numeric
 THEN RAISE EXCEPTION 'B96: revisión de permisos fuera de rango' USING ERRCODE='42501'; END IF;
 IF jsonb_typeof(p_captura->'emitida_en') IS DISTINCT FROM 'string'
 OR jsonb_typeof(p_captura->'valida_hasta') IS DISTINCT FROM 'string'
 THEN RAISE EXCEPTION 'B96: ventana de captura ausente' USING ERRCODE='42501'; END IF;
 BEGIN
  v_emitida_en:=(p_captura->>'emitida_en')::timestamptz;
  v_valida_hasta:=(p_captura->>'valida_hasta')::timestamptz;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'B96: ventana de captura inválida' USING ERRCODE='42501';
 END;
 IF v_persona IS NULL OR v_persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_captura->>'accion' IS DISTINCT FROM p_accion
 OR p_captura->>'perfil_ref' IS DISTINCT FROM v_contexto->>'perfil_activo_ref'
 OR p_captura->>'perfil_ref' IS DISTINCT FROM v_vinculo->>'perfil_activo_ref'
 OR p_captura->>'persona_ref' IS DISTINCT FROM v_contexto->>'principal_ref'
 OR p_captura->>'persona_ref' IS DISTINCT FROM v_vinculo->>'principal_id'
 OR p_captura->>'cuenta_ref' IS DISTINCT FROM v_contexto->>'cuenta_ref'
 OR p_captura->>'cuenta_ref' IS DISTINCT FROM v_vinculo->>'cuenta_ref'
 OR p_captura->>'sesion_ref' IS DISTINCT FROM v_vinculo->>'sesion_ref'
 OR p_captura->>'autenticacion_ref' IS DISTINCT FROM v_vinculo->>'autenticacion_ref'
 OR p_captura->>'canal' IS DISTINCT FROM v_vinculo->>'superficie'
 OR v_idioma IS NULL OR v_idioma NOT IN('es','en')
 OR v_canal IS NULL OR v_canal NOT IN('externa_personal','interna_corporativa')
 OR (v_emitida_en<=v_ahora AND v_valida_hasta>v_ahora) IS NOT TRUE
 OR NOT isfinite(v_emitida_en) OR NOT isfinite(v_valida_hasta)
 OR coalesce(p_captura->>'intento_ref','') !~ '^lectura_[0-9a-f]{32}$'
 OR p_captura->>'correlacion_ref' IS NULL
 OR p_captura->>'finalidad' IS NULL
 OR v_recurso IS NULL
 THEN RAISE EXCEPTION 'B96: sesión o captura no ligadas' USING ERRCODE='42501'; END IF;
 IF (
  (session_user='vec_bolsa_inscripciones_lector'
   AND v_canal='externa_personal'
   AND p_accion IN('bolsa.inscripcion.convocatorias.listar',
    'bolsa.inscripcion.convocatoria.consultar',
    'bolsa.inscripcion.propias.listar','bolsa.inscripcion.propia.consultar'))
  OR (session_user='vec_bolsa_inscripciones_rrhh_lector'
   AND v_canal='interna_corporativa'
   AND p_accion IN('bolsa.inscripcion.rrhh.listar',
    'bolsa.inscripcion.rrhh.consultar','bolsa.inscripcion.rrhh.motivos',
    'bolsa.inscripcion.rrhh.convocatorias.listar'))
 ) IS NOT TRUE THEN RAISE EXCEPTION 'B96: LOGIN lector fuera de competencia' USING ERRCODE='42501'; END IF;
 IF p_accion IN('bolsa.inscripcion.rrhh.listar','bolsa.inscripcion.rrhh.consultar',
  'bolsa.inscripcion.rrhh.motivos','bolsa.inscripcion.rrhh.convocatorias.listar') THEN
  v_conjunto:=vec_catalogos_configurables.cotejar_conjunto_gestion_rrhh_inscripcion_actual_v1();
  IF jsonb_typeof(p_captura->'conjunto_gestion') IS DISTINCT FROM 'object'
   OR p_captura->'conjunto_gestion' IS DISTINCT FROM v_conjunto
  THEN RAISE EXCEPTION 'B96: conjunto de gestión divergente' USING ERRCODE='42501'; END IF;
  v_unidad_ref:=v_conjunto->>'unidad_ref';
  v_ambito_ref:=v_conjunto->>'ambito_ref';
 ELSIF p_captura ? 'conjunto_gestion' THEN
  RAISE EXCEPTION 'B96: conjunto fuera de canal RRHH' USING ERRCODE='42501';
 END IF;

 -- Mismo canon que application/inscripcion.RecursoLectura: el recurso del
 -- permiso y de la auditoría se ata al selector, idioma y persona exactos.
 v_filtro:=p_captura->'filtro';
 IF jsonb_typeof(v_filtro) IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(v_filtro) ORDER BY 1) IS DISTINCT FROM
  ARRAY['convocatoria_ref','cursor','estado','limite']
 OR jsonb_typeof(v_filtro->'estado') IS DISTINCT FROM 'string'
 OR jsonb_typeof(v_filtro->'convocatoria_ref') IS DISTINCT FROM 'string'
 OR jsonb_typeof(v_filtro->'cursor') IS DISTINCT FROM 'string'
 OR jsonb_typeof(v_filtro->'limite') IS DISTINCT FROM 'number'
 OR coalesce(v_filtro->>'limite','') !~ '^(0|[1-9][0-9]{0,2})$'
 THEN RAISE EXCEPTION 'B96: filtro de captura inválido' USING ERRCODE='42501'; END IF;
 v_f_estado:=v_filtro->>'estado';
 v_f_convocatoria:=v_filtro->>'convocatoria_ref';
 v_f_cursor:=v_filtro->>'cursor';
 v_f_limite:=(v_filtro->>'limite')::integer;
 IF p_accion IN('bolsa.inscripcion.convocatoria.consultar',
    'bolsa.inscripcion.propia.consultar','bolsa.inscripcion.rrhh.consultar',
    'bolsa.inscripcion.rrhh.motivos') THEN
  IF v_filtro IS DISTINCT FROM jsonb_build_object(
     'estado','','convocatoria_ref','','limite',0,'cursor','')
  THEN RAISE EXCEPTION 'B96: filtro de detalle divergente' USING ERRCODE='42501'; END IF;
 END IF;
 IF p_accion='bolsa.inscripcion.convocatoria.consultar' THEN
  v_ref_selector:=p_selector->>'convocatoria_ref';
  v_recurso_esperado:=v_ref_selector;
 ELSIF p_accion IN('bolsa.inscripcion.propia.consultar',
                   'bolsa.inscripcion.rrhh.consultar') THEN
  v_ref_selector:=p_selector->>'solicitud_ref';
  v_recurso_esperado:=v_ref_selector;
 ELSIF p_accion='bolsa.inscripcion.convocatorias.listar' THEN
  IF v_f_estado<>'' OR v_f_convocatoria<>''
   OR v_f_limite::text IS DISTINCT FROM p_selector->>'limite'
   OR v_f_cursor IS DISTINCT FROM p_selector->>'cursor'
  THEN RAISE EXCEPTION 'B96: lista abierta no ligada' USING ERRCODE='42501'; END IF;
  v_ref_selector:='';v_prefijo:='inscripciones_abiertas_';
 ELSIF p_accion='bolsa.inscripcion.propias.listar' THEN
  IF v_filtro IS DISTINCT FROM p_selector THEN
   RAISE EXCEPTION 'B96: lista propia no ligada' USING ERRCODE='42501'; END IF;
  v_ref_selector:='';v_prefijo:='inscripciones_propias_';
 ELSIF p_accion='bolsa.inscripcion.rrhh.listar' THEN
  IF v_filtro IS DISTINCT FROM p_selector THEN
   RAISE EXCEPTION 'B96: lista RRHH no ligada' USING ERRCODE='42501'; END IF;
  v_ref_selector:='';v_prefijo:='inscripciones_rrhh_';
 ELSIF p_accion='bolsa.inscripcion.rrhh.convocatorias.listar' THEN
  IF v_f_estado<>'' OR v_f_convocatoria<>''
   OR v_f_limite::text IS DISTINCT FROM p_selector->>'limite'
   OR v_f_cursor IS DISTINCT FROM p_selector->>'cursor'
  THEN RAISE EXCEPTION 'B96: selector de gestión no ligado' USING ERRCODE='42501'; END IF;
  v_ref_selector:='';v_prefijo:='inscripciones_rrhh_convocatorias_';
 ELSIF p_accion='bolsa.inscripcion.rrhh.motivos' THEN
  v_ref_selector:=p_selector->>'decision';v_prefijo:='motivos_inscripcion_';
 ELSE RAISE EXCEPTION 'B96: acción lectora desconocida' USING ERRCODE='42501'; END IF;
 IF v_prefijo IS NOT NULL THEN
  IF p_accion='bolsa.inscripcion.rrhh.convocatorias.listar' THEN
   v_recurso_canonico:='{"accion":'||to_json(p_accion)::text||
    ',"persona_ref":'||to_json(v_persona)::text||
    ',"idioma":'||to_json(v_idioma)::text||
    ',"limite":'||v_f_limite::text||
    ',"cursor":'||to_json(v_f_cursor)::text||'}';
  ELSE
   v_recurso_canonico:='{"accion":'||to_json(p_accion)::text||
    ',"persona_ref":'||to_json(v_persona)::text||
    ',"idioma":'||to_json(v_idioma)::text||
    ',"estado":'||to_json(v_f_estado)::text||
    ',"convocatoria_ref":'||to_json(v_f_convocatoria)::text||
    ',"limite":'||v_f_limite::text||
    ',"cursor":'||to_json(v_f_cursor)::text||
    ',"ref":'||to_json(v_ref_selector)::text||'}';
  END IF;
  v_recurso_esperado:=v_prefijo||encode(sha256(convert_to(v_recurso_canonico,'UTF8')),'hex');
 END IF;
 IF v_recurso IS DISTINCT FROM v_recurso_esperado THEN
  RAISE EXCEPTION 'B96: recurso de lectura no ligado al selector' USING ERRCODE='42501';
 END IF;

 IF p_accion='bolsa.inscripcion.convocatorias.listar' THEN
  IF v_canal<>'externa_personal'
   OR ARRAY(SELECT jsonb_object_keys(p_selector) ORDER BY 1) IS DISTINCT FROM ARRAY['cursor','limite']
   OR jsonb_typeof(p_selector->'limite') IS DISTINCT FROM 'number'
   OR coalesce(p_selector->>'limite','') !~ '^[1-9][0-9]{0,2}$'
   OR (p_selector->>'limite')::integer NOT BETWEEN 1 AND 100
   OR jsonb_typeof(p_selector->'cursor') IS DISTINCT FROM 'string'
  THEN RAISE EXCEPTION 'B96: lista abierta inválida' USING ERRCODE='B9605'; END IF;
  v_proyeccion:=vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(
   NULLIF(p_selector->>'cursor',''),(p_selector->>'limite')::integer);
  v_proyeccion:=vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(
   v_proyeccion,v_persona,v_idioma,v_contexto->>'metodo',v_contexto->>'garantia',true,v_canal);
 ELSIF p_accion='bolsa.inscripcion.convocatoria.consultar' THEN
  IF v_canal<>'externa_personal'
   OR ARRAY(SELECT jsonb_object_keys(p_selector) ORDER BY 1) IS DISTINCT FROM ARRAY['convocatoria_ref']
   OR v_recurso IS DISTINCT FROM p_selector->>'convocatoria_ref'
  THEN RAISE EXCEPTION 'B96: detalle abierto inválido' USING ERRCODE='B9605'; END IF;
  v_proyeccion:=vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(
   p_selector->>'convocatoria_ref');
  IF v_proyeccion IS NULL THEN v_resultado:='no_encontrada';
  ELSE v_proyeccion:=vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(
   v_proyeccion,v_persona,v_idioma,v_contexto->>'metodo',v_contexto->>'garantia',false,v_canal);
  END IF;
 ELSIF p_accion='bolsa.inscripcion.rrhh.convocatorias.listar' THEN
  IF v_canal<>'interna_corporativa'
   OR ARRAY(SELECT jsonb_object_keys(p_selector) ORDER BY 1) IS DISTINCT FROM ARRAY['cursor','limite']
   OR jsonb_typeof(p_selector->'limite') IS DISTINCT FROM 'number'
   OR coalesce(p_selector->>'limite','') !~ '^[1-9][0-9]{0,2}$'
   OR (p_selector->>'limite')::integer NOT BETWEEN 1 AND 100
   OR jsonb_typeof(p_selector->'cursor') IS DISTINCT FROM 'string'
   OR (p_selector->>'cursor'<>'' AND
      p_selector->>'cursor' !~ '^cv1_[0-9a-f]{64}_v[1-9][0-9]{0,15}$')
  THEN RAISE EXCEPTION 'B96: lista de gestión inválida' USING ERRCODE='B9605'; END IF;
  v_proyeccion:=vec_bolsa_llamamientos.listar_convocatorias_rrhh_inscripcion_interna_v1(
   v_unidad_ref,v_ambito_ref,p_selector->>'cursor',(p_selector->>'limite')::integer,v_idioma);
 ELSIF p_accion IN('bolsa.inscripcion.propias.listar','bolsa.inscripcion.rrhh.listar') THEN
  IF (p_accion='bolsa.inscripcion.propias.listar'
      AND v_canal<>'externa_personal')
   OR (p_accion='bolsa.inscripcion.rrhh.listar' AND v_canal<>'interna_corporativa')
   OR p_captura->'filtro' IS DISTINCT FROM p_selector
  THEN RAISE EXCEPTION 'B96: filtro no ligado' USING ERRCODE='42501'; END IF;
  v_proyeccion:=vec_bolsa_llamamientos.listar_solicitudes_inscripcion_interna_v1(
   p_accion='bolsa.inscripcion.propias.listar',v_persona,p_selector,v_idioma,v_canal,
   v_unidad_ref,v_ambito_ref);
 ELSIF p_accion IN('bolsa.inscripcion.propia.consultar','bolsa.inscripcion.rrhh.consultar') THEN
  IF (p_accion='bolsa.inscripcion.propia.consultar'
      AND v_canal<>'externa_personal')
   OR (p_accion='bolsa.inscripcion.rrhh.consultar' AND v_canal<>'interna_corporativa')
   OR ARRAY(SELECT jsonb_object_keys(p_selector) ORDER BY 1) IS DISTINCT FROM ARRAY['solicitud_ref']
   OR v_recurso IS DISTINCT FROM p_selector->>'solicitud_ref'
  THEN RAISE EXCEPTION 'B96: detalle no ligado' USING ERRCODE='42501'; END IF;
  v_proyeccion:=vec_bolsa_llamamientos.leer_solicitud_inscripcion_interna_v1(
   p_accion='bolsa.inscripcion.propia.consultar',v_persona,
   p_selector->>'solicitud_ref',v_idioma,v_canal,v_unidad_ref,v_ambito_ref);
  IF v_proyeccion IS NULL THEN v_resultado:='no_encontrada';
  ELSIF p_accion='bolsa.inscripcion.rrhh.consultar' THEN
   SELECT jsonb_build_object('solicitud_ref',s.solicitud_ref,
    'unidad_ref',s.unidad_ref,'ambito_ref',s.ambito_ref,
    'fuente_ref',s.ambito_fuente_ref,'fuente_version',s.ambito_fuente_version,
    'fuente_sha256',s.ambito_fuente_sha256),s.persona_ref
    INTO STRICT v_ambito_solicitud,v_persona_objetivo
   FROM vec_bolsa_llamamientos.solicitud_inscripcion s
   WHERE s.solicitud_ref=p_selector->>'solicitud_ref'
    AND s.unidad_ref=v_unidad_ref AND s.ambito_ref=v_ambito_ref;
   -- El Go envía el ámbito sin solicitud_ref: recurso_ref ya ata la solicitud.
   IF p_captura->'ambito_solicitud' IS DISTINCT FROM (v_ambito_solicitud-'solicitud_ref')
   THEN RAISE EXCEPTION 'B96: ámbito histórico divergente' USING ERRCODE='42501'; END IF;
  END IF;
 ELSIF p_accion='bolsa.inscripcion.rrhh.motivos' THEN
  IF v_canal<>'interna_corporativa'
   OR ARRAY(SELECT jsonb_object_keys(p_selector) ORDER BY 1) IS DISTINCT FROM ARRAY['decision']
   OR jsonb_typeof(p_selector->'decision') IS DISTINCT FROM 'string'
   OR (p_selector->>'decision' IN('admitir','rechazar')) IS NOT TRUE
  THEN RAISE EXCEPTION 'B96: motivos no ligados' USING ERRCODE='42501'; END IF;
  v_motivos:=vec_catalogos_configurables.listar_motivos_inscripcion_v1(v_idioma);
  SELECT coalesce(jsonb_agg(jsonb_build_object('codigo',x.valor->>'motivo_ref',
    'etiqueta',x.valor->>'motivo_etiqueta','obligatorio',true) ORDER BY x.valor->>'motivo_ref'),'[]'::jsonb)
  INTO v_motivos_proyeccion FROM jsonb_array_elements(v_motivos->'motivos') AS x(valor)
  WHERE p_selector->>'decision'='rechazar';
  v_proyeccion:=jsonb_build_object('catalogo_version',(v_motivos->>'catalogo_version')::bigint,
   'motivos',v_motivos_proyeccion);
 ELSE RAISE EXCEPTION 'B96: acción lectora desconocida' USING ERRCODE='42501'; END IF;

 v_orden:=jsonb_build_object(
  'intento_ref',p_captura->>'intento_ref',
  'registro_contexto_ref',v_vinculo->>'registro_contexto_ref',
  'contexto_sha256',v_vinculo->>'contexto_actor_huella_sha256',
  'procedencia_sha256',v_vinculo->>'manifiesto_procedencia_huella_sha256',
  'autenticacion_ref',v_vinculo->>'autenticacion_ref',
  'sesion_ref',v_vinculo->>'sesion_ref',
  'autenticacion_sha256',v_vinculo->>'autenticacion_huella_sha256',
  'lectura_revision_permisos',v_revision_permisos::text,
  'lectura_instantanea_sha256',v_huella_instantanea,
  'accion',p_accion,'modulo_id','bolsa','recurso_ref',v_recurso,
  'finalidad_ref',p_captura->>'finalidad',
  'resultado',v_resultado,
  'motivo_ref',CASE WHEN v_resultado='obtenida' THEN 'inscripcion_lectura_correcta'
                   ELSE 'inscripcion_no_encontrada' END,
  'proceso','vec-server','canal',v_canal,
  'correlacion_ref',p_captura->>'correlacion_ref');
 SELECT * INTO STRICT v_auditoria
  FROM vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(
   p_contexto_canonico,p_vinculo_canonico,v_orden);
 IF v_auditoria.auditoria_ref IS NULL
 OR v_auditoria.correlacion_ref IS DISTINCT FROM p_captura->>'correlacion_ref'
 THEN RAISE EXCEPTION 'B96: asiento de lectura incompatible' USING ERRCODE='55000'; END IF;
 IF p_accion='bolsa.inscripcion.rrhh.consultar' AND v_resultado='obtenida'
  AND (v_persona_objetivo IS NULL OR v_persona_objetivo !~ '^per_[A-Za-z0-9_-]{22,128}$')
 THEN RAISE EXCEPTION 'B111: persona objetivo incompatible' USING ERRCODE='55000'; END IF;
 RETURN jsonb_build_object('resultado',v_resultado,'proyeccion',v_proyeccion,
  'auditoria_ref',v_auditoria.auditoria_ref,
  'correlacion_ref',v_auditoria.correlacion_ref,'consultada_en',v_auditoria.registrada_en)
  || CASE WHEN p_accion='bolsa.inscripcion.rrhh.consultar' AND v_resultado='obtenida'
     THEN jsonb_build_object('persona_ref_interna',v_persona_objetivo)
     ELSE '{}'::jsonb END;
END $function$;

DO $verificar_literal$ BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef('vec_bolsa_llamamientos.consultar_inscripcion_v1(text,jsonb,bytea,bytea,jsonb)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM '974749bbd41e460520496a22dbcbd10f8450ac69c4da405b41b3a625aede1e0f'
 THEN RAISE EXCEPTION 'B111: definición final divergente' USING ERRCODE='55000'; END IF;
END $verificar_literal$;


CREATE FUNCTION vec_bolsa_llamamientos.cotejar_objetivo_revision_meritos_v1(
 p_solicitud_ref text,p_persona_ref text,p_actor_ref text,p_perfil_ref text,
 p_contexto bytea,p_vinculo bytea,p_captura jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE c jsonb; v jsonb; detalle jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR session_user<>'vec_bolsa_inscripciones_rrhh_lector'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR p_solicitud_ref !~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 OR p_persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_actor_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_perfil_ref !~ '^prf_[A-Za-z0-9_-]{22,128}$'
 OR jsonb_typeof(p_captura) IS DISTINCT FROM 'object'
 OR p_captura->>'accion' IS DISTINCT FROM 'bolsa.inscripcion.rrhh.consultar'
 OR p_captura->>'recurso_ref' IS DISTINCT FROM p_solicitud_ref
 OR p_captura->>'persona_ref' IS DISTINCT FROM p_actor_ref
 THEN RETURN false; END IF;
 BEGIN c:=convert_from(p_contexto,'UTF8')::jsonb;v:=convert_from(p_vinculo,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN RETURN false; END;
 IF c->>'principal_ref' IS DISTINCT FROM p_actor_ref
 OR c->>'perfil_activo_ref' IS DISTINCT FROM p_perfil_ref
 OR v->>'principal_id' IS DISTINCT FROM p_actor_ref
 OR v->>'perfil_activo_ref' IS DISTINCT FROM p_perfil_ref
 THEN RETURN false; END IF;
 detalle:=vec_bolsa_llamamientos.consultar_inscripcion_v1(
  'bolsa.inscripcion.rrhh.consultar',jsonb_build_object('solicitud_ref',p_solicitud_ref),
  p_contexto,p_vinculo,p_captura);
 RETURN detalle->>'resultado' IS NOT DISTINCT FROM 'obtenida'
  AND detalle->>'persona_ref_interna' IS NOT DISTINCT FROM p_persona_ref
  AND coalesce(detalle->>'auditoria_ref','')<>'';
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.cotejar_objetivo_revision_meritos_v1(
 text,text,text,text,bytea,bytea,jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_meritos_propietario;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.cotejar_objetivo_revision_meritos_v1(
 text,text,text,text,bytea,bytea,jsonb) TO vec_meritos_propietario;
COMMIT;
