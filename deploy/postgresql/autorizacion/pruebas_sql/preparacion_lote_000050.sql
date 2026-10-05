\set ON_ERROR_STOP on
-- Vector de la preparación AUT50 y de su encaje con el efecto AUT44. Ejecutar como superusuario en un clon con
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


-- Funciones auxiliares de la preparación.
CREATE FUNCTION pg_temp.material_prep(op_ref text) RETURNS text LANGUAGE sql AS $f$
 SELECT jsonb_build_object('Esquema','administracion_perfiles_lote_preparacion:v1','OperacionRef',op_ref,'ActorPersonaRef',current_setting('prueba.actor'),
  'PerfilActivoRef',current_setting('prueba.actor_perfil'),'AsignacionRef',current_setting('prueba.actor_asig'),'OrganizacionRef',current_setting('prueba.org'),
  'UnidadRef',current_setting('prueba.unidad'),'PersonaRef',current_setting('prueba.destino'))::text
$f$;
CREATE FUNCTION pg_temp.decision_prep(mat text,principal text) RETURNS bytea LANGUAGE sql SECURITY DEFINER AS $f$
 SELECT convert_to(jsonb_build_object('recurso_ref',r->>'recurso_ref','contexto_recurso_huella_sha256',r->>'contexto_sha256','principal_id',principal,
  'perfil_activo_ref',current_setting('prueba.actor_perfil'),'asignacion_ref',current_setting('prueba.actor_asig'),'decision_ref','decision:'||md5(mat))::text,'UTF8')
 FROM (SELECT vec_autorizacion.recurso_preparacion_lote_admin_v1(mat,mat::jsonb) r) x
$f$;
CREATE FUNCTION pg_temp.capacidad_prep(mat text) RETURNS bytea LANGUAGE sql SECURITY DEFINER AS $f$
 SELECT convert_to(jsonb_build_object('efecto_ref',r->>'recurso_ref','huella_efecto_sha256',r->>'contexto_sha256')::text,'UTF8') FROM (SELECT vec_autorizacion.recurso_preparacion_lote_admin_v1(mat,mat::jsonb) r) x
$f$;
CREATE FUNCTION pg_temp.preparar(mat text,p_dec bytea,p_cap bytea) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN
 RETURN vec_autorizacion.preparar_lote_ordinario_admin_v1(mat,p_cap,p_dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00')::text;
EXCEPTION WHEN OTHERS THEN RETURN 'ERROR '||SQLSTATE||' '||SQLERRM;
END $f$;
-- Construye la orden de lote v3 con lo que devuelve la preparación (como haría la pantalla).
CREATE FUNCTION pg_temp.orden_desde(prep jsonb,op_ref text,altas text[],bajas text[]) RETURNS text LANGUAGE sql AS $f$
 SELECT pg_temp.material(op_ref,
  coalesce((SELECT jsonb_agg(jsonb_build_object('Operacion','otorgar','InicioVigencia','inmediato','RolVersionRef',a->>'rol_version_ref','Objetivo',jsonb_build_object(
   'UnidadRef',prep->>'unidad_ref','CentroRef','','CuentaRef',prep->>'cuenta_ref','CuentaVersion',prep->'cuenta_version','PersonaRef',prep->>'persona_ref','PersonaVersion',prep->'persona_version',
   'PerfilRef',a->>'perfil_ref','PerfilVersion',0,'VinculoRef',a->>'vinculo_ref','VinculoVersion',0,'HuellaSHA256',a->>'huella_sha256','RevisionContinuidad',0,
   'ProcedenciaRef',prep->>'procedencia_ref','ProcedenciaVersion',prep->'procedencia_version','ProcedenciaHuellaSHA256',prep->>'procedencia_huella_sha256',
   'VigenteDesde','0001-01-01T00:00:00Z','VigenteHasta',to_char(date_trunc('second',now()+interval '20 hours') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'))))
   FROM jsonb_array_elements(prep->'altas') a WHERE a->>'rol_version_ref'=ANY(altas)),'[]'::jsonb)
  ||coalesce((SELECT jsonb_agg(jsonb_build_object('Operacion','revocar','InicioVigencia','','RolVersionRef',b->>'rol_version_ref','Objetivo',jsonb_build_object(
   'UnidadRef',prep->>'unidad_ref','CentroRef','','CuentaRef',prep->>'cuenta_ref','CuentaVersion',prep->'cuenta_version','PersonaRef',prep->>'persona_ref','PersonaVersion',prep->'persona_version',
   'PerfilRef',b->>'perfil_ref','PerfilVersion',b->'perfil_version','VinculoRef',b->>'vinculo_ref','VinculoVersion',b->'vinculo_version','HuellaSHA256',b->>'huella_sha256','RevisionContinuidad',0,
   'ProcedenciaRef',prep->>'procedencia_ref','ProcedenciaVersion',prep->'procedencia_version','ProcedenciaHuellaSHA256',prep->>'procedencia_huella_sha256',
   'VigenteDesde','0001-01-01T00:00:00Z','VigenteHasta','0001-01-01T00:00:00Z')))
   FROM jsonb_array_elements(prep->'bajas') b WHERE b->>'rol_version_ref'=ANY(bajas)),'[]'::jsonb))
$f$;

-- 1. Preparación: cuenta ordinaria, tres perfiles asignables y ninguno actual.
SELECT pg_temp.material_prep('prep_admin:'||repeat('1',32)) AS p1 \gset
SET SESSION AUTHORIZATION prueba_aut44_lote;
SELECT pg_temp.preparar(:'p1',pg_temp.decision_prep(:'p1',:'actor'),pg_temp.capacidad_prep(:'p1')) AS r1 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('preparacion',:'r1' NOT LIKE 'ERROR%' AND (:'r1'::jsonb)->>'cuenta_ref'=:'cuenta'
 AND (SELECT count(*) FROM jsonb_array_elements((:'r1'::jsonb)->'altas'))=3 AND jsonb_array_length((:'r1'::jsonb)->'bajas')=0
 AND (SELECT count(*) FROM vec_autorizacion.registro_preparacion_lote_admin_v1)=1
 AND ((:'r1'::jsonb)->>'truncado')::boolean IS FALSE AND ((:'r1'::jsonb)#>>'{altas,0,duracion_propuesta_segundos}')::int=86400
 AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.registro_lote_admin_v1));

-- 2. Con esas huellas el lote se aplica (dos altas a la vez).
SELECT pg_temp.orden_desde(:'r1'::jsonb,'acto_admin:'||repeat('c',32),ARRAY['rol:tecnico_rrhh_desarrollo:v1','rol:llamamiento_desarrollo:v1'],ARRAY[]::text[]) AS o2 \gset
SET SESSION AUTHORIZATION prueba_aut44_lote;
SELECT pg_temp.aplicar(:'o2',pg_temp.fuentes(:'o2',2),pg_temp.decision(:'o2',:'actor'),pg_temp.capacidad(:'o2')) AS a2 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('lote_desde_preparacion',:'a2' NOT LIKE 'ERROR%' AND jsonb_array_length((:'a2'::jsonb)->'cambios')=2);

-- 3. Una preparación vieja ya no vale: el lote con sus huellas da 40001.
SELECT pg_temp.orden_desde(:'r1'::jsonb,'acto_admin:'||repeat('d',32),ARRAY['rol:consulta_cuadro_rrhh_desarrollo:v1'],ARRAY[]::text[]) AS o3 \gset
SET SESSION AUTHORIZATION prueba_aut44_lote;
SELECT pg_temp.aplicar(:'o3',pg_temp.fuentes(:'o3',1),pg_temp.decision(:'o3',:'actor'),pg_temp.capacidad(:'o3')) AS a3 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('preparacion_caducada',:'a3' LIKE 'ERROR 40001%');

-- 4. Nueva preparación: ofrece el que falta y ofrece retirar los dos dados.
SELECT pg_temp.material_prep('prep_admin:'||repeat('2',32)) AS p4 \gset
SET SESSION AUTHORIZATION prueba_aut44_lote;
SELECT pg_temp.preparar(:'p4',pg_temp.decision_prep(:'p4',:'actor'),pg_temp.capacidad_prep(:'p4')) AS r4 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('preparacion_actualizada',:'r4' NOT LIKE 'ERROR%' AND jsonb_array_length((:'r4'::jsonb)->'altas')=1 AND jsonb_array_length((:'r4'::jsonb)->'bajas')=2
 AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements((:'r4'::jsonb)->'bajas') b WHERE jsonb_typeof(b->'nombre') IS DISTINCT FROM 'string' OR b->>'nombre'=''));

-- 5. Alta del que falta y baja de uno en el mismo lote.
SELECT pg_temp.orden_desde(:'r4'::jsonb,'acto_admin:'||repeat('e',32),ARRAY['rol:consulta_cuadro_rrhh_desarrollo:v1'],ARRAY['rol:llamamiento_desarrollo:v1']) AS o5 \gset
SET SESSION AUTHORIZATION prueba_aut44_lote;
SELECT pg_temp.aplicar(:'o5',pg_temp.fuentes(:'o5',2),pg_temp.decision(:'o5',:'actor'),pg_temp.capacidad(:'o5')) AS a5 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('alta_y_baja',:'a5' NOT LIKE 'ERROR%' AND (:'a5'::jsonb)#>>'{cambios,1,estado_posterior}'='revocado');

-- 6. Negativos: decisión de otra preparación, para uno mismo, LOGIN ajeno y material con otra unidad.
SELECT pg_temp.material_prep('prep_admin:'||repeat('3',32)) AS p6 \gset
SELECT replace(pg_temp.material_prep('prep_admin:'||repeat('4',32)),:'destino',:'actor') AS p7 \gset
SELECT replace(pg_temp.material_prep('prep_admin:'||repeat('5',32)),:'unidad','unidad:otra') AS p8 \gset
SET SESSION AUTHORIZATION prueba_aut44_lote;
SELECT pg_temp.preparar(:'p6',pg_temp.decision_prep(:'p1',:'actor'),pg_temp.capacidad_prep(:'p6')) AS r6 \gset
SELECT pg_temp.preparar(:'p7',pg_temp.decision_prep(:'p7',:'actor'),pg_temp.capacidad_prep(:'p7')) AS r7 \gset
SELECT pg_temp.preparar(:'p8',pg_temp.decision_prep(:'p8',:'actor'),pg_temp.capacidad_prep(:'p8')) AS r8 \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut44_ajeno;
SELECT pg_temp.preparar(:'p6',pg_temp.decision_prep(:'p6',:'actor'),pg_temp.capacidad_prep(:'p6')) AS r9 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('decision_de_otra_preparacion',:'r6' LIKE 'ERROR 42501%');
SELECT pg_temp.comprobar('preparacion_propia',:'r7' LIKE 'ERROR 22023%');
SELECT pg_temp.comprobar('unidad_fuera_de_ambito',:'r8' LIKE 'ERROR 42501%');
SELECT pg_temp.comprobar('login_ajeno',:'r9' LIKE 'ERROR 42501%');
-- Decisiones cruzadas entre preparación y efecto, y referencia repetida.
SET SESSION AUTHORIZATION prueba_aut44_lote;
SELECT pg_temp.aplicar(:'o5',pg_temp.fuentes(:'o5',2),pg_temp.decision_prep(:'p6',:'actor'),pg_temp.capacidad_prep(:'p6')) AS x1 \gset
SELECT pg_temp.preparar(:'p6',pg_temp.decision(:'o5',:'actor'),pg_temp.capacidad(:'o5')) AS x2 \gset
SELECT pg_temp.preparar(:'p1',pg_temp.decision_prep(:'p1',:'actor'),pg_temp.capacidad_prep(:'p1')) AS x3 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('decision_de_preparacion_en_efecto',:'x1' LIKE 'ERROR 42501%');
SELECT pg_temp.comprobar('decision_de_efecto_en_preparacion',:'x2' LIKE 'ERROR 42501%');
SELECT pg_temp.comprobar('preparacion_repetida',:'x3' LIKE 'ERROR 23505%');
SELECT pg_temp.comprobar('acl',has_function_privilege('vec_admin_perfiles_lote_ejecutor','vec_autorizacion.preparar_lote_ordinario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND NOT has_table_privilege('vec_admin_perfiles_lote_ejecutor','vec_autorizacion.registro_preparacion_lote_admin_v1','SELECT')
 AND (SELECT count(*) FROM vec_autorizacion.registro_preparacion_lote_admin_v1)=2);
ROLLBACK;
