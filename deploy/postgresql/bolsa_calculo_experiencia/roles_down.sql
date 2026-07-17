-- Reversion DBA conservadora. Debe ejecutarse sin administracion concurrente.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_calculo_experiencia:roles_down:v1', 0
    )
);

LOCK TABLE pg_catalog.pg_authid IN ACCESS EXCLUSIVE MODE;
LOCK TABLE pg_catalog.pg_auth_members IN ACCESS EXCLUSIVE MODE;
LOCK TABLE pg_catalog.pg_database IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
DECLARE
    rol text;
    oid_roles oid[];
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'down de roles del calculo: requiere superusuario';
    END IF;
    IF pg_catalog.to_regnamespace(
        'vec_bolsa_calculo_experiencia'
    ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down de roles rechazado: el esquema sigue instalado';
    END IF;

    SELECT array_agg(oid ORDER BY oid)
      INTO oid_roles
      FROM pg_catalog.pg_roles
     WHERE rolname = ANY (ARRAY[
         'vec_bolsa_calculo_experiencia_propietario',
         'vec_bolsa_calculo_experiencia_migrador',
         'vec_bolsa_calculo_experiencia_aplicacion',
         'vec_bolsa_calculo_experiencia_lector_operativo',
         'vec_bolsa_calculo_experiencia_publicador'
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
                          WHERE rolname =
                              'vec_bolsa_calculo_experiencia_propietario')
               AND member = (SELECT oid FROM pg_catalog.pg_roles
                              WHERE rolname =
                                  'vec_bolsa_calculo_experiencia_migrador')
           )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: existen membresias ajenas';
    END IF;

    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_calculo_experiencia_propietario',
        'vec_bolsa_calculo_experiencia_migrador',
        'vec_bolsa_calculo_experiencia_aplicacion',
        'vec_bolsa_calculo_experiencia_lector_operativo',
        'vec_bolsa_calculo_experiencia_publicador'
    ] LOOP
        IF EXISTS (
            SELECT 1 FROM pg_catalog.pg_roles
             WHERE rolname = rol
               AND (rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole
                    OR rolreplication OR rolbypassrls)
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'down rechazado: cambiaron opciones del rol',
                DETAIL = rol;
        END IF;
    END LOOP;
END
$prevalidacion$;

REVOKE vec_bolsa_calculo_experiencia_propietario
    FROM vec_bolsa_calculo_experiencia_migrador;

DO $retirar_acl_base$
DECLARE
    rol text;
BEGIN
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_calculo_experiencia_propietario',
        'vec_bolsa_calculo_experiencia_migrador',
        'vec_bolsa_calculo_experiencia_aplicacion',
        'vec_bolsa_calculo_experiencia_lector_operativo',
        'vec_bolsa_calculo_experiencia_publicador'
    ] LOOP
        EXECUTE format(
            'REVOKE ALL ON DATABASE %I FROM %I', current_database(), rol
        );
    END LOOP;
END
$retirar_acl_base$;

DROP ROLE vec_bolsa_calculo_experiencia_publicador;
DROP ROLE vec_bolsa_calculo_experiencia_lector_operativo;
DROP ROLE vec_bolsa_calculo_experiencia_aplicacion;
DROP ROLE vec_bolsa_calculo_experiencia_migrador;
DROP ROLE vec_bolsa_calculo_experiencia_propietario;
COMMIT;
