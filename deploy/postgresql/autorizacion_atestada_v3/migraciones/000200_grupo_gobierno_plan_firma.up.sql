\set ON_ERROR_STOP on
-- AD200: grupo técnico dedicado del gobierno del plan nominal de firma
-- (decisión de dirección del 5 de octubre de 2026). AD177 dejó la fachada
-- registrar_y_confirmar_gobierno_plan_firma_v1 en el runtime CT y el núcleo
-- clasificaba el perfil gobierno_plan_nominal_firma_ct como runtime CT (hoy con
-- dos LOGIN). Desde AD200 sólo la ejecuta un LOGIN exclusivo del grupo NOLOGIN
-- vec_plan_firma_gobierno_ejecutor, con una sola pertenencia, como el lote de
-- AD190. No cambia el contrato de la decisión, el CHECK de audiencias ni CC7, no
-- crea LOGIN (lo hace el DBA) y no concede otro permiso.
-- Preimagen medida en clon sobre main con AD190/CA35/AUT44 (núcleo 05e6753a…,
-- fuente 1a5c3e67…). Si otro consumidor reescribe antes el núcleo (AD197,
-- AD198, AD199…), se detiene con PARO sin tocar nada y hay que remedirla.
-- Una sola vez; sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000200',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE fachada oid:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AD200: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF fachada IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.login_lote_perfiles_admin_valido_v1()') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.login_gobierno_plan_firma_valido_v1()') IS NOT NULL
 OR to_regrole('vec_plan_firma_gobierno_ejecutor') IS NOT NULL
 OR to_regrole('vec_contratacion_temporal_ejecutor') IS NULL
 THEN RAISE EXCEPTION 'AD200: PARO clave=preimagen actual=incompatible esperado=AD177_AD190_sin_AD200' USING ERRCODE='55000'; END IF;
 -- La fachada tiene exactamente el ACL que dejó AD177: propietario y runtime CT.
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=fachada AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef)
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=fachada)<>2
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=fachada AND a.grantee IN(p.proowner,'vec_contratacion_temporal_ejecutor'::regrole)
   AND a.grantor=p.proowner AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)<>2
 THEN RAISE EXCEPTION 'AD200: PARO clave=acl_fachada actual=incompatible esperado=propietario_y_runtime_CT_AD177' USING ERRCODE='55000'; END IF;
END $pre$;
-- Grupo del LOGIN técnico que ejecutará el gobierno del plan. Sin LOGIN propio
-- ni herencia hacia otros grupos; sólo recibe la fachada de AD177.
CREATE ROLE vec_plan_firma_gobierno_ejecutor NOLOGIN NOINHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_plan_firma_gobierno_ejecutor',current_database()); END $conexion$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

-- Mismo criterio que login_lote_perfiles_admin_valido_v1 (AD190): LOGIN sin
-- atributos privilegiados ni configuración, una sola pertenencia (al grupo, con
-- INHERIT, sin SET ni ADMIN), sin SET ROLE activo y grupo sin miembros propios.
CREATE FUNCTION vec_autorizacion_atestada_v3.login_gobierno_plan_firma_valido_v1()
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT current_setting('role')='none' AND EXISTS(
 SELECT 1 FROM pg_catalog.pg_roles l JOIN pg_catalog.pg_auth_members m ON m.member=l.oid JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
 WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls) AND l.rolconfig IS NULL
 AND g.oid=to_regrole('vec_plan_firma_gobierno_ejecutor') AND NOT(g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls) AND g.rolconfig IS NULL
 AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
 AND (SELECT count(*) FROM pg_catalog.pg_auth_members x WHERE x.member=l.oid)=1
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members x WHERE x.member=g.oid)
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting x WHERE x.setrole IN(l.oid,g.oid)))
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.login_gobierno_plan_firma_valido_v1() FROM PUBLIC;

DO $nucleo$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text;fuente text;nueva text;actual text;meta jsonb;deps jsonb;compartidas jsonb;h text;
 -- Rama de sesión: el perfil de gobierno exige el LOGIN dedicado y deja de
 -- clasificarse como runtime CT.
 runtime_marca text:=E'       OR NOT (\n           (p_perfil_mutacion IS NOT DISTINCT FROM ''lote_perfiles_admin''';
 runtime_nueva text:=E'       OR NOT (\n           (p_perfil_mutacion IS NOT DISTINCT FROM ''gobierno_plan_nominal_firma_ct''\n'
  ||E'            AND vec_autorizacion_atestada_v3.login_gobierno_plan_firma_valido_v1() IS TRUE)\n'
  ||E'           OR (p_perfil_mutacion IS NOT DISTINCT FROM ''lote_perfiles_admin''';
 excl_marca text:=$x$p_perfil_mutacion IS DISTINCT FROM 'servicios_certificados_propios' AND p_perfil_mutacion IS DISTINCT FROM 'lote_perfiles_admin'$x$;
 excl_nueva text:=excl_marca||' AND p_perfil_mutacion IS DISTINCT FROM ''gobierno_plan_nominal_firma_ct''';
 -- La rama de contrato de AD178 sigue igual: sólo cambia quién puede llegar.
 contrato text:=E'p_perfil_mutacion IS NOT DISTINCT FROM ''gobierno_plan_nominal_firma_ct''\n AND (c->>''operacion'' IN';
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD200: PARO clave=nucleo esperado=presente actual=ausente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 h:=encode(sha256(convert_to(original,'UTF8')),'hex');
 IF h IS DISTINCT FROM '05e6753a55805eb763817202c8ca5bd8612c904fae8c2cc6c5f4baa517935323'
 THEN RAISE EXCEPTION 'AD200: PARO clave=nucleo_postAD190_def_SHA actual=% esperado=05e6753a55805eb763817202c8ca5bd8612c904fae8c2cc6c5f4baa517935323',h USING ERRCODE='55000'; END IF;
 h:=encode(sha256(convert_to(fuente,'UTF8')),'hex');
 IF h IS DISTINCT FROM '1a5c3e67332c6865c42cc7d6f26450958304090137f995f5376ba6ca11a5b087'
 THEN RAISE EXCEPTION 'AD200: PARO clave=nucleo_postAD190_src_SHA actual=% esperado=1a5c3e67332c6865c42cc7d6f26450958304090137f995f5376ba6ca11a5b087',h USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
   AND p.provolatile='v' AND p.proparallel='u' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND a.grantee=p.proowner AND a.grantor=p.proowner AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
 THEN RAISE EXCEPTION 'AD200: PARO clave=nucleo_metadatos esperado=propietario_privado actual=incompatible' USING ERRCODE='55000'; END IF;
 -- Cada marca aparece una sola vez; el perfil de gobierno sólo está en la rama
 -- de contrato de AD178 y todavía no tiene rama de sesión propia.
 IF length(original)-length(replace(original,runtime_marca,''))<>length(runtime_marca)
 OR length(original)-length(replace(original,excl_marca,''))<>length(excl_marca)
 OR length(original)-length(replace(original,contrato,''))<>length(contrato)
 OR length(original)-length(replace(original,'''gobierno_plan_nominal_firma_ct''',''))<>length('''gobierno_plan_nominal_firma_ct''')
 OR strpos(original,'login_gobierno_plan_firma_valido_v1')<>0
 THEN RAISE EXCEPTION 'AD200: PARO clave=marcas esperado=dos_marcas_unicas_perfil_solo_en_contrato actual=incompatible' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
  INTO compartidas FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
   AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database());
 nueva:=replace(replace(original,runtime_marca,runtime_nueva),excl_marca,excl_nueva);
 EXECUTE nueva;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nueva
 OR replace(replace(actual,excl_nueva,excl_marca),runtime_nueva,runtime_marca) IS DISTINCT FROM original
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
     FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
     FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
      AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())) IS DISTINCT FROM compartidas
 THEN RAISE EXCEPTION 'AD200: PARO clave=postimagen_nucleo esperado=cambio_minimo_metadatos_intactos actual=divergente' USING ERRCODE='55000'; END IF;
END $nucleo$;

-- La fachada pasa del runtime CT al grupo dedicado. USAGE del esquema para el
-- grupo; el runtime CT conserva el USAGE que ya tenía por otras fachadas.
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_plan_firma_gobierno_ejecutor;
REVOKE EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v1(
 bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v1(
 bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_plan_firma_gobierno_ejecutor;
RESET ROLE;

DO $post$
DECLARE fachada oid:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 login oid:=to_regprocedure('vec_autorizacion_atestada_v3.login_gobierno_plan_firma_valido_v1()');
BEGIN
 IF (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=fachada)<>2
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=fachada AND a.grantee IN(p.proowner,'vec_plan_firma_gobierno_ejecutor'::regrole)
   AND a.grantor=p.proowner AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)<>2
 OR has_function_privilege('vec_contratacion_temporal_ejecutor',fachada,'EXECUTE')
 THEN RAISE EXCEPTION 'AD200: PARO clave=acl_fachada_post esperado=propietario_y_grupo_dedicado actual=divergente' USING ERRCODE='55000'; END IF;
 IF login IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=login AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef)
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=login)<>1
 OR EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.roleid='vec_plan_firma_gobierno_ejecutor'::regrole OR m.member='vec_plan_firma_gobierno_ejecutor'::regrole)
 THEN RAISE EXCEPTION 'AD200: PARO clave=grupo_post esperado=sin_miembros_login_privado actual=divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
