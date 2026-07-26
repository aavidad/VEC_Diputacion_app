#!/usr/bin/env bash
# Se carga desde probar_o4_04e_pg18_4.sh.
# shellcheck disable=SC2154
source "$raiz/deploy/postgresql/contratacion_temporal/pruebas_o4_04e_retirada.sh"
snapshot_fallos_o404e() {
  docker exec "$contenedor" psql -X -A -t --set ON_ERROR_STOP=1 \
    --username postgres --dbname postgres \
    --file /repo/contratacion_temporal/pruebas_sql/o404e_snapshot_fallos.sql
}
probar_fallo_despues_escritura() {
  local rama=$1
  local tabla=$2
  local evento=$3
  local condicion=${4:-}
  local caso=${5:-uno}
  local identidad=${6:-eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee}
  local cantidad=${7:-1}
  local cuando=
  local salida snapshot_anterior snapshot_posterior
  local es_denegada=false
  [[ $rama == denegada ]] && es_denegada=true
  printf '[O4-04E:PG18.4]   %s %s %s %s\n' \
    "$rama" "$evento" "$tabla" "${condicion:-todas}"
  if [[ -n $condicion ]]; then
    cuando="WHEN ($condicion)"
  fi
  snapshot_anterior=$(snapshot_fallos_o404e)
  psql_admin --command \
    "CREATE TRIGGER o404e_fallo AFTER $evento ON $tabla FOR EACH ROW $cuando EXECUTE FUNCTION vec_o404e_fallos.despues_escritura()" \
    </dev/null >/dev/null
  if salida="$(docker exec --interactive "$contenedor" psql -X \
    --set ON_ERROR_STOP=1 \
    --set "o404e_denegada=$es_denegada" --set "o404e_caso=$caso" \
    --set "o404e_identidad=$identidad" \
    --set "o404e_cantidad=$cantidad" \
    --quiet --username postgres \
    --dbname postgres 2>&1 <<'SQL'
\set VERBOSITY verbose
SET SESSION AUTHORIZATION vec_o404e_tcb;
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
\if :o404e_denegada
SELECT vec_o404e_prueba.preparar_denegacion(
  :'o404e_identidad'
) AS carga
\else
SELECT vec_o404e_concedida.preparar(
  :'o404e_caso',
  :'o404e_cantidad'::integer
) AS carga
\endif
\gset
SELECT * FROM vec_contratacion_temporal
  .confirmar_operacion_decision_cobertura_o404e_v1(:'carga'::jsonb);
SQL
  )"; then
    printf 'failpoint no alcanzado: %s %s %s\n' \
      "$rama" "$evento" "$tabla" >&2
    return 1
  fi
  if [[ $salida != *'ERROR:  Z04E1:'* ]]; then
    printf '%s\n' "$salida" >&2
    return 1
  fi
  psql_admin --command "DROP TRIGGER o404e_fallo ON $tabla" \
    </dev/null >/dev/null
  snapshot_posterior=$(snapshot_fallos_o404e)
  if [[ $snapshot_anterior != "$snapshot_posterior" ]]; then
    printf 'failpoint %s %s %s alteró snapshot completo\n' \
      "$rama" "$evento" "$tabla" >&2
    return 1
  fi
}

paso 'failpoints AFTER denegados: helpers indirectos y snapshot completo'
indice=0
while IFS='|' read -r tabla evento condicion; do
  indice=$((indice+1))
  printf -v identidad '%032x' "$indice"
  probar_fallo_despues_escritura \
    denegada "$tabla" "$evento" "$condicion" uno "$identidad"
done <<'PUNTOS'
vec_autorizacion.decision_denegada_contexto_actor_v3|INSERT|
vec_autorizacion.enlace_decision_cobertura_ct_o404e|INSERT|
vec_contratacion_temporal.prueba_denegacion_decision_cobertura|INSERT|
vec_contratacion_temporal.auditoria_decision_cobertura|INSERT|
vec_contratacion_temporal.outbox_expediente_integral|INSERT|
vec_contratacion_temporal.control_cadenas_expediente_integral|UPDATE|
vec_contratacion_temporal.confirmacion_operacion_decision_cobertura|INSERT|
vec_contratacion_temporal.reserva_operacion_decision_cobertura_version|INSERT|NEW.estado='denegada_vec'
vec_contratacion_temporal.terminal_operacion_decision_cobertura|INSERT|
vec_contratacion_temporal.reserva_operacion_decision_cobertura_actual|UPDATE|
PUNTOS

docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
  --quiet --username vec_o404e_tcb --dbname postgres <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT vec_o404e_prueba.preparar_denegacion();
COMMIT;
SQL

probar_credencial_combinada_fachada() {
  local rol=$1
  psql_admin --command "GRANT $rol TO vec_o404e_tcb" </dev/null >/dev/null
  docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --quiet --username vec_o404e_tcb --dbname postgres <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
DO $rechazo$
BEGIN
  BEGIN
    PERFORM * FROM vec_contratacion_temporal
      .confirmar_operacion_decision_cobertura_o404e_v1(
        vec_o404e_prueba.carga()
      );
    RAISE EXCEPTION 'credencial combinada alcanzó confirmación';
  EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
  END;
END
$rechazo$;
ROLLBACK;
SQL
  psql_admin --command "REVOKE $rol FROM vec_o404e_tcb" \
    </dev/null >/dev/null
}

paso 'fachada rechaza confirmador combinado con ejecutor o gobernador'
probar_credencial_combinada_fachada vec_contratacion_temporal_ejecutor
probar_credencial_combinada_fachada vec_contratacion_temporal_gobernador
probar_credencial_combinada_fachada vec_contratacion_temporal_migrador
probar_credencial_combinada_fachada vec_autorizacion_registro

confirmar_denegada_primera() {
  local salida
  if salida="$(docker exec --interactive "$contenedor" \
    psql -X --set ON_ERROR_STOP=1 --quiet \
    --username vec_o404e_tcb --dbname postgres 2>&1 <<'SQL'
\set VERBOSITY verbose
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
DO $carrera$
DECLARE
  v_recibo jsonb;
BEGIN
  SELECT recibo_json INTO STRICT v_recibo
    FROM vec_contratacion_temporal
      .confirmar_operacion_decision_cobertura_o404e_v1(
        vec_o404e_prueba.carga()
      );
  PERFORM vec_o404e_prueba.guardar_recibo(v_recibo);
  IF v_recibo IS DISTINCT FROM vec_o404e_prueba.recibo() THEN
    RAISE EXCEPTION 'carrera denegada produjo recibos divergentes';
  END IF;
END
$carrera$;
COMMIT;
SQL
  )"; then
    return
  fi
  if [[ $salida != *'ERROR:  40001:'* ]]; then
    printf '%s\n' "$salida" >&2
    return 1
  fi
  docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --quiet --username vec_o404e_tcb --dbname postgres <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
DO $reconciliar$
DECLARE
  v_resultado jsonb;
BEGIN
  SELECT resultado_json INTO STRICT v_resultado
    FROM vec_contratacion_temporal
      .leer_terminal_primario_decision_cobertura_o404e_v1(
        jsonb_build_object(
          'esquema',
            'vec.contratacion-temporal.consulta-primaria-decision-cobertura.o4-04e.v1',
          'organizacion_ref','organizacion:o404e',
          'expediente_ref','expediente:o404e:denegado',
          'version_expediente',2,
          'reserva_ref','reserva:o404e:denegada',
          'recibo_ref','recibo:o404e:denegada',
          'correlacion_vec_ref',
            'correlacion_eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee',
          'decision_vec_ref','decision:o404e:denegada',
          'revision_cercado',1,
          'huella_orden_sha256',repeat('d',64)
        )
      );
  IF v_resultado->>'encontrado' IS DISTINCT FROM 'true'
     OR v_resultado->'recibo' IS DISTINCT FROM vec_o404e_prueba.recibo()
  THEN
    RAISE EXCEPTION 'reconciliación denegada no fue JSON exacta';
  END IF;
END
$reconciliar$;
COMMIT;
SQL
}

paso 'carrera de primera confirmación denegada y reconciliación primaria'
pids=()
for _ in {1..4}; do
  confirmar_denegada_primera &
  pids+=("$!")
done
for pid in "${pids[@]}"; do
  wait "$pid"
done
psql_admin <<'SQL' >/dev/null
DO $unicidad_denegada$
BEGIN
  IF (SELECT count(*) FROM
        vec_contratacion_temporal.confirmacion_operacion_decision_cobertura
       WHERE reserva_ref='reserva:o404e:denegada')<>1
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.terminal_operacion_decision_cobertura t
        JOIN vec_contratacion_temporal
          .reserva_operacion_decision_cobertura b USING(ambito_raiz_hmac)
       WHERE b.reserva_ref='reserva:o404e:denegada')<>1
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.prueba_denegacion_decision_cobertura p
        JOIN vec_contratacion_temporal
          .reserva_operacion_decision_cobertura b USING(ambito_raiz_hmac)
       WHERE b.reserva_ref='reserva:o404e:denegada')<>1
     OR EXISTS(SELECT 1 FROM
        vec_contratacion_temporal.acreditacion_gobierno_decision_cobertura
       WHERE reserva_ref='reserva:o404e:denegada')
     OR EXISTS(SELECT 1 FROM
        vec_contratacion_temporal.consumo_cobertura_lote
       WHERE reserva_ref='reserva:o404e:denegada')
     OR EXISTS(SELECT 1 FROM
        vec_contratacion_temporal.decision_cobertura_gobernada_durable
       WHERE reserva_ref='reserva:o404e:denegada')
     OR EXISTS(SELECT 1 FROM
        vec_contratacion_temporal.actuacion_expediente_integral
       WHERE operacion_ref='actuacion:o404e:denegada') THEN
    RAISE EXCEPTION 'carrera denegada duplicó o fabricó efectos concedidos';
  END IF;
END
$unicidad_denegada$;
SQL

probar_credencial_combinada_lector() {
  local rol=$1
  psql_admin --command "GRANT $rol TO vec_o404e_tcb" </dev/null >/dev/null
  docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --quiet --username vec_o404e_tcb --dbname postgres <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
DO $rechazo$
BEGIN
  BEGIN
    PERFORM * FROM vec_contratacion_temporal
      .leer_terminal_primario_decision_cobertura_o404e_v1(
        jsonb_build_object(
          'esquema',
            'vec.contratacion-temporal.consulta-primaria-decision-cobertura.o4-04e.v1',
          'organizacion_ref','organizacion:o404e',
          'expediente_ref','expediente:o404e:denegado',
          'version_expediente',2,
          'reserva_ref','reserva:o404e:denegada',
          'recibo_ref','recibo:o404e:denegada',
          'correlacion_vec_ref',
            'correlacion_eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee',
          'decision_vec_ref','decision:o404e:denegada',
          'revision_cercado',1,
          'huella_orden_sha256',repeat('d',64)
        )
      );
    RAISE EXCEPTION 'credencial combinada alcanzó lector';
  EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
  END;
END
$rechazo$;
ROLLBACK;
SQL
  psql_admin --command "REVOKE $rol FROM vec_o404e_tcb" \
    </dev/null >/dev/null
}

paso 'lector rechaza confirmador combinado con ejecutor o gobernador'
probar_credencial_combinada_lector vec_contratacion_temporal_ejecutor
probar_credencial_combinada_lector vec_contratacion_temporal_gobernador
probar_credencial_combinada_lector vec_contratacion_temporal_migrador
probar_credencial_combinada_lector vec_autorizacion_registro

psql_admin --set o404e_solo_casos=1 --set o404e_saltar_primera=1 \
  --file /repo/contratacion_temporal/pruebas_sql/o404e_denegacion_replay.sql \
  >/dev/null

paso 'credencial gobernadora y golden concedido C1=1/512'
psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_o404e_gob LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_gobernador TO vec_o404e_gob
 WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
archivo contratacion_temporal/pruebas_sql/o404e_concesion_golden.sql
archivo contratacion_temporal/pruebas_sql/o404e_helpers_concurrencia.sql
source "$raiz/deploy/postgresql/contratacion_temporal/pruebas_o4_04e_concurrencia_adversaria.sh"

paso 'failpoints AFTER concedidos: snapshot completo y extremos C1=512'
indice=0
while IFS='|' read -r tabla evento condicion caso; do
  indice=$((indice+1))
  printf -v identidad_caso 'fp_c_%02d' "$indice"
  probar_fallo_despues_escritura \
    concedida "$tabla" "$evento" "$condicion" "$identidad_caso" \
    eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee \
    "$([[ $caso == maximo ]] && printf 512 || printf 1)"
done <<'PUNTOS'
vec_autorizacion.decision_concedida_contexto_actor_v3|INSERT|
vec_autorizacion.enlace_decision_cobertura_ct_o404e|INSERT|
vec_contratacion_temporal.consumo_cobertura_lote|INSERT|
vec_contratacion_temporal.consumo_cobertura_evidencia|INSERT|NEW.posicion=1|maximo
vec_contratacion_temporal.consumo_cobertura_evidencia|INSERT|NEW.posicion=512|maximo
vec_contratacion_temporal.acreditacion_gobierno_decision_cobertura|INSERT|
vec_contratacion_temporal.expediente_version_integral|INSERT|NEW.version=3
vec_contratacion_temporal.expediente_integral_actual|UPDATE|
vec_contratacion_temporal.actuacion_expediente_integral|INSERT|
vec_contratacion_temporal.decision_cobertura_gobernada_durable|INSERT|
vec_contratacion_temporal.auditoria_decision_cobertura|INSERT|
vec_contratacion_temporal.outbox_expediente_integral|INSERT|
vec_contratacion_temporal.control_cadenas_expediente_integral|UPDATE|
vec_contratacion_temporal.confirmacion_operacion_decision_cobertura|INSERT|
vec_contratacion_temporal.reserva_operacion_decision_cobertura_version|INSERT|NEW.estado='aplicada'
vec_contratacion_temporal.terminal_operacion_decision_cobertura|INSERT|
vec_contratacion_temporal.reserva_operacion_decision_cobertura_actual|UPDATE|
PUNTOS

archivo contratacion_temporal/pruebas_sql/o404e_concesion_casos.sql

confirmar_maximo_primera() {
  local salida
  if salida="$(docker exec --interactive "$contenedor" \
    psql -X --set ON_ERROR_STOP=1 --quiet \
    --username vec_o404e_tcb --dbname postgres 2>&1 <<'SQL'
\set VERBOSITY verbose
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
DO $carrera$
DECLARE
  v_recibo jsonb;
BEGIN
  SELECT recibo_json INTO STRICT v_recibo
    FROM vec_contratacion_temporal
      .confirmar_operacion_decision_cobertura_o404e_v1(
        vec_o404e_concedida.carga('maximo')
      );
  PERFORM vec_o404e_concedida.guardar_recibo('maximo',v_recibo);
  IF v_recibo IS DISTINCT FROM vec_o404e_concedida.recibo('maximo') THEN
    RAISE EXCEPTION 'carrera inicial C1=512 produjo recibos divergentes';
  END IF;
END
$carrera$;
COMMIT;
SQL
  )"; then
    return
  fi
  if [[ $salida != *'ERROR:  40001:'* ]]; then
    printf '%s\n' "$salida" >&2
    return 1
  fi
  docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --quiet --username vec_o404e_tcb --dbname postgres <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
DO $reconciliar$
DECLARE
  c jsonb:=vec_o404e_concedida.carga('maximo')->'cabecera';
  v_resultado jsonb;
BEGIN
  SELECT resultado_json INTO STRICT v_resultado
    FROM vec_contratacion_temporal
      .leer_terminal_primario_decision_cobertura_o404e_v1(
        pg_catalog.jsonb_build_object(
          'esquema',
            'vec.contratacion-temporal.consulta-primaria-decision-cobertura.o4-04e.v1',
          'organizacion_ref',c->>'organizacion_ref',
          'expediente_ref',c->>'expediente_ref',
          'version_expediente',(c->>'version_expediente')::numeric,
          'reserva_ref',c->>'reserva_ref','recibo_ref',c->>'recibo_ref',
          'correlacion_vec_ref',c->>'correlacion_vec_ref',
          'decision_vec_ref',c->>'decision_vec_ref',
          'revision_cercado',(c->>'revision_cercado')::numeric,
          'huella_orden_sha256',c->>'huella_orden_sha256'
        )
      );
  IF v_resultado->>'encontrado' IS DISTINCT FROM 'true'
     OR v_resultado->'recibo' IS DISTINCT FROM
        vec_o404e_concedida.recibo('maximo') THEN
    RAISE EXCEPTION 'reconciliación C1=512 no fue JSON exacta';
  END IF;
END
$reconciliar$;
COMMIT;
SQL
}

paso 'carrera concedida C1=512 serializada antes de retirada real'
preparar_confirmacion_posterior_retirada_o404e
preparar_retirada_concurrente_o404e
pids=()
for _ in {1..4}; do
  confirmar_maximo_primera &
  pids+=("$!")
done
lanzar_retirada_concurrente_o404e
for pid in "${pids[@]}"; do
  wait "$pid"
done
verificar_retirada_concurrente_o404e
verificar_confirmacion_posterior_retirada_o404e
psql_admin <<'SQL' >/dev/null
DO $unicidad$
BEGIN
  IF (SELECT count(*) FROM
        vec_contratacion_temporal.confirmacion_operacion_decision_cobertura
       WHERE reserva_ref='reserva:o404e:maximo')<>1
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.terminal_operacion_decision_cobertura t
        JOIN vec_contratacion_temporal
          .reserva_operacion_decision_cobertura b USING(ambito_raiz_hmac)
       WHERE b.reserva_ref='reserva:o404e:maximo')<>1
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.decision_cobertura_gobernada_durable
       WHERE reserva_ref='reserva:o404e:maximo')<>1
     OR (SELECT count(*) FROM
        vec_contratacion_temporal.consumo_cobertura_evidencia e
        JOIN vec_contratacion_temporal.consumo_cobertura_lote l
          USING(lote_ref)
       WHERE l.reserva_ref='reserva:o404e:maximo')<>512 THEN
    RAISE EXCEPTION 'carrera inicial C1=512 duplicó efectos';
  END IF;
END
$unicidad$;
SQL

source "$raiz/deploy/postgresql/contratacion_temporal/pruebas_o4_04e_replays.sh"
