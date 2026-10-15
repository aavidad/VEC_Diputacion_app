#!/usr/bin/env bash
set -euo pipefail
umask 077

carpeta=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
raiz=$(cd -- "$carpeta/../../.." && pwd)
case ${LC_ALL:-${LANG:-es}} in en*) mensajes=$carpeta/mensajes.en.json ;; *) mensajes=$carpeta/mensajes.es.json ;; esac
mensaje() { jq -r --arg clave "$1" '.[$clave] // $clave' "$mensajes"; }
rechazar() { mensaje "$1" >&2; exit 1; }
case "${1:-}" in
  --comprobar) [[ $# == 1 ]] || rechazar uso ;;
  --aplicar) [[ $# == 2 && ${2:-} == PUBLICA-PLAZOS-CT-V3 ]] || rechazar uso ;;
  *) rechazar uso ;;
esac

fuente=$raiz/data/catalogos/contratacion-temporal/reglas.v3.json
huella=fadf23dcdcdcf857d327500bdf017dd2c2c91b490566a5ce68c1b915a7932dec
aprobacion=operacion:plazos-ct:20261015
[[ -f $fuente && ! -L $fuente && $(sha256sum -- "$fuente" | cut -d' ' -f1) == "$huella" ]] || rechazar fuente_modificada

# Reutiliza el cotejo de la base v2, sus fuentes y sus ajustes. Solo lectura.
estado=$(bash "$raiz/scripts/operacion/publicar_reglas_ct_limpias/publicar.sh" --comprobar)
jq -e '.resultado == "replay" and .version == 2 and .secuencia_actual == 2 and .ajustes == 0' \
  <<<"$estado" >/dev/null || rechazar base_incompatible
if [[ $1 == --comprobar ]]; then
  jq -nc --arg fuente "$huella" '{resultado:"preparado",version:3,fuente_sha256:$fuente,secuencia_esperada:2,ajustes:0}'
  exit 0
fi

# La admisión debe estar detenida hasta terminar la comprobación en navegador.
preparador=${VEC_REGLAS_CT_PREPARADOR:-}
[[ -n $preparador && -x $preparador ]] || rechazar preparador_ausente
privado=$(mktemp -d "${TMPDIR:-/var/tmp}/vec-plazos-ct-v3.XXXXXXXX")
trap 'rm -rf -- "$privado"' EXIT
cp -- "$fuente" "$privado/fuente.json"
[[ $(sha256sum -- "$privado/fuente.json" | cut -d' ' -f1) == "$huella" ]] || rechazar fuente_modificada
"$preparador" -fuente "$privado/fuente.json" -salida "$privado/canonico.json" \
  -manifiesto "$privado/manifiesto.json" -aprobacion-ref "$aprobacion" >/dev/null
VEC_CT190_ENTORNO=desarrollo VEC_CT190_APROBACION_REF="$aprobacion" \
  VEC_CT190_PSQL="${VEC_REGLAS_CT_PSQL:-psql}" \
  bash "$raiz/deploy/principal/publicar_catalogo_ct190.sh" \
  "$privado/fuente.json" "$privado/canonico.json" "$privado/manifiesto.json" 2
