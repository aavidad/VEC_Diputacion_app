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
    base oid := pg_catalog.to_regprocedure(
        'vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)'
    );
    propietario oid := pg_catalog.to_regrole(
        'vec_identidad_sesiones_v1_propietario'
    );
    consumidor oid := pg_catalog.to_regrole(
        'vec_contexto_actor_v1_propietario'
    );
    selector oid := pg_catalog.to_regrole(
        'vec_contexto_actor_corporativo_rrhh_selector'
    );
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) OR base IS NULL OR propietario IS NULL OR consumidor IS NULL
       OR selector IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'alta de fachada corporativa RRHH rechazada';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS p
         WHERE p.pronamespace =
               'vec_identidad_sesiones_v1'::regnamespace
           AND p.proname =
               'revalidar_contexto_corporativo_rrhh_v1'
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace AS n
          CROSS JOIN LATERAL pg_catalog.aclexplode(n.nspacl) AS a
         WHERE n.oid = 'vec_identidad_sesiones_v1'::regnamespace
           AND (a.grantee = consumidor OR a.grantor = consumidor)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'alta de fachada corporativa RRHH rechazada';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles AS r
         WHERE r.oid = selector
           AND NOT r.rolcanlogin AND NOT r.rolsuper
           AND NOT r.rolcreatedb AND NOT r.rolcreaterole
           AND NOT r.rolinherit AND NOT r.rolreplication
           AND NOT r.rolbypassrls AND r.rolconnlimit = -1
           AND r.rolvaliduntil IS NULL AND r.rolconfig IS NULL
           AND pg_catalog.shobj_description(r.oid, 'pg_authid') =
               'vec_contexto_actor_v1:rol-contexto-corporativo-rrhh-selector:v1'
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_db_role_setting
         WHERE setrole = selector
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS m
          LEFT JOIN pg_catalog.pg_roles AS login ON login.oid = m.member
          LEFT JOIN pg_catalog.pg_roles AS otorgante ON otorgante.oid = m.grantor
         WHERE m.member = selector OR m.grantor = selector
            OR m.roleid = selector AND (
                NOT login.rolcanlogin OR NOT login.rolinherit
                OR login.rolsuper OR login.rolcreatedb OR login.rolcreaterole
                OR login.rolreplication OR login.rolbypassrls
                OR login.rolconfig IS NOT NULL
                OR m.admin_option OR NOT m.inherit_option OR m.set_option
                OR otorgante.rolsuper IS NOT TRUE
            )
    ) OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_auth_members
         WHERE roleid = selector
    ) > 1 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'alta de fachada corporativa RRHH rechazada';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS p
          JOIN pg_catalog.pg_language AS l ON l.oid = p.prolang
         WHERE p.oid = base
           AND l.lanname = 'plpgsql' AND p.proowner = propietario
           AND p.prokind = 'f' AND p.provolatile = 'v'
           AND p.proparallel = 'u' AND p.prosecdef
           AND NOT p.proleakproof AND NOT p.proisstrict AND p.proretset
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
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'alta de fachada corporativa RRHH rechazada';
    END IF;
END
$prevalidacion$;

SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;

CREATE FUNCTION
    vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(
        p_autenticacion_ref text,
        p_sesion_ref text
    )
RETURNS TABLE (
    cuenta_ref text,
    metodo_observado text,
    garantia_observada text,
    identidad_valida_hasta timestamptz
)
LANGUAGE SQL
VOLATILE
CALLED ON NULL INPUT
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog
SET lock_timeout = '1s'
BEGIN ATOMIC
    WITH guarda AS MATERIALIZED (
        SELECT pg_catalog.pg_advisory_xact_lock_shared(
            pg_catalog.hashtextextended(
                'vec_contexto_actor_v1:rol-contexto-corporativo-rrhh-selector:v1',
                0
            )
        )
    ), identidades AS MATERIALIZED (
        SELECT login.oid AS login_oid, selector.oid AS selector_oid,
               propia.oid AS propia_oid, base.oid AS base_oid,
               identidad.oid AS identidad_oid,
               consumidor.oid AS consumidor_oid,
               revalidador.oid AS revalidador_oid
          FROM guarda
          CROSS JOIN pg_catalog.pg_roles AS login
          CROSS JOIN pg_catalog.pg_roles AS selector
          CROSS JOIN pg_catalog.pg_roles AS identidad
          CROSS JOIN pg_catalog.pg_roles AS consumidor
          CROSS JOIN pg_catalog.pg_roles AS revalidador
          CROSS JOIN pg_catalog.pg_proc AS propia
          CROSS JOIN pg_catalog.pg_proc AS base
         WHERE login.rolname = session_user
           AND selector.rolname =
               'vec_contexto_actor_corporativo_rrhh_selector'
           AND identidad.rolname =
               'vec_identidad_sesiones_v1_propietario'
           AND consumidor.rolname =
               'vec_contexto_actor_v1_propietario'
           AND revalidador.rolname =
               'vec_identidad_sesiones_v1_revalidador'
           AND propia.oid = pg_catalog.to_regprocedure(
               'vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(text,text)'
           )
           AND base.oid = pg_catalog.to_regprocedure(
               'vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)'
           )
           AND current_user =
               'vec_identidad_sesiones_v1_propietario'
           AND pg_catalog.current_setting('role') = 'none'
           AND pg_catalog.current_setting('transaction_isolation') =
               'serializable'
           AND pg_catalog.current_setting('transaction_read_only') = 'off'
           AND pg_catalog.pg_is_in_recovery() IS FALSE
           AND login.rolcanlogin AND login.rolinherit
           AND NOT login.rolsuper AND NOT login.rolcreatedb
           AND NOT login.rolcreaterole AND NOT login.rolreplication
           AND NOT login.rolbypassrls AND login.rolconfig IS NULL
           AND (
               login.rolvaliduntil IS NULL
               OR pg_catalog.clock_timestamp() < login.rolvaliduntil
           )
           AND NOT selector.rolcanlogin AND NOT selector.rolsuper
           AND NOT selector.rolcreatedb AND NOT selector.rolcreaterole
           AND NOT selector.rolinherit AND NOT selector.rolreplication
           AND NOT selector.rolbypassrls AND selector.rolconnlimit = -1
           AND selector.rolvaliduntil IS NULL
           AND selector.rolconfig IS NULL
           AND pg_catalog.shobj_description(
                   selector.oid, 'pg_authid'
               ) =
               'vec_contexto_actor_v1:rol-contexto-corporativo-rrhh-selector:v1'
           AND NOT EXISTS (
               SELECT 1
                 FROM pg_catalog.pg_db_role_setting AS ajuste
                WHERE ajuste.setrole IN (login.oid, selector.oid)
           )
           AND (
               SELECT pg_catalog.count(*) = 1
                  AND pg_catalog.bool_and(
                      m.roleid = selector.oid AND m.member = login.oid
                      AND NOT m.admin_option AND m.inherit_option
                      AND NOT m.set_option
                      AND otorgante.rolsuper
                  )
                 FROM pg_catalog.pg_auth_members AS m
                 JOIN pg_catalog.pg_roles AS otorgante
                   ON otorgante.oid = m.grantor
                WHERE m.member IN (login.oid, selector.oid)
                   OR m.roleid IN (login.oid, selector.oid)
                   OR m.grantor IN (login.oid, selector.oid)
           )
           AND NOT EXISTS (
               SELECT 1
                 FROM pg_catalog.pg_namespace AS n
                 CROSS JOIN LATERAL
                      pg_catalog.aclexplode(n.nspacl) AS a
                WHERE n.oid =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND (
                      a.grantee IN (login.oid, selector.oid)
                      OR a.grantor IN (login.oid, selector.oid)
                  )
               UNION ALL
               SELECT 1
                 FROM pg_catalog.pg_proc AS p
                 CROSS JOIN LATERAL
                      pg_catalog.aclexplode(p.proacl) AS a
                WHERE p.pronamespace =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND (
                      a.grantee IN (login.oid, selector.oid)
                      OR a.grantor IN (login.oid, selector.oid)
                  )
               UNION ALL
               SELECT 1
                 FROM pg_catalog.pg_class AS c
                 CROSS JOIN LATERAL
                      pg_catalog.aclexplode(c.relacl) AS a
                WHERE c.relnamespace =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND (
                      a.grantee IN (login.oid, selector.oid)
                      OR a.grantor IN (login.oid, selector.oid)
                  )
               UNION ALL
               SELECT 1
                 FROM pg_catalog.pg_type AS t
                 CROSS JOIN LATERAL
                      pg_catalog.aclexplode(t.typacl) AS a
                WHERE t.typnamespace =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND (
                      a.grantee IN (login.oid, selector.oid)
                      OR a.grantor IN (login.oid, selector.oid)
                  )
               UNION ALL
               SELECT 1
                 FROM pg_catalog.pg_default_acl AS d
                 CROSS JOIN LATERAL
                      pg_catalog.aclexplode(d.defaclacl) AS a
                WHERE d.defaclnamespace =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND (
                      d.defaclrole IN (login.oid, selector.oid)
                      OR a.grantee IN (login.oid, selector.oid)
                      OR a.grantor IN (login.oid, selector.oid)
                  )
               UNION ALL
               SELECT 1
                 FROM pg_catalog.pg_policy AS politica
                 JOIN pg_catalog.pg_class AS c
                   ON c.oid = politica.polrelid
                WHERE c.relnamespace =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND (
                      login.oid = ANY (politica.polroles)
                      OR selector.oid = ANY (politica.polroles)
                  )
           )
           AND NOT EXISTS (
               SELECT 1
                 FROM pg_catalog.pg_namespace AS n
                 CROSS JOIN LATERAL
                      pg_catalog.aclexplode(n.nspacl) AS a
                WHERE n.oid =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND a.grantee = 0
           )
    ), manifiestos AS MATERIALIZED (
        SELECT i.*
          FROM identidades AS i
          JOIN pg_catalog.pg_proc AS propia ON propia.oid = i.propia_oid
          JOIN pg_catalog.pg_language AS lenguaje_propio
            ON lenguaje_propio.oid = propia.prolang
          JOIN pg_catalog.pg_proc AS base ON base.oid = i.base_oid
          JOIN pg_catalog.pg_language AS lenguaje_base
            ON lenguaje_base.oid = base.prolang
         WHERE lenguaje_propio.lanname = 'sql'
           AND propia.proowner = i.identidad_oid
           AND propia.prokind = 'f' AND propia.provolatile = 'v'
           AND propia.proparallel = 'u' AND propia.prosecdef
           AND NOT propia.proleakproof AND NOT propia.proisstrict
           AND propia.proretset AND propia.pronargs = 2
           AND propia.pronargdefaults = 0
           AND propia.proargtypes = '25 25'::oidvector
           AND propia.proconfig = ARRAY[
               'search_path=pg_catalog', 'lock_timeout=1s'
           ]::text[]
           AND propia.prosqlbody IS NOT NULL
           AND (
               SELECT pg_catalog.count(*) = 2
                  AND pg_catalog.count(DISTINCT a.grantee) = 2
                  AND pg_catalog.bool_and(
                      a.grantor = i.identidad_oid
                      AND a.grantee = ANY (ARRAY[
                          i.identidad_oid, i.consumidor_oid
                      ])
                      AND a.privilege_type = 'EXECUTE'
                      AND NOT a.is_grantable
                  )
                 FROM pg_catalog.aclexplode(propia.proacl) AS a
           )
           AND lenguaje_base.lanname = 'plpgsql'
           AND base.proowner = i.identidad_oid
           AND base.prokind = 'f' AND base.provolatile = 'v'
           AND base.proparallel = 'u' AND base.prosecdef
           AND NOT base.proleakproof AND NOT base.proisstrict
           AND base.proretset AND base.pronargs = 2
           AND base.pronargdefaults = 0
           AND base.proargtypes = '25 25'::oidvector
           AND base.proconfig =
               ARRAY['search_path=pg_catalog, pg_temp']::text[]
           AND base.prosqlbody IS NULL
           AND pg_catalog.encode(pg_catalog.sha256(
               pg_catalog.convert_to(base.prosrc, 'UTF8')
           ), 'hex') =
               '277ab207e23d261522c6397578ebba9412f5111d25b985896756fe62677c3d46'
           AND pg_catalog.octet_length(base.prosrc) = 7783
           AND (
               SELECT pg_catalog.count(*) = 2
                  AND pg_catalog.count(DISTINCT a.grantee) = 2
                  AND pg_catalog.bool_and(
                      a.grantor = i.identidad_oid
                      AND a.grantee = ANY (ARRAY[
                          i.identidad_oid, i.revalidador_oid
                      ])
                      AND a.privilege_type = 'EXECUTE'
                      AND NOT a.is_grantable
                  )
                 FROM pg_catalog.aclexplode(base.proacl) AS a
           )
           AND (
               SELECT pg_catalog.count(*) = 1
                 FROM pg_catalog.pg_depend AS d
                WHERE d.classid = 'pg_catalog.pg_proc'::regclass
                  AND d.objid = propia.oid AND d.objsubid = 0
                  AND d.refclassid = 'pg_catalog.pg_proc'::regclass
                  AND d.refobjid = base.oid AND d.refobjsubid = 0
                  AND d.deptype = 'n'
           )
           AND NOT pg_catalog.pg_has_role(
               i.consumidor_oid, i.identidad_oid, 'MEMBER'
           )
           AND NOT pg_catalog.has_function_privilege(
               i.consumidor_oid, base.oid, 'EXECUTE'
           )
           AND NOT EXISTS (
               SELECT 1
                 FROM pg_catalog.pg_class AS c
                WHERE c.relnamespace =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND c.relkind IN ('r', 'p', 'v', 'm', 'f')
                  AND (
                      pg_catalog.has_table_privilege(
                          i.consumidor_oid, c.oid, 'SELECT'
                      )
                      OR pg_catalog.has_table_privilege(
                          i.consumidor_oid, c.oid, 'INSERT'
                      )
                      OR pg_catalog.has_table_privilege(
                          i.consumidor_oid, c.oid, 'UPDATE'
                      )
                      OR pg_catalog.has_table_privilege(
                          i.consumidor_oid, c.oid, 'DELETE'
                      )
                      OR pg_catalog.has_table_privilege(
                          i.consumidor_oid, c.oid, 'TRUNCATE'
                      )
                      OR pg_catalog.has_table_privilege(
                          i.consumidor_oid, c.oid, 'REFERENCES'
                      )
                      OR pg_catalog.has_table_privilege(
                          i.consumidor_oid, c.oid, 'TRIGGER'
                      )
                      OR pg_catalog.has_table_privilege(
                          i.consumidor_oid, c.oid, 'MAINTAIN'
                      )
                  )
           )
           AND NOT EXISTS (
               SELECT 1
                 FROM pg_catalog.pg_policy AS politica
                 JOIN pg_catalog.pg_class AS c
                   ON c.oid = politica.polrelid
                WHERE c.relnamespace =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND i.consumidor_oid = ANY (politica.polroles)
           )
    ), resultado AS MATERIALIZED (
        SELECT r.*, pg_catalog.count(*) OVER () AS total
          FROM manifiestos
          CROSS JOIN LATERAL
               vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
                   p_autenticacion_ref, p_sesion_ref
               ) AS r
    ), ventana AS MATERIALIZED (
        SELECT r.*, CASE
                   WHEN r.autenticacion_verificada_en IS NOT NULL
                    AND pg_catalog.isfinite(
                        r.autenticacion_verificada_en
                    )
                    AND extract(
                        year FROM (
                            r.autenticacion_verificada_en
                            AT TIME ZONE 'UTC'
                        )
                    ) BETWEEN 1 AND 9999
                    AND r.autenticacion_verificada_en <=
                        timestamptz '9999-12-31 23:44:59.999999+00'
                   THEN r.autenticacion_verificada_en +
                        interval '15 minutes'
                   ELSE NULL
               END AS autenticacion_valida_hasta
          FROM resultado AS r
    )
    SELECT r.cuenta_ref, r.metodo_observado, r.garantia_observada,
           LEAST(
               r.sesion_valida_hasta,
               r.autenticacion_valida_hasta
           )
      FROM ventana AS r
     WHERE r.total = 1
       AND r.autenticacion_ref = p_autenticacion_ref
       AND r.sesion_ref = p_sesion_ref
       AND r.cuenta_ref IS NOT NULL
       AND r.metodo_observado IS NOT NULL
       AND r.garantia_observada = 'alto'
       AND r.superficie = 'interna_corporativa'
       AND NOT r.cuenta_privilegiada
       AND r.cuenta_ref = r.cuenta_ordinaria_ref
       AND r.autenticacion_verificada_en IS NOT NULL
       AND r.sesion_valida_hasta IS NOT NULL
       AND r.autenticacion_valida_hasta IS NOT NULL
       AND pg_catalog.isfinite(r.autenticacion_verificada_en)
       AND pg_catalog.isfinite(r.sesion_valida_hasta)
       AND extract(
           year FROM (r.sesion_valida_hasta AT TIME ZONE 'UTC')
       ) BETWEEN 1 AND 9999
       AND pg_catalog.clock_timestamp() < LEAST(
           r.sesion_valida_hasta,
           r.autenticacion_valida_hasta
       );
END;

REVOKE ALL ON FUNCTION
    vec_identidad_sesiones_v1.
    revalidar_contexto_corporativo_rrhh_v1(text, text)
    FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_identidad_sesiones_v1
    TO vec_contexto_actor_v1_propietario;
GRANT EXECUTE ON FUNCTION
    vec_identidad_sesiones_v1.
    revalidar_contexto_corporativo_rrhh_v1(text, text)
    TO vec_contexto_actor_v1_propietario;

RESET ROLE;

DO $postvalidacion$
DECLARE
    propia oid := pg_catalog.to_regprocedure(
        'vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(text,text)'
    );
    base oid := pg_catalog.to_regprocedure(
        'vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)'
    );
    propietario oid :=
        'vec_identidad_sesiones_v1_propietario'::regrole;
    consumidor oid := 'vec_contexto_actor_v1_propietario'::regrole;
BEGIN
    IF propia IS NULL OR (
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
              AND pg_catalog.regexp_count(
                  pg_catalog.pg_get_functiondef(p.oid),
                  'CROSS JOIN LATERAL[[:space:]]+vec_identidad_sesiones_v1[.]revalidar_autenticacion_actor_v1'
              ) = 1
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
            MESSAGE = 'alta de fachada corporativa RRHH incompleta';
    END IF;
END
$postvalidacion$;

COMMIT;
