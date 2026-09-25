\set ON_ERROR_STOP on
-- Datos sintéticos y auxiliares SOLO del arnés desechable de 000008. Se
-- ejecuta después de cronos_empleado_000007_preparar.sql (usa prueba.v3).
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
-- Semana de prueba: primer lunes desde el 2 de marzo del año en curso, con
-- un festivo el miércoles. Calendario asignado sólo a la persona A.
CREATE TEMP TABLE semana ON COMMIT DROP AS
 SELECT d::date lunes FROM (SELECT make_date(extract(year FROM current_date)::int,3,2)+((8-extract(isodow FROM make_date(extract(year FROM current_date)::int,3,2))::int)%7) d) q;
INSERT INTO vec_cronos_v1.calendario_dia SELECT 'calendario:cronos:sintetico-granada',lunes+2,'festivo','Festivo sintético','fuente:sintetica:calendario',clock_timestamp() FROM semana;
INSERT INTO vec_cronos_v1.calendario_dia VALUES
 ('calendario:cronos:sintetico-granada',make_date(extract(year FROM current_date)::int,1,1),'festivo','Año nuevo sintético','fuente:sintetica:calendario',clock_timestamp());
SELECT set_config('vec.cronos.empleado_ref','emp_AAAAAAAAAAAAAAAAAAAAAA',true);
INSERT INTO vec_cronos_v1.calendario_asignacion VALUES
 ('emp_AAAAAAAAAAAAAAAAAAAAAA',extract(year FROM current_date)::int,'calendario:cronos:sintetico-granada','fuente:sintetica:calendario',clock_timestamp());
INSERT INTO vec_cronos_v1.permiso_catalogo VALUES
 ('catalogo:cronos:asuntos-propios:v1','permiso:cronos:asuntos-propios','Asuntos propios',10,clock_timestamp()-interval '400 days',NULL,'dia','laborables','A',1,NULL,NULL,6,false,true,true,'fuente:sintetica:duda-41',clock_timestamp()),
 ('catalogo:cronos:vacaciones:v1','permiso:cronos:vacaciones','Vacaciones',20,clock_timestamp()-interval '400 days',NULL,'dia','laborables','J-A',1,NULL,NULL,22,false,true,true,'fuente:sintetica:duda-41',clock_timestamp()),
 ('catalogo:cronos:horas-medico:v1','permiso:cronos:horas-medico','Horas de médico',30,clock_timestamp()-interval '400 days',NULL,'hora','laborables','J-A',15,NULL,NULL,180,true,true,true,'fuente:sintetica:duda-41',clock_timestamp()),
 ('catalogo:cronos:traslado:v1','permiso:cronos:traslado','Traslado de domicilio',40,clock_timestamp()-interval '400 days',NULL,'dia','naturales','J-A',1,1,NULL,1,true,true,true,'fuente:sintetica:duda-41',clock_timestamp()),
 ('catalogo:cronos:maternidad:v1','permiso:cronos:maternidad','Permiso por nacimiento',50,clock_timestamp()-interval '400 days',NULL,'dia','naturales','A',1,NULL,NULL,112,false,false,true,'fuente:sintetica:duda-41',clock_timestamp());
-- Un permiso concedido pendiente de justificar, escrito como lo hará el
-- corte de concesión (fuera de este arnés no hay otra vía).
INSERT INTO vec_cronos_v1.permiso_solicitud VALUES
 ('permiso:cronos:solicitud:concedido-0001','emp_AAAAAAAAAAAAAAAAAAAAAA','concedido-0001','per_AAAAAAAAAAAAAAAAAAAAAA','prf_PPPPPPPPPPPPPPPPPPPPPP',
  'catalogo:cronos:traslado:v1','permiso:cronos:traslado',current_date-3,current_date-3,NULL,NULL,1,'dia',repeat('a',64),
  'decision:sintetica:c1','auditoria:sintetica:c1',repeat('b',64),clock_timestamp());
INSERT INTO vec_cronos_v1.permiso_estado VALUES
 ('permiso:estado:00000000-0000-4000-8000-000000000001','permiso:cronos:solicitud:concedido-0001','emp_AAAAAAAAAAAAAAAAAAAAAA',1,'solicitado',false,'recibo:cronos:00000000-0000-4000-8000-000000000001',clock_timestamp()),
 ('permiso:estado:00000000-0000-4000-8000-000000000002','permiso:cronos:solicitud:concedido-0001','emp_AAAAAAAAAAAAAAAAAAAAAA',2,'concedido',true,'recibo:cronos:00000000-0000-4000-8000-000000000002',clock_timestamp());
COMMIT;

CREATE FUNCTION prueba.semana() RETURNS date LANGUAGE sql STABLE AS $f$
 SELECT make_date(extract(year FROM current_date)::int,3,2)+((8-extract(isodow FROM make_date(extract(year FROM current_date)::int,3,2))::int)%7)
$f$;
CREATE FUNCTION prueba.material(VARIADIC pares text[]) RETURNS text LANGUAGE sql IMMUTABLE AS $f$
 SELECT jsonb_object(pares)::text
$f$;
CREATE FUNCTION prueba.movimientos(p_empleado text,p_desde date,p_hasta date,p_nonce text)
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE mat text; v record;
BEGIN
 mat:=prueba.material('actor_ref','per_'||substr(p_empleado,5),'perfil_ref','prf_PPPPPPPPPPPPPPPPPPPPPP','empleado_ref',p_empleado,
   'desde',to_char(p_desde,'YYYY-MM-DD'),'hasta',to_char(p_hasta,'YYYY-MM-DD'),'zona_horaria','Europe/Madrid');
 SELECT * INTO v FROM prueba.v3('vec_cronos_v1.movimientos_propio.consultar.v1','cronos.movimientos.propio.consultar','movimientos_propio',
   'consultar_movimientos_propio','movimientos:cronos:'||p_empleado,mat,p_empleado,'per_'||substr(p_empleado,5),'prf_PPPPPPPPPPPPPPPPPPPPPP',p_nonce);
 RETURN vec_cronos_v1.consultar_movimientos_propio_v1(mat,v.cap,v.dcs,'\x00'::bytea,v.ctx,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
END $f$;
CREATE FUNCTION prueba.corregir(p_empleado text,p_clave text,p_fecha date,p_hora text,p_nonce text,p_original text DEFAULT '')
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE mat text; v record;
BEGIN
 mat:=prueba.material('actor_ref','per_'||substr(p_empleado,5),'perfil_ref','prf_PPPPPPPPPPPPPPPPPPPPPP','empleado_ref',p_empleado,
   'clave_operacion',p_clave,'marcaje_original_ref',p_original,'movimiento','entrada','fecha_civil',to_char(p_fecha,'YYYY-MM-DD'),
   'hora_pretendida',p_hora,'zona_horaria','Europe/Madrid');
 SELECT * INTO v FROM prueba.v3('vec_cronos_v1.correccion_propia.solicitar.v1','cronos.correccion.solicitar','correccion_marcaje',
   'solicitar_correccion_marcaje','correccion:cronos:'||p_clave,mat,p_empleado,'per_'||substr(p_empleado,5),'prf_PPPPPPPPPPPPPPPPPPPPPP',p_nonce);
 RETURN vec_cronos_v1.solicitar_correccion_propia_v1(mat,v.cap,v.dcs,'\x00'::bytea,v.ctx,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
END $f$;
CREATE FUNCTION prueba.permisos(p_empleado text,p_nonce text)
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE mat text; v record;
BEGIN
 mat:=prueba.material('actor_ref','per_'||substr(p_empleado,5),'perfil_ref','prf_PPPPPPPPPPPPPPPPPPPPPP','empleado_ref',p_empleado,
   'anio',extract(year FROM current_date)::int::text,'zona_horaria','Europe/Madrid');
 SELECT * INTO v FROM prueba.v3('vec_cronos_v1.permisos_propio.consultar.v1','cronos.permisos.propio.consultar','permisos_propio',
   'consultar_permisos_propio','permisos:cronos:'||p_empleado,mat,p_empleado,'per_'||substr(p_empleado,5),'prf_PPPPPPPPPPPPPPPPPPPPPP',p_nonce);
 RETURN vec_cronos_v1.consultar_permisos_propio_v1(mat,v.cap,v.dcs,'\x00'::bytea,v.ctx,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
END $f$;
CREATE FUNCTION prueba.pedir(p_empleado text,p_clave text,p_permiso text,p_desde date,p_hasta date,p_hi text,p_hf text,p_nonce text)
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE mat text; v record;
BEGIN
 mat:=prueba.material('actor_ref','per_'||substr(p_empleado,5),'perfil_ref','prf_PPPPPPPPPPPPPPPPPPPPPP','empleado_ref',p_empleado,
   'clave_operacion',p_clave,'permiso_ref','permiso:cronos:'||p_permiso,'desde',to_char(p_desde,'YYYY-MM-DD'),'hasta',to_char(p_hasta,'YYYY-MM-DD'),
   'hora_inicio',p_hi,'hora_fin',p_hf,'zona_horaria','Europe/Madrid');
 SELECT * INTO v FROM prueba.v3('vec_cronos_v1.permiso_propio.solicitar.v1','cronos.permiso.solicitar','solicitud_permiso',
   'solicitar_permiso_propio','permiso:cronos:solicitud:'||p_clave,mat,p_empleado,'per_'||substr(p_empleado,5),'prf_PPPPPPPPPPPPPPPPPPPPPP',p_nonce);
 RETURN vec_cronos_v1.solicitar_permiso_propio_v1(mat,v.cap,v.dcs,'\x00'::bytea,v.ctx,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
END $f$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA prueba TO login_cronos_prueba;
