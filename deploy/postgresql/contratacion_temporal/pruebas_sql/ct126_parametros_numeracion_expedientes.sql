-- Pruebas funcionales y negativas de CT-000126 (parámetros de numeración).
-- Se ejecuta como superusuario de un PostgreSQL desechable; las llamadas de
-- la aplicación se hacen con SET SESSION AUTHORIZATION a los inicios de sesión
-- reales del volcado: vec_ad3_o207_gobierno (gobernador) publica y
-- vec_ct_o207_runtime (ejecutor) reserva números. Cada comprobación imprime
-- «ok»; un fallo imprime «FALLO ...» y detiene el ensayo.
\set ON_ERROR_STOP on
\pset tuples_only on
\pset format unaligned

CREATE TEMP TABLE ct126_antes AS
SELECT pg_catalog.md5(pg_catalog.string_agg(expediente_ref || '=' || numero_visible, ',' ORDER BY expediente_ref)) AS huella,
       pg_catalog.count(*) AS total
  FROM vec_contratacion_temporal.expediente_alta;
GRANT SELECT ON ct126_antes TO PUBLIC;

CREATE FUNCTION pg_temp.comprobar(p_condicion boolean, p_nombre text)
RETURNS text LANGUAGE plpgsql AS $$
BEGIN
    IF p_condicion IS DISTINCT FROM true THEN
        RAISE EXCEPTION 'FALLO %', p_nombre;
    END IF;
    RETURN 'ok';
END $$;

-- 1. Sin versión publicada rige el formato de siempre, sin escribir historia.
SET SESSION AUTHORIZATION vec_ad3_o207_gobierno;
SELECT pg_temp.comprobar(r.resultado = 'vigente' AND r.version = 0, 'predeterminada sin historia')
  FROM vec_contratacion_temporal.publicar_numeracion_parametros_v1(
      'CT-', 6, 'configuracion:ct:numeracion:predeterminada') r;
SELECT pg_temp.comprobar(r.resultado = 'vigente' AND r.version = 0, 'segundo arranque sin historia')
  FROM vec_contratacion_temporal.publicar_numeracion_parametros_v1(
      'CT-', 6, 'configuracion:ct:numeracion:predeterminada') r;
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar(pg_catalog.count(*) = 0, 'historia vacía con el formato de siempre')
  FROM vec_contratacion_temporal.numeracion_parametros;

SET SESSION AUTHORIZATION vec_ct_o207_runtime;
SELECT pg_temp.comprobar(
    vec_contratacion_temporal.siguiente_numero_visible_v1(2031) = '2031/CT-000001',
    'formato de siempre');
RESET SESSION AUTHORIZATION;

-- 2. Un formato propio añade la versión 1; repetirlo no añade otra.
SET SESSION AUTHORIZATION vec_ad3_o207_gobierno;
SELECT pg_temp.comprobar(r.resultado = 'publicada' AND r.version = 1, 'formato propio publicado')
  FROM vec_contratacion_temporal.publicar_numeracion_parametros_v1(
      'CTEMP-', 4, 'vec.contratacion_temporal.reglas:1:c16.numeracion') r;
SELECT pg_temp.comprobar(r.resultado = 'vigente' AND r.version = 1, 'mismo formato no republica')
  FROM vec_contratacion_temporal.publicar_numeracion_parametros_v1(
      'CTEMP-', 4, 'vec.contratacion_temporal.reglas:2:c16.numeracion') r;
RESET SESSION AUTHORIZATION;

SET SESSION AUTHORIZATION vec_ct_o207_runtime;
SELECT pg_temp.comprobar(
    vec_contratacion_temporal.siguiente_numero_visible_v1(2031) = '2031/CTEMP-0002',
    'formato propio aplicado sin reiniciar el contador');
RESET SESSION AUTHORIZATION;

-- 3. Menos dígitos que el número: se escribe entero, nunca se recorta.
UPDATE vec_contratacion_temporal.numeracion_expedientes SET ultimo = 12344 WHERE anio = 2031;
SET SESSION AUTHORIZATION vec_ct_o207_runtime;
SELECT pg_temp.comprobar(
    vec_contratacion_temporal.siguiente_numero_visible_v1(2031) = '2031/CTEMP-12345',
    'número más largo que los dígitos');
RESET SESSION AUTHORIZATION;

-- 4. Prefijo vacío y vuelta al formato de siempre: versiones 2 y 3.
SET SESSION AUTHORIZATION vec_ad3_o207_gobierno;
SELECT pg_temp.comprobar(r.resultado = 'publicada' AND r.version = 2, 'prefijo vacío')
  FROM vec_contratacion_temporal.publicar_numeracion_parametros_v1(
      '', 5, 'vec.contratacion_temporal.reglas:3:c16.numeracion') r;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_ct_o207_runtime;
SELECT pg_temp.comprobar(
    vec_contratacion_temporal.siguiente_numero_visible_v1(2032) = '2032/00001',
    'prefijo vacío aplicado');
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_ad3_o207_gobierno;
SELECT pg_temp.comprobar(r.resultado = 'publicada' AND r.version = 3, 'vuelta al formato de siempre')
  FROM vec_contratacion_temporal.publicar_numeracion_parametros_v1(
      'CT-', 6, 'configuracion:ct:numeracion:predeterminada') r;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_ct_o207_runtime;
SELECT pg_temp.comprobar(
    vec_contratacion_temporal.siguiente_numero_visible_v1(2032) = '2032/CT-000002',
    'formato de siempre restablecido');
RESET SESSION AUTHORIZATION;

-- 5. Negativos: parámetros no válidos, privilegios y historia inmutable.
SET SESSION AUTHORIZATION vec_ad3_o207_gobierno;
DO $$
DECLARE v_caso text[];
BEGIN
    FOREACH v_caso SLICE 1 IN ARRAY ARRAY[
        ARRAY['CT/', '6'], ARRAY['CT-', '0'], ARRAY['CT-', '10'],
        ARRAY['C T', '6'], ARRAY[repeat('A', 21), '6']
    ] LOOP
        BEGIN
            PERFORM vec_contratacion_temporal.publicar_numeracion_parametros_v1(
                v_caso[1], v_caso[2]::integer, 'configuracion:ct:numeracion:prueba');
            RAISE EXCEPTION 'FALLO se aceptó %/%', v_caso[1], v_caso[2];
        EXCEPTION WHEN invalid_parameter_value THEN NULL;
        END;
    END LOOP;
    BEGIN
        PERFORM vec_contratacion_temporal.publicar_numeracion_parametros_v1('CT-', 6, 'x');
        RAISE EXCEPTION 'FALLO se aceptó una fuente no válida';
    EXCEPTION WHEN invalid_parameter_value THEN NULL;
    END;
    BEGIN
        PERFORM 1 FROM vec_contratacion_temporal.numeracion_parametros;
        RAISE EXCEPTION 'FALLO el gobernador lee la tabla';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
END $$;
SELECT 'ok';
RESET SESSION AUTHORIZATION;

SET SESSION AUTHORIZATION vec_ct_o207_runtime;
DO $$
BEGIN
    BEGIN
        PERFORM vec_contratacion_temporal.publicar_numeracion_parametros_v1('X-', 3, 'configuracion:ct:numeracion:prueba');
        RAISE EXCEPTION 'FALLO el ejecutor publica';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
    BEGIN
        PERFORM 1 FROM vec_contratacion_temporal.numeracion_parametros;
        RAISE EXCEPTION 'FALLO el ejecutor lee la tabla';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
END $$;
SELECT 'ok';
RESET SESSION AUTHORIZATION;

-- El superusuario es miembro de todo: la frontera lo rechaza igualmente.
DO $$
BEGIN
    PERFORM vec_contratacion_temporal.publicar_numeracion_parametros_v1('X-', 3, 'configuracion:ct:numeracion:prueba');
    RAISE EXCEPTION 'FALLO publicó una sesión con varios roles';
EXCEPTION WHEN insufficient_privilege THEN NULL;
END $$;
SELECT 'ok';

SET ROLE vec_contratacion_temporal_propietario;
DO $$
BEGIN
    BEGIN
        UPDATE vec_contratacion_temporal.numeracion_parametros SET prefijo = 'X-';
        RAISE EXCEPTION 'FALLO se modificó la historia';
    EXCEPTION WHEN OTHERS THEN
        IF SQLERRM LIKE 'FALLO%' THEN RAISE; END IF;
    END;
    BEGIN
        DELETE FROM vec_contratacion_temporal.numeracion_parametros;
        RAISE EXCEPTION 'FALLO se borró la historia';
    EXCEPTION WHEN OTHERS THEN
        IF SQLERRM LIKE 'FALLO%' THEN RAISE; END IF;
    END;
    BEGIN
        TRUNCATE vec_contratacion_temporal.numeracion_parametros;
        RAISE EXCEPTION 'FALLO se vació la historia';
    EXCEPTION WHEN OTHERS THEN
        IF SQLERRM LIKE 'FALLO%' THEN RAISE; END IF;
    END;
    BEGIN
        INSERT INTO vec_contratacion_temporal.numeracion_parametros VALUES
            (9, 'X-', 3, repeat('a', 64), 'configuracion:ct:numeracion:prueba', 'x', now());
        RAISE EXCEPTION 'FALLO se aceptó una huella falsa';
    EXCEPTION WHEN check_violation THEN NULL;
    END;
END $$;
SELECT 'ok';
RESET ROLE;

-- 6. Los expedientes existentes conservan su número.
SELECT pg_temp.comprobar(
    (SELECT pg_catalog.md5(pg_catalog.string_agg(expediente_ref || '=' || numero_visible, ',' ORDER BY expediente_ref))
       FROM vec_contratacion_temporal.expediente_alta) IS NOT DISTINCT FROM a.huella
    AND (SELECT pg_catalog.count(*) FROM vec_contratacion_temporal.expediente_alta) = a.total,
    'expedientes existentes sin renumerar')
  FROM ct126_antes a;
SELECT pg_temp.comprobar(pg_catalog.count(*) = 3 AND pg_catalog.max(version) = 3
    AND pg_catalog.bool_and(publicado_por = 'vec_ad3_o207_gobierno'), 'historia de tres versiones')
  FROM vec_contratacion_temporal.numeracion_parametros;
SELECT 'CT126 OK';
