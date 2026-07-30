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

-- Los seis AccessExclusive cierran la inspección frente a DDL, DML y cambios
-- del trío de disparadores. La guarda advisory se tomó antes que las tablas.
LOCK TABLE
    vec_contexto_actor_v1.control_generacion_punteros_actuales_v2,
    vec_contexto_actor_v1.proyeccion_cuenta_actual,
    vec_contexto_actor_v1.persona_actual,
    vec_contexto_actor_v1.perfil_actual,
    vec_contexto_actor_v1.vinculo_contexto_actual,
    vec_contexto_actor_v1.vinculo_referencia_actual
IN ACCESS EXCLUSIVE MODE;

DO $inventario$
DECLARE
    propietario constant oid :=
        'vec_contexto_actor_v1_propietario'::regrole;
    esquema constant oid := 'vec_contexto_actor_v1'::regnamespace;
    control constant oid :=
        'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass;
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
        'vec_contexto_actor_v1.persona_actual'::regclass,
        'vec_contexto_actor_v1.perfil_actual'::regclass,
        'vec_contexto_actor_v1.vinculo_contexto_actual'::regclass,
        'vec_contexto_actor_v1.vinculo_referencia_actual'::regclass
    ]::oid[];
    funciones oid[];
    observado text;
    esperado constant text :=
        '860f743862b27fd6bacaed4e4dae3e7b7ed75e12c44bc39f28d79bafb9c50f0d';
BEGIN
    funciones := ARRAY[acreditar, avanzar, serializar];
    IF pg_catalog.current_setting('server_version_num')::integer / 10000 <> 18
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
       ) <> 3 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada ContextoActor V2 rechazada: funciones incompletas';
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

    -- Manifiesto simbólico completo de 000001+000002. No incluye OID ni datos
    -- de negocio; sí forma, propietario, ACL, cuerpo, comentarios y catálogos.
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
            'a|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
            c.relname, a.attnum, a.attname,
            pg_catalog.format_type(a.atttypid, a.atttypmod),
            a.attnotnull, a.attidentity, a.attgenerated, a.attisdropped,
            a.attstorage, a.attcompression,
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
            'i|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
            i.indrelid::regclass::text, ci.relname, i.indisunique,
            i.indisprimary, i.indisexclusion, i.indimmediate,
            i.indisvalid, i.indisready, i.indislive, i.indisreplident,
            i.indcheckxmin, i.indnullsnotdistinct,
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
            'g|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
            t.tgrelid::regclass::text,
            CASE WHEN t.tgisinternal THEN coalesce(c.conname, '<interno>')
                 ELSE t.tgname END,
            t.tgtype, t.tgenabled, t.tgisinternal,
            t.tgfoid::regprocedure::text, t.tgdeferrable,
            t.tginitdeferred, t.tgnargs, pg_catalog.encode(t.tgargs, 'hex'),
            t.tgattr::text,
            coalesce(pg_catalog.pg_get_expr(t.tgqual, t.tgrelid, false), ''),
            coalesce(t.tgoldtable, ''), coalesce(t.tgnewtable, ''),
            t.tgparentid = 0, t.tgconstrrelid = 0, t.tgconstrindid = 0
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
            p.pronargs, p.pronargdefaults, p.proargtypes::text,
            coalesce(p.proallargtypes::text, ''),
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
            p.polpermissive, p.polroles::text,
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
            'd|%s|%s|%s', d.defaclobjtype,
            d.defaclrole::regrole::text, d.defaclacl::text
        ) FROM pg_catalog.pg_default_acl AS d
         WHERE d.defaclrole = propietario
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
           SELECT 1 FROM pg_catalog.pg_shdepend AS d
            WHERE d.classid IN (
                      'pg_catalog.pg_class'::regclass,
                      'pg_catalog.pg_proc'::regclass,
                      'pg_catalog.pg_type'::regclass
                  )
              AND d.objid = ANY(ARRAY[
                  control, acreditar, avanzar, serializar,
                  (SELECT reltype FROM pg_catalog.pg_class WHERE oid = control),
                  (SELECT typarray FROM pg_catalog.pg_type
                    WHERE typrelid = control)
              ]::oid[])
              AND d.deptype <> 'o'
       )
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
           SELECT 1 FROM pg_catalog.pg_seclabel AS s
            WHERE (s.classoid, s.objoid) IN (
                ('pg_catalog.pg_class'::regclass, control),
                ('pg_catalog.pg_proc'::regclass, acreditar),
                ('pg_catalog.pg_proc'::regclass, avanzar),
                ('pg_catalog.pg_proc'::regclass, serializar)
            )
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_init_privs AS i
            WHERE (i.classoid, i.objoid) IN (
                ('pg_catalog.pg_class'::regclass, control),
                ('pg_catalog.pg_proc'::regclass, acreditar),
                ('pg_catalog.pg_proc'::regclass, avanzar),
                ('pg_catalog.pg_proc'::regclass, serializar)
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
BEGIN
    FOR tabla IN
        SELECT t.tgrelid::regclass
          FROM pg_catalog.pg_trigger AS t
         WHERE t.tgname = 'puntero_actual_no_truncable_v2'
           AND NOT t.tgisinternal
         ORDER BY t.tgrelid::regclass::text COLLATE "C"
    LOOP
        EXECUTE pg_catalog.format(
            'DROP TRIGGER avanzar_generacion_punteros_actuales_v2 ON %s RESTRICT',
            tabla
        );
        EXECUTE pg_catalog.format(
            'DROP TRIGGER serializar_mutacion_punteros_actuales_v2 ON %s RESTRICT',
            tabla
        );
        EXECUTE pg_catalog.format(
            'DROP TRIGGER puntero_actual_no_truncable_v2 ON %s RESTRICT',
            tabla
        );
    END LOOP;
END
$retirar_trios$;

DROP FUNCTION
    vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
        text, text, text, text, text, text, numeric, text, numeric,
        text, numeric, text, numeric, text, text, timestamptz, timestamptz
    ) RESTRICT;
DROP FUNCTION
    vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2()
    RESTRICT;
DROP FUNCTION
    vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2()
    RESTRICT;
DROP TABLE
    vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
    RESTRICT;

DO $postcondiciones$
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
