\set ON_ERROR_STOP on
-- AD245: aislamiento del ejecutor B103 en el consumidor nominal V3 existente.
-- AUT75 crea el grupo B103; AD227/AD236 y su historia quedan intactos.
-- CONFIG NUEVA (privada, fuera de Git): un LOGIN B103 exclusivo, miembro sólo
-- de vec_admin_version_rol_bolsa_b103_ejecutor, y dos filas AD172 por LOGIN
-- para las audiencias versionar_rol_bolsa.propuesta.v1/cierre.v1, acciones
-- administracion.perfiles.version_bolsa.proponer/aprobar y canal
-- administracion_privilegiada. Sin filas, VA172 mantiene el acto cerrado.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.login_version_rol_bolsa_valido_v1()');
 meta jsonb;origen text;h text;g record;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR f IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_b103_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion.proponer_version_rol_bolsa_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regrole('vec_admin_version_rol_bolsa_ejecutor') IS NULL
 OR to_regrole('vec_admin_version_rol_bolsa_b103_ejecutor') IS NULL
 THEN RAISE EXCEPTION 'AD245: falta preimagen AUT75/AD227' USING ERRCODE='55000';END IF;
 SELECT p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT origen,meta
 FROM pg_proc p WHERE p.oid=f AND p.proowner=to_regrole('vec_autorizacion_atestada_v3_propietario')
 AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u' AND NOT p.proisstrict
 AND p.proconfig IS NOT DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[]
 AND p.proacl IS NOT DISTINCT FROM ARRAY[
  'vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario'::aclitem];
 h:=encode(sha256(convert_to(origen,'UTF8')),'hex');
 IF h IS DISTINCT FROM 'b82852925bbf8dbd794a8b2557afb780b63fee0103b2fdd3a9b3dc01a2667b3e'
 OR has_function_privilege('public',f,'EXECUTE') IS TRUE
 THEN RAISE EXCEPTION 'AD245: validador instalado divergente, actual=%',h USING ERRCODE='55000';END IF;
 SELECT * INTO STRICT g FROM pg_roles WHERE rolname='vec_admin_version_rol_bolsa_b103_ejecutor';
 IF g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreaterole OR g.rolcreatedb
 OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
 OR EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=g.oid)
 THEN RAISE EXCEPTION 'AD245: grupo B103 no aislado' USING ERRCODE='55000';END IF;
 PERFORM set_config('vec.ad245.login_oid',f::text,true);
 PERFORM set_config('vec.ad245.login_meta',meta::text,true);
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.login_version_rol_bolsa_valido_v1()
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT current_setting('role')='none' AND EXISTS(
 SELECT 1 FROM pg_catalog.pg_roles l JOIN pg_catalog.pg_auth_members m ON m.member=l.oid
 JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
 WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
 AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
 AND l.rolconfig IS NULL AND g.rolname IN ('vec_admin_version_rol_bolsa_ejecutor','vec_admin_version_rol_bolsa_b103_ejecutor')
 AND NOT(g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
 AND g.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
 AND (SELECT count(*) FROM pg_catalog.pg_auth_members x WHERE x.member=l.oid)=1
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members x WHERE x.member=g.oid)
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting x WHERE x.setrole IN(l.oid,g.oid)))
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.login_version_rol_bolsa_b103_valido_v1()
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT current_setting('role')='none' AND EXISTS(
 SELECT 1 FROM pg_catalog.pg_roles l JOIN pg_catalog.pg_auth_members m ON m.member=l.oid
 JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
 WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
 AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
 AND l.rolconfig IS NULL AND g.oid=to_regrole('vec_admin_version_rol_bolsa_b103_ejecutor')
 AND NOT(g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
 AND g.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
 AND (SELECT count(*) FROM pg_catalog.pg_auth_members x WHERE x.member=l.oid)=1
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members x WHERE x.member=g.oid)
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting x WHERE x.setrole IN(l.oid,g.oid)))
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.login_version_rol_bolsa_b103_valido_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_b103_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR vec_autorizacion_atestada_v3.login_version_rol_bolsa_b103_valido_v1() IS NOT TRUE
 THEN RAISE EXCEPTION 'AD245: consumidor B103 denegado' USING ERRCODE='42501';END IF;
 RETURN QUERY SELECT x.* FROM vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz) x;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_b103_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_b103_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_autorizacion_propietario;
RESET ROLE;
DO $post$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.login_version_rol_bolsa_valido_v1()');
 meta jsonb;h text;
BEGIN
 SELECT to_jsonb(p)-'prosrc',encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')
 INTO STRICT meta,h FROM pg_proc p WHERE p.oid=f;
 IF f::text IS DISTINCT FROM current_setting('vec.ad245.login_oid')
 OR meta IS DISTINCT FROM current_setting('vec.ad245.login_meta')::jsonb
 OR h IS DISTINCT FROM '6ea566931f00890d5aab8aef1c4465813d1a91af97023d963c950c62ca8b7a25'
 OR has_function_privilege('public',f,'EXECUTE') IS TRUE
 OR has_function_privilege('public','vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_b103_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') IS TRUE
 OR has_function_privilege('vec_autorizacion_propietario','vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_b103_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD245: postimagen OID/ACL/validador divergente, actual=%',h USING ERRCODE='55000';END IF;
END $post$;
COMMIT;
