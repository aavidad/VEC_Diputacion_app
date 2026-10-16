\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_autorizacion.seleccionar_perfil_justificante_rrhh_v1(text,text,text,text,numeric,text,text,text)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole OR
  NOT p.prosecdef OR EXISTS(SELECT 1 FROM aclexplode(p.proacl) WHERE grantee=0) OR
  NOT has_function_privilege('vec_autorizacion_fuente',p.oid,'EXECUTE') OR
  has_function_privilege('vec_documentos_bolsa_rrhh',p.oid,'EXECUTE') OR
  position('entrada#>>''{concesion,garantia_minima}''=''alto''' IN p.prosrc)=0 OR
  position('c->>''garantia_minima''=''alto''' IN p.prosrc)=0 OR
  position('''sustancial''' IN p.prosrc)<>0
 THEN RAISE EXCEPTION 'AUT101: selector público o permiso directo documental'; END IF;
END $test$;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $test$
BEGIN
 BEGIN
  PERFORM vec_autorizacion.seleccionar_perfil_justificante_rrhh_v1(
   'aut_ajena','ses_ajena','cta_ajena','cse_ajena',1,
   repeat('a',64),repeat('b',64),'correlacion_'||repeat('c',32));
  RAISE EXCEPTION 'AUT101: expuso selección sin LOGIN fuente';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
END $test$;
ROLLBACK;
