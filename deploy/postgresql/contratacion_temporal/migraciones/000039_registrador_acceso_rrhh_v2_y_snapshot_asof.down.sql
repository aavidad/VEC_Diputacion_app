-- Reversión segura de C2-D2-C. Conserva toda historia anterior al baseline.
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
 WHERE control AND version_esquema = 19
 FOR UPDATE;
SELECT control
  FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
 WHERE control AND version_esquema = 3
 FOR UPDATE;

LOCK TABLE
    vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2,
    vec_contratacion_temporal.control_registrador_acceso_rrhh_v2,
    vec_contratacion_temporal.registro_acceso_rrhh,
    vec_contratacion_temporal.control_cadena_accesos_rrhh,
    vec_contratacion_temporal.publicacion_version_rrhh
IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
DECLARE
    huella_acl text;
    huella_columnas text;
    huella_disparadores text;
    huella_disparadores_ri text;
    huella_funcion text;
    huella_indices text;
    huella_politicas text;
    huella_reglas text;
    huella_relaciones text;
    huella_restricciones text;
    funcion oid := pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.'
        || 'registrar_acceso_rrhh_interno_v2(jsonb)'
    );
    indice oid := pg_catalog.to_regclass(
        'vec_contratacion_temporal.'
        || 'publicacion_rrhh_organizacion_expediente_corte_desc_idx'
    );
    tablas oid[] := ARRAY[
        'vec_contratacion_temporal.'
        'control_registrador_acceso_rrhh_v2'::regclass,
        'vec_contratacion_temporal.'
        'vinculo_identidad_acceso_rrhh_v2'::regclass
    ]::oid[];
BEGIN
    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               COALESCE(pg_catalog.jsonb_agg(
                   pg_catalog.jsonb_build_array(
                       tabla.oid::pg_catalog.regclass::text,
                       tabla.relkind::text, tabla.relpersistence::text,
                       tabla.relowner::pg_catalog.regrole::text,
                       tabla.relrowsecurity, tabla.relforcerowsecurity,
                       tabla.relhasrules, tabla.relhastriggers,
                       tabla.relhassubclass, tabla.relispartition,
                       pg_catalog.pg_get_expr(
                           tabla.relpartbound, tabla.oid, false
                       ),
                       tabla.relreplident::text,
                       metodo.amname, espacio.spcname,
                       pg_catalog.obj_description(
                           tabla.oid, 'pg_class'
                       ),
                       ARRAY(
                           SELECT opcion
                             FROM pg_catalog.unnest(tabla.reloptions) opcion
                            ORDER BY opcion COLLATE "C"
                       )
                   )
                   ORDER BY (
                       tabla.oid::pg_catalog.regclass::text
                   ) COLLATE "C"
               )::text, '[]'), 'UTF8')), 'hex')
      INTO huella_relaciones
      FROM pg_catalog.pg_class tabla
      LEFT JOIN pg_catalog.pg_am metodo ON metodo.oid = tabla.relam
      LEFT JOIN pg_catalog.pg_tablespace espacio
        ON espacio.oid = tabla.reltablespace
     WHERE tabla.oid = ANY(tablas);

    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               COALESCE(pg_catalog.jsonb_agg(
                   pg_catalog.jsonb_build_array(
                       atributo.attrelid::pg_catalog.regclass::text,
                       atributo.attnum, atributo.attname,
                       pg_catalog.format_type(
                           atributo.atttypid, atributo.atttypmod
                       ),
                       atributo.attnotnull, atributo.attidentity::text,
                       atributo.attgenerated::text,
                       CASE WHEN atributo.attcollation = 0 THEN NULL
                            ELSE atributo.attcollation
                                 ::pg_catalog.regcollation::text END,
                       atributo.atthasdef,
                       pg_catalog.pg_get_expr(
                           defecto.adbin, defecto.adrelid, false
                       ),
                       atributo.attisdropped, atributo.attstorage::text,
                       atributo.attcompression::text,
                       pg_catalog.col_description(
                           atributo.attrelid, atributo.attnum
                       )
                   )
                   ORDER BY (
                       atributo.attrelid::pg_catalog.regclass::text
                   ) COLLATE "C", atributo.attnum
               )::text, '[]'), 'UTF8')), 'hex')
      INTO huella_columnas
      FROM pg_catalog.pg_attribute atributo
      LEFT JOIN pg_catalog.pg_attrdef defecto
        ON defecto.adrelid = atributo.attrelid
       AND defecto.adnum = atributo.attnum
     WHERE atributo.attrelid = ANY(tablas)
       AND atributo.attnum > 0;

    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               COALESCE(pg_catalog.jsonb_agg(
                   pg_catalog.jsonb_build_array(
                       i.indrelid::pg_catalog.regclass::text,
                       i.indexrelid::pg_catalog.regclass::text,
                       clase.relowner::pg_catalog.regrole::text,
                       metodo.amname, espacio.spcname,
                       i.indisunique, i.indisprimary, i.indisexclusion,
                       i.indimmediate, i.indisclustered, i.indisvalid,
                       i.indisready, i.indislive, i.indisreplident,
                       i.indnullsnotdistinct, i.indcheckxmin,
                       pg_catalog.pg_get_indexdef(i.indexrelid, 0, false),
                       pg_catalog.obj_description(
                           i.indexrelid, 'pg_class'
                       ),
                       ARRAY(
                           SELECT opcion
                             FROM pg_catalog.unnest(clase.reloptions) opcion
                            ORDER BY opcion COLLATE "C"
                       )
                   )
                   ORDER BY (
                       i.indrelid::pg_catalog.regclass::text
                   ) COLLATE "C",
                   (i.indexrelid::pg_catalog.regclass::text) COLLATE "C"
               )::text, '[]'), 'UTF8')), 'hex')
      INTO huella_indices
      FROM pg_catalog.pg_index i
      JOIN pg_catalog.pg_class clase ON clase.oid = i.indexrelid
      JOIN pg_catalog.pg_am metodo ON metodo.oid = clase.relam
      LEFT JOIN pg_catalog.pg_tablespace espacio
        ON espacio.oid = clase.reltablespace
     WHERE i.indrelid = ANY(tablas)
        OR i.indexrelid = indice;

    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               COALESCE(pg_catalog.jsonb_agg(
                   pg_catalog.jsonb_build_array(
                       politica.polrelid::pg_catalog.regclass::text,
                       politica.polname, politica.polcmd::text,
                       politica.polpermissive,
                       (
                           SELECT pg_catalog.jsonb_agg(
                                      CASE WHEN rol = 0 THEN 'PUBLIC'
                                           ELSE rol::pg_catalog.regrole::text
                                      END
                                      ORDER BY (
                                          CASE WHEN rol = 0 THEN 'PUBLIC'
                                               ELSE rol::pg_catalog.regrole
                                                    ::text END
                                      ) COLLATE "C"
                                  )
                             FROM pg_catalog.unnest(politica.polroles) rol
                       ),
                       pg_catalog.pg_get_expr(
                           politica.polqual, politica.polrelid, false
                       ),
                       pg_catalog.pg_get_expr(
                           politica.polwithcheck, politica.polrelid, false
                       )
                   )
                   ORDER BY (
                       politica.polrelid::pg_catalog.regclass::text
                   ) COLLATE "C", politica.polname COLLATE "C"
               )::text, '[]'), 'UTF8')), 'hex')
      INTO huella_politicas
      FROM pg_catalog.pg_policy politica
     WHERE politica.polrelid = ANY(tablas);

    WITH acl_normalizada AS (
        SELECT 'tabla'::text clase,
               tabla.oid::pg_catalog.regclass::text objeto,
               NULL::text columna,
               CASE WHEN a.grantee = 0 THEN 'PUBLIC'
                    ELSE a.grantee::pg_catalog.regrole::text END receptor,
               CASE WHEN a.grantor = 0 THEN 'PUBLIC'
                    ELSE a.grantor::pg_catalog.regrole::text END otorgante,
               a.privilege_type, a.is_grantable
          FROM pg_catalog.pg_class tabla
         CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
             tabla.relacl, pg_catalog.acldefault('r', tabla.relowner)
         )) a
         WHERE tabla.oid = ANY(tablas)
        UNION ALL
        SELECT 'columna',
               atributo.attrelid::pg_catalog.regclass::text,
               atributo.attname,
               CASE WHEN a.grantee = 0 THEN 'PUBLIC'
                    ELSE a.grantee::pg_catalog.regrole::text END,
               CASE WHEN a.grantor = 0 THEN 'PUBLIC'
                    ELSE a.grantor::pg_catalog.regrole::text END,
               a.privilege_type, a.is_grantable
          FROM pg_catalog.pg_attribute atributo
         CROSS JOIN LATERAL pg_catalog.aclexplode(atributo.attacl) a
         WHERE atributo.attrelid = ANY(tablas)
           AND atributo.attnum > 0
        UNION ALL
        SELECT 'funcion',
               p.oid::pg_catalog.regprocedure::text,
               NULL::text,
               CASE WHEN a.grantee = 0 THEN 'PUBLIC'
                    ELSE a.grantee::pg_catalog.regrole::text END,
               CASE WHEN a.grantor = 0 THEN 'PUBLIC'
                    ELSE a.grantor::pg_catalog.regrole::text END,
               a.privilege_type, a.is_grantable
          FROM pg_catalog.pg_proc p
         CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
             p.proacl, pg_catalog.acldefault('f', p.proowner)
         )) a
         WHERE p.oid = funcion
    )
    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               COALESCE(pg_catalog.jsonb_agg(
                   pg_catalog.jsonb_build_array(
                       clase, objeto, columna, receptor, otorgante,
                       privilege_type, is_grantable
                   )
                   ORDER BY clase COLLATE "C", objeto COLLATE "C",
                            columna COLLATE "C", receptor COLLATE "C",
                            otorgante COLLATE "C",
                            privilege_type COLLATE "C", is_grantable
               )::text, '[]'), 'UTF8')), 'hex')
      INTO huella_acl
      FROM acl_normalizada;

    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               COALESCE(pg_catalog.jsonb_agg(
                   pg_catalog.jsonb_build_array(
                       c.conrelid::pg_catalog.regclass::text,
                       c.conname, c.contype::text, c.condeferrable,
                       c.condeferred, c.conenforced, c.convalidated,
                       c.conislocal, c.coninhcount, c.connoinherit,
                       c.conparentid = 0,
                       pg_catalog.pg_get_constraintdef(c.oid, false)
                   )
                   ORDER BY (
                       c.conrelid::pg_catalog.regclass::text
                   ) COLLATE "C", c.conname COLLATE "C"
               )::text, '[]'), 'UTF8')), 'hex')
      INTO huella_restricciones
      FROM pg_catalog.pg_constraint c
     WHERE c.conrelid = ANY(tablas);

    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               COALESCE(pg_catalog.jsonb_agg(
                   pg_catalog.jsonb_build_array(
                       t.tgrelid::pg_catalog.regclass::text,
                       t.tgname, t.tgtype::integer, t.tgenabled::text,
                       t.tgdeferrable, t.tginitdeferred,
                       t.tgfoid::pg_catalog.regprocedure::text,
                       pg_catalog.pg_get_triggerdef(t.oid, false)
                   )
                   ORDER BY (
                       t.tgrelid::pg_catalog.regclass::text
                   ) COLLATE "C", t.tgname COLLATE "C"
               )::text, '[]'), 'UTF8')), 'hex')
      INTO huella_disparadores
      FROM pg_catalog.pg_trigger t
     WHERE t.tgrelid = ANY(tablas)
       AND NOT t.tgisinternal;

    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               COALESCE(pg_catalog.jsonb_agg(
                   pg_catalog.jsonb_build_array(
                       c.conrelid::pg_catalog.regclass::text, c.conname,
                       t.tgrelid::pg_catalog.regclass::text,
                       t.tgtype::integer, t.tgenabled::text,
                       t.tgdeferrable, t.tginitdeferred,
                       t.tgfoid::pg_catalog.regprocedure::text,
                       CASE WHEN t.tgconstrrelid = 0 THEN NULL
                            ELSE t.tgconstrrelid
                                 ::pg_catalog.regclass::text END,
                       CASE WHEN t.tgconstrindid = 0 THEN NULL
                            ELSE t.tgconstrindid
                                 ::pg_catalog.regclass::text END,
                       pg_catalog.encode(t.tgargs, 'hex'), t.tgattr::text,
                       pg_catalog.pg_get_expr(
                           t.tgqual, t.tgrelid, false
                       )
                   )
                   ORDER BY (
                       c.conrelid::pg_catalog.regclass::text
                   ) COLLATE "C", c.conname COLLATE "C",
                   (t.tgrelid::pg_catalog.regclass::text) COLLATE "C",
                   (t.tgfoid::pg_catalog.regprocedure::text) COLLATE "C",
                   t.tgtype
               )::text, '[]'), 'UTF8')), 'hex')
      INTO huella_disparadores_ri
      FROM pg_catalog.pg_trigger t
      JOIN pg_catalog.pg_constraint c ON c.oid = t.tgconstraint
     WHERE c.conrelid = ANY(tablas)
       AND t.tgisinternal;

    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               COALESCE(pg_catalog.jsonb_agg(
                   pg_catalog.jsonb_build_array(
                       regla.ev_class::pg_catalog.regclass::text,
                       regla.rulename, regla.ev_type::text,
                       regla.ev_enabled::text, regla.is_instead,
                       pg_catalog.pg_get_ruledef(regla.oid, false)
                   )
                   ORDER BY (
                       regla.ev_class::pg_catalog.regclass::text
                   ) COLLATE "C", regla.rulename COLLATE "C"
               )::text, '[]'), 'UTF8')), 'hex')
      INTO huella_reglas
      FROM pg_catalog.pg_rewrite regla
     WHERE regla.ev_class = ANY(tablas);

    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               COALESCE(pg_catalog.jsonb_agg(
                   pg_catalog.jsonb_build_array(
                       n.nspname, p.proname,
                       pg_catalog.pg_get_function_identity_arguments(p.oid),
                       pg_catalog.pg_get_function_result(p.oid),
                       l.lanname,
                       p.proowner::pg_catalog.regrole::text,
                       p.prokind::text, p.prosecdef, p.proleakproof,
                       p.proisstrict, p.provolatile::text,
                       p.proparallel::text, p.procost, p.prorows,
                       p.prosupport::pg_catalog.regprocedure::text,
                       ARRAY(
                           SELECT opcion
                             FROM pg_catalog.unnest(p.proconfig) opcion
                            ORDER BY opcion COLLATE "C"
                       ),
                       pg_catalog.obj_description(p.oid, 'pg_proc'),
                       pg_catalog.pg_get_functiondef(p.oid)
                   )
               )::text, '[]'), 'UTF8')), 'hex')
      INTO huella_funcion
      FROM pg_catalog.pg_proc p
      JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
      JOIN pg_catalog.pg_language l ON l.oid = p.prolang
     WHERE p.oid = funcion;

    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 19
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 3
    ) OR funcion IS NULL OR indice IS NULL
      OR (
          SELECT pg_catalog.count(*)
            FROM vec_contratacion_temporal
                 .control_registrador_acceso_rrhh_v2
           WHERE control AND version_esquema = 1
             AND prueba_huella_sha256 = pg_catalog.encode(
                 pg_catalog.sha256(prueba_canonica), 'hex'
             )
      ) <> 1
      OR (
          SELECT pg_catalog.count(*)
            FROM vec_contratacion_temporal
                 .control_registrador_acceso_rrhh_v2
      ) <> 1
      OR huella_relaciones IS DISTINCT FROM
         'd78b01c877b644487662a3b79d0ca776c262c7f30ec85bb5e62d3e647da8663e'
      OR huella_columnas IS DISTINCT FROM
         '3c974e312e438c48af1714a0433e88b5bd9acd3a703bb8ec65d74f6157618b80'
      OR huella_indices IS DISTINCT FROM
         'a4568223df3f4665b5f11771fa2e51cab41d64537740420c85bc385cb74e23b0'
      OR huella_politicas IS DISTINCT FROM
         'b0dae1468ed281d5d8bc579e20ee0382642ad2d60ab7f05fd0a2770cd7fd9571'
      OR huella_acl IS DISTINCT FROM
         '6850ab0b95f24871a9186dc477f64d1b22e348f4787baf71fe8705754eec22e0'
      OR huella_restricciones IS DISTINCT FROM
         '08c3430b38558227d352ef74b6387771ab2a9c99ca033ff09c02fdbdc21accde'
      OR huella_disparadores IS DISTINCT FROM
         'b06111aba3ea491e139910458241d5aaece59c2694d0f7aa6a62fc7751866487'
      OR huella_disparadores_ri IS DISTINCT FROM
         'dda0fd372fce52bc5f33032ed1eb67089a6d8da6e2ecb55dad0de24c2657817a'
      OR huella_reglas IS DISTINCT FROM
         '4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945'
      OR huella_funcion IS DISTINCT FROM
         '53f11b9a6c0ac408ff4089be340011ae8ebec24803cfcb0ab1da5ad8c7add376'
      OR EXISTS (
          SELECT 1
            FROM pg_catalog.pg_trigger t
            JOIN pg_catalog.pg_constraint c
              ON c.oid = t.tgconstraint
           WHERE c.conrelid = ANY(tablas)
             AND t.tgisinternal
             AND t.tgenabled <> 'O'
      ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down del registrador RRHH v2 rechazado por deriva';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_constraint c
         WHERE c.confrelid = ANY(tablas)
           AND c.conrelid <> ALL(tablas)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_rewrite regla
          JOIN pg_catalog.pg_depend dependencia
            ON dependencia.classid =
               'pg_catalog.pg_rewrite'::regclass
           AND dependencia.objid = regla.oid
         WHERE dependencia.refclassid =
               'pg_catalog.pg_class'::regclass
           AND dependencia.refobjid = ANY(tablas)
           AND regla.ev_class <> ALL(tablas)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_depend dependencia
         WHERE (
             dependencia.refclassid = 'pg_catalog.pg_proc'::regclass
             AND dependencia.refobjid = funcion
         ) OR (
             dependencia.refclassid = 'pg_catalog.pg_class'::regclass
             AND dependencia.refobjid = indice
         )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc p
         WHERE p.oid <> funcion
           AND p.prokind IN ('f', 'p')
           AND (
               p.prosrc LIKE
                   '%registrar_acceso_rrhh_interno_v2%'
               OR p.prosrc LIKE
                   '%control_registrador_acceso_rrhh_v2%'
               OR p.prosrc LIKE
                   '%vinculo_identidad_acceso_rrhh_v2%'
           )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_inherits herencia
         WHERE herencia.inhrelid = ANY(tablas)
            OR herencia.inhparent = ANY(tablas)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_publication_rel publicacion
         WHERE publicacion.prrelid = ANY(tablas)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_statistic_ext estadistica
         WHERE estadistica.stxrelid = ANY(tablas)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_seclabel etiqueta
         WHERE (
             etiqueta.classoid = 'pg_catalog.pg_class'::regclass
             AND etiqueta.objoid = ANY(tablas || ARRAY[indice]::oid[])
         ) OR (
             etiqueta.classoid = 'pg_catalog.pg_proc'::regclass
             AND etiqueta.objoid = funcion
         )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'dependencias impiden retirar registrador RRHH v2';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .vinculo_identidad_acceso_rrhh_v2
    ) OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .control_registrador_acceso_rrhh_v2 base
          JOIN vec_contratacion_temporal
               .control_cadena_accesos_rrhh cadena
            ON cadena.control = base.control
         WHERE base.control
           AND (
               cadena.ultima_secuencia <> base.secuencia_base
               OR cadena.cabeza_sha256 <> base.cabeza_base_sha256
           )
    ) OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.registro_acceso_rrhh acceso
          CROSS JOIN vec_contratacion_temporal
               .control_registrador_acceso_rrhh_v2 base
         WHERE base.control
           AND acceso.secuencia > base.secuencia_base
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'historia v2 impide retirar el registrador RRHH';
    END IF;
END
$prevalidacion$;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(jsonb)
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;
DROP FUNCTION
    vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(jsonb)
    RESTRICT;
DROP TABLE
    vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2
    RESTRICT;
DROP TABLE
    vec_contratacion_temporal.control_registrador_acceso_rrhh_v2
    RESTRICT;
DROP INDEX
    vec_contratacion_temporal.
    publicacion_rrhh_organizacion_expediente_corte_desc_idx
    RESTRICT;

UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 2,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 3;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 18,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 19;

COMMIT;
