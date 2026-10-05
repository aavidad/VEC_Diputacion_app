\set ON_ERROR_STOP on
-- AUT48. Las comprobaciones nominales del perfil fijo de Aplicación dejan de
-- fijar la versión del rol en el código. Antes, cada versión nueva (AUT42 v5,
-- AUT45 v6) obligaba a copiar cuatro funciones y a añadir una rama en cuatro
-- despachadores; una v7 con otra concesión habría dejado sin acreditar la
-- lectura de usuarios, la denominación, los certificados y el cargo competencial.
-- La versión aceptada es la que señala la asignación ACTUAL del perfil, con:
-- rol administracion_perfiles publicado y con huella canónica, control de
-- vigencia habilitado con huella, metadatos de perfil fijo de Aplicación ligados
-- a esa versión, entrada de catálogo de esa versión con la concesión exacta y la
-- misma concesión dentro del documento del rol. No hay «>= vN»: sin la concesión
-- exacta en esa versión, se deniega. Las funciones por versión siguen instaladas.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000048',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
DECLARE caso record;actual text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AUT48: PARO clave=operador actual=incompatible esperado=superusuario_PG18' USING ERRCODE='55000';END IF;
 -- Despachadores exactos de AUT45 (medidos en copia fría H10-30 con AUT45 aplicada).
 FOR caso IN SELECT * FROM (VALUES
  ('vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)','7c0b82cfce3ea461472ce9be0c31f39231c381bbfa57cc5d09b872c0a95a878b'),
  ('vec_autorizacion.acreditar_ambito_certificado_nominal_v1(text,text,text)','7c2e67e3add32763c52e8dca5ceeb20575085a2d89666348eec6ba220943e00b'),
  ('vec_autorizacion.validar_administrador_denominacion_persona_v1(jsonb,jsonb)','12b8c47cbaeb5143118ce064501806554a7ab7682f7058fe7bee319804616176'),
  ('vec_autorizacion.validar_administrador_usuarios_v1(jsonb,jsonb)','b094926dd379be4449ec28729b4173552ab2037f92510b9ba59a367562461351')) v(firma,def) LOOP
  IF to_regprocedure(caso.firma) IS NULL THEN RAISE EXCEPTION 'AUT48: PARO clave=despachador actual=ausente esperado=%',caso.firma USING ERRCODE='55000';END IF;
  actual:=encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(caso.firma)),'UTF8')),'hex');
  IF actual IS DISTINCT FROM caso.def THEN RAISE EXCEPTION 'AUT48: PARO clave=despachador_% actual=% esperado=%',split_part(caso.firma,'(',1),actual,caso.def USING ERRCODE='55000';END IF;
 END LOOP;
 IF to_regprocedure('vec_autorizacion.acreditar_perfil_aplicacion_nominal_v4_preservada_aut42(text,text,text,text,text,text,text,text,jsonb,jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion.acreditar_ambito_certificado_nominal_v4_preservada_aut42(text,text,text)') IS NULL
 OR to_regprocedure('vec_autorizacion.validar_administrador_usuarios_vigente_aut48(jsonb,jsonb)') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT48: PARO clave=preimagen actual=incompatible esperado=AUT45_sin_AUT48' USING ERRCODE='55000';END IF;
END $pre$;
CREATE TEMPORARY TABLE aut48_meta ON COMMIT DROP AS
 SELECT p.oid, to_jsonb(p)-'prosrc' AS meta FROM pg_proc p WHERE p.oid IN(
  'vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)'::regprocedure,
  'vec_autorizacion.acreditar_ambito_certificado_nominal_v1(text,text,text)'::regprocedure,
  'vec_autorizacion.validar_administrador_denominacion_persona_v1(jsonb,jsonb)'::regprocedure,
  'vec_autorizacion.validar_administrador_usuarios_v1(jsonb,jsonb)'::regprocedure);
SET LOCAL ROLE vec_autorizacion_propietario;

-- Comprobaciones genéricas: cuerpo de las funciones v6 de AUT45 cambiando sólo
-- la versión fija por la de la decisión y la fuente de metadatos y catálogo
-- por r.version. El número de versión de la entrada de catálogo no se fija:
-- (version_rol_ref, accion_ref) es único y se comprueban su fuente y su huella.
CREATE FUNCTION vec_autorizacion.acreditar_perfil_aplicacion_nominal_vigente_aut48(version_ref text, p_asignacion_ref text, principal_ref text, perfil_ref text, accion text, modulo text, tipo text, finalidad text, campos jsonb, autenticacion jsonb)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog'
AS $function$
DECLARE r record;ct record;asig record;meta record;ahora timestamptz;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR (version_ref ~ '^rol:administracion_perfiles:v[1-9][0-9]{0,8}$') IS NOT TRUE
  OR accion IS NULL OR modulo IS NULL OR tipo IS NULL OR finalidad IS NULL
  OR campos IS NULL OR pg_catalog.jsonb_typeof(campos)<>'array'
  OR pg_catalog.jsonb_typeof(autenticacion)<>'object'
  OR vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(autenticacion) IS NOT TRUE
  THEN RETURN false; END IF;
 SELECT v.* INTO r FROM vec_autorizacion.version_rol v WHERE v.version_rol_ref=version_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT c.* INTO ct FROM vec_autorizacion.control_vigencia_version_rol_actual a
 JOIN vec_autorizacion.control_vigencia_version_rol c USING(version_rol_ref,revision)
 WHERE a.version_rol_ref=version_ref FOR SHARE OF a,c;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO asig FROM vec_autorizacion.asignacion_perfil_actual a
 JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
 WHERE a.perfil_activo_ref=perfil_ref AND a.asignacion_ref=p_asignacion_ref FOR SHARE OF a,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO meta FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 x WHERE x.version_rol_ref=version_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 ahora:=pg_catalog.clock_timestamp();
 RETURN r.rol_id='administracion_perfiles' AND r.version_rol_ref='rol:administracion_perfiles:v'||r.version::text AND r.documento->>'estado'='publicada'
  AND r.huella_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
  AND ct.estado='habilitada' AND ct.huella_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_control_rol_admin_v1(ct.documento),'UTF8')),'hex')
  AND meta.version_rol_huella_sha256=r.huella_sha256 AND meta.categoria_administrativa='aplicacion' AND meta.tipo_perfil='fijo_sistema'
  AND meta.fuente_ref=version_ref AND meta.fuente_version=r.version AND meta.fuente_huella_sha256=r.huella_sha256
  AND asig.version_rol_ref=version_ref AND asig.principal_id=principal_ref AND asig.perfil_activo_ref=perfil_ref
  AND asig.documento->>'estado'='activa'
  AND ahora>=(asig.documento->>'vigente_desde')::timestamptz
  AND ahora<(asig.documento->>'vigente_hasta')::timestamptz
  AND EXISTS(SELECT 1 FROM vec_autorizacion.catalogo_accion_nominal_v1 catalogo
    WHERE catalogo.version_rol_ref=version_ref AND catalogo.accion_ref='accion:'||accion
     AND catalogo.fuente_ref=version_ref AND catalogo.fuente_version=r.version
     AND catalogo.fuente_huella_sha256=r.huella_sha256
     AND catalogo.clase_control='administrador_aplicacion'
     AND catalogo.dimensiones_ambito=CASE WHEN modulo='administracion' THEN '["organizacion_ref"]'::jsonb
       ELSE '["organizacion_ref","unidad_ref"]'::jsonb END
     AND ahora>=catalogo.vigente_desde AND (catalogo.vigente_hasta IS NULL OR ahora<catalogo.vigente_hasta)
     AND catalogo.concesion->>'accion'=accion AND catalogo.concesion->>'modulo_id'=modulo
     AND catalogo.concesion->>'tipo_recurso'=tipo
     AND catalogo.concesion->'finalidades'=pg_catalog.jsonb_build_array(finalidad)
     AND catalogo.concesion->>'garantia_minima'='alto'
     AND COALESCE(catalogo.concesion->'campos_permitidos','[]'::jsonb)=campos
     AND COALESCE(catalogo.concesion->'obligaciones','[]'::jsonb)='[]'::jsonb)
  AND EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(r.documento->'concesiones') c
   WHERE c->>'accion'=accion AND c->>'modulo_id'=modulo AND c->>'tipo_recurso'=tipo
    AND c->'finalidades'=pg_catalog.jsonb_build_array(finalidad) AND c->>'garantia_minima'='alto'
    AND COALESCE(c->'campos_permitidos','[]'::jsonb)=campos AND COALESCE(c->'obligaciones','[]'::jsonb)='[]'::jsonb);
END 
$function$;

CREATE FUNCTION vec_autorizacion.acreditar_ambito_certificado_nominal_vigente_aut48(version_ref text, p_asignacion_ref text, org text)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog'
AS $function$
DECLARE a record;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR (version_ref ~ '^rol:administracion_perfiles:v[1-9][0-9]{0,8}$') IS NOT TRUE
  OR org IS NULL OR org !~ '^org_[a-z0-9]{16,80}$' THEN RETURN false; END IF;
 -- Además de la v6: sólo versiones con metadatos de perfil fijo de Aplicación
 -- (v4 en adelante). Sin esto, una asignación v1-v3 pasaría la comprobación.
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol r JOIN vec_autorizacion.perfil_fijo_categoria_nominal_v1 meta
  ON meta.version_rol_ref=r.version_rol_ref AND meta.fuente_ref=r.version_rol_ref AND meta.fuente_version=r.version AND meta.version_rol_huella_sha256=r.huella_sha256
  WHERE r.version_rol_ref=version_ref AND r.rol_id='administracion_perfiles' AND r.version_rol_ref='rol:administracion_perfiles:v'||r.version::text
  AND meta.categoria_administrativa='aplicacion' AND meta.tipo_perfil='fijo_sistema') THEN RETURN false; END IF;
 SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil_actual p
 JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
 WHERE p.asignacion_ref=p_asignacion_ref FOR SHARE OF p,x;
 IF NOT FOUND OR a.version_rol_ref<>version_ref OR a.documento->>'estado'<>'activa' THEN RETURN false; END IF;
 RETURN a.documento->'ambitos' @> pg_catalog.jsonb_build_array(
  pg_catalog.jsonb_build_object('clave','organizacion_ref','valores',pg_catalog.jsonb_build_array(org)));
END 
$function$;

CREATE FUNCTION vec_autorizacion.validar_administrador_denominacion_persona_vigente_aut48(d jsonb, m jsonb)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog'
 SET row_security TO 'on'
AS $function$
DECLARE recurso jsonb;a record;r record;ct record;meta record;catalogo record;ahora timestamptz;campos jsonb;org text;unidad text;raw text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR to_regprocedure('vec_autorizacion_atestada_v3.canon_material_denominacion_persona_v1(jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.recurso_denominacion_persona_v1(text)') IS NULL
 OR (d->>'version_rol_ref' ~ '^rol:administracion_perfiles:v[1-9][0-9]{0,8}$') IS NOT TRUE OR d->>'concedida' IS DISTINCT FROM 'true'
 OR d->>'accion' NOT IN('vec.persona.denominacion.publicar','vec.persona.denominacion.leer')
 OR d->>'modulo_id' IS DISTINCT FROM 'vec' OR d->>'tipo_recurso' IS DISTINCT FROM 'persona_denominacion'
 OR d->>'finalidad' IS DISTINCT FROM 'presentacion_persona' OR d->'obligaciones' IS DISTINCT FROM '["auditar"]'::jsonb
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'administracion_privilegiada'
 OR d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'true'
 THEN RETURN false; END IF;
 campos:=CASE WHEN d->>'accion'='vec.persona.denominacion.publicar' THEN '["denominacion"]'::jsonb ELSE '["nombre_mostrar"]'::jsonb END;
 IF d->'campos_permitidos' IS DISTINCT FROM campos THEN RETURN false; END IF;
 raw:=vec_autorizacion_atestada_v3.canon_material_denominacion_persona_v1(m);
 recurso:=vec_autorizacion_atestada_v3.recurso_denominacion_persona_v1(raw);
 IF recurso->>'accion' IS DISTINCT FROM d->>'accion' OR recurso->>'recurso_ref' IS DISTINCT FROM d->>'recurso_ref'
 OR recurso->>'contexto_sha256' IS DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 OR d->>'recurso_ref' IS DISTINCT FROM m->>'persona_ref' THEN RETURN false; END IF;
 org:=m#>>'{ambitos,organizacion_ref}';unidad:=m#>>'{ambitos,unidad_ref}';
 IF org IS NULL OR unidad IS NULL OR org='' OR unidad='' THEN RETURN false; END IF;
 SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
 WHERE q.perfil_activo_ref=d->>'perfil_activo_ref' AND q.asignacion_ref=d->>'asignacion_ref' FOR SHARE OF q,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT * INTO r FROM vec_autorizacion.version_rol WHERE version_rol_ref=d->>'version_rol_ref' FOR SHARE;IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO ct FROM vec_autorizacion.control_vigencia_version_rol_actual q JOIN vec_autorizacion.control_vigencia_version_rol x USING(version_rol_ref,revision) WHERE q.version_rol_ref=r.version_rol_ref FOR SHARE OF q,x;IF NOT FOUND THEN RETURN false;END IF;
 SELECT * INTO meta FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 WHERE version_rol_ref=r.version_rol_ref FOR SHARE;IF NOT FOUND THEN RETURN false;END IF;
 SELECT * INTO catalogo FROM vec_autorizacion.catalogo_accion_nominal_v1 WHERE version_rol_ref=r.version_rol_ref AND accion_ref='accion:'||(d->>'accion') FOR SHARE;IF NOT FOUND THEN RETURN false;END IF;
 ahora:=clock_timestamp();
 RETURN a.version_rol_ref=r.version_rol_ref AND a.principal_id=d->>'principal_id' AND a.documento->>'estado'='activa'
 AND a.huella_sha256=d->>'asignacion_huella_sha256' AND a.huella_sha256=encode(pg_catalog.sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(a.documento),'UTF8')),'hex')
 AND ahora>=(a.documento->>'vigente_desde')::timestamptz AND ahora<(a.documento->>'vigente_hasta')::timestamptz
 AND a.documento->'ambitos' @> jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(org)),jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(unidad)))
 AND r.rol_id='administracion_perfiles' AND r.version_rol_ref='rol:administracion_perfiles:v'||r.version::text AND r.documento->>'estado'='publicada' AND r.huella_sha256=d->>'version_rol_huella_sha256'
 AND r.huella_sha256=encode(pg_catalog.sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
 AND ct.estado='habilitada' AND ct.version_rol_ref=d->>'control_vigencia_version_rol_ref' AND ct.revision::text=d->>'control_vigencia_version_rol_revision' AND ct.huella_sha256=d->>'control_vigencia_version_rol_huella_sha256'
 AND ct.huella_sha256=encode(pg_catalog.sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(ct.documento),'UTF8')),'hex')
 AND meta.categoria_administrativa='aplicacion' AND meta.tipo_perfil='fijo_sistema' AND meta.version_rol_huella_sha256=r.huella_sha256 AND meta.fuente_ref=r.version_rol_ref AND meta.fuente_version=r.version AND meta.fuente_huella_sha256=r.huella_sha256
 AND catalogo.fuente_ref=r.version_rol_ref AND catalogo.fuente_version=r.version AND catalogo.fuente_huella_sha256=r.huella_sha256 AND catalogo.clase_control='administrador_aplicacion'
 AND catalogo.dimensiones_ambito='["organizacion_ref","unidad_ref"]'::jsonb AND ahora>=catalogo.vigente_desde AND(catalogo.vigente_hasta IS NULL OR ahora<catalogo.vigente_hasta)
 AND catalogo.concesion=(CASE WHEN d->>'accion'='vec.persona.denominacion.publicar' THEN vec_autorizacion.concesiones_denominacion_persona_admin_v1()->0 ELSE vec_autorizacion.concesiones_denominacion_persona_admin_v1()->1 END)
 AND EXISTS(SELECT 1 FROM jsonb_array_elements(r.documento->'concesiones') x WHERE x=catalogo.concesion)
 AND d->>'garantia_minima'='alto' AND ahora<(d->>'valida_hasta')::timestamptz
 AND vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(d->'vinculo_autenticacion_actor') IS TRUE;
EXCEPTION WHEN data_exception OR no_data_found THEN RETURN false;
END 
$function$;

CREATE FUNCTION vec_autorizacion.validar_administrador_usuarios_vigente_aut48(d jsonb, m jsonb)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog'
 SET "TimeZone" TO 'UTC'
 SET row_security TO 'on'
AS $function$
DECLARE recurso jsonb;a record;r record;ct record;meta record;catalogo record;ahora timestamptz;concesion jsonb;org text;unidad text;
BEGIN
 PERFORM vec_autorizacion.exigir_runtime_usuarios_admin_v1();
 recurso:=vec_autorizacion.recurso_lectura_usuarios_admin_v1(vec_autorizacion.canon_lectura_usuarios_admin_v1(m));
 concesion:=vec_autorizacion.concesiones_usuarios_admin_v1()->(CASE WHEN recurso->>'accion'='administracion.usuarios.listar' THEN 0 ELSE 1 END);
 IF (d->>'version_rol_ref' ~ '^rol:administracion_perfiles:v[1-9][0-9]{0,8}$') IS NOT TRUE OR d->>'concedida' IS DISTINCT FROM 'true'
 OR d->>'accion' IS DISTINCT FROM recurso->>'accion' OR d->>'modulo_id' IS DISTINCT FROM 'administracion' OR d->>'tipo_recurso' IS DISTINCT FROM recurso->>'tipo_recurso'
 OR d->>'recurso_ref' IS DISTINCT FROM recurso->>'recurso_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM recurso->>'contexto_sha256'
 OR d->>'finalidad' IS DISTINCT FROM 'gestion_usuarios' OR d->'campos_permitidos' IS DISTINCT FROM concesion->'campos_permitidos'
 OR d->'obligaciones' IS DISTINCT FROM '["auditar"]'::jsonb OR d->>'garantia_minima' IS DISTINCT FROM 'alto'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'administracion_privilegiada'
 OR d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'true' THEN RETURN false;END IF;
 org:=m->>'organizacion_ref';unidad:=m->>'unidad_ref';
 SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref) WHERE q.perfil_activo_ref=d->>'perfil_activo_ref' AND q.asignacion_ref=d->>'asignacion_ref' FOR SHARE OF q,x;
 IF NOT FOUND THEN RETURN false;END IF;
 SELECT x.* INTO r FROM vec_autorizacion.version_rol x WHERE version_rol_ref=d->>'version_rol_ref' FOR SHARE;IF NOT FOUND THEN RETURN false;END IF;
 SELECT x.* INTO ct FROM vec_autorizacion.control_vigencia_version_rol_actual q JOIN vec_autorizacion.control_vigencia_version_rol x USING(version_rol_ref,revision) WHERE q.version_rol_ref=r.version_rol_ref FOR SHARE OF q,x;IF NOT FOUND THEN RETURN false;END IF;
 SELECT x.* INTO meta FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 x WHERE version_rol_ref=r.version_rol_ref FOR SHARE;IF NOT FOUND THEN RETURN false;END IF;
 SELECT x.* INTO catalogo FROM vec_autorizacion.catalogo_accion_nominal_v1 x WHERE version_rol_ref=r.version_rol_ref AND accion_ref='accion:'||(d->>'accion') FOR SHARE;IF NOT FOUND THEN RETURN false;END IF;
 ahora:=clock_timestamp();
 RETURN a.version_rol_ref=r.version_rol_ref AND a.principal_id=d->>'principal_id' AND a.documento->>'estado'='activa'
 AND a.huella_sha256=d->>'asignacion_huella_sha256' AND a.huella_sha256=encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(a.documento),'UTF8')),'hex')
 AND ahora>=(a.documento->>'vigente_desde')::timestamptz AND ahora<(a.documento->>'vigente_hasta')::timestamptz
 AND a.documento->'ambitos' @> jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(org)),jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(unidad)))
 AND(coalesce(m#>>'{filtros,unidad_ref}','')='' OR a.documento->'ambitos' @> jsonb_build_array(jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(m#>>'{filtros,unidad_ref}'))))
 AND r.rol_id='administracion_perfiles' AND r.version_rol_ref='rol:administracion_perfiles:v'||r.version::text AND r.documento->>'estado'='publicada' AND r.huella_sha256=d->>'version_rol_huella_sha256'
 AND r.huella_sha256=encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
 AND ct.estado='habilitada' AND ct.version_rol_ref=d->>'control_vigencia_version_rol_ref' AND ct.revision::text=d->>'control_vigencia_version_rol_revision' AND ct.huella_sha256=d->>'control_vigencia_version_rol_huella_sha256'
 AND ct.huella_sha256=encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(ct.documento),'UTF8')),'hex')
 AND meta.categoria_administrativa='aplicacion' AND meta.tipo_perfil='fijo_sistema' AND meta.version_rol_huella_sha256=r.huella_sha256 AND meta.fuente_ref=r.version_rol_ref AND meta.fuente_version=r.version AND meta.fuente_huella_sha256=r.huella_sha256
 AND catalogo.fuente_ref=r.version_rol_ref AND catalogo.fuente_version=r.version AND catalogo.fuente_huella_sha256=r.huella_sha256 AND catalogo.clase_control='administrador_aplicacion'
 AND catalogo.dimensiones_ambito='["organizacion_ref","unidad_ref"]'::jsonb AND catalogo.concesion=concesion
 AND ahora>=catalogo.vigente_desde AND(catalogo.vigente_hasta IS NULL OR ahora<catalogo.vigente_hasta)
 AND EXISTS(SELECT 1 FROM jsonb_array_elements(r.documento->'concesiones') x WHERE x=concesion)
 AND ahora<(d->>'valida_hasta')::timestamptz
 AND vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(d->'vinculo_autenticacion_actor') IS TRUE;
EXCEPTION WHEN data_exception OR no_data_found THEN RETURN false;
END 
$function$;

-- Despachadores: v4 conserva su función preservada; cualquier otra versión del
-- rol de Aplicación pasa por la comprobación genérica. Usuarios y denominación
-- no existían en v4, así que no tienen rama v4.
CREATE OR REPLACE FUNCTION vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(version_ref text, p_asignacion_ref text, principal_ref text, perfil_ref text, accion text, modulo text, tipo text, finalidad text, campos jsonb, autenticacion jsonb)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog'
AS $function$
BEGIN
 IF version_ref='rol:administracion_perfiles:v4' THEN RETURN vec_autorizacion.acreditar_perfil_aplicacion_nominal_v4_preservada_aut42(version_ref,p_asignacion_ref,principal_ref,perfil_ref,accion,modulo,tipo,finalidad,campos,autenticacion);END IF;
 RETURN vec_autorizacion.acreditar_perfil_aplicacion_nominal_vigente_aut48(version_ref,p_asignacion_ref,principal_ref,perfil_ref,accion,modulo,tipo,finalidad,campos,autenticacion);
END $function$;
CREATE OR REPLACE FUNCTION vec_autorizacion.acreditar_ambito_certificado_nominal_v1(version_ref text, p_asignacion_ref text, org text)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog'
AS $function$
BEGIN
 IF version_ref='rol:administracion_perfiles:v4' THEN RETURN vec_autorizacion.acreditar_ambito_certificado_nominal_v4_preservada_aut42(version_ref,p_asignacion_ref,org);END IF;
 RETURN vec_autorizacion.acreditar_ambito_certificado_nominal_vigente_aut48(version_ref,p_asignacion_ref,org);
END $function$;
CREATE OR REPLACE FUNCTION vec_autorizacion.validar_administrador_denominacion_persona_v1(d jsonb, m jsonb)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog'
 SET row_security TO 'on'
AS $function$
BEGIN
 RETURN vec_autorizacion.validar_administrador_denominacion_persona_vigente_aut48(d,m);
END $function$;
CREATE OR REPLACE FUNCTION vec_autorizacion.validar_administrador_usuarios_v1(d jsonb, m jsonb)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog'
 SET "TimeZone" TO 'UTC'
 SET row_security TO 'on'
AS $function$
BEGIN
 RETURN vec_autorizacion.validar_administrador_usuarios_vigente_aut48(d,m);
END $function$;

REVOKE ALL ON FUNCTION vec_autorizacion.acreditar_perfil_aplicacion_nominal_vigente_aut48(text,text,text,text,text,text,text,text,jsonb,jsonb),
 vec_autorizacion.acreditar_ambito_certificado_nominal_vigente_aut48(text,text,text),
 vec_autorizacion.validar_administrador_denominacion_persona_vigente_aut48(jsonb,jsonb),
 vec_autorizacion.validar_administrador_usuarios_vigente_aut48(jsonb,jsonb) FROM PUBLIC;
RESET ROLE;
DO $post$
BEGIN
 -- CREATE OR REPLACE conserva OID, propietario, ACL y configuración.
 IF EXISTS(SELECT 1 FROM aut48_meta a JOIN pg_proc p ON p.oid=a.oid WHERE (to_jsonb(p)-'prosrc') IS DISTINCT FROM a.meta)
 OR (SELECT count(*) FROM aut48_meta)<>4
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
  WHERE p.pronamespace='vec_autorizacion'::regnamespace AND p.proname LIKE '%\_vigente\_aut48' AND x.grantee<>p.proowner)
 OR (SELECT count(*) FROM pg_proc p WHERE p.pronamespace='vec_autorizacion'::regnamespace AND p.proname LIKE '%\_vigente\_aut48'
  AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef)<>4
 THEN RAISE EXCEPTION 'AUT48: PARO clave=postimagen actual=divergente esperado=metadatos_ACL_conservados' USING ERRCODE='55000';END IF;
END $post$;
COMMIT;
