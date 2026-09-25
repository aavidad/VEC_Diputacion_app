\set ON_ERROR_STOP on
-- Datos sintéticos y auxiliares SOLO del arnés desechable de 000010. Se
-- ejecuta después de los preparativos de 000007, 000008 y 000009 (usa
-- prueba.v3, prueba.v3r, prueba.material y sus personas).
-- Además de A, B, J y R: C tiene RRHH (R) pero ni jefatura ni marca vigente
-- (sólo una futura); D tiene jefatura (J), RRHH (R) y la marca expresa de
-- circuito directo de su unidad. R tiene además una asignación de
-- administración sobre sí mismo, para probar que nunca ve ni atiende lo suyo.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
INSERT INTO vec_cronos_v1.permiso_resolutor VALUES
 ('resolutor:cronos:r-administracion-r','emp_RRRRRRRRRRRRRRRRRRRRRR','RRHH sintético','administracion','per_RRRRRRRRRRRRRRRRRRRRRR',clock_timestamp()-interval '1 day',NULL,true,'fuente:sintetica:duda-47',clock_timestamp()),
 ('resolutor:cronos:c-administracion-r','emp_CCCCCCCCCCCCCCCCCCCCCC','Persona sintética C','administracion','per_RRRRRRRRRRRRRRRRRRRRRR',clock_timestamp()-interval '1 day',NULL,true,'fuente:sintetica:duda-47',clock_timestamp()),
 ('resolutor:cronos:d-administracion-r','emp_DDDDDDDDDDDDDDDDDDDDDD','Persona sintética D','administracion','per_RRRRRRRRRRRRRRRRRRRRRR',clock_timestamp()-interval '1 day',NULL,true,'fuente:sintetica:duda-47',clock_timestamp()),
 ('resolutor:cronos:d-responsable-j','emp_DDDDDDDDDDDDDDDDDDDDDD','Persona sintética D','responsable','per_JJJJJJJJJJJJJJJJJJJJJJ',clock_timestamp()-interval '1 day',NULL,true,'fuente:sintetica:duda-47',clock_timestamp());
INSERT INTO vec_cronos_v1.permiso_circuito_directo VALUES
 ('circuito:cronos:unidad-sin-jefatura:d','emp_DDDDDDDDDDDDDDDDDDDDDD','unidad:sintetica:sin-jefatura','Unidad sintética sin jefatura',clock_timestamp()-interval '1 day',NULL,true,'fuente:sintetica:duda-47',clock_timestamp()),
 ('circuito:cronos:unidad-futura:c','emp_CCCCCCCCCCCCCCCCCCCCCC','unidad:sintetica:futura','Unidad sintética futura',clock_timestamp()+interval '10 days',NULL,true,'fuente:sintetica:duda-47',clock_timestamp());
COMMIT;
-- Los tipos sintéticos (datos_sinteticos/…duda48.sql) los carga el arnés
-- antes de este fichero; aquí sólo se añade un tipo ya caducado.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
INSERT INTO vec_cronos_v1.notificacion_tipo VALUES
 ('notificacion:cronos:tipo:caducado:v1','notificacion:cronos:tipo:caducado','Tipo sintético caducado',90,clock_timestamp()-interval '10 days',clock_timestamp()-interval '1 day',true,'fuente:sintetica:duda-48',clock_timestamp());
COMMIT;

CREATE FUNCTION prueba.notificar(p_empleado text,p_clave text,p_tipo text,p_fecha date,p_texto text,p_adjunto text,p_huella text,p_nonce text)
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE mat text; v record;
BEGIN
 mat:=prueba.material('actor_ref','per_'||substr(p_empleado,5),'perfil_ref','prf_PPPPPPPPPPPPPPPPPPPPPP','empleado_ref',p_empleado,
   'clave_operacion',p_clave,'tipo_version_ref',p_tipo,'fecha_referida',to_char(p_fecha,'YYYY-MM-DD'),'texto',p_texto,
   'adjunto_ref',p_adjunto,'adjunto_sha256',p_huella,'zona_horaria','Europe/Madrid');
 SELECT * INTO v FROM prueba.v3('vec_cronos_v1.notificacion_propia.registrar.v1','cronos.notificacion.propia.registrar','notificacion_propia',
   'comunicar_incidencia_rrhh','notificacion:cronos:'||p_clave,mat,p_empleado,'per_'||substr(p_empleado,5),'prf_PPPPPPPPPPPPPPPPPPPPPP',p_nonce);
 RETURN vec_cronos_v1.registrar_notificacion_propia_v1(mat,v.cap,v.dcs,'\x00'::bytea,v.ctx,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
END $f$;
CREATE FUNCTION prueba.notificaciones(p_empleado text,p_nonce text)
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE mat text; v record;
BEGIN
 mat:=prueba.material('actor_ref','per_'||substr(p_empleado,5),'perfil_ref','prf_PPPPPPPPPPPPPPPPPPPPPP','empleado_ref',p_empleado,'zona_horaria','Europe/Madrid');
 SELECT * INTO v FROM prueba.v3('vec_cronos_v1.notificaciones_propio.consultar.v1','cronos.notificaciones.propio.consultar','notificaciones_propio',
   'consultar_notificaciones_propio','notificaciones:cronos:'||p_empleado,mat,p_empleado,'per_'||substr(p_empleado,5),'prf_PPPPPPPPPPPPPPPPPPPPPP',p_nonce);
 RETURN vec_cronos_v1.consultar_notificaciones_propio_v1(mat,v.cap,v.dcs,'\x00'::bytea,v.ctx,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
END $f$;
-- p_quien: J o R. Su empleado propio es emp_ con la misma letra.
CREATE FUNCTION prueba.bandeja_notificaciones(p_quien text,p_nonce text)
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE mat text; v record; actor text:='per_'||repeat(p_quien,22); propio text:='emp_'||repeat(p_quien,22);
BEGIN
 mat:=prueba.material('actor_ref',actor,'perfil_ref','prf_QQQQQQQQQQQQQQQQQQQQQQ','empleado_ref',propio,'zona_horaria','Europe/Madrid');
 SELECT * INTO v FROM prueba.v3r('vec_cronos_v1.notificaciones_bandeja.consultar.v1','cronos.notificaciones.bandeja.consultar','bandeja_notificaciones',
   'consultar_bandeja_notificaciones','bandeja:cronos:notificaciones',mat,propio,actor,'administracion',p_nonce);
 RETURN vec_cronos_v1.consultar_bandeja_notificaciones_v1(mat,v.cap,v.dcs,'\x00'::bytea,v.ctx,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
END $f$;
CREATE FUNCTION prueba.atender(p_quien text,p_clave text,p_notificacion text,p_nonce text)
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE mat text; v record; actor text:='per_'||repeat(p_quien,22); propio text:='emp_'||repeat(p_quien,22);
BEGIN
 mat:=prueba.material('actor_ref',actor,'perfil_ref','prf_QQQQQQQQQQQQQQQQQQQQQQ','empleado_ref',propio,'clave_operacion',p_clave,
   'notificacion_ref',p_notificacion,'zona_horaria','Europe/Madrid');
 SELECT * INTO v FROM prueba.v3r('vec_cronos_v1.notificacion.atender.v1','cronos.notificacion.atender','atencion_notificacion',
   'atender_notificacion','notificacion:cronos:atencion:'||p_clave,mat,propio,actor,'administracion',p_nonce);
 RETURN vec_cronos_v1.atender_notificacion_v1(mat,v.cap,v.dcs,'\x00'::bytea,v.ctx,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
END $f$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA prueba TO login_cronos_prueba;
