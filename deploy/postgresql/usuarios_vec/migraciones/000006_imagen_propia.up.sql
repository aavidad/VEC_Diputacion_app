\set ON_ERROR_STOP on
-- Usuarios 000006 (5.08c): «Mi imagen». Aplicar tras AD3-108, Documentos
-- 000007 y Usuarios 000001–000005. Cada llamada usa una transacción
-- SERIALIZABLE READ WRITE con el LOGIN ejecutor de su superficie.
-- Aquí solo vive la elección (iniciales, icono o foto, con su paleta) y la
-- referencia opaca de la foto; los bytes los custodia Documentos, al que se
-- llama después de consumir la V3 del titular en la misma transacción.
BEGIN;
SET LOCAL ROLE vec_usuarios_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_usuarios:migracion:000006',0));
DO $pre$ BEGIN
 IF current_user<>'vec_usuarios_propietario'
    OR to_regclass('vec_usuarios.correos_conjunto') IS NULL
    OR to_regprocedure('vec_usuarios.superficie_sesion_correos()') IS NULL
    OR to_regprocedure('vec_usuarios.rechazar_cambio_inmutable()') IS NULL
    OR to_regprocedure('vec_usuarios.sesion_migradora_catalogo()') IS NULL
    OR to_regclass('vec_usuarios.imagen_actual') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_imagen_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_usuarios_propietario','vec_documentos.custodiar_imagen_personal_v1(text,bytea,text,text)','EXECUTE')
    OR NOT has_function_privilege('vec_usuarios_propietario','vec_documentos.retirar_imagen_personal_v1(text,text,text)','EXECUTE')
    OR NOT has_function_privilege('vec_usuarios_propietario','vec_documentos.abrir_imagen_personal_v1(text,text)','EXECUTE')
 THEN RAISE EXCEPTION 'Usuarios 000006: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Vocabulario cerrado, igual que domain/imagen.go.
CREATE FUNCTION vec_usuarios.eleccion_imagen_valida(v jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
BEGIN
 RETURN jsonb_typeof(v)='object'
  AND ARRAY(SELECT jsonb_object_keys(v) ORDER BY 1) IS NOT DISTINCT FROM ARRAY['icono','modo','paleta']
  AND jsonb_typeof(v->'modo')='string' AND jsonb_typeof(v->'paleta')='string' AND jsonb_typeof(v->'icono')='string'
  AND v->>'modo' IN ('iniciales','icono','foto')
  AND v->>'paleta' IN ('azul','turquesa','verde','naranja','morado','gris')
  AND ((v->>'modo'='icono' AND v->>'icono' IN ('persona','estrella','hoja','sol','corazon','libro','cafe','montana'))
    OR (v->>'modo'<>'icono' AND v->>'icono'=''));
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.eleccion_imagen_valida(jsonb) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

CREATE TABLE vec_usuarios.catalogo_imagen (
 version_ref text PRIMARY KEY CHECK(version_ref ~ '^[a-z][A-Za-z0-9:._-]{2,95}$'),
 definicion jsonb NOT NULL CHECK(jsonb_typeof(definicion)='object'),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 publicado_en timestamptz(6) NOT NULL,
 CHECK(coalesce(vec_usuarios.eleccion_imagen_valida(definicion->'predeterminada'),false)
   AND coalesce(definicion#>>'{predeterminada,modo}'<>'foto',false))
);
CREATE TABLE vec_usuarios.catalogo_imagen_publicacion (
 secuencia bigint PRIMARY KEY CHECK(secuencia>0),
 version_ref text NOT NULL UNIQUE REFERENCES vec_usuarios.catalogo_imagen(version_ref),
 publicada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_usuarios.imagen_actual (
 persona_ref text PRIMARY KEY CHECK(persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 version bigint NOT NULL CHECK(version>0),
 catalogo_version_ref text NOT NULL REFERENCES vec_usuarios.catalogo_imagen(version_ref),
 eleccion jsonb NOT NULL CHECK(vec_usuarios.eleccion_imagen_valida(eleccion)),
 foto_ref text CHECK(foto_ref ~ '^docimg_[0-9a-f]{32}$'),
 foto_sha256 text CHECK(foto_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL,
 actualizado_en timestamptz(6) NOT NULL,
 CHECK((eleccion->>'modo'='foto')=(foto_ref IS NOT NULL) AND (foto_ref IS NULL)=(foto_sha256 IS NULL))
);
CREATE TABLE vec_usuarios.imagen_historia (
 persona_ref text NOT NULL CHECK(persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 version bigint NOT NULL CHECK(version>0),
 catalogo_version_ref text NOT NULL REFERENCES vec_usuarios.catalogo_imagen(version_ref),
 eleccion jsonb NOT NULL CHECK(vec_usuarios.eleccion_imagen_valida(eleccion)),
 foto_ref text CHECK(foto_ref ~ '^docimg_[0-9a-f]{32}$'),
 foto_sha256 text CHECK(foto_sha256 ~ '^[0-9a-f]{64}$'),
 foto_retirada_ref text CHECK(foto_retirada_ref ~ '^docimg_[0-9a-f]{32}$'),
 recibo_ref text NOT NULL UNIQUE,
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(persona_ref,version),
 CHECK(foto_retirada_ref IS NULL OR foto_retirada_ref IS DISTINCT FROM foto_ref),
 FOREIGN KEY(persona_ref) REFERENCES vec_usuarios.imagen_actual(persona_ref) DEFERRABLE INITIALLY DEFERRED
);
CREATE TABLE vec_usuarios.imagen_recibo (
 persona_ref text NOT NULL CHECK(persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 clave_operacion text NOT NULL CHECK(clave_operacion ~ '^[A-Za-z0-9:._-]{16,128}$'),
 huella_peticion text NOT NULL CHECK(huella_peticion ~ '^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^img_[0-9a-f]{32}$'),
 version bigint NOT NULL CHECK(version>0),
 catalogo_version_ref text NOT NULL REFERENCES vec_usuarios.catalogo_imagen(version_ref),
 eleccion jsonb NOT NULL CHECK(vec_usuarios.eleccion_imagen_valida(eleccion)),
 foto_nueva boolean NOT NULL,
 foto_retirada boolean NOT NULL,
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(persona_ref,clave_operacion),
 FOREIGN KEY(persona_ref,version) REFERENCES vec_usuarios.imagen_historia(persona_ref,version) DEFERRABLE INITIALLY DEFERRED
);
-- imagen_historia no tiene política de lectura a propósito: solo un DBA la
-- consulta (auditoría o ejercicio de derechos).
-- Marcador privado y transitorio, como en correos: se crea tras un consumo
-- V3 nuevo en esta transacción y se retira antes de devolver.
CREATE TABLE vec_usuarios.imagen_contexto (
 xid xid8 NOT NULL,
 backend_pid integer NOT NULL,
 sesion text NOT NULL,
 superficie text NOT NULL CHECK(superficie IN ('interna_corporativa','externa_personal')),
 persona_ref text NOT NULL,
 modo text NOT NULL CHECK(modo IN ('consultar','recuperar','actualizar')),
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL,
 PRIMARY KEY(xid,backend_pid,sesion)
);
CREATE TRIGGER catalogo_imagen_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios.catalogo_imagen FOR EACH ROW EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER catalogo_imagen_no_truncar BEFORE TRUNCATE ON vec_usuarios.catalogo_imagen FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER catalogo_imagen_publicacion_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios.catalogo_imagen_publicacion FOR EACH ROW EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER catalogo_imagen_publicacion_no_truncar BEFORE TRUNCATE ON vec_usuarios.catalogo_imagen_publicacion FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER imagen_historia_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios.imagen_historia FOR EACH ROW EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER imagen_historia_no_truncar BEFORE TRUNCATE ON vec_usuarios.imagen_historia FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER imagen_recibo_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios.imagen_recibo FOR EACH ROW EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER imagen_recibo_no_truncar BEFORE TRUNCATE ON vec_usuarios.imagen_recibo FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER imagen_actual_no_truncar BEFORE TRUNCATE ON vec_usuarios.imagen_actual FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER imagen_contexto_no_truncar BEFORE TRUNCATE ON vec_usuarios.imagen_contexto FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();

DO $rls$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['catalogo_imagen','catalogo_imagen_publicacion','imagen_actual','imagen_historia','imagen_recibo','imagen_contexto'] LOOP
  EXECUTE format('ALTER TABLE vec_usuarios.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_usuarios.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_usuarios.%I FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo,vec_usuarios_registrador_frontera_interno,vec_usuarios_registrador_frontera_externo',t);
 END LOOP;
END $rls$;

CREATE FUNCTION vec_usuarios.contexto_autorizado_imagen(p_persona text,p_modos text[])
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
 SELECT EXISTS(SELECT 1 FROM vec_usuarios.imagen_contexto c
 WHERE c.xid=pg_current_xact_id_if_assigned() AND c.backend_pid=pg_backend_pid()
 AND c.sesion=session_user AND c.superficie=vec_usuarios.superficie_sesion_correos()
 AND c.persona_ref=p_persona AND c.modo=ANY(p_modos))
$f$;
REVOKE ALL ON FUNCTION vec_usuarios.contexto_autorizado_imagen(text,text[]) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;
CREATE FUNCTION vec_usuarios.sesion_lectora_catalogo_imagen()
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT vec_usuarios.superficie_sesion_correos() IS NOT NULL OR vec_usuarios.sesion_migradora_catalogo()
$f$;
REVOKE ALL ON FUNCTION vec_usuarios.sesion_lectora_catalogo_imagen() FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

CREATE POLICY catalogo_imagen_lectura ON vec_usuarios.catalogo_imagen FOR SELECT TO vec_usuarios_propietario
 USING (vec_usuarios.sesion_lectora_catalogo_imagen() AND definicion->>'version_ref'=version_ref
  AND huella_sha256=encode(sha256(convert_to(definicion::text,'UTF8')),'hex'));
CREATE POLICY catalogo_imagen_alta ON vec_usuarios.catalogo_imagen FOR INSERT TO vec_usuarios_propietario
 WITH CHECK (vec_usuarios.sesion_migradora_catalogo() AND definicion->>'version_ref'=version_ref
  AND huella_sha256=encode(sha256(convert_to(definicion::text,'UTF8')),'hex'));
CREATE POLICY catalogo_imagen_publicacion_lectura ON vec_usuarios.catalogo_imagen_publicacion FOR SELECT TO vec_usuarios_propietario
 USING (vec_usuarios.sesion_lectora_catalogo_imagen());
CREATE POLICY catalogo_imagen_publicacion_alta ON vec_usuarios.catalogo_imagen_publicacion FOR INSERT TO vec_usuarios_propietario
 WITH CHECK (vec_usuarios.sesion_migradora_catalogo());
CREATE POLICY imagen_contexto_lectura ON vec_usuarios.imagen_contexto FOR SELECT TO vec_usuarios_propietario
 USING (backend_pid=pg_backend_pid() AND sesion=session_user AND superficie=vec_usuarios.superficie_sesion_correos());
CREATE POLICY imagen_contexto_alta ON vec_usuarios.imagen_contexto FOR INSERT TO vec_usuarios_propietario
 WITH CHECK (xid=pg_current_xact_id_if_assigned() AND backend_pid=pg_backend_pid() AND sesion=session_user
  AND superficie=vec_usuarios.superficie_sesion_correos());
CREATE POLICY imagen_contexto_baja ON vec_usuarios.imagen_contexto FOR DELETE TO vec_usuarios_propietario
 USING (backend_pid=pg_backend_pid() AND sesion=session_user AND superficie=vec_usuarios.superficie_sesion_correos());
CREATE POLICY imagen_actual_lectura ON vec_usuarios.imagen_actual FOR SELECT TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_imagen(persona_ref,ARRAY['consultar','actualizar']));
CREATE POLICY imagen_actual_alta ON vec_usuarios.imagen_actual FOR INSERT TO vec_usuarios_propietario WITH CHECK (vec_usuarios.contexto_autorizado_imagen(persona_ref,ARRAY['actualizar']));
CREATE POLICY imagen_actual_cambio ON vec_usuarios.imagen_actual FOR UPDATE TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_imagen(persona_ref,ARRAY['actualizar'])) WITH CHECK (vec_usuarios.contexto_autorizado_imagen(persona_ref,ARRAY['actualizar']));
CREATE POLICY imagen_historia_alta ON vec_usuarios.imagen_historia FOR INSERT TO vec_usuarios_propietario WITH CHECK (vec_usuarios.contexto_autorizado_imagen(persona_ref,ARRAY['actualizar']));
CREATE POLICY imagen_recibo_lectura ON vec_usuarios.imagen_recibo FOR SELECT TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_imagen(persona_ref,ARRAY['recuperar','actualizar']));
CREATE POLICY imagen_recibo_alta ON vec_usuarios.imagen_recibo FOR INSERT TO vec_usuarios_propietario WITH CHECK (vec_usuarios.contexto_autorizado_imagen(persona_ref,ARRAY['actualizar']));

-- Misma cadena que canonico.HuellaPeticionImagen.
CREATE FUNCTION vec_usuarios.huella_peticion_imagen(p_m jsonb)
RETURNS text LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT encode(sha256(convert_to('usuarios.imagen.peticion.v1|'||(p_m->>'persona_ref')||'|'||(p_m->>'version_esperada')||'|'||
  (p_m->>'catalogo_version_ref')||'|'||(p_m#>>'{eleccion,modo}')||'|'||(p_m#>>'{eleccion,paleta}')||'|'||
  (p_m#>>'{eleccion,icono}')||'|'||(p_m->>'foto_sha256'),'UTF8')),'hex')
$f$;
REVOKE ALL ON FUNCTION vec_usuarios.huella_peticion_imagen(jsonb) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;
CREATE FUNCTION vec_usuarios.huella_contexto_imagen(p_material text)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE persona text;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 8192
 THEN RAISE EXCEPTION 'Usuarios: material imagen inválido' USING ERRCODE='22023'; END IF;
 BEGIN persona:=p_material::jsonb->>'persona_ref';
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Usuarios: material imagen inválido' USING ERRCODE='22023'; END;
 IF persona IS NULL OR persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
 THEN RAISE EXCEPTION 'Usuarios: persona inválida' USING ERRCODE='22023'; END IF;
 RETURN encode(sha256(convert_to('{"ambitos":{"persona_ref":'||to_jsonb(persona)::text||'},"atributos":{"material_sha256":"'||
  encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}','UTF8')),'hex');
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.huella_contexto_imagen(text) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

-- El material es el JSON literal de canonico.SerializarMaterialImagen.
CREATE FUNCTION vec_usuarios.validar_material_imagen(
 p_material text,p_capacidad bytea,p_decision bytea,p_persona_version numeric,p_perfil_version numeric)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE m jsonb; c jsonb; d jsonb; k text[]; h text; accion text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 8192
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_persona_version IS NULL OR p_perfil_version IS NULL
 THEN RAISE EXCEPTION 'Usuarios: material imagen denegado' USING ERRCODE='42501'; END IF;
 BEGIN m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Usuarios: material imagen inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'Usuarios: material imagen inválido' USING ERRCODE='22023'; END IF;
 SELECT array_agg(x ORDER BY x) INTO k FROM jsonb_object_keys(m) x;
 accion:=m->>'accion';
 h:=vec_usuarios.huella_contexto_imagen(p_material);
 IF k IS DISTINCT FROM ARRAY['accion','catalogo_version_ref','clave_operacion','eleccion','finalidad_ref','foto_sha256','huella_peticion','perfil_ref','persona_ref','superficie','version_esperada']
    OR EXISTS(SELECT 1 FROM jsonb_each(m) z WHERE z.key NOT IN ('version_esperada','eleccion') AND jsonb_typeof(z.value)<>'string')
    OR jsonb_typeof(m->'version_esperada') IS DISTINCT FROM 'number'
    OR jsonb_typeof(m->'eleccion') IS DISTINCT FROM 'object'
    OR accion NOT IN ('vec.imagen.consultar','vec.imagen.actualizar')
    OR m->>'superficie' IS DISTINCT FROM vec_usuarios.superficie_sesion_correos()
    OR m->>'superficie' IS DISTINCT FROM d #>> '{vinculo_autenticacion_actor,superficie}'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_usuarios.imagen.'||split_part(accion,'.',3)||'.'||(m->>'superficie')||'.v1'
    OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR m->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
    OR m->>'finalidad_ref' IS DISTINCT FROM 'finalidad:usuarios:imagen-propia:v1'
    OR m->>'catalogo_version_ref' !~ '^[a-z][A-Za-z0-9:._-]{2,95}$'
    OR m->>'persona_ref' IS DISTINCT FROM d->>'principal_id'
    OR m->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
    OR m->>'persona_ref' IS DISTINCT FROM d->>'recurso_ref'
    OR m->>'persona_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'accion' IS DISTINCT FROM accion OR c->>'operacion' IS DISTINCT FROM accion
    OR d->>'modulo_id' IS DISTINCT FROM 'usuarios' OR d->>'tipo_recurso' IS DISTINCT FROM 'imagen_persona'
    OR d->>'finalidad' IS DISTINCT FROM 'finalidad:usuarios:imagen-propia:v1' OR d->'concedida' IS DISTINCT FROM 'true'::jsonb
    OR d->'campos_permitidos' IS DISTINCT FROM '["foto","icono","modo","paleta","version"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'decision_ref' IS NULL OR length(d->>'decision_ref') NOT BETWEEN 1 AND 256
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h OR c->>'huella_efecto_sha256' IS DISTINCT FROM h
    OR m->>'version_esperada' !~ '^(0|[1-9][0-9]{0,17})$'
 THEN RAISE EXCEPTION 'Usuarios: imagen no autorizada' USING ERRCODE='42501'; END IF;
 IF accion='vec.imagen.consultar' THEN
  IF m->>'clave_operacion'<>'' OR m->>'huella_peticion'<>'' OR m->>'foto_sha256'<>''
     OR m->'eleccion' IS DISTINCT FROM '{"modo":"","paleta":"","icono":""}'::jsonb
     OR (m->>'version_esperada')::bigint<>0
  THEN RAISE EXCEPTION 'Usuarios: consulta de imagen inválida' USING ERRCODE='22023'; END IF;
 ELSIF m->>'clave_operacion' !~ '^[A-Za-z0-9:._-]{16,128}$'
    OR NOT vec_usuarios.eleccion_imagen_valida(m->'eleccion')
    OR (m->>'foto_sha256'<>'' AND (m->>'foto_sha256' !~ '^[0-9a-f]{64}$' OR m#>>'{eleccion,modo}'<>'foto'))
    OR m->>'huella_peticion' IS DISTINCT FROM vec_usuarios.huella_peticion_imagen(m)
 THEN RAISE EXCEPTION 'Usuarios: cambio de imagen inválido' USING ERRCODE='22023'; END IF;
 RETURN m;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.validar_material_imagen(text,bytea,bytea,numeric,numeric) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

-- Consume V3 fresco antes de mirar ninguna fila y abre el contexto RLS.
CREATE FUNCTION vec_usuarios.consumir_contexto_imagen(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_modo text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE m jsonb; x record; xid_actual xid8;
BEGIN
 m:=vec_usuarios.validar_material_imagen(p_material,p_capacidad,p_decision,p_persona_version,p_perfil_version);
 IF p_modo NOT IN ('consultar','recuperar','actualizar')
    OR (p_modo='consultar')<>(m->>'accion'='vec.imagen.consultar')
 THEN RAISE EXCEPTION 'Usuarios: modo de imagen inválido' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_imagen_v3_atestada(
  m->>'accion',m->>'superficie',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.efecto_ref IS DISTINCT FROM m->>'persona_ref'
    OR x.decision_ref IS DISTINCT FROM convert_from(p_decision,'UTF8')::jsonb->>'decision_ref'
    OR x.huella_efecto_sha256 IS DISTINCT FROM vec_usuarios.huella_contexto_imagen(p_material)
    OR x.consumo_huella_sha256 !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'Usuarios: consumo de imagen divergente' USING ERRCODE='42501'; END IF;
 xid_actual:=pg_current_xact_id();
 DELETE FROM vec_usuarios.imagen_contexto WHERE backend_pid=pg_backend_pid() AND xid<>xid_actual;
 IF EXISTS(SELECT 1 FROM vec_usuarios.imagen_contexto WHERE xid=xid_actual AND backend_pid=pg_backend_pid() AND sesion=session_user)
 THEN RAISE EXCEPTION 'Usuarios: contexto de imagen ya activo' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_usuarios.imagen_contexto(xid,backend_pid,sesion,superficie,persona_ref,modo,decision_ref,auditoria_ref,consumo_huella_sha256)
 VALUES(xid_actual,pg_backend_pid(),session_user,m->>'superficie',m->>'persona_ref',p_modo,x.decision_ref,x.auditoria_ref,x.consumo_huella_sha256);
 RETURN m;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.consumir_contexto_imagen(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

CREATE FUNCTION vec_usuarios.retirar_contexto_imagen(p_persona text,p_modo text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE n integer;
BEGIN
 DELETE FROM vec_usuarios.imagen_contexto WHERE xid=pg_current_xact_id_if_assigned()
 AND backend_pid=pg_backend_pid() AND sesion=session_user AND persona_ref=p_persona AND modo=p_modo;
 GET DIAGNOSTICS n=ROW_COUNT;
 IF n<>1 THEN RAISE EXCEPTION 'Usuarios: contexto de imagen incompleto' USING ERRCODE='42501'; END IF;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.retirar_contexto_imagen(text,text) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

CREATE FUNCTION vec_usuarios.catalogo_imagen_publicado()
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE c record;
BEGIN
 SELECT p.version_ref,p.definicion,p.huella_sha256 INTO c
 FROM vec_usuarios.catalogo_imagen_publicacion b JOIN vec_usuarios.catalogo_imagen p USING(version_ref)
 ORDER BY b.secuencia DESC LIMIT 1;
 IF NOT FOUND OR c.definicion->>'version_ref' IS DISTINCT FROM c.version_ref
    OR encode(sha256(convert_to(c.definicion::text,'UTF8')),'hex') IS DISTINCT FROM c.huella_sha256
 THEN RAISE EXCEPTION 'Usuarios: catálogo de imagen incompatible' USING ERRCODE='55000'; END IF;
 RETURN c.definicion;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.catalogo_imagen_publicado() FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

CREATE FUNCTION vec_usuarios.catalogo_vigente_imagen_v1(p_superficie text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
BEGIN
 IF p_superficie IS NULL OR p_superficie IS DISTINCT FROM vec_usuarios.superficie_sesion_correos()
 THEN RAISE EXCEPTION 'Usuarios: superficie de catálogo denegada' USING ERRCODE='42501'; END IF;
 RETURN vec_usuarios.catalogo_imagen_publicado();
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.catalogo_vigente_imagen_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios.catalogo_vigente_imagen_v1(text) TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

-- Consulta propia. Devuelve el estado y, solo en modo foto, los bytes que
-- Documentos entrega para la persona titular.
CREATE FUNCTION vec_usuarios.consultar_imagen_propia_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(datos jsonb,foto bytea) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; e record; cat jsonb; b bytea;
BEGIN
 m:=vec_usuarios.consumir_contexto_imagen(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'consultar');
 cat:=vec_usuarios.catalogo_imagen_publicado();
 IF m->>'catalogo_version_ref' IS DISTINCT FROM cat->>'version_ref'
 THEN RAISE EXCEPTION 'Usuarios: catálogo de imagen obsoleto' USING ERRCODE='P1409'; END IF;
 SELECT version,catalogo_version_ref,eleccion,foto_ref INTO e FROM vec_usuarios.imagen_actual WHERE persona_ref=m->>'persona_ref';
 IF FOUND THEN
  IF e.foto_ref IS NOT NULL THEN b:=vec_documentos.abrir_imagen_personal_v1(m->>'persona_ref',e.foto_ref); END IF;
  datos:=jsonb_build_object('existe',true,'persona_ref',m->>'persona_ref','version',e.version,
   'catalogo_version_ref',e.catalogo_version_ref,'eleccion',e.eleccion);
 ELSE
  datos:=jsonb_build_object('existe',false,'persona_ref',m->>'persona_ref','version',0,
   'catalogo_version_ref',cat->>'version_ref','eleccion',cat->'predeterminada');
 END IF;
 foto:=b;
 PERFORM vec_usuarios.retirar_contexto_imagen(m->>'persona_ref','consultar');
 RETURN NEXT;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.consultar_imagen_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios.consultar_imagen_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

CREATE FUNCTION vec_usuarios.recibo_imagen_json(r vec_usuarios.imagen_recibo,p_replay boolean)
RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT jsonb_build_object('recibo_ref',r.recibo_ref,'persona_ref',r.persona_ref,'version',r.version,
  'catalogo_version_ref',r.catalogo_version_ref,'eleccion',r.eleccion,'foto_nueva',r.foto_nueva,
  'foto_retirada',r.foto_retirada,'fecha_utc',r.registrada_en,'replay',p_replay)
$f$;
REVOKE ALL ON FUNCTION vec_usuarios.recibo_imagen_json(vec_usuarios.imagen_recibo,boolean) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

-- Replay por persona+clave: recibo original o NULL; otra huella es conflicto.
CREATE FUNCTION vec_usuarios.recuperar_imagen_operacion_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; r vec_usuarios.imagen_recibo;
BEGIN
 m:=vec_usuarios.consumir_contexto_imagen(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'recuperar');
 SELECT * INTO r FROM vec_usuarios.imagen_recibo WHERE persona_ref=m->>'persona_ref' AND clave_operacion=m->>'clave_operacion';
 PERFORM vec_usuarios.retirar_contexto_imagen(m->>'persona_ref','recuperar');
 IF r.recibo_ref IS NULL THEN RETURN NULL; END IF;
 IF r.huella_peticion IS DISTINCT FROM m->>'huella_peticion'
 THEN RAISE EXCEPTION 'Usuarios: clave reutilizada con otra petición' USING ERRCODE='P1409'; END IF;
 RETURN vec_usuarios.recibo_imagen_json(r,true);
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.recuperar_imagen_operacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios.recuperar_imagen_operacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

-- Cambio de imagen. Con foto nueva: retira la anterior (sus bytes se borran)
-- y custodia la nueva. Sin foto: si había una, se retira. Modo foto sin foto
-- nueva conserva la foto vigente (solo cambia la paleta). Todo en la misma
-- transacción que la V3, el CAS, el estado, la historia y el recibo.
CREATE FUNCTION vec_usuarios.guardar_imagen_propia_v1(
 p_material text,p_foto bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; cx record; cat jsonb; previo record; v bigint; ref text; ahora timestamptz(6); n integer;
 v_foto_ref text; v_foto_sha text; v_retirada text; r vec_usuarios.imagen_recibo;
BEGIN
 m:=vec_usuarios.consumir_contexto_imagen(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'actualizar');
 IF (m->>'foto_sha256'='')<>(p_foto IS NULL)
    OR (p_foto IS NOT NULL AND encode(sha256(p_foto),'hex') IS DISTINCT FROM m->>'foto_sha256')
 THEN RAISE EXCEPTION 'Usuarios: foto divergente del material' USING ERRCODE='22023'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_usuarios:imagen:'||(m->>'persona_ref'),0));
 IF EXISTS(SELECT 1 FROM vec_usuarios.imagen_recibo WHERE persona_ref=m->>'persona_ref' AND clave_operacion=m->>'clave_operacion')
 THEN RAISE EXCEPTION 'Usuarios: operación ya registrada; recuperar recibo' USING ERRCODE='40001'; END IF;
 cat:=vec_usuarios.catalogo_imagen_publicado();
 IF m->>'catalogo_version_ref' IS DISTINCT FROM cat->>'version_ref'
 THEN RAISE EXCEPTION 'Usuarios: catálogo de imagen obsoleto' USING ERRCODE='P1409'; END IF;
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(cat->'paletas') q WHERE q->>'codigo'=m#>>'{eleccion,paleta}')
    OR (m#>>'{eleccion,modo}'='icono' AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements(cat->'iconos') q WHERE q->>'codigo'=m#>>'{eleccion,icono}'))
 THEN RAISE EXCEPTION 'Usuarios: imagen fuera de catálogo' USING ERRCODE='22023'; END IF;
 SELECT version,foto_ref,foto_sha256 INTO previo FROM vec_usuarios.imagen_actual WHERE persona_ref=m->>'persona_ref' FOR UPDATE;
 IF coalesce(previo.version,0)::text IS DISTINCT FROM m->>'version_esperada'
 THEN RAISE EXCEPTION 'Usuarios: versión de imagen en conflicto' USING ERRCODE='P1409'; END IF;
 v:=coalesce(previo.version,0)+1;
 ref:='img_'||replace(gen_random_uuid()::text,'-','');
 ahora:=date_trunc('microseconds',clock_timestamp());
 v_foto_ref:=previo.foto_ref; v_foto_sha:=previo.foto_sha256;
 IF m#>>'{eleccion,modo}'='foto' AND p_foto IS NULL AND previo.foto_ref IS NULL
 THEN RAISE EXCEPTION 'Usuarios: no hay foto que conservar' USING ERRCODE='P1409'; END IF;
 IF previo.foto_ref IS NOT NULL AND (p_foto IS NOT NULL OR m#>>'{eleccion,modo}'<>'foto') THEN
  PERFORM vec_documentos.retirar_imagen_personal_v1(m->>'persona_ref',previo.foto_ref,ref);
  v_retirada:=previo.foto_ref; v_foto_ref:=NULL; v_foto_sha:=NULL;
 END IF;
 IF p_foto IS NOT NULL THEN
  v_foto_ref:=vec_documentos.custodiar_imagen_personal_v1(m->>'persona_ref',p_foto,m->>'foto_sha256',ref);
  v_foto_sha:=m->>'foto_sha256';
 END IF;
 IF previo.version IS NULL THEN
  INSERT INTO vec_usuarios.imagen_actual(persona_ref,version,catalogo_version_ref,eleccion,foto_ref,foto_sha256,recibo_ref,actualizado_en)
  VALUES(m->>'persona_ref',v,cat->>'version_ref',m->'eleccion',v_foto_ref,v_foto_sha,ref,ahora);
 ELSE
  UPDATE vec_usuarios.imagen_actual SET version=v,catalogo_version_ref=cat->>'version_ref',eleccion=m->'eleccion',
   foto_ref=v_foto_ref,foto_sha256=v_foto_sha,recibo_ref=ref,actualizado_en=ahora
  WHERE persona_ref=m->>'persona_ref' AND version=v-1;
  GET DIAGNOSTICS n=ROW_COUNT;
  IF n<>1 THEN RAISE EXCEPTION 'Usuarios: versión de imagen concurrente' USING ERRCODE='40001'; END IF;
 END IF;
 SELECT * INTO STRICT cx FROM vec_usuarios.imagen_contexto WHERE xid=pg_current_xact_id_if_assigned()
  AND backend_pid=pg_backend_pid() AND sesion=session_user AND modo='actualizar';
 INSERT INTO vec_usuarios.imagen_historia(persona_ref,version,catalogo_version_ref,eleccion,foto_ref,foto_sha256,foto_retirada_ref,recibo_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(m->>'persona_ref',v,cat->>'version_ref',m->'eleccion',v_foto_ref,v_foto_sha,v_retirada,ref,cx.decision_ref,cx.auditoria_ref,ahora);
 INSERT INTO vec_usuarios.imagen_recibo(persona_ref,clave_operacion,huella_peticion,recibo_ref,version,catalogo_version_ref,eleccion,foto_nueva,foto_retirada,decision_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
 VALUES(m->>'persona_ref',m->>'clave_operacion',m->>'huella_peticion',ref,v,cat->>'version_ref',m->'eleccion',p_foto IS NOT NULL,v_retirada IS NOT NULL,cx.decision_ref,cx.auditoria_ref,cx.consumo_huella_sha256,ahora)
 RETURNING * INTO r;
 PERFORM vec_usuarios.retirar_contexto_imagen(m->>'persona_ref','actualizar');
 RETURN vec_usuarios.recibo_imagen_json(r,false);
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.guardar_imagen_propia_v1(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios.guardar_imagen_propia_v1(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

INSERT INTO vec_usuarios.catalogo_imagen(version_ref,definicion,huella_sha256,publicado_en)
SELECT 'usuarios-imagen-v1',c,encode(sha256(convert_to(c::text,'UTF8')),'hex'),date_trunc('microseconds',clock_timestamp())
FROM (SELECT '{
 "version_ref":"usuarios-imagen-v1",
 "paletas":[{"codigo":"azul","nombre_key":"ui.usuarios.imagen.paleta.azul"},{"codigo":"turquesa","nombre_key":"ui.usuarios.imagen.paleta.turquesa"},{"codigo":"verde","nombre_key":"ui.usuarios.imagen.paleta.verde"},{"codigo":"naranja","nombre_key":"ui.usuarios.imagen.paleta.naranja"},{"codigo":"morado","nombre_key":"ui.usuarios.imagen.paleta.morado"},{"codigo":"gris","nombre_key":"ui.usuarios.imagen.paleta.gris"}],
 "iconos":[{"codigo":"persona","nombre_key":"ui.usuarios.imagen.icono.persona"},{"codigo":"estrella","nombre_key":"ui.usuarios.imagen.icono.estrella"},{"codigo":"hoja","nombre_key":"ui.usuarios.imagen.icono.hoja"},{"codigo":"sol","nombre_key":"ui.usuarios.imagen.icono.sol"},{"codigo":"corazon","nombre_key":"ui.usuarios.imagen.icono.corazon"},{"codigo":"libro","nombre_key":"ui.usuarios.imagen.icono.libro"},{"codigo":"cafe","nombre_key":"ui.usuarios.imagen.icono.cafe"},{"codigo":"montana","nombre_key":"ui.usuarios.imagen.icono.montana"}],
 "predeterminada":{"modo":"iniciales","paleta":"azul","icono":""}
}'::jsonb AS c) q;
INSERT INTO vec_usuarios.catalogo_imagen_publicacion(secuencia,version_ref,publicada_en)
VALUES(1,'usuarios-imagen-v1',date_trunc('microseconds',clock_timestamp()));

DO $acl$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p WHERE p.pronamespace='vec_usuarios'::regnamespace
     AND p.proname LIKE '%imagen%' AND p.proname NOT IN
      ('catalogo_vigente_imagen_v1','consultar_imagen_propia_v1','recuperar_imagen_operacion_v1','guardar_imagen_propia_v1')
     AND (has_function_privilege('vec_usuarios_ejecutor_interno',p.oid,'EXECUTE')
       OR has_function_privilege('vec_usuarios_ejecutor_externo',p.oid,'EXECUTE')))
    OR EXISTS(SELECT 1 FROM pg_class c WHERE c.relnamespace='vec_usuarios'::regnamespace AND c.relname LIKE '%imagen%' AND c.relkind='r'
     AND (has_table_privilege('vec_usuarios_ejecutor_interno',c.oid,'SELECT,INSERT,UPDATE,DELETE')
       OR has_table_privilege('vec_usuarios_ejecutor_externo',c.oid,'SELECT,INSERT,UPDATE,DELETE')))
    OR EXISTS(SELECT 1 FROM pg_policies WHERE schemaname='vec_usuarios' AND tablename LIKE '%imagen%'
     AND roles<>ARRAY['vec_usuarios_propietario']::name[])
 THEN RAISE EXCEPTION 'Usuarios 000006: ACL de imagen incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
