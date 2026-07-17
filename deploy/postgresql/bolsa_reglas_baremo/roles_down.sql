-- Reversion DBA conservadora. Debe ejecutarse despues de retirar funciones,
-- esquema y frontera de autorizacion.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_reglas_baremo:roles_down:v1', 0)
);

LOCK TABLE pg_catalog.pg_authid IN ACCESS EXCLUSIVE MODE;
LOCK TABLE pg_catalog.pg_auth_members IN ACCESS EXCLUSIVE MODE;
LOCK TABLE pg_catalog.pg_database IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
DECLARE
    oid_roles oid[];
    rol text;
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'down de roles rechazado: requiere superusuario';
    END IF;
    IF pg_catalog.to_regnamespace('vec_bolsa_reglas_baremo') IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.revalidar_decision_reglas_baremo_v1(jsonb,bytea,bytea,text,text,text,text,timestamp with time zone)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down de roles rechazado: quedan objetos instalados';
    END IF;

    SELECT array_agg(oid ORDER BY oid)
      INTO oid_roles
      FROM pg_catalog.pg_roles
     WHERE rolname = ANY (ARRAY[
         'vec_bolsa_reglas_baremo_propietario',
         'vec_bolsa_reglas_baremo_migrador',
         'vec_bolsa_reglas_baremo_ejecutor_gobierno',
         'vec_bolsa_reglas_baremo_ejecutor_consulta',
         'vec_bolsa_reglas_baremo_publicador_outbox'
     ]);
    IF cardinality(oid_roles) <> 5 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: inventario de roles incompleto';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members
         WHERE (roleid = ANY (oid_roles) OR member = ANY (oid_roles)
                OR grantor = ANY (oid_roles))
           AND NOT (
               roleid = (SELECT oid FROM pg_catalog.pg_roles
                          WHERE rolname = 'vec_bolsa_reglas_baremo_propietario')
               AND member = (SELECT oid FROM pg_catalog.pg_roles
                              WHERE rolname = 'vec_bolsa_reglas_baremo_migrador')
           )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: existen membresias ajenas';
    END IF;
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_reglas_baremo_propietario',
        'vec_bolsa_reglas_baremo_migrador',
        'vec_bolsa_reglas_baremo_ejecutor_gobierno',
        'vec_bolsa_reglas_baremo_ejecutor_consulta',
        'vec_bolsa_reglas_baremo_publicador_outbox'
    ] LOOP
        IF EXISTS (
            SELECT 1 FROM pg_catalog.pg_roles
             WHERE rolname = rol
               AND (rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole
                    OR rolreplication OR rolbypassrls)
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'down rechazado: cambiaron las opciones de un rol',
                DETAIL = rol;
        END IF;
    END LOOP;
END
$prevalidacion$;

REVOKE vec_bolsa_reglas_baremo_propietario
    FROM vec_bolsa_reglas_baremo_migrador;

DO $retirar_acl_base$
DECLARE
    rol text;
BEGIN
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_reglas_baremo_propietario',
        'vec_bolsa_reglas_baremo_migrador',
        'vec_bolsa_reglas_baremo_ejecutor_gobierno',
        'vec_bolsa_reglas_baremo_ejecutor_consulta',
        'vec_bolsa_reglas_baremo_publicador_outbox'
    ] LOOP
        EXECUTE format(
            'REVOKE ALL ON DATABASE %I FROM %I', current_database(), rol
        );
    END LOOP;
END
$retirar_acl_base$;

DROP ROLE vec_bolsa_reglas_baremo_publicador_outbox;
DROP ROLE vec_bolsa_reglas_baremo_ejecutor_consulta;
DROP ROLE vec_bolsa_reglas_baremo_ejecutor_gobierno;
DROP ROLE vec_bolsa_reglas_baremo_migrador;
DROP ROLE vec_bolsa_reglas_baremo_propietario;
COMMIT;
