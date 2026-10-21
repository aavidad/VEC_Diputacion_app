\set ON_ERROR_STOP on
-- Focales de frontera B149 sobre copia PG18 con AD306 y B149. ROLLBACK.
BEGIN;
DO $acl$
DECLARE f oid:='vec_bolsa_llamamientos.consultar_identidad_documental_aceptacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_proc WHERE oid=f;
 IF p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
  OR NOT p.prosecdef OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
  OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
  OR EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE a.grantee=0)
  OR has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
 THEN RAISE EXCEPTION 'B149: ACL o entorno abiertos' USING ERRCODE='42501';END IF;
END $acl$;
ROLLBACK;

BEGIN ISOLATION LEVEL READ COMMITTED;
SET SESSION AUTHORIZATION vec_bolsa_llamamientos_desarrollo;
DO $g$ BEGIN
 BEGIN
  PERFORM vec_bolsa_llamamientos.consultar_identidad_documental_aceptacion_ct_v1(NULL,NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B149: READ COMMITTED aceptado';
 EXCEPTION WHEN insufficient_privilege THEN NULL;END;
END $g$;
RESET SESSION AUTHORIZATION;
ROLLBACK;

BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SET SESSION AUTHORIZATION vec_bolsa_llamamientos_desarrollo;
DO $g$ BEGIN
 BEGIN
  PERFORM vec_bolsa_llamamientos.consultar_identidad_documental_aceptacion_ct_v1(NULL,NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B149: READ ONLY aceptado';
 EXCEPTION WHEN insufficient_privilege THEN NULL;END;
END $g$;
RESET SESSION AUTHORIZATION;
ROLLBACK;

BEGIN ISOLATION LEVEL SERIALIZABLE;
SET SESSION AUTHORIZATION vec_bolsa_llamamientos_desarrollo;
DO $g$
DECLARE s text:='{"esquema":"vec.bolsa.aceptacion-nativa-ct.identidad-documento.consulta.v1","organizacion_ref":"organizacion:demo","expediente_ref":"expediente:demo","clave_ct":"clave:demo","unidad_ref":"unidad:demo","categoria_ref":"categoria:demo","necesidad_ref":"necesidad:demo","bolsa_ref":"bolsa:demo","llamamiento_ref":"llamamiento:demo","recibo_emision_ref":"recibo:emision:demo","recibo_respuesta_ref":"recibo:respuesta:demo"}';
 h text;
BEGIN
 IF encode(sha256(convert_to(s,'UTF8')),'hex') IS DISTINCT FROM '3df28a8e095cba2ca0dd1ccb8e98e35fe06fb792e69834bb6431f400e8e23fdc'
 THEN RAISE EXCEPTION 'B149: vector material cambió';END IF;
 h:=encode(sha256(convert_to('{"ambitos":{"unidad_ref":"unidad:demo"},"atributos":{"material_sha256":"'||encode(sha256(convert_to(s,'UTF8')),'hex')||'"}}','UTF8')),'hex');
 IF h IS DISTINCT FROM '3486e5f48107b6c674a9b7a26e7531ef16ae3d671ce8cf9cd7f2bef30786add7'
 THEN RAISE EXCEPTION 'B149: vector contexto cambió';END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.consultar_identidad_documental_aceptacion_ct_v1('{'||s,NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B149: selector malformado aceptado';
 EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.consultar_identidad_documental_aceptacion_ct_v1(replace(s,'"unidad_ref":"unidad:demo"','"unidad_ref": "unidad:demo"'),NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B149: bytes no canónicos aceptados';
 EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.consultar_identidad_documental_aceptacion_ct_v1(
   s,NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B149: consulta sin autorización aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL;END;
END $g$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
