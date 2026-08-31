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
    v_recibo jsonb := '{"propuesta_generada":true,"propuesta_ref":"propuesta:o6:prueba"}';
    v_artefacto jsonb;
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
    v_artefacto := pg_catalog.jsonb_build_object(
        'esquema', 'vec.contratacion-temporal.artefacto-bolsa',
        'version', 1, 'tipo', 'recibo_llamamiento', 'recibo', v_recibo
    );
    PERFORM vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(
        v_clave_confirmada, v_huella, v_reserva, v_solicitud,
        v_recibo, v_artefacto
    );
    SELECT * INTO STRICT v_fila
      FROM vec_contratacion_temporal.resolver_terminal_seleccion_llamamiento_o6_v1(
          v_clave_confirmada
      );
    IF v_fila.situacion <> 'confirmada'
       OR v_fila.recibo_json::jsonb IS DISTINCT FROM v_recibo
       OR v_fila.artefacto_json::jsonb IS DISTINCT FROM v_artefacto THEN
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

psql_focal --command "
    SET ROLE vec_contratacion_temporal_propietario;
    DELETE FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6;
" >/dev/null
psql_focal --file "$migracion_down" >/dev/null
trap - EXIT
printf '[CT-LITE-O6-03:PG18.4] GO focal\n'
