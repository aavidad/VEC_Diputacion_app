\set ON_ERROR_STOP on
-- Focal del predicado temporal REAL de Personal30. No concede acceso ni produce DTO.
-- Dirección la ejecuta en el clon después del UP30 autorizado; nunca reaplica27.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='15s';
DO $prueba$
DECLARE
 f oid:=to_regprocedure('vec_personal.consultar_relacion_para_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 fuente text; conversion text; guardia text; comparacion text; caso record; codigo text; orden text;
BEGIN
 SELECT prosrc INTO STRICT fuente FROM pg_proc WHERE oid=f;
 IF encode(sha256(convert_to(fuente,'UTF8')),'hex')<>'dd0b06eb3ecb2a8317054739ad01f6c363c725f477d7f98d345f01d83487a78f' THEN
  RAISE EXCEPTION 'Personal30 prueba: cuerpo no revisado' USING ERRCODE='55000';
 END IF;
 -- Extracción del cuerpo instalado: no ejecutar una copia que pueda divergir.
 conversion:='cap_dec_hasta:='||split_part(split_part(fuente,'cap_dec_hasta:=',2),';',1)||';';
 guardia:='dec_hasta IS NULL'||split_part(split_part(fuente,' OR dec_hasta IS NULL',2),E'\n',1);
 comparacion:='cap_dec_hasta IS DISTINCT FROM'||split_part(split_part(fuente,' OR cap_dec_hasta IS DISTINCT FROM',2),E'\n',1);
 FOR caso IN SELECT * FROM (VALUES
  ('fraccion_equivalente','2026-10-03T15:03:17.71397Z','2026-10-03T15:03:17.713970Z','ok'),
  ('offset_equivalente','2026-10-03T16:03:17.713970+01:00','2026-10-03T15:03:17.713970Z','ok'),
  ('microsegundo_distinto','2026-10-03T15:03:17.713971Z','2026-10-03T15:03:17.713970Z','42501'),
  ('nulo',NULL,'2026-10-03T15:03:17.713970Z','42501'),
  ('ausente',NULL,'2026-10-03T15:03:17.713970Z','42501'),
  ('infinito','infinity','2026-10-03T15:03:17.713970Z','42501'),
  ('infinito_negativo','-infinity','2026-10-03T15:03:17.713970Z','42501'),
  ('malformado','no-es-fecha','2026-10-03T15:03:17.713970Z','22023')
 ) AS t(nombre,capacidad,decision,esperado) LOOP
  orden:=format($q$DO $unidad$
  DECLARE c jsonb:=%L::jsonb; dec_hasta timestamptz:=%L::timestamptz; cap_dec_hasta timestamptz;
  BEGIN
   BEGIN %s
   EXCEPTION WHEN others THEN RAISE EXCEPTION 'Personal30 unidad: conversión inválida' USING ERRCODE='22023'; END;
   IF %s OR %s THEN RAISE EXCEPTION 'Personal30 unidad: instante divergente' USING ERRCODE='42501'; END IF;
  END $unidad$;$q$,
   CASE WHEN caso.nombre='ausente' THEN '{}'::jsonb ELSE jsonb_build_object('decision_valida_hasta',caso.capacidad) END,
   caso.decision,conversion,guardia,comparacion);
  codigo:='ok';
  BEGIN EXECUTE orden;
  EXCEPTION WHEN others THEN GET STACKED DIAGNOSTICS codigo=RETURNED_SQLSTATE;
  END;
  IF codigo IS DISTINCT FROM caso.esperado THEN
   RAISE EXCEPTION 'Personal30 prueba: caso=%, actual=%, esperado=%',caso.nombre,codigo,caso.esperado USING ERRCODE='55000';
  END IF;
 END LOOP;
END $prueba$;
ROLLBACK;
