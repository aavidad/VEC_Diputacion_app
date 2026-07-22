-- Registro durable V3 compuesto con la autoridad unica de ContextoActor V2.
--
-- Contrato SQL exterior (cerrado mientras madura el adaptador Go):
--   registrar_decision_contexto_actor_v3(bytea, bytea, numeric, numeric)
-- recibe decision canonica V3, motivo canonico V2 y las versiones de persona
-- y perfil necesarias por la acreditacion ContextoActor. Estas dos versiones
-- no viajan en VinculoAutenticacionActorV2, pero quedan comprometidas por el
-- rca_ y su huella; la autoridad de contexto las coteja dos veces.
--
-- La funcion devuelve cero filas al rechazar, una fila probatoria sin
-- confirmacion al registrar una denegacion y un resultado minimo sellado
-- (resultado, huella e instante durable) solo despues de insertar/CAS una
-- concesion. Un replay byte a byte devuelve el mismo resultado; una colision
-- falla. La confirmacion opaca pertenece al wrapper nominal de aplicacion.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion:migracion:registro_contexto_actor_v3:000005', 0
    )
);

DO $prevalidacion$
DECLARE
    tabla regclass;
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'composicion V3 requiere migracion superusuario';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_namespace
         WHERE nspname = 'vec_autorizacion'
           AND nspowner = 'vec_autorizacion_propietario'::regrole
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_namespace
         WHERE nspname = 'vec_contexto_actor_v1'
           AND nspowner = 'vec_contexto_actor_v1_propietario'::regrole
    ) OR pg_catalog.to_regprocedure(
        'vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamp with time zone,timestamp with time zone)'
    ) IS NULL OR pg_catalog.to_regprocedure(
        'vec_autorizacion.vinculo_autenticacion_actor_v1_estructura_valida(jsonb)'
    ) IS NULL OR pg_catalog.to_regclass(
        'vec_autorizacion.motivo_v2_checkpoint_origen'
    ) IS NULL OR pg_catalog.to_regclass(
        'vec_autorizacion.decision_autorizacion_solicitud_ligada_v2'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000005 requiere ContextoActor 000002 y autorizacion 000004 en la misma base';
    END IF;
    FOREACH tabla IN ARRAY ARRAY[
        pg_catalog.to_regclass('vec_autorizacion.decision_concedida_contexto_actor_v3'),
        pg_catalog.to_regclass('vec_autorizacion.decision_denegada_contexto_actor_v3')
    ] LOOP
        IF tabla IS NOT NULL AND NOT EXISTS (
            SELECT 1
              FROM pg_catalog.pg_class AS c
             WHERE c.oid = tabla
               AND c.relowner = 'vec_autorizacion_propietario'::regrole
               AND pg_catalog.obj_description(c.oid, 'pg_class') =
                   'vec_autorizacion:registro-contexto-actor-v3:000005'
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = '000005 no adopta una tabla V3 preexistente';
        END IF;
    END LOOP;
END
$prevalidacion$;

-- Capacidad cruzada minima. No se concede ninguna tabla ni membresia.
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1
    TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION
    vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
        text, text, text, text, text, text, numeric, text, numeric,
        text, numeric, text, numeric, text, text, timestamptz, timestamptz
    ) TO vec_autorizacion_propietario;

RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;

CREATE OR REPLACE FUNCTION vec_autorizacion.vinculo_contexto_actor_v2_valido(
    p_vinculo jsonb
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    base_v1 jsonb;
BEGIN
    IF p_vinculo IS NULL OR pg_catalog.jsonb_typeof(p_vinculo) <> 'object'
       OR pg_catalog.pg_column_size(p_vinculo) > 65536
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(p_vinculo)) <> 31
       OR NOT (p_vinculo ?& ARRAY[
           'esquema', 'bloque_version', 'autenticacion_ref',
           'autenticacion_huella_sha256', 'asercion_ref', 'sesion_ref',
           'control_sesion_ref', 'control_sesion_revision',
           'control_sesion_huella_sha256', 'cuenta_ref',
           'cuenta_ordinaria_ref', 'principal_id', 'perfil_activo_ref',
           'cuenta_privilegiada', 'superficie', 'metodo_observado',
           'garantia_observada', 'politica_garantia_ref',
           'politica_garantia_huella_sha256',
           'autenticacion_verificada_en', 'sesion_emitida_en',
           'sesion_valida_hasta', 'sesion_revalidada_en',
           'registro_contexto_ref', 'contexto_actor_esquema',
           'contexto_actor_ref', 'contexto_actor_version',
           'contexto_actor_cuenta_version',
           'contexto_actor_huella_sha256',
           'manifiesto_procedencia_huella_sha256', 'autoridad_efectiva'
       ])
       OR p_vinculo ->> 'esquema' IS DISTINCT FROM
          'vec.autenticacion-actor.vinculo.v2.contexto-registrado'
       OR p_vinculo ->> 'bloque_version' IS DISTINCT FROM '2'
       OR p_vinculo ->> 'registro_contexto_ref' !~
          '^rca_[A-Za-z0-9_-]{22,128}$'
       OR p_vinculo ->> 'contexto_actor_esquema' IS DISTINCT FROM
          'vec.contexto-actor.vinculado.v2'
       OR vec_autorizacion.entero_uint64_json_valido(
              p_vinculo -> 'contexto_actor_cuenta_version'
          ) IS NOT TRUE
       OR p_vinculo ->> 'manifiesto_procedencia_huella_sha256' !~
          '^[0-9a-f]{64}$'
       OR p_vinculo ->> 'autoridad_efectiva' IS DISTINCT FROM
          'autoridad_maestra_acreditada' THEN
        RETURN false;
    END IF;

    base_v1 := pg_catalog.jsonb_set(
        p_vinculo - ARRAY[
            'esquema', 'registro_contexto_ref', 'contexto_actor_esquema',
            'contexto_actor_cuenta_version',
            'manifiesto_procedencia_huella_sha256', 'autoridad_efectiva'
        ],
        '{bloque_version}', '1'::jsonb, false
    );
    RETURN vec_autorizacion.vinculo_autenticacion_actor_v1_estructura_valida(
        base_v1
    ) IS TRUE;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR datetime_field_overflow THEN
        RETURN false;
END
$funcion$;

-- El canon V3 procede de encoding/json sobre structs cerrados de Go. jsonb
-- sirve para validar el arbol, pero no puede probar por si solo los bytes: pierde
-- orden, espacios, escapes y claves duplicadas. Estos helpers reconstruyen la
-- unica preimagen admitida sin exponer un parser/capacidad al runtime.
CREATE OR REPLACE FUNCTION vec_autorizacion.texto_ascii_visible_v3_valido(
    p_valor text,
    p_maximo integer
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT p_valor IS NOT NULL
       AND p_maximo > 0
       AND pg_catalog.octet_length(p_valor) BETWEEN 1 AND p_maximo
       AND p_valor COLLATE "C" ~ '^[!-~]+$'
       AND pg_catalog.strpos(p_valor, '*') = 0
$funcion$;

CREATE OR REPLACE FUNCTION vec_autorizacion.texto_json_go_v3(
    p_valor text
)
RETURNS text
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.replace(
             pg_catalog.replace(
               pg_catalog.replace(pg_catalog.to_json(p_valor)::text,
                                  '&', '\u0026'),
               '<', '\u003c'),
             '>', '\u003e')
$funcion$;

CREATE OR REPLACE FUNCTION vec_autorizacion.manifiesto_politicas_v3_canonico(
    p_manifiesto jsonb
)
RETURNS text
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT '[' || coalesce(pg_catalog.string_agg(
        '{"referencia":' ||
          vec_autorizacion.texto_json_go_v3(entrada ->> 'referencia') ||
        ',"huella_sha256":' ||
          vec_autorizacion.texto_json_go_v3(entrada ->> 'huella_sha256') || '}',
        ',' ORDER BY posicion
    ), '') || ']'
      FROM pg_catalog.jsonb_array_elements(p_manifiesto)
           WITH ORDINALITY AS e(entrada, posicion)
$funcion$;

CREATE OR REPLACE FUNCTION vec_autorizacion.lista_textos_v3_canonica(
    p_lista jsonb
)
RETURNS text
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT '[' || coalesce(pg_catalog.string_agg(
        vec_autorizacion.texto_json_go_v3(entrada #>> '{}'),
        ',' ORDER BY posicion
    ), '') || ']'
      FROM pg_catalog.jsonb_array_elements(p_lista)
           WITH ORDINALITY AS e(entrada, posicion)
$funcion$;

CREATE OR REPLACE FUNCTION vec_autorizacion.vinculo_contexto_actor_v2_canonico(
    p_vinculo jsonb
)
RETURNS text
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT '{"esquema":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'esquema') ||
      ',"bloque_version":' || (p_vinculo -> 'bloque_version')::text ||
      ',"autenticacion_ref":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'autenticacion_ref') ||
      ',"autenticacion_huella_sha256":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'autenticacion_huella_sha256') ||
      ',"asercion_ref":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'asercion_ref') ||
      ',"sesion_ref":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'sesion_ref') ||
      ',"control_sesion_ref":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'control_sesion_ref') ||
      ',"control_sesion_revision":' || (p_vinculo -> 'control_sesion_revision')::text ||
      ',"control_sesion_huella_sha256":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'control_sesion_huella_sha256') ||
      ',"cuenta_ref":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'cuenta_ref') ||
      ',"cuenta_ordinaria_ref":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'cuenta_ordinaria_ref') ||
      ',"principal_id":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'principal_id') ||
      ',"perfil_activo_ref":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'perfil_activo_ref') ||
      ',"cuenta_privilegiada":' || (p_vinculo -> 'cuenta_privilegiada')::text ||
      ',"superficie":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'superficie') ||
      ',"metodo_observado":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'metodo_observado') ||
      ',"garantia_observada":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'garantia_observada') ||
      ',"politica_garantia_ref":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'politica_garantia_ref') ||
      ',"politica_garantia_huella_sha256":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'politica_garantia_huella_sha256') ||
      ',"autenticacion_verificada_en":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'autenticacion_verificada_en') ||
      ',"sesion_emitida_en":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'sesion_emitida_en') ||
      ',"sesion_valida_hasta":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'sesion_valida_hasta') ||
      ',"sesion_revalidada_en":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'sesion_revalidada_en') ||
      ',"registro_contexto_ref":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'registro_contexto_ref') ||
      ',"contexto_actor_esquema":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'contexto_actor_esquema') ||
      ',"contexto_actor_ref":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'contexto_actor_ref') ||
      ',"contexto_actor_version":' || (p_vinculo -> 'contexto_actor_version')::text ||
      ',"contexto_actor_cuenta_version":' || (p_vinculo -> 'contexto_actor_cuenta_version')::text ||
      ',"contexto_actor_huella_sha256":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'contexto_actor_huella_sha256') ||
      ',"manifiesto_procedencia_huella_sha256":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'manifiesto_procedencia_huella_sha256') ||
      ',"autoridad_efectiva":' || vec_autorizacion.texto_json_go_v3(p_vinculo ->> 'autoridad_efectiva') || '}'
$funcion$;

CREATE OR REPLACE FUNCTION vec_autorizacion.decision_contexto_actor_v3_valida(
    p_documento jsonb
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    clave text;
    concedida boolean;
    vinculo jsonb;
BEGIN
    IF p_documento IS NULL
       OR pg_catalog.jsonb_typeof(p_documento) <> 'object'
       OR pg_catalog.pg_column_size(p_documento) > 524288
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(p_documento)) <> 35
       OR NOT (p_documento ?& ARRAY[
           'esquema', 'bloque_version', 'decision_ref', 'concedida',
           'codigo', 'principal_id', 'perfil_activo_ref', 'accion',
           'recurso_ref', 'modulo_id', 'tipo_recurso',
           'contexto_recurso_huella_sha256', 'finalidad',
           'correlacion_ref', 'esquema_huella_solicitud',
           'solicitud_huella_sha256', 'esquema_huella_motivo',
           'motivo_huella_sha256', 'vinculo_autenticacion_actor',
           'asignacion_ref', 'asignacion_huella_sha256',
           'version_rol_ref', 'version_rol_huella_sha256',
           'control_vigencia_version_rol_ref',
           'control_vigencia_version_rol_revision',
           'control_vigencia_version_rol_huella_sha256',
           'revision_catalogo_politicas',
           'catalogo_politicas_huella_sha256', 'politicas_evaluadas',
           'politicas_aplicables', 'garantia_minima',
           'campos_permitidos', 'obligaciones', 'emitida_en',
           'valida_hasta'
       ])
       OR p_documento ->> 'esquema' IS DISTINCT FROM
          'vec.autorizacion.decision.v3.solicitud-ligada.actor-v2'
       OR p_documento ->> 'bloque_version' IS DISTINCT FROM '3'
       OR pg_catalog.jsonb_typeof(p_documento -> 'concedida') <> 'boolean'
       OR p_documento ->> 'esquema_huella_solicitud' IS DISTINCT FROM
          'vec.autorizacion.solicitud.v3.efectiva-minimizada.actor-v2'
       OR p_documento ->> 'esquema_huella_motivo' IS DISTINCT FROM
          'vec.autorizacion.motivo.v2.referencia-opaca-catalogada'
       OR p_documento ->> 'correlacion_ref' !~
          '^correlacion_[0-9a-f]{32}$'
       OR vec_autorizacion.entero_uint64_json_valido(
              p_documento -> 'control_vigencia_version_rol_revision'
          ) IS NOT TRUE
       OR vec_autorizacion.entero_uint64_json_valido(
              p_documento -> 'revision_catalogo_politicas'
          ) IS NOT TRUE
       OR vec_autorizacion.manifiesto_decision_v2_canonico_valido(
              p_documento -> 'politicas_evaluadas'
          ) IS NOT TRUE
       OR vec_autorizacion.manifiesto_decision_v2_canonico_valido(
              p_documento -> 'politicas_aplicables'
          ) IS NOT TRUE
       OR vec_autorizacion.lista_decision_v2_canonica_valida(
              p_documento -> 'campos_permitidos'
          ) IS NOT TRUE
       OR vec_autorizacion.lista_decision_v2_canonica_valida(
              p_documento -> 'obligaciones'
          ) IS NOT TRUE
       OR p_documento ->> 'emitida_en' !~
          '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$'
       OR p_documento ->> 'valida_hasta' !~
          '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$'
       OR vec_autorizacion.instante_utc_microsegundo_valido(
              p_documento ->> 'emitida_en'
          ) IS NOT TRUE
       OR vec_autorizacion.instante_utc_microsegundo_valido(
              p_documento ->> 'valida_hasta'
          ) IS NOT TRUE THEN
        RETURN false;
    END IF;

    FOREACH clave IN ARRAY ARRAY[
        'decision_ref', 'principal_id', 'perfil_activo_ref', 'accion',
        'recurso_ref', 'modulo_id', 'tipo_recurso', 'finalidad',
        'asignacion_ref', 'version_rol_ref',
        'control_vigencia_version_rol_ref'
    ] LOOP
        IF vec_autorizacion.texto_ascii_visible_v3_valido(
               p_documento ->> clave,
               CASE WHEN clave IN ('accion') THEN 256
                    WHEN clave IN ('modulo_id', 'tipo_recurso') THEN 128
                    ELSE 512 END
           ) IS NOT TRUE THEN
            RETURN false;
        END IF;
    END LOOP;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.jsonb_array_elements(
                 (p_documento -> 'politicas_evaluadas') ||
                 (p_documento -> 'politicas_aplicables')
               ) AS politica
         WHERE vec_autorizacion.texto_ascii_visible_v3_valido(
                   politica ->> 'referencia', 512
               ) IS NOT TRUE
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.jsonb_array_elements(
                 (p_documento -> 'campos_permitidos') ||
                 (p_documento -> 'obligaciones')
               ) AS elemento
         WHERE vec_autorizacion.texto_ascii_visible_v3_valido(
                   elemento #>> '{}', 512
               ) IS NOT TRUE
    ) THEN
        RETURN false;
    END IF;
    FOREACH clave IN ARRAY ARRAY[
        'contexto_recurso_huella_sha256', 'solicitud_huella_sha256',
        'motivo_huella_sha256', 'asignacion_huella_sha256',
        'version_rol_huella_sha256',
        'control_vigencia_version_rol_huella_sha256',
        'catalogo_politicas_huella_sha256'
    ] LOOP
        IF pg_catalog.jsonb_typeof(p_documento -> clave) <> 'string'
           OR p_documento ->> clave !~ '^[0-9a-f]{64}$' THEN
            RETURN false;
        END IF;
    END LOOP;
    IF p_documento ->> 'solicitud_huella_sha256' = pg_catalog.repeat('0', 64)
       OR p_documento ->> 'motivo_huella_sha256' = pg_catalog.repeat('0', 64)
       OR p_documento ->> 'control_vigencia_version_rol_ref' IS DISTINCT FROM
          p_documento ->> 'version_rol_ref'
       OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
              vec_autorizacion.manifiesto_politicas_v3_canonico(
                  p_documento -> 'politicas_evaluadas'
              ), 'UTF8'
          )), 'hex') IS DISTINCT FROM
          p_documento ->> 'catalogo_politicas_huella_sha256' THEN
        RETURN false;
    END IF;

    concedida := (p_documento ->> 'concedida')::boolean;
    IF (concedida AND p_documento ->> 'codigo' <> 'concedida')
       OR (NOT concedida AND p_documento ->> 'codigo' NOT IN (
           'perfil_no_vigente', 'ambito_no_autorizado',
           'rol_no_publicado', 'rol_retirado', 'accion_no_concedida',
           'finalidad_no_autorizada', 'denegada_por_politica',
           'restriccion_abac_incumplida', 'garantia_insuficiente'
       ))
       OR (concedida AND p_documento ->> 'garantia_minima' NOT IN (
           'bajo', 'sustancial', 'alto'
       ))
       OR (NOT concedida AND p_documento ->> 'garantia_minima' NOT IN (
           '', 'bajo', 'sustancial', 'alto'
       )) THEN
        RETURN false;
    END IF;

    vinculo := p_documento -> 'vinculo_autenticacion_actor';
    IF vec_autorizacion.vinculo_contexto_actor_v2_valido(vinculo) IS NOT TRUE
       OR p_documento ->> 'principal_id' IS DISTINCT FROM
          vinculo ->> 'principal_id'
       OR p_documento ->> 'perfil_activo_ref' IS DISTINCT FROM
          vinculo ->> 'perfil_activo_ref'
       OR (p_documento ->> 'emitida_en')::timestamptz <
          (vinculo ->> 'sesion_revalidada_en')::timestamptz
       OR (p_documento ->> 'valida_hasta')::timestamptz >
          (vinculo ->> 'sesion_valida_hasta')::timestamptz
       OR (p_documento ->> 'valida_hasta')::timestamptz <=
          (p_documento ->> 'emitida_en')::timestamptz
       OR (p_documento ->> 'valida_hasta')::timestamptz >
          (p_documento ->> 'emitida_en')::timestamptz + interval '5 minutes'
       OR (concedida AND CASE vinculo ->> 'garantia_observada'
             WHEN 'bajo' THEN 1 WHEN 'sustancial' THEN 2 WHEN 'alto' THEN 3
             ELSE 0 END < CASE p_documento ->> 'garantia_minima'
             WHEN 'bajo' THEN 1 WHEN 'sustancial' THEN 2 WHEN 'alto' THEN 3
             ELSE 4 END)
       OR NOT ((p_documento -> 'politicas_aplicables') <@
               (p_documento -> 'politicas_evaluadas'))
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_array_elements(
                    p_documento -> 'politicas_aplicables'
                  ) AS aplicada
            WHERE NOT EXISTS (
                SELECT 1
                  FROM pg_catalog.jsonb_array_elements(
                         p_documento -> 'politicas_evaluadas'
                       ) AS evaluada
                 WHERE evaluada IS NOT DISTINCT FROM aplicada
            )
       ) THEN
        RETURN false;
    END IF;
    RETURN true;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR datetime_field_overflow THEN
        RETURN false;
END
$funcion$;

CREATE TABLE IF NOT EXISTS
vec_autorizacion.decision_concedida_contexto_actor_v3 (
    decision_ref text PRIMARY KEY,
    huella_decision_sha256 text NOT NULL UNIQUE,
    decision_canonica bytea NOT NULL,
    documento jsonb NOT NULL,
    motivo_canonico bytea NOT NULL,
    motivo_catalogo_id text NOT NULL,
    motivo_catalogo_version integer NOT NULL,
    motivo_entrada_clave text NOT NULL,
    registro_contexto_ref text NOT NULL,
    contexto_actor_huella_sha256 text NOT NULL,
    manifiesto_procedencia_huella_sha256 text NOT NULL,
    persona_version numeric(20,0) NOT NULL CHECK (
        persona_version BETWEEN 1 AND 18446744073709551615::numeric
    ),
    perfil_version numeric(20,0) NOT NULL CHECK (
        perfil_version BETWEEN 1 AND 18446744073709551615::numeric
    ),
    asignacion_ref text NOT NULL,
    version_rol_ref text NOT NULL,
    control_vigencia_version_rol_revision numeric(20,0) NOT NULL,
    revision_catalogo_politicas numeric(20,0) NOT NULL,
    emitida_en timestamptz(6) NOT NULL,
    valida_hasta timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    CONSTRAINT decision_concedida_v3_documento CHECK (
        vec_autorizacion.decision_contexto_actor_v3_valida(documento)
        AND (documento ->> 'concedida')::boolean
        AND documento ->> 'decision_ref' = decision_ref
        AND pg_catalog.encode(pg_catalog.sha256(decision_canonica), 'hex') =
            huella_decision_sha256
        AND documento ->> 'registro_contexto_ref' IS NULL
        AND documento -> 'vinculo_autenticacion_actor' ->>
            'registro_contexto_ref' = registro_contexto_ref
        AND documento -> 'vinculo_autenticacion_actor' ->>
            'contexto_actor_huella_sha256' = contexto_actor_huella_sha256
        AND documento -> 'vinculo_autenticacion_actor' ->>
            'manifiesto_procedencia_huella_sha256' =
            manifiesto_procedencia_huella_sha256
        AND documento ->> 'asignacion_ref' = asignacion_ref
        AND documento ->> 'version_rol_ref' = version_rol_ref
        AND (documento ->> 'control_vigencia_version_rol_revision')::numeric =
            control_vigencia_version_rol_revision
        AND (documento ->> 'revision_catalogo_politicas')::numeric =
            revision_catalogo_politicas
        AND (documento ->> 'emitida_en')::timestamptz = emitida_en
        AND (documento ->> 'valida_hasta')::timestamptz = valida_hasta
        AND registrada_en >= emitida_en AND registrada_en < valida_hasta
    ),
    CONSTRAINT decision_concedida_v3_huellas CHECK (
        huella_decision_sha256 ~ '^[0-9a-f]{64}$'
        AND contexto_actor_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND manifiesto_procedencia_huella_sha256 ~ '^[0-9a-f]{64}$'
    ),
    FOREIGN KEY (asignacion_ref)
        REFERENCES vec_autorizacion.asignacion_perfil(asignacion_ref),
    FOREIGN KEY (version_rol_ref)
        REFERENCES vec_autorizacion.version_rol(version_rol_ref),
    FOREIGN KEY (motivo_catalogo_id, motivo_catalogo_version)
        REFERENCES vec_autorizacion.motivo_v2_catalogo_publicado(
            catalogo_id, catalogo_version
        ),
    FOREIGN KEY (
        motivo_catalogo_id, motivo_catalogo_version, motivo_entrada_clave
    ) REFERENCES vec_autorizacion.motivo_v2_entrada(
        catalogo_id, catalogo_version, entrada_clave
    )
);

CREATE TABLE IF NOT EXISTS
vec_autorizacion.decision_denegada_contexto_actor_v3 (
    decision_ref text PRIMARY KEY,
    huella_decision_sha256 text NOT NULL UNIQUE,
    decision_canonica bytea NOT NULL,
    documento jsonb NOT NULL,
    motivo_canonico bytea NOT NULL,
    motivo_catalogo_id text NOT NULL,
    motivo_catalogo_version integer NOT NULL,
    motivo_entrada_clave text NOT NULL,
    registro_contexto_ref text NOT NULL,
    contexto_actor_huella_sha256 text NOT NULL,
    manifiesto_procedencia_huella_sha256 text NOT NULL,
    persona_version numeric(20,0) NOT NULL CHECK (
        persona_version BETWEEN 1 AND 18446744073709551615::numeric
    ),
    perfil_version numeric(20,0) NOT NULL CHECK (
        perfil_version BETWEEN 1 AND 18446744073709551615::numeric
    ),
    asignacion_ref text NOT NULL,
    version_rol_ref text NOT NULL,
    control_vigencia_version_rol_revision numeric(20,0) NOT NULL,
    revision_catalogo_politicas numeric(20,0) NOT NULL,
    codigo text NOT NULL,
    emitida_en timestamptz(6) NOT NULL,
    valida_hasta timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    CONSTRAINT decision_denegada_v3_documento CHECK (
        vec_autorizacion.decision_contexto_actor_v3_valida(documento)
        AND NOT (documento ->> 'concedida')::boolean
        AND documento ->> 'decision_ref' = decision_ref
        AND documento ->> 'codigo' = codigo
        AND pg_catalog.encode(pg_catalog.sha256(decision_canonica), 'hex') =
            huella_decision_sha256
        AND documento -> 'vinculo_autenticacion_actor' ->>
            'registro_contexto_ref' = registro_contexto_ref
        AND documento -> 'vinculo_autenticacion_actor' ->>
            'contexto_actor_huella_sha256' = contexto_actor_huella_sha256
        AND documento -> 'vinculo_autenticacion_actor' ->>
            'manifiesto_procedencia_huella_sha256' =
            manifiesto_procedencia_huella_sha256
        AND documento ->> 'asignacion_ref' = asignacion_ref
        AND documento ->> 'version_rol_ref' = version_rol_ref
        AND (documento ->> 'control_vigencia_version_rol_revision')::numeric =
            control_vigencia_version_rol_revision
        AND (documento ->> 'revision_catalogo_politicas')::numeric =
            revision_catalogo_politicas
        AND (documento ->> 'emitida_en')::timestamptz = emitida_en
        AND (documento ->> 'valida_hasta')::timestamptz = valida_hasta
        AND registrada_en >= emitida_en AND registrada_en < valida_hasta
    ),
    CONSTRAINT decision_denegada_v3_huellas CHECK (
        huella_decision_sha256 ~ '^[0-9a-f]{64}$'
        AND contexto_actor_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND manifiesto_procedencia_huella_sha256 ~ '^[0-9a-f]{64}$'
    ),
    FOREIGN KEY (asignacion_ref)
        REFERENCES vec_autorizacion.asignacion_perfil(asignacion_ref),
    FOREIGN KEY (version_rol_ref)
        REFERENCES vec_autorizacion.version_rol(version_rol_ref),
    FOREIGN KEY (motivo_catalogo_id, motivo_catalogo_version)
        REFERENCES vec_autorizacion.motivo_v2_catalogo_publicado(
            catalogo_id, catalogo_version
        ),
    FOREIGN KEY (
        motivo_catalogo_id, motivo_catalogo_version, motivo_entrada_clave
    ) REFERENCES vec_autorizacion.motivo_v2_entrada(
        catalogo_id, catalogo_version, entrada_clave
    )
);

COMMENT ON TABLE vec_autorizacion.decision_concedida_contexto_actor_v3 IS
    'vec_autorizacion:registro-contexto-actor-v3:000005';
COMMENT ON TABLE vec_autorizacion.decision_denegada_contexto_actor_v3 IS
    'vec_autorizacion:registro-contexto-actor-v3:000005';

CREATE INDEX IF NOT EXISTS decision_concedida_v3_contexto_fecha
    ON vec_autorizacion.decision_concedida_contexto_actor_v3(
        registro_contexto_ref, registrada_en DESC
    );
CREATE INDEX IF NOT EXISTS decision_denegada_v3_contexto_fecha
    ON vec_autorizacion.decision_denegada_contexto_actor_v3(
        registro_contexto_ref, registrada_en DESC
    );

DO $protecciones$
DECLARE
    tabla regclass;
    nombre text;
BEGIN
    FOREACH nombre IN ARRAY ARRAY[
        'decision_concedida_contexto_actor_v3',
        'decision_denegada_contexto_actor_v3'
    ] LOOP
        tabla := pg_catalog.to_regclass('vec_autorizacion.' || nombre);
        EXECUTE pg_catalog.format(
            'ALTER TABLE %s ENABLE ROW LEVEL SECURITY', tabla
        );
        EXECUTE pg_catalog.format(
            'ALTER TABLE %s FORCE ROW LEVEL SECURITY', tabla
        );
        IF NOT EXISTS (
            SELECT 1 FROM pg_catalog.pg_policy
             WHERE polrelid = tabla AND polname = 'acceso_propietario_exacto'
        ) THEN
            EXECUTE pg_catalog.format(
              'CREATE POLICY acceso_propietario_exacto ON %s FOR ALL TO vec_autorizacion_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
              tabla, 'vec_autorizacion_propietario',
              'vec_autorizacion_propietario'
            );
        END IF;
        IF NOT EXISTS (
            SELECT 1 FROM pg_catalog.pg_trigger
             WHERE tgrelid = tabla AND tgname = 'decision_v3_inmutable'
               AND NOT tgisinternal
        ) THEN
            EXECUTE pg_catalog.format(
              'CREATE TRIGGER decision_v3_inmutable BEFORE UPDATE OR DELETE ON %s FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',
              tabla
            );
        END IF;
        IF NOT EXISTS (
            SELECT 1 FROM pg_catalog.pg_trigger
             WHERE tgrelid = tabla AND tgname = 'decision_v3_no_truncar'
               AND NOT tgisinternal
        ) THEN
            EXECUTE pg_catalog.format(
              'CREATE TRIGGER decision_v3_no_truncar BEFORE TRUNCATE ON %s FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',
              tabla
            );
        END IF;
    END LOOP;
END
$protecciones$;

-- Revalida exclusivamente la sesion. ContextoActor no se consulta aqui: su
-- autoridad unica es vec_contexto_actor_v1 y se invoca antes y despues.
CREATE OR REPLACE FUNCTION vec_autorizacion.revalidar_sesion_vinculo_v2(
    p_vinculo jsonb,
    p_emitida_en timestamptz,
    p_valida_hasta timestamptz,
    p_instante timestamptz
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    sesion record;
BEGIN
    IF vec_autorizacion.vinculo_contexto_actor_v2_valido(p_vinculo) IS NOT TRUE
       OR p_emitida_en IS NULL OR p_valida_hasta IS NULL OR p_instante IS NULL
       OR p_valida_hasta <= p_emitida_en THEN
        RETURN false;
    END IF;
    SELECT base.autenticacion_ref, base.autenticacion_huella_sha256,
           base.asercion_ref, base.sesion_ref, base.cuenta_ref,
           base.cuenta_ordinaria_ref, base.cuenta_privilegiada,
           base.superficie, base.metodo_observado, base.garantia_observada,
           base.politica_garantia_ref,
           base.politica_garantia_huella_sha256,
           base.autenticacion_verificada_en, base.sesion_emitida_en,
           control.control_sesion_ref, control.revision, control.estado,
           control.huella_sha256, control.sesion_revalidada_en,
           control.sesion_valida_hasta
      INTO sesion
      FROM vec_autorizacion.sesion_autenticacion_v1 AS base
      JOIN vec_autorizacion.control_sesion_actual_v1 AS actual
        ON actual.sesion_ref = base.sesion_ref
      JOIN vec_autorizacion.control_sesion_v1 AS control
        ON control.sesion_ref = actual.sesion_ref
       AND control.control_sesion_ref = actual.control_sesion_ref
       AND control.revision = actual.revision
     WHERE base.sesion_ref = p_vinculo ->> 'sesion_ref'
       AND base.autenticacion_ref = p_vinculo ->> 'autenticacion_ref'
     FOR UPDATE OF actual;
    IF NOT FOUND OR sesion.estado <> 'activa'
       OR sesion.autenticacion_huella_sha256 IS DISTINCT FROM
          p_vinculo ->> 'autenticacion_huella_sha256'
       OR sesion.asercion_ref IS DISTINCT FROM p_vinculo ->> 'asercion_ref'
       OR sesion.control_sesion_ref IS DISTINCT FROM
          p_vinculo ->> 'control_sesion_ref'
       OR sesion.revision IS DISTINCT FROM
          (p_vinculo ->> 'control_sesion_revision')::numeric
       OR sesion.huella_sha256 IS DISTINCT FROM
          p_vinculo ->> 'control_sesion_huella_sha256'
       OR sesion.cuenta_ref IS DISTINCT FROM p_vinculo ->> 'cuenta_ref'
       OR sesion.cuenta_ordinaria_ref IS DISTINCT FROM
          p_vinculo ->> 'cuenta_ordinaria_ref'
       OR sesion.cuenta_privilegiada IS DISTINCT FROM
          (p_vinculo ->> 'cuenta_privilegiada')::boolean
       OR sesion.superficie IS DISTINCT FROM p_vinculo ->> 'superficie'
       OR sesion.metodo_observado IS DISTINCT FROM
          p_vinculo ->> 'metodo_observado'
       OR sesion.garantia_observada IS DISTINCT FROM
          p_vinculo ->> 'garantia_observada'
       OR sesion.politica_garantia_ref IS DISTINCT FROM
          p_vinculo ->> 'politica_garantia_ref'
       OR sesion.politica_garantia_huella_sha256 IS DISTINCT FROM
          p_vinculo ->> 'politica_garantia_huella_sha256'
       OR sesion.autenticacion_verificada_en IS DISTINCT FROM
          (p_vinculo ->> 'autenticacion_verificada_en')::timestamptz
       OR sesion.sesion_emitida_en IS DISTINCT FROM
          (p_vinculo ->> 'sesion_emitida_en')::timestamptz
       OR sesion.sesion_revalidada_en IS DISTINCT FROM
          (p_vinculo ->> 'sesion_revalidada_en')::timestamptz
       OR sesion.sesion_valida_hasta IS DISTINCT FROM
          (p_vinculo ->> 'sesion_valida_hasta')::timestamptz THEN
        RETURN false;
    END IF;
    RETURN p_emitida_en >= sesion.sesion_revalidada_en
       AND p_valida_hasta <= sesion.sesion_valida_hasta
       AND p_instante >= sesion.sesion_revalidada_en
       AND p_instante < sesion.sesion_valida_hasta;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR datetime_field_overflow OR no_data_found OR too_many_rows THEN
        RETURN false;
END
$funcion$;

-- Los tipos compuestos de tablas reciben USAGE de PUBLIC por defecto en
-- PostgreSQL. La acreditacion cerrada del runtime de contexto detecta esa
-- superficie ajena; la composicion local la elimina para todo el esquema.
DO $cerrar_tipos_publicos$
DECLARE
    tipo record;
BEGIN
    FOR tipo IN
        SELECT t.oid::regtype AS nombre
          FROM pg_catalog.pg_type AS t
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.typnamespace
         WHERE n.nspname = 'vec_autorizacion'
           AND t.typtype IN ('c', 'd', 'e', 'm', 'r')
    LOOP
        EXECUTE pg_catalog.format(
            'REVOKE ALL PRIVILEGES ON TYPE %s FROM PUBLIC', tipo.nombre
        );
    END LOOP;
END
$cerrar_tipos_publicos$;

REVOKE ALL ON FUNCTION
    vec_autorizacion.texto_ascii_visible_v3_valido(text, integer),
    vec_autorizacion.texto_json_go_v3(text),
    vec_autorizacion.manifiesto_politicas_v3_canonico(jsonb),
    vec_autorizacion.lista_textos_v3_canonica(jsonb),
    vec_autorizacion.vinculo_contexto_actor_v2_canonico(jsonb)
    FROM PUBLIC, vec_autorizacion_registro, vec_autorizacion_fuente,
         vec_autorizacion_motivos_proyector,
         vec_autorizacion_motivos_evaluador;

COMMIT;
