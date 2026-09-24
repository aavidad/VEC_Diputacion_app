#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
script="$raiz/exportar_png_base.sh"
bash -n "$script"

if "$script" --salida /no-usar --tesela 13/1/1 >/dev/null 2>&1; then
  echo 'ERROR: aceptó zoom fuera del contrato.' >&2; exit 1
fi
if OSM_EXPORTACION_COMPLETA_CONFIRMADA=no "$script" --salida /no-usar --completo >/dev/null 2>&1; then
  echo 'ERROR: aceptó campaña completa sin confirmación.' >&2; exit 1
fi

if [[ -z "${OSM_EXPORTACION_ESTADO:-}" ]]; then
  echo 'OK: contrato estático; para prueba real indique OSM_EXPORTACION_ESTADO y OSM_EXPORTACION_SALIDA_PRUEBA.'
  exit 0
fi
[[ -n "${OSM_EXPORTACION_SALIDA_PRUEBA:-}" ]] || {
  echo 'ERROR: falta OSM_EXPORTACION_SALIDA_PRUEBA privada.' >&2; exit 1
}
mkdir -p -- "$OSM_EXPORTACION_SALIDA_PRUEBA"
temporal="$(mktemp -d -- "$OSM_EXPORTACION_SALIDA_PRUEBA/.prueba-osm.XXXXXXXX")"
trap 'rm -rf -- "$temporal"' EXIT

# La validación de procedencia debe fallar antes de iniciar Docker.
mkdir -p -- "$temporal/falsa/releases/20260719T115334Z-53aba0ad43c4"
printf 'fuente distinta\n' >"$temporal/falsa/releases/20260719T115334Z-53aba0ad43c4/granada.mbtiles"
cp -- "$OSM_EXPORTACION_ESTADO/releases/20260719T115334Z-53aba0ad43c4/manifiesto.json" \
  "$temporal/falsa/releases/20260719T115334Z-53aba0ad43c4/manifiesto.json"
if OSM_EXPORTACION_ESTADO="$temporal/falsa" "$script" --salida "$temporal/falsa-salida" --tesela 8/125/99 >/dev/null 2>&1; then
  echo 'ERROR: aceptó MBTiles con huella distinta.' >&2; exit 1
fi
if OSM_EXPORTACION_ESTADO="$OSM_EXPORTACION_ESTADO" "$script" \
  --salida "$temporal/ausente" --tesela 10/503/400 >/dev/null 2>&1; then
  echo 'ERROR: fabricó una tesela exterior sin vector en el MBTiles.' >&2; exit 1
fi

for n in 1 2; do
  OSM_EXPORTACION_ESTADO="$OSM_EXPORTACION_ESTADO" "$script" \
    --salida "$temporal/salida-$n" --tesela 8/125/99 >/dev/null
done
python3 - "$temporal/salida-1" "$temporal/salida-2" <<'PY'
import hashlib, json, pathlib, struct, sys, zipfile
out1, out2 = map(pathlib.Path, sys.argv[1:])
basename = 'granada-base-20260719-z8-z12-muestra-8-125-99'
for out in (out1, out2):
    zip_path = out / (basename + '.zip')
    meta = json.loads((out / (basename + '.index.json')).read_text(encoding='utf-8'))
    digest = hashlib.sha256(zip_path.read_bytes()).hexdigest()
    assert meta['sha256_zip'] == digest
    assert meta['version_mbtiles'] == '20260719T115334Z-53aba0ad43c4'
    assert meta['sha256_mbtiles'] == '1ad538ef1f9331eca95137edc078d3a3b77336d54cabc838524172fba6547ebe'
    assert meta['total'] == 1
    assert meta['posiciones_bbox_importacion'] == 1246
    assert len(meta['sin_vector_en_bbox_importacion']) == 206
    assert '10/503/400' in meta['sin_vector_en_bbox_importacion']
    with zipfile.ZipFile(zip_path) as archive:
        assert archive.namelist() == ['tiles/8/125/99.png']
        info = archive.infolist()[0]
        assert info.date_time == (1980, 1, 1, 0, 0, 0)
        data = archive.read(info)
        assert data[:8] == b'\x89PNG\r\n\x1a\n'
        assert struct.unpack('>II', data[16:24]) == (256, 256)
        assert meta['teselas'][0]['sha256'] == hashlib.sha256(data).hexdigest()
assert hashlib.sha256((out1 / (basename + '.zip')).read_bytes()).hexdigest() == \
       hashlib.sha256((out2 / (basename + '.zip')).read_bytes()).hexdigest()
print('OK: fuente histórica, PNG 256×256, ZIP canónico e índice versionado reproducibles.')
PY
