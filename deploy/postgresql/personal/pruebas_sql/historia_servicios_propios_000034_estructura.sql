\set ON_ERROR_STOP on
-- BORRADOR de comprobación posterior a instalación en clon autorizado.
-- No instala, revierte, publica fuentes ni crea identidades. No ejecutado.
BEGIN;
SET LOCAL search_path=pg_catalog;
DO $estructura$
DECLARE
 f oid:=to_regprocedure('vec_personal.consultar_historia_servicios_propios_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 consumidor oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_historia_servicios_propios_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 fuente text;
BEGIN
 IF f IS NULL OR consumidor IS NULL THEN
  RAISE EXCEPTION 'Personal34: dependencias no instaladas' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_personal_propietario'::regrole
      AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u' AND NOT p.proretset AND p.prorettype='jsonb'::regtype
      AND p.proconfig=ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','DateStyle=ISO, YMD','lock_timeout=2s','statement_timeout=15s'])
    OR NOT has_function_privilege('vec_personal_ejecutor',f,'EXECUTE')
    OR has_function_privilege('vec_personal_registrador_frontera',f,'EXECUTE')
    OR EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,'vec_personal_ejecutor'::regrole)
          OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable)) THEN
  RAISE EXCEPTION 'Personal34: función o ACL divergentes'; END IF;
 IF has_function_privilege('vec_personal_ejecutor',consumidor,'EXECUTE')
    OR NOT has_function_privilege('vec_personal_propietario',consumidor,'EXECUTE')
    OR has_table_privilege('vec_personal_ejecutor','vec_personal.servicio_reconocido_historia','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
    OR has_table_privilege('vec_personal_ejecutor','vec_personal.relacion_servicio_historia','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') THEN
  RAISE EXCEPTION 'Personal34: privilegios efectivos demasiado amplios'; END IF;
 SELECT p.prosrc INTO STRICT fuente FROM pg_proc p WHERE p.oid=f;
 IF fuente ~* '(^|;)[[:space:]]*(INSERT[[:space:]]+INTO|UPDATE[[:space:]]+vec_|DELETE[[:space:]]+FROM|TRUNCATE[[:space:]]|CREATE[[:space:]]|ALTER[[:space:]]|DROP[[:space:]]|EXECUTE[[:space:]])'
    OR fuente ~* 'DISTINCT[[:space:]]+ON'
    OR strpos(fuente,'consultar_registro_empleado_rrhh_v1')>0
    OR strpos(fuente,'consultar_ficha_propia_empleado_v1')>0
    OR strpos(fuente,'consumir_historia_servicios_propios_v3_atestada')=0
    OR strpos(fuente,'no_acreditada')=0 THEN
  RAISE EXCEPTION 'Personal34: fuente fuera del contrato de lectura'; END IF;
END $estructura$;
ROLLBACK;
