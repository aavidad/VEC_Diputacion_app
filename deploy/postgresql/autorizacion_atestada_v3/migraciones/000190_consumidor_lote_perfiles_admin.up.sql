\set ON_ERROR_STOP on
-- AD190: consumidor nominal del lote ordinario de perfiles de Administración.
-- Acción administracion.perfiles.aplicar_lote_ordinario (rol de Aplicación v6 o
-- posterior, AUT45/AUT51), audiencia vec_autorizacion.administracion_perfiles.lote_ordinario.v1,
-- recurso «persona» destinataria. Sólo lo ejecuta el propietario de Autorización
-- desde su efecto (AUT44); el núcleo exige además que la sesión sea un LOGIN
-- exclusivo del grupo vec_admin_perfiles_lote_ejecutor. Comprueba la puerta de
-- AUT45 antes y después del consumo. No concede permisos ni escribe efectos.
-- Preimágenes medidas sobre main tras AD193, AD195, AD196, AUT48, AUT49,
-- AD178/AD177 y AUT51 (núcleo y CHECK de audiencias POST178). Una sola vez; sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000190',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE g oid:=to_regprocedure('vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)');
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AD190: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF g IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=g AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef AND p.prorettype='boolean'::regtype)
 OR has_function_privilege('vec_autorizacion_atestada_v3_propietario',g,'EXECUTE') IS NOT TRUE
 OR to_regprocedure('vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)') IS NULL
 OR has_function_privilege('vec_autorizacion_atestada_v3_propietario','vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)','EXECUTE') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD190: PARO clave=AUT45 actual=ausente_o_no_acreditado esperado=puerta_lote_y_revalidacion_propietario_AUT' USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.consumir_lote_perfiles_admin_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_servicios_certificados_propios_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.login_usuarios_admin_valido_v1()') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_recuperacion_firmas_r5_ct_v2_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1(jsonb)') IS NULL
 OR to_regrole('vec_admin_perfiles_lote_ejecutor') IS NOT NULL
 THEN RAISE EXCEPTION 'AD190: PARO clave=preimagen actual=incompatible esperado=AD185_AD195_AD178_AD177_sin_AD190' USING ERRCODE='55000'; END IF;
END $pre$;
-- Grupo del LOGIN técnico de vec-admin que ejecutará el lote. Sin LOGIN propio,
-- sin herencia hacia otros grupos; AUT44 le concede sólo su fachada.
CREATE ROLE vec_admin_perfiles_lote_ejecutor NOLOGIN NOINHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_admin_perfiles_lote_ejecutor',current_database()); END $conexion$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.login_lote_perfiles_admin_valido_v1()
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT current_setting('role')='none' AND EXISTS(
 SELECT 1 FROM pg_catalog.pg_roles l JOIN pg_catalog.pg_auth_members m ON m.member=l.oid JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
 WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls) AND l.rolconfig IS NULL
 AND g.oid=to_regrole('vec_admin_perfiles_lote_ejecutor') AND NOT(g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls) AND g.rolconfig IS NULL
 AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
 AND (SELECT count(*) FROM pg_catalog.pg_auth_members x WHERE x.member=l.oid)=1
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members x WHERE x.member=g.oid)
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting x WHERE x.setrole IN(l.oid,g.oid)))
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.login_lote_perfiles_admin_valido_v1() FROM PUBLIC;

DO $nucleo$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text;fuente text;nueva text;actual text;meta jsonb;deps jsonb;compartidas jsonb;h text;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 runtime_marca text:=E'       OR NOT (\n           (p_perfil_mutacion IS NOT DISTINCT FROM ''servicios_certificados_propios''';
 runtime_nueva text:=E'       OR NOT (\n           (p_perfil_mutacion IS NOT DISTINCT FROM ''lote_perfiles_admin''\n'
  ||E'            AND vec_autorizacion_atestada_v3.login_lote_perfiles_admin_valido_v1() IS TRUE)\n'
  ||E'           OR (p_perfil_mutacion IS NOT DISTINCT FROM ''servicios_certificados_propios''';
 excl_marca text:=$x$p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento' AND p_perfil_mutacion IS DISTINCT FROM 'publicacion_certificado_nominal' AND p_perfil_mutacion IS DISTINCT FROM 'publicacion_cargo_competencial' AND p_perfil_mutacion IS DISTINCT FROM 'capacidades_admin' AND p_perfil_mutacion IS DISTINCT FROM 'persona_denominacion_publicar' AND p_perfil_mutacion IS DISTINCT FROM 'persona_denominacion_leer' AND p_perfil_mutacion IS DISTINCT FROM 'usuarios_admin_listar' AND p_perfil_mutacion IS DISTINCT FROM 'usuarios_admin_consultar' AND p_perfil_mutacion IS DISTINCT FROM 'servicios_certificados_propios'$x$;
 excl_nueva text:=excl_marca||' AND p_perfil_mutacion IS DISTINCT FROM ''lote_perfiles_admin''';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'lote_perfiles_admin'
 AND c->>'operacion' IS NOT DISTINCT FROM 'administracion.perfiles.aplicar_lote_ordinario'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_autorizacion.administracion_perfiles.lote_ordinario.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'administracion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_perfiles'
 AND d->>'garantia_minima' IS NOT DISTINCT FROM 'alto'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^per_[A-Za-z0-9_-]{22,128}$'
 AND d->>'recurso_ref' IS DISTINCT FROM d->>'principal_id'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true'
 AND vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(
   d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
   'administracion.perfiles.aplicar_lote_ordinario','administracion','persona','gestion_perfiles',
   '[]'::jsonb,d->'vinculo_autenticacion_actor') IS TRUE)
$x$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD190: PARO clave=nucleo esperado=presente actual=ausente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 h:=encode(sha256(convert_to(original,'UTF8')),'hex');
 IF h IS DISTINCT FROM '2ccd704afe6140d604faa626631e9743edda8f785517d1136054c746cb9b1381'
 THEN RAISE EXCEPTION 'AD190: PARO clave=nucleo_postAD177_def_SHA actual=% esperado=2ccd704afe6140d604faa626631e9743edda8f785517d1136054c746cb9b1381',h USING ERRCODE='55000'; END IF;
 h:=encode(sha256(convert_to(fuente,'UTF8')),'hex');
 IF h IS DISTINCT FROM '4729b6666065a8a3582443d803bf8535940f0eae650b27a315aca260f3183b8b'
 THEN RAISE EXCEPTION 'AD190: PARO clave=nucleo_postAD177_src_SHA actual=% esperado=4729b6666065a8a3582443d803bf8535940f0eae650b27a315aca260f3183b8b',h USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
   AND p.provolatile='v' AND p.proparallel='u' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND a.grantee=p.proowner AND a.grantor=p.proowner AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
 THEN RAISE EXCEPTION 'AD190: PARO clave=nucleo_metadatos esperado=propietario_privado actual=incompatible' USING ERRCODE='55000'; END IF;
 -- Cada marca aparece una sola vez en el cuerpo real; el perfil es nuevo.
 IF length(original)-length(replace(original,marca,''))<>length(marca)
 OR length(original)-length(replace(original,runtime_marca,''))<>length(runtime_marca)
 OR length(original)-length(replace(original,excl_marca,''))<>length(excl_marca)
 OR strpos(original,'lote_perfiles_admin')<>0
 OR strpos(original,'resolver_origen_consumo_v1')=0
 THEN RAISE EXCEPTION 'AD190: PARO clave=marcas esperado=tres_marcas_unicas_sin_perfil actual=incompatible' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
  INTO compartidas FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
   AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database());
 nueva:=replace(replace(replace(original,runtime_marca,runtime_nueva),excl_marca,excl_nueva),marca,extension||marca);
 EXECUTE nueva;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nueva
 OR replace(replace(replace(actual,extension||marca,marca),excl_nueva,excl_marca),runtime_nueva,runtime_marca) IS DISTINCT FROM original
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
     FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
     FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
      AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())) IS DISTINCT FROM compartidas
 THEN RAISE EXCEPTION 'AD190: PARO clave=postimagen_nucleo esperado=extension_minima_metadatos_intactos actual=divergente' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE anterior text;nueva text;h text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 h:=encode(sha256(convert_to(anterior,'UTF8')),'hex');
 -- Preimagen medida tras AD178/AD177.
 IF h IS DISTINCT FROM 'e76428d2ecd1c79da88a827cf33138f5c75026ed2138858c67f93420e932eae7'
 OR left(anterior,7)<>'CHECK (' OR right(anterior,1)<>')'
 OR strpos(anterior,'vec_autorizacion.administracion_perfiles.lote_ordinario.v1')<>0
 THEN RAISE EXCEPTION 'AD190: PARO clave=CHECK_audiencias_SHA actual=% esperado=e76428d2ecd1c79da88a827cf33138f5c75026ed2138858c67f93420e932eae7',h USING ERRCODE='55000'; END IF;
 nueva:='CHECK (('||substr(anterior,8,length(anterior)-8)||') OR audiencia_consumo = ''vec_autorizacion.administracion_perfiles.lote_ordinario.v1'')';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated
   AND strpos(pg_get_constraintdef(c.oid,false),'vec_autorizacion.administracion_perfiles.lote_ordinario.v1')>0)
 THEN RAISE EXCEPTION 'AD190: PARO clave=CHECK_postimagen esperado=audiencia_anadida actual=ausente' USING ERRCODE='55000'; END IF;
END $audiencias$;

-- Fachada de consumo: sólo la ejecuta el propietario de Autorización desde el
-- efecto del lote. Comprueba el contrato cerrado y la puerta del lote (versión vigente del rol) antes
-- de entrar en el núcleo (firma, gobierno, revocación, origen y vigencia) y
-- vuelve a comprobar la decisión viva y la puerta después del consumo.
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_lote_perfiles_admin_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
    OR vec_autorizacion_atestada_v3.login_lote_perfiles_admin_valido_v1() IS NOT TRUE
    OR p_capacidad IS NULL OR vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(p_capacidad) IS NOT TRUE
    OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
    OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 1 AND 262144
    OR p_motivo IS NULL OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
    OR p_persona_version IS NULL OR p_perfil_version IS NULL
    OR p_payload IS NULL OR octet_length(p_payload) NOT BETWEEN 1 AND 1048576
    OR p_sobre IS NULL OR octet_length(p_sobre) NOT BETWEEN 1 AND 1048576
    OR p_evidencia IS NULL OR octet_length(p_evidencia) NOT BETWEEN 1 AND 262144
    OR p_raiz IS NULL OR octet_length(p_raiz)<>44 THEN
  RAISE EXCEPTION 'AD190: consumo del lote denegado' USING ERRCODE='42501';
 END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'AD190: material del lote inválido' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR d->'concedida' IS DISTINCT FROM 'true'::jsonb
    OR d->>'modulo_id' IS DISTINCT FROM 'administracion'
    OR d->>'accion' IS DISTINCT FROM 'administracion.perfiles.aplicar_lote_ordinario'
    OR c->>'operacion' IS DISTINCT FROM d->>'accion'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_autorizacion.administracion_perfiles.lote_ordinario.v1'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'persona'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_perfiles'
    OR d->>'garantia_minima' IS DISTINCT FROM 'alto'
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '["auditar"]'::jsonb
    OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'administracion_privilegiada'
    OR d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'true'
    OR d->>'recurso_ref' IS NULL OR d->>'recurso_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR d->>'recurso_ref' IS NOT DISTINCT FROM d->>'principal_id'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS NULL
    OR d->>'contexto_recurso_huella_sha256' !~ '^[0-9a-f]{64}$'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR c->>'decision_ref' IS DISTINCT FROM d->>'decision_ref'
    OR vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(
      d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
      'administracion.perfiles.aplicar_lote_ordinario','administracion','persona','gestion_perfiles',
      '[]'::jsonb,d->'vinculo_autenticacion_actor') IS NOT TRUE THEN
  RAISE EXCEPTION 'AD190: consumo del lote denegado' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT x
 FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'lote_perfiles_admin',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR x.efecto_ref IS DISTINCT FROM c->>'efecto_ref'
    OR x.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR x.consumo_huella_sha256 IS NULL OR x.consumo_huella_sha256 !~ '^[0-9a-f]{64}$'
    OR x.auditoria_ref IS DISTINCT FROM 'aud_v3_'||substr(x.consumo_huella_sha256,1,32)
    OR x.consumida_en IS NULL OR NOT isfinite(x.consumida_en)
    OR vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
    OR vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(
      d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
      'administracion.perfiles.aplicar_lote_ordinario','administracion','persona','gestion_perfiles',
      '[]'::jsonb,d->'vinculo_autenticacion_actor') IS NOT TRUE THEN
  RAISE EXCEPTION 'AD190: consumo del lote divergente' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,
  x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_lote_perfiles_admin_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_lote_perfiles_admin_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_autorizacion_propietario;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_autorizacion_propietario;
DO $acl$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_lote_perfiles_admin_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 l oid:=to_regprocedure('vec_autorizacion_atestada_v3.login_lote_perfiles_admin_valido_v1()');
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR NOT has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
 OR has_function_privilege('vec_admin_perfiles_lote_ejecutor',f,'EXECUTE')
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,'vec_autorizacion_propietario'::regrole) OR a.privilege_type<>'EXECUTE'
     OR a.is_grantable OR a.grantor<>p.proowner))
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=l AND a.grantee<>p.proowner)
 THEN RAISE EXCEPTION 'AD190: PARO clave=ACL esperado=propietario_AD_y_propietario_AUT actual=ampliada' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
