\set ON_ERROR_STOP on
-- Fuente candidata. Requiere roles de contacto, AD3-35 y T13-6 instalados.
-- No crea LOGIN, políticas de perfiles ni un vínculo con candidatos de Bolsa.
BEGIN;
SET LOCAL ROLE vec_contacto_usuario_owner;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:migracion:1',0));

DO $precondicion$
BEGIN
    IF current_user <> 'vec_contacto_usuario_owner'
       OR EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_contacto_usuario_v1')
       OR to_regprocedure('vec_bolsa_registro_accesos.registrar_contacto_usuario_v1(bytea,bytea,bytea,bytea,bytea,text,text,text)') IS NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_alta_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_actualizar_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consultar_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.revalidar_consulta_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.revalidar_alta_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.revalidar_actualizar_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL THEN
        RAISE EXCEPTION 'contacto: preimagen o dependencias incompatibles' USING ERRCODE='55000';
    END IF;
END
$precondicion$;

CREATE SCHEMA vec_contacto_usuario_v1 AUTHORIZATION vec_contacto_usuario_owner;
REVOKE ALL ON SCHEMA vec_contacto_usuario_v1 FROM PUBLIC;

CREATE TABLE vec_contacto_usuario_v1.versiones (
    sujeto_ref text NOT NULL,
    version numeric(20,0) NOT NULL CHECK(version BETWEEN 1 AND 9223372036854775807),
    clave_ref text NOT NULL CHECK(length(clave_ref) BETWEEN 1 AND 512),
    nonce bytea NOT NULL CHECK(octet_length(nonce) BETWEEN 12 AND 64),
    cifrado bytea NOT NULL CHECK(octet_length(cifrado) BETWEEN 16 AND 32768),
    negocio bytea NOT NULL CHECK(octet_length(negocio) BETWEEN 1 AND 65536),
    contexto_recurso bytea NOT NULL CHECK(octet_length(contexto_recurso) BETWEEN 1 AND 65536),
    decision_ref text NOT NULL,
    consumo_ref text NOT NULL UNIQUE,
    consumo_huella_sha256 text NOT NULL CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
    auditoria_central bytea NOT NULL CHECK(octet_length(auditoria_central)>0),
    registrada_en timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY(sujeto_ref,version)
);
CREATE TABLE vec_contacto_usuario_v1.actual (
    sujeto_ref text PRIMARY KEY,
    version numeric(20,0) NOT NULL,
    FOREIGN KEY(sujeto_ref,version) REFERENCES vec_contacto_usuario_v1.versiones(sujeto_ref,version)
);
CREATE TABLE vec_contacto_usuario_v1.outbox (
    consumo_ref text PRIMARY KEY REFERENCES vec_contacto_usuario_v1.versiones(consumo_ref),
    sujeto_ref text NOT NULL,
    version numeric(20,0) NOT NULL,
    accion text NOT NULL CHECK(accion IN ('vec.contacto_usuario.alta','vec.contacto_usuario.actualizar')),
    registrada_en timestamptz NOT NULL DEFAULT clock_timestamp(),
    FOREIGN KEY(sujeto_ref,version) REFERENCES vec_contacto_usuario_v1.versiones(sujeto_ref,version)
);

ALTER TABLE vec_contacto_usuario_v1.versiones ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contacto_usuario_v1.versiones FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_contacto_usuario_v1.actual ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contacto_usuario_v1.actual FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_contacto_usuario_v1.outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contacto_usuario_v1.outbox FORCE ROW LEVEL SECURITY;
-- El contexto se fija sólo después de validar AD3 y con sus selectores exactos.
-- No hay concesiones directas a tablas para los roles de aplicación.
CREATE POLICY lectura ON vec_contacto_usuario_v1.versiones FOR SELECT TO vec_contacto_usuario_owner USING(
    sujeto_ref=current_setting('vec.contacto.sujeto_ref',true)
    AND version::text=current_setting('vec.contacto.version',true)
    AND current_setting('vec.contacto.accion',true)='vec.contacto_usuario.consultar');
CREATE POLICY alta ON vec_contacto_usuario_v1.versiones FOR INSERT TO vec_contacto_usuario_owner WITH CHECK(
    sujeto_ref=current_setting('vec.contacto.sujeto_ref',true)
    AND version::text=current_setting('vec.contacto.version',true)
    AND current_setting('vec.contacto.accion',true) IN ('vec.contacto_usuario.alta','vec.contacto_usuario.actualizar'));
CREATE POLICY lectura ON vec_contacto_usuario_v1.actual FOR SELECT TO vec_contacto_usuario_owner USING(
    sujeto_ref=current_setting('vec.contacto.sujeto_ref',true)
    AND version::text IN (current_setting('vec.contacto.version',true),current_setting('vec.contacto.version_anterior',true))
    AND current_setting('vec.contacto.accion',true) IN ('vec.contacto_usuario.alta','vec.contacto_usuario.actualizar','vec.contacto_usuario.consultar'));
CREATE POLICY alta ON vec_contacto_usuario_v1.actual FOR INSERT TO vec_contacto_usuario_owner WITH CHECK(
    sujeto_ref=current_setting('vec.contacto.sujeto_ref',true)
    AND version::text=current_setting('vec.contacto.version',true)
    AND current_setting('vec.contacto.accion',true) IN ('vec.contacto_usuario.alta','vec.contacto_usuario.actualizar'));
CREATE POLICY cambio ON vec_contacto_usuario_v1.actual FOR UPDATE TO vec_contacto_usuario_owner USING(
    sujeto_ref=current_setting('vec.contacto.sujeto_ref',true)
    AND version::text=current_setting('vec.contacto.version_anterior',true)
    AND current_setting('vec.contacto.accion',true)='vec.contacto_usuario.actualizar') WITH CHECK(
    sujeto_ref=current_setting('vec.contacto.sujeto_ref',true)
    AND version::text=current_setting('vec.contacto.version',true)
    AND current_setting('vec.contacto.accion',true)='vec.contacto_usuario.actualizar');
-- SELECT FOR SHARE también requiere una política UPDATE visible. Esta política
-- permite bloquear la fila consultada, pero nunca modificarla durante la lectura.
CREATE POLICY bloqueo_consulta ON vec_contacto_usuario_v1.actual FOR UPDATE TO vec_contacto_usuario_owner USING(
    sujeto_ref=current_setting('vec.contacto.sujeto_ref',true)
    AND version::text=current_setting('vec.contacto.version',true)
    AND current_setting('vec.contacto.accion',true)='vec.contacto_usuario.consultar') WITH CHECK(false);
CREATE POLICY alta ON vec_contacto_usuario_v1.outbox FOR INSERT TO vec_contacto_usuario_owner WITH CHECK(
    sujeto_ref=current_setting('vec.contacto.sujeto_ref',true)
    AND version::text=current_setting('vec.contacto.version',true)
    AND accion=current_setting('vec.contacto.accion',true));
REVOKE ALL ON ALL TABLES IN SCHEMA vec_contacto_usuario_v1 FROM PUBLIC;

CREATE FUNCTION vec_contacto_usuario_v1.registrar_contacto_v1(
    p_accion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload_v3 bytea,p_sobre_cose bytea,
    p_evidencia bytea,p_raiz bytea,p_negocio bytea,p_recurso bytea,p_auditoria bytea
) RETURNS TABLE(sujeto_ref text,version numeric,auditoria_central bytea,consumo_ref text,consumo_huella_sha256 text)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC'
AS $f$
DECLARE
    b jsonb; v_sujeto text; v_anterior numeric; v_actual numeric; v_nueva numeric;
    v_clave text; v_nonce bytea; v_cifrado bytea; v_consumo record; v_recibo bytea; v_revalidacion record;
BEGIN
    IF p_accion IS NULL OR p_accion NOT IN ('vec.contacto_usuario.alta','vec.contacto_usuario.actualizar')
       OR current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off'
       OR NOT pg_has_role(session_user,'vec_contacto_usuario_writer','MEMBER') THEN
        RAISE EXCEPTION 'contacto: escritura denegada' USING ERRCODE='42501';
    END IF;
    IF p_accion='vec.contacto_usuario.alta' THEN
        SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_alta_contacto_usuario_v3_atestada(
            p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
            p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
    ELSE
        SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_actualizar_contacto_usuario_v3_atestada(
            p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
            p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
    END IF;
    IF v_consumo.consumo_nuevo IS DISTINCT FROM true THEN
        RAISE EXCEPTION 'contacto: se requiere autorización fresca' USING ERRCODE='P1102';
    END IF;
    b:=convert_from(p_negocio,'UTF8')::jsonb;
    v_sujeto:=b->>'SujetoRef';
    v_anterior:=(b->>'VersionEsperada')::numeric;
    v_nueva:=(b->>'VersionNueva')::numeric;
    v_clave:=b->'Sobre'->>'ClaveRef';
    v_nonce:=decode(b->'Sobre'->>'Nonce','base64');
    v_cifrado:=decode(b->'Sobre'->>'Cifrado','base64');
    IF v_sujeto IS NULL OR v_anterior IS NULL OR v_nueva IS NULL
       OR v_anterior<0 OR v_nueva<>v_anterior+1
       OR (p_accion='vec.contacto_usuario.alta') IS DISTINCT FROM (v_anterior=0)
       OR (b->'Sobre'->>'Version')::numeric IS DISTINCT FROM v_nueva THEN
        RAISE EXCEPTION 'contacto: versión inválida' USING ERRCODE='22023';
    END IF;
    PERFORM set_config('vec.contacto.sujeto_ref',v_sujeto,true),set_config('vec.contacto.accion',p_accion,true),
        set_config('vec.contacto.version',v_nueva::text,true),set_config('vec.contacto.version_anterior',v_anterior::text,true);
    SELECT a.version INTO v_actual FROM vec_contacto_usuario_v1.actual a
        WHERE a.sujeto_ref=v_sujeto FOR UPDATE;
    IF coalesce(v_actual,0)<>v_anterior THEN
        RAISE EXCEPTION 'contacto: conflicto de versión' USING ERRCODE='P1103';
    END IF;
    v_recibo:=convert_to(vec_bolsa_registro_accesos.registrar_contacto_usuario_v1(
        p_auditoria,p_negocio,p_recurso,p_decision,p_contexto,v_consumo.decision_ref,
        v_consumo.auditoria_ref,v_consumo.consumo_huella_sha256)::text,'UTF8');
    INSERT INTO vec_contacto_usuario_v1.versiones(
        sujeto_ref,version,clave_ref,nonce,cifrado,negocio,contexto_recurso,decision_ref,
        consumo_ref,consumo_huella_sha256,auditoria_central)
        VALUES(v_sujeto,v_nueva,v_clave,v_nonce,v_cifrado,p_negocio,p_recurso,v_consumo.decision_ref,
            v_consumo.auditoria_ref,v_consumo.consumo_huella_sha256,v_recibo);
    INSERT INTO vec_contacto_usuario_v1.actual AS a(sujeto_ref,version) VALUES(v_sujeto,v_nueva)
        ON CONFLICT ON CONSTRAINT actual_pkey DO UPDATE SET version=excluded.version;
    INSERT INTO vec_contacto_usuario_v1.outbox(consumo_ref,sujeto_ref,version,accion)
        VALUES(v_consumo.auditoria_ref,v_sujeto,v_nueva,p_accion);
    -- Toda espera de estado/outbox/T13 ha terminado: comprobar la autorización
    -- viva del mismo consumo antes de devolver un resultado confirmable.
    IF p_accion='vec.contacto_usuario.alta' THEN
        SELECT * INTO STRICT v_revalidacion FROM vec_autorizacion_atestada_v3.revalidar_alta_contacto_usuario_v3_atestada(
            p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
            p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
    ELSE
        SELECT * INTO STRICT v_revalidacion FROM vec_autorizacion_atestada_v3.revalidar_actualizar_contacto_usuario_v3_atestada(
            p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
            p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
    END IF;
    IF v_revalidacion.decision_ref IS DISTINCT FROM v_consumo.decision_ref
       OR v_revalidacion.consumo_huella_sha256 IS DISTINCT FROM v_consumo.consumo_huella_sha256
       OR v_revalidacion.revalidada_en IS NULL THEN
        RAISE EXCEPTION 'contacto: revalidación incompleta' USING ERRCODE='42501';
    END IF;
    RETURN QUERY SELECT v_sujeto,v_nueva,v_recibo,v_consumo.auditoria_ref::text,v_consumo.consumo_huella_sha256::text;
END
$f$;

CREATE FUNCTION vec_contacto_usuario_v1.consultar_contacto_v1(
    p_accion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload_v3 bytea,p_sobre_cose bytea,
    p_evidencia bytea,p_raiz bytea,p_negocio bytea,p_recurso bytea,p_auditoria bytea
) RETURNS TABLE(version numeric,clave_ref text,nonce bytea,cifrado bytea,auditoria_central bytea,consumo_ref text,consumo_huella_sha256 text)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC'
AS $f$
DECLARE b jsonb; v_consumo record; v_contacto record; v_recibo bytea; v_revalidacion record;
BEGIN
    IF p_accion IS DISTINCT FROM 'vec.contacto_usuario.consultar'
       OR current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off'
       OR NOT pg_has_role(session_user,'vec_contacto_usuario_reader','MEMBER') THEN
        RAISE EXCEPTION 'contacto: consulta denegada' USING ERRCODE='42501';
    END IF;
    SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_consultar_contacto_usuario_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
    IF v_consumo.consumo_nuevo IS DISTINCT FROM true THEN
        RAISE EXCEPTION 'contacto: se requiere autorización fresca' USING ERRCODE='P1102';
    END IF;
    b:=convert_from(p_negocio,'UTF8')::jsonb;
    PERFORM set_config('vec.contacto.sujeto_ref',b->>'SujetoRef',true),set_config('vec.contacto.accion',p_accion,true),
        set_config('vec.contacto.version',b->>'Version',true),set_config('vec.contacto.version_anterior',b->>'Version',true);
    SELECT v.version,v.clave_ref,v.nonce,v.cifrado INTO STRICT v_contacto
        FROM vec_contacto_usuario_v1.actual a JOIN vec_contacto_usuario_v1.versiones v
        ON v.sujeto_ref=a.sujeto_ref AND v.version=a.version
        WHERE a.sujeto_ref=b->>'SujetoRef' AND a.version=(b->>'Version')::numeric FOR SHARE OF a;
    v_recibo:=convert_to(vec_bolsa_registro_accesos.registrar_contacto_usuario_v1(
        p_auditoria,p_negocio,p_recurso,p_decision,p_contexto,v_consumo.decision_ref,
        v_consumo.auditoria_ref,v_consumo.consumo_huella_sha256)::text,'UTF8');
    -- T13 y el bloqueo de fila pueden esperar: comprobar vigencia después.
    SELECT * INTO STRICT v_revalidacion FROM vec_autorizacion_atestada_v3.revalidar_consulta_contacto_usuario_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
    IF v_revalidacion.decision_ref IS NULL OR v_revalidacion.consumo_huella_sha256 IS NULL
       OR v_revalidacion.revalidada_en IS NULL THEN
        RAISE EXCEPTION 'contacto: revalidación incompleta' USING ERRCODE='42501';
    END IF;
    RETURN QUERY SELECT v_contacto.version::numeric,v_contacto.clave_ref::text,
        v_contacto.nonce::bytea,v_contacto.cifrado::bytea,v_recibo,
        v_consumo.auditoria_ref::text,v_consumo.consumo_huella_sha256::text;
EXCEPTION WHEN no_data_found THEN
    RAISE EXCEPTION 'contacto: versión no disponible' USING ERRCODE='P1103';
END
$f$;

CREATE FUNCTION vec_contacto_usuario_v1.revalidar_consulta_contacto_v1(
    p_accion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload_v3 bytea,p_sobre_cose bytea,
    p_evidencia bytea,p_raiz bytea,p_negocio bytea,p_recurso bytea,p_auditoria bytea
) RETURNS boolean
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC'
AS $f$
DECLARE b jsonb; v_revalidacion record; v_version numeric;
BEGIN
    IF p_accion IS DISTINCT FROM 'vec.contacto_usuario.consultar'
       OR current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off'
       OR NOT pg_has_role(session_user,'vec_contacto_usuario_reader','MEMBER') THEN
        RAISE EXCEPTION 'contacto: revalidación denegada' USING ERRCODE='42501';
    END IF;
    SELECT * INTO STRICT v_revalidacion FROM vec_autorizacion_atestada_v3.revalidar_consulta_contacto_usuario_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
    IF v_revalidacion.decision_ref IS NULL OR v_revalidacion.consumo_huella_sha256 IS NULL
       OR v_revalidacion.revalidada_en IS NULL THEN
        RAISE EXCEPTION 'contacto: revalidación incompleta' USING ERRCODE='42501';
    END IF;
    b:=convert_from(p_negocio,'UTF8')::jsonb;
    PERFORM set_config('vec.contacto.sujeto_ref',b->>'SujetoRef',true),set_config('vec.contacto.accion',p_accion,true),
        set_config('vec.contacto.version',b->>'Version',true),set_config('vec.contacto.version_anterior',b->>'Version',true);
    SELECT a.version INTO STRICT v_version FROM vec_contacto_usuario_v1.actual a
        WHERE a.sujeto_ref=b->>'SujetoRef' FOR SHARE;
    IF v_version IS DISTINCT FROM (b->>'Version')::numeric THEN
        RAISE EXCEPTION 'contacto: versión modificada' USING ERRCODE='P1103';
    END IF;
    -- Revalidar de nuevo después de cualquier espera del bloqueo compartido.
    SELECT * INTO STRICT v_revalidacion FROM vec_autorizacion_atestada_v3.revalidar_consulta_contacto_usuario_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
    IF v_revalidacion.decision_ref IS NULL OR v_revalidacion.consumo_huella_sha256 IS NULL
       OR v_revalidacion.revalidada_en IS NULL THEN
        RAISE EXCEPTION 'contacto: revalidación incompleta' USING ERRCODE='42501';
    END IF;
    RETURN true;
EXCEPTION WHEN no_data_found THEN
    RAISE EXCEPTION 'contacto: versión no disponible' USING ERRCODE='P1103';
END
$f$;

REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_contacto_usuario_v1 FROM PUBLIC;
-- Retira también concesiones procedentes de ALTER DEFAULT PRIVILEGES, sólo
-- sobre los objetos que acaba de crear esta migración.
DO $cerrar_acl$
DECLARE a record;
BEGIN
    FOR a IN SELECT DISTINCT x.grantee FROM pg_namespace n,
        LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) x
        WHERE n.nspname='vec_contacto_usuario_v1' AND x.grantee<>0 AND x.grantee<>n.nspowner LOOP
        EXECUTE format('REVOKE ALL ON SCHEMA vec_contacto_usuario_v1 FROM %I',pg_get_userbyid(a.grantee));
    END LOOP;
    FOR a IN SELECT c.oid::regclass AS objeto,x.grantee FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace,
        LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
        WHERE n.nspname='vec_contacto_usuario_v1' AND c.relkind='r' AND x.grantee<>0 AND x.grantee<>c.relowner LOOP
        EXECUTE format('REVOKE ALL ON TABLE %s FROM %I',a.objeto,pg_get_userbyid(a.grantee));
    END LOOP;
    FOR a IN SELECT p.oid::regprocedure AS objeto,x.grantee FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace,
        LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
        WHERE n.nspname='vec_contacto_usuario_v1' AND x.grantee<>0 AND x.grantee<>p.proowner LOOP
        EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',a.objeto,pg_get_userbyid(a.grantee));
    END LOOP;
END
$cerrar_acl$;
GRANT USAGE ON SCHEMA vec_contacto_usuario_v1 TO vec_contacto_usuario_writer,vec_contacto_usuario_reader;
GRANT EXECUTE ON FUNCTION vec_contacto_usuario_v1.registrar_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) TO vec_contacto_usuario_writer;
GRANT EXECUTE ON FUNCTION vec_contacto_usuario_v1.consultar_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) TO vec_contacto_usuario_reader;
GRANT EXECUTE ON FUNCTION vec_contacto_usuario_v1.revalidar_consulta_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) TO vec_contacto_usuario_reader;
DO $postcondicion$
DECLARE r text; t text; privilegio text;
BEGIN
    FOREACH r IN ARRAY ARRAY['vec_contacto_usuario_writer','vec_contacto_usuario_reader'] LOOP
        IF has_database_privilege(r,current_database(),'CREATE') OR has_schema_privilege(r,'vec_contacto_usuario_v1','CREATE') THEN
            RAISE EXCEPTION 'contacto: permiso CREATE excesivo' USING ERRCODE='42501';
        END IF;
        FOREACH t IN ARRAY ARRAY['versiones','actual','outbox'] LOOP
            FOREACH privilegio IN ARRAY ARRAY['SELECT','INSERT','UPDATE','DELETE','TRUNCATE','REFERENCES','TRIGGER'] LOOP
                IF has_table_privilege(r,format('vec_contacto_usuario_v1.%I',t),privilegio) THEN
                    RAISE EXCEPTION 'contacto: acceso directo a tablas' USING ERRCODE='42501';
                END IF;
            END LOOP;
        END LOOP;
    END LOOP;
    IF has_function_privilege('vec_contacto_usuario_reader','vec_contacto_usuario_v1.registrar_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE')
       OR has_function_privilege('vec_contacto_usuario_writer','vec_contacto_usuario_v1.consultar_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE')
       OR has_function_privilege('vec_contacto_usuario_writer','vec_contacto_usuario_v1.revalidar_consulta_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE') THEN
        RAISE EXCEPTION 'contacto: permiso cruzado de funciones' USING ERRCODE='42501';
    END IF;
END
$postcondicion$;
COMMIT;
