\set ON_ERROR_STOP on

DO $inmutabilidad$
DECLARE
    v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'alcance_acceso_rrhh',
        'familia_cursor_cuadro_rrhh',
        'cursor_cuadro_rrhh',
        'consumo_cursor_cuadro_rrhh',
        'revocacion_familia_cursor_rrhh'
    ]::text[] LOOP
        BEGIN
            EXECUTE pg_catalog.format(
                'UPDATE vec_contratacion_temporal.%I '
                || 'SET prueba_huella_sha256=prueba_huella_sha256',
                v_tabla
            );
            RAISE EXCEPTION 'UPDATE aceptado en %', v_tabla;
        EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
        END;
        BEGIN
            EXECUTE pg_catalog.format(
                'DELETE FROM vec_contratacion_temporal.%I', v_tabla
            );
            RAISE EXCEPTION 'DELETE aceptado en %', v_tabla;
        EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
        END;
    END LOOP;
    BEGIN
        TRUNCATE
            vec_contratacion_temporal.revocacion_familia_cursor_rrhh,
            vec_contratacion_temporal.consumo_cursor_cuadro_rrhh,
            vec_contratacion_temporal.cursor_cuadro_rrhh,
            vec_contratacion_temporal.familia_cursor_cuadro_rrhh,
            vec_contratacion_temporal.alcance_acceso_rrhh;
        RAISE EXCEPTION 'TRUNCATE conjunto aceptado';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
END
$inmutabilidad$;

RESET ROLE;
