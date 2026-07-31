BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $precondiciones$
BEGIN
    IF pg_catalog.current_setting(
           'vec.confirmar_retirada_acreditacion_contexto_actor_v2', true
       ) IS DISTINCT FROM 'RETIRAR_ACREDITACION_CONTEXTO_ACTOR_V2' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada ContextoActor V2 requiere confirmacion explicita';
    ELSIF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'retirada de acreditacion ContextoActor V2 requiere superusuario';
    END IF;
END
$precondiciones$;
-- El orden es común a la retirada base y a las ampliaciones corporativas.
SELECT pg_catalog.pg_advisory_xact_lock_shared(
    pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:migracion:base:v1', 0
    )
);
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:migracion:acreditacion_uso:v2', 0
    )
);
-- Las doce relaciones de 000001 y el control de 000002 permanecen inmóviles
-- desde antes del inventario hasta el COMMIT. La guarda advisory se tomó antes
-- que las tablas, en el mismo orden que la retirada base.
LOCK TABLE
    vec_contexto_actor_v1.control_generacion_punteros_actuales_v2,
    vec_contexto_actor_v1.procedencias,
    vec_contexto_actor_v1.proyeccion_cuenta_versiones,
    vec_contexto_actor_v1.proyeccion_cuenta_actual,
    vec_contexto_actor_v1.persona_versiones,
    vec_contexto_actor_v1.persona_actual,
    vec_contexto_actor_v1.perfil_versiones,
    vec_contexto_actor_v1.perfil_actual,
    vec_contexto_actor_v1.vinculo_contexto_versiones,
    vec_contexto_actor_v1.vinculo_contexto_actual,
    vec_contexto_actor_v1.vinculo_referencia_versiones,
    vec_contexto_actor_v1.vinculo_referencia_actual,
    vec_contexto_actor_v1.registros_contexto
IN ACCESS EXCLUSIVE MODE;
-- SHARE conserva la fotografía catalogal desde el inventario hasta el COMMIT:
-- inmoviliza roles, ACL, comentarios, dependencias y metadatos canonizados.
LOCK TABLE
    pg_catalog.pg_authid, pg_catalog.pg_auth_members,
    pg_catalog.pg_db_role_setting, pg_catalog.pg_class,
    pg_catalog.pg_attribute, pg_catalog.pg_index, pg_catalog.pg_namespace, pg_catalog.pg_language, pg_catalog.pg_collation,
    pg_catalog.pg_proc, pg_catalog.pg_type, pg_catalog.pg_default_acl,
    pg_catalog.pg_description, pg_catalog.pg_seclabel,
    pg_catalog.pg_init_privs, pg_catalog.pg_depend, pg_catalog.pg_shdepend,
    pg_catalog.pg_publication, pg_catalog.pg_publication_namespace,
    pg_catalog.pg_publication_rel, pg_catalog.pg_subscription_rel,
    pg_catalog.pg_statistic_ext
IN SHARE MODE;
DO $inventario$
DECLARE
    propietario constant oid := 'vec_contexto_actor_v1_propietario'::regrole;
    roles constant oid[] := ARRAY[
        propietario, 'vec_contexto_actor_v1_migrador'::regrole,
        'vec_contexto_actor_v1_runtime'::regrole
    ]::oid[];
    esquema constant oid := 'vec_contexto_actor_v1'::regnamespace;
    control constant oid := 'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass;
    serializar constant oid := pg_catalog.to_regprocedure(
        'vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2()'
    );
    avanzar constant oid := pg_catalog.to_regprocedure(
        'vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2()'
    );
    acreditar constant oid := pg_catalog.to_regprocedure(
        'vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)'
    );
    punteros constant oid[] := ARRAY[
        'vec_contexto_actor_v1.proyeccion_cuenta_actual'::regclass,
        'vec_contexto_actor_v1.persona_actual'::regclass, 'vec_contexto_actor_v1.perfil_actual'::regclass,
        'vec_contexto_actor_v1.vinculo_contexto_actual'::regclass,
        'vec_contexto_actor_v1.vinculo_referencia_actual'::regclass
    ]::oid[];
    funciones oid[];
    toast oid;
    observado text;
    esperado constant text := 'cf2895f6f30f25bca2310161ad900c241e382ebe3bb11f2dd7ff517700422bed';
BEGIN
    funciones := ARRAY[acreditar, avanzar, serializar];
    SELECT reltoastrelid INTO toast FROM pg_catalog.pg_class WHERE oid = control;
    IF pg_catalog.current_setting('server_version_num')::integer < 180004
       OR pg_catalog.current_setting('server_version_num')::integer >= 190000
       OR serializar IS NULL OR avanzar IS NULL OR acreditar IS NULL
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_proc
            WHERE pronamespace = esquema
              AND proname IN (
                  'serializar_mutacion_punteros_actuales_v2',
                  'avanzar_generacion_punteros_actuales_v2',
                  'acreditar_uso_registro_contexto_actor_v2'
              )
           ) <> 3
       OR (
           SELECT pg_catalog.count(*) <> 3
             FROM pg_catalog.pg_authid AS r
            WHERE r.oid = ANY(roles)
              AND NOT r.rolsuper AND NOT r.rolinherit
              AND NOT r.rolcreaterole AND NOT r.rolcreatedb
              AND NOT r.rolcanlogin AND NOT r.rolreplication
              AND NOT r.rolbypassrls AND r.rolconnlimit = -1
              AND r.rolvaliduntil IS NULL AND r.rolpassword IS NULL
       )
       OR (
           SELECT pg_catalog.count(*) <> 1
                  OR NOT pg_catalog.bool_and(
                      m.roleid = propietario AND m.member =
                          'vec_contexto_actor_v1_migrador'::regrole
                      AND NOT m.admin_option AND NOT m.inherit_option
                      AND m.set_option
                  )
             FROM pg_catalog.pg_auth_members AS m
            WHERE m.member = ANY(roles) OR m.grantor = ANY(roles)
               OR m.roleid IN (propietario, 'vec_contexto_actor_v1_migrador'::regrole)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members AS m
           JOIN pg_catalog.pg_authid AS r ON r.oid = m.member
          WHERE m.roleid = 'vec_contexto_actor_v1_runtime'::regrole
            AND (m.admin_option OR NOT m.inherit_option OR m.set_option
                 OR NOT r.rolcanlogin OR r.rolsuper OR NOT r.rolinherit
                 OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication OR r.rolbypassrls
                 OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members AS x
                      WHERE x.member = m.member) <> 1
                 OR EXISTS (SELECT 1 FROM pg_catalog.pg_db_role_setting AS s WHERE s.setrole = m.member))
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_db_role_setting AS s
            WHERE s.setrole = ANY(roles) OR (s.setrole = 0 AND s.setdatabase IN
                  (0, (SELECT oid FROM pg_catalog.pg_database WHERE datname = pg_catalog.current_database())))
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada ContextoActor V2 rechazada: funciones o roles incompletos';
    END IF;
    -- Descubrimiento por relación, no una cuenta fija de quince nombres.
    IF EXISTS (
        WITH candidatos AS (
            SELECT t.*
              FROM pg_catalog.pg_trigger AS t
             WHERE NOT t.tgisinternal
               AND (
                   t.tgname IN (
                       'puntero_actual_no_truncable_v2',
                       'serializar_mutacion_punteros_actuales_v2',
                       'avanzar_generacion_punteros_actuales_v2'
                   )
                   OR t.tgfoid = ANY(ARRAY[serializar, avanzar]::oid[])
               )
               AND t.tgrelid IN (
                   SELECT oid FROM pg_catalog.pg_class
                    WHERE relnamespace = esquema
               )
        ), grupos AS (
            SELECT tgrelid, pg_catalog.count(*) AS numero,
                   pg_catalog.bool_and(
                       tgenabled = 'O' AND NOT tgisinternal
                       AND tgconstraint = 0 AND tgconstrrelid = 0
                       AND tgconstrindid = 0 AND tgparentid = 0
                       AND NOT tgdeferrable AND NOT tginitdeferred
                       AND tgnargs = 0 AND tgargs = '\x'::bytea
                       AND tgattr = ''::int2vector AND tgqual IS NULL
                       AND tgoldtable IS NULL AND tgnewtable IS NULL
                   ) AS forma,
                   pg_catalog.count(*) FILTER (
                       WHERE tgname = 'puntero_actual_no_truncable_v2'
                         AND tgfoid =
                             'vec_contexto_actor_v1.rechazar_truncado()'::regprocedure
                         AND tgtype = 34
                   ) AS no_truncable,
                   pg_catalog.count(*) FILTER (
                       WHERE tgname =
                             'serializar_mutacion_punteros_actuales_v2'
                         AND tgfoid = serializar AND tgtype = 30
                   ) AS serializa,
                   pg_catalog.count(*) FILTER (
                       WHERE tgname =
                             'avanzar_generacion_punteros_actuales_v2'
                         AND tgfoid = avanzar AND tgtype = 28
                   ) AS avanza
              FROM candidatos
             GROUP BY tgrelid
        )
        SELECT 1
          FROM (
              SELECT pg_catalog.count(*) AS numero,
                     pg_catalog.array_agg(
                         tgrelid ORDER BY tgrelid::regclass::text COLLATE "C"
                     ) AS relaciones,
                     pg_catalog.bool_and(
                         grupos.numero = 3 AND forma
                         AND no_truncable = 1
                         AND serializa = 1 AND avanza = 1
                     ) AS exactos
                FROM grupos
          ) AS inventario
         WHERE numero <> 5
            OR relaciones IS DISTINCT FROM (
                SELECT pg_catalog.array_agg(
                           oid ORDER BY oid::regclass::text COLLATE "C"
                       )
                  FROM pg_catalog.unnest(punteros) AS oid
            )
            OR exactos IS NOT TRUE
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_trigger AS t
         WHERE t.tgrelid = ANY(punteros) AND NOT t.tgisinternal
         GROUP BY t.oid
        HAVING pg_catalog.count(*) <> 1
            OR (
                SELECT pg_catalog.count(*)
                  FROM pg_catalog.pg_depend AS d
                 WHERE d.classid = 'pg_catalog.pg_trigger'::regclass
                   AND d.objid = t.oid
                   AND d.objsubid = 0
                   AND (
                       (d.refclassid = 'pg_catalog.pg_class'::regclass
                        AND d.refobjid = t.tgrelid
                        AND d.refobjsubid = 0 AND d.deptype = 'a')
                       OR
                       (d.refclassid = 'pg_catalog.pg_proc'::regclass
                        AND d.refobjid = t.tgfoid
                        AND d.refobjsubid = 0 AND d.deptype = 'n')
                   )
            ) <> 2
            OR (
                SELECT pg_catalog.count(*)
                  FROM pg_catalog.pg_depend AS d
                 WHERE d.classid = 'pg_catalog.pg_trigger'::regclass
                   AND d.objid = t.oid AND d.objsubid = 0
            ) <> 2
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada ContextoActor V2 rechazada: trios de punteros derivados';
    END IF;
    -- Manifiesto simbólico del alcance estructural gestionado de 000001+000002.
    -- No incluye OID ni datos de negocio; las dependencias y superficies
    -- implícitas que no tienen forma canónica propia se rechazan después.
    WITH elementos AS (
        SELECT pg_catalog.format(
            'n|%s|%s|%s', n.nspname, n.nspowner::regrole::text,
            coalesce((
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        a.grantor::regrole::text,
                        CASE WHEN a.grantee = 0 THEN 'PUBLIC'
                             ELSE a.grantee::regrole::text END,
                        a.privilege_type, a.is_grantable
                    ) ORDER BY a.grantor, a.grantee,
                               a.privilege_type, a.is_grantable
                )::text
                  FROM pg_catalog.aclexplode(coalesce(
                      n.nspacl, pg_catalog.acldefault('n', n.nspowner)
                  )) AS a
            ), '[]')
        ) AS elemento
          FROM pg_catalog.pg_namespace AS n WHERE n.oid = esquema
        UNION ALL
        SELECT pg_catalog.format(
            'c|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
            c.relname, c.relkind, c.relpersistence,
            c.relowner::regrole::text, c.relrowsecurity,
            c.relforcerowsecurity, c.relhasrules, c.relhastriggers,
            c.relhassubclass, c.relispartition, c.relreplident,
            coalesce(am.amname, ''), coalesce(e.spcname, ''),
            coalesce(c.reloptions::text, ''),
            coalesce(pg_catalog.obj_description(c.oid, 'pg_class'), ''),
            coalesce((
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        a.grantor::regrole::text,
                        CASE WHEN a.grantee = 0 THEN 'PUBLIC'
                             ELSE a.grantee::regrole::text END,
                        a.privilege_type, a.is_grantable
                    ) ORDER BY a.grantor, a.grantee,
                               a.privilege_type, a.is_grantable
                )::text FROM pg_catalog.aclexplode(coalesce(
                    c.relacl, pg_catalog.acldefault(
                        CASE WHEN c.relkind = 'S' THEN 'S'::"char"
                             ELSE 'r'::"char" END, c.relowner
                    )
                )) AS a
            ), '[]')
        ) FROM pg_catalog.pg_class AS c
          LEFT JOIN pg_catalog.pg_am AS am ON am.oid = c.relam
          LEFT JOIN pg_catalog.pg_tablespace AS e ON e.oid = c.reltablespace
         WHERE c.relnamespace = esquema
        UNION ALL
        SELECT pg_catalog.format(
            'a|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
            c.relname, a.attnum, a.attname,
            pg_catalog.format_type(a.atttypid, a.atttypmod),
            a.attnotnull, a.attidentity, a.attgenerated, a.attisdropped,
            a.attstorage, a.attcompression,
            a.attstattarget, a.atthasmissing,
            coalesce(a.attmissingval::text, ''),
            coalesce(a.attoptions::text, ''),
            coalesce(a.attfdwoptions::text, ''),
            coalesce(a.attcollation::regcollation::text, ''),
            coalesce(pg_catalog.pg_get_expr(d.adbin, d.adrelid, false), ''),
            coalesce(pg_catalog.col_description(a.attrelid, a.attnum), ''),
            coalesce((
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        x.grantor::regrole::text,
                        CASE WHEN x.grantee = 0 THEN 'PUBLIC'
                             ELSE x.grantee::regrole::text END,
                        x.privilege_type, x.is_grantable
                    ) ORDER BY x.grantor, x.grantee,
                               x.privilege_type, x.is_grantable
                )::text FROM pg_catalog.aclexplode(a.attacl) AS x
            ), '[]')
        ) FROM pg_catalog.pg_attribute AS a
          JOIN pg_catalog.pg_class AS c ON c.oid = a.attrelid
          LEFT JOIN pg_catalog.pg_attrdef AS d
            ON (d.adrelid, d.adnum) = (a.attrelid, a.attnum)
         WHERE c.relnamespace = esquema AND a.attnum > 0
        UNION ALL
        SELECT pg_catalog.format(
            'i|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
            i.indrelid::regclass::text, ci.relname, i.indisunique,
            i.indisprimary, i.indisexclusion, i.indimmediate,
            i.indisvalid, i.indisready, i.indisclustered, i.indislive,
            i.indisreplident, i.indcheckxmin, i.indnullsnotdistinct,
            pg_catalog.pg_get_indexdef(i.indexrelid, 0, false)
        ) FROM pg_catalog.pg_index AS i
          JOIN pg_catalog.pg_class AS ci ON ci.oid = i.indexrelid
         WHERE ci.relnamespace = esquema
        UNION ALL
        SELECT pg_catalog.format(
            'k|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
            coalesce(c.conrelid::regclass::text, ''), c.conname, c.contype,
            c.condeferrable, c.condeferred, c.convalidated, c.connoinherit,
            c.conislocal, c.coninhcount,
            pg_catalog.pg_get_constraintdef(c.oid, false),
            coalesce(pg_catalog.obj_description(c.oid, 'pg_constraint'), '')
        ) FROM pg_catalog.pg_constraint AS c WHERE c.connamespace = esquema
        UNION ALL
        SELECT pg_catalog.format(
            'g|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
            t.tgrelid::regclass::text,
            CASE WHEN t.tgisinternal THEN coalesce(c.conname, '<interno>')
                 ELSE t.tgname END,
            t.tgtype, t.tgenabled, t.tgisinternal,
            t.tgfoid::regprocedure::text, t.tgdeferrable,
            t.tginitdeferred, t.tgnargs, pg_catalog.encode(t.tgargs, 'hex'),
            t.tgattr::text,
            coalesce(pg_catalog.pg_get_expr(t.tgqual, t.tgrelid, false), ''),
            coalesce(t.tgoldtable, ''), coalesce(t.tgnewtable, ''),
            t.tgparentid = 0, t.tgconstrrelid = 0, t.tgconstrindid = 0,
            coalesce(pg_catalog.obj_description(t.oid, 'pg_trigger'), '')
        ) FROM pg_catalog.pg_trigger AS t
          JOIN pg_catalog.pg_class AS r ON r.oid = t.tgrelid
          LEFT JOIN pg_catalog.pg_constraint AS c ON c.oid = t.tgconstraint
         WHERE r.relnamespace = esquema
        UNION ALL
        SELECT pg_catalog.format(
            'f|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
            p.proname, pg_catalog.pg_get_function_identity_arguments(p.oid),
            pg_catalog.pg_get_function_result(p.oid), l.lanname,
            p.proowner::regrole::text, p.prokind, p.provolatile, p.proparallel,
            p.prosecdef, p.proleakproof, p.proisstrict, p.proretset,
            p.pronargs, p.pronargdefaults,
            coalesce((
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.format_type(a.tipo, NULL)
                    ORDER BY a.posicion
                )::text
                  FROM pg_catalog.unnest(p.proargtypes::oid[])
                       WITH ORDINALITY AS a(tipo, posicion)
            ), '[]'),
            coalesce((
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.format_type(a.tipo, NULL)
                    ORDER BY a.posicion
                )::text
                  FROM pg_catalog.unnest(p.proallargtypes)
                       WITH ORDINALITY AS a(tipo, posicion)
            ), '[]'),
            coalesce(p.proargmodes::text, ''),
            coalesce(p.proargnames::text, ''),
            coalesce(p.proconfig::text, ''), p.procost, p.prorows,
            coalesce(pg_catalog.obj_description(p.oid, 'pg_proc'), ''),
            coalesce((
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        a.grantor::regrole::text,
                        CASE WHEN a.grantee = 0 THEN 'PUBLIC'
                             ELSE a.grantee::regrole::text END,
                        a.privilege_type, a.is_grantable
                    ) ORDER BY a.grantor, a.grantee,
                               a.privilege_type, a.is_grantable
                )::text FROM pg_catalog.aclexplode(coalesce(
                    p.proacl, pg_catalog.acldefault('f', p.proowner)
                )) AS a
            ), '[]'),
            pg_catalog.pg_get_functiondef(p.oid)
        ) FROM pg_catalog.pg_proc AS p
          JOIN pg_catalog.pg_language AS l ON l.oid = p.prolang
         WHERE p.pronamespace = esquema
        UNION ALL
        SELECT pg_catalog.format(
            't|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
            t.typname, t.typtype, t.typowner::regrole::text,
            coalesce(t.typelem::regtype::text, ''),
            coalesce(t.typarray::regtype::text, ''),
            coalesce(t.typrelid::regclass::text, ''),
            coalesce(t.typbasetype::regtype::text, ''),
            t.typnotnull, t.typcategory, t.typispreferred, t.typdelim,
            t.typalign, t.typstorage, t.typbyval,
            coalesce(pg_catalog.obj_description(t.oid, 'pg_type'), ''),
            coalesce((
                SELECT pg_catalog.jsonb_agg(
                    pg_catalog.jsonb_build_array(
                        a.grantor::regrole::text,
                        CASE WHEN a.grantee = 0 THEN 'PUBLIC'
                             ELSE a.grantee::regrole::text END,
                        a.privilege_type, a.is_grantable
                    ) ORDER BY a.grantor, a.grantee,
                               a.privilege_type, a.is_grantable
                )::text FROM pg_catalog.aclexplode(coalesce(
                    t.typacl, pg_catalog.acldefault('T', t.typowner)
                )) AS a
            ), '[]')
        ) FROM pg_catalog.pg_type AS t WHERE t.typnamespace = esquema
        UNION ALL
        SELECT pg_catalog.format(
            'p|%s|%s|%s|%s|%s|%s|%s',
            p.polrelid::regclass::text, p.polname, p.polcmd,
            p.polpermissive, coalesce((
                SELECT pg_catalog.jsonb_agg(
                    CASE WHEN r.rol = 0 THEN 'PUBLIC'
                         ELSE r.rol::regrole::text END
                    ORDER BY r.posicion
                )::text
                  FROM pg_catalog.unnest(p.polroles)
                       WITH ORDINALITY AS r(rol, posicion)
            ), '[]'),
            coalesce(pg_catalog.pg_get_expr(p.polqual, p.polrelid, false), ''),
            coalesce(
                pg_catalog.pg_get_expr(p.polwithcheck, p.polrelid, false), ''
            )
        ) FROM pg_catalog.pg_policy AS p
          JOIN pg_catalog.pg_class AS c ON c.oid = p.polrelid
         WHERE c.relnamespace = esquema
        UNION ALL
        SELECT pg_catalog.format(
            'r|%s|%s|%s', r.ev_class::regclass::text,
            r.rulename, pg_catalog.pg_get_ruledef(r.oid, false)
        ) FROM pg_catalog.pg_rewrite AS r
          JOIN pg_catalog.pg_class AS c ON c.oid = r.ev_class
         WHERE c.relnamespace = esquema
        UNION ALL
        SELECT pg_catalog.format(
            'd|%s|%s|%s|%s', d.defaclobjtype,
            d.defaclrole::regrole::text,
            coalesce(d.defaclnamespace::regnamespace::text, ''),
            d.defaclacl::text
        ) FROM pg_catalog.pg_default_acl AS d
         WHERE d.defaclrole = propietario
        UNION ALL
        SELECT pg_catalog.format(
            's|%s|%s|%s|%s|%s',
            identidad.type, identidad.object_names::text,
            identidad.object_args::text, s.provider, s.label
        ) FROM pg_catalog.pg_seclabel AS s
          CROSS JOIN LATERAL pg_catalog.pg_identify_object_as_address(
              s.classoid, s.objoid, s.objsubid
          ) AS identidad
         WHERE (s.classoid = 'pg_catalog.pg_namespace'::regclass
                AND s.objoid = esquema)
            OR (s.classoid = 'pg_catalog.pg_class'::regclass
                AND s.objoid IN (
                    SELECT oid FROM pg_catalog.pg_class
                     WHERE relnamespace = esquema
                ))
            OR (s.classoid = 'pg_catalog.pg_proc'::regclass
                AND s.objoid IN (
                    SELECT oid FROM pg_catalog.pg_proc
                     WHERE pronamespace = esquema
                ))
            OR (s.classoid = 'pg_catalog.pg_type'::regclass
                AND s.objoid IN (
                    SELECT oid FROM pg_catalog.pg_type
                     WHERE typnamespace = esquema
                ))
        UNION ALL
        SELECT pg_catalog.format(
            'x|%s|%s|%s|%s|%s',
            identidad.type, identidad.object_names::text,
            identidad.object_args::text, i.privtype, i.initprivs::text
        ) FROM pg_catalog.pg_init_privs AS i
          CROSS JOIN LATERAL pg_catalog.pg_identify_object_as_address(
              i.classoid, i.objoid, i.objsubid
          ) AS identidad
         WHERE (i.classoid = 'pg_catalog.pg_class'::regclass
                AND i.objoid IN (
                    SELECT oid FROM pg_catalog.pg_class
                     WHERE relnamespace = esquema
                ))
            OR (i.classoid = 'pg_catalog.pg_proc'::regclass
                AND i.objoid IN (
                    SELECT oid FROM pg_catalog.pg_proc
                     WHERE pronamespace = esquema
                ))
            OR (i.classoid = 'pg_catalog.pg_type'::regclass
                AND i.objoid IN (
                    SELECT oid FROM pg_catalog.pg_type
                     WHERE typnamespace = esquema
                ))
    )
    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               pg_catalog.string_agg(elemento, E'\n' ORDER BY elemento),
               'UTF8'
           )), 'hex')
      INTO observado
      FROM elementos;
    IF observado IS DISTINCT FROM esperado
       OR (
           SELECT pg_catalog.count(*) = 1
                  AND pg_catalog.bool_and(
                      control_id AND generacion >= 0
                      AND pg_catalog.scale(generacion) = 0
                      AND vec_contexto_actor_v1.instante_valido(actualizada_en)
                  )
             FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
       ) IS NOT TRUE
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_proc AS p
            WHERE p.oid <> ALL(funciones)
              AND (
                  p.prosrc LIKE '%control_generacion_punteros_actuales_v2%'
                  OR p.prosrc LIKE '%serializar_mutacion_punteros_actuales_v2%'
                  OR p.prosrc LIKE '%avanzar_generacion_punteros_actuales_v2%'
                  OR p.prosrc LIKE '%acreditar_uso_registro_contexto_actor_v2%'
              )
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_publication AS p
            WHERE p.puballtables
               OR EXISTS (
                   SELECT 1 FROM pg_catalog.pg_publication_namespace AS pn
                    WHERE pn.pnpubid = p.oid AND pn.pnnspid = esquema
               )
               OR EXISTS (
                   SELECT 1 FROM pg_catalog.pg_publication_rel AS pr
                    WHERE pr.prpubid = p.oid
                      AND pr.prrelid IN (
                          SELECT oid FROM pg_catalog.pg_class
                           WHERE relnamespace = esquema
                      )
               )
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_subscription_rel AS s
            WHERE s.srrelid IN (
                SELECT oid FROM pg_catalog.pg_class WHERE relnamespace = esquema
            )
       )
       OR EXISTS (
           WITH RECURSIVE raices(classid, objid, objsubid) AS (
               SELECT 'pg_catalog.pg_class'::regclass, control, 0
               UNION ALL
               SELECT 'pg_catalog.pg_proc'::regclass, f.oid, 0
                 FROM pg_catalog.unnest(funciones) AS f(oid)
               UNION ALL
               SELECT 'pg_catalog.pg_trigger'::regclass, t.oid, 0
                 FROM pg_catalog.pg_trigger AS t
                WHERE t.tgrelid = ANY(punteros) AND NOT t.tgisinternal
                  AND t.tgname IN (
                      'puntero_actual_no_truncable_v2',
                      'serializar_mutacion_punteros_actuales_v2',
                      'avanzar_generacion_punteros_actuales_v2'
                  )
           ), retirables(classid, objid, objsubid) AS (
               SELECT * FROM raices
               UNION
               SELECT d.classid, d.objid, d.objsubid
                 FROM pg_catalog.pg_depend AS d
                   JOIN retirables AS r
                     ON d.refclassid = r.classid
                    AND d.refobjid = r.objid
                    AND (d.refobjsubid = r.objsubid OR r.objsubid = 0)
                WHERE d.deptype IN ('a', 'i', 'P', 'S')
           )
           SELECT 1 FROM retirables AS r
            WHERE (
                   SELECT pg_catalog.jsonb_object_agg(classid::regclass::text, n)
                     FROM (SELECT classid, pg_catalog.count(*) AS n
                             FROM retirables GROUP BY classid) AS clases
               ) <> '{"pg_attrdef":1,"pg_class":4,"pg_constraint":7,"pg_proc":3,"pg_trigger":15,"pg_type":2}'::jsonb
               OR EXISTS (
                   SELECT 1 FROM pg_catalog.pg_depend AS d
                    WHERE (d.classid, d.objid) = (r.classid, r.objid)
                      AND (d.objsubid = r.objsubid OR r.objsubid = 0)
                      AND d.refclassid =
                          'pg_catalog.pg_extension'::regclass
                      AND d.deptype IN ('e', 'x')
               )
               OR EXISTS (
                   SELECT 1 FROM pg_catalog.pg_shdepend AS d
                    WHERE d.dbid = (
                              SELECT oid FROM pg_catalog.pg_database
                               WHERE datname = pg_catalog.current_database()
                          )
                      AND (d.classid, d.objid) = (r.classid, r.objid)
                      AND (d.objsubid = r.objsubid OR r.objsubid = 0)
                      AND d.deptype <> 'o'
               )
               OR EXISTS (
                   SELECT 1 FROM pg_catalog.pg_statistic_ext AS s
                    WHERE s.stxnamespace = esquema
                       OR (r.classid = 'pg_catalog.pg_class'::regclass
                           AND s.stxrelid = r.objid)
               )
               OR (
                   r.classid = 'pg_catalog.pg_class'::regclass
                   AND EXISTS (
                       SELECT 1 FROM pg_catalog.pg_class AS c
                         LEFT JOIN pg_catalog.pg_am AS am ON am.oid = c.relam
                        WHERE c.oid = r.objid AND c.relnamespace <> esquema
                          AND (
                              c.relnamespace <> 'pg_toast'::regnamespace
                              OR c.relowner <> propietario
                              OR c.relpersistence <> 'p' OR c.reltablespace <> 0
                              OR c.relacl IS NOT NULL OR c.reloptions IS NOT NULL
                              OR c.relrowsecurity OR c.relforcerowsecurity
                              OR c.relhasrules OR c.relhastriggers
                              OR c.relhassubclass OR c.relispartition
                              OR c.relreplident <> 'n'
                              OR c.relname IS DISTINCT FROM CASE WHEN c.oid = toast
                                  THEN pg_catalog.format('pg_toast_%s', control)
                                  ELSE pg_catalog.format('pg_toast_%s_index', control) END
                              OR (
                                  c.oid = toast
                                  AND (c.relkind <> 't'
                                       OR am.amname IS DISTINCT FROM 'heap')
                              )
                              OR (
                                  c.oid <> toast
                                  AND (
                                      c.relkind <> 'i'
                                      OR am.amname IS DISTINCT FROM 'btree'
                                      OR NOT EXISTS (
                                          SELECT 1
                                            FROM pg_catalog.pg_index AS i
                                           WHERE i.indexrelid = c.oid
                                             AND i.indrelid = toast
                                             AND i.indisunique
                                             AND i.indisprimary
                                             AND NOT i.indisexclusion
                                             AND i.indimmediate
                                             AND i.indisvalid
                                             AND i.indisready
                                             AND i.indislive
                                             AND NOT i.indisclustered
                                             AND NOT i.indisreplident
                                             AND NOT i.indcheckxmin
                                             AND NOT i.indnullsnotdistinct
                                             AND pg_catalog.pg_get_indexdef(c.oid, 0, false) =
                                                 pg_catalog.format(
                                                 'CREATE UNIQUE INDEX %I ON pg_toast.%I USING btree (chunk_id, chunk_seq)',
                                                 c.relname,
                                                 pg_catalog.format('pg_toast_%s', control))
                                      )
                                  )
                              )
                          )
                   )
               )
               OR (
                   r.classid = 'pg_catalog.pg_class'::regclass
                   AND EXISTS (
                       SELECT 1 FROM pg_catalog.pg_class AS c
                         JOIN pg_catalog.pg_attribute AS a
                           ON a.attrelid = c.oid AND a.attnum > 0
                        WHERE c.oid = r.objid AND c.relnamespace <> esquema
                          AND (
                              coalesce(a.attstattarget, -1) <> -1
                              OR a.attacl IS NOT NULL
                              OR a.attoptions IS NOT NULL
                              OR a.attfdwoptions IS NOT NULL
                              OR a.attstorage <> 'p' OR a.attcompression <> ''
                          )
                   )
               )
               OR EXISTS (
                   SELECT 1 FROM (
                       SELECT classoid, objoid, objsubid
                         FROM pg_catalog.pg_description
                       UNION ALL
                       SELECT classoid, objoid, objsubid
                         FROM pg_catalog.pg_seclabel
                       UNION ALL
                       SELECT classoid, objoid, objsubid
                         FROM pg_catalog.pg_init_privs
                   ) AS m
                     JOIN pg_catalog.pg_class AS c ON c.oid = r.objid
                    WHERE r.classid = 'pg_catalog.pg_class'::regclass
                      AND c.relnamespace <> esquema
                      AND (m.classoid, m.objoid) = (r.classid, r.objid)
                      AND (m.objsubid = r.objsubid OR r.objsubid = 0)
               )
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada ContextoActor V2 rechazada: inventario no acreditado',
            DETAIL = 'huella observada ' || coalesce(observado, '<ausente>');
    END IF;
END
$inventario$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;
DO $retirar_trios$
DECLARE
    tabla regclass;
    punteros constant oid[] := ARRAY[
        'vec_contexto_actor_v1.proyeccion_cuenta_actual'::regclass,
        'vec_contexto_actor_v1.persona_actual'::regclass,
        'vec_contexto_actor_v1.perfil_actual'::regclass,
        'vec_contexto_actor_v1.vinculo_contexto_actual'::regclass,
        'vec_contexto_actor_v1.vinculo_referencia_actual'::regclass
    ]::oid[];
BEGIN
    FOR tabla IN
        SELECT t.tgrelid::regclass
          FROM pg_catalog.pg_trigger AS t
         WHERE t.tgname = 'puntero_actual_no_truncable_v2'
           AND NOT t.tgisinternal
           AND t.tgrelid = ANY(punteros)
         ORDER BY t.tgrelid::regclass::text COLLATE "C"
    LOOP
        EXECUTE pg_catalog.format('DROP TRIGGER avanzar_generacion_punteros_actuales_v2 ON %s RESTRICT', tabla);
        EXECUTE pg_catalog.format('DROP TRIGGER serializar_mutacion_punteros_actuales_v2 ON %s RESTRICT', tabla);
        EXECUTE pg_catalog.format('DROP TRIGGER puntero_actual_no_truncable_v2 ON %s RESTRICT', tabla);
    END LOOP;
END
$retirar_trios$;
DROP FUNCTION
    vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
        text, text, text, text, text, text, numeric, text, numeric,
        text, numeric, text, numeric, text, text, timestamptz, timestamptz
    ) RESTRICT;
DROP FUNCTION vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2() RESTRICT;
DROP FUNCTION vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2() RESTRICT;
DROP TABLE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 RESTRICT;
DO $postcondiciones$
DECLARE
    punteros constant oid[] := ARRAY[
        'vec_contexto_actor_v1.proyeccion_cuenta_actual'::regclass,
        'vec_contexto_actor_v1.persona_actual'::regclass, 'vec_contexto_actor_v1.perfil_actual'::regclass,
        'vec_contexto_actor_v1.vinculo_contexto_actual'::regclass,
        'vec_contexto_actor_v1.vinculo_referencia_actual'::regclass
    ]::oid[];
BEGIN
    IF pg_catalog.to_regclass(
           'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2()'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2()'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)'
       ) IS NOT NULL
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_trigger
            WHERE NOT tgisinternal
              AND tgrelid = ANY(punteros)
              AND tgname IN (
                  'puntero_actual_no_truncable_v2',
                  'serializar_mutacion_punteros_actuales_v2',
                  'avanzar_generacion_punteros_actuales_v2'
              )
       )
       OR pg_catalog.to_regclass(
           'vec_contexto_actor_v1.registros_contexto'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada ContextoActor V2 incompleta';
    END IF;
END
$postcondiciones$;
COMMIT;
