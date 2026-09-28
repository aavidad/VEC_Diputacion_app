#!/usr/bin/env bash
# Runtime exclusivo del runner PG18 local. Solo usa el clúster y material efímero
# que recibe de scripts/ensayar_cadena_sql_pg18_local.sh.
# WIP APARCADO: membresías de gobierno y sonda HTTP pendientes de revisión final;
# no acredita arranque real de CT ni debe usarse como puerta de entrega.
set -Eeuo pipefail
IFS=$'\n\t'
umask 077

fallar() { printf 'ERROR runtime CT PG18: %s\n' "$*" >&2; exit 1; }

if (( $# == 1 )) && [[ $1 == --help ]]; then
  cat <<'AYUDA'
Runtime para --runtime de ensayar_cadena_sql_pg18_local.sh.
Estado: WIP aparcado, sin GO de arranque binario ni consulta PostgreSQL real.
Requiere --tls-material auto en el runner y sus variables VEC_ENSAYO_*.
Crea doce LOGIN nominales sin contraseña solo en vec_ensayo, genera material
de desarrollo sintético en /dev/shm y ejecuta VEC_ENSAYO_BINARIO en loopback.
Publica URL=https://localhost:PUERTO al completar /livez con mTLS.
AYUDA
  exit 0
fi
if (( $# != 0 )); then fallar 'no admite argumentos'; fi
for orden in docker git python3 curl openssl; do
  command -v "$orden" >/dev/null 2>&1 || fallar "falta la herramienta local: $orden"
done

contenedor=${VEC_ENSAYO_CONTENEDOR:-}
directorio=${VEC_ENSAYO_DIRECTORIO:-}
binario=${VEC_ENSAYO_BINARIO:-}
material=${VEC_ENSAYO_TLS_MATERIAL:-}
raiz=${VEC_ENSAYO_REPO:-}
[[ $contenedor =~ ^vec-cadena-pg18-[0-9]+$ ]] || fallar 'contenedor ajeno al runner'
[[ $directorio == /dev/shm/vec-cadena-pg18.* && -d $directorio && ! -L $directorio ]] || fallar 'directorio efímero ajeno al runner'
[[ $binario == "$directorio/vec-server" && -x $binario && ! -L $binario ]] || fallar 'binario ajeno al runner'
[[ $material == "$directorio/material" ]] || fallar 'el material TLS debe generarse en el ensayo'
[[ $raiz == /* && -x $raiz/scripts/generar_credenciales_desarrollo.sh ]] || fallar 'checkout VEC del ensayo ausente'
[[ $(git -C "$raiz" rev-parse --show-toplevel 2>/dev/null || true) == "$raiz" ]] || fallar 'checkout VEC no canónico'
[[ ${PGHOST:-} == 127.0.0.1 && ${PGDATABASE:-} == vec_ensayo ]] || fallar 'PostgreSQL debe ser la base local vec_ensayo'
[[ ${PGPORT:-} =~ ^[0-9]{1,5}$ ]] || fallar 'puerto PostgreSQL inválido'
(( PGPORT >= 1 && PGPORT <= 65535 )) || fallar 'puerto PostgreSQL fuera de rango'
[[ -S /var/run/docker.sock && ${DOCKER_HOST:-} == unix:///var/run/docker.sock ]] || fallar 'Docker debe ser local'
[[ -z ${DOCKER_CONTEXT:-} ]] || fallar 'contexto Docker externo no permitido'

generador="$raiz/scripts/generar_credenciales_desarrollo.sh"
[[ -x $generador ]] || fallar 'generador de material de desarrollo ausente'

puerto_publicado=$(docker port "$contenedor" 5432/tcp 2>/dev/null || true)
[[ $puerto_publicado == "127.0.0.1:$PGPORT" ]] || fallar 'contenedor PostgreSQL no publica el puerto local esperado'
inspeccion=$(docker inspect -f '{{.State.Running}}|{{range .Mounts}}{{.Source}}:{{.Destination}};{{end}}' "$contenedor" 2>/dev/null || true)
[[ $inspeccion == true\|* && $inspeccion == *"$directorio/data:/var/lib/postgresql/data;"* ]] || fallar 'contenedor PostgreSQL sin PGDATA efímero esperado'
version=$(docker exec "$contenedor" psql -XAt -U postgres -d vec_ensayo -c 'SHOW server_version' 2>/dev/null || true)
[[ $version == 18.4 ]] || fallar 'la base efímera no usa PostgreSQL 18.4'

# Una sola arista técnica por LOGIN. El grupo de gobierno debe poder hacer
# SET ROLE al propietario; los demás heredan únicamente su grupo nominal.
variables=(
  VEC_CT_DATABASE_URL
  VEC_CT_GOBIERNO_DATABASE_URL
  VEC_CT_REGISTRO_AUTORIZACION_DATABASE_URL
  VEC_CT_CONFIRMADOR_DATABASE_URL
  VEC_CT_LECTOR_RESULTADO_DATABASE_URL
  VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL
  VEC_CT_CONSULTAS_RRHH_DATABASE_URL
  VEC_CT_MOTIVOS_RRHH_DATABASE_URL
  VEC_CT_REGISTRO_IDENTIDAD_DATABASE_URL
  VEC_CT_REVALIDACION_IDENTIDAD_DATABASE_URL
  VEC_CT_CONTEXTO_ACTOR_DATABASE_URL
  VEC_CT_AUDITORIA_FRONTERA_DATABASE_URL
)
grupos=(
  vec_contratacion_temporal_ejecutor
  vec_autorizacion_atestada_v3_migrador
  vec_autorizacion_registro
  vec_contratacion_temporal_confirmador_cobertura
  vec_contratacion_temporal_lector_resultado_cobertura
  vec_bolsa_llamamientos_ejecutor
  vec_contratacion_temporal_consultor_rrhh
  vec_autorizacion_motivos_evaluador
  vec_identidad_sesiones_v1_registrador
  vec_identidad_sesiones_v1_revalidador
  vec_contexto_actor_v1_runtime
  vec_contratacion_temporal_registrador_frontera
)
roles_gobierno=(
  vec_contexto_actor_v1_propietario
  vec_autorizacion_propietario
  vec_autorizacion_motivos_proyector
  vec_identidad_sesiones_v1_propietario
)
(( ${#variables[@]} == ${#grupos[@]} )) || fallar 'mapa técnico incoherente'

sql="$directorio/roles_runtime_ct.sql"
printf 'BEGIN;\n' > "$sql"
entorno=(
  "PATH=$PATH"
  "HOME=$directorio"
  "TMPDIR=$directorio"
  "VEC_EXECUTION_PROFILE=desarrollo"
  "VEC_AUTH_MODE=desarrollo"
  "VEC_DEVELOPMENT_GUARD=ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO"
  "VEC_DEVELOPMENT_MATERIAL_DIR=$material"
  "VEC_TLS_CERT_FILE=$material/tls/servidor.crt"
  "VEC_TLS_KEY_FILE=$material/tls/servidor.key"
)
for i in "${!variables[@]}"; do
  usuario=$(printf 'vec_ensayo_ct_%02d' "$i")
  grupo=${grupos[i]}
  if [[ $i == 1 ]]; then
    opciones='INHERIT FALSE, SET TRUE'
  else
    opciones='INHERIT TRUE, SET FALSE'
  fi
  {
    printf 'CREATE ROLE %s LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;\n' "$usuario"
    printf 'GRANT %s TO %s WITH ADMIN FALSE, %s;\n' "$grupo" "$usuario" "$opciones"
    printf 'GRANT CONNECT ON DATABASE vec_ensayo TO %s;\n' "$usuario"
    if [[ $i == 1 ]]; then
      for rol in "${roles_gobierno[@]}"; do
        printf 'GRANT %s TO %s WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;\n' "$rol" "$usuario"
      done
    fi
  } >> "$sql"
  entorno+=("${variables[i]}=postgresql://$usuario@127.0.0.1:$PGPORT/vec_ensayo?sslmode=disable")
done
printf 'COMMIT;\n' >> "$sql"
docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d vec_ensayo < "$sql" >/dev/null \
  || fallar 'no se pudieron crear los LOGIN nominales locales; revise roles del plan SQL'

"$generador" "$material" > "$directorio/material-generacion.log" 2>&1 \
  || fallar 'no se pudo generar material TLS/mTLS sintético en /dev/shm'

# Selección de puerto local: la comprobación posterior del runner fija
# localhost a 127.0.0.1 y no acepta una URL aportada por el entorno.
puerto_http=$(python3 - <<'PY'
import socket
with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
    sock.bind(('127.0.0.1', 0))
    print(sock.getsockname()[1])
PY
) || fallar 'no se pudo reservar un puerto HTTP loopback'
[[ $puerto_http =~ ^[0-9]{1,5}$ ]] || fallar 'puerto HTTP inválido'
entorno+=("VEC_HTTP_ADDR=127.0.0.1:$puerto_http")

servidor_pid=''
# Invocada indirectamente por trap.
# shellcheck disable=SC2329
limpiar() {
  if [[ -n $servidor_pid ]] && kill -0 "$servidor_pid" 2>/dev/null; then
    kill -TERM "$servidor_pid" 2>/dev/null || true
    wait "$servidor_pid" 2>/dev/null || true
  fi
}
trap limpiar EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
(
  cd "$raiz"
  exec env -i "${entorno[@]}" "$binario"
) > "$directorio/vec-server.log" 2>&1 &
servidor_pid=$!
url="https://localhost:$puerto_http"
for (( intento=0; intento<200; intento++ )); do
  if ! kill -0 "$servidor_pid" 2>/dev/null; then
    fallar 'vec-server terminó antes de responder en loopback'
  fi
  if curl --fail --silent --show-error --max-time 2 \
      --resolve "localhost:$puerto_http:127.0.0.1" \
      --cacert "$material/ca/ca.crt" --cert "$material/mtls/cliente.crt" \
      --key "$material/mtls/cliente.key" "$url/livez" >/dev/null 2>&1; then
    printf 'URL=%s\n' "$url"
    wait "$servidor_pid"
    exit $?
  fi
  sleep 0.1
done
fallar 'vec-server no respondió con TLS/mTLS local en el plazo de arranque'
