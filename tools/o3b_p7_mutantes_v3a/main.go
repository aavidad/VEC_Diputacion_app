package main

import (
	"bytes"
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
	"sort"
	"strings"
	"syscall"
	"time"
)

const base = "d9f8aeb547f5d1b3b9ab3eb786382f78ef964e28"
const paquete = "deploy/postgresql/autorizacion_atestada_v3/pruebas_sql"

var archivos = map[string]string{
	"autoridad": "captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad.go",
	"barrera":   "captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_barrera.go",
	"ticket":    "captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_ticket.go",
	"stop":      "captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_stop.go",
	"identidad": "captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_identidad.go",
	"handoff":   "captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff.go",
}

type mutante struct{ id, familia, alternativa, fuente, antes, despues, regla string }

func m(familia, alternativa, fuente, antes, despues, regla string) mutante {
	return mutante{familia: familia, alternativa: alternativa, fuente: fuente, antes: antes, despues: despues, regla: regla}
}

// catalogo contiene una transformación viva y distinta por alternativa. Las
// aserciones no comparan hashes: vuelven a parsear el programa y acreditan la
// forma contractual concreta que la transformación rompe.
func catalogo() []mutante {
	c := []mutante{
		m("B01", "nula", "autoridad", "entrada == nil || *entrada == nil", "entrada == nil && *entrada == nil", "guardia-nula-disyuntiva"),
		m("B01", "no-A6", "autoridad", "!c.autoridad.es(arranqueA6EntregadoM38)", "false", "discriminante-A6"),
		m("B01", "alias", "autoridad", "*entrada = nil", "_ = entrada", "alias-consumido-anulado"),
		m("B01", "consumida", "autoridad", "!agregado.custodia.consumida.CompareAndSwap(1, 2)", "agregado.custodia.consumida.CompareAndSwap(1, 2)", "CAS-1-a-2"),
		m("B02", "autoidentidad", "autoridad", "l.auto == l", "true", "lease-autoidentidad"),
		m("B02", "registro", "autoridad", "l.registro != nil", "true", "lease-registro-presente"),
		m("B02", "pertenencia", "autoridad", "l.registro.leases[l] == l.generacion", "true", "lease-pertenencia"),
		m("B02", "generacion", "autoridad", "o.registro.observadores[o] != o.generacion", "false", "observador-generacion"),
		m("B02", "TID", "autoridad", "l.registro.tid == tid && l.tid == tid", "true", "lease-TID-doble"),
		m("B02", "CAS", "autoridad", "CompareAndSwap(1, 2)", "CompareAndSwap(1, 1)", "consumo-CAS"),
		m("B03", "primera-lectura", "barrera", "actual != c.baselineSenal || senal != 0", "actual != actual || senal != 0", "baseline-primera"),
		m("B03", "ultima-lectura", "handoff", "actual != a.custodia.baselineSenal || senal != 0", "actual != actual || senal != 0", "baseline-ultima"),
		m("B03", "reset", "barrera", "actual, senal, valido := c.observador.observar()", "c.observador.palabra.Store(c.baselineSenal); actual, senal, valido := c.observador.observar()", "contador-no-reset"),
		m("B03", "sustituir-causa", "barrera", "if !valido || actual != c.baselineSenal || senal != 0 {\n\t\treturn barreraO3bSenalM38, nil\n\t}", "if !valido || actual != c.baselineSenal || senal != 0 {\n\t\treturn barreraO3bBootstrapM38, nil\n\t}", "primera-causa-senal"),
		m("B04", "CONTROL-senal", "stop", "if err := leerControlO3bM38(a.custodia); err != nil {", "_, _, _ = a.custodia.observador.observar(); if err := leerControlO3bM38(a.custodia); err != nil {", "CONTROL-antes-senal"),
		m("B04", "senal-bootstrap", "barrera", "func revalidarAutoridadBarreraO3bM38(c *custodiaO3aM38) (causaBarreraO3bM38, error) {", "func revalidarAutoridadBarreraO3bM38(c *custodiaO3aM38) (causaBarreraO3bM38, error) { if !bootstrapDisponibleO3bM38(c.finBootstrap, time.Now()) { return barreraO3bBootstrapM38, nil }", "senal-antes-bootstrap"),
		m("B04", "bootstrap-pidfd", "barrera", "if !bootstrapDisponibleO3bM38(c.finBootstrap, time.Now()) {", "if _, _, _ = acreditarPidfdBarreraO3bM38(c); !bootstrapDisponibleO3bM38(c.finBootstrap, time.Now()) {", "bootstrap-antes-pidfd"),
		m("B04", "pidfd-STOP", "stop", "time.Sleep(100 * time.Millisecond)", "_, _ = leerStatStopO3bM38(a.custodia); time.Sleep(100 * time.Millisecond)", "pidfd-antes-STOP"),
		m("B04", "STOP-identidad", "identidad", "runtime.Gosched()", "_, _ = leerStatStopO3bM38(a.custodia); runtime.Gosched()", "precedencia-antes-identidad"),
		m("B04", "verde-prematuro", "barrera", "a.estado = capturaB1BarreraVerdeM38", "a.estado = capturaB2TicketCerradoM38", "verde-solo-B1"),
		m("B05", "parcial", "barrera", "consumidos != n", "consumidos != consumidos", "CONTROL-consumo-completo"),
		m("B05", "EOF", "barrera", "if err != nil || n == 0", "if err != nil && n == 0", "EOF-no-verde"),
		m("B05", "framing", "barrera", "fallo != nil || consumidos != n", "fallo != fallo || consumidos != n", "framing-no-verde"),
		m("B05", "EINTR", "barrera", "interrupciones++", "interrupciones--", "EINTR-acotado"),
		m("B05", "presupuesto", "barrera", "lecturas < 4 && total < 4096", "lecturas <= 4 && total <= 4096", "presupuesto-estricto"),
		m("B06", "reiniciar-bootstrap", "stop", "fin := finStopO3bM38(time.Now(), a.custodia.finBootstrap)", "a.custodia.finBootstrap = time.Now().Add(time.Second); fin := finStopO3bM38(time.Now(), a.custodia.finBootstrap)", "bootstrap-no-reinicia"),
		m("B06", "menos-de-1s", "barrera", "!fin.Before(ahora.Add(reservaO3bO3cM38))", "fin.After(ahora)", "reserva-minima"),
		m("B06", "reserva-exclusiva", "barrera", "reservaO3bO3cM38", "2 * reservaO3bO3cM38", "reserva-conjunta-literal"),
		m("B06", "plazo-STOP", "stop", "ahora.Add(time.Second)", "ahora.Add(2 * time.Second)", "STOP-un-segundo"),
		m("B07", "omitir-primario", "barrera", "fiablePrimario := errPrimario == nil && errVivoPrimario == nil && vivoPrimario", "_ = vivoPrimario; fiablePrimario := true", "dos-pidfd-sondeados"),
		m("B07", "poll-adverso", "barrera", "return false, errProcesoO3aM38", "return true, nil", "poll-adverso-rechazado"),
		m("B07", "cuarta-referencia", "handoff", "if referencias != 3", "if referencias != 4", "tres-referencias"),
		m("B08", "dup", "barrera", "sonda := pollfdO3aM38{", "_, _ = syscall.Dup(fd); sonda := pollfdO3aM38{", "sin-dup"),
		m("B08", "pidfd_open", "barrera", "sonda := pollfdO3aM38{", "_, _, _ = syscall.Syscall(434, uintptr(c.cmd.Process.Pid), 0, 0); sonda := pollfdO3aM38{", "sin-pidfd-open"),
		m("B08", "os-NewFile", "autoridad", "controlFD, terminalFD, ticketFD :=", "_ = os.NewFile(uintptr(c.pidfdPrimario), \"pidfd\"); controlFD, terminalFD, ticketFD :=", "sin-os-NewFile"),
		m("B08", "reabrir", "stop", "fd := -1", "fd := c.pidfdPrimario", "sin-reabrir-pidfd"),
		m("B09", "PID-recibido", "ticket", "int64(os.Getpid())", "int64(os.Getpid() - os.Getpid() + a.custodia.ppid)", "PID-supervisor-local"),
		m("B09", "PID-Bash", "ticket", "int64(os.Getpid())", "int64(os.Getpid() - os.Getpid() + a.custodia.cmd.Process.Pid)", "PID-no-Bash"),
		m("B09", "signo-o-cero", "ticket", "trama := strconv.AppendInt(preparado.trama[:0], int64(os.Getpid()), 10)", "trama := strconv.AppendInt(preparado.trama[:0], int64(os.Getpid()), 10); trama = append(trama, '+', '0')", "PID-decimal-canonico"),
		m("B09", "separador", "ticket", "append(trama, '|')", "append(trama, ':')", "separador-pipe"),
		m("B10", "alterar", "ticket", "append(trama, ticket...)", "append(trama, (ticket + \"X\")...)", "ticket-inalterado"),
		m("B10", "truncar", "ticket", "append(trama, ticket...)", "append(trama, ticket[:len(ticket)-1]...)", "ticket-no-truncado"),
		m("B10", "normalizar", "ticket", "ticket := c.control.recepcion.sobre.ticket", "ticket := c.control.recepcion.sobre.ticket; if len(ticket) > 0 && ticket[0] == ' ' { ticket = ticket[1:] }", "ticket-no-normalizado"),
		m("B10", "registrar", "ticket", "return ticket, true", "println(\"ticket\", ticket); return ticket, true", "ticket-no-registrado"),
		m("B10", "exponer", "ticket", "return ticket, true", "panic(ticket)", "ticket-no-expuesto"),
		m("B11", "omitir-LF", "ticket", "append(trama, '\\n')", "trama", "LF-obligatorio"),
		m("B11", "duplicar-trama", "ticket", "preparado.longitud = len(trama)", "trama = append(trama, trama...); preparado.longitud = len(trama)", "una-trama"),
		m("B11", "bytes-extra", "ticket", "append(trama, '\\n')", "append(trama, '\\n', 0)", "sin-byte-extra"),
		m("B11", "reiniciar-escritura", "ticket", "offset += avance", "if avance > 0 { offset = 0 }", "offset-monotono"),
		m("B12", "Write-cero", "ticket", "if n == 0", "if n < 0", "write-cero-falla"),
		m("B12", "Write-corto", "ticket", "if n == 0 {\n\t\treturn 0, interrupciones, errEscrituraO3bM38\n\t}\n\treturn n, interrupciones, nil", "if n == 0 {\n\t\treturn 0, interrupciones, errEscrituraO3bM38\n\t}\n\treturn restantes, interrupciones, nil", "write-corto-avanza-n"),
		m("B12", "noveno-EINTR", "ticket", "interrupciones > 8", "interrupciones > 9", "ocho-EINTR"),
		m("B12", "EPIPE", "ticket", "if err != nil {", "if err != nil && !errors.Is(err, syscall.EPIPE) {", "EPIPE-falla"),
		m("B12", "cierre-fallido", "ticket", "if errCierre != nil {", "if false && errCierre != nil {", "cierre-fallido-falla"),
		m("B13", "omitir", "ticket", "errCierre := archivo.Close()", "_ = archivo; errCierre := error(nil)", "close-unico"),
		m("B13", "anticipar", "ticket", "if err := escribirTicketO3bM38(a); err != nil {", "_ = cerrarTicketO3bM38(a); if err := escribirTicketO3bM38(a); err != nil {", "close-posterior-write"),
		m("B13", "duplicar", "ticket", "errCierre := archivo.Close()", "errCierre := archivo.Close(); _ = archivo.Close()", "close-no-duplica"),
		m("B13", "retrasar", "ticket", "a.custodia.ticketEscritor = nil", "defer func(){ a.custodia.ticketEscritor = nil }()", "custodia-separa-antes-close"),
		m("B13", "reintentar-exito", "ticket", "errCierre := archivo.Close()", "errCierre := archivo.Close(); if errCierre == nil { errCierre = archivo.Close() }", "close-no-reintenta-exito"),
		m("B13", "reintentar-EINTR", "ticket", "if errCierre != nil {", "if errors.Is(errCierre, syscall.EINTR) { errCierre = archivo.Close() }; if errCierre != nil {", "close-no-reintenta-EINTR"),
		m("B13", "reintentar-EBADF", "ticket", "if errCierre != nil {", "if errors.Is(errCierre, syscall.EBADF) { errCierre = archivo.Close() }; if errCierre != nil {", "close-no-reintenta-EBADF"),
		m("B13", "reintentar-otro", "ticket", "if errCierre != nil {", "if errCierre != nil { errCierre = archivo.Close() }; if errCierre != nil {", "close-no-reintenta-error"),
		m("B14", "STOP-desde-Go", "stop", "uintptr(pidfd), 0, 0, pidfdSignalGrupoO3bM38", "uintptr(pidfd), uintptr(syscall.SIGSTOP), 0, pidfdSignalGrupoO3bM38", "Go-no-STOP"),
		m("B14", "PID", "stop", "uintptr(pidfd), 0, 0, pidfdSignalGrupoO3bM38", "uintptr(c.cmd.Process.Pid), uintptr(syscall.SIGSTOP), 0, 0", "sin-PID-signal"),
		m("B14", "PGID", "stop", "uintptr(pidfd), 0, 0, pidfdSignalGrupoO3bM38", "uintptr(-c.cmd.Process.Pid), uintptr(syscall.SIGSTOP), 0, 0", "sin-PGID-signal"),
		m("B15", "omitir-flag", "stop", "pidfdSignalGrupoO3bM38 = uintptr(1 << 2)", "pidfdSignalGrupoO3bM38 = 0", "flag-grupo"),
		m("B15", "senal-no-cero", "stop", "uintptr(pidfd), 0, 0, pidfdSignalGrupoO3bM38", "uintptr(pidfd), uintptr(syscall.SIGCONT), 0, pidfdSignalGrupoO3bM38", "senal-cero"),
		m("B15", "promover-reserva", "stop", "a.custodia.pidfdPrimario", "a.custodia.pidfdReserva", "primario-fiable"),
		m("B16", "unico-T", "stop", "runtime.Gosched()", "return nil // una sola T\n\truntime.Gosched()", "dos-T"),
		m("B16", "estado-distinto", "stop", "datos[i+2] == 'T'", "datos[i+2] != 'Z'", "solo-T"),
		m("B16", "sin-estabilidad", "stop", "!bytes.Equal(primera, segunda)", "!bytes.Equal(primera, primera)", "muestras-estables"),
		m("B17", "split-espacios", "identidad", "bytes.Split(restoCrudo, []byte{' '})", "bytes.Fields(datos)", "parser-no-split-simple"),
		m("B17", "primer-parentesis", "identidad", "bytes.LastIndex(datos, []byte(\") \"))", "bytes.Index(datos, []byte(\") \"))", "ultimo-parentesis"),
		m("B17", "sin-limite", "identidad", "len(datos) > maximoStatO3bM38", "false", "limite-4096"),
		m("B17", "signo", "identidad", "if len(campo) == 0 || campo[0] == '+' || campo[0] == '-' || len(campo) > 1 && campo[0] == '0' {", "if len(campo) == 0 || len(campo) > 1 && campo[0] == '0' {", "sin-signo"),
		m("B17", "desbordamiento", "identidad", "valor, err := strconv.ParseUint(string(campo), 10, 64)", "valor, err := uint64(1), error(nil)", "rechaza-desbordamiento"),
		m("B18", "PID", "identidad", "muestra.pid == a.custodia.cmd.Process.Pid", "true", "compara-PID"),
		m("B18", "PPID", "identidad", "muestra.ppid == os.Getpid()", "os.Getpid() == os.Getpid()", "compara-PPID"),
		m("B18", "PGID", "identidad", "muestra.pgid == a.custodia.cmd.Process.Pid", "true", "compara-PGID"),
		m("B18", "SID", "identidad", "muestra.sid == a.sidSupervisor", "true", "compara-SID"),
		m("B18", "starttime", "identidad", "muestra.inicio > 0 && (inicio == 0 || muestra.inicio == inicio)", "true", "compara-starttime"),
	}
	for i := range c {
		c[i].id = fmt.Sprintf("M%03d", i+1)
	}
	return c
}

func suma(b []byte) string { x := sha256.Sum256(b); return hex.EncodeToString(x[:]) }

func fuenteBase(repo, nombre string) []byte {
	c := exec.Command("git", "show", base+":"+paquete+"/"+nombre)
	c.Dir = repo
	b, err := c.Output()
	if err != nil {
		panic(err)
	}
	return b
}

// validarAST es un oráculo causal, no una lista de hashes: requiere que el AST
// mutado siga siendo válido y que la expresión contractual exacta permanezca.
func validarAST(x mutante, b []byte) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, x.fuente, b, parser.AllErrors)
	if err != nil {
		return err
	}
	texto := func(n ast.Node) string { var s bytes.Buffer; _ = format.Node(&s, fset, n); return s.String() }
	fallo := ""
	ast.Inspect(f, func(n ast.Node) bool {
		if a, ok := n.(*ast.AssignStmt); ok && x.familia == "B08" && x.alternativa == "reabrir" && len(a.Lhs) == 1 && len(a.Rhs) == 1 {
			izq, _ := a.Lhs[0].(*ast.Ident)
			der, _ := a.Rhs[0].(*ast.SelectorExpr)
			if izq != nil && izq.Name == "fd" && der != nil && der.Sel.Name == "pidfdPrimario" {
				fallo = "reapertura desde primario"
			}
		}
		if s, ok := n.(*ast.SelectorExpr); ok && x.familia == "B15" && x.alternativa == "promover-reserva" && s.Sel.Name == "pidfdReserva" {
			fallo = "reserva promovida"
		}
		if v, ok := n.(*ast.ValueSpec); ok && x.familia == "B15" && x.alternativa == "omitir-flag" && len(v.Names) == 1 && v.Names[0].Name == "pidfdSignalGrupoO3bM38" && (len(v.Values) != 1 || texto(v.Values[0]) != "uintptr(1 << 2)") {
			fallo = "flag grupo ausente"
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, _ := call.Fun.(*ast.SelectorExpr)
		nombre := ""
		if sel != nil {
			nombre = sel.Sel.Name
		}
		switch x.familia {
		case "B08":
			if nombre == "Dup" || nombre == "NewFile" || nombre == "Syscall" && len(call.Args) > 0 && texto(call.Args[0]) == "434" {
				fallo = "API de duplicacion/reapertura"
			}
		case "B14", "B15":
			if nombre == "Syscall6" && len(call.Args) >= 4 {
				if texto(call.Args[0]) != "sysPidfdSendSignal" || texto(call.Args[1]) != "uintptr(pidfd)" || texto(call.Args[2]) != "0" || texto(call.Args[3]) != "0" || texto(call.Args[4]) != "pidfdSignalGrupoO3bM38" {
					fallo = "firma pidfd signal cero/grupo"
				}
			}
		}
		return true
	})
	if fallo != "" {
		return fmt.Errorf("%s: %s", x.regla, fallo)
	}
	if x.familia == "B08" || x.familia == "B14" || x.familia == "B15" {
		return nil
	}
	return errors.New("familia conductual")
}

func ejecutar(repo, out string, desde, hasta int) error {
	if b, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output(); err != nil || strings.TrimSpace(string(b)) != base {
		return errors.New("base distinta")
	}
	tmp, err := os.MkdirTemp("", "o3b-v3a-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if filepath.Clean(out) == filepath.Clean(tmp) || strings.HasPrefix(filepath.Clean(out)+string(os.PathSeparator), filepath.Clean(tmp)+string(os.PathSeparator)) {
		return errors.New("la evidencia no puede quedar bajo el temporal efimero")
	}
	fmt.Printf("temporal=%s evidencia=%s\n", tmp, out)
	cache := filepath.Join(tmp, "cache")
	_ = os.MkdirAll(cache, 0700)
	lista := catalogo()
	if desde < 1 || hasta < desde || hasta > len(lista) {
		return fmt.Errorf("rango invalido %d..%d de %d", desde, hasta, len(lista))
	}
	lista = lista[desde-1 : hasta]
	var cat, res, fuentes strings.Builder
	cat.WriteString("id\tfamilia\talternativa\tarchivo\tantes\tdespues\tregla\n")
	seen := map[string]bool{}
	for _, x := range lista {
		fmt.Printf("%s %s/%s\n", x.id, x.familia, x.alternativa)
		clave := x.familia + "/" + x.alternativa
		if seen[clave] {
			return fmt.Errorf("duplicado %s", clave)
		}
		seen[clave] = true
		original := fuenteBase(repo, archivos[x.fuente])
		if bytes.Count(original, []byte(x.antes)) != 1 {
			return fmt.Errorf("%s cardinalidad anterior", x.id)
		}
		mutado := bytes.Replace(original, []byte(x.antes), []byte(x.despues), 1)
		estatica := x.familia == "B08" || x.familia == "B14" || x.familia == "B15"
		if err := validarAST(x, mutado); estatica && err == nil {
			return fmt.Errorf("%s sobrevivio AST semantico", x.id)
		}
		dir := filepath.Join(tmp, x.id)
		if err := os.MkdirAll(filepath.Join(dir, paquete), 0755); err != nil {
			return err
		}
		arc := exec.Command("git", "archive", "--format=tar", base)
		arc.Dir = repo
		tar := exec.Command("tar", "-x", "-C", dir)
		p, err := arc.StdoutPipe()
		if err != nil {
			return err
		}
		tar.Stdin = p
		if err = tar.Start(); err != nil {
			return err
		}
		if err = arc.Run(); err != nil {
			return err
		}
		if err = tar.Wait(); err != nil {
			return err
		}
		ruta := filepath.Join(dir, paquete, archivos[x.fuente])
		if err = os.WriteFile(ruta, mutado, 0644); err != nil {
			return err
		}
		gs, err := filepath.Glob(filepath.Join(dir, paquete, "*.go"))
		if err != nil {
			return err
		}
		args := []string{"test"}
		for _, g := range gs {
			if filepath.Base(g) != "capturar_snapshot_fuente_corporativa_contexto_actor_v1.go" {
				args = append(args, filepath.Base(g))
			}
		}
		args = append(args, "-c", "-o", filepath.Join(dir, "mutante.test"))
		cmd := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), args...)
		cmd.Dir = filepath.Join(dir, paquete)
		cmd.Env = []string{"CGO_ENABLED=0", "GOTOOLCHAIN=local", "GOCACHE=" + cache, "HOME=/nonexistent", "PATH=/usr/bin:/bin"}
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		inicio := time.Now()
		salida, e := cmd.CombinedOutput()
		if e != nil {
			return fmt.Errorf("%s no compila: %v %s", x.id, e, salida)
		}
		if cmd.Process != nil {
			if e = syscall.Kill(-cmd.Process.Pid, 0); !errors.Is(e, syscall.ESRCH) {
				return fmt.Errorf("%s PGID no ESRCH: %v", x.id, e)
			}
		}
		salidaPrueba := []byte("estado_analizador=1; regla=" + x.regla)
		tipoMuerte := "MUERTO-AST-CAUSAL"
		if !estatica {
			pruebaArgs := append([]string{}, args[:len(args)-4]...)
			pruebaArgs = append(pruebaArgs, "-run", "O3b", "-count=1", "-timeout=12s")
			prueba := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), pruebaArgs...)
			prueba.Dir = cmd.Dir
			prueba.Env = cmd.Env
			prueba.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			salidaPrueba, e = prueba.CombinedOutput()
			if e == nil {
				return fmt.Errorf("%s sobrevivio prueba conductual", x.id)
			}
			if prueba.Process != nil {
				if e = syscall.Kill(-prueba.Process.Pid, 0); !errors.Is(e, syscall.ESRCH) {
					return fmt.Errorf("%s PGID prueba no ESRCH: %v", x.id, e)
				}
			}
			tipoMuerte = "MUERTO-PRUEBA-CAUSAL"
		}
		fmt.Fprintf(&cat, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", x.id, x.familia, x.alternativa, archivos[x.fuente], hex.EncodeToString([]byte(x.antes)), hex.EncodeToString([]byte(x.despues)), x.regla)
		fmt.Fprintf(&res, "%s\t%s\t%s\tCOMPILA\t%s\t%d\tcompila_sha=%s\tprueba_sha=%s\n", x.id, x.familia, x.alternativa, tipoMuerte, time.Since(inicio).Nanoseconds(), suma(salida), suma(salidaPrueba))
	}
	claves := make([]string, 0, len(archivos))
	for k := range archivos {
		claves = append(claves, k)
	}
	sort.Strings(claves)
	for _, k := range claves {
		b := fuenteBase(repo, archivos[k])
		fmt.Fprintf(&fuentes, "%s\t%s\t%s\n", k, archivos[k], suma(b))
	}
	if err = os.MkdirAll(out, 0755); err != nil {
		return err
	}
	fmt.Printf("escribiendo evidencia tras %d mutantes en %s\n", len(lista), out)
	for n, b := range map[string][]byte{"catalogo.tsv": []byte(cat.String()), "resultados.tsv": []byte(res.String()), "fuentes.tsv": []byte(fuentes.String())} {
		if err = os.WriteFile(filepath.Join(out, n), b, 0644); err != nil {
			return err
		}
	}
	manifest := fmt.Sprintf("base\t%s\ntoolchain\t%s\nmutantes\t%d/%d compilables-y-muertos\npgid\tESRCH por mutante\n", base, runtime.Version(), len(lista), len(lista))
	if err := os.WriteFile(filepath.Join(out, "manifiesto.tsv"), []byte(manifest), 0644); err != nil {
		return err
	}
	fmt.Println("evidencia escrita")
	return nil
}

func main() {
	repo := flag.String("repo", ".", "repositorio")
	out := flag.String("out", "tools/o3b_p7_mutantes_v3a/evidencia", "evidencia")
	desde := flag.Int("desde", 1, "primer mutante inclusivo")
	hasta := flag.Int("hasta", len(catalogo()), "ultimo mutante inclusivo")
	soloCatalogo := flag.Bool("solo-catalogo", false, "escribe el catalogo canonico sin ejecutar")
	flag.Parse()
	destino := *out
	if !filepath.IsAbs(destino) {
		destino = filepath.Join(*repo, destino)
	}
	if *soloCatalogo {
		var b strings.Builder
		for _, x := range catalogo() {
			fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", x.id, x.familia, x.alternativa, archivos[x.fuente], hex.EncodeToString([]byte(x.antes)), hex.EncodeToString([]byte(x.despues)), x.regla)
		}
		if err := os.MkdirAll(destino, 0755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := os.WriteFile(filepath.Join(destino, "catalogo.tsv"), []byte(b.String()), 0644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := ejecutar(*repo, destino, *desde, *hasta); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
