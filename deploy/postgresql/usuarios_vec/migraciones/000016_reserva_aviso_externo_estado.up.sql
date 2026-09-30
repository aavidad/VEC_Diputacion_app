\set ON_ERROR_STOP on
-- U16 corrige únicamente la referencia ambigua a estado al reservar un aviso
-- con correo propio verificado. U14/U15 ya tienen historia: no se reejecutan.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('usuarios_vec:migracion:000016',0));
DO $corregir$
DECLARE
 f pg_catalog.regprocedure:=pg_catalog.to_regprocedure('vec_usuarios_correos_externo.reservar_aviso_externo_v1(text)');
 antes pg_catalog.pg_proc%ROWTYPE;
 despues pg_catalog.pg_proc%ROWTYPE;
 definicion pg_catalog.text;
 cuerpo_nuevo pg_catalog.text;
 fragmento_anterior CONSTANT pg_catalog.text:='FROM vec_usuarios_correos_externo.correos_direccion WHERE persona_ref=persona AND activo AND estado=''verificado''';
 fragmento_nuevo CONSTANT pg_catalog.text:='FROM vec_usuarios_correos_externo.correos_direccion c WHERE c.persona_ref=persona AND c.activo AND c.estado=''verificado''';
 propietario pg_catalog.oid:=pg_catalog.to_regrole('vec_usuarios_correos_externo_propietario');
 ejecutor pg_catalog.oid:=pg_catalog.to_regrole('vec_usuarios_ejecutor_externo');
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.current_setting('transaction_isolation')<>'read committed'
    OR f IS NULL OR propietario IS NULL OR ejecutor IS NULL THEN
  RAISE EXCEPTION 'U16: migrador o preimagen no disponible' USING ERRCODE='55000';
 END IF;
 SELECT p.* INTO STRICT antes FROM pg_catalog.pg_proc p WHERE p.oid=f;
 IF antes.proowner IS DISTINCT FROM propietario
    OR antes.prosecdef IS NOT TRUE OR antes.provolatile<>'v'
    OR antes.prolang IS DISTINCT FROM (SELECT oid FROM pg_catalog.pg_language WHERE lanname='plpgsql')
    OR antes.prorettype IS DISTINCT FROM 'pg_catalog.jsonb'::pg_catalog.regtype
    OR antes.proretset
    OR antes.proacl IS NULL
    OR antes.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on']::pg_catalog.text[]
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(antes.prosrc,'UTF8')),'hex')
       <> '8b7bcb21a1c20f75ce741adc07ea734f2693a109d5394d924c67a2b10071e837'
    OR NOT pg_catalog.has_function_privilege(ejecutor,f,'EXECUTE')
    OR pg_catalog.has_function_privilege('vec_usuarios_ejecutor_interno',f,'EXECUTE')
    OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(antes.proacl) a
       WHERE a.grantee NOT IN(propietario,ejecutor)
          OR (a.grantee=ejecutor AND (a.privilege_type<>'EXECUTE' OR a.is_grantable))) THEN
  RAISE EXCEPTION 'U16: función de reserva distinta de U14/U15' USING ERRCODE='55000';
 END IF;
 IF (pg_catalog.length(antes.prosrc)-pg_catalog.length(pg_catalog.replace(antes.prosrc,fragmento_anterior,'')))
       IS DISTINCT FROM pg_catalog.length(fragmento_anterior) THEN
  RAISE EXCEPTION 'U16: sentencia de reserva ausente o repetida' USING ERRCODE='55000';
 END IF;
 cuerpo_nuevo:=pg_catalog.replace(antes.prosrc,fragmento_anterior,fragmento_nuevo);
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(cuerpo_nuevo,'UTF8')),'hex')
       <> 'eb1a20bda0ed7ded62ce80e1f39484d69faa42017484d270a8521377730c8ce8' THEN
  RAISE EXCEPTION 'U16: corrección fuera del cuerpo esperado' USING ERRCODE='55000';
 END IF;
 definicion:=pg_catalog.pg_get_functiondef(f);
 IF (pg_catalog.length(definicion)-pg_catalog.length(pg_catalog.replace(definicion,fragmento_anterior,'')))
       IS DISTINCT FROM pg_catalog.length(fragmento_anterior) THEN
  RAISE EXCEPTION 'U16: definición SQL incompatible' USING ERRCODE='55000';
 END IF;
 EXECUTE pg_catalog.replace(definicion,fragmento_anterior,fragmento_nuevo);
 SELECT p.* INTO STRICT despues FROM pg_catalog.pg_proc p WHERE p.oid=f;
 IF despues.oid IS DISTINCT FROM antes.oid OR despues.prosrc IS DISTINCT FROM cuerpo_nuevo
    OR (pg_catalog.to_jsonb(despues)-'prosrc') IS DISTINCT FROM (pg_catalog.to_jsonb(antes)-'prosrc') THEN
  RAISE EXCEPTION 'U16: firma, metadatos o historia alterados' USING ERRCODE='55000';
 END IF;
END $corregir$;
COMMIT;
