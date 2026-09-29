\set ON_ERROR_STOP on
-- Aspirantes 000001: ficha propia de la persona aspirante (portal externo).
-- Requiere roles_up.sql y AD3-111. Cada llamada usa una transacción
-- SERIALIZABLE READ WRITE con el LOGIN técnico del portal externo.
-- Los datos llegan ya cifrados (AES-GCM en Go); aquí solo hay sobres,
-- índices ciegos (HMAC) y referencias opacas. Nada guarda `per_`.
BEGIN;
SET LOCAL ROLE vec_aspirantes_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_aspirantes:migracion:000001',0));
DO $pre$ BEGIN
 IF current_user<>'vec_aspirantes_propietario'
    OR to_regnamespace('vec_aspirantes') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_aspirantes_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_aspirantes_propietario','vec_autorizacion_atestada_v3.consumir_aspirantes_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_aspirantes_ejecutor_externo' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'Aspirantes 000001: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE SCHEMA vec_aspirantes AUTHORIZATION vec_aspirantes_propietario;
REVOKE ALL ON SCHEMA vec_aspirantes FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_aspirantes_propietario REVOKE ALL ON TABLES FROM PUBLIC,vec_aspirantes_ejecutor_externo;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_aspirantes_propietario REVOKE ALL ON SEQUENCES FROM PUBLIC,vec_aspirantes_ejecutor_externo;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_aspirantes_propietario REVOKE ALL ON FUNCTIONS FROM PUBLIC,vec_aspirantes_ejecutor_externo;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_aspirantes_propietario REVOKE ALL ON TYPES FROM PUBLIC,vec_aspirantes_ejecutor_externo;

CREATE FUNCTION vec_aspirantes.rechazar_cambio_inmutable()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN RAISE EXCEPTION 'historia de Aspirantes inmutable' USING ERRCODE='55000'; END $f$;
REVOKE ALL ON FUNCTION vec_aspirantes.rechazar_cambio_inmutable() FROM PUBLIC;

-- La sesión es un LOGIN técnico sin atributos especiales y miembro directo
-- exclusivo del ejecutor externo (INHERIT, sin SET ni ADMIN).
CREATE FUNCTION vec_aspirantes.sesion_valida()
RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT session_user<>'vec_aspirantes_propietario'
  AND EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolcanlogin AND NOT rolsuper AND NOT rolbypassrls
    AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication)
  AND (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)=1
  AND EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
    AND m.roleid='vec_aspirantes_ejecutor_externo'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
  AND NOT EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member='vec_aspirantes_ejecutor_externo'::regrole)
$f$;
REVOKE ALL ON FUNCTION vec_aspirantes.sesion_valida() FROM PUBLIC,vec_aspirantes_ejecutor_externo;

CREATE TABLE vec_aspirantes.ficha (
 aspirante_ref text PRIMARY KEY CHECK(aspirante_ref ~ '^asp_[A-Za-z0-9_-]{22}$'),
 version bigint NOT NULL CHECK(version>0),
 estado text NOT NULL CHECK(estado='activa'),
 creada_en timestamptz(6) NOT NULL,
 actualizada_en timestamptz(6) NOT NULL CHECK(actualizada_en>=creada_en)
);
-- Cada versión de un campo es una fila nueva; el vigente es el de mayor
-- versión. Retirar un dato de contacto añade una fila sin sobre.
CREATE TABLE vec_aspirantes.valor (
 aspirante_ref text NOT NULL REFERENCES vec_aspirantes.ficha(aspirante_ref),
 campo text NOT NULL CHECK(campo IN ('nombre','apellidos','telefono','movil','domicilio','codigo_postal')),
 version bigint NOT NULL CHECK(version>0),
 estado text NOT NULL CHECK(estado IN ('presente','retirado')),
 origen text NOT NULL,
 clave_ref text,
 nonce bytea,
 cifrado bytea,
 registrado_en timestamptz(6) NOT NULL,
 PRIMARY KEY(aspirante_ref,campo,version),
 CHECK((campo IN ('nombre','apellidos') AND origen='certificado' AND estado='presente')
    OR (campo IN ('telefono','movil','domicilio','codigo_postal') AND origen='titular')),
 CHECK((estado='presente' AND clave_ref ~ '^[A-Za-z0-9:._-]{1,128}$' AND octet_length(nonce)=12 AND octet_length(cifrado) BETWEEN 17 AND 1040)
    OR (estado='retirado' AND clave_ref IS NULL AND nonce IS NULL AND cifrado IS NULL))
);
CREATE TABLE vec_aspirantes.documento (
 aspirante_ref text NOT NULL REFERENCES vec_aspirantes.ficha(aspirante_ref),
 documento_ref text PRIMARY KEY CHECK(documento_ref ~ '^aspdoc_[A-Za-z0-9_-]{22}$'),
 tipo text NOT NULL CHECK(tipo IN ('dni','nie','pasaporte','otro')),
 pais text NOT NULL CHECK(pais ~ '^[A-Z]{2}$'),
 clave_ref text NOT NULL CHECK(clave_ref ~ '^[A-Za-z0-9:._-]{1,128}$'),
 nonce bytea NOT NULL CHECK(octet_length(nonce)=12),
 cifrado bytea NOT NULL CHECK(octet_length(cifrado) BETWEEN 19 AND 46),
 clave_indice_ref text NOT NULL CHECK(clave_indice_ref ~ '^[A-Za-z0-9:._-]{1,128}$'),
 indice bytea NOT NULL CHECK(octet_length(indice)=32),
 origen text NOT NULL CHECK(origen='certificado'),
 version_alta bigint NOT NULL CHECK(version_alta>0),
 registrado_en timestamptz(6) NOT NULL,
 CHECK(clave_ref<>clave_indice_ref),
 CHECK((tipo IN ('dni','nie') AND pais='ES') OR tipo='pasaporte' OR (tipo='otro' AND pais<>'ES')),
 UNIQUE(clave_indice_ref,indice),
 UNIQUE(aspirante_ref,documento_ref,clave_indice_ref,indice)
);
-- Índice ciego: encuentra la ficha por documento sin descifrar y hace
-- imposible una segunda ficha con el mismo documento.
CREATE TABLE vec_aspirantes.indice_documento (
 clave_indice_ref text NOT NULL,
 indice bytea NOT NULL CHECK(octet_length(indice)=32),
 aspirante_ref text NOT NULL,
 documento_ref text NOT NULL,
 PRIMARY KEY(clave_indice_ref,indice),
 FOREIGN KEY(aspirante_ref,documento_ref,clave_indice_ref,indice)
  REFERENCES vec_aspirantes.documento(aspirante_ref,documento_ref,clave_indice_ref,indice) DEFERRABLE INITIALLY DEFERRED
);
CREATE TABLE vec_aspirantes.historia (
 aspirante_ref text NOT NULL REFERENCES vec_aspirantes.ficha(aspirante_ref),
 version bigint NOT NULL CHECK(version>0),
 accion text NOT NULL CHECK(accion IN ('vec.aspirantes.ficha.alta','vec.aspirantes.ficha.rectificar')),
 motivo text NOT NULL CHECK(motivo IN ('alta_titular','dato_nuevo','cambio_de_dato','correccion_de_error')),
 campos text[] NOT NULL CHECK(cardinality(campos) BETWEEN 1 AND 7
  AND campos <@ ARRAY['documento','nombre','apellidos','telefono','movil','domicilio','codigo_postal']),
 catalogo_ref text NOT NULL CHECK(catalogo_ref ~ '^[A-Za-z0-9:._-]{1,128}$'),
 recibo_ref text NOT NULL UNIQUE,
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(aspirante_ref,version),
 UNIQUE(aspirante_ref,version,accion,recibo_ref),
 CHECK((accion='vec.aspirantes.ficha.alta' AND motivo='alta_titular' AND version=1)
    OR (accion='vec.aspirantes.ficha.rectificar' AND motivo<>'alta_titular' AND version>1))
);
CREATE TABLE vec_aspirantes.recibo (
 aspirante_ref text NOT NULL,
 clave_operacion text NOT NULL CHECK(clave_operacion ~ '^[A-Za-z0-9:._-]{16,128}$'),
 huella_clave_ref text NOT NULL CHECK(huella_clave_ref ~ '^[A-Za-z0-9:._-]{1,128}$'),
 huella_valor text NOT NULL CHECK(huella_valor ~ '^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^asprec_[0-9a-f]{32}$'),
 accion text NOT NULL,
 version bigint NOT NULL,
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(aspirante_ref,clave_operacion),
 FOREIGN KEY(aspirante_ref,version,accion,recibo_ref)
  REFERENCES vec_aspirantes.historia(aspirante_ref,version,accion,recibo_ref) DEFERRABLE INITIALLY DEFERRED
);
-- Registro de lecturas en claro: quién (tipo de actor y decisión V3), para
-- qué, qué campos y cuándo. Solo referencias; nunca valores.
CREATE TABLE vec_aspirantes.acceso (
 acceso_ref text PRIMARY KEY CHECK(acceso_ref ~ '^aspacc_[0-9a-f]{32}$'),
 aspirante_ref text NOT NULL REFERENCES vec_aspirantes.ficha(aspirante_ref),
 finalidad text NOT NULL CHECK(finalidad='consulta_propia'),
 actor_tipo text NOT NULL CHECK(actor_tipo='titular'),
 campos text[] NOT NULL CHECK(cardinality(campos) BETWEEN 1 AND 7
  AND campos <@ ARRAY['documento','nombre','apellidos','telefono','movil','domicilio','codigo_postal']),
 resultado text NOT NULL CHECK(resultado='entregado'),
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 ocurrido_en timestamptz(6) NOT NULL
);
-- Hechos para otros módulos (Bolsa, cuando use asp_). Solo referencias.
CREATE TABLE vec_aspirantes.evento_salida (
 evento_ref text PRIMARY KEY CHECK(evento_ref ~ '^aspevt_[0-9a-f]{32}$'),
 aspirante_ref text NOT NULL REFERENCES vec_aspirantes.ficha(aspirante_ref),
 tipo text NOT NULL CHECK(tipo IN ('ficha.alta','ficha.contacto_rectificado')),
 version bigint NOT NULL CHECK(version>0),
 recibo_ref text NOT NULL REFERENCES vec_aspirantes.historia(recibo_ref),
 estado text NOT NULL CHECK(estado='pendiente'),
 creado_en timestamptz(6) NOT NULL,
 UNIQUE(aspirante_ref,version)
);
-- Clave del índice ciego en uso. La primera alta la fija; una clave distinta
-- (rotación sin reindexar) cierra todas las operaciones con 55000: si no,
-- tras rotar, la misma persona obtendría «sin ficha» y podría crear otra.
CREATE TABLE vec_aspirantes.clave_indice (
 unica boolean PRIMARY KEY DEFAULT true CHECK(unica),
 clave_indice_ref text NOT NULL CHECK(clave_indice_ref ~ '^[A-Za-z0-9:._-]{1,128}$'),
 fijada_en timestamptz(6) NOT NULL
);
-- Marcador de la transacción autorizada. Se crea tras consumir V3 y se
-- retira antes de devolver; un ROLLBACK lo deshace.
CREATE TABLE vec_aspirantes.contexto (
 xid xid8 NOT NULL,
 backend_pid integer NOT NULL,
 sesion text NOT NULL,
 modo text NOT NULL CHECK(modo IN ('consultar','alta','rectificar')),
 accion text NOT NULL,
 indice_clave_ref text NOT NULL,
 indice bytea NOT NULL,
 clave_operacion text,
 version_esperada bigint NOT NULL,
 huella_clave_ref text,
 huella_valor text,
 aspirante_ref text,
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL,
 PRIMARY KEY(xid,backend_pid,sesion)
);
-- Un marcador nunca sobrevive a su transacción: si al confirmar queda uno
-- (p. ej. preparar sin aplicar), la confirmación falla y todo se deshace.
CREATE FUNCTION vec_aspirantes.contexto_cerrado()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_aspirantes.contexto c WHERE c.xid=NEW.xid AND c.backend_pid=NEW.backend_pid AND c.sesion=NEW.sesion)
 THEN RAISE EXCEPTION 'Aspirantes: operación sin cerrar' USING ERRCODE='42501'; END IF;
 RETURN NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_aspirantes.contexto_cerrado() FROM PUBLIC,vec_aspirantes_ejecutor_externo;
CREATE CONSTRAINT TRIGGER contexto_sin_cerrar AFTER INSERT ON vec_aspirantes.contexto
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION vec_aspirantes.contexto_cerrado();
DO $rls$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['ficha','valor','documento','indice_documento','historia','recibo','acceso','evento_salida','contexto','clave_indice'] LOOP
  EXECUTE format('ALTER TABLE vec_aspirantes.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_aspirantes.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_aspirantes.%I FROM PUBLIC,vec_aspirantes_ejecutor_externo',t);
 END LOOP;
END $rls$;
GRANT USAGE ON SCHEMA vec_aspirantes TO vec_aspirantes_ejecutor_externo;

-- Filas inmutables: historia, sobres, índices, recibos, accesos y eventos.
DO $inmutable$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['valor','documento','indice_documento','historia','recibo','acceso','evento_salida','clave_indice'] LOOP
  EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE OR DELETE ON vec_aspirantes.%I FOR EACH ROW EXECUTE FUNCTION vec_aspirantes.rechazar_cambio_inmutable()',t||'_inmutable',t);
  EXECUTE format('CREATE TRIGGER %I BEFORE TRUNCATE ON vec_aspirantes.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_aspirantes.rechazar_cambio_inmutable()',t||'_no_truncar',t);
 END LOOP;
END $inmutable$;
CREATE TRIGGER ficha_no_truncar BEFORE TRUNCATE ON vec_aspirantes.ficha FOR EACH STATEMENT EXECUTE FUNCTION vec_aspirantes.rechazar_cambio_inmutable();
-- La ficha solo avanza de versión de uno en uno; nunca se borra.
CREATE FUNCTION vec_aspirantes.ficha_transicion_valida()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 IF TG_OP='DELETE' OR NEW.aspirante_ref<>OLD.aspirante_ref OR NEW.estado<>OLD.estado OR NEW.creada_en<>OLD.creada_en
    OR NEW.version<>OLD.version+1 OR NEW.actualizada_en<OLD.actualizada_en
 THEN RAISE EXCEPTION 'transición de ficha inválida' USING ERRCODE='55000'; END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_aspirantes.ficha_transicion_valida() FROM PUBLIC;
CREATE TRIGGER ficha_transicion BEFORE UPDATE OR DELETE ON vec_aspirantes.ficha FOR EACH ROW EXECUTE FUNCTION vec_aspirantes.ficha_transicion_valida();

-- Políticas: una fila solo se ve o se escribe si el marcador de la
-- transacción autorizada apunta a su ficha o a su índice.
CREATE FUNCTION vec_aspirantes.contexto_permite_ficha(p_asp text,p_modos text[])
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
 SELECT vec_aspirantes.sesion_valida() AND EXISTS(SELECT 1 FROM vec_aspirantes.contexto c
  WHERE c.xid=pg_current_xact_id_if_assigned() AND c.backend_pid=pg_backend_pid() AND c.sesion=session_user
    AND c.modo=ANY(p_modos) AND c.aspirante_ref=p_asp)
$f$;
REVOKE ALL ON FUNCTION vec_aspirantes.contexto_permite_ficha(text,text[]) FROM PUBLIC,vec_aspirantes_ejecutor_externo;
CREATE FUNCTION vec_aspirantes.contexto_permite_indice(p_clave text,p_indice bytea,p_modos text[])
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
 SELECT vec_aspirantes.sesion_valida() AND EXISTS(SELECT 1 FROM vec_aspirantes.contexto c
  WHERE c.xid=pg_current_xact_id_if_assigned() AND c.backend_pid=pg_backend_pid() AND c.sesion=session_user
    AND c.modo=ANY(p_modos) AND c.indice_clave_ref=p_clave AND c.indice=p_indice)
$f$;
REVOKE ALL ON FUNCTION vec_aspirantes.contexto_permite_indice(text,bytea,text[]) FROM PUBLIC,vec_aspirantes_ejecutor_externo;
CREATE POLICY contexto_propio ON vec_aspirantes.contexto FOR ALL TO vec_aspirantes_propietario
 USING (xid=pg_current_xact_id_if_assigned() AND backend_pid=pg_backend_pid() AND sesion=session_user)
 WITH CHECK (xid=pg_current_xact_id_if_assigned() AND backend_pid=pg_backend_pid() AND sesion=session_user AND vec_aspirantes.sesion_valida());
CREATE POLICY clave_indice_lectura ON vec_aspirantes.clave_indice FOR SELECT TO vec_aspirantes_propietario USING (vec_aspirantes.sesion_valida());
CREATE POLICY clave_indice_alta ON vec_aspirantes.clave_indice FOR INSERT TO vec_aspirantes_propietario
 WITH CHECK (vec_aspirantes.sesion_valida() AND EXISTS(SELECT 1 FROM vec_aspirantes.contexto c WHERE c.xid=pg_current_xact_id_if_assigned()
  AND c.backend_pid=pg_backend_pid() AND c.sesion=session_user AND c.modo='alta' AND c.indice_clave_ref=clave_indice_ref));
CREATE POLICY indice_lectura ON vec_aspirantes.indice_documento FOR SELECT TO vec_aspirantes_propietario
 USING (vec_aspirantes.contexto_permite_indice(clave_indice_ref,indice,ARRAY['consultar','alta','rectificar']));
CREATE POLICY indice_alta ON vec_aspirantes.indice_documento FOR INSERT TO vec_aspirantes_propietario
 WITH CHECK (vec_aspirantes.contexto_permite_indice(clave_indice_ref,indice,ARRAY['alta']) AND vec_aspirantes.contexto_permite_ficha(aspirante_ref,ARRAY['alta']));
CREATE POLICY ficha_lectura ON vec_aspirantes.ficha FOR SELECT TO vec_aspirantes_propietario USING (vec_aspirantes.contexto_permite_ficha(aspirante_ref,ARRAY['consultar','alta','rectificar']));
CREATE POLICY ficha_alta ON vec_aspirantes.ficha FOR INSERT TO vec_aspirantes_propietario WITH CHECK (vec_aspirantes.contexto_permite_ficha(aspirante_ref,ARRAY['alta']));
CREATE POLICY ficha_cambio ON vec_aspirantes.ficha FOR UPDATE TO vec_aspirantes_propietario
 USING (vec_aspirantes.contexto_permite_ficha(aspirante_ref,ARRAY['rectificar'])) WITH CHECK (vec_aspirantes.contexto_permite_ficha(aspirante_ref,ARRAY['rectificar']));
CREATE POLICY valor_lectura ON vec_aspirantes.valor FOR SELECT TO vec_aspirantes_propietario USING (vec_aspirantes.contexto_permite_ficha(aspirante_ref,ARRAY['consultar','rectificar']));
CREATE POLICY valor_alta ON vec_aspirantes.valor FOR INSERT TO vec_aspirantes_propietario WITH CHECK (vec_aspirantes.contexto_permite_ficha(aspirante_ref,ARRAY['alta','rectificar']));
CREATE POLICY documento_lectura ON vec_aspirantes.documento FOR SELECT TO vec_aspirantes_propietario USING (vec_aspirantes.contexto_permite_ficha(aspirante_ref,ARRAY['consultar']));
CREATE POLICY documento_alta ON vec_aspirantes.documento FOR INSERT TO vec_aspirantes_propietario WITH CHECK (vec_aspirantes.contexto_permite_ficha(aspirante_ref,ARRAY['alta']));
CREATE POLICY historia_alta ON vec_aspirantes.historia FOR INSERT TO vec_aspirantes_propietario WITH CHECK (vec_aspirantes.contexto_permite_ficha(aspirante_ref,ARRAY['alta','rectificar']));
CREATE POLICY recibo_lectura ON vec_aspirantes.recibo FOR SELECT TO vec_aspirantes_propietario USING (vec_aspirantes.contexto_permite_ficha(aspirante_ref,ARRAY['alta','rectificar']));
CREATE POLICY recibo_alta ON vec_aspirantes.recibo FOR INSERT TO vec_aspirantes_propietario WITH CHECK (vec_aspirantes.contexto_permite_ficha(aspirante_ref,ARRAY['alta','rectificar']));
CREATE POLICY acceso_alta ON vec_aspirantes.acceso FOR INSERT TO vec_aspirantes_propietario WITH CHECK (vec_aspirantes.contexto_permite_ficha(aspirante_ref,ARRAY['consultar']));
CREATE POLICY evento_alta ON vec_aspirantes.evento_salida FOR INSERT TO vec_aspirantes_propietario WITH CHECK (vec_aspirantes.contexto_permite_ficha(aspirante_ref,ARRAY['alta','rectificar']));

-- Huella canónica del recurso V3: la misma que calcula Go
-- (RecursoAutorizable.HuellaContextoAutorizacionSHA256).
CREATE FUNCTION vec_aspirantes.huella_contexto(p_material text)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE persona text; canon text;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 8192
 THEN RAISE EXCEPTION 'Aspirantes: material inválido' USING ERRCODE='22023'; END IF;
 BEGIN persona:=p_material::jsonb->>'persona_ref';
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Aspirantes: material inválido' USING ERRCODE='22023'; END;
 IF persona IS NULL OR persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
 THEN RAISE EXCEPTION 'Aspirantes: persona inválida' USING ERRCODE='22023'; END IF;
 canon:='{"ambitos":{"persona_ref":'||to_jsonb(persona)::text||'},"atributos":{"material_sha256":"'||
  encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}';
 RETURN encode(sha256(convert_to(canon,'UTF8')),'hex');
END $f$;
REVOKE ALL ON FUNCTION vec_aspirantes.huella_contexto(text) FROM PUBLIC,vec_aspirantes_ejecutor_externo;

-- p_material es el JSON literal que serializa Go (canonico.SerializarMaterial).
CREATE FUNCTION vec_aspirantes.validar_material(p_material text,p_capacidad bytea,p_decision bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE m jsonb; c jsonb; d jsonb; k text[]; h text; v_accion text; segmento text; campos jsonb; sello jsonb; claves text[];
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR NOT vec_aspirantes.sesion_valida()
    OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 8192 OR p_capacidad IS NULL OR p_decision IS NULL
 THEN RAISE EXCEPTION 'Aspirantes: material denegado' USING ERRCODE='42501'; END IF;
 BEGIN m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Aspirantes: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'Aspirantes: material inválido' USING ERRCODE='22023'; END IF;
 SELECT array_agg(x ORDER BY x) INTO k FROM jsonb_object_keys(m) x;
 v_accion:=m->>'accion';
 SELECT q.segmento,q.campos INTO segmento,campos FROM (VALUES
 ('vec.aspirantes.ficha.consultar','consultar','["apellidos","codigo_postal","documento","domicilio","movil","nombre","telefono","version"]'::jsonb),
 ('vec.aspirantes.ficha.alta','alta','["apellidos","codigo_postal","documento","domicilio","movil","nombre","telefono","version"]'::jsonb),
 ('vec.aspirantes.ficha.rectificar','rectificar','["codigo_postal","domicilio","movil","telefono","version"]'::jsonb)
 ) q(accion,segmento,campos) WHERE q.accion=v_accion;
 h:=vec_aspirantes.huella_contexto(p_material);
 IF k IS DISTINCT FROM ARRAY['accion','clave_operacion','finalidad_ref','huellas_peticion','indice_documento','perfil_ref','persona_ref','superficie','version_esperada']
    OR EXISTS(SELECT 1 FROM jsonb_each(m) z WHERE z.key IN ('accion','clave_operacion','finalidad_ref','perfil_ref','persona_ref','superficie')
      AND jsonb_typeof(z.value)<>'string')
    OR jsonb_typeof(m->'version_esperada') IS DISTINCT FROM 'number'
    OR jsonb_typeof(m->'huellas_peticion') IS DISTINCT FROM 'object'
    OR jsonb_typeof(m->'indice_documento') IS DISTINCT FROM 'object'
    OR segmento IS NULL
    OR m->>'superficie' IS DISTINCT FROM 'externa_personal'
    OR d #>> '{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'externa_personal'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_aspirantes.ficha.'||segmento||'.externa_personal.v1'
    OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR m->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
    OR m->>'finalidad_ref' IS DISTINCT FROM 'finalidad:aspirantes:ficha-propia:v1'
    OR m->>'persona_ref' IS DISTINCT FROM d->>'principal_id'
    OR m->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
    OR m->>'persona_ref' IS DISTINCT FROM d->>'recurso_ref'
    OR m->>'persona_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'accion' IS DISTINCT FROM v_accion OR c->>'operacion' IS DISTINCT FROM v_accion
    OR d->>'modulo_id' IS DISTINCT FROM 'aspirantes' OR d->>'tipo_recurso' IS DISTINCT FROM 'ficha_aspirante_propia'
    OR d->>'finalidad' IS DISTINCT FROM 'finalidad:aspirantes:ficha-propia:v1' OR d->'concedida' IS DISTINCT FROM 'true'::jsonb
    OR d->'campos_permitidos' IS DISTINCT FROM campos OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'decision_ref' IS NULL OR length(d->>'decision_ref') NOT BETWEEN 1 AND 256
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h OR c->>'huella_efecto_sha256' IS DISTINCT FROM h
    OR m->>'version_esperada' !~ '^(0|[1-9][0-9]{0,17})$'
    OR (SELECT array_agg(x ORDER BY x) FROM jsonb_object_keys(m->'indice_documento') x) IS DISTINCT FROM ARRAY['clave_ref','valor']
    OR jsonb_typeof(m#>'{indice_documento,clave_ref}') IS DISTINCT FROM 'string'
    OR jsonb_typeof(m#>'{indice_documento,valor}') IS DISTINCT FROM 'string'
    OR m#>>'{indice_documento,clave_ref}' !~ '^[A-Za-z0-9:._-]{1,128}$'
    OR m#>>'{indice_documento,valor}' !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'Aspirantes: ficha no autorizada' USING ERRCODE='42501'; END IF;
 IF v_accion='vec.aspirantes.ficha.consultar' THEN
  IF m->>'clave_operacion'<>'' OR m->'huellas_peticion' IS DISTINCT FROM '{}'::jsonb OR (m->>'version_esperada')::bigint<>0
  THEN RAISE EXCEPTION 'Aspirantes: consulta inválida' USING ERRCODE='22023'; END IF;
 ELSE
  IF m->>'clave_operacion' !~ '^[A-Za-z0-9:._-]{16,128}$'
     OR (v_accion='vec.aspirantes.ficha.alta' AND (m->>'version_esperada')::bigint<>0)
     OR (v_accion='vec.aspirantes.ficha.rectificar' AND (m->>'version_esperada')::bigint<1)
     OR (SELECT array_agg(x ORDER BY x) FROM jsonb_object_keys(m->'huellas_peticion') x) IS DISTINCT FROM ARRAY['activa','retenidas']
     OR jsonb_typeof(m#>'{huellas_peticion,retenidas}') IS DISTINCT FROM 'array'
     OR jsonb_array_length(m#>'{huellas_peticion,retenidas}')>8
  THEN RAISE EXCEPTION 'Aspirantes: mutación inválida' USING ERRCODE='22023'; END IF;
  claves:=ARRAY[]::text[];
  FOR sello IN SELECT value FROM jsonb_array_elements(jsonb_build_array(m#>'{huellas_peticion,activa}') || (m#>'{huellas_peticion,retenidas}')) LOOP
   IF jsonb_typeof(sello) IS DISTINCT FROM 'object'
      OR (SELECT array_agg(x ORDER BY x) FROM jsonb_object_keys(sello) x) IS DISTINCT FROM ARRAY['clave_ref','valor']
      OR jsonb_typeof(sello->'clave_ref') IS DISTINCT FROM 'string' OR jsonb_typeof(sello->'valor') IS DISTINCT FROM 'string'
      OR sello->>'clave_ref' !~ '^[A-Za-z0-9:._-]{1,128}$' OR sello->>'valor' !~ '^[0-9a-f]{64}$' OR sello->>'clave_ref'=ANY(claves)
   THEN RAISE EXCEPTION 'Aspirantes: huella inválida' USING ERRCODE='22023'; END IF;
   claves:=array_append(claves,sello->>'clave_ref');
  END LOOP;
 END IF;
 RETURN m;
END $f$;
REVOKE ALL ON FUNCTION vec_aspirantes.validar_material(text,bytea,bytea) FROM PUBLIC,vec_aspirantes_ejecutor_externo;

CREATE FUNCTION vec_aspirantes.retirar_contexto(p_modo text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE n integer;
BEGIN
 DELETE FROM vec_aspirantes.contexto WHERE xid=pg_current_xact_id_if_assigned()
  AND backend_pid=pg_backend_pid() AND sesion=session_user AND modo=p_modo;
 GET DIAGNOSTICS n=ROW_COUNT;
 IF n<>1 THEN RAISE EXCEPTION 'Aspirantes: contexto incompleto' USING ERRCODE='42501'; END IF;
END $f$;
REVOKE ALL ON FUNCTION vec_aspirantes.retirar_contexto(text) FROM PUBLIC,vec_aspirantes_ejecutor_externo;

-- Consume V3 fresca antes de mirar ninguna fila y abre el marcador.
CREATE FUNCTION vec_aspirantes.consumir_contexto(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_modo text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE m jsonb; x record; xid_actual xid8;
BEGIN
 m:=vec_aspirantes.validar_material(p_material,p_capacidad,p_decision);
 IF p_modo IS DISTINCT FROM (SELECT q.modo FROM (VALUES ('vec.aspirantes.ficha.consultar','consultar'),
      ('vec.aspirantes.ficha.alta','alta'),('vec.aspirantes.ficha.rectificar','rectificar')) q(accion,modo) WHERE q.accion=m->>'accion')
 THEN RAISE EXCEPTION 'Aspirantes: acción fuera de su operación' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_aspirantes_v3_atestada(
  m->>'accion',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.efecto_ref IS DISTINCT FROM m->>'persona_ref'
    OR x.decision_ref IS DISTINCT FROM convert_from(p_decision,'UTF8')::jsonb->>'decision_ref'
    OR x.huella_efecto_sha256 IS DISTINCT FROM vec_aspirantes.huella_contexto(p_material)
    OR x.consumo_huella_sha256 !~ '^[0-9a-f]{64}$' OR x.auditoria_ref IS NULL
 THEN RAISE EXCEPTION 'Aspirantes: consumo divergente' USING ERRCODE='42501'; END IF;
 IF EXISTS(SELECT 1 FROM vec_aspirantes.clave_indice k WHERE k.clave_indice_ref IS DISTINCT FROM m#>>'{indice_documento,clave_ref}')
 THEN RAISE EXCEPTION 'Aspirantes: clave del índice distinta de la fijada; reindexado pendiente' USING ERRCODE='55000'; END IF;
 xid_actual:=pg_current_xact_id();
 IF EXISTS(SELECT 1 FROM vec_aspirantes.contexto WHERE xid=xid_actual AND backend_pid=pg_backend_pid() AND sesion=session_user)
 THEN RAISE EXCEPTION 'Aspirantes: contexto ya activo' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_aspirantes.contexto(xid,backend_pid,sesion,modo,accion,indice_clave_ref,indice,clave_operacion,version_esperada,
  huella_clave_ref,huella_valor,aspirante_ref,decision_ref,auditoria_ref,consumo_huella_sha256)
 VALUES(xid_actual,pg_backend_pid(),session_user,p_modo,m->>'accion',m#>>'{indice_documento,clave_ref}',
  decode(m#>>'{indice_documento,valor}','hex'),nullif(m->>'clave_operacion',''),(m->>'version_esperada')::bigint,
  m#>>'{huellas_peticion,activa,clave_ref}',m#>>'{huellas_peticion,activa,valor}',NULL,x.decision_ref,x.auditoria_ref,x.consumo_huella_sha256);
 RETURN m;
END $f$;
REVOKE ALL ON FUNCTION vec_aspirantes.consumir_contexto(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text) FROM PUBLIC,vec_aspirantes_ejecutor_externo;

-- Localiza la ficha por el índice del marcador y la fija en él.
CREATE FUNCTION vec_aspirantes.fijar_ficha_por_indice()
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE asp text; n integer;
BEGIN
 SELECT i.aspirante_ref INTO asp FROM vec_aspirantes.indice_documento i
 JOIN vec_aspirantes.contexto c ON c.indice_clave_ref=i.clave_indice_ref AND c.indice=i.indice
 WHERE c.xid=pg_current_xact_id_if_assigned() AND c.backend_pid=pg_backend_pid() AND c.sesion=session_user AND c.aspirante_ref IS NULL;
 IF asp IS NOT NULL THEN
  UPDATE vec_aspirantes.contexto SET aspirante_ref=asp WHERE xid=pg_current_xact_id_if_assigned()
   AND backend_pid=pg_backend_pid() AND sesion=session_user AND aspirante_ref IS NULL;
  GET DIAGNOSTICS n=ROW_COUNT;
  IF n<>1 THEN RAISE EXCEPTION 'Aspirantes: contexto incompleto' USING ERRCODE='42501'; END IF;
 END IF;
 RETURN asp;
END $f$;
REVOKE ALL ON FUNCTION vec_aspirantes.fijar_ficha_por_indice() FROM PUBLIC,vec_aspirantes_ejecutor_externo;

CREATE FUNCTION vec_aspirantes.contexto_vigente(p_modo text)
RETURNS vec_aspirantes.contexto LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE c vec_aspirantes.contexto;
BEGIN
 SELECT * INTO c FROM vec_aspirantes.contexto WHERE xid=pg_current_xact_id_if_assigned()
  AND backend_pid=pg_backend_pid() AND sesion=session_user AND modo=p_modo;
 IF NOT FOUND OR NOT vec_aspirantes.sesion_valida() THEN RAISE EXCEPTION 'Aspirantes: sin contexto' USING ERRCODE='42501'; END IF;
 RETURN c;
END $f$;
REVOKE ALL ON FUNCTION vec_aspirantes.contexto_vigente(text) FROM PUBLIC,vec_aspirantes_ejecutor_externo;

CREATE FUNCTION vec_aspirantes.sobre_json(p_clave text,p_nonce bytea,p_cifrado bytea)
RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT CASE WHEN p_cifrado IS NULL THEN NULL ELSE
  jsonb_build_object('clave_ref',p_clave,'nonce_hex',encode(p_nonce,'hex'),'cifrado_hex',encode(p_cifrado,'hex')) END
$f$;
REVOKE ALL ON FUNCTION vec_aspirantes.sobre_json(text,bytea,bytea) FROM PUBLIC,vec_aspirantes_ejecutor_externo;

-- Comprueba un sobre recibido de Go: clave, nonce de 12 bytes y cifrado
-- entre 17 bytes (un carácter y la etiqueta) y p_maximo.
CREATE FUNCTION vec_aspirantes.sobre_valido(p jsonb,p_maximo integer)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT jsonb_typeof(p->'clave_ref')='string' AND jsonb_typeof(p->'nonce_hex')='string' AND jsonb_typeof(p->'cifrado_hex')='string'
  AND p->>'clave_ref' ~ '^[A-Za-z0-9:._-]{1,128}$' AND p->>'nonce_hex' ~ '^[0-9a-f]{24}$'
  AND p->>'cifrado_hex' ~ '^([0-9a-f]{2})+$' AND length(p->>'cifrado_hex') BETWEEN 34 AND 2*p_maximo
$f$;
REVOKE ALL ON FUNCTION vec_aspirantes.sobre_valido(jsonb,integer) FROM PUBLIC,vec_aspirantes_ejecutor_externo;

-- Repetición por ficha y clave: devuelve el recibo original; la misma
-- clave con otra huella semántica o con otra acción es conflicto.
CREATE FUNCTION vec_aspirantes.replay(p_m jsonb,p_asp text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE r record;
BEGIN
 SELECT * INTO r FROM vec_aspirantes.recibo WHERE aspirante_ref=p_asp AND clave_operacion=p_m->>'clave_operacion';
 IF NOT FOUND THEN RETURN NULL; END IF;
 IF r.accion IS DISTINCT FROM p_m->>'accion' OR NOT EXISTS(
    SELECT 1 FROM jsonb_array_elements(jsonb_build_array(p_m#>'{huellas_peticion,activa}') || (p_m#>'{huellas_peticion,retenidas}')) h
    WHERE h->>'clave_ref'=r.huella_clave_ref AND h->>'valor'=r.huella_valor)
 THEN RAISE EXCEPTION 'Aspirantes: clave reutilizada con otra petición' USING ERRCODE='P1409'; END IF;
 RETURN jsonb_build_object('recibo_ref',r.recibo_ref,'accion',r.accion,'version',r.version,'fecha_utc',r.registrada_en,'replay',true);
END $f$;
REVOKE ALL ON FUNCTION vec_aspirantes.replay(jsonb,text) FROM PUBLIC,vec_aspirantes_ejecutor_externo;

CREATE FUNCTION vec_aspirantes.registrar_recibo(p_c vec_aspirantes.contexto,p_asp text,p_version bigint,p_motivo text,
 p_campos text[],p_catalogo text,p_tipo_evento text,p_fecha timestamptz)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE ref text;
BEGIN
 IF p_c.clave_operacion IS NULL OR p_c.huella_clave_ref IS NULL OR p_c.huella_valor IS NULL OR p_c.aspirante_ref IS DISTINCT FROM p_asp
 THEN RAISE EXCEPTION 'Aspirantes: recibo sin contexto' USING ERRCODE='42501'; END IF;
 ref:='asprec_'||replace(gen_random_uuid()::text,'-','');
 INSERT INTO vec_aspirantes.historia(aspirante_ref,version,accion,motivo,campos,catalogo_ref,recibo_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(p_asp,p_version,p_c.accion,p_motivo,p_campos,p_catalogo,ref,p_c.decision_ref,p_c.auditoria_ref,p_fecha);
 INSERT INTO vec_aspirantes.recibo(aspirante_ref,clave_operacion,huella_clave_ref,huella_valor,recibo_ref,accion,version,decision_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
 VALUES(p_asp,p_c.clave_operacion,p_c.huella_clave_ref,p_c.huella_valor,ref,p_c.accion,p_version,p_c.decision_ref,p_c.auditoria_ref,p_c.consumo_huella_sha256,p_fecha);
 INSERT INTO vec_aspirantes.evento_salida(evento_ref,aspirante_ref,tipo,version,recibo_ref,estado,creado_en)
 VALUES('aspevt_'||replace(gen_random_uuid()::text,'-',''),p_asp,p_tipo_evento,p_version,ref,'pendiente',p_fecha);
 RETURN jsonb_build_object('recibo_ref',ref,'accion',p_c.accion,'version',p_version,'fecha_utc',p_fecha,'replay',false);
END $f$;
REVOKE ALL ON FUNCTION vec_aspirantes.registrar_recibo(vec_aspirantes.contexto,text,bigint,text,text[],text,text,timestamptz) FROM PUBLIC,vec_aspirantes_ejecutor_externo;

-- Vista de la finalidad «consulta propia»: el titular recibe su ficha y la
-- lectura queda anotada. «entregado» significa entregado al portal, que
-- descifra después. Sin ficha no se anota nada: no se leyó nada.
CREATE FUNCTION vec_aspirantes.consultar_ficha_propia_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; c vec_aspirantes.contexto; asp text; f record; doc record; valores jsonb; campos text[]; acc text;
BEGIN
 m:=vec_aspirantes.consumir_contexto(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'consultar');
 asp:=vec_aspirantes.fijar_ficha_por_indice();
 IF asp IS NULL THEN
  PERFORM vec_aspirantes.retirar_contexto('consultar');
  RETURN jsonb_build_object('estado','sin_ficha');
 END IF;
 c:=vec_aspirantes.contexto_vigente('consultar');
 SELECT * INTO STRICT f FROM vec_aspirantes.ficha WHERE aspirante_ref=asp;
 SELECT d.* INTO STRICT doc FROM vec_aspirantes.documento d
 WHERE d.aspirante_ref=asp AND d.clave_indice_ref=c.indice_clave_ref AND d.indice=c.indice;
 SELECT coalesce(jsonb_agg(jsonb_build_object('campo',v.campo,'version',v.version,'origen',v.origen,
   'sobre',vec_aspirantes.sobre_json(v.clave_ref,v.nonce,v.cifrado)) ORDER BY v.campo),'[]'::jsonb),
  array['documento']||coalesce(array_agg(v.campo ORDER BY v.campo) FILTER (WHERE v.estado='presente'),ARRAY[]::text[])
 INTO valores,campos
 FROM (SELECT DISTINCT ON (x.campo) x.* FROM vec_aspirantes.valor x WHERE x.aspirante_ref=asp ORDER BY x.campo,x.version DESC) v;
 acc:='aspacc_'||replace(gen_random_uuid()::text,'-','');
 INSERT INTO vec_aspirantes.acceso(acceso_ref,aspirante_ref,finalidad,actor_tipo,campos,resultado,decision_ref,auditoria_ref,consumo_huella_sha256,ocurrido_en)
 VALUES(acc,asp,'consulta_propia','titular',campos,'entregado',c.decision_ref,c.auditoria_ref,c.consumo_huella_sha256,date_trunc('microseconds',clock_timestamp()));
 PERFORM vec_aspirantes.retirar_contexto('consultar');
 RETURN jsonb_build_object('estado','activa','aspirante_ref',asp,'version',f.version,'acceso_ref',acc,
  'documento',jsonb_build_object('documento_ref',doc.documento_ref,'tipo',doc.tipo,'pais',doc.pais,
   'sobre',vec_aspirantes.sobre_json(doc.clave_ref,doc.nonce,doc.cifrado),
   'indice',jsonb_build_object('clave_ref',doc.clave_indice_ref,'valor',encode(doc.indice,'hex'))),
  'valores',valores);
END $f$;
REVOKE ALL ON FUNCTION vec_aspirantes.consultar_ficha_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_aspirantes.consultar_ficha_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_aspirantes_ejecutor_externo;

-- Alta por el titular. p_ficha llega cifrada desde Go:
-- {aspirante_ref, catalogo_ref, documento:{documento_ref,tipo,pais,clave_ref,
--  nonce_hex,cifrado_hex,indice:{clave_ref,valor}}, valores:[{campo,version,
--  origen,clave_ref,nonce_hex,cifrado_hex}]}. Un documento que ya tiene ficha
-- devuelve el recibo original si es la misma operación o P1411 si no.
CREATE FUNCTION vec_aspirantes.alta_ficha_propia_v1(p_material text,p_ficha jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; c vec_aspirantes.contexto; existente text; asp text; r jsonb; doc jsonb; v jsonb; ahora timestamptz(6);
 campos text[]; n integer;
BEGIN
 m:=vec_aspirantes.consumir_contexto(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'alta');
 -- El candado global solo se toma mientras no hay clave fijada.
 IF NOT EXISTS(SELECT 1 FROM vec_aspirantes.clave_indice) THEN
  PERFORM pg_advisory_xact_lock(hashtextextended('vec_aspirantes:clave_indice',0));
  INSERT INTO vec_aspirantes.clave_indice(unica,clave_indice_ref,fijada_en)
  VALUES(true,m#>>'{indice_documento,clave_ref}',date_trunc('microseconds',clock_timestamp())) ON CONFLICT (unica) DO NOTHING;
 END IF;
 IF (SELECT clave_indice_ref FROM vec_aspirantes.clave_indice) IS DISTINCT FROM m#>>'{indice_documento,clave_ref}'
 THEN RAISE EXCEPTION 'Aspirantes: clave del índice distinta de la fijada' USING ERRCODE='55000'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_aspirantes:indice:'||(m#>>'{indice_documento,clave_ref}')||':'||(m#>>'{indice_documento,valor}'),0));
 existente:=vec_aspirantes.fijar_ficha_por_indice();
 IF existente IS NOT NULL THEN
  r:=vec_aspirantes.replay(m,existente);
  PERFORM vec_aspirantes.retirar_contexto('alta');
  IF r IS NULL THEN RAISE EXCEPTION 'Aspirantes: ficha ya existente' USING ERRCODE='P1411'; END IF;
  RETURN r;
 END IF;
 doc:=p_ficha->'documento';
 IF jsonb_typeof(p_ficha) IS DISTINCT FROM 'object'
    OR (SELECT array_agg(x ORDER BY x) FROM jsonb_object_keys(p_ficha) x) IS DISTINCT FROM ARRAY['aspirante_ref','catalogo_ref','documento','valores']
    OR jsonb_typeof(p_ficha->'aspirante_ref') IS DISTINCT FROM 'string' OR p_ficha->>'aspirante_ref' !~ '^asp_[A-Za-z0-9_-]{22}$'
    OR jsonb_typeof(p_ficha->'catalogo_ref') IS DISTINCT FROM 'string' OR p_ficha->>'catalogo_ref' !~ '^[A-Za-z0-9:._-]{1,128}$'
    OR jsonb_typeof(doc) IS DISTINCT FROM 'object'
    OR (SELECT array_agg(x ORDER BY x) FROM jsonb_object_keys(doc) x) IS DISTINCT FROM ARRAY['cifrado_hex','clave_ref','documento_ref','indice','nonce_hex','pais','tipo']
    OR doc->'indice' IS DISTINCT FROM m->'indice_documento'
    OR jsonb_typeof(doc->'documento_ref') IS DISTINCT FROM 'string' OR doc->>'documento_ref' !~ '^aspdoc_[A-Za-z0-9_-]{22}$'
    OR jsonb_typeof(doc->'tipo') IS DISTINCT FROM 'string' OR jsonb_typeof(doc->'pais') IS DISTINCT FROM 'string'
    OR NOT vec_aspirantes.sobre_valido(doc,46) OR length(doc->>'cifrado_hex')<38
    OR NOT ((doc->>'tipo' IN ('dni','nie') AND doc->>'pais'='ES')
         OR (doc->>'tipo'='pasaporte' AND doc->>'pais' ~ '^[A-Z]{2}$')
         OR (doc->>'tipo'='otro' AND doc->>'pais' ~ '^[A-Z]{2}$' AND doc->>'pais'<>'ES'))
    OR doc->>'clave_ref'=m#>>'{indice_documento,clave_ref}'
    OR jsonb_typeof(p_ficha->'valores') IS DISTINCT FROM 'array'
    OR jsonb_array_length(p_ficha->'valores') NOT BETWEEN 2 AND 6
 THEN RAISE EXCEPTION 'Aspirantes: ficha inválida' USING ERRCODE='22023'; END IF;
 FOR v IN SELECT value FROM jsonb_array_elements(p_ficha->'valores') LOOP
  IF jsonb_typeof(v) IS DISTINCT FROM 'object'
     OR (SELECT array_agg(x ORDER BY x) FROM jsonb_object_keys(v) x) IS DISTINCT FROM ARRAY['campo','cifrado_hex','clave_ref','nonce_hex','origen','version']
     OR jsonb_typeof(v->'campo') IS DISTINCT FROM 'string' OR jsonb_typeof(v->'origen') IS DISTINCT FROM 'string'
     OR NOT ((v->>'campo' IN ('nombre','apellidos') AND v->>'origen'='certificado')
          OR (v->>'campo' IN ('telefono','movil','domicilio','codigo_postal') AND v->>'origen'='titular'))
     OR v->'version' IS DISTINCT FROM '1'::jsonb OR NOT vec_aspirantes.sobre_valido(v,1040)
  THEN RAISE EXCEPTION 'Aspirantes: valor inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 SELECT array_agg(value->>'campo' ORDER BY value->>'campo') INTO campos FROM jsonb_array_elements(p_ficha->'valores');
 IF (SELECT count(DISTINCT x) FROM unnest(campos) x)<>cardinality(campos)
    OR NOT campos @> ARRAY['nombre','apellidos']
 THEN RAISE EXCEPTION 'Aspirantes: valores incompletos o repetidos' USING ERRCODE='22023'; END IF;
 asp:=p_ficha->>'aspirante_ref';
 UPDATE vec_aspirantes.contexto SET aspirante_ref=asp WHERE xid=pg_current_xact_id_if_assigned()
  AND backend_pid=pg_backend_pid() AND sesion=session_user AND modo='alta' AND aspirante_ref IS NULL;
 GET DIAGNOSTICS n=ROW_COUNT;
 IF n<>1 THEN RAISE EXCEPTION 'Aspirantes: contexto incompleto' USING ERRCODE='42501'; END IF;
 c:=vec_aspirantes.contexto_vigente('alta');
 ahora:=date_trunc('microseconds',clock_timestamp());
 BEGIN
  INSERT INTO vec_aspirantes.ficha(aspirante_ref,version,estado,creada_en,actualizada_en) VALUES(asp,1,'activa',ahora,ahora);
  INSERT INTO vec_aspirantes.documento(aspirante_ref,documento_ref,tipo,pais,clave_ref,nonce,cifrado,clave_indice_ref,indice,origen,version_alta,registrado_en)
  VALUES(asp,doc->>'documento_ref',doc->>'tipo',doc->>'pais',doc->>'clave_ref',decode(doc->>'nonce_hex','hex'),decode(doc->>'cifrado_hex','hex'),
   c.indice_clave_ref,c.indice,'certificado',1,ahora);
 EXCEPTION WHEN unique_violation THEN
  RAISE EXCEPTION 'Aspirantes: referencia repetida' USING ERRCODE='40001';
 END;
 INSERT INTO vec_aspirantes.indice_documento(clave_indice_ref,indice,aspirante_ref,documento_ref)
 VALUES(c.indice_clave_ref,c.indice,asp,doc->>'documento_ref');
 INSERT INTO vec_aspirantes.valor(aspirante_ref,campo,version,estado,origen,clave_ref,nonce,cifrado,registrado_en)
 SELECT asp,value->>'campo',1,'presente',value->>'origen',value->>'clave_ref',decode(value->>'nonce_hex','hex'),decode(value->>'cifrado_hex','hex'),ahora
 FROM jsonb_array_elements(p_ficha->'valores');
 r:=vec_aspirantes.registrar_recibo(c,asp,1,'alta_titular',ARRAY['documento']||campos,p_ficha->>'catalogo_ref','ficha.alta',ahora);
 PERFORM vec_aspirantes.retirar_contexto('alta');
 RETURN r;
END $f$;
REVOKE ALL ON FUNCTION vec_aspirantes.alta_ficha_propia_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_aspirantes.alta_ficha_propia_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_aspirantes_ejecutor_externo;

-- Rectificación en dos llamadas de la MISMA transacción: preparar consume
-- V3, bloquea la ficha y dice qué campos tienen valor; Go cifra con la
-- ficha y la versión nuevas; aplicar escribe valores, historia y recibo.
CREATE FUNCTION vec_aspirantes.preparar_rectificacion_ficha_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; asp text; r jsonb; f record; presentes jsonb;
BEGIN
 m:=vec_aspirantes.consumir_contexto(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'rectificar');
 asp:=vec_aspirantes.fijar_ficha_por_indice();
 IF asp IS NULL THEN RAISE EXCEPTION 'Aspirantes: sin ficha' USING ERRCODE='P1404'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_aspirantes:ficha:'||asp,0));
 r:=vec_aspirantes.replay(m,asp);
 IF r IS NOT NULL THEN
  PERFORM vec_aspirantes.retirar_contexto('rectificar');
  RETURN jsonb_build_object('replay',r);
 END IF;
 SELECT * INTO STRICT f FROM vec_aspirantes.ficha WHERE aspirante_ref=asp FOR UPDATE;
 IF f.version::text IS DISTINCT FROM m->>'version_esperada'
 THEN RAISE EXCEPTION 'Aspirantes: versión en conflicto' USING ERRCODE='P1409'; END IF;
 SELECT coalesce(jsonb_agg(v.campo ORDER BY v.campo) FILTER (WHERE v.estado='presente' AND v.campo IN ('telefono','movil','domicilio','codigo_postal')),'[]'::jsonb)
 INTO presentes FROM (SELECT DISTINCT ON (x.campo) x.* FROM vec_aspirantes.valor x WHERE x.aspirante_ref=asp ORDER BY x.campo,x.version DESC) v;
 RETURN jsonb_build_object('aspirante_ref',asp,'version',f.version,'presentes',presentes);
END $f$;
REVOKE ALL ON FUNCTION vec_aspirantes.preparar_rectificacion_ficha_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_aspirantes.preparar_rectificacion_ficha_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_aspirantes_ejecutor_externo;

-- p_cambios: {catalogo_ref, motivo, valores:[{campo,version,origen,estado,
-- clave_ref,nonce_hex,cifrado_hex}]}; un valor retirado lleva los tres
-- últimos a null. El motivo se coteja con lo que había: «dato_nuevo» si
-- ninguno de los campos tenía valor; si alguno lo tenía, cambio o corrección.
CREATE FUNCTION vec_aspirantes.aplicar_rectificacion_ficha_v1(p_cambios jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE c vec_aspirantes.contexto; f record; v jsonb; nueva bigint; ahora timestamptz(6); presentes text[]; campos text[];
 motivo text; alguno boolean; n integer; r jsonb;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'Aspirantes: rectificación denegada' USING ERRCODE='42501'; END IF;
 c:=vec_aspirantes.contexto_vigente('rectificar');
 IF c.aspirante_ref IS NULL THEN RAISE EXCEPTION 'Aspirantes: rectificación sin preparar' USING ERRCODE='42501'; END IF;
 nueva:=c.version_esperada+1;
 motivo:=p_cambios->>'motivo';
 IF jsonb_typeof(p_cambios) IS DISTINCT FROM 'object'
    OR (SELECT array_agg(x ORDER BY x) FROM jsonb_object_keys(p_cambios) x) IS DISTINCT FROM ARRAY['catalogo_ref','motivo','valores']
    OR jsonb_typeof(p_cambios->'catalogo_ref') IS DISTINCT FROM 'string' OR p_cambios->>'catalogo_ref' !~ '^[A-Za-z0-9:._-]{1,128}$'
    OR jsonb_typeof(p_cambios->'motivo') IS DISTINCT FROM 'string' OR motivo NOT IN ('dato_nuevo','cambio_de_dato','correccion_de_error')
    OR jsonb_typeof(p_cambios->'valores') IS DISTINCT FROM 'array' OR jsonb_array_length(p_cambios->'valores') NOT BETWEEN 1 AND 4
 THEN RAISE EXCEPTION 'Aspirantes: cambios inválidos' USING ERRCODE='22023'; END IF;
 FOR v IN SELECT value FROM jsonb_array_elements(p_cambios->'valores') LOOP
  IF jsonb_typeof(v) IS DISTINCT FROM 'object'
     OR (SELECT array_agg(x ORDER BY x) FROM jsonb_object_keys(v) x) IS DISTINCT FROM ARRAY['campo','cifrado_hex','clave_ref','estado','nonce_hex','origen','version']
     OR jsonb_typeof(v->'campo') IS DISTINCT FROM 'string' OR v->>'campo' NOT IN ('telefono','movil','domicilio','codigo_postal')
     OR v->'origen' IS DISTINCT FROM '"titular"'::jsonb OR v->'version' IS DISTINCT FROM to_jsonb(nueva)
     OR NOT ((v->'estado'='"presente"'::jsonb AND vec_aspirantes.sobre_valido(v,1040))
          OR (v->'estado'='"retirado"'::jsonb AND v->'clave_ref'='null'::jsonb AND v->'nonce_hex'='null'::jsonb AND v->'cifrado_hex'='null'::jsonb))
  THEN RAISE EXCEPTION 'Aspirantes: valor inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 SELECT array_agg(value->>'campo') INTO campos FROM jsonb_array_elements(p_cambios->'valores');
 IF (SELECT count(DISTINCT x) FROM unnest(campos) x)<>cardinality(campos)
 THEN RAISE EXCEPTION 'Aspirantes: campo repetido' USING ERRCODE='22023'; END IF;
 SELECT * INTO STRICT f FROM vec_aspirantes.ficha WHERE aspirante_ref=c.aspirante_ref FOR UPDATE;
 IF f.version<>c.version_esperada THEN RAISE EXCEPTION 'Aspirantes: versión en conflicto' USING ERRCODE='P1409'; END IF;
 SELECT coalesce(array_agg(v2.campo) FILTER (WHERE v2.estado='presente'),ARRAY[]::text[]) INTO presentes
 FROM (SELECT DISTINCT ON (x.campo) x.* FROM vec_aspirantes.valor x WHERE x.aspirante_ref=c.aspirante_ref ORDER BY x.campo,x.version DESC) v2;
 alguno:=campos && presentes;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(p_cambios->'valores') e WHERE e.value->>'estado'='retirado' AND NOT (e.value->>'campo')=ANY(presentes))
    OR alguno=(motivo='dato_nuevo')
 THEN RAISE EXCEPTION 'Aspirantes: motivo incoherente' USING ERRCODE='22023'; END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 INSERT INTO vec_aspirantes.valor(aspirante_ref,campo,version,estado,origen,clave_ref,nonce,cifrado,registrado_en)
 SELECT c.aspirante_ref,e.value->>'campo',nueva,e.value->>'estado','titular',e.value->>'clave_ref',
  decode(e.value->>'nonce_hex','hex'),decode(e.value->>'cifrado_hex','hex'),ahora
 FROM jsonb_array_elements(p_cambios->'valores') e;
 UPDATE vec_aspirantes.ficha SET version=nueva,actualizada_en=ahora WHERE aspirante_ref=c.aspirante_ref AND version=nueva-1;
 GET DIAGNOSTICS n=ROW_COUNT;
 IF n<>1 THEN RAISE EXCEPTION 'Aspirantes: versión concurrente' USING ERRCODE='40001'; END IF;
 r:=vec_aspirantes.registrar_recibo(c,c.aspirante_ref,nueva,motivo,(SELECT array_agg(x ORDER BY x) FROM unnest(campos) x),
  p_cambios->>'catalogo_ref','ficha.contacto_rectificado',ahora);
 PERFORM vec_aspirantes.retirar_contexto('rectificar');
 RETURN r;
END $f$;
REVOKE ALL ON FUNCTION vec_aspirantes.aplicar_rectificacion_ficha_v1(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_aspirantes.aplicar_rectificacion_ficha_v1(jsonb) TO vec_aspirantes_ejecutor_externo;

-- El ejecutor solo tiene las cuatro fachadas: ninguna tabla ni auxiliar.
DO $acl$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p WHERE p.pronamespace='vec_aspirantes'::regnamespace
     AND p.proname NOT IN ('consultar_ficha_propia_v1','alta_ficha_propia_v1','preparar_rectificacion_ficha_v1','aplicar_rectificacion_ficha_v1')
     AND has_function_privilege('vec_aspirantes_ejecutor_externo',p.oid,'EXECUTE'))
    OR (SELECT count(*) FROM pg_proc p WHERE p.pronamespace='vec_aspirantes'::regnamespace
     AND has_function_privilege('vec_aspirantes_ejecutor_externo',p.oid,'EXECUTE'))<>4
    OR EXISTS(SELECT 1 FROM pg_class c WHERE c.relnamespace='vec_aspirantes'::regnamespace AND c.relkind='r'
     AND has_table_privilege('vec_aspirantes_ejecutor_externo',c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'))
    OR EXISTS(SELECT 1 FROM pg_policies WHERE schemaname='vec_aspirantes' AND roles<>ARRAY['vec_aspirantes_propietario']::name[])
    OR EXISTS(SELECT 1 FROM pg_class c WHERE c.relnamespace='vec_aspirantes'::regnamespace AND c.relkind='r' AND NOT (c.relrowsecurity AND c.relforcerowsecurity))
 THEN RAISE EXCEPTION 'Aspirantes 000001: ACL incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
