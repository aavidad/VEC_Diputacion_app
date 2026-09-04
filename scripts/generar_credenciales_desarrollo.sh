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
  mtls/cliente.p12 mtls/cliente.p12.password
  mtls/intervencion.crt mtls/intervencion.key
  mtls/intervencion.p12 mtls/intervencion.p12.password
  kms/clave-maestra.bin
  kms/atestacion-ed25519.key kms/atestacion-ed25519.pub
  kms/revalidacion-ed25519.key kms/revalidacion-ed25519.pub
  tsa/clave-hmac.bin
  idempotencia/configuracion.json
  idempotencia/g2-localizador.bin idempotencia/g2-huella-solicitud.bin
  idempotencia/g1-localizador.bin idempotencia/g1-huella-solicitud.bin
  identidad/identidad.json identidad/intervencion.json manifiesto.json desarrollo.env
)
CONFIGURACION_IDEMPOTENCIA_CANONICA='{"version":1,"esquema":"vec.bolsa.convocatoria.idempotencia-hmac.desarrollo.v1","autoridad":"no_autoritativo","version_esquema_hmac":2,"generaciones":[{"generacion":2,"referencia_localizador":"clave:hmac:convocatorias:localizador:desarrollo:v2","referencia_huella_solicitud":"clave:hmac:convocatorias:huella:desarrollo:v2"},{"generacion":1,"referencia_localizador":"clave:hmac:convocatorias:localizador:desarrollo:v1","referencia_huella_solicitud":"clave:hmac:convocatorias:huella:desarrollo:v1"}]}'

huella_clave_publica_certificado() {
  openssl x509 -in "$1" -pubkey -noout |
    openssl pkey -pubin -outform DER 2>/dev/null |
    sha256sum | awk '{print $1}'
}

huella_clave_publica_privada() {
  openssl pkey -in "$1" -pubout -outform DER 2>/dev/null |
    sha256sum | awk '{print $1}'
}

huella_certificado_cliente_pkcs12() {
  openssl pkcs12 -in "$1" -passin "file:$2" -clcerts -nokeys 2>/dev/null |
    openssl x509 -outform DER 2>/dev/null |
    sha256sum | awk '{print $1}'
}

huella_clave_publica_pkcs12() {
  openssl pkcs12 -in "$1" -passin "file:$2" -nocerts -nodes 2>/dev/null |
    openssl pkey -pubout -outform DER 2>/dev/null |
    sha256sum | awk '{print $1}'
}

huella_ca_pkcs12() {
  openssl pkcs12 -in "$1" -passin "file:$2" -cacerts -nokeys 2>/dev/null |
    openssl x509 -outform DER 2>/dev/null |
    sha256sum | awk '{print $1}'
}

mostrar_instrucciones_navegador() {
  local raiz=$1
  printf 'Acceso desde navegador con mTLS:\n'
  printf '  1. Importe %s como autoridad certificadora local de confianza.\n' "$raiz/ca/ca.crt"
  printf '  2. Importe %s como certificado personal.\n' "$raiz/mtls/cliente.p12"
  printf '  3. Use en el dialogo local la contrasena guardada en %s; no la copie a registros.\n' \
    "$raiz/mtls/cliente.p12.password"
  printf '  4. Para fiscalizar, importe %s como identidad separada de Intervencion.\n' \
    "$raiz/mtls/intervencion.p12"
  printf '  5. Su contrasena independiente esta en %s; no la copie a registros.\n' \
    "$raiz/mtls/intervencion.p12.password"
  printf '  6. Abra https://localhost:<puerto>/portal-empleado/ con VEC en perfil desarrollo.\n'
}

verificar_directorio() {
  local raiz=$1 archivo modo huella_ca huella_ca_der huella_ca_p12
  local huella_servidor huella_cliente huella_cliente_p12 huella_intervencion huella_intervencion_p12
  local huella_clave_cliente huella_clave_cliente_p12 huella_clave_intervencion huella_clave_intervencion_p12
  local numero_certificados
  local huella_publica_atestacion_kms huella_publica_revalidacion_kms
  local -a secretos_hmac
  local indice anterior huella
  [[ -d "$raiz" && ! -L "$raiz" ]] || fallar "directorio de credenciales no valido: $raiz"
  [[ -z "$(find "$raiz" -type l -print -quit)" ]] || fallar "hay enlaces simbolicos en el material local"
  [[ -z "$(find "$raiz" -type d -perm /077 -print -quit)" ]] || fallar "hay directorios accesibles por grupo u otros"

  for archivo in "${archivos_obligatorios[@]}"; do
    [[ -f "$raiz/$archivo" && ! -L "$raiz/$archivo" ]] || fallar "falta el fichero regular $archivo"
    modo=$(stat -c '%a' -- "$raiz/$archivo")
    (( (8#$modo & 077) == 0 )) || fallar "permisos demasiado amplios en $archivo ($modo)"
  done

  [[ $(stat -c '%a' -- "$raiz/mtls/cliente.p12") == 600 ]] ||
    fallar "el paquete PKCS#12 debe tener permisos 0600"
  [[ $(stat -c '%a' -- "$raiz/mtls/cliente.p12.password") == 600 ]] ||
    fallar "la contrasena del paquete PKCS#12 debe tener permisos 0600"
  if [[ $(awk 'END {print NR}' "$raiz/mtls/cliente.p12.password") -ne 1 ]] ||
    ! grep -Eq '^[0-9a-f]{64}$' "$raiz/mtls/cliente.p12.password"; then
    fallar "la contrasena del paquete PKCS#12 no tiene el formato aleatorio esperado"
  fi
  openssl pkcs12 -in "$raiz/mtls/cliente.p12" \
    -passin "file:$raiz/mtls/cliente.p12.password" -noout 2>/dev/null ||
    fallar "el paquete PKCS#12 no supera su comprobacion de integridad"
  numero_certificados=$(openssl pkcs12 -in "$raiz/mtls/cliente.p12" \
    -passin "file:$raiz/mtls/cliente.p12.password" -clcerts -nokeys 2>/dev/null |
    grep -c -- '-----BEGIN CERTIFICATE-----' || true)
  [[ "$numero_certificados" -eq 1 ]] || fallar "el paquete PKCS#12 no contiene una unica identidad cliente"
  numero_certificados=$(openssl pkcs12 -in "$raiz/mtls/cliente.p12" \
    -passin "file:$raiz/mtls/cliente.p12.password" -cacerts -nokeys 2>/dev/null |
    grep -c -- '-----BEGIN CERTIFICATE-----' || true)
  [[ "$numero_certificados" -eq 1 ]] || fallar "el paquete PKCS#12 no contiene la cadena local esperada"
  [[ $(stat -c '%a' -- "$raiz/mtls/intervencion.p12") == 600 ]] ||
    fallar "el paquete PKCS#12 de Intervencion debe tener permisos 0600"
  [[ $(stat -c '%a' -- "$raiz/mtls/intervencion.p12.password") == 600 ]] ||
    fallar "la contrasena PKCS#12 de Intervencion debe tener permisos 0600"
  if [[ $(awk 'END {print NR}' "$raiz/mtls/intervencion.p12.password") -ne 1 ]] ||
    ! grep -Eq '^[0-9a-f]{64}$' "$raiz/mtls/intervencion.p12.password"; then
    fallar "la contrasena PKCS#12 de Intervencion no tiene el formato aleatorio esperado"
  fi
  openssl pkcs12 -in "$raiz/mtls/intervencion.p12" \
    -passin "file:$raiz/mtls/intervencion.p12.password" -noout 2>/dev/null ||
    fallar "el paquete PKCS#12 de Intervencion no supera su comprobacion de integridad"
  numero_certificados=$(openssl pkcs12 -in "$raiz/mtls/intervencion.p12" \
    -passin "file:$raiz/mtls/intervencion.p12.password" -clcerts -nokeys 2>/dev/null |
    grep -c -- '-----BEGIN CERTIFICATE-----' || true)
  [[ "$numero_certificados" -eq 1 ]] ||
    fallar "el paquete PKCS#12 de Intervencion no contiene una unica identidad cliente"
  numero_certificados=$(openssl pkcs12 -in "$raiz/mtls/intervencion.p12" \
    -passin "file:$raiz/mtls/intervencion.p12.password" -cacerts -nokeys 2>/dev/null |
    grep -c -- '-----BEGIN CERTIFICATE-----' || true)
  [[ "$numero_certificados" -eq 1 ]] ||
    fallar "el paquete PKCS#12 de Intervencion no contiene la cadena local esperada"

  [[ $(stat -c '%s' -- "$raiz/kms/clave-maestra.bin") -eq 32 ]] || fallar "secreto KMS con longitud incorrecta"
  [[ $(stat -c '%s' -- "$raiz/tsa/clave-hmac.bin") -eq 32 ]] || fallar "secreto TSA con longitud incorrecta"
  secretos_hmac=(
    "$raiz/kms/clave-maestra.bin"
    "$raiz/tsa/clave-hmac.bin"
    "$raiz/idempotencia/g2-localizador.bin"
    "$raiz/idempotencia/g2-huella-solicitud.bin"
    "$raiz/idempotencia/g1-localizador.bin"
    "$raiz/idempotencia/g1-huella-solicitud.bin"
  )
  for indice in "${!secretos_hmac[@]}"; do
    [[ $(stat -c '%s' -- "${secretos_hmac[$indice]}") -eq 32 ]] ||
      fallar "secreto HMAC con longitud incorrecta: ${secretos_hmac[$indice]}"
    huella=$(sha256sum -- "${secretos_hmac[$indice]}" | awk '{print $1}')
    for (( anterior = 0; anterior < indice; anterior++ )); do
      [[ "$huella" != "$(sha256sum -- "${secretos_hmac[$anterior]}" | awk '{print $1}')" ]] ||
        fallar "se reutilizo material entre dominios criptograficos"
    done
  done
  openssl verify -purpose sslserver -CAfile "$raiz/ca/ca.crt" "$raiz/tls/servidor.crt" >/dev/null
  openssl verify -purpose sslclient -CAfile "$raiz/ca/ca.crt" "$raiz/mtls/cliente.crt" >/dev/null
  openssl verify -purpose sslclient -CAfile "$raiz/ca/ca.crt" "$raiz/mtls/intervencion.crt" >/dev/null
  openssl x509 -in "$raiz/ca/ca.crt" -noout -checkend 0 >/dev/null
  openssl x509 -in "$raiz/tls/servidor.crt" -noout -checkend 0 >/dev/null
  openssl x509 -in "$raiz/mtls/cliente.crt" -noout -checkend 0 >/dev/null
  openssl x509 -in "$raiz/mtls/intervencion.crt" -noout -checkend 0 >/dev/null
  openssl x509 -in "$raiz/tls/servidor.crt" -noout -ext subjectAltName |
    grep -Fq 'DNS:localhost' || fallar "el certificado servidor no incluye localhost"
  [[ "$(huella_clave_publica_certificado "$raiz/ca/ca.crt")" == "$(huella_clave_publica_privada "$raiz/ca/ca.key")" ]] ||
    fallar "la clave de CA no corresponde al certificado"
  [[ "$(huella_clave_publica_certificado "$raiz/tls/servidor.crt")" == "$(huella_clave_publica_privada "$raiz/tls/servidor.key")" ]] ||
    fallar "la clave de servidor no corresponde al certificado"
  [[ "$(huella_clave_publica_certificado "$raiz/mtls/cliente.crt")" == "$(huella_clave_publica_privada "$raiz/mtls/cliente.key")" ]] ||
    fallar "la clave de cliente no corresponde al certificado"
  [[ "$(huella_clave_publica_certificado "$raiz/mtls/intervencion.crt")" == "$(huella_clave_publica_privada "$raiz/mtls/intervencion.key")" ]] ||
    fallar "la clave de Intervencion no corresponde al certificado"
  huella_cliente=$(openssl x509 -in "$raiz/mtls/cliente.crt" -outform DER | sha256sum | awk '{print $1}')
  huella_cliente_p12=$(huella_certificado_cliente_pkcs12 \
    "$raiz/mtls/cliente.p12" "$raiz/mtls/cliente.p12.password")
  [[ "$huella_cliente_p12" == "$huella_cliente" ]] ||
    fallar "la identidad del paquete PKCS#12 no corresponde al certificado cliente"
  huella_clave_cliente=$(huella_clave_publica_privada "$raiz/mtls/cliente.key")
  huella_clave_cliente_p12=$(huella_clave_publica_pkcs12 \
    "$raiz/mtls/cliente.p12" "$raiz/mtls/cliente.p12.password")
  [[ "$huella_clave_cliente_p12" == "$huella_clave_cliente" ]] ||
    fallar "la clave del paquete PKCS#12 no corresponde a la identidad cliente"
  huella_ca_der=$(openssl x509 -in "$raiz/ca/ca.crt" -outform DER | sha256sum | awk '{print $1}')
  huella_ca_p12=$(huella_ca_pkcs12 "$raiz/mtls/cliente.p12" "$raiz/mtls/cliente.p12.password")
  [[ "$huella_ca_p12" == "$huella_ca_der" ]] ||
    fallar "la cadena del paquete PKCS#12 no corresponde a la CA local"
  huella_intervencion=$(openssl x509 -in "$raiz/mtls/intervencion.crt" -outform DER | sha256sum | awk '{print $1}')
  huella_intervencion_p12=$(huella_certificado_cliente_pkcs12 \
    "$raiz/mtls/intervencion.p12" "$raiz/mtls/intervencion.p12.password")
  [[ "$huella_intervencion_p12" == "$huella_intervencion" ]] ||
    fallar "la identidad PKCS#12 de Intervencion no corresponde a su certificado"
  huella_clave_intervencion=$(huella_clave_publica_privada "$raiz/mtls/intervencion.key")
  huella_clave_intervencion_p12=$(huella_clave_publica_pkcs12 \
    "$raiz/mtls/intervencion.p12" "$raiz/mtls/intervencion.p12.password")
  [[ "$huella_clave_intervencion_p12" == "$huella_clave_intervencion" ]] ||
    fallar "la clave PKCS#12 de Intervencion no corresponde a su identidad"
  huella_ca_p12=$(huella_ca_pkcs12 \
    "$raiz/mtls/intervencion.p12" "$raiz/mtls/intervencion.p12.password")
  [[ "$huella_ca_p12" == "$huella_ca_der" ]] ||
    fallar "la cadena PKCS#12 de Intervencion no corresponde a la CA local"
  [[ "$huella_intervencion" != "$huella_cliente" ]] ||
    fallar "RRHH e Intervencion no pueden compartir certificado"
  [[ "$huella_clave_intervencion" != "$huella_clave_cliente" ]] ||
    fallar "RRHH e Intervencion no pueden compartir clave"
  [[ "$(openssl pkey -in "$raiz/kms/atestacion-ed25519.key" -pubout -outform DER 2>/dev/null | sha256sum | awk '{print $1}')" == \
    "$(openssl pkey -pubin -in "$raiz/kms/atestacion-ed25519.pub" -outform DER 2>/dev/null | sha256sum | awk '{print $1}')" ]] ||
    fallar "la clave de atestacion KMS no corresponde a la publica"
  [[ "$(openssl pkey -in "$raiz/kms/revalidacion-ed25519.key" -pubout -outform DER 2>/dev/null | sha256sum | awk '{print $1}')" == \
    "$(openssl pkey -pubin -in "$raiz/kms/revalidacion-ed25519.pub" -outform DER 2>/dev/null | sha256sum | awk '{print $1}')" ]] ||
    fallar "la clave de revalidacion KMS no corresponde a la publica"
  huella_ca=$(openssl x509 -in "$raiz/ca/ca.crt" -noout -fingerprint -sha256 | cut -d= -f2 | tr -d ':')
  huella_servidor=$(openssl x509 -in "$raiz/tls/servidor.crt" -noout -fingerprint -sha256 | cut -d= -f2 | tr -d ':')
  huella_publica_atestacion_kms=$(openssl pkey -pubin -in "$raiz/kms/atestacion-ed25519.pub" -outform DER 2>/dev/null | sha256sum | awk '{print $1}')
  huella_publica_revalidacion_kms=$(openssl pkey -pubin -in "$raiz/kms/revalidacion-ed25519.pub" -outform DER 2>/dev/null | sha256sum | awk '{print $1}')
  grep -Fq "\"huella_ca_sha256\":\"$huella_ca\"" "$raiz/manifiesto.json" || fallar "huella CA incoherente en manifiesto"
  grep -Fq "\"huella_servidor_sha256\":\"$huella_servidor\"" "$raiz/manifiesto.json" || fallar "huella servidor incoherente en manifiesto"
  grep -Fq "\"huella_cliente_sha256\":\"$huella_cliente\"" "$raiz/manifiesto.json" || fallar "huella cliente incoherente en manifiesto"
  grep -Fq "\"huella_intervencion_sha256\":\"$huella_intervencion\"" "$raiz/manifiesto.json" ||
    fallar "huella de Intervencion incoherente en manifiesto"
  grep -Fq "\"huella_publica_atestacion_kms_sha256\":\"$huella_publica_atestacion_kms\"" "$raiz/manifiesto.json" ||
    fallar "huella publica de atestacion KMS incoherente en manifiesto"
  grep -Fq "\"huella_publica_revalidacion_kms_sha256\":\"$huella_publica_revalidacion_kms\"" "$raiz/manifiesto.json" ||
    fallar "huella publica de revalidacion KMS incoherente en manifiesto"
  grep -Fq "\"certificate_sha256\":\"$huella_cliente\"" "$raiz/identidad/identidad.json" || fallar "identidad no ligada al certificado cliente"
  grep -Fq "\"certificate_sha256\":\"$huella_intervencion\"" "$raiz/identidad/intervencion.json" ||
    fallar "identidad de Intervencion no ligada a su certificado"
  grep -Fq '"esquema":"vec.bolsa.convocatoria.idempotencia-hmac.desarrollo.v1"' \
    "$raiz/idempotencia/configuracion.json" || fallar "esquema HMAC de idempotencia ausente"
  grep -Fq '"version_esquema_hmac":2' "$raiz/idempotencia/configuracion.json" ||
    fallar "version HMAC de idempotencia incoherente"
  grep -Fq '"generacion":2,"referencia_localizador":"clave:hmac:convocatorias:localizador:desarrollo:v2","referencia_huella_solicitud":"clave:hmac:convocatorias:huella:desarrollo:v2"' \
    "$raiz/idempotencia/configuracion.json" || fallar "generacion primaria HMAC incoherente"
  grep -Fq '"generacion":1,"referencia_localizador":"clave:hmac:convocatorias:localizador:desarrollo:v1","referencia_huella_solicitud":"clave:hmac:convocatorias:huella:desarrollo:v1"' \
    "$raiz/idempotencia/configuracion.json" || fallar "generacion historica HMAC incoherente"
  grep -Fq '"idempotencia_hmac":"idempotencia-hmac-fichero-local-v1"' "$raiz/manifiesto.json" ||
    fallar "proveedor HMAC de idempotencia ausente en manifiesto"
  grep -Fq '"version":4' "$raiz/manifiesto.json" || fallar "version de manifiesto incoherente"
  [[ "$(sha256sum -- "$raiz/idempotencia/configuracion.json" | awk '{print $1}')" == \
    "$(printf '%s\n' "$CONFIGURACION_IDEMPOTENCIA_CANONICA" | sha256sum | awk '{print $1}')" ]] ||
    fallar "configuracion HMAC alterada respecto del acuerdo generado"
  grep -Fxq 'VEC_EXECUTION_PROFILE=desarrollo' "$raiz/desarrollo.env" || fallar "perfil ausente en desarrollo.env"
  grep -Fxq 'VEC_AUTH_MODE=desarrollo' "$raiz/desarrollo.env" || fallar "modo ausente en desarrollo.env"
  grep -Fxq 'VEC_DEVELOPMENT_GUARD=ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO' "$raiz/desarrollo.env" ||
    fallar "segunda llave ausente en desarrollo.env"
}

if [[ -e "$DESTINO" || -L "$DESTINO" ]]; then
  verificar_directorio "$DESTINO"
  printf 'Credenciales de desarrollo ya existentes y verificadas: %s\n' "$DESTINO"
  mostrar_instrucciones_navegador "$DESTINO"
  exit 0
fi

TEMPORAL=$(mktemp -d -- "$PADRE/.vec-desarrollo.XXXXXX")
limpiar() {
  if [[ -n ${TEMPORAL:-} && -d ${TEMPORAL:-} ]]; then
    rm -rf -- "$TEMPORAL"
  fi
}
trap limpiar EXIT HUP INT TERM

for subdirectorio in ca tls mtls kms tsa identidad idempotencia; do
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
openssl rand -hex 32 >"$TEMPORAL/mtls/cliente.p12.password"
openssl pkcs12 -export -out "$TEMPORAL/mtls/cliente.p12" \
  -inkey "$TEMPORAL/mtls/cliente.key" -in "$TEMPORAL/mtls/cliente.crt" \
  -certfile "$TEMPORAL/ca/ca.crt" -name 'VEC desarrollo - operador RRHH' \
  -passout "file:$TEMPORAL/mtls/cliente.p12.password" 2>/dev/null

openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "$TEMPORAL/mtls/intervencion.key" 2>/dev/null
openssl req -new -sha256 -key "$TEMPORAL/mtls/intervencion.key" \
  -subj '/CN=intervencion-desarrollo/O=VEC Desarrollo/OU=NO AUTORITATIVO' \
  -out "$TEMPORAL/mtls/intervencion.csr" 2>/dev/null
cat >"$TEMPORAL/mtls/intervencion.ext" <<'EXT'
basicConstraints=critical,CA:FALSE
keyUsage=critical,digitalSignature
extendedKeyUsage=clientAuth
subjectAltName=URI:urn:vec:desarrollo:intervencion
subjectKeyIdentifier=hash
authorityKeyIdentifier=keyid,issuer
EXT
openssl x509 -req -sha256 -days 397 -in "$TEMPORAL/mtls/intervencion.csr" \
  -CA "$TEMPORAL/ca/ca.crt" -CAkey "$TEMPORAL/ca/ca.key" -CAserial "$TEMPORAL/ca/serie" \
  -extfile "$TEMPORAL/mtls/intervencion.ext" -out "$TEMPORAL/mtls/intervencion.crt" 2>/dev/null
openssl rand -hex 32 >"$TEMPORAL/mtls/intervencion.p12.password"
openssl pkcs12 -export -out "$TEMPORAL/mtls/intervencion.p12" \
  -inkey "$TEMPORAL/mtls/intervencion.key" -in "$TEMPORAL/mtls/intervencion.crt" \
  -certfile "$TEMPORAL/ca/ca.crt" -name 'VEC desarrollo - Intervencion' \
  -passout "file:$TEMPORAL/mtls/intervencion.p12.password" 2>/dev/null

rm -f -- "$TEMPORAL/tls/servidor.csr" "$TEMPORAL/tls/servidor.ext" \
  "$TEMPORAL/mtls/cliente.csr" "$TEMPORAL/mtls/cliente.ext" \
  "$TEMPORAL/mtls/intervencion.csr" "$TEMPORAL/mtls/intervencion.ext"
openssl rand 32 >"$TEMPORAL/kms/clave-maestra.bin"
openssl genpkey -algorithm ED25519 -out "$TEMPORAL/kms/atestacion-ed25519.key" 2>/dev/null
openssl pkey -in "$TEMPORAL/kms/atestacion-ed25519.key" -pubout -out "$TEMPORAL/kms/atestacion-ed25519.pub" 2>/dev/null
openssl genpkey -algorithm ED25519 -out "$TEMPORAL/kms/revalidacion-ed25519.key" 2>/dev/null
openssl pkey -in "$TEMPORAL/kms/revalidacion-ed25519.key" -pubout -out "$TEMPORAL/kms/revalidacion-ed25519.pub" 2>/dev/null
openssl rand 32 >"$TEMPORAL/tsa/clave-hmac.bin"
openssl rand 32 >"$TEMPORAL/idempotencia/g2-localizador.bin"
openssl rand 32 >"$TEMPORAL/idempotencia/g2-huella-solicitud.bin"
openssl rand 32 >"$TEMPORAL/idempotencia/g1-localizador.bin"
openssl rand 32 >"$TEMPORAL/idempotencia/g1-huella-solicitud.bin"

HUELLA_CA=$(openssl x509 -in "$TEMPORAL/ca/ca.crt" -noout -fingerprint -sha256 | cut -d= -f2 | tr -d ':')
HUELLA_SERVIDOR=$(openssl x509 -in "$TEMPORAL/tls/servidor.crt" -noout -fingerprint -sha256 | cut -d= -f2 | tr -d ':')
# La huella del manifiesto nace de la identidad importable y se coteja tambien
# con el PEM; asi ambas representaciones mantienen una sola identidad canonica.
HUELLA_CLIENTE=$(huella_certificado_cliente_pkcs12 \
  "$TEMPORAL/mtls/cliente.p12" "$TEMPORAL/mtls/cliente.p12.password")
HUELLA_INTERVENCION=$(huella_certificado_cliente_pkcs12 \
  "$TEMPORAL/mtls/intervencion.p12" "$TEMPORAL/mtls/intervencion.p12.password")
HUELLA_PUBLICA_ATESTACION_KMS=$(openssl pkey -pubin -in "$TEMPORAL/kms/atestacion-ed25519.pub" -outform DER 2>/dev/null | sha256sum | awk '{print $1}')
HUELLA_PUBLICA_REVALIDACION_KMS=$(openssl pkey -pubin -in "$TEMPORAL/kms/revalidacion-ed25519.pub" -outform DER 2>/dev/null | sha256sum | awk '{print $1}')

cat >"$TEMPORAL/identidad/identidad.json" <<JSON
{"version":1,"autoridad":"no_autoritativo","certificate_sha256":"$HUELLA_CLIENTE","subject":"desarrollo:operador-rrhh","display_name":"Operador RRHH de desarrollo","roles":["tecnico_rrhh"]}
JSON
cat >"$TEMPORAL/identidad/intervencion.json" <<JSON
{"version":1,"autoridad":"no_autoritativo","certificate_sha256":"$HUELLA_INTERVENCION","subject":"desarrollo:intervencion","display_name":"Intervencion de desarrollo","roles":["intervencion"]}
JSON
printf '%s\n' "$CONFIGURACION_IDEMPOTENCIA_CANONICA" >"$TEMPORAL/idempotencia/configuracion.json"
cat >"$TEMPORAL/manifiesto.json" <<JSON
{"version":4,"perfil":"desarrollo","autoridad":"no_autoritativo","migrable_a_produccion":false,"huella_ca_sha256":"$HUELLA_CA","huella_servidor_sha256":"$HUELLA_SERVIDOR","huella_cliente_sha256":"$HUELLA_CLIENTE","huella_intervencion_sha256":"$HUELLA_INTERVENCION","huella_publica_atestacion_kms_sha256":"$HUELLA_PUBLICA_ATESTACION_KMS","huella_publica_revalidacion_kms_sha256":"$HUELLA_PUBLICA_REVALIDACION_KMS","proveedores":{"identidad":"identidad-mtls-local-v1","idempotencia_hmac":"idempotencia-hmac-fichero-local-v1","kms_emisor":"kms-emisor-fichero-local-v2","kms_revalidador":"kms-revalidador-ed25519-local-v1","kms_verificador_recibo":"kms-verificador-publico-local-v1","tsa":"tsa-determinista-local-v1","tls":"tls-ca-local-v1"}}
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
if mv --no-target-directory --update=none-fail -- "$TEMPORAL" "$DESTINO"; then
  [[ ! -e "$TEMPORAL" && ! -L "$TEMPORAL" ]] ||
    fallar "la publicacion no consumio el directorio temporal"
  TEMPORAL=''
elif [[ -e "$DESTINO" || -L "$DESTINO" ]]; then
  verificar_directorio "$DESTINO"
  printf 'Otro proceso publico credenciales de desarrollo validas: %s\n' "$DESTINO"
  mostrar_instrucciones_navegador "$DESTINO"
  exit 0
else
  fallar "no se pudieron publicar atomicamente las credenciales de desarrollo"
fi

printf 'Credenciales de desarrollo generadas fuera de Git: %s\n' "$DESTINO"
printf 'CA SHA-256: %s\n' "$HUELLA_CA"
printf 'Cliente mTLS SHA-256: %s\n' "$HUELLA_CLIENTE"
printf 'Intervencion mTLS SHA-256: %s\n' "$HUELLA_INTERVENCION"
printf 'Cargue la configuracion solo en una terminal local: set -a; source %q; set +a\n' "$DESTINO/desarrollo.env"
mostrar_instrucciones_navegador "$DESTINO"
