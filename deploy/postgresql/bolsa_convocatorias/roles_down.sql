-- Reversion DBA conservadora. Debe ejecutarse en ventana de mantenimiento.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_convocatorias:roles_down:v1', 0)
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
            MESSAGE = 'down de roles rechazado: requiere superusuario';
    END IF;
    IF pg_catalog.to_regnamespace('vec_bolsa_convocatorias') IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.revalidar_decision_bolsa_convocatorias_v1(jsonb,bytea,bytea,text,text,text,jsonb,timestamp with time zone)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down de roles rechazado: quedan objetos instalados';
    END IF;

    SELECT array_agg(oid ORDER BY oid)
      INTO oid_roles
      FROM pg_catalog.pg_roles
     WHERE rolname = ANY (ARRAY[
         'vec_bolsa_convocatorias_propietario',
         'vec_bolsa_convocatorias_migrador',
         'vec_bolsa_convocatorias_ejecutor_consulta',
         'vec_bolsa_convocatorias_proyector_gobierno',
         'vec_bolsa_convocatorias_registrador_atestacion',
         'vec_bolsa_convocatorias_verificador_recibo'
     ]);
    IF cardinality(oid_roles) <> 6 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: inventario de roles incompleto';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members
         WHERE (roleid = ANY (oid_roles) OR member = ANY (oid_roles)
                OR grantor = ANY (oid_roles))
           AND NOT (
               roleid = (SELECT oid FROM pg_catalog.pg_roles
                          WHERE rolname = 'vec_bolsa_convocatorias_propietario')
               AND member = (SELECT oid FROM pg_catalog.pg_roles
                              WHERE rolname = 'vec_bolsa_convocatorias_migrador')
           )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: existen membresias ajenas';
    END IF;
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_convocatorias_propietario',
        'vec_bolsa_convocatorias_migrador',
        'vec_bolsa_convocatorias_ejecutor_consulta',
        'vec_bolsa_convocatorias_proyector_gobierno',
        'vec_bolsa_convocatorias_registrador_atestacion',
        'vec_bolsa_convocatorias_verificador_recibo'
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

REVOKE vec_bolsa_convocatorias_propietario
    FROM vec_bolsa_convocatorias_migrador;

DO $retirar_acl_base$
DECLARE
    rol text;
BEGIN
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_convocatorias_propietario',
        'vec_bolsa_convocatorias_migrador',
        'vec_bolsa_convocatorias_ejecutor_consulta',
        'vec_bolsa_convocatorias_proyector_gobierno',
        'vec_bolsa_convocatorias_registrador_atestacion',
        'vec_bolsa_convocatorias_verificador_recibo'
    ] LOOP
        EXECUTE format('REVOKE ALL ON DATABASE %I FROM %I', current_database(), rol);
    END LOOP;
END
$retirar_acl_base$;

DROP ROLE vec_bolsa_convocatorias_registrador_atestacion;
DROP ROLE vec_bolsa_convocatorias_verificador_recibo;
DROP ROLE vec_bolsa_convocatorias_proyector_gobierno;
DROP ROLE vec_bolsa_convocatorias_ejecutor_consulta;
DROP ROLE vec_bolsa_convocatorias_migrador;
DROP ROLE vec_bolsa_convocatorias_propietario;
COMMIT;
