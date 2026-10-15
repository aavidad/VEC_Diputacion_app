\set ON_ERROR_STOP on
-- AD277: autoridad documental de Bolsa y prerequisito D20 tras AD263/AD265.
-- Conserva el consumidor documental de Méritos de AD273C. No publica actos
-- de negocio ni crea LOGIN: requieren gobierno, AUT95, D20 y D22.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000277',0));
DO $pre$
DECLARE p oid; c oid;
BEGIN
 p:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 c:=(SELECT oid FROM pg_constraint WHERE conrelid=to_regclass('vec_autorizacion_atestada_v3.clave_capacidad_version')
   AND conname='clave_capacidad_version_audiencia_consumo_check');
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR p IS NULL OR c IS NULL
 OR (SELECT encode(sha256(convert_to(pg_get_functiondef(p),'UTF8')),'hex'))
   IS DISTINCT FROM '3e3f87faab57f26dce85d85fb71d2db45692f8bcdde8d4f273f1aa6ee1118986'
 OR (SELECT encode(sha256(convert_to(pg_get_constraintdef(c),'UTF8')),'hex'))
   IS DISTINCT FROM '36d55a6098e8e6c334ad8dba43aaef77b16f27b32424ff6b2673ec4cdfb54503'
 OR to_regprocedure('vec_autorizacion.revalidar_sesion_documental_bolsa_v1(jsonb,timestamptz,timestamptz,timestamptz)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion.revalidar_sesion_documental_bolsa_rrhh_v1(jsonb,timestamptz,timestamptz,timestamptz)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.login_documentos_bolsa_v1(boolean)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_subida_local_bolsa_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regrole('vec_autorizacion_propietario') IS NULL
 OR to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
 OR to_regrole('vec_documentos_propietario') IS NULL
 OR to_regrole('vec_documentos_subida_datos') IS NOT NULL
 OR to_regrole('vec_documentos_frontera_externa') IS NOT NULL
 OR encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion_atestada_v3.leer_configuracion_externa_v1(text,jsonb)'::regprocedure),'UTF8')),'hex')
   IS DISTINCT FROM '9ba9b9da65cc79b5f029b7b23d85039a364ad58271d0cddd790f803db872fed5'
 OR encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1(text,jsonb)'::regprocedure),'UTF8')),'hex')
   IS DISTINCT FROM 'f11ba3e5b7e5eef28c5b5ff8f46860f52080b7f0b74b951925d6a7bdd902135b'
 OR encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(jsonb)'::regprocedure),'UTF8')),'hex')
   IS DISTINCT FROM '8729e562d090fad7e171950d589cda4d718338fad8eb30b1650755a9780acc38'
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=to_regprocedure(
   'vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)')
   AND p.proowner='vec_identidad_sesiones_v1_propietario'::regrole AND p.prosecdef AND
   encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')=
    '374711632248d198f2e9ac54525f82e382dba2c42e80bfea0d942bacd293274b' AND
   p.proacl::text='{vec_identidad_sesiones_v1_propietario=X/vec_identidad_sesiones_v1_propietario,vec_identidad_sesiones_v1_revalidador=X/vec_identidad_sesiones_v1_propietario}')
 THEN RAISE EXCEPTION 'AD277: preimagen AD263/AD265 incompatible' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=p AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND prosecdef AND provolatile='v' AND proacl::text=
   '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}')
 OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND relowner='vec_autorizacion_atestada_v3_propietario'::regrole AND relrowsecurity AND relforcerowsecurity)
 THEN RAISE EXCEPTION 'AD277: propietario o RLS incompatibles' USING ERRCODE='55000'; END IF;
END $pre$;

GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(
 text,text,text,text,text,text,boolean,text,text,text,text,text,
 timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)
 TO vec_autorizacion_propietario;

CREATE ROLE vec_documentos_subida_datos NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_documentos_frontera_externa NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $db$ BEGIN
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_documentos_subida_datos,vec_documentos_frontera_externa',current_database());
END $db$;

SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.revalidar_sesion_documental_bolsa_rrhh_v1(
 p_vinculo jsonb,p_emitida_en timestamptz,p_valida_hasta timestamptz,p_instante timestamptz)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF current_user<>'vec_autorizacion_propietario' OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' OR
  jsonb_typeof(p_vinculo) IS DISTINCT FROM 'object' OR
  vec_autorizacion.vinculo_contexto_actor_v2_valido(p_vinculo) IS NOT TRUE OR
  session_user<>'vec_interno_documentos_bolsa_rrhh' OR
  p_vinculo->>'superficie' IS DISTINCT FROM 'interna_corporativa' OR
  p_vinculo->>'cuenta_privilegiada' IS DISTINCT FROM 'false' OR
  p_vinculo->>'metodo_observado' NOT IN ('certificado','dnie') OR
  p_vinculo->>'garantia_observada' NOT IN ('alto','sustancial') OR
  p_emitida_en IS NULL OR p_valida_hasta IS NULL OR p_instante IS NULL OR
  p_valida_hasta<=p_emitida_en OR p_instante<p_emitida_en OR p_instante>=p_valida_hasta
 THEN RETURN false; END IF;
 IF vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(
   p_vinculo->>'autenticacion_ref',p_vinculo->>'autenticacion_huella_sha256',
   p_vinculo->>'asercion_ref',p_vinculo->>'sesion_ref',p_vinculo->>'cuenta_ref',
   p_vinculo->>'cuenta_ordinaria_ref',(p_vinculo->>'cuenta_privilegiada')::boolean,
   p_vinculo->>'superficie',p_vinculo->>'metodo_observado',p_vinculo->>'garantia_observada',
   p_vinculo->>'politica_garantia_ref',p_vinculo->>'politica_garantia_huella_sha256',
   (p_vinculo->>'autenticacion_verificada_en')::timestamptz,
   (p_vinculo->>'sesion_emitida_en')::timestamptz,
   p_vinculo->>'control_sesion_ref',p_vinculo->>'control_sesion_revision','activa',
   p_vinculo->>'control_sesion_huella_sha256',
   (p_vinculo->>'sesion_revalidada_en')::timestamptz,
   (p_vinculo->>'sesion_valida_hasta')::timestamptz) IS NOT TRUE
 THEN RETURN false; END IF;
 RETURN p_emitida_en >= (p_vinculo->>'sesion_revalidada_en')::timestamptz
  AND p_valida_hasta <= (p_vinculo->>'sesion_valida_hasta')::timestamptz
  AND p_instante >= (p_vinculo->>'sesion_revalidada_en')::timestamptz
  AND p_instante < (p_vinculo->>'sesion_valida_hasta')::timestamptz;
EXCEPTION WHEN data_exception OR invalid_text_representation OR datetime_field_overflow THEN
 RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.revalidar_sesion_documental_bolsa_rrhh_v1(jsonb,timestamptz,timestamptz,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.revalidar_sesion_documental_bolsa_rrhh_v1(jsonb,timestamptz,timestamptz,timestamptz)
 TO vec_autorizacion_atestada_v3_propietario;

CREATE FUNCTION vec_autorizacion.revalidar_sesion_documental_bolsa_v1(
 p_vinculo jsonb,p_emitida_en timestamptz,p_valida_hasta timestamptz,p_instante timestamptz)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF current_user<>'vec_autorizacion_propietario' OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' OR
  jsonb_typeof(p_vinculo) IS DISTINCT FROM 'object' OR
  vec_autorizacion.vinculo_contexto_actor_v2_valido(p_vinculo) IS NOT TRUE OR
  p_vinculo->>'superficie' IS DISTINCT FROM 'externa_personal' OR
  p_vinculo->>'cuenta_privilegiada' IS DISTINCT FROM 'false' OR
  p_vinculo->>'metodo_observado' NOT IN ('certificado','dnie') OR
  p_vinculo->>'garantia_observada' IS DISTINCT FROM 'alto' OR
  p_emitida_en IS NULL OR p_valida_hasta IS NULL OR p_instante IS NULL OR
  p_valida_hasta<=p_emitida_en OR p_instante<p_emitida_en OR p_instante>=p_valida_hasta OR
  session_user<>'vec_externo_documentos_bolsa_datos' OR
  NOT EXISTS(SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
   JOIN pg_roles g ON g.oid=m.roleid WHERE l.rolname=session_user AND l.rolcanlogin
   AND l.rolinherit AND l.rolconfig IS NULL AND NOT(l.rolsuper OR l.rolcreatedb OR
    l.rolcreaterole OR l.rolreplication OR l.rolbypassrls) AND
    g.rolname='vec_documentos_bolsa_subida_datos' AND NOT g.rolcanlogin AND
    g.rolconfig IS NULL AND NOT(g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR
    g.rolreplication OR g.rolbypassrls) AND m.inherit_option AND NOT m.set_option
    AND NOT m.admin_option) OR
  (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1 OR
  EXISTS(SELECT 1 FROM pg_auth_members WHERE member=to_regrole('vec_documentos_bolsa_subida_datos'))
 THEN RETURN false; END IF;
 IF vec_identidad_externa_v1.acreditar_sesion_externa_v1(
   p_vinculo->>'autenticacion_ref',p_vinculo->>'autenticacion_huella_sha256',
   p_vinculo->>'asercion_ref',p_vinculo->>'sesion_ref',p_vinculo->>'cuenta_ref',
   p_vinculo->>'cuenta_ordinaria_ref',(p_vinculo->>'cuenta_privilegiada')::boolean,
   p_vinculo->>'superficie',p_vinculo->>'metodo_observado',p_vinculo->>'garantia_observada',
   p_vinculo->>'politica_garantia_ref',p_vinculo->>'politica_garantia_huella_sha256',
   (p_vinculo->>'autenticacion_verificada_en')::timestamptz,
   (p_vinculo->>'sesion_emitida_en')::timestamptz,
   p_vinculo->>'control_sesion_ref',p_vinculo->>'control_sesion_revision','activa',
   p_vinculo->>'control_sesion_huella_sha256',
   (p_vinculo->>'sesion_revalidada_en')::timestamptz,
   (p_vinculo->>'sesion_valida_hasta')::timestamptz) IS NOT TRUE
 THEN RETURN false; END IF;
 RETURN p_emitida_en >= (p_vinculo->>'sesion_revalidada_en')::timestamptz
  AND p_valida_hasta <= (p_vinculo->>'sesion_valida_hasta')::timestamptz
  AND p_instante >= (p_vinculo->>'sesion_revalidada_en')::timestamptz
  AND p_instante < (p_vinculo->>'sesion_valida_hasta')::timestamptz;
EXCEPTION WHEN data_exception OR invalid_text_representation OR datetime_field_overflow THEN
 RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.revalidar_sesion_documental_bolsa_v1(jsonb,timestamptz,timestamptz,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.revalidar_sesion_documental_bolsa_v1(jsonb,timestamptz,timestamptz,timestamptz)
 TO vec_autorizacion_atestada_v3_propietario;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.login_documentos_bolsa_v1(p_rrhh boolean)
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE login_esperado text; grupo_esperado text;
BEGIN
 login_esperado:=CASE WHEN p_rrhh THEN 'vec_interno_documentos_bolsa_rrhh'
  ELSE 'vec_externo_documentos_bolsa_datos' END;
 grupo_esperado:=CASE WHEN p_rrhh THEN 'vec_documentos_bolsa_rrhh'
  ELSE 'vec_documentos_bolsa_subida_datos' END;
 RETURN current_user='vec_autorizacion_atestada_v3_propietario'
  AND session_user=login_esperado AND current_setting('role')='none'
  AND EXISTS(SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
   JOIN pg_roles g ON g.oid=m.roleid WHERE l.rolname=login_esperado AND l.rolcanlogin
    AND l.rolinherit AND l.rolconfig IS NULL AND NOT(l.rolsuper OR l.rolcreatedb OR
     l.rolcreaterole OR l.rolreplication OR l.rolbypassrls) AND g.rolname=grupo_esperado
    AND NOT g.rolcanlogin AND g.rolconfig IS NULL AND NOT(g.rolsuper OR g.rolcreatedb OR
     g.rolcreaterole OR g.rolreplication OR g.rolbypassrls) AND m.inherit_option
    AND NOT m.set_option AND NOT m.admin_option)
  AND (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)=1
  AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=to_regrole(grupo_esperado))
  AND NOT EXISTS(SELECT 1 FROM pg_db_role_setting WHERE setrole IN
   (session_user::regrole,to_regrole(grupo_esperado)));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.login_documentos_bolsa_v1(boolean) FROM PUBLIC;

-- La reconstrucción parte de la definición instalada, cuya huella se ha fijado
-- arriba. Cada marca se comprueba una vez; ninguna otra rama se reemplaza.
DO $patch$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 definicion text; vieja text; nueva text; marca text;
BEGIN
 definicion:=pg_get_functiondef(f);
 marca:=$m$           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_cartera_retirada'$m$;
 nueva:=$m$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'documentos_subida_local_externa'
               AND session_user='vec_externo_documentos_datos'
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
                 AND NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls))
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                 AND m.roleid='vec_documentos_subida_datos'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_documentos_subida_datos'::regrole)
           )
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'documentos_bolsa_externa'
            AND vec_autorizacion_atestada_v3.login_documentos_bolsa_v1(false) IS TRUE)
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'documentos_bolsa_rrhh'
            AND vec_autorizacion_atestada_v3.login_documentos_bolsa_v1(true) IS TRUE)
$m$||marca;
 IF length(definicion)-length(replace(definicion,marca,''))<>length(marca) THEN
  RAISE EXCEPTION 'AD277: marca de perfil nominal incompatible' USING ERRCODE='55000'; END IF;
 definicion:=replace(definicion,marca,nueva);
 marca:=$m$               AND p_perfil_mutacion IS DISTINCT FROM 'meritos_cartera_retirada'$m$;
 nueva:=$m$               AND p_perfil_mutacion IS DISTINCT FROM 'documentos_subida_local_externa'
               AND p_perfil_mutacion IS DISTINCT FROM 'documentos_bolsa_externa'
               AND p_perfil_mutacion IS DISTINCT FROM 'documentos_bolsa_rrhh'
$m$||marca;
 IF length(definicion)-length(replace(definicion,marca,''))<>length(marca) THEN
  RAISE EXCEPTION 'AD277: marca de exclusión incompatible' USING ERRCODE='55000'; END IF;
 definicion:=replace(definicion,marca,nueva);
 marca:=$m$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_cartera_retirada'
$m$;
 nueva:=$m$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'documentos_subida_local_externa'
 AND c->>'operacion' IN ('meritos.cartera.comprobante.subir_propio','meritos.cartera.comprobante.descargar_propio')
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM (CASE c->>'operacion'
  WHEN 'meritos.cartera.comprobante.subir_propio' THEN 'vec_documentos.subida_local.acto_modulo.v1'
  ELSE 'vec_documentos.subida_local.descarga_propia.v1' END)
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'meritos'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'hecho'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'custodiar_evidencia_meritos'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'recurso_ref' ~ '^ref:[0-9a-f]{64}$'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM (CASE c->>'operacion'
  WHEN 'meritos.cartera.comprobante.subir_propio' THEN '["contenido","estado","recibo"]'::jsonb
  ELSE '["contenido"]'::jsonb END)
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IN ('documentos_bolsa_externa','documentos_bolsa_rrhh')
 AND c->>'operacion' IN ('bolsa.justificante_renuncia.subir_propio',
   'bolsa.justificante_renuncia.descargar_propio','bolsa.justificante_renuncia.descargar_rrhh')
 AND ((p_perfil_mutacion='documentos_bolsa_externa' AND
      c->>'operacion' IN ('bolsa.justificante_renuncia.subir_propio',
       'bolsa.justificante_renuncia.descargar_propio')) OR
      (p_perfil_mutacion='documentos_bolsa_rrhh' AND
       c->>'operacion'='bolsa.justificante_renuncia.descargar_rrhh'))
 AND ((c->>'operacion'='bolsa.justificante_renuncia.subir_propio' AND
   c->>'audiencia_consumo' IN ('vec_documentos.bolsa.justificante_renuncia.preparar_subida.v1',
    'vec_documentos.bolsa.justificante_renuncia.subida.v1')) OR
  (c->>'operacion'='bolsa.justificante_renuncia.descargar_propio' AND
   c->>'audiencia_consumo'='vec_documentos.bolsa.justificante_renuncia.descarga_propia.v1') OR
  (c->>'operacion'='bolsa.justificante_renuncia.descargar_rrhh' AND
   c->>'audiencia_consumo' IN ('vec_documentos.bolsa.justificante_renuncia.preparar_descarga_rrhh.v1',
    'vec_documentos.bolsa.justificante_renuncia.descarga_rrhh.v1')))
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'justificante_renuncia'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'custodiar_justificante_renuncia_bolsa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM
  (CASE WHEN p_perfil_mutacion='documentos_bolsa_externa' THEN 'externa_personal'
   ELSE 'interna_corporativa' END)
 AND (p_perfil_mutacion<>'documentos_bolsa_externa' OR
  (d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'false'
   AND d#>>'{vinculo_autenticacion_actor,garantia_observada}' IS NOT DISTINCT FROM 'alto'))
 AND (p_perfil_mutacion<>'documentos_bolsa_rrhh' OR
  d#>>'{vinculo_autenticacion_actor,garantia_observada}' IN ('alto','sustancial'))
 AND d->'campos_permitidos' IS NOT DISTINCT FROM
  (CASE WHEN c->>'operacion'='bolsa.justificante_renuncia.subir_propio'
   THEN '["contenido","estado","recibo"]'::jsonb ELSE '["contenido"]'::jsonb END)
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$m$||marca;
 IF length(definicion)-length(replace(definicion,marca,''))<>length(marca) THEN
  RAISE EXCEPTION 'AD277: marca de acto incompatible' USING ERRCODE='55000'; END IF;
 definicion:=replace(definicion,marca,nueva);
 marca:=$m$    IF p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_cartera_retirada'
       AND vec_autorizacion.revalidar_sesion_cartera_meritos_v1($m$;
 nueva:=$m$    IF p_perfil_mutacion IN ('meritos_cartera_retirada','documentos_subida_local_externa')
       AND vec_autorizacion.revalidar_sesion_cartera_meritos_v1($m$;
 IF length(definicion)-length(replace(definicion,marca,''))<>length(marca) THEN
  RAISE EXCEPTION 'AD277: marca sesión Méritos incompatible' USING ERRCODE='55000'; END IF;
 definicion:=replace(definicion,marca,nueva);
 marca:=$m$    IF p_perfil_mutacion IN ('meritos_cartera_retirada','documentos_subida_local_externa')
       AND vec_autorizacion.revalidar_sesion_cartera_meritos_v1($m$;
 nueva:=$m$    IF p_perfil_mutacion IS NOT DISTINCT FROM 'documentos_bolsa_externa'
       AND vec_autorizacion.revalidar_sesion_documental_bolsa_v1(
         d->'vinculo_autenticacion_actor',
         (c->>'emitida_en')::timestamptz,(c->>'expira_en')::timestamptz,
         clock_timestamp()) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='sesión documental Bolsa denegada';
    END IF;
    IF p_perfil_mutacion IS NOT DISTINCT FROM 'documentos_bolsa_rrhh'
       AND vec_autorizacion.revalidar_sesion_documental_bolsa_rrhh_v1(
         d->'vinculo_autenticacion_actor',
         (c->>'emitida_en')::timestamptz,(c->>'expira_en')::timestamptz,
         clock_timestamp()) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='sesión RRHH documental Bolsa denegada';
    END IF;
$m$||marca;
 IF length(definicion)-length(replace(definicion,marca,''))<>length(marca) THEN
  RAISE EXCEPTION 'AD277: marca de sesión incompatible' USING ERRCODE='55000'; END IF;
 definicion:=replace(definicion,marca,nueva);
 marca:=$m$    IF p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_cartera_retirada' THEN
        SELECT * INTO v_registro$m$;
 nueva:=$m$    IF p_perfil_mutacion IN ('meritos_cartera_retirada','documentos_subida_local_externa') THEN
        SELECT * INTO v_registro$m$;
 IF length(definicion)-length(replace(definicion,marca,''))<>length(marca) THEN
  RAISE EXCEPTION 'AD277: marca registrador Méritos incompatible' USING ERRCODE='55000'; END IF;
 definicion:=replace(definicion,marca,nueva);
 marca:=$m$    IF p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_cartera_retirada' THEN
        v_revalidada_en:=vec_autorizacion.revalidar_decision_meritos_externo_v3_viva($m$;
 nueva:=$m$    IF p_perfil_mutacion IN ('meritos_cartera_retirada','documentos_subida_local_externa') THEN
        v_revalidada_en:=vec_autorizacion.revalidar_decision_meritos_externo_v3_viva($m$;
 IF length(definicion)-length(replace(definicion,marca,''))<>length(marca) THEN
  RAISE EXCEPTION 'AD277: marca revalidador Méritos incompatible' USING ERRCODE='55000'; END IF;
 definicion:=replace(definicion,marca,nueva);
 EXECUTE definicion;
END $patch$;

-- Las audiencias nacen sin claves; la publicación posterior exige actos
-- de gobierno y versiones explícitas de los perfiles de negocio.
DO $audiencias$
DECLARE anterior text; ampliada text; marca text;
BEGIN
 SELECT pg_get_constraintdef(oid) INTO STRICT anterior FROM pg_constraint WHERE
  conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND
  conname='clave_capacidad_version_audiencia_consumo_check';
 marca:=$m$'vec_documentos.operacion.v1'::text$m$;
 ampliada:=replace(anterior,marca,marca||$m$, 'vec_documentos.subida_local.acto_modulo.v1'::text,
  'vec_documentos.subida_local.descarga_propia.v1'::text,
  'vec_documentos.bolsa.justificante_renuncia.preparar_subida.v1'::text,
  'vec_documentos.bolsa.justificante_renuncia.subida.v1'::text,
  'vec_documentos.bolsa.justificante_renuncia.descarga_propia.v1'::text,
  'vec_documentos.bolsa.justificante_renuncia.preparar_descarga_rrhh.v1'::text,
  'vec_documentos.bolsa.justificante_renuncia.descarga_rrhh.v1'::text$m$);
 IF length(anterior)-length(replace(anterior,marca,''))<>length(marca) THEN
  RAISE EXCEPTION 'AD277: restricción sin marca única' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
  DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||ampliada;
END $audiencias$;

-- AD263/AD265 dejan 23 claves externas. Se conservan y se añaden dos
-- audiencias documentales de Méritos y tres de Documentos Bolsa.
DO $config$
DECLARE nombre text; f oid; definicion text; marca text; nueva text;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['leer_configuracion_externa_v1(text,jsonb)',
  'comprobar_material_emision_externa_v1(text,jsonb)'] LOOP
  f:=to_regprocedure('vec_autorizacion_atestada_v3.'||nombre);
  definicion:=pg_get_functiondef(f);
  marca:=$m$            'vec_meritos.hecho.declarar.v1']$m$;
  nueva:=$m$            'vec_meritos.hecho.declarar.v1',
            'vec_documentos.subida_local.acto_modulo.v1',
            'vec_documentos.subida_local.descarga_propia.v1']
        WHEN 'bolsa_justificantes' THEN ARRAY[
            'vec_documentos.bolsa.justificante_renuncia.preparar_subida.v1',
            'vec_documentos.bolsa.justificante_renuncia.subida.v1',
            'vec_documentos.bolsa.justificante_renuncia.descarga_propia.v1']$m$;
  IF length(definicion)-length(replace(definicion,marca,''))<>length(marca) THEN
   RAISE EXCEPTION 'AD277: configuración externa incompatible: %',nombre USING ERRCODE='55000'; END IF;
  EXECUTE replace(definicion,marca,nueva);
 END LOOP;
 f:='vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(jsonb)'::regprocedure;
 definicion:=pg_get_functiondef(f);
 marca:=$m$'vec_meritos.hecho.declarar.v1'];$m$;
 nueva:=$m$'vec_meritos.hecho.declarar.v1',
 'vec_documentos.subida_local.acto_modulo.v1',
 'vec_documentos.subida_local.descarga_propia.v1',
 'vec_documentos.bolsa.justificante_renuncia.preparar_subida.v1',
 'vec_documentos.bolsa.justificante_renuncia.subida.v1',
 'vec_documentos.bolsa.justificante_renuncia.descarga_propia.v1'];$m$;
 IF length(definicion)-length(replace(definicion,marca,''))<>length(marca) THEN
  RAISE EXCEPTION 'AD277: publicador externo incompatible' USING ERRCODE='55000'; END IF;
 definicion:=replace(definicion,marca,nueva);
 IF length(definicion)-length(replace(definicion,$m$jsonb_array_length(m->'claves')<>23$m$,''))
   <>length($m$jsonb_array_length(m->'claves')<>23$m$) OR
  length(definicion)-length(replace(definicion,$m$audiencia_consumo') FROM pg_catalog.jsonb_array_elements(m->'claves') AS u(item))<>23$m$, ''))
   <>length($m$audiencia_consumo') FROM pg_catalog.jsonb_array_elements(m->'claves') AS u(item))<>23$m$)
 THEN RAISE EXCEPTION 'AD277: conteo de claves externo incompatible' USING ERRCODE='55000'; END IF;
 definicion:=replace(definicion,$m$jsonb_array_length(m->'claves')<>23$m$,
  $m$jsonb_array_length(m->'claves')<>28$m$);
 definicion:=replace(definicion,$m$audiencia_consumo') FROM pg_catalog.jsonb_array_elements(m->'claves') AS u(item))<>23$m$,
  $m$audiencia_consumo') FROM pg_catalog.jsonb_array_elements(m->'claves') AS u(item))<>28$m$);
 EXECUTE definicion;
END $config$;

-- Consumidor D20 extraído literalmente de AD273C para que AUT95 y D20
-- puedan instalarse antes de la consolidación de Cronos. Solo Méritos.
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_subida_local_modulo_v1(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; v record; accion text;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' OR
  session_user<>'vec_externo_documentos_datos' OR
  NOT EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
   AND m.roleid='vec_documentos_subida_datos'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option) OR
  (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)<>1
  OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE
    p.oid=to_regprocedure('vec_autorizacion.acciones_meritos_cartera_v2()')
    AND p.proowner='vec_autorizacion_propietario'::regrole)
  OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE
    p.oid=to_regprocedure('vec_autorizacion.decision_meritos_externa_exacta_v1(jsonb,jsonb)')
    AND p.proowner='vec_autorizacion_propietario'::regrole)
  OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE
    p.oid=to_regprocedure('vec_autorizacion.revalidar_decision_meritos_externo_v3_viva(bytea,bytea,numeric,numeric)')
    AND p.proowner='vec_autorizacion_propietario'::regrole)
 THEN RAISE EXCEPTION 'AD270: consumidor nominal denegado' USING ERRCODE='42501'; END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AD270: material inválido' USING ERRCODE='22023'; END;
 accion:=d->>'accion';
 IF accion IS NULL OR accion NOT IN ('meritos.cartera.comprobante.subir_propio','meritos.cartera.comprobante.descargar_propio') OR
  c->>'operacion' IS DISTINCT FROM accion OR
  c->>'audiencia_consumo' IS DISTINCT FROM (CASE accion
   WHEN 'meritos.cartera.comprobante.subir_propio' THEN 'vec_documentos.subida_local.acto_modulo.v1'
   ELSE 'vec_documentos.subida_local.descarga_propia.v1' END) OR
  d->>'modulo_id' IS DISTINCT FROM 'meritos' OR d->>'tipo_recurso' IS DISTINCT FROM 'hecho' OR
  d->>'finalidad' IS DISTINCT FROM 'custodiar_evidencia_meritos' OR
  d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref' OR
  d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' OR
  d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'externa_personal' OR
  d->'campos_permitidos' IS DISTINCT FROM (CASE accion
   WHEN 'meritos.cartera.comprobante.subir_propio' THEN '["contenido","estado","recibo"]'::jsonb
   ELSE '["contenido"]'::jsonb END) OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD270: acto no corresponde' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'documentos_subida_local_externa',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS DISTINCT FROM TRUE OR v.decision_ref IS DISTINCT FROM c->>'decision_ref' OR
  v.efecto_ref IS DISTINCT FROM c->>'efecto_ref' OR v.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'AD270: consumo incoherente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT v.decision_ref,v.efecto_ref,v.huella_efecto_sha256,v.consumo_huella_sha256,
  v.auditoria_ref,v.consumida_en,v.consumo_nuevo;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_subida_local_modulo_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_subida_local_modulo_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_subida_local_bolsa_v1(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; v record; accion text; perfil text;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' OR
  (vec_autorizacion_atestada_v3.login_documentos_bolsa_v1(false) IS NOT TRUE AND
   vec_autorizacion_atestada_v3.login_documentos_bolsa_v1(true) IS NOT TRUE)
 THEN RAISE EXCEPTION 'AD277: consumidor documental denegado' USING ERRCODE='42501'; END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation OR character_not_in_repertoire THEN
  RAISE EXCEPTION 'AD277: material ilegible' USING ERRCODE='22023'; END;
 accion:=d->>'accion';
 perfil:=CASE WHEN session_user='vec_externo_documentos_bolsa_datos'
  THEN 'documentos_bolsa_externa' ELSE 'documentos_bolsa_rrhh' END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object' OR
  accion NOT IN ('bolsa.justificante_renuncia.subir_propio',
   'bolsa.justificante_renuncia.descargar_propio','bolsa.justificante_renuncia.descargar_rrhh') OR
  ((perfil='documentos_bolsa_externa') IS DISTINCT FROM
   (accion IN ('bolsa.justificante_renuncia.subir_propio',
    'bolsa.justificante_renuncia.descargar_propio'))) OR
  c->>'operacion' IS DISTINCT FROM accion OR
  ((accion='bolsa.justificante_renuncia.subir_propio' AND
    c->>'audiencia_consumo' IN ('vec_documentos.bolsa.justificante_renuncia.preparar_subida.v1',
     'vec_documentos.bolsa.justificante_renuncia.subida.v1')) OR
   (accion='bolsa.justificante_renuncia.descargar_propio' AND
    c->>'audiencia_consumo'='vec_documentos.bolsa.justificante_renuncia.descarga_propia.v1') OR
   (accion='bolsa.justificante_renuncia.descargar_rrhh' AND
    c->>'audiencia_consumo' IN ('vec_documentos.bolsa.justificante_renuncia.preparar_descarga_rrhh.v1',
     'vec_documentos.bolsa.justificante_renuncia.descarga_rrhh.v1'))) IS NOT TRUE OR
  d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR
  d->>'tipo_recurso' IS DISTINCT FROM 'justificante_renuncia' OR
  d->>'finalidad' IS DISTINCT FROM 'custodiar_justificante_renuncia_bolsa' OR
  d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref' OR
  d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' OR
  d->'campos_permitidos' IS DISTINCT FROM (CASE
   WHEN accion='bolsa.justificante_renuncia.subir_propio'
   THEN '["contenido","estado","recibo"]'::jsonb ELSE '["contenido"]'::jsonb END) OR
  d->'obligaciones' IS DISTINCT FROM '[]'::jsonb OR
  d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM
   (CASE WHEN perfil='documentos_bolsa_externa' THEN 'externa_personal'
    ELSE 'interna_corporativa' END)
 THEN RAISE EXCEPTION 'AD277: acto documental divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  perfil,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS DISTINCT FROM TRUE OR v.decision_ref IS DISTINCT FROM c->>'decision_ref' OR
  v.efecto_ref IS DISTINCT FROM c->>'efecto_ref' OR
  v.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'AD277: consumo no corresponde' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT v.decision_ref,v.efecto_ref,v.huella_efecto_sha256,v.consumo_huella_sha256,
  v.auditoria_ref,v.consumida_en,v.consumo_nuevo;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_subida_local_bolsa_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_subida_local_bolsa_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_documentos_propietario;

DO $post$
DECLARE f oid;
BEGIN
 f:='vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)'::regprocedure;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND
  p.proowner='vec_identidad_sesiones_v1_propietario'::regrole AND p.prosecdef AND
  p.proacl::text='{vec_identidad_sesiones_v1_propietario=X/vec_identidad_sesiones_v1_propietario,vec_identidad_sesiones_v1_revalidador=X/vec_identidad_sesiones_v1_propietario,vec_autorizacion_propietario=X/vec_identidad_sesiones_v1_propietario}')
 THEN RAISE EXCEPTION 'AD277: ACL de identidad incompatible' USING ERRCODE='55000'; END IF;
 IF encode(sha256(convert_to(pg_get_functiondef(
  'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
   IS DISTINCT FROM '7d13731f452b0f8d7020ea0ab602c465897617fce61a84a7887491309e1bbb48' OR
  encode(sha256(convert_to(pg_get_functiondef(
   'vec_autorizacion_atestada_v3.leer_configuracion_externa_v1(text,jsonb)'::regprocedure),'UTF8')),'hex')
   IS DISTINCT FROM '72c0162c96d365f2a9b907eda64681ce0628424c8d03d4c5fa5baded55321ff9' OR
  encode(sha256(convert_to(pg_get_functiondef(
   'vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1(text,jsonb)'::regprocedure),'UTF8')),'hex')
   IS DISTINCT FROM '89c33e9d7cff6b5c127e77ff798828f9bec4dc4f45b1f614a1618543de0f772d' OR
  encode(sha256(convert_to(pg_get_functiondef(
   'vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(jsonb)'::regprocedure),'UTF8')),'hex')
   IS DISTINCT FROM 'e7d87c2a63a927d9dddea33a5f4bf5bfbd6f1d767651da2ca5e11b0bf8678726'
 THEN RAISE EXCEPTION 'AD277: postimagen central divergente' USING ERRCODE='55000'; END IF;
 FOREACH f IN ARRAY ARRAY[
  'vec_autorizacion_atestada_v3.login_documentos_bolsa_v1(boolean)'::regprocedure::oid,
  'vec_autorizacion_atestada_v3.consumir_subida_local_modulo_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure::oid,
  'vec_autorizacion_atestada_v3.consumir_subida_local_bolsa_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure::oid]
 LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND
   p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
   AND NOT EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE a.grantee=0 AND a.privilege_type='EXECUTE'))
  THEN RAISE EXCEPTION 'AD277: ACL central incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 f:='vec_autorizacion.revalidar_sesion_documental_bolsa_v1(jsonb,timestamptz,timestamptz,timestamptz)'::regprocedure;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_propietario'::regrole
  AND p.prosecdef AND has_function_privilege('vec_autorizacion_atestada_v3_propietario',p.oid,'EXECUTE')
  AND NOT has_function_privilege('vec_documentos_propietario',p.oid,'EXECUTE'))
 THEN RAISE EXCEPTION 'AD277: ACL de sesión incompatible' USING ERRCODE='55000'; END IF;
 f:='vec_autorizacion.revalidar_sesion_documental_bolsa_rrhh_v1(jsonb,timestamptz,timestamptz,timestamptz)'::regprocedure;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_propietario'::regrole
  AND p.prosecdef AND p.proacl::text=
   '{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario}')
 THEN RAISE EXCEPTION 'AD277: ACL de sesión RRHH incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
