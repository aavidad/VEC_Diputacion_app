\set ON_ERROR_STOP on
-- Datos sintéticos y auxiliares SOLO del arnés desechable de 000009. Se
-- ejecuta después de los preparativos de 000007 y 000008 (usa prueba.v3,
-- prueba.material y los permisos del catálogo de prueba).
-- Personas: A y B empleadas; J jefatura sintética; R RRHH sintético.
-- Circuito publicado: J resuelve a A como responsable y también como RRHH
-- (para probar la separación de funciones), R resuelve a A como RRHH, J
-- tiene una asignación sobre sí mismo (para probar que no se resuelve lo
-- propio) y nadie resuelve a B.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
INSERT INTO vec_cronos_v1.permiso_resolutor VALUES
 ('resolutor:cronos:a-responsable-j','emp_AAAAAAAAAAAAAAAAAAAAAA','Persona sintética A','responsable','per_JJJJJJJJJJJJJJJJJJJJJJ',clock_timestamp()-interval '1 day',NULL,true,'fuente:sintetica:duda-47',clock_timestamp()),
 ('resolutor:cronos:a-administracion-r','emp_AAAAAAAAAAAAAAAAAAAAAA','Persona sintética A','administracion','per_RRRRRRRRRRRRRRRRRRRRRR',clock_timestamp()-interval '1 day',NULL,true,'fuente:sintetica:duda-47',clock_timestamp()),
 ('resolutor:cronos:a-administracion-j','emp_AAAAAAAAAAAAAAAAAAAAAA','Persona sintética A','administracion','per_JJJJJJJJJJJJJJJJJJJJJJ',clock_timestamp()-interval '1 day',NULL,true,'fuente:sintetica:duda-47',clock_timestamp()),
 ('resolutor:cronos:j-responsable-j','emp_JJJJJJJJJJJJJJJJJJJJJJ','Jefatura sintética','responsable','per_JJJJJJJJJJJJJJJJJJJJJJ',clock_timestamp()-interval '1 day',NULL,true,'fuente:sintetica:duda-47',clock_timestamp()),
 ('resolutor:cronos:a-futura-r','emp_AAAAAAAAAAAAAAAAAAAAAA','Persona sintética A','responsable','per_RRRRRRRRRRRRRRRRRRRRRR',clock_timestamp()+interval '10 days',NULL,true,'fuente:sintetica:duda-47',clock_timestamp());
COMMIT;

-- Material V3 de prueba con la huella exacta del recurso de quien resuelve
-- ({paso_resolucion, persona_ref}), escrita aquí de forma independiente de
-- la función de la migración.
CREATE FUNCTION prueba.v3r(p_audiencia text,p_accion text,p_tipo text,p_finalidad text,p_recurso text,
  p_material text,p_propio text,p_actor text,p_paso text,p_nonce text)
RETURNS TABLE(cap bytea,dcs bytea,ctx bytea) LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE huella text; ahora timestamptz:=clock_timestamp();
BEGIN
 huella:=encode(sha256(convert_to('{"ambitos":{"paso_resolucion":"'||p_paso||'","persona_ref":"'||p_actor||'"},"atributos":{"material_sha256":"'||
   encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}','UTF8')),'hex');
 RETURN QUERY SELECT
  convert_to(jsonb_build_object('nonce',p_nonce,'decision_ref','decision:prueba:'||p_nonce,'efecto_ref',p_recurso,
    'huella_efecto_sha256',huella,'audiencia_consumo',p_audiencia,'operacion',p_accion,
    'expira_en',ahora+interval '30 seconds','decision_valida_hasta',ahora+interval '30 seconds',
    'configuracion_expira_en',ahora+interval '1 day','raiz_valida_hasta',ahora+interval '1 day')::text,'UTF8'),
  convert_to(jsonb_build_object('accion',p_accion,'modulo_id','cronos','tipo_recurso',p_tipo,'finalidad',p_finalidad,
    'recurso_ref',p_recurso,'principal_id',p_actor,'perfil_activo_ref','prf_QQQQQQQQQQQQQQQQQQQQQQ',
    'contexto_recurso_huella_sha256',huella,'valida_hasta',ahora+interval '30 seconds')::text,'UTF8'),
  convert_to(jsonb_build_object('principal_ref',p_actor,'perfil_activo_ref','prf_QQQQQQQQQQQQQQQQQQQQQQ','vigente_hasta',ahora+interval '1 hour',
    'vinculos',jsonb_build_array(jsonb_build_object('tipo','empleado','estado','activo','referencia',p_propio,
      'vigente_desde',ahora-interval '1 day','vigente_hasta',ahora+interval '1 day')))::text,'UTF8');
END $f$;
-- p_quien: J o R. Su empleado propio es emp_ con la misma letra.
CREATE FUNCTION prueba.bandeja(p_quien text,p_paso text,p_nonce text)
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE mat text; v record; actor text:='per_'||repeat(p_quien,22); propio text:='emp_'||repeat(p_quien,22);
BEGIN
 mat:=prueba.material('actor_ref',actor,'perfil_ref','prf_QQQQQQQQQQQQQQQQQQQQQQ','empleado_ref',propio,'paso',p_paso,'zona_horaria','Europe/Madrid');
 SELECT * INTO v FROM prueba.v3r('vec_cronos_v1.permisos_bandeja.consultar.v1','cronos.permisos.bandeja.consultar','bandeja_permisos',
   'consultar_bandeja_permisos','bandeja:cronos:permisos:'||p_paso,mat,propio,actor,p_paso,p_nonce);
 RETURN vec_cronos_v1.consultar_bandeja_permisos_v1(mat,v.cap,v.dcs,'\x00'::bytea,v.ctx,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
END $f$;
CREATE FUNCTION prueba.resolver(p_quien text,p_clave text,p_solicitud text,p_paso text,p_decision text,p_motivo text,p_version int,p_nonce text)
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE mat text; v record; actor text:='per_'||repeat(p_quien,22); propio text:='emp_'||repeat(p_quien,22);
BEGIN
 mat:=prueba.material('actor_ref',actor,'perfil_ref','prf_QQQQQQQQQQQQQQQQQQQQQQ','empleado_ref',propio,'clave_operacion',p_clave,
   'solicitud_ref','permiso:cronos:solicitud:'||p_solicitud,'paso',p_paso,'decision',p_decision,'motivo',p_motivo,
   'version_esperada',p_version::text,'zona_horaria','Europe/Madrid');
 SELECT * INTO v FROM prueba.v3r('vec_cronos_v1.permiso.resolver.v1','cronos.permiso.resolver','resolucion_permiso',
   'resolver_permiso','permiso:cronos:resolucion:'||p_clave,mat,propio,actor,p_paso,p_nonce);
 RETURN vec_cronos_v1.resolver_permiso_v1(mat,v.cap,v.dcs,'\x00'::bytea,v.ctx,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
END $f$;
CREATE FUNCTION prueba.avisos(p_empleado text,p_nonce text)
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE mat text; v record;
BEGIN
 mat:=prueba.material('actor_ref','per_'||substr(p_empleado,5),'perfil_ref','prf_PPPPPPPPPPPPPPPPPPPPPP','empleado_ref',p_empleado,'zona_horaria','Europe/Madrid');
 SELECT * INTO v FROM prueba.v3('vec_cronos_v1.avisos_propio.consultar.v1','cronos.avisos.propio.consultar','avisos_propio',
   'consultar_avisos_propio','avisos:cronos:'||p_empleado,mat,p_empleado,'per_'||substr(p_empleado,5),'prf_PPPPPPPPPPPPPPPPPPPPPP',p_nonce);
 RETURN vec_cronos_v1.consultar_avisos_propio_v1(mat,v.cap,v.dcs,'\x00'::bytea,v.ctx,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
END $f$;
CREATE FUNCTION prueba.archivar(p_empleado text,p_clave text,p_aviso text,p_nonce text)
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE mat text; v record;
BEGIN
 mat:=prueba.material('actor_ref','per_'||substr(p_empleado,5),'perfil_ref','prf_PPPPPPPPPPPPPPPPPPPPPP','empleado_ref',p_empleado,
   'clave_operacion',p_clave,'aviso_ref',p_aviso,'zona_horaria','Europe/Madrid');
 SELECT * INTO v FROM prueba.v3('vec_cronos_v1.aviso_propio.archivar.v1','cronos.aviso.propio.archivar','archivo_aviso',
   'archivar_aviso_propio','aviso:cronos:archivo:'||p_clave,mat,p_empleado,'per_'||substr(p_empleado,5),'prf_PPPPPPPPPPPPPPPPPPPPPP',p_nonce);
 RETURN vec_cronos_v1.archivar_aviso_propio_v1(mat,v.cap,v.dcs,'\x00'::bytea,v.ctx,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
END $f$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA prueba TO login_cronos_prueba;
