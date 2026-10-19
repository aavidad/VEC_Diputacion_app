\set ON_ERROR_STOP on
-- AD294 / F1-12: el intento nominal externo conserva transaccion_origen NULL.
-- Definición final explícita extraída del cuerpo instalado de AD246.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000294',0));
DO $pre$
DECLARE f regprocedure:=to_regprocedure(
 'vec_autorizacion_atestada_v3.registrar_intento_lectura_preferencias_bolsas_v1(bytea,bytea,jsonb)');
 p pg_proc%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 OR current_user<>'vec_autorizacion_atestada_v3_propietario' OR f IS NULL
 THEN RAISE EXCEPTION 'AD294: preimagen incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
 OR p.prosecdef IS NOT TRUE OR p.provolatile<>'v' OR p.proparallel<>'u'
 OR p.proacl::text IS DISTINCT FROM
  '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_bolsa_llamamientos_propietario=X/vec_autorizacion_atestada_v3_propietario}'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s','statement_timeout=10s']
 OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM
  'd72e98e0858d29535de28eb3ca06dcf166ac9a96c9c7827351499c2a73e49b75'
 THEN RAISE EXCEPTION 'AD294: cuerpo o ACL previos divergentes' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_intento_lectura_preferencias_bolsas_v1(p_contexto_canonico bytea, p_vinculo_canonico bytea, p_orden jsonb)
 RETURNS TABLE(auditoria_ref text, secuencia numeric, huella_sha256 text, correlacion_ref text, registrada_en timestamp with time zone)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '10s'
AS $function$
DECLARE v_contexto jsonb;v_vinculo jsonb;v_candidato text;v_acreditada timestamptz;
 v_claves constant text[]:=ARRAY['intento_ref','registro_contexto_ref','contexto_sha256',
  'procedencia_sha256','autenticacion_ref','sesion_ref','autenticacion_sha256',
  'accion','modulo_id','recurso_ref','finalidad_ref','resultado','motivo_ref',
  'proceso','canal','correlacion_ref'];
 v_clave text;v_material bytea;v_material_sha text;v_anterior text;v_secuencia numeric;
 v_instante timestamptz(6);v_emitida timestamptz(6);v_valida timestamptz(6);v_huella text;v_ref text;v_existente record;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 OR vec_autorizacion_atestada_v3.login_lectura_preferencias_bolsas_valido_v1() IS NOT TRUE
 OR p_contexto_canonico IS NULL OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
 OR p_vinculo_canonico IS NULL OR octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_orden))<>array_length(v_claves,1)
 OR (p_orden ?& v_claves) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD246: intento externo sin emisor acreditado' USING ERRCODE='42501'; END IF;
 FOREACH v_clave IN ARRAY v_claves LOOP
  IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'string'
  OR octet_length(p_orden->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD246: orden de intento inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_orden->>'intento_ref' !~ '^intento_[0-9a-f]{32}$'
 OR p_orden->>'contexto_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'procedencia_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'autenticacion_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'recurso_ref' !~ '^mi-bolsa:[0-9a-f]{64}$'
 OR p_orden->>'accion' IS DISTINCT FROM 'bolsa.preferencias_bolsas.consultar_propias'
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'gestion_preferencias_bolsas_propias'
 OR p_orden->>'resultado' NOT IN ('denegado','error')
 OR p_orden->>'motivo_ref' IS DISTINCT FROM 'lectura_preferencias_bolsas_'||(p_orden->>'resultado')
 OR p_orden->>'proceso' IS DISTINCT FROM 'vec-server'
 OR p_orden->>'canal' IS DISTINCT FROM 'externa_personal'
 OR p_orden->>'correlacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 THEN RAISE EXCEPTION 'AD246: contrato de intento inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  v_contexto:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
  v_vinculo:=convert_from(p_vinculo_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AD246: evidencia de intento inválida' USING ERRCODE='22023'; END;
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
 OR v_vinculo->>'superficie' IS DISTINCT FROM 'externa_personal'
 OR v_vinculo->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
 OR v_vinculo->>'metodo_observado' NOT IN ('certificado','dnie')
 OR v_vinculo->>'garantia_observada' IS DISTINCT FROM 'alto'
 OR encode(sha256(p_contexto_canonico),'hex') IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR jsonb_typeof(v_contexto->'vinculos') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'AD246: vínculo externo de intento inválido' USING ERRCODE='42501'; END IF;
 v_emitida:=clock_timestamp();
 SELECT x->>'referencia' INTO v_candidato FROM jsonb_array_elements(v_contexto->'vinculos') x
 WHERE x->>'tipo'='candidato' AND x->>'estado'='activo';
 IF (SELECT count(*) FROM jsonb_array_elements(v_contexto->'vinculos') x
     WHERE x->>'tipo'='candidato' AND x->>'estado'='activo')<>1
 OR v_candidato IS NULL OR v_candidato !~ '^can_[A-Za-z0-9_-]{22,128}$'
 OR p_orden->>'recurso_ref' IS DISTINCT FROM
  'mi-bolsa:'||encode(sha256(convert_to('mi-bolsa:'||v_candidato,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'AD246: candidato externo denegado' USING ERRCODE='42501'; END IF;
 SELECT least(v_emitida+interval '1 second',
  (v_vinculo->>'sesion_valida_hasta')::timestamptz,
  (v_contexto->>'vigente_hasta')::timestamptz,
  (x->>'vigente_hasta')::timestamptz) INTO v_valida
 FROM jsonb_array_elements(v_contexto->'vinculos') x
 WHERE x->>'tipo'='candidato' AND x->>'estado'='activo';
 IF v_valida<=v_emitida OR vec_autorizacion.revalidar_sesion_vinculo_externo_v2(
  v_vinculo,v_emitida,v_valida,clock_timestamp()) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD246: sesión externa denegada' USING ERRCODE='42501'; END IF;
 v_acreditada:=vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
  p_orden->>'registro_contexto_ref',v_contexto->>'esquema',p_orden->>'contexto_sha256',
  p_orden->>'procedencia_sha256',v_vinculo->>'autoridad_efectiva',
  v_contexto->>'cuenta_ref',(v_contexto->>'cuenta_version')::numeric,
  v_contexto->>'persona_ref',(v_contexto->>'persona_version')::numeric,
  v_contexto->>'perfil_activo_ref',(v_contexto->>'perfil_version')::numeric,
  v_contexto->>'contexto_actor_ref',(v_contexto->>'contexto_version')::numeric,
  v_contexto->>'metodo',v_contexto->>'garantia',
  v_emitida,v_valida);
 IF v_acreditada IS NULL THEN
  RAISE EXCEPTION 'AD246: contexto externo no acreditado' USING ERRCODE='42501'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.intento.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_contexto_canonico),'hex'))||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_vinculo_canonico),'hex'));
 FOREACH v_clave IN ARRAY v_claves LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_orden->>v_clave);
 END LOOP;
 v_material_sha:=encode(sha256(v_material),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:intento:'||(p_orden->>'intento_ref'),0));
 SELECT a.auditoria_ref,a.secuencia,a.huella_sha256,a.correlacion_ref,a.registrada_en,a.intento_material_sha256
 INTO v_existente FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
 WHERE a.intento_ref=p_orden->>'intento_ref';
 IF FOUND THEN
  IF v_existente.intento_material_sha256 IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'AD246: intento incompatible' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,v_existente.huella_sha256,
   v_existente.correlacion_ref,v_existente.registrada_en;
  RETURN;
 END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 v_secuencia:=v_secuencia+1;v_instante:=clock_timestamp();
 v_ref:='aud_v3_i_'||substr(p_orden->>'intento_ref',9,32);
 v_huella:=encode(sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.intento.v1')||
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
  correlacion_ref,vinculo_sha256)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'intento_nominal',
  p_orden->>'intento_ref',v_material_sha,v_contexto->>'principal_ref',
  v_contexto->>'perfil_activo_ref',p_orden->>'registro_contexto_ref',
  p_orden->>'contexto_sha256',p_orden->>'procedencia_sha256',
  p_orden->>'autenticacion_ref',p_orden->>'sesion_ref',p_orden->>'autenticacion_sha256',
  p_orden->>'accion',p_orden->>'modulo_id',p_orden->>'recurso_ref',
  p_orden->>'finalidad_ref',p_orden->>'resultado',p_orden->>'motivo_ref',
  p_orden->>'proceso',p_orden->>'canal',p_orden->>'correlacion_ref',
  encode(sha256(p_vinculo_canonico),'hex'));
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_orden->>'correlacion_ref',v_instante;
END $function$
;

DO $post$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.registrar_intento_lectura_preferencias_bolsas_v1(bytea,bytea,jsonb)'::regprocedure;
 p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
 OR p.prosecdef IS NOT TRUE OR p.provolatile<>'v' OR p.proparallel<>'u'
 OR p.proacl::text IS DISTINCT FROM
  '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_bolsa_llamamientos_propietario=X/vec_autorizacion_atestada_v3_propietario}'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s','statement_timeout=10s']
 OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM
  '819b55746d49c058c718bd4e69bfa41477391d72ca7ce16d6854b57532de7d27'
 THEN RAISE EXCEPTION 'AD294: cuerpo o ACL posteriores divergentes' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
