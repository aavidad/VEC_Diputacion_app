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
    esquema constant oid := 'vec_contexto_actor_v1'::regnamespace;
    versiones constant oid := 'vec_contexto_actor_v1.vinculo_corporativo_versiones'::regclass;
    actual constant oid := 'vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass;
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
      SELECT pg_catalog.format('rel|%s|%s|%s|%s|%s|%s|%s|%s|%s',c.relname,c.relkind,
               pg_catalog.pg_get_userbyid(c.relowner),c.relrowsecurity,
               c.relforcerowsecurity,coalesce(c.relacl::text,''),
               coalesce(am.amname,''),c.reltablespace,coalesce(c.reloptions::text,'')) AS e
        FROM pg_catalog.pg_class c
        LEFT JOIN pg_catalog.pg_am am ON am.oid=c.relam
       WHERE c.relnamespace=esquema
         AND c.relkind IN ('r','v','m','f','p','S')
      UNION ALL
      SELECT pg_catalog.format('col|%s|%s|%s|%s|%s|%s|%s|%s|%s',
               c.relname,a.attnum,a.attname,
               pg_catalog.format_type(a.atttypid,a.atttypmod),a.attnotnull,
               a.attstorage,a.attcompression,coalesce(a.attacl::text,''),
               coalesce(pg_catalog.pg_get_expr(d.adbin,d.adrelid),''))
        FROM pg_catalog.pg_attribute a JOIN pg_catalog.pg_class c ON c.oid=a.attrelid
        LEFT JOIN pg_catalog.pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum
       WHERE c.relnamespace=esquema AND a.attnum>0 AND NOT a.attisdropped
      UNION ALL
      SELECT pg_catalog.format('con|%s|%s|%s|%s',c.conrelid::regclass::text,
               c.conname,c.contype,pg_catalog.pg_get_constraintdef(c.oid,false))
        FROM pg_catalog.pg_constraint c WHERE c.connamespace=esquema
      UNION ALL
      SELECT pg_catalog.format('idx|%s|%s|%s',t.relname,i.relname,
               pg_catalog.pg_get_indexdef(i.oid))
        FROM pg_catalog.pg_index x JOIN pg_catalog.pg_class t ON t.oid=x.indrelid
        JOIN pg_catalog.pg_class i ON i.oid=x.indexrelid
       WHERE t.relnamespace=esquema
      UNION ALL
      SELECT pg_catalog.format('fun|%s|%s|%s|%s|%s',p.proname,
               pg_catalog.pg_get_function_identity_arguments(p.oid),p.provolatile,
               coalesce(p.proconfig::text,''),p.prosrc)
        FROM pg_catalog.pg_proc p WHERE p.pronamespace=esquema
      UNION ALL
      SELECT pg_catalog.format('typ|%s|%s|%s|%s|%s|%s',t.typname,t.typtype,
               t.typcategory,pg_catalog.pg_get_userbyid(t.typowner),
               coalesce(e.typname,''),coalesce(t.typacl::text,''))
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
               pg_catalog.pg_get_userbyid(t.relowner),tam.amname,t.reltablespace,
               coalesce(t.relacl::text,''),pg_catalog.pg_get_userbyid(x.relowner),
               xam.amname,i.indkey::text)
        FROM pg_catalog.pg_class p JOIN pg_catalog.pg_class t ON t.oid=p.reltoastrelid
        JOIN pg_catalog.pg_am tam ON tam.oid=t.relam
        JOIN pg_catalog.pg_index i ON i.indrelid=t.oid
        JOIN pg_catalog.pg_class x ON x.oid=i.indexrelid
        JOIN pg_catalog.pg_am xam ON xam.oid=x.relam
       WHERE p.relnamespace=esquema
      UNION ALL
      SELECT pg_catalog.format('pol|%s|%s|%s|%s|%s',p.polrelid::regclass::text,
               p.polname,p.polcmd,pg_catalog.pg_get_expr(p.polqual,p.polrelid),
               pg_catalog.pg_get_expr(p.polwithcheck,p.polrelid))
        FROM pg_catalog.pg_policy p JOIN pg_catalog.pg_class c ON c.oid=p.polrelid
       WHERE c.relnamespace=esquema
      UNION ALL
      SELECT pg_catalog.format('trg|%s|%s|%s|%s',t.tgrelid::regclass::text,
               t.tgname,t.tgtype,t.tgfoid::regprocedure::text)
        FROM pg_catalog.pg_trigger t JOIN pg_catalog.pg_class c ON c.oid=t.tgrelid
       WHERE c.relnamespace=esquema AND NOT t.tgisinternal
    )
    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
             pg_catalog.string_agg(e,E'\n' ORDER BY e),'UTF8')),'hex')
      INTO observado FROM elementos;
    IF observado IS DISTINCT FROM '49444be2b4dbf83f3e20e96b5340292720170a1afd9dccf0fd63be44c48cc544' THEN
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
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
                   WHERE p.prosrc LIKE '%vinculo_corporativo_versiones%'
                      OR p.prosrc LIKE '%vinculo_corporativo_actual%')
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
