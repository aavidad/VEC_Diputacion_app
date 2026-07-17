BEGIN;
SET LOCAL search_path = pg_catalog;

DO $comprobar$
DECLARE
    oid_propietario oid;
    oid_lector oid;
    oid_esquema oid;
    oid_salida oid;
    nombres_salida text[] := ARRAY[
        'revision',
        'huella_configuracion_sha256',
        'configuracion_publicada_en',
        'configuracion_expira_en',
        'configuracion_estado',
        'configuracion_revocada_en',
        'clave_id',
        'algoritmo_cose',
        'suite',
        'audiencia_despliegue',
        'clave_publica_spki',
        'huella_clave_spki_sha256',
        'raiz_valida_desde',
        'raiz_valida_hasta',
        'raiz_estado',
        'raiz_revocada_en'
    ];
BEGIN
    SELECT oid INTO oid_propietario
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_confianza_atestacion_v2_propietario'
       AND rolcanlogin IS FALSE AND rolsuper IS FALSE
       AND rolcreatedb IS FALSE AND rolcreaterole IS FALSE
       AND rolinherit IS FALSE AND rolreplication IS FALSE
       AND rolbypassrls IS FALSE;
    SELECT oid INTO oid_lector
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_confianza_atestacion_v2_lector_autoridad'
       AND rolcanlogin IS FALSE AND rolsuper IS FALSE
       AND rolcreatedb IS FALSE AND rolcreaterole IS FALSE
       AND rolinherit IS TRUE AND rolreplication IS FALSE
       AND rolbypassrls IS FALSE;
    SELECT oid INTO oid_esquema
      FROM pg_catalog.pg_namespace
     WHERE nspname = 'vec_confianza_atestacion_v2';
    SELECT oid INTO oid_salida
      FROM pg_catalog.pg_proc
     WHERE oid = pg_catalog.to_regprocedure(
         'vec_confianza_atestacion_v2.obtener_confianza_actual()'
     );
    IF oid_propietario IS NULL OR oid_lector IS NULL
       OR oid_esquema IS NULL OR oid_salida IS NULL THEN
        RAISE EXCEPTION 'faltan roles u objetos de confianza V2';
    END IF;

    IF NOT pg_catalog.has_schema_privilege(
           'vec_confianza_atestacion_v2_lector_autoridad',
           oid_esquema,
           'USAGE'
       ) OR pg_catalog.has_schema_privilege(
           'vec_confianza_atestacion_v2_lector_autoridad',
           oid_esquema,
           'CREATE'
       ) THEN
        RAISE EXCEPTION 'ACL de esquema incorrecta para el lector';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_database AS base
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  base.datacl,
                  pg_catalog.acldefault('d', base.datdba)
              )
          ) AS privilegio
         WHERE base.datname = current_database()
           AND privilegio.grantee = 0
           AND privilegio.privilege_type IN (
               'CONNECT', 'CREATE', 'TEMPORARY'
           )
    ) OR pg_catalog.has_schema_privilege(
        'public', oid_esquema, 'USAGE,CREATE'
    ) THEN
        RAISE EXCEPTION 'PUBLIC conserva acceso a base o esquema V2';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS clase
         WHERE clase.relnamespace = oid_esquema
           AND clase.relkind IN ('r', 'p', 'v', 'm', 'S')
           AND (
               pg_catalog.has_table_privilege(
                   'vec_confianza_atestacion_v2_lector_autoridad',
                   clase.oid,
                   'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
               ) OR EXISTS (
                   SELECT 1
                     FROM pg_catalog.aclexplode(
                         COALESCE(
                             clase.relacl,
                             pg_catalog.acldefault(
                                 CASE WHEN clase.relkind = 'S' THEN 'S'::"char"
                                      ELSE 'r'::"char" END,
                                 clase.relowner
                             )
                         )
                     ) AS privilegio
                    WHERE privilegio.grantee = 0
               )
           )
    ) THEN
        RAISE EXCEPTION 'un lector o PUBLIC conserva privilegios de relacion';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS clase
         WHERE clase.relnamespace = oid_esquema
           AND clase.relkind IN ('r', 'p')
           AND (clase.relrowsecurity IS FALSE
                OR clase.relforcerowsecurity IS FALSE)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_policy AS politica
          JOIN pg_catalog.pg_class AS clase
            ON clase.oid = politica.polrelid
         WHERE clase.relnamespace = oid_esquema
           AND (
               politica.polname <> 'propietario_exacto'
               OR politica.polroles <> ARRAY[oid_propietario]
               OR politica.polcmd <> '*'
               OR position(
                   'vec_confianza_atestacion_v2_propietario' IN
                   pg_catalog.pg_get_expr(
                       politica.polqual, politica.polrelid
                   )
               ) = 0
               OR position(
                   'vec_confianza_atestacion_v2_propietario' IN
                   pg_catalog.pg_get_expr(
                       politica.polwithcheck, politica.polrelid
                   )
               ) = 0
           )
    ) OR (SELECT count(*)
            FROM pg_catalog.pg_policy AS politica
            JOIN pg_catalog.pg_class AS clase
              ON clase.oid = politica.polrelid
           WHERE clase.relnamespace = oid_esquema) <> 8 THEN
        RAISE EXCEPTION 'RLS no esta cerrado al propietario exacto';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type AS tipo
         WHERE tipo.typnamespace = oid_esquema
           AND tipo.typelem = 0
           AND tipo.typisdefined
           AND (
               pg_catalog.has_type_privilege(
                   'vec_confianza_atestacion_v2_lector_autoridad',
                   tipo.oid,
                   'USAGE'
               ) OR EXISTS (
                   SELECT 1
                     FROM pg_catalog.aclexplode(
                         COALESCE(
                             tipo.typacl,
                             pg_catalog.acldefault('T', tipo.typowner)
                         )
                     ) AS privilegio
                    WHERE privilegio.grantee = 0
                      AND privilegio.privilege_type = 'USAGE'
               )
           )
    ) THEN
        RAISE EXCEPTION 'un tipo V2 sigue expuesto al lector o a PUBLIC';
    END IF;

    IF NOT pg_catalog.has_function_privilege(
           'vec_confianza_atestacion_v2_lector_autoridad',
           oid_salida,
           'EXECUTE'
       ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
         WHERE funcion.pronamespace = oid_esquema
           AND funcion.oid <> oid_salida
           AND pg_catalog.has_function_privilege(
               'vec_confianza_atestacion_v2_lector_autoridad',
               funcion.oid,
               'EXECUTE'
           )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  funcion.proacl,
                  pg_catalog.acldefault('f', funcion.proowner)
              )
          ) AS privilegio
         WHERE funcion.pronamespace = oid_esquema
           AND privilegio.grantee = 0
           AND privilegio.privilege_type = 'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'superficie de funciones demasiado amplia';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
         WHERE funcion.pronamespace = oid_esquema
           AND funcion.proconfig IS DISTINCT FROM ARRAY[
               'search_path=pg_catalog'
           ]::text[]
    ) THEN
        RAISE EXCEPTION 'una funcion V2 admite search_path no cerrado';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
         WHERE funcion.oid = oid_salida
           AND funcion.proowner = oid_propietario
           AND funcion.prosecdef IS TRUE
           AND funcion.pronargs = 0
           AND funcion.pronargdefaults = 0
           AND funcion.proretset IS TRUE
           AND cardinality(funcion.proallargtypes) = 16
           AND funcion.proargnames = nombres_salida
           AND funcion.proargmodes = array_fill(
               't'::"char", ARRAY[16]
           )
           AND funcion.proconfig = ARRAY[
               'search_path=pg_catalog'
           ]::text[]
           AND funcion.provolatile = 'v'
           AND funcion.prosrc !~* '\m(INSERT|UPDATE|DELETE|MERGE|TRUNCATE|COPY|CALL)\M'
    ) THEN
        RAISE EXCEPTION 'contrato o endurecimiento de la funcion de salida invalido';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_event_trigger AS disparador
         WHERE disparador.evtname =
                   'vec_confianza_atestacion_v2_cerrar_acl_tipos'
           AND disparador.evtevent = 'ddl_command_end'
           AND disparador.evtenabled = 'O'
           AND disparador.evtfoid = pg_catalog.to_regprocedure(
               'vec_confianza_atestacion_v2_guardia.cerrar_acl_tipos()'
           )
    ) THEN
        RAISE EXCEPTION 'falta la guarda de tipos futuros';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_default_acl AS defecto
          CROSS JOIN LATERAL pg_catalog.aclexplode(defecto.defaclacl)
              AS privilegio
         WHERE defecto.defaclrole = oid_propietario
           AND defecto.defaclobjtype IN ('f', 'T')
           AND privilegio.grantee = 0
    ) THEN
        RAISE EXCEPTION 'default ACL global reabre funciones o tipos';
    END IF;
END
$comprobar$;

ROLLBACK;
