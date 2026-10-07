\set ON_ERROR_STOP on
-- Ejecutar sólo como LOGIN técnico efímero del clon PG18, miembro único de
-- vec_personal_ejecutor con INHERIT y sin SET/ADMIN. No crea identidades ni ACL.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
DO $sobretamano$
DECLARE caso record; rechazado boolean;
BEGIN
 IF session_user<>current_user
    OR NOT EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname=session_user
       AND r.rolcanlogin AND r.rolinherit AND NOT r.rolsuper
       AND NOT r.rolcreatedb AND NOT r.rolcreaterole AND NOT r.rolreplication
       AND NOT r.rolbypassrls AND r.rolconfig IS NULL)
    OR NOT EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
       AND m.roleid='vec_personal_ejecutor'::regrole
       AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
    OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)<>1
    OR NOT has_function_privilege(session_user,
       'vec_personal.consultar_historia_servicios_propios_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
       'EXECUTE') THEN
  RAISE EXCEPTION 'Personal34: falta LOGIN técnico de ensayo' USING ERRCODE='55000';
 END IF;
 -- Cada pieza fuera de tamaño tampoco es JSON. La versión sin guarda previa
 -- intentaría convertirla y devolvería 22023. Las demás piezas respetan sus
 -- límites para que cada caso compruebe el argumento indicado.
 FOR caso IN SELECT * FROM (VALUES
  ('capacidad_511',convert_to(repeat('x',511),'UTF8'),convert_to('{}','UTF8')),
  ('capacidad_32769',convert_to(repeat('x',32769),'UTF8'),convert_to('{}','UTF8')),
  ('decision_524289',convert_to(repeat('x',512),'UTF8'),convert_to(repeat('x',524289),'UTF8'))
 ) AS v(nombre,capacidad,decision) LOOP
  rechazado:=false;
  BEGIN
   PERFORM vec_personal.consultar_historia_servicios_propios_empleado_v1(
    '{}',caso.capacidad,caso.decision,decode('01','hex'),
    convert_to('{}','UTF8'),1,1,decode('01','hex'),decode('01','hex'),
    decode('01','hex'),convert_to(repeat('0',44),'UTF8'));
  EXCEPTION WHEN SQLSTATE '42501' THEN rechazado:=true;
  END;
  IF NOT rechazado THEN
   RAISE EXCEPTION 'Personal34: % fuera de tamaño admitida',caso.nombre USING ERRCODE='55000';
  END IF;
 END LOOP;
END $sobretamano$;
ROLLBACK;
