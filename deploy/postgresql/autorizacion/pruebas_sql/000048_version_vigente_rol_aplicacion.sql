\set ON_ERROR_STOP on
-- Prueba estructural AUT48: no crea asignaciones ni decisiones favorables.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
DO $p$
DECLARE f record;s text;a record;
BEGIN
 FOR f IN SELECT p.oid,p.proname,p.prosrc FROM pg_proc p WHERE p.pronamespace='vec_autorizacion'::regnamespace AND p.proname LIKE '%\_vigente\_aut48' LOOP
  IF f.prosrc ~ 'administracion_perfiles:v[0-9]' OR f.prosrc ~ 'version=[0-9]' OR f.prosrc ~ 'fuente_version=[0-9]'
  THEN RAISE EXCEPTION 'AUT48: versión fija en %',f.proname; END IF;
  IF strpos(f.prosrc,'^rol:administracion_perfiles:v[1-9][0-9]{0,8}$')=0 THEN RAISE EXCEPTION 'AUT48: sin acotar al rol de Aplicación %',f.proname; END IF;
  IF f.proname<>'acreditar_ambito_certificado_nominal_vigente_aut48' AND (strpos(f.prosrc,'fuente_version=r.version')=0 OR strpos(f.prosrc,'concesion')=0 OR strpos(f.prosrc,'habilitada')=0)
  THEN RAISE EXCEPTION 'AUT48: comprobación genérica incompleta %',f.proname; END IF;
  IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f.oid AND x.grantee<>p.proowner)
  THEN RAISE EXCEPTION 'AUT48: ACL abierta %',f.proname; END IF;
 END LOOP;
 IF (SELECT count(*) FROM pg_proc p WHERE p.pronamespace='vec_autorizacion'::regnamespace AND p.proname LIKE '%\_vigente\_aut48')<>4 THEN RAISE EXCEPTION 'AUT48: faltan genéricas'; END IF;
 FOREACH s IN ARRAY ARRAY['acreditar_perfil_aplicacion_nominal_v1','acreditar_ambito_certificado_nominal_v1','validar_administrador_denominacion_persona_v1','validar_administrador_usuarios_v1'] LOOP
  IF (SELECT prosrc FROM pg_proc WHERE pronamespace='vec_autorizacion'::regnamespace AND proname=s) !~ ('vigente_aut48') 
  OR (SELECT prosrc FROM pg_proc WHERE pronamespace='vec_autorizacion'::regnamespace AND proname=s) ~ 'administracion_perfiles:v[5-9]'
  THEN RAISE EXCEPTION 'AUT48: despachador % no usa la genérica',s; END IF;
 END LOOP;
 IF strpos((SELECT prosrc FROM pg_proc WHERE pronamespace='vec_autorizacion'::regnamespace AND proname='acreditar_ambito_certificado_nominal_vigente_aut48'),'perfil_fijo_categoria_nominal_v1')=0
 THEN RAISE EXCEPTION 'AUT48: el ámbito no exige metadatos de perfil fijo de Aplicación'; END IF;
 -- Fuera del rol de Aplicación o sin runtime: la genérica deniega antes de leer.
 IF vec_autorizacion.acreditar_ambito_certificado_nominal_v1('rol:otro_rol:v7','asignacion:x','org_0123456789abcdef') IS NOT FALSE
 OR vec_autorizacion.acreditar_ambito_certificado_nominal_v1('rol:administracion_perfiles:v99','asignacion:x','org_0123456789abcdef') IS NOT FALSE
 OR vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1('rol:administracion_perfiles:v99','a','p','f','administracion.usuarios.listar','administracion','conjunto_usuarios','gestion_usuarios','[]','{}') IS NOT FALSE
 THEN RAISE EXCEPTION 'AUT48: versión inexistente aceptada'; END IF;
 -- Con datos reales, si existen: la versión actual de la asignación pasa; otra
 -- versión existente pero no actual, y v1, no.
 SELECT x.asignacion_ref,x.version_rol_ref,x.documento#>>'{ambitos,0,valores,0}' AS org INTO a
 FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
 WHERE x.version_rol_ref LIKE 'rol:administracion_perfiles:v%' AND x.documento->>'estado'='activa' AND x.documento#>>'{ambitos,0,clave}'='organizacion_ref'
 ORDER BY x.asignacion_ref LIMIT 1;
 IF FOUND THEN
  IF vec_autorizacion.acreditar_ambito_certificado_nominal_v1(a.version_rol_ref,a.asignacion_ref,a.org) IS NOT TRUE
  THEN RAISE EXCEPTION 'AUT48: la versión actual de la asignación no acredita el ámbito'; END IF;
  FOREACH s IN ARRAY ARRAY['rol:administracion_perfiles:v1','rol:administracion_perfiles:v3','rol:administracion_perfiles:v5','rol:administracion_perfiles:v6'] LOOP
   IF s<>a.version_rol_ref AND vec_autorizacion.acreditar_ambito_certificado_nominal_v1(s,a.asignacion_ref,a.org) IS NOT FALSE
   THEN RAISE EXCEPTION 'AUT48: versión no actual % aceptada',s; END IF;
  END LOOP;
 ELSE
  RAISE NOTICE 'AUT48: sin asignación de Aplicación; se omiten los casos con datos reales';
 END IF;
END $p$;
ROLLBACK;
