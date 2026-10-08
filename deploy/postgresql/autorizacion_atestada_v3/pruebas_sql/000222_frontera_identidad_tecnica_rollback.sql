\set ON_ERROR_STOP on
-- Ejecutar sólo en base PG18 desechable con AD207/219/221/222 instaladas y
-- sellador AD207 vivo. Todo el material y LOGIN son sintéticos. El ROLLBACK
-- conserva tablas/historia, aunque la secuencia AD207 puede dejar un hueco.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $pre$
BEGIN
 IF pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_frontera_identidad_tecnica_v1(jsonb)') IS NULL
 OR pg_catalog.to_regrole('vec_identidad_frontera_tecnica_ejecutor') IS NULL
 OR pg_catalog.to_regrole('vec_ad222_ensayo_login') IS NOT NULL
 THEN RAISE EXCEPTION 'AD222 prueba: base incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_ad222_ensayo_login
 LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_identidad_frontera_tecnica_ejecutor TO vec_ad222_ensayo_login
 WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
-- La precondición productiva niega TEMP al LOGIN. Se simula sólo en esta TX.
DO $base$ BEGIN EXECUTE pg_catalog.format(
 'REVOKE TEMPORARY ON DATABASE %I FROM PUBLIC',pg_catalog.current_database()); END $base$;
INSERT INTO vec_autorizacion_atestada_v3.config_frontera_identidad_tecnica_v1(
 login_nombre,proceso,canal,superficie,vigente_desde,vigente_hasta)
VALUES('vec_ad222_ensayo_login','vec_interno',
 'identidad_http_interno_preacreditacion','interna_corporativa',
 pg_catalog.clock_timestamp()-interval '1 hour',
 pg_catalog.clock_timestamp()+interval '1 hour');
SET SESSION AUTHORIZATION vec_ad222_ensayo_login;
DO $prueba$
DECLARE c jsonb;e jsonb;g jsonb;v_recurso text;a record;b record;z record;
 v_material bytea;v_clave text;
 v_orden constant text[]:=ARRAY[
  'tipo_registro','evento_ref','operador_login','fase','metodo_esperado',
  'ruta','accion','recurso_ref','resultado','motivo_ref','proceso','canal',
  'superficie','finalidad_ref','correlacion_ref'];
BEGIN
 SELECT vec_autorizacion_atestada_v3.acreditar_frontera_identidad_tecnica_v1() INTO c;
 IF pg_catalog.jsonb_typeof(c) IS DISTINCT FROM 'object'
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(c))<>6
 OR c->>'operador_login' IS DISTINCT FROM session_user::text
 OR c->>'proceso' IS DISTINCT FROM 'vec_interno'
 OR c->>'canal' IS DISTINCT FROM 'identidad_http_interno_preacreditacion'
 OR c->>'superficie' IS DISTINCT FROM 'interna_corporativa'
 OR c->'rutas' IS DISTINCT FROM
  '[{"metodo_esperado":"GET","ruta":"/api/vec/session"},
    {"metodo_esperado":"POST","ruta":"/api/vec/session/start"}]'::jsonb
 OR c->'codigos' IS DISTINCT FROM
  '[{"motivo_ref":"certificado_requerido","resultado":"denegado"},
    {"motivo_ref":"autenticacion_requerida","resultado":"denegado"},
    {"motivo_ref":"acceso_denegado","resultado":"denegado"},
    {"motivo_ref":"metodo_no_permitido","resultado":"denegado"},
    {"motivo_ref":"recurso_no_encontrado","resultado":"denegado"},
    {"motivo_ref":"solicitud_invalida","resultado":"denegado"},
    {"motivo_ref":"servicio_no_disponible","resultado":"error"},
    {"motivo_ref":"respuesta_incompatible","resultado":"error"}]'::jsonb
 THEN RAISE EXCEPTION 'AD222 prueba: preflight incompatible' USING ERRCODE='55000'; END IF;
 v_recurso:='solicitud_sesion:'||pg_catalog.substr(pg_catalog.encode(
  pg_catalog.sha256(pg_catalog.convert_to(
   E'vec.identidad.preacreditacion.solicitud.v1\n'||
   'correlacion_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb','UTF8')),'hex'),1,32);
 -- El evento es fijo sólo en este vector sintético. El adaptador real usa CSPRNG.
 e:=pg_catalog.jsonb_build_object(
  'tipo_registro','pre_identidad_tecnica_v1',
  'evento_ref','evento_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
  'operador_login',session_user::text,'fase','preacreditacion',
  'metodo_esperado','POST','ruta','/api/vec/session/start',
  'accion','registrar_pre_identidad_tecnica_v1','recurso_ref',v_recurso,
  'resultado','denegado','motivo_ref','certificado_requerido',
  'proceso',c->>'proceso','canal',c->>'canal',
  'superficie',c->>'superficie','finalidad_ref','preacreditacion_identidad',
  'correlacion_ref','correlacion_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb');
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac(
  'vec.auditoria.pre-identidad-tecnica.v1');
 FOREACH v_clave IN ARRAY v_orden LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(e->>v_clave);
 END LOOP;
 SELECT * INTO a FROM vec_autorizacion_atestada_v3.registrar_frontera_identidad_tecnica_v1(e);
 SELECT * INTO b FROM vec_autorizacion_atestada_v3.registrar_frontera_identidad_tecnica_v1(e);
 IF a.auditoria_ref IS DISTINCT FROM 'aud_v3_pit_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
 OR a.material_sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(v_material),'hex')
 OR a.correlacion_ref IS DISTINCT FROM 'correlacion_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb'
 OR a.secuencia IS DISTINCT FROM b.secuencia
 OR a.auditoria_ref IS DISTINCT FROM b.auditoria_ref
 OR a.material_sha256 IS DISTINCT FROM b.material_sha256
 OR a.registrada_en IS DISTINCT FROM b.registrada_en
 THEN RAISE EXCEPTION 'AD222 prueba: recibo/replay incompatible' USING ERRCODE='55000'; END IF;
 g:=e||pg_catalog.jsonb_build_object(
  'evento_ref','evento_cccccccccccccccccccccccccccccccc',
  'correlacion_ref','correlacion_dddddddddddddddddddddddddddddddd',
  'recurso_ref','solicitud_sesion:'||pg_catalog.substr(pg_catalog.encode(
    pg_catalog.sha256(pg_catalog.convert_to(
     E'vec.identidad.preacreditacion.solicitud.v1\n'||
     'correlacion_dddddddddddddddddddddddddddddddd','UTF8')),'hex'),1,32),
  'metodo_esperado','GET','ruta','/api/vec/session',
  'motivo_ref','autenticacion_requerida');
 SELECT * INTO z FROM vec_autorizacion_atestada_v3.registrar_frontera_identidad_tecnica_v1(g);
 IF z.auditoria_ref IS DISTINCT FROM 'aud_v3_pit_cccccccccccccccccccccccccccccccc'
 OR z.correlacion_ref IS DISTINCT FROM 'correlacion_dddddddddddddddddddddddddddddddd'
 THEN RAISE EXCEPTION 'AD222 prueba: ruta GET incompatible' USING ERRCODE='55000'; END IF;
 BEGIN
  PERFORM 1 FROM vec_autorizacion_atestada_v3.registrar_frontera_identidad_tecnica_v1(
   e||'{"motivo_ref":"acceso_denegado"}'::jsonb);
  RAISE EXCEPTION 'AD222 prueba: replay distinto aceptado' USING ERRCODE='55000';
 EXCEPTION WHEN unique_violation THEN NULL;
 END;
 BEGIN
  PERFORM 1 FROM vec_autorizacion_atestada_v3.registrar_frontera_identidad_tecnica_v1(
   e||'{"actor_ref":"persona_ajena"}'::jsonb);
  RAISE EXCEPTION 'AD222 prueba: campo personal aceptado' USING ERRCODE='55000';
 EXCEPTION WHEN invalid_parameter_value THEN NULL;
 END;
END $prueba$;
RESET SESSION AUTHORIZATION;
DO $post$
BEGIN
 IF (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
     WHERE evento_ref IN('evento_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
       'evento_cccccccccccccccccccccccccccccccc'))<>2
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
     WHERE a.evento_ref='evento_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
       AND a.tipo_registro='pre_identidad_tecnica_v1'
       AND a.actor_ref IS NULL AND a.perfil_activo_ref IS NULL AND a.decision_ref IS NULL
       AND a.pre_identidad_tecnica_detalle='{"fase":"preacreditacion",
        "metodo_esperado":"POST","ruta":"/api/vec/session/start",
        "superficie":"interna_corporativa"}'::jsonb)
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
     WHERE a.evento_ref='evento_cccccccccccccccccccccccccccccccc'
       AND a.pre_identidad_tecnica_detalle='{"fase":"preacreditacion",
        "metodo_esperado":"GET","ruta":"/api/vec/session",
        "superficie":"interna_corporativa"}'::jsonb)
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.pendiente_sellado_auditoria_v5 p
     JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
       ON a.secuencia=p.secuencia
     WHERE a.evento_ref='evento_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa')
 THEN RAISE EXCEPTION 'AD222 prueba: historia/cola/NULL incompatibles'
 USING ERRCODE='55000'; END IF;
END $post$;
ROLLBACK;
