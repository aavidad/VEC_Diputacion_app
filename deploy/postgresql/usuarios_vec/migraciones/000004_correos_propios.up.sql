\set ON_ERROR_STOP on
-- Usuarios 000004 (5.08b): correos propios verificados. Aplicar tras AD3-107
-- y Usuarios 000001/000002/000003. Cada llamada usa una transacción
-- SERIALIZABLE READ WRITE con el LOGIN ejecutor de su superficie.
-- La dirección se guarda cifrada (AEAD) y con una huella HMAC de igualdad por
-- persona; historia, recibos y envíos sólo guardan referencias opacas.
BEGIN;
SET LOCAL ROLE vec_usuarios_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_usuarios:migracion:000004',0));
DO $pre$ BEGIN
 IF current_user<>'vec_usuarios_propietario'
    OR to_regclass('vec_usuarios.preferencias_recibo') IS NULL
    OR to_regclass('vec_usuarios.denegacion_frontera_preferencias') IS NULL
    OR to_regprocedure('vec_usuarios.rechazar_cambio_inmutable()') IS NULL
    OR to_regclass('vec_usuarios.correos_conjunto') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_correos_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_ejecutor_interno' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_ejecutor_externo' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'Usuarios 000004: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_usuarios.correos_conjunto (
 persona_ref text PRIMARY KEY CHECK(persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 version bigint NOT NULL CHECK(version>0),
 clave_igualdad_ref text NOT NULL CHECK(clave_igualdad_ref ~ '^[A-Za-z0-9:._-]{1,128}$'),
 actualizado_en timestamptz(6) NOT NULL,
 UNIQUE(persona_ref,clave_igualdad_ref)
);
CREATE TABLE vec_usuarios.correos_direccion (
 persona_ref text NOT NULL REFERENCES vec_usuarios.correos_conjunto(persona_ref),
 correo_ref text NOT NULL CHECK(correo_ref ~ '^correo:[0-9a-f]{32}$'),
 version_sobre bigint NOT NULL CHECK(version_sobre>0),
 clave_sobre_ref text NOT NULL CHECK(clave_sobre_ref ~ '^[A-Za-z0-9:._-]{1,128}$'),
 clave_igualdad_ref text NOT NULL CHECK(clave_igualdad_ref ~ '^[A-Za-z0-9:._-]{1,128}$'),
 CHECK(clave_sobre_ref<>clave_igualdad_ref),
 nonce bytea NOT NULL CHECK(octet_length(nonce)=12),
 cifrado bytea NOT NULL CHECK(octet_length(cifrado) BETWEEN 19 AND 270),
 huella_igualdad bytea NOT NULL CHECK(octet_length(huella_igualdad)=32),
 estado text NOT NULL CHECK(estado IN ('pendiente','verificado','retirado')),
 activo boolean NOT NULL DEFAULT false,
 creado_en timestamptz(6) NOT NULL,
 verificado_en timestamptz(6),
 retirado_en timestamptz(6),
 PRIMARY KEY(persona_ref,correo_ref),
 FOREIGN KEY(persona_ref,clave_igualdad_ref)
  REFERENCES vec_usuarios.correos_conjunto(persona_ref,clave_igualdad_ref)
  DEFERRABLE INITIALLY DEFERRED,
 CHECK(NOT activo OR estado='verificado'),
 CHECK((estado='pendiente' AND verificado_en IS NULL AND retirado_en IS NULL)
    OR (estado='verificado' AND verificado_en IS NOT NULL AND retirado_en IS NULL)
    OR (estado='retirado' AND retirado_en IS NOT NULL))
);
CREATE UNIQUE INDEX correos_un_activo ON vec_usuarios.correos_direccion(persona_ref) WHERE activo;
CREATE UNIQUE INDEX correos_un_no_retirado ON vec_usuarios.correos_direccion(persona_ref,huella_igualdad) WHERE estado<>'retirado';
-- El código nunca se guarda: sólo su HMAC con clave propia, ligado a persona,
-- correo, desafío y vencimiento. Un desafío se usa una vez y admite 5 intentos.
CREATE TABLE vec_usuarios.correos_desafio (
 persona_ref text NOT NULL,
 correo_ref text NOT NULL,
 desafio_ref text NOT NULL CHECK(desafio_ref ~ '^desafio:[0-9a-f]{32}$'),
 huella_codigo bytea NOT NULL CHECK(octet_length(huella_codigo)=32),
 clave_ref text NOT NULL CHECK(clave_ref ~ '^[A-Za-z0-9:._-]{1,128}$'),
 vence_en timestamptz(6) NOT NULL,
 estado text NOT NULL CHECK(estado IN ('pendiente','usado','sustituido','agotado')),
 intentos smallint NOT NULL DEFAULT 0 CHECK(intentos BETWEEN 0 AND 5),
 creado_en timestamptz(6) NOT NULL,
 PRIMARY KEY(persona_ref,correo_ref,desafio_ref),
 UNIQUE(desafio_ref),
 FOREIGN KEY(persona_ref,correo_ref) REFERENCES vec_usuarios.correos_direccion(persona_ref,correo_ref),
 CHECK(vence_en>creado_en AND vence_en<=creado_en+interval '72 hours')
);
CREATE UNIQUE INDEX correos_un_desafio_pendiente ON vec_usuarios.correos_desafio(persona_ref,correo_ref) WHERE estado='pendiente';
CREATE INDEX correos_desafio_ventana ON vec_usuarios.correos_desafio(persona_ref,creado_en);
CREATE TABLE vec_usuarios.correos_intento_fallido (
 persona_ref text NOT NULL,
 correo_ref text NOT NULL,
 desafio_ref text NOT NULL,
 numero smallint NOT NULL CHECK(numero BETWEEN 1 AND 5),
 clave_operacion text NOT NULL CHECK(clave_operacion ~ '^[A-Za-z0-9:._-]{16,128}$'),
 huella_clave_ref text NOT NULL CHECK(huella_clave_ref ~ '^[A-Za-z0-9:._-]{1,128}$'),
 huella_valor text NOT NULL CHECK(huella_valor ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 ocurrido_en timestamptz(6) NOT NULL,
 PRIMARY KEY(persona_ref,correo_ref,desafio_ref,numero),
 UNIQUE(persona_ref,clave_operacion),
 FOREIGN KEY(persona_ref,correo_ref,desafio_ref)
  REFERENCES vec_usuarios.correos_desafio(persona_ref,correo_ref,desafio_ref)
);
CREATE TABLE vec_usuarios.correos_historia (
 persona_ref text NOT NULL REFERENCES vec_usuarios.correos_conjunto(persona_ref),
 version bigint NOT NULL CHECK(version>0),
 accion text NOT NULL CHECK(accion IN ('vec.correos.anadir','vec.correos.reenviar','vec.correos.verificar','vec.correos.activar','vec.correos.retirar')),
 correo_ref text NOT NULL,
 estado_resultante text NOT NULL CHECK(estado_resultante IN ('pendiente','verificado','retirado')),
 activo_resultante boolean NOT NULL,
 anterior_activo_ref text,
 recibo_ref text NOT NULL UNIQUE,
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(persona_ref,version),
 UNIQUE(persona_ref,version,accion,correo_ref,recibo_ref),
 CHECK(anterior_activo_ref IS NULL OR anterior_activo_ref<>correo_ref),
 FOREIGN KEY(persona_ref,correo_ref) REFERENCES vec_usuarios.correos_direccion(persona_ref,correo_ref)
);
CREATE TABLE vec_usuarios.correos_recibo (
 persona_ref text NOT NULL,
 clave_operacion text NOT NULL CHECK(clave_operacion ~ '^[A-Za-z0-9:._-]{16,128}$'),
 huella_clave_ref text NOT NULL,
 huella_valor text NOT NULL CHECK(huella_valor ~ '^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^correo_recibo:[0-9a-f]{32}$'),
 accion text NOT NULL,
 correo_ref text NOT NULL,
 version bigint NOT NULL,
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(persona_ref,clave_operacion),
 FOREIGN KEY(persona_ref,correo_ref) REFERENCES vec_usuarios.correos_direccion(persona_ref,correo_ref),
 FOREIGN KEY(persona_ref,version,accion,correo_ref,recibo_ref)
  REFERENCES vec_usuarios.correos_historia(persona_ref,version,accion,correo_ref,recibo_ref)
  DEFERRABLE INITIALLY DEFERRED
);
-- Salida de correo. Se crea ya reservada en la misma transacción que el
-- efecto; la reserva (128 bits aleatorios) sólo vuelve al proceso que la
-- creó y es la única llave para anotar una vez la respuesta del relay SMTP.
-- Ni la dirección ni el código figuran aquí.
CREATE TABLE vec_usuarios.correos_envio (
 envio_ref text PRIMARY KEY CHECK(envio_ref ~ '^correo_envio:[0-9a-f]{32}$'),
 persona_ref text NOT NULL,
 correo_ref text NOT NULL,
 superficie text NOT NULL CHECK(superficie IN ('interna_corporativa','externa_personal')),
 tipo text NOT NULL CHECK(tipo IN ('verificacion','aviso_cambio')),
 desafio_ref text,
 recibo_ref text NOT NULL,
 reserva_ref text NOT NULL UNIQUE CHECK(reserva_ref ~ '^reserva:[0-9a-f]{32}$'),
 estado text NOT NULL CHECK(estado IN ('reservado','aceptado','no_aceptado')),
 creado_en timestamptz(6) NOT NULL,
 resuelto_en timestamptz(6),
 FOREIGN KEY(persona_ref,correo_ref) REFERENCES vec_usuarios.correos_direccion(persona_ref,correo_ref),
 FOREIGN KEY(desafio_ref) REFERENCES vec_usuarios.correos_desafio(desafio_ref),
 CHECK((tipo='verificacion' AND desafio_ref IS NOT NULL) OR (tipo='aviso_cambio' AND desafio_ref IS NULL)),
 CHECK((estado='reservado' AND resuelto_en IS NULL) OR (estado<>'reservado' AND resuelto_en IS NOT NULL))
);
CREATE TABLE vec_usuarios.correos_contexto (
 xid xid8 NOT NULL,
 backend_pid integer NOT NULL,
 sesion text NOT NULL,
 superficie text NOT NULL CHECK(superficie IN ('interna_corporativa','externa_personal')),
 persona_ref text NOT NULL,
 modo text NOT NULL CHECK(modo IN ('consultar','recuperar','actualizar','verificar','envio')),
 accion text NOT NULL,
 clave_operacion text,
 correo_ref text,
 version_esperada bigint,
 huella_clave_ref text,
 huella_valor text,
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL,
 PRIMARY KEY(xid,backend_pid,sesion)
);

DO $rls$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['correos_conjunto','correos_direccion','correos_desafio','correos_intento_fallido','correos_historia','correos_recibo','correos_envio','correos_contexto'] LOOP
  EXECUTE format('ALTER TABLE vec_usuarios.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_usuarios.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_usuarios.%I FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo,vec_usuarios_registrador_frontera_interno,vec_usuarios_registrador_frontera_externo',t);
 END LOOP;
END $rls$;
GRANT USAGE ON SCHEMA vec_usuarios TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

-- Superficie del LOGIN técnico: una sola membresía INHERIT en el ejecutor.
CREATE FUNCTION vec_usuarios.superficie_sesion_correos()
RETURNS text LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE rol text; superficie text;
BEGIN
 IF current_user<>'vec_usuarios_propietario' OR session_user=current_user
    OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user
      AND (NOT rolcanlogin OR rolsuper OR rolbypassrls OR rolcreatedb OR rolcreaterole OR rolreplication))
    OR (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1
 THEN RETURN NULL; END IF;
 SELECT r.rolname INTO rol FROM pg_auth_members m JOIN pg_roles r ON r.oid=m.roleid
 WHERE m.member=session_user::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option;
 superficie:=CASE rol
  WHEN 'vec_usuarios_ejecutor_interno' THEN 'interna_corporativa'
  WHEN 'vec_usuarios_ejecutor_externo' THEN 'externa_personal'
  ELSE NULL END;
 IF superficie IS NULL OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=rol::regrole)
 THEN RETURN NULL; END IF;
 RETURN superficie;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.superficie_sesion_correos() FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

CREATE FUNCTION vec_usuarios.contexto_autorizado_correos(p_persona text,p_modos text[])
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
 SELECT EXISTS(SELECT 1 FROM vec_usuarios.correos_contexto c
 WHERE c.xid=pg_current_xact_id_if_assigned() AND c.backend_pid=pg_backend_pid()
 AND c.sesion=session_user AND c.superficie=vec_usuarios.superficie_sesion_correos()
 AND c.persona_ref=p_persona AND c.modo=ANY(p_modos))
$f$;
REVOKE ALL ON FUNCTION vec_usuarios.contexto_autorizado_correos(text,text[]) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;
CREATE POLICY correos_contexto_lectura ON vec_usuarios.correos_contexto FOR SELECT TO vec_usuarios_propietario
 USING (backend_pid=pg_backend_pid() AND sesion=session_user AND superficie=vec_usuarios.superficie_sesion_correos());
CREATE POLICY correos_contexto_alta ON vec_usuarios.correos_contexto FOR INSERT TO vec_usuarios_propietario
 WITH CHECK (xid=pg_current_xact_id_if_assigned() AND backend_pid=pg_backend_pid() AND sesion=session_user
  AND superficie=vec_usuarios.superficie_sesion_correos());
CREATE POLICY correos_contexto_baja ON vec_usuarios.correos_contexto FOR DELETE TO vec_usuarios_propietario
 USING (backend_pid=pg_backend_pid() AND sesion=session_user AND superficie=vec_usuarios.superficie_sesion_correos());
CREATE POLICY correos_conjunto_lectura ON vec_usuarios.correos_conjunto FOR SELECT TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['consultar','actualizar','verificar']));
CREATE POLICY correos_conjunto_alta ON vec_usuarios.correos_conjunto FOR INSERT TO vec_usuarios_propietario WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar']));
CREATE POLICY correos_conjunto_cambio ON vec_usuarios.correos_conjunto FOR UPDATE TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar','verificar'])) WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar','verificar']));
CREATE POLICY correos_direccion_lectura ON vec_usuarios.correos_direccion FOR SELECT TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['consultar','actualizar','verificar']));
CREATE POLICY correos_direccion_alta ON vec_usuarios.correos_direccion FOR INSERT TO vec_usuarios_propietario WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar']));
CREATE POLICY correos_direccion_cambio ON vec_usuarios.correos_direccion FOR UPDATE TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar','verificar'])) WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar','verificar']));
CREATE POLICY correos_desafio_lectura ON vec_usuarios.correos_desafio FOR SELECT TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['consultar','actualizar','verificar']));
CREATE POLICY correos_desafio_alta ON vec_usuarios.correos_desafio FOR INSERT TO vec_usuarios_propietario WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar']));
CREATE POLICY correos_desafio_cambio ON vec_usuarios.correos_desafio FOR UPDATE TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar','verificar'])) WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar','verificar']));
CREATE POLICY correos_intento_fallido_lectura ON vec_usuarios.correos_intento_fallido FOR SELECT TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['recuperar','actualizar','verificar']));
CREATE POLICY correos_intento_fallido_alta ON vec_usuarios.correos_intento_fallido FOR INSERT TO vec_usuarios_propietario WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['verificar']));
CREATE POLICY correos_historia_alta ON vec_usuarios.correos_historia FOR INSERT TO vec_usuarios_propietario WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar','verificar']));
CREATE POLICY correos_recibo_lectura ON vec_usuarios.correos_recibo FOR SELECT TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['recuperar','actualizar','verificar']));
CREATE POLICY correos_recibo_alta ON vec_usuarios.correos_recibo FOR INSERT TO vec_usuarios_propietario WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar','verificar']));
CREATE POLICY correos_envio_lectura ON vec_usuarios.correos_envio FOR SELECT TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar','envio']));
CREATE POLICY correos_envio_alta ON vec_usuarios.correos_envio FOR INSERT TO vec_usuarios_propietario WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar']));
CREATE POLICY correos_envio_cambio ON vec_usuarios.correos_envio FOR UPDATE TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['envio'])) WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['envio']));
CREATE TRIGGER correos_historia_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios.correos_historia FOR EACH ROW EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER correos_recibo_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios.correos_recibo FOR EACH ROW EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER correos_intento_fallido_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios.correos_intento_fallido FOR EACH ROW EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER correos_historia_no_truncar BEFORE TRUNCATE ON vec_usuarios.correos_historia FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER correos_recibo_no_truncar BEFORE TRUNCATE ON vec_usuarios.correos_recibo FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER correos_intento_fallido_no_truncar BEFORE TRUNCATE ON vec_usuarios.correos_intento_fallido FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER correos_envio_no_truncar BEFORE TRUNCATE ON vec_usuarios.correos_envio FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER correos_desafio_no_truncar BEFORE TRUNCATE ON vec_usuarios.correos_desafio FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER correos_direccion_no_truncar BEFORE TRUNCATE ON vec_usuarios.correos_direccion FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();

-- Filas vivas: dirección, desafío y envío sólo admiten las transiciones
-- nominales y nunca se borran. El resto de columnas queda congelado.
CREATE FUNCTION vec_usuarios.correos_transicion_valida()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'historia de Usuarios inmutable' USING ERRCODE='55000'; END IF;
 IF TG_TABLE_NAME='correos_direccion' THEN
  IF (to_jsonb(NEW)-'estado'-'activo'-'verificado_en'-'retirado_en') IS DISTINCT FROM (to_jsonb(OLD)-'estado'-'activo'-'verificado_en'-'retirado_en')
     OR OLD.estado='retirado'
     OR (OLD.estado='verificado' AND NEW.estado='pendiente')
     OR (OLD.verificado_en IS NOT NULL AND NEW.verificado_en IS DISTINCT FROM OLD.verificado_en)
  THEN RAISE EXCEPTION 'transición de correo inválida' USING ERRCODE='55000'; END IF;
 ELSIF TG_TABLE_NAME='correos_desafio' THEN
  IF (to_jsonb(NEW)-'estado'-'intentos') IS DISTINCT FROM (to_jsonb(OLD)-'estado'-'intentos')
     OR OLD.estado<>'pendiente' OR NEW.intentos<OLD.intentos
  THEN RAISE EXCEPTION 'transición de desafío inválida' USING ERRCODE='55000'; END IF;
 ELSIF TG_TABLE_NAME='correos_envio' THEN
  IF (to_jsonb(NEW)-'estado'-'resuelto_en') IS DISTINCT FROM (to_jsonb(OLD)-'estado'-'resuelto_en')
     OR OLD.estado<>'reservado' OR NEW.estado='reservado'
  THEN RAISE EXCEPTION 'transición de envío inválida' USING ERRCODE='55000'; END IF;
 END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.correos_transicion_valida() FROM PUBLIC;
CREATE TRIGGER correos_direccion_transicion BEFORE UPDATE OR DELETE ON vec_usuarios.correos_direccion FOR EACH ROW EXECUTE FUNCTION vec_usuarios.correos_transicion_valida();
CREATE TRIGGER correos_desafio_transicion BEFORE UPDATE OR DELETE ON vec_usuarios.correos_desafio FOR EACH ROW EXECUTE FUNCTION vec_usuarios.correos_transicion_valida();
CREATE TRIGGER correos_envio_transicion BEFORE UPDATE OR DELETE ON vec_usuarios.correos_envio FOR EACH ROW EXECUTE FUNCTION vec_usuarios.correos_transicion_valida();

CREATE FUNCTION vec_usuarios.huella_contexto_correos(p_material text)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE persona text; canon text;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 8192
 THEN RAISE EXCEPTION 'Usuarios: material correo inválido' USING ERRCODE='22023'; END IF;
 BEGIN persona:=p_material::jsonb->>'persona_ref';
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Usuarios: material correo inválido' USING ERRCODE='22023'; END;
 IF persona IS NULL OR persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
 THEN RAISE EXCEPTION 'Usuarios: persona inválida' USING ERRCODE='22023'; END IF;
 canon:='{"ambitos":{"persona_ref":'||to_jsonb(persona)::text||'},"atributos":{"material_sha256":"'||
 encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}';
 RETURN encode(sha256(convert_to(canon,'UTF8')),'hex');
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.huella_contexto_correos(text) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

-- El material es el JSON literal que Go serializa (canonico.SerializarMaterialCorreos).
CREATE FUNCTION vec_usuarios.validar_material_correos(
 p_material text,p_capacidad bytea,p_decision bytea,p_persona_version numeric,p_perfil_version numeric)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE m jsonb; c jsonb; d jsonb; k text[]; h text; accion text; campos jsonb; sello jsonb; claves text[];
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 8192
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_persona_version IS NULL OR p_perfil_version IS NULL
 THEN RAISE EXCEPTION 'Usuarios: material correo denegado' USING ERRCODE='42501'; END IF;
 BEGIN m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Usuarios: material correo inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'Usuarios: material correo inválido' USING ERRCODE='22023'; END IF;
 SELECT array_agg(x ORDER BY x) INTO k FROM jsonb_object_keys(m) x;
 accion:=m->>'accion';
 SELECT x.campos INTO campos FROM (VALUES
 ('vec.correos.consultar','["activo","correo_ref","direccion","estado","version"]'::jsonb),
 ('vec.correos.anadir','["correo_ref","direccion","estado","version"]'::jsonb),
 ('vec.correos.reenviar','["correo_ref","estado","version"]'::jsonb),
 ('vec.correos.verificar','["correo_ref","estado","version"]'::jsonb),
 ('vec.correos.activar','["activo","correo_ref","version"]'::jsonb),
 ('vec.correos.retirar','["activo","correo_ref","estado","version"]'::jsonb)
 ) x(nombre,campos) WHERE x.nombre=accion;
 h:=vec_usuarios.huella_contexto_correos(p_material);
 IF k IS DISTINCT FROM ARRAY['accion','clave_operacion','correo_ref','finalidad_ref','huellas_peticion','perfil_ref','persona_ref','superficie','version_esperada']
    OR EXISTS(SELECT 1 FROM jsonb_each(m) z WHERE z.key IN
      ('accion','clave_operacion','correo_ref','finalidad_ref','perfil_ref','persona_ref','superficie')
      AND jsonb_typeof(z.value)<>'string')
    OR jsonb_typeof(m->'version_esperada') IS DISTINCT FROM 'number'
    OR campos IS NULL
    OR m->>'superficie' IS DISTINCT FROM vec_usuarios.superficie_sesion_correos()
    OR m->>'superficie' IS DISTINCT FROM d #>> '{vinculo_autenticacion_actor,superficie}'
    OR c->>'audiencia_consumo' IS DISTINCT FROM
       'vec_usuarios.correos.'||split_part(accion,'.',3)||'.'||(m->>'superficie')||'.v1'
    OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR m->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
    OR m->>'finalidad_ref' IS DISTINCT FROM 'finalidad:usuarios:correos-propios:v1'
    OR m->>'persona_ref' IS DISTINCT FROM d->>'principal_id'
    OR m->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
    OR m->>'persona_ref' IS DISTINCT FROM d->>'recurso_ref'
    OR m->>'persona_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'accion' IS DISTINCT FROM accion OR c->>'operacion' IS DISTINCT FROM accion
    OR d->>'modulo_id' IS DISTINCT FROM 'usuarios' OR d->>'tipo_recurso' IS DISTINCT FROM 'correos_persona'
    OR d->>'finalidad' IS DISTINCT FROM 'finalidad:usuarios:correos-propios:v1' OR d->'concedida' IS DISTINCT FROM 'true'::jsonb
    OR d->'campos_permitidos' IS DISTINCT FROM campos OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'decision_ref' IS NULL OR length(d->>'decision_ref') NOT BETWEEN 1 AND 256
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h OR c->>'huella_efecto_sha256' IS DISTINCT FROM h
    OR m->>'version_esperada' !~ '^(0|[1-9][0-9]{0,17})$'
 THEN RAISE EXCEPTION 'Usuarios: correo no autorizado' USING ERRCODE='42501'; END IF;
 IF accion='vec.correos.consultar' THEN
  IF m->>'clave_operacion'<>'' OR m->>'correo_ref'<>'' OR m->'huellas_peticion' IS DISTINCT FROM '{}'::jsonb
     OR (m->>'version_esperada')::bigint<>0
  THEN RAISE EXCEPTION 'Usuarios: consulta correo inválida' USING ERRCODE='22023'; END IF;
 ELSE
  IF m->>'clave_operacion' !~ '^[A-Za-z0-9:._-]{16,128}$'
     OR jsonb_typeof(m->'huellas_peticion') IS DISTINCT FROM 'object'
     OR (SELECT array_agg(x ORDER BY x) FROM jsonb_object_keys(m->'huellas_peticion') x) IS DISTINCT FROM ARRAY['activa','retenidas']
     OR jsonb_typeof(m#>'{huellas_peticion,retenidas}') IS DISTINCT FROM 'array'
     OR jsonb_array_length(m#>'{huellas_peticion,retenidas}')>8
     OR (accion='vec.correos.anadir' AND m->>'correo_ref'<>'')
     OR (accion<>'vec.correos.anadir' AND m->>'correo_ref' !~ '^correo:[0-9a-f]{32}$')
  THEN RAISE EXCEPTION 'Usuarios: mutación correo inválida' USING ERRCODE='22023'; END IF;
  claves:=ARRAY[]::text[];
  FOR sello IN SELECT value FROM jsonb_array_elements(jsonb_build_array(m#>'{huellas_peticion,activa}') || (m#>'{huellas_peticion,retenidas}')) LOOP
   IF jsonb_typeof(sello) IS DISTINCT FROM 'object'
      OR (SELECT array_agg(x ORDER BY x) FROM jsonb_object_keys(sello) x) IS DISTINCT FROM ARRAY['clave_ref','valor']
      OR jsonb_typeof(sello->'clave_ref') IS DISTINCT FROM 'string'
      OR jsonb_typeof(sello->'valor') IS DISTINCT FROM 'string'
      OR sello->>'clave_ref' !~ '^[A-Za-z0-9:._-]{1,128}$'
      OR sello->>'valor' !~ '^[0-9a-f]{64}$' OR sello->>'clave_ref'=ANY(claves)
   THEN RAISE EXCEPTION 'Usuarios: huella correo inválida' USING ERRCODE='22023'; END IF;
   claves:=array_append(claves,sello->>'clave_ref');
  END LOOP;
 END IF;
 RETURN m;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.validar_material_correos(text,bytea,bytea,numeric,numeric) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

CREATE FUNCTION vec_usuarios.activar_contexto_correos(p_m jsonb,p_modo text,p_x jsonb)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE xid_actual xid8;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR p_modo NOT IN ('consultar','recuperar','actualizar','verificar')
    OR p_m->>'superficie' IS DISTINCT FROM vec_usuarios.superficie_sesion_correos()
    OR p_x->>'decision_ref' IS NULL OR p_x->>'auditoria_ref' IS NULL
    OR p_x->>'consumo_huella_sha256' IS NULL OR p_x->>'consumo_huella_sha256' !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'Usuarios: contexto correo denegado' USING ERRCODE='42501'; END IF;
 xid_actual:=pg_current_xact_id();
 DELETE FROM vec_usuarios.correos_contexto WHERE backend_pid=pg_backend_pid() AND xid<>xid_actual;
 IF EXISTS(SELECT 1 FROM vec_usuarios.correos_contexto WHERE xid=xid_actual AND backend_pid=pg_backend_pid() AND sesion=session_user)
 THEN RAISE EXCEPTION 'Usuarios: contexto correo ya activo' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_usuarios.correos_contexto(xid,backend_pid,sesion,superficie,persona_ref,modo,accion,clave_operacion,correo_ref,version_esperada,huella_clave_ref,huella_valor,decision_ref,auditoria_ref,consumo_huella_sha256)
 VALUES(xid_actual,pg_backend_pid(),session_user,p_m->>'superficie',p_m->>'persona_ref',p_modo,p_m->>'accion',
  nullif(p_m->>'clave_operacion',''),nullif(p_m->>'correo_ref',''),(p_m->>'version_esperada')::bigint,
  p_m#>>'{huellas_peticion,activa,clave_ref}',p_m#>>'{huellas_peticion,activa,valor}',p_x->>'decision_ref',p_x->>'auditoria_ref',p_x->>'consumo_huella_sha256');
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.activar_contexto_correos(jsonb,text,jsonb) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

CREATE FUNCTION vec_usuarios.retirar_contexto_correos(p_persona text,p_modo text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE n integer;
BEGIN
 DELETE FROM vec_usuarios.correos_contexto WHERE xid=pg_current_xact_id_if_assigned()
 AND backend_pid=pg_backend_pid() AND sesion=session_user AND persona_ref=p_persona AND modo=p_modo;
 GET DIAGNOSTICS n=ROW_COUNT;
 IF n<>1 THEN RAISE EXCEPTION 'Usuarios: contexto correo incompleto' USING ERRCODE='42501'; END IF;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.retirar_contexto_correos(text,text) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

-- Consume V3 fresco antes de mirar ninguna fila y abre el contexto RLS.
CREATE FUNCTION vec_usuarios.consumir_contexto_correos(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_modo text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE m jsonb; x record; resultado jsonb;
BEGIN
 m:=vec_usuarios.validar_material_correos(p_material,p_capacidad,p_decision,p_persona_version,p_perfil_version);
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_correos_v3_atestada(
  m->>'accion',m->>'superficie',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.efecto_ref IS DISTINCT FROM m->>'persona_ref'
    OR x.decision_ref IS DISTINCT FROM convert_from(p_decision,'UTF8')::jsonb->>'decision_ref'
    OR x.huella_efecto_sha256 IS DISTINCT FROM vec_usuarios.huella_contexto_correos(p_material)
 THEN RAISE EXCEPTION 'Usuarios: consumo correo divergente' USING ERRCODE='42501'; END IF;
 resultado:=jsonb_build_object('decision_ref',x.decision_ref,'auditoria_ref',x.auditoria_ref,'consumo_huella_sha256',x.consumo_huella_sha256);
 PERFORM vec_usuarios.activar_contexto_correos(m,p_modo,resultado);
 RETURN m;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.consumir_contexto_correos(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

-- Replay por persona+clave: devuelve el recibo original o el intento fallido
-- original. La misma clave con otra huella semántica es conflicto.
CREATE FUNCTION vec_usuarios.replay_correos_autorizado(p_m jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE r record; coincide boolean; fallido boolean:=false;
BEGIN
 SELECT * INTO r FROM vec_usuarios.correos_recibo
 WHERE persona_ref=p_m->>'persona_ref' AND clave_operacion=p_m->>'clave_operacion';
 IF NOT FOUND THEN
  SELECT * INTO r FROM vec_usuarios.correos_intento_fallido
  WHERE persona_ref=p_m->>'persona_ref' AND clave_operacion=p_m->>'clave_operacion';
  IF NOT FOUND THEN RETURN NULL; END IF;
  fallido:=true;
 END IF;
 SELECT EXISTS(SELECT 1 FROM jsonb_array_elements(jsonb_build_array(p_m#>'{huellas_peticion,activa}') || (p_m#>'{huellas_peticion,retenidas}')) h
  WHERE h->>'clave_ref'=r.huella_clave_ref AND h->>'valor'=r.huella_valor) INTO coincide;
 IF NOT coincide
 THEN RAISE EXCEPTION 'Usuarios: clave reutilizada con otra petición' USING ERRCODE='P1409'; END IF;
 IF fallido THEN
  IF p_m->>'accion' IS DISTINCT FROM 'vec.correos.verificar'
  THEN RAISE EXCEPTION 'Usuarios: clave reutilizada con otra petición' USING ERRCODE='P1409'; END IF;
  RETURN jsonb_build_object('resultado','codigo_invalido','persona_ref',r.persona_ref,
   'correo_ref',r.correo_ref,'replay',true);
 END IF;
 IF r.accion IS DISTINCT FROM p_m->>'accion'
 THEN RAISE EXCEPTION 'Usuarios: clave reutilizada con otra petición' USING ERRCODE='P1409'; END IF;
 RETURN jsonb_build_object('recibo_ref',r.recibo_ref,'persona_ref',r.persona_ref,'accion',r.accion,
  'correo_ref',r.correo_ref,'version',r.version,'fecha_utc',r.registrada_en,'replay',true,'envios','[]'::jsonb);
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.replay_correos_autorizado(jsonb) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

CREATE FUNCTION vec_usuarios.recuperar_correos_operacion_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; r jsonb;
BEGIN
 m:=vec_usuarios.consumir_contexto_correos(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'recuperar');
 IF m->>'accion' NOT IN ('vec.correos.anadir','vec.correos.reenviar','vec.correos.verificar','vec.correos.activar','vec.correos.retirar')
 THEN RAISE EXCEPTION 'Usuarios: acción de recuperación inválida' USING ERRCODE='42501'; END IF;
 r:=vec_usuarios.replay_correos_autorizado(m);
 PERFORM vec_usuarios.retirar_contexto_correos(m->>'persona_ref','recuperar');
 RETURN r;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.recuperar_correos_operacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios.recuperar_correos_operacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

CREATE FUNCTION vec_usuarios.sobre_correo_json(d vec_usuarios.correos_direccion)
RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT jsonb_build_object('version',d.version_sobre,'clave_ref',d.clave_sobre_ref,
  'nonce_hex',encode(d.nonce,'hex'),'cifrado_hex',encode(d.cifrado,'hex'))
$f$;
REVOKE ALL ON FUNCTION vec_usuarios.sobre_correo_json(vec_usuarios.correos_direccion) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

CREATE FUNCTION vec_usuarios.consultar_correos_propios_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; v bigint; correos jsonb; ahora timestamptz(6);
BEGIN
 m:=vec_usuarios.consumir_contexto_correos(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'consultar');
 IF m->>'accion' IS DISTINCT FROM 'vec.correos.consultar'
 THEN RAISE EXCEPTION 'Usuarios: consulta correo inválida' USING ERRCODE='42501'; END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 SELECT version INTO v FROM vec_usuarios.correos_conjunto WHERE persona_ref=m->>'persona_ref';
 SELECT coalesce(jsonb_agg(jsonb_build_object('correo_ref',d.correo_ref,'estado',d.estado,'activo',d.activo,
  'creado_utc',d.creado_en,'verificado_utc',d.verificado_en,'sobre',vec_usuarios.sobre_correo_json(d),
  'codigo',(SELECT jsonb_build_object('vence_utc',h.vence_en,'intentos_restantes',5-h.intentos)
            FROM vec_usuarios.correos_desafio h WHERE h.persona_ref=d.persona_ref AND h.correo_ref=d.correo_ref
            AND h.estado='pendiente' AND h.intentos<5 AND h.vence_en>ahora AND d.estado='pendiente'))
  ORDER BY d.creado_en,d.correo_ref),'[]'::jsonb)
 INTO correos FROM vec_usuarios.correos_direccion d WHERE d.persona_ref=m->>'persona_ref' AND d.estado<>'retirado';
 PERFORM vec_usuarios.retirar_contexto_correos(m->>'persona_ref','consultar');
 RETURN jsonb_build_object('persona_ref',m->>'persona_ref','version',coalesce(v,0),'correos',correos);
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.consultar_correos_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios.consultar_correos_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

CREATE FUNCTION vec_usuarios.registrar_recibo_correos(p_estado text,p_activo boolean,p_correo_ref text,p_anterior_activo_ref text,p_version bigint,p_fecha timestamptz)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE c record; ref text;
BEGIN
 SELECT * INTO STRICT c FROM vec_usuarios.correos_contexto WHERE xid=pg_current_xact_id_if_assigned()
 AND backend_pid=pg_backend_pid() AND sesion=session_user AND modo IN ('actualizar','verificar');
 IF c.huella_valor IS NULL OR c.huella_clave_ref IS NULL OR c.clave_operacion IS NULL
 THEN RAISE EXCEPTION 'Usuarios: recibo sin contexto' USING ERRCODE='42501'; END IF;
 ref:='correo_recibo:'||replace(gen_random_uuid()::text,'-','');
 INSERT INTO vec_usuarios.correos_historia(persona_ref,version,accion,correo_ref,estado_resultante,activo_resultante,anterior_activo_ref,recibo_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(c.persona_ref,p_version,c.accion,p_correo_ref,p_estado,p_activo,p_anterior_activo_ref,ref,c.decision_ref,c.auditoria_ref,p_fecha);
 INSERT INTO vec_usuarios.correos_recibo(persona_ref,clave_operacion,huella_clave_ref,huella_valor,recibo_ref,accion,correo_ref,version,decision_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
 VALUES(c.persona_ref,c.clave_operacion,c.huella_clave_ref,c.huella_valor,ref,c.accion,p_correo_ref,p_version,c.decision_ref,c.auditoria_ref,c.consumo_huella_sha256,p_fecha);
 RETURN jsonb_build_object('recibo_ref',ref,'persona_ref',c.persona_ref,'accion',c.accion,
  'correo_ref',p_correo_ref,'version',p_version,'fecha_utc',p_fecha,'replay',false);
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.registrar_recibo_correos(text,boolean,text,text,bigint,timestamptz) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

-- Reserva una salida de correo ligada al recibo recién creado.
CREATE FUNCTION vec_usuarios.reservar_envio_correos(p_persona text,p_superficie text,p_correo text,p_tipo text,p_desafio text,p_recibo text,p_fecha timestamptz)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE d vec_usuarios.correos_direccion; envio text; reserva text;
BEGIN
 SELECT * INTO STRICT d FROM vec_usuarios.correos_direccion WHERE persona_ref=p_persona AND correo_ref=p_correo;
 envio:='correo_envio:'||replace(gen_random_uuid()::text,'-','');
 reserva:='reserva:'||replace(gen_random_uuid()::text,'-','');
 INSERT INTO vec_usuarios.correos_envio(envio_ref,persona_ref,correo_ref,superficie,tipo,desafio_ref,recibo_ref,reserva_ref,estado,creado_en)
 VALUES(envio,p_persona,p_correo,p_superficie,p_tipo,p_desafio,p_recibo,reserva,'reservado',p_fecha);
 RETURN jsonb_build_object('envio_ref',envio,'reserva_ref',reserva,'tipo',p_tipo,'correo_ref',p_correo,
  'desafio_ref',p_desafio,'sobre',vec_usuarios.sobre_correo_json(d));
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.reservar_envio_correos(text,text,text,text,text,text,timestamptz) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

-- anadir/reenviar/activar/retirar. Límites: 5 direcciones vivas, 5 códigos
-- por persona y hora, 3 por dirección y hora. La activa no se retira: antes
-- se elige otra. Un cambio de activa avisa a la dirección anterior.
CREATE FUNCTION vec_usuarios.aplicar_correos_propios_v1(
 p_material text,p_sobre jsonb,p_reserva jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre_v3 bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; c record; d record; v bigint; ahora timestamptz(6); vence timestamptz(6);
 correo text; accion text; nuevo_estado text; nuevo_activo boolean; anterior text; n integer; r jsonb;
 envios jsonb:='[]'::jsonb; desafio text;
BEGIN
 m:=vec_usuarios.consumir_contexto_correos(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre_v3,p_evidencia,p_raiz,'actualizar');
 accion:=m->>'accion';
 IF accion NOT IN ('vec.correos.anadir','vec.correos.reenviar','vec.correos.activar','vec.correos.retirar')
 THEN RAISE EXCEPTION 'Usuarios: acción correo inválida' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_usuarios:correos:'||(m->>'persona_ref'),0));
 r:=vec_usuarios.replay_correos_autorizado(m);
 IF r IS NOT NULL THEN
  PERFORM vec_usuarios.retirar_contexto_correos(m->>'persona_ref','actualizar');
  IF r ? 'resultado' THEN RAISE EXCEPTION 'Usuarios: clave reutilizada con otra petición' USING ERRCODE='P1409'; END IF;
  RETURN r;
 END IF;
 SELECT * INTO c FROM vec_usuarios.correos_conjunto WHERE persona_ref=m->>'persona_ref' FOR UPDATE;
 IF coalesce(c.version,0)::text IS DISTINCT FROM m->>'version_esperada'
 THEN RAISE EXCEPTION 'Usuarios: versión correo en conflicto' USING ERRCODE='P1409'; END IF;
 v:=coalesce(c.version,0)+1; ahora:=date_trunc('microseconds',clock_timestamp());
 IF accion='vec.correos.anadir' THEN
  IF jsonb_typeof(p_sobre) IS DISTINCT FROM 'object'
     OR (SELECT array_agg(x ORDER BY x) FROM jsonb_object_keys(p_sobre) x) IS DISTINCT FROM ARRAY['cifrado_hex','clave_igualdad_ref','clave_ref','correo_ref','huella_igualdad_hex','nonce_hex','version']
     OR EXISTS(SELECT 1 FROM jsonb_each(p_sobre) z WHERE z.key<>'version' AND jsonb_typeof(z.value)<>'string')
     OR jsonb_typeof(p_sobre->'version') IS DISTINCT FROM 'number'
     OR p_sobre->>'correo_ref' !~ '^correo:[0-9a-f]{32}$'
     OR p_sobre->>'version' IS DISTINCT FROM v::text
     OR p_sobre->>'clave_ref' !~ '^[A-Za-z0-9:._-]{1,128}$'
     OR p_sobre->>'clave_igualdad_ref' !~ '^[A-Za-z0-9:._-]{1,128}$'
     OR p_sobre->>'clave_igualdad_ref' IS NOT DISTINCT FROM p_sobre->>'clave_ref'
     OR p_sobre->>'nonce_hex' !~ '^[0-9a-f]{24}$'
     OR p_sobre->>'cifrado_hex' !~ '^[0-9a-f]+$' OR length(p_sobre->>'cifrado_hex') NOT BETWEEN 38 AND 540 OR length(p_sobre->>'cifrado_hex')%2<>0
     OR p_sobre->>'huella_igualdad_hex' !~ '^[0-9a-f]{64}$'
  THEN RAISE EXCEPTION 'Usuarios: sobre correo inválido' USING ERRCODE='22023'; END IF;
  correo:=p_sobre->>'correo_ref';
  IF c.version IS NOT NULL AND c.clave_igualdad_ref IS DISTINCT FROM p_sobre->>'clave_igualdad_ref'
  THEN RAISE EXCEPTION 'Usuarios: generación de igualdad requiere reindexado' USING ERRCODE='55000'; END IF;
  IF EXISTS(SELECT 1 FROM vec_usuarios.correos_direccion e WHERE e.persona_ref=m->>'persona_ref'
      AND e.estado<>'retirado' AND e.huella_igualdad=decode(p_sobre->>'huella_igualdad_hex','hex'))
  THEN RAISE EXCEPTION 'Usuarios: correo ya registrado' USING ERRCODE='P1411'; END IF;
  IF EXISTS(SELECT 1 FROM vec_usuarios.correos_direccion e WHERE e.persona_ref=m->>'persona_ref' AND e.correo_ref=correo)
  THEN RAISE EXCEPTION 'Usuarios: referencia de correo repetida' USING ERRCODE='40001'; END IF;
  SELECT count(*) INTO n FROM vec_usuarios.correos_direccion e WHERE e.persona_ref=m->>'persona_ref' AND e.estado<>'retirado';
  IF n>=5 THEN RAISE EXCEPTION 'Usuarios: máximo de correos' USING ERRCODE='P1412'; END IF;
  IF c.version IS NULL THEN
   INSERT INTO vec_usuarios.correos_conjunto(persona_ref,version,clave_igualdad_ref,actualizado_en)
   VALUES(m->>'persona_ref',v,p_sobre->>'clave_igualdad_ref',ahora);
  END IF;
  INSERT INTO vec_usuarios.correos_direccion(persona_ref,correo_ref,version_sobre,clave_sobre_ref,clave_igualdad_ref,nonce,cifrado,huella_igualdad,estado,activo,creado_en)
  VALUES(m->>'persona_ref',correo,v,p_sobre->>'clave_ref',p_sobre->>'clave_igualdad_ref',decode(p_sobre->>'nonce_hex','hex'),decode(p_sobre->>'cifrado_hex','hex'),
   decode(p_sobre->>'huella_igualdad_hex','hex'),'pendiente',false,ahora);
 ELSE
  IF p_sobre IS NOT NULL AND p_sobre<>'null'::jsonb
  THEN RAISE EXCEPTION 'Usuarios: sobre no esperado' USING ERRCODE='22023'; END IF;
  IF c.version IS NULL THEN RAISE EXCEPTION 'Usuarios: correo ausente' USING ERRCODE='P1409'; END IF;
  correo:=m->>'correo_ref';
  SELECT * INTO d FROM vec_usuarios.correos_direccion WHERE persona_ref=m->>'persona_ref' AND correo_ref=correo FOR UPDATE;
  IF NOT FOUND OR d.estado='retirado' THEN RAISE EXCEPTION 'Usuarios: correo ausente' USING ERRCODE='P1409'; END IF;
 END IF;
 IF accion IN ('vec.correos.anadir','vec.correos.reenviar') THEN
  IF accion='vec.correos.reenviar' THEN
   IF d.estado<>'pendiente' THEN RAISE EXCEPTION 'Usuarios: correo no pendiente' USING ERRCODE='P1409'; END IF;
  END IF;
  IF p_reserva IS NULL OR p_reserva='null'::jsonb
  THEN RAISE EXCEPTION 'Usuarios: desafío ausente' USING ERRCODE='22023'; END IF;
  IF jsonb_typeof(p_reserva) IS DISTINCT FROM 'object'
     OR (SELECT array_agg(x ORDER BY x) FROM jsonb_object_keys(p_reserva) x) IS DISTINCT FROM ARRAY['clave_ref','desafio_ref','huella_codigo_hex','vence_utc']
     OR EXISTS(SELECT 1 FROM jsonb_each(p_reserva) z WHERE jsonb_typeof(z.value)<>'string')
     OR p_reserva->>'desafio_ref' !~ '^desafio:[0-9a-f]{32}$'
     OR p_reserva->>'huella_codigo_hex' !~ '^[0-9a-f]{64}$'
     OR p_reserva->>'clave_ref' !~ '^[A-Za-z0-9:._-]{1,128}$'
  THEN RAISE EXCEPTION 'Usuarios: desafío inválido' USING ERRCODE='22023'; END IF;
  BEGIN vence:=(p_reserva->>'vence_utc')::timestamptz;
  EXCEPTION WHEN others THEN RAISE EXCEPTION 'Usuarios: vigencia inválida' USING ERRCODE='22023'; END;
  IF vence<=ahora+interval '10 minutes' OR vence>ahora+interval '72 hours'
  THEN RAISE EXCEPTION 'Usuarios: vigencia inválida' USING ERRCODE='22023'; END IF;
  SELECT count(*) INTO n FROM vec_usuarios.correos_desafio h
  WHERE h.persona_ref=m->>'persona_ref' AND h.creado_en>ahora-interval '1 hour';
  IF n>=5 THEN RAISE EXCEPTION 'Usuarios: límite de códigos' USING ERRCODE='P1429'; END IF;
  SELECT count(*) INTO n FROM vec_usuarios.correos_desafio h
  WHERE h.persona_ref=m->>'persona_ref' AND h.correo_ref=correo AND h.creado_en>ahora-interval '1 hour';
  IF n>=3 THEN RAISE EXCEPTION 'Usuarios: límite de códigos' USING ERRCODE='P1429'; END IF;
  UPDATE vec_usuarios.correos_desafio SET estado='sustituido'
  WHERE persona_ref=m->>'persona_ref' AND correo_ref=correo AND estado='pendiente';
  desafio:=p_reserva->>'desafio_ref';
  INSERT INTO vec_usuarios.correos_desafio(persona_ref,correo_ref,desafio_ref,huella_codigo,clave_ref,vence_en,estado,intentos,creado_en)
  VALUES(m->>'persona_ref',correo,desafio,decode(p_reserva->>'huella_codigo_hex','hex'),p_reserva->>'clave_ref',vence,'pendiente',0,ahora);
  nuevo_estado:='pendiente'; nuevo_activo:=false;
 ELSE
  IF p_reserva IS NOT NULL AND p_reserva<>'null'::jsonb
  THEN RAISE EXCEPTION 'Usuarios: desafío no esperado' USING ERRCODE='22023'; END IF;
  IF accion='vec.correos.activar' THEN
   IF d.estado<>'verificado' THEN RAISE EXCEPTION 'Usuarios: correo no verificado' USING ERRCODE='P1409'; END IF;
   IF d.activo THEN RAISE EXCEPTION 'Usuarios: correo ya activo' USING ERRCODE='P1409'; END IF;
   SELECT correo_ref INTO anterior FROM vec_usuarios.correos_direccion
   WHERE persona_ref=m->>'persona_ref' AND activo FOR UPDATE;
   UPDATE vec_usuarios.correos_direccion SET activo=false WHERE persona_ref=m->>'persona_ref' AND activo;
   UPDATE vec_usuarios.correos_direccion SET activo=true WHERE persona_ref=m->>'persona_ref' AND correo_ref=correo;
   nuevo_estado:='verificado'; nuevo_activo:=true;
  ELSE
   IF d.activo THEN RAISE EXCEPTION 'Usuarios: correo en uso' USING ERRCODE='P1413'; END IF;
   UPDATE vec_usuarios.correos_direccion SET estado='retirado',retirado_en=ahora
    WHERE persona_ref=m->>'persona_ref' AND correo_ref=correo;
   UPDATE vec_usuarios.correos_desafio SET estado='sustituido'
    WHERE persona_ref=m->>'persona_ref' AND correo_ref=correo AND estado='pendiente';
   nuevo_estado:='retirado'; nuevo_activo:=false;
  END IF;
 END IF;
 IF c.version IS NOT NULL THEN
  UPDATE vec_usuarios.correos_conjunto SET version=v,actualizado_en=ahora
  WHERE persona_ref=m->>'persona_ref' AND version=v-1;
  GET DIAGNOSTICS n=ROW_COUNT;
  IF n<>1 THEN RAISE EXCEPTION 'Usuarios: versión concurrente' USING ERRCODE='40001'; END IF;
 END IF;
 r:=vec_usuarios.registrar_recibo_correos(nuevo_estado,nuevo_activo,correo,anterior,v,ahora);
 IF accion IN ('vec.correos.anadir','vec.correos.reenviar') THEN
  envios:=jsonb_build_array(vec_usuarios.reservar_envio_correos(m->>'persona_ref',m->>'superficie',correo,'verificacion',desafio,r->>'recibo_ref',ahora));
 ELSIF accion='vec.correos.activar' AND anterior IS NOT NULL THEN
  envios:=jsonb_build_array(vec_usuarios.reservar_envio_correos(m->>'persona_ref',m->>'superficie',anterior,'aviso_cambio',NULL,r->>'recibo_ref',ahora));
 END IF;
 PERFORM vec_usuarios.retirar_contexto_correos(m->>'persona_ref','actualizar');
 RETURN r||jsonb_build_object('envios',envios);
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.aplicar_correos_propios_v1(text,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios.aplicar_correos_propios_v1(text,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

-- Verificación en dos llamadas de la MISMA transacción: preparar consume V3,
-- bloquea el desafío y devuelve sus metadatos; Go compara el código en tiempo
-- constante; cerrar consume el intento o el éxito. SQL nunca ve el código.
CREATE FUNCTION vec_usuarios.preparar_verificacion_correo_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; c record; d record; h record; r jsonb;
BEGIN
 m:=vec_usuarios.consumir_contexto_correos(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'verificar');
 IF m->>'accion' IS DISTINCT FROM 'vec.correos.verificar'
 THEN RAISE EXCEPTION 'Usuarios: verificación inválida' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_usuarios:correos:'||(m->>'persona_ref'),0));
 r:=vec_usuarios.replay_correos_autorizado(m);
 IF r IS NOT NULL THEN
  PERFORM vec_usuarios.retirar_contexto_correos(m->>'persona_ref','verificar');
  RETURN jsonb_build_object('replay',r);
 END IF;
 SELECT * INTO c FROM vec_usuarios.correos_conjunto WHERE persona_ref=m->>'persona_ref' FOR UPDATE;
 IF c.version IS NULL OR c.version::text IS DISTINCT FROM m->>'version_esperada'
 THEN RAISE EXCEPTION 'Usuarios: versión correo en conflicto' USING ERRCODE='P1409'; END IF;
 SELECT * INTO d FROM vec_usuarios.correos_direccion WHERE persona_ref=m->>'persona_ref' AND correo_ref=m->>'correo_ref' FOR UPDATE;
 IF NOT FOUND OR d.estado<>'pendiente'
 THEN RAISE EXCEPTION 'Usuarios: correo no pendiente' USING ERRCODE='P1409'; END IF;
 SELECT * INTO h FROM vec_usuarios.correos_desafio WHERE persona_ref=m->>'persona_ref'
  AND correo_ref=m->>'correo_ref' AND estado='pendiente' FOR UPDATE;
 IF NOT FOUND OR h.intentos>=5 OR h.vence_en<=clock_timestamp()
 THEN RAISE EXCEPTION 'Usuarios: código caducado o agotado' USING ERRCODE='P1410'; END IF;
 RETURN jsonb_build_object('persona_ref',h.persona_ref,'correo_ref',h.correo_ref,'desafio_ref',h.desafio_ref,
  'huella_codigo_hex',encode(h.huella_codigo,'hex'),'clave_ref',h.clave_ref,'vence_utc',h.vence_en,
  'intentos',h.intentos);
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.preparar_verificacion_correo_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios.preparar_verificacion_correo_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

-- Sin dirección activa, la primera verificada pasa a ser la activa.
CREATE FUNCTION vec_usuarios.cerrar_verificacion_correo_v1(p_persona text,p_clave text,p_valido boolean)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE c record; h record; v bigint; ahora timestamptz(6); n integer; r jsonb; activar boolean;
BEGIN
 IF vec_usuarios.superficie_sesion_correos() IS NULL
    OR current_setting('transaction_isolation')<>'serializable' OR p_valido IS NULL
 THEN RAISE EXCEPTION 'Usuarios: cierre de verificación denegado' USING ERRCODE='42501'; END IF;
 SELECT * INTO c FROM vec_usuarios.correos_contexto WHERE xid=pg_current_xact_id_if_assigned()
  AND backend_pid=pg_backend_pid() AND sesion=session_user AND persona_ref=p_persona
  AND modo='verificar' AND accion='vec.correos.verificar' AND clave_operacion=p_clave;
 IF NOT FOUND OR c.superficie IS DISTINCT FROM vec_usuarios.superficie_sesion_correos()
 THEN RAISE EXCEPTION 'Usuarios: verificación sin preparación' USING ERRCODE='42501'; END IF;
 SELECT * INTO h FROM vec_usuarios.correos_desafio WHERE persona_ref=c.persona_ref
  AND correo_ref=c.correo_ref AND estado='pendiente' FOR UPDATE;
 IF NOT FOUND OR h.intentos>=5 OR h.vence_en<=clock_timestamp()
 THEN RAISE EXCEPTION 'Usuarios: código caducado o agotado' USING ERRCODE='P1410'; END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 IF NOT p_valido THEN
  UPDATE vec_usuarios.correos_desafio SET intentos=intentos+1,
   estado=CASE WHEN intentos+1>=5 THEN 'agotado' ELSE 'pendiente' END
  WHERE persona_ref=c.persona_ref AND correo_ref=c.correo_ref AND desafio_ref=h.desafio_ref
   AND estado='pendiente' AND intentos<5;
  GET DIAGNOSTICS n=ROW_COUNT;
  IF n<>1 THEN RAISE EXCEPTION 'Usuarios: intento concurrente' USING ERRCODE='40001'; END IF;
  INSERT INTO vec_usuarios.correos_intento_fallido(persona_ref,correo_ref,desafio_ref,numero,
   clave_operacion,huella_clave_ref,huella_valor,decision_ref,auditoria_ref,consumo_huella_sha256,ocurrido_en)
  VALUES(c.persona_ref,c.correo_ref,h.desafio_ref,h.intentos+1,c.clave_operacion,c.huella_clave_ref,
   c.huella_valor,c.decision_ref,c.auditoria_ref,c.consumo_huella_sha256,ahora);
  PERFORM vec_usuarios.retirar_contexto_correos(c.persona_ref,'verificar');
  RETURN jsonb_build_object('valido',false,'intentos_restantes',4-h.intentos);
 END IF;
 SELECT version INTO v FROM vec_usuarios.correos_conjunto WHERE persona_ref=c.persona_ref FOR UPDATE;
 IF v IS DISTINCT FROM c.version_esperada THEN RAISE EXCEPTION 'Usuarios: versión correo en conflicto' USING ERRCODE='P1409'; END IF;
 v:=v+1;
 activar:=NOT EXISTS(SELECT 1 FROM vec_usuarios.correos_direccion WHERE persona_ref=c.persona_ref AND activo);
 UPDATE vec_usuarios.correos_desafio SET estado='usado'
 WHERE persona_ref=c.persona_ref AND correo_ref=c.correo_ref AND desafio_ref=h.desafio_ref AND estado='pendiente';
 UPDATE vec_usuarios.correos_direccion SET estado='verificado',verificado_en=ahora,activo=activar
 WHERE persona_ref=c.persona_ref AND correo_ref=c.correo_ref AND estado='pendiente';
 GET DIAGNOSTICS n=ROW_COUNT;
 IF n<>1 THEN RAISE EXCEPTION 'Usuarios: correo concurrente' USING ERRCODE='40001'; END IF;
 UPDATE vec_usuarios.correos_conjunto SET version=v,actualizado_en=ahora
 WHERE persona_ref=c.persona_ref AND version=v-1;
 GET DIAGNOSTICS n=ROW_COUNT;
 IF n<>1 THEN RAISE EXCEPTION 'Usuarios: versión concurrente' USING ERRCODE='40001'; END IF;
 r:=vec_usuarios.registrar_recibo_correos('verificado',activar,c.correo_ref,NULL,v,ahora);
 PERFORM vec_usuarios.retirar_contexto_correos(c.persona_ref,'verificar');
 RETURN r||jsonb_build_object('envios','[]'::jsonb);
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.cerrar_verificacion_correo_v1(text,text,boolean) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios.cerrar_verificacion_correo_v1(text,text,boolean) TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

-- Anota una sola vez la respuesta del relay. La reserva es la capacidad: sólo
-- la conoce el proceso que ejecutó el efecto autorizado en su superficie.
-- «aceptado» sólo acredita aceptación del relay, nunca entrega ni lectura.
CREATE FUNCTION vec_usuarios.confirmar_envio_correo_v1(p_persona text,p_envio text,p_reserva text,p_aceptado boolean)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE v_superficie text; xid_actual xid8; n integer; huella text;
BEGIN
 v_superficie:=vec_usuarios.superficie_sesion_correos();
 IF v_superficie IS NULL OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off' OR p_aceptado IS NULL
    OR p_persona IS NULL OR p_persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_envio IS NULL OR p_envio !~ '^correo_envio:[0-9a-f]{32}$'
    OR p_reserva IS NULL OR p_reserva !~ '^reserva:[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'Usuarios: confirmación de envío denegada' USING ERRCODE='42501'; END IF;
 xid_actual:=pg_current_xact_id();
 DELETE FROM vec_usuarios.correos_contexto WHERE backend_pid=pg_backend_pid() AND xid<>xid_actual;
 huella:=encode(sha256(convert_to(p_envio,'UTF8')),'hex');
 INSERT INTO vec_usuarios.correos_contexto(xid,backend_pid,sesion,superficie,persona_ref,modo,accion,decision_ref,auditoria_ref,consumo_huella_sha256)
 VALUES(xid_actual,pg_backend_pid(),session_user,v_superficie,p_persona,'envio','vec.correos.envio',p_envio,p_envio,huella);
 UPDATE vec_usuarios.correos_envio SET estado=CASE WHEN p_aceptado THEN 'aceptado' ELSE 'no_aceptado' END,
  resuelto_en=date_trunc('microseconds',clock_timestamp())
 WHERE envio_ref=p_envio AND persona_ref=p_persona AND reserva_ref=p_reserva
  AND superficie=v_superficie AND estado='reservado';
 GET DIAGNOSTICS n=ROW_COUNT;
 PERFORM vec_usuarios.retirar_contexto_correos(p_persona,'envio');
 IF n<>1 THEN RAISE EXCEPTION 'Usuarios: envío desconocido o ya anotado' USING ERRCODE='42501'; END IF;
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.confirmar_envio_correo_v1(text,text,text,boolean) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios.confirmar_envio_correo_v1(text,text,text,boolean) TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

DO $acl$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p WHERE p.pronamespace='vec_usuarios'::regnamespace
     AND (p.proname LIKE '%correo%') AND p.proname NOT IN
      ('recuperar_correos_operacion_v1','consultar_correos_propios_v1','aplicar_correos_propios_v1',
       'preparar_verificacion_correo_v1','cerrar_verificacion_correo_v1','confirmar_envio_correo_v1')
     AND (has_function_privilege('vec_usuarios_ejecutor_interno',p.oid,'EXECUTE')
       OR has_function_privilege('vec_usuarios_ejecutor_externo',p.oid,'EXECUTE')))
    OR EXISTS(SELECT 1 FROM pg_class c WHERE c.relnamespace='vec_usuarios'::regnamespace AND c.relname LIKE 'correos_%'
     AND (has_table_privilege('vec_usuarios_ejecutor_interno',c.oid,'SELECT,INSERT,UPDATE,DELETE')
       OR has_table_privilege('vec_usuarios_ejecutor_externo',c.oid,'SELECT,INSERT,UPDATE,DELETE')))
 THEN RAISE EXCEPTION 'Usuarios 000004: ACL de correos incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
