BEGIN;
SET LOCAL ROLE vec_confianza_atestacion_v2_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

DO $prevalidacion$
BEGIN
    IF to_regnamespace('vec_autorizacion_atestada_v2') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retire primero el registro atestado V2';
    END IF;
END
$prevalidacion$;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_confianza_atestacion_v2:gobierno:v1', 0
    )
);
LOCK TABLE
    vec_confianza_atestacion_v2_consumo_atestado.checkpoint_gobierno
    IN ACCESS EXCLUSIVE MODE;

REVOKE ALL ON FUNCTION
    vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_v1(
        text, text, timestamptz, timestamptz, text, numeric, text,
        timestamptz, timestamptz, text, text, timestamptz
    ) FROM PUBLIC, vec_confianza_atestacion_v2_lector_autoridad,
        vec_autorizacion_atestada_v2_propietario;
REVOKE ALL ON FUNCTION
    vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_en_v1(
        text, text, timestamptz, timestamptz, text, numeric, text,
        timestamptz, timestamptz, text, text, timestamptz, timestamptz
    ) FROM PUBLIC, vec_confianza_atestacion_v2_lector_autoridad,
        vec_autorizacion_atestada_v2_propietario;
DROP FUNCTION
    vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_v1(
        text, text, timestamptz, timestamptz, text, numeric, text,
        timestamptz, timestamptz, text, text, timestamptz
    );
DROP FUNCTION
    vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_en_v1(
        text, text, timestamptz, timestamptz, text, numeric, text,
        timestamptz, timestamptz, text, text, timestamptz, timestamptz
    );

DO $retirar_triggers_checkpoint$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'acto_gobierno', 'configuracion_confianza_version',
        'raiz_confianza_version',
        'configuracion_raiz', 'revocacion_configuracion',
        'revocacion_raiz', 'puntero_raiz_actual',
        'puntero_configuracion_actual'
    ] LOOP
        EXECUTE format(
            'DROP TRIGGER a05_sellar_conocimiento_consumo_atestado ON vec_confianza_atestacion_v2.%I',
            tabla
        );
        EXECUTE format(
            'DROP TRIGGER z90_avanzar_checkpoint_consumo_atestado ON vec_confianza_atestacion_v2.%I',
            tabla
        );
    END LOOP;
END
$retirar_triggers_checkpoint$;

DROP TABLE
    vec_confianza_atestacion_v2_consumo_atestado.checkpoint_gobierno;
DROP FUNCTION
    vec_confianza_atestacion_v2_consumo_atestado.sellar_conocimiento_gobierno();
DROP FUNCTION
    vec_confianza_atestacion_v2_consumo_atestado.avanzar_checkpoint_gobierno();
DROP FUNCTION
    vec_confianza_atestacion_v2_consumo_atestado.rechazar_retirada_checkpoint();
DROP SCHEMA vec_confianza_atestacion_v2_consumo_atestado;

REVOKE REFERENCES (
    configuracion_revision, clave_id, version
) ON vec_confianza_atestacion_v2.configuracion_raiz
  FROM vec_autorizacion_atestada_v2_propietario;
REVOKE REFERENCES (
    clave_id, version, clave_publica_spki, huella_clave_spki_sha256,
    valida_desde, valida_hasta, suite, audiencia_despliegue
) ON vec_confianza_atestacion_v2.raiz_confianza_version
  FROM vec_autorizacion_atestada_v2_propietario;
REVOKE REFERENCES (
    revision, huella_configuracion_sha256, publicada_en, expira_en
) ON vec_confianza_atestacion_v2.configuracion_confianza_version
  FROM vec_autorizacion_atestada_v2_propietario;
ALTER TABLE vec_confianza_atestacion_v2.raiz_confianza_version
    DROP CONSTRAINT raiz_confianza_v2_datos_atestados_unicos;
ALTER TABLE vec_confianza_atestacion_v2.configuracion_confianza_version
    DROP CONSTRAINT configuracion_confianza_v2_datos_atestados_unicos;
COMMIT;
