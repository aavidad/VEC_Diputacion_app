\set ON_ERROR_STOP on
-- AD194: corrección prospectiva de AD192. La repetición de registrar y el
-- cotejo devolvían auditoria_consumo_v3.secuencia, numeric(20,0), en una
-- columna declarada bigint, y PL/pgSQL abortaba con «structure of query does
-- not match function result type». Sólo se convierte ese valor a bigint en
-- las dos funciones internas. Firma, OID, propietario, ACL, SECURITY DEFINER,
-- volatilidad y proconfig se conservan. AD192 instalada no se reaplica ni
-- tiene DOWN; una segunda ejecución de AD194 se detiene por la preimagen.
BEGIN;
SET LOCAL search_path=pg_catalog;SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$BEGIN
 IF pg_catalog.current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper) THEN
  RAISE EXCEPTION 'AD194: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501';
 END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $ad194$
DECLARE
 marca constant text:='RETURN QUERY SELECT existente.auditoria_ref,existente.secuencia,existente.huella_sha256,existente.correlacion_ref,existente.registrada_en;';
 nueva constant text:='RETURN QUERY SELECT existente.auditoria_ref,existente.secuencia::bigint,existente.huella_sha256,existente.correlacion_ref,existente.registrada_en;';
 -- firma, definición previa, fuente previa, definición nueva, fuente nueva.
 casos constant text[][]:=ARRAY[
  ['vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_interna_v1(jsonb,text)',
   '827159876f00b21f14a1f925a409a00aee99b6ec791defa5d2470e7dadff38aa','90219f669dccc426c8ba66958937bb0a85d028ae515868b40cdea788aec2fdf1',
   '468bd4109d73d9a65da60f50789f394a70c4c73adac551e302e68ffb0891345b','5c2e538ef2a6a6d4392b74e06d8035fc9e67e426edaec1e93a0d97e4d05ac711'],
  ['vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_interna_v1(jsonb,text)',
   'a916927b8f400b31352b68d639ec1fd841a5e668a1543c9dbc119de659da7254','ff9c375abc732eb4ace4d2bc942b130cabe7623c9874e63802798aa450892b4c',
   '4df08307048ce453acb6109f7a0d5b0b9c7ab14c75b9e7266dbbcf2e3c23fd2e','6936123223976653d3a8cb666e034867d8261b68771b85c65e35b9313cb6f50c']];
 i int;
 anterior pg_catalog.pg_proc;
 posterior pg_catalog.pg_proc;
 definicion text;
 actual text;
BEGIN
 FOR i IN 1..pg_catalog.array_length(casos,1) LOOP
  SELECT p.* INTO anterior FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure(casos[i][1]);
  IF NOT FOUND THEN
   RAISE EXCEPTION 'AD194: PARO clave=funcion actual=ausente esperado=%',casos[i][1] USING ERRCODE='55000';
  END IF;
  definicion:=pg_catalog.pg_get_functiondef(anterior.oid);
  actual:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(definicion,'UTF8')),'hex');
  IF actual IS DISTINCT FROM casos[i][2] THEN
   RAISE EXCEPTION 'AD194: PARO clave=definicion funcion=% actual=% esperado=%',anterior.proname,actual,casos[i][2] USING ERRCODE='55000';
  END IF;
  actual:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(anterior.prosrc,'UTF8')),'hex');
  IF actual IS DISTINCT FROM casos[i][3] THEN
   RAISE EXCEPTION 'AD194: PARO clave=fuente funcion=% actual=% esperado=%',anterior.proname,actual,casos[i][3] USING ERRCODE='55000';
  END IF;
  IF anterior.proowner<>'vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole OR NOT anterior.prosecdef
  OR anterior.proacl IS DISTINCT FROM ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario']::pg_catalog.aclitem[]
  OR anterior.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC']::text[] THEN
   RAISE EXCEPTION 'AD194: PARO clave=autoridad funcion=% actual=metadata_distinta esperado=owner_ACL_proconfig_AD192',anterior.proname USING ERRCODE='55000';
  END IF;
  IF (pg_catalog.length(definicion)-pg_catalog.length(pg_catalog.replace(definicion,marca,'')))/pg_catalog.length(marca)<>1 THEN
   RAISE EXCEPTION 'AD194: PARO clave=marca funcion=% actual=no_unica esperado=una',anterior.proname USING ERRCODE='55000';
  END IF;
  definicion:=pg_catalog.replace(definicion,marca,nueva);
  actual:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(definicion,'UTF8')),'hex');
  IF actual IS DISTINCT FROM casos[i][4] THEN
   RAISE EXCEPTION 'AD194: PARO clave=delta funcion=% actual=% esperado=%',anterior.proname,actual,casos[i][4] USING ERRCODE='55000';
  END IF;
  EXECUTE definicion;
  SELECT p.* INTO STRICT posterior FROM pg_catalog.pg_proc p WHERE p.oid=anterior.oid;
  IF (pg_catalog.to_jsonb(posterior)-'prosrc') IS DISTINCT FROM (pg_catalog.to_jsonb(anterior)-'prosrc') THEN
   RAISE EXCEPTION 'AD194: PARO clave=metadata funcion=% actual=metadata_distinta esperado=metadata_original',anterior.proname USING ERRCODE='55000';
  END IF;
  actual:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(posterior.prosrc,'UTF8')),'hex');
  IF actual IS DISTINCT FROM casos[i][5] OR posterior.prosrc IS DISTINCT FROM pg_catalog.replace(anterior.prosrc,marca,nueva) THEN
   RAISE EXCEPTION 'AD194: PARO clave=fuente_final funcion=% actual=% esperado=%',anterior.proname,actual,casos[i][5] USING ERRCODE='55000';
  END IF;
 END LOOP;
END
$ad194$;
COMMIT;
