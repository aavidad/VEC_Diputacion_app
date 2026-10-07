\set ON_ERROR_STOP on
-- Documentos-15. Rechaza el identificador CT todo cero al evaluar una
-- referencia tipada. La rama opaca y el CHECK de formato histórico no cambian.
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog, pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000015',0));

DO $pre$
DECLARE n integer;
BEGIN
 IF current_user<>'vec_documentos_propietario'
    OR to_regclass('vec_documentos.modulo_referencia_expediente_v1') IS NULL
    OR to_regprocedure('vec_documentos.referencia_expediente_v1(text)') IS NULL
    OR to_regprocedure('vec_documentos.referencia_expediente_modulo_v1(text,text)') IS NULL
    OR to_regprocedure('vec_documentos.referencia_expediente_formato_v1(text,text)') IS NULL
    OR to_regprocedure('vec_documentos.exigir_catalogo_expediente_v1()') IS NULL
    OR to_regclass('vec_documentos.documento') IS NULL
    OR to_regclass('vec_documentos.referencia_externa') IS NULL
    OR to_regclass('vec_documentos.reserva_original_firmable') IS NULL
 THEN RAISE EXCEPTION 'Documentos-15: preimagen D14 ausente' USING ERRCODE='55000'; END IF;

 SELECT count(*) INTO n FROM vec_documentos.modulo_referencia_expediente_v1 c;
 IF n<>1 OR NOT EXISTS (
   SELECT 1 FROM vec_documentos.modulo_referencia_expediente_v1 c
   WHERE c.codigo='ct' AND c.modulo_id='contratacion_temporal'
     AND c.version=1 AND c.patron_id='^[0-9a-f]{64}$'
 ) THEN RAISE EXCEPTION 'Documentos-15: catálogo CT divergente' USING ERRCODE='55000'; END IF;

 IF NOT EXISTS (
   SELECT 1 FROM pg_class c
   WHERE c.oid='vec_documentos.modulo_referencia_expediente_v1'::regclass
     AND c.relowner='vec_documentos_propietario'::regrole
     AND c.relrowsecurity AND c.relforcerowsecurity
 ) THEN RAISE EXCEPTION 'Documentos-15: autoridad del catálogo divergente' USING ERRCODE='55000'; END IF;

 SELECT count(*) INTO n FROM pg_trigger t
 WHERE t.tgname='validar_expediente_tipado' AND NOT t.tgisinternal
   AND t.tgfoid=to_regprocedure('vec_documentos.exigir_catalogo_expediente_v1()')
   AND t.tgtype=7
   AND t.tgrelid IN ('vec_documentos.documento'::regclass,
     'vec_documentos.referencia_externa'::regclass,
     'vec_documentos.reserva_original_firmable'::regclass);
 IF n<>3 THEN RAISE EXCEPTION 'Documentos-15: guardas INSERT D14 incompletas' USING ERRCODE='55000'; END IF;

 IF (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') FROM pg_proc p
     WHERE p.oid='vec_documentos.referencia_expediente_formato_v1(text,text)'::regprocedure)
      IS DISTINCT FROM 'fc7d3a9e7642b3711f38fb3860d78f11a541e2e2d3d508005f442d5f1e91acd1'
    OR (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') FROM pg_proc p
     WHERE p.oid='vec_documentos.exigir_catalogo_expediente_v1()'::regprocedure)
      IS DISTINCT FROM '1ca55cba6d0bfa8818b7e98119998430d9551d422234f19af0007566f116c6b6'
 THEN RAISE EXCEPTION 'Documentos-15: CHECK o trigger D14 divergente' USING ERRCODE='55000'; END IF;
END $pre$;

-- pg_get_functiondef conserva firma y opciones; CREATE OR REPLACE conserva
-- OID/ACL. Se cotejan además cuerpo, catálogo pg_proc y dependencias antes y
-- después de insertar una única condición en cada función.
DO $cambio$
DECLARE r record; funcion_oid oid; nuevo_oid oid;
        origen text; nuevo text; definicion text;
        marca text:='p ~ ''^expediente:[a-z][a-z0-9_]{1,31}:[0-9a-f]{64}$''';
        sustituta text; metadatos_antes jsonb; metadatos_despues jsonb;
        dependencias_antes jsonb; dependencias_despues jsonb; n integer;
BEGIN
 sustituta:=marca||' AND split_part(p,'':'',3) <> repeat(''0'',64)';
 FOR r IN SELECT * FROM (VALUES
   ('vec_documentos.referencia_expediente_v1(text)',
    '9e01cf67c34ea231c1e00a8b8145154b850e3be85e0e3b5a72ea1c3741f5dd24',
    '5611172df952932366ee49c107541010869e5e64e256058d4e0d9e0bc6c5fb90'),
   ('vec_documentos.referencia_expediente_modulo_v1(text,text)',
    'b711f615cae7bdbab2f8d52ba67c04bc7efb380ab3379d1b65334cc9881907bf',
    'ce27fb14b14eab2530fd79b8adf8d15d3bc42d8fa3ee54b03ab646ec8682f218')
 ) AS v(firma,huella_antes,huella_despues) LOOP
  funcion_oid:=to_regprocedure(r.firma)::oid;
  SELECT p.prosrc,to_jsonb(p)-'prosrc'
   INTO origen,metadatos_antes FROM pg_proc p WHERE p.oid=funcion_oid;
  IF origen IS NULL OR funcion_oid IS NULL
     OR encode(sha256(convert_to(origen,'UTF8')),'hex')<>r.huella_antes
     OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=funcion_oid
       AND p.proowner='vec_documentos_propietario'::regrole
       AND p.prosecdef AND p.provolatile='s'
       AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on']::text[])
     OR EXISTS (SELECT 1 FROM pg_proc p,
       LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE p.oid=funcion_oid AND a.grantee=0 AND a.privilege_type='EXECUTE')
  THEN RAISE EXCEPTION 'Documentos-15: cuerpo/owner/ACL/opciones D14 divergentes %',r.firma
    USING ERRCODE='55000'; END IF;

  SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,
     d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   INTO dependencias_antes FROM pg_depend d
   WHERE d.classid='pg_proc'::regclass AND d.objid=funcion_oid;
  n:=(length(origen)-length(replace(origen,marca,'')))/length(marca);
  definicion:=pg_get_functiondef(funcion_oid);
  IF n<>1 OR (length(definicion)-length(replace(definicion,marca,'')))/length(marca)<>1
  THEN RAISE EXCEPTION 'Documentos-15: marca tipada no única %',r.firma USING ERRCODE='55000'; END IF;

  EXECUTE replace(definicion,marca,sustituta);
  nuevo_oid:=to_regprocedure(r.firma)::oid;
  SELECT p.prosrc,to_jsonb(p)-'prosrc'
   INTO nuevo,metadatos_despues FROM pg_proc p WHERE p.oid=nuevo_oid;
  SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,
     d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   INTO dependencias_despues FROM pg_depend d
   WHERE d.classid='pg_proc'::regclass AND d.objid=nuevo_oid;
  IF nuevo_oid IS DISTINCT FROM funcion_oid
     OR nuevo IS DISTINCT FROM replace(origen,marca,sustituta)
     OR encode(sha256(convert_to(nuevo,'UTF8')),'hex') IS DISTINCT FROM r.huella_despues
     OR metadatos_despues IS DISTINCT FROM metadatos_antes
     OR dependencias_despues IS DISTINCT FROM dependencias_antes
  THEN RAISE EXCEPTION 'Documentos-15: reemplazo alteró contrato %',r.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $cambio$;

DO $post$
DECLARE cero text:='expediente:ct:'||repeat('0',64);
        uno text:='expediente:ct:'||repeat('0',63)||'1';
        otro text:='expediente:ct:'||repeat('a',64);
        generica text:='ref:'||repeat('b',64);
BEGIN
 IF vec_documentos.referencia_expediente_v1(cero)
    OR vec_documentos.referencia_expediente_modulo_v1('contratacion_temporal',cero)
    OR NOT vec_documentos.referencia_expediente_v1(uno)
    OR NOT vec_documentos.referencia_expediente_v1(otro)
    OR NOT vec_documentos.referencia_expediente_modulo_v1('contratacion_temporal',uno)
    OR NOT vec_documentos.referencia_expediente_modulo_v1('contratacion_temporal',otro)
    OR vec_documentos.referencia_expediente_modulo_v1('bolsa',uno)
    OR vec_documentos.referencia_expediente_v1(generica)
       IS DISTINCT FROM vec_documentos.referencia_opaca_v1(generica)
    OR vec_documentos.referencia_expediente_modulo_v1('dietas',generica)
       IS DISTINCT FROM vec_documentos.referencia_opaca_v1(generica)
    OR NOT vec_documentos.referencia_expediente_formato_v1('contratacion_temporal',cero)
 THEN RAISE EXCEPTION 'Documentos-15: predicados postimagen incoherentes' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
