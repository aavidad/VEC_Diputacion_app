#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-autorizacion-contexto-v3-${USER:-usuario}-$$"
base=vec_autorizacion_contexto_v3
clave=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
clave_registro=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')

limpiar() { docker rm -f "$contenedor" >/dev/null 2>&1 || true; }
trap limpiar EXIT INT TERM

docker run --detach --rm --name "$contenedor" --publish 127.0.0.1::5432 \
  --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave" "$imagen" >/dev/null

# La imagen oficial abre primero un PostgreSQL temporal para inicializar la
# base, lo apaga y después ejecuta el servidor definitivo. pg_isready puede
# observar esa ventana temporal y provocar un FATAL "system is shutting down".
# La marca del entrypoint fijado por digest acredita que esa fase terminó.
postgresql_definitivo_disponible=false
for _ in $(seq 1 120); do
  if ! docker inspect --format '{{.State.Running}}' "$contenedor" \
    2>/dev/null | grep -Fxq true; then
    break
  fi
  if docker logs "$contenedor" 2>&1 |
      LC_ALL=C grep -Fq 'PostgreSQL init process complete; ready for start up.' &&
    docker exec "$contenedor" psql -XAt --set ON_ERROR_STOP=1 \
      -U postgres -d "$base" -c 'SELECT 1' 2>/dev/null |
      grep -Fxq 1; then
    postgresql_definitivo_disponible=true
    break
  fi
  sleep 1
done
if [[ $postgresql_definitivo_disponible != true ]]; then
  docker logs "$contenedor" >&2 || true
  echo "PostgreSQL definitivo no quedó disponible" >&2
  exit 1
fi
version_mayor=$(docker exec "$contenedor" psql -XAt --set ON_ERROR_STOP=1 \
  -U postgres -d "$base" -c \
  "SELECT current_setting('server_version_num')::integer / 10000")
[[ $version_mayor == 18 ]] || {
  echo "se requiere PostgreSQL 18; inicio PostgreSQL $version_mayor" >&2
  exit 1
}

psql_archivo() {
  docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    -U postgres -d "$base" < "$raiz/$1"
}

docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
  -U postgres -d "$base" <<'SQL'
DO $b$ BEGIN
  EXECUTE format('REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',current_database());
END $b$;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL

psql_archivo deploy/postgresql/contexto_actor_v1/roles_up.sql
psql_archivo deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql
psql_archivo deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql
psql_archivo deploy/postgresql/contexto_actor_v1/pruebas_sql/fixtures_sinteticos.sql
psql_archivo deploy/postgresql/autorizacion/pruebas_sql/fixture_contexto_actor_v3.sql

docker exec --interactive --env CLAVE="$clave" "$contenedor" \
  psql -X --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
\getenv clave CLAVE
CREATE ROLE vec_contexto_v3_runtime_prueba LOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS PASSWORD :'clave';
GRANT vec_contexto_actor_v1_runtime TO vec_contexto_v3_runtime_prueba
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL
docker exec --interactive --env PGPASSWORD="$clave" "$contenedor" \
  psql -X --set ON_ERROR_STOP=1 -h 127.0.0.1 \
  -U vec_contexto_v3_runtime_prueba -d "$base" <<'SQL'
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SELECT count(*)
  FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
    'oca_registro_v3_000000000000000000000000',
    'rca_registro_v3_000000000000000000000000',
    'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',
    'prf_sintetico_cccccccccccccccccccccccc',
    'certificado','alto',clock_timestamp()
  );
COMMIT;
SQL

for f in \
  deploy/postgresql/autorizacion/roles_up.sql \
  deploy/postgresql/autorizacion/roles_v2_up.sql \
  deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql \
  deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql \
  deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql \
  deploy/postgresql/autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql \
  deploy/postgresql/autorizacion/migraciones/000005_registro_decisiones_contexto_actor_v3.up.sql \
  deploy/postgresql/autorizacion/migraciones/000006_funcion_registro_decisiones_contexto_actor_v3.up.sql \
  deploy/postgresql/autorizacion/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.up.sql
do
  psql_archivo "$f"
done

# Down inverso verde sin evidencia; despues, reapply idempotente y contrato.
psql_archivo deploy/postgresql/autorizacion/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.down.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000006_funcion_registro_decisiones_contexto_actor_v3.down.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000005_registro_decisiones_contexto_actor_v3.down.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000005_registro_decisiones_contexto_actor_v3.up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000006_funcion_registro_decisiones_contexto_actor_v3.up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000005_registro_decisiones_contexto_actor_v3.up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000006_funcion_registro_decisiones_contexto_actor_v3.up.sql

psql_archivo deploy/postgresql/autorizacion/pruebas_sql/fixture_autorizacion_contexto_actor_v3.sql
psql_archivo deploy/postgresql/autorizacion/pruebas_sql/integracion_contexto_actor_v3.sql

docker exec --interactive --env CLAVE_REGISTRO="$clave_registro" "$contenedor" \
  psql -X --set ON_ERROR_STOP=1 -U postgres -d "$base" <<'SQL'
\getenv clave_registro CLAVE_REGISTRO
CREATE ROLE vec_autorizacion_registro_v3_prueba LOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS PASSWORD :'clave_registro';
GRANT CONNECT ON DATABASE vec_autorizacion_contexto_v3
  TO vec_autorizacion_registro_v3_prueba;
GRANT vec_autorizacion_registro TO vec_autorizacion_registro_v3_prueba
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL

# Helper exclusivo de la base efimera: genera ordenes canonicas desde la fila
# sintetica ya comprobada. No recibe privilegios de runtime y se elimina antes
# de la auditoria final de ACL.
docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
  -U postgres -d "$base" <<'SQL'
CREATE FUNCTION public.probar_registro_contexto_actor_v3(
  p_decision_ref text,
  p_valida_hasta timestamptz DEFAULT NULL
) RETURNS integer
LANGUAGE plpgsql VOLATILE
SET search_path = pg_catalog
AS $f$
DECLARE
  base record;
  d jsonb;
  emitida timestamptz(6) := clock_timestamp();
  hasta timestamptz(6) := coalesce(
    p_valida_hasta, emitida + interval '2 minutes'
  );
  filas integer;
BEGIN
  SELECT documento, motivo_canonico INTO STRICT base
    FROM vec_autorizacion.decision_concedida_contexto_actor_v3
   WHERE decision_ref='decision:registro-v3:positiva';
  d := jsonb_set(base.documento,'{decision_ref}',to_jsonb(p_decision_ref));
  d := jsonb_set(d,'{emitida_en}',to_jsonb(to_char(
    emitida AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
  )));
  d := jsonb_set(d,'{valida_hasta}',to_jsonb(to_char(
    hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
  )));
  SELECT count(*) INTO filas
    FROM vec_autorizacion.registrar_decision_contexto_actor_v3(
      vec_autorizacion.decision_contexto_actor_v3_canonica(d),
      base.motivo_canonico,2,2
    );
  RETURN filas;
END
$f$;
REVOKE ALL ON FUNCTION public.probar_registro_contexto_actor_v3(
  text,timestamptz
) FROM PUBLIC, vec_autorizacion_registro;
SQL

psql_valor() {
  docker exec "$contenedor" psql -XAt --set ON_ERROR_STOP=1 \
    -U postgres -d "$base" -c "$1"
}

[[ $(psql_valor "SELECT (to_regprocedure('vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)') IS NOT NULL)::text") == true ]]
[[ $(psql_valor "SELECT has_function_privilege('vec_autorizacion_propietario','vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)','EXECUTE')::text") == true ]]
[[ $(psql_valor "SELECT has_function_privilege('vec_autorizacion_registro','vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)','EXECUTE')::text") == false ]]
[[ $(psql_valor "SELECT has_function_privilege('public','vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)','EXECUTE')::text") == false ]]

esperar_actividad() {
  local aplicacion=$1 esperadas=$2 modo=${3:-activa} numero
  for _ in $(seq 1 160); do
    if [[ $modo == espera ]]; then
      numero=$(psql_valor \
        "SELECT count(*) FROM pg_stat_activity WHERE application_name LIKE '${aplicacion}%' AND wait_event_type='Lock'")
    else
      numero=$(psql_valor \
        "SELECT count(*) FROM pg_stat_activity WHERE application_name LIKE '${aplicacion}%' AND state='active'")
    fi
    (( numero >= esperadas )) && return 0
  done
  docker exec "$contenedor" psql -X -U postgres -d "$base" -c \
    "SELECT application_name,state,wait_event_type,wait_event,left(query,120) FROM pg_stat_activity WHERE datname=current_database()" >&2
  echo "no se observaron $esperadas sesiones $modo para $aplicacion" >&2
  return 1
}

# Expiracion durante espera real del lock de identidad de decision. El holder se libera
# por clock_timestamp; el exito depende de observar el waiter y de que no haya
# fila, nunca de una pausa elegida por el shell.
echo 'concurrencia: caducidad bajo lock' >&2
limite=$(psql_valor \
  "SELECT to_char(clock_timestamp()+interval '4 seconds','YYYY-MM-DD HH24:MI:SS.USOF')")
docker exec --interactive --env PGAPPNAME=vec-v3-holder-expiracion "$contenedor" \
  psql -X --set ON_ERROR_STOP=1 -U postgres -d "$base" >/dev/null <<SQL &
BEGIN;
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_autorizacion:decision-v3:decision:registro-v3:expira-bajo-lock',0
));
DO \$b\$ BEGIN
  WHILE clock_timestamp() < '$limite'::timestamptz LOOP PERFORM 1; END LOOP;
END \$b\$;
COMMIT;
SQL
holder=$!
esperar_actividad vec-v3-holder-expiracion 1
docker exec --interactive --env PGAPPNAME=vec-v3-expiracion "$contenedor" \
  psql -X --set ON_ERROR_STOP=1 -U postgres -d "$base" >/dev/null <<SQL &
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
DO \$b\$ DECLARE filas integer; BEGIN
  filas := public.probar_registro_contexto_actor_v3(
    'decision:registro-v3:expira-bajo-lock','$limite'::timestamptz
  );
  IF filas <> 0 THEN RAISE EXCEPTION 'expiracion bajo lock concedio'; END IF;
END \$b\$;
COMMIT;
SQL
registrador=$!
esperar_actividad vec-v3-expiracion 1 espera
wait "$holder"
wait "$registrador"
[[ $(psql_valor "SELECT count(*) FROM vec_autorizacion.decision_concedida_contexto_actor_v3 WHERE decision_ref='decision:registro-v3:expira-bajo-lock'") == 0 ]]

# Dos registradores parten a la vez detras de la barrera global de contexto.
# Tras liberarla deben terminar sin 40P01. Un 40001 es valido y se reintenta
# en una transaccion nueva, como exige SERIALIZABLE.
echo 'concurrencia: dos registradores y reintento SERIALIZABLE' >&2
limite=$(psql_valor \
  "SELECT to_char(clock_timestamp()+interval '4 seconds','YYYY-MM-DD HH24:MI:SS.USOF')")
docker exec --interactive --env PGAPPNAME=vec-v3-holder-deadlock "$contenedor" \
  psql -X --set ON_ERROR_STOP=1 -U postgres -d "$base" >/dev/null <<SQL &
BEGIN;
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0
));
DO \$b\$ BEGIN
  WHILE clock_timestamp() < '$limite'::timestamptz LOOP PERFORM 1; END LOOP;
END \$b\$;
COMMIT;
SQL
holder=$!
esperar_actividad vec-v3-holder-deadlock 1
for sufijo in a b; do
  docker exec --interactive --env PGAPPNAME="vec-v3-deadlock-$sufijo" "$contenedor" \
    psql -X --set ON_ERROR_STOP=1 -U postgres -d "$base" >/dev/null <<SQL &
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL deadlock_timeout='100ms';
SET LOCAL statement_timeout='15s';
DO \$b\$ DECLARE filas integer; BEGIN
  BEGIN
    filas := public.probar_registro_contexto_actor_v3(
      'decision:registro-v3:deadlock-$sufijo'
    );
    IF filas <> 1 THEN RAISE EXCEPTION 'registro concurrente ausente'; END IF;
  EXCEPTION WHEN serialization_failure THEN NULL;
  END;
END \$b\$;
COMMIT;
SQL
  eval "registrador_$sufijo=$!"
done
esperar_actividad vec-v3-deadlock- 2 espera
wait "$holder"
wait "$registrador_a"
wait "$registrador_b"
for sufijo in a b; do
  if [[ $(psql_valor "SELECT count(*) FROM vec_autorizacion.decision_concedida_contexto_actor_v3 WHERE decision_ref='decision:registro-v3:deadlock-$sufijo'") == 0 ]]; then
    [[ $(psql_valor "BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE; SELECT public.probar_registro_contexto_actor_v3('decision:registro-v3:deadlock-$sufijo'); COMMIT") == $'BEGIN\n1\nCOMMIT' ]]
  fi
done
[[ $(psql_valor "SELECT count(*) FROM vec_autorizacion.decision_concedida_contexto_actor_v3 WHERE decision_ref LIKE 'decision:registro-v3:deadlock-%'") == 2 ]]

# Insercion fantasma de un puntero de modulo: la generacion/advisory hace que la
# acreditacion espere. El rollback del writer no deja fantasma ni generacion y
# el registro posterior debe usar la historia original.
echo 'concurrencia: puntero fantasma revertido' >&2
generacion_antes=$(psql_valor \
  "SELECT generacion FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2")
limite=$(psql_valor \
  "SELECT to_char(clock_timestamp()+interval '4 seconds','YYYY-MM-DD HH24:MI:SS.USOF')")
docker exec --interactive --env PGAPPNAME=vec-v3-holder-phantom "$contenedor" \
  psql -X --set ON_ERROR_STOP=1 -U postgres -d "$base" >/dev/null <<SQL &
BEGIN;
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones VALUES (
 'vin_fantasma_iiiiiiiiiiiiiiiiiiiiiiii',1,
 'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb','candidato',
 'can_fantasma_jjjjjjjjjjjjjjjjjjjjjjjj',
 'prc_maestra_sintetica_v3_00000001',1,repeat('a',64),
 'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',
 clock_timestamp()+interval '1 hour'
);
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_actual VALUES (
 'vin_fantasma_iiiiiiiiiiiiiiiiiiiiiiii',1
);
DO \$b\$ BEGIN
  WHILE clock_timestamp() < '$limite'::timestamptz LOOP PERFORM 1; END LOOP;
END \$b\$;
ROLLBACK;
SQL
holder=$!
esperar_actividad vec-v3-holder-phantom 1
docker exec --interactive --env PGAPPNAME=vec-v3-phantom "$contenedor" \
  psql -X --set ON_ERROR_STOP=1 -U postgres -d "$base" >/dev/null <<'SQL' &
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
DO $b$ DECLARE filas integer; BEGIN
  filas := public.probar_registro_contexto_actor_v3(
    'decision:registro-v3:phantom-rollback'
  );
  IF filas <> 1 THEN RAISE EXCEPTION 'rollback phantom no recupero'; END IF;
END $b$;
COMMIT;
SQL
registrador=$!
esperar_actividad vec-v3-phantom 1 espera
wait "$holder"
wait "$registrador"
[[ $(psql_valor "SELECT generacion FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2") == "$generacion_antes" ]]

# Prepara dos revisiones: una activa para avance concurrente y otra revocada.
docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
  -U postgres -d "$base" >/dev/null <<'SQL'
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones
SELECT cuenta_ref,3,procedencia_ref,procedencia_version,
       procedencia_huella_sha256,procedencia_autoridad,'activo',
       vigente_desde,vigente_hasta
  FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones
 WHERE cuenta_ref='cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa' AND version=2;
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones
SELECT cuenta_ref,4,procedencia_ref,procedencia_version,
       procedencia_huella_sha256,procedencia_autoridad,'revocado',
       vigente_desde,vigente_hasta
  FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones
 WHERE cuenta_ref='cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa' AND version=2;
SQL

# Avance comprometido mientras la acreditacion espera: nunca queda una
# concesion de la generacion obsoleta. Se admite el 40001 autoritativo.
echo 'concurrencia: avance de puntero comprometido' >&2
limite=$(psql_valor \
  "SELECT to_char(clock_timestamp()+interval '4 seconds','YYYY-MM-DD HH24:MI:SS.USOF')")
docker exec --interactive --env PGAPPNAME=vec-v3-holder-avance "$contenedor" \
  psql -X --set ON_ERROR_STOP=1 -U postgres -d "$base" >/dev/null <<SQL &
BEGIN;
UPDATE vec_contexto_actor_v1.proyeccion_cuenta_actual SET version=3
 WHERE cuenta_ref='cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa';
DO \$b\$ BEGIN
  WHILE clock_timestamp() < '$limite'::timestamptz LOOP PERFORM 1; END LOOP;
END \$b\$;
COMMIT;
SQL
holder=$!
esperar_actividad vec-v3-holder-avance 1
docker exec --interactive --env PGAPPNAME=vec-v3-avance "$contenedor" \
  psql -X --set ON_ERROR_STOP=1 -U postgres -d "$base" >/dev/null <<'SQL' &
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
DO $b$ DECLARE filas integer; BEGIN
  BEGIN
    filas := public.probar_registro_contexto_actor_v3(
      'decision:registro-v3:avance-concurrente'
    );
    IF filas <> 0 THEN RAISE EXCEPTION 'avance obsoleto concedio'; END IF;
  EXCEPTION WHEN serialization_failure THEN NULL;
  END;
END $b$;
COMMIT;
SQL
registrador=$!
esperar_actividad vec-v3-avance 1 espera
wait "$holder"
wait "$registrador"
[[ $(psql_valor "SELECT count(*) FROM vec_autorizacion.decision_concedida_contexto_actor_v3 WHERE decision_ref='decision:registro-v3:avance-concurrente'") == 0 ]]
psql_valor "UPDATE vec_contexto_actor_v1.proyeccion_cuenta_actual SET version=2 WHERE cuenta_ref='cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa'" >/dev/null
[[ $(psql_valor "BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE; SELECT public.probar_registro_contexto_actor_v3('decision:registro-v3:avance-concurrente'); COMMIT") == $'BEGIN\n1\nCOMMIT' ]]

# Revocacion comprometida en carrera. Después se demuestra que el replay de
# una fila previa sigue reconciliando aunque el contexto ya no sea utilizable.
echo 'concurrencia: revocacion y replay durable' >&2
limite=$(psql_valor \
  "SELECT to_char(clock_timestamp()+interval '4 seconds','YYYY-MM-DD HH24:MI:SS.USOF')")
docker exec --interactive --env PGAPPNAME=vec-v3-holder-revocacion "$contenedor" \
  psql -X --set ON_ERROR_STOP=1 -U postgres -d "$base" >/dev/null <<SQL &
BEGIN;
UPDATE vec_contexto_actor_v1.proyeccion_cuenta_actual SET version=4
 WHERE cuenta_ref='cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa';
DO \$b\$ BEGIN
  WHILE clock_timestamp() < '$limite'::timestamptz LOOP PERFORM 1; END LOOP;
END \$b\$;
COMMIT;
SQL
holder=$!
esperar_actividad vec-v3-holder-revocacion 1
docker exec --interactive --env PGAPPNAME=vec-v3-revocacion "$contenedor" \
  psql -X --set ON_ERROR_STOP=1 -U postgres -d "$base" >/dev/null <<'SQL' &
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
DO $b$ DECLARE filas integer; BEGIN
  BEGIN
    filas := public.probar_registro_contexto_actor_v3(
      'decision:registro-v3:revocacion-concurrente'
    );
    IF filas <> 0 THEN RAISE EXCEPTION 'revocacion obsoleta concedio'; END IF;
  EXCEPTION WHEN serialization_failure THEN NULL;
  END;
END $b$;
COMMIT;
SQL
registrador=$!
esperar_actividad vec-v3-revocacion 1 espera
wait "$holder"
wait "$registrador"
filas_revocacion=$(psql_valor "SELECT count(*) FROM vec_autorizacion.decision_concedida_contexto_actor_v3 WHERE decision_ref='decision:registro-v3:revocacion-concurrente'")
[[ $filas_revocacion == 0 ]] || {
  echo "revocacion concurrente dejo $filas_revocacion concesiones" >&2
  exit 1
}

replay=$(psql_valor \
  "BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE; SELECT count(*) FROM vec_autorizacion.decision_concedida_contexto_actor_v3 AS d CROSS JOIN LATERAL vec_autorizacion.registrar_decision_contexto_actor_v3(d.decision_canonica,d.motivo_canonico,d.persona_version,d.perfil_version) AS r WHERE d.decision_ref='decision:registro-v3:positiva'; COMMIT")
[[ $replay == $'BEGIN\n1\nCOMMIT' ]] || {
  echo "replay durable tras revocacion inesperado: $replay" >&2
  exit 1
}

docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 -U postgres -d "$base" \
  -c 'DROP FUNCTION public.probar_registro_contexto_actor_v3(text,timestamptz)' >/dev/null

# El LOGIN real heredando el rol nominal solo puede invocar la frontera exacta.
echo 'seguridad: LOGIN efectivo, capacidades negativas y downs con evidencia' >&2
decision_hex=$(psql_valor \
  "SELECT encode(decision_canonica,'hex') FROM vec_autorizacion.decision_concedida_contexto_actor_v3 WHERE decision_ref='decision:registro-v3:positiva'")
motivo_hex=$(psql_valor \
  "SELECT encode(motivo_canonico,'hex') FROM vec_autorizacion.decision_concedida_contexto_actor_v3 WHERE decision_ref='decision:registro-v3:positiva'")
replay_runtime=$(docker exec --interactive --env PGPASSWORD="$clave_registro" "$contenedor" \
  psql -XAt --set ON_ERROR_STOP=1 -h 127.0.0.1 \
    -U vec_autorizacion_registro_v3_prueba -d "$base" \
    --set decision_hex="$decision_hex" --set motivo_hex="$motivo_hex" <<'SQL'
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SELECT count(*) FROM vec_autorizacion.registrar_decision_contexto_actor_v3(
  decode(:'decision_hex','hex'),decode(:'motivo_hex','hex'),2,2
);
COMMIT;
SQL
)
[[ $replay_runtime == $'BEGIN\n1\nCOMMIT' ]] || {
  echo "replay del LOGIN efectivo inesperado: $replay_runtime" >&2
  exit 1
}
for consulta in \
  "SELECT * FROM vec_autorizacion.decision_concedida_contexto_actor_v3" \
  "SELECT vec_autorizacion.decision_contexto_actor_v3_canonica('{}'::jsonb)" \
  "SET ROLE vec_autorizacion_propietario"
do
  if docker exec --env PGPASSWORD="$clave_registro" "$contenedor" \
    psql -X --set ON_ERROR_STOP=1 -h 127.0.0.1 \
      -U vec_autorizacion_registro_v3_prueba -d "$base" \
      -c "$consulta" >/dev/null 2>&1; then
    echo "runtime V3 obtuvo capacidad prohibida: $consulta" >&2
    exit 1
  fi
done
membresia=$(psql_valor "SELECT count(*) FROM pg_auth_members WHERE member='vec_autorizacion_registro_v3_prueba'::regrole AND roleid='vec_autorizacion_registro'::regrole AND NOT admin_option AND inherit_option AND NOT set_option")
[[ $membresia == 1 ]] || {
  echo "membresia efectiva inesperada: $membresia" >&2
  exit 1
}

# La migración aditiva no conserva evidencia propia. Se retira antes de la
# frontera histórica y no deja función ni privilegio residual.
psql_archivo deploy/postgresql/autorizacion/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.down.sql
[[ $(psql_valor "SELECT (to_regprocedure('vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)') IS NULL)::text") == true ]]
[[ $(psql_valor "SELECT (to_regprocedure('vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_v3(bytea,bytea,numeric,numeric)') IS NULL)::text") == true ]]

# Con evidencia, ambos downs deben fallar sin borrar ni alterar filas.
filas_antes=$(docker exec "$contenedor" psql -XAt -U postgres -d "$base" -c \
  "SELECT (SELECT count(*) FROM vec_autorizacion.decision_concedida_contexto_actor_v3)+(SELECT count(*) FROM vec_autorizacion.decision_denegada_contexto_actor_v3)")
limite=$(psql_valor \
  "SELECT to_char(clock_timestamp()+interval '4 seconds','YYYY-MM-DD HH24:MI:SS.USOF')")
docker exec --interactive --env PGAPPNAME=vec-v3-holder-down "$contenedor" \
  psql -X --set ON_ERROR_STOP=1 -U postgres -d "$base" >/dev/null <<SQL &
BEGIN;
SELECT pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
  'vec_autorizacion:migracion:registro_contexto_actor_v3:000005',0
));
DO \$b\$ BEGIN
  WHILE clock_timestamp() < '$limite'::timestamptz LOOP PERFORM 1; END LOOP;
END \$b\$;
COMMIT;
SQL
holder=$!
esperar_actividad vec-v3-holder-down 1
docker exec --interactive --env PGAPPNAME=vec-v3-down-con-evidencia \
  "$contenedor" psql -X --set ON_ERROR_STOP=1 -U postgres -d "$base" \
  < "$raiz/deploy/postgresql/autorizacion/migraciones/000006_funcion_registro_decisiones_contexto_actor_v3.down.sql" \
  >/dev/null 2>&1 &
retirada=$!
esperar_actividad vec-v3-down-con-evidencia 1 espera
wait "$holder"
if wait "$retirada"; then
  echo '000006 down elimino frontera con evidencia V3' >&2
  exit 1
fi
if psql_archivo deploy/postgresql/autorizacion/migraciones/000005_registro_decisiones_contexto_actor_v3.down.sql >/dev/null 2>&1; then
  echo '000005 down elimino tablas con evidencia V3' >&2
  exit 1
fi
filas=$(docker exec "$contenedor" psql -XAt -U postgres -d "$base" -c \
  "SELECT (SELECT count(*) FROM vec_autorizacion.decision_concedida_contexto_actor_v3)+(SELECT count(*) FROM vec_autorizacion.decision_denegada_contexto_actor_v3)")
[[ $filas == "$filas_antes" ]] || {
  echo "down altero evidencia V3: antes=$filas_antes despues=$filas" >&2
  exit 1
}

if docker ps --format '{{.Names}}' | grep -Fx "$contenedor" >/dev/null; then
  : # El trap retira el unico contenedor creado por este runner al salir.
else
  echo 'el contenedor efimero desaparecio antes de completar la auditoria' >&2
  exit 1
fi

echo 'OK: registro ContextoActor/PDP V3 PostgreSQL 18'
