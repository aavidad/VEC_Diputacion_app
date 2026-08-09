#!/usr/bin/env bash
set -euo pipefail

target=${1:-/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/o3a-v5-auditoria-20260809}
runtime_target=${CND_RUNTIME_TARGET:-$target}
salida=${2:-$PWD/evidencia_c01_c07.tsv}
base="$target/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql"
prefijo=supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1
raiz=$(mktemp -d "${TMPDIR:-/var/tmp}/o3a-c01-c07.XXXXXX")
trap 'rm -rf -- "$raiz"' EXIT
mapfile -t fuentes < <(find "$base" -maxdepth 1 -name "$prefijo*.go" -print | sort)
[[ -z $(gofmt -d "${fuentes[@]}") ]] || { printf 'fuentes_no_formateadas\n' >&2; exit 2; }
go vet "${fuentes[@]}"
opciones_build=()
[[ ${CND_RACE:-0} == 1 ]] && opciones_build=(-race)
CGO_ENABLED=${CND_CGO_ENABLED:-0} go build "${opciones_build[@]}" -trimpath -o "$raiz/supervisor" "${fuentes[@]}"
printf 'caso\testado\tstdout\tstderr\tresultado\n' >"$salida"

for numero in 01 02 03 04 05 06 07; do
  case $numero in
    01) id=C01_PREFLIGHT ;;
    02) id=C02_SENAL_HANDOFF ;;
    03) id=C03_SELECTORES ;;
    04) id=C04_MAPA_HERENCIA ;;
    05) id=C05_LEASE_COPIAS ;;
    06) id=C06_TERMINAL ;;
    07) id=C07_TICKET_EOF ;;
  esac
  set +e
  if [[ $numero == 04 ]]; then
    (cd "$runtime_target" && timeout 30s "$raiz/supervisor" --autoprueba-o3a-caso "$id" 3</dev/null 4</dev/null 5</dev/null 6</dev/null 7</dev/null 8</dev/null 9</dev/null) >"$raiz/o" 2>"$raiz/e"
  else
    (cd "$runtime_target" && timeout 30s "$raiz/supervisor" --autoprueba-o3a-caso "$id") >"$raiz/o" 2>"$raiz/e"
  fi
  estado=$?; set -e
  stdout=$(wc -c <"$raiz/o"); stderr=$(wc -c <"$raiz/e"); resultado=GO
  [[ $estado -eq 0 && $stdout -eq 0 && $stderr -eq 0 ]] || resultado=NO-GO
  printf '%s\t%s\t%s\t%s\t%s\n' "$id" "$estado" "$stdout" "$stderr" "$resultado" >>"$salida"
  [[ $resultado == GO ]]
done
