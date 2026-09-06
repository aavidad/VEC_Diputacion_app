\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000067',0));

CREATE TABLE vec_contratacion_temporal.peticion_centro_revision (
    peticion_ref text NOT NULL,
    version smallint NOT NULL CHECK (version IN (1,2)),
    clave_idempotencia uuid NOT NULL UNIQUE,
    operacion text NOT NULL CHECK (operacion IN ('presentar','ratificar')),
    material text NOT NULL CHECK (octet_length(material) BETWEEN 1 AND 65536),
    material_sha256 text NOT NULL CHECK (material_sha256=encode(sha256(convert_to(material,'UTF8')),'hex')),
    actor_ref text NOT NULL, perfil_ref text NOT NULL, centro_ref text NOT NULL, puesto_ref text NOT NULL,
    configuracion_ref text NOT NULL, configuracion_version bigint NOT NULL CHECK (configuracion_version>0),
    comando jsonb NOT NULL CHECK (jsonb_typeof(comando)='object'),
    peticion jsonb NOT NULL CHECK (jsonb_typeof(peticion)='object'),
    estado text NOT NULL,
    recibo_ref text NOT NULL UNIQUE,
    recibo_json jsonb NOT NULL CHECK (jsonb_typeof(recibo_json)='object'),
    auditoria_ref text NOT NULL UNIQUE, decision_ref text NOT NULL UNIQUE,
    consumo_huella_sha256 text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (peticion_ref,version),
    CHECK ((version=1 AND operacion='presentar' AND estado='pendiente_ratificacion')
        OR (version=2 AND operacion='ratificar' AND estado='ratificada'))
);
CREATE TABLE vec_contratacion_temporal.peticion_centro_outbox (
    evento_ref text PRIMARY KEY,
    peticion_ref text NOT NULL, version smallint NOT NULL,
    tipo text NOT NULL CHECK (tipo IN ('contratacion_temporal.peticion_centro.presentada','contratacion_temporal.peticion_centro.ratificada')),
    carga_json jsonb NOT NULL CHECK (jsonb_typeof(carga_json)='object'),
    creada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (peticion_ref,version) REFERENCES vec_contratacion_temporal.peticion_centro_revision,
    UNIQUE (peticion_ref,version)
);
CREATE TABLE vec_contratacion_temporal.peticion_centro_acceso (
    acceso_ref text PRIMARY KEY,
    modo text NOT NULL CHECK (modo IN ('operacion','peticion','bandeja')),
    recurso_ref text NOT NULL, referencia text NOT NULL,
    actor_ref text NOT NULL, perfil_ref text NOT NULL, centro_ref text NOT NULL, puesto_ref text NOT NULL,
    auditoria_ref text NOT NULL UNIQUE, decision_ref text NOT NULL UNIQUE,
    consumo_huella_sha256 text NOT NULL UNIQUE,
    consultada_en timestamptz(6) NOT NULL
);
DO $seguridad$
DECLARE v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY['peticion_centro_revision','peticion_centro_outbox','peticion_centro_acceso'] LOOP
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY',v_tabla);
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY',v_tabla);
        EXECUTE format('CREATE POLICY propietario ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true)',v_tabla);
        EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.%I FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()',v_tabla);
        EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM PUBLIC,vec_contratacion_temporal_ejecutor',v_tabla);
    END LOOP;
END
$seguridad$;

CREATE FUNCTION vec_contratacion_temporal.registrar_peticion_centro_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
    m jsonb; c jsonb; a jsonb; p jsonb; cfg jsonb; sol jsonb; docs jsonb; d jsonb;
    v_operacion text; v_accion text; v_clave_text text; v_clave uuid; v_ref text; v_cfg_version bigint;
    v_material_huella text; v_contexto text; v_contexto_huella text;
    v_creada timestamptz(6); v_ratificada timestamptz(6); v_fecha timestamptz(6);
    v_consumo record; v_previa vec_contratacion_temporal.peticion_centro_revision%ROWTYPE;
    v_actual vec_contratacion_temporal.peticion_centro_revision%ROWTYPE;
    v_recibo_ref text; v_recibo jsonb; v_estado text; v_version smallint;
BEGIN
    IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN
        RAISE EXCEPTION 'material de petición de centro inválido' USING ERRCODE='P0670';
    END IF;
    BEGIN
        m:=p_material::jsonb; c:=m->'comando'; a:=m->'actor'; p:=m->'peticion';
        cfg:=p->'configuracion'; sol:=p->'solicitud';
        docs:=CASE WHEN sol->'documentos_adjuntos'='null'::jsonb THEN '[]'::jsonb ELSE sol->'documentos_adjuntos' END;
        v_operacion:=c->>'operacion';
        v_clave_text:=c->>'clave_idempotencia'; v_ref:=p->>'referencia';
        IF v_clave_text !~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$' THEN RAISE data_exception; END IF;
        v_clave:=v_clave_text::uuid; v_cfg_version:=(cfg->>'version')::bigint;
        v_creada:=(p->>'creada_en')::timestamptz;
        IF p ? 'ratificada_en' THEN v_ratificada:=(p->>'ratificada_en')::timestamptz; END IF;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'estructura de petición de centro inválida' USING ERRCODE='P0670';
    END;
    BEGIN
    IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(m))<>3
       OR (m-ARRAY['comando','actor','peticion'])<>'{}'::jsonb
       OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(a) IS DISTINCT FROM 'object'
       OR jsonb_typeof(p) IS DISTINCT FROM 'object' OR jsonb_typeof(cfg) IS DISTINCT FROM 'object'
       OR jsonb_typeof(sol) IS DISTINCT FROM 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(a))<>4 OR (a-ARRAY['actor_ref','perfil_ref','centro_ref','puesto_ref'])<>'{}'::jsonb
       OR (SELECT count(*) FROM jsonb_object_keys(cfg))<>4 OR (cfg-ARRAY['referencia','version','solicitante','ratificador'])<>'{}'::jsonb
       OR jsonb_typeof(cfg->'solicitante') IS DISTINCT FROM 'object'
       OR jsonb_typeof(cfg->'ratificador') IS DISTINCT FROM 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(cfg->'solicitante'))<>4
       OR (SELECT count(*) FROM jsonb_object_keys(cfg->'ratificador'))<>4
       OR ((cfg->'solicitante')-ARRAY['actor_ref','perfil_ref','centro_ref','puesto_ref'])<>'{}'::jsonb
       OR ((cfg->'ratificador')-ARRAY['actor_ref','perfil_ref','centro_ref','puesto_ref'])<>'{}'::jsonb
       OR EXISTS (SELECT 1 FROM jsonb_each(a) e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
       OR EXISTS (SELECT 1 FROM jsonb_each(cfg->'solicitante') e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
       OR EXISTS (SELECT 1 FROM jsonb_each(cfg->'ratificador') e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
       OR (SELECT count(*) FROM jsonb_object_keys(sol)) NOT IN (9,10)
       OR (sol-ARRAY['centro_ref','contacto_ref','categoria_ref','grupo_subgrupo','motivo_clave','detalle','periodo','rc','documentos_adjuntos','observaciones'])<>'{}'::jsonb
       OR jsonb_typeof(sol->'periodo') IS DISTINCT FROM 'object' OR jsonb_typeof(sol->'rc') IS DISTINCT FROM 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(sol->'periodo'))<>2 OR ((sol->'periodo')-ARRAY['inicio','fin'])<>'{}'::jsonb
       OR NOT (sol ?& ARRAY['centro_ref','contacto_ref','categoria_ref','grupo_subgrupo','motivo_clave','detalle','periodo','rc','documentos_adjuntos'])
       OR jsonb_typeof(docs) IS DISTINCT FROM 'array'
       OR coalesce(jsonb_array_length(docs),65)>64
       OR EXISTS (SELECT 1 FROM jsonb_array_elements_text(docs) AS elementos(valor)
                  WHERE elementos.valor !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR (SELECT count(*) FROM jsonb_array_elements_text(docs))<>
          (SELECT count(DISTINCT elementos.valor) FROM jsonb_array_elements_text(docs) AS elementos(valor))
       OR EXISTS (SELECT 1 FROM jsonb_each_text(a) e WHERE e.value !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR EXISTS (SELECT 1 FROM jsonb_each_text(cfg->'solicitante') e WHERE e.value !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR EXISTS (SELECT 1 FROM jsonb_each_text(cfg->'ratificador') e WHERE e.value !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR a->>'actor_ref' IS NULL OR a->>'perfil_ref' IS NULL OR a->>'centro_ref' IS NULL OR a->>'puesto_ref' IS NULL
       OR cfg->>'referencia' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' OR v_cfg_version IS NULL OR v_cfg_version<1
       OR cfg->'solicitante'->>'actor_ref'=cfg->'ratificador'->>'actor_ref'
       OR cfg->'solicitante'->>'centro_ref' IS DISTINCT FROM cfg->'ratificador'->>'centro_ref'
       OR sol->>'centro_ref' IS DISTINCT FROM a->>'centro_ref'
       OR a->>'centro_ref' IS DISTINCT FROM cfg->'solicitante'->>'centro_ref'
       OR a->>'centro_ref' IS DISTINCT FROM cfg->'ratificador'->>'centro_ref'
       OR v_ref IS NULL OR v_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR v_creada IS NULL OR p->>'creada_en' !~ 'Z$' OR v_creada<>date_trunc('microseconds',v_creada) THEN
        RAISE EXCEPTION 'petición de centro incompatible' USING ERRCODE='P0670';
    END IF;
    IF v_operacion='presentar' THEN
        v_accion:='contratacion_temporal.peticion_centro.presentar'; v_version:=1; v_estado:='pendiente_ratificacion';
        IF (SELECT count(*) FROM jsonb_object_keys(c))<>3 OR (c-ARRAY['operacion','clave_idempotencia','solicitud'])<>'{}'::jsonb OR NOT c ? 'solicitud'
           OR c->'solicitud' IS DISTINCT FROM sol OR v_ref IS DISTINCT FROM 'peticion:centro:'||v_clave_text
           OR p->>'version' IS DISTINCT FROM '1' OR p->>'estado' IS DISTINCT FROM v_estado
           OR (SELECT count(*) FROM jsonb_object_keys(p))<>6 OR (p-ARRAY['referencia','version','configuracion','solicitud','estado','creada_en'])<>'{}'::jsonb
           OR a IS DISTINCT FROM cfg->'solicitante' THEN
            RAISE EXCEPTION 'presentación de centro incompatible' USING ERRCODE='P0670';
        END IF;
    ELSIF v_operacion='ratificar' THEN
        v_accion:='contratacion_temporal.peticion_centro.ratificar'; v_version:=2; v_estado:='ratificada';
        IF (SELECT count(*) FROM jsonb_object_keys(c))<>5 OR (c-ARRAY['operacion','clave_idempotencia','peticion_ref','version_esperada','motivo'])<>'{}'::jsonb
           OR c->>'peticion_ref' IS DISTINCT FROM v_ref OR c->>'version_esperada' IS DISTINCT FROM '1'
           OR p->>'version' IS DISTINCT FROM '2' OR p->>'estado' IS DISTINCT FROM v_estado
           OR p->>'motivo_ratificacion' IS DISTINCT FROM c->>'motivo' OR coalesce(octet_length(c->>'motivo'),0) NOT BETWEEN 1 AND 1000
           OR btrim(c->>'motivo') IS DISTINCT FROM c->>'motivo' OR c->>'motivo' ~ '[[:cntrl:]]'
           OR v_ratificada IS NULL OR p->>'ratificada_en' !~ 'Z$'
           OR v_ratificada<>date_trunc('microseconds',v_ratificada) OR v_ratificada<v_creada
           OR (SELECT count(*) FROM jsonb_object_keys(p))<>8 OR (p-ARRAY['referencia','version','configuracion','solicitud','estado','creada_en','ratificada_en','motivo_ratificacion'])<>'{}'::jsonb
           OR a IS DISTINCT FROM cfg->'ratificador' THEN
            RAISE EXCEPTION 'ratificación de centro incompatible' USING ERRCODE='P0670';
        END IF;
    ELSE
        RAISE EXCEPTION 'operación de petición de centro inválida' USING ERRCODE='P0670';
    END IF;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'estructura de petición de centro inválida' USING ERRCODE='P0670';
    END;
    v_material_huella:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
    v_contexto:='{"ambitos":{"centro_ref":'||to_jsonb(a->>'centro_ref')::text||',"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"material_sha256":"'||v_material_huella||'"}}';
    v_contexto_huella:=encode(sha256(convert_to(v_contexto,'UTF8')),'hex');
    BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
    EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'decisión de petición inválida' USING ERRCODE='P0673'; END;
    IF d->>'accion' IS DISTINCT FROM v_accion OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'peticion_centro' OR d->>'finalidad' IS DISTINCT FROM 'gestionar_peticion_centro'
       OR d->>'recurso_ref' IS DISTINCT FROM v_ref OR d->>'principal_id' IS DISTINCT FROM a->>'actor_ref'
       OR d->>'perfil_activo_ref' IS DISTINCT FROM a->>'perfil_ref'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'autorización de petición divergente' USING ERRCODE='P0673';
    END IF;
    SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_peticion_centro_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM v_ref
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'consumo de petición divergente' USING ERRCODE='P0673';
    END IF;
    PERFORM pg_advisory_xact_lock(hashtextextended('vec:peticion-centro:clave:'||v_clave_text,0));
    PERFORM pg_advisory_xact_lock(hashtextextended('vec:peticion-centro:ref:'||v_ref,0));
    SELECT * INTO v_previa FROM vec_contratacion_temporal.peticion_centro_revision WHERE clave_idempotencia=v_clave;
    IF FOUND THEN
        IF v_previa.actor_ref IS DISTINCT FROM a->>'actor_ref' OR v_previa.perfil_ref IS DISTINCT FROM a->>'perfil_ref'
           OR v_previa.centro_ref IS DISTINCT FROM a->>'centro_ref' OR v_previa.puesto_ref IS DISTINCT FROM a->>'puesto_ref' THEN
            RAISE EXCEPTION 'operación de petición ajena' USING ERRCODE='P0673';
        ELSIF v_previa.operacion IS DISTINCT FROM v_operacion OR v_previa.comando IS DISTINCT FROM c THEN
            RAISE EXCEPTION 'clave usada por otro comando' USING ERRCODE='P0671';
        ELSIF v_previa.material IS DISTINCT FROM p_material THEN
            RAISE EXCEPTION 'material concurrente divergente; reintentar con la misma clave' USING ERRCODE='P0674';
        END IF;
        RETURN v_previa.recibo_json||jsonb_build_object('estado_local','replay_confirmado');
    END IF;
    SELECT * INTO v_actual FROM vec_contratacion_temporal.peticion_centro_revision
     WHERE peticion_ref=v_ref ORDER BY version DESC LIMIT 1 FOR UPDATE;
    IF v_operacion='presentar' THEN
        IF FOUND THEN RAISE EXCEPTION 'referencia de petición ya usada' USING ERRCODE='P0671'; END IF;
    ELSE
        IF NOT FOUND OR v_actual.version<>1 OR v_actual.estado<>'pendiente_ratificacion' THEN
            RAISE EXCEPTION 'versión de petición en conflicto' USING ERRCODE='P0672';
        END IF;
        IF (p-ARRAY['version','estado','ratificada_en','motivo_ratificacion']) IS DISTINCT FROM
           (v_actual.peticion-ARRAY['version','estado','ratificada_en','motivo_ratificacion'])
           OR a->>'actor_ref' IS NOT DISTINCT FROM v_actual.actor_ref THEN
            RAISE EXCEPTION 'ratificación no conserva la petición' USING ERRCODE='P0673';
        END IF;
    END IF;
    v_fecha:=date_trunc('microseconds',clock_timestamp());
    IF v_fecha<v_creada OR (v_ratificada IS NOT NULL AND v_fecha<v_ratificada) THEN
        RAISE EXCEPTION 'cronología de petición inválida' USING ERRCODE='P0670';
    END IF;
    v_recibo_ref:='recibo:peticion-centro:'||gen_random_uuid()::text;
    v_recibo:=jsonb_build_object('recibo_ref',v_recibo_ref,'peticion_ref',v_ref,'version',v_version,
        'estado',v_estado,'actor_ref',a->>'actor_ref','registrado_en',v_fecha,'estado_local','registrado');
    INSERT INTO vec_contratacion_temporal.peticion_centro_revision(
        peticion_ref,version,clave_idempotencia,operacion,material,material_sha256,actor_ref,perfil_ref,centro_ref,puesto_ref,
        configuracion_ref,configuracion_version,comando,peticion,estado,recibo_ref,recibo_json,auditoria_ref,decision_ref,consumo_huella_sha256,registrada_en)
    VALUES(v_ref,v_version,v_clave,v_operacion,p_material,v_material_huella,a->>'actor_ref',a->>'perfil_ref',a->>'centro_ref',a->>'puesto_ref',
        cfg->>'referencia',v_cfg_version,c,p,v_estado,v_recibo_ref,v_recibo,v_consumo.auditoria_ref,v_consumo.decision_ref,v_consumo.consumo_huella_sha256,v_fecha);
    INSERT INTO vec_contratacion_temporal.peticion_centro_outbox(evento_ref,peticion_ref,version,tipo,carga_json,creada_en)
    VALUES('evento:peticion-centro:'||gen_random_uuid()::text,v_ref,v_version,
        CASE v_operacion WHEN 'presentar' THEN 'contratacion_temporal.peticion_centro.presentada' ELSE 'contratacion_temporal.peticion_centro.ratificada' END,
        jsonb_build_object('recibo_ref',v_recibo_ref,'peticion_ref',v_ref,'version',v_version,'estado',v_estado),v_fecha);
    RETURN v_recibo;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.consultar_peticiones_centro_v1(
    p_consulta text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
    q jsonb; a jsonb; d jsonb; v_modo text; v_ref text; v_recurso text;
    v_material_huella text; v_contexto text; v_contexto_huella text; v_resultado jsonb;
    v_consumo record; v_revision vec_contratacion_temporal.peticion_centro_revision%ROWTYPE;
BEGIN
    IF p_consulta IS NULL OR octet_length(p_consulta) NOT BETWEEN 1 AND 65536 THEN
        RAISE EXCEPTION 'consulta de petición inválida' USING ERRCODE='P0670';
    END IF;
    BEGIN q:=p_consulta::jsonb; a:=q->'actor'; v_modo:=q->>'modo'; v_ref:=q->>'referencia';
    EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'consulta de petición inválida' USING ERRCODE='P0670'; END;
    BEGIN
    IF jsonb_typeof(q) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(q))<>3 OR (q-ARRAY['modo','actor','referencia'])<>'{}'::jsonb
       OR jsonb_typeof(a) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(a))<>4
       OR (a-ARRAY['actor_ref','perfil_ref','centro_ref','puesto_ref'])<>'{}'::jsonb
       OR EXISTS (SELECT 1 FROM jsonb_each(a) e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
       OR EXISTS (SELECT 1 FROM jsonb_each_text(a) e WHERE e.value !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR a->>'actor_ref' IS NULL OR a->>'perfil_ref' IS NULL OR a->>'centro_ref' IS NULL OR a->>'puesto_ref' IS NULL THEN
        RAISE EXCEPTION 'actor de consulta inválido' USING ERRCODE='P0670';
    END IF;
    IF v_modo='operacion' AND v_ref ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$' THEN
        v_recurso:='operacion:peticion-centro:'||v_ref;
    ELSIF v_modo='peticion' AND v_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
        v_recurso:=v_ref;
    ELSIF v_modo='bandeja' AND v_ref IS NOT DISTINCT FROM a->>'centro_ref' THEN
        v_recurso:='peticiones:centro:'||v_ref;
    ELSE RAISE EXCEPTION 'referencia de consulta inválida' USING ERRCODE='P0670'; END IF;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'estructura de consulta inválida' USING ERRCODE='P0670';
    END;
    v_material_huella:=encode(sha256(convert_to(p_consulta,'UTF8')),'hex');
    v_contexto:='{"ambitos":{"centro_ref":'||to_jsonb(a->>'centro_ref')::text||',"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"material_sha256":"'||v_material_huella||'"}}';
    v_contexto_huella:=encode(sha256(convert_to(v_contexto,'UTF8')),'hex');
    BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
    EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'decisión de consulta inválida' USING ERRCODE='P0673'; END;
    IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.peticion_centro.consultar'
       OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal' OR d->>'tipo_recurso' IS DISTINCT FROM 'peticion_centro'
       OR d->>'finalidad' IS DISTINCT FROM 'gestionar_peticion_centro' OR d->>'recurso_ref' IS DISTINCT FROM v_recurso
       OR d->>'principal_id' IS DISTINCT FROM a->>'actor_ref' OR d->>'perfil_activo_ref' IS DISTINCT FROM a->>'perfil_ref'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'autorización de consulta divergente' USING ERRCODE='P0673';
    END IF;
    SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_peticion_centro_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM v_recurso
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'consumo de consulta divergente' USING ERRCODE='P0673';
    END IF;
    IF v_modo='operacion' THEN
        SELECT * INTO v_revision FROM vec_contratacion_temporal.peticion_centro_revision WHERE clave_idempotencia=v_ref::uuid;
        IF FOUND AND (v_revision.actor_ref IS DISTINCT FROM a->>'actor_ref' OR v_revision.perfil_ref IS DISTINCT FROM a->>'perfil_ref'
                      OR v_revision.centro_ref IS DISTINCT FROM a->>'centro_ref' OR v_revision.puesto_ref IS DISTINCT FROM a->>'puesto_ref') THEN
            RAISE EXCEPTION 'operación de petición ajena' USING ERRCODE='P0673';
        END IF;
        v_resultado:=CASE WHEN FOUND THEN v_revision.material::jsonb ELSE 'null'::jsonb END;
    ELSIF v_modo='peticion' THEN
        SELECT * INTO v_revision FROM vec_contratacion_temporal.peticion_centro_revision WHERE peticion_ref=v_ref ORDER BY version DESC LIMIT 1;
        IF NOT FOUND THEN RAISE EXCEPTION 'petición no encontrada' USING ERRCODE='P0672'; END IF;
        IF a IS DISTINCT FROM v_revision.peticion->'configuracion'->'solicitante'
           AND a IS DISTINCT FROM v_revision.peticion->'configuracion'->'ratificador' THEN
            RAISE EXCEPTION 'petición no visible' USING ERRCODE='P0673';
        END IF;
        v_resultado:=v_revision.peticion;
    ELSE
        SELECT coalesce(jsonb_agg(x.peticion ORDER BY x.registrada_en DESC,x.peticion_ref),'[]'::jsonb) INTO v_resultado
        FROM (SELECT ultimas.peticion_ref,ultimas.peticion,ultimas.registrada_en
              FROM (SELECT DISTINCT ON (r.peticion_ref) r.peticion_ref,r.peticion,r.registrada_en
                    FROM vec_contratacion_temporal.peticion_centro_revision r
                    WHERE a=r.peticion->'configuracion'->'solicitante' OR a=r.peticion->'configuracion'->'ratificador'
                    ORDER BY r.peticion_ref,r.version DESC) ultimas
              ORDER BY ultimas.registrada_en DESC,ultimas.peticion_ref LIMIT 50) x;
    END IF;
    INSERT INTO vec_contratacion_temporal.peticion_centro_acceso(acceso_ref,modo,recurso_ref,referencia,actor_ref,perfil_ref,centro_ref,puesto_ref,
        auditoria_ref,decision_ref,consumo_huella_sha256,consultada_en)
    VALUES('acceso:peticion-centro:'||gen_random_uuid()::text,v_modo,v_recurso,v_ref,a->>'actor_ref',a->>'perfil_ref',a->>'centro_ref',a->>'puesto_ref',
        v_consumo.auditoria_ref,v_consumo.decision_ref,v_consumo.consumo_huella_sha256,date_trunc('microseconds',clock_timestamp()));
    RETURN v_resultado;
END
$funcion$;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_contratacion_temporal.consultar_peticiones_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC,vec_contratacion_temporal_migrador;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_contratacion_temporal.consultar_peticiones_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_ejecutor;
COMMIT;
