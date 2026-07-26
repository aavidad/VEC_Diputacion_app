\set ON_ERROR_STOP on

CREATE FUNCTION pg_temp.registro_rrhh_v1_prueba(p_indice integer)
RETURNS jsonb
LANGUAGE sql
STABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.jsonb_build_object(
        'accion', 'contratacion_temporal.cuadro.consultar',
        'actor_ref', 'actor:rrhh:' || p_indice::text,
        'ambito_ref', 'organizacion:rrhh:' || p_indice::text,
        'audiencia',
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
        'auditoria_vec_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 80), 64, '8'),
        'auditoria_vec_ref', 'auditoria:vec:rrhh:' || p_indice::text,
        'capacidad_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 50), 64, '5'),
        'consulta_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 40), 64, '4'),
        'consumo_vec_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 60), 64, '6'),
        'correlacion_ref', 'correlacion:rrhh:' || p_indice::text,
        'decision_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 10), 64, '1'),
        'decision_ref', 'decision:rrhh:' || p_indice::text,
        'dominio_huella_consulta',
        'vec.contratacion_temporal.consulta_rrhh.cuadro.v1',
        'expediente_ref', NULL,
        'finalidad', 'gestion_operativa_contratacion_temporal',
        'modulo_id', 'contratacion_temporal',
        'organizacion_ref', 'organizacion:rrhh:' || p_indice::text,
        'perfil_id', 'perfil:rrhh:' || p_indice::text,
        'perfil_version', 1,
        'recurso_ref', 'organizacion:rrhh:' || p_indice::text,
        'recurso_tipo', 'cuadro_rrhh_contratacion_temporal',
        'resultado_generico', 'entregado',
        'resultado_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 70), 64, '7'),
        'sesion_huella_sha256',
        pg_catalog.lpad(pg_catalog.to_hex(p_indice + 20), 64, '2'),
        'sesion_id', 'sesion:rrhh:' || p_indice::text,
        'tipo_consulta', 'cuadro', 'total', 1,
        'version_expediente', NULL
    )
$funcion$;

SET ROLE vec_contratacion_temporal_propietario;

DO $v1_cerrado$
BEGIN
    BEGIN
        PERFORM
            vec_contratacion_temporal
                .registrar_acceso_rrhh_interno_v1(
                    pg_temp.registro_rrhh_v1_prueba(9001)
                );
        RAISE EXCEPTION 'v1 aceptó una barrera posterior';
    EXCEPTION WHEN SQLSTATE '55000' THEN
        NULL;
    END;
END
$v1_cerrado$;

\if :antes
DO $antes$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 18
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 2
    ) THEN
        RAISE EXCEPTION 'barrera anterior a 000039 incorrecta';
    END IF;
END
$antes$;
\else
DO $estructura$
DECLARE
    tabla text;
    plan json;
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 19
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 3
    ) OR (
        SELECT pg_catalog.count(*)
          FROM vec_contratacion_temporal
               .control_registrador_acceso_rrhh_v2
         WHERE control AND version_esquema = 1
           AND prueba_huella_sha256 = pg_catalog.encode(
               pg_catalog.sha256(prueba_canonica), 'hex'
           )
    ) <> 1 THEN
        RAISE EXCEPTION 'barreras o baseline v2 incorrectos';
    END IF;

    FOREACH tabla IN ARRAY ARRAY[
        'control_registrador_acceso_rrhh_v2',
        'vinculo_identidad_acceso_rrhh_v2'
    ]::text[] LOOP
        IF NOT EXISTS (
            SELECT 1
              FROM pg_catalog.pg_class c
              JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
             WHERE n.nspname = 'vec_contratacion_temporal'
               AND c.relname = tabla
               AND c.relowner =
                   'vec_contratacion_temporal_propietario'::regrole
               AND c.relrowsecurity AND c.relforcerowsecurity
        ) OR pg_catalog.has_table_privilege(
            'vec_contratacion_temporal_consultor_rrhh',
            'vec_contratacion_temporal.' || tabla,
            'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
        ) THEN
            RAISE EXCEPTION 'RLS/ACL incorrecta en %', tabla;
        END IF;
    END LOOP;
    IF pg_catalog.has_function_privilege(
        'vec_contratacion_temporal_consultor_rrhh',
        'vec_contratacion_temporal.'
        || 'registrar_acceso_rrhh_interno_v2(jsonb)',
        'EXECUTE'
    ) OR pg_catalog.has_function_privilege(
        'public',
        'vec_contratacion_temporal.'
        || 'registrar_acceso_rrhh_interno_v2(jsonb)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'registrador v2 expuesto fuera del propietario';
    END IF;

    SET LOCAL enable_seqscan = off;
    EXECUTE $sql$
        EXPLAIN (FORMAT JSON, COSTS OFF)
        SELECT DISTINCT ON (expediente_ref COLLATE "C")
               expediente_ref, version, corte_global
          FROM vec_contratacion_temporal.publicacion_version_rrhh
         WHERE organizacion_ref COLLATE "C" =
               'organizacion:diputacion:granada' COLLATE "C"
           AND corte_global <= 9007199254740991::numeric
         ORDER BY expediente_ref COLLATE "C", corte_global DESC
    $sql$ INTO plan;
    IF plan::text NOT LIKE
       '%publicacion_rrhh_organizacion_expediente_corte_desc_idx%' THEN
        RAISE EXCEPTION 'el índice as-of no resuelve el plan esperado: %',
            plan;
    END IF;

    BEGIN
        UPDATE vec_contratacion_temporal
               .control_registrador_acceso_rrhh_v2
           SET version_esquema = 1
         WHERE control;
        RAISE EXCEPTION 'baseline mutable';
    EXCEPTION WHEN SQLSTATE '55000' THEN
        NULL;
    END;
    BEGIN
        TRUNCATE vec_contratacion_temporal
                 .control_registrador_acceso_rrhh_v2;
        RAISE EXCEPTION 'baseline truncable';
    EXCEPTION WHEN SQLSTATE '55000' THEN
        NULL;
    END;
END
$estructura$;
\endif

RESET ROLE;
