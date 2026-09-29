\set ON_ERROR_STOP on
-- ContextoActor 000011 (Fase 1, paso 3): la fachada persona_candidato_avisos_v1
-- (000010) pasa del propietario común de Usuarios al propietario de los
-- correos del Área personal, que desde Usuarios 000010 posee la lectura del
-- correo activo para los avisos de llamamiento. No cambia la función.
-- Instalar después de roles_000010 de Usuarios y antes de Usuarios 000010.
-- DOWN prohibido tras historia.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contexto_actor_v1:migracion:persona_candidato_avisos_poblacion:v1', 0));

DO $preimagen$
DECLARE f regprocedure := pg_catalog.to_regprocedure('vec_contexto_actor_v1.persona_candidato_avisos_v1(text)');
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = current_user AND rolsuper)
       OR f IS NULL
       OR NOT pg_catalog.has_function_privilege('vec_usuarios_propietario', f, 'EXECUTE')
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'vec_usuarios_correos_externo_propietario'
                        AND NOT rolcanlogin AND NOT rolbypassrls AND NOT rolsuper) THEN
        RAISE EXCEPTION 'ContextoActor 000011: preimagen incompatible' USING ERRCODE = '55000';
    END IF;
END
$preimagen$;

SET LOCAL ROLE vec_contexto_actor_v1_propietario;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_usuarios_correos_externo_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.persona_candidato_avisos_v1(text)
    TO vec_usuarios_correos_externo_propietario;
REVOKE EXECUTE ON FUNCTION vec_contexto_actor_v1.persona_candidato_avisos_v1(text)
    FROM vec_usuarios_propietario;
-- Sin otra función concedida, el propietario común deja también el esquema.
DO $uso$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
                    WHERE p.pronamespace = 'vec_contexto_actor_v1'::regnamespace
                      AND pg_catalog.has_function_privilege('vec_usuarios_propietario', p.oid, 'EXECUTE')) THEN
        REVOKE USAGE ON SCHEMA vec_contexto_actor_v1 FROM vec_usuarios_propietario;
    END IF;
END $uso$;
RESET ROLE;

DO $postimagen$
DECLARE f regprocedure := 'vec_contexto_actor_v1.persona_candidato_avisos_v1(text)'::regprocedure;
BEGIN
    IF (SELECT proowner FROM pg_catalog.pg_proc WHERE oid = f) IS DISTINCT FROM 'vec_contexto_actor_v1_propietario'::regrole
       OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid = f) IS NOT TRUE
       OR NOT pg_catalog.has_function_privilege('vec_usuarios_correos_externo_propietario', f, 'EXECUTE')
       OR pg_catalog.has_function_privilege('vec_usuarios_propietario', f, 'EXECUTE')
       OR pg_catalog.has_function_privilege('vec_usuarios_correos_interno_propietario', f, 'EXECUTE')
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
                  CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl, pg_catalog.acldefault('f', p.proowner))) a
                   WHERE p.oid = f AND (a.grantee = 0 OR a.grantee NOT IN (p.proowner, 'vec_usuarios_correos_externo_propietario'::regrole)))
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
                   WHERE p.pronamespace = 'vec_contexto_actor_v1'::regnamespace AND p.oid <> f
                     AND pg_catalog.has_function_privilege('vec_usuarios_correos_externo_propietario', p.oid, 'EXECUTE')) THEN
        RAISE EXCEPTION 'ContextoActor 000011: ACL incompatible' USING ERRCODE = '55000';
    END IF;
END
$postimagen$;
COMMIT;
