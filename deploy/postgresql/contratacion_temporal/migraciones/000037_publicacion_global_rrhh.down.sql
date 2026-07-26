-- Reversión segura de O4-05/C2-C. La proyección es derivada, pero solo puede
-- retirarse si desde el backfill no se publicó ninguna versión ni se usó.
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
   AND version_esquema = 17
 FOR UPDATE;

-- Fija historia y proyección mientras se comprueba que siguen siendo 1:1.
LOCK TABLE vec_contratacion_temporal.expediente_version_integral
    IN SHARE MODE;
LOCK TABLE vec_contratacion_temporal.publicacion_version_rrhh
    IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
DECLARE
    v_control record;
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control
           AND version_esquema = 17
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .control_migracion_consultas_rrhh
         WHERE control
           AND version_esquema = 1
    ) OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.control_publicacion_rrhh'
    ) IS NULL OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.publicacion_version_rrhh'
    ) IS NULL OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.extraer_publicacion_rrhh_v1(text,numeric,jsonb,text,text,numeric,text,text,text,timestamp with time zone,text,text)'
    ) IS NULL OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.publicar_version_rrhh_v1()'
    ) IS NULL OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_trigger
         WHERE tgrelid =
               'vec_contratacion_temporal.expediente_version_integral'
                   ::regclass
           AND tgname = 'expediente_version_integral_publicar_rrhh'
           AND tgenabled = 'O'
           AND NOT tgisinternal
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'down de publicación global RRHH fuera de orden';
    END IF;

    SELECT corte_base, ultimo_corte
      INTO STRICT v_control
      FROM vec_contratacion_temporal.control_publicacion_rrhh
     WHERE control
     FOR UPDATE;
    IF v_control.ultimo_corte <> v_control.corte_base
       OR EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.publicacion_version_rrhh
            WHERE corte_global > v_control.corte_base
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'publicaciones posteriores impiden retirar corte RRHH';
    END IF;

    -- Los accesos C2-B preexistentes no usan ni referencian esta proyección.
    -- Una futura fachada/cursor C2-C sí queda cercada nominalmente y por sus
    -- dependencias, además de la barrera global exacta.
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class objeto
          JOIN pg_catalog.pg_namespace esquema
            ON esquema.oid = objeto.relnamespace
         WHERE esquema.nspname = 'vec_contratacion_temporal'
           AND objeto.relname ~
               '(^cursor_.*rrhh|^rrhh_.*cursor|^cursor_cuadro_rrhh$)'
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc objeto
          JOIN pg_catalog.pg_namespace esquema
            ON esquema.oid = objeto.pronamespace
         WHERE esquema.nspname = 'vec_contratacion_temporal'
           AND objeto.proname ~
               '(^consultar_(cuadro|detalle)_rrhh|cursor.*rrhh|rrhh.*cursor)'
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'fachadas o cursores impiden retirar corte RRHH';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_constraint dependencia
         WHERE dependencia.confrelid =
               'vec_contratacion_temporal.publicacion_version_rrhh'
                   ::regclass
           AND dependencia.conrelid <>
               'vec_contratacion_temporal.publicacion_version_rrhh'
                   ::regclass
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_depend dependencia
          JOIN pg_catalog.pg_rewrite regla
            ON dependencia.classid = 'pg_catalog.pg_rewrite'::regclass
           AND regla.oid = dependencia.objid
         WHERE dependencia.refobjid =
               'vec_contratacion_temporal.publicacion_version_rrhh'
                   ::regclass
           AND regla.ev_class <>
               'vec_contratacion_temporal.publicacion_version_rrhh'
                   ::regclass
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_trigger disparador
         WHERE disparador.tgrelid =
               'vec_contratacion_temporal.publicacion_version_rrhh'
                   ::regclass
           AND NOT disparador.tgisinternal
           AND disparador.tgname NOT IN (
               'publicacion_version_rrhh_inmutable',
               'publicacion_version_rrhh_no_truncar'
           )
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'dependencias futuras impiden retirar corte RRHH';
    END IF;

    IF (SELECT pg_catalog.count(*)
          FROM vec_contratacion_temporal.publicacion_version_rrhh)
           <> v_control.corte_base
       OR COALESCE((
           SELECT pg_catalog.max(corte_global)
             FROM vec_contratacion_temporal.publicacion_version_rrhh
       ), 0) <> v_control.corte_base
       OR EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .expediente_version_integral historia
        FULL JOIN vec_contratacion_temporal.publicacion_version_rrhh publicacion
               ON publicacion.expediente_ref = historia.expediente_ref
              AND publicacion.version = historia.version
            WHERE historia.expediente_ref IS NULL
               OR publicacion.expediente_ref IS NULL
       ) OR EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .expediente_version_integral historia
             JOIN vec_contratacion_temporal.expediente_alta alta
               ON alta.expediente_ref = historia.expediente_ref
             JOIN vec_contratacion_temporal.publicacion_version_rrhh publicacion
               ON publicacion.expediente_ref = historia.expediente_ref
              AND publicacion.version = historia.version
       CROSS JOIN LATERAL
            vec_contratacion_temporal.extraer_publicacion_rrhh_v1(
                historia.expediente_ref, historia.version,
                historia.agregado_json,
                historia.agregado_json_huella_sha256,
                historia.flujo_ref, historia.flujo_version,
                historia.flujo_huella_sha256, historia.fase_clave,
                historia.estado, historia.registrada_en,
                alta.organizacion_ref, alta.numero_visible
            ) extraida
            WHERE publicacion.organizacion_ref
                      IS DISTINCT FROM extraida.organizacion_ref
               OR publicacion.numero_visible
                      IS DISTINCT FROM extraida.numero_visible
               OR publicacion.flujo_ref
                      IS DISTINCT FROM historia.flujo_ref
               OR publicacion.flujo_version
                      IS DISTINCT FROM historia.flujo_version
               OR publicacion.flujo_huella_sha256
                      IS DISTINCT FROM historia.flujo_huella_sha256
               OR publicacion.fase_clave
                      IS DISTINCT FROM historia.fase_clave
               OR publicacion.estado_clave
                      IS DISTINCT FROM historia.estado
               OR publicacion.centro_ref
                      IS DISTINCT FROM extraida.centro_ref
               OR publicacion.categoria_ref
                      IS DISTINCT FROM extraida.categoria_ref
               OR publicacion.modalidad_clave
                      IS DISTINCT FROM extraida.modalidad_clave
               OR publicacion.unidad_ref
                      IS DISTINCT FROM extraida.unidad_ref
               OR publicacion.creado_en
                      IS DISTINCT FROM extraida.creado_en
               OR publicacion.actualizado_en
                      IS DISTINCT FROM extraida.actualizado_en
               OR publicacion.agregado_huella_sha256
                      IS DISTINCT FROM
                         historia.agregado_json_huella_sha256
               OR publicacion.registrada_en
                      IS DISTINCT FROM historia.registrada_en
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'divergencia impide retirar corte RRHH';
    END IF;
END
$prevalidacion$;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.extraer_publicacion_rrhh_v1(
        text, numeric, jsonb, text, text, numeric, text, text, text,
        timestamptz, text, text
    ),
    vec_contratacion_temporal.publicar_version_rrhh_v1()
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;

DROP TRIGGER expediente_version_integral_publicar_rrhh
    ON vec_contratacion_temporal.expediente_version_integral;
DROP TRIGGER publicacion_version_rrhh_inmutable
    ON vec_contratacion_temporal.publicacion_version_rrhh;
DROP TRIGGER publicacion_version_rrhh_no_truncar
    ON vec_contratacion_temporal.publicacion_version_rrhh;
DROP FUNCTION vec_contratacion_temporal.publicar_version_rrhh_v1();
DROP FUNCTION vec_contratacion_temporal.extraer_publicacion_rrhh_v1(
    text, numeric, jsonb, text, text, numeric, text, text, text,
    timestamptz, text, text
);
DROP TABLE vec_contratacion_temporal.publicacion_version_rrhh;
DROP TABLE vec_contratacion_temporal.control_publicacion_rrhh;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 16,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds',
           pg_catalog.clock_timestamp()
       )
 WHERE control
   AND version_esquema = 17;

COMMIT;
