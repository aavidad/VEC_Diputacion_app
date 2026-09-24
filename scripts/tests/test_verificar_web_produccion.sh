#!/bin/sh
set -eu

cd "$(dirname "$0")/../.."

temporal="$(mktemp -d)"
limpiar() { rm -rf "$temporal"; }
trap limpiar EXIT INT TERM

crear_arbol_valido() {
  rm -rf "$temporal/web"
  mkdir -p \
    "$temporal/web/static/acceso/locales" \
    "$temporal/web/static/assets" \
    "$temporal/web/static/area-personal/locales" \
    "$temporal/web/static/bolsa" \
    "$temporal/web/static/comun/oportunidades" \
    "$temporal/web/static/portal-empleado/modulos/contratacion-temporal" \
    "$temporal/web/static/verificar"
  printf '%s\n' 'body { color: #111; }' >"$temporal/web/static/styles.css"
  printf '%s\n' '<!doctype html><html lang="es"></html>' >"$temporal/web/static/acceso/index.html"
  printf '%s\n' 'body { color: #111; }' >"$temporal/web/static/acceso/acceso.css"
  printf '%s\n' 'export const acceso = true;' >"$temporal/web/static/acceso/acceso-i18n.js"
  printf '%s\n' '{"acceso":"Acceso"}' >"$temporal/web/static/acceso/locales/es.json"
  printf '%s\n' '{"areaPersonal.rutas.inicio":"Inicio y plazos"}' >"$temporal/web/static/area-personal/locales/es.json"
  printf '%s\n' '<svg xmlns="http://www.w3.org/2000/svg"/>' >"$temporal/web/static/favicon.svg"
  printf '%s\n' '<svg xmlns="http://www.w3.org/2000/svg"/>' >"$temporal/web/static/assets/logo-diputacion-granada.svg"
  printf '%s\n' 'export const iniciar = true;' >"$temporal/web/static/bolsa/bolsa.js"
  printf '%s\n' 'body { color: #111; }' >"$temporal/web/static/comun/tema-vec.css"
  printf '%s\n' 'export const tema = true;' >"$temporal/web/static/comun/tema-vec.js"
  printf '%s\n' 'export const vista = true;' >"$temporal/web/static/comun/oportunidades/vista.js"
  printf '%s\n' 'export const i18n = true;' >"$temporal/web/static/comun/oportunidades/i18n.js"
  printf '%s\n' 'body { color: #111; }' >"$temporal/web/static/comun/oportunidades/oportunidades.css"
  cp web/static/portal-empleado/modulos/contratacion-temporal/formalizacion-desarrollo.json \
    "$temporal/web/static/portal-empleado/modulos/contratacion-temporal/formalizacion-desarrollo.json"
  printf '%s\n' \
    produccion.manifest \
    static/acceso/index.html \
    static/acceso/acceso.css \
    static/acceso/acceso-i18n.js \
    static/acceso/locales/es.json \
    static/area-personal/locales/es.json \
    static/assets/logo-diputacion-granada.svg \
    static/bolsa/bolsa.js \
    static/comun/tema-vec.css \
    static/comun/tema-vec.js \
    static/comun/oportunidades/vista.js \
    static/comun/oportunidades/i18n.js \
    static/comun/oportunidades/oportunidades.css \
    static/favicon.svg \
    static/portal-empleado/modulos/contratacion-temporal/formalizacion-desarrollo.json \
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

for extension in css js; do
  crear_arbol_valido
  printf '%s\n' 'contenido ajeno' >"$temporal/web/static/comun/otro.$extension"
  printf '%s\n' "static/comun/otro.$extension" >>"$temporal/manifiesto"
  cp "$temporal/manifiesto" "$temporal/web/produccion.manifest"
  debe_fallar "otro activo comun .$extension enumerado"
done

crear_arbol_valido
printf '%s\n' 'export const extra = true;' >"$temporal/web/static/comun/oportunidades/extra.js"
printf '%s\n' 'static/comun/oportunidades/extra.js' >>"$temporal/manifiesto"
cp "$temporal/manifiesto" "$temporal/web/produccion.manifest"
debe_fallar "un cuarto archivo de oportunidades enumerado"

crear_arbol_valido
printf '%s\n' 'export const extra = true;' >"$temporal/web/static/comun/oportunidades/extra.js"
debe_fallar "un cuarto archivo de oportunidades sin enumerar"

crear_arbol_valido
printf '%s\n' 'export const extra = true;' >"$temporal/web/static/acceso/extra.js"
printf '%s\n' 'static/acceso/extra.js' >>"$temporal/manifiesto"
cp "$temporal/manifiesto" "$temporal/web/produccion.manifest"
debe_fallar "otro activo de acceso enumerado"

crear_arbol_valido
printf '%s\n' '{"otro":"No autorizado"}' >"$temporal/web/static/acceso/locales/otro.json"
printf '%s\n' 'static/acceso/locales/otro.json' >>"$temporal/manifiesto"
cp "$temporal/manifiesto" "$temporal/web/produccion.manifest"
debe_fallar "otro catalogo de acceso enumerado"

crear_arbol_valido
printf '%s\n' '<svg xmlns="http://www.w3.org/2000/svg"/>' >"$temporal/web/static/assets/logo-no-autorizado.svg"
printf '%s\n' 'static/assets/logo-no-autorizado.svg' >>"$temporal/manifiesto"
cp "$temporal/manifiesto" "$temporal/web/produccion.manifest"
debe_fallar "un activo compartido no autorizado"

crear_arbol_valido
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

# Enumerarlos no autoriza otros JSON ni alias de los catálogos permitidos.
for ruta_json in \
  static/area-personal/locales/otro.json \
  static/area-personal/locales/es.JSON \
  static/area-personal/otro/es.json \
  static/portal-empleado/modulos/contratacion-temporal/otra.json \
  static/portal-empleado/modulos/contratacion-temporal/formalizacion-desarrollo.JSON \
  static/bolsa/formalizacion-desarrollo.json; do
  crear_arbol_valido
  mkdir -p "$(dirname "$temporal/web/$ruta_json")"
  cp web/static/portal-empleado/modulos/contratacion-temporal/formalizacion-desarrollo.json "$temporal/web/$ruta_json"
  printf '%s\n' "$ruta_json" >>"$temporal/manifiesto"
  cp "$temporal/manifiesto" "$temporal/web/produccion.manifest"
  debe_fallar "el JSON no autorizado $ruta_json"
done

echo "Verificador del arbol web productivo probado."
