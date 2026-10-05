\set ON_ERROR_STOP on
-- Regresión P2: 50/51 limita Personas, nunca perfiles de una Persona.
-- Fixture sintética de metadata/proyección privada. No acredita PDP, sesión,
-- permiso HTTP ni consumo V3 favorable. Sólo clon desechable y ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL search_path=pg_catalog;
CREATE TEMP TABLE aut43_51_control(persona text,org text,unidad text,procedencia text,rol text,desde timestamptz,hasta timestamptz);
INSERT INTO aut43_51_control
 SELECT 'per_'||replace(gen_random_uuid()::text,'-',''),'org_'||replace(gen_random_uuid()::text,'-',''),
 'unidad:'||replace(gen_random_uuid()::text,'-',''),'prc_'||replace(gen_random_uuid()::text,'-',''),r.version_rol_ref,
 date_trunc('second',clock_timestamp())-interval '1 second',date_trunc('second',clock_timestamp())+interval '1 hour'
 FROM vec_autorizacion.version_rol r WHERE r.rol_id<>'administracion_perfiles' AND r.documento->>'estado'='publicada'
 AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.rol_sensible_exacto s WHERE s.version_rol_ref=r.version_rol_ref)
 ORDER BY r.version_rol_ref COLLATE "C" LIMIT 1;
DO $pre$
BEGIN
 IF(SELECT count(*) FROM pg_temp.aut43_51_control)<>1 THEN RAISE EXCEPTION 'AUT43: PARO clave=fixture_rol actual=ausente esperado=rol_ordinario_publicado_existente' USING ERRCODE='55000';END IF;
END $pre$;
CREATE TEMP TABLE aut43_51_perfil(perfil text PRIMARY KEY,asignacion_id text,asignacion_ref text);
INSERT INTO aut43_51_perfil
 SELECT 'prf_'||replace(gen_random_uuid()::text,'-',''),'asg_'||replace(gen_random_uuid()::text,'-',''),NULL FROM generate_series(1,51);
UPDATE aut43_51_perfil SET asignacion_ref='asignacion:'||asignacion_id||':v1';
GRANT SELECT ON TABLE pg_temp.aut43_51_control,pg_temp.aut43_51_perfil TO vec_contexto_actor_v1_propietario,vec_autorizacion_propietario;

SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.procedencias
 SELECT procedencia,1,encode(sha256(convert_to('fixture_sintetica_aut43_51_perfiles','UTF8')),'hex'),'no_autoritativa' FROM pg_temp.aut43_51_control;
INSERT INTO vec_contexto_actor_v1.persona_versiones
 SELECT persona,1,procedencia,1,encode(sha256(convert_to('fixture_sintetica_aut43_51_perfiles','UTF8')),'hex'),'no_autoritativa','activo',desde,hasta FROM pg_temp.aut43_51_control;
INSERT INTO vec_contexto_actor_v1.persona_actual SELECT persona,1 FROM pg_temp.aut43_51_control;
INSERT INTO vec_contexto_actor_v1.perfil_versiones
 SELECT p.perfil,1,c.persona,c.procedencia,1,encode(sha256(convert_to('fixture_sintetica_aut43_51_perfiles','UTF8')),'hex'),'no_autoritativa','activo',c.desde,c.hasta
 FROM pg_temp.aut43_51_perfil p CROSS JOIN pg_temp.aut43_51_control c;
INSERT INTO vec_contexto_actor_v1.perfil_actual SELECT perfil,1 FROM pg_temp.aut43_51_perfil;
RESET ROLE;

SET LOCAL ROLE vec_autorizacion_propietario;
-- No se altera ningún perfil fijo ni se salta su sello: sólo rol ordinario
-- existente, sin LOGIN, cuenta, vínculo de autenticación o concesión nueva.
INSERT INTO vec_autorizacion.asignacion_perfil(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 SELECT p.asignacion_ref,p.asignacion_id,1,p.perfil,c.persona,c.rol,
 encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(d.documento),'UTF8')),'hex'),c.desde,d.documento
 FROM pg_temp.aut43_51_perfil p CROSS JOIN pg_temp.aut43_51_control c
 CROSS JOIN LATERAL(SELECT jsonb_build_object('asignacion_id',p.asignacion_id,'version',1,'perfil_activo_ref',p.perfil,'principal_id',c.persona,'version_rol_ref',c.rol,
 'estado','activa','ambitos',jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(c.org)),jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(c.unidad))),
 'emitida_por','ensayo:aut43:51_perfiles','emitida_en',to_char(c.desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
 'vigente_desde',to_char(c.desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),'vigente_hasta',to_char(c.hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),'revocada_en','0001-01-01T00:00:00Z') AS documento)d;
INSERT INTO vec_autorizacion.asignacion_perfil_actual
 SELECT perfil,asignacion_ref,c.desde,'ensayo:aut43:51_perfiles','acto:ensayo:aut43:51_perfiles' FROM pg_temp.aut43_51_perfil p CROSS JOIN pg_temp.aut43_51_control c;
DO $regresion$
DECLARE c record;r jsonb;m jsonb;pagina jsonb;ficha jsonb;
BEGIN
 SELECT * INTO STRICT c FROM pg_temp.aut43_51_control;
 ficha:=vec_autorizacion.proyectar_persona_usuarios_admin_v1(c.persona,c.org,c.unidad);
 IF ficha->>'persona_ref' IS DISTINCT FROM c.persona OR jsonb_array_length(ficha->'perfiles')<>51
 OR EXISTS(SELECT 1 FROM pg_temp.aut43_51_perfil p WHERE NOT EXISTS(SELECT 1 FROM jsonb_array_elements(ficha->'perfiles') x WHERE x->>'perfil_ref'=p.perfil))
 THEN RAISE EXCEPTION 'AUT43: perfiles_truncados_o_rechazados';END IF;
 m:=jsonb_build_object('esquema','vec.admin.usuarios.listar.v1','organizacion_ref',c.org,'unidad_ref',c.unidad,'conjunto_ref',vec_autorizacion.conjunto_usuarios_admin_v1(c.org,c.unidad),
 'filtros',jsonb_build_object('perfil_ref','','unidad_ref','','estado',''),'cursor','','limite',50);
 pagina:=vec_autorizacion.proyectar_usuarios_admin_v1(m);
 IF jsonb_array_length(pagina->'personas')<>1 OR pagina->>'siguiente_cursor' IS DISTINCT FROM '' OR jsonb_array_length(pagina#>'{personas,0,perfiles}')<>51 THEN RAISE EXCEPTION 'AUT43: pagina_limita_perfiles';END IF;
END $regresion$;
RESET ROLE;
ROLLBACK;
