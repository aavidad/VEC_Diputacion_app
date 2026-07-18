#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'
umask 077

fallar() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

for orden in openssl realpath mktemp install stat find sha256sum grep mv awk cut tr chmod rm mkdir; do
  command -v "$orden" >/dev/null 2>&1 || fallar "falta la herramienta obligatoria: $orden"
done

if (( $# > 1 )); then
  fallar "uso: $0 [directorio-absoluto-fuera-del-repositorio]"
fi

DIRECTORIO_SCRIPT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
RAIZ_REPOSITORIO=$(cd -- "$DIRECTORIO_SCRIPT/.." && pwd -P)
RAIZ_ESTADO=${XDG_STATE_HOME:-${HOME:?HOME no definida}/.local/state}
DESTINO=${1:-${VEC_DEVELOPMENT_MATERIAL_DIR:-$RAIZ_ESTADO/vec-diputacion/desarrollo}}
[[ "$DESTINO" == /* ]] || fallar "el destino debe ser absoluto"
DESTINO=$(realpath -m -- "$DESTINO")

case "$DESTINO" in
  "$RAIZ_REPOSITORIO"|"$RAIZ_REPOSITORIO"/*)
    fallar "se prohibe generar material criptografico dentro del repositorio"
    ;;
esac

PADRE=$(dirname -- "$DESTINO")
mkdir -p -- "$PADRE"
PADRE=$(realpath -- "$PADRE")
[[ ! -L "$PADRE" ]] || fallar "el directorio padre no puede ser un enlace simbolico"

archivos_obligatorios=(
  ca/ca.crt ca/ca.key ca/serie
  tls/servidor.crt tls/servidor.key
  mtls/cliente.crt mtls/cliente.key
  kms/clave-maestra.bin
  kms/atestacion-ed25519.key kms/atestacion-ed25519.pub
  kms/revalidacion-ed25519.key kms/revalidacion-ed25519.pub
  tsa/clave-hmac.bin
  identidad/identidad.json manifiesto.json desarrollo.env
)

huella_clave_publica_certificado() {
  openssl x509 -in "$1" -pubkey -noout |
    openssl pkey -pubin -outform DER 2>/dev/null |
    sha256sum | awk '{print $1}'
}

huella_clave_publica_privada() {
  openssl pkey -in "$1" -pubout -outform DER 2>/dev/null |
    sha256sum | awk '{print $1}'
}

verificar_directorio() {
  local raiz=$1 archivo modo huella_ca huella_servidor huella_cliente
  local huella_publica_atestacion_kms huella_publica_revalidacion_kms
  [[ -d "$raiz" && ! -L "$raiz" ]] || fallar "directorio de credenciales no valido: $raiz"
  [[ -z "$(find "$raiz" -type l -print -quit)" ]] || fallar "hay enlaces simbolicos en el material local"
  [[ -z "$(find "$raiz" -type d -perm /077 -print -quit)" ]] || fallar "hay directorios accesibles por grupo u otros"

  for archivo in "${archivos_obligatorios[@]}"; do
    [[ -f "$raiz/$archivo" && ! -L "$raiz/$archivo" ]] || fallar "falta el fichero regular $archivo"
    modo=$(stat -c '%a' -- "$raiz/$archivo")
    (( (8#$modo & 077) == 0 )) || fallar "permisos demasiado amplios en $archivo ($modo)"
  done

  [[ $(stat -c '%s' -- "$raiz/kms/clave-maestra.bin") -eq 32 ]] || fallar "secreto KMS con longitud incorrecta"
  [[ $(stat -c '%s' -- "$raiz/tsa/clave-hmac.bin") -eq 32 ]] || fallar "secreto TSA con longitud incorrecta"
  openssl verify -purpose sslserver -CAfile "$raiz/ca/ca.crt" "$raiz/tls/servidor.crt" >/dev/null
  openssl verify -purpose sslclient -CAfile "$raiz/ca/ca.crt" "$raiz/mtls/cliente.crt" >/dev/null
  openssl x509 -in "$raiz/ca/ca.crt" -noout -checkend 0 >/dev/null
  openssl x509 -in "$raiz/tls/servidor.crt" -noout -checkend 0 >/dev/null
  openssl x509 -in "$raiz/mtls/cliente.crt" -noout -checkend 0 >/dev/null
  openssl x509 -in "$raiz/tls/servidor.crt" -noout -ext subjectAltName |
    grep -Fq 'DNS:localhost' || fallar "el certificado servidor no incluye localhost"
  [[ "$(huella_clave_publica_certificado "$raiz/ca/ca.crt")" == "$(huella_clave_publica_privada "$raiz/ca/ca.key")" ]] ||
    fallar "la clave de CA no corresponde al certificado"
  [[ "$(huella_clave_publica_certificado "$raiz/tls/servidor.crt")" == "$(huella_clave_publica_privada "$raiz/tls/servidor.key")" ]] ||
    fallar "la clave de servidor no corresponde al certificado"
  [[ "$(huella_clave_publica_certificado "$raiz/mtls/cliente.crt")" == "$(huella_clave_publica_privada "$raiz/mtls/cliente.key")" ]] ||
    fallar "la clave de cliente no corresponde al certificado"
  [[ "$(openssl pkey -in "$raiz/kms/atestacion-ed25519.key" -pubout -outform DER 2>/dev/null | sha256sum | awk '{print $1}')" == \
    "$(openssl pkey -pubin -in "$raiz/kms/atestacion-ed25519.pub" -outform DER 2>/dev/null | sha256sum | awk '{print $1}')" ]] ||
    fallar "la clave de atestacion KMS no corresponde a la publica"
  [[ "$(openssl pkey -in "$raiz/kms/revalidacion-ed25519.key" -pubout -outform DER 2>/dev/null | sha256sum | awk '{print $1}')" == \
    "$(openssl pkey -pubin -in "$raiz/kms/revalidacion-ed25519.pub" -outform DER 2>/dev/null | sha256sum | awk '{print $1}')" ]] ||
    fallar "la clave de revalidacion KMS no corresponde a la publica"
  huella_ca=$(openssl x509 -in "$raiz/ca/ca.crt" -noout -fingerprint -sha256 | cut -d= -f2 | tr -d ':')
  huella_servidor=$(openssl x509 -in "$raiz/tls/servidor.crt" -noout -fingerprint -sha256 | cut -d= -f2 | tr -d ':')
  huella_cliente=$(openssl x509 -in "$raiz/mtls/cliente.crt" -outform DER | sha256sum | awk '{print $1}')
  huella_publica_atestacion_kms=$(openssl pkey -pubin -in "$raiz/kms/atestacion-ed25519.pub" -outform DER 2>/dev/null | sha256sum | awk '{print $1}')
  huella_publica_revalidacion_kms=$(openssl pkey -pubin -in "$raiz/kms/revalidacion-ed25519.pub" -outform DER 2>/dev/null | sha256sum | awk '{print $1}')
  grep -Fq "\"huella_ca_sha256\":\"$huella_ca\"" "$raiz/manifiesto.json" || fallar "huella CA incoherente en manifiesto"
  grep -Fq "\"huella_servidor_sha256\":\"$huella_servidor\"" "$raiz/manifiesto.json" || fallar "huella servidor incoherente en manifiesto"
  grep -Fq "\"huella_cliente_sha256\":\"$huella_cliente\"" "$raiz/manifiesto.json" || fallar "huella cliente incoherente en manifiesto"
  grep -Fq "\"huella_publica_atestacion_kms_sha256\":\"$huella_publica_atestacion_kms\"" "$raiz/manifiesto.json" ||
    fallar "huella publica de atestacion KMS incoherente en manifiesto"
  grep -Fq "\"huella_publica_revalidacion_kms_sha256\":\"$huella_publica_revalidacion_kms\"" "$raiz/manifiesto.json" ||
    fallar "huella publica de revalidacion KMS incoherente en manifiesto"
  grep -Fq "\"certificate_sha256\":\"$huella_cliente\"" "$raiz/identidad/identidad.json" || fallar "identidad no ligada al certificado cliente"
  grep -Fxq 'VEC_EXECUTION_PROFILE=desarrollo' "$raiz/desarrollo.env" || fallar "perfil ausente en desarrollo.env"
  grep -Fxq 'VEC_AUTH_MODE=desarrollo' "$raiz/desarrollo.env" || fallar "modo ausente en desarrollo.env"
  grep -Fxq 'VEC_DEVELOPMENT_GUARD=ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO' "$raiz/desarrollo.env" ||
    fallar "segunda llave ausente en desarrollo.env"
}

if [[ -e "$DESTINO" || -L "$DESTINO" ]]; then
  verificar_directorio "$DESTINO"
  printf 'Credenciales de desarrollo ya existentes y verificadas: %s\n' "$DESTINO"
  exit 0
fi

TEMPORAL=$(mktemp -d -- "$PADRE/.vec-desarrollo.XXXXXX")
limpiar() {
  if [[ -n ${TEMPORAL:-} && -d ${TEMPORAL:-} ]]; then
    rm -rf -- "$TEMPORAL"
  fi
}
trap limpiar EXIT HUP INT TERM

for subdirectorio in ca tls mtls kms tsa identidad; do
  install -d -m 700 -- "$TEMPORAL/$subdirectorio"
done

openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "$TEMPORAL/ca/ca.key" 2>/dev/null
openssl req -new -x509 -sha256 -days 825 -key "$TEMPORAL/ca/ca.key" \
  -subj '/CN=VEC Desarrollo CA NO AUTORITATIVA/O=VEC Desarrollo/OU=NO AUTORITATIVO' \
  -addext 'basicConstraints=critical,CA:TRUE,pathlen:0' \
  -addext 'keyUsage=critical,keyCertSign,cRLSign' \
  -addext 'subjectKeyIdentifier=hash' \
  -out "$TEMPORAL/ca/ca.crt" 2>/dev/null
openssl rand -hex 16 >"$TEMPORAL/ca/serie"

openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "$TEMPORAL/tls/servidor.key" 2>/dev/null
openssl req -new -sha256 -key "$TEMPORAL/tls/servidor.key" \
  -subj '/CN=localhost/O=VEC Desarrollo/OU=NO AUTORITATIVO' \
  -out "$TEMPORAL/tls/servidor.csr" 2>/dev/null
cat >"$TEMPORAL/tls/servidor.ext" <<'EXT'
basicConstraints=critical,CA:FALSE
keyUsage=critical,digitalSignature
extendedKeyUsage=serverAuth
subjectAltName=DNS:localhost,IP:127.0.0.1,IP:::1
subjectKeyIdentifier=hash
authorityKeyIdentifier=keyid,issuer
EXT
openssl x509 -req -sha256 -days 397 -in "$TEMPORAL/tls/servidor.csr" \
  -CA "$TEMPORAL/ca/ca.crt" -CAkey "$TEMPORAL/ca/ca.key" -CAserial "$TEMPORAL/ca/serie" \
  -extfile "$TEMPORAL/tls/servidor.ext" -out "$TEMPORAL/tls/servidor.crt" 2>/dev/null

openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "$TEMPORAL/mtls/cliente.key" 2>/dev/null
openssl req -new -sha256 -key "$TEMPORAL/mtls/cliente.key" \
  -subj '/CN=operador-rrhh-desarrollo/O=VEC Desarrollo/OU=NO AUTORITATIVO' \
  -out "$TEMPORAL/mtls/cliente.csr" 2>/dev/null
cat >"$TEMPORAL/mtls/cliente.ext" <<'EXT'
basicConstraints=critical,CA:FALSE
keyUsage=critical,digitalSignature
extendedKeyUsage=clientAuth
subjectAltName=URI:urn:vec:desarrollo:operador-rrhh
subjectKeyIdentifier=hash
authorityKeyIdentifier=keyid,issuer
EXT
openssl x509 -req -sha256 -days 397 -in "$TEMPORAL/mtls/cliente.csr" \
  -CA "$TEMPORAL/ca/ca.crt" -CAkey "$TEMPORAL/ca/ca.key" -CAserial "$TEMPORAL/ca/serie" \
  -extfile "$TEMPORAL/mtls/cliente.ext" -out "$TEMPORAL/mtls/cliente.crt" 2>/dev/null

rm -f -- "$TEMPORAL/tls/servidor.csr" "$TEMPORAL/tls/servidor.ext" \
  "$TEMPORAL/mtls/cliente.csr" "$TEMPORAL/mtls/cliente.ext"
openssl rand 32 >"$TEMPORAL/kms/clave-maestra.bin"
openssl genpkey -algorithm ED25519 -out "$TEMPORAL/kms/atestacion-ed25519.key" 2>/dev/null
openssl pkey -in "$TEMPORAL/kms/atestacion-ed25519.key" -pubout -out "$TEMPORAL/kms/atestacion-ed25519.pub" 2>/dev/null
openssl genpkey -algorithm ED25519 -out "$TEMPORAL/kms/revalidacion-ed25519.key" 2>/dev/null
openssl pkey -in "$TEMPORAL/kms/revalidacion-ed25519.key" -pubout -out "$TEMPORAL/kms/revalidacion-ed25519.pub" 2>/dev/null
openssl rand 32 >"$TEMPORAL/tsa/clave-hmac.bin"

HUELLA_CA=$(openssl x509 -in "$TEMPORAL/ca/ca.crt" -noout -fingerprint -sha256 | cut -d= -f2 | tr -d ':')
HUELLA_SERVIDOR=$(openssl x509 -in "$TEMPORAL/tls/servidor.crt" -noout -fingerprint -sha256 | cut -d= -f2 | tr -d ':')
HUELLA_CLIENTE=$(openssl x509 -in "$TEMPORAL/mtls/cliente.crt" -outform DER | sha256sum | awk '{print $1}')
HUELLA_PUBLICA_ATESTACION_KMS=$(openssl pkey -pubin -in "$TEMPORAL/kms/atestacion-ed25519.pub" -outform DER 2>/dev/null | sha256sum | awk '{print $1}')
HUELLA_PUBLICA_REVALIDACION_KMS=$(openssl pkey -pubin -in "$TEMPORAL/kms/revalidacion-ed25519.pub" -outform DER 2>/dev/null | sha256sum | awk '{print $1}')

cat >"$TEMPORAL/identidad/identidad.json" <<JSON
{"version":1,"autoridad":"no_autoritativo","certificate_sha256":"$HUELLA_CLIENTE","subject":"desarrollo:operador-rrhh","display_name":"Operador RRHH de desarrollo","roles":["tecnico_rrhh"]}
JSON
cat >"$TEMPORAL/manifiesto.json" <<JSON
{"version":2,"perfil":"desarrollo","autoridad":"no_autoritativo","migrable_a_produccion":false,"huella_ca_sha256":"$HUELLA_CA","huella_servidor_sha256":"$HUELLA_SERVIDOR","huella_cliente_sha256":"$HUELLA_CLIENTE","huella_publica_atestacion_kms_sha256":"$HUELLA_PUBLICA_ATESTACION_KMS","huella_publica_revalidacion_kms_sha256":"$HUELLA_PUBLICA_REVALIDACION_KMS","proveedores":{"identidad":"identidad-mtls-local-v1","kms_emisor":"kms-emisor-fichero-local-v2","kms_revalidador":"kms-revalidador-ed25519-local-v1","kms_verificador_recibo":"kms-verificador-publico-local-v1","tsa":"tsa-determinista-local-v1","tls":"tls-ca-local-v1"}}
JSON
{
  printf 'VEC_EXECUTION_PROFILE=desarrollo\n'
  printf 'VEC_AUTH_MODE=desarrollo\n'
  printf 'VEC_DEVELOPMENT_GUARD=ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO\n'
  printf 'VEC_DEVELOPMENT_MATERIAL_DIR=%q\n' "$DESTINO"
  printf 'VEC_TLS_CERT_FILE=%q\n' "$DESTINO/tls/servidor.crt"
  printf 'VEC_TLS_KEY_FILE=%q\n' "$DESTINO/tls/servidor.key"
} >"$TEMPORAL/desarrollo.env"

find "$TEMPORAL" -type d -exec chmod 700 {} +
find "$TEMPORAL" -type f -exec chmod 600 {} +
verificar_directorio "$TEMPORAL"
mv -- "$TEMPORAL" "$DESTINO"
TEMPORAL=''

printf 'Credenciales de desarrollo generadas fuera de Git: %s\n' "$DESTINO"
printf 'CA SHA-256: %s\n' "$HUELLA_CA"
printf 'Cliente mTLS SHA-256: %s\n' "$HUELLA_CLIENTE"
printf 'Cargue la configuracion solo en una terminal local: set -a; source %q; set +a\n' "$DESTINO/desarrollo.env"
