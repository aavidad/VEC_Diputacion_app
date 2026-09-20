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
    IF pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.registrar_consumo_participaciones_propias_b11_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.revalidar_consumo_participaciones_propias_b11_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '2BP01',
            MESSAGE = 'retirada de rol B11 bloqueada por consumidor AD3 o historia';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_bolsa_llamamientos_propietario'
           AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
           AND NOT rolcreaterole AND NOT rolinherit AND NOT rolreplication
           AND NOT rolbypassrls
    )
       OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_autorizacion_atestada_v3_propietario'
           AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
           AND NOT rolcreaterole AND NOT rolinherit AND NOT rolreplication
           AND NOT rolbypassrls
    )
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_namespace n
            WHERE n.nspname = 'vec_autorizacion_atestada_v3'
              AND n.nspowner =
                  'vec_autorizacion_atestada_v3_propietario'::regrole
       )
       OR NOT pg_catalog.has_schema_privilege(
           'vec_bolsa_llamamientos_propietario',
           'vec_autorizacion_atestada_v3', 'USAGE'
       )
       OR pg_catalog.has_schema_privilege(
           'vec_bolsa_llamamientos_propietario',
           'vec_autorizacion_atestada_v3', 'CREATE'
       )
       -- USAGE es compartido por cualquier consumidor AD3 al que todavía se
       -- pueda ejecutar; no se retira mientras sobreviva alguno, aunque no
       -- pertenezca a la pareja nominal B11.
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc p
             JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
            WHERE n.nspname = 'vec_autorizacion_atestada_v3'
              AND pg_catalog.has_function_privilege(
                  'vec_bolsa_llamamientos_propietario', p.oid, 'EXECUTE'
              )
       )
       OR (SELECT count(*)
             FROM pg_catalog.pg_namespace n
             CROSS JOIN LATERAL pg_catalog.aclexplode(
                 coalesce(n.nspacl, pg_catalog.acldefault('n', n.nspowner))
             ) a
            WHERE n.nspname = 'vec_autorizacion_atestada_v3'
              AND a.grantee = 'vec_bolsa_llamamientos_propietario'::regrole
              AND a.privilege_type = 'USAGE' AND NOT a.is_grantable) <> 1
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_namespace n
             CROSS JOIN LATERAL pg_catalog.aclexplode(
                 coalesce(n.nspacl, pg_catalog.acldefault('n', n.nspowner))
             ) a
            WHERE n.nspname = 'vec_autorizacion_atestada_v3'
              AND a.grantee = 'vec_bolsa_llamamientos_propietario'::regrole
              AND (a.privilege_type <> 'USAGE' OR a.is_grantable)) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'ACL VEC-AD-3 B11 no es exactamente retirable';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members WHERE roleid = v_rol OR member = v_rol
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '2BP01',
            MESSAGE = 'retirada de rol B11 bloqueada por membresías';
    END IF;
END
$preservar$;

REVOKE USAGE ON SCHEMA vec_autorizacion_atestada_v3
    FROM vec_bolsa_llamamientos_propietario;

DO $retirada$
BEGIN
    IF pg_catalog.has_schema_privilege(
           'vec_bolsa_llamamientos_propietario',
           'vec_autorizacion_atestada_v3', 'USAGE'
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_namespace n
             CROSS JOIN LATERAL pg_catalog.aclexplode(
                 coalesce(n.nspacl, pg_catalog.acldefault('n', n.nspowner))
             ) a
            WHERE n.nspname = 'vec_autorizacion_atestada_v3'
              AND a.grantee = 'vec_bolsa_llamamientos_propietario'::regrole
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'ACL VEC-AD-3 B11 no retirada';
    END IF;
END
$retirada$;

DROP ROLE vec_bolsa_llamamientos_consultor_participaciones_propias;
COMMIT;
