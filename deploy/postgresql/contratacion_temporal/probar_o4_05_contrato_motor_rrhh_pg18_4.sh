#!/usr/bin/env bash
set -Eeuo pipefail

directorio="$(
    cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1
    pwd -P
)"

# Reutiliza el montaje, la imagen fijada y el estado 000039 ya acreditado.
# Al cargarse, el runner anterior conserva su trap de limpieza hasta que este
# proceso termine; el contenedor nunca se publica ni abandona la red aislada.
# shellcheck disable=SC1091
source \
    "$directorio/probar_o4_05_registrador_acceso_rrhh_v2_pg18_4.sh"

paso() {
    printf '[O4-05:C2-D2:000040:PG18.4] %s\n' "$1"
}

estado_000040() {
    valor "SELECT pg_catalog.concat_ws('|',
        cobertura.version_esquema::text,
        consultas.version_esquema::text,
        (pg_catalog.to_regclass(
            'vec_contratacion_temporal.control_motor_consultas_rrhh_v1'
        ) IS NOT NULL)::text,
        (pg_catalog.to_regtype(
            'vec_contratacion_temporal.alcance_consulta_rrhh_v1'
        ) IS NOT NULL)::text,
        (pg_catalog.to_regtype(
            'vec_contratacion_temporal.consulta_cuadro_rrhh_v1'
        ) IS NOT NULL)::text,
        (pg_catalog.to_regtype(
            'vec_contratacion_temporal.consulta_detalle_rrhh_v1'
        ) IS NOT NULL)::text,
        (pg_catalog.to_regtype(
            'vec_contratacion_temporal.evidencia_resultado_rrhh_v1'
        ) IS NOT NULL)::text,
        (SELECT catalogo_huella_sha256
           FROM vec_contratacion_temporal.control_motor_consultas_rrhh_v1
          WHERE control)
    )
      FROM vec_contratacion_temporal.control_migracion_cobertura_o4 cobertura
      CROSS JOIN
           vec_contratacion_temporal.control_migracion_consultas_rrhh consultas
     WHERE cobertura.control AND consultas.control"
}

comprobar_estado_000040() {
    local esperado=$1
    local contexto=$2
    local obtenido
    obtenido="$(estado_000040)"
    if [[ $obtenido != "$esperado" ]]; then
        printf 'estado 000040 alterado tras %s\n' "$contexto" >&2
        printf 'esperado=%s\nobtenido=%s\n' "$esperado" "$obtenido" >&2
        return 1
    fi
}

paso 'instalación y matriz estructural/canónica'
archivo \
    contratacion_temporal/migraciones/000040_contrato_tipado_consultas_rrhh.up.sql
archivo \
    contratacion_temporal/pruebas_sql/o405_contrato_motor_rrhh_canones.sql
estado_limpio="$(estado_000040)"
if [[ $estado_limpio != '20|4|true|true|true|true|true|'* ]]; then
    printf 'estado inicial 000040 inesperado: %s\n' "$estado_limpio" >&2
    exit 1
fi

paso 'reentrada y deriva se rechazan sin efectos parciales'
esperar_fallo 'segunda instalación 000040' 55000 \
    'estado incompatible para motor de consultas RRHH' archivo \
    contratacion_temporal/migraciones/000040_contrato_tipado_consultas_rrhh.up.sql
comprobar_estado_000040 "$estado_limpio" 'segunda instalación'
psql_admin --command \
    'GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.json_rrhh_seguro_v1(jsonb) TO vec_contratacion_temporal_consultor_rrhh' \
    >/dev/null
esperar_fallo 'down 000040 con ACL derivada' 55000 \
    'catálogo derivado impide revertir contrato RRHH' archivo \
    contratacion_temporal/migraciones/000040_contrato_tipado_consultas_rrhh.down.sql
psql_admin --command \
    'REVOKE EXECUTE ON FUNCTION vec_contratacion_temporal.json_rrhh_seguro_v1(jsonb) FROM vec_contratacion_temporal_consultor_rrhh' \
    >/dev/null
comprobar_estado_000040 "$estado_limpio" 'ACL derivada restaurada'

paso 'dependencia futura queda protegida por RESTRICT'
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
CREATE FUNCTION vec_contratacion_temporal.dependencia_ct000040_prueba(
    p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1
)
RETURNS integer
LANGUAGE sql IMMUTABLE STRICT
SET search_path = pg_catalog
RETURN 1;
SQL
esperar_fallo 'down 000040 con dependencia futura' 2BP01 \
    'cannot drop type vec_contratacion_temporal.consulta_detalle_rrhh_v1' \
    archivo \
    contratacion_temporal/migraciones/000040_contrato_tipado_consultas_rrhh.down.sql
comprobar_estado_000040 "$estado_limpio" 'dependencia futura'
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
DROP FUNCTION vec_contratacion_temporal.dependencia_ct000040_prueba(
    vec_contratacion_temporal.consulta_detalle_rrhh_v1
);
SQL

paso 'barreras posteriores impiden la retirada'
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 21
 WHERE control AND version_esquema = 20;
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 5
 WHERE control AND version_esquema = 4;
SQL
esperar_fallo 'down 000040 con barrera futura' 55000 \
    'estado incompatible para revertir contrato RRHH' archivo \
    contratacion_temporal/migraciones/000040_contrato_tipado_consultas_rrhh.down.sql
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 20
 WHERE control AND version_esquema = 21;
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 4
 WHERE control AND version_esquema = 5;
SQL
comprobar_estado_000040 "$estado_limpio" 'barreras futuras restauradas'

paso 'ciclo limpio de retirada y reinstalación'
archivo \
    contratacion_temporal/migraciones/000040_contrato_tipado_consultas_rrhh.down.sql
estado_retirado="$(valor "SELECT pg_catalog.concat_ws('|',
    cobertura.version_esquema::text,
    consultas.version_esquema::text,
    (pg_catalog.to_regclass(
        'vec_contratacion_temporal.control_motor_consultas_rrhh_v1'
    ) IS NULL)::text,
    (pg_catalog.to_regtype(
        'vec_contratacion_temporal.consulta_cuadro_rrhh_v1'
    ) IS NULL)::text
) FROM vec_contratacion_temporal.control_migracion_cobertura_o4 cobertura
CROSS JOIN
    vec_contratacion_temporal.control_migracion_consultas_rrhh consultas
WHERE cobertura.control AND consultas.control")"
if [[ $estado_retirado != '19|3|true|true' ]]; then
    printf 'down 000040 incompleto: %s\n' "$estado_retirado" >&2
    exit 1
fi
archivo \
    contratacion_temporal/migraciones/000040_contrato_tipado_consultas_rrhh.up.sql
archivo \
    contratacion_temporal/pruebas_sql/o405_contrato_motor_rrhh_canones.sql
comprobar_estado_000040 "$estado_limpio" 'segundo ciclo'

paso 'contrato tipado y reversión CT-000040 superados'
