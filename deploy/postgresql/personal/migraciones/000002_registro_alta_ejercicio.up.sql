\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000002:registro:v1',0));

DO $dependencias$
DECLARE v_nombre text;
BEGIN
    FOREACH v_nombre IN ARRAY ARRAY['material_alta_ejercicio_canonico_v1',
        'solicitud_alta_ejercicio_canonica_v1','contexto_alta_ejercicio_canonico_v1'] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
            WHERE n.nspname='vec_personal' AND p.proname=v_nombre
              AND p.pronargs=1 AND p.proargtypes[0]='jsonb'::regtype AND p.prorettype='bytea'::regtype
              AND p.proowner=current_user::regrole AND p.provolatile='i' AND NOT p.prosecdef
              AND has_function_privilege(current_user,p.oid,'EXECUTE')) THEN
            RAISE EXCEPTION 'codec Personal requerido' USING ERRCODE='55000';
        END IF;
    END LOOP;
    IF NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
        WHERE n.nspname='vec_personal' AND p.proname='fecha_civil_go_valida_v1'
          AND p.pronargs=1 AND p.proargtypes[0]='text'::regtype AND p.prorettype='boolean'::regtype
          AND p.proowner=current_user::regrole AND p.provolatile='i' AND NOT p.prosecdef
          AND has_function_privilege(current_user,p.oid,'EXECUTE')) THEN
        RAISE EXCEPTION 'codec de fecha civil Personal requerido' USING ERRCODE='55000';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=to_regprocedure(
        'vec_autorizacion_atestada_v3.registrar_y_consumir_alta_personal_ejercicio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
        AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
        AND has_function_privilege(current_user,p.oid,'EXECUTE'))
       OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
        WHERE n.nspname='vec_personal' AND p.proname='registrar_alta_ejercicio_v1') THEN
        RAISE EXCEPTION 'dependencia AD3-26 o registro Personal incompatible' USING ERRCODE='55000';
    END IF;
END
$dependencias$;

-- Validación del contrato de entrada, separada del codec puro de 000001.
-- Ningún dato de este JSON concede autoridad; se liga después a V3 real.
CREATE FUNCTION vec_personal.claves_registro_alta_v1(obj jsonb, claves text[]) RETURNS boolean
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog
AS $claves$
BEGIN
    IF jsonb_typeof(obj) IS DISTINCT FROM 'object' THEN RETURN false; END IF;
    RETURN (SELECT count(*) FROM jsonb_object_keys(obj))=cardinality(claves)
       AND NOT EXISTS (SELECT 1 FROM jsonb_object_keys(obj) k WHERE NOT k=ANY(claves));
END
$claves$;

-- Go usa año astronómico 0000; PostgreSQL lo representa como 1 a. C.
-- El codec conserva la gramática y el calendario; aquí solo cambia la representación.
CREATE FUNCTION vec_personal.fecha_registro_alta_pg_v1(valor text) RETURNS date
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog
AS $fecha$
DECLARE anio integer;
BEGIN
    IF vec_personal.fecha_civil_go_valida_v1(valor) IS NOT TRUE THEN
        RAISE EXCEPTION 'fecha civil Personal inválida' USING ERRCODE='22023';
    END IF;
    anio:=substring(valor FROM 1 FOR 4)::integer;
    RETURN make_date(CASE WHEN anio=0 THEN -1 ELSE anio END,
        substring(valor FROM 6 FOR 2)::integer,substring(valor FROM 9 FOR 2)::integer);
END
$fecha$;

CREATE FUNCTION vec_personal.material_registro_alta_valido_v1(m jsonb) RETURNS boolean
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog
AS $material$
DECLARE p jsonb; s jsonb; f jsonb; x jsonb; r jsonb; v jsonb; o jsonb; k text; inicio date; fin date;
BEGIN
    IF m IS NULL OR octet_length(m::text)>65536
       OR NOT vec_personal.claves_registro_alta_v1(m,ARRAY['Preparacion','OrganizacionRef','ActorRef','PerfilRef']) THEN RETURN false; END IF;
    p:=m->'Preparacion'; s:=p->'Solicitud'; f:=p->'Fuente'; x:=p->'Vinculo'; r:=s->'fuente_rpt';
    IF NOT vec_personal.claves_registro_alta_v1(p,ARRAY['Solicitud','Fuente','Vinculo'])
       OR NOT vec_personal.claves_registro_alta_v1(s,ARRAY['esquema','contrato_version','solicitud_ref',
           'expediente_ref','version_expediente','capacidad_ref','correlacion_ref','idempotencia_ref','fuente_rpt','puesto_ref','plaza_ref'])
       OR NOT vec_personal.claves_registro_alta_v1(f,ARRAY['Referencia','Version','HuellaSHA256'])
       OR NOT vec_personal.claves_registro_alta_v1(x,ARRAY['PersonaSinteticaRef','CentroRef','PuestoRef','PlazaRef',
           'FuenteRPT','Desde','Hasta','AntecedenteEjercicioRef'])
       OR NOT vec_personal.claves_registro_alta_v1(r,ARRAY['referencia','version','huella_sha256'])
       OR x->'FuenteRPT' IS DISTINCT FROM r
       OR s->'esquema' IS DISTINCT FROM to_jsonb('vec.contratacion-temporal.personal-rpt.alta.v1'::text)
       OR s->'contrato_version' IS DISTINCT FROM '1'::jsonb THEN RETURN false; END IF;
    FOR o,k IN SELECT * FROM (VALUES (m,'OrganizacionRef'),(m,'ActorRef'),(m,'PerfilRef'),
        (s,'solicitud_ref'),(s,'expediente_ref'),(s,'capacidad_ref'),(s,'correlacion_ref'),(s,'idempotencia_ref'),
        (s,'puesto_ref'),(s,'plaza_ref'),(f,'Referencia'),(r,'referencia'),
        (x,'PersonaSinteticaRef'),(x,'CentroRef'),(x,'PuestoRef'),(x,'PlazaRef')) AS campos(obj,clave) LOOP
        IF jsonb_typeof(o->k) IS DISTINCT FROM 'string'
           OR (o->>k) !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN RETURN false; END IF;
    END LOOP;
    FOR v IN SELECT valor FROM (VALUES (s->'contrato_version'),(s->'version_expediente'),
        (f->'Version'),(r->'version')) AS versiones(valor) LOOP
        IF jsonb_typeof(v) IS DISTINCT FROM 'number' OR v::text !~ '^[1-9][0-9]{0,15}$' THEN RETURN false; END IF;
        IF (v::text)::numeric>9007199254740991 THEN RETURN false; END IF;
    END LOOP;
    FOR v IN SELECT valor FROM (VALUES (f->'HuellaSHA256'),(r->'huella_sha256')) AS huellas(valor) LOOP
        IF jsonb_typeof(v) IS DISTINCT FROM 'string' OR (v#>>'{}') !~ '^[a-f0-9]{64}$'
           OR v#>>'{}'=repeat('0',64) THEN RETURN false; END IF;
    END LOOP;
    IF x->>'PersonaSinteticaRef' NOT LIKE 'persona:ejercicio:%'
       OR x->>'PersonaSinteticaRef'=m->>'ActorRef'
       OR x->'PuestoRef' IS DISTINCT FROM s->'puesto_ref' OR x->'PlazaRef' IS DISTINCT FROM s->'plaza_ref'
       OR jsonb_typeof(x->'AntecedenteEjercicioRef') IS DISTINCT FROM 'string'
       OR (x->>'AntecedenteEjercicioRef'<>'' AND x->>'AntecedenteEjercicioRef' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR jsonb_typeof(x->'Desde') IS DISTINCT FROM 'string' OR jsonb_typeof(x->'Hasta') IS DISTINCT FROM 'string'
       OR x->>'Desde' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' OR x->>'Hasta' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN RETURN false; END IF;
    inicio:=vec_personal.fecha_registro_alta_pg_v1(x->>'Desde');
    fin:=vec_personal.fecha_registro_alta_pg_v1(x->>'Hasta');
    RETURN isfinite(inicio) AND isfinite(fin) AND fin>=inicio;
EXCEPTION WHEN data_exception THEN RETURN false;
END
$material$;

CREATE FUNCTION vec_personal.rechazar_mutacion_alta_ejercicio_v1() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog
AS $inmutable$
BEGIN
    RAISE EXCEPTION 'historia Personal de ejercicio inmutable' USING ERRCODE='55000';
END
$inmutable$;

CREATE TABLE vec_personal.relacion_alta_ejercicio (
    relacion_ref text PRIMARY KEY CHECK (relacion_ref ~ '^relacion:personal:ejercicio:[0-9a-f-]{36}$'),
    organizacion_ref text NOT NULL,
    persona_sintetica_ref text NOT NULL CHECK (persona_sintetica_ref LIKE 'persona:ejercicio:%'),
    desde date NOT NULL CHECK (isfinite(desde)), hasta date NOT NULL CHECK (isfinite(hasta) AND hasta>=desde),
    registrada_en timestamptz(6) NOT NULL CHECK (isfinite(registrada_en)),
    ejercicio_sintetico boolean NOT NULL CHECK (ejercicio_sintetico),
    firma_oficial boolean NOT NULL CHECK (NOT firma_oficial),
    eficacia_administrativa boolean NOT NULL CHECK (NOT eficacia_administrativa)
);
CREATE TABLE vec_personal.ocupacion_alta_ejercicio (
    ocupacion_ref text PRIMARY KEY CHECK (ocupacion_ref ~ '^ocupacion:personal:ejercicio:[0-9a-f-]{36}$'),
    relacion_ref text NOT NULL UNIQUE REFERENCES vec_personal.relacion_alta_ejercicio,
    centro_ref text NOT NULL, puesto_ref text NOT NULL, plaza_ref text NOT NULL,
    fuente_rpt jsonb NOT NULL CHECK (jsonb_typeof(fuente_rpt)='object'),
    UNIQUE (ocupacion_ref,relacion_ref)
);

-- Este registro ES el recibo de Personal y su material original, no historia CT ni AD3.
CREATE TABLE vec_personal.registro_alta_ejercicio (
    resultado_ref text PRIMARY KEY CHECK (resultado_ref ~ '^resultado:personal:ejercicio:[0-9a-f-]{36}$'),
    recibo_ref text NOT NULL UNIQUE,
    organizacion_ref text NOT NULL, solicitud_ref text NOT NULL, expediente_ref text NOT NULL,
    idempotencia_ref text NOT NULL, actor_ref text NOT NULL, perfil_ref text NOT NULL,
    material_json jsonb NOT NULL CHECK (vec_personal.material_registro_alta_valido_v1(material_json) IS TRUE),
    material_canonico bytea NOT NULL CHECK (octet_length(material_canonico) BETWEEN 1 AND 65536),
    material_sha256 text NOT NULL CHECK (material_sha256=encode(sha256(material_canonico),'hex')),
    solicitud_canonica bytea NOT NULL,
    solicitud_sha256 text NOT NULL CHECK (solicitud_sha256=encode(sha256(solicitud_canonica),'hex')),
    contexto_canonico bytea NOT NULL,
    contexto_sha256 text NOT NULL CHECK (contexto_sha256=encode(sha256(contexto_canonico),'hex')),
    relacion_ref text NOT NULL UNIQUE REFERENCES vec_personal.relacion_alta_ejercicio,
    ocupacion_ref text NOT NULL UNIQUE,
    decision_original_ref text NOT NULL UNIQUE,
    auditoria_ref text NOT NULL UNIQUE, outbox_ref text NOT NULL UNIQUE,
    registrado_en timestamptz(6) NOT NULL CHECK (isfinite(registrado_en)
        AND registrado_en>'0001-01-01T00:00:00Z'::timestamptz AND registrado_en<'10000-01-01T00:00:00Z'::timestamptz),
    ejercicio_sintetico boolean NOT NULL CHECK (ejercicio_sintetico),
    firma_oficial boolean NOT NULL CHECK (NOT firma_oficial),
    eficacia_administrativa boolean NOT NULL CHECK (NOT eficacia_administrativa),
    recibo_json jsonb NOT NULL,
    UNIQUE (organizacion_ref,idempotencia_ref), UNIQUE (organizacion_ref,solicitud_ref), UNIQUE (organizacion_ref,expediente_ref),
    UNIQUE (resultado_ref,material_sha256,contexto_sha256,actor_ref,perfil_ref),
    FOREIGN KEY (ocupacion_ref,relacion_ref) REFERENCES vec_personal.ocupacion_alta_ejercicio(ocupacion_ref,relacion_ref),
    CHECK (material_canonico=vec_personal.material_alta_ejercicio_canonico_v1(
        jsonb_build_object('Esquema','vec.personal.alta-ejercicio.material.v1','Material',material_json))),
    CHECK (solicitud_canonica=vec_personal.solicitud_alta_ejercicio_canonica_v1(material_json#>'{Preparacion,Solicitud}')),
    CHECK (contexto_canonico=vec_personal.contexto_alta_ejercicio_canonico_v1(
        jsonb_build_object('Esquema','vec.personal.alta-ejercicio.material.v1','Material',material_json))),
    CHECK (organizacion_ref=material_json->>'OrganizacionRef' AND actor_ref=material_json->>'ActorRef'
        AND perfil_ref=material_json->>'PerfilRef' AND solicitud_ref=material_json#>>'{Preparacion,Solicitud,solicitud_ref}'
        AND expediente_ref=material_json#>>'{Preparacion,Solicitud,expediente_ref}'
        AND idempotencia_ref=material_json#>>'{Preparacion,Solicitud,idempotencia_ref}'),
    CHECK (recibo_json=jsonb_build_object(
        'resultado',jsonb_build_object('esquema','vec.contratacion-temporal.personal-rpt.alta.v1','contrato_version',1,
            'resultado_ref',resultado_ref,'recibo_ref',recibo_ref,'solicitud_ref',solicitud_ref,
            'correlacion_ref',material_json#>>'{Preparacion,Solicitud,correlacion_ref}','idempotencia_ref',idempotencia_ref,
            'huella_solicitud_sha256',solicitud_sha256,'estado','confirmada','relacion_ref',relacion_ref,'ocupacion_ref',ocupacion_ref),
        'material_sha256',material_sha256,'registrado_en',to_char(registrado_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'decision_original_ref',decision_original_ref,'auditoria_ref',auditoria_ref,'outbox_ref',outbox_ref,
        'ejercicio_sintetico',true,'firma_oficial',false,'eficacia_administrativa',false,
        'replay',false,'decision_consumida_ref',decision_original_ref))
);

-- Cada recuperación añade su propio acceso autorizado, sin reescribir el recibo original.
CREATE TABLE vec_personal.auditoria_alta_ejercicio (
    auditoria_ref text PRIMARY KEY,
    resultado_ref text NOT NULL REFERENCES vec_personal.registro_alta_ejercicio,
    decision_ref text NOT NULL UNIQUE, consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[a-f0-9]{64}$'),
    auditoria_v3_ref text NOT NULL UNIQUE,
    material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[a-f0-9]{64}$'),
    contexto_sha256 text NOT NULL CHECK (contexto_sha256 ~ '^[a-f0-9]{64}$'),
    actor_ref text NOT NULL, perfil_ref text NOT NULL, recuperacion boolean NOT NULL,
    consumida_en timestamptz(6) NOT NULL CHECK (isfinite(consumida_en)),
    registrada_en timestamptz(6) NOT NULL CHECK (isfinite(registrada_en) AND registrada_en>=consumida_en),
    UNIQUE (auditoria_ref,resultado_ref,decision_ref),
    FOREIGN KEY (resultado_ref,material_sha256,contexto_sha256,actor_ref,perfil_ref)
        REFERENCES vec_personal.registro_alta_ejercicio(resultado_ref,material_sha256,contexto_sha256,actor_ref,perfil_ref)
);
CREATE TABLE vec_personal.outbox_alta_ejercicio (
    outbox_ref text PRIMARY KEY,
    resultado_ref text NOT NULL UNIQUE REFERENCES vec_personal.registro_alta_ejercicio,
    tipo_evento text NOT NULL CHECK (tipo_evento='personal.alta_ejercicio.registrada.v1'),
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload)='object'),
    payload_sha256 text NOT NULL CHECK (payload_sha256=encode(sha256(convert_to(payload::text,'UTF8')),'hex')),
    registrada_en timestamptz(6) NOT NULL CHECK (isfinite(registrada_en)),
    UNIQUE (outbox_ref,resultado_ref)
);
-- Ciclo diferido: no puede confirmarse un recibo sin su auditoría y su outbox.
ALTER TABLE vec_personal.registro_alta_ejercicio
    ADD CONSTRAINT registro_alta_auditoria_fk FOREIGN KEY (auditoria_ref,resultado_ref,decision_original_ref)
        REFERENCES vec_personal.auditoria_alta_ejercicio(auditoria_ref,resultado_ref,decision_ref) DEFERRABLE INITIALLY DEFERRED,
    ADD CONSTRAINT registro_alta_outbox_fk FOREIGN KEY (outbox_ref,resultado_ref)
        REFERENCES vec_personal.outbox_alta_ejercicio(outbox_ref,resultado_ref) DEFERRABLE INITIALLY DEFERRED;

DO $tablas$
DECLARE t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['relacion_alta_ejercicio','ocupacion_alta_ejercicio','registro_alta_ejercicio',
        'auditoria_alta_ejercicio','outbox_alta_ejercicio'] LOOP
        EXECUTE format('ALTER TABLE vec_personal.%I ENABLE ROW LEVEL SECURITY',t);
        EXECUTE format('ALTER TABLE vec_personal.%I FORCE ROW LEVEL SECURITY',t);
        EXECUTE format('CREATE POLICY propietario ON vec_personal.%I TO vec_personal_propietario USING (true) WITH CHECK (true)',t);
        EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_personal.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_alta_ejercicio_v1()',t);
        EXECUTE format('REVOKE ALL ON TABLE vec_personal.%I FROM PUBLIC,vec_personal_ejecutor,vec_personal_migrador',t);
    END LOOP;
END
$tablas$;

CREATE FUNCTION vec_personal.registrar_alta_ejercicio_v1(
    p_material jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version bigint,p_perfil_version bigint,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s'
AS $registro$
DECLARE
    s jsonb; x jsonb; d jsonb; c jsonb; actor jsonb; envoltorio jsonb;
    material bytea; solicitud bytea; recurso bytea; hm text; hs text; hr text;
    consumo record; previa vec_personal.registro_alta_ejercicio%ROWTYPE;
    ahora timestamptz(6); registrada timestamptz(6); v_lock bigint;
    id text; ref_resultado text; ref_recibo text; ref_relacion text; ref_ocupacion text; ref_auditoria text; ref_outbox text;
    recuperacion boolean; respuesta jsonb; evento jsonb;
BEGIN
    IF current_user<>'vec_personal_propietario' OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
       OR current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' THEN
        RAISE EXCEPTION 'alta Personal denegada' USING ERRCODE='42501';
    END IF;
    PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
    IF vec_personal.material_registro_alta_valido_v1(p_material) IS NOT TRUE THEN
        RAISE EXCEPTION 'material de alta Personal inválido' USING ERRCODE='22023';
    END IF;
    s:=p_material#>'{Preparacion,Solicitud}'; x:=p_material#>'{Preparacion,Vinculo}';
    envoltorio:=jsonb_build_object('Esquema','vec.personal.alta-ejercicio.material.v1','Material',p_material);
    material:=vec_personal.material_alta_ejercicio_canonico_v1(envoltorio);
    solicitud:=vec_personal.solicitud_alta_ejercicio_canonica_v1(s);
    recurso:=vec_personal.contexto_alta_ejercicio_canonico_v1(envoltorio);
    IF material IS NULL OR octet_length(material) NOT BETWEEN 1 AND 65536
       OR solicitud IS NULL OR octet_length(solicitud) NOT BETWEEN 1 AND 65536
       OR recurso IS NULL OR octet_length(recurso) NOT BETWEEN 1 AND 65536 THEN
        RAISE EXCEPTION 'codec de alta Personal no disponible' USING ERRCODE='55000';
    END IF;
    hm:=encode(sha256(material),'hex'); hs:=encode(sha256(solicitud),'hex'); hr:=encode(sha256(recurso),'hex');
    IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
       OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 1 AND 262144 THEN
        RAISE EXCEPTION 'autoridad Personal inválida' USING ERRCODE='42501';
    END IF;
    BEGIN
        d:=convert_from(p_decision,'UTF8')::jsonb; actor:=convert_from(p_contexto,'UTF8')::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'autoridad Personal inválida' USING ERRCODE='42501';
    END;
    IF d->>'accion' IS DISTINCT FROM 'personal.alta_ejercicio.registrar'
       OR d->>'modulo_id' IS DISTINCT FROM 'personal' OR d->>'tipo_recurso' IS DISTINCT FROM 'alta_personal_ejercicio'
       OR d->>'finalidad' IS DISTINCT FROM 'registrar_relacion_ocupacion_sinteticas'
       OR d->>'recurso_ref' IS DISTINCT FROM s->>'solicitud_ref'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM hr
       OR d->>'principal_id' IS DISTINCT FROM p_material->>'ActorRef'
       OR d->>'perfil_activo_ref' IS DISTINCT FROM p_material->>'PerfilRef'
       OR actor->>'principal_ref' IS DISTINCT FROM p_material->>'ActorRef'
       OR actor->>'perfil_activo_ref' IS DISTINCT FROM p_material->>'PerfilRef'
       OR d#>>'{vinculo_autenticacion_actor,garantia_observada}' IS DISTINCT FROM 'alto'
       OR (d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
           AND d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'administracion_privilegiada') THEN
        RAISE EXCEPTION 'permiso de alta Personal divergente' USING ERRCODE='42501';
    END IF;
    -- Primera autoridad efectiva: el wrapper verifica y consume V3 en ESTA transacción.
    SELECT * INTO STRICT consumo
      FROM vec_autorizacion_atestada_v3.registrar_y_consumir_alta_personal_ejercicio_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version::numeric,p_perfil_version::numeric,
        p_payload,p_sobre,p_evidencia,p_raiz);
    IF consumo.consumo_nuevo IS NOT TRUE OR consumo.decision_ref IS DISTINCT FROM d->>'decision_ref'
       OR consumo.efecto_ref IS DISTINCT FROM s->>'solicitud_ref' OR consumo.huella_efecto_sha256 IS DISTINCT FROM hr THEN
        RAISE EXCEPTION 'consumo Personal no ligado' USING ERRCODE='42501';
    END IF;
    c:=convert_from(p_capacidad,'UTF8')::jsonb;
    -- Mismo orden por hash incluso si coinciden claves; serialización/idempotencia
    -- no se traduce de forma genérica desde 23505/40001 a conflicto de negocio.
    FOR v_lock IN SELECT DISTINCT hashtextextended(clave,0) FROM unnest(ARRAY[
        'vec_personal:alta:idempotencia:'||jsonb_build_array(p_material->>'OrganizacionRef',s->>'idempotencia_ref')::text,
        'vec_personal:alta:solicitud:'||jsonb_build_array(p_material->>'OrganizacionRef',s->>'solicitud_ref')::text,
        'vec_personal:alta:expediente:'||jsonb_build_array(p_material->>'OrganizacionRef',s->>'expediente_ref')::text
    ]) AS claves(clave) ORDER BY 1 LOOP
        PERFORM pg_advisory_xact_lock(v_lock);
    END LOOP;
    ahora:=clock_timestamp();
    IF consumo.consumida_en IS NULL OR NOT isfinite(consumo.consumida_en) OR consumo.consumida_en>ahora
       OR ahora>=(c->>'expira_en')::timestamptz OR ahora>=(d->>'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION 'vigencia de alta Personal agotada' USING ERRCODE='42501';
    END IF;
    SELECT * INTO previa FROM vec_personal.registro_alta_ejercicio
     WHERE organizacion_ref=p_material->>'OrganizacionRef' AND idempotencia_ref=s->>'idempotencia_ref';
    recuperacion:=FOUND;
    ref_auditoria:='auditoria:personal:ejercicio:'||gen_random_uuid()::text;
    IF recuperacion THEN
        IF previa.material_json IS DISTINCT FROM p_material OR previa.material_canonico IS DISTINCT FROM material
           OR previa.actor_ref IS DISTINCT FROM d->>'principal_id' OR previa.perfil_ref IS DISTINCT FROM d->>'perfil_activo_ref' THEN
            RAISE EXCEPTION 'conflicto de alta Personal' USING ERRCODE='P1102';
        END IF;
        IF previa.material_sha256 IS DISTINCT FROM hm OR previa.material_sha256 IS DISTINCT FROM encode(sha256(previa.material_canonico),'hex')
           OR previa.solicitud_canonica IS DISTINCT FROM solicitud OR previa.contexto_canonico IS DISTINCT FROM recurso
           OR previa.decision_original_ref=consumo.decision_ref OR previa.registrado_en>ahora
           OR NOT EXISTS (SELECT 1 FROM vec_personal.relacion_alta_ejercicio r
                JOIN vec_personal.ocupacion_alta_ejercicio o USING(relacion_ref)
                WHERE r.relacion_ref=previa.relacion_ref AND o.ocupacion_ref=previa.ocupacion_ref
                  AND r.organizacion_ref=previa.organizacion_ref AND r.persona_sintetica_ref=x->>'PersonaSinteticaRef'
                  AND r.desde=vec_personal.fecha_registro_alta_pg_v1(x->>'Desde')
                  AND r.hasta=vec_personal.fecha_registro_alta_pg_v1(x->>'Hasta')
                  AND o.centro_ref=x->>'CentroRef' AND o.puesto_ref=x->>'PuestoRef' AND o.plaza_ref=x->>'PlazaRef'
                  AND o.fuente_rpt=x->'FuenteRPT') THEN
            RAISE EXCEPTION 'historia Personal inconsistente' USING ERRCODE='55000';
        END IF;
        ref_resultado:=previa.resultado_ref;
        respuesta:=previa.recibo_json||jsonb_build_object('replay',true,'decision_consumida_ref',consumo.decision_ref);
    ELSE
        IF EXISTS (SELECT 1 FROM vec_personal.registro_alta_ejercicio
            WHERE organizacion_ref=p_material->>'OrganizacionRef'
              AND (solicitud_ref=s->>'solicitud_ref' OR expediente_ref=s->>'expediente_ref')) THEN
            RAISE EXCEPTION 'conflicto de alta Personal' USING ERRCODE='P1102';
        END IF;
        id:=gen_random_uuid()::text; registrada:=ahora;
        ref_resultado:='resultado:personal:ejercicio:'||id; ref_recibo:='recibo:personal:ejercicio:'||id;
        ref_relacion:='relacion:personal:ejercicio:'||id; ref_ocupacion:='ocupacion:personal:ejercicio:'||id;
        ref_outbox:='outbox:personal:ejercicio:'||id;
        respuesta:=jsonb_build_object('resultado',jsonb_build_object(
            'esquema','vec.contratacion-temporal.personal-rpt.alta.v1','contrato_version',1,
            'resultado_ref',ref_resultado,'recibo_ref',ref_recibo,'solicitud_ref',s->>'solicitud_ref',
            'correlacion_ref',s->>'correlacion_ref','idempotencia_ref',s->>'idempotencia_ref',
            'huella_solicitud_sha256',hs,'estado','confirmada','relacion_ref',ref_relacion,'ocupacion_ref',ref_ocupacion),
            'material_sha256',hm,'registrado_en',to_char(registrada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
            'decision_original_ref',consumo.decision_ref,'auditoria_ref',ref_auditoria,'outbox_ref',ref_outbox,
            'ejercicio_sintetico',true,'firma_oficial',false,'eficacia_administrativa',false,
            'replay',false,'decision_consumida_ref',consumo.decision_ref);
        INSERT INTO vec_personal.relacion_alta_ejercicio VALUES
            (ref_relacion,p_material->>'OrganizacionRef',x->>'PersonaSinteticaRef',
             vec_personal.fecha_registro_alta_pg_v1(x->>'Desde'),vec_personal.fecha_registro_alta_pg_v1(x->>'Hasta'),
             registrada,true,false,false);
        INSERT INTO vec_personal.ocupacion_alta_ejercicio VALUES
            (ref_ocupacion,ref_relacion,x->>'CentroRef',x->>'PuestoRef',x->>'PlazaRef',x->'FuenteRPT');
        INSERT INTO vec_personal.registro_alta_ejercicio VALUES
            (ref_resultado,ref_recibo,p_material->>'OrganizacionRef',s->>'solicitud_ref',s->>'expediente_ref',
             s->>'idempotencia_ref',d->>'principal_id',d->>'perfil_activo_ref',p_material,material,hm,solicitud,hs,recurso,hr,
             ref_relacion,ref_ocupacion,consumo.decision_ref,ref_auditoria,ref_outbox,registrada,true,false,false,respuesta);
        evento:=jsonb_build_object('esquema','vec.personal.alta-ejercicio.registrada.v1','resultado_ref',ref_resultado,
            'recibo_ref',ref_recibo,'relacion_ref',ref_relacion,'ocupacion_ref',ref_ocupacion,'material_sha256',hm,
            'ejercicio_sintetico',true,'firma_oficial',false,'eficacia_administrativa',false);
        INSERT INTO vec_personal.outbox_alta_ejercicio VALUES
            (ref_outbox,ref_resultado,'personal.alta_ejercicio.registrada.v1',evento,
             encode(sha256(convert_to(evento::text,'UTF8')),'hex'),registrada);
    END IF;
    INSERT INTO vec_personal.auditoria_alta_ejercicio VALUES
        (ref_auditoria,ref_resultado,consumo.decision_ref,consumo.consumo_huella_sha256,consumo.auditoria_ref,
         hm,hr,d->>'principal_id',d->>'perfil_activo_ref',recuperacion,consumo.consumida_en,ahora);
    -- Una lectura/bloqueo/escritura lenta no prolonga la autorización consumida.
    ahora:=clock_timestamp();
    IF ahora<(c->>'emitida_en')::timestamptz OR ahora>=(c->>'expira_en')::timestamptz
       OR ahora>=(d->>'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION 'vigencia de alta Personal agotada' USING ERRCODE='42501';
    END IF;
    RETURN respuesta;
END
$registro$;

REVOKE ALL ON FUNCTION vec_personal.claves_registro_alta_v1(jsonb,text[]),
    vec_personal.fecha_registro_alta_pg_v1(text),
    vec_personal.material_registro_alta_valido_v1(jsonb),vec_personal.rechazar_mutacion_alta_ejercicio_v1(),
    vec_personal.registrar_alta_ejercicio_v1(jsonb,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)
    FROM PUBLIC,vec_personal_migrador,vec_personal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_personal.registrar_alta_ejercicio_v1(jsonb,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)
    TO vec_personal_ejecutor;

DO $acl$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_class t CROSS JOIN LATERAL aclexplode(coalesce(t.relacl,acldefault('r',t.relowner))) a
        WHERE t.relnamespace='vec_personal'::regnamespace AND t.relname=ANY(ARRAY['relacion_alta_ejercicio',
            'ocupacion_alta_ejercicio','registro_alta_ejercicio','auditoria_alta_ejercicio','outbox_alta_ejercicio'])
          AND (t.relowner<>'vec_personal_propietario'::regrole OR NOT t.relrowsecurity OR NOT t.relforcerowsecurity
               OR a.grantee<>'vec_personal_propietario'::regrole))
       OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
        WHERE p.pronamespace='vec_personal'::regnamespace AND p.proname=ANY(ARRAY['claves_registro_alta_v1',
            'fecha_registro_alta_pg_v1','material_registro_alta_valido_v1','rechazar_mutacion_alta_ejercicio_v1','registrar_alta_ejercicio_v1'])
          AND (p.proowner<>'vec_personal_propietario'::regrole OR (a.grantee<>'vec_personal_propietario'::regrole
               AND NOT (p.proname='registrar_alta_ejercicio_v1' AND a.grantee='vec_personal_ejecutor'::regrole
                        AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)))) THEN
        RAISE EXCEPTION 'ACL de registro Personal incompatible' USING ERRCODE='42501';
    END IF;
END
$acl$;
COMMIT;
