\set ON_ERROR_STOP on
-- Guardado V2: intención propia estable y HMAC de petición; no persiste correo claro.
-- El mismo intento requiere la misma identidad KMS; rotación no implementada.
BEGIN;
SET LOCAL ROLE vec_contacto_usuario_owner;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:migracion:3',0));
DO $pre$
DECLARE t text;
BEGIN
    IF getdatabaseencoding()<>'UTF8'
       OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_contacto_usuario_v1' AND nspowner=current_user::regrole)
       OR to_regclass('vec_contacto_usuario_v1.intenciones') IS NOT NULL
       OR EXISTS (SELECT 1 FROM pg_proc WHERE pronamespace='vec_contacto_usuario_v1'::regnamespace AND proname='registrar_contacto_v2')
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_alta_contacto_usuario_v3_atestada_v2(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_actualizar_contacto_usuario_v3_atestada_v2(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.revalidar_alta_contacto_usuario_v3_atestada_v2(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.revalidar_actualizar_contacto_usuario_v3_atestada_v2(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_bolsa_registro_accesos.registrar_intento_contacto_usuario_v2(bytea,bytea,bytea,bytea,bytea,text,text,text,bytea)') IS NULL THEN
        RAISE EXCEPTION 'contacto V2: dependencias incompatibles' USING ERRCODE='55000';
    END IF;
    FOREACH t IN ARRAY ARRAY['versiones','actual','outbox'] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_class WHERE oid=to_regclass('vec_contacto_usuario_v1.'||t)
            AND relowner=current_user::regrole AND relrowsecurity AND relforcerowsecurity) THEN
            RAISE EXCEPTION 'contacto V2: almacenamiento incompatible' USING ERRCODE='55000';
        END IF;
    END LOOP;
END $pre$;
CREATE TABLE vec_contacto_usuario_v1.intenciones (
    sujeto_ref text NOT NULL CHECK(sujeto_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
    intent_ref uuid NOT NULL CHECK(intent_ref::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    version_esperada numeric(20,0) NOT NULL CHECK(version_esperada BETWEEN 0 AND 9007199254740990),
    version_nueva numeric(20,0) NOT NULL CHECK(version_nueva=version_esperada+1),
    huella_peticion text NOT NULL CHECK(huella_peticion ~ '^[0-9a-f]{64}$' AND huella_peticion<>repeat('0',64)),
    PRIMARY KEY(sujeto_ref,intent_ref),
    UNIQUE(sujeto_ref,version_nueva),
    FOREIGN KEY(sujeto_ref,version_nueva) REFERENCES vec_contacto_usuario_v1.versiones(sujeto_ref,version)
);
REVOKE ALL ON TABLE vec_contacto_usuario_v1.intenciones FROM PUBLIC;
ALTER TABLE vec_contacto_usuario_v1.intenciones ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contacto_usuario_v1.intenciones FORCE ROW LEVEL SECURITY;
CREATE POLICY lectura ON vec_contacto_usuario_v1.intenciones FOR SELECT TO vec_contacto_usuario_owner USING(
    sujeto_ref=current_setting('vec.contacto.sujeto_ref',true)
    AND intent_ref::text=current_setting('vec.contacto.intent_ref',true)
    AND current_setting('vec.contacto.accion',true) IN ('vec.contacto_usuario.alta','vec.contacto_usuario.actualizar'));
CREATE POLICY alta ON vec_contacto_usuario_v1.intenciones FOR INSERT TO vec_contacto_usuario_owner WITH CHECK(
    sujeto_ref=current_setting('vec.contacto.sujeto_ref',true)
    AND intent_ref::text=current_setting('vec.contacto.intent_ref',true)
    AND version_nueva::text=current_setting('vec.contacto.version',true)
    AND version_esperada::text=current_setting('vec.contacto.version_anterior',true)
    AND huella_peticion=current_setting('vec.contacto.huella_peticion',true)
    AND current_setting('vec.contacto.accion',true) IN ('vec.contacto_usuario.alta','vec.contacto_usuario.actualizar'));
-- Permiso histórico mínimo para recuperar la versión enlazada con esa intención.
CREATE POLICY lectura_intencion ON vec_contacto_usuario_v1.versiones FOR SELECT TO vec_contacto_usuario_owner USING(
    sujeto_ref=current_setting('vec.contacto.sujeto_ref',true)
    AND version::text=current_setting('vec.contacto.version',true)
    AND current_setting('vec.contacto.accion',true) IN ('vec.contacto_usuario.alta','vec.contacto_usuario.actualizar')
    AND EXISTS (SELECT 1 FROM vec_contacto_usuario_v1.intenciones i
        WHERE i.sujeto_ref=versiones.sujeto_ref AND i.version_nueva=versiones.version
            AND i.intent_ref::text=current_setting('vec.contacto.intent_ref',true)));

CREATE FUNCTION vec_contacto_usuario_v1.registrar_contacto_v2(
    p_accion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload_v3 bytea,p_sobre_cose bytea,
    p_evidencia bytea,p_raiz bytea,p_negocio bytea,p_recurso bytea,p_auditoria bytea)
RETURNS TABLE(recuperado boolean,sujeto_ref text,version numeric,recibo_original bytea,
    consumo_original_ref text,consumo_original_huella_sha256 text,auditoria_intento bytea,
    consumo_intento_ref text,consumo_intento_huella_sha256 text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s'
AS $f$
DECLARE b jsonb; o jsonb; v_consumo record; v_revalidacion record; v_intencion record; v_original record;
    v_sujeto text; v_intent uuid; v_huella text; v_anterior numeric; v_nueva numeric; v_actual numeric;
    v_recuperado boolean; v_recibo bytea; v_audit bytea; v_ref text; v_hash text;
BEGIN
    IF p_accion IS NULL OR p_accion NOT IN ('vec.contacto_usuario.alta','vec.contacto_usuario.actualizar')
       OR current_user<>'vec_contacto_usuario_owner' OR current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
       OR NOT pg_has_role(session_user,'vec_contacto_usuario_writer','MEMBER') THEN
        RAISE EXCEPTION 'contacto V2: guardado denegado' USING ERRCODE='42501';
    END IF;
    IF p_accion='vec.contacto_usuario.alta' THEN
        SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_alta_contacto_usuario_v3_atestada_v2(
            p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
            p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
    ELSE
        SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_actualizar_contacto_usuario_v3_atestada_v2(
            p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
            p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
    END IF;
    IF v_consumo.consumo_nuevo IS DISTINCT FROM true THEN
        RAISE EXCEPTION 'contacto V2: autorización nueva requerida' USING ERRCODE='P1102';
    END IF;
    b:=convert_from(p_negocio,'UTF8')::jsonb;
    v_sujeto:=b->>'SujetoRef'; v_intent:=(b->>'IntentRef')::uuid; v_huella:=b->>'HuellaPeticion';
    v_anterior:=(b->>'VersionEsperada')::numeric; v_nueva:=(b->>'VersionNueva')::numeric;
    IF v_sujeto IS NULL OR v_sujeto IS DISTINCT FROM convert_from(p_contexto,'UTF8')::jsonb->>'persona_ref' THEN
        RAISE EXCEPTION 'contacto V2: sujeto ajeno' USING ERRCODE='42501';
    END IF;
    PERFORM set_config('vec.contacto.sujeto_ref',v_sujeto,true),set_config('vec.contacto.accion',p_accion,true),
        set_config('vec.contacto.version',v_nueva::text,true),set_config('vec.contacto.version_anterior',v_anterior::text,true),
        set_config('vec.contacto.intent_ref',v_intent::text,true),set_config('vec.contacto.huella_peticion',v_huella,true);
    -- Misma intención serializada; tras una espera SERIALIZABLE puede exigir reintento fresco.
    PERFORM pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:intent:'||v_sujeto||':'||v_intent::text,0));
    SELECT i.version_esperada,i.version_nueva,i.huella_peticion INTO v_intencion
        FROM vec_contacto_usuario_v1.intenciones i WHERE i.sujeto_ref=v_sujeto AND i.intent_ref=v_intent;
    v_recuperado:=FOUND;
    IF v_recuperado THEN
        IF v_intencion.version_esperada IS DISTINCT FROM v_anterior OR v_intencion.version_nueva IS DISTINCT FROM v_nueva
           OR v_intencion.huella_peticion IS DISTINCT FROM v_huella THEN
            RAISE EXCEPTION 'contacto V2: intención divergente' USING ERRCODE='P1104';
        END IF;
        SELECT v.auditoria_central,v.decision_ref,v.consumo_ref,v.consumo_huella_sha256,
            encode(sha256(v.negocio),'hex') AS material_sha256,
            encode(sha256(v.contexto_recurso),'hex') AS contexto_recurso_sha256
            INTO STRICT v_original FROM vec_contacto_usuario_v1.versiones v
            WHERE v.sujeto_ref=v_sujeto AND v.version=v_nueva;
        o:=convert_from(v_original.auditoria_central,'UTF8')::jsonb;
        IF o->>'authorization_ref' IS DISTINCT FROM v_original.decision_ref
           OR o->>'subject_ref' IS DISTINCT FROM v_sujeto OR (o->>'object_version')::numeric IS DISTINCT FROM v_nueva
           OR o->>'after_hash' IS DISTINCT FROM v_original.material_sha256
           OR o->'metadata' IS DISTINCT FROM jsonb_build_object('consumo_ref',v_original.consumo_ref,
                'consumo_huella_sha256',v_original.consumo_huella_sha256,'material_sha256',v_original.material_sha256,
                'contexto_recurso_sha256',v_original.contexto_recurso_sha256) THEN
            RAISE EXCEPTION 'contacto V2: evidencia histórica divergente' USING ERRCODE='55000';
        END IF;
        v_recibo:=v_original.auditoria_central; v_ref:=v_original.consumo_ref; v_hash:=v_original.consumo_huella_sha256;
        v_audit:=convert_to(vec_bolsa_registro_accesos.registrar_intento_contacto_usuario_v2(
            p_auditoria,p_negocio,p_recurso,p_decision,p_contexto,v_consumo.decision_ref,
            v_consumo.auditoria_ref,v_consumo.consumo_huella_sha256,v_recibo)::text,'UTF8');
    ELSE
        SELECT a.version INTO v_actual FROM vec_contacto_usuario_v1.actual a WHERE a.sujeto_ref=v_sujeto FOR UPDATE;
        IF coalesce(v_actual,0)<>v_anterior THEN
            RAISE EXCEPTION 'contacto V2: conflicto de versión' USING ERRCODE='P1103';
        END IF;
        v_recibo:=convert_to(vec_bolsa_registro_accesos.registrar_intento_contacto_usuario_v2(
            p_auditoria,p_negocio,p_recurso,p_decision,p_contexto,v_consumo.decision_ref,
            v_consumo.auditoria_ref,v_consumo.consumo_huella_sha256,''::bytea)::text,'UTF8');
        v_audit:=v_recibo; v_ref:=v_consumo.auditoria_ref; v_hash:=v_consumo.consumo_huella_sha256;
        INSERT INTO vec_contacto_usuario_v1.versiones(
            sujeto_ref,version,clave_ref,nonce,cifrado,negocio,contexto_recurso,decision_ref,
            consumo_ref,consumo_huella_sha256,auditoria_central)
            VALUES(v_sujeto,v_nueva,b#>>'{Sobre,ClaveRef}',decode(b#>>'{Sobre,Nonce}','base64'),
                decode(b#>>'{Sobre,Cifrado}','base64'),p_negocio,p_recurso,v_consumo.decision_ref,v_ref,v_hash,v_recibo);
        INSERT INTO vec_contacto_usuario_v1.actual AS a(sujeto_ref,version) VALUES(v_sujeto,v_nueva)
            ON CONFLICT ON CONSTRAINT actual_pkey DO UPDATE SET version=excluded.version;
        INSERT INTO vec_contacto_usuario_v1.outbox(consumo_ref,sujeto_ref,version,accion)
            VALUES(v_ref,v_sujeto,v_nueva,p_accion);
        INSERT INTO vec_contacto_usuario_v1.intenciones(sujeto_ref,intent_ref,version_esperada,version_nueva,huella_peticion)
            VALUES(v_sujeto,v_intent,v_anterior,v_nueva,v_huella);
    END IF;
    -- Revalidar el consumo de ESTE intento tras T13, locks y toda escritura.
    IF p_accion='vec.contacto_usuario.alta' THEN
        SELECT * INTO STRICT v_revalidacion FROM vec_autorizacion_atestada_v3.revalidar_alta_contacto_usuario_v3_atestada_v2(
            p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
            p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
    ELSE
        SELECT * INTO STRICT v_revalidacion FROM vec_autorizacion_atestada_v3.revalidar_actualizar_contacto_usuario_v3_atestada_v2(
            p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
            p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
    END IF;
    IF v_revalidacion.decision_ref IS DISTINCT FROM v_consumo.decision_ref
       OR v_revalidacion.consumo_huella_sha256 IS DISTINCT FROM v_consumo.consumo_huella_sha256
       OR v_revalidacion.revalidada_en IS NULL THEN
        RAISE EXCEPTION 'contacto V2: revalidación divergente' USING ERRCODE='42501';
    END IF;
    RETURN QUERY SELECT v_recuperado,v_sujeto,v_nueva,v_recibo,v_ref,v_hash,
        v_audit,v_consumo.auditoria_ref::text,v_consumo.consumo_huella_sha256::text;
END $f$;
DO $acl$
DECLARE f regprocedure:='vec_contacto_usuario_v1.registrar_contacto_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)'::regprocedure; a record;
BEGIN
    EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f);
    FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
        WHERE p.oid=f AND x.grantee<>0 AND x.grantee<>p.proowner LOOP
        EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f,pg_get_userbyid(a.grantee));
    END LOOP;
    FOR a IN SELECT DISTINCT x.grantee FROM pg_class c,LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
        WHERE c.oid='vec_contacto_usuario_v1.intenciones'::regclass AND x.grantee<>0 AND x.grantee<>c.relowner LOOP
        EXECUTE format('REVOKE ALL ON TABLE vec_contacto_usuario_v1.intenciones FROM %I',pg_get_userbyid(a.grantee));
    END LOOP;
END $acl$;
GRANT EXECUTE ON FUNCTION vec_contacto_usuario_v1.registrar_contacto_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) TO vec_contacto_usuario_writer;
DO $post$
DECLARE f oid:='vec_contacto_usuario_v1.registrar_contacto_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)'::regprocedure;
    propietario oid:='vec_contacto_usuario_owner'::regrole; escritor oid:='vec_contacto_usuario_writer'::regrole; r text;
BEGIN
    IF NOT COALESCE((SELECT count(*)=2 AND count(DISTINCT x.grantee)=2
        AND bool_and(x.grantee IN (propietario,escritor) AND x.grantor=propietario
            AND x.privilege_type='EXECUTE' AND NOT x.is_grantable)
        FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f),false)
       OR has_function_privilege('vec_contacto_usuario_reader',f,'EXECUTE') THEN
        RAISE EXCEPTION 'contacto V2: ACL de fachada divergente' USING ERRCODE='55000';
    END IF;
    FOREACH r IN ARRAY ARRAY['vec_contacto_usuario_writer','vec_contacto_usuario_reader'] LOOP
        IF has_table_privilege(r,'vec_contacto_usuario_v1.intenciones','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') THEN
            RAISE EXCEPTION 'contacto V2: privilegios directos a intenciones' USING ERRCODE='55000';
        END IF;
    END LOOP;
END $post$;
COMMIT;
