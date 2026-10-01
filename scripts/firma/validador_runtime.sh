#!/usr/bin/env bash
set -euo pipefail
mkdir -p /work/datos/crl
go build -trimpath -o /work/bin/autofirmav2 ./cmd/autofirma
go build -trimpath -o /work/bin/fixture ./cmd/vecfixture
/work/bin/fixture /work/datos

token='vec-e3-token-sintetico-solo-ensayo'
AUTOFIRMAV2_REST_TOKEN="$token" /work/bin/autofirmav2 -rest-solo-verificacion \
  -verificacion-anclas /work/datos/ancla.pem -verificacion-crl /work/datos/crl \
  -direccion-rest 127.0.0.1:63118 > /work/servicio.log 2>&1 &
server_pid=$!
cleanup() { kill "$server_pid" 2>/dev/null || true; wait "$server_pid" 2>/dev/null || true; }
trap cleanup EXIT

for _ in $(seq 1 50); do
  if curl -ksS --max-time 1 https://127.0.0.1:63118/health -o /work/salud.json 2>/dev/null; then break; fi
  kill -0 "$server_pid" 2>/dev/null || { cat /work/servicio.log >&2; exit 1; }
  sleep 0.1
done
jq -e '.ok == true and .modo == "solo_verificacion"' /work/salud.json >/dev/null

verify() {
  local pdf="$1" expected="$2" reason="$3" revocation="$4" name="$5"
  jq -n --arg signed "$(base64 -w0 "/work/datos/$pdf")" \
    --arg original "$(base64 -w0 /work/datos/original.pdf)" \
    '{name:"ensayo.pdf",mime_type:"application/pdf",content_base64:$signed,original_content_base64:$original}' \
    > /work/peticion.json
  local status
  status="$(curl -ksS --max-time 10 -o "/work/$name.json" -w '%{http_code}' \
    -H "Authorization: Bearer $token" -H 'Content-Type: application/json' \
    --data-binary @/work/peticion.json https://127.0.0.1:63118/verify)"
  [[ "$status" == 200 ]] || { echo "HTTP $status en $name" >&2; cat "/work/$name.json" >&2; exit 1; }
  local signed_hash original_hash
  signed_hash="$(sha256sum "/work/datos/$pdf" | cut -d' ' -f1)"
  original_hash="$(sha256sum /work/datos/original.pdf | cut -d' ' -f1)"
  jq -e --arg state "$expected" --arg reason "$reason" --arg revocation "$revocation" \
    --arg signed_hash "$signed_hash" --arg original_hash "$original_hash" \
    '.dictamen.contrato == "autofirmav2.dictamen-verificacion.v1" and .dictamen.estado == $state and .dictamen.motivo == $reason and .dictamen.formato == "PAdES" and .dictamen.integridad.estado == "valida" and .dictamen.cadena.estado == "valida" and .dictamen.certificado.estado == "vigente" and .dictamen.revocacion.estado == $revocation and .dictamen.vinculoOriginal.estado == "acreditado" and .dictamen.huellaFirmadoSHA256 == $signed_hash and .dictamen.huellaOriginalSHA256 == $original_hash and (.dictamen.firmantes | length) == 1 and .dictamen.extensiones.revocacionRemota == "desactivada" and .dictamen.extensiones.selloTiempoRemoto == "desactivada"' \
    "/work/$name.json" >/dev/null || { jq '.dictamen' "/work/$name.json" >&2; exit 1; }
  jq -r --arg label "$name" '"\($label): \(.dictamen.estado)/\(.dictamen.motivo), integridad=\(.dictamen.integridad.estado), cadena=\(.dictamen.cadena.estado), revocacion=\(.dictamen.revocacion.estado)"' "/work/$name.json"
}
verify firmado.pdf valida verificada vigente valida
cp /work/datos/intermedia_revocada.crl /work/datos/crl/intermedia.crl
verify firmado.pdf no_valida certificado_no_valido revocado no_valida
rm /work/datos/crl/intermedia.crl
verify firmado.pdf indeterminada revocacion_no_acreditada no_comprobada indeterminada

# Una alteración de los bytes firmados se rechaza antes del dictamen.
jq -n --arg signed "$(base64 -w0 /work/datos/alterado.pdf)" --arg original "$(base64 -w0 /work/datos/original.pdf)" \
  '{name:"ensayo.pdf",mime_type:"application/pdf",content_base64:$signed,original_content_base64:$original}' > /work/peticion_alterada.json
status="$(curl -ksS --max-time 10 -o /work/alteracion.json -w '%{http_code}' \
  -H "Authorization: Bearer $token" -H 'Content-Type: application/json' \
  --data-binary @/work/peticion_alterada.json https://127.0.0.1:63118/verify)"
[[ "$status" == 400 ]] || { echo "La alteración respondió $status, se esperaba 400" >&2; exit 1; }

# La superficie del servicio no admite firma ni rutas de fichero.
status="$(curl -ksS --max-time 5 -o /work/ruta.json -w '%{http_code}' -H "Authorization: Bearer $token" -d '{}' https://127.0.0.1:63118/sign)"
[[ "$status" == 404 ]] || { echo "La ruta /sign respondió $status" >&2; exit 1; }
echo 'salud=200, /sign=404, alteracion=400; tres dictámenes HTTP 200 comprobados'
