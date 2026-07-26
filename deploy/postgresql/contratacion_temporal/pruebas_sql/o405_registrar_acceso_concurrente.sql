\set ON_ERROR_STOP on
\if :{?indice}
\else
  \echo 'falta variable indice'
  \quit 3
\endif

CREATE FUNCTION pg_temp.registro_rrhh_concurrente(p_indice integer)
RETURNS jsonb
LANGUAGE sql
STABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.jsonb_build_object(
        'accion', 'contratacion_temporal.cuadro.consultar',
        'actor_ref', 'actor:concurrente:' || p_indice::text,
        'ambito_ref', 'ambito:concurrente:' || p_indice::text,
        'audiencia',
            'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
        'auditoria_vec_huella_sha256',
            pg_catalog.lpad(pg_catalog.to_hex(p_indice + 80), 64, '8'),
        'auditoria_vec_ref',
            'auditoria:vec:concurrente:' || p_indice::text,
        'capacidad_huella_sha256',
            pg_catalog.lpad(pg_catalog.to_hex(p_indice + 50), 64, '5'),
        'consumo_vec_huella_sha256',
            pg_catalog.lpad(pg_catalog.to_hex(p_indice + 60), 64, '6'),
        'correlacion_ref',
            'correlacion:concurrente:' || p_indice::text,
        'decision_huella_sha256',
            pg_catalog.lpad(pg_catalog.to_hex(p_indice + 10), 64, '1'),
        'decision_ref', 'decision:concurrente:' || p_indice::text,
        'dominio_huella_consulta',
            'vec.contratacion_temporal.consulta_rrhh.cuadro.v1',
        'expediente_ref', NULL,
        'finalidad', 'gestion_operativa_contratacion_temporal',
        'modulo_id', 'contratacion_temporal',
        'organizacion_ref',
            'organizacion:concurrente:' || p_indice::text,
        'perfil_id', 'perfil:concurrente:' || p_indice::text,
        'perfil_version', 1,
        'recurso_ref', 'ambito:concurrente:' || p_indice::text,
        'recurso_tipo', 'cuadro_rrhh_contratacion_temporal',
        'resultado_generico', 'entregado',
        'resultado_huella_sha256',
            pg_catalog.lpad(pg_catalog.to_hex(p_indice + 70), 64, '7'),
        'sesion_huella_sha256',
            pg_catalog.lpad(pg_catalog.to_hex(p_indice + 20), 64, '2'),
        'sesion_id', 'sesion:concurrente:' || p_indice::text,
        'tipo_consulta', 'cuadro',
        'total', 0,
        'version_expediente', NULL,
        'consulta_huella_sha256',
            pg_catalog.lpad(pg_catalog.to_hex(p_indice + 40), 64, '4')
    )
$funcion$;

BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SELECT vec_contratacion_temporal.registrar_acceso_rrhh_interno_v1(
    pg_temp.registro_rrhh_concurrente(:indice)
);
COMMIT;
