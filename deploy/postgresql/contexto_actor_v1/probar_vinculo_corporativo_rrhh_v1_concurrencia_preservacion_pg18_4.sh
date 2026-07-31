#!/usr/bin/env bash
set -euo pipefail
export LC_ALL=C TZ=UTC

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-vinculo-concurrencia-${USER:-usuario}-$$"
base=c22b_vinculo_concurrencia
clave=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
directorio=deploy/postgresql/contexto_actor_v1
roles="$directorio/roles_up.sql"
roles_selector="$directorio/roles_contexto_corporativo_rrhh_selector_v1_up.sql"
up_1="$directorio/migraciones/000001_contexto_actor_v1.up.sql"
up_2="$directorio/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql"
down_2="$directorio/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.down.sql"
up_3="$directorio/migraciones/000003_organizacion_corporativa_v1.up.sql"
down_3="$directorio/migraciones/000003_organizacion_corporativa_v1.down.sql"
up="$directorio/migraciones/000004_vinculo_corporativo_rrhh_v1.up.sql"
down="$directorio/migraciones/000004_vinculo_corporativo_rrhh_v1.down.sql"
temporal=$(mktemp -d)
procesos=()
sesion_fd=
sesion_pid=
sesion_fifo=
sesion_log=

limpiar() {
  local pid
  set +e
  if [[ -n ${sesion_fd:-} ]]; then
    exec {sesion_fd}>&-
  fi
  for pid in "${procesos[@]:-}"; do
    kill "$pid" >/dev/null 2>&1 || true
    wait "$pid" >/dev/null 2>&1 || true
  done
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  rm -rf -- "$temporal"
}
trap limpiar EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

fallar() {
  local archivo
  echo "$1" >&2
  for archivo in "$temporal"/*.log; do
    [[ ! -s $archivo ]] || {
      echo "diagnostico ${archivo##*/}:" >&2
      tail -n 40 "$archivo" >&2
    }
  done
  docker logs --tail 160 "$contenedor" >&2 2>/dev/null || true
  exit 1
}

esperar_postgres() {
  local consecutivas=0 respuesta
  for _ in $(seq 1 240); do
    if respuesta=$(docker exec --env PGPASSWORD="$clave" "$contenedor" \
      psql -XAt -h 127.0.0.1 -U postgres -d "$base" \
      -c "SELECT current_setting('server_version_num')||'|'||pg_is_in_recovery()" \
      2>/dev/null) && [[ $respuesta == '180004|false' ]]; then
      consecutivas=$((consecutivas + 1))
      [[ $consecutivas -eq 3 ]] && return 0
    else
      consecutivas=0
    fi
    sleep 0.25
  done
  fallar 'PostgreSQL 18.4 primario no quedo disponible tres veces consecutivas'
}

psql_archivo() {
  docker exec --interactive --env PGPASSWORD="$clave" "$contenedor" \
    psql -Xq -h 127.0.0.1 -U postgres -d "$base" -v ON_ERROR_STOP=1 \
    < "$raiz/$1"
}

psql_sql() {
  docker exec --interactive --env PGPASSWORD="$clave" "$contenedor" \
    psql -Xq -h 127.0.0.1 -U postgres -d "$base" -v ON_ERROR_STOP=1
}

consulta() {
  docker exec --env PGPASSWORD="$clave" "$contenedor" \
    psql -XAt -h 127.0.0.1 -U postgres -d "$base" -v ON_ERROR_STOP=1 -c "$1"
}

esperar_valor() {
  local descripcion=$1 esperado=$2 sql=$3 observado
  for _ in $(seq 1 500); do
    observado=$(consulta "$sql" 2>/dev/null || true)
    [[ $observado == "$esperado" ]] && return 0
    sleep 0.02
  done
  fallar "espera agotada: $descripcion; observado=${observado:-<ausente>}"
}

huella_total() {
  docker exec --env PGPASSWORD="$clave" "$contenedor" pg_dump \
    -h 127.0.0.1 -U postgres -d "$base" --format=plain |
    sed -E '/^\\(un)?restrict /d' | sha256sum | cut -d' ' -f1
}

huella_esquema() {
  docker exec --env PGPASSWORD="$clave" "$contenedor" pg_dump \
    -h 127.0.0.1 -U postgres -d "$base" --schema-only \
    --schema=vec_contexto_actor_v1 |
    sed -E '/^\\(un)?restrict /d' | sha256sum | cut -d' ' -f1
}

huella_filas_base() {
  docker exec --env PGPASSWORD="$clave" "$contenedor" pg_dump \
    -h 127.0.0.1 -U postgres -d "$base" --data-only \
    --schema=vec_contexto_actor_v1 \
    --exclude-table=vec_contexto_actor_v1.vinculo_corporativo_versiones \
    --exclude-table=vec_contexto_actor_v1.vinculo_corporativo_actual |
    sed -E '/^\\(un)?restrict /d' | sha256sum | cut -d' ' -f1
}

iniciar_sesion() {
  local aplicacion=$1
  sesion_fifo="$temporal/sesion.fifo"
  sesion_log="$temporal/${aplicacion}.log"
  rm -f -- "$sesion_fifo"
  mkfifo "$sesion_fifo"
  exec {sesion_fd}<>"$sesion_fifo"
  docker exec --interactive --env PGPASSWORD="$clave" \
    --env PGAPPNAME="$aplicacion" "$contenedor" \
    psql -Xq -h 127.0.0.1 -U postgres -d "$base" -v ON_ERROR_STOP=1 \
    < "$sesion_fifo" > "$sesion_log" 2>&1 &
  sesion_pid=$!
  procesos+=("$sesion_pid")
}

enviar_sesion() {
  local linea
  while IFS= read -r linea; do
    printf '%s\n' "$linea" >&"$sesion_fd"
  done
}

confirmar_sesion() {
  local estado
  printf 'COMMIT;\n\\q\n' >&"$sesion_fd"
  exec {sesion_fd}>&-
  sesion_fd=
  set +e
  wait "$sesion_pid"
  estado=$?
  set -e
  [[ $estado -eq 0 ]] || fallar 'la sesion coordinada no pudo confirmar'
  sesion_pid=
  rm -f -- "$sesion_fifo"
  sesion_fifo=
}

comprobar_un_exito() {
  local descripcion=$1 pid_1=$2 pid_2=$3 estado_1 estado_2
  set +e
  wait "$pid_1"; estado_1=$?
  wait "$pid_2"; estado_2=$?
  set -e
  if ! { [[ $estado_1 -eq 0 && $estado_2 -ne 0 ]] ||
         [[ $estado_1 -ne 0 && $estado_2 -eq 0 ]]; }; then
    fallar "$descripcion no produjo exactamente un exito: $estado_1/$estado_2"
  fi
}

esperar_fallo() {
  local descripcion=$1 pid=$2 estado
  set +e
  wait "$pid"; estado=$?
  set -e
  [[ $estado -ne 0 ]] || fallar "se acepto: $descripcion"
}

docker run -d --rm --name "$contenedor" --publish 127.0.0.1::5432 \
  -e POSTGRES_DB="$base" -e POSTGRES_PASSWORD="$clave" "$imagen" >/dev/null
esperar_postgres
psql_sql <<'SQL'
DO $base$
BEGIN
  CREATE ROLE c22b_dueno_base NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOINHERIT NOREPLICATION NOBYPASSRLS;
  EXECUTE format('ALTER DATABASE %I OWNER TO c22b_dueno_base',current_database());
  EXECUTE format('REVOKE ALL ON DATABASE %I FROM PUBLIC',current_database());
END
$base$;
REVOKE ALL ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
psql_archivo "$roles"
psql_archivo "$up_1"
psql_archivo "$up_2"
psql_archivo "$roles_selector"
psql_archivo "$up_3"
huella_a=$(huella_total)

# Caso 11.1: dos altas compiten por la barrera B; solo una puede confirmar.
iniciar_sesion c22b_barrera_up
enviar_sesion <<'SQL'
BEGIN;
SELECT pg_advisory_xact_lock(hashtextextended(
  'vec_contexto_actor_v1:vinculo-corporativo-rrhh:v1',0));
SQL
esperar_valor 'barrera exclusiva de alta' 1 "SELECT count(*) FROM pg_locks l
 JOIN pg_stat_activity a ON a.pid=l.pid WHERE a.application_name='c22b_barrera_up'
 AND l.locktype='advisory' AND l.granted"
docker exec --interactive --env PGPASSWORD="$clave" \
  --env PGAPPNAME=c22b_up_1 "$contenedor" \
  psql -Xq -h 127.0.0.1 -U postgres -d "$base" -v ON_ERROR_STOP=1 \
  < "$raiz/$up" > "$temporal/up_1.log" 2>&1 & pid_up_1=$!
procesos+=("$pid_up_1")
docker exec --interactive --env PGPASSWORD="$clave" \
  --env PGAPPNAME=c22b_up_2 "$contenedor" \
  psql -Xq -h 127.0.0.1 -U postgres -d "$base" -v ON_ERROR_STOP=1 \
  < "$raiz/$up" > "$temporal/up_2.log" 2>&1 & pid_up_2=$!
procesos+=("$pid_up_2")
esperar_valor 'dos altas bloqueadas' 2 "SELECT count(*) FROM pg_stat_activity
 WHERE application_name IN ('c22b_up_1','c22b_up_2')
 AND cardinality(pg_blocking_pids(pid))>0"
confirmar_sesion
comprobar_un_exito 'carrera up/up' "$pid_up_1" "$pid_up_2"
[[ $(consulta "SELECT count(*) FROM pg_class WHERE oid IN (
 'vec_contexto_actor_v1.vinculo_corporativo_versiones'::regclass,
 'vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass)") == 2 ]] ||
  fallar 'la carrera up/up no dejo B instalado exactamente una vez'

# Caso 11.2: dos retiradas compiten; una retira y la segunda falla cerrada.
iniciar_sesion c22b_barrera_down
enviar_sesion <<'SQL'
BEGIN;
SELECT pg_advisory_xact_lock(hashtextextended(
  'vec_contexto_actor_v1:vinculo-corporativo-rrhh:v1',0));
SQL
esperar_valor 'barrera exclusiva de retirada' 1 "SELECT count(*) FROM pg_locks l
 JOIN pg_stat_activity a ON a.pid=l.pid WHERE a.application_name='c22b_barrera_down'
 AND l.locktype='advisory' AND l.granted"
for numero in 1 2; do
  docker exec --interactive --env PGPASSWORD="$clave" \
    --env PGAPPNAME="c22b_down_$numero" \
    --env PGOPTIONS='-c vec.confirmar_retirada_vinculo_corporativo_rrhh_v1=RETIRAR_VINCULO_CORPORATIVO_RRHH_V1' \
    "$contenedor" psql -Xq -h 127.0.0.1 -U postgres -d "$base" \
    -v ON_ERROR_STOP=1 < "$raiz/$down" > "$temporal/down_$numero.log" 2>&1 &
  if [[ $numero -eq 1 ]]; then pid_down_1=$!; else pid_down_2=$!; fi
  procesos+=("$!")
done
esperar_valor 'dos retiradas bloqueadas' 2 "SELECT count(*) FROM pg_stat_activity
 WHERE application_name IN ('c22b_down_1','c22b_down_2')
 AND cardinality(pg_blocking_pids(pid))>0"
confirmar_sesion
comprobar_un_exito 'carrera down/down' "$pid_down_1" "$pid_down_2"
[[ $(consulta "SELECT to_regclass('vec_contexto_actor_v1.vinculo_corporativo_versiones') IS NULL") == t ]] ||
  fallar 'la carrera down/down dejo B parcialmente instalado'
[[ $(huella_total) == "$huella_a" ]] ||
  fallar 'la carrera down/down no preservo exactamente A'

psql_archivo "$up"
huella_b=$(huella_total)
huella_esquema_b=$(huella_esquema)

# Caso 11.3: DDL hostil confirma primero; down observa la deriva y no modifica.
iniciar_sesion c22b_ddl_hostil
enviar_sesion <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
  REPLICA IDENTITY FULL;
SQL
esperar_valor 'DDL hostil inmoviliza historia' 1 "SELECT count(*) FROM pg_locks l
 JOIN pg_stat_activity a ON a.pid=l.pid
 WHERE a.application_name='c22b_ddl_hostil' AND l.locktype='relation'
 AND l.relation='vec_contexto_actor_v1.vinculo_corporativo_versiones'::regclass
 AND l.granted AND l.mode IN ('ShareRowExclusiveLock','AccessExclusiveLock')"
docker exec --interactive --env PGPASSWORD="$clave" \
  --env PGAPPNAME=c22b_down_ddl \
  --env PGOPTIONS='-c vec.confirmar_retirada_vinculo_corporativo_rrhh_v1=RETIRAR_VINCULO_CORPORATIVO_RRHH_V1' \
  "$contenedor" psql -Xq -h 127.0.0.1 -U postgres -d "$base" \
  -v ON_ERROR_STOP=1 < "$raiz/$down" > "$temporal/down_ddl.log" 2>&1 &
pid_down_ddl=$!; procesos+=("$pid_down_ddl")
esperar_valor 'down bloqueada por DDL' 1 "SELECT count(*) FROM pg_stat_activity
 WHERE application_name='c22b_down_ddl' AND cardinality(pg_blocking_pids(pid))>0"
confirmar_sesion
esperar_fallo 'retirada concurrente con DDL hostil' "$pid_down_ddl"
[[ $(consulta "SELECT relreplident FROM pg_class WHERE oid=
 'vec_contexto_actor_v1.vinculo_corporativo_versiones'::regclass") == f ]] ||
  fallar 'el rechazo de down no preservo el DDL hostil confirmado'
[[ $(huella_esquema) != "$huella_esquema_b" ]] ||
  fallar 'la sonda DDL no altero realmente el catalogo'
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
  REPLICA IDENTITY DEFAULT;
RESET ROLE;
SQL
[[ $(huella_total) == "$huella_b" ]] ||
  fallar 'la restauracion tras DDL no recupero B exacto'

# Caso 11.4: la cadena 000002/000003 no puede atravesar B.
iniciar_sesion c22b_barrera_predecesores
enviar_sesion <<'SQL'
BEGIN;
SELECT pg_advisory_xact_lock(hashtextextended(
  'vec_contexto_actor_v1:migracion:acreditacion_uso:v2',0));
SQL
esperar_valor 'barrera de predecesores' 1 "SELECT count(*) FROM pg_locks l
 JOIN pg_stat_activity a ON a.pid=l.pid
 WHERE a.application_name='c22b_barrera_predecesores'
 AND l.locktype='advisory' AND l.granted"
docker exec --interactive --env PGPASSWORD="$clave" \
  --env PGAPPNAME=c22b_down_2_pre \
  --env PGOPTIONS='-c vec.confirmar_retirada_acreditacion_contexto_actor_v2=RETIRAR_ACREDITACION_CONTEXTO_ACTOR_V2' \
  "$contenedor" psql -Xq -h 127.0.0.1 -U postgres -d "$base" \
  -v ON_ERROR_STOP=1 < "$raiz/$down_2" > "$temporal/down_2_pre.log" 2>&1 &
pid_pre_2=$!; procesos+=("$pid_pre_2")
docker exec --interactive --env PGPASSWORD="$clave" \
  --env PGAPPNAME=c22b_down_3_pre \
  --env PGOPTIONS='-c vec.confirmar_retirada_organizacion_corporativa_v1=RETIRAR_ORGANIZACION_CORPORATIVA_V1' \
  "$contenedor" psql -Xq -h 127.0.0.1 -U postgres -d "$base" \
  -v ON_ERROR_STOP=1 < "$raiz/$down_3" > "$temporal/down_3_pre.log" 2>&1 &
pid_pre_3=$!; procesos+=("$pid_pre_3")
esperar_valor 'dos predecesores bloqueados' 2 "SELECT count(*) FROM pg_stat_activity
 WHERE application_name IN ('c22b_down_2_pre','c22b_down_3_pre')
 AND cardinality(pg_blocking_pids(pid))>0"
confirmar_sesion
esperar_fallo 'retirada prematura 000002' "$pid_pre_2"
esperar_fallo 'retirada prematura 000003' "$pid_pre_3"
[[ $(huella_total) == "$huella_b" ]] ||
  fallar 'la carrera de predecesores altero B o A'

# Fixtures sintéticos mínimos para las carreras con evidencia durable.
psql_sql <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.procedencias VALUES
 ('prc_autoridad_corporativa_rrhh_0001',1,repeat('a',64),'autoridad_maestra_acreditada'),
 ('prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_corporativa_rrhh_000000000001',1,'prc_vinculo_corporativo_rrhh_000001',1,
  repeat('b',64),'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('per_corporativa_rrhh_000000000001',1,'prc_vinculo_corporativo_rrhh_000001',1,
  repeat('b',64),'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('prf_corporativo_rrhh_000000000001',1,'per_corporativa_rrhh_000000000001',
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_corporativo_rrhh_000000000001',1,'cta_corporativa_rrhh_000000000001',
  'prf_corporativo_rrhh_000000000001','per_corporativa_rrhh_000000000001',
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES
 ('org_diputaciondemo0001',1,'prc_autoridad_corporativa_rrhh_0001',1,
  repeat('a',64),'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01');
COMMIT;
SQL
generacion_inicial=$(consulta 'SELECT generacion FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2')

# Caso 11.5: una inserción confirmada gana a down y permanece íntegra.
iniciar_sesion c22b_insercion
enviar_sesion <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones VALUES
 ('vcr_corporativo_rrhh_000000000001',1,
  'cta_corporativa_rrhh_000000000001',1,'per_corporativa_rrhh_000000000001',1,
  'prf_corporativo_rrhh_000000000001',1,'vca_corporativo_rrhh_000000000001',1,
  'org_diputaciondemo0001',1,'prc_autoridad_corporativa_rrhh_0001',1,
  repeat('a',64),'autoridad_maestra_acreditada','interna_corporativa','consulta_rrhh',
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01');
SQL
esperar_valor 'insercion mantiene lock de historia' 1 "SELECT count(*) FROM pg_locks l
 JOIN pg_stat_activity a ON a.pid=l.pid WHERE a.application_name='c22b_insercion'
 AND l.locktype='relation' AND l.relation=
 'vec_contexto_actor_v1.vinculo_corporativo_versiones'::regclass
 AND l.mode='RowExclusiveLock' AND l.granted"
docker exec --interactive --env PGPASSWORD="$clave" \
  --env PGAPPNAME=c22b_down_insercion \
  --env PGOPTIONS='-c vec.confirmar_retirada_vinculo_corporativo_rrhh_v1=RETIRAR_VINCULO_CORPORATIVO_RRHH_V1' \
  "$contenedor" psql -Xq -h 127.0.0.1 -U postgres -d "$base" \
  -v ON_ERROR_STOP=1 < "$raiz/$down" > "$temporal/down_insercion.log" 2>&1 &
pid_down_insercion=$!; procesos+=("$pid_down_insercion")
esperar_valor 'down bloqueada por insercion' 1 "SELECT count(*) FROM pg_stat_activity
 WHERE application_name='c22b_down_insercion'
 AND cardinality(pg_blocking_pids(pid))>0"
confirmar_sesion
esperar_fallo 'retirada concurrente con insercion' "$pid_down_insercion"
[[ $(consulta 'SELECT count(*) FROM vec_contexto_actor_v1.vinculo_corporativo_versiones') == 1 ]] ||
  fallar 'el rechazo de down perdio la historia confirmada'
[[ $(consulta 'SELECT generacion FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2') == "$generacion_inicial" ]] ||
  fallar 'la insercion de historia altero la generacion'
[[ $(huella_esquema) == "$huella_esquema_b" ]] ||
  fallar 'la carrera de insercion altero el catalogo'

# Caso 11.6: el avance de puntero confirma primero y down conserva su generación.
iniciar_sesion c22b_puntero
enviar_sesion <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual VALUES
 ('cta_corporativa_rrhh_000000000001','interna_corporativa','consulta_rrhh',
  'vcr_corporativo_rrhh_000000000001',1);
SQL
esperar_valor 'puntero mantiene lock de escritura' 1 "SELECT count(*) FROM pg_locks l
 JOIN pg_stat_activity a ON a.pid=l.pid WHERE a.application_name='c22b_puntero'
 AND l.locktype='relation' AND l.relation=
 'vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass
 AND l.mode='RowExclusiveLock' AND l.granted"
docker exec --interactive --env PGPASSWORD="$clave" \
  --env PGAPPNAME=c22b_down_puntero \
  --env PGOPTIONS='-c vec.confirmar_retirada_vinculo_corporativo_rrhh_v1=RETIRAR_VINCULO_CORPORATIVO_RRHH_V1' \
  "$contenedor" psql -Xq -h 127.0.0.1 -U postgres -d "$base" \
  -v ON_ERROR_STOP=1 < "$raiz/$down" > "$temporal/down_puntero.log" 2>&1 &
pid_down_puntero=$!; procesos+=("$pid_down_puntero")
esperar_valor 'down bloqueada por avance de puntero' 1 "SELECT count(*) FROM pg_stat_activity
 WHERE application_name='c22b_down_puntero'
 AND cardinality(pg_blocking_pids(pid))>0"
confirmar_sesion
esperar_fallo 'retirada concurrente con avance de puntero' "$pid_down_puntero"
[[ $(consulta 'SELECT count(*) FROM vec_contexto_actor_v1.vinculo_corporativo_actual') == 1 ]] ||
  fallar 'el rechazo de down perdio el puntero confirmado'
[[ $(consulta 'SELECT generacion FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2') == "$((generacion_inicial + 1))" ]] ||
  fallar 'el rechazo de down perdio el avance de generacion'
[[ $(huella_esquema) == "$huella_esquema_b" ]] ||
  fallar 'la carrera de puntero altero el catalogo'

# B queda vacío; su retirada debe preservar filas y generación base pobladas.
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
DELETE FROM vec_contexto_actor_v1.vinculo_corporativo_actual;
RESET ROLE;
SET session_replication_role=replica;
DELETE FROM vec_contexto_actor_v1.vinculo_corporativo_versiones;
SET session_replication_role=origin;
SQL
huella_base_poblada=$(huella_filas_base)
generacion_base_poblada=$(consulta 'SELECT generacion FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2')
docker exec --interactive --env PGPASSWORD="$clave" \
  --env PGOPTIONS='-c vec.confirmar_retirada_vinculo_corporativo_rrhh_v1=RETIRAR_VINCULO_CORPORATIVO_RRHH_V1' \
  "$contenedor" psql -Xq -h 127.0.0.1 -U postgres -d "$base" \
  -v ON_ERROR_STOP=1 < "$raiz/$down"
[[ $(huella_filas_base) == "$huella_base_poblada" ]] ||
  fallar 'la retirada final de B altero filas de A'
[[ $(consulta 'SELECT generacion FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2') == "$generacion_base_poblada" ]] ||
  fallar 'la retirada final de B altero la generacion'

# La cadena ordenada queda utilizable después de retirar B y sus fixtures.
psql_sql <<'SQL'
SET session_replication_role=replica;
DELETE FROM vec_contexto_actor_v1.organizacion_versiones
 WHERE organizacion_ref='org_diputaciondemo0001';
DELETE FROM vec_contexto_actor_v1.vinculo_contexto_versiones
 WHERE vinculo_ref='vca_corporativo_rrhh_000000000001';
DELETE FROM vec_contexto_actor_v1.perfil_versiones
 WHERE perfil_ref='prf_corporativo_rrhh_000000000001';
DELETE FROM vec_contexto_actor_v1.persona_versiones
 WHERE persona_ref='per_corporativa_rrhh_000000000001';
DELETE FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones
 WHERE cuenta_ref='cta_corporativa_rrhh_000000000001';
DELETE FROM vec_contexto_actor_v1.procedencias
 WHERE procedencia_ref IN ('prc_autoridad_corporativa_rrhh_0001',
                           'prc_vinculo_corporativo_rrhh_000001');
SET session_replication_role=origin;
SQL
docker exec --interactive --env PGPASSWORD="$clave" \
  --env PGOPTIONS='-c vec.confirmar_retirada_organizacion_corporativa_v1=RETIRAR_ORGANIZACION_CORPORATIVA_V1' \
  "$contenedor" psql -Xq -h 127.0.0.1 -U postgres -d "$base" \
  -v ON_ERROR_STOP=1 < "$raiz/$down_3"
docker exec --interactive --env PGPASSWORD="$clave" \
  --env PGOPTIONS='-c vec.confirmar_retirada_acreditacion_contexto_actor_v2=RETIRAR_ACREDITACION_CONTEXTO_ACTOR_V2' \
  "$contenedor" psql -Xq -h 127.0.0.1 -U postgres -d "$base" \
  -v ON_ERROR_STOP=1 < "$raiz/$down_2"
[[ $(consulta "SELECT to_regclass('vec_contexto_actor_v1.organizacion_versiones') IS NULL
 AND to_regclass('vec_contexto_actor_v1.control_generacion_punteros_actuales_v2') IS NULL") == t ]] ||
  fallar 'la cadena ordenada de retirada no finalizo'

echo 'OK: vinculo corporativo RRHH V1 supera concurrencia y preservacion PG 18.4'
