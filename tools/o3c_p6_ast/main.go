package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const base = "c0f2a9945ed2fc5648980ee48b91424a04977655"

var fuentes = map[string]string{
	"continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad.go":    "150c46ebeef8b6d2850d735b1f679701d620f0ef54850ab01f20fca986c9a599",
	"continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_revalidacion.go": "35409603d803a6d74288a391bea93239d2246fbe3c0d35eecf2063c0da1fe1aa",
	"continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_cont.go":         "447d5779ea90731b3b53a46870861b79bc95a478bd0fa7540b717ea279bd94be",
	"continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_observacion.go":  "bf2a5814608479cfe03628be31e951087a6da29f21f8a88a053108fa0d6620b0",
	"continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff.go":      "66fb9c71e8c5d5e03cd7a32380986c23a0139720673a9b2348e1bcf03d3ec4cf",
}

func esSoporte(n string) bool {
	return strings.HasPrefix(n, "supervisor_procesos_m38_h0b_") && strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_test.go") ||
		strings.HasPrefix(n, "captura_procesos_m38_h0b_") && strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_test.go")
}

type arista struct {
	Desde, Hacia, Tipo, Archivo string
	Linea                       int
}

type resultado struct {
	Base, Go, FuentesSHA, DAGSHA string
	Fuentes                      map[string]string
	Definiciones, Raices         []string
	Aristas                      []arista
	Aserciones                   []string
}

type analisis struct {
	fset        *token.FileSet
	archivos    map[string]*ast.File
	contenido   map[string][]byte
	funciones   map[string]*ast.FuncDecl
	info        *types.Info
	aristas     []arista
	entrantes   map[string]int
	permitirSHA bool
}

func fatal(formato string, args ...any) {
	fmt.Fprintf(os.Stderr, "ast_o3c_p6=NO_GO "+formato+"\n", args...)
	os.Exit(1)
}

func main() {
	dir := flag.String("dir", "", "directorio pruebas_sql")
	permitirSHA := flag.Bool("permitir-sha", false, "modo mutante: omite solo el sello SHA")
	flag.Parse()
	if *dir == "" || flag.NArg() != 0 {
		fatal("uso: -dir DIRECTORIO")
	}
	a, err := cargar(*dir)
	if err != nil {
		fatal("%v", err)
	}
	a.permitirSHA = *permitirSHA
	r, err := a.verificar()
	if err != nil {
		fatal("%v", err)
	}
	datos, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		fatal("json: %v", err)
	}
	fmt.Println(string(datos))
	fmt.Fprintf(os.Stderr, "ast_o3c_p6=GO tipado=GO aserciones=%d fuentes=5\n", len(r.Aserciones))
}

func cargar(dir string) (*analisis, error) {
	entradas, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("leer directorio: %w", err)
	}
	a := &analisis{fset: token.NewFileSet(), archivos: map[string]*ast.File{}, contenido: map[string][]byte{},
		funciones: map[string]*ast.FuncDecl{}, entrantes: map[string]int{}, info: &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}}
	var tipados []*ast.File
	for _, e := range entradas {
		n := e.Name()
		_, objetivo := fuentes[n]
		if e.IsDir() || (!objetivo && !esSoporte(n)) {
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
		return nil, fmt.Errorf("fuentes O3c encontradas=%d esperadas=%d", len(a.archivos), len(fuentes))
	}
	conf := types.Config{Importer: importer.Default()}
	if _, err := conf.Check("evidencia/o3c_p6", a.fset, tipados, a.info); err != nil {
		return nil, fmt.Errorf("tipado productivo: %w", err)
	}
	return a, nil
}

func (a *analisis) verificar() (resultado, error) {
	r := resultado{Base: base, Go: runtime.Version(), Fuentes: map[string]string{}}
	for n, esperado := range fuentes {
		suma := fmt.Sprintf("%x", sha256.Sum256(a.contenido[n]))
		if err := validarSHA(suma, esperado, a.permitirSHA); err != nil {
			return r, fmt.Errorf("SHA %s=%s esperado=%s", n, suma, esperado)
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
				a.aristas = append(a.aristas, arista{"func:" + nombre, "func:" + hacia, "llamada", archivo, p.Line})
				a.entrantes[hacia]++
			}
			return true
		})
	}
	a.construirOwnership()
	if ciclo := detectarCiclo(a.funciones, a.aristas); ciclo != "" {
		return r, fmt.Errorf("DAG cíclico: %s", ciclo)
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
	r.DAGSHA = hashJSON(r.Aristas)
	comprobaciones := []struct {
		nombre string
		fn     func() error
	}{
		{"entrada_B5_readonly_consumo_lineal", a.entrada},
		{"maquina_C0_CF_total", a.maquina},
		{"autoridad_CAS_owners_aciclica", a.autoridad},
		{"revalidacion_final_y_lease", a.revalidacion},
		{"marca_CONT_unicos_y_ordenados", a.cont},
		{"observacion_union_cerrada_y_precedencia", a.observacion},
		{"handoff_conjunto_y_retirada_C7", a.handoff},
		{"O4_opaco_sin_efectos", a.o4},
		{"APIs_prohibidas_ausentes", a.prohibidas},
		{"DAG_sin_pruebas_productivas", a.ownership},
	}
	for _, c := range comprobaciones {
		if err := c.fn(); err != nil {
			return r, fmt.Errorf("%s: %w", c.nombre, err)
		}
		r.Aserciones = append(r.Aserciones, c.nombre)
	}
	return r, nil
}

func validarSHA(obtenido, esperado string, permitir bool) error {
	if !permitir && obtenido != esperado {
		return fmt.Errorf("huella divergente")
	}
	return nil
}

func (a *analisis) de(sufijo string) []byte {
	for n, b := range a.contenido {
		if strings.HasSuffix(n, sufijo) {
			return b
		}
	}
	return nil
}

func (a *analisis) cuerpo(nombre string) []byte {
	fd := a.funciones[nombre]
	if fd == nil || fd.Body == nil {
		return nil
	}
	n := filepath.Base(a.fset.Position(fd.Pos()).Filename)
	b := a.contenido[n]
	i := a.fset.Position(fd.Body.Pos()).Offset
	j := a.fset.Position(fd.Body.End()).Offset
	if i < 0 || j > len(b) || i >= j {
		return nil
	}
	return b[i:j]
}

func exigir(b []byte, patron string, n int, etiqueta string) error {
	if c := bytes.Count(b, []byte(patron)); c != n {
		return fmt.Errorf("%s cardinalidad=%d esperada=%d", etiqueta, c, n)
	}
	return nil
}

func enOrden(b []byte, patrones []string, etiqueta string) error {
	pos := -1
	for _, p := range patrones {
		i := bytes.Index(b[pos+1:], []byte(p))
		if i < 0 {
			return fmt.Errorf("%s falta/fuera de orden %s", etiqueta, p)
		}
		pos += i + 1
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

func expresion(e ast.Expr) string {
	var b bytes.Buffer
	if err := format.Node(&b, token.NewFileSet(), e); err != nil {
		return ""
	}
	return b.String()
}

func nodo(fset *token.FileSet, n ast.Node) []byte {
	var b bytes.Buffer
	if err := format.Node(&b, fset, n); err != nil {
		return nil
	}
	return b.Bytes()
}

func nombresTipo(e ast.Expr) []string {
	var nombres []string
	ast.Inspect(e, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			nombres = append(nombres, id.Name)
		}
		return true
	})
	return nombres
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

func contarEnTodos(m map[string][]byte, p string) int {
	n := 0
	for _, b := range m {
		n += bytes.Count(b, []byte(p))
	}
	return n
}

func asignacionTopLevelExacta(fd *ast.FuncDecl, lhs, rhs string) error {
	total := 0
	if fd == nil || fd.Body == nil {
		return fmt.Errorf("función ausente")
	}
	for _, st := range fd.Body.List {
		as, ok := st.(*ast.AssignStmt)
		if !ok || len(as.Lhs) != 1 || expresion(as.Lhs[0]) != lhs {
			continue
		}
		total++
		if len(as.Rhs) != 1 || expresion(as.Rhs[0]) != rhs {
			return fmt.Errorf("RHS divergente")
		}
	}
	if total != 1 {
		return fmt.Errorf("cardinalidad top-level=%d", total)
	}
	return nil
}

func segundaRondaExacta(fd *ast.FuncDecl) error {
	control, err := posicionLlamada(fd, "leerControlO3bM38", "a.custodia")
	if err != nil {
		return fmt.Errorf("CONTROL: %w", err)
	}
	observador, err := posicionLlamada(fd, "autoridadSenalO3cM38", "a.custodia")
	if err != nil || observador != control+1 {
		return fmt.Errorf("CONTROL→observador: %d/%d %v", control, observador, err)
	}
	si, ok := fd.Body.List[control].(*ast.IfStmt)
	if !ok || expresion(si.Cond) != "err != nil" || len(si.Body.List) != 1 {
		return fmt.Errorf("rama CONTROL divergente")
	}
	ret, ok := si.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 2 || expresion(ret.Results[0]) != "false" || expresion(ret.Results[1]) != "err" {
		return fmt.Errorf("error CONTROL no retorna false,err")
	}
	return nil
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
		g[e.Desde] = append(g[e.Desde], e.Hacia)
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
		if c := visitar("func:" + n); c != "" {
			return c
		}
	}
	for n := range g {
		if c := visitar(n); c != "" {
			return c
		}
	}
	return ""
}
