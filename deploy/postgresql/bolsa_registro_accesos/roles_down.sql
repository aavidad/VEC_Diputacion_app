BEGIN;
SET LOCAL search_path = pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_registro_accesos:roles:v1', 0)
);

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'down de roles T13: requiere superusuario';
    END IF;
    IF pg_catalog.to_regnamespace('vec_bolsa_registro_accesos') IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.revalidar_decision_registro_accesos_bolsa_v2(jsonb,bytea,bytea,text,text,text,jsonb,text,text,text)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down de roles rechazado: queda el esquema T13';
    END IF;
END
$prevalidacion$;

REVOKE vec_bolsa_accesos_propietario FROM vec_bolsa_accesos_migrador;
DO $acl_base$
DECLARE
    rol text;
BEGIN
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_accesos_propietario', 'vec_bolsa_accesos_migrador',
        'vec_bolsa_accesos_registrador', 'vec_bolsa_accesos_consultor',
        'vec_bolsa_accesos_gobernador'
    ] LOOP
        EXECUTE format('REVOKE ALL ON DATABASE %I FROM %I',
                       current_database(), rol);
    END LOOP;
END
$acl_base$;

DROP ROLE vec_bolsa_accesos_gobernador;
DROP ROLE vec_bolsa_accesos_consultor;
DROP ROLE vec_bolsa_accesos_registrador;
DROP ROLE vec_bolsa_accesos_migrador;
DROP ROLE vec_bolsa_accesos_propietario;
COMMIT;
