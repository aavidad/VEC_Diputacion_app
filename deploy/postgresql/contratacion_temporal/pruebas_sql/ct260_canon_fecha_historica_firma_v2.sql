\set ON_ERROR_STOP on
-- CT260, sobre la migración instalada en una copia local. Extrae y ejecuta
-- la guarda real de registrar_firma_verificada_v2. No crea firmas ni autoridad.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='15s';
CREATE TEMP TABLE ct260_guardia_instalada (
 clave boolean PRIMARY KEY DEFAULT true CHECK (clave),
 expresion text NOT NULL
) ON COMMIT DROP;
DO $extraer$
DECLARE
 f oid; definicion text; guarda text; expresion text;
BEGIN
 f:=to_regprocedure('vec_contratacion_temporal.registrar_firma_verificada_v2(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')
      IS DISTINCT FROM 'e25c5f2396341320420d8dced263eaff3d7c50114db1b784751368a242fdbac0'
 THEN RAISE EXCEPTION 'CT260: función instalada distinta' USING ERRCODE='55000'; END IF;
 definicion:=pg_get_functiondef(f);
 IF (length(definicion)-length(replace(definicion,'-- CT260_GUARD_FECHA_BEGIN','')))
      /length('-- CT260_GUARD_FECHA_BEGIN')<>1
    OR (length(definicion)-length(replace(definicion,'-- CT260_GUARD_FECHA_END','')))
      /length('-- CT260_GUARD_FECHA_END')<>1
 THEN RAISE EXCEPTION 'CT260: guarda instalada ausente o duplicada' USING ERRCODE='55000'; END IF;
 guarda:=btrim(regexp_replace(split_part(split_part(definicion,'-- CT260_GUARD_FECHA_BEGIN',2),
   '-- CT260_GUARD_FECHA_END',1),'[[:space:]]+',' ','g'));
 IF regexp_replace(guarda,'[[:space:]]+',' ','g') IS DISTINCT FROM
   'OR (canon->>''fecha_historica'')::timestamptz IS DISTINCT FROM (descriptor->>''fecha_historica'')::timestamptz'
 THEN RAISE EXCEPTION 'CT260: guarda instalada inesperada' USING ERRCODE='55000'; END IF;
 expresion:=replace(replace(substr(guarda,4),'canon->>','$1->>'),'descriptor->>','$2->>');
 INSERT INTO pg_temp.ct260_guardia_instalada(expresion) VALUES (expresion);
END $extraer$;
DO $evaluar$
DECLARE expresion text; observada boolean; caso record;
BEGIN
 SELECT g.expresion INTO STRICT expresion FROM pg_temp.ct260_guardia_instalada g;
 FOR caso IN SELECT * FROM (VALUES
  ('cero final', '{"fecha_historica":"2026-10-21T07:22:00.12345Z"}'::jsonb,
    '{"fecha_historica":"2026-10-21T07:22:00.123450Z"}'::jsonb,false),
  ('varios ceros', '{"fecha_historica":"2026-10-21T07:22:00.12Z"}'::jsonb,
    '{"fecha_historica":"2026-10-21T07:22:00.120000Z"}'::jsonb,false),
  ('microsegundos cero', '{"fecha_historica":"2026-10-21T07:22:00Z"}'::jsonb,
    '{"fecha_historica":"2026-10-21T07:22:00.000000Z"}'::jsonb,false),
  ('seis cifras', '{"fecha_historica":"2026-10-21T07:22:00.123456Z"}'::jsonb,
    '{"fecha_historica":"2026-10-21T07:22:00.123456Z"}'::jsonb,false),
  ('microsegundo distinto', '{"fecha_historica":"2026-10-21T07:22:00.123456Z"}'::jsonb,
    '{"fecha_historica":"2026-10-21T07:22:00.123455Z"}'::jsonb,true),
  ('canon nulo', '{"fecha_historica":null}'::jsonb,
    '{"fecha_historica":"2026-10-21T07:22:00Z"}'::jsonb,true),
  ('fecha ausente', '{}'::jsonb,
    '{"fecha_historica":"2026-10-21T07:22:00Z"}'::jsonb,true)
 ) AS x(nombre,canon,descriptor,esperada) LOOP
  EXECUTE 'SELECT '||expresion INTO observada USING caso.canon,caso.descriptor;
  IF observada IS DISTINCT FROM caso.esperada THEN
   RAISE EXCEPTION 'CT260: cotejo incorrecto para %',caso.nombre USING ERRCODE='55000'; END IF;
 END LOOP;
END $evaluar$;
ROLLBACK;
