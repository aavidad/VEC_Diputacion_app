\set ON_ERROR_STOP on
-- Comprobación de la publicación prospectiva; no concede asignaciones.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path=pg_catalog;
DO $comprobar$
DECLARE v record;vieja record;m record;n integer;
BEGIN
 SELECT * INTO STRICT vieja FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v3';
 SELECT * INTO STRICT v FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v4';
 SELECT * INTO STRICT m FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1
 WHERE version_rol_ref=v.version_rol_ref;
 IF vieja.documento->>'estado'<>'publicada' OR v.rol_id<>'administracion_perfiles' OR v.version<>4
  OR m.categoria_administrativa<>'aplicacion' OR m.tipo_perfil<>'fijo_sistema'
  OR m.version_rol_huella_sha256<>v.huella_sha256
  OR v.huella_sha256<>pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_version_rol_admin_v1(v.documento),'UTF8')),'hex')
  OR (SELECT count(*) FROM vec_autorizacion.asignacion_perfil WHERE version_rol_ref=v.version_rol_ref)<>0
  OR EXISTS(SELECT 1 FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1
    WHERE version_rol_ref IN ('rol:administracion_perfiles:v1','rol:administracion_perfiles:v2','rol:administracion_perfiles:v3'))
 THEN RAISE EXCEPTION 'AUT33: perfil fijo o historia divergente'; END IF;
 SELECT count(*) INTO n FROM pg_catalog.jsonb_array_elements(v.documento->'concesiones') c
 WHERE c->>'accion' IN ('administracion.certificados.nominal.publicar','administracion.certificados.nominal.retirar','personal.cargo_competencial.publicar');
 IF n<>3 OR pg_catalog.jsonb_array_length(v.documento->'concesiones')<>pg_catalog.jsonb_array_length(vieja.documento->'concesiones')+3
  OR (SELECT count(*) FROM vec_autorizacion.catalogo_accion_nominal_v1 WHERE version_rol_ref=v.version_rol_ref)<>3
  OR EXISTS(SELECT 1 FROM vec_autorizacion.catalogo_accion_nominal_v1 e
   WHERE e.version_rol_ref=v.version_rol_ref AND
    (e.fuente_ref<>v.version_rol_ref OR e.fuente_version<>4 OR e.fuente_huella_sha256<>v.huella_sha256
     OR e.dimensiones_ambito<>CASE WHEN e.concesion->>'modulo_id'='administracion'
       THEN '["organizacion_ref"]'::jsonb ELSE '["organizacion_ref","unidad_ref"]'::jsonb END
     OR e.clase_control<>'administrador_aplicacion'
     OR NOT EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(v.documento->'concesiones') c WHERE c=e.concesion)))
 THEN RAISE EXCEPTION 'AUT33: concesiones nuevas o previas divergentes'; END IF;
 IF vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(v.version_rol_ref,'ausente','ausente','ausente',
   'administracion.certificados.nominal.publicar','administracion','vinculo_certificado_nominal','gestionar_certificados_firmantes','[]'::jsonb,'{}'::jsonb)
 THEN RAISE EXCEPTION 'AUT33: perfil no asignado obtuvo permiso'; END IF;
 IF vec_autorizacion.destino_no_administrador_certificado_nominal_v1('cta_ausente_sintetica_aaaaaaaa','per_ausente_sintetica_bbbbbbbb')
 THEN RAISE EXCEPTION 'AUT33: cuenta IS ausente tratada como ordinaria'; END IF;
 IF vec_autorizacion.acreditar_ambito_certificado_nominal_v1(v.version_rol_ref,'asignacion:ausente:v1','org_ca25sinteticaaaaaaaaaaaaa')
 THEN RAISE EXCEPTION 'AUT33: ámbito de asignación ausente aceptado'; END IF;
END $comprobar$;
RESET ROLE;
DO $acl$
BEGIN
 IF has_function_privilege('vec_autorizacion_fuente',
   'vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)','EXECUTE')
  OR NOT has_function_privilege('vec_autorizacion_atestada_v3_propietario',
   'vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)','EXECUTE')
 THEN RAISE EXCEPTION 'AUT33: ACL nominal divergente'; END IF;
END $acl$;
ROLLBACK;
