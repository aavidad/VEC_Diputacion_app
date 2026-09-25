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
	// ReglaJustificacion no es una infracción: cuenta las directivas válidas
	// por motivo, que como la línea base solo pueden decrecer.
	ReglaJustificacion = "VSJ"
)

// ficheroJustificaciones agrupa las directivas en una huella por motivo,
// independiente del fichero: mover una justificación no altera el recuento.
const ficheroJustificaciones = "(justificadas)"

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

// Huella devuelve fichero:función:regla. Las justificaciones se agrupan
// por motivo: (justificadas):MOTIVO:VSJ.
func (h Hallazgo) Huella() string {
	if h.Regla == ReglaJustificacion {
		return ficheroJustificaciones + ":" + h.Funcion + ":" + h.Regla
	}
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

// estados2xx son las constantes de net/http de éxito: una respuesta con
// ellas no traduce el fallo al cliente.
var estados2xx = map[string]struct{}{
	"StatusOK": {}, "StatusCreated": {}, "StatusAccepted": {}, "StatusNonAuthoritativeInfo": {},
	"StatusNoContent": {}, "StatusResetContent": {}, "StatusPartialContent": {},
	"StatusMultiStatus": {}, "StatusAlreadyReported": {}, "StatusIMUsed": {},
}

// EsPaqueteDePruebas reconoce los paquetes de apoyo exclusivo a pruebas por
// su nombre (pruebas, *prueba, *pruebas): la composición real no puede
// importarlos y no forman parte de la superficie auditada.
func EsPaqueteDePruebas(nombre string) bool {
	return nombre == "pruebas" || strings.HasSuffix(nombre, "prueba") || strings.HasSuffix(nombre, "pruebas")
}

// AnalizarFichero aplica VS000–VS003 a un fichero ya analizado con
// comentarios y cuenta sus justificaciones (VSJ). ruta es la ruta relativa
// con barras normales.
func AnalizarFichero(fset *token.FileSet, fichero *ast.File, ruta string) []Hallazgo {
	a := &analisis{fset: fset, ruta: ruta, justificadas: map[int]bool{}}
	a.leerDirectivas(fichero)
	for _, decl := range fichero.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Body != nil {
				a.analizarFuncion(nombreFuncion(d), nuevoAmbito(nil, d.Recv, d.Type), d.Type, d.Body)
			}
		case *ast.GenDecl:
			ast.Inspect(d, func(n ast.Node) bool {
				if lit, ok := n.(*ast.FuncLit); ok {
					a.analizarFuncion("(global)", nuevoAmbito(nil, lit.Type), lit.Type, lit.Body)
					return false
				}
				return true
			})
		}
	}
	return a.hallazgos
}

// ambito reúne, sin información de tipos, los nombres de una función (y de
// las que la contienen) declarados como http.ResponseWriter o como error.
// Sin go/types no se conoce el tipo de `a, b := f()`: esas variables solo se
// reconocen como error por su nombre (err, errX, xErr, xError).
type ambito struct {
	escritores map[string]bool
	errores    map[string]bool
}

func nuevoAmbito(padre *ambito, listas ...any) *ambito {
	a := &ambito{escritores: map[string]bool{}, errores: map[string]bool{}}
	if padre != nil {
		for n := range padre.escritores {
			a.escritores[n] = true
		}
		for n := range padre.errores {
			a.errores[n] = true
		}
	}
	for _, l := range listas {
		switch x := l.(type) {
		case *ast.FieldList:
			a.anotarCampos(x)
		case *ast.FuncType:
			if x != nil {
				a.anotarCampos(x.Params)
				a.anotarCampos(x.Results)
			}
		}
	}
	return a
}

func (a *ambito) anotarCampos(campos *ast.FieldList) {
	if campos == nil {
		return
	}
	for _, c := range campos.List {
		for _, n := range c.Names {
			a.anotarNombre(n.Name, c.Type)
		}
	}
}

func (a *ambito) anotarNombre(nombre string, tipo ast.Expr) {
	switch {
	case esTipoEscritorHTTP(tipo):
		a.escritores[nombre] = true
	case esTipoError(tipo):
		a.errores[nombre] = true
	}
}

// anotarDeclaraciones añade las `var x error` y `var w http.ResponseWriter`
// del cuerpo, sin descender a funciones anónimas.
func (a *ambito) anotarDeclaraciones(cuerpo *ast.BlockStmt) {
	inspeccionarSinAnidadas(cuerpo, func(n ast.Node) {
		if v, ok := n.(*ast.ValueSpec); ok && v.Type != nil {
			for _, nombre := range v.Names {
				a.anotarNombre(nombre.Name, v.Type)
			}
		}
	})
}

func esTipoEscritorHTTP(tipo ast.Expr) bool {
	sel, ok := tipo.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == "http" && sel.Sel.Name == "ResponseWriter"
}

func esTipoError(tipo ast.Expr) bool {
	id, ok := tipo.(*ast.Ident)
	return ok && id.Name == "error"
}

func (a *ambito) esError(nombre string) bool {
	return nombreDeError(nombre) || a.errores[nombre]
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
			a.anotar(campos[0], ReglaJustificacion, linea)
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
func (a *analisis) analizarFuncion(nombre string, amb *ambito, tipo *ast.FuncType, cuerpo *ast.BlockStmt) {
	amb.anotarDeclaraciones(cuerpo)
	a.revisarUnidad(nombre, cuerpo)
	resultadosNombrados := tipo != nil && tipo.Results != nil && len(tipo.Results.List) > 0 && len(tipo.Results.List[0].Names) > 0
	ast.Inspect(cuerpo, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncLit:
			a.analizarFuncion(nombre, nuevoAmbito(amb, x.Type), x.Type, x.Body)
			return false
		case *ast.IfStmt:
			if a.errorSilencioso(x, amb, resultadosNombrados) && !a.justificada(x.Pos()) {
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

// errorSilencioso reconoce `if err != nil { ... }` (también como término de
// una disyunción `err != nil || …`) cuyo bloque no usa el error (`_ = err` no
// cuenta), no registra, no responde al cliente con un estado de fallo, no
// relanza y termina en un retorno sin error (nil, false, literales, vacío) o
// en continue/break.
func (a *analisis) errorSilencioso(si *ast.IfStmt, amb *ambito, resultadosNombrados bool) bool {
	nombre, ok := variableErrorComparadaConNil(si.Cond, amb)
	if !ok {
		return false
	}
	cuerpo := si.Body
	respondeCliente := func(llamada *ast.CallExpr) bool { return esRespuestaCliente(llamada, amb) }
	if usaVariable(cuerpo, nombre) || contieneLlamada(cuerpo, esLlamadaRegistro) ||
		contieneLlamada(cuerpo, esPanico) || contieneLlamada(cuerpo, respondeCliente) {
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

// variableErrorComparadaConNil devuelve la variable de error de `err != nil`
// o de cualquier término de una disyunción que la contenga.
func variableErrorComparadaConNil(cond ast.Expr, amb *ambito) (string, bool) {
	if p, ok := cond.(*ast.ParenExpr); ok {
		return variableErrorComparadaConNil(p.X, amb)
	}
	bin, ok := cond.(*ast.BinaryExpr)
	if !ok {
		return "", false
	}
	if bin.Op == token.LOR {
		if nombre, ok := variableErrorComparadaConNil(bin.X, amb); ok {
			return nombre, true
		}
		return variableErrorComparadaConNil(bin.Y, amb)
	}
	if bin.Op != token.NEQ {
		return "", false
	}
	x, y := bin.X, bin.Y
	if esNil(x) {
		x, y = y, x
	}
	id, ok := x.(*ast.Ident)
	if !ok || !esNil(y) || !amb.esError(id.Name) {
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

// esMotivoCerrado reconoce un motivo o error cerrado con nombre (Motivo*,
// Err*): devolverlo propaga el fallo al llamante.
func esMotivoCerrado(nombre string) bool {
	return nombreDeError(nombre) || (strings.HasPrefix(nombre, "Motivo") && len(nombre) > len("Motivo"))
}

// resultadoInocuo es cierto para valores que no pueden transportar un error.
// Una llamada no lo es: puede construir o propagar uno.
func resultadoInocuo(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.Ident:
		return !esMotivoCerrado(x.Name)
	case *ast.BasicLit:
		return true
	case *ast.CompositeLit:
		return len(x.Elts) == 0
	case *ast.UnaryExpr:
		return x.Op == token.AND && resultadoInocuo(x.X)
	case *ast.SelectorExpr:
		return !esMotivoCerrado(x.Sel.Name)
	case *ast.ParenExpr:
		return resultadoInocuo(x.X)
	}
	return false
}

// usaVariable indica si n usa nombre. Un descarte explícito `_ = err` no es
// atender el error y no cuenta.
func usaVariable(n ast.Node, nombre string) bool {
	encontrado := false
	ast.Inspect(n, func(m ast.Node) bool {
		if encontrado {
			return false
		}
		if asignacion, ok := m.(*ast.AssignStmt); ok && soloDescarta(asignacion) {
			return false
		}
		if id, ok := m.(*ast.Ident); ok && id.Name == nombre {
			encontrado = true
		}
		return !encontrado
	})
	return encontrado
}

// soloDescarta reconoce `_ = x` (o `_, _ = x, y`) con identificadores
// desnudos a la derecha.
func soloDescarta(asignacion *ast.AssignStmt) bool {
	if asignacion.Tok != token.ASSIGN {
		return false
	}
	for _, izquierda := range asignacion.Lhs {
		if id, ok := izquierda.(*ast.Ident); !ok || id.Name != "_" {
			return false
		}
	}
	for _, derecha := range asignacion.Rhs {
		if _, ok := derecha.(*ast.Ident); !ok {
			return false
		}
	}
	return true
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
	"Emitir": {}, "EmitirIncidenciaTecnicaEnPeticion": {},
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
// Cuenta toda llamada que recibe un http.ResponseWriter del ámbito (p. ej.
// m.denegar(w, …)) y las de nombre http.Error/NotFound/Redirect,
// WriteHeader, write*, escribir* o responder*; ninguna libra si el estado que
// recibe es, de forma determinable, 2xx.
func esRespuestaCliente(llamada *ast.CallExpr, amb *ambito) bool {
	if recibeEstado2xx(llamada) {
		return false
	}
	receptor, nombre := nombreLlamada(llamada)
	if receptor == "http" && (nombre == "Error" || nombre == "NotFound" || nombre == "Redirect") {
		return true
	}
	minus := strings.ToLower(nombre)
	if nombre == "WriteHeader" || strings.HasPrefix(minus, "write") ||
		strings.HasPrefix(minus, "escribir") || strings.HasPrefix(minus, "responder") {
		return true
	}
	if amb == nil {
		return false
	}
	for _, arg := range llamada.Args {
		if id, ok := arg.(*ast.Ident); ok && amb.escritores[id.Name] {
			return true
		}
	}
	return false
}

// recibeEstado2xx detecta un argumento que es un estado 2xx (constante de
// net/http o literal).
func recibeEstado2xx(llamada *ast.CallExpr) bool {
	for _, arg := range llamada.Args {
		switch x := arg.(type) {
		case *ast.SelectorExpr:
			if id, ok := x.X.(*ast.Ident); ok && id.Name == "http" {
				if _, ok := estados2xx[x.Sel.Name]; ok {
					return true
				}
			}
		case *ast.BasicLit:
			if v, err := strconv.Atoi(x.Value); x.Kind == token.INT && err == nil && v >= 200 && v <= 299 {
				return true
			}
		}
	}
	return false
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
			if x.Kind == token.INT && esRespuestaCliente(llamada, nil) {
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
