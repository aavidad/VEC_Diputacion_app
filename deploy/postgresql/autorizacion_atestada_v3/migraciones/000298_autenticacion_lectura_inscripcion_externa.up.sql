\set ON_ERROR_STOP on
-- AD298: la auditoría de la lectura de inscripción del aspirante externo coteja
-- la autenticación en el almacén del portal externo (IS22, F1-84). AD289 ya
-- cotejaba allí el contexto (CA47), pero la sesión seguía buscándose solo en el
-- almacén interno (IS20) y la lectura fallaba con P0002. Reconstruye solo
-- registrar_lectura_inscripcion_v1 desde su cuerpo instalado (AD289); el canal
-- interno conserva exactamente su cotejo. Mismo propietario, ACL,
-- configuración y firma; no toca el núcleo, otras familias ni datos.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000298',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
-- Solo metadatos de esta transacción; el operador DBA conserva su propiedad.
CREATE TEMP TABLE ad298_meta ON COMMIT DROP AS
 SELECT p.oid,to_jsonb(p)-'prosrc' AS meta,
  encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') AS huella
 FROM pg_proc p WHERE p.oid=to_regprocedure(
  'vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(bytea,bytea,jsonb)');
DO $pre$
DECLARE c regprocedure:=to_regprocedure('vec_identidad_sesiones_v1.cotejar_autenticacion_historica_portal_externo_v1(bytea)');
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR (SELECT count(*) FROM pg_temp.ad298_meta)<>1
 -- Posimagen exacta de AD289.
 OR NOT EXISTS(SELECT 1 FROM pg_temp.ad298_meta
  WHERE huella='34313f1d0796f4c7e5964c7f5d6415097a46c88261d4eb66c963ebe9c094f989'
   AND meta->>'proowner'=('vec_autorizacion_atestada_v3_propietario'::regrole::oid)::text)
 OR (SELECT p.proacl FROM pg_proc p WHERE p.oid=(SELECT oid FROM pg_temp.ad298_meta)) IS DISTINCT FROM
    ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario',
     'vec_bolsa_llamamientos_propietario=X/vec_autorizacion_atestada_v3_propietario']::aclitem[]
 -- IS22 instalada y solo ejecutable por su propietario y el de AD3.
 OR c IS NULL
 OR NOT has_function_privilege('vec_autorizacion_atestada_v3_propietario',c,'EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_propietario',c,'EXECUTE')
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=c)<>2
 THEN RAISE EXCEPTION 'AD298: preimagen, ACL o IS22 divergente; no reaplicar' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(p_contexto_canonico bytea, p_vinculo_canonico bytea, p_orden jsonb)
 RETURNS TABLE(auditoria_ref text, secuencia numeric, huella_sha256 text, correlacion_ref text, registrada_en timestamp with time zone)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '10s'
AS $function$
DECLARE
 v_contexto jsonb;v_vinculo jsonb;
 v_claves constant text[]:=ARRAY['intento_ref','registro_contexto_ref','contexto_sha256','procedencia_sha256',
  'autenticacion_ref','sesion_ref','autenticacion_sha256','accion','modulo_id','recurso_ref',
  'finalidad_ref','resultado','motivo_ref','proceso','canal','correlacion_ref',
  'lectura_revision_permisos','lectura_instantanea_sha256'];
 v_clave text;v_material bytea;v_material_sha text;v_anterior text;v_secuencia numeric;
 v_instante timestamptz(6);v_huella text;v_ref text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 OR vec_autorizacion_atestada_v3.login_lector_inscripciones_valido_v1(
     p_orden->>'accion',p_orden->>'canal') IS NOT TRUE
 OR p_contexto_canonico IS NULL OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
 OR p_vinculo_canonico IS NULL OR octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_orden))<>18
 OR (p_orden ?& v_claves) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD228: lectura sin emisor acreditado' USING ERRCODE='42501';END IF;
 FOREACH v_clave IN ARRAY v_claves LOOP
  IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'string'
  OR octet_length(p_orden->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD228: orden de lectura inválida' USING ERRCODE='22023';END IF;
 END LOOP;
 IF p_orden->>'intento_ref' !~ '^lectura_[0-9a-f]{32}$'
 OR p_orden->>'contexto_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'procedencia_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'autenticacion_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'lectura_revision_permisos' !~ '^[1-9][0-9]{0,19}$'
 OR p_orden->>'lectura_instantanea_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'modulo_id'<>'bolsa' OR p_orden->>'proceso'<>'vec-server'
 OR vec_autorizacion_atestada_v3.contrato_lectura_inscripcion_v1(
      p_orden->>'accion',p_orden->>'finalidad_ref',p_orden->>'canal',p_orden->>'recurso_ref') IS NOT TRUE
 OR NOT ((p_orden->>'resultado'='obtenida' AND p_orden->>'motivo_ref'='inscripcion_lectura_correcta')
   OR (p_orden->>'resultado'='no_encontrada' AND p_orden->>'motivo_ref'='inscripcion_no_encontrada'))
 OR p_orden->>'correlacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 THEN RAISE EXCEPTION 'AD228: contrato de lectura inválido' USING ERRCODE='22023';END IF;
 IF (p_orden->>'lectura_revision_permisos')::numeric > 18446744073709551615::numeric
 THEN RAISE EXCEPTION 'AD228: revisión de permisos fuera de rango' USING ERRCODE='22023';END IF;
 BEGIN v_contexto:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
       v_vinculo:=convert_from(p_vinculo_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD228: material de lectura inválido' USING ERRCODE='22023';END;
 IF jsonb_typeof(v_contexto) IS DISTINCT FROM 'object' OR jsonb_typeof(v_vinculo) IS DISTINCT FROM 'object'
 OR v_contexto->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
 OR v_vinculo->>'esquema' IS DISTINCT FROM 'vec.autenticacion-actor.vinculo.v2.contexto-registrado'
 OR v_vinculo->>'bloque_version' IS DISTINCT FROM '2'
 OR v_vinculo->>'principal_id' IS DISTINCT FROM v_contexto->>'principal_ref'
 OR v_vinculo->>'perfil_activo_ref' IS DISTINCT FROM v_contexto->>'perfil_activo_ref'
 OR v_vinculo->>'cuenta_ref' IS DISTINCT FROM v_contexto->>'cuenta_ref'
 OR v_vinculo->>'metodo_observado' IS DISTINCT FROM v_contexto->>'metodo'
 OR v_vinculo->>'garantia_observada' IS DISTINCT FROM v_contexto->>'garantia'
 OR v_vinculo->>'contexto_actor_esquema' IS DISTINCT FROM v_contexto->>'esquema'
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
 OR v_vinculo->>'superficie' IS DISTINCT FROM p_orden->>'canal'
 OR (((v_contexto->>'resuelto_en')::timestamptz >=
      (v_vinculo->>'sesion_revalidada_en')::timestamptz)
     AND ((v_contexto->>'resuelto_en')::timestamptz <
      (v_vinculo->>'sesion_valida_hasta')::timestamptz)) IS NOT TRUE
 OR encode(sha256(p_contexto_canonico),'hex') IS DISTINCT FROM p_orden->>'contexto_sha256'
 -- AD289: el portal externo coteja su almacén propio (CA47); RRHH sigue en CA26/CA27.
 OR (CASE p_orden->>'canal'
      WHEN 'externa_personal' THEN vec_contexto_actor_v1.cotejar_contexto_historico_portal_externo_v1(
       p_orden->>'registro_contexto_ref',p_orden->>'contexto_sha256',
       p_orden->>'procedencia_sha256',p_contexto_canonico)
      ELSE vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(
       p_orden->>'registro_contexto_ref',p_orden->>'contexto_sha256',
       p_orden->>'procedencia_sha256',p_contexto_canonico) END) IS NOT TRUE
 -- AD298: la autenticación del portal externo se coteja en su almacén (IS22).
 OR (CASE p_orden->>'canal'
      WHEN 'externa_personal' THEN vec_identidad_sesiones_v1.cotejar_autenticacion_historica_portal_externo_v1(
       p_vinculo_canonico)
      ELSE vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(
       p_vinculo_canonico) END) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD228: evidencia histórica de lectura inválida' USING ERRCODE='42501';END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.lectura_inscripcion.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_contexto_canonico),'hex'))||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_vinculo_canonico),'hex'));
 FOREACH v_clave IN ARRAY v_claves LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_orden->>v_clave);
 END LOOP;
 v_material_sha:=encode(sha256(v_material),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:lectura:'||(p_orden->>'intento_ref'),0));
 PERFORM 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
 WHERE a.intento_ref=p_orden->>'intento_ref';
 IF FOUND THEN
  RAISE EXCEPTION 'AD228: lectura ya asentada' USING ERRCODE='23505';
 END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 v_secuencia:=v_secuencia+1;v_instante:=clock_timestamp();
 v_ref:='aud_v3_li_'||substr(p_orden->>'intento_ref',9,32);
 v_huella:=encode(sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.lectura_inscripcion.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(to_char(v_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  intento_ref,intento_material_sha256,actor_ref,perfil_activo_ref,registro_contexto_ref,
  contexto_sha256,procedencia_sha256,autenticacion_ref,sesion_ref,autenticacion_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,
  correlacion_ref,vinculo_sha256,transaccion_origen,
  lectura_revision_permisos,lectura_instantanea_sha256)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'lectura_inscripcion',
  p_orden->>'intento_ref',v_material_sha,v_contexto->>'principal_ref',v_contexto->>'perfil_activo_ref',
  p_orden->>'registro_contexto_ref',p_orden->>'contexto_sha256',p_orden->>'procedencia_sha256',
  p_orden->>'autenticacion_ref',p_orden->>'sesion_ref',p_orden->>'autenticacion_sha256',
  p_orden->>'accion',p_orden->>'modulo_id',p_orden->>'recurso_ref',p_orden->>'finalidad_ref',
  p_orden->>'resultado',p_orden->>'motivo_ref',p_orden->>'proceso',p_orden->>'canal',
  p_orden->>'correlacion_ref',encode(sha256(p_vinculo_canonico),'hex'),pg_current_xact_id(),
  (p_orden->>'lectura_revision_permisos')::numeric,p_orden->>'lectura_instantanea_sha256');
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_orden->>'correlacion_ref',v_instante;
END $function$;

RESET ROLE;
DO $post$
DECLARE
 firma constant regprocedure:='vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(bytea,bytea,jsonb)'::regprocedure;
 anterior jsonb; actual jsonb;
BEGIN
 SELECT meta INTO STRICT anterior FROM pg_temp.ad298_meta;
 SELECT to_jsonb(p)-'prosrc' INTO STRICT actual FROM pg_proc p WHERE p.oid=firma;
 IF actual IS DISTINCT FROM anterior
 OR encode(sha256(convert_to(pg_get_functiondef(firma),'UTF8')),'hex')
  IS DISTINCT FROM '3f8dcdf992d79a3dba5114bc6552f50887eb390d3e3ed1a6af276801050249be' THEN
  RAISE EXCEPTION 'AD298: posimagen o metadatos divergentes' USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
