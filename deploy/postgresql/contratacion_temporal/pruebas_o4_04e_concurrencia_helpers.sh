#!/usr/bin/env bash
# Helpers privados del script adversario; se cargan con source.
# shellcheck disable=SC2034,SC2154
o404e_tmp="/tmp/o404e-concurrencia-${PPID}-${RANDOM}"
mkdir -m 700 "$o404e_tmp"
o404e_pid_ultimo=
o404e_pid_bloqueador=
o404e_liberador=
o404e_ganador=
ejecutar_acotado_o404e() {
  local limite=$1 gracia=$2
  shift 2
  timeout --signal=TERM --kill-after="$gracia" "$limite" "$@"
}
pid_hijo_o404e() {
  local pid=$1
  [[ $pid =~ ^[0-9]+$ ]] &&
    [[ $(ps -o ppid= -p "$pid" 2>/dev/null | tr -d ' ') == "$$" ]]
}
pid_ejecutando_o404e() {
  local pid=$1 estado
  kill -0 "$pid" 2>/dev/null || return 1
  estado=$(ps -o stat= -p "$pid" 2>/dev/null | tr -d ' ')
  [[ $estado != Z* ]]
}
terminar_pid_o404e() {
  local pid=$1 limite=${2:-2} fin
  pid_hijo_o404e "$pid" || return 0
  kill -TERM "$pid" 2>/dev/null || true
  fin=$((SECONDS+limite))
  while pid_ejecutando_o404e "$pid" && (( SECONDS < fin )); do
    sleep 0.05
  done
  if pid_ejecutando_o404e "$pid" && pid_hijo_o404e "$pid"; then
    kill -KILL "$pid" 2>/dev/null || true
  fi
  fin=$((SECONDS+limite))
  while pid_ejecutando_o404e "$pid" && (( SECONDS < fin )); do
    sleep 0.05
  done
  pid_ejecutando_o404e "$pid" || wait "$pid" 2>/dev/null || true
}
limpiar_concurrencia_o404e() {
  local pid
  trap - ERR
  [[ -z ${o404e_liberador:-} ]] ||
    ejecutar_acotado_o404e 1s 0.2s touch "$o404e_liberador" 2>/dev/null ||
    true
  for pid in "${pid_a:-}" "${pid_b:-}" "${o404e_pid_bloqueador:-}"; do
    [[ -n $pid ]] || continue
    terminar_pid_o404e "$pid" 2 || true
  done
  ejecutar_acotado_o404e 4s 1s docker exec --interactive "$contenedor" \
    psql -X --set ON_ERROR_STOP=0 \
    --username postgres --dbname postgres <<'SQL' >/dev/null 2>&1 || true
DROP FUNCTION IF EXISTS
  vec_o404e_fallos.escribir_vec_externo_adversario(text);
DROP FUNCTION IF EXISTS
  vec_o404e_fallos.bloquear_reserva_adversaria(text,bigint);
REVOKE USAGE ON SCHEMA vec_o404e_fallos FROM vec_o404e_tcb;
SQL
  ejecutar_acotado_o404e 2s 0.5s rm -rf "$o404e_tmp" 2>/dev/null || true
}
trap limpiar_concurrencia_o404e ERR
trap 'limpiar_concurrencia_o404e; limpiar' INT TERM
fallar_concurrencia_o404e() {
  printf 'O4-04E concurrencia adversaria: %s\n' "$1" >&2
  return 1
}
sqlstate_o404e() {
  sed -nE 's/^ERROR:[[:space:]]+([0-9A-Z]{5})[[:space:]]*$/\1/p' "$1" |
    head -n 1
}
esperar_actividad_o404e() {
  local aplicacion=$1 esperados=$2 modo=${3:-bloqueada}
  local consulta cantidad
  if [[ $modo == bloqueada ]]; then
    consulta="SELECT count(*) FROM pg_stat_activity WHERE
      application_name LIKE '${aplicacion}%' AND wait_event_type='Lock'"
  else
    consulta="SELECT count(*) FROM pg_stat_activity WHERE
      application_name='${aplicacion}' AND state='idle in transaction'"
  fi
  # El sondeo completo queda por debajo del lock_timeout productivo de 2 s.
  for _ in {1..5}; do
    cantidad=$(ejecutar_acotado_o404e 0.2s 0.1s docker exec \
      --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
      --username postgres --dbname postgres --tuples-only --no-align \
      --command "$consulta" </dev/null) || cantidad=
    [[ $cantidad == "$esperados" ]] && return 0
    sleep 0.02
  done
  fallar_concurrencia_o404e \
    "barrera ${aplicacion}: se esperaban ${esperados} sesiones ${modo}"
}
iniciar_bloqueador_o404e() {
  local aplicacion=$1 sentencia=$2
  o404e_liberador="$o404e_tmp/liberar-${aplicacion}"
  ejecutar_acotado_o404e 1s 0.2s rm -f "$o404e_liberador"
  (
    {
      printf "SET application_name='%s'; SET lock_timeout='5s'; SET statement_timeout='7s'; BEGIN;\n%s;\n" \
        "$aplicacion" "$sentencia"
      for _ in {1..200}; do
        [[ -e $o404e_liberador ]] && break
        sleep 0.03
      done
      [[ -e $o404e_liberador ]] && printf 'COMMIT;\n' ||
        printf 'ROLLBACK;\n'
    } | ejecutar_acotado_o404e 8s 1s docker exec \
      --interactive "$contenedor" \
      psql -X --set ON_ERROR_STOP=1 --quiet \
      --username postgres --dbname postgres >/dev/null
  ) &
  o404e_pid_bloqueador=$!
  esperar_actividad_o404e "$aplicacion" 1 ociosa
}
liberar_bloqueador_o404e() {
  ejecutar_acotado_o404e 1s 0.2s touch "$o404e_liberador"
  if ! esperar_pid_o404e "$o404e_pid_bloqueador" bloqueador 4; then
    fallar_concurrencia_o404e 'el bloqueador de la barrera falló'
    return 1
  fi
  ejecutar_acotado_o404e 1s 0.2s rm -f "$o404e_liberador"
  o404e_pid_bloqueador=
}
esperar_pid_o404e() {
  local pid=$1 etiqueta=$2 limite=${3:-18}
  local fin=$((SECONDS+limite))
  while pid_ejecutando_o404e "$pid"; do
    if (( SECONDS >= fin )); then
      terminar_pid_o404e "$pid" 2 || true
      fallar_concurrencia_o404e "deadline shell agotado: $etiqueta"
      return 1
    fi
    sleep 0.05
  done
  wait "$pid"
}
preparar_normal_o404e() {
  local caso=$1
  ejecutar_acotado_o404e 17s 1s docker exec --interactive "$contenedor" \
    psql -X --set ON_ERROR_STOP=1 --quiet \
    --username vec_o404e_tcb --dbname postgres \
    --set "o404e_caso=$caso" <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT vec_o404e_concedida.preparar(:'o404e_caso',1);
COMMIT;
SQL
}
preparar_ortogonal_o404e() {
  local caso=$1 expediente=${2:-} evidencia=${3:-}
  ejecutar_acotado_o404e 17s 1s docker exec --interactive "$contenedor" \
    psql -X --set ON_ERROR_STOP=1 --quiet \
    --username vec_o404e_tcb --dbname postgres \
    --set "o404e_caso=$caso" --set "o404e_expediente=$expediente" \
    --set "o404e_evidencia=$evidencia" <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT vec_o404e_concedida.preparar_ortogonal(:'o404e_caso',1,
  NULLIF(:'o404e_expediente',''),
  NULLIF(:'o404e_evidencia',''));
COMMIT;
SQL
}
lanzar_confirmacion_o404e() {
  local caso=$1 aplicacion=$2 salida=$3 estado=$4
  (
    set +e
    trap - ERR
    ejecutar_acotado_o404e 17s 1s docker exec \
      --env "PGAPPNAME=$aplicacion" --interactive "$contenedor" \
      psql -X --set ON_ERROR_STOP=1 --set VERBOSITY=sqlstate --quiet \
      --username vec_o404e_tcb --dbname postgres \
      --set "o404e_caso=$caso" >"$salida" 2>&1 <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT recibo_json AS recibo
FROM vec_contratacion_temporal
  .confirmar_operacion_decision_cobertura_o404e_v1(
    vec_o404e_concedida.carga(:'o404e_caso'))
\gset o404e_
SELECT vec_o404e_concedida.guardar_recibo(
  :'o404e_caso',:'o404e_recibo'::jsonb);
COMMIT;
\echo O404E_COMMIT
SQL
    printf '%s\n' "$?" >"$estado"
  ) &
  o404e_pid_ultimo=$!
}
esperar_carrera_o404e() {
  local p_a=$1 p_b=$2
  esperar_pid_o404e "$p_a" contendiente-a
  esperar_pid_o404e "$p_b" contendiente-b
  pid_a=
  pid_b=
}
validar_un_ganador_o404e() {
  local salida_a=$1 estado_a=$2 salida_b=$3 estado_b=$4
  local codigo_perdedor=$5 etiqueta=$6
  local commit_a=0 commit_b=0 codigo
  grep -qx 'O404E_COMMIT' "$salida_a" && commit_a=1
  grep -qx 'O404E_COMMIT' "$salida_b" && commit_b=1
  if (( commit_a + commit_b != 1 )); then
    printf '%s\n' "A=$(<"$estado_a") B=$(<"$estado_b")" >&2
    fallar_concurrencia_o404e "$etiqueta: no hubo exactamente un primer COMMIT"
    return 1
  fi
  if (( commit_a == 1 )); then
    o404e_ganador=a
    codigo=$(sqlstate_o404e "$salida_b")
    [[ $(<"$estado_a") == 0 ]] ||
      fallar_concurrencia_o404e "$etiqueta: COMMIT A devolvió error"
  else
    o404e_ganador=b
    codigo=$(sqlstate_o404e "$salida_a")
    [[ $(<"$estado_b") == 0 ]] ||
      fallar_concurrencia_o404e "$etiqueta: COMMIT B devolvió error"
  fi
  if [[ ! $codigo =~ ^($codigo_perdedor)$ ]]; then
    printf 'SQLSTATE observado: %s\n' "${codigo:-ausente}" >&2
    fallar_concurrencia_o404e \
      "$etiqueta: SQLSTATE fuera del contrato $codigo_perdedor"
    return 1
  fi
}
reconciliar_exacto_o404e() {
  local caso=$1 expectativa=${2:-exacta}
  ejecutar_acotado_o404e 17s 1s docker exec \
    --env PGAPPNAME=o404e_lector_nuevo --interactive "$contenedor" \
    psql -X --set ON_ERROR_STOP=1 --quiet \
    --username vec_o404e_tcb --dbname postgres \
    --set "o404e_caso=$caso" --set "o404e_expectativa=$expectativa" \
    <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT pg_catalog.set_config(
  'vec.prueba_o404e_caso', :'o404e_caso', true);
SELECT pg_catalog.set_config(
  'vec.prueba_o404e_expectativa', :'o404e_expectativa', true);
DO $reconciliacion$
DECLARE
  v_caso text:=pg_catalog.current_setting('vec.prueba_o404e_caso');
  v_expectativa text:=pg_catalog.current_setting(
    'vec.prueba_o404e_expectativa');
  c jsonb:=vec_o404e_concedida.carga(v_caso)->'cabecera';
  r jsonb;
BEGIN
  IF v_expectativa IS DISTINCT FROM 'exacta'
     AND v_expectativa IS DISTINCT FROM 'ausente' THEN
    RAISE EXCEPTION USING ERRCODE='Z04E2',
      MESSAGE='expectativa de reconciliación adversaria inválida';
  END IF;
  SELECT resultado_json INTO STRICT r
  FROM vec_contratacion_temporal
    .leer_terminal_primario_decision_cobertura_o404e_v1(
      jsonb_build_object(
        'esquema',
          'vec.contratacion-temporal.consulta-primaria-decision-cobertura.o4-04e.v1',
        'organizacion_ref',c->>'organizacion_ref',
        'expediente_ref',c->>'expediente_ref',
        'version_expediente',c->'version_expediente',
        'reserva_ref',c->>'reserva_ref','recibo_ref',c->>'recibo_ref',
        'correlacion_vec_ref',c->>'correlacion_vec_ref',
        'decision_vec_ref',c->>'decision_vec_ref',
        'revision_cercado',c->'revision_cercado',
        'huella_orden_sha256',c->>'huella_orden_sha256'));
  IF (v_expectativa IS NOT DISTINCT FROM 'exacta' AND (
        r->>'encontrado' IS DISTINCT FROM 'true'
        OR r->'recibo' IS DISTINCT FROM
          vec_o404e_concedida.recibo(v_caso)))
     OR (v_expectativa IS NOT DISTINCT FROM 'ausente'
         AND r->>'encontrado'
          IS DISTINCT FROM 'false') THEN
    RAISE EXCEPTION USING ERRCODE='Z04E2',
      MESSAGE='reconciliación primaria adversaria no exacta';
  END IF;
END
$reconciliacion$;
COMMIT;
SQL
}
snapshot_identidad_o404e() {
  local caso=$1 control=${2:-si}
  psql_admin --tuples-only --no-align --set "o404e_caso=$caso" \
    --set "o404e_control=$control" <<'SQL'
WITH i AS (
 SELECT vec_o404e_concedida.carga(:'o404e_caso')->'cabecera' c
), s AS (
 SELECT jsonb_build_object(
  'decision_vec',(SELECT coalesce(jsonb_agg(to_jsonb(x)
    ORDER BY to_jsonb(x)::text),'[]') FROM
    vec_autorizacion.decision_concedida_contexto_actor_v3 x
    WHERE x.decision_ref=i.c->>'decision_vec_ref'),
 'acreditacion',(SELECT coalesce(jsonb_agg(to_jsonb(x)
    ORDER BY to_jsonb(x)::text),'[]') FROM
    vec_contratacion_temporal.acreditacion_gobierno_decision_cobertura x
    WHERE x.reserva_ref=i.c->>'reserva_ref'),
  'fixture',(SELECT coalesce(jsonb_agg(to_jsonb(x)
    ORDER BY to_jsonb(x)::text),'[]') FROM vec_o404e_concedida.cargas x
    WHERE x.caso=:'o404e_caso'),
  'reserva',(SELECT coalesce(jsonb_agg(to_jsonb(x)
    ORDER BY to_jsonb(x)::text),'[]') FROM
    vec_contratacion_temporal.reserva_operacion_decision_cobertura x
    WHERE x.reserva_ref=i.c->>'reserva_ref'),
  'alias',(SELECT coalesce(jsonb_agg(to_jsonb(a)
    ORDER BY to_jsonb(a)::text),'[]') FROM
    vec_contratacion_temporal.alias_operacion_decision_cobertura a
    JOIN vec_contratacion_temporal.reserva_operacion_decision_cobertura b
      USING(ambito_raiz_hmac) WHERE b.reserva_ref=i.c->>'reserva_ref'),
  'reserva_version',(SELECT coalesce(jsonb_agg(to_jsonb(v)
    ORDER BY to_jsonb(v)::text),'[]') FROM
    vec_contratacion_temporal.reserva_operacion_decision_cobertura_version v
    JOIN vec_contratacion_temporal.reserva_operacion_decision_cobertura b
      USING(ambito_raiz_hmac) WHERE b.reserva_ref=i.c->>'reserva_ref'),
  'reserva_actual',(SELECT coalesce(jsonb_agg(to_jsonb(a)
    ORDER BY to_jsonb(a)::text),'[]') FROM
    vec_contratacion_temporal.reserva_operacion_decision_cobertura_actual a
    JOIN vec_contratacion_temporal.reserva_operacion_decision_cobertura b
      USING(ambito_raiz_hmac) WHERE b.reserva_ref=i.c->>'reserva_ref'),
  'versiones',(SELECT coalesce(jsonb_agg(to_jsonb(x)
    ORDER BY to_jsonb(x)::text),'[]') FROM
    vec_contratacion_temporal.expediente_version_integral x
    WHERE x.operacion_ref=i.c->>'actuacion_ref'),
  'actual',(SELECT coalesce(jsonb_agg(to_jsonb(x)
    ORDER BY to_jsonb(x)::text),'[]') FROM
    vec_contratacion_temporal.expediente_integral_actual x
    WHERE x.expediente_ref=i.c->>'expediente_ref'),
  'actuacion',(SELECT coalesce(jsonb_agg(to_jsonb(x)
    ORDER BY to_jsonb(x)::text),'[]') FROM
    vec_contratacion_temporal.actuacion_expediente_integral x
    WHERE x.operacion_ref=i.c->>'actuacion_ref'),
  'decision_ct',(SELECT coalesce(jsonb_agg(to_jsonb(x)
    ORDER BY to_jsonb(x)::text),'[]') FROM
    vec_contratacion_temporal.decision_cobertura_gobernada_durable x
    WHERE x.reserva_ref=i.c->>'reserva_ref'),
  'auditoria',(SELECT coalesce(jsonb_agg(to_jsonb(x)
    ORDER BY to_jsonb(x)::text),'[]') FROM
    vec_contratacion_temporal.auditoria_decision_cobertura x
    WHERE x.reserva_ref=i.c->>'reserva_ref'),
  'outbox',(SELECT coalesce(jsonb_agg(to_jsonb(x)
    ORDER BY to_jsonb(x)::text),'[]') FROM
    vec_contratacion_temporal.outbox_expediente_integral x
    WHERE x.evento_ref=i.c->>'evento_ref'),
  'control_cadenas',(SELECT to_jsonb(x) FROM
    vec_contratacion_temporal.control_cadenas_expediente_integral x),
  'confirmacion',(SELECT coalesce(jsonb_agg(to_jsonb(x)
    ORDER BY to_jsonb(x)::text),'[]') FROM
    vec_contratacion_temporal.confirmacion_operacion_decision_cobertura x
    WHERE x.reserva_ref=i.c->>'reserva_ref'),
  'terminal',(SELECT coalesce(jsonb_agg(to_jsonb(t)
    ORDER BY to_jsonb(t)::text),'[]') FROM
    vec_contratacion_temporal.terminal_operacion_decision_cobertura t
    JOIN vec_contratacion_temporal.reserva_operacion_decision_cobertura b
      USING(ambito_raiz_hmac)
    WHERE b.reserva_ref=i.c->>'reserva_ref'),
  'lotes',(SELECT coalesce(jsonb_agg(to_jsonb(x)
    ORDER BY to_jsonb(x)::text),'[]') FROM
    vec_contratacion_temporal.consumo_cobertura_lote x
    WHERE x.reserva_ref=i.c->>'reserva_ref'),
  'evidencias',(SELECT coalesce(jsonb_agg(to_jsonb(e)
    ORDER BY to_jsonb(e)::text),'[]') FROM
    vec_contratacion_temporal.consumo_cobertura_evidencia e
    JOIN vec_contratacion_temporal.consumo_cobertura_lote l USING(lote_ref)
    WHERE l.reserva_ref=i.c->>'reserva_ref'),
  'enlace',(SELECT coalesce(jsonb_agg(to_jsonb(x)
    ORDER BY to_jsonb(x)::text),'[]') FROM
    vec_autorizacion.enlace_decision_cobertura_ct_o404e x
    WHERE x.decision_ref=i.c->>'decision_vec_ref')
 ) j FROM i
)
SELECT CASE :'o404e_control'
       WHEN 'si' THEN j
       WHEN 'producto' THEN j-'control_cadenas'-'decision_vec'-'enlace'
       WHEN 'rollback' THEN j-'control_cadenas'
       ELSE j-'control_cadenas' END FROM s;
SQL
}
snapshot_cadenas_o404e() {
  psql_admin --tuples-only --no-align --command \
    "SELECT to_jsonb(x) FROM
       vec_contratacion_temporal.control_cadenas_expediente_integral x" \
    </dev/null
}
exigir_snapshot_igual_o404e() {
  local anterior=$1 posterior=$2 etiqueta=$3
  if [[ $anterior != "$posterior" ]]; then
    fallar_concurrencia_o404e "snapshot divergente: $etiqueta"
    return 1
  fi
}
