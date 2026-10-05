-- Sólo comprobación estructural posterior a AD191; no publica datos.
BEGIN;
DO $verificar_ad191$
DECLARE p pg_catalog.pg_proc; actual text;
BEGIN
 SELECT x.* INTO STRICT p FROM pg_catalog.pg_proc x WHERE x.oid='vec_autorizacion_atestada_v3.efecto_gobierno_usuarios_admin_v1(text,text,text)'::pg_catalog.regprocedure;
 actual:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex');
 IF actual <> '4dc9a79e919633b6c9f355e368dfef00c33701cf6ce1dc5ed39def98b4f7ad2e' THEN
  RAISE EXCEPTION 'PARO clave=AD191.vector.fuente actual=% esperado=4dc9a79e919633b6c9f355e368dfef00c33701cf6ce1dc5ed39def98b4f7ad2e',actual;
 END IF;
 IF p.proacl IS DISTINCT FROM ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario']::pg_catalog.aclitem[] OR NOT p.prosecdef
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[] THEN
  RAISE EXCEPTION 'PARO clave=AD191.vector.autoridad actual=metadata_distinta esperado=owner_ACL_search_path_originales';
 END IF;
END
$verificar_ad191$;
ROLLBACK;
