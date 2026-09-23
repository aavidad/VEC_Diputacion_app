\set ON_ERROR_STOP on
-- Fuente candidata: consulta del recibo histórico propio; no acredita replay de Guardar.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000053',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 IN ACCESS EXCLUSIVE MODE;
DO $historia$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN
        RAISE EXCEPTION 'AD3 contacto: DOWN requiere DBA' USING ERRCODE='42501';
    END IF;
    IF to_regprocedure('vec_contacto_usuario_v1.consultar_recibo_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR to_regprocedure('vec_bolsa_registro_accesos.registrar_consulta_recibo_contacto_v1(bytea,bytea,bytea,bytea,bytea,text,text,text,bytea)') IS NOT NULL
       OR to_regprocedure('vec_bolsa_registro_accesos.registrar_consulta_version_contacto_v1(bytea,bytea,bytea,bytea,bytea,text,text,text,boolean,numeric)') IS NOT NULL
       OR to_regprocedure('vec_contacto_usuario_v1.consultar_version_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
            WHERE audiencia_consumo IN ('vec.contacto_usuario.recibo.v1','vec.contacto_usuario.version_propia.v1','vec.contacto_usuario.version_llamamiento.v1'))
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3
            WHERE convert_from(capacidad_canonica,'UTF8')::jsonb->>'audiencia_consumo' IN ('vec.contacto_usuario.recibo.v1','vec.contacto_usuario.version_propia.v1','vec.contacto_usuario.version_llamamiento.v1')) THEN
        RAISE EXCEPTION 'AD3-53: dependencias o historia conservada; no admite DOWN' USING ERRCODE='55000';
    END IF;
END $historia$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
-- El selector sólo se retira en ensayo sin historial ni dependencias T13/almacén.
DROP FUNCTION vec_autorizacion_atestada_v3.contacto_version_material_auditoria_v1(text,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.revalidar_version_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_version_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.contacto_version_validar_material_v1(text,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DO $version_nucleo$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    def text; nueva text; metadata jsonb; deps jsonb; r record;
BEGIN
    SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc' INTO STRICT def,metadata FROM pg_proc p
        WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
        AND p.prosecdef AND p.provolatile='v' AND p.pronargdefaults=0
        AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']
        AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='63c14d42c5fce79d92be437bd5bb61328ab14a26e2e5392cf7063f87371e2ffe';
    SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
        INTO deps FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
        OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f);
    nueva:=def;
    FOR r IN SELECT * FROM (VALUES
        ($antes0$p_perfil_mutacion IN ('contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_consultar','contacto_usuario_recibo','contacto_usuario_version_propia','contacto_usuario_version_llamamiento')$antes0$,$despues0$p_perfil_mutacion IN ('contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_consultar','contacto_usuario_recibo')$despues0$,1),
        ($antes1$contacto_sesion_nominal_v1(CASE WHEN p_perfil_mutacion IN ('contacto_usuario_recibo','contacto_usuario_version_propia') THEN 'contacto_usuario_alta' WHEN p_perfil_mutacion='contacto_usuario_version_llamamiento' THEN 'contacto_usuario_consultar' ELSE p_perfil_mutacion END)$antes1$,$despues1$contacto_sesion_nominal_v1(CASE WHEN p_perfil_mutacion='contacto_usuario_recibo' THEN 'contacto_usuario_alta' ELSE p_perfil_mutacion END)$despues1$,1),
        ($antes2$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_usuario_recibo'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.contacto_usuario.recibo.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'vec.contacto_usuario.consultar'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'vec.contacto_usuario.consultar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'vec.module.usuarios'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'contacto_usuario'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestion_contacto_propio'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_usuario_version_propia'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.contacto_usuario.version_propia.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'vec.contacto_usuario.version_propia'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'vec.contacto_usuario.version_propia'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'vec.module.usuarios'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'contacto_usuario'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestion_contacto_propio'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_usuario_version_llamamiento'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.contacto_usuario.version_llamamiento.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'vec.contacto_usuario.version_para_llamamiento'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'vec.contacto_usuario.version_para_llamamiento'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'vec.module.usuarios'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'contacto_usuario'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'envio_llamamiento'
           )$antes2$,$despues2$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_usuario_recibo'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.contacto_usuario.recibo.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'vec.contacto_usuario.consultar'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'vec.contacto_usuario.consultar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'vec.module.usuarios'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'contacto_usuario'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestion_contacto_propio'
           )$despues2$,1)
    ) AS cambios(antes,despues,veces) LOOP
        IF length(nueva)-length(replace(nueva,r.antes,''))<>length(r.antes)*r.veces THEN
            RAISE EXCEPTION 'AD3-53: selector DOWN sin postimagen exacta' USING ERRCODE='55000';
        END IF;
        nueva:=replace(nueva,r.antes,r.despues);
    END LOOP;
    EXECUTE nueva;
    IF pg_get_functiondef(f) IS DISTINCT FROM nueva
       OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM '4440e6acb9e69624fff683ae48dd0204d6d8217bc1bd3aba22d96d22b6c1df91'
       OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata
       OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
            FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
            OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f)) IS DISTINCT FROM deps THEN
        RAISE EXCEPTION 'AD3-53: selector cambia núcleo ajeno' USING ERRCODE='55000';
    END IF;
END $version_nucleo$;

DO $version_revalidacion$
DECLARE f oid:='vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    def text; nueva text; metadata jsonb; deps jsonb; r record;
BEGIN
    SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc' INTO STRICT def,metadata FROM pg_proc p
        WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
        AND p.prosecdef AND p.provolatile='v' AND p.pronargdefaults=0
        AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=1s']
        AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='58d7d00d858132f8e08cc470c716ca04fc0521268834482e293894f7be3d806b';
    SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
        INTO deps FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
        OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f);
    nueva:=def;
    FOR r IN SELECT * FROM (VALUES
        ($antes0$p_perfil_consulta IN ('contacto_usuario','contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_recibo','contacto_usuario_version_propia','contacto_usuario_version_llamamiento')$antes0$,$despues0$p_perfil_consulta IN ('contacto_usuario','contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_recibo')$despues0$,2),
        ($antes1$CASE WHEN p_perfil_consulta='contacto_usuario' THEN 'contacto_usuario_consultar' WHEN p_perfil_consulta IN ('contacto_usuario_recibo','contacto_usuario_version_propia') THEN 'contacto_usuario_alta' WHEN p_perfil_consulta='contacto_usuario_version_llamamiento' THEN 'contacto_usuario_consultar' ELSE p_perfil_consulta END$antes1$,$despues1$CASE WHEN p_perfil_consulta='contacto_usuario' THEN 'contacto_usuario_consultar' WHEN p_perfil_consulta='contacto_usuario_recibo' THEN 'contacto_usuario_alta' ELSE p_perfil_consulta END$despues1$,1),
        ($antes2$p_perfil_consulta NOT IN ('cuadro', 'detalle', 'contacto_usuario', 'contacto_usuario_alta', 'contacto_usuario_actualizar', 'contacto_usuario_recibo', 'contacto_usuario_version_propia', 'contacto_usuario_version_llamamiento')$antes2$,$despues2$p_perfil_consulta NOT IN ('cuadro', 'detalle', 'contacto_usuario', 'contacto_usuario_alta', 'contacto_usuario_actualizar', 'contacto_usuario_recibo')$despues2$,1),
        ($antes3$IF p_perfil_consulta = 'contacto_usuario_version_propia' THEN
        v_audiencia := 'vec.contacto_usuario.version_propia.v1';
        v_operacion := 'vec.contacto_usuario.version_propia';
        v_tipo_recurso := 'contacto_usuario';
        v_finalidad := 'gestion_contacto_propio';
    ELSIF p_perfil_consulta = 'contacto_usuario_version_llamamiento' THEN
        v_audiencia := 'vec.contacto_usuario.version_llamamiento.v1';
        v_operacion := 'vec.contacto_usuario.version_para_llamamiento';
        v_tipo_recurso := 'contacto_usuario';
        v_finalidad := 'envio_llamamiento';
    ELSIF p_perfil_consulta = 'contacto_usuario_recibo' THEN
        v_audiencia := 'vec.contacto_usuario.recibo.v1';
        v_operacion := 'vec.contacto_usuario.consultar';
        v_tipo_recurso := 'contacto_usuario';
        v_finalidad := 'gestion_contacto_propio';$antes3$,$despues3$IF p_perfil_consulta = 'contacto_usuario_recibo' THEN
        v_audiencia := 'vec.contacto_usuario.recibo.v1';
        v_operacion := 'vec.contacto_usuario.consultar';
        v_tipo_recurso := 'contacto_usuario';
        v_finalidad := 'gestion_contacto_propio';$despues3$,1)
    ) AS cambios(antes,despues,veces) LOOP
        IF length(nueva)-length(replace(nueva,r.antes,''))<>length(r.antes)*r.veces THEN
            RAISE EXCEPTION 'AD3-53: selector DOWN sin revalidación exacta' USING ERRCODE='55000';
        END IF;
        nueva:=replace(nueva,r.antes,r.despues);
    END LOOP;
    EXECUTE nueva;
    IF pg_get_functiondef(f) IS DISTINCT FROM nueva
       OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'b13d24219dd5f18beb873e6391c06b93bea423d6d242b40568d5c8bc80802145'
       OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata
       OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
            FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
            OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f)) IS DISTINCT FROM deps THEN
        RAISE EXCEPTION 'AD3-53: selector cambia revalidación ajena' USING ERRCODE='55000';
    END IF;
END $version_revalidacion$;
DO $version_audiencias$
DECLARE def text; valores text[]; canon text;
BEGIN
    SELECT regexp_replace(pg_get_constraintdef(oid,true),'\s+',' ','g') INTO STRICT def FROM pg_constraint
        WHERE conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
        AND conname='clave_capacidad_version_audiencia_consumo_check' AND contype='c' AND convalidated AND conkey=ARRAY[8]::smallint[];
    SELECT array_agg(m[1] ORDER BY n) INTO valores FROM regexp_matches(def,'''([a-zA-Z0-9_.-]+)''::text','g') WITH ORDINALITY x(m,n);
    canon:='CHECK (audiencia_consumo = ANY (ARRAY['||array_to_string(ARRAY(
        SELECT quote_literal(v)||'::text' FROM unnest(valores) WITH ORDINALITY x(v,n) ORDER BY n),', ')||']))';
    IF def IS DISTINCT FROM canon OR NOT (ARRAY['vec.contacto_usuario.recibo.v1',
        'vec.contacto_usuario.version_propia.v1','vec.contacto_usuario.version_llamamiento.v1']<@valores) THEN
        RAISE EXCEPTION 'AD3-53: audiencias de selector divergentes' USING ERRCODE='55000';
    END IF;
    valores:=array_remove(array_remove(valores,'vec.contacto_usuario.version_propia.v1'),
        'vec.contacto_usuario.version_llamamiento.v1');
    ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
    EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check CHECK (audiencia_consumo IN ('||
        array_to_string(ARRAY(SELECT quote_literal(v) FROM unnest(valores) WITH ORDINALITY x(v,n) ORDER BY n),', ')||'))';
END $version_audiencias$;

-- Parche literal con pre/postimagen completa; conserva metadatos y dependencias.
DO $nucleo$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    def text; nueva text; metadata jsonb; deps jsonb; r record;
BEGIN
    SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc' INTO STRICT def,metadata FROM pg_proc p
        WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
        AND p.prosecdef AND p.provolatile='v' AND p.pronargdefaults=0
        AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']
        AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='4440e6acb9e69624fff683ae48dd0204d6d8217bc1bd3aba22d96d22b6c1df91';
    IF NOT COALESCE((SELECT count(*)=1 AND bool_and(x.grantee=p.proowner AND x.grantor=p.proowner
        AND x.privilege_type='EXECUTE' AND NOT x.is_grantable) FROM pg_proc p,
        LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f),false) THEN
        RAISE EXCEPTION 'AD3-53: ACL interna divergente' USING ERRCODE='55000';
    END IF;
    SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
        INTO deps FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
        OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f);
    nueva:=def;
    FOR r IN SELECT * FROM (VALUES
        ($antes0$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_usuario_recibo'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.contacto_usuario.recibo.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'vec.contacto_usuario.consultar'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'vec.contacto_usuario.consultar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'vec.module.usuarios'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'contacto_usuario'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestion_contacto_propio'
           )
       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'$antes0$,$despues0$       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'$despues0$,1),
        ($antes1$contacto_sesion_nominal_v1(CASE WHEN p_perfil_mutacion='contacto_usuario_recibo' THEN 'contacto_usuario_alta' ELSE p_perfil_mutacion END)$antes1$,$despues1$contacto_sesion_nominal_v1(p_perfil_mutacion)$despues1$,1),
        ($antes2$p_perfil_mutacion IN ('contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_consultar','contacto_usuario_recibo')$antes2$,$despues2$p_perfil_mutacion IN ('contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_consultar')$despues2$,1)
    ) AS cambios(antes,despues,veces) LOOP
        IF length(nueva)-length(replace(nueva,r.antes,''))<>length(r.antes)*r.veces THEN
            RAISE EXCEPTION 'AD3-53: fragmento no exacto' USING ERRCODE='55000';
        END IF;
        nueva:=replace(nueva,r.antes,r.despues);
    END LOOP;
    EXECUTE nueva;
    IF pg_get_functiondef(f) IS DISTINCT FROM nueva
       OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM '9abb29c2a598fe6853eb3281b4231eb46a3158b8a8afffe718d3951296e50738'
       OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata
       OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
            FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
            OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f)) IS DISTINCT FROM deps THEN
        RAISE EXCEPTION 'AD3-53: cambio ajeno al perfil de recibo' USING ERRCODE='55000';
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
        AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='b13d24219dd5f18beb873e6391c06b93bea423d6d242b40568d5c8bc80802145';
    IF NOT COALESCE((SELECT count(*)=1 AND bool_and(x.grantee=p.proowner AND x.grantor=p.proowner
        AND x.privilege_type='EXECUTE' AND NOT x.is_grantable) FROM pg_proc p,
        LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f),false) THEN
        RAISE EXCEPTION 'AD3-53: ACL interna divergente' USING ERRCODE='55000';
    END IF;
    SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
        INTO deps FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
        OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f);
    nueva:=def;
    FOR r IN SELECT * FROM (VALUES
        ($antes0$IF p_perfil_consulta = 'contacto_usuario_recibo' THEN
        v_audiencia := 'vec.contacto_usuario.recibo.v1';
        v_operacion := 'vec.contacto_usuario.consultar';
        v_tipo_recurso := 'contacto_usuario';
        v_finalidad := 'gestion_contacto_propio';
    ELSIF p_perfil_consulta = 'contacto_usuario_alta' THEN$antes0$,$despues0$IF p_perfil_consulta = 'contacto_usuario_alta' THEN$despues0$,1),
        ($antes1$p_perfil_consulta NOT IN ('cuadro', 'detalle', 'contacto_usuario', 'contacto_usuario_alta', 'contacto_usuario_actualizar', 'contacto_usuario_recibo')$antes1$,$despues1$p_perfil_consulta NOT IN ('cuadro', 'detalle', 'contacto_usuario', 'contacto_usuario_alta', 'contacto_usuario_actualizar')$despues1$,1),
        ($antes2$CASE WHEN p_perfil_consulta='contacto_usuario' THEN 'contacto_usuario_consultar' WHEN p_perfil_consulta='contacto_usuario_recibo' THEN 'contacto_usuario_alta' ELSE p_perfil_consulta END$antes2$,$despues2$CASE WHEN p_perfil_consulta='contacto_usuario' THEN 'contacto_usuario_consultar' ELSE p_perfil_consulta END$despues2$,1),
        ($antes3$p_perfil_consulta IN ('contacto_usuario','contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_recibo')$antes3$,$despues3$p_perfil_consulta IN ('contacto_usuario','contacto_usuario_alta','contacto_usuario_actualizar')$despues3$,2)
    ) AS cambios(antes,despues,veces) LOOP
        IF length(nueva)-length(replace(nueva,r.antes,''))<>length(r.antes)*r.veces THEN
            RAISE EXCEPTION 'AD3-53: fragmento no exacto' USING ERRCODE='55000';
        END IF;
        nueva:=replace(nueva,r.antes,r.despues);
    END LOOP;
    EXECUTE nueva;
    IF pg_get_functiondef(f) IS DISTINCT FROM nueva
       OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM '9078f7304058d5b2a07fc1756ca13877696b47399b726735fed31dd466e5c295'
       OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata
       OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
            FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
            OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f)) IS DISTINCT FROM deps THEN
        RAISE EXCEPTION 'AD3-53: cambio ajeno al perfil de recibo' USING ERRCODE='55000';
    END IF;
END $revalidacion$;
DO $audiencias$
DECLARE def text; nueva text; marca text:=', ''vec.contacto_usuario.recibo.v1''::text';
BEGIN
    SELECT pg_get_constraintdef(oid,true) INTO STRICT def FROM pg_constraint
        WHERE conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
        AND conname='clave_capacidad_version_audiencia_consumo_check' AND contype='c' AND convalidated;
    IF length(def)-length(replace(def,marca,''))<>length(marca) THEN
        RAISE EXCEPTION 'AD3-53: audiencias divergentes' USING ERRCODE='55000';
    END IF;
    nueva:=replace(def,marca,'');
    ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
    EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencias$;
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_recibo_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.revalidar_recibo_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.contacto_recibo_material_auditoria_v1(text,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.contacto_recibo_validar_material_v1(text,bytea,bytea,bytea,bytea,bytea) RESTRICT;
COMMIT;
