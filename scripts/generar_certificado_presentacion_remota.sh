#!/usr/bin/env bash
# Genera un certificado TLS autofirmado para la entrada remota de la
# presentación (entrada-remota-presentacion, puerto convencional 18081).
# El material nunca se escribe dentro del repositorio: el repositorio es
# público y ni una clave ni un certificado pueden entrar en Git (ver
# .gitignore y scripts/generar_credenciales_desarrollo.sh, mismo criterio).
set -Eeuo pipefail
IFS=$'\n\t'
umask 077

fallar() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

for orden in openssl realpath mkdir chmod mktemp rm dirname mv cut; do
  command -v "$orden" >/dev/null 2>&1 || fallar "falta la herramienta obligatoria: $orden"
done

if (( $# > 2 )); then
  fallar "uso: $0 [directorio-absoluto-fuera-del-repositorio] [IP-o-nombre-adicional]"
fi

DIRECTORIO_SCRIPT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
RAIZ_REPOSITORIO=$(cd -- "$DIRECTORIO_SCRIPT/.." && pwd -P)
VAR_LOCAL_IGNORADA="$RAIZ_REPOSITORIO/var/tls-presentacion-remota"
DESTINO=${1:-${VEC_PRESENTACION_REMOTE_TLS_DIR:-$VAR_LOCAL_IGNORADA}}
SAN_ADICIONAL=${2:-${VEC_PRESENTACION_REMOTE_BIND_ADDRESS:-}}
[[ "$DESTINO" == /* ]] || fallar "el destino debe ser absoluto"
DESTINO=$(realpath -m -- "$DESTINO")

# El único destino admitido dentro del árbol del repositorio es var/, que
# .gitignore excluye por completo (igual que var/revision-web-*). Cualquier
# otra ruta dentro del repositorio se rechaza; para un despliegue real se pasa
# una ruta absoluta fuera del repositorio, custodiada por Sistemas.
case "$DESTINO" in
  "$VAR_LOCAL_IGNORADA"|"$VAR_LOCAL_IGNORADA"/*) ;;
  "$RAIZ_REPOSITORIO"|"$RAIZ_REPOSITORIO"/*)
    fallar "solo se admite var/tls-presentacion-remota dentro del repositorio; usa una ruta absoluta fuera de él para un despliegue real"
    ;;
esac

PADRE=$(dirname -- "$DESTINO")
mkdir -p -- "$PADRE"
PADRE=$(realpath -- "$PADRE")
[[ ! -L "$PADRE" ]] || fallar "el directorio padre no puede ser un enlace simbolico"

if [[ -e "$DESTINO/servidor.crt" || -e "$DESTINO/servidor.key" ]]; then
  [[ -f "$DESTINO/servidor.crt" && -f "$DESTINO/servidor.key" ]] ||
    fallar "el material existente debe contener dos ficheros regulares; no se sustituye automáticamente"
  openssl x509 -in "$DESTINO/servidor.crt" -noout -checkend 0 >/dev/null ||
    fallar "el certificado existente no es válido o ha caducado; archiva $DESTINO y repite"
  openssl pkey -in "$DESTINO/servidor.key" -check -noout >/dev/null 2>&1 ||
    fallar "la clave privada existente no es válida"
  HUELLA_CERT=$(openssl x509 -in "$DESTINO/servidor.crt" -pubkey -noout |
    openssl pkey -pubin -outform DER 2>/dev/null |
    openssl dgst -sha256)
  HUELLA_CLAVE=$(openssl pkey -in "$DESTINO/servidor.key" -pubout -outform DER 2>/dev/null |
    openssl dgst -sha256)
  [[ "$HUELLA_CERT" == "$HUELLA_CLAVE" ]] ||
    fallar "el certificado y la clave existentes no forman pareja"
  chmod 0644 "$DESTINO/servidor.crt"
  chmod 0640 "$DESTINO/servidor.key"
  printf 'Certificado de presentación remota ya existente y vigente: %s\n' "$DESTINO"
  exit 0
fi

TEMPORAL=$(mktemp -d -- "$PADRE/.vec-presentacion-remota-tls.XXXXXX")
limpiar() { [[ -n ${TEMPORAL:-} && -d ${TEMPORAL:-} ]] && rm -rf -- "$TEMPORAL"; }
trap limpiar EXIT HUP INT TERM

SAN="DNS:localhost,IP:127.0.0.1,IP:::1"
if [[ -n "$SAN_ADICIONAL" ]]; then
  if [[ "$SAN_ADICIONAL" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    SAN="$SAN,IP:$SAN_ADICIONAL"
  else
    SAN="$SAN,DNS:$SAN_ADICIONAL"
  fi
fi

openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 \
  -out "$TEMPORAL/servidor.key" 2>/dev/null
openssl req -new -x509 -sha256 -days 397 -key "$TEMPORAL/servidor.key" \
  -subj '/CN=VEC Presentacion Remota NO AUTORITATIVA/O=VEC Demo/OU=SOLO PRESENTACION' \
  -addext 'basicConstraints=critical,CA:FALSE' \
  -addext 'keyUsage=critical,digitalSignature,keyEncipherment' \
  -addext 'extendedKeyUsage=serverAuth' \
  -addext "subjectAltName=$SAN" \
  -out "$TEMPORAL/servidor.crt" 2>/dev/null

openssl x509 -in "$TEMPORAL/servidor.crt" -noout -checkend 0 >/dev/null
chmod 0644 "$TEMPORAL/servidor.crt"
chmod 0640 "$TEMPORAL/servidor.key"
mv -- "$TEMPORAL" "$DESTINO"
TEMPORAL=''

HUELLA=$(openssl x509 -in "$DESTINO/servidor.crt" -noout -fingerprint -sha256 | cut -d= -f2)
printf 'Certificado autofirmado generado fuera de Git: %s\n' "$DESTINO"
printf 'Huella SHA-256: %s\n' "$HUELLA"
printf 'Es autofirmado y NO AUTORITATIVO: el proxy corporativo debe confiar en él\n'
printf 'explícitamente (fingerprint pinning o CA propia), nunca validarlo como público.\n'
printf 'Variables para docker-compose (perfil presentacion-remota):\n'
printf '  VEC_PRESENTACION_REMOTE_TLS_CERT_PATH=%q\n' "$DESTINO/servidor.crt"
printf '  VEC_PRESENTACION_REMOTE_TLS_KEY_PATH=%q\n' "$DESTINO/servidor.key"
