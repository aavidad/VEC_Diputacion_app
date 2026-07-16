#!/usr/bin/env bash
# Comprueba el tamano de los ficheros de codigo (auditoria 2026-07-16,
# hallazgo H-06). Dos niveles:
#   - Objetivo de diseno: 500 lineas. No lo vigila esta puerta; es la
#     referencia al escribir ficheros nuevos.
#   - Tope duro: 800 lineas. Un fichero cohesionado puede superar el
#     objetivo con margen, pero por encima del tope un agente ya no puede
#     leerlo entero sin hipotecar su contexto y debe partirse.
# Los ficheros que ya superaban el tope estan congelados en la linea base
# scripts/tamano_ficheros_base.txt: no pueden crecer y la linea base solo
# puede menguar; nunca se regenera para dar cabida a crecimiento nuevo.

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

LIMITE=800
BASE="scripts/tamano_ficheros_base.txt"

declare -A permitidas
if [[ -f "${BASE}" ]]; then
  while read -r lineas fichero; do
    [[ -n "${fichero:-}" ]] && permitidas["${fichero}"]="${lineas}"
  done < "${BASE}"
fi

fallos=0
while IFS= read -r fichero; do
  [[ -f "${fichero}" ]] || continue
  lineas=$(wc -l < "${fichero}")
  if (( lineas > LIMITE )); then
    tope="${permitidas[${fichero}]:-0}"
    if (( lineas > tope )); then
      printf '%s: %d lineas (limite %d, linea base %d)\n' \
        "${fichero}" "${lineas}" "${LIMITE}" "${tope}" >&2
      fallos=1
    fi
  fi
done < <(git ls-files -co --exclude-standard -- cmd config internal scripts web \
  | grep -E '\.(go|js|py|sh|css)$')

if (( fallos )); then
  cat >&2 <<'MENSAJE'
Hay ficheros de codigo por encima del tope duro de 800 lineas.
El objetivo de diseno son 500 lineas; hasta 800 hay margen para ficheros
cohesionados. Por encima del tope hay que trocear antes de ampliar (en Go,
dividir un fichero en varios del mismo paquete conserva API y
comportamiento). Vease
docs/portal_vec/auditoria_diseno_y_seguridad_2026-07-16.md, directriz 9.
MENSAJE
  exit 1
fi

printf 'Tamano de ficheros dentro del limite.\n'
