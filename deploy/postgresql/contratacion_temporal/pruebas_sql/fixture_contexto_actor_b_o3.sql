\set ON_ERROR_STOP 1

-- Segunda identidad enteramente sintética para probar segregación de
-- funciones en la rectificación O3. Se carga antes de instalar consumidores,
-- cuando el LOGIN mínimo de ContextoActor conserva su acreditación cerrada.
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $identidad_b$
DECLARE
    v_desde timestamptz(6) := pg_catalog.clock_timestamp() -
        interval '1 hour';
    v_hasta timestamptz(6) := pg_catalog.clock_timestamp() +
        interval '1 hour';
BEGIN
    INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES (
        'cta_o3b_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 2,
        'prc_maestra_sintetica_v3_00000001', 1,
        pg_catalog.repeat('a', 64), 'autoridad_maestra_acreditada',
        'activo', v_desde, v_hasta
    );
    INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES (
        'cta_o3b_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 2
    );
    INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES (
        'per_o3b_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbb', 2,
        'prc_maestra_sintetica_v3_00000001', 1,
        pg_catalog.repeat('a', 64), 'autoridad_maestra_acreditada',
        'activo', v_desde, v_hasta
    );
    INSERT INTO vec_contexto_actor_v1.persona_actual VALUES (
        'per_o3b_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbb', 2
    );
    INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES (
        'prf_o3b_cccccccccccccccccccccccccccc', 2,
        'per_o3b_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
        'prc_maestra_sintetica_v3_00000001', 1,
        pg_catalog.repeat('a', 64), 'autoridad_maestra_acreditada',
        'activo', v_desde, v_hasta
    );
    INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES (
        'prf_o3b_cccccccccccccccccccccccccccc', 2
    );
    INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES (
        'vca_o3b_dddddddddddddddddddddddddddd', 2,
        'cta_o3b_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
        'prf_o3b_cccccccccccccccccccccccccccc',
        'per_o3b_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
        'prc_maestra_sintetica_v3_00000001', 1,
        pg_catalog.repeat('a', 64), 'autoridad_maestra_acreditada',
        'activo', v_desde, v_hasta
    );
    INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES (
        'vca_o3b_dddddddddddddddddddddddddddd', 2
    );
    INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones VALUES (
        'vin_o3b_eeeeeeeeeeeeeeeeeeeeeeeeeeeeee', 2,
        'per_o3b_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbb', 'empleado',
        'emp_o3b_ffffffffffffffffffffffffffffff',
        'prc_maestra_sintetica_v3_00000001', 1,
        pg_catalog.repeat('a', 64), 'autoridad_maestra_acreditada',
        'activo', v_desde, v_hasta
    );
    INSERT INTO vec_contexto_actor_v1.vinculo_referencia_actual VALUES (
        'vin_o3b_eeeeeeeeeeeeeeeeeeeeeeeeeeeeee', 2
    );
END
$identidad_b$;

COMMIT;
