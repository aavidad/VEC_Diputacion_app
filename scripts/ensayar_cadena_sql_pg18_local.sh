#!/usr/bin/env bash
# Puerta local: cadena SQL completa, binario sobre PostgreSQL 18.4 y sonda HTTP.
# Uso: scripts/ensayar_cadena_sql_pg18_local.sh --plan ORDEN --runtime EJECUTABLE --probe RUTA [--repo CHECKOUT] [--browser EJECUTABLE]
# ORDEN contiene una ruta relativa deploy/postgresql/... por línea, en orden causal.
# Debe incluir todos los roles*_up.sql y *.up.sql del alcance rastreados por
# Git; puede añadir dependencias SQL rastreadas de los módulos comunes.
# EJECUTABLE configura identidades nominales en la base efímera y arranca el
# binario indicado por VEC_ENSAYO_BINARIO. Recibe PGHOST, PGPORT y PGDATABASE.
# RUTA es una sonda HTTP que realmente consulta PostgreSQL (no /livez).
# EJECUTABLE de navegador recibe VEC_ENSAYO_URL y usa solo datos sintéticos.
set -Eeuo pipefail
IFS=$'\n\t'
umask 077

fallar() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }
ayuda() {
  cat <<'AYUDA'
Uso: ensayar_cadena_sql_pg18_local.sh --plan ORDEN --runtime EJECUTABLE
     --probe RUTA [--repo CHECKOUT] [--scope ct-llamamientos|ct-bolsa]
     [--tls-material DIRECTORIO] [--browser EJECUTABLE]

ORDEN enumera rutas relativas deploy/postgresql/... en orden causal. Debe
contener todos los UP y roles de CT y Bolsa Llamamientos rastreados en CHECKOUT
(o de todos los módulos bolsa_* con --scope ct-bolsa). Puede incluir
dependencias rastreadas de otros módulos. Cada SQL se aplica
una vez, con ON_ERROR_STOP, a una base nueva de PostgreSQL 18.4.

Requiere Docker local accesible, imagen postgres:18.4-alpine ya cargada,
/dev/shm escribible y Go local compatible. La base, binario y logs son
efímeros; PostgreSQL solo publica en 127.0.0.1. No usa datos ni red remotos.

EJECUTABLE recibe PGHOST/PGPORT/PGDATABASE, VEC_ENSAYO_BINARIO,
VEC_ENSAYO_CONTENEDOR y VEC_ENSAYO_DIRECTORIO. Puede crear LOGIN nominales
separados mediante `docker exec` y `psql` dentro del contenedor indicado;
después arranca el binario y escribe
URL=http://127.0.0.1:PUERTO en stdout. Con --tls-material escribe en cambio
URL=https://localhost:PUERTO y se usan ca/ca.crt, mtls/cliente.crt y
mtls/cliente.key de ese directorio privado; localhost se fija a 127.0.0.1.
El runtime y navegador reciben VEC_ENSAYO_TLS_MATERIAL cuando se configura.
RUTA debe ser un GET que consulta la base; se exige respuesta HTTP 2xx y una
conexión PG de aplicación activa.
El navegador es opcional y recibe VEC_ENSAYO_URL para recorrer el caso local.

Para scripts/verificar_calidad.sh, si /tmp/.git existe y los fixtures privados
se rechazan por estar «dentro del repositorio», use TMPDIR=/var/tmp escribible.
AYUDA
}

plan='' runtime='' probe='' browser='' checkout='' scope='ct-llamamientos' tls_material=''
while (( $# )); do
  case "$1" in
    --plan|--runtime|--probe|--browser|--repo|--scope|--tls-material)
      (( $# >= 2 )) || fallar "falta valor para $1"
      case "$1" in
        --plan) plan=$2 ;; --runtime) runtime=$2 ;;
        --probe) probe=$2 ;; --browser) browser=$2 ;; --repo) checkout=$2 ;; --scope) scope=$2 ;;
        --tls-material) tls_material=$2 ;;
      esac
      shift 2 ;;
    -h|--help) ayuda; exit 0 ;;
    *) fallar "argumento desconocido: $1" ;;
  esac
done
[[ -n $plan && -n $runtime && -n $probe ]] || fallar 'uso: --plan ORDEN --runtime EJECUTABLE --probe RUTA [--browser EJECUTABLE]'
[[ $scope == ct-llamamientos || $scope == ct-bolsa ]] || fallar "alcance SQL desconocido: $scope"
[[ -f $plan ]] || fallar "no existe el plan SQL: $plan"
[[ -x $runtime ]] || fallar "runtime no ejecutable: $runtime"
[[ -z $browser || -x $browser ]] || fallar "navegador no ejecutable: $browser"
[[ $probe == /* && $probe != /livez && $probe != *://* && $probe != *'?'* ]] || fallar 'la sonda debe ser una ruta HTTP local y distinta de /livez'
if [[ -n $tls_material ]]; then
  [[ $tls_material == /* ]] || fallar 'el directorio TLS debe ser absoluto'
  for certificado in ca/ca.crt mtls/cliente.crt mtls/cliente.key; do
    [[ -f $tls_material/$certificado ]] || fallar "falta material TLS local: $certificado"
  done
fi

if [[ -z $checkout ]]; then checkout=$(dirname -- "${BASH_SOURCE[0]}"); fi
[[ -d $checkout ]] || fallar "no existe el checkout: $checkout"
repo=$(git -C "$checkout" rev-parse --show-toplevel) || fallar "no es un checkout Git: $checkout"
[[ -d $repo/deploy/postgresql && -d $repo/cmd/vec-server ]] || fallar "checkout VEC incompleto: $repo"
for orden in docker git python3 curl go; do
  command -v "$orden" >/dev/null 2>&1 || fallar "falta la herramienta: $orden"
done
[[ -x $repo/scripts/seleccionar_toolchain_go_local.sh ]] || fallar 'falta selector de Go local'
[[ -d /dev/shm && -w /dev/shm ]] || fallar '/dev/shm no es escribible'

# Comprobar cobertura antes de crear la base. Un SQL ajeno, repetido o ausente
# hace fallar la puerta; no se ejecuta una cadena parcial por accidente.
python3 - "$repo" "$plan" "$scope" <<'PY'
import pathlib, subprocess, sys
repo, plan, scope = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2]), sys.argv[3]
tracked = subprocess.check_output(
    ['git', '-C', str(repo), 'ls-files', '-z', 'deploy/postgresql'],
).decode().split('\0')
available = {p for p in tracked if p.endswith('.up.sql') or
             (p.rsplit('/', 1)[-1].startswith('roles') and p.endswith('_up.sql'))}
required = {p for p in available if p.split('/')[2] == 'contratacion_temporal'
            or (scope == 'ct-bolsa' and p.split('/')[2].startswith('bolsa_'))
            or (scope == 'ct-llamamientos' and p.split('/')[2] == 'bolsa_llamamientos')}
lines = plan.read_text(encoding='utf-8').splitlines()
if any(line and line != line.strip() for line in lines):
    raise SystemExit('ERROR: el plan SQL contiene espacios alrededor de una ruta')
ordered = [line.strip() for line in lines if line.strip() and not line.lstrip().startswith('#')]
if len(ordered) != len(set(ordered)):
    raise SystemExit('ERROR: el plan SQL repite archivos')
missing, extra = required - set(ordered), set(ordered) - available
if missing or extra:
    print(f'ERROR: plan incompleto o ajeno: faltan {len(missing)}, sobran {len(extra)}', file=sys.stderr)
    for p in sorted(missing)[:8]: print(f'  falta: {p}', file=sys.stderr)
    for p in sorted(extra)[:8]: print(f'  sobra: {p}', file=sys.stderr)
    raise SystemExit(1)
if not required:
    raise SystemExit('ERROR: no se encontraron migraciones SQL del alcance')
for p in ordered:
    if not (repo / p).is_file():
        raise SystemExit(f'ERROR: fichero SQL ausente: {p}')
print(f'Plan {scope} completo: {len(required)} propios, {len(ordered) - len(required)} dependencias.', flush=True)
PY

if ! docker info >/dev/null 2>&1; then
  fallar 'Docker local inaccesible; se requiere acceso al daemon para PostgreSQL 18.4 efímero. No se contactó ningún servicio compartido.'
fi
if ! docker image inspect postgres:18.4-alpine >/dev/null 2>&1; then
  fallar 'falta la imagen local postgres:18.4-alpine; cargarla localmente antes del ensayo (esta puerta no hace pull)'
fi

ensayo=$(mktemp -d /dev/shm/vec-cadena-pg18.XXXXXXXX)
contenedor="vec-cadena-pg18-$$"
pid=''
limpiar() {
  estado=$?
  trap - EXIT INT TERM
  if [[ -n $pid ]]; then kill "$pid" 2>/dev/null || true; wait "$pid" 2>/dev/null || true; fi
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  case "$ensayo" in
    /dev/shm/vec-cadena-pg18.*)
      # PGDATA pertenece al UID de postgres dentro de Docker.
      docker run --rm --network none --user 0 \
        --mount "type=bind,src=$ensayo/data,dst=/datos" \
        --entrypoint sh postgres:18.4-alpine -c 'rm -rf /datos/pgdata' \
        >/dev/null 2>&1 || printf 'AVISO: PGDATA temporal requiere limpieza manual: %s\n' "$ensayo" >&2
      rm -rf -- "$ensayo" 2>/dev/null || true
      ;;
  esac
  exit "$estado"
}
trap limpiar EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
mkdir -m 700 "$ensayo/data"
chmod 1777 "$ensayo/data" # postgres del contenedor crea pgdata con modo 0700

# Docker publica solo loopback; la data del contenedor vive en /dev/shm y se
# elimina incluso al fallar. El puerto libre lo elige Docker y se verifica.
docker run -d --name "$contenedor" \
  --mount "type=bind,src=$ensayo/data,dst=/var/lib/postgresql/data" \
  -e PGDATA=/var/lib/postgresql/data/pgdata \
  -e POSTGRES_HOST_AUTH_METHOD=trust \
  -p 127.0.0.1::5432 postgres:18.4-alpine >/dev/null \
  || fallar 'no se pudo arrancar PostgreSQL local; revise Docker y permisos de /dev/shm'
for (( intento=0; intento<120; intento++ )); do
  if docker exec "$contenedor" pg_isready -q -U postgres -d postgres 2>/dev/null; then break; fi
  if [[ $(docker inspect -f '{{.State.Running}}' "$contenedor" 2>/dev/null) == false ]]; then
    docker logs "$contenedor" 2>&1 | tail -25 >&2
    fallar 'PostgreSQL 18.4 terminó durante el arranque'
  fi
  sleep 0.25
done
docker exec "$contenedor" pg_isready -q -U postgres -d postgres 2>/dev/null || {
  docker logs "$contenedor" 2>&1 | tail -25 >&2
  fallar 'PostgreSQL 18.4 no quedó listo'
}
version=$(docker exec "$contenedor" psql -XAt -U postgres -d postgres -c 'SHOW server_version')
[[ $version == 18.4 ]] || fallar "versión PostgreSQL inesperada: $version"
puerto=$(docker port "$contenedor" 5432/tcp | sed -n 's/^127\.0\.0\.1://p')
[[ $puerto =~ ^[0-9]+$ ]] || fallar 'Docker no publicó PostgreSQL solo en 127.0.0.1'
docker exec "$contenedor" createdb -U postgres vec_ensayo
docker exec "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d vec_ensayo \
  -c 'CREATE EXTENSION pgcrypto WITH SCHEMA public; REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC;' \
  >/dev/null || fallar 'no se pudo instalar pgcrypto en la base efímera'

num=0
while IFS= read -r archivo || [[ -n $archivo ]]; do
  [[ -z $archivo || $archivo == \#* ]] && continue
  num=$((num + 1))
  printf '[%d] %s\n' "$num" "$archivo"
  opciones=()
  if [[ $archivo == deploy/postgresql/bolsa_baremacion/migraciones/000005_manifiesto_probatorio_v3.up.sql ]]; then
    # El clúster es nuevo y no hay tráfico de aplicación durante esta fase.
    opciones=(-e PGOPTIONS=-c\ vec.confirmar_mantenimiento_bolsa_baremacion_v3=INSTALAR_MIGRACION_BOLSA_BAREMACION_V3_SIN_TRAFICO)
  fi
  if ! docker exec -i "${opciones[@]}" "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d vec_ensayo \
      < "$repo/$archivo" >"$ensayo/sql.out" 2>"$ensayo/sql.err"; then
    sed -n '1,16p' "$ensayo/sql.err" >&2
    fallar "cadena SQL detenida en [$num] $archivo"
  fi
done < "$plan"

export GOENV=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
go_local=$("$repo/scripts/seleccionar_toolchain_go_local.sh")
(cd "$repo" && "$go_local" build -buildvcs=false -o "$ensayo/vec-server" ./cmd/vec-server) \
  || fallar 'no se pudo compilar vec-server'

export PGHOST=127.0.0.1 PGPORT="$puerto" PGDATABASE=vec_ensayo
export VEC_ENSAYO_BINARIO="$ensayo/vec-server" VEC_ENSAYO_TLS_MATERIAL="$tls_material"
export VEC_ENSAYO_CONTENEDOR="$contenedor" VEC_ENSAYO_DIRECTORIO="$ensayo"
# El runtime debe generar sus LOGIN locales nominales y configurar la aplicación
# con DSN distintos, todos al PGHOST/PGPORT/PGDATABASE suministrados. No se
# acepta una variable heredada que pudiera señalar otra base.
while IFS= read -r nombre; do unset "$nombre"; done < <(compgen -e | grep -E '^(PGSERVICE|PGDATABASE_URL|VEC_.*DATABASE_URL)$' || true)
export PGHOST PGPORT PGDATABASE VEC_ENSAYO_BINARIO
"$runtime" >"$ensayo/server.log" 2>&1 & pid=$!

# El runtime escribe exclusivamente una línea URL local
# en stdout cuando termina de configurar la aplicación.
url=''
for (( intento=0; intento<120; intento++ )); do
  if ! kill -0 "$pid" 2>/dev/null; then
    sed -n '1,20p' "$ensayo/server.log" >&2
    fallar 'el runtime terminó antes de publicar URL local'
  fi
  if [[ -n $tls_material ]]; then
    url=$(sed -n 's/^URL=\(https:\/\/localhost:[0-9][0-9]*\)$/\1/p' "$ensayo/server.log" | head -1)
  else
    url=$(sed -n 's/^URL=\(http:\/\/127\.0\.0\.1:[0-9][0-9]*\)$/\1/p' "$ensayo/server.log" | head -1)
  fi
  [[ -n $url ]] && break
  sleep 0.25
done
[[ -n $url ]] || fallar 'el runtime no publicó la URL local esperada para el modo HTTP/TLS'
export VEC_ENSAYO_URL="$url"
curl_local=(curl --fail --silent --show-error --max-time 10)
if [[ -n $tls_material ]]; then
  puerto_http=${url##*:}
  curl_local+=(--resolve "localhost:$puerto_http:127.0.0.1" --cacert "$tls_material/ca/ca.crt"
    --cert "$tls_material/mtls/cliente.crt" --key "$tls_material/mtls/cliente.key")
fi
"${curl_local[@]}" "$url$probe" >"$ensayo/probe.out" \
  || { sed -n '1,20p' "$ensayo/server.log" >&2; fallar "sonda PostgreSQL HTTP fallida: $probe"; }
conexiones=$(docker exec "$contenedor" psql -XAt -U postgres -d vec_ensayo -c \
  "SELECT count(*) FROM pg_stat_activity WHERE datname = 'vec_ensayo' AND usename <> 'postgres'")
[[ $conexiones =~ ^[0-9]+$ && $conexiones -gt 0 ]] || fallar \
  'la sonda respondió, pero no hay una conexión PostgreSQL de aplicación con LOGIN nominal; no se acredita arranque sobre esta base'
printf 'OK PostgreSQL 18.4, %d SQL, binario y sonda HTTP %s\n' "$num" "$probe"
if [[ -n $browser ]]; then
  "$browser" || fallar 'recorrido navegador local fallido'
  printf 'OK recorrido navegador local\n'
else
  printf 'Navegador pendiente: repetir con --browser EJECUTABLE para completar el E2E.\n'
fi
