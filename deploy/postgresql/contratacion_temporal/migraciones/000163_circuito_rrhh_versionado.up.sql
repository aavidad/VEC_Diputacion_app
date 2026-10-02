\set ON_ERROR_STOP on
-- CT163. Guardas del circuito RRHH ligado a la definición de flujo del alta.
-- Las funciones no conceden una operación: las confirmaciones CT existentes
-- consumen su autorización nominal V3 y añaden los hitos que les corresponden.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000163', 0));

DO $pre$
BEGIN
    IF pg_catalog.to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NULL
       OR pg_catalog.to_regclass('vec_contratacion_temporal.expediente_integral_actual') IS NULL
       OR pg_catalog.to_regprocedure('vec_contratacion_temporal.circuito_vinculado_ct163(jsonb)') IS NOT NULL
       OR pg_catalog.to_regprocedure('vec_contratacion_temporal.circuito_siguiente_ct163(jsonb,jsonb)') IS NOT NULL
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_circuito_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
        RAISE EXCEPTION 'CT163: dependencias circuito incompatibles: consulta AD151 instalada=%, requerida=true',
            pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_circuito_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
            USING ERRCODE = '55000';
    END IF;
END
$pre$;

-- El circuito pertenece al mismo agregado y a la misma definición inmutable
-- que se asignó al expediente. Un hito no puede introducir otro flujo.
CREATE FUNCTION vec_contratacion_temporal.circuito_vinculado_ct163(p_agregado jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE STRICT SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_circuito jsonb := p_agregado -> 'circuito';
    v_flujo jsonb := p_agregado -> 'flujo';
BEGIN
    RETURN coalesce(pg_catalog.jsonb_typeof(p_agregado) = 'object'
       AND pg_catalog.jsonb_typeof(v_flujo) = 'object'
       AND pg_catalog.jsonb_typeof(v_circuito) = 'object'
       AND v_circuito -> 'definicion' = v_flujo
       AND pg_catalog.jsonb_typeof(v_circuito -> 'hitos') = 'array'
       AND pg_catalog.jsonb_typeof(v_circuito -> 'estado_actual') = 'string'
       AND p_agregado ->> 'version' ~ '^[1-9][0-9]{0,15}$'
       AND v_flujo ->> 'definicion_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       AND v_flujo ->> 'version' ~ '^[1-9][0-9]{0,15}$'
       AND v_flujo ->> 'huella_sha256' ~ '^[0-9a-f]{64}$', false);
END
$funcion$;

-- Una confirmación añade una versión y un hito, salvo el primer análisis del
-- flujo RRHH publicado: petición firmada y autorización comparten actuación.
-- La cabecera previa, los hitos anteriores y el triple del flujo conservan sus bytes JSONB.
-- Esta guarda no sustituye la evidencia ni la autorización de la operación.
CREATE FUNCTION vec_contratacion_temporal.circuito_siguiente_ct163(
    p_anterior jsonb, p_siguiente jsonb
) RETURNS boolean LANGUAGE plpgsql IMMUTABLE STRICT SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_antes jsonb := p_anterior #> '{circuito,hitos}';
    v_despues jsonb := p_siguiente #> '{circuito,hitos}';
    v_longitud integer;
    v_total integer;
    v_indice integer;
    v_estado text;
    v_hito jsonb;
    v_actuacion jsonb;
BEGIN
    IF NOT vec_contratacion_temporal.circuito_vinculado_ct163(p_anterior)
       OR NOT vec_contratacion_temporal.circuito_vinculado_ct163(p_siguiente)
       OR p_siguiente -> 'flujo' IS DISTINCT FROM p_anterior -> 'flujo'
       OR p_siguiente #> '{circuito,definicion}' IS DISTINCT FROM
          p_anterior #> '{circuito,definicion}'
       OR (p_siguiente ->> 'version')::numeric <> (p_anterior ->> 'version')::numeric + 1
       OR p_siguiente ->> 'referencia' IS DISTINCT FROM p_anterior ->> 'referencia'
       OR p_siguiente ->> 'organizacion_ref' IS DISTINCT FROM p_anterior ->> 'organizacion_ref'
       OR pg_catalog.jsonb_typeof(p_siguiente -> 'actuaciones') IS DISTINCT FROM 'array'
       OR pg_catalog.jsonb_array_length(p_siguiente -> 'actuaciones') <> (p_siguiente ->> 'version')::numeric
       OR (p_siguiente -> 'actuaciones') -
            (pg_catalog.jsonb_array_length(p_siguiente -> 'actuaciones') - 1)
            IS DISTINCT FROM (p_anterior -> 'actuaciones') THEN
        RETURN false;
    END IF;
    v_longitud := pg_catalog.jsonb_array_length(v_antes);
    v_total := pg_catalog.jsonb_array_length(v_despues);
    FOR v_indice IN 0..v_longitud-1 LOOP
        IF v_despues -> v_indice IS DISTINCT FROM v_antes -> v_indice THEN
            RETURN false;
        END IF;
    END LOOP;
    v_estado := p_anterior #>> '{circuito,estado_actual}';
    v_actuacion := p_siguiente -> 'actuaciones' ->
        (pg_catalog.jsonb_array_length(p_siguiente -> 'actuaciones') - 1);
    IF v_longitud = 0 THEN
        IF NOT coalesce(p_anterior ->> 'version' = '1'
            AND v_estado = 'solicitud'
            AND v_actuacion ->> 'accion_clave' = 'contratacion_temporal.analisis.registrar'
            AND v_total = 2
            -- CT164 coteja la huella publicada; aquí se inmoviliza la terna
            -- entre versiones y se reconoce la referencia/version del circuito.
            AND p_anterior #>> '{flujo,definicion_ref}' = 'flujo:ct:rrhh:20261002'
            AND p_anterior #>> '{flujo,version}' = '2'
            AND v_despues -> 0 ->> 'clave' = 'contratacion_temporal.circuito.peticion_firmada'
            AND v_despues -> 0 ->> 'tipo' = 'peticion_firmada'
            AND v_despues -> 0 ->> 'origen' = 'solicitud'
            AND v_despues -> 0 ->> 'destino' = 'autorizacion_rrhh'
            AND v_despues -> 1 ->> 'clave' = 'contratacion_temporal.circuito.autorizacion_rrhh'
            AND v_despues -> 1 ->> 'tipo' = 'autorizacion_rrhh'
            AND v_despues -> 1 ->> 'origen' = 'autorizacion_rrhh'
            AND v_despues -> 1 ->> 'destino' = 'credito', false) THEN
            RETURN false;
        END IF;
    ELSIF v_total <> v_longitud + 1 THEN
        RETURN false;
    END IF;
    FOR v_indice IN v_longitud..v_total-1 LOOP
        v_hito := v_despues -> v_indice;
        IF NOT coalesce(pg_catalog.jsonb_typeof(v_hito) = 'object'
           AND v_hito ->> 'secuencia' = (v_indice + 1)::text
           AND v_hito ->> 'version_expediente_entrada' = p_anterior ->> 'version'
           AND v_hito ->> 'actuacion_clave' = v_actuacion ->> 'accion_clave'
           AND v_hito ->> 'recibo_ref' = v_actuacion ->> 'recibo_ref'
           AND v_hito ->> 'actor_ref' = v_actuacion ->> 'actor_ref'
           AND v_hito ->> 'unidad_ref' = v_actuacion ->> 'unidad_ref'
           AND v_hito -> 'registrado_en' = v_actuacion -> 'realizada_en'
           AND v_hito ->> 'origen' = v_estado
           AND pg_catalog.jsonb_typeof(v_hito -> 'destino') = 'string'
           AND v_hito ->> 'recibo_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$', false) THEN
            RETURN false;
        END IF;
        v_estado := v_hito ->> 'destino';
    END LOOP;
    RETURN v_estado IS NOT DISTINCT FROM p_siguiente #>> '{circuito,estado_actual}';
END
$funcion$;

-- La tabla de versiones es la única autoridad del agregado. Las operaciones
-- anteriores pueden conservar el circuito, pero no cambiar sus hitos ni flujo.
CREATE FUNCTION vec_contratacion_temporal.proteger_version_circuito_ct163()
RETURNS trigger LANGUAGE plpgsql VOLATILE SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_anterior jsonb;
    v_tiene_circuito boolean;
BEGIN
    v_tiene_circuito := NEW.agregado_json ? 'circuito';
    IF NEW.version = 1 THEN
        IF v_tiene_circuito AND (
            NEW.origen_version IS DISTINCT FROM 'alta_o2'
            OR NOT vec_contratacion_temporal.circuito_vinculado_ct163(NEW.agregado_json)
            OR NEW.agregado_json ->> 'version' IS DISTINCT FROM NEW.version::text
            OR NEW.agregado_json #>> '{circuito,definicion,definicion_ref}' IS DISTINCT FROM NEW.flujo_ref
            OR (NEW.agregado_json #>> '{circuito,definicion,version}')::numeric IS DISTINCT FROM NEW.flujo_version
            OR NEW.agregado_json #>> '{circuito,definicion,huella_sha256}' IS DISTINCT FROM NEW.flujo_huella_sha256
            OR pg_catalog.jsonb_array_length(NEW.agregado_json #> '{circuito,hitos}') <> 0
        ) THEN
            RAISE EXCEPTION 'CT163: alta de circuito divergente' USING ERRCODE = '55000';
        END IF;
        RETURN NEW;
    END IF;
    SELECT v.agregado_json INTO v_anterior
      FROM vec_contratacion_temporal.expediente_version_integral v
     WHERE v.expediente_ref = NEW.expediente_ref AND v.version = NEW.version - 1;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'CT163: falta la versión anterior' USING ERRCODE = '55000';
    END IF;
    IF v_tiene_circuito OR v_anterior ? 'circuito' THEN
        IF NOT vec_contratacion_temporal.circuito_vinculado_ct163(NEW.agregado_json)
           OR NEW.agregado_json ->> 'version' IS DISTINCT FROM NEW.version::text
           OR NEW.agregado_json -> 'flujo' IS DISTINCT FROM v_anterior -> 'flujo'
           OR NEW.agregado_json #>> '{circuito,definicion,definicion_ref}' IS DISTINCT FROM NEW.flujo_ref
           OR (NEW.agregado_json #>> '{circuito,definicion,version}')::numeric IS DISTINCT FROM NEW.flujo_version
           OR NEW.agregado_json #>> '{circuito,definicion,huella_sha256}' IS DISTINCT FROM NEW.flujo_huella_sha256
           OR (NEW.agregado_json -> 'circuito' IS DISTINCT FROM v_anterior -> 'circuito'
               AND NOT vec_contratacion_temporal.circuito_siguiente_ct163(v_anterior, NEW.agregado_json)) THEN
            RAISE EXCEPTION 'CT163: transición de circuito divergente' USING ERRCODE = '55000';
        END IF;
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER proteger_version_circuito_ct163
BEFORE INSERT ON vec_contratacion_temporal.expediente_version_integral
FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.proteger_version_circuito_ct163();

-- Preparación de la selección desde crédito v3. Se conserva privada mientras
-- la ejecución O6 no tenga un consumidor nominal de lectura; la pertenencia
-- técnica al rol ejecutor no autoriza ver el agregado con firmas y actores.
CREATE FUNCTION vec_contratacion_temporal.leer_expediente_seleccion_v3(
    p_organizacion text, p_expediente text, p_version bigint
) RETURNS TABLE(expediente_json jsonb, version_actual bigint)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '5s'
AS $funcion$
DECLARE
    v_agregado jsonb;
    v_version bigint;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR session_user = current_user
       OR NOT pg_catalog.pg_has_role(session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
       OR pg_catalog.pg_has_role(session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
       OR p_organizacion IS NULL OR p_organizacion !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_expediente IS NULL OR p_expediente !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_version IS DISTINCT FROM 3 THEN
        RAISE EXCEPTION 'preparación de oferta de circuito denegada' USING ERRCODE = '42501';
    END IF;
    SELECT v.agregado_json, a.version::bigint INTO v_agregado, v_version
      FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v
        ON v.expediente_ref = a.expediente_ref AND v.version = a.version
     WHERE a.expediente_ref = p_expediente AND a.version = p_version
       AND v.agregado_json ->> 'organizacion_ref' = p_organizacion
       AND v.agregado_json ->> 'fase_actual' = 'asignacion_unidad'
       AND v.agregado_json ->> 'estado_actual' = 'en_curso';
    IF NOT FOUND OR NOT vec_contratacion_temporal.circuito_vinculado_ct163(v_agregado)
       OR NOT EXISTS (
            SELECT 1 FROM pg_catalog.jsonb_array_elements(v_agregado #> '{circuito,hitos}') h
             WHERE h ->> 'credito_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       ) THEN
        RAISE EXCEPTION 'preparación de oferta de circuito denegada' USING ERRCODE = '42501';
    END IF;
    RETURN QUERY SELECT v_agregado, v_version;
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.leer_expediente_seleccion_v3(
    text,text,bigint) FROM PUBLIC, vec_contratacion_temporal_ejecutor;

-- Consulta solo la cabeza vigente. La autorización y la lectura comparten
-- transacción: el ejecutor no recibe acceso directo a las tablas históricas.
CREATE FUNCTION vec_contratacion_temporal.consultar_circuito_rrhh_v1(
    p_organizacion text, p_expediente text, p_version bigint,
    p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
    p_persona_version numeric, p_perfil_version numeric,
    p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea
) RETURNS TABLE(circuito_json jsonb, version_expediente bigint, flujo_json jsonb)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '5s'
AS $funcion$
DECLARE
    v_material text;
    v_material_huella text;
    v_contexto_huella text;
    v_decision jsonb;
    v_consumo record;
    v_agregado jsonb;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR session_user = current_user
       OR NOT pg_catalog.pg_has_role(session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
       OR pg_catalog.pg_has_role(session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
       OR p_organizacion IS NULL OR p_organizacion !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_expediente IS NULL OR p_expediente !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_version IS NULL OR p_version NOT BETWEEN 1 AND 9007199254740991
       OR p_decision IS NULL OR pg_catalog.octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
        RAISE EXCEPTION 'consulta de circuito denegada' USING ERRCODE = '42501';
    END IF;
    v_material := '{"expediente_ref":"' || p_expediente ||
        '","organizacion_ref":"' || p_organizacion ||
        '","version_expediente":' || p_version::text || '}';
    v_material_huella := pg_catalog.encode(
        pg_catalog.sha256(pg_catalog.convert_to(v_material, 'UTF8')), 'hex');
    v_contexto_huella := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
        '{"ambitos":{"organizacion_ref":"' || p_organizacion ||
        '"},"atributos":{"material_sha256":"' || v_material_huella || '"}}', 'UTF8')), 'hex');
    v_decision := pg_catalog.convert_from(p_decision, 'UTF8')::jsonb;
    IF v_decision ->> 'accion' IS DISTINCT FROM 'contratacion_temporal.circuito.consultar'
       OR v_decision ->> 'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR v_decision ->> 'tipo_recurso' IS DISTINCT FROM 'expediente_circuito_rrhh'
       OR v_decision ->> 'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
       OR v_decision ->> 'recurso_ref' IS DISTINCT FROM p_expediente
       OR v_decision ->> 'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_huella
       OR v_decision -> 'campos_permitidos' IS DISTINCT FROM '["circuito"]'::jsonb
       OR v_decision -> 'obligaciones' IS DISTINCT FROM '["auditar"]'::jsonb THEN
        RAISE EXCEPTION 'autorización de consulta de circuito divergente' USING ERRCODE = '42501';
    END IF;
    SELECT * INTO STRICT v_consumo FROM
        vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_circuito_ct_v3_atestada(
            v_material, p_capacidad, p_decision, p_motivo, p_contexto,
            p_persona_version, p_perfil_version, p_payload, p_sobre,
            p_evidencia, p_raiz);
    IF v_consumo.efecto_ref IS DISTINCT FROM p_expediente
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_huella
       OR v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'consumo de circuito divergente' USING ERRCODE = '42501';
    END IF;
    SELECT v.agregado_json INTO v_agregado
      FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v
        ON v.expediente_ref = a.expediente_ref AND v.version = a.version
     WHERE a.expediente_ref = p_expediente AND a.version = p_version
       AND v.agregado_json ->> 'organizacion_ref' = p_organizacion;
    IF NOT FOUND OR NOT vec_contratacion_temporal.circuito_vinculado_ct163(v_agregado) THEN
        RETURN;
    END IF;
    RETURN QUERY SELECT v_agregado -> 'circuito', p_version, v_agregado -> 'flujo';
END
$funcion$;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_circuito_rrhh_v1(
    text,text,bigint,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_circuito_rrhh_v1(
    text,text,bigint,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_ejecutor;

-- Los lectores anteriores devolvían el agregado entero para el circuito
-- fiscalizado. Para una terna nueva ese agregado contiene fuentes de firma;
-- la preparación/aviso nuevos quedan cerrados hasta lector O6 nominal.
DO $lectores_anteriores$
DECLARE
    v_firma text;
    v_oid oid;
    v_def text;
    v_nueva text;
    v_acl aclitem[];
    v_propietario oid;
    v_config text[];
    v_definidora boolean;
    v_ancla text := 'AND v.agregado_json->>''organizacion_ref'' = p_organizacion';
    v_guardia text := E'\n       AND NOT (v.agregado_json ? ''circuito'')';
BEGIN
    FOREACH v_firma IN ARRAY ARRAY[
        'vec_contratacion_temporal.leer_expediente_seleccion_v1(text,text,bigint)',
        'vec_contratacion_temporal.leer_expediente_seleccion_v2(text,text,bigint)'
    ] LOOP
        v_oid := pg_catalog.to_regprocedure(v_firma);
        IF v_oid IS NULL THEN
            RAISE EXCEPTION 'CT163: lector anterior ausente: %', v_firma USING ERRCODE = '55000';
        END IF;
        SELECT pg_catalog.pg_get_functiondef(p.oid), p.proacl, p.proowner,
               p.proconfig, p.prosecdef
          INTO v_def, v_acl, v_propietario, v_config, v_definidora
          FROM pg_catalog.pg_proc p WHERE p.oid = v_oid;
        IF v_propietario IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
           OR v_definidora IS NOT TRUE
           OR pg_catalog.length(v_def) - pg_catalog.length(pg_catalog.replace(v_def,v_ancla,''))
                <> pg_catalog.length(v_ancla)
           OR pg_catalog.strpos(v_def,v_guardia) <> 0 THEN
            RAISE EXCEPTION 'CT163: preimagen incompatible del lector %', v_firma USING ERRCODE = '55000';
        END IF;
        -- Conservar el predicado original y añadir solo la exclusión. La
        -- sustitución se comprueba en ambas direcciones antes de confirmar.
        v_nueva := pg_catalog.replace(v_def, v_ancla, v_ancla || v_guardia);
        EXECUTE v_nueva;
        IF (SELECT p.proacl FROM pg_catalog.pg_proc p WHERE p.oid=v_oid) IS DISTINCT FROM v_acl
           OR (SELECT p.proowner FROM pg_catalog.pg_proc p WHERE p.oid=v_oid) IS DISTINCT FROM v_propietario
           OR (SELECT p.proconfig FROM pg_catalog.pg_proc p WHERE p.oid=v_oid) IS DISTINCT FROM v_config
           OR (SELECT p.prosecdef FROM pg_catalog.pg_proc p WHERE p.oid=v_oid) IS NOT TRUE
           OR pg_catalog.replace(pg_catalog.pg_get_functiondef(v_oid),
                v_ancla || v_guardia, v_ancla) IS DISTINCT FROM v_def THEN
            RAISE EXCEPTION 'CT163: lector anterior alterado: %', v_firma USING ERRCODE = '55000';
        END IF;
    END LOOP;
END
$lectores_anteriores$;

DO $aviso_anterior$
DECLARE
    v_oid oid := pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.leer_expediente_aviso_confirmado_v1(text,text,text)');
    v_def text;
    v_nueva text;
    v_acl aclitem[];
    v_propietario oid;
    v_config text[];
    v_definidora boolean;
    v_ancla text := $ancla$    v_version := (v_seleccion.solicitud_json->>'version_expediente')::bigint;$ancla$;
    v_guardia text := $guardia$
    IF EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral v
         WHERE v.expediente_ref = p_expediente AND v.version = v_version
           AND v.agregado_json ->> 'organizacion_ref' = p_organizacion
           AND v.agregado_json ? 'circuito'
    ) THEN
        RAISE EXCEPTION 'antecedente de aviso no disponible' USING ERRCODE = '42501';
    END IF;$guardia$;
BEGIN
    IF v_oid IS NULL THEN
        RAISE EXCEPTION 'CT163: lector de aviso anterior ausente' USING ERRCODE = '55000';
    END IF;
    SELECT pg_catalog.pg_get_functiondef(p.oid), p.proacl, p.proowner,
           p.proconfig, p.prosecdef
      INTO v_def, v_acl, v_propietario, v_config, v_definidora
      FROM pg_catalog.pg_proc p WHERE p.oid = v_oid;
    IF v_propietario IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
       OR v_definidora IS NOT TRUE
       OR pg_catalog.length(v_def) - pg_catalog.length(pg_catalog.replace(v_def,v_ancla,''))
            <> pg_catalog.length(v_ancla)
       OR pg_catalog.strpos(v_def,v_guardia) <> 0 THEN
        RAISE EXCEPTION 'CT163: preimagen incompatible del lector de aviso' USING ERRCODE = '55000';
    END IF;
    v_nueva := pg_catalog.replace(v_def, v_ancla, v_ancla || v_guardia);
    EXECUTE v_nueva;
    IF (SELECT p.proacl FROM pg_catalog.pg_proc p WHERE p.oid=v_oid) IS DISTINCT FROM v_acl
       OR (SELECT p.proowner FROM pg_catalog.pg_proc p WHERE p.oid=v_oid) IS DISTINCT FROM v_propietario
       OR (SELECT p.proconfig FROM pg_catalog.pg_proc p WHERE p.oid=v_oid) IS DISTINCT FROM v_config
       OR (SELECT p.prosecdef FROM pg_catalog.pg_proc p WHERE p.oid=v_oid) IS NOT TRUE
       OR pg_catalog.replace(pg_catalog.pg_get_functiondef(v_oid),
            v_ancla || v_guardia, v_ancla) IS DISTINCT FROM v_def THEN
        RAISE EXCEPTION 'CT163: lector de aviso alterado' USING ERRCODE = '55000';
    END IF;
END
$aviso_anterior$;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.circuito_vinculado_ct163(jsonb),
    vec_contratacion_temporal.circuito_siguiente_ct163(jsonb,jsonb),
    vec_contratacion_temporal.proteger_version_circuito_ct163() FROM PUBLIC;
COMMIT;
