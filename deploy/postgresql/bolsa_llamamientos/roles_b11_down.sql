-- Solo retira el grupo técnico B11 si la fachada y sus membresías ya no existen.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_llamamientos:roles:b11:down', 0)
);

DO $preservar$
DECLARE
    v_rol oid := pg_catalog.to_regrole(
        'vec_bolsa_llamamientos_consultor_participaciones_propias'
    );
BEGIN
    IF v_rol IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'rol exterior B11 ausente';
    END IF;
    IF pg_catalog.to_regprocedure(
           'vec_bolsa_llamamientos.consultar_participaciones_propias_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '2BP01',
            MESSAGE = 'retirada de rol B11 bloqueada por fachada instalada';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members WHERE roleid = v_rol OR member = v_rol
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '2BP01',
            MESSAGE = 'retirada de rol B11 bloqueada por membresías';
    END IF;
END
$preservar$;

DROP ROLE vec_bolsa_llamamientos_consultor_participaciones_propias;
COMMIT;
