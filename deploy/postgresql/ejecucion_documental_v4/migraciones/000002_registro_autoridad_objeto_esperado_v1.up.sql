BEGIN;
SET LOCAL ROLE vec_ejecucion_documental_v4_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $precondicion$
BEGIN
    IF to_regclass(
           'vec_ejecucion_documental_v4.orden_generacion_documental'
       ) IS NULL
       OR to_regclass(
           'vec_ejecucion_documental_v4.control_cadena_auditoria'
       ) IS NULL
       OR to_regprocedure(
           'vec_ejecucion_documental_v4.rechazar_mutacion_inmutable()'
       ) IS NULL
       OR to_regprocedure('pg_catalog.sha256(bytea)') IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'falta la autoridad PostgreSQL V4 requerida';
    END IF;
END
$precondicion$;

-- Replica la lista positiva minima de referencias V2. No admite rutas,
-- localizadores, comodines ni formas evidentes de identificador personal.
CREATE FUNCTION
vec_ejecucion_documental_v4.referencia_opaca_autoridad_objeto_v1(
    p_valor text,
    p_maximo integer
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor IS NOT NULL AND p_maximo > 0
       AND octet_length(p_valor) BETWEEN 1 AND p_maximo
       AND p_valor ~ '^[!-~]+$'
       AND strpos(p_valor, '/') = 0
       AND strpos(p_valor, chr(92)) = 0
       AND strpos(p_valor, '?') = 0
       AND strpos(p_valor, '@') = 0
       AND strpos(p_valor, '#') = 0
       AND strpos(p_valor, '*') = 0
       AND strpos(p_valor, '..') = 0
       AND strpos(p_valor, '://') = 0
       AND lower(p_valor) !~
           '(arn:|etag:|kms:|bucket:|bucket_|endpoint:|ruta:|path:|file:|s3:|http:|https:|dni:|nif:|nie:|nombre:|apellido:|correo:|email:|telefono:|direccion:)'
       AND upper(p_valor) !~ '^([0-9]{8}|[XYZ][0-9]{7})[A-Z]$'
$funcion$;

CREATE FUNCTION
vec_ejecucion_documental_v4.encuadrar_autoridad_objeto_v1(p_valor bytea)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT int8send(octet_length(p_valor)::bigint) || p_valor
$funcion$;

-- Comprueba la estructura TLV V2 completa y obtiene solo las seis ligaduras
-- que este agregado necesita. La atestacion ya fue verificada en vivo por
-- PrepararRegistro; SQL no intenta reconstruir esa autoridad.
CREATE FUNCTION
vec_ejecucion_documental_v4.recibo_material_v2_coteja_autoridad_objeto_v1(
    p_recibo bytea,
    p_recibo_ref text,
    p_conector_id text,
    p_efecto_ref text,
    p_objeto_ref text,
    p_objeto_version text
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    posicion integer := 0;
    total integer;
    etiqueta integer;
    esperada integer := 0;
    longitud bigint;
    indice integer;
    valor bytea;
    texto text;
    tiene_retencion boolean := false;
BEGIN
    total := octet_length(p_recibo);
    IF total NOT BETWEEN 1 AND 65536 THEN
        RETURN false;
    END IF;
    WHILE posicion < total LOOP
        IF total - posicion < 10 THEN
            RETURN false;
        END IF;
        etiqueta := get_byte(p_recibo, posicion) * 256
            + get_byte(p_recibo, posicion + 1);
        longitud := 0;
        FOR indice IN 2..9 LOOP
            longitud := longitud * 256 + get_byte(p_recibo, posicion + indice);
            IF longitud > 65536 THEN
                RETURN false;
            END IF;
        END LOOP;
        posicion := posicion + 10;
        IF longitud > total - posicion THEN
            RETURN false;
        END IF;
        IF esperada = 28 AND etiqueta = 29 AND NOT tiene_retencion THEN
            esperada := 29;
        END IF;
        IF etiqueta <> esperada THEN
            RETURN false;
        END IF;
        valor := substring(p_recibo FROM posicion + 1 FOR longitud::integer);
        IF etiqueta = 1 AND (longitud <> 2 OR valor <> decode('0002', 'hex'))
           OR etiqueta IN (5, 15) AND longitud <> 4
           OR etiqueta IN (6, 16, 17, 24) AND longitud <> 32
           OR etiqueta IN (23, 26, 28) AND longitud <> 8
           OR etiqueta = 27 AND (
               longitud <> 1 OR get_byte(valor, 0) NOT IN (0, 1)
           ) THEN
            RETURN false;
        END IF;
        IF etiqueta IN (
            0, 2, 3, 4, 7, 8, 9, 10, 11, 12, 13, 14, 18, 19, 20,
            21, 22, 25, 29, 30
        ) THEN
            IF longitud = 0 THEN
                RETURN false;
            END IF;
            texto := convert_from(valor, 'UTF8');
            IF texto !~ '^[!-~]+$' THEN
                RETURN false;
            ELSIF etiqueta = 0 AND texto <>
                   'vec.almacen.recibo-escritura-material.v2'
               OR etiqueta = 2 AND texto <> p_recibo_ref
               OR etiqueta = 3 AND texto <> p_conector_id
               OR etiqueta = 13 AND texto <> p_efecto_ref
               OR etiqueta = 19 AND texto <> p_objeto_ref
               OR etiqueta = 20 AND texto <> p_objeto_version THEN
                RETURN false;
            END IF;
        ELSIF etiqueta = 27 THEN
            tiene_retencion := get_byte(valor, 0) = 1;
        ELSIF etiqueta = 28 AND NOT tiene_retencion THEN
            RETURN false;
        END IF;
        posicion := posicion + longitud::integer;
        esperada := etiqueta + 1;
    END LOOP;
    RETURN posicion = total AND esperada = 31;
EXCEPTION
    WHEN data_exception THEN
        RETURN false;
END
$funcion$;

CREATE TABLE
vec_ejecucion_documental_v4.registro_autoridad_objeto_esperado_v1 (
    registro_ref text PRIMARY KEY,
    esquema text NOT NULL,
    version_contrato numeric(5, 0) NOT NULL,
    orden_ref text NOT NULL REFERENCES
        vec_ejecucion_documental_v4.orden_generacion_documental(orden_ref),
    decision_ref text NOT NULL REFERENCES
        vec_ejecucion_documental_v4.consumo_decision_atomico(decision_ref),
    efecto_ref text NOT NULL REFERENCES
        vec_ejecucion_documental_v4.consumo_decision_atomico(efecto_ref),
    paso_ref text NOT NULL,
    version_estado numeric(20, 0) NOT NULL,
    estado text NOT NULL,
    recibo_material_ref text NOT NULL UNIQUE,
    huella_recibo_material_sha256 text NOT NULL,
    huella_declaracion_v4_sha256 text NOT NULL,
    objeto_ref text NOT NULL,
    objeto_version text NOT NULL,
    conector_id text NOT NULL,
    huella_plan_efecto_sha256 text NOT NULL,
    huella_manifiesto_sha256 text NOT NULL,
    huella_paso_sha256 text NOT NULL,
    recibo_material_canonico bytea NOT NULL,
    atestacion_algoritmo text NOT NULL,
    atestacion_clave_ref text NOT NULL,
    atestacion_clave_version numeric(10, 0) NOT NULL,
    atestacion_dominio text NOT NULL,
    atestacion_codigo_longitud integer NOT NULL,
    huella_atestacion_codigo_sha256 text NOT NULL,
    huella_sello_atestacion_sha256 text NOT NULL,
    huella_plan_autorizacion_sha256 text NOT NULL,
    huella_efecto_v4_sha256 text NOT NULL,
    huella_proyeccion_sha256 text NOT NULL UNIQUE,
    correlacion_ref text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    UNIQUE (efecto_ref, paso_ref),
    CHECK (esquema = 'vec.documentos.autoridad-objeto-esperado.v1'
        AND version_contrato = 1),
    CHECK (version_estado = 1 AND estado = 'autoridad_objeto_registrada'),
    CHECK (paso_ref IN (
        '01_preparar_carga_directa', '02_abandonar_carga_directa',
        '01_confirmar_carga_directa', '01_leer_para_analisis',
        '02_analizar_contenido', '01_promover', '01_custodiar_decision',
        '01_custodiar_documento_firmado',
        '01_retener_documento_firmado'
    )),
    CHECK (
        vec_ejecucion_documental_v4.referencia_opaca_autoridad_objeto_v1(
            registro_ref, 512
        ) AND
        vec_ejecucion_documental_v4.referencia_opaca_autoridad_objeto_v1(
            recibo_material_ref, 512
        ) AND
        vec_ejecucion_documental_v4.referencia_opaca_autoridad_objeto_v1(
            objeto_ref, 512
        ) AND
        vec_ejecucion_documental_v4.referencia_opaca_autoridad_objeto_v1(
            objeto_version, 256
        ) AND
        vec_ejecucion_documental_v4.referencia_opaca_autoridad_objeto_v1(
            conector_id, 128
        ) AND
        vec_ejecucion_documental_v4.referencia_opaca_autoridad_objeto_v1(
            paso_ref, 512
        ) AND
        vec_ejecucion_documental_v4.referencia_opaca_autoridad_objeto_v1(
            efecto_ref, 512
        ) AND
        vec_ejecucion_documental_v4.referencia_opaca_autoridad_objeto_v1(
            decision_ref, 512
        ) AND
        vec_ejecucion_documental_v4.referencia_opaca_autoridad_objeto_v1(
            correlacion_ref, 512
        ) AND
        vec_ejecucion_documental_v4.referencia_opaca_autoridad_objeto_v1(
            atestacion_clave_ref, 512
        )
    ),
    CHECK (orden_ref = efecto_ref),
    CHECK (octet_length(recibo_material_canonico) BETWEEN 1 AND 65536),
    CHECK (atestacion_algoritmo IN ('hmac-sha-256', 'cose-sign1')),
    CHECK (atestacion_clave_version BETWEEN 1 AND 4294967295),
    CHECK (
        atestacion_dominio = 'recibo-escritura-objeto-material-v2'
        AND (
            atestacion_algoritmo = 'hmac-sha-256'
            AND atestacion_codigo_longitud = 32
            OR atestacion_algoritmo = 'cose-sign1'
            AND atestacion_codigo_longitud BETWEEN 16 AND 16384
        )
    ),
    CHECK (
        huella_recibo_material_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_declaracion_v4_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_plan_efecto_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_manifiesto_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_paso_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_atestacion_codigo_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_sello_atestacion_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_plan_autorizacion_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_efecto_v4_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_proyeccion_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_recibo_material_sha256 <> repeat('0', 64)
        AND huella_declaracion_v4_sha256 <> repeat('0', 64)
        AND huella_plan_efecto_sha256 <> repeat('0', 64)
        AND huella_manifiesto_sha256 <> repeat('0', 64)
        AND huella_paso_sha256 <> repeat('0', 64)
        AND huella_atestacion_codigo_sha256 <> repeat('0', 64)
        AND huella_sello_atestacion_sha256 <> repeat('0', 64)
        AND huella_plan_autorizacion_sha256 <> repeat('0', 64)
        AND huella_efecto_v4_sha256 <> repeat('0', 64)
        AND huella_proyeccion_sha256 <> repeat('0', 64)
    )
);

CREATE TABLE vec_ejecucion_documental_v4.auditoria_autoridad_objeto_v1 (
    auditoria_ref text PRIMARY KEY,
    secuencia numeric(20, 0) NOT NULL UNIQUE,
    registro_ref text NOT NULL UNIQUE REFERENCES
        vec_ejecucion_documental_v4.registro_autoridad_objeto_esperado_v1(
            registro_ref
        ),
    accion text NOT NULL,
    resultado text NOT NULL,
    ocurrida_en timestamptz(6) NOT NULL,
    huella_anterior_sha256 text NOT NULL,
    huella_registro_sha256 text NOT NULL UNIQUE,
    CHECK (accion = 'registrar_autoridad_objeto_esperado_v1'
        AND resultado = 'autoridad_objeto_registrada'),
    CHECK (huella_anterior_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_registro_sha256 ~ '^[0-9a-f]{64}$')
);

CREATE TABLE vec_ejecucion_documental_v4.outbox_autoridad_objeto_v1 (
    evento_ref text PRIMARY KEY,
    secuencia numeric(20, 0) NOT NULL UNIQUE,
    tipo text NOT NULL,
    estado text NOT NULL,
    registro_ref text NOT NULL UNIQUE REFERENCES
        vec_ejecucion_documental_v4.registro_autoridad_objeto_esperado_v1(
            registro_ref
        ),
    auditoria_ref text NOT NULL UNIQUE REFERENCES
        vec_ejecucion_documental_v4.auditoria_autoridad_objeto_v1(
            auditoria_ref
        ),
    huella_auditoria_sha256 text NOT NULL,
    huella_proyeccion_sha256 text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    huella_registro_sha256 text NOT NULL UNIQUE,
    CHECK (tipo = 'autoridad_objeto_esperado_registrada' AND estado = 'pendiente'),
    CHECK (huella_auditoria_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_proyeccion_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_registro_sha256 ~ '^[0-9a-f]{64}$')
);

-- SECURITY DEFINER es imprescindible para que la futura identidad dedicada
-- reciba solo USAGE/EXECUTE y no DML sobre estado, auditoria u outbox.
-- Este corte no concede todavia esa ejecucion a ningun runtime.
CREATE FUNCTION
vec_ejecucion_documental_v4.registrar_autoridad_objeto_esperado_v1(
    p_version_esperada numeric,
    p_proyeccion_canonica bytea
)
RETURNS TABLE (
    resultado text,
    registro_ref text,
    estado text,
    version_estado numeric,
    auditoria_ref text,
    evento_outbox_ref text,
    huella_proyeccion_sha256 text,
    registrada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    texto_original text;
    documento_json json;
    documento jsonb;
    recibo bytea;
    codigo bytea;
    sello bytea;
    version_clave numeric;
    existente record;
    autoridad_v4 record;
    cantidad integer;
    instante timestamptz(6);
    secuencia numeric(20, 0);
    huella_anterior text;
    huella_auditoria text;
    huella_evento text;
    registro_calculado text;
    auditoria_calculada text;
    evento_calculado text;
    huella_entrada text;
BEGIN
    IF p_version_esperada IS DISTINCT FROM 0
       OR p_proyeccion_canonica IS NULL
       OR octet_length(p_proyeccion_canonica) NOT BETWEEN 2 AND 196608 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'proyeccion de autoridad documental no valida';
    END IF;
    huella_entrada := encode(sha256(p_proyeccion_canonica), 'hex');
    texto_original := convert_from(p_proyeccion_canonica, 'UTF8');
    IF texto_original IS DISTINCT FROM btrim(texto_original) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'texto canonico de autoridad documental no valido';
    END IF;
    documento_json := texto_original::json;
    IF json_typeof(documento_json) <> 'object'
       OR (SELECT count(*) FROM json_object_keys(documento_json)) <> 19
       OR (SELECT count(DISTINCT clave)
             FROM json_object_keys(documento_json) AS clave) <> 19 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'estructura de autoridad documental no valida';
    END IF;
    documento := documento_json::jsonb;
    IF NOT (documento ?& ARRAY[
           'esquema', 'version', 'recibo_material_ref',
           'huella_recibo_material_sha256', 'huella_declaracion_v4_sha256',
           'objeto_ref', 'objeto_version', 'conector_id', 'efecto_ref',
           'huella_plan_efecto_sha256', 'huella_manifiesto_sha256',
           'paso_ref', 'huella_paso_sha256',
           'recibo_material_canonico_hex', 'atestacion_algoritmo',
           'atestacion_clave_ref', 'atestacion_clave_version',
           'atestacion_dominio', 'atestacion_codigo_hex'
       ]) OR jsonb_typeof(documento -> 'version') <> 'number'
       OR jsonb_typeof(documento -> 'atestacion_clave_version') <> 'number'
       OR EXISTS (
           SELECT 1
             FROM jsonb_object_keys(documento) AS clave
            WHERE clave NOT IN ('version', 'atestacion_clave_version')
              AND jsonb_typeof(documento -> clave) <> 'string'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'tipos de autoridad documental no validos';
    END IF;
    IF documento ->> 'esquema' <>
           'vec.documentos.autoridad-objeto-esperado.v1'
       OR documento ->> 'version' <> '1'
       OR documento ->> 'atestacion_dominio' <>
           'recibo-escritura-objeto-material-v2'
       OR documento ->> 'atestacion_algoritmo' NOT IN (
           'hmac-sha-256', 'cose-sign1'
       ) OR documento ->> 'paso_ref' NOT IN (
           '01_preparar_carga_directa', '02_abandonar_carga_directa',
           '01_confirmar_carga_directa', '01_leer_para_analisis',
           '02_analizar_contenido', '01_promover',
           '01_custodiar_decision', '01_custodiar_documento_firmado',
           '01_retener_documento_firmado'
       ) OR (documento ->> 'atestacion_clave_version')::numeric <> trunc(
           (documento ->> 'atestacion_clave_version')::numeric
       ) OR (documento ->> 'atestacion_clave_version')::numeric NOT BETWEEN
           1 AND 4294967295 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'perfil de autoridad documental no valido';
    END IF;
    IF EXISTS (
        SELECT 1 FROM unnest(ARRAY[
            'huella_recibo_material_sha256', 'huella_declaracion_v4_sha256',
            'huella_plan_efecto_sha256', 'huella_manifiesto_sha256',
            'huella_paso_sha256'
        ]) AS clave
        WHERE (documento ->> clave) !~ '^[0-9a-f]{64}$'
           OR documento ->> clave = repeat('0', 64)
    ) OR (documento ->> 'recibo_material_canonico_hex') !~
          '^[0-9a-f]+$'
       OR mod(length(documento ->> 'recibo_material_canonico_hex'), 2) <> 0
       OR length(documento ->> 'recibo_material_canonico_hex') > 131072
       OR (documento ->> 'atestacion_codigo_hex') !~ '^[0-9a-f]+$'
       OR mod(length(documento ->> 'atestacion_codigo_hex'), 2) <> 0
       OR length(documento ->> 'atestacion_codigo_hex') > 32768 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'material hexadecimal de autoridad no valido';
    END IF;
    recibo := decode(documento ->> 'recibo_material_canonico_hex', 'hex');
    codigo := decode(documento ->> 'atestacion_codigo_hex', 'hex');
    version_clave := (documento ->> 'atestacion_clave_version')::numeric;
    IF octet_length(recibo) NOT BETWEEN 1 AND 65536
       OR encode(sha256(recibo), 'hex') <>
          documento ->> 'huella_recibo_material_sha256'
       OR documento ->> 'atestacion_algoritmo' = 'hmac-sha-256'
          AND octet_length(codigo) <> 32
       OR documento ->> 'atestacion_algoritmo' = 'cose-sign1'
          AND octet_length(codigo) NOT BETWEEN 16 AND 16384
       OR vec_ejecucion_documental_v4.referencia_opaca_autoridad_objeto_v1(
              documento ->> 'recibo_material_ref', 512
          ) IS NOT TRUE
       OR vec_ejecucion_documental_v4.referencia_opaca_autoridad_objeto_v1(
              documento ->> 'objeto_ref', 512
          ) IS NOT TRUE
       OR vec_ejecucion_documental_v4.referencia_opaca_autoridad_objeto_v1(
              documento ->> 'objeto_version', 256
          ) IS NOT TRUE
       OR vec_ejecucion_documental_v4.referencia_opaca_autoridad_objeto_v1(
              documento ->> 'conector_id', 128
          ) IS NOT TRUE
       OR vec_ejecucion_documental_v4.referencia_opaca_autoridad_objeto_v1(
              documento ->> 'efecto_ref', 512
          ) IS NOT TRUE
       OR vec_ejecucion_documental_v4.referencia_opaca_autoridad_objeto_v1(
              documento ->> 'paso_ref', 512
          ) IS NOT TRUE
       OR vec_ejecucion_documental_v4.referencia_opaca_autoridad_objeto_v1(
              documento ->> 'atestacion_clave_ref', 512
          ) IS NOT TRUE
       OR vec_ejecucion_documental_v4.recibo_material_v2_coteja_autoridad_objeto_v1(
              recibo, documento ->> 'recibo_material_ref',
              documento ->> 'conector_id', documento ->> 'efecto_ref',
              documento ->> 'objeto_ref', documento ->> 'objeto_version'
          ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'recibo material no coincide con la autoridad preparada';
    END IF;
    sello :=
        vec_ejecucion_documental_v4.encuadrar_autoridad_objeto_v1(
            convert_to(
                'vec.documentos.sello-atestacion-autoridad-objeto-esperado.v1',
                'UTF8'
            )
        ) ||
        vec_ejecucion_documental_v4.encuadrar_autoridad_objeto_v1(recibo) ||
        vec_ejecucion_documental_v4.encuadrar_autoridad_objeto_v1(
            convert_to(documento ->> 'atestacion_algoritmo', 'UTF8')
        ) ||
        vec_ejecucion_documental_v4.encuadrar_autoridad_objeto_v1(
            convert_to(documento ->> 'atestacion_clave_ref', 'UTF8')
        ) ||
        vec_ejecucion_documental_v4.encuadrar_autoridad_objeto_v1(
            decode(lpad(to_hex(version_clave::bigint), 8, '0'), 'hex')
        ) ||
        vec_ejecucion_documental_v4.encuadrar_autoridad_objeto_v1(
            convert_to(documento ->> 'atestacion_dominio', 'UTF8')
        ) ||
        vec_ejecucion_documental_v4.encuadrar_autoridad_objeto_v1(codigo);

    PERFORM pg_advisory_xact_lock(clave)
      FROM (
          SELECT DISTINCT hashtextextended(valor, 0) AS clave
            FROM unnest(ARRAY[
                'efecto-paso:' || documento ->> 'efecto_ref' || ':' ||
                    documento ->> 'paso_ref',
                'recibo:' || documento ->> 'recibo_material_ref'
            ]) AS valor
      ) AS bloqueos
     ORDER BY clave;

    SELECT count(*), min(r.registro_ref)
      INTO cantidad, registro_calculado
      FROM vec_ejecucion_documental_v4.registro_autoridad_objeto_esperado_v1 AS r
     WHERE (r.efecto_ref = documento ->> 'efecto_ref'
       AND r.paso_ref = documento ->> 'paso_ref')
        OR r.recibo_material_ref = documento ->> 'recibo_material_ref';
    IF cantidad > 1 THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'autoridad documental preexistente ambigua';
    ELSIF cantidad = 1 THEN
        SELECT r.* INTO STRICT existente
          FROM vec_ejecucion_documental_v4.registro_autoridad_objeto_esperado_v1 AS r
         WHERE r.registro_ref = registro_calculado
         FOR SHARE;
        IF existente.huella_proyeccion_sha256 <> huella_entrada
           OR existente.version_estado <> 1
           OR existente.estado <> 'autoridad_objeto_registrada' THEN
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'replay de autoridad documental incompatible';
        END IF;
        SELECT a.auditoria_ref, o.evento_ref
          INTO STRICT auditoria_calculada, evento_calculado
          FROM vec_ejecucion_documental_v4.auditoria_autoridad_objeto_v1 AS a
          JOIN vec_ejecucion_documental_v4.outbox_autoridad_objeto_v1 AS o
            ON o.auditoria_ref = a.auditoria_ref
         WHERE a.registro_ref = existente.registro_ref;
        resultado := 'repetida';
        registro_ref := existente.registro_ref;
        estado := existente.estado;
        version_estado := existente.version_estado;
        auditoria_ref := auditoria_calculada;
        evento_outbox_ref := evento_calculado;
        huella_proyeccion_sha256 := existente.huella_proyeccion_sha256;
        registrada_en := existente.registrada_en;
        RETURN NEXT;
        RETURN;
    END IF;

    SELECT orden.orden_ref, orden.decision_ref, orden.efecto_ref,
           orden.estado, orden.huella_plan_sha256,
           orden.huella_decision_sha256, orden.huella_aplicacion_sha256,
           orden.correlacion_ref, atestacion.aplicacion_registro,
           consumo.huella_aplicacion_sha256 AS aplicacion_consumida,
           capacidad.capacidad, capacidad.huella_capacidad_sha256
      INTO STRICT autoridad_v4
      FROM vec_ejecucion_documental_v4.orden_generacion_documental AS orden
      JOIN vec_ejecucion_documental_v4.atestacion_pdp AS atestacion
        ON atestacion.decision_ref = orden.decision_ref
       AND atestacion.efecto_ref = orden.efecto_ref
       AND atestacion.huella_plan_sha256 = orden.huella_plan_sha256
       AND atestacion.huella_decision_sha256 = orden.huella_decision_sha256
      JOIN vec_ejecucion_documental_v4.consumo_decision_atomico AS consumo
        ON consumo.decision_ref = orden.decision_ref
       AND consumo.efecto_ref = orden.efecto_ref
       AND consumo.orden_ref = orden.orden_ref
       AND consumo.huella_aplicacion_sha256 = orden.huella_aplicacion_sha256
      JOIN vec_ejecucion_documental_v4.consumo_capacidad AS capacidad
        ON capacidad.decision_ref = orden.decision_ref
     WHERE orden.efecto_ref = documento ->> 'efecto_ref'
     FOR SHARE OF orden, atestacion, consumo, capacidad;
    IF autoridad_v4.estado <> 'pendiente_generacion'
       OR autoridad_v4.orden_ref <> autoridad_v4.efecto_ref
       OR autoridad_v4.aplicacion_registro ->> 'decision_ref' <>
          autoridad_v4.decision_ref
       OR autoridad_v4.aplicacion_registro ->> 'efecto_ref' <>
          autoridad_v4.efecto_ref
       OR autoridad_v4.aplicacion_registro ->> 'huella_plan_sha256' <>
          autoridad_v4.huella_plan_sha256
       OR autoridad_v4.aplicacion_registro ->>
          'huella_solicitud_aplicacion_sha256' <>
          autoridad_v4.aplicacion_consumida
       OR encode(
              sha256(convert_to(autoridad_v4.capacidad::text, 'UTF8')), 'hex'
          ) <> autoridad_v4.huella_capacidad_sha256
       OR autoridad_v4.capacidad ->> 'huella_decision_sha256' <>
          autoridad_v4.huella_decision_sha256
       OR autoridad_v4.capacidad ->> 'huella_efecto_sha256' !~
          '^[0-9a-f]{64}$'
       OR autoridad_v4.capacidad ->> 'huella_efecto_sha256' = repeat('0', 64)
       OR autoridad_v4.huella_plan_sha256 = repeat('0', 64) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'compromisos P0-1 de la orden V4 no validos';
    END IF;

    instante := clock_timestamp();
    registro_calculado := 'registro:autoridad-objeto:v1:' || encode(sha256(
        convert_to(documento ->> 'efecto_ref', 'UTF8') || decode('00', 'hex') ||
        convert_to(documento ->> 'paso_ref', 'UTF8') || decode('00', 'hex') ||
        convert_to(huella_entrada, 'UTF8')
    ), 'hex');
    auditoria_calculada := 'auditoria:autoridad-objeto:v1:' || encode(sha256(
        convert_to(registro_calculado, 'UTF8') || decode('00', 'hex') ||
        convert_to('registrar', 'UTF8')
    ), 'hex');
    evento_calculado := 'evento:autoridad-objeto:v1:' || encode(sha256(
        convert_to(registro_calculado, 'UTF8') || decode('00', 'hex') ||
        convert_to('outbox', 'UTF8')
    ), 'hex');

    INSERT INTO
    vec_ejecucion_documental_v4.registro_autoridad_objeto_esperado_v1 (
        registro_ref, esquema, version_contrato,
        orden_ref, decision_ref, efecto_ref, paso_ref,
        version_estado, estado, recibo_material_ref,
        huella_recibo_material_sha256, huella_declaracion_v4_sha256,
        objeto_ref, objeto_version, conector_id,
        huella_plan_efecto_sha256, huella_manifiesto_sha256,
        huella_paso_sha256, recibo_material_canonico,
        atestacion_algoritmo, atestacion_clave_ref,
        atestacion_clave_version, atestacion_dominio,
        atestacion_codigo_longitud, huella_atestacion_codigo_sha256,
        huella_sello_atestacion_sha256,
        huella_plan_autorizacion_sha256, huella_efecto_v4_sha256,
        huella_proyeccion_sha256, correlacion_ref, registrada_en
    ) VALUES (
        registro_calculado, documento ->> 'esquema',
        (documento ->> 'version')::numeric, autoridad_v4.orden_ref,
        autoridad_v4.decision_ref, autoridad_v4.efecto_ref,
        documento ->> 'paso_ref', 1, 'autoridad_objeto_registrada',
        documento ->> 'recibo_material_ref',
        documento ->> 'huella_recibo_material_sha256',
        documento ->> 'huella_declaracion_v4_sha256',
        documento ->> 'objeto_ref', documento ->> 'objeto_version',
        documento ->> 'conector_id',
        documento ->> 'huella_plan_efecto_sha256',
        documento ->> 'huella_manifiesto_sha256',
        documento ->> 'huella_paso_sha256', recibo,
        documento ->> 'atestacion_algoritmo',
        documento ->> 'atestacion_clave_ref', version_clave,
        documento ->> 'atestacion_dominio', octet_length(codigo),
        encode(sha256(codigo), 'hex'), encode(sha256(sello), 'hex'),
        autoridad_v4.huella_plan_sha256,
        autoridad_v4.capacidad ->> 'huella_efecto_sha256',
        huella_entrada, autoridad_v4.correlacion_ref, instante
    );

    SELECT ultima_secuencia + 1, ultima_huella_sha256
      INTO STRICT secuencia, huella_anterior
      FROM vec_ejecucion_documental_v4.control_cadena_auditoria
     WHERE control_id = true
     FOR UPDATE;
    IF secuencia > 18446744073709551615 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'cadena de auditoria agotada';
    END IF;
    huella_auditoria := encode(sha256(convert_to(concat_ws(E'\n',
        auditoria_calculada, secuencia::text, registro_calculado,
        'registrar_autoridad_objeto_esperado_v1',
        'autoridad_objeto_registrada', huella_entrada,
        autoridad_v4.orden_ref, autoridad_v4.decision_ref,
        autoridad_v4.efecto_ref, autoridad_v4.huella_plan_sha256,
        autoridad_v4.capacidad ->> 'huella_efecto_sha256',
        autoridad_v4.correlacion_ref,
        to_char(instante AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        huella_anterior
    ), 'UTF8')), 'hex');
    huella_evento := encode(sha256(convert_to(concat_ws(E'\n',
        evento_calculado, secuencia::text,
        'autoridad_objeto_esperado_registrada', 'pendiente',
        registro_calculado, auditoria_calculada, huella_auditoria,
        huella_entrada,
        to_char(instante AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    ), 'UTF8')), 'hex');
    INSERT INTO vec_ejecucion_documental_v4.auditoria_autoridad_objeto_v1 (
        auditoria_ref, secuencia, registro_ref, accion, resultado,
        ocurrida_en, huella_anterior_sha256, huella_registro_sha256
    ) VALUES (
        auditoria_calculada, secuencia, registro_calculado,
        'registrar_autoridad_objeto_esperado_v1',
        'autoridad_objeto_registrada', instante, huella_anterior,
        huella_auditoria
    );
    INSERT INTO vec_ejecucion_documental_v4.outbox_autoridad_objeto_v1 (
        evento_ref, secuencia, tipo, estado, registro_ref, auditoria_ref,
        huella_auditoria_sha256, huella_proyeccion_sha256,
        registrada_en, huella_registro_sha256
    ) VALUES (
        evento_calculado, secuencia, 'autoridad_objeto_esperado_registrada',
        'pendiente', registro_calculado, auditoria_calculada,
        huella_auditoria, huella_entrada, instante, huella_evento
    );
    UPDATE vec_ejecucion_documental_v4.control_cadena_auditoria
       SET ultima_secuencia = secuencia,
           ultima_huella_sha256 = huella_auditoria
     WHERE control_id = true
       AND ultima_secuencia = secuencia - 1
       AND ultima_huella_sha256 = huella_anterior;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'OCC de auditoria documental perdido';
    END IF;

    resultado := 'registrada';
    registro_ref := registro_calculado;
    estado := 'autoridad_objeto_registrada';
    version_estado := 1;
    auditoria_ref := auditoria_calculada;
    evento_outbox_ref := evento_calculado;
    huella_proyeccion_sha256 := huella_entrada;
    registrada_en := instante;
    RETURN NEXT;
EXCEPTION
    WHEN data_exception THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'proyeccion de autoridad documental no valida';
    WHEN no_data_found OR too_many_rows THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'orden V4 ausente o ambigua';
END
$funcion$;

DO $protecciones$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'registro_autoridad_objeto_esperado_v1',
        'auditoria_autoridad_objeto_v1',
        'outbox_autoridad_objeto_v1'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER %I_inmutable BEFORE UPDATE OR DELETE ON vec_ejecucion_documental_v4.%I FOR EACH ROW EXECUTE FUNCTION vec_ejecucion_documental_v4.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
        EXECUTE format(
            'CREATE TRIGGER %I_no_truncar BEFORE TRUNCATE ON vec_ejecucion_documental_v4.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_ejecucion_documental_v4.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
        EXECUTE format(
            'ALTER TABLE vec_ejecucion_documental_v4.%I ENABLE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'ALTER TABLE vec_ejecucion_documental_v4.%I FORCE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'CREATE POLICY acceso_propietario_exacto ON vec_ejecucion_documental_v4.%I FOR ALL TO vec_ejecucion_documental_v4_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
            tabla, 'vec_ejecucion_documental_v4_propietario',
            'vec_ejecucion_documental_v4_propietario'
        );
    END LOOP;
END
$protecciones$;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_ejecucion_documental_v4 FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_ejecucion_documental_v4 FROM PUBLIC;
DO $cerrar_tipos$
DECLARE
    tipo text;
BEGIN
    FOREACH tipo IN ARRAY ARRAY[
        'registro_autoridad_objeto_esperado_v1',
        'auditoria_autoridad_objeto_v1',
        'outbox_autoridad_objeto_v1'
    ] LOOP
        EXECUTE format(
            'REVOKE ALL ON TYPE vec_ejecucion_documental_v4.%I FROM PUBLIC',
            tipo
        );
    END LOOP;
END
$cerrar_tipos$;

-- No se concede EXECUTE a ningun runtime. El siguiente corte debe incorporar
-- una credencial dedicada que haya invocado PrepararRegistro en el proceso.
COMMENT ON FUNCTION
vec_ejecucion_documental_v4.registrar_autoridad_objeto_esperado_v1(
    numeric, bytea
) IS
    'Registra solo la salida especializada de PrepararRegistro; sin concesion runtime hasta el adaptador V2 durable.';
COMMIT;
