\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
DO $test$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 definicion text; fuente text; def_sha text; src_sha text;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD225 prueba: núcleo ausente'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc INTO STRICT definicion,fuente FROM pg_proc p WHERE p.oid=f;
 def_sha:=encode(sha256(convert_to(definicion,'UTF8')),'hex');
 src_sha:=encode(sha256(convert_to(fuente,'UTF8')),'hex');
 IF NOT ((def_sha='7727d537e05e3320f013ec216e83da32f64ddacf5fd13b47b39f883e6f33abcb'
          AND src_sha='d43a29e1827b2e809b00475d4f34dbe0f57a195ffcc7acb17d72f69de573fe15')
       OR (def_sha='03f151286ed21039cfe2f96840f474062a8aecb61c546601ef71cf20cc716212'
          AND src_sha='fc80e4851d7a63a147cce6a40ca924d24d0b5e671686134be90e98e0f53c906c')) THEN
  RAISE EXCEPTION 'AD225 prueba: postimagen divergente def=% src=%',def_sha,src_sha;
 END IF;
 IF strpos(definicion,'reanudacion_seleccion')=0
    OR strpos(definicion,'reanudacion_solicitud_llamamiento')=0
    OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL
        aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
        AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
        AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']) THEN
  RAISE EXCEPTION 'AD225 prueba: autoridad, marcas o ACL divergentes';
 END IF;
END
$test$;
ROLLBACK;
