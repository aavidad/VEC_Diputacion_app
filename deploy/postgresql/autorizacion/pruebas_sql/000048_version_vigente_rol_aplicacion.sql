\set ON_ERROR_STOP on
-- Prueba estructural AUT48: no crea asignaciones ni decisiones favorables.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
DO $p$
DECLARE f record;s text;
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
 -- Fuera del rol de Aplicación o sin runtime: la genérica deniega antes de leer.
 IF vec_autorizacion.acreditar_ambito_certificado_nominal_v1('rol:otro_rol:v7','asignacion:x','org_0123456789abcdef') IS NOT FALSE
 OR vec_autorizacion.acreditar_ambito_certificado_nominal_v1('rol:administracion_perfiles:v99','asignacion:x','org_0123456789abcdef') IS NOT FALSE
 OR vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1('rol:administracion_perfiles:v99','a','p','f','administracion.usuarios.listar','administracion','conjunto_usuarios','gestion_usuarios','[]','{}') IS NOT FALSE
 THEN RAISE EXCEPTION 'AUT48: versión inexistente aceptada'; END IF;
END $p$;
ROLLBACK;
