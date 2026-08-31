#!/usr/bin/env bash
set -Eeuo pipefail

directorio="$(
    cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1
    pwd -P
)"
migracion_up="$directorio/migraciones/000046_ejecuciones_seleccion_llamamiento_o6.up.sql"
migracion_down="$directorio/migraciones/000046_ejecuciones_seleccion_llamamiento_o6.down.sql"

if [[ ${VEC_CT_O6_BD_DESECHABLE:-} != SI ]]; then
    printf 'O6-03 exige VEC_CT_O6_BD_DESECHABLE=SI y una base exclusiva\n' >&2
    exit 64
fi
command -v psql >/dev/null 2>&1 || {
    printf 'psql no disponible\n' >&2
    exit 69
}

psql_focal() {
    psql -X --no-psqlrc --set ON_ERROR_STOP=1 --set VERBOSITY=terse "$@"
}

preflight="$(psql_focal --tuples-only --no-align --command "
    SELECT pg_catalog.concat_ws('|',
        pg_catalog.current_setting('server_version_num'),
        pg_catalog.getdatabaseencoding(),
        (SELECT rol.rolsuper::text
           FROM pg_catalog.pg_roles rol
          WHERE rol.rolname = pg_catalog.current_user),
        (pg_catalog.to_regnamespace('vec_contratacion_temporal') IS NOT NULL)::text,
        (pg_catalog.to_regrole('vec_contratacion_temporal_propietario') IS NOT NULL)::text,
        (pg_catalog.to_regrole('vec_contratacion_temporal_ejecutor') IS NOT NULL)::text,
        (pg_catalog.to_regclass(
            'vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'
        ) IS NULL)::text
    )")"
if [[ $preflight != '180004|UTF8|true|true|true|true|true' ]]; then
    printf 'preflight O6-03 incompatible; se exige PG18.4 y base exclusiva preparada\n' >&2
    exit 65
fi
directorio_temporal_o6="$(mktemp -d)"

limpiar() {
    local estado=$?
    trap - EXIT
    if psql_focal --tuples-only --no-align --command "
        SELECT (pg_catalog.to_regclass(
            'vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'
        ) IS NOT NULL)::text" 2>/dev/null | rg -qx true; then
        psql_focal --command "
            SET ROLE vec_contratacion_temporal_propietario;
            DELETE FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6;
        " >/dev/null 2>&1 || true
        psql_focal --file "$migracion_down" >/dev/null 2>&1 || true
    fi
    rm -rf -- "$directorio_temporal_o6"
    exit "$estado"
}
trap limpiar EXIT

psql_focal --file "$migracion_up" >/dev/null

psql_focal <<'SQL'
SET SESSION AUTHORIZATION vec_contratacion_temporal_ejecutor;

DO $prueba$
DECLARE
    v_clave_indeterminada uuid := '10000000-0000-4000-8000-000000000001';
    v_clave_liberada uuid := '10000000-0000-4000-8000-000000000002';
    v_clave_confirmada uuid := '10000000-0000-4000-8000-000000000003';
    v_huella text := pg_catalog.repeat('a', 64);
    v_referencia jsonb;
    v_base jsonb;
    v_solicitud jsonb;
    v_recibo jsonb;
    v_artefacto jsonb;
    v_artefacto_canonico text;
    v_contexto jsonb;
    v_datos jsonb;
    v_evidencia_nominal jsonb;
    v_procedencia jsonb;
    v_evidencia jsonb;
    v_huella_artefacto text;
    v_fila record;
    v_reserva text;
BEGIN
    v_referencia := pg_catalog.jsonb_build_object(
        'referencia', 'catalogo:o6:prueba', 'version', 1,
        'huella_sha256', pg_catalog.repeat('c', 64)
    );
    v_base := pg_catalog.jsonb_build_object(
        'organizacion_ref', 'organizacion:o6:prueba',
        'expediente_ref', 'expediente:o6:prueba',
        'version_expediente', 1,
        'correlacion_ref', 'correlacion:o6:prueba',
        'accion_orden', v_referencia,
        'finalidad', v_referencia,
        'necesidad', v_referencia,
        'bolsa', v_referencia,
        'politica', v_referencia,
        'maximo_posiciones', 3,
        'cantidad_disponible', 2
    );
    v_solicitud := v_base || pg_catalog.jsonb_build_object(
        'clave_idempotencia', v_clave_indeterminada::text,
        'huella_semantica', v_huella
    );
    SELECT * INTO STRICT v_fila
      FROM vec_contratacion_temporal.resolver_terminal_seleccion_llamamiento_o6_v1(
          v_clave_indeterminada
      );
    IF v_fila.situacion <> '' THEN
        RAISE EXCEPTION 'resolver O6 invento un terminal';
    END IF;

    SELECT * INTO STRICT v_fila
      FROM vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(
          v_clave_indeterminada, v_huella, v_solicitud
      );
    v_reserva := v_fila.reserva_ref;
    IF v_fila.situacion <> 'propietaria' OR v_reserva = '' THEN
        RAISE EXCEPTION 'reserva O6 no obtuvo propiedad';
    END IF;
    SELECT * INTO STRICT v_fila
      FROM vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(
          v_clave_indeterminada, v_huella, v_solicitud
      );
    IF v_fila.situacion <> 'ocupada' OR v_fila.reserva_ref <> '' THEN
        RAISE EXCEPTION 'reserva O6 viva no quedo ocupada';
    END IF;
    SELECT * INTO STRICT v_fila
      FROM vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(
          v_clave_indeterminada, pg_catalog.repeat('b', 64),
          v_solicitud || pg_catalog.jsonb_build_object(
              'huella_semantica', pg_catalog.repeat('b', 64)
          )
      );
    IF v_fila.situacion <> 'colision' THEN
        RAISE EXCEPTION 'reserva O6 no distinguio colision';
    END IF;

    PERFORM vec_contratacion_temporal.abrir_ventana_seleccion_llamamiento_o6_v1(
        v_clave_indeterminada, v_huella, v_reserva, v_solicitud, 'preparar_orden'
    );
    BEGIN
        PERFORM vec_contratacion_temporal.abrir_ventana_seleccion_llamamiento_o6_v1(
            v_clave_indeterminada, v_huella, v_reserva, v_solicitud, 'preparar_orden'
        );
        RAISE EXCEPTION 'ventana O6 repetida fue aceptada';
    EXCEPTION WHEN SQLSTATE '55000' THEN
        NULL;
    END;
    PERFORM vec_contratacion_temporal.marcar_indeterminada_seleccion_llamamiento_o6_v1(
        v_clave_indeterminada, v_huella, v_reserva, v_solicitud, 'preparar_orden'
    );
    SELECT * INTO STRICT v_fila
      FROM vec_contratacion_temporal.resolver_terminal_seleccion_llamamiento_o6_v1(
          v_clave_indeterminada
      );
    IF v_fila.situacion <> 'indeterminada' OR v_fila.efecto <> 'preparar_orden' THEN
        RAISE EXCEPTION 'terminal O6 indeterminado no fue durable';
    END IF;

    v_solicitud := v_base || pg_catalog.jsonb_build_object(
        'clave_idempotencia', v_clave_liberada::text,
        'huella_semantica', v_huella
    );
    SELECT * INTO STRICT v_fila
      FROM vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(
          v_clave_liberada, v_huella, v_solicitud
      );
    PERFORM vec_contratacion_temporal.liberar_seleccion_llamamiento_o6_v1(
        v_clave_liberada, v_huella, v_fila.reserva_ref, v_solicitud
    );
    SELECT * INTO STRICT v_fila
      FROM vec_contratacion_temporal.resolver_terminal_seleccion_llamamiento_o6_v1(
          v_clave_liberada
      );
    IF v_fila.situacion <> '' THEN
        RAISE EXCEPTION 'liberacion O6 anterior a efectos dejo historia';
    END IF;

    v_solicitud := v_base || pg_catalog.jsonb_build_object(
        'clave_idempotencia', v_clave_confirmada::text,
        'huella_semantica', v_huella
    );
    SELECT * INTO STRICT v_fila
      FROM vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(
          v_clave_confirmada, v_huella, v_solicitud
      );
    v_reserva := v_fila.reserva_ref;
    PERFORM vec_contratacion_temporal.abrir_ventana_seleccion_llamamiento_o6_v1(
        v_clave_confirmada, v_huella, v_reserva, v_solicitud, 'preparar_orden'
    );
    PERFORM vec_contratacion_temporal.abrir_ventana_seleccion_llamamiento_o6_v1(
        v_clave_confirmada, v_huella, v_reserva, v_solicitud, 'solicitar_llamamiento'
    );
    v_datos := pg_catalog.jsonb_build_object(
        'operacion_ref', 'operacion:llamamiento:o6',
        'organizacion_ref', 'organizacion:o6:prueba',
        'expediente_ref', 'expediente:o6:prueba',
        'version_expediente', 1,
        'correlacion_ref', 'correlacion:o6:prueba',
        'contrato_version', 1,
        'autoridad_solicitante', 'autoridad:contratacion-temporal',
        'autorizacion', v_referencia, 'accion', v_referencia,
        'recurso', v_referencia, 'finalidad', v_referencia,
        'solicitada_en', '2026-08-31T09:00:00Z',
        'valida_hasta', '2026-08-31T09:10:00Z'
    );
    v_contexto := pg_catalog.jsonb_build_object(
        'datos', v_datos,
        'clave_verificacion_ref',
            'vec.contratacion-temporal.integracion-bolsa-peticion/v1',
        'sello_hmac',
            'hmac-sha256:vec.contratacion-temporal.integracion-bolsa-peticion/v1:' ||
            pg_catalog.repeat('d', 64)
    );
    v_evidencia_nominal := pg_catalog.jsonb_build_object(
        'evidencia_ref', 'evidencia:llamamiento:o6',
        'clave_verificacion_ref',
            'vec.contratacion-temporal.integracion-bolsa-respuesta/v1',
        'sello_hmac',
            'hmac-sha256:vec.contratacion-temporal.integracion-bolsa-respuesta/v1:' ||
            pg_catalog.repeat('e', 64),
        'emitida_en', '2026-08-31T09:02:00Z',
        'valida_hasta', '2026-08-31T09:08:00Z',
        'retener_hasta', '2026-09-01T09:00:00Z'
    );
    v_procedencia := pg_catalog.jsonb_build_object(
        'autoridad_ref', 'autoridad:bolsa',
        'respuesta_ref', 'respuesta:llamamiento:o6',
        'contrato_version', 1, 'fuente', v_referencia,
        'evidencia', v_evidencia_nominal
    );
    v_recibo := pg_catalog.jsonb_build_object(
        'operacion_ref', 'operacion:llamamiento:o6',
        'organizacion_ref', 'organizacion:o6:prueba',
        'expediente_ref', 'expediente:o6:prueba',
        'version_expediente', 1,
        'correlacion_ref', 'correlacion:o6:prueba',
        'necesidad', v_referencia, 'bolsa', v_referencia,
        'orden', v_referencia, 'politica', v_referencia,
        'resultado', v_referencia, 'propuesta_generada', true,
        'propuesta', v_referencia, 'accion_evento', v_referencia,
        'llamamiento_ref', 'llamamiento:o6:prueba',
        'seleccion_ref',
            'hmac-sha256:vec.contratacion-temporal.seleccion/v1:' ||
            pg_catalog.repeat('9', 64),
        'retencion_seleccion', v_referencia,
        'orden_seleccionado', 2,
        'recibo_ref', 'recibo:llamamiento:o6',
        'auditoria_ref', 'auditoria:llamamiento:o6',
        'evento_ref', 'evento:llamamiento:o6',
        'confirmada_en', '2026-08-31T09:01:00Z',
        'procedencia', v_procedencia
    );
    v_evidencia := pg_catalog.jsonb_build_object(
        'esquema', 'vec.contratacion-temporal.evidencia-bolsa.v1',
        'tipo_material', 'recibo_llamamiento',
        'autoridad_ref', 'autoridad:bolsa',
        'clave_verificacion_ref',
            'vec.contratacion-temporal.integracion-bolsa-respuesta/v1',
        'evidencia_ref', 'evidencia:llamamiento:o6',
        'peticion_ref', 'operacion:llamamiento:o6',
        'huella_peticion_sha256', pg_catalog.repeat('f', 64),
        'respuesta_ref', 'respuesta:llamamiento:o6',
        'huella_respuesta_sha256', pg_catalog.repeat('1', 64),
        'sello_hmac', v_evidencia_nominal->>'sello_hmac',
        'emitida_en', '2026-08-31T09:02:00Z',
        'valida_hasta', '2026-08-31T09:08:00Z',
        'retener_hasta', '2026-09-01T09:00:00Z'
    );
    v_artefacto := pg_catalog.jsonb_build_object(
        'esquema', 'vec.contratacion-temporal.artefacto-bolsa',
        'version', 1, 'tipo', 'recibo_llamamiento',
        'comando', pg_catalog.jsonb_build_object(
            'contexto', v_contexto, 'necesidad', v_referencia,
            'bolsa', v_referencia, 'orden', v_referencia,
            'politica', v_referencia, 'total_posiciones_orden', 3,
            'maxima_posicion_evaluable', 3,
            'huella_recibo_orden', pg_catalog.repeat('3', 64)
        ),
        'recibo', v_recibo, 'evidencia', v_evidencia,
        'clave_verificacion_ref', v_evidencia->>'clave_verificacion_ref',
        'sello_hmac', v_evidencia->>'sello_hmac',
        'huella_artefacto_sha256', ''
    );
    v_artefacto_canonico := pg_catalog.replace(
        pg_catalog.replace(v_artefacto::text, ': ', ':'), ', ', ','
    );
    v_huella_artefacto := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(v_artefacto_canonico, 'UTF8')
    ), 'hex');
    v_artefacto_canonico := pg_catalog.replace(
        v_artefacto_canonico, '"huella_artefacto_sha256":""',
        '"huella_artefacto_sha256":"' || v_huella_artefacto || '"'
    );
    v_artefacto := v_artefacto_canonico::jsonb;

    BEGIN
        PERFORM vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(
            v_clave_confirmada, v_huella, v_reserva, v_solicitud,
            v_recibo, '{"esquema":"vec.contratacion-temporal.artefacto-bolsa"}'
        );
        RAISE EXCEPTION 'confirmacion O6 acepto artefacto parcial';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(
            v_clave_confirmada, v_huella, v_reserva, v_solicitud,
            v_recibo, pg_catalog.replace(pg_catalog.replace(
                (v_artefacto || pg_catalog.jsonb_build_object(
                    'huella_artefacto_sha256', pg_catalog.repeat('0', 64)
                ))::text, ': ', ':'), ', ', ',')
        );
        RAISE EXCEPTION 'confirmacion O6 acepto huella nula';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(
            v_clave_confirmada, v_huella, v_reserva, v_solicitud,
            v_recibo, pg_catalog.repeat('x', 1048577)
        );
        RAISE EXCEPTION 'confirmacion O6 acepto artefacto sobredimensionado';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(
            v_clave_confirmada, v_huella, v_reserva, v_solicitud,
            pg_catalog.jsonb_set(v_recibo, '{seleccion_ref}', '"dni:00000000T"'),
            pg_catalog.replace(pg_catalog.replace(pg_catalog.jsonb_set(
                v_artefacto, '{recibo,seleccion_ref}', '"dni:00000000T"'
            )::text, ': ', ':'), ', ', ',')
        );
        RAISE EXCEPTION 'confirmacion O6 acepto seudonimo directo';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(
            v_clave_confirmada, v_huella, v_reserva, v_solicitud,
            pg_catalog.jsonb_set(
                v_recibo, '{confirmada_en}', '"2026-08-31T09:03:00Z"'
            ),
            pg_catalog.replace(pg_catalog.replace(pg_catalog.jsonb_set(
                v_artefacto, '{recibo,confirmada_en}', '"2026-08-31T09:03:00Z"'
            )::text, ': ', ':'), ', ', ',')
        );
        RAISE EXCEPTION 'confirmacion O6 acepto cronologia invertida';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(
            v_clave_confirmada, v_huella, v_reserva, v_solicitud,
            v_recibo, pg_catalog.replace(pg_catalog.replace(pg_catalog.jsonb_set(
                v_artefacto, '{comando,contexto,datos,organizacion_ref}',
                '"organizacion:o6:ajena"'
            )::text, ': ', ':'), ', ', ',')
        );
        RAISE EXCEPTION 'confirmacion O6 acepto contexto desligado';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    PERFORM vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(
        v_clave_confirmada, v_huella, v_reserva, v_solicitud,
        v_recibo, v_artefacto_canonico
    );
    SELECT * INTO STRICT v_fila
      FROM vec_contratacion_temporal.resolver_terminal_seleccion_llamamiento_o6_v1(
          v_clave_confirmada
      );
    IF v_fila.situacion <> 'confirmada'
       OR v_fila.recibo_json::jsonb IS DISTINCT FROM v_recibo
       OR v_fila.artefacto_json IS DISTINCT FROM v_artefacto_canonico THEN
        RAISE EXCEPTION 'confirmacion O6 no conservo recibo y artefacto exactos';
    END IF;
END
$prueba$;

RESET SESSION AUTHORIZATION;

DO $acl$
BEGIN
    IF pg_catalog.has_table_privilege(
        'vec_contratacion_temporal_ejecutor',
        'vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6',
        'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
    ) THEN
        RAISE EXCEPTION 'el ejecutor O6 obtuvo acceso directo a la tabla';
    END IF;
END
$acl$;
SQL

consulta_carrera_o6="$(sed 's/^    //' <<'SQL'
    WITH referencia AS (
        SELECT pg_catalog.jsonb_build_object(
            'referencia', 'catalogo:o6:carrera', 'version', 1,
            'huella_sha256', pg_catalog.repeat('6', 64)
        ) AS valor
    ), solicitud AS (
        SELECT pg_catalog.jsonb_build_object(
            'clave_idempotencia', '10000000-0000-4000-8000-000000000004',
            'huella_semantica', pg_catalog.repeat('4', 64),
            'organizacion_ref', 'organizacion:o6:carrera',
            'expediente_ref', 'expediente:o6:carrera',
            'version_expediente', 1,
            'correlacion_ref', 'correlacion:o6:carrera',
            'accion_orden', valor, 'finalidad', valor, 'necesidad', valor,
            'bolsa', valor, 'politica', valor,
            'maximo_posiciones', 8, 'cantidad_disponible', 8
        ) AS valor FROM referencia
    )
    SELECT situacion
      FROM vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(
          '10000000-0000-4000-8000-000000000004', pg_catalog.repeat('4', 64),
          (SELECT valor FROM solicitud)
      );
SQL
)"
pids_o6=()
for indice_o6 in {1..8}; do
    psql_focal --tuples-only --no-align --command \
        "SET SESSION AUTHORIZATION vec_contratacion_temporal_ejecutor; $consulta_carrera_o6" \
        >"$directorio_temporal_o6/carrera-$indice_o6.out" &
    pids_o6+=("$!")
done
for pid_o6 in "${pids_o6[@]}"; do
    wait "$pid_o6"
done
conteo_carrera_o6="$(awk '
    $0 == "propietaria" { propietarias++ }
    $0 == "ocupada" { ocupadas++ }
    END { print propietarias + 0 "|" ocupadas + 0 }
' "$directorio_temporal_o6"/carrera-*.out)"
if [[ $conteo_carrera_o6 != '1|7' ]]; then
    printf 'carrera O6 inesperada: %s\n' "$conteo_carrera_o6" >&2
    exit 1
fi

if psql_focal --file "$migracion_down" \
    >"$directorio_temporal_o6/down-con-historia.out" 2>&1; then
    printf 'down O6 acepto historia durable\n' >&2
    exit 1
fi

psql_focal --command "
    SET ROLE vec_contratacion_temporal_propietario;
    DELETE FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6;
" >/dev/null
psql_focal --file "$migracion_down" >/dev/null
rm -rf -- "$directorio_temporal_o6"
trap - EXIT
printf '[CT-LITE-O6-03:PG18.4] GO focal\n'
