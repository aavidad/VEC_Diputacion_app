\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:entrega-peticion-centro:dependencia:000024-000068',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000068',0));

DO $prevalidacion$
BEGIN
    IF to_regclass('vec_contratacion_temporal.peticion_centro_revision') IS NULL
       OR to_regclass('vec_contratacion_temporal.alias_ambito_alta') IS NULL
       OR to_regclass('vec_contratacion_temporal.confirmacion_agregado_alta') IS NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_entrega_peticion_centro_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regclass('vec_contratacion_temporal.entrega_peticion_centro_reserva') IS NOT NULL
       OR to_regprocedure('vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
        RAISE EXCEPTION 'estado incompatible para entrega de petición de centro' USING ERRCODE='55000';
    END IF;
END
$prevalidacion$;

-- Una fila por uso autorizado. No conserva la petición, el recibo ni la clave.
CREATE TABLE vec_contratacion_temporal.entrega_peticion_centro_acceso (
    acceso_ref text PRIMARY KEY,
    modo text NOT NULL CHECK (modo IN ('bandeja','preparar','confirmar')),
    recurso_ref text NOT NULL,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
    auditoria_ref text NOT NULL UNIQUE,
    decision_ref text NOT NULL UNIQUE,
    consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
    registrada_en timestamptz(6) NOT NULL CHECK (registrada_en=date_trunc('microseconds',registrada_en)),
    CHECK (acceso_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (recurso_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (actor_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (perfil_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
);

-- La reserva es la recuperación honesta entre el alta existente y el enlace.
-- Clave y HMAC son candidatos creados por el servidor antes del alta. El
-- primer uso los fija juntos; ningún replay puede sustituir el par reservado.
CREATE TABLE vec_contratacion_temporal.entrega_peticion_centro_reserva (
    peticion_ref text PRIMARY KEY,
    peticion_version smallint NOT NULL CHECK (peticion_version=2),
    clave_alta uuid NOT NULL UNIQUE,
    ambito_alta_hmac text NOT NULL UNIQUE,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    acceso_preparacion_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.entrega_peticion_centro_acceso(acceso_ref),
    reservada_en timestamptz(6) NOT NULL CHECK (reservada_en=date_trunc('microseconds',reservada_en)),
    FOREIGN KEY (peticion_ref,peticion_version)
        REFERENCES vec_contratacion_temporal.peticion_centro_revision(peticion_ref,version),
    CHECK (actor_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (perfil_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (clave_alta::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    CHECK (clave_alta::text<>'00000000-0000-4000-8000-000000000000'),
    CHECK (ambito_alta_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'),
    CHECK (right(ambito_alta_hmac,64)<>repeat('0',64))
);

CREATE TABLE vec_contratacion_temporal.entrega_peticion_centro_confirmacion (
    peticion_ref text PRIMARY KEY REFERENCES vec_contratacion_temporal.entrega_peticion_centro_reserva(peticion_ref),
    entrega_ref text NOT NULL UNIQUE,
    expediente_ref text NOT NULL UNIQUE,
    numero_visible text NOT NULL,
    version_alta numeric(20,0) NOT NULL CHECK (version_alta=1),
    recibo_ref text NOT NULL UNIQUE,
    auditoria_alta_ref text NOT NULL,
    evento_alta_ref text NOT NULL,
    alta_confirmada_en timestamptz(6) NOT NULL CHECK (alta_confirmada_en=date_trunc('microseconds',alta_confirmada_en)),
    ambito_alta_hmac text NOT NULL REFERENCES vec_contratacion_temporal.alias_ambito_alta(alias_hmac),
    ambito_alta_raiz_hmac text NOT NULL REFERENCES vec_contratacion_temporal.identidad_reserva_alta(ambito_hmac),
    confirmacion_alta_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.confirmacion_agregado_alta(confirmacion_ref),
    recibo_alta jsonb NOT NULL CHECK (jsonb_typeof(recibo_alta)='object'),
    acceso_confirmacion_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.entrega_peticion_centro_acceso(acceso_ref),
    registrada_en timestamptz(6) NOT NULL CHECK (registrada_en=date_trunc('microseconds',registrada_en)),
    CHECK (entrega_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (expediente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (numero_visible ~ '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$'),
    CHECK (recibo_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (auditoria_alta_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (evento_alta_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (ambito_alta_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'),
    CHECK (right(ambito_alta_hmac,64)<>repeat('0',64)),
    CHECK (ambito_alta_raiz_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'),
    CHECK (right(ambito_alta_raiz_hmac,64)<>repeat('0',64))
);

CREATE TABLE vec_contratacion_temporal.entrega_peticion_centro_outbox (
    evento_ref text PRIMARY KEY,
    peticion_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.entrega_peticion_centro_confirmacion(peticion_ref),
    expediente_ref text NOT NULL UNIQUE,
    recibo_ref text NOT NULL UNIQUE,
    tipo text NOT NULL CHECK (tipo='contratacion_temporal.peticion_centro.entregada_rrhh.v1'),
    carga_json jsonb NOT NULL CHECK (jsonb_typeof(carga_json)='object'),
    creada_en timestamptz(6) NOT NULL CHECK (creada_en=date_trunc('microseconds',creada_en)),
    CHECK (evento_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    FOREIGN KEY (expediente_ref) REFERENCES vec_contratacion_temporal.entrega_peticion_centro_confirmacion(expediente_ref),
    FOREIGN KEY (recibo_ref) REFERENCES vec_contratacion_temporal.entrega_peticion_centro_confirmacion(recibo_ref)
);

DO $seguridad$
DECLARE v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'entrega_peticion_centro_acceso','entrega_peticion_centro_reserva',
        'entrega_peticion_centro_confirmacion','entrega_peticion_centro_outbox'
    ] LOOP
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY',v_tabla);
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY',v_tabla);
        EXECUTE format('CREATE POLICY propietario ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true)',v_tabla);
        EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.%I FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()',v_tabla);
        EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM PUBLIC,vec_contratacion_temporal_ejecutor,vec_contratacion_temporal_migrador',v_tabla);
    END LOOP;
END
$seguridad$;

CREATE FUNCTION vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' SET idle_in_transaction_session_timeout='20s'
AS $funcion$
DECLARE
    m jsonb; d jsonb; r jsonb; v_alta jsonb; v_solicitud_peticion jsonb;
    v_claves text[]; v_claves_recibo text[];
    v_modo text; v_accion text; v_recurso text; v_peticion_ref text;
    v_actor_ref text; v_perfil_ref text; v_ambito_alta_hmac text;
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
        IF v_claves IS DISTINCT FROM ARRAY['actor_ref','ambito_alta_hmac','clave_alta_candidata','modo','perfil_ref','peticion_ref','version_esperada']::text[]
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
        IF v_claves IS DISTINCT FROM ARRAY['actor_ref','ambito_alta_hmac','modo','perfil_ref','peticion_ref','recibo_alta','version_esperada']::text[]
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

    v_material_sha256:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
    v_contexto:='{"ambitos":{"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"material_sha256":"'||v_material_sha256||'"}}';
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

    IF v_modo='preparar' THEN
        IF NOT FOUND THEN
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
                'clave_alta',v_reserva.clave_alta::text,'ambito_alta_hmac',v_reserva.ambito_alta_hmac,'actor_ref',v_actor_ref,'perfil_ref',v_perfil_ref);
        END IF;
        SELECT * INTO v_confirmacion FROM vec_contratacion_temporal.entrega_peticion_centro_confirmacion
         WHERE peticion_ref=v_peticion_ref;
        IF FOUND THEN
            IF v_reserva.actor_ref IS DISTINCT FROM v_actor_ref OR v_reserva.perfil_ref IS DISTINCT FROM v_perfil_ref THEN
                RAISE EXCEPTION 'petición confirmada por otra identidad RRHH' USING ERRCODE='P0681';
            END IF;
            RETURN jsonb_build_object('peticion',v_peticion.peticion,'estado_entrega','confirmada',
                'clave_alta',v_reserva.clave_alta::text,'ambito_alta_hmac',v_reserva.ambito_alta_hmac,'actor_ref',v_actor_ref,'perfil_ref',v_perfil_ref,
                'recibo_alta',v_confirmacion.recibo_alta);
        END IF;
        IF v_reserva.actor_ref IS DISTINCT FROM v_actor_ref OR v_reserva.perfil_ref IS DISTINCT FROM v_perfil_ref THEN
            RAISE EXCEPTION 'petición preparada por otra identidad RRHH' USING ERRCODE='P0681';
        END IF;
        RETURN jsonb_build_object('peticion',v_peticion.peticion,'estado_entrega','preparada',
            'clave_alta',v_reserva.clave_alta::text,'ambito_alta_hmac',v_reserva.ambito_alta_hmac,'actor_ref',v_actor_ref,'perfil_ref',v_perfil_ref);
    END IF;

    IF NOT FOUND OR v_reserva.actor_ref IS DISTINCT FROM v_actor_ref
       OR v_reserva.perfil_ref IS DISTINCT FROM v_perfil_ref THEN
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
            'clave_alta',v_reserva.clave_alta::text,'ambito_alta_hmac',v_reserva.ambito_alta_hmac,'actor_ref',v_actor_ref,'perfil_ref',v_perfil_ref,
            'recibo_alta',v_confirmacion.recibo_alta);
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
           AND identidad.actor_ref=v_actor_ref AND identidad.perfil_ref=v_perfil_ref
           AND alta.organizacion_ref=identidad.organizacion_ref
           AND alta.actor_ref=v_actor_ref AND alta.perfil_ref=v_perfil_ref
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
       OR v_alta->>'perfil_ref' IS DISTINCT FROM v_perfil_ref
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
        'clave_alta',v_reserva.clave_alta::text,'ambito_alta_hmac',v_reserva.ambito_alta_hmac,'actor_ref',v_actor_ref,'perfil_ref',v_perfil_ref,
        'recibo_alta',v_confirmacion.recibo_alta);
END
$funcion$;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC,vec_contratacion_temporal_migrador,
    vec_autorizacion_atestada_v3_consumidor,vec_autorizacion_atestada_v3_emisor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_ejecutor;
COMMIT;
