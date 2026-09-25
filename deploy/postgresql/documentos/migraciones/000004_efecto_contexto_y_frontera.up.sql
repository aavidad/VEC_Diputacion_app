\set ON_ERROR_STOP on
-- Documentos-4: (1) publica huella_efecto_v1, la huella del contexto de
-- recurso V3 documental, y (2) registra las denegaciones de la frontera HTTP
-- de /api/vec/documentos/.
--
-- (1) consumir_v3_v1/v2 (000001/000002) ya exigen huella_efecto_sha256 =
-- contexto_recurso_huella_sha256 = SHA-256 del contexto canónico
-- {"ambitos":{},"atributos":{"preimagen_sha256":"<hex>"}}; esta migración no
-- los reescribe: comprueba por huella SHA-256 que su cuerpo instalado es el
-- exacto de 000001/000002, que liga esa huella, y publica la misma
-- expresión como función para las pruebas y el preflight.
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
    -- Cuerpos exactos instalados por 000001/000002 (SHA-256 de prosrc en
    -- UTF-8): cualquier alteración, aunque conserve las subcadenas de la
    -- ligadura, aborta la migración.
    OR (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') FROM pg_proc p WHERE p.oid=
        'vec_documentos.consumir_v3_v1(bytea,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
       IS DISTINCT FROM '42fbad78031867af219766d9998166814b33c2d353a0b4570fac3c6e914f719d'
    OR (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') FROM pg_proc p WHERE p.oid=
        'vec_documentos.consumir_v3_v2(bytea,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
       IS DISTINCT FROM '1376ed85a21a46c7805af86ea0a47c10d1adb58a96e4c7801953b57d7dc10b6f'
 THEN RAISE EXCEPTION 'Documentos-4: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Contexto canónico del recurso V3 documental (mismo texto que produce
-- json.Marshal en Go para RecursoAutorizable.HuellaContextoAutorizacionSHA256).
CREATE FUNCTION vec_documentos.huella_efecto_v1(p bytea) RETURNS text
LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog AS $f$
 SELECT encode(sha256(convert_to('{"ambitos":{},"atributos":{"preimagen_sha256":"'||encode(sha256(p),'hex')||'"}}','UTF8')),'hex')
 $f$;

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
    OR NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_documentos_auditor'::regrole
       AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
    OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)<>1
    OR EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_documentos_auditor'::regrole)
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
