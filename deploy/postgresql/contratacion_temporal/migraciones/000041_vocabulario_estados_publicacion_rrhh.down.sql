-- Reversión segura CT-000041A: nunca descarta estados ya publicados.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_05:consultas_rrhh:migraciones', 0
));
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema = 21
 FOR UPDATE;
SELECT control
  FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
 WHERE control AND version_esquema = 5
 FOR UPDATE;
LOCK TABLE vec_contratacion_temporal.publicacion_version_rrhh
IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_contratacion_temporal
    .control_vocabulario_estados_publicacion_rrhh_v1
IN ACCESS EXCLUSIVE MODE;

CREATE FUNCTION
vec_contratacion_temporal.material_estructura_relacion_ct000041a(
    p_relacion regclass
)
RETURNS text
LANGUAGE sql
STABLE
SET search_path = pg_catalog
AS $funcion$
WITH relacion AS (
    SELECT tabla.*, propietario.rolname AS propietario,
           acceso.amname AS metodo_acceso,
           espacio.spcname AS espacio_tablas,
           tabla.relacl IS NULL AS acl_nula,
           COALESCE((
               SELECT pg_catalog.jsonb_agg(
                          permiso::text ORDER BY permiso::text
                      )
                 FROM pg_catalog.unnest(tabla.relacl) permiso
           ), '[]'::jsonb) AS acl,
           tabla.reloptions IS NULL AS opciones_nulas,
           COALESCE((
               SELECT pg_catalog.jsonb_agg(opcion ORDER BY opcion)
                 FROM pg_catalog.unnest(tabla.reloptions) opcion
           ), '[]'::jsonb) AS opciones
      FROM pg_catalog.pg_class tabla
      JOIN pg_catalog.pg_roles propietario
        ON propietario.oid = tabla.relowner
 LEFT JOIN pg_catalog.pg_am acceso ON acceso.oid = tabla.relam
 LEFT JOIN pg_catalog.pg_tablespace espacio
        ON espacio.oid = tabla.reltablespace
     WHERE tabla.oid = p_relacion
), columnas AS (
    SELECT COALESCE(pg_catalog.jsonb_agg(
               pg_catalog.jsonb_build_array(
                   atributo.attnum, atributo.attname,
                   pg_catalog.format_type(
                       atributo.atttypid, atributo.atttypmod
                   ),
                   atributo.attnotnull, atributo.atthasdef,
                   COALESCE(pg_catalog.pg_get_expr(
                       defecto.adbin, defecto.adrelid, false
                   ), ''),
                   atributo.attidentity, atributo.attgenerated,
                   atributo.attstorage, atributo.attcompression,
                   atributo.attstattarget, atributo.attndims,
                   atributo.attislocal, atributo.attinhcount,
                   atributo.atthasmissing,
                   COALESCE(atributo.attmissingval::text, ''),
                   COALESCE(
                       esquema_colacion.nspname || '.' || colacion.collname,
                       ''
                   ),
                   atributo.attacl IS NULL,
                   COALESCE((
                       SELECT pg_catalog.jsonb_agg(
                                  permiso::text ORDER BY permiso::text
                              )
                         FROM pg_catalog.unnest(atributo.attacl) permiso
                   ), '[]'::jsonb),
                   atributo.attoptions IS NULL,
                   COALESCE((
                       SELECT pg_catalog.jsonb_agg(opcion ORDER BY opcion)
                         FROM pg_catalog.unnest(atributo.attoptions) opcion
                   ), '[]'::jsonb),
                   atributo.attfdwoptions IS NULL,
                   COALESCE((
                       SELECT pg_catalog.jsonb_agg(opcion ORDER BY opcion)
                         FROM pg_catalog.unnest(atributo.attfdwoptions) opcion
                   ), '[]'::jsonb)
               ) ORDER BY atributo.attnum
           ), '[]'::jsonb) AS valor
      FROM pg_catalog.pg_attribute atributo
 LEFT JOIN pg_catalog.pg_attrdef defecto
        ON defecto.adrelid = atributo.attrelid
       AND defecto.adnum = atributo.attnum
 LEFT JOIN pg_catalog.pg_collation colacion
        ON colacion.oid = atributo.attcollation
 LEFT JOIN pg_catalog.pg_namespace esquema_colacion
        ON esquema_colacion.oid = colacion.collnamespace
     WHERE atributo.attrelid = p_relacion
       AND atributo.attnum > 0
       AND NOT atributo.attisdropped
), restricciones AS (
    SELECT COALESCE(pg_catalog.jsonb_agg(
               pg_catalog.jsonb_build_array(
                   restriccion.conname, restriccion.contype,
                   restriccion.convalidated, restriccion.conenforced,
                   restriccion.conislocal, restriccion.connoinherit,
                   restriccion.condeferrable, restriccion.condeferred,
                   restriccion.coninhcount,
                   restriccion.conparentid = 0,
                   COALESCE(restriccion.conkey::text, ''),
                   COALESCE(restriccion.confrelid::regclass::text, ''),
                   COALESCE(restriccion.confkey::text, ''),
                   restriccion.confupdtype, restriccion.confdeltype,
                   restriccion.confmatchtype, restriccion.conperiod,
                   pg_catalog.pg_get_constraintdef(
                       restriccion.oid, false
                   )
               ) ORDER BY restriccion.conname
           ), '[]'::jsonb) AS valor
      FROM pg_catalog.pg_constraint restriccion
     WHERE restriccion.conrelid = p_relacion
), indices AS (
    SELECT COALESCE(pg_catalog.jsonb_agg(
               pg_catalog.jsonb_build_array(
                   clase.relname, propietario.rolname, acceso.amname,
                   COALESCE(espacio.spcname, ''), clase.relpersistence,
                   clase.relacl IS NULL,
                   COALESCE((
                       SELECT pg_catalog.jsonb_agg(
                                  permiso::text ORDER BY permiso::text
                              )
                         FROM pg_catalog.unnest(clase.relacl) permiso
                   ), '[]'::jsonb),
                   clase.reloptions IS NULL,
                   COALESCE((
                       SELECT pg_catalog.jsonb_agg(opcion ORDER BY opcion)
                         FROM pg_catalog.unnest(clase.reloptions) opcion
                   ), '[]'::jsonb),
                   indice.indisunique, indice.indnullsnotdistinct,
                   indice.indisprimary, indice.indisvalid,
                   indice.indisready, indice.indislive,
                   indice.indisreplident, indice.indisclustered,
                   indice.indimmediate, indice.indcheckxmin,
                   indice.indnkeyatts, indice.indnatts,
                   pg_catalog.pg_get_indexdef(indice.indexrelid)
               ) ORDER BY clase.relname
           ), '[]'::jsonb) AS valor
      FROM pg_catalog.pg_index indice
      JOIN pg_catalog.pg_class clase ON clase.oid = indice.indexrelid
      JOIN pg_catalog.pg_roles propietario
        ON propietario.oid = clase.relowner
      JOIN pg_catalog.pg_am acceso ON acceso.oid = clase.relam
 LEFT JOIN pg_catalog.pg_tablespace espacio
        ON espacio.oid = clase.reltablespace
     WHERE indice.indrelid = p_relacion
), disparadores AS (
    SELECT COALESCE(pg_catalog.jsonb_agg(
               pg_catalog.jsonb_build_array(
                   disparador.tgname, disparador.tgtype,
                   disparador.tgenabled, disparador.tgisinternal,
                   disparador.tgfoid::regprocedure::text,
                   pg_catalog.encode(disparador.tgargs, 'hex'),
                   disparador.tgconstraint = 0,
                   disparador.tgparentid = 0,
                   pg_catalog.pg_get_triggerdef(disparador.oid, false),
                   propietario_funcion.rolname,
                   lenguaje.lanname,
                   funcion.prokind, funcion.provolatile,
                   funcion.proparallel, funcion.prosecdef,
                   funcion.proleakproof, funcion.proisstrict,
                   funcion.proretset, funcion.prorettype::regtype::text,
                   funcion.procost, funcion.prorows,
                   funcion.proconfig IS NULL,
                   COALESCE((
                       SELECT pg_catalog.jsonb_agg(opcion ORDER BY opcion)
                         FROM pg_catalog.unnest(funcion.proconfig) opcion
                   ), '[]'::jsonb),
                   funcion.proacl IS NULL,
                   COALESCE((
                       SELECT pg_catalog.jsonb_agg(
                                  permiso::text ORDER BY permiso::text
                              )
                         FROM pg_catalog.unnest(funcion.proacl) permiso
                   ), '[]'::jsonb),
                   pg_catalog.pg_get_functiondef(funcion.oid)
               ) ORDER BY disparador.tgname
           ), '[]'::jsonb) AS valor
      FROM pg_catalog.pg_trigger disparador
      JOIN pg_catalog.pg_proc funcion
        ON funcion.oid = disparador.tgfoid
      JOIN pg_catalog.pg_roles propietario_funcion
        ON propietario_funcion.oid = funcion.proowner
      JOIN pg_catalog.pg_language lenguaje
        ON lenguaje.oid = funcion.prolang
     WHERE disparador.tgrelid = p_relacion
       AND NOT disparador.tgisinternal
), politicas AS (
    SELECT COALESCE(pg_catalog.jsonb_agg(
               pg_catalog.jsonb_build_array(
                   politica.polname, politica.polcmd,
                   politica.polpermissive,
                   COALESCE((
                       SELECT pg_catalog.jsonb_agg(
                                  rol.rolname ORDER BY rol.rolname
                              )
                         FROM pg_catalog.unnest(
                                  politica.polroles
                              ) rol_oid(oid)
                         JOIN pg_catalog.pg_roles rol
                           ON rol.oid = rol_oid.oid
                   ), '[]'::jsonb),
                   COALESCE(pg_catalog.pg_get_expr(
                       politica.polqual, politica.polrelid, false
                   ), ''),
                   COALESCE(pg_catalog.pg_get_expr(
                       politica.polwithcheck, politica.polrelid, false
                   ), '')
               ) ORDER BY politica.polname
           ), '[]'::jsonb) AS valor
      FROM pg_catalog.pg_policy politica
     WHERE politica.polrelid = p_relacion
), reglas AS (
    SELECT COALESCE(pg_catalog.jsonb_agg(
               pg_catalog.jsonb_build_array(
                   regla.rulename, regla.ev_type, regla.ev_enabled,
                   regla.is_instead,
                   pg_catalog.pg_get_ruledef(regla.oid, false)
               ) ORDER BY regla.rulename
           ), '[]'::jsonb) AS valor
      FROM pg_catalog.pg_rewrite regla
     WHERE regla.ev_class = p_relacion
), herencia AS (
    SELECT COALESCE(pg_catalog.jsonb_agg(
               pg_catalog.jsonb_build_array(
                   vinculo.inhrelid::regclass::text,
                   vinculo.inhparent::regclass::text, vinculo.inhseqno
               ) ORDER BY vinculo.inhrelid::regclass::text,
                          vinculo.inhparent::regclass::text,
                          vinculo.inhseqno
           ), '[]'::jsonb) AS valor
      FROM pg_catalog.pg_inherits vinculo
     WHERE vinculo.inhrelid = p_relacion
        OR vinculo.inhparent = p_relacion
), publicaciones AS (
    SELECT COALESCE(pg_catalog.jsonb_agg(
               pg_catalog.jsonb_build_array(
                   publicacion.pubname,
                   COALESCE(pertenencia.prattrs::text, ''),
                   COALESCE(pg_catalog.pg_get_expr(
                       pertenencia.prqual, pertenencia.prrelid, false
                   ), '')
               ) ORDER BY publicacion.pubname
           ), '[]'::jsonb) AS valor
      FROM pg_catalog.pg_publication_rel pertenencia
      JOIN pg_catalog.pg_publication publicacion
        ON publicacion.oid = pertenencia.prpubid
     WHERE pertenencia.prrelid = p_relacion
), estadisticas AS (
    SELECT COALESCE(pg_catalog.jsonb_agg(
               pg_catalog.jsonb_build_array(
                   estadistica.stxname, propietario.rolname,
                   pg_catalog.pg_get_statisticsobjdef(estadistica.oid)
               ) ORDER BY estadistica.stxname
           ), '[]'::jsonb) AS valor
      FROM pg_catalog.pg_statistic_ext estadistica
      JOIN pg_catalog.pg_roles propietario
        ON propietario.oid = estadistica.stxowner
     WHERE estadistica.stxrelid = p_relacion
), etiquetas AS (
    SELECT COALESCE(pg_catalog.jsonb_agg(
               pg_catalog.jsonb_build_array(
                   etiqueta.objsubid, etiqueta.provider, etiqueta.label
               ) ORDER BY etiqueta.objsubid, etiqueta.provider
           ), '[]'::jsonb) AS valor
      FROM pg_catalog.pg_seclabel etiqueta
     WHERE etiqueta.classoid = 'pg_catalog.pg_class'::regclass
       AND etiqueta.objoid = p_relacion
), comentarios AS (
    SELECT COALESCE(pg_catalog.jsonb_agg(
               pg_catalog.jsonb_build_array(
                   comentario.objsubid, comentario.description
               ) ORDER BY comentario.objsubid
           ), '[]'::jsonb) AS valor
     FROM pg_catalog.pg_description comentario
     WHERE comentario.classoid = 'pg_catalog.pg_class'::regclass
       AND comentario.objoid = p_relacion
), tipos_relacion AS (
    SELECT COALESCE(pg_catalog.jsonb_agg(
               pg_catalog.jsonb_build_array(
                   pg_catalog.format_type(tipo.oid, NULL),
                   propietario.rolname,
                   tipo.typtype, tipo.typcategory,
                   tipo.typispreferred, tipo.typisdefined,
                   tipo.typdelim, tipo.typnotnull,
                   tipo.typbyval, tipo.typalign, tipo.typstorage,
                   tipo.typndims, tipo.typtypmod,
                   COALESCE(pg_catalog.format_type(
                       NULLIF(tipo.typelem, 0), NULL
                   ), ''),
                   COALESCE(pg_catalog.format_type(
                       NULLIF(tipo.typarray, 0), NULL
                   ), ''),
                   tipo.typinput::regprocedure::text,
                   tipo.typoutput::regprocedure::text,
                   tipo.typreceive::regprocedure::text,
                   tipo.typsend::regprocedure::text,
                   tipo.typmodin::regprocedure::text,
                   tipo.typmodout::regprocedure::text,
                   tipo.typanalyze::regprocedure::text,
                   tipo.typsubscript::regprocedure::text,
                   tipo.typacl IS NULL,
                   COALESCE((
                       SELECT pg_catalog.jsonb_agg(
                                  permiso::text ORDER BY permiso::text
                              )
                         FROM pg_catalog.unnest(tipo.typacl) permiso
                   ), '[]'::jsonb),
                   COALESCE(pg_catalog.obj_description(
                       tipo.oid, 'pg_type'
                   ), ''),
                   COALESCE((
                       SELECT pg_catalog.jsonb_agg(
                                  pg_catalog.jsonb_build_array(
                                      etiqueta.provider, etiqueta.label
                                  ) ORDER BY etiqueta.provider
                              )
                         FROM pg_catalog.pg_seclabel etiqueta
                        WHERE etiqueta.classoid =
                              'pg_catalog.pg_type'::regclass
                          AND etiqueta.objoid = tipo.oid
                   ), '[]'::jsonb)
               ) ORDER BY pg_catalog.format_type(tipo.oid, NULL)
           ), '[]'::jsonb) AS valor
      FROM pg_catalog.pg_class tabla
      JOIN pg_catalog.pg_type fila ON fila.oid = tabla.reltype
      JOIN pg_catalog.pg_type tipo
        ON tipo.oid = fila.oid OR tipo.oid = fila.typarray
      JOIN pg_catalog.pg_roles propietario
        ON propietario.oid = tipo.typowner
     WHERE tabla.oid = p_relacion
), secuencias_propias AS (
    SELECT COALESCE(pg_catalog.jsonb_agg(
               pg_catalog.jsonb_build_array(
                   atributo.attname, secuencia.relname,
                   propietario.rolname, secuencia.relpersistence,
                   COALESCE(espacio.spcname, ''),
                   secuencia.relacl IS NULL,
                   COALESCE((
                       SELECT pg_catalog.jsonb_agg(
                                  permiso::text ORDER BY permiso::text
                              )
                         FROM pg_catalog.unnest(secuencia.relacl) permiso
                   ), '[]'::jsonb),
                   datos.seqtypid::regtype::text,
                   datos.seqstart, datos.seqincrement,
                   datos.seqmax, datos.seqmin,
                   datos.seqcache, datos.seqcycle,
                   COALESCE(pg_catalog.obj_description(
                       secuencia.oid, 'pg_class'
                   ), ''),
                   COALESCE((
                       SELECT pg_catalog.jsonb_agg(
                                  pg_catalog.jsonb_build_array(
                                      etiqueta.provider, etiqueta.label
                                  ) ORDER BY etiqueta.provider
                              )
                         FROM pg_catalog.pg_seclabel etiqueta
                        WHERE etiqueta.classoid =
                              'pg_catalog.pg_class'::regclass
                          AND etiqueta.objoid = secuencia.oid
                   ), '[]'::jsonb)
               ) ORDER BY atributo.attname, secuencia.relname
           ), '[]'::jsonb) AS valor
      FROM pg_catalog.pg_depend dependencia
      JOIN pg_catalog.pg_class secuencia
        ON secuencia.oid = dependencia.objid
       AND secuencia.relkind = 'S'
      JOIN pg_catalog.pg_sequence datos
        ON datos.seqrelid = secuencia.oid
      JOIN pg_catalog.pg_attribute atributo
        ON atributo.attrelid = dependencia.refobjid
       AND atributo.attnum = dependencia.refobjsubid
      JOIN pg_catalog.pg_roles propietario
        ON propietario.oid = secuencia.relowner
 LEFT JOIN pg_catalog.pg_tablespace espacio
        ON espacio.oid = secuencia.reltablespace
     WHERE dependencia.classid = 'pg_catalog.pg_class'::regclass
       AND dependencia.refclassid = 'pg_catalog.pg_class'::regclass
       AND dependencia.refobjid = p_relacion
       AND dependencia.deptype IN ('a', 'i')
), tostado AS (
    SELECT pg_catalog.jsonb_build_array(
               relacion.reltoastrelid = 0,
               COALESCE(tostada.relpersistence::text, ''),
               COALESCE(propietario.rolname, ''),
               COALESCE(acceso.amname, ''),
               COALESCE(espacio.spcname, ''),
               tostada.relacl IS NULL,
               COALESCE((
                   SELECT pg_catalog.jsonb_agg(
                              permiso::text ORDER BY permiso::text
                          )
                     FROM pg_catalog.unnest(tostada.relacl) permiso
               ), '[]'::jsonb),
               tostada.reloptions IS NULL,
               COALESCE((
                   SELECT pg_catalog.jsonb_agg(opcion ORDER BY opcion)
                     FROM pg_catalog.unnest(tostada.reloptions) opcion
               ), '[]'::jsonb),
               COALESCE(pg_catalog.obj_description(
                   tostada.oid, 'pg_class'
               ), ''),
               COALESCE((
                   SELECT pg_catalog.jsonb_agg(
                              pg_catalog.jsonb_build_array(
                                  etiqueta.provider, etiqueta.label
                              ) ORDER BY etiqueta.provider
                          )
                     FROM pg_catalog.pg_seclabel etiqueta
                    WHERE etiqueta.classoid =
                          'pg_catalog.pg_class'::regclass
                      AND etiqueta.objoid = tostada.oid
               ), '[]'::jsonb),
               COALESCE((
                   SELECT pg_catalog.jsonb_agg(
                              pg_catalog.jsonb_build_array(
                                  acceso_indice.amname,
                                  propietario_indice.rolname,
                                  indice.indisunique,
                                  indice.indisprimary,
                                  indice.indisvalid,
                                  indice.indisready,
                                  indice.indislive,
                                  indice.indnkeyatts,
                                  indice.indnatts,
                                  indice.indkey::text,
                                  clase_indice.relacl IS NULL,
                                  COALESCE(pg_catalog.obj_description(
                                      clase_indice.oid, 'pg_class'
                                  ), '')
                              ) ORDER BY clase_indice.relname
                          )
                     FROM pg_catalog.pg_index indice
                     JOIN pg_catalog.pg_class clase_indice
                       ON clase_indice.oid = indice.indexrelid
                     JOIN pg_catalog.pg_am acceso_indice
                       ON acceso_indice.oid = clase_indice.relam
                     JOIN pg_catalog.pg_roles propietario_indice
                       ON propietario_indice.oid = clase_indice.relowner
                    WHERE indice.indrelid = tostada.oid
               ), '[]'::jsonb)
           ) AS valor
      FROM relacion
 LEFT JOIN pg_catalog.pg_class tostada
        ON tostada.oid = relacion.reltoastrelid
 LEFT JOIN pg_catalog.pg_roles propietario
        ON propietario.oid = tostada.relowner
 LEFT JOIN pg_catalog.pg_am acceso ON acceso.oid = tostada.relam
 LEFT JOIN pg_catalog.pg_tablespace espacio
        ON espacio.oid = tostada.reltablespace
)
SELECT pg_catalog.jsonb_build_object(
           'relacion', pg_catalog.jsonb_build_array(
               relacion.relkind, relacion.relpersistence,
               relacion.relispartition, relacion.relrowsecurity,
               relacion.relforcerowsecurity, relacion.relreplident,
               relacion.relispopulated, relacion.relhasindex,
               relacion.relchecks, relacion.relhasrules,
               relacion.relhastriggers, relacion.relhassubclass,
               relacion.propietario,
               COALESCE(relacion.metodo_acceso, ''),
               COALESCE(relacion.espacio_tablas, ''),
               relacion.acl_nula, relacion.acl,
               relacion.opciones_nulas, relacion.opciones
           ),
           'columnas', columnas.valor,
           'restricciones', restricciones.valor,
           'indices', indices.valor,
           'disparadores', disparadores.valor,
           'politicas', politicas.valor,
           'reglas', reglas.valor,
           'herencia', herencia.valor,
           'publicaciones', publicaciones.valor,
           'estadisticas', estadisticas.valor,
           'etiquetas', etiquetas.valor,
           'comentarios', comentarios.valor,
           'tipos_relacion', tipos_relacion.valor,
           'secuencias_propias', secuencias_propias.valor,
           'tostado', tostado.valor
       )::text
  FROM relacion
 CROSS JOIN columnas
 CROSS JOIN restricciones
 CROSS JOIN indices
 CROSS JOIN disparadores
 CROSS JOIN politicas
 CROSS JOIN reglas
 CROSS JOIN herencia
 CROSS JOIN publicaciones
 CROSS JOIN estadisticas
 CROSS JOIN etiquetas
 CROSS JOIN comentarios
 CROSS JOIN tipos_relacion
 CROSS JOIN secuencias_propias
 CROSS JOIN tostado;
$funcion$;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 21
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 5
    ) OR (
        SELECT pg_catalog.count(*)
          FROM vec_contratacion_temporal
               .control_vocabulario_estados_publicacion_rrhh_v1
         WHERE control
           AND version_esquema = 1
           AND restriccion_nombre =
               'publicacion_version_rrhh_estado_clave_valido'
           AND restriccion_definicion =
               $def$CHECK ((estado_clave = ANY (ARRAY['pendiente'::text, 'en_curso'::text, 'espera_externa'::text, 'completado'::text, 'incidencia'::text, 'cancelado'::text])))$def$
           AND restriccion_validada
    ) <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para revertir estados RRHH';
    END IF;

    IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
           vec_contratacion_temporal
               .material_estructura_relacion_ct000041a(
                   'vec_contratacion_temporal.'
                   'publicacion_version_rrhh'::regclass
               ),
           'UTF8'
       )), 'hex') <>
       '27023128e8af79b741a3a4f9ddcc43bcc0444cfda14b7076fa7bdb6c44b329d1'
       OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
           vec_contratacion_temporal
               .material_estructura_relacion_ct000041a(
                   'vec_contratacion_temporal.'
                   'control_vocabulario_estados_publicacion_rrhh_v1'::regclass
               ),
           'UTF8'
       )), 'hex') <>
       '930ee7d905e575309ca7621aa4feb67d7ef8329512c7a8fa19ee16c789fa85ac'
       OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
           vec_contratacion_temporal
               .material_estructura_relacion_ct000041a(
                   'vec_contratacion_temporal.'
                   'expediente_version_integral'::regclass
               ),
           'UTF8'
       )), 'hex') <>
       'a6a567bbb0fc0a4ed392447088a6269ed39b1eb81bbb3d86922ffb8e458b389d'
    THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estructura derivada impide revertir estados RRHH';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.publicacion_version_rrhh
         WHERE estado_clave IN (
             'pendiente', 'espera_externa', 'incidencia'
         )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '2BP01',
            MESSAGE = 'estados nuevos impiden revertir estados RRHH';
    END IF;
END
$prevalidacion$;

ALTER TABLE vec_contratacion_temporal.publicacion_version_rrhh
DROP CONSTRAINT publicacion_version_rrhh_estado_clave_valido;
ALTER TABLE vec_contratacion_temporal.publicacion_version_rrhh
ADD CONSTRAINT publicacion_version_rrhh_estado_clave_check
CHECK (estado_clave IN (
    'en_curso',
    'completado',
    'cancelado'
)) NOT VALID;
ALTER TABLE vec_contratacion_temporal.publicacion_version_rrhh
VALIDATE CONSTRAINT publicacion_version_rrhh_estado_clave_check;

DROP TRIGGER control_vocabulario_estados_rrhh_no_truncar
ON vec_contratacion_temporal
   .control_vocabulario_estados_publicacion_rrhh_v1;
DROP TRIGGER control_vocabulario_estados_rrhh_inmutable
ON vec_contratacion_temporal
   .control_vocabulario_estados_publicacion_rrhh_v1;
DROP POLICY propietario_total
ON vec_contratacion_temporal
   .control_vocabulario_estados_publicacion_rrhh_v1;
DROP TABLE vec_contratacion_temporal
    .control_vocabulario_estados_publicacion_rrhh_v1
RESTRICT;

DO $manifiesto_retirado$
BEGIN
    IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
           vec_contratacion_temporal
               .material_estructura_relacion_ct000041a(
                   'vec_contratacion_temporal.'
                   'publicacion_version_rrhh'::regclass
               ),
           'UTF8'
       )), 'hex') <>
       'b0d098190a2d5cbdd01885342232e231315e1be100344ab2b9150a1bd98e1276'
    THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'restauración de estados RRHH incompleta';
    END IF;
END
$manifiesto_retirado$;

DROP FUNCTION
    vec_contratacion_temporal
        .material_estructura_relacion_ct000041a(regclass);

UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 4,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 5;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 20,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 21;
COMMIT;
