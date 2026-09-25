\set ON_ERROR_STOP on
-- Calendarios 000002: carga inicial de 2026. Las fiestas nacionales y de
-- Andalucía reproducen las normas oficiales citadas en su procedencia. Las
-- fiestas locales, los municipios asignados a los centros y los cierres
-- internos son SINTÉTICOS y así constan (sintetica=true): no proceden de
-- ningún acuerdo municipal ni de la Diputación. La escritura ordinaria del
-- catálogo será un acto gobernado de RRHH, no esta migración.
BEGIN;
SET LOCAL ROLE vec_calendarios_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_calendarios:migracion:000002:calendarios-2026:v1',0));
DO $pre$ BEGIN
 IF current_user<>'vec_calendarios_propietario' OR to_regclass('vec_calendarios.version_calendario') IS NULL
    OR to_regprocedure('vec_calendarios.versiones_vigentes_v1(integer,text[],text[],timestamptz)') IS NULL
    OR EXISTS (SELECT 1 FROM vec_calendarios.version_calendario WHERE anio=2026)
 THEN RAISE EXCEPTION 'Calendarios 000002: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

INSERT INTO vec_calendarios.version_calendario
  (id, ambito_tipo, ambito_ref, anio, numero, denominacion, procedencia_norma, procedencia_referencia, procedencia_publicada_en, sintetica, comunidad_ref, municipio_ref)
VALUES
 ('calendario:nacional:es:2026:v1', 'nacional', 'es', 2026, 1, 'Fiestas laborales nacionales 2026',
  'Resolución de 17 de octubre de 2025, de la Dirección General de Trabajo, por la que se publica la relación de fiestas laborales para el año 2026',
  'BOE-A-2025-21667 https://www.boe.es/buscar/doc.php?id=BOE-A-2025-21667', '2025-10-28', false, NULL, NULL),
 ('calendario:autonomico:es-an:2026:v1', 'autonomico', 'es-an', 2026, 1, 'Fiestas laborales de Andalucía 2026',
  'Decreto 101/2025, de 14 de mayo, por el que se determina el calendario de fiestas laborales de la Comunidad Autónoma de Andalucía para el año 2026',
  'BOJA núm. 93 https://www.juntadeandalucia.es/boja/2025/93/1.html', '2025-05-19', false, NULL, NULL),
 ('calendario:local:municipio:ine:18087:2026:v1', 'local', 'municipio:ine:18087', 2026, 1, 'Fiestas locales de Granada 2026 (sintéticas)',
  'Datos sintéticos de desarrollo; no reproducen el acuerdo municipal ni su publicación oficial',
  'vec:sintetico:calendarios:2026', '2026-09-25', true, 'es-an', NULL),
 ('calendario:local:municipio:sintetico:a:2026:v1', 'local', 'municipio:sintetico:a', 2026, 1, 'Fiestas locales del municipio sintético A 2026',
  'Datos sintéticos de desarrollo; municipio y fechas ficticios',
  'vec:sintetico:calendarios:2026', '2026-09-25', true, 'es-an', NULL),
 ('calendario:centro:centro-530:2026:v1', 'centro', 'centro-530', 2026, 1, 'Recursos Humanos',
  'Datos sintéticos de desarrollo; calendario laboral de centro no aprobado',
  'vec:sintetico:calendarios:2026', '2026-09-25', true, 'es-an', 'municipio:ine:18087'),
 ('calendario:centro:centro-520:2026:v1', 'centro', 'centro-520', 2026, 1, 'Transformación Digital',
  'Datos sintéticos de desarrollo; calendario laboral de centro no aprobado',
  'vec:sintetico:calendarios:2026', '2026-09-25', true, 'es-an', 'municipio:ine:18087'),
 ('calendario:centro:centro-102:2026:v1', 'centro', 'centro-102', 2026, 1, 'Secretaría General',
  'Datos sintéticos de desarrollo; calendario laboral de centro no aprobado',
  'vec:sintetico:calendarios:2026', '2026-09-25', true, 'es-an', 'municipio:ine:18087'),
 ('calendario:centro:centro-752:2026:v1', 'centro', 'centro-752', 2026, 1, 'Residencia «Rodríguez Penalva»',
  'Datos sintéticos de desarrollo; municipio asignado ficticio y calendario de centro no aprobado',
  'vec:sintetico:calendarios:2026', '2026-09-25', true, 'es-an', 'municipio:sintetico:a');

-- Criterio: se clasifica como nacional lo que el anexo de BOE-A-2025-21667 fija para todas las comunidades
-- en 2026 (incluido el 6 de enero, sustituible por las comunidades pero no sustituido este año); lo que una
-- comunidad sustituye o añade va en su calendario autonómico.
INSERT INTO vec_calendarios.dia_calendario (version_id, fecha, efecto, denominacion) VALUES
 ('calendario:nacional:es:2026:v1', '2026-01-01', 'festivo', 'Año Nuevo'),
 ('calendario:nacional:es:2026:v1', '2026-01-06', 'festivo', 'Epifanía del Señor'),
 ('calendario:nacional:es:2026:v1', '2026-04-03', 'festivo', 'Viernes Santo'),
 ('calendario:nacional:es:2026:v1', '2026-05-01', 'festivo', 'Fiesta del Trabajo'),
 ('calendario:nacional:es:2026:v1', '2026-08-15', 'festivo', 'Asunción de la Virgen'),
 ('calendario:nacional:es:2026:v1', '2026-10-12', 'festivo', 'Fiesta Nacional de España'),
 ('calendario:nacional:es:2026:v1', '2026-12-08', 'festivo', 'Inmaculada Concepción'),
 ('calendario:nacional:es:2026:v1', '2026-12-25', 'festivo', 'Natividad del Señor'),
 ('calendario:autonomico:es-an:2026:v1', '2026-02-28', 'festivo', 'Día de Andalucía'),
 ('calendario:autonomico:es-an:2026:v1', '2026-04-02', 'festivo', 'Jueves Santo'),
 ('calendario:autonomico:es-an:2026:v1', '2026-11-02', 'festivo', 'Lunes siguiente a Todos los Santos'),
 ('calendario:autonomico:es-an:2026:v1', '2026-12-07', 'festivo', 'Lunes siguiente al Día de la Constitución Española'),
 ('calendario:local:municipio:ine:18087:2026:v1', '2026-03-16', 'festivo', 'Fiesta local sintética'),
 ('calendario:local:municipio:ine:18087:2026:v1', '2026-06-12', 'festivo', 'Fiesta local sintética'),
 ('calendario:local:municipio:sintetico:a:2026:v1', '2026-07-24', 'festivo', 'Fiesta local sintética'),
 ('calendario:local:municipio:sintetico:a:2026:v1', '2026-09-08', 'festivo', 'Fiesta local sintética'),
 ('calendario:centro:centro-530:2026:v1', '2026-12-24', 'no_laborable', 'Cierre interno sintético'),
 ('calendario:centro:centro-530:2026:v1', '2026-12-31', 'no_laborable', 'Cierre interno sintético'),
 ('calendario:centro:centro-520:2026:v1', '2026-12-24', 'no_laborable', 'Cierre interno sintético'),
 ('calendario:centro:centro-520:2026:v1', '2026-12-31', 'no_laborable', 'Cierre interno sintético'),
 ('calendario:centro:centro-102:2026:v1', '2026-12-24', 'no_laborable', 'Cierre interno sintético'),
 ('calendario:centro:centro-102:2026:v1', '2026-12-31', 'no_laborable', 'Cierre interno sintético');
COMMIT;
