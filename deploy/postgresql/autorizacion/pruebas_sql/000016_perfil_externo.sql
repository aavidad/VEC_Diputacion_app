\set ON_ERROR_STOP on
-- Ejecutar tras AUT-16 en un clon desechable. No deja datos ni roles.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
CREATE ROLE aut16_ensayo_publicador LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_publicador_candidato_externo TO aut16_ensayo_publicador
 WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT EXECUTE ON FUNCTION vec_autorizacion.publicador_candidato_externo_interno_valido_v1()
 TO aut16_ensayo_publicador;
GRANT EXECUTE ON FUNCTION vec_autorizacion.rol_candidato_externo_acotado_v1(jsonb)
 TO aut16_ensayo_publicador;
SET SESSION AUTHORIZATION aut16_ensayo_publicador;

DO $ensayo$
DECLARE d jsonb; c jsonb; a jsonb; portal jsonb; control_portal jsonb;
        b_rol bytea; b_control bytea; b_asignacion bytea; b_portal bytea; b_control_portal bytea;
        h_rol text; h_control text; h_control_previo text; h_asignacion text; h_asignacion_previa text;
        desde text; hasta text; publicado record;
BEGIN
 d:=jsonb_build_object(
  'rol_id','candidato_bolsa_historial_propio_desarrollo','version',1,
  'nombre','Consulta propia de bolsa en desarrollo','estado','publicada',
  'concesiones',jsonb_build_array(
   jsonb_build_object('accion','bolsa.participaciones_propias.consultar','modulo_id','bolsa',
    'tipo_recurso','participaciones_candidato','finalidades',jsonb_build_array('consulta_participaciones_propias'),
    'campos_permitidos',jsonb_build_array('participaciones_candidato_minimizadas'),'garantia_minima','alto'),
   jsonb_build_object('accion','bolsa.historial_propio.consultar','modulo_id','bolsa',
    'tipo_recurso','participaciones_candidato','finalidades',jsonb_build_array('consulta_historial_propio'),
    'campos_permitidos',jsonb_build_array('contratos_propios','llamamientos_propios','renuncias_propias'),'garantia_minima','alto')),
  'publicada_por','seguridad:desarrollo:no-autoritativa','publicada_en','2026-09-29T00:00:00Z');
 c:=jsonb_build_object('version_rol_ref','rol:candidato_bolsa_historial_propio_desarrollo:v1',
  'revision',1,'estado','habilitada','actualizado_por','seguridad:desarrollo:no-autoritativa',
  'actualizado_en','2026-09-29T00:00:00Z');
 b_rol:=convert_to(d::text,'UTF8'); b_control:=convert_to(c::text,'UTF8');
 h_rol:=encode(sha256(b_rol),'hex'); h_control:=encode(sha256(b_control),'hex');
 SELECT * INTO publicado FROM vec_autorizacion.publicar_rol_candidato_externo_v1(
  b_rol,h_rol,b_control,h_control,0,NULL,'seguridad:desarrollo:no-autoritativa','acto:aut16:rol:ensayo');
 IF publicado.version_rol_ref IS DISTINCT FROM 'rol:candidato_bolsa_historial_propio_desarrollo:v1'
    OR publicado.huella_rol IS DISTINCT FROM h_rol OR publicado.huella_control IS DISTINCT FROM h_control
 THEN RAISE EXCEPTION 'AUT-16: publicación positiva incorrecta'; END IF;
 -- La semilla Go exterior usa la clave i18n del rol portal, nunca su texto.
 portal:=jsonb_set(jsonb_set(d,'{rol_id}','"candidato_bolsa_portal_historial_propio_desarrollo"'::jsonb),
   '{nombre}','"areaPersonal.miBolsa.rolPortal"'::jsonb);
 portal:=jsonb_set(portal,'{concesiones}',(portal->'concesiones')||(
  SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
   'accion',accion,'modulo_id','bolsa',
   'tipo_recurso',CASE WHEN accion='bolsa.participaciones_propias.manifestar_disposicion'
      THEN 'oferta_bolsa' ELSE 'participaciones_candidato' END,
   'finalidades',pg_catalog.jsonb_build_array('gestion_participaciones_propias'),
   'campos_permitidos','[]'::jsonb,'garantia_minima','alto'))
  FROM pg_catalog.unnest(ARRAY[
   'bolsa.participaciones_propias.solicitar_pausa',
   'bolsa.participaciones_propias.solicitar_reactivacion',
   'bolsa.participaciones_propias.responder_llamamiento',
   'bolsa.participaciones_propias.manifestar_disposicion',
   'bolsa.participaciones_propias.confirmar_contacto']) accion));
 IF vec_autorizacion.rol_candidato_externo_acotado_v1(portal) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT-16: semilla i18n del rol portal rechazada'; END IF;
 control_portal:=jsonb_set(c,'{version_rol_ref}',
   '"rol:candidato_bolsa_portal_historial_propio_desarrollo:v1"'::jsonb);
 b_portal:=pg_catalog.convert_to(portal::text,'UTF8');
 b_control_portal:=pg_catalog.convert_to(control_portal::text,'UTF8');
 PERFORM vec_autorizacion.publicar_rol_candidato_externo_v1(
  b_portal,pg_catalog.encode(pg_catalog.sha256(b_portal),'hex'),
  b_control_portal,pg_catalog.encode(pg_catalog.sha256(b_control_portal),'hex'),
  0,NULL,'seguridad:desarrollo:no-autoritativa','acto:aut16:portal:i18n');
 h_control_previo:=h_control;
 c:=jsonb_set(jsonb_set(c,'{revision}','2'::jsonb),'{actualizado_en}',
  '"2026-09-29T00:00:01Z"'::jsonb);
 b_control:=convert_to(c::text,'UTF8'); h_control:=encode(sha256(b_control),'hex');
 SELECT * INTO publicado FROM vec_autorizacion.publicar_rol_candidato_externo_v1(
  b_rol,h_rol,b_control,h_control,1,h_control_previo,
  'seguridad:desarrollo:no-autoritativa','acto:aut16:rol:ensayo-v2');
 IF publicado.revision IS DISTINCT FROM 2 OR publicado.huella_control IS DISTINCT FROM h_control
 THEN RAISE EXCEPTION 'AUT-16: avance positivo de control incorrecto'; END IF;
 BEGIN
  PERFORM 1 FROM vec_autorizacion.asignacion_perfil;
  RAISE EXCEPTION 'AUT-16: tabla compartida visible para publicador';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_autorizacion.publicar_rol_candidato_externo_v1(
   b_rol,h_rol,b_control,h_control,1,h_control_previo,
   'seguridad:desarrollo:no-autoritativa','acto:aut16:rol:repetido');
  RAISE EXCEPTION 'AUT-16: CAS duplicado aceptado';
 EXCEPTION WHEN serialization_failure THEN NULL; END;
 BEGIN
  b_rol:=convert_to(jsonb_set(d,'{concesiones,0,persona_ref}','"per_sintetica"'::jsonb)::text,'UTF8');
  PERFORM vec_autorizacion.publicar_rol_candidato_externo_v1(
   b_rol,encode(sha256(b_rol),'hex'),b_control,h_control,1,h_control_previo,
   'seguridad:desarrollo:no-autoritativa','acto:aut16:rol:pii');
  RAISE EXCEPTION 'AUT-16: dato personal insertable en rol común';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_autorizacion.publicar_asignacion_candidato_externo_v1(
   convert_to('{}','UTF8'),repeat('0',64),0,NULL,'identidad:desarrollo:no-autoritativa','acto:aut16:asignacion:invalida');
  RAISE EXCEPTION 'AUT-16: asignación sin huella válida aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 desde:=to_char((clock_timestamp()-interval '1 minute') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 hasta:=to_char((clock_timestamp()+interval '1 hour') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 a:=jsonb_build_object('asignacion_id','asg_ctx13_sintetica','version',1,
  'perfil_activo_ref','prf_ctx13_sintetico_abcdefghijklmnopqrstuv',
  'principal_id','per_ctx13_sintetica_abcdefghijklmnopqrstuv',
  'version_rol_ref','rol:candidato_bolsa_historial_propio_desarrollo:v1',
  'estado','activa','ambitos',jsonb_build_array(jsonb_build_object(
   'clave','candidato_ref','valores',jsonb_build_array('can_ctx13_sintetico_abcdefghijklmnopqrstuv'))),
  'vigente_desde',desde,'vigente_hasta',hasta,
  'emitida_por','identidad:desarrollo:no-autoritativa','emitida_en',desde);
 b_asignacion:=convert_to(a::text,'UTF8'); h_asignacion:=encode(sha256(b_asignacion),'hex');
 SELECT * INTO publicado FROM vec_autorizacion.publicar_asignacion_candidato_externo_v1(
  b_asignacion,h_asignacion,0,NULL,'identidad:desarrollo:no-autoritativa','acto:aut16:asignacion:v1');
 IF publicado.version IS DISTINCT FROM 1 OR publicado.huella_sha256 IS DISTINCT FROM h_asignacion
 THEN RAISE EXCEPTION 'AUT-16: alta de asignación externa incorrecta'; END IF;
 h_asignacion_previa:=h_asignacion;
 a:=jsonb_set(a,'{version}','2'::jsonb);
 b_asignacion:=convert_to(a::text,'UTF8'); h_asignacion:=encode(sha256(b_asignacion),'hex');
 SELECT * INTO publicado FROM vec_autorizacion.publicar_asignacion_candidato_externo_v1(
  b_asignacion,h_asignacion,1,h_asignacion_previa,
  'identidad:desarrollo:no-autoritativa','acto:aut16:asignacion:v2');
 IF publicado.version IS DISTINCT FROM 2 OR publicado.huella_sha256 IS DISTINCT FROM h_asignacion
 THEN RAISE EXCEPTION 'AUT-16: avance de asignación externa incorrecto'; END IF;
 PERFORM set_config('vec.aut16.huella_asignacion',h_asignacion,true);
 BEGIN
  a:=jsonb_set(a,'{version}','3'::jsonb);
  a:=jsonb_set(a,'{ambitos,0,valores,0}','"can_ctx13_ajeno_abcdefghijklmnopqrstuv"'::jsonb);
  b_asignacion:=convert_to(a::text,'UTF8');
  PERFORM vec_autorizacion.publicar_asignacion_candidato_externo_v1(
   b_asignacion,encode(sha256(b_asignacion),'hex'),2,h_asignacion,
   'identidad:desarrollo:no-autoritativa','acto:aut16:asignacion:ajena');
  RAISE EXCEPTION 'AUT-16: candidato ajeno aceptado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  a:=jsonb_set(a,'{ambitos,0,valores,0}','"can_ctx13_sintetico_abcdefghijklmnopqrstuv"'::jsonb);
  a:=a||jsonb_build_object('dato_personal','no permitido');
  b_asignacion:=convert_to(a::text,'UTF8');
  PERFORM vec_autorizacion.publicar_asignacion_candidato_externo_v1(
   b_asignacion,encode(sha256(b_asignacion),'hex'),2,h_asignacion,
   'identidad:desarrollo:no-autoritativa','acto:aut16:asignacion:extra');
  RAISE EXCEPTION 'AUT-16: propiedad extra aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $ensayo$;

RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $rol_preexistente$
DECLARE r jsonb; c jsonb;
BEGIN
 SELECT documento INTO STRICT r FROM vec_autorizacion.version_rol
 WHERE version_rol_ref='rol:candidato_bolsa_historial_propio_desarrollo:v1';
 r:=jsonb_set(r,'{version}','2'::jsonb);
 r:=jsonb_set(r,'{concesiones,0,campos_permitidos}','["dato_personal"]'::jsonb);
 INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
 VALUES('rol:candidato_bolsa_historial_propio_desarrollo:v2',r->>'rol_id',2,repeat('f',64),
  (r->>'publicada_en')::timestamptz,r);
 c:=jsonb_build_object('version_rol_ref','rol:candidato_bolsa_historial_propio_desarrollo:v2',
  'revision',1,'estado','habilitada','actualizado_por','seguridad:desarrollo:no-autoritativa',
  'actualizado_en','2026-09-29T00:00:00Z');
 INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
 VALUES('rol:candidato_bolsa_historial_propio_desarrollo:v2',1,'habilitada',repeat('e',64),
  (c->>'actualizado_en')::timestamptz,c);
 INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
 VALUES('rol:candidato_bolsa_historial_propio_desarrollo:v2',1,
  (c->>'actualizado_en')::timestamptz,'seguridad:desarrollo:no-autoritativa','acto:aut16:rol:ajeno');
END $rol_preexistente$;
RESET ROLE;
SET SESSION AUTHORIZATION aut16_ensayo_publicador;
DO $rol_inseguro$
DECLARE a jsonb; b bytea; desde text; hasta text;
BEGIN
 desde:=to_char((clock_timestamp()-interval '1 minute') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 hasta:=to_char((clock_timestamp()+interval '1 hour') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 a:=jsonb_build_object('asignacion_id','asg_ctx13_sintetica','version',3,
  'perfil_activo_ref','prf_ctx13_sintetico_abcdefghijklmnopqrstuv',
  'principal_id','per_ctx13_sintetica_abcdefghijklmnopqrstuv',
  'version_rol_ref','rol:candidato_bolsa_historial_propio_desarrollo:v2',
  'estado','activa','ambitos',jsonb_build_array(jsonb_build_object(
   'clave','candidato_ref','valores',jsonb_build_array('can_ctx13_sintetico_abcdefghijklmnopqrstuv'))),
  'vigente_desde',desde,'vigente_hasta',hasta,
  'emitida_por','identidad:desarrollo:no-autoritativa','emitida_en',desde);
 b:=convert_to(a::text,'UTF8');
 BEGIN
  PERFORM vec_autorizacion.publicar_asignacion_candidato_externo_v1(b,encode(sha256(b),'hex'),
   2,current_setting('vec.aut16.huella_asignacion',true),
   'identidad:desarrollo:no-autoritativa','acto:aut16:asignacion:rol-ampliado');
  RAISE EXCEPTION 'AUT-16: rol preexistente ampliado aceptado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $rol_inseguro$;
RESET SESSION AUTHORIZATION;
CREATE ROLE aut16_ensayo_transitivo NOLOGIN INHERIT;
GRANT aut16_ensayo_transitivo TO vec_autorizacion_publicador_candidato_externo
 WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SET SESSION AUTHORIZATION aut16_ensayo_publicador;
DO $transitivo$
BEGIN
 IF vec_autorizacion.publicador_candidato_externo_interno_valido_v1() IS TRUE THEN
  RAISE EXCEPTION 'AUT-16: herencia transitiva aceptada';
 END IF;
END $transitivo$;
RESET SESSION AUTHORIZATION;
DO $estructura$
BEGIN
 IF EXISTS (SELECT 1 FROM pg_catalog.pg_constraint c
  WHERE c.contype='f' AND c.conrelid IN (
   'vec_autorizacion.decision_concedida_contexto_actor_v3_externa'::regclass,
   'vec_autorizacion.decision_denegada_contexto_actor_v3_externa'::regclass)
    AND c.confrelid='vec_autorizacion.asignacion_perfil'::regclass)
 OR (SELECT count(*) FROM pg_catalog.pg_constraint c WHERE c.contype='f'
    AND c.confrelid='vec_autorizacion.asignacion_perfil_externa'::regclass
    AND c.conrelid IN ('vec_autorizacion.decision_concedida_contexto_actor_v3_externa'::regclass,
     'vec_autorizacion.decision_denegada_contexto_actor_v3_externa'::regclass))<>2
 THEN RAISE EXCEPTION 'AUT-16: FK externa no segregada'; END IF;
 IF (SELECT count(*) FROM vec_autorizacion.publicacion_candidato_externo_evento
      WHERE tipo='rol_control' AND objeto_ref='rol:candidato_bolsa_historial_propio_desarrollo:v1')<>2
    OR (SELECT count(*) FROM vec_autorizacion.publicacion_candidato_externo_evento WHERE tipo='asignacion')<>2
 THEN RAISE EXCEPTION 'AUT-16: historia de publicación incorrecta'; END IF;
END $estructura$;
ROLLBACK;
