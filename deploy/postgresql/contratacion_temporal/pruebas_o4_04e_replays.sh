#!/usr/bin/env bash
# Se carga desde pruebas_o4_04e_failpoints_carreras.sh.
# shellcheck disable=SC2154,SC2317
replay_denegado() {
  docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --quiet --username vec_o404e_tcb --dbname postgres <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
DO $replay$
DECLARE
  v_recibo jsonb;
BEGIN
  SELECT recibo_json INTO STRICT v_recibo
    FROM vec_contratacion_temporal
      .confirmar_operacion_decision_cobertura_o404e_v1(
        vec_o404e_prueba.carga()
      );
  IF v_recibo IS DISTINCT FROM vec_o404e_prueba.recibo() THEN
    RAISE EXCEPTION 'replay denegado no fue JSON exacto';
  END IF;
END
$replay$;
COMMIT;
SQL
}

replay_concedido() {
  docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --quiet --username vec_o404e_tcb --dbname postgres <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
DO $replay$
DECLARE
  v_recibo jsonb;
BEGIN
  SELECT recibo_json INTO STRICT v_recibo
    FROM vec_contratacion_temporal
      .confirmar_operacion_decision_cobertura_o404e_v1(
        vec_o404e_concedida.carga('uno')
      );
  IF v_recibo IS DISTINCT FROM vec_o404e_concedida.recibo('uno') THEN
    RAISE EXCEPTION 'replay concedido no fue JSON exacto';
  END IF;
END
$replay$;
COMMIT;
SQL
}

paso 'replay terminal concurrente de ambas ramas en ocho sesiones'
snapshot_replay_anterior=$(snapshot_fallos_o404e)
pids=()
for _ in {1..4}; do
  replay_denegado &
  pids+=("$!")
  replay_concedido &
  pids+=("$!")
done
for pid in "${pids[@]}"; do
  wait "$pid"
done
snapshot_replay_posterior=$(snapshot_fallos_o404e)
if [[ $snapshot_replay_anterior != "$snapshot_replay_posterior" ]]; then
  printf 'replay terminal concurrente alteró snapshot completo\n' >&2
  return 1 2>/dev/null || exit 1
fi

paso 'reinicio real y replay durable sin lease vigente'
docker restart "$contenedor" >/dev/null
for _ in {1..60}; do
  docker exec "$contenedor" pg_isready -q -U postgres -d postgres &&
    break
  sleep 1
done
replay_retirada_o404e
replay_denegado
replay_concedido

paso 'OK: fresh, ACL/RLS, down, upgrade, ambas ramas, C1, concurrencia y reinicio'
