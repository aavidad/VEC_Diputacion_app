\set ON_ERROR_STOP on
-- Usuarios 000002: AD3-106 y Usuarios 000001 son precondiciones. El llamador
-- usa transacción SERIALIZABLE READ WRITE; 40001 obliga a repetirla entera.
BEGIN;
SET LOCAL ROLE vec_usuarios_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_usuarios:migracion:000002',0));
DO $pre$ BEGIN
 IF current_user<>'vec_usuarios_propietario'
    OR to_regclass('vec_usuarios.preferencias_recibo') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_preferencias_consulta_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_usuarios.guardar_preferencias_propias_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'Usuarios 000002: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_usuarios.exigir_ejecutor_preferencias()
RETURNS void LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF current_user<>'vec_usuarios_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_usuarios_ejecutor','MEMBER')
    OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND (rolsuper OR rolbypassrls))
    OR NOT EXISTS (SELECT 1 FROM pg_auth_members m
      WHERE m.member=session_user::regrole AND m.roleid='vec_usuarios_ejecutor'::regrole
      AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
    OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)<>1
    OR EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_usuarios_ejecutor'::regrole)
 THEN RAISE EXCEPTION 'Usuarios: ejecutor denegado' USING ERRCODE='42501'; END IF;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.exigir_ejecutor_preferencias() FROM PUBLIC,vec_usuarios_ejecutor;

CREATE FUNCTION vec_usuarios.activar_contexto_preferencias(
 p_persona text,p_modo text,p_decision_ref text,p_consumo_huella_sha256 text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE xid_actual xid8;
BEGIN
 PERFORM vec_usuarios.exigir_ejecutor_preferencias();
 IF current_setting('transaction_isolation')<>'serializable'
    OR p_persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_modo NOT IN ('consultar','recuperar','actualizar')
    OR p_decision_ref IS NULL OR length(p_decision_ref) NOT BETWEEN 1 AND 256
    OR p_consumo_huella_sha256 !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'Usuarios: contexto denegado' USING ERRCODE='42501'; END IF;
 xid_actual:=pg_current_xact_id();
 -- Los únicos restos posibles de una interrupción anterior comparten PID,
 -- pero jamás xid8. Esta limpieza no afecta otros backends ni al xid actual.
 DELETE FROM vec_usuarios.contexto_transaccion
 WHERE backend_pid=pg_backend_pid() AND xid<>xid_actual;
 INSERT INTO vec_usuarios.contexto_transaccion
  (xid,backend_pid,sesion,persona_ref,modo,decision_ref,consumo_huella_sha256)
 VALUES(xid_actual,pg_backend_pid(),session_user,p_persona,p_modo,p_decision_ref,p_consumo_huella_sha256);
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.activar_contexto_preferencias(text,text,text,text) FROM PUBLIC,vec_usuarios_ejecutor;

CREATE FUNCTION vec_usuarios.retirar_contexto_preferencias(p_persona text,p_modo text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE n integer;
BEGIN
 PERFORM vec_usuarios.exigir_ejecutor_preferencias();
 DELETE FROM vec_usuarios.contexto_transaccion
 WHERE xid=pg_current_xact_id_if_assigned() AND backend_pid=pg_backend_pid()
   AND sesion=session_user AND persona_ref=p_persona AND modo=p_modo;
 GET DIAGNOSTICS n=ROW_COUNT;
 IF n<>1 THEN RAISE EXCEPTION 'Usuarios: contexto incompleto' USING ERRCODE='42501'; END IF;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.retirar_contexto_preferencias(text,text) FROM PUBLIC,vec_usuarios_ejecutor;

-- La secuencia reproduce el struct de huellaPeticion de application. Todos
-- los textos admitidos aquí son códigos cerrados, sin caracteres escapables.
CREATE FUNCTION vec_usuarios.huella_semantica_preferencias(
 p_persona text,p_version bigint,p_catalogo text,p_valores jsonb)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE canon text;
BEGIN
 IF p_persona !~ '^per_[A-Za-z0-9_-]{22,128}$' OR p_version<0
    OR p_catalogo !~ '^[a-z][A-Za-z0-9:._-]{2,95}$'
    OR vec_usuarios.valores_validos(p_valores) IS NOT TRUE
 THEN RAISE EXCEPTION 'Usuarios: huella inválida' USING ERRCODE='22023'; END IF;
 canon:='{"persona_ref":'||to_jsonb(p_persona)::text||',"version_esperada":'||p_version::text||
  ',"catalogo_version_ref":'||to_jsonb(p_catalogo)::text||',"valores":{"idioma":'||to_jsonb(p_valores->>'idioma')::text||
  ',"tamano_texto":'||to_jsonb(p_valores->>'tamano_texto')::text||',"alto_contraste":'||(p_valores->'alto_contraste')::text||
  ',"tema":'||to_jsonb(p_valores->>'tema')::text||',"inicio":'||to_jsonb(p_valores->>'inicio')::text||
  ',"filas":'||(p_valores->>'filas')||',"aviso_correo_tareas":'||(p_valores->'aviso_correo_tareas')::text||
  ',"aviso_correo_plazos":'||(p_valores->'aviso_correo_plazos')::text||'}}';
 RETURN encode(sha256(convert_to(canon,'UTF8')),'hex');
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.huella_semantica_preferencias(text,bigint,text,jsonb) FROM PUBLIC,vec_usuarios_ejecutor;

-- RecursoAutorizable.HuellaContextoAutorizacionSHA256(): el ámbito persona_ref
-- satisface la asignación V3 exacta; el atributo deriva de los bytes literales.
CREATE FUNCTION vec_usuarios.huella_contexto_preferencias(p_material text)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE material_sha text; canon text; persona text;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 8192
 THEN RAISE EXCEPTION 'Usuarios: material inválido' USING ERRCODE='22023'; END IF;
 BEGIN persona:=p_material::jsonb->>'persona_ref';
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Usuarios: material inválido' USING ERRCODE='22023'; END;
 IF persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
 THEN RAISE EXCEPTION 'Usuarios: persona inválida' USING ERRCODE='22023'; END IF;
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 canon:='{"ambitos":{"persona_ref":'||to_jsonb(persona)::text||'},"atributos":{"material_sha256":"'||material_sha||'"}}';
 RETURN encode(sha256(convert_to(canon,'UTF8')),'hex');
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.huella_contexto_preferencias(text) FROM PUBLIC,vec_usuarios_ejecutor;

CREATE FUNCTION vec_usuarios.validar_material_preferencias(
 p_material text,p_accion text,p_capacidad bytea,p_decision bytea,
 p_persona_version numeric,p_perfil_version numeric)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE m jsonb; c jsonb; d jsonb; h text; k text[];
BEGIN
 PERFORM vec_usuarios.exigir_ejecutor_preferencias();
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 8192
    OR p_capacidad IS NULL OR p_decision IS NULL
    OR p_persona_version IS NULL OR p_perfil_version IS NULL
 THEN RAISE EXCEPTION 'Usuarios: material denegado' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Usuarios: material inválido' USING ERRCODE='22023'; END;
 SELECT array_agg(x ORDER BY x) INTO k FROM jsonb_object_keys(m) x;
 h:=vec_usuarios.huella_contexto_preferencias(p_material);
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
    OR k IS DISTINCT FROM ARRAY['accion','catalogo_version_ref','clave_operacion','finalidad_ref','huella_peticion','perfil_ref','persona_ref','valores','version_esperada']
    OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR m->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
    OR m->>'accion' IS DISTINCT FROM p_accion
    OR m->>'finalidad_ref' IS DISTINCT FROM 'finalidad:usuarios:preferencias-propias:v1'
    OR m->>'catalogo_version_ref' !~ '^[a-z][A-Za-z0-9:._-]{2,95}$'
    OR m->>'version_esperada' !~ '^(0|[1-9][0-9]{0,18})$'
    OR (CASE WHEN m->>'version_esperada' ~ '^(0|[1-9][0-9]{0,18})$'
       THEN (m->>'version_esperada')::numeric>=9223372036854775807::numeric
       ELSE true END)
    OR m->>'persona_ref' IS DISTINCT FROM d->>'principal_id'
    OR m->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
    OR m->>'persona_ref' IS DISTINCT FROM d->>'recurso_ref'
    OR m->>'persona_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'accion' IS DISTINCT FROM p_accion
    OR c->>'operacion' IS DISTINCT FROM p_accion
    OR d->>'finalidad' IS DISTINCT FROM m->>'finalidad_ref'
    OR d->>'modulo_id' IS DISTINCT FROM 'usuarios'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'preferencias_persona'
    OR d->>'concedida' IS DISTINCT FROM 'true'
    OR d->>'decision_ref' IS NULL OR length(d->>'decision_ref') NOT BETWEEN 1 AND 256
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM h
    OR jsonb_typeof(m->'valores') IS DISTINCT FROM 'object'
    OR (p_accion='vec.preferencias.actualizar' AND
      (m->>'clave_operacion' !~ '^[A-Za-z0-9:._-]{16,128}$'
       OR m->>'huella_peticion' !~ '^[0-9a-f]{64}$'
       OR vec_usuarios.valores_validos(m->'valores') IS NOT TRUE
       OR m->>'huella_peticion' IS DISTINCT FROM vec_usuarios.huella_semantica_preferencias(
          m->>'persona_ref',(m->>'version_esperada')::bigint,m->>'catalogo_version_ref',m->'valores')))
 THEN RAISE EXCEPTION 'Usuarios: material no autorizado' USING ERRCODE='42501'; END IF;
 RETURN m;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.validar_material_preferencias(text,text,bytea,bytea,numeric,numeric) FROM PUBLIC,vec_usuarios_ejecutor;

CREATE FUNCTION vec_usuarios.catalogo_vigente_preferencias_v1()
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE c record;
BEGIN
 PERFORM vec_usuarios.exigir_ejecutor_preferencias();
 SELECT p.version_ref,p.definicion,p.huella_sha256 INTO STRICT c
 FROM vec_usuarios.catalogo_publicacion b JOIN vec_usuarios.catalogo_preferencias p USING(version_ref)
 ORDER BY b.secuencia DESC LIMIT 1;
 IF c.definicion->>'version_ref' IS DISTINCT FROM c.version_ref
    OR encode(sha256(convert_to(c.definicion::text,'UTF8')),'hex') IS DISTINCT FROM c.huella_sha256
 THEN RAISE EXCEPTION 'Usuarios: catálogo incompatible' USING ERRCODE='55000'; END IF;
 RETURN c.definicion;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.catalogo_vigente_preferencias_v1() FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_usuarios TO vec_usuarios_ejecutor;
GRANT EXECUTE ON FUNCTION vec_usuarios.catalogo_vigente_preferencias_v1() TO vec_usuarios_ejecutor;

CREATE FUNCTION vec_usuarios.consultar_preferencias_propias_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; x record; e record; c record; respuesta jsonb;
BEGIN
 m:=vec_usuarios.validar_material_preferencias(p_material,'vec.preferencias.consultar',p_capacidad,p_decision,p_persona_version,p_perfil_version);
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_preferencias_consulta_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.efecto_ref IS DISTINCT FROM m->>'persona_ref'
    OR x.decision_ref IS DISTINCT FROM convert_from(p_decision,'UTF8')::jsonb->>'decision_ref'
    OR x.huella_efecto_sha256 IS DISTINCT FROM vec_usuarios.huella_contexto_preferencias(p_material)
 THEN RAISE EXCEPTION 'Usuarios: consumo divergente' USING ERRCODE='42501'; END IF;
 PERFORM vec_usuarios.activar_contexto_preferencias(m->>'persona_ref','consultar',x.decision_ref,x.consumo_huella_sha256);
 SELECT p.version_ref,p.definicion INTO STRICT c
 FROM vec_usuarios.catalogo_publicacion b JOIN vec_usuarios.catalogo_preferencias p USING(version_ref)
 ORDER BY b.secuencia DESC LIMIT 1;
 IF m->>'catalogo_version_ref' IS DISTINCT FROM c.version_ref
 THEN RAISE EXCEPTION 'Usuarios: catálogo obsoleto' USING ERRCODE='P1409'; END IF;
 SELECT version,catalogo_version_ref,valores INTO e FROM vec_usuarios.preferencias_actual WHERE persona_ref=m->>'persona_ref';
 IF FOUND THEN
  respuesta:=jsonb_build_object('existe',true,'persona_ref',m->>'persona_ref','version',e.version,
    'catalogo_version_ref',e.catalogo_version_ref,'valores',e.valores);
 ELSE
  respuesta:=jsonb_build_object('existe',false,'persona_ref',m->>'persona_ref','version',0,
   'catalogo_version_ref',c.version_ref,'valores',c.definicion->'predeterminados');
 END IF;
 PERFORM vec_usuarios.retirar_contexto_preferencias(m->>'persona_ref','consultar');
 RETURN respuesta;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.consultar_preferencias_propias_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios.consultar_preferencias_propias_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_ejecutor;

CREATE FUNCTION vec_usuarios.recuperar_preferencias_operacion_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; r record; x record; respuesta jsonb;
BEGIN
 m:=vec_usuarios.validar_material_preferencias(p_material,'vec.preferencias.actualizar',p_capacidad,p_decision,p_persona_version,p_perfil_version);
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.efecto_ref IS DISTINCT FROM m->>'persona_ref'
    OR x.decision_ref IS DISTINCT FROM convert_from(p_decision,'UTF8')::jsonb->>'decision_ref'
    OR x.huella_efecto_sha256 IS DISTINCT FROM vec_usuarios.huella_contexto_preferencias(p_material)
 THEN RAISE EXCEPTION 'Usuarios: recuperación denegada' USING ERRCODE='42501'; END IF;
 PERFORM vec_usuarios.activar_contexto_preferencias(m->>'persona_ref','recuperar',x.decision_ref,x.consumo_huella_sha256);
 SELECT * INTO r FROM vec_usuarios.preferencias_recibo
  WHERE persona_ref=m->>'persona_ref' AND clave_operacion=m->>'clave_operacion';
 IF NOT FOUND THEN
  PERFORM vec_usuarios.retirar_contexto_preferencias(m->>'persona_ref','recuperar');
  RETURN NULL;
 END IF;
 IF r.huella_peticion IS DISTINCT FROM m->>'huella_peticion'
 THEN RAISE EXCEPTION 'Usuarios: clave reutilizada con otra petición' USING ERRCODE='P1409'; END IF;
 respuesta:=jsonb_build_object('recibo_ref',r.recibo_ref,'persona_ref',r.persona_ref,'version',r.version,
   'catalogo_version_ref',r.catalogo_version_ref,'valores',r.valores,'fecha_utc',r.registrada_en,'replay',true);
 PERFORM vec_usuarios.retirar_contexto_preferencias(m->>'persona_ref','recuperar');
 RETURN respuesta;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.recuperar_preferencias_operacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios.recuperar_preferencias_operacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_ejecutor;

CREATE FUNCTION vec_usuarios.guardar_preferencias_propias_v1(
 p_material text,p_valores jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; c record; previo record; x record; v bigint; ref text; ahora timestamptz(6); n integer; respuesta jsonb;
BEGIN
 m:=vec_usuarios.validar_material_preferencias(p_material,'vec.preferencias.actualizar',p_capacidad,p_decision,p_persona_version,p_perfil_version);
 IF p_valores IS DISTINCT FROM m->'valores' OR vec_usuarios.valores_validos(p_valores) IS NOT TRUE
 THEN RAISE EXCEPTION 'Usuarios: valores divergentes' USING ERRCODE='22023'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.efecto_ref IS DISTINCT FROM m->>'persona_ref'
    OR x.decision_ref IS DISTINCT FROM convert_from(p_decision,'UTF8')::jsonb->>'decision_ref'
    OR x.huella_efecto_sha256 IS DISTINCT FROM vec_usuarios.huella_contexto_preferencias(p_material)
 THEN RAISE EXCEPTION 'Usuarios: consumo divergente' USING ERRCODE='42501'; END IF;
 PERFORM vec_usuarios.activar_contexto_preferencias(m->>'persona_ref','actualizar',x.decision_ref,x.consumo_huella_sha256);
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_usuarios:preferencias:'||(m->>'persona_ref'),0));
 IF EXISTS (SELECT 1 FROM vec_usuarios.preferencias_recibo WHERE persona_ref=m->>'persona_ref' AND clave_operacion=m->>'clave_operacion')
 THEN RAISE EXCEPTION 'Usuarios: operación ya registrada; recuperar recibo' USING ERRCODE='40001'; END IF;
 SELECT p.version_ref,p.definicion,p.huella_sha256 INTO STRICT c
 FROM vec_usuarios.catalogo_publicacion b JOIN vec_usuarios.catalogo_preferencias p USING(version_ref)
 ORDER BY b.secuencia DESC LIMIT 1;
 IF m->>'catalogo_version_ref' IS DISTINCT FROM c.version_ref
 THEN RAISE EXCEPTION 'Usuarios: catálogo obsoleto' USING ERRCODE='P1409'; END IF;
 IF encode(sha256(convert_to(c.definicion::text,'UTF8')),'hex') IS DISTINCT FROM c.huella_sha256
    OR NOT (SELECT EXISTS(SELECT 1 FROM jsonb_array_elements(c.definicion->'idiomas') q WHERE q->>'codigo'=p_valores->>'idioma'))
    OR NOT (SELECT EXISTS(SELECT 1 FROM jsonb_array_elements(c.definicion->'tamanos_texto') q WHERE q->>'codigo'=p_valores->>'tamano_texto'))
    OR NOT (SELECT EXISTS(SELECT 1 FROM jsonb_array_elements(c.definicion->'temas') q WHERE q->>'codigo'=p_valores->>'tema'))
    OR NOT (SELECT EXISTS(SELECT 1 FROM jsonb_array_elements(c.definicion->'inicios') q WHERE q->>'codigo'=p_valores->>'inicio'))
    OR NOT (c.definicion->'filas') @> jsonb_build_array((p_valores->>'filas')::integer)
 THEN RAISE EXCEPTION 'Usuarios: valores fuera de catálogo' USING ERRCODE='22023'; END IF;
 SELECT version INTO previo FROM vec_usuarios.preferencias_actual WHERE persona_ref=m->>'persona_ref' FOR UPDATE;
 IF coalesce(previo.version,0)::text IS DISTINCT FROM m->>'version_esperada'
 THEN RAISE EXCEPTION 'Usuarios: versión en conflicto' USING ERRCODE='P1409'; END IF;
 v:=coalesce(previo.version,0)+1;
 ref:='pref_'||replace(gen_random_uuid()::text,'-','');
 ahora:=date_trunc('microseconds',clock_timestamp());
 IF previo.version IS NULL THEN
  INSERT INTO vec_usuarios.preferencias_actual(persona_ref,version,catalogo_version_ref,valores,recibo_ref,actualizado_en)
  VALUES(m->>'persona_ref',v,c.version_ref,p_valores,ref,ahora);
 ELSE
  UPDATE vec_usuarios.preferencias_actual SET version=v,catalogo_version_ref=c.version_ref,
   valores=p_valores,recibo_ref=ref,actualizado_en=ahora WHERE persona_ref=m->>'persona_ref' AND version=v-1;
  GET DIAGNOSTICS n=ROW_COUNT;
  IF n<>1 THEN RAISE EXCEPTION 'Usuarios: versión en conflicto' USING ERRCODE='40001'; END IF;
 END IF;
 INSERT INTO vec_usuarios.preferencias_historia(persona_ref,version,catalogo_version_ref,valores,recibo_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(m->>'persona_ref',v,c.version_ref,p_valores,ref,x.decision_ref,x.auditoria_ref,ahora);
 INSERT INTO vec_usuarios.preferencias_recibo(persona_ref,clave_operacion,huella_peticion,recibo_ref,version,catalogo_version_ref,valores,decision_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
 VALUES(m->>'persona_ref',m->>'clave_operacion',m->>'huella_peticion',ref,v,c.version_ref,p_valores,x.decision_ref,x.auditoria_ref,x.consumo_huella_sha256,ahora);
 respuesta:=jsonb_build_object('recibo_ref',ref,'persona_ref',m->>'persona_ref','version',v,
   'catalogo_version_ref',c.version_ref,'valores',p_valores,'fecha_utc',ahora,'replay',false);
 PERFORM vec_usuarios.retirar_contexto_preferencias(m->>'persona_ref','actualizar');
 RETURN respuesta;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.guardar_preferencias_propias_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios.guardar_preferencias_propias_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_ejecutor;
COMMIT;
