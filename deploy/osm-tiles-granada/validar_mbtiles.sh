#!/usr/bin/env bash
set -Eeuo pipefail

archivo="${1:-}"

if [[ -z "$archivo" || ! -f "$archivo" || -L "$archivo" ]]; then
  echo "ERROR: se esperaba un MBTiles regular: $archivo" >&2
  exit 1
fi

if ! command -v sqlite3 >/dev/null 2>&1; then
  echo "ERROR: sqlite3 es obligatorio para validar el artefacto." >&2
  exit 1
fi

if [[ "$(sqlite3 "$archivo" 'PRAGMA quick_check;')" != "ok" ]]; then
  echo "ERROR: PRAGMA quick_check no ha validado el MBTiles." >&2
  exit 1
fi

tablas="$(sqlite3 "$archivo" "SELECT count(*) FROM sqlite_master WHERE type IN ('table','view') AND name IN ('metadata','tiles');")"
if [[ "$tablas" != "2" ]]; then
  echo "ERROR: el MBTiles no expone metadata y tiles." >&2
  exit 1
fi

formato="$(sqlite3 "$archivo" "SELECT lower(value) FROM metadata WHERE name='format' LIMIT 1;")"
if [[ "$formato" != "pbf" && "$formato" != "mvt" ]]; then
  echo "ERROR: formato vectorial inesperado en metadata: $formato" >&2
  exit 1
fi

IFS='|' read -r minimo maximo cantidad <<<"$(sqlite3 "$archivo" 'SELECT min(zoom_level), max(zoom_level), count(*) FROM tiles;')"
if [[ -z "$minimo" || -z "$maximo" || "$cantidad" -lt 1 || "$minimo" -lt 0 || "$maximo" -gt 14 ]]; then
  echo "ERROR: niveles o cantidad de teselas fuera del contrato: $minimo|$maximo|$cantidad" >&2
  exit 1
fi

# Granada capital: XYZ z8/125/99 equivale a fila TMS 156 en MBTiles.
cobertura_granada="$(sqlite3 "$archivo" 'SELECT count(*) FROM tiles WHERE zoom_level=8 AND tile_column=125 AND tile_row=156;')"
if [[ "$cobertura_granada" -lt 1 ]]; then
  echo "ERROR: el artefacto no contiene la tesela semilla de Granada." >&2
  exit 1
fi

echo "OK: MBTiles valido ($cantidad teselas, zoom $minimo-$maximo)."
