package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
	"runtime"
	"sort"
	"strconv"
	"strings"
)

const base = "d9f8aeb547f5d1b3b9ab3eb786382f78ef964e28"

var fuentes = map[string]string{
	"captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad.go": "0b95fe0cfda784089c904941f56a3ad52a0eb519ef43b2057af22d24498c0e53",
	"captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_barrera.go":   "0499ede483615d57d5438d579f32580649b40fbf34b3be6bc26d31fa6c86c02d",
	"captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_ticket.go":    "4c3e7b888107a20b0ec2ae2479aa9a30378aad3a23b92ffc398597e6ce3dc484",
	"captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_stop.go":      "99ae6fc4267062863f7e5adbccbe9e3698f08f114f107fd07f8cb6ed2f4af5f2",
	"captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_identidad.go": "803b9a343934e5423e0d3f5f5f4cbfee86f7321790d979e017254a924ac2be7a",
	"captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff.go":   "4b8b856e506b188a750922b4e320502b597a4aacf930507376eec9feccf4a534",
}

var soporte = map[string]bool{
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1.go":                           true,
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_control_preinicio.go":         true,
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_control_preinicio_pruebas.go": true,
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operativo.go":                 true,
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_sobre_s0.go":                  true,
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_autoridad.go":        true,
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_preparacion.go":      true,
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_inicio.go":           true,
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas.go":          true,
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas_adversas.go": true,
}

type arista struct {
	Desde, Hacia, Archivo string
	Linea                 int
}

type resultado struct {
	Base, Go, FuentesSHA, DAG_SHA string
	Fuentes                       map[string]string
	Definiciones, Raices          []string
	Aristas                       []arista
	Aserciones                    []string
}

type analisis struct {
	fset      *token.FileSet
	archivos  map[string]*ast.File
	contenido map[string][]byte
	funciones map[string]*ast.FuncDecl
	info      *types.Info
	aristas   []arista
	entrantes map[string]int
}

func fatal(formato string, args ...any) {
	fmt.Fprintf(os.Stderr, "ast_o3b_p7=NO_GO "+formato+"\n", args...)
	os.Exit(1)
}

func main() {
	dir := flag.String("dir", "", "directorio pruebas_sql")
	flag.Parse()
	if *dir == "" || flag.NArg() != 0 {
		fatal("uso: -dir DIRECTORIO")
	}
	a, err := cargar(*dir)
	if err != nil {
		fatal("%v", err)
	}
	r, err := a.verificar()
	if err != nil {
		fatal("%v", err)
	}
	datos, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		fatal("json: %v", err)
	}
	fmt.Println(string(datos))
	fmt.Fprintf(os.Stderr, "ast_o3b_p7=GO tipado=GO aserciones=%d fuentes=6\n", len(r.Aserciones))
}

func cargar(dir string) (*analisis, error) {
	entradas, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("leer directorio: %w", err)
	}
	a := &analisis{fset: token.NewFileSet(), archivos: map[string]*ast.File{}, contenido: map[string][]byte{},
		funciones: map[string]*ast.FuncDecl{}, entrantes: map[string]int{}, info: &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}}
	var tipados []*ast.File
	for _, e := range entradas {
		n := e.Name()
		_, objetivo := fuentes[n]
		if e.IsDir() || (!objetivo && !soporte[n]) {
			continue
		}
		datos, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			return nil, fmt.Errorf("leer %s: %w", n, err)
		}
		f, err := parser.ParseFile(a.fset, n, datos, parser.AllErrors)
		if err != nil {
			return nil, fmt.Errorf("parsear %s: %w", n, err)
		}
		tipados = append(tipados, f)
		if objetivo {
			a.archivos[n], a.contenido[n] = f, datos
			for _, d := range f.Decls {
				if fd, ok := d.(*ast.FuncDecl); ok {
					a.funciones[fd.Name.Name] = fd
				}
			}
		}
	}
	if len(a.archivos) != len(fuentes) {
		return nil, fmt.Errorf("fuentes O3b encontradas=%d, esperadas=%d", len(a.archivos), len(fuentes))
	}
	conf := types.Config{Importer: importer.Default()}
	if _, err := conf.Check("evidencia/o3b_p7", a.fset, tipados, a.info); err != nil {
		return nil, fmt.Errorf("tipado productivo: %w", err)
	}
	return a, nil
}

func (a *analisis) verificar() (resultado, error) {
	r := resultado{Base: base, Go: runtime.Version(), Fuentes: map[string]string{}}
	for n, esperado := range fuentes {
		suma := fmt.Sprintf("%x", sha256.Sum256(a.contenido[n]))
		if suma != esperado {
			return r, fmt.Errorf("SHA %s=%s, esperado=%s", n, suma, esperado)
		}
		r.Fuentes[n] = suma
	}
	r.FuentesSHA = resumenFuentes(r.Fuentes)
	for nombre, fd := range a.funciones {
		r.Definiciones = append(r.Definiciones, nombre)
		archivo := filepath.Base(a.fset.Position(fd.Pos()).Filename)
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			hacia := nombreLlamada(c)
			if _, local := a.funciones[hacia]; local {
				p := a.fset.Position(c.Pos())
				a.aristas = append(a.aristas, arista{nombre, hacia, archivo, p.Line})
				a.entrantes[hacia]++
			}
			return true
		})
	}
	if ciclo := detectarCiclo(a.funciones, a.aristas); ciclo != "" {
		return r, fmt.Errorf("ownership/call DAG cíclico: %s", ciclo)
	}
	for _, n := range r.Definiciones {
		if a.entrantes[n] == 0 {
			r.Raices = append(r.Raices, n)
		}
	}
	r.Aristas = a.aristas
	sort.Strings(r.Definiciones)
	sort.Strings(r.Raices)
	sort.Slice(r.Aristas, func(i, j int) bool { return fmt.Sprint(r.Aristas[i]) < fmt.Sprint(r.Aristas[j]) })
	r.DAG_SHA = hashJSON(r.Aristas)
	comprobaciones := []struct {
		nombre string
		fn     func() error
	}{
		{"entrada_unica_y_sin_cli", a.entrada}, {"maquina_total_y_cas", a.maquina},
		{"ticket_escritura_cierre_unicos", a.ticket}, {"wait_solo_retirada_terminal", a.wait},
		{"pidfd_senal_cero_grupo", a.senal}, {"proc_stat_unico_acotado", a.proc},
		{"dos_T_identidad_completa", a.identidad}, {"revalidacion_handoff_conjunto", a.handoff},
		{"o3c_sin_efectos", a.o3c}, {"apis_prohibidas_ausentes", a.prohibidas},
		{"ownership_dag_sin_test_productivo", a.ownership},
	}
	for _, c := range comprobaciones {
		if err := c.fn(); err != nil {
			return r, fmt.Errorf("%s: %w", c.nombre, err)
		}
		r.Aserciones = append(r.Aserciones, c.nombre)
	}
	return r, nil
}

func (a *analisis) entrada() error {
	fd := a.funciones["consumirAutoridadO3bM38"]
	if fd == nil || fd.Type == nil || len(fd.Type.Params.List) != 1 || !doblePunteroA(fd.Type.Params.List[0].Type, "agregadoO3aM38") {
		return fmt.Errorf("firma de entrada ausente")
	}
	entradas := 0
	for _, f := range a.funciones {
		if f.Type != nil {
			for _, p := range f.Type.Params.List {
				if doblePunteroA(p.Type, "agregadoO3aM38") {
					entradas++
				}
			}
		}
	}
	if entradas != 1 {
		return fmt.Errorf("entradas **agregadoO3aM38=%d", entradas)
	}
	return exigir(a.contenidoDe("autoridad.go"), "*entrada = nil", 1, "consumo del puntero")
}

func (a *analisis) maquina() error {
	b := a.contenidoDe("autoridad.go")
	for _, estado := range []string{"B0Recibido", "B1BarreraVerde", "B2TicketCerrado", "B3StopObservado", "B4IdentidadAcreditada", "B4TTransfiriendo", "B5Capturado", "B7Retirando", "B8Retirado", "BFFatal"} {
		if !bytes.Contains(b, []byte("captura"+estado+"M38")) {
			return fmt.Errorf("falta %s", estado)
		}
	}
	if err := exigir(b, "consumida.CompareAndSwap(1, 2)", 1, "CAS de consumo"); err != nil {
		return err
	}
	if err := exigir(b, "func transicionCapturaO3bM38", 1, "transición total"); err != nil {
		return err
	}
	return nil
}

func (a *analisis) ticket() error {
	b := a.contenidoDe("ticket.go")
	// Los dos callsites son alternativas dentro del único escritor lógico:
	// permiso preadquirido y permisos de continuaciones parciales.
	if err := exigir(b, "syscall.Write(preparado.fd", 2, "callsites del escritor lógico"); err != nil {
		return err
	}
	for nombre, fd := range a.funciones {
		if nombre == "escribirTicketO3bM38" {
			continue
		}
		if contarLlamadas(fd, "Write") != 0 {
			return fmt.Errorf("Write fuera del escritor lógico: %s", nombre)
		}
	}
	if err := exigir(b, "errCierre := archivo.Close()", 1, "Close ticket"); err != nil {
		return err
	}
	return exigir(b, "if err := escribirTicketO3bM38(a); err != nil {", 1, "Write antes de Close")
}

func (a *analisis) wait() error {
	for n, b := range a.contenido {
		if bytes.Contains(b, []byte(".Wait(")) || bytes.Contains(b, []byte("waitid")) {
			return fmt.Errorf("Wait/waitid en %s; O3b delega retirada", n)
		}
	}
	return nil
}

func (a *analisis) senal() error {
	b := a.contenidoDe("stop.go")
	if err := exigir(b, "syscall.Syscall6(sysPidfdSendSignal, uintptr(pidfd), 0, 0, pidfdSignalGrupoO3bM38, 0, 0)", 1, "pidfd señal cero y flag grupo"); err != nil {
		return err
	}
	return exigir(b, "pidfdSignalGrupoO3bM38 = uintptr(1 << 2)", 1, "flag grupo")
}

func (a *analisis) proc() error {
	for n, b := range a.contenido {
		for i := 0; i < len(b); i++ {
			if bytes.HasPrefix(b[i:], []byte("\"/proc/")) && !bytes.Contains(b[i:], []byte("\"/proc/\" + strconv.Itoa(pid) + \"/stat\"")) {
				return fmt.Errorf("ruta /proc no autorizada en %s", n)
			}
		}
	}
	b := a.contenidoDe("stop.go")
	if err := exigir(b, "\"/proc/\" + strconv.Itoa(pid) + \"/stat\"", 1, "ruta stat"); err != nil {
		return err
	}
	if err := exigir(b, "maximoStatO3bM38+1", 1, "detección >4096"); err != nil {
		return err
	}
	return exigir(b, "n > maximoStatO3bM38", 1, "límite lectura")
}

func (a *analisis) identidad() error {
	b := a.contenidoDe("identidad.go")
	for _, s := range []string{"bytes.LastIndex(datos, []byte(\") \"))", "len(resto) != camposStatO3bM38-3", "primeraRaw, err := leerStatStopO3bM38", "segundaRaw, err := leerStatStopO3bM38", "muestra.pid == a.custodia.cmd.Process.Pid", "muestra.estado == 'T'", "muestra.ppid == os.Getpid()", "muestra.pgid == a.custodia.cmd.Process.Pid", "muestra.sid == a.sidSupervisor", "muestra.inicio > 0", "segunda.pid != primera.pid", "segunda.ppid != primera.ppid", "segunda.pgid != primera.pgid", "segunda.sid != primera.sid"} {
		if err := exigir(b, s, 1, s); err != nil {
			return err
		}
	}
	return nil
}

func (a *analisis) handoff() error {
	b := a.contenidoDe("handoff.go")
	for _, s := range []string{"if err := leerControlO3bM38", "actual, senal, valido := a.custodia.observador.observar()", "acreditarPidfdBarreraO3bM38", "inventarioPostTicketO3bM38", "time.Now().Before(a.custodia.finBootstrap)", "prevalidarTransferenciaO3bM38(a, identidad)", "capturaB4TTransfiriendoM38", "transferirObservadorCapturadoO3bM38", "transferirLeaseCapturadaO3bM38", "a.custodia = nil", "*identidad = identidadProcesoO3bM38{}"} {
		if !bytes.Contains(b, []byte(s)) {
			return fmt.Errorf("falta %s", s)
		}
	}
	if bytes.Index(b, []byte("transferirObservadorCapturadoO3bM38(a.custodia.observador")) > bytes.Index(b, []byte("transferirLeaseCapturadaO3bM38(a.custodia.lease")) {
		return fmt.Errorf("orden lease antes de observador")
	}
	return nil
}

func (a *analisis) o3c() error {
	b := a.contenidoDe("handoff.go")
	if err := exigir(b, "type agregadoO3cM38 struct {", 1, "agregado O3c opaco"); err != nil {
		return err
	}
	for _, s := range []string{"ticketEscritor", "finCaso", "180", "SIGCONT"} {
		if bytes.Contains(b[bIndex(b, "type agregadoO3cM38 struct {"):bIndex(b, "func prevalidarTransferenciaO3bM38")], []byte(s)) {
			return fmt.Errorf("campo O3c prohibido %s", s)
		}
	}
	return nil
}

func (a *analisis) prohibidas() error {
	prohibidas := map[string]bool{"Start": true, "Run": true, "Output": true, "CombinedOutput": true, "StartProcess": true, "Wait": true, "Kill": true, "Signal": true, "NewFile": true, "Command": true, "CommandContext": true}
	textos := []string{"pidfd_open", "F_DUPFD", "SYS_DUP", "close_range", "SIGSTOP", "SIGCONT", "finCaso", "go func", "init()"}
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
		for _, s := range textos {
			if bytes.Contains(a.contenido[n], []byte(s)) {
				return fmt.Errorf("marca prohibida %s en %s", s, n)
			}
		}
	}
	return nil
}

func (a *analisis) ownership() error {
	for n, f := range a.archivos {
		for _, imp := range f.Imports {
			v, _ := strconv.Unquote(imp.Path.Value)
			if v == "testing" || strings.Contains(v, "tools/o3b") {
				return fmt.Errorf("arista productiva a prueba/herramienta en %s", n)
			}
		}
	}
	return nil
}

func (a *analisis) contenidoDe(sufijo string) []byte {
	for n, b := range a.contenido {
		if strings.HasSuffix(n, sufijo) {
			return b
		}
	}
	return nil
}
func exigir(b []byte, patron string, n int, etiqueta string) error {
	if c := bytes.Count(b, []byte(patron)); c != n {
		return fmt.Errorf("%s cardinalidad=%d esperada=%d", etiqueta, c, n)
	}
	return nil
}
func nombreLlamada(c *ast.CallExpr) string {
	switch x := c.Fun.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return x.Sel.Name
	}
	return ""
}
func contarLlamadas(fd *ast.FuncDecl, nombre string) int {
	n := 0
	if fd == nil {
		return 0
	}
	ast.Inspect(fd.Body, func(x ast.Node) bool {
		if c, ok := x.(*ast.CallExpr); ok && nombreLlamada(c) == nombre {
			n++
		}
		return true
	})
	return n
}
func doblePunteroA(e ast.Expr, nombre string) bool {
	a, ok := e.(*ast.StarExpr)
	if !ok {
		return false
	}
	b, ok := a.X.(*ast.StarExpr)
	if !ok {
		return false
	}
	id, ok := b.X.(*ast.Ident)
	return ok && id.Name == nombre
}
func bIndex(b []byte, s string) int {
	i := bytes.Index(b, []byte(s))
	if i < 0 {
		return len(b)
	}
	return i
}
func hashJSON(v any) string {
	b, _ := json.Marshal(v)
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}
func resumenFuentes(m map[string]string) string {
	ns := make([]string, 0, len(m))
	for n := range m {
		ns = append(ns, n)
	}
	sort.Strings(ns)
	h := sha256.New()
	for _, n := range ns {
		fmt.Fprintf(h, "%s\t%s\n", m[n], n)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func detectarCiclo(funciones map[string]*ast.FuncDecl, aristas []arista) string {
	g := map[string][]string{}
	for _, e := range aristas {
		if e.Desde != e.Hacia {
			g[e.Desde] = append(g[e.Desde], e.Hacia)
		}
	}
	estado := map[string]uint8{}
	var visitar func(string) string
	visitar = func(n string) string {
		if estado[n] == 1 {
			return n
		}
		if estado[n] == 2 {
			return ""
		}
		estado[n] = 1
		for _, v := range g[n] {
			if c := visitar(v); c != "" {
				return n + "->" + c
			}
		}
		estado[n] = 2
		return ""
	}
	for n := range funciones {
		if c := visitar(n); c != "" {
			return c
		}
	}
	return ""
}
