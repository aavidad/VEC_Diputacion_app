\set ON_ERROR_STOP on
-- AD194: repetir registrar y cotejar con el mismo evento devuelve el mismo
-- acuse sin error ni fila nueva. Ejecutar como superusuario migrador, después
-- de AD194. Todo ocurre en una transacción que termina en ROLLBACK: el LOGIN
-- de prueba, su configuración y el evento no quedan en la base.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';SET LOCAL statement_timeout='60s';
DO $estructura$ DECLARE p pg_proc;BEGIN
 FOR p IN SELECT x.* FROM pg_proc x WHERE x.oid IN(
  to_regprocedure('vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_interna_v1(jsonb,text)'),
  to_regprocedure('vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_interna_v1(jsonb,text)')) LOOP
  IF p.proowner<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT p.prosecdef
  OR p.proacl IS DISTINCT FROM ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario']::aclitem[]
  OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC']::text[]
  OR p.prosrc NOT LIKE '%existente.secuencia::bigint%' OR p.prosrc LIKE '%existente.secuencia,%' THEN
   RAISE EXCEPTION 'AD194 vector: % sin postimagen', p.proname;
  END IF;
 END LOOP;
 IF (SELECT count(*) FROM pg_proc WHERE proname IN('registrar_contexto_admin_pre_v2_interna_v1','cotejar_contexto_admin_pre_v2_interna_v1') AND pronamespace='vec_autorizacion_atestada_v3'::regnamespace)<>2 THEN
  RAISE EXCEPTION 'AD194 vector: funciones_ausentes';
 END IF;
END $estructura$;

CREATE ROLE ad194_prueba_login LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_identidad_sesiones_v1_propietario TO ad194_prueba_login;
INSERT INTO vec_autorizacion_atestada_v3.config_contexto_admin_pre_v2_v1(login_nombre,proceso,canal,vigente_desde,vigente_hasta)
VALUES('ad194_prueba_login','ad194.prueba','administracion_privilegiada',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
SELECT set_config('ad194.evento',jsonb_build_object(
 'tipo_registro','contexto_admin_pre_v2','evento_ref','evento_'||md5(clock_timestamp()::text||random()::text),
 'operador_login','ad194_prueba_login','actor_ref',NULL,'perfil_activo_ref',NULL,'accion','resolver_cuenta_admin',
 'recurso_ref','cuenta_admin:ad194','resultado','error','motivo_ref','contexto_admin_pre_v2_error','proceso','ad194.prueba',
 'canal','administracion_privilegiada','finalidad_ref','establecer_contexto_admin',
 'correlacion_ref','correlacion_'||md5(random()::text),'fuente_ref',NULL,'fuente_sha256',NULL)::text,true),
 set_config('ad194.cabeza0',(SELECT secuencia::text FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id),true),
 set_config('ad194.filas0',(SELECT count(*)::text FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3),true) \g /dev/null

SET SESSION AUTHORIZATION ad194_prueba_login;
SELECT set_config('ad194.registro1',to_jsonb(r)::text,true) FROM vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_is_v1(current_setting('ad194.evento')::jsonb) r \g /dev/null
SELECT set_config('ad194.registro2',to_jsonb(r)::text,true) FROM vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_is_v1(current_setting('ad194.evento')::jsonb) r \g /dev/null
SELECT set_config('ad194.cotejo1',to_jsonb(r)::text,true) FROM vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_is_v1(current_setting('ad194.evento')::jsonb) r \g /dev/null
SELECT set_config('ad194.cotejo2',to_jsonb(r)::text,true) FROM vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_is_v1(current_setting('ad194.evento')::jsonb) r \g /dev/null
RESET SESSION AUTHORIZATION;

DO $comprobar$ DECLARE r1 jsonb:=current_setting('ad194.registro1',true)::jsonb;cabeza numeric;filas bigint;BEGIN
 IF r1 IS NULL OR jsonb_typeof(r1->'secuencia')<>'number' THEN RAISE EXCEPTION 'AD194 vector: primer_registro_sin_acuse';END IF;
 IF current_setting('ad194.registro2',true)::jsonb IS DISTINCT FROM r1 THEN RAISE EXCEPTION 'AD194 vector: repeticion_registro_distinta';END IF;
 IF current_setting('ad194.cotejo1',true)::jsonb IS DISTINCT FROM r1 OR current_setting('ad194.cotejo2',true)::jsonb IS DISTINCT FROM r1 THEN RAISE EXCEPTION 'AD194 vector: cotejo_distinto';END IF;
 SELECT secuencia INTO STRICT cabeza FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id;
 SELECT count(*) INTO filas FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 IF cabeza<>current_setting('ad194.cabeza0')::numeric+1 OR (r1->>'secuencia')::numeric<>cabeza OR filas<>current_setting('ad194.filas0')::bigint+1 THEN
  RAISE EXCEPTION 'AD194 vector: cadena_distinta cabeza=% filas=%',cabeza,filas;
 END IF;
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE evento_ref=(current_setting('ad194.evento')::jsonb->>'evento_ref'))<>1 THEN
  RAISE EXCEPTION 'AD194 vector: evento_duplicado';
 END IF;
 RAISE NOTICE 'AD194 vector: OK secuencia=% repeticion_y_cotejo_identicos',r1->>'secuencia';
END $comprobar$;
ROLLBACK;
