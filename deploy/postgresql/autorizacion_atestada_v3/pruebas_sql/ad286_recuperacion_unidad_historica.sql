\set ON_ERROR_STOP on
-- Comprobación posterior a AD286, antes de CT228 y sin modificar datos.
BEGIN READ ONLY;
DO $prueba$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.comprobar_consumo_recuperacion_firmas_ct_v2(bytea,jsonb)');
 cuerpo text;
BEGIN
 SELECT p.prosrc INTO STRICT cuerpo FROM pg_proc p WHERE p.oid=f
  AND p.proowner=to_regrole('vec_autorizacion_atestada_v3_propietario')
  AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
  AND p.proacl=ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario',
   'vec_autorizacion_propietario=X/vec_autorizacion_atestada_v3_propietario']::aclitem[];
 IF encode(sha256(convert_to(cuerpo,'UTF8')),'hex')<>'1e45359539948ccd4e63f28abc9b13dc04f00b2affdd072f0422bb75c00656ea'
  OR strpos(cuerpo,'huella_recurso_consulta_firmas_r5_ct_v1(')=0
  OR strpos(cuerpo,$m$s->>'PasoOrden'='3' AND (s->>'Via' IS DISTINCT FROM 'certificado_vec'$m$)=0
 THEN RAISE EXCEPTION 'AD286: comprobador o límites de vía divergentes' USING ERRCODE='55000'; END IF;
END $prueba$;
ROLLBACK;
