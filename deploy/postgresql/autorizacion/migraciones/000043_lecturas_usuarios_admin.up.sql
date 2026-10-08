\set ON_ERROR_STOP on
-- AUT43. Conjunto acotado por asignaciones; consume AD185 nuevo antes de leer.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000043',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
DECLARE f oid;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN RAISE EXCEPTION 'AUT43: PARO clave=operador actual=incompatible esperado=superusuario_PG18' USING ERRCODE='55000';END IF;
 FOREACH f IN ARRAY ARRAY[to_regprocedure('vec_contexto_actor_v1.metadatos_persona_administrable_v1(text)'),to_regprocedure('vec_contexto_actor_v1.metadatos_perfil_administrable_v1(text,text)'),to_regprocedure('vec_contexto_actor_v1.version_actual_denominacion_persona_v1(text)')] LOOP
  IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_contexto_actor_v1_propietario'::regrole AND p.prosecdef)
  OR has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE') IS NOT TRUE THEN RAISE EXCEPTION 'AUT43: PARO clave=puertos_CA actual=incompatible esperado=CA32_CA34_owner_CA_EXECUTE_AUT' USING ERRCODE='55000';END IF;
 END LOOP;
 IF to_regprocedure('vec_autorizacion.concesiones_usuarios_admin_v1()') IS NULL
 OR to_regprocedure('vec_autorizacion.validar_administrador_usuarios_v1(jsonb,jsonb)') IS NOT NULL THEN RAISE EXCEPTION 'AUT43: PARO clave=preimagen actual=incompatible esperado=AUT42_sin_AUT43' USING ERRCODE='55000';END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_admin_usuarios_lector') THEN CREATE ROLE vec_admin_usuarios_lector NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_admin_usuarios_lector' AND(rolcanlogin OR rolinherit OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls OR rolconfig IS NOT NULL))
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member='vec_admin_usuarios_lector'::regrole)
 OR EXISTS(SELECT 1 FROM pg_db_role_setting WHERE setrole='vec_admin_usuarios_lector'::regrole) THEN RAISE EXCEPTION 'AUT43: PARO clave=grupo_lector actual=incompatible esperado=grupo_NOLOGIN_aislado' USING ERRCODE='55000';END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;

CREATE FUNCTION vec_autorizacion.referencia_usuarios_admin_v1(v text)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$SELECT v IS NOT NULL AND octet_length(v) BETWEEN 3 AND 128 AND v ~ '^[A-Za-z0-9_:-]+$'$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.referencia_usuarios_admin_v1(text) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion.conjunto_usuarios_admin_v1(org text,unidad text)
RETURNS text LANGUAGE plpgsql IMMUTABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE canon text;
BEGIN
 IF vec_autorizacion.referencia_usuarios_admin_v1(org) IS NOT TRUE OR vec_autorizacion.referencia_usuarios_admin_v1(unidad) IS NOT TRUE THEN RAISE EXCEPTION 'AUT43: ambito_invalido' USING ERRCODE='22023';END IF;
 canon:='{"organizacion_ref":'||to_jsonb(org)::text||',"unidad_ref":'||to_jsonb(unidad)::text||'}';
 RETURN 'conjunto_admin:'||left(encode(sha256(convert_to(E'vec.admin.conjunto-usuarios.v1\n'||canon,'UTF8')),'hex'),32);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.conjunto_usuarios_admin_v1(text,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.ligadura_cursor_usuarios_admin_v1(conjunto text,f jsonb)
RETURNS text LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT encode(sha256(convert_to(E'vec.admin.cursor-usuarios.v1\n'||conjunto||E'\n'||'{"perfil_ref":'||to_jsonb(f->>'perfil_ref')::text||',"unidad_ref":'||to_jsonb(f->>'unidad_ref')::text||',"estado":'||to_jsonb(f->>'estado')::text||'}','UTF8')),'hex')
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.ligadura_cursor_usuarios_admin_v1(text,jsonb) FROM PUBLIC;

-- Canon cerrado propio. Ningún actor, nombre, cuenta o permiso en el material.
CREATE FUNCTION vec_autorizacion.canon_lectura_usuarios_admin_v1(m jsonb)
RETURNS text LANGUAGE plpgsql IMMUTABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE can text;f jsonb;k text;listar boolean;
BEGIN
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_path_exists(m,'$.** ? (@ == null)') THEN RAISE EXCEPTION 'AUT43: material_invalido' USING ERRCODE='22023';END IF;
 listar:=m->>'esquema'='vec.admin.usuarios.listar.v1';
 IF (listar IS NOT TRUE AND m->>'esquema' IS DISTINCT FROM 'vec.admin.usuarios.consultar.v1')
 OR jsonb_typeof(m->'esquema') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'AUT43: esquema_invalido' USING ERRCODE='22023';END IF;
 FOREACH k IN ARRAY ARRAY['organizacion_ref','unidad_ref'] LOOP
  IF jsonb_typeof(m->k) IS DISTINCT FROM 'string' OR vec_autorizacion.referencia_usuarios_admin_v1(m->>k) IS NOT TRUE THEN RAISE EXCEPTION 'AUT43: ambito_invalido' USING ERRCODE='22023';END IF;
 END LOOP;
 IF jsonb_typeof(m->'conjunto_ref') IS DISTINCT FROM 'string' OR m->>'conjunto_ref' IS DISTINCT FROM vec_autorizacion.conjunto_usuarios_admin_v1(m->>'organizacion_ref',m->>'unidad_ref') THEN RAISE EXCEPTION 'AUT43: conjunto_divergente' USING ERRCODE='22023';END IF;
 can:='{"esquema":'||to_jsonb(m->>'esquema')::text||',"organizacion_ref":'||to_jsonb(m->>'organizacion_ref')::text||',"unidad_ref":'||to_jsonb(m->>'unidad_ref')::text||',"conjunto_ref":'||to_jsonb(m->>'conjunto_ref')::text;
 IF listar THEN
  f:=m->'filtros';
  IF (SELECT count(*) FROM jsonb_object_keys(m))<>7 OR NOT m ?& ARRAY['esquema','organizacion_ref','unidad_ref','conjunto_ref','filtros','cursor','limite']
  OR jsonb_typeof(f) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(f))<>3 OR NOT f ?& ARRAY['perfil_ref','unidad_ref','estado']
  OR jsonb_typeof(f->'perfil_ref') IS DISTINCT FROM 'string' OR (f->>'perfil_ref'<>'' AND (octet_length(f->>'perfil_ref')>128 OR f->>'perfil_ref' !~ '^rol:[A-Za-z0-9_-]+:v[1-9][0-9]*$'))
  OR jsonb_typeof(f->'unidad_ref') IS DISTINCT FROM 'string' OR (f->>'unidad_ref'<>'' AND vec_autorizacion.referencia_usuarios_admin_v1(f->>'unidad_ref') IS NOT TRUE)
  OR jsonb_typeof(f->'estado') IS DISTINCT FROM 'string' OR f->>'estado' NOT IN('','vigente','caducado')
  OR jsonb_typeof(m->'cursor') IS DISTINCT FROM 'string' OR octet_length(m->>'cursor')>256
  OR jsonb_typeof(m->'limite') IS DISTINCT FROM 'number' OR m->>'limite' IS DISTINCT FROM '50' THEN RAISE EXCEPTION 'AUT43: filtros_invalidos' USING ERRCODE='22023';END IF;
  IF m->>'cursor'<>'' AND(split_part(m->>'cursor',':',3)!~'^per_[A-Za-z0-9_-]{22,124}$' OR m->>'cursor' IS DISTINCT FROM 'usuarios:'||vec_autorizacion.ligadura_cursor_usuarios_admin_v1(m->>'conjunto_ref',f)||':'||split_part(m->>'cursor',':',3)) THEN RAISE EXCEPTION 'AUT43: cursor_divergente' USING ERRCODE='22023';END IF;
  can:=can||',"filtros":{"perfil_ref":'||to_jsonb(f->>'perfil_ref')::text||',"unidad_ref":'||to_jsonb(f->>'unidad_ref')::text||',"estado":'||to_jsonb(f->>'estado')::text||'},"cursor":'||to_jsonb(m->>'cursor')::text||',"limite":50';
 ELSE
  IF (SELECT count(*) FROM jsonb_object_keys(m))<>5 OR NOT m ?& ARRAY['esquema','organizacion_ref','unidad_ref','conjunto_ref','persona_ref']
  OR jsonb_typeof(m->'persona_ref') IS DISTINCT FROM 'string' OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,124}$' OR octet_length(m->>'persona_ref')>128 THEN RAISE EXCEPTION 'AUT43: persona_invalida' USING ERRCODE='22023';END IF;
  can:=can||',"persona_ref":'||to_jsonb(m->>'persona_ref')::text;
 END IF;
 RETURN can||'}';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.canon_lectura_usuarios_admin_v1(jsonb) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion.recurso_lectura_usuarios_admin_v1(p_material text)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE m jsonb;canon text;listar boolean;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 4096 THEN RAISE EXCEPTION 'AUT43: material_invalido' USING ERRCODE='22023';END IF;
 m:=p_material::jsonb;
 IF vec_autorizacion.canon_lectura_usuarios_admin_v1(m) IS DISTINCT FROM p_material THEN RAISE EXCEPTION 'AUT43: canon_divergente' USING ERRCODE='22023';END IF;
 listar:=m->>'esquema'='vec.admin.usuarios.listar.v1';
 canon:='{"ambitos":{"organizacion_ref":'||to_jsonb(m->>'organizacion_ref')::text||',"unidad_ref":'||to_jsonb(m->>'unidad_ref')::text||'},"atributos":{"material_sha256":'||to_jsonb(encode(sha256(convert_to(p_material,'UTF8')),'hex'))::text||'}}';
 RETURN jsonb_build_object('material',m,'accion',CASE WHEN listar THEN 'administracion.usuarios.listar' ELSE 'administracion.usuarios.consultar' END,
  'audiencia',CASE WHEN listar THEN 'vec.admin.usuarios.listar.v1' ELSE 'vec.admin.usuarios.consultar.v1' END,
  'tipo_recurso',CASE WHEN listar THEN 'conjunto_usuarios' ELSE 'persona_administrable' END,
  'recurso_ref',CASE WHEN listar THEN m->>'conjunto_ref' ELSE m->>'persona_ref' END,
  'contexto_canonico',canon,'contexto_sha256',encode(sha256(convert_to(canon,'UTF8')),'hex'));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.recurso_lectura_usuarios_admin_v1(text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.exigir_runtime_usuarios_admin_v1()
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('role')<>'none' OR session_user=current_user
 OR NOT EXISTS(SELECT 1 FROM pg_roles l JOIN pg_auth_members a ON a.member=l.oid JOIN pg_roles g ON g.oid=a.roleid
  WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls) AND l.rolconfig IS NULL
  AND g.oid=to_regrole('vec_admin_usuarios_lector') AND NOT(g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls) AND g.rolconfig IS NULL
  AND a.inherit_option AND NOT a.set_option AND NOT a.admin_option
  AND (SELECT count(*) FROM pg_auth_members a2 WHERE a2.member=l.oid)=1
  AND NOT EXISTS(SELECT 1 FROM pg_auth_members a2 WHERE a2.member=g.oid)
  AND NOT EXISTS(SELECT 1 FROM pg_db_role_setting s WHERE s.setrole IN(l.oid,g.oid))) THEN RAISE EXCEPTION 'AUT43: runtime_rechazado' USING ERRCODE='42501';END IF;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.exigir_runtime_usuarios_admin_v1() FROM PUBLIC;

-- Gate nominal propietario. No concede por material ni sustituye PDP/contexto.
CREATE FUNCTION vec_autorizacion.validar_administrador_usuarios_v1(d jsonb,m jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET row_security=on AS $f$
DECLARE recurso jsonb;a record;r record;ct record;meta record;catalogo record;ahora timestamptz;concesion jsonb;org text;unidad text;
BEGIN
 PERFORM vec_autorizacion.exigir_runtime_usuarios_admin_v1();
 recurso:=vec_autorizacion.recurso_lectura_usuarios_admin_v1(vec_autorizacion.canon_lectura_usuarios_admin_v1(m));
 concesion:=vec_autorizacion.concesiones_usuarios_admin_v1()->(CASE WHEN recurso->>'accion'='administracion.usuarios.listar' THEN 0 ELSE 1 END);
 IF d->>'version_rol_ref' IS DISTINCT FROM 'rol:administracion_perfiles:v5' OR d->>'concedida' IS DISTINCT FROM 'true'
 OR d->>'accion' IS DISTINCT FROM recurso->>'accion' OR d->>'modulo_id' IS DISTINCT FROM 'administracion' OR d->>'tipo_recurso' IS DISTINCT FROM recurso->>'tipo_recurso'
 OR d->>'recurso_ref' IS DISTINCT FROM recurso->>'recurso_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM recurso->>'contexto_sha256'
 OR d->>'finalidad' IS DISTINCT FROM 'gestion_usuarios' OR d->'campos_permitidos' IS DISTINCT FROM concesion->'campos_permitidos'
 OR d->'obligaciones' IS DISTINCT FROM '["auditar"]'::jsonb OR d->>'garantia_minima' IS DISTINCT FROM 'alto'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'administracion_privilegiada'
 OR d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'true' THEN RETURN false;END IF;
 org:=m->>'organizacion_ref';unidad:=m->>'unidad_ref';
 SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref) WHERE q.perfil_activo_ref=d->>'perfil_activo_ref' AND q.asignacion_ref=d->>'asignacion_ref' FOR SHARE OF q,x;
 IF NOT FOUND THEN RETURN false;END IF;
 SELECT x.* INTO r FROM vec_autorizacion.version_rol x WHERE version_rol_ref='rol:administracion_perfiles:v5' FOR SHARE;IF NOT FOUND THEN RETURN false;END IF;
 SELECT x.* INTO ct FROM vec_autorizacion.control_vigencia_version_rol_actual q JOIN vec_autorizacion.control_vigencia_version_rol x USING(version_rol_ref,revision) WHERE q.version_rol_ref=r.version_rol_ref FOR SHARE OF q,x;IF NOT FOUND THEN RETURN false;END IF;
 SELECT x.* INTO meta FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 x WHERE version_rol_ref=r.version_rol_ref FOR SHARE;IF NOT FOUND THEN RETURN false;END IF;
 SELECT x.* INTO catalogo FROM vec_autorizacion.catalogo_accion_nominal_v1 x WHERE version_rol_ref=r.version_rol_ref AND accion_ref='accion:'||(d->>'accion') FOR SHARE;IF NOT FOUND THEN RETURN false;END IF;
 ahora:=clock_timestamp();
 RETURN a.version_rol_ref=r.version_rol_ref AND a.principal_id=d->>'principal_id' AND a.documento->>'estado'='activa'
 AND a.huella_sha256=d->>'asignacion_huella_sha256' AND a.huella_sha256=encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(a.documento),'UTF8')),'hex')
 AND ahora>=(a.documento->>'vigente_desde')::timestamptz AND ahora<(a.documento->>'vigente_hasta')::timestamptz
 AND a.documento->'ambitos' @> jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(org)),jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(unidad)))
 AND(coalesce(m#>>'{filtros,unidad_ref}','')='' OR a.documento->'ambitos' @> jsonb_build_array(jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(m#>>'{filtros,unidad_ref}'))))
 AND r.rol_id='administracion_perfiles' AND r.version=5 AND r.documento->>'estado'='publicada' AND r.huella_sha256=d->>'version_rol_huella_sha256'
 AND r.huella_sha256=encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
 AND ct.estado='habilitada' AND ct.version_rol_ref=d->>'control_vigencia_version_rol_ref' AND ct.revision::text=d->>'control_vigencia_version_rol_revision' AND ct.huella_sha256=d->>'control_vigencia_version_rol_huella_sha256'
 AND ct.huella_sha256=encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(ct.documento),'UTF8')),'hex')
 AND meta.categoria_administrativa='aplicacion' AND meta.tipo_perfil='fijo_sistema' AND meta.version_rol_huella_sha256=r.huella_sha256 AND meta.fuente_ref=r.version_rol_ref AND meta.fuente_version=5 AND meta.fuente_huella_sha256=r.huella_sha256
 AND catalogo.version=1 AND catalogo.fuente_ref=r.version_rol_ref AND catalogo.fuente_version=5 AND catalogo.fuente_huella_sha256=r.huella_sha256 AND catalogo.clase_control='administrador_aplicacion'
 AND catalogo.dimensiones_ambito='["organizacion_ref","unidad_ref"]'::jsonb AND catalogo.concesion=concesion
 AND ahora>=catalogo.vigente_desde AND(catalogo.vigente_hasta IS NULL OR ahora<catalogo.vigente_hasta)
 AND EXISTS(SELECT 1 FROM jsonb_array_elements(r.documento->'concesiones') x WHERE x=concesion)
 AND ahora<(d->>'valida_hasta')::timestamptz
 AND vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(d->'vinculo_autenticacion_actor') IS TRUE;
EXCEPTION WHEN data_exception OR no_data_found THEN RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.validar_administrador_usuarios_v1(jsonb,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.validar_administrador_usuarios_v1(jsonb,jsonb),vec_autorizacion.canon_lectura_usuarios_admin_v1(jsonb),vec_autorizacion.recurso_lectura_usuarios_admin_v1(text) TO vec_autorizacion_atestada_v3_propietario;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_autorizacion_atestada_v3_propietario;

CREATE FUNCTION vec_autorizacion.estado_asignacion_usuarios_admin_v1(d jsonb,ahora timestamptz)
RETURNS text LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT CASE WHEN d->>'estado'='revocada' THEN 'revocado'
 WHEN ahora<(d->>'vigente_desde')::timestamptz THEN 'pendiente'
 WHEN ahora>=(d->>'vigente_hasta')::timestamptz THEN 'caducado' ELSE 'vigente' END
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.estado_asignacion_usuarios_admin_v1(jsonb,timestamptz) FROM PUBLIC;

-- Proyección privada DESPUÉS del consumo. AUT sólo lee tablas de su autoridad.
CREATE FUNCTION vec_autorizacion.proyectar_persona_usuarios_admin_v1(p_persona text,org text,unidad text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET row_security=on AS $f$
DECLARE a record;ca jsonb;persona jsonb;nombre jsonb;perfiles jsonb:='[]'::jsonb;ahora timestamptz:=clock_timestamp();
BEGIN
 persona:=vec_contexto_actor_v1.metadatos_persona_administrable_v1(p_persona);
 IF persona IS NULL OR persona->>'persona_ref' IS DISTINCT FROM p_persona THEN RAISE EXCEPTION 'AUT43: metadata_no_disponible' USING ERRCODE='55000';END IF;
 FOR a IN SELECT x.* FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
  WHERE x.principal_id=p_persona AND x.documento->'ambitos' @> jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(org)),jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(unidad)))
  ORDER BY x.perfil_activo_ref COLLATE "C" FOR SHARE OF q LOOP
  ca:=vec_contexto_actor_v1.metadatos_perfil_administrable_v1(p_persona,a.perfil_activo_ref);
  IF ca IS NULL OR ca->>'perfil_ref' IS DISTINCT FROM a.perfil_activo_ref THEN RAISE EXCEPTION 'AUT43: metadata_no_disponible' USING ERRCODE='55000';END IF;
  perfiles:=perfiles||jsonb_build_array(jsonb_build_object('perfil_ref',a.perfil_activo_ref,'rol_version_ref',a.version_rol_ref,'version',ca->'version',
   'estado',vec_autorizacion.estado_asignacion_usuarios_admin_v1(a.documento,ahora),'vigente_desde',(a.documento->>'vigente_desde')::timestamptz,'vigente_hasta',(a.documento->>'vigente_hasta')::timestamptz));
 END LOOP;
 IF jsonb_array_length(perfiles)=0 THEN RAISE EXCEPTION 'AUT43: conjunto_incoherente' USING ERRCODE='55000';END IF;
 nombre:=vec_contexto_actor_v1.version_actual_denominacion_persona_v1(p_persona);
 IF nombre IS NOT NULL AND(nombre->>'persona_ref' IS DISTINCT FROM p_persona OR jsonb_typeof(nombre->'version') IS DISTINCT FROM 'number' OR(nombre->>'version')::numeric NOT BETWEEN 1 AND 9007199254740991) THEN RAISE EXCEPTION 'AUT43: metadata_no_disponible' USING ERRCODE='55000';END IF;
 RETURN jsonb_build_object('persona_ref',p_persona,'unidad_ref',unidad,'denominacion_version',nombre->'version','perfiles',perfiles);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.proyectar_persona_usuarios_admin_v1(text,text,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.proyectar_usuarios_admin_v1(m jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET row_security=on AS $f$
DECLARE org text:=m->>'organizacion_ref';unidad text:=m->>'unidad_ref';f jsonb:=m->'filtros';posicion text:='';cursor text:=coalesce(m->>'cursor','');ligadura text;pagina text[];cantidad integer;datos jsonb:='[]'::jsonb;persona text;ahora timestamptz:=clock_timestamp();
BEGIN
 IF m->>'esquema'='vec.admin.usuarios.consultar.v1' THEN
  IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
   WHERE x.principal_id=m->>'persona_ref' AND vec_contexto_actor_v1.metadatos_persona_administrable_v1(x.principal_id) IS NOT NULL AND x.documento->'ambitos' @> jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(org)),jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(unidad)))) THEN RETURN NULL;END IF;
  RETURN vec_autorizacion.proyectar_persona_usuarios_admin_v1(m->>'persona_ref',org,unidad);
 END IF;
 ligadura:=vec_autorizacion.ligadura_cursor_usuarios_admin_v1(m->>'conjunto_ref',f);
 IF cursor<>'' THEN
  IF split_part(cursor,':',1)<>'usuarios' OR split_part(cursor,':',2) IS DISTINCT FROM ligadura OR split_part(cursor,':',3)!~'^per_[A-Za-z0-9_-]{22,124}$' OR cursor IS DISTINCT FROM 'usuarios:'||ligadura||':'||split_part(cursor,':',3) THEN RAISE EXCEPTION 'AUT43: cursor_divergente' USING ERRCODE='22023';END IF;
  posicion:=split_part(cursor,':',3);
 END IF;
 -- TODOS los predicados corresponden a x, la misma asignación actual.
 -- La página se limita tras DISTINCT; CA no pagina el conjunto primero.
 SELECT array_agg(persona_ref ORDER BY persona_ref COLLATE "C"),count(*) INTO pagina,cantidad FROM (
  SELECT d.persona_ref FROM (SELECT DISTINCT x.principal_id COLLATE "C" AS persona_ref FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
  WHERE x.documento->'ambitos' @> jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(org)),jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(unidad)))
   AND(f->>'perfil_ref'='' OR x.version_rol_ref=f->>'perfil_ref')
   AND(f->>'unidad_ref'='' OR x.documento->'ambitos' @> jsonb_build_array(jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(f->>'unidad_ref'))))
   AND(f->>'estado'='' OR vec_autorizacion.estado_asignacion_usuarios_admin_v1(x.documento,ahora)=f->>'estado')
   AND(posicion='' OR x.principal_id COLLATE "C">posicion COLLATE "C")
   AND vec_contexto_actor_v1.metadatos_persona_administrable_v1(x.principal_id) IS NOT NULL
  ) d ORDER BY d.persona_ref COLLATE "C" LIMIT 51
 ) p;
 FOREACH persona IN ARRAY coalesce(pagina[1:50],ARRAY[]::text[]) LOOP datos:=datos||jsonb_build_array(vec_autorizacion.proyectar_persona_usuarios_admin_v1(persona,org,unidad));END LOOP;
 RETURN jsonb_build_object('personas',datos,'siguiente_cursor',CASE WHEN cantidad>50 THEN 'usuarios:'||ligadura||':'||pagina[50] ELSE '' END);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.proyectar_usuarios_admin_v1(jsonb) FROM PUBLIC;

-- AD185 se declara después del gate AUT43: referencia PL/pgSQL diferida, no
-- instala ni copia núcleo ajeno. AD185 sólo presta su fachada propietaria nueva.
CREATE FUNCTION vec_autorizacion.leer_usuarios_admin_interna_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_listar boolean)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET row_security=on AS $f$
DECLARE r jsonb;m jsonb;d jsonb;c jsonb;consumo record;datos jsonb;
BEGIN
 PERFORM vec_autorizacion.exigir_runtime_usuarios_admin_v1();
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 r:=vec_autorizacion.recurso_lectura_usuarios_admin_v1(p_material);m:=r->'material';
 IF p_listar IS NULL OR p_listar IS DISTINCT FROM (m->>'esquema'='vec.admin.usuarios.listar.v1') THEN RAISE EXCEPTION 'AUT43: consulta_divergente' USING ERRCODE='22023';END IF;
 d:=convert_from(p_decision,'UTF8')::jsonb;c:=convert_from(p_capacidad,'UTF8')::jsonb;
 IF vec_autorizacion.validar_administrador_usuarios_v1(d,m) IS NOT TRUE THEN RAISE EXCEPTION 'AUT43: administrador_rechazado' USING ERRCODE='42501';END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_usuarios_admin_v3_atestada(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.decision_ref IS DISTINCT FROM d->>'decision_ref' OR consumo.efecto_ref IS DISTINCT FROM r->>'recurso_ref' OR consumo.huella_efecto_sha256 IS DISTINCT FROM r->>'contexto_sha256' OR consumo.auditoria_ref IS NULL OR consumo.auditoria_ref !~'^aud_v3_[0-9a-f]{32}$' THEN RAISE EXCEPTION 'AUT43: acuse_rechazado' USING ERRCODE='42501';END IF;
 datos:=vec_autorizacion.proyectar_usuarios_admin_v1(m);
 IF vec_autorizacion.validar_administrador_usuarios_v1(d,m) IS NOT TRUE
 OR vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
 OR clock_timestamp()>=(c->>'expira_en')::timestamptz OR clock_timestamp()>=(c->>'decision_valida_hasta')::timestamptz
 OR clock_timestamp()>=(c->>'configuracion_expira_en')::timestamptz OR clock_timestamp()>=(c->>'raiz_valida_hasta')::timestamptz THEN RAISE EXCEPTION 'AUT43: vigencia_agotada' USING ERRCODE='42501';END IF;
 RETURN jsonb_build_object('datos',datos,'consumo',to_jsonb(consumo));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.leer_usuarios_admin_interna_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,boolean) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion.listar_usuarios_admin_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$ SELECT vec_autorizacion.leer_usuarios_admin_interna_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,true)$f$;
CREATE FUNCTION vec_autorizacion.consultar_usuario_admin_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$ SELECT vec_autorizacion.leer_usuarios_admin_interna_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,false)$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.listar_usuarios_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_autorizacion.consultar_usuario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_usuarios_lector;
GRANT EXECUTE ON FUNCTION vec_autorizacion.listar_usuarios_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_autorizacion.consultar_usuario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_admin_usuarios_lector;
DO $acl$
DECLARE f oid;objetivo oid;nombre text;
BEGIN
 FOR f,nombre IN SELECT p.oid,p.proname FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_autorizacion' AND p.proname IN('conjunto_usuarios_admin_v1','canon_lectura_usuarios_admin_v1','recurso_lectura_usuarios_admin_v1','exigir_runtime_usuarios_admin_v1','validar_administrador_usuarios_v1','proyectar_persona_usuarios_admin_v1','proyectar_usuarios_admin_v1','leer_usuarios_admin_interna_v1','listar_usuarios_admin_v1','consultar_usuario_admin_v1') LOOP
  objetivo:=CASE WHEN nombre IN('validar_administrador_usuarios_v1','canon_lectura_usuarios_admin_v1','recurso_lectura_usuarios_admin_v1') THEN to_regrole('vec_autorizacion_atestada_v3_propietario')
    WHEN nombre IN('listar_usuarios_admin_v1','consultar_usuario_admin_v1') THEN to_regrole('vec_admin_usuarios_lector') ELSE to_regrole('vec_autorizacion_propietario') END;
  IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef)
  OR NOT has_function_privilege(objetivo,f,'EXECUTE')
  OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,objetivo) OR a.privilege_type<>'EXECUTE' OR a.is_grantable)) THEN
   RAISE EXCEPTION 'AUT43: PARO clave=ACL actual=incompatible esperado=owner_AUT_y_puerto_exacto' USING ERRCODE='55000';END IF;
 END LOOP;
END $acl$;
COMMIT;
