\set ON_ERROR_STOP on
-- Prueba focal sobre una copia ficticia ya provista por ADMIN y AUT53.
-- Exige el mismo fichero de parámetros privados que la semilla. Todo termina
-- en ROLLBACK; no genera aprobaciones ni recibos de negocio.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
\set semilla_sin_tx 1
\ir datos_asignaciones_firma_legada.sql

CREATE TEMP TABLE _f163_primera AS
SELECT
 (SELECT count(*) FROM vec_autorizacion.asignacion_perfil) AS historia,
 (SELECT count(*) FROM vec_contexto_actor_v1.correspondencia_organizacion_versiones_v1) AS correspondencias,
 (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
   coalesce(pg_catalog.jsonb_agg(to_jsonb(x) ORDER BY x.perfil_activo_ref)::text,'[]'),'UTF8')),'hex')
  FROM vec_autorizacion.asignacion_perfil_actual x) AS punteros_sha256;

\ir datos_asignaciones_firma_legada.sql
DO $replay$
DECLARE antes record; despues record;
BEGIN
 SELECT * INTO STRICT antes FROM _f163_primera;
 SELECT (SELECT count(*) FROM vec_autorizacion.asignacion_perfil) AS historia,
        (SELECT count(*) FROM vec_contexto_actor_v1.correspondencia_organizacion_versiones_v1) AS correspondencias,
        (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
          coalesce(pg_catalog.jsonb_agg(to_jsonb(x) ORDER BY x.perfil_activo_ref)::text,'[]'),'UTF8')),'hex')
         FROM vec_autorizacion.asignacion_perfil_actual x) AS punteros_sha256 INTO despues;
 IF ROW(antes.historia,antes.correspondencias,antes.punteros_sha256)
    IS DISTINCT FROM ROW(despues.historia,despues.correspondencias,despues.punteros_sha256)
 THEN RAISE EXCEPTION 'F1-63 prueba: segunda pasada cambió historia o punteros' USING ERRCODE='55000'; END IF;
END $replay$;

-- Una asignación canónica vigente para la primera persona debe denegar la
-- semilla, incluso si las cuatro asignaciones legadas ya estaban completas.
SET LOCAL ROLE vec_autorizacion_propietario;
DO $fixture$
DECLARE j jsonb:=pg_catalog.current_setting('vec.f163.firma')::jsonb;
 perfil text:='prf_'||pg_catalog.repeat('e',32);
 id text:='asg_'||pg_catalog.repeat('e',32);
 ref text:='asignacion:asg_'||pg_catalog.repeat('e',32)||':v1';
 persona text:=j#>>'{firmantes,0,persona}'; rol text:='rol:ct_cargo_tecnico_firma_dev:v1';
 d jsonb; sha text;
BEGIN
 IF EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil WHERE asignacion_id=id OR perfil_activo_ref=perfil)
 THEN RAISE EXCEPTION 'F1-63 prueba: referencias del negativo ocupadas' USING ERRCODE='55000'; END IF;
 d:=pg_catalog.jsonb_build_object('asignacion_id',id,'version',1,
  'perfil_activo_ref',perfil,'principal_id',persona,'version_rol_ref',rol,
  'estado','activa','ambitos',pg_catalog.jsonb_build_array(
    pg_catalog.jsonb_build_object('clave','organizacion_ref',
      'valores',pg_catalog.jsonb_build_array(j->>'organizacion_canonica_ref'))),
  'emitida_por','prueba:f1-63:rollback','emitida_en',j->>'emitida_en',
  'vigente_desde',j->>'vigente_desde','vigente_hasta',j->>'vigente_hasta',
  'revocada_en','0001-01-01T00:00:00Z');
 sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
   vec_autorizacion.canon_asignacion_perfil_admin_v1(d),'UTF8')),'hex');
 INSERT INTO vec_autorizacion.asignacion_perfil
  (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,
   version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES(ref,id,1,perfil,persona,rol,sha,(j->>'emitida_en')::timestamptz,d);
 INSERT INTO vec_autorizacion.asignacion_perfil_actual
  (perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
 VALUES(perfil,ref,pg_catalog.clock_timestamp(),'prueba:f1-63:rollback','prueba:f1-63:rollback');
END $fixture$;
RESET ROLE;

SAVEPOINT f163_negativo;
\set ON_ERROR_STOP off
\set solo_precondicion 1
\ir datos_asignaciones_firma_legada.sql
\unset solo_precondicion
\set ON_ERROR_STOP on
ROLLBACK TO SAVEPOINT f163_negativo;
SELECT pg_catalog.set_config('vec.f163.pre_estado',:'f163_pre_estado',true);
DO $negativo$
BEGIN
 IF pg_catalog.current_setting('vec.f163.pre_estado') <> '42501' THEN
  RAISE EXCEPTION 'F1-63 prueba: asignación canónica no denegada' USING ERRCODE='55000';
 END IF;
END $negativo$;
ROLLBACK;
