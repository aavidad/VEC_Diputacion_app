-- Retirada gobernada del vinculo corporativo RRHH C2.2-B.
-- Solo retira una instalacion vacia, exacta y sin consumidores posteriores.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

DO $precondiciones_minimas$
BEGIN
    IF pg_catalog.current_setting(
         'vec.confirmar_retirada_vinculo_corporativo_rrhh_v1',true
       ) IS DISTINCT FROM 'RETIRAR_VINCULO_CORPORATIVO_RRHH_V1' THEN
        RAISE EXCEPTION USING ERRCODE='55000',
          MESSAGE='retirada de vinculo corporativo RRHH V1 requiere confirmacion explicita';
    ELSIF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                       WHERE rolname=current_user AND rolsuper) THEN
        RAISE EXCEPTION USING ERRCODE='42501',
          MESSAGE='retirada de vinculo corporativo RRHH V1 requiere superusuario';
    ELSIF pg_catalog.current_setting('server_version_num')::integer < 180000
       OR pg_catalog.current_setting('server_version_num')::integer >= 190000 THEN
        RAISE EXCEPTION USING ERRCODE='55000',
          MESSAGE='retirada de vinculo corporativo RRHH V1 requiere PostgreSQL 18';
    END IF;
END
$precondiciones_minimas$;

SELECT pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
    'vec_contexto_actor_v1:migracion:acreditacion_uso:v2',0));
SELECT pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
    'vec_contexto_actor_v1:organizacion-corporativa-rrhh:v1',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contexto_actor_v1:vinculo-corporativo-rrhh:v1',0));

LOCK TABLE vec_contexto_actor_v1.vinculo_corporativo_actual,
    vec_contexto_actor_v1.vinculo_corporativo_versiones IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_contexto_actor_v1.procedencias,
    vec_contexto_actor_v1.proyeccion_cuenta_versiones,
    vec_contexto_actor_v1.persona_versiones IN SHARE MODE;
LOCK TABLE vec_contexto_actor_v1.perfil_versiones,
    vec_contexto_actor_v1.vinculo_contexto_versiones IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_contexto_actor_v1.organizacion_versiones,
    vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 IN SHARE MODE;
LOCK TABLE pg_catalog.pg_authid, pg_catalog.pg_auth_members,
    pg_catalog.pg_db_role_setting, pg_catalog.pg_database,
    pg_catalog.pg_class, pg_catalog.pg_attribute, pg_catalog.pg_attrdef,
    pg_catalog.pg_index, pg_catalog.pg_namespace, pg_catalog.pg_language,
    pg_catalog.pg_collation, pg_catalog.pg_proc, pg_catalog.pg_type,
    pg_catalog.pg_default_acl, pg_catalog.pg_description,
    pg_catalog.pg_seclabel, pg_catalog.pg_init_privs,
    pg_catalog.pg_depend, pg_catalog.pg_shdepend,
    pg_catalog.pg_constraint, pg_catalog.pg_trigger, pg_catalog.pg_policy,
    pg_catalog.pg_inherits, pg_catalog.pg_rewrite, pg_catalog.pg_am,
    pg_catalog.pg_tablespace, pg_catalog.pg_publication,
    pg_catalog.pg_publication_namespace, pg_catalog.pg_publication_rel,
    pg_catalog.pg_subscription_rel, pg_catalog.pg_statistic_ext IN SHARE MODE;

DO $inventario$
DECLARE
    propietario constant oid := 'vec_contexto_actor_v1_propietario'::regrole;
    migrador constant oid := 'vec_contexto_actor_v1_migrador'::regrole;
    runtime constant oid := 'vec_contexto_actor_v1_runtime'::regrole;
    selector constant oid :=
      'vec_contexto_actor_corporativo_rrhh_selector'::regrole;
    esquema constant oid := 'vec_contexto_actor_v1'::regnamespace;
    versiones constant oid := 'vec_contexto_actor_v1.vinculo_corporativo_versiones'::regclass;
    actual constant oid := 'vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass;
    clases oid[];
    restricciones oid[];
    disparadores oid[];
    politicas oid[];
    tipos oid[];
    observado text;
BEGIN
    IF pg_catalog.current_setting(
         'vec.confirmar_retirada_vinculo_corporativo_rrhh_v1',true
       ) IS DISTINCT FROM 'RETIRAR_VINCULO_CORPORATIVO_RRHH_V1'
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_authid
                       WHERE rolname=current_user AND rolsuper)
       OR EXISTS (SELECT 1 FROM vec_contexto_actor_v1.vinculo_corporativo_versiones)
       OR EXISTS (SELECT 1 FROM vec_contexto_actor_v1.vinculo_corporativo_actual) THEN
        RAISE EXCEPTION USING ERRCODE='55000',
          MESSAGE='retirada de vinculo corporativo RRHH V1 rechazada por evidencia';
    END IF;

    IF (SELECT pg_catalog.count(*) FROM pg_catalog.pg_authid r
         WHERE r.oid IN (propietario,migrador,runtime,selector)
           AND NOT r.rolcanlogin AND NOT r.rolsuper AND NOT r.rolinherit
           AND NOT r.rolcreatedb AND NOT r.rolcreaterole
           AND NOT r.rolreplication AND NOT r.rolbypassrls
           AND r.rolconnlimit=-1 AND r.rolpassword IS NULL
           AND r.rolvaliduntil IS NULL) <> 4
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members m
            WHERE m.roleid IN (propietario,migrador,selector)
               OR m.member IN (propietario,migrador,runtime,selector)) <> 1
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
            WHERE m.roleid=propietario AND m.member=migrador
              AND NOT m.admin_option AND NOT m.inherit_option AND m.set_option)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
            JOIN pg_catalog.pg_authid l ON l.oid=m.member
            WHERE m.roleid=runtime AND (m.admin_option OR NOT m.inherit_option
              OR m.set_option OR NOT l.rolcanlogin OR l.rolsuper
              OR NOT l.rolinherit OR l.rolcreatedb OR l.rolcreaterole
              OR l.rolreplication OR l.rolbypassrls
              OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members x
                   WHERE x.member=m.member)<>1
              OR EXISTS (SELECT 1 FROM pg_catalog.pg_db_role_setting s
                          WHERE s.setrole=m.member)
              OR NOT vec_contexto_actor_v1.privilegios_efectivos_runtime_minimos(
                m.member,(SELECT oid FROM pg_catalog.pg_database
                           WHERE datname=pg_catalog.current_database()),esquema,
                ARRAY[
                  'vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()'::regprocedure,
                  'vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)'::regprocedure,
                  'vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)'::regprocedure
                ])))
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_db_role_setting s
                   WHERE s.setrole IN (propietario,migrador,runtime,selector))
       OR pg_catalog.shobj_description(selector,'pg_authid') IS DISTINCT FROM
          'vec_contexto_actor_v1:rol-contexto-corporativo-rrhh-selector:v1'
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_authid r
                   WHERE r.oid IN (propietario,migrador,runtime)
                     AND pg_catalog.shobj_description(r.oid,'pg_authid') IS NOT NULL)
       OR NOT pg_catalog.has_database_privilege(selector,
            (SELECT oid FROM pg_catalog.pg_database
              WHERE datname=pg_catalog.current_database()),'CONNECT')
       OR pg_catalog.has_database_privilege(selector,
            (SELECT oid FROM pg_catalog.pg_database
              WHERE datname=pg_catalog.current_database()),'CREATE,TEMPORARY')
       OR pg_catalog.has_schema_privilege(selector,esquema,'USAGE')
       OR pg_catalog.has_schema_privilege(selector,esquema,'CREATE') THEN
        RAISE EXCEPTION USING ERRCODE='55000',
          MESSAGE='retirada rechazada: roles o privilegios efectivos hostiles';
    END IF;

    SELECT pg_catalog.array_agg(oid ORDER BY oid) INTO restricciones
      FROM pg_catalog.pg_constraint WHERE conrelid IN (versiones,actual)
        OR (connamespace=esquema AND conname IN
          ('perfil_versiones_persona_uq','vinculo_contexto_versiones_actor_uq'));
    SELECT pg_catalog.array_agg(DISTINCT objeto ORDER BY objeto) INTO clases
      FROM (
        SELECT versiones objeto UNION ALL SELECT actual
        UNION ALL SELECT reltoastrelid FROM pg_catalog.pg_class
          WHERE oid IN (versiones,actual)
        UNION ALL SELECT indexrelid FROM pg_catalog.pg_index
          WHERE indrelid IN (versiones,actual,
            'vec_contexto_actor_v1.perfil_versiones'::regclass,
            'vec_contexto_actor_v1.vinculo_contexto_versiones'::regclass)
          AND (indrelid IN (versiones,actual) OR indexrelid::regclass::text IN
            ('vec_contexto_actor_v1.perfil_versiones_persona_uq',
             'vec_contexto_actor_v1.vinculo_contexto_versiones_actor_uq'))
        UNION ALL SELECT i.indexrelid FROM pg_catalog.pg_index i
          JOIN pg_catalog.pg_class p ON p.oid=i.indrelid
         WHERE p.oid IN (SELECT reltoastrelid FROM pg_catalog.pg_class
                          WHERE oid IN (versiones,actual))
      ) q;
    SELECT pg_catalog.array_agg(oid ORDER BY oid) INTO disparadores
      FROM pg_catalog.pg_trigger WHERE tgrelid IN (versiones,actual)
         OR tgconstraint=ANY(restricciones);
    SELECT pg_catalog.array_agg(oid ORDER BY oid) INTO politicas
      FROM pg_catalog.pg_policy WHERE polrelid IN (versiones,actual);
    SELECT pg_catalog.array_agg(oid ORDER BY oid) INTO tipos
      FROM pg_catalog.pg_type WHERE typnamespace=esquema AND (
        typrelid IN (versiones,actual) OR typelem IN
          ((SELECT reltype FROM pg_catalog.pg_class WHERE oid=versiones),
           (SELECT reltype FROM pg_catalog.pg_class WHERE oid=actual)));

    IF (SELECT pg_catalog.count(*) FROM pg_catalog.pg_class
         WHERE oid IN (versiones,actual) AND relkind='r' AND relpersistence='p'
           AND relowner=propietario AND relrowsecurity AND relforcerowsecurity
           AND reloptions IS NULL) <> 2
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_policy
            WHERE polrelid IN (versiones,actual)
              AND polname='acceso_propietario_exacto' AND polcmd='*'
              AND polpermissive AND polroles=ARRAY[propietario]::oid[]
              AND pg_catalog.pg_get_expr(polqual,polrelid)=
                '(CURRENT_USER = ''vec_contexto_actor_v1_propietario''::name)'
              AND pg_catalog.pg_get_expr(polwithcheck,polrelid)=
                '(CURRENT_USER = ''vec_contexto_actor_v1_propietario''::name)') <> 2
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_constraint
            WHERE conrelid=versiones) <> 50
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_constraint
            WHERE conrelid=actual) <> 10
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_constraint
            WHERE connamespace=esquema AND conname IN
             ('perfil_versiones_persona_uq','vinculo_contexto_versiones_actor_uq')) <> 2
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_constraint
            WHERE conrelid IN (versiones,actual) AND contype='f'
              AND confmatchtype='f' AND confupdtype='a' AND confdeltype='a'
              AND NOT condeferrable AND NOT condeferred AND convalidated) <> 7
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_index
            WHERE indrelid IN (versiones,actual)) <> 3
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_trigger
            WHERE tgrelid IN (versiones,actual) AND NOT tgisinternal) <> 5
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_trigger
            WHERE tgrelid IN (versiones,actual) AND tgisinternal) <> 16
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_index i
            JOIN pg_catalog.pg_class x ON x.oid=i.indexrelid
            JOIN pg_catalog.pg_am am ON am.oid=x.relam
            WHERE i.indrelid IN (versiones,actual,
              'vec_contexto_actor_v1.perfil_versiones'::regclass,
              'vec_contexto_actor_v1.vinculo_contexto_versiones'::regclass)
              AND x.relname IN ('vinculo_corporativo_versiones_pk',
                'vinculo_corporativo_versiones_actual_uq',
                'vinculo_corporativo_actual_pk','perfil_versiones_persona_uq',
                'vinculo_contexto_versiones_actor_uq')
              AND x.relowner=propietario AND x.reltablespace=0
              AND x.reloptions IS NULL AND am.amname='btree') <> 5
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_attribute a
            WHERE a.attrelid IN (versiones,actual) AND a.attnum>0
              AND NOT a.attisdropped AND (a.attacl IS NOT NULL OR a.atthasdef
                OR a.attstorage<>(SELECT typstorage FROM pg_catalog.pg_type
                                   WHERE oid=a.atttypid)
                OR a.attcompression<>''))
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_type t
            WHERE t.oid IN (
              (SELECT reltype FROM pg_catalog.pg_class WHERE oid=versiones),
              (SELECT reltype FROM pg_catalog.pg_class WHERE oid=actual),
              (SELECT typarray FROM pg_catalog.pg_type WHERE typrelid=versiones),
              (SELECT typarray FROM pg_catalog.pg_type WHERE typrelid=actual))
              AND t.typowner=propietario AND (
                (t.typrelid IN (versiones,actual) AND NOT EXISTS (
                  SELECT 1 FROM pg_catalog.aclexplode(coalesce(t.typacl,
                    pg_catalog.acldefault('T',t.typowner))) a
                   WHERE a.grantee<>propietario))
                OR (t.typrelid=0 AND t.typacl IS NULL AND t.typelem IN (
                  (SELECT reltype FROM pg_catalog.pg_class WHERE oid=versiones),
                  (SELECT reltype FROM pg_catalog.pg_class WHERE oid=actual))))
              ) <> 4
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_class c
            JOIN pg_catalog.pg_am am ON am.oid=c.relam
            WHERE c.oid IN (versiones,actual) AND c.relowner=propietario
              AND c.reltablespace=0 AND c.reloptions IS NULL
              AND am.amname='heap' AND c.reltoastrelid<>0) <> 2
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_class t
            JOIN pg_catalog.pg_class p ON p.reltoastrelid=t.oid
            JOIN pg_catalog.pg_am am ON am.oid=t.relam
            WHERE p.oid IN (versiones,actual) AND t.relkind='t'
              AND t.relowner=propietario AND t.reltablespace=0
              AND t.reloptions IS NULL AND am.amname='heap'
              AND NOT EXISTS (SELECT 1 FROM pg_catalog.aclexplode(
                coalesce(t.relacl,pg_catalog.acldefault('r',t.relowner))) a
                WHERE a.grantee<>propietario)) <> 2
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_index i
            JOIN pg_catalog.pg_class t ON t.oid=i.indrelid
            JOIN pg_catalog.pg_class x ON x.oid=i.indexrelid
            JOIN pg_catalog.pg_am am ON am.oid=x.relam
            WHERE t.oid IN ((SELECT reltoastrelid FROM pg_catalog.pg_class WHERE oid=versiones),
                            (SELECT reltoastrelid FROM pg_catalog.pg_class WHERE oid=actual))
              AND i.indisunique AND i.indisprimary AND i.indisvalid AND i.indisready
              AND x.relowner=propietario AND x.reltablespace=0
              AND x.reloptions IS NULL AND am.amname='btree') <> 2
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c,
            LATERAL pg_catalog.aclexplode(coalesce(c.relacl,
              pg_catalog.acldefault('r',c.relowner))) a
            WHERE c.oid IN (versiones,actual) AND a.grantee<>propietario)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_attribute a,
            LATERAL pg_catalog.aclexplode(a.attacl) x
            WHERE a.attrelid IN (versiones,actual) AND a.attnum>0
              AND NOT a.attisdropped AND x.grantee<>propietario) THEN
        RAISE EXCEPTION USING ERRCODE='55000',
          MESSAGE='retirada rechazada: forma de vinculo corporativo no exacta';
    END IF;

    IF (SELECT pg_catalog.string_agg(pg_catalog.format('%s|%s|%s|%s',
          attnum,attname,pg_catalog.format_type(atttypid,atttypmod),attnotnull),
          ';' ORDER BY attnum) FROM pg_catalog.pg_attribute
         WHERE attrelid=versiones AND attnum>0 AND NOT attisdropped)
       IS DISTINCT FROM
       '1|vinculo_corporativo_ref|text|t;2|version|numeric(20,0)|t;3|cuenta_ref|text|t;4|cuenta_version|numeric(20,0)|t;5|persona_ref|text|t;6|persona_version|numeric(20,0)|t;7|perfil_ref|text|t;8|perfil_version|numeric(20,0)|t;9|vinculo_contexto_ref|text|t;10|vinculo_contexto_version|numeric(20,0)|t;11|organizacion_ref|text|t;12|organizacion_version|numeric(20,0)|t;13|organizacion_procedencia_ref|text|t;14|organizacion_procedencia_version|numeric(20,0)|t;15|organizacion_procedencia_huella_sha256|text|t;16|organizacion_procedencia_autoridad|text|t;17|superficie|text|t;18|uso|text|t;19|procedencia_ref|text|t;20|procedencia_version|numeric(20,0)|t;21|procedencia_huella_sha256|text|t;22|procedencia_autoridad|text|t;23|estado|text|t;24|vigente_desde|timestamp(6) with time zone|t;25|vigente_hasta|timestamp(6) with time zone|t'
       OR (SELECT pg_catalog.string_agg(pg_catalog.format('%s|%s|%s|%s',
          attnum,attname,pg_catalog.format_type(atttypid,atttypmod),attnotnull),
          ';' ORDER BY attnum) FROM pg_catalog.pg_attribute
         WHERE attrelid=actual AND attnum>0 AND NOT attisdropped)
       IS DISTINCT FROM
       '1|cuenta_ref|text|t;2|superficie|text|t;3|uso|text|t;4|vinculo_corporativo_ref|text|t;5|version|numeric(20,0)|t' THEN
        RAISE EXCEPTION USING ERRCODE='55000',
          MESSAGE='retirada rechazada: columnas de vinculo corporativo no exactas';
    END IF;

    -- Huella simbolica de todo el esquema: nombres y definiciones, nunca OID.
    WITH elementos AS (
      SELECT pg_catalog.format('nsp|%s|%s|%s|%s',n.nspname,
               pg_catalog.pg_get_userbyid(n.nspowner),coalesce(n.nspacl::text,''),
               coalesce(pg_catalog.obj_description(n.oid,'pg_namespace'),'')) e
        FROM pg_catalog.pg_namespace n WHERE n.oid=esquema
      UNION ALL
      SELECT pg_catalog.format('rel|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',c.relname,c.relkind,
               c.relpersistence,
               pg_catalog.pg_get_userbyid(c.relowner),c.relrowsecurity,
               c.relforcerowsecurity,coalesce(c.relacl::text,''),
               coalesce(am.amname,''),coalesce(s.spcname,''),
               coalesce(c.reloptions::text,''),
               coalesce(pg_catalog.obj_description(c.oid,'pg_class'),'')) AS e
        FROM pg_catalog.pg_class c
        LEFT JOIN pg_catalog.pg_am am ON am.oid=c.relam
        LEFT JOIN pg_catalog.pg_tablespace s ON s.oid=c.reltablespace
       WHERE c.relnamespace=esquema
         AND c.relkind IN ('r','v','m','f','p','S')
      UNION ALL
      SELECT pg_catalog.format('col|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
               c.relname,a.attnum,a.attname,
               pg_catalog.format_type(a.atttypid,a.atttypmod),a.attnotnull,
               a.attidentity,a.attgenerated,a.attstorage,a.attcompression,
               a.attcollation::regcollation::text,coalesce(a.attacl::text,''),
               coalesce(pg_catalog.pg_get_expr(d.adbin,d.adrelid),''),
               coalesce(pg_catalog.col_description(a.attrelid,a.attnum),''))
        FROM pg_catalog.pg_attribute a JOIN pg_catalog.pg_class c ON c.oid=a.attrelid
        LEFT JOIN pg_catalog.pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum
       WHERE c.relnamespace=esquema AND a.attnum>0 AND NOT a.attisdropped
      UNION ALL
      SELECT pg_catalog.format('con|%s|%s|%s|%s|%s|%s|%s|%s',c.conrelid::regclass::text,
               c.conname,c.contype,c.condeferrable,c.condeferred,c.convalidated,
               pg_catalog.pg_get_constraintdef(c.oid,false),
               coalesce(pg_catalog.obj_description(c.oid,'pg_constraint'),''))
        FROM pg_catalog.pg_constraint c WHERE c.connamespace=esquema
      UNION ALL
      SELECT pg_catalog.format('idx|%s|%s|%s|%s|%s|%s|%s|%s',t.relname,i.relname,
               pg_catalog.pg_get_userbyid(i.relowner),am.amname,
               coalesce(s.spcname,''),coalesce(i.reloptions::text,''),
               coalesce(pg_catalog.obj_description(i.oid,'pg_class'),''),
               pg_catalog.pg_get_indexdef(i.oid))
        FROM pg_catalog.pg_index x JOIN pg_catalog.pg_class t ON t.oid=x.indrelid
        JOIN pg_catalog.pg_class i ON i.oid=x.indexrelid
        JOIN pg_catalog.pg_am am ON am.oid=i.relam
        LEFT JOIN pg_catalog.pg_tablespace s ON s.oid=i.reltablespace
       WHERE t.relnamespace=esquema
      UNION ALL
      SELECT pg_catalog.format('fun|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
               p.proname,pg_catalog.pg_get_function_identity_arguments(p.oid),
               pg_catalog.pg_get_function_result(p.oid),l.lanname,
               pg_catalog.pg_get_userbyid(p.proowner),p.provolatile,p.proparallel,
               p.prosecdef,p.proisstrict,p.proleakproof,coalesce(p.proacl::text,''),
               pg_catalog.pg_get_functiondef(p.oid),
               coalesce(pg_catalog.obj_description(p.oid,'pg_proc'),''))
        FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_language l ON l.oid=p.prolang
       WHERE p.pronamespace=esquema
      UNION ALL
      SELECT pg_catalog.format('typ|%s|%s|%s|%s|%s|%s|%s',t.typname,t.typtype,
               t.typcategory,pg_catalog.pg_get_userbyid(t.typowner),
               coalesce(e.typname,''),coalesce(t.typacl::text,''),
               coalesce(pg_catalog.obj_description(t.oid,'pg_type'),''))
        FROM pg_catalog.pg_type t LEFT JOIN pg_catalog.pg_type e ON e.oid=t.typelem
       WHERE t.typnamespace=esquema
      UNION ALL
      SELECT pg_catalog.format('defacl|%s|%s|%s|%s',
               pg_catalog.pg_get_userbyid(d.defaclrole),
               coalesce(n.nspname,''),d.defaclobjtype,d.defaclacl::text)
        FROM pg_catalog.pg_default_acl d LEFT JOIN pg_catalog.pg_namespace n
          ON n.oid=d.defaclnamespace
       WHERE d.defaclnamespace=esquema OR d.defaclrole=propietario
      UNION ALL
      SELECT pg_catalog.format('toast|%s|%s|%s|%s|%s|%s|%s|%s',p.relname,
               pg_catalog.pg_get_userbyid(t.relowner),tam.amname,coalesce(ts.spcname,''),
               coalesce(t.relacl::text,''),pg_catalog.pg_get_userbyid(x.relowner),
               xam.amname,i.indkey::text)
        FROM pg_catalog.pg_class p JOIN pg_catalog.pg_class t ON t.oid=p.reltoastrelid
        JOIN pg_catalog.pg_am tam ON tam.oid=t.relam
        JOIN pg_catalog.pg_index i ON i.indrelid=t.oid
        JOIN pg_catalog.pg_class x ON x.oid=i.indexrelid
        JOIN pg_catalog.pg_am xam ON xam.oid=x.relam
        LEFT JOIN pg_catalog.pg_tablespace ts ON ts.oid=t.reltablespace
       WHERE p.relnamespace=esquema
      UNION ALL
      SELECT pg_catalog.format('pol|%s|%s|%s|%s|%s|%s|%s|%s',p.polrelid::regclass::text,
               p.polname,p.polcmd,p.polpermissive,p.polroles::text,
               pg_catalog.pg_get_expr(p.polqual,p.polrelid),
               pg_catalog.pg_get_expr(p.polwithcheck,p.polrelid),
               coalesce(pg_catalog.obj_description(p.oid,'pg_policy'),''))
        FROM pg_catalog.pg_policy p JOIN pg_catalog.pg_class c ON c.oid=p.polrelid
       WHERE c.relnamespace=esquema
      UNION ALL
      SELECT pg_catalog.format('trg|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
               t.tgrelid::regclass::text,
               CASE WHEN t.tgisinternal THEN coalesce(k.conname,'<interno>')
                    ELSE t.tgname END,t.tgtype,t.tgenabled,t.tgisinternal,
               t.tgfoid::regprocedure::text,t.tgnargs,
               pg_catalog.encode(t.tgargs,'hex'),t.tgattr::text,
               coalesce(pg_catalog.pg_get_expr(t.tgqual,t.tgrelid,false),''),
               coalesce(pg_catalog.obj_description(t.oid,'pg_trigger'),''))
        FROM pg_catalog.pg_trigger t JOIN pg_catalog.pg_class c ON c.oid=t.tgrelid
        LEFT JOIN pg_catalog.pg_constraint k ON k.oid=t.tgconstraint
       WHERE c.relnamespace=esquema
    )
    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
             pg_catalog.string_agg(e,E'\n' ORDER BY e),'UTF8')),'hex')
      INTO observado FROM elementos;
    IF observado IS DISTINCT FROM 'b4ea9332a2ac5cf86359abfb7c698fbb224b807874c06a0c0e3acc4e8a423e1d' THEN
        RAISE EXCEPTION USING ERRCODE='55000',
          MESSAGE='retirada rechazada: manifiesto simbolico no acreditado',
          DETAIL='huella observada '||coalesce(observado,'<ausente>');
    END IF;

    IF EXISTS (SELECT 1 FROM pg_catalog.pg_publication WHERE puballtables)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_publication_namespace
                   WHERE pnnspid=esquema)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_publication_rel
                   WHERE prrelid IN (versiones,actual))
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_subscription_rel
                   WHERE srrelid IN (versiones,actual))
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_statistic_ext
                   WHERE stxrelid IN (versiones,actual))
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_inherits
                   WHERE inhrelid IN (versiones,actual) OR inhparent IN (versiones,actual))
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_description d
                   WHERE (d.classoid='pg_class'::regclass AND d.objoid IN (
                     versiones,actual,
                     (SELECT reltoastrelid FROM pg_catalog.pg_class WHERE oid=versiones),
                     (SELECT reltoastrelid FROM pg_catalog.pg_class WHERE oid=actual)
                   )) OR (d.classoid='pg_class'::regclass AND d.objoid IN (
                     SELECT i.indexrelid FROM pg_catalog.pg_index i
                      WHERE i.indrelid IN (versiones,actual,
                        (SELECT reltoastrelid FROM pg_catalog.pg_class WHERE oid=versiones),
                        (SELECT reltoastrelid FROM pg_catalog.pg_class WHERE oid=actual))
                   )) OR (d.classoid='pg_class'::regclass AND d.objoid IN (
                     SELECT x.oid FROM pg_catalog.pg_class x
                      WHERE x.relnamespace=esquema AND x.relname IN
                        ('perfil_versiones_persona_uq','vinculo_contexto_versiones_actor_uq')
                   )) OR (d.classoid='pg_type'::regclass AND d.objoid IN (
                     (SELECT reltype FROM pg_catalog.pg_class WHERE oid=versiones),
                     (SELECT reltype FROM pg_catalog.pg_class WHERE oid=actual),
                     (SELECT typarray FROM pg_catalog.pg_type WHERE typrelid=versiones),
                     (SELECT typarray FROM pg_catalog.pg_type WHERE typrelid=actual)
                   )) OR (d.classoid='pg_constraint'::regclass AND EXISTS (
                     SELECT 1 FROM pg_catalog.pg_constraint c WHERE c.oid=d.objoid
                       AND (c.conrelid IN (versiones,actual) OR c.conname IN
                         ('perfil_versiones_persona_uq','vinculo_contexto_versiones_actor_uq'))
                   )) OR (d.classoid='pg_trigger'::regclass AND EXISTS (
                     SELECT 1 FROM pg_catalog.pg_trigger t WHERE t.oid=d.objoid
                       AND t.tgrelid IN (versiones,actual))))
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_seclabel s
                   WHERE (s.classoid='pg_class'::regclass AND s.objoid IN
                          (versiones,actual))
                      OR (s.classoid='pg_type'::regclass AND s.objoid IN (
                        (SELECT reltype FROM pg_catalog.pg_class WHERE oid=versiones),
                        (SELECT reltype FROM pg_catalog.pg_class WHERE oid=actual),
                        (SELECT typarray FROM pg_catalog.pg_type WHERE typrelid=versiones),
                        (SELECT typarray FROM pg_catalog.pg_type WHERE typrelid=actual))))
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_init_privs p
                   WHERE (p.classoid='pg_class'::regclass AND p.objoid IN
                          (versiones,actual))
                      OR (p.classoid='pg_type'::regclass AND p.objoid IN (
                        (SELECT reltype FROM pg_catalog.pg_class WHERE oid=versiones),
                        (SELECT reltype FROM pg_catalog.pg_class WHERE oid=actual),
                        (SELECT typarray FROM pg_catalog.pg_type WHERE typrelid=versiones),
                        (SELECT typarray FROM pg_catalog.pg_type WHERE typrelid=actual))))
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_constraint
                   WHERE confrelid IN (versiones,actual)
                     AND conname<>'vinculo_corporativo_actual_version_fk')
       OR EXISTS (
          SELECT 1 FROM pg_catalog.pg_depend d
          JOIN pg_catalog.pg_rewrite r ON d.classid='pg_rewrite'::regclass
                                      AND d.objid=r.oid
           WHERE d.refclassid='pg_class'::regclass
             AND d.refobjid IN (versiones,actual)
             AND r.ev_class NOT IN (versiones,actual)
       ) THEN
        RAISE EXCEPTION USING ERRCODE='55000',
          MESSAGE='retirada rechazada: consumidor, metadato o publicacion hostil';
    END IF;

    IF EXISTS (
      SELECT 1 FROM pg_catalog.pg_depend d
       WHERE ((d.refclassid='pg_class'::regclass AND d.refobjid=ANY(clases))
           OR (d.refclassid='pg_type'::regclass AND d.refobjid=ANY(tipos))
           OR (d.refclassid='pg_constraint'::regclass
               AND d.refobjid=ANY(restricciones)))
         AND NOT ((d.classid='pg_class'::regclass AND d.objid=ANY(clases))
           OR (d.classid='pg_type'::regclass AND d.objid=ANY(tipos))
           OR (d.classid='pg_constraint'::regclass
               AND d.objid=ANY(restricciones))
           OR (d.classid='pg_trigger'::regclass AND d.objid=ANY(disparadores))
           OR (d.classid='pg_policy'::regclass AND d.objid=ANY(politicas)))
    ) OR EXISTS (
      SELECT 1 FROM pg_catalog.pg_depend d
       WHERE d.deptype IN ('e','x') AND (
         (d.classid='pg_class'::regclass AND d.objid=ANY(clases))
         OR (d.classid='pg_type'::regclass AND d.objid=ANY(tipos))
         OR (d.classid='pg_constraint'::regclass
             AND d.objid=ANY(restricciones))
         OR (d.classid='pg_trigger'::regclass AND d.objid=ANY(disparadores))
         OR (d.classid='pg_policy'::regclass AND d.objid=ANY(politicas)))
    ) OR EXISTS (
      SELECT 1 FROM pg_catalog.pg_shdepend d
       WHERE d.dbid=(SELECT oid FROM pg_catalog.pg_database
                      WHERE datname=pg_catalog.current_database())
         AND ((d.classid='pg_class'::regclass AND d.objid=ANY(clases))
           OR (d.classid='pg_type'::regclass AND d.objid=ANY(tipos))
           OR (d.classid='pg_constraint'::regclass
               AND d.objid=ANY(restricciones))
           OR (d.classid='pg_trigger'::regclass AND d.objid=ANY(disparadores))
           OR (d.classid='pg_policy'::regclass AND d.objid=ANY(politicas)))
         AND NOT (d.refclassid='pg_authid'::regclass
           AND d.refobjid=propietario AND (d.deptype='o'
             OR (d.classid='pg_policy'::regclass AND d.deptype='r')))
    ) THEN
      RAISE EXCEPTION USING ERRCODE='55000',
        MESSAGE='retirada rechazada: dependencia catalogal ajena';
    END IF;
END
$inventario$;

SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;

DROP TRIGGER avanzar_generacion_punteros_actuales_v2
    ON vec_contexto_actor_v1.vinculo_corporativo_actual RESTRICT;
DROP TRIGGER serializar_mutacion_punteros_actuales_v2
    ON vec_contexto_actor_v1.vinculo_corporativo_actual RESTRICT;
DROP TRIGGER puntero_actual_no_truncable_v2
    ON vec_contexto_actor_v1.vinculo_corporativo_actual RESTRICT;
DROP POLICY acceso_propietario_exacto
    ON vec_contexto_actor_v1.vinculo_corporativo_actual RESTRICT;
DROP TRIGGER historia_no_truncable
    ON vec_contexto_actor_v1.vinculo_corporativo_versiones RESTRICT;
DROP TRIGGER historia_inmutable
    ON vec_contexto_actor_v1.vinculo_corporativo_versiones RESTRICT;
DROP POLICY acceso_propietario_exacto
    ON vec_contexto_actor_v1.vinculo_corporativo_versiones RESTRICT;
DROP TABLE vec_contexto_actor_v1.vinculo_corporativo_actual RESTRICT;
DROP TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones RESTRICT;
ALTER TABLE vec_contexto_actor_v1.vinculo_contexto_versiones
    DROP CONSTRAINT vinculo_contexto_versiones_actor_uq RESTRICT;
ALTER TABLE vec_contexto_actor_v1.perfil_versiones
    DROP CONSTRAINT perfil_versiones_persona_uq RESTRICT;

RESET ROLE;
DO $postcondiciones$
BEGIN
    IF pg_catalog.to_regclass(
         'vec_contexto_actor_v1.vinculo_corporativo_versiones') IS NOT NULL
       OR pg_catalog.to_regclass(
         'vec_contexto_actor_v1.vinculo_corporativo_actual') IS NOT NULL
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_constraint
                   WHERE connamespace='vec_contexto_actor_v1'::regnamespace
                     AND conname IN ('perfil_versiones_persona_uq',
                                     'vinculo_contexto_versiones_actor_uq'))
       OR pg_catalog.to_regclass(
         'vec_contexto_actor_v1.organizacion_versiones') IS NULL
       OR pg_catalog.to_regclass(
         'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2') IS NULL THEN
        RAISE EXCEPTION USING ERRCODE='55000',
          MESSAGE='retirada de vinculo corporativo RRHH V1 incompleta';
    END IF;
END
$postcondiciones$;

COMMIT;
