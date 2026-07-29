-- CT-000043: ninguna dependencia futura puede desaparecer con las columnas.
DO $guardia_columnas_padre$
DECLARE
    v_relacion oid :=
        'vec_contratacion_temporal.'
        'registro_acceso_rrhh'::pg_catalog.regclass;
    v_columnas smallint[];
BEGIN
    SELECT pg_catalog.array_agg(
               atributo.attnum::smallint
               ORDER BY atributo.attname COLLATE "C"
           )
      INTO STRICT v_columnas
      FROM pg_catalog.pg_attribute atributo
     WHERE atributo.attrelid = v_relacion
       AND atributo.attname IN (
           'expediente_ref_prueba_v2',
           'version_expediente_prueba_v2'
       )
       AND NOT atributo.attisdropped;

    IF pg_catalog.cardinality(v_columnas) <> 2
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_index indice
            WHERE indice.indrelid = v_relacion
              AND (
                  indice.indkey::smallint[] && v_columnas
                  OR EXISTS (
                      SELECT 1
                        FROM pg_catalog.pg_depend dependencia
                       WHERE dependencia.classid =
                             'pg_catalog.pg_class'::pg_catalog.regclass
                         AND dependencia.objid = indice.indexrelid
                         AND dependencia.refclassid =
                             'pg_catalog.pg_class'::pg_catalog.regclass
                         AND dependencia.refobjid = v_relacion
                         AND dependencia.refobjsubid =
                             ANY(v_columnas)
                  )
              )
              AND NOT EXISTS (
                  SELECT 1
                    FROM pg_catalog.pg_constraint restriccion
                   WHERE restriccion.conindid = indice.indexrelid
                     AND restriccion.conrelid = v_relacion
                     AND restriccion.conname = ANY(ARRAY[
                         'registro_acceso_rrhh_prueba_resultado_v2_unica',
                         'registro_acceso_rrhh_prueba_cadena_v2_unica',
                         'registro_acceso_rrhh_prueba_vec_v2_unica'
                     ]::name[])
              )
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_constraint restriccion
            WHERE (
                restriccion.conrelid = v_relacion
                AND restriccion.conkey::smallint[] && v_columnas
                OR EXISTS (
                    SELECT 1
                      FROM pg_catalog.pg_depend dependencia
                     WHERE dependencia.classid =
                           'pg_catalog.pg_constraint'
                           ::pg_catalog.regclass
                       AND dependencia.objid = restriccion.oid
                       AND dependencia.refclassid =
                           'pg_catalog.pg_class'::pg_catalog.regclass
                       AND dependencia.refobjid = v_relacion
                       AND dependencia.refobjsubid = ANY(v_columnas)
                )
            )
              AND NOT (
                  restriccion.conrelid = v_relacion
                  AND restriccion.conname = ANY(ARRAY[
                      'registro_acceso_rrhh_prueba_resultado_v2_unica',
                      'registro_acceso_rrhh_prueba_cadena_v2_unica',
                      'registro_acceso_rrhh_prueba_vec_v2_unica'
                  ]::name[])
                  OR restriccion.conrelid =
                     'vec_contratacion_temporal.'
                     'prueba_resultado_recibo_rrhh_v2'
                     ::pg_catalog.regclass
              )
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_statistic_ext estadistica
            WHERE estadistica.stxrelid = v_relacion
              AND (
                  estadistica.stxkeys::smallint[] && v_columnas
                  OR EXISTS (
                      SELECT 1
                        FROM pg_catalog.pg_depend dependencia
                       WHERE dependencia.classid =
                             'pg_catalog.pg_statistic_ext'
                             ::pg_catalog.regclass
                         AND dependencia.objid = estadistica.oid
                         AND dependencia.refclassid =
                             'pg_catalog.pg_class'::pg_catalog.regclass
                         AND dependencia.refobjid = v_relacion
                         AND dependencia.refobjsubid =
                             ANY(v_columnas)
                  )
              )
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_publication_rel pertenencia
            WHERE pertenencia.prrelid = v_relacion
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_inherits herencia
            WHERE herencia.inhparent = v_relacion
               OR herencia.inhrelid = v_relacion
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE =
                'dependencias de columnas de prueba RRHH incompatibles';
    END IF;
END
$guardia_columnas_padre$;
