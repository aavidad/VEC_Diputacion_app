\set ON_ERROR_STOP on
-- Comprobación estructural; sólo después de instalar el borrador en el clon.
BEGIN;
SET LOCAL search_path=pg_catalog;
DO $prueba$ DECLARE comun regprocedure; propio regprocedure; BEGIN
 comun:='vec_autorizacion.acreditar_intento_consulta_meritos_v1(text,text)'::regprocedure;
 propio:='vec_meritos.registrar_intento_consulta_propia_v1(text,text,text)'::regprocedure;
 IF NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=comun AND proowner='vec_autorizacion_propietario'::regrole
  AND prosecdef AND proconfig @> ARRAY['search_path=pg_catalog','row_security=on','timezone=UTC'])
 OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=propio AND proowner='vec_meritos_propietario'::regrole
  AND prosecdef AND proconfig @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on','timezone=UTC'])
 OR NOT has_function_privilege('vec_meritos_propietario',comun,'EXECUTE')
 OR has_function_privilege('vec_meritos_registrador_intento_consulta',comun,'EXECUTE')
 OR NOT has_function_privilege('vec_meritos_registrador_intento_consulta',propio,'EXECUTE')
 OR has_function_privilege('vec_meritos_consulta_propia_interno',propio,'EXECUTE')
 OR has_function_privilege('vec_meritos_registrador_intento_consulta',
  'vec_meritos.consultar_hecho_propio_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 OR has_table_privilege('vec_meritos_registrador_intento_consulta','vec_meritos.auditoria_operacion','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
 OR has_table_privilege('vec_meritos_registrador_intento_consulta','vec_autorizacion.decision_concedida_contexto_actor_v3','SELECT')
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid IN (comun,propio) AND (a.is_grantable OR a.privilege_type<>'EXECUTE'
   OR (a.grantee<>p.proowner AND a.grantee IS DISTINCT FROM
    (CASE WHEN p.oid=comun THEN 'vec_meritos_propietario'::regrole::oid ELSE 'vec_meritos_registrador_intento_consulta'::regrole::oid END))))
 THEN RAISE EXCEPTION 'AUT29: contrato ACL fallido'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_class WHERE oid='vec_meritos.auditoria_operacion'::regclass
  AND relrowsecurity AND relforcerowsecurity)
 OR NOT EXISTS(SELECT 1 FROM pg_trigger WHERE tgrelid='vec_meritos.auditoria_operacion'::regclass
  AND tgname='historia_inmutable' AND NOT tgisinternal)
 THEN RAISE EXCEPTION 'AUT29: auditoría sin RLS o inmutabilidad'; END IF;
END $prueba$;
ROLLBACK;
