#!/usr/bin/env bash
set -euo pipefail
target=${1:-/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/o3a-v5-auditoria-20260809}; salida=${2:-$PWD/evidencia_c02_interleavings.tsv}
runtime_target=${CND_RUNTIME_TARGET:-$target}
base="$target/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql"; prefijo=supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1
raiz=$(mktemp -d "${TMPDIR:-/var/tmp}/o3a-c02.XXXXXX"); trap 'rm -rf -- "$raiz"' EXIT
transformar() { local f=$1 v=$2 n=$3 t="$1.n"; awk -v v="$v" -v n="$n" '$0==v{print n;c++;next}{print}END{if(c!=1)exit 42}' "$f" >"$t"; mv "$t" "$f"; }
caso() {
  local id=$2 viejo=$3 nuevo=$4 estado resultado=GO so se
  local dir="$raiz/$1"
  mkdir "$dir"; cp "$base"/"$prefijo"*.go "$dir/"; transformar "$dir/${prefijo}_arranque_inicio.go" "$viejo" "$nuevo"
  mapfile -t fs < <(find "$dir" -maxdepth 1 -name "$prefijo*.go" -print | sort); gofmt -w "${fs[@]}"; go vet "${fs[@]}"
  local opciones=()
  [[ ${CND_RACE:-0} == 1 ]] && opciones=(-race)
  CGO_ENABLED=${CND_CGO_ENABLED:-0} go build "${opciones[@]}" -trimpath -o "$dir/s" "${fs[@]}"
  set +e; (cd "$runtime_target" && timeout 10s "$dir/s" --autoprueba-o3a-caso "$id") >"$dir/o" 2>"$dir/e"; estado=$?; set -e
  so=$(wc -c <"$dir/o"); se=$(wc -c <"$dir/e"); [[ $estado -eq 73 && $so -eq 0 && $se -eq 0 ]] || resultado=NO-GO
  printf '%s\t%s\t%s\t%s\t%s\n' "$id" "$estado" "$so" "$se" "$resultado" >>"$salida"; [[ $resultado == GO ]]
}
printf 'caso\testado\tstdout\tstderr\tresultado\n' >"$salida"
start='\terrStart := c.cmd.Start()'; handoff='\tnuevoBaseline, observadorTransferido := c.observador.transferirCritico(c.baselineSenal)'
caso prestart C02_SENAL_PRESTART "$start" '\tif !c.observador.anotar(syscall.SIGUSR1) { fatalO3aM38() }\n\terrStart := c.cmd.Start()'
caso poststart C02_SENAL_POSTSTART "$start" '\terrStart := c.cmd.Start()\n\tif !c.observador.anotar(syscall.SIGUSR1) { fatalO3aM38() }'
caso prehandoff C02_SENAL_PREHANDOFF "$handoff" '\tif !c.observador.anotar(syscall.SIGUSR1) { fatalO3aM38() }\n\tnuevoBaseline, observadorTransferido := c.observador.transferirCritico(c.baselineSenal)'
