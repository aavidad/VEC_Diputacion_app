#!/usr/bin/env bash
# Comprueba el limite de tamano de ficheros de codigo (auditoria 2026-07-16,
# hallazgo H-06): ningun fichero nuevo puede superar las 500 lineas y los que
# ya las superaban no pueden crecer. La linea base congelada vive en
# scripts/tamano_ficheros_base.txt y solo puede menguar: se regenera
# exclusivamente al partir o reducir ficheros, nunca para dar cabida a
# crecimiento nuevo.

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

LIMITE=500
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
Hay ficheros de codigo por encima del limite de 500 lineas.
Los ficheros nuevos deben nacer por debajo del limite y los antiguos no
pueden crecer: hay que trocearlos antes de ampliarlos (en Go, dividir un
fichero en varios del mismo paquete conserva API y comportamiento). Vease
docs/portal_vec/auditoria_diseno_y_seguridad_2026-07-16.md, directriz 9.
MENSAJE
  exit 1
fi

printf 'Tamano de ficheros dentro del limite.\n'
