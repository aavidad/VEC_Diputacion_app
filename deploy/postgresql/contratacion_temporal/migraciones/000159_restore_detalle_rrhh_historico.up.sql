\set ON_ERROR_STOP on
-- CT159: conservar el contenido histórico y comprobar el canon al insertar.
-- pg_dump instala los triggers después de cargar los datos históricos.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_05:consultas_rrhh:migraciones', 0
));
LOCK TABLE vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
    IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_constraint restriccion
        JOIN pg_catalog.pg_class tabla ON tabla.oid = restriccion.conrelid
        WHERE restriccion.conrelid =
            'vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2'::regclass
          AND restriccion.conname = 'prueba_resultado_recibo_rrhh_v2_check1'
          AND restriccion.contype = 'c'
          AND restriccion.convalidated
          AND tabla.relowner =
              'vec_contratacion_temporal_propietario'::regrole
          AND pg_catalog.md5(pg_catalog.pg_get_constraintdef(restriccion.oid)) =
              'e71cec4ee064be28db9316fd8350dc22'
    ) OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.validar_detalle_nuevo_recibo_rrhh_ct159()'
    ) IS NOT NULL OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_trigger
        WHERE tgrelid =
            'vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2'::regclass
          AND tgname = 'prueba_resultado_recibo_rrhh_v2_detalle_nuevo_ct159'
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_attribute
        WHERE attrelid =
            'vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1'::regclass
          AND attname = 'fiscalizacion_presente'
          AND atttypid = 'boolean'::regtype AND NOT attisdropped
    ) OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1('
        || 'timestamp with time zone,'
        || 'vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1)'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'preimagen incompatible para CT159';
    END IF;
END
$prevalidacion$;

ALTER TABLE vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
    DROP CONSTRAINT prueba_resultado_recibo_rrhh_v2_check1;
SET LOCAL statement_timeout = '20min';
ALTER TABLE vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
    ADD CONSTRAINT prueba_resultado_recibo_rrhh_v2_check1
    CHECK (
        (
            tipo_consulta = 'cuadro'
            AND total = pg_catalog.cardinality(resumenes)
            AND detalle IS NOT DISTINCT FROM
                NULL::vec_contratacion_temporal
                    .entrada_detalle_expediente_rrhh_v1
            AND contenido_canonico =
                vec_contratacion_temporal
                .canon_contenido_cuadro_rrhh_v1(
                    generada_en, resumenes, hay_mas,
                    cursor_material_huella_sha256
                )
            AND cursor_huella_sha256 IS NOT DISTINCT FROM
                CASE WHEN hay_mas THEN pg_catalog.encode(
                    cursor_material_huella_sha256, 'hex'
                ) ELSE NULL END
        )
        OR (
            tipo_consulta = 'detalle'
            AND pg_catalog.array_ndims(resumenes) IS NULL
            AND NOT hay_mas
            AND pg_catalog.octet_length(
                cursor_material_huella_sha256
            ) = 0
            AND detalle IS DISTINCT FROM
                NULL::vec_contratacion_temporal
                    .entrada_detalle_expediente_rrhh_v1
            AND cursor_huella_sha256 IS NULL
        )
    );
SET LOCAL statement_timeout = '30s';

CREATE FUNCTION
vec_contratacion_temporal.validar_detalle_nuevo_recibo_rrhh_ct159()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
SECURITY INVOKER
SET search_path = pg_catalog
SET timezone = 'UTC'
AS $funcion$
BEGIN
    IF NEW.tipo_consulta = 'detalle' THEN
        IF NEW.contenido_canonico IS DISTINCT FROM
            vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
                NEW.generada_en, NEW.detalle
            ) THEN
            RAISE EXCEPTION USING ERRCODE = '23514',
                CONSTRAINT = 'prueba_resultado_recibo_rrhh_v2_detalle_nuevo_ct159',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
    END IF;
    RETURN NEW;
END
$funcion$;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.validar_detalle_nuevo_recibo_rrhh_ct159()
FROM PUBLIC;

CREATE TRIGGER prueba_resultado_recibo_rrhh_v2_detalle_nuevo_ct159
BEFORE INSERT ON vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
FOR EACH ROW EXECUTE FUNCTION
    vec_contratacion_temporal.validar_detalle_nuevo_recibo_rrhh_ct159();

COMMIT;
