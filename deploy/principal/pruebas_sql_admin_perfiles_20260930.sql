\set ON_ERROR_STOP on
-- Ejecutar solo en un clon desechable H1+H3+H4 después de CA19 y AUT22.
-- Usa referencias sintéticas ya presentes en el clon y revierte cada ensayo.
\set ON_ERROR_STOP on
BEGIN;
DO $test$
DECLARE p record; v record; l record; lv record; next_version numeric; blocked boolean; msg text;
BEGIN
 SELECT a.* INTO STRICT p FROM vec_contexto_actor_v1.perfil_actual a
 JOIN vec_contexto_actor_v1.perfil_versiones x USING(perfil_ref,version)
 WHERE x.estado='activo' ORDER BY a.perfil_ref LIMIT 1;
 SELECT * INTO STRICT v FROM vec_contexto_actor_v1.perfil_versiones
 WHERE perfil_ref=p.perfil_ref AND version=p.version;
 SELECT max(version)+1 INTO next_version FROM vec_contexto_actor_v1.perfil_versiones WHERE perfil_ref=p.perfil_ref;
 INSERT INTO vec_contexto_actor_v1.perfil_versiones
 (perfil_ref,version,persona_ref,procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES(v.perfil_ref,next_version,v.persona_ref,v.procedencia_ref,v.procedencia_version,v.procedencia_huella_sha256,v.procedencia_autoridad,'revocado',v.vigente_desde,v.vigente_hasta);
 UPDATE vec_contexto_actor_v1.perfil_actual SET version=next_version WHERE perfil_ref=p.perfil_ref;
 INSERT INTO vec_contexto_actor_v1.perfil_versiones
 (perfil_ref,version,persona_ref,procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES(v.perfil_ref,next_version+1,v.persona_ref,v.procedencia_ref,v.procedencia_version,v.procedencia_huella_sha256,v.procedencia_autoridad,'activo',v.vigente_desde,v.vigente_hasta);
 blocked:=false;
 BEGIN
  UPDATE vec_contexto_actor_v1.perfil_actual SET version=next_version+1 WHERE perfil_ref=p.perfil_ref;
 EXCEPTION WHEN check_violation THEN
   GET STACKED DIAGNOSTICS msg = MESSAGE_TEXT;
   blocked := msg = 'perfil revocado no revivible';
 END;
 IF NOT blocked THEN RAISE EXCEPTION 'perfil revivido'; END IF;

 SELECT a.* INTO STRICT l FROM vec_contexto_actor_v1.vinculo_contexto_actual a
 JOIN vec_contexto_actor_v1.vinculo_contexto_versiones x USING(vinculo_ref,version)
 WHERE x.estado='activo' ORDER BY a.vinculo_ref LIMIT 1;
 SELECT * INTO STRICT lv FROM vec_contexto_actor_v1.vinculo_contexto_versiones
 WHERE vinculo_ref=l.vinculo_ref AND version=l.version;
 SELECT max(version)+1 INTO next_version FROM vec_contexto_actor_v1.vinculo_contexto_versiones WHERE vinculo_ref=l.vinculo_ref;
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones
 (vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref,procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES(lv.vinculo_ref,next_version,lv.cuenta_ref,lv.perfil_ref,lv.persona_ref,lv.procedencia_ref,lv.procedencia_version,lv.procedencia_huella_sha256,lv.procedencia_autoridad,'revocado',lv.vigente_desde,lv.vigente_hasta);
 UPDATE vec_contexto_actor_v1.vinculo_contexto_actual SET version=next_version WHERE vinculo_ref=l.vinculo_ref;
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones
 (vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref,procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES(lv.vinculo_ref,next_version+1,lv.cuenta_ref,lv.perfil_ref,lv.persona_ref,lv.procedencia_ref,lv.procedencia_version,lv.procedencia_huella_sha256,lv.procedencia_autoridad,'activo',lv.vigente_desde,lv.vigente_hasta);
 blocked:=false;
 BEGIN
  UPDATE vec_contexto_actor_v1.vinculo_contexto_actual SET version=next_version+1 WHERE vinculo_ref=l.vinculo_ref;
 EXCEPTION WHEN check_violation THEN
   GET STACKED DIAGNOSTICS msg = MESSAGE_TEXT;
   blocked := msg = 'vinculo revocado no revivible';
 END;
 IF NOT blocked THEN RAISE EXCEPTION 'vinculo revivido'; END IF;
END $test$;
ROLLBACK;
\set ON_ERROR_STOP on
BEGIN;
DO $test$
DECLARE actual record; v record; next_version bigint; d jsonb; blocked boolean; msg text;
BEGIN
 SELECT aa.* INTO STRICT actual FROM vec_autorizacion.asignacion_perfil_actual aa
 JOIN vec_autorizacion.asignacion_perfil x ON x.asignacion_ref=aa.asignacion_ref
 WHERE x.documento->>'estado'='activa' ORDER BY aa.perfil_activo_ref LIMIT 1;
 SELECT * INTO STRICT v FROM vec_autorizacion.asignacion_perfil WHERE asignacion_ref=actual.asignacion_ref;
 SELECT max(version)+1 INTO next_version FROM vec_autorizacion.asignacion_perfil WHERE asignacion_id=v.asignacion_id;
 d:=jsonb_set(jsonb_set(v.documento,'{version}',to_jsonb(next_version)),'{estado}','"revocada"'::jsonb);
 INSERT INTO vec_autorizacion.asignacion_perfil
 (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES('asignacion:'||v.asignacion_id||':v'||next_version,v.asignacion_id,next_version,v.perfil_activo_ref,v.principal_id,v.version_rol_ref,
 encode(sha256(convert_to(d::text,'UTF8')),'hex'),v.emitida_en,d);
 UPDATE vec_autorizacion.asignacion_perfil_actual SET asignacion_ref='asignacion:'||v.asignacion_id||':v'||next_version
 WHERE perfil_activo_ref=actual.perfil_activo_ref;
 d:=jsonb_set(jsonb_set(v.documento,'{version}',to_jsonb(next_version+1)),'{estado}','"activa"'::jsonb);
 INSERT INTO vec_autorizacion.asignacion_perfil
 (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES('asignacion:'||v.asignacion_id||':v'||(next_version+1),v.asignacion_id,next_version+1,v.perfil_activo_ref,v.principal_id,v.version_rol_ref,
 encode(sha256(convert_to(d::text,'UTF8')),'hex'),v.emitida_en,d);
 blocked:=false;
 BEGIN
  UPDATE vec_autorizacion.asignacion_perfil_actual SET asignacion_ref='asignacion:'||v.asignacion_id||':v'||(next_version+1)
  WHERE perfil_activo_ref=actual.perfil_activo_ref;
 EXCEPTION WHEN check_violation THEN
  GET STACKED DIAGNOSTICS msg = MESSAGE_TEXT;
  blocked := msg = 'asignacion revocada no revivible';
 END;
 IF NOT blocked THEN RAISE EXCEPTION 'asignacion revivida'; END IF;
END $test$;
ROLLBACK;
DO $comprobacion$
DECLARE numero integer;
BEGIN
  SELECT count(*) INTO numero FROM vec_autorizacion.version_rol r
  CROSS JOIN LATERAL pg_catalog.jsonb_array_elements(r.documento->'concesiones') c
  WHERE r.rol_id='administracion_perfiles'
    AND c->>'modulo_id'='administracion'
    AND c->>'accion'='administracion.perfiles.consultar';
  IF numero <> 1 OR NOT EXISTS (
    SELECT 1 FROM vec_autorizacion.version_rol r
    WHERE r.rol_id='administracion_perfiles'
      AND pg_catalog.jsonb_array_length(r.documento->'concesiones')=1
      AND r.documento->>'retirada_en'='0001-01-01T00:00:00Z')
    OR EXISTS (
    SELECT 1 FROM vec_autorizacion.asignacion_perfil a
    WHERE a.version_rol_ref='rol:administracion_perfiles:v1')
    OR EXISTS (
      SELECT 1 FROM pg_catalog.pg_proc p
      CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) acl
      WHERE p.oid='vec_contexto_actor_v1.listar_perfiles_cuenta_v1(text,text)'::regprocedure
        AND acl.grantee=0 AND acl.privilege_type='EXECUTE')
    OR pg_catalog.has_function_privilege('vec_contexto_actor_v1_runtime',
        'vec_contexto_actor_v1.listar_perfiles_cuenta_v1(text,text)','EXECUTE')
    OR pg_catalog.has_function_privilege('vec_autorizacion_fuente',
        'vec_contexto_actor_v1.listar_perfiles_cuenta_v1(text,text)','EXECUTE')
    OR pg_catalog.has_table_privilege('vec_contexto_actor_v1_runtime',
        'vec_contexto_actor_v1.perfil_actual','SELECT')
  THEN RAISE EXCEPTION 'rol, asignaciones o ACL no acotados'; END IF;
END $comprobacion$;
