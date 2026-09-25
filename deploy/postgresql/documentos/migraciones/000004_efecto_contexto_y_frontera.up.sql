\set ON_ERROR_STOP on
-- Documentos-4: (1) liga la decisión V3 a la preimagen mediante la huella del
-- contexto de recurso que emite el PDP real, y (2) registra las denegaciones
-- de la frontera HTTP de /api/vec/documentos/.
--
-- (1) El núcleo V3 fija huella_efecto_sha256 = contexto_recurso_huella_sha256
-- = SHA-256 del contexto canónico {"ambitos":{...},"atributos":{...}} del
-- recurso autorizado. 000001/000002 exigían SHA-256(preimagen), que ningún
-- emisor V3 real puede producir. El recurso documental lleva ámbitos vacíos
-- y el único atributo preimagen_sha256; se recalcula aquí sin cambiar la
-- firma, ACL ni propietario de consumir_v3_v1/v2. La columna
-- huella_preimagen_sha256 conserva SHA-256(preimagen) como hasta ahora.
--
-- Requiere roles_000004_up.sql (DBA). Conserva 000001–000003 e historia.
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000004',0));
DO $pre$
BEGIN
 IF current_user<>'vec_documentos_propietario'
    OR to_regprocedure('vec_documentos.consumir_v3_v1(bytea,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_documentos.consumir_v3_v2(bytea,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_documentos.referencia_externa') IS NULL
    OR to_regprocedure('vec_documentos.huella_efecto_v1(bytea)') IS NOT NULL
    OR to_regclass('vec_documentos.denegacion_frontera') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_documentos_auditor' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'Documentos-4: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Contexto canónico del recurso V3 documental (mismo texto que produce
-- json.Marshal en Go para RecursoAutorizable.HuellaContextoAutorizacionSHA256).
CREATE FUNCTION vec_documentos.huella_efecto_v1(p bytea) RETURNS text
LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog AS $f$
 SELECT encode(sha256(convert_to('{"ambitos":{},"atributos":{"preimagen_sha256":"'||encode(sha256(p),'hex')||'"}}','UTF8')),'hex')
 $f$;

CREATE OR REPLACE FUNCTION vec_documentos.consumir_v3_v1(
 p_preimagen bytea,p_accion text,p_recurso text,p_principal text,p_perfil text,p_finalidad text,p_correlacion text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; v record; h text;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_documentos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_documentos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_documentos_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR octet_length(p_preimagen) NOT BETWEEN 1 AND 16384
 THEN RAISE EXCEPTION 'documentos: contexto de ejecución denegado' USING ERRCODE='42501'; END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'documentos: autorización inválida' USING ERRCODE='42501'; END;
 h:=vec_documentos.huella_efecto_v1(p_preimagen);
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_documentos.operacion.v1'
    OR c->>'operacion' IS DISTINCT FROM p_accion
    OR c->>'efecto_ref' IS DISTINCT FROM p_recurso
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM h
    OR d->>'accion' IS DISTINCT FROM p_accion
    OR d->>'modulo_id' IS DISTINCT FROM 'documentos'
    OR d->>'recurso_ref' IS DISTINCT FROM p_recurso
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h
    OR d->>'principal_id' IS DISTINCT FROM p_principal
    OR d->>'perfil_activo_ref' IS DISTINCT FROM p_perfil
    OR d->>'finalidad' IS DISTINCT FROM p_finalidad
    OR d->>'correlacion_ref' IS DISTINCT FROM p_correlacion
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'documentos: decisión no ligada al efecto' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_operacion_documentos_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE OR v.efecto_ref IS DISTINCT FROM p_recurso OR v.huella_efecto_sha256 IS DISTINCT FROM h
 THEN RAISE EXCEPTION 'documentos: consumo AD3 no ligado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT v.decision_ref,v.efecto_ref,v.huella_efecto_sha256,v.consumo_huella_sha256,v.auditoria_ref,v.consumida_en,true;
END $f$;

CREATE OR REPLACE FUNCTION vec_documentos.consumir_v3_v2(
 p_preimagen bytea,p_accion text,p_recurso text,p_principal text,p_perfil text,p_finalidad text,p_correlacion text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; v record; h text;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_documentos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_documentos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_documentos_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR octet_length(p_preimagen) NOT BETWEEN 1 AND 16384
 THEN RAISE EXCEPTION 'documentos: contexto de ejecución denegado' USING ERRCODE='42501'; END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'documentos: autorización inválida' USING ERRCODE='42501'; END;
 h:=vec_documentos.huella_efecto_v1(p_preimagen);
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_documentos.operacion.v1'
    OR c->>'operacion' IS DISTINCT FROM p_accion
    OR c->>'efecto_ref' IS DISTINCT FROM p_recurso
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM h
    OR d->>'accion' IS DISTINCT FROM p_accion
    OR d->>'modulo_id' IS DISTINCT FROM 'documentos'
    OR d->>'recurso_ref' IS DISTINCT FROM p_recurso
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h
    OR d->>'principal_id' IS DISTINCT FROM p_principal
    OR d->>'perfil_activo_ref' IS DISTINCT FROM p_perfil
    OR d->>'finalidad' IS DISTINCT FROM p_finalidad
    OR d->>'correlacion_ref' IS DISTINCT FROM p_correlacion
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'documentos: decisión no ligada al efecto' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NULL OR v.efecto_ref IS DISTINCT FROM p_recurso OR v.huella_efecto_sha256 IS DISTINCT FROM h
 THEN RAISE EXCEPTION 'documentos: consumo AD3 no ligado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT v.decision_ref,v.efecto_ref,v.huella_efecto_sha256,v.consumo_huella_sha256,v.auditoria_ref,v.consumida_en,v.consumo_nuevo;
END $f$;

-- (2) Denegaciones de la frontera HTTP: sin identidad, acceso denegado o
-- dependencia caída. Solo referencias opacas y valores cerrados; escribe el
-- auditor, nadie lee por esta vía y la historia es de solo adición.
CREATE TABLE vec_documentos.denegacion_frontera (
 denegacion_ref text PRIMARY KEY,
 correlacion_ref text NOT NULL CHECK(correlacion_ref ~ '^corr_([0-9a-f]{32}|no_disponible)$'),
 motivo text NOT NULL CHECK(motivo IN ('autenticacion_requerida','acceso_denegado','dependencia')),
 ruta text NOT NULL CHECK(ruta IN ('/api/vec/documentos/expedientes/consultas','/api/vec/documentos/originales/descargas','otra')),
 metodo text NOT NULL CHECK(metodo IN ('POST','otro')),
 actor_ref text CHECK(actor_ref IS NULL OR (length(actor_ref) BETWEEN 3 AND 256 AND actor_ref ~ '^[A-Za-z0-9:_-]+$')),
 registrada_en timestamptz(6) NOT NULL
);
ALTER TABLE vec_documentos.denegacion_frontera ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.denegacion_frontera FORCE ROW LEVEL SECURITY;
CREATE POLICY registrar_denegacion ON vec_documentos.denegacion_frontera
 FOR INSERT TO vec_documentos_propietario WITH CHECK (true);
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_documentos.denegacion_frontera
 FOR EACH ROW EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_documentos.denegacion_frontera
 FOR EACH STATEMENT EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1();

CREATE FUNCTION vec_documentos.registrar_denegacion_frontera_v1(
 p_correlacion text,p_motivo text,p_ruta text,p_metodo text,p_actor text) RETURNS text
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE ref text;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_documentos_auditor','MEMBER')
    OR pg_has_role(session_user,'vec_documentos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_documentos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_documentos_migrador','MEMBER')
 THEN RAISE EXCEPTION 'documentos: auditor de frontera inválido' USING ERRCODE='42501'; END IF;
 -- Sin RETURNING ni SELECT: la tabla no concede lectura a nadie por esta vía.
 ref:='denegacion:documentos:'||gen_random_uuid()::text;
 INSERT INTO vec_documentos.denegacion_frontera(denegacion_ref,correlacion_ref,motivo,ruta,metodo,actor_ref,registrada_en)
 VALUES(ref,p_correlacion,p_motivo,p_ruta,p_metodo,nullif(p_actor,''),date_trunc('microseconds',clock_timestamp()));
 RETURN ref;
END $f$;

GRANT USAGE ON SCHEMA vec_documentos TO vec_documentos_auditor;
REVOKE ALL ON TABLE vec_documentos.denegacion_frontera FROM PUBLIC,vec_documentos_ejecutor,vec_documentos_migrador,vec_documentos_auditor;
REVOKE ALL ON FUNCTION vec_documentos.huella_efecto_v1(bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_documentos.registrar_denegacion_frontera_v1(text,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.registrar_denegacion_frontera_v1(text,text,text,text,text) TO vec_documentos_auditor;
COMMIT;
