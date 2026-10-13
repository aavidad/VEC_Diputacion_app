#!/usr/bin/env bash
set -euo pipefail
umask 077

fallar() { printf '{"error":"%s"}\n' "$1" >&2; exit 1; }
[[ $# -ge 1 ]] || fallar argumentos
modo=$1
shift
case "$modo" in
  --comprobar) ;;
  --aplicar)
    [[ ${1:-} == PUBLICA-PLAN-FIRMA-CT-V2 ]] || fallar confirmacion
    shift ;;
  *) fallar argumentos ;;
esac
[[ $# == 5 ]] || fallar argumentos
crear=$1
sha_crear=$2
publicar=$3
sha_publicar=$4
salida=$5
bin=${VEC_PLAN_FIRMA_VALIDAR_BIN:?}
[[ -x "$bin" ]] || fallar validador

trabajo=$(python3 - "$salida" <<'PY'
import os, pathlib, stat, sys, tempfile
p = pathlib.Path(sys.argv[1])
s = p.lstat()
if not stat.S_ISDIR(s.st_mode) or stat.S_IMODE(s.st_mode) != 0o700 or s.st_uid != os.getuid():
    raise SystemExit('carpeta_privada_requerida')
print(tempfile.mkdtemp(prefix='plan-v2-', dir=p))
PY
)
"$bin" "$crear" "$sha_crear" > "$trabajo/crear-validacion.json"
"$bin" "$publicar" "$sha_publicar" > "$trabajo/publicar-validacion.json"

catalogo=$(python3 - "$crear" "$publicar" "$trabajo" "$sha_crear" "$sha_publicar" <<'PY'
import base64, hashlib, json, pathlib, sys
materiales = [pathlib.Path(p).read_bytes() for p in sys.argv[1:3]]
if [hashlib.sha256(b).hexdigest() for b in materiales] != sys.argv[4:6]:
    raise SystemExit('material_modificado')
envolturas = [json.loads(b) for b in materiales]
catalogos = [json.loads(base64.b64decode(e['catalogo_canonico_base64'], validate=True)) for e in envolturas]
a, b = catalogos
if (envolturas[0]['operacion'] != 'crear' or envolturas[1]['operacion'] != 'publicar'
        or a['id'] != b['id'] or a['version'] != 2 or b['version'] != 2
        or a['creado_por'] != b['creado_por'] or a['entradas'] != b['entradas']
        or a['creado_por'] == b['publicado_por']
        or envolturas[1]['huella_esperada'] != envolturas[0]['catalogo_sha256']
        or any(e['atributos']['circuito_ref'] != 'vec.contratacion_temporal.circuito_firma:2'
               for e in b['entradas'])):
    raise SystemExit('materiales_cruzados')
for nombre, material in zip(('crear', 'publicar'), materiales):
    p = pathlib.Path(sys.argv[3]) / (nombre + '-peticion.json')
    with p.open('x') as f:
        json.dump({'material_base64': base64.b64encode(material).decode()}, f, separators=(',', ':'))
print(b['id'] + ':2')
PY
)
if [[ "$modo" == --comprobar ]]; then
  python3 - "$catalogo" "$trabajo" <<'PY'
import json, sys
print(json.dumps({'estado': 'preparado_sin_publicar', 'catalogo_ref': sys.argv[1], 'carpeta': sys.argv[2]}))
PY
  exit 0
fi

url=${VEC_ADMIN_PLAN_ORIGEN:?}
ca=${VEC_ADMIN_PLAN_CA:?}
cert_a=${VEC_ADMIN_PLAN_CERT_A:?}
clave_a=${VEC_ADMIN_PLAN_CLAVE_A:?}
cert_b=${VEC_ADMIN_PLAN_CERT_B:?}
clave_b=${VEC_ADMIN_PLAN_CLAVE_B:?}
python3 - "$url" <<'PY'
import sys, urllib.parse
u = urllib.parse.urlsplit(sys.argv[1])
if u.scheme != 'https' or not u.hostname or u.username is not None or u.password is not None or u.path or u.query or u.fragment:
    raise SystemExit('origen_https_requerido')
PY
sha_a=$(openssl x509 -in "$cert_a" -outform DER | sha256sum | cut -d ' ' -f 1)
sha_b=$(openssl x509 -in "$cert_b" -outform DER | sha256sum | cut -d ' ' -f 1)
[[ "$sha_a" != "$sha_b" ]] || fallar certificados_iguales

enviar() {
  local operacion=$1 certificado=$2 clave=$3 estado=$4 codigo
  codigo=$(curl -q --silent --show-error --noproxy '*' --proto '=https' \
    --connect-timeout 10 --max-time 60 --max-filesize 65536 \
    --cacert "$ca" --cert "$certificado" --key "$clave" \
    --request POST --header 'Content-Type: application/json' --header 'Accept: application/json' \
    --data-binary "@$trabajo/$operacion-peticion.json" \
    --output "$trabajo/$operacion-respuesta.json" --write-out '%{http_code}' \
    "$url/api/admin/perfiles/v1/plan-firma/gobierno")
  [[ "$codigo" == 200 ]] || fallar "http_$codigo"
  python3 - "$trabajo/$operacion-respuesta.json" "$catalogo" "$estado" <<'PY'
import json, pathlib, sys
d = json.loads(pathlib.Path(sys.argv[1]).read_bytes())
r = d.get('recibo', {})
if r.get('catalogo_ref') != sys.argv[2] or r.get('estado') != sys.argv[3] or not r.get('recibo_ref'):
    raise SystemExit('recibo_no_corresponde')
print(json.dumps(r, ensure_ascii=False))
PY
}
enviar crear "$cert_a" "$clave_a" borrador
enviar publicar "$cert_b" "$clave_b" publicado
