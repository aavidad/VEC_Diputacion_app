package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

func (a *analisis) entrada() error {
	fd := a.funciones["consumirAutoridadO3cM38"]
	if fd == nil || fd.Type == nil || len(fd.Type.Params.List) != 1 || !doblePunteroA(fd.Type.Params.List[0].Type, "agregadoO3cM38") {
		return fmt.Errorf("firma única de entrada ausente")
	}
	entradas := 0
	for _, f := range a.funciones {
		for _, p := range f.Type.Params.List {
			if doblePunteroA(p.Type, "agregadoO3cM38") {
				entradas++
			}
		}
	}
	if entradas != 1 {
		return fmt.Errorf("entradas **agregadoO3cM38=%d", entradas)
	}
	if err := a.guardaB5Exacta(); err != nil {
		return err
	}
	b := a.de("autoridad.go")
	for _, p := range []string{"agregado := *entrada", "*entrada = nil", "switch clasificarEntradaO3cM38(agregado)", "estadoLease != 1 || estadoObservador != 1", "estadoLease == 3 && estadoObservador == 2"} {
		if err := exigir(b, p, 1, p); err != nil {
			return err
		}
	}
	valida := a.cuerpo("custodiaConsumidaValidaO3cM38")
	for _, p := range []string{
		"c.cmd != nil", "c.cmd.Process != nil", "c.controlFD != nil", "c.terminal != nil",
		"c.ticketEscritor == nil", "c.ticketLector == nil", "a.primera == nil",
		"c.pidfdPrimario >= 0", "c.pidfdReserva >= 0", "c.pidfdOpaco >= 0",
		"c.pidfdPrimario != c.pidfdReserva", "c.pidfdPrimario != c.pidfdOpaco", "c.pidfdReserva != c.pidfdOpaco",
		"a.identidad.pid == c.cmd.Process.Pid", "a.identidad.estado == 'T'", "a.identidad.inicio > 0",
	} {
		if err := exigir(valida, p, 1, "C02 "+p); err != nil {
			return err
		}
	}
	return nil
}

func (a *analisis) guardaB5Exacta() error {
	fd := a.funciones["clasificarEntradaO3cM38"]
	if fd == nil || fd.Body == nil || len(fd.Body.List) == 0 {
		return fmt.Errorf("clasificador B5 ausente")
	}
	primero, ok := fd.Body.List[0].(*ast.IfStmt)
	if !ok {
		return fmt.Errorf("B5 no es la primera guarda")
	}
	esperada := "a == nil || a.estado != capturaB5CapturadoM38 || a.custodia == nil"
	if got := expresion(primero.Cond); got != esperada {
		return fmt.Errorf("guarda B5=%q esperada=%q", got, esperada)
	}
	obj := a.info.Uses[selector(primero.Cond, "estado")]
	if obj == nil || obj.Type() == nil || obj.Type().String() != "evidencia/o3c_p6.estadoCapturaO3bM38" {
		return fmt.Errorf("estado B5 sin tipo autoritativo")
	}
	for nombre, f := range a.funciones {
		var mutada bool
		ast.Inspect(f.Body, func(n ast.Node) bool {
			as, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}
			for _, lhs := range as.Lhs {
				if s, ok := lhs.(*ast.SelectorExpr); ok && s.Sel.Name == "estado" {
					if id, ok := s.X.(*ast.Ident); ok && id.Name == "agregado" {
						mutada = true
					}
				}
			}
			return true
		})
		if mutada {
			return fmt.Errorf("B5 mutado por %s", nombre)
		}
	}
	consumo := a.cuerpo("consumirAutoridadO3cM38")
	return enOrden(consumo, []string{"switch clasificarEntradaO3cM38(agregado)", "consolidarEntradaO3cM38(c, autoridad)"}, "B5 antes de CAS")
}

func selector(n ast.Node, nombre string) *ast.Ident {
	var encontrado *ast.Ident
	ast.Inspect(n, func(x ast.Node) bool {
		if s, ok := x.(*ast.SelectorExpr); ok && s.Sel.Name == nombre {
			encontrado = s.Sel
			return false
		}
		return encontrado == nil
	})
	return encontrado
}

func (a *analisis) maquina() error {
	b := a.de("autoridad.go")
	for _, e := range []string{"C0Recibido", "C1Revalidado", "C2ContIntentado", "C3Observado", "C4TTransfiriendo", "C5Entregado", "C7Retirando", "C8Retirado", "CFFatal"} {
		if !bytes.Contains(b, []byte("continuacion"+e+"M38")) {
			return fmt.Errorf("falta estado %s", e)
		}
	}
	if err := exigir(b, "func transicionContinuacionO3cM38", 1, "transición"); err != nil {
		return err
	}
	for _, desde := range []string{"continuacionC0RecibidoM38:", "continuacionC1RevalidadoM38:", "continuacionC2ContIntentadoM38:", "continuacionC3ObservadoM38:", "continuacionC4TTransfiriendoM38:", "continuacionC7RetirandoM38:"} {
		if !bytes.Contains(b, []byte("case "+desde)) {
			return fmt.Errorf("estado no total %s", desde)
		}
	}
	for _, arista := range []string{
		"hacia == continuacionC1RevalidadoM38 || hacia == continuacionC7RetirandoM38 || hacia == continuacionCFFatalM38",
		"hacia == continuacionC2ContIntentadoM38 || hacia == continuacionCFFatalM38",
		"hacia == continuacionC3ObservadoM38 || hacia == continuacionCFFatalM38",
		"hacia == continuacionC4TTransfiriendoM38 || hacia == continuacionCFFatalM38",
		"hacia == continuacionC5EntregadoM38 || hacia == continuacionCFFatalM38",
		"hacia == continuacionC8RetiradoM38 || hacia == continuacionCFFatalM38",
	} {
		if err := exigir(b, arista, 1, arista); err != nil {
			return err
		}
	}
	return nil
}

func (a *analisis) autoridad() error {
	b := a.de("autoridad.go")
	if err := casOwnersExactos(a.funciones["activarO3c"], "propietarioInactivoO3cM38", "propietarioO3cM38"); err != nil {
		return fmt.Errorf("owners activación: %w", err)
	}
	if err := a.esquemaExacto("autoridadCustodiaO3cM38", map[string]string{
		"auto": "*autoridadCustodiaO3cM38", "ownerObservador": "atomic.Uint32", "ownerLease": "atomic.Uint32",
	}); err != nil {
		return fmt.Errorf("owners autoridad: %w", err)
	}
	clasifica := a.cuerpo("clasificarEntradaO3cM38")
	for _, p := range []string{
		"c.lease.auto != c.lease", "c.observador.auto != c.observador", "c.lease.registro == nil",
		"c.lease.registro.auto != c.lease.registro", "c.lease.registro != c.observador.registro",
		"c.lease.registro.tid != c.tid", "c.lease.tid != c.tid",
		"c.lease.registro.leases[c.lease] != c.lease.generacion",
		"c.observador.registro.observadores[c.observador] != c.observador.generacion",
		"c.baselineSenal != c.observador.palabra.Load()", "uint8(c.baselineSenal>>2) != 0",
	} {
		if err := exigir(clasifica, p, 1, "C03 "+p); err != nil {
			return err
		}
	}
	identidad := a.cuerpo("identidadEjecucionO3cM38")
	for _, p := range []string{"tid == c.tid", "ppid == c.ppid", "pdeathsig == int32(syscall.SIGTERM)"} {
		if err := exigir(identidad, p, 1, "C03 "+p); err != nil {
			return err
		}
	}
	for _, p := range []string{
		"c.observador.transferirCritico(c.baselineSenal)", "c.lease.transferirCritico()", "autoridad.activarO3c()",
		"ownerObservador.CompareAndSwap(uint32(propietarioInactivoO3cM38), uint32(propietarioO3cM38))",
		"ownerLease.CompareAndSwap(uint32(propietarioInactivoO3cM38), uint32(propietarioO3cM38))",
	} {
		if err := exigir(b, p, 1, p); err != nil {
			return err
		}
	}
	if bytes.Index(b, []byte("c.observador.transferirCritico")) > bytes.Index(b, []byte("c.lease.transferirCritico")) {
		return fmt.Errorf("lease consumida antes de observador")
	}
	for _, reversa := range []string{"CompareAndSwap(2, 1)", "CompareAndSwap(3, 1)", "CompareAndSwap(uint32(propietarioO4aM38), uint32(propietarioO3cM38))"} {
		if contarEnTodos(a.contenido, reversa) != 0 {
			return fmt.Errorf("rollback prohibido %s", reversa)
		}
	}
	return nil
}

func (a *analisis) revalidacion() error {
	b := a.de("revalidacion.go")
	cuerpo := a.cuerpo("revalidarAntesContO3cM38")
	if err := segundaRondaExacta(a.funciones["segundaRondaO3cM38"]); err != nil {
		return fmt.Errorf("C024 segunda ronda: %w", err)
	}
	inventario := a.cuerpo("pidfdEInventarioO3cM38")
	for _, p := range []string{
		"identidadPidfdBarreraO3bM38(c, c.pidfdPrimario)", "identidadPidfdBarreraO3bM38(c, c.pidfdReserva)",
		"identidadPidfdBarreraO3bM38(c, c.pidfdOpaco)", "pidfdVivoBarreraO3bM38(c, c.pidfdPrimario)",
		"pidfdVivoBarreraO3bM38(c, c.pidfdReserva)", "!fiablePrimario && !fiableReserva",
		"errPrimario == nil && errVivoPrimario == nil && vivoPrimario",
		"errReserva == nil && errVivoReserva == nil && vivoReserva",
		"!fiablePrimario || !fiableReserva || errOpaco != nil", "identidadFisicaO3aM38(primario, reserva)",
		"identidadFisicaO3aM38(primario, opaco)", "referencias != 3",
		"primario.fdflags&syscall.FD_CLOEXEC == 0", "reserva.fdflags&syscall.FD_CLOEXEC == 0",
		"opaco.fdflags&syscall.FD_CLOEXEC == 0",
	} {
		if err := exigir(inventario, p, 1, "C06 "+p); err != nil {
			return err
		}
	}
	sellos := a.cuerpo("sellosCustodiaValidosO3cM38")
	for _, p := range []string{"c.cmd.Process.WithHandle", "handle == uintptr(c.pidfdOpaco)", "[3]int{c.pidfdPrimario, c.pidfdReserva, c.pidfdOpaco} != s.pidfd"} {
		if err := exigir(sellos, p, 1, "C06 "+p); err != nil {
			return err
		}
	}
	proc := a.cuerpo("identidadProcesoFinalO3cM38")
	for _, p := range []string{
		"leerStatStopO3bM38(a.custodia)", "parsearStatO3bM38(raw)", "muestra == a.identidad",
		"a.identidad.estado == 'T'", "a.identidad.pid == c.cmd.Process.Pid", "a.identidad.ppid > 0",
		"a.identidad.pgid == c.cmd.Process.Pid", "a.identidad.sid > 0", "a.identidad.inicio > 0",
	} {
		if err := exigir(proc, p, 1, "C07 "+p); err != nil {
			return err
		}
	}
	if contarEnTodos(a.contenido, "leerStatStopO3bM38(") != 1 || contarEnTodos(a.contenido, "parsearStatO3bM38(") != 1 {
		return fmt.Errorf("C07 lector/parser proc no únicos")
	}
	fd := a.funciones["revalidarAntesContO3cM38"]
	control, err := posicionLlamada(fd, "leerControlO3bM38", "c")
	if err != nil {
		return fmt.Errorf("C04 CONTROL: %w", err)
	}
	if err := validarIfControl(fd, control, true); err != nil {
		return fmt.Errorf("C04 causa CONTROL: %w", err)
	}
	primeraObservador := len(fd.Body.List)
	for i, st := range fd.Body.List {
		ast.Inspect(st, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok && nombreLlamada(c) == "autoridadSenalO3cM38" && i < primeraObservador {
				primeraObservador = i
			}
			return true
		})
	}
	if primeraObservador <= control {
		return fmt.Errorf("C024 observador previo a CONTROL: control=%d observador=%d", control, primeraObservador)
	}
	observador, err := posicionLlamada(fd, "autoridadSenalO3cM38", "c")
	if err != nil || observador != control+1 {
		return fmt.Errorf("C04 CONTROL→observador no adyacente: control=%d observador=%d err=%v", control, observador, err)
	}
	orden := []string{"leerControlO3bM38(c)", "autoridadSenalO3cM38(c)", "identidadEjecucionO3cM38(c)", "time.Now().Before(c.finBootstrap)", "pidfdEInventarioO3cM38(a)", "identidadProcesoFinalO3cM38(a)", "segundaRondaO3cM38(a)", "c.lease.comenzar(operacionContO3cM38", "preasignacionO4aValidaO3cM38(a)", "time.Now().Before(c.finBootstrap)"}
	if err := enOrden(cuerpo, orden, "ronda final"); err != nil {
		return err
	}
	for _, p := range []string{"referencias != 3", "leerStatStopO3bM38(a.custodia)", "parsearStatO3bM38(raw)", "syscallLeaseO3cM38", "operarConLeaseBarreraO3bM38"} {
		if !bytes.Contains(b, []byte(p)) {
			return fmt.Errorf("falta %s", p)
		}
	}
	return nil
}

func posicionLlamada(fd *ast.FuncDecl, nombre, argumento string) (int, error) {
	if fd == nil || fd.Body == nil {
		return -1, fmt.Errorf("función ausente")
	}
	posicion, total := -1, 0
	for i, st := range fd.Body.List {
		ast.Inspect(st, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok || nombreLlamada(c) != nombre {
				return true
			}
			total++
			if len(c.Args) == 1 && expresion(c.Args[0]) == argumento {
				posicion = i
			}
			return true
		})
	}
	if total != 1 || posicion < 0 {
		return -1, fmt.Errorf("%s cardinalidad=%d/argumento", nombre, total)
	}
	return posicion, nil
}

func validarIfControl(fd *ast.FuncDecl, posicion int, revalidacion bool) error {
	if posicion < 0 || posicion >= len(fd.Body.List) {
		return fmt.Errorf("posición inválida")
	}
	si, ok := fd.Body.List[posicion].(*ast.IfStmt)
	if !ok || expresion(si.Cond) != "err != nil" {
		return fmt.Errorf("condición no es err != nil")
	}
	if revalidacion {
		if len(si.Body.List) != 1 {
			return fmt.Errorf("rama revalidación cardinalidad=%d", len(si.Body.List))
		}
		ret, ok := si.Body.List[0].(*ast.ReturnStmt)
		esperado := "resolverRevalidacionO3cM38(a, err, preContControlO3cM38)"
		if !ok || len(ret.Results) != 1 || expresion(ret.Results[0]) != esperado {
			return fmt.Errorf("retorno no conserva err/causa CONTROL")
		}
		return nil
	}
	if len(si.Body.List) != 2 || !bytes.Contains(nodo(token.NewFileSet(), si.Body.List[0]), []byte("errors.Is(err, errLeaseBarreraO3bM38)")) {
		return fmt.Errorf("rama observación no preserva fatalidad lease")
	}
	ret, ok := si.Body.List[1].(*ast.ReturnStmt)
	esperado := "instalarObservacionO3cM38(a, observacionControlRawO3cM38)"
	if !ok || len(ret.Results) != 1 || expresion(ret.Results[0]) != esperado {
		return fmt.Errorf("error CONTROL no instala control_raw")
	}
	return nil
}

func (a *analisis) cont() error {
	b := a.de("cont.go")
	permiso := "l != nil && l.auto == l && l.registro != nil && l.registro.auto == l.registro && l.registro.leases[l] == l.generacion && l.tid == l.registro.tid && l.estado.Load() == 2 && p.lease == l && p.generacion == l.secuencia && p.operacion == l.operacion && p.cardinalidad == l.cardinal && p.objetivos == l.objetivos && p.estadoPrevio == 3 && p.operacion == operacionContO3cM38 && p.cardinalidad == 1 && p.objetivos == [2]int{primario, -1}"
	if err := retornoExacto(a.funciones["permisoContMemoriaValidoO3cM38"], permiso); err != nil {
		return fmt.Errorf("C08 permiso: %w", err)
	}
	consolidacion := "permisoContMemoriaValidoO3cM38(l, p, primario) && l.estado.CompareAndSwap(2, p.estadoPrevio)"
	if err := retornoExacto(a.funciones["consolidarContO3cM38"], consolidacion); err != nil {
		return fmt.Errorf("C08 consolidación: %w", err)
	}
	if err := retornoExacto(a.funciones["tiempoMonotonoO3cM38"], "!marca.IsZero() && marca != marca.Round(0)"); err != nil {
		return fmt.Errorf("C09 monotónico: %w", err)
	}
	fin := a.cuerpo("finCasoExactoO3cM38")
	for _, p := range []string{"ahora.Add(duracionCasoO3cM38)", "tiempoMonotonoO3cM38(fin)", "fin.After(ahora)", "fin.Sub(ahora) == duracionCasoO3cM38"} {
		if err := exigir(fin, p, 1, "C09 "+p); err != nil {
			return err
		}
	}
	if bytes.Count(b, []byte("time.Now()")) != 1 || bytes.Count(b, []byte("finCasoExactoO3cM38(ahoraCaso)")) != 1 {
		return fmt.Errorf("C09 reloj/marca no únicos")
	}
	for _, p := range []string{"duracionCasoO3cM38     = 180 * time.Second", "pidfdSignalGrupoO3cM38 = uintptr(1 << 2)", "*entrada = nil", "syscall.SIGCONT", "if !consolidarContO3cM38"} {
		if err := exigir(b, p, 1, p); err != nil {
			return err
		}
	}
	intento := a.cuerpo("intentarContO3cM38")
	for lhs, rhs := range map[string]string{"a.salida.ahoraCaso": "ahoraCaso", "a.salida.finCaso": "finCaso", "a.salida.retornoCont": "int(retornoRaw)"} {
		if err := asignacionTopLevelExacta(a.funciones["intentarContO3cM38"], lhs, rhs); err != nil {
			return fmt.Errorf("C12/C18 %s: %w", lhs, err)
		}
	}
	for _, p := range []string{"r := *entrada", "*entrada = nil", "r.auto = nil", "a.salida.ahoraCaso = ahoraCaso", "a.salida.finCaso = finCaso", "a.salida.retornoCont = int(retornoRaw)", "a.estado = continuacionC2ContIntentadoM38"} {
		if err := exigir(intento, p, 1, "C12 "+p); err != nil {
			return err
		}
	}
	if err := a.syscallContExacto(); err != nil {
		return err
	}
	return enOrden(a.cuerpo("intentarContO3cM38"), []string{"ahoraCaso := time.Now()", "finCaso, marcaValida := finCasoExactoO3cM38(ahoraCaso)", "syscall.Syscall6(sysPidfdSendSignal", "consolidarContO3cM38", "a.salida.ahoraCaso = ahoraCaso", "continuacionC2ContIntentadoM38"}, "marca→CONT")
}

func (a *analisis) syscallContExacto() error {
	fd := a.funciones["intentarContO3cM38"]
	if fd == nil || fd.Body == nil {
		return fmt.Errorf("función CONT ausente")
	}
	indiceSyscall := -1
	var llamada *ast.CallExpr
	for i, st := range fd.Body.List {
		ast.Inspect(st, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if ok && nombreLlamada(c) == "Syscall6" {
				if llamada != nil {
					indiceSyscall = -2
					return false
				}
				llamada, indiceSyscall = c, i
			}
			return true
		})
	}
	if llamada == nil || indiceSyscall < 3 || indiceSyscall == -2 || len(llamada.Args) != 7 {
		return fmt.Errorf("Syscall6 CONT único/cardinalidad inválida")
	}
	fun, ok := llamada.Fun.(*ast.SelectorExpr)
	if !ok {
		return fmt.Errorf("callee CONT no es selector")
	}
	paquete, paqueteOK := fun.X.(*ast.Ident)
	if !paqueteOK || paquete.Name != "syscall" || fun.Sel.Name != "Syscall6" {
		return fmt.Errorf("callee CONT no es syscall.Syscall6")
	}
	esperados := []string{"sysPidfdSendSignal", "uintptr(c.pidfdPrimario)", "uintptr(syscall.SIGCONT)", "0", "pidfdSignalGrupoO3cM38", "0", "0"}
	for i, arg := range llamada.Args {
		if got := expresion(arg); got != esperados[i] {
			return fmt.Errorf("argumento CONT[%d]=%s esperado=%s", i, got, esperados[i])
		}
		if a.info.Types[arg].Type == nil {
			return fmt.Errorf("argumento CONT[%d] sin tipo", i)
		}
	}
	asignacion, ok := fd.Body.List[indiceSyscall].(*ast.AssignStmt)
	if !ok || asignacion.Tok != token.DEFINE || len(asignacion.Lhs) != 3 || len(asignacion.Rhs) != 1 ||
		expresion(asignacion.Lhs[0]) != "_" || expresion(asignacion.Lhs[1]) != "_" || expresion(asignacion.Lhs[2]) != "retornoRaw" {
		return fmt.Errorf("C12 intento CONT no es asignación top-level incondicional")
	}
	barrera, ok := fd.Body.List[indiceSyscall-1].(*ast.IfStmt)
	if !ok || expresion(barrera.Cond) != "!marcaValida || !ahoraCaso.Before(c.finBootstrap)" {
		return fmt.Errorf("C13 barrera bootstrap divergente")
	}
	var bucle bool
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			bucle = true
		}
		return true
	})
	if bucle {
		return fmt.Errorf("C12 bucle/reintento presente")
	}
	if _, ok := fd.Body.List[indiceSyscall-1].(*ast.IfStmt); !ok ||
		!bytes.Contains(nodo(a.fset, fd.Body.List[indiceSyscall-2]), []byte("finCasoExactoO3cM38(ahoraCaso)")) ||
		!bytes.Contains(nodo(a.fset, fd.Body.List[indiceSyscall-3]), []byte("time.Now()")) || indiceSyscall+1 >= len(fd.Body.List) ||
		!bytes.Contains(nodo(a.fset, fd.Body.List[indiceSyscall+1]), []byte("if !consolidarContO3cM38(c.lease, permiso, c.pidfdPrimario)")) {
		return fmt.Errorf("Syscall6 no adyacente a reloj→marca→barrera")
	}
	return nil
}

func retornoExacto(fd *ast.FuncDecl, esperado string) error {
	if fd == nil || fd.Body == nil || len(fd.Body.List) != 1 {
		return fmt.Errorf("cuerpo no es retorno único")
	}
	ret, ok := fd.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 || expresion(ret.Results[0]) != esperado {
		return fmt.Errorf("retorno divergente")
	}
	return nil
}

func (a *analisis) observacion() error {
	b := a.de("observacion.go")
	instalar := a.cuerpo("instalarObservacionO3cM38")
	clasificador := a.cuerpo("clasificarPollO3cM38")
	for p, n := range map[string]int{
		"return observacionPidfdVacioO3cM38": 1, "return observacionPidfdTerminalNaturalO3cM38": 1,
		"return observacionPidfdInfraestructuraO3cM38": 2,
	} {
		if err := exigir(clasificador, p, n, "C14 "+p); err != nil {
			return err
		}
	}
	sondeo := a.cuerpo("observarPidfdO3cM38")
	if err := exigir(sondeo, "if !fiablePrimario && !fiableReserva {\n\t\treturn observacionPidfdInfraestructuraO3cM38", 1, "C14 dos no fiables"); err != nil {
		return err
	}
	union := "d >= observacionControlRawO3cM38 && d <= observacionPidfdInfraestructuraO3cM38"
	if err := retornoExacto(a.funciones["discriminanteObservacionValidoO3cM38"], union); err != nil {
		return fmt.Errorf("C14 unión discriminante: %w", err)
	}
	for _, p := range []string{
		"pollInO3cM38   int16 = 0x001", "pollErrO3cM38  int16 = 0x008", "pollHupO3cM38  int16 = 0x010", "pollNvalO3cM38 int16 = 0x020",
		"n == 0 && primero == 0 && segundo == 0", "primero == pollInO3cM38", "segundo == pollInO3cM38",
		"eventos > 0 && n == eventos", "primero == 0 || primero == pollInO3cM38", "segundo == 0 || segundo == pollInO3cM38",
		"syscall.Syscall(syscall.SYS_POLL, punteroPollO3aM38(&sondas[0]), uintptr(cardinalidad), 0)",
	} {
		if !bytes.Contains(b, []byte(p)) {
			return fmt.Errorf("C14 falta %s", p)
		}
	}
	if contarEnTodos(a.contenido, "syscall.SYS_POLL") != 1 || bytes.Count(instalar, []byte(".primera.CompareAndSwap(")) != 1 || bytes.Count(instalar, []byte(".primera.CompareAndSwap(0, uint32(d))")) != 1 ||
		bytes.Contains(instalar, []byte(".primera.Store(")) || bytes.Contains(instalar, []byte(".primera.Swap(")) {
		return fmt.Errorf("C14/C16 poll o instalación no únicos")
	}
	fd := a.funciones["observarInmediatoO3cM38"]
	control, err := posicionLlamada(fd, "leerControlO3bM38", "a.custodia")
	if err != nil {
		return fmt.Errorf("CONTROL observación: %w", err)
	}
	if err := validarIfControl(fd, control, false); err != nil {
		return fmt.Errorf("CONTROL raw: %w", err)
	}
	observador, err := posicionLlamada(fd, "autoridadSenalO3cM38", "a.custodia")
	if err != nil || observador != control+1 {
		return fmt.Errorf("observación CONTROL→observador no adyacente: %d/%d %v", control, observador, err)
	}
	pidfd, err := posicionLlamada(fd, "observarPidfdO3cM38", "a")
	if err != nil || pidfd <= observador {
		return fmt.Errorf("observación observador→pidfd divergente: %d/%d %v", observador, pidfd, err)
	}
	for _, p := range []string{
		"return instalarObservacionO3cM38(a, observacionControlRawO3cM38)",
		"return instalarObservacionO3cM38(a, observacionSenalRawO3cM38)",
		"return instalarObservacionO3cM38(a, observarPidfdO3cM38(a))",
	} {
		if err := exigir(a.cuerpo("observarInmediatoO3cM38"), p, 1, "C15 "+p); err != nil {
			return err
		}
	}
	for _, d := range []string{"observacionControlRawO3cM38", "observacionSenalRawO3cM38", "observacionPidfdVacioO3cM38", "observacionPidfdTerminalNaturalO3cM38", "observacionPidfdInfraestructuraO3cM38"} {
		if !bytes.Contains(b, []byte(d)) {
			return fmt.Errorf("falta discriminante %s", d)
		}
	}
	if err := exigir(b, "a.salida.primera.CompareAndSwap(0, uint32(d))", 1, "CAS observación"); err != nil {
		return err
	}
	for _, p := range []string{"discriminanteObservacionValidoO3cM38(d)", "transicionContinuacionO3cM38(a.estado, continuacionC3ObservadoM38)", "a.estado = continuacionC3ObservadoM38"} {
		if err := exigir(instalar, p, 1, "C16 "+p); err != nil {
			return err
		}
	}
	for _, p := range []string{"causa", "SALIDA", "PLAZO", "CANCELADO"} {
		if bytes.Contains(b, []byte(p)) {
			return fmt.Errorf("C16 causa funcional presente %s", p)
		}
	}
	return enOrden(a.cuerpo("observarInmediatoO3cM38"), []string{"leerControlO3bM38(a.custodia)", "autoridadSenalO3cM38(a.custodia)", "observarPidfdO3cM38(a)"}, "CONTROL→observador→pidfd")
}

func (a *analisis) handoff() error {
	b := a.de("handoff.go")
	if err := a.validarRetirada(); err != nil {
		return fmt.Errorf("C19 retirada: %w", err)
	}
	prevalidar := a.cuerpo("autoridadHandoffO3cM38")
	for _, p := range []string{
		"a.es(continuacionC3ObservadoM38)", "a.salida.auto != a.salida", "a.salida.autoridad != a.autoridad",
		"a.salida.custodia != a.custodia", "a.salida.identidad != a.identidad", "discriminanteObservacionValidoO3cM38",
		"a.salida.finCaso.Sub(a.salida.ahoraCaso) != 180*time.Second", "a.autoridad.auto != a.autoridad",
		"a.autoridad.poseeO3c()", "autoridadesExactasO3cM38(a)", "sellosMemoriaValidosO3cM38(a)",
		"c.lease.estado.Load() == 3", "c.observador.palabra.Load()&mascaraEstadoObservadorO3aM38 == 2",
	} {
		if !bytes.Contains(prevalidar, []byte(p)) {
			return fmt.Errorf("C17 prevalidación omite %s", p)
		}
	}
	transferir := a.cuerpo("transferirHandoffO3cM38")
	for _, p := range []string{"a := *entrada", "*entrada = nil", "!autoridadHandoffO3cM38(a)", "transicionContinuacionO3cM38(a.estado, continuacionC4TTransfiriendoM38)", "a.estado = continuacionC4TTransfiriendoM38", "return consolidarHandoffO3cM38(a)"} {
		if err := exigir(transferir, p, 1, "C17 "+p); err != nil {
			return err
		}
	}
	if err := casOwnersExactos(a.funciones["consolidarHandoffO3cM38"], "propietarioO3cM38", "propietarioO4aM38"); err != nil {
		return fmt.Errorf("C17 owners handoff: %w", err)
	}
	if err := enOrden(a.cuerpo("consolidarHandoffO3cM38"), []string{"ownerObservador.CompareAndSwap", "ownerLease.CompareAndSwap", "continuacionC5EntregadoM38", "a.custodia, a.autoridad, a.salida = nil, nil, nil"}, "handoff conjunto"); err != nil {
		return err
	}
	consolidar := a.cuerpo("consolidarHandoffO3cM38")
	fdConsolidar := a.funciones["consolidarHandoffO3cM38"]
	if fdConsolidar == nil || fdConsolidar.Body == nil || len(fdConsolidar.Body.List) != 5 ||
		string(nodo(a.fset, fdConsolidar.Body.List[1])) != "a.estado = continuacionC5EntregadoM38" ||
		string(nodo(a.fset, fdConsolidar.Body.List[2])) != "salida := a.salida" ||
		string(nodo(a.fset, fdConsolidar.Body.List[3])) != "a.custodia, a.autoridad, a.salida = nil, nil, nil" ||
		string(nodo(a.fset, fdConsolidar.Body.List[4])) != "return salida" {
		return fmt.Errorf("C18 cuerpo handoff no exacto")
	}
	var mutacionSalida string
	ast.Inspect(fdConsolidar.Body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for _, lhs := range as.Lhs {
			x := expresion(lhs)
			if strings.HasPrefix(x, "salida.") || strings.HasPrefix(x, "a.salida.") {
				mutacionSalida = x
			}
		}
		return mutacionSalida == ""
	})
	if mutacionSalida != "" {
		return fmt.Errorf("C18 mutación tardía agregado: %s", mutacionSalida)
	}
	for _, p := range []string{"salida := a.salida", "a.custodia, a.autoridad, a.salida = nil, nil, nil", "return salida"} {
		if err := exigir(consolidar, p, 1, "C18 "+p); err != nil {
			return err
		}
	}
	if err := exigir(consolidar, "fatalHandoffO3cM38(a)", 1, "C17 partición fatal"); err != nil {
		return err
	}
	for _, p := range []string{"duracionRetiradaO3cM38 = 3 * time.Second", "continuacionC7RetirandoM38", "syscall.SIGKILL", "esperarTerminalRetiradaO3cM38", "esperarConLeaseO3aM38", "syscall.ECHILD", "grupoAusenteO3cM38", "syscall.ESRCH", "cerrarRecursosRetiradaO3cM38", "c.observador.liberar()", "c.lease.liberar()", "continuacionC8RetiradoM38"} {
		if !bytes.Contains(b, []byte(p)) {
			return fmt.Errorf("falta retirada %s", p)
		}
	}
	return enOrden(a.cuerpo("retirarAntesContO3cM38"), []string{"esperarConLeaseO3aM38(a.custodia)", "drenarAdoptadosO3cM38", "grupoAusenteO3cM38", "cerrarRecursosRetiradaO3cM38", "inventarioLiberadoO3cM38", "liberarRetiradaO3cM38"}, "Wait→ECHILD→ESRCH→cierres→liberación")
}

func casOwnersExactos(fd *ast.FuncDecl, desde, hacia string) error {
	if fd == nil || fd.Body == nil {
		return fmt.Errorf("función ausente")
	}
	var owners []string
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok || nombreLlamada(c) != "CompareAndSwap" {
			return true
		}
		s, ok := c.Fun.(*ast.SelectorExpr)
		if !ok || len(c.Args) != 2 || expresion(c.Args[0]) != "uint32("+desde+")" || expresion(c.Args[1]) != "uint32("+hacia+")" {
			owners = append(owners, "invalido")
			return true
		}
		owners = append(owners, expresion(s.X))
		return true
	})
	esperados := []string{"a.ownerObservador", "a.ownerLease"}
	if strings.Contains(fd.Name.Name, "Handoff") {
		esperados = []string{"a.autoridad.ownerObservador", "a.autoridad.ownerLease"}
	}
	if len(owners) != 2 || owners[0] != esperados[0] || owners[1] != esperados[1] {
		return fmt.Errorf("orden/cardinalidad owners=%v esperado=%v", owners, esperados)
	}
	return nil
}

func (a *analisis) o4() error {
	b := a.de("autoridad.go")
	if err := a.esquemaExacto("agregadoO4aM38", map[string]string{
		"auto": "*agregadoO4aM38", "autoridad": "*autoridadCustodiaO3cM38", "custodia": "*custodiaO3aM38",
		"identidad": "muestraStatO3bM38", "primera": "atomic.Uint32", "ahoraCaso": "time.Time", "finCaso": "time.Time", "retornoCont": "int",
	}); err != nil {
		return fmt.Errorf("agregado O4a: %w", err)
	}
	crear := a.cuerpo("consumirAutoridadO3cM38")
	for _, p := range []string{"salida := &agregadoO4aM38{autoridad: autoridad}", "salida.auto = salida", "salida.custodia, salida.identidad = c, agregado.identidad"} {
		if err := exigir(crear, p, 1, "C18 "+p); err != nil {
			return err
		}
	}
	for nombre, fd := range a.funciones {
		if fd.Recv != nil && len(fd.Recv.List) > 0 && bytes.Contains(nodo(a.fset, fd.Recv.List[0].Type), []byte("agregadoO4aM38")) {
			return fmt.Errorf("C18 método/getter O4a prohibido: %s", nombre)
		}
	}
	i, j := bytes.Index(b, []byte("type agregadoO4aM38 struct {")), bytes.Index(b, []byte("type sellosCustodiaO3cM38 struct {"))
	if i < 0 || j <= i {
		return fmt.Errorf("agregado O4a opaco ausente")
	}
	cuerpo := b[i:j]
	for _, p := range []string{"autoridad", "custodia", "identidad", "primera", "ahoraCaso", "finCaso", "retornoCont"} {
		if !bytes.Contains(cuerpo, []byte(p)) {
			return fmt.Errorf("O4a omite %s", p)
		}
	}
	for _, p := range []string{"ticket", "nonce", "pidfd", "func ("} {
		if bytes.Contains(cuerpo, []byte(p)) {
			return fmt.Errorf("O4a expone %s", p)
		}
	}
	return nil
}

func (a *analisis) prohibidas() error {
	if err := a.validarCFyFrontera(); err != nil {
		return err
	}
	prohibidas := map[string]bool{"Start": true, "Run": true, "Output": true, "CombinedOutput": true, "StartProcess": true, "Wait": true, "Kill": true, "Signal": true, "NewFile": true, "Command": true, "CommandContext": true}
	textos := []string{"waitid", "pidfd_open", "F_DUPFD", "SYS_DUP", "SIGSTOP", "go func", "init()", "os.NewFile", "Process.Signal", "Process.Kill"}
	for n, f := range a.archivos {
		var fallo error
		ast.Inspect(f, func(x ast.Node) bool {
			if c, ok := x.(*ast.CallExpr); ok && prohibidas[nombreLlamada(c)] {
				fallo = fmt.Errorf("API %s en %s:%d", nombreLlamada(c), n, a.fset.Position(c.Pos()).Line)
				return false
			}
			return fallo == nil
		})
		if fallo != nil {
			return fallo
		}
		for _, p := range textos {
			if bytes.Contains(a.contenido[n], []byte(p)) {
				return fmt.Errorf("marca prohibida %s en %s", p, n)
			}
		}
	}
	if contarEnTodos(a.contenido, "syscall.SIGCONT") != 1 || contarEnTodos(a.contenido, "syscall.SIGKILL") != 1 || contarEnTodos(a.contenido, "esperarConLeaseO3aM38(") != 1 {
		return fmt.Errorf("cardinalidad efectos CONT/KILL/Wait divergente")
	}
	if contarEnTodos(a.contenido, "syscall.SIGTERM") != 1 || !bytes.Contains(a.de("revalidacion.go"), []byte("pdeathsig == int32(syscall.SIGTERM)")) {
		return fmt.Errorf("SIGTERM fuera de la comparación PDEATHSIG")
	}
	if contarEnTodos(a.contenido, "syscall.Write") != 0 || contarEnTodos(a.contenido, ".Write(") != 0 {
		return fmt.Errorf("escritura TERMINAL/FD presente")
	}
	if err := exigir(a.de("handoff.go"), "cerrarUnoConLeaseO3aM38(c.lease, c.terminal", 1, "cierre TERMINAL único C7"); err != nil {
		return err
	}
	return nil
}

func (a *analisis) ownership() error {
	for n, f := range a.archivos {
		for _, imp := range f.Imports {
			v, _ := strconv.Unquote(imp.Path.Value)
			if v == "testing" || strings.Contains(v, "tools/o3c") {
				return fmt.Errorf("arista productiva a prueba/herramienta en %s", n)
			}
		}
	}
	return nil
}

func (a *analisis) construirOwnership() {
	tipos := map[string]bool{}
	for _, f := range a.archivos {
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.TYPE {
				continue
			}
			for _, s := range g.Specs {
				if ts, ok := s.(*ast.TypeSpec); ok {
					tipos[ts.Name.Name] = true
				}
			}
		}
	}
	for n, f := range a.archivos {
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.TYPE {
				continue
			}
			for _, s := range g.Specs {
				ts, ok := s.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, campo := range st.Fields.List {
					if len(campo.Names) == 1 && campo.Names[0].Name == "auto" {
						continue
					}
					for _, destino := range nombresTipo(campo.Type) {
						if tipos[destino] {
							p := a.fset.Position(campo.Pos())
							a.aristas = append(a.aristas, arista{"tipo:" + ts.Name.Name, "tipo:" + destino, "ownership", n, p.Line})
						}
					}
				}
			}
		}
	}
}

func (a *analisis) esquemaExacto(nombre string, esperado map[string]string) error {
	encontrado := map[string]string{}
	for _, f := range a.archivos {
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.TYPE {
				continue
			}
			for _, s := range g.Specs {
				ts, ok := s.(*ast.TypeSpec)
				if !ok || ts.Name.Name != nombre {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					return fmt.Errorf("no es struct")
				}
				for _, campo := range st.Fields.List {
					for _, n := range campo.Names {
						encontrado[n.Name] = expresion(campo.Type)
					}
				}
			}
		}
	}
	if len(encontrado) != len(esperado) {
		return fmt.Errorf("cardinalidad campos=%d esperada=%d", len(encontrado), len(esperado))
	}
	for n, tipo := range esperado {
		if encontrado[n] != tipo {
			return fmt.Errorf("campo %s tipo=%q esperado=%q", n, encontrado[n], tipo)
		}
	}
	return nil
}
