-- Reversión segura de O4-05/C2-D1. La historia probatoria nunca se destruye:
-- solo puede retirarse una instalación aún vacía y sin fachadas posteriores.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:o4_05:consultas_rrhh:migraciones',
        0
    )
);

SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control
   AND version_esquema = 18
 FOR UPDATE;
SELECT control
  FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
 WHERE control
   AND version_esquema = 2
 FOR UPDATE;

LOCK TABLE
    vec_contratacion_temporal.revocacion_familia_cursor_rrhh,
    vec_contratacion_temporal.consumo_cursor_cuadro_rrhh,
    vec_contratacion_temporal.cursor_cuadro_rrhh,
    vec_contratacion_temporal.familia_cursor_cuadro_rrhh,
    vec_contratacion_temporal.alcance_acceso_rrhh,
    vec_contratacion_temporal.control_cursores_cuadro_rrhh
IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
DECLARE
    v_huella_acl text;
    v_huella_columnas text;
    v_huella_disparadores text;
    v_huella_disparadores_ri text;
    v_huella_indices text;
    v_huella_politicas text;
    v_huella_relaciones text;
    v_huella_restricciones text;
    v_funcion oid := pg_catalog.to_regprocedure(
        'public.gen_random_bytes(integer)'
    );
    v_tablas oid[] := ARRAY[
        'vec_contratacion_temporal.control_cursores_cuadro_rrhh'::regclass,
        'vec_contratacion_temporal.alcance_acceso_rrhh'::regclass,
        'vec_contratacion_temporal.familia_cursor_cuadro_rrhh'::regclass,
        'vec_contratacion_temporal.cursor_cuadro_rrhh'::regclass,
        'vec_contratacion_temporal.consumo_cursor_cuadro_rrhh'::regclass,
        'vec_contratacion_temporal.revocacion_familia_cursor_rrhh'::regclass
    ]::oid[];
BEGIN
    SELECT pg_catalog.encode(
               pg_catalog.sha256(pg_catalog.convert_to(
                   COALESCE(
                       pg_catalog.jsonb_agg(
                           pg_catalog.jsonb_build_array(
                               tabla.oid::pg_catalog.regclass::text,
                               tabla.relkind::text,
                               tabla.relpersistence::text,
                               tabla.relowner::pg_catalog.regrole::text,
                               tabla.relrowsecurity,
                               tabla.relforcerowsecurity,
                               tabla.relhasrules,
                               tabla.relreplident::text,
                               metodo.amname,
                               espacio.spcname,
                               ARRAY(
                                   SELECT opcion
                                     FROM pg_catalog.unnest(
                                         tabla.reloptions
                                     ) opcion
                                    ORDER BY opcion COLLATE "C"
                               )
                           )
                           ORDER BY (
                               tabla.oid::pg_catalog.regclass::text
                           ) COLLATE "C"
                       )::text,
                       '[]'
                   ),
                   'UTF8'
               )),
               'hex'
           )
      INTO v_huella_relaciones
      FROM pg_catalog.pg_class tabla
      LEFT JOIN pg_catalog.pg_am metodo
        ON metodo.oid = tabla.relam
      LEFT JOIN pg_catalog.pg_tablespace espacio
        ON espacio.oid = tabla.reltablespace
     WHERE tabla.oid = ANY(v_tablas);

    SELECT pg_catalog.encode(
               pg_catalog.sha256(pg_catalog.convert_to(
                   COALESCE(
                       pg_catalog.jsonb_agg(
                           pg_catalog.jsonb_build_array(
                               atributo.attrelid::pg_catalog.regclass::text,
                               atributo.attnum,
                               atributo.attname,
                               pg_catalog.format_type(
                                   atributo.atttypid, atributo.atttypmod
                               ),
                               atributo.attnotnull,
                               atributo.attidentity::text,
                               atributo.attgenerated::text,
                               CASE
                                   WHEN atributo.attcollation = 0 THEN NULL
                                   ELSE atributo.attcollation
                                        ::pg_catalog.regcollation::text
                               END,
                               atributo.atthasdef,
                               pg_catalog.pg_get_expr(
                                   defecto.adbin, defecto.adrelid, false
                               ),
                               atributo.attisdropped,
                               atributo.attstorage::text,
                               atributo.attcompression::text
                           )
                           ORDER BY (
                               atributo.attrelid::pg_catalog.regclass::text
                           ) COLLATE "C",
                           atributo.attnum
                       )::text,
                       '[]'
                   ),
                   'UTF8'
               )),
               'hex'
           )
      INTO v_huella_columnas
      FROM pg_catalog.pg_attribute atributo
      LEFT JOIN pg_catalog.pg_attrdef defecto
        ON defecto.adrelid = atributo.attrelid
       AND defecto.adnum = atributo.attnum
     WHERE atributo.attrelid = ANY(v_tablas)
       AND atributo.attnum > 0;

    SELECT pg_catalog.encode(
               pg_catalog.sha256(pg_catalog.convert_to(
                   COALESCE(
                       pg_catalog.jsonb_agg(
                           pg_catalog.jsonb_build_array(
                               indice.indrelid::pg_catalog.regclass::text,
                               indice.indexrelid::pg_catalog.regclass::text,
                               clase.relowner::pg_catalog.regrole::text,
                               metodo.amname,
                               espacio.spcname,
                               indice.indisunique,
                               indice.indisprimary,
                               indice.indisexclusion,
                               indice.indimmediate,
                               indice.indisclustered,
                               indice.indisvalid,
                               indice.indisready,
                               indice.indislive,
                               indice.indisreplident,
                               indice.indnullsnotdistinct,
                               pg_catalog.pg_get_indexdef(
                                   indice.indexrelid, 0, false
                               ),
                               ARRAY(
                                   SELECT opcion
                                     FROM pg_catalog.unnest(
                                         clase.reloptions
                                     ) opcion
                                    ORDER BY opcion COLLATE "C"
                               )
                           )
                           ORDER BY (
                               indice.indrelid::pg_catalog.regclass::text
                           ) COLLATE "C",
                           (
                               indice.indexrelid::pg_catalog.regclass::text
                           ) COLLATE "C"
                       )::text,
                       '[]'
                   ),
                   'UTF8'
               )),
               'hex'
           )
      INTO v_huella_indices
      FROM pg_catalog.pg_index indice
      JOIN pg_catalog.pg_class clase
        ON clase.oid = indice.indexrelid
      JOIN pg_catalog.pg_am metodo
        ON metodo.oid = clase.relam
      LEFT JOIN pg_catalog.pg_tablespace espacio
        ON espacio.oid = clase.reltablespace
     WHERE indice.indrelid = ANY(v_tablas)
        OR indice.indexrelid IN (
            SELECT restriccion.conindid
              FROM pg_catalog.pg_constraint restriccion
             WHERE restriccion.conrelid =
                   'vec_contratacion_temporal.registro_acceso_rrhh'::regclass
               AND restriccion.conname = ANY(ARRAY[
                   'registro_acceso_rrhh_cursor_alcance_unico',
                   'registro_acceso_rrhh_cursor_identidad_unica',
                   'registro_acceso_rrhh_cursor_consumo_unico',
                   'registro_acceso_rrhh_cursor_tipo_unico'
               ]::name[])
        );

    SELECT pg_catalog.encode(
               pg_catalog.sha256(pg_catalog.convert_to(
                   COALESCE(
                       pg_catalog.jsonb_agg(
                           pg_catalog.jsonb_build_array(
                               politica.polrelid::pg_catalog.regclass::text,
                               politica.polname,
                               politica.polcmd::text,
                               politica.polpermissive,
                               (
                                   SELECT pg_catalog.jsonb_agg(
                                              CASE
                                                  WHEN rol_oid = 0
                                                      THEN 'PUBLIC'
                                                  ELSE rol_oid
                                                       ::pg_catalog.regrole
                                                       ::text
                                              END
                                              ORDER BY (
                                                  CASE
                                                      WHEN rol_oid = 0
                                                          THEN 'PUBLIC'
                                                      ELSE rol_oid
                                                           ::pg_catalog.regrole
                                                           ::text
                                                  END
                                              ) COLLATE "C"
                                          )
                                     FROM pg_catalog.unnest(
                                         politica.polroles
                                     ) rol_oid
                               ),
                               pg_catalog.pg_get_expr(
                                   politica.polqual,
                                   politica.polrelid,
                                   false
                               ),
                               pg_catalog.pg_get_expr(
                                   politica.polwithcheck,
                                   politica.polrelid,
                                   false
                               )
                           )
                           ORDER BY (
                               politica.polrelid::pg_catalog.regclass::text
                           ) COLLATE "C",
                           politica.polname COLLATE "C"
                       )::text,
                       '[]'
                   ),
                   'UTF8'
               )),
               'hex'
           )
      INTO v_huella_politicas
      FROM pg_catalog.pg_policy politica
     WHERE politica.polrelid = ANY(v_tablas);

    WITH acl_normalizada AS (
        SELECT
            'tabla'::text AS clase,
            tabla.oid::pg_catalog.regclass::text AS objeto,
            NULL::text AS columna,
            CASE
                WHEN privilegio.grantee = 0 THEN 'PUBLIC'
                ELSE privilegio.grantee::pg_catalog.regrole::text
            END AS receptor,
            CASE
                WHEN privilegio.grantor = 0 THEN 'PUBLIC'
                ELSE privilegio.grantor::pg_catalog.regrole::text
            END AS otorgante,
            privilegio.privilege_type,
            privilegio.is_grantable
          FROM pg_catalog.pg_class tabla
         CROSS JOIN LATERAL pg_catalog.aclexplode(
             COALESCE(
                 tabla.relacl,
                 pg_catalog.acldefault('r', tabla.relowner)
             )
         ) privilegio
         WHERE tabla.oid = ANY(v_tablas)
        UNION ALL
        SELECT
            'columna',
            atributo.attrelid::pg_catalog.regclass::text,
            atributo.attname,
            CASE
                WHEN privilegio.grantee = 0 THEN 'PUBLIC'
                ELSE privilegio.grantee::pg_catalog.regrole::text
            END,
            CASE
                WHEN privilegio.grantor = 0 THEN 'PUBLIC'
                ELSE privilegio.grantor::pg_catalog.regrole::text
            END,
            privilegio.privilege_type,
            privilegio.is_grantable
          FROM pg_catalog.pg_attribute atributo
         CROSS JOIN LATERAL pg_catalog.aclexplode(
             atributo.attacl
         ) privilegio
         WHERE atributo.attrelid = ANY(v_tablas)
           AND atributo.attnum > 0
    )
    SELECT pg_catalog.encode(
               pg_catalog.sha256(pg_catalog.convert_to(
                   COALESCE(
                       pg_catalog.jsonb_agg(
                           pg_catalog.jsonb_build_array(
                               clase, objeto, columna, receptor, otorgante,
                               privilege_type, is_grantable
                           )
                           ORDER BY clase COLLATE "C",
                                    objeto COLLATE "C",
                                    columna COLLATE "C",
                                    receptor COLLATE "C",
                                    otorgante COLLATE "C",
                                    privilege_type COLLATE "C",
                                    is_grantable
                       )::text,
                       '[]'
                   ),
                   'UTF8'
               )),
               'hex'
           )
      INTO v_huella_acl
      FROM acl_normalizada;

    SELECT pg_catalog.encode(
               pg_catalog.sha256(pg_catalog.convert_to(
                   COALESCE(
                       pg_catalog.jsonb_agg(
                           pg_catalog.jsonb_build_array(
                               restriccion.conrelid::pg_catalog.regclass::text,
                               restriccion.conname,
                               restriccion.contype::text,
                               restriccion.condeferrable,
                               restriccion.condeferred,
                               restriccion.conenforced,
                               restriccion.convalidated,
                               pg_catalog.pg_get_constraintdef(
                                   restriccion.oid, false
                               )
                           )
                           ORDER BY (
                               restriccion.conrelid::pg_catalog.regclass::text
                           ) COLLATE "C",
                           restriccion.conname COLLATE "C"
                       )::text,
                       '[]'
                   ),
                   'UTF8'
               )),
               'hex'
           )
      INTO v_huella_restricciones
      FROM pg_catalog.pg_constraint restriccion
     WHERE restriccion.conrelid = ANY(v_tablas)
        OR (
            restriccion.conrelid =
                'vec_contratacion_temporal.registro_acceso_rrhh'::regclass
            AND restriccion.conname = ANY(ARRAY[
                'registro_acceso_rrhh_cursor_alcance_unico',
                'registro_acceso_rrhh_cursor_identidad_unica',
                'registro_acceso_rrhh_cursor_consumo_unico',
                'registro_acceso_rrhh_cursor_tipo_unico'
            ]::name[])
        );

    SELECT pg_catalog.encode(
               pg_catalog.sha256(pg_catalog.convert_to(
                   COALESCE(
                       pg_catalog.jsonb_agg(
                           pg_catalog.jsonb_build_array(
                               disparador.tgrelid::pg_catalog.regclass::text,
                               disparador.tgname,
                               disparador.tgtype::integer,
                               disparador.tgenabled::text,
                               disparador.tgdeferrable,
                               disparador.tginitdeferred,
                               disparador.tgfoid::pg_catalog.regprocedure::text,
                               pg_catalog.pg_get_triggerdef(
                                   disparador.oid, false
                               )
                           )
                           ORDER BY (
                               disparador.tgrelid::pg_catalog.regclass::text
                           ) COLLATE "C",
                           disparador.tgname COLLATE "C"
                       )::text,
                       '[]'
                   ),
                   'UTF8'
               )),
               'hex'
           )
      INTO v_huella_disparadores
      FROM pg_catalog.pg_trigger disparador
     WHERE disparador.tgrelid = ANY(v_tablas)
       AND NOT disparador.tgisinternal;

    SELECT pg_catalog.encode(
               pg_catalog.sha256(pg_catalog.convert_to(
                   COALESCE(
                       pg_catalog.jsonb_agg(
                           pg_catalog.jsonb_build_array(
                               restriccion.conrelid
                                   ::pg_catalog.regclass::text,
                               restriccion.conname,
                               disparador.tgrelid
                                   ::pg_catalog.regclass::text,
                               disparador.tgtype::integer,
                               disparador.tgenabled::text,
                               disparador.tgdeferrable,
                               disparador.tginitdeferred,
                               disparador.tgfoid
                                   ::pg_catalog.regprocedure::text,
                               CASE
                                   WHEN disparador.tgconstrrelid = 0
                                       THEN NULL
                                   ELSE disparador.tgconstrrelid
                                        ::pg_catalog.regclass::text
                               END,
                               CASE
                                   WHEN disparador.tgconstrindid = 0
                                       THEN NULL
                                   ELSE disparador.tgconstrindid
                                        ::pg_catalog.regclass::text
                               END,
                               pg_catalog.encode(
                                   disparador.tgargs, 'hex'
                               ),
                               disparador.tgattr::text,
                               pg_catalog.pg_get_expr(
                                   disparador.tgqual,
                                   disparador.tgrelid,
                                   false
                               )
                           )
                           ORDER BY (
                               restriccion.conrelid
                                   ::pg_catalog.regclass::text
                           ) COLLATE "C",
                           restriccion.conname COLLATE "C",
                           (
                               disparador.tgrelid
                                   ::pg_catalog.regclass::text
                           ) COLLATE "C",
                           (
                               disparador.tgfoid
                                   ::pg_catalog.regprocedure::text
                           ) COLLATE "C",
                           disparador.tgtype,
                           (
                               CASE
                                   WHEN disparador.tgconstrrelid = 0
                                       THEN NULL
                                   ELSE disparador.tgconstrrelid
                                        ::pg_catalog.regclass::text
                               END
                           ) COLLATE "C",
                           (
                               CASE
                                   WHEN disparador.tgconstrindid = 0
                                       THEN NULL
                                   ELSE disparador.tgconstrindid
                                        ::pg_catalog.regclass::text
                               END
                           ) COLLATE "C"
                       )::text,
                       '[]'
                   ),
                   'UTF8'
               )),
               'hex'
           )
      INTO v_huella_disparadores_ri
      FROM pg_catalog.pg_trigger disparador
      JOIN pg_catalog.pg_constraint restriccion
        ON restriccion.oid = disparador.tgconstraint
     WHERE restriccion.conrelid = ANY(v_tablas)
       AND disparador.tgisinternal;

    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 18
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 2
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .control_cursores_cuadro_rrhh
         WHERE control AND version_esquema = 1
    ) OR (
        SELECT pg_catalog.count(*)
          FROM vec_contratacion_temporal
               .control_cursores_cuadro_rrhh
    ) <> 1 OR v_huella_relaciones IS DISTINCT FROM
        '927f0e752b9ff78503310bca9c86456ae720ca38db3236c2e8411e811532f070'
      OR v_huella_columnas IS DISTINCT FROM
        '18acef5f0cd97c106cbd130a058bccfa945d70aef15e1c7190ce7ffa65f7c748'
      OR v_huella_indices IS DISTINCT FROM
        '5afadebd0638ffc5709884a172c4433e5209f0d786d44e901cf72cfc30b9ee3e'
      OR v_huella_politicas IS DISTINCT FROM
        '9725429ed5c1ea51effdf9cd2e1e64dd7c19ddc32f37ba614ed0c190b0b819e4'
      OR v_huella_acl IS DISTINCT FROM
        '7213de9f6c70f4dc627d76c6ad535b0af533a59717238518fe7230bbf9d1ce75'
      OR v_huella_restricciones IS DISTINCT FROM
        '8f4907cbdc5f3f4a27a8aecb65bdce464c15bdef8ea3227100ade121967f3ae3'
      OR v_huella_disparadores IS DISTINCT FROM
        'c9e96c86896d211a8456fc5c8175bf40b6c0c71e016a5207336473f351328820'
      OR v_huella_disparadores_ri IS DISTINCT FROM
        '0c7248eac4d248cef11a1fea8450a6b041a39a389f927efb2a1255aaf702fcb0'
      OR EXISTS (
          SELECT 1
            FROM pg_catalog.pg_trigger disparador
            JOIN pg_catalog.pg_constraint restriccion
              ON restriccion.oid = disparador.tgconstraint
           WHERE restriccion.conrelid = ANY(v_tablas)
             AND disparador.tgisinternal
             AND disparador.tgenabled <> 'O'
      )
      OR v_funcion IS NULL
      OR NOT EXISTS (
          SELECT 1
            FROM pg_catalog.pg_proc funcion
            JOIN pg_catalog.pg_namespace esquema
              ON esquema.oid = funcion.pronamespace
            JOIN pg_catalog.pg_depend dependencia
              ON dependencia.objid = funcion.oid
            JOIN pg_catalog.pg_extension extension
              ON extension.oid = dependencia.refobjid
           WHERE funcion.oid = v_funcion
             AND esquema.nspname = 'public'
             AND extension.extname = 'pgcrypto'
             AND extension.extnamespace = esquema.oid
             AND funcion.proowner = extension.extowner
             AND funcion.prosupport = 0
             AND funcion.proconfig IS NULL
             AND pg_catalog.encode(pg_catalog.sha256(
                 pg_catalog.convert_to(
                     pg_catalog.pg_get_functiondef(funcion.oid), 'UTF8'
                 )
             ), 'hex') =
                 '3e5d4a298efb95a8c94a2e47a06244bb747c33f2400461b01531b7b12bc010b6'
             AND dependencia.classid = 'pg_catalog.pg_proc'::regclass
             AND dependencia.refclassid =
                 'pg_catalog.pg_extension'::regclass
             AND dependencia.deptype = 'e'
      )
      OR NOT pg_catalog.has_schema_privilege(
        'vec_contratacion_temporal_propietario', 'public', 'USAGE'
    ) OR NOT pg_catalog.has_function_privilege(
        'vec_contratacion_temporal_propietario',
        'public.gen_random_bytes(integer)', 'EXECUTE'
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'down de cursores del cuadro RRHH fuera de orden';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.alcance_acceso_rrhh
    ) OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.familia_cursor_cuadro_rrhh
    ) OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.cursor_cuadro_rrhh
    ) OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.consumo_cursor_cuadro_rrhh
    ) OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.revocacion_familia_cursor_rrhh
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'historia impide retirar cursores del cuadro RRHH';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc funcion
          JOIN pg_catalog.pg_namespace esquema
            ON esquema.oid = funcion.pronamespace
         WHERE esquema.nspname = 'vec_contratacion_temporal'
           AND funcion.proname ~
               '(cursor.*rrhh|rrhh.*cursor|^consultar_(cuadro|detalle)_rrhh)'
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_constraint dependencia
         WHERE dependencia.confrelid = ANY(v_tablas)
           AND dependencia.conrelid <> ALL(v_tablas)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_rewrite regla
         WHERE regla.ev_class = ANY(v_tablas)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_depend dependencia
          JOIN pg_catalog.pg_rewrite regla
            ON dependencia.classid = 'pg_catalog.pg_rewrite'::regclass
           AND regla.oid = dependencia.objid
         WHERE dependencia.refobjid = ANY(v_tablas)
           AND regla.ev_class <> ALL(v_tablas)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_trigger disparador
          JOIN pg_catalog.pg_class tabla
            ON tabla.oid = disparador.tgrelid
         WHERE disparador.tgrelid = ANY(v_tablas)
           AND NOT disparador.tgisinternal
           AND disparador.tgname NOT IN (
               tabla.relname || '_inmutable',
               tabla.relname || '_no_truncar'
           )
    ) OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_trigger disparador
         WHERE disparador.tgrelid = ANY(v_tablas)
           AND NOT disparador.tgisinternal
    ) <> 10 THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'dependencias futuras impiden retirar cursores RRHH';
    END IF;
END
$prevalidacion$;

DROP TABLE
    vec_contratacion_temporal.revocacion_familia_cursor_rrhh;
ALTER TABLE vec_contratacion_temporal.cursor_cuadro_rrhh
    DROP CONSTRAINT cursor_cuadro_rrhh_consumo_padre_fk;
DROP TABLE vec_contratacion_temporal.consumo_cursor_cuadro_rrhh;
DROP TABLE vec_contratacion_temporal.cursor_cuadro_rrhh;
ALTER TABLE vec_contratacion_temporal.alcance_acceso_rrhh
    DROP CONSTRAINT alcance_acceso_rrhh_familia_fk;
DROP TABLE vec_contratacion_temporal.familia_cursor_cuadro_rrhh;
DROP TABLE vec_contratacion_temporal.alcance_acceso_rrhh;
DROP TABLE vec_contratacion_temporal.control_cursores_cuadro_rrhh;
ALTER TABLE vec_contratacion_temporal.registro_acceso_rrhh
    DROP CONSTRAINT registro_acceso_rrhh_cursor_tipo_unico,
    DROP CONSTRAINT registro_acceso_rrhh_cursor_consumo_unico,
    DROP CONSTRAINT registro_acceso_rrhh_cursor_identidad_unica,
    DROP CONSTRAINT registro_acceso_rrhh_cursor_alcance_unico;

UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 1,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 2;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 17,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 18;

COMMIT;
