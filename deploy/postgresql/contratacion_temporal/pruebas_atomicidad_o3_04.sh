#!/usr/bin/env bash
# shellcheck disable=SC2154

# Cargado por probar_integracion_o2_05.sh. Todo el instrumental vive en
# `public`, se instala después del vector sintético y se retira antes de ACL.

invocar_con_fallo_analisis_o304() {
    local pieza="$1"
    docker exec --interactive "${contenedor}" \
        psql -XAtq --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username vec_ct_o205_runtime --dbname postgres <<SQL
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SET LOCAL vec.o304_fallo_pieza='${pieza}';
SELECT recibo_json::text
  FROM public.invocar_vector_confirmacion_analisis_o3();
COMMIT;
SQL
}

esperar_fallo_inyectado_analisis_o304() {
    local pieza="$1"
    local salida
    if salida="$(invocar_con_fallo_analisis_o304 "${pieza}" 2>&1)"; then
        printf 'O3-04 no falló en la escritura %s\n%s\n' \
            "${pieza}" "${salida}" >&2
        return 1
    fi
    if [[ "${salida}" != *"fallo inyectado O3-04 en ${pieza}"* ]]; then
        printf 'O3-04 falló antes de alcanzar %s:\n%s\n' \
            "${pieza}" "${salida}" >&2
        return 1
    fi
    paso "rechazo verificado: O3-04 rollback en ${pieza}"
}

refrescar_vector_analisis_o304() {
    local numero="$1"
    local sufijo
    printf -v sufijo '%03d' "${numero}"
    sql postgres \
        "BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
         SELECT public.preparar_vector_confirmacion_analisis_o3(
           'decision:ct:o3:analisis-${sufijo}'
         );
         COMMIT" >/dev/null
}

instalar_fallos_analisis_o304() {
    # V3 revalida una decisión VEC ya durable y no escribe en su autoridad.
    # Su fila completa forma parte del estado comparado tras cada rollback;
    # `decision_local` cubre la única escritura de decisión de esta unidad.
    docker exec --interactive "${contenedor}" \
        psql -X --set ON_ERROR_STOP=1 --username postgres \
        --dbname postgres >/dev/null <<'SQL'
CREATE FUNCTION public.fallo_escritura_analisis_o304()
RETURNS trigger
LANGUAGE plpgsql
AS $funcion$
BEGIN
    IF pg_catalog.current_setting(
           'vec.o304_fallo_pieza', true
       ) = TG_ARGV[0] THEN
        RAISE EXCEPTION USING
            ERRCODE = 'P0001',
            MESSAGE = 'fallo inyectado O3-04 en ' || TG_ARGV[0];
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER fallo_o304_version
  BEFORE INSERT
  ON vec_contratacion_temporal.expediente_version_integral
  FOR EACH ROW
  EXECUTE FUNCTION public.fallo_escritura_analisis_o304('version');
CREATE TRIGGER fallo_o304_puntero
  BEFORE UPDATE
  ON vec_contratacion_temporal.expediente_integral_actual
  FOR EACH ROW
  EXECUTE FUNCTION public.fallo_escritura_analisis_o304('puntero');
CREATE TRIGGER fallo_o304_actuacion
  BEFORE INSERT
  ON vec_contratacion_temporal.actuacion_expediente_integral
  FOR EACH ROW
  EXECUTE FUNCTION public.fallo_escritura_analisis_o304('actuacion');
CREATE TRIGGER fallo_o304_fuente
  BEFORE INSERT
  ON vec_contratacion_temporal.consumo_fuentes_analisis
  FOR EACH ROW
  EXECUTE FUNCTION public.fallo_escritura_analisis_o304('fuente');
CREATE TRIGGER fallo_o304_decision_local
  BEFORE INSERT
  ON vec_contratacion_temporal.consumo_decision_analisis
  FOR EACH ROW
  EXECUTE FUNCTION public.fallo_escritura_analisis_o304('decision_local');
CREATE TRIGGER fallo_o304_auditoria
  BEFORE INSERT
  ON vec_contratacion_temporal.auditoria_expediente_integral
  FOR EACH ROW
  EXECUTE FUNCTION public.fallo_escritura_analisis_o304('auditoria');
CREATE TRIGGER fallo_o304_outbox
  BEFORE INSERT
  ON vec_contratacion_temporal.outbox_expediente_integral
  FOR EACH ROW
  EXECUTE FUNCTION public.fallo_escritura_analisis_o304('outbox');
CREATE TRIGGER fallo_o304_cadenas
  BEFORE UPDATE
  ON vec_contratacion_temporal.control_cadenas_expediente_integral
  FOR EACH ROW
  EXECUTE FUNCTION public.fallo_escritura_analisis_o304('cadenas');
CREATE TRIGGER fallo_o304_confirmacion
  BEFORE INSERT
  ON vec_contratacion_temporal.confirmacion_operacion_analisis
  FOR EACH ROW
  EXECUTE FUNCTION public.fallo_escritura_analisis_o304('confirmacion');
CREATE TRIGGER fallo_o304_alias
  BEFORE INSERT
  ON vec_contratacion_temporal.alias_consulta_operacion_analisis
  FOR EACH ROW
  EXECUTE FUNCTION public.fallo_escritura_analisis_o304('alias');
CREATE TRIGGER fallo_o304_reserva_version
  BEFORE INSERT
  ON vec_contratacion_temporal.reserva_operacion_analisis_version
  FOR EACH ROW
  EXECUTE FUNCTION public.fallo_escritura_analisis_o304('reserva_version');
CREATE TRIGGER fallo_o304_reserva_actual
  BEFORE UPDATE
  ON vec_contratacion_temporal.reserva_operacion_analisis_actual
  FOR EACH ROW
  EXECUTE FUNCTION public.fallo_escritura_analisis_o304('reserva_actual');
CREATE TRIGGER fallo_o304_vinculo_replay
  BEFORE INSERT
  ON vec_contratacion_temporal.vinculo_replay_operacion_analisis_v2
  FOR EACH ROW
  EXECUTE FUNCTION public.fallo_escritura_analisis_o304('vinculo_replay');
SQL
}

retirar_fallos_analisis_o304() {
    docker exec --interactive "${contenedor}" \
        psql -X --set ON_ERROR_STOP=1 --username postgres \
        --dbname postgres >/dev/null <<'SQL'
DROP TRIGGER fallo_o304_version
  ON vec_contratacion_temporal.expediente_version_integral;
DROP TRIGGER fallo_o304_puntero
  ON vec_contratacion_temporal.expediente_integral_actual;
DROP TRIGGER fallo_o304_actuacion
  ON vec_contratacion_temporal.actuacion_expediente_integral;
DROP TRIGGER fallo_o304_fuente
  ON vec_contratacion_temporal.consumo_fuentes_analisis;
DROP TRIGGER fallo_o304_decision_local
  ON vec_contratacion_temporal.consumo_decision_analisis;
DROP TRIGGER fallo_o304_auditoria
  ON vec_contratacion_temporal.auditoria_expediente_integral;
DROP TRIGGER fallo_o304_outbox
  ON vec_contratacion_temporal.outbox_expediente_integral;
DROP TRIGGER fallo_o304_cadenas
  ON vec_contratacion_temporal.control_cadenas_expediente_integral;
DROP TRIGGER fallo_o304_confirmacion
  ON vec_contratacion_temporal.confirmacion_operacion_analisis;
DROP TRIGGER fallo_o304_alias
  ON vec_contratacion_temporal.alias_consulta_operacion_analisis;
DROP TRIGGER fallo_o304_reserva_version
  ON vec_contratacion_temporal.reserva_operacion_analisis_version;
DROP TRIGGER fallo_o304_reserva_actual
  ON vec_contratacion_temporal.reserva_operacion_analisis_actual;
DROP TRIGGER fallo_o304_vinculo_replay
  ON vec_contratacion_temporal.vinculo_replay_operacion_analisis_v2;
DROP FUNCTION public.fallo_escritura_analisis_o304();
SQL
}

estado_durable_analisis_o304() {
    valor "SELECT pg_catalog.jsonb_build_object(
      'decision_vec', (
        SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.decision_ref)
          FROM vec_autorizacion.decision_concedida_contexto_actor_v3 d
         WHERE d.decision_ref LIKE 'decision:ct:o3:analisis-%'
      ),
      'versiones', (
        SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(v) ORDER BY v.version)
          FROM vec_contratacion_temporal.expediente_version_integral v
         WHERE v.expediente_ref='expediente:ct:o205:alta_valida'
      ),
      'puntero', (
        SELECT pg_catalog.to_jsonb(a)
          FROM vec_contratacion_temporal.expediente_integral_actual a
         WHERE a.expediente_ref='expediente:ct:o205:alta_valida'
      ),
      'actuaciones', (
        SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(a) ORDER BY a.secuencia)
          FROM vec_contratacion_temporal.actuacion_expediente_integral a
         WHERE a.expediente_ref='expediente:ct:o205:alta_valida'
      ),
      'fuentes', (
        SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(f) ORDER BY f.consumo_ref)
          FROM vec_contratacion_temporal.consumo_fuentes_analisis f
         WHERE f.expediente_ref='expediente:ct:o205:alta_valida'
      ),
      'decisiones_locales', (
        SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.decision_ref)
          FROM vec_contratacion_temporal.consumo_decision_analisis d
      ),
      'auditoria', (
        SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(a) ORDER BY a.secuencia)
          FROM vec_contratacion_temporal.auditoria_expediente_integral a
      ),
      'outbox', (
        SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(o) ORDER BY o.secuencia)
          FROM vec_contratacion_temporal.outbox_expediente_integral o
      ),
      'cadenas', (
        SELECT pg_catalog.to_jsonb(c)
          FROM vec_contratacion_temporal.control_cadenas_expediente_integral c
         WHERE c.control_id
      ),
      'confirmaciones', (
        SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(c) ORDER BY c.ambito_raiz_hmac)
          FROM vec_contratacion_temporal.confirmacion_operacion_analisis c
      ),
      'aliases_consulta', (
        SELECT pg_catalog.jsonb_agg(
                   pg_catalog.to_jsonb(a)
                   ORDER BY a.alias_ambito_consulta_hmac
               )
          FROM vec_contratacion_temporal.alias_consulta_operacion_analisis a
      ),
      'reserva', (
        SELECT pg_catalog.to_jsonb(r)
          FROM vec_contratacion_temporal.reserva_operacion_analisis r
         WHERE r.expediente_ref='expediente:ct:o205:alta_valida'
      ),
      'reserva_versiones', (
        SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(v) ORDER BY v.revision)
          FROM vec_contratacion_temporal.reserva_operacion_analisis_version v
         WHERE v.ambito_raiz_hmac=(
           SELECT r.ambito_raiz_hmac
             FROM vec_contratacion_temporal.reserva_operacion_analisis r
            WHERE r.expediente_ref='expediente:ct:o205:alta_valida'
         )
      ),
      'reserva_actual', (
        SELECT pg_catalog.to_jsonb(a)
          FROM vec_contratacion_temporal.reserva_operacion_analisis_actual a
         WHERE a.ambito_raiz_hmac=(
           SELECT r.ambito_raiz_hmac
             FROM vec_contratacion_temporal.reserva_operacion_analisis r
            WHERE r.expediente_ref='expediente:ct:o205:alta_valida'
         )
      ),
      'replay', (
        SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(v) ORDER BY v.ambito_raiz_hmac)
          FROM vec_contratacion_temporal.vinculo_replay_operacion_analisis_v2 v
      )
    )::text"
}

afirmar_estado_analisis_o304() {
    local esperado="$1"
    local obtenido
    obtenido="$(estado_durable_analisis_o304)"
    if [[ "${obtenido}" != "${esperado}" ]]; then
        printf 'estado O3-04 cambió pese al rollback\n' >&2
        diff -u <(printf '%s\n' "${esperado}") \
            <(printf '%s\n' "${obtenido}") >&2 || true
        return 1
    fi
}

probar_invariantes_directas_analisis_o304() {
    docker exec --interactive "${contenedor}" \
        psql -X --set ON_ERROR_STOP=1 --username postgres \
        --dbname postgres >/dev/null <<'SQL'
DO $prueba$
DECLARE
    a jsonb;
    alterado jsonb;
BEGIN
    SELECT operacion #> '{expediente_siguiente,analisis}'
      INTO STRICT a
      FROM public.vector_confirmacion_analisis_o3
     WHERE caso = 'registrar';
    IF vec_contratacion_temporal.analisis_rrhh_valido_v3(a) IS NOT TRUE THEN
        RAISE EXCEPTION 'el análisis sintético base no es válido en V3';
    END IF;
    alterado := pg_catalog.jsonb_set(
        a, '{categoria_ref}', '""'::jsonb
    );
    IF vec_contratacion_temporal
           .analisis_rrhh_valido_v3(alterado) IS NOT FALSE THEN
        RAISE EXCEPTION 'V3 aceptó categoría vacía';
    END IF;
    alterado := pg_catalog.jsonb_set(
        pg_catalog.jsonb_set(
            a,
            '{entrada_rc_esperada,huella_sha256}',
            pg_catalog.to_jsonb(pg_catalog.repeat('0', 64))
        ),
        '{validacion_rc,huella_entrada_sha256}',
        pg_catalog.to_jsonb(pg_catalog.repeat('0', 64))
    );
    IF vec_contratacion_temporal
           .analisis_rrhh_valido_v3(alterado) IS NOT FALSE THEN
        RAISE EXCEPTION 'V3 aceptó huellas cero coordinadas';
    END IF;
    alterado := pg_catalog.jsonb_set(
        a, '{validacion_rc,motivo}', '""'::jsonb
    );
    IF vec_contratacion_temporal
           .analisis_rrhh_valido_v3(alterado) IS NOT FALSE THEN
        RAISE EXCEPTION 'V3 aceptó motivo no requerida vacío';
    END IF;
    alterado := pg_catalog.jsonb_set(
        a, '{periodo,fin}', '"2126-08-02T00:00:00Z"'::jsonb
    );
    IF vec_contratacion_temporal
           .analisis_rrhh_valido_v3(alterado) IS NOT FALSE THEN
        RAISE EXCEPTION 'V3 aceptó periodo superior a cien años';
    END IF;
END
$prueba$;
SQL
}

probar_cancelacion_precommit_analisis_o304() {
    local aplicacion='o304_cancelacion_precommit'
    local log="${directorio_temporal}/o304-cancelacion-precommit.log"
    local inicial cliente
    inicial="$(estado_durable_analisis_o304)"
    docker exec --env PGAPPNAME="${aplicacion}" --interactive \
        "${contenedor}" psql -XAtq --set ON_ERROR_STOP=1 \
        --username vec_ct_o205_runtime --dbname postgres \
        >"${log}" 2>&1 <<'SQL' &
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT recibo_json::text
  FROM public.invocar_vector_confirmacion_analisis_o3();
SELECT pg_sleep(30) /* o304_cancelacion_precommit */;
COMMIT;
SQL
    cliente=$!
    if ! esperar_sesion_o2_05 \
        "${aplicacion}" '%o304_cancelacion_precommit%'; then
        sed -n '1,160p' "${log}" >&2
        return 1
    fi
    [[ "$(valor "SELECT pg_cancel_backend(pid)::text
      FROM pg_stat_activity
     WHERE usename='vec_ct_o205_runtime'
       AND query LIKE '%o304_cancelacion_precommit%'")" == 'true' ]]
    if wait "${cliente}"; then
        printf 'cancelación O3-04 pre-COMMIT terminó con éxito\n' >&2
        sed -n '1,160p' "${log}" >&2
        return 1
    fi
    afirmar_estado_analisis_o304 "${inicial}"
}

invocar_analisis_o304_con_reintentos() {
    local indice="$1"
    local log="${directorio_temporal}/o304-carrera-${indice}.log"
    local _
    for _ in 1 2 3 4 5; do
        if confirmar_analisis_o3 >"${log}" 2>&1; then
            return
        fi
        if ! grep -Eq 'ERROR:  (40001|40P01):' "${log}"; then
            printf 'fallo O3-04 concurrente no reintentable, sesión %s:\n' \
                "${indice}" >&2
            sed -n '1,160p' "${log}" >&2
            return 1
        fi
    done
    printf 'O3-04 agotó reintentos concurrentes, sesión %s\n' \
        "${indice}" >&2
    sed -n '1,160p' "${log}" >&2
    return 1
}

probar_carrera_analisis_o304() {
    local inicial="$1"
    local indice pid terminal
    local -a procesos=()
    for indice in 1 2 3 4; do
        invocar_analisis_o304_con_reintentos "${indice}" &
        procesos+=("$!")
    done
    for pid in "${procesos[@]}"; do
        wait "${pid}"
    done
    terminal="$(
        sed -n '/^{/,$p' "${directorio_temporal}/o304-carrera-1.log"
    )"
    [[ -n "${terminal}" ]]
    for indice in 2 3 4; do
        [[ "$(
            sed -n '/^{/,$p' \
                "${directorio_temporal}/o304-carrera-${indice}.log"
        )" == "${terminal}" ]]
    done
    [[ "$(valor "SELECT (
      (SELECT count(*)
         FROM vec_autorizacion.decision_concedida_contexto_actor_v3
        WHERE decision_ref LIKE 'decision:ct:o3:analisis-%')=10
      AND (SELECT count(*)
             FROM vec_contratacion_temporal.expediente_version_integral
            WHERE expediente_ref='expediente:ct:o205:alta_valida')=2
      AND (SELECT version
             FROM vec_contratacion_temporal.expediente_integral_actual
            WHERE expediente_ref='expediente:ct:o205:alta_valida')=2
      AND (SELECT count(*)
             FROM vec_contratacion_temporal.actuacion_expediente_integral)=1
      AND (SELECT count(*)
             FROM vec_contratacion_temporal.consumo_fuentes_analisis)=1
      AND (SELECT count(*)
             FROM vec_contratacion_temporal.consumo_decision_analisis)=1
      AND (SELECT count(*)
             FROM vec_contratacion_temporal.auditoria_expediente_integral)=1
      AND (SELECT count(*)
             FROM vec_contratacion_temporal.outbox_expediente_integral)=1
      AND (SELECT secuencia_auditoria
             FROM vec_contratacion_temporal.control_cadenas_expediente_integral
            WHERE control_id)=1
      AND (SELECT secuencia_outbox
             FROM vec_contratacion_temporal.control_cadenas_expediente_integral
            WHERE control_id)=1
      AND (SELECT count(*)
             FROM vec_contratacion_temporal.confirmacion_operacion_analisis)=1
      AND (SELECT count(*)
             FROM vec_contratacion_temporal.alias_consulta_operacion_analisis)=1
      AND (SELECT count(*)
             FROM vec_contratacion_temporal.reserva_operacion_analisis_version
            WHERE estado='confirmada')=1
      AND (SELECT count(*)
             FROM vec_contratacion_temporal.reserva_operacion_analisis_actual
            WHERE revision=2)=1
      AND (SELECT count(*)
             FROM vec_contratacion_temporal.vinculo_replay_operacion_analisis_v2)=1
      AND (SELECT cabeza_auditoria_sha256
             FROM vec_contratacion_temporal.control_cadenas_expediente_integral
            WHERE control_id)=(
          SELECT huella_sha256
            FROM vec_contratacion_temporal.auditoria_expediente_integral
           WHERE secuencia=1
      )
      AND (SELECT cabeza_outbox_sha256
             FROM vec_contratacion_temporal.control_cadenas_expediente_integral
            WHERE control_id)=(
          SELECT huella_sha256
            FROM vec_contratacion_temporal.outbox_expediente_integral
           WHERE secuencia=1
      )
    )::text")" == 'true' ]]
    [[ "$(estado_durable_analisis_o304)" != "${inicial}" ]]
}

probar_atomicidad_y_carrera_analisis_o304() {
    local inicial pieza numero=2 dentro=0
    paso 'O3-04: rollback en cada escritura durable real'
    instalar_fallos_analisis_o304
    refrescar_vector_analisis_o304 "${numero}"
    inicial="$(estado_durable_analisis_o304)"
    for pieza in version puntero actuacion fuente decision_local auditoria \
        outbox cadenas confirmacion alias reserva_version reserva_actual \
        vinculo_replay; do
        if [[ "${dentro}" -eq 2 ]]; then
            numero=$((numero + 1))
            refrescar_vector_analisis_o304 "${numero}"
            inicial="$(estado_durable_analisis_o304)"
            dentro=0
        fi
        esperar_fallo_inyectado_analisis_o304 "${pieza}"
        afirmar_estado_analisis_o304 "${inicial}"
        dentro=$((dentro + 1))
    done
    retirar_fallos_analisis_o304
    numero=$((numero + 1))
    refrescar_vector_analisis_o304 "${numero}"
    paso 'O3-04: cancelación concluyente antes de COMMIT'
    probar_cancelacion_precommit_analisis_o304
    numero=$((numero + 1))
    refrescar_vector_analisis_o304 "${numero}"
    paso 'O3-04: cuatro sesiones, un efecto y un recibo terminal'
    probar_carrera_analisis_o304 "${inicial}"
}
