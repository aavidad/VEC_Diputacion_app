BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock_shared(
    pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:rol-contexto-corporativo-rrhh-selector:v1',
        0
    )
);
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_identidad_sesiones_v1:fachada-contexto-corporativo-rrhh:v1',
        0
    )
);

DO $prevalidacion$
DECLARE
    propia oid := pg_catalog.to_regprocedure(
        'vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(text,text)'
    );
    base oid := pg_catalog.to_regprocedure(
        'vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)'
    );
    propietario oid := pg_catalog.to_regrole(
        'vec_identidad_sesiones_v1_propietario'
    );
    consumidor oid := pg_catalog.to_regrole(
        'vec_contexto_actor_v1_propietario'
    );
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) OR propia IS NULL OR base IS NULL OR propietario IS NULL
       OR consumidor IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'retirada de fachada corporativa RRHH rechazada';
    END IF;
    IF (
        SELECT pg_catalog.count(*) <> 1
          FROM pg_catalog.pg_proc AS p
         WHERE p.pronamespace =
               'vec_identidad_sesiones_v1'::regnamespace
           AND p.proname =
               'revalidar_contexto_corporativo_rrhh_v1'
    ) OR (
        SELECT pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to(
                pg_catalog.pg_get_functiondef(propia), 'UTF8'
            )
        ), 'hex')
    ) <> '73a6bb319a24bab619335ae550465c1d0ca43cf8b6c71ea6a03efa246ffb7e78'
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc AS p
             JOIN pg_catalog.pg_language AS l ON l.oid = p.prolang
            WHERE p.oid = propia AND l.lanname = 'sql'
              AND p.proowner = propietario AND p.prokind = 'f'
              AND p.provolatile = 'v' AND p.proparallel = 'u'
              AND p.prosecdef AND NOT p.proleakproof
              AND NOT p.proisstrict AND p.proretset
              AND p.pronargs = 2 AND p.pronargdefaults = 0
              AND p.proargtypes = '25 25'::oidvector
              AND p.proconfig = ARRAY[
                  'search_path=pg_catalog', 'lock_timeout=1s'
              ]::text[]
              AND p.prosqlbody IS NOT NULL
       ) OR (
           SELECT pg_catalog.count(*) <> 1
             FROM pg_catalog.pg_depend AS d
            WHERE d.classid = 'pg_catalog.pg_proc'::regclass
              AND d.objid = propia AND d.objsubid = 0
              AND d.refclassid = 'pg_catalog.pg_proc'::regclass
              AND d.refobjid = base AND d.refobjsubid = 0
              AND d.deptype = 'n'
       ) OR (
           SELECT pg_catalog.count(*) <> 2
               OR pg_catalog.count(DISTINCT a.grantee) <> 2
               OR pg_catalog.bool_and(
                   a.grantor = propietario
                   AND a.grantee = ANY (ARRAY[propietario, consumidor])
                   AND a.privilege_type = 'EXECUTE'
                   AND NOT a.is_grantable
               ) IS NOT TRUE
             FROM pg_catalog.pg_proc AS p
             CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) AS a
            WHERE p.oid = propia
       ) OR (
           SELECT pg_catalog.count(*) <> 1
               OR pg_catalog.bool_and(
                   a.grantor = n.nspowner AND a.grantee = consumidor
                   AND a.privilege_type = 'USAGE'
                   AND NOT a.is_grantable
               ) IS NOT TRUE
             FROM pg_catalog.pg_namespace AS n
             CROSS JOIN LATERAL pg_catalog.aclexplode(n.nspacl) AS a
            WHERE n.oid = 'vec_identidad_sesiones_v1'::regnamespace
              AND (a.grantee = consumidor OR a.grantor = consumidor)
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada de fachada corporativa RRHH rechazada';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS p
          JOIN pg_catalog.pg_language AS l ON l.oid = p.prolang
         WHERE p.oid = base AND l.lanname = 'plpgsql'
           AND p.proowner = propietario AND p.prokind = 'f'
           AND p.provolatile = 'v' AND p.proparallel = 'u'
           AND p.prosecdef AND NOT p.proleakproof
           AND NOT p.proisstrict AND p.proretset
           AND p.pronargs = 2 AND p.pronargdefaults = 0
           AND p.proargtypes = '25 25'::oidvector
           AND p.proconfig =
               ARRAY['search_path=pg_catalog, pg_temp']::text[]
           AND p.prosqlbody IS NULL
           AND pg_catalog.encode(pg_catalog.sha256(
               pg_catalog.convert_to(p.prosrc, 'UTF8')
           ), 'hex') =
               '277ab207e23d261522c6397578ebba9412f5111d25b985896756fe62677c3d46'
           AND pg_catalog.octet_length(p.prosrc) = 7783
           AND (
               SELECT pg_catalog.count(*) = 2
                  AND pg_catalog.count(DISTINCT a.grantee) = 2
                  AND pg_catalog.bool_and(
                      a.grantor = propietario
                      AND a.grantee = ANY (ARRAY[
                          propietario,
                          'vec_identidad_sesiones_v1_revalidador'::regrole
                      ])
                      AND a.privilege_type = 'EXECUTE'
                      AND NOT a.is_grantable
                  )
                 FROM pg_catalog.aclexplode(p.proacl) AS a
           )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS p
         WHERE p.pronamespace = 'vec_contexto_actor_v1'::regnamespace
           AND p.proname = ANY (ARRAY[
               'resolver_y_registrar_contexto_corporativo_rrhh_v1',
               'reconciliar_contexto_corporativo_rrhh_v1',
               'acreditar_uso_registro_contexto_corporativo_rrhh_v1'
           ]::name[])
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada de fachada corporativa RRHH rechazada';
    END IF;
END
$prevalidacion$;

SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;

REVOKE EXECUTE ON FUNCTION
    vec_identidad_sesiones_v1.
    revalidar_contexto_corporativo_rrhh_v1(text, text)
    FROM vec_contexto_actor_v1_propietario;

DROP FUNCTION
    vec_identidad_sesiones_v1.
    revalidar_contexto_corporativo_rrhh_v1(text, text)
    RESTRICT;

DO $uso_esquema$
DECLARE
    consumidor oid := 'vec_contexto_actor_v1_propietario'::regrole;
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS p
          CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) AS a
         WHERE p.pronamespace =
               'vec_identidad_sesiones_v1'::regnamespace
           AND a.grantee = consumidor
        UNION ALL
        SELECT 1
          FROM pg_catalog.pg_class AS c
          CROSS JOIN LATERAL pg_catalog.aclexplode(c.relacl) AS a
         WHERE c.relnamespace =
               'vec_identidad_sesiones_v1'::regnamespace
           AND a.grantee = consumidor
        UNION ALL
        SELECT 1
          FROM pg_catalog.pg_type AS t
          CROSS JOIN LATERAL pg_catalog.aclexplode(t.typacl) AS a
         WHERE t.typnamespace =
               'vec_identidad_sesiones_v1'::regnamespace
           AND a.grantee = consumidor
    ) THEN
        REVOKE USAGE ON SCHEMA vec_identidad_sesiones_v1
            FROM vec_contexto_actor_v1_propietario;
    END IF;
END
$uso_esquema$;

RESET ROLE;

DO $postvalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
        'vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(text,text)'
    ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada de fachada corporativa RRHH incompleta';
    END IF;
END
$postvalidacion$;

COMMIT;
