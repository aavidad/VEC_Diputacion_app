-- Safe-down M1.2: no elimina eventos ni publicaciones y no toca 000008/V2.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000008', 0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000009', 0));

SET LOCAL ROLE vec_autorizacion_propietario;
LOCK TABLE
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1,
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1
    IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
DECLARE
    v_tabla regclass :=
      'vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass;
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_class
         WHERE oid = v_tabla
           AND relkind = 'r' AND relpersistence = 'p'
           AND relowner = 'vec_autorizacion_propietario'::regrole
           AND relrowsecurity AND relforcerowsecurity
           AND pg_catalog.obj_description(oid, 'pg_class') =
             'vec_autorizacion:vinculacion-motivo-consulta-rrhh:evento-v1:000009'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000009 no retira una tabla ajena o alterada';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_class AS c
         WHERE c.oid = v_tabla
           AND (c.relam IS DISTINCT FROM (
                  SELECT a.oid FROM pg_catalog.pg_am AS a
                   WHERE a.amname = 'heap' AND a.amtype = 't'
                ) OR c.relispartition)
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_inherits
         WHERE inhrelid = v_tabla OR inhparent = v_tabla
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_rewrite WHERE ev_class = v_tabla
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000009 no retira una tabla no heap, heredada o con reglas';
    END IF;
    IF EXISTS (
        WITH esperado(numero,nombre,tipo,no_nulo,defecto,identidad,generada,
                      ausente,eliminada) AS (
          VALUES
            (1,'clase_consulta','text',true,NULL::text,'','',false,false),
            (2,'operacion','text',true,NULL::text,'','',false,false),
            (3,'publicacion_version','bigint',true,NULL::text,'','',false,false),
            (4,'evento_ref','text',true,NULL::text,'','',false,false),
            (5,'evento_huella_sha256','text',true,NULL::text,'','',false,false),
            (6,'publicacion_ref','text',true,NULL::text,'','',false,false),
            (7,'publicacion_huella_sha256','text',true,NULL::text,'','',false,false),
            (8,'ocurrida_en','timestamp(6) with time zone',true,NULL::text,'','',false,false),
            (9,'actor_tecnico_ref','text',true,NULL::text,'','',false,false),
            (10,'prueba_vec_secuencia_origen','bigint',true,NULL::text,'','',false,false),
            (11,'prueba_vec_evento_origen_ref','text',true,NULL::text,'','',false,false),
            (12,'prueba_vec_evento_huella_sha256','text',true,NULL::text,'','',false,false),
            (13,'prueba_vec_validada_en','timestamp(6) with time zone',true,NULL::text,'','',false,false),
            (14,'registrada_en','timestamp(6) with time zone',true,'clock_timestamp()','','',false,false),
            (15,'registrada_txid','bigint',true,'txid_current()','','',false,false)
        ), actual AS (
          SELECT a.attnum::integer,a.attname::text,
                 pg_catalog.format_type(a.atttypid,a.atttypmod),a.attnotnull,
                 pg_catalog.pg_get_expr(d.adbin,d.adrelid,true),
                 a.attidentity::text,a.attgenerated::text,
                 a.atthasmissing,a.attisdropped
            FROM pg_catalog.pg_attribute AS a
            LEFT JOIN pg_catalog.pg_attrdef AS d
              ON d.adrelid=a.attrelid AND d.adnum=a.attnum
           WHERE a.attrelid=v_tabla AND a.attnum>0
        )
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
        UNION ALL
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_attribute AS a
          JOIN pg_catalog.pg_type AS t ON t.oid=a.atttypid
         WHERE a.attrelid=v_tabla AND a.attnum>0
           AND a.attcollation IS DISTINCT FROM t.typcollation
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_attribute AS a
          CROSS JOIN LATERAL pg_catalog.aclexplode(a.attacl) AS acl
         WHERE a.attrelid=v_tabla AND a.attnum>0
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000009 no retira columnas, colaciones o ACL alteradas';
    END IF;
    IF EXISTS (
        WITH esperado(nombre,tipo,definicion) AS (
          VALUES
            ('vinculacion_motivo_rrhh_evento_pk','p','PRIMARY KEY (clase_consulta, operacion, publicacion_version)'),
            ('vinculacion_motivo_rrhh_evento_ref_unica','u','UNIQUE (evento_ref)'),
            ('vinculacion_motivo_rrhh_evento_huella_unica','u','UNIQUE (evento_huella_sha256)'),
            ('vinculacion_motivo_rrhh_evento_clase_cerrada','c','CHECK (clase_consulta = ANY (ARRAY[''cuadro''::text, ''detalle''::text]))'),
            ('vinculacion_motivo_rrhh_evento_operacion_cerrada','c','CHECK (operacion = ANY (ARRAY[''publicacion''::text, ''retirada''::text]))'),
            ('vinculacion_motivo_rrhh_evento_version_positiva','c','CHECK (publicacion_version > 0)'),
            ('vinculacion_motivo_rrhh_evento_refs_validas','c','CHECK (evento_ref ~ ''^evento_vinculacion_motivo_rrhh_[0-9a-f]{32}$''::text AND publicacion_ref ~ ''^publicacion_motivo_rrhh_[0-9a-f]{32}$''::text AND prueba_vec_evento_origen_ref ~ ''^evento_[0-9a-f]{32}$''::text)'),
            ('vinculacion_motivo_rrhh_evento_huellas_validas','c','CHECK (evento_huella_sha256 ~ ''^[0-9a-f]{64}$''::text AND evento_huella_sha256 <> repeat(''0''::text, 64) AND publicacion_huella_sha256 ~ ''^[0-9a-f]{64}$''::text AND publicacion_huella_sha256 <> repeat(''0''::text, 64) AND prueba_vec_evento_huella_sha256 ~ ''^[0-9a-f]{64}$''::text AND prueba_vec_evento_huella_sha256 <> repeat(''0''::text, 64))'),
            ('vinculacion_motivo_rrhh_evento_actor_tecnico','c','CHECK (char_length(actor_tecnico_ref) >= 1 AND char_length(actor_tecnico_ref) <= 63)'),
            ('vinculacion_motivo_rrhh_evento_prueba_secuencia','c','CHECK (prueba_vec_secuencia_origen > 0)'),
            ('vinculacion_motivo_rrhh_evento_instantes','c','CHECK (isfinite(ocurrida_en) AND isfinite(prueba_vec_validada_en) AND isfinite(registrada_en) AND ocurrida_en <= registrada_en AND prueba_vec_validada_en <= registrada_en AND EXTRACT(year FROM (ocurrida_en AT TIME ZONE ''UTC''::text)) >= 1::numeric AND EXTRACT(year FROM (ocurrida_en AT TIME ZONE ''UTC''::text)) <= 9999::numeric AND EXTRACT(year FROM (prueba_vec_validada_en AT TIME ZONE ''UTC''::text)) >= 1::numeric AND EXTRACT(year FROM (prueba_vec_validada_en AT TIME ZONE ''UTC''::text)) <= 9999::numeric AND EXTRACT(year FROM (registrada_en AT TIME ZONE ''UTC''::text)) >= 1::numeric AND EXTRACT(year FROM (registrada_en AT TIME ZONE ''UTC''::text)) <= 9999::numeric)'),
            ('vinculacion_motivo_rrhh_evento_txid','c','CHECK (registrada_txid > 0)'),
            ('vinculacion_motivo_rrhh_evento_publicacion_fk','f','FOREIGN KEY (clase_consulta, publicacion_version, publicacion_ref, publicacion_huella_sha256) REFERENCES vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1(clase_consulta, publicacion_version, publicacion_ref, publicacion_huella_sha256)'),
            ('vinculacion_motivo_rrhh_evento_prueba_vec_fk','f','FOREIGN KEY (prueba_vec_secuencia_origen, prueba_vec_evento_origen_ref) REFERENCES vec_autorizacion.motivo_v2_evento_origen(secuencia_origen, evento_origen_ref)')
        ), actual AS (
          SELECT c.conname::text,c.contype::text,
                 pg_catalog.pg_get_constraintdef(c.oid,true)
            FROM pg_catalog.pg_constraint AS c
           WHERE c.conrelid=v_tabla AND c.contype<>'n'
             AND c.convalidated AND NOT c.condeferrable
             AND NOT c.condeferred
             AND c.connoinherit=(c.contype<>'c')
        )
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
        UNION ALL
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_constraint AS c
         WHERE c.conrelid=v_tabla AND c.contype<>'n'
           AND (NOT c.convalidated OR c.condeferrable OR c.condeferred
                OR c.connoinherit IS DISTINCT FROM (c.contype<>'c'))
    ) OR EXISTS (
        WITH esperado(columnas) AS (
          SELECT ARRAY[a.attnum]::smallint[]
            FROM pg_catalog.pg_attribute AS a
           WHERE a.attrelid=v_tabla AND a.attnum>0
             AND NOT a.attisdropped AND a.attnotnull
        ), actual AS (
          SELECT c.conkey FROM pg_catalog.pg_constraint AS c
           WHERE c.conrelid=v_tabla AND c.contype='n'
             AND c.convalidated AND NOT c.condeferrable
             AND NOT c.condeferred AND NOT c.connoinherit
        )
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
        UNION ALL
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
    ) OR EXISTS (
        WITH esperado(indice,primaria,unica,exclusion,valida,lista,viva,
                      inmediata) AS (
          SELECT c.conindid,c.contype='p',true,false,true,true,true,true
            FROM pg_catalog.pg_constraint AS c
           WHERE c.conrelid=v_tabla AND c.contype IN ('p','u')
        ), actual AS (
          SELECT i.indexrelid,i.indisprimary,i.indisunique,i.indisexclusion,
                 i.indisvalid,i.indisready,i.indislive,i.indimmediate
            FROM pg_catalog.pg_index AS i WHERE i.indrelid=v_tabla
        )
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
        UNION ALL
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000009 no retira restricciones o indices alterados';
    END IF;
    IF EXISTS (
        WITH esperado(objeto, marca, retorno, definidor, lenguaje) AS (
          VALUES
            ('vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1()'::regprocedure::oid,
             'vec_autorizacion:vinculacion-motivo-consulta-rrhh:bloqueo-evento-v1:000009',
             'trigger'::regtype::oid, false, 'plpgsql'::name),
            ('vec_autorizacion.validar_insercion_vinculacion_motivo_rrhh_evento_v1()'::regprocedure::oid,
             'vec_autorizacion:vinculacion-motivo-consulta-rrhh:validacion-evento-v1:000009',
             'trigger'::regtype::oid, false, 'plpgsql'::name),
            ('vec_autorizacion.registrar_publicacion_vinculacion_motivo_consulta_rrhh_v1(text,text,text,bigint,text,text,text,integer,text,text,timestamptz)'::regprocedure::oid,
             'vec_autorizacion:vinculacion-motivo-consulta-rrhh:kernel-publicacion-v1:000009',
             'boolean'::regtype::oid, false, 'plpgsql'::name),
            ('vec_autorizacion.registrar_retirada_vinculacion_motivo_consulta_rrhh_v1(text,text,text,bigint,text,text,timestamptz)'::regprocedure::oid,
             'vec_autorizacion:vinculacion-motivo-consulta-rrhh:kernel-retirada-v1:000009',
             'boolean'::regtype::oid, false, 'plpgsql'::name),
            ('vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1(text,text,bigint,text,text,text,integer,text,text,timestamptz)'::regprocedure::oid,
             'vec_autorizacion:vinculacion-motivo-consulta-rrhh:publicar-cuadro-v1:000009',
             'boolean'::regtype::oid, true, 'sql'::name),
            ('vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1(text,text,bigint,text,text,text,integer,text,text,timestamptz)'::regprocedure::oid,
             'vec_autorizacion:vinculacion-motivo-consulta-rrhh:publicar-detalle-v1:000009',
             'boolean'::regtype::oid, true, 'sql'::name),
            ('vec_autorizacion.retirar_vinculacion_motivo_cuadro_rrhh_v1(text,text,bigint,text,text,timestamptz)'::regprocedure::oid,
             'vec_autorizacion:vinculacion-motivo-consulta-rrhh:retirar-cuadro-v1:000009',
             'boolean'::regtype::oid, true, 'sql'::name),
            ('vec_autorizacion.retirar_vinculacion_motivo_detalle_rrhh_v1(text,text,bigint,text,text,timestamptz)'::regprocedure::oid,
             'vec_autorizacion:vinculacion-motivo-consulta-rrhh:retirar-detalle-v1:000009',
             'boolean'::regtype::oid, true, 'sql'::name)
        ), actual AS (
          SELECT p.oid, pg_catalog.obj_description(p.oid, 'pg_proc'),
                 p.prorettype, p.prosecdef, l.lanname
            FROM pg_catalog.pg_proc AS p
            JOIN pg_catalog.pg_language AS l ON l.oid = p.prolang
           WHERE p.pronamespace = 'vec_autorizacion'::regnamespace
             AND p.proname = ANY (ARRAY[
               'bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1',
               'validar_insercion_vinculacion_motivo_rrhh_evento_v1',
               'registrar_publicacion_vinculacion_motivo_consulta_rrhh_v1',
               'registrar_retirada_vinculacion_motivo_consulta_rrhh_v1',
               'publicar_vinculacion_motivo_cuadro_rrhh_v1',
               'publicar_vinculacion_motivo_detalle_rrhh_v1',
               'retirar_vinculacion_motivo_cuadro_rrhh_v1',
               'retirar_vinculacion_motivo_detalle_rrhh_v1'
             ]::name[])
             AND p.proowner = 'vec_autorizacion_propietario'::regrole
             AND p.prokind = 'f' AND p.provolatile = 'v'
             AND p.proconfig IS NOT DISTINCT FROM
                 ARRAY['search_path=pg_catalog']
        )
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
        UNION ALL
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000009 no retira funciones ajenas o alteradas';
    END IF;
    IF (SELECT pg_catalog.count(*) FROM pg_catalog.pg_policy
         WHERE polrelid = v_tabla
           AND polname = 'acceso_propietario_exacto' AND polcmd = '*'
           AND polpermissive
           AND polroles = ARRAY[
               'vec_autorizacion_propietario'::regrole::oid]
           AND pg_catalog.pg_get_expr(polqual, polrelid) =
               '(CURRENT_USER = ''vec_autorizacion_propietario''::name)'
           AND pg_catalog.pg_get_expr(polwithcheck, polrelid) =
               '(CURRENT_USER = ''vec_autorizacion_propietario''::name)'
       ) <> 1
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_policy
            WHERE polrelid = v_tabla) <> 1
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_policy
            WHERE polrelid IN (
              'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
              'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass)
              AND polname='acceso_propietario_exacto' AND polcmd='*'
              AND polpermissive
              AND polroles=ARRAY['vec_autorizacion_propietario'::regrole::oid]
              AND pg_catalog.pg_get_expr(polqual,polrelid)=
                '(CURRENT_USER = ''vec_autorizacion_propietario''::name)'
              AND pg_catalog.pg_get_expr(polwithcheck,polrelid)=
                '(CURRENT_USER = ''vec_autorizacion_propietario''::name)') <> 2
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_policy
            WHERE polrelid IN (
              'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
              'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass)) <> 2 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000009 no retira una politica RLS alterada';
    END IF;
    IF EXISTS (
        WITH esperado(nombre,tipo,habilitado,funcion,padre,otra,indice,
                      restriccion,diferible,diferida,argumentos,atributos,
                      bytes,sin_when,sin_old,sin_new) AS (
          VALUES
            ('vinculacion_motivo_rrhh_evento_validar'::name,7::smallint,'O',
             'vec_autorizacion.validar_insercion_vinculacion_motivo_rrhh_evento_v1()'::regprocedure::oid,
             0::oid,0::oid,0::oid,0::oid,false,false,0,'','',true,true,true),
            ('vinculacion_motivo_rrhh_evento_inmutable'::name,27::smallint,'O',
             'vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1()'::regprocedure::oid,
             0::oid,0::oid,0::oid,0::oid,false,false,0,'','',true,true,true),
            ('vinculacion_motivo_rrhh_evento_no_truncar'::name,34::smallint,'O',
             'vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1()'::regprocedure::oid,
             0::oid,0::oid,0::oid,0::oid,false,false,0,'','',true,true,true)
        ), actual AS (
          SELECT t.tgname,t.tgtype,t.tgenabled::text,t.tgfoid,
                 t.tgparentid,t.tgconstrrelid,t.tgconstrindid,t.tgconstraint,
                 t.tgdeferrable,t.tginitdeferred,t.tgnargs,t.tgattr::text,
                 pg_catalog.encode(t.tgargs,'hex'),t.tgqual IS NULL,
                 t.tgoldtable IS NULL,t.tgnewtable IS NULL
            FROM pg_catalog.pg_trigger AS t
           WHERE t.tgrelid=v_tabla AND NOT t.tgisinternal
        )
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
        UNION ALL
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000009 no retira disparadores alterados';
    END IF;
    IF EXISTS (
        WITH claves AS (
          SELECT c.oid,c.conrelid,c.confrelid,c.conindid
            FROM pg_catalog.pg_constraint AS c
           WHERE c.conrelid=v_tabla AND c.contype='f'
        ), esperado AS (
          SELECT c.oid,x.tabla,x.otra,c.conindid,x.tipo,'O',true,x.funcion,true
            FROM claves AS c CROSS JOIN LATERAL (VALUES
              (c.conrelid,c.confrelid,5::smallint,'RI_FKey_check_ins'),
              (c.conrelid,c.confrelid,17::smallint,'RI_FKey_check_upd'),
              (c.confrelid,c.conrelid,9::smallint,'RI_FKey_noaction_del'),
              (c.confrelid,c.conrelid,17::smallint,'RI_FKey_noaction_upd')
            ) AS x(tabla,otra,tipo,funcion)
        ), actual AS (
          SELECT t.tgconstraint,t.tgrelid,t.tgconstrrelid,t.tgconstrindid,
                 t.tgtype,t.tgenabled::text,t.tgisinternal,p.proname::text,
                 ROW(t.tgparentid,t.tgdeferrable,t.tginitdeferred,t.tgnargs,
                     t.tgattr::text,pg_catalog.encode(t.tgargs,'hex'),
                     t.tgqual IS NULL,t.tgoldtable IS NULL,t.tgnewtable IS NULL)
                 = ROW(0::oid,false,false,0,'','',true,true,true)
            FROM pg_catalog.pg_trigger AS t
            JOIN pg_catalog.pg_proc AS p ON p.oid=t.tgfoid
           WHERE t.tgconstraint IN (SELECT oid FROM claves)
        )
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
        UNION ALL
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000009 no retira disparadores RI alterados';
    END IF;
    IF EXISTS (
        WITH esperado(grantee, privilege_type, is_grantable) AS (
          SELECT 'vec_autorizacion_propietario'::regrole::oid, permiso, false
            FROM pg_catalog.unnest(ARRAY[
              'INSERT','SELECT','UPDATE','DELETE','TRUNCATE','REFERENCES','TRIGGER'
              ,'MAINTAIN'
            ]) AS permiso
        ), actual AS (
          SELECT a.grantee, a.privilege_type, a.is_grantable
            FROM pg_catalog.pg_class AS c
            CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(c.relacl, pg_catalog.acldefault('r', c.relowner))
            ) AS a
           WHERE c.oid = v_tabla
        )
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
        UNION ALL
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
    ) OR EXISTS (
        WITH funciones AS (
          SELECT p.oid, p.proowner, p.prosecdef, p.proacl
            FROM pg_catalog.pg_proc AS p
           WHERE p.oid IN (
             'vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1()'::regprocedure,
             'vec_autorizacion.validar_insercion_vinculacion_motivo_rrhh_evento_v1()'::regprocedure,
             'vec_autorizacion.registrar_publicacion_vinculacion_motivo_consulta_rrhh_v1(text,text,text,bigint,text,text,text,integer,text,text,timestamptz)'::regprocedure,
             'vec_autorizacion.registrar_retirada_vinculacion_motivo_consulta_rrhh_v1(text,text,text,bigint,text,text,timestamptz)'::regprocedure,
             'vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1(text,text,bigint,text,text,text,integer,text,text,timestamptz)'::regprocedure,
             'vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1(text,text,bigint,text,text,text,integer,text,text,timestamptz)'::regprocedure,
             'vec_autorizacion.retirar_vinculacion_motivo_cuadro_rrhh_v1(text,text,bigint,text,text,timestamptz)'::regprocedure,
             'vec_autorizacion.retirar_vinculacion_motivo_detalle_rrhh_v1(text,text,bigint,text,text,timestamptz)'::regprocedure
           )
        ), esperado(objeto, grantee, privilege_type, is_grantable) AS (
          SELECT oid, proowner, 'EXECUTE', false FROM funciones
          UNION ALL
          SELECT oid, 'vec_autorizacion_motivos_proyector'::regrole::oid,
                 'EXECUTE', false
            FROM funciones WHERE prosecdef
        ), actual AS (
          SELECT f.oid, a.grantee, a.privilege_type, a.is_grantable
            FROM funciones AS f
            CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(f.proacl, pg_catalog.acldefault('f', f.proowner))
            ) AS a
        )
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
        UNION ALL
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
    ) OR EXISTS (
        WITH esperado(grantee,privilege_type,is_grantable) AS (
          VALUES ('vec_autorizacion_propietario'::regrole::oid,'USAGE',false)
        ), tipos AS (
          SELECT base.oid AS tipo_base,base.typowner,base.typacl,
                 vector.typacl AS vector_acl,vector.typelem
            FROM pg_catalog.pg_type AS base
            JOIN pg_catalog.pg_type AS vector ON vector.oid=base.typarray
           WHERE base.typrelid=v_tabla
        ), actual AS (
          SELECT a.grantee,a.privilege_type,a.is_grantable
            FROM tipos AS t CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(t.typacl,pg_catalog.acldefault('T',t.typowner))
            ) AS a
        )
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
        UNION ALL
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
        UNION ALL
        SELECT NULL::oid,NULL::text,NULL::boolean FROM tipos
         WHERE vector_acl IS NOT NULL OR typelem<>tipo_base
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000009 no retira objetos con ACL alteradas';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc AS p
         WHERE p.pronamespace = 'vec_autorizacion'::regnamespace
           AND p.proname IN (
             'resolver_motivo_cuadro_rrhh_v1',
             'resolver_motivo_detalle_rrhh_v1'
           )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '2BP01',
            MESSAGE = '000009 no se retira con dependencias M1.3';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1
    ) OR EXISTS (
        SELECT 1
          FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
    ) OR (SELECT pg_catalog.count(*)
            FROM vec_autorizacion
                 .vinculacion_motivo_consulta_rrhh_checkpoint_v1
           WHERE clase_consulta IN ('cuadro', 'detalle')
             AND ultima_publicacion_version = 0
             AND ultima_publicacion_ref IS NULL
             AND ultima_publicacion_huella_sha256 IS NULL) <> 2
       OR (SELECT pg_catalog.count(*)
             FROM vec_autorizacion
                  .vinculacion_motivo_consulta_rrhh_checkpoint_v1) <> 2 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000009 no se revierte: existe evidencia de motivos RRHH';
    END IF;
END
$prevalidacion$;

REVOKE EXECUTE ON FUNCTION
    vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1(
      text,text,bigint,text,text,text,integer,text,text,timestamptz),
    vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1(
      text,text,bigint,text,text,text,integer,text,text,timestamptz),
    vec_autorizacion.retirar_vinculacion_motivo_cuadro_rrhh_v1(
      text,text,bigint,text,text,timestamptz),
    vec_autorizacion.retirar_vinculacion_motivo_detalle_rrhh_v1(
      text,text,bigint,text,text,timestamptz)
    FROM vec_autorizacion_motivos_proyector;

DROP FUNCTION vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1(
    text,text,bigint,text,text,text,integer,text,text,timestamptz) RESTRICT;
DROP FUNCTION vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1(
    text,text,bigint,text,text,text,integer,text,text,timestamptz) RESTRICT;
DROP FUNCTION vec_autorizacion.retirar_vinculacion_motivo_cuadro_rrhh_v1(
    text,text,bigint,text,text,timestamptz) RESTRICT;
DROP FUNCTION vec_autorizacion.retirar_vinculacion_motivo_detalle_rrhh_v1(
    text,text,bigint,text,text,timestamptz) RESTRICT;
DROP FUNCTION
  vec_autorizacion.registrar_publicacion_vinculacion_motivo_consulta_rrhh_v1(
    text,text,text,bigint,text,text,text,integer,text,text,timestamptz) RESTRICT;
DROP FUNCTION
  vec_autorizacion.registrar_retirada_vinculacion_motivo_consulta_rrhh_v1(
    text,text,text,bigint,text,text,timestamptz) RESTRICT;
DROP TABLE
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 RESTRICT;
DROP FUNCTION
    vec_autorizacion.validar_insercion_vinculacion_motivo_rrhh_evento_v1()
    RESTRICT;
DROP FUNCTION
    vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1()
    RESTRICT;
COMMIT;
