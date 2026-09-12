\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
-- Compartido con AD3-29; no se adquiere este advisory lock en runtime.
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000005:lector:v2',0));

DO $precondiciones$
DECLARE r text; f oid; t oid;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_personal' AND nspowner=current_user::regrole)
       OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
           WHERE n.nspname='vec_personal' AND p.proname='acreditar_alta_ejercicio_v2')
       OR to_regprocedure('vec_personal.acreditar_alta_ejercicio_v1(text,text,text,bigint,text,text,text,text,text,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)') IS NULL
       OR NOT has_schema_privilege('vec_contratacion_temporal_propietario','vec_personal','USAGE') THEN
        RAISE EXCEPTION 'estado incompatible Personal 000005' USING ERRCODE='55000';
    END IF;
    FOREACH r IN ARRAY ARRAY['vec_personal_propietario','vec_personal_migrador','vec_personal_ejecutor',
        'vec_contratacion_temporal_propietario','vec_contratacion_temporal_migrador','vec_contratacion_temporal_ejecutor'] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=r AND NOT rolcanlogin AND NOT rolsuper
            AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls) THEN
            RAISE EXCEPTION 'roles incompatibles Personal 000005' USING ERRCODE='55000';
        END IF;
    END LOOP;
    -- USAGE CT directo no delegable ya instalado por 000003; no se concede aquí.
    IF has_schema_privilege('vec_contratacion_temporal_propietario','vec_personal','CREATE')
       OR EXISTS (SELECT 1 FROM pg_namespace n, LATERAL aclexplode(COALESCE(n.nspacl,acldefault('n',n.nspowner))) a
           WHERE n.nspname='vec_personal' AND a.grantee='vec_contratacion_temporal_propietario'::regrole
             AND (a.privilege_type<>'USAGE' OR a.is_grantable OR a.grantor<>n.nspowner))
       OR (has_schema_privilege('vec_contratacion_temporal_propietario','vec_personal','USAGE')
           AND NOT EXISTS (SELECT 1 FROM pg_namespace n, LATERAL aclexplode(n.nspacl) a
               WHERE n.nspname='vec_personal' AND a.grantee='vec_contratacion_temporal_propietario'::regrole
                 AND a.privilege_type='USAGE' AND NOT a.is_grantable AND a.grantor=n.nspowner))
       OR pg_has_role('vec_contratacion_temporal_propietario','vec_personal_propietario','MEMBER')
       OR pg_has_role('vec_contratacion_temporal_ejecutor','vec_personal_ejecutor','MEMBER')
       OR pg_has_role('vec_personal_ejecutor','vec_contratacion_temporal_ejecutor','MEMBER') THEN
        RAISE EXCEPTION 'separación o ACL incompatible Personal 000005' USING ERRCODE='55000';
    END IF;
    FOREACH r IN ARRAY ARRAY['material_alta_ejercicio_canonico_v1','solicitud_alta_ejercicio_canonica_v1',
        'contexto_alta_ejercicio_canonico_v1'] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
            WHERE n.nspname='vec_personal' AND p.proname=r AND p.pronargs=1 AND p.proargtypes[0]='jsonb'::regtype
              AND p.prorettype='bytea'::regtype AND p.proowner=current_user::regrole AND NOT p.prosecdef
              AND p.provolatile='i') THEN
            RAISE EXCEPTION 'codec Personal requerido' USING ERRCODE='55000';
        END IF;
    END LOOP;
    FOREACH r IN ARRAY ARRAY['registro_alta_ejercicio','relacion_alta_ejercicio','ocupacion_alta_ejercicio',
        'auditoria_alta_ejercicio','outbox_alta_ejercicio','auditoria_lectura_incorporacion'] LOOP
        SELECT c.oid INTO t FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
            WHERE n.nspname='vec_personal' AND c.relname=r AND c.relkind='r'
              AND c.relowner=current_user::regrole AND c.relrowsecurity AND c.relforcerowsecurity;
        IF t IS NULL OR EXISTS (SELECT 1 FROM aclexplode(COALESCE((SELECT relacl FROM pg_class WHERE oid=t),
            acldefault('r',current_user::regrole))) a WHERE a.grantee<>current_user::regrole)
           OR (SELECT count(*) FROM pg_policy WHERE polrelid=t)<>1
           OR NOT EXISTS (SELECT 1 FROM pg_policy WHERE polrelid=t AND polname='propietario' AND polcmd='*'
               AND polpermissive AND polroles=ARRAY[current_user::regrole::oid]
               AND pg_get_expr(polqual,polrelid)='true' AND pg_get_expr(polwithcheck,polrelid)='true') THEN
            RAISE EXCEPTION 'registro propietario Personal requerido' USING ERRCODE='55000';
        END IF;
    END LOOP;
    SELECT p.oid INTO f FROM pg_proc p WHERE p.oid=to_regprocedure(
        'vec_autorizacion_atestada_v3.registrar_y_consumir_lectura_personal_incorporacion_v2_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
        AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef AND p.proretset
        AND p.prorettype='record'::regtype AND p.provolatile='v'
        AND p.proallargtypes[11:17]=ARRAY['text'::regtype::oid,'text'::regtype::oid,'text'::regtype::oid,
            'text'::regtype::oid,'text'::regtype::oid,'timestamptz'::regtype::oid,'boolean'::regtype::oid]
        AND p.proargnames[11:17]=ARRAY['decision_ref','efecto_ref','huella_efecto_sha256',
            'consumo_huella_sha256','auditoria_ref','consumida_en','consumo_nuevo']
        AND has_function_privilege(current_user,p.oid,'EXECUTE');
    IF f IS NULL OR NOT has_schema_privilege(current_user,'vec_autorizacion_atestada_v3','USAGE')
       OR EXISTS (SELECT 1 FROM pg_proc p, LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a
           WHERE p.oid=f AND (a.grantor<>p.proowner OR a.is_grantable OR a.grantee NOT IN (p.proowner,current_user::regrole) OR
               (a.grantee<>p.proowner AND (a.privilege_type<>'EXECUTE' OR a.is_grantable)))) THEN
        RAISE EXCEPTION 'wrapper AD3-29 propietario requerido' USING ERRCODE='55000';
    END IF;
END
$precondiciones$;


DO $helpers$
DECLARE f oid; r record;
BEGIN
    FOR r IN SELECT * FROM (VALUES
        ('vec_personal.texto_json_go_v1(text)','text'),
        ('vec_personal.referencia_alta_ejercicio_valida_v1(text)','boolean'),
        ('vec_personal.material_registro_alta_valido_v1(jsonb)','boolean'),
        ('vec_personal.fecha_registro_alta_pg_v1(text)','date')
    ) x(firma,retorno) LOOP
        f:=to_regprocedure(r.firma);
        IF f IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f AND proowner=current_user::regrole
            AND prorettype=to_regtype(r.retorno) AND NOT prosecdef AND provolatile='i'
            AND proconfig @> ARRAY['search_path=pg_catalog']) THEN
            RAISE EXCEPTION 'Personal 000005: helper propietario requerido' USING ERRCODE='55000';
        END IF;
    END LOOP;
    IF (SELECT count(*) FROM pg_trigger WHERE tgrelid='vec_personal.auditoria_lectura_incorporacion'::regclass
        AND NOT tgisinternal)<>1 OR NOT EXISTS (SELECT 1 FROM pg_trigger
        WHERE tgrelid='vec_personal.auditoria_lectura_incorporacion'::regclass AND NOT tgisinternal
        AND tgname='inmutable' AND tgtype=58 AND tgenabled='O'
        AND tgfoid='vec_personal.rechazar_mutacion_alta_ejercicio_v1()'::regprocedure) THEN
        RAISE EXCEPTION 'Personal 000005: auditoría inmutable requerida' USING ERRCODE='55000';
    END IF;
END
$helpers$;

CREATE FUNCTION vec_personal.acreditar_alta_ejercicio_v2(
    p_organizacion_ref text,p_solicitud_ref text,p_expediente_ref text,p_version_expediente bigint,
    p_resultado_ref text,p_recibo_ref text,p_relacion_ref text,p_ocupacion_ref text,p_material_sha256 text,p_unidad_ref text,
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version bigint,p_perfil_version bigint,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s'
AS $lector$
DECLARE
    d jsonb; c jsonb; actor jsonb; canon_contexto bytea; h text;
    consumo record; r vec_personal.registro_alta_ejercicio%ROWTYPE;
    rel vec_personal.relacion_alta_ejercicio%ROWTYPE; ocu vec_personal.ocupacion_alta_ejercicio%ROWTYPE;
    aud vec_personal.auditoria_alta_ejercicio%ROWTYPE; ob vec_personal.outbox_alta_ejercicio%ROWTYPE;
    w jsonb; s jsonb; x jsonb; recibo jsonb; evento jsonb; respuesta jsonb;
    mc bytea; sc bytea; rc bytea; hm text; hs text; hr text;
    inicio timestamptz(6); ahora timestamptz(6); leida timestamptz(6);
    ci timestamptz(6); cf timestamptz(6); di timestamptz(6); df timestamptz(6);
    config_fin timestamptz(6); raiz_fin timestamptz(6); limite timestamptz(6);
    ref_auditoria text; v text; discrepancia_nominal boolean:=false;
BEGIN
    IF current_user<>'vec_personal_propietario' OR session_user=current_user
       OR (pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
           =pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER'))
       -- Una sola concesión física, no una cadena de roles ni opciones delegables.
       OR (SELECT count(*) FROM pg_auth_members m JOIN pg_roles u ON u.oid=m.member
           WHERE u.rolname=session_user)<>1
       OR NOT EXISTS (SELECT 1 FROM pg_auth_members m
           JOIN pg_roles u ON u.oid=m.member JOIN pg_roles e ON e.oid=m.roleid
           WHERE u.rolname=session_user
             AND e.rolname IN ('vec_personal_ejecutor','vec_contratacion_temporal_ejecutor')
             AND m.admin_option IS FALSE AND m.inherit_option IS TRUE AND m.set_option IS FALSE
             AND NOT e.rolcanlogin AND e.rolinherit AND NOT e.rolsuper AND NOT e.rolcreatedb
             AND NOT e.rolcreaterole AND NOT e.rolreplication AND NOT e.rolbypassrls
             AND NOT EXISTS (SELECT 1 FROM pg_auth_members superior WHERE superior.member=e.oid))
       OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user
           AND (rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls
               OR NOT rolcanlogin OR NOT rolinherit))
       -- Lista cerrada incluso para membresías transitivas y roles VEC futuros.
       OR EXISTS (SELECT 1 FROM pg_roles WHERE left(rolname,4)='vec_'
           AND rolname<>session_user
           AND rolname NOT IN ('vec_personal_ejecutor','vec_contratacion_temporal_ejecutor')
           AND pg_has_role(session_user,oid,'MEMBER'))
       OR current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' THEN
        RAISE EXCEPTION 'lectura Personal denegada' USING ERRCODE='42501';
    END IF;
    -- Unidad sólo acota el permiso CT; no se asigna al registro Personal.
    IF p_unidad_ref IS NULL OR p_unidad_ref !~ '^ref:[0-9a-f]{64}$'
       OR p_unidad_ref='ref:'||repeat('0',64) THEN
        RAISE EXCEPTION 'unidad lectora V2 inválida' USING ERRCODE='22023';
    END IF;
    inicio:=clock_timestamp();
    FOREACH v IN ARRAY ARRAY[p_organizacion_ref,p_solicitud_ref,p_expediente_ref,p_resultado_ref,
        p_recibo_ref,p_relacion_ref,p_ocupacion_ref] LOOP
        IF vec_personal.referencia_alta_ejercicio_valida_v1(v) IS NOT TRUE THEN
            RAISE EXCEPTION 'selector Personal inválido' USING ERRCODE='22023';
        END IF;
    END LOOP;
    IF p_version_expediente IS NULL OR p_version_expediente NOT BETWEEN 1 AND 9007199254740991
       OR p_material_sha256 IS NULL OR p_material_sha256 !~ '^[a-f0-9]{64}$' OR p_material_sha256=repeat('0',64)
       OR p_persona_version IS NULL OR p_persona_version NOT BETWEEN 1 AND 9007199254740991
       OR p_perfil_version IS NULL OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991 THEN
        RAISE EXCEPTION 'selector o versiones Personal inválidos' USING ERRCODE='22023';
    END IF;
    -- Única canonicalización nueva: el mapa 2/9 de Recurso.HuellaContextoAutorizacionSHA256.
    -- El hash no incluye ref/módulo/tipo: se cotejan separadamente, nunca se infiere autoridad del SHA.
    canon_contexto:=convert_to('{"ambitos":{"organizacion_ref":'||vec_personal.texto_json_go_v1(p_organizacion_ref)||
        ',"unidad_ref":'||vec_personal.texto_json_go_v1(p_unidad_ref)||
        '},"atributos":{"expediente_ref":'||vec_personal.texto_json_go_v1(p_expediente_ref)||
        ',"material_sha256":'||vec_personal.texto_json_go_v1(p_material_sha256)||
        ',"ocupacion_ref":'||vec_personal.texto_json_go_v1(p_ocupacion_ref)||
        ',"recibo_ref":'||vec_personal.texto_json_go_v1(p_recibo_ref)||
        ',"relacion_ref":'||vec_personal.texto_json_go_v1(p_relacion_ref)||
        ',"resultado_ref":'||vec_personal.texto_json_go_v1(p_resultado_ref)||
        ',"solicitud_ref":'||vec_personal.texto_json_go_v1(p_solicitud_ref)||
        ',"tipo_validacion":"ejercicio_sintetico","version_expediente":'||
        vec_personal.texto_json_go_v1(p_version_expediente::text)||'}}','UTF8');
    h:=encode(sha256(canon_contexto),'hex');
    IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
       OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 1 AND 524288
       OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 1 AND 262144 THEN
        RAISE EXCEPTION 'autoridad lectora inválida' USING ERRCODE='42501';
    END IF;
    BEGIN
        d:=convert_from(p_decision,'UTF8')::jsonb;
        c:=convert_from(p_capacidad,'UTF8')::jsonb;
        actor:=convert_from(p_contexto,'UTF8')::jsonb;
        -- UTC textual sin redondear nanosegundos a microsegundos.
        FOREACH v IN ARRAY ARRAY[c->>'emitida_en',c->>'expira_en',d->>'emitida_en',d->>'valida_hasta',
            c->>'configuracion_expira_en',c->>'raiz_valida_hasta'] LOOP
            IF v IS NULL OR v !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$' THEN
                RAISE EXCEPTION 'ventana inválida';
            END IF;
        END LOOP;
        ci:=(c->>'emitida_en')::timestamptz; cf:=(c->>'expira_en')::timestamptz;
        di:=(d->>'emitida_en')::timestamptz; df:=(d->>'valida_hasta')::timestamptz;
        config_fin:=(c->>'configuracion_expira_en')::timestamptz;
        raiz_fin:=(c->>'raiz_valida_hasta')::timestamptz;
        limite:=least(cf,df,config_fin,raiz_fin);
    EXCEPTION WHEN OTHERS THEN
        RAISE EXCEPTION 'autoridad lectora inválida' USING ERRCODE='42501';
    END;
    IF jsonb_typeof(d) IS DISTINCT FROM 'object' OR jsonb_typeof(c) IS DISTINCT FROM 'object'
       OR jsonb_typeof(actor) IS DISTINCT FROM 'object'
       OR d->>'accion' IS DISTINCT FROM 'personal.alta_ejercicio.registro.consultar_incorporacion'
       OR d->>'modulo_id' IS DISTINCT FROM 'personal'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'registro_alta_ejercicio_incorporacion_v2'
       OR d->>'finalidad' IS DISTINCT FROM 'preparar_confirmacion_incorporacion_ct'
       OR d->>'recurso_ref' IS DISTINCT FROM p_resultado_ref
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h
       OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.lectura_incorporacion.v2'
       OR c->>'operacion' IS DISTINCT FROM d->>'accion'
       OR c->>'efecto_ref' IS DISTINCT FROM p_resultado_ref
       OR c->>'decision_ref' IS DISTINCT FROM d->>'decision_ref'
       OR c->>'huella_efecto_sha256' IS DISTINCT FROM h
       OR d->>'principal_id' IS DISTINCT FROM actor->>'principal_ref'
       OR d->>'perfil_activo_ref' IS DISTINCT FROM actor->>'perfil_activo_ref'
       OR vec_personal.referencia_alta_ejercicio_valida_v1(d->>'principal_id') IS NOT TRUE
       OR vec_personal.referencia_alta_ejercicio_valida_v1(d->>'perfil_activo_ref') IS NOT TRUE
       OR vec_personal.referencia_alta_ejercicio_valida_v1(d->>'decision_ref') IS NOT TRUE
       OR d#>>'{vinculo_autenticacion_actor,garantia_observada}' IS DISTINCT FROM 'alto'
       OR COALESCE(d#>>'{vinculo_autenticacion_actor,superficie}','') NOT IN ('interna_corporativa','administracion_privilegiada')
       OR NOT isfinite(inicio) OR NOT isfinite(ci) OR NOT isfinite(cf) OR NOT isfinite(di) OR NOT isfinite(df)
       OR NOT isfinite(config_fin) OR NOT isfinite(raiz_fin) OR inicio>=limite
       OR ci<di OR cf>df OR ci>=cf OR cf-ci>interval '5 seconds'
       OR inicio<ci OR inicio>=cf OR inicio<di OR inicio>=df THEN
        RAISE EXCEPTION 'permiso lector no ligado o vencido' USING ERRCODE='42501';
    END IF;
    -- AD3 verifica canon completo, firma/raíz, actor/perfil vivo, contexto y consume fresco.
    -- No se consulta ninguna fila de negocio antes de esta frontera.
    SELECT * INTO STRICT consumo FROM
        vec_autorizacion_atestada_v3.registrar_y_consumir_lectura_personal_incorporacion_v2_atestada(
            p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version::numeric,p_perfil_version::numeric,
            p_payload,p_sobre,p_evidencia,p_raiz);
    ahora:=clock_timestamp();
    IF consumo.consumo_nuevo IS DISTINCT FROM true OR consumo.decision_ref IS DISTINCT FROM d->>'decision_ref'
       OR consumo.efecto_ref IS DISTINCT FROM p_resultado_ref OR consumo.huella_efecto_sha256 IS DISTINCT FROM h
       OR consumo.consumo_huella_sha256 IS NULL OR consumo.consumo_huella_sha256 !~ '^[a-f0-9]{64}$'
       OR vec_personal.referencia_alta_ejercicio_valida_v1(consumo.auditoria_ref) IS NOT TRUE
       OR consumo.consumida_en IS NULL OR NOT isfinite(consumo.consumida_en)
       OR consumo.consumida_en<inicio OR consumo.consumida_en>ahora
       OR NOT isfinite(ahora) OR ahora<inicio OR ahora<ci OR ahora>=limite OR ahora<di THEN
        RAISE EXCEPTION 'consumo lector no vigente' USING ERRCODE='42501';
    END IF;
    SELECT * INTO r FROM vec_personal.registro_alta_ejercicio
        WHERE resultado_ref=p_resultado_ref FOR SHARE;
    IF NOT FOUND THEN RAISE EXCEPTION 'registro Personal no disponible' USING ERRCODE='55000'; END IF;
    IF r.organizacion_ref IS DISTINCT FROM p_organizacion_ref OR r.solicitud_ref IS DISTINCT FROM p_solicitud_ref
       OR r.expediente_ref IS DISTINCT FROM p_expediente_ref OR r.recibo_ref IS DISTINCT FROM p_recibo_ref
       OR r.relacion_ref IS DISTINCT FROM p_relacion_ref OR r.ocupacion_ref IS DISTINCT FROM p_ocupacion_ref
       OR r.material_sha256 IS DISTINCT FROM p_material_sha256
       OR r.material_json#>>'{Preparacion,Solicitud,version_expediente}' IS DISTINCT FROM p_version_expediente::text THEN
        discrepancia_nominal:=true;
        RAISE EXCEPTION 'selector Personal no coincide' USING ERRCODE='P1102';
    END IF;
    IF vec_personal.material_registro_alta_valido_v1(r.material_json) IS NOT TRUE THEN
        RAISE EXCEPTION 'registro Personal inconsistente' USING ERRCODE='55000';
    END IF;
    s:=r.material_json#>'{Preparacion,Solicitud}'; x:=r.material_json#>'{Preparacion,Vinculo}';
    w:=jsonb_build_object('Esquema','vec.personal.alta-ejercicio.material.v1','Material',r.material_json);
    mc:=vec_personal.material_alta_ejercicio_canonico_v1(w);
    sc:=vec_personal.solicitud_alta_ejercicio_canonica_v1(s);
    rc:=vec_personal.contexto_alta_ejercicio_canonico_v1(w);
    hm:=encode(sha256(mc),'hex'); hs:=encode(sha256(sc),'hex'); hr:=encode(sha256(rc),'hex');
    SELECT * INTO rel FROM vec_personal.relacion_alta_ejercicio WHERE relacion_ref=r.relacion_ref FOR SHARE;
    IF NOT FOUND THEN RAISE EXCEPTION 'registro Personal inconsistente' USING ERRCODE='55000'; END IF;
    SELECT * INTO ocu FROM vec_personal.ocupacion_alta_ejercicio WHERE ocupacion_ref=r.ocupacion_ref FOR SHARE;
    IF NOT FOUND THEN RAISE EXCEPTION 'registro Personal inconsistente' USING ERRCODE='55000'; END IF;
    SELECT * INTO aud FROM vec_personal.auditoria_alta_ejercicio WHERE auditoria_ref=r.auditoria_ref FOR SHARE;
    IF NOT FOUND THEN RAISE EXCEPTION 'registro Personal inconsistente' USING ERRCODE='55000'; END IF;
    SELECT * INTO ob FROM vec_personal.outbox_alta_ejercicio WHERE outbox_ref=r.outbox_ref FOR SHARE;
    IF NOT FOUND THEN RAISE EXCEPTION 'registro Personal inconsistente' USING ERRCODE='55000'; END IF;
    recibo:=jsonb_build_object(
        'resultado',jsonb_build_object('esquema','vec.contratacion-temporal.personal-rpt.alta.v1','contrato_version',1,
            'resultado_ref',r.resultado_ref,'recibo_ref',r.recibo_ref,'solicitud_ref',r.solicitud_ref,
            'correlacion_ref',s->>'correlacion_ref','idempotencia_ref',r.idempotencia_ref,
            'huella_solicitud_sha256',hs,'estado','confirmada','relacion_ref',r.relacion_ref,'ocupacion_ref',r.ocupacion_ref),
        'material_sha256',hm,'registrado_en',to_char(r.registrado_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'decision_original_ref',r.decision_original_ref,'auditoria_ref',r.auditoria_ref,'outbox_ref',r.outbox_ref,
        'ejercicio_sintetico',true,'firma_oficial',false,'eficacia_administrativa',false,
        'replay',false,'decision_consumida_ref',r.decision_original_ref);
    evento:=jsonb_build_object('esquema','vec.personal.alta-ejercicio.registrada.v1',
        'resultado_ref',r.resultado_ref,'recibo_ref',r.recibo_ref,'relacion_ref',r.relacion_ref,
        'ocupacion_ref',r.ocupacion_ref,'material_sha256',hm,
        'ejercicio_sintetico',true,'firma_oficial',false,'eficacia_administrativa',false);
    IF mc IS NULL OR sc IS NULL OR rc IS NULL OR octet_length(mc) NOT BETWEEN 1 AND 65536
       OR r.material_canonico IS DISTINCT FROM mc OR r.solicitud_canonica IS DISTINCT FROM sc
       OR r.contexto_canonico IS DISTINCT FROM rc OR r.material_sha256 IS DISTINCT FROM hm
       OR r.solicitud_sha256 IS DISTINCT FROM hs OR r.contexto_sha256 IS DISTINCT FROM hr
       OR r.organizacion_ref IS DISTINCT FROM r.material_json->>'OrganizacionRef'
       OR r.actor_ref IS DISTINCT FROM r.material_json->>'ActorRef'
       OR r.perfil_ref IS DISTINCT FROM r.material_json->>'PerfilRef'
       OR r.solicitud_ref IS DISTINCT FROM s->>'solicitud_ref' OR r.expediente_ref IS DISTINCT FROM s->>'expediente_ref'
       OR r.idempotencia_ref IS DISTINCT FROM s->>'idempotencia_ref' OR r.recibo_json IS DISTINCT FROM recibo
       OR r.ejercicio_sintetico IS DISTINCT FROM true OR r.firma_oficial IS DISTINCT FROM false
       OR r.eficacia_administrativa IS DISTINCT FROM false OR NOT isfinite(r.registrado_en) OR r.registrado_en>inicio
       OR r.decision_original_ref IS NOT DISTINCT FROM consumo.decision_ref
       OR rel.organizacion_ref IS DISTINCT FROM r.organizacion_ref
       OR rel.persona_sintetica_ref IS DISTINCT FROM x->>'PersonaSinteticaRef'
       OR rel.desde IS DISTINCT FROM vec_personal.fecha_registro_alta_pg_v1(x->>'Desde')
       OR rel.hasta IS DISTINCT FROM vec_personal.fecha_registro_alta_pg_v1(x->>'Hasta')
       OR rel.registrada_en IS DISTINCT FROM r.registrado_en
       OR rel.ejercicio_sintetico IS DISTINCT FROM true OR rel.firma_oficial IS DISTINCT FROM false
       OR rel.eficacia_administrativa IS DISTINCT FROM false
       OR ocu.relacion_ref IS DISTINCT FROM r.relacion_ref OR ocu.centro_ref IS DISTINCT FROM x->>'CentroRef'
       OR ocu.puesto_ref IS DISTINCT FROM x->>'PuestoRef' OR ocu.plaza_ref IS DISTINCT FROM x->>'PlazaRef'
       OR ocu.fuente_rpt IS DISTINCT FROM x->'FuenteRPT'
       OR aud.resultado_ref IS DISTINCT FROM r.resultado_ref OR aud.decision_ref IS DISTINCT FROM r.decision_original_ref
       OR aud.material_sha256 IS DISTINCT FROM hm OR aud.contexto_sha256 IS DISTINCT FROM hr
       OR aud.actor_ref IS DISTINCT FROM r.actor_ref OR aud.perfil_ref IS DISTINCT FROM r.perfil_ref
       OR aud.recuperacion IS DISTINCT FROM false OR aud.registrada_en IS DISTINCT FROM r.registrado_en
       OR NOT isfinite(aud.consumida_en) OR aud.consumida_en>aud.registrada_en
       OR aud.consumo_huella_sha256 IS NULL OR aud.consumo_huella_sha256 !~ '^[a-f0-9]{64}$'
       OR vec_personal.referencia_alta_ejercicio_valida_v1(aud.auditoria_v3_ref) IS NOT TRUE
       OR ob.resultado_ref IS DISTINCT FROM r.resultado_ref OR ob.payload IS DISTINCT FROM evento
       OR ob.tipo_evento IS DISTINCT FROM 'personal.alta_ejercicio.registrada.v1'
       OR ob.payload_sha256 IS DISTINCT FROM encode(sha256(convert_to(evento::text,'UTF8')),'hex')
       OR ob.registrada_en IS DISTINCT FROM r.registrado_en THEN
        RAISE EXCEPTION 'registro Personal inconsistente' USING ERRCODE='55000';
    END IF;
    leida:=clock_timestamp();
    IF NOT isfinite(leida) OR leida<ahora OR leida<ci OR leida>=limite OR leida<di THEN
        RAISE EXCEPTION 'lectura Personal vencida' USING ERRCODE='42501';
    END IF;
    ref_auditoria:='auditoria:personal:lectura:'||gen_random_uuid()::text;
    INSERT INTO vec_personal.auditoria_lectura_incorporacion VALUES (
        ref_auditoria,r.resultado_ref,consumo.decision_ref,consumo.consumo_huella_sha256,consumo.auditoria_ref,
        h,hm,d->>'principal_id',d->>'perfil_activo_ref',consumo.consumida_en,leida);
    respuesta:=jsonb_build_object('registro',jsonb_build_object(
        'solicitud',s,'resultado',r.recibo_json->'resultado',
        'material_canonico',replace(encode(r.material_canonico,'base64'),E'\n',''),
        'material_sha256',r.material_sha256,'registrado_en',r.recibo_json->'registrado_en',
        'decision_original_ref',r.decision_original_ref,'auditoria_ref',r.auditoria_ref,'outbox_ref',r.outbox_ref,
        'ejercicio_sintetico',true,'firma_oficial',false,'eficacia_administrativa',false),
        'decision_lectura_ref',consumo.decision_ref,'consumo_huella_sha256',consumo.consumo_huella_sha256,
        'auditoria_lectura_ref',ref_auditoria,'leida_en',to_char(leida AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
    ahora:=clock_timestamp();
    IF NOT isfinite(ahora) OR ahora<leida OR ahora<ci OR ahora>=limite OR ahora<di THEN
        RAISE EXCEPTION 'lectura Personal vencida' USING ERRCODE='42501';
    END IF;
    -- El adaptador propietario NO entrega este DTO hasta COMMIT y revalidación de su orden.
    RETURN respuesta;
EXCEPTION WHEN OTHERS THEN
    -- Suprime DETAIL de casts/constraints (podría contener material o actor).
    -- OTHERS no intercepta query_canceled; nunca se reintenta y todo consumo/audit revierte.
    IF discrepancia_nominal AND SQLSTATE='P1102' THEN
        RAISE EXCEPTION 'selector Personal no coincide' USING ERRCODE='P1102';
    END IF;
    RAISE EXCEPTION 'lectura Personal no disponible'
        USING ERRCODE=CASE WHEN SQLSTATE='P1102' THEN '55000' ELSE SQLSTATE END;
END
$lector$;

COMMENT ON FUNCTION vec_personal.acreditar_alta_ejercicio_v2(text,text,text,bigint,text,text,text,text,text,text,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)
    IS 'Personal000005:lector_incorporacion:v2:ambitos_org_unidad';
DO $acl_funcion$
DECLARE a record; f oid:='vec_personal.acreditar_alta_ejercicio_v2(text,text,text,bigint,text,text,text,text,text,text,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
    EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f::regprocedure);
    FOR a IN SELECT DISTINCT grantee FROM pg_proc p,LATERAL aclexplode(p.proacl)
        WHERE p.oid=f AND grantee<>p.proowner LOOP
        EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f::regprocedure,pg_get_userbyid(a.grantee));
    END LOOP;
    EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_personal_ejecutor,vec_contratacion_temporal_propietario',f::regprocedure);
END
$acl_funcion$;
-- USAGE de 000003 se exige pero nunca se modifica; V1 permanece accesible.
COMMIT;
