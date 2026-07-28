#!/usr/bin/env bash
set -Eeuo pipefail

directorio="$(
    cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1
    pwd -P
)"

# Instala y acredita la línea base exacta CT-000040 sobre PostgreSQL 18.4.
# El runner fuente mantiene su trap de limpieza durante todo este proceso.
# shellcheck disable=SC1091
source "$directorio/probar_o4_05_contrato_motor_rrhh_pg18_4.sh"

paso() {
    printf '[O4-05:CT-000041A:PG18.4] %s\n' "$1"
}

estado_000041a() {
    valor "SELECT pg_catalog.concat_ws('|',
        cobertura.version_esquema::text,
        consultas.version_esquema::text,
        (pg_catalog.to_regclass(
            'vec_contratacion_temporal.control_vocabulario_estados_publicacion_rrhh_v1'
        ) IS NOT NULL)::text,
        restriccion.convalidated::text,
        restriccion.conenforced::text,
        pg_catalog.pg_get_constraintdef(restriccion.oid, false)
    )
      FROM vec_contratacion_temporal.control_migracion_cobertura_o4 cobertura
      CROSS JOIN
           vec_contratacion_temporal.control_migracion_consultas_rrhh consultas
      JOIN pg_catalog.pg_constraint restriccion
        ON restriccion.conrelid =
           'vec_contratacion_temporal.publicacion_version_rrhh'::regclass
       AND restriccion.conname =
           'publicacion_version_rrhh_estado_clave_valido'
     WHERE cobertura.control AND consultas.control"
}

comprobar_estado_000041a() {
    local esperado=$1
    local contexto=$2
    local obtenido
    obtenido="$(estado_000041a)"
    if [[ $obtenido != "$esperado" ]]; then
        printf 'estado CT-000041A alterado tras %s\n' "$contexto" >&2
        printf 'esperado=%s\nobtenido=%s\n' "$esperado" "$obtenido" >&2
        return 1
    fi
}

paso 'instalación, manifiesto, ACL, historia y seis estados exactos'
archivo \
    contratacion_temporal/migraciones/000041_vocabulario_estados_publicacion_rrhh.up.sql
archivo \
    contratacion_temporal/pruebas_sql/o405_vocabulario_estados_publicacion_rrhh.sql
estado_limpio="$(estado_000041a)"
if [[ $estado_limpio != '21|5|true|true|true|CHECK '* ]]; then
    printf 'estado inicial CT-000041A inesperado: %s\n' \
        "$estado_limpio" >&2
    exit 1
fi

# shellcheck disable=SC1091
source \
    "$directorio/pruebas_o405_vocabulario_estados_adversarial.sh"
probar_derivas_vocabulario_instalado "$estado_limpio"

paso 'la reentrada se rechaza sin efectos parciales'
esperar_fallo 'segunda instalación CT-000041A' 55000 \
    'estado incompatible para ampliar estados RRHH' \
    archivo \
    contratacion_temporal/migraciones/000041_vocabulario_estados_publicacion_rrhh.up.sql
comprobar_estado_000041a "$estado_limpio" 'reentrada'

paso 'la deriva no validada se detecta y conserva todo el estado'
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
ALTER TABLE vec_contratacion_temporal.publicacion_version_rrhh
DROP CONSTRAINT publicacion_version_rrhh_estado_clave_valido;
ALTER TABLE vec_contratacion_temporal.publicacion_version_rrhh
ADD CONSTRAINT publicacion_version_rrhh_estado_clave_valido
CHECK (estado_clave IN (
    'pendiente', 'en_curso', 'espera_externa',
    'completado', 'incidencia', 'cancelado'
)) NOT VALID;
SQL
esperar_fallo 'down con restricción sin validar' 55000 \
    'estructura derivada impide revertir estados RRHH' \
    archivo \
    contratacion_temporal/migraciones/000041_vocabulario_estados_publicacion_rrhh.down.sql
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
ALTER TABLE vec_contratacion_temporal.publicacion_version_rrhh
VALIDATE CONSTRAINT publicacion_version_rrhh_estado_clave_valido;
SQL
comprobar_estado_000041a "$estado_limpio" 'deriva restaurada'

paso 'objetos derivados y ACL ajena bloquean la reversión'
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
CREATE INDEX control_vocabulario_estados_rrhh_derivado_idx
ON vec_contratacion_temporal
   .control_vocabulario_estados_publicacion_rrhh_v1(version_esquema);
SQL
esperar_fallo 'down con índice derivado' 55000 \
    'estructura derivada impide revertir estados RRHH' \
    archivo \
    contratacion_temporal/migraciones/000041_vocabulario_estados_publicacion_rrhh.down.sql
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
DROP INDEX vec_contratacion_temporal
    .control_vocabulario_estados_rrhh_derivado_idx;
GRANT SELECT ON
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
TO vec_contratacion_temporal_consultor_rrhh;
SQL
esperar_fallo 'down con ACL derivada' 55000 \
    'estructura derivada impide revertir estados RRHH' \
    archivo \
    contratacion_temporal/migraciones/000041_vocabulario_estados_publicacion_rrhh.down.sql
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
REVOKE SELECT ON
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
FROM vec_contratacion_temporal_consultor_rrhh;
SQL
comprobar_estado_000041a "$estado_limpio" 'catálogo y ACL restaurados'

paso 'una dependencia futura queda protegida por RESTRICT'
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
CREATE TABLE vec_contratacion_temporal.dependencia_ct000041a_prueba (
    control boolean REFERENCES
        vec_contratacion_temporal
            .control_vocabulario_estados_publicacion_rrhh_v1(control)
);
SQL
esperar_fallo 'down con dependencia futura' 2BP01 \
    'cannot drop table' \
    archivo \
    contratacion_temporal/migraciones/000041_vocabulario_estados_publicacion_rrhh.down.sql
comprobar_estado_000041a "$estado_limpio" 'dependencia futura'
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
DROP TABLE vec_contratacion_temporal.dependencia_ct000041a_prueba;
SQL

paso 'las barreras posteriores impiden retirar el vocabulario'
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 22
 WHERE control AND version_esquema = 21;
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 6
 WHERE control AND version_esquema = 5;
SQL
esperar_fallo 'down con barreras futuras' 55000 \
    'estado incompatible para revertir estados RRHH' \
    archivo \
    contratacion_temporal/migraciones/000041_vocabulario_estados_publicacion_rrhh.down.sql
psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 21
 WHERE control AND version_esquema = 22;
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 5
 WHERE control AND version_esquema = 6;
SQL
comprobar_estado_000041a "$estado_limpio" 'barreras restauradas'

paso 'ciclo limpio UP/DOWN/UP'
archivo \
    contratacion_temporal/migraciones/000041_vocabulario_estados_publicacion_rrhh.down.sql
estado_retirado="$(valor "SELECT pg_catalog.concat_ws('|',
    cobertura.version_esquema::text,
    consultas.version_esquema::text,
    (pg_catalog.to_regclass(
        'vec_contratacion_temporal.control_vocabulario_estados_publicacion_rrhh_v1'
    ) IS NULL)::text,
    restriccion.convalidated::text,
    pg_catalog.pg_get_constraintdef(restriccion.oid, false)
)
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4 cobertura
  CROSS JOIN
       vec_contratacion_temporal.control_migracion_consultas_rrhh consultas
  JOIN pg_catalog.pg_constraint restriccion
    ON restriccion.conrelid =
       'vec_contratacion_temporal.publicacion_version_rrhh'::regclass
   AND restriccion.conname =
       'publicacion_version_rrhh_estado_clave_check'
 WHERE cobertura.control AND consultas.control")"
if [[ $estado_retirado != '20|4|true|true|CHECK '* ]]; then
    printf 'down CT-000041A incompleto: %s\n' "$estado_retirado" >&2
    exit 1
fi
probar_derivas_antes_up "$estado_retirado"
archivo \
    contratacion_temporal/migraciones/000041_vocabulario_estados_publicacion_rrhh.up.sql
archivo \
    contratacion_temporal/pruebas_sql/o405_vocabulario_estados_publicacion_rrhh.sql
comprobar_estado_000041a "$estado_limpio" 'segundo ciclo'

paso 'una publicación con estado nuevo hace el down transaccionalmente seguro'
psql_admin <<'SQL'
SELECT vec_o405_publicacion_prueba.crear_alta(
    'expediente:ct000041:down', '2026/CT000041'
);
SET session_replication_role = replica;
WITH datos AS (
    SELECT vec_o405_publicacion_prueba.agregado(
        'expediente:ct000041:down', '2026/CT000041', 1, 'base'
    ) AS agregado
), prueba AS (
    SELECT agregado,
           pg_catalog.convert_to(
               pg_catalog.repeat('prueba:ct000041:', 16), 'UTF8'
           ) AS bytes
      FROM datos
)
INSERT INTO vec_contratacion_temporal.expediente_version_integral (
    expediente_ref, version, agregado_json,
    agregado_json_huella_sha256, prueba_canonica,
    prueba_huella_sha256, flujo_ref, flujo_version,
    flujo_huella_sha256, fase_clave, estado, origen_version,
    operacion_ref, registrada_en
)
SELECT 'expediente:ct000041:down', 1, agregado,
       pg_catalog.encode(pg_catalog.sha256(
           pg_catalog.convert_to(agregado::text, 'UTF8')
       ), 'hex'),
       bytes,
       pg_catalog.encode(pg_catalog.sha256(bytes), 'hex'),
       'flujo:contratacion:publicacion', 3,
       pg_catalog.repeat('a', 64),
       'analisis_rrhh', 'en_curso', 'analisis_o3',
       'operacion:ct000041:down',
       '2026-01-03T00:00:00Z'::timestamptz
  FROM prueba;
SET session_replication_role = origin;

BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $fixture_publicado$
DECLARE
    v_corte numeric;
    v_fila record;
BEGIN
    SELECT ultimo_corte + 1
      INTO STRICT v_corte
      FROM vec_contratacion_temporal.control_publicacion_rrhh
     WHERE control
     FOR UPDATE;
    SELECT historia.*, alta.organizacion_ref, alta.numero_visible
      INTO STRICT v_fila
      FROM vec_contratacion_temporal.expediente_version_integral historia
      JOIN vec_contratacion_temporal.expediente_alta alta USING (
          expediente_ref
      )
     WHERE historia.expediente_ref = 'expediente:ct000041:down'
       AND historia.version = 1;
    INSERT INTO vec_contratacion_temporal.publicacion_version_rrhh (
        expediente_ref, version, corte_global, organizacion_ref,
        numero_visible, flujo_ref, flujo_version, flujo_huella_sha256,
        fase_clave, estado_clave, centro_ref, categoria_ref,
        modalidad_clave, unidad_ref, creado_en, actualizado_en,
        agregado_huella_sha256, registrada_en
    )
    SELECT v_fila.expediente_ref, v_fila.version, v_corte,
           extraida.organizacion_ref, extraida.numero_visible,
           v_fila.flujo_ref, v_fila.flujo_version,
           v_fila.flujo_huella_sha256, v_fila.fase_clave,
           'incidencia', extraida.centro_ref, extraida.categoria_ref,
           extraida.modalidad_clave, extraida.unidad_ref,
           extraida.creado_en, extraida.actualizado_en,
           v_fila.agregado_json_huella_sha256, v_fila.registrada_en
      FROM vec_contratacion_temporal.extraer_publicacion_rrhh_v1(
          v_fila.expediente_ref, v_fila.version, v_fila.agregado_json,
          v_fila.agregado_json_huella_sha256, v_fila.flujo_ref,
          v_fila.flujo_version, v_fila.flujo_huella_sha256,
          v_fila.fase_clave, v_fila.estado, v_fila.registrada_en,
          v_fila.organizacion_ref, v_fila.numero_visible
      ) extraida;
    UPDATE vec_contratacion_temporal.control_publicacion_rrhh
       SET ultimo_corte = v_corte,
           actualizada_en = pg_catalog.date_trunc(
               'microseconds', pg_catalog.clock_timestamp()
           )
     WHERE control;
END
$fixture_publicado$;
COMMIT;
SQL
estado_con_nuevo="$(estado_000041a)"
esperar_fallo 'down con estado nuevo publicado' 2BP01 \
    'estados nuevos impiden revertir estados RRHH' \
    archivo \
    contratacion_temporal/migraciones/000041_vocabulario_estados_publicacion_rrhh.down.sql
comprobar_estado_000041a "$estado_con_nuevo" \
    'rechazo del down con estado nuevo'
if [[ $(valor "SELECT estado_clave
  FROM vec_contratacion_temporal.publicacion_version_rrhh
 WHERE expediente_ref = 'expediente:ct000041:down'
   AND version = 1") != incidencia ]]; then
    printf 'el down rechazado no conservó el estado nuevo\n' >&2
    exit 1
fi

paso 'vocabulario durable CT-000041A superado'
