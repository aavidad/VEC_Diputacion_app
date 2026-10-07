\set ON_ERROR_STOP on
-- Solo clon preparado, después del conjunto completo. No crea historia.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='30s';
DO $prueba$
DECLARE nombre text;funcion regprocedure;rol record;control record;bloqueado boolean;metadato text;
BEGIN
 IF (SELECT count(*) FROM vec_autorizacion.version_rol WHERE rol_id='administracion_perfiles')<>3
 OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil a JOIN vec_autorizacion.version_rol r USING(version_rol_ref) WHERE r.rol_id='administracion_perfiles')
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.control_continuidad_admin WHERE control_id AND revision=1 AND bootstrap_estado='pendiente')
 OR EXISTS(SELECT 1 FROM vec_autorizacion.bootstrap_admin_v2)
 THEN RAISE EXCEPTION 'ADMIN: instalación alteró la población o bootstrap'; END IF;

 SELECT * INTO STRICT rol FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v3';
 SELECT * INTO STRICT control FROM vec_autorizacion.control_vigencia_version_rol WHERE version_rol_ref=rol.version_rol_ref AND revision=1;
 IF rol.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(rol.documento),'UTF8')),'hex')
 OR control.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(control.documento),'UTF8')),'hex')
 THEN RAISE EXCEPTION 'ADMIN: publicación incompatible con canon central'; END IF;

 FOREACH nombre IN ARRAY ARRAY['aplicar_acto_ordinario_admin_v1','proponer_acto_admin_v1','cerrar_propuesta_admin_v1','preparar_preimagen_admin_v1'] LOOP
  funcion:=to_regprocedure('vec_autorizacion.'||nombre||'(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
  IF funcion IS NULL OR pg_catalog.has_function_privilege('vec_admin_perfiles_ejecutor',funcion,'EXECUTE')
  OR pg_catalog.has_function_privilege('vec_admin_perfiles_bootstrap_ejecutor',funcion,'EXECUTE')
  THEN RAISE EXCEPTION 'ADMIN: fachada web abierta antes del consumidor %',nombre; END IF;
 END LOOP;

 FOREACH nombre IN ARRAY ARRAY['registro_acto_admin_v1','bootstrap_admin_v2','sello_efecto_admin_tx_v1','rol_administrable_exacto_v1'] LOOP
  IF pg_catalog.has_table_privilege('vec_admin_perfiles_ejecutor','vec_autorizacion.'||nombre,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
  OR pg_catalog.has_table_privilege('vec_admin_perfiles_bootstrap_ejecutor','vec_autorizacion.'||nombre,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
  THEN RAISE EXCEPTION 'ADMIN: grupo técnico con datos o DML %',nombre; END IF;
 END LOOP;

 FOREACH metadato IN ARRAY ARRAY['cuenta_ref','vinculo_ref','rol_huella_sha256','referencia_acto'] LOOP
  bloqueado:=false;
  BEGIN
   PERFORM vec_autorizacion.canon_asignacion_perfil_admin_v1(jsonb_build_object(metadato,'referencia:sintetica'));
  EXCEPTION WHEN invalid_parameter_value THEN bloqueado:=true;
  END;
  IF NOT bloqueado THEN RAISE EXCEPTION 'ADMIN: canon aceptó metadata ajena %',metadato; END IF;
 END LOOP;

 bloqueado:=false;
 BEGIN
  PERFORM vec_autorizacion.fecha_canonica_go_admin_v1('2026-10-03T12:00:00.1234567Z');
 EXCEPTION WHEN invalid_parameter_value THEN bloqueado:=true;
 END;
 IF NOT bloqueado THEN RAISE EXCEPTION 'ADMIN: canon redondeó precisión submicrosegundo'; END IF;

 bloqueado:=false;
 BEGIN
  PERFORM vec_autorizacion.provisionar_dos_administradores_iniciales_v2('{}',repeat('a',64));
 EXCEPTION WHEN insufficient_privilege THEN bloqueado:=true;
 END;
 IF NOT bloqueado THEN RAISE EXCEPTION 'ADMIN: bootstrap sin operador segregado'; END IF;
END $prueba$;
ROLLBACK;
