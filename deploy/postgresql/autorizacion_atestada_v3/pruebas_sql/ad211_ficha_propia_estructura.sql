\set ON_ERROR_STOP on
-- Ejecutar tras AD211, en el clon aislado. No escribe datos.
DO $prueba$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_ficha_propia_empleado_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 n oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 cuerpo text; audiencia text; accion constant text:='personal.registro_empleado.ficha_propia.consultar';
BEGIN
 IF f IS NULL OR n IS NULL THEN
  RAISE EXCEPTION 'AD211 prueba: clave=funciones esperado=presentes actual=ausentes'; END IF;
 SELECT pg_get_functiondef(n) INTO STRICT cuerpo;
 IF (length(cuerpo)-length(replace(cuerpo,accion,'')))/length(accion)<>1 THEN
  RAISE EXCEPTION 'AD211 prueba: clave=accion_nucleo esperado=1 actual=%',
   (length(cuerpo)-length(replace(cuerpo,accion,'')))/length(accion); END IF;
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT audiencia FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(audiencia,'vec_personal.registro_empleado.ficha_propia.v1')=0 THEN
  RAISE EXCEPTION 'AD211 prueba: clave=audiencia esperado=ficha_propia_v1 actual=ausente'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
   OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
   OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
     WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,'vec_personal_propietario'::regrole)
       OR a.privilege_type<>'EXECUTE' OR a.is_grantable OR a.grantor<>p.proowner))
   OR has_function_privilege('vec_personal_ejecutor',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD211 prueba: clave=ACL esperado=solo_propietarios actual=ampliada'; END IF;
END $prueba$;
