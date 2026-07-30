BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE SCHEMA vec_autorizacion AUTHORIZATION vec_autorizacion_propietario;
REVOKE ALL ON SCHEMA vec_autorizacion FROM PUBLIC;

ALTER DEFAULT PRIVILEGES FOR ROLE vec_autorizacion_propietario IN SCHEMA vec_autorizacion
    REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_autorizacion_propietario IN SCHEMA vec_autorizacion
    REVOKE ALL ON SEQUENCES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_autorizacion_propietario IN SCHEMA vec_autorizacion
    REVOKE ALL ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_autorizacion_propietario IN SCHEMA vec_autorizacion
    REVOKE ALL ON TYPES FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.texto_positivo_valido(p_valor text, p_maximo integer)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
	SELECT p_valor IS NOT NULL
	   AND p_maximo > 0
       AND length(p_valor) BETWEEN 1 AND p_maximo
       AND p_valor = btrim(p_valor)
       AND p_valor !~ '[[:space:][:cntrl:]]'
       AND strpos(p_valor, '*') = 0
$funcion$;

CREATE FUNCTION vec_autorizacion.lista_positiva_valida(
    p_lista jsonb,
    p_obligatoria boolean
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    elemento jsonb;
    valor text;
    vistos text[] := ARRAY[]::text[];
BEGIN
    IF jsonb_typeof(p_lista) IS DISTINCT FROM 'array'
       OR jsonb_array_length(p_lista) > 512
       OR (p_obligatoria AND jsonb_array_length(p_lista) = 0) THEN
        RETURN false;
    END IF;
    FOR elemento IN SELECT value FROM jsonb_array_elements(p_lista) LOOP
        IF jsonb_typeof(elemento) IS DISTINCT FROM 'string' THEN
            RETURN false;
        END IF;
        valor := elemento #>> '{}';
		IF vec_autorizacion.texto_positivo_valido(valor, 512) IS NOT TRUE
           OR valor = ANY(vistos) THEN
            RETURN false;
        END IF;
        vistos := array_append(vistos, valor);
    END LOOP;
    RETURN true;
END
$funcion$;

CREATE FUNCTION vec_autorizacion.instante_utc_microsegundo_valido(p_valor text)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
	IF p_valor IS NULL
	   OR p_valor !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}([.][0-9]{1,6})?Z$' THEN
		RETURN false;
	END IF;
	PERFORM p_valor::timestamptz;
	RETURN true;
EXCEPTION WHEN OTHERS THEN
	RETURN false;
END
$funcion$;

CREATE FUNCTION vec_autorizacion.ambitos_positivos_validos(p_documento jsonb)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    ambito jsonb;
    clave text;
    claves text[] := ARRAY[]::text[];
    numero_claves integer;
BEGIN
    IF jsonb_typeof(p_documento -> 'ambitos') IS DISTINCT FROM 'array'
       OR jsonb_array_length(p_documento -> 'ambitos') NOT BETWEEN 1 AND 512 THEN
        RETURN false;
    END IF;
    FOR ambito IN SELECT value FROM jsonb_array_elements(p_documento -> 'ambitos') LOOP
        IF jsonb_typeof(ambito) IS DISTINCT FROM 'object' THEN
            RETURN false;
        END IF;
        SELECT count(*) INTO numero_claves FROM jsonb_object_keys(ambito);
        clave := ambito ->> 'clave';
		IF numero_claves <> 2
		   OR clave = 'global'
		   OR vec_autorizacion.texto_positivo_valido(clave, 128) IS NOT TRUE
		   OR clave = ANY(claves)
		   OR vec_autorizacion.lista_positiva_valida(ambito -> 'valores', true) IS NOT TRUE THEN
            RETURN false;
        END IF;
        claves := array_append(claves, clave);
    END LOOP;
    RETURN true;
END
$funcion$;

CREATE FUNCTION vec_autorizacion.concesiones_positivas_validas(p_documento jsonb)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    concesion jsonb;
    clave text;
    claves text[] := ARRAY[]::text[];
BEGIN
    IF jsonb_typeof(p_documento -> 'concesiones') IS DISTINCT FROM 'array'
       OR jsonb_array_length(p_documento -> 'concesiones') NOT BETWEEN 1 AND 512 THEN
        RETURN false;
    END IF;
    FOR concesion IN SELECT value FROM jsonb_array_elements(p_documento -> 'concesiones') LOOP
		IF jsonb_typeof(concesion) IS DISTINCT FROM 'object'
		   OR vec_autorizacion.texto_positivo_valido(concesion ->> 'accion', 256) IS NOT TRUE
		   OR vec_autorizacion.texto_positivo_valido(concesion ->> 'modulo_id', 128) IS NOT TRUE
		   OR vec_autorizacion.texto_positivo_valido(concesion ->> 'tipo_recurso', 128) IS NOT TRUE
		   OR vec_autorizacion.lista_positiva_valida(concesion -> 'finalidades', true) IS NOT TRUE
		   OR vec_autorizacion.lista_positiva_valida(COALESCE(concesion -> 'campos_permitidos', '[]'::jsonb), false) IS NOT TRUE
		   OR vec_autorizacion.lista_positiva_valida(COALESCE(concesion -> 'obligaciones', '[]'::jsonb), false) IS NOT TRUE
		   OR ((concesion ->> 'garantia_minima') IN ('bajo', 'sustancial', 'alto')) IS NOT TRUE THEN
            RETURN false;
        END IF;
		clave := jsonb_build_array(
		    concesion ->> 'accion', concesion ->> 'modulo_id', concesion ->> 'tipo_recurso'
		)::text;
        IF clave = ANY(claves) THEN
            RETURN false;
        END IF;
        claves := array_append(claves, clave);
    END LOOP;
    RETURN true;
END
$funcion$;

CREATE FUNCTION vec_autorizacion.manifesto_huellas_valido(p_manifesto jsonb)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    entrada record;
    total integer := 0;
BEGIN
    IF jsonb_typeof(p_manifesto) IS DISTINCT FROM 'object' THEN
        RETURN false;
    END IF;
	FOR entrada IN SELECT key, value FROM jsonb_each(p_manifesto) LOOP
		total := total + 1;
		IF total > 512
		   OR vec_autorizacion.texto_positivo_valido(entrada.key, 512) IS NOT TRUE
		   OR jsonb_typeof(entrada.value) IS DISTINCT FROM 'string'
		   OR ((entrada.value #>> '{}') ~ '^[0-9a-f]{64}$') IS NOT TRUE THEN
			RETURN false;
		END IF;
	END LOOP;
	RETURN true;
END
$funcion$;

CREATE FUNCTION vec_autorizacion.entero_uint64_json_valido(p_valor jsonb)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    texto text;
    numero numeric;
BEGIN
    IF jsonb_typeof(p_valor) IS DISTINCT FROM 'number' THEN
        RETURN false;
    END IF;
    texto := p_valor #>> '{}';
    IF texto !~ '^[1-9][0-9]{0,19}$' THEN
        RETURN false;
    END IF;
    numero := texto::numeric;
    RETURN numero BETWEEN 1 AND 18446744073709551615;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

-- Una decision es una capacidad breve, no un contenedor documental. El limite
-- global de 512 KiB admite catalogos amplios y corta antes de almacenar un JSON
-- patologico. Todos los campos, incluidos conjuntos vacios, son obligatorios:
-- una ausencia nunca adquiere semantica de alcance total.
CREATE FUNCTION vec_autorizacion.documento_decision_estructura_valida(p_documento jsonb)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    clave text;
BEGIN
    IF jsonb_typeof(p_documento) IS DISTINCT FROM 'object'
       OR pg_column_size(p_documento) > 524288
       OR (SELECT count(*) FROM jsonb_object_keys(p_documento)) <> 30 THEN
        RETURN false;
    END IF;

    FOREACH clave IN ARRAY ARRAY[
        'decision_ref', 'concedida', 'codigo', 'principal_id', 'perfil_activo_ref',
        'accion', 'recurso_ref', 'modulo_id', 'tipo_recurso',
        'contexto_recurso_huella_sha256', 'finalidad', 'correlacion_ref',
        'asignacion_ref', 'asignacion_huella_sha256', 'version_rol_ref',
        'version_rol_huella_sha256', 'control_vigencia_version_rol_ref',
        'control_vigencia_version_rol_revision',
        'control_vigencia_version_rol_huella_sha256', 'revision_catalogo_politicas',
        'catalogo_politicas_huella_sha256', 'politicas_evaluadas_refs',
        'politicas_evaluadas_huellas_sha256', 'politicas_refs',
        'politicas_huellas_sha256', 'garantia_minima', 'campos_permitidos',
        'obligaciones', 'emitida_en', 'valida_hasta'
    ] LOOP
        IF NOT (p_documento ? clave) THEN
            RETURN false;
        END IF;
    END LOOP;

    IF jsonb_typeof(p_documento -> 'concedida') IS DISTINCT FROM 'boolean'
       OR (p_documento -> 'concedida') IS DISTINCT FROM 'true'::jsonb
       OR jsonb_typeof(p_documento -> 'codigo') IS DISTINCT FROM 'string'
       OR (p_documento ->> 'codigo') IS DISTINCT FROM 'concedida'
       OR jsonb_typeof(p_documento -> 'decision_ref') IS DISTINCT FROM 'string'
       OR vec_autorizacion.texto_positivo_valido(p_documento ->> 'decision_ref', 512) IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'principal_id') IS DISTINCT FROM 'string'
       OR vec_autorizacion.texto_positivo_valido(p_documento ->> 'principal_id', 512) IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'perfil_activo_ref') IS DISTINCT FROM 'string'
       OR vec_autorizacion.texto_positivo_valido(p_documento ->> 'perfil_activo_ref', 512) IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'accion') IS DISTINCT FROM 'string'
       OR vec_autorizacion.texto_positivo_valido(p_documento ->> 'accion', 256) IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'recurso_ref') IS DISTINCT FROM 'string'
       OR vec_autorizacion.texto_positivo_valido(p_documento ->> 'recurso_ref', 512) IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'modulo_id') IS DISTINCT FROM 'string'
       OR vec_autorizacion.texto_positivo_valido(p_documento ->> 'modulo_id', 128) IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'tipo_recurso') IS DISTINCT FROM 'string'
       OR vec_autorizacion.texto_positivo_valido(p_documento ->> 'tipo_recurso', 128) IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'finalidad') IS DISTINCT FROM 'string'
       OR vec_autorizacion.texto_positivo_valido(p_documento ->> 'finalidad', 512) IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'correlacion_ref') IS DISTINCT FROM 'string'
       OR vec_autorizacion.texto_positivo_valido(p_documento ->> 'correlacion_ref', 512) IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'asignacion_ref') IS DISTINCT FROM 'string'
       OR vec_autorizacion.texto_positivo_valido(p_documento ->> 'asignacion_ref', 512) IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'version_rol_ref') IS DISTINCT FROM 'string'
       OR vec_autorizacion.texto_positivo_valido(p_documento ->> 'version_rol_ref', 512) IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'control_vigencia_version_rol_ref') IS DISTINCT FROM 'string'
       OR vec_autorizacion.texto_positivo_valido(p_documento ->> 'control_vigencia_version_rol_ref', 512) IS NOT TRUE THEN
        RETURN false;
    END IF;

    IF jsonb_typeof(p_documento -> 'contexto_recurso_huella_sha256') IS DISTINCT FROM 'string'
       OR ((p_documento ->> 'contexto_recurso_huella_sha256') ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'asignacion_huella_sha256') IS DISTINCT FROM 'string'
       OR ((p_documento ->> 'asignacion_huella_sha256') ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'version_rol_huella_sha256') IS DISTINCT FROM 'string'
       OR ((p_documento ->> 'version_rol_huella_sha256') ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'control_vigencia_version_rol_huella_sha256') IS DISTINCT FROM 'string'
       OR ((p_documento ->> 'control_vigencia_version_rol_huella_sha256') ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'catalogo_politicas_huella_sha256') IS DISTINCT FROM 'string'
       OR ((p_documento ->> 'catalogo_politicas_huella_sha256') ~ '^[0-9a-f]{64}$') IS NOT TRUE THEN
        RETURN false;
    END IF;

    IF vec_autorizacion.entero_uint64_json_valido(p_documento -> 'control_vigencia_version_rol_revision') IS NOT TRUE
       OR vec_autorizacion.entero_uint64_json_valido(p_documento -> 'revision_catalogo_politicas') IS NOT TRUE
       OR vec_autorizacion.lista_positiva_valida(p_documento -> 'politicas_evaluadas_refs', false) IS NOT TRUE
       OR vec_autorizacion.manifesto_huellas_valido(p_documento -> 'politicas_evaluadas_huellas_sha256') IS NOT TRUE
       OR vec_autorizacion.lista_positiva_valida(p_documento -> 'politicas_refs', false) IS NOT TRUE
       OR vec_autorizacion.manifesto_huellas_valido(p_documento -> 'politicas_huellas_sha256') IS NOT TRUE
       OR vec_autorizacion.lista_positiva_valida(p_documento -> 'campos_permitidos', false) IS NOT TRUE
       OR vec_autorizacion.lista_positiva_valida(p_documento -> 'obligaciones', false) IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'garantia_minima') IS DISTINCT FROM 'string'
       OR ((p_documento ->> 'garantia_minima') IN ('bajo', 'sustancial', 'alto')) IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'emitida_en') IS DISTINCT FROM 'string'
       OR vec_autorizacion.instante_utc_microsegundo_valido(p_documento ->> 'emitida_en') IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'valida_hasta') IS DISTINCT FROM 'string'
       OR vec_autorizacion.instante_utc_microsegundo_valido(p_documento ->> 'valida_hasta') IS NOT TRUE THEN
        RETURN false;
    END IF;

    RETURN true;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

CREATE TABLE vec_autorizacion.version_rol (
    version_rol_ref text PRIMARY KEY,
    rol_id text NOT NULL,
    version bigint NOT NULL,
    huella_sha256 text NOT NULL,
    publicada_en timestamptz(6) NOT NULL,
    documento jsonb NOT NULL,
    creada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT version_rol_referencia_segura CHECK (
		vec_autorizacion.texto_positivo_valido(version_rol_ref, 512) IS TRUE
    ),
    CONSTRAINT version_rol_id_seguro CHECK (
		vec_autorizacion.texto_positivo_valido(rol_id, 128) IS TRUE
    ),
    CONSTRAINT version_rol_version_positiva CHECK (version > 0),
    CONSTRAINT version_rol_huella_canonica CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT version_rol_documento_objeto CHECK (jsonb_typeof(documento) = 'object'),
    CONSTRAINT version_rol_documento_identidad CHECK (
		(documento ->> 'rol_id') IS NOT DISTINCT FROM rol_id
		AND (documento ->> 'version')::bigint IS NOT DISTINCT FROM version
		AND version_rol_ref IS NOT DISTINCT FROM 'rol:' || rol_id || ':v' || version::text
		AND vec_autorizacion.instante_utc_microsegundo_valido(documento ->> 'publicada_en') IS TRUE
		AND (documento ->> 'publicada_en')::timestamptz IS NOT DISTINCT FROM publicada_en
    ),
    -- RBAC es una lista positiva cerrada. Ningun comodin concede acciones,
    -- modulos, recursos, finalidades, campos u obligaciones futuras.
    CONSTRAINT version_rol_sin_comodines_positivos CHECK (
		NOT jsonb_path_exists(documento, '$.concesiones[*].accion ? (@ like_regex ".*\\*.*")')
		AND NOT jsonb_path_exists(documento, '$.concesiones[*].modulo_id ? (@ like_regex ".*\\*.*")')
		AND NOT jsonb_path_exists(documento, '$.concesiones[*].tipo_recurso ? (@ like_regex ".*\\*.*")')
		AND NOT jsonb_path_exists(documento, '$.concesiones[*].finalidades[*] ? (@ like_regex ".*\\*.*")')
		AND NOT jsonb_path_exists(documento, '$.concesiones[*].campos_permitidos[*] ? (@ like_regex ".*\\*.*")')
		AND NOT jsonb_path_exists(documento, '$.concesiones[*].obligaciones[*] ? (@ like_regex ".*\\*.*")')
		AND vec_autorizacion.concesiones_positivas_validas(documento)
    ),
    UNIQUE (rol_id, version),
    UNIQUE (rol_id, version_rol_ref)
);

CREATE TABLE vec_autorizacion.control_vigencia_version_rol (
    version_rol_ref text NOT NULL REFERENCES vec_autorizacion.version_rol(version_rol_ref),
    revision numeric(20, 0) NOT NULL,
    estado text NOT NULL CHECK (estado IN ('habilitada', 'retirada')),
    huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
    actualizado_en timestamptz(6) NOT NULL,
    documento jsonb NOT NULL CHECK (jsonb_typeof(documento) = 'object'),
    creada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT control_rol_revision_rango CHECK (
        revision BETWEEN 1 AND 18446744073709551615
    ),
    CONSTRAINT control_rol_documento_identidad CHECK (
		(documento ->> 'version_rol_ref') IS NOT DISTINCT FROM version_rol_ref
		AND (documento ->> 'revision')::numeric IS NOT DISTINCT FROM revision
		AND (documento ->> 'estado') IS NOT DISTINCT FROM estado
		AND vec_autorizacion.instante_utc_microsegundo_valido(documento ->> 'actualizado_en') IS TRUE
		AND (documento ->> 'actualizado_en')::timestamptz IS NOT DISTINCT FROM actualizado_en
    ),
    PRIMARY KEY (version_rol_ref, revision),
    UNIQUE (version_rol_ref, revision, huella_sha256)
);

CREATE TABLE vec_autorizacion.control_vigencia_version_rol_actual (
    version_rol_ref text PRIMARY KEY,
    revision numeric(20, 0) NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    actualizada_por text NOT NULL,
    acto_ref text NOT NULL,
    CONSTRAINT control_rol_actual_referencia_segura CHECK (
		vec_autorizacion.texto_positivo_valido(actualizada_por, 512) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(acto_ref, 512) IS TRUE
    ),
    FOREIGN KEY (version_rol_ref, revision)
        REFERENCES vec_autorizacion.control_vigencia_version_rol(version_rol_ref, revision)
);

CREATE TABLE vec_autorizacion.asignacion_perfil (
    asignacion_ref text PRIMARY KEY,
    asignacion_id text NOT NULL,
    version bigint NOT NULL CHECK (version > 0),
    perfil_activo_ref text NOT NULL,
    principal_id text NOT NULL,
    version_rol_ref text NOT NULL REFERENCES vec_autorizacion.version_rol(version_rol_ref),
    huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
    emitida_en timestamptz(6) NOT NULL,
    documento jsonb NOT NULL CHECK (jsonb_typeof(documento) = 'object'),
    creada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT asignacion_referencias_seguras CHECK (
		vec_autorizacion.texto_positivo_valido(asignacion_ref, 512) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(asignacion_id, 512) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(perfil_activo_ref, 512) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(principal_id, 512) IS TRUE
    ),
    CONSTRAINT asignacion_documento_identidad CHECK (
		(documento ->> 'asignacion_id') IS NOT DISTINCT FROM asignacion_id
		AND (documento ->> 'version')::bigint IS NOT DISTINCT FROM version
		AND (documento ->> 'perfil_activo_ref') IS NOT DISTINCT FROM perfil_activo_ref
		AND (documento ->> 'principal_id') IS NOT DISTINCT FROM principal_id
		AND (documento ->> 'version_rol_ref') IS NOT DISTINCT FROM version_rol_ref
		AND asignacion_ref IS NOT DISTINCT FROM 'asignacion:' || asignacion_id || ':v' || version::text
		AND vec_autorizacion.instante_utc_microsegundo_valido(documento ->> 'emitida_en') IS TRUE
		AND vec_autorizacion.instante_utc_microsegundo_valido(documento ->> 'vigente_desde') IS TRUE
		AND vec_autorizacion.instante_utc_microsegundo_valido(documento ->> 'vigente_hasta') IS TRUE
		AND (documento ->> 'emitida_en')::timestamptz IS NOT DISTINCT FROM emitida_en
    ),
    -- Incluso si una version antigua del binario aceptase global=["*"], la
    -- barrera durable lo rechaza. Toda dimension y valor debe ser exacto.
    CONSTRAINT asignacion_sin_comodines_positivos CHECK (
		NOT jsonb_path_exists(documento, '$.ambitos[*].clave ? (@ like_regex ".*\\*.*")')
		AND NOT jsonb_path_exists(documento, '$.ambitos[*].valores[*] ? (@ like_regex ".*\\*.*")')
		AND vec_autorizacion.ambitos_positivos_validos(documento)
    ),
    UNIQUE (asignacion_id, version),
    UNIQUE (perfil_activo_ref, asignacion_ref)
);

CREATE INDEX asignacion_perfil_principal_perfil
    ON vec_autorizacion.asignacion_perfil(principal_id, perfil_activo_ref, version DESC);

CREATE TABLE vec_autorizacion.asignacion_perfil_actual (
    perfil_activo_ref text PRIMARY KEY,
    asignacion_ref text NOT NULL UNIQUE,
    actualizada_en timestamptz(6) NOT NULL,
    actualizada_por text NOT NULL,
    acto_ref text NOT NULL,
    CONSTRAINT asignacion_actual_evidencia_segura CHECK (
		vec_autorizacion.texto_positivo_valido(actualizada_por, 512) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(acto_ref, 512) IS TRUE
    ),
    FOREIGN KEY (perfil_activo_ref, asignacion_ref)
        REFERENCES vec_autorizacion.asignacion_perfil(perfil_activo_ref, asignacion_ref)
);

CREATE TABLE vec_autorizacion.politica_restrictiva (
    politica_ref text PRIMARY KEY,
    politica_id text NOT NULL,
    version bigint NOT NULL CHECK (version > 0),
    huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
    publicada_en timestamptz(6) NOT NULL,
    documento jsonb NOT NULL CHECK (jsonb_typeof(documento) = 'object'),
    creada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT politica_referencias_seguras CHECK (
		vec_autorizacion.texto_positivo_valido(politica_ref, 512) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(politica_id, 128) IS TRUE
    ),
    CONSTRAINT politica_documento_identidad CHECK (
		(documento ->> 'politica_id') IS NOT DISTINCT FROM politica_id
		AND (documento ->> 'version')::bigint IS NOT DISTINCT FROM version
		AND politica_ref IS NOT DISTINCT FROM 'politica:' || politica_id || ':v' || version::text
		AND vec_autorizacion.instante_utc_microsegundo_valido(documento ->> 'publicada_en') IS TRUE
		AND vec_autorizacion.instante_utc_microsegundo_valido(documento ->> 'vigente_desde') IS TRUE
		AND vec_autorizacion.instante_utc_microsegundo_valido(documento ->> 'vigente_hasta') IS TRUE
		AND (documento ->> 'publicada_en')::timestamptz IS NOT DISTINCT FROM publicada_en
    ),
    UNIQUE (politica_id, version),
    UNIQUE (politica_id, politica_ref)
);

CREATE TABLE vec_autorizacion.politica_restrictiva_actual (
    politica_id text PRIMARY KEY,
    politica_ref text NOT NULL UNIQUE,
    actualizada_en timestamptz(6) NOT NULL,
    actualizada_por text NOT NULL,
    acto_ref text NOT NULL,
    CONSTRAINT politica_actual_evidencia_segura CHECK (
		vec_autorizacion.texto_positivo_valido(actualizada_por, 512) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(acto_ref, 512) IS TRUE
    ),
    FOREIGN KEY (politica_id, politica_ref)
        REFERENCES vec_autorizacion.politica_restrictiva(politica_id, politica_ref)
);

CREATE TABLE vec_autorizacion.control_catalogo_politicas (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    revision numeric(20, 0) NOT NULL CHECK (
        revision BETWEEN 1 AND 18446744073709551615
    ),
    huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
    actualizado_en timestamptz(6) NOT NULL,
    actualizado_por text NOT NULL,
    acto_ref text NOT NULL,
    ultima_transaccion bigint NOT NULL DEFAULT txid_current(),
    CONSTRAINT catalogo_evidencia_segura CHECK (
		vec_autorizacion.texto_positivo_valido(actualizado_por, 512) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(acto_ref, 512) IS TRUE
    )
);

INSERT INTO vec_autorizacion.control_catalogo_politicas (
    control_id, revision, huella_sha256, actualizado_en, actualizado_por, acto_ref
) VALUES (
    true,
    1,
    '4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945',
    clock_timestamp(),
    'migracion:000001',
    'acto:migracion:000001:catalogo-vacio'
);

CREATE TABLE vec_autorizacion.decision_autorizacion (
    decision_ref text PRIMARY KEY,
    concedida boolean NOT NULL,
    codigo text NOT NULL,
    principal_id text NOT NULL,
    perfil_activo_ref text NOT NULL,
    accion text NOT NULL,
    recurso_ref text NOT NULL,
    modulo_id text NOT NULL,
    tipo_recurso text NOT NULL,
    contexto_recurso_huella_sha256 text NOT NULL,
    finalidad text NOT NULL,
    correlacion_ref text NOT NULL,
    asignacion_ref text NOT NULL,
    asignacion_huella_sha256 text NOT NULL,
    version_rol_ref text NOT NULL,
    version_rol_huella_sha256 text NOT NULL,
    control_vigencia_version_rol_ref text NOT NULL,
    control_vigencia_version_rol_revision numeric(20, 0) NOT NULL,
    control_vigencia_version_rol_huella_sha256 text NOT NULL,
    revision_catalogo_politicas numeric(20, 0) NOT NULL,
    catalogo_politicas_huella_sha256 text NOT NULL,
    politicas_evaluadas_manifesto jsonb NOT NULL,
    politicas_aplicadas_manifesto jsonb NOT NULL,
    emitida_en timestamptz(6) NOT NULL,
    valida_hasta timestamptz(6) NOT NULL,
    documento jsonb NOT NULL,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT decision_referencias_seguras CHECK (
		vec_autorizacion.texto_positivo_valido(decision_ref, 512) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(codigo, 128) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(principal_id, 512) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(perfil_activo_ref, 512) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(accion, 256) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(recurso_ref, 512) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(modulo_id, 128) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(tipo_recurso, 128) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(finalidad, 512) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(correlacion_ref, 512) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(asignacion_ref, 512) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(version_rol_ref, 512) IS TRUE
		AND vec_autorizacion.texto_positivo_valido(control_vigencia_version_rol_ref, 512) IS TRUE
	),
	CONSTRAINT decision_solo_concesion_ejecutable CHECK (
		concedida IS TRUE AND codigo = 'concedida'
    ),
    CONSTRAINT decision_huellas_canonicas CHECK (
        contexto_recurso_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND asignacion_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND version_rol_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND control_vigencia_version_rol_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND catalogo_politicas_huella_sha256 ~ '^[0-9a-f]{64}$'
    ),
    CONSTRAINT decision_revisiones_rango CHECK (
        control_vigencia_version_rol_revision BETWEEN 1 AND 18446744073709551615
        AND revision_catalogo_politicas BETWEEN 1 AND 18446744073709551615
    ),
    CONSTRAINT decision_vigencia_breve CHECK (
        valida_hasta > emitida_en
        AND valida_hasta <= emitida_en + interval '5 minutes'
        AND registrada_en >= emitida_en
        AND registrada_en < valida_hasta
    ),
	CONSTRAINT decision_documentos_tipo CHECK (
		vec_autorizacion.documento_decision_estructura_valida(documento) IS TRUE
	),
    CONSTRAINT decision_documento_identidad CHECK (
		(documento ->> 'decision_ref') IS NOT DISTINCT FROM decision_ref
		AND (documento ->> 'concedida')::boolean IS NOT DISTINCT FROM concedida
		AND (documento ->> 'codigo') IS NOT DISTINCT FROM codigo
		AND (documento ->> 'principal_id') IS NOT DISTINCT FROM principal_id
		AND (documento ->> 'perfil_activo_ref') IS NOT DISTINCT FROM perfil_activo_ref
		AND (documento ->> 'accion') IS NOT DISTINCT FROM accion
		AND (documento ->> 'recurso_ref') IS NOT DISTINCT FROM recurso_ref
		AND (documento ->> 'modulo_id') IS NOT DISTINCT FROM modulo_id
		AND (documento ->> 'tipo_recurso') IS NOT DISTINCT FROM tipo_recurso
		AND (documento ->> 'contexto_recurso_huella_sha256') IS NOT DISTINCT FROM contexto_recurso_huella_sha256
		AND (documento ->> 'finalidad') IS NOT DISTINCT FROM finalidad
		AND (documento ->> 'correlacion_ref') IS NOT DISTINCT FROM correlacion_ref
		AND (documento ->> 'asignacion_ref') IS NOT DISTINCT FROM asignacion_ref
		AND (documento ->> 'asignacion_huella_sha256') IS NOT DISTINCT FROM asignacion_huella_sha256
		AND (documento ->> 'version_rol_ref') IS NOT DISTINCT FROM version_rol_ref
		AND (documento ->> 'version_rol_huella_sha256') IS NOT DISTINCT FROM version_rol_huella_sha256
		AND (documento ->> 'control_vigencia_version_rol_ref') IS NOT DISTINCT FROM control_vigencia_version_rol_ref
		AND (documento ->> 'control_vigencia_version_rol_revision')::numeric IS NOT DISTINCT FROM control_vigencia_version_rol_revision
		AND (documento ->> 'control_vigencia_version_rol_huella_sha256') IS NOT DISTINCT FROM control_vigencia_version_rol_huella_sha256
		AND (documento ->> 'revision_catalogo_politicas')::numeric IS NOT DISTINCT FROM revision_catalogo_politicas
		AND (documento ->> 'catalogo_politicas_huella_sha256') IS NOT DISTINCT FROM catalogo_politicas_huella_sha256
		AND (documento -> 'politicas_evaluadas_huellas_sha256') IS NOT DISTINCT FROM politicas_evaluadas_manifesto
		AND (documento -> 'politicas_huellas_sha256') IS NOT DISTINCT FROM politicas_aplicadas_manifesto
		AND (documento ->> 'emitida_en')::timestamptz IS NOT DISTINCT FROM emitida_en
		AND (documento ->> 'valida_hasta')::timestamptz IS NOT DISTINCT FROM valida_hasta
    ),
    FOREIGN KEY (asignacion_ref) REFERENCES vec_autorizacion.asignacion_perfil(asignacion_ref),
    FOREIGN KEY (version_rol_ref) REFERENCES vec_autorizacion.version_rol(version_rol_ref),
    FOREIGN KEY (control_vigencia_version_rol_ref, control_vigencia_version_rol_revision)
        REFERENCES vec_autorizacion.control_vigencia_version_rol(version_rol_ref, revision)
);

CREATE INDEX decision_autorizacion_principal_fecha
    ON vec_autorizacion.decision_autorizacion(principal_id, registrada_en DESC);
CREATE INDEX decision_autorizacion_perfil_fecha
    ON vec_autorizacion.decision_autorizacion(perfil_activo_ref, registrada_en DESC);
CREATE INDEX decision_autorizacion_correlacion
    ON vec_autorizacion.decision_autorizacion(correlacion_ref, registrada_en DESC);

CREATE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'objeto inmutable';
END
$funcion$;

CREATE FUNCTION vec_autorizacion.rechazar_eliminacion_versionada()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'eliminacion no permitida';
END
$funcion$;

CREATE TRIGGER version_rol_inmutable
    BEFORE UPDATE OR DELETE ON vec_autorizacion.version_rol
    FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER version_rol_no_truncar
    BEFORE TRUNCATE ON vec_autorizacion.version_rol
    FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER control_rol_inmutable
    BEFORE UPDATE OR DELETE ON vec_autorizacion.control_vigencia_version_rol
    FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER control_rol_no_truncar
    BEFORE TRUNCATE ON vec_autorizacion.control_vigencia_version_rol
    FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER asignacion_inmutable
    BEFORE UPDATE OR DELETE ON vec_autorizacion.asignacion_perfil
    FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER asignacion_no_truncar
    BEFORE TRUNCATE ON vec_autorizacion.asignacion_perfil
    FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER politica_inmutable
    BEFORE UPDATE OR DELETE ON vec_autorizacion.politica_restrictiva
    FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER politica_no_truncar
    BEFORE TRUNCATE ON vec_autorizacion.politica_restrictiva
    FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER decision_inmutable
    BEFORE UPDATE OR DELETE ON vec_autorizacion.decision_autorizacion
    FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER decision_no_truncar
    BEFORE TRUNCATE ON vec_autorizacion.decision_autorizacion
    FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();

CREATE TRIGGER control_rol_actual_no_eliminar
    BEFORE DELETE OR TRUNCATE ON vec_autorizacion.control_vigencia_version_rol_actual
    FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_eliminacion_versionada();
CREATE TRIGGER asignacion_actual_no_eliminar
    BEFORE DELETE OR TRUNCATE ON vec_autorizacion.asignacion_perfil_actual
    FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_eliminacion_versionada();
CREATE TRIGGER politica_actual_no_eliminar
    BEFORE DELETE OR TRUNCATE ON vec_autorizacion.politica_restrictiva_actual
    FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_eliminacion_versionada();
CREATE TRIGGER catalogo_no_eliminar
    BEFORE DELETE OR TRUNCATE ON vec_autorizacion.control_catalogo_politicas
    FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_eliminacion_versionada();

CREATE FUNCTION vec_autorizacion.validar_avance_control_rol()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF NEW.version_rol_ref IS DISTINCT FROM OLD.version_rol_ref
       OR NEW.revision IS DISTINCT FROM OLD.revision + 1 THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'secuencia de control de rol invalida';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER control_rol_actual_avance
    BEFORE UPDATE ON vec_autorizacion.control_vigencia_version_rol_actual
    FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.validar_avance_control_rol();

CREATE FUNCTION vec_autorizacion.validar_avance_asignacion_actual()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    anterior vec_autorizacion.asignacion_perfil%ROWTYPE;
    nueva vec_autorizacion.asignacion_perfil%ROWTYPE;
BEGIN
    SELECT * INTO STRICT anterior FROM vec_autorizacion.asignacion_perfil
     WHERE asignacion_ref = OLD.asignacion_ref;
    SELECT * INTO STRICT nueva FROM vec_autorizacion.asignacion_perfil
     WHERE asignacion_ref = NEW.asignacion_ref;
    IF NEW.perfil_activo_ref IS DISTINCT FROM OLD.perfil_activo_ref
       OR nueva.asignacion_id IS DISTINCT FROM anterior.asignacion_id
       OR nueva.principal_id IS DISTINCT FROM anterior.principal_id
       OR nueva.version <= anterior.version THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'avance de asignacion invalido';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER asignacion_actual_avance
    BEFORE UPDATE ON vec_autorizacion.asignacion_perfil_actual
    FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.validar_avance_asignacion_actual();

CREATE FUNCTION vec_autorizacion.validar_avance_politica_actual()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    version_anterior bigint;
    version_nueva bigint;
BEGIN
    SELECT version INTO STRICT version_anterior FROM vec_autorizacion.politica_restrictiva
     WHERE politica_ref = OLD.politica_ref;
    SELECT version INTO STRICT version_nueva FROM vec_autorizacion.politica_restrictiva
     WHERE politica_ref = NEW.politica_ref;
    IF NEW.politica_id IS DISTINCT FROM OLD.politica_id OR version_nueva <= version_anterior THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'avance de politica invalido';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER politica_actual_avance
    BEFORE UPDATE ON vec_autorizacion.politica_restrictiva_actual
    FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.validar_avance_politica_actual();

CREATE FUNCTION vec_autorizacion.validar_avance_catalogo()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF NEW.control_id IS DISTINCT FROM OLD.control_id OR NEW.revision IS DISTINCT FROM OLD.revision + 1 THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'avance de catalogo invalido';
    END IF;
    NEW.ultima_transaccion := txid_current();
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER catalogo_avance
    BEFORE UPDATE ON vec_autorizacion.control_catalogo_politicas
    FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.validar_avance_catalogo();

CREATE FUNCTION vec_autorizacion.exigir_catalogo_actualizado_en_transaccion()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    transaccion_catalogo bigint;
BEGIN
    SELECT ultima_transaccion INTO STRICT transaccion_catalogo
      FROM vec_autorizacion.control_catalogo_politicas WHERE control_id = true;
    IF transaccion_catalogo IS DISTINCT FROM txid_current() THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'catalogo no actualizado atomicamente';
    END IF;
    RETURN NULL;
END
$funcion$;

CREATE CONSTRAINT TRIGGER politica_actual_exige_catalogo
    AFTER INSERT OR UPDATE OR DELETE ON vec_autorizacion.politica_restrictiva_actual
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.exigir_catalogo_actualizado_en_transaccion();

-- RLS se fuerza incluso al propietario. El unico acceso total es una politica
-- positiva para el propietario NOLOGIN que ejecuta las funciones definidoras.
DO $bloque$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'version_rol',
        'control_vigencia_version_rol',
        'control_vigencia_version_rol_actual',
        'asignacion_perfil',
        'asignacion_perfil_actual',
        'politica_restrictiva',
        'politica_restrictiva_actual',
        'control_catalogo_politicas',
        'decision_autorizacion'
    ] LOOP
        EXECUTE format('ALTER TABLE vec_autorizacion.%I ENABLE ROW LEVEL SECURITY', tabla);
        EXECUTE format('ALTER TABLE vec_autorizacion.%I FORCE ROW LEVEL SECURITY', tabla);
        EXECUTE format(
            'CREATE POLICY acceso_propietario_exacto ON vec_autorizacion.%I FOR ALL TO vec_autorizacion_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
            tabla, 'vec_autorizacion_propietario', 'vec_autorizacion_propietario'
        );
    END LOOP;
END
$bloque$;

CREATE FUNCTION vec_autorizacion.obtener_instantanea(
    p_principal_id text,
    p_perfil_activo_ref text
)
RETURNS TABLE (
    documento_asignacion jsonb,
    documento_rol jsonb,
    documento_control_rol jsonb,
    revision_catalogo text,
    huella_catalogo text,
    documentos_politicas jsonb
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT
        asignacion.documento,
        rol.documento,
        control.documento,
        catalogo.revision::text,
        catalogo.huella_sha256,
        COALESCE((
            SELECT jsonb_agg(politica.documento ORDER BY politica.politica_ref)
              FROM vec_autorizacion.politica_restrictiva_actual AS politica_actual
              JOIN vec_autorizacion.politica_restrictiva AS politica
                ON politica.politica_id = politica_actual.politica_id
               AND politica.politica_ref = politica_actual.politica_ref
        ), '[]'::jsonb)
      FROM vec_autorizacion.asignacion_perfil_actual AS asignacion_actual
      JOIN vec_autorizacion.asignacion_perfil AS asignacion
        ON asignacion.perfil_activo_ref = asignacion_actual.perfil_activo_ref
       AND asignacion.asignacion_ref = asignacion_actual.asignacion_ref
      JOIN vec_autorizacion.version_rol AS rol
        ON rol.version_rol_ref = asignacion.version_rol_ref
      JOIN vec_autorizacion.control_vigencia_version_rol_actual AS control_actual
        ON control_actual.version_rol_ref = rol.version_rol_ref
      JOIN vec_autorizacion.control_vigencia_version_rol AS control
        ON control.version_rol_ref = control_actual.version_rol_ref
       AND control.revision = control_actual.revision
      CROSS JOIN vec_autorizacion.control_catalogo_politicas AS catalogo
     WHERE asignacion_actual.perfil_activo_ref = p_perfil_activo_ref
       AND asignacion.principal_id = p_principal_id
       AND catalogo.control_id = true
$funcion$;

CREATE FUNCTION vec_autorizacion.registrar_decision_si_vigente(p_documento jsonb)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    asignacion_actual record;
    rol_actual record;
    catalogo_actual record;
    manifesto_actual jsonb;
    referencias_actuales jsonb;
    referencias_decision jsonb;
	referencias_aplicadas_manifesto jsonb;
	referencias_aplicadas_decision jsonb;
    instante_registro timestamptz(6);
BEGIN
	IF vec_autorizacion.documento_decision_estructura_valida(p_documento) IS NOT TRUE THEN
		RETURN false;
	END IF;

    SELECT actual.asignacion_ref, asignacion.principal_id, asignacion.version_rol_ref,
           asignacion.huella_sha256
      INTO asignacion_actual
      FROM vec_autorizacion.asignacion_perfil_actual AS actual
      JOIN vec_autorizacion.asignacion_perfil AS asignacion
        ON asignacion.perfil_activo_ref = actual.perfil_activo_ref
       AND asignacion.asignacion_ref = actual.asignacion_ref
     WHERE actual.perfil_activo_ref = p_documento ->> 'perfil_activo_ref'
     FOR UPDATE OF actual;
    IF NOT FOUND
       OR asignacion_actual.asignacion_ref IS DISTINCT FROM p_documento ->> 'asignacion_ref'
       OR asignacion_actual.principal_id IS DISTINCT FROM p_documento ->> 'principal_id'
       OR asignacion_actual.version_rol_ref IS DISTINCT FROM p_documento ->> 'version_rol_ref'
       OR asignacion_actual.huella_sha256 IS DISTINCT FROM p_documento ->> 'asignacion_huella_sha256' THEN
        RETURN false;
    END IF;

	SELECT rol.huella_sha256,
	       control.version_rol_ref, control.revision, control.huella_sha256 AS huella_control
      INTO rol_actual
      FROM vec_autorizacion.version_rol AS rol
      JOIN vec_autorizacion.control_vigencia_version_rol_actual AS actual
        ON actual.version_rol_ref = rol.version_rol_ref
      JOIN vec_autorizacion.control_vigencia_version_rol AS control
        ON control.version_rol_ref = actual.version_rol_ref
       AND control.revision = actual.revision
     WHERE rol.version_rol_ref = asignacion_actual.version_rol_ref
     FOR UPDATE OF actual;
    IF NOT FOUND
       OR rol_actual.huella_sha256 IS DISTINCT FROM p_documento ->> 'version_rol_huella_sha256'
       OR rol_actual.version_rol_ref IS DISTINCT FROM p_documento ->> 'control_vigencia_version_rol_ref'
       OR rol_actual.revision IS DISTINCT FROM (p_documento ->> 'control_vigencia_version_rol_revision')::numeric
       OR rol_actual.huella_control IS DISTINCT FROM p_documento ->> 'control_vigencia_version_rol_huella_sha256' THEN
        RETURN false;
    END IF;

    SELECT revision, huella_sha256
      INTO catalogo_actual
      FROM vec_autorizacion.control_catalogo_politicas
     WHERE control_id = true
     FOR UPDATE;
    IF NOT FOUND
       OR catalogo_actual.revision IS DISTINCT FROM (p_documento ->> 'revision_catalogo_politicas')::numeric
       OR catalogo_actual.huella_sha256 IS DISTINCT FROM p_documento ->> 'catalogo_politicas_huella_sha256' THEN
        RETURN false;
    END IF;

    SELECT
        COALESCE(jsonb_object_agg(politica.politica_ref, politica.huella_sha256 ORDER BY politica.politica_ref), '{}'::jsonb),
        COALESCE(jsonb_agg(politica.politica_ref ORDER BY politica.politica_ref), '[]'::jsonb)
      INTO manifesto_actual, referencias_actuales
      FROM vec_autorizacion.politica_restrictiva_actual AS actual
      JOIN vec_autorizacion.politica_restrictiva AS politica
        ON politica.politica_id = actual.politica_id
       AND politica.politica_ref = actual.politica_ref;
    SELECT COALESCE(jsonb_agg(referencia ORDER BY referencia), '[]'::jsonb)
      INTO referencias_decision
      FROM jsonb_array_elements_text(p_documento -> 'politicas_evaluadas_refs') AS referencia;
	SELECT COALESCE(jsonb_agg(referencia ORDER BY referencia), '[]'::jsonb)
	  INTO referencias_aplicadas_manifesto
	  FROM jsonb_object_keys(p_documento -> 'politicas_huellas_sha256') AS referencia;
	SELECT COALESCE(jsonb_agg(referencia ORDER BY referencia), '[]'::jsonb)
	  INTO referencias_aplicadas_decision
	  FROM jsonb_array_elements_text(p_documento -> 'politicas_refs') AS referencia;
    IF manifesto_actual IS DISTINCT FROM p_documento -> 'politicas_evaluadas_huellas_sha256'
       OR referencias_actuales IS DISTINCT FROM referencias_decision
       OR jsonb_array_length(referencias_actuales) IS DISTINCT FROM jsonb_array_length(p_documento -> 'politicas_evaluadas_refs')
	   OR referencias_aplicadas_manifesto IS DISTINCT FROM referencias_aplicadas_decision
	   OR jsonb_array_length(referencias_aplicadas_manifesto) IS DISTINCT FROM jsonb_array_length(p_documento -> 'politicas_refs')
       OR EXISTS (
           SELECT 1
             FROM jsonb_each(p_documento -> 'politicas_huellas_sha256') AS aplicada
            WHERE manifesto_actual -> aplicada.key IS DISTINCT FROM aplicada.value
       ) THEN
        RETURN false;
    END IF;

    instante_registro := clock_timestamp();
    IF instante_registro < (p_documento ->> 'emitida_en')::timestamptz
       OR instante_registro >= (p_documento ->> 'valida_hasta')::timestamptz THEN
        RETURN false;
    END IF;

    INSERT INTO vec_autorizacion.decision_autorizacion (
        decision_ref, concedida, codigo, principal_id, perfil_activo_ref,
        accion, recurso_ref, modulo_id, tipo_recurso, contexto_recurso_huella_sha256,
        finalidad, correlacion_ref, asignacion_ref, asignacion_huella_sha256,
        version_rol_ref, version_rol_huella_sha256,
        control_vigencia_version_rol_ref, control_vigencia_version_rol_revision,
        control_vigencia_version_rol_huella_sha256, revision_catalogo_politicas,
        catalogo_politicas_huella_sha256, politicas_evaluadas_manifesto,
        politicas_aplicadas_manifesto, emitida_en, valida_hasta, documento, registrada_en
    ) VALUES (
        p_documento ->> 'decision_ref',
        (p_documento ->> 'concedida')::boolean,
        p_documento ->> 'codigo',
        p_documento ->> 'principal_id',
        p_documento ->> 'perfil_activo_ref',
        p_documento ->> 'accion',
        p_documento ->> 'recurso_ref',
        p_documento ->> 'modulo_id',
        p_documento ->> 'tipo_recurso',
        p_documento ->> 'contexto_recurso_huella_sha256',
        p_documento ->> 'finalidad',
        p_documento ->> 'correlacion_ref',
        p_documento ->> 'asignacion_ref',
        p_documento ->> 'asignacion_huella_sha256',
        p_documento ->> 'version_rol_ref',
        p_documento ->> 'version_rol_huella_sha256',
        p_documento ->> 'control_vigencia_version_rol_ref',
        (p_documento ->> 'control_vigencia_version_rol_revision')::numeric,
        p_documento ->> 'control_vigencia_version_rol_huella_sha256',
        (p_documento ->> 'revision_catalogo_politicas')::numeric,
        p_documento ->> 'catalogo_politicas_huella_sha256',
        p_documento -> 'politicas_evaluadas_huellas_sha256',
        p_documento -> 'politicas_huellas_sha256',
        (p_documento ->> 'emitida_en')::timestamptz,
        (p_documento ->> 'valida_hasta')::timestamptz,
        p_documento,
        instante_registro
    );
    RETURN true;
END
$funcion$;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_autorizacion FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA vec_autorizacion FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_autorizacion FROM PUBLIC;
-- Los tipos compuestos implícitos de las tablas nacen con USAGE para PUBLIC:
-- ALTER DEFAULT PRIVILEGES ON TYPES no los cierra en PostgreSQL 18.
REVOKE ALL ON TYPE vec_autorizacion.version_rol FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.control_vigencia_version_rol FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.control_vigencia_version_rol_actual FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.asignacion_perfil FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.asignacion_perfil_actual FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.politica_restrictiva FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.politica_restrictiva_actual FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.control_catalogo_politicas FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.decision_autorizacion FROM PUBLIC;

GRANT USAGE ON SCHEMA vec_autorizacion TO vec_autorizacion_fuente;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_autorizacion_registro;
GRANT EXECUTE ON FUNCTION vec_autorizacion.obtener_instantanea(text, text)
    TO vec_autorizacion_fuente;
GRANT EXECUTE ON FUNCTION vec_autorizacion.registrar_decision_si_vigente(jsonb)
    TO vec_autorizacion_registro;

COMMIT;
