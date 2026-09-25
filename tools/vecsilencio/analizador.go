package main

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
	"unicode"
)

// Reglas de la guarda. VS000 no es heredable: una justificación con motivo
// fuera de la lista cerrada siempre falla.
const (
	ReglaJustificacionInvalida = "VS000"
	ReglaErrorSilencioso       = "VS001"
	ReglaRecoverSinRegistro    = "VS002"
	Regla5xxSinEmisor          = "VS003"
)

// directivaJustificacion exime la línea en la que aparece o la siguiente.
const directivaJustificacion = "//vec:silencio-justificado"

// motivosJustificacion es la lista cerrada de motivos admitidos. Añadir uno
// exige revisión de Dirección: cada motivo describe una clase de descarte
// que no puede perder información técnica.
var motivosJustificacion = map[string]struct{}{
	// Escritura en hash.Hash o fmt.State: no puede fallar.
	"HASH_INFALIBLE": {},
	// Limpieza tras un fallo ya propagado o registrado (Close, Rollback).
	"LIMPIEZA_MEJOR_ESFUERZO": {},
	// Escritura de respuesta a un cliente que puede haberse desconectado.
	"CLIENTE_DESCONECTADO": {},
	// El valor ya se validó en su constructor; el error es inalcanzable.
	"INVARIANTE_CONSTRUCTOR": {},
	// Predicado de validación: el error se traduce a false por contrato.
	"PREDICADO_VALIDACION": {},
	// Salida de una herramienta de línea de órdenes o del propio registro.
	"CLI_SALIDA": {},
}

// Hallazgo es una infracción localizada. La huella excluye la línea para
// sobrevivir a ediciones del fichero.
type Hallazgo struct {
	Fichero string
	Funcion string
	Regla   string
	Linea   int
}

// Huella devuelve fichero:función:regla.
func (h Hallazgo) Huella() string {
	return h.Fichero + ":" + h.Funcion + ":" + h.Regla
}

// estados5xx son las constantes de net/http con estado >= 500.
var estados5xx = map[string]struct{}{
	"StatusInternalServerError":           {},
	"StatusNotImplemented":                {},
	"StatusBadGateway":                    {},
	"StatusServiceUnavailable":            {},
	"StatusGatewayTimeout":                {},
	"StatusHTTPVersionNotSupported":       {},
	"StatusVariantAlsoNegotiates":         {},
	"StatusInsufficientStorage":           {},
	"StatusLoopDetected":                  {},
	"StatusNotExtended":                   {},
	"StatusNetworkAuthenticationRequired": {},
}

// AnalizarFichero aplica VS000–VS003 a un fichero ya analizado con
// comentarios. ruta es la ruta relativa con barras normales.
func AnalizarFichero(fset *token.FileSet, fichero *ast.File, ruta string) []Hallazgo {
	a := &analisis{fset: fset, ruta: ruta, justificadas: map[int]bool{}}
	a.leerDirectivas(fichero)
	for _, decl := range fichero.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Body != nil {
				a.analizarFuncion(nombreFuncion(d), d.Type, d.Body)
			}
		case *ast.GenDecl:
			ast.Inspect(d, func(n ast.Node) bool {
				if lit, ok := n.(*ast.FuncLit); ok {
					a.analizarFuncion("(global)", lit.Type, lit.Body)
					return false
				}
				return true
			})
		}
	}
	return a.hallazgos
}

type analisis struct {
	fset         *token.FileSet
	ruta         string
	justificadas map[int]bool
	hallazgos    []Hallazgo
}

func (a *analisis) leerDirectivas(fichero *ast.File) {
	for _, grupo := range fichero.Comments {
		for _, c := range grupo.List {
			if !strings.HasPrefix(c.Text, directivaJustificacion) {
				continue
			}
			linea := a.fset.Position(c.Pos()).Line
			campos := strings.Fields(strings.TrimPrefix(c.Text, directivaJustificacion))
			if len(campos) < 2 {
				a.anotar("(directiva)", ReglaJustificacionInvalida, linea)
				continue
			}
			if _, ok := motivosJustificacion[campos[0]]; !ok {
				a.anotar("(directiva)", ReglaJustificacionInvalida, linea)
				continue
			}
			a.justificadas[linea] = true
			a.justificadas[linea+1] = true
		}
	}
}

func (a *analisis) anotar(funcion, regla string, linea int) {
	a.hallazgos = append(a.hallazgos, Hallazgo{Fichero: a.ruta, Funcion: funcion, Regla: regla, Linea: linea})
}

func (a *analisis) justificada(pos token.Pos) bool {
	return a.justificadas[a.fset.Position(pos).Line]
}

// analizarFuncion recorre un cuerpo. Las funciones anónimas se analizan como
// unidades propias (para VS002/VS003) pero se atribuyen a la declaración que
// las contiene, de modo que la huella es estable.
func (a *analisis) analizarFuncion(nombre string, tipo *ast.FuncType, cuerpo *ast.BlockStmt) {
	a.revisarUnidad(nombre, cuerpo)
	resultadosNombrados := tipo != nil && tipo.Results != nil && len(tipo.Results.List) > 0 && len(tipo.Results.List[0].Names) > 0
	ast.Inspect(cuerpo, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncLit:
			a.analizarFuncion(nombre, x.Type, x.Body)
			return false
		case *ast.IfStmt:
			if a.errorSilencioso(x, resultadosNombrados) && !a.justificada(x.Pos()) {
				a.anotar(nombre, ReglaErrorSilencioso, a.fset.Position(x.Pos()).Line)
			}
		}
		return true
	})
}

// revisarUnidad aplica VS002 y VS003 a un cuerpo sin descender a funciones
// anónimas, que forman su propia unidad.
func (a *analisis) revisarUnidad(nombre string, cuerpo *ast.BlockStmt) {
	registra := contieneLlamada(cuerpo, esLlamadaRegistro) || contieneLlamada(cuerpo, esPanico)
	inspeccionarSinAnidadas(cuerpo, func(n ast.Node) {
		llamada, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}
		if id, ok := llamada.Fun.(*ast.Ident); ok && id.Name == "recover" && len(llamada.Args) == 0 {
			if !registra && !a.justificada(llamada.Pos()) {
				a.anotar(nombre, ReglaRecoverSinRegistro, a.fset.Position(llamada.Pos()).Line)
			}
		}
		if !registra && escribeEstado5xx(llamada) && !a.justificada(llamada.Pos()) {
			a.anotar(nombre, Regla5xxSinEmisor, a.fset.Position(llamada.Pos()).Line)
		}
	})
}

// errorSilencioso reconoce `if err != nil { ... }` cuyo bloque no usa el
// error, no registra, no responde al cliente, no relanza y termina en un
// retorno sin error (nil, false, literales, vacío) o en continue/break.
func (a *analisis) errorSilencioso(si *ast.IfStmt, resultadosNombrados bool) bool {
	nombre, ok := variableErrorComparadaConNil(si.Cond)
	if !ok {
		return false
	}
	cuerpo := si.Body
	if referencia(cuerpo, nombre) || contieneLlamada(cuerpo, esLlamadaRegistro) ||
		contieneLlamada(cuerpo, esPanico) || contieneLlamada(cuerpo, esRespuestaCliente) {
		return false
	}
	if len(cuerpo.List) == 0 {
		return true
	}
	switch fin := cuerpo.List[len(cuerpo.List)-1].(type) {
	case *ast.BranchStmt:
		return fin.Tok == token.CONTINUE || fin.Tok == token.BREAK
	case *ast.ReturnStmt:
		if len(fin.Results) == 0 {
			return !resultadosNombrados
		}
		for _, r := range fin.Results {
			if !resultadoInocuo(r) {
				return false
			}
		}
		return true
	}
	return false
}

func variableErrorComparadaConNil(cond ast.Expr) (string, bool) {
	bin, ok := cond.(*ast.BinaryExpr)
	if !ok || bin.Op != token.NEQ {
		return "", false
	}
	x, y := bin.X, bin.Y
	if esNil(x) {
		x, y = y, x
	}
	id, ok := x.(*ast.Ident)
	if !ok || !esNil(y) || !nombreDeError(id.Name) {
		return "", false
	}
	return id.Name, true
}

func esNil(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "nil"
}

// nombreDeError reconoce err, errAlgo, algoErr y ErrAlgo.
func nombreDeError(n string) bool {
	switch {
	case n == "err" || n == "Err":
		return true
	case (strings.HasPrefix(n, "err") || strings.HasPrefix(n, "Err")) && len(n) > 3:
		return unicode.IsUpper(rune(n[3])) || n[3] == '_'
	case strings.HasSuffix(n, "Err") || strings.HasSuffix(n, "Error"):
		return true
	}
	return false
}

// resultadoInocuo es cierto para valores que no pueden transportar un error.
// Una llamada no lo es: puede construir o propagar uno.
func resultadoInocuo(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.Ident:
		return !nombreDeError(x.Name)
	case *ast.BasicLit:
		return true
	case *ast.CompositeLit:
		return len(x.Elts) == 0
	case *ast.UnaryExpr:
		return x.Op == token.AND && resultadoInocuo(x.X)
	case *ast.SelectorExpr:
		return !nombreDeError(x.Sel.Name)
	case *ast.ParenExpr:
		return resultadoInocuo(x.X)
	}
	return false
}

func referencia(n ast.Node, nombre string) bool {
	encontrado := false
	ast.Inspect(n, func(m ast.Node) bool {
		if id, ok := m.(*ast.Ident); ok && id.Name == nombre {
			encontrado = true
		}
		return !encontrado
	})
	return encontrado
}

func contieneLlamada(n ast.Node, criterio func(*ast.CallExpr) bool) bool {
	encontrado := false
	inspeccionarSinAnidadas(n, func(m ast.Node) {
		if llamada, ok := m.(*ast.CallExpr); ok && criterio(llamada) {
			encontrado = true
		}
	})
	return encontrado
}

// inspeccionarSinAnidadas visita n sin entrar en funciones anónimas.
func inspeccionarSinAnidadas(n ast.Node, visitar func(ast.Node)) {
	ast.Inspect(n, func(m ast.Node) bool {
		if m == nil {
			return false
		}
		if _, ok := m.(*ast.FuncLit); ok && m != n {
			return false
		}
		visitar(m)
		return true
	})
}

func nombreLlamada(llamada *ast.CallExpr) (receptor, nombre string) {
	switch f := llamada.Fun.(type) {
	case *ast.Ident:
		return "", f.Name
	case *ast.SelectorExpr:
		if id, ok := f.X.(*ast.Ident); ok {
			return id.Name, f.Sel.Name
		}
		return "", f.Sel.Name
	}
	return "", ""
}

var metodosRegistro = map[string]struct{}{
	"Emitir": {}, "EmitirIncidenciaTecnicaDesdeContexto": {},
	"Print": {}, "Printf": {}, "Println": {}, "Fatal": {}, "Fatalf": {}, "Fatalln": {},
	"Panic": {}, "Panicf": {}, "Error": {}, "ErrorContext": {}, "Warn": {}, "WarnContext": {},
	"Info": {}, "InfoContext": {}, "Log": {}, "LogAttrs": {},
}

var prefijosRegistro = []string{
	"registrarFallo", "registrarIncidencia", "emitirIncidencia", "recuperarYEmitir", "RegistrarFallo",
}

func esLlamadaRegistro(llamada *ast.CallExpr) bool {
	receptor, nombre := nombreLlamada(llamada)
	if receptor == "log" || receptor == "slog" {
		return true
	}
	if _, ok := metodosRegistro[nombre]; ok {
		return true
	}
	for _, p := range prefijosRegistro {
		if strings.HasPrefix(nombre, p) {
			return true
		}
	}
	return false
}

func esPanico(llamada *ast.CallExpr) bool {
	id, ok := llamada.Fun.(*ast.Ident)
	return ok && id.Name == "panic"
}

// esRespuestaCliente reconoce la traducción del fallo a una respuesta HTTP:
// el cliente recibe el resultado y los 5xx los declara el middleware común.
func esRespuestaCliente(llamada *ast.CallExpr) bool {
	receptor, nombre := nombreLlamada(llamada)
	if receptor == "http" && (nombre == "Error" || nombre == "NotFound" || nombre == "Redirect") {
		return true
	}
	minus := strings.ToLower(nombre)
	return nombre == "WriteHeader" || strings.HasPrefix(minus, "write") ||
		strings.HasPrefix(minus, "escribir") || strings.HasPrefix(minus, "responder")
}

// escribeEstado5xx detecta una llamada que recibe un estado >= 500 como
// argumento (constante de net/http o literal).
func escribeEstado5xx(llamada *ast.CallExpr) bool {
	for _, arg := range llamada.Args {
		switch x := arg.(type) {
		case *ast.SelectorExpr:
			if id, ok := x.X.(*ast.Ident); ok && id.Name == "http" {
				if _, ok := estados5xx[x.Sel.Name]; ok {
					return true
				}
			}
		case *ast.BasicLit:
			if x.Kind == token.INT && esRespuestaCliente(llamada) {
				if v, err := strconv.Atoi(x.Value); err == nil && v >= 500 && v <= 599 {
					return true
				}
			}
		}
	}
	return false
}

func nombreFuncion(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return d.Name.Name
	}
	tipo := d.Recv.List[0].Type
	for {
		switch t := tipo.(type) {
		case *ast.StarExpr:
			tipo = t.X
			continue
		case *ast.IndexExpr:
			tipo = t.X
			continue
		case *ast.IndexListExpr:
			tipo = t.X
			continue
		case *ast.Ident:
			return t.Name + "." + d.Name.Name
		}
		return d.Name.Name
	}
}
