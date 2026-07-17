-- Reversion DBA conservadora. Debe ejecutarse tras retirar esquema y frontera.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_panel:roles_down:v1', 0)
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
            MESSAGE = 'down de roles del panel rechazado: requiere superusuario';
    END IF;
    IF to_regnamespace('vec_bolsa_panel') IS NOT NULL
       OR to_regprocedure(
          'vec_autorizacion.revalidar_decision_panel_bolsa_v2(jsonb,bytea,bytea,text,text,text,timestamp with time zone)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down de roles rechazado: quedan objetos instalados';
    END IF;

    SELECT array_agg(oid ORDER BY oid) INTO oid_roles
      FROM pg_catalog.pg_roles
     WHERE rolname = ANY (ARRAY[
         'vec_bolsa_panel_propietario',
         'vec_bolsa_panel_migrador',
         'vec_bolsa_panel_proyector',
         'vec_bolsa_panel_ejecutor_consulta',
         'vec_bolsa_panel_registrador_atestacion'
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
                          WHERE rolname = 'vec_bolsa_panel_propietario')
               AND member = (SELECT oid FROM pg_catalog.pg_roles
                              WHERE rolname = 'vec_bolsa_panel_migrador')
           )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: existen membresias ajenas';
    END IF;
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_panel_propietario', 'vec_bolsa_panel_migrador',
        'vec_bolsa_panel_proyector', 'vec_bolsa_panel_ejecutor_consulta',
        'vec_bolsa_panel_registrador_atestacion'
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

REVOKE vec_bolsa_panel_propietario FROM vec_bolsa_panel_migrador;

DO $retirar_acl_base$
DECLARE
    rol text;
BEGIN
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_panel_propietario', 'vec_bolsa_panel_migrador',
        'vec_bolsa_panel_proyector', 'vec_bolsa_panel_ejecutor_consulta',
        'vec_bolsa_panel_registrador_atestacion'
    ] LOOP
        EXECUTE format('REVOKE ALL ON DATABASE %I FROM %I',
                       current_database(), rol);
    END LOOP;
END
$retirar_acl_base$;

DROP ROLE vec_bolsa_panel_registrador_atestacion;
DROP ROLE vec_bolsa_panel_ejecutor_consulta;
DROP ROLE vec_bolsa_panel_proyector;
DROP ROLE vec_bolsa_panel_migrador;
DROP ROLE vec_bolsa_panel_propietario;
COMMIT;
