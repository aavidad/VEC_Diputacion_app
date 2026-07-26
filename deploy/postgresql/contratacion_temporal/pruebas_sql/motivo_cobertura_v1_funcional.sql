\set ON_ERROR_STOP 1

BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SET LOCAL ROLE vec_autorizacion_motivos_proyector;
DO $publicacion$
DECLARE
    v_publicado timestamptz(6) :=
        pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
        - interval '2 hours';
    v_desde timestamptz(6) := v_publicado - interval '1 hour';
    v_hasta timestamptz(6) := v_publicado + interval '1 hour';
    v_entradas jsonb;
    v_replay jsonb;
BEGIN
    v_entradas := pg_catalog.jsonb_build_array(
        pg_catalog.jsonb_build_object(
            'clave', 'rectificacion_decision',
            'clave_i18n', 'cobertura.motivo.rectificacion',
            'vigente_desde', pg_catalog.to_char(
                v_desde AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
            'vigente_hasta', NULL
        ),
        pg_catalog.jsonb_build_object(
            'clave', 'desviacion_recomendacion',
            'clave_i18n', 'cobertura.motivo.desviacion',
            'vigente_desde', pg_catalog.to_char(
                v_desde AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
            'vigente_hasta', pg_catalog.to_char(
                v_hasta AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            )
        )
    );
    IF vec_autorizacion.publicar_motivos_cobertura_v1(
        'evento_ffffffffffffffffffffffffffffffff', 9007199254740992,
        pg_catalog.repeat('f', 64), 'motivos_cobertura', 1,
        pg_catalog.repeat('a', 64), 'contratacion_temporal',
        v_publicado, v_entradas
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'se aceptó una secuencia fuera del entero seguro';
    END IF;
    IF vec_autorizacion.publicar_motivos_cobertura_v1(
        'evento_00000000000000000000000000000009', 9,
        pg_catalog.repeat('9', 64), 'motivos_cobertura', 1,
        pg_catalog.repeat('a', 64), 'contratacion_temporal',
        v_publicado, v_entradas
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'se aceptó un salto de secuencia';
    END IF;
    IF vec_autorizacion.publicar_motivos_cobertura_v1(
        'evento_00000000000000000000000000000001', 1,
        pg_catalog.repeat('1', 64), 'motivos_cobertura', 2,
        pg_catalog.repeat('b', 64), 'contratacion_temporal',
        v_publicado, v_entradas
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'se aceptó una versión sin predecesora';
    END IF;
    IF vec_autorizacion.publicar_motivos_cobertura_v1(
        'evento_00000000000000000000000000000001', 1,
        pg_catalog.repeat('1', 64), 'motivos_cobertura', 1,
        pg_catalog.repeat('a', 64), 'otro_modulo',
        v_publicado, v_entradas
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'se aceptó un módulo ajeno';
    END IF;
    IF vec_autorizacion.publicar_motivos_cobertura_v1(
        'evento_00000000000000000000000000000001', 1,
        pg_catalog.repeat('1', 64), 'motivos_cobertura', 1,
        pg_catalog.repeat('a', 64), 'contratacion_temporal',
        v_publicado,
        pg_catalog.jsonb_build_array(v_entradas -> 0, v_entradas -> 0)
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'se aceptaron entradas duplicadas';
    END IF;
    IF vec_autorizacion.publicar_motivos_cobertura_v1(
        'evento_00000000000000000000000000000001', 1,
        pg_catalog.repeat('1', 64), 'motivos_cobertura', 1,
        pg_catalog.repeat('a', 64), 'contratacion_temporal',
        v_publicado, (v_entradas #- '{0,clave_i18n}')
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'se aceptó una entrada sin clave i18n';
    END IF;
    IF vec_autorizacion.publicar_motivos_cobertura_v1(
        'evento_00000000000000000000000000000001', 1,
        pg_catalog.repeat('1', 64), 'motivos_cobertura', 1,
        pg_catalog.repeat('a', 64), 'contratacion_temporal',
        v_publicado, v_entradas
    ) IS NOT TRUE THEN
        RAISE EXCEPTION 'falló la publicación válida';
    END IF;
    v_replay := pg_catalog.jsonb_build_array(v_entradas -> 1, v_entradas -> 0);
    IF vec_autorizacion.publicar_motivos_cobertura_v1(
        'evento_00000000000000000000000000000001', 1,
        pg_catalog.repeat('1', 64), 'motivos_cobertura', 1,
        pg_catalog.repeat('a', 64), 'contratacion_temporal',
        v_publicado, v_replay
    ) IS NOT TRUE THEN
        RAISE EXCEPTION 'el replay exacto no fue idempotente';
    END IF;
    v_replay := pg_catalog.jsonb_set(
        v_entradas, '{0,clave_i18n}',
        '"cobertura.motivo.mutado"'::jsonb
    );
    IF vec_autorizacion.publicar_motivos_cobertura_v1(
        'evento_00000000000000000000000000000001', 1,
        pg_catalog.repeat('1', 64), 'motivos_cobertura', 1,
        pg_catalog.repeat('a', 64), 'contratacion_temporal',
        v_publicado, v_replay
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'un replay alterado fue aceptado';
    END IF;
    IF vec_autorizacion.publicar_motivos_cobertura_v1(
        'evento_99999999999999999999999999999999', 1,
        pg_catalog.repeat('9', 64), 'motivos_cobertura', 1,
        pg_catalog.repeat('a', 64), 'contratacion_temporal',
        v_publicado, v_entradas
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'una secuencia se reutilizó con otro evento';
    END IF;
END
$publicacion$;

RESET ROLE;
SET LOCAL ROLE vec_autorizacion_motivos_evaluador;
DO $historico$
DECLARE
    v_instante timestamptz(6) :=
        pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
        - interval '30 minutes';
BEGIN
    IF vec_autorizacion.resolver_motivo_cobertura_historico_v1(
        'motivos_cobertura', 1, pg_catalog.repeat('a', 64),
        'rectificacion_decision', 'cobertura.motivo.rectificacion',
        v_instante
    ) IS NOT TRUE
       OR vec_autorizacion.resolver_motivo_cobertura_historico_v1(
           'motivos_cobertura', 1, pg_catalog.repeat('a', 64),
           'rectificacion_decision', 'cobertura.motivo.mutado',
           v_instante
       ) IS NOT FALSE
       OR vec_autorizacion.resolver_motivo_cobertura_historico_v1(
           'motivos_cobertura', 1, pg_catalog.repeat('b', 64),
           'rectificacion_decision', 'cobertura.motivo.rectificacion',
           v_instante
       ) IS NOT FALSE THEN
        RAISE EXCEPTION 'la resolución histórica no fue exacta';
    END IF;
END
$historico$;

RESET ROLE;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $actual$
BEGIN
    IF vec_autorizacion.resolver_motivo_cobertura_actual_v1(
        'motivos_cobertura', 1, pg_catalog.repeat('a', 64),
        'rectificacion_decision', 'cobertura.motivo.rectificacion'
    ) IS NOT TRUE
       OR vec_autorizacion.resolver_motivo_cobertura_actual_v1(
           'motivos_cobertura', 1, pg_catalog.repeat('a', 64),
           'rectificacion_decision', 'cobertura.motivo.mutado'
       ) IS NOT FALSE THEN
        RAISE EXCEPTION 'la barrera actual no fue exacta';
    END IF;
END
$actual$;

RESET ROLE;
SET LOCAL ROLE vec_autorizacion_motivos_proyector;
DO $retirada$
DECLARE
    v_retirado timestamptz(6) :=
        pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp());
BEGIN
    IF vec_autorizacion.retirar_motivos_cobertura_v1(
        'evento_00000000000000000000000000000002', 2,
        pg_catalog.repeat('2', 64), 'motivos_cobertura', 1,
        pg_catalog.repeat('a', 64), pg_catalog.repeat('c', 64),
        'contratacion_temporal', v_retirado
    ) IS NOT TRUE
       OR vec_autorizacion.retirar_motivos_cobertura_v1(
           'evento_00000000000000000000000000000002', 2,
           pg_catalog.repeat('2', 64), 'motivos_cobertura', 1,
           pg_catalog.repeat('a', 64), pg_catalog.repeat('c', 64),
           'contratacion_temporal', v_retirado
       ) IS NOT TRUE
       OR vec_autorizacion.retirar_motivos_cobertura_v1(
           'evento_00000000000000000000000000000002', 2,
           pg_catalog.repeat('2', 64), 'motivos_cobertura', 1,
           pg_catalog.repeat('a', 64), pg_catalog.repeat('d', 64),
           'contratacion_temporal', v_retirado
       ) IS NOT FALSE THEN
        RAISE EXCEPTION 'retirada o replay de retirada incorrectos';
    END IF;
END
$retirada$;

RESET ROLE;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $retirada_actual$
BEGIN
    IF vec_autorizacion.resolver_motivo_cobertura_actual_v1(
        'motivos_cobertura', 1, pg_catalog.repeat('a', 64),
        'rectificacion_decision', 'cobertura.motivo.rectificacion'
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'la retirada no invalidó la barrera actual';
    END IF;
END
$retirada_actual$;

RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $inventario$
BEGIN
    IF (
        SELECT ultima_secuencia
          FROM vec_autorizacion.motivo_cobertura_v1_checkpoint_origen
         WHERE control_id
    ) <> 2
       OR (
           SELECT pg_catalog.count(*)
             FROM vec_autorizacion.motivo_cobertura_v1_evento_origen
       ) <> 2
       OR (
           SELECT pg_catalog.count(*)
             FROM vec_autorizacion.motivo_cobertura_v1_catalogo_publicado
       ) <> 1
       OR (
           SELECT pg_catalog.count(*)
             FROM vec_autorizacion.motivo_cobertura_v1_entrada
       ) <> 2
       OR (
           SELECT pg_catalog.count(*)
             FROM vec_autorizacion.motivo_cobertura_v1_retirada
       ) <> 1 THEN
        RAISE EXCEPTION 'la proyección dejó un inventario incoherente';
    END IF;
END
$inventario$;

ROLLBACK;
