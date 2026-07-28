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

probar_base_con_huella_incompatible() {
    psql_admin <<'SQL'
BEGIN;
SET ROLE vec_contratacion_temporal_propietario;
DO $fila_previa$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.registro_acceso_rrhh acceso
         WHERE NOT EXISTS (
             SELECT 1
               FROM vec_contratacion_temporal
                    .vinculo_identidad_acceso_rrhh_v2 vinculo
              WHERE vinculo.acceso_ref = acceso.acceso_ref
         )
    ) THEN
        RAISE EXCEPTION 'fixture v1 sin vínculo no disponible';
    END IF;
END
$fila_previa$;
WITH base AS (
    SELECT acceso.*
      FROM vec_contratacion_temporal.registro_acceso_rrhh acceso
     WHERE NOT EXISTS (
         SELECT 1
           FROM vec_contratacion_temporal
                .vinculo_identidad_acceso_rrhh_v2 vinculo
          WHERE vinculo.acceso_ref = acceso.acceso_ref
     )
     ORDER BY acceso.secuencia
     LIMIT 1
), datos AS (
    SELECT base.acceso_ref,
           'cta_' || pg_catalog.repeat('i', 22) AS cuenta_ref,
           'autenticacion:incompatible'::text AS autenticacion_ref,
           pg_catalog.repeat('a', 64) AS autenticacion_huella_sha256,
           base.sesion_id AS sesion_ref,
           'control:incompatible'::text AS control_sesion_ref,
           1::numeric AS control_sesion_revision,
           CASE WHEN base.sesion_huella_sha256 =
                          pg_catalog.repeat('f', 64)
                THEN pg_catalog.repeat('e', 64)
                ELSE pg_catalog.repeat('f', 64)
            END AS control_sesion_huella_sha256,
           base.actor_ref, base.perfil_id, base.perfil_version,
           base.organizacion_ref,
           CASE WHEN base.ambito_ref = base.organizacion_ref
                THEN 'organizacion' ELSE 'centro' END AS clase_ambito,
           base.ambito_ref, base.sesion_huella_sha256,
           base.registrada_en
      FROM base
), prueba AS (
    SELECT datos.*, pg_catalog.convert_to(
               'VEC-CT-VINCULO-IDENTIDAD-ACCESO-RRHH-V2'
               || pg_catalog.chr(10), 'UTF8'
           )
           || vec_contratacion_temporal.encuadrar_texto_v1(acceso_ref)
           || vec_contratacion_temporal.encuadrar_texto_v1(cuenta_ref)
           || vec_contratacion_temporal.encuadrar_texto_v1(cuenta_ref)
           || vec_contratacion_temporal.encuadrar_texto_v1(
                  autenticacion_ref)
           || vec_contratacion_temporal.encuadrar_texto_v1(
                  autenticacion_huella_sha256)
           || vec_contratacion_temporal.encuadrar_texto_v1(sesion_ref)
           || vec_contratacion_temporal.encuadrar_texto_v1(
                  control_sesion_ref)
           || vec_contratacion_temporal.encuadrar_texto_v1(
                  control_sesion_revision::text)
           || vec_contratacion_temporal.encuadrar_texto_v1(
                  control_sesion_huella_sha256)
           || vec_contratacion_temporal.encuadrar_texto_v1(actor_ref)
           || vec_contratacion_temporal.encuadrar_texto_v1(perfil_id)
           || vec_contratacion_temporal.encuadrar_texto_v1(
                  perfil_version::text)
           || vec_contratacion_temporal.encuadrar_texto_v1(
                  organizacion_ref)
           || vec_contratacion_temporal.encuadrar_texto_v1(clase_ambito)
           || vec_contratacion_temporal.encuadrar_texto_v1(ambito_ref)
           || vec_contratacion_temporal.encuadrar_texto_v1(
                  sesion_huella_sha256)
           || vec_contratacion_temporal.encuadrar_texto_v1(
                  vec_contratacion_temporal.instante_utc_v1(registrada_en)
              ) AS canon
      FROM datos
)
INSERT INTO vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2 (
    acceso_ref, cuenta_ref, cuenta_ordinaria_ref, autenticacion_ref,
    autenticacion_huella_sha256, sesion_ref, control_sesion_ref,
    control_sesion_revision, control_sesion_huella_sha256, actor_ref,
    perfil_ref, perfil_version, organizacion_ref, clase_ambito,
    ambito_ref, sesion_huella_sha256, acceso_registrado_en,
    prueba_canonica, prueba_huella_sha256
)
SELECT acceso_ref, cuenta_ref, cuenta_ref, autenticacion_ref,
       autenticacion_huella_sha256, sesion_ref, control_sesion_ref,
       control_sesion_revision, control_sesion_huella_sha256, actor_ref,
       perfil_id, perfil_version, organizacion_ref, clase_ambito,
       ambito_ref, sesion_huella_sha256, registrada_en, canon,
       pg_catalog.encode(pg_catalog.sha256(canon), 'hex')
  FROM prueba;
\ir /repo/contratacion_temporal/migraciones/000040_contrato_tipado_consultas_rrhh.up.sql
ROLLBACK;
SQL
}

paso 'una historia incoherente bloquea la actualización sin efectos'
estado_previo_000040="$(estado_instalacion_000039)"
esperar_fallo 'instalación 000040 con huella de sesión ajena' 55000 \
    'estado incompatible para motor de consultas RRHH' \
    probar_base_con_huella_incompatible
comprobar_estado_000039 "$estado_previo_000040" \
    'actualización rechazada por huella de sesión ajena'

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

paso 'objetos derivados de la tabla bloquean la retirada'
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
CREATE INDEX control_motor_consultas_rrhh_v1_derivado_idx
    ON vec_contratacion_temporal.control_motor_consultas_rrhh_v1(
        version_esquema
    );
SQL
esperar_fallo 'down 000040 con índice derivado' 55000 \
    'catálogo derivado impide revertir contrato RRHH' archivo \
    contratacion_temporal/migraciones/000040_contrato_tipado_consultas_rrhh.down.sql
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
DROP INDEX vec_contratacion_temporal
    .control_motor_consultas_rrhh_v1_derivado_idx;
CREATE RULE control_motor_consultas_rrhh_v1_regla_derivada AS
    ON INSERT TO
        vec_contratacion_temporal.control_motor_consultas_rrhh_v1
    DO INSTEAD NOTHING;
SQL
esperar_fallo 'down 000040 con regla derivada' 55000 \
    'catálogo derivado impide revertir contrato RRHH' archivo \
    contratacion_temporal/migraciones/000040_contrato_tipado_consultas_rrhh.down.sql
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
DROP RULE control_motor_consultas_rrhh_v1_regla_derivada ON
    vec_contratacion_temporal.control_motor_consultas_rrhh_v1;
CREATE TABLE vec_contratacion_temporal
    .control_motor_consultas_rrhh_v1_hija ()
INHERITS (
    vec_contratacion_temporal.control_motor_consultas_rrhh_v1
);
SQL
esperar_fallo 'down 000040 con herencia derivada' 55000 \
    'catálogo derivado impide revertir contrato RRHH' archivo \
    contratacion_temporal/migraciones/000040_contrato_tipado_consultas_rrhh.down.sql
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
DROP TABLE vec_contratacion_temporal
    .control_motor_consultas_rrhh_v1_hija;
CREATE STATISTICS vec_contratacion_temporal
    .control_motor_consultas_rrhh_v1_estadistica
    ON version_esquema, creada_en
    FROM vec_contratacion_temporal.control_motor_consultas_rrhh_v1;
SQL
esperar_fallo 'down 000040 con estadística derivada' 55000 \
    'catálogo derivado impide revertir contrato RRHH' archivo \
    contratacion_temporal/migraciones/000040_contrato_tipado_consultas_rrhh.down.sql
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
DROP STATISTICS vec_contratacion_temporal
    .control_motor_consultas_rrhh_v1_estadistica;
RESET ROLE;
CREATE PUBLICATION control_motor_consultas_rrhh_v1_publicacion
    FOR TABLE
        vec_contratacion_temporal.control_motor_consultas_rrhh_v1;
SQL
esperar_fallo 'down 000040 con publicación derivada' 55000 \
    'catálogo derivado impide revertir contrato RRHH' archivo \
    contratacion_temporal/migraciones/000040_contrato_tipado_consultas_rrhh.down.sql
psql_admin --command \
    'DROP PUBLICATION control_motor_consultas_rrhh_v1_publicacion' \
    >/dev/null
comprobar_estado_000040 "$estado_limpio" 'objetos derivados retirados'

paso 'etiquetas de seguridad forman parte del manifiesto'
for migracion in \
    000040_contrato_tipado_consultas_rrhh.up.sql \
    000040_contrato_tipado_consultas_rrhh.down.sql; do
    if [[ $(rg -Fc 'FROM pg_catalog.pg_seclabel etiqueta' \
        "$directorio/migraciones/$migracion") != 1 ]]; then
        printf 'manifiesto sin etiquetas de seguridad en %s\n' \
            "$migracion" >&2
        exit 1
    fi
done

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
    'cannot drop type' \
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
