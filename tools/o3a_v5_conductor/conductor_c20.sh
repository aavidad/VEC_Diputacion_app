#!/usr/bin/env bash
set -euo pipefail

target=${1:-/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/o3a-v5-auditoria-20260809}
runtime_target=${CND_RUNTIME_TARGET:-$target}
salida=${2:-$PWD/evidencia_c20.tsv}
base="$target/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql"
prefijo=supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1
raiz=$(mktemp -d "${TMPDIR:-/var/tmp}/o3a-c20.XXXXXX")
trap 'rm -rf -- "$raiz"' EXIT

construir() {
  local variante=$1 dir="$raiz/$1"
  mkdir "$dir"
  cp "$base"/"$prefijo"*.go "$dir/"
  if [[ $variante == m18 ]]; then
    local fichero="$dir/${prefijo}_arranque_preparacion.go" temporal="$dir/m18.nuevo"
    awk '{ linea=$0; sub("Pdeathsig: syscall.SIGKILL", "Pdeathsig: 0", linea); if(linea!=$0)n++; print linea } END{if(n!=1)exit 42}' "$fichero" >"$temporal"
    mv "$temporal" "$fichero"
  fi
  mapfile -t fuentes < <(find "$dir" -maxdepth 1 -name "$prefijo*.go" -print | sort)
	gofmt -w "${fuentes[@]}"; go vet "${fuentes[@]}"
	local opciones=()
	[[ ${CND_RACE:-0} == 1 ]] && opciones=(-race)
	CGO_ENABLED=${CND_CGO_ENABLED:-0} go build "${opciones[@]}" -trimpath -o "$dir/supervisor" "${fuentes[@]}"
}

[[ -z $(gofmt -d abuelo_c20.go) ]] || { printf 'abuelo_no_formateado\n' >&2; exit 2; }
GOFLAGS='' go vet abuelo_c20.go
GOFLAGS='' CGO_ENABLED=0 go build -trimpath -o "$raiz/abuelo" abuelo_c20.go
printf 'variante\tcaso\testado\tstdout\tstderr\treap\tdelta_fd\tgrupo_cero\tresultado\n' >"$salida"
construir canonica
construir m18
{
  (cd "$runtime_target" && "$raiz/abuelo" "$raiz/canonica/supervisor" C20_PDEATH_CREADOR 74 1 canonica)
  (cd "$runtime_target" && "$raiz/abuelo" "$raiz/canonica/supervisor" C20_PDEATH_OTRO 75 0 canonica)
  (cd "$runtime_target" && "$raiz/abuelo" "$raiz/m18/supervisor" C20_PDEATH_CREADOR 76 1 m18_sin_pdeathsig)
} >>"$salida"
