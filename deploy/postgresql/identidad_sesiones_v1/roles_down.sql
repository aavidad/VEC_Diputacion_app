-- Reversion DBA. Debe ejecutarse despues de retirar las migraciones y todas
-- las membresias LOGIN de los grupos tecnicos.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_identidad_sesiones_v1:roles_down:v1', 0)
);

DO $prevalidacion$
DECLARE
    rol text;
    miembros text[];
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'down de identidad rechazado: requiere superusuario';
    END IF;
    IF pg_catalog.to_regnamespace('vec_identidad_sesiones_v1') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down de roles rechazado: el esquema sigue instalado';
    END IF;
    FOREACH rol IN ARRAY ARRAY[
        'vec_identidad_sesiones_v1_propietario',
        'vec_identidad_sesiones_v1_migrador',
        'vec_identidad_sesiones_v1_provisionador',
        'vec_identidad_sesiones_v1_registrador',
        'vec_identidad_sesiones_v1_revalidador',
        'vec_identidad_sesiones_v1_revocador'
    ] LOOP
        SELECT array_agg(miembro.rolname::text ORDER BY miembro.rolname)
          INTO miembros
          FROM pg_catalog.pg_auth_members AS enlace
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = enlace.roleid
          JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = enlace.member
         WHERE grupo.rolname = rol
           AND NOT (
               grupo.rolname = 'vec_identidad_sesiones_v1_propietario'
               AND miembro.rolname = 'vec_identidad_sesiones_v1_migrador'
           );
        IF cardinality(miembros) > 0 THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'down rechazado: existen membresias externas',
                DETAIL = rol || ':' || array_to_string(miembros, ',');
        END IF;
    END LOOP;
END
$prevalidacion$;

REVOKE vec_identidad_sesiones_v1_propietario
    FROM vec_identidad_sesiones_v1_migrador;
REVOKE EXECUTE ON FUNCTION public.gen_random_bytes(integer)
    FROM vec_identidad_sesiones_v1_propietario;
REVOKE EXECUTE ON FUNCTION public.digest(bytea, text)
    FROM vec_identidad_sesiones_v1_propietario;
REVOKE USAGE ON SCHEMA public FROM vec_identidad_sesiones_v1_propietario;

-- No se restaura EXECUTE de PUBLIC sobre pgcrypto: la revocacion pertenece al
-- baseline de la base VEC dedicada y solo el DBA puede revertirla expresamente.

-- 000001 endurece los valores por defecto globales del propietario. Al no
-- quedar ya esquema ni objetos suyos, se restauran los defaults nativos para
-- eliminar las entradas pg_default_acl que impedirian retirar el rol.
ALTER DEFAULT PRIVILEGES FOR ROLE vec_identidad_sesiones_v1_propietario
    GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_identidad_sesiones_v1_propietario
    GRANT USAGE ON TYPES TO PUBLIC;

DO $privilegios_base$
DECLARE
    rol text;
BEGIN
    FOREACH rol IN ARRAY ARRAY[
        'vec_identidad_sesiones_v1_migrador',
        'vec_identidad_sesiones_v1_provisionador',
        'vec_identidad_sesiones_v1_registrador',
        'vec_identidad_sesiones_v1_revalidador',
        'vec_identidad_sesiones_v1_revocador',
        'vec_identidad_sesiones_v1_propietario'
    ] LOOP
        EXECUTE format(
            'REVOKE ALL PRIVILEGES ON DATABASE %I FROM %I',
            current_database(), rol
        );
    END LOOP;
END
$privilegios_base$;

DROP ROLE vec_identidad_sesiones_v1_provisionador;
DROP ROLE vec_identidad_sesiones_v1_registrador;
DROP ROLE vec_identidad_sesiones_v1_revalidador;
DROP ROLE vec_identidad_sesiones_v1_revocador;
DROP ROLE vec_identidad_sesiones_v1_migrador;
DROP ROLE vec_identidad_sesiones_v1_propietario;
COMMIT;
