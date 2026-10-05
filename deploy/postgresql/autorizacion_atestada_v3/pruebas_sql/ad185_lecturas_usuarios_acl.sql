\set ON_ERROR_STOP on
-- Ejecutar únicamente en clon después de AUT42, AD184 y AUT43, con ROLLBACK.
BEGIN;
SET LOCAL search_path=pg_catalog;
DO $test$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_usuarios_admin_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 n oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 s text;
BEGIN
 IF f IS NULL OR n IS NULL THEN RAISE EXCEPTION 'AD185: fachada_o_nucleo_ausente';END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef AND p.provolatile='v' AND p.proconfig @> ARRAY['search_path=pg_catalog'])
 OR has_function_privilege('vec_admin_usuarios_lector',f,'EXECUTE')
 OR NOT has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,'vec_autorizacion_propietario'::regrole) OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'AD185: ACL_de_fachada_incorrecta';END IF;
 SELECT p.prosrc INTO STRICT s FROM pg_proc p WHERE p.oid=n;
 IF s NOT LIKE '%usuarios_admin_listar%' OR s NOT LIKE '%usuarios_admin_consultar%'
 OR s NOT LIKE '%persona_denominacion_publicar%' OR s NOT LIKE '%persona_denominacion_leer%'
 OR s NOT LIKE '%resolver_origen_consumo_v1%'
 THEN RAISE EXCEPTION 'AD185: extension_o_origen_perdidos';END IF;
 IF (SELECT count(*) FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.convalidated
  AND pg_get_constraintdef(c.oid,false) LIKE '%vec.persona.denominacion.publicar.v1%'
  AND pg_get_constraintdef(c.oid,false) LIKE '%vec.persona.denominacion.leer.v1%'
  AND pg_get_constraintdef(c.oid,false) LIKE '%vec.admin.usuarios.listar.v1%'
  AND pg_get_constraintdef(c.oid,false) LIKE '%vec.admin.usuarios.consultar.v1%')<>1
 THEN RAISE EXCEPTION 'AD185: audiencias_ausentes';END IF;
END $test$;
ROLLBACK;
