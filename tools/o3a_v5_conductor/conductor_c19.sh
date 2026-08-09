#!/usr/bin/env bash
set -euo pipefail

target=${1:-/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/o3a-v5-auditoria-20260809}
runtime_target=${CND_RUNTIME_TARGET:-$target}
salida=${2:-$PWD/evidencia_c19.tsv}
base="$target/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql"
prefijo=supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1
raiz=$(mktemp -d "${TMPDIR:-/var/tmp}/o3a-c19.XXXXXX")
trap 'rm -rf -- "$raiz"' EXIT

transformar() {
  local fichero=$1 viejo=$2 nuevo=$3
  local temporal="$1.nuevo"
  awk -v viejo="$viejo" -v nuevo="$nuevo" '$0==viejo{print nuevo;n++;next}{print}END{if(n!=1)exit 42}' "$fichero" >"$temporal"
  mv "$temporal" "$fichero"
}
construir() {
  local variante=$1 viejo=$2 nuevo=$3
  local dir="$raiz/$1"
  mkdir "$dir"; cp "$base"/"$prefijo"*.go "$dir/"
  transformar "$dir/${prefijo}_arranque_inicio.go" "$viejo" "$nuevo"
  mapfile -t fuentes < <(find "$dir" -maxdepth 1 -name "$prefijo*.go" -print | sort)
	gofmt -w "${fuentes[@]}"; go vet "${fuentes[@]}"
	local opciones=()
	[[ ${CND_RACE:-0} == 1 ]] && opciones=(-race)
	CGO_ENABLED=${CND_CGO_ENABLED:-0} go build "${opciones[@]}" -trimpath -o "$dir/supervisor" "${fuentes[@]}"
}
ejecutar() {
  local variante=$1 caso=$2 estado resultado=GO
  set +e
  (cd "$runtime_target" && timeout 10s "$raiz/$variante/supervisor" --autoprueba-o3a-caso "$caso") >"$raiz/o" 2>"$raiz/e"
  estado=$?; set -e
  local stdout stderr
  stdout=$(wc -c <"$raiz/o"); stderr=$(wc -c <"$raiz/e")
  [[ $estado -eq 73 && $stdout -eq 0 && $stderr -eq 0 ]] || resultado=NO-GO
  printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$caso" 73 "$estado" "$stdout" "$stderr" "$resultado" >>"$salida"
  [[ $resultado == GO ]]
}

lectura='\t\tn, err := syscall.Read(int(c.controlFD.Fd()), buffer[:])'
contadores='\tlecturas, total, interrupciones := 0, 0, 0'
poll='\tvivoPrimario, errPrimario := pidfdVivoO3aM38(c.pidfdPrimario)'
printf 'caso\tesperado\testado\tstdout\tstderr\tresultado\n' >"$salida"
construir eintr "$lectura" '\t\tn, err := syscall.Read(int(c.controlFD.Fd()), buffer[:])\n\t\tif c.cmd.Process != nil { n, err = 0, syscall.EINTR }'
ejecutar eintr C19_EINTR
construir presupuesto "$contadores" '\tlecturas, total, interrupciones := 0, 0, 0\n\tif c.cmd.Process != nil { lecturas = 4 }'
ejecutar presupuesto C19_PRESUPUESTO
construir poll "$poll" '\tvivoPrimario, errPrimario := pidfdVivoO3aM38(c.pidfdPrimario)\n\tif c.cmd.Process != nil { vivoPrimario, errPrimario = false, syscall.EINVAL }'
ejecutar poll C19_POLL
