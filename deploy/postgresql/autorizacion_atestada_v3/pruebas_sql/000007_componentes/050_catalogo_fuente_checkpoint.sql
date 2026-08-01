-- Matriz adversarial del catalogo y checkpoint F0-B1.

CREATE FUNCTION
vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba()
RETURNS pg_catalog.bool
LANGUAGE sql
VOLATILE
SET search_path = pg_catalog
AS $funcion$
    WITH propietario AS (
        SELECT r.oid
          FROM pg_catalog.pg_roles AS r
         WHERE r.rolname = 'vec_autorizacion_atestada_v3_propietario'
           AND NOT r.rolcanlogin AND NOT r.rolsuper
           AND NOT r.rolcreatedb AND NOT r.rolcreaterole
           AND NOT r.rolreplication AND NOT r.rolbypassrls
    ), relaciones AS (
        SELECT c.oid,c.relname,c.relowner,c.reltoastrelid FROM pg_catalog.pg_class AS c
          JOIN pg_catalog.pg_namespace AS n ON n.oid=c.relnamespace
         WHERE n.nspname = 'vec_autorizacion_atestada_v3'
           AND c.relname IN ('fuente_corporativa_contexto_actor_v1','revocacion_fuente_corporativa_contexto_actor_v1')
           AND c.relkind = 'r' AND c.relpersistence = 'p'
           AND c.relam=(SELECT a.oid FROM pg_catalog.pg_am AS a WHERE a.amname='heap' AND a.amtype='t')
           AND c.relnatts=CASE c.relname WHEN 'fuente_corporativa_contexto_actor_v1' THEN 22 ELSE 7 END
           AND NOT c.relispartition AND c.reloftype = 0
           AND c.reloptions IS NULL AND c.relreplident = 'd'
           AND c.relrowsecurity AND c.relforcerowsecurity
           AND NOT c.relhasrules AND NOT c.relhassubclass
           AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_rewrite AS w WHERE w.ev_class=c.oid)
           AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_inherits AS h WHERE h.inhrelid=c.oid OR h.inhparent=c.oid)
           AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_publication_tables AS u WHERE u.schemaname=n.nspname AND u.tablename=c.relname)
           AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_publication AS u WHERE u.puballtables)
           AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_publication_namespace AS u WHERE u.pnnspid=n.oid)
           AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_publication_rel AS u WHERE u.prrelid=c.oid)
           AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_statistic_ext AS s WHERE s.stxrelid=c.oid)
           AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_depend AS d JOIN pg_catalog.pg_rewrite AS w ON d.classid='pg_catalog.pg_rewrite'::pg_catalog.regclass AND d.objid=w.oid WHERE d.refclassid='pg_catalog.pg_class'::pg_catalog.regclass AND d.refobjid=c.oid AND w.ev_class<>c.oid)
    ), columnas_esperadas(relacion,posicion,nombre,tipo,no_nula,colacion,expresion_defecto) AS (
        VALUES
        ('fuente_corporativa_contexto_actor_v1',1,'fuente_ref','text',true,'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid,NULL), ('fuente_corporativa_contexto_actor_v1',2,'fuente_version','numeric(20,0)',true,0::pg_catalog.oid,NULL),
        ('fuente_corporativa_contexto_actor_v1',3,'audiencia_consumo','text',true,'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid,NULL), ('fuente_corporativa_contexto_actor_v1',4,'accion','text',true,'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid,NULL),
        ('fuente_corporativa_contexto_actor_v1',5,'tipo_efecto','text',true,'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid,NULL), ('fuente_corporativa_contexto_actor_v1',6,'clave_id','text',true,'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid,NULL),
        ('fuente_corporativa_contexto_actor_v1',7,'clave_version','numeric(20,0)',true,0::pg_catalog.oid,NULL), ('fuente_corporativa_contexto_actor_v1',8,'revision_gobierno','numeric(20,0)',true,0::pg_catalog.oid,NULL),
        ('fuente_corporativa_contexto_actor_v1',9,'huella_gobierno_sha256','text',true,'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid,NULL), ('fuente_corporativa_contexto_actor_v1',10,'emisor_id','text',true,'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid,NULL),
        ('fuente_corporativa_contexto_actor_v1',11,'configuracion_revision','text',true,'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid,NULL), ('fuente_corporativa_contexto_actor_v1',12,'configuracion_secuencia','numeric(20,0)',true,0::pg_catalog.oid,NULL),
        ('fuente_corporativa_contexto_actor_v1',13,'huella_configuracion_sha256','text',true,'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid,NULL), ('fuente_corporativa_contexto_actor_v1',14,'raiz_clave_id','text',true,'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid,NULL),
        ('fuente_corporativa_contexto_actor_v1',15,'raiz_version','numeric(20,0)',true,0::pg_catalog.oid,NULL), ('fuente_corporativa_contexto_actor_v1',16,'huella_raiz_spki_sha256','text',true,'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid,NULL),
        ('fuente_corporativa_contexto_actor_v1',17,'audiencia_despliegue','text',true,'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid,NULL), ('fuente_corporativa_contexto_actor_v1',18,'suite','text',true,'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid,NULL),
        ('fuente_corporativa_contexto_actor_v1',19,'valida_desde','timestamp(6) with time zone',true,0::pg_catalog.oid,NULL), ('fuente_corporativa_contexto_actor_v1',20,'valida_hasta','timestamp(6) with time zone',true,0::pg_catalog.oid,NULL),
        ('fuente_corporativa_contexto_actor_v1',21,'acto_ref','text',true,'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid,NULL), ('fuente_corporativa_contexto_actor_v1',22,'registrada_en','timestamp(6) with time zone',true,0::pg_catalog.oid,'clock_timestamp()'),
        ('revocacion_fuente_corporativa_contexto_actor_v1',1,'fuente_ref','text',true,'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid,NULL), ('revocacion_fuente_corporativa_contexto_actor_v1',2,'fuente_version','numeric(20,0)',true,0::pg_catalog.oid,NULL),
        ('revocacion_fuente_corporativa_contexto_actor_v1',3,'audiencia_consumo','text',true,'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid,NULL), ('revocacion_fuente_corporativa_contexto_actor_v1',4,'revocada_en','timestamp(6) with time zone',true,0::pg_catalog.oid,NULL),
        ('revocacion_fuente_corporativa_contexto_actor_v1',5,'motivo_catalogado_ref','text',true,'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid,NULL), ('revocacion_fuente_corporativa_contexto_actor_v1',6,'acto_ref','text',true,'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid,NULL),
        ('revocacion_fuente_corporativa_contexto_actor_v1',7,'registrada_en','timestamp(6) with time zone',true,0::pg_catalog.oid,'clock_timestamp()')
    ), columnas_exactas AS (
        SELECT a.attrelid, a.attnum
          FROM columnas_esperadas AS e
          JOIN pg_catalog.pg_class AS c ON c.relname = e.relacion
          JOIN pg_catalog.pg_namespace AS n ON n.oid=c.relnamespace AND n.nspname='vec_autorizacion_atestada_v3'
          JOIN pg_catalog.pg_attribute AS a ON a.attrelid=c.oid AND a.attnum=e.posicion
          LEFT JOIN pg_catalog.pg_attrdef AS d ON d.adrelid=a.attrelid AND d.adnum=a.attnum
         WHERE NOT a.attisdropped AND a.attname = e.nombre
           AND pg_catalog.format_type(a.atttypid, a.atttypmod) = e.tipo
           AND a.attnotnull = e.no_nula AND a.attcollation = e.colacion
           AND a.attidentity = '' AND a.attgenerated = ''
           AND a.attacl IS NULL
           AND a.attstorage=CASE WHEN e.tipo='text' THEN 'x' WHEN e.tipo LIKE 'numeric%' THEN 'm' ELSE 'p' END
           AND a.attcompression=''::pg_catalog."char" AND a.attstattarget IS NULL
           AND a.attoptions IS NULL AND a.attfdwoptions IS NULL AND a.attislocal
           AND a.attinhcount=0 AND a.attndims=0 AND NOT a.atthasmissing AND a.attmissingval IS NULL
           AND a.atthasdef=(e.expresion_defecto IS NOT NULL)
           AND pg_catalog.pg_get_expr(d.adbin, d.adrelid)
               IS NOT DISTINCT FROM e.expresion_defecto
    ), toast_exactos AS (
        SELECT r.oid FROM relaciones AS r
          JOIN pg_catalog.pg_class AS t ON t.oid=r.reltoastrelid
          JOIN pg_catalog.pg_namespace AS n ON n.oid=t.relnamespace
          JOIN pg_catalog.pg_am AS a ON a.oid=t.relam
         WHERE n.nspname='pg_toast' AND t.relname='pg_toast_'||r.oid::pg_catalog.text
           AND t.relowner=r.relowner AND t.relkind='t' AND t.relpersistence='p'
           AND a.amname='heap' AND a.amtype='t' AND t.reltablespace=0
           AND t.reloptions IS NULL AND t.relacl IS NULL AND t.reltype=0
           AND t.reloftype=0 AND t.reltoastrelid=0 AND t.relnatts=3
           AND t.relhasindex AND t.relchecks=0 AND NOT t.relhasrules
           AND NOT t.relhastriggers AND NOT t.relhassubclass
           AND NOT t.relrowsecurity AND NOT t.relforcerowsecurity
           AND NOT t.relispartition AND t.relreplident='n'
           AND (SELECT pg_catalog.count(*) FROM pg_catalog.pg_attribute AS x WHERE x.attrelid=t.oid AND x.attnum>0)=3
           AND NOT EXISTS (
               SELECT 1 FROM pg_catalog.pg_attribute AS x LEFT JOIN (VALUES
                   (1,'chunk_id','oid'),(2,'chunk_seq','integer'),(3,'chunk_data','bytea')
               ) AS e(posicion,nombre,tipo) ON e.posicion=x.attnum
                WHERE x.attrelid=t.oid AND x.attnum>0 AND (e.posicion IS NULL
                  OR x.attname<>e.nombre OR pg_catalog.format_type(x.atttypid,x.atttypmod)<>e.tipo
                  OR x.attisdropped OR x.attnotnull OR x.attstorage<>'p' OR x.attcompression<>''::pg_catalog."char"
                  OR x.attstattarget IS NOT NULL OR x.attoptions IS NOT NULL OR x.attfdwoptions IS NOT NULL
                  OR NOT x.attislocal OR x.attinhcount<>0 OR x.attndims<>0 OR x.atthasmissing
                  OR x.attmissingval IS NOT NULL OR x.atthasdef OR x.attidentity<>'' OR x.attgenerated<>''
                  OR x.attacl IS NOT NULL OR x.attcollation<>0)
           )
           AND (SELECT pg_catalog.count(*) FROM pg_catalog.pg_index AS i WHERE i.indrelid=t.oid)=1
           AND EXISTS (
               SELECT 1 FROM pg_catalog.pg_index AS i JOIN pg_catalog.pg_class AS c ON c.oid=i.indexrelid JOIN pg_catalog.pg_namespace AS ni ON ni.oid=c.relnamespace JOIN pg_catalog.pg_am AS ai ON ai.oid=c.relam
                WHERE i.indrelid=t.oid AND ni.nspname='pg_toast' AND c.relname=t.relname||'_index'
                  AND c.relowner=r.relowner AND c.relkind='i' AND c.relpersistence='p' AND ai.amname='btree' AND ai.amtype='i'
                  AND c.reltablespace=0 AND c.reloptions IS NULL AND c.relacl IS NULL AND NOT c.relispartition
                  AND c.reltype=0 AND c.reloftype=0 AND c.reltoastrelid=0 AND c.relnatts=2 AND NOT c.relhasindex AND c.relchecks=0 AND NOT c.relhasrules AND NOT c.relhastriggers AND NOT c.relhassubclass AND NOT c.relrowsecurity AND NOT c.relforcerowsecurity AND c.relreplident='n'
                  AND (SELECT pg_catalog.count(*) FROM pg_catalog.pg_attribute AS x WHERE x.attrelid=c.oid AND x.attnum>0)=2 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_attribute AS x WHERE x.attrelid=c.oid AND x.attnum>0 AND x.attisdropped)
                  AND i.indnatts=2 AND i.indnkeyatts=2 AND i.indkey::pg_catalog.text='1 2'
                  AND i.indcollation::pg_catalog.text='0 0' AND i.indoption::pg_catalog.text='0 0'
                  AND (SELECT pg_catalog.string_agg(no.nspname||'.'||o.opcname,',' ORDER BY u.ord) FROM pg_catalog.unnest(i.indclass) WITH ORDINALITY AS u(oid,ord) JOIN pg_catalog.pg_opclass AS o ON o.oid=u.oid JOIN pg_catalog.pg_namespace AS no ON no.oid=o.opcnamespace)='pg_catalog.oid_ops,pg_catalog.int4_ops'
                  AND i.indisunique AND i.indisprimary AND i.indimmediate AND i.indisvalid AND i.indisready AND i.indislive
                  AND NOT i.indisexclusion AND NOT i.indisclustered AND NOT i.indisreplident
                  AND NOT i.indcheckxmin AND NOT i.indnullsnotdistinct AND i.indpred IS NULL AND i.indexprs IS NULL
           )
    ), restricciones AS (
        SELECT c.*
          FROM pg_catalog.pg_constraint AS c
         WHERE c.conrelid IN (SELECT r.oid FROM relaciones AS r)
           AND c.contype <> 'n'
    ), indices_esperados(nombre,relacion,columnas,clases,colaciones,opciones,unica,primaria) AS (VALUES
        ('f0_fuente_pk','fuente_corporativa_contexto_actor_v1','1 2 3','pg_catalog.text_ops,pg_catalog.numeric_ops,pg_catalog.text_ops','pg_catalog.C,0,pg_catalog.C','0 0 0',true,true),
        ('f0_fuente_clave_idx','fuente_corporativa_contexto_actor_v1','6 7','pg_catalog.text_ops,pg_catalog.numeric_ops','pg_catalog.C,0','0 0',false,false),
        ('f0_fuente_config_raiz_idx','fuente_corporativa_contexto_actor_v1','11 14 15','pg_catalog.text_ops,pg_catalog.text_ops,pg_catalog.numeric_ops','pg_catalog.C,pg_catalog.C,0','0 0 0',false,false),
        ('f0_fuente_raiz_idx','fuente_corporativa_contexto_actor_v1','14 15','pg_catalog.text_ops,pg_catalog.numeric_ops','pg_catalog.C,0','0 0',false,false),
        ('f0_revocacion_fuente_pk','revocacion_fuente_corporativa_contexto_actor_v1','1 2 3','pg_catalog.text_ops,pg_catalog.numeric_ops,pg_catalog.text_ops','pg_catalog.C,0,pg_catalog.C','0 0 0',true,true)
    ), indices AS (
        SELECT i.*, ci.relname AS nombre, ct.relname AS relacion,
               a.amname, ci.relkind AS clase_relacion,
               ci.relpersistence, ci.relispartition, ci.reloptions,
               ci.reltablespace, ci.relowner,
               (SELECT pg_catalog.string_agg(no.nspname||'.'||o.opcname,',' ORDER BY u.ord)
                  FROM pg_catalog.unnest(i.indclass) WITH ORDINALITY AS u(oid,ord)
                  JOIN pg_catalog.pg_opclass AS o ON o.oid=u.oid
                  JOIN pg_catalog.pg_namespace AS no ON no.oid=o.opcnamespace) AS clases,
               (SELECT pg_catalog.string_agg(CASE WHEN u.oid=0 THEN '0' ELSE nc.nspname||'.'||co.collname END,',' ORDER BY u.ord)
                  FROM pg_catalog.unnest(i.indcollation) WITH ORDINALITY AS u(oid,ord)
                  LEFT JOIN pg_catalog.pg_collation AS co ON co.oid=u.oid
                  LEFT JOIN pg_catalog.pg_namespace AS nc ON nc.oid=co.collnamespace) AS colaciones
          FROM pg_catalog.pg_index AS i
          JOIN pg_catalog.pg_class AS ci ON ci.oid = i.indexrelid
          JOIN pg_catalog.pg_class AS ct ON ct.oid = i.indrelid
          JOIN pg_catalog.pg_am AS a ON a.oid = ci.relam
         WHERE i.indrelid IN (SELECT r.oid FROM relaciones AS r)
    ), disparadores_esperados(relacion,nombre,tipo,funcion,definicion) AS (
        VALUES
        ('fuente_corporativa_contexto_actor_v1','f0_checkpoint_antes',7,'vec_autorizacion_atestada_v3.avanzar_checkpoint_fuente_corporativa_contexto_actor_v1()','CREATE TRIGGER f0_checkpoint_antes BEFORE INSERT ON vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.avanzar_checkpoint_fuente_corporativa_contexto_actor_v1()'),
        ('revocacion_fuente_corporativa_contexto_actor_v1','f0_checkpoint_antes',7,'vec_autorizacion_atestada_v3.avanzar_checkpoint_fuente_corporativa_contexto_actor_v1()','CREATE TRIGGER f0_checkpoint_antes BEFORE INSERT ON vec_autorizacion_atestada_v3.revocacion_fuente_corporativa_contexto_actor_v1 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.avanzar_checkpoint_fuente_corporativa_contexto_actor_v1()'),
        ('fuente_corporativa_contexto_actor_v1','f0_historia_inmutable',27,'vec_autorizacion_atestada_v3.rechazar_mutacion()','CREATE TRIGGER f0_historia_inmutable BEFORE DELETE OR UPDATE ON vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion()'),
        ('revocacion_fuente_corporativa_contexto_actor_v1','f0_historia_inmutable',27,'vec_autorizacion_atestada_v3.rechazar_mutacion()','CREATE TRIGGER f0_historia_inmutable BEFORE DELETE OR UPDATE ON vec_autorizacion_atestada_v3.revocacion_fuente_corporativa_contexto_actor_v1 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion()'),
        ('fuente_corporativa_contexto_actor_v1','f0_historia_no_truncable',34,'vec_autorizacion_atestada_v3.rechazar_truncado()','CREATE TRIGGER f0_historia_no_truncable BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado()'),
        ('revocacion_fuente_corporativa_contexto_actor_v1','f0_historia_no_truncable',34,'vec_autorizacion_atestada_v3.rechazar_truncado()','CREATE TRIGGER f0_historia_no_truncable BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.revocacion_fuente_corporativa_contexto_actor_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado()')
    ), disparadores AS (
        SELECT t.*, c.relname AS relacion,
               pg_catalog.pg_get_triggerdef(t.oid, false) AS definicion
          FROM pg_catalog.pg_trigger AS t
          JOIN pg_catalog.pg_class AS c ON c.oid=t.tgrelid
         WHERE t.tgrelid IN (SELECT r.oid FROM relaciones AS r)
           AND NOT t.tgisinternal
    ), claves_foraneas AS (
        SELECT c.oid,c.conname,c.conrelid,c.confrelid,c.conindid
          FROM restricciones AS c WHERE c.contype='f'
    ), disparadores_ri_esperados AS (
        SELECT c.oid,x.tabla,x.otra,c.conindid,x.tipo,'O'::pg_catalog.text,
               true,x.funcion::pg_catalog.text,true
          FROM claves_foraneas AS c CROSS JOIN LATERAL (VALUES
            (c.conrelid,c.confrelid,5::pg_catalog.int2,'RI_FKey_check_ins'),
            (c.conrelid,c.confrelid,17::pg_catalog.int2,'RI_FKey_check_upd'),
            (c.confrelid,c.conrelid,9::pg_catalog.int2,'RI_FKey_noaction_del'),
            (c.confrelid,c.conrelid,17::pg_catalog.int2,'RI_FKey_noaction_upd')
          ) AS x(tabla,otra,tipo,funcion)
    ), disparadores_ri_actuales AS (
        SELECT t.tgconstraint,t.tgrelid,t.tgconstrrelid,t.tgconstrindid,
               t.tgtype,t.tgenabled::pg_catalog.text,t.tgisinternal,
               p.proname::pg_catalog.text,
               ROW(t.tgparentid,t.tgdeferrable,t.tginitdeferred,t.tgnargs,
                   t.tgattr::pg_catalog.text,pg_catalog.encode(t.tgargs,'hex'),
                   t.tgqual IS NULL,t.tgoldtable IS NULL,t.tgnewtable IS NULL)=
               ROW(0::pg_catalog.oid,false,false,0,'','',true,true,true)
          FROM pg_catalog.pg_trigger AS t JOIN pg_catalog.pg_proc AS p ON p.oid=t.tgfoid
         WHERE t.tgconstraint IN (SELECT c.oid FROM claves_foraneas AS c)
    ), funcion AS (
        SELECT p.*, l.lanname
          FROM pg_catalog.pg_proc AS p
          JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
          JOIN pg_catalog.pg_language AS l ON l.oid = p.prolang
         WHERE n.nspname = 'vec_autorizacion_atestada_v3'
           AND p.proname =
               'avanzar_checkpoint_fuente_corporativa_contexto_actor_v1'
    ), clases_toast(oid,etiqueta) AS (
        SELECT t.oid,'toast:'||r.relname FROM relaciones AS r JOIN pg_catalog.pg_class AS t ON t.oid=r.reltoastrelid
        UNION ALL SELECT i.indexrelid,'indice_toast:'||r.relname FROM relaciones AS r JOIN pg_catalog.pg_index AS i ON i.indrelid=r.reltoastrelid
    ), objetos_b1(classid,objid,etiqueta) AS (
        SELECT 'pg_catalog.pg_class'::pg_catalog.regclass,r.oid,'tabla:'||r.relname FROM relaciones AS r
        UNION SELECT 'pg_catalog.pg_type'::pg_catalog.regclass,t.oid,'tipo:'||n.nspname||'.'||t.typname FROM pg_catalog.pg_type AS t JOIN pg_catalog.pg_namespace AS n ON n.oid=t.typnamespace WHERE t.typrelid IN (SELECT r.oid FROM relaciones AS r) OR t.oid IN (SELECT x.typarray FROM pg_catalog.pg_type AS x WHERE x.typrelid IN (SELECT r.oid FROM relaciones AS r))
        UNION SELECT 'pg_catalog.pg_class'::pg_catalog.regclass,i.indexrelid,'indice:'||i.nombre FROM indices AS i
        UNION SELECT 'pg_catalog.pg_constraint'::pg_catalog.regclass,c.oid,'restriccion:'||c.conrelid::pg_catalog.regclass::pg_catalog.text||'.'||c.conname FROM restricciones AS c
        UNION SELECT 'pg_catalog.pg_trigger'::pg_catalog.regclass,t.oid,'trigger:'||t.relacion||'.'||t.tgname FROM disparadores AS t
        UNION SELECT 'pg_catalog.pg_trigger'::pg_catalog.regclass,t.oid,'trigger_ri:'||c.conname||':'||t.tgrelid::pg_catalog.regclass::pg_catalog.text||':'||t.tgtype::pg_catalog.text||':'||p.proname FROM pg_catalog.pg_trigger AS t JOIN claves_foraneas AS c ON c.oid=t.tgconstraint JOIN pg_catalog.pg_proc AS p ON p.oid=t.tgfoid
        UNION SELECT 'pg_catalog.pg_proc'::pg_catalog.regclass,f.oid,'funcion:avanzar_checkpoint_fuente' FROM funcion AS f
        UNION SELECT 'pg_catalog.pg_class'::pg_catalog.regclass,c.oid,c.etiqueta FROM clases_toast AS c
        UNION SELECT 'pg_catalog.pg_type'::pg_catalog.regclass,t.oid,'tipo_toast:'||n.nspname||'.'||t.typname FROM pg_catalog.pg_type AS t JOIN pg_catalog.pg_namespace AS n ON n.oid=t.typnamespace WHERE t.typrelid IN (SELECT c.oid FROM clases_toast AS c) OR t.oid IN (SELECT x.typarray FROM pg_catalog.pg_type AS x WHERE x.typrelid IN (SELECT c.oid FROM clases_toast AS c))
    ), dependencias_b1 AS (
        SELECT d.* FROM pg_catalog.pg_depend AS d WHERE EXISTS (SELECT 1 FROM objetos_b1 AS o WHERE (o.classid,o.objid)=(d.classid,d.objid) OR (o.classid,o.objid)=(d.refclassid,d.refobjid))
    ), resumen_dependencias AS (
        SELECT pg_catalog.count(*)::pg_catalog.text||'|'||pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.string_agg(COALESCE(o.etiqueta,pg_catalog.pg_describe_object(d.classid,d.objid,d.objsubid))||':'||d.objsubid::pg_catalog.text||'>'||d.deptype::pg_catalog.text||'>'||COALESCE(r.etiqueta,pg_catalog.pg_describe_object(d.refclassid,d.refobjid,d.refobjsubid))||':'||d.refobjsubid::pg_catalog.text,E'\n' ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'UTF8')),'hex') AS firma
          FROM dependencias_b1 AS d LEFT JOIN objetos_b1 AS o ON (o.classid,o.objid)=(d.classid,d.objid) LEFT JOIN objetos_b1 AS r ON (r.classid,r.objid)=(d.refclassid,d.refobjid)
    )
    SELECT (SELECT pg_catalog.count(*) FROM propietario) = 1
       AND (SELECT pg_catalog.count(*) FROM relaciones) = 2
       AND NOT EXISTS (
           SELECT 1 FROM relaciones AS r, propietario AS p
            WHERE r.relowner <> p.oid
       )
       AND (SELECT pg_catalog.count(*) FROM columnas_esperadas) = 29
       AND (SELECT pg_catalog.count(*) FROM columnas_exactas) = 29
       AND (SELECT pg_catalog.count(*) FROM toast_exactos) = 2
       AND (SELECT pg_catalog.count(*) FROM pg_catalog.pg_attribute AS a WHERE a.attrelid IN (SELECT r.oid FROM relaciones AS r) AND a.attnum>0) = 29
       AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_attribute AS a WHERE a.attrelid IN (SELECT r.oid FROM relaciones AS r) AND a.attnum>0 AND a.attisdropped)
       AND (SELECT pg_catalog.count(*) FROM restricciones) = 35
       AND (
           SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               pg_catalog.string_agg(
                   c.conrelid::pg_catalog.regclass::pg_catalog.text || '|' ||
                   c.conname || '|' || c.contype::pg_catalog.text || '|' ||
                   pg_catalog.pg_get_constraintdef(c.oid), E'\n' ORDER BY
                   c.conrelid::pg_catalog.regclass::pg_catalog.text, c.conname
               ), 'UTF8')), 'hex')
             FROM restricciones AS c
       ) = '0c36f6be90e7dcaa096ae8f28f62505bf51594fc77d1bf29840afcfffe74e00a'
       AND NOT EXISTS (
           SELECT 1 FROM restricciones AS c
            WHERE NOT c.convalidated OR c.condeferrable OR c.condeferred
              OR NOT c.conislocal OR c.coninhcount <> 0
       )
       AND (
           SELECT pg_catalog.count(*) FROM restricciones AS c
            WHERE c.contype = 'f' AND c.confmatchtype = 'f'
              AND c.confupdtype = 'a' AND c.confdeltype = 'a'
       ) = 5
       AND EXISTS (
           SELECT 1 FROM restricciones AS c
            WHERE c.conname = 'f0_fuente_clave_fk'
              AND c.conkey = ARRAY[6,7]::pg_catalog.int2[]
              AND c.confrelid =
                  'vec_autorizacion_atestada_v3.clave_capacidad_version'::pg_catalog.regclass
              AND c.confkey = ARRAY[1,2]::pg_catalog.int2[]
       )
       AND EXISTS (
           SELECT 1 FROM restricciones AS c
            WHERE c.conname = 'f0_fuente_config_raiz_fk'
              AND c.conkey = ARRAY[11,14,15]::pg_catalog.int2[]
              AND c.confrelid =
                  'vec_autorizacion_atestada_v3.configuracion_raiz'::pg_catalog.regclass
              AND c.confkey = ARRAY[1,2,3]::pg_catalog.int2[]
       )
       AND EXISTS (
           SELECT 1 FROM restricciones AS c
            WHERE c.conname = 'f0_revocacion_fuente_fk'
              AND c.conkey = ARRAY[1,2,3]::pg_catalog.int2[]
              AND c.confrelid =
                  'vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1'::pg_catalog.regclass
              AND c.confkey = ARRAY[1,2,3]::pg_catalog.int2[]
       )
       AND (SELECT pg_catalog.count(*) FROM indices) = 5
       AND NOT EXISTS (
           SELECT 1
             FROM indices AS i FULL JOIN indices_esperados AS e
               ON e.nombre=i.nombre
            WHERE e.nombre IS NULL OR i.nombre IS NULL
              OR i.relacion<>e.relacion OR i.amname<>'btree'
              OR i.clase_relacion<>'i' OR i.relpersistence<>'p'
              OR i.relispartition OR i.reloptions IS NOT NULL
              OR i.reltablespace<>0 OR i.relowner<>(SELECT oid FROM propietario)
              OR i.indkey::pg_catalog.text<>e.columnas
              OR i.clases<>e.clases OR i.colaciones<>e.colaciones
              OR i.indoption::pg_catalog.text<>e.opciones
              OR i.indnatts<>pg_catalog.array_length(pg_catalog.string_to_array(e.columnas,' '),1)
              OR i.indnkeyatts<>i.indnatts OR i.indisunique<>e.unica
              OR i.indisprimary<>e.primaria OR i.indnullsnotdistinct
              OR NOT i.indisvalid OR NOT i.indisready OR NOT i.indislive
              OR i.indisexclusion OR i.indisclustered OR i.indisreplident
              OR NOT i.indimmediate OR i.indcheckxmin
              OR i.indpred IS NOT NULL OR i.indexprs IS NOT NULL
       )
       AND (SELECT pg_catalog.count(*) FROM disparadores) = 6
       AND NOT EXISTS (
           SELECT 1
             FROM disparadores AS t FULL JOIN disparadores_esperados AS e
               ON e.relacion=t.relacion AND e.nombre=t.tgname
            WHERE e.nombre IS NULL OR t.tgname IS NULL OR t.tgtype<>e.tipo
              OR t.tgfoid<>e.funcion::pg_catalog.regprocedure
              OR t.tgenabled<>'O' OR t.tgqual IS NOT NULL OR t.tgnargs<>0
              OR pg_catalog.octet_length(t.tgargs)<>0 OR t.tgattr::pg_catalog.text<>''
              OR t.tgoldtable IS NOT NULL OR t.tgnewtable IS NOT NULL
              OR t.tgparentid<>0 OR t.tgconstrrelid<>0 OR t.tgconstraint<>0
              OR t.tgdeferrable OR t.tginitdeferred OR t.definicion<>e.definicion
       )
       AND NOT EXISTS (
           (SELECT * FROM disparadores_ri_esperados EXCEPT ALL
            SELECT * FROM disparadores_ri_actuales)
           UNION ALL
           (SELECT * FROM disparadores_ri_actuales EXCEPT ALL
            SELECT * FROM disparadores_ri_esperados)
       )
       AND NOT EXISTS (SELECT 1 FROM dependencias_b1 AS d WHERE d.deptype IN ('e','x'))
       AND (SELECT r.firma FROM resumen_dependencias AS r)='201|056c54da40ca8b22b7897e3d375ccbcc20d693632410b4bc48e0159104211215'
       AND (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_policy AS p, propietario AS o
            WHERE p.polrelid IN (SELECT r.oid FROM relaciones AS r)
              AND p.polname = 'propietario_exacto'
              AND p.polpermissive AND p.polcmd = '*'
              AND p.polroles = ARRAY[o.oid]::pg_catalog.oid[]
              AND pg_catalog.pg_get_expr(p.polqual, p.polrelid) =
                  '(CURRENT_USER = ''vec_autorizacion_atestada_v3_propietario''::name)'
              AND pg_catalog.pg_get_expr(p.polwithcheck, p.polrelid) =
                  '(CURRENT_USER = ''vec_autorizacion_atestada_v3_propietario''::name)'
       ) = 2
       AND NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_policy AS p
            WHERE p.polrelid IN (SELECT r.oid FROM relaciones AS r)
              AND p.polname <> 'propietario_exacto'
       )
       AND NOT EXISTS (
           SELECT 1 FROM relaciones AS r
           CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
               (SELECT c.relacl FROM pg_catalog.pg_class AS c WHERE c.oid=r.oid),
               pg_catalog.acldefault('r', r.relowner)
           )) AS a
            WHERE a.grantee <> r.relowner OR a.is_grantable
       )
       AND NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_type AS t, propietario AS o
           CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
               t.typacl, pg_catalog.acldefault('T', t.typowner)
           )) AS a
            WHERE t.typrelid IN (SELECT r.oid FROM relaciones AS r)
              AND (t.typowner <> o.oid OR a.grantee <> o.oid
                   OR a.is_grantable)
       )
       AND (SELECT pg_catalog.count(*) FROM funcion) = 1
       AND EXISTS (
           SELECT 1 FROM funcion AS f, propietario AS o
            WHERE f.proowner = o.oid AND f.prokind = 'f'
              AND f.prorettype = 'pg_catalog.trigger'::pg_catalog.regtype
              AND f.pronargs = 0 AND f.pronargdefaults = 0
              AND f.provariadic = 0 AND NOT f.proretset
              AND f.provolatile = 'v' AND NOT f.proisstrict
              AND f.prosecdef AND NOT f.proleakproof AND f.proparallel = 'u'
              AND f.proconfig = ARRAY['search_path=pg_catalog']
              AND f.lanname = 'plpgsql'
              AND pg_catalog.encode(pg_catalog.sha256(
                      pg_catalog.convert_to(f.prosrc,'UTF8')),'hex') =
                  'f2153ed75259bb84a8aad1ba293b95bfd12c3e9c6c11d38ed9bfc2c26fea9cb4'
              AND NOT EXISTS (
                  SELECT 1 FROM pg_catalog.aclexplode(COALESCE(
                      f.proacl, pg_catalog.acldefault('f', f.proowner)
                  )) AS a
                   WHERE a.grantee <> o.oid OR a.is_grantable
              )
       )
       AND EXISTS (
           SELECT 1 FROM pg_catalog.pg_constraint AS c
            WHERE c.conrelid =
                  'vec_autorizacion_atestada_v3.clave_capacidad_version'::pg_catalog.regclass
              AND c.conname =
                  'clave_capacidad_version_audiencia_consumo_check'
              AND c.contype = 'c' AND c.convalidated
              AND pg_catalog.pg_get_constraintdef(c.oid) =
                  'CHECK ((audiencia_consumo = ANY (ARRAY[' ||
                  '''vec_contratacion_temporal.confirmar_alta_atestada.v1''::text, ' ||
                  '''vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1''::text, ' ||
                  '''vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1''::text, ' ||
                  '''vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1''::text, ' ||
                  '''vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1''::text, ' ||
                  '''vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1''::text, ' ||
                  '''vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1''::text])))'
       )
$funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3
    .acreditar_forma_catalogo_fuente_b1_prueba() FROM PUBLIC;
DO $forma_inicial$ BEGIN IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT TRUE THEN RAISE EXCEPTION 'B1: forma inicial del catalogo invalida' USING ERRCODE='XX000'; END IF; END $forma_inicial$;
DO $huella_acreditador$ BEGIN IF (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') FROM pg_catalog.pg_proc AS p WHERE p.oid='vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba()'::pg_catalog.regprocedure)<>'62117d89c765f8d453d74f5b1feb7f401cc0cbb89c6a4f0ecbb65bb8f73a96ac' THEN RAISE EXCEPTION 'B1: cuerpo del acreditador alterado' USING ERRCODE='XX000'; END IF; END $huella_acreditador$;
GRANT SELECT ON vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 TO vec_autorizacion_atestada_v3_migrador;
DO $acl_hostil$ BEGIN IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT FALSE THEN RAISE EXCEPTION 'B1: ACL adicional no detectada' USING ERRCODE='XX000'; END IF; END $acl_hostil$;
REVOKE ALL ON vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 FROM vec_autorizacion_atestada_v3_migrador;
ALTER TABLE vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 DISABLE ROW LEVEL SECURITY;
DO $rls_hostil$ BEGIN IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT FALSE THEN RAISE EXCEPTION 'B1: RLS deshabilitada no detectada' USING ERRCODE='XX000'; END IF; END $rls_hostil$;
ALTER TABLE vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 DISABLE TRIGGER f0_checkpoint_antes;
DO $trigger_hostil$ BEGIN IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT FALSE THEN RAISE EXCEPTION 'B1: trigger deshabilitado no detectado' USING ERRCODE='XX000'; END IF; END $trigger_hostil$;
ALTER TABLE vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 ENABLE TRIGGER f0_checkpoint_antes;
DO $formas_hostiles$
BEGIN
    BEGIN EXECUTE 'DROP INDEX vec_autorizacion_atestada_v3.f0_fuente_raiz_idx';
        EXECUTE 'CREATE UNIQUE INDEX f0_fuente_raiz_idx ON vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 USING btree (acto_ref)';
        IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT FALSE THEN RAISE EXCEPTION 'B1: indice homonimo hostil no detectado' USING ERRCODE='XX000'; END IF; RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT TRUE THEN RAISE EXCEPTION 'B1: indice exacto no restaurado' USING ERRCODE='XX000'; END IF;
    BEGIN EXECUTE 'DROP TRIGGER f0_checkpoint_antes ON vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1';
        EXECUTE 'CREATE TRIGGER f0_checkpoint_antes BEFORE INSERT ON vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 FOR EACH ROW WHEN (false) EXECUTE FUNCTION vec_autorizacion_atestada_v3.avanzar_checkpoint_fuente_corporativa_contexto_actor_v1()';
        IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT FALSE THEN RAISE EXCEPTION 'B1: condicion hostil de trigger no detectada' USING ERRCODE='XX000'; END IF; RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT TRUE THEN RAISE EXCEPTION 'B1: trigger exacto no restaurado' USING ERRCODE='XX000'; END IF;
    BEGIN EXECUTE $ddl$CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.avanzar_checkpoint_fuente_corporativa_contexto_actor_v1()
RETURNS pg_catalog.trigger LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog AS $cuerpo$ BEGIN RETURN NEW; END $cuerpo$$ddl$;
        IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT FALSE THEN RAISE EXCEPTION 'B1: cuerpo hostil de funcion no detectado' USING ERRCODE='XX000'; END IF; RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT TRUE THEN RAISE EXCEPTION 'B1: funcion exacta no restaurada' USING ERRCODE='XX000'; END IF;
END $formas_hostiles$;
CREATE FUNCTION vec_autorizacion_atestada_v3
    .avanzar_checkpoint_fuente_corporativa_contexto_actor_v1(pg_catalog.text)
RETURNS pg_catalog.bool LANGUAGE sql IMMUTABLE
SET search_path = pg_catalog AS $sobrecarga$ SELECT false $sobrecarga$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3
    .avanzar_checkpoint_fuente_corporativa_contexto_actor_v1(pg_catalog.text)
FROM PUBLIC;
DO $sobrecarga_hostil$ BEGIN IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT FALSE THEN RAISE EXCEPTION 'B1: sobrecarga homonima no detectada' USING ERRCODE='XX000'; END IF; END $sobrecarga_hostil$;
DROP FUNCTION vec_autorizacion_atestada_v3
    .avanzar_checkpoint_fuente_corporativa_contexto_actor_v1(pg_catalog.text);
ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
    DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
    ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check
    CHECK (audiencia_consumo <> '');
DO $audiencias_hostiles$ BEGIN IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT FALSE THEN RAISE EXCEPTION 'B1: ampliacion arbitraria de audiencias no detectada' USING ERRCODE='XX000'; END IF; END $audiencias_hostiles$;
ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
    DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
    ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check CHECK (
        audiencia_consumo IN (
            'vec_contratacion_temporal.confirmar_alta_atestada.v1',
            'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
            'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1',
            'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
            'vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1',
            'vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1',
            'vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1'
        )
    );

DO $forma_restaurada$ BEGIN IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT TRUE THEN RAISE EXCEPTION 'B1: forma exacta no restaurada' USING ERRCODE='XX000'; END IF; END $forma_restaurada$;
DO $catalogos_hostiles$
BEGIN
    BEGIN
        EXECUTE 'CREATE RULE f0_fuente_borrado_hostil AS ON DELETE TO vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 DO INSTEAD NOTHING';
        IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT FALSE THEN RAISE EXCEPTION 'B1: regla hostil no detectada' USING ERRCODE='XX000'; END IF;
        RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT TRUE THEN RAISE EXCEPTION 'B1: regla no restaurada' USING ERRCODE='XX000'; END IF;
    BEGIN
        EXECUTE 'CREATE TABLE vec_autorizacion_atestada_v3.f0_fuente_hija_hostil () INHERITS (vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1)';
        IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT FALSE THEN RAISE EXCEPTION 'B1: herencia hostil no detectada' USING ERRCODE='XX000'; END IF;
        RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT TRUE THEN RAISE EXCEPTION 'B1: herencia no restaurada' USING ERRCODE='XX000'; END IF;
    BEGIN
        EXECUTE 'CREATE VIEW vec_autorizacion_atestada_v3.f0_fuente_vista_hostil AS SELECT fuente_ref FROM vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1';
        IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT FALSE THEN RAISE EXCEPTION 'B1: vista entrante no detectada' USING ERRCODE='XX000'; END IF;
        RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT TRUE THEN RAISE EXCEPTION 'B1: vista no restaurada' USING ERRCODE='XX000'; END IF;
    BEGIN
        EXECUTE 'CREATE STATISTICS vec_autorizacion_atestada_v3.f0_fuente_estadistica_hostil ON fuente_version,clave_version FROM vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1';
        IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT FALSE THEN RAISE EXCEPTION 'B1: estadistica extendida no detectada' USING ERRCODE='XX000'; END IF;
        RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT TRUE THEN RAISE EXCEPTION 'B1: estadistica no restaurada' USING ERRCODE='XX000'; END IF;
    BEGIN
        EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 ALTER COLUMN fuente_ref SET STATISTICS 100';
        IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT FALSE THEN RAISE EXCEPTION 'B1: objetivo estadistico hostil no detectado' USING ERRCODE='XX000'; END IF; RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT TRUE THEN RAISE EXCEPTION 'B1: objetivo estadistico no restaurado' USING ERRCODE='XX000'; END IF;
    BEGIN
        EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 ALTER COLUMN fuente_ref SET STORAGE PLAIN';
        IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT FALSE THEN RAISE EXCEPTION 'B1: almacenamiento hostil no detectado' USING ERRCODE='XX000'; END IF; RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT TRUE THEN RAISE EXCEPTION 'B1: almacenamiento no restaurado' USING ERRCODE='XX000'; END IF;
    BEGIN
        EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 ALTER COLUMN fuente_ref SET COMPRESSION pglz';
        IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT FALSE THEN RAISE EXCEPTION 'B1: compresion hostil no detectada' USING ERRCODE='XX000'; END IF; RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT TRUE THEN RAISE EXCEPTION 'B1: compresion no restaurada' USING ERRCODE='XX000'; END IF;
    BEGIN
        EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 SET (toast.autovacuum_enabled=false)';
        IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT FALSE THEN RAISE EXCEPTION 'B1: opcion TOAST hostil no detectada' USING ERRCODE='XX000'; END IF; RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT TRUE THEN RAISE EXCEPTION 'B1: opcion TOAST no restaurada' USING ERRCODE='XX000'; END IF;
    BEGIN
        EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 ADD COLUMN f0_columna_hostil text'; EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 DROP COLUMN f0_columna_hostil';
        IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT FALSE THEN RAISE EXCEPTION 'B1: atributo fisico caido no detectado' USING ERRCODE='XX000'; END IF; RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    IF vec_autorizacion_atestada_v3.acreditar_forma_catalogo_fuente_b1_prueba() IS NOT TRUE THEN RAISE EXCEPTION 'B1: columnas fisicas no restauradas' USING ERRCODE='XX000'; END IF;
END $catalogos_hostiles$;
-- Publicaciones, extension y RI deshabilitado requieren DBA; la
-- contrarrevision los muta como postgres sin escalada local del componente.
DO $catalogo_y_checkpoint$
DECLARE
    v_revision_base pg_catalog.numeric(20, 0); v_revision_clave pg_catalog.numeric(20, 0);
    v_configuracion_secuencia pg_catalog.numeric(20, 0); v_raiz_version pg_catalog.numeric(20, 0);
    v_revision_checkpoint pg_catalog.numeric(20, 0); v_config_checkpoint pg_catalog.numeric(20, 0); v_raiz_checkpoint pg_catalog.numeric(20, 0);
    v_config_ref constant pg_catalog.text := 'config:f0-b1-prueba'; v_raiz_ref constant pg_catalog.text := 'raiz:f0-b1-prueba';
    v_fuente_ref constant pg_catalog.text := 'fuente:f0-b1-prueba'; v_huella_config constant pg_catalog.text := 'c'||pg_catalog.repeat('0',63);
    v_spki pg_catalog.bytea := pg_catalog.decode(pg_catalog.repeat('a1',44),'hex'); v_huella_raiz pg_catalog.text := pg_catalog.encode(pg_catalog.sha256(v_spki),'hex');
    v_audiencias constant pg_catalog.text[] := ARRAY['vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1','vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1','vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1','vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1'];
    v_acciones constant pg_catalog.text[] := ARRAY['contexto_actor.organizacion_corporativa.publicar','contexto_actor.organizacion_corporativa.revocar','contexto_actor.vinculo_corporativo.publicar','contexto_actor.vinculo_corporativo.revocar'];
    v_tipos constant pg_catalog.text[] := ARRAY['organizacion_corporativa.alta','organizacion_corporativa.revocacion','vinculo_corporativo.alta','vinculo_corporativo.revocacion'];
    v_i pg_catalog.int4; v_secreto pg_catalog.bytea;
    v_huella_secreto pg_catalog.text; v_huella_gobierno pg_catalog.text;
BEGIN
    SELECT cp.revision,cp.configuracion_secuencia_minima,cp.raiz_version_minima
      INTO v_revision_checkpoint, v_config_checkpoint, v_raiz_checkpoint
      FROM vec_autorizacion_atestada_v3.checkpoint_gobierno AS cp WHERE cp.control_id;
    SELECT COALESCE(pg_catalog.max(k.revision_gobierno), 0) + 100
      INTO v_revision_clave
      FROM vec_autorizacion_atestada_v3.clave_capacidad_version AS k;
    SELECT GREATEST(COALESCE(pg_catalog.max(c.secuencia),0),v_config_checkpoint)+100
      INTO v_configuracion_secuencia
      FROM vec_autorizacion_atestada_v3.configuracion_confianza_version AS c;
    SELECT GREATEST(COALESCE(pg_catalog.max(r.version),0),v_raiz_checkpoint)+100
      INTO v_raiz_version
      FROM vec_autorizacion_atestada_v3.raiz_confianza_version AS r;
    IF GREATEST(v_revision_clave+4,v_configuracion_secuencia,v_raiz_version)>=9007199254740991::pg_catalog.numeric THEN
        RAISE EXCEPTION 'B1: base sintetica sin margen numerico' USING ERRCODE='XX000';
    END IF;

    INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version (revision,secuencia,huella_configuracion_sha256,publicada_en,expira_en,acto_ref) VALUES (v_config_ref,v_configuracion_secuencia,v_huella_config,'2026-01-01 00:00:00+00','2030-01-01 00:00:00+00','acto:f0-b1-configuracion');
    INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version (
        clave_id, version, clave_publica_spki, huella_spki_sha256,
        valida_desde, valida_hasta, suite, audiencia_despliegue, acto_ref
    ) VALUES (v_raiz_ref,v_raiz_version,v_spki,pg_catalog.encode(pg_catalog.sha256(v_spki),'hex'),'2026-01-01 00:00:00+00','2030-01-01 00:00:00+00','VEC-AD-3-COSE-EDDSA-1','despliegue:f0-b1','acto:f0-b1-raiz');
    INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz VALUES (v_config_ref,v_raiz_ref,v_raiz_version);

    FOR v_i IN 1..4 LOOP
        v_secreto:=pg_catalog.decode(pg_catalog.repeat(pg_catalog.lpad(pg_catalog.to_hex(16+v_i),2,'0'),32),'hex');
        v_huella_secreto:=pg_catalog.encode(pg_catalog.sha256(v_secreto),'hex');
        v_huella_gobierno:=pg_catalog.substr(pg_catalog.repeat(pg_catalog.to_hex(v_i),64),1,64);
        INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version (
            clave_id, version, revision_gobierno, huella_gobierno_sha256,
            secreto_hmac, huella_secreto_sha256, emisor_id,
            audiencia_consumo, valida_desde, valida_hasta, acto_ref
        ) VALUES (
            'clave:f0-b1-' || v_i::pg_catalog.text, 1,
            v_revision_clave + v_i, v_huella_gobierno,
            v_secreto, v_huella_secreto, 'emisor:f0-b1', v_audiencias[v_i],
            '2026-01-01 00:00:00+00', '2030-01-01 00:00:00+00',
            'acto:f0-b1-clave-' || v_i::pg_catalog.text
        );
    END LOOP;

    SELECT cp.revision INTO v_revision_base
      FROM vec_autorizacion_atestada_v3.checkpoint_gobierno AS cp
     WHERE cp.control_id;
    FOR v_i IN 1..4 LOOP
        INSERT INTO vec_autorizacion_atestada_v3
            .fuente_corporativa_contexto_actor_v1 (
            fuente_ref, fuente_version, audiencia_consumo, accion,
            tipo_efecto, clave_id, clave_version, revision_gobierno,
            huella_gobierno_sha256, emisor_id, configuracion_revision,
            configuracion_secuencia, huella_configuracion_sha256,
            raiz_clave_id, raiz_version, huella_raiz_spki_sha256,
            audiencia_despliegue, suite, valida_desde, valida_hasta, acto_ref
        ) VALUES (
            v_fuente_ref, 1, v_audiencias[v_i], v_acciones[v_i],
            v_tipos[v_i], 'clave:f0-b1-' || v_i::pg_catalog.text, 1,
            v_revision_clave + v_i,
            pg_catalog.substr(
                pg_catalog.repeat(pg_catalog.to_hex(v_i),64),1,64
            ),
            'emisor:f0-b1', v_config_ref, v_configuracion_secuencia,
            v_huella_config, v_raiz_ref, v_raiz_version, v_huella_raiz,
            'despliegue:f0-b1', 'VEC-AD-3-COSE-EDDSA-1',
            '2026-02-01 00:00:00+00', '2029-01-01 00:00:00+00',
            'acto:f0-b1-fuente-' || v_i::pg_catalog.text
        );
    END LOOP;
    IF NOT EXISTS (
        SELECT 1 FROM vec_autorizacion_atestada_v3.checkpoint_gobierno AS cp
         WHERE cp.control_id AND cp.revision = v_revision_base + 4
           AND cp.configuracion_secuencia_minima =
               GREATEST(v_config_checkpoint,v_configuracion_secuencia)
           AND cp.raiz_version_minima =
               GREATEST(v_raiz_checkpoint,v_raiz_version)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'B1: alta no avanzo causalmente el checkpoint';
    END IF;

    SELECT cp.revision INTO v_revision_base
      FROM vec_autorizacion_atestada_v3.checkpoint_gobierno AS cp
     WHERE cp.control_id;
    BEGIN
        INSERT INTO vec_autorizacion_atestada_v3
            .fuente_corporativa_contexto_actor_v1
        SELECT 'fuente:f0-b1-cruce', f.fuente_version,
               f.audiencia_consumo, 'accion:incorrecta', f.tipo_efecto,
               f.clave_id, f.clave_version, f.revision_gobierno,
               f.huella_gobierno_sha256, f.emisor_id,
               f.configuracion_revision, f.configuracion_secuencia,
               f.huella_configuracion_sha256, f.raiz_clave_id,
               f.raiz_version, f.huella_raiz_spki_sha256,
               f.audiencia_despliegue, f.suite, f.valida_desde,
               f.valida_hasta, 'acto:f0-b1-cruce', f.registrada_en
          FROM vec_autorizacion_atestada_v3
                   .fuente_corporativa_contexto_actor_v1 AS f
         WHERE f.fuente_ref=v_fuente_ref AND f.audiencia_consumo=v_audiencias[1];
        RAISE EXCEPTION USING ERRCODE='XX000',
            MESSAGE='B1: cruce nominal invalido aceptado';
    EXCEPTION WHEN check_violation THEN NULL;
    END;
    BEGIN
        INSERT INTO vec_autorizacion_atestada_v3
            .fuente_corporativa_contexto_actor_v1
        SELECT 'fuente:f0-b1-fk', f.fuente_version, f.audiencia_consumo,
               f.accion, f.tipo_efecto, 'clave:f0-b1-ausente',
               f.clave_version, f.revision_gobierno,
               f.huella_gobierno_sha256, f.emisor_id,
               f.configuracion_revision, f.configuracion_secuencia,
               f.huella_configuracion_sha256, f.raiz_clave_id,
               f.raiz_version, f.huella_raiz_spki_sha256,
               f.audiencia_despliegue, f.suite, f.valida_desde,
               f.valida_hasta, 'acto:f0-b1-fk', f.registrada_en
          FROM vec_autorizacion_atestada_v3
                   .fuente_corporativa_contexto_actor_v1 AS f
         WHERE f.fuente_ref=v_fuente_ref AND f.audiencia_consumo=v_audiencias[1];
        RAISE EXCEPTION USING ERRCODE='XX000',
            MESSAGE='B1: clave ausente aceptada';
    EXCEPTION WHEN foreign_key_violation THEN NULL;
    END;
    BEGIN
        INSERT INTO vec_autorizacion_atestada_v3
            .fuente_corporativa_contexto_actor_v1
        SELECT f.* FROM vec_autorizacion_atestada_v3
            .fuente_corporativa_contexto_actor_v1 AS f
         WHERE f.fuente_ref=v_fuente_ref AND f.audiencia_consumo=v_audiencias[1];
        RAISE EXCEPTION USING ERRCODE='XX000',
            MESSAGE='B1: duplicado de fuente aceptado';
    EXCEPTION WHEN unique_violation THEN NULL;
    END;
    IF (SELECT cp.revision FROM vec_autorizacion_atestada_v3
            .checkpoint_gobierno AS cp WHERE cp.control_id) <> v_revision_base
    THEN
        RAISE EXCEPTION USING ERRCODE='XX000',
            MESSAGE='B1: rechazo dejo avance de checkpoint';
    END IF;
    BEGIN
        UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno
           SET revision = 9007199254740991::pg_catalog.numeric
         WHERE control_id;
        INSERT INTO vec_autorizacion_atestada_v3
            .fuente_corporativa_contexto_actor_v1
        SELECT f.* FROM vec_autorizacion_atestada_v3
            .fuente_corporativa_contexto_actor_v1 AS f
         WHERE f.fuente_ref=v_fuente_ref
           AND f.audiencia_consumo=v_audiencias[1];
        RAISE EXCEPTION USING ERRCODE='XX000',
            MESSAGE='B1: checkpoint agotado aceptado';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
    END;
    IF (SELECT cp.revision FROM vec_autorizacion_atestada_v3
            .checkpoint_gobierno AS cp WHERE cp.control_id) <> v_revision_base
    THEN
        RAISE EXCEPTION USING ERRCODE='XX000',
            MESSAGE='B1: agotamiento no revirtio checkpoint';
    END IF;

    SELECT cp.revision INTO v_revision_base
      FROM vec_autorizacion_atestada_v3.checkpoint_gobierno AS cp
     WHERE cp.control_id;
    INSERT INTO vec_autorizacion_atestada_v3
        .revocacion_fuente_corporativa_contexto_actor_v1 VALUES (
        v_fuente_ref, 1, v_audiencias[1],
        '2027-01-01 00:00:00+00', 'motivo:f0-b1-revocacion',
        'acto:f0-b1-revocacion', pg_catalog.clock_timestamp()
    );
    IF NOT EXISTS (
        SELECT 1 FROM vec_autorizacion_atestada_v3.checkpoint_gobierno AS cp
         WHERE cp.control_id AND cp.revision = v_revision_base + 1
           AND cp.configuracion_secuencia_minima >= v_configuracion_secuencia
           AND cp.raiz_version_minima >= v_raiz_version
    ) THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'B1: revocacion no avanzo causalmente el checkpoint';
    END IF;

    SELECT cp.revision INTO v_revision_base
      FROM vec_autorizacion_atestada_v3.checkpoint_gobierno AS cp
     WHERE cp.control_id;
    BEGIN
        INSERT INTO vec_autorizacion_atestada_v3
            .revocacion_fuente_corporativa_contexto_actor_v1 VALUES (
            'fuente:f0-b1-ausente', 1, v_audiencias[1],
            '2027-01-01 00:00:00+00', 'motivo:f0-b1-ausente',
            'acto:f0-b1-ausente', pg_catalog.clock_timestamp()
        );
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'B1: revocacion sin fuente aceptada';
    EXCEPTION WHEN foreign_key_violation THEN NULL;
    END;
    IF (SELECT cp.revision FROM
            vec_autorizacion_atestada_v3.checkpoint_gobierno AS cp
           WHERE cp.control_id) <> v_revision_base THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'B1: fallo de FK dejo avance de checkpoint';
    END IF;

    BEGIN
        UPDATE vec_autorizacion_atestada_v3
            .fuente_corporativa_contexto_actor_v1
           SET acto_ref = acto_ref WHERE fuente_ref = v_fuente_ref;
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'B1: UPDATE de historia aceptado';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
    END;
    BEGIN
        DELETE FROM vec_autorizacion_atestada_v3
            .revocacion_fuente_corporativa_contexto_actor_v1
         WHERE fuente_ref = v_fuente_ref;
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'B1: DELETE de historia aceptado';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
    END;
    BEGIN
        EXECUTE 'TRUNCATE vec_autorizacion_atestada_v3.revocacion_fuente_corporativa_contexto_actor_v1';
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'B1: TRUNCATE de historia aceptado';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
    END;
END
$catalogo_y_checkpoint$;

GRANT SELECT ON
    vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1,
    vec_autorizacion_atestada_v3.revocacion_fuente_corporativa_contexto_actor_v1
TO vec_autorizacion_atestada_v3_migrador;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3
TO vec_autorizacion_atestada_v3_migrador;
SET LOCAL ROLE vec_autorizacion_atestada_v3_migrador;
DO $rls_denegacion$
BEGIN
    IF (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3
            .fuente_corporativa_contexto_actor_v1) <> 0
       OR (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3
            .revocacion_fuente_corporativa_contexto_actor_v1) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'B1: RLS expuso el catalogo a migrador';
    END IF;
END
$rls_denegacion$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON
    vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1,
    vec_autorizacion_atestada_v3.revocacion_fuente_corporativa_contexto_actor_v1
FROM vec_autorizacion_atestada_v3_migrador;
REVOKE ALL ON SCHEMA vec_autorizacion_atestada_v3
FROM vec_autorizacion_atestada_v3_migrador;

DO $locks_y_cierre$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_locks AS l
         WHERE l.pid = pg_catalog.pg_backend_pid() AND l.granted
           AND l.relation =
               'vec_autorizacion_atestada_v3.clave_capacidad_version'::pg_catalog.regclass
           AND l.mode = 'AccessExclusiveLock'
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_locks AS l
         WHERE l.pid = pg_catalog.pg_backend_pid() AND l.granted
           AND l.relation =
               'vec_autorizacion_atestada_v3.checkpoint_gobierno'::pg_catalog.regclass
           AND l.mode = 'RowExclusiveLock'
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_locks AS l
         WHERE l.pid = pg_catalog.pg_backend_pid() AND l.granted
           AND l.relation =
               'vec_autorizacion_atestada_v3.configuracion_raiz'::pg_catalog.regclass
           AND l.mode = 'ShareRowExclusiveLock'
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_prepared_xacts
    ) THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'B1: plan de locks o cierre transaccional invalido';
    END IF;
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_catalogo_fuente_b1_prueba() IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'B1: forma final del catalogo invalida';
    END IF;
END
$locks_y_cierre$;

DROP FUNCTION vec_autorizacion_atestada_v3
    .acreditar_forma_catalogo_fuente_b1_prueba();
