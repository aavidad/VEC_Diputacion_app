-- Fachadas cerradas para crear, leer y evolucionar la saga con CAS y cercado.
BEGIN;
SET LOCAL ROLE vec_bolsa_firma_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
BEGIN
    IF to_regclass('vec_bolsa_firma.version_flujo') IS NULL OR
       to_regprocedure(
         'vec_bolsa_firma.crear_o_recuperar_flujo_v1(jsonb,bytea)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar operaciones de firma';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_bolsa_firma.expediente_valido(
    p_documento jsonb,
    p_cifrado bytea
)
RETURNS boolean
LANGUAGE plpgsql
STABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    v_punto jsonb;
    v_indice integer := 0;
    v_total integer;
    v_claves_esperadas integer;
    v_pasos constant text[] := ARRAY[
        'preparar_firma', 'completar_firma', 'completar_firma',
        'custodiar_firma', 'retener_firma', 'reservar_cambio',
        'confirmar_cambio'
    ];
BEGIN
    IF p_documento IS NULL OR jsonb_typeof(p_documento) <> 'object' OR
       octet_length(convert_to(p_documento::text, 'UTF8'))
           NOT BETWEEN 2 AND 262144 OR
       (SELECT count(*) FROM jsonb_object_keys(p_documento)) <>
           17 + (p_documento ? 'proyeccion_lanzamiento')::integer +
                (p_documento ? 'resultado')::integer OR
       NOT (p_documento ?& ARRAY[
           'esquema', 'flujo_ref', 'version',
           'indice_idempotencia_hmac', 'huella_solicitud_hmac',
           'vinculo_actor_hmac', 'perfil_actor_clave', 'proceso_ref',
           'solicitud_ref', 'baremacion_merito_ref', 'decision_ref',
           'estado', 'estado_protegido', 'puntos_control',
           'creado_en', 'actualizado_en', 'sello_estado_hmac'
       ]) OR
       p_documento ->> 'esquema' IS DISTINCT FROM
           'vec.bolsa.firma.expediente-postgresql.v1' OR
       vec_bolsa_firma.texto_opaco_valido(
           p_documento ->> 'flujo_ref', 512
       ) IS NOT TRUE OR
       vec_bolsa_firma.entero_canonico_valido(
           p_documento ->> 'version'
       ) IS NOT TRUE OR
       vec_bolsa_firma.huella_hmac_valida(
           p_documento ->> 'indice_idempotencia_hmac'
       ) IS NOT TRUE OR
       vec_bolsa_firma.huella_hmac_valida(
           p_documento ->> 'huella_solicitud_hmac'
       ) IS NOT TRUE OR
       vec_bolsa_firma.huella_hmac_valida(
           p_documento ->> 'vinculo_actor_hmac'
       ) IS NOT TRUE OR
       (p_documento ->> 'perfil_actor_clave') !~
           '^[a-z][a-z0-9._-]{0,127}$' OR
       vec_bolsa_firma.texto_opaco_valido(
           p_documento ->> 'proceso_ref', 512
       ) IS NOT TRUE OR
       vec_bolsa_firma.texto_opaco_valido(
           p_documento ->> 'solicitud_ref', 512
       ) IS NOT TRUE OR
       vec_bolsa_firma.texto_opaco_valido(
           p_documento ->> 'baremacion_merito_ref', 512
       ) IS NOT TRUE OR
       vec_bolsa_firma.texto_opaco_valido(
           p_documento ->> 'decision_ref', 512
       ) IS NOT TRUE OR
       p_documento ->> 'estado' NOT IN (
           'preparando', 'pendiente_interaccion', 'finalizando', 'completado'
       ) OR
       vec_bolsa_firma.instante_utc_valido(
           p_documento ->> 'creado_en'
       ) IS NOT TRUE OR
       vec_bolsa_firma.instante_utc_valido(
           p_documento ->> 'actualizado_en'
       ) IS NOT TRUE OR
       (p_documento ->> 'actualizado_en')::timestamptz <
           (p_documento ->> 'creado_en')::timestamptz OR
       vec_bolsa_firma.huella_hmac_valida(
           p_documento ->> 'sello_estado_hmac'
       ) IS NOT TRUE OR
       jsonb_typeof(p_documento -> 'estado_protegido') <> 'object' OR
       (SELECT count(*) FROM jsonb_object_keys(
           p_documento -> 'estado_protegido'
       )) <> 5 OR
       NOT ((p_documento -> 'estado_protegido') ?& ARRAY[
           'esquema', 'algoritmo', 'clave_ref', 'nonce_hex', 'huella_sha256'
       ]) OR
       p_documento #>> '{estado_protegido,esquema}' IS DISTINCT FROM
           'bolsa.firma.estado-protegido.v1' OR
       p_documento #>> '{estado_protegido,algoritmo}' IS DISTINCT FROM
           'aes-256-gcm' OR
       vec_bolsa_firma.texto_opaco_valido(
           p_documento #>> '{estado_protegido,clave_ref}', 256
       ) IS NOT TRUE OR
       (p_documento #>> '{estado_protegido,nonce_hex}') !~
           '^[0-9a-f]{24}$' OR
       vec_bolsa_firma.huella_sha256_valida(
           p_documento #>> '{estado_protegido,huella_sha256}'
       ) IS NOT TRUE OR
       octet_length(p_cifrado) NOT BETWEEN 16 AND 67108928 OR
       encode(sha256(p_cifrado), 'hex') IS DISTINCT FROM
           p_documento #>> '{estado_protegido,huella_sha256}' OR
       jsonb_typeof(p_documento -> 'puntos_control') <> 'array'
    THEN
        RETURN false;
    END IF;

    v_total := jsonb_array_length(p_documento -> 'puntos_control');
    IF v_total > 7 THEN
        RETURN false;
    END IF;
    FOR v_punto IN
        SELECT value FROM jsonb_array_elements(
            p_documento -> 'puntos_control'
        )
    LOOP
        v_indice := v_indice + 1;
        v_claves_esperadas := 5;
        IF v_punto ? 'resultado_ref' THEN
            v_claves_esperadas := v_claves_esperadas + 1;
        END IF;
        IF v_punto ? 'huella_resultado_sha256' THEN
            v_claves_esperadas := v_claves_esperadas + 1;
        END IF;
        IF v_punto ? 'completado_en' THEN
            v_claves_esperadas := v_claves_esperadas + 1;
        END IF;
        IF jsonb_typeof(v_punto) <> 'object' OR
           (SELECT count(*) FROM jsonb_object_keys(v_punto)) <>
               v_claves_esperadas OR
           NOT (v_punto ?& ARRAY[
               'paso', 'estado', 'efecto_ref',
               'clave_idempotencia_hmac', 'declarado_en'
           ]) OR
           v_punto ->> 'paso' IS DISTINCT FROM v_pasos[v_indice] OR
           v_punto ->> 'estado' NOT IN ('declarado', 'completado') OR
           vec_bolsa_firma.texto_opaco_valido(
               v_punto ->> 'efecto_ref', 512
           ) IS NOT TRUE OR
           vec_bolsa_firma.huella_hmac_valida(
               v_punto ->> 'clave_idempotencia_hmac'
           ) IS NOT TRUE OR
           vec_bolsa_firma.instante_utc_valido(
               v_punto ->> 'declarado_en'
           ) IS NOT TRUE OR
           (v_punto ->> 'declarado_en')::timestamptz <
               (p_documento ->> 'creado_en')::timestamptz OR
           (v_punto ->> 'declarado_en')::timestamptz >
               (p_documento ->> 'actualizado_en')::timestamptz
        THEN
            RETURN false;
        END IF;
        IF v_punto ->> 'estado' = 'declarado' AND (
            v_punto ? 'resultado_ref' OR
            v_punto ? 'huella_resultado_sha256' OR
            v_punto ? 'completado_en'
        ) THEN
            RETURN false;
        END IF;
        IF v_punto ->> 'estado' = 'completado' AND (
            NOT (v_punto ?& ARRAY[
                'resultado_ref', 'huella_resultado_sha256', 'completado_en'
            ]) OR
            vec_bolsa_firma.texto_opaco_valido(
                v_punto ->> 'resultado_ref', 512
            ) IS NOT TRUE OR
            vec_bolsa_firma.huella_sha256_valida(
                v_punto ->> 'huella_resultado_sha256'
            ) IS NOT TRUE OR
            vec_bolsa_firma.instante_utc_valido(
                v_punto ->> 'completado_en'
            ) IS NOT TRUE OR
            (v_punto ->> 'completado_en')::timestamptz <
                (v_punto ->> 'declarado_en')::timestamptz OR
            (v_punto ->> 'completado_en')::timestamptz >
                (p_documento ->> 'actualizado_en')::timestamptz
        ) THEN
            RETURN false;
        END IF;
        IF v_indice < v_total AND
           v_punto ->> 'estado' <> 'completado' THEN
            RETURN false;
        END IF;
    END LOOP;

    IF p_documento ? 'proyeccion_lanzamiento' THEN
        IF jsonb_typeof(p_documento -> 'proyeccion_lanzamiento') <>
               'object' OR
           (SELECT count(*) FROM jsonb_object_keys(
               p_documento -> 'proyeccion_lanzamiento'
           )) <> 6 OR
           NOT ((p_documento -> 'proyeccion_lanzamiento') ?& ARRAY[
               'flujo_ref', 'sesion_firma_ref', 'lanzamiento_ref',
               'canal_lanzamiento_clave', 'preparada_en', 'expira_en'
           ]) OR
           p_documento #>> '{proyeccion_lanzamiento,flujo_ref}'
               IS DISTINCT FROM p_documento ->> 'flujo_ref' OR
           vec_bolsa_firma.texto_opaco_valido(
               p_documento #>> '{proyeccion_lanzamiento,sesion_firma_ref}',
               512
           ) IS NOT TRUE OR
           vec_bolsa_firma.texto_opaco_valido(
               p_documento #>> '{proyeccion_lanzamiento,lanzamiento_ref}',
               512
           ) IS NOT TRUE OR
           (p_documento #>>
               '{proyeccion_lanzamiento,canal_lanzamiento_clave}') !~
               '^[a-z][a-z0-9._-]{0,127}$' OR
           vec_bolsa_firma.instante_utc_valido(
               p_documento #>> '{proyeccion_lanzamiento,preparada_en}'
           ) IS NOT TRUE OR
           vec_bolsa_firma.instante_utc_valido(
               p_documento #>> '{proyeccion_lanzamiento,expira_en}'
           ) IS NOT TRUE OR
           (p_documento #>>
               '{proyeccion_lanzamiento,expira_en}')::timestamptz <=
               (p_documento #>>
               '{proyeccion_lanzamiento,preparada_en}')::timestamptz
        THEN
            RETURN false;
        END IF;
    END IF;

    IF p_documento ? 'resultado' THEN
        IF jsonb_typeof(p_documento -> 'resultado') <> 'object' OR
           (SELECT count(*) FROM jsonb_object_keys(
               p_documento -> 'resultado'
           )) <> 8 OR
           NOT ((p_documento -> 'resultado') ?& ARRAY[
               'flujo_ref', 'decision_ref', 'documento_firmado_ref',
               'huella_documento_firmado_sha256', 'version_baremacion',
               'evidencia_confirmacion_ref', 'huella_resultado_sha256',
               'completado_en'
           ]) OR
           p_documento #>> '{resultado,flujo_ref}'
               IS DISTINCT FROM p_documento ->> 'flujo_ref' OR
           p_documento #>> '{resultado,decision_ref}'
               IS DISTINCT FROM p_documento ->> 'decision_ref' OR
           vec_bolsa_firma.texto_opaco_valido(
               p_documento #>> '{resultado,documento_firmado_ref}', 512
           ) IS NOT TRUE OR
           vec_bolsa_firma.huella_sha256_valida(
               p_documento #>>
                   '{resultado,huella_documento_firmado_sha256}'
           ) IS NOT TRUE OR
           vec_bolsa_firma.entero_canonico_valido(
               p_documento #>> '{resultado,version_baremacion}'
           ) IS NOT TRUE OR
           (p_documento #>> '{resultado,version_baremacion}')::bigint < 2 OR
           vec_bolsa_firma.texto_opaco_valido(
               p_documento #>> '{resultado,evidencia_confirmacion_ref}', 512
           ) IS NOT TRUE OR
           vec_bolsa_firma.huella_sha256_valida(
               p_documento #>> '{resultado,huella_resultado_sha256}'
           ) IS NOT TRUE OR
           vec_bolsa_firma.instante_utc_valido(
               p_documento #>> '{resultado,completado_en}'
           ) IS NOT TRUE
        THEN
            RETURN false;
        END IF;
    END IF;

    RETURN CASE
        WHEN v_total = 0 THEN
            p_documento ->> 'estado' = 'preparando' AND
            NOT (p_documento ? 'proyeccion_lanzamiento') AND
            NOT (p_documento ? 'resultado')
        WHEN (p_documento #>>
              ARRAY['puntos_control', (v_total - 1)::text, 'estado']) =
              'declarado' THEN
            p_documento ->> 'estado' IN (
                'preparando', 'pendiente_interaccion', 'finalizando'
            ) AND NOT (p_documento ? 'resultado')
        WHEN v_total = 1 THEN
            p_documento ->> 'estado' = 'pendiente_interaccion' AND
            p_documento ? 'proyeccion_lanzamiento' AND
            NOT (p_documento ? 'resultado')
        WHEN v_total = 7 THEN
            p_documento ->> 'estado' = 'completado' AND
            p_documento ? 'proyeccion_lanzamiento' AND
            p_documento ? 'resultado'
        ELSE
            p_documento ->> 'estado' = 'finalizando' AND
            p_documento ? 'proyeccion_lanzamiento' AND
            NOT (p_documento ? 'resultado')
    END;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

CREATE FUNCTION vec_bolsa_firma.transicion_valida(
    p_anterior jsonb,
    p_siguiente jsonb
)
RETURNS boolean
LANGUAGE plpgsql
STABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    v_anteriores integer;
    v_siguientes integer;
    v_ultimo_anterior jsonb;
    v_ultimo_siguiente jsonb;
BEGIN
    IF p_anterior IS NULL OR p_siguiente IS NULL OR
       (p_siguiente ->> 'version')::bigint <>
           (p_anterior ->> 'version')::bigint + 1 OR
       p_anterior ->> 'flujo_ref' IS DISTINCT FROM
           p_siguiente ->> 'flujo_ref' OR
       p_anterior ->> 'indice_idempotencia_hmac' IS DISTINCT FROM
           p_siguiente ->> 'indice_idempotencia_hmac' OR
       p_anterior ->> 'huella_solicitud_hmac' IS DISTINCT FROM
           p_siguiente ->> 'huella_solicitud_hmac' OR
       p_anterior ->> 'vinculo_actor_hmac' IS DISTINCT FROM
           p_siguiente ->> 'vinculo_actor_hmac' OR
       p_anterior ->> 'perfil_actor_clave' IS DISTINCT FROM
           p_siguiente ->> 'perfil_actor_clave' OR
       p_anterior ->> 'proceso_ref' IS DISTINCT FROM
           p_siguiente ->> 'proceso_ref' OR
       p_anterior ->> 'solicitud_ref' IS DISTINCT FROM
           p_siguiente ->> 'solicitud_ref' OR
       p_anterior ->> 'baremacion_merito_ref' IS DISTINCT FROM
           p_siguiente ->> 'baremacion_merito_ref' OR
       p_anterior ->> 'decision_ref' IS DISTINCT FROM
           p_siguiente ->> 'decision_ref' OR
       p_anterior ->> 'creado_en' IS DISTINCT FROM
           p_siguiente ->> 'creado_en' OR
       (p_siguiente ->> 'actualizado_en')::timestamptz <
           (p_anterior ->> 'actualizado_en')::timestamptz
    THEN
        RETURN false;
    END IF;
    v_anteriores := jsonb_array_length(p_anterior -> 'puntos_control');
    v_siguientes := jsonb_array_length(p_siguiente -> 'puntos_control');

    IF v_siguientes = v_anteriores + 1 THEN
        RETURN (p_siguiente -> 'puntos_control') - v_anteriores =
                   p_anterior -> 'puntos_control' AND
               p_siguiente #>>
                   ARRAY['puntos_control', v_anteriores::text, 'estado'] =
                   'declarado' AND
               p_anterior -> 'estado_protegido' =
                   p_siguiente -> 'estado_protegido' AND
               p_anterior -> 'proyeccion_lanzamiento' IS NOT DISTINCT FROM
                   p_siguiente -> 'proyeccion_lanzamiento' AND
               p_anterior -> 'resultado' IS NOT DISTINCT FROM
                   p_siguiente -> 'resultado';
    END IF;
    IF v_siguientes <> v_anteriores OR v_anteriores = 0 OR
       (p_siguiente -> 'puntos_control') - (v_siguientes - 1) <>
       (p_anterior -> 'puntos_control') - (v_anteriores - 1)
    THEN
        RETURN false;
    END IF;
    v_ultimo_anterior :=
        p_anterior #> ARRAY['puntos_control', (v_anteriores - 1)::text];
    v_ultimo_siguiente :=
        p_siguiente #> ARRAY['puntos_control', (v_siguientes - 1)::text];
    IF v_ultimo_anterior ->> 'estado' <> 'declarado' OR
       v_ultimo_siguiente ->> 'estado' <> 'completado' OR
       v_ultimo_anterior ->> 'paso' IS DISTINCT FROM
           v_ultimo_siguiente ->> 'paso' OR
       v_ultimo_anterior ->> 'efecto_ref' IS DISTINCT FROM
           v_ultimo_siguiente ->> 'efecto_ref' OR
       v_ultimo_anterior ->> 'clave_idempotencia_hmac' IS DISTINCT FROM
           v_ultimo_siguiente ->> 'clave_idempotencia_hmac' OR
       v_ultimo_anterior ->> 'declarado_en' IS DISTINCT FROM
           v_ultimo_siguiente ->> 'declarado_en'
    THEN
        RETURN false;
    END IF;
    IF v_ultimo_siguiente ->> 'paso' = 'preparar_firma' THEN
        RETURN NOT (p_anterior ? 'proyeccion_lanzamiento') AND
               p_siguiente ? 'proyeccion_lanzamiento' AND
               NOT (p_anterior ? 'resultado') AND
               NOT (p_siguiente ? 'resultado');
    END IF;
    IF v_ultimo_siguiente ->> 'paso' = 'confirmar_cambio' THEN
        RETURN p_anterior -> 'proyeccion_lanzamiento' IS NOT DISTINCT FROM
                   p_siguiente -> 'proyeccion_lanzamiento' AND
               NOT (p_anterior ? 'resultado') AND
               p_siguiente ? 'resultado';
    END IF;
    RETURN p_anterior -> 'proyeccion_lanzamiento' IS NOT DISTINCT FROM
               p_siguiente -> 'proyeccion_lanzamiento' AND
           p_anterior -> 'resultado' IS NOT DISTINCT FROM
               p_siguiente -> 'resultado';
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

CREATE FUNCTION vec_bolsa_firma.crear_o_recuperar_flujo_v1(
    p_documento jsonb,
    p_cifrado bytea
)
RETURNS TABLE (
    resultado text,
    expediente_documento jsonb,
    estado_cifrado bytea
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_existente record;
    v_huella_cifrado text;
    v_huella_documento text;
BEGIN
    IF current_setting('transaction_isolation') <> 'serializable' OR
       current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION USING ERRCODE = '25001',
            MESSAGE = 'crear flujo requiere SERIALIZABLE READ WRITE';
    END IF;
    IF vec_bolsa_firma.expediente_valido(
           p_documento, p_cifrado
       ) IS NOT TRUE OR
       p_documento ->> 'version' <> '1' OR
       p_documento ->> 'estado' <> 'preparando' OR
       jsonb_array_length(p_documento -> 'puntos_control') <> 0 OR
       (p_documento ->> 'actualizado_en')::timestamptz > clock_timestamp()
    THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'expediente inicial de firma inválido';
    END IF;
    PERFORM pg_advisory_xact_lock(hashtextextended(
        'firma:idempotencia:' ||
        (p_documento ->> 'indice_idempotencia_hmac'), 0
    ));
    PERFORM pg_advisory_xact_lock(hashtextextended(
        'firma:referencia:' || (p_documento ->> 'flujo_ref'), 0
    ));

    SELECT f.*, v.expediente_documento, c.cifrado
      INTO v_existente
      FROM vec_bolsa_firma.flujo AS f
      JOIN vec_bolsa_firma.version_flujo AS v
        ON v.flujo_ref = f.flujo_ref AND v.version = 1
      JOIN vec_bolsa_firma.estado_cifrado AS c
        ON c.huella_sha256 = v.huella_cifrado_sha256
     WHERE f.indice_idempotencia_hmac =
               p_documento ->> 'indice_idempotencia_hmac'
     FOR UPDATE OF f;
    IF FOUND THEN
        IF v_existente.huella_solicitud_hmac =
               p_documento ->> 'huella_solicitud_hmac' AND
           v_existente.vinculo_actor_hmac =
               p_documento ->> 'vinculo_actor_hmac' AND
           v_existente.perfil_actor_clave =
               p_documento ->> 'perfil_actor_clave' AND
           v_existente.proceso_ref = p_documento ->> 'proceso_ref' AND
           v_existente.solicitud_ref = p_documento ->> 'solicitud_ref' AND
           v_existente.baremacion_merito_ref =
               p_documento ->> 'baremacion_merito_ref' AND
           v_existente.decision_ref = p_documento ->> 'decision_ref'
        THEN
            RETURN QUERY SELECT
                'recuperado'::text,
                v_existente.expediente_documento,
                v_existente.cifrado;
        ELSE
            RETURN QUERY SELECT 'reutilizada'::text, NULL::jsonb, NULL::bytea;
        END IF;
        RETURN;
    END IF;
    IF EXISTS (
        SELECT 1 FROM vec_bolsa_firma.flujo
         WHERE flujo_ref = p_documento ->> 'flujo_ref'
    ) THEN
        RETURN QUERY SELECT 'conflicto'::text, NULL::jsonb, NULL::bytea;
        RETURN;
    END IF;

    v_huella_cifrado :=
        p_documento #>> '{estado_protegido,huella_sha256}';
    v_huella_documento := encode(
        sha256(convert_to(p_documento::text, 'UTF8')), 'hex'
    );
    INSERT INTO vec_bolsa_firma.estado_cifrado(huella_sha256, cifrado)
    VALUES (v_huella_cifrado, p_cifrado)
    ON CONFLICT (huella_sha256) DO NOTHING;
    IF NOT EXISTS (
        SELECT 1 FROM vec_bolsa_firma.estado_cifrado
         WHERE huella_sha256 = v_huella_cifrado AND cifrado = p_cifrado
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'colisión de estado cifrado';
    END IF;
    INSERT INTO vec_bolsa_firma.flujo(
        flujo_ref, indice_idempotencia_hmac, huella_solicitud_hmac,
        vinculo_actor_hmac, perfil_actor_clave, proceso_ref, solicitud_ref,
        baremacion_merito_ref, decision_ref, version_actual
    ) VALUES (
        p_documento ->> 'flujo_ref',
        p_documento ->> 'indice_idempotencia_hmac',
        p_documento ->> 'huella_solicitud_hmac',
        p_documento ->> 'vinculo_actor_hmac',
        p_documento ->> 'perfil_actor_clave',
        p_documento ->> 'proceso_ref',
        p_documento ->> 'solicitud_ref',
        p_documento ->> 'baremacion_merito_ref',
        p_documento ->> 'decision_ref',
        1
    );
    INSERT INTO vec_bolsa_firma.version_flujo(
        flujo_ref, version, expediente_documento,
        huella_documento_sha256, huella_cifrado_sha256
    ) VALUES (
        p_documento ->> 'flujo_ref', 1, p_documento,
        v_huella_documento, v_huella_cifrado
    );
    PERFORM vec_bolsa_firma.registrar_evidencia(
        p_documento ->> 'flujo_ref', 1, 'flujo_creado',
        jsonb_build_object(
            'huella_documento_sha256', v_huella_documento,
            'huella_cifrado_sha256', v_huella_cifrado
        ),
        true
    );
    RETURN QUERY SELECT 'creado'::text, p_documento, p_cifrado;
END
$funcion$;

CREATE FUNCTION vec_bolsa_firma.obtener_flujo_v1(
    p_flujo_ref text,
    p_indice_idempotencia_hmac text,
    p_vinculo_actor_hmac text
)
RETURNS TABLE (
    expediente_documento jsonb,
    estado_cifrado bytea
)
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    -- La lectura pública comienza en READ ONLY. Guardar reutiliza esta misma
    -- fachada dentro de su única transacción SERIALIZABLE READ WRITE para
    -- validar la transición antes del CAS; la función sigue declarada STABLE
    -- y no posee ninguna sentencia de mutación.
    IF current_setting('transaction_isolation') <> 'serializable' OR
       vec_bolsa_firma.texto_opaco_valido(p_flujo_ref, 512) IS NOT TRUE OR
       vec_bolsa_firma.huella_hmac_valida(
           p_indice_idempotencia_hmac
       ) IS NOT TRUE OR
       vec_bolsa_firma.huella_hmac_valida(
           p_vinculo_actor_hmac
       ) IS NOT TRUE
    THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'consulta de flujo inválida';
    END IF;
    RETURN QUERY
    SELECT v.expediente_documento, c.cifrado
      FROM vec_bolsa_firma.flujo AS f
      JOIN vec_bolsa_firma.version_flujo AS v
        ON v.flujo_ref = f.flujo_ref AND v.version = f.version_actual
      JOIN vec_bolsa_firma.estado_cifrado AS c
        ON c.huella_sha256 = v.huella_cifrado_sha256
     WHERE f.flujo_ref = p_flujo_ref
       AND f.indice_idempotencia_hmac = p_indice_idempotencia_hmac
       AND f.vinculo_actor_hmac = p_vinculo_actor_hmac;
END
$funcion$;

CREATE FUNCTION vec_bolsa_firma.adquirir_arrendamiento_v1(
    p_operacion jsonb,
    p_huella_token_hmac bytea
)
RETURNS TABLE (
    resultado text,
    expediente_documento jsonb,
    estado_cifrado bytea,
    secuencia_cercado text,
    expira_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_flujo record;
    v_arrendamiento record;
    v_ahora timestamptz(6);
    v_expira timestamptz(6);
    v_secuencia bigint;
    v_duracion bigint;
BEGIN
    IF current_setting('transaction_isolation') <> 'serializable' OR
       current_setting('transaction_read_only') <> 'off' OR
       p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object' OR
       (SELECT count(*) FROM jsonb_object_keys(p_operacion)) <> 7 OR
       NOT (p_operacion ?& ARRAY[
           'esquema', 'flujo_ref', 'indice_idempotencia_hmac',
           'vinculo_actor_hmac', 'version_esperada', 'propietario_ref',
           'duracion_microsegundos'
       ]) OR
       p_operacion ->> 'esquema' IS DISTINCT FROM
           'vec.bolsa.firma.arrendamiento-postgresql.v1' OR
       vec_bolsa_firma.texto_opaco_valido(
           p_operacion ->> 'flujo_ref', 512
       ) IS NOT TRUE OR
       vec_bolsa_firma.huella_hmac_valida(
           p_operacion ->> 'indice_idempotencia_hmac'
       ) IS NOT TRUE OR
       vec_bolsa_firma.huella_hmac_valida(
           p_operacion ->> 'vinculo_actor_hmac'
       ) IS NOT TRUE OR
       vec_bolsa_firma.entero_canonico_valido(
           p_operacion ->> 'version_esperada'
       ) IS NOT TRUE OR
       vec_bolsa_firma.texto_opaco_valido(
           p_operacion ->> 'propietario_ref', 512
       ) IS NOT TRUE OR
       vec_bolsa_firma.entero_canonico_valido(
           p_operacion ->> 'duracion_microsegundos'
       ) IS NOT TRUE OR
       (p_operacion ->> 'duracion_microsegundos')::bigint
           NOT BETWEEN 1000000 AND 300000000 OR
       octet_length(p_huella_token_hmac) <> 32
    THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'adquisición de arrendamiento inválida';
    END IF;
    SELECT * INTO v_flujo
      FROM vec_bolsa_firma.flujo
     WHERE flujo_ref = p_operacion ->> 'flujo_ref'
       AND indice_idempotencia_hmac =
           p_operacion ->> 'indice_idempotencia_hmac'
       AND vinculo_actor_hmac = p_operacion ->> 'vinculo_actor_hmac'
     FOR UPDATE;
    IF NOT FOUND THEN
        RETURN QUERY SELECT
            'no_encontrado'::text, NULL::jsonb, NULL::bytea,
            NULL::text, NULL::timestamptz;
        RETURN;
    END IF;
    IF v_flujo.version_actual <>
       (p_operacion ->> 'version_esperada')::bigint THEN
        RETURN QUERY SELECT
            'conflicto'::text, NULL::jsonb, NULL::bytea,
            NULL::text, NULL::timestamptz;
        RETURN;
    END IF;
    v_ahora := clock_timestamp();
    SELECT * INTO v_arrendamiento
      FROM vec_bolsa_firma.arrendamiento
     WHERE flujo_ref = v_flujo.flujo_ref
     FOR UPDATE;
    IF FOUND AND v_arrendamiento.expira_en > v_ahora THEN
        RETURN QUERY SELECT
            'ocupado'::text, NULL::jsonb, NULL::bytea,
            NULL::text, NULL::timestamptz;
        RETURN;
    END IF;
    v_duracion := (p_operacion ->> 'duracion_microsegundos')::bigint;
    v_expira := v_ahora + v_duracion * interval '1 microsecond';
    UPDATE vec_bolsa_firma.flujo AS destino
       SET secuencia_cercado = destino.secuencia_cercado + 1
     WHERE destino.flujo_ref = v_flujo.flujo_ref
     RETURNING destino.secuencia_cercado INTO v_secuencia;
    INSERT INTO vec_bolsa_firma.arrendamiento(
        flujo_ref, propietario_ref, secuencia_cercado, expira_en,
        huella_token_hmac, adquirido_en
    ) VALUES (
        v_flujo.flujo_ref, p_operacion ->> 'propietario_ref',
        v_secuencia, v_expira, p_huella_token_hmac, v_ahora
    )
    ON CONFLICT (flujo_ref) DO UPDATE SET
        propietario_ref = EXCLUDED.propietario_ref,
        secuencia_cercado = EXCLUDED.secuencia_cercado,
        expira_en = EXCLUDED.expira_en,
        huella_token_hmac = EXCLUDED.huella_token_hmac,
        adquirido_en = EXCLUDED.adquirido_en;
    PERFORM vec_bolsa_firma.registrar_evidencia(
        v_flujo.flujo_ref, v_flujo.version_actual,
        'arrendamiento_adquirido',
        jsonb_build_object(
            'propietario_ref', p_operacion ->> 'propietario_ref',
            'secuencia_cercado', v_secuencia::text,
            'expira_en', to_char(
                v_expira AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            )
        ),
        false
    );
    RETURN QUERY
    SELECT 'adquirido'::text, v.expediente_documento, c.cifrado,
           v_secuencia::text, v_expira
      FROM vec_bolsa_firma.version_flujo AS v
      JOIN vec_bolsa_firma.estado_cifrado AS c
        ON c.huella_sha256 = v.huella_cifrado_sha256
     WHERE v.flujo_ref = v_flujo.flujo_ref
       AND v.version = v_flujo.version_actual;
END
$funcion$;

CREATE FUNCTION vec_bolsa_firma.guardar_flujo_v1(
    p_operacion jsonb,
    p_documento jsonb,
    p_cifrado bytea,
    p_huella_token_hmac bytea
)
RETURNS TABLE (
    resultado text,
    expediente_documento jsonb,
    estado_cifrado bytea
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_flujo record;
    v_arrendamiento record;
    v_anterior jsonb;
    v_huella_cifrado text;
    v_huella_documento text;
    v_version_esperada bigint;
    v_ahora timestamptz(6);
BEGIN
    IF current_setting('transaction_isolation') <> 'serializable' OR
       current_setting('transaction_read_only') <> 'off' OR
       p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object' OR
       (SELECT count(*) FROM jsonb_object_keys(p_operacion)) <> 6 OR
       NOT (p_operacion ?& ARRAY[
           'esquema', 'flujo_ref', 'version_esperada',
           'propietario_ref', 'secuencia_cercado', 'expira_en'
       ]) OR
       p_operacion ->> 'esquema' IS DISTINCT FROM
           'vec.bolsa.firma.arrendamiento-postgresql.v1' OR
       vec_bolsa_firma.texto_opaco_valido(
           p_operacion ->> 'flujo_ref', 512
       ) IS NOT TRUE OR
       vec_bolsa_firma.entero_canonico_valido(
           p_operacion ->> 'version_esperada'
       ) IS NOT TRUE OR
       vec_bolsa_firma.texto_opaco_valido(
           p_operacion ->> 'propietario_ref', 512
       ) IS NOT TRUE OR
       vec_bolsa_firma.entero_canonico_valido(
           p_operacion ->> 'secuencia_cercado'
       ) IS NOT TRUE OR
       vec_bolsa_firma.instante_utc_valido(
           p_operacion ->> 'expira_en'
       ) IS NOT TRUE OR
       octet_length(p_huella_token_hmac) <> 32 OR
       vec_bolsa_firma.expediente_valido(
           p_documento, p_cifrado
       ) IS NOT TRUE
    THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'guardado de flujo inválido';
    END IF;
    v_version_esperada :=
        (p_operacion ->> 'version_esperada')::bigint;
    IF p_documento ->> 'flujo_ref' IS DISTINCT FROM
           p_operacion ->> 'flujo_ref' OR
       (p_documento ->> 'version')::bigint <> v_version_esperada + 1
    THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'versión siguiente de flujo inválida';
    END IF;
    SELECT * INTO v_flujo
      FROM vec_bolsa_firma.flujo
     WHERE flujo_ref = p_operacion ->> 'flujo_ref'
     FOR UPDATE;
    IF NOT FOUND THEN
        RETURN QUERY SELECT 'no_encontrado'::text, NULL::jsonb, NULL::bytea;
        RETURN;
    END IF;
    IF v_flujo.version_actual <> v_version_esperada THEN
        RETURN QUERY SELECT 'conflicto'::text, NULL::jsonb, NULL::bytea;
        RETURN;
    END IF;
    SELECT * INTO v_arrendamiento
      FROM vec_bolsa_firma.arrendamiento
     WHERE flujo_ref = v_flujo.flujo_ref
     FOR UPDATE;
    v_ahora := clock_timestamp();
    IF NOT FOUND OR v_arrendamiento.expira_en <= v_ahora OR
       v_arrendamiento.propietario_ref IS DISTINCT FROM
           p_operacion ->> 'propietario_ref' OR
       v_arrendamiento.secuencia_cercado <>
           (p_operacion ->> 'secuencia_cercado')::bigint OR
       v_arrendamiento.expira_en <>
           (p_operacion ->> 'expira_en')::timestamptz OR
       v_arrendamiento.huella_token_hmac <> p_huella_token_hmac
    THEN
        RETURN QUERY SELECT
            'arrendamiento_invalido'::text, NULL::jsonb, NULL::bytea;
        RETURN;
    END IF;
    SELECT version_actual.expediente_documento INTO v_anterior
      FROM vec_bolsa_firma.version_flujo AS version_actual
     WHERE version_actual.flujo_ref = v_flujo.flujo_ref
       AND version_actual.version = v_flujo.version_actual;
    IF vec_bolsa_firma.transicion_valida(
           v_anterior, p_documento
       ) IS NOT TRUE OR
       (p_documento ->> 'actualizado_en')::timestamptz > v_ahora
    THEN
        RETURN QUERY SELECT
            'estado_alterado'::text, NULL::jsonb, NULL::bytea;
        RETURN;
    END IF;

    v_huella_cifrado :=
        p_documento #>> '{estado_protegido,huella_sha256}';
    v_huella_documento := encode(
        sha256(convert_to(p_documento::text, 'UTF8')), 'hex'
    );
    INSERT INTO vec_bolsa_firma.estado_cifrado(huella_sha256, cifrado)
    VALUES (v_huella_cifrado, p_cifrado)
    ON CONFLICT (huella_sha256) DO NOTHING;
    IF NOT EXISTS (
        SELECT 1 FROM vec_bolsa_firma.estado_cifrado
         WHERE huella_sha256 = v_huella_cifrado AND cifrado = p_cifrado
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'colisión de estado cifrado';
    END IF;
    INSERT INTO vec_bolsa_firma.version_flujo(
        flujo_ref, version, expediente_documento,
        huella_documento_sha256, huella_cifrado_sha256
    ) VALUES (
        v_flujo.flujo_ref, v_version_esperada + 1, p_documento,
        v_huella_documento, v_huella_cifrado
    );
    UPDATE vec_bolsa_firma.flujo
       SET version_actual = v_version_esperada + 1
     WHERE flujo_ref = v_flujo.flujo_ref;
    PERFORM vec_bolsa_firma.registrar_evidencia(
        v_flujo.flujo_ref, v_version_esperada + 1, 'version_guardada',
        jsonb_build_object(
            'huella_documento_sha256', v_huella_documento,
            'huella_cifrado_sha256', v_huella_cifrado,
            'secuencia_cercado',
                v_arrendamiento.secuencia_cercado::text
        ),
        true
    );
    RETURN QUERY SELECT 'guardado'::text, p_documento, p_cifrado;
END
$funcion$;

CREATE FUNCTION vec_bolsa_firma.liberar_arrendamiento_v1(
    p_operacion jsonb,
    p_huella_token_hmac bytea
)
RETURNS TABLE (resultado text)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_arrendamiento record;
    v_version bigint;
BEGIN
    IF current_setting('transaction_isolation') <> 'serializable' OR
       current_setting('transaction_read_only') <> 'off' OR
       p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object' OR
       (SELECT count(*) FROM jsonb_object_keys(p_operacion)) <> 5 OR
       NOT (p_operacion ?& ARRAY[
           'esquema', 'flujo_ref', 'propietario_ref',
           'secuencia_cercado', 'expira_en'
       ]) OR
       p_operacion ->> 'esquema' IS DISTINCT FROM
           'vec.bolsa.firma.arrendamiento-postgresql.v1' OR
       vec_bolsa_firma.texto_opaco_valido(
           p_operacion ->> 'flujo_ref', 512
       ) IS NOT TRUE OR
       vec_bolsa_firma.texto_opaco_valido(
           p_operacion ->> 'propietario_ref', 512
       ) IS NOT TRUE OR
       vec_bolsa_firma.entero_canonico_valido(
           p_operacion ->> 'secuencia_cercado'
       ) IS NOT TRUE OR
       vec_bolsa_firma.instante_utc_valido(
           p_operacion ->> 'expira_en'
       ) IS NOT TRUE OR
       octet_length(p_huella_token_hmac) <> 32
    THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'liberación de arrendamiento inválida';
    END IF;
    SELECT * INTO v_arrendamiento
      FROM vec_bolsa_firma.arrendamiento
     WHERE flujo_ref = p_operacion ->> 'flujo_ref'
     FOR UPDATE;
    IF NOT FOUND THEN
        RETURN QUERY SELECT 'ausente'::text;
        RETURN;
    END IF;
    IF v_arrendamiento.propietario_ref IS DISTINCT FROM
           p_operacion ->> 'propietario_ref' OR
       v_arrendamiento.secuencia_cercado <>
           (p_operacion ->> 'secuencia_cercado')::bigint OR
       v_arrendamiento.expira_en <>
           (p_operacion ->> 'expira_en')::timestamptz OR
       v_arrendamiento.huella_token_hmac <> p_huella_token_hmac
    THEN
        RETURN QUERY SELECT 'arrendamiento_invalido'::text;
        RETURN;
    END IF;
    SELECT version_actual INTO v_version
      FROM vec_bolsa_firma.flujo
     WHERE flujo_ref = v_arrendamiento.flujo_ref;
    DELETE FROM vec_bolsa_firma.arrendamiento
     WHERE flujo_ref = v_arrendamiento.flujo_ref;
    PERFORM vec_bolsa_firma.registrar_evidencia(
        v_arrendamiento.flujo_ref, v_version,
        'arrendamiento_liberado',
        jsonb_build_object(
            'propietario_ref', v_arrendamiento.propietario_ref,
            'secuencia_cercado',
                v_arrendamiento.secuencia_cercado::text
        ),
        false
    );
    RETURN QUERY SELECT 'liberado'::text;
END
$funcion$;

REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_firma FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_firma TO vec_bolsa_firma_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_firma.crear_o_recuperar_flujo_v1(jsonb, bytea)
    TO vec_bolsa_firma_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_firma.obtener_flujo_v1(text, text, text)
    TO vec_bolsa_firma_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_firma.adquirir_arrendamiento_v1(jsonb, bytea)
    TO vec_bolsa_firma_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_firma.guardar_flujo_v1(jsonb, jsonb, bytea, bytea)
    TO vec_bolsa_firma_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_firma.liberar_arrendamiento_v1(jsonb, bytea)
    TO vec_bolsa_firma_ejecutor;
COMMIT;
