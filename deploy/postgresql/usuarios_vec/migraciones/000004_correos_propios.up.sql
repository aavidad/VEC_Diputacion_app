\set ON_ERROR_STOP on
-- Usuarios 000004: aplicar tras AD3-107, Usuarios 000001/000002 y la auditoría
-- 000003. Cada llamada usa una transacción SERIALIZABLE READ WRITE.
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
    OR to_regclass('vec_usuarios.correos_conjunto') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_correos_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_despachador' AND NOT rolcanlogin AND NOT rolbypassrls)
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
 correo_ref text NOT NULL CHECK(correo_ref ~ '^[A-Za-z0-9:._-]{12,128}$'),
 version_sobre bigint NOT NULL CHECK(version_sobre>0),
 clave_sobre_ref text NOT NULL CHECK(length(clave_sobre_ref) BETWEEN 1 AND 128),
 clave_igualdad_ref text NOT NULL CHECK(clave_igualdad_ref ~ '^[A-Za-z0-9:._-]{1,128}$'),
 CHECK(clave_sobre_ref<>clave_igualdad_ref),
 nonce bytea NOT NULL CHECK(octet_length(nonce) BETWEEN 12 AND 32),
 cifrado bytea NOT NULL CHECK(octet_length(cifrado) BETWEEN 16 AND 4096),
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
 CHECK(NOT activo OR (estado='verificado' AND verificado_en IS NOT NULL)),
 CHECK((estado='pendiente' AND verificado_en IS NULL AND retirado_en IS NULL)
    OR (estado='verificado' AND verificado_en IS NOT NULL AND retirado_en IS NULL)
    OR (estado='retirado' AND retirado_en IS NOT NULL))
);
CREATE UNIQUE INDEX correos_un_activo ON vec_usuarios.correos_direccion(persona_ref) WHERE activo;
CREATE UNIQUE INDEX correos_un_no_retirado ON vec_usuarios.correos_direccion(persona_ref,huella_igualdad) WHERE estado<>'retirado';
CREATE TABLE vec_usuarios.correos_desafio (
 persona_ref text NOT NULL,
 correo_ref text NOT NULL,
 desafio_ref text NOT NULL CHECK(length(desafio_ref) BETWEEN 16 AND 128),
 huella_codigo bytea NOT NULL CHECK(octet_length(huella_codigo)=32),
 clave_ref text NOT NULL CHECK(length(clave_ref) BETWEEN 1 AND 128),
 vence_en timestamptz(6) NOT NULL,
 estado text NOT NULL CHECK(estado IN ('pendiente','usado','sustituido','agotado')),
 intentos smallint NOT NULL DEFAULT 0 CHECK(intentos BETWEEN 0 AND 5),
 creado_en timestamptz(6) NOT NULL,
 PRIMARY KEY(persona_ref,correo_ref,desafio_ref),
 FOREIGN KEY(persona_ref,correo_ref) REFERENCES vec_usuarios.correos_direccion(persona_ref,correo_ref),
 CHECK(vence_en>creado_en AND vence_en<=creado_en+interval '72 hours')
);
CREATE UNIQUE INDEX correos_un_desafio_pendiente ON vec_usuarios.correos_desafio(persona_ref,correo_ref) WHERE estado='pendiente';
CREATE TABLE vec_usuarios.correos_intento_fallido (
 persona_ref text NOT NULL,
 correo_ref text NOT NULL,
 desafio_ref text NOT NULL,
 numero smallint NOT NULL CHECK(numero BETWEEN 1 AND 5),
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 ocurrido_en timestamptz(6) NOT NULL,
 PRIMARY KEY(persona_ref,correo_ref,desafio_ref,numero),
 FOREIGN KEY(persona_ref,correo_ref,desafio_ref)
  REFERENCES vec_usuarios.correos_desafio(persona_ref,correo_ref,desafio_ref)
);
-- Ventana deslizante de una hora. Los registros no se borran al reenviar.
CREATE TABLE vec_usuarios.correos_reenvio (
 persona_ref text NOT NULL,
 correo_ref text NOT NULL,
 desafio_ref text NOT NULL,
 ocurrido_en timestamptz(6) NOT NULL,
 PRIMARY KEY(persona_ref,correo_ref,desafio_ref),
 FOREIGN KEY(persona_ref,correo_ref,desafio_ref) REFERENCES vec_usuarios.correos_desafio(persona_ref,correo_ref,desafio_ref)
);
CREATE TABLE vec_usuarios.correos_historia (
 persona_ref text NOT NULL REFERENCES vec_usuarios.correos_conjunto(persona_ref),
 version bigint NOT NULL CHECK(version>0),
 accion text NOT NULL CHECK(accion IN ('vec.correos.anadir','vec.correos.reenviar','vec.correos.verificar','vec.correos.activar','vec.correos.retirar')),
 correo_ref text NOT NULL,
 estado_resultante text NOT NULL CHECK(estado_resultante IN ('pendiente','verificado','retirado')),
 activo_resultante boolean NOT NULL,
 anterior_activo_ref text,
 sustituto_ref text,
 recibo_ref text NOT NULL UNIQUE,
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(persona_ref,version),
 UNIQUE(persona_ref,version,accion,correo_ref,recibo_ref),
 CHECK((accion='vec.correos.retirar' AND (sustituto_ref IS NULL OR sustituto_ref<>correo_ref))
    OR (accion<>'vec.correos.retirar' AND sustituto_ref IS NULL)),
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
CREATE TABLE vec_usuarios.correos_outbox (
 outbox_ref text PRIMARY KEY CHECK(outbox_ref ~ '^correo_outbox:[0-9a-f]{32}$'),
 persona_ref text NOT NULL,
 correo_ref text NOT NULL,
 desafio_ref text,
 desafio bytea,
 clave_ref text,
 vence_en timestamptz(6),
 tipo text NOT NULL CHECK(tipo IN ('verificacion','aviso_anterior')),
 estado text NOT NULL DEFAULT 'pendiente' CHECK(estado IN ('pendiente','reservado','aceptado')),
 reserva_ref text,
 reservado_en timestamptz(6),
 aceptado_en timestamptz(6),
 creado_en timestamptz(6) NOT NULL,
 CHECK((tipo='verificacion' AND desafio_ref IS NOT NULL AND desafio IS NOT NULL AND octet_length(desafio)>=16 AND clave_ref IS NOT NULL AND vence_en IS NOT NULL)
    OR (tipo='aviso_anterior' AND desafio_ref IS NULL AND desafio IS NULL AND clave_ref IS NULL AND vence_en IS NULL)),
 CHECK((estado='pendiente' AND reserva_ref IS NULL AND reservado_en IS NULL AND aceptado_en IS NULL)
    OR (estado='reservado' AND reserva_ref IS NOT NULL AND reservado_en IS NOT NULL AND aceptado_en IS NULL)
    OR (estado='aceptado' AND reserva_ref IS NOT NULL AND reservado_en IS NOT NULL AND aceptado_en IS NOT NULL))
);
CREATE UNIQUE INDEX correos_outbox_reserva_unica ON vec_usuarios.correos_outbox(reserva_ref) WHERE reserva_ref IS NOT NULL;
CREATE TABLE vec_usuarios.correos_despacho_historia (
 outbox_ref text NOT NULL REFERENCES vec_usuarios.correos_outbox(outbox_ref),
 secuencia smallint NOT NULL CHECK(secuencia IN (1,2)),
 persona_ref text NOT NULL,
 reserva_ref text NOT NULL,
 evento text NOT NULL CHECK(evento IN ('reservado','smtp_aceptado')),
 ocurrido_en timestamptz(6) NOT NULL,
 PRIMARY KEY(outbox_ref,secuencia),
 CHECK((secuencia=1 AND evento='reservado') OR (secuencia=2 AND evento='smtp_aceptado'))
);
CREATE TABLE vec_usuarios.correos_contexto (
 xid xid8 NOT NULL,
 backend_pid integer NOT NULL,
 sesion text NOT NULL,
 superficie text NOT NULL CHECK(superficie IN ('interna_corporativa','externa_personal')),
 persona_ref text NOT NULL,
 modo text NOT NULL CHECK(modo IN ('consultar','recuperar','actualizar','verificar','activo_ct','activo_bolsa','despacho')),
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
 FOREACH t IN ARRAY ARRAY['correos_conjunto','correos_direccion','correos_desafio','correos_intento_fallido','correos_reenvio','correos_historia','correos_recibo','correos_outbox','correos_despacho_historia','correos_contexto'] LOOP
  EXECUTE format('ALTER TABLE vec_usuarios.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_usuarios.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_usuarios.%I FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo,vec_usuarios_despachador',t);
 END LOOP;
END $rls$;
GRANT USAGE ON SCHEMA vec_usuarios TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;
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
REVOKE ALL ON FUNCTION vec_usuarios.contexto_autorizado_correos(text,text[]) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo,vec_usuarios_despachador;
CREATE POLICY correos_contexto_lectura ON vec_usuarios.correos_contexto FOR SELECT TO vec_usuarios_propietario
 USING (backend_pid=pg_backend_pid() AND sesion=session_user AND superficie=vec_usuarios.superficie_sesion_correos());
CREATE POLICY correos_contexto_alta ON vec_usuarios.correos_contexto FOR INSERT TO vec_usuarios_propietario
 WITH CHECK (xid=pg_current_xact_id_if_assigned() AND backend_pid=pg_backend_pid() AND sesion=session_user
  AND superficie=vec_usuarios.superficie_sesion_correos());
CREATE POLICY correos_contexto_baja ON vec_usuarios.correos_contexto FOR DELETE TO vec_usuarios_propietario
 USING (backend_pid=pg_backend_pid() AND sesion=session_user AND superficie=vec_usuarios.superficie_sesion_correos());
CREATE POLICY correos_conjunto_lectura ON vec_usuarios.correos_conjunto FOR SELECT TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['consultar','recuperar','actualizar','verificar','activo_ct','activo_bolsa','despacho']));
CREATE POLICY correos_conjunto_alta ON vec_usuarios.correos_conjunto FOR INSERT TO vec_usuarios_propietario WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar']));
CREATE POLICY correos_conjunto_cambio ON vec_usuarios.correos_conjunto FOR UPDATE TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar','verificar'])) WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar','verificar']));
CREATE POLICY correos_direccion_lectura ON vec_usuarios.correos_direccion FOR SELECT TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['consultar','actualizar','verificar','activo_ct','activo_bolsa','despacho']));
CREATE POLICY correos_direccion_alta ON vec_usuarios.correos_direccion FOR INSERT TO vec_usuarios_propietario WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar']));
CREATE POLICY correos_direccion_cambio ON vec_usuarios.correos_direccion FOR UPDATE TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar','verificar'])) WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar','verificar']));
CREATE POLICY correos_desafio_lectura ON vec_usuarios.correos_desafio FOR SELECT TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar','verificar','despacho']));
CREATE POLICY correos_desafio_alta ON vec_usuarios.correos_desafio FOR INSERT TO vec_usuarios_propietario WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar']));
CREATE POLICY correos_desafio_cambio ON vec_usuarios.correos_desafio FOR UPDATE TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar','verificar'])) WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar','verificar']));
CREATE POLICY correos_intento_fallido_alta ON vec_usuarios.correos_intento_fallido FOR INSERT TO vec_usuarios_propietario WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['verificar']));
CREATE POLICY correos_reenvio_lectura ON vec_usuarios.correos_reenvio FOR SELECT TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar']));
CREATE POLICY correos_reenvio_alta ON vec_usuarios.correos_reenvio FOR INSERT TO vec_usuarios_propietario WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar']));
CREATE POLICY correos_historia_alta ON vec_usuarios.correos_historia FOR INSERT TO vec_usuarios_propietario WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar','verificar']));
CREATE POLICY correos_recibo_lectura ON vec_usuarios.correos_recibo FOR SELECT TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['recuperar','actualizar','verificar']));
CREATE POLICY correos_recibo_alta ON vec_usuarios.correos_recibo FOR INSERT TO vec_usuarios_propietario WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar','verificar']));
CREATE POLICY correos_outbox_lectura ON vec_usuarios.correos_outbox FOR SELECT TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['despacho']));
CREATE POLICY correos_outbox_alta ON vec_usuarios.correos_outbox FOR INSERT TO vec_usuarios_propietario WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['actualizar']));
CREATE POLICY correos_outbox_cambio ON vec_usuarios.correos_outbox FOR UPDATE TO vec_usuarios_propietario USING (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['despacho'])) WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['despacho']));
CREATE POLICY correos_despacho_historia_alta ON vec_usuarios.correos_despacho_historia FOR INSERT TO vec_usuarios_propietario WITH CHECK (vec_usuarios.contexto_autorizado_correos(persona_ref,ARRAY['despacho']));
CREATE TRIGGER correos_historia_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios.correos_historia FOR EACH ROW EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER correos_recibo_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios.correos_recibo FOR EACH ROW EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER correos_reenvio_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios.correos_reenvio FOR EACH ROW EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER correos_intento_fallido_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios.correos_intento_fallido FOR EACH ROW EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER correos_historia_no_truncar BEFORE TRUNCATE ON vec_usuarios.correos_historia FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER correos_recibo_no_truncar BEFORE TRUNCATE ON vec_usuarios.correos_recibo FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER correos_reenvio_no_truncar BEFORE TRUNCATE ON vec_usuarios.correos_reenvio FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER correos_intento_fallido_no_truncar BEFORE TRUNCATE ON vec_usuarios.correos_intento_fallido FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER correos_despacho_historia_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios.correos_despacho_historia FOR EACH ROW EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER correos_despacho_historia_no_truncar BEFORE TRUNCATE ON vec_usuarios.correos_despacho_historia FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();

CREATE FUNCTION vec_usuarios.huella_contexto_correos(p_material text)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE persona text; canon text;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 8192
 THEN RAISE EXCEPTION 'Usuarios: material correo inválido' USING ERRCODE='22023'; END IF;
 BEGIN persona:=p_material::jsonb->>'persona_ref';
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Usuarios: material correo inválido' USING ERRCODE='22023'; END;
 IF persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
 THEN RAISE EXCEPTION 'Usuarios: persona inválida' USING ERRCODE='22023'; END IF;
 canon:='{"ambitos":{"persona_ref":'||to_jsonb(persona)::text||'},"atributos":{"material_sha256":"'||
 encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}';
 RETURN encode(sha256(convert_to(canon,'UTF8')),'hex');
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.huella_contexto_correos(text) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

CREATE FUNCTION vec_usuarios.validar_material_correos(
 p_material text,p_capacidad bytea,p_decision bytea,p_persona_version numeric,p_perfil_version numeric)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE m jsonb; c jsonb; d jsonb; k text[]; h text; accion text; finalidad text; campos jsonb; sello jsonb; claves text[];
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 8192
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_persona_version IS NULL OR p_perfil_version IS NULL
 THEN RAISE EXCEPTION 'Usuarios: material correo denegado' USING ERRCODE='42501'; END IF;
 BEGIN m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Usuarios: material correo inválido' USING ERRCODE='22023'; END;
 SELECT array_agg(x ORDER BY x) INTO k FROM jsonb_object_keys(m) x;
 accion:=m->>'accion';
 SELECT x.finalidad,x.campos INTO finalidad,campos FROM (VALUES
 ('vec.correos.consultar','finalidad:usuarios:correos-propios:v1','["activo","correo_ref","direccion","estado","version"]'::jsonb),
 ('vec.correos.anadir','finalidad:usuarios:correos-propios:v1','["correo_ref","direccion","estado","version"]'::jsonb),
 ('vec.correos.reenviar','finalidad:usuarios:correos-propios:v1','["correo_ref","estado","version"]'::jsonb),
 ('vec.correos.verificar','finalidad:usuarios:correos-propios:v1','["correo_ref","estado","version"]'::jsonb),
 ('vec.correos.activar','finalidad:usuarios:correos-propios:v1','["activo","correo_ref","version"]'::jsonb),
 ('vec.correos.retirar','finalidad:usuarios:correos-propios:v1','["activo","correo_ref","estado","version"]'::jsonb)
 ) x(accion,finalidad,campos) WHERE x.accion=m->>'accion';
 h:=vec_usuarios.huella_contexto_correos(p_material);
 IF k IS DISTINCT FROM ARRAY['accion','clave_operacion','correo_ref','finalidad_ref','huellas_peticion','perfil_ref','persona_ref','superficie','sustituto_ref','version_esperada']
    OR EXISTS(SELECT 1 FROM jsonb_each(m) z WHERE z.key IN
      ('accion','clave_operacion','correo_ref','finalidad_ref','perfil_ref','persona_ref','superficie','sustituto_ref')
      AND jsonb_typeof(z.value)<>'string')
    OR jsonb_typeof(m->'version_esperada') IS DISTINCT FROM 'number'
    OR m->>'superficie' IS DISTINCT FROM vec_usuarios.superficie_sesion_correos()
    OR m->>'superficie' IS DISTINCT FROM d #>> '{vinculo_autenticacion_actor,superficie}'
    OR c->>'audiencia_consumo' IS DISTINCT FROM
       'vec_usuarios.correos.'||split_part(accion,'.',3)||'.'||(m->>'superficie')||'.v1'
    OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR m->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
    OR finalidad IS NULL OR m->>'finalidad_ref' IS DISTINCT FROM finalidad
    OR m->>'persona_ref' IS DISTINCT FROM d->>'principal_id'
    OR m->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
    OR m->>'persona_ref' IS DISTINCT FROM d->>'recurso_ref'
    OR m->>'persona_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'accion' IS DISTINCT FROM accion OR c->>'operacion' IS DISTINCT FROM accion
    OR d->>'modulo_id' IS DISTINCT FROM 'usuarios' OR d->>'tipo_recurso' IS DISTINCT FROM 'correos_persona'
    OR d->>'finalidad' IS DISTINCT FROM finalidad OR d->'concedida' IS DISTINCT FROM 'true'::jsonb
    OR d->'campos_permitidos' IS DISTINCT FROM campos OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'decision_ref' IS NULL OR length(d->>'decision_ref') NOT BETWEEN 1 AND 256
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h OR c->>'huella_efecto_sha256' IS DISTINCT FROM h
    OR m->>'version_esperada' !~ '^(0|[1-9][0-9]{0,18})$'
    OR (CASE WHEN m->>'version_esperada' ~ '^(0|[1-9][0-9]{0,18})$'
       THEN (m->>'version_esperada')::numeric>=9223372036854775807::numeric ELSE true END)
 THEN RAISE EXCEPTION 'Usuarios: correo no autorizado' USING ERRCODE='42501'; END IF;
 IF accion='vec.correos.consultar' THEN
  IF coalesce(m->>'clave_operacion','')<>'' OR coalesce(m->>'correo_ref','')<>''
     OR coalesce(m->>'sustituto_ref','')<>'' OR m->'huellas_peticion' IS DISTINCT FROM '{}'::jsonb
  THEN RAISE EXCEPTION 'Usuarios: consulta correo inválida' USING ERRCODE='22023'; END IF;
 ELSE
  IF m->>'clave_operacion' !~ '^[A-Za-z0-9:._-]{16,128}$'
     OR jsonb_typeof(m->'huellas_peticion') IS DISTINCT FROM 'object'
     OR (SELECT array_agg(x ORDER BY x) FROM jsonb_object_keys(m->'huellas_peticion') x) IS DISTINCT FROM ARRAY['activa','retenidas']
     OR jsonb_typeof(m#>'{huellas_peticion,retenidas}') IS DISTINCT FROM 'array'
     OR jsonb_array_length(m#>'{huellas_peticion,retenidas}')>8
     OR (accion='vec.correos.anadir' AND coalesce(m->>'correo_ref','')<>'')
     OR (accion<>'vec.correos.anadir' AND m->>'correo_ref' !~ '^[A-Za-z0-9:._-]{12,128}$')
     OR (accion<>'vec.correos.retirar' AND coalesce(m->>'sustituto_ref','')<>'')
     OR (accion='vec.correos.retirar' AND coalesce(m->>'sustituto_ref','')<>'' AND (m->>'sustituto_ref' !~ '^[A-Za-z0-9:._-]{12,128}$' OR m->>'sustituto_ref'=m->>'correo_ref'))
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
 IF current_setting('transaction_isolation')<>'serializable' OR p_modo NOT IN ('consultar','recuperar','actualizar','verificar','activo_ct','activo_bolsa')
    OR p_m->>'superficie' IS DISTINCT FROM vec_usuarios.superficie_sesion_correos()
    OR p_x->>'decision_ref' IS NULL OR p_x->>'auditoria_ref' IS NULL
    OR p_x->>'consumo_huella_sha256' !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'Usuarios: contexto correo denegado' USING ERRCODE='42501'; END IF;
 xid_actual:=pg_current_xact_id();
 DELETE FROM vec_usuarios.correos_contexto WHERE backend_pid=pg_backend_pid() AND xid<>xid_actual;
 INSERT INTO vec_usuarios.correos_contexto(xid,backend_pid,sesion,superficie,persona_ref,modo,accion,clave_operacion,correo_ref,version_esperada,huella_clave_ref,huella_valor,decision_ref,auditoria_ref,consumo_huella_sha256)
 VALUES(xid_actual,pg_backend_pid(),session_user,p_m->>'superficie',p_m->>'persona_ref',p_modo,p_m->>'accion',p_m->>'clave_operacion',p_m->>'correo_ref',(p_m->>'version_esperada')::bigint,
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

CREATE FUNCTION vec_usuarios.replay_correos_autorizado(p_m jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE r record; coincide boolean;
BEGIN
 SELECT * INTO r FROM vec_usuarios.correos_recibo
 WHERE persona_ref=p_m->>'persona_ref' AND clave_operacion=p_m->>'clave_operacion';
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT EXISTS(SELECT 1 FROM jsonb_array_elements(jsonb_build_array(p_m#>'{huellas_peticion,activa}') || (p_m#>'{huellas_peticion,retenidas}')) h
  WHERE h->>'clave_ref'=r.huella_clave_ref AND h->>'valor'=r.huella_valor) INTO coincide;
 IF NOT coincide OR r.accion IS DISTINCT FROM p_m->>'accion'
 THEN RAISE EXCEPTION 'Usuarios: clave reutilizada con otra petición' USING ERRCODE='P1409'; END IF;
 RETURN jsonb_build_object('recibo_ref',r.recibo_ref,'persona_ref',r.persona_ref,'accion',r.accion,
  'correo_ref',r.correo_ref,'version',r.version,'fecha_utc',r.registrada_en,'replay',true);
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

CREATE FUNCTION vec_usuarios.consultar_correos_propios_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; v bigint; correos jsonb;
BEGIN
 m:=vec_usuarios.consumir_contexto_correos(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'consultar');
 IF m->>'accion' IS DISTINCT FROM 'vec.correos.consultar'
 THEN RAISE EXCEPTION 'Usuarios: consulta correo inválida' USING ERRCODE='42501'; END IF;
 SELECT version INTO v FROM vec_usuarios.correos_conjunto WHERE persona_ref=m->>'persona_ref';
 SELECT coalesce(jsonb_agg(jsonb_build_object('correo_ref',d.correo_ref,'estado',d.estado,'activo',d.activo,
  'creado_utc',d.creado_en,'verificado_utc',d.verificado_en,
  'sobre',jsonb_build_object('version',d.version_sobre,'clave_ref',d.clave_sobre_ref,
   'nonce_hex',encode(d.nonce,'hex'),'cifrado_hex',encode(d.cifrado,'hex'))) ORDER BY d.creado_en,d.correo_ref),'[]'::jsonb)
 INTO correos FROM vec_usuarios.correos_direccion d WHERE d.persona_ref=m->>'persona_ref' AND d.estado<>'retirado';
 PERFORM vec_usuarios.retirar_contexto_correos(m->>'persona_ref','consultar');
 RETURN jsonb_build_object('persona_ref',m->>'persona_ref','version',coalesce(v,0),'correos',correos);
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.consultar_correos_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios.consultar_correos_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

CREATE FUNCTION vec_usuarios.registrar_recibo_correos(p_m jsonb,p_estado text,p_activo boolean,p_correo_ref text,p_anterior_activo_ref text,p_version bigint,p_fecha timestamptz)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE c record; ref text;
BEGIN
 SELECT * INTO STRICT c FROM vec_usuarios.correos_contexto WHERE xid=pg_current_xact_id_if_assigned()
 AND backend_pid=pg_backend_pid() AND sesion=session_user AND persona_ref=p_m->>'persona_ref'
 AND modo IN ('actualizar','verificar');
 IF c.accion IS DISTINCT FROM p_m->>'accion' OR c.huella_valor IS NULL OR c.huella_clave_ref IS NULL
 THEN RAISE EXCEPTION 'Usuarios: recibo sin contexto' USING ERRCODE='42501'; END IF;
 ref:='correo_recibo:'||replace(gen_random_uuid()::text,'-','');
 INSERT INTO vec_usuarios.correos_historia(persona_ref,version,accion,correo_ref,estado_resultante,activo_resultante,anterior_activo_ref,sustituto_ref,recibo_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(c.persona_ref,p_version,c.accion,p_correo_ref,p_estado,p_activo,p_anterior_activo_ref,
  nullif(p_m->>'sustituto_ref',''),ref,c.decision_ref,c.auditoria_ref,p_fecha);
 INSERT INTO vec_usuarios.correos_recibo(persona_ref,clave_operacion,huella_clave_ref,huella_valor,recibo_ref,accion,correo_ref,version,decision_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
 VALUES(c.persona_ref,c.clave_operacion,c.huella_clave_ref,c.huella_valor,ref,c.accion,p_correo_ref,p_version,c.decision_ref,c.auditoria_ref,c.consumo_huella_sha256,p_fecha);
 RETURN jsonb_build_object('recibo_ref',ref,'persona_ref',c.persona_ref,'accion',c.accion,
  'correo_ref',p_correo_ref,'version',p_version,'fecha_utc',p_fecha,'replay',false);
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.registrar_recibo_correos(jsonb,text,boolean,text,text,bigint,timestamptz) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

CREATE FUNCTION vec_usuarios.aplicar_correos_propios_v1(
 p_material text,p_sobre jsonb,p_reserva jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre_v3 bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; c record; d record; sustituto record; desafio record; v bigint; ahora timestamptz(6);
 correo text; accion text; nuevo_estado text; nuevo_activo boolean; anterior text; n integer; r jsonb;
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
  RETURN r;
 END IF;
 SELECT * INTO c FROM vec_usuarios.correos_conjunto WHERE persona_ref=m->>'persona_ref' FOR UPDATE;
 IF coalesce(c.version,0)::text IS DISTINCT FROM m->>'version_esperada'
 THEN RAISE EXCEPTION 'Usuarios: versión correo en conflicto' USING ERRCODE='P1409'; END IF;
 v:=coalesce(c.version,0)+1; ahora:=date_trunc('microseconds',clock_timestamp());
 IF accion='vec.correos.anadir' THEN
  IF jsonb_typeof(p_sobre) IS DISTINCT FROM 'object'
     OR (SELECT array_agg(x ORDER BY x) FROM jsonb_object_keys(p_sobre) x) IS DISTINCT FROM ARRAY['cifrado_hex','clave_igualdad_ref','clave_ref','correo_ref','huella_igualdad_hex','nonce_hex','version']
     OR EXISTS(SELECT 1 FROM jsonb_each(p_sobre) z WHERE z.key IN
       ('cifrado_hex','clave_igualdad_ref','clave_ref','correo_ref','huella_igualdad_hex','nonce_hex') AND jsonb_typeof(z.value)<>'string')
     OR jsonb_typeof(p_sobre->'version') IS DISTINCT FROM 'number'
     OR p_sobre->>'correo_ref' !~ '^[A-Za-z0-9:._-]{12,128}$'
     OR p_sobre->>'version' IS DISTINCT FROM v::text
     OR p_sobre->>'clave_ref' !~ '^[A-Za-z0-9:._-]{1,128}$'
     OR p_sobre->>'clave_igualdad_ref' !~ '^[A-Za-z0-9:._-]{1,128}$'
     OR p_sobre->>'clave_igualdad_ref' IS NOT DISTINCT FROM p_sobre->>'clave_ref'
     OR p_sobre->>'nonce_hex' !~ '^[0-9a-f]{24,64}$' OR length(p_sobre->>'nonce_hex')%2<>0
     OR p_sobre->>'cifrado_hex' !~ '^[0-9a-f]+$' OR length(p_sobre->>'cifrado_hex') NOT BETWEEN 32 AND 8192 OR length(p_sobre->>'cifrado_hex')%2<>0
     OR p_sobre->>'huella_igualdad_hex' !~ '^[0-9a-f]{64}$'
  THEN RAISE EXCEPTION 'Usuarios: sobre correo inválido' USING ERRCODE='22023'; END IF;
  correo:=p_sobre->>'correo_ref';
  IF c.version IS NOT NULL AND c.clave_igualdad_ref IS DISTINCT FROM p_sobre->>'clave_igualdad_ref'
  THEN RAISE EXCEPTION 'Usuarios: generación de igualdad requiere reindexado' USING ERRCODE='P1409'; END IF;
  IF EXISTS(SELECT 1 FROM vec_usuarios.correos_direccion e WHERE e.persona_ref=m->>'persona_ref'
      AND (e.correo_ref=correo OR (e.estado<>'retirado'
       AND e.huella_igualdad=decode(p_sobre->>'huella_igualdad_hex','hex'))))
  THEN RAISE EXCEPTION 'Usuarios: correo ya registrado' USING ERRCODE='P1409'; END IF;
  IF c.version IS NULL THEN
   INSERT INTO vec_usuarios.correos_conjunto(persona_ref,version,clave_igualdad_ref,actualizado_en)
   VALUES(m->>'persona_ref',v,p_sobre->>'clave_igualdad_ref',ahora);
  END IF;
  INSERT INTO vec_usuarios.correos_direccion(persona_ref,correo_ref,version_sobre,clave_sobre_ref,clave_igualdad_ref,nonce,cifrado,huella_igualdad,estado,activo,creado_en)
  VALUES(m->>'persona_ref',correo,v,p_sobre->>'clave_ref',p_sobre->>'clave_igualdad_ref',decode(p_sobre->>'nonce_hex','hex'),decode(p_sobre->>'cifrado_hex','hex'),
   decode(p_sobre->>'huella_igualdad_hex','hex'),'pendiente',false,ahora);
  nuevo_estado:='pendiente'; nuevo_activo:=false;
 ELSE
  IF p_sobre IS NOT NULL AND p_sobre<>'{}'::jsonb
  THEN RAISE EXCEPTION 'Usuarios: sobre no esperado' USING ERRCODE='22023'; END IF;
  IF c.version IS NULL THEN RAISE EXCEPTION 'Usuarios: correo ausente' USING ERRCODE='P1409'; END IF;
  correo:=m->>'correo_ref';
  SELECT * INTO d FROM vec_usuarios.correos_direccion WHERE persona_ref=m->>'persona_ref' AND correo_ref=correo FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION 'Usuarios: correo ausente' USING ERRCODE='P1409'; END IF;
 END IF;
 IF accion IN ('vec.correos.anadir','vec.correos.reenviar') THEN
  IF accion='vec.correos.reenviar' THEN
   IF d.estado<>'pendiente' THEN RAISE EXCEPTION 'Usuarios: correo no pendiente' USING ERRCODE='P1409'; END IF;
  END IF;
  IF jsonb_typeof(p_reserva) IS DISTINCT FROM 'object'
     OR (SELECT array_agg(x ORDER BY x) FROM jsonb_object_keys(p_reserva) x) IS DISTINCT FROM ARRAY['clave_ref','desafio_hex','desafio_ref','huella_codigo_hex','vence_utc']
     OR EXISTS(SELECT 1 FROM jsonb_each(p_reserva) z WHERE jsonb_typeof(z.value)<>'string')
     OR p_reserva->>'desafio_ref' !~ '^[A-Za-z0-9:._-]{16,128}$'
     OR p_reserva->>'desafio_hex' !~ '^[0-9a-f]+$' OR length(p_reserva->>'desafio_hex') NOT BETWEEN 32 AND 512 OR length(p_reserva->>'desafio_hex')%2<>0
     OR p_reserva->>'huella_codigo_hex' !~ '^[0-9a-f]{64}$'
     OR p_reserva->>'clave_ref' !~ '^[A-Za-z0-9:._-]{1,128}$'
  THEN RAISE EXCEPTION 'Usuarios: desafío inválido' USING ERRCODE='22023'; END IF;
  BEGIN
   IF (p_reserva->>'vence_utc')::timestamptz<=ahora+interval '14 minutes'
      OR (p_reserva->>'vence_utc')::timestamptz>ahora+interval '72 hours'
   THEN RAISE EXCEPTION 'Usuarios: vigencia inválida' USING ERRCODE='22023'; END IF;
  EXCEPTION WHEN invalid_datetime_format OR datetime_field_overflow THEN
   RAISE EXCEPTION 'Usuarios: vigencia inválida' USING ERRCODE='22023'; END;
  IF accion='vec.correos.reenviar' THEN
   SELECT count(*) INTO n FROM vec_usuarios.correos_reenvio
   WHERE persona_ref=m->>'persona_ref' AND correo_ref=correo AND ocurrido_en>ahora-interval '1 hour';
   IF n>=3 THEN RAISE EXCEPTION 'Usuarios: límite de reenvío' USING ERRCODE='P1429'; END IF;
   UPDATE vec_usuarios.correos_desafio SET estado='sustituido'
   WHERE persona_ref=m->>'persona_ref' AND correo_ref=correo AND estado='pendiente';
  END IF;
  INSERT INTO vec_usuarios.correos_desafio(persona_ref,correo_ref,desafio_ref,huella_codigo,clave_ref,vence_en,estado,intentos,creado_en)
  VALUES(m->>'persona_ref',correo,p_reserva->>'desafio_ref',decode(p_reserva->>'huella_codigo_hex','hex'),p_reserva->>'clave_ref',
   (p_reserva->>'vence_utc')::timestamptz,'pendiente',0,ahora);
  IF accion='vec.correos.reenviar' THEN
   INSERT INTO vec_usuarios.correos_reenvio(persona_ref,correo_ref,desafio_ref,ocurrido_en)
   VALUES(m->>'persona_ref',correo,p_reserva->>'desafio_ref',ahora);
  END IF;
  INSERT INTO vec_usuarios.correos_outbox(outbox_ref,persona_ref,correo_ref,desafio_ref,desafio,clave_ref,vence_en,tipo,creado_en)
  VALUES('correo_outbox:'||replace(gen_random_uuid()::text,'-',''),m->>'persona_ref',correo,p_reserva->>'desafio_ref',
   decode(p_reserva->>'desafio_hex','hex'),p_reserva->>'clave_ref',(p_reserva->>'vence_utc')::timestamptz,'verificacion',ahora);
  nuevo_estado:='pendiente'; nuevo_activo:=false;
 ELSIF accion='vec.correos.activar' THEN
  IF d.estado<>'verificado' THEN RAISE EXCEPTION 'Usuarios: correo no verificado' USING ERRCODE='P1409'; END IF;
  SELECT correo_ref INTO anterior FROM vec_usuarios.correos_direccion
  WHERE persona_ref=m->>'persona_ref' AND activo AND correo_ref<>correo FOR UPDATE;
  UPDATE vec_usuarios.correos_direccion SET activo=false WHERE persona_ref=m->>'persona_ref' AND activo AND correo_ref<>correo;
  UPDATE vec_usuarios.correos_direccion SET activo=true WHERE persona_ref=m->>'persona_ref' AND correo_ref=correo;
  IF anterior IS NOT NULL THEN
   INSERT INTO vec_usuarios.correos_outbox(outbox_ref,persona_ref,correo_ref,tipo,creado_en)
   VALUES('correo_outbox:'||replace(gen_random_uuid()::text,'-',''),m->>'persona_ref',anterior,'aviso_anterior',ahora);
  END IF;
  nuevo_estado:='verificado'; nuevo_activo:=true;
 ELSIF accion='vec.correos.retirar' THEN
  IF d.estado='retirado' THEN RAISE EXCEPTION 'Usuarios: correo retirado' USING ERRCODE='P1409'; END IF;
  IF d.activo THEN
   anterior:=correo;
   IF m->>'sustituto_ref' !~ '^[A-Za-z0-9:._-]{12,128}$'
   THEN RAISE EXCEPTION 'Usuarios: sustituto explícito requerido' USING ERRCODE='P1409'; END IF;
   SELECT * INTO sustituto FROM vec_usuarios.correos_direccion WHERE persona_ref=m->>'persona_ref'
    AND correo_ref=m->>'sustituto_ref' AND estado='verificado' AND NOT activo FOR UPDATE;
   IF NOT FOUND THEN RAISE EXCEPTION 'Usuarios: sustituto no verificado' USING ERRCODE='P1409'; END IF;
   UPDATE vec_usuarios.correos_direccion SET activo=false,estado='retirado',retirado_en=ahora
    WHERE persona_ref=m->>'persona_ref' AND correo_ref=correo;
   UPDATE vec_usuarios.correos_direccion SET activo=true
    WHERE persona_ref=m->>'persona_ref' AND correo_ref=sustituto.correo_ref;
   INSERT INTO vec_usuarios.correos_outbox(outbox_ref,persona_ref,correo_ref,tipo,creado_en)
   VALUES('correo_outbox:'||replace(gen_random_uuid()::text,'-',''),m->>'persona_ref',correo,'aviso_anterior',ahora);
  ELSE
   IF coalesce(m->>'sustituto_ref','')<>'' THEN RAISE EXCEPTION 'Usuarios: sustituto innecesario' USING ERRCODE='P1409'; END IF;
   UPDATE vec_usuarios.correos_direccion SET estado='retirado',retirado_en=ahora
    WHERE persona_ref=m->>'persona_ref' AND correo_ref=correo;
  END IF;
  UPDATE vec_usuarios.correos_desafio SET estado='sustituido'
   WHERE persona_ref=m->>'persona_ref' AND correo_ref=correo AND estado='pendiente';
  nuevo_estado:='retirado'; nuevo_activo:=false;
 END IF;
 IF c.version IS NOT NULL THEN
  UPDATE vec_usuarios.correos_conjunto SET version=v,actualizado_en=ahora
  WHERE persona_ref=m->>'persona_ref' AND version=v-1;
  GET DIAGNOSTICS n=ROW_COUNT;
  IF n<>1 THEN RAISE EXCEPTION 'Usuarios: versión concurrente' USING ERRCODE='40001'; END IF;
 END IF;
 r:=vec_usuarios.registrar_recibo_correos(m,nuevo_estado,nuevo_activo,correo,anterior,v,ahora);
 PERFORM vec_usuarios.retirar_contexto_correos(m->>'persona_ref','actualizar');
 RETURN r;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.aplicar_correos_propios_v1(text,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios.aplicar_correos_propios_v1(text,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

CREATE FUNCTION vec_usuarios.preparar_verificacion_correo_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; c record; d record; h record;
BEGIN
 m:=vec_usuarios.consumir_contexto_correos(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'verificar');
 IF m->>'accion' IS DISTINCT FROM 'vec.correos.verificar'
 THEN RAISE EXCEPTION 'Usuarios: verificación inválida' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_usuarios:correos:'||(m->>'persona_ref'),0));
 IF EXISTS(SELECT 1 FROM vec_usuarios.correos_recibo WHERE persona_ref=m->>'persona_ref' AND clave_operacion=m->>'clave_operacion')
 THEN RAISE EXCEPTION 'Usuarios: operación registrada, recuperar recibo' USING ERRCODE='40001'; END IF;
 SELECT * INTO c FROM vec_usuarios.correos_conjunto WHERE persona_ref=m->>'persona_ref' FOR UPDATE;
 IF c.version IS NULL OR c.version::text IS DISTINCT FROM m->>'version_esperada'
 THEN RAISE EXCEPTION 'Usuarios: versión correo en conflicto' USING ERRCODE='P1409'; END IF;
 SELECT * INTO d FROM vec_usuarios.correos_direccion WHERE persona_ref=m->>'persona_ref' AND correo_ref=m->>'correo_ref' FOR UPDATE;
 IF NOT FOUND OR d.estado<>'pendiente' OR d.activo
 THEN RAISE EXCEPTION 'Usuarios: correo no pendiente' USING ERRCODE='P1409'; END IF;
 SELECT * INTO h FROM vec_usuarios.correos_desafio WHERE persona_ref=m->>'persona_ref'
  AND correo_ref=m->>'correo_ref' AND estado='pendiente' FOR UPDATE;
 IF NOT FOUND OR h.intentos>=5 OR h.vence_en<=clock_timestamp()
 THEN RAISE EXCEPTION 'Usuarios: desafío agotado o vencido' USING ERRCODE='P1429'; END IF;
 RETURN jsonb_build_object('persona_ref',h.persona_ref,'correo_ref',h.correo_ref,'desafio_ref',h.desafio_ref,
  'huella_codigo_hex',encode(h.huella_codigo,'hex'),'clave_ref',h.clave_ref,'vence_utc',h.vence_en);
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.preparar_verificacion_correo_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios.preparar_verificacion_correo_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

-- El adaptador llama al comprobador Go dentro de la MISMA transacción. La
-- cerradura FOR UPDATE de preparar sigue vigente. No se acepta código en SQL.
CREATE FUNCTION vec_usuarios.cerrar_verificacion_correo_v1(p_persona text,p_clave text,p_valido boolean)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE c record; h record; v bigint; ahora timestamptz(6); n integer; r jsonb;
BEGIN
 IF vec_usuarios.superficie_sesion_correos() IS NULL
 THEN RAISE EXCEPTION 'Usuarios: ejecutor correo denegado' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable' OR p_valido IS NULL
 THEN RAISE EXCEPTION 'Usuarios: cierre de verificación denegado' USING ERRCODE='42501'; END IF;
 SELECT * INTO c FROM vec_usuarios.correos_contexto WHERE xid=pg_current_xact_id_if_assigned()
  AND backend_pid=pg_backend_pid() AND sesion=session_user AND persona_ref=p_persona
  AND modo='verificar' AND accion='vec.correos.verificar' AND clave_operacion=p_clave;
 IF NOT FOUND THEN RAISE EXCEPTION 'Usuarios: verificación sin preparación' USING ERRCODE='42501'; END IF;
 IF c.superficie IS DISTINCT FROM vec_usuarios.superficie_sesion_correos()
 THEN RAISE EXCEPTION 'Usuarios: superficie de verificación divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO h FROM vec_usuarios.correos_desafio WHERE persona_ref=c.persona_ref
  AND correo_ref=c.correo_ref AND estado='pendiente' FOR UPDATE;
 IF NOT FOUND OR h.intentos>=5 OR h.vence_en<=clock_timestamp()
 THEN RAISE EXCEPTION 'Usuarios: desafío agotado o vencido' USING ERRCODE='P1429'; END IF;
 IF NOT p_valido THEN
  UPDATE vec_usuarios.correos_desafio SET intentos=intentos+1,
   estado=CASE WHEN intentos+1>=5 THEN 'agotado' ELSE 'pendiente' END
  WHERE persona_ref=c.persona_ref AND correo_ref=c.correo_ref AND desafio_ref=h.desafio_ref
   AND estado='pendiente' AND intentos<5;
  GET DIAGNOSTICS n=ROW_COUNT;
  IF n<>1 THEN RAISE EXCEPTION 'Usuarios: intento concurrente' USING ERRCODE='40001'; END IF;
  INSERT INTO vec_usuarios.correos_intento_fallido(persona_ref,correo_ref,desafio_ref,numero,
   decision_ref,auditoria_ref,consumo_huella_sha256,ocurrido_en)
  VALUES(c.persona_ref,c.correo_ref,h.desafio_ref,h.intentos+1,c.decision_ref,c.auditoria_ref,
   c.consumo_huella_sha256,date_trunc('microseconds',clock_timestamp()));
  PERFORM vec_usuarios.retirar_contexto_correos(c.persona_ref,'verificar');
  RETURN jsonb_build_object('valido',false);
 END IF;
 SELECT version INTO v FROM vec_usuarios.correos_conjunto WHERE persona_ref=c.persona_ref FOR UPDATE;
 IF v IS DISTINCT FROM c.version_esperada THEN RAISE EXCEPTION 'Usuarios: versión correo en conflicto' USING ERRCODE='P1409'; END IF;
 ahora:=date_trunc('microseconds',clock_timestamp()); v:=v+1;
 UPDATE vec_usuarios.correos_desafio SET estado='usado'
 WHERE persona_ref=c.persona_ref AND correo_ref=c.correo_ref AND desafio_ref=h.desafio_ref AND estado='pendiente';
 UPDATE vec_usuarios.correos_direccion SET estado='verificado',verificado_en=ahora
 WHERE persona_ref=c.persona_ref AND correo_ref=c.correo_ref AND estado='pendiente';
 GET DIAGNOSTICS n=ROW_COUNT;
 IF n<>1 THEN RAISE EXCEPTION 'Usuarios: correo concurrente' USING ERRCODE='40001'; END IF;
 UPDATE vec_usuarios.correos_conjunto SET version=v,actualizado_en=ahora
 WHERE persona_ref=c.persona_ref AND version=v-1;
 GET DIAGNOSTICS n=ROW_COUNT;
 IF n<>1 THEN RAISE EXCEPTION 'Usuarios: versión concurrente' USING ERRCODE='40001'; END IF;
 r:=vec_usuarios.registrar_recibo_correos(
  jsonb_build_object('persona_ref',c.persona_ref,'accion',c.accion),'verificado',false,c.correo_ref,NULL,v,ahora);
 PERFORM vec_usuarios.retirar_contexto_correos(c.persona_ref,'verificar');
 RETURN r;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.cerrar_verificacion_correo_v1(text,text,boolean) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios.cerrar_verificacion_correo_v1(text,text,boolean) TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

CREATE FUNCTION vec_usuarios.consultar_correo_activo_verificado_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; modo text; d record; existe boolean;
BEGIN
 modo:=CASE p_material::jsonb->>'accion'
  WHEN 'vec.correos.activo.ct.consultar' THEN 'activo_ct'
  WHEN 'vec.correos.activo.bolsa.consultar' THEN 'activo_bolsa'
  ELSE NULL END;
 IF modo IS NULL THEN RAISE EXCEPTION 'Usuarios: lectura activa denegada' USING ERRCODE='42501'; END IF;
 m:=vec_usuarios.consumir_contexto_correos(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,modo);
 SELECT d.*,c.version AS conjunto_version INTO d
 FROM vec_usuarios.correos_direccion d JOIN vec_usuarios.correos_conjunto c USING(persona_ref)
 WHERE d.persona_ref=m->>'persona_ref' AND d.activo AND d.estado='verificado';
 existe:=FOUND;
 PERFORM vec_usuarios.retirar_contexto_correos(m->>'persona_ref',modo);
 IF NOT existe THEN RETURN NULL; END IF;
 RETURN jsonb_build_object('persona_ref',d.persona_ref,'correo_ref',d.correo_ref,
  'version',d.version_sobre,'conjunto_version',d.conjunto_version,'clave_ref',d.clave_sobre_ref,
  'nonce_hex',encode(d.nonce,'hex'),'cifrado_hex',encode(d.cifrado,'hex'));
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.consultar_correo_activo_verificado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
-- Sin EXECUTE: CT/Bolsa requieren consumidor técnico interno separado, V3
-- nominal y finalidad propia. El permiso web no habilita esta proyección.

CREATE FUNCTION vec_usuarios.exigir_despachador_correos()
RETURNS void LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF current_user<>'vec_usuarios_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_usuarios_despachador','MEMBER')
    OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND (rolsuper OR rolbypassrls))
    OR NOT EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
       AND m.roleid='vec_usuarios_despachador'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
    OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)<>1
    OR EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member='vec_usuarios_despachador'::regrole)
 THEN RAISE EXCEPTION 'Usuarios: despacho denegado' USING ERRCODE='42501'; END IF;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.exigir_despachador_correos() FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo,vec_usuarios_despachador;

CREATE FUNCTION vec_usuarios.activar_contexto_despacho_correos(p_persona text,p_outbox text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
BEGIN
 PERFORM vec_usuarios.exigir_despachador_correos();
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR p_persona !~ '^per_[A-Za-z0-9_-]{22,128}$' OR p_outbox !~ '^correo_outbox:[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'Usuarios: despacho inválido' USING ERRCODE='42501'; END IF;
 DELETE FROM vec_usuarios.correos_contexto WHERE backend_pid=pg_backend_pid() AND xid<>pg_current_xact_id();
 INSERT INTO vec_usuarios.correos_contexto(xid,backend_pid,sesion,persona_ref,modo,accion,decision_ref,auditoria_ref,consumo_huella_sha256)
 VALUES(pg_current_xact_id(),pg_backend_pid(),session_user,p_persona,'despacho','vec.correos.despachar',
  p_outbox,p_outbox,encode(sha256(convert_to(p_outbox,'UTF8')),'hex'));
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.activar_contexto_despacho_correos(text,text) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo,vec_usuarios_despachador;

-- Un reintento de reserva con la misma clave NO devuelve el desafío otra vez:
-- el resultado externo anterior puede ser desconocido y no autoriza reenviar.
CREATE FUNCTION vec_usuarios.reservar_despacho_correo_v1(p_persona text,p_outbox text,p_reserva text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE o record; ahora timestamptz(6); respuesta jsonb;
BEGIN
 PERFORM vec_usuarios.exigir_despachador_correos();
 IF p_reserva !~ '^[A-Za-z0-9:._-]{16,128}$'
 THEN RAISE EXCEPTION 'Usuarios: reserva inválida' USING ERRCODE='22023'; END IF;
 PERFORM vec_usuarios.activar_contexto_despacho_correos(p_persona,p_outbox);
 SELECT * INTO o FROM vec_usuarios.correos_outbox WHERE persona_ref=p_persona AND outbox_ref=p_outbox FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'Usuarios: salida ausente' USING ERRCODE='P1409'; END IF;
 IF o.estado<>'pendiente' THEN
  IF o.reserva_ref IS DISTINCT FROM p_reserva THEN RAISE EXCEPTION 'Usuarios: salida ya reservada' USING ERRCODE='P1409'; END IF;
  respuesta:=jsonb_build_object('reservado',false,'estado',o.estado,'outbox_ref',o.outbox_ref);
 ELSE
  ahora:=date_trunc('microseconds',clock_timestamp());
  UPDATE vec_usuarios.correos_outbox SET estado='reservado',reserva_ref=p_reserva,reservado_en=ahora
   WHERE persona_ref=p_persona AND outbox_ref=p_outbox AND estado='pendiente';
  INSERT INTO vec_usuarios.correos_despacho_historia(outbox_ref,secuencia,persona_ref,reserva_ref,evento,ocurrido_en)
   VALUES(p_outbox,1,p_persona,p_reserva,'reservado',ahora);
  respuesta:=jsonb_build_object('reservado',true,'estado','reservado','outbox_ref',o.outbox_ref,
   'persona_ref',o.persona_ref,'correo_ref',o.correo_ref,'tipo',o.tipo,'desafio_ref',o.desafio_ref,
   'desafio_hex',CASE WHEN o.desafio IS NULL THEN NULL ELSE encode(o.desafio,'hex') END,
   'clave_ref',o.clave_ref,'vence_utc',o.vence_en);
 END IF;
 PERFORM vec_usuarios.retirar_contexto_correos(p_persona,'despacho');
 RETURN respuesta;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.reservar_despacho_correo_v1(text,text,text) FROM PUBLIC;
-- Bloqueada para todo LOGIN: un rol técnico aislado no sustituye consumo V3
-- nominal y auditoría central de reserva/resultado. Una migración posterior
-- implementará esos contratos antes de conceder EXECUTE.

CREATE FUNCTION vec_usuarios.leer_sobre_despacho_correo_v1(p_persona text,p_outbox text,p_reserva text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE o record; d record;
BEGIN
 PERFORM vec_usuarios.activar_contexto_despacho_correos(p_persona,p_outbox);
 SELECT * INTO o FROM vec_usuarios.correos_outbox WHERE persona_ref=p_persona AND outbox_ref=p_outbox AND estado='reservado' AND reserva_ref=p_reserva;
 IF NOT FOUND THEN RAISE EXCEPTION 'Usuarios: reserva de despacho ausente' USING ERRCODE='42501'; END IF;
 SELECT * INTO d FROM vec_usuarios.correos_direccion WHERE persona_ref=p_persona AND correo_ref=o.correo_ref;
 IF NOT FOUND THEN RAISE EXCEPTION 'Usuarios: dirección de despacho ausente' USING ERRCODE='55000'; END IF;
 PERFORM vec_usuarios.retirar_contexto_correos(p_persona,'despacho');
 RETURN jsonb_build_object('persona_ref',p_persona,'correo_ref',d.correo_ref,'version',d.version_sobre,
  'clave_ref',d.clave_sobre_ref,'nonce_hex',encode(d.nonce,'hex'),'cifrado_hex',encode(d.cifrado,'hex'));
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.leer_sobre_despacho_correo_v1(text,text,text) FROM PUBLIC;

CREATE FUNCTION vec_usuarios.reconciliar_aceptacion_correo_v1(p_persona text,p_outbox text,p_reserva text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE o record; ahora timestamptz(6); respuesta jsonb;
BEGIN
 PERFORM vec_usuarios.activar_contexto_despacho_correos(p_persona,p_outbox);
 SELECT * INTO o FROM vec_usuarios.correos_outbox WHERE persona_ref=p_persona AND outbox_ref=p_outbox FOR UPDATE;
 IF NOT FOUND OR o.reserva_ref IS DISTINCT FROM p_reserva OR o.estado NOT IN ('reservado','aceptado')
 THEN RAISE EXCEPTION 'Usuarios: reconciliación denegada' USING ERRCODE='42501'; END IF;
 IF o.estado='reservado' THEN
  ahora:=date_trunc('microseconds',clock_timestamp());
  UPDATE vec_usuarios.correos_outbox SET estado='aceptado',aceptado_en=ahora
   WHERE persona_ref=p_persona AND outbox_ref=p_outbox AND estado='reservado' AND reserva_ref=p_reserva;
  INSERT INTO vec_usuarios.correos_despacho_historia(outbox_ref,secuencia,persona_ref,reserva_ref,evento,ocurrido_en)
   VALUES(p_outbox,2,p_persona,p_reserva,'smtp_aceptado',ahora);
 ELSE ahora:=o.aceptado_en; END IF;
 respuesta:=jsonb_build_object('outbox_ref',p_outbox,'estado','aceptado','smtp_aceptado_en',ahora);
 PERFORM vec_usuarios.retirar_contexto_correos(p_persona,'despacho');
 RETURN respuesta;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.reconciliar_aceptacion_correo_v1(text,text,text) FROM PUBLIC;
COMMIT;
