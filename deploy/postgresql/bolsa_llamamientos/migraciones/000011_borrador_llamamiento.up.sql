\set ON_ERROR_STOP on
-- Candidata inédita local. Si la canónica ya tuviera 000011 instalada, esta
-- segregación se entrega como migración aditiva numerada allí; no se reaplica
-- ni se modifica esa historia desde este archivo.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000011',0));

-- B-BACK-01 solo conserva un borrador interno. No contiene candidato,
-- contacto, plazo, envío, disponibilidad, elegibilidad ni resultado.
DO $precondicion$
BEGIN
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_creacion_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.borrador_llamamiento_interno') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_registrador_frontera' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND rolinherit AND NOT rolreplication AND NOT rolbypassrls)
    OR EXISTS (SELECT 1 FROM pg_auth_members m JOIN pg_roles miembro ON miembro.oid=m.member WHERE miembro.rolname='vec_bolsa_llamamientos_registrador_frontera')
    OR EXISTS (SELECT 1 FROM pg_roles r WHERE r.oid<>'vec_bolsa_llamamientos_registrador_frontera'::regrole AND pg_has_role('vec_bolsa_llamamientos_registrador_frontera',r.oid,'MEMBER'))
    OR has_schema_privilege('vec_bolsa_llamamientos_registrador_frontera','vec_bolsa_llamamientos','USAGE,CREATE')
    OR EXISTS (SELECT 1 FROM pg_namespace WHERE nspowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR EXISTS (SELECT 1 FROM pg_class WHERE relowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR EXISTS (SELECT 1 FROM pg_proc WHERE proowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR EXISTS (SELECT 1 FROM pg_type WHERE typowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_bolsa_llamamientos' AND ((c.relkind='S' AND has_sequence_privilege('vec_bolsa_llamamientos_registrador_frontera',c.oid,'USAGE,SELECT,UPDATE')) OR (c.relkind IN ('r','p','v','m','f') AND (has_table_privilege('vec_bolsa_llamamientos_registrador_frontera',c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN') OR has_any_column_privilege('vec_bolsa_llamamientos_registrador_frontera',c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))))
    OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_bolsa_llamamientos' AND has_function_privilege('vec_bolsa_llamamientos_registrador_frontera',p.oid,'EXECUTE')) THEN
  RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='dependencia o estado incompatible para borrador interno';
 END IF;
END $precondicion$;

CREATE FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_contenido_valido(p jsonb)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT jsonb_typeof(p)='object' AND p ?& ARRAY['resumen']
   AND (p-'resumen')='{}'::jsonb
   AND jsonb_typeof(p->'resumen')='string'
   AND octet_length(convert_to(p->>'resumen','UTF8')) BETWEEN 3 AND 2000
   AND p->>'resumen'=btrim(p->>'resumen')
   AND p->>'resumen' !~* '(^|[^[:alpha:]])(dni|nie|nif|pasaporte|email|tel[eé]fono)([^[:alpha:]]|$)'
   AND position('@' IN p->>'resumen')=0
   AND position(chr(8232) IN p->>'resumen')=0
   AND position(chr(8233) IN p->>'resumen')=0
$f$;

CREATE TABLE vec_bolsa_llamamientos.borrador_llamamiento_interno (
 borrador_ref text PRIMARY KEY,
 propietario_ref text NOT NULL,
 unidad_ref text NOT NULL,
 ambito_ref text NOT NULL,
 clave_idempotencia text NOT NULL,
 contenido_canonico jsonb NOT NULL,
 huella_comando_sha256 text NOT NULL,
 estado text NOT NULL DEFAULT 'borrador_interno',
 version bigint NOT NULL DEFAULT 1,
 decision_ref text NOT NULL UNIQUE,
 recibo_ref text NOT NULL UNIQUE,
 creada_en timestamptz(6) NOT NULL,
 CONSTRAINT borrador_llamamiento_ref_check CHECK (borrador_ref ~ '^borrador-llamamiento:alta:[0-9a-f]{64}$'),
 CONSTRAINT borrador_llamamiento_propietario_check CHECK (propietario_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 CONSTRAINT borrador_llamamiento_referencias_check CHECK (unidad_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$' AND ambito_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$' AND clave_idempotencia ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$'),
 CONSTRAINT borrador_llamamiento_huella_check CHECK (huella_comando_sha256 ~ '^[0-9a-f]{64}$' AND huella_comando_sha256<>repeat('0',64)),
 CONSTRAINT borrador_llamamiento_estado_version_check CHECK (estado='borrador_interno' AND version=1),
 CONSTRAINT borrador_llamamiento_contenido_check CHECK (vec_bolsa_llamamientos.borrador_llamamiento_contenido_valido(contenido_canonico)),
 UNIQUE(propietario_ref,unidad_ref,clave_idempotencia)
);
CREATE TABLE vec_bolsa_llamamientos.borrador_llamamiento_historia (
 historia_ref text PRIMARY KEY,
 borrador_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.borrador_llamamiento_interno(borrador_ref),
 secuencia bigint NOT NULL CHECK (secuencia>0),
 anterior_sha256 text NOT NULL CHECK (anterior_sha256 ~ '^[0-9a-f]{64}$'),
 hecho_canonico bytea NOT NULL CHECK (octet_length(hecho_canonico) BETWEEN 1 AND 16384),
 huella_sha256 text NOT NULL CHECK (huella_sha256=encode(sha256(hecho_canonico),'hex')),
 UNIQUE(borrador_ref,secuencia)
);
CREATE TABLE vec_bolsa_llamamientos.borrador_llamamiento_auditoria (
 auditoria_ref text PRIMARY KEY,
 borrador_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.borrador_llamamiento_interno(borrador_ref),
 accion text NOT NULL CHECK (accion IN ('crear','consultar')),
 decision_ref text NOT NULL,
 registrada_en timestamptz(6) NOT NULL,
 UNIQUE(borrador_ref,accion,decision_ref)
);
CREATE TABLE vec_bolsa_llamamientos.borrador_llamamiento_outbox (
 evento_ref text PRIMARY KEY,
 borrador_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.borrador_llamamiento_interno(borrador_ref),
 evento_canonico bytea NOT NULL CHECK (octet_length(evento_canonico) BETWEEN 1 AND 16384),
 huella_sha256 text NOT NULL CHECK (huella_sha256=encode(sha256(evento_canonico),'hex')),
 emitido_en timestamptz(6) NOT NULL
);
-- Bitácora E06 de intentos fallidos/indeterminados. No es historia del
-- agregado: no contiene cuerpo, clave, referencia de recurso ni material AD3.
CREATE TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento (
 intento_ref text PRIMARY KEY,
 correlacion_ref text NOT NULL CHECK (correlacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$'),
 accion text NOT NULL CHECK (accion IN ('crear','consultar')),
 ruta_clase text NOT NULL CHECK (ruta_clase IN ('coleccion','detalle')),
 actor_ref text CHECK (actor_ref IS NULL OR actor_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 resultado text NOT NULL CHECK (resultado IN ('autenticacion_requerida','acceso_denegado','recurso_no_disponible','infraestructura_no_disponible','resultado_indeterminado')),
 registrada_en timestamptz(6) NOT NULL
);

CREATE FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_rechazar_mutacion()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='historia de borrador llamamiento inmutable'; END $f$;
CREATE TRIGGER borrador_llamamiento_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.borrador_llamamiento_interno FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_rechazar_mutacion();
CREATE TRIGGER borrador_llamamiento_historia_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.borrador_llamamiento_historia FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_rechazar_mutacion();
CREATE TRIGGER borrador_llamamiento_auditoria_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.borrador_llamamiento_auditoria FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_rechazar_mutacion();
CREATE TRIGGER borrador_llamamiento_outbox_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.borrador_llamamiento_outbox FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_rechazar_mutacion();
CREATE TRIGGER bitacora_intento_borrador_llamamiento_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_rechazar_mutacion();

ALTER TABLE vec_bolsa_llamamientos.borrador_llamamiento_interno ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.borrador_llamamiento_interno FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.borrador_llamamiento_historia ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.borrador_llamamiento_historia FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.borrador_llamamiento_auditoria ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.borrador_llamamiento_auditoria FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.borrador_llamamiento_outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.borrador_llamamiento_outbox FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento FORCE ROW LEVEL SECURITY;
CREATE POLICY borrador_llamamiento_solo_propietario ON vec_bolsa_llamamientos.borrador_llamamiento_interno TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
CREATE POLICY borrador_llamamiento_historia_solo_propietario ON vec_bolsa_llamamientos.borrador_llamamiento_historia TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
CREATE POLICY borrador_llamamiento_auditoria_solo_propietario ON vec_bolsa_llamamientos.borrador_llamamiento_auditoria TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
CREATE POLICY borrador_llamamiento_outbox_solo_propietario ON vec_bolsa_llamamientos.borrador_llamamiento_outbox TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
CREATE POLICY bitacora_intento_borrador_llamamiento_solo_propietario ON vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');

CREATE FUNCTION vec_bolsa_llamamientos.exigir_runtime_borrador_llamamiento()
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user OR
    NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') OR
    EXISTS (
        SELECT 1 FROM pg_roles r
         WHERE r.rolname LIKE 'vec_bolsa_llamamientos\_%' ESCAPE '\'
           AND r.rolname <> 'vec_bolsa_llamamientos_ejecutor'
           AND pg_has_role(session_user,r.oid,'MEMBER')
    ) OR
    EXISTS (
        SELECT 1 FROM pg_roles r
         WHERE r.rolname LIKE 'vec_contratacion_temporal\_%' ESCAPE '\'
           AND pg_has_role(session_user,r.oid,'MEMBER')
    ) OR
    current_setting('transaction_isolation')<>'serializable' OR current_setting('TimeZone')<>'UTC' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='runtime de borrador llamamiento rechazado';
 END IF;
END $f$;

-- El registrador de frontera opera después de un rechazo o rollback de negocio.
-- Su identidad técnica es distinta del ejecutor y no obtiene funciones del agregado.
CREATE FUNCTION vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento()
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user OR
    NOT pg_has_role(session_user,'vec_bolsa_llamamientos_registrador_frontera','MEMBER') OR
    EXISTS (
        SELECT 1 FROM pg_roles r
         WHERE r.rolname LIKE 'vec_bolsa_llamamientos\_%' ESCAPE '\'
           AND r.rolname <> 'vec_bolsa_llamamientos_registrador_frontera'
           AND pg_has_role(session_user,r.oid,'MEMBER')
    ) OR
    EXISTS (
        SELECT 1 FROM pg_roles r
         WHERE r.rolname LIKE 'vec_contratacion_temporal\_%' ESCAPE '\'
           AND pg_has_role(session_user,r.oid,'MEMBER')
    ) OR
    current_setting('transaction_isolation')<>'serializable' OR current_setting('TimeZone')<>'UTC' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='runtime de registrador de frontera rechazado';
 END IF;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(
 p_comando bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE(borrador_ref text,recibo_ref text,huella_comando_sha256 text,reintento_idempotente boolean,registrado_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; contenido jsonb; canon text; h text; h_contexto text; preimagen_contexto text; ref text; consumo record; existente record; ahora timestamptz; hecho bytea; evento bytea;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_borrador_llamamiento();
 IF p_comando IS NULL OR octet_length(p_comando) NOT BETWEEN 1 AND 16384 THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='comando de borrador inválido'; END IF;
 BEGIN c:=convert_from(p_comando,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='comando de borrador inválido'; END;
 IF jsonb_typeof(c)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(c))<>6 OR NOT c ?& ARRAY['esquema','propietario_ref','unidad_ref','ambito_ref','clave_idempotencia','contenido'] OR c->>'esquema'<>'vec.bolsa.llamamiento.borrador-interno.crear.v1' OR jsonb_typeof(c->'propietario_ref')<>'string' OR c->>'propietario_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$' OR jsonb_typeof(c->'unidad_ref')<>'string' OR c->>'unidad_ref' !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$' OR jsonb_typeof(c->'ambito_ref')<>'string' OR c->>'ambito_ref' !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$' OR jsonb_typeof(c->'clave_idempotencia')<>'string' OR c->>'clave_idempotencia' !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$' THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='comando de borrador inválido'; END IF;
 contenido:=c->'contenido';
 IF vec_bolsa_llamamientos.borrador_llamamiento_contenido_valido(contenido) IS NOT TRUE THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='contenido de borrador inválido'; END IF;
 canon:=format('{"esquema":%s,"propietario_ref":%s,"unidad_ref":%s,"ambito_ref":%s,"clave_idempotencia":%s,"contenido":{"resumen":%s}}',to_json(c->>'esquema')::text,to_json(c->>'propietario_ref')::text,to_json(c->>'unidad_ref')::text,to_json(c->>'ambito_ref')::text,to_json(c->>'clave_idempotencia')::text,to_json(contenido->>'resumen')::text);
 IF p_comando IS DISTINCT FROM convert_to(canon,'UTF8') THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='comando de borrador no canónico'; END IF;
 h:=encode(sha256(convert_to(canon,'UTF8')),'hex'); ref:='borrador-llamamiento:alta:'||h;
 PERFORM pg_advisory_xact_lock(hashtextextended('borrador-llamamiento:'||(c->>'propietario_ref')||':'||(c->>'unidad_ref')||':'||(c->>'clave_idempotencia'),0));
 SELECT * INTO consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_creacion_borrador_llamamiento_interno_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='autorización de borrador no ligada'; END;
 preimagen_contexto:=format('{"ambitos":{"ambito_ref":%s,"unidad_ref":%s},"atributos":{}}',to_json(c->>'ambito_ref')::text,to_json(c->>'unidad_ref')::text);
 h_contexto:=encode(sha256(convert_to(preimagen_contexto,'UTF8')),'hex');
 IF consumo.efecto_ref IS DISTINCT FROM ref OR consumo.consumo_nuevo IS NOT TRUE
    OR d->>'principal_id' IS DISTINCT FROM c->>'propietario_ref'
    OR d->>'accion' IS DISTINCT FROM 'bolsa.llamamiento.borrador_interno.crear'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'borrador_llamamiento_interno'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_borradores_llamamiento_interno'
    OR d->>'recurso_ref' IS DISTINCT FROM ref
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h_contexto
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM h_contexto THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='autorización de borrador no ligada'; END IF;
 SELECT * INTO existente FROM vec_bolsa_llamamientos.borrador_llamamiento_interno WHERE propietario_ref=c->>'propietario_ref' AND unidad_ref=c->>'unidad_ref' AND clave_idempotencia=c->>'clave_idempotencia' FOR SHARE;
 IF FOUND THEN
  IF existente.huella_comando_sha256 IS DISTINCT FROM h THEN RAISE EXCEPTION USING ERRCODE='VBL01',MESSAGE='clave de borrador divergente'; END IF;
  RETURN QUERY SELECT existente.borrador_ref,existente.recibo_ref,existente.huella_comando_sha256,true,existente.creada_en; RETURN;
 END IF;
 ahora:=clock_timestamp();
 INSERT INTO vec_bolsa_llamamientos.borrador_llamamiento_interno VALUES(ref,c->>'propietario_ref',c->>'unidad_ref',c->>'ambito_ref',c->>'clave_idempotencia',contenido,h,'borrador_interno',1,consumo.decision_ref,'recibo:'||translate(h,'0123456789','ghijklmnop'),ahora);
 hecho:=convert_to(jsonb_build_object('esquema','vec.bolsa.llamamiento.borrador.historia.v1','borrador_ref',ref,'accion','crear','version',1,'decision_ref',consumo.decision_ref,'registrada_en',ahora)::text,'UTF8');
 INSERT INTO vec_bolsa_llamamientos.borrador_llamamiento_historia VALUES('historia:'||translate(h,'0123456789','ghijklmnop'),ref,1,repeat('0',64),hecho,encode(sha256(hecho),'hex'));
 INSERT INTO vec_bolsa_llamamientos.borrador_llamamiento_auditoria VALUES('auditoria:'||translate(h,'0123456789','ghijklmnop'),ref,'crear',consumo.decision_ref,ahora);
 evento:=convert_to(jsonb_build_object('esquema','vec.bolsa.llamamiento.borrador.outbox.v1','tipo','bolsa.llamamiento.borrador_interno.creado','borrador_ref',ref,'recibo_ref','recibo:'||translate(h,'0123456789','ghijklmnop'),'emitido_en',ahora)::text,'UTF8');
 INSERT INTO vec_bolsa_llamamientos.borrador_llamamiento_outbox VALUES('evento:'||translate(h,'0123456789','ghijklmnop'),ref,evento,encode(sha256(evento),'hex'),ahora);
 RETURN QUERY SELECT ref,'recibo:'||translate(h,'0123456789','ghijklmnop'),h,false,ahora;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.consultar_borrador_llamamiento_interno_v1(
 p_consulta bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE(borrador_ref text,propietario_ref text,unidad_ref text,ambito_ref text,contenido_canonico jsonb,estado text,version bigint,huella_comando_sha256 text,recibo_ref text,creada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE b record; consumo record; c jsonb; d jsonb; h_contexto text; preimagen_contexto text; ahora timestamptz;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_borrador_llamamiento();
 IF p_consulta IS NULL OR octet_length(p_consulta) NOT BETWEEN 1 AND 4096 THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='consulta de borrador inválida'; END IF;
 BEGIN c:=convert_from(p_consulta,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='consulta de borrador inválida'; END;
 IF jsonb_typeof(c)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(c))<>5 OR NOT c ?& ARRAY['esquema','borrador_ref','propietario_ref','unidad_ref','ambito_ref'] OR c->>'esquema'<>'vec.bolsa.llamamiento.borrador-interno-consulta.v1' OR c->>'borrador_ref' !~ '^borrador-llamamiento:alta:[0-9a-f]{64}$' OR c->>'propietario_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$' OR c->>'unidad_ref' !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$' OR c->>'ambito_ref' !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$' THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='consulta de borrador inválida'; END IF;
 SELECT * INTO consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_borrador_llamamiento_interno_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de borrador no autorizada'; END;
 preimagen_contexto:=format('{"ambitos":{"ambito_ref":%s,"unidad_ref":%s},"atributos":{}}',to_json(c->>'ambito_ref')::text,to_json(c->>'unidad_ref')::text);
 h_contexto:=encode(sha256(convert_to(preimagen_contexto,'UTF8')),'hex');
 IF consumo.efecto_ref IS DISTINCT FROM c->>'borrador_ref' OR consumo.consumo_nuevo IS NOT TRUE
    OR d->>'principal_id' IS DISTINCT FROM c->>'propietario_ref'
    OR d->>'accion' IS DISTINCT FROM 'bolsa.llamamiento.borrador_interno.consultar'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'borrador_llamamiento_interno'
    OR d->>'finalidad' IS DISTINCT FROM 'consulta_borrador_llamamiento_interno'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'borrador_ref'
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h_contexto
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM h_contexto THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de borrador no autorizada'; END IF;
 SELECT bi.* INTO b FROM vec_bolsa_llamamientos.borrador_llamamiento_interno bi WHERE bi.borrador_ref=c->>'borrador_ref' AND bi.propietario_ref=c->>'propietario_ref' AND bi.unidad_ref=c->>'unidad_ref' AND bi.ambito_ref=c->>'ambito_ref' FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='borrador no disponible'; END IF;
 ahora:=clock_timestamp(); INSERT INTO vec_bolsa_llamamientos.borrador_llamamiento_auditoria (auditoria_ref,borrador_ref,accion,decision_ref,registrada_en) VALUES ('auditoria:lectura:'||translate(encode(sha256(convert_to((c->>'borrador_ref')||':'||consumo.decision_ref,'UTF8')),'hex'),'0123456789','ghijklmnop'),c->>'borrador_ref','consultar',consumo.decision_ref,ahora);
 RETURN QUERY SELECT b.borrador_ref,b.propietario_ref,b.unidad_ref,b.ambito_ref,b.contenido_canonico,b.estado,b.version,b.huella_comando_sha256,b.recibo_ref,b.creada_en;
END $f$;

-- Firma: registrar_intento_borrador_llamamiento_v1(text,text,text,text,text).
-- Se invoca en una operación posterior a una transacción fallida; por eso no
-- tiene FK ni toca agregado, historia, auditoría de éxito ni outbox.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(
 p_correlacion_ref text,p_accion text,p_ruta_clase text,p_actor_ref text,p_resultado text
) RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_ref text;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento();
 IF p_correlacion_ref IS NULL OR p_correlacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$'
    OR p_accion NOT IN ('crear','consultar') OR p_ruta_clase NOT IN ('coleccion','detalle')
    OR (p_actor_ref IS NOT NULL AND p_actor_ref !~ '^per_[A-Za-z0-9_-]{22,128}$')
    OR p_resultado NOT IN ('autenticacion_requerida','acceso_denegado','recurso_no_disponible','infraestructura_no_disponible','resultado_indeterminado') THEN
  RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='intento de borrador inválido';
 END IF;
 v_ref:='intento:'||translate(encode(sha256(convert_to(p_correlacion_ref||':'||p_accion||':'||p_ruta_clase||':'||coalesce(p_actor_ref,'')||':'||p_resultado||':'||clock_timestamp()::text,'UTF8')),'hex'),'0123456789','ghijklmnop');
 INSERT INTO vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento (intento_ref,correlacion_ref,accion,ruta_clase,actor_ref,resultado,registrada_en)
 VALUES(v_ref,p_correlacion_ref,p_accion,p_ruta_clase,p_actor_ref,p_resultado,clock_timestamp());
END $f$;

REVOKE ALL ON vec_bolsa_llamamientos.borrador_llamamiento_interno,vec_bolsa_llamamientos.borrador_llamamiento_historia,vec_bolsa_llamamientos.borrador_llamamiento_auditoria,vec_bolsa_llamamientos.borrador_llamamiento_outbox,vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_registrador_frontera;
-- Las ACL históricas no se revocan ni normalizan aquí: una concesión residual
-- al registrador, incluso por PUBLIC o membresía, rechaza la candidata.
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_contenido_valido(jsonb),vec_bolsa_llamamientos.borrador_llamamiento_rechazar_mutacion(),vec_bolsa_llamamientos.exigir_runtime_borrador_llamamiento(),vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento(),vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_bolsa_llamamientos.consultar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_contenido_valido(jsonb),vec_bolsa_llamamientos.borrador_llamamiento_rechazar_mutacion(),vec_bolsa_llamamientos.exigir_runtime_borrador_llamamiento(),vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento(),vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_bolsa_llamamientos.consultar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_registrador_frontera;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text) FROM vec_bolsa_llamamientos_ejecutor;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_bolsa_llamamientos_registrador_frontera;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_bolsa_llamamientos.consultar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text) TO vec_bolsa_llamamientos_registrador_frontera;
REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_bolsa_llamamientos.consultar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_ejecutor;
REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text) FROM vec_bolsa_llamamientos_registrador_frontera;
-- Las ACL por defecto pueden haber sido ampliadas por una instalación anfitriona.
-- RLS no cubre TRUNCATE, REFERENCES ni TRIGGER: retirar toda concesión que no
-- pertenezca al propietario técnico, y para las fachadas solo conservar ejecutor.
DO $acl_cerrada$
DECLARE objeto regclass; funcion regprocedure; a record;
BEGIN
 FOREACH objeto IN ARRAY ARRAY['vec_bolsa_llamamientos.borrador_llamamiento_interno'::regclass,'vec_bolsa_llamamientos.borrador_llamamiento_historia'::regclass,'vec_bolsa_llamamientos.borrador_llamamiento_auditoria'::regclass,'vec_bolsa_llamamientos.borrador_llamamiento_outbox'::regclass,'vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass] LOOP
  FOR a IN SELECT DISTINCT x.grantee FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x WHERE c.oid=objeto AND x.grantee<>c.relowner LOOP
   IF a.grantee=0 THEN EXECUTE format('REVOKE ALL ON TABLE %s FROM PUBLIC',objeto::text);
   ELSE EXECUTE format('REVOKE ALL ON TABLE %s FROM %I',objeto::text,pg_get_userbyid(a.grantee)); END IF;
  END LOOP;
 END LOOP;
 FOREACH funcion IN ARRAY ARRAY['vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'vec_bolsa_llamamientos.consultar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure] LOOP
  FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=funcion AND x.grantee<>p.proowner AND x.grantee<>'vec_bolsa_llamamientos_ejecutor'::regrole LOOP
   IF a.grantee=0 THEN EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',funcion::text);
   ELSE EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',funcion::text,pg_get_userbyid(a.grantee)); END IF;
  END LOOP;
 END LOOP;
 funcion:='vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text)'::regprocedure;
 FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=funcion AND x.grantee<>p.proowner AND x.grantee<>'vec_bolsa_llamamientos_registrador_frontera'::regrole LOOP
  IF a.grantee=0 THEN EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',funcion::text);
  ELSE EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',funcion::text,pg_get_userbyid(a.grantee)); END IF;
 END LOOP;
END $acl_cerrada$;

-- Antes del COMMIT se comprueban privilegios efectivos, no solo ACL directas.
-- La única capacidad visible del grupo es la fachada exacta de bitácora.
DO $segregacion_efectiva$
DECLARE f regprocedure := 'vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text)'::regprocedure;
BEGIN
 IF NOT has_schema_privilege('vec_bolsa_llamamientos_registrador_frontera','vec_bolsa_llamamientos','USAGE')
    OR has_schema_privilege('vec_bolsa_llamamientos_registrador_frontera','vec_bolsa_llamamientos','CREATE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_registrador_frontera',f,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_roles r WHERE r.oid<>'vec_bolsa_llamamientos_registrador_frontera'::regrole AND pg_has_role('vec_bolsa_llamamientos_registrador_frontera',r.oid,'MEMBER'))
    OR EXISTS (SELECT 1 FROM pg_namespace WHERE nspowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR EXISTS (SELECT 1 FROM pg_class WHERE relowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR EXISTS (SELECT 1 FROM pg_proc WHERE proowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR EXISTS (SELECT 1 FROM pg_type WHERE typowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_bolsa_llamamientos' AND ((c.relkind='S' AND has_sequence_privilege('vec_bolsa_llamamientos_registrador_frontera',c.oid,'USAGE,SELECT,UPDATE')) OR (c.relkind IN ('r','p','v','m','f') AND (has_table_privilege('vec_bolsa_llamamientos_registrador_frontera',c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN') OR has_any_column_privilege('vec_bolsa_llamamientos_registrador_frontera',c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))))
    OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_bolsa_llamamientos' AND p.oid<>f AND has_function_privilege('vec_bolsa_llamamientos_registrador_frontera',p.oid,'EXECUTE')) THEN
  RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='segregación efectiva del registrador de frontera incompatible';
 END IF;
END $segregacion_efectiva$;
COMMIT;
