\set ON_ERROR_STOP on
-- AD191: corrección prospectiva de una variable que ocultaba un alias SQL.
-- AD188 instalada conserva su archivo y su historia; no hay DOWN ni reaplicación.
BEGIN;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $ad191$
DECLARE
 firma constant text := 'vec_autorizacion_atestada_v3.efecto_gobierno_usuarios_admin_v1(text,text,text)';
 anterior pg_catalog.pg_proc;
 posterior pg_catalog.pg_proc;
 definicion text;
 actual text;
 fuente_esperada text;
BEGIN
 SELECT p.* INTO STRICT anterior FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure(firma);
 definicion:=pg_catalog.pg_get_functiondef(anterior.oid);
 actual:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(definicion,'UTF8')),'hex');
 IF actual <> 'f2d7076ec1aaee4c4070f3caabbfc03b4ea725f55553ff525f0935e49421dc39' THEN
  RAISE EXCEPTION 'PARO clave=AD191.definicion actual=% esperado=f2d7076ec1aaee4c4070f3caabbfc03b4ea725f55553ff525f0935e49421dc39',actual USING ERRCODE='55000';
 END IF;
 actual:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(anterior.prosrc,'UTF8')),'hex');
 IF actual <> '29ac3718c0b255641ccf68994ff07147cdd3e1c006a5969d9c73d664faa57e7e' THEN
  RAISE EXCEPTION 'PARO clave=AD191.fuente actual=% esperado=29ac3718c0b255641ccf68994ff07147cdd3e1c006a5969d9c73d664faa57e7e',actual USING ERRCODE='55000';
 END IF;
 IF anterior.proowner <> 'vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole
    OR anterior.proacl IS DISTINCT FROM ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario']::pg_catalog.aclitem[]
    OR anterior.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[] THEN
  RAISE EXCEPTION 'PARO clave=AD191.autoridad actual=metadata_distinta esperado=owner_ACL_search_path_originales' USING ERRCODE='55000';
 END IF;
 fuente_esperada:=anterior.prosrc;
 IF (pg_catalog.length(definicion)-pg_catalog.length(pg_catalog.replace(definicion,'a record;','')))/pg_catalog.length('a record;') <> 1 THEN
  RAISE EXCEPTION 'PARO clave=AD191.marca actual=no_unica esperado=una' USING ERRCODE='55000';
 END IF;
 definicion:=pg_catalog.replace(definicion,'a record;','v_confirmacion record;');
 fuente_esperada:=pg_catalog.replace(fuente_esperada,'a record;','v_confirmacion record;');
 IF (pg_catalog.length(definicion)-pg_catalog.length(pg_catalog.replace(definicion,'SELECT * INTO a FROM vec_autorizacion_atestada_v3.registrar_gobierno_usuarios_admin_v1','')))/pg_catalog.length('SELECT * INTO a FROM vec_autorizacion_atestada_v3.registrar_gobierno_usuarios_admin_v1') <> 1 THEN
  RAISE EXCEPTION 'PARO clave=AD191.marca actual=no_unica esperado=una' USING ERRCODE='55000';
 END IF;
 definicion:=pg_catalog.replace(definicion,'SELECT * INTO a FROM vec_autorizacion_atestada_v3.registrar_gobierno_usuarios_admin_v1','SELECT * INTO v_confirmacion FROM vec_autorizacion_atestada_v3.registrar_gobierno_usuarios_admin_v1');
 fuente_esperada:=pg_catalog.replace(fuente_esperada,'SELECT * INTO a FROM vec_autorizacion_atestada_v3.registrar_gobierno_usuarios_admin_v1','SELECT * INTO v_confirmacion FROM vec_autorizacion_atestada_v3.registrar_gobierno_usuarios_admin_v1');
 IF (pg_catalog.length(definicion)-pg_catalog.length(pg_catalog.replace(definicion,'to_jsonb(a)','')))/pg_catalog.length('to_jsonb(a)') <> 1 THEN
  RAISE EXCEPTION 'PARO clave=AD191.marca actual=no_unica esperado=una' USING ERRCODE='55000';
 END IF;
 definicion:=pg_catalog.replace(definicion,'to_jsonb(a)','to_jsonb(v_confirmacion)');
 fuente_esperada:=pg_catalog.replace(fuente_esperada,'to_jsonb(a)','to_jsonb(v_confirmacion)');
 actual:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(definicion,'UTF8')),'hex');
 IF actual <> 'aadaaa7f9a1029564e4c02587394a175ab4a5dd45c67526d50ea034f8b6ef973' THEN
  RAISE EXCEPTION 'PARO clave=AD191.delta actual=% esperado=aadaaa7f9a1029564e4c02587394a175ab4a5dd45c67526d50ea034f8b6ef973',actual USING ERRCODE='55000';
 END IF;
 EXECUTE definicion;
 SELECT p.* INTO STRICT posterior FROM pg_catalog.pg_proc p WHERE p.oid=anterior.oid;
 IF (pg_catalog.to_jsonb(posterior)-'prosrc') IS DISTINCT FROM (pg_catalog.to_jsonb(anterior)-'prosrc') THEN
  RAISE EXCEPTION 'PARO clave=AD191.metadata actual=metadata_distinta esperado=metadata_original' USING ERRCODE='55000';
 END IF;
 actual:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(posterior.prosrc,'UTF8')),'hex');
 IF actual <> '4dc9a79e919633b6c9f355e368dfef00c33701cf6ce1dc5ed39def98b4f7ad2e' OR posterior.prosrc IS DISTINCT FROM fuente_esperada THEN
  RAISE EXCEPTION 'PARO clave=AD191.fuente_final actual=% esperado=4dc9a79e919633b6c9f355e368dfef00c33701cf6ce1dc5ed39def98b4f7ad2e',actual USING ERRCODE='55000';
 END IF;
END
$ad191$;
COMMIT;
