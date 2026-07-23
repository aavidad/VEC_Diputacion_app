\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL TRANSACTION ISOLATION LEVEL SERIALIZABLE, READ ONLY;

DO $prueba$
DECLARE
    v_propietario oid := (
        SELECT oid
          FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_contratacion_temporal_propietario'
    );
    v_migrador oid := (
        SELECT oid
          FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_contratacion_temporal_migrador'
    );
    v_ejecutor oid := (
        SELECT oid
          FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_contratacion_temporal_ejecutor'
    );
BEGIN
    IF v_propietario IS NULL
       OR v_migrador IS NULL
       OR v_ejecutor IS NULL THEN
        RAISE EXCEPTION 'falta algún rol técnico';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE oid = ANY (
             ARRAY[v_propietario, v_migrador, v_ejecutor]
         )
           AND (
               rolcanlogin
               OR rolsuper
               OR rolcreatedb
               OR rolcreaterole
               OR rolreplication
               OR rolbypassrls
           )
    ) THEN
        RAISE EXCEPTION 'un rol técnico posee facultades administrativas';
    END IF;
    IF pg_catalog.has_database_privilege(
        v_propietario,
        current_database(),
        'CREATE'
    ) OR pg_catalog.has_database_privilege(
        v_migrador,
        current_database(),
        'CREATE'
    ) OR pg_catalog.has_database_privilege(
        v_ejecutor,
        current_database(),
        'CREATE'
    ) THEN
        RAISE EXCEPTION 'un rol técnico puede crear esquemas ajenos';
    END IF;
    IF (
        SELECT nspowner <> v_propietario
          FROM pg_catalog.pg_namespace
         WHERE nspname = 'vec_contratacion_temporal'
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS objeto
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = objeto.relnamespace
         WHERE espacio.nspname = 'vec_contratacion_temporal'
           AND objeto.relkind IN ('r', 'p', 'S', 'v', 'm')
           AND objeto.relowner <> v_propietario
    ) THEN
        RAISE EXCEPTION 'hay objetos sin propietario técnico cerrado';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS procedimiento
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = procedimiento.pronamespace
         WHERE espacio.nspname = 'vec_contratacion_temporal'
           AND procedimiento.proowner <> v_propietario
    ) THEN
        RAISE EXCEPTION 'hay funciones sin propietario técnico cerrado';
    END IF;
    IF NOT (
        SELECT procedimiento.prosecdef
               AND procedimiento.provolatile = 'v'
               AND procedimiento.proparallel = 'u'
               AND procedimiento.proconfig @> ARRAY[
                   'search_path=pg_catalog'
               ]::text[]
          FROM pg_catalog.pg_proc AS procedimiento
         WHERE procedimiento.oid =
               'vec_contratacion_temporal.preparar_alta_v1(jsonb)'::regprocedure
    ) THEN
        RAISE EXCEPTION 'la función pública no conserva su cierre esperado';
    END IF;
    IF NOT pg_catalog.has_schema_privilege(
        v_ejecutor,
        'vec_contratacion_temporal',
        'USAGE'
    ) THEN
        RAISE EXCEPTION 'el ejecutor no puede resolver el esquema';
    END IF;
    IF NOT pg_catalog.has_function_privilege(
        v_ejecutor,
        'vec_contratacion_temporal.preparar_alta_v1(jsonb)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'el ejecutor no puede invocar la función';
    END IF;
    IF pg_catalog.has_table_privilege(
        v_ejecutor,
        'vec_contratacion_temporal.identidad_reserva_alta',
        'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
    ) OR pg_catalog.has_table_privilege(
        v_ejecutor,
        'vec_contratacion_temporal.reserva_alta_version',
        'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
    ) OR pg_catalog.has_table_privilege(
        v_ejecutor,
        'vec_contratacion_temporal.reserva_alta_actual',
        'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
    ) THEN
        RAISE EXCEPTION 'el ejecutor conserva privilegios directos';
    END IF;
    IF pg_catalog.pg_has_role(
        v_ejecutor,
        'vec_contratacion_temporal_propietario',
        'MEMBER'
    ) OR pg_catalog.pg_has_role(
        v_ejecutor,
        'vec_contratacion_temporal_migrador',
        'MEMBER'
    ) THEN
        RAISE EXCEPTION 'el ejecutor puede escalar a roles de gestión';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS procedimiento
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              coalesce(
                  procedimiento.proacl,
                  pg_catalog.acldefault(
                      'f',
                      procedimiento.proowner
                  )
              )
          ) AS privilegio
         WHERE procedimiento.oid =
               'vec_contratacion_temporal.preparar_alta_v1(jsonb)'::regprocedure
           AND privilegio.grantee = 0
           AND privilegio.privilege_type = 'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'PUBLIC conserva ejecución de la función';
    END IF;
END
$prueba$;

COMMIT;
