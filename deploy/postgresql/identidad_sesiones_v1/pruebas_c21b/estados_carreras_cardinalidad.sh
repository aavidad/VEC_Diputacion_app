#!/usr/bin/env bash

crear_sesion_ct85() {
  local etiqueta=$1

  psql_valor "SELECT cuenta_ref FROM
vec_identidad_sesiones_v1.provisionar_cuenta_v1(
'opr_'||substr(encode(sha256(convert_to('$etiqueta-provision','UTF8')),'hex'),1,24),
'vec.identidad.hmac-sha256.v1','idh_aaaaaaaaaaaaaaaaaaaaaaaa',
'clave-hsm-prueba',1,
sha256(convert_to('$etiqueta-cuenta','UTF8')),
sha256(convert_to('$etiqueta-sujeto','UTF8')),false,NULL)" >/dev/null
  psql_valor "SELECT autenticacion_ref||'|'||sesion_ref||'|'||
control_sesion_ref||'|'||control_sesion_revision_texto||'|'||cuenta_ref
FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
'opr_'||substr(encode(sha256(convert_to('$etiqueta-registro','UTF8')),'hex'),1,24),
'vec.identidad.hmac-sha256.v1','idh_aaaaaaaaaaaaaaaaaaaaaaaa',
'clave-hsm-prueba',1,
sha256(convert_to('$etiqueta-asercion','UTF8')),
sha256(convert_to('$etiqueta-sesion','UTF8')),
sha256(convert_to('$etiqueta-sujeto','UTF8')),
sha256(convert_to('$etiqueta-cuenta','UTF8')),NULL,false,
'interna_corporativa','kerberos_ad','alto',
encode(sha256(convert_to('$etiqueta-autenticacion','UTF8')),'hex'),
date_trunc('microseconds',clock_timestamp()-interval '2 seconds'),
date_trunc('microseconds',clock_timestamp()-interval '1 second'),
date_trunc('microseconds',clock_timestamp()+interval '4 minutes'),
'pga_aaaaaaaaaaaaaaaaaaaaaaaa',repeat('b',64))"
}

probar_estados_carreras_cardinalidad() {
  local contenedor=$1
  local base=$2
  local clave=$3
  local login=$4
  local proxy=$5
  local salidas=$6
  local autenticacion_expirada sesion_expirada
  local autenticacion_frontera sesion_frontera
  local autenticacion_cuenta sesion_cuenta cuenta_cuenta
  local autenticacion_primera sesion_primera cuenta_primera
  local autenticacion_segunda sesion_segunda cuenta_segunda
  local autenticacion_cardinal sesion_cardinal cuenta_cardinal
  local fifo_actor fifo_revoca pid_actor pid_revoca
  local fd_actor fd_revoca estado_actor

  paso 'caducidad y frontera semiabierta de sesion_valida_hasta'
  IFS='|' read -r autenticacion_expirada sesion_expirada _ _ _ \
    <<<"$(crear_sesion_ct85 caducidad)"
  psql_valor "SET session_replication_role=replica;
UPDATE vec_autorizacion.control_sesion_v1
SET sesion_revalidada_en=date_trunc(
      'microseconds',clock_timestamp()-interval '2 seconds'),
    sesion_valida_hasta=date_trunc(
      'microseconds',clock_timestamp()-interval '1 second')
WHERE sesion_ref='$sesion_expirada'; RESET session_replication_role" \
    >/dev/null
  exigir_cero "$autenticacion_expirada" "$sesion_expirada" \
    'sesion_valida_hasta caducada'

  IFS='|' read -r autenticacion_frontera sesion_frontera _ _ _ \
    <<<"$(crear_sesion_ct85 frontera)"
  psql_valor "SET session_replication_role=replica;
UPDATE vec_autorizacion.control_sesion_v1
SET sesion_revalidada_en=date_trunc(
      'microseconds',clock_timestamp()-interval '1 second'),
    sesion_valida_hasta=date_trunc('microseconds',clock_timestamp())
WHERE sesion_ref='$sesion_frontera'; RESET session_replication_role" \
    >/dev/null
  exigir_cero "$autenticacion_frontera" "$sesion_frontera" \
    'frontera exacta y exclusiva de sesion_valida_hasta'

  paso 'cuenta inactiva y reactivada no resucita la sesion revocada'
  IFS='|' read -r autenticacion_cuenta sesion_cuenta _ _ cuenta_cuenta \
    <<<"$(crear_sesion_ct85 cuenta)"
  [[ $(psql_valor "SELECT vec_identidad_sesiones_v1.cambiar_estado_cuenta_v1(
'$cuenta_cuenta','1','inactiva','opr_'||repeat('u',24))") == 2 ]]
  exigir_cero "$autenticacion_cuenta" "$sesion_cuenta" 'cuenta inactiva'
  [[ $(psql_valor "SELECT vec_identidad_sesiones_v1.cambiar_estado_cuenta_v1(
'$cuenta_cuenta','2','activa','opr_'||repeat('v',24))") == 3 ]]
  exigir_cero "$autenticacion_cuenta" "$sesion_cuenta" \
    'reactivacion no recupera la revision de cuenta de la sesion'

  paso 'carrera: revalidacion gana y revocacion de cuenta espera'
  IFS='|' read -r autenticacion_primera sesion_primera _ _ cuenta_primera \
    <<<"$(crear_sesion_ct85 carrera-revalidacion)"
  fifo_actor="$salidas/fifo_cuenta_actor_primero"
  mkfifo "$fifo_actor"
  docker exec -i --env PGPASSWORD="$clave" "$contenedor" psql -XAtq \
    -v ON_ERROR_STOP=1 -v VERBOSITY=verbose -h 127.0.0.1 \
    -U "$login" -d "$base" <"$fifo_actor" \
    >"$salidas/cuenta_actor_primero" 2>&1 &
  pid_actor=$!
  exec {fd_actor}>"$fifo_actor"
  printf "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET application_name='c21b_cuenta_actor_primero';
SELECT count(*) FROM %s('%s','%s');\n" \
    "$proxy" "$autenticacion_primera" "$sesion_primera" >&"$fd_actor"
  esperar_estado "SELECT count(*) FROM pg_stat_activity
WHERE application_name='c21b_cuenta_actor_primero'
AND state='idle in transaction'" 1 'revalidacion retiene lock de cuenta'
  (psql_valor "SET application_name='c21b_cuenta_revoca_despues';
SELECT vec_identidad_sesiones_v1.cambiar_estado_cuenta_v1(
'$cuenta_primera','1','inactiva','opr_'||repeat('w',24))" \
    >"$salidas/cuenta_revoca_despues" 2>&1) &
  pid_revoca=$!
  esperar_estado "SELECT count(*) FROM pg_stat_activity
WHERE application_name='c21b_cuenta_revoca_despues'
AND cardinality(pg_blocking_pids(pid))=1" 1 \
    'revocacion de cuenta espera revalidacion'
  printf 'COMMIT;\n' >&"$fd_actor"
  exec {fd_actor}>&-
  wait "$pid_actor"
  wait "$pid_revoca"
  grep -Fxq 1 "$salidas/cuenta_actor_primero"
  grep -Fxq 2 "$salidas/cuenta_revoca_despues"
  exigir_cero "$autenticacion_primera" "$sesion_primera" \
    'revocacion de cuenta posterior'

  paso 'carrera: revocacion de cuenta gana y revalidacion falla cerrada'
  IFS='|' read -r autenticacion_segunda sesion_segunda _ _ cuenta_segunda \
    <<<"$(crear_sesion_ct85 carrera-revocacion)"
  fifo_revoca="$salidas/fifo_cuenta_revoca_primero"
  mkfifo "$fifo_revoca"
  docker exec -i --env PGAPPNAME=c21b_cuenta_revoca_primero \
    "$contenedor" psql -XAtq -v ON_ERROR_STOP=1 -v VERBOSITY=verbose \
    -U postgres -d "$base" <"$fifo_revoca" \
    >"$salidas/cuenta_revoca_primero" 2>&1 &
  pid_revoca=$!
  exec {fd_revoca}>"$fifo_revoca"
  printf "BEGIN;
SELECT vec_identidad_sesiones_v1.cambiar_estado_cuenta_v1(
'%s','1','inactiva','opr_'||repeat('x',24));\n" \
    "$cuenta_segunda" >&"$fd_revoca"
  esperar_estado "SELECT count(*) FROM pg_stat_activity
WHERE application_name='c21b_cuenta_revoca_primero'
AND state='idle in transaction'" 1 'revocacion conserva lock de cuenta'
  (actor_valor "SET application_name='c21b_cuenta_actor_despues';
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT count(*) FROM $proxy('$autenticacion_segunda','$sesion_segunda');
COMMIT" >"$salidas/cuenta_actor_despues" 2>&1) &
  pid_actor=$!
  esperar_estado "SELECT count(*) FROM pg_stat_activity
WHERE application_name='c21b_cuenta_actor_despues'
AND cardinality(pg_blocking_pids(pid))=1" 1 \
    'revalidacion espera revocacion de cuenta'
  printf 'COMMIT;\n' >&"$fd_revoca"
  exec {fd_revoca}>&-
  wait "$pid_revoca"
  set +e
  wait "$pid_actor"
  estado_actor=$?
  set -e
  [[ $estado_actor -ne 0 ]]
  grep -Fq '40001' "$salidas/cuenta_actor_despues"

  paso 'cardinalidad hostil: dos punteros actuales para una cuenta'
  IFS='|' read -r autenticacion_cardinal sesion_cardinal _ _ cuenta_cardinal \
    <<<"$(crear_sesion_ct85 cardinalidad)"
  psql_admin <<SQL
ALTER TABLE vec_identidad_sesiones_v1.estado_cuenta_actual
  DISABLE TRIGGER estado_cuenta_actual_no_eliminar;
ALTER TABLE vec_identidad_sesiones_v1.estado_cuenta_actual
  DROP CONSTRAINT estado_cuenta_actual_pkey;
INSERT INTO vec_identidad_sesiones_v1.estado_cuenta_actual
SELECT * FROM vec_identidad_sesiones_v1.estado_cuenta_actual
WHERE cuenta_ref='$cuenta_cardinal';
SQL
  [[ $(psql_valor "SELECT count(*) FROM
vec_identidad_sesiones_v1.estado_cuenta_actual
WHERE cuenta_ref='$cuenta_cardinal'") == 2 ]]
  exigir_cero "$autenticacion_cardinal" "$sesion_cardinal" \
    'dos estados actuales fuerzan cardinalidad 2 cuando se espera 1'
  psql_admin <<SQL
DELETE FROM vec_identidad_sesiones_v1.estado_cuenta_actual
WHERE ctid IN (
  SELECT ctid FROM vec_identidad_sesiones_v1.estado_cuenta_actual
  WHERE cuenta_ref='$cuenta_cardinal' ORDER BY ctid OFFSET 1
);
ALTER TABLE vec_identidad_sesiones_v1.estado_cuenta_actual
  ADD CONSTRAINT estado_cuenta_actual_pkey PRIMARY KEY (cuenta_ref);
ALTER TABLE vec_identidad_sesiones_v1.estado_cuenta_actual
  ENABLE TRIGGER estado_cuenta_actual_no_eliminar;
SQL
  exigir_uno "$autenticacion_cardinal" "$sesion_cardinal" \
    'cardinalidad actual restaurada'
}
