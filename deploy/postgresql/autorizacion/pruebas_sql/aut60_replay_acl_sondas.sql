\set ON_ERROR_STOP on
-- Ejecutar sólo en el clon PG18 después de AD219, AUT58, AUT59, AD220 y AUT60.
-- No crea propuestas: todas las sondas son sintéticas y terminan en ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='15s';

DO $sonda$
DECLARE original jsonb;actual jsonb;grupo oid:=to_regrole('vec_admin_gobierno_roles_ejecutor');
 permitidas oid[]:=ARRAY[
  'vec_autorizacion.proponer_gobierno_rol_nuevo_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure::oid,
  'vec_autorizacion.cerrar_gobierno_rol_nuevo_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure::oid,
  'vec_autorizacion.resolver_rol_administrable_v1(text)'::regprocedure::oid];
BEGIN
 IF grupo IS NULL THEN RAISE EXCEPTION 'AUT60 sonda: grupo ausente';END IF;
 IF EXISTS(SELECT 1 FROM pg_roles r WHERE r.oid=grupo
  AND (r.rolcanlogin OR r.rolinherit OR r.rolsuper OR r.rolcreatedb OR r.rolcreaterole
   OR r.rolreplication OR r.rolbypassrls OR r.rolconfig IS NOT NULL))
 OR EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=grupo)
 THEN RAISE EXCEPTION 'AUT60 sonda: grupo no aislado';END IF;
 original:=jsonb_build_object('esquema','administracion_gobierno_rol_nuevo_propuesta_v1',
  'material_canon','{}','material_sha256',repeat('a',64),'plan_sha256',repeat('b',64),
  'correlacion_ref','correlacion_'||repeat('1',32));
 actual:=jsonb_set(original,'{correlacion_ref}',to_jsonb('correlacion_'||repeat('2',32)));
 IF vec_autorizacion.solicitud_replay_gobierno_rol_nuevo_v1(convert_to(original::text,'UTF8'),actual::text) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT60 sonda: nueva correlacion propuesta rechazada';END IF;
 actual:=jsonb_set(actual,'{material_sha256}',to_jsonb(repeat('c',64)));
 IF vec_autorizacion.solicitud_replay_gobierno_rol_nuevo_v1(convert_to(original::text,'UTF8'),actual::text) IS NOT FALSE
 THEN RAISE EXCEPTION 'AUT60 sonda: material propuesta diferente aceptado';END IF;
 original:=jsonb_build_object('esquema','administracion_gobierno_rol_nuevo_cierre_v1',
  'operacion_ref','cierre_admin:'||repeat('1',32),'propuesta_ref','propuesta_admin:'||repeat('2',32),
  'propuesta_huella_sha256',repeat('a',64),'decision','aprobada',
  'actor_persona_ref','per_'||repeat('a',22),'actor_perfil_ref','prf_'||repeat('b',22),
  'asignacion_ref','asignacion:ejemplo','motivo',jsonb_build_object('entrada_clave','motivo_'||repeat('c',32)),
  'correlacion_ref','correlacion_'||repeat('3',32));
 actual:=jsonb_set(original,'{correlacion_ref}',to_jsonb('correlacion_'||repeat('4',32)));
 IF vec_autorizacion.solicitud_replay_gobierno_rol_nuevo_v1(convert_to(original::text,'UTF8'),actual::text) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT60 sonda: nueva correlacion cierre rechazada';END IF;
 actual:=jsonb_set(actual,'{propuesta_huella_sha256}',to_jsonb(repeat('d',64)));
 IF vec_autorizacion.solicitud_replay_gobierno_rol_nuevo_v1(convert_to(original::text,'UTF8'),actual::text) IS NOT FALSE
 THEN RAISE EXCEPTION 'AUT60 sonda: material cierre diferente aceptado';END IF;
 IF EXISTS(SELECT 1 FROM pg_proc f JOIN pg_namespace n ON n.oid=f.pronamespace
  WHERE n.nspname LIKE 'vec\_%' ESCAPE '\' AND has_function_privilege(grupo,f.oid,'EXECUTE')
  AND f.oid<>ALL(permitidas))
 OR EXISTS(SELECT 1 FROM unnest(permitidas) f WHERE NOT has_function_privilege(grupo,f,'EXECUTE'))
 OR EXISTS(SELECT 1 FROM (VALUES
   ('vec_autorizacion.propuesta_gobierno_rol_nuevo_v1'::regclass),
   ('vec_autorizacion.cierre_gobierno_rol_nuevo_v1'::regclass),
   ('vec_autorizacion.outbox_gobierno_rol_nuevo_v1'::regclass)) t(tabla)
   WHERE has_table_privilege(grupo,t.tabla,'SELECT') OR has_table_privilege(grupo,t.tabla,'INSERT')
    OR has_table_privilege(grupo,t.tabla,'UPDATE') OR has_table_privilege(grupo,t.tabla,'DELETE'))
 THEN RAISE EXCEPTION 'AUT60 sonda: ACL grupo divergente';END IF;
END $sonda$;

-- El LOGIN migrador carece de membresía exclusiva Gov. El fallo de consumo
-- fuerza un intento; al fallar también su auditoría, la fachada debe lanzar
-- excepción, nunca devolver denegación aparentemente asentada.
DO $audit$
DECLARE anteriores bigint;posteriores bigint;fallo text;
BEGIN
 SELECT count(*) INTO anteriores FROM vec_autorizacion.propuesta_gobierno_rol_nuevo_v1;
 BEGIN
  PERFORM vec_autorizacion.proponer_gobierno_rol_nuevo_v1(
   NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'AUT60 sonda: exito sin auditoria';
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS fallo=RETURNED_SQLSTATE;
  IF fallo IS DISTINCT FROM '25000' THEN
   RAISE EXCEPTION 'AUT60 sonda: fallo inesperado %',fallo;END IF;
 END;
 SELECT count(*) INTO posteriores FROM vec_autorizacion.propuesta_gobierno_rol_nuevo_v1;
 IF posteriores<>anteriores THEN RAISE EXCEPTION 'AUT60 sonda: efecto sin auditoria';END IF;
END $audit$;
ROLLBACK;
