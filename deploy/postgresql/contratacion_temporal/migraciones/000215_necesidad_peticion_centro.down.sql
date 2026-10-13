\set ON_ERROR_STOP on
-- CT215 DOWN solo antes de cualquier petición v3. La historia sellada no se reescribe.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000215',0));
LOCK TABLE vec_contratacion_temporal.peticion_centro_revision IN SHARE MODE;
DO $pre$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.peticion_centro_revision p
      WHERE p.peticion#>'{solicitud,necesidad}' IS NOT NULL OR p.comando ? 'esquema')
    OR encode(sha256(convert_to((SELECT prosrc FROM pg_proc WHERE oid=
      'vec_contratacion_temporal.registrar_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
       IS DISTINCT FROM 'f414c20fa088b3a86ab8987fe58f69fe3796eda75af2ef6a4d1c8b2e8d140016'
    OR encode(sha256(convert_to((SELECT prosrc FROM pg_proc WHERE oid=
      'vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
       IS DISTINCT FROM '954e275051e54b259a690820208ab75cab3d9a668cf520741803c0b6a995cbc9'
 THEN RAISE EXCEPTION 'CT215 DOWN: historia v3 o definición incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.registrar_peticion_centro_v1(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path TO 'pg_catalog', 'pg_temp'
    SET row_security TO 'on'
    SET "TimeZone" TO 'UTC'
    SET lock_timeout TO '2s'
    AS $_$
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
       OR (SELECT count(*) FROM jsonb_object_keys(sol)) NOT IN (9,10,12,13)
       OR (sol-ARRAY['centro_ref','contacto_ref','categoria_ref','grupo_subgrupo','motivo_clave','detalle','periodo','rc','documentos_adjuntos','observaciones','jornada_minutos','numero_personas','puesto_solicitado'])<>'{}'::jsonb
       OR jsonb_typeof(sol->'periodo') IS DISTINCT FROM 'object' OR jsonb_typeof(sol->'rc') IS DISTINCT FROM 'object'
       OR NOT vec_contratacion_temporal.periodo_previsto_estructural_v1(sol->'periodo')
       OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(sol) IS DISTINCT FROM true
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
    -- Una presentación nueva debe traer los tres datos antes de consumir la
    -- autorización V3. Si la clave ya existe es un replay de una revisión
    -- antigua, que conserva su respuesta exacta.
    IF v_operacion='presentar' AND NOT (sol ?& ARRAY['jornada_minutos','numero_personas','puesto_solicitado'])
       AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.peticion_centro_revision WHERE clave_idempotencia=v_clave) THEN
        RAISE EXCEPTION 'datos de puesto de la petición ausentes' USING ERRCODE='P0670';
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
    -- Segunda barrera, ya con la clave bloqueada: las revisiones antiguas
    -- conservan su replay exacto y una presentación nueva lleva los tres datos.
    IF v_operacion='presentar' AND NOT (sol ?& ARRAY['jornada_minutos','numero_personas','puesto_solicitado']) THEN
        RAISE EXCEPTION 'datos de puesto de la petición ausentes' USING ERRCODE='P0670';
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
$_$;

-- Restaura el cuerpo CT151 instalado byte a byte; el metadato search_path
-- conserva pg_temp para el SECURITY DEFINER según la guarda de CI1114.
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path TO 'pg_catalog', 'pg_temp'
    SET row_security TO 'on'
    SET "TimeZone" TO 'UTC'
    SET lock_timeout TO '2s'
    SET statement_timeout TO '15s'
    SET idle_in_transaction_session_timeout TO '20s'
    AS $_$
DECLARE
    m jsonb; d jsonb; r jsonb; v_alta jsonb; v_solicitud_peticion jsonb;
    v_claves text[]; v_claves_recibo text[];
    v_modo text; v_accion text; v_recurso text; v_peticion_ref text;
    v_actor_ref text; v_perfil_ref text; v_perfil_efecto_ref text; v_ambito_alta_hmac text;
    v_centro_ref text; v_categoria_ref text; v_alta_durable record;
    v_clave_alta_candidata text;
    v_material_sha256 text; v_contexto text; v_contexto_sha256 text;
    v_fecha timestamptz(6); v_alta_confirmada_en timestamptz(6);
    v_acceso_ref text; v_resultado jsonb; v_alta_canonica bytea;
    v_confirmacion_alta_ref text; v_ambito_alta_raiz_hmac text;
    v_consumo record;
    v_peticion vec_contratacion_temporal.peticion_centro_revision%ROWTYPE;
    v_reserva vec_contratacion_temporal.entrega_peticion_centro_reserva%ROWTYPE;
    v_confirmacion vec_contratacion_temporal.entrega_peticion_centro_confirmacion%ROWTYPE;
BEGIN
    IF session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
       OR current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'off'
       OR current_setting('TimeZone')<>'UTC' THEN
        RAISE EXCEPTION 'identidad o transacción de entrega rechazada' USING ERRCODE='P0683';
    END IF;
    IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN
        RAISE EXCEPTION 'material de entrega inválido' USING ERRCODE='P0680';
    END IF;
    BEGIN
        m:=p_material::jsonb;
        v_modo:=m->>'modo'; v_actor_ref:=m->>'actor_ref';
        v_perfil_ref:=m->>'perfil_ref'; v_peticion_ref:=m->>'peticion_ref';
        v_ambito_alta_hmac:=m->>'ambito_alta_hmac';
        v_centro_ref:=m->>'centro_ref'; v_categoria_ref:=m->>'categoria_ref';
        v_perfil_efecto_ref:=v_perfil_ref;
        v_clave_alta_candidata:=m->>'clave_alta_candidata'; r:=m->'recibo_alta';
        SELECT array_agg(clave ORDER BY clave) INTO v_claves FROM jsonb_object_keys(m) AS k(clave);
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'material de entrega inválido' USING ERRCODE='P0680';
    END;
    IF jsonb_typeof(m) IS DISTINCT FROM 'object'
       OR jsonb_typeof(m->'modo') IS DISTINCT FROM 'string'
       OR jsonb_typeof(m->'actor_ref') IS DISTINCT FROM 'string'
       OR jsonb_typeof(m->'perfil_ref') IS DISTINCT FROM 'string'
       OR coalesce(v_actor_ref,'') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(v_perfil_ref,'') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
        RAISE EXCEPTION 'actor de entrega inválido' USING ERRCODE='P0680';
    END IF;
    IF v_modo='bandeja' THEN
        IF v_claves IS DISTINCT FROM ARRAY['actor_ref','modo','perfil_ref']::text[] THEN
            RAISE EXCEPTION 'bandeja de entrega inválida' USING ERRCODE='P0680';
        END IF;
        v_accion:='contratacion_temporal.peticion_centro.rrhh.consultar';
        v_recurso:='peticiones:centro:rrhh';
    ELSIF v_modo='preparar' THEN
        IF v_claves IS DISTINCT FROM ARRAY['actor_ref','ambito_alta_hmac','categoria_ref','centro_ref','clave_alta_candidata','modo','perfil_ref','peticion_ref','version_esperada']::text[]
           OR jsonb_typeof(m->'peticion_ref') IS DISTINCT FROM 'string'
           OR jsonb_typeof(m->'clave_alta_candidata') IS DISTINCT FROM 'string'
           OR jsonb_typeof(m->'ambito_alta_hmac') IS DISTINCT FROM 'string'
           OR coalesce(v_peticion_ref,'') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR jsonb_typeof(m->'version_esperada') IS DISTINCT FROM 'number'
           OR m->>'version_esperada' IS DISTINCT FROM '2'
           OR coalesce(v_clave_alta_candidata,'') !~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
           OR v_clave_alta_candidata='00000000-0000-4000-8000-000000000000'
           OR coalesce(v_ambito_alta_hmac,'') !~ '^hmac-sha256:vec[.]contratacion-temporal[.]ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
           OR right(v_ambito_alta_hmac,64)=repeat('0',64) THEN
            RAISE EXCEPTION 'preparación de entrega inválida' USING ERRCODE='P0680';
        END IF;
        v_accion:='contratacion_temporal.peticion_centro.rrhh.entregar';
        v_recurso:=v_peticion_ref;
    ELSIF v_modo='confirmar' THEN
        IF v_claves IS DISTINCT FROM ARRAY['actor_ref','ambito_alta_hmac','categoria_ref','centro_ref','modo','perfil_ref','peticion_ref','recibo_alta','version_esperada']::text[]
           OR jsonb_typeof(m->'peticion_ref') IS DISTINCT FROM 'string'
           OR jsonb_typeof(m->'ambito_alta_hmac') IS DISTINCT FROM 'string'
           OR coalesce(v_peticion_ref,'') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR jsonb_typeof(m->'version_esperada') IS DISTINCT FROM 'number'
           OR m->>'version_esperada' IS DISTINCT FROM '2'
           OR jsonb_typeof(r) IS DISTINCT FROM 'object'
           OR coalesce(v_ambito_alta_hmac,'') !~ '^hmac-sha256:vec[.]contratacion-temporal[.]ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
           OR right(v_ambito_alta_hmac,64)=repeat('0',64) THEN
            RAISE EXCEPTION 'confirmación de entrega inválida' USING ERRCODE='P0680';
        END IF;
        SELECT array_agg(clave ORDER BY clave) INTO v_claves_recibo FROM jsonb_object_keys(r) AS k(clave);
        IF v_claves_recibo IS DISTINCT FROM ARRAY['auditoria_ref','confirmada_en','evento_ref','expediente_ref','numero_visible','recibo_ref','version']::text[]
           OR EXISTS (SELECT 1 FROM jsonb_each(r) AS campo(clave,valor)
                       WHERE clave<>'version' AND jsonb_typeof(valor) IS DISTINCT FROM 'string')
           OR jsonb_typeof(r->'version') IS DISTINCT FROM 'number' OR r->>'version' IS DISTINCT FROM '1'
           OR coalesce(r->>'expediente_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR coalesce(r->>'numero_visible','') !~ '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$'
           OR coalesce(r->>'recibo_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR coalesce(r->>'auditoria_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR coalesce(r->>'evento_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR coalesce(r->>'confirmada_en','') !~ 'Z$' THEN
            RAISE EXCEPTION 'recibo de alta inválido' USING ERRCODE='P0680';
        END IF;
        BEGIN v_alta_confirmada_en:=(r->>'confirmada_en')::timestamptz;
        EXCEPTION WHEN data_exception THEN
            RAISE EXCEPTION 'fecha de alta inválida' USING ERRCODE='P0680';
        END;
        IF v_alta_confirmada_en<>date_trunc('microseconds',v_alta_confirmada_en) THEN
            RAISE EXCEPTION 'fecha de alta no canónica' USING ERRCODE='P0680';
        END IF;
        v_accion:='contratacion_temporal.peticion_centro.rrhh.entregar';
        v_recurso:=v_peticion_ref;
    ELSE
        RAISE EXCEPTION 'modo de entrega inválido' USING ERRCODE='P0680';
    END IF;

    -- La revisión ratificada fija los ámbitos antes del consumo V3. El bloqueo
    -- y la transacción serializable impiden cambiarla entre lectura y efecto.
    IF v_modo<>'bandeja' THEN
        PERFORM pg_advisory_xact_lock(hashtextextended('vec:entrega-peticion-centro:'||v_peticion_ref,0));
        SELECT * INTO v_peticion FROM vec_contratacion_temporal.peticion_centro_revision
         WHERE peticion_ref=v_peticion_ref AND version=2 AND operacion='ratificar' AND estado='ratificada';
        IF NOT FOUND THEN
            RAISE EXCEPTION 'petición ratificada no disponible' USING ERRCODE='P0681';
        END IF;
        IF jsonb_typeof(m->'centro_ref') IS DISTINCT FROM 'string'
           OR jsonb_typeof(m->'categoria_ref') IS DISTINCT FROM 'string'
           OR coalesce(v_centro_ref,'') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR coalesce(v_categoria_ref,'') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR v_centro_ref IS DISTINCT FROM v_peticion.peticion->'solicitud'->>'centro_ref'
           OR v_categoria_ref IS DISTINCT FROM v_peticion.peticion->'solicitud'->>'categoria_ref' THEN
            RAISE EXCEPTION 'ámbitos de petición divergentes' USING ERRCODE='P0681';
        END IF;
    END IF;
    v_material_sha256:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
    IF v_modo='bandeja' THEN
        v_contexto:='{"ambitos":{"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"material_sha256":"'||v_material_sha256||'"}}';
    ELSE
        v_contexto:='{"ambitos":{"categoria_ref":'||vec_contratacion_temporal.texto_json_go_v1(v_categoria_ref)||
            ',"centro_ref":'||vec_contratacion_temporal.texto_json_go_v1(v_centro_ref)||
            ',"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"material_sha256":"'||v_material_sha256||'"}}';
    END IF;
    v_contexto_sha256:=encode(sha256(convert_to(v_contexto,'UTF8')),'hex');
    BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'decisión de entrega inválida' USING ERRCODE='P0683';
    END;
    IF d->>'accion' IS DISTINCT FROM v_accion
       OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'entrega_peticion_centro'
       OR d->>'finalidad' IS DISTINCT FROM 'tramitar_peticion_centro_rrhh'
       OR d->>'recurso_ref' IS DISTINCT FROM v_recurso
       OR d->>'principal_id' IS DISTINCT FROM v_actor_ref
       OR d->>'perfil_activo_ref' IS DISTINCT FROM v_perfil_ref
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_sha256 THEN
        RAISE EXCEPTION 'autorización de entrega divergente' USING ERRCODE='P0683';
    END IF;
    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3.registrar_y_consumir_entrega_peticion_centro_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.consumo_nuevo IS NOT TRUE
       OR v_consumo.efecto_ref IS DISTINCT FROM v_recurso
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_sha256 THEN
        RAISE EXCEPTION 'consumo de entrega divergente' USING ERRCODE='P0683';
    END IF;
    v_fecha:=date_trunc('microseconds',clock_timestamp());
    v_acceso_ref:='acceso:entrega-peticion-centro:'||gen_random_uuid()::text;
    INSERT INTO vec_contratacion_temporal.entrega_peticion_centro_acceso(
        acceso_ref,modo,recurso_ref,actor_ref,perfil_ref,material_sha256,
        auditoria_ref,decision_ref,consumo_huella_sha256,registrada_en)
    VALUES(v_acceso_ref,v_modo,v_recurso,v_actor_ref,v_perfil_ref,v_material_sha256,
        v_consumo.auditoria_ref,v_consumo.decision_ref,v_consumo.consumo_huella_sha256,v_fecha);

    IF v_modo='bandeja' THEN
        SELECT coalesce(jsonb_agg(x.fila ORDER BY x.registrada_en DESC,x.peticion_ref),'[]'::jsonb)
          INTO v_resultado
          FROM (
            SELECT revision.peticion_ref,revision.registrada_en,
              CASE WHEN confirmacion.peticion_ref IS NOT NULL THEN
                jsonb_build_object('peticion',revision.peticion,'estado_entrega','confirmada','recibo_alta',confirmacion.recibo_alta)
              WHEN reserva.peticion_ref IS NOT NULL THEN
                jsonb_build_object('peticion',revision.peticion,'estado_entrega','preparada')
              ELSE jsonb_build_object('peticion',revision.peticion,'estado_entrega','pendiente') END AS fila
              FROM vec_contratacion_temporal.peticion_centro_revision revision
              LEFT JOIN vec_contratacion_temporal.entrega_peticion_centro_reserva reserva
                ON reserva.peticion_ref=revision.peticion_ref
              LEFT JOIN vec_contratacion_temporal.entrega_peticion_centro_confirmacion confirmacion
                ON confirmacion.peticion_ref=revision.peticion_ref
             WHERE revision.version=2 AND revision.operacion='ratificar' AND revision.estado='ratificada'
             ORDER BY revision.registrada_en DESC,revision.peticion_ref LIMIT 50
          ) x;
        RETURN v_resultado;
    END IF;

    PERFORM pg_advisory_xact_lock(hashtextextended('vec:entrega-peticion-centro:'||v_peticion_ref,0));
    SELECT * INTO v_peticion FROM vec_contratacion_temporal.peticion_centro_revision
     WHERE peticion_ref=v_peticion_ref AND version=2 AND operacion='ratificar' AND estado='ratificada';
    IF NOT FOUND THEN
        RAISE EXCEPTION 'petición ratificada no disponible' USING ERRCODE='P0681';
    END IF;
    SELECT * INTO v_reserva FROM vec_contratacion_temporal.entrega_peticion_centro_reserva
     WHERE peticion_ref=v_peticion_ref;

    -- Reserva histórica: la nueva decisión autoriza el recurso completo,
    -- pero el alta durable debe estar sellada con el perfil originalmente
    -- reservado. Sin alta durable no se crea otra bajo una identidad distinta.
    IF v_modo='preparar' AND v_reserva.peticion_ref IS NOT NULL
       AND v_reserva.perfil_ref IS DISTINCT FROM v_perfil_ref THEN
        IF v_reserva.actor_ref IS DISTINCT FROM v_actor_ref THEN
            RAISE EXCEPTION 'reserva histórica de otra identidad' USING ERRCODE='P0681';
        END IF;
        SELECT * INTO v_confirmacion FROM vec_contratacion_temporal.entrega_peticion_centro_confirmacion
         WHERE peticion_ref=v_peticion_ref;
        IF FOUND THEN
            RETURN jsonb_build_object('peticion',v_peticion.peticion,'estado_entrega','confirmada',
                'clave_alta',v_reserva.clave_alta::text,'ambito_alta_hmac',v_reserva.ambito_alta_hmac,
                'actor_ref',v_reserva.actor_ref,'perfil_ref',v_reserva.perfil_ref,
                'recibo_alta',v_confirmacion.recibo_alta,
            'reserva_creada_ahora',false,'confirmacion_creada_ahora',false);
        END IF;
        SELECT c.*, version.alta_canonica INTO v_alta_durable
          FROM vec_contratacion_temporal.alias_ambito_alta alias
          JOIN vec_contratacion_temporal.identidad_reserva_alta identidad
            ON identidad.ambito_hmac=alias.ambito_raiz_hmac
          JOIN vec_contratacion_temporal.confirmacion_agregado_alta c
            ON c.ambito_hmac=identidad.ambito_hmac
          JOIN vec_contratacion_temporal.expediente_alta alta
            ON alta.expediente_ref=c.expediente_ref
          JOIN vec_contratacion_temporal.expediente_alta_version version
            ON version.expediente_ref=c.expediente_ref AND version.version=1
         WHERE alias.alias_hmac=v_reserva.ambito_alta_hmac
           AND identidad.organizacion_ref='organizacion:desarrollo:dipgra'
           AND identidad.actor_ref=v_reserva.actor_ref AND identidad.perfil_ref=v_reserva.perfil_ref
           AND alta.organizacion_ref=identidad.organizacion_ref
           AND alta.actor_ref=v_reserva.actor_ref AND alta.perfil_ref=v_reserva.perfil_ref
           AND c.version_expediente=1;
        IF NOT FOUND THEN
            RAISE EXCEPTION 'reserva histórica pendiente de reconciliación' USING ERRCODE='P0681';
        END IF;
        r:=jsonb_build_object('version',1,'expediente_ref',v_alta_durable.expediente_ref,
            'numero_visible',v_alta_durable.numero_visible,'recibo_ref',v_alta_durable.recibo_ref,
            'auditoria_ref',v_alta_durable.auditoria_ref,'evento_ref',v_alta_durable.evento_ref,
            'confirmada_en',CASE
                WHEN v_alta_durable.confirmada_en=date_trunc('second',v_alta_durable.confirmada_en)
                    THEN to_char(v_alta_durable.confirmada_en AT TIME ZONE 'UTC',
                        'YYYY-MM-DD"T"HH24:MI:SS"Z"')
                ELSE regexp_replace(to_char(v_alta_durable.confirmada_en AT TIME ZONE 'UTC',
                        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'0+Z$','Z')
                END);
        v_alta_confirmada_en:=v_alta_durable.confirmada_en;
        v_ambito_alta_hmac:=v_reserva.ambito_alta_hmac;
        v_perfil_efecto_ref:=v_reserva.perfil_ref;
        v_modo:='confirmar';
    END IF;

    IF v_modo='preparar' THEN
        IF v_reserva.peticion_ref IS NULL THEN
            BEGIN
                INSERT INTO vec_contratacion_temporal.entrega_peticion_centro_reserva(
                    peticion_ref,peticion_version,clave_alta,ambito_alta_hmac,
                    actor_ref,perfil_ref,acceso_preparacion_ref,reservada_en)
                VALUES(v_peticion_ref,2,v_clave_alta_candidata::uuid,v_ambito_alta_hmac,
                    v_actor_ref,v_perfil_ref,v_acceso_ref,v_fecha)
                RETURNING * INTO v_reserva;
            EXCEPTION WHEN unique_violation THEN
                RAISE EXCEPTION 'candidato de alta ya reservado' USING ERRCODE='P0681';
            END;
            RETURN jsonb_build_object('peticion',v_peticion.peticion,'estado_entrega','preparada',
                'clave_alta',v_reserva.clave_alta::text,'ambito_alta_hmac',v_reserva.ambito_alta_hmac,'actor_ref',v_actor_ref,'perfil_ref',v_reserva.perfil_ref,
                'reserva_creada_ahora',true,'confirmacion_creada_ahora',false);
        END IF;
        SELECT * INTO v_confirmacion FROM vec_contratacion_temporal.entrega_peticion_centro_confirmacion
         WHERE peticion_ref=v_peticion_ref;
        IF FOUND THEN
            IF v_reserva.actor_ref IS DISTINCT FROM v_actor_ref OR v_reserva.perfil_ref IS DISTINCT FROM v_perfil_ref THEN
                RAISE EXCEPTION 'petición confirmada por otra identidad RRHH' USING ERRCODE='P0681';
            END IF;
            RETURN jsonb_build_object('peticion',v_peticion.peticion,'estado_entrega','confirmada',
                'clave_alta',v_reserva.clave_alta::text,'ambito_alta_hmac',v_reserva.ambito_alta_hmac,'actor_ref',v_actor_ref,'perfil_ref',v_reserva.perfil_ref,
                'recibo_alta',v_confirmacion.recibo_alta,
            'reserva_creada_ahora',false,'confirmacion_creada_ahora',false);
        END IF;
        IF v_reserva.actor_ref IS DISTINCT FROM v_actor_ref OR v_reserva.perfil_ref IS DISTINCT FROM v_perfil_ref THEN
            RAISE EXCEPTION 'petición preparada por otra identidad RRHH' USING ERRCODE='P0681';
        END IF;
        RETURN jsonb_build_object('peticion',v_peticion.peticion,'estado_entrega','preparada',
            'clave_alta',v_reserva.clave_alta::text,'ambito_alta_hmac',v_reserva.ambito_alta_hmac,'actor_ref',v_actor_ref,'perfil_ref',v_reserva.perfil_ref,
            'reserva_creada_ahora',false,'confirmacion_creada_ahora',false);
    END IF;

    IF v_reserva.peticion_ref IS NULL OR v_reserva.actor_ref IS DISTINCT FROM v_actor_ref
       OR v_reserva.perfil_ref IS DISTINCT FROM v_perfil_efecto_ref THEN
        RAISE EXCEPTION 'reserva de entrega incompatible' USING ERRCODE='P0681';
    END IF;
    IF v_reserva.ambito_alta_hmac IS DISTINCT FROM v_ambito_alta_hmac THEN
        RAISE EXCEPTION 'ámbito de alta distinto del reservado' USING ERRCODE='P0681';
    END IF;
    SELECT * INTO v_confirmacion FROM vec_contratacion_temporal.entrega_peticion_centro_confirmacion
     WHERE peticion_ref=v_peticion_ref;
    IF FOUND THEN
        IF v_confirmacion.recibo_alta IS DISTINCT FROM r
           OR v_confirmacion.ambito_alta_hmac IS DISTINCT FROM v_ambito_alta_hmac THEN
            RAISE EXCEPTION 'confirmación divergente para la petición' USING ERRCODE='P0681';
        END IF;
        RETURN jsonb_build_object('peticion',v_peticion.peticion,'estado_entrega','confirmada',
            'clave_alta',v_reserva.clave_alta::text,'ambito_alta_hmac',v_reserva.ambito_alta_hmac,'actor_ref',v_actor_ref,'perfil_ref',v_reserva.perfil_ref,
            'recibo_alta',v_confirmacion.recibo_alta,
            'reserva_creada_ahora',false,'confirmacion_creada_ahora',false);
    END IF;

    -- El alta ya fue confirmada por el servicio propietario. Esta transacción
    -- solo acredita y enlaza esa historia; no hace atómica el alta con el puente.
    -- El canon Go vigente ubica el centro en solicitud.centro_ref.
    BEGIN
        SELECT confirmacion.confirmacion_ref,confirmacion.ambito_hmac,version.alta_canonica
          INTO v_confirmacion_alta_ref,v_ambito_alta_raiz_hmac,v_alta_canonica
          FROM vec_contratacion_temporal.alias_ambito_alta alias
          JOIN vec_contratacion_temporal.identidad_reserva_alta identidad
            ON identidad.ambito_hmac=alias.ambito_raiz_hmac
          JOIN vec_contratacion_temporal.confirmacion_agregado_alta confirmacion
            ON confirmacion.ambito_hmac=identidad.ambito_hmac
          JOIN vec_contratacion_temporal.expediente_alta alta
            ON alta.expediente_ref=confirmacion.expediente_ref
          JOIN vec_contratacion_temporal.expediente_alta_version version
            ON version.expediente_ref=confirmacion.expediente_ref AND version.version=1
         WHERE alias.alias_hmac=v_ambito_alta_hmac
           AND identidad.organizacion_ref='organizacion:desarrollo:dipgra'
           AND identidad.actor_ref=v_actor_ref AND identidad.perfil_ref=v_perfil_efecto_ref
           AND alta.organizacion_ref=identidad.organizacion_ref
           AND alta.actor_ref=v_actor_ref AND alta.perfil_ref=v_perfil_efecto_ref
           AND confirmacion.expediente_ref=r->>'expediente_ref'
           AND confirmacion.numero_visible=r->>'numero_visible'
           AND confirmacion.version_expediente=1
           AND confirmacion.recibo_ref=r->>'recibo_ref'
           AND confirmacion.auditoria_ref=r->>'auditoria_ref'
           AND confirmacion.evento_ref=r->>'evento_ref'
           AND confirmacion.confirmada_en=v_alta_confirmada_en;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'historia de alta no disponible' USING ERRCODE='P0684';
    END;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'recibo de alta no acreditado' USING ERRCODE='P0683';
    END IF;
    BEGIN v_alta:=convert_from(v_alta_canonica,'UTF8')::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'historia de alta no disponible' USING ERRCODE='P0684';
    END;
    v_solicitud_peticion:=v_peticion.peticion->'solicitud';
    IF v_solicitud_peticion->'documentos_adjuntos'='null'::jsonb THEN
        v_solicitud_peticion:=pg_catalog.jsonb_set(
            v_solicitud_peticion,'{documentos_adjuntos}','[]'::jsonb,false
        );
    END IF;
    IF pg_catalog.jsonb_typeof(v_solicitud_peticion #> '{periodo,inicio}')='string'
       AND v_solicitud_peticion #>> '{periodo,inicio}'~'^[0-9]{4}-[0-9]{2}-[0-9]{2}T00:00:00Z$' THEN
        v_solicitud_peticion:=pg_catalog.jsonb_set(
            v_solicitud_peticion,'{periodo,inicio}',pg_catalog.to_jsonb(
                pg_catalog.to_char(
                    (v_solicitud_peticion #>> '{periodo,inicio}')::timestamptz AT TIME ZONE 'UTC',
                    'YYYY-MM-DD'
                )
            ),false
        );
    END IF;
    IF pg_catalog.jsonb_typeof(v_solicitud_peticion #> '{periodo,fin}')='string'
       AND v_solicitud_peticion #>> '{periodo,fin}'~'^[0-9]{4}-[0-9]{2}-[0-9]{2}T00:00:00Z$' THEN
        v_solicitud_peticion:=pg_catalog.jsonb_set(
            v_solicitud_peticion,'{periodo,fin}',pg_catalog.to_jsonb(
                pg_catalog.to_char(
                    (v_solicitud_peticion #>> '{periodo,fin}')::timestamptz AT TIME ZONE 'UTC',
                    'YYYY-MM-DD'
                )
            ),false
        );
    END IF;
    IF v_solicitud_peticion #> '{rc,existe}'='false'::jsonb THEN
        v_solicitud_peticion:=pg_catalog.jsonb_set(
            v_solicitud_peticion,'{rc,numero}',
            CASE WHEN v_solicitud_peticion #> '{rc,numero}' IS NULL
                       OR v_solicitud_peticion #> '{rc,numero}'='""'::jsonb
                 THEN '""'::jsonb ELSE v_solicitud_peticion #> '{rc,numero}' END,true
        );
        v_solicitud_peticion:=pg_catalog.jsonb_set(
            v_solicitud_peticion,'{rc,fecha}',
            CASE WHEN v_solicitud_peticion #> '{rc,fecha}' IS NULL
                       OR v_solicitud_peticion #> '{rc,fecha}'='"0001-01-01T00:00:00Z"'::jsonb
                 THEN '""'::jsonb ELSE v_solicitud_peticion #> '{rc,fecha}' END,true
        );
        v_solicitud_peticion:=pg_catalog.jsonb_set(
            v_solicitud_peticion,'{rc,importe}',
            CASE WHEN v_solicitud_peticion #> '{rc,importe}' IS NULL
                       OR v_solicitud_peticion #> '{rc,importe}'='{"centimos":0,"moneda":""}'::jsonb
                 THEN '{"centimos":0,"moneda":"EUR"}'::jsonb
                 ELSE v_solicitud_peticion #> '{rc,importe}' END,true
        );
        v_solicitud_peticion:=pg_catalog.jsonb_set(
            v_solicitud_peticion,'{rc,documento_ref}',
            CASE WHEN v_solicitud_peticion #> '{rc,documento_ref}' IS NULL
                       OR v_solicitud_peticion #> '{rc,documento_ref}'='""'::jsonb
                 THEN '""'::jsonb ELSE v_solicitud_peticion #> '{rc,documento_ref}' END,true
        );
    ELSIF v_solicitud_peticion #> '{rc,existe}'='true'::jsonb
       AND pg_catalog.jsonb_typeof(v_solicitud_peticion #> '{rc,fecha}')='string'
       AND v_solicitud_peticion #>> '{rc,fecha}'~'^[0-9]{4}-[0-9]{2}-[0-9]{2}T00:00:00Z$' THEN
        v_solicitud_peticion:=pg_catalog.jsonb_set(
            v_solicitud_peticion,'{rc,fecha}',pg_catalog.to_jsonb(
                pg_catalog.to_char(
                    (v_solicitud_peticion #>> '{rc,fecha}')::timestamptz AT TIME ZONE 'UTC',
                    'YYYY-MM-DD'
                )
            ),false
        );
    END IF;
    IF v_solicitud_peticion->'observaciones' IS NULL THEN
        v_solicitud_peticion:=pg_catalog.jsonb_set(
            v_solicitud_peticion,'{observaciones}','""'::jsonb,true
        );
    END IF;
    IF jsonb_typeof(v_alta) IS DISTINCT FROM 'object'
       OR v_alta->>'esquema' IS DISTINCT FROM 'vec.contratacion-temporal.efecto-alta.v2'
       OR jsonb_typeof(v_alta->'solicitud') IS DISTINCT FROM 'object'
       OR v_alta->>'expediente_ref' IS DISTINCT FROM r->>'expediente_ref'
       OR v_alta->>'numero_visible' IS DISTINCT FROM r->>'numero_visible'
       OR v_alta->>'recibo_ref' IS DISTINCT FROM r->>'recibo_ref'
       OR v_alta->>'organizacion_ref' IS DISTINCT FROM 'organizacion:desarrollo:dipgra'
       OR v_alta->>'actor_ref' IS DISTINCT FROM v_actor_ref
       OR v_alta->>'perfil_ref' IS DISTINCT FROM v_perfil_efecto_ref
       OR v_alta->>'version' IS DISTINCT FROM '1'
       OR v_alta->'solicitud' IS DISTINCT FROM v_solicitud_peticion THEN
        RAISE EXCEPTION 'alta ajena a la petición preparada' USING ERRCODE='P0683';
    END IF;

    BEGIN
        INSERT INTO vec_contratacion_temporal.entrega_peticion_centro_confirmacion(
            peticion_ref,entrega_ref,expediente_ref,numero_visible,version_alta,recibo_ref,
            auditoria_alta_ref,evento_alta_ref,alta_confirmada_en,ambito_alta_hmac,
            ambito_alta_raiz_hmac,confirmacion_alta_ref,recibo_alta,acceso_confirmacion_ref,registrada_en)
        VALUES(v_peticion_ref,'entrega:peticion-centro:'||gen_random_uuid()::text,
            r->>'expediente_ref',r->>'numero_visible',1,r->>'recibo_ref',r->>'auditoria_ref',r->>'evento_ref',
            v_alta_confirmada_en,v_ambito_alta_hmac,v_ambito_alta_raiz_hmac,
            v_confirmacion_alta_ref,r,v_acceso_ref,v_fecha)
        RETURNING * INTO v_confirmacion;
        INSERT INTO vec_contratacion_temporal.entrega_peticion_centro_outbox(
            evento_ref,peticion_ref,expediente_ref,recibo_ref,tipo,carga_json,creada_en)
        VALUES('evento:entrega-peticion-centro:'||gen_random_uuid()::text,v_peticion_ref,
            v_confirmacion.expediente_ref,v_confirmacion.recibo_ref,
            'contratacion_temporal.peticion_centro.entregada_rrhh.v1',
            jsonb_build_object('entrega_ref',v_confirmacion.entrega_ref,'peticion_ref',v_peticion_ref,
                'expediente_ref',v_confirmacion.expediente_ref,'recibo_ref',v_confirmacion.recibo_ref),v_fecha);
    EXCEPTION WHEN unique_violation THEN
        RAISE EXCEPTION 'alta ya enlazada por otra entrega' USING ERRCODE='P0681';
    END;
    RETURN jsonb_build_object('peticion',v_peticion.peticion,'estado_entrega','confirmada',
        'clave_alta',v_reserva.clave_alta::text,'ambito_alta_hmac',v_reserva.ambito_alta_hmac,'actor_ref',v_actor_ref,'perfil_ref',v_reserva.perfil_ref,
        'recibo_alta',v_confirmacion.recibo_alta,
        'reserva_creada_ahora',false,'confirmacion_creada_ahora',true);
END
$_$;
DO $post$
DECLARE f regprocedure:='vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF (SELECT proowner<>'vec_contratacion_temporal_propietario'::regrole OR NOT prosecdef
       OR proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp',
         'row_security=on','TimeZone=UTC','lock_timeout=2s',
         'statement_timeout=15s','idle_in_transaction_session_timeout=20s']::text[]
       OR encode(sha256(convert_to(prosrc,'UTF8')),'hex')
         IS DISTINCT FROM '1faa7d805e292180a5d16f41f05afe18e1c4fd1f05a966d64b44dd8aa80be02a'
      FROM pg_proc WHERE oid=f) IS NOT FALSE
 THEN RAISE EXCEPTION 'CT215 DOWN: CT151 restaurada con metadatos incompatibles' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
