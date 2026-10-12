\set ON_ERROR_STOP on
-- AD249: literal obtenido del clon HZ21 post AD248; instalación única.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE h text;
BEGIN
 SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') INTO h FROM pg_proc p WHERE p.oid=to_regprocedure('vec_autorizacion.cotejar_lectura_respuestas_llamamiento_actual_v1(jsonb,text,text)');
 IF h IS DISTINCT FROM '664ed47eae7c0e16f909103a9567858a3b21b4557f5928e79e241e59363d2709' THEN RAISE EXCEPTION 'PARO clave=AD249.cotejar_lectura_respuestas_llamamiento_actual_v1 actual=% esperado=664ed47eae7c0e16f909103a9567858a3b21b4557f5928e79e241e59363d2709',h USING ERRCODE='55000'; END IF;
 SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') INTO h FROM pg_proc p WHERE p.oid=to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_respuestas_llamamiento_v1(bytea,bytea,jsonb)');
 IF h IS DISTINCT FROM 'd660933af1b96394bf4ae25c2d79dac13a1d43c8f52cda3d0f837b8062f69435' THEN RAISE EXCEPTION 'PARO clave=AD249.registrar_lectura_respuestas_llamamiento_v1 actual=% esperado=d660933af1b96394bf4ae25c2d79dac13a1d43c8f52cda3d0f837b8062f69435',h USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
DO $familia$
DECLARE anterior text;nueva text;marca text:='''lectura_respuestas_llamamiento''::text';h text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 h:=encode(sha256(convert_to(anterior,'UTF8')),'hex');
 IF h IS DISTINCT FROM '2af234035d9b941aa568dfde21dfb2cea320f22b422a054bb37441cc5437d490' OR (length(anterior)-length(replace(anterior,marca,'')))/length(marca)<>1 THEN
  RAISE EXCEPTION 'PARO clave=AD249.familia actual=% esperado=2af234035d9b941aa568dfde21dfb2cea320f22b422a054bb37441cc5437d490',h USING ERRCODE='55000'; END IF;
 nueva:=replace(anterior,marca,marca||',''lectura_origen_datos_contacto''::text');
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||nueva;
END $familia$;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD CONSTRAINT auditoria_lectura_origen_datos_contacto_formato_v1 CHECK (
 tipo_registro<>'lectura_origen_datos_contacto' OR (
 intento_ref ~ '^lectura_[0-9a-f]{32}$' AND intento_material_sha256 ~ '^[0-9a-f]{64}$'
 AND contexto_sha256 ~ '^[0-9a-f]{64}$' AND procedencia_sha256 ~ '^[0-9a-f]{64}$'
 AND autenticacion_sha256 ~ '^[0-9a-f]{64}$' AND vinculo_sha256 ~ '^[0-9a-f]{64}$'
 AND lectura_revision_permisos BETWEEN 1 AND 18446744073709551615::numeric
 AND lectura_instantanea_sha256 ~ '^[0-9a-f]{64}$'
 AND modulo_id='bolsa' AND proceso='vec-server' AND canal='interna_corporativa'
 AND accion='bolsa.datos_contacto_participacion.origen.consultar' AND recurso_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._-]{0,511}$'
 AND finalidad_ref='consulta_origen_datos_contacto_participacion' AND resultado='obtenida'
 AND motivo_ref='lectura_origen_datos_contacto_correcta'
 AND correlacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'));
RESET ROLE;
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
 WHEN 'bolsa.rrhh.respuestas_llamamiento.consultar' THEN
  finalidad:='seguimiento_contacto_rrhh';tipo:='llamamiento_bolsa';
  campos:='["llamamiento_ref","respuestas[].participacion_ref","respuestas[].respuesta"]'::jsonb;patron:='^llamamiento:[0-9a-f]{64}$';
 WHEN 'bolsa.datos_contacto_participacion.origen.consultar' THEN
  finalidad:='consulta_origen_datos_contacto_participacion';tipo:='participacion_bolsa';
  campos:='["enmascarados","origen","participacion_ref","registrada_en","sin_contacto","version"]'::jsonb;patron:='^[A-Za-z0-9][A-Za-z0-9:._-]{0,511}$';
 ELSE RETURN false; END CASE;
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR p_orden->>'accion' IS DISTINCT FROM accion
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM finalidad
 OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_orden->>'recurso_ref' !~ patron
 OR p_orden->>'bolsa_ref' IS NULL OR p_orden->>'bolsa_ref' LIKE '%*%'
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
  AND CASE WHEN accion='bolsa.datos_contacto_participacion.origen.consultar' OR p_orden ? 'unidad_ref' THEN
   jsonb_typeof(p_orden->'unidad_ref')='string' AND jsonb_typeof(p_orden->'ambito_ref')='string'
   AND vec_autorizacion.texto_positivo_valido(p_orden->>'unidad_ref',512) IS TRUE
   AND vec_autorizacion.texto_positivo_valido(p_orden->>'ambito_ref',512) IS TRUE
   AND jsonb_typeof(a.documento->'ambitos')='array'
   AND jsonb_array_length(a.documento->'ambitos')=2
   AND vec_autorizacion.ambitos_positivos_validos(a.documento) IS TRUE
   AND a.documento->'ambitos' @> jsonb_build_array(
    jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(p_orden->>'unidad_ref')),
    jsonb_build_object('clave','ambito_ref','valores',jsonb_build_array(p_orden->>'ambito_ref')))
   ELSE EXISTS(SELECT 1 FROM jsonb_array_elements(a.documento->'ambitos') ambito
    WHERE ambito->>'clave'='bolsa_ref'
     AND ambito->'valores' @> jsonb_build_array(p_orden->>'bolsa_ref')) END
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
END $function$

REVOKE ALL ON FUNCTION vec_autorizacion.cotejar_lectura_bolsa_actual_interna_v1(jsonb,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.cotejar_lectura_bolsa_actual_interna_v1(jsonb,text,text) TO vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion.cotejar_lectura_respuestas_llamamiento_actual_v1(p_orden jsonb, p_principal text, p_perfil text)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
AS $function$
BEGIN
 IF p_orden->>'accion' IS DISTINCT FROM 'bolsa.rrhh.respuestas_llamamiento.consultar' THEN RETURN false; END IF;
 RETURN vec_autorizacion.cotejar_lectura_bolsa_actual_interna_v1(p_orden,p_principal,p_perfil);
END
$function$;

RESET ROLE;
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
  'recurso_ref','bolsa_ref','finalidad_ref','campos_permitidos','resultado','motivo_ref',
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
 WHEN 'bolsa.rrhh.respuestas_llamamiento.consultar' THEN
  v_tipo:='lectura_respuestas_llamamiento';v_dominio_material:='vec.auditoria.lectura_respuestas_llamamiento.v1';v_dominio:='vec.auditoria.eslabon.lectura_respuestas_llamamiento.v1';v_prefijo:='aud_v3_lr_';
  v_finalidad:='seguimiento_contacto_rrhh';v_motivo:='lectura_respuestas_llamamiento_correcta';
  v_campos:='["llamamiento_ref","respuestas[].participacion_ref","respuestas[].respuesta"]'::jsonb;v_patron:='^llamamiento:[0-9a-f]{64}$';
 WHEN 'bolsa.datos_contacto_participacion.origen.consultar' THEN
  v_tipo:='lectura_origen_datos_contacto';v_dominio_material:='vec.auditoria.lectura_origen_datos_contacto.v1';v_dominio:='vec.auditoria.eslabon.lectura_origen_datos_contacto.v1';v_prefijo:='aud_v3_lc_';
  v_finalidad:='consulta_origen_datos_contacto_participacion';v_motivo:='lectura_origen_datos_contacto_correcta';
  v_campos:='["enmascarados","origen","participacion_ref","registrada_en","sin_contacto","version"]'::jsonb;v_patron:='^[A-Za-z0-9][A-Za-z0-9:._-]{0,511}$';
 ELSE RAISE EXCEPTION 'AD249: operación de lectura no admitida' USING ERRCODE='42501'; END CASE;
 IF v_accion='bolsa.datos_contacto_participacion.origen.consultar' OR p_orden ? 'unidad_ref' THEN
  v_claves:=v_claves||ARRAY['unidad_ref','ambito_ref'];
 END IF;
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
 OR p_orden->>'bolsa_ref' LIKE '%*%'
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
END $function$

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_bolsa_interna_v1(bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_bolsa_interna_v1(bytea,bytea,jsonb) TO vec_bolsa_llamamientos_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_respuestas_llamamiento_v1(p_contexto_canonico bytea, p_vinculo_canonico bytea, p_orden jsonb)
 RETURNS TABLE(auditoria_ref text, secuencia numeric, huella_sha256 text, correlacion_ref text, registrada_en timestamp with time zone)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '10s'
AS $function$
BEGIN
 IF p_orden->>'accion' IS DISTINCT FROM 'bolsa.rrhh.respuestas_llamamiento.consultar' THEN
  RAISE EXCEPTION 'AD249: contrato B103 de respuesta no admitido' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT * FROM vec_autorizacion_atestada_v3.registrar_lectura_bolsa_interna_v1(p_contexto_canonico,p_vinculo_canonico,p_orden);
END
$function$;

COMMIT;
