#!/bin/sh
set -eu

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ] || [ ! -d "$1" ]; then
  echo "uso: $0 RUTA_WEB_EXTRAIDA [MANIFIESTO]" >&2
  exit 2
fi

raiz="${1%/}"
estaticos="$raiz/static"
repositorio="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
manifiesto="${2:-$repositorio/web/produccion.manifest}"
inventario_real="$(mktemp)"
inventario_esperado="$(mktemp)"
limpiar() { rm -f "$inventario_real" "$inventario_esperado"; }
trap limpiar EXIT INT TERM

if [ ! -d "$estaticos" ] || [ ! -f "$manifiesto" ]; then
  echo "ERROR: el artefacto no contiene el directorio web/static" >&2
  exit 1
fi

LC_ALL=C sort "$manifiesto" >"$inventario_esperado"
find "$raiz" -type f -printf '%P\n' | LC_ALL=C sort >"$inventario_real"
if ! cmp -s "$inventario_esperado" "$inventario_real"; then
  echo "ERROR: el arbol web no coincide con el manifiesto productivo exacto" >&2
  diff -u "$inventario_esperado" "$inventario_real" >&2 || true
  exit 1
fi

if grep -Ev '^(produccion[.]manifest|cartografia/granada-base-20260719-z8-z12[.](zip|json)|static/(area-personal|bolsa|portal-empleado|verificar)/[^/].*|static/acceso/(index[.]html|acceso[.]css|acceso-i18n[.]js|locales/es[.]json)|static/assets/logo-diputacion-granada[.]svg|static/comun/(tema-vec[.](css|js)|iconos-vec[.]js)|static/comun/oportunidades/(vista[.]js|i18n[.]js|oportunidades[.]css)|static/(styles[.]css|favicon[.]svg))$' "$inventario_esperado" | grep -q .; then
  echo "ERROR: el manifiesto contiene una ruta fuera de las superficies permitidas" >&2
  exit 1
fi

if [ ! -f "$raiz/produccion.manifest" ] || ! cmp -s "$manifiesto" "$raiz/produccion.manifest"; then
  echo "ERROR: el artefacto no conserva el manifiesto exacto usado para construirlo" >&2
  exit 1
fi

for ruta in \
  cartografia/granada-base-20260719-z8-z12.zip \
  cartografia/granada-base-20260719-z8-z12.json
do
  if ! grep -Fxq "$ruta" "$manifiesto" || [ ! -f "$raiz/$ruta" ]; then
    echo "ERROR: falta la cartografia historica declarada en el artefacto" >&2
    exit 1
  fi
done

if ! python3 - "$raiz/cartografia/granada-base-20260719-z8-z12.zip" \
  "$raiz/cartografia/granada-base-20260719-z8-z12.json" <<'PY'
import hashlib
import json
import sys
import zipfile

zip_path, index_path = sys.argv[1:]
with open(index_path, "rb") as source:
    index_bytes = source.read()
index = json.loads(index_bytes)
assert hashlib.sha256(index_bytes).hexdigest() == "9d69233ad62c4371a2248bef2f91db8340823cb2a618272627693805ad021f81"
with open(zip_path, "rb") as source:
    zip_hash = hashlib.file_digest(source, "sha256").hexdigest()
assert zip_hash == index["sha256_zip"] == "0f0d78212832493c424699a42847069ae24b8b3717917780c4d64444aa250165"
assert index["esquema"] == "vec.osm.png-export.v1"
assert index["total"] == 1040 and len(index["teselas"]) == 1040
expected = {tile["ruta"]: tile["sha256"] for tile in index["teselas"]}
assert len(expected) == 1040
with zipfile.ZipFile(zip_path) as archive:
    members = archive.infolist()
    assert len(members) == 1040
    assert {member.filename for member in members} == set(expected)
    for member in members:
        assert hashlib.sha256(archive.read(member)).hexdigest() == expected[member.filename]
PY
then
  echo "ERROR: la cartografia del artefacto no coincide con su indice" >&2
  exit 1
fi

for ruta_prohibida in \
  "$estaticos/index.html" \
  "$estaticos/app.js" \
  "$estaticos/catalogo-categorias.js" \
  "$estaticos/catalogo-categorias.css" \
  "$estaticos/modulos" \
  "$estaticos/presentacion"
do
  if [ -e "$ruta_prohibida" ]; then
    echo "ERROR: el artefacto contiene la SPA historica o material no autorizado: $ruta_prohibida" >&2
    exit 1
  fi
done

if find "$raiz" -type l -print -quit | grep -q .; then
  echo "ERROR: el arbol web productivo contiene enlaces simbolicos" >&2
  exit 1
fi

# Solo estos JSON nominales; siguen sujetos al manifiesto y demás filtros.
if find "$estaticos" -type f \( \
  -iname '*presentacion*' -o -iname '*demo*' -o -iname '*fixture*' -o \
  -iname '*.test.js' -o -iname '*.test.mjs' -o \
  -iname '*.md' -o -iname '*.markdown' -o \
  \( -iname '*.json' ! -path "$estaticos/portal-empleado/modulos/contratacion-temporal/formalizacion-desarrollo.json" ! -path "$estaticos/acceso/locales/es.json" ! -path "$estaticos/area-personal/locales/es.json" \) -o \
  -iname '*.jsonl' -o -iname '*.ndjson' -o \
  -iname '*.zip' -o \
  -iname '*.csv' -o -iname '*.tsv' -o -iname '*.db' -o \
  -iname '*.sqlite' -o -iname '*.sqlite3' -o -iname '*.pem' -o -iname '*.key' \
  \) -print -quit | grep -q .; then
  echo "ERROR: el arbol web productivo contiene material sintetico, de prueba o datos plausibles" >&2
  exit 1
fi

if coincidencias="$(grep -RIlE \
  '(^|[^[:alnum:]_])(localStorage|sessionStorage|indexedDB)([^[:alnum:]_]|$)|document[.]cookie' \
  "$estaticos" 2>/dev/null)" && [ -n "$coincidencias" ]; then
  printf '%s\n' "$coincidencias" >&2
  echo "ERROR: el arbol web productivo contiene almacenamiento o cookies de navegador" >&2
  exit 1
fi

if coincidencias="$(grep -RIlE \
  'NIF-DEMO-0001|persona[.]demo@example[.]test|empleado[.]demo@dipgra[.]es|CAND-0007 - DNI|employeeDirectoryNIF|NOMINAS_CONTROL_DATA' \
  "$estaticos" 2>/dev/null)" && [ -n "$coincidencias" ]; then
  printf '%s\n' "$coincidencias" >&2
  echo "ERROR: el arbol web productivo contiene datos o generadores de la SPA historica" >&2
  exit 1
fi

echo "Arbol web productivo verificado."
