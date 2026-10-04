\set ON_ERROR_STOP on
-- Prueba estructural; la prueba nominal requiere CT174, CA25, Personal29 y V3 reales.
-- Los vectores Go se generan con canon_aut35_go y se ejecutan en el clon.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL timezone='UTC';
DO $test$
DECLARE f regprocedure; a record;
BEGIN
 f:=to_regprocedure('vec_autorizacion.construir_contexto_nominal_firmante_ct_v1(jsonb,jsonb,jsonb)');
 IF f IS NULL OR to_regprocedure('vec_autorizacion.canon_json_competencia_firmante_ct_v1(jsonb,text)') IS NULL
  OR to_regprocedure('vec_autorizacion.canon_texto_json_go_ct_v1(text)') IS NULL
  OR NOT has_function_privilege('vec_contratacion_temporal_propietario',f,'EXECUTE')
 THEN
  RAISE EXCEPTION 'AUT35 ABI/ACL ausentes' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
    aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f
    AND x.grantee NOT IN ('vec_autorizacion_propietario'::regrole,
      'vec_contratacion_temporal_propietario'::regrole)) THEN
  RAISE EXCEPTION 'AUT35 permiso exterior inesperado' USING ERRCODE='55000'; END IF;
 BEGIN
  PERFORM vec_autorizacion.construir_contexto_nominal_firmante_ct_v1(
    '{}'::jsonb,'{}'::jsonb,'{}'::jsonb);
  RAISE EXCEPTION 'AUT35 aceptó descriptor vacío' USING ERRCODE='55000';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
 -- No se ha creado evidencia por el constructor ni por las funciones de canon.
 IF EXISTS(SELECT 1 FROM vec_autorizacion.evidencia_competencia_firmante_ct_v1
  WHERE efecto_ref='aut35_prueba_estructural') THEN
  RAISE EXCEPTION 'AUT35 escribió evidencia' USING ERRCODE='55000'; END IF;
END $test$;
ROLLBACK;
\echo AUT35_ESTRUCTURA_OK
