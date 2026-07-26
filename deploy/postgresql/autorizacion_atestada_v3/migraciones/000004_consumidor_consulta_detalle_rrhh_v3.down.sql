BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v3:migracion:000004', 0
    )
);

DO $proteccion$
DECLARE
    v_hay_historia boolean;
    v_definicion text;
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_detalle_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_cuadro_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'consulta detalle RRHH VEC-AD-3 no instalada';
    END IF;
    SELECT pg_catalog.regexp_replace(
               pg_catalog.pg_get_constraintdef(c.oid, true),
               '\s+', ' ', 'g'
           )
      INTO v_definicion
      FROM pg_catalog.pg_constraint c
     WHERE c.conrelid =
           'vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND c.conname =
           'clave_capacidad_version_audiencia_consumo_check'
       AND c.contype = 'c'
       AND c.convalidated
       AND c.conkey = ARRAY[8]::smallint[];
    IF v_definicion IS NULL
       OR v_definicion <>
          'CHECK (audiencia_consumo = ANY (ARRAY[''vec_contratacion_temporal.confirmar_alta_atestada.v1''::text, ''vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1''::text, ''vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1''::text]))' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'gobierno de audiencias de detalle inesperado';
    END IF;
    SELECT
        EXISTS (
            SELECT 1
              FROM vec_autorizacion_atestada_v3.clave_capacidad_version k
             WHERE k.audiencia_consumo =
                   'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
        )
        OR EXISTS (
            SELECT 1
              FROM vec_autorizacion_atestada_v3.puntero_clave_emision p
              JOIN vec_autorizacion_atestada_v3.clave_capacidad_version k
                USING (clave_id, version)
             WHERE k.audiencia_consumo =
                   'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
        )
        OR EXISTS (
            SELECT 1
              FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 a
             WHERE pg_catalog.convert_from(
                       a.capacidad_canonica, 'UTF8'
                   )::jsonb ->> 'audiencia_consumo' =
                   'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
        )
        OR EXISTS (
            SELECT 1
              FROM vec_autorizacion_atestada_v3.consumo_decision_v3 c
              JOIN vec_autorizacion_atestada_v3.atestacion_decision_v3 a
                USING (decision_ref)
             WHERE pg_catalog.convert_from(
                       a.capacidad_canonica, 'UTF8'
                   )::jsonb ->> 'audiencia_consumo' =
                   'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
        )
      INTO v_hay_historia;
    IF v_hay_historia THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'reversión protegida: existe historia de detalle RRHH';
    END IF;
END
$proteccion$;

ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
    DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
    ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check
    CHECK (audiencia_consumo IN (
        'vec_contratacion_temporal.confirmar_alta_atestada.v1',
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'
    ));

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3
    .registrar_y_consumir_consulta_detalle_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    ) FROM PUBLIC, vec_autorizacion_atestada_v3_consumidor,
           vec_autorizacion_atestada_v3_emisor,
           vec_contratacion_temporal_propietario;
DROP FUNCTION
    vec_autorizacion_atestada_v3
    .registrar_y_consumir_consulta_detalle_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    );
COMMIT;
