\set ON_ERROR_STOP on
-- AUT41 tras L/AD178 y AUT41 en clon. Sin concesiones ni evidencia sintética.
-- SERIALIZABLE: así el rechazo sale de la validación del selector, no del aislamiento.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='15s';
DO $prueba$
DECLARE f regprocedure:='vec_autorizacion.recuperar_canon_historico_firmante_ct_v1(jsonb,bytea,jsonb)'::regprocedure;
 a record; rechazo boolean:=false;
 owner_id oid:='vec_autorizacion_propietario'::regrole;
 ct_id oid:='vec_contratacion_temporal_propietario'::regrole;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f AND p.proowner=owner_id
   AND p.prosecdef AND p.provolatile='v' AND p.proconfig @> ARRAY['search_path=pg_catalog','row_security=on'])
  OR NOT pg_catalog.has_function_privilege(ct_id,f,'EXECUTE') THEN
  RAISE EXCEPTION 'AUT41: función privada no compuesta' USING ERRCODE='55000'; END IF;
 FOR a IN SELECT x.grantee,x.grantor,x.privilege_type,x.is_grantable
  FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
   pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x
  WHERE p.oid=f LOOP
  IF a.grantee NOT IN(owner_id,ct_id) OR a.privilege_type<>'EXECUTE'
     OR (a.grantee=ct_id AND (a.grantor<>owner_id OR a.is_grantable)) THEN
   RAISE EXCEPTION 'AUT41: ACL excesiva' USING ERRCODE='55000'; END IF;
 END LOOP;
 IF (SELECT count(*) FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
   pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x
   WHERE p.oid=f)<>2 THEN
  RAISE EXCEPTION 'AUT41: ACL incompleta' USING ERRCODE='55000'; END IF;
 -- Rechazo anterior a cualquier acceso AD: no constituye una lectura positiva.
 BEGIN
  PERFORM vec_autorizacion.recuperar_canon_historico_firmante_ct_v1(
   '{}'::jsonb,pg_catalog.convert_to('{}','UTF8'),'{}'::jsonb);
 EXCEPTION WHEN SQLSTATE '42501' THEN rechazo:=true; END;
 IF NOT rechazo THEN
  RAISE EXCEPTION 'AUT41: selector vacío aceptado' USING ERRCODE='55000'; END IF;
END $prueba$;
ROLLBACK;
