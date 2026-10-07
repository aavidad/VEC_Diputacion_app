\set ON_ERROR_STOP on
-- AD220: consumidor V3 nominal de la creación gobernada de un RolID nuevo.
-- Requiere AUT59 y AD219; la puerta AUT60 se invoca sólo en ejecución, pues
-- AUT60 depende a su vez de esta fachada. No instala concesiones ni LOGIN.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000220',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AD220: migrador PG18 no acreditado' USING ERRCODE='42501'; END IF;
 IF f IS NULL
 OR to_regprocedure('vec_autorizacion.concesiones_gobierno_definiciones_admin_v1()') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_catalogo_acciones_admin_v1(jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5()') IS NULL
 OR to_regprocedure('vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_gobierno_rol_nuevo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regrole('vec_admin_gobierno_roles_ejecutor') IS NOT NULL
 THEN RAISE EXCEPTION 'AD220: preimagen AUT59/AD219 incompatible' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR NOT has_function_privilege('vec_autorizacion_atestada_v3_propietario',
   'vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)','EXECUTE')
 THEN RAISE EXCEPTION 'AD220: núcleo o revalidación no acreditados' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_admin_gobierno_roles_ejecutor NOLOGIN NOINHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_admin_gobierno_roles_ejecutor',current_database()); END $conexion$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.login_gobierno_rol_nuevo_valido_v1()
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT current_setting('role')='none' AND EXISTS(
 SELECT 1 FROM pg_catalog.pg_roles l JOIN pg_catalog.pg_auth_members m ON m.member=l.oid
 JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
 WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
 AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
 AND l.rolconfig IS NULL AND g.oid=to_regrole('vec_admin_gobierno_roles_ejecutor')
 AND NOT(g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
 AND g.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
 AND (SELECT count(*) FROM pg_catalog.pg_auth_members x WHERE x.member=l.oid)=1
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members x WHERE x.member=g.oid)
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting x WHERE x.setrole IN(l.oid,g.oid)))
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.login_gobierno_rol_nuevo_valido_v1() FROM PUBLIC;

DO $nucleo$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text;fuente text;nueva text;actual text;meta jsonb;deps jsonb;compartidas jsonb;h text;
 runtime_marca text:=$m$           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'lote_perfiles_admin'$m$;
 runtime_nueva text:=$m$           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'gobierno_rol_nuevo'
            AND vec_autorizacion_atestada_v3.login_gobierno_rol_nuevo_valido_v1() IS TRUE)
$m$;
 excl_marca text:=$m$AND p_perfil_mutacion IS DISTINCT FROM 'gobierno_plan_nominal_firma_ct'$m$;
 excl_nueva text:=excl_marca||$m$ AND p_perfil_mutacion IS DISTINCT FROM 'gobierno_rol_nuevo'$m$;
 tupla_marca text:=$m$       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'$m$;
 tupla_nueva text:=$m$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'gobierno_rol_nuevo'
 AND (c->>'operacion' IN ('administracion.perfiles.definicion.proponer','administracion.perfiles.definicion.aprobar')) IS TRUE
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM (
   CASE c->>'operacion'
    WHEN 'administracion.perfiles.definicion.proponer' THEN 'vec_autorizacion.gobierno_rol_nuevo.propuesta.v1'
    WHEN 'administracion.perfiles.definicion.aprobar' THEN 'vec_autorizacion.gobierno_rol_nuevo.cierre.v1' END)
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'administracion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM (
   CASE c->>'operacion' WHEN 'administracion.perfiles.definicion.proponer' THEN 'definicion_rol'
    ELSE 'propuesta_definicion_rol' END)
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gobierno_definiciones_perfiles'
 AND d->>'garantia_minima' IS NOT DISTINCT FROM 'alto'
 AND d->>'recurso_ref' IS NOT NULL
 AND ((c->>'operacion'='administracion.perfiles.definicion.proponer'
      AND d->>'recurso_ref' ~ '^rol:[a-z][a-z0-9_]{2,63}:v1$')
   OR (c->>'operacion'='administracion.perfiles.definicion.aprobar'
      AND d->>'recurso_ref' ~ '^propuesta_admin:[0-9a-f]{32}$'))
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND d->>'contexto_recurso_huella_sha256' ~ '^[0-9a-f]{64}$'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true'
 AND vec_autorizacion.acreditar_gobierno_rol_nuevo_v1(d) IS TRUE)
$m$;
BEGIN
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 h:=encode(sha256(convert_to(original,'UTF8')),'hex');
 IF h IS DISTINCT FROM 'c74551eab17bea78bb564d14d96b77f33bd24a5874b7bdb6e100a59d8f2fe714'
 OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM '79d2f29752235a01716d49095777fe8e890a6deed5269a8647d1f866d4b671b5'
 THEN RAISE EXCEPTION 'AD220: núcleo POST218 divergente; def=%',h USING ERRCODE='55000'; END IF;
 IF (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
 OR length(original)-length(replace(original,runtime_marca,''))<>length(runtime_marca)
 OR length(original)-length(replace(original,excl_marca,''))<>length(excl_marca)
 OR length(original)-length(replace(original,tupla_marca,''))<>length(tupla_marca)
 OR strpos(original,'gobierno_rol_nuevo')<>0
 THEN RAISE EXCEPTION 'AD220: marcas o ACL del núcleo divergentes' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO compartidas FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
 AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database());
 nueva:=replace(replace(replace(original,runtime_marca,runtime_nueva||runtime_marca),excl_marca,excl_nueva),tupla_marca,tupla_nueva||tupla_marca);
 EXECUTE nueva;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nueva
 OR replace(replace(replace(actual,tupla_nueva||tupla_marca,tupla_marca),excl_nueva,excl_marca),runtime_nueva||runtime_marca,runtime_marca) IS DISTINCT FROM original
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb) FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())) IS DISTINCT FROM compartidas
 THEN RAISE EXCEPTION 'AD220: postimagen de núcleo divergente' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE anterior text;nueva text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF left(anterior,7)<>'CHECK (' OR right(anterior,1)<>')'
 OR strpos(anterior,'vec_bolsa_llamamientos.carga_convoca.confirmar.v1')=0
 OR strpos(anterior,'vec_autorizacion.gobierno_rol_nuevo.propuesta.v1')<>0
 THEN RAISE EXCEPTION 'AD220: CHECK de audiencias incompatible' USING ERRCODE='55000'; END IF;
 nueva:='CHECK (('||substr(anterior,8,length(anterior)-8)||') OR audiencia_consumo = ANY (ARRAY[''vec_autorizacion.gobierno_rol_nuevo.propuesta.v1'',''vec_autorizacion.gobierno_rol_nuevo.cierre.v1'']))';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_gobierno_rol_nuevo_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb;d jsonb;x record;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR vec_autorizacion_atestada_v3.login_gobierno_rol_nuevo_valido_v1() IS NOT TRUE
 OR p_capacidad IS NULL OR vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(p_capacidad) IS NOT TRUE
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
 OR p_motivo IS NULL OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
 OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 1 AND 262144
 OR p_persona_version IS NULL OR p_perfil_version IS NULL
 OR p_payload IS NULL OR octet_length(p_payload) NOT BETWEEN 1 AND 1048576
 OR p_sobre IS NULL OR octet_length(p_sobre) NOT BETWEEN 1 AND 1048576
 OR p_evidencia IS NULL OR octet_length(p_evidencia) NOT BETWEEN 1 AND 262144
 OR p_raiz IS NULL OR octet_length(p_raiz)<>44
 THEN RAISE EXCEPTION 'AD220: consumo denegado' USING ERRCODE='42501'; END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD220: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
 OR d->'concedida' IS DISTINCT FROM 'true'::jsonb
 OR (c->>'operacion' IN ('administracion.perfiles.definicion.proponer','administracion.perfiles.definicion.aprobar')) IS NOT TRUE
 OR c->>'audiencia_consumo' IS DISTINCT FROM (
   CASE c->>'operacion' WHEN 'administracion.perfiles.definicion.proponer' THEN 'vec_autorizacion.gobierno_rol_nuevo.propuesta.v1'
   ELSE 'vec_autorizacion.gobierno_rol_nuevo.cierre.v1' END)
 OR d->>'accion' IS DISTINCT FROM c->>'operacion'
 OR d->>'modulo_id' IS DISTINCT FROM 'administracion'
 OR d->>'tipo_recurso' IS DISTINCT FROM (
   CASE c->>'operacion' WHEN 'administracion.perfiles.definicion.proponer' THEN 'definicion_rol'
   ELSE 'propuesta_definicion_rol' END)
 OR d->>'finalidad' IS DISTINCT FROM 'gobierno_definiciones_perfiles'
 OR d->>'garantia_minima' IS DISTINCT FROM 'alto'
 OR d->>'recurso_ref' IS NULL
 OR (((c->>'operacion'='administracion.perfiles.definicion.proponer' AND d->>'recurso_ref' ~ '^rol:[a-z][a-z0-9_]{2,63}:v1$')
   OR (c->>'operacion'='administracion.perfiles.definicion.aprobar' AND d->>'recurso_ref' ~ '^propuesta_admin:[0-9a-f]{32}$'))) IS NOT TRUE
 OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
 OR d->>'contexto_recurso_huella_sha256' IS NULL
 OR d->>'contexto_recurso_huella_sha256' !~ '^[0-9a-f]{64}$'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'administracion_privilegiada'
 OR d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'true'
 OR c->>'decision_ref' IS DISTINCT FROM d->>'decision_ref'
 OR vec_autorizacion.acreditar_gobierno_rol_nuevo_v1(d) IS NOT TRUE
 OR vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
 THEN RAISE EXCEPTION 'AD220: concesión denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'gobierno_rol_nuevo',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.decision_ref IS DISTINCT FROM d->>'decision_ref'
 OR x.efecto_ref IS DISTINCT FROM c->>'efecto_ref'
 OR x.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR x.consumo_huella_sha256 IS NULL OR x.consumo_huella_sha256 !~ '^[0-9a-f]{64}$'
 OR x.auditoria_ref IS DISTINCT FROM 'aud_v3_'||substr(x.consumo_huella_sha256,1,32)
 OR x.consumida_en IS NULL OR NOT isfinite(x.consumida_en)
 OR vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
 OR vec_autorizacion.acreditar_gobierno_rol_nuevo_v1(d) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD220: consumo divergente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,
  x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_gobierno_rol_nuevo_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_gobierno_rol_nuevo_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_autorizacion_propietario;

-- Intentos fallidos de AUT60. El éxito se registra por el consumo V3; los
-- fallos se asientan después de deshacer el subbloque del efecto. AD207 encola
-- el asiento para sellado y rechaza toda nueva auditoría si el sellador caduca.
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD COLUMN gobierno_rol_nuevo_solicitud_sha256 text;
DO $familia$
DECLARE anterior text;nuevo text;nulas text;
 propias constant text[]:=ARRAY['auditoria_ref','secuencia','anterior_sha256','huella_sha256','registrada_en','tipo_registro',
  'evento_ref','evento_material_sha256','operador_login','accion','modulo_id','recurso_ref','finalidad_ref',
  'resultado','motivo_ref','proceso','canal','correlacion_ref','gobierno_rol_nuevo_solicitud_sha256'];
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF anterior NOT LIKE '%catalogo_acciones_detalle%'
 OR anterior LIKE '%gobierno_rol_nuevo_solicitud_sha256%'
 THEN RAISE EXCEPTION 'AD220: familia de auditoría AD219 incompatible' USING ERRCODE='55000'; END IF;
 SELECT string_agg(format('%I IS NULL',a.attname),' AND ' ORDER BY a.attnum) INTO STRICT nulas
 FROM pg_attribute a WHERE a.attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND a.attnum>0 AND NOT a.attisdropped AND NOT a.attname=ANY(propias);
 IF nulas IS NULL OR nulas NOT LIKE '%catalogo_acciones_detalle IS NULL%'
 OR nulas NOT LIKE '%decision_ref IS NULL%'
 THEN RAISE EXCEPTION 'AD220: columnas ajenas no acreditadas' USING ERRCODE='55000'; END IF;
 nuevo:='CHECK ((gobierno_rol_nuevo_solicitud_sha256 IS NULL AND ('||
  substr(anterior,8,length(anterior)-8)||')) OR ('||nulas||
  ' AND tipo_registro = ''intento_gobierno_rol_nuevo'''
  ||' AND evento_ref IS NOT NULL AND evento_material_sha256 IS NOT NULL'
  ||' AND operador_login IS NOT NULL AND accion IN (''administracion.perfiles.definicion.proponer'',''administracion.perfiles.definicion.aprobar'')'
  ||' AND modulo_id = ''administracion'' AND finalidad_ref = ''gobierno_definiciones_perfiles'''
  ||' AND recurso_ref IS NOT NULL AND correlacion_ref IS NOT NULL'
  ||' AND resultado IS NOT NULL AND resultado IN (''denegado'',''error'') AND motivo_ref IS NOT NULL'
  ||' AND proceso = ''postgresql'' AND canal = ''operacion_tecnica_privada'''
  ||' AND gobierno_rol_nuevo_solicitud_sha256 IS NOT NULL))';
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||nuevo;
END $familia$;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD CONSTRAINT auditoria_gobierno_rol_nuevo_formato_v1 CHECK (
 tipo_registro IS DISTINCT FROM 'intento_gobierno_rol_nuevo' OR (
 evento_ref ~ '^evento_[0-9a-f]{32}$'
 AND evento_material_sha256 ~ '^[0-9a-f]{64}$'
 AND correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'
 AND gobierno_rol_nuevo_solicitud_sha256 ~ '^[0-9a-f]{64}$'
 AND recurso_ref ~ '^solicitud_gobierno_rol_nuevo:[0-9a-f]{32}$'
 AND ((resultado='denegado' AND motivo_ref='gobierno_rol_nuevo_denegado')
   OR (resultado='error' AND motivo_ref='gobierno_rol_nuevo_error'))));

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_intento_gobierno_rol_nuevo_v1(p_evento jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE
 v_orden constant text[]:=ARRAY['tipo_registro','evento_ref','operador_login','solicitud_sha256',
  'accion','recurso_ref','resultado','motivo_ref','proceso','canal','finalidad_ref','correlacion_ref'];
 v_claves text[];v_clave text;v_material bytea;v_material_sha text;v_anterior text;
 v_secuencia numeric;v_instante timestamptz(6);v_ref text;v_huella text;v_existente record;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR current_setting('role')<>'none'
 OR vec_autorizacion_atestada_v3.login_gobierno_rol_nuevo_valido_v1() IS NOT TRUE
 THEN RAISE EXCEPTION 'AD220: transacción de intento incompatible' USING ERRCODE='25000'; END IF;
 IF jsonb_typeof(p_evento) IS DISTINCT FROM 'object'
 OR octet_length(p_evento::text)>8192
 THEN RAISE EXCEPTION 'AD220: intento inválido' USING ERRCODE='22023'; END IF;
 SELECT array_agg(k ORDER BY k COLLATE "C") INTO v_claves FROM jsonb_object_keys(p_evento) k;
 IF v_claves IS DISTINCT FROM (SELECT array_agg(k ORDER BY k COLLATE "C") FROM unnest(v_orden) k)
 THEN RAISE EXCEPTION 'AD220: ABI de intento incompatible' USING ERRCODE='22023'; END IF;
 FOREACH v_clave IN ARRAY v_orden LOOP
  IF jsonb_typeof(p_evento->v_clave) IS DISTINCT FROM 'string'
  OR octet_length(p_evento->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD220: campo de intento inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_evento->>'tipo_registro' IS DISTINCT FROM 'intento_gobierno_rol_nuevo'
 OR p_evento->>'operador_login' IS DISTINCT FROM session_user::text
 OR octet_length(p_evento->>'operador_login')>63
 OR p_evento->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR p_evento->>'solicitud_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'accion' NOT IN ('administracion.perfiles.definicion.proponer','administracion.perfiles.definicion.aprobar')
 OR p_evento->>'recurso_ref' !~ '^solicitud_gobierno_rol_nuevo:[0-9a-f]{32}$'
 OR NOT ((p_evento->>'resultado'='denegado' AND p_evento->>'motivo_ref'='gobierno_rol_nuevo_denegado')
  OR (p_evento->>'resultado'='error' AND p_evento->>'motivo_ref'='gobierno_rol_nuevo_error'))
 OR p_evento->>'proceso' IS DISTINCT FROM 'postgresql'
 OR p_evento->>'canal' IS DISTINCT FROM 'operacion_tecnica_privada'
 OR p_evento->>'finalidad_ref' IS DISTINCT FROM 'gobierno_definiciones_perfiles'
 OR p_evento->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'AD220: semántica de intento inválida' USING ERRCODE='22023'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.intento-gobierno-rol-nuevo.v1');
 FOREACH v_clave IN ARRAY v_orden LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_evento->>v_clave);
 END LOOP;
 v_material_sha:=encode(sha256(v_material),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:evento-admin:'||(p_evento->>'evento_ref'),0));
 SELECT a.tipo_registro,a.auditoria_ref,a.secuencia,a.huella_sha256,a.correlacion_ref,a.registrada_en,
  a.evento_material_sha256 INTO v_existente FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
 WHERE a.evento_ref=p_evento->>'evento_ref';
 IF FOUND THEN
  IF v_existente.tipo_registro IS DISTINCT FROM 'intento_gobierno_rol_nuevo'
  OR v_existente.evento_material_sha256 IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'AD220: replay de intento con material distinto' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,v_existente.huella_sha256,
   v_existente.correlacion_ref,v_existente.registrada_en;
  RETURN;
 END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 IF v_secuencia>=9007199254740991::numeric
 THEN RAISE EXCEPTION 'AD220: secuencia agotada' USING ERRCODE='22003'; END IF;
 v_secuencia:=v_secuencia+1;v_instante:=clock_timestamp();
 v_ref:='aud_v3_grni_'||substr(p_evento->>'evento_ref',8,32);
 v_huella:=encode(sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.intento-gobierno-rol-nuevo.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(to_char(v_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  evento_ref,evento_material_sha256,operador_login,gobierno_rol_nuevo_solicitud_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'intento_gobierno_rol_nuevo',
  p_evento->>'evento_ref',v_material_sha,(p_evento->>'operador_login')::name,p_evento->>'solicitud_sha256',
  p_evento->>'accion','administracion',p_evento->>'recurso_ref',p_evento->>'finalidad_ref',
  p_evento->>'resultado',p_evento->>'motivo_ref',p_evento->>'proceso',p_evento->>'canal',p_evento->>'correlacion_ref');
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_evento->>'correlacion_ref',v_instante;
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_gobierno_rol_nuevo_v1(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_gobierno_rol_nuevo_v1(jsonb) TO vec_autorizacion_propietario;
DO $acl$
DECLARE f oid;n integer:=0;
BEGIN
 FOR f IN SELECT p.oid FROM pg_proc p WHERE p.oid IN (
  'vec_autorizacion_atestada_v3.consumir_gobierno_rol_nuevo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_autorizacion_atestada_v3.registrar_intento_gobierno_rol_nuevo_v1(jsonb)'::regprocedure) LOOP
  n:=n+1;
  IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
   AND p.provolatile='v' AND p.proparallel='u')
  OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL
    aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
  OR NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
    aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f
    AND a.grantee='vec_autorizacion_propietario'::regrole AND a.grantor=p.proowner
    AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
  OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
    aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f
    AND (a.grantee NOT IN(p.proowner,'vec_autorizacion_propietario'::regrole)
      OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
  OR has_function_privilege('vec_admin_gobierno_roles_ejecutor',f,'EXECUTE')
  THEN RAISE EXCEPTION 'AD220: ACL de función divergente' USING ERRCODE='55000'; END IF;
 END LOOP;
 IF n<>2 OR has_function_privilege('vec_admin_gobierno_roles_ejecutor',
 'vec_autorizacion_atestada_v3.login_gobierno_rol_nuevo_valido_v1()','EXECUTE')
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=
   'vec_autorizacion_atestada_v3.login_gobierno_rol_nuevo_valido_v1()'::regprocedure
   AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp'])
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid='vec_autorizacion_atestada_v3.login_gobierno_rol_nuevo_valido_v1()'::regprocedure
   AND (a.grantee<>p.proowner OR a.grantor<>p.proowner
     OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_admin_gobierno_roles_ejecutor'
   AND NOT(rolcanlogin OR rolinherit OR rolsuper OR rolcreatedb OR rolcreaterole
     OR rolreplication OR rolbypassrls) AND rolconfig IS NULL)
 THEN RAISE EXCEPTION 'AD220: ACL de grupo divergente' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
