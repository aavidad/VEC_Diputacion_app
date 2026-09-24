#!/usr/bin/env bash
set -Eeuo pipefail

# Exporta el MBTiles HISTORICO con un TileServer GL aislado. Nunca consulta 8091.
raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repositorio="$(cd -- "$raiz/../.." && pwd)"
version="20260719T115334Z-53aba0ad43c4"
estado="${OSM_EXPORTACION_ESTADO:-$raiz/estado}"
mbtiles="$estado/releases/$version/granada.mbtiles"
manifiesto="$estado/releases/$version/manifiesto.json"
estilo="$raiz/config/estilos/osm-granada.json"
configuracion="$raiz/config/tileserver.json"
imagen='maptiler/tileserver-gl:v5.6.0@sha256:3a9ccdb24820b6814c8119bcc8a4376c39867cb0ffe69d62919ef898b90c2427'
sha_mbtiles='1ad538ef1f9331eca95137edc078d3a3b77336d54cabc838524172fba6547ebe'
sha_estilo='ff8c36d2a9b64f15c8b4cc0bd115ed2973e9469cedd95d2e18a3085290a6b233'
sha_configuracion='10c0e8d21c47c2d4eb66c4c1644dc57fc3c869495b3523f3eecf011aa0340812'
id_imagen='sha256:fa01ab9f902f0a8ae80b3486476a8427d1625e677ccc53c326ddae55a2eb0e29'

uso() {
  echo "Uso: $0 --salida DIRECTORIO (--tesela Z/X/Y | --completo)" >&2
  echo 'La exportación completa exige OSM_EXPORTACION_COMPLETA_CONFIRMADA=si.' >&2
  exit 2
}

salida=''
tesela=''
completo=false
while (($#)); do
  case "$1" in
    --salida) (($# >= 2)) || uso; salida="$2"; shift 2 ;;
    --tesela) (($# >= 2)) || uso; tesela="$2"; shift 2 ;;
    --completo) completo=true; shift ;;
    *) uso ;;
  esac
done
[[ -n "$salida" ]] || uso
if [[ "$completo" == true ]]; then
  [[ -z "$tesela" && "${OSM_EXPORTACION_COMPLETA_CONFIRMADA:-}" == si ]] || uso
else
  [[ "$tesela" =~ ^(8|9|10|11|12)/([0-9]+)/([0-9]+)$ ]] || uso
  z="${BASH_REMATCH[1]}"; x="${BASH_REMATCH[2]}"; y="${BASH_REMATCH[3]}"
  [[ "$x" =~ ^(0|[1-9][0-9]*)$ && "$y" =~ ^(0|[1-9][0-9]*)$ ]] || uso
  ((x < 1 << z && y < 1 << z)) || uso
fi

for programa in docker python3 sqlite3 sha256sum; do
  command -v "$programa" >/dev/null || { echo "ERROR: falta $programa" >&2; exit 1; }
done
[[ -f "$mbtiles" && ! -L "$mbtiles" && -f "$manifiesto" && -f "$estilo" && -f "$configuracion" ]] || {
  echo 'ERROR: faltan archivos de la versión histórica.' >&2; exit 1;
}
[[ "$(sha256sum "$mbtiles" | cut -d' ' -f1)" == "$sha_mbtiles" ]] || {
  echo 'ERROR: huella MBTiles histórica distinta.' >&2; exit 1;
}
[[ "$(sha256sum "$estilo" | cut -d' ' -f1)" == "$sha_estilo" ]] || {
  echo 'ERROR: estilo distinto del fijado.' >&2; exit 1;
}
[[ "$(sha256sum "$configuracion" | cut -d' ' -f1)" == "$sha_configuracion" ]] || {
  echo 'ERROR: configuración distinta de la fijada.' >&2; exit 1;
}
python3 - "$manifiesto" "$version" "$sha_mbtiles" <<'PY'
import json, sys
m = json.load(open(sys.argv[1], encoding='utf-8'))
if m.get('version') != sys.argv[2] or m.get('sha256_mbtiles') != sys.argv[3]:
    raise SystemExit('ERROR: manifiesto histórico incoherente.')
PY
[[ "$(docker image inspect "$imagen" --format '{{.Id}}' 2>/dev/null)" == "$id_imagen" ]] || {
  echo 'ERROR: imagen TileServer GL fijada no disponible localmente.' >&2; exit 1;
}

mkdir -p -- "$salida"
salida="$(realpath -- "$salida")"
[[ "$salida" != "$repositorio"* ]] || {
  echo 'ERROR: use una salida privada fuera del repositorio.' >&2; exit 1;
}
if [[ "$completo" == true ]]; then
  nombre_previsto='granada-base-20260719-z8-z12'
else
  nombre_previsto="granada-base-20260719-z8-z12-muestra-$z-$x-$y"
fi
[[ ! -e "$salida/$nombre_previsto.zip" && ! -e "$salida/$nombre_previsto.index.json" ]] || {
  echo 'ERROR: el artefacto ya existe; no se sobrescribe.' >&2; exit 1;
}
temporal="$(mktemp -d -- "$salida/.osm-export.XXXXXXXX")"
contenedor="vec-osm-export-$(basename -- "$temporal" | tr -cd '[:alnum:]')"
contenedor_creado=false
limpiar() {
  if [[ "$contenedor_creado" == true ]]; then
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
  fi
  rm -rf -- "$temporal"
}
trap limpiar EXIT

python3 - "$configuracion" "$temporal/config.json" <<'PY'
import json, sys
with open(sys.argv[1], encoding='utf-8') as f:
    config = json.load(f)
if config['data']['granada']['mbtiles'] != 'activo/granada.mbtiles':
    raise SystemExit('ERROR: la configuración original cambió de fuente.')
config['data']['granada']['mbtiles'] = 'granada.mbtiles'
with open(sys.argv[2], 'w', encoding='utf-8') as f:
    json.dump(config, f, sort_keys=True, separators=(',', ':'))
PY

sqlite3 -readonly -separator / "$mbtiles" \
  'SELECT zoom_level,tile_column,((1 << zoom_level)-1-tile_row) FROM tiles WHERE zoom_level BETWEEN 8 AND 12 ORDER BY zoom_level,tile_column,((1 << zoom_level)-1-tile_row);' \
  >"$temporal/indice-tiles.txt"
if [[ "$completo" == true ]]; then
  cp -- "$temporal/indice-tiles.txt" "$temporal/seleccion.txt"
  nombre='granada-base-20260719-z8-z12'
else
  rg -Fx -- "$tesela" "$temporal/indice-tiles.txt" >"$temporal/seleccion.txt" || {
    echo 'ERROR: la tesela solicitada no pertenece al MBTiles histórico.' >&2; exit 1;
  }
  nombre="granada-base-20260719-z8-z12-muestra-$z-$x-$y"
fi
[[ -s "$temporal/seleccion.txt" ]] || { echo 'ERROR: selección vacía.' >&2; exit 1; }
mkdir -p -- "$temporal/estilos" "$temporal/tiles"
cp -- "$estilo" "$temporal/estilos/osm-granada.json"

docker run -d --rm --name "$contenedor" --network none --init \
  --read-only --cap-drop ALL --security-opt no-new-privileges \
  --pids-limit 256 --memory 1g --cpus 1 --shm-size 128m \
  --tmpfs /tmp:rw,noexec,nosuid,nodev,size=128m,mode=1777 \
  --tmpfs /home/node/.cache:rw,noexec,nosuid,nodev,size=32m,mode=0700,uid=999,gid=999 \
  --mount "type=bind,src=$temporal/config.json,dst=/data/config.json,readonly" \
  --mount "type=bind,src=$temporal/estilos,dst=/data/estilos,readonly" \
  --mount "type=bind,src=$mbtiles,dst=/data/teselas/granada.mbtiles,readonly" \
  "$imagen" --config /data/config.json --bind 127.0.0.1 --port 8080 --no-cors --silent \
  >/dev/null
contenedor_creado=true
[[ "$(docker inspect "$contenedor" --format '{{.HostConfig.NetworkMode}}')" == none ]] || {
  echo 'ERROR: el renderer no está aislado de la red.' >&2; exit 1;
}
for _ in {1..30}; do
  if docker exec "$contenedor" node -e 'fetch("http://127.0.0.1:8080/styles/osm-granada/256/8/125/99.png").then(r=>process.exit(r.ok?0:1)).catch(()=>process.exit(1))' >/dev/null 2>&1; then
    listo=true; break
  fi
  sleep 1
done
[[ "${listo:-false}" == true ]] || { echo 'ERROR: renderer aislado no respondió.' >&2; exit 1; }

while IFS=/ read -r z x y; do
  destino="$temporal/tiles/$z/$x/$y.png"
  mkdir -p -- "$(dirname -- "$destino")"
  docker exec "$contenedor" node -e '
    const [z,x,y]=process.argv.slice(1);
    fetch(`http://127.0.0.1:8080/styles/osm-granada/256/${z}/${x}/${y}.png`)
      .then(async r=>{if(!r.ok) throw Error(`HTTP ${r.status}`); process.stdout.write(Buffer.from(await r.arrayBuffer()));})
      .catch(e=>{console.error(e.message); process.exitCode=1});
  ' "$z" "$x" "$y" >"$destino"
  python3 - "$destino" <<'PY'
import sys
with open(sys.argv[1], 'rb') as f:
    if f.read(8) != b'\x89PNG\r\n\x1a\n':
        raise SystemExit('ERROR: la tesela no es PNG.')
PY
done <"$temporal/seleccion.txt"

python3 - "$temporal" "$salida" "$nombre" "$version" "$sha_mbtiles" "$sha_estilo" "$id_imagen" "$manifiesto" <<'PY'
import hashlib, json, math, pathlib, sys, zipfile
tmp, out = map(pathlib.Path, sys.argv[1:3])
name, version, source_sha, style_sha, image_id, manifest_path = sys.argv[3:]
tiles = sorted((tmp / 'tiles').rglob('*.png'))
zip_path = tmp / (name + '.zip')
index = []
with zipfile.ZipFile(zip_path, 'w', compression=zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
    for path in tiles:
        arcname = path.relative_to(tmp).as_posix()
        data = path.read_bytes()
        info = zipfile.ZipInfo(arcname, date_time=(1980, 1, 1, 0, 0, 0))
        info.compress_type = zipfile.ZIP_DEFLATED
        info.external_attr = 0o100644 << 16
        archive.writestr(info, data, compress_type=zipfile.ZIP_DEFLATED, compresslevel=9)
        index.append({'ruta': arcname, 'sha256': hashlib.sha256(data).hexdigest()})
digest = hashlib.sha256(zip_path.read_bytes()).hexdigest()
source_tiles = set((tmp / 'indice-tiles.txt').read_text(encoding='utf-8').splitlines())
manifest = json.loads(pathlib.Path(manifest_path).read_text(encoding='utf-8'))
west, south, east, north = manifest['bbox_importacion']
missing = []
positions = 0
for z in range(8, 13):
    n = 1 << z
    x0, x1 = math.floor((west + 180) / 360 * n), math.floor((east + 180) / 360 * n)
    def tile_y(lat):
        return math.floor((1 - math.asinh(math.tan(math.radians(lat))) / math.pi) / 2 * n)
    y0, y1 = tile_y(north), tile_y(south)
    for x in range(x0, x1 + 1):
        for y in range(y0, y1 + 1):
            positions += 1
            coord = f'{z}/{x}/{y}'
            if coord not in source_tiles:
                missing.append(coord)
meta = {'esquema': 'vec.osm.png-export.v1', 'version_mbtiles': version,
        'sha256_mbtiles': source_sha, 'sha256_estilo': style_sha,
        'imagen_tileserver_id': image_id, 'zoom_min': 8, 'zoom_max': 12,
        'cobertura': 'teselas presentes en MBTiles histórico; no acredita provincia + 15 km',
        'posiciones_bbox_importacion': positions,
        'sin_vector_en_bbox_importacion': missing,
        'teselas': index, 'total': len(index), 'sha256_zip': digest}
index_path = tmp / (name + '.index.json')
index_path.write_text(json.dumps(meta, sort_keys=True, ensure_ascii=False, separators=(',', ':')) + '\n', encoding='utf-8')
for path in (zip_path, index_path):
    target = out / path.name
    if target.exists():
        raise SystemExit(f'ERROR: no se sobrescribe {target}')
for path in (zip_path, index_path):
    target = out / path.name
    path.replace(target)
print(f'OK: {out / zip_path.name} teselas={len(index)} SHA256={digest}')
PY
