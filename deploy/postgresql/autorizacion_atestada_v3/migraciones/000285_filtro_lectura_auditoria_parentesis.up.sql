\set ON_ERROR_STOP on
-- AD285 / F0-01: agrupar las extracciones JSON antes de concatenar el filtro.
-- Definición completa desde AD251 instalada; conserva autoridad y auditoría.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
-- Solo metadatos de esta transacción; el operador DBA conserva su propiedad.
CREATE TEMP TABLE ad285_meta ON COMMIT DROP AS
 SELECT p.oid,to_jsonb(p)-'prosrc' AS meta,
  encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') AS huella
 FROM pg_proc p WHERE p.oid=to_regprocedure(
  'vec_autorizacion_atestada_v3.registrar_lectura_auditoria_rrhh_v1(bytea,bytea,jsonb)');
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR (SELECT count(*) FROM pg_temp.ad285_meta)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_temp.ad285_meta
  WHERE huella='9b7b81f9f02569dddfe7c16b7678ed3ed587d2482605f1a3521b96db0eebb604'
   AND meta->>'proowner'=('vec_autorizacion_atestada_v3_propietario'::regrole::oid)::text)
 OR EXISTS(SELECT 1 FROM pg_proc p, LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=(SELECT oid FROM pg_temp.ad285_meta) AND a.grantee=0) THEN
  RAISE EXCEPTION 'AD285: preimagen, propietario o ACL divergente; no reaplicar' USING ERRCODE='55000';
 END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_auditoria_rrhh_v1(p_contexto_canonico bytea, p_vinculo_canonico bytea, p_orden jsonb)
 RETURNS TABLE(auditoria_ref text, secuencia numeric, huella_sha256 text, correlacion_ref text, registrada_en timestamp with time zone)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '10s'
AS $function$
DECLARE
 v_contexto jsonb;v_vinculo jsonb;v_sesion record;v_filtro jsonb;
 v_claves constant text[]:=ARRAY[
  'intento_ref','registro_contexto_ref','contexto_sha256','procedencia_sha256',
  'autenticacion_ref','sesion_ref','autenticacion_sha256','accion','modulo_id',
  'recurso_ref','organizacion_ref','unidad_ref','ambito_ref','fuente','finalidad_ref','campos_permitidos',
  'resultado','motivo_ref','proceso','canal','correlacion_ref',
  'lectura_revision_permisos','lectura_instantanea_sha256',
  'asignacion_ref','asignacion_version','asignacion_huella_sha256',
  'version_rol_ref','version_rol_version','version_rol_huella_sha256',
  'control_version_rol_ref','revision_control_version_rol',
  'control_version_rol_huella_sha256','revision_catalogo_politicas',
  'catalogo_politicas_huella_sha256','politicas_evaluadas','emitida_en','valida_hasta',
  'operacion','filtro_sha256','motivo_catalogo_ref','motivo_catalogo_sha256','motivo_consulta'];
 v_campos_filtro constant text[]:=ARRAY[
  'fuente','recurso_ref','actor_ref','desde','hasta','limite','antes_en',
  'antes_fuente','antes_id','finalidad_ref','motivo_ref','motivo_consulta',
  'autor_lectura_ref','busqueda','despues_ref'];
 v_clave text;v_material bytea;v_material_sha text;v_anterior text;v_secuencia numeric;
 v_instante timestamptz(6);v_huella text;v_ref text;v_canon text;v_filtro_sha text;
 v_detalle_sha text;v_entrada text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 OR vec_autorizacion_atestada_v3.login_lectura_auditoria_rrhh_valido_v1() IS NOT TRUE
 OR p_contexto_canonico IS NULL OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
 OR p_vinculo_canonico IS NULL OR octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_orden))<>array_length(v_claves,1)+1
 OR (p_orden ?& v_claves) IS NOT TRUE OR jsonb_typeof(p_orden->'filtro') IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'AD251: emisor de lectura inválido' USING ERRCODE='42501'; END IF;
 FOREACH v_clave IN ARRAY v_claves LOOP
  IF v_clave IN ('campos_permitidos','politicas_evaluadas') THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'array'
   THEN RAISE EXCEPTION 'AD251: orden inválida' USING ERRCODE='22023'; END IF;
  ELSIF v_clave IN ('asignacion_version','version_rol_version',
                    'revision_control_version_rol','revision_catalogo_politicas') THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'number'
   OR (p_orden->>v_clave) !~ '^[1-9][0-9]{0,19}$'
   THEN RAISE EXCEPTION 'AD251: versión inválida' USING ERRCODE='22023'; END IF;
  ELSIF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'string'
   OR octet_length(p_orden->>v_clave) NOT BETWEEN 1 AND 512
  THEN RAISE EXCEPTION 'AD251: orden inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 v_filtro:=p_orden->'filtro';
 IF (SELECT count(*) FROM jsonb_object_keys(v_filtro))<>array_length(v_campos_filtro,1)
 OR (v_filtro ?& v_campos_filtro) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD251: filtro inválido' USING ERRCODE='22023'; END IF;
 FOREACH v_clave IN ARRAY v_campos_filtro LOOP
  IF jsonb_typeof(v_filtro->v_clave) IS DISTINCT FROM 'string'
  OR octet_length(v_filtro->>v_clave)>1024
  THEN RAISE EXCEPTION 'AD251: filtro inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_orden->>'intento_ref' !~ '^lectura_[0-9a-f]{32}$'
 OR p_orden->>'contexto_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'procedencia_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'autenticacion_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'lectura_revision_permisos' !~ '^[1-9][0-9]{0,19}$'
 OR p_orden->>'lectura_instantanea_sha256' !~ '^[0-9a-f]{64}$'
 OR (p_orden->>'lectura_revision_permisos')::numeric>18446744073709551615::numeric
 OR p_orden->>'accion' IS DISTINCT FROM 'vec.auditoria.consultar'
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'auditoria'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'consulta_auditoria_rrhh'
 OR p_orden->>'resultado' IS DISTINCT FROM 'obtenida'
 OR p_orden->>'motivo_ref' IS DISTINCT FROM p_orden->>'motivo_catalogo_ref'
 OR p_orden->>'motivo_catalogo_ref' !~ '^motivos_autorizacion_auditoria:2:[a-z][a-z0-9_.:-]{0,127}$'
 OR p_orden->>'motivo_catalogo_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'proceso' IS DISTINCT FROM 'vec-server'
 OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_orden->>'correlacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_orden->>'filtro_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'organizacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 OR p_orden->>'unidad_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 OR p_orden->>'ambito_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 OR p_orden->>'fuente' NOT IN ('ct','bolsa','persona')
 OR p_orden->>'operacion' NOT IN ('opciones','buscar','consultar')
 OR p_orden->'campos_permitidos' IS DISTINCT FROM
 '["accion","actor_ref","antes","antes_sha256","canal","datos_disponibles","despues","despues_sha256","expediente_ref","finalidad_ref","fuente","id","modulo_id","motivo","ocurrido_en","perfil_ref","recibo_ref","resultado"]'::jsonb
 OR octet_length(p_orden->>'motivo_consulta') NOT BETWEEN 3 AND 500
 OR p_orden->>'motivo_consulta' ~ '[[:cntrl:]]'
 OR EXISTS(SELECT 1 FROM unnest(v_campos_filtro) AS k WHERE v_filtro->>k ~ '[[:cntrl:]]')
 OR v_filtro->>'fuente' IS DISTINCT FROM p_orden->>'fuente'
 OR v_filtro->>'recurso_ref' IS DISTINCT FROM p_orden->>'recurso_ref'
 OR v_filtro->>'finalidad_ref' IS DISTINCT FROM p_orden->>'finalidad_ref'
 OR v_filtro->>'motivo_ref' IS DISTINCT FROM p_orden->>'motivo_catalogo_ref'
 OR v_filtro->>'motivo_consulta' IS DISTINCT FROM p_orden->>'motivo_consulta'
 OR v_filtro->>'autor_lectura_ref' IS NULL
 OR v_filtro->>'limite' !~ '^[1-9][0-9]{0,2}$'
 OR (v_filtro->>'limite')::integer NOT BETWEEN 1 AND 100
 OR v_filtro->>'desde' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
 OR v_filtro->>'hasta' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
 OR (v_filtro->>'antes_en'<>'' AND v_filtro->>'antes_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$')
 OR (v_filtro->>'antes_en'='' AND (v_filtro->>'antes_fuente'<>'' OR v_filtro->>'antes_id'<>''))
 OR (v_filtro->>'antes_en'<>'' AND (v_filtro->>'antes_fuente'<>p_orden->>'fuente' OR v_filtro->>'antes_id'=''))
 OR (p_orden->>'operacion'='buscar' AND (octet_length(v_filtro->>'busqueda')>80 OR octet_length(v_filtro->>'despues_ref')>512))
 OR (p_orden->>'operacion'<>'buscar' AND (v_filtro->>'busqueda'<>'' OR v_filtro->>'despues_ref'<>''))
 OR (p_orden->>'operacion'='opciones' AND (p_orden->>'recurso_ref'<>'auditoria:opciones' OR p_orden->>'fuente'<>'ct'))
 OR (p_orden->>'operacion'='buscar' AND p_orden->>'recurso_ref' IS DISTINCT FROM
  ('auditoria:objetivos:'||(p_orden->>'fuente')))
 THEN RAISE EXCEPTION 'AD251: contrato de lectura inválido' USING ERRCODE='22023'; END IF;
 v_canon:=array_to_string(ARRAY[
  'vec.auditoria.filtro.v1',v_filtro->>'fuente',v_filtro->>'recurso_ref',
  v_filtro->>'actor_ref',v_filtro->>'desde',v_filtro->>'hasta',v_filtro->>'limite',
  v_filtro->>'antes_en',v_filtro->>'antes_fuente',v_filtro->>'antes_id',
  v_filtro->>'finalidad_ref',v_filtro->>'motivo_ref'],E'\n')||E'\nvec.auditoria.lectura-sesion.v1\n'||
  (v_filtro->>'motivo_consulta')||E'\n'||(v_filtro->>'autor_lectura_ref');
 IF p_orden->>'operacion'='buscar' THEN
  v_canon:=v_canon||E'\nvec.auditoria.busqueda.v1\n'||(v_filtro->>'busqueda')||E'\n'||(v_filtro->>'despues_ref');
 END IF;
 v_filtro_sha:=encode(sha256(convert_to(v_canon,'UTF8')),'hex');
 IF v_filtro_sha IS DISTINCT FROM p_orden->>'filtro_sha256'
 THEN RAISE EXCEPTION 'AD251: huella de filtro divergente' USING ERRCODE='42501'; END IF;
 v_entrada:=regexp_replace(p_orden->>'motivo_catalogo_ref','^motivos_autorizacion_auditoria:2:','');
 IF vec_autorizacion.resolver_motivo_autorizacion_v2_actual(
  'motivos_autorizacion_auditoria',2,p_orden->>'motivo_catalogo_sha256',v_entrada) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD251: motivo no vigente' USING ERRCODE='42501'; END IF;
 BEGIN
  v_contexto:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
  v_vinculo:=convert_from(p_vinculo_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AD251: vínculo de lectura inválido' USING ERRCODE='22023';
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
 OR v_filtro->>'autor_lectura_ref' IS DISTINCT FROM
   (v_contexto->>'principal_ref')||':'||(v_contexto->>'perfil_activo_ref')
 OR vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(
  p_orden->>'registro_contexto_ref',p_orden->>'contexto_sha256',
  p_orden->>'procedencia_sha256',p_contexto_canonico) IS NOT TRUE
 OR vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(
  p_contexto_canonico) IS NOT TRUE
 OR vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(
  p_vinculo_canonico) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD251: vínculo histórico inválido' USING ERRCODE='42501'; END IF;
 SELECT * INTO v_sesion FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
  p_orden->>'autenticacion_ref',p_orden->>'sesion_ref');
 IF NOT FOUND THEN RAISE EXCEPTION 'AD251: sesión actual ausente' USING ERRCODE='42501'; END IF;
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
 OR vec_autorizacion.cotejar_lectura_auditoria_rrhh_actual_v1(
  p_orden,v_contexto->>'principal_ref',v_contexto->>'perfil_activo_ref') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD251: autoridad actual denegada' USING ERRCODE='42501'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.lectura_auditoria_rrhh.v1')||
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
 IF FOUND THEN RAISE EXCEPTION 'AD251: lectura ya asentada' USING ERRCODE='23505'; END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 v_secuencia:=v_secuencia+1;v_instante:=clock_timestamp();
 v_ref:='aud_v3_la_'||substr(p_orden->>'intento_ref',9,32);
 v_huella:=encode(sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.lectura_auditoria_rrhh.v1')||
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
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'lectura_auditoria_rrhh',
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
 v_detalle_sha:=encode(sha256(convert_to(array_to_string(ARRAY[
  'vec.auditoria.detalle-lectura-rrhh.v1',v_ref,
  p_orden->>'operacion',p_orden->>'fuente',p_orden->>'organizacion_ref',
  p_orden->>'unidad_ref',p_orden->>'ambito_ref',
  p_orden->>'recurso_ref',v_filtro_sha,p_orden->>'motivo_catalogo_ref',
  p_orden->>'motivo_catalogo_sha256',p_orden->>'motivo_consulta'],E'\n'),'UTF8')),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.detalle_lectura_auditoria_rrhh_v1(
  auditoria_ref,transaccion_origen,operacion,fuente,organizacion_ref,unidad_ref,ambito_ref,recurso_ref,
  filtro_sha256,filtro,motivo_catalogo_ref,motivo_catalogo_sha256,motivo_consulta,detalle_sha256)
 VALUES(v_ref,pg_current_xact_id(),p_orden->>'operacion',p_orden->>'fuente',
  p_orden->>'organizacion_ref',p_orden->>'unidad_ref',p_orden->>'ambito_ref',
  p_orden->>'recurso_ref',v_filtro_sha,v_filtro,
  p_orden->>'motivo_catalogo_ref',p_orden->>'motivo_catalogo_sha256',
  p_orden->>'motivo_consulta',v_detalle_sha);
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_orden->>'correlacion_ref',v_instante;
END $function$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_auditoria_rrhh_v1(bytea,bytea,jsonb) FROM PUBLIC;
RESET ROLE;
DO $post$
DECLARE
 firma constant regprocedure := 'vec_autorizacion_atestada_v3.registrar_lectura_auditoria_rrhh_v1(bytea,bytea,jsonb)'::regprocedure;
 anterior jsonb;
 actual jsonb;
BEGIN
 SELECT meta INTO STRICT anterior FROM pg_temp.ad285_meta;
 SELECT to_jsonb(p)-'prosrc' INTO STRICT actual FROM pg_proc p WHERE p.oid=firma;
 IF actual IS DISTINCT FROM anterior
 OR encode(sha256(convert_to(pg_get_functiondef(firma),'UTF8')),'hex')
  IS DISTINCT FROM '5c832f2a04ce2907f0c8a56f47f4e61dfab8c8523386ba340de7e01a599b76db' THEN
  RAISE EXCEPTION 'AD285: posimagen o metadatos divergentes' USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
