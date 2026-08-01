-- Matriz adversarial de las historias minimizadas F0-B2.
CREATE FUNCTION
vec_autorizacion_atestada_v3.acreditar_forma_atestacion_consumo_b2_prueba()
RETURNS pg_catalog.bool
LANGUAGE sql
VOLATILE
SET search_path = pg_catalog
AS $funcion$
    WITH propietario AS (
        SELECT r.oid FROM pg_catalog.pg_roles AS r
         WHERE r.rolname = 'vec_autorizacion_atestada_v3_propietario'
           AND NOT r.rolcanlogin AND NOT r.rolsuper
           AND NOT r.rolcreatedb AND NOT r.rolcreaterole
           AND NOT r.rolreplication AND NOT r.rolbypassrls
    ), relaciones AS (
        SELECT c.oid, c.relname, c.relowner, c.reltoastrelid FROM pg_catalog.pg_class AS c
          JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
         WHERE n.nspname = 'vec_autorizacion_atestada_v3'
           AND c.relname IN (
               'atestacion_fuente_corporativa_contexto_actor_v1',
               'consumo_fuente_corporativa_contexto_actor_v1'
           )
           AND c.relkind = 'r' AND c.relpersistence = 'p'
           AND c.relam = (
               SELECT a.oid FROM pg_catalog.pg_am AS a
                WHERE a.amname = 'heap' AND a.amtype = 't'
           )
           AND c.relnatts = CASE c.relname
               WHEN 'atestacion_fuente_corporativa_contexto_actor_v1' THEN 4
               ELSE 6
           END
           AND NOT c.relispartition AND c.reloftype = 0
           AND c.reloptions IS NULL AND c.relreplident = 'd'
           AND c.relrowsecurity AND c.relforcerowsecurity
           AND NOT c.relhasrules AND NOT c.relhassubclass
           AND NOT EXISTS (
               SELECT 1 FROM pg_catalog.pg_rewrite AS w
                WHERE w.ev_class = c.oid
           )
           AND NOT EXISTS (
               SELECT 1 FROM pg_catalog.pg_inherits AS h
                WHERE h.inhrelid = c.oid OR h.inhparent = c.oid
           )
           AND NOT EXISTS (
               SELECT 1 FROM pg_catalog.pg_publication_tables AS u
                WHERE u.schemaname = n.nspname AND u.tablename = c.relname
           )
           AND NOT EXISTS (
               SELECT 1 FROM pg_catalog.pg_publication AS u
                WHERE u.puballtables
           )
           AND NOT EXISTS (
               SELECT 1 FROM pg_catalog.pg_publication_namespace AS u
                WHERE u.pnnspid = n.oid
           )
           AND NOT EXISTS (
               SELECT 1 FROM pg_catalog.pg_publication_rel AS u
                WHERE u.prrelid = c.oid
           )
           AND NOT EXISTS (
               SELECT 1 FROM pg_catalog.pg_statistic_ext AS s
                WHERE s.stxrelid = c.oid
           )
    ), columnas_esperadas(
        relacion, posicion, nombre, tipo, colacion, almacenamiento) AS (
        VALUES
        ('atestacion_fuente_corporativa_contexto_actor_v1', 1, 'capacidad_ref', 'text',
         'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid, 'x'),
        ('atestacion_fuente_corporativa_contexto_actor_v1', 2, 'fuente_ref', 'text',
         'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid, 'x'),
        ('atestacion_fuente_corporativa_contexto_actor_v1', 3,
         'fuente_version', 'numeric(20,0)', 0::pg_catalog.oid, 'm'),
        ('atestacion_fuente_corporativa_contexto_actor_v1', 4, 'evento_fuente_ref', 'text',
         'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid, 'x'),
        ('consumo_fuente_corporativa_contexto_actor_v1', 1, 'capacidad_ref', 'text',
         'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid, 'x'),
        ('consumo_fuente_corporativa_contexto_actor_v1', 2, 'nonce', 'text',
         'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid, 'x'),
        ('consumo_fuente_corporativa_contexto_actor_v1', 3, 'operacion_ref', 'text',
         'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid, 'x'),
        ('consumo_fuente_corporativa_contexto_actor_v1', 4,
         'consumo_canonico', 'bytea', 0::pg_catalog.oid, 'x'),
        ('consumo_fuente_corporativa_contexto_actor_v1', 5, 'consumo_huella_sha256', 'text',
         'pg_catalog."C"'::pg_catalog.regcollation::pg_catalog.oid, 'x'),
        ('consumo_fuente_corporativa_contexto_actor_v1', 6,
         'consumida_en', 'timestamp(6) with time zone',
         0::pg_catalog.oid, 'p')
    ), columnas_exactas AS (
        SELECT a.attrelid, a.attnum FROM columnas_esperadas AS e
          JOIN pg_catalog.pg_class AS c ON c.relname = e.relacion
          JOIN pg_catalog.pg_namespace AS n
            ON n.oid = c.relnamespace
           AND n.nspname = 'vec_autorizacion_atestada_v3'
          JOIN pg_catalog.pg_attribute AS a
            ON a.attrelid = c.oid AND a.attnum = e.posicion
          LEFT JOIN pg_catalog.pg_attrdef AS d
            ON d.adrelid = a.attrelid AND d.adnum = a.attnum
         WHERE NOT a.attisdropped AND a.attname = e.nombre
           AND pg_catalog.format_type(a.atttypid, a.atttypmod) = e.tipo
           AND a.attnotnull AND a.attcollation = e.colacion
           AND a.attidentity = '' AND a.attgenerated = ''
           AND a.attacl IS NULL AND a.attstorage = e.almacenamiento
           AND a.attcompression = ''::pg_catalog."char"
           AND a.attstattarget IS NULL AND a.attoptions IS NULL
           AND a.attfdwoptions IS NULL AND a.attislocal
           AND a.attinhcount = 0 AND a.attndims = 0
           AND NOT a.atthasmissing AND a.attmissingval IS NULL
           AND NOT a.atthasdef AND d.oid IS NULL
    ), toast_exactos AS (
        SELECT r.oid FROM relaciones AS r
          JOIN pg_catalog.pg_class AS t ON t.oid = r.reltoastrelid
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.relnamespace
          JOIN pg_catalog.pg_am AS a ON a.oid = t.relam
         WHERE n.nspname = 'pg_toast'
           AND t.relname = 'pg_toast_' || r.oid::pg_catalog.text
           AND t.relowner = r.relowner AND t.relkind = 't'
           AND t.relpersistence = 'p' AND a.amname = 'heap'
           AND a.amtype = 't' AND t.reltablespace = 0
           AND t.reloptions IS NULL AND t.relacl IS NULL
           AND t.reltype = 0 AND t.reloftype = 0
           AND t.reltoastrelid = 0 AND t.relnatts = 3
           AND t.relhasindex AND t.relchecks = 0
           AND NOT t.relhasrules AND NOT t.relhastriggers
           AND NOT t.relhassubclass AND NOT t.relrowsecurity
           AND NOT t.relforcerowsecurity AND NOT t.relispartition
           AND t.relreplident = 'n'
           AND (
               SELECT pg_catalog.count(*)
                 FROM pg_catalog.pg_attribute AS x
                WHERE x.attrelid = t.oid AND x.attnum > 0
           ) = 3
           AND NOT EXISTS (
               SELECT 1
                 FROM pg_catalog.pg_attribute AS x
                 LEFT JOIN (VALUES
                     (1, 'chunk_id', 'oid'),
                     (2, 'chunk_seq', 'integer'),
                     (3, 'chunk_data', 'bytea')
                 ) AS e(posicion, nombre, tipo) ON e.posicion = x.attnum
                WHERE x.attrelid = t.oid AND x.attnum > 0
                  AND (e.posicion IS NULL OR x.attname <> e.nombre
                   OR pg_catalog.format_type(x.atttypid, x.atttypmod) <>
                      e.tipo OR x.attisdropped OR x.attnotnull
                   OR x.attstorage <> 'p'
                   OR x.attcompression <> ''::pg_catalog."char"
                   OR x.attstattarget IS NOT NULL
                   OR x.attoptions IS NOT NULL OR x.attfdwoptions IS NOT NULL
                   OR NOT x.attislocal OR x.attinhcount <> 0
                   OR x.attndims <> 0 OR x.atthasmissing
                   OR x.attmissingval IS NOT NULL OR x.atthasdef
                   OR x.attidentity <> '' OR x.attgenerated <> ''
                   OR x.attacl IS NOT NULL OR x.attcollation <> 0)
           )
           AND (
               SELECT pg_catalog.count(*) FROM pg_catalog.pg_index AS i
                WHERE i.indrelid = t.oid
           ) = 1
           AND EXISTS (
               SELECT 1
                 FROM pg_catalog.pg_index AS i
                 JOIN pg_catalog.pg_class AS c ON c.oid = i.indexrelid
                 JOIN pg_catalog.pg_namespace AS ni
                   ON ni.oid = c.relnamespace
                 JOIN pg_catalog.pg_am AS ai ON ai.oid = c.relam
                WHERE i.indrelid = t.oid AND ni.nspname = 'pg_toast'
                  AND c.relname = t.relname || '_index'
                  AND c.relowner = r.relowner AND c.relkind = 'i'
                  AND c.relpersistence = 'p' AND ai.amname = 'btree'
                  AND ai.amtype = 'i' AND c.reltablespace = 0
                  AND c.reloptions IS NULL AND c.relacl IS NULL
                  AND NOT c.relispartition AND c.relnatts = 2
                  AND i.indnatts = 2 AND i.indnkeyatts = 2
                  AND i.indkey::pg_catalog.text = '1 2'
                  AND i.indcollation::pg_catalog.text = '0 0'
                  AND i.indoption::pg_catalog.text = '0 0'
                  AND i.indisunique AND i.indisprimary
                  AND i.indimmediate AND i.indisvalid AND i.indisready
                  AND i.indislive AND NOT i.indisexclusion
                  AND NOT i.indisclustered AND NOT i.indisreplident
                  AND NOT i.indcheckxmin AND NOT i.indnullsnotdistinct
                  AND i.indpred IS NULL AND i.indexprs IS NULL
           )
    ), restricciones AS (
        SELECT c.* FROM pg_catalog.pg_constraint AS c
         WHERE c.conrelid IN (SELECT r.oid FROM relaciones AS r)
           AND c.contype <> 'n'
    ), indices_esperados(
        nombre, relacion, columnas, clases, colaciones, unica, primaria
    ) AS (
        VALUES
        ('f0_atestacion_fuente_pk',
         'atestacion_fuente_corporativa_contexto_actor_v1', '1',
         'pg_catalog.text_ops', 'pg_catalog.C', true, true),
        ('f0_atestacion_fuente_evento_uq',
         'atestacion_fuente_corporativa_contexto_actor_v1', '2 4',
         'pg_catalog.text_ops,pg_catalog.text_ops',
         'pg_catalog.C,pg_catalog.C', true, false),
        ('f0_consumo_fuente_pk',
         'consumo_fuente_corporativa_contexto_actor_v1', '1',
         'pg_catalog.text_ops', 'pg_catalog.C', true, true),
        ('f0_consumo_fuente_nonce_uq',
         'consumo_fuente_corporativa_contexto_actor_v1', '2',
         'pg_catalog.text_ops', 'pg_catalog.C', true, false),
        ('f0_consumo_fuente_operacion_uq',
         'consumo_fuente_corporativa_contexto_actor_v1', '3',
         'pg_catalog.text_ops', 'pg_catalog.C', true, false)
    ), indices AS (
        SELECT i.*, ci.relname AS nombre, ct.relname AS relacion,
               a.amname, ci.relkind AS clase_relacion,
               ci.relpersistence, ci.relispartition, ci.reloptions,
               ci.reltablespace, ci.relowner,
               (
                   SELECT pg_catalog.string_agg(
                       no.nspname || '.' || o.opcname, ',' ORDER BY u.ord
                   )
                     FROM pg_catalog.unnest(i.indclass)
                          WITH ORDINALITY AS u(oid, ord)
                     JOIN pg_catalog.pg_opclass AS o ON o.oid = u.oid
                     JOIN pg_catalog.pg_namespace AS no
                       ON no.oid = o.opcnamespace
               ) AS clases,
               (
                   SELECT pg_catalog.string_agg(
                       CASE WHEN u.oid = 0 THEN '0'
                            ELSE nc.nspname || '.' || co.collname END,
                       ',' ORDER BY u.ord
                   )
                     FROM pg_catalog.unnest(i.indcollation)
                          WITH ORDINALITY AS u(oid, ord)
                     LEFT JOIN pg_catalog.pg_collation AS co ON co.oid = u.oid
                     LEFT JOIN pg_catalog.pg_namespace AS nc
                       ON nc.oid = co.collnamespace
               ) AS colaciones
          FROM pg_catalog.pg_index AS i
          JOIN pg_catalog.pg_class AS ci ON ci.oid = i.indexrelid
          JOIN pg_catalog.pg_class AS ct ON ct.oid = i.indrelid
          JOIN pg_catalog.pg_am AS a ON a.oid = ci.relam
         WHERE i.indrelid IN (SELECT r.oid FROM relaciones AS r)
    ), disparadores_esperados(
        relacion, nombre, tipo, funcion, definicion
    ) AS (
        VALUES
        ('atestacion_fuente_corporativa_contexto_actor_v1',
         'f0_historia_inmutable', 27,
         'vec_autorizacion_atestada_v3.rechazar_mutacion()',
         'CREATE TRIGGER f0_historia_inmutable BEFORE DELETE OR UPDATE ON vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion()'),
        ('atestacion_fuente_corporativa_contexto_actor_v1',
         'f0_historia_no_truncable', 34,
         'vec_autorizacion_atestada_v3.rechazar_truncado()',
         'CREATE TRIGGER f0_historia_no_truncable BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado()'),
        ('consumo_fuente_corporativa_contexto_actor_v1',
         'f0_historia_inmutable', 27,
         'vec_autorizacion_atestada_v3.rechazar_mutacion()',
         'CREATE TRIGGER f0_historia_inmutable BEFORE DELETE OR UPDATE ON vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion()'),
        ('consumo_fuente_corporativa_contexto_actor_v1',
         'f0_historia_no_truncable', 34,
         'vec_autorizacion_atestada_v3.rechazar_truncado()',
         'CREATE TRIGGER f0_historia_no_truncable BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado()')
    ), disparadores AS (
        SELECT t.*, c.relname AS relacion,
               pg_catalog.pg_get_triggerdef(t.oid, false) AS definicion
          FROM pg_catalog.pg_trigger AS t
          JOIN pg_catalog.pg_class AS c ON c.oid = t.tgrelid
         WHERE t.tgrelid IN (SELECT r.oid FROM relaciones AS r)
           AND NOT t.tgisinternal
    ), claves_foraneas AS (
        SELECT c.oid, c.conname, c.conrelid, c.confrelid, c.conindid
          FROM restricciones AS c WHERE c.contype = 'f'
    ), disparadores_ri_esperados AS (
        SELECT c.oid, x.tabla, x.otra, c.conindid, x.tipo,
               'O'::pg_catalog.text, true, x.funcion::pg_catalog.text, true
          FROM claves_foraneas AS c
          CROSS JOIN LATERAL (VALUES
              (c.conrelid, c.confrelid, 5::pg_catalog.int2,
               'RI_FKey_check_ins'),
              (c.conrelid, c.confrelid, 17::pg_catalog.int2,
               'RI_FKey_check_upd'),
              (c.confrelid, c.conrelid, 9::pg_catalog.int2,
               'RI_FKey_noaction_del'),
              (c.confrelid, c.conrelid, 17::pg_catalog.int2,
               'RI_FKey_noaction_upd')
          ) AS x(tabla, otra, tipo, funcion)
    ), disparadores_ri_actuales AS (
        SELECT t.tgconstraint, t.tgrelid, t.tgconstrrelid,
               t.tgconstrindid, t.tgtype, t.tgenabled::pg_catalog.text,
               t.tgisinternal, p.proname::pg_catalog.text,
               ROW(t.tgparentid, t.tgdeferrable, t.tginitdeferred,
                   t.tgnargs, t.tgattr::pg_catalog.text,
                   pg_catalog.encode(t.tgargs, 'hex'), t.tgqual IS NULL,
                   t.tgoldtable IS NULL, t.tgnewtable IS NULL) =
               ROW(0::pg_catalog.oid, false, false, 0, '', '',
                   true, true, true)
          FROM pg_catalog.pg_trigger AS t
          JOIN pg_catalog.pg_proc AS p ON p.oid = t.tgfoid
         WHERE t.tgconstraint IN (SELECT c.oid FROM claves_foraneas AS c)
    ), clases_toast(oid, etiqueta) AS (
        SELECT t.oid, 'toast:' || r.relname
          FROM relaciones AS r
          JOIN pg_catalog.pg_class AS t ON t.oid = r.reltoastrelid
        UNION ALL
        SELECT i.indexrelid, 'indice_toast:' || r.relname
          FROM relaciones AS r
          JOIN pg_catalog.pg_index AS i ON i.indrelid = r.reltoastrelid
    ), objetos_b2(classid, objid, etiqueta) AS (
        SELECT 'pg_catalog.pg_class'::pg_catalog.regclass,
               r.oid, 'tabla:' || r.relname FROM relaciones AS r
        UNION
        SELECT 'pg_catalog.pg_type'::pg_catalog.regclass, t.oid,
               'tipo:' || n.nspname || '.' || t.typname
          FROM pg_catalog.pg_type AS t
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.typnamespace
         WHERE t.typrelid IN (SELECT r.oid FROM relaciones AS r)
            OR t.oid IN (
                SELECT x.typarray FROM pg_catalog.pg_type AS x
                 WHERE x.typrelid IN (SELECT r.oid FROM relaciones AS r)
            )
        UNION
        SELECT 'pg_catalog.pg_class'::pg_catalog.regclass,
               i.indexrelid, 'indice:' || i.nombre FROM indices AS i
        UNION
        SELECT 'pg_catalog.pg_constraint'::pg_catalog.regclass, c.oid,
               'restriccion:' ||
               c.conrelid::pg_catalog.regclass::pg_catalog.text || '.' ||
               c.conname FROM restricciones AS c
        UNION
        SELECT 'pg_catalog.pg_trigger'::pg_catalog.regclass, t.oid,
               'trigger:' || t.relacion || '.' || t.tgname
          FROM disparadores AS t
        UNION
        SELECT 'pg_catalog.pg_trigger'::pg_catalog.regclass, t.oid,
               'trigger_ri:' || c.conname || ':' ||
               t.tgrelid::pg_catalog.regclass::pg_catalog.text || ':' ||
               t.tgtype::pg_catalog.text || ':' || p.proname
          FROM pg_catalog.pg_trigger AS t
          JOIN claves_foraneas AS c ON c.oid = t.tgconstraint
          JOIN pg_catalog.pg_proc AS p ON p.oid = t.tgfoid
        UNION
        SELECT 'pg_catalog.pg_class'::pg_catalog.regclass,
               c.oid, c.etiqueta FROM clases_toast AS c
        UNION
        SELECT 'pg_catalog.pg_type'::pg_catalog.regclass, t.oid,
               'tipo_toast:' || n.nspname || '.' || t.typname
          FROM pg_catalog.pg_type AS t
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.typnamespace
         WHERE t.typrelid IN (SELECT c.oid FROM clases_toast AS c)
            OR t.oid IN (
                SELECT x.typarray FROM pg_catalog.pg_type AS x
                 WHERE x.typrelid IN (SELECT c.oid FROM clases_toast AS c)
            )
    ), dependencias_b2 AS (
        SELECT d.* FROM pg_catalog.pg_depend AS d
         WHERE EXISTS (
             SELECT 1 FROM objetos_b2 AS o
              WHERE (o.classid, o.objid) = (d.classid, d.objid)
                 OR (o.classid, o.objid) = (d.refclassid, d.refobjid)
         )
    ), resumen_dependencias AS (
        SELECT pg_catalog.count(*)::pg_catalog.text || '|' ||
               pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
                   pg_catalog.string_agg(
                       COALESCE(o.etiqueta, pg_catalog.pg_describe_object(
                           d.classid, d.objid, d.objsubid)) || ':' ||
                       d.objsubid::pg_catalog.text || '>' ||
                       d.deptype::pg_catalog.text || '>' ||
                       COALESCE(r.etiqueta, pg_catalog.pg_describe_object(
                           d.refclassid, d.refobjid, d.refobjsubid)) || ':' ||
                       d.refobjsubid::pg_catalog.text, E'\n' ORDER BY
                       d.classid, d.objid, d.objsubid, d.refclassid,
                       d.refobjid, d.refobjsubid, d.deptype
                   ), 'UTF8')), 'hex') AS firma
          FROM dependencias_b2 AS d
          LEFT JOIN objetos_b2 AS o
            ON (o.classid, o.objid) = (d.classid, d.objid)
          LEFT JOIN objetos_b2 AS r
            ON (r.classid, r.objid) = (d.refclassid, d.refobjid)
    )
    SELECT (SELECT pg_catalog.count(*) FROM propietario) = 1
       AND (SELECT pg_catalog.count(*) FROM relaciones) = 2
       AND NOT EXISTS (
           SELECT 1 FROM relaciones AS r, propietario AS p
            WHERE r.relowner <> p.oid
       )
       AND (SELECT pg_catalog.count(*) FROM columnas_esperadas) = 10
       AND (SELECT pg_catalog.count(*) FROM columnas_exactas) = 10
       AND (
           SELECT pg_catalog.count(*) FROM pg_catalog.pg_attribute AS a
            WHERE a.attrelid IN (SELECT r.oid FROM relaciones AS r)
              AND a.attnum > 0
       ) = 10
       AND NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_attribute AS a
            WHERE a.attrelid IN (SELECT r.oid FROM relaciones AS r)
              AND a.attnum > 0 AND a.attisdropped
       )
       AND (SELECT pg_catalog.count(*) FROM toast_exactos) = 2
       AND (SELECT pg_catalog.count(*) FROM restricciones) = 16
       AND NOT EXISTS (
           SELECT 1 FROM restricciones AS c
            WHERE NOT c.convalidated OR c.condeferrable OR c.condeferred
               OR NOT c.conislocal OR c.coninhcount <> 0
       )
       AND (
           SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               pg_catalog.string_agg(
                   c.conrelid::pg_catalog.regclass::pg_catalog.text || '|' ||
                   c.conname || '|' || c.contype::pg_catalog.text || '|' ||
                   pg_catalog.pg_get_constraintdef(c.oid), E'\n' ORDER BY
                   c.conrelid::pg_catalog.regclass::pg_catalog.text,
                   c.conname
               ), 'UTF8')), 'hex') FROM restricciones AS c
       ) = '2b1cfb02323b166b0510168486fbe6a0c48b16d5559f6886bf3ee7c15b4c93b3'
       AND (SELECT pg_catalog.count(*) FROM claves_foraneas) = 1
       AND EXISTS (
           SELECT 1 FROM claves_foraneas AS c
           JOIN pg_catalog.pg_constraint AS x ON x.oid = c.oid
            WHERE c.conname = 'f0_consumo_fuente_atestacion_fk'
              AND x.conkey = ARRAY[1]::pg_catalog.int2[]
              AND x.confrelid =
                  'vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1'::pg_catalog.regclass
              AND x.confkey = ARRAY[1]::pg_catalog.int2[]
              AND x.confmatchtype = 'f' AND x.confupdtype = 'a'
              AND x.confdeltype = 'a'
       )
       AND (SELECT pg_catalog.count(*) FROM indices) = 5
       AND NOT EXISTS (
           SELECT 1 FROM indices AS i
           FULL JOIN indices_esperados AS e ON e.nombre = i.nombre
            WHERE e.nombre IS NULL OR i.nombre IS NULL
               OR i.relacion <> e.relacion OR i.amname <> 'btree'
               OR i.clase_relacion <> 'i' OR i.relpersistence <> 'p'
               OR i.relispartition OR i.reloptions IS NOT NULL
               OR i.reltablespace <> 0
               OR i.relowner <> (SELECT oid FROM propietario)
               OR i.indkey::pg_catalog.text <> e.columnas
               OR i.clases <> e.clases OR i.colaciones <> e.colaciones
               OR i.indoption::pg_catalog.text <>
                  pg_catalog.repeat('0 ', i.indnatts - 1) || '0'
               OR i.indnatts <> pg_catalog.array_length(
                   pg_catalog.string_to_array(e.columnas, ' '), 1)
               OR i.indnkeyatts <> i.indnatts
               OR i.indisunique <> e.unica OR i.indisprimary <> e.primaria
               OR i.indnullsnotdistinct OR NOT i.indisvalid
               OR NOT i.indisready OR NOT i.indislive
               OR i.indisexclusion OR i.indisclustered
               OR i.indisreplident OR NOT i.indimmediate
               OR i.indcheckxmin OR i.indpred IS NOT NULL
               OR i.indexprs IS NOT NULL
       )
       AND (SELECT pg_catalog.count(*) FROM disparadores) = 4
       AND NOT EXISTS (
           SELECT 1 FROM disparadores AS t
           FULL JOIN disparadores_esperados AS e
             ON e.relacion = t.relacion AND e.nombre = t.tgname
            WHERE e.nombre IS NULL OR t.tgname IS NULL
               OR t.tgtype <> e.tipo
               OR t.tgfoid <> e.funcion::pg_catalog.regprocedure
               OR t.tgenabled <> 'O' OR t.tgqual IS NOT NULL
               OR t.tgnargs <> 0 OR pg_catalog.octet_length(t.tgargs) <> 0
               OR t.tgattr::pg_catalog.text <> ''
               OR t.tgoldtable IS NOT NULL OR t.tgnewtable IS NOT NULL
               OR t.tgparentid <> 0 OR t.tgconstrrelid <> 0
               OR t.tgconstraint <> 0 OR t.tgdeferrable
               OR t.tginitdeferred OR t.definicion <> e.definicion
       )
       AND NOT EXISTS (
           (SELECT * FROM disparadores_ri_esperados EXCEPT ALL
            SELECT * FROM disparadores_ri_actuales)
           UNION ALL
           (SELECT * FROM disparadores_ri_actuales EXCEPT ALL
            SELECT * FROM disparadores_ri_esperados)
       )
       AND NOT EXISTS (
           SELECT 1 FROM dependencias_b2 AS d WHERE d.deptype IN ('e', 'x')
       )
       AND (SELECT r.firma FROM resumen_dependencias AS r) =
           '78|7b7b1b891ca4bc902ee92cebfc50fe76532cd277f72543029c4e0f18ac17d788'
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
               (SELECT c.relacl FROM pg_catalog.pg_class AS c
                 WHERE c.oid = r.oid),
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
       AND (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_type AS b
             JOIN pg_catalog.pg_type AS a ON a.oid = b.typarray,
                  propietario AS o
            WHERE b.typrelid IN (SELECT r.oid FROM relaciones AS r)
              AND b.typowner = o.oid AND a.typowner = o.oid
              AND ROW(a.typelem, a.typarray, a.typrelid, a.typcategory,
                      a.typtype, a.typisdefined, a.typacl) IS NOT DISTINCT FROM
                  ROW(b.oid, 0::pg_catalog.oid, 0::pg_catalog.oid, 'A', 'b',
                      true, NULL::pg_catalog.aclitem[])
       ) = 2
       AND NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_type AS b
             JOIN pg_catalog.pg_type AS a ON a.oid = b.typarray
             CROSS JOIN (VALUES
                 ('vec_autorizacion_atestada_v3_migrador'),
                 ('vec_autorizacion_atestada_v3_emisor'),
                 ('vec_autorizacion_atestada_v3_consumidor'),
                 ('vec_contratacion_temporal_propietario'),
                 ('vec_contexto_actor_v1_propietario')
             ) AS d(rol)
            WHERE b.typrelid IN (SELECT r.oid FROM relaciones AS r)
              AND (
                  pg_catalog.has_type_privilege(d.rol, b.oid, 'USAGE')
                  OR pg_catalog.has_type_privilege(d.rol, a.oid, 'USAGE')
              )
       )
$funcion$;
DO $formas_hostiles$
BEGIN
    BEGIN
        EXECUTE 'GRANT SELECT ON vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1 TO vec_autorizacion_atestada_v3_migrador';
        IF vec_autorizacion_atestada_v3
               .acreditar_forma_atestacion_consumo_b2_prueba() IS NOT FALSE
        THEN RAISE EXCEPTION 'B2: ACL hostil no detectada' USING ERRCODE = 'XX000'; END IF;
        RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    BEGIN
        EXECUTE 'GRANT USAGE ON TYPE vec_autorizacion_atestada_v3.' ||
                'atestacion_fuente_corporativa_contexto_actor_v1 ' ||
                'TO vec_autorizacion_atestada_v3_migrador';
        IF vec_autorizacion_atestada_v3
               .acreditar_forma_atestacion_consumo_b2_prueba() IS NOT FALSE
           OR pg_catalog.has_type_privilege(
               'vec_autorizacion_atestada_v3_migrador',
               'vec_autorizacion_atestada_v3.' ||
               '_atestacion_fuente_corporativa_contexto_actor_v1',
               'USAGE'
           ) IS NOT TRUE THEN
            RAISE EXCEPTION 'B2: ACL derivada de array no detectada' USING ERRCODE = 'XX000';
        END IF;
        RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    BEGIN
        EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1 DISABLE ROW LEVEL SECURITY';
        IF vec_autorizacion_atestada_v3
               .acreditar_forma_atestacion_consumo_b2_prueba() IS NOT FALSE
        THEN RAISE EXCEPTION 'B2: RLS hostil no detectada' USING ERRCODE = 'XX000'; END IF;
        RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    BEGIN
        EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1 DROP CONSTRAINT f0_consumo_fuente_nonce_uq';
        EXECUTE 'CREATE UNIQUE INDEX f0_consumo_fuente_nonce_uq ON vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1 USING btree (operacion_ref)';
        IF vec_autorizacion_atestada_v3
               .acreditar_forma_atestacion_consumo_b2_prueba() IS NOT FALSE
        THEN RAISE EXCEPTION 'B2: indice hostil no detectado' USING ERRCODE = 'XX000'; END IF;
        RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    BEGIN
        EXECUTE 'DROP TRIGGER f0_historia_inmutable ON vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1';
        EXECUTE 'CREATE TRIGGER f0_historia_inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1 FOR EACH ROW WHEN (false) EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion()';
        IF vec_autorizacion_atestada_v3
               .acreditar_forma_atestacion_consumo_b2_prueba() IS NOT FALSE
        THEN RAISE EXCEPTION 'B2: trigger hostil no detectado' USING ERRCODE = 'XX000'; END IF;
        RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    BEGIN
        EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1 DROP CONSTRAINT f0_atestacion_fuente_version_ck';
        EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1 ADD CONSTRAINT f0_atestacion_fuente_version_ck CHECK (true)';
        IF vec_autorizacion_atestada_v3
               .acreditar_forma_atestacion_consumo_b2_prueba() IS NOT FALSE
        THEN RAISE EXCEPTION 'B2: restriccion hostil no detectada' USING ERRCODE = 'XX000'; END IF;
        RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    BEGIN
        EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1 ADD COLUMN f0_columna_hostil text';
        EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1 DROP COLUMN f0_columna_hostil';
        IF vec_autorizacion_atestada_v3
               .acreditar_forma_atestacion_consumo_b2_prueba() IS NOT FALSE
        THEN RAISE EXCEPTION 'B2: columna fisica caida no detectada' USING ERRCODE = 'XX000'; END IF;
        RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    BEGIN
        EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1 ALTER COLUMN nonce SET STATISTICS 100';
        IF vec_autorizacion_atestada_v3
               .acreditar_forma_atestacion_consumo_b2_prueba() IS NOT FALSE
        THEN RAISE EXCEPTION 'B2: estadistica de columna no detectada' USING ERRCODE = 'XX000'; END IF;
        RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    BEGIN
        EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1 ALTER COLUMN nonce SET STORAGE PLAIN';
        IF vec_autorizacion_atestada_v3
               .acreditar_forma_atestacion_consumo_b2_prueba() IS NOT FALSE
        THEN RAISE EXCEPTION 'B2: almacenamiento no detectado' USING ERRCODE = 'XX000'; END IF;
        RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    BEGIN
        EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1 ALTER COLUMN nonce SET COMPRESSION pglz';
        IF vec_autorizacion_atestada_v3
               .acreditar_forma_atestacion_consumo_b2_prueba() IS NOT FALSE
        THEN RAISE EXCEPTION 'B2: compresion no detectada' USING ERRCODE = 'XX000'; END IF;
        RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    BEGIN
        EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1 SET (toast.autovacuum_enabled=false)';
        IF vec_autorizacion_atestada_v3
               .acreditar_forma_atestacion_consumo_b2_prueba() IS NOT FALSE
        THEN RAISE EXCEPTION 'B2: opcion TOAST no detectada' USING ERRCODE = 'XX000'; END IF;
        RAISE no_data_found;
    EXCEPTION WHEN no_data_found THEN NULL; END;
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_atestacion_consumo_b2_prueba() IS NOT TRUE THEN
        RAISE EXCEPTION 'B2: forma hostil no fue restaurada' USING ERRCODE = 'XX000';
    END IF;
END
$formas_hostiles$;
DO $historias$
DECLARE
    v_capacidad_uno constant pg_catalog.text :=
        'cfc_' || pg_catalog.repeat('1', 64);
    v_capacidad_dos constant pg_catalog.text :=
        'cfc_' || pg_catalog.repeat('2', 64);
    v_canon_uno pg_catalog.bytea :=
        pg_catalog.decode(pg_catalog.repeat('a1', 512), 'hex');
    v_canon_dos pg_catalog.bytea :=
        pg_catalog.decode(pg_catalog.repeat('b2', 512), 'hex');
BEGIN
    INSERT INTO vec_autorizacion_atestada_v3
        .atestacion_fuente_corporativa_contexto_actor_v1
    VALUES (
        v_capacidad_uno, 'fuente:f0-b2-prueba', 7,
        'evento:f0-b2-prueba-uno'
    );
    INSERT INTO vec_autorizacion_atestada_v3
        .consumo_fuente_corporativa_contexto_actor_v1
    VALUES (
        v_capacidad_uno, pg_catalog.repeat('a', 64),
        'oca_' || pg_catalog.repeat('A', 24), v_canon_uno,
        pg_catalog.encode(pg_catalog.sha256(v_canon_uno), 'hex'),
        '2026-08-01 10:00:00+00'
    );
    INSERT INTO vec_autorizacion_atestada_v3
        .atestacion_fuente_corporativa_contexto_actor_v1
    VALUES (
        v_capacidad_dos, 'fuente:f0-b2-prueba', 8,
        'evento:f0-b2-prueba-dos'
    );
    IF NOT EXISTS (
        SELECT 1 FROM vec_autorizacion_atestada_v3
            .atestacion_fuente_corporativa_contexto_actor_v1 AS a
        JOIN vec_autorizacion_atestada_v3
            .consumo_fuente_corporativa_contexto_actor_v1 AS c
          ON c.capacidad_ref = a.capacidad_ref
         WHERE a.capacidad_ref = v_capacidad_uno
           AND a.fuente_version = 7
           AND c.consumo_canonico = v_canon_uno
           AND c.consumo_huella_sha256 =
               pg_catalog.encode(pg_catalog.sha256(v_canon_uno), 'hex')
           AND c.consumida_en = '2026-08-01 10:00:00+00'
    ) THEN RAISE EXCEPTION 'B2: historia nominal no persistida' USING ERRCODE = 'XX000'; END IF;

    BEGIN
        INSERT INTO vec_autorizacion_atestada_v3
            .atestacion_fuente_corporativa_contexto_actor_v1
        VALUES (
            v_capacidad_uno, 'fuente:f0-b2-distinta', 9,
            'evento:f0-b2-distinto'
        );
        RAISE EXCEPTION 'B2: capacidad duplicada aceptada' USING ERRCODE = 'XX000';
    EXCEPTION WHEN unique_violation THEN NULL; END;
    BEGIN
        INSERT INTO vec_autorizacion_atestada_v3
            .atestacion_fuente_corporativa_contexto_actor_v1
        VALUES (
            'cfc_' || pg_catalog.repeat('3', 64),
            'fuente:f0-b2-prueba', 9, 'evento:f0-b2-prueba-uno'
        );
        RAISE EXCEPTION 'B2: evento estable duplicado aceptado' USING ERRCODE = 'XX000';
    EXCEPTION WHEN unique_violation THEN NULL; END;
    BEGIN
        INSERT INTO vec_autorizacion_atestada_v3
            .consumo_fuente_corporativa_contexto_actor_v1
        VALUES (
            'cfc_' || pg_catalog.repeat('4', 64),
            pg_catalog.repeat('c', 64),
            'oca_' || pg_catalog.repeat('C', 24), v_canon_dos,
            pg_catalog.encode(pg_catalog.sha256(v_canon_dos), 'hex'),
            '2026-08-01 10:00:01+00'
        );
        RAISE EXCEPTION 'B2: consumo sin atestacion aceptado' USING ERRCODE = 'XX000';
    EXCEPTION WHEN foreign_key_violation THEN NULL; END;
    BEGIN
        INSERT INTO vec_autorizacion_atestada_v3
            .consumo_fuente_corporativa_contexto_actor_v1
        VALUES (
            v_capacidad_dos, pg_catalog.repeat('a', 64),
            'oca_' || pg_catalog.repeat('B', 24), v_canon_dos,
            pg_catalog.encode(pg_catalog.sha256(v_canon_dos), 'hex'),
            '2026-08-01 10:00:01+00'
        );
        RAISE EXCEPTION 'B2: nonce duplicado aceptado' USING ERRCODE = 'XX000';
    EXCEPTION WHEN unique_violation THEN NULL; END;
    BEGIN
        INSERT INTO vec_autorizacion_atestada_v3
            .consumo_fuente_corporativa_contexto_actor_v1
        VALUES (
            v_capacidad_dos, pg_catalog.repeat('b', 64),
            'oca_' || pg_catalog.repeat('A', 24), v_canon_dos,
            pg_catalog.encode(pg_catalog.sha256(v_canon_dos), 'hex'),
            '2026-08-01 10:00:01+00'
        );
        RAISE EXCEPTION 'B2: operacion duplicada aceptada' USING ERRCODE = 'XX000';
    EXCEPTION WHEN unique_violation THEN NULL; END;
    BEGIN
        INSERT INTO vec_autorizacion_atestada_v3
            .consumo_fuente_corporativa_contexto_actor_v1
        VALUES (
            v_capacidad_dos, pg_catalog.repeat('b', 64),
            'oca_' || pg_catalog.repeat('B', 24), v_canon_dos,
            pg_catalog.repeat('d', 64), '2026-08-01 10:00:01+00'
        );
        RAISE EXCEPTION 'B2: huella de canon cruzada aceptada' USING ERRCODE = 'XX000';
    EXCEPTION WHEN check_violation THEN NULL; END;
    BEGIN
        UPDATE vec_autorizacion_atestada_v3
            .atestacion_fuente_corporativa_contexto_actor_v1
           SET fuente_version = fuente_version
         WHERE capacidad_ref = v_capacidad_uno;
        RAISE EXCEPTION 'B2: UPDATE de atestacion aceptado' USING ERRCODE = 'XX000';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
    BEGIN
        DELETE FROM vec_autorizacion_atestada_v3
            .consumo_fuente_corporativa_contexto_actor_v1
         WHERE capacidad_ref = v_capacidad_uno;
        RAISE EXCEPTION 'B2: DELETE de consumo aceptado' USING ERRCODE = 'XX000';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
    BEGIN
        EXECUTE 'TRUNCATE vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1';
        RAISE EXCEPTION 'B2: TRUNCATE de consumo aceptado' USING ERRCODE = 'XX000';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
END
$historias$;
GRANT SELECT ON
    vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1,
    vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1
TO vec_autorizacion_atestada_v3_migrador;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3
TO vec_autorizacion_atestada_v3_migrador;
SET LOCAL ROLE vec_autorizacion_atestada_v3_migrador;
DO $rls_denegacion$
BEGIN
    IF (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3
            .atestacion_fuente_corporativa_contexto_actor_v1) <> 0
       OR (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3
            .consumo_fuente_corporativa_contexto_actor_v1) <> 0 THEN
        RAISE EXCEPTION 'B2: RLS expuso historias al migrador' USING ERRCODE = 'XX000';
    END IF;
END
$rls_denegacion$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON
    vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1,
    vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1
FROM vec_autorizacion_atestada_v3_migrador;
REVOKE ALL ON SCHEMA vec_autorizacion_atestada_v3
FROM vec_autorizacion_atestada_v3_migrador;
DO $cierre$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_atestacion_consumo_b2_prueba() IS NOT TRUE THEN
        RAISE EXCEPTION 'B2: forma final de historias invalida' USING ERRCODE = 'XX000';
    END IF;
END
$cierre$;
DROP FUNCTION vec_autorizacion_atestada_v3
    .acreditar_forma_atestacion_consumo_b2_prueba();
