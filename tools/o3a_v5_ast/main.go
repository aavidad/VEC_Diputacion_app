package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var fuentes = []string{
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1.go",
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_control_preinicio.go",
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_control_preinicio_pruebas.go",
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operativo.go",
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_sobre_s0.go",
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_autoridad.go",
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_preparacion.go",
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_inicio.go",
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas.go",
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas_adversas.go",
}

type arista struct {
	Desde, Hacia, Archivo string
	Linea                 int
}
type informe struct {
	Definiciones  []string `json:"definiciones"`
	Raices        []string `json:"raices"`
	Aristas       []arista `json:"aristas"`
	LlamadasStart []string `json:"llamadas_start"`
}

func fatal(f string, a ...any) { fmt.Fprintf(os.Stderr, "ast_o3a_v5=fallo "+f+"\n", a...); os.Exit(1) }

func main() {
	dir := flag.String("dir", "", "directorio pruebas_sql proyectado")
	flag.Parse()
	if *dir == "" {
		fatal("falta -dir")
	}
	fset := token.NewFileSet()
	var fs []*ast.File
	contenido := map[string][]byte{}
	porNombre := map[string]string{}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}
	for _, n := range fuentes {
		datos, err := os.ReadFile(filepath.Join(*dir, n))
		if err != nil {
			fatal("lectura %s: %v", n, err)
		}
		contenido[n] = datos
		f, err := parser.ParseFile(fset, filepath.Join(*dir, n), datos, parser.AllErrors)
		if err != nil {
			fatal("parseo %s: %v", n, err)
		}
		fs = append(fs, f)
		for _, d := range f.Decls {
			if x, ok := d.(*ast.FuncDecl); ok {
				porNombre[x.Name.Name] = n
			}
		}
	}
	conf := types.Config{Importer: importer.Default()}
	if _, err := conf.Check("evidencia/o3a_v5", fset, fs, info); err != nil {
		fatal("tipado: %v", err)
	}
	r := informe{}
	entrantes := map[string]int{}
	waitsRetirada, cierresTicket := 0, 0
	prohibidas := map[string]bool{"Command": true, "CommandContext": true, "LookPath": true, "Run": true, "Output": true, "CombinedOutput": true, "StartProcess": true, "Kill": true}
	for i, f := range fs {
		archivo := fuentes[i]
		for _, d := range f.Decls {
			if strings.Contains(archivo, "arranque_pruebas") {
				switch x := d.(type) {
				case *ast.FuncDecl:
					if x.Name.Name == "init" {
						fatal("init en G7")
					}
				case *ast.GenDecl:
					if x.Tok == token.VAR {
						for _, s := range x.Specs {
							if v, ok := s.(*ast.ValueSpec); ok {
								for _, n := range v.Names {
									bajo := strings.ToLower(n.Name)
									if strings.Contains(bajo, "hook") || strings.Contains(bajo, "mock") {
										fatal("global/hook/mock en G7")
									}
								}
							}
						}
					}
					if x.Tok == token.TYPE {
						for _, s := range x.Specs {
							if t, ok := s.(*ast.TypeSpec); ok && strings.Contains(strings.ToLower(t.Name.Name), "mock") {
								fatal("mock en G7")
							}
						}
					}
				}
			}
			fd, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			desde := fd.Name.Name
			r.Definiciones = append(r.Definiciones, archivo+":"+desde)
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING && productivoArchivo(archivo) {
					v, _ := strconv.Unquote(lit.Value)
					if strings.HasPrefix(v, "/proc/") && v != "/proc/self/fd/8" {
						fatal("ruta /proc prohibida en %s:%d", archivo, fset.Position(lit.Pos()).Line)
					}
				}
				c, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				hacia := ""
				switch x := c.Fun.(type) {
				case *ast.Ident:
					hacia = x.Name
				case *ast.SelectorExpr:
					hacia = x.Sel.Name
				}
				if _, local := porNombre[hacia]; local {
					p := fset.Position(c.Pos())
					r.Aristas = append(r.Aristas, arista{desde, hacia, archivo, p.Line})
					entrantes[hacia]++
				}
				productivoO3a := productivoArchivo(archivo)
				if productivoO3a && hacia == "Start" {
					p := fset.Position(c.Pos())
					r.LlamadasStart = append(r.LlamadasStart, fmt.Sprintf("%s:%d:%s", archivo, p.Line, desde))
					if !strings.Contains(archivo, "arranque_inicio.go") {
						fatal("Start fuera de G6c: %s", r.LlamadasStart[len(r.LlamadasStart)-1])
					}
				}
				if strings.Contains(archivo, "arranque_pruebas") && hacia == "Start" {
					fatal("Start directo en G7")
				}
				if strings.HasSuffix(archivo, "arranque_pruebas.go") && hacia == "esperarConLeaseO3aM38" {
					fatal("Wait test-only fuera de G7b")
				}
				if productivoO3a && prohibidas[hacia] && hacia != "Start" {
					fatal("API prohibida %s en %s:%d", hacia, archivo, fset.Position(c.Pos()).Line)
				}
				if productivoO3a && hacia == "esperarConLeaseO3aM38" {
					if desde != "retirarConHijoO3aM38" {
						fatal("Wait fuera de retirada: %s", desde)
					}
					waitsRetirada++
				}
				if productivoO3a && hacia == "cerrarUnoConLeaseO3aM38" && len(c.Args) > 1 {
					if s, ok := c.Args[1].(*ast.SelectorExpr); ok && s.Sel.Name == "ticketEscritor" && desde == "retirarConHijoO3aM38" {
						cierresTicket++
					}
				}
				if productivoO3a && hacia == "Write" {
					if s, ok := c.Fun.(*ast.SelectorExpr); ok {
						if x, ok := s.X.(*ast.SelectorExpr); ok && x.Sel.Name == "ticketEscritor" {
							fatal("escritura ticket productiva")
						}
					}
				}
				if productivoO3a && hacia == "Close" {
					if desde == "barreraDespuesStartO3aM38" || desde == "avanzarArranqueO3aM38" {
						fatal("cierre directo antes de handoff")
					}
					if s, ok := c.Fun.(*ast.SelectorExpr); ok {
						if x, ok := s.X.(*ast.SelectorExpr); ok && (x.Sel.Name == "controlFD" || x.Sel.Name == "terminal" || x.Sel.Name == "destinados") {
							fatal("cierre directo de capacidad O3a")
						}
					}
				}
				return true
			})
		}
	}
	if len(r.LlamadasStart) != 1 {
		fatal("cardinalidad Start=%d", len(r.LlamadasStart))
	}
	if waitsRetirada != 1 {
		fatal("cardinalidad Wait en retirada=%d", waitsRetirada)
	}
	if cierresTicket != 1 {
		fatal("cardinalidad cierre ticket en retirada=%d", cierresTicket)
	}
	g6b := "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_preparacion.go"
	g6a := "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_autoridad.go"
	g4 := "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_control_preinicio.go"
	g6c := "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_inicio.go"
	g1 := "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1.go"
	if suma := fmt.Sprintf("%x", sha256.Sum256(contenido[g4])); suma != "2befe2a4c16fc7a57aacd421ea6c8419ab49160bb2ae0d0eb6f03786194aa744" {
		fatal("G4 modificado: %s", suma)
	}
	exigirLiteralUnico(contenido[g6b], "e.control.fase != controlPreinicioS3M38", "guarda S3")
	exigirLiteralUnico(contenido[g6b], "!observadorValido || contadorSenal != e.baselineSenal", "observador preparación")
	exigirLiteralUnico(contenido[g6b], "contadorSenal, _, observadorValido := e.observador.observar()", "observador obligatorio")
	exigirLiteralUnico(contenido[g6b], "e.tid != syscall.Gettid() || e.ppid != os.Getppid() || !observadorValido", "validez observador preparación")
	exigirLiteralUnico(contenido[g6c], "if !valido || contador != c.baselineSenal {", "observador vivo")
	exigirLiteralUnico(contenido[g6c], "if !observadorTransferido {", "transferencia observador")
	exigirLiteralUnico(contenido[g6a], "if o == nil || o.auto != o || o.registro == nil || o.registro.observadores[o] != o.generacion || o.registro.tid != syscall.Gettid() {", "pertenencia observador al observar")
	exigirLiteralUnico(contenido[g6a], "return palabra, syscall.Signal(uint8(palabra >> 2)), estado == 1 || estado == 2", "signo observador")
	exigirLiteralUnico(contenido[g6c], "if err != nil || vuelta <= c.vueltaInicio {", "vuelta posterior")
	exigirLiteralUnico(contenido[g6a], "c == nil || t.celda != c || c.auto != c || c.reloj != r", "identidad testigo")
	exigirLiteralUnico(contenido[g6a], "c.reloj != r ||\n\t\tc.tid != r.tid", "reloj testigo")
	exigirLiteralUnico(contenido[g6a], "!c.consumo.CompareAndSwap(0, 1)", "CAS testigo")
	exigirLiteralUnico(contenido[g6c], "c.pidfdReserva = int(reserva)", "reserva distinta")
	exigirLiteralUnico(contenido[g6c], "if n == 1 && p.retorno == 1 {", "terminalidad estricta")
	exigirLiteralUnico(contenido[g6c], "if n == 0 && p.retorno == 0 {", "vivo estricto")
	exigirLiteralUnico(contenido[g6c], "func barreraDespuesStartO3aM38(c *custodiaO3aM38) error {\n\tresultado, err :=", "precedencia CONTROL")
	exigirLiteralUnico(contenido[g6c], "syscall.Syscall(syscall.SYS_FCNTL, uintptr(pidfdPrimario), syscall.F_DUPFD_CLOEXEC, minFDDuplicadoM38)", "duplicación reserva única")
	exigirLiteralUnico(contenido[g6c], "pidfdOpaco, deltaValido := deltaPidfdO3aM38(c.lease.pre, conPidfd, pidfdPrimario, reservaFD)", "identidad delta pidfd")
	exigirLiteralUnico(contenido[g6c], "if errStart != nil {\n\t\treturn retirarConHijoO3aM38(c, errStart, finRetirada)\n\t}", "Start fallido con hijo retira")
	exigirLiteralUnico(contenido[g1], "if len(os.Args) == 2 && os.Args[1] == \"--supervisar-m38\" {\n\t\tos.Exit(supervisarM38())\n\t}", "modo supervisor cerrado")
	exigirLiteralUnico(contenido[g6b], "autoridad: autoridad, control: e.control, controlFD: e.controlFD, terminal: e.terminal", "propiedad TERMINAL")
	exigirLiteralUnico(contenido[g6b], "raiz, runner, ticketLector}", "mapa ExtraFiles")
	exigirLiteralUnico(contenido[g6b], "ExtraFiles:  []*os.File{destinados[5], destinados[6], destinados[7], destinados[8], raiz, runner, ticketLector}", "orden ExtraFiles exacto")
	exigirLiteralUnico(contenido[g6b], "Stdin: destinados[4], Stdout: salida, Stderr: errorCaso", "stdout stderr exactos")
	exigirLiteralUnico(contenido[g6b], "forma.modo&syscall.S_IFMT != syscall.S_IFREG || forma.uid != uint32(os.Geteuid()) ||\n\t\tforma.modo&07777 != 0600", "regular tipo y propietario")
	exigirLiteralUnico(contenido[g6b], "forma.modo&07777 != 0600 || forma.enlaces != 1 || forma.offset != 0", "regular modo")
	exigirLiteralUnico(contenido[g6b], "forma.enlaces != 1 || forma.offset != 0 ||\n\t\tforma.flags != flagsEsperados", "regular enlaces")
	exigirLiteralUnico(contenido[g6b], "forma.offset != 0 ||\n\t\tforma.flags != flagsEsperados", "regular offset")
	exigirLiteralUnico(contenido[g6b], "forma.flags != flagsEsperados ||\n\t\tforma.fdflags&syscall.FD_CLOEXEC == 0", "regular flags")
	exigirLiteralUnico(contenido[g6b], "forma.modo&syscall.S_IFMT != syscall.S_IFIFO", "pipe tipo")
	exigirLiteralUnico(contenido[g6b], "forma.modo&07777 != 0600 || forma.enlaces != 1 || forma.flags&syscall.O_ACCMODE != syscall.O_RDONLY", "pipe modo")
	exigirLiteralUnico(contenido[g6b], "forma.enlaces != 1 || forma.flags&syscall.O_ACCMODE != syscall.O_RDONLY", "pipe enlaces")
	exigirLiteralUnico(contenido[g6b], "forma.flags&syscall.O_ACCMODE != syscall.O_RDONLY ||\n\t\tforma.flags != syscall.O_NONBLOCK", "pipe acceso")
	exigirLiteralUnico(contenido[g6b], "forma.flags != syscall.O_NONBLOCK || forma.fdflags&syscall.FD_CLOEXEC == 0", "pipe nonblock")
	exigirLiteralUnico(contenido[g6b], "forma.modo&syscall.S_IFMT != syscall.S_IFDIR", "raíz tipo")
	exigirLiteralUnico(contenido[g6b], "!identidadFisicaO3aM38(forma, sellada.identidad)", "raíz identidad")
	exigirLiteralUnico(contenido[g6b], "forma.modo&syscall.S_IFMT != syscall.S_IFCHR || forma.rdev != 259", "devnull tipo")
	exigirLiteralUnico(contenido[g6b], "forma.rdev != 259 ||\n\t\tforma.flags&syscall.O_ACCMODE != syscall.O_RDONLY", "devnull rdev")
	exigirLiteralUnico(contenido[g6b], "formas[5], err = validarRegularO3aM38(e.terminal, syscall.O_WRONLY, true)", "validación TERMINAL")
	exigirLiteralUnico(contenido[g6b], "Env: []string{\"LC_ALL=C\", \"PATH=/usr/local/go/bin:/usr/bin:/bin\"}", "entorno exacto")
	exigirLiteralUnico(contenido[g6c], "forma != c.huellasDestinadas[i] || forma.fdflags&syscall.FD_CLOEXEC == 0", "revalidación viva destinados")
	exigirLiteralUnico(contenido[g6c], "if err != nil || forma != c.huellasDestinadas[i] || forma.fdflags&syscall.FD_CLOEXEC == 0 {", "revalidación íntegra destinados")
	exigirLiteralUnico(contenido[g6c], "for i, f := range c.destinados {\n\t\tcerrado, err :=", "cierre completo destinados")
	if bytes.Contains(contenido[g6c], []byte("Syscall6(436")) {
		fatal("close_range en padre")
	}
	exigirLiteralUnico(contenido[g6a], "maxFDInspeccionM38     = 1_048_576", "máximo FD")
	exigirLiteralUnico(contenido[g6a], "minFDDuplicadoM38      = 10", "mínimo FD")
	exigirLiteralUnico(contenido[g6a], "duracionRetiradaO3aM38 = 3 * time.Second", "retirada tres segundos")
	exigirLiteralUnico(contenido[g6a], "reservaO3bO3cM38       = time.Second", "reserva un segundo")
	exigirLiteralUnico(contenido[g6c], "var buffer [1024]byte", "buffer 1024")
	exigirLiteralUnico(contenido[g6c], "for lecturas < 4 && total < 4096", "lecturas y total")
	exigirLiteralUnico(contenido[g6c], "if interrupciones > 8", "EINTR ocho")
	exigirLiteralUnico(contenido[g6c], "origen: retiradaConHijoO3aM38, primera: primera, controlFD: c.controlFD", "primera observación")
	exigirLiteralUnico(contenido[g6a], "func fatalO3aM38() { os.Exit(estadoFallo) }", "fatal directo")
	exigirLiteralUnico(contenido[g6c], "resultado, err := leerControlBarreraO3aM38(c, false)\n\tif resultado != barreraVerdeO3aM38 || err != nil {", "causa antes de entrega")
	exigirLiteralUnico(contenido[g6a], "a == nil || a.auto != a || a.registro == nil || a.registro.preflight[a] != a.generacion", "preflight completo")
	exigirLiteralUnico(contenido[g6b], "limite.Cur < minFDDuplicadoM38 || limite.Cur > maxFDInspeccionM38", "RLIMIT acotado")
	exigirLiteralUnico(contenido[g6b], "time.Until(e.finBootstrap) < 4*time.Second", "límite pre-Start")
	exigirLiteralUnico(contenido[g6a], "type agregadoO3aM38 struct{ custodia *custodiaO3aM38 }", "agregado opaco")
	exigirLiteralUnico(contenido[g6c], "if resultado == controlPreinicioNecesitaDatosM38 || c.control.lector.estado == lectorAbiertoParcialM38 {", "parcial cerrado")
	exigirLiteralUnico(contenido[g6b], "if fd >= 3 && h.identidad.fdflags&syscall.FD_CLOEXEC == 0 {", "FD ajeno CLOEXEC")
	guardaPreflight := "a == nil || a.auto != a || a.registro == nil || a.registro.preflight[a] != a.generacion || a.registro.tid != syscall.Gettid() || !a.consumo.CompareAndSwap(0, 1)"
	exigirLiteralUnico(contenido[g6a], guardaPreflight, "guarda preflight atómica")
	exigirLiteralUnico(contenido[g6a], "func consumirPreflightPidfdGoM38(a *acreditacionPidfdGoM38) bool {\n\tif "+guardaPreflight, "orden preflight")
	exigirLiteralUnico(contenido[g6b], "reloj: e.reloj, vueltaInicio: e.vueltaInicio, finBootstrap: e.finBootstrap", "deadline heredado")
	exigirLiteralUnico(contenido[g6c], "if err = entornoEstableO3aM38(c, reservaO3bO3cM38); err != nil {\n\t\treturn err\n\t}\n\treturn nil", "prehandoff un segundo")
	exigirLiteralUnico(contenido[g6c], "c.destinados = nil\n\tif err = entornoEstableO3aM38(c, reservaO3bO3cM38); err != nil {", "revalidación final")
	exigirLiteralUnico(contenido[g6b], "for fd := 0; fd < int(limite.Cur); fd++ {", "inventario 0..Cur")
	exigirLiteralUnico(contenido[g6b], "if _, existe := s.mapa[fd]; !existe {", "inventario 0/1/2")
	exigirLiteralUnico(contenido[g6a], "l.auto, r.leases[l] = l, g", "registro lease")
	exigirLiteralUnico(contenido[g6a], "delete(l.registro.leases, l)", "liberación lease")
	exigirLiteralUnico(contenido[g6a], "o.auto, r.observadores[o] = o, g", "registro observador")
	exigirLiteralUnico(contenido[g6a], "delete(o.registro.observadores, o)", "liberación observador")
	exigirLiteralUnico(contenido[g6a], "!l.estado.CompareAndSwap(estadoPrevio, 2)", "begin físico")
	exigirLiteralUnico(contenido[g6c], "permisoStart, permisoValido := c.lease.comenzarCritico(operacionStartO3aM38, 1)", "begin Start")
	exigirLiteralUnico(contenido[g6a], "p.lease == l && p.generacion == l.secuencia", "permiso lease")
	exigirLiteralUnico(contenido[g6a], "p.generacion == l.secuencia && p.operacion == l.operacion", "permiso generación")
	exigirLiteralUnico(contenido[g6a], "p.cardinalidad == l.cardinal && p.objetivos == l.objetivos", "permiso resultado")
	exigirLiteralUnico(contenido[g6a], "} else if !snapshotsIgualesO3aM38(actual, l.pre) {\n\t\tl.estado.Store(5)", "fallo físico enclavado")
	exigirLiteralUnico(contenido[g6a], "p.objetivos == l.objetivos && l.estado.Load() == 2", "pending no entregable")
	g7a := "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas.go"
	g7b := "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas_adversas.go"
	exigirLiteralUnico(contenido[g7b], "fallos = append(fallos, retirarDirectorioFixtureO3aM38(f))\n\tif f.canalSenal != nil {", "cleanup G7b ordenado")
	for _, marca := range []string{"aristaNoPermitidaO3aM38", "cicloUnoO3aM38", "segundoOwner", "segundoConstructorO3aM38", "escribirAutoridadO3aM38", "escribirEstadoO3aM38", "cleanupFueraO3aM38", "estadoGlobal", "callback"} {
		if bytes.Contains(contenido[g7a], []byte(marca)) {
			fatal("estructura G7 prohibida: %s", marca)
		}
	}
	if bytes.Contains(contenido[g6c], []byte("c.pidfdReserva = c.pidfdPrimario")) {
		fatal("reserva durante retirada B")
	}
	if bytes.Contains(contenido[g6a], []byte("syscall.SYS_GETPID")) {
		fatal("syscall durante consolidación")
	}
	if bytes.Contains(contenido[g6c], []byte("segundoParser")) {
		fatal("segundo parser O3a")
	}
	for _, d := range r.Definiciones {
		n := d[strings.LastIndex(d, ":")+1:]
		if entrantes[n] == 0 {
			r.Raices = append(r.Raices, d)
		}
	}
	sort.Strings(r.Definiciones)
	sort.Strings(r.Raices)
	sort.Slice(r.Aristas, func(i, j int) bool {
		a, b := r.Aristas[i], r.Aristas[j]
		if a.Desde != b.Desde {
			return a.Desde < b.Desde
		}
		if a.Hacia != b.Hacia {
			return a.Hacia < b.Hacia
		}
		return a.Linea < b.Linea
	})
	b, _ := json.MarshalIndent(r, "", "  ")
	fmt.Println(string(b))
	fmt.Fprintln(os.Stderr, "ast_o3a_v5=ok tipado=ok start=1")
}

func productivoArchivo(archivo string) bool {
	return strings.Contains(archivo, "arranque_") && !strings.Contains(archivo, "pruebas")
}

func exigirLiteralUnico(datos []byte, literal, etiqueta string) {
	if n := bytes.Count(datos, []byte(literal)); n != 1 {
		fatal("%s cardinalidad=%d", etiqueta, n)
	}
}
