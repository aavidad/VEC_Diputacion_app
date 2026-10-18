\set ON_ERROR_STOP on
-- AD294 / F1-12: el intento nominal no es un consumo ni una lectura confirmada.
-- Reconstruye solo el productor externo instalado por AD246; conserva su firma y ACL.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000294',0));
DO $migracion$
DECLARE
 f regprocedure:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_lectura_preferencias_bolsas_v1(bytea,bytea,jsonb)');
 p pg_proc%ROWTYPE;
 anterior text;
 nueva text;
 marca_columnas constant text:='correlacion_ref,vinculo_sha256,transaccion_origen)';
 marca_valores constant text:='encode(sha256(p_vinculo_canonico),''hex''),pg_current_xact_id());';
 post pg_proc%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 OR current_user<>'vec_autorizacion_atestada_v3_propietario' OR f IS NULL
 THEN RAISE EXCEPTION 'AD294: preimagen incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 anterior:=pg_get_functiondef(f);
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
 OR p.prosecdef IS NOT TRUE OR p.provolatile<>'v' OR p.proparallel<>'u'
 OR p.proacl::text IS DISTINCT FROM
  '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_bolsa_llamamientos_propietario=X/vec_autorizacion_atestada_v3_propietario}'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s','statement_timeout=10s']
 OR encode(sha256(convert_to(anterior,'UTF8')),'hex') IS DISTINCT FROM
  'd72e98e0858d29535de28eb3ca06dcf166ac9a96c9c7827351499c2a73e49b75'
 OR (length(anterior)-length(replace(anterior,marca_columnas,'')))/length(marca_columnas)<>1
 OR (length(anterior)-length(replace(anterior,marca_valores,'')))/length(marca_valores)<>1
 THEN RAISE EXCEPTION 'AD294: cuerpo o ACL previos divergentes' USING ERRCODE='55000'; END IF;
 nueva:=replace(replace(anterior,marca_columnas,'correlacion_ref,vinculo_sha256)'),
  marca_valores,'encode(sha256(p_vinculo_canonico),''hex''));');
 IF encode(sha256(convert_to(nueva,'UTF8')),'hex') IS DISTINCT FROM
  '819b55746d49c058c718bd4e69bfa41477391d72ca7ce16d6854b57532de7d27'
 THEN RAISE EXCEPTION 'AD294: sustitución inesperada' USING ERRCODE='55000'; END IF;
 EXECUTE nueva;
 SELECT * INTO STRICT post FROM pg_proc WHERE oid=f;
 IF post.proowner IS DISTINCT FROM p.proowner OR post.prosecdef IS DISTINCT FROM p.prosecdef
 OR post.provolatile IS DISTINCT FROM p.provolatile OR post.proparallel IS DISTINCT FROM p.proparallel
 OR post.proacl IS DISTINCT FROM p.proacl OR post.proconfig IS DISTINCT FROM p.proconfig
 OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM
  '819b55746d49c058c718bd4e69bfa41477391d72ca7ce16d6854b57532de7d27'
 THEN RAISE EXCEPTION 'AD294: cuerpo o ACL posteriores divergentes' USING ERRCODE='55000'; END IF;
END $migracion$;
COMMIT;
