\set ON_ERROR_STOP on
-- D20: estado local durable de una subida autorizada por el acto del módulo.
-- No usa D17, Mayan ni inventa una autorización documental autónoma.
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000020',0));
DO $pre$
BEGIN
 IF current_user<>'vec_documentos_propietario' OR
  to_regprocedure('vec_autorizacion_atestada_v3.consumir_subida_local_modulo_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL OR
  to_regclass('vec_documentos.subida_local') IS NOT NULL OR
  to_regrole('vec_documentos_subida_datos') IS NULL OR
  to_regrole('vec_documentos_frontera_externa') IS NULL
 THEN RAISE EXCEPTION 'D20: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
GRANT USAGE ON SCHEMA vec_documentos TO vec_documentos_subida_datos,vec_documentos_frontera_externa;

CREATE TABLE vec_documentos.subida_local(
 carga_id text PRIMARY KEY CHECK (carga_id ~ '^ref:[0-9a-f]{64}$'),
 recurso_ref text NOT NULL,
 persona_ref text NOT NULL,
 perfil_activo_ref text NOT NULL,
 modulo_id text NOT NULL CHECK(modulo_id='meritos'),
 tipo_recurso text NOT NULL CHECK(tipo_recurso='hecho'),
 finalidad text NOT NULL CHECK(finalidad='custodiar_evidencia_meritos'),
 clasificacion text NOT NULL CHECK(clasificacion='confidencial'),
 propietario_version numeric(20,0) NOT NULL CHECK(propietario_version>0),
 clave_idempotencia_hmac text NOT NULL CHECK(clave_idempotencia_hmac ~ '^hmac-sha256:local_v1:[0-9a-f]{64}$'),
 huella_solicitud_hmac text NOT NULL CHECK(huella_solicitud_hmac ~ '^hmac-sha256:[a-z0-9_]{1,64}:[0-9a-f]{64}$'),
 huella_pdf_sha256 text NOT NULL CHECK(huella_pdf_sha256 ~ '^[0-9a-f]{64}$'),
 tamano bigint NOT NULL CHECK(tamano BETWEEN 1 AND 16777216),
 mime text NOT NULL CHECK(mime='application/pdf'),
 correlacion_ref text NOT NULL,
 registro_contexto_ref text NOT NULL,
 huella_contexto_sha256 text NOT NULL CHECK(huella_contexto_sha256 ~ '^[0-9a-f]{64}$'),
 version bigint NOT NULL CHECK(version BETWEEN 1 AND 5),
 estado text NOT NULL CHECK(estado IN ('reservada','recibido','analizando','admitido','rechazado')),
 cuarentena jsonb,
 analisis jsonb,
 admitido jsonb,
 recibo_ref text,
 documento_id text UNIQUE,
 documento_version bigint,
 codigo_rechazo text NOT NULL DEFAULT '',
 registrada_en timestamptz,
 creada_en timestamptz NOT NULL DEFAULT clock_timestamp(),
 actualizada_en timestamptz NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(persona_ref,clave_idempotencia_hmac),
 CHECK ((estado='reservada' AND version=1 AND cuarentena IS NULL AND recibo_ref IS NULL AND analisis IS NULL AND admitido IS NULL AND documento_id IS NULL AND registrada_en IS NULL) OR
        (estado='recibido' AND version=2 AND cuarentena IS NOT NULL AND recibo_ref IS NOT NULL AND analisis IS NULL AND admitido IS NULL AND documento_id IS NULL AND registrada_en IS NULL) OR
        (estado='analizando' AND version IN (3,4) AND cuarentena IS NOT NULL AND recibo_ref IS NOT NULL AND
         (analisis IS NOT NULL)=(version=4) AND admitido IS NULL AND documento_id IS NULL AND registrada_en IS NULL) OR
        (estado='admitido' AND version=5 AND cuarentena IS NOT NULL AND recibo_ref IS NOT NULL AND analisis IS NOT NULL AND admitido IS NOT NULL AND documento_id IS NOT NULL AND documento_version=1 AND codigo_rechazo='' AND registrada_en IS NOT NULL) OR
        (estado='rechazado' AND version=5 AND cuarentena IS NOT NULL AND recibo_ref IS NOT NULL AND analisis IS NOT NULL AND admitido IS NULL AND documento_id IS NULL AND documento_version IS NULL AND codigo_rechazo IN ('pdf_no_valido','contenido_no_permitido','analisis_no_concluyente') AND registrada_en IS NULL))
);
ALTER TABLE vec_documentos.subida_local ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.subida_local FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_subida_local ON vec_documentos.subida_local
 TO vec_documentos_propietario
 USING (current_user='vec_documentos_propietario') WITH CHECK (current_user='vec_documentos_propietario');
REVOKE ALL ON TABLE vec_documentos.subida_local FROM PUBLIC;
CREATE FUNCTION vec_documentos.guardar_transicion_subida_local_v1() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 IF TG_OP<>'UPDATE' OR current_user<>'vec_documentos_propietario' OR
  (to_jsonb(NEW)-ARRAY['version','estado','cuarentena','analisis','admitido','recibo_ref',
   'documento_id','documento_version','codigo_rechazo','registrada_en','actualizada_en']) IS DISTINCT FROM
  (to_jsonb(OLD)-ARRAY['version','estado','cuarentena','analisis','admitido','recibo_ref',
   'documento_id','documento_version','codigo_rechazo','registrada_en','actualizada_en']) OR
  NEW.version<>OLD.version+1 OR
  NOT ((OLD.estado='reservada' AND OLD.version=1 AND NEW.estado='recibido' AND NEW.version=2) OR
       (OLD.estado='recibido' AND OLD.version=2 AND NEW.estado='analizando' AND NEW.version=3) OR
       (OLD.estado='analizando' AND OLD.version=3 AND NEW.estado='analizando' AND NEW.version=4) OR
       (OLD.estado='analizando' AND OLD.version=4 AND NEW.estado IN ('admitido','rechazado') AND NEW.version=5))
 THEN RAISE EXCEPTION 'D20: transición inmutable o fuera de orden' USING ERRCODE='42501'; END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.guardar_transicion_subida_local_v1() FROM PUBLIC;
CREATE TRIGGER subida_local_transicion BEFORE UPDATE OR DELETE ON vec_documentos.subida_local
 FOR EACH ROW EXECUTE FUNCTION vec_documentos.guardar_transicion_subida_local_v1();

CREATE TABLE vec_documentos.subida_local_historia(
 carga_id text NOT NULL REFERENCES vec_documentos.subida_local(carga_id),
 version bigint NOT NULL,
 estado text NOT NULL,
 decision_ref text NOT NULL,
 auditoria_v3_ref text NOT NULL,
 huella_preimagen_sha256 text NOT NULL CHECK(huella_preimagen_sha256 ~ '^[0-9a-f]{64}$'),
 huella_postestado_sha256 text NOT NULL CHECK(huella_postestado_sha256 ~ '^[0-9a-f]{64}$'),
 anterior_sha256 text NOT NULL CHECK(anterior_sha256 ~ '^[0-9a-f]{64}$'),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 evento_en timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(carga_id,version), UNIQUE(decision_ref)
);
ALTER TABLE vec_documentos.subida_local_historia ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.subida_local_historia FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_subida_local_historia ON vec_documentos.subida_local_historia
 TO vec_documentos_propietario
 USING (current_user='vec_documentos_propietario') WITH CHECK (current_user='vec_documentos_propietario');
REVOKE ALL ON TABLE vec_documentos.subida_local_historia FROM PUBLIC;

CREATE TABLE vec_documentos.frontera_subida_local(
 auditoria_ref text PRIMARY KEY,
 correlacion_ref text NOT NULL,
 motivo text NOT NULL CHECK(motivo IN ('autenticacion_requerida','acceso_denegado')),
 ruta text NOT NULL CHECK(ruta IN ('/api/vec/meritos/cartera/comprobantes',
  '/api/vec/meritos/cartera/comprobantes/{comprobante_ref}/original',
  '/api/vec/meritos/cartera/comprobantes/estado')),
 actor_ref text NOT NULL,
 ocurrida_en timestamptz NOT NULL,
 registrada_en timestamptz NOT NULL DEFAULT clock_timestamp(),
 CHECK ((motivo='autenticacion_requerida' AND actor_ref='') OR
  (motivo='acceso_denegado' AND actor_ref ~ '^docactor_[0-9a-f]{64}$'))
);
ALTER TABLE vec_documentos.frontera_subida_local ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.frontera_subida_local FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_frontera_subida_local ON vec_documentos.frontera_subida_local
 TO vec_documentos_propietario
 USING (current_user='vec_documentos_propietario') WITH CHECK (current_user='vec_documentos_propietario');
REVOKE ALL ON TABLE vec_documentos.frontera_subida_local FROM PUBLIC;

CREATE FUNCTION vec_documentos.prohibir_edicion_evidencia_subida_local_v1() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 RAISE EXCEPTION 'D20: evidencia inmutable' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.prohibir_edicion_evidencia_subida_local_v1() FROM PUBLIC;
CREATE TRIGGER historia_subida_local_inmutable BEFORE UPDATE OR DELETE ON vec_documentos.subida_local_historia
 FOR EACH ROW EXECUTE FUNCTION vec_documentos.prohibir_edicion_evidencia_subida_local_v1();
CREATE TRIGGER frontera_subida_local_inmutable BEFORE UPDATE OR DELETE ON vec_documentos.frontera_subida_local
 FOR EACH ROW EXECUTE FUNCTION vec_documentos.prohibir_edicion_evidencia_subida_local_v1();

CREATE FUNCTION vec_documentos.subida_local_login_valido_v1(p_frontera boolean) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT current_user='vec_documentos_propietario' AND session_user=CASE WHEN p_frontera
  THEN 'vec_externo_documentos_frontera' ELSE 'vec_externo_documentos_datos' END AND
  EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit AND
   NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)) AND
  (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1 AND
  EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND
   m.roleid=(CASE WHEN p_frontera THEN 'vec_documentos_frontera_externa' ELSE 'vec_documentos_subida_datos' END)::regrole
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
$f$;
REVOKE ALL ON FUNCTION vec_documentos.subida_local_login_valido_v1(boolean) FROM PUBLIC;

-- Esta proyección nunca recibe una persona ni un identificador del llamador
-- exterior: solo la invocan las fachadas tras consumo V3 y cotejo nominal.
CREATE FUNCTION vec_documentos.instantanea_subida_local_v1(p_carga_id text) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE r vec_documentos.subida_local%ROWTYPE;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR
  vec_documentos.subida_local_login_valido_v1(false) IS NOT TRUE
 THEN RAISE EXCEPTION 'D20: proyección denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT r FROM vec_documentos.subida_local WHERE carga_id=p_carga_id;
 PERFORM vec_documentos.verificar_integridad_subida_local_v1(r.carga_id);
 RETURN jsonb_build_object('resultado',jsonb_build_object('carga_id',r.carga_id,
  'comprobante_ref',coalesce(r.documento_id,''),'documento_version',coalesce(r.documento_version,0),
  'recibo_ref',coalesce(r.recibo_ref,''),'estado',r.estado,'codigo_rechazo',r.codigo_rechazo,
  'registrado_en',r.registrada_en),
  'recurso_ref',r.recurso_ref,'persona_ref',r.persona_ref,'correlacion_ref',r.correlacion_ref,
  'clasificacion',r.clasificacion,'huella_solicitud_hmac',r.huella_solicitud_hmac,
  'huella_pdf_sha256',r.huella_pdf_sha256,'mime',r.mime,'tamano',r.tamano,'version',r.version,
  'cuarentena',r.cuarentena,'analisis',r.analisis,'admitido',r.admitido);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.instantanea_subida_local_v1(text) FROM PUBLIC;

-- Material de decisión y preimagen se cotejan antes de tocar una sola fila.
-- Cada llamada consume una decisión nueva y registra historia dentro de la
-- misma transacción serializable que cambia el agregado.
CREATE FUNCTION vec_documentos.consumir_preimagen_subida_local_v1(
 p_preimagen bytea,p_auth jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE j jsonb; c jsonb; d jsonb; x jsonb; v record; claves text[]; fase text; accion text;
 efecto text; ambito text; pre_sha text; esperado text;
BEGIN
 IF vec_documentos.subida_local_login_valido_v1(false) IS NOT TRUE OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' OR
  octet_length(p_preimagen) NOT BETWEEN 1 AND 65536 OR
  jsonb_typeof(p_auth) IS DISTINCT FROM 'object' OR
  (SELECT count(*) FROM jsonb_object_keys(p_auth))<>7 OR
  p_auth ?& ARRAY['accion','recurso_ref','ambito_ref','principal_id','perfil_activo_ref','finalidad','correlacion_ref'] IS NOT TRUE
 THEN RAISE EXCEPTION 'D20: invocación denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  j:=convert_from(p_preimagen,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation OR character_not_in_repertoire THEN
  RAISE EXCEPTION 'D20: material ilegible' USING ERRCODE='22023'; END;
 IF jsonb_typeof(j) IS DISTINCT FROM 'object' OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR
  jsonb_typeof(d) IS DISTINCT FROM 'object' OR jsonb_typeof(x) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'D20: objeto inválido' USING ERRCODE='22023'; END IF;
 fase:=j->>'fase'; accion:=j->>'accion';
 claves:=CASE fase
  WHEN 'reservar' THEN ARRAY['accion','fase','carga_id','recurso_ref','persona_ref','perfil_activo_ref','modulo_id','tipo_recurso','finalidad','clave_idempotencia_hmac','huella_solicitud_hmac','huella_pdf_sha256','tamano','mime','propietario_version','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'confirmar_recepcion' THEN ARRAY['accion','fase','carga_id','recurso_ref','persona_ref','perfil_activo_ref','finalidad','version_esperada','huella_pdf_sha256','objeto','evidencia_almacen','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'iniciar_analisis' THEN ARRAY['accion','fase','carga_id','recurso_ref','persona_ref','perfil_activo_ref','finalidad','version_esperada','recibo_ref','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'registrar_analisis' THEN ARRAY['accion','fase','carga_id','recurso_ref','persona_ref','perfil_activo_ref','finalidad','version_esperada','analisis','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'cerrar_analisis' THEN ARRAY['accion','fase','carga_id','recurso_ref','persona_ref','perfil_activo_ref','finalidad','version_esperada','estado_final','codigo_rechazo','analisis','promocion','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'consulta_carga' THEN ARRAY['accion','fase','carga_id','documento_id','persona_ref','perfil_activo_ref','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'consulta_documento' THEN ARRAY['accion','fase','carga_id','documento_id','persona_ref','perfil_activo_ref','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'abrir' THEN ARRAY['accion','fase','carga_id','documento_id','persona_ref','perfil_activo_ref','objeto_ref','objeto_version','huella_pdf_sha256','tamano','mime','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  ELSE NULL END;
 IF claves IS NULL OR (SELECT count(*) FROM jsonb_object_keys(j))<>array_length(claves,1) OR
  j ?& claves IS NOT TRUE OR
  ((fase IN ('reservar','confirmar_recepcion','iniciar_analisis','registrar_analisis','cerrar_analisis','consulta_carga'))
    <> (accion='meritos.cartera.comprobante.subir_propio')) OR
  accion NOT IN ('meritos.cartera.comprobante.subir_propio','meritos.cartera.comprobante.descargar_propio') OR
  j->>'persona_ref' !~ '^[A-Za-z0-9:_-]{1,128}$' OR
  j->>'persona_ref' IS DISTINCT FROM x->>'persona_ref' OR
  j->>'perfil_activo_ref' IS DISTINCT FROM x->>'perfil_activo_ref' OR
  j->>'registro_contexto_ref' IS DISTINCT FROM c->>'contexto_ref' OR
  j->>'huella_contexto_sha256' IS DISTINCT FROM c->>'huella_contexto_sha256' OR
  j->>'huella_contexto_sha256' IS DISTINCT FROM encode(sha256(p_contexto),'hex') OR
  j->>'finalidad' IS DISTINCT FROM (CASE WHEN fase IN ('reservar','confirmar_recepcion','iniciar_analisis','registrar_analisis','cerrar_analisis')
   THEN 'custodiar_evidencia_meritos' ELSE j->>'finalidad' END) OR
  p_auth->>'accion' IS DISTINCT FROM accion OR
  p_auth->>'principal_id' IS DISTINCT FROM j->>'persona_ref' OR
  p_auth->>'perfil_activo_ref' IS DISTINCT FROM j->>'perfil_activo_ref' OR
  p_auth->>'finalidad' IS DISTINCT FROM 'custodiar_evidencia_meritos' OR
  p_auth->>'correlacion_ref' IS DISTINCT FROM j->>'correlacion_ref'
 THEN RAISE EXCEPTION 'D20: preimagen divergente' USING ERRCODE='42501'; END IF;
 efecto:=CASE WHEN fase='consulta_documento' THEN j->>'documento_id' ELSE j->>'carga_id' END;
 ambito:=CASE WHEN fase IN ('consulta_carga','consulta_documento') THEN efecto
  WHEN fase='abrir' THEN p_auth->>'ambito_ref' ELSE j->>'recurso_ref' END;
 IF efecto IS NULL OR efecto !~ '^ref:[0-9a-f]{64}$' OR
  p_auth->>'recurso_ref' IS DISTINCT FROM efecto OR
  p_auth->>'ambito_ref' IS DISTINCT FROM ambito OR
  (fase IN ('consulta_carga','consulta_documento') AND p_auth->>'ambito_ref' IS DISTINCT FROM efecto) OR
  c->>'efecto_ref' IS DISTINCT FROM efecto OR d->>'recurso_ref' IS DISTINCT FROM efecto OR
  c->>'operacion' IS DISTINCT FROM accion OR d->>'accion' IS DISTINCT FROM accion OR
  c->>'audiencia_consumo' IS DISTINCT FROM (CASE WHEN accion='meritos.cartera.comprobante.subir_propio'
   THEN 'vec_documentos.subida_local.acto_modulo.v1' ELSE 'vec_documentos.subida_local.descarga_propia.v1' END) OR
  d->>'modulo_id' IS DISTINCT FROM 'meritos' OR d->>'tipo_recurso' IS DISTINCT FROM 'hecho' OR
  d->>'finalidad' IS DISTINCT FROM 'custodiar_evidencia_meritos' OR
  d->'campos_permitidos' IS DISTINCT FROM (CASE WHEN accion='meritos.cartera.comprobante.subir_propio'
   THEN '["contenido","estado","recibo"]'::jsonb ELSE '["contenido"]'::jsonb END) OR
  d->'obligaciones' IS DISTINCT FROM '[]'::jsonb OR
  d#>>'{vinculo_autenticacion_actor,registro_contexto_ref}' IS DISTINCT FROM j->>'registro_contexto_ref' OR
  d#>>'{vinculo_autenticacion_actor,contexto_actor_huella_sha256}' IS DISTINCT FROM j->>'huella_contexto_sha256'
 THEN RAISE EXCEPTION 'D20: decisión divergente' USING ERRCODE='42501'; END IF;
 pre_sha:=encode(sha256(p_preimagen),'hex');
 esperado:=encode(sha256(convert_to(format(
  '{"ambitos":{"persona_ref":"%s"},"atributos":{"preimagen_sha256":"%s"}}',
  j->>'persona_ref',pre_sha),'UTF8')),'hex');
 IF d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM esperado OR
  c->>'huella_efecto_sha256' IS DISTINCT FROM esperado
 THEN RAISE EXCEPTION 'D20: efecto no ligado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_subida_local_modulo_v1(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS DISTINCT FROM TRUE OR v.efecto_ref IS DISTINCT FROM efecto OR
  v.huella_efecto_sha256 IS DISTINCT FROM esperado
 THEN RAISE EXCEPTION 'D20: consumo inválido' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('preimagen',j,'decision_ref',v.decision_ref,'auditoria_ref',v.auditoria_ref,
  'preimagen_sha256',pre_sha,'persona_version',p_persona_version);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.consumir_preimagen_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE TABLE vec_documentos.subida_local_lectura(
 decision_ref text PRIMARY KEY,
 carga_id text NOT NULL REFERENCES vec_documentos.subida_local(carga_id),
 fase text NOT NULL CHECK(fase IN ('consulta_carga','consulta_documento','abrir')),
 persona_ref text NOT NULL,
 auditoria_v3_ref text NOT NULL,
 huella_preimagen_sha256 text NOT NULL CHECK(huella_preimagen_sha256 ~ '^[0-9a-f]{64}$'),
 leida_en timestamptz NOT NULL DEFAULT clock_timestamp()
);
ALTER TABLE vec_documentos.subida_local_lectura ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.subida_local_lectura FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_subida_local_lectura ON vec_documentos.subida_local_lectura
 TO vec_documentos_propietario
 USING (current_user='vec_documentos_propietario') WITH CHECK (current_user='vec_documentos_propietario');
REVOKE ALL ON TABLE vec_documentos.subida_local_lectura FROM PUBLIC;
CREATE TRIGGER lectura_subida_local_inmutable BEFORE UPDATE OR DELETE ON vec_documentos.subida_local_lectura
 FOR EACH ROW EXECUTE FUNCTION vec_documentos.prohibir_edicion_evidencia_subida_local_v1();

CREATE TABLE vec_documentos.subida_local_outbox(
 carga_id text NOT NULL REFERENCES vec_documentos.subida_local(carga_id),
 version bigint NOT NULL,
 estado text NOT NULL,
 huella_postestado_sha256 text NOT NULL CHECK(huella_postestado_sha256 ~ '^[0-9a-f]{64}$'),
 creada_en timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(carga_id,version)
);
ALTER TABLE vec_documentos.subida_local_outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.subida_local_outbox FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_subida_local_outbox ON vec_documentos.subida_local_outbox
 TO vec_documentos_propietario
 USING (current_user='vec_documentos_propietario') WITH CHECK (current_user='vec_documentos_propietario');
REVOKE ALL ON TABLE vec_documentos.subida_local_outbox FROM PUBLIC;
CREATE TRIGGER outbox_subida_local_inmutable BEFORE UPDATE OR DELETE ON vec_documentos.subida_local_outbox
 FOR EACH ROW EXECUTE FUNCTION vec_documentos.prohibir_edicion_evidencia_subida_local_v1();

-- La postimagen se fija después de cada INSERT/UPDATE, incluidos sus tiempos,
-- objeto y recibo. Solo las fachadas con V3 consumida invocan esta función
-- privada del propietario antes de entregar una lectura o repetición.
CREATE FUNCTION vec_documentos.verificar_integridad_subida_local_v1(p_carga_id text) RETURNS void
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE r vec_documentos.subida_local%ROWTYPE; e record; siguiente bigint:=1;
 anterior text:=repeat('0',64); postestado text; huella text;
BEGIN
 IF current_user<>'vec_documentos_propietario'
 THEN RAISE EXCEPTION 'D20: integridad denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT r FROM vec_documentos.subida_local WHERE carga_id=p_carga_id;
 postestado:=encode(sha256(convert_to(to_jsonb(r)::text,'UTF8')),'hex');
 FOR e IN
  SELECT h.version,h.estado,h.decision_ref,h.auditoria_v3_ref,h.huella_preimagen_sha256,
   h.huella_postestado_sha256,h.anterior_sha256,h.huella_sha256,
   o.version AS outbox_version,o.estado AS outbox_estado,
   o.huella_postestado_sha256 AS outbox_postestado_sha256
  FROM vec_documentos.subida_local_historia h
  FULL JOIN vec_documentos.subida_local_outbox o
   ON (o.carga_id,o.version)=(h.carga_id,h.version)
  WHERE coalesce(h.carga_id,o.carga_id)=p_carga_id
  ORDER BY coalesce(h.version,o.version)
 LOOP
  huella:=encode(sha256(convert_to(jsonb_build_object('carga_id',p_carga_id,
   'version',e.version,'estado',e.estado,'decision_ref',e.decision_ref,
   'auditoria_ref',e.auditoria_v3_ref,'preimagen_sha256',e.huella_preimagen_sha256,
   'postestado_sha256',e.huella_postestado_sha256,
   'anterior_sha256',e.anterior_sha256)::text,'UTF8')),'hex');
  IF e.version IS DISTINCT FROM siguiente OR e.outbox_version IS DISTINCT FROM siguiente OR
   e.anterior_sha256 IS DISTINCT FROM anterior OR e.huella_sha256 IS DISTINCT FROM huella OR
   e.outbox_estado IS DISTINCT FROM e.estado OR
   e.outbox_postestado_sha256 IS DISTINCT FROM e.huella_postestado_sha256 OR
   (siguiente=r.version AND (e.estado IS DISTINCT FROM r.estado OR
    e.huella_postestado_sha256 IS DISTINCT FROM postestado))
  THEN RAISE EXCEPTION 'D20: historia o postestado divergente' USING ERRCODE='42501'; END IF;
  anterior:=e.huella_sha256; siguiente:=siguiente+1;
 END LOOP;
 IF siguiente IS DISTINCT FROM r.version+1
 THEN RAISE EXCEPTION 'D20: historia incompleta' USING ERRCODE='42501'; END IF;
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.verificar_integridad_subida_local_v1(text) FROM PUBLIC;

CREATE FUNCTION vec_documentos.registrar_cambio_subida_local_v1(p_carga_id text,p_version bigint,p_estado text,
 p_decision_ref text,p_auditoria_ref text,p_preimagen_sha256 text) RETURNS void
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE anterior text; huella text; postestado text; r vec_documentos.subida_local%ROWTYPE;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR vec_documentos.subida_local_login_valido_v1(false) IS NOT TRUE OR
  p_version NOT BETWEEN 1 AND 5 OR p_preimagen_sha256 !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'D20: historia denegada' USING ERRCODE='42501'; END IF;
 IF p_version=1 THEN anterior:=repeat('0',64);
 ELSE SELECT h.huella_sha256 INTO STRICT anterior FROM vec_documentos.subida_local_historia h
  WHERE h.carga_id=p_carga_id AND h.version=p_version-1 FOR SHARE; END IF;
 SELECT * INTO STRICT r FROM vec_documentos.subida_local WHERE carga_id=p_carga_id FOR SHARE;
 IF r.version IS DISTINCT FROM p_version OR r.estado IS DISTINCT FROM p_estado
 THEN RAISE EXCEPTION 'D20: postestado no corresponde' USING ERRCODE='42501'; END IF;
 postestado:=encode(sha256(convert_to(to_jsonb(r)::text,'UTF8')),'hex');
 huella:=encode(sha256(convert_to(jsonb_build_object('carga_id',p_carga_id,'version',p_version,
  'estado',p_estado,'decision_ref',p_decision_ref,'auditoria_ref',p_auditoria_ref,
  'preimagen_sha256',p_preimagen_sha256,'postestado_sha256',postestado,
  'anterior_sha256',anterior)::text,'UTF8')),'hex');
 INSERT INTO vec_documentos.subida_local_historia(carga_id,version,estado,decision_ref,auditoria_v3_ref,
  huella_preimagen_sha256,huella_postestado_sha256,anterior_sha256,huella_sha256)
 VALUES(p_carga_id,p_version,p_estado,p_decision_ref,p_auditoria_ref,p_preimagen_sha256,postestado,anterior,huella);
 INSERT INTO vec_documentos.subida_local_outbox(carga_id,version,estado,huella_postestado_sha256)
 VALUES(p_carga_id,p_version,p_estado,postestado);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.registrar_cambio_subida_local_v1(text,bigint,text,text,text,text) FROM PUBLIC;

CREATE FUNCTION vec_documentos.operar_subida_local_interna_v1(
 p_preimagen bytea,p_auth jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE material jsonb; j jsonb; r vec_documentos.subida_local%ROWTYPE; fase text;
 objeto jsonb; evidencia jsonb; v_analisis jsonb; v_promocion jsonb; huella text;
 documento text; recibo text; codigo text; estado_final text;
BEGIN
 material:=vec_documentos.consumir_preimagen_subida_local_v1(p_preimagen,p_auth,p_capacidad,p_decision,
  p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 j:=material->'preimagen'; fase:=j->>'fase';
 IF fase='reservar' THEN
  IF j->>'modulo_id' IS DISTINCT FROM 'meritos' OR j->>'tipo_recurso' IS DISTINCT FROM 'hecho' OR
   j->>'finalidad' IS DISTINCT FROM 'custodiar_evidencia_meritos' OR
   j->>'recurso_ref' !~ '^ref:[0-9a-f]{64}$' OR
   j->>'clave_idempotencia_hmac' !~ '^hmac-sha256:local_v1:[0-9a-f]{64}$' OR
   j->>'huella_solicitud_hmac' !~ '^hmac-sha256:[a-z0-9_]{1,64}:[0-9a-f]{64}$' OR
   j->>'huella_pdf_sha256' !~ '^[0-9a-f]{64}$' OR
   j->>'mime' IS DISTINCT FROM 'application/pdf' OR
   j->>'tamano' !~ '^[1-9][0-9]{0,7}$' OR
   (j->>'tamano')::bigint>16777216 OR
   j->>'propietario_version' !~ '^[1-9][0-9]{0,18}$' OR
   (j->>'propietario_version')::numeric IS DISTINCT FROM p_persona_version OR
   j->>'persona_ref' IS DISTINCT FROM (convert_from(p_contexto,'UTF8')::jsonb)->>'persona_ref'
  THEN RAISE EXCEPTION 'D20: reserva inválida' USING ERRCODE='42501'; END IF;
  PERFORM pg_advisory_xact_lock(hashtextextended('vec_documentos:subida_local:'||(j->>'persona_ref')||':'||(j->>'clave_idempotencia_hmac'),0));
  SELECT * INTO r FROM vec_documentos.subida_local WHERE
   persona_ref=j->>'persona_ref' AND clave_idempotencia_hmac=j->>'clave_idempotencia_hmac' FOR UPDATE;
  IF FOUND THEN
   IF r.carga_id IS DISTINCT FROM j->>'carga_id' OR r.recurso_ref IS DISTINCT FROM j->>'recurso_ref' OR
    r.perfil_activo_ref IS DISTINCT FROM j->>'perfil_activo_ref' OR
    r.huella_solicitud_hmac IS DISTINCT FROM j->>'huella_solicitud_hmac' OR
    r.huella_pdf_sha256 IS DISTINCT FROM j->>'huella_pdf_sha256' OR
    r.tamano IS DISTINCT FROM (j->>'tamano')::bigint
   THEN RAISE EXCEPTION 'D20: clave usada para otra carga' USING ERRCODE='23505'; END IF;
   RETURN vec_documentos.instantanea_subida_local_v1(r.carga_id);
  END IF;
  INSERT INTO vec_documentos.subida_local(carga_id,recurso_ref,persona_ref,perfil_activo_ref,
   modulo_id,tipo_recurso,finalidad,clasificacion,propietario_version,clave_idempotencia_hmac,
   huella_solicitud_hmac,huella_pdf_sha256,tamano,mime,correlacion_ref,registro_contexto_ref,
   huella_contexto_sha256,version,estado)
  VALUES(j->>'carga_id',j->>'recurso_ref',j->>'persona_ref',j->>'perfil_activo_ref',
   'meritos','hecho','custodiar_evidencia_meritos','confidencial',(j->>'propietario_version')::numeric,
   j->>'clave_idempotencia_hmac',j->>'huella_solicitud_hmac',j->>'huella_pdf_sha256',
   (j->>'tamano')::bigint,'application/pdf',j->>'correlacion_ref',j->>'registro_contexto_ref',
   j->>'huella_contexto_sha256',1,'reservada');
  PERFORM vec_documentos.registrar_cambio_subida_local_v1(j->>'carga_id',1,'reservada',
   material->>'decision_ref',material->>'auditoria_ref',material->>'preimagen_sha256');
  RETURN vec_documentos.instantanea_subida_local_v1(j->>'carga_id');
 END IF;
 IF fase='consulta_documento' THEN
  IF j->>'carga_id' IS DISTINCT FROM '' OR j->>'documento_id' !~ '^ref:[0-9a-f]{64}$'
  THEN RAISE EXCEPTION 'D20: consulta documental inválida' USING ERRCODE='42501'; END IF;
  SELECT * INTO r FROM vec_documentos.subida_local WHERE documento_id=j->>'documento_id' FOR SHARE;
 ELSE
  IF j->>'carga_id' !~ '^ref:[0-9a-f]{64}$' THEN RAISE EXCEPTION 'D20: carga inválida' USING ERRCODE='42501'; END IF;
  SELECT * INTO r FROM vec_documentos.subida_local WHERE carga_id=j->>'carga_id'
   FOR UPDATE;
 END IF;
 IF NOT FOUND OR r.persona_ref IS DISTINCT FROM j->>'persona_ref' OR
  (fase IN ('confirmar_recepcion','iniciar_analisis','registrar_analisis','cerrar_analisis') AND
   (r.recurso_ref IS DISTINCT FROM j->>'recurso_ref' OR r.finalidad IS DISTINCT FROM j->>'finalidad' OR
    r.correlacion_ref IS DISTINCT FROM j->>'correlacion_ref')) OR
  (fase='abrir' AND (r.documento_id IS DISTINCT FROM j->>'documento_id' OR
   p_auth->>'ambito_ref' IS DISTINCT FROM r.recurso_ref OR
   j->>'objeto_ref' IS DISTINCT FROM r.admitido#>>'{Objeto,Objeto,Referencia}' OR
   j->>'objeto_version' IS DISTINCT FROM r.admitido#>>'{Objeto,Objeto,Version}' OR
   j->>'huella_pdf_sha256' IS DISTINCT FROM r.admitido#>>'{Objeto,HuellaSHA256}' OR
   j->>'tamano' IS DISTINCT FROM r.admitido#>>'{Objeto,Tamano}' OR
   j->>'mime' IS DISTINCT FROM r.admitido#>>'{Objeto,MIME}')) OR
  (fase='consulta_carga' AND j->>'documento_id' IS DISTINCT FROM '') OR
  (fase='consulta_documento' AND r.estado IS DISTINCT FROM 'admitido')
 THEN RAISE EXCEPTION 'D20: carga propia ausente' USING ERRCODE='42501'; END IF;
 PERFORM vec_documentos.verificar_integridad_subida_local_v1(r.carga_id);
 IF fase IN ('consulta_carga','consulta_documento','abrir') THEN
  IF fase='abrir' AND (r.estado IS DISTINCT FROM 'admitido' OR r.admitido IS NULL OR
   r.analisis->>'Estado' IS DISTINCT FROM 'limpio')
  THEN RAISE EXCEPTION 'D20: original no admitido' USING ERRCODE='42501'; END IF;
  INSERT INTO vec_documentos.subida_local_lectura(decision_ref,carga_id,fase,persona_ref,auditoria_v3_ref,huella_preimagen_sha256)
  VALUES(material->>'decision_ref',r.carga_id,fase,r.persona_ref,material->>'auditoria_ref',material->>'preimagen_sha256');
  RETURN vec_documentos.instantanea_subida_local_v1(r.carga_id);
 END IF;
 IF j->>'version_esperada' !~ '^[1-4]$' OR r.version<>(j->>'version_esperada')::bigint OR
  r.propietario_version>p_persona_version OR
  r.estado IS DISTINCT FROM (CASE fase WHEN 'confirmar_recepcion' THEN 'reservada'
   WHEN 'iniciar_analisis' THEN 'recibido' ELSE 'analizando' END)
 THEN RAISE EXCEPTION 'D20: transición obsoleta' USING ERRCODE='40001'; END IF;
 IF fase='confirmar_recepcion' THEN
  objeto:=j->'objeto'; evidencia:=j->'evidencia_almacen';
  IF jsonb_typeof(objeto)<>'object' OR jsonb_typeof(evidencia)<>'object' OR
   j->>'huella_pdf_sha256' IS DISTINCT FROM r.huella_pdf_sha256 OR
   objeto->>'Zona' IS DISTINCT FROM 'cuarentena' OR objeto->>'MIME' IS DISTINCT FROM r.mime OR
   objeto->>'HuellaSHA256' IS DISTINCT FROM r.huella_pdf_sha256 OR
   objeto->>'Tamano' IS DISTINCT FROM r.tamano::text OR
   objeto#>>'{Objeto,Referencia}' !~ '^obj_[0-9a-f]{32}$' OR
   objeto#>>'{Objeto,Version}' IS DISTINCT FROM '1' OR
   evidencia->>'Accion' IS DISTINCT FROM 'escribir' OR
   evidencia->>'AccionNegocio' IS DISTINCT FROM j->>'accion' OR
   evidencia->>'CargaRef' IS DISTINCT FROM r.carga_id OR
   evidencia->>'RecursoRef' IS DISTINCT FROM r.carga_id OR
   evidencia->>'ModuloID' IS DISTINCT FROM r.modulo_id OR
   evidencia->>'Finalidad' IS DISTINCT FROM r.finalidad OR
   evidencia->>'CorrelacionRef' IS DISTINCT FROM r.correlacion_ref OR
   evidencia->>'Referencia' IS DISTINCT FROM objeto->>'EvidenciaCreacionRef' OR
   evidencia->>'ConectorID' IS DISTINCT FROM objeto->>'ConectorID' OR
   evidencia#>>'{Objeto,Referencia}' IS DISTINCT FROM objeto#>>'{Objeto,Referencia}' OR
   evidencia#>>'{Objeto,Version}' IS DISTINCT FROM objeto#>>'{Objeto,Version}'
  THEN RAISE EXCEPTION 'D20: recibo de cuarentena divergente' USING ERRCODE='42501'; END IF;
  recibo:='ref:'||encode(sha256(convert_to('vec.documentos.subida_local.recibo.v1:'||r.carga_id||':'||
   (objeto#>>'{Objeto,Referencia}')||':'||(objeto#>>'{Objeto,Version}')||':'||r.huella_pdf_sha256,'UTF8')),'hex');
  UPDATE vec_documentos.subida_local SET estado='recibido',version=2,cuarentena=jsonb_build_object(
   'Objeto',objeto,'Evidencia',evidencia),recibo_ref=recibo,actualizada_en=clock_timestamp() WHERE carga_id=r.carga_id;
  PERFORM vec_documentos.registrar_cambio_subida_local_v1(r.carga_id,2,'recibido',
   material->>'decision_ref',material->>'auditoria_ref',material->>'preimagen_sha256');
 ELSIF fase='iniciar_analisis' THEN
  IF j->>'recibo_ref' IS DISTINCT FROM r.recibo_ref
  THEN RAISE EXCEPTION 'D20: recibo divergente' USING ERRCODE='42501'; END IF;
  UPDATE vec_documentos.subida_local SET estado='analizando',version=3,actualizada_en=clock_timestamp()
   WHERE carga_id=r.carga_id;
  PERFORM vec_documentos.registrar_cambio_subida_local_v1(r.carga_id,3,'analizando',
   material->>'decision_ref',material->>'auditoria_ref',material->>'preimagen_sha256');
 ELSIF fase IN ('registrar_analisis','cerrar_analisis') THEN
  v_analisis:=j->'analisis'; v_promocion:=j->'promocion'; codigo:=j->>'codigo_rechazo';
  estado_final:=j->>'estado_final';
  IF jsonb_typeof(v_analisis)<>'object' OR
   v_analisis#>>'{Objeto,Referencia}' IS DISTINCT FROM r.cuarentena#>>'{Objeto,Objeto,Referencia}' OR
   v_analisis#>>'{Objeto,Version}' IS DISTINCT FROM r.cuarentena#>>'{Objeto,Objeto,Version}' OR
   v_analisis->>'HuellaObjetoSHA256' IS DISTINCT FROM r.huella_pdf_sha256 OR
   v_analisis->>'TamanoObjeto' IS DISTINCT FROM r.tamano::text OR
   v_analisis->>'MIMEDeclarado' IS DISTINCT FROM r.mime OR
   v_analisis->>'CorrelacionRef' IS DISTINCT FROM r.correlacion_ref OR
   v_analisis->>'Finalidad' IS DISTINCT FROM r.finalidad OR
   nullif(v_analisis->>'EvidenciaRef','') IS NULL
  THEN RAISE EXCEPTION 'D20: análisis sin vínculo' USING ERRCODE='42501'; END IF;
  IF fase='registrar_analisis' THEN
   IF r.version<>3 OR r.analisis IS NOT NULL OR
    (v_analisis->>'Estado'=ANY(ARRAY['limpio','malicioso','sospechoso','no_concluyente','error'])) IS NOT TRUE
   THEN RAISE EXCEPTION 'D20: registro de análisis inválido' USING ERRCODE='42501'; END IF;
   UPDATE vec_documentos.subida_local SET analisis=v_analisis,version=4,actualizada_en=clock_timestamp()
    WHERE carga_id=r.carga_id;
   PERFORM vec_documentos.registrar_cambio_subida_local_v1(r.carga_id,4,'analizando',
    material->>'decision_ref',material->>'auditoria_ref',material->>'preimagen_sha256');
   RETURN vec_documentos.instantanea_subida_local_v1(r.carga_id);
  END IF;
  IF r.version<>4 OR v_analisis IS DISTINCT FROM r.analisis OR
   ((estado_final='admitido' AND (codigo IS DISTINCT FROM '' OR
    v_analisis->>'Estado' IS DISTINCT FROM 'limpio' OR
    v_analisis->>'MIMEDetectado' IS DISTINCT FROM r.mime OR
    jsonb_typeof(v_promocion) IS DISTINCT FROM 'object')) OR
    (estado_final='rechazado' AND (codigo NOT IN ('pdf_no_valido','contenido_no_permitido','analisis_no_concluyente') OR
     jsonb_typeof(v_promocion) IS DISTINCT FROM 'null' OR
     (v_analisis->>'Estado'=ANY(ARRAY['malicioso','sospechoso','no_concluyente','error'])) IS NOT TRUE OR
     (codigo='pdf_no_valido' AND v_analisis->>'CodigoResultado' IS DISTINCT FROM 'PDF_INVALIDO') OR
     (v_analisis->>'Estado' IN ('malicioso','sospechoso') AND
       codigo NOT IN ('contenido_no_permitido','pdf_no_valido')) OR
     (v_analisis->>'Estado' IN ('no_concluyente','error') AND
       codigo NOT IN ('analisis_no_concluyente','pdf_no_valido')))) OR
    estado_final NOT IN ('admitido','rechazado'))
  THEN RAISE EXCEPTION 'D20: análisis divergente' USING ERRCODE='42501'; END IF;
  IF estado_final='admitido' THEN
   objeto:=v_promocion->'Objeto'; evidencia:=v_promocion->'Evidencia';
   IF objeto->>'Zona' IS DISTINCT FROM 'admitida' OR objeto->>'MIME' IS DISTINCT FROM r.mime OR
    objeto->>'HuellaSHA256' IS DISTINCT FROM r.huella_pdf_sha256 OR
    objeto->>'Tamano' IS DISTINCT FROM r.tamano::text OR
    objeto#>>'{Objeto,Referencia}' IS NOT DISTINCT FROM r.cuarentena#>>'{Objeto,Objeto,Referencia}' OR
    objeto#>>'{Objeto,Referencia}' !~ '^obj_[0-9a-f]{32}$' OR
    objeto#>>'{Objeto,Version}' IS DISTINCT FROM '1' OR
    evidencia->>'Accion' IS DISTINCT FROM 'promover' OR
    evidencia->>'AccionNegocio' IS DISTINCT FROM j->>'accion' OR
    evidencia->>'CargaRef' IS DISTINCT FROM r.carga_id OR
    evidencia->>'RecursoRef' IS DISTINCT FROM r.carga_id OR
    evidencia->>'ModuloID' IS DISTINCT FROM r.modulo_id OR
    evidencia->>'Finalidad' IS DISTINCT FROM r.finalidad OR
    evidencia->>'FundamentoRef' IS DISTINCT FROM v_analisis->>'EvidenciaRef' OR
    evidencia->>'CorrelacionRef' IS DISTINCT FROM r.correlacion_ref OR
    evidencia->>'Referencia' IS DISTINCT FROM objeto->>'EvidenciaCreacionRef' OR
    evidencia->>'ConectorID' IS DISTINCT FROM objeto->>'ConectorID' OR
    evidencia#>>'{Objeto,Referencia}' IS DISTINCT FROM objeto#>>'{Objeto,Referencia}' OR
    evidencia#>>'{Objeto,Version}' IS DISTINCT FROM objeto#>>'{Objeto,Version}'
   THEN RAISE EXCEPTION 'D20: promoción divergente' USING ERRCODE='42501'; END IF;
   documento:='ref:'||encode(sha256(convert_to('vec.documentos.subida_local.comprobante.v1:'||r.carga_id,'UTF8')),'hex');
   UPDATE vec_documentos.subida_local SET estado='admitido',version=5,analisis=v_analisis,
    admitido=v_promocion,documento_id=documento,documento_version=1,codigo_rechazo='',
    registrada_en=clock_timestamp(),actualizada_en=clock_timestamp() WHERE carga_id=r.carga_id;
  ELSE
   UPDATE vec_documentos.subida_local SET estado='rechazado',version=5,analisis=v_analisis,
    codigo_rechazo=codigo,actualizada_en=clock_timestamp() WHERE carga_id=r.carga_id;
  END IF;
  PERFORM vec_documentos.registrar_cambio_subida_local_v1(r.carga_id,5,estado_final,
   material->>'decision_ref',material->>'auditoria_ref',material->>'preimagen_sha256');
 ELSE RAISE EXCEPTION 'D20: fase desconocida' USING ERRCODE='22023'; END IF;
 RETURN vec_documentos.instantanea_subida_local_v1(r.carga_id);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.operar_subida_local_interna_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE FUNCTION vec_documentos.reservar_subida_local_v1(
 p_preimagen bytea,p_auth jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE j jsonb;
BEGIN
 IF vec_documentos.subida_local_login_valido_v1(false) IS NOT TRUE
 THEN RAISE EXCEPTION 'D20: fachada denegada' USING ERRCODE='42501'; END IF;
 BEGIN j:=convert_from(p_preimagen,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'D20: preimagen ilegible' USING ERRCODE='22023'; END;
 IF j->>'fase' NOT IN ('reservar')
 THEN RAISE EXCEPTION 'D20: fase denegada' USING ERRCODE='42501'; END IF;
 RETURN vec_documentos.operar_subida_local_interna_v1(
  p_preimagen,p_auth,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.reservar_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.reservar_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_subida_datos;

CREATE FUNCTION vec_documentos.confirmar_recepcion_subida_local_v1(
 p_preimagen bytea,p_auth jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE j jsonb;
BEGIN
 IF vec_documentos.subida_local_login_valido_v1(false) IS NOT TRUE
 THEN RAISE EXCEPTION 'D20: fachada denegada' USING ERRCODE='42501'; END IF;
 BEGIN j:=convert_from(p_preimagen,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'D20: preimagen ilegible' USING ERRCODE='22023'; END;
 IF j->>'fase' NOT IN ('confirmar_recepcion')
 THEN RAISE EXCEPTION 'D20: fase denegada' USING ERRCODE='42501'; END IF;
 RETURN vec_documentos.operar_subida_local_interna_v1(
  p_preimagen,p_auth,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.confirmar_recepcion_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.confirmar_recepcion_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_subida_datos;

CREATE FUNCTION vec_documentos.iniciar_analisis_subida_local_v1(
 p_preimagen bytea,p_auth jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE j jsonb;
BEGIN
 IF vec_documentos.subida_local_login_valido_v1(false) IS NOT TRUE
 THEN RAISE EXCEPTION 'D20: fachada denegada' USING ERRCODE='42501'; END IF;
 BEGIN j:=convert_from(p_preimagen,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'D20: preimagen ilegible' USING ERRCODE='22023'; END;
 IF j->>'fase' NOT IN ('iniciar_analisis')
 THEN RAISE EXCEPTION 'D20: fase denegada' USING ERRCODE='42501'; END IF;
 RETURN vec_documentos.operar_subida_local_interna_v1(
  p_preimagen,p_auth,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.iniciar_analisis_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.iniciar_analisis_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_subida_datos;

CREATE FUNCTION vec_documentos.registrar_analisis_subida_local_v1(
 p_preimagen bytea,p_auth jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE j jsonb;
BEGIN
 IF vec_documentos.subida_local_login_valido_v1(false) IS NOT TRUE
 THEN RAISE EXCEPTION 'D20: fachada denegada' USING ERRCODE='42501'; END IF;
 BEGIN j:=convert_from(p_preimagen,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'D20: preimagen ilegible' USING ERRCODE='22023'; END;
 IF j->>'fase' IS DISTINCT FROM 'registrar_analisis'
 THEN RAISE EXCEPTION 'D20: fase denegada' USING ERRCODE='42501'; END IF;
 RETURN vec_documentos.operar_subida_local_interna_v1(
  p_preimagen,p_auth,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.registrar_analisis_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.registrar_analisis_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_subida_datos;

CREATE FUNCTION vec_documentos.cerrar_analisis_subida_local_v1(
 p_preimagen bytea,p_auth jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE j jsonb;
BEGIN
 IF vec_documentos.subida_local_login_valido_v1(false) IS NOT TRUE
 THEN RAISE EXCEPTION 'D20: fachada denegada' USING ERRCODE='42501'; END IF;
 BEGIN j:=convert_from(p_preimagen,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'D20: preimagen ilegible' USING ERRCODE='22023'; END;
 IF j->>'fase' NOT IN ('cerrar_analisis')
 THEN RAISE EXCEPTION 'D20: fase denegada' USING ERRCODE='42501'; END IF;
 RETURN vec_documentos.operar_subida_local_interna_v1(
  p_preimagen,p_auth,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.cerrar_analisis_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.cerrar_analisis_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_subida_datos;

CREATE FUNCTION vec_documentos.consultar_subida_local_propia_v1(
 p_preimagen bytea,p_auth jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE j jsonb;
BEGIN
 IF vec_documentos.subida_local_login_valido_v1(false) IS NOT TRUE
 THEN RAISE EXCEPTION 'D20: fachada denegada' USING ERRCODE='42501'; END IF;
 BEGIN j:=convert_from(p_preimagen,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'D20: preimagen ilegible' USING ERRCODE='22023'; END;
 IF j->>'fase' NOT IN ('consulta_carga','consulta_documento','abrir')
 THEN RAISE EXCEPTION 'D20: fase denegada' USING ERRCODE='42501'; END IF;
 RETURN vec_documentos.operar_subida_local_interna_v1(
  p_preimagen,p_auth,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.consultar_subida_local_propia_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.consultar_subida_local_propia_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_subida_datos;

CREATE FUNCTION vec_documentos.registrar_denegacion_frontera_subida_local_v1(
 p_correlacion text,p_motivo text,p_ruta text,p_actor_ref text,p_ocurrida_en timestamptz) RETURNS text
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE referencia text;
BEGIN
 IF vec_documentos.subida_local_login_valido_v1(true) IS NOT TRUE OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' OR
  p_correlacion !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' OR
  p_motivo NOT IN ('autenticacion_requerida','acceso_denegado') OR
  p_ruta NOT IN ('/api/vec/meritos/cartera/comprobantes',
   '/api/vec/meritos/cartera/comprobantes/{comprobante_ref}/original',
   '/api/vec/meritos/cartera/comprobantes/estado') OR
  ((p_motivo='autenticacion_requerida' AND p_actor_ref<>'') OR
   (p_motivo='acceso_denegado' AND p_actor_ref !~ '^docactor_[0-9a-f]{64}$')) OR
  p_ocurrida_en IS NULL OR p_ocurrida_en>clock_timestamp()+interval '5 seconds' OR
  p_ocurrida_en<clock_timestamp()-interval '5 minutes'
 THEN RAISE EXCEPTION 'D20: denegación de frontera inválida' USING ERRCODE='42501'; END IF;
 referencia:='ref:'||encode(sha256(convert_to('vec.documentos.frontera_subida_local.v1:'||p_correlacion||':'||
  p_motivo||':'||p_ruta||':'||p_actor_ref||':'||to_char(p_ocurrida_en AT TIME ZONE 'UTC',
  'YYYY-MM-DD"T"HH24:MI:SS.US'), 'UTF8')),'hex');
 INSERT INTO vec_documentos.frontera_subida_local(auditoria_ref,correlacion_ref,motivo,ruta,actor_ref,ocurrida_en)
 VALUES(referencia,p_correlacion,p_motivo,p_ruta,p_actor_ref,p_ocurrida_en);
 RETURN referencia;
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.registrar_denegacion_frontera_subida_local_v1(text,text,text,text,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.registrar_denegacion_frontera_subida_local_v1(text,text,text,text,timestamptz) TO vec_documentos_frontera_externa;
COMMIT;
