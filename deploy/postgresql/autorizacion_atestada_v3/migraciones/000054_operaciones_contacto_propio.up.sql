\set ON_ERROR_STOP on
-- AD3-54: perfiles propios de intención de contacto, posteriores a CT51 y
-- Contacto52/53. No conceden lectura de correo ni autoridad de B7.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000054',0));
DO $pre$
DECLARE r text;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' OR getdatabaseencoding()<>'UTF8'
    OR to_regprocedure('vec_autorizacion_atestada_v3.contacto_version_material_auditoria_v1(text,bytea,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.contacto_huellas_replay_canonicas_v1(jsonb)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.contacto_operacion_material_auditoria_v1(text,bytea,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
        WHERE n.nspname='vec_bolsa_registro_accesos' AND p.proname='registrar_operacion_contacto_v1')
    OR to_regclass('vec_contacto_usuario_v1.operaciones') IS NOT NULL THEN
    RAISE EXCEPTION 'AD3-54: preimagen Contacto53 incompatible' USING ERRCODE='55000';
 END IF;
 FOREACH r IN ARRAY ARRAY['vec_contacto_usuario_owner','vec_contacto_usuario_writer',
      'vec_contacto_usuario_reader','vec_contacto_usuario_migrador'] LOOP
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=r AND NOT rolcanlogin AND NOT rolinherit
        AND NOT rolsuper AND NOT rolbypassrls AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication)
       OR has_function_privilege(r,'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
       RAISE EXCEPTION 'AD3-54: roles o ACL de contacto incompatibles' USING ERRCODE='55000';
    END IF;
 END LOOP;
END $pre$;

-- AuditEntry previa con las mismas 17 claves y orden JSON de la autoridad
-- común. Se diferencia de alta/consulta del correo por cuatro acciones propias.
CREATE FUNCTION vec_autorizacion_atestada_v3.contacto_operacion_auditoria_previa_v1(p_auditoria bytea)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog
AS $f$
DECLARE a jsonb; c text; instante timestamptz; fecha text;
BEGIN
 IF p_auditoria IS NULL OR octet_length(p_auditoria) NOT BETWEEN 2 AND 16384 THEN
    RAISE EXCEPTION 'AD3-54: auditoría inválida' USING ERRCODE='22023';
 END IF;
 a:=convert_from(p_auditoria,'UTF8')::jsonb;
 IF vec_autorizacion_atestada_v3.contacto_objeto_v1(a,'{
    "id":"string","seq":"number","actor_id":"string","actor_profile":"string",
    "actor_roles":"array","auth_method":"string","auth_assurance":"string",
    "purpose":"string","action":"string","module_id":"string","subject_ref":"string",
    "object_version":"number","result":"string","correlation_ref":"string",
    "occurred_at":"string","signature":"string"}'::jsonb) IS NOT TRUE THEN
    RAISE EXCEPTION 'AD3-54: campos de auditoría inválidos' USING ERRCODE='22023';
 END IF;
 -- La comparación explícita evita que NULL/JSON null salten guardas ternarias.
 IF a->>'id' IS DISTINCT FROM '' OR a->>'seq' IS DISTINCT FROM '0'
    OR a->>'signature' IS DISTINCT FROM ''
    OR a->>'actor_id' !~ '^hmac-sha256:[a-z][a-z0-9._-]{0,63}:[0-9a-f]{64}$'
    OR split_part(a->>'actor_id',':',3)=repeat('0',64)
    OR a->>'actor_profile' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR jsonb_array_length(a->'actor_roles')<>1 OR jsonb_typeof(a#>'{actor_roles,0}') IS DISTINCT FROM 'string'
    OR a#>>'{actor_roles,0}' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR a->>'auth_method' NOT IN ('certificado','dnie') OR a->>'auth_assurance'<>'alto'
    OR a->>'purpose'<>'gestion_contacto_propio'
    OR a->>'action' NOT IN ('vec.contacto_usuario.operacion.preparar','vec.contacto_usuario.operacion.cancelar',
                           'vec.contacto_usuario.operacion.listar','vec.contacto_usuario.operacion.detalle')
    OR a->>'module_id'<>'vec.module.usuarios' OR a->>'subject_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR a->>'object_version' !~ '^[1-9][0-9]{0,15}$' OR (a->>'object_version')::numeric>9007199254740991
    OR a->>'result'<>'accepted' OR a->>'correlation_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR a->>'occurred_at' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}([.][0-9]{1,6})?Z$' THEN
    RAISE EXCEPTION 'AD3-54: auditoría nominal inválida' USING ERRCODE='22023';
 END IF;
 instante:=(a->>'occurred_at')::timestamptz;
 fecha:=to_char(instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS')||
    CASE WHEN to_char(instante AT TIME ZONE 'UTC','US')='000000' THEN ''
        ELSE '.'||rtrim(to_char(instante AT TIME ZONE 'UTC','US'),'0') END||'Z';
 IF NOT isfinite(instante) OR fecha IS DISTINCT FROM a->>'occurred_at' THEN
    RAISE EXCEPTION 'AD3-54: instante inválido' USING ERRCODE='22023';
 END IF;
 c:='{"id":"","seq":0,"actor_id":'||vec_autorizacion_atestada_v3.texto_json_go(a->>'actor_id')||
    ',"actor_profile":'||vec_autorizacion_atestada_v3.texto_json_go(a->>'actor_profile')||
    ',"actor_roles":['||vec_autorizacion_atestada_v3.texto_json_go(a#>>'{actor_roles,0}')||
    '],"auth_method":'||vec_autorizacion_atestada_v3.texto_json_go(a->>'auth_method')||
    ',"auth_assurance":"alto","purpose":"gestion_contacto_propio","action":'||
    vec_autorizacion_atestada_v3.texto_json_go(a->>'action')||
    ',"module_id":"vec.module.usuarios","subject_ref":'||
    vec_autorizacion_atestada_v3.texto_json_go(a->>'subject_ref')||
    ',"object_version":'||(a->>'object_version')||',"result":"accepted","correlation_ref":'||
    vec_autorizacion_atestada_v3.texto_json_go(a->>'correlation_ref')||
    ',"occurred_at":'||vec_autorizacion_atestada_v3.texto_json_go(a->>'occurred_at')||',"signature":""}';
 IF convert_to(c,'UTF8') IS DISTINCT FROM p_auditoria THEN
    RAISE EXCEPTION 'AD3-54: auditoría no canónica' USING ERRCODE='22023';
 END IF;
 RETURN a;
EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'AD3-54: auditoría inválida' USING ERRCODE='22023';
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.contacto_operacion_validar_material_v1(
    p_accion text,p_negocio bytea,p_recurso bytea,p_auditoria bytea,p_decision bytea,p_contexto bytea)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog
AS $f$
DECLARE b jsonb; r jsonb; a jsonb; d jsonb; x jsonb; canon text; rc text; attrs jsonb;
    sujeto text; operacion text; audiencia text; esperado text; huellas text; limite text; cursor text; version text;
BEGIN
 IF p_accion IS NULL OR p_accion NOT IN ('vec.contacto_usuario.operacion.preparar','vec.contacto_usuario.operacion.cancelar',
          'vec.contacto_usuario.operacion.listar','vec.contacto_usuario.operacion.detalle')
    OR p_negocio IS NULL OR octet_length(p_negocio) NOT BETWEEN 2 AND 65536
    OR p_recurso IS NULL OR octet_length(p_recurso) NOT BETWEEN 2 AND 16384
    OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
    OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 2 AND 262144 THEN
    RAISE EXCEPTION 'AD3-54: material inválido' USING ERRCODE='22023';
 END IF;
 b:=convert_from(p_negocio,'UTF8')::jsonb; r:=convert_from(p_recurso,'UTF8')::jsonb;
 d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
 a:=vec_autorizacion_atestada_v3.contacto_operacion_auditoria_previa_v1(p_auditoria);
 IF vec_autorizacion_atestada_v3.contacto_objeto_v1(r,'{"ambitos":"object","atributos":"object"}'::jsonb) IS NOT TRUE
    OR vec_autorizacion_atestada_v3.contacto_objeto_v1(r->'ambitos','{"persona_ref":"string"}'::jsonb) IS NOT TRUE THEN
    RAISE EXCEPTION 'AD3-54: recurso propio inválido' USING ERRCODE='22023';
 END IF;
 rc:='{"ambitos":'||vec_autorizacion_atestada_v3.contacto_mapa_canonico_v1(r->'ambitos')||
     ',"atributos":'||vec_autorizacion_atestada_v3.contacto_mapa_canonico_v1(r->'atributos')||'}';
 IF p_recurso IS DISTINCT FROM convert_to(rc,'UTF8') THEN
    RAISE EXCEPTION 'AD3-54: recurso no canónico' USING ERRCODE='22023';
 END IF;
 sujeto:=b->>'SujetoRef'; operacion:=b->>'OperacionRef'; attrs:=r->'atributos';
 IF sujeto IS NULL OR sujeto !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR r#>>'{ambitos,persona_ref}' IS DISTINCT FROM sujeto
    OR attrs->>'contacto_sujeto_ref' IS DISTINCT FROM sujeto
    OR attrs->>'material_sha256' IS DISTINCT FROM encode(sha256(p_negocio),'hex')
    OR b->'Auditoria' IS DISTINCT FROM a OR a->>'subject_ref' IS DISTINCT FROM sujeto
    OR a->>'action' IS DISTINCT FROM p_accion THEN
    RAISE EXCEPTION 'AD3-54: persona, auditoría o compromiso ajenos' USING ERRCODE='42501';
 END IF;
 audiencia:=CASE p_accion
    WHEN 'vec.contacto_usuario.operacion.preparar' THEN 'vec.contacto_usuario.operacion.preparar.v1'
    WHEN 'vec.contacto_usuario.operacion.cancelar' THEN 'vec.contacto_usuario.operacion.cancelar.v1'
    WHEN 'vec.contacto_usuario.operacion.listar' THEN 'vec.contacto_usuario.operacion.listar.v1'
    ELSE 'vec.contacto_usuario.operacion.detalle.v1' END;
 IF p_accion='vec.contacto_usuario.operacion.preparar' THEN
    IF vec_autorizacion_atestada_v3.contacto_objeto_v1(b,'{"Esquema":"string","SujetoRef":"string","OperacionRef":"string","VersionEsperada":"number","HuellasReplay":"array","Auditoria":"object"}'::jsonb) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.contacto_objeto_v1(attrs,'{"contacto_operacion_ref":"string","contacto_sujeto_ref":"string","contacto_version_esperada":"string","material_sha256":"string"}'::jsonb) IS NOT TRUE THEN
       RAISE EXCEPTION 'AD3-54: campos de preparación inválidos' USING ERRCODE='22023';
    END IF;
    version:=b->>'VersionEsperada';
    IF version IS NULL OR version !~ '^(0|[1-9][0-9]{0,15})$' OR version::numeric>9007199254740990
       OR attrs->>'contacto_version_esperada' IS DISTINCT FROM version
       OR (a->>'object_version')::numeric IS DISTINCT FROM version::numeric+1 THEN
       RAISE EXCEPTION 'AD3-54: versión de preparación inválida' USING ERRCODE='22023';
    END IF;
    huellas:=vec_autorizacion_atestada_v3.contacto_huellas_replay_canonicas_v1(b->'HuellasReplay');
    canon:='{"Esquema":"vec.contacto_usuario.operacion.preparar.v1","SujetoRef":'||
       vec_autorizacion_atestada_v3.texto_json_go(sujeto)||',"OperacionRef":'||
       vec_autorizacion_atestada_v3.texto_json_go(operacion)||',"VersionEsperada":'||version||
       ',"HuellasReplay":'||huellas||',"Auditoria":'||convert_from(p_auditoria,'UTF8')||'}';
 ELSIF p_accion='vec.contacto_usuario.operacion.listar' THEN
    IF vec_autorizacion_atestada_v3.contacto_objeto_v1(b,'{"Esquema":"string","SujetoRef":"string","Limite":"number","DespuesDe":"string","Auditoria":"object"}'::jsonb) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.contacto_objeto_v1(attrs,'{"contacto_sujeto_ref":"string","material_sha256":"string"}'::jsonb) IS NOT TRUE THEN
       RAISE EXCEPTION 'AD3-54: campos de lista inválidos' USING ERRCODE='22023';
    END IF;
    limite:=b->>'Limite'; cursor:=b->>'DespuesDe';
    IF limite IS NULL OR limite !~ '^[1-9][0-9]?$' OR limite::integer>50
       OR cursor IS NULL OR (cursor<>'' AND cursor !~ '^opr_[A-Za-z0-9_-]{22,128}$')
       OR (a->>'object_version')::numeric<>1 THEN
       RAISE EXCEPTION 'AD3-54: paginación inválida' USING ERRCODE='22023';
    END IF;
    canon:='{"Esquema":"vec.contacto_usuario.operacion.listar.v1","SujetoRef":'||
       vec_autorizacion_atestada_v3.texto_json_go(sujeto)||',"Limite":'||limite||',"DespuesDe":'||
       vec_autorizacion_atestada_v3.texto_json_go(cursor)||',"Auditoria":'||convert_from(p_auditoria,'UTF8')||'}';
 ELSE
    IF vec_autorizacion_atestada_v3.contacto_objeto_v1(b,'{"Esquema":"string","SujetoRef":"string","OperacionRef":"string","Auditoria":"object"}'::jsonb) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.contacto_objeto_v1(attrs,'{"contacto_operacion_ref":"string","contacto_sujeto_ref":"string","material_sha256":"string"}'::jsonb) IS NOT TRUE
       OR (a->>'object_version')::numeric<>1 THEN
       RAISE EXCEPTION 'AD3-54: selector inválido' USING ERRCODE='22023';
    END IF;
    canon:='{"Esquema":'||vec_autorizacion_atestada_v3.texto_json_go(audiencia)||',"SujetoRef":'||
       vec_autorizacion_atestada_v3.texto_json_go(sujeto)||',"OperacionRef":'||
       vec_autorizacion_atestada_v3.texto_json_go(operacion)||',"Auditoria":'||convert_from(p_auditoria,'UTF8')||'}';
 END IF;
 IF p_accion<>'vec.contacto_usuario.operacion.listar' AND
    (operacion IS NULL OR operacion !~ '^opr_[A-Za-z0-9_-]{22,128}$'
     OR attrs->>'contacto_operacion_ref' IS DISTINCT FROM operacion) THEN
    RAISE EXCEPTION 'AD3-54: operación desligada' USING ERRCODE='42501';
 END IF;
 IF b->>'Esquema' IS DISTINCT FROM audiencia OR p_negocio IS DISTINCT FROM convert_to(canon,'UTF8')
    OR d->'concedida' IS DISTINCT FROM 'true'::jsonb OR d->>'codigo' IS DISTINCT FROM 'concedida'
    OR d->>'accion' IS DISTINCT FROM p_accion OR d->>'modulo_id' IS DISTINCT FROM 'vec.module.usuarios'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'contacto_usuario' OR d->>'recurso_ref' IS DISTINCT FROM sujeto
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_contacto_propio'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM encode(sha256(p_recurso),'hex')
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'perfil_activo_ref' IS DISTINCT FROM a->>'actor_profile'
    OR d->>'version_rol_ref' IS DISTINCT FROM a#>>'{actor_roles,0}'
    OR d->>'correlacion_ref' IS DISTINCT FROM a->>'correlation_ref'
    OR d#>>'{vinculo_autenticacion_actor,metodo_observado}' IS DISTINCT FROM a->>'auth_method'
    OR d#>>'{vinculo_autenticacion_actor,garantia_observada}' IS DISTINCT FROM a->>'auth_assurance'
    OR x->>'persona_ref' IS DISTINCT FROM sujeto OR x->>'principal_ref' IS DISTINCT FROM d->>'principal_id'
    OR x->>'perfil_activo_ref' IS DISTINCT FROM a->>'actor_profile' THEN
    RAISE EXCEPTION 'AD3-54: material no ligado a decisión y contexto' USING ERRCODE='42501';
 END IF;
 RETURN b;
EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'AD3-54: material inválido' USING ERRCODE='22023';
END $f$;

-- Amplía exclusivamente los perfiles de la intención propia. Los hashes
-- candidatos siguen sujetos al cotejo real PG18; la guarda WIP aborta UP.
DO $operacion_nucleo$
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
      ($antes0$p_perfil_mutacion IN ('contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_consultar','contacto_usuario_recibo','contacto_usuario_version_propia','contacto_usuario_version_llamamiento')$antes0$,
       $despues0$p_perfil_mutacion IN ('contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_consultar','contacto_usuario_recibo','contacto_usuario_version_propia','contacto_usuario_version_llamamiento','contacto_operacion_preparar','contacto_operacion_cancelar','contacto_operacion_listar','contacto_operacion_detalle')$despues0$,1),
      ($antes1$contacto_sesion_nominal_v1(CASE WHEN p_perfil_mutacion IN ('contacto_usuario_recibo','contacto_usuario_version_propia') THEN 'contacto_usuario_alta' WHEN p_perfil_mutacion='contacto_usuario_version_llamamiento' THEN 'contacto_usuario_consultar' ELSE p_perfil_mutacion END)$antes1$,
       $despues1$contacto_sesion_nominal_v1(CASE WHEN p_perfil_mutacion IN ('contacto_usuario_recibo','contacto_usuario_version_propia','contacto_operacion_preparar','contacto_operacion_cancelar','contacto_operacion_listar','contacto_operacion_detalle') THEN 'contacto_usuario_alta' WHEN p_perfil_mutacion='contacto_usuario_version_llamamiento' THEN 'contacto_usuario_consultar' ELSE p_perfil_mutacion END)$despues1$,1),
      ($antes2$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_usuario_version_llamamiento'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.contacto_usuario.version_llamamiento.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'vec.contacto_usuario.version_para_llamamiento'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'vec.contacto_usuario.version_para_llamamiento'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'vec.module.usuarios'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'contacto_usuario'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'envio_llamamiento'
           )$antes2$,
       $despues2$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_usuario_version_llamamiento'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.contacto_usuario.version_llamamiento.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'vec.contacto_usuario.version_para_llamamiento'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'vec.contacto_usuario.version_para_llamamiento'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'vec.module.usuarios'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'contacto_usuario'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'envio_llamamiento'
           )
           OR (
               p_perfil_mutacion IN ('contacto_operacion_preparar','contacto_operacion_cancelar',
                                     'contacto_operacion_listar','contacto_operacion_detalle')
               AND p_perfil_mutacion IS NOT DISTINCT FROM CASE d ->> 'accion'
                   WHEN 'vec.contacto_usuario.operacion.preparar' THEN 'contacto_operacion_preparar'
                   WHEN 'vec.contacto_usuario.operacion.cancelar' THEN 'contacto_operacion_cancelar'
                   WHEN 'vec.contacto_usuario.operacion.listar' THEN 'contacto_operacion_listar'
                   WHEN 'vec.contacto_usuario.operacion.detalle' THEN 'contacto_operacion_detalle' END
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM CASE d ->> 'accion'
                   WHEN 'vec.contacto_usuario.operacion.preparar' THEN 'vec.contacto_usuario.operacion.preparar.v1'
                   WHEN 'vec.contacto_usuario.operacion.cancelar' THEN 'vec.contacto_usuario.operacion.cancelar.v1'
                   WHEN 'vec.contacto_usuario.operacion.listar' THEN 'vec.contacto_usuario.operacion.listar.v1'
                   WHEN 'vec.contacto_usuario.operacion.detalle' THEN 'vec.contacto_usuario.operacion.detalle.v1' END
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'vec.module.usuarios'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'contacto_usuario'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestion_contacto_propio'
           )$despues2$,1)
    ) AS cambios(antes,despues,veces) LOOP
        IF length(nueva)-length(replace(nueva,r.antes,''))<>length(r.antes)*r.veces THEN
            RAISE EXCEPTION 'AD3-54: fragmento del núcleo no exacto' USING ERRCODE='55000';
        END IF;
        nueva:=replace(nueva,r.antes,r.despues);
    END LOOP;
    EXECUTE nueva;
    IF pg_get_functiondef(f) IS DISTINCT FROM nueva
       OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM '3434a876cc67ee9331d1f5a9340790495c3ce531994ba8ef235d4a455d627942'
       OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata
       OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
             FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
                OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f)) IS DISTINCT FROM deps THEN
        RAISE EXCEPTION 'AD3-54: cambio ajeno al núcleo' USING ERRCODE='55000';
    END IF;
END $operacion_nucleo$;

DO $operacion_revalidacion$
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
      ($antes0$p_perfil_consulta IN ('contacto_usuario','contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_recibo','contacto_usuario_version_propia','contacto_usuario_version_llamamiento')$antes0$,
       $despues0$p_perfil_consulta IN ('contacto_usuario','contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_recibo','contacto_usuario_version_propia','contacto_usuario_version_llamamiento','contacto_operacion_preparar','contacto_operacion_cancelar','contacto_operacion_listar','contacto_operacion_detalle')$despues0$,2),
      ($antes1$CASE WHEN p_perfil_consulta='contacto_usuario' THEN 'contacto_usuario_consultar' WHEN p_perfil_consulta IN ('contacto_usuario_recibo','contacto_usuario_version_propia') THEN 'contacto_usuario_alta' WHEN p_perfil_consulta='contacto_usuario_version_llamamiento' THEN 'contacto_usuario_consultar' ELSE p_perfil_consulta END$antes1$,
       $despues1$CASE WHEN p_perfil_consulta='contacto_usuario' THEN 'contacto_usuario_consultar' WHEN p_perfil_consulta IN ('contacto_usuario_recibo','contacto_usuario_version_propia','contacto_operacion_preparar','contacto_operacion_cancelar','contacto_operacion_listar','contacto_operacion_detalle') THEN 'contacto_usuario_alta' WHEN p_perfil_consulta='contacto_usuario_version_llamamiento' THEN 'contacto_usuario_consultar' ELSE p_perfil_consulta END$despues1$,1),
      ($antes2$p_perfil_consulta NOT IN ('cuadro', 'detalle', 'contacto_usuario', 'contacto_usuario_alta', 'contacto_usuario_actualizar', 'contacto_usuario_recibo', 'contacto_usuario_version_propia', 'contacto_usuario_version_llamamiento')$antes2$,
       $despues2$p_perfil_consulta NOT IN ('cuadro', 'detalle', 'contacto_usuario', 'contacto_usuario_alta', 'contacto_usuario_actualizar', 'contacto_usuario_recibo', 'contacto_usuario_version_propia', 'contacto_usuario_version_llamamiento', 'contacto_operacion_preparar', 'contacto_operacion_cancelar', 'contacto_operacion_listar', 'contacto_operacion_detalle')$despues2$,1),
      ($antes3$IF p_perfil_consulta = 'contacto_usuario_version_propia' THEN$antes3$,
       $despues3$IF p_perfil_consulta IN ('contacto_operacion_preparar','contacto_operacion_cancelar',
                                'contacto_operacion_listar','contacto_operacion_detalle') THEN
        v_audiencia := CASE p_perfil_consulta
          WHEN 'contacto_operacion_preparar' THEN 'vec.contacto_usuario.operacion.preparar.v1'
          WHEN 'contacto_operacion_cancelar' THEN 'vec.contacto_usuario.operacion.cancelar.v1'
          WHEN 'contacto_operacion_listar' THEN 'vec.contacto_usuario.operacion.listar.v1'
          WHEN 'contacto_operacion_detalle' THEN 'vec.contacto_usuario.operacion.detalle.v1' END;
        v_operacion := CASE p_perfil_consulta
          WHEN 'contacto_operacion_preparar' THEN 'vec.contacto_usuario.operacion.preparar'
          WHEN 'contacto_operacion_cancelar' THEN 'vec.contacto_usuario.operacion.cancelar'
          WHEN 'contacto_operacion_listar' THEN 'vec.contacto_usuario.operacion.listar'
          WHEN 'contacto_operacion_detalle' THEN 'vec.contacto_usuario.operacion.detalle' END;
        v_tipo_recurso := 'contacto_usuario';
        v_finalidad := 'gestion_contacto_propio';
    ELSIF p_perfil_consulta = 'contacto_usuario_version_propia' THEN$despues3$,1)
    ) AS cambios(antes,despues,veces) LOOP
        IF length(nueva)-length(replace(nueva,r.antes,''))<>length(r.antes)*r.veces THEN
            RAISE EXCEPTION 'AD3-54: fragmento de revalidación no exacto' USING ERRCODE='55000';
        END IF;
        nueva:=replace(nueva,r.antes,r.despues);
    END LOOP;
    EXECUTE nueva;
    IF pg_get_functiondef(f) IS DISTINCT FROM nueva
       OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM '51f4b05feb19efeff9f1e985a723ed7f64e03d1b8d264691ad57131676467b14'
       OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata
       OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
             FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
                OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f)) IS DISTINCT FROM deps THEN
        RAISE EXCEPTION 'AD3-54: cambio ajeno a revalidación' USING ERRCODE='55000';
    END IF;
END $operacion_revalidacion$;

DO $audiencias$
DECLARE def text; valores text[]; canon text;
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(oid,true),'\s+',' ','g') INTO STRICT def FROM pg_constraint
   WHERE conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
     AND conname='clave_capacidad_version_audiencia_consumo_check' AND contype='c' AND convalidated AND conkey=ARRAY[8]::smallint[];
 SELECT array_agg(m[1] ORDER BY n) INTO valores FROM regexp_matches(def,'''([a-zA-Z0-9_.-]+)''::text','g') WITH ORDINALITY x(m,n);
 canon:='CHECK (audiencia_consumo = ANY (ARRAY['||array_to_string(ARRAY(
   SELECT quote_literal(v)||'::text' FROM unnest(valores) WITH ORDINALITY x(v,n) ORDER BY n),', ')||']))';
 IF def IS DISTINCT FROM canon OR cardinality(valores) NOT BETWEEN 15 AND 68
    OR NOT (ARRAY['vec.contacto_usuario.registro.v1','vec.contacto_usuario.consulta.v1',
        'vec.contacto_usuario.recibo.v1','vec.contacto_usuario.version_propia.v1',
        'vec.contacto_usuario.version_llamamiento.v1']<@valores)
    OR ARRAY['vec.contacto_usuario.operacion.preparar.v1','vec.contacto_usuario.operacion.cancelar.v1',
        'vec.contacto_usuario.operacion.listar.v1','vec.contacto_usuario.operacion.detalle.v1']&&valores THEN
    RAISE EXCEPTION 'AD3-54: audiencias previas incompatibles' USING ERRCODE='55000';
 END IF;
 valores:=valores||ARRAY['vec.contacto_usuario.operacion.preparar.v1','vec.contacto_usuario.operacion.cancelar.v1',
   'vec.contacto_usuario.operacion.listar.v1','vec.contacto_usuario.operacion.detalle.v1'];
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check CHECK (audiencia_consumo IN ('||
   array_to_string(ARRAY(SELECT quote_literal(v) FROM unnest(valores) WITH ORDINALITY x(v,n) ORDER BY n),', ')||'))';
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.contacto_operacion_material_auditoria_v1(
    p_accion text,p_negocio bytea,p_recurso bytea,p_auditoria bytea,p_decision bytea,p_contexto bytea)
RETURNS jsonb LANGUAGE sql IMMUTABLE SECURITY DEFINER SET search_path=pg_catalog
AS $f$
 SELECT vec_autorizacion_atestada_v3.contacto_operacion_validar_material_v1(
    p_accion,p_negocio,p_recurso,p_auditoria,p_decision,p_contexto)
$f$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.contacto_operacion_auditoria_previa_v1(bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.contacto_operacion_validar_material_v1(text,bytea,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.contacto_operacion_material_auditoria_v1(text,bytea,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.contacto_operacion_material_auditoria_v1(text,bytea,bytea,bytea,bytea,bytea)
    TO vec_bolsa_accesos_propietario,vec_contacto_usuario_owner;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_operacion_contacto_v3_atestada(
    p_accion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
    p_negocio bytea,p_recurso bytea,p_auditoria bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
    auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s'
AS $f$
DECLARE perfil text; consumo record;
BEGIN
 perfil:=CASE p_accion
    WHEN 'vec.contacto_usuario.operacion.preparar' THEN 'contacto_operacion_preparar'
    WHEN 'vec.contacto_usuario.operacion.cancelar' THEN 'contacto_operacion_cancelar'
    WHEN 'vec.contacto_usuario.operacion.listar' THEN 'contacto_operacion_listar'
    WHEN 'vec.contacto_usuario.operacion.detalle' THEN 'contacto_operacion_detalle' END;
 IF perfil IS NULL OR vec_autorizacion_atestada_v3.contacto_sesion_nominal_v1('contacto_usuario_alta') IS NOT TRUE THEN
    RAISE EXCEPTION 'AD3-54: operación propia denegada' USING ERRCODE='42501';
 END IF;
 PERFORM vec_autorizacion_atestada_v3.contacto_operacion_validar_material_v1(
    p_accion,p_negocio,p_recurso,p_auditoria,p_decision,p_contexto);
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
    perfil,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
    p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE THEN
    RAISE EXCEPTION 'AD3-54: operación requiere concesión nueva' USING ERRCODE='P1102';
 END IF;
 RETURN QUERY SELECT consumo.decision_ref,consumo.efecto_ref,consumo.huella_efecto_sha256,
    consumo.consumo_huella_sha256,consumo.auditoria_ref,consumo.consumida_en,true;
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.revalidar_operacion_contacto_v3_atestada(
    p_accion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
    p_negocio bytea,p_recurso bytea,p_auditoria bytea)
RETURNS TABLE(decision_ref text,consumo_huella_sha256 text,revalidada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='1s'
AS $f$
DECLARE perfil text;
BEGIN
 perfil:=CASE p_accion
    WHEN 'vec.contacto_usuario.operacion.preparar' THEN 'contacto_operacion_preparar'
    WHEN 'vec.contacto_usuario.operacion.cancelar' THEN 'contacto_operacion_cancelar'
    WHEN 'vec.contacto_usuario.operacion.listar' THEN 'contacto_operacion_listar'
    WHEN 'vec.contacto_usuario.operacion.detalle' THEN 'contacto_operacion_detalle' END;
 IF perfil IS NULL OR vec_autorizacion_atestada_v3.contacto_sesion_nominal_v1('contacto_usuario_alta') IS NOT TRUE THEN
    RAISE EXCEPTION 'AD3-54: revalidación propia denegada' USING ERRCODE='42501';
 END IF;
 PERFORM vec_autorizacion_atestada_v3.contacto_operacion_validar_material_v1(
    p_accion,p_negocio,p_recurso,p_auditoria,p_decision,p_contexto);
 RETURN QUERY SELECT * FROM vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(
    perfil,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
    p_payload,p_sobre,p_evidencia,p_raiz);
END $f$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_operacion_contacto_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.revalidar_operacion_contacto_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_operacion_contacto_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) TO vec_contacto_usuario_owner;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.revalidar_operacion_contacto_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) TO vec_contacto_usuario_owner;

DO $acl$
DECLARE f oid; r record; propietario oid:='vec_autorizacion_atestada_v3_propietario'::regrole;
BEGIN
 FOR r IN SELECT * FROM (VALUES
   ('contacto_operacion_auditoria_previa_v1(bytea)',1,'i','search_path=pg_catalog'),
   ('contacto_operacion_validar_material_v1(text,bytea,bytea,bytea,bytea,bytea)',1,'i','search_path=pg_catalog'),
   ('contacto_operacion_material_auditoria_v1(text,bytea,bytea,bytea,bytea,bytea)',3,'i','search_path=pg_catalog'),
   ('registrar_y_consumir_operacion_contacto_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)',2,'v','lock_timeout=2s'),
   ('revalidar_operacion_contacto_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)',2,'v','lock_timeout=1s')
 ) AS funciones(firma,numero,volatilidad,configuracion) LOOP
   f:=to_regprocedure('vec_autorizacion_atestada_v3.'||r.firma);
   IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner=propietario
       AND p.provolatile=r.volatilidad AND p.pronargdefaults=0 AND p.prosecdef=(r.numero>1)
       AND p.proconfig=CASE WHEN r.numero=1 OR r.firma LIKE 'contacto_operacion_material%'
           THEN ARRAY['search_path=pg_catalog'] ELSE ARRAY['search_path=pg_catalog',r.configuracion] END)
      OR NOT COALESCE((SELECT count(*)=r.numero AND count(DISTINCT x.grantee)=r.numero
           AND bool_and(x.grantor=propietario AND x.privilege_type='EXECUTE' AND NOT x.is_grantable
             AND x.grantee IN (propietario,'vec_contacto_usuario_owner'::regrole,'vec_bolsa_accesos_propietario'::regrole))
           FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f),false)
      OR (r.numero>1 AND NOT has_function_privilege('vec_contacto_usuario_owner',f,'EXECUTE'))
      OR (r.numero=3 AND NOT has_function_privilege('vec_bolsa_accesos_propietario',f,'EXECUTE'))
      OR (r.numero<3 AND has_function_privilege('vec_bolsa_accesos_propietario',f,'EXECUTE'))
      OR (r.numero=1 AND has_function_privilege('vec_contacto_usuario_owner',f,'EXECUTE'))
      OR has_function_privilege('vec_contacto_usuario_writer',f,'EXECUTE')
      OR has_function_privilege('vec_contacto_usuario_reader',f,'EXECUTE') THEN
      RAISE EXCEPTION 'AD3-54: ACL nominal divergente' USING ERRCODE='55000';
   END IF;
 END LOOP;
END $acl$;

-- Huellas candidatas derivadas por parche textual sobre CT51/Contacto52/53;
-- faltan cotejo pg_proc.prosrc/ACL, inversión/ensayo PG18 y revisión E10.
DO $incompleta$ BEGIN RAISE EXCEPTION 'AD3-54 WIP: postimagen y ensayo PG18 pendientes' USING ERRCODE='55000'; END $incompleta$;
COMMIT;
