-- Regresion de ACL de tipos y de la extension criptografica. Las comprobaciones
-- consultan los privilegios efectivos de los dos roles runtime, no solo el
-- texto de GRANT/REVOKE de las migraciones.
BEGIN;
SET LOCAL search_path = pg_catalog;

DO $privilegios$
DECLARE
    oid_propietario oid;
    oid_esquema oid;
    oid_esquema_autorizacion oid;
BEGIN
    SELECT oid INTO oid_propietario
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_ejecucion_documental_v4_propietario';
    SELECT oid INTO oid_esquema
      FROM pg_catalog.pg_namespace
     WHERE nspname = 'vec_ejecucion_documental_v4';
    SELECT oid INTO oid_esquema_autorizacion
      FROM pg_catalog.pg_namespace
     WHERE nspname = 'vec_autorizacion';
    IF oid_propietario IS NULL
       OR oid_esquema IS NULL
       OR oid_esquema_autorizacion IS NULL THEN
        RAISE EXCEPTION 'faltan propietario o esquemas de la integración V4';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type AS tipo
         WHERE tipo.typnamespace = oid_esquema_autorizacion
           AND tipo.typtype = 'c'
           AND tipo.typname IN (
               'sesion_autenticacion_v1',
               'control_sesion_v1',
               'control_sesion_actual_v1',
               'contexto_actor_v1',
               'contexto_actor_actual_v1'
           )
           AND pg_catalog.has_type_privilege(
               'public', tipo.oid, 'USAGE'
           )
    ) THEN
        RAISE EXCEPTION
            'PUBLIC conserva tipos del vínculo autenticación-actor';
    END IF;

    IF pg_catalog.to_regprocedure(
           'vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos()'
       ) IS NULL OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_event_trigger AS disparador
         WHERE disparador.evtname =
                   'vec_ejecucion_documental_v4_cerrar_acl_tipos'
           AND disparador.evtevent = 'ddl_command_end'
           AND disparador.evtenabled = 'O'
           AND disparador.evtfoid = pg_catalog.to_regprocedure(
               'vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos()'
           )
    ) THEN
        RAISE EXCEPTION 'falta la guarda automatica de tipos V4';
    END IF;

    -- Sin una entrada de privilegios por defecto para TYPES, PostgreSQL vuelve
    -- a conceder USAGE a PUBLIC al crear nuevos tipos en una migracion futura.
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_default_acl AS defecto
         WHERE defecto.defaclrole = oid_propietario
           AND defecto.defaclnamespace = 0
           AND defecto.defaclobjtype = 'T'
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_default_acl AS defecto
          CROSS JOIN LATERAL pg_catalog.aclexplode(defecto.defaclacl)
              AS privilegio
         WHERE defecto.defaclrole = oid_propietario
           AND defecto.defaclnamespace = 0
           AND defecto.defaclobjtype = 'T'
           AND privilegio.grantee = 0
           AND privilegio.privilege_type = 'USAGE'
    ) THEN
        RAISE EXCEPTION 'los privilegios por defecto de TYPES no estan cerrados';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type AS tipo
         WHERE tipo.typnamespace = oid_esquema
           AND (
               pg_catalog.has_type_privilege(
                   'vec_ejecucion_documental_v4_emisor_capacidad',
                   tipo.oid,
                   'USAGE'
               ) OR pg_catalog.has_type_privilege(
                   'vec_ejecucion_documental_v4_ejecutor_atestado',
                   tipo.oid,
                   'USAGE'
               )
           )
    ) THEN
        RAISE EXCEPTION 'un runtime conserva USAGE sobre un tipo V4';
    END IF;

    -- Ninguna funcion de pgcrypto queda ejecutable por PUBLIC ni por runtimes.
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_extension AS extension
          JOIN pg_catalog.pg_depend AS dependencia
            ON dependencia.refclassid = 'pg_catalog.pg_extension'::regclass
           AND dependencia.refobjid = extension.oid
           AND dependencia.classid = 'pg_catalog.pg_proc'::regclass
           AND dependencia.deptype = 'e'
          JOIN pg_catalog.pg_proc AS funcion
            ON funcion.oid = dependencia.objid
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  funcion.proacl,
                  pg_catalog.acldefault('f', funcion.proowner)
              )
          ) AS privilegio
         WHERE extension.extname = 'pgcrypto'
           AND privilegio.grantee = 0
           AND privilegio.privilege_type = 'EXECUTE'
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_extension AS extension
          JOIN pg_catalog.pg_depend AS dependencia
            ON dependencia.refclassid = 'pg_catalog.pg_extension'::regclass
           AND dependencia.refobjid = extension.oid
           AND dependencia.classid = 'pg_catalog.pg_proc'::regclass
           AND dependencia.deptype = 'e'
          JOIN pg_catalog.pg_proc AS funcion
            ON funcion.oid = dependencia.objid
         WHERE extension.extname = 'pgcrypto'
           AND (
               pg_catalog.has_function_privilege(
                   'vec_ejecucion_documental_v4_emisor_capacidad',
                   funcion.oid,
                   'EXECUTE'
               ) OR pg_catalog.has_function_privilege(
                   'vec_ejecucion_documental_v4_ejecutor_atestado',
                   funcion.oid,
                   'EXECUTE'
               )
           )
    ) THEN
        RAISE EXCEPTION 'pgcrypto sigue expuesto a PUBLIC o a un runtime';
    END IF;

    IF NOT pg_catalog.has_function_privilege(
           'vec_ejecucion_documental_v4_propietario',
           'public.hmac(bytea,bytea,text)',
           'EXECUTE'
       ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_extension AS extension
          JOIN pg_catalog.pg_depend AS dependencia
            ON dependencia.refclassid = 'pg_catalog.pg_extension'::regclass
           AND dependencia.refobjid = extension.oid
           AND dependencia.classid = 'pg_catalog.pg_proc'::regclass
           AND dependencia.deptype = 'e'
          JOIN pg_catalog.pg_proc AS funcion
            ON funcion.oid = dependencia.objid
         WHERE extension.extname = 'pgcrypto'
           AND funcion.oid <>
               'public.hmac(bytea,bytea,text)'::pg_catalog.regprocedure
           AND pg_catalog.has_function_privilege(
               'vec_ejecucion_documental_v4_propietario',
               funcion.oid,
               'EXECUTE'
           )
    ) THEN
        RAISE EXCEPTION 'el propietario no tiene exclusivamente HMAC bytea';
    END IF;
END
$privilegios$;

ROLLBACK;
