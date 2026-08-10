#!/usr/bin/bash
set -Eeuo pipefail

if (($# < 1)); then
    printf 'uso: %s RAIZ_PROYECCION [MUTANTE...]\n' "$0" >&2
    exit 64
fi
raiz=$1
ruta=deploy/postgresql/autorizacion_atestada_v3/pruebas_sql
origen=${raiz}/${ruta}
go_lanzador=${VEP_GO_BIN:-$(command -v go)}
[[ -x "$go_lanzador" ]] || { printf 'VEP_GO_BIN no ejecutable: %s\n' "$go_lanzador" >&2; exit 64; }
for variable in GOFLAGS GOENV GOTOOLCHAIN GOROOT GOTOOLDIR GOEXPERIMENT GOAMD64 GOARCH GOOS CGO_ENABLED CC CXX; do
    [[ -z "${!variable-}" ]] || { printf 'ambiente Go heredado prohibido: %s\n' "$variable" >&2; exit 64; }
done
control_home=${HOME:?HOME requerido}
toolchain_cache=${GOMODCACHE:-}
[[ "$toolchain_cache" == /* && -d "$toolchain_cache" ]] || { printf 'cache bootstrap de toolchain inválida\n' >&2; exit 64; }
goroot=$(env -i HOME="$control_home" PATH=/usr/bin:/bin GOENV=off GOFLAGS= GOTOOLCHAIN=go1.26.5 GOMODCACHE="$toolchain_cache" "$go_lanzador" env GOROOT)
[[ -n "$goroot" && "$goroot" == /* && -d "$goroot" ]] || { printf 'GOROOT inválido derivado de VEP_GO_BIN: %s\n' "$goroot" >&2; exit 64; }
go_bin=${goroot}/bin/go
[[ -x "$go_bin" ]] || { printf 'Go efectivo no ejecutable: %s\n' "$go_bin" >&2; exit 64; }
huella_go_inicial=$(sha256sum -- "$go_bin")
[[ "${huella_go_inicial%% *}" == "8da5fd321795754b994c64e3eb8a5a14ff47bd285559a7e876f3c79abafc67f9" ]] || { printf 'binario Go efectivo no autorizado\n' >&2; exit 65; }
[[ -z "${VEP_AST_BIN:-}${VEP_APLICADOR_BIN:-}" ]] || { printf 'binarios externos de herramientas prohibidos\n' >&2; exit 64; }
manifest=${VEP_MANIFEST:-}
ledger=${VEP_LEDGER:-}
base_sha=${VEP_BASE_SHA:-}
temporal=$(mktemp -d)
mkdir -p -- "$temporal/tmp" "$temporal/go-cache" "$temporal/go-mod"
go_controlado() {
    env -i HOME="$control_home" PATH=/usr/bin:/bin TMPDIR="$temporal/tmp" GOROOT="$goroot" GOENV=off GOFLAGS= GOTOOLCHAIN=local CGO_ENABLED=0 GOCACHE="$temporal/go-cache" GOMODCACHE="$temporal/go-mod" "$go_bin" "$@"
}
herramienta_controlada() {
    env -i HOME="$control_home" PATH=/usr/bin:/bin TMPDIR="$temporal/tmp" GOROOT="$goroot" GOENV=off GOFLAGS= GOTOOLCHAIN=local CGO_ENABLED=0 GOCACHE="$temporal/go-cache" GOMODCACHE="$temporal/go-mod" "$@"
}
go_version=$(go_controlado version)
[[ "$go_version" =~ ^go\ version\ go1\.26\.5\  ]] || { printf 'Go efectivo no autorizado: %s\n' "$go_version" >&2; exit 64; }
gotooldir=$(go_controlado env GOTOOLDIR)
[[ "$gotooldir" == "$goroot"/* && -d "$gotooldir" ]] || { printf 'GOTOOLDIR inválido: %s\n' "$gotooldir" >&2; exit 65; }
huella_gotool_inicial=$(while IFS= read -r herramienta; do huella=$(sha256sum -- "$gotooldir/$herramienta"); printf '%s\t%s\n' "${huella%% *}" "$herramienta"; done < <(find "$gotooldir" -maxdepth 1 -type f -printf '%f\n' | sort) | sha256sum)
[[ "${huella_gotool_inicial%% *}" == "1061bd99d16310f8f549e375a5c0cb18a79d66441ca0ed4dee60f70fde633f9b" ]] || { printf 'GOTOOLDIR no autorizado\n' >&2; exit 65; }
inventario_goroot_tmp=${temporal}/goroot_completo.tsv
: >"$inventario_goroot_tmp"
while IFS= read -r archivo; do huella=$(sha256sum -- "$goroot/$archivo"); printf '%s\t%s\n' "${huella%% *}" "$archivo" >>"$inventario_goroot_tmp"; done < <(find "$goroot" -type f -printf '%P\n' | LC_ALL=C sort)
huella_goroot=$(sha256sum -- "$inventario_goroot_tmp"); cantidad_goroot=$(wc -l <"$inventario_goroot_tmp")
[[ "${huella_goroot%% *}" == "b53ebeab1542ea933c6f995a2bcf862d505cb8343ad2b0d1f7a7de3238157ae6" && "$cantidad_goroot" == 11536 ]] || { printf 'árbol GOROOT no autorizado hash=%s count=%s\n' "${huella_goroot%% *}" "$cantidad_goroot" >&2; exit 65; }
limpiar() {
    local estado=$?
    trap - EXIT
    find "$temporal" -type f -exec unlink -- {} \;
    find "$temporal" -depth -type d -exec rmdir -- {} \;
    exit "$estado"
}
trap limpiar EXIT
repo_raiz=$(git -C "$(dirname -- "$0")" rev-parse --show-toplevel)
receta=${repo_raiz}/tools/o3a_v5_ast/receta_herramientas.txt
huella_receta=$(sha256sum -- "$receta")
[[ "${huella_receta%% *}" == "e0937279b46f946919c146de56446b7123ad03adcca29d54713da5ff56038135" ]] || { printf 'receta de herramientas no autorizada\n' >&2; exit 65; }
mkdir -p -- "$temporal/herramientas"
ast_bin=${temporal}/herramientas/o3a_v5_ast_checker
aplicador_bin=${temporal}/herramientas/o3a_v5_aplicador
(cd "$repo_raiz" && go_controlado build -trimpath -buildvcs=false -o "$ast_bin" ./tools/o3a_v5_ast && go_controlado build -trimpath -buildvcs=false -o "$aplicador_bin" ./tools/o3a_v5_ast/aplicar_manifest)

registrar() {
    local id=$1 compilo=$2 oraculo=$3 estado=$4 clasificacion=$5
    printf '%s=%s compilado=%s oraculo=%s estado=%s\n' "$id" "$clasificacion" "$compilo" "$oraculo" "$estado"
    if [[ -n "$ledger" ]]; then
        printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$base_sha" "$id" "$compilo" "$oraculo" "$estado" "$clasificacion" >>"$ledger"
    fi
}

fuentes=(
    supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1.go
    supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_control_preinicio.go
    supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_control_preinicio_pruebas.go
    supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operativo.go
    supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_sobre_s0.go
    supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_autoridad.go
    supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_preparacion.go
    supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_inicio.go
    supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas.go
    supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas_adversas.go
)

if [[ -n "$ledger" ]]; then
    [[ -n "$base_sha" ]] || { printf 'VEP_BASE_SHA obligatorio con VEP_LEDGER\n' >&2; exit 64; }
    [[ -x "$ast_bin" && -x "$aplicador_bin" && -f "$manifest" ]] || { printf 'staging o manifiesto inválido\n' >&2; exit 64; }
    mkdir -p -- "$(dirname -- "$ledger")"
    ledger_fuentes=${ledger%.tsv}_fuentes.tsv
    : >"$ledger_fuentes"
    for fuente in "${fuentes[@]}"; do
        huella=$(sha256sum -- "${origen}/${fuente}")
        printf '%s\t%s\n' "${huella%% *}" "$fuente" >>"$ledger_fuentes"
    done
    huella_conjunto=$(sha256sum -- "$ledger_fuentes")
    [[ "${huella_conjunto%% *}" == "$base_sha" ]] || { printf 'BASE de diez fuentes no autorizada: %s\n' "${huella_conjunto%% *}" >&2; exit 65; }
    ledger_herramientas=${ledger%.tsv}_herramientas.tsv
    ledger_manifest=${ledger%.tsv}_manifest.json
    perl -0777 -e '$s=<>; $e=()=$s=~/"estado"\s*:\s*"PARCIAL_NO_GO"/g; $c=()=$s=~/"cobertura_familias_completa"\s*:\s*false/g; die "manifest no revocado estado=$e cobertura=$c\n" unless $e == 1 && $c == 1' "$manifest"
    cp -- "$manifest" "$ledger_manifest"
    perl -0777 -pi -e '$e=s/"estado"\s*:\s*"PARCIAL_NO_GO"/"estado": "EJECUCION_AUTORIZADA"/; $c=s/"cobertura_familias_completa"\s*:\s*false/"cobertura_familias_completa": true/; die "snapshot ejecutable no normalizado\n" unless $e == 1 && $c == 1' "$ledger_manifest"
    printf 'tipo\treferencia\tsha256\tversion\n' >"$ledger_herramientas"
    for par in "runner:$0" "checker_fuente:${repo_raiz}/tools/o3a_v5_ast/main.go" "aplicador_fuente:${repo_raiz}/tools/o3a_v5_ast/aplicar_manifest/main.go" "go_mod:${repo_raiz}/go.mod" "go_sum:${repo_raiz}/go.sum" "receta:$receta" "manifest:$ledger_manifest"; do
        tipo=${par%%:*}; herramienta=${par#*:}; ruta_real=$(realpath -- "$herramienta")
        [[ "$ruta_real" == "$repo_raiz"/* ]] || { printf 'herramienta no portable: %s\n' "$ruta_real" >&2; exit 65; }
        referencia=${ruta_real#"$repo_raiz"/}; huella=$(sha256sum -- "$ruta_real")
        printf '%s\t%s\t%s\tcontenido\n' "$tipo" "$referencia" "${huella%% *}" >>"$ledger_herramientas"
    done
    huella_checker=$(sha256sum -- "$ast_bin"); huella_aplicador=$(sha256sum -- "$aplicador_bin")
    printf 'checker_bin\tSTAGING/o3a_v5_ast_checker\t%s\treconstruido\n' "${huella_checker%% *}" >>"$ledger_herramientas"
    printf 'aplicador_bin\tSTAGING/o3a_v5_aplicador\t%s\treconstruido\n' "${huella_aplicador%% *}" >>"$ledger_herramientas"
    huella_go=$(sha256sum -- "$go_bin")
    ledger_gotool=${ledger%.tsv}_gotool.tsv
    : >"$ledger_gotool"
    while IFS= read -r herramienta; do huella=$(sha256sum -- "$gotooldir/$herramienta"); printf '%s\t%s\n' "${huella%% *}" "$herramienta" >>"$ledger_gotool"; done < <(find "$gotooldir" -maxdepth 1 -type f -printf '%f\n' | sort)
    [[ -s "$ledger_gotool" ]] || { printf 'inventario GOTOOLDIR vacío\n' >&2; exit 65; }
    huella_gotool=$(sha256sum -- "$ledger_gotool"); cantidad_gotool=$(wc -l <"$ledger_gotool")
    printf 'gotooldir\tGOROOT/pkg/tool/%s_%s\t%s\tarchivos=%s\n' "$(go_controlado env GOOS)" "$(go_controlado env GOARCH)" "${huella_gotool%% *}" "$cantidad_gotool" >>"$ledger_herramientas"
    ledger_goroot=${ledger%.tsv}_goroot.tsv
    cp -- "$inventario_goroot_tmp" "$ledger_goroot"
    printf 'goroot_tree\tGOROOT/\t%s\tarchivos=%s\n' "${huella_goroot%% *}" "$cantidad_goroot" >>"$ledger_herramientas"
    version_file=${goroot}/VERSION; [[ -f "$version_file" ]] || { printf 'VERSION de GOROOT ausente\n' >&2; exit 65; }
    huella_version=$(sha256sum -- "$version_file"); version_goroot=$(tr -d '\r\n' <"$version_file")
    printf 'go\tGOROOT/bin/go\t%s\t%s\n' "${huella_go%% *}" "$go_version" >>"$ledger_herramientas"
    printf 'goroot\tGOROOT/VERSION\t%s\t%s\n' "${huella_version%% *}" "$version_goroot" >>"$ledger_herramientas"
    printf 'base_sha\tid\tcompilo\toraculo\testado\tclasificacion\n' >"$ledger"
fi

reemplazar_unico() {
    local archivo=$1 anterior=$2 posterior=$3
    perl -0777 - "$archivo" "$anterior" "$posterior" <<'PERL'
use strict; use warnings;
my ($ruta,$a,$p)=@ARGV;
open my $in,'<:raw',$ruta or die "$ruta: $!\n"; local $/; my $s=<$in>; close $in;
my $i=index($s,$a); die "patron ausente: $a\n" if $i<0;
die "patron multiple: $a\n" if index($s,$a,$i+length($a))>=0;
substr($s,$i,length($a),$p);
open my $out,'>:raw',$ruta or die "$ruta: $!\n"; print {$out} $s; close $out;
PERL
}

preparar() {
    local f
    for f in "${fuentes[@]}"; do cp -- "${origen}/${f}" "${temporal}/${f}"; done
    perl -0777 -e '$s=<>; $n=()=$s=~/if err = autoprobarArranqueO3aM38\(\); err != nil/g; die "autoprueba cardinalidad=$n\n" unless $n == 1' "${temporal}/${fuentes[0]}"
}

instrumentar_oraculo_retirada() {
    local g7b=${temporal}/${fuentes[9]}
    reemplazar_unico "$g7b" \
        'func autoprobarArranqueO3aM38() error {' \
        $'func probarRetiradaForzadaConHijoO3aM38() (err error) {\n\tf, err := crearFixtureO3aM38("NOMINAL")\n\tif err != nil { return err }\n\tdefer func() { err = errors.Join(err, limpiarFixtureO3aM38(f)) }()\n\tif err = prepararFixtureO3aM38(f); err != nil { return err }\n\tc := f.preparado.custodia\n\tr := avanzarArranqueO3aM38(f.preparado, c.reloj.emitir())\n\tf.preparado = nil\n\tif r.clase != resultadoRetiradoO3aM38 || r.retirada == nil || r.retirada.origen != retiradaConHijoO3aM38 { return errInvarianteO3aM38 }\n\tf.retirada = r.retirada\n\treturn nil\n}\n\nfunc autoprobarArranqueO3aM38() error {'
    reemplazar_unico "$g7b" \
        $'\tfor i := 0; i < 100; i++ {\n\t\tif err = probarArranqueRealO3aM38(); err != nil {\n\t\t\treturn fmt.Errorf("iteración O3a %d: %w", i, err)\n\t\t}\n\t}' \
        $'\tif err = probarRetiradaForzadaConHijoO3aM38(); err != nil {\n\t\treturn fmt.Errorf("retirada forzada O3a: %w", err)\n\t}'
}

aplicar() {
    local id=$1 g6b=${temporal}/${fuentes[6]} g6c=${temporal}/${fuentes[7]}
    if [[ "$id" =~ ^M052([ERO])([0-8])$ ]]; then
        local variante=${BASH_REMATCH[1]} indice=${BASH_REMATCH[2]}
        case "$variante" in
            E) reemplazar_unico "$g6c" 'if err = revalidarDestinadosO3aM38(c); err != nil {' $'_ = c.destinados['"$indice"$'].Close()\n\tif err = revalidarDestinadosO3aM38(c); err != nil {' ;;
            R) reemplazar_unico "$g6c" 'forma != c.huellasDestinadas[i] || forma.fdflags&syscall.FD_CLOEXEC == 0' 'i != '"$indice"' && forma != c.huellasDestinadas[i] || forma.fdflags&syscall.FD_CLOEXEC == 0' ;;
            O) reemplazar_unico "$g6c" $'for i, f := range c.destinados {\n\t\tcerrado, err :=' $'for i, f := range c.destinados {\n\t\tif i == '"$indice"$' { continue }\n\t\tcerrado, err :=' ;;
        esac
        return
    fi
    case "$id" in
        BASE) ;;
        C043|M043)
            instrumentar_oraculo_retirada
            reemplazar_unico "$g6c" \
                'reserva, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(pidfdPrimario), syscall.F_DUPFD_CLOEXEC, minFDDuplicadoM38)' \
                'reserva, errno := uintptr(0), syscall.EMFILE'
            if [[ "$id" == C043 ]]; then return; fi
            reemplazar_unico "$g6c" \
                $'\tif !reservaEntera {\n\t\treturn retirarConHijoO3aM38(c, fmt.Errorf("%w: %v", errProcesoO3aM38, errno), finRetirada)\n\t}' \
                $'\tif !reservaEntera {\n\t\tc.pidfdReserva = pidfdPrimario\n\t\t_ = fmt.Sprintf("%v", errno)\n\t}' ;;
        C054|M054)
            instrumentar_oraculo_retirada
            reemplazar_unico "$g6c" \
                $'\tc.pidfdReserva = int(reserva)\n\tif errStart != nil {' \
                $'\tc.pidfdReserva = int(reserva)\n\tif time.Now().UnixNano() != 0 {\n\t\treturn retirarConHijoO3aM38(c, errProcesoO3aM38, finRetirada)\n\t}\n\tif errStart != nil {'
            if [[ "$id" == C054 ]]; then return; fi
            reemplazar_unico "$g6c" \
                $'\tif err := esperarConLeaseO3aM38(c); err != nil {' \
                $'\tif err := error(nil); err != nil {' ;;
        C055|M055)
            instrumentar_oraculo_retirada
            reemplazar_unico "$g6c" \
                $'\tc.pidfdReserva = int(reserva)\n\tif errStart != nil {' \
                $'\tc.pidfdReserva = int(reserva)\n\tif time.Now().UnixNano() != 0 {\n\t\treturn retirarConHijoO3aM38(c, errProcesoO3aM38, finRetirada)\n\t}\n\tif errStart != nil {'
            if [[ "$id" == C055 ]]; then return; fi
            reemplazar_unico "$g6c" \
                $'\tif !terminalAntesO3aM38(fiable, fin) {' \
                $'\t_ = esperarConLeaseO3aM38(c)\n\tif !terminalAntesO3aM38(fiable, fin) {' ;;
        C064|M064)
            instrumentar_oraculo_retirada
            reemplazar_unico "$g6c" \
                $'\tc.pidfdReserva = int(reserva)\n\tif errStart != nil {' \
                $'\tc.pidfdReserva = int(reserva)\n\terrStart = errProcesoO3aM38\n\tif errStart != nil {'
            if [[ "$id" == C064 ]]; then return; fi
            reemplazar_unico "$g6c" \
                $'\tif errStart != nil {\n\t\treturn retirarConHijoO3aM38(c, errStart, finRetirada)\n\t}' \
                $'\tif errStart != nil {\n\t\treturn retirarPreparadoO3aM38(c, errStart)\n\t}' ;;
        M016)
            reemplazar_unico "$g6b" 'Setpgid: true, Pgid: 0, Pdeathsig: syscall.SIGKILL, PidFD: &pidfdPrimario' 'Setpgid: false, Pgid: 0, Pdeathsig: syscall.SIGKILL, PidFD: &pidfdPrimario' ;;
        M017)
            reemplazar_unico "$g6b" 'Setpgid: true, Pgid: 0, Pdeathsig: syscall.SIGKILL, PidFD: &pidfdPrimario' 'Setpgid: true, Pgid: 1, Pdeathsig: syscall.SIGKILL, PidFD: &pidfdPrimario' ;;
        M018)
            reemplazar_unico "$g6b" 'Setpgid: true, Pgid: 0, Pdeathsig: syscall.SIGKILL, PidFD: &pidfdPrimario' 'Setpgid: true, Pgid: 0, Pdeathsig: 0, PidFD: &pidfdPrimario' ;;
        M019)
            reemplazar_unico "$g6b" 'Setpgid: true, Pgid: 0, Pdeathsig: syscall.SIGKILL, PidFD: &pidfdPrimario' 'Setpgid: true, Pgid: 0, Pdeathsig: syscall.SIGKILL, PidFD: nil' ;;
        M020A)
            reemplazar_unico "$g6b" 'Setpgid: true, Pgid: 0, Pdeathsig: syscall.SIGKILL, PidFD: &pidfdPrimario' 'Setpgid: true, Pgid: 0, Pdeathsig: syscall.SIGKILL, PidFD: &pidfdPrimario, Setsid: true' ;;
        M020B)
            reemplazar_unico "$g6b" 'Setpgid: true, Pgid: 0, Pdeathsig: syscall.SIGKILL, PidFD: &pidfdPrimario' 'Setpgid: true, Pgid: 0, Pdeathsig: syscall.SIGKILL, PidFD: &pidfdPrimario, AmbientCaps: []uintptr{1}' ;;
        M021)
            reemplazar_unico "$g6c" 'resultado, err := barreraAntesStartO3aM38(c)' 'resultado, err := barreraVerdeO3aM38, error(nil)' ;;
        M022)
            reemplazar_unico "$g6c" 'if err = barreraDespuesStartO3aM38(c); err != nil {' 'if err = error(nil); err != nil {' ;;
        M028)
            reemplazar_unico "$g6b" 'e.control.fase != controlPreinicioS3M38 || e.control.causa != (causaPreinicioM38{})' 'false || e.control.causa != (causaPreinicioM38{})' ;;
        M029)
            reemplazar_unico "$g6c" $'\terrStart := c.cmd.Start()\n\tpidfdPrimario :=' $'\terrStart := c.cmd.Start()\n\t_ = c.cmd.Start()\n\tpidfdPrimario :=' ;;
        M030)
            reemplazar_unico "$g6c" $'\tpidfdPrimario := *c.cmd.SysProcAttr.PidFD\n\tc.pidfdPrimario = pidfdPrimario' $'\tpidfdPrimario := *c.cmd.SysProcAttr.PidFD\n\tpidfdPrimario = -1\n\tc.pidfdPrimario = pidfdPrimario' ;;
        M031)
            reemplazar_unico "$g6c" $'\terrStart := c.cmd.Start()\n\tpidfdPrimario :=' $'\terrStart := c.cmd.Start()\n\t_, _ = c.ticketEscritor.Write([]byte{1})\n\tpidfdPrimario :=' ;;
        M032)
            reemplazar_unico "$g6c" $'\terrStart := c.cmd.Start()\n\tpidfdPrimario :=' $'\terrStart := c.cmd.Start()\n\t_, _ = os.ReadFile(fmt.Sprintf("/proc/%d/stat", c.cmd.Process.Pid))\n\tpidfdPrimario :=' ;;
        M033)
            reemplazar_unico "$g6c" 'cerrado, errCierre := cerrarUnoConLeaseO3aM38(c.lease, c.ticketEscritor, operacionCerrarTicketO3aM38)' 'cerrado, errCierre := true, error(nil)' ;;
        M034)
            reemplazar_unico "$g6c" 'enviarPidfdIndividualO3aM38(fiable, syscall.SIGKILL)' 'syscall.Kill(c.cmd.Process.Pid, syscall.SIGKILL)' ;;
        M035A)
            reemplazar_unico "$g6c" 'if err := esperarConLeaseO3aM38(c); err != nil {' 'if err := error(nil); err != nil {' ;;
        M035B)
            reemplazar_unico "$g6c" 'if err := esperarConLeaseO3aM38(c); err != nil {' $'_ = esperarConLeaseO3aM38(c)\n\tif err := esperarConLeaseO3aM38(c); err != nil {' ;;
        M026A)
            reemplazar_unico "$g6b" '!observadorValido || contadorSenal != e.baselineSenal' 'false && !observadorValido || contadorSenal != e.baselineSenal' ;;
        M026B)
            reemplazar_unico "$g6b" 'contadorSenal != e.baselineSenal || time.Until(e.finBootstrap)' 'false && contadorSenal != e.baselineSenal || time.Until(e.finBootstrap)' ;;
        M026C)
            reemplazar_unico "$g6c" 'if !valido || contador != c.baselineSenal {' 'if false && (!valido || contador != c.baselineSenal) {' ;;
        M026D)
            reemplazar_unico "$g6c" 'if !observadorTransferido {' 'if false && !observadorTransferido {' ;;
        M026E)
            reemplazar_unico "${temporal}/${fuentes[5]}" 'o.registro.observadores[o] != o.generacion || o.registro.tid != syscall.Gettid()' 'o.registro.tid != syscall.Gettid()' ;;
        M026F)
            reemplazar_unico "$g6b" 'contadorSenal, _, observadorValido := e.observador.observar()' 'contadorSenal, observadorValido := e.baselineSenal, true' ;;
        M026G)
            reemplazar_unico "${temporal}/${fuentes[5]}" 'return palabra, syscall.Signal(uint8(palabra >> 2)), estado == 1 || estado == 2' 'return palabra, 0, estado == 1 || estado == 2' ;;
        M026H)
            reemplazar_unico "${temporal}/${fuentes[5]}" 'o.registro.observadores[o] != o.generacion || o.registro.tid != syscall.Gettid()' 'o.registro.observadores[o] != o.generacion' ;;
        M037)
            reemplazar_unico "$g6c" 'if err != nil || vuelta <= c.vueltaInicio {' 'if err != nil || false && vuelta <= c.vueltaInicio {' ;;
        M038A)
            reemplazar_unico "${temporal}/${fuentes[5]}" 'c == nil || t.celda != c || c.auto != c || c.reloj != r' 'c == nil || t.celda != c || c.reloj != r' ;;
        M038B)
            reemplazar_unico "${temporal}/${fuentes[5]}" 'c == nil || t.celda != c || c.auto != c || c.reloj != r' 'c == nil || c.auto != c || c.reloj != r' ;;
        M038C)
            reemplazar_unico "${temporal}/${fuentes[5]}" $'c.reloj != r ||\n\t\tc.tid != r.tid' 'c.tid != r.tid' ;;
        M038D)
            reemplazar_unico "${temporal}/${fuentes[5]}" '!c.consumo.CompareAndSwap(0, 1)' 'false' ;;
        M042)
            reemplazar_unico "$g6c" 'c.pidfdReserva = int(reserva)' 'c.pidfdReserva = pidfdPrimario' ;;
        M044A)
            reemplazar_unico "$g6c" 'if n == 1 && p.retorno == 1 {' 'if n == 1 && p.retorno != 0 {' ;;
        M044B)
            reemplazar_unico "$g6c" 'if n == 0 && p.retorno == 0 {' 'if n == 0 {' ;;
        M045)
            reemplazar_unico "$g6c" $'func barreraDespuesStartO3aM38(c *custodiaO3aM38) error {\n\tresultado, err :=' $'func barreraDespuesStartO3aM38(c *custodiaO3aM38) error {\n\tif vivo, _ := pidfdVivoO3aM38(c.pidfdPrimario); !vivo { return errProcesoO3aM38 }\n\tresultado, err :=' ;;
        M039A)
            reemplazar_unico "$g6c" 'func leerControlBarreraO3aM38(' $'func segundoParserControlO3aM38(b []byte) bool { return len(b) > 0 }\n\nfunc leerControlBarreraO3aM38(' ;;
        M039B)
            reemplazar_unico "${temporal}/${fuentes[1]}" 'nuevoLectorTramaM38("CONTROL")' 'nuevoLectorTramaM38("CONTROL2")' ;;
        M040A)
            reemplazar_unico "$g6c" 'syscall.F_DUPFD_CLOEXEC, minFDDuplicadoM38' 'syscall.F_DUPFD, minFDDuplicadoM38' ;;
        M040B)
            reemplazar_unico "$g6c" $'reserva, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(pidfdPrimario), syscall.F_DUPFD_CLOEXEC, minFDDuplicadoM38)\n\tahora :=' $'reserva, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(pidfdPrimario), syscall.F_DUPFD_CLOEXEC, minFDDuplicadoM38)\n\t_, _, _ = syscall.Syscall(syscall.SYS_FCNTL, uintptr(pidfdPrimario), syscall.F_DUPFD_CLOEXEC, minFDDuplicadoM38)\n\tahora :=' ;;
        M040C)
            reemplazar_unico "$g6b" 'PidFD: &pidfdPrimario' 'PidFD: nil' ;;
        M040D)
            reemplazar_unico "$g6c" 'pidfdOpaco, deltaValido := deltaPidfdO3aM38(c.lease.pre, conPidfd, pidfdPrimario, reservaFD)' $'_ = reservaFD\n\tpidfdOpaco, deltaValido := deltaPidfdO3aM38(c.lease.pre, conPidfd, pidfdPrimario, pidfdPrimario)' ;;
        M040E)
            reemplazar_unico "$g6c" 'reserva, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(pidfdPrimario), syscall.F_DUPFD_CLOEXEC, minFDDuplicadoM38)' 'reserva, errno := uintptr(pidfdPrimario), syscall.Errno(0)' ;;
        M041A)
            reemplazar_unico "$g6c" $'\tahora := time.Now()' $'\t_ = fmt.Sprintf("duplicacion-tardia")\n\t_, _, _ = syscall.Syscall(syscall.SYS_FCNTL, uintptr(pidfdPrimario), syscall.F_DUPFD_CLOEXEC, minFDDuplicadoM38)\n\tahora := time.Now()' ;;
        M041B)
            reemplazar_unico "$g6c" $'\tconPidfd, errInventario := snapshotActualO3aM38()' $'\tconPidfd, errInventario := snapshotActualO3aM38()\n\t_, _, _ = syscall.Syscall(syscall.SYS_FCNTL, uintptr(pidfdPrimario), syscall.F_DUPFD_CLOEXEC, minFDDuplicadoM38)' ;;
        M041C)
            reemplazar_unico "$g6c" $'\tc.ticketLector = nil\n\tdelete(conTres.mapa, fdTicket)' $'\tc.ticketLector = nil\n\t_, _, _ = syscall.Syscall(syscall.SYS_FCNTL, uintptr(pidfdPrimario), syscall.F_DUPFD_CLOEXEC, minFDDuplicadoM38)\n\tdelete(conTres.mapa, fdTicket)' ;;
        M041D)
            reemplazar_unico "$g6c" $'\tif err = barreraDespuesStartO3aM38(c); err != nil {' $'\t_, _, _ = syscall.Syscall(syscall.SYS_FCNTL, uintptr(pidfdPrimario), syscall.F_DUPFD_CLOEXEC, minFDDuplicadoM38)\n\tif err = barreraDespuesStartO3aM38(c); err != nil {' ;;
        M046)
            reemplazar_unico "$g6b" 'autoridad: autoridad, control: e.control, controlFD: e.controlFD, terminal: e.terminal' 'autoridad: autoridad, control: e.control, controlFD: e.controlFD, terminal: nil' ;;
        M047)
            reemplazar_unico "$g6b" 'raiz, runner, ticketLector}' 'raiz, runner, ticketLector, e.terminal}' ;;
        M048)
            reemplazar_unico "$g6c" $'return resultadoArranqueO3aM38{clase: resultadoRetiradoO3aM38, retirada: &retiradaO3aM38{\n\t\torigen: retiradaConHijoO3aM38' $'_ = c.terminal.Close()\n\treturn resultadoArranqueO3aM38{clase: resultadoRetiradoO3aM38, retirada: &retiradaO3aM38{\n\t\torigen: retiradaConHijoO3aM38' ;;
        M049)
            reemplazar_unico "$g6b" 'formas[5], err = validarRegularO3aM38(e.terminal, syscall.O_WRONLY, true)' 'formas[5], err = validarRegularO3aM38(e.salida, syscall.O_WRONLY, true)' ;;
        M050A)
            reemplazar_unico "$g6b" 'Env: []string{"LC_ALL=C", "PATH=/usr/local/go/bin:/usr/bin:/bin"}' 'Env: os.Environ()' ;;
        M050B)
            reemplazar_unico "$g6b" 'Env: []string{"LC_ALL=C", "PATH=/usr/local/go/bin:/usr/bin:/bin"}' 'Env: []string{"LC_ALL=C", "PATH=/usr/local/go/bin:/usr/bin:/bin", "HOME=/"}' ;;
        M051)
            reemplazar_unico "$g6c" $'\terrStart := c.cmd.Start()' $'\t_, _, _ = syscall.Syscall6(436, 3, ^uintptr(0), 0, 0, 0, 0)\n\terrStart := c.cmd.Start()' ;;
        M053A)
            reemplazar_unico "$g6c" 'return resultadoArranqueO3aM38{clase: resultadoEntregadoO3aM38' $'_ = c.controlFD.Close()\n\treturn resultadoArranqueO3aM38{clase: resultadoEntregadoO3aM38' ;;
        M053B)
            reemplazar_unico "$g6c" 'return resultadoArranqueO3aM38{clase: resultadoEntregadoO3aM38' $'_ = c.terminal.Close()\n\treturn resultadoArranqueO3aM38{clase: resultadoEntregadoO3aM38' ;;
        *)
            if [[ -n "$aplicador_bin" && -n "$manifest" ]]; then
                herramienta_controlada "$aplicador_bin" "$manifest" "$id" "$temporal" >/dev/null
            else
                printf 'mutante desconocido: %s\n' "$id" >&2
                return 64
            fi ;;
    esac
}

ejecutar() {
    local id=$1
    local bin=${temporal}/supervisor-${id} salida=${temporal}/${id}.out
    preparar
    aplicar "$id"
    go_controlado fmt "${temporal}"/*.go >/dev/null
    go_controlado vet "${temporal}"/*.go
    go_controlado build -trimpath -buildvcs=false -o "$bin" "${temporal}"/*.go
    if [[ -n "$ast_bin" ]]; then
        set +e
        herramienta_controlada "$ast_bin" -dir "$temporal" >"${temporal}/${id}.ast" 2>&1
        local estado_ast=$?
        set -e
        if ((estado_ast != 0)); then
            registrar "$id" si ast_tipado "$estado_ast" muerto_ast
            find "$temporal" -maxdepth 1 -type f -exec unlink -- {} \;
            return
        fi
    fi
    set +e
    timeout 45s env -i HOME="$control_home" PATH=/usr/bin:/bin TMPDIR="$temporal/tmp" "$bin" --autoprueba >"$salida" 2>&1
    local estado=$?
    set -e
    if [[ "$id" == BASE && "$estado" == 0 || "$id" == C* && "$estado" == 0 ]]; then
        registrar "$id" si autoprueba "$estado" verde
    elif ((estado == 0)); then
        registrar "$id" si autoprueba "$estado" superviviente
    else
        registrar "$id" si autoprueba "$estado" muerto
    fi
    find "$temporal" -maxdepth 1 -type f -exec unlink -- {} \;
}

if (($# > 1)); then
    shift
    for mutante in "$@"; do ejecutar "$mutante"; done
else
    for mutante in BASE C043 M043 C054 M054 C055 M055 C064 M064; do ejecutar "$mutante"; done
fi
