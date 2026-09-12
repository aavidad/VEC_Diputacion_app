\set ON_ERROR_STOP on
-- Fuente candidata: consulta del recibo histórico propio; no acredita replay de Guardar.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000036',0));
DO $pre$
BEGIN
    IF getdatabaseencoding()<>'UTF8'
       OR to_regprocedure('vec_autorizacion_atestada_v3.contacto_validar_material_v1(text,bytea,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.contacto_sesion_nominal_v1(text)') IS NULL
       OR EXISTS (SELECT 1 FROM pg_proc WHERE pronamespace='vec_autorizacion_atestada_v3'::regnamespace
            AND proname IN ('contacto_recibo_validar_material_v1','contacto_recibo_material_auditoria_v1',
                'registrar_y_consumir_recibo_contacto_usuario_v3_atestada','revalidar_recibo_contacto_usuario_v3_atestada')) THEN
        RAISE EXCEPTION 'AD3-36: preimagen incompatible' USING ERRCODE='55000';
    END IF;
END $pre$;
CREATE FUNCTION vec_autorizacion_atestada_v3.contacto_recibo_validar_material_v1(
    p_accion text,p_negocio bytea,p_recurso bytea,p_auditoria bytea,p_decision bytea,p_contexto bytea)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog
AS $f$
DECLARE b jsonb; r jsonb; d jsonb; x jsonb; a jsonb;
    canon text; rc text; audiencia text; finalidad text; anterior text; nueva text; sujeto text;
BEGIN
    IF p_accion IS DISTINCT FROM 'vec.contacto_usuario.consultar'
       OR p_negocio IS NULL OR octet_length(p_negocio) NOT BETWEEN 2 AND 65536
       OR p_recurso IS NULL OR octet_length(p_recurso) NOT BETWEEN 2 AND 16384
       OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
       OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 2 AND 262144 THEN
        RAISE EXCEPTION 'AD3-36: material inválido' USING ERRCODE='22023';
    END IF;
    b:=convert_from(p_negocio,'UTF8')::jsonb; r:=convert_from(p_recurso,'UTF8')::jsonb;
    d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
    a:=vec_autorizacion_atestada_v3.contacto_auditoria_previa_v1(p_auditoria);
    IF vec_autorizacion_atestada_v3.contacto_objeto_v1(r,'{"ambitos":"object","atributos":"object"}'::jsonb) IS NOT TRUE THEN
        RAISE EXCEPTION 'AD3-36: recurso inválido' USING ERRCODE='22023';
    END IF;
    rc:='{"ambitos":'||vec_autorizacion_atestada_v3.contacto_mapa_canonico_v1(r->'ambitos')||
        ',"atributos":'||vec_autorizacion_atestada_v3.contacto_mapa_canonico_v1(r->'atributos')||'}';
    IF p_recurso IS DISTINCT FROM convert_to(rc,'UTF8') THEN
        RAISE EXCEPTION 'AD3-36: recurso no canónico' USING ERRCODE='22023';
    END IF;
    audiencia:='vec.contacto_usuario.recibo.v1'; finalidad:='gestion_contacto_propio';
    IF vec_autorizacion_atestada_v3.contacto_objeto_v1(b,'{"Esquema":"string","SujetoRef":"string","FinalidadRef":"string","Audiencia":"string","Version":"number"}'::jsonb) IS NOT TRUE
       OR b->>'Esquema' IS DISTINCT FROM 'vec.contacto_usuario.recibo.v1'
       OR r#>>'{atributos,auditoria_sha256}' IS DISTINCT FROM encode(sha256(p_auditoria),'hex') THEN
        RAISE EXCEPTION 'AD3-36: consulta de recibo inválida' USING ERRCODE='22023';
    END IF;
    anterior:=b->>'Version'; nueva:=anterior;
    canon:='{"Esquema":"vec.contacto_usuario.recibo.v1","SujetoRef":'||
        vec_autorizacion_atestada_v3.texto_json_go(b->>'SujetoRef')||',"FinalidadRef":"gestion_contacto_propio",'||
        '"Audiencia":"vec.contacto_usuario.recibo.v1","Version":'||nueva||'}';
    IF x->>'persona_ref' IS NULL OR x->>'persona_ref' IS DISTINCT FROM b->>'SujetoRef' THEN
        RAISE EXCEPTION 'AD3-36: recibo propio ajeno' USING ERRCODE='42501';
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
        RAISE EXCEPTION 'AD3-36: material no ligado a decisión y contexto' USING ERRCODE='42501';
    END IF;
    RETURN b;
EXCEPTION WHEN data_exception THEN
    RAISE EXCEPTION 'AD3-36: material inválido' USING ERRCODE='22023';
END $f$;
-- Parche literal con pre/postimagen completa; conserva metadatos y dependencias.
DO $nucleo$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    def text; nueva text; metadata jsonb; deps jsonb; r record;
BEGIN
    SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc' INTO STRICT def,metadata FROM pg_proc p
        WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
        AND p.prosecdef AND p.provolatile='v' AND p.pronargdefaults=0
        AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']
        AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='d469a9b06ea400d38450c5e1e51913ef913f1333f97ffd2cc386f783b3d0b1d0';
    IF NOT COALESCE((SELECT count(*)=1 AND bool_and(x.grantee=p.proowner AND x.grantor=p.proowner
        AND x.privilege_type='EXECUTE' AND NOT x.is_grantable) FROM pg_proc p,
        LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f),false) THEN
        RAISE EXCEPTION 'AD3-36: ACL interna divergente' USING ERRCODE='55000';
    END IF;
    SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
        INTO deps FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
        OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f);
    nueva:=def;
    FOR r IN SELECT * FROM (VALUES
        ($antes0$p_perfil_mutacion IN ('contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_consultar')$antes0$,$despues0$p_perfil_mutacion IN ('contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_consultar','contacto_usuario_recibo')$despues0$,1),
        ($antes1$contacto_sesion_nominal_v1(p_perfil_mutacion)$antes1$,$despues1$contacto_sesion_nominal_v1(CASE WHEN p_perfil_mutacion='contacto_usuario_recibo' THEN 'contacto_usuario_alta' ELSE p_perfil_mutacion END)$despues1$,1),
        ($antes2$       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'$antes2$,$despues2$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_usuario_recibo'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.contacto_usuario.recibo.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'vec.contacto_usuario.consultar'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'vec.contacto_usuario.consultar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'vec.module.usuarios'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'contacto_usuario'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestion_contacto_propio'
           )
       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'$despues2$,1)
    ) AS cambios(antes,despues,veces) LOOP
        IF length(nueva)-length(replace(nueva,r.antes,''))<>length(r.antes)*r.veces THEN
            RAISE EXCEPTION 'AD3-36: fragmento no exacto' USING ERRCODE='55000';
        END IF;
        nueva:=replace(nueva,r.antes,r.despues);
    END LOOP;
    EXECUTE nueva;
    IF pg_get_functiondef(f) IS DISTINCT FROM nueva
       OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM '91f1c6e892ee49949b45ca7a912a9eb1eb4356ea35fc3b4dc10375a89fdc4d89'
       OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata
       OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
            FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
            OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f)) IS DISTINCT FROM deps THEN
        RAISE EXCEPTION 'AD3-36: cambio ajeno al perfil de recibo' USING ERRCODE='55000';
    END IF;
END $nucleo$;
-- Parche literal con pre/postimagen completa; conserva metadatos y dependencias.
DO $revalidacion$
DECLARE f oid:='vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    def text; nueva text; metadata jsonb; deps jsonb; r record;
BEGIN
    SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc' INTO STRICT def,metadata FROM pg_proc p
        WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
        AND p.prosecdef AND p.provolatile='v' AND p.pronargdefaults=0
        AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=1s']
        AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='7b93dd88a453fc052344825762a8ea0d46a8072fe6e058db09186c3bd50bde80';
    IF NOT COALESCE((SELECT count(*)=1 AND bool_and(x.grantee=p.proowner AND x.grantor=p.proowner
        AND x.privilege_type='EXECUTE' AND NOT x.is_grantable) FROM pg_proc p,
        LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f),false) THEN
        RAISE EXCEPTION 'AD3-36: ACL interna divergente' USING ERRCODE='55000';
    END IF;
    SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
        INTO deps FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
        OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f);
    nueva:=def;
    FOR r IN SELECT * FROM (VALUES
        ($antes0$p_perfil_consulta IN ('contacto_usuario','contacto_usuario_alta','contacto_usuario_actualizar')$antes0$,$despues0$p_perfil_consulta IN ('contacto_usuario','contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_recibo')$despues0$,2),
        ($antes1$CASE WHEN p_perfil_consulta='contacto_usuario' THEN 'contacto_usuario_consultar' ELSE p_perfil_consulta END$antes1$,$despues1$CASE WHEN p_perfil_consulta='contacto_usuario' THEN 'contacto_usuario_consultar' WHEN p_perfil_consulta='contacto_usuario_recibo' THEN 'contacto_usuario_alta' ELSE p_perfil_consulta END$despues1$,1),
        ($antes2$p_perfil_consulta NOT IN ('cuadro', 'detalle', 'contacto_usuario', 'contacto_usuario_alta', 'contacto_usuario_actualizar')$antes2$,$despues2$p_perfil_consulta NOT IN ('cuadro', 'detalle', 'contacto_usuario', 'contacto_usuario_alta', 'contacto_usuario_actualizar', 'contacto_usuario_recibo')$despues2$,1),
        ($antes3$IF p_perfil_consulta = 'contacto_usuario_alta' THEN$antes3$,$despues3$IF p_perfil_consulta = 'contacto_usuario_recibo' THEN
        v_audiencia := 'vec.contacto_usuario.recibo.v1';
        v_operacion := 'vec.contacto_usuario.consultar';
        v_tipo_recurso := 'contacto_usuario';
        v_finalidad := 'gestion_contacto_propio';
    ELSIF p_perfil_consulta = 'contacto_usuario_alta' THEN$despues3$,1)
    ) AS cambios(antes,despues,veces) LOOP
        IF length(nueva)-length(replace(nueva,r.antes,''))<>length(r.antes)*r.veces THEN
            RAISE EXCEPTION 'AD3-36: fragmento no exacto' USING ERRCODE='55000';
        END IF;
        nueva:=replace(nueva,r.antes,r.despues);
    END LOOP;
    EXECUTE nueva;
    IF pg_get_functiondef(f) IS DISTINCT FROM nueva
       OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM '631372341459e51db45786772e41562aa04fe08470c1a9727c7ba9907ccf2590'
       OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata
       OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
            FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
            OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f)) IS DISTINCT FROM deps THEN
        RAISE EXCEPTION 'AD3-36: cambio ajeno al perfil de recibo' USING ERRCODE='55000';
    END IF;
END $revalidacion$;
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
       OR NOT (ARRAY['vec.contacto_usuario.registro.v1','vec.contacto_usuario.consulta.v1']<@valores)
       OR ARRAY['vec.contacto_usuario.recibo.v1']&&valores THEN
        RAISE EXCEPTION 'AD3-36: audiencias previas incompatibles' USING ERRCODE='55000';
    END IF;
    valores:=valores||ARRAY['vec.contacto_usuario.recibo.v1'];
    ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
    EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check CHECK (audiencia_consumo IN ('||
        array_to_string(ARRAY(SELECT quote_literal(v) FROM unnest(valores) WITH ORDINALITY x(v,n) ORDER BY n),', ')||'))';
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_recibo_contacto_usuario_v3_atestada(
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
        RAISE EXCEPTION 'AD3-36: consumo contacto denegado' USING ERRCODE='42501';
    END IF;
    PERFORM vec_autorizacion_atestada_v3.contacto_recibo_validar_material_v1(
        'vec.contacto_usuario.consultar',p_negocio,p_recurso,p_auditoria,p_decision,p_contexto);
    SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
        'contacto_usuario_recibo',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload,p_sobre,p_evidencia,p_raiz);
    IF consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'AD3-36: contacto requiere concesión nueva' USING ERRCODE='P1102';
    END IF;
    RETURN QUERY SELECT consumo.decision_ref,consumo.efecto_ref,consumo.huella_efecto_sha256,
        consumo.consumo_huella_sha256,consumo.auditoria_ref,consumo.consumida_en,true;
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.revalidar_recibo_contacto_usuario_v3_atestada(
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
    p_negocio bytea,p_recurso bytea,p_auditoria bytea)
RETURNS TABLE(decision_ref text,consumo_huella_sha256 text,revalidada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='1s'
AS $f$
BEGIN
    IF vec_autorizacion_atestada_v3.contacto_sesion_nominal_v1('contacto_usuario_alta') IS NOT TRUE THEN
        RAISE EXCEPTION 'AD3-36: revalidación contacto denegada' USING ERRCODE='42501';
    END IF;
    PERFORM vec_autorizacion_atestada_v3.contacto_recibo_validar_material_v1(
        'vec.contacto_usuario.consultar',p_negocio,p_recurso,p_auditoria,p_decision,p_contexto);
    RETURN QUERY SELECT * FROM vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(
        'contacto_usuario_recibo',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload,p_sobre,p_evidencia,p_raiz);
END $f$;
-- Codec puro para la autoridad T13; por sí solo no concede acceso.
CREATE FUNCTION vec_autorizacion_atestada_v3.contacto_recibo_material_auditoria_v1(
    p_accion text,p_negocio bytea,p_recurso bytea,p_auditoria bytea,p_decision bytea,p_contexto bytea)
RETURNS jsonb LANGUAGE sql IMMUTABLE SECURITY DEFINER SET search_path=pg_catalog
AS $f$ SELECT vec_autorizacion_atestada_v3.contacto_recibo_validar_material_v1($1,$2,$3,$4,$5,$6) $f$;
DO $acl$
DECLARE r record; a record; f regprocedure; propietario oid:='vec_autorizacion_atestada_v3_propietario'::regrole; esperado oid;
BEGIN
    FOR r IN SELECT * FROM (VALUES
        ('contacto_recibo_validar_material_v1(text,bytea,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario'),
        ('registrar_y_consumir_recibo_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)','vec_contacto_usuario_owner'),
        ('revalidar_recibo_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)','vec_contacto_usuario_owner'),
        ('contacto_recibo_material_auditoria_v1(text,bytea,bytea,bytea,bytea,bytea)','vec_bolsa_accesos_propietario')
    ) AS funciones(firma,rol) LOOP
        f:=('vec_autorizacion_atestada_v3.'||r.firma)::regprocedure; esperado:=r.rol::regrole;
        EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f);
        FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
            WHERE p.oid=f AND x.grantee<>0 AND x.grantee<>p.proowner LOOP
            EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f,pg_get_userbyid(a.grantee));
        END LOOP;
        IF esperado<>propietario THEN EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO %I',f,r.rol); END IF;
        IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
           OR NOT COALESCE((SELECT count(*)=CASE WHEN esperado=propietario THEN 1 ELSE 2 END
                AND count(DISTINCT x.grantee)=count(*)
                AND bool_and(x.grantee IN (propietario,esperado) AND x.grantor=propietario
                    AND x.privilege_type='EXECUTE' AND NOT x.is_grantable)
                FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f),false) THEN
            RAISE EXCEPTION 'contacto recibo: ACL divergente' USING ERRCODE='55000';
        END IF;
    END LOOP;
END $acl$;
COMMIT;
