#!/usr/bin/env bash
# Se ejecuta con source desde probar_o4_04e_pg18_4.sh.
# Requiere Bash, PostgreSQL 18 y los fixtures sintéticos O4-04E.
# shellcheck disable=SC2154
source "$raiz/deploy/postgresql/contratacion_temporal/pruebas_o4_04e_concurrencia_helpers.sh"
paso 'concurrencia adversaria: misma reserva, un COMMIT y 40001'
preparar_normal_o404e adv_reserva
iniciar_bloqueador_o404e o404e_bloqueo_reserva \
  "SELECT 1
     FROM vec_contratacion_temporal.reserva_operacion_decision_cobertura
    WHERE reserva_ref='reserva:o404e:adv_reserva' FOR UPDATE"
lanzar_confirmacion_o404e adv_reserva o404e_r1_a \
  "$o404e_tmp/r1-a.out" "$o404e_tmp/r1-a.rc"
pid_a=$o404e_pid_ultimo
lanzar_confirmacion_o404e adv_reserva o404e_r1_b \
  "$o404e_tmp/r1-b.out" "$o404e_tmp/r1-b.rc"
pid_b=$o404e_pid_ultimo
esperar_actividad_o404e o404e_r1_ 2
liberar_bloqueador_o404e
esperar_carrera_o404e "$pid_a" "$pid_b"
validar_un_ganador_o404e \
  "$o404e_tmp/r1-a.out" "$o404e_tmp/r1-a.rc" \
  "$o404e_tmp/r1-b.out" "$o404e_tmp/r1-b.rc" \
  40001 'misma reserva'
reconciliar_exacto_o404e adv_reserva
psql_admin <<'SQL' >/dev/null
DO $unica$
BEGIN
  IF (SELECT count(*) FROM
        vec_contratacion_temporal.confirmacion_operacion_decision_cobertura
       WHERE reserva_ref='reserva:o404e:adv_reserva') IS DISTINCT FROM 1
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.terminal_operacion_decision_cobertura t
        JOIN vec_contratacion_temporal.reserva_operacion_decision_cobertura b
          USING(ambito_raiz_hmac)
       WHERE b.reserva_ref='reserva:o404e:adv_reserva')
          IS DISTINCT FROM 1 THEN
    RAISE EXCEPTION 'misma reserva dejó cardinalidad distinta de uno';
  END IF;
END
$unica$;
SQL
snapshot_replay_antes=$(snapshot_identidad_o404e adv_reserva)
ejecutar_acotado_o404e 17s 1s docker exec \
  --env PGAPPNAME=o404e_replay_rw --interactive "$contenedor" \
  psql -X --set ON_ERROR_STOP=1 --quiet --username vec_o404e_tcb \
  --dbname postgres <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT recibo_json AS recibo FROM vec_contratacion_temporal
 .confirmar_operacion_decision_cobertura_o404e_v1(
   vec_o404e_concedida.carga('adv_reserva')) \gset replay_
SELECT :'replay_recibo'::jsonb IS NOT DISTINCT FROM
  vec_o404e_concedida.recibo('adv_reserva') AS exacto \gset
\if :exacto
\else
  SELECT 1/0;
\endif
COMMIT;
SQL
snapshot_replay_despues=$(snapshot_identidad_o404e adv_reserva)
exigir_snapshot_igual_o404e "$snapshot_replay_antes" \
  "$snapshot_replay_despues" 'replay RW de misma reserva'
if [[ $(psql_admin --tuples-only --no-align --command \
  "SELECT to_regprocedure(
    'vec_o404e_concedida.preparar_ortogonal(text,integer,text,text)')
     IS NOT NULL" </dev/null) != t ]]; then
  fallar_concurrencia_o404e \
    'falta preparar_ortogonal(text,integer,text,text): caso, cantidad, expediente compartido y evidencia compartida'
  return 1
fi
paso 'concurrencia adversaria: dos reservas alcanzan el mismo CAS agregado'
preparar_ortogonal_o404e adv_cas_a adv_cas_compartido
preparar_ortogonal_o404e adv_cas_b adv_cas_compartido
iniciar_bloqueador_o404e o404e_bloqueo_cas \
  "SELECT 1 FROM vec_contratacion_temporal.expediente_integral_actual
    WHERE expediente_ref='expediente:o404e:adv_cas_compartido' FOR UPDATE"
lanzar_confirmacion_o404e adv_cas_a o404e_r2_a \
  "$o404e_tmp/r2-a.out" "$o404e_tmp/r2-a.rc"
pid_a=$o404e_pid_ultimo
lanzar_confirmacion_o404e adv_cas_b o404e_r2_b \
  "$o404e_tmp/r2-b.out" "$o404e_tmp/r2-b.rc"
pid_b=$o404e_pid_ultimo
esperar_actividad_o404e o404e_r2_ 2
liberar_bloqueador_o404e
esperar_carrera_o404e "$pid_a" "$pid_b"
validar_un_ganador_o404e \
  "$o404e_tmp/r2-a.out" "$o404e_tmp/r2-a.rc" \
  "$o404e_tmp/r2-b.out" "$o404e_tmp/r2-b.rc" \
  40001 'CAS agregado'
[[ $o404e_ganador == a ]] && caso_perdedor=adv_cas_b ||
  caso_perdedor=adv_cas_a
reconciliar_exacto_o404e "$caso_perdedor" ausente
psql_admin <<'SQL' >/dev/null
DO $cas$
BEGIN
  IF (SELECT version FROM
        vec_contratacion_temporal.expediente_integral_actual
       WHERE expediente_ref='expediente:o404e:adv_cas_compartido')
          IS DISTINCT FROM 3
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.confirmacion_operacion_decision_cobertura
       WHERE reserva_ref IN
        ('reserva:o404e:adv_cas_a','reserva:o404e:adv_cas_b'))
          IS DISTINCT FROM 1 THEN
    RAISE EXCEPTION 'CAS agregado no dejó un único ganador';
  END IF;
END
$cas$;
SQL
paso 'concurrencia adversaria: evidencia C1 compartida se consume una vez'
preparar_ortogonal_o404e adv_c1_a '' adv_c1_compartida
preparar_ortogonal_o404e adv_c1_b '' adv_c1_compartida
snapshot_c1_a_antes=$(snapshot_identidad_o404e adv_c1_a rollback)
snapshot_c1_b_antes=$(snapshot_identidad_o404e adv_c1_b rollback)
IFS='|' read -r sec_auditoria_c1_antes sec_outbox_c1_antes < <(
  psql_admin --tuples-only --no-align --field-separator='|' --command \
    'SELECT secuencia_auditoria,secuencia_outbox FROM
       vec_contratacion_temporal.control_cadenas_expediente_integral' \
    </dev/null)
iniciar_bloqueador_o404e o404e_bloqueo_c1 \
  'LOCK TABLE vec_contratacion_temporal.consumo_cobertura_evidencia IN SHARE MODE'
lanzar_confirmacion_o404e adv_c1_a o404e_r3_a \
  "$o404e_tmp/r3-a.out" "$o404e_tmp/r3-a.rc"
pid_a=$o404e_pid_ultimo
lanzar_confirmacion_o404e adv_c1_b o404e_r3_b \
  "$o404e_tmp/r3-b.out" "$o404e_tmp/r3-b.rc"
pid_b=$o404e_pid_ultimo
esperar_actividad_o404e o404e_r3_ 2
liberar_bloqueador_o404e
esperar_carrera_o404e "$pid_a" "$pid_b"
validar_un_ganador_o404e \
  "$o404e_tmp/r3-a.out" "$o404e_tmp/r3-a.rc" \
  "$o404e_tmp/r3-b.out" "$o404e_tmp/r3-b.rc" \
  '23505|40001' 'consumo C1 compartido'
[[ $o404e_ganador == a ]] && caso_perdedor=adv_c1_b ||
  caso_perdedor=adv_c1_a
reconciliar_exacto_o404e "$caso_perdedor" ausente
snapshot_c1_perdedor_despues=$(
  snapshot_identidad_o404e "$caso_perdedor" rollback)
if [[ $caso_perdedor == adv_c1_a ]]; then
  snapshot_c1_perdedor_antes=$snapshot_c1_a_antes
else
  snapshot_c1_perdedor_antes=$snapshot_c1_b_antes
fi
exigir_snapshot_igual_o404e "$snapshot_c1_perdedor_antes" \
  "$snapshot_c1_perdedor_despues" 'rollback perdedor C1'
psql_admin --set "o404e_sec_a=$sec_auditoria_c1_antes" \
  --set "o404e_sec_o=$sec_outbox_c1_antes" <<'SQL' >/dev/null
SELECT set_config('vec.prueba_o404e_sec_a',:'o404e_sec_a',false);
SELECT set_config('vec.prueba_o404e_sec_o',:'o404e_sec_o',false);
DO $c1$
DECLARE
  v_sec_a numeric:=current_setting('vec.prueba_o404e_sec_a')::numeric;
  v_sec_o numeric:=current_setting('vec.prueba_o404e_sec_o')::numeric;
BEGIN
  IF (SELECT count(*) FROM
        vec_contratacion_temporal.consumo_cobertura_evidencia
       WHERE peticion_ref='peticion:o404e:adv_c1_compartida:1')
          IS DISTINCT FROM 1
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.consumo_cobertura_lote
       WHERE reserva_ref IN
        ('reserva:o404e:adv_c1_a','reserva:o404e:adv_c1_b'))
          IS DISTINCT FROM 1
     OR (SELECT count(*) FROM
        vec_autorizacion.enlace_decision_cobertura_ct_o404e
       WHERE reserva_ref IN
        ('reserva:o404e:adv_c1_a','reserva:o404e:adv_c1_b'))
          IS DISTINCT FROM 1
     OR (SELECT count(*) FROM
        vec_autorizacion.decision_concedida_contexto_actor_v3
       WHERE decision_ref IN ('decision-vec:o404e:adv_c1_a',
         'decision-vec:o404e:adv_c1_b')) IS DISTINCT FROM 1
     OR (SELECT count(*) FROM vec_contratacion_temporal
          .acreditacion_gobierno_decision_cobertura
       WHERE reserva_ref IN ('reserva:o404e:adv_c1_a',
         'reserva:o404e:adv_c1_b')) IS DISTINCT FROM 1
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.expediente_version_integral
       WHERE expediente_ref IN ('expediente:o404e:adv_c1_a',
         'expediente:o404e:adv_c1_b')
         AND version IS NOT DISTINCT FROM 3) IS DISTINCT FROM 1
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.expediente_integral_actual
       WHERE expediente_ref IN
        ('expediente:o404e:adv_c1_a','expediente:o404e:adv_c1_b')
         AND version IS NOT DISTINCT FROM 3) IS DISTINCT FROM 1
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.expediente_integral_actual
       WHERE expediente_ref IN
        ('expediente:o404e:adv_c1_a','expediente:o404e:adv_c1_b')
         AND version IS NOT DISTINCT FROM 2) IS DISTINCT FROM 1
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.actuacion_expediente_integral
       WHERE operacion_ref IN ('actuacion:o404e:adv_c1_a',
         'actuacion:o404e:adv_c1_b')) IS DISTINCT FROM 1
     OR (SELECT count(*) FROM vec_contratacion_temporal
          .decision_cobertura_gobernada_durable
       WHERE reserva_ref IN ('reserva:o404e:adv_c1_a',
         'reserva:o404e:adv_c1_b')) IS DISTINCT FROM 1
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.auditoria_decision_cobertura
       WHERE reserva_ref IN ('reserva:o404e:adv_c1_a',
         'reserva:o404e:adv_c1_b')) IS DISTINCT FROM 1
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.outbox_expediente_integral
       WHERE evento_ref IN ('evento:o404e:adv_c1_a',
         'evento:o404e:adv_c1_b')) IS DISTINCT FROM 1
     OR (SELECT count(*) FROM vec_contratacion_temporal
          .confirmacion_operacion_decision_cobertura
       WHERE reserva_ref IN ('reserva:o404e:adv_c1_a',
         'reserva:o404e:adv_c1_b')) IS DISTINCT FROM 1
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.terminal_operacion_decision_cobertura t
        JOIN vec_contratacion_temporal.reserva_operacion_decision_cobertura b
          USING(ambito_raiz_hmac)
       WHERE b.reserva_ref IN ('reserva:o404e:adv_c1_a',
         'reserva:o404e:adv_c1_b')) IS DISTINCT FROM 1
     OR (SELECT cabeza_auditoria_sha256 FROM
        vec_contratacion_temporal.control_cadenas_expediente_integral)
       IS DISTINCT FROM (SELECT huella_sha256 FROM
        vec_contratacion_temporal.auditoria_decision_cobertura
       WHERE reserva_ref IN ('reserva:o404e:adv_c1_a',
         'reserva:o404e:adv_c1_b'))
     OR (SELECT cabeza_outbox_sha256 FROM
        vec_contratacion_temporal.control_cadenas_expediente_integral)
       IS DISTINCT FROM (SELECT huella_sha256 FROM
        vec_contratacion_temporal.outbox_expediente_integral
       WHERE evento_ref IN ('evento:o404e:adv_c1_a',
         'evento:o404e:adv_c1_b'))
     OR (SELECT secuencia_auditoria FROM
        vec_contratacion_temporal.control_cadenas_expediente_integral)
       IS DISTINCT FROM v_sec_a+1
     OR (SELECT secuencia_outbox FROM
        vec_contratacion_temporal.control_cadenas_expediente_integral)
       IS DISTINCT FROM v_sec_o+1 THEN
    RAISE EXCEPTION 'C1 compartida no revirtió íntegramente al perdedor';
  END IF;
END
$c1$;
SQL
source "$raiz/deploy/postgresql/contratacion_temporal/pruebas_o4_04e_concurrencia_vec.sh"
paso 'concurrencia adversaria: ciclo 40P01 y lectura nueva sin retry'
preparar_normal_o404e adv_dl_a
preparar_normal_o404e adv_dl_b
psql_admin <<'SQL' >/dev/null
CREATE FUNCTION vec_o404e_fallos.bloquear_reserva_adversaria(
  p_caso text,p_barrera bigint)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path=pg_catalog
AS $$
BEGIN
  IF session_user IS DISTINCT FROM 'vec_o404e_tcb' THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='actor adversario inválido';
  END IF;
  PERFORM 1
    FROM vec_contratacion_temporal.reserva_operacion_decision_cobertura
   WHERE reserva_ref='reserva:o404e:'||p_caso FOR UPDATE;
  IF NOT FOUND THEN
    RAISE EXCEPTION USING ERRCODE='P0002',MESSAGE='reserva adversaria ausente';
  END IF;
  PERFORM pg_advisory_xact_lock_shared(p_barrera);
END
$$;
REVOKE ALL ON FUNCTION
  vec_o404e_fallos.bloquear_reserva_adversaria(text,bigint) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_o404e_fallos TO vec_o404e_tcb;
GRANT EXECUTE ON FUNCTION
  vec_o404e_fallos.bloquear_reserva_adversaria(text,bigint)
  TO vec_o404e_tcb;
SQL
iniciar_bloqueador_o404e o404e_bloqueo_deadlock \
  'SELECT pg_advisory_xact_lock(9040405)'
lanzar_deadlock_o404e() {
  local bloqueada=$1 objetivo=$2 aplicacion=$3 salida=$4 estado=$5
  (
    set +e
    trap - ERR
    ejecutar_acotado_o404e 17s 1s docker exec \
      --env "PGAPPNAME=$aplicacion" --interactive "$contenedor" \
      psql -X --set ON_ERROR_STOP=1 --set VERBOSITY=sqlstate --quiet \
      --username vec_o404e_tcb --dbname postgres \
      --set "o404e_bloqueada=$bloqueada" --set "o404e_objetivo=$objetivo" \
      >"$salida" 2>&1 <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT vec_o404e_fallos.bloquear_reserva_adversaria(
  :'o404e_bloqueada',9040405);
SELECT recibo_json AS recibo FROM vec_contratacion_temporal
 .confirmar_operacion_decision_cobertura_o404e_v1(
   vec_o404e_concedida.carga(:'o404e_objetivo'))
\gset o404e_
SELECT vec_o404e_concedida.guardar_recibo(
  :'o404e_objetivo',:'o404e_recibo'::jsonb);
COMMIT;
\echo O404E_COMMIT
SQL
    printf '%s\n' "$?" >"$estado"
  ) &
  o404e_pid_ultimo=$!
}
lanzar_deadlock_o404e adv_dl_a adv_dl_b o404e_r5_a \
  "$o404e_tmp/r5-a.out" "$o404e_tmp/r5-a.rc"
pid_a=$o404e_pid_ultimo
lanzar_deadlock_o404e adv_dl_b adv_dl_a o404e_r5_b \
  "$o404e_tmp/r5-b.out" "$o404e_tmp/r5-b.rc"
pid_b=$o404e_pid_ultimo
esperar_actividad_o404e o404e_r5_ 2
liberar_bloqueador_o404e
esperar_carrera_o404e "$pid_a" "$pid_b"
validar_un_ganador_o404e \
  "$o404e_tmp/r5-a.out" "$o404e_tmp/r5-a.rc" \
  "$o404e_tmp/r5-b.out" "$o404e_tmp/r5-b.rc" \
  40P01 'deadlock controlado'
if [[ $o404e_ganador == a ]]; then
  caso_reconciliado=adv_dl_b
  caso_abortado=adv_dl_a
else
  caso_reconciliado=adv_dl_a
  caso_abortado=adv_dl_b
fi
# Transacción nueva y exclusivamente lectora: no se vuelve a confirmar.
reconciliar_exacto_o404e "$caso_reconciliado"
reconciliar_exacto_o404e "$caso_abortado" ausente
psql_admin <<'SQL' >/dev/null
DO $deadlock$
BEGIN
  IF (SELECT count(*) FROM
        vec_contratacion_temporal.confirmacion_operacion_decision_cobertura
       WHERE reserva_ref IN
        ('reserva:o404e:adv_dl_a','reserva:o404e:adv_dl_b'))
          IS DISTINCT FROM 1 THEN
    RAISE EXCEPTION '40P01 fue seguido por escritura/retry inesperado';
  END IF;
END
$deadlock$;
DROP FUNCTION vec_o404e_fallos.bloquear_reserva_adversaria(text,bigint);
REVOKE USAGE ON SCHEMA vec_o404e_fallos FROM vec_o404e_tcb;
SQL
limpiar_concurrencia_o404e
trap - ERR
trap limpiar INT TERM
paso 'OK: reserva, CAS, C1, identidad VEC, 40P01 y reconciliación primaria'
