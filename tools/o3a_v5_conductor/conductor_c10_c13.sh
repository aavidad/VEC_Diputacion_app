#!/usr/bin/env bash
set -euo pipefail

target=${1:-/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/o3a-v5-auditoria-20260809}
runtime_target=${CND_RUNTIME_TARGET:-$target}
base="$target/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql"
salida=${2:-$PWD/evidencia_c10_c13.tsv}
raiz=$(mktemp -d "${TMPDIR:-/var/tmp}/o3a-c10-c13.XXXXXX")
trap 'rm -rf -- "$raiz"' EXIT
prefijo=supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1

copiar() {
  local variante=$1
  mkdir "$raiz/$variante"
  cp "$base"/"$prefijo"*.go "$raiz/$variante/"
}

transformar() {
  local fichero=$1 viejo=$2 nuevo=$3
  local temporal="$fichero.nuevo"
  awk -v viejo="$viejo" -v nuevo="$nuevo" '
    $0 == viejo { print nuevo; encontrados++; next }
    { print }
    END { if (encontrados != 1) exit 42 }
  ' "$fichero" >"$temporal"
  mv "$temporal" "$fichero"
}

construir() {
  local variante=$1
  local dir="$raiz/$variante"
  mapfile -t fuentes < <(find "$dir" -maxdepth 1 -name "$prefijo*.go" -print | sort)
  gofmt -w "${fuentes[@]}"
	go vet "${fuentes[@]}"
	local opciones=()
	[[ ${CND_RACE:-0} == 1 ]] && opciones=(-race)
	CGO_ENABLED=${CND_CGO_ENABLED:-0} go build "${opciones[@]}" -trimpath -o "$dir/supervisor" "${fuentes[@]}"
}

ejecutar() {
  local variante=$1 caso=$2 esperado=$3 estado
  local bin="$raiz/$variante/supervisor"
  set +e
  (cd "$runtime_target" && "$bin" --autoprueba-o3a-caso "$caso") >"$raiz/$variante.out" 2>"$raiz/$variante.err"
  estado=$?
  set -e
  local stdout stderr resultado=GO
  stdout=$(wc -c <"$raiz/$variante.out")
  stderr=$(wc -c <"$raiz/$variante.err")
  if [[ $estado -ne $esperado || $stdout -ne 0 || $stderr -ne 0 ]]; then
    resultado=NO-GO
  fi
  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$variante" "$caso" "$esperado" "$estado" "$stdout" "$stderr" "$resultado" >>"$salida"
  [[ $resultado == GO ]]
}

inicio='\terrStart := c.cmd.Start()'
duplicacion='\treserva, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(pidfdPrimario), syscall.F_DUPFD_CLOEXEC, minFDDuplicadoM38)'
barrera='\tif err = barreraDespuesStartO3aM38(c); err != nil {'

printf 'variante\tcaso\tesperado\testado\tstdout\tstderr\tresultado\n' >"$salida"
fallos=0

copiar canonica
construir canonica
ejecutar canonica TUPLA_C 0 || fallos=1

copiar tupla_a
transformar "$raiz/tupla_a/${prefijo}_arranque_inicio.go" "$inicio" '\terrStart := error(syscall.EBADF)'
construir tupla_a
ejecutar tupla_a TUPLA_A 72 || fallos=1

copiar tupla_b
transformar "$raiz/tupla_b/${prefijo}_arranque_inicio.go" "$inicio" '\terrStart := c.cmd.Start(); c.cmd.Process = nil'
construir tupla_b
ejecutar tupla_b TUPLA_B 65 || fallos=1

copiar dupfd
transformar "$raiz/dupfd/${prefijo}_arranque_inicio.go" "$duplicacion" '\treserva, errno := uintptr(0), syscall.EBADF'
construir dupfd
ejecutar dupfd C10_DUPFD_POST_START 73 || fallos=1

copiar primario_cerrado
transformar "$raiz/primario_cerrado/${prefijo}_arranque_inicio.go" "$barrera" '\tif cerrado, fallo := cerrarPidfdConLeaseO3aM38(c.lease, c.pidfdPrimario); !cerrado || fallo != nil { fatalO3aM38() }\n\tif err = barreraDespuesStartO3aM38(c); err != nil {'
construir primario_cerrado
ejecutar primario_cerrado AF_PIDFD_PRIMARIO_CERRADO 73 || fallos=1

copiar reserva_cerrada
transformar "$raiz/reserva_cerrada/${prefijo}_arranque_inicio.go" "$barrera" '\tif cerrado, fallo := cerrarPidfdConLeaseO3aM38(c.lease, c.pidfdReserva); !cerrado || fallo != nil { fatalO3aM38() }\n\tif err = barreraDespuesStartO3aM38(c); err != nil {'
construir reserva_cerrada
ejecutar reserva_cerrada AF_PIDFD_RESERVA_CERRADA 73 || fallos=1

copiar ambos_cerrados
transformar "$raiz/ambos_cerrados/${prefijo}_arranque_inicio.go" "$barrera" '\tif cerrado, fallo := cerrarPidfdConLeaseO3aM38(c.lease, c.pidfdPrimario); !cerrado || fallo != nil { fatalO3aM38() }\n\tif cerrado, fallo := cerrarPidfdConLeaseO3aM38(c.lease, c.pidfdReserva); !cerrado || fallo != nil { fatalO3aM38() }\n\tif err = barreraDespuesStartO3aM38(c); err != nil {'
construir ambos_cerrados
ejecutar ambos_cerrados AF_PIDFD_AMBOS_CERRADOS 65 || fallos=1

exit "$fallos"
