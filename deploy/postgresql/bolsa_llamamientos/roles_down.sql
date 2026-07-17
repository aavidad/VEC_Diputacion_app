BEGIN;
SET LOCAL search_path = pg_catalog;

DO $prevalidacion$
BEGIN
    IF to_regnamespace('vec_bolsa_llamamientos') IS NOT NULL OR
       to_regprocedure(
          'vec_autorizacion.revalidar_decision_bolsa_llamamientos_v1(jsonb,bytea,bytea,text,text,text,jsonb,timestamp with time zone)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retire primero las migraciones de llamamientos';
    END IF;
END
$prevalidacion$;

DO $revocar_base$
DECLARE rol text;
BEGIN
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_llamamientos_despachador_outbox',
        'vec_bolsa_llamamientos_registrador_atestacion',
        'vec_bolsa_llamamientos_proyector_autoritativo',
        'vec_bolsa_llamamientos_ejecutor',
        'vec_bolsa_llamamientos_migrador',
        'vec_bolsa_llamamientos_propietario'
    ] LOOP
        EXECUTE format(
            'REVOKE ALL PRIVILEGES ON DATABASE %I FROM %I',
            current_database(), rol
        );
    END LOOP;
END
$revocar_base$;

REVOKE vec_bolsa_llamamientos_propietario
    FROM vec_bolsa_llamamientos_migrador;

-- Restaura el default nativo para eliminar pg_default_acl del propietario.
-- No concede acceso a objetos existentes: el esquema ya no existe.
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_bolsa_llamamientos_propietario
    GRANT EXECUTE ON FUNCTIONS TO PUBLIC;

DROP ROLE vec_bolsa_llamamientos_despachador_outbox;
DROP ROLE vec_bolsa_llamamientos_registrador_atestacion;
DROP ROLE vec_bolsa_llamamientos_proyector_autoritativo;
DROP ROLE vec_bolsa_llamamientos_ejecutor;
DROP ROLE vec_bolsa_llamamientos_migrador;
DROP ROLE vec_bolsa_llamamientos_propietario;
COMMIT;
