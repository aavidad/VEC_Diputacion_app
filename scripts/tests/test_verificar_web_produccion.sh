#!/bin/sh
set -eu

cd "$(dirname "$0")/../.."

temporal="$(mktemp -d)"
limpiar() { rm -rf "$temporal"; }
trap limpiar EXIT INT TERM

crear_arbol_valido() {
  rm -rf "$temporal/web"
  mkdir -p \
    "$temporal/web/static/area-personal" \
    "$temporal/web/static/bolsa" \
    "$temporal/web/static/portal-empleado" \
    "$temporal/web/static/verificar"
  printf '%s\n' 'body { color: #111; }' >"$temporal/web/static/styles.css"
  printf '%s\n' '<svg xmlns="http://www.w3.org/2000/svg"/>' >"$temporal/web/static/favicon.svg"
  printf '%s\n' 'export const iniciar = true;' >"$temporal/web/static/bolsa/bolsa.js"
  printf '%s\n' \
    produccion.manifest \
    static/bolsa/bolsa.js \
    static/favicon.svg \
    static/styles.css >"$temporal/manifiesto"
  cp "$temporal/manifiesto" "$temporal/web/produccion.manifest"
}

debe_fallar() {
  if scripts/verificar_web_produccion.sh "$temporal/web" "$temporal/manifiesto" >/dev/null 2>&1; then
    echo "ERROR: el verificador acepto $1" >&2
    exit 1
  fi
}

crear_arbol_valido
scripts/verificar_web_produccion.sh "$temporal/web" "$temporal/manifiesto" >/dev/null

printf '%s\n' 'window.localStorage.setItem("sesion", "valor");' >"$temporal/web/static/bolsa/estado.js"
debe_fallar "localStorage"

crear_arbol_valido
printf '%s\n' 'NIF-DEMO-0001' >"$temporal/web/static/bolsa/estado.js"
debe_fallar "contenido plausible de la SPA historica"

crear_arbol_valido
printf '%s\n' 'id,nombre' >"$temporal/web/static/bolsa/personas.csv"
debe_fallar "un fichero de datos plausible"

crear_arbol_valido
printf '%s\n' '<html></html>' >"$temporal/web/static/index.html"
debe_fallar "la raiz historica"

crear_arbol_valido
mkdir -p "$temporal/web/static/modulo-no-enumerado"
printf '%s\n' 'export const neutral = true;' >"$temporal/web/static/modulo-no-enumerado/neutral.js"
debe_fallar "una superficie no enumerada"

crear_arbol_valido
printf '%s\n' 'export const neutral = true;' >"$temporal/web/static/bolsa/neutral.js"
debe_fallar "un JavaScript neutral no enumerado"

echo "Verificador del arbol web productivo probado."
