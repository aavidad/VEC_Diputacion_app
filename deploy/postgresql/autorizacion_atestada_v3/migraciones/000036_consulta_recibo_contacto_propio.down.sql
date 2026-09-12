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
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 IN ACCESS EXCLUSIVE MODE;
DO $historia$
BEGIN
    IF to_regprocedure('vec_contacto_usuario_v1.consultar_recibo_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR to_regprocedure('vec_bolsa_registro_accesos.registrar_consulta_recibo_contacto_v1(bytea,bytea,bytea,bytea,bytea,text,text,text,bytea)') IS NOT NULL
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
            WHERE audiencia_consumo='vec.contacto_usuario.recibo.v1')
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3
            WHERE convert_from(capacidad_canonica,'UTF8')::jsonb->>'audiencia_consumo'='vec.contacto_usuario.recibo.v1') THEN
        RAISE EXCEPTION 'AD3-36: dependencias o historia conservada; no admite DOWN' USING ERRCODE='55000';
    END IF;
END $historia$;
-- Parche literal con pre/postimagen completa; conserva metadatos y dependencias.
DO $nucleo$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    def text; nueva text; metadata jsonb; deps jsonb; r record;
BEGIN
    SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc' INTO STRICT def,metadata FROM pg_proc p
        WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
        AND p.prosecdef AND p.provolatile='v' AND p.pronargdefaults=0
        AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']
        AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='91f1c6e892ee49949b45ca7a912a9eb1eb4356ea35fc3b4dc10375a89fdc4d89';
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
            RAISE EXCEPTION 'AD3-36: fragmento no exacto' USING ERRCODE='55000';
        END IF;
        nueva:=replace(nueva,r.antes,r.despues);
    END LOOP;
    EXECUTE nueva;
    IF pg_get_functiondef(f) IS DISTINCT FROM nueva
       OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'd469a9b06ea400d38450c5e1e51913ef913f1333f97ffd2cc386f783b3d0b1d0'
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
        AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='631372341459e51db45786772e41562aa04fe08470c1a9727c7ba9907ccf2590';
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
            RAISE EXCEPTION 'AD3-36: fragmento no exacto' USING ERRCODE='55000';
        END IF;
        nueva:=replace(nueva,r.antes,r.despues);
    END LOOP;
    EXECUTE nueva;
    IF pg_get_functiondef(f) IS DISTINCT FROM nueva
       OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM '7b93dd88a453fc052344825762a8ea0d46a8072fe6e058db09186c3bd50bde80'
       OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata
       OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
            FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
            OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f)) IS DISTINCT FROM deps THEN
        RAISE EXCEPTION 'AD3-36: cambio ajeno al perfil de recibo' USING ERRCODE='55000';
    END IF;
END $revalidacion$;
DO $audiencias$
DECLARE def text; nueva text; marca text:=', ''vec.contacto_usuario.recibo.v1''::text';
BEGIN
    SELECT pg_get_constraintdef(oid,true) INTO STRICT def FROM pg_constraint
        WHERE conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
        AND conname='clave_capacidad_version_audiencia_consumo_check' AND contype='c' AND convalidated;
    IF length(def)-length(replace(def,marca,''))<>length(marca) THEN
        RAISE EXCEPTION 'AD3-36: audiencias divergentes' USING ERRCODE='55000';
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
