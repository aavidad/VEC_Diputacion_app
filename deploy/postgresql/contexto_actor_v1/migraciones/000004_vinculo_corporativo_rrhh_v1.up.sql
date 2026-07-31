-- Historia y puntero opacos del vinculo corporativo RRHH C2.2-B.
-- Documento autonomo: no publica, selecciona ni concede acceso de negocio.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

DO $precondiciones_minimas$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'vinculo corporativo RRHH V1 requiere migracion superusuario';
    ELSIF pg_catalog.current_setting('server_version_num')::integer < 180000
       OR pg_catalog.current_setting('server_version_num')::integer >= 190000
       OR pg_catalog.current_setting('server_encoding') <> 'UTF8' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'vinculo corporativo RRHH V1 requiere PostgreSQL 18 y UTF8';
    END IF;
END
$precondiciones_minimas$;

-- Barreras globales A -> B -> C; ninguna decision de catalogo se toma antes.
SELECT pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
    'vec_contexto_actor_v1:migracion:acreditacion_uso:v2', 0));
SELECT pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
    'vec_contexto_actor_v1:organizacion-corporativa-rrhh:v1', 0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contexto_actor_v1:vinculo-corporativo-rrhh:v1', 0));

LOCK TABLE vec_contexto_actor_v1.procedencias,
    vec_contexto_actor_v1.proyeccion_cuenta_versiones,
    vec_contexto_actor_v1.persona_versiones IN SHARE MODE;
LOCK TABLE vec_contexto_actor_v1.perfil_versiones,
    vec_contexto_actor_v1.vinculo_contexto_versiones IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_contexto_actor_v1.organizacion_versiones,
    vec_contexto_actor_v1.organizacion_actual,
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

DO $acreditar_base_y_ausencia$
DECLARE
    propietario constant oid := 'vec_contexto_actor_v1_propietario'::regrole;
    migrador constant oid := 'vec_contexto_actor_v1_migrador'::regrole;
    runtime constant oid := 'vec_contexto_actor_v1_runtime'::regrole;
    selector constant oid :=
      'vec_contexto_actor_corporativo_rrhh_selector'::regrole;
    esquema constant oid := 'vec_contexto_actor_v1'::regnamespace;
    observado text;
BEGIN
    IF pg_catalog.current_setting('timezone') <> 'UTC'
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_authid
                       WHERE rolname=current_user AND rolsuper)
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_namespace
                       WHERE oid=esquema AND nspowner=propietario) THEN
        RAISE EXCEPTION USING ERRCODE='55000',
            MESSAGE='entorno de vinculo corporativo RRHH V1 no reacreditado';
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
          MESSAGE='roles o privilegios efectivos predecesores no acreditados';
    END IF;

    -- Sin proveedor MAC configurado, el contrato base exige cero etiquetas.
    -- Un superusuario/DBA comprometido queda fuera: es frontera ENS/operativa.
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_seclabel s
        WHERE (s.classoid='pg_namespace'::regclass AND s.objoid=esquema)
           OR (s.classoid='pg_proc'::regclass AND EXISTS (
                SELECT 1 FROM pg_catalog.pg_proc p
                 WHERE p.oid=s.objoid AND p.pronamespace=esquema))
           OR (s.classoid='pg_type'::regclass AND EXISTS (
                SELECT 1 FROM pg_catalog.pg_type t
                 WHERE t.oid=s.objoid AND t.typnamespace=esquema))
           OR (s.classoid='pg_class'::regclass AND EXISTS (
                SELECT 1 FROM pg_catalog.pg_class c
                 WHERE c.oid=s.objoid AND (c.relnamespace=esquema OR c.oid IN (
                   SELECT p.reltoastrelid FROM pg_catalog.pg_class p
                    WHERE p.relnamespace=esquema AND p.reltoastrelid<>0) OR c.oid IN (
                   SELECT i.indexrelid FROM pg_catalog.pg_class p
                   JOIN pg_catalog.pg_class t ON t.oid=p.reltoastrelid
                   JOIN pg_catalog.pg_index i ON i.indrelid=t.oid
                    WHERE p.relnamespace=esquema))))) THEN
        RAISE EXCEPTION USING ERRCODE='55000',
          MESSAGE='etiquetas de seguridad predecesoras no acreditadas';
    END IF;

    IF (SELECT pg_catalog.count(*) FROM pg_catalog.pg_class
         WHERE oid IN (
           'vec_contexto_actor_v1.procedencias'::regclass,
           'vec_contexto_actor_v1.proyeccion_cuenta_versiones'::regclass,
           'vec_contexto_actor_v1.persona_versiones'::regclass,
           'vec_contexto_actor_v1.perfil_versiones'::regclass,
           'vec_contexto_actor_v1.vinculo_contexto_versiones'::regclass,
           'vec_contexto_actor_v1.organizacion_versiones'::regclass,
           'vec_contexto_actor_v1.organizacion_actual'::regclass,
           'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass
         ) AND relkind='r' AND relpersistence='p' AND relowner=propietario
           AND relreplident='d') <> 8
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc
            WHERE oid IN (
              'vec_contexto_actor_v1.referencia_valida(text,text)'::regprocedure,
              'vec_contexto_actor_v1.procedencia_valida(text,numeric,text,text)'::regprocedure,
              'vec_contexto_actor_v1.organizacion_ref_valida(text)'::regprocedure,
              'vec_contexto_actor_v1.instante_valido(timestamptz)'::regprocedure,
              'vec_contexto_actor_v1.rechazar_mutacion_historia()'::regprocedure,
              'vec_contexto_actor_v1.rechazar_truncado()'::regprocedure,
              'vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2()'::regprocedure,
              'vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2()'::regprocedure
            ) AND proowner=propietario) <> 8 THEN
        RAISE EXCEPTION USING ERRCODE='55000',
            MESSAGE='dependencias de vinculo corporativo RRHH V1 no acreditadas';
    END IF;

    IF (SELECT pg_catalog.count(*) FROM pg_catalog.pg_attribute a
         WHERE a.attrelid IN (
           'vec_contexto_actor_v1.procedencias'::regclass,
           'vec_contexto_actor_v1.proyeccion_cuenta_versiones'::regclass,
           'vec_contexto_actor_v1.persona_versiones'::regclass,
           'vec_contexto_actor_v1.perfil_versiones'::regclass,
           'vec_contexto_actor_v1.vinculo_contexto_versiones'::regclass,
           'vec_contexto_actor_v1.organizacion_versiones'::regclass,
           'vec_contexto_actor_v1.organizacion_actual'::regclass,
           'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass)
           AND a.attnum>0 AND NOT a.attisdropped) <> 58
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c,
            LATERAL pg_catalog.aclexplode(coalesce(c.relacl,
              pg_catalog.acldefault('r',c.relowner))) a
            WHERE c.oid IN (
              'vec_contexto_actor_v1.procedencias'::regclass,
              'vec_contexto_actor_v1.proyeccion_cuenta_versiones'::regclass,
              'vec_contexto_actor_v1.persona_versiones'::regclass,
              'vec_contexto_actor_v1.perfil_versiones'::regclass,
              'vec_contexto_actor_v1.vinculo_contexto_versiones'::regclass,
              'vec_contexto_actor_v1.organizacion_versiones'::regclass,
              'vec_contexto_actor_v1.organizacion_actual'::regclass,
              'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass)
              AND a.grantee<>propietario)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_attribute a,
            LATERAL pg_catalog.aclexplode(a.attacl) x
            WHERE a.attrelid IN (
              'vec_contexto_actor_v1.procedencias'::regclass,
              'vec_contexto_actor_v1.proyeccion_cuenta_versiones'::regclass,
              'vec_contexto_actor_v1.persona_versiones'::regclass,
              'vec_contexto_actor_v1.perfil_versiones'::regclass,
              'vec_contexto_actor_v1.vinculo_contexto_versiones'::regclass,
              'vec_contexto_actor_v1.organizacion_versiones'::regclass,
              'vec_contexto_actor_v1.organizacion_actual'::regclass,
              'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass)
              AND a.attnum>0 AND NOT a.attisdropped AND x.grantee<>propietario)
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_trigger t
            WHERE t.tgrelid IN (
              'vec_contexto_actor_v1.procedencias'::regclass,
              'vec_contexto_actor_v1.proyeccion_cuenta_versiones'::regclass,
              'vec_contexto_actor_v1.persona_versiones'::regclass,
              'vec_contexto_actor_v1.perfil_versiones'::regclass,
              'vec_contexto_actor_v1.vinculo_contexto_versiones'::regclass,
              'vec_contexto_actor_v1.organizacion_versiones'::regclass,
              'vec_contexto_actor_v1.organizacion_actual'::regclass)
              AND NOT t.tgisinternal) <> 16
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_class c
            WHERE c.oid IN (
              'vec_contexto_actor_v1.organizacion_versiones'::regclass,
              'vec_contexto_actor_v1.organizacion_actual'::regclass)
              AND c.relrowsecurity AND c.relforcerowsecurity) <> 2
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_policy p
            WHERE p.polrelid IN (
              'vec_contexto_actor_v1.organizacion_versiones'::regclass,
              'vec_contexto_actor_v1.organizacion_actual'::regclass)
              AND p.polname='acceso_propietario_exacto' AND p.polpermissive
              AND p.polcmd='*' AND p.polroles=ARRAY[propietario]::oid[]
              AND pg_catalog.pg_get_expr(p.polqual,p.polrelid)=
                '(CURRENT_USER = ''vec_contexto_actor_v1_propietario''::name)'
              AND pg_catalog.pg_get_expr(p.polwithcheck,p.polrelid)=
                '(CURRENT_USER = ''vec_contexto_actor_v1_propietario''::name)') <> 2
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_constraint c
            WHERE c.conrelid='vec_contexto_actor_v1.organizacion_versiones'::regclass
              AND c.conname='organizacion_versiones_procedencia_uq'
              AND c.contype='u' AND c.conkey=ARRAY[1,2,3,4,5,6]::smallint[]
              AND c.convalidated AND NOT c.condeferrable AND NOT c.condeferred)
       OR (SELECT pg_catalog.count(*) FROM
            (VALUES
              ('referencia_valida(text,text)','sql','i',false),
              ('procedencia_valida(text,numeric,text,text)','sql','i',false),
              ('organizacion_ref_valida(text)','sql','i',false),
              ('instante_valido(timestamp with time zone)','sql','i',false),
              ('rechazar_mutacion_historia()','plpgsql','v',false),
              ('rechazar_truncado()','plpgsql','v',false),
              ('serializar_mutacion_punteros_actuales_v2()','plpgsql','v',true),
              ('avanzar_generacion_punteros_actuales_v2()','plpgsql','v',true)
            ) e(firma,lenguaje,volatilidad,definidor)
            JOIN pg_catalog.pg_proc p ON p.oid=
              pg_catalog.to_regprocedure('vec_contexto_actor_v1.'||e.firma)
            JOIN pg_catalog.pg_language l ON l.oid=p.prolang
           WHERE p.proowner=propietario AND l.lanname=e.lenguaje
             AND p.provolatile=e.volatilidad::"char" AND p.prosecdef=e.definidor
             AND p.proconfig=ARRAY['search_path=pg_catalog']::text[]) <> 8
       OR (SELECT pg_catalog.count(*) FROM
             vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
            WHERE control_id AND generacion>=0 AND pg_catalog.scale(generacion)=0
              AND vec_contexto_actor_v1.instante_valido(actualizada_en)) <> 1 THEN
        RAISE EXCEPTION USING ERRCODE='55000',
          MESSAGE='forma estructural del predecesor 000001..000003 no acreditada';
    END IF;

    WITH objetos AS (
      SELECT pg_catalog.format('nsp|%s|%s|%s|%s',n.nspname,
               pg_catalog.pg_get_userbyid(n.nspowner),coalesce(n.nspacl::text,''),
               coalesce(pg_catalog.obj_description(n.oid,'pg_namespace'),'')) e
        FROM pg_catalog.pg_namespace n WHERE n.oid=esquema
      UNION ALL
      SELECT pg_catalog.format('rel|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
               c.relname,c.relkind,c.relpersistence,
               pg_catalog.pg_get_userbyid(c.relowner),c.relrowsecurity,
               c.relforcerowsecurity,c.relreplident,coalesce(am.amname,''),
               coalesce(s.spcname,''),coalesce(c.reloptions::text,''),
               coalesce(pg_catalog.obj_description(c.oid,'pg_class'),''),
               coalesce(c.relacl::text,'')) e
        FROM pg_catalog.pg_class c LEFT JOIN pg_catalog.pg_am am ON am.oid=c.relam
        LEFT JOIN pg_catalog.pg_tablespace s ON s.oid=c.reltablespace
       WHERE c.relnamespace=esquema
      UNION ALL
      SELECT pg_catalog.format(
               'col|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',c.relname,a.attnum,
               a.attname,pg_catalog.format_type(a.atttypid,a.atttypmod),
               a.attnotnull,a.attidentity,a.attgenerated,a.attstorage,
               a.attcompression,a.attcollation::regcollation::text,
               coalesce(a.attacl::text,''),
               coalesce(pg_catalog.pg_get_expr(d.adbin,d.adrelid,false),''),
               coalesce(pg_catalog.col_description(a.attrelid,a.attnum),''))
        FROM pg_catalog.pg_attribute a JOIN pg_catalog.pg_class c ON c.oid=a.attrelid
        LEFT JOIN pg_catalog.pg_attrdef d ON (d.adrelid,d.adnum)=(a.attrelid,a.attnum)
       WHERE c.relnamespace=esquema AND c.relkind IN ('r','p')
         AND a.attnum>0 AND NOT a.attisdropped
      UNION ALL
      SELECT pg_catalog.format('con|%s|%s|%s|%s|%s|%s|%s|%s',
               coalesce(c.conrelid::regclass::text,''),c.conname,c.contype,
               c.condeferrable,c.condeferred,c.convalidated,
               pg_catalog.pg_get_constraintdef(c.oid,false),
               coalesce(pg_catalog.obj_description(c.oid,'pg_constraint'),''))
        FROM pg_catalog.pg_constraint c WHERE c.connamespace=esquema
      UNION ALL
      SELECT pg_catalog.format('idx|%s|%s|%s|%s|%s|%s|%s|%s|%s',
               t.relname,i.relname,pg_catalog.pg_get_userbyid(i.relowner),
               am.amname,coalesce(s.spcname,''),coalesce(i.reloptions::text,''),
               coalesce(pg_catalog.obj_description(i.oid,'pg_class'),''),
               ROW(x.indisunique,x.indnullsnotdistinct,x.indisprimary,
                 x.indisexclusion,x.indimmediate,x.indisclustered,x.indisvalid,
                 x.indcheckxmin,x.indisready,x.indislive,x.indisreplident)::text,
               pg_catalog.pg_get_indexdef(i.oid))
        FROM pg_catalog.pg_index x JOIN pg_catalog.pg_class t ON t.oid=x.indrelid
        JOIN pg_catalog.pg_class i ON i.oid=x.indexrelid
        JOIN pg_catalog.pg_am am ON am.oid=i.relam
        LEFT JOIN pg_catalog.pg_tablespace s ON s.oid=i.reltablespace
       WHERE t.relnamespace=esquema
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
      UNION ALL
      SELECT pg_catalog.format('fun|%s|%s|%s|%s|%s|%s|%s|%s|%s',p.proname,
               pg_catalog.pg_get_function_identity_arguments(p.oid),l.lanname,
               pg_catalog.pg_get_userbyid(p.proowner),p.provolatile,p.prosecdef,
               coalesce(p.proacl::text,''),pg_catalog.pg_get_functiondef(p.oid),
               coalesce(pg_catalog.obj_description(p.oid,'pg_proc'),''))
        FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_language l ON l.oid=p.prolang
       WHERE p.pronamespace=esquema
      UNION ALL
      SELECT pg_catalog.format('pol|%s|%s|%s|%s|%s|%s|%s|%s',
               p.polrelid::regclass::text,p.polname,p.polcmd,p.polpermissive,
               coalesce((SELECT pg_catalog.string_agg(
                 CASE WHEN r.rol=0 THEN 'PUBLIC'
                      ELSE pg_catalog.pg_get_userbyid(r.rol) END,',' ORDER BY
                 CASE WHEN r.rol=0 THEN 'PUBLIC'
                      ELSE pg_catalog.pg_get_userbyid(r.rol) END)
                 FROM pg_catalog.unnest(p.polroles) r(rol)),''),
               coalesce(pg_catalog.pg_get_expr(
                 p.polqual,p.polrelid,false),''),coalesce(pg_catalog.pg_get_expr(
                 p.polwithcheck,p.polrelid,false),''),
               coalesce(pg_catalog.obj_description(p.oid,'pg_policy'),''))
        FROM pg_catalog.pg_policy p JOIN pg_catalog.pg_class c ON c.oid=p.polrelid
       WHERE c.relnamespace=esquema
      UNION ALL
      SELECT pg_catalog.format('typ|%s|%s|%s|%s|%s|%s',t.typname,t.typtype,
               pg_catalog.pg_get_userbyid(t.typowner),t.typcategory,
               coalesce(e.typname,''),coalesce(t.typacl::text,''))
       FROM pg_catalog.pg_type t LEFT JOIN pg_catalog.pg_type e ON e.oid=t.typelem
       WHERE t.typnamespace=esquema
      UNION ALL
      SELECT pg_catalog.format('toast|%s|%s|%s|%s|%s|%s|%s|%s',p.relname,
               pg_catalog.pg_get_userbyid(t.relowner),tam.amname,
               coalesce(ts.spcname,''),coalesce(t.relacl::text,''),
               coalesce(t.reloptions::text,''),
               coalesce(pg_catalog.obj_description(t.oid,'pg_class'),''),
               t.relreplident)
             || pg_catalog.format('|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s',
               pg_catalog.pg_get_userbyid(x.relowner),xam.amname,
               coalesce(xs.spcname,''),coalesce(x.reloptions::text,''),
               coalesce(pg_catalog.obj_description(x.oid,'pg_class'),''),
               i.indkey::text,i.indnkeyatts,i.indnatts,
               coalesce((SELECT pg_catalog.string_agg(
                 pg_catalog.pg_get_indexdef(x.oid,posicion,false),','
                 ORDER BY posicion)
                 FROM pg_catalog.generate_series(1,i.indnatts) posicion),''),
               ROW(i.indisunique,i.indnullsnotdistinct,i.indisprimary,
                 i.indisexclusion,i.indimmediate,i.indisclustered,i.indisvalid,
                 i.indcheckxmin,i.indisready,i.indislive,i.indisreplident)::text) e
        FROM pg_catalog.pg_class p
        JOIN pg_catalog.pg_class t ON t.oid=p.reltoastrelid
        JOIN pg_catalog.pg_am tam ON tam.oid=t.relam
        JOIN pg_catalog.pg_index i ON i.indrelid=t.oid
        JOIN pg_catalog.pg_class x ON x.oid=i.indexrelid
        JOIN pg_catalog.pg_am xam ON xam.oid=x.relam
        LEFT JOIN pg_catalog.pg_tablespace ts ON ts.oid=t.reltablespace
        LEFT JOIN pg_catalog.pg_tablespace xs ON xs.oid=x.reltablespace
       WHERE p.relnamespace=esquema
      UNION ALL
      SELECT pg_catalog.format('defacl|%s|%s|%s|%s',
               pg_catalog.pg_get_userbyid(d.defaclrole),coalesce(n.nspname,''),
               d.defaclobjtype,d.defaclacl::text)
        FROM pg_catalog.pg_default_acl d LEFT JOIN pg_catalog.pg_namespace n
          ON n.oid=d.defaclnamespace
       WHERE d.defaclnamespace=esquema OR d.defaclrole=propietario
    )
    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
      pg_catalog.string_agg(e,E'\n' ORDER BY e),'UTF8')),'hex') INTO observado
      FROM objetos;
    IF observado IS DISTINCT FROM 'bddc55742ae4d509cb884bbf464ac4f90c23c6b680d338943160ea1ee3b1742c' THEN
      RAISE EXCEPTION USING ERRCODE='55000',
        MESSAGE='manifiesto simbolico del predecesor no acreditado',
        DETAIL='huella observada '||coalesce(observado,'<ausente>');
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_constraint
         WHERE conrelid='vec_contexto_actor_v1.organizacion_versiones'::regclass
           AND conname='organizacion_versiones_procedencia_uq'
           AND contype='u' AND convalidated AND NOT condeferrable
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_class
         WHERE relnamespace=esquema AND relname IN (
           'vinculo_corporativo_versiones','vinculo_corporativo_actual',
           'perfil_versiones_persona_uq','vinculo_contexto_versiones_actor_uq'
         )
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_constraint
         WHERE connamespace=esquema AND conname IN (
           'perfil_versiones_persona_uq','vinculo_contexto_versiones_actor_uq'
         )
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_type
         WHERE typnamespace=esquema AND typname IN (
           'vinculo_corporativo_versiones','_vinculo_corporativo_versiones',
           'vinculo_corporativo_actual','_vinculo_corporativo_actual'
         )
    ) OR EXISTS (SELECT 1 FROM pg_catalog.pg_publication WHERE puballtables)
      OR EXISTS (SELECT 1 FROM pg_catalog.pg_publication_namespace
                  WHERE pnnspid=esquema) THEN
        RAISE EXCEPTION USING ERRCODE='55000',
            MESSAGE='base o ausencia nominal de vinculo corporativo RRHH V1 no acreditada';
    END IF;
END
$acreditar_base_y_ausencia$;

SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

ALTER TABLE vec_contexto_actor_v1.perfil_versiones
    ADD CONSTRAINT perfil_versiones_persona_uq
    UNIQUE (perfil_ref, version, persona_ref);
ALTER TABLE vec_contexto_actor_v1.vinculo_contexto_versiones
    ADD CONSTRAINT vinculo_contexto_versiones_actor_uq
    UNIQUE (vinculo_ref, version, cuenta_ref, perfil_ref, persona_ref);

CREATE TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones (
    vinculo_corporativo_ref text NOT NULL,
    version numeric(20,0) NOT NULL,
    cuenta_ref text NOT NULL,
    cuenta_version numeric(20,0) NOT NULL,
    persona_ref text NOT NULL,
    persona_version numeric(20,0) NOT NULL,
    perfil_ref text NOT NULL,
    perfil_version numeric(20,0) NOT NULL,
    vinculo_contexto_ref text NOT NULL,
    vinculo_contexto_version numeric(20,0) NOT NULL,
    organizacion_ref text NOT NULL,
    organizacion_version numeric(20,0) NOT NULL,
    organizacion_procedencia_ref text NOT NULL,
    organizacion_procedencia_version numeric(20,0) NOT NULL,
    organizacion_procedencia_huella_sha256 text NOT NULL,
    organizacion_procedencia_autoridad text NOT NULL,
    superficie text NOT NULL,
    uso text NOT NULL,
    procedencia_ref text NOT NULL,
    procedencia_version numeric(20,0) NOT NULL,
    procedencia_huella_sha256 text NOT NULL,
    procedencia_autoridad text NOT NULL,
    estado text NOT NULL,
    vigente_desde timestamptz(6) NOT NULL,
    vigente_hasta timestamptz(6) NOT NULL,
    CONSTRAINT vinculo_corporativo_versiones_pk
      PRIMARY KEY (vinculo_corporativo_ref, version),
    CONSTRAINT vinculo_corporativo_versiones_actual_uq UNIQUE
      (cuenta_ref, superficie, uso, vinculo_corporativo_ref, version),
    CONSTRAINT vinculo_corporativo_versiones_cuenta_fk FOREIGN KEY
      (cuenta_ref, cuenta_version) REFERENCES
      vec_contexto_actor_v1.proyeccion_cuenta_versiones(cuenta_ref,version)
      MATCH FULL,
    CONSTRAINT vinculo_corporativo_versiones_persona_fk FOREIGN KEY
      (persona_ref, persona_version) REFERENCES
      vec_contexto_actor_v1.persona_versiones(persona_ref,version) MATCH FULL,
    CONSTRAINT vinculo_corporativo_versiones_perfil_persona_fk FOREIGN KEY
      (perfil_ref,perfil_version,persona_ref) REFERENCES
      vec_contexto_actor_v1.perfil_versiones(perfil_ref,version,persona_ref)
      MATCH FULL,
    CONSTRAINT vinculo_corporativo_versiones_vinculo_contexto_fk FOREIGN KEY
      (vinculo_contexto_ref,vinculo_contexto_version,cuenta_ref,perfil_ref,persona_ref)
      REFERENCES vec_contexto_actor_v1.vinculo_contexto_versiones
      (vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref) MATCH FULL,
    CONSTRAINT vinculo_corporativo_versiones_organizacion_fk FOREIGN KEY
      (organizacion_ref,organizacion_version,organizacion_procedencia_ref,
       organizacion_procedencia_version,organizacion_procedencia_huella_sha256,
       organizacion_procedencia_autoridad) REFERENCES
      vec_contexto_actor_v1.organizacion_versiones
      (organizacion_ref,version,procedencia_ref,procedencia_version,
       procedencia_huella_sha256,procedencia_autoridad) MATCH FULL,
    CONSTRAINT vinculo_corporativo_versiones_procedencia_fk FOREIGN KEY
      (procedencia_ref,procedencia_version,procedencia_huella_sha256,
       procedencia_autoridad) REFERENCES vec_contexto_actor_v1.procedencias
      (procedencia_ref,procedencia_version,procedencia_huella_sha256,
       procedencia_autoridad) MATCH FULL,
    CONSTRAINT vinculo_corporativo_versiones_ref_ck CHECK
      (vec_contexto_actor_v1.referencia_valida(vinculo_corporativo_ref,'vcr_')),
    CONSTRAINT vinculo_corporativo_versiones_version_ck CHECK
      (version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_corporativo_versiones_cuenta_version_ck CHECK
      (cuenta_version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_corporativo_versiones_persona_version_ck CHECK
      (persona_version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_corporativo_versiones_perfil_version_ck CHECK
      (perfil_version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_corporativo_versiones_vinculo_contexto_version_ck CHECK
      (vinculo_contexto_version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_corporativo_versiones_organizacion_version_ck CHECK
      (organizacion_version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_corporativo_versiones_superficie_ck CHECK
      (superficie='interna_corporativa'),
    CONSTRAINT vinculo_corporativo_versiones_uso_ck CHECK (uso='consulta_rrhh'),
    CONSTRAINT vinculo_corporativo_versiones_organizacion_procedencia_ck CHECK
      (vec_contexto_actor_v1.procedencia_valida(
       organizacion_procedencia_ref,organizacion_procedencia_version,
       organizacion_procedencia_huella_sha256,organizacion_procedencia_autoridad)),
    CONSTRAINT vinculo_corporativo_versiones_organizacion_autoridad_ck CHECK
      (organizacion_procedencia_autoridad='autoridad_maestra_acreditada'),
    CONSTRAINT vinculo_corporativo_versiones_procedencia_ck CHECK
      (vec_contexto_actor_v1.procedencia_valida(procedencia_ref,
       procedencia_version,procedencia_huella_sha256,procedencia_autoridad)),
    CONSTRAINT vinculo_corporativo_versiones_procedencia_autoridad_ck CHECK
      (procedencia_autoridad='autoridad_maestra_acreditada'),
    CONSTRAINT vinculo_corporativo_versiones_estado_ck CHECK
      (estado IN ('activo','revocado')),
    CONSTRAINT vinculo_corporativo_versiones_vigente_desde_ck CHECK
      (vec_contexto_actor_v1.instante_valido(vigente_desde)),
    CONSTRAINT vinculo_corporativo_versiones_vigente_hasta_ck CHECK
      (vec_contexto_actor_v1.instante_valido(vigente_hasta)),
    CONSTRAINT vinculo_corporativo_versiones_ventana_ck CHECK
      (vigente_hasta > vigente_desde)
);

CREATE TABLE vec_contexto_actor_v1.vinculo_corporativo_actual (
    cuenta_ref text NOT NULL,
    superficie text NOT NULL,
    uso text NOT NULL,
    vinculo_corporativo_ref text NOT NULL,
    version numeric(20,0) NOT NULL,
    CONSTRAINT vinculo_corporativo_actual_pk
      PRIMARY KEY (cuenta_ref,superficie,uso),
    CONSTRAINT vinculo_corporativo_actual_superficie_ck CHECK
      (superficie='interna_corporativa'),
    CONSTRAINT vinculo_corporativo_actual_uso_ck CHECK (uso='consulta_rrhh'),
    CONSTRAINT vinculo_corporativo_actual_version_ck CHECK
      (version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_corporativo_actual_version_fk FOREIGN KEY
      (cuenta_ref,superficie,uso,vinculo_corporativo_ref,version) REFERENCES
      vec_contexto_actor_v1.vinculo_corporativo_versiones
      (cuenta_ref,superficie,uso,vinculo_corporativo_ref,version) MATCH FULL
);

ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
    FORCE ROW LEVEL SECURITY;
CREATE POLICY acceso_propietario_exacto
    ON vec_contexto_actor_v1.vinculo_corporativo_versiones
    AS PERMISSIVE FOR ALL TO vec_contexto_actor_v1_propietario
    USING (CURRENT_USER='vec_contexto_actor_v1_propietario')
    WITH CHECK (CURRENT_USER='vec_contexto_actor_v1_propietario');
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_actual
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_actual
    FORCE ROW LEVEL SECURITY;
CREATE POLICY acceso_propietario_exacto
    ON vec_contexto_actor_v1.vinculo_corporativo_actual
    AS PERMISSIVE FOR ALL TO vec_contexto_actor_v1_propietario
    USING (CURRENT_USER='vec_contexto_actor_v1_propietario')
    WITH CHECK (CURRENT_USER='vec_contexto_actor_v1_propietario');

CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE
    ON vec_contexto_actor_v1.vinculo_corporativo_versiones FOR EACH ROW
    EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia();
CREATE TRIGGER historia_no_truncable BEFORE TRUNCATE
    ON vec_contexto_actor_v1.vinculo_corporativo_versiones FOR EACH STATEMENT
    EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();
CREATE TRIGGER puntero_actual_no_truncable_v2 BEFORE TRUNCATE
    ON vec_contexto_actor_v1.vinculo_corporativo_actual FOR EACH STATEMENT
    EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();
CREATE TRIGGER serializar_mutacion_punteros_actuales_v2
    BEFORE INSERT OR UPDATE OR DELETE
    ON vec_contexto_actor_v1.vinculo_corporativo_actual FOR EACH STATEMENT
    EXECUTE FUNCTION vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2();
CREATE TRIGGER avanzar_generacion_punteros_actuales_v2
    AFTER INSERT OR UPDATE OR DELETE
    ON vec_contexto_actor_v1.vinculo_corporativo_actual FOR EACH STATEMENT
    EXECUTE FUNCTION vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2();

REVOKE ALL ON TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones,
    vec_contexto_actor_v1.vinculo_corporativo_actual
    FROM PUBLIC, vec_contexto_actor_v1_runtime;
REVOKE ALL ON TYPE vec_contexto_actor_v1.vinculo_corporativo_versiones,
    vec_contexto_actor_v1.vinculo_corporativo_actual
    FROM PUBLIC, vec_contexto_actor_v1_runtime;

DO $postcondiciones$
DECLARE
    propietario constant oid := 'vec_contexto_actor_v1_propietario'::regrole;
    esquema constant oid := 'vec_contexto_actor_v1'::regnamespace;
    versiones constant oid := 'vec_contexto_actor_v1.vinculo_corporativo_versiones'::regclass;
    actual constant oid := 'vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass;
BEGIN
    IF (SELECT pg_catalog.count(*) FROM pg_catalog.pg_class
         WHERE oid IN (versiones,actual) AND relkind='r' AND relowner=propietario
           AND relpersistence='p' AND relreplident='d'
           AND relrowsecurity AND relforcerowsecurity) <> 2
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_policy
            WHERE polrelid IN (versiones,actual)
              AND polname='acceso_propietario_exacto' AND polcmd='*'
              AND polpermissive AND polroles=ARRAY[propietario]::oid[]
              AND polqual IS NOT NULL AND polwithcheck IS NOT NULL) <> 2
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_constraint
            WHERE conrelid IN (versiones,actual)) <> 60
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_constraint
            WHERE connamespace=esquema AND conname IN
             ('perfil_versiones_persona_uq','vinculo_contexto_versiones_actor_uq')) <> 2
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_constraint
            WHERE conrelid IN (versiones,actual) AND contype='f'
              AND confmatchtype='f' AND confupdtype='a' AND confdeltype='a'
              AND NOT condeferrable AND NOT condeferred AND convalidated) <> 7
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_index
            WHERE indrelid IN (versiones,actual)) <> 3
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_index i
            JOIN pg_catalog.pg_class x ON x.oid=i.indexrelid
            JOIN pg_catalog.pg_am am ON am.oid=x.relam
            JOIN (VALUES
              ('vinculo_corporativo_versiones_pk',true,
               'vinculo_corporativo_ref,version'),
              ('vinculo_corporativo_versiones_actual_uq',false,
               'cuenta_ref,superficie,uso,vinculo_corporativo_ref,version'),
              ('vinculo_corporativo_actual_pk',true,
               'cuenta_ref,superficie,uso'),
              ('perfil_versiones_persona_uq',false,
               'perfil_ref,version,persona_ref'),
              ('vinculo_contexto_versiones_actor_uq',false,
               'vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref')
            ) e(nombre,primario,columnas) ON e.nombre=x.relname
            WHERE i.indrelid IN (versiones,actual,
              'vec_contexto_actor_v1.perfil_versiones'::regclass,
              'vec_contexto_actor_v1.vinculo_contexto_versiones'::regclass)
              AND x.relowner=propietario AND x.reltablespace=0
              AND x.relpersistence='p' AND x.relreplident='n'
              AND x.reloptions IS NULL AND am.amname='btree'
              AND i.indisunique AND NOT i.indnullsnotdistinct
              AND i.indisprimary=e.primario AND NOT i.indisexclusion
              AND i.indimmediate AND NOT i.indisclustered AND i.indisvalid
              AND NOT i.indcheckxmin AND i.indisready AND i.indislive
              AND NOT i.indisreplident AND i.indnkeyatts=i.indnatts
              AND (SELECT pg_catalog.string_agg(pg_catalog.pg_get_indexdef(
                    i.indexrelid,posicion,false),',' ORDER BY posicion)
                     FROM pg_catalog.generate_series(1,i.indnatts) posicion)
                  = e.columnas) <> 5
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_trigger
            WHERE tgrelid IN (versiones,actual) AND NOT tgisinternal) <> 5
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_attribute
                   WHERE attrelid IN (versiones,actual) AND attnum>0
                     AND NOT attisdropped AND (attacl IS NOT NULL
                       OR atthasdef OR attstorage<>(SELECT typstorage
                         FROM pg_catalog.pg_type WHERE oid=atttypid)
                       OR attcompression<>''))
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
              AND t.relpersistence='p' AND t.relreplident='n'
              AND t.reloptions IS NULL AND am.amname='heap') <> 2
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_index i
            JOIN pg_catalog.pg_class t ON t.oid=i.indrelid
            JOIN pg_catalog.pg_class x ON x.oid=i.indexrelid
            JOIN pg_catalog.pg_am am ON am.oid=x.relam
            WHERE t.oid IN (
              (SELECT reltoastrelid FROM pg_catalog.pg_class WHERE oid=versiones),
              (SELECT reltoastrelid FROM pg_catalog.pg_class WHERE oid=actual))
              AND i.indisunique AND NOT i.indnullsnotdistinct
              AND i.indisprimary AND NOT i.indisexclusion AND i.indimmediate
              AND NOT i.indisclustered AND i.indisvalid AND NOT i.indcheckxmin
              AND i.indisready AND i.indislive AND NOT i.indisreplident
              AND i.indnkeyatts=2 AND i.indnatts=2
              AND (SELECT pg_catalog.string_agg(pg_catalog.pg_get_indexdef(
                    i.indexrelid,posicion,false),',' ORDER BY posicion)
                     FROM pg_catalog.generate_series(1,i.indnatts) posicion)
                  = 'chunk_id,chunk_seq'
              AND x.relowner=propietario AND x.reltablespace=0
              AND x.relpersistence='p' AND x.relreplident='n'
              AND x.reloptions IS NULL AND am.amname='btree') <> 2
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c,
            LATERAL pg_catalog.aclexplode(coalesce(c.relacl,
              pg_catalog.acldefault('r',c.relowner))) a
            WHERE c.oid IN (versiones,actual) AND a.grantee<>propietario)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_attribute a,
            LATERAL pg_catalog.aclexplode(a.attacl) x
            WHERE a.attrelid IN (versiones,actual)
              AND a.attnum>0 AND NOT a.attisdropped
              AND x.grantee<>propietario) THEN
        RAISE EXCEPTION USING ERRCODE='55000',
            MESSAGE='postcondiciones de vinculo corporativo RRHH V1 no acreditadas';
    END IF;
END
$postcondiciones$;

COMMIT;
