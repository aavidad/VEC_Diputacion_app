-- Retirada protegida de la revalidación final nominal de consultas RRHH.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v3:migracion:000005', 0
    )
);

DO $prevalidacion$
DECLARE
    v_interna oid := pg_catalog.to_regprocedure(
        'vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    );
    v_cuadro oid := pg_catalog.to_regprocedure(
        'vec_autorizacion_atestada_v3.revalidar_consumo_consulta_cuadro_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    );
    v_detalle oid := pg_catalog.to_regprocedure(
        'vec_autorizacion_atestada_v3.revalidar_consumo_consulta_detalle_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    );
    v_propietario oid;
    v_propietario_ct oid;
BEGIN
    SELECT oid INTO v_propietario
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_autorizacion_atestada_v3_propietario'
       AND NOT rolcanlogin
       AND NOT rolsuper
       AND NOT rolcreatedb
       AND NOT rolcreaterole
       AND NOT rolinherit
       AND NOT rolreplication
       AND NOT rolbypassrls;
    SELECT oid INTO v_propietario_ct
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_contratacion_temporal_propietario'
       AND NOT rolcanlogin
       AND NOT rolsuper
       AND NOT rolcreatedb
       AND NOT rolcreaterole
       AND NOT rolinherit
       AND NOT rolreplication
       AND NOT rolbypassrls;

    IF v_interna IS NULL OR v_cuadro IS NULL OR v_detalle IS NULL
       OR v_propietario IS NULL OR v_propietario_ct IS NULL
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc f
             JOIN pg_catalog.pg_language l ON l.oid = f.prolang
            WHERE f.oid = ANY(ARRAY[v_interna, v_cuadro, v_detalle])
              AND (
                  f.proowner <> v_propietario
                  OR l.lanname <> 'plpgsql'
                  OR NOT f.prosecdef
                  OR f.provolatile <> 'v'
                  OR f.proisstrict
                  OR f.proleakproof
                  OR f.proparallel <> 'u'
                  OR f.prokind <> 'f'
                  OR f.proconfig IS DISTINCT FROM
                     ARRAY[
                         'search_path=pg_catalog',
                         'lock_timeout=1s'
                     ]::text[]
              )
       )
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_proc f
            WHERE f.oid = ANY(ARRAY[v_interna, v_cuadro, v_detalle])
       ) <> 3 THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'catálogo de revalidación final RRHH divergente';
    END IF;

    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_proc f
         CROSS JOIN LATERAL pg_catalog.aclexplode(
             COALESCE(
                 f.proacl,
                 pg_catalog.acldefault('f', f.proowner)
             )
         ) a
         WHERE f.oid = v_interna
           AND a.privilege_type = 'EXECUTE'
    ) <> 1
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc f
            CROSS JOIN LATERAL pg_catalog.aclexplode(
                COALESCE(
                    f.proacl,
                    pg_catalog.acldefault('f', f.proowner)
                )
            ) a
            WHERE f.oid = v_interna
              AND a.grantee = v_propietario
              AND a.privilege_type = 'EXECUTE'
              AND NOT a.is_grantable
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc f
            CROSS JOIN LATERAL pg_catalog.aclexplode(
                COALESCE(
                    f.proacl,
                    pg_catalog.acldefault('f', f.proowner)
                )
            ) a
            WHERE f.oid = v_interna
              AND (
                  a.privilege_type <> 'EXECUTE'
                  OR a.grantee <> v_propietario
              )
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'ACL interna de revalidación final RRHH divergente';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc f
         WHERE f.oid = ANY(ARRAY[v_cuadro, v_detalle])
           AND (
               SELECT pg_catalog.count(*)
                 FROM pg_catalog.aclexplode(
                     COALESCE(
                         f.proacl,
                         pg_catalog.acldefault('f', f.proowner)
                     )
                 ) a
                WHERE a.privilege_type = 'EXECUTE'
           ) <> 2
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc f
         CROSS JOIN LATERAL pg_catalog.aclexplode(
             COALESCE(
                 f.proacl,
                 pg_catalog.acldefault('f', f.proowner)
             )
         ) a
         WHERE f.oid = ANY(ARRAY[v_cuadro, v_detalle])
           AND (
               a.privilege_type <> 'EXECUTE'
               OR a.grantee NOT IN (v_propietario, v_propietario_ct)
               OR (
                   a.grantee = v_propietario
                   AND a.is_grantable
               )
               OR (
                   a.grantee = v_propietario_ct
                   AND a.is_grantable
               )
           )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc f
         WHERE f.oid = ANY(ARRAY[v_cuadro, v_detalle])
           AND (
               NOT pg_catalog.has_function_privilege(
                   v_propietario_ct, f.oid, 'EXECUTE'
               )
           )
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'ACL exterior de revalidación final RRHH divergente';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc f
          JOIN pg_catalog.pg_namespace n ON n.oid = f.pronamespace
         WHERE n.nspname = 'vec_autorizacion_atestada_v3'
           AND f.proname ~
               '^revalidar_consumo_consulta_.*rrhh_v3'
           AND f.oid <> ALL(ARRAY[v_interna, v_cuadro, v_detalle])
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_depend d
         WHERE d.refclassid = 'pg_catalog.pg_proc'::regclass
           AND d.refobjid = ANY(ARRAY[v_interna, v_cuadro, v_detalle])
           AND NOT (
               d.classid = 'pg_catalog.pg_proc'::regclass
               AND d.objid = ANY(ARRAY[v_interna, v_cuadro, v_detalle])
           )
           AND d.deptype NOT IN ('i', 'e')
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc f
         WHERE f.oid <> ALL(ARRAY[v_interna, v_cuadro, v_detalle])
           AND f.prokind IN ('f', 'p')
           AND (
               pg_catalog.pg_get_functiondef(f.oid) LIKE
                   '%revalidar_consumo_consulta_cuadro_rrhh_v3_atestada%'
               OR pg_catalog.pg_get_functiondef(f.oid) LIKE
                   '%revalidar_consumo_consulta_detalle_rrhh_v3_atestada%'
           )
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '2BP01',
            MESSAGE = 'dependencias impiden retirar revalidación final RRHH';
    END IF;
END
$prevalidacion$;

DROP FUNCTION
    vec_autorizacion_atestada_v3
    .revalidar_consumo_consulta_cuadro_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    );
DROP FUNCTION
    vec_autorizacion_atestada_v3
    .revalidar_consumo_consulta_detalle_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    );
DROP FUNCTION
    vec_autorizacion_atestada_v3
    .revalidar_consumo_consulta_rrhh_v3_interna(
        text, bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    );

COMMIT;
