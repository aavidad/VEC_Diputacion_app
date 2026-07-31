#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-contexto-actor-v1-pg-${USER:-usuario}-$$"
base=vec_contexto_actor_v1_prueba
clave_admin=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
clave_runtime=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
clave_acreditador=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')

limpiar() { docker rm -f "$contenedor" >/dev/null 2>&1 || true; }
trap limpiar EXIT INT TERM

docker run --detach --rm --name "$contenedor" --publish 127.0.0.1::5432 \
  --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave_admin" \
  --env POSTGRES_INITDB_ARGS="${VEC_POSTGRES_INITDB_ARGS:-}" "$imagen" >/dev/null
for _ in $(seq 1 60); do
  docker exec "$contenedor" pg_isready --username postgres --dbname "$base" >/dev/null 2>&1 && break
  sleep 1
done
docker exec "$contenedor" pg_isready --username postgres --dbname "$base" >/dev/null
version_mayor=$(docker exec "$contenedor" psql -X --no-align --tuples-only \
  --set ON_ERROR_STOP=1 --username postgres --dbname "$base" --command \
  "SELECT current_setting('server_version_num')::integer / 10000")
[[ $version_mayor == 18 ]] || {
  echo "se requiere PostgreSQL 18; la imagen inicio PostgreSQL $version_mayor" >&2
  exit 1
}

psql_archivo() {
  docker exec --interactive "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/$1"
}
consulta_runtime() {
  docker exec --env PGPASSWORD="$clave_runtime" "$contenedor" psql -X --no-align --tuples-only \
    --set ON_ERROR_STOP=1 --host 127.0.0.1 --username vec_contexto_actor_runtime_prueba \
    --dbname "$base" --command "$1"
}
rechazar_runtime() {
  local consulta=$1 descripcion=$2 salida estado
  set +e
  salida=$(consulta_runtime "$consulta" 2>&1)
  estado=$?
  set -e
  if [[ $estado -eq 0 ]]; then
    echo "fallo cerrado no demostrado: $descripcion" >&2
    exit 1
  fi
}
resolver_sql() {
  local operacion=$1 recibo=$2 perfil=${3:-prf_sintetico_cccccccccccccccccccccccc}
  printf "BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE; SELECT count(*) FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2('%s','%s','cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa','%s','certificado','alto',clock_timestamp()); COMMIT" \
    "$operacion" "$recibo" "$perfil"
}
psql_admin() {
  docker exec --interactive "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base"
}
retirar_acreditacion_contexto_actor_v2() {
  docker exec --interactive \
    --env PGOPTIONS="-c vec.confirmar_retirada_acreditacion_contexto_actor_v2=RETIRAR_ACREDITACION_CONTEXTO_ACTOR_V2" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.down.sql"
}
rechazar_retirada_acreditacion() {
  local descripcion=$1
  if retirar_acreditacion_contexto_actor_v2 >/dev/null 2>&1; then
    echo "down 000002 aceptó $descripcion" >&2
    exit 1
  fi
}

# La base dedicada llega preendurecida. El bootstrap del modulo valida este
# estado, pero no muta ACL globales que el down no podria reconstruir.
psql_admin <<'SQL'
DO $base$
BEGIN
  EXECUTE format('REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',current_database());
END
$base$;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
psql_archivo deploy/postgresql/contexto_actor_v1/roles_up.sql
psql_archivo deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql
psql_archivo deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql
propietario_crea=$(docker exec "$contenedor" psql -X --no-align --tuples-only \
  --set ON_ERROR_STOP=1 --username postgres --dbname "$base" --command \
  "SELECT pg_catalog.has_database_privilege(
    'vec_contexto_actor_v1_propietario',current_database(),'CREATE')")
[[ $propietario_crea == f ]] || { echo 'el propietario conserva CREATE sobre la base' >&2; exit 1; }
psql_archivo deploy/postgresql/contexto_actor_v1/pruebas_sql/acl_y_contrato.sql
psql_archivo deploy/postgresql/contexto_actor_v1/pruebas_sql/fixtures_sinteticos.sql
psql_archivo deploy/postgresql/contexto_actor_v1/pruebas_sql/historias_append_only.sql

docker exec --interactive --env CLAVE_RUNTIME="$clave_runtime" "$contenedor" \
  psql -X --quiet --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
\getenv clave_runtime CLAVE_RUNTIME
CREATE ROLE vec_contexto_actor_runtime_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS PASSWORD :'clave_runtime';
GRANT vec_contexto_actor_v1_runtime TO vec_contexto_actor_runtime_prueba
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL

acreditada=$(consulta_runtime 'SELECT acreditada FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()')
[[ $acreditada == t ]] || { echo 'pool runtime no acreditado' >&2; exit 1; }

rechazar_runtime "BEGIN TRANSACTION ISOLATION LEVEL READ COMMITTED;
  SELECT * FROM vec_contexto_actor_v1.reconciliar_contexto_actor_v2(
    'oca_corta','rca_corta','cta_corta','prf_corto','inventado','inventada',clock_timestamp());
  COMMIT" 'reconciliacion con argumentos no canonicos'

set +e
salida=$(consulta_runtime 'SELECT count(*) FROM vec_contexto_actor_v1.registros_contexto' 2>&1)
estado=$?
set -e
if [[ $estado -eq 0 || $salida != *'permission denied'* ]]; then
  echo 'runtime accedio directamente a tablas' >&2
  exit 1
fi

# Los fixtures iniciales son deliberadamente no autoritativos. La funcion
# productiva debe denegarlos antes de insertar cualquier recibo.
rechazar_runtime "$(resolver_sql \
  oca_noautoritativa_000000000000000000000000 \
  rca_noautoritativa_000000000000000000000000)" \
  'fixture no_autoritativa por ruta productiva'
filas_noautoritativas=$(docker exec "$contenedor" psql -X --no-align --tuples-only \
  --set ON_ERROR_STOP=1 --username postgres --dbname "$base" --command \
  "SELECT count(*) FROM vec_contexto_actor_v1.registros_contexto
    WHERE operacion_ref='oca_noautoritativa_000000000000000000000000'")
[[ $filas_noautoritativas == 0 ]] || { echo 'la ruta productiva registro procedencia no autoritativa' >&2; exit 1; }

# Revisión maestra SINTETICA DE PRUEBA. Solo acredita el contrato del runner;
# no representa ni suplanta un feed corporativo real.
psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.procedencias VALUES
 ('prc_maestra_sintetica_pruebas_000001',1,repeat('a',64),'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',2,
  'prc_maestra_sintetica_pruebas_000001',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.proyeccion_cuenta_actual SET version=2
 WHERE cuenta_ref='cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa';
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',2,
  'prc_maestra_sintetica_pruebas_000001',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.persona_actual SET version=2
 WHERE persona_ref='per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb';
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('prf_sintetico_cccccccccccccccccccccccc',2,'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
  'prc_maestra_sintetica_pruebas_000001',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.perfil_actual SET version=2
 WHERE perfil_ref='prf_sintetico_cccccccccccccccccccccccc';
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_sintetico_dddddddddddddddddddddddd',2,
  'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa','prf_sintetico_cccccccccccccccccccccccc',
  'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
  'prc_maestra_sintetica_pruebas_000001',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.vinculo_contexto_actual SET version=2
 WHERE vinculo_ref='vca_sintetico_dddddddddddddddddddddddd';
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones VALUES
 ('vin_sintetico_eeeeeeeeeeeeeeeeeeeeeeee',2,
  'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb','candidato','can_sintetico_ffffffffffffffffffffffff',
  'prc_maestra_sintetica_pruebas_000001',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour'),
 ('vin_sintetico_gggggggggggggggggggggggg',2,
  'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb','empleado','emp_sintetico_hhhhhhhhhhhhhhhhhhhhhhhh',
  'prc_maestra_sintetica_pruebas_000001',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.vinculo_referencia_actual SET version=2
 WHERE vinculo_ref IN ('vin_sintetico_eeeeeeeeeeeeeeeeeeeeeeee','vin_sintetico_gggggggggggggggggggggggg');
COMMIT;
SQL

docker exec --interactive --env CLAVE_ACREDITADOR="$clave_acreditador" "$contenedor" \
  psql -X --quiet --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
\getenv clave_acreditador CLAVE_ACREDITADOR
CREATE ROLE vec_contexto_actor_acreditador_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS PASSWORD :'clave_acreditador';
GRANT CONNECT ON DATABASE vec_contexto_actor_v1_prueba
  TO vec_contexto_actor_acreditador_prueba;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1
  TO vec_contexto_actor_acreditador_prueba;
GRANT EXECUTE ON FUNCTION
  vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
    text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,
    text,text,timestamptz,timestamptz
  ) TO vec_contexto_actor_acreditador_prueba;
SQL

puerto=$(docker port "$contenedor" 5432/tcp | head -n1); puerto=${puerto##*:}
dsn="postgres://vec_contexto_actor_runtime_prueba:${clave_runtime}@127.0.0.1:${puerto}/${base}?sslmode=disable"
dsn_acreditador="postgres://vec_contexto_actor_acreditador_prueba:${clave_acreditador}@127.0.0.1:${puerto}/${base}?sslmode=disable"
dsn_migracion="postgres://postgres:${clave_admin}@127.0.0.1:${puerto}/${base}?sslmode=disable"
if [[ ${VEC_CONTEXTO_ACTOR_OMITIR_GO:-0} != 1 ]]; then
  go test ./internal/vec/application \
    -run '^TestServicioContextoActorProductivoRechazaAutoridadNoAutoritativa$' -count=1
  VEC_CONTEXTO_ACTOR_V2_POSTGRES_DSN="$dsn" \
    VEC_CONTEXTO_ACTOR_ACREDITADOR_V2_POSTGRES_DSN="$dsn_acreditador" \
    go test ./internal/vec/adapters/contextoactor/postgres \
    -run '^(TestIntegracionPostgreSQLContextoActorV2|TestReconciliacionPostgreSQLContextoActorV2EsperaFinalizacionConcurrente|TestResolutorContextoActorPostgreSQLRechazaManifiestoAdulteradoONoAutoritativo|TestResolutorContextoActorPostgreSQLNoHaceSegundoReintento|TestAcreditadorUsoRegistroContextoActorPostgreSQLV2IntegracionPostgreSQL18)$' \
    -count=1
fi
psql_admin <<'SQL'
REVOKE EXECUTE ON FUNCTION
  vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
    text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,
    text,text,timestamptz,timestamptz
  ) FROM vec_contexto_actor_acreditador_prueba;
REVOKE USAGE ON SCHEMA vec_contexto_actor_v1
  FROM vec_contexto_actor_acreditador_prueba;
REVOKE CONNECT ON DATABASE vec_contexto_actor_v1_prueba
  FROM vec_contexto_actor_acreditador_prueba;
DROP ROLE vec_contexto_actor_acreditador_prueba;
SQL

# Recibo nominal para probar la acreditacion cerrada sin depender de datos ni
# tipos del adaptador Go/PDP.
consulta_runtime "$(resolver_sql \
  oca_acredita_000000000000000000000000 \
  rca_acredita_000000000000000000000000)" >/dev/null
psql_archivo deploy/postgresql/contexto_actor_v1/pruebas_sql/acreditacion_uso_v2.sql

# La migracion no se adopta ni repara a si misma.
if psql_archivo \
  deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql \
  >/dev/null 2>&1; then
  echo '000002 de acreditacion permitio una segunda aplicacion' >&2
  exit 1
fi

acreditar_propietario() {
  local intervalo=$1
  docker exec "$contenedor" psql -X --quiet --no-align --tuples-only \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" --command "
      BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
      SET LOCAL ROLE vec_contexto_actor_v1_propietario;
      SELECT vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
        r.registro_contexto_ref,'vec.contexto-actor.vinculado.v2',
        r.huella_sha256,r.manifiesto_procedencia_huella_sha256,
        r.autoridad_efectiva,
        'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',2,
        'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',2,
        'prf_sintetico_cccccccccccccccccccccccc',2,
        'vca_sintetico_dddddddddddddddddddddddd',2,
        'certificado','alto',clock_timestamp(),
        clock_timestamp()+interval '$intervalo') IS NULL
      FROM vec_contexto_actor_v1.registros_contexto AS r
      WHERE r.registro_contexto_ref='rca_acredita_000000000000000000000000';
      COMMIT;"
}

esperar_guarda_global_exclusiva() {
  local bloqueada=false intento
  for _ in $(seq 1 100); do
    intento=$(docker exec "$contenedor" psql -X --quiet --no-align --tuples-only \
      --set ON_ERROR_STOP=1 --username postgres --dbname "$base" --command \
      "SELECT pg_catalog.pg_try_advisory_lock(pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0))")
    if [[ $intento == f ]]; then
      bloqueada=true
      break
    fi
    sleep 0.03
  done
  [[ $bloqueada == true ]]
}

# El reloj se toma despues de esperar la primera guarda advisory. Una decision
# que expira durante la espera termina en denegacion.
docker exec --interactive "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
  --username postgres --dbname "$base" <<'SQL' >/dev/null &
BEGIN;
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
SELECT pg_catalog.pg_sleep(2);
COMMIT;
SQL
retenedor_acreditacion=$!
esperar_guarda_global_exclusiva || { echo 'no se observo guarda de expiracion' >&2; exit 1; }
expirada=$(acreditar_propietario '1 second' | sed '/^$/d' | head -n1)
wait "$retenedor_acreditacion"
[[ $expirada == t ]] || { echo 'acreditacion no denego expiracion durante advisory lock' >&2; exit 1; }

# Fantasma real: el INSERT del segundo puntero ocurre despues de que la
# acreditacion haya fijado su snapshot y mantiene el advisory exclusivo hasta
# COMMIT. El puntero nuevo es invisible en ese snapshot; la fila de generacion
# actualizada por el AFTER STATEMENT debe forzar 40001 o denegacion, nunca un
# resultado acreditado sobre el conjunto antiguo.
docker exec --interactive "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
  --username postgres --dbname "$base" <<'SQL' >/dev/null &
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_fantasma_00000000000000000000000',1,
  'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa','prf_sintetico_cccccccccccccccccccccccc',
  'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
  'prc_maestra_sintetica_pruebas_000001',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo',clock_timestamp()-interval '1 minute',clock_timestamp()+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES
 ('vca_fantasma_00000000000000000000000',1);
SELECT pg_catalog.pg_sleep(1);
COMMIT;
SQL
insertador_fantasma=$!
esperar_guarda_global_exclusiva || { echo 'no se observo guarda del INSERT fantasma' >&2; exit 1; }
set +e
salida_fantasma=$(acreditar_propietario '10 minutes' 2>&1)
estado_fantasma=$?
set -e
wait "$insertador_fantasma"
if [[ $estado_fantasma -eq 0 ]]; then
  fantasma_denegado=$(sed '/^$/d' <<<"$salida_fantasma" | head -n1)
  [[ $fantasma_denegado == t ]] || { echo 'acreditacion acepto snapshot anterior al INSERT fantasma' >&2; exit 1; }
elif [[ $salida_fantasma != *'could not serialize access due to concurrent update'* ]]; then
  echo "acreditacion fallo de forma inesperada ante INSERT fantasma: $salida_fantasma" >&2
  exit 1
fi
psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DELETE FROM vec_contexto_actor_v1.vinculo_contexto_actual
 WHERE vinculo_ref='vca_fantasma_00000000000000000000000';
COMMIT;
SQL

# Una revocacion que posee primero la guarda exclusiva compromete y la
# acreditacion, al despertar, relee el puntero revocado.
docker exec --interactive "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
  --username postgres --dbname "$base" <<'SQL' >/dev/null &
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',3,
  'prc_maestra_sintetica_pruebas_000001',1,repeat('a',64),'autoridad_maestra_acreditada',
  'revocado',clock_timestamp()-interval '1 minute',clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.proyeccion_cuenta_actual SET version=3
 WHERE cuenta_ref='cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa';
SELECT pg_catalog.pg_sleep(1);
COMMIT;
SQL
revocador_acreditacion=$!
esperar_guarda_global_exclusiva || { echo 'no se observo guarda de revocacion' >&2; exit 1; }
set +e
salida_revocada=$(acreditar_propietario '10 minutes' 2>&1)
estado_revocada=$?
set -e
wait "$revocador_acreditacion"
revocada=$(sed '/^$/d' <<<"$salida_revocada" | head -n1)
if [[ $estado_revocada -eq 0 ]]; then
  [[ $revocada == t ]] || { echo 'acreditacion acepto avance revocado concurrente' >&2; exit 1; }
elif [[ $salida_revocada != *'could not serialize access due to concurrent update'* ]]; then
  echo "acreditacion fallo de forma inesperada ante revocacion: $salida_revocada" >&2
  exit 1
fi

psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',4,
  'prc_maestra_sintetica_pruebas_000001',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo',clock_timestamp()-interval '1 minute',clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.proyeccion_cuenta_actual SET version=4
 WHERE cuenta_ref='cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa';
COMMIT;
SQL
manifiestos_invalidos=$(docker exec "$contenedor" psql -X --no-align --tuples-only \
  --set ON_ERROR_STOP=1 --username postgres --dbname "$base" --command \
  "SELECT count(*) FROM vec_contexto_actor_v1.registros_contexto
    WHERE encode(pg_catalog.sha256(manifiesto_procedencia_canonico),'hex')<>manifiesto_procedencia_huella_sha256
       OR autoridad_efectiva<>'autoridad_maestra_acreditada'
       OR convert_from(representacion_canonica,'UTF8')::jsonb->>'esquema'
            <>'vec.contexto-actor.vinculado.v2'
       OR convert_from(representacion_canonica,'UTF8')::jsonb->>'cuenta_version'<>'2'")
[[ $manifiestos_invalidos == 0 ]] || { echo 'procedencia sintetica no quedo comprometida exactamente' >&2; exit 1; }

# Cero coincidencias y varias coincidencias se deniegan del mismo modo.
rechazar_runtime "$(resolver_sql \
  oca_cero_000000000000000000000000 \
  rca_cero_000000000000000000000000 \
  prf_inexistente_000000000000000000000000)" 'cero coincidencias'
psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_ambiguo_000000000000000000000000',1,
  'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa','prf_sintetico_cccccccccccccccccccccccc',
  'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
  'prc_maestra_sintetica_pruebas_000001',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES
 ('vca_ambiguo_000000000000000000000000',1);
COMMIT;
SQL
rechazar_runtime "$(resolver_sql \
  oca_ambiguo_000000000000000000000000 \
  rca_ambiguo_000000000000000000000000)" 'varias coincidencias'
psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DELETE FROM vec_contexto_actor_v1.vinculo_contexto_actual
 WHERE vinculo_ref='vca_ambiguo_000000000000000000000000';
COMMIT;
SQL

# Una revocacion versionada del enlace externo invalida el snapshot completo.
psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones VALUES
 ('vin_sintetico_eeeeeeeeeeeeeeeeeeeeeeee',3,
  'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb','candidato',
  'can_sintetico_ffffffffffffffffffffffff',
  'prc_maestra_sintetica_pruebas_000001',1,repeat('a',64),'autoridad_maestra_acreditada',
  'revocado',clock_timestamp()-interval '1 minute',clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.vinculo_referencia_actual SET version=3
 WHERE vinculo_ref='vin_sintetico_eeeeeeeeeeeeeeeeeeeeeeee';
COMMIT;
SQL
rechazar_runtime "$(resolver_sql \
  oca_revocado_000000000000000000000000 \
  rca_revocado_000000000000000000000000)" 'vinculo externo revocado'
psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones VALUES
 ('vin_sintetico_eeeeeeeeeeeeeeeeeeeeeeee',4,
  'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb','candidato',
  'can_sintetico_ffffffffffffffffffffffff',
  'prc_maestra_sintetica_pruebas_000001',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo',clock_timestamp()-interval '1 minute',clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.vinculo_referencia_actual SET version=4
 WHERE vinculo_ref='vin_sintetico_eeeeeeeeeeeeeeeeeeeeeeee';
COMMIT;
SQL

# El reloj autoritativo se toma al adquirir el ultimo lock: una ventana que
# caduca durante la espera no produce recibo.
psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_sintetico_dddddddddddddddddddddddd',3,
  'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa','prf_sintetico_cccccccccccccccccccccccc',
  'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
  'prc_maestra_sintetica_pruebas_000001',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo',clock_timestamp()-interval '1 minute',clock_timestamp()+interval '1.2 seconds');
UPDATE vec_contexto_actor_v1.vinculo_contexto_actual SET version=3
 WHERE vinculo_ref='vca_sintetico_dddddddddddddddddddddddd';
COMMIT;
SQL
docker exec --interactive "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
  --username postgres --dbname "$base" <<'SQL' >/dev/null &
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SELECT pg_catalog.pg_advisory_xact_lock(76110721);
SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual
 WHERE vinculo_ref='vca_sintetico_dddddddddddddddddddddddd' FOR UPDATE;
SELECT pg_catalog.pg_sleep(2.2);
COMMIT;
SQL
retenedor=$!
bloqueo=false
for _ in $(seq 1 100); do
  marca=$(docker exec "$contenedor" psql -X --no-align --tuples-only --username postgres --dbname "$base" \
    --command 'SELECT pg_catalog.pg_try_advisory_lock(76110721)')
  if [[ $marca == f ]]; then bloqueo=true; break; fi
  sleep 0.03
done
[[ $bloqueo == true ]] || { echo 'no se acredito el bloqueo de expiracion' >&2; exit 1; }
rechazar_runtime "$(resolver_sql \
  oca_expira_000000000000000000000000 \
  rca_expira_000000000000000000000000)" 'expiracion durante espera de locks'
wait "$retenedor"
psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_sintetico_dddddddddddddddddddddddd',4,
  'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa','prf_sintetico_cccccccccccccccccccccccc',
  'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
  'prc_maestra_sintetica_pruebas_000001',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo',clock_timestamp()-interval '1 minute',clock_timestamp()+interval '1 hour');
UPDATE vec_contexto_actor_v1.vinculo_contexto_actual SET version=4
 WHERE vinculo_ref='vca_sintetico_dddddddddddddddddddddddd';
COMMIT;
SQL

# La acreditacion cae si el LOGIN hereda cualquier grupo adicional.
psql_admin <<'SQL'
CREATE ROLE vec_contexto_actor_grupo_extra NOLOGIN;
GRANT vec_contexto_actor_grupo_extra TO vec_contexto_actor_runtime_prueba WITH SET FALSE;
SQL
rechazar_runtime 'SELECT * FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()' \
  'LOGIN runtime con grupo adicional'
psql_admin <<'SQL'
REVOKE vec_contexto_actor_grupo_extra FROM vec_contexto_actor_runtime_prueba;
DROP ROLE vec_contexto_actor_grupo_extra;
SQL

# Tampoco se acredita un LOGIN que haya recibido privilegios directos, aunque
# conserve la membresia nominal correcta.
psql_admin <<'SQL'
GRANT SELECT ON vec_contexto_actor_v1.proyeccion_cuenta_actual TO vec_contexto_actor_runtime_prueba;
SQL
rechazar_runtime 'SELECT * FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()' \
  'LOGIN runtime con acceso directo a tabla'
psql_admin <<'SQL'
REVOKE SELECT ON vec_contexto_actor_v1.proyeccion_cuenta_actual FROM vec_contexto_actor_runtime_prueba;
SQL

psql_admin <<'SQL'
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()
  TO vec_contexto_actor_runtime_prueba;
SQL
rechazar_runtime 'SELECT * FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()' \
  'LOGIN runtime con EXECUTE directo fuera del manifiesto'
psql_admin <<'SQL'
REVOKE EXECUTE ON FUNCTION vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()
  FROM vec_contexto_actor_runtime_prueba;
SQL

psql_admin <<'SQL'
ALTER ROLE vec_contexto_actor_v1_runtime SET application_name='configuracion_hostil';
SQL
rechazar_runtime 'SELECT * FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()' \
  'grupo runtime con rolconfig'
psql_admin <<'SQL'
ALTER ROLE vec_contexto_actor_v1_runtime RESET ALL;
CREATE ROLE vec_contexto_actor_padre_hostil NOLOGIN;
GRANT vec_contexto_actor_padre_hostil TO vec_contexto_actor_v1_runtime WITH SET FALSE;
SQL
rechazar_runtime 'SELECT * FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()' \
  'grupo runtime miembro de otro grupo'
psql_admin <<'SQL'
REVOKE vec_contexto_actor_padre_hostil FROM vec_contexto_actor_v1_runtime;
DROP ROLE vec_contexto_actor_padre_hostil;
CREATE SCHEMA vec_contexto_actor_objeto_hostil AUTHORIZATION vec_contexto_actor_v1_runtime;
SQL
rechazar_runtime 'SELECT * FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()' \
  'grupo runtime propietario fuera del manifiesto'
psql_admin <<'SQL'
DROP SCHEMA vec_contexto_actor_objeto_hostil;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres
  GRANT SELECT ON TABLES TO vec_contexto_actor_v1_runtime;
SQL
rechazar_runtime 'SELECT * FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()' \
  'grupo runtime presente en default ACL'
psql_admin <<'SQL'
ALTER DEFAULT PRIVILEGES FOR ROLE postgres
  REVOKE SELECT ON TABLES FROM vec_contexto_actor_v1_runtime;
SQL

# Los privilegios efectivos por PUBLIC fuera del esquema tambien invalidan el
# LOGIN aunque no creen una dependencia directa contra el rol runtime.
psql_admin <<'SQL'
CREATE SCHEMA vec_contexto_actor_superficie_hostil AUTHORIZATION postgres;
CREATE FUNCTION vec_contexto_actor_superficie_hostil.publica()
RETURNS integer LANGUAGE sql AS 'SELECT 1';
SQL
rechazar_runtime 'SELECT * FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()' \
  'EXECUTE efectivo por PUBLIC fuera de allowlist'
psql_admin <<'SQL'
DROP SCHEMA vec_contexto_actor_superficie_hostil CASCADE;
DO $base$
BEGIN
  EXECUTE format('GRANT TEMPORARY ON DATABASE %I TO PUBLIC',current_database());
END
$base$;
SQL
rechazar_runtime 'SELECT * FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()' \
  'TEMPORARY efectivo por PUBLIC en la base'
psql_admin <<'SQL'
DO $base$
BEGIN
  EXECUTE format('REVOKE TEMPORARY ON DATABASE %I FROM PUBLIC',current_database());
END
$base$;
SQL

# PostgreSQL 18 incorpora MAINTAIN al ACL de relaciones. Debe invalidar el
# LOGIN tanto si llega por PUBLIC como por su unica membresia permitida.
psql_admin <<'SQL'
GRANT MAINTAIN ON TABLE vec_contexto_actor_v1.proyeccion_cuenta_actual TO PUBLIC;
SQL
maintain_efectivo=$(consulta_runtime \
  "SELECT pg_catalog.has_table_privilege(current_user,
    'vec_contexto_actor_v1.proyeccion_cuenta_actual','MAINTAIN')")
[[ $maintain_efectivo == t ]] || { echo 'precondicion MAINTAIN por PUBLIC no demostrada' >&2; exit 1; }
rechazar_runtime 'SELECT * FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()' \
  'MAINTAIN efectivo por PUBLIC'
psql_admin <<'SQL'
REVOKE MAINTAIN ON TABLE vec_contexto_actor_v1.proyeccion_cuenta_actual FROM PUBLIC;
GRANT MAINTAIN ON TABLE vec_contexto_actor_v1.proyeccion_cuenta_actual
  TO vec_contexto_actor_v1_runtime;
SQL
maintain_membresia=$(consulta_runtime \
  "SELECT pg_catalog.has_table_privilege(current_user,
    'vec_contexto_actor_v1.proyeccion_cuenta_actual','MAINTAIN')")
[[ $maintain_membresia == t ]] || { echo 'precondicion MAINTAIN por membresia no demostrada' >&2; exit 1; }
rechazar_runtime 'SELECT * FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()' \
  'MAINTAIN efectivo por membresia'
psql_admin <<'SQL'
REVOKE MAINTAIN ON TABLE vec_contexto_actor_v1.proyeccion_cuenta_actual
  FROM vec_contexto_actor_v1_runtime;
SQL

# Las ACL de parametros son una clase de objeto global separada. Se recorren
# desde pg_parameter_acl y se comprueban con has_parameter_privilege para no
# omitir SET ni ALTER SYSTEM heredados de PUBLIC o del grupo runtime.
psql_admin <<'SQL'
GRANT SET ON PARAMETER application_name TO PUBLIC;
SQL
set_parametro_efectivo=$(consulta_runtime \
  "SELECT pg_catalog.has_parameter_privilege(current_user,'application_name','SET')")
[[ $set_parametro_efectivo == t ]] || { echo 'precondicion SET de parametro por PUBLIC no demostrada' >&2; exit 1; }
rechazar_runtime 'SELECT * FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()' \
  'SET de parametro efectivo por PUBLIC'
psql_admin <<'SQL'
REVOKE SET ON PARAMETER application_name FROM PUBLIC;
GRANT ALTER SYSTEM ON PARAMETER application_name TO vec_contexto_actor_v1_runtime;
SQL
alter_parametro_membresia=$(consulta_runtime \
  "SELECT pg_catalog.has_parameter_privilege(current_user,'application_name','ALTER SYSTEM')")
[[ $alter_parametro_membresia == t ]] || { echo 'precondicion ALTER SYSTEM por membresia no demostrada' >&2; exit 1; }
rechazar_runtime 'SELECT * FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()' \
  'ALTER SYSTEM de parametro efectivo por membresia'
psql_admin <<'SQL'
REVOKE ALTER SYSTEM ON PARAMETER application_name FROM vec_contexto_actor_v1_runtime;
SQL

# Tras limpiar las contaminaciones, el mismo LOGIN debe volver a acreditarse.
acreditada=$(consulta_runtime 'SELECT acreditada FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()')
[[ $acreditada == t ]] || { echo 'pool runtime no recupero la acreditacion tras limpiar ACL hostiles' >&2; exit 1; }

# La retirada aditiva exige opt-in y falla si una composicion futura conserva
# una concesion nominal sobre la funcion.
if [[ ${VEC_CONTEXTO_ACTOR_OMITIR_GO:-0} != 1 ]]; then
  VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN="$dsn_migracion" \
    go test ./internal/vec/adapters/contextoactor/postgres \
      -run '^TestRetiradaAcreditacionUsoV2(Cancela|Bloquea|Rechaza|Ignora|Destruye)' -count=1
  VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN="$dsn_migracion" \
    go test -race ./internal/vec/adapters/contextoactor/postgres \
      -run '^TestRetiradaAcreditacionUsoV2(Cancela|Bloquea|Rechaza|Ignora|Destruye)' -count=1
  VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN="$dsn_migracion" \
    go test ./internal/vec/adapters/contextoactor/postgres \
      -run '^TestRetiradaAcreditacionUsoV2Ejecuta' -count=1
  psql_archivo \
    deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql
fi
set +e
psql_archivo \
  deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.down.sql \
  >/dev/null 2>&1
set -e
funcion_acreditacion=$(docker exec "$contenedor" psql -X --no-align --tuples-only \
  --set ON_ERROR_STOP=1 --username postgres --dbname "$base" --command \
  "SELECT pg_catalog.to_regprocedure(
    'vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)') IS NOT NULL")
[[ $funcion_acreditacion == t ]] || { echo 'down 000002 sin opt-in retiro la funcion' >&2; exit 1; }

# PUBLIC tambien es una concesion externa (grantee=0), no una ausencia de rol.
psql_admin <<'SQL'
GRANT EXECUTE ON FUNCTION
  vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
    text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,
    text,text,timestamptz,timestamptz
  ) TO PUBLIC;
SQL
rechazar_retirada_acreditacion 'EXECUTE concedido a PUBLIC'
psql_admin <<'SQL'
REVOKE EXECUTE ON FUNCTION
  vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
    text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,
    text,text,timestamptz,timestamptz
  ) FROM PUBLIC;
SQL

psql_admin <<'SQL'
CREATE ROLE vec_consumidor_acreditacion_prueba NOLOGIN;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1
  TO vec_consumidor_acreditacion_prueba;
GRANT USAGE ON TYPE
  vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
  TO vec_consumidor_acreditacion_prueba;
SQL
rechazar_retirada_acreditacion 'tipo compuesto con concesión externa'
psql_admin <<'SQL'
REVOKE USAGE ON TYPE
  vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
  FROM vec_consumidor_acreditacion_prueba;
GRANT EXECUTE ON FUNCTION
  vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
    text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,
    text,text,timestamptz,timestamptz
  ) TO vec_consumidor_acreditacion_prueba;
SQL
rechazar_retirada_acreditacion 'función con concesión externa'
psql_admin <<'SQL'
REVOKE EXECUTE ON FUNCTION
  vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
    text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,
    text,text,timestamptz,timestamptz
  ) FROM vec_consumidor_acreditacion_prueba;
REVOKE USAGE ON SCHEMA vec_contexto_actor_v1
  FROM vec_consumidor_acreditacion_prueba;
DROP ROLE vec_consumidor_acreditacion_prueba;
SQL

# Propiedades de columna y comentarios de trigger forman parte del canon.
psql_admin <<'SQL'
ALTER TABLE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
  ALTER COLUMN generacion SET STATISTICS 777;
SQL
rechazar_retirada_acreditacion 'estadística de columna derivada'
[[ $(docker exec "$contenedor" psql -XAt --username postgres --dbname "$base" \
  --command "SELECT attstattarget FROM pg_catalog.pg_attribute
    WHERE attrelid='vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass
      AND attname='generacion'") == 777 ]]
psql_admin <<'SQL'
ALTER TABLE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
  ALTER COLUMN generacion SET STATISTICS -1;
COMMENT ON TRIGGER avanzar_generacion_punteros_actuales_v2
  ON vec_contexto_actor_v1.proyeccion_cuenta_actual
  IS 'ct118 comentario hostil sintético';
SQL
rechazar_retirada_acreditacion 'comentario hostil de trigger'
psql_admin <<'SQL'
COMMENT ON TRIGGER avanzar_generacion_punteros_actuales_v2
  ON vec_contexto_actor_v1.proyeccion_cuenta_actual IS NULL;
CREATE SCHEMA ct118_homonimo AUTHORIZATION postgres;
CREATE FUNCTION ct118_homonimo.rechazar() RETURNS trigger
  LANGUAGE plpgsql AS 'BEGIN RETURN NULL; END';
CREATE TABLE ct118_homonimo.puntero (id integer);
CREATE TRIGGER puntero_actual_no_truncable_v2 BEFORE TRUNCATE
  ON ct118_homonimo.puntero FOR EACH STATEMENT
  EXECUTE FUNCTION ct118_homonimo.rechazar();
SQL
retirar_acreditacion_contexto_actor_v2
[[ $(docker exec "$contenedor" psql -XAt --username postgres --dbname "$base" \
  --command "SELECT count(*) FROM pg_catalog.pg_trigger
    WHERE tgrelid='ct118_homonimo.puntero'::regclass
      AND tgname='puntero_actual_no_truncable_v2'") == 1 ]]
psql_admin <<'SQL'
DROP SCHEMA ct118_homonimo CASCADE;
SQL

set +e
psql_archivo deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.down.sql \
  >/dev/null 2>&1
set -e
esquema=$(docker exec "$contenedor" psql -X --no-align --tuples-only --set ON_ERROR_STOP=1 \
  --username postgres --dbname "$base" --command \
  "SELECT pg_catalog.to_regnamespace('vec_contexto_actor_v1') IS NOT NULL")
[[ $esquema == t ]] || { echo 'down sin opt-in destruyo el esquema' >&2; exit 1; }
if docker exec --interactive "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
  --set confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1 \
  --username postgres --dbname "$base" \
  < "$raiz/deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.down.sql" \
  >/dev/null 2>&1; then
  echo 'down 000001 elimino historia durable con confirmacion' >&2
  exit 1
fi
filas_conservadas=$(docker exec "$contenedor" psql -X --no-align --tuples-only \
  --set ON_ERROR_STOP=1 --username postgres --dbname "$base" --command \
  "SELECT count(*) FROM vec_contexto_actor_v1.registros_contexto")
[[ $filas_conservadas -gt 0 ]] || { echo 'down 000001 altero evidencia tras rechazo' >&2; exit 1; }

echo 'contexto actor durable V2: integracion PostgreSQL 18 superada'
