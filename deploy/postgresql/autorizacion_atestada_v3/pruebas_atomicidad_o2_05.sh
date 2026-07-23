#!/usr/bin/env bash
# shellcheck disable=SC2154

# Cargado por probar_integracion_o2_05.sh; reutiliza sus ayudantes y contenedor.
invocar_con_fallo_o2_05() {
    local caso="$1"
    local pieza="$2"
    docker exec --interactive "${contenedor}" \
        psql -X --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username vec_ct_o205_runtime --dbname postgres <<SQL
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL vec.o205_fallo_pieza='${pieza}';
SELECT * FROM public.invocar_vector_o2_05('${caso}');
COMMIT;
SQL
}

esperar_sesion_o2_05() {
    local aplicacion="$1"
    local patron="$2"
    for _ in {1..100}; do
        if [[ "$(valor "SELECT count(*) FROM pg_stat_activity WHERE usename='vec_ct_o205_runtime' AND query LIKE '${patron}'")" == '1' ]]; then
            return
        fi
        sleep 0.1
    done
    printf 'no apareció sesión O2-05 %s (%s)\n' \
        "${aplicacion}" "${patron}" >&2
    return 1
}

instalar_fallos_o2_05() {
    docker exec --interactive "${contenedor}" \
        psql -X --set ON_ERROR_STOP=1 --username postgres \
        --dbname postgres >/dev/null <<'SQL'
CREATE FUNCTION public.fallo_escritura_o205()
RETURNS trigger LANGUAGE plpgsql AS $f$
BEGIN
    IF current_setting('vec.o205_fallo_pieza', true) = TG_ARGV[0] THEN
        IF TG_ARGV[0] <> 'reserva_confirmada'
           OR to_jsonb(NEW) ->> 'estado' = 'confirmada' THEN
            RAISE EXCEPTION 'fallo inyectado O2-05 en %', TG_ARGV[0];
        END IF;
    END IF;
    RETURN NEW;
END
$f$;
CREATE TRIGGER fallo_o205_consumo
  BEFORE INSERT ON vec_autorizacion_atestada_v3.consumo_decision_v3
  FOR EACH ROW EXECUTE FUNCTION public.fallo_escritura_o205('consumo');
CREATE TRIGGER fallo_o205_expediente
  BEFORE INSERT ON vec_contratacion_temporal.expediente_alta
  FOR EACH ROW EXECUTE FUNCTION public.fallo_escritura_o205('expediente');
CREATE TRIGGER fallo_o205_version
  BEFORE INSERT ON vec_contratacion_temporal.expediente_alta_version
  FOR EACH ROW EXECUTE FUNCTION public.fallo_escritura_o205('version');
CREATE TRIGGER fallo_o205_actuacion
  BEFORE INSERT ON vec_contratacion_temporal.actuacion_alta
  FOR EACH ROW EXECUTE FUNCTION public.fallo_escritura_o205('actuacion');
CREATE TRIGGER fallo_o205_auditoria
  BEFORE INSERT ON vec_contratacion_temporal.auditoria_alta
  FOR EACH ROW EXECUTE FUNCTION public.fallo_escritura_o205('auditoria');
CREATE TRIGGER fallo_o205_outbox
  BEFORE INSERT ON vec_contratacion_temporal.outbox_alta
  FOR EACH ROW EXECUTE FUNCTION public.fallo_escritura_o205('outbox');
CREATE TRIGGER fallo_o205_reserva
  BEFORE INSERT ON vec_contratacion_temporal.reserva_alta_version
  FOR EACH ROW EXECUTE FUNCTION public.fallo_escritura_o205(
    'reserva_confirmada'
  );
CREATE TRIGGER fallo_o205_marcador
  BEFORE INSERT ON vec_contratacion_temporal.confirmacion_agregado_alta
  FOR EACH ROW EXECUTE FUNCTION public.fallo_escritura_o205('marcador');
SQL
}

retirar_fallos_o2_05() {
    sql postgres \
        "DROP TRIGGER fallo_o205_consumo ON vec_autorizacion_atestada_v3.consumo_decision_v3; DROP TRIGGER fallo_o205_expediente ON vec_contratacion_temporal.expediente_alta; DROP TRIGGER fallo_o205_version ON vec_contratacion_temporal.expediente_alta_version; DROP TRIGGER fallo_o205_actuacion ON vec_contratacion_temporal.actuacion_alta; DROP TRIGGER fallo_o205_auditoria ON vec_contratacion_temporal.auditoria_alta; DROP TRIGGER fallo_o205_outbox ON vec_contratacion_temporal.outbox_alta; DROP TRIGGER fallo_o205_reserva ON vec_contratacion_temporal.reserva_alta_version; DROP TRIGGER fallo_o205_marcador ON vec_contratacion_temporal.confirmacion_agregado_alta; DROP FUNCTION public.fallo_escritura_o205()" \
        >/dev/null
}

probar_cancelacion_precommit_o2_05() {
    local caso='cancelacion_precommit'
    local aplicacion='o205_cancelacion_precommit'
    local log="${directorio_temporal}/cancelacion-precommit.log"
    preparar "${caso}"
    docker exec --env PGAPPNAME="${aplicacion}" --interactive "${contenedor}" \
        psql -X --set ON_ERROR_STOP=1 --username vec_ct_o205_runtime \
        --dbname postgres >"${log}" 2>&1 <<SQL &
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT * FROM public.invocar_vector_o2_05('${caso}');
SELECT pg_sleep(30) /* o205_cancelacion_precommit */;
COMMIT;
SQL
    local cliente=$!
    if ! esperar_sesion_o2_05 "${aplicacion}" '%o205_cancelacion_precommit%'; then
        sed -n '1,160p' "${log}" >&2
        return 1
    fi
    [[ "$(valor "SELECT pg_cancel_backend(pid)::text FROM pg_stat_activity WHERE usename='vec_ct_o205_runtime' AND query LIKE '%o205_cancelacion_precommit%'")" == 'true' ]]
    if wait "${cliente}"; then
        printf 'la cancelación pre-COMMIT terminó con éxito inesperado\n' >&2
        sed -n '1,160p' "${log}" >&2
        return 1
    fi
    [[ "$(estado_agregado_o2_05 "${caso}")" == '0:0:0:0:0:0:0:0' ]]
}

probar_respuesta_perdida_o2_05() {
    local caso='respuesta_perdida'
    local aplicacion='o205_respuesta_perdida'
    local log="${directorio_temporal}/respuesta-perdida.log"
    preparar "${caso}"
    docker exec --env PGAPPNAME="${aplicacion}" --interactive "${contenedor}" \
        psql -X --set ON_ERROR_STOP=1 --username vec_ct_o205_runtime \
        --dbname postgres >"${log}" 2>&1 <<SQL &
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT * FROM public.invocar_vector_o2_05('${caso}');
COMMIT;
SELECT pg_sleep(30) /* o205_respuesta_perdida */;
SQL
    local cliente=$!
    if ! esperar_sesion_o2_05 "${aplicacion}" '%o205_respuesta_perdida%'; then
        sed -n '1,160p' "${log}" >&2
        return 1
    fi
    [[ "$(estado_agregado_o2_05 "${caso}")" == '1:1:1:1:1:1:1:1' ]]
    [[ "$(valor "SELECT pg_terminate_backend(pid)::text FROM pg_stat_activity WHERE usename='vec_ct_o205_runtime' AND query LIKE '%o205_respuesta_perdida%'")" == 'true' ]]
    if wait "${cliente}"; then
        printf 'la conexión con respuesta perdida no se cortó\n' >&2
        return 1
    fi
    local primero segundo
    primero="$(invocar "${caso}")"
    segundo="$(invocar "${caso}")"
    [[ "${primero}" == "${segundo}" ]]
    afirmar_agregado_completo_o2_05 "${caso}"
}

probar_reinicio_sin_memoria_o2_05() {
    local caso='reinicio_conexion'
    local antes despues
    preparar "${caso}"
    antes="$(invocar "${caso}")"
    [[ "$(valor "SELECT count(*) FROM pg_stat_activity WHERE usename='vec_ct_o205_runtime'")" == '0' ]]
    # invocar abre otro proceso psql y otro backend: no comparte memoria del
    # cliente que recibió la primera confirmación.
    despues="$(invocar "${caso}")"
    [[ "${antes}" == "${despues}" ]]
    afirmar_agregado_completo_o2_05 "${caso}"
}

probar_atomicidad_y_reconciliacion_o2_05() {
    local pieza caso
    paso 'fallo inyectado en cada una de las ocho escrituras del agregado'
    instalar_fallos_o2_05
    for pieza in consumo expediente version actuacion auditoria outbox \
        reserva_confirmada marcador; do
        caso="fallo_${pieza}"
        preparar "${caso}"
        esperar_fallo "rollback en ${pieza}" \
            invocar_con_fallo_o2_05 "${caso}" "${pieza}"
        [[ "$(estado_agregado_o2_05 "${caso}")" == '0:0:0:0:0:0:0:0' ]]
    done
    retirar_fallos_o2_05
    paso 'cancelación concluyente antes de COMMIT'
    probar_cancelacion_precommit_o2_05
    paso 'COMMIT indeterminado, respuesta perdida y reconciliación'
    probar_respuesta_perdida_o2_05
    paso 'replay tras reinicio del proceso cliente y conexión sin memoria'
    probar_reinicio_sin_memoria_o2_05
}
