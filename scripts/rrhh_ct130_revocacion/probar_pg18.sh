#!/usr/bin/env bash
# CT130: preimagen central V3 real sobre PostgreSQL 18.4 desechable.
# Solo datos sintéticos. Requiere Docker y Go; no usa bases ni volcados privados.
set -Eeuo pipefail

script=$(realpath -- "${BASH_SOURCE[0]}")
repo=$(cd "$(dirname "$script")/../.." && pwd)
cd "$repo"
candidato=1530394862f482bb979476169640d5796c6582d8
git merge-base --is-ancestor "$candidato" HEAD || {
  echo 'Falta el candidato CT130 con perfil dedicado en este checkout' >&2
  exit 2
}
if [[ -n $(git status --porcelain) ]] || ! git diff --quiet "$candidato" HEAD -- \
    internal go.mod go.sum deploy/postgresql; then
  echo 'El árbol o las fuentes internas/SQL cubiertas difieren del candidato fijado' >&2
  exit 2
fi
printf 'Candidato CT130: %s; checkout: %s\n' \
  "$candidato" "$(git rev-parse HEAD)"

imagen=${VEC_CT130_PG18_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
base=vec_ct130_revocacion_prueba
contenedor="vec-ct130-revocacion-$$"
container_id=
socket_parent="/tmp/vec-ct130-pg18-$(id -u)"
mkdir -p "$socket_parent"
chmod 0700 "$socket_parent"
socket_dir=$(mktemp -d "$socket_parent/socket.XXXXXX")
chmod 0777 "$socket_dir"
pgdata_parent="/dev/shm/vec-ct130-pg18-$(id -u)"
mkdir -p "$pgdata_parent"
chmod 0700 "$pgdata_parent"
pgdata_dir=$(mktemp -d "$pgdata_parent/data.XXXXXX")
chmod 0777 "$pgdata_dir"
clave_admin=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
clave_gobierno=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')

limpiar() {
  if [[ -n $container_id ]]; then
    docker exec "$container_id" sh -c 'rm -f /var/run/postgresql/.s.PGSQL.*' >/dev/null 2>&1 || true
    docker rm -f "$container_id" >/dev/null 2>&1 || true
  fi
  if ! rm -rf -- "$socket_dir" 2>/dev/null; then
    docker run --rm --network none --entrypoint sh \
      --mount "type=bind,src=$socket_dir,dst=/cleanup" "$imagen" \
      -c 'chmod 0777 /cleanup; rm -f /cleanup/.s.PGSQL.*' >/dev/null 2>&1 || true
    rm -rf -- "$socket_dir" || echo "Revisar limpieza del socket efímero: $socket_dir" >&2
  fi
  rmdir "$socket_parent" 2>/dev/null || true
  if ! rm -rf -- "$pgdata_dir" 2>/dev/null; then
    docker run --rm --network none --entrypoint sh \
      --mount "type=bind,src=$pgdata_dir,dst=/cleanup" "$imagen" \
      -c 'rm -rf /cleanup/* /cleanup/.[!.]* /cleanup/..?*; chmod 0777 /cleanup' >/dev/null 2>&1 || true
    rm -rf -- "$pgdata_dir" || echo "Revisar limpieza del PGDATA efímero: $pgdata_dir" >&2
  fi
  rmdir "$pgdata_parent" 2>/dev/null || true
}
trap limpiar EXIT INT TERM

container_id=$(docker run --detach --rm --network none --name "$contenedor" \
  --shm-size 64m --mount "type=bind,src=$socket_dir,dst=/var/run/postgresql" \
  --mount "type=bind,src=$pgdata_dir,dst=/var/lib/postgresql" \
  --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave_admin" \
  --env POSTGRES_INITDB_ARGS='--auth-local=scram-sha-256' \
  "$imagen")

psql_admin() {
  docker exec --interactive --env PGPASSWORD="$clave_admin" "$container_id" \
    psql -Xq --set ON_ERROR_STOP=1 -h /var/run/postgresql -U postgres -d "$base" "$@"
}
archivo() { psql_admin < "$repo/$1"; }
valor() { psql_admin -At -c "$1"; }

listo=false
for _ in $(seq 1 120); do
  if [[ $(valor "SELECT current_setting('server_version_num') || '|' || pg_is_in_recovery()" 2>/dev/null || true) == '180004|false' ]]; then
    listo=true
    break
  fi
  sleep 0.5
done
[[ $listo == true ]] || {
  echo 'PostgreSQL 18.4 no quedó disponible' >&2
  docker logs --tail 80 "$container_id" >&2 || true
  exit 1
}
[[ $(valor "SELECT bool_and(auth_method='scram-sha-256') FROM pg_hba_file_rules WHERE type='local' AND error IS NULL") == t ]] || {
  echo 'El socket local no exige SCRAM para todas las entradas' >&2
  exit 1
}

# La base dedicada llega sin concesiones PUBLIC antes de los bootstraps reales.
psql_admin <<'SQL'
DO $b$ BEGIN
  EXECUTE format('REVOKE ALL ON DATABASE %I FROM PUBLIC', current_database());
END $b$;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL

# Se instalan funciones, restricciones, RLS y ACL originales; ningún doble SQL.
for sql in \
  deploy/postgresql/contexto_actor_v1/roles_up.sql \
  deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql \
  deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql \
  deploy/postgresql/autorizacion/roles_up.sql \
  deploy/postgresql/autorizacion/roles_v2_up.sql \
  deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql \
  deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql \
  deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql \
  deploy/postgresql/autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql \
  deploy/postgresql/autorizacion/migraciones/000005_registro_decisiones_contexto_actor_v3.up.sql
do
  archivo "$sql" >/dev/null
done

# Login nominal de prueba: solo puede adoptar, dentro de cada transacción, los
# dos propietarios que el propio seam usa. No es superusuario ni bypass RLS.
docker exec --interactive --env PGPASSWORD="$clave_admin" \
  --env CLAVE_GOBIERNO="$clave_gobierno" "$container_id" \
  psql -Xq --set ON_ERROR_STOP=1 -h /var/run/postgresql -U postgres -d "$base" <<'SQL'
\getenv clave_gobierno CLAVE_GOBIERNO
CREATE ROLE vec_ct130_gobierno_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  NOINHERIT NOREPLICATION NOBYPASSRLS PASSWORD :'clave_gobierno';
GRANT CONNECT ON DATABASE vec_ct130_revocacion_prueba TO vec_ct130_gobierno_prueba;
GRANT vec_contexto_actor_v1_propietario TO vec_ct130_gobierno_prueba
  WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
GRANT vec_autorizacion_propietario TO vec_ct130_gobierno_prueba
  WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
SQL

[[ $(valor "SELECT NOT rolsuper AND NOT rolbypassrls AND rolcanlogin FROM pg_roles WHERE rolname='vec_ct130_gobierno_prueba'") == t ]] || {
  echo 'Login de gobierno de prueba inválido' >&2; exit 1;
}
[[ $(valor "SELECT to_regclass('vec_autorizacion.control_sesion_actual_v1') IS NOT NULL AND to_regclass('vec_contexto_actor_v1.registros_contexto') IS NOT NULL") == t ]] || {
  echo 'Preimagen V3 incompleta' >&2; exit 1;
}

huella_datos() {
  local tablas tabla huella
  tablas=$(valor "SELECT format('%I.%I',n.nspname,c.relname)
    FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
    WHERE n.nspname IN ('vec_autorizacion','vec_contexto_actor_v1')
      AND c.relkind='r' ORDER BY n.nspname,c.relname") || return 1
  [[ -n $tablas ]] || return 1
  while IFS= read -r tabla; do
    huella=$(valor "SELECT md5(coalesce(string_agg(to_jsonb(t)::text,'|' ORDER BY to_jsonb(t)::text),'')) FROM $tabla AS t") || return 1
    [[ $huella =~ ^[0-9a-f]{32}$ ]] || return 1
    printf '%s %s\n' "$tabla" "$huella"
  done <<< "$tablas" | sha256sum | cut -d' ' -f1
}

estado_central() {
  valor "SELECT
    (SELECT count(*) FROM vec_autorizacion.asignacion_perfil_actual x
      JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=x.asignacion_ref
      WHERE a.documento->>'estado'='revocada')::text || '|' ||
    (SELECT count(*) FROM vec_autorizacion.control_sesion_actual_v1 x
      JOIN vec_autorizacion.control_sesion_v1 s ON s.control_sesion_ref=x.control_sesion_ref
        AND s.revision=x.revision WHERE s.estado='activa')::text"
}

# pgx accede al socket montado, sin puerto publicado y sin red del contenedor.
export VEC_CT130_REVOCACION_PG_DESECHABLE=1
export VEC_CT130_REVOCACION_PG_DSN_GOBIERNO="host=$socket_dir user=vec_ct130_gobierno_prueba password=$clave_gobierno dbname=$base sslmode=disable"
go test ./internal/app/bootstrap -run '^TestCT130(PreimagenCentralPostgreSQL|PublicacionSinPreimagenPreparadaFallaCerrada)$' -count=1 -v
go test ./internal/modules/contrataciontemporal/application \
  -run '^TestReincorporacionDenegadaAntesDeLectorYRepositorio$' -count=1

huella_antes=$(huella_datos)
estado_antes=$(estado_central)
IFS='|' read -r revocadas_antes sesiones_antes <<< "$estado_antes"
[[ $revocadas_antes -ge 1 && $sesiones_antes -ge 1 ]] || {
  echo 'El test no dejó revocación central y sesión viva verificables' >&2
  exit 1
}
docker restart "$container_id" >/dev/null
listo=false
for _ in $(seq 1 120); do
  if [[ $(valor "SELECT current_setting('server_version_num') || '|' || pg_is_in_recovery()" 2>/dev/null || true) == '180004|false' ]]; then
    listo=true
    break
  fi
  sleep 0.5
done
[[ $listo == true ]] || { echo 'PostgreSQL no se recuperó tras reinicio real' >&2; exit 1; }
[[ $(huella_datos) == "$huella_antes" && $(estado_central) == "$estado_antes" ]] || {
  echo 'La historia, revocación o sesión cambió tras reiniciar PostgreSQL' >&2
  exit 1
}

echo 'OK CT130 sobre PostgreSQL 18.4 efímero con ContextoActor y Autorización reales'
echo "OK reinicio PostgreSQL: $revocadas_antes asignaciones actuales revocadas y $sesiones_antes sesiones activas, huella de todas las tablas centrales idéntica"
echo 'Alcance: referencias sintéticas de expediente, perfiles separados, sesión viva, revocación/restricción, CAS concurrente, recomposición base y replay.'
echo 'La prueba de aplicación deniega antes de lector/repositorio; no hay tablas CT/Bolsa ni recorrido HTTP en este fixture.'
sha256sum "$script"
