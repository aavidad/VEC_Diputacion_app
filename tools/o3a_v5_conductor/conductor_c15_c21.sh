#!/usr/bin/env bash
set -euo pipefail

target=${1:-/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/o3a-v5-auditoria-20260809}
runtime_target=${CND_RUNTIME_TARGET:-$target}
salida=${2:-$PWD/evidencia_c15_c21.tsv}
base="$target/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql"
prefijo=supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1
raiz=$(mktemp -d "${TMPDIR:-/var/tmp}/o3a-c15-c21.XXXXXX")
trap 'rm -rf -- "$raiz"' EXIT
mapfile -t fuentes < <(find "$base" -maxdepth 1 -name "$prefijo*.go" -print | sort)
[[ -z $(gofmt -d "${fuentes[@]}") ]] || { printf 'fuentes_no_formateadas\n' >&2; exit 2; }
go vet "${fuentes[@]}"
opciones_build=()
[[ ${CND_RACE:-0} == 1 ]] && opciones_build=(-race)
CGO_ENABLED=${CND_CGO_ENABLED:-0} go build "${opciones_build[@]}" -trimpath -o "$raiz/supervisor" "${fuentes[@]}"
printf 'caso\tesperado\testado\tstdout\tstderr\tresultado\n' >"$salida"

ejecutar() {
  local caso=$1 esperado=$2 estado resultado=GO
  set +e
  (cd "$runtime_target" && timeout 90s "$raiz/supervisor" --autoprueba-o3a-caso "$caso") >"$raiz/o" 2>"$raiz/e"
  estado=$?
  set -e
  local stdout stderr
  stdout=$(wc -c <"$raiz/o"); stderr=$(wc -c <"$raiz/e")
  [[ $estado -eq $esperado && $stdout -eq 0 && $stderr -eq 0 ]] || resultado=NO-GO
  printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$caso" "$esperado" "$estado" "$stdout" "$stderr" "$resultado" >>"$salida"
  [[ $resultado == GO ]]
}

ejecutar C15_AF 65
ejecutar C16_ALIAS 0
ejecutar C17_TESTIGOS_TID 0
ejecutar C18_BORDES 0
ejecutar C21_CIEN_INVENTARIOS 0
