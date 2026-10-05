\set ON_ERROR_STOP on
-- Vector del efecto AUT44 (con CA35). Ejecutar como superusuario en un clon con
-- fuentes de organización y unidad vigentes al menos un día más (las altas de
-- prueba terminan en 20 h o 1 día),
-- el arranque 2+1, AUT49, AD190, CA35 y AUT44; todo termina en ROLLBACK.
-- El consumo nominal de AD190 necesita una decisión firmada por el emisor real:
-- aquí se sustituye, SÓLO dentro de esta transacción de prueba, por un doble
-- que devuelve un acuse coherente, para ejercitar todas las reglas del efecto.
-- El consumo real se acredita en el recorrido con vec-admin (A9).
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
CREATE FUNCTION pg_temp.comprobar(caso text,condicion boolean) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN IF condicion IS NOT TRUE THEN RAISE EXCEPTION 'FALLO %',caso; END IF; RETURN 'OK '||caso; END $f$;

-- Doble del consumidor AD190 (se revierte con la transacción).
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_lote_perfiles_admin_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb:=convert_from(p_capacidad,'UTF8')::jsonb;h text:=encode(sha256(p_decision),'hex');
BEGIN
 RETURN QUERY SELECT 'decision:'||substr(h,1,32),c->>'efecto_ref',c->>'huella_efecto_sha256',h,'aud_v3_'||substr(h,1,32),clock_timestamp(),true;
END $f$;

-- Datos del arranque 2+1: actor = persona A (Aplicación), destinataria = persona B.
SELECT (convert_from(plan_canonico,'UTF8')::jsonb)#>>'{personas,0,persona_ref}' AS actor,
 (convert_from(plan_canonico,'UTF8')::jsonb)#>>'{personas,0,perfil_ref}' AS actor_perfil,
 (convert_from(plan_canonico,'UTF8')::jsonb)#>>'{personas,1,persona_ref}' AS destino,
 (convert_from(plan_canonico,'UTF8')::jsonb)#>'{personas,1,ambitos}' AS amb_fuente,
 (convert_from(plan_canonico,'UTF8')::jsonb)#>>'{personas,1,ambitos,0,valores,0}' AS org,
 (convert_from(plan_canonico,'UTF8')::jsonb)#>>'{personas,1,ambitos,1,valores,0}' AS unidad
FROM vec_autorizacion.bootstrap_central_admin_v3 \gset
SELECT q.asignacion_ref AS actor_asig FROM vec_autorizacion.asignacion_perfil_actual q WHERE q.perfil_activo_ref=:'actor_perfil' \gset
SELECT t.cuenta_ref AS cuenta FROM vec_contexto_actor_v1.titularidad_cuenta_persona_v1 t JOIN vec_identidad_sesiones_v1.cuenta c USING(cuenta_ref)
 WHERE t.persona_ref=:'destino' AND NOT c.cuenta_privilegiada \gset
SELECT t.cuenta_ref AS cuenta_priv FROM vec_contexto_actor_v1.titularidad_cuenta_persona_v1 t JOIN vec_identidad_sesiones_v1.cuenta c USING(cuenta_ref)
 WHERE t.persona_ref=:'destino' AND c.cuenta_privilegiada \gset
SELECT pv.version AS pver, pv.procedencia_ref AS prc, pv.procedencia_version AS prcv, pv.procedencia_huella_sha256 AS prch
FROM vec_contexto_actor_v1.persona_actual pa JOIN vec_contexto_actor_v1.persona_versiones pv USING(persona_ref,version) WHERE pa.persona_ref=:'destino' \gset
SELECT cv.version AS cver FROM vec_contexto_actor_v1.proyeccion_cuenta_actual ca JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones cv USING(cuenta_ref,version) WHERE ca.cuenta_ref=:'cuenta' \gset

-- Perfil asignable de prueba (fixture directo; en el entorno lo registra AUT49).
INSERT INTO vec_autorizacion.rol_administrable_exacto_v1
SELECT v.version_rol_ref,'ordinario',v.huella_sha256,now()-interval '1 hour',now()+interval '400 days',false,'vec_autorizacion.administracion_perfiles.lote_ordinario.v1',
 jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(:'org'))),interval '1 day'
FROM vec_autorizacion.version_rol v WHERE v.version_rol_ref IN('rol:tecnico_rrhh_desarrollo:v1','rol:llamamiento_desarrollo:v1','rol:consulta_cuadro_rrhh_desarrollo:v1');
-- Uno más configurado para otra organización: el lote no puede usarlo.
INSERT INTO vec_autorizacion.rol_administrable_exacto_v1
SELECT v.version_rol_ref,'ordinario',v.huella_sha256,now()-interval '1 hour',now()+interval '400 days',false,'vec_autorizacion.administracion_perfiles.lote_ordinario.v1',
 '[{"clave":"organizacion_ref","valores":["organizacion:otra"]}]'::jsonb,interval '1 day'
FROM vec_autorizacion.version_rol v WHERE v.version_rol_ref='rol:respuesta_recibida_desarrollo:v1';

-- Construcción del material (canon Go v3) con la preimagen real.
CREATE FUNCTION pg_temp.cambio(op text,inicio text,perfil text,vinculo text,pv numeric,vv numeric,desde text,hasta text,huella text) RETURNS jsonb LANGUAGE sql AS $f$
 SELECT jsonb_build_object('Operacion',op,'InicioVigencia',inicio,'RolVersionRef','rol:tecnico_rrhh_desarrollo:v1','Objetivo',jsonb_build_object(
  'UnidadRef',current_setting('prueba.unidad'),'CentroRef','','CuentaRef',current_setting('prueba.cuenta'),'CuentaVersion',current_setting('prueba.cver')::numeric,
  'PersonaRef',current_setting('prueba.destino'),'PersonaVersion',current_setting('prueba.pver')::numeric,'PerfilRef',perfil,'PerfilVersion',pv,
  'VinculoRef',vinculo,'VinculoVersion',vv,'HuellaSHA256',huella,'RevisionContinuidad',0,'ProcedenciaRef',current_setting('prueba.prc'),
  'ProcedenciaVersion',current_setting('prueba.prcv')::numeric,'ProcedenciaHuellaSHA256',current_setting('prueba.prch'),'VigenteDesde',desde,'VigenteHasta',hasta))
$f$;
CREATE FUNCTION pg_temp.material(op_ref text,cambios jsonb) RETURNS text LANGUAGE sql AS $f$
 SELECT jsonb_build_object('Esquema','administracion_perfiles_lote:v3','OperacionRef',op_ref,'ActorPersonaRef',current_setting('prueba.actor'),
  'PerfilActivoRef',current_setting('prueba.actor_perfil'),'AsignacionRef',current_setting('prueba.actor_asig'),'OrganizacionRef',current_setting('prueba.org'),
  'Cambios',cambios,'Motivo',jsonb_build_object('catalogo_id','motivos','catalogo_version',1,'catalogo_huella_sha256',repeat('a',64),'entrada_clave','alta'),'ReferenciaActo','')::text
$f$;
-- Recalcula la huella de cada cambio con la preimagen actual.
CREATE FUNCTION pg_temp.sellar(op_ref text,cambios jsonb) RETURNS text LANGUAGE plpgsql AS $f$
DECLARE r jsonb:='[]';c jsonb;
BEGIN
 FOR c IN SELECT value FROM jsonb_array_elements(cambios) LOOP
  r:=r||jsonb_build_array(jsonb_set(c,'{Objetivo,HuellaSHA256}',to_jsonb(encode(sha256(convert_to(vec_autorizacion.preimagen_cambio_lote_admin_v1(c,current_setting('prueba.org'))::text,'UTF8')),'hex'))));
 END LOOP;
 RETURN pg_temp.material(op_ref,r);
END $f$;
CREATE FUNCTION pg_temp.decision(mat text,principal text) RETURNS bytea LANGUAGE sql SECURITY DEFINER AS $f$
 SELECT convert_to(jsonb_build_object('recurso_ref',r->>'recurso_ref','contexto_recurso_huella_sha256',r->>'contexto_sha256','principal_id',principal,
  'perfil_activo_ref',current_setting('prueba.actor_perfil'),'asignacion_ref',current_setting('prueba.actor_asig'),'correlacion_ref','correlacion_'||substr(md5(mat),1,32),
  'decision_ref','decision:'||md5(mat))::text,'UTF8') FROM (SELECT vec_autorizacion.recurso_lote_admin_v1(mat,mat::jsonb) r) x
$f$;
CREATE FUNCTION pg_temp.capacidad(mat text) RETURNS bytea LANGUAGE sql SECURITY DEFINER AS $f$
 SELECT convert_to(jsonb_build_object('efecto_ref',r->>'recurso_ref','huella_efecto_sha256',r->>'contexto_sha256')::text,'UTF8') FROM (SELECT vec_autorizacion.recurso_lote_admin_v1(mat,mat::jsonb) r) x
$f$;
CREATE FUNCTION pg_temp.fuentes(mat text,n int) RETURNS jsonb LANGUAGE sql SECURITY DEFINER AS $f$
 SELECT jsonb_build_object('esquema','vec.admin.perfiles.lote.fuentes.v1','version',1,'solicitud_sha256',encode(sha256(convert_to(mat,'UTF8')),'hex'),
  'organizacion_ref',current_setting('prueba.org'),'unidad_ref',current_setting('prueba.unidad'),
  'ambitos_por_cambio',(SELECT jsonb_agg(current_setting('prueba.amb_fuente')::jsonb) FROM generate_series(1,n)))
$f$;
-- Llama al efecto como el LOGIN del lote y devuelve el recibo o el SQLSTATE.
CREATE FUNCTION pg_temp.aplicar(mat text,fu jsonb,p_dec bytea,p_cap bytea) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN
 RETURN vec_autorizacion.aplicar_lote_ordinario_admin_v1(mat,fu,p_cap,p_dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00')::text;
EXCEPTION WHEN OTHERS THEN RETURN 'ERROR '||SQLSTATE||' '||SQLERRM;
END $f$;
SELECT set_config('prueba.actor',:'actor',true),set_config('prueba.actor_perfil',:'actor_perfil',true),set_config('prueba.actor_asig',:'actor_asig',true),
 set_config('prueba.destino',:'destino',true),set_config('prueba.org',:'org',true),set_config('prueba.unidad',:'unidad',true),set_config('prueba.amb_fuente',:'amb_fuente',true),
 set_config('prueba.cuenta',:'cuenta',true),set_config('prueba.cver',:'cver',true),set_config('prueba.pver',:'pver',true),
 set_config('prueba.prc',:'prc',true),set_config('prueba.prcv',:'prcv',true),set_config('prueba.prch',:'prch',true) \gset
CREATE ROLE prueba_aut44_lote LOGIN;
GRANT vec_admin_perfiles_lote_ejecutor TO prueba_aut44_lote WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
CREATE ROLE prueba_aut44_ajeno LOGIN;
SELECT secuencia AS aud0 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id \gset

-- 1. Alta inmediata de un perfil ordinario a la persona B.
SELECT pg_temp.sellar('acto_admin:'||repeat('1',32),jsonb_build_array(pg_temp.cambio('otorgar','inmediato','prf_prueba_aut44_alta_0000000001','vca_prueba_aut44_alta_0000000001',0,0,'0001-01-01T00:00:00Z',to_char(date_trunc('second',now()+interval '1 day') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),repeat('0',64)))) AS m1 \gset
SET SESSION AUTHORIZATION prueba_aut44_lote;
SELECT pg_temp.aplicar(:'m1',pg_temp.fuentes(:'m1',1),pg_temp.decision(:'m1',:'actor'),pg_temp.capacidad(:'m1')) AS r1 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('alta_inmediata',:'r1' NOT LIKE 'ERROR%'
 AND (:'r1'::jsonb)#>>'{inicios,0,modo}'='inmediato' AND (:'r1'::jsonb)#>>'{inicios,0,vigente_desde}'=(:'r1'::jsonb)->>'confirmado_en'
 AND (:'r1'::jsonb)#>>'{cambios,0,estado_posterior}'='activo' AND (:'r1'::jsonb)#>'{cambios,0,version_posterior}'='1'::jsonb
 AND EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual va JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
  WHERE va.vinculo_ref='vca_prueba_aut44_alta_0000000001' AND v.estado='activo' AND v.cuenta_ref=:'cuenta')
 AND EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)
  WHERE q.perfil_activo_ref='prf_prueba_aut44_alta_0000000001' AND a.principal_id=:'destino' AND a.documento->>'estado'='activa'
  AND a.documento->'ambitos'=jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(:'org')),jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(:'unidad'))))
 AND (SELECT count(*) FROM vec_autorizacion.registro_lote_admin_v1)=1 AND (SELECT count(*) FROM vec_autorizacion.outbox_lote_admin_v1)=1
 AND EXISTS(SELECT 1 FROM vec_contexto_actor_v1.procedencias WHERE procedencia_ref LIKE 'prc_%' AND procedencia_huella_sha256=encode(sha256(convert_to(:'m1','UTF8')),'hex')));

-- 2. Replay con otra decisión: mismo recibo, sin efecto nuevo.
SET SESSION AUTHORIZATION prueba_aut44_lote;
SELECT pg_temp.aplicar(:'m1',NULL,convert_to((convert_from(pg_temp.decision(:'m1',:'actor'),'UTF8')::jsonb||'{"decision_ref":"decision:otra"}')::text,'UTF8'),pg_temp.capacidad(:'m1')) AS r2 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('replay',:'r2'::jsonb=:'r1'::jsonb AND (SELECT count(*) FROM vec_autorizacion.registro_lote_admin_v1)=1);

-- 3. Misma referencia con otro material: 23505 sin efecto.
SELECT pg_temp.sellar('acto_admin:'||repeat('1',32),jsonb_build_array(pg_temp.cambio('otorgar','inmediato','prf_prueba_aut44_otro_00000000001','vca_prueba_aut44_otro_00000000001',0,0,'0001-01-01T00:00:00Z',to_char(date_trunc('second',now()+interval '20 hours') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),repeat('0',64)))) AS m3 \gset
SET SESSION AUTHORIZATION prueba_aut44_lote;
SELECT pg_temp.aplicar(:'m3',pg_temp.fuentes(:'m3',1),pg_temp.decision(:'m3',:'actor'),pg_temp.capacidad(:'m3')) AS r3 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('idempotencia_divergente',:'r3' LIKE 'ERROR 23505%');

-- 4. El mismo perfil otra vez a la misma persona y unidad: 23505.
SELECT pg_temp.sellar('acto_admin:'||repeat('2',32),jsonb_build_array(pg_temp.cambio('otorgar','inmediato','prf_prueba_aut44_doble_0000000001','vca_prueba_aut44_doble_0000000001',0,0,'0001-01-01T00:00:00Z',to_char(date_trunc('second',now()+interval '20 hours') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),repeat('0',64)))) AS m4 \gset
SET SESSION AUTHORIZATION prueba_aut44_lote;
SELECT pg_temp.aplicar(:'m4',pg_temp.fuentes(:'m4',1),pg_temp.decision(:'m4',:'actor'),pg_temp.capacidad(:'m4')) AS r4 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('perfil_ya_asignado',:'r4' LIKE 'ERROR 23505%');

-- 5. Preimagen desactualizada (huella falsa): 40001.
SELECT pg_temp.material('acto_admin:'||repeat('3',32),jsonb_build_array(pg_temp.cambio('otorgar','inmediato','prf_prueba_aut44_cas_00000000001','vca_prueba_aut44_cas_00000000001',0,0,'0001-01-01T00:00:00Z',to_char(date_trunc('second',now()+interval '20 hours') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),repeat('e',64)))) AS m5 \gset
SET SESSION AUTHORIZATION prueba_aut44_lote;
SELECT pg_temp.aplicar(:'m5',pg_temp.fuentes(:'m5',1),pg_temp.decision(:'m5',:'actor'),pg_temp.capacidad(:'m5')) AS r5 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('preimagen_divergente',:'r5' LIKE 'ERROR 40001%');

-- 6. Decisión para otro material o de otra persona: 42501 antes de consumir.
SET SESSION AUTHORIZATION prueba_aut44_lote;
SELECT pg_temp.aplicar(:'m5',pg_temp.fuentes(:'m5',1),pg_temp.decision(:'m1',:'actor'),pg_temp.capacidad(:'m5')) AS r6a \gset
SELECT pg_temp.aplicar(:'m5',pg_temp.fuentes(:'m5',1),pg_temp.decision(:'m5',:'destino'),pg_temp.capacidad(:'m5')) AS r6b \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('decision_ajena',:'r6a' LIKE 'ERROR 42501%' AND :'r6b' LIKE 'ERROR 42501%');

-- 7. Sin fuentes, LOGIN ajeno o superusuario: 42501.
SELECT pg_temp.sellar('acto_admin:'||repeat('4',32),jsonb_build_array(pg_temp.cambio('otorgar','inmediato','prf_prueba_aut44_fu_000000000001','vca_prueba_aut44_fu_000000000001',0,0,'0001-01-01T00:00:00Z',to_char(date_trunc('second',now()+interval '20 hours') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),repeat('0',64)))) AS m7 \gset
SET SESSION AUTHORIZATION prueba_aut44_lote;
SELECT pg_temp.aplicar(:'m7',NULL,pg_temp.decision(:'m7',:'actor'),pg_temp.capacidad(:'m7')) AS r7a \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut44_ajeno;
SELECT pg_temp.aplicar(:'m7',pg_temp.fuentes(:'m7',1),pg_temp.decision(:'m7',:'actor'),pg_temp.capacidad(:'m7')) AS r7b \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.aplicar(:'m7',pg_temp.fuentes(:'m7',1),pg_temp.decision(:'m7',:'actor'),pg_temp.capacidad(:'m7')) AS r7c \gset
SELECT pg_temp.comprobar('sin_fuentes',:'r7a' LIKE 'ERROR 42501%');
SELECT pg_temp.comprobar('login_ajeno',:'r7b' LIKE 'ERROR 42501%');
SELECT pg_temp.comprobar('superusuario',:'r7c' LIKE 'ERROR 42501%');

-- 8. Autoasignación, rol no ordinario y alta programada en pasado.
SELECT replace(pg_temp.sellar('acto_admin:'||repeat('5',32),jsonb_build_array(pg_temp.cambio('otorgar','inmediato','prf_prueba_aut44_auto_00000000001','vca_prueba_aut44_auto_00000000001',0,0,'0001-01-01T00:00:00Z',to_char(date_trunc('second',now()+interval '20 hours') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),repeat('0',64)))),:'destino',:'actor') AS m8a \gset
SELECT replace(pg_temp.sellar('acto_admin:'||repeat('6',32),jsonb_build_array(pg_temp.cambio('otorgar','inmediato','prf_prueba_aut44_rol_000000000001','vca_prueba_aut44_rol_000000000001',0,0,'0001-01-01T00:00:00Z',to_char(date_trunc('second',now()+interval '20 hours') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),repeat('0',64)))),'rol:tecnico_rrhh_desarrollo:v1','rol:respuesta_recibida_desarrollo:v1') AS m8b \gset
SELECT pg_temp.sellar('acto_admin:'||repeat('7',32),jsonb_build_array(pg_temp.cambio('otorgar','programado','prf_prueba_aut44_prog_00000000001','vca_prueba_aut44_prog_00000000001',0,0,to_char(date_trunc('second',now()-interval '1 day') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),to_char(date_trunc('second',now()+interval '20 hours') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),repeat('0',64)))) AS m8c \gset
SET SESSION AUTHORIZATION prueba_aut44_lote;
SELECT pg_temp.aplicar(:'m8a',pg_temp.fuentes(:'m8a',1),pg_temp.decision(:'m8a',:'actor'),pg_temp.capacidad(:'m8a')) AS r8a \gset
SELECT pg_temp.aplicar(:'m8b',pg_temp.fuentes(:'m8b',1),pg_temp.decision(:'m8b',:'actor'),pg_temp.capacidad(:'m8b')) AS r8b \gset
SELECT pg_temp.aplicar(:'m8c',pg_temp.fuentes(:'m8c',1),pg_temp.decision(:'m8c',:'actor'),pg_temp.capacidad(:'m8c')) AS r8c \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('autoasignacion',:'r8a' LIKE 'ERROR 22023%');
SELECT pg_temp.comprobar('perfil_de_otra_organizacion',:'r8b' LIKE 'ERROR 42501%');
-- Cuenta privilegiada de la destinataria y material con un valor null.
SELECT pg_temp.sellar('acto_admin:'||repeat('b',32),jsonb_build_array(jsonb_set(jsonb_set(pg_temp.cambio('otorgar','inmediato','prf_prueba_aut44_priv_00000000001','vca_prueba_aut44_priv_00000000001',0,0,'0001-01-01T00:00:00Z',to_char(date_trunc('second',now()+interval '20 hours') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),repeat('0',64)),'{Objetivo,CuentaRef}',to_jsonb(:'cuenta_priv'::text)),'{RolVersionRef}','"rol:llamamiento_desarrollo:v1"'))) AS m8d \gset
SELECT replace(:'m8d','"ReferenciaActo": ""','"ReferenciaActo": null') AS m8e \gset
SET SESSION AUTHORIZATION prueba_aut44_lote;
SELECT pg_temp.aplicar(:'m8d',pg_temp.fuentes(:'m8d',1),pg_temp.decision(:'m8d',:'actor'),pg_temp.capacidad(:'m8d')) AS r8d \gset
SELECT pg_temp.aplicar(:'m8e',pg_temp.fuentes(:'m8e',1),pg_temp.decision(:'m8e',:'actor'),pg_temp.capacidad(:'m8e')) AS r8e \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('cuenta_privilegiada',:'r8d' LIKE 'ERROR 42501%');
SELECT pg_temp.comprobar('material_con_null',:'r8e' LIKE 'ERROR 22023%');
SELECT pg_temp.comprobar('programado_pasado',:'r8c' LIKE 'ERROR 22023%');

-- 9. Baja del perfil dado de alta en 1: nueva versión revocada, sin reactivar.
SELECT pg_temp.sellar('acto_admin:'||repeat('8',32),jsonb_build_array(pg_temp.cambio('revocar','','prf_prueba_aut44_alta_0000000001','vca_prueba_aut44_alta_0000000001',1,1,'0001-01-01T00:00:00Z','0001-01-01T00:00:00Z',repeat('0',64)))) AS m9 \gset
SET SESSION AUTHORIZATION prueba_aut44_lote;
SELECT pg_temp.aplicar(:'m9',pg_temp.fuentes(:'m9',1),pg_temp.decision(:'m9',:'actor'),pg_temp.capacidad(:'m9')) AS r9 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('baja',:'r9' NOT LIKE 'ERROR%' AND (:'r9'::jsonb)#>>'{cambios,0,estado_posterior}'='revocado'
 AND (:'r9'::jsonb)#>'{cambios,0,version_posterior}'='2'::jsonb AND (:'r9'::jsonb)#>>'{inicios,0,modo}'=''
 AND (:'r9'::jsonb)#>>'{cambios,0,vigente_desde}'=(:'r1'::jsonb)#>>'{cambios,0,vigente_desde}'
 AND EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual va JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
  WHERE va.vinculo_ref='vca_prueba_aut44_alta_0000000001' AND v.version=2 AND v.estado='revocado')
 AND EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)
  WHERE q.perfil_activo_ref='prf_prueba_aut44_alta_0000000001' AND a.version=2 AND a.documento->>'estado'='revocada'));

-- 10. Repetir la baja con la preimagen ya revocada: 40001.
SELECT pg_temp.sellar('acto_admin:'||repeat('9',32),jsonb_build_array(pg_temp.cambio('revocar','','prf_prueba_aut44_alta_0000000001','vca_prueba_aut44_alta_0000000001',2,2,'0001-01-01T00:00:00Z','0001-01-01T00:00:00Z',repeat('0',64)))) AS m10 \gset
SET SESSION AUTHORIZATION prueba_aut44_lote;
SELECT pg_temp.aplicar(:'m10',pg_temp.fuentes(:'m10',1),pg_temp.decision(:'m10',:'actor'),pg_temp.capacidad(:'m10')) AS r10 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('baja_de_revocado',:'r10' LIKE 'ERROR%' AND (SELECT count(*) FROM vec_autorizacion.registro_lote_admin_v1)=2);

-- 11. Lote de dos altas programadas en una sola transacción.
CREATE FUNCTION pg_temp.con_rol(c jsonb,rol text) RETURNS jsonb LANGUAGE sql AS $f$ SELECT jsonb_set(c,'{RolVersionRef}',to_jsonb(rol)) $f$;
SELECT pg_temp.sellar('acto_admin:'||repeat('a',32),jsonb_build_array(
 pg_temp.con_rol(pg_temp.cambio('otorgar','programado','prf_prueba_aut44_prog_a000000001','vca_prueba_aut44_prog_a000000001',0,0,to_char(date_trunc('second',now()+interval '1 hour') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),to_char(date_trunc('second',now()+interval '20 hours') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),repeat('0',64)),'rol:llamamiento_desarrollo:v1'),
 pg_temp.con_rol(pg_temp.cambio('otorgar','programado','prf_prueba_aut44_prog_b000000001','vca_prueba_aut44_prog_b000000001',0,0,to_char(date_trunc('second',now()+interval '2 hours') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),to_char(date_trunc('second',now()+interval '20 hours') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),repeat('0',64)),'rol:consulta_cuadro_rrhh_desarrollo:v1'))) AS m11 \gset
SET SESSION AUTHORIZATION prueba_aut44_lote;
SELECT pg_temp.aplicar(:'m11',pg_temp.fuentes(:'m11',2),pg_temp.decision(:'m11',:'actor'),pg_temp.capacidad(:'m11')) AS r11 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('lote_dos_programadas',:'r11' NOT LIKE 'ERROR%' AND jsonb_array_length((:'r11'::jsonb)->'cambios')=2
 AND (:'r11'::jsonb)#>>'{inicios,1,modo}'='programado' AND ((:'r11'::jsonb)#>>'{inicios,0,vigente_desde}')::timestamptz>((:'r11'::jsonb)->>'confirmado_en')::timestamptz
 AND (SELECT count(*) FROM vec_autorizacion.registro_lote_admin_v1)=3);

-- 12. ACL: el grupo sólo ejecuta la fachada y la acreditación del LOGIN.
SELECT pg_temp.comprobar('acl',has_function_privilege('vec_admin_perfiles_lote_ejecutor','vec_autorizacion.aplicar_lote_ordinario_admin_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND NOT has_function_privilege('vec_admin_perfiles_lote_ejecutor','vec_autorizacion.preimagen_cambio_lote_admin_v1(jsonb,text)','EXECUTE')
 AND NOT has_function_privilege('vec_admin_perfiles_lote_ejecutor','vec_contexto_actor_v1.registrar_procedencia_acto_admin_lote_v1(text,text)','EXECUTE')
 AND has_function_privilege('vec_admin_perfiles_lote_ejecutor','vec_autorizacion.resolver_rol_administrable_v1(text)','EXECUTE')
 AND NOT has_table_privilege('vec_admin_perfiles_lote_ejecutor','vec_autorizacion.rol_administrable_exacto_v1','SELECT')
 AND NOT has_table_privilege('vec_admin_perfiles_lote_ejecutor','vec_autorizacion.registro_lote_admin_v1','SELECT'));
ROLLBACK;
