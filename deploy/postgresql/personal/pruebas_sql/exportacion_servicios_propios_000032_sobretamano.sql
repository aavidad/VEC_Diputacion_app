\set ON_ERROR_STOP on
-- Clon PG18 con Personal32 instalada. Dirección provee sólo en el clon un
-- LOGIN técnico, miembro único de vec_personal_ejecutor. Conectar con role=none.
-- No crea actor, capacidad firmada ni concesión; los tres argumentos son
-- deliberadamente inválidos y deben rechazarse antes del primer parseo.
BEGIN ISOLATION LEVEL SERIALIZABLE, READ WRITE;
SET LOCAL timezone='UTC';
DO $sobretamano$
DECLARE
 f oid:=to_regprocedure('vec_personal.exportar_servicios_propios_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 caso record;
 codigo text;
BEGIN
 IF current_user<>session_user
    OR f IS NULL OR NOT has_schema_privilege(session_user,'vec_personal','USAGE')
    OR NOT has_function_privilege(session_user,f,'EXECUTE')
    OR NOT EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
      AND NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
      AND r.rolconfig IS NULL)
    OR NOT EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
      AND m.roleid='vec_personal_ejecutor'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
    OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)<>1
    OR EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member='vec_personal_ejecutor'::regrole)
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER') THEN
  RAISE EXCEPTION 'Personal32 prueba: clave=login_tecnico esperado=ejecutor_Personal_exclusivo actual=incompatible';
 END IF;
 FOR caso IN SELECT * FROM (VALUES
   ('capacidad_511',511,2),
   ('capacidad_32769',32769,2),
   ('decision_524289',512,524289)) AS casos(nombre,bytes_capacidad,bytes_decision)
 LOOP
  codigo:='aceptado';
  BEGIN
   PERFORM vec_personal.exportar_servicios_propios_empleado_v1(
    '{}',convert_to(repeat('x',caso.bytes_capacidad),'UTF8'),
    convert_to(repeat('x',caso.bytes_decision),'UTF8'),
    convert_to('x','UTF8'),convert_to('x','UTF8'),
    1,1,convert_to('x','UTF8'),convert_to('x','UTF8'),
    convert_to('x','UTF8'),decode(repeat('00',44),'hex'));
  EXCEPTION WHEN OTHERS THEN codigo:=SQLSTATE;
  END;
  IF codigo IS DISTINCT FROM '42501' THEN
   RAISE EXCEPTION 'Personal32 prueba: clave=% esperado=42501_preparse actual=%',caso.nombre,codigo;
  END IF;
 END LOOP;
END $sobretamano$;
ROLLBACK;
