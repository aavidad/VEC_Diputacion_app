-- Retirada gobernada de la proyeccion organizativa C2.2-A.
-- Solo admite una instalacion vacia, exacta y sin consumidores posteriores.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

DO $precondiciones_minimas$
BEGIN
    IF pg_catalog.current_setting(
           'vec.confirmar_retirada_organizacion_corporativa_v1', true
       ) IS DISTINCT FROM 'RETIRAR_ORGANIZACION_CORPORATIVA_V1' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada de organizacion corporativa V1 requiere confirmacion explicita';
    ELSIF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'retirada de organizacion corporativa V1 requiere superusuario';
    ELSIF pg_catalog.current_setting('server_version_num')::integer < 180000
       OR pg_catalog.current_setting('server_version_num')::integer >= 190000 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada de organizacion corporativa V1 requiere PostgreSQL 18';
    END IF;
END
$precondiciones_minimas$;

SELECT pg_catalog.pg_advisory_xact_lock_shared(
    pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:migracion:acreditacion_uso:v2', 0
    )
);
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:organizacion-corporativa-rrhh:v1', 0
    )
);
LOCK TABLE vec_contexto_actor_v1.organizacion_actual
IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_contexto_actor_v1.organizacion_versiones
IN ACCESS EXCLUSIVE MODE;
LOCK TABLE
    pg_catalog.pg_authid, pg_catalog.pg_auth_members,
    pg_catalog.pg_db_role_setting, pg_catalog.pg_class,
    pg_catalog.pg_attribute, pg_catalog.pg_index,
    pg_catalog.pg_namespace, pg_catalog.pg_language,
    pg_catalog.pg_collation, pg_catalog.pg_proc, pg_catalog.pg_type,
    pg_catalog.pg_default_acl, pg_catalog.pg_description,
    pg_catalog.pg_seclabel, pg_catalog.pg_init_privs,
    pg_catalog.pg_depend, pg_catalog.pg_shdepend,
    pg_catalog.pg_constraint, pg_catalog.pg_trigger, pg_catalog.pg_policy,
    pg_catalog.pg_publication, pg_catalog.pg_publication_namespace,
    pg_catalog.pg_publication_rel, pg_catalog.pg_subscription_rel,
    pg_catalog.pg_statistic_ext
IN SHARE MODE;

DO $inventario$
DECLARE
    propietario constant oid := 'vec_contexto_actor_v1_propietario'::regrole;
    migrador constant oid := 'vec_contexto_actor_v1_migrador'::regrole;
    runtime constant oid := 'vec_contexto_actor_v1_runtime'::regrole;
    esquema constant oid := 'vec_contexto_actor_v1'::regnamespace;
    versiones constant oid := 'vec_contexto_actor_v1.organizacion_versiones'::regclass;
    actual constant oid := 'vec_contexto_actor_v1.organizacion_actual'::regclass;
    validador constant oid := 'vec_contexto_actor_v1.organizacion_ref_valida(text)'::regprocedure;
    toast_versiones oid;
    toast_actual oid;
    indices oid[];
    tipos oid[];
    restricciones oid[];
    disparadores oid[];
    politicas oid[];
BEGIN
    -- Se reacreditan las precondiciones despues de inmovilizar el catalogo.
    IF pg_catalog.current_setting(
           'vec.confirmar_retirada_organizacion_corporativa_v1', true
       ) IS DISTINCT FROM 'RETIRAR_ORGANIZACION_CORPORATIVA_V1'
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_authid
            WHERE rolname = current_user AND rolsuper
       ) OR pg_catalog.current_setting('server_version_num')::integer < 180000
       OR pg_catalog.current_setting('server_version_num')::integer >= 190000
       OR EXISTS (
           SELECT 1 FROM vec_contexto_actor_v1.organizacion_versiones
       ) OR EXISTS (
           SELECT 1 FROM vec_contexto_actor_v1.organizacion_actual
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada de organizacion corporativa V1 rechazada por precondicion o evidencia';
    END IF;

    IF (
        SELECT pg_catalog.count(*) FROM pg_catalog.pg_authid
         WHERE oid IN (propietario,migrador,runtime)
           AND NOT rolsuper AND NOT rolcanlogin AND NOT rolinherit
           AND NOT rolcreaterole AND NOT rolcreatedb
           AND NOT rolreplication AND NOT rolbypassrls
           AND rolconnlimit=-1 AND rolpassword IS NULL
           AND rolvaliduntil IS NULL
    ) <> 3 OR (
        SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members AS m
         WHERE m.roleid=propietario
            OR m.member=ANY(ARRAY[propietario,migrador,runtime]::oid[])
            OR m.roleid=migrador
    ) <> 1 OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members AS m
         WHERE m.roleid=propietario AND m.member=migrador
           AND NOT m.admin_option AND NOT m.inherit_option AND m.set_option
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members AS m
        JOIN pg_catalog.pg_authid AS login ON login.oid=m.member
         WHERE m.roleid=runtime
           AND (m.admin_option OR NOT m.inherit_option OR m.set_option
                OR NOT login.rolcanlogin OR login.rolsuper
                OR NOT login.rolinherit OR login.rolcreaterole
                OR login.rolcreatedb OR login.rolreplication
                OR login.rolbypassrls
                OR (SELECT pg_catalog.count(*)
                      FROM pg_catalog.pg_auth_members AS otra
                     WHERE otra.member=m.member) <> 1
                OR EXISTS (
                    SELECT 1 FROM pg_catalog.pg_db_role_setting AS ajuste
                     WHERE ajuste.setrole=m.member
                ))
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members AS m
         WHERE m.roleid = (
                   SELECT oid FROM pg_catalog.pg_authid
                    WHERE rolname =
                      'vec_contexto_actor_corporativo_rrhh_selector'
               ) OR m.member = (
                   SELECT oid FROM pg_catalog.pg_authid
                    WHERE rolname =
                      'vec_contexto_actor_corporativo_rrhh_selector'
               )
    ) OR EXISTS (
        WITH roles_objetivo AS (
            SELECT oid FROM pg_catalog.pg_authid
             WHERE rolname IN (
                 'vec_contexto_actor_v1_runtime',
                 'vec_contexto_actor_corporativo_rrhh_selector'
             )
            UNION
            SELECT m.member FROM pg_catalog.pg_auth_members AS m
             WHERE m.roleid=runtime
        ), tablas AS (
            SELECT versiones AS oid UNION ALL SELECT actual
        ), tipos_objetivo AS (
            SELECT t.oid FROM pg_catalog.pg_type AS t
             WHERE t.typrelid IN (versiones,actual)
                OR t.typelem IN (
                    SELECT fila.oid FROM pg_catalog.pg_type AS fila
                     WHERE fila.typrelid IN (versiones,actual)
                )
        )
        SELECT 1 FROM roles_objetivo AS rol
         WHERE pg_catalog.has_function_privilege(
                   rol.oid,validador,'EXECUTE'
               ) OR EXISTS (
                   SELECT 1 FROM tablas AS tabla
                    WHERE pg_catalog.has_table_privilege(
                              rol.oid,tabla.oid,'SELECT'
                          ) OR pg_catalog.has_table_privilege(
                              rol.oid,tabla.oid,'INSERT'
                          ) OR pg_catalog.has_table_privilege(
                              rol.oid,tabla.oid,'UPDATE'
                          ) OR pg_catalog.has_table_privilege(
                              rol.oid,tabla.oid,'DELETE'
                          ) OR pg_catalog.has_table_privilege(
                              rol.oid,tabla.oid,'TRUNCATE'
                          ) OR pg_catalog.has_table_privilege(
                              rol.oid,tabla.oid,'REFERENCES'
                          ) OR pg_catalog.has_table_privilege(
                              rol.oid,tabla.oid,'TRIGGER'
                          ) OR pg_catalog.has_table_privilege(
                              rol.oid,tabla.oid,'MAINTAIN'
                          ) OR pg_catalog.has_any_column_privilege(
                              rol.oid,tabla.oid,'SELECT'
                          ) OR pg_catalog.has_any_column_privilege(
                              rol.oid,tabla.oid,'INSERT'
                          ) OR pg_catalog.has_any_column_privilege(
                              rol.oid,tabla.oid,'UPDATE'
                          ) OR pg_catalog.has_any_column_privilege(
                              rol.oid,tabla.oid,'REFERENCES'
                          )
               ) OR EXISTS (
                   SELECT 1 FROM tipos_objetivo AS tipo
                    WHERE pg_catalog.has_type_privilege(
                              rol.oid,tipo.oid,'USAGE'
                          )
               )
    ) THEN
        RAISE EXCEPTION USING ERRCODE='55000',
            MESSAGE='retirada rechazada: topologia o acceso efectivo hostil';
    END IF;

    SELECT reltoastrelid INTO STRICT toast_versiones
      FROM pg_catalog.pg_class WHERE oid = versiones;
    SELECT reltoastrelid INTO STRICT toast_actual
      FROM pg_catalog.pg_class WHERE oid = actual;
    SELECT pg_catalog.array_agg(indexrelid ORDER BY indexrelid)
      INTO indices
      FROM pg_catalog.pg_index
     WHERE indrelid IN (versiones, actual, toast_versiones, toast_actual);
    SELECT pg_catalog.array_agg(oid ORDER BY oid)
      INTO tipos
      FROM pg_catalog.pg_type
     WHERE typrelid IN (versiones, actual)
        OR typelem IN (
            SELECT oid FROM pg_catalog.pg_type
             WHERE typrelid IN (versiones, actual)
        );
    SELECT pg_catalog.array_agg(oid ORDER BY oid)
      INTO restricciones
      FROM pg_catalog.pg_constraint
     WHERE conrelid IN (versiones, actual);
    SELECT pg_catalog.array_agg(oid ORDER BY oid)
      INTO disparadores
      FROM pg_catalog.pg_trigger
     WHERE tgrelid IN (versiones, actual);
    SELECT pg_catalog.array_agg(oid ORDER BY oid)
      INTO politicas
      FROM pg_catalog.pg_policy
     WHERE polrelid IN (versiones, actual);

    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_class
         WHERE oid IN (versiones, actual) AND relkind = 'r'
           AND relpersistence = 'p' AND relowner = propietario
           AND relrowsecurity AND relforcerowsecurity
           AND reloptions IS NULL AND relreplident = 'd'
           AND relispartition IS FALSE AND relhassubclass IS FALSE
    ) <> 2 OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_class
         WHERE oid IN (toast_versiones, toast_actual) AND relkind = 't'
           AND relpersistence = 'p' AND relowner = propietario
           AND relnamespace = 'pg_toast'::regnamespace
           AND relacl IS NULL AND reloptions IS NULL AND reltoastrelid = 0
    ) <> 2 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada rechazada: relaciones o TOAST no exactas';
    END IF;

    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_type
         WHERE typrelid IN (versiones, actual) AND typtype = 'c'
           AND typowner = propietario AND typacl IS NOT NULL
           AND typelem = 0 AND typarray <> 0
    ) <> 2 OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_type AS matriz
          JOIN pg_catalog.pg_type AS fila ON fila.typarray = matriz.oid
         WHERE fila.typrelid IN (versiones, actual)
           AND matriz.typtype = 'b' AND matriz.typcategory = 'A'
           AND matriz.typowner = propietario AND matriz.typacl IS NULL
           AND matriz.typelem = fila.oid AND matriz.typarray = 0
    ) <> 2 OR (
        SELECT pg_catalog.count(*) FROM pg_catalog.pg_type
         WHERE oid = ANY(tipos)
    ) <> 4 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada rechazada: tipos fila o matrices no exactos';
    END IF;

    IF (
        SELECT pg_catalog.string_agg(
                   pg_catalog.format(
                       '%s|%s|%s|%s|%s', attnum, attname,
                       pg_catalog.format_type(atttypid, atttypmod),
                       attnotnull, attcollation
                   ), ';' ORDER BY attnum
               )
          FROM pg_catalog.pg_attribute
         WHERE attrelid = versiones AND attnum > 0 AND NOT attisdropped
    ) IS DISTINCT FROM
       '1|organizacion_ref|text|t|100;2|version|numeric(20,0)|t|0;3|procedencia_ref|text|t|100;4|procedencia_version|numeric(20,0)|t|0;5|procedencia_huella_sha256|text|t|100;6|procedencia_autoridad|text|t|100;7|estado|text|t|100;8|vigente_desde|timestamp(6) with time zone|t|0;9|vigente_hasta|timestamp(6) with time zone|t|0'
       OR (
        SELECT pg_catalog.string_agg(
                   pg_catalog.format(
                       '%s|%s|%s|%s|%s', attnum, attname,
                       pg_catalog.format_type(atttypid, atttypmod),
                       attnotnull, attcollation
                   ), ';' ORDER BY attnum
               )
          FROM pg_catalog.pg_attribute
         WHERE attrelid = actual AND attnum > 0 AND NOT attisdropped
       ) IS DISTINCT FROM
       '1|organizacion_ref|text|t|100;2|version|numeric(20,0)|t|0'
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_attribute
            WHERE attrelid IN (versiones, actual) AND attnum > 0
              AND NOT attisdropped
              AND (atthasdef OR attidentity <> '' OR attgenerated <> ''
                   OR attacl IS NOT NULL OR attfdwoptions IS NOT NULL
                   OR attoptions IS NOT NULL OR attstattarget <> -1
                   OR attcompression <> ''
                   OR (atttypid = 'text'::regtype AND attstorage <> 'x')
                   OR (atttypid = 'numeric'::regtype AND attstorage <> 'm')
                   OR (atttypid = 'timestamptz'::regtype AND attstorage <> 'p'))
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada rechazada: columnas no exactas';
    END IF;

    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_constraint
         WHERE conrelid IN (versiones, actual)
    ) <> 25 OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_constraint
         WHERE conrelid IN (versiones, actual)
           AND conname <> ALL(ARRAY[
               'organizacion_actual_organizacion_ref_not_null',
               'organizacion_actual_pk', 'organizacion_actual_version_ck',
               'organizacion_actual_version_fk',
               'organizacion_actual_version_not_null',
               'organizacion_versiones_autoridad_ck',
               'organizacion_versiones_estado_ck',
               'organizacion_versiones_estado_not_null',
               'organizacion_versiones_organizacion_ref_not_null',
               'organizacion_versiones_pk',
               'organizacion_versiones_procedencia_autoridad_not_null',
               'organizacion_versiones_procedencia_ck',
               'organizacion_versiones_procedencia_fk',
               'organizacion_versiones_procedencia_huella_sha256_not_null',
               'organizacion_versiones_procedencia_ref_not_null',
               'organizacion_versiones_procedencia_uq',
               'organizacion_versiones_procedencia_version_not_null',
               'organizacion_versiones_ref_ck',
               'organizacion_versiones_ventana_ck',
               'organizacion_versiones_version_ck',
               'organizacion_versiones_version_not_null',
               'organizacion_versiones_vigente_desde_ck',
               'organizacion_versiones_vigente_desde_not_null',
               'organizacion_versiones_vigente_hasta_ck',
               'organizacion_versiones_vigente_hasta_not_null'
           ])
    ) OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_constraint
         WHERE conrelid IN (versiones, actual)
           AND NOT condeferrable AND NOT condeferred AND convalidated
    ) <> 25 OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_constraint
         WHERE conrelid IN (versiones, actual) AND contype = 'f'
           AND confmatchtype = 'f' AND confupdtype = 'a'
           AND confdeltype = 'a'
    ) <> 2 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada rechazada: restricciones no exactas';
    END IF;
    IF (
        SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
                   pg_catalog.string_agg(pg_catalog.format(
                     '%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
                     conname, contype::text, conkey::text,
                     CASE WHEN confrelid=0 THEN ''
                          ELSE confrelid::regclass::text END,
                     confkey::text, confmatchtype::text,
                     confupdtype::text, confdeltype::text,
                     condeferrable, condeferred, convalidated,
                     pg_catalog.pg_get_constraintdef(oid,false)
                   ), E'\n' ORDER BY conname), 'UTF8')), 'hex')
          FROM pg_catalog.pg_constraint
         WHERE conrelid IN (versiones, actual)
    ) IS DISTINCT FROM
      'e4a9482197ea72dbeba0b8cc8a4c47790b8cca8e52283bd2de82920481073ec4'
    THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada rechazada: definiciones de restriccion no exactas';
    END IF;

    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_index AS x
          JOIN pg_catalog.pg_class AS i ON i.oid = x.indexrelid
         WHERE x.indrelid IN (versiones, actual)
           AND i.relname = ANY(ARRAY[
               'organizacion_versiones_pk',
               'organizacion_versiones_procedencia_uq',
               'organizacion_actual_pk'
           ]) AND i.relowner = propietario AND i.relpersistence = 'p'
           AND i.relacl IS NULL AND i.reloptions IS NULL
           AND i.reltablespace = 0
           AND i.relam = 403
           AND x.indisunique AND NOT x.indisexclusion
           AND x.indimmediate AND x.indisvalid AND x.indisready
           AND x.indislive AND NOT x.indisclustered
           AND x.indexprs IS NULL AND x.indpred IS NULL
    ) <> 3 OR (
        SELECT pg_catalog.count(*) FROM pg_catalog.pg_index
         WHERE indrelid IN (versiones, actual)
    ) <> 3 OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_index AS x
          JOIN pg_catalog.pg_class AS i ON i.oid = x.indexrelid
         WHERE x.indrelid IN (toast_versiones, toast_actual)
           AND i.relowner = propietario AND i.relacl IS NULL
           AND i.relkind = 'i' AND i.relpersistence = 'p'
           AND x.indisunique AND x.indisprimary AND x.indisvalid
           AND x.indisready AND x.indislive
    ) <> 2 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada rechazada: indices no exactos';
    END IF;
    IF (
        SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
                   pg_catalog.string_agg(pg_catalog.format(
                     '%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
                     c.relname, i.relname, x.indkey::text,
                     x.indclass::text, x.indcollation::text,
                     x.indoption::text, x.indnkeyatts, x.indnatts,
                     x.indisprimary, x.indisunique,
                     pg_catalog.pg_get_indexdef(i.oid)
                   ), E'\n' ORDER BY c.relname,i.relname), 'UTF8')), 'hex')
          FROM pg_catalog.pg_index AS x
          JOIN pg_catalog.pg_class AS c ON c.oid=x.indrelid
          JOIN pg_catalog.pg_class AS i ON i.oid=x.indexrelid
         WHERE x.indrelid IN (versiones,actual)
    ) IS DISTINCT FROM
      'd7dd73ef70b6c43829e458329820a136942808d8ad4969fa1ce6edfd309d29a5'
    THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada rechazada: definiciones de indice no exactas';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc
         WHERE oid = validador AND pronamespace = esquema
           AND proname = 'organizacion_ref_valida'
           AND proowner = propietario AND prolang = (
               SELECT oid FROM pg_catalog.pg_language WHERE lanname = 'sql'
           )
           AND prokind = 'f' AND provolatile = 'i' AND proparallel = 'u'
           AND NOT proisstrict AND NOT prosecdef AND NOT proleakproof
           AND NOT proretset AND pronargs = 1 AND pronargdefaults = 0
           AND prorettype = 'boolean'::regtype
           AND procost = 100 AND prorows = 0
           AND provariadic = 0 AND prosupport = 0
           AND proargtypes = '25'::oidvector
           AND proallargtypes IS NULL AND proargmodes IS NULL
           AND proargnames = ARRAY['p_valor']::text[]
           AND proargdefaults IS NULL AND protrftypes IS NULL
           AND probin IS NULL AND prosqlbody IS NULL
           AND proconfig = ARRAY['search_path=pg_catalog']
           AND pg_catalog.encode(pg_catalog.sha256(
                   pg_catalog.convert_to(prosrc, 'UTF8')
               ), 'hex') =
               '1ca9bb73cb52682eba6a2d78b3a2243630d31fed62d6f2902b6cda6c64ae5633'
    ) OR (
        SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc
         WHERE pronamespace = esquema AND proname = 'organizacion_ref_valida'
    ) <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada rechazada: validador no exacto';
    END IF;

    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_policy
         WHERE polrelid IN (versiones, actual)
           AND polname = 'acceso_propietario_exacto'
           AND polpermissive AND polcmd = '*'
           AND polroles = ARRAY[propietario]::oid[]
           AND pg_catalog.pg_get_expr(polqual, polrelid) =
               '(CURRENT_USER = ''vec_contexto_actor_v1_propietario''::name)'
           AND pg_catalog.pg_get_expr(polwithcheck, polrelid) =
               '(CURRENT_USER = ''vec_contexto_actor_v1_propietario''::name)'
    ) <> 2 OR (
        SELECT pg_catalog.count(*) FROM pg_catalog.pg_policy
         WHERE polrelid IN (versiones, actual)
    ) <> 2 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada rechazada: politicas no exactas';
    END IF;

    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_trigger
         WHERE tgrelid = versiones AND NOT tgisinternal
           AND tgenabled = 'O' AND NOT tgdeferrable
           AND NOT tginitdeferred AND tgnargs = 0 AND tgargs = '\x'::bytea
           AND tgattr = ''::int2vector AND tgqual IS NULL
           AND tgoldtable IS NULL AND tgnewtable IS NULL
           AND ((tgname = 'historia_inmutable' AND tgtype = 27
                 AND tgfoid = 'vec_contexto_actor_v1.rechazar_mutacion_historia()'::regprocedure)
                OR (tgname = 'historia_no_truncable' AND tgtype = 34
                 AND tgfoid = 'vec_contexto_actor_v1.rechazar_truncado()'::regprocedure))
    ) <> 2 OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_trigger
         WHERE tgrelid = actual AND NOT tgisinternal
           AND tgenabled = 'O' AND NOT tgdeferrable
           AND NOT tginitdeferred AND tgnargs = 0 AND tgargs = '\x'::bytea
           AND tgattr = ''::int2vector AND tgqual IS NULL
           AND tgoldtable IS NULL AND tgnewtable IS NULL
           AND ((tgname = 'puntero_actual_no_truncable_v2' AND tgtype = 34
                 AND tgfoid = 'vec_contexto_actor_v1.rechazar_truncado()'::regprocedure)
                OR (tgname = 'serializar_mutacion_punteros_actuales_v2' AND tgtype = 30
                 AND tgfoid = 'vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2()'::regprocedure)
                OR (tgname = 'avanzar_generacion_punteros_actuales_v2' AND tgtype = 28
                 AND tgfoid = 'vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2()'::regprocedure))
    ) <> 3 OR (
        SELECT pg_catalog.count(*) FROM pg_catalog.pg_trigger
         WHERE tgrelid IN (versiones, actual) AND NOT tgisinternal
    ) <> 5 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada rechazada: disparadores no exactos';
    END IF;
    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_trigger
         WHERE tgrelid IN (versiones, actual) AND tgisinternal
           AND tgenabled = 'O' AND tgconstraint = ANY(restricciones)
           AND tgnargs = 0 AND tgargs = '\x'::bytea
           AND tgattr = ''::int2vector AND tgqual IS NULL
           AND tgoldtable IS NULL AND tgnewtable IS NULL
           AND tgfoid::regprocedure::text = ANY(ARRAY[
               '"RI_FKey_check_ins"()', '"RI_FKey_check_upd"()',
               '"RI_FKey_noaction_del"()', '"RI_FKey_noaction_upd"()'
           ])
    ) <> 6 OR (
        SELECT pg_catalog.count(*) FROM pg_catalog.pg_trigger
         WHERE tgrelid IN (versiones,actual) AND tgisinternal
    ) <> 6 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada rechazada: triggers FK internos no exactos';
    END IF;

    -- ACL exacta: ocho privilegios de tabla, EXECUTE y USAGE, todos solo
    -- para el propietario y sin opcion de concesion.
    IF EXISTS (
        SELECT 1
          FROM (
              SELECT c.oid, pg_catalog.count(a.*) AS numero,
                     pg_catalog.bool_and(
                         a.grantor = propietario AND a.grantee = propietario
                         AND NOT a.is_grantable
                         AND a.privilege_type = ANY(ARRAY[
                             'INSERT', 'SELECT', 'UPDATE', 'DELETE',
                             'TRUNCATE', 'REFERENCES', 'TRIGGER', 'MAINTAIN'
                         ])
                     ) AS propios
                FROM pg_catalog.pg_class AS c
                LEFT JOIN LATERAL pg_catalog.aclexplode(c.relacl) AS a ON true
               WHERE c.oid IN (versiones, actual)
               GROUP BY c.oid
          ) AS acl WHERE numero <> 8 OR propios IS NOT TRUE
    ) OR (
        SELECT pg_catalog.count(*) <> 1 OR NOT pg_catalog.bool_and(
                   a.grantor = propietario AND a.grantee = propietario
                   AND a.privilege_type = 'EXECUTE' AND NOT a.is_grantable
               )
          FROM pg_catalog.pg_proc AS p
          LEFT JOIN LATERAL pg_catalog.aclexplode(p.proacl) AS a ON true
         WHERE p.oid = validador
    ) OR EXISTS (
        SELECT 1
          FROM (
              SELECT t.oid, pg_catalog.count(a.*) AS numero,
                     pg_catalog.bool_and(
                         a.grantor = propietario AND a.grantee = propietario
                         AND a.privilege_type = 'USAGE'
                         AND NOT a.is_grantable
                     ) AS propios
                FROM pg_catalog.pg_type AS t
                LEFT JOIN LATERAL pg_catalog.aclexplode(t.typacl) AS a ON true
               WHERE t.typrelid IN (versiones, actual)
               GROUP BY t.oid
          ) AS acl WHERE numero <> 1 OR propios IS NOT TRUE
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_attribute
         WHERE attrelid IN (versiones, actual) AND attnum > 0
           AND NOT attisdropped AND attacl IS NOT NULL
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada rechazada: ACL no exacta';
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_inherits
         WHERE inhrelid IN (versiones, actual)
            OR inhparent IN (versiones, actual)
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_rewrite
         WHERE ev_class IN (versiones, actual)
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_publication_rel
         WHERE prrelid IN (versiones, actual)
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_publication WHERE puballtables
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_publication_namespace
         WHERE pnnspid = esquema
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_subscription_rel
         WHERE srrelid IN (versiones, actual)
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_statistic_ext
         WHERE stxrelid IN (versiones, actual)
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_description
         WHERE (classoid = 'pg_class'::regclass
                AND objoid = ANY(ARRAY[
                    versiones, actual, toast_versiones, toast_actual
                ]::oid[] || indices))
            OR (classoid = 'pg_proc'::regclass AND objoid = validador)
            OR (classoid = 'pg_type'::regclass AND objoid = ANY(tipos))
            OR (classoid = 'pg_constraint'::regclass
                AND objoid = ANY(restricciones))
            OR (classoid = 'pg_trigger'::regclass
                AND objoid = ANY(disparadores))
            OR (classoid = 'pg_policy'::regclass
                AND objoid = ANY(politicas))
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_seclabel
         WHERE (classoid = 'pg_class'::regclass
                AND objoid = ANY(ARRAY[
                    versiones, actual, toast_versiones, toast_actual
                ]::oid[] || indices))
            OR (classoid = 'pg_proc'::regclass AND objoid = validador)
            OR (classoid = 'pg_type'::regclass AND objoid = ANY(tipos))
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_init_privs
         WHERE (classoid = 'pg_class'::regclass
                AND objoid = ANY(ARRAY[
                    versiones, actual, toast_versiones, toast_actual
                ]::oid[] || indices))
            OR (classoid = 'pg_proc'::regclass AND objoid = validador)
            OR (classoid = 'pg_type'::regclass AND objoid = ANY(tipos))
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada rechazada: metadatos o publicacion ajenos';
    END IF;

    -- Toda dependencia entrante debe pertenecer a una pieza nativa ya
    -- inventariada. Una vista, funcion, FK o 000004 queda fuera y se deniega
    -- antes de iniciar los drops. Las dependencias de extension se prohíben.
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_depend AS d
         WHERE (
             (d.refclassid = 'pg_class'::regclass
              AND d.refobjid = ANY(ARRAY[
                  versiones, actual, toast_versiones, toast_actual
              ]::oid[] || indices))
             OR (d.refclassid = 'pg_proc'::regclass
                 AND d.refobjid = validador)
             OR (d.refclassid = 'pg_type'::regclass
                 AND d.refobjid = ANY(tipos))
         ) AND NOT (
             (d.classid = 'pg_class'::regclass
              AND d.objid = ANY(ARRAY[
                  versiones, actual, toast_versiones, toast_actual
              ]::oid[] || indices))
             OR (d.classid = 'pg_type'::regclass
                 AND d.objid = ANY(tipos))
             OR (d.classid = 'pg_constraint'::regclass
                 AND d.objid = ANY(restricciones))
             OR (d.classid = 'pg_trigger'::regclass
                 AND d.objid = ANY(disparadores))
             OR (d.classid = 'pg_policy'::regclass
                 AND d.objid = ANY(politicas))
         )
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_depend AS d
         WHERE d.deptype IN ('e','x')
           AND ((d.classid = 'pg_class'::regclass
                 AND d.objid = ANY(ARRAY[
                     versiones, actual, toast_versiones, toast_actual
                 ]::oid[] || indices))
             OR (d.classid = 'pg_proc'::regclass AND d.objid = validador)
             OR (d.classid = 'pg_type'::regclass AND d.objid = ANY(tipos)))
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada rechazada: dependencia consumidora o extension';
    END IF;
END
$inventario$;

-- Los drops son contiguos al inventario inmovilizado y todos usan RESTRICT.
DROP TRIGGER avanzar_generacion_punteros_actuales_v2
    ON vec_contexto_actor_v1.organizacion_actual RESTRICT;
DROP TRIGGER serializar_mutacion_punteros_actuales_v2
    ON vec_contexto_actor_v1.organizacion_actual RESTRICT;
DROP TRIGGER puntero_actual_no_truncable_v2
    ON vec_contexto_actor_v1.organizacion_actual RESTRICT;
DROP POLICY acceso_propietario_exacto
    ON vec_contexto_actor_v1.organizacion_actual RESTRICT;
DROP TRIGGER historia_no_truncable
    ON vec_contexto_actor_v1.organizacion_versiones RESTRICT;
DROP TRIGGER historia_inmutable
    ON vec_contexto_actor_v1.organizacion_versiones RESTRICT;
DROP POLICY acceso_propietario_exacto
    ON vec_contexto_actor_v1.organizacion_versiones RESTRICT;
DROP TABLE vec_contexto_actor_v1.organizacion_actual RESTRICT;
DROP TABLE vec_contexto_actor_v1.organizacion_versiones RESTRICT;
DROP FUNCTION vec_contexto_actor_v1.organizacion_ref_valida(text) RESTRICT;

DO $postcondiciones$
BEGIN
    IF pg_catalog.to_regclass(
           'vec_contexto_actor_v1.organizacion_actual'
       ) IS NOT NULL OR pg_catalog.to_regclass(
           'vec_contexto_actor_v1.organizacion_versiones'
       ) IS NOT NULL OR pg_catalog.to_regprocedure(
           'vec_contexto_actor_v1.organizacion_ref_valida(text)'
       ) IS NOT NULL OR pg_catalog.to_regclass(
           'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'
       ) IS NULL OR pg_catalog.to_regprocedure(
           'vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2()'
       ) IS NULL OR pg_catalog.to_regprocedure(
           'vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2()'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada de organizacion corporativa V1 incompleta';
    END IF;
END
$postcondiciones$;

COMMIT;
