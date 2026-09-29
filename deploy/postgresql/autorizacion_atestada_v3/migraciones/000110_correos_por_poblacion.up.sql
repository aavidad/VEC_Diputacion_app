\set ON_ERROR_STOP on
-- AD3-110: el consumo V3 de «Mis correos» pasa del propietario común de
-- Usuarios a los dos propietarios por población (roles_000010 de Usuarios).
-- No cambia firma, cuerpo, audiencias ni perfiles de AD3-107: solo quién
-- puede invocar la fachada. El consumo de la lectura de avisos (AD3-109)
-- pasa al propietario externo, que es quien posee esa lectura desde Usuarios
-- 000010. Instalar después de roles_000010 y antes de
-- Usuarios 000010. Entre ambas, los correos antiguos quedan denegados.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000110',0));
DO $pre$ DECLARE f regprocedure:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_correos_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' OR f IS NULL
    OR NOT has_function_privilege('vec_usuarios_propietario',f,'EXECUTE')
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_usuarios_propietario','vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_correos_interno_propietario' AND NOT rolcanlogin AND NOT rolbypassrls AND NOT rolinherit)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_correos_externo_propietario' AND NOT rolcanlogin AND NOT rolbypassrls AND NOT rolinherit)
 THEN RAISE EXCEPTION 'AD3-110: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_usuarios_correos_interno_propietario,vec_usuarios_correos_externo_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_correos_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_usuarios_correos_interno_propietario,vec_usuarios_correos_externo_propietario;
REVOKE EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_correos_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM vec_usuarios_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_correos_externo_propietario;
REVOKE EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_usuarios_propietario;
DO $acl$ DECLARE f regprocedure:='vec_autorizacion_atestada_v3.consumir_correos_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR NOT has_function_privilege('vec_usuarios_correos_interno_propietario',f,'EXECUTE')
    OR NOT has_function_privilege('vec_usuarios_correos_externo_propietario',f,'EXECUTE')
    OR has_function_privilege('vec_usuarios_propietario',f,'EXECUTE')
    OR has_function_privilege('vec_usuarios_ejecutor_interno',f,'EXECUTE')
    OR has_function_privilege('vec_usuarios_ejecutor_externo',f,'EXECUTE')
    OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
      WHERE p.oid=f AND (a.grantee=0 OR a.grantee NOT IN (p.proowner,'vec_usuarios_correos_interno_propietario'::regrole,'vec_usuarios_correos_externo_propietario'::regrole)))
    -- La lectura de avisos: solo el propietario externo.
    OR NOT has_function_privilege('vec_usuarios_correos_externo_propietario','vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('vec_usuarios_propietario','vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
      WHERE p.oid='vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
        AND (a.grantee=0 OR a.grantee NOT IN (p.proowner,'vec_usuarios_correos_externo_propietario'::regrole)))
    -- Los propietarios por población no alcanzan ninguna otra fachada V3.
    OR EXISTS(SELECT 1 FROM pg_proc p WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace AND p.oid<>f
      AND p.oid<>'vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
      AND (has_function_privilege('vec_usuarios_correos_interno_propietario',p.oid,'EXECUTE')
        OR has_function_privilege('vec_usuarios_correos_externo_propietario',p.oid,'EXECUTE')))
    OR has_function_privilege('vec_usuarios_correos_interno_propietario','vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'AD3-110: ACL incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
