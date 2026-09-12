\set ON_ERROR_STOP on
-- AD3-35: fuente nominal de contacto propio VEC; no instala gobierno funcional.
-- Requiere roles/esquema contacto provisionados y núcleo AD3-32. Sin ADMIN.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000035',0));
DO $pre$
DECLARE r text; propietario oid:='vec_autorizacion_atestada_v3_propietario'::regrole;
BEGIN
    IF current_user<>'vec_autorizacion_atestada_v3_propietario' OR getdatabaseencoding()<>'UTF8'
       OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_autorizacion_atestada_v3' AND nspowner=propietario)
       OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_despacho_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
           AND proowner=propietario AND prosecdef AND provolatile='v' AND pronargdefaults=0
           AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']
           AND encode(sha256(convert_to(prosrc,'UTF8')),'hex')='d4da65fd5a27a6f22fa610e0465b099e98739d0c5eb338095b54efba8f3108f3')
       OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_resultado_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
           AND proowner=propietario AND prosecdef AND provolatile='v' AND pronargdefaults=0
           AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']
           AND encode(sha256(convert_to(prosrc,'UTF8')),'hex')='64dc16b857018d1fb877a6f77807cf0c067e415b4f102563a3ace9987885c30e')
       OR EXISTS (SELECT 1 FROM pg_proc WHERE pronamespace='vec_autorizacion_atestada_v3'::regnamespace AND proname LIKE 'contacto_%_v1') THEN
        RAISE EXCEPTION 'AD3-35: dependencias incompatibles' USING ERRCODE='55000';
    END IF;
    FOREACH r IN ARRAY ARRAY['vec_contacto_usuario_owner','vec_contacto_usuario_writer','vec_contacto_usuario_reader','vec_contacto_usuario_migrador'] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=r AND NOT rolcanlogin AND NOT rolinherit
            AND NOT rolsuper AND NOT rolbypassrls AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication)
           OR has_function_privilege(r,'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
            RAISE EXCEPTION 'AD3-35: roles de contacto incompatibles' USING ERRCODE='55000';
        END IF;
    END LOOP;
END $pre$;

CREATE FUNCTION vec_autorizacion_atestada_v3.contacto_objeto_v1(p_objeto jsonb,p_tipos jsonb)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog
AS $f$
    SELECT COALESCE(jsonb_typeof(p_objeto)='object' AND
        (SELECT array_agg(key ORDER BY key COLLATE "C") FROM jsonb_each(p_objeto))=
        (SELECT array_agg(key ORDER BY key COLLATE "C") FROM jsonb_each(p_tipos)) AND
        NOT EXISTS (SELECT 1 FROM jsonb_each_text(p_tipos) t
            WHERE jsonb_typeof(p_objeto->t.key) IS DISTINCT FROM t.value),false)
$f$;

-- Sólo codifica mapas de referencias opacas. No admite escalares, NULL o secretos.
CREATE FUNCTION vec_autorizacion_atestada_v3.contacto_mapa_canonico_v1(p_mapa jsonb)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog
AS $f$
DECLARE v text;
BEGIN
    IF jsonb_typeof(p_mapa) IS DISTINCT FROM 'object'
       OR (SELECT count(*) FROM jsonb_each(p_mapa)) NOT BETWEEN 1 AND 32
       OR EXISTS (SELECT 1 FROM jsonb_each(p_mapa) x WHERE jsonb_typeof(x.value)<>'string'
            OR x.key !~ '^[a-z][a-z0-9_]{0,63}$'
            OR x.value#>>'{}' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{0,159}$') THEN
        RAISE EXCEPTION 'AD3-35: contexto de recurso inválido' USING ERRCODE='22023';
    END IF;
    SELECT '{'||string_agg(vec_autorizacion_atestada_v3.texto_json_go(x.key)||':'||
        vec_autorizacion_atestada_v3.texto_json_go(x.value#>>'{}'),',' ORDER BY x.key COLLATE "C")||'}'
        INTO v FROM jsonb_each(p_mapa) x;
    RETURN v;
END $f$;

-- AuditEntry previo al efecto: campos exactos de encoding/json y AuthorizationRef ausente.
CREATE FUNCTION vec_autorizacion_atestada_v3.contacto_auditoria_previa_v1(p_auditoria bytea)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog
AS $f$
DECLARE a jsonb; c text; instante timestamptz; fecha text;
BEGIN
    IF p_auditoria IS NULL OR octet_length(p_auditoria) NOT BETWEEN 2 AND 16384 THEN
        RAISE EXCEPTION 'AD3-35: auditoría inválida' USING ERRCODE='22023';
    END IF;
    a:=convert_from(p_auditoria,'UTF8')::jsonb;
    IF vec_autorizacion_atestada_v3.contacto_objeto_v1(a,'{
        "id":"string","seq":"number","actor_id":"string","actor_profile":"string",
        "actor_roles":"array","auth_method":"string","auth_assurance":"string",
        "purpose":"string","action":"string","module_id":"string","subject_ref":"string",
        "object_version":"number","result":"string","correlation_ref":"string",
        "occurred_at":"string","signature":"string"}'::jsonb) IS NOT TRUE THEN
        RAISE EXCEPTION 'AD3-35: campos de auditoría inválidos' USING ERRCODE='22023';
    END IF;
    IF a->>'id'<>'' OR a->>'seq'<>'0' OR a->>'signature'<>''
       OR a->>'actor_id' !~ '^hmac-sha256:[a-z][a-z0-9._-]{0,63}:[0-9a-f]{64}$'
       OR split_part(a->>'actor_id',':',3)=repeat('0',64)
       OR a->>'actor_profile' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR jsonb_array_length(a->'actor_roles')<>1 OR jsonb_typeof(a#>'{actor_roles,0}') IS DISTINCT FROM 'string'
       OR a#>>'{actor_roles,0}' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR a->>'auth_method' NOT IN ('certificado','dnie') OR a->>'auth_assurance'<>'alto'
       OR a->>'purpose' NOT IN ('gestion_contacto_propio','envio_llamamiento')
       OR a->>'action' NOT IN ('vec.contacto_usuario.alta','vec.contacto_usuario.actualizar','vec.contacto_usuario.consultar')
       OR a->>'module_id'<>'vec.module.usuarios' OR a->>'subject_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
       OR a->>'object_version' !~ '^[1-9][0-9]{0,15}$' OR (a->>'object_version')::numeric>9007199254740991
       OR a->>'result'<>'accepted' OR a->>'correlation_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR a->>'occurred_at' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}([.][0-9]{1,6})?Z$' THEN
        RAISE EXCEPTION 'AD3-35: auditoría nominal inválida' USING ERRCODE='22023';
    END IF;
    instante:=(a->>'occurred_at')::timestamptz;
    fecha:=to_char(instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS')||
        CASE WHEN to_char(instante AT TIME ZONE 'UTC','US')='000000' THEN ''
            ELSE '.'||rtrim(to_char(instante AT TIME ZONE 'UTC','US'),'0') END||'Z';
    IF NOT isfinite(instante) OR fecha IS DISTINCT FROM a->>'occurred_at' THEN
        RAISE EXCEPTION 'AD3-35: instante inválido' USING ERRCODE='22023';
    END IF;
    c:='{"id":"","seq":0,"actor_id":'||vec_autorizacion_atestada_v3.texto_json_go(a->>'actor_id')||
        ',"actor_profile":'||vec_autorizacion_atestada_v3.texto_json_go(a->>'actor_profile')||
        ',"actor_roles":['||vec_autorizacion_atestada_v3.texto_json_go(a#>>'{actor_roles,0}')||
        '],"auth_method":'||vec_autorizacion_atestada_v3.texto_json_go(a->>'auth_method')||
        ',"auth_assurance":"alto","purpose":'||vec_autorizacion_atestada_v3.texto_json_go(a->>'purpose')||
        ',"action":'||vec_autorizacion_atestada_v3.texto_json_go(a->>'action')||
        ',"module_id":"vec.module.usuarios","subject_ref":'||vec_autorizacion_atestada_v3.texto_json_go(a->>'subject_ref')||
        ',"object_version":'||(a->>'object_version')||',"result":"accepted","correlation_ref":'||
        vec_autorizacion_atestada_v3.texto_json_go(a->>'correlation_ref')||',"occurred_at":'||
        vec_autorizacion_atestada_v3.texto_json_go(a->>'occurred_at')||',"signature":""}';
    IF convert_to(c,'UTF8') IS DISTINCT FROM p_auditoria THEN
        RAISE EXCEPTION 'AD3-35: auditoría no canónica' USING ERRCODE='22023';
    END IF;
    RETURN a;
EXCEPTION WHEN data_exception THEN
    RAISE EXCEPTION 'AD3-35: auditoría inválida' USING ERRCODE='22023';
END $f$;

-- Verificación pura compartida por consumidor y T13. No acredita autorización sola.
CREATE FUNCTION vec_autorizacion_atestada_v3.contacto_validar_material_v1(
    p_accion text,p_negocio bytea,p_recurso bytea,p_auditoria bytea,p_decision bytea,p_contexto bytea)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog
AS $f$
DECLARE b jsonb; r jsonb; d jsonb; x jsonb; a jsonb; s jsonb;
    canon text; rc text; audiencia text; finalidad text; anterior text; nueva text; sujeto text;
BEGIN
    IF p_accion IS NULL OR p_accion NOT IN ('vec.contacto_usuario.alta','vec.contacto_usuario.actualizar','vec.contacto_usuario.consultar')
       OR p_negocio IS NULL OR octet_length(p_negocio) NOT BETWEEN 2 AND 65536
       OR p_recurso IS NULL OR octet_length(p_recurso) NOT BETWEEN 2 AND 16384
       OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
       OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 2 AND 262144 THEN
        RAISE EXCEPTION 'AD3-35: material inválido' USING ERRCODE='22023';
    END IF;
    b:=convert_from(p_negocio,'UTF8')::jsonb; r:=convert_from(p_recurso,'UTF8')::jsonb;
    d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
    a:=vec_autorizacion_atestada_v3.contacto_auditoria_previa_v1(p_auditoria);
    IF vec_autorizacion_atestada_v3.contacto_objeto_v1(r,'{"ambitos":"object","atributos":"object"}'::jsonb) IS NOT TRUE THEN
        RAISE EXCEPTION 'AD3-35: recurso inválido' USING ERRCODE='22023';
    END IF;
    rc:='{"ambitos":'||vec_autorizacion_atestada_v3.contacto_mapa_canonico_v1(r->'ambitos')||
        ',"atributos":'||vec_autorizacion_atestada_v3.contacto_mapa_canonico_v1(r->'atributos')||'}';
    IF p_recurso IS DISTINCT FROM convert_to(rc,'UTF8') THEN
        RAISE EXCEPTION 'AD3-35: recurso no canónico' USING ERRCODE='22023';
    END IF;
    IF p_accion='vec.contacto_usuario.consultar' THEN
        audiencia:='vec.contacto_usuario.consulta.v1'; finalidad:='envio_llamamiento';
        IF vec_autorizacion_atestada_v3.contacto_objeto_v1(b,'{"Esquema":"string","SujetoRef":"string","FinalidadRef":"string","Audiencia":"string","Version":"number"}'::jsonb) IS NOT TRUE
           OR b->>'Esquema' IS DISTINCT FROM 'vec.contacto_usuario.consulta.v1'
           OR r#>>'{atributos,auditoria_sha256}' IS DISTINCT FROM encode(sha256(p_auditoria),'hex') THEN
            RAISE EXCEPTION 'AD3-35: consulta inválida' USING ERRCODE='22023';
        END IF;
        anterior:=b->>'Version'; nueva:=anterior;
        canon:='{"Esquema":"vec.contacto_usuario.consulta.v1","SujetoRef":'||
            vec_autorizacion_atestada_v3.texto_json_go(b->>'SujetoRef')||',"FinalidadRef":"envio_llamamiento",'||
            '"Audiencia":"vec.contacto_usuario.consulta.v1","Version":'||nueva||'}';
    ELSE
        audiencia:='vec.contacto_usuario.registro.v1'; finalidad:='gestion_contacto_propio';
        IF vec_autorizacion_atestada_v3.contacto_objeto_v1(b,'{"Esquema":"string","SujetoRef":"string","Audiencia":"string","FinalidadRef":"string","VersionEsperada":"number","VersionNueva":"number","Sobre":"object","Auditoria":"object"}'::jsonb) IS NOT TRUE
           OR b->>'Esquema' IS DISTINCT FROM 'vec.contacto_usuario.registro-cuerpo.v1'
           OR b->'Auditoria' IS DISTINCT FROM a THEN
            RAISE EXCEPTION 'AD3-35: registro inválido' USING ERRCODE='22023';
        END IF;
        anterior:=b->>'VersionEsperada'; nueva:=b->>'VersionNueva'; s:=b->'Sobre';
        IF vec_autorizacion_atestada_v3.contacto_objeto_v1(s,'{"Version":"number","ClaveRef":"string","Nonce":"string","Cifrado":"string"}'::jsonb) IS NOT TRUE
           OR s->>'Version' IS DISTINCT FROM nueva OR s->>'ClaveRef' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR octet_length(decode(s->>'Nonce','base64')) NOT BETWEEN 12 AND 64
           OR octet_length(decode(s->>'Cifrado','base64')) NOT BETWEEN 16 AND 32768
           OR replace(encode(decode(s->>'Nonce','base64'),'base64'),E'\n','') IS DISTINCT FROM s->>'Nonce'
           OR replace(encode(decode(s->>'Cifrado','base64'),'base64'),E'\n','') IS DISTINCT FROM s->>'Cifrado'
           OR anterior !~ '^(0|[1-9][0-9]{0,15})$'
           OR (p_accion='vec.contacto_usuario.alta' AND (anterior<>'0' OR nueva<>'1'))
           OR (p_accion='vec.contacto_usuario.actualizar' AND (anterior='0' OR anterior::numeric+1<>nueva::numeric)) THEN
            RAISE EXCEPTION 'AD3-35: sobre o versión inválidos' USING ERRCODE='22023';
        END IF;
        canon:='{"Esquema":"vec.contacto_usuario.registro-cuerpo.v1","SujetoRef":'||
            vec_autorizacion_atestada_v3.texto_json_go(b->>'SujetoRef')||
            ',"Audiencia":"vec.contacto_usuario.registro.v1","FinalidadRef":"gestion_contacto_propio","VersionEsperada":'||
            anterior||',"VersionNueva":'||nueva||',"Sobre":{"Version":'||nueva||',"ClaveRef":'||
            vec_autorizacion_atestada_v3.texto_json_go(s->>'ClaveRef')||',"Nonce":'||
            vec_autorizacion_atestada_v3.texto_json_go(s->>'Nonce')||',"Cifrado":'||
            vec_autorizacion_atestada_v3.texto_json_go(s->>'Cifrado')||'},"Auditoria":'||convert_from(p_auditoria,'UTF8')||'}';
        IF x->>'persona_ref' IS DISTINCT FROM b->>'SujetoRef' THEN
            RAISE EXCEPTION 'AD3-35: contacto propio ajeno' USING ERRCODE='42501';
        END IF;
    END IF;
    sujeto:=b->>'SujetoRef';
    IF sujeto IS NULL OR sujeto !~ '^per_[A-Za-z0-9_-]{22,128}$'
       OR nueva IS NULL OR nueva !~ '^[1-9][0-9]{0,15}$' OR nueva::numeric>9007199254740991
       OR b->>'Audiencia' IS DISTINCT FROM audiencia OR b->>'FinalidadRef' IS DISTINCT FROM finalidad
       OR p_negocio IS DISTINCT FROM convert_to(canon,'UTF8')
       OR r#>>'{atributos,contacto_sujeto_ref}' IS DISTINCT FROM sujeto
       OR r#>>'{atributos,contacto_finalidad_ref}' IS DISTINCT FROM finalidad
       OR r#>>'{atributos,contacto_version_esperada}' IS DISTINCT FROM anterior
       OR r#>>'{atributos,contacto_version}' IS DISTINCT FROM nueva
       OR r#>>'{atributos,material_sha256}' IS DISTINCT FROM encode(sha256(p_negocio),'hex')
       OR d->'concedida' IS DISTINCT FROM 'true'::jsonb OR d->>'codigo' IS DISTINCT FROM 'concedida'
       OR d->>'accion' IS DISTINCT FROM p_accion OR d->>'modulo_id' IS DISTINCT FROM 'vec.module.usuarios'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'contacto_usuario' OR d->>'recurso_ref' IS DISTINCT FROM sujeto
       OR d->>'finalidad' IS DISTINCT FROM finalidad
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM encode(sha256(p_recurso),'hex')
       OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
       OR a->>'action' IS DISTINCT FROM p_accion OR a->>'purpose' IS DISTINCT FROM finalidad
       OR a->>'subject_ref' IS DISTINCT FROM sujeto OR a->>'object_version' IS DISTINCT FROM nueva
       OR d->>'perfil_activo_ref' IS DISTINCT FROM a->>'actor_profile'
       OR d->>'version_rol_ref' IS DISTINCT FROM a#>>'{actor_roles,0}'
       OR d->>'correlacion_ref' IS DISTINCT FROM a->>'correlation_ref'
       OR d#>>'{vinculo_autenticacion_actor,metodo_observado}' IS DISTINCT FROM a->>'auth_method'
       OR d#>>'{vinculo_autenticacion_actor,garantia_observada}' IS DISTINCT FROM a->>'auth_assurance'
       OR x->>'principal_ref' IS NULL OR x->>'principal_ref' IS DISTINCT FROM d->>'principal_id'
       OR x->>'perfil_activo_ref' IS DISTINCT FROM a->>'actor_profile'
       OR x->>'metodo' IS DISTINCT FROM a->>'auth_method' OR x->>'garantia' IS DISTINCT FROM a->>'auth_assurance' THEN
        RAISE EXCEPTION 'AD3-35: material no ligado a decisión y contexto' USING ERRCODE='42501';
    END IF;
    RETURN b;
EXCEPTION WHEN data_exception THEN
    RAISE EXCEPTION 'AD3-35: material inválido' USING ERRCODE='22023';
END $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.contacto_sesion_nominal_v1(p_perfil text)
RETURNS boolean LANGUAGE plpgsql STABLE SET search_path=pg_catalog
AS $f$
DECLARE rol text;
BEGIN
    rol:=CASE p_perfil WHEN 'contacto_usuario_alta' THEN 'vec_contacto_usuario_writer'
        WHEN 'contacto_usuario_actualizar' THEN 'vec_contacto_usuario_writer'
        WHEN 'contacto_usuario_consultar' THEN 'vec_contacto_usuario_reader' END;
    RETURN COALESCE(rol IS NOT NULL
        AND current_setting('transaction_isolation')='serializable'
        AND current_setting('transaction_read_only')='off' AND current_setting('TimeZone')='UTC'
        AND current_user='vec_autorizacion_atestada_v3_propietario' AND session_user<>current_user
        AND EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolcanlogin
            AND NOT rolsuper AND NOT rolbypassrls AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication)
        AND (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)=1
        AND EXISTS (SELECT 1 FROM pg_auth_members WHERE member=session_user::regrole AND roleid=rol::regrole
            AND NOT admin_option AND inherit_option AND NOT set_option)
        AND NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE member=rol::regrole),false);
END $f$;

-- Ampliación nominal: la guarda anterior queda literal dentro del ELSE.
-- Una sesión CT/Personal/Bolsa jamás entra en los perfiles contacto.
DO $nucleo$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    definicion text; nueva text; metadata jsonb; dependencias jsonb;
    inicio integer; fin integer; guarda text;
    marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
    extension text:=$contactoperfiles$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_usuario_alta'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.contacto_usuario.registro.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'vec.contacto_usuario.alta'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'vec.contacto_usuario.alta'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'vec.module.usuarios'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'contacto_usuario'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestion_contacto_propio'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_usuario_actualizar'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.contacto_usuario.registro.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'vec.contacto_usuario.actualizar'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'vec.contacto_usuario.actualizar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'vec.module.usuarios'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'contacto_usuario'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestion_contacto_propio'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_usuario_consultar'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.contacto_usuario.consulta.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'vec.contacto_usuario.consultar'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'vec.contacto_usuario.consultar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'vec.module.usuarios'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'contacto_usuario'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'envio_llamamiento'
           )
$contactoperfiles$;
BEGIN
    SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc' INTO STRICT definicion,metadata
        FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
        AND p.prosecdef AND p.provolatile='v' AND p.pronargdefaults=0
        AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'];
    IF NOT COALESCE((SELECT count(*)=1 AND bool_and(x.grantee=p.proowner AND x.grantor=p.proowner
        AND x.privilege_type='EXECUTE' AND NOT x.is_grantable) FROM pg_proc p,
        LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f),false) THEN
        RAISE EXCEPTION 'AD3-35: ACL del núcleo divergente' USING ERRCODE='55000';
    END IF;
    inicio:=strpos(definicion,E'BEGIN\n    IF pg_catalog.current_setting(');
    fin:=strpos(definicion,E' THEN\n        RAISE EXCEPTION USING');
    IF inicio=0 OR fin<=inicio OR fin-inicio>40000
       OR strpos(definicion,'contacto_usuario')<>0
       OR strpos(definicion,'despacho_correo_llamamiento_ct')=0
       OR strpos(definicion,'resultado_correo_llamamiento_ct')=0
       OR strpos(definicion,'lectura_registro_personal_incorporacion_v2')=0
       OR length(definicion)-length(replace(definicion,marca,''))<>length(marca) THEN
        RAISE EXCEPTION 'AD3-35: núcleo posterior a AD3-32 incompatible' USING ERRCODE='55000';
    END IF;
    guarda:=substring(definicion FROM inicio+13 FOR fin-inicio-13);
    IF left(guarda,28)<>'pg_catalog.current_setting('''
       OR strpos(guarda,'session_user')=0 OR strpos(guarda,'vec_contratacion_temporal_ejecutor')=0 THEN
        RAISE EXCEPTION 'AD3-35: guarda previa incompatible' USING ERRCODE='55000';
    END IF;
    SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
        INTO dependencias FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
        OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f);
    nueva:=overlay(definicion placing
        E'/* AD3-35 GUARDA INICIO */ CASE WHEN p_perfil_mutacion IN (''contacto_usuario_alta'',''contacto_usuario_actualizar'',''contacto_usuario_consultar'')\n'
        ||E'        THEN vec_autorizacion_atestada_v3.contacto_sesion_nominal_v1(p_perfil_mutacion) IS NOT TRUE\n'
        ||E'        ELSE (/* AD3-35 GUARDA ORIGINAL */'||guarda||E') END /* AD3-35 GUARDA FIN */'
        FROM inicio+13 FOR fin-inicio-13);
    nueva:=replace(nueva,marca,extension||marca);
    EXECUTE nueva;
    IF pg_get_functiondef(f) IS DISTINCT FROM nueva
       OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata
       OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
           FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
           OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f)) IS DISTINCT FROM dependencias THEN
        RAISE EXCEPTION 'AD3-35: cambio ajeno a extensión nominal' USING ERRCODE='55000';
    END IF;
END $nucleo$;

-- Conserva literalmente todas las audiencias previas; sólo añade dos constantes.
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE def text; valores text[]; canon text;
BEGIN
    SELECT regexp_replace(pg_get_constraintdef(oid,true),'\s+',' ','g') INTO STRICT def FROM pg_constraint
        WHERE conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
        AND conname='clave_capacidad_version_audiencia_consumo_check' AND contype='c' AND convalidated AND conkey=ARRAY[8]::smallint[];
    SELECT array_agg(m[1] ORDER BY n) INTO valores FROM regexp_matches(def,'''([a-zA-Z0-9_.]+)''::text','g') WITH ORDINALITY x(m,n);
    canon:='CHECK (audiencia_consumo = ANY (ARRAY['||array_to_string(ARRAY(
        SELECT quote_literal(v)||'::text' FROM unnest(valores) WITH ORDINALITY x(v,n) ORDER BY n),', ')||']))';
    IF def IS DISTINCT FROM canon OR cardinality(valores) NOT BETWEEN 12 AND 64
       OR NOT (ARRAY['vec_contratacion_temporal.despacho_correo_llamamiento.v1','vec_contratacion_temporal.resultado_correo_llamamiento.v1']<@valores)
       OR ARRAY['vec.contacto_usuario.registro.v1','vec.contacto_usuario.consulta.v1']&&valores THEN
        RAISE EXCEPTION 'AD3-35: audiencias previas incompatibles' USING ERRCODE='55000';
    END IF;
    valores:=valores||ARRAY['vec.contacto_usuario.registro.v1','vec.contacto_usuario.consulta.v1'];
    ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
    EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check CHECK (audiencia_consumo IN ('||
        array_to_string(ARRAY(SELECT quote_literal(v) FROM unnest(valores) WITH ORDINALITY x(v,n) ORDER BY n),', ')||'))';
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_alta_contacto_usuario_v3_atestada(
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
    p_negocio bytea,p_recurso bytea,p_auditoria bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
    auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s'
AS $f$
DECLARE consumo record;
BEGIN
    IF p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
       OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL OR p_sobre IS NULL
       OR p_evidencia IS NULL OR p_raiz IS NULL
       OR vec_autorizacion_atestada_v3.contacto_sesion_nominal_v1('contacto_usuario_alta') IS NOT TRUE THEN
        RAISE EXCEPTION 'AD3-35: consumo contacto denegado' USING ERRCODE='42501';
    END IF;
    PERFORM vec_autorizacion_atestada_v3.contacto_validar_material_v1(
        'vec.contacto_usuario.alta',p_negocio,p_recurso,p_auditoria,p_decision,p_contexto);
    SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
        'contacto_usuario_alta',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload,p_sobre,p_evidencia,p_raiz);
    IF consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'AD3-35: contacto requiere concesión nueva' USING ERRCODE='P1102';
    END IF;
    RETURN QUERY SELECT consumo.decision_ref,consumo.efecto_ref,consumo.huella_efecto_sha256,
        consumo.consumo_huella_sha256,consumo.auditoria_ref,consumo.consumida_en,true;
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_actualizar_contacto_usuario_v3_atestada(
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
    p_negocio bytea,p_recurso bytea,p_auditoria bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
    auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s'
AS $f$
DECLARE consumo record;
BEGIN
    IF p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
       OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL OR p_sobre IS NULL
       OR p_evidencia IS NULL OR p_raiz IS NULL
       OR vec_autorizacion_atestada_v3.contacto_sesion_nominal_v1('contacto_usuario_actualizar') IS NOT TRUE THEN
        RAISE EXCEPTION 'AD3-35: consumo contacto denegado' USING ERRCODE='42501';
    END IF;
    PERFORM vec_autorizacion_atestada_v3.contacto_validar_material_v1(
        'vec.contacto_usuario.actualizar',p_negocio,p_recurso,p_auditoria,p_decision,p_contexto);
    SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
        'contacto_usuario_actualizar',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload,p_sobre,p_evidencia,p_raiz);
    IF consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'AD3-35: contacto requiere concesión nueva' USING ERRCODE='P1102';
    END IF;
    RETURN QUERY SELECT consumo.decision_ref,consumo.efecto_ref,consumo.huella_efecto_sha256,
        consumo.consumo_huella_sha256,consumo.auditoria_ref,consumo.consumida_en,true;
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consultar_contacto_usuario_v3_atestada(
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
    p_negocio bytea,p_recurso bytea,p_auditoria bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
    auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s'
AS $f$
DECLARE consumo record;
BEGIN
    IF p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
       OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL OR p_sobre IS NULL
       OR p_evidencia IS NULL OR p_raiz IS NULL
       OR vec_autorizacion_atestada_v3.contacto_sesion_nominal_v1('contacto_usuario_consultar') IS NOT TRUE THEN
        RAISE EXCEPTION 'AD3-35: consumo contacto denegado' USING ERRCODE='42501';
    END IF;
    PERFORM vec_autorizacion_atestada_v3.contacto_validar_material_v1(
        'vec.contacto_usuario.consultar',p_negocio,p_recurso,p_auditoria,p_decision,p_contexto);
    SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
        'contacto_usuario_consultar',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload,p_sobre,p_evidencia,p_raiz);
    IF consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'AD3-35: contacto requiere concesión nueva' USING ERRCODE='P1102';
    END IF;
    RETURN QUERY SELECT consumo.decision_ref,consumo.efecto_ref,consumo.huella_efecto_sha256,
        consumo.consumo_huella_sha256,consumo.auditoria_ref,consumo.consumida_en,true;
END $f$;

DO $acl$
DECLARE f regprocedure; a record;
BEGIN
    FOR f IN SELECT oid::regprocedure FROM pg_proc WHERE pronamespace='vec_autorizacion_atestada_v3'::regnamespace
        AND (proname LIKE 'contacto_%_v1' OR proname IN (
            'registrar_y_consumir_alta_contacto_usuario_v3_atestada','registrar_y_consumir_actualizar_contacto_usuario_v3_atestada',
            'registrar_y_consumir_consultar_contacto_usuario_v3_atestada','revalidar_consulta_contacto_usuario_v3_atestada')) LOOP
        EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f);
        FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
            WHERE p.oid=f AND x.grantee<>0 AND x.grantee<>p.proowner LOOP
            EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f,pg_get_userbyid(a.grantee));
        END LOOP;
    END LOOP;
END $acl$;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_contacto_usuario_owner;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_alta_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) TO vec_contacto_usuario_owner;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_actualizar_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) TO vec_contacto_usuario_owner;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consultar_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) TO vec_contacto_usuario_owner;

-- T13 ejecuta únicamente el codec puro a través de un wrapper SECURITY DEFINER.
CREATE FUNCTION vec_autorizacion_atestada_v3.contacto_material_auditoria_v1(
    p_accion text,p_negocio bytea,p_recurso bytea,p_auditoria bytea,p_decision bytea,p_contexto bytea)
RETURNS jsonb LANGUAGE sql IMMUTABLE SECURITY DEFINER SET search_path=pg_catalog
AS $f$ SELECT vec_autorizacion_atestada_v3.contacto_validar_material_v1($1,$2,$3,$4,$5,$6) $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.contacto_material_auditoria_v1(text,bytea,bytea,bytea,bytea,bytea) FROM PUBLIC;
DO $acl_codec$
DECLARE a record; f regprocedure:='vec_autorizacion_atestada_v3.contacto_material_auditoria_v1(text,bytea,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
    FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
        WHERE p.oid=f AND x.grantee<>0 AND x.grantee<>p.proowner LOOP
        EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f,pg_get_userbyid(a.grantee));
    END LOOP;
END $acl_codec$;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_accesos_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.contacto_material_auditoria_v1(text,bytea,bytea,bytea,bytea,bytea) TO vec_bolsa_accesos_propietario;

-- Reutiliza la revalidación central de AD3-5/21: consumo previo exacto,
-- gobierno, HMAC, raíz, revocación, contexto y política vivos; nunca replay temprano.
DO $revalidacion$
DECLARE f oid:='vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    def text; nueva text; metadata jsonb; deps jsonb; inicio integer; fin integer; guarda text;
BEGIN
    SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc' INTO STRICT def,metadata FROM pg_proc p
        WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
        AND p.provolatile='v' AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=1s']
        AND encode(sha256(convert_to(prosrc,'UTF8')),'hex')='7cfc002cff8878fc36288fa1200de1c51965e9ae4d84b4ffe6179bc62372b5ff';
    SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
        INTO deps FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
        OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f);
    inicio:=strpos(def,E'BEGIN\n    IF pg_catalog.current_setting(')+13;
    fin:=strpos(def,E' THEN\n        RAISE EXCEPTION USING');
    IF inicio=13 OR fin<=inicio THEN
        RAISE EXCEPTION 'AD3-35: revalidación previa incompatible' USING ERRCODE='55000';
    END IF;
    guarda:=substring(def FROM inicio FOR fin-inicio);
    nueva:=overlay(def placing $inicio$/* AD3-35 RV INICIO */ CASE WHEN p_perfil_consulta='contacto_usuario'
        THEN vec_autorizacion_atestada_v3.contacto_sesion_nominal_v1('contacto_usuario_consultar') IS NOT TRUE OR pg_catalog.pg_is_in_recovery()
        ELSE (/* AD3-35 RV ORIGINAL */$inicio$||guarda||') END /* AD3-35 RV FIN */' FROM inicio FOR fin-inicio);
    nueva:=replace(nueva,$antes$p_perfil_consulta NOT IN ('cuadro', 'detalle')$antes$,$despues$p_perfil_consulta NOT IN ('cuadro', 'detalle', 'contacto_usuario')$despues$);
    nueva:=replace(nueva,$antes$IF p_perfil_consulta = 'cuadro' THEN$antes$,$despues$IF p_perfil_consulta = 'contacto_usuario' THEN
        v_audiencia := 'vec.contacto_usuario.consulta.v1';
        v_operacion := 'vec.contacto_usuario.consultar';
        v_tipo_recurso := 'contacto_usuario';
        v_finalidad := 'envio_llamamiento';
    ELSIF p_perfil_consulta = 'cuadro' THEN$despues$);
    nueva:=replace(nueva,$antes$d ->> 'modulo_id' <> 'contratacion_temporal'$antes$,$despues$d ->> 'modulo_id' IS DISTINCT FROM CASE WHEN p_perfil_consulta='contacto_usuario'
           THEN 'vec.module.usuarios' ELSE 'contratacion_temporal' END$despues$);
    EXECUTE nueva;
    IF pg_get_functiondef(f) IS DISTINCT FROM nueva
       OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM '2f15b9ca8b64d453e79063fe706a5ac87ec3453e5c9451e071ecfc6bc30b2ff0'
       OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata
       OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
            FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
            OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f)) IS DISTINCT FROM deps THEN
        RAISE EXCEPTION 'AD3-35: revalidación modificada fuera del perfil nominal' USING ERRCODE='55000';
    END IF;
END $revalidacion$;

CREATE FUNCTION vec_autorizacion_atestada_v3.revalidar_consulta_contacto_usuario_v3_atestada(
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
    p_negocio bytea,p_recurso bytea,p_auditoria bytea)
RETURNS TABLE(decision_ref text,consumo_huella_sha256 text,revalidada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='1s'
AS $f$
BEGIN
    IF vec_autorizacion_atestada_v3.contacto_sesion_nominal_v1('contacto_usuario_consultar') IS NOT TRUE THEN
        RAISE EXCEPTION 'AD3-35: revalidación contacto denegada' USING ERRCODE='42501';
    END IF;
    PERFORM vec_autorizacion_atestada_v3.contacto_validar_material_v1(
        'vec.contacto_usuario.consultar',p_negocio,p_recurso,p_auditoria,p_decision,p_contexto);
    RETURN QUERY SELECT * FROM vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(
        'contacto_usuario',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload,p_sobre,p_evidencia,p_raiz);
END $f$;
DO $acl_revalidacion$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.revalidar_consulta_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)'::regprocedure; a record;
BEGIN
    EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f);
    FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
        WHERE p.oid=f AND x.grantee<>0 AND x.grantee<>p.proowner LOOP
        EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f,pg_get_userbyid(a.grantee));
    END LOOP;
END $acl_revalidacion$;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.revalidar_consulta_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) TO vec_contacto_usuario_owner;

DO $post$
DECLARE f record; esperado oid; a record;
BEGIN
    FOR f IN SELECT * FROM pg_proc WHERE pronamespace='vec_autorizacion_atestada_v3'::regnamespace AND
        (proname LIKE 'contacto_%_v1' OR proname IN ('registrar_y_consumir_alta_contacto_usuario_v3_atestada',
            'registrar_y_consumir_actualizar_contacto_usuario_v3_atestada','registrar_y_consumir_consultar_contacto_usuario_v3_atestada','revalidar_consulta_contacto_usuario_v3_atestada')) LOOP
        esperado:=CASE WHEN f.proname='contacto_material_auditoria_v1' THEN 'vec_bolsa_accesos_propietario'::regrole
            WHEN f.proname LIKE 'registrar_y_consumir_%' OR f.proname='revalidar_consulta_contacto_usuario_v3_atestada' THEN 'vec_contacto_usuario_owner'::regrole ELSE f.proowner END;
        IF f.proowner<>'vec_autorizacion_atestada_v3_propietario'::regrole
           OR NOT COALESCE((SELECT count(*)=CASE WHEN esperado=f.proowner THEN 1 ELSE 2 END
               AND count(DISTINCT x.grantee)=count(*)
               AND bool_and(x.grantee IN (f.proowner,esperado) AND x.grantor=f.proowner
                    AND x.privilege_type='EXECUTE' AND NOT x.is_grantable)
               FROM aclexplode(coalesce(f.proacl,acldefault('f',f.proowner))) x),false) THEN
            RAISE EXCEPTION 'AD3-35: ACL final divergente' USING ERRCODE='55000';
        END IF;
    END LOOP;
END $post$;
COMMIT;
