\set ON_ERROR_STOP on
-- Sonda focal tras AUT68 UP en copia aislada PostgreSQL 18.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $prueba$
DECLARE f oid;firma text;accion text;n integer;rechazo boolean:=false;
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 THEN RAISE EXCEPTION 'AUT68: requiere PG18';END IF;
 IF pg_catalog.has_function_privilege('vec_autorizacion_propietario',
  'vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(text,text)','EXECUTE') IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT68: CA39 sin ACL de propietario';END IF;
 IF pg_catalog.has_function_privilege('vec_autorizacion_propietario',
  'vec_personal.cotejar_unidad_bootstrap_admin_v1(text,jsonb,timestamptz)','EXECUTE') IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT68: Personal31 sin ACL de propietario';END IF;
 FOREACH firma IN ARRAY ARRAY[
  'vec_autorizacion.administradores_version_inscripcion_v1()',
  'vec_autorizacion.acreditar_cambios_empleado_inscripcion_v1(jsonb)',
  'vec_autorizacion.acreditar_cambios_externos_inscripcion_v1(jsonb)',
  'vec_autorizacion.acreditar_cambios_rrhh_inscripcion_v1(jsonb,text)',
  'vec_autorizacion.revalidar_catalogo_cierre_inscripcion_v1(jsonb)',
  'vec_autorizacion.solicitud_replay_version_inscripcion_v1(bytea,text)',
  'vec_autorizacion.comprobar_postimagen_version_inscripcion_v1(text)',
  'vec_autorizacion.aplicar_version_inscripcion_v1(boolean,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_autorizacion.registrar_fallo_version_inscripcion_v1(text,text,text)',
  'vec_autorizacion.proponer_version_inscripcion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_autorizacion.cerrar_version_inscripcion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
  f:=pg_catalog.to_regprocedure(firma);
  IF f IS NULL THEN RAISE EXCEPTION 'AUT68: falta %',firma;END IF;
  IF EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
   CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,
    pg_catalog.acldefault('f',p.proowner))) acl
   WHERE p.oid=f AND acl.grantee=0 AND acl.privilege_type='EXECUTE')
  THEN RAISE EXCEPTION 'AUT68: función pública %',firma;END IF;
 END LOOP;
 FOR accion IN SELECT x FROM pg_catalog.unnest(ARRAY[
  'vec_autorizacion.proponer_version_inscripcion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_autorizacion.cerrar_version_inscripcion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)']) x LOOP
  IF NOT pg_catalog.has_function_privilege('vec_admin_version_inscripcion_ejecutor',accion,'EXECUTE')
  THEN RAISE EXCEPTION 'AUT68: fachada no concedida %',accion;END IF;
 END LOOP;
 IF vec_autorizacion.solicitud_replay_version_inscripcion_v1(
  pg_catalog.convert_to('{"material":"a","correlacion_ref":"correlacion_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}','UTF8'),
  '{"material":"a","correlacion_ref":"correlacion_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}') IS NOT TRUE
 OR vec_autorizacion.solicitud_replay_version_inscripcion_v1(
  pg_catalog.convert_to('{"material":"a","correlacion_ref":"correlacion_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}','UTF8'),
  '{"material":"b","correlacion_ref":"correlacion_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}') IS NOT FALSE
 THEN RAISE EXCEPTION 'AUT68: replay semántico abierto';END IF;
 BEGIN
  PERFORM vec_autorizacion.acreditar_cambios_empleado_inscripcion_v1(
   '{"perfil_objetivo":"empleado","asignaciones":[{"principal_id":"per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","perfil_activo_ref":"prf_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","empleado_ref":"emp_cccccccccccccccccccccccccccccccc","proyeccion_empleado":{"proyeccion_ref":"pep_dddddddddddddddddddddddddddddddd","version":1,"procedencia_ref":"origen_inexistente","procedencia_version":1,"procedencia_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}]}'::jsonb);
 EXCEPTION WHEN SQLSTATE '42501' THEN rechazo:=true;
 END;
 IF NOT rechazo THEN RAISE EXCEPTION 'AUT68: empleado sin CA39 admitido';END IF;
 rechazo:=false;
 BEGIN
  PERFORM vec_autorizacion.acreditar_cambios_rrhh_inscripcion_v1(
   '{"perfil_objetivo":"rrhh","asignaciones":[{"ambitos":[{"clave":"unidad_ref","valores":["unidad:rrhh:sintetica"]}],"vigente_hasta":"2026-12-31T00:00:00Z"}]}'::jsonb,
   'organizacion:sintetica');
 EXCEPTION WHEN SQLSTATE '42501' THEN rechazo:=true;
 END;
 IF NOT rechazo THEN RAISE EXCEPTION 'AUT68: RRHH sin procedencia Personal31 admitido';END IF;
 rechazo:=false;
 BEGIN
  PERFORM vec_autorizacion.acreditar_cambios_rrhh_inscripcion_v1(
   '{"perfil_objetivo":"rrhh","asignaciones":[{"asignacion_id":"asg_aut68_sintetico","ambitos":[{"clave":"unidad_ref","valores":["unidad:aut68:inexistente"]}],"fuente_unidad":{"referencia":"fuente:aut68:inexistente","version":1,"huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"vigente_hasta":"2099-12-31T00:00:00Z"}]}'::jsonb,
   'organizacion:aut68:inexistente');
 EXCEPTION WHEN SQLSTATE '42501' THEN rechazo:=true;
 END;
 IF NOT rechazo THEN RAISE EXCEPTION 'AUT68: unidad sin Personal31 admitida';END IF;
 rechazo:=false;
 BEGIN
  PERFORM vec_autorizacion.revalidar_catalogo_cierre_inscripcion_v1(
   '{"catalogo_ref":"catalogo:aut68:inexistente","catalogo_version":1,"catalogo_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","selecciones":[]}'::jsonb);
 EXCEPTION WHEN SQLSTATE '40001' THEN rechazo:=true;
 END;
 IF NOT rechazo THEN RAISE EXCEPTION 'AUT68: cabeza inexistente admitida';END IF;
 SELECT pg_catalog.count(*) INTO n FROM vec_autorizacion.propuesta_version_inscripcion_v1;
 IF n<>0 THEN RAISE EXCEPTION 'AUT68: propuesta sembrada';END IF;
 SELECT pg_catalog.count(*) INTO n FROM vec_autorizacion.cierre_version_inscripcion_v1;
 IF n<>0 THEN RAISE EXCEPTION 'AUT68: cierre sembrado';END IF;
 SELECT pg_catalog.count(*) INTO n FROM vec_autorizacion.outbox_version_inscripcion_v1;
 IF n<>0 THEN RAISE EXCEPTION 'AUT68: outbox sembrado';END IF;
END $prueba$;
ROLLBACK;
