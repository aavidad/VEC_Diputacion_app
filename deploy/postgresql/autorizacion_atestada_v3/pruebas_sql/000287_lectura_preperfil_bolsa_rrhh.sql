\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE p pg_proc%ROWTYPE; c record; anterior text;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_autorizacion_atestada_v3.registrar_preperfil_bolsa_rrhh_v1(jsonb)'::regprocedure;
 SELECT pg_get_constraintdef(oid,false) AS definicion INTO STRICT c FROM pg_constraint WHERE
  conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND
  conname='auditoria_tipo_disjunto_v4' AND contype='c' AND convalidated;
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole OR
  NOT p.prosecdef OR EXISTS(SELECT 1 FROM aclexplode(p.proacl) WHERE grantee=0) OR
  NOT has_function_privilege('vec_autorizacion_propietario',p.oid,'EXECUTE') OR
  has_function_privilege('vec_autorizacion_fuente',p.oid,'EXECUTE') OR
  position('preperfil_bolsa_rrhh' IN c.definicion)=0
 THEN RAISE EXCEPTION 'AD287: auditoría preperfil abierta o sin CHECK'; END IF;
END $test$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $test$
BEGIN
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.registrar_preperfil_bolsa_rrhh_v1('{}'::jsonb);
  RAISE EXCEPTION 'AD287: aceptó evento sin fuente acreditada';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
END $test$;
ROLLBACK;
