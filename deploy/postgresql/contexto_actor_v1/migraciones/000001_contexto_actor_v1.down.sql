\set ON_ERROR_STOP on
\if :{?confirmar_destruccion_contexto_actor_v1}
\else
DO $confirmacion_ausente$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'falta confirmacion explicita de retirada de ContextoActor V1';
END
$confirmacion_ausente$;
\endif
SELECT :'confirmar_destruccion_contexto_actor_v1' =
       'DESTRUIR_CONTEXTO_ACTOR_V1' AS confirmacion_valida \gset
\if :confirmacion_valida
\else
DO $confirmacion_incorrecta$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'confirmacion de retirada de ContextoActor V1 incorrecta';
END
$confirmacion_incorrecta$;
\endif

BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

DO $precondiciones$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'retirada de ContextoActor V1 requiere superusuario';
    END IF;
END
$precondiciones$;

-- La guarda base precede a cualquier guarda específica: las migraciones
-- posteriores deben tomarla compartida antes de crear un objeto y la retirada
-- la conserva exclusiva hasta el COMMIT. Después se coordina con 000002, que
-- todavía usa únicamente su guarda histórica.
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:migracion:base:v1', 0
    )
);
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:migracion:acreditacion_uso:v2', 0
    )
);

-- Los nombres son deliberadamente explícitos. Una ausencia, renombrado o
-- clase distinta falla antes de inspeccionar evidencia.
LOCK TABLE
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

DO $sin_evidencia$
BEGIN
    IF EXISTS (
        SELECT 1 FROM vec_contexto_actor_v1.procedencias
        UNION ALL
        SELECT 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones
        UNION ALL
        SELECT 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_actual
        UNION ALL
        SELECT 1 FROM vec_contexto_actor_v1.persona_versiones
        UNION ALL
        SELECT 1 FROM vec_contexto_actor_v1.persona_actual
        UNION ALL
        SELECT 1 FROM vec_contexto_actor_v1.perfil_versiones
        UNION ALL
        SELECT 1 FROM vec_contexto_actor_v1.perfil_actual
        UNION ALL
        SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_versiones
        UNION ALL
        SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual
        UNION ALL
        SELECT 1 FROM vec_contexto_actor_v1.vinculo_referencia_versiones
        UNION ALL
        SELECT 1 FROM vec_contexto_actor_v1.vinculo_referencia_actual
        UNION ALL
        SELECT 1 FROM vec_contexto_actor_v1.registros_contexto
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada de ContextoActor V1 rechazada: existe evidencia',
            HINT = 'este down nunca elimina filas; conserve el esquema';
    END IF;
END
$sin_evidencia$;

-- El manifiesto no usa OID. Incluye definición y ACL del esquema, roles del
-- módulo, membresías, objetos, columnas, índices, funciones, restricciones,
-- triggers propios, tipos, reglas, políticas y privilegios predeterminados.
-- Cualquier objeto 000002/futuro, deriva o concesión cambia la huella y
-- aborta sin ejecutar ningún DROP.
DO $manifiesto$
DECLARE
    observado text;
    esperado constant text :=
        '0184a061a46a5bcb379450ea9d5ab9e69393bc487684f7df0ab47c86f0fe4bef';
BEGIN
    WITH elementos AS (
        SELECT format(
                   'esquema|%s|%s|%s|%s',
                   n.nspname,
                   pg_catalog.pg_get_userbyid(n.nspowner),
                   coalesce((
                       SELECT string_agg(
                                  format(
                                      '%s:%s:%s:%s',
                                      CASE WHEN a.grantee = 0 THEN 'PUBLIC'
                                           ELSE pg_catalog.pg_get_userbyid(a.grantee) END,
                                      pg_catalog.pg_get_userbyid(a.grantor),
                                      a.privilege_type,
                                      a.is_grantable
                                  ),
                                  ',' ORDER BY a.grantee, a.grantor,
                                               a.privilege_type, a.is_grantable
                              )
                         FROM pg_catalog.aclexplode(
                             coalesce(
                                 n.nspacl,
                                 pg_catalog.acldefault('n', n.nspowner)
                             )
                         ) AS a
                   ), ''),
                   coalesce(
                       pg_catalog.obj_description(
                           n.oid, 'pg_catalog.pg_namespace'
                       ),
                       ''
                   )
               ) AS elemento
          FROM pg_catalog.pg_namespace AS n
         WHERE n.nspname = 'vec_contexto_actor_v1'
        UNION ALL
        SELECT format(
                   'rol|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
                   r.rolname, r.rolsuper, r.rolinherit, r.rolcreaterole,
                   r.rolcreatedb, r.rolcanlogin, r.rolreplication,
                   r.rolbypassrls, r.rolconnlimit,
                   r.rolvaliduntil IS NULL,
                   CASE WHEN r.rolpassword IS NULL
                        THEN 'credencial_ausente'
                        ELSE 'credencial_presente' END,
                   coalesce(r.rolconfig::text, ''),
                   coalesce(
                       pg_catalog.shobj_description(
                           r.oid, 'pg_catalog.pg_authid'
                       ),
                       ''
                   )
               )
          FROM pg_catalog.pg_roles AS r
         WHERE r.rolname IN (
             'vec_contexto_actor_v1_propietario',
             'vec_contexto_actor_v1_migrador',
             'vec_contexto_actor_v1_runtime'
         )
        UNION ALL
        SELECT format(
                   'membresia|%s|%s|%s|%s|%s',
                   pg_catalog.pg_get_userbyid(m.roleid),
                   pg_catalog.pg_get_userbyid(m.member),
                   pg_catalog.pg_get_userbyid(m.grantor),
                   m.admin_option, m.inherit_option || ':' || m.set_option
               )
          FROM pg_catalog.pg_auth_members AS m
         WHERE m.roleid IN (
                   'vec_contexto_actor_v1_propietario'::regrole,
                   'vec_contexto_actor_v1_migrador'::regrole,
                   'vec_contexto_actor_v1_runtime'::regrole
               )
            OR m.member IN (
                   'vec_contexto_actor_v1_propietario'::regrole,
                   'vec_contexto_actor_v1_migrador'::regrole,
                   'vec_contexto_actor_v1_runtime'::regrole
               )
        UNION ALL
        SELECT format(
                   'base_acl|%s|%s|%s|%s',
                   CASE WHEN a.grantee = 0 THEN 'PUBLIC'
                        ELSE pg_catalog.pg_get_userbyid(a.grantee) END,
                   pg_catalog.pg_get_userbyid(a.grantor),
                   a.privilege_type, a.is_grantable
               )
          FROM pg_catalog.pg_database AS b
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              coalesce(b.datacl, pg_catalog.acldefault('d', b.datdba))
          ) AS a
         WHERE b.datname = current_database()
           AND a.grantee IN (
               'vec_contexto_actor_v1_propietario'::regrole,
               'vec_contexto_actor_v1_migrador'::regrole,
               'vec_contexto_actor_v1_runtime'::regrole
           )
        UNION ALL
        SELECT format(
                   'clase|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
                   c.relkind, c.relname,
                   pg_catalog.pg_get_userbyid(c.relowner),
                   c.relpersistence, c.relrowsecurity,
                   c.relforcerowsecurity, c.relreplident,
                   coalesce(c.reloptions::text, ''),
                   coalesce((
                       SELECT string_agg(
                                  format(
                                      '%s:%s:%s:%s',
                                      CASE WHEN a.grantee = 0 THEN 'PUBLIC'
                                           ELSE pg_catalog.pg_get_userbyid(a.grantee) END,
                                      pg_catalog.pg_get_userbyid(a.grantor),
                                      a.privilege_type, a.is_grantable
                                  ),
                                  ',' ORDER BY a.grantee, a.grantor,
                                               a.privilege_type, a.is_grantable
                              )
                         FROM pg_catalog.aclexplode(
                             coalesce(
                                 c.relacl,
                                 pg_catalog.acldefault(
                                     CASE WHEN c.relkind = 'S' THEN 'S'::"char"
                                          ELSE 'r'::"char" END,
                                     c.relowner
                                 )
                             )
                         ) AS a
                   ), ''),
                   coalesce(
                       pg_catalog.obj_description(
                           c.oid, 'pg_catalog.pg_class'
                       ),
                       ''
                   )
               )
          FROM pg_catalog.pg_class AS c
          JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
         WHERE n.nspname = 'vec_contexto_actor_v1'
        UNION ALL
        SELECT format(
                   'columna|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
                   c.relname, a.attnum, a.attname,
                   pg_catalog.format_type(a.atttypid, a.atttypmod),
                   a.attnotnull, a.attidentity, a.attgenerated,
                   a.attstorage, a.attcompression,
                   a.attstattarget, a.attndims, a.attinhcount, a.attislocal,
                   a.attisdropped, a.atthasdef, a.atthasmissing,
                   coalesce(a.attmissingval::text, ''),
                   CASE WHEN a.attcollation = 0 THEN ''
                        ELSE (
                            SELECT format('%I.%I', nc.nspname, co.collname)
                              FROM pg_catalog.pg_collation AS co
                              JOIN pg_catalog.pg_namespace AS nc
                                ON nc.oid = co.collnamespace
                             WHERE co.oid = a.attcollation
                        ) END,
                   coalesce(a.attoptions::text, ''),
                   coalesce(a.attfdwoptions::text, ''),
                   coalesce((
                       SELECT string_agg(
                                  format(
                                      '%s:%s:%s:%s',
                                      CASE WHEN x.grantee = 0 THEN 'PUBLIC'
                                           ELSE pg_catalog.pg_get_userbyid(x.grantee) END,
                                      pg_catalog.pg_get_userbyid(x.grantor),
                                      x.privilege_type, x.is_grantable
                                  ),
                                  ',' ORDER BY x.grantee, x.grantor,
                                               x.privilege_type, x.is_grantable
                              )
                         FROM pg_catalog.aclexplode(a.attacl) AS x
                   ), ''),
                   coalesce(pg_catalog.pg_get_expr(d.adbin, d.adrelid), ''),
                   coalesce(
                       pg_catalog.col_description(c.oid, a.attnum),
                       ''
                   )
               )
          FROM pg_catalog.pg_attribute AS a
          JOIN pg_catalog.pg_class AS c ON c.oid = a.attrelid
          JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
          LEFT JOIN pg_catalog.pg_attrdef AS d
            ON d.adrelid = a.attrelid AND d.adnum = a.attnum
         WHERE n.nspname = 'vec_contexto_actor_v1'
           AND c.relkind IN ('r', 'p')
           AND a.attnum > 0
        UNION ALL
        SELECT format(
                   'indice|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
                   c.relname, i.indisunique, i.indisprimary,
                   i.indisvalid, i.indisready, i.indislive,
                   i.indisexclusion, i.indimmediate, i.indisclustered,
                   i.indisreplident, pg_catalog.pg_get_indexdef(c.oid)
               )
          FROM pg_catalog.pg_index AS i
          JOIN pg_catalog.pg_class AS c ON c.oid = i.indexrelid
          JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
         WHERE n.nspname = 'vec_contexto_actor_v1'
        UNION ALL
        SELECT format(
                   'funcion|%s|%s|%s|%s|%s|%s|%s|%s',
                   p.prokind, p.proname,
                   pg_catalog.pg_get_function_identity_arguments(p.oid),
                   pg_catalog.pg_get_function_result(p.oid),
                   pg_catalog.pg_get_userbyid(p.proowner),
                   coalesce(p.proacl::text, ''),
                   pg_catalog.pg_get_functiondef(p.oid),
                   coalesce(
                       pg_catalog.obj_description(
                           p.oid, 'pg_catalog.pg_proc'
                       ),
                       ''
                   )
               )
          FROM pg_catalog.pg_proc AS p
          JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
         WHERE n.nspname = 'vec_contexto_actor_v1'
        UNION ALL
        SELECT format(
                   'restriccion|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
                   c.conrelid::regclass::text, c.contype, c.conname,
                   c.condeferrable, c.condeferred, c.convalidated,
                   c.conislocal, c.coninhcount, c.connoinherit,
                   pg_catalog.pg_get_constraintdef(c.oid, false),
                   coalesce(
                       pg_catalog.obj_description(
                           c.oid, 'pg_catalog.pg_constraint'
                       ),
                       ''
                   )
               )
          FROM pg_catalog.pg_constraint AS c
         WHERE c.connamespace = 'vec_contexto_actor_v1'::regnamespace
        UNION ALL
        SELECT format(
                   'trigger|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
                   t.tgrelid::regclass::text,
                   CASE WHEN t.tgisinternal
                        THEN coalesce(c.conname, '<interno_sin_restriccion>')
                        ELSE t.tgname END,
                   t.tgenabled, t.tgisinternal, t.tgtype,
                   t.tgdeferrable, t.tginitdeferred, t.tgnargs,
                   t.tgattr::text,
                   pg_catalog.encode(t.tgargs, 'hex'),
                   coalesce(
                       pg_catalog.pg_get_expr(
                           t.tgqual, t.tgrelid, false
                       ),
                       ''
                   ),
                   coalesce(t.tgoldtable, ''),
                   coalesce(t.tgnewtable, ''),
                   CASE WHEN t.tgconstrrelid = 0 THEN ''
                        ELSE t.tgconstrrelid::regclass::text END,
                   coalesce(ci.relname, ''),
                   t.tgparentid = 0,
                   format(
                       '%I.%I(%s)',
                       nf.nspname, f.proname,
                       pg_catalog.pg_get_function_identity_arguments(f.oid)
                   ),
                   coalesce(
                       pg_catalog.obj_description(
                           t.oid, 'pg_catalog.pg_trigger'
                       ),
                       ''
                   )
               )
          FROM pg_catalog.pg_trigger AS t
          JOIN pg_catalog.pg_proc AS f ON f.oid = t.tgfoid
          JOIN pg_catalog.pg_namespace AS nf ON nf.oid = f.pronamespace
          LEFT JOIN pg_catalog.pg_constraint AS c
            ON c.oid = t.tgconstraint
          LEFT JOIN pg_catalog.pg_class AS ci
            ON ci.oid = t.tgconstrindid
         WHERE t.tgrelid IN (
               SELECT c.oid
                 FROM pg_catalog.pg_class AS c
                WHERE c.relnamespace =
                      'vec_contexto_actor_v1'::regnamespace
           )
        UNION ALL
        SELECT format(
                   'tipo|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
                   t.typtype, t.typname,
                   pg_catalog.pg_get_userbyid(t.typowner),
                   coalesce(t.typacl::text, ''),
                   coalesce(c.relname, ''),
                   coalesce(e.typname, ''),
                   t.typlen, t.typbyval, t.typalign, t.typcategory,
                   t.typispreferred, t.typnotnull, t.typstorage,
                   coalesce(
                       pg_catalog.obj_description(
                           t.oid, 'pg_catalog.pg_type'
                       ),
                       ''
                   )
               )
          FROM pg_catalog.pg_type AS t
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.typnamespace
          LEFT JOIN pg_catalog.pg_class AS c ON c.oid = t.typrelid
          LEFT JOIN pg_catalog.pg_type AS e ON e.oid = t.typelem
         WHERE n.nspname = 'vec_contexto_actor_v1'
        UNION ALL
        SELECT format(
                   'regla|%s|%s|%s',
                   r.ev_class::regclass::text, r.rulename,
                   pg_catalog.pg_get_ruledef(r.oid, false)
               )
          FROM pg_catalog.pg_rewrite AS r
         WHERE r.ev_class IN (
             SELECT c.oid FROM pg_catalog.pg_class AS c
              WHERE c.relnamespace = 'vec_contexto_actor_v1'::regnamespace
         )
        UNION ALL
        SELECT format(
                   'politica|%s|%s|%s|%s|%s',
                   p.polrelid::regclass::text, p.polname, p.polcmd,
                   p.polpermissive, p.polroles::text
               )
          FROM pg_catalog.pg_policy AS p
         WHERE p.polrelid IN (
             SELECT c.oid FROM pg_catalog.pg_class AS c
             WHERE c.relnamespace = 'vec_contexto_actor_v1'::regnamespace
         )
        UNION ALL
        SELECT format(
                   'herencia|%s|%s|%s',
                   h.inhrelid::regclass::text,
                   h.inhparent::regclass::text,
                   h.inhseqno
               )
          FROM pg_catalog.pg_inherits AS h
         WHERE h.inhrelid IN (
                   SELECT c.oid FROM pg_catalog.pg_class AS c
                    WHERE c.relnamespace =
                          'vec_contexto_actor_v1'::regnamespace
               )
            OR h.inhparent IN (
                   SELECT c.oid FROM pg_catalog.pg_class AS c
                    WHERE c.relnamespace =
                          'vec_contexto_actor_v1'::regnamespace
               )
        UNION ALL
        SELECT format(
                   'publicacion_rel|%s|%s|%s|%s|%s',
                   p.pubname, n.nspname, c.relname,
                   coalesce(pr.prattrs::text, ''),
                   coalesce(
                       pg_catalog.pg_get_expr(
                           pr.prqual, pr.prrelid, false
                       ),
                       ''
                   )
               )
          FROM pg_catalog.pg_publication_rel AS pr
          JOIN pg_catalog.pg_publication AS p ON p.oid = pr.prpubid
          JOIN pg_catalog.pg_class AS c ON c.oid = pr.prrelid
          JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
         WHERE n.nspname = 'vec_contexto_actor_v1'
        UNION ALL
        SELECT format(
                   'publicacion_esquema|%s|%s',
                   p.pubname, n.nspname
               )
          FROM pg_catalog.pg_publication_namespace AS pn
          JOIN pg_catalog.pg_publication AS p ON p.oid = pn.pnpubid
          JOIN pg_catalog.pg_namespace AS n ON n.oid = pn.pnnspid
         WHERE n.nspname = 'vec_contexto_actor_v1'
        UNION ALL
        SELECT format(
                   'suscripcion_rel|%s|%s|%s|%s|%s',
                   s.subname, n.nspname, c.relname,
                   sr.srsubstate, coalesce(sr.srsublsn::text, '')
               )
          FROM pg_catalog.pg_subscription_rel AS sr
          JOIN pg_catalog.pg_subscription AS s ON s.oid = sr.srsubid
          JOIN pg_catalog.pg_class AS c ON c.oid = sr.srrelid
          JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
         WHERE n.nspname = 'vec_contexto_actor_v1'
        UNION ALL
        SELECT format(
                   'acl_predeterminada|%s|%s|%s',
                   pg_catalog.pg_get_userbyid(d.defaclrole),
                   d.defaclobjtype, d.defaclacl::text
               )
          FROM pg_catalog.pg_default_acl AS d
         WHERE d.defaclrole =
               'vec_contexto_actor_v1_propietario'::regrole
        UNION ALL
        SELECT 'etiqueta_seguridad|objeto_del_modulo'
          FROM pg_catalog.pg_seclabel AS s
         WHERE (
                   s.classoid = 'pg_catalog.pg_namespace'::regclass
               AND s.objoid = 'vec_contexto_actor_v1'::regnamespace
               )
            OR (
                   s.classoid = 'pg_catalog.pg_class'::regclass
               AND s.objoid IN (
                   SELECT c.oid FROM pg_catalog.pg_class AS c
                    WHERE c.relnamespace =
                          'vec_contexto_actor_v1'::regnamespace
               )
               )
            OR (
                   s.classoid = 'pg_catalog.pg_proc'::regclass
               AND s.objoid IN (
                   SELECT p.oid FROM pg_catalog.pg_proc AS p
                    WHERE p.pronamespace =
                          'vec_contexto_actor_v1'::regnamespace
               )
               )
            OR (
                   s.classoid = 'pg_catalog.pg_type'::regclass
               AND s.objoid IN (
                   SELECT t.oid FROM pg_catalog.pg_type AS t
                    WHERE t.typnamespace =
                          'vec_contexto_actor_v1'::regnamespace
               )
               )
        UNION ALL
        SELECT 'etiqueta_seguridad|rol_del_modulo'
          FROM pg_catalog.pg_shseclabel AS s
         WHERE s.classoid = 'pg_catalog.pg_authid'::regclass
           AND s.objoid IN (
               'vec_contexto_actor_v1_propietario'::regrole,
               'vec_contexto_actor_v1_migrador'::regrole,
               'vec_contexto_actor_v1_runtime'::regrole
           )
        UNION ALL
        SELECT 'privilegio_inicial_extension|objeto_del_modulo'
          FROM pg_catalog.pg_init_privs AS i
         WHERE (
                   i.classoid = 'pg_catalog.pg_class'::regclass
               AND i.objoid IN (
                   SELECT c.oid FROM pg_catalog.pg_class AS c
                    WHERE c.relnamespace =
                          'vec_contexto_actor_v1'::regnamespace
               )
               )
            OR (
                   i.classoid = 'pg_catalog.pg_proc'::regclass
               AND i.objoid IN (
                   SELECT p.oid FROM pg_catalog.pg_proc AS p
                    WHERE p.pronamespace =
                          'vec_contexto_actor_v1'::regnamespace
               )
               )
            OR (
                   i.classoid = 'pg_catalog.pg_type'::regclass
               AND i.objoid IN (
                   SELECT t.oid FROM pg_catalog.pg_type AS t
                    WHERE t.typnamespace =
                          'vec_contexto_actor_v1'::regnamespace
               )
               )
        UNION ALL
        SELECT format('objeto_nsp|collation|%s', c.collname)
          FROM pg_catalog.pg_collation AS c
         WHERE c.collnamespace = 'vec_contexto_actor_v1'::regnamespace
        UNION ALL
        SELECT format('objeto_nsp|conversion|%s', c.conname)
          FROM pg_catalog.pg_conversion AS c
         WHERE c.connamespace = 'vec_contexto_actor_v1'::regnamespace
        UNION ALL
        SELECT format('objeto_nsp|operator|%s', o.oprname)
          FROM pg_catalog.pg_operator AS o
         WHERE o.oprnamespace = 'vec_contexto_actor_v1'::regnamespace
        UNION ALL
        SELECT format('objeto_nsp|opclass|%s', o.opcname)
          FROM pg_catalog.pg_opclass AS o
         WHERE o.opcnamespace = 'vec_contexto_actor_v1'::regnamespace
        UNION ALL
        SELECT format('objeto_nsp|opfamily|%s', o.opfname)
          FROM pg_catalog.pg_opfamily AS o
         WHERE o.opfnamespace = 'vec_contexto_actor_v1'::regnamespace
        UNION ALL
        SELECT format('objeto_nsp|estadistica|%s', s.stxname)
          FROM pg_catalog.pg_statistic_ext AS s
         WHERE s.stxnamespace = 'vec_contexto_actor_v1'::regnamespace
        UNION ALL
        SELECT format('objeto_nsp|ts_config|%s', t.cfgname)
          FROM pg_catalog.pg_ts_config AS t
         WHERE t.cfgnamespace = 'vec_contexto_actor_v1'::regnamespace
        UNION ALL
        SELECT format('objeto_nsp|ts_dict|%s', t.dictname)
          FROM pg_catalog.pg_ts_dict AS t
         WHERE t.dictnamespace = 'vec_contexto_actor_v1'::regnamespace
        UNION ALL
        SELECT format('objeto_nsp|ts_parser|%s', t.prsname)
          FROM pg_catalog.pg_ts_parser AS t
         WHERE t.prsnamespace = 'vec_contexto_actor_v1'::regnamespace
        UNION ALL
        SELECT format('objeto_nsp|ts_template|%s', t.tmplname)
          FROM pg_catalog.pg_ts_template AS t
         WHERE t.tmplnamespace = 'vec_contexto_actor_v1'::regnamespace
    )
    SELECT pg_catalog.encode(
               pg_catalog.sha256(
                   pg_catalog.convert_to(
                       string_agg(elemento, E'\n' ORDER BY elemento),
                       'UTF8'
                   )
               ),
               'hex'
           )
      INTO observado
      FROM elementos;

    IF observado IS DISTINCT FROM esperado THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada de ContextoActor V1 rechazada: manifiesto no acreditado',
            DETAIL = 'huella observada ' || coalesce(observado, '<ausente>'),
            HINT = 'retire antes migraciones, consumidores o derivas; no repare en este down';
    END IF;
END
$manifiesto$;

SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;

DROP TABLE vec_contexto_actor_v1.registros_contexto RESTRICT;
DROP TABLE vec_contexto_actor_v1.vinculo_referencia_actual RESTRICT;
DROP TABLE vec_contexto_actor_v1.vinculo_referencia_versiones RESTRICT;
DROP TABLE vec_contexto_actor_v1.vinculo_contexto_actual RESTRICT;
DROP TABLE vec_contexto_actor_v1.vinculo_contexto_versiones RESTRICT;
DROP TABLE vec_contexto_actor_v1.perfil_actual RESTRICT;
DROP TABLE vec_contexto_actor_v1.perfil_versiones RESTRICT;
DROP TABLE vec_contexto_actor_v1.persona_actual RESTRICT;
DROP TABLE vec_contexto_actor_v1.persona_versiones RESTRICT;
DROP TABLE vec_contexto_actor_v1.proyeccion_cuenta_actual RESTRICT;
DROP TABLE vec_contexto_actor_v1.proyeccion_cuenta_versiones RESTRICT;
DROP TABLE vec_contexto_actor_v1.procedencias RESTRICT;

DROP FUNCTION
    vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
        text, text, text, text, text, text, timestamptz
    ) RESTRICT;
DROP FUNCTION
    vec_contexto_actor_v1.reconciliar_contexto_actor_v2(
        text, text, text, text, text, text, timestamptz
    ) RESTRICT;
DROP FUNCTION
    vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()
    RESTRICT;
DROP FUNCTION
    vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()
    RESTRICT;
DROP FUNCTION
    vec_contexto_actor_v1.privilegios_efectivos_runtime_minimos(
        oid, oid, oid, oid[]
    ) RESTRICT;
DROP FUNCTION
    vec_contexto_actor_v1.validar_procedencia_monotona()
    RESTRICT;
DROP FUNCTION vec_contexto_actor_v1.rechazar_truncado() RESTRICT;
DROP FUNCTION
    vec_contexto_actor_v1.rechazar_mutacion_historia()
    RESTRICT;
DROP FUNCTION
    vec_contexto_actor_v1.procedencia_valida(
        text, numeric, text, text
    ) RESTRICT;
DROP FUNCTION
    vec_contexto_actor_v1.referencia_operacion_valida(text, text)
    RESTRICT;
DROP FUNCTION
    vec_contexto_actor_v1.instante_valido(timestamptz)
    RESTRICT;
DROP FUNCTION
    vec_contexto_actor_v1.referencia_valida(text, text)
    RESTRICT;

-- 000001 creó únicamente estas dos ACL predeterminadas globales del
-- propietario. El manifiesto ya ha probado que siguen exactas; restaurar los
-- valores nativos permite que roles_down retire después el rol sin una
-- destrucción global de objetos.
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_contexto_actor_v1_propietario
    GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_contexto_actor_v1_propietario
    GRANT USAGE ON TYPES TO PUBLIC;

DROP SCHEMA vec_contexto_actor_v1 RESTRICT;
COMMIT;
