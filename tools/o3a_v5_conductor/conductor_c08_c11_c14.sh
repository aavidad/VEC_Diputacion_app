#!/usr/bin/env bash
set -euo pipefail

target=${1:-/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/o3a-v5-auditoria-20260809}
runtime_target=${CND_RUNTIME_TARGET:-$target}
salida=${2:-$PWD/evidencia_c08_c11_c14.tsv}
base="$target/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql"
prefijo=supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1
raiz=$(mktemp -d "${TMPDIR:-/var/tmp}/o3a-c08-c14.XXXXXX")
trap 'rm -rf -- "$raiz"' EXIT

copiar() { mkdir "$raiz/$1"; cp "$base"/"$prefijo"*.go "$raiz/$1/"; }
transformar() {
  local fichero=$1 viejo=$2 nuevo=$3 temporal="$1.nuevo"
  awk -v viejo="$viejo" -v nuevo="$nuevo" '$0==viejo{print nuevo;n++;next}{print}END{if(n!=1)exit 42}' "$fichero" >"$temporal"
  mv "$temporal" "$fichero"
}
construir() {
  local dir="$raiz/$1"
  mapfile -t fuentes < <(find "$dir" -maxdepth 1 -name "$prefijo*.go" -print | sort)
	gofmt -w "${fuentes[@]}"; go vet "${fuentes[@]}"
	local opciones=()
	[[ ${CND_RACE:-0} == 1 ]] && opciones=(-race)
	CGO_ENABLED=${CND_CGO_ENABLED:-0} go build "${opciones[@]}" -trimpath -o "$dir/supervisor" "${fuentes[@]}"
}
registrar() {
  local variante=$1 caso=$2 esperado=$3 modo=${4:-normal} estado stdout stderr resultado=GO
  set +e
  if [[ $modo == low ]]; then
    (cd "$runtime_target" && "$raiz/$variante/supervisor" --autoprueba-o3a-caso "$caso" 3</dev/null 4</dev/null 5</dev/null 6</dev/null 7</dev/null 8</dev/null 9</dev/null) >"$raiz/o" 2>"$raiz/e"
  else
    (cd "$runtime_target" && "$raiz/$variante/supervisor" --autoprueba-o3a-caso "$caso") >"$raiz/o" 2>"$raiz/e"
  fi
  estado=$?; set -e
  stdout=$(wc -c <"$raiz/o"); stderr=$(wc -c <"$raiz/e")
  [[ $estado -eq $esperado && $stdout -eq 0 && $stderr -eq 0 ]] || resultado=NO-GO
  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$variante" "$caso" "$esperado" "$estado" "$stdout" "$stderr" "$resultado" >>"$salida"
  [[ $resultado == GO ]]
}

snapshot='\tconPidfd, errInventario := snapshotActualO3aM38()'
barrera='\tif err = barreraDespuesStartO3aM38(c); err != nil {'
extra='\t\tExtraFiles:  []*os.File{destinados[5], destinados[6], destinados[7], destinados[8], raiz, runner, ticketLector},'
barrido='\tfor fd := 0; fd < int(limite.Cur); fd++ {'
printf 'variante\tcaso\tesperado\testado\tstdout\tstderr\tresultado\n' >"$salida"
fallos=0

copiar low_hole; construir low_hole
registrar low_hole LOW_HOLE 0 low || fallos=1

copiar barrido
transformar "$raiz/barrido/${prefijo}_arranque_preparacion.go" "$barrido" '\tfor fd := minFDDuplicadoM38; fd < int(limite.Cur); fd++ {'
construir barrido
registrar barrido C08_BARRIDO 66 low || fallos=1

copiar cuarta
transformar "$raiz/cuarta/${prefijo}_arranque_inicio.go" "$snapshot" '\tif _, _, fallo := syscall.Syscall(syscall.SYS_FCNTL, uintptr(pidfdPrimario), syscall.F_DUPFD_CLOEXEC, minFDDuplicadoM38); fallo != 0 { fatalO3aM38() }\n\tconPidfd, errInventario := snapshotActualO3aM38()'
construir cuarta; registrar cuarta C08_CUARTA 65 || fallos=1

copiar flag
transformar "$raiz/flag/${prefijo}_arranque_inicio.go" "$snapshot" '\tif _, _, fallo := syscall.Syscall(syscall.SYS_FCNTL, uintptr(pidfdPrimario), syscall.F_SETFD, 0); fallo != 0 { fatalO3aM38() }\n\tconPidfd, errInventario := snapshotActualO3aM38()'
construir flag; registrar flag C08_FLAG 65 || fallos=1

copiar kill
transformar "$raiz/kill/${prefijo}_arranque_inicio.go" "$barrera" '\tif fallo := enviarPidfdIndividualO3aM38(c.pidfdPrimario, syscall.SIGKILL); fallo != nil || !terminalAntesO3aM38(c.pidfdPrimario, time.Now().Add(time.Second)) { fatalO3aM38() }\n\tif err = barreraDespuesStartO3aM38(c); err != nil {'
construir kill
registrar kill C11_TERMINAL_POST_START 73 || fallos=1
registrar kill C14_KILL 73 || fallos=1

copiar eof
transformar "$raiz/eof/${prefijo}_arranque_preparacion.go" "$extra" '\t\tExtraFiles:  []*os.File{destinados[5], destinados[6], destinados[7], destinados[8], raiz, runner, destinados[4]},'
transformar "$raiz/eof/${prefijo}_arranque_inicio.go" "$barrera" '\tif !terminalAntesO3aM38(c.pidfdPrimario, time.Now().Add(time.Second)) { fatalO3aM38() }\n\tif err = barreraDespuesStartO3aM38(c); err != nil {'
construir eof; registrar eof C14_EOF 73 || fallos=1

copiar cancelar_terminal
transformar "$raiz/cancelar_terminal/${prefijo}_arranque_inicio.go" "$barrera" '\tdatosControl := []byte("V1|CONTROL|CANCELAR|" + string(c.control.nonce[:]) + "|CANCELADO|65\\n")\n\tif escritos, fallo := syscall.Write(int(c.controlFD.Fd())+1, datosControl); fallo != nil || escritos != len(datosControl) { fatalO3aM38() }\n\tif fallo := enviarPidfdIndividualO3aM38(c.pidfdPrimario, syscall.SIGKILL); fallo != nil || !terminalAntesO3aM38(c.pidfdPrimario, time.Now().Add(time.Second)) { fatalO3aM38() }\n\tif err = barreraDespuesStartO3aM38(c); err != nil {'
construir cancelar_terminal; registrar cancelar_terminal C12_CANCELAR_TERMINAL 73 || fallos=1

copiar eof_terminal
transformar "$raiz/eof_terminal/${prefijo}_arranque_inicio.go" "$barrera" '\tfdControlEscritor := int(c.controlFD.Fd()) + 1\n\tpermisoControl, autorizadoControl := c.lease.comenzar(operacionCerrarDestinosO3aM38, 1, [2]int{fdControlEscritor, -1})\n\tif !autorizadoControl || syscall.Close(fdControlEscritor) != nil { fatalO3aM38() }\n\tsnapshotControl, falloSnapshotControl := snapshotActualO3aM38()\n\taplicadoControl, consolidadoControl := consolidarOperacionFisicaO3aM38(c.lease, permisoControl, snapshotControl)\n\tif falloSnapshotControl != nil || !aplicadoControl || !consolidadoControl { fatalO3aM38() }\n\tif fallo := enviarPidfdIndividualO3aM38(c.pidfdPrimario, syscall.SIGKILL); fallo != nil || !terminalAntesO3aM38(c.pidfdPrimario, time.Now().Add(time.Second)) { fatalO3aM38() }\n\tif err = barreraDespuesStartO3aM38(c); err != nil {'
construir eof_terminal; registrar eof_terminal C12_EOF_TERMINAL 73 || fallos=1

exit "$fallos"
