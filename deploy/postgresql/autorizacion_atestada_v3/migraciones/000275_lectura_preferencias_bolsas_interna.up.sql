\set ON_ERROR_STOP on
-- AD275: fachadas nominales de lectura y denegación de preferencias de Mi Bolsa interna.
-- AD279 publica el consumidor de escritura y amplía el CHECK auditado antes de este corte.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000275',0));
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regrole('vec_bolsa_llamamientos_preferencias_interna') IS NULL
 OR to_regprocedure('vec_autorizacion.cotejar_lectura_preferencias_bolsas_actual_interna_v1(jsonb,text,text)') IS NULL
 OR has_function_privilege('vec_autorizacion_atestada_v3_propietario',
  to_regprocedure('vec_autorizacion.cotejar_lectura_preferencias_bolsas_actual_interna_v1(jsonb,text,text)'),'EXECUTE') IS NOT TRUE
 OR to_regprocedure('vec_autorizacion.revalidar_sesion_vinculo_v2(jsonb,timestamptz,timestamptz,timestamptz)') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_interna_v1(bytea,bytea,jsonb)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_lectura_preferencias_bolsas_interna_v1(bytea,bytea,jsonb)') IS NOT NULL
 OR NOT EXISTS(SELECT 1 FROM pg_constraint c
  WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
    AND c.conname='auditoria_lectura_preferencias_bolsas_formato_v2'
    AND c.contype='c' AND c.convalidated
    AND pg_get_constraintdef(c.oid,false) LIKE '%interna_corporativa%')
 THEN RAISE EXCEPTION 'AD275: preimagen interna AD279 incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.login_lectura_preferencias_bolsas_interna_valido_v1()
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp AS $f$
 SELECT current_setting('role')='none' AND EXISTS(
  SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
  JOIN pg_roles g ON g.oid=m.roleid
  WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
   AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
   AND l.rolconfig IS NULL
   AND g.rolname='vec_bolsa_llamamientos_preferencias_interna'
   AND NOT(g.rolcanlogin
    OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
   AND g.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
   AND (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1
   AND NOT EXISTS(SELECT 1 FROM pg_auth_members x WHERE x.member=g.oid)
   AND NOT EXISTS(SELECT 1 FROM pg_db_role_setting x WHERE x.setrole IN(l.oid,g.oid)));
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.login_lectura_preferencias_bolsas_interna_valido_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_interna_v1(
 p_contexto_canonico bytea,p_vinculo_canonico bytea,p_orden jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,
 correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s'
SET statement_timeout='10s' AS $f$
DECLARE
 v_contexto jsonb;v_vinculo jsonb;v_vinculos integer;v_acreditada timestamptz;
 v_claves constant text[]:=ARRAY[
  'intento_ref','registro_contexto_ref','contexto_sha256','procedencia_sha256',
  'autenticacion_ref','sesion_ref','autenticacion_sha256','accion','modulo_id',
  'recurso_ref','candidato_ref','tipo_recurso','finalidad_ref',
  'campos_permitidos','resultado','motivo_ref',
  'proceso','canal','correlacion_ref','lectura_revision_permisos',
  'lectura_instantanea_sha256','asignacion_ref','asignacion_version',
  'asignacion_huella_sha256','version_rol_ref','version_rol_version',
  'version_rol_huella_sha256','control_version_rol_ref','revision_control_version_rol',
  'control_version_rol_huella_sha256','revision_catalogo_politicas',
  'catalogo_politicas_huella_sha256','politicas_evaluadas','emitida_en','valida_hasta'];
 v_clave text;v_material bytea;v_material_sha text;v_anterior text;v_secuencia numeric;
 v_instante timestamptz(6);v_huella text;v_ref text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 OR vec_autorizacion_atestada_v3.login_lectura_preferencias_bolsas_interna_valido_v1() IS NOT TRUE
 OR p_contexto_canonico IS NULL OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
 OR p_vinculo_canonico IS NULL OR octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_orden))<>array_length(v_claves,1)
 OR (p_orden ?& v_claves) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD275: lectura sin emisor acreditado' USING ERRCODE='42501'; END IF;
 FOREACH v_clave IN ARRAY v_claves LOOP
  IF v_clave IN ('campos_permitidos','politicas_evaluadas') THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'array'
   THEN RAISE EXCEPTION 'AD275: orden de lectura inválida' USING ERRCODE='22023'; END IF;
  ELSIF v_clave IN ('asignacion_version','version_rol_version',
                    'revision_control_version_rol','revision_catalogo_politicas') THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'number'
   OR (p_orden->>v_clave) !~ '^[1-9][0-9]{0,19}$'
   THEN RAISE EXCEPTION 'AD275: versión de lectura inválida' USING ERRCODE='22023'; END IF;
  ELSIF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'string'
   OR octet_length(p_orden->>v_clave) NOT BETWEEN 1 AND 512
  THEN RAISE EXCEPTION 'AD275: orden de lectura inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_orden->>'intento_ref' !~ '^lectura_[0-9a-f]{32}$'
 OR p_orden->>'recurso_ref' !~ '^mi-bolsa:can_[A-Za-z0-9_-]{22,128}$'
 OR p_orden->>'contexto_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'procedencia_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'autenticacion_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'lectura_revision_permisos' !~ '^[1-9][0-9]{0,19}$'
 OR p_orden->>'lectura_instantanea_sha256' !~ '^[0-9a-f]{64}$'
 OR (p_orden->>'lectura_revision_permisos')::numeric>18446744073709551615::numeric
 OR p_orden->>'accion' IS DISTINCT FROM 'bolsa.preferencias_bolsas.consultar_propias'
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'gestion_preferencias_bolsas_propias'
 OR p_orden->>'resultado' IS DISTINCT FROM 'obtenida'
 OR p_orden->>'motivo_ref' IS DISTINCT FROM 'lectura_preferencias_bolsas_correcta'
 OR p_orden->>'proceso' IS DISTINCT FROM 'vec-server'
 OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_orden->>'candidato_ref' !~ '^can_[A-Za-z0-9_-]{22,128}$'
 OR p_orden->>'recurso_ref' IS DISTINCT FROM 'mi-bolsa:'||(p_orden->>'candidato_ref')
 OR p_orden->>'tipo_recurso' IS DISTINCT FROM 'participaciones_candidato'
 OR p_orden->>'correlacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_orden->'campos_permitidos' IS DISTINCT FROM
    '["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","recibo_ref","registrado_en"]'::jsonb
 THEN RAISE EXCEPTION 'AD275: contrato de lectura inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  v_contexto:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
  v_vinculo:=convert_from(p_vinculo_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AD275: evidencia de lectura inválida' USING ERRCODE='22023';
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
 OR v_vinculo->>'metodo_observado' NOT IN ('certificado','dnie')
 OR v_vinculo->>'garantia_observada' IS DISTINCT FROM 'alto'
 OR encode(sha256(p_contexto_canonico),'hex') IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR jsonb_typeof(v_contexto->'vinculos') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'AD275: vínculo interno de lectura inválido' USING ERRCODE='42501'; END IF;
 SELECT count(*) INTO v_vinculos FROM jsonb_array_elements(v_contexto->'vinculos') x
 WHERE x->>'tipo'='candidato' AND x->>'estado'='activo';
 IF v_vinculos<>1 OR NOT EXISTS(
  SELECT 1 FROM jsonb_array_elements(v_contexto->'vinculos') x
  WHERE x->>'tipo'='candidato' AND x->>'estado'='activo'
   AND x->>'referencia'=p_orden->>'candidato_ref')
 OR vec_autorizacion.revalidar_sesion_vinculo_v2(v_vinculo,
  (p_orden->>'emitida_en')::timestamptz,(p_orden->>'valida_hasta')::timestamptz,
  clock_timestamp()) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD275: sesión o candidato interno denegado' USING ERRCODE='42501'; END IF;
 v_acreditada:=vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
  p_orden->>'registro_contexto_ref',v_contexto->>'esquema',p_orden->>'contexto_sha256',
  p_orden->>'procedencia_sha256',v_vinculo->>'autoridad_efectiva',
  v_contexto->>'cuenta_ref',(v_contexto->>'cuenta_version')::numeric,
  v_contexto->>'persona_ref',(v_contexto->>'persona_version')::numeric,
  v_contexto->>'perfil_activo_ref',(v_contexto->>'perfil_version')::numeric,
  v_contexto->>'contexto_actor_ref',(v_contexto->>'contexto_version')::numeric,
  v_contexto->>'metodo',v_contexto->>'garantia',
  (p_orden->>'emitida_en')::timestamptz,(p_orden->>'valida_hasta')::timestamptz);
 IF v_acreditada IS NULL
 OR vec_autorizacion.cotejar_lectura_preferencias_bolsas_actual_interna_v1(
  p_orden,v_contexto->>'principal_ref',v_contexto->>'perfil_activo_ref') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD275: autoridad actual de lectura denegada' USING ERRCODE='42501'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.lectura_preferencias_bolsas.v1')||
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
 IF FOUND THEN RAISE EXCEPTION 'AD275: lectura ya asentada' USING ERRCODE='23505'; END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 v_secuencia:=v_secuencia+1;v_instante:=clock_timestamp();
 v_ref:='aud_v3_lp_'||substr(p_orden->>'intento_ref',9,32);
 v_huella:=encode(sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.lectura_preferencias_bolsas.v1')||
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
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'lectura_preferencias_bolsas',
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
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_interna_v1(bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_interna_v1(bytea,bytea,jsonb)
 TO vec_bolsa_llamamientos_propietario;
-- Intento nominal interno. AD169 usa un registrador interno y no acepta este
-- LOGIN. Esta función solo toma sus reglas de cadena; AUT15 y CA12 acreditan
-- de nuevo la sesión y el candidato. No consulta la concesión GET.
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_intento_lectura_preferencias_bolsas_interna_v1(
 p_contexto_canonico bytea,p_vinculo_canonico bytea,p_orden jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,
 correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s'
SET statement_timeout='10s' AS $f$
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
 OR vec_autorizacion_atestada_v3.login_lectura_preferencias_bolsas_interna_valido_v1() IS NOT TRUE
 OR p_contexto_canonico IS NULL OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
 OR p_vinculo_canonico IS NULL OR octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_orden))<>array_length(v_claves,1)
 OR (p_orden ?& v_claves) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD275: intento interno sin emisor acreditado' USING ERRCODE='42501'; END IF;
 FOREACH v_clave IN ARRAY v_claves LOOP
  IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'string'
  OR octet_length(p_orden->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD275: orden de intento inválida' USING ERRCODE='22023'; END IF;
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
 OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_orden->>'correlacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 THEN RAISE EXCEPTION 'AD275: contrato de intento inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  v_contexto:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
  v_vinculo:=convert_from(p_vinculo_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AD275: evidencia de intento inválida' USING ERRCODE='22023'; END;
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
 OR v_vinculo->>'metodo_observado' NOT IN ('certificado','dnie')
 OR v_vinculo->>'garantia_observada' IS DISTINCT FROM 'alto'
 OR encode(sha256(p_contexto_canonico),'hex') IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR jsonb_typeof(v_contexto->'vinculos') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'AD275: vínculo interno de intento inválido' USING ERRCODE='42501'; END IF;
 v_emitida:=clock_timestamp();
 SELECT x->>'referencia' INTO v_candidato FROM jsonb_array_elements(v_contexto->'vinculos') x
 WHERE x->>'tipo'='candidato' AND x->>'estado'='activo';
 IF (SELECT count(*) FROM jsonb_array_elements(v_contexto->'vinculos') x
     WHERE x->>'tipo'='candidato' AND x->>'estado'='activo')<>1
 OR v_candidato IS NULL OR v_candidato !~ '^can_[A-Za-z0-9_-]{22,128}$'
 OR p_orden->>'recurso_ref' IS DISTINCT FROM
  'mi-bolsa:'||encode(sha256(convert_to('mi-bolsa:'||v_candidato,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'AD275: candidato interno denegado' USING ERRCODE='42501'; END IF;
 SELECT least(v_emitida+interval '1 second',
  (v_vinculo->>'sesion_valida_hasta')::timestamptz,
  (v_contexto->>'vigente_hasta')::timestamptz,
  (x->>'vigente_hasta')::timestamptz) INTO v_valida
 FROM jsonb_array_elements(v_contexto->'vinculos') x
 WHERE x->>'tipo'='candidato' AND x->>'estado'='activo';
 IF v_valida<=v_emitida OR vec_autorizacion.revalidar_sesion_vinculo_v2(
  v_vinculo,v_emitida,v_valida,clock_timestamp()) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD275: sesión interna denegada' USING ERRCODE='42501'; END IF;
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
  RAISE EXCEPTION 'AD275: contexto interno no acreditado' USING ERRCODE='42501'; END IF;
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
  THEN RAISE EXCEPTION 'AD275: intento incompatible' USING ERRCODE='23505'; END IF;
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
  correlacion_ref,vinculo_sha256,transaccion_origen)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'intento_nominal',
  p_orden->>'intento_ref',v_material_sha,v_contexto->>'principal_ref',
  v_contexto->>'perfil_activo_ref',p_orden->>'registro_contexto_ref',
  p_orden->>'contexto_sha256',p_orden->>'procedencia_sha256',
  p_orden->>'autenticacion_ref',p_orden->>'sesion_ref',p_orden->>'autenticacion_sha256',
  p_orden->>'accion',p_orden->>'modulo_id',p_orden->>'recurso_ref',
  p_orden->>'finalidad_ref',p_orden->>'resultado',p_orden->>'motivo_ref',
  p_orden->>'proceso',p_orden->>'canal',p_orden->>'correlacion_ref',
  encode(sha256(p_vinculo_canonico),'hex'),pg_current_xact_id());
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_orden->>'correlacion_ref',v_instante;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_lectura_preferencias_bolsas_interna_v1(bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_lectura_preferencias_bolsas_interna_v1(bytea,bytea,jsonb)
 TO vec_bolsa_llamamientos_propietario;
RESET ROLE;

DO $post$
BEGIN
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_interna_v1(bytea,bytea,jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_lectura_preferencias_bolsas_interna_v1(bytea,bytea,jsonb)') IS NULL
 OR has_function_privilege('public','vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_interna_v1(bytea,bytea,jsonb)','EXECUTE') IS TRUE
 OR has_function_privilege('public','vec_autorizacion_atestada_v3.registrar_intento_lectura_preferencias_bolsas_interna_v1(bytea,bytea,jsonb)','EXECUTE') IS TRUE
 OR has_function_privilege('vec_bolsa_llamamientos_propietario','vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_interna_v1(bytea,bytea,jsonb)','EXECUTE') IS NOT TRUE
 OR has_function_privilege('vec_bolsa_llamamientos_propietario','vec_autorizacion_atestada_v3.registrar_intento_lectura_preferencias_bolsas_interna_v1(bytea,bytea,jsonb)','EXECUTE') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD275: postimagen o ACL divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
