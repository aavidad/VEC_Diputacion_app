#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"

imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-contexto-actor-s02c-pg-${USER:-usuario}-$$"
base_control=ct133_control
base=ct133_s02c
clave_admin=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
roles=deploy/postgresql/contexto_actor_v1/roles_up.sql
up_base=deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql
up=deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql
down=deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.down.sql
temporales=()

limpiar() {
  local proceso archivo
  while IFS= read -r proceso; do
    kill "$proceso" >/dev/null 2>&1 || true
    wait "$proceso" >/dev/null 2>&1 || true
  done < <(jobs -pr)
  for archivo in "${temporales[@]:-}"; do
    rm -f "$archivo"
  done
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

fallar() {
  echo "$1" >&2
  exit 1
}

esperar_postgres() {
  local consecutivas=0 respuesta
  for _ in $(seq 1 240); do
    if respuesta=$(docker exec "$contenedor" psql -XAt \
      --set ON_ERROR_STOP=1 --username postgres --dbname "$base_control" \
      --command \
      "SELECT current_setting('server_version_num') || '|' ||
              pg_catalog.pg_is_in_recovery()" 2>/dev/null) &&
      [[ $respuesta == '180004|false' ]]; then
      consecutivas=$((consecutivas + 1))
      [[ $consecutivas -eq 3 ]] && return 0
    else
      consecutivas=0
    fi
    sleep 0.05
  done
  fallar 'PostgreSQL 18.4 primario no quedó disponible'
}

psql_archivo() {
  local nombre_base=$1 archivo=$2
  shift 2
  docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$nombre_base" "$@" \
    < "$raiz/$archivo"
}

psql_admin() {
  local nombre_base=$1
  docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$nombre_base"
}

consulta() {
  local sql=$1
  docker exec "$contenedor" psql -XAt --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command "$sql"
}

retirar() {
  docker exec --interactive \
    --env PGOPTIONS="-c vec.confirmar_retirada_acreditacion_contexto_actor_v2=RETIRAR_ACREDITACION_CONTEXTO_ACTOR_V2" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/$down"
}

instalar() {
  psql_archivo "$base" "$up"
}

normalizar_dump() {
  sed -E '/^\\(un)?restrict /d'
}

# Para un rechazo se exige identidad física: esquema y datos completos, OID
# de los objetos gestionados y dependencias que los alcanzan.
huella_fisica() {
  {
    docker exec "$contenedor" pg_dump --schema-only \
      --username postgres --dbname "$base" | normalizar_dump
    docker exec "$contenedor" pg_dump --data-only --column-inserts \
      --username postgres --dbname "$base" | normalizar_dump
    consulta "
WITH objetos AS (
  SELECT 'pg_namespace'::regclass::oid AS clase, n.oid AS objeto, 0 AS subobjeto
    FROM pg_catalog.pg_namespace AS n
   WHERE n.nspname='vec_contexto_actor_v1'
  UNION ALL
  SELECT 'pg_class'::regclass::oid, c.oid, 0
    FROM pg_catalog.pg_class AS c
    JOIN pg_catalog.pg_namespace AS n ON n.oid=c.relnamespace
   WHERE n.nspname='vec_contexto_actor_v1'
      OR c.oid IN (
           SELECT reltoastrelid FROM pg_catalog.pg_class
            WHERE oid='vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass
         )
      OR c.oid IN (
           SELECT i.indexrelid FROM pg_catalog.pg_index AS i
            WHERE i.indrelid IN (
              SELECT reltoastrelid FROM pg_catalog.pg_class
               WHERE oid='vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass
            )
         )
  UNION ALL
  SELECT 'pg_proc'::regclass::oid, p.oid, 0
    FROM pg_catalog.pg_proc AS p
    JOIN pg_catalog.pg_namespace AS n ON n.oid=p.pronamespace
   WHERE n.nspname='vec_contexto_actor_v1'
  UNION ALL
  SELECT 'pg_type'::regclass::oid, t.oid, 0
    FROM pg_catalog.pg_type AS t
    JOIN pg_catalog.pg_namespace AS n ON n.oid=t.typnamespace
   WHERE n.nspname='vec_contexto_actor_v1'
  UNION ALL
  SELECT 'pg_trigger'::regclass::oid, t.oid, 0
    FROM pg_catalog.pg_trigger AS t
   WHERE t.tgrelid IN (
     SELECT c.oid FROM pg_catalog.pg_class AS c
      JOIN pg_catalog.pg_namespace AS n ON n.oid=c.relnamespace
     WHERE n.nspname='vec_contexto_actor_v1'
   )
)
SELECT linea FROM (
  SELECT pg_catalog.format(
           'o|%s|%s|%s', clase::regclass::text, objeto, subobjeto
         ) AS linea
    FROM objetos
  UNION ALL
  SELECT pg_catalog.format(
           'd|%s|%s|%s|%s|%s|%s|%s',
           d.classid, d.objid, d.objsubid, d.refclassid,
           d.refobjid, d.refobjsubid, d.deptype
         )
    FROM pg_catalog.pg_depend AS d
   WHERE EXISTS (
     SELECT 1 FROM objetos AS o
      WHERE (o.clase,o.objeto)=(d.classid,d.objid)
         OR (o.clase,o.objeto)=(d.refclassid,d.refobjid)
   )
) AS lineas
ORDER BY linea"
  } | sha256sum | cut -d' ' -f1
}

# Entre reinstalaciones cambian necesariamente los OID y actualizada_en. Esta
# huella exige el mismo contrato y los mismos datos autoritativos, normalizando
# únicamente ese instante técnico de bootstrap.
huella_logica() {
  {
    docker exec "$contenedor" pg_dump --schema-only \
      --username postgres --dbname "$base" | normalizar_dump
    docker exec "$contenedor" pg_dump --data-only --column-inserts \
      --exclude-table-data=vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 \
      --username postgres --dbname "$base" | normalizar_dump
    consulta "
SELECT control_id::text || '|' || generacion::text
  FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
 ORDER BY control_id"
  } | sha256sum | cut -d' ' -f1
}

identidades_000002() {
  consulta "
SELECT oid FROM (
  SELECT c.oid
    FROM pg_catalog.pg_class AS c
   WHERE c.oid='vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass
  UNION ALL
  SELECT p.oid
    FROM pg_catalog.pg_proc AS p
   WHERE p.oid IN (
     'vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2()'::regprocedure,
     'vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2()'::regprocedure,
     'vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)'::regprocedure
   )
  UNION ALL
  SELECT t.oid
    FROM pg_catalog.pg_trigger AS t
   WHERE NOT t.tgisinternal
     AND t.tgname IN (
       'puntero_actual_no_truncable_v2',
       'serializar_mutacion_punteros_actuales_v2',
       'avanzar_generacion_punteros_actuales_v2'
     )
) AS objetos ORDER BY oid"
}

exigir_instalacion_exacta() {
  local observado
  observado=$(consulta "
SELECT
  (SELECT count(*) FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2)
  || '|' ||
  (SELECT count(*) FROM pg_catalog.pg_proc
    WHERE oid IN (
      'vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2()'::regprocedure,
      'vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2()'::regprocedure,
      'vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)'::regprocedure
    ))
  || '|' ||
  (SELECT count(*) FROM pg_catalog.pg_trigger
    WHERE NOT tgisinternal
      AND tgname IN (
        'puntero_actual_no_truncable_v2',
        'serializar_mutacion_punteros_actuales_v2',
        'avanzar_generacion_punteros_actuales_v2'
      ))
  || '|' ||
  (SELECT pg_catalog.bool_and(control_id AND generacion=0)
     FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2)")
  [[ $observado == '1|3|15|true' ]] ||
    fallar "instalación 000002 no exacta: $observado"
}

exigir_retirada_completa() {
  local observado
  observado=$(consulta "
SELECT
  (pg_catalog.to_regclass(
     'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'
   ) IS NULL)::text
  || '|' ||
  (SELECT count(*) FROM pg_catalog.pg_proc
    WHERE pronamespace='vec_contexto_actor_v1'::regnamespace
      AND proname IN (
        'serializar_mutacion_punteros_actuales_v2',
        'avanzar_generacion_punteros_actuales_v2',
        'acreditar_uso_registro_contexto_actor_v2'
      ))
  || '|' ||
  (SELECT count(*) FROM pg_catalog.pg_trigger
    WHERE NOT tgisinternal
      AND tgname IN (
        'puntero_actual_no_truncable_v2',
        'serializar_mutacion_punteros_actuales_v2',
        'avanzar_generacion_punteros_actuales_v2'
      ))
  || '|' ||
  (pg_catalog.to_regclass(
     'vec_contexto_actor_v1.registros_contexto'
   ) IS NOT NULL)::text")
  [[ $observado == 'true|0|0|true' ]] ||
    fallar "retirada 000002 parcial: $observado"
}

exigir_fallo_preservado() {
  local descripcion=$1 modo=${2:-down}
  local antes despues salida estado
  antes=$(huella_fisica)
  set +e
  case "$modo" in
    down) salida=$(retirar 2>&1); estado=$? ;;
    down_sin_confirmacion)
      salida=$(psql_archivo "$base" "$down" 2>&1); estado=$?
      ;;
    up) salida=$(instalar 2>&1); estado=$? ;;
    *) fallar "modo de rechazo desconocido: $modo" ;;
  esac
  set -e
  [[ $estado -ne 0 ]] ||
    fallar "operación aceptada indebidamente: $descripcion"
  despues=$(huella_fisica)
  if [[ $antes != "$despues" ]]; then
    echo "$salida" >&2
    fallar "estado alterado tras rechazo: $descripcion"
  fi
}

esperar_sesion() {
  local aplicacion=$1 condicion=$2 descripcion=$3
  local observado
  for _ in $(seq 1 300); do
    observado=$(consulta "
SELECT count(*) FROM pg_catalog.pg_stat_activity
 WHERE application_name='$aplicacion' AND ($condicion)")
    [[ $observado == 1 ]] && return 0
    sleep 0.02
  done
  fallar "no se observó $descripcion"
}

iniciar_bloqueador() {
  local aplicacion=$1
  docker exec --env PGAPPNAME="$aplicacion" "$contenedor" \
    psql -Xq --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    --command \
    "BEGIN;
     LOCK TABLE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
       IN ACCESS SHARE MODE;
     SELECT pg_catalog.pg_sleep(300);
     ROLLBACK" >/dev/null 2>&1 &
  proceso_iniciado=$!
  esperar_sesion "$aplicacion" \
    "state='active' AND query LIKE '%pg_sleep(300)%'" \
    "el bloqueador $aplicacion"
}

iniciar_retirada() {
  local aplicacion=$1 salida=$2
  docker exec --interactive --env PGAPPNAME="$aplicacion" \
    --env PGOPTIONS="-c vec.confirmar_retirada_acreditacion_contexto_actor_v2=RETIRAR_ACREDITACION_CONTEXTO_ACTOR_V2" \
    "$contenedor" psql -Xq --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    < "$raiz/$down" >"$salida" 2>&1 &
  proceso_iniciado=$!
}

terminar_backend() {
  local aplicacion=$1
  [[ $(consulta "
SELECT pg_catalog.pg_terminate_backend(pid)
  FROM pg_catalog.pg_stat_activity
 WHERE application_name='$aplicacion'") == t ]]
}

comparar_oid_nuevos() {
  local anteriores=$1 actuales=$2
  if comm -12 <(printf '%s\n' "$anteriores" | sort -n) \
    <(printf '%s\n' "$actuales" | sort -n) | grep -q .; then
    fallar 'la reinstalación reutilizó una identidad OID de 000002'
  fi
}

docker run --detach --rm --name "$contenedor" \
  --env POSTGRES_DB="$base_control" \
  --env POSTGRES_PASSWORD="$clave_admin" \
  "$imagen" >/dev/null
esperar_postgres

[[ $(docker exec "$contenedor" psql -XAt --set ON_ERROR_STOP=1 \
  --username postgres --dbname "$base_control" \
  --command "SELECT current_setting('server_version_num')") == 180004 ]] ||
  fallar 'se requiere PostgreSQL 18.4 exacto'

psql_admin "$base_control" <<'SQL'
DO $base$
BEGIN
  EXECUTE pg_catalog.format(
    'REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',
    pg_catalog.current_database()
  );
END
$base$;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
psql_archivo "$base_control" "$roles"
psql_admin "$base_control" <<'SQL'
DO $base$
BEGIN
  EXECUTE pg_catalog.format(
    'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_contexto_actor_v1_propietario, vec_contexto_actor_v1_migrador, vec_contexto_actor_v1_runtime',
    pg_catalog.current_database()
  );
END
$base$;
SQL

docker exec "$contenedor" createdb --username postgres "$base"
psql_admin "$base" <<'SQL'
DO $base$
BEGIN
  EXECUTE pg_catalog.format(
    'REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',
    pg_catalog.current_database()
  );
  EXECUTE pg_catalog.format(
    'GRANT CONNECT ON DATABASE %I TO vec_contexto_actor_v1_propietario, vec_contexto_actor_v1_migrador, vec_contexto_actor_v1_runtime',
    pg_catalog.current_database()
  );
END
$base$;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
psql_archivo "$base" "$up_base"
instalar
exigir_instalacion_exacta
huella_inicial=$(huella_logica)
oid_iniciales=$(identidades_000002)
[[ $(printf '%s\n' "$oid_iniciales" | sed '/^$/d' | wc -l) == 19 ]] ||
  fallar 'inventario OID inicial de 000002 incompleto'

# Confirmación ausente y reentrada de up: ambas deben abortar sin tocar ni un
# OID, dato o dependencia.
exigir_fallo_preservado 'confirmación ausente' down_sin_confirmacion
exigir_fallo_preservado 'reentrada de 000002' up

# Un consumidor exterior provoca el fallo final con RESTRICT. Aunque los
# triggers y funciones se hayan intentado retirar antes, el rollback debe
# restaurar también sus OID y todos los datos.
psql_admin "$base" <<'SQL'
CREATE SCHEMA ct133_consumidor AUTHORIZATION postgres;
CREATE VIEW ct133_consumidor.generacion_contexto_actor AS
SELECT control_id, generacion, actualizada_en
  FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2;
SQL
exigir_fallo_preservado 'consumidor exterior de la generación'
psql_admin "$base" <<'SQL'
DROP VIEW ct133_consumidor.generacion_contexto_actor;
DROP SCHEMA ct133_consumidor RESTRICT;
SQL
[[ $(huella_logica) == "$huella_inicial" ]] ||
  fallar 'el escenario de consumidor no restauró el estado de partida'

# Cancelación explícita durante una espera real: la sesión de retirada se
# identifica por application_name y se cancela solo tras observar el wait.
huella_antes=$(huella_fisica)
salida_cancelacion=$(mktemp "$raiz/.ct133-cancelacion.XXXXXX")
temporales+=("$salida_cancelacion")
iniciar_bloqueador ct133_bloqueador_cancelacion
proceso_bloqueador=$proceso_iniciado
iniciar_retirada ct133_retirada_cancelada "$salida_cancelacion"
proceso_retirada=$proceso_iniciado
esperar_sesion ct133_retirada_cancelada \
  "wait_event_type='Lock' AND wait_event='relation'" \
  'la retirada esperando la relación antes de cancelarla'
pid_retirada=$(consulta "
SELECT pid FROM pg_catalog.pg_stat_activity
 WHERE application_name='ct133_retirada_cancelada'")
[[ $(consulta "SELECT pg_catalog.pg_cancel_backend($pid_retirada)") == t ]]
set +e
wait "$proceso_retirada"
estado_cancelacion=$?
set -e
[[ $estado_cancelacion -ne 0 ]] ||
  fallar 'la retirada cancelada terminó con éxito'
terminar_backend ct133_bloqueador_cancelacion
wait "$proceso_bloqueador" >/dev/null 2>&1 || true
[[ $(huella_fisica) == "$huella_antes" ]] ||
  fallar 'la cancelación no preservó datos y catálogo'
[[ $(consulta "
SELECT pg_catalog.current_setting(
  'vec.confirmar_retirada_acreditacion_contexto_actor_v2', true
) IS NULL") == t ]] ||
  fallar 'una conexión nueva heredó la confirmación de retirada'
exigir_instalacion_exacta

# lock_timeout debe cerrar la contención antes del límite global de sentencia.
# La espera se acredita en pg_stat_activity; el reloj solo acota su resultado.
grep -Fq "SET LOCAL lock_timeout = '5s';" "$down" ||
  fallar 'falta el límite local de bloqueo de cinco segundos'
grep -Fq "SET LOCAL statement_timeout = '30s';" "$down" ||
  fallar 'falta el límite local de sentencia de treinta segundos'
huella_antes=$(huella_fisica)
salida_limite=$(mktemp "$raiz/.ct133-limite.XXXXXX")
temporales+=("$salida_limite")
iniciar_bloqueador ct133_bloqueador_limite
proceso_bloqueador=$proceso_iniciado
inicio_ns=$(date +%s%N)
iniciar_retirada ct133_retirada_limite "$salida_limite"
proceso_retirada=$proceso_iniciado
esperar_sesion ct133_retirada_limite \
  "wait_event_type='Lock' AND wait_event='relation'" \
  'la retirada sometida al límite de bloqueo'
set +e
wait "$proceso_retirada"
estado_limite=$?
set -e
fin_ns=$(date +%s%N)
transcurrido_ms=$(((fin_ns - inicio_ns) / 1000000))
[[ $estado_limite -ne 0 ]] ||
  fallar 'la retirada superó indebidamente la contención'
[[ $transcurrido_ms -ge 4000 && $transcurrido_ms -le 12000 ]] ||
  fallar "lock_timeout fuera de cota: ${transcurrido_ms}ms"
grep -Fq 'canceling statement due to lock timeout' "$salida_limite" ||
  fallar 'la contención no terminó por lock_timeout'
terminar_backend ct133_bloqueador_limite
wait "$proceso_bloqueador" >/dev/null 2>&1 || true
[[ $(huella_fisica) == "$huella_antes" ]] ||
  fallar 'el límite de bloqueo no preservó datos y catálogo'
exigir_instalacion_exacta

# Los nombres de los tres roles forman parte del manifiesto. Una barrera de
# prueba externa detiene el primer DROP después de acreditar el inventario;
# un RENAME concurrente debe quedar bloqueado o hacer fallar cerrada toda la
# retirada. El event trigger no depende de objetos del módulo.
psql_admin "$base" <<'SQL'
CREATE FUNCTION public.ct133_barrera_posterior_inventario()
RETURNS event_trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $barrera$
BEGIN
  IF pg_catalog.current_setting('application_name') =
       'ct133_retirada_renombrado' THEN
    PERFORM pg_catalog.pg_advisory_xact_lock(
      pg_catalog.hashtextextended('ct133:posterior-inventario', 0)
    );
  END IF;
END
$barrera$;
CREATE EVENT TRIGGER ct133_barrera_posterior_inventario
  ON ddl_command_start
  WHEN TAG IN ('DROP TRIGGER')
  EXECUTE FUNCTION public.ct133_barrera_posterior_inventario();
SQL
docker exec --env PGAPPNAME=ct133_barrera_renombrado "$contenedor" \
  psql -Xq --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
  --command \
  "SELECT pg_catalog.pg_advisory_lock(
     pg_catalog.hashtextextended('ct133:posterior-inventario',0));
   SELECT pg_catalog.pg_sleep(300)" >/dev/null 2>&1 &
proceso_barrera=$!
esperar_sesion ct133_barrera_renombrado \
  "state='active' AND query LIKE '%pg_sleep(300)%'" \
  'la barrera posterior al inventario'
salida_renombrado=$(mktemp "$raiz/.ct133-renombrado.XXXXXX")
temporales+=("$salida_renombrado")
iniciar_retirada ct133_retirada_renombrado "$salida_renombrado"
proceso_retirada=$proceso_iniciado
esperar_sesion ct133_retirada_renombrado \
  "wait_event_type='Lock' AND wait_event='advisory'" \
  'la retirada posterior al inventario'
salida_mutacion_rol=$(mktemp "$raiz/.ct133-mutacion-rol.XXXXXX")
temporales+=("$salida_mutacion_rol")
docker exec --env PGAPPNAME=ct133_renombrado_concurrente "$contenedor" \
  psql -Xq --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
  --command \
  "SET lock_timeout='2s';
   ALTER ROLE vec_contexto_actor_v1_runtime
     RENAME TO ct133_runtime_renombrado" >"$salida_mutacion_rol" 2>&1 &
proceso_mutacion_rol=$!
mutacion_rol_bloqueada=false
for _ in $(seq 1 150); do
  if [[ $(consulta "
SELECT count(*) FROM pg_catalog.pg_stat_activity
 WHERE application_name='ct133_renombrado_concurrente'
   AND wait_event_type='Lock'") == 1 ]]; then
    mutacion_rol_bloqueada=true
    break
  fi
  kill -0 "$proceso_mutacion_rol" >/dev/null 2>&1 || break
  sleep 0.02
done
if [[ $mutacion_rol_bloqueada != true ]]; then
  set +e
  wait "$proceso_mutacion_rol"
  estado_mutacion_rol=$?
  set -e
  [[ $estado_mutacion_rol -ne 0 ]] ||
    fallar 'el renombrado de rol atravesó el inventario y la retirada'
  fallar 'el renombrado concurrente no quedó serializado por el down'
fi
set +e
wait "$proceso_mutacion_rol"
estado_mutacion_rol=$?
set -e
[[ $estado_mutacion_rol -ne 0 ]] ||
  fallar 'el renombrado bloqueado terminó antes de cerrar la retirada'
grep -Fq 'canceling statement due to lock timeout' "$salida_mutacion_rol" ||
  fallar 'el renombrado no terminó por su límite de bloqueo'
terminar_backend ct133_barrera_renombrado
wait "$proceso_barrera" >/dev/null 2>&1 || true
wait "$proceso_retirada"
exigir_retirada_completa
[[ $(consulta "
SELECT count(*) FROM pg_catalog.pg_roles
 WHERE rolname='vec_contexto_actor_v1_runtime'
   AND NOT rolcanlogin") == 1 ]] ||
  fallar 'la retirada no conservó el nombre nominal del rol runtime'
[[ $(consulta "
SELECT count(*) FROM pg_catalog.pg_roles
 WHERE rolname='ct133_runtime_renombrado'") == 0 ]] ||
  fallar 'sobrevivió el nombre hostil del rol runtime'
psql_admin "$base" <<'SQL'
DROP EVENT TRIGGER ct133_barrera_posterior_inventario;
DROP FUNCTION public.ct133_barrera_posterior_inventario();
SQL
instalar
exigir_instalacion_exacta
oid_posteriores_rol=$(identidades_000002)
comparar_oid_nuevos "$oid_iniciales" "$oid_posteriores_rol"
oid_iniciales=$oid_posteriores_rol
[[ $(huella_logica) == "$huella_inicial" ]] ||
  fallar 'la carrera de rol no restauró el contrato lógico'

# Carrera soportada: una ACL nacida después de la petición de ACCESS
# EXCLUSIVE queda en cola. Al liberar el lector, la retirada gana, es atómica
# y la ACL falla sobre el objeto ya retirado; nunca queda una instalación
# parcial.
salida_carrera=$(mktemp "$raiz/.ct133-carrera.XXXXXX")
salida_acl=$(mktemp "$raiz/.ct133-acl.XXXXXX")
temporales+=("$salida_carrera" "$salida_acl")
iniciar_bloqueador ct133_bloqueador_carrera
proceso_bloqueador=$proceso_iniciado
iniciar_retirada ct133_retirada_carrera "$salida_carrera"
proceso_retirada=$proceso_iniciado
esperar_sesion ct133_retirada_carrera \
  "wait_event_type='Lock' AND wait_event='relation'" \
  'la retirada en la carrera ACL'
docker exec --env PGAPPNAME=ct133_acl_concurrente "$contenedor" \
  psql -Xq --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
  --command \
  'GRANT SELECT ON vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 TO vec_contexto_actor_v1_runtime' \
  >"$salida_acl" 2>&1 &
proceso_acl=$!
esperar_sesion ct133_acl_concurrente \
  "wait_event_type='Lock' AND wait_event='relation'" \
  'la ACL concurrente en cola'
terminar_backend ct133_bloqueador_carrera
wait "$proceso_bloqueador" >/dev/null 2>&1 || true
wait "$proceso_retirada"
set +e
wait "$proceso_acl"
estado_acl=$?
set -e
[[ $estado_acl -ne 0 ]] ||
  fallar 'la ACL atravesó la ventana exclusiva'
exigir_retirada_completa
[[ $(consulta "
SELECT count(*) FROM pg_catalog.pg_locks
 WHERE granted=false
   AND pid IN (
     SELECT pid FROM pg_catalog.pg_stat_activity
      WHERE application_name LIKE 'ct133_%'
   )") == 0 ]] ||
  fallar 'quedaron esperas de la carrera'

instalar
exigir_instalacion_exacta
oid_actuales=$(identidades_000002)
comparar_oid_nuevos "$oid_iniciales" "$oid_actuales"
[[ $(huella_logica) == "$huella_inicial" ]] ||
  fallar 'la primera reinstalación no restauró el contrato lógico'

# Dos ciclos adicionales prueban reentrada, retirada, OID frescos y
# restauración del mismo contrato sin depender del primer orden físico.
for ciclo in 2 3; do
  exigir_fallo_preservado "reentrada del ciclo $ciclo" up
  oid_anteriores=$(identidades_000002)
  retirar
  exigir_retirada_completa
  instalar
  exigir_instalacion_exacta
  oid_actuales=$(identidades_000002)
  comparar_oid_nuevos "$oid_anteriores" "$oid_actuales"
  [[ $(huella_logica) == "$huella_inicial" ]] ||
    fallar "el ciclo $ciclo no restauró el contrato lógico"
done

[[ $(consulta "
SELECT count(*) FROM pg_catalog.pg_stat_activity
 WHERE application_name LIKE 'ct133_%'") == 0 ]] ||
  fallar 'quedó una sesión de prueba contaminada'
[[ $(consulta "
SELECT pg_catalog.to_regnamespace('ct133_consumidor') IS NULL") == t ]] ||
  fallar 'quedó el consumidor de prueba'
exigir_instalacion_exacta
[[ $(huella_logica) == "$huella_inicial" ]] ||
  fallar 'la restauración final no es exacta'

echo 'ContextoActor V2 S0.2c: concurrencia, preservación y ciclos PostgreSQL 18.4 superados'
