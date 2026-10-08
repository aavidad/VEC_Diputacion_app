-- B79/AD203: comprobaciones negativas en un clon desechable (nunca en la principal).
-- Uso: psql -U postgres -v ON_ERROR_STOP=1 -f b79_carga_convoca_negativos_clon.sql
-- Crea dos LOGIN efímeros, comprueba que cada intento falla con el código
-- esperado y los borra. No deja constituciones, consumos ni roles.
\set ON_ERROR_STOP on
CREATE ROLE b79_prueba_ejecutor LOGIN INHERIT IN ROLE vec_bolsa_llamamientos_ejecutor;
CREATE ROLE b79_prueba_ajeno LOGIN;
DO $$ BEGIN
 IF NOT has_function_privilege('b79_prueba_ejecutor','vec_bolsa_llamamientos.constituir_bolsa_carga_convoca_v1(text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 OR has_function_privilege('b79_prueba_ejecutor','vec_autorizacion_atestada_v3.consumir_carga_convoca_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 OR has_function_privilege('b79_prueba_ajeno','vec_bolsa_llamamientos.constituir_bolsa_carga_convoca_v1(text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'B79 prueba: ACL inesperada'; END IF;
END $$;
SET SESSION AUTHORIZATION b79_prueba_ejecutor;
BEGIN ISOLATION LEVEL SERIALIZABLE;
DO $$
DECLARE h text:=repeat('c',64); cat text:='categoria:rpt:x'; estado text;
 acta text:='acta:importacion-convoca:'||encode(sha256(convert_to(repeat('c',64)||chr(31)||'categoria:rpt:x','UTF8')),'hex');
 bolsa bytea:=convert_to('{"bolsa_ref":"bolsa:x","categoria_ref":"categoria:rpt:x","huella_listado_sha256":"'||repeat('c',64)||'"}','UTF8');
 decision bytea:=convert_to('{"principal_id":"per_x","recurso_ref":"'||acta||'"}','UTF8');
BEGIN
 -- 1) Decisión ilegible.
 BEGIN
  PERFORM vec_bolsa_llamamientos.constituir_bolsa_carga_convoca_v1(acta,'per_x',cat,'bolsa:x',1,bolsa,'2026-10-05','ins',1,'\x7b7d',now(),now(),'[]',now(),'\x00','\x00','\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B79 prueba: decisión ilegible aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 -- 2) Decisión de otro actor.
 BEGIN
  PERFORM vec_bolsa_llamamientos.constituir_bolsa_carga_convoca_v1(acta,'per_x',cat,'bolsa:x',1,bolsa,'2026-10-05','ins',1,'\x7b7d',now(),now(),'[]',now(),'\x00',convert_to('{"principal_id":"per_y","recurso_ref":"'||acta||'"}','UTF8'),'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B79 prueba: decisión de otro actor aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 -- 3) Decisión de otra acta.
 BEGIN
  PERFORM vec_bolsa_llamamientos.constituir_bolsa_carga_convoca_v1(acta,'per_x',cat,'bolsa:x',1,bolsa,'2026-10-05','ins',1,'\x7b7d',now(),now(),'[]',now(),'\x00',convert_to('{"principal_id":"per_x","recurso_ref":"acta:importacion-convoca:'||repeat('b',64)||'"}','UTF8'),'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B79 prueba: decisión de otra acta aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 -- 4) Misma acta y decisión, pero otra categoría (el acta no corresponde).
 BEGIN
  PERFORM vec_bolsa_llamamientos.constituir_bolsa_carga_convoca_v1(acta,'per_x','categoria:rpt:y','bolsa:x',1,
   convert_to('{"bolsa_ref":"bolsa:x","categoria_ref":"categoria:rpt:y","huella_listado_sha256":"'||h||'"}','UTF8'),
   '2026-10-05','ins',1,'\x7b7d',now(),now(),'[]',now(),'\x00',decision,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B79 prueba: categoría ajena al acta aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 -- 5) Bolsa canónica de otra bolsa.
 BEGIN
  PERFORM vec_bolsa_llamamientos.constituir_bolsa_carga_convoca_v1(acta,'per_x',cat,'bolsa:otra',1,bolsa,'2026-10-05','ins',1,'\x7b7d',now(),now(),'[]',now(),'\x00',decision,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B79 prueba: bolsa canónica ajena aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 -- 6) Todo ligado, pero sin capacidad firmada: la para AD203.
 BEGIN
  PERFORM vec_bolsa_llamamientos.constituir_bolsa_carga_convoca_v1(acta,'per_x',cat,'bolsa:x',1,bolsa,'2026-10-05','ins',1,'\x7b7d',now(),now(),'[]',now(),'\x00',decision,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B79 prueba: capacidad sin firma aceptada';
 EXCEPTION WHEN insufficient_privilege THEN GET STACKED DIAGNOSTICS estado=MESSAGE_TEXT;
  IF strpos(estado,'AD203')=0 THEN RAISE EXCEPTION 'B79 prueba: denegación fuera de AD203: %',estado; END IF;
 END;
END $$;
ROLLBACK;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION b79_prueba_ajeno;
DO $$ BEGIN
 BEGIN
  PERFORM vec_bolsa_llamamientos.constituir_bolsa_carga_convoca_v1('acta:importacion-convoca:'||repeat('a',64),'per_x','c','b',1,'\x7b7d',now(),'i',1,'\x7b7d',now(),now(),'[]',now(),'\x00','\x00','\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B79 prueba: LOGIN ajeno aceptado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $$;
RESET SESSION AUTHORIZATION;
DROP ROLE b79_prueba_ejecutor;
DROP ROLE b79_prueba_ajeno;
SELECT 'B79 negativos OK' AS resultado;
