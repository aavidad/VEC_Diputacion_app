\set ON_ERROR_STOP on
-- Datos sintéticos y auxiliares SOLO del arnés desechable de 000007.
CREATE ROLE login_cronos_prueba LOGIN INHERIT;
GRANT vec_cronos_v1_ejecutor TO login_cronos_prueba WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
CREATE ROLE login_cronos_auditor_prueba LOGIN INHERIT;
GRANT vec_cronos_v1_auditor TO login_cronos_auditor_prueba WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;

BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
INSERT INTO vec_cronos_v1.clasificacion_canal VALUES
 ('politica:canal:cronos:v1','portal-empleado-web','remoto','mtls-certificado','remoto','fuente:sintetica:canal',clock_timestamp()),
 ('politica:canal:cronos:v1','terminal-sintetico','terminal','tarjeta','terminal','fuente:sintetica:canal',clock_timestamp());
SELECT set_config('vec.cronos.empleado_ref','emp_AAAAAAAAAAAAAAAAAAAAAA',true);
INSERT INTO vec_cronos_v1.teletrabajo_autorizacion
 (autorizacion_ref,empleado_ref,periodo,resolucion_ref,politica_version_ref,actor_resolutor_ref,auditoria_ref,registrada_en)
VALUES ('teletrabajo:cronos:sintetico-a','emp_AAAAAAAAAAAAAAAAAAAAAA',
 tstzrange(date_trunc('hour',clock_timestamp())-interval '1 hour',date_trunc('hour',clock_timestamp())+interval '9 hours','[)'),
 'resolucion:sintetica:tt-a','politica:teletrabajo:1','per_RRRRRRRRRRRRRRRRRRRRRR','auditoria:sintetica:tt-a',clock_timestamp());
INSERT INTO vec_cronos_v1.programacion_jornada
 (programacion_ref,empleado_ref,fecha,version,turno_ref,politica_version_ref,fuente_ref,zona_horaria,minutos_previstos,publicada_en)
VALUES ('programacion:cronos:sintetica-a',
 'emp_AAAAAAAAAAAAAAAAAAAAAA',(clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date,1,'turno:manana','politica:jornada:1',
 'fuente:sintetica:jornada','Europe/Madrid',450,clock_timestamp());
COMMIT;

CREATE SCHEMA prueba;
GRANT USAGE ON SCHEMA prueba TO login_cronos_prueba;
-- Construye material V3 de prueba ligado a recurso y huella como Go.
CREATE FUNCTION prueba.v3(p_audiencia text,p_accion text,p_tipo text,p_finalidad text,p_recurso text,
  p_material text,p_empleado text,p_actor text,p_perfil text,p_nonce text,p_vinculos jsonb DEFAULT NULL)
RETURNS TABLE(cap bytea,dcs bytea,ctx bytea) LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE huella text; ahora timestamptz:=clock_timestamp();
BEGIN
 huella:=encode(sha256(convert_to('{"ambitos":{"empleado_ref":'||to_jsonb(p_empleado)::text||'},"atributos":{"material_sha256":"'||
   encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}','UTF8')),'hex');
 RETURN QUERY SELECT
  convert_to(jsonb_build_object('nonce',p_nonce,'decision_ref','decision:prueba:'||p_nonce,'efecto_ref',p_recurso,
    'huella_efecto_sha256',huella,'audiencia_consumo',p_audiencia,'operacion',p_accion,
    'expira_en',ahora+interval '30 seconds','decision_valida_hasta',ahora+interval '30 seconds',
    'configuracion_expira_en',ahora+interval '1 day','raiz_valida_hasta',ahora+interval '1 day')::text,'UTF8'),
  convert_to(jsonb_build_object('accion',p_accion,'modulo_id','cronos','tipo_recurso',p_tipo,'finalidad',p_finalidad,
    'recurso_ref',p_recurso,'principal_id',p_actor,'perfil_activo_ref',p_perfil,
    'contexto_recurso_huella_sha256',huella,'valida_hasta',ahora+interval '30 seconds')::text,'UTF8'),
  convert_to(jsonb_build_object('principal_ref',p_actor,'perfil_activo_ref',p_perfil,'vigente_hasta',ahora+interval '1 hour',
    'vinculos',coalesce(p_vinculos,jsonb_build_array(jsonb_build_object('tipo','empleado','estado','activo','referencia',p_empleado,
      'vigente_desde',ahora-interval '1 day','vigente_hasta',ahora+interval '1 day'))))::text,'UTF8');
END $f$;
CREATE FUNCTION prueba.ahora() RETURNS text LANGUAGE sql VOLATILE AS $f$
 SELECT to_char(clock_timestamp() AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
$f$;
CREATE FUNCTION prueba.canal() RETURNS jsonb LANGUAGE sql IMMUTABLE AS $f$
 SELECT '{"politica_version_ref":"politica:canal:cronos:v1","canal_ref":"portal-empleado-web","origen_ref":"remoto","calidad_ref":"mtls-certificado"}'::jsonb
$f$;
-- Fichaje remoto como lo hace el adaptador Go: material, V3 y función durable.
CREATE FUNCTION prueba.fichar(p_empleado text,p_clave text,p_movimiento text,p_nonce text,p_vinculos jsonb DEFAULT NULL)
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE mat text; v record;
BEGIN
 mat:=jsonb_build_object('actor_ref','per_'||substr(p_empleado,5),'perfil_ref','prf_PPPPPPPPPPPPPPPPPPPPPP','empleado_ref',p_empleado,
   'clave_operacion',p_clave,'movimiento',p_movimiento,'instante_utc',prueba.ahora(),'canal',prueba.canal())::text;
 SELECT * INTO v FROM prueba.v3('vec_cronos_v1.marcaje_propio.v1','cronos.marcaje.propio.registrar','marcaje_propio','registrar_marcaje_propio',
   'marcaje:cronos:'||p_clave,mat,p_empleado,'per_'||substr(p_empleado,5),'prf_PPPPPPPPPPPPPPPPPPPPPP',p_nonce,p_vinculos);
 RETURN vec_cronos_v1.registrar_marcaje_remoto_v1(mat,v.cap,v.dcs,'\x00'::bytea,v.ctx,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
END $f$;
CREATE FUNCTION prueba.disponibilidad(p_empleado text,p_clave text,p_nonce text)
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE mat text; v record;
BEGIN
 mat:=jsonb_build_object('actor_ref','per_'||substr(p_empleado,5),'perfil_ref','prf_PPPPPPPPPPPPPPPPPPPPPP','empleado_ref',p_empleado,
   'clave_operacion',p_clave,'instante_utc',prueba.ahora(),'canal',prueba.canal())::text;
 SELECT * INTO v FROM prueba.v3('vec_cronos_v1.marcaje_remoto_disponibilidad.v1','cronos.marcaje.remoto.disponibilidad.consultar',
   'marcaje_remoto_disponibilidad','consultar_disponibilidad_marcaje_remoto','teletrabajo:cronos:'||p_empleado,mat,p_empleado,
   'per_'||substr(p_empleado,5),'prf_PPPPPPPPPPPPPPPPPPPPPP',p_nonce);
 RETURN vec_cronos_v1.consultar_estado_remoto_propio_v1(mat,v.cap,v.dcs,'\x00'::bytea,v.ctx,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
END $f$;
CREATE FUNCTION prueba.recuperar(p_empleado text,p_clave text,p_movimiento text,p_nonce text)
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE mat text; v record;
BEGIN
 mat:=jsonb_build_object('actor_ref','per_'||substr(p_empleado,5),'perfil_ref','prf_PPPPPPPPPPPPPPPPPPPPPP','empleado_ref',p_empleado,
   'clave_operacion',p_clave,'movimiento',p_movimiento,'canal',prueba.canal())::text;
 SELECT * INTO v FROM prueba.v3('vec_cronos_v1.marcaje_remoto_recibo.v1','cronos.marcaje.remoto.recibo.consultar','marcaje_remoto_recibo',
   'recuperar_recibo_marcaje_remoto','marcaje:cronos:'||p_clave,mat,p_empleado,'per_'||substr(p_empleado,5),'prf_PPPPPPPPPPPPPPPPPPPPPP',p_nonce);
 RETURN vec_cronos_v1.recuperar_marcaje_remoto_v1(mat,v.cap,v.dcs,'\x00'::bytea,v.ctx,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
END $f$;
CREATE FUNCTION prueba.saldo(p_empleado text,p_nonce text)
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE mat text; v record; hoy text:=to_char((clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date,'YYYY-MM-DD');
BEGIN
 mat:=jsonb_build_object('actor_ref','per_'||substr(p_empleado,5),'perfil_ref','prf_PPPPPPPPPPPPPPPPPPPPPP','empleado_ref',p_empleado,
   'desde',hoy,'hasta',hoy,'zona_horaria','Europe/Madrid')::text;
 SELECT * INTO v FROM prueba.v3('vec_cronos_v1.saldo_propio.consultar.v1','cronos.saldo.propio.consultar','saldo_propio','consultar_saldo_propio',
   'saldo:cronos:'||p_empleado,mat,p_empleado,'per_'||substr(p_empleado,5),'prf_PPPPPPPPPPPPPPPPPPPPPP',p_nonce);
 RETURN vec_cronos_v1.consultar_saldo_propio_v1(mat,v.cap,v.dcs,'\x00'::bytea,v.ctx,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
END $f$;
-- Intento de eludir el teletrabajo por la función de terminal de 000002.
CREATE FUNCTION prueba.fichar_por_terminal(p_empleado text,p_clave text,p_nonce text)
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE mat text; v record;
BEGIN
 mat:=jsonb_build_object('actor_ref','per_'||substr(p_empleado,5),'perfil_ref','prf_PPPPPPPPPPPPPPPPPPPPPP','empleado_ref',p_empleado,
   'clave_operacion',p_clave,'movimiento','entrada','instante_utc',prueba.ahora(),'canal',prueba.canal())::text;
 SELECT * INTO v FROM prueba.v3('vec_cronos_v1.marcaje_propio.v1','cronos.marcaje.propio.registrar','marcaje_propio','registrar_marcaje_propio',
   'marcaje:cronos:'||p_clave,mat,p_empleado,'per_'||substr(p_empleado,5),'prf_PPPPPPPPPPPPPPPPPPPPPP',p_nonce);
 RETURN vec_cronos_v1.registrar_marcaje_propio_v1(mat,v.cap,v.dcs,'\x00'::bytea,v.ctx,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
END $f$;
CREATE FUNCTION prueba.espera_error(p_sql text,p_estado text) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN
 EXECUTE p_sql;
 RAISE EXCEPTION 'se esperaba % y la sentencia terminó bien: %',p_estado,p_sql;
EXCEPTION WHEN OTHERS THEN
 IF SQLSTATE<>p_estado THEN RAISE EXCEPTION 'se esperaba % y llegó % (%)',p_estado,SQLSTATE,SQLERRM; END IF;
 RETURN 'OK '||p_estado;
END $f$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA prueba TO login_cronos_prueba;
