-- Retirada segura: nunca elimina evidencia V3. No usa CASCADE.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion:migracion:registro_contexto_actor_v3:000005', 0
    )
);

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'retirada V3 requiere superusuario';
    END IF;
END
$prevalidacion$;

SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;

DO $conservacion$
DECLARE
    tabla regclass;
    tiene_filas boolean;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        pg_catalog.to_regclass(
            'vec_autorizacion.decision_concedida_contexto_actor_v3'
        ),
        pg_catalog.to_regclass(
            'vec_autorizacion.decision_denegada_contexto_actor_v3'
        )
    ] LOOP
        IF tabla IS NOT NULL THEN
            EXECUTE pg_catalog.format(
                'LOCK TABLE %s IN ACCESS EXCLUSIVE MODE', tabla
            );
            EXECUTE pg_catalog.format(
                'SELECT EXISTS (SELECT 1 FROM %s)', tabla
            ) INTO tiene_filas;
            IF tiene_filas THEN
                RAISE EXCEPTION USING ERRCODE = '55000',
                    MESSAGE = 'retirada rechazada: existen decisiones V3 durables';
            END IF;
        END IF;
    END LOOP;
END
$conservacion$;

DROP FUNCTION IF EXISTS vec_autorizacion.revalidar_sesion_vinculo_v2(
    jsonb, timestamptz, timestamptz, timestamptz
);
DROP TABLE IF EXISTS
    vec_autorizacion.decision_concedida_contexto_actor_v3;
DROP TABLE IF EXISTS
    vec_autorizacion.decision_denegada_contexto_actor_v3;
DROP FUNCTION IF EXISTS vec_autorizacion.decision_contexto_actor_v3_valida(
    jsonb
);
DROP FUNCTION IF EXISTS vec_autorizacion.vinculo_contexto_actor_v2_canonico(
    jsonb
);
DROP FUNCTION IF EXISTS vec_autorizacion.lista_textos_v3_canonica(jsonb);
DROP FUNCTION IF EXISTS vec_autorizacion.manifiesto_politicas_v3_canonico(
    jsonb
);
DROP FUNCTION IF EXISTS vec_autorizacion.texto_json_go_v3(text);
DROP FUNCTION IF EXISTS vec_autorizacion.texto_ascii_visible_v3_valido(
    text, integer
);
DROP FUNCTION IF EXISTS vec_autorizacion.vinculo_contexto_actor_v2_valido(
    jsonb
);

RESET ROLE;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;
REVOKE EXECUTE ON FUNCTION
    vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
        text, text, text, text, text, text, numeric, text, numeric,
        text, numeric, text, numeric, text, text, timestamptz, timestamptz
    ) FROM vec_autorizacion_propietario;
REVOKE USAGE ON SCHEMA vec_contexto_actor_v1
    FROM vec_autorizacion_propietario;

COMMIT;
