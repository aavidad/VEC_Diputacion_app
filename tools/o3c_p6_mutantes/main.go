// El ejecutor mutante O3c construye cada variante sobre un archive limpio.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"syscall"
	"time"
)

const base = "c0f2a9945ed2fc5648980ee48b91424a04977655"
const paquete = "deploy/postgresql/autorizacion_atestada_v3/pruebas_sql"

var fuentes = map[string]string{
	"A": "continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad.go",
	"R": "continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_revalidacion.go",
	"C": "continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_cont.go",
	"O": "continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_observacion.go",
	"H": "continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff.go",
	"B": "captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_barrera.go",
	"G": "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_autoridad.go",
	"T": "continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff_test.go",
	"Q": "tools/o3c_p6_conductor/conductor.sh",
}

var hashesBase = map[string]string{
	"A": "150c46ebeef8b6d2850d735b1f679701d620f0ef54850ab01f20fca986c9a599", "B": "0499ede483615d57d5438d579f32580649b40fbf34b3be6bc26d31fa6c86c02d",
	"C": "447d5779ea90731b3b53a46870861b79bc95a478bd0fa7540b717ea279bd94be", "G": "9015dff049f04f839920c964a5d8471c1b3f7f9e3dcab339266cf2e13f155bd8",
	"H": "66fb9c71e8c5d5e03cd7a32380986c23a0139720673a9b2348e1bcf03d3ec4cf", "O": "bf2a5814608479cfe03628be31e951087a6da29f21f8a88a053108fa0d6620b0",
	"R": "35409603d803a6d74288a391bea93239d2246fbe3c0d35eecf2063c0da1fe1aa", "T": "13bac7a37f5d82bc27d2cb4f767b963eee6f99961cf574a8616186afde391d76",
}

type mutante struct{ id, familia, alternativa, fuente, antes, despues, oraculo string }
type resultado struct {
	mutante
	compilacion, muerte, duracion, salidaSHA string
}

func m(f, a, src, before, after, oracle string) mutante {
	return mutante{familia: f, alternativa: a, fuente: src, antes: before, despues: after, oraculo: oracle}
}

// catalogo expande todas las alternativas normativas. Cada transformación
// cambia exactamente una ocurrencia y produce una desviación ejecutable real.
func catalogo() []mutante {
	c := []mutante{
		m("C01", "omitir-CAS-observador", "A", "nuevoBaseline, ok := c.observador.transferirCritico(c.baselineSenal)", "nuevoBaseline, ok := c.baselineSenal, true", "consumo-observador-1-a-2"),
		m("C01", "omitir-CAS-lease", "A", "!c.lease.transferirCritico()", "false", "consumo-lease-1-a-3"),
		m("C01", "reordenar-CAS", "A", "nuevoBaseline, ok := c.observador.transferirCritico(c.baselineSenal)", "_ = c.lease.transferirCritico(); nuevoBaseline, ok := c.observador.transferirCritico(c.baselineSenal)", "observador-antes-lease"),
		m("C01", "mutar-B5", "A", "a.estado != capturaB5CapturadoM38", "a.estado == capturaB5CapturadoM38", "B5-read-only-exacto"),
		m("C01", "no-anular-llamador", "A", "*entrada = nil", "_ = entrada", "puntero-anulado-antes-validar"),
		m("C01", "aceptar-alias", "A", "if estadoLease == 3 && estadoObservador == 2", "if estadoLease == 3 && estadoObservador == 1", "alias-consumido"),
		m("C01", "aceptar-clon", "A", "return entradaO3cConsumidaM38\n\t}\n\tif estadoLease != 1", "return entradaO3cValidaM38\n\t}\n\tif estadoLease != 1", "clon-no-readquiere"),
		m("C01", "aceptar-reuso-antes-C5", "A", "if estadoLease == 3 && estadoObservador == 2", "if estadoLease == 3", "reuso-antes-C5"),
		m("C01", "aceptar-replay-despues-C5", "A", "if estadoLease == 3 && estadoObservador == 2", "if estadoObservador == 2", "replay-despues-C5"),
		m("C01", "retornar-particion", "A", "fatalO3cM38()\n\t}\n\treturn nuevoBaseline", "return nuevoBaseline\n\t}\n\treturn nuevoBaseline", "particion-no-retorna"),

		m("C02", "aceptar-estado-no-B5", "A", "a.estado != capturaB5CapturadoM38", "false", "estado-B5-exacto"),
		m("C02", "aceptar-atomicos-no-1-1", "A", "estadoLease != 1 || estadoObservador != 1", "estadoLease == 0 && estadoObservador == 0", "atomicos-1-1"),
		m("C02", "aceptar-recurso-nulo", "A", "c.cmd != nil && c.cmd.Process != nil", "true", "cmd-process-presentes"),
		m("C02", "aceptar-ticket-writer", "A", "c.ticketEscritor == nil", "true", "ticket-writer-ausente"),
		m("C02", "aceptar-observacion-previa", "A", "a.primera == nil", "true", "observacion-entrada-vacia"),

		m("C03", "omitir-registro", "A", "c.lease.registro == nil", "false", "registro-presente"),
		m("C03", "omitir-pertenencia", "A", "c.lease.registro.leases[c.lease] != c.lease.generacion", "false", "lease-pertenece-registro"),
		m("C03", "omitir-generacion", "A", "c.observador.registro.observadores[c.observador] != c.observador.generacion", "false", "observador-generacion"),
		m("C03", "omitir-TID", "A", "c.lease.tid != c.tid", "false", "TID-lease"),
		m("C03", "omitir-PPID", "R", "ppid == c.ppid", "ppid >= 0", "PPID-sellado"),
		m("C03", "omitir-PDEATHSIG", "R", "pdeathsig == int32(syscall.SIGTERM)", "pdeathsig >= 0", "PDEATHSIG-SIGTERM"),
		m("C03", "omitir-baseline", "A", "c.baselineSenal != c.observador.palabra.Load()", "false", "baseline-exacto"),
		m("C03", "omitir-signo", "A", "uint8(c.baselineSenal>>2) != 0", "false", "signo-cero"),

		m("C04", "reordenar-CONTROL-segunda-ronda", "R", "if err := leerControlO3bM38(a.custodia); err != nil {", "_, _ = autoridadSenalO3cM38(a.custodia); if err := leerControlO3bM38(a.custodia); err != nil {", "CONTROL-primero-segunda-ronda"),
		m("C04", "reordenar-CONTROL-ronda-principal", "R", "if err := leerControlO3bM38(c); err != nil {", "_, _ = autoridadSenalO3cM38(c); if err := leerControlO3bM38(c); err != nil {", "CONTROL-primero-ronda-principal"),
		m("C04", "aceptar-EOF", "R", "return resolverRevalidacionO3cM38(a, err, preContControlO3cM38)", "return nil, nil", "EOF-retira"),
		m("C04", "aceptar-parcial", "O", "return instalarObservacionO3cM38(a, observacionControlRawO3cM38)", "return instalarObservacionO3cM38(a, observacionPidfdVacioO3cM38)", "CONTROL-raw-cierra"),
		m("C04", "aceptar-framing", "R", "return resolverRevalidacionO3cM38(a, err, preContControlO3cM38)", "return resolverRevalidacionO3cM38(a, err, preContPidfdO3cM38)", "framing-precedencia"),
		m("C04", "aceptar-presupuesto", "R", "leerControlO3bM38(a.custodia)", "error(nil)", "presupuesto-no-verde"),
		m("C04", "aceptar-EINTR", "O", "if err := leerControlO3bM38(a.custodia); err != nil", "if err := leerControlO3bM38(a.custodia); err != nil && !errors.Is(err, syscall.EINTR)", "EINTR-no-verde"),

		m("C05", "recrear-bootstrap", "R", "if c.finBootstrap.IsZero() || !time.Now().Before(c.finBootstrap)", "c.finBootstrap = time.Now().Add(time.Hour); if c.finBootstrap.IsZero() || !time.Now().Before(c.finBootstrap)", "bootstrap-no-recreado"),
		m("C05", "extender-bootstrap", "R", "c.finBootstrap == s.finBootstrap", "c.finBootstrap.After(s.finBootstrap)", "bootstrap-no-extendido"),
		m("C05", "ignorar-vencimiento", "C", "!ahoraCaso.Before(c.finBootstrap)", "false", "vencimiento-cero-CONT"),

		m("C06", "omitir-primario", "R", "fiablePrimario := errPrimario == nil && errVivoPrimario == nil && vivoPrimario", "_ = errPrimario; _ = errVivoPrimario; _ = vivoPrimario; fiablePrimario := true", "primario-acreditado"),
		m("C06", "omitir-reserva", "R", "fiableReserva := errReserva == nil && errVivoReserva == nil && vivoReserva", "_ = errReserva; _ = errVivoReserva; _ = vivoReserva; fiableReserva := true", "reserva-acreditada"),
		m("C06", "omitir-handle", "R", "errOpaco != nil", "false", "handle-opaco-acreditado"),
		m("C06", "aceptar-cuarta", "R", "referencias != 3", "referencias != 4", "exactamente-tres-referencias"),
		m("C06", "duplicar", "R", "actual, err := snapshotBarreraO3bM38(c)", "_, _ = syscall.Dup(c.pidfdPrimario); actual, err := snapshotBarreraO3bM38(c)", "sin-dup"),
		m("C06", "promover-reserva", "C", "uintptr(c.pidfdPrimario)", "uintptr(c.pidfdReserva)", "CONT-solo-primario"),

		m("C07", "aceptar-terminalidad", "R", "!fiablePrimario && !fiableReserva", "false", "terminalidad-preCONT-retira"),
		m("C07", "aceptar-flags", "R", "primario.fdflags&syscall.FD_CLOEXEC == 0", "false", "CLOEXEC-obligatorio"),
		m("C07", "ruta-proc-distinta", "R", "leerStatStopO3bM38(a.custodia)", "os.ReadFile(\"/proc/self/stat\")", "proc-unico-del-PID"),
		m("C07", "omitir-T", "R", "a.identidad.estado == 'T'", "true", "proc-estado-T"),
		m("C07", "omitir-PID", "R", "a.identidad.pid == c.cmd.Process.Pid", "true", "proc-PID"),
		m("C07", "omitir-PPID", "R", "a.identidad.ppid > 0", "true", "proc-PPID"),
		m("C07", "omitir-PGID", "R", "a.identidad.pgid == c.cmd.Process.Pid", "true", "proc-PGID"),
		m("C07", "omitir-SID", "R", "a.identidad.sid > 0", "true", "proc-SID"),
		m("C07", "omitir-starttime", "R", "a.identidad.inicio > 0", "true", "proc-starttime"),

		m("C08", "compartir-permiso", "R", "return operarConLeaseBarreraO3bM38(c, operacion)", "return operacion()", "permiso-lease-por-syscall"),
		m("C08", "forjar-permiso", "C", "p.lease == l", "true", "permiso-no-forjado"),
		m("C08", "omitir-comenzar", "R", "operarConLeaseBarreraO3bM38(c, operacion)", "operacion()", "comenzar-obligatorio"),
		m("C08", "omitir-consolidar", "C", "l.estado.CompareAndSwap(2, p.estadoPrevio)", "true", "consolidacion-inmediata"),

		m("C09", "deadline-antes", "C", "ahoraCaso := time.Now()", "finCaso, _ := finCasoExactoO3cM38(time.Now()); ahoraCaso := time.Now(); _ = finCaso", "deadline-tras-reloj-final"),
		m("C09", "deadline-despues", "C", "finCaso, marcaValida := finCasoExactoO3cM38(ahoraCaso)", "finCaso, marcaValida := ahoraCaso, true", "deadline-antes-CONT"),
		m("C09", "deadline-doble", "C", "finCaso, marcaValida := finCasoExactoO3cM38(ahoraCaso)", "finCaso, marcaValida := finCasoExactoO3cM38(ahoraCaso); finCaso, marcaValida = finCasoExactoO3cM38(ahoraCaso)", "deadline-unico"),
		m("C09", "179-segundos", "C", "duracionCasoO3cM38     = 180 * time.Second", "duracionCasoO3cM38     = 179 * time.Second", "deadline-180s"),
		m("C09", "181-segundos", "C", "duracionCasoO3cM38     = 180 * time.Second", "duracionCasoO3cM38     = 181 * time.Second", "deadline-180s"),
		m("C09", "aceptar-overflow", "C", "tiempoMonotonoO3cM38(fin) && fin.After(ahora)", "tiempoMonotonoO3cM38(fin)", "overflow-rechazado"),

		m("C10", "insertar-syscall", "C", "_, _, retornoRaw := syscall.Syscall6", "_ = syscall.Gettimeofday(nil); _, _, retornoRaw := syscall.Syscall6", "sin-syscall-entre-marca-CONT"),
		m("C10", "insertar-reloj", "C", "_, _, retornoRaw := syscall.Syscall6", "_ = time.Now(); _, _, retornoRaw := syscall.Syscall6", "sin-reloj-entre-marca-CONT"),
		m("C10", "insertar-log", "C", "_, _, retornoRaw := syscall.Syscall6", "println(\"CONT\"); _, _, retornoRaw := syscall.Syscall6", "sin-log-entre-marca-CONT"),
		m("C10", "insertar-asignacion", "C", "_, _, retornoRaw := syscall.Syscall6", "intermedio := make([]byte, 1); _ = intermedio; _, _, retornoRaw := syscall.Syscall6", "sin-asignacion-falible"),
		m("C10", "insertar-IO", "C", "_, _, retornoRaw := syscall.Syscall6", "_, _ = syscall.Write(1, nil); _, _, retornoRaw := syscall.Syscall6", "sin-IO-entre-marca-CONT"),

		m("C11", "senal-cero", "C", "uintptr(syscall.SIGCONT)", "0", "SIGCONT-exacto"),
		m("C11", "senal-STOP", "C", "uintptr(syscall.SIGCONT)", "uintptr(syscall.SIGSTOP)", "SIGCONT-exacto"),
		m("C11", "senal-TERM", "C", "uintptr(syscall.SIGCONT)", "uintptr(syscall.SIGTERM)", "SIGCONT-exacto"),
		m("C11", "senal-KILL", "C", "uintptr(syscall.SIGCONT)", "uintptr(syscall.SIGKILL)", "SIGCONT-exacto"),
		m("C11", "flag-cero", "C", "uintptr(syscall.SIGCONT), 0, pidfdSignalGrupoO3cM38, 0, 0", "uintptr(syscall.SIGCONT), 0, 0, 0, 0", "flag-grupo"),
		m("C11", "PID-numerico", "C", "uintptr(c.pidfdPrimario)", "uintptr(c.cmd.Process.Pid)", "sin-PID-signal"),
		m("C11", "PGID-numerico", "C", "uintptr(c.pidfdPrimario)", "uintptr(-c.cmd.Process.Pid)", "sin-PGID-signal"),

		m("C12", "reintentar-CONT", "C", "if !consolidarContO3cM38", "_, _, _ = syscall.Syscall6(sysPidfdSendSignal, uintptr(c.pidfdPrimario), uintptr(syscall.SIGCONT), 0, pidfdSignalGrupoO3cM38, 0, 0); if !consolidarContO3cM38", "CONT-un-intento"),
		m("C12", "EINTR-no-intento", "C", "a.salida.retornoCont = int(retornoRaw)", "if retornoRaw != syscall.EINTR { a.salida.retornoCont = int(retornoRaw) }", "EINTR-consume-intento"),
		m("C12", "perder-raw", "C", "a.salida.retornoCont = int(retornoRaw)", "_ = retornoRaw; a.salida.retornoCont = 0", "retorno-raw-conservado"),
		m("C13", "CONT-tras-bootstrap", "C", "!ahoraCaso.Before(c.finBootstrap)", "ahoraCaso.Before(c.finBootstrap)", "CONT-antes-bootstrap"),

		m("C14", "poll-bloqueante", "O", "uintptr(cardinalidad), 0)", "uintptr(cardinalidad), 1)", "poll-timeout-cero"),
		m("C14", "omitir-poll", "O", "n, _, errno = syscall.Syscall", "n, errno = 0, 0; _, _, _ = syscall.Syscall", "poll-unico"),
		m("C14", "confundir-vacio", "O", "return observacionPidfdVacioO3cM38", "return observacionPidfdTerminalNaturalO3cM38", "vacio-exacto"),
		m("C14", "confundir-POLLIN", "O", "return observacionPidfdTerminalNaturalO3cM38", "return observacionPidfdInfraestructuraO3cM38", "POLLIN-natural"),
		m("C14", "confundir-infra", "O", "return observacionPidfdInfraestructuraO3cM38\n}\n\nfunc observarPidfdO3cM38", "return observacionPidfdVacioO3cM38\n}\n\nfunc observarPidfdO3cM38", "bits-extra-infra"),
		m("C14", "variante-fuera-union", "O", "return d >= observacionControlRawO3cM38", "return d != 0", "union-cerrada"),

		m("C15", "senal-antes-CONTROL", "O", "if err := leerControlO3bM38(a.custodia); err != nil {", "_, _ = autoridadSenalO3cM38(a.custodia); if err := leerControlO3bM38(a.custodia); err != nil {", "CONTROL-antes-senal"),
		m("C15", "pidfd-antes-senal", "O", "senalVerde, err := autoridadSenalO3cM38(a.custodia)", "_ = observarPidfdO3cM38(a); senalVerde, err := autoridadSenalO3cM38(a.custodia)", "senal-antes-pidfd"),
		m("C15", "consultar-pidfd-tras-raw", "O", "return instalarObservacionO3cM38(a, observacionControlRawO3cM38)", "_ = observarPidfdO3cM38(a); return instalarObservacionO3cM38(a, observacionControlRawO3cM38)", "raw-corta-pidfd"),
		m("C15", "combinar-discriminantes", "O", "uint32(d)", "uint32(d)|uint32(observacionControlRawO3cM38)", "un-discriminante"),

		m("C16", "omitir-discriminante", "O", "CompareAndSwap(0, uint32(d))", "CompareAndSwap(0, 0)", "discriminante-obligatorio"),
		m("C16", "sustituir-discriminante", "O", "a.estado = continuacionC3ObservadoM38", "a.salida.primera.Store(uint32(observacionPidfdVacioO3cM38)); a.estado = continuacionC3ObservadoM38", "observacion-inmutable"),
		m("C16", "segundo-CAS", "O", "a.estado = continuacionC3ObservadoM38", "_ = a.salida.primera.CompareAndSwap(uint32(d), uint32(d)); a.estado = continuacionC3ObservadoM38", "CAS-unico"),
		m("C16", "fijar-causa", "O", "a.estado = continuacionC3ObservadoM38", "causa := d; _ = causa; a.estado = continuacionC3ObservadoM38", "sin-causa-funcional"),

		m("C17", "lease-owner-primero", "H", "!a.autoridad.ownerObservador.CompareAndSwap(uint32(propietarioO3cM38), uint32(propietarioO4aM38)) ||\n\t\t!a.autoridad.ownerLease.CompareAndSwap(uint32(propietarioO3cM38), uint32(propietarioO4aM38))", "!a.autoridad.ownerLease.CompareAndSwap(uint32(propietarioO3cM38), uint32(propietarioO4aM38)) ||\n\t\t!a.autoridad.ownerObservador.CompareAndSwap(uint32(propietarioO3cM38), uint32(propietarioO4aM38))", "owner-observador-primero"),
		m("C17", "revertir-1-1", "H", "a.estado = continuacionC5EntregadoM38", "a.custodia.lease.estado.Store(1); a.custodia.observador.palabra.Store(1); a.estado = continuacionC5EntregadoM38", "subyacentes-3-2"),
		m("C17", "omitir-autoridad-conjunta", "H", "a.autoridad == nil || a.autoridad.auto != a.autoridad", "false", "autoridad-conjunta"),
		m("C17", "omitir-prevalidacion", "H", "if !autoridadHandoffO3cM38(a)", "if false", "prevalidacion-conjunta"),
		m("C17", "omitir-C4T", "H", "a.estado = continuacionC4TTransfiriendoM38", "_ = a.estado", "C4T-no-entregable"),
		m("C17", "retornar-parcial", "H", "fatalHandoffO3cM38(a)\n\t}\n\ta.estado = continuacionC5EntregadoM38", "return a.salida\n\t}\n\ta.estado = continuacionC5EntregadoM38", "particion-CF-no-retorna"),

		m("C18", "exponer-PID", "A", "retornoCont int", "retornoCont int; PID int", "PID-no-expuesto"),
		m("C18", "exponer-pidfd", "A", "retornoCont int", "retornoCont int; Pidfd int", "pidfd-no-expuesto"),
		m("C18", "omitir-autoridad", "H", "salida := a.salida", "salida := a.salida; salida.autoridad = nil", "agregado-con-autoridad"),
		m("C18", "omitir-recurso", "H", "salida := a.salida", "salida := a.salida; salida.custodia = nil", "agregado-con-custodia"),
		m("C18", "omitir-plazo", "H", "salida := a.salida", "salida := a.salida; salida.finCaso = time.Time{}", "agregado-con-plazo"),
		m("C18", "omitir-raw-CONT", "H", "salida := a.salida", "salida := a.salida; salida.retornoCont = 0", "agregado-con-raw-CONT"),
		m("C18", "omitir-union", "H", "salida := a.salida", "salida := a.salida; salida.primera.Store(0)", "agregado-con-observacion"),
		m("C18", "mezclar-raw-observacion", "H", "salida := a.salida", "salida := a.salida; salida.retornoCont = int(salida.primera.Load())", "raw-separado"),
		m("C18", "incluir-ticket", "A", "retornoCont int", "retornoCont int; ticket string", "ticket-ausente"),

		m("C19", "CONT-en-retirada", "H", "uintptr(syscall.SIGKILL)", "uintptr(syscall.SIGCONT)", "retirada-KILL-individual"),
		m("C19", "flag-grupo-retirada", "H", "uintptr(syscall.SIGKILL), 0, 0", "uintptr(syscall.SIGKILL), 0, pidfdSignalGrupoO3cM38", "retirada-sin-grupo"),
		m("C19", "Wait-antes-terminalidad", "H", "fd, vivo, fiable := pidfdRetiradaO3cM38(a)", "_ = esperarConLeaseO3aM38(a.custodia); fd, vivo, fiable := pidfdRetiradaO3cM38(a)", "Wait-dominado-terminalidad"),
		m("C19", "omitir-ECHILD", "H", "if errors.Is(err, syscall.ECHILD)", "if err == nil", "ECHILD-obligatorio"),
		m("C19", "omitir-ESRCH", "H", "return errors.Is(err, syscall.ESRCH)", "return err == nil", "grupo-ESRCH"),
		m("C19", "cerrar-TERMINAL-antes", "H", "if err := esperarConLeaseO3aM38(a.custodia); err != nil {", "_ = a.custodia.terminal.Close(); if err := esperarConLeaseO3aM38(a.custodia); err != nil {", "TERMINAL-tras-Wait-ECHILD-ESRCH"),
		m("C19", "cerrar-pidfd-antes-ESRCH", "H", "if err := esperarConLeaseO3aM38(a.custodia); err != nil {", "_ = syscall.Close(fd); if err := esperarConLeaseO3aM38(a.custodia); err != nil {", "pidfd-tras-ESRCH"),
		m("C19", "reiniciar-3s", "H", "fin, ok := finRetiradaO3cM38(time.Now(), a.custodia.finBootstrap)", "fin, ok := finRetiradaO3cM38(time.Now(), time.Now().Add(duracionRetiradaO3cM38))", "retirada-no-reinicia"),

		m("C20", "cerrar-tras-CF", "H", "func fatalHandoffO3cM38(a *autoridadContinuacionO3cM38) {", "func fatalHandoffO3cM38(a *autoridadContinuacionO3cM38) { defer syscall.Close(0)", "CF-sin-cierre"),
		m("C20", "loguear-tras-CF", "H", "func fatalHandoffO3cM38(a *autoridadContinuacionO3cM38) {", "func fatalHandoffO3cM38(a *autoridadContinuacionO3cM38) { println(\"CF\")", "CF-sin-log"),
		m("C20", "retornar-tras-CF", "H", "func fatalHandoffO3cM38(a *autoridadContinuacionO3cM38) {", "func fatalHandoffO3cM38(a *autoridadContinuacionO3cM38) { if a != nil { return }", "CF-no-retorna"),
		m("C20", "retirar-post-CONT", "H", "a.estado = continuacionCFFatalM38", "a.estado = continuacionC7RetirandoM38", "post-CONT-no-retira"),

		m("C21", "waitid", "H", "pid, err = syscall.Wait4", "_, _, _ = syscall.Syscall6(syscall.SYS_WAITID, 0, 0, 0, 0, 0, 0); pid, err = syscall.Wait4", "waitid-ausente"),
		m("C21", "segundo-Wait", "H", "if err := esperarConLeaseO3aM38(a.custodia); err != nil {", "_ = esperarConLeaseO3aM38(a.custodia); if err := esperarConLeaseO3aM38(a.custodia); err != nil {", "Wait-unico"),
		m("C21", "Wait-fuera-C7", "O", "if entrada == nil || *entrada == nil", "_ = esperarConLeaseO3aM38((*entrada).custodia); if entrada == nil || *entrada == nil", "Wait-solo-C7"),
		m("C21", "CONT-extra", "O", "if entrada == nil || *entrada == nil", "_, _, _ = syscall.Syscall6(sysPidfdSendSignal, 0, uintptr(syscall.SIGCONT), 0, pidfdSignalGrupoO3cM38, 0, 0); if entrada == nil || *entrada == nil", "CONT-unico"),
		m("C21", "TERM-funcional", "O", "if entrada == nil || *entrada == nil", "_, _, _ = syscall.Syscall6(sysPidfdSendSignal, 0, uintptr(syscall.SIGTERM), 0, 0, 0, 0); if entrada == nil || *entrada == nil", "TERM-funcional-ausente"),
		m("C21", "KILL-funcional", "O", "if entrada == nil || *entrada == nil", "_, _, _ = syscall.Syscall6(sysPidfdSendSignal, 0, uintptr(syscall.SIGKILL), 0, 0, 0, 0); if entrada == nil || *entrada == nil", "KILL-funcional-ausente"),
		m("C21", "escribir-TERMINAL", "H", "c.terminal = nil", "_, _ = c.terminal.Write(nil); c.terminal = nil", "TERMINAL-no-se-escribe"),
		m("C21", "cerrar-fuera-C7", "O", "if entrada == nil || *entrada == nil", "_ = (*entrada).custodia.terminal.Close(); if entrada == nil || *entrada == nil", "TERMINAL-cierre-solo-C7"),
		m("C21", "fin-alternativo", "H", "return errRetiradaO3cM38", "return nil", "resultado-retirada-cerrado"),

		m("C22", "falsear-inventario", "Q", "[[ $estado -eq 0 && $grupo_cero_aislado == si && $fdi -eq $fdf && $hi -eq $hf && $zi -eq $zf && $gi -eq $gf && $ti -eq $tf ]]", "[[ $estado -eq 0 ]]", "inventarios-delta-cero"),
		m("C22", "aceptar-residuos", "Q", "kill -0 -- \"-$lider\" 2>/dev/null && grupo_cero_aislado=no", "kill -0 -- \"-$lider\" 2>/dev/null && grupo_cero_aislado=si", "PGID-ESRCH-residuos-cero"),
		m("C22", "aceptar-SKIP", "Q", "[[ $id == id ]] && continue", "[[ $id == id || $id == C01 ]] && continue", "sin-SKIP"),
		m("C22", "aceptar-reintento", "Q", "ejecutar \"$id\" \"$modo\" \"$prueba\" \"$oraculo\" \"$bin\"", "ejecutar \"$id\" \"$modo\" \"$prueba\" \"$oraculo\" \"$bin\" || ejecutar \"$id\" \"$modo\" \"$prueba\" \"$oraculo\" \"$bin\"", "sin-reintentos"),
		m("C22", "reutilizar-evidencia", "Q", "[[ -d $target && ! -e $evidencia ]]", "[[ -d $target && -d $evidencia ]]", "evidencia-nueva"),

		m("C23", "segundo-owner", "A", "retornoCont int", "retornoCont int; ownerAjeno atomic.Uint32", "autoridad-conjunta-unica"),
		m("C23", "ciclo", "A", "auto            *autoridadCustodiaO3cM38", "auto *autoridadCustodiaO3cM38; ciclo *agregadoO4aM38", "ownership-aciclico"),
		m("C23", "hook", "A", "func fatalO3cM38()", "var hookO3c = func() {}; func fatalO3cM38()", "sin-hook"),
		m("C23", "callback", "A", "func fatalO3cM38()", "var callbackO3c func(); func fatalO3cM38()", "sin-callback"),
		m("C23", "goroutine", "A", "func fatalO3cM38()", "func init(){ go func(){}() }; func fatalO3cM38()", "sin-goroutine"),
		m("C23", "global", "A", "func fatalO3cM38()", "var autoridadGlobalO3c *autoridadContinuacionO3cM38; func fatalO3cM38()", "sin-global"),
		m("C23", "init", "A", "func fatalO3cM38()", "func init(){}; func fatalO3cM38()", "sin-init"),

		m("C24", "limite-1024", "B", "var buffer [1024]byte", "var buffer [1025]byte", "limite-1024"),
		m("C24", "limite-4", "B", "lecturas < 4", "lecturas < 5", "limite-4"),
		m("C24", "limite-4096", "B", "total < 4096", "total < 4097", "limite-4096"),
		m("C24", "limite-8", "B", "interrupciones <= 8", "interrupciones <= 9", "limite-8"),
		m("C24", "plazo-180", "C", "180 * time.Second", "181 * time.Second", "plazo-180s"),
		m("C24", "retirada-3", "H", "3 * time.Second", "4 * time.Second", "retirada-3s"),
		m("C24", "minimo-FD", "G", "minFDDuplicadoM38      = 10", "minFDDuplicadoM38      = 9", "FD-minimo"),
		m("C24", "parada-650", "A", "package main", "package main\n\n// "+strings.Repeat("linea-mutante\n// ", 651), "parada-650"),
	}
	for i := range c {
		c[i].id = fmt.Sprintf("C%03d", i+1)
	}
	return c
}

func sha(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func digestObjetivos() string {
	keys := make([]string, 0, len(hashesBase))
	for k := range hashesBase {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s\t%s\t%s\n", k, fuentes[k], hashesBase[k])
	}
	return sha([]byte(b.String()))
}

func ejecutarGrupo(dir, cache string, argv ...string) ([]byte, error, time.Duration) {
	inicio := time.Now()
	dir, _ = filepath.Abs(dir)
	ctx, cancelar := context.WithTimeout(context.Background(), 65*time.Second)
	defer cancelar()
	c := exec.CommandContext(ctx, argv[0], argv[1:]...)
	c.Dir = dir
	c.Env = []string{"CGO_ENABLED=0", "GOTOOLCHAIN=local", "GOCACHE=" + cache, "HOME=/nonexistent", "PATH=/usr/bin:/bin"}
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cancel = func() error {
		if c.Process == nil {
			return nil
		}
		err := syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		return err
	}
	c.WaitDelay = 2 * time.Second
	out, err := c.CombinedOutput()
	if c.Process != nil {
		if ctx.Err() != nil {
			_ = syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
		}
		limite := time.Now().Add(time.Second)
		for {
			e := syscall.Kill(-c.Process.Pid, 0)
			if errors.Is(e, syscall.ESRCH) {
				break
			}
			if time.Now().After(limite) {
				return out, fmt.Errorf("PGID residual: %v", e), time.Since(inicio)
			}
			time.Sleep(time.Millisecond)
		}
	}
	if ctx.Err() != nil {
		return out, fmt.Errorf("infraestructura timeout: %w", ctx.Err()), time.Since(inicio)
	}
	return out, err, time.Since(inicio)
}

func gruposAST(familia string) []string {
	switch familia {
	case "C01", "C02":
		return []string{"entrada_B5_readonly_consumo_lineal", "autoridad_CAS_owners_aciclica"}
	case "C05", "C06", "C07":
		return []string{"revalidacion_final_y_lease", "entrada_B5_readonly_consumo_lineal", "APIs_prohibidas_ausentes"}
	case "C03":
		return []string{"revalidacion_final_y_lease", "autoridad_CAS_owners_aciclica", "entrada_B5_readonly_consumo_lineal", "APIs_prohibidas_ausentes"}
	case "C04":
		return []string{"revalidacion_final_y_lease", "observacion_union_cerrada_y_precedencia", "APIs_prohibidas_ausentes"}
	case "C08":
		return []string{"revalidacion_final_y_lease", "marca_CONT_unicos_y_ordenados", "APIs_prohibidas_ausentes"}
	case "C09", "C10", "C11", "C12", "C13":
		return []string{"marca_CONT_unicos_y_ordenados", "APIs_prohibidas_ausentes"}
	case "C14", "C15", "C16":
		return []string{"observacion_union_cerrada_y_precedencia", "APIs_prohibidas_ausentes"}
	case "C17", "C18", "C19", "C20":
		return []string{"handoff_conjunto_y_retirada_C7", "O4_opaco_sin_efectos", "autoridad_CAS_owners_aciclica", "APIs_prohibidas_ausentes"}
	case "C21":
		return []string{"APIs_prohibidas_ausentes", "handoff_conjunto_y_retirada_C7"}
	case "C22":
		return []string{"DAG_sin_pruebas_productivas", "APIs_prohibidas_ausentes"}
	case "C23":
		return []string{"autoridad_CAS_owners_aciclica", "O4_opaco_sin_efectos", "DAG_sin_pruebas_productivas", "APIs_prohibidas_ausentes"}
	case "C24":
		return []string{"revalidacion_final_y_lease", "marca_CONT_unicos_y_ordenados", "handoff_conjunto_y_retirada_C7", "APIs_prohibidas_ausentes"}
	}
	return nil
}

func muerteASTCausal(x mutante, salida []byte, err error) bool {
	if err == nil || !bytes.Contains(salida, []byte("ast_o3c_p6=NO_GO ")) || bytes.Contains(salida, []byte("parsear ")) || bytes.Contains(salida, []byte("tipar: ")) {
		return false
	}
	if x.familia == "C23" && x.alternativa == "ciclo" && bytes.Contains(salida, []byte("ast_o3c_p6=NO_GO DAG cíclico:")) {
		return true
	}
	for _, grupo := range gruposAST(x.familia) {
		if bytes.Contains(salida, []byte("ast_o3c_p6=NO_GO "+grupo+":")) {
			return true
		}
	}
	return false
}

func requiereASTDirecto(x mutante) bool {
	if x.familia == "C10" || x.familia == "C11" || x.familia == "C20" || x.familia == "C21" {
		return true
	}
	if x.familia == "C19" && x.alternativa == "Wait-antes-terminalidad" {
		return true
	}
	return x.familia == "C22" && x.alternativa != "SKIP"
}

func esperaBF65(x mutante) bool {
	if x.familia != "C01" {
		return false
	}
	switch x.alternativa {
	case "reordenar-CAS", "aceptar-alias", "aceptar-clon", "aceptar-reuso-antes-C5", "aceptar-replay-despues-C5":
		return true
	}
	return false
}

func falloPruebaCausal(x mutante, salida []byte, err error) bool {
	e, ok := err.(*exec.ExitError)
	if !ok {
		return false
	}
	if esperaBF65(x) {
		return e.ExitCode() == 65 && len(salida) == 0
	}
	return e.ExitCode() == 1 && bytes.Contains(salida, []byte("--- FAIL: Test")) && bytes.Contains(salida, []byte("O3c"))
}

func muerteMetaCausal(x mutante, ruta string) bool {
	if x.familia == "C22" {
		b, err := os.ReadFile(ruta)
		if err != nil {
			return false
		}
		return validarConductorO3cP6(b) != nil
	}
	if x.familia == "C24" {
		b, err := os.ReadFile(ruta)
		if err != nil {
			return false
		}
		if x.alternativa == "parada-650" {
			return bytes.Count(b, []byte{'\n'}) > 650
		}
		literales := map[string]string{"limite-1024": "1025", "limite-4": "5", "limite-4096": "4097", "limite-8": "9", "minimo-FD": "9"}
		esperado, ok := literales[x.alternativa]
		if !ok { /* plazo y retirada pertenecen al AST O3c */
		} else {
			hallado := false
			f, err := parser.ParseFile(token.NewFileSet(), ruta, b, 0)
			if err != nil {
				return false
			}
			ast.Inspect(f, func(n ast.Node) bool {
				if v, ok := n.(*ast.BasicLit); ok && v.Value == esperado {
					hallado = true
				}
				return true
			})
			return hallado
		}
	}
	f, err := parser.ParseFile(token.NewFileSet(), ruta, nil, 0)
	if err != nil {
		return false
	}
	if x.familia == "C23" && (x.alternativa == "hook" || x.alternativa == "callback" || x.alternativa == "global" || x.alternativa == "goroutine" || x.alternativa == "init") {
		variables := 0
		tieneInit, tieneGo := false, false
		for _, d := range f.Decls {
			switch n := d.(type) {
			case *ast.GenDecl:
				if n.Tok == token.VAR {
					for _, s := range n.Specs {
						variables += len(s.(*ast.ValueSpec).Names)
					}
				}
			case *ast.FuncDecl:
				if n.Name.Name == "init" {
					tieneInit = true
				}
				ast.Inspect(n, func(n ast.Node) bool {
					if _, ok := n.(*ast.GoStmt); ok {
						tieneGo = true
					}
					return true
				})
			}
		}
		return variables > 3 || tieneInit || tieneGo
	}
	return false
}

// validarConductorO3cP6 es un meta-oráculo único e independiente del catálogo:
// acredita las cinco propiedades del conductor congelado, no la presencia del
// patrón usado por una transformación concreta.
func validarConductorO3cP6(b []byte) error {
	s := string(b)
	propiedades := []struct{ nombre, clausula string }{
		{"evidencia-nueva", "[[ -d $target && ! -e $evidencia ]]"},
		{"inventario-delta-cero", "[[ $estado -eq 0 && $grupo_cero_aislado == si && $fdi -eq $fdf && $hi -eq $hf && $zi -eq $zf && $gi -eq $gf && $ti -eq $tf ]]"},
		{"PGID-ESRCH", "kill -0 -- \"-$lider\" 2>/dev/null && grupo_cero_aislado=no"},
		{"cabecera-no-SKIP", "[[ $id == id ]] && continue"},
		{"un-intento", "    ejecutar \"$id\" \"$modo\" \"$prueba\" \"$oraculo\" \"$bin\"\n"},
	}
	for _, p := range propiedades {
		if strings.Count(s, p.clausula) != 1 {
			return fmt.Errorf("meta_conductor=%s cardinalidad=%d", p.nombre, strings.Count(s, p.clausula))
		}
	}
	if strings.Contains(s, "t.Skip(") || strings.Contains(s, "|| ejecutar \"$id\"") {
		return errors.New("meta_conductor=SKIP_o_reintento")
	}
	return nil
}

func archive(repo, dir string) error {
	a := exec.Command("git", "archive", "--format=tar", "HEAD")
	a.Dir = repo
	t := exec.Command("tar", "-x", "-C", dir)
	p, err := a.StdoutPipe()
	if err != nil {
		return err
	}
	t.Stdin = p
	if err = t.Start(); err != nil {
		return err
	}
	if err = a.Run(); err != nil {
		return err
	}
	return t.Wait()
}

func ejecutar(repo, out string, desde, hasta int, soloCompilar bool) error {
	for k, esperado := range hashesBase {
		b, err := os.ReadFile(filepath.Join(repo, paquete, fuentes[k]))
		if err != nil || sha(b) != esperado {
			return fmt.Errorf("fuente base %s no coincide con %s", k, base)
		}
	}
	lista := catalogo()
	if desde < 1 || hasta < desde || hasta > len(lista) {
		return fmt.Errorf("rango invalido")
	}
	lista = lista[desde-1 : hasta]
	destinoRel, _ := filepath.Rel(repo, out)
	if (desde != 1 || hasta != len(catalogo())) && !strings.HasPrefix(destinoRel, "..") && strings.HasPrefix(filepath.ToSlash(destinoRel), "tools/o3c_p6_mutantes/evidencia") {
		return errors.New("un rango parcial no puede sobrescribir evidencia final")
	}
	tmp, err := os.MkdirTemp("", "o3c-p6-mutantes-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	cache := filepath.Join(tmp, "gocache")
	goInicial, err := exec.LookPath("go")
	if err != nil {
		return err
	}
	resolver := exec.Command(goInicial, "env", "GOROOT")
	resolver.Dir = repo
	goRoot, err := resolver.Output()
	if err != nil {
		return fmt.Errorf("resolver toolchain: %w", err)
	}
	goBin := filepath.Join(strings.TrimSpace(string(goRoot)), "bin", "go")
	if _, err := os.Stat(goBin); err != nil {
		return fmt.Errorf("toolchain no ejecutable: %w", err)
	}
	versionGo, err := exec.Command(goBin, "version").Output()
	if err != nil || !bytes.Contains(versionGo, []byte(runtime.Version())) {
		return fmt.Errorf("toolchain distinta de %s: %s", runtime.Version(), versionGo)
	}
	var resultados, catalogoTSV, fuentesTSV strings.Builder
	catalogoTSV.WriteString("id\tfamilia\talternativa\tarchivo\tantes_hex\tdespues_hex\toraculo\n")
	resultados.WriteString("id\tfamilia\talternativa\tcompilacion\tmuerte_causal\tduracion_ns\tsalida_sha256\n")
	fuentesTSV.WriteString("clave\tarchivo\tsha256\n")
	for _, x := range lista {
		dir := filepath.Join(tmp, x.id)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		if err := archive(repo, dir); err != nil {
			return err
		}
		ruta := filepath.Join(dir, paquete, fuentes[x.fuente])
		if x.fuente == "Q" {
			ruta = filepath.Join(dir, fuentes[x.fuente])
			if err := os.MkdirAll(filepath.Dir(ruta), 0700); err != nil {
				return err
			}
			origen, err := os.ReadFile(filepath.Join(repo, fuentes[x.fuente]))
			if err != nil {
				return err
			}
			if err := os.WriteFile(ruta, origen, 0700); err != nil {
				return err
			}
		}
		original, err := os.ReadFile(ruta)
		if err != nil {
			return err
		}
		if bytes.Count(original, []byte(x.antes)) != 1 {
			return fmt.Errorf("%s cardinalidad anterior=%d", x.id, bytes.Count(original, []byte(x.antes)))
		}
		mutado := bytes.Replace(original, []byte(x.antes), []byte(x.despues), 1)
		if x.fuente != "Q" {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, ruta, mutado, parser.AllErrors)
			if err != nil {
				return fmt.Errorf("%s AST: %w", x.id, err)
			}
			mutado, err = format.Source(mutado)
			if err != nil || f == nil {
				return fmt.Errorf("%s gofmt: %v", x.id, err)
			}
		}
		if err = os.WriteFile(ruta, mutado, 0600); err != nil {
			return err
		}
		pkg := filepath.Join(dir, paquete)
		if x.fuente == "Q" {
			bashOut, bashErr, dur := ejecutarGrupo(filepath.Dir(ruta), cache, "/bin/bash", "-n", ruta)
			if bashErr != nil {
				return fmt.Errorf("%s no compila bash: %v %s", x.id, bashErr, bashOut)
			}
			if soloCompilar {
				fmt.Fprintf(&catalogoTSV, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", x.id, x.familia, x.alternativa, fuentes[x.fuente], hex.EncodeToString([]byte(x.antes)), hex.EncodeToString([]byte(x.despues)), x.oraculo)
				fmt.Fprintf(&resultados, "%s\t%s\t%s\tCOMPILA\tNO-EJECUTADO-PREFLIGHT\t%d\t%s\n", x.id, x.familia, x.alternativa, dur.Nanoseconds(), sha(bashOut))
				fmt.Printf("%s COMPILA-VET\n", x.id)
				continue
			}
			if !muerteMetaCausal(x, ruta) {
				return fmt.Errorf("%s sobrevivio meta conductor", x.id)
			}
			fmt.Fprintf(&catalogoTSV, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", x.id, x.familia, x.alternativa, fuentes[x.fuente], hex.EncodeToString([]byte(x.antes)), hex.EncodeToString([]byte(x.despues)), x.oraculo)
			fmt.Fprintf(&resultados, "%s\t%s\t%s\tCOMPILA\tMUERTO-META-CONDUCTOR-CAUSAL\t%d\t%s\n", x.id, x.familia, x.alternativa, dur.Nanoseconds(), sha(bashOut))
			fmt.Printf("%s %s/%s COMPILA MUERTO-META-CONDUCTOR-CAUSAL\n", x.id, x.familia, x.alternativa)
			continue
		}
		gs, _ := filepath.Glob(filepath.Join(pkg, "*.go"))
		args := []string{"test"}
		for _, g := range gs {
			if filepath.Base(g) != "capturar_snapshot_fuente_corporativa_contexto_actor_v1.go" {
				args = append(args, filepath.Base(g))
			}
		}
		vetOut, vetErr, _ := ejecutarGrupo(pkg, cache, append([]string{goBin, "vet"}, args[1:]...)...)
		if vetErr != nil {
			return fmt.Errorf("%s no pasa vet: %v %s", x.id, vetErr, vetOut)
		}
		bin := filepath.Join(dir, "mutante.test")
		buildArgs := append([]string{goBin}, args...)
		buildArgs = append(buildArgs, "-c", "-o", bin)
		buildOut, buildErr, buildDur := ejecutarGrupo(pkg, cache, buildArgs...)
		if buildErr != nil {
			return fmt.Errorf("%s no compila: %v %s", x.id, buildErr, buildOut)
		}
		if soloCompilar {
			fmt.Fprintf(&catalogoTSV, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", x.id, x.familia, x.alternativa, fuentes[x.fuente], hex.EncodeToString([]byte(x.antes)), hex.EncodeToString([]byte(x.despues)), x.oraculo)
			fmt.Fprintf(&resultados, "%s\t%s\t%s\tCOMPILA\tNO-EJECUTADO-PREFLIGHT\t%d\t%s\n", x.id, x.familia, x.alternativa, buildDur.Nanoseconds(), sha(append(vetOut, buildOut...)))
			fmt.Printf("%s COMPILA-VET\n", x.id)
			continue
		}
		var testOut []byte
		var testErr error
		var dur time.Duration
		if !requiereASTDirecto(x) || x.familia == "C22" && x.alternativa == "SKIP" {
			argsPrueba := []string{"/usr/bin/timeout", "--signal=KILL", "20", bin, "-test.run=O3c", "-test.count=1", "-test.timeout=18s"}
			if x.familia == "C22" && x.alternativa == "SKIP" {
				argsPrueba = append(argsPrueba[:4], append([]string{"-test.v"}, argsPrueba[4:]...)...)
			}
			testOut, testErr, dur = ejecutarGrupo(pkg, cache, argsPrueba...)
		}
		muerte := "PRUEBA-CAUSAL"
		if muerteMetaCausal(x, ruta) {
			muerte = "META-EVIDENCIA-NUEVA-CAUSAL"
		} else if testErr == nil && x.familia == "C22" && x.alternativa == "SKIP" && bytes.Contains(testOut, []byte("--- SKIP:")) {
			muerte = "CONDUCTOR-SIN-SKIP-CAUSAL"
		} else if requiereASTDirecto(x) || testErr == nil {
			astOut, astErr, astDur := ejecutarGrupo(repo, cache, "/usr/bin/timeout", "--signal=KILL", "30", goBin, "run", "./tools/o3c_p6_ast", "-permitir-sha", "-dir", pkg)
			testOut = append(testOut, astOut...)
			dur += astDur
			if !muerteASTCausal(x, astOut, astErr) {
				return fmt.Errorf("%s sobrevivio o fallo AST no causal: err=%v salida=%s", x.id, astErr, astOut)
			}
			muerte = "AST-TIPOS-DAG-CAUSAL"
		} else if !falloPruebaCausal(x, testOut, testErr) {
			return fmt.Errorf("%s fallo de prueba no causal/infraestructura: err=%v salida=%s", x.id, testErr, testOut)
		}
		fmt.Fprintf(&catalogoTSV, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", x.id, x.familia, x.alternativa, fuentes[x.fuente], hex.EncodeToString([]byte(x.antes)), hex.EncodeToString([]byte(x.despues)), x.oraculo)
		fmt.Fprintf(&resultados, "%s\t%s\t%s\tCOMPILA\tMUERTO-%s\t%d\t%s\n", x.id, x.familia, x.alternativa, muerte, dur.Nanoseconds(), sha(testOut))
		fmt.Printf("%s %s/%s COMPILA MUERTO-%s\n", x.id, x.familia, x.alternativa, muerte)
	}
	keys := make([]string, 0, len(fuentes))
	for k := range fuentes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		var b []byte
		var err error
		if k == "Q" {
			b, err = os.ReadFile(filepath.Join(repo, fuentes[k]))
		} else {
			b, err = os.ReadFile(filepath.Join(repo, paquete, fuentes[k]))
		}
		if err != nil {
			return err
		}
		fmt.Fprintf(&fuentesTSV, "%s\t%s\t%s\n", k, fuentes[k], sha(b))
	}
	if err = os.MkdirAll(out, 0755); err != nil {
		return err
	}
	archivos := map[string][]byte{"catalogo.tsv": []byte(catalogoTSV.String()), "resultados.tsv": []byte(resultados.String()), "fuentes.tsv": []byte(fuentesTSV.String())}
	for n, b := range archivos {
		if err = os.WriteFile(filepath.Join(out, n), b, 0644); err != nil {
			return err
		}
	}
	mainBytes, err := os.ReadFile(filepath.Join(repo, "tools/o3c_p6_mutantes/main.go"))
	if err != nil {
		return err
	}
	fusionBytes, err := os.ReadFile(filepath.Join(repo, "tools/o3c_p6_mutantes/fusion.go"))
	if err != nil {
		return err
	}
	testBytes, err := os.ReadFile(filepath.Join(repo, "tools/o3c_p6_mutantes/main_test.go"))
	if err != nil {
		return err
	}
	readmeBytes, err := os.ReadFile(filepath.Join(repo, "tools/o3c_p6_mutantes/README.md"))
	if err != nil {
		return err
	}
	astMain, err := os.ReadFile(filepath.Join(repo, "tools/o3c_p6_ast/main.go"))
	if err != nil {
		return err
	}
	astTest, err := os.ReadFile(filepath.Join(repo, "tools/o3c_p6_ast/main_test.go"))
	if err != nil {
		return err
	}
	astInvariantes, err := os.ReadFile(filepath.Join(repo, "tools/o3c_p6_ast/invariantes.go"))
	if err != nil {
		return err
	}
	astREADME, err := os.ReadFile(filepath.Join(repo, "tools/o3c_p6_ast/README.md"))
	if err != nil {
		return err
	}
	ejecutable, err := os.Executable()
	if err != nil {
		return err
	}
	binBytes, err := os.ReadFile(ejecutable)
	if err != nil {
		return err
	}
	astRetirada, err := os.ReadFile(filepath.Join(repo, "tools/o3c_p6_ast/retirada.go"))
	if err != nil {
		return err
	}
	astSeguridad, err := os.ReadFile(filepath.Join(repo, "tools/o3c_p6_ast/seguridad.go"))
	if err != nil {
		return err
	}
	conductor, err := os.ReadFile(filepath.Join(repo, fuentes["Q"]))
	if err != nil {
		return err
	}
	modo, estadoLote := "mutantes", fmt.Sprintf("mutantes_lote\t%d/%d compilables-y-muertos", len(lista), hasta-desde+1)
	if soloCompilar {
		modo, estadoLote = "preflight", fmt.Sprintf("solo_compilacion\t%d/%d", len(lista), hasta-desde+1)
	}
	manifest := fmt.Sprintf("base_producto\t%s\nfuentes_objetivo_sha256\t%s\ntoolchain\t%s\ncatalogo_total\t%d\nmodo\t%s\nrango\t%d-%d\n%s\npgid\tESRCH por build/vet/oraculo\nresiduos\tcero\nrunner_sha256\t%s\nrunner_fusion_sha256\t%s\nrunner_test_sha256\t%s\nrunner_readme_sha256\t%s\nrunner_bin_sha256\t%s\nast_main_sha256\t%s\nast_invariantes_sha256\t%s\nast_retirada_sha256\t%s\nast_seguridad_sha256\t%s\nast_test_sha256\t%s\nast_readme_sha256\t%s\nconductor_sha256\t%s\n", base, digestObjetivos(), runtime.Version(), len(catalogo()), modo, desde, hasta, estadoLote, sha(mainBytes), sha(fusionBytes), sha(testBytes), sha(readmeBytes), sha(binBytes), sha(astMain), sha(astInvariantes), sha(astRetirada), sha(astSeguridad), sha(astTest), sha(astREADME), sha(conductor))
	if err := os.WriteFile(filepath.Join(out, "manifiesto.tsv"), []byte(manifest), 0644); err != nil {
		return err
	}
	nombres := []string{"catalogo.tsv", "fuentes.tsv", "manifiesto.tsv", "resultados.tsv"}
	var sumas strings.Builder
	for _, n := range nombres {
		b, err := os.ReadFile(filepath.Join(out, n))
		if err != nil {
			return err
		}
		fmt.Fprintf(&sumas, "%s  tools/o3c_p6_mutantes/evidencia/%s\n", sha(b), n)
	}
	return os.WriteFile(filepath.Join(out, "SHA256SUMS"), []byte(sumas.String()), 0644)
}

func main() {
	info, ok := debug.ReadBuildInfo()
	trimpath := false
	if ok {
		for _, s := range info.Settings {
			trimpath = trimpath || s.Key == "-trimpath" && s.Value == "true"
		}
	}
	if !trimpath {
		fmt.Fprintln(os.Stderr, "binario no canonico: exige go build -trimpath")
		os.Exit(2)
	}
	repo := flag.String("repo", ".", "checkout limpio")
	out := flag.String("out", "tools/o3c_p6_mutantes/evidencia", "evidencia")
	desde := flag.Int("desde", 1, "primero")
	hasta := flag.Int("hasta", len(catalogo()), "ultimo")
	fusion := flag.String("fusionar", "", "directorio que contiene lotes congelados")
	soloCompilar := flag.Bool("solo-compilar", false, "preflight gofmt, compilacion tipada y vet sin oraculos")
	flag.Parse()
	destino := *out
	if !filepath.IsAbs(destino) {
		destino = filepath.Join(*repo, destino)
	}
	var err error
	if *fusion != "" {
		err = fusionar(*repo, *fusion, destino)
	} else {
		err = ejecutar(*repo, destino, *desde, *hasta, *soloCompilar)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
