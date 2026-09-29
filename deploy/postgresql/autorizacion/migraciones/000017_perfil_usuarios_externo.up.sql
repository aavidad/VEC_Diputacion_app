\set ON_ERROR_STOP on
-- AUT-17. Exige AUT-16 y ContextoActor-14; provisión administrativa explícita.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:migracion:000017',0));
DO $preimagen$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL
    OR pg_catalog.to_regrole('vec_autorizacion_publicador_usuarios_externo') IS NOT NULL
    OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(text,text,text)') IS NULL
    OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.perfil_usuarios_externo_provisionado_v1(text,text)') IS NULL
    OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario','vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(text,text,text)','EXECUTE')
    OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario','vec_contexto_actor_v1.perfil_usuarios_externo_provisionado_v1(text,text)','EXECUTE')
    OR pg_catalog.to_regclass('vec_autorizacion.asignacion_perfil_externa') IS NULL
    OR pg_catalog.to_regclass('vec_autorizacion.asignacion_perfil_actual_externa') IS NULL
    OR pg_catalog.to_regclass('vec_autorizacion.decision_concedida_contexto_actor_v3_externa') IS NULL
    OR pg_catalog.to_regclass('vec_autorizacion.decision_denegada_contexto_actor_v3_externa') IS NULL
    OR pg_catalog.to_regprocedure('vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)') IS NULL
    OR pg_catalog.to_regprocedure('vec_autorizacion.revalidar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)') IS NULL
 THEN RAISE EXCEPTION 'AUT-17: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $preimagen$;
CREATE ROLE vec_autorizacion_publicador_usuarios_externo NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_autorizacion_fuente_usuarios_externa NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_autorizacion_motivos_usuarios_externos NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_autorizacion_registro_usuarios_externo NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
DO $conexion$
DECLARE nombre text;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['vec_autorizacion_publicador_usuarios_externo','vec_autorizacion_fuente_usuarios_externa',
  'vec_autorizacion_motivos_usuarios_externos','vec_autorizacion_registro_usuarios_externo'] LOOP
  EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO %I',pg_catalog.current_database(),nombre);
 END LOOP;
END $conexion$;
SET LOCAL ROLE vec_autorizacion_propietario;

CREATE FUNCTION vec_autorizacion.publicador_usuarios_externo_interno_valido_v1()
RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT current_setting('role')='none'
   AND EXISTS (SELECT 1 FROM pg_catalog.pg_roles l WHERE l.rolname=session_user
     AND l.rolcanlogin AND l.rolinherit AND NOT l.rolsuper AND NOT l.rolcreatedb
     AND NOT l.rolcreaterole AND NOT l.rolreplication AND NOT l.rolbypassrls AND l.rolconfig IS NULL)
   AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole)=1
   AND EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
     WHERE m.member=session_user::regrole AND m.roleid='vec_autorizacion_publicador_usuarios_externo'::regrole
       AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
   AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles r WHERE r.oid<>session_user::regrole
     AND r.oid<>'vec_autorizacion_publicador_usuarios_externo'::regrole
     AND pg_catalog.pg_has_role(session_user::regrole,r.oid,'MEMBER'))
   AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles r WHERE r.oid<>'vec_autorizacion_publicador_usuarios_externo'::regrole
     AND pg_catalog.pg_has_role('vec_autorizacion_publicador_usuarios_externo'::regrole,r.oid,'MEMBER'))
$f$;

CREATE FUNCTION vec_autorizacion.rol_usuarios_externo_acotado_v1(d jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE x jsonb; accion text; tipo text; finalidad text; campos jsonb;
        observadas text[]; esperadas text[] := ARRAY[
         'vec.preferencias.consultar','vec.preferencias.actualizar',
         'vec.imagen.consultar','vec.imagen.actualizar',
         'vec.correos.consultar','vec.correos.anadir','vec.correos.reenviar',
         'vec.correos.verificar','vec.correos.activar','vec.correos.retirar'];
BEGIN
 IF pg_catalog.jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR d->>'rol_id' IS DISTINCT FROM 'candidato_usuarios_propios_desarrollo'
    OR d->>'nombre' IS DISTINCT FROM 'areaPersonal.usuarios.rolPropio'
    OR d->>'estado' IS DISTINCT FROM 'publicada'
    OR d->>'publicada_por' IS DISTINCT FROM 'seguridad:desarrollo:no-autoritativa'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>7
    OR NOT d ?& ARRAY['rol_id','version','nombre','estado','concesiones','publicada_por','publicada_en']
    OR pg_catalog.jsonb_typeof(d->'concesiones') IS DISTINCT FROM 'array'
 THEN RETURN false; END IF;
 SELECT pg_catalog.array_agg(e->>'accion' ORDER BY e->>'accion') INTO observadas
 FROM pg_catalog.jsonb_array_elements(d->'concesiones') e;
 IF observadas IS DISTINCT FROM
    (SELECT pg_catalog.array_agg(a ORDER BY a) FROM pg_catalog.unnest(esperadas) a)
 THEN RETURN false; END IF;
 FOR x IN SELECT value FROM pg_catalog.jsonb_array_elements(d->'concesiones') LOOP
  accion:=x->>'accion';
  tipo:=CASE WHEN accion LIKE 'vec.preferencias.%' THEN 'preferencias_persona'
    WHEN accion LIKE 'vec.imagen.%' THEN 'imagen_persona' ELSE 'correos_persona' END;
  finalidad:=CASE WHEN tipo='preferencias_persona' THEN 'finalidad:usuarios:preferencias-propias:v1'
    WHEN tipo='imagen_persona' THEN 'finalidad:usuarios:imagen-propia:v1'
    ELSE 'finalidad:usuarios:correos-propios:v1' END;
  campos:=CASE accion
   WHEN 'vec.preferencias.consultar' THEN '["catalogo","valores","version"]'::jsonb
   WHEN 'vec.preferencias.actualizar' THEN '["valores","version"]'::jsonb
   WHEN 'vec.imagen.consultar' THEN '["foto","icono","modo","paleta","version"]'::jsonb
   WHEN 'vec.imagen.actualizar' THEN '["foto","icono","modo","paleta","version"]'::jsonb
   WHEN 'vec.correos.consultar' THEN '["activo","correo_ref","direccion","estado","version"]'::jsonb
   WHEN 'vec.correos.anadir' THEN '["correo_ref","direccion","estado","version"]'::jsonb
   WHEN 'vec.correos.reenviar' THEN '["correo_ref","estado","version"]'::jsonb
   WHEN 'vec.correos.verificar' THEN '["correo_ref","estado","version"]'::jsonb
   WHEN 'vec.correos.activar' THEN '["activo","correo_ref","version"]'::jsonb
   WHEN 'vec.correos.retirar' THEN '["activo","correo_ref","estado","version"]'::jsonb
   ELSE NULL END;
  IF pg_catalog.jsonb_typeof(x) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(x))<>6
    OR NOT x ?& ARRAY['accion','modulo_id','tipo_recurso','finalidades','campos_permitidos','garantia_minima']
    OR x->>'modulo_id' IS DISTINCT FROM 'usuarios'
    OR x->>'tipo_recurso' IS DISTINCT FROM tipo
    OR x->'finalidades' IS DISTINCT FROM pg_catalog.to_jsonb(ARRAY[finalidad])
    OR x->'campos_permitidos' IS DISTINCT FROM campos
    OR x ? 'obligaciones'
    OR x->>'garantia_minima' IS DISTINCT FROM 'alto'
  THEN RETURN false; END IF;
 END LOOP;
 RETURN true;
EXCEPTION WHEN data_exception OR invalid_text_representation THEN RETURN false;
END $f$;
CREATE FUNCTION vec_autorizacion.publicar_rol_usuarios_externo_v1(
 p_rol_canonico bytea,p_rol_huella text,p_control_canonico bytea,p_control_huella text,
 p_revision_esperada numeric,p_control_huella_esperada text,p_actualizada_por text,p_acto_ref text)
RETURNS TABLE(version_rol_ref text,version bigint,revision numeric,huella_rol text,huella_control text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE d jsonb; c jsonb; ref text; id text; v bigint; rev numeric;
        existente record; actual record; x jsonb; acciones text[]; esperadas text[];
BEGIN
 IF vec_autorizacion.publicador_usuarios_externo_interno_valido_v1() IS NOT TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_rol_canonico IS NULL OR pg_catalog.octet_length(p_rol_canonico) NOT BETWEEN 1 AND 65536
    OR p_control_canonico IS NULL OR pg_catalog.octet_length(p_control_canonico) NOT BETWEEN 1 AND 16384
    OR p_rol_huella !~ '^[0-9a-f]{64}$' OR p_control_huella !~ '^[0-9a-f]{64}$'
    OR pg_catalog.encode(pg_catalog.sha256(p_rol_canonico),'hex') IS DISTINCT FROM p_rol_huella
    OR pg_catalog.encode(pg_catalog.sha256(p_control_canonico),'hex') IS DISTINCT FROM p_control_huella
    OR p_revision_esperada IS NULL OR p_revision_esperada<0 OR p_revision_esperada<>trunc(p_revision_esperada)
    OR (p_revision_esperada=0 AND p_control_huella_esperada IS NOT NULL)
    OR (p_revision_esperada>0 AND p_control_huella_esperada !~ '^[0-9a-f]{64}$')
    OR vec_autorizacion.texto_positivo_valido(p_actualizada_por,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(p_acto_ref,512) IS NOT TRUE
 THEN RAISE EXCEPTION 'publicación de rol externo rechazada' USING ERRCODE='42501'; END IF;
 BEGIN
  d:=pg_catalog.convert_from(p_rol_canonico,'UTF8')::jsonb;
  c:=pg_catalog.convert_from(p_control_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'documento de rol externo inválido' USING ERRCODE='22023'; END;
 id:=d->>'rol_id'; v:=(d->>'version')::bigint;
 ref:='rol:'||id||':v'||v::text; rev:=(c->>'revision')::numeric;
 IF id NOT IN ('candidato_usuarios_propios_desarrollo')
    OR vec_autorizacion.rol_usuarios_externo_acotado_v1(d) IS NOT TRUE
    OR v IS NULL OR v<1 OR v=9223372036854775807
    OR d->>'estado' IS DISTINCT FROM 'publicada'
    OR d->>'nombre' IS DISTINCT FROM 'areaPersonal.usuarios.rolPropio'
    OR d->>'publicada_por' IS DISTINCT FROM 'seguridad:desarrollo:no-autoritativa'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>7
    OR d ?| ARRAY['retirada_por','retirada_en','retirada_ref','motivo_retirada_codigo']
    OR c->>'version_rol_ref' IS DISTINCT FROM ref
    OR c->>'estado' IS DISTINCT FROM 'habilitada'
    OR rev IS DISTINCT FROM p_revision_esperada+1
    OR c->>'actualizado_por' IS DISTINCT FROM p_actualizada_por
    OR c ?| ARRAY['acto_ref','motivo_codigo']
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(c))<>5
    OR pg_catalog.jsonb_typeof(d->'concesiones') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'rol externo inválido' USING ERRCODE='42501'; END IF;
 esperadas:=ARRAY['vec.preferencias.consultar','vec.preferencias.actualizar','vec.imagen.consultar','vec.imagen.actualizar','vec.correos.consultar','vec.correos.anadir','vec.correos.reenviar','vec.correos.verificar','vec.correos.activar','vec.correos.retirar'];

 SELECT pg_catalog.array_agg(e->>'accion' ORDER BY e->>'accion') INTO acciones
 FROM pg_catalog.jsonb_array_elements(d->'concesiones') e;
 IF acciones IS DISTINCT FROM (SELECT pg_catalog.array_agg(a ORDER BY a) FROM pg_catalog.unnest(esperadas) a)
 THEN RAISE EXCEPTION 'concesiones de rol externo incompatibles' USING ERRCODE='42501'; END IF;
 IF vec_autorizacion.rol_usuarios_externo_acotado_v1(d) IS NOT TRUE
 THEN RAISE EXCEPTION 'concesión Usuarios externa no acotada' USING ERRCODE='42501'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:rol_usuarios_externo:'||id,0));
 SELECT r.version_rol_ref,r.rol_id,r.version,r.huella_sha256,r.documento INTO existente
 FROM vec_autorizacion.version_rol r WHERE r.version_rol_ref=ref FOR SHARE;
 IF FOUND THEN
  IF existente.rol_id IS DISTINCT FROM id OR existente.version IS DISTINCT FROM v
     OR existente.huella_sha256 IS DISTINCT FROM p_rol_huella OR existente.documento IS DISTINCT FROM d
  THEN RAISE EXCEPTION 'rol externo publicado divergente' USING ERRCODE='40001'; END IF;
 ELSE
  INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
  VALUES(ref,id,v,p_rol_huella,(d->>'publicada_en')::timestamptz,d);
 END IF;
 SELECT a.revision,h.huella_sha256 INTO actual
 FROM vec_autorizacion.control_vigencia_version_rol_actual a
 JOIN vec_autorizacion.control_vigencia_version_rol h USING(version_rol_ref,revision)
 WHERE a.version_rol_ref=ref FOR UPDATE OF a;
 IF (NOT FOUND AND (p_revision_esperada<>0 OR p_control_huella_esperada IS NOT NULL))
    OR (FOUND AND (actual.revision IS DISTINCT FROM p_revision_esperada
      OR actual.huella_sha256 IS DISTINCT FROM p_control_huella_esperada))
 THEN RAISE EXCEPTION 'control de rol externo: CAS divergente' USING ERRCODE='40001'; END IF;
 INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
 VALUES(ref,rev,'habilitada',p_control_huella,(c->>'actualizado_en')::timestamptz,c);
 IF p_revision_esperada=0 THEN
  INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
  VALUES(ref,rev,(c->>'actualizado_en')::timestamptz,p_actualizada_por,p_acto_ref);
 ELSE
  UPDATE vec_autorizacion.control_vigencia_version_rol_actual AS puntero SET revision=rev,
   actualizada_en=(c->>'actualizado_en')::timestamptz,actualizada_por=p_actualizada_por,acto_ref=p_acto_ref
  WHERE puntero.version_rol_ref=ref AND puntero.revision=p_revision_esperada;
  IF NOT FOUND THEN RAISE EXCEPTION 'control de rol externo cambiado' USING ERRCODE='40001'; END IF;
 END IF;
 INSERT INTO vec_autorizacion.publicacion_candidato_externo_evento(
  tipo,acto_ref,objeto_ref,version,huella_previa,huella_nueva,publicada_por)
 VALUES('rol_control',p_acto_ref,ref,rev,p_control_huella_esperada,p_control_huella,p_actualizada_por);
 RETURN QUERY SELECT ref,v,rev,p_rol_huella,p_control_huella;
END $f$;

-- Fachada exclusiva de provisión interna. El bytea es el JSON canónico Go;
-- su huella se comprueba antes de convertirlo a jsonb. Sin alta automática.
CREATE FUNCTION vec_autorizacion.publicar_asignacion_usuarios_externo_v1(
 p_documento_canonico bytea,p_huella_sha256 text,p_version_esperada bigint,
 p_huella_esperada text,p_actualizada_por text,p_acto_ref text,p_cuenta_ref text)
RETURNS TABLE(asignacion_ref text,version bigint,huella_sha256 text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE d jsonb; r record; anterior record; ahora timestamptz:=pg_catalog.clock_timestamp();
        id text; perfil text; principal text; rol text; nueva_version bigint; ref text;
BEGIN
 IF vec_autorizacion.publicador_usuarios_externo_interno_valido_v1() IS NOT TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_documento_canonico IS NULL OR pg_catalog.octet_length(p_documento_canonico) NOT BETWEEN 1 AND 65536
    OR p_huella_sha256 !~ '^[0-9a-f]{64}$'
    OR pg_catalog.encode(pg_catalog.sha256(p_documento_canonico),'hex') IS DISTINCT FROM p_huella_sha256
    OR p_version_esperada IS NULL OR p_version_esperada<0
    OR (p_version_esperada=0 AND p_huella_esperada IS NOT NULL)
    OR (p_version_esperada>0 AND p_huella_esperada !~ '^[0-9a-f]{64}$')
    OR vec_autorizacion.texto_positivo_valido(p_actualizada_por,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(p_acto_ref,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(p_cuenta_ref,512) IS NOT TRUE
 THEN RAISE EXCEPTION 'publicación externa rechazada' USING ERRCODE='42501'; END IF;
 BEGIN d:=pg_catalog.convert_from(p_documento_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'documento externo inválido' USING ERRCODE='22023'; END;
 id:=d->>'asignacion_id'; perfil:=d->>'perfil_activo_ref'; principal:=d->>'principal_id'; rol:=d->>'version_rol_ref';
 nueva_version:=p_version_esperada+1; ref:='asignacion:'||id||':v'||nueva_version::text;
 IF vec_autorizacion.texto_positivo_valido(id,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(perfil,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(principal,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(rol,512) IS NOT TRUE
    OR (d->>'version')::bigint IS DISTINCT FROM nueva_version
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>11
    OR NOT d ?& ARRAY['asignacion_id','version','perfil_activo_ref','principal_id','version_rol_ref',
      'estado','ambitos','vigente_desde','vigente_hasta','emitida_por','emitida_en']
    OR d->>'estado' IS DISTINCT FROM 'activa'
    OR d->>'revocada_por' IS NOT NULL OR d->>'revocada_en' IS NOT NULL OR d->>'revocacion_ref' IS NOT NULL
    OR pg_catalog.jsonb_array_length(d->'ambitos')<>1
    OR pg_catalog.jsonb_typeof(d->'ambitos'->0) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d->'ambitos'->0))<>2
    OR NOT (d->'ambitos'->0) ?& ARRAY['clave','valores']
    OR d->'ambitos'->0->>'clave' IS DISTINCT FROM 'persona_ref'
    OR pg_catalog.jsonb_array_length(d->'ambitos'->0->'valores')<>1
    OR vec_autorizacion.texto_positivo_valido(d->'ambitos'->0->'valores'->>0,512) IS NOT TRUE
    OR d->'ambitos'->0->'valores'->>0 IS DISTINCT FROM principal
    OR d->>'emitida_por' IS DISTINCT FROM p_actualizada_por
    OR d->>'vigente_desde' IS NULL OR d->>'vigente_hasta' IS NULL
    OR (d->>'vigente_desde')::timestamptz>ahora
    OR (d->>'vigente_hasta')::timestamptz<=ahora
    OR vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(p_cuenta_ref,principal,perfil) IS NOT TRUE
 THEN RAISE EXCEPTION 'asignación externa inválida o sin provisión' USING ERRCODE='42501'; END IF;
 SELECT vr.rol_id,vr.huella_sha256,vr.documento INTO r FROM vec_autorizacion.version_rol vr
 JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=vr.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
 WHERE vr.version_rol_ref=rol AND c.estado='habilitada' FOR SHARE OF ca;
 IF NOT FOUND OR r.rol_id NOT IN ('candidato_usuarios_propios_desarrollo')
    OR vec_autorizacion.rol_usuarios_externo_acotado_v1(r.documento) IS NOT TRUE
 THEN RAISE EXCEPTION 'rol externo no aprobado' USING ERRCODE='42501'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:asignacion_usuarios_externo:'||perfil,0));
 SELECT a.asignacion_ref,a.asignacion_id,a.version,a.principal_id,a.huella_sha256
 INTO anterior FROM vec_autorizacion.asignacion_perfil_actual_externa aa
 JOIN vec_autorizacion.asignacion_perfil_externa a ON a.asignacion_ref=aa.asignacion_ref
 WHERE aa.perfil_activo_ref=perfil FOR UPDATE OF aa;
 IF (NOT FOUND AND (p_version_esperada<>0 OR p_huella_esperada IS NOT NULL))
    OR (FOUND AND (anterior.version<>p_version_esperada OR anterior.huella_sha256 IS DISTINCT FROM p_huella_esperada
      OR anterior.asignacion_id IS DISTINCT FROM id OR anterior.principal_id IS DISTINCT FROM principal))
 THEN RAISE EXCEPTION 'asignación externa: CAS divergente' USING ERRCODE='40001'; END IF;
 INSERT INTO vec_autorizacion.asignacion_perfil_externa(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES(ref,id,nueva_version,perfil,principal,rol,p_huella_sha256,(d->>'emitida_en')::timestamptz,d);
 IF p_version_esperada=0 THEN
  INSERT INTO vec_autorizacion.asignacion_perfil_actual_externa(perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
  VALUES(perfil,ref,ahora,p_actualizada_por,p_acto_ref);
 ELSE
  UPDATE vec_autorizacion.asignacion_perfil_actual_externa AS puntero
   SET asignacion_ref=ref,actualizada_en=ahora,actualizada_por=p_actualizada_por,acto_ref=p_acto_ref
  WHERE puntero.perfil_activo_ref=perfil AND puntero.asignacion_ref=anterior.asignacion_ref;
  IF NOT FOUND THEN RAISE EXCEPTION 'asignación externa: puntero cambiado' USING ERRCODE='40001'; END IF;
 END IF;
 INSERT INTO vec_autorizacion.publicacion_candidato_externo_evento(
  tipo,acto_ref,objeto_ref,version,huella_previa,huella_nueva,publicada_por)
 VALUES('asignacion',p_acto_ref,ref,nueva_version,p_huella_esperada,p_huella_sha256,p_actualizada_por);
 RETURN QUERY SELECT ref,nueva_version,p_huella_sha256;
END $f$;

CREATE FUNCTION vec_autorizacion.obtener_instantanea_usuarios_externo_v1(p_principal_id text,p_perfil_activo_ref text)
RETURNS TABLE(documento_asignacion jsonb,documento_rol jsonb,documento_control_rol jsonb,revision_catalogo text,huella_catalogo text,documentos_politicas jsonb)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1('vec_externo_usuarios_v3_fuente_autorizacion_desarrollo','vec_autorizacion_fuente_usuarios_externa') IS NOT TRUE
    OR vec_contexto_actor_v1.perfil_usuarios_externo_provisionado_v1(p_principal_id,p_perfil_activo_ref) IS NOT TRUE
 THEN RAISE EXCEPTION 'fuente externa rechazada' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT a.documento,r.documento,c.documento,cat.revision::text,cat.huella_sha256,
   COALESCE((SELECT pg_catalog.jsonb_agg(p.documento ORDER BY p.politica_ref)
     FROM vec_autorizacion.politica_restrictiva_actual pa
     JOIN vec_autorizacion.politica_restrictiva p ON p.politica_id=pa.politica_id AND p.politica_ref=pa.politica_ref),'[]'::jsonb)
 FROM vec_autorizacion.asignacion_perfil_actual_externa aa
 JOIN vec_autorizacion.asignacion_perfil_externa a ON a.perfil_activo_ref=aa.perfil_activo_ref AND a.asignacion_ref=aa.asignacion_ref
 JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=r.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
 CROSS JOIN vec_autorizacion.control_catalogo_politicas cat
 WHERE aa.perfil_activo_ref=p_perfil_activo_ref AND a.principal_id=p_principal_id AND cat.control_id=true
   AND r.rol_id='candidato_usuarios_propios_desarrollo'
   AND vec_autorizacion.rol_usuarios_externo_acotado_v1(r.documento) IS TRUE;
END $f$;

CREATE FUNCTION vec_autorizacion.resolver_motivo_usuarios_externo_v1(p_catalogo_id text,p_catalogo_version integer,p_huella text,p_entrada text,p_instante timestamptz)
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1('vec_externo_usuarios_v3_motivos_autorizacion_desarrollo','vec_autorizacion_motivos_usuarios_externos') IS NOT TRUE
    OR p_catalogo_id NOT IN ('motivos_usuarios_propios_desarrollo')
 THEN RAISE EXCEPTION 'motivo externo rechazado' USING ERRCODE='42501'; END IF;
 RETURN vec_autorizacion.resolver_motivo_autorizacion_v2_historico(p_catalogo_id,p_catalogo_version,p_huella,p_entrada,p_instante);
END $funcion$;

CREATE FUNCTION vec_autorizacion.revalidar_sesion_vinculo_usuarios_externo_v2(
 p_vinculo jsonb,p_emitida_en timestamptz,p_valida_hasta timestamptz,p_instante timestamptz)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
BEGIN
 IF vec_autorizacion.vinculo_contexto_actor_v2_valido(p_vinculo) IS NOT TRUE
    OR NOT (vec_autorizacion.login_candidato_externo_v1('vec_externo_usuarios_v3_registro_autorizacion_desarrollo','vec_autorizacion_registro_usuarios_externo')
       OR vec_autorizacion.login_candidato_externo_v1('vec_externo_usuarios_desarrollo','vec_usuarios_ejecutor_externo'))
    OR p_vinculo->>'superficie' IS DISTINCT FROM 'externa_personal'
    OR p_vinculo->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
    OR p_emitida_en IS NULL OR p_valida_hasta IS NULL OR p_instante IS NULL
    OR p_valida_hasta<=p_emitida_en
 THEN RETURN false; END IF;
 IF vec_identidad_externa_v1.acreditar_sesion_externa_v1(
   p_vinculo->>'autenticacion_ref',p_vinculo->>'autenticacion_huella_sha256',
   p_vinculo->>'asercion_ref',p_vinculo->>'sesion_ref',p_vinculo->>'cuenta_ref',
   p_vinculo->>'cuenta_ordinaria_ref',(p_vinculo->>'cuenta_privilegiada')::boolean,
   p_vinculo->>'superficie',p_vinculo->>'metodo_observado',p_vinculo->>'garantia_observada',
   p_vinculo->>'politica_garantia_ref',p_vinculo->>'politica_garantia_huella_sha256',
   (p_vinculo->>'autenticacion_verificada_en')::timestamptz,(p_vinculo->>'sesion_emitida_en')::timestamptz,
   p_vinculo->>'control_sesion_ref',p_vinculo->>'control_sesion_revision','activa',
   p_vinculo->>'control_sesion_huella_sha256',
   (p_vinculo->>'sesion_revalidada_en')::timestamptz,(p_vinculo->>'sesion_valida_hasta')::timestamptz
 ) IS NOT TRUE THEN RETURN false; END IF;
 RETURN p_emitida_en >= (p_vinculo->>'sesion_revalidada_en')::timestamptz
    AND p_valida_hasta <= (p_vinculo->>'sesion_valida_hasta')::timestamptz
    AND p_instante >= (p_vinculo->>'sesion_revalidada_en')::timestamptz
    AND p_instante < (p_vinculo->>'sesion_valida_hasta')::timestamptz;
EXCEPTION WHEN data_exception OR invalid_text_representation OR datetime_field_overflow THEN RETURN false;
END $funcion$;

CREATE FUNCTION vec_autorizacion.registrar_decision_usuarios_externo_v3(p_decision bytea,p_motivo bytea,p_persona_version numeric,p_perfil_version numeric)
RETURNS TABLE(concedida boolean,codigo text,decision_huella_sha256 text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
DECLARE d jsonb; v jsonb; m jsonb;
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1('vec_externo_usuarios_v3_registro_autorizacion_desarrollo','vec_autorizacion_registro_usuarios_externo') IS NOT TRUE
    OR p_decision IS NULL OR p_motivo IS NULL
    OR pg_catalog.octet_length(p_decision) NOT BETWEEN 1 AND 524288
    OR pg_catalog.octet_length(p_motivo) NOT BETWEEN 1 AND 65536
 THEN RAISE EXCEPTION 'registro externo rechazado' USING ERRCODE='42501'; END IF;
 BEGIN
  d := pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
  m := pg_catalog.convert_from(p_motivo,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'registro externo rechazado' USING ERRCODE='42501'; END;
 v := d->'vinculo_autenticacion_actor';
 IF vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE
    OR vec_autorizacion.decision_contexto_actor_v3_canonica(d) IS DISTINCT FROM p_decision
    OR vec_autorizacion.motivo_contexto_actor_v3_canonico(m) IS DISTINCT FROM p_motivo
    OR d->>'modulo_id' IS DISTINCT FROM 'usuarios'
    OR v->>'superficie' IS DISTINCT FROM 'externa_personal'
    OR v->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
    OR d->>'principal_id' IS DISTINCT FROM v->>'principal_id'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM v->>'perfil_activo_ref'
    OR vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(
      v->>'cuenta_ref',d->>'principal_id',d->>'perfil_activo_ref') IS NOT TRUE
    OR m->'referencia'->>'catalogo_id' NOT IN ('motivos_usuarios_propios_desarrollo')
    OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.version_rol r
      WHERE r.version_rol_ref=d->>'version_rol_ref'
        AND r.rol_id='candidato_usuarios_propios_desarrollo'
        AND vec_autorizacion.rol_usuarios_externo_acotado_v1(r.documento) IS TRUE)
 THEN RAISE EXCEPTION 'registro externo rechazado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT * FROM vec_autorizacion.registrar_decision_contexto_actor_v3_usuarios_interna(p_decision,p_motivo,p_persona_version,p_perfil_version);
END $funcion$;


-- Se conserva la semántica V3 de AUT-16, sustituyendo solo la revalidación de
-- sesión para la cuenta Usuarios. Las decisiones siguen en historia externa.
DO $clonar_v3$
DECLARE nombre text; original text; nuevo text; origen regprocedure;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['registrar_decision_contexto_actor_v3_externa_interna',
  'revalidar_decision_contexto_actor_v3_externa_interna'] LOOP
  origen:=pg_catalog.to_regprocedure('vec_autorizacion.'||nombre||'(bytea,bytea,numeric,numeric)');
  IF origen IS NULL THEN RAISE EXCEPTION 'AUT-17: registrador externo ausente' USING ERRCODE='55000'; END IF;
  SELECT pg_catalog.pg_get_functiondef(origen) INTO STRICT original;
  IF (length(original)-length(replace(original,nombre,'')))/length(nombre)<>1
     OR (length(original)-length(replace(original,'vec_autorizacion.revalidar_sesion_vinculo_externo_v2','')))/
        length('vec_autorizacion.revalidar_sesion_vinculo_externo_v2')<1
  THEN RAISE EXCEPTION 'AUT-17: registrador externo cambió' USING ERRCODE='55000'; END IF;
  nuevo:=replace(original,nombre,replace(nombre,'_externa_interna','_usuarios_interna'));
  nuevo:=replace(nuevo,'vec_autorizacion.revalidar_sesion_vinculo_externo_v2',
    'vec_autorizacion.revalidar_sesion_vinculo_usuarios_externo_v2');
  EXECUTE nuevo;
 END LOOP;
END $clonar_v3$;
CREATE FUNCTION vec_autorizacion.revalidar_decision_usuarios_externo_v3_viva(
 p_decision bytea,p_motivo bytea,p_persona_version numeric,p_perfil_version numeric)
RETURNS timestamptz LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
DECLARE d jsonb; m jsonb; v jsonb;
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1('vec_externo_usuarios_desarrollo','vec_usuarios_ejecutor_externo') IS NOT TRUE
    OR p_decision IS NULL OR p_motivo IS NULL
    OR pg_catalog.octet_length(p_decision) NOT BETWEEN 1 AND 524288
    OR pg_catalog.octet_length(p_motivo) NOT BETWEEN 1 AND 65536
 THEN RAISE EXCEPTION 'revalidación externa rechazada' USING ERRCODE='42501'; END IF;
 BEGIN
  d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
  m:=pg_catalog.convert_from(p_motivo,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'revalidación externa rechazada' USING ERRCODE='42501'; END;
 v:=d->'vinculo_autenticacion_actor';
 IF vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE
    OR vec_autorizacion.decision_contexto_actor_v3_canonica(d) IS DISTINCT FROM p_decision
    OR vec_autorizacion.motivo_contexto_actor_v3_canonico(m) IS DISTINCT FROM p_motivo
    OR d->>'modulo_id' IS DISTINCT FROM 'usuarios'
    OR v->>'superficie' IS DISTINCT FROM 'externa_personal'
    OR v->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
    OR d->>'principal_id' IS DISTINCT FROM v->>'principal_id'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM v->>'perfil_activo_ref'
    OR vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(
      v->>'cuenta_ref',d->>'principal_id',d->>'perfil_activo_ref') IS NOT TRUE
    OR m->'referencia'->>'catalogo_id' IS DISTINCT FROM 'motivos_usuarios_propios_desarrollo'
    OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.version_rol r
      WHERE r.version_rol_ref=d->>'version_rol_ref'
        AND r.rol_id='candidato_usuarios_propios_desarrollo'
        AND vec_autorizacion.rol_usuarios_externo_acotado_v1(r.documento) IS TRUE)
 THEN RAISE EXCEPTION 'revalidación externa rechazada' USING ERRCODE='42501'; END IF;
 RETURN vec_autorizacion.revalidar_decision_contexto_actor_v3_usuarios_interna(
   p_decision,p_motivo,p_persona_version,p_perfil_version);
END $funcion$;

CREATE FUNCTION vec_autorizacion.registrar_y_revalidar_decision_usuarios_externo_v3(
 p_decision bytea,p_motivo bytea,p_persona_version numeric,p_perfil_version numeric)
RETURNS TABLE(concedida boolean,codigo text,decision_huella_sha256 text,registrada_en timestamptz,revalidada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
DECLARE d jsonb; m jsonb; v jsonb; registro record; viva timestamptz;
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1('vec_externo_usuarios_desarrollo','vec_usuarios_ejecutor_externo') IS NOT TRUE
    OR p_decision IS NULL OR p_motivo IS NULL
    OR pg_catalog.octet_length(p_decision) NOT BETWEEN 1 AND 524288
    OR pg_catalog.octet_length(p_motivo) NOT BETWEEN 1 AND 65536
 THEN RAISE EXCEPTION 'registro AD3 externo rechazado' USING ERRCODE='42501'; END IF;
 BEGIN
  d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
  m:=pg_catalog.convert_from(p_motivo,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'registro AD3 externo rechazado' USING ERRCODE='42501'; END;
 v:=d->'vinculo_autenticacion_actor';
 IF vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE
    OR vec_autorizacion.decision_contexto_actor_v3_canonica(d) IS DISTINCT FROM p_decision
    OR vec_autorizacion.motivo_contexto_actor_v3_canonico(m) IS DISTINCT FROM p_motivo
    OR d->>'modulo_id' IS DISTINCT FROM 'usuarios'
    OR v->>'superficie' IS DISTINCT FROM 'externa_personal'
    OR v->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
    OR d->>'principal_id' IS DISTINCT FROM v->>'principal_id'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM v->>'perfil_activo_ref'
    OR vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(
      v->>'cuenta_ref',d->>'principal_id',d->>'perfil_activo_ref') IS NOT TRUE
    OR m->'referencia'->>'catalogo_id' NOT IN ('motivos_usuarios_propios_desarrollo')
    OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.version_rol r
       WHERE r.version_rol_ref=d->>'version_rol_ref'
         AND r.rol_id='candidato_usuarios_propios_desarrollo'
         AND vec_autorizacion.rol_usuarios_externo_acotado_v1(r.documento) IS TRUE)
 THEN RAISE EXCEPTION 'registro AD3 externo rechazado' USING ERRCODE='42501'; END IF;
 SELECT * INTO registro FROM vec_autorizacion.registrar_decision_contexto_actor_v3_usuarios_interna(
   p_decision,p_motivo,p_persona_version,p_perfil_version);
 IF NOT FOUND OR registro.concedida IS NOT TRUE THEN RETURN; END IF;
 viva:=vec_autorizacion.revalidar_decision_contexto_actor_v3_usuarios_interna(
   p_decision,p_motivo,p_persona_version,p_perfil_version);
 IF viva IS NULL THEN RETURN; END IF;
 RETURN QUERY SELECT registro.concedida,registro.codigo,registro.decision_huella_sha256,registro.registrada_en,viva;
END $funcion$;


REVOKE ALL ON FUNCTION
 vec_autorizacion.publicador_usuarios_externo_interno_valido_v1(),
 vec_autorizacion.rol_usuarios_externo_acotado_v1(jsonb),
 vec_autorizacion.publicar_rol_usuarios_externo_v1(bytea,text,bytea,text,numeric,text,text,text),
 vec_autorizacion.publicar_asignacion_usuarios_externo_v1(bytea,text,bigint,text,text,text,text),
 vec_autorizacion.obtener_instantanea_usuarios_externo_v1(text,text),
 vec_autorizacion.resolver_motivo_usuarios_externo_v1(text,integer,text,text,timestamptz),
 vec_autorizacion.revalidar_sesion_vinculo_usuarios_externo_v2(jsonb,timestamptz,timestamptz,timestamptz),
 vec_autorizacion.registrar_decision_contexto_actor_v3_usuarios_interna(bytea,bytea,numeric,numeric),
 vec_autorizacion.revalidar_decision_contexto_actor_v3_usuarios_interna(bytea,bytea,numeric,numeric),
 vec_autorizacion.registrar_decision_usuarios_externo_v3(bytea,bytea,numeric,numeric),
 vec_autorizacion.revalidar_decision_usuarios_externo_v3_viva(bytea,bytea,numeric,numeric),
 vec_autorizacion.registrar_y_revalidar_decision_usuarios_externo_v3(bytea,bytea,numeric,numeric)
 FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_autorizacion_publicador_usuarios_externo,
 vec_autorizacion_fuente_usuarios_externa,vec_autorizacion_motivos_usuarios_externos,
 vec_autorizacion_registro_usuarios_externo;
GRANT EXECUTE ON FUNCTION
 vec_autorizacion.publicar_rol_usuarios_externo_v1(bytea,text,bytea,text,numeric,text,text,text),
 vec_autorizacion.publicar_asignacion_usuarios_externo_v1(bytea,text,bigint,text,text,text,text)
 TO vec_autorizacion_publicador_usuarios_externo;
GRANT EXECUTE ON FUNCTION vec_autorizacion.obtener_instantanea_usuarios_externo_v1(text,text)
 TO vec_autorizacion_fuente_usuarios_externa;
GRANT EXECUTE ON FUNCTION vec_autorizacion.resolver_motivo_usuarios_externo_v1(text,integer,text,text,timestamptz)
 TO vec_autorizacion_motivos_usuarios_externos;
GRANT EXECUTE ON FUNCTION vec_autorizacion.registrar_decision_usuarios_externo_v3(bytea,bytea,numeric,numeric)
 TO vec_autorizacion_registro_usuarios_externo;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_autorizacion_atestada_v3_propietario;
GRANT EXECUTE ON FUNCTION
 vec_autorizacion.revalidar_decision_usuarios_externo_v3_viva(bytea,bytea,numeric,numeric),
 vec_autorizacion.registrar_y_revalidar_decision_usuarios_externo_v3(bytea,bytea,numeric,numeric)
 TO vec_autorizacion_atestada_v3_propietario;
COMMIT;
