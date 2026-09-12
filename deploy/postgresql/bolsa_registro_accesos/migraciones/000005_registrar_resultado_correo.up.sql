\set ON_ERROR_STOP on
-- T13/5 candidata: fachada nominal CT sobre cadena/retención T13/1 existentes.
-- No crea tablas, no expone append genérico ni acredita exportación global.
BEGIN;
SET LOCAL ROLE vec_bolsa_accesos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_registro_accesos:migracion:000005',0));
DO $pre$
DECLARE propietario oid := 'vec_bolsa_accesos_propietario'::regrole;
    ct oid := 'vec_contratacion_temporal_propietario'::regrole; rol text;
BEGIN
    IF current_user <> 'vec_bolsa_accesos_propietario'
       OR getdatabaseencoding()<>'UTF8'
       OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_bolsa_registro_accesos' AND nspowner=propietario)
       OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=to_regprocedure('vec_bolsa_registro_accesos.registrar_interno_v1(jsonb)')
           AND proowner=propietario AND NOT prosecdef AND provolatile='v' AND proconfig=ARRAY['search_path=pg_catalog']
           AND encode(sha256(convert_to(prosrc,'UTF8')),'hex')='d5a61411e6f547eddb2f5cfc386366484929a17b1e68d568b1c1caa891b280d9')
       OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_resultado_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
           AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND prosecdef)
       OR NOT EXISTS (SELECT 1 FROM pg_class WHERE oid=to_regclass('vec_bolsa_registro_accesos.registro_acceso')
           AND relowner=propietario AND relrowsecurity AND relforcerowsecurity)
       OR EXISTS (SELECT 1 FROM pg_proc WHERE pronamespace='vec_bolsa_registro_accesos'::regnamespace
           AND proname IN ('resultado_correo_validar_auditoria_v1','registrar_resultado_correo_llamamiento_ct_v1'))
       OR (SELECT count(*) FROM pg_roles WHERE oid IN (propietario,ct)
           AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls AND NOT rolinherit
           AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication)<>2 THEN
        RAISE EXCEPTION 'T13/5: dependencias incompatibles' USING ERRCODE='55000';
    END IF;
    FOREACH rol IN ARRAY ARRAY['vec_contratacion_temporal_propietario','vec_contratacion_temporal_ejecutor','vec_contratacion_temporal_migrador'] LOOP
        IF has_function_privilege(rol,'vec_bolsa_registro_accesos.registrar_interno_v1(jsonb)','EXECUTE')
           OR has_function_privilege(rol,'vec_bolsa_registro_accesos.registrar_acceso_v1(jsonb)','EXECUTE')
           OR has_table_privilege(rol,'vec_bolsa_registro_accesos.registro_acceso','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') THEN
            RAISE EXCEPTION 'T13/5: privilegio CT genérico incompatible' USING ERRCODE='55000';
        END IF;
    END LOOP;
END $pre$;

-- Codec propietario puro: la procedencia del actor HMAC la acredita la frontera
-- común de preparación y su huella atestada V3. Un texto HMAC no concede acceso.
CREATE FUNCTION vec_bolsa_registro_accesos.resultado_correo_validar_auditoria_v1(
    p_auditoria bytea,p_solicitud text,p_decision bytea,p_contexto bytea,
    p_decision_ref text,p_consumo_ref text,p_consumo_huella text,p_recuperado boolean
) RETURNS jsonb LANGUAGE plpgsql IMMUTABLE
SET search_path=pg_catalog
AS $funcion$
DECLARE a jsonb; s jsonb; d jsonb; x jsonb; canonico text;
    h text; ha text; hc text; instante timestamptz;
BEGIN
    IF p_auditoria IS NULL OR octet_length(p_auditoria) NOT BETWEEN 2 AND 16384
       OR p_solicitud IS NULL OR octet_length(p_solicitud) NOT BETWEEN 2 AND 16384
       OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
       OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 2 AND 65536
       OR p_recuperado IS NULL THEN
        RAISE EXCEPTION 'T13/5: material de auditoría inválido' USING ERRCODE='22023';
    END IF;
    a:=convert_from(p_auditoria,'UTF8')::jsonb;
    s:=p_solicitud::jsonb;
    d:=convert_from(p_decision,'UTF8')::jsonb;
    x:=convert_from(p_contexto,'UTF8')::jsonb;
    IF vec_bolsa_registro_accesos.objeto_tipos_exactos_v1(a,'{
        "id":"string","seq":"number","signature":"string",
        "actor_id":"string","actor_profile":"string","actor_roles":"array",
        "auth_method":"string","auth_assurance":"string","purpose":"string",
        "action":"string","module_id":"string","subject_ref":"string",
        "object_version":"number","expediente_ref":"string","result":"string",
        "correlation_ref":"string","occurred_at":"string"}'::jsonb) IS NOT TRUE
       OR vec_bolsa_registro_accesos.objeto_tipos_exactos_v1(s,'{
        "OrganizacionRef":"string","ExpedienteRef":"string","LlamamientoRef":"string",
        "ComunicacionRef":"string","IntencionEnvioRef":"string","IntentoRef":"string",
        "SolicitudHuella":"string","Estado":"string","PlantillaRef":"string",
        "VersionEsperada":"number"}'::jsonb) IS NOT TRUE THEN
        RAISE EXCEPTION 'T13/5: campos de auditoría inválidos' USING ERRCODE='22023';
    END IF;
    IF a->>'id'<>'' OR a->>'seq'<>'0' OR a->>'signature'<>''
       OR a->>'actor_id' !~ '^hmac-sha256:[a-z][a-z0-9._-]{0,63}:[0-9a-f]{64}$'
       OR split_part(a->>'actor_id',':',3)=repeat('0',64)
       OR a->>'actor_profile' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR jsonb_array_length(a->'actor_roles')<>1
       OR jsonb_typeof(a#>'{actor_roles,0}') IS DISTINCT FROM 'string'
       OR a#>>'{actor_roles,0}' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR a->>'auth_method' NOT IN ('certificado','dnie') OR a->>'auth_assurance'<>'alto'
       OR a->>'purpose'<>'gestionar_contratacion_temporal'
       OR a->>'action'<>'contratacion_temporal.llamamiento.correo.registrar_resultado'
       OR a->>'module_id'<>'vec.module.contratacion_temporal'
       OR a->>'subject_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR a->>'expediente_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR a->>'correlation_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR a->>'object_version'<>'2' OR a->>'result'<>'accepted'
       OR a->>'occurred_at' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$'
       OR s->>'VersionEsperada'<>'1'
       OR s->>'Estado' NOT IN ('no_aceptado_transitorio','no_aceptado_permanente','indeterminado','aceptado_por_relay')
       OR s->>'PlantillaRef'<>'llamamiento_rrhh_v1'
       OR s->>'SolicitudHuella' !~ '^[0-9a-f]{64}$' OR s->>'SolicitudHuella'=repeat('0',64)
       OR EXISTS (SELECT 1 FROM jsonb_each(s) v WHERE v.key IN ('OrganizacionRef','ExpedienteRef',
            'LlamamientoRef','ComunicacionRef','IntencionEnvioRef','IntentoRef')
            AND v.value#>>'{}' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$') THEN
        RAISE EXCEPTION 'T13/5: auditoría nominal inválida' USING ERRCODE='22023';
    END IF;
    instante:=(a->>'occurred_at')::timestamptz;
    IF NOT isfinite(instante) OR to_char(instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')<>a->>'occurred_at' THEN
        RAISE EXCEPTION 'T13/5: instante de auditoría inválido' USING ERRCODE='22023';
    END IF;
    canonico:='{"id":"","seq":0,"signature":"","actor_id":"'||(a->>'actor_id')||
        '","actor_profile":"'||(a->>'actor_profile')||'","actor_roles":["'||(a#>>'{actor_roles,0}')||
        '"],"auth_method":"'||(a->>'auth_method')||'","auth_assurance":"alto",'||
        '"purpose":"gestionar_contratacion_temporal","action":"contratacion_temporal.llamamiento.correo.registrar_resultado",'||
        '"module_id":"vec.module.contratacion_temporal","subject_ref":"'||(a->>'subject_ref')||
        '","object_version":2,"expediente_ref":"'||(a->>'expediente_ref')||
        '","result":"accepted","correlation_ref":"'||(a->>'correlation_ref')||
        '","occurred_at":"'||(a->>'occurred_at')||'"}';
    IF p_auditoria IS DISTINCT FROM convert_to(canonico,'UTF8') THEN
        RAISE EXCEPTION 'T13/5: auditoría no canónica' USING ERRCODE='22023';
    END IF;
    h:=encode(sha256(convert_to(p_solicitud,'UTF8')),'hex');
    ha:=encode(sha256(p_auditoria),'hex');
    hc:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
        '"},"atributos":{"auditoria_sha256":"'||ha||'","material_sha256":"'||h||'"}}','UTF8')),'hex');
    IF p_decision_ref IS NULL OR p_decision_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_consumo_huella IS NULL OR p_consumo_huella !~ '^[0-9a-f]{64}$' OR p_consumo_huella=repeat('0',64)
       OR p_consumo_ref IS DISTINCT FROM 'aud_v3_'||substr(p_consumo_huella,1,32)
       OR d->>'decision_ref' IS DISTINCT FROM p_decision_ref
       OR d->'concedida' IS DISTINCT FROM 'true'::jsonb OR d->>'codigo' IS DISTINCT FROM 'concedida'
       OR d->>'accion' IS DISTINCT FROM a->>'action'
       OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'resultado_correo_llamamiento_contratacion_temporal'
       OR d->>'finalidad' IS DISTINCT FROM a->>'purpose'
       OR d->>'recurso_ref' IS DISTINCT FROM a->>'subject_ref'
       OR a->>'subject_ref' IS DISTINCT FROM s->>'IntentoRef'
       OR a->>'expediente_ref' IS DISTINCT FROM s->>'ExpedienteRef'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM hc
       OR d->>'perfil_activo_ref' IS DISTINCT FROM a->>'actor_profile'
       OR d->>'version_rol_ref' IS DISTINCT FROM a#>>'{actor_roles,0}'
       OR d->>'correlacion_ref' IS DISTINCT FROM a->>'correlation_ref'
       OR d#>>'{vinculo_autenticacion_actor,metodo_observado}' IS DISTINCT FROM a->>'auth_method'
       OR d#>>'{vinculo_autenticacion_actor,garantia_observada}' IS DISTINCT FROM a->>'auth_assurance'
       OR x->>'principal_ref' IS NULL OR x->>'principal_ref' IS DISTINCT FROM d->>'principal_id'
       OR x->>'perfil_activo_ref' IS DISTINCT FROM a->>'actor_profile'
       OR x->>'metodo' IS DISTINCT FROM a->>'auth_method'
       OR x->>'garantia' IS DISTINCT FROM a->>'auth_assurance' THEN
        RAISE EXCEPTION 'T13/5: auditoría no ligada a consumo y contexto' USING ERRCODE='42501';
    END IF;
    RETURN jsonb_build_object(
        'actor_id',a->>'actor_id','actor_profile',a->>'actor_profile','actor_roles',a->'actor_roles',
        'represented_subject_id','','auth_method',a->>'auth_method','auth_assurance',a->>'auth_assurance',
        'authorization_ref',p_decision_ref,'purpose',a->>'purpose','action',a->>'action',
        'module_id',a->>'module_id','subject_ref',a->>'subject_ref','object_version',2,
        'expediente_ref',a->>'expediente_ref','document_ref','','rule_ref','','reason','',
        'result','permitido','before_hash','','after_hash',h,'correlation_ref',a->>'correlation_ref',
        'metadata',jsonb_build_object('consumo_ref',p_consumo_ref,'consumo_huella_sha256',p_consumo_huella,
            'resultado_sha256',h,'estado_smtp',s->>'Estado','recuperado',p_recuperado::text),
        'occurred_at',a->>'occurred_at');
EXCEPTION WHEN data_exception THEN
    RAISE EXCEPTION 'T13/5: material de auditoría inválido' USING ERRCODE='22023';
END $funcion$;

-- Único llamador externo: propietario CT NOLOGIN. Los tres parámetros consumo
-- proceden del record AD3 NUEVO, nunca de runtime; CT ya validó secreto, JSON10,
-- hash contexto y versión. No se lee ningún almacén de CT/AD3 desde T13.
-- El mismo llamador preserva la TX hasta estado+hecho+outbox; cualquier error
-- revierte también esta entrada T13. No es una concesión reutilizable autónoma.
CREATE FUNCTION vec_bolsa_registro_accesos.registrar_resultado_correo_llamamiento_ct_v1(
    p_auditoria bytea,p_solicitud text,p_decision bytea,p_contexto bytea,
    p_decision_ref text,p_consumo_ref text,p_consumo_huella text,p_recuperado boolean,
    p_auditoria_original_ref text
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' SET row_security=on
AS $funcion$
DECLARE entrada jsonb; recibo jsonb;
BEGIN
    IF current_user<>'vec_bolsa_accesos_propietario' OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
       OR pg_has_role(session_user,'vec_bolsa_accesos_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_bolsa_accesos_migrador','MEMBER')
       OR current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'off' THEN
        RAISE EXCEPTION 'T13/5: registro de resultado denegado' USING ERRCODE='42501';
    END IF;
    entrada:=vec_bolsa_registro_accesos.resultado_correo_validar_auditoria_v1(
        p_auditoria,p_solicitud,p_decision,p_contexto,p_decision_ref,p_consumo_ref,p_consumo_huella,p_recuperado);
    -- La recuperación tampoco puede ocultar una auditoría original ausente.
    -- Sólo la autoridad T13 consulta su propio almacén; CT no lo lee directamente.
    IF p_recuperado THEN
        IF p_auditoria_original_ref IS NULL OR p_auditoria_original_ref !~ '^acc_[0-9a-f]{40}$'
           OR NOT EXISTS (SELECT 1 FROM vec_bolsa_registro_accesos.registro_acceso r
                WHERE r.registro_ref=p_auditoria_original_ref
                  AND r.module_id=entrada->>'module_id' AND r.action=entrada->>'action'
                  AND r.subject_ref=entrada->>'subject_ref' AND r.expediente_ref=entrada->>'expediente_ref'
                  AND r.object_version=2 AND r.result='permitido'
                  AND r.after_hash=entrada->>'after_hash'
                  AND r.metadata->>'estado_smtp'=entrada#>>'{metadata,estado_smtp}'
                  AND r.metadata->>'recuperado'='false') THEN
            RAISE EXCEPTION 'T13/5: auditoría original ausente o divergente' USING ERRCODE='55000';
        END IF;
    ELSIF p_auditoria_original_ref IS NOT NULL THEN
        RAISE EXCEPTION 'T13/5: referencia histórica incompatible' USING ERRCODE='22023';
    END IF;
    recibo:=vec_bolsa_registro_accesos.registrar_interno_v1(entrada);
    IF recibo->>'id' IS NULL OR recibo->>'id' !~ '^acc_[0-9a-f]{40}$'
       OR recibo->>'signature' IS NULL OR recibo->>'signature' !~ '^[0-9a-f]{64}$'
       OR recibo->>'authorization_ref' IS DISTINCT FROM p_decision_ref
       OR recibo->>'subject_ref' IS DISTINCT FROM entrada->>'subject_ref'
       OR recibo->>'after_hash' IS DISTINCT FROM entrada->>'after_hash'
       OR recibo->'metadata' IS DISTINCT FROM entrada->'metadata' THEN
        RAISE EXCEPTION 'T13/5: recibo de autoridad divergente' USING ERRCODE='55000';
    END IF;
    RETURN recibo;
END $funcion$;

DO $acl$
DECLARE f regprocedure; a record;
BEGIN
    FOREACH f IN ARRAY ARRAY[
        'vec_bolsa_registro_accesos.resultado_correo_validar_auditoria_v1(bytea,text,bytea,bytea,text,text,text,boolean)'::regprocedure,
        'vec_bolsa_registro_accesos.registrar_resultado_correo_llamamiento_ct_v1(bytea,text,bytea,bytea,text,text,text,boolean,text)'::regprocedure
    ] LOOP
        EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f);
        FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,
            LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
            WHERE p.oid=f AND x.grantee<>0 AND x.grantee<>p.proowner LOOP
            EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f,pg_get_userbyid(a.grantee));
        END LOOP;
    END LOOP;
END $acl$;
GRANT USAGE ON SCHEMA vec_bolsa_registro_accesos TO vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION vec_bolsa_registro_accesos.registrar_resultado_correo_llamamiento_ct_v1(bytea,text,bytea,bytea,text,text,text,boolean,text) TO vec_contratacion_temporal_propietario;
DO $post$
DECLARE f oid:='vec_bolsa_registro_accesos.registrar_resultado_correo_llamamiento_ct_v1(bytea,text,bytea,bytea,text,text,text,boolean,text)'::regprocedure;
    propietario oid:='vec_bolsa_accesos_propietario'::regrole; ct oid:='vec_contratacion_temporal_propietario'::regrole;
BEGIN
    IF NOT COALESCE((SELECT count(*)=2 AND count(DISTINCT a.grantee)=2
        AND bool_and(a.grantee IN(propietario,ct) AND a.grantor=propietario AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
        FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f),false) THEN
        RAISE EXCEPTION 'T13/5: ACL nominal divergente' USING ERRCODE='55000';
    END IF;
END $post$;
COMMIT;
