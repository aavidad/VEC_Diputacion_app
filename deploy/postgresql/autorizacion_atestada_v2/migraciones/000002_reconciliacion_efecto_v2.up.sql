-- Reconciliacion cerrada por las dos claves unicas del efecto VEC-AD-2.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v2_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v2:migracion:000001', 0
    )
);
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v2:migracion:000002', 0
    )
);

DO $prevalidacion$
DECLARE
    propietario oid;
BEGIN
    SELECT oid INTO propietario
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_autorizacion_atestada_v2_propietario'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
       AND NOT rolcreaterole AND NOT rolinherit AND NOT rolreplication
       AND NOT rolbypassrls;
    IF propietario IS NULL
       OR current_user IS DISTINCT FROM
          'vec_autorizacion_atestada_v2_propietario'
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_namespace
            WHERE nspname = 'vec_autorizacion_atestada_v2'
              AND nspowner = propietario
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'base VEC-AD-2 no acreditada para reconciliacion';
    END IF;

    IF to_regprocedure(
           'vec_autorizacion_atestada_v2.identidad_runtime_valida(text,boolean)'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion_atestada_v2.texto_tecnico_valido(text,integer)'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion_atestada_v2.huella_sha256_valida(text)'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion_atestada_v2.reconciliar_consumo_decision_v2(text,text,text,text,text)'
       ) IS NULL
       OR to_regclass(
           'vec_autorizacion_atestada_v2.consumo_capacidad_v2'
       ) IS NULL
       OR to_regclass(
           'vec_autorizacion_atestada_v2.consumo_decision_v2'
       ) IS NULL
       OR to_regclass(
           'vec_autorizacion_atestada_v2.auditoria_consumo_v2'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'base VEC-AD-2 no acreditada para reconciliacion';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = funcion.pronamespace
         WHERE espacio.nspname = 'vec_autorizacion_atestada_v2'
           AND funcion.proname = 'reconciliar_consumo_efecto_v2'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42710',
            MESSAGE = 'reconciliacion de efecto V2 ya definida';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_constraint
         WHERE conrelid =
               'vec_autorizacion_atestada_v2.consumo_decision_v2'::regclass
           AND conname = 'consumo_decision_v2_efecto_ref_key'
           AND contype = 'u'
           AND pg_catalog.pg_get_constraintdef(oid, false) =
               'UNIQUE (efecto_ref)'
    ) OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_constraint
         WHERE conrelid =
               'vec_autorizacion_atestada_v2.consumo_decision_v2'::regclass
           AND conname = 'consumo_decision_v2_huella_efecto_sha256_key'
           AND contype = 'u'
           AND pg_catalog.pg_get_constraintdef(oid, false) =
               'UNIQUE (huella_efecto_sha256)'
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class
         WHERE oid IN (
             'vec_autorizacion_atestada_v2.consumo_capacidad_v2'::regclass,
             'vec_autorizacion_atestada_v2.consumo_decision_v2'::regclass,
             'vec_autorizacion_atestada_v2.auditoria_consumo_v2'::regclass
         )
           AND (relowner <> propietario
                OR NOT relrowsecurity OR NOT relforcerowsecurity)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'unicidad o confinamiento VEC-AD-2 no acreditados';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION
vec_autorizacion_atestada_v2.reconciliar_consumo_efecto_v2(
    p_efecto_ref text,
    p_huella_efecto_sha256 text
)
RETURNS TABLE (
    estado text,
    registro_ref text,
    consumo_ref text,
    auditoria_ref text,
    consumida_en timestamptz(6),
    huella_auditoria_sha256 text
)
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    identidad_valida boolean;
    numero_coincidencias bigint;
    numero_pares_exactos bigint;
    numero_recibos bigint;
BEGIN
    BEGIN
        identidad_valida :=
            vec_autorizacion_atestada_v2.identidad_runtime_valida(
                'vec_autorizacion_atestada_v2_consumidor', true
            );
    EXCEPTION WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'reconciliacion de efecto V2 indeterminada';
    END;
    IF identidad_valida IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'identidad consumidora atestada rechazada';
    END IF;

    BEGIN
        IF vec_autorizacion_atestada_v2.texto_tecnico_valido(
               p_efecto_ref, 512
           ) IS NOT TRUE
           OR vec_autorizacion_atestada_v2.huella_sha256_valida(
               p_huella_efecto_sha256
           ) IS NOT TRUE THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'reconciliacion de efecto V2 indeterminada';
        END IF;

        SELECT count(*),
               count(*) FILTER (
                   WHERE consumo.efecto_ref = p_efecto_ref
                     AND consumo.huella_efecto_sha256 =
                         p_huella_efecto_sha256
               )
          INTO numero_coincidencias, numero_pares_exactos
          FROM vec_autorizacion_atestada_v2.consumo_decision_v2 AS consumo
         WHERE consumo.efecto_ref = p_efecto_ref
            OR consumo.huella_efecto_sha256 = p_huella_efecto_sha256;

        IF numero_coincidencias = 0 AND numero_pares_exactos = 0 THEN
            RETURN QUERY SELECT 'ausente'::text, NULL::text, NULL::text,
                NULL::text, NULL::timestamptz, NULL::text;
            RETURN;
        ELSIF numero_pares_exactos = 0
              AND numero_coincidencias BETWEEN 1 AND 2 THEN
            RETURN QUERY SELECT 'colision'::text, NULL::text, NULL::text,
                NULL::text, NULL::timestamptz, NULL::text;
            RETURN;
        ELSIF numero_coincidencias <> 1 OR numero_pares_exactos <> 1 THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'reconciliacion de efecto V2 indeterminada';
        END IF;

        RETURN QUERY
        SELECT 'exacto'::text, consumo.registro_ref, consumo.consumo_ref,
               auditoria.auditoria_ref, consumo.consumida_en,
               auditoria.huella_registro_sha256
          FROM vec_autorizacion_atestada_v2.consumo_decision_v2 AS consumo
          JOIN vec_autorizacion_atestada_v2.consumo_capacidad_v2 AS capacidad
            ON capacidad.registro_ref = consumo.registro_ref
          JOIN vec_autorizacion_atestada_v2.auditoria_consumo_v2 AS auditoria
            ON auditoria.consumo_ref = consumo.consumo_ref
           AND auditoria.registro_ref = consumo.registro_ref
           AND auditoria.decision_ref = consumo.decision_ref
           AND auditoria.efecto_ref = consumo.efecto_ref
         WHERE consumo.efecto_ref = p_efecto_ref
           AND consumo.huella_efecto_sha256 = p_huella_efecto_sha256;
        GET DIAGNOSTICS numero_recibos = ROW_COUNT;
        IF numero_recibos <> 1 THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'reconciliacion de efecto V2 indeterminada';
        END IF;
        RETURN;
    EXCEPTION WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'reconciliacion de efecto V2 indeterminada';
    END;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v2.reconciliar_consumo_efecto_v2(text, text)
    FROM PUBLIC, vec_autorizacion_atestada_v2_emisor_capacidad,
         vec_autorizacion_atestada_v2_consumidor;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v2.reconciliar_consumo_efecto_v2(text, text)
    TO vec_autorizacion_atestada_v2_consumidor;

COMMIT;
