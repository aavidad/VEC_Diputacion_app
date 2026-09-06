#!/usr/bin/env bash
# Abre el portal local con un perfil Firefox exclusivo de desarrollo.
# No modifica la confianza del sistema ni los perfiles personales.
set -euo pipefail
umask 077

fallar() { printf '%s\n' "$*" >&2; exit 1; }
if [[ $# -lt 1 || $# -gt 2 || (${2:-} != '' && ${2:-} != --solo-preparar) ]]; then
  fallar "Uso: $0 DIRECTORIO_MATERIAL [--solo-preparar]"
fi
for programa in firefox certutil pk12util openssl curl realpath stat sha256sum flock install rg; do
  command -v "$programa" >/dev/null || fallar "Falta el programa: $programa"
done

VEC_MATERIAL=$(realpath -e -- "$1")
[[ -d $VEC_MATERIAL && -O $VEC_MATERIAL ]] || fallar 'Material de desarrollo no disponible para este usuario.'
for archivo in ca/ca.crt mtls/cliente.crt mtls/cliente.key mtls/cliente.p12 mtls/cliente.p12.password; do
  ruta="$VEC_MATERIAL/$archivo"
  [[ -f $ruta && -r $ruta && -O $ruta && ! -L $ruta ]] || fallar "Material no disponible: $archivo"
done
for archivo in mtls/cliente.key mtls/cliente.p12 mtls/cliente.p12.password; do
  permisos=$(stat -c '%a' -- "$VEC_MATERIAL/$archivo")
  (( (8#$permisos & 077) == 0 )) || fallar "El material privado permite acceso a otros usuarios: $archivo"
done
openssl verify -purpose sslclient -CAfile "$VEC_MATERIAL/ca/ca.crt" \
  "$VEC_MATERIAL/mtls/cliente.crt" >/dev/null || fallar 'Certificado de cliente no válido.'
VEC_CERT_PEM=$(openssl x509 -in "$VEC_MATERIAL/mtls/cliente.crt" -outform DER | sha256sum)
VEC_CERTIFICADOS_P12=$(openssl pkcs12 -in "$VEC_MATERIAL/mtls/cliente.p12" \
  -passin "file:$VEC_MATERIAL/mtls/cliente.p12.password" -clcerts -nokeys)
[[ $(rg -c '^-----BEGIN CERTIFICATE-----$' <<< "$VEC_CERTIFICADOS_P12") == 1 ]] ||
  fallar 'El paquete del navegador debe contener una única identidad cliente.'
VEC_CERT_P12=$(openssl x509 -outform DER <<< "$VEC_CERTIFICADOS_P12" | sha256sum)
[[ $VEC_CERT_PEM == "$VEC_CERT_P12" ]] ||
  fallar 'El certificado del navegador no coincide con la identidad comprobada del portal.'
VEC_URL='https://localhost:8443/portal-empleado/'
curl --fail --silent --show-error --connect-timeout 2 --max-time 5 \
  --cacert "$VEC_MATERIAL/ca/ca.crt" --cert "$VEC_MATERIAL/mtls/cliente.crt" \
  --key "$VEC_MATERIAL/mtls/cliente.key" --output /dev/null "$VEC_URL" ||
  fallar 'El portal no responde con la identidad de desarrollo. Compruebe el lanzador de la aplicación.'

VEC_PERFIL="$VEC_MATERIAL/navegador-vec-rrhh"
# Firefox distribuido como Snap no puede leer ~/.local/state.
# Su directorio común permite mantener otro perfil sin tocar el habitual.
if [[ -x /snap/bin/firefox && -d ${HOME:?}/snap/firefox/common ]]; then
  VEC_PERFIL="$HOME/snap/firefox/common/navegador-vec-rrhh"
fi
[[ ! -L $VEC_PERFIL ]] || fallar 'El perfil dedicado no puede ser un enlace.'
if [[ ! -e $VEC_PERFIL ]]; then
  mkdir -m 700 -- "$VEC_PERFIL"
  mkdir -m 700 -- "$VEC_PERFIL/.vec-desarrollo"
fi
[[ -d $VEC_PERFIL && -O $VEC_PERFIL && $(stat -c '%a' -- "$VEC_PERFIL") == 700 &&
   -d $VEC_PERFIL/.vec-desarrollo && ! -L $VEC_PERFIL/.vec-desarrollo ]] ||
  fallar 'El destino no es un perfil privado creado por este lanzador.'
for archivo in .arranque.lock user.js cert9.db key4.db pkcs11.txt; do
  [[ ! -L $VEC_PERFIL/$archivo ]] || fallar 'El perfil contiene un enlace no permitido.'
done
exec 9>"$VEC_PERFIL/.arranque.lock"
flock -n 9 || fallar 'El navegador de VEC ya está abierto. Utilice su ventana.'

VEC_HUELLA=$(sha256sum "$VEC_MATERIAL/ca/ca.crt" "$VEC_MATERIAL/mtls/cliente.p12" | sha256sum | cut -d ' ' -f 1)
VEC_MARCA="$VEC_PERFIL/.vec-desarrollo/material.sha256"
[[ ! -L $VEC_MARCA ]] || fallar 'La marca del perfil no puede ser un enlace.'
if [[ -e $VEC_MARCA ]]; then
  read -r VEC_HUELLA_ANTERIOR < "$VEC_MARCA"
  [[ $VEC_HUELLA_ANTERIOR == "$VEC_HUELLA" ]] ||
    fallar 'El material cambió. Revise el perfil dedicado antes de sustituir certificados.'
else
  if [[ ! -f $VEC_PERFIL/cert9.db ]]; then
    certutil -N -d "sql:$VEC_PERFIL" --empty-password
  fi
  certutil -A -d "sql:$VEC_PERFIL" -n 'VEC desarrollo local' -t 'C,,' -i "$VEC_MATERIAL/ca/ca.crt"
  pk12util -i "$VEC_MATERIAL/mtls/cliente.p12" -d "sql:$VEC_PERFIL" \
    -K '' -w "$VEC_MATERIAL/mtls/cliente.p12.password" >/dev/null
  printf '%s\n' "$VEC_HUELLA" > "$VEC_MARCA"
fi
VEC_SCRIPTS=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
install -m 600 -- "$VEC_SCRIPTS/vec_navegador_desarrollo.js" "$VEC_PERFIL/user.js"
printf 'Portal: %s\nPerfil exclusivo: %s\n' "$VEC_URL" "$VEC_PERFIL"
[[ ${2:-} == --solo-preparar ]] && exit 0
[[ -n ${DISPLAY:-} || -n ${WAYLAND_DISPLAY:-} ]] || fallar 'No hay sesión gráfica para abrir el navegador.'

# Mantener el bloqueo en el padre aunque Firefox cierre descriptores heredados.
firefox --no-remote --profile "$VEC_PERFIL" --new-window "$VEC_URL" &
VEC_NAVEGADOR_PID=$!
trap 'kill -TERM "$VEC_NAVEGADOR_PID" 2>/dev/null || true' INT TERM
wait "$VEC_NAVEGADOR_PID"
