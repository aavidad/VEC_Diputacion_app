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

ejecutar_binario() {
  local caso=$1 estado_var=$2 stdout_var=$3 stderr_var=$4 estado_real stdout_real stderr_real
  set +e
  (cd "$runtime_target" && timeout 90s "$raiz/supervisor" --autoprueba-o3a-caso "$caso") >"$raiz/o" 2>"$raiz/e"
  estado_real=$?
  set -e
  stdout_real=$(wc -c <"$raiz/o")
  stderr_real=$(wc -c <"$raiz/e")
  printf -v "$estado_var" '%d' "$estado_real"
  printf -v "$stdout_var" '%d' "$stdout_real"
  printf -v "$stderr_var" '%d' "$stderr_real"
}

ejecutar() {
  local caso=$1 esperado=$2 estado stdout stderr resultado=GO
  ejecutar_binario "$caso" estado stdout stderr
  [[ $estado -eq $esperado && $stdout -eq 0 && $stderr -eq 0 ]] || resultado=NO-GO
  printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$caso" "$esperado" "$estado" "$stdout" "$stderr" "$resultado" >>"$salida"
  [[ $resultado == GO ]]
}

ejecutar_c21() {
  local inventario=${salida%.tsv}_c21_indices.tsv indice estado stdout stderr resultado
  printf 'indice\tcaso\tesperado\testado\tstdout\tstderr\tresultado\n' >"$inventario"
  for ((indice = 1; indice <= 100; indice++)); do
    resultado=GO
    ejecutar_binario TUPLA_C estado stdout stderr
    [[ $estado -eq 0 && $stdout -eq 0 && $stderr -eq 0 ]] || resultado=NO-GO
    printf '%s\tTUPLA_C\t0\t%s\t%s\t%s\t%s\n' "$indice" "$estado" "$stdout" "$stderr" "$resultado" >>"$inventario"
    if [[ $resultado != GO ]]; then
      printf 'C21_CIEN_INVENTARIOS\t0\t%s\t%s\t%s\tNO-GO\n' "$estado" "$stdout" "$stderr" >>"$salida"
      return 1
    fi
  done
  printf 'C21_CIEN_INVENTARIOS\t0\t0\t0\t0\tGO\n' >>"$salida"
}

ejecutar C15_AF 65
ejecutar C16_ALIAS 0
ejecutar C17_TESTIGOS_TID 0
ejecutar C18_BORDES 0
ejecutar_c21
