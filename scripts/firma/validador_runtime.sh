#!/usr/bin/env bash
set -euo pipefail
mkdir -p /work/datos/crl /work/home /work/tmp /work/cache /work/bin

(
  cd /work/src
  go build -trimpath -o /work/bin/autofirmav2 ./cmd/autofirma
  go build -trimpath -o /work/bin/fixture ./cmd/vecfixture
)
(
  cd /work/vec
  go build -trimpath -o /work/bin/clientevec ./cmd/clientevec
)
/work/bin/fixture /work/datos

# Credencial sintética efímera; no aparece en argumentos ni en el registro.
head -c 32 /dev/urandom | base64 -w0 > /work/datos/token
chmod 600 /work/datos/token
AUTOFIRMAV2_REST_TOKEN="$(cat /work/datos/token)" /work/bin/autofirmav2 -rest-solo-verificacion \
  -verificacion-anclas /work/datos/ancla.pem -verificacion-crl /work/datos/crl \
  -direccion-rest 127.0.0.1:63118 > /work/servicio.log 2>&1 &
server_pid=$!
cleanup() { kill "$server_pid" 2>/dev/null || true; wait "$server_pid" 2>/dev/null || true; }
trap cleanup EXIT

tls_ca=/work/home/.config/autofirma-v2/tls/rest-localhost-root.crt.pem
ready=false
for _ in $(seq 1 100); do
  if [[ -f "$tls_ca" ]] && curl -fsS --tlsv1.3 --tls-max 1.3 --max-time 1 \
    --cacert "$tls_ca" https://127.0.0.1:63118/health -o /work/salud.json 2>/dev/null; then
    ready=true
    break
  fi
  kill -0 "$server_pid" 2>/dev/null || { echo 'El validador terminó antes de responder' >&2; exit 1; }
  sleep 0.1
done
[[ "$ready" == true ]] || { echo 'El validador no quedó listo' >&2; exit 1; }
jq -e '.ok == true and .modo == "solo_verificacion"' /work/salud.json >/dev/null

cliente() { /work/bin/clientevec "$1" /work/datos "$tls_ca"; }
cliente valida
cp -- /work/datos/intermedia_revocada.crl /work/datos/crl/intermedia.crl
cliente revocado
rm -- /work/datos/crl/intermedia.crl
cliente crl_incompleta

# Los fallos de transporte y autenticación deben quedar indeterminados.
cliente ca_incorrecta
cliente nombre_incorrecto
cliente credencial_incorrecta
cliente firmado_alterado

# El servicio de solo verificación no monta la ruta de firma.
status="$(curl -sS --tlsv1.3 --tls-max 1.3 --max-time 5 -o /work/ruta.json -w '%{http_code}' \
  --cacert "$tls_ca" -H "Authorization: Bearer $(cat /work/datos/token)" \
  -d '{}' https://127.0.0.1:63118/sign)"
[[ "$status" == 404 ]] || { echo "La ruta /sign respondió $status" >&2; exit 1; }
echo 'Ensayo real VEC→GrxFirma: siete casos, TLS 1.3, CA local y /sign=404 verificados'
