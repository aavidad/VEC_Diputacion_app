BEGIN;
SET LOCAL search_path = pg_catalog;

DO $prueba$
DECLARE
    tabla text;
    rol text;
    total integer;
BEGIN
    SELECT count(*) INTO total
      FROM pg_catalog.pg_tables
     WHERE schemaname = 'vec_bolsa_reglas_baremo';
    IF total <> 11 THEN
        RAISE EXCEPTION 'inventario inesperado de tablas: %', total;
    END IF;

    FOR tabla IN
        SELECT tablename FROM pg_catalog.pg_tables
         WHERE schemaname = 'vec_bolsa_reglas_baremo'
    LOOP
        IF NOT EXISTS (
            SELECT 1 FROM pg_catalog.pg_class
             WHERE oid = format(
                 'vec_bolsa_reglas_baremo.%I', tabla
             )::regclass
               AND relrowsecurity AND relforcerowsecurity
        ) THEN
            RAISE EXCEPTION 'RLS no forzada en %', tabla;
        END IF;
        IF EXISTS (
            SELECT 1
              FROM pg_catalog.pg_class AS clase,
                   LATERAL pg_catalog.aclexplode(COALESCE(
                       clase.relacl,
                       pg_catalog.acldefault('r', clase.relowner)
                   )) AS acl
             WHERE clase.oid = format(
                       'vec_bolsa_reglas_baremo.%I', tabla
                   )::regclass
               AND acl.grantee = 0
               AND acl.privilege_type IN (
                   'SELECT', 'INSERT', 'UPDATE', 'DELETE', 'TRUNCATE',
                   'REFERENCES', 'TRIGGER'
               )
        ) THEN
            RAISE EXCEPTION 'PUBLIC tiene privilegios sobre %', tabla;
        END IF;
        FOREACH rol IN ARRAY ARRAY[
            'vec_bolsa_reglas_baremo_ejecutor_gobierno',
            'vec_bolsa_reglas_baremo_ejecutor_consulta',
            'vec_bolsa_reglas_baremo_publicador_outbox'
        ] LOOP
            IF has_table_privilege(rol, format(
                   'vec_bolsa_reglas_baremo.%I', tabla
               ), 'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') THEN
                RAISE EXCEPTION '% tiene privilegios DML sobre %', rol, tabla;
            END IF;
        END LOOP;
    END LOOP;

    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_reglas_baremo_ejecutor_gobierno',
        'vec_bolsa_reglas_baremo_ejecutor_consulta',
        'vec_bolsa_reglas_baremo_publicador_outbox'
    ] LOOP
        IF has_schema_privilege(
               rol, 'vec_bolsa_reglas_baremo', 'USAGE'
           )
           OR has_function_privilege(
               rol,
               'vec_bolsa_reglas_baremo.confirmar_cambio_v1(jsonb,jsonb,bytea,bytea,bytea)',
               'EXECUTE'
           )
           OR has_function_privilege(
               rol,
               'vec_bolsa_reglas_baremo.obtener_version_exacta_v1(jsonb,jsonb,bytea,bytea)',
               'EXECUTE'
           ) THEN
            RAISE EXCEPTION 'funcion del almacen abierta sin composicion VEC-AD-2 para %', rol;
        END IF;
    END LOOP;
    IF EXISTS (
           SELECT 1
             FROM pg_catalog.pg_namespace AS espacio,
                  LATERAL pg_catalog.aclexplode(COALESCE(
                      espacio.nspacl,
                      pg_catalog.acldefault('n', espacio.nspowner)
                  )) AS acl
            WHERE espacio.nspname = 'vec_bolsa_reglas_baremo'
              AND acl.grantee = 0 AND acl.privilege_type = 'USAGE'
       ) OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc AS procedimiento,
                  LATERAL pg_catalog.aclexplode(COALESCE(
                      procedimiento.proacl,
                      pg_catalog.acldefault('f', procedimiento.proowner)
                  )) AS acl
            WHERE procedimiento.oid IN (
                      'vec_bolsa_reglas_baremo.confirmar_cambio_v1(jsonb,jsonb,bytea,bytea,bytea)'::regprocedure,
                      'vec_bolsa_reglas_baremo.obtener_version_exacta_v1(jsonb,jsonb,bytea,bytea)'::regprocedure
                  )
              AND acl.grantee = 0 AND acl.privilege_type = 'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'PUBLIC puede atravesar una puerta cerrada';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname LIKE 'vec_bolsa_reglas_baremo_%'
           AND (rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole
                OR rolreplication OR rolbypassrls)
    ) THEN
        RAISE EXCEPTION 'rol tecnico con capacidades incompatibles';
    END IF;
END
$prueba$;
ROLLBACK;
