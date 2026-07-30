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
    ) <> 'e6b45360b65a0d5e58289a2ca4e63044a650fe0753d00b8beb0b8116fb56888f'
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
         WHERE p.pronamespace =
               'vec_identidad_sesiones_v1'::regnamespace
           AND pg_catalog.has_function_privilege(
               consumidor, p.oid, 'EXECUTE'
           )
        UNION ALL
        SELECT 1
          FROM pg_catalog.pg_class AS c
         WHERE c.relnamespace =
               'vec_identidad_sesiones_v1'::regnamespace
           AND (
               c.relkind = 'S' AND (
                   pg_catalog.has_sequence_privilege(
                       consumidor, c.oid, 'USAGE'
                   ) OR pg_catalog.has_sequence_privilege(
                       consumidor, c.oid, 'SELECT'
                   ) OR pg_catalog.has_sequence_privilege(
                       consumidor, c.oid, 'UPDATE'
                   )
               ) OR c.relkind IN ('r', 'p', 'v', 'm', 'f') AND (
                   pg_catalog.has_table_privilege(
                       consumidor, c.oid, 'SELECT'
                   ) OR pg_catalog.has_table_privilege(
                       consumidor, c.oid, 'INSERT'
                   ) OR pg_catalog.has_table_privilege(
                       consumidor, c.oid, 'UPDATE'
                   ) OR pg_catalog.has_table_privilege(
                       consumidor, c.oid, 'DELETE'
                   ) OR pg_catalog.has_table_privilege(
                       consumidor, c.oid, 'TRUNCATE'
                   ) OR pg_catalog.has_table_privilege(
                       consumidor, c.oid, 'REFERENCES'
                   ) OR pg_catalog.has_table_privilege(
                       consumidor, c.oid, 'TRIGGER'
                   ) OR pg_catalog.has_table_privilege(
                       consumidor, c.oid, 'MAINTAIN'
                   )
               )
           )
        UNION ALL
        SELECT 1
          FROM pg_catalog.pg_attribute AS a
          JOIN pg_catalog.pg_class AS c ON c.oid = a.attrelid
         WHERE c.relnamespace =
               'vec_identidad_sesiones_v1'::regnamespace
           AND a.attnum > 0 AND NOT a.attisdropped
           AND (
               pg_catalog.has_column_privilege(
                   consumidor, c.oid, a.attnum, 'SELECT'
               ) OR pg_catalog.has_column_privilege(
                   consumidor, c.oid, a.attnum, 'INSERT'
               ) OR pg_catalog.has_column_privilege(
                   consumidor, c.oid, a.attnum, 'UPDATE'
               ) OR pg_catalog.has_column_privilege(
                   consumidor, c.oid, a.attnum, 'REFERENCES'
               )
           )
        UNION ALL
        SELECT 1
          FROM pg_catalog.pg_type AS t
         WHERE t.typnamespace =
               'vec_identidad_sesiones_v1'::regnamespace
           AND NOT EXISTS (
               SELECT 1 FROM pg_catalog.pg_type AS e
                WHERE e.oid = t.typelem AND e.typarray = t.oid
           )
           AND pg_catalog.has_type_privilege(
               consumidor, t.oid, 'USAGE'
           )
        UNION ALL
        SELECT 1
          FROM pg_catalog.pg_default_acl AS d
          CROSS JOIN LATERAL
               pg_catalog.aclexplode(d.defaclacl) AS a
         WHERE d.defaclnamespace =
               'vec_identidad_sesiones_v1'::regnamespace
           AND (
               d.defaclrole = consumidor OR a.grantor = consumidor
               OR CASE WHEN a.grantee = 0 THEN true ELSE
                    pg_catalog.pg_has_role(
                        consumidor, a.grantee, 'MEMBER'
                    )
                  END
           )
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
