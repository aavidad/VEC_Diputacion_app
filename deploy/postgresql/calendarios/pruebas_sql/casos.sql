\set ON_ERROR_STOP on
-- Casos de Calendarios sobre PostgreSQL desechable. Requiere roles, 000001 y
-- 000002 instalados. Ejecutar como superusuario de la instancia de prueba.
SET search_path=pg_catalog;
SET timezone='UTC';

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_prueba_calendarios_lector') THEN
    CREATE ROLE vec_prueba_calendarios_lector LOGIN INHERIT;
    GRANT vec_calendarios_lector TO vec_prueba_calendarios_lector WITH INHERIT TRUE, SET FALSE;
    CREATE ROLE vec_prueba_calendarios_ajeno LOGIN;
  END IF;
END $$;

CREATE OR REPLACE FUNCTION pg_temp.espera_error(sentencia text, codigo text, rol text) RETURNS void LANGUAGE plpgsql AS $f$
DECLARE obtenido text;
BEGIN
  BEGIN
    EXECUTE format('SET LOCAL ROLE %I', rol);
    EXECUTE sentencia;
    obtenido := 'ninguno';
  EXCEPTION WHEN OTHERS THEN
    obtenido := SQLSTATE;
  END;
  RESET ROLE;
  IF obtenido IS DISTINCT FROM codigo THEN
    RAISE EXCEPTION 'esperado % y obtenido % en: %', codigo, obtenido, sentencia;
  END IF;
END $f$;

-- 1. Estructura: RLS forzada, funciones SECURITY DEFINER con search_path
--    fijo, sin EXECUTE ni privilegios de tabla para PUBLIC ni el lector.
DO $$ BEGIN
  IF (SELECT count(*) FROM pg_class WHERE relnamespace='vec_calendarios'::regnamespace AND relkind='r'
        AND relrowsecurity AND relforcerowsecurity AND relowner='vec_calendarios_propietario'::regrole)<>2
     OR EXISTS (SELECT 1 FROM pg_policy p JOIN pg_class c ON c.oid=p.polrelid
                 WHERE c.relnamespace='vec_calendarios'::regnamespace AND p.polroles<>ARRAY['vec_calendarios_propietario'::regrole::oid])
     OR (SELECT count(*) FROM pg_proc WHERE pronamespace='vec_calendarios'::regnamespace AND prosecdef
           AND proconfig=ARRAY['search_path=pg_catalog, pg_temp'])<>2
     OR (SELECT count(*) FROM pg_proc WHERE pronamespace='vec_calendarios'::regnamespace AND prosecdef)<>2
     OR EXISTS (SELECT 1 FROM pg_proc WHERE pronamespace='vec_calendarios'::regnamespace AND has_function_privilege('public', oid, 'EXECUTE'))
     OR has_table_privilege('vec_calendarios_lector','vec_calendarios.version_calendario','SELECT')
     OR has_table_privilege('vec_calendarios_lector','vec_calendarios.dia_calendario','SELECT,INSERT,UPDATE,DELETE')
     OR NOT has_function_privilege('vec_calendarios_lector','vec_calendarios.versiones_vigentes_v1(integer,text[],text[],timestamptz)','EXECUTE')
     OR has_schema_privilege('vec_prueba_calendarios_ajeno','vec_calendarios','USAGE')
  THEN RAISE EXCEPTION 'estructura o ACL de Calendarios incorrecta'; END IF;
END $$;

-- 2. El lector consulta por función, no por tabla, y no escribe.
SELECT pg_temp.espera_error('SELECT count(*) FROM vec_calendarios.version_calendario', '42501', 'vec_prueba_calendarios_lector');
SELECT pg_temp.espera_error($q$INSERT INTO vec_calendarios.dia_calendario VALUES ('calendario:nacional:es:2026:v1','2026-03-03','festivo','x')$q$, '42501', 'vec_prueba_calendarios_lector');
SELECT pg_temp.espera_error($q$SELECT * FROM vec_calendarios.centros_con_calendario_v1(2026, now())$q$, '42501', 'vec_prueba_calendarios_ajeno');
SELECT pg_temp.espera_error($q$SELECT * FROM vec_calendarios.versiones_vigentes_v1(2026, ARRAY['nacional'], ARRAY['es'], now()+interval '1 hour')$q$, '22023', 'vec_prueba_calendarios_lector');
SELECT pg_temp.espera_error($q$SELECT * FROM vec_calendarios.versiones_vigentes_v1(2026, ARRAY['nacional','local'], ARRAY['es'], now())$q$, '22023', 'vec_prueba_calendarios_lector');
SELECT pg_temp.espera_error($q$SELECT * FROM vec_calendarios.versiones_vigentes_v1(1999, ARRAY['nacional'], ARRAY['es'], now())$q$, '22023', 'vec_prueba_calendarios_lector');

SET ROLE vec_prueba_calendarios_lector;
DO $$ DECLARE n integer; total_dias integer; BEGIN
  SELECT count(*), sum(jsonb_array_length(v.dias)) INTO n, total_dias FROM vec_calendarios.versiones_vigentes_v1(2026,
    ARRAY['nacional','autonomico','local','centro'], ARRAY['es','es-an','municipio:ine:18087','centro-530'], now()) v;
  IF n<>4 OR total_dias<>16 THEN RAISE EXCEPTION 'lectura de 2026: % versiones, % días', n, total_dias; END IF;
  SELECT count(*) INTO n FROM vec_calendarios.centros_con_calendario_v1(2026, now());
  IF n<>4 THEN RAISE EXCEPTION 'centros: %', n; END IF;
  SELECT count(*) INTO n FROM vec_calendarios.versiones_vigentes_v1(2027, ARRAY['nacional'], ARRAY['es'], now());
  IF n<>0 THEN RAISE EXCEPTION '2027 no está cargado'; END IF;
  SELECT count(*) INTO n FROM vec_calendarios.versiones_vigentes_v1(2026, ARRAY['nacional'], ARRAY['es'], '2020-01-01Z');
  IF n<>0 THEN RAISE EXCEPTION 'antes de conocerse no existe'; END IF;
END $$;
RESET ROLE;

-- 3. Solo adición: ni el propietario modifica, borra ni vacía la historia.
SELECT pg_temp.espera_error($q$UPDATE vec_calendarios.dia_calendario SET denominacion='x'$q$, '55000', 'vec_calendarios_propietario');
SELECT pg_temp.espera_error($q$DELETE FROM vec_calendarios.version_calendario WHERE anio=2026$q$, '55000', 'vec_calendarios_propietario');
SELECT pg_temp.espera_error($q$TRUNCATE vec_calendarios.dia_calendario$q$, '55000', 'vec_calendarios_propietario');
SELECT pg_temp.espera_error($q$INSERT INTO vec_calendarios.dia_calendario VALUES ('calendario:nacional:es:2026:v1','2026-03-03','festivo','Añadido tardío')$q$, '55000', 'vec_calendarios_propietario');
SELECT pg_temp.espera_error($q$INSERT INTO vec_calendarios.version_calendario (id,ambito_tipo,ambito_ref,anio,numero,denominacion,procedencia_norma,procedencia_referencia,procedencia_publicada_en,sintetica)
  VALUES ('calendario:nacional:es:2026:bis','nacional','es',2026,1,'Duplicada','Prueba','prueba','2026-01-01',true)$q$, '23505', 'vec_calendarios_propietario');
SELECT pg_temp.espera_error($q$INSERT INTO vec_calendarios.version_calendario (id,ambito_tipo,ambito_ref,anio,numero,sustituye_id,denominacion,procedencia_norma,procedencia_referencia,procedencia_publicada_en,sintetica)
  VALUES ('calendario:nacional:es:2026:v3','nacional','es',2026,3,'calendario:nacional:es:2026:v1','Salto','Prueba','prueba','2026-01-01',true)$q$, '23514', 'vec_calendarios_propietario');
SELECT pg_temp.espera_error($q$INSERT INTO vec_calendarios.version_calendario (id,ambito_tipo,ambito_ref,anio,numero,sustituye_id,denominacion,procedencia_norma,procedencia_referencia,procedencia_publicada_en,sintetica,comunidad_ref)
  VALUES ('calendario:local:municipio:ine:18087:2026:mal','local','municipio:ine:18087',2026,2,'calendario:local:municipio:ine:18087:2026:v1','Otra comunidad','Prueba','prueba','2026-01-01',true,'es-md')$q$, '23514', 'vec_calendarios_propietario');
SELECT pg_temp.espera_error($q$DO $d$ BEGIN
  INSERT INTO vec_calendarios.version_calendario (id,ambito_tipo,ambito_ref,anio,numero,denominacion,procedencia_norma,procedencia_referencia,procedencia_publicada_en,sintetica)
    VALUES ('calendario:nacional:es:2030:v1','nacional','es',2030,1,'Prueba','Prueba','prueba','2026-01-01',true);
  INSERT INTO vec_calendarios.dia_calendario VALUES ('calendario:nacional:es:2030:v1','2031-01-01','festivo','Otro año');
END $d$$q$, '23514', 'vec_calendarios_propietario');

-- 4. Corrección bitemporal: una versión sucesora con sus días en la misma
--    transacción. Lo conocido antes se reconstruye con la versión anterior.
CREATE TEMP TABLE instante_prueba AS SELECT clock_timestamp() AS antes_correccion;
SELECT pg_sleep(0.01);
BEGIN;
SET LOCAL ROLE vec_calendarios_propietario;
INSERT INTO vec_calendarios.version_calendario (id,ambito_tipo,ambito_ref,anio,numero,sustituye_id,denominacion,procedencia_norma,procedencia_referencia,procedencia_publicada_en,sintetica,comunidad_ref)
  VALUES ('calendario:local:municipio:ine:18087:2026:v2','local','municipio:ine:18087',2026,2,'calendario:local:municipio:ine:18087:2026:v1',
          'Fiestas locales de Granada 2026 (sintéticas, corrección)','Corrección sintética de prueba','vec:sintetico:calendarios:2026','2026-09-25',true,'es-an');
INSERT INTO vec_calendarios.dia_calendario VALUES
  ('calendario:local:municipio:ine:18087:2026:v2','2026-03-16','festivo','Fiesta local sintética'),
  ('calendario:local:municipio:ine:18087:2026:v2','2026-06-15','festivo','Fiesta local sintética corregida');
COMMIT;
DO $$ DECLARE id_antes text; id_ahora text; BEGIN
  SELECT v.id INTO id_antes FROM instante_prueba i, vec_calendarios.versiones_vigentes_v1(2026, ARRAY['local'], ARRAY['municipio:ine:18087'], i.antes_correccion) v;
  SELECT v.id INTO id_ahora FROM vec_calendarios.versiones_vigentes_v1(2026, ARRAY['local'], ARRAY['municipio:ine:18087'], now()) v;
  IF id_antes<>'calendario:local:municipio:ine:18087:2026:v1' OR id_ahora<>'calendario:local:municipio:ine:18087:2026:v2' THEN
    RAISE EXCEPTION 'reconstrucción: antes %, ahora %', id_antes, id_ahora;
  END IF;
END $$;
SELECT pg_temp.espera_error($q$INSERT INTO vec_calendarios.version_calendario (id,ambito_tipo,ambito_ref,anio,numero,sustituye_id,denominacion,procedencia_norma,procedencia_referencia,procedencia_publicada_en,sintetica,comunidad_ref)
  VALUES ('calendario:local:municipio:ine:18087:2026:rama','local','municipio:ine:18087',2026,2,'calendario:local:municipio:ine:18087:2026:v1','Rama','Prueba','prueba','2026-01-01',true,'es-an')$q$, '23505', 'vec_calendarios_propietario');
SELECT 'OK casos Calendarios' AS resultado;
