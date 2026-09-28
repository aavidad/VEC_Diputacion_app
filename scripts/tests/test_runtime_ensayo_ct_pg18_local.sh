#!/usr/bin/env bash
set -Eeuo pipefail
repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
script="$repo/scripts/runtime_ensayo_ct_pg18_local.sh"
temporal=$(mktemp -d /tmp/vec-runtime-ct-guardas.XXXXXXXX)
trap 'rm -rf -- "$temporal"' EXIT

esperar_fallo() {
  local esperado=$1
  shift
  if "$@" >"$temporal/salida" 2>"$temporal/error"; then
    printf 'Runtime aceptó entrada insegura: %s\n' "$esperado" >&2
    exit 1
  fi
  grep -Fq -- "$esperado" "$temporal/error" || {
    printf 'Falta rechazo esperado: %s\n' "$esperado" >&2
    cat "$temporal/error" >&2
    exit 1
  }
}

"$script" --help | grep -Fq -- '--tls-material auto'
esperar_fallo 'no admite argumentos' "$script" argumento
esperar_fallo 'contenedor ajeno al runner' env -u VEC_ENSAYO_CONTENEDOR "$script"
esperar_fallo 'contenedor ajeno al runner' env VEC_ENSAYO_CONTENEDOR='contenedor-compartido' "$script"
esperar_fallo 'directorio efímero ajeno al runner' env \
  VEC_ENSAYO_CONTENEDOR=vec-cadena-pg18-123 \
  VEC_ENSAYO_DIRECTORIO="$temporal" "$script"
printf 'OK runtime CT: rechazo de destinos y argumentos ajenos\n'

if (( $# == 0 )); then exit 0; fi
[[ $# == 1 && $1 == --pg18 ]] || { printf 'Uso: %s [--pg18]\n' "$0" >&2; exit 2; }
[[ -S /var/run/docker.sock && -w /dev/shm ]] || { echo 'Docker local y /dev/shm escribible requeridos para --pg18' >&2; exit 2; }
unset DOCKER_CONTEXT
export DOCKER_HOST=unix:///var/run/docker.sock
docker image inspect postgres:18.4-alpine >/dev/null 2>&1 || { echo 'Falta imagen local postgres:18.4-alpine' >&2; exit 2; }

ensayo=$(mktemp -d /dev/shm/vec-cadena-pg18.XXXXXXXX)
contenedor="vec-cadena-pg18-$$"
limpiar_pg() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  docker run --rm --network none --user 0 \
    --mount "type=bind,src=$ensayo/data,dst=/datos" \
    --entrypoint sh postgres:18.4-alpine -c 'rm -rf /datos/pgdata' >/dev/null 2>&1 || true
  rm -rf -- "$ensayo" 2>/dev/null || true
}
trap 'limpiar_pg; rm -rf -- "$temporal"' EXIT
mkdir "$ensayo/data"
chmod 1777 "$ensayo/data"
docker run -d --name "$contenedor" \
  --mount "type=bind,src=$ensayo/data,dst=/var/lib/postgresql/data" \
  -e PGDATA=/var/lib/postgresql/data/pgdata -e POSTGRES_HOST_AUTH_METHOD=trust \
  -p 127.0.0.1::5432 postgres:18.4-alpine >/dev/null
for (( i=0; i<100; i++ )); do
  if docker exec "$contenedor" pg_isready -q -U postgres -d postgres 2>/dev/null; then break; fi
  sleep 0.1
done
docker exec "$contenedor" pg_isready -q -U postgres -d postgres 2>/dev/null || { echo 'PG18 no arrancó' >&2; exit 1; }
docker exec "$contenedor" createdb -U postgres vec_ensayo

grupos=(
  vec_contratacion_temporal_ejecutor vec_autorizacion_atestada_v3_migrador
  vec_autorizacion_registro vec_contratacion_temporal_confirmador_cobertura
  vec_contratacion_temporal_lector_resultado_cobertura vec_bolsa_llamamientos_ejecutor
  vec_contratacion_temporal_consultor_rrhh vec_autorizacion_motivos_evaluador
  vec_identidad_sesiones_v1_registrador vec_identidad_sesiones_v1_revalidador
  vec_contexto_actor_v1_runtime vec_contratacion_temporal_registrador_frontera
  vec_contexto_actor_v1_propietario vec_autorizacion_propietario
  vec_autorizacion_motivos_proyector vec_identidad_sesiones_v1_propietario
)
for grupo in "${grupos[@]}"; do
  docker exec "$contenedor" psql -Xq -v ON_ERROR_STOP=1 -U postgres -d vec_ensayo \
    -c "CREATE ROLE $grupo NOLOGIN" >/dev/null
done
cat > "$ensayo/vec-server" <<'SH'
#!/usr/bin/env bash
exit 7
SH
chmod 700 "$ensayo/vec-server"
puerto=$(docker port "$contenedor" 5432/tcp | sed -n 's/^127\.0\.0\.1://p')
if env VEC_ENSAYO_CONTENEDOR="$contenedor" VEC_ENSAYO_DIRECTORIO="$ensayo" \
    VEC_ENSAYO_BINARIO="$ensayo/vec-server" VEC_ENSAYO_TLS_MATERIAL="$ensayo/material" \
    VEC_ENSAYO_REPO="$repo" PGHOST=127.0.0.1 PGPORT="$puerto" PGDATABASE=vec_ensayo \
    "$script" >"$temporal/runtime.out" 2>"$temporal/runtime.err"; then
  echo 'Runtime aceptó un binario que terminó antes de /livez' >&2
  exit 1
fi
grep -Fq 'vec-server terminó antes de responder en loopback' "$temporal/runtime.err" || {
  cat "$temporal/runtime.err" >&2
  exit 1
}
cuenta=$(docker exec "$contenedor" psql -XAt -U postgres -d vec_ensayo -c \
  "SELECT count(*) FROM pg_authid r WHERE r.rolname LIKE 'vec_ensayo_ct_%' AND r.rolcanlogin AND r.rolinherit AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls AND r.rolpassword IS NULL")
[[ $cuenta == 12 ]] || { printf 'LOGIN nominales: %s/12\n' "$cuenta" >&2; exit 1; }
membresias=$(docker exec "$contenedor" psql -XAt -U postgres -d vec_ensayo -c \
  "SELECT count(*) FROM pg_auth_members m JOIN pg_roles r ON r.oid=m.member JOIN pg_roles g ON g.oid=m.roleid WHERE r.rolname LIKE 'vec_ensayo_ct_%' AND NOT m.admin_option AND ((r.rolname='vec_ensayo_ct_01' AND g.rolname IN ('vec_autorizacion_atestada_v3_migrador','vec_contexto_actor_v1_propietario','vec_autorizacion_propietario','vec_autorizacion_motivos_proyector','vec_identidad_sesiones_v1_propietario') AND NOT m.inherit_option AND m.set_option) OR (r.rolname<>'vec_ensayo_ct_01' AND m.inherit_option AND NOT m.set_option))")
[[ $membresias == 16 ]] || { printf 'Membresías nominales esperadas: %s/16\n' "$membresias" >&2; exit 1; }
total_membresias=$(docker exec "$contenedor" psql -XAt -U postgres -d vec_ensayo -c \
  "SELECT count(*) FROM pg_auth_members m JOIN pg_roles r ON r.oid=m.member WHERE r.rolname LIKE 'vec_ensayo_ct_%'")
[[ $total_membresias == 16 ]] || { printf 'Membresías ajenas detectadas: %s/16\n' "$total_membresias" >&2; exit 1; }
for rol in vec_autorizacion_atestada_v3_migrador vec_contexto_actor_v1_propietario \
    vec_autorizacion_propietario vec_autorizacion_motivos_proyector \
    vec_identidad_sesiones_v1_propietario; do
  efectivo=$(docker exec "$contenedor" psql -XAt -h 127.0.0.1 -U vec_ensayo_ct_01 -d vec_ensayo \
    -c "BEGIN; SET LOCAL ROLE $rol; SELECT current_user; ROLLBACK;" | grep -Fx "$rol" || true)
  [[ $efectivo == "$rol" ]] || { printf 'Gobierno no puede asumir %s\n' "$rol" >&2; exit 1; }
  if docker exec "$contenedor" psql -Xq -v ON_ERROR_STOP=1 -h 127.0.0.1 -U vec_ensayo_ct_00 -d vec_ensayo \
      -c "BEGIN; SET LOCAL ROLE $rol; ROLLBACK;" >/dev/null 2>&1; then
    printf 'Ejecución asumió rol ajeno: %s\n' "$rol" >&2
    exit 1
  fi
done
[[ -s $ensayo/material/ca/ca.crt && -s $ensayo/material/mtls/cliente.crt ]] || {
  echo 'Material sintético TLS/mTLS ausente' >&2
  exit 1
}
printf 'OK runtime CT: 12 LOGIN nominales y material sintético PG18 local\n'
