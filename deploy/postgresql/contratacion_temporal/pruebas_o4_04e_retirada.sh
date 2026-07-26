#!/usr/bin/env bash
# Se carga desde pruebas_o4_04e_failpoints_carreras.sh.
# shellcheck disable=SC2154

preparar_confirmacion_posterior_retirada_o404e() {
  docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --quiet --username vec_o404e_tcb --dbname postgres <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT vec_o404e_concedida.preparar('retiro_vence',1);
COMMIT;
SQL
}

preparar_retirada_concurrente_o404e() {
  psql_admin <<'SQL' >/dev/null
CREATE FUNCTION vec_o404e_fallos.esperar_retirada()
RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog
AS $$
BEGIN
  PERFORM pg_catalog.pg_advisory_xact_lock(4040404);
  RETURN NEW;
END
$$;
REVOKE ALL ON FUNCTION vec_o404e_fallos.esperar_retirada() FROM PUBLIC;
CREATE TRIGGER o404e_esperar_retirada
AFTER INSERT ON
  vec_contratacion_temporal.acreditacion_gobierno_decision_cobertura
FOR EACH ROW EXECUTE FUNCTION vec_o404e_fallos.esperar_retirada();
SQL
  docker exec --interactive --env PGAPPNAME=o404e_retiro_holder \
    "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --quiet --username postgres --dbname postgres <<'SQL' >/dev/null &
SELECT pg_catalog.pg_advisory_lock(4040404);
SELECT pg_catalog.pg_sleep(20);
SELECT pg_catalog.pg_advisory_unlock(4040404);
SQL
  o404e_pid_cerrojo=$!
  for _ in {1..100}; do
    [[ $(docker exec "$contenedor" psql -XAt -U postgres -d postgres \
      -c "SELECT count(*) FROM pg_locks WHERE locktype='advisory' AND objid=4040404 AND granted") == 1 ]] &&
      return
    sleep 0.05
  done
  return 1
}

lanzar_retirada_concurrente_o404e() {
  local retirada_bloqueada=0 gobierno_bloqueado=0
  for _ in {1..300}; do
    if [[ $(docker exec "$contenedor" psql -XAt -U postgres -d postgres \
      -c "SELECT count(*) FROM pg_locks WHERE locktype='advisory' AND objid=4040404 AND NOT granted") -ge 1 ]]; then
      retirada_bloqueada=1
      break
    fi
    sleep 0.05
  done
  if [[ $retirada_bloqueada != 1 ]]; then
    printf 'la confirmación no alcanzó la barrera de retirada\n' >&2
    return 1
  fi
  docker exec --interactive --env PGAPPNAME=o404e_retiro_gob \
    "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --quiet --username vec_o404e_gob --dbname postgres <<'SQL' >/dev/null &
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT vec_o404e_concedida.retirar_gobierno();
COMMIT;
SQL
  o404e_pid_retirada=$!
  for _ in {1..100}; do
    [[ $(docker exec "$contenedor" psql -XAt -U postgres -d postgres \
      -c "SELECT count(*) FROM pg_stat_activity WHERE application_name='o404e_retiro_gob' AND cardinality(pg_blocking_pids(pid))>=1") -ge 1 ]] && {
      gobierno_bloqueado=1
      break
    }
    sleep 0.05
  done
  if [[ $gobierno_bloqueado != 1 ]]; then
    printf 'la retirada no quedó bloqueada por la confirmación\n' >&2
    return 1
  fi
  docker exec "$contenedor" psql -XAt -U postgres -d postgres \
    -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE pid<>pg_backend_pid() AND application_name='o404e_retiro_holder'" \
    >/dev/null
}

verificar_retirada_concurrente_o404e() {
  wait "$o404e_pid_cerrojo" || true
  wait "$o404e_pid_retirada"
  psql_admin <<'SQL' >/dev/null
DROP TRIGGER o404e_esperar_retirada ON
  vec_contratacion_temporal.acreditacion_gobierno_decision_cobertura;
DROP FUNCTION vec_o404e_fallos.esperar_retirada();
DO $retiro$
BEGIN
  IF (SELECT resultado->>'resultado'
        FROM vec_o404e_concedida.retiro_gobierno)
       IS DISTINCT FROM 'retirada'
     OR (SELECT count(*) FROM
          vec_contratacion_temporal.gobi_o404b_retirada
         WHERE organizacion_ref='organizacion:o404e:golden'
           AND accion='contratacion_temporal.cobertura.decidir')<>1
     OR (SELECT ultima_secuencia FROM
          vec_contratacion_temporal.gobi_o404b_checkpoint
         WHERE control) IS DISTINCT FROM 2 THEN
    RAISE EXCEPTION 'retirada real no quedó serializada tras confirmación';
  END IF;
END
$retiro$;
SQL
}

verificar_confirmacion_posterior_retirada_o404e() {
  local anterior posterior salida
  anterior=$(snapshot_fallos_o404e)
  if salida="$(docker exec --interactive "$contenedor" psql -X \
    --set ON_ERROR_STOP=1 --quiet --username vec_o404e_tcb \
    --dbname postgres 2>&1 <<'SQL'
\set VERBOSITY verbose
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT * FROM vec_contratacion_temporal
  .confirmar_operacion_decision_cobertura_o404e_v1(
    vec_o404e_concedida.carga('retiro_vence')
  );
SQL
  )"; then
    printf 'confirmación posterior a retirada fue aceptada\n' >&2
    return 1
  fi
  if [[ $salida != *'ERROR:  55000:'* ]]; then
    printf '%s\n' "$salida" >&2
    return 1
  fi
  posterior=$(snapshot_fallos_o404e)
  if [[ $anterior != "$posterior" ]]; then
    printf 'confirmación posterior a retirada dejó efectos parciales\n' >&2
    return 1
  fi
}

replay_retirada_o404e() {
  docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --quiet --username vec_o404e_gob --dbname postgres <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
DO $replay$
DECLARE v_resultado jsonb;
BEGIN
  v_resultado:=vec_o404e_concedida.retirar_gobierno();
  IF v_resultado->>'resultado' IS DISTINCT FROM 'repetida'
     OR v_resultado->>'evento_ref' IS DISTINCT FROM
        (vec_o404e_concedida.carga_retirada_gobierno()
          ->>'evento_ref') THEN
    RAISE EXCEPTION 'replay de retirada real divergente';
  END IF;
END
$replay$;
COMMIT;
SQL
}
