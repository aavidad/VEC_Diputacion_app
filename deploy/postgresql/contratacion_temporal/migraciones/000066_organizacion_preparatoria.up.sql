\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000066',0));

-- La organización es configuración preparatoria de Personal utilizada por CT.
-- No sustituye la RPT oficial, asigna perfiles ni cambia solicitudes existentes.
CREATE TABLE vec_contratacion_temporal.organizacion_catalogo_revision (
    version integer NOT NULL CHECK (version>0),
    revision integer NOT NULL CHECK (revision>0),
    catalogo text NOT NULL CHECK (octet_length(catalogo) BETWEEN 1 AND 1048576),
    huella_sha256 text NOT NULL CHECK (huella_sha256=encode(sha256(convert_to(catalogo,'UTF8')),'hex')),
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (version,revision)
);
CREATE TABLE vec_contratacion_temporal.organizacion_cambio (
    clave_idempotencia uuid PRIMARY KEY,
    version integer NOT NULL,
    revision integer NOT NULL,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    material text NOT NULL CHECK (octet_length(material) BETWEEN 1 AND 2097152),
    solicitud jsonb NOT NULL CHECK (jsonb_typeof(solicitud)='object'),
    recibo_ref text NOT NULL UNIQUE,
    recibo_json jsonb NOT NULL CHECK (jsonb_typeof(recibo_json)='object'),
    auditoria_ref text NOT NULL UNIQUE,
    decision_ref text NOT NULL UNIQUE,
    consumo_huella_sha256 text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (version,revision) REFERENCES vec_contratacion_temporal.organizacion_catalogo_revision,
    UNIQUE (version,revision)
);
CREATE TABLE vec_contratacion_temporal.organizacion_outbox (
    evento_ref text PRIMARY KEY,
    clave_idempotencia uuid NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.organizacion_cambio,
    tipo text NOT NULL CHECK (tipo='personal.organizacion.borrador_actualizado'),
    carga_json jsonb NOT NULL,
    creada_en timestamptz(6) NOT NULL
);
DO $seguridad$
DECLARE v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY['organizacion_catalogo_revision','organizacion_cambio','organizacion_outbox'] LOOP
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY',v_tabla);
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY',v_tabla);
        EXECUTE format('CREATE POLICY propietario ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true)',v_tabla);
        EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.%I FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()',v_tabla);
        EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM PUBLIC,vec_contratacion_temporal_ejecutor',v_tabla);
    END LOOP;
END
$seguridad$;

CREATE FUNCTION vec_contratacion_temporal.inicializar_organizacion_preparatoria_v1(p_catalogo text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s'
AS $funcion$
DECLARE c jsonb; v_huella text;
BEGIN
    IF p_catalogo IS NULL OR octet_length(p_catalogo) NOT BETWEEN 1 AND 1048576 THEN
        RAISE EXCEPTION 'semilla de organización inválida' USING ERRCODE='P0660';
    END IF;
    c:=p_catalogo::jsonb;
    IF c->>'id' IS DISTINCT FROM 'estructura-organizativa-dipgra'
       OR c->>'modulo_id' IS DISTINCT FROM 'personal'
       OR c->>'version' IS DISTINCT FROM '1' OR c->>'revision' IS DISTINCT FROM '1'
       OR c->>'estado' IS DISTINCT FROM 'borrador'
       OR jsonb_typeof(c->'entradas') IS DISTINCT FROM 'array' THEN
        RAISE EXCEPTION 'semilla de organización incompatible' USING ERRCODE='P0660';
    END IF;
    v_huella:=encode(sha256(convert_to(p_catalogo,'UTF8')),'hex');
    PERFORM pg_advisory_xact_lock(hashtextextended('vec:organizacion:1',0));
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.organizacion_catalogo_revision WHERE version=1) THEN
        IF NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.organizacion_catalogo_revision WHERE version=1 AND revision=1 AND huella_sha256=v_huella) THEN
            RAISE EXCEPTION 'no sustituir semilla existente' USING ERRCODE='P0661';
        END IF;
        RETURN;
    END IF;
    INSERT INTO vec_contratacion_temporal.organizacion_catalogo_revision(version,revision,catalogo,huella_sha256)
    VALUES(1,1,p_catalogo,v_huella);
END
$funcion$;
-- Igual que las migraciones, la semilla se ejecuta con SET ROLE propietario.
-- El migrador puede asumir ese rol, pero no tiene USAGE directo del esquema.
REVOKE ALL ON FUNCTION vec_contratacion_temporal.inicializar_organizacion_preparatoria_v1(text) FROM PUBLIC,vec_contratacion_temporal_ejecutor,vec_contratacion_temporal_migrador;

CREATE FUNCTION vec_contratacion_temporal.obtener_catalogo_organizacion_v1(p_version integer)
RETURNS bytea LANGUAGE sql STABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on
AS $funcion$
    SELECT convert_to(catalogo,'UTF8') FROM vec_contratacion_temporal.organizacion_catalogo_revision
    WHERE version=p_version ORDER BY revision DESC LIMIT 1
$funcion$;
CREATE FUNCTION vec_contratacion_temporal.listar_catalogos_organizacion_v1()
RETURNS SETOF bytea LANGUAGE sql STABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on
AS $funcion$
    SELECT convert_to(catalogo,'UTF8') FROM (
        SELECT DISTINCT ON (version) version,revision,catalogo
        FROM vec_contratacion_temporal.organizacion_catalogo_revision
        ORDER BY version DESC,revision DESC LIMIT 65
    ) AS versiones ORDER BY version DESC
$funcion$;
CREATE FUNCTION vec_contratacion_temporal.obtener_cambio_organizacion_v1(p_clave uuid)
RETURNS bytea LANGUAGE sql STABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on
AS $funcion$
    SELECT convert_to(material,'UTF8') FROM vec_contratacion_temporal.organizacion_cambio WHERE clave_idempotencia=p_clave
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.obtener_catalogo_organizacion_v1(integer),
    vec_contratacion_temporal.listar_catalogos_organizacion_v1(),
    vec_contratacion_temporal.obtener_cambio_organizacion_v1(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.obtener_catalogo_organizacion_v1(integer),
    vec_contratacion_temporal.listar_catalogos_organizacion_v1(),
    vec_contratacion_temporal.obtener_cambio_organizacion_v1(uuid) TO vec_contratacion_temporal_ejecutor;

CREATE FUNCTION vec_contratacion_temporal.registrar_cambio_organizacion_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
    m jsonb; s jsonb; c jsonb; d jsonb; anterior jsonb; unidad jsonb;
    v_version integer; v_revision integer; v_clave uuid; v_catalogo text;
    v_material_huella text; v_contexto_huella text; v_catalogo_huella text;
    v_consumo record; v_actual vec_contratacion_temporal.organizacion_catalogo_revision%ROWTYPE;
    v_previa vec_contratacion_temporal.organizacion_cambio%ROWTYPE;
    v_fecha timestamptz(6); v_recibo_ref text; v_recibo jsonb;
BEGIN
    IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 2097152 THEN
        RAISE EXCEPTION 'cambio de organización inválido' USING ERRCODE='P0660';
    END IF;
    BEGIN
        m:=p_material::jsonb; s:=m->'solicitud'; v_catalogo:=m->>'catalogo_canonico'; c:=v_catalogo::jsonb;
        v_version:=(s->>'catalogo_version')::integer;
        v_revision:=(s->>'catalogo_revision')::integer;
        v_clave:=(s->>'clave_idempotencia')::uuid;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'material de organización inválido' USING ERRCODE='P0660';
    END;
    IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_typeof(s) IS DISTINCT FROM 'object'
       OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR v_catalogo IS NULL
       OR octet_length(v_catalogo) NOT BETWEEN 1 AND 1048576
       OR v_version IS NULL OR v_version<1 OR v_revision IS NULL OR v_revision<1 OR v_clave IS NULL
       OR c->>'id' IS DISTINCT FROM 'estructura-organizativa-dipgra'
       OR c->>'modulo_id' IS DISTINCT FROM 'personal' OR c->>'estado' IS DISTINCT FROM 'borrador'
       OR c->>'version' IS DISTINCT FROM v_version::text
       OR c->>'revision' IS DISTINCT FROM (v_revision::bigint+1)::text
       OR c->>'ultima_modificacion_por' IS DISTINCT FROM m->>'actor_id'
       OR c->>'motivo_modificacion' IS DISTINCT FROM s->>'motivo'
       OR jsonb_typeof(c->'entradas') IS DISTINCT FROM 'array'
       OR coalesce(s->>'huella_esperada','') !~ '^[0-9a-f]{64}$' THEN
        RAISE EXCEPTION 'catálogo de organización incompatible' USING ERRCODE='P0660';
    END IF;
    IF jsonb_array_length(c->'entradas')>1000 THEN
        RAISE EXCEPTION 'organización demasiado grande' USING ERRCODE='P0660';
    END IF;
    v_material_huella:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
    v_catalogo_huella:=encode(sha256(convert_to(v_catalogo,'UTF8')),'hex');
    v_contexto_huella:=encode(sha256(convert_to(
        '{"ambitos":{"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"material_sha256":"'||v_material_huella||'"}}','UTF8')),'hex');
    IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
        RAISE EXCEPTION 'decisión de organización inválida' USING ERRCODE='P0663';
    END IF;
    BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'decisión de organización inválida' USING ERRCODE='P0663';
    END;
    IF d->>'accion' IS DISTINCT FROM 'personal.organizacion.actualizar'
       OR d->>'modulo_id' IS DISTINCT FROM 'personal'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'estructura_organizativa'
       OR d->>'finalidad' IS DISTINCT FROM 'gestionar_estructura_organizativa'
       OR d->>'recurso_ref' IS DISTINCT FROM 'estructura-organizativa-dipgra:'||v_version::text
       OR d->>'principal_id' IS DISTINCT FROM m->>'actor_id'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'autorización de organización divergente' USING ERRCODE='P0663';
    END IF;
    SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_organizacion_preparacion_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_huella
       OR v_consumo.efecto_ref IS DISTINCT FROM d->>'recurso_ref' THEN
        RAISE EXCEPTION 'consumo de organización divergente' USING ERRCODE='P0663';
    END IF;
    PERFORM pg_advisory_xact_lock(hashtextextended('vec:organizacion:cambio:'||v_clave::text,0));
    SELECT * INTO v_previa FROM vec_contratacion_temporal.organizacion_cambio WHERE clave_idempotencia=v_clave;
    IF FOUND THEN
        IF v_previa.actor_ref IS DISTINCT FROM d->>'principal_id' OR v_previa.perfil_ref IS DISTINCT FROM d->>'perfil_activo_ref' THEN
            RAISE EXCEPTION 'recibo de organización ajeno' USING ERRCODE='P0663';
        END IF;
        IF v_previa.solicitud IS DISTINCT FROM s OR v_previa.material IS DISTINCT FROM p_material THEN
            RAISE EXCEPTION 'clave de organización divergente' USING ERRCODE='P0661';
        END IF;
        RETURN v_previa.recibo_json||jsonb_build_object('estado_local','replay_confirmado');
    END IF;
    PERFORM pg_advisory_xact_lock(hashtextextended('vec:organizacion:'||v_version::text,0));
    SELECT * INTO v_actual FROM vec_contratacion_temporal.organizacion_catalogo_revision
    WHERE version=v_version ORDER BY revision DESC LIMIT 1;
    IF NOT FOUND OR v_actual.revision<>v_revision OR v_actual.huella_sha256 IS DISTINCT FROM s->>'huella_esperada' THEN
        RAISE EXCEPTION 'organización ha cambiado; recargar' USING ERRCODE='P0662';
    END IF;
    anterior:=v_actual.catalogo::jsonb;
    IF anterior->>'estado' IS DISTINCT FROM 'borrador'
       OR (anterior-ARRAY['entradas','revision','ultima_modificacion_por','ultima_modificacion_en','motivo_modificacion']) IS DISTINCT FROM
          (c-ARRAY['entradas','revision','ultima_modificacion_por','ultima_modificacion_en','motivo_modificacion']) THEN
        RAISE EXCEPTION 'no modificar identidad ni publicación del catálogo' USING ERRCODE='P0660';
    END IF;
    -- El permiso está ligado a todo el material; además se limita la escritura
    -- al único objeto anunciado en el formulario, preservando las demás unidades.
    IF (SELECT count(*) FROM jsonb_array_elements(c->'entradas') AS e WHERE e->>'clave'=s->'unidad'->>'clave')<>1 THEN
        RAISE EXCEPTION 'unidad de cambio no única' USING ERRCODE='P0660';
    END IF;
    SELECT e INTO STRICT unidad FROM jsonb_array_elements(c->'entradas') AS e WHERE e->>'clave'=s->'unidad'->>'clave';
    IF unidad->>'etiqueta' IS DISTINCT FROM s->'unidad'->>'etiqueta'
       OR unidad->'atributos'->>'tipo' IS DISTINCT FROM s->'unidad'->>'tipo'
       OR coalesce(unidad->'atributos'->>'adscripcion_clave','') IS DISTINCT FROM coalesce(s->'unidad'->>'adscripcion_clave','')
       OR unidad->'atributos'->>'modificada_localmente' IS DISTINCT FROM 'si'
       OR (SELECT coalesce(jsonb_agg(e ORDER BY e->>'clave'),'[]'::jsonb) FROM jsonb_array_elements(anterior->'entradas') AS e WHERE e->>'clave'<>s->'unidad'->>'clave')
          IS DISTINCT FROM
          (SELECT coalesce(jsonb_agg(e ORDER BY e->>'clave'),'[]'::jsonb) FROM jsonb_array_elements(c->'entradas') AS e WHERE e->>'clave'<>s->'unidad'->>'clave') THEN
        RAISE EXCEPTION 'cambio distinto de la unidad solicitada' USING ERRCODE='P0660';
    END IF;
    v_fecha:=clock_timestamp();
    v_recibo_ref:='recibo:'||gen_random_uuid()::text;
    v_recibo:=jsonb_build_object('recibo_ref',v_recibo_ref,'registrado_en',v_fecha,
        'catalogo_version',v_version,'catalogo_revision',v_revision+1,
        'huella_anterior',v_actual.huella_sha256,'huella_posterior',v_catalogo_huella,'estado_local','registrado');
    INSERT INTO vec_contratacion_temporal.organizacion_catalogo_revision(version,revision,catalogo,huella_sha256,registrada_en)
        VALUES(v_version,v_revision+1,v_catalogo,v_catalogo_huella,v_fecha);
    INSERT INTO vec_contratacion_temporal.organizacion_cambio(clave_idempotencia,version,revision,actor_ref,perfil_ref,material,solicitud,
        recibo_ref,recibo_json,auditoria_ref,decision_ref,consumo_huella_sha256,registrada_en)
        VALUES(v_clave,v_version,v_revision+1,d->>'principal_id',d->>'perfil_activo_ref',p_material,s,
            v_recibo_ref,v_recibo,v_consumo.auditoria_ref,v_consumo.decision_ref,v_consumo.consumo_huella_sha256,v_fecha);
    INSERT INTO vec_contratacion_temporal.organizacion_outbox(evento_ref,clave_idempotencia,tipo,carga_json,creada_en)
        VALUES('evento:'||gen_random_uuid()::text,v_clave,'personal.organizacion.borrador_actualizado',v_recibo,v_fecha);
    RETURN v_recibo;
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_cambio_organizacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_cambio_organizacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;
COMMIT;
