package interna

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

const rutaPaqueteComposicionIdentidad = "vec-diputacion-granada/internal/app/composicion/interna"

func TestFachadaIdentidadOfflineAutenticarYVincularNoExponeCanalEnFronteras(t *testing.T) {
	contenido, err := os.ReadFile("identidad.go")
	if err != nil {
		t.Fatalf("leer identidad.go: %v", err)
	}
	if err := nuevoAnalizadorFronterasIdentidad(t).analizar("identidad.go", contenido); err != nil {
		t.Fatalf("identidad.go expone o difiere el canal: %v", err)
	}
}

const fuenteSoporteIdentidad = `package interna
import "crypto/tls"
type tokenServidorInterno struct{ marca byte }
type ServidorInterno struct { propietario *ServidorInterno; token *tokenServidorInterno }
type claveContextoCanalTLSInterno struct{}
type capacidadCanalTLSInterno struct{}
func (*capacidadCanalTLSInterno) consumir(*tokenServidorInterno) (tls.ConnectionState, bool) { return tls.ConnectionState{}, false }`

type marcasFronteraIdentidad uint8

const (
	marcaCanalIdentidad marcasFronteraIdentidad = 1 << iota
	marcaCapsulaIdentidad
	marcaContextoIdentidad
	marcaAbiertaIdentidad
)

type analizadorFronterasIdentidad struct {
	archivos   *token.FileSet
	importador types.Importer
}

type estadoAnalisisFronterasIdentidad struct {
	paquete      *types.Package
	informacion  *types.Info
	procedencias map[types.Object]marcasFronteraIdentidad
}

func nuevoAnalizadorFronterasIdentidad(t *testing.T) *analizadorFronterasIdentidad {
	t.Helper()
	archivos := token.NewFileSet()
	return &analizadorFronterasIdentidad{
		archivos:   archivos,
		importador: importer.ForCompiler(archivos, "gc", abrirExportIdentidad),
	}
}

func abrirExportIdentidad(ruta string) (io.ReadCloser, error) {
	orden := exec.Command("go", "list", "-export", "-f={{.Export}}", ruta)
	orden.Env = append(os.Environ(), "GOPROXY=off", "GOFLAGS=-mod=readonly")
	salida, err := orden.Output()
	if err != nil {
		return nil, fmt.Errorf("localizar export de %s: %w", ruta, err)
	}
	return os.Open(strings.TrimSpace(string(salida)))
}

func (a *analizadorFronterasIdentidad) analizar(nombre string, contenido []byte) error {
	objetivo, err := parser.ParseFile(a.archivos, nombre, contenido, 0)
	if err != nil {
		return fmt.Errorf("parsear %s: %w", nombre, err)
	}
	soporte, err := parser.ParseFile(a.archivos, "soporte.go", fuenteSoporteIdentidad, 0)
	if err != nil {
		return fmt.Errorf("parsear soporte: %w", err)
	}
	informacion := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Instances:  make(map[*ast.Ident]types.Instance),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	configuracion := types.Config{Importer: a.importador, GoVersion: "go1.26"}
	paquete, err := configuracion.Check(
		rutaPaqueteComposicionIdentidad, a.archivos, []*ast.File{objetivo, soporte}, informacion,
	)
	if err != nil {
		return fmt.Errorf("tipar %s: %w", nombre, err)
	}
	estado := &estadoAnalisisFronterasIdentidad{
		paquete: paquete, informacion: informacion,
		procedencias: calcularProcedenciasIdentidad(objetivo, paquete, informacion),
	}
	return validarFronterasIdentidad(a.archivos, objetivo, estado)
}

func marcasTipoIdentidad(tipo types.Type) marcasFronteraIdentidad {
	return marcasTipoIdentidadVisitado(tipo, map[types.Type]bool{})
}

func marcasTipoIdentidadVisitado(
	tipo types.Type,
	vistos map[types.Type]bool,
) marcasFronteraIdentidad {
	if tipo == nil {
		return 0
	}
	tipo = types.Unalias(tipo)
	if vistos[tipo] {
		return 0
	}
	vistos[tipo] = true
	if nombrado, ok := tipo.(*types.Named); ok {
		objeto := nombrado.Obj()
		if objeto.Pkg() != nil {
			switch objeto.Pkg().Path() + "." + objeto.Name() {
			case "vec-diputacion-granada/internal/vec/adapters/httpseguridad.CanalProxyAutenticado":
				return marcaCanalIdentidad
			case "vec-diputacion-granada/internal/vec/adapters/httpseguridad.CapsulaIdentidadPeticion":
				return marcaCapsulaIdentidad
			case "context.Context":
				return marcaContextoIdentidad
			}
			if objeto.Pkg().Path() != rutaPaqueteComposicionIdentidad {
				return 0
			}
		}
		return marcasTipoIdentidadVisitado(nombrado.Underlying(), vistos)
	}
	var marcas marcasFronteraIdentidad
	sumar := func(actual types.Type) {
		marcas |= marcasTipoIdentidadVisitado(actual, vistos)
	}
	switch actual := tipo.(type) {
	case *types.Pointer:
		sumar(actual.Elem())
	case *types.Array:
		sumar(actual.Elem())
	case *types.Slice:
		sumar(actual.Elem())
	case *types.Map:
		sumar(actual.Key())
		sumar(actual.Elem())
	case *types.Chan:
		sumar(actual.Elem())
	case *types.Struct:
		for indice := range actual.NumFields() {
			sumar(actual.Field(indice).Type())
		}
	case *types.Tuple:
		for indice := range actual.Len() {
			sumar(actual.At(indice).Type())
		}
	case *types.Signature:
		if actual.Recv() != nil {
			sumar(actual.Recv().Type())
		}
		sumarParametrosTipoIdentidad(actual.RecvTypeParams(), sumar)
		sumarParametrosTipoIdentidad(actual.TypeParams(), sumar)
		sumar(actual.Params())
		sumar(actual.Results())
	case *types.Interface:
		marcas |= marcaAbiertaIdentidad
		for indice := range actual.NumExplicitMethods() {
			sumar(actual.ExplicitMethod(indice).Type())
		}
		for indice := range actual.NumEmbeddeds() {
			sumar(actual.EmbeddedType(indice))
		}
	case *types.TypeParam:
		marcas |= marcaAbiertaIdentidad
		sumar(actual.Constraint())
	case *types.Union:
		for indice := range actual.Len() {
			sumar(actual.Term(indice).Type())
		}
	}
	return marcas
}

func sumarParametrosTipoIdentidad(
	parametros *types.TypeParamList,
	sumar func(types.Type),
) {
	if parametros == nil {
		return
	}
	for indice := range parametros.Len() {
		sumar(parametros.At(indice))
	}
}

func calcularProcedenciasIdentidad(
	archivo *ast.File,
	paquete *types.Package,
	info *types.Info,
) map[types.Object]marcasFronteraIdentidad {
	procedencias := make(map[types.Object]marcasFronteraIdentidad)
	for _, objeto := range info.Defs {
		if objeto != nil {
			procedencias[objeto] |= marcasTipoIdentidad(objeto.Type())
		}
	}
	estado := &estadoAnalisisFronterasIdentidad{
		paquete: paquete, informacion: info, procedencias: procedencias,
	}
	for cambiado := true; cambiado; {
		cambiado = false
		ast.Inspect(archivo, func(nodo ast.Node) bool {
			switch actual := nodo.(type) {
			case *ast.AssignStmt:
				marcas := marcasDerechaIdentidad(actual.Rhs, len(actual.Lhs), estado)
				for indice, izquierda := range actual.Lhs {
					if indice < len(marcas) {
						cambiado = propagarObjetoIdentidad(izquierda, marcas[indice], estado) || cambiado
					}
				}
			case *ast.ValueSpec:
				marcas := marcasDerechaIdentidad(actual.Values, len(actual.Names), estado)
				for indice, nombre := range actual.Names {
					if indice < len(marcas) {
						cambiado = propagarObjetoIdentidad(nombre, marcas[indice], estado) || cambiado
					}
				}
			case *ast.RangeStmt:
				marcas := marcasExpresionIdentidad(actual.X, estado)
				cambiado = propagarObjetoIdentidad(actual.Key, marcas, estado) || cambiado
				cambiado = propagarObjetoIdentidad(actual.Value, marcas, estado) || cambiado
			}
			return true
		})
	}
	return procedencias
}

func propagarObjetoIdentidad(
	expresion ast.Expr,
	marcas marcasFronteraIdentidad,
	estado *estadoAnalisisFronterasIdentidad,
) bool {
	identificador, ok := expresion.(*ast.Ident)
	if !ok || identificador.Name == "_" {
		return false
	}
	objeto := estado.informacion.ObjectOf(identificador)
	if objeto == nil {
		return false
	}
	antes := estado.procedencias[objeto]
	estado.procedencias[objeto] |= marcas
	return antes != estado.procedencias[objeto]
}

func marcasDerechaIdentidad(
	expresiones []ast.Expr,
	cantidad int,
	estado *estadoAnalisisFronterasIdentidad,
) []marcasFronteraIdentidad {
	if len(expresiones) == 1 && cantidad > 1 {
		if tupla, ok := types.Unalias(estado.informacion.TypeOf(expresiones[0])).(*types.Tuple); ok &&
			tupla.Len() == cantidad {
			resultado := make([]marcasFronteraIdentidad, cantidad)
			extra := marcasExpresionIdentidad(expresiones[0], estado) &^
				marcasTipoIdentidad(tupla)
			for indice := range cantidad {
				resultado[indice] = marcasTipoIdentidad(tupla.At(indice).Type()) | extra
			}
			return resultado
		}
	}
	resultado := make([]marcasFronteraIdentidad, len(expresiones))
	for indice, expresion := range expresiones {
		resultado[indice] = marcasExpresionIdentidad(expresion, estado)
	}
	return resultado
}

func marcasExpresionIdentidad(
	expresion ast.Expr,
	estado *estadoAnalisisFronterasIdentidad,
) marcasFronteraIdentidad {
	if expresion == nil {
		return 0
	}
	info := estado.informacion
	marcas := marcasTipoIdentidad(info.TypeOf(expresion))
	sumar := func(actual ast.Expr) {
		marcas |= marcasExpresionIdentidad(actual, estado)
	}
	switch actual := expresion.(type) {
	case *ast.Ident:
		if objeto := info.ObjectOf(actual); objeto != nil {
			marcas |= estado.procedencias[objeto]
		}
	case *ast.ParenExpr:
		sumar(actual.X)
	case *ast.UnaryExpr:
		sumar(actual.X)
	case *ast.StarExpr:
		sumar(actual.X)
	case *ast.SelectorExpr:
		sumar(actual.X)
	case *ast.IndexExpr:
		sumar(actual.X)
		sumar(actual.Index)
	case *ast.IndexListExpr:
		sumar(actual.X)
		for _, indice := range actual.Indices {
			sumar(indice)
		}
	case *ast.SliceExpr:
		sumar(actual.X)
		sumar(actual.Low)
		sumar(actual.High)
		sumar(actual.Max)
	case *ast.TypeAssertExpr:
		sumar(actual.X)
	case *ast.KeyValueExpr:
		sumar(actual.Key)
		sumar(actual.Value)
	case *ast.CompositeLit:
		for _, elemento := range actual.Elts {
			if valor, ok := elemento.(ast.Expr); ok {
				sumar(valor)
			}
		}
	case *ast.CallExpr:
		if llamadaPropagaProcedenciaIdentidad(actual, estado) {
			if receptor := receptorLlamadaIdentidad(actual, info); receptor != nil {
				sumar(receptor)
			}
			for _, argumento := range actual.Args {
				sumar(argumento)
			}
		}
	}
	return marcas
}

func llamadaPropagaProcedenciaIdentidad(
	llamada *ast.CallExpr,
	estado *estadoAnalisisFronterasIdentidad,
) bool {
	if tipo, ok := estado.informacion.Types[llamada.Fun]; ok && tipo.IsType() {
		return true
	}
	if _, ok := objetoFuncionIdentidad(llamada.Fun, estado.informacion).(*types.Builtin); ok {
		return true
	}
	return !llamadaConcretaExternaIdentidad(llamada, estado.informacion)
}

func llamadaConcretaExternaIdentidad(llamada *ast.CallExpr, info *types.Info) bool {
	funcion, ok := objetoFuncionIdentidad(llamada.Fun, info).(*types.Func)
	if !ok || funcion.Pkg() == nil || funcion.Pkg().Path() == rutaPaqueteComposicionIdentidad {
		return false
	}
	firma, ok := types.Unalias(funcion.Type()).(*types.Signature)
	if !ok || parametrosTipoPresentesIdentidad(firma) {
		return false
	}
	return firma.Recv() == nil || !tipoInterfazIdentidad(firma.Recv().Type())
}

func parametrosTipoPresentesIdentidad(firma *types.Signature) bool {
	return firma.TypeParams() != nil && firma.TypeParams().Len() != 0 ||
		firma.RecvTypeParams() != nil && firma.RecvTypeParams().Len() != 0
}

func tipoInterfazIdentidad(tipo types.Type) bool {
	tipo = types.Unalias(tipo)
	if nombrado, ok := tipo.(*types.Named); ok {
		tipo = nombrado.Underlying()
	}
	_, ok := tipo.(*types.Interface)
	return ok
}

func objetoFuncionIdentidad(expresion ast.Expr, info *types.Info) types.Object {
	switch actual := expresion.(type) {
	case *ast.ParenExpr:
		return objetoFuncionIdentidad(actual.X, info)
	case *ast.IndexExpr:
		return objetoFuncionIdentidad(actual.X, info)
	case *ast.IndexListExpr:
		return objetoFuncionIdentidad(actual.X, info)
	case *ast.Ident:
		return info.ObjectOf(actual)
	case *ast.SelectorExpr:
		if seleccion := info.Selections[actual]; seleccion != nil {
			return seleccion.Obj()
		}
		return info.ObjectOf(actual.Sel)
	default:
		return nil
	}
}

func receptorLlamadaIdentidad(llamada *ast.CallExpr, info *types.Info) ast.Expr {
	selector := selectorFuncionIdentidad(llamada.Fun)
	if selector == nil {
		return nil
	}
	seleccion := info.Selections[selector]
	if seleccion == nil || seleccion.Kind() != types.MethodVal {
		return nil
	}
	return selector.X
}

func selectorFuncionIdentidad(expresion ast.Expr) *ast.SelectorExpr {
	switch actual := expresion.(type) {
	case *ast.ParenExpr:
		return selectorFuncionIdentidad(actual.X)
	case *ast.IndexExpr:
		return selectorFuncionIdentidad(actual.X)
	case *ast.IndexListExpr:
		return selectorFuncionIdentidad(actual.X)
	case *ast.SelectorExpr:
		return actual
	default:
		return nil
	}
}

func validarFronterasIdentidad(
	archivos *token.FileSet,
	archivo *ast.File,
	estado *estadoAnalisisFronterasIdentidad,
) error {
	var infraccion error
	falla := func(nodo ast.Node, clase string) {
		if infraccion == nil {
			infraccion = fmt.Errorf("frontera: %s en %s", clase, archivos.Position(nodo.Pos()))
		}
	}
	ast.Inspect(archivo, func(nodo ast.Node) bool {
		if nodo == nil || infraccion != nil {
			return infraccion == nil
		}
		switch actual := nodo.(type) {
		case *ast.FuncDecl:
			if objeto, ok := estado.informacion.Defs[actual.Name].(*types.Func); ok &&
				marcasTipoIdentidad(objeto.Type())&marcaCanalIdentidad != 0 {
				falla(actual, "firma o callback transporta canal")
			}
		case *ast.FuncLit:
			validarClausuraIdentidad(actual, estado, falla)
		case *ast.TypeSpec:
			if marcasTipoIdentidad(estado.informacion.TypeOf(actual.Type))&marcaCanalIdentidad != 0 {
				falla(actual, "tipo o alias transporta canal")
			}
		case *ast.ValueSpec:
			validarValoresIdentidad(actual, estado, falla)
		case *ast.CompositeLit:
			if marcasExpresionIdentidad(actual, estado)&marcaCanalIdentidad != 0 {
				falla(actual, "contenedor transporta canal")
			}
		case *ast.ReturnStmt:
			for _, resultado := range actual.Results {
				if marcasExpresionIdentidad(resultado, estado)&marcaCanalIdentidad != 0 {
					falla(actual, "retorno transporta canal")
				}
			}
		case *ast.AssignStmt:
			validarAsignacionIdentidad(actual, estado, falla)
		case *ast.DeferStmt:
			if marcasCanalLlamadaIdentidad(actual.Call, estado) {
				falla(actual, "llamada diferida transporta canal")
			}
		case *ast.GoStmt:
			if marcasCanalLlamadaIdentidad(actual.Call, estado) {
				falla(actual, "goroutine transporta canal")
			}
		case *ast.CallExpr:
			validarLlamadaIdentidad(actual, estado, falla)
		}
		return infraccion == nil
	})
	return infraccion
}

func validarClausuraIdentidad(
	clausura *ast.FuncLit,
	estado *estadoAnalisisFronterasIdentidad,
	falla func(ast.Node, string),
) {
	if marcasTipoIdentidad(estado.informacion.TypeOf(clausura))&marcaCanalIdentidad != 0 {
		falla(clausura, "firma o callback transporta canal")
		return
	}
	ast.Inspect(clausura.Body, func(nodo ast.Node) bool {
		identificador, ok := nodo.(*ast.Ident)
		if !ok {
			return true
		}
		objeto := estado.informacion.Uses[identificador]
		if objeto != nil && (objeto.Pos() < clausura.Pos() || objeto.Pos() > clausura.End()) &&
			estado.procedencias[objeto]&marcaCanalIdentidad != 0 {
			falla(clausura, "clausura captura canal")
			return false
		}
		return true
	})
}

func validarValoresIdentidad(
	valores *ast.ValueSpec,
	estado *estadoAnalisisFronterasIdentidad,
	falla func(ast.Node, string),
) {
	marcas := marcasDerechaIdentidad(valores.Values, len(valores.Names), estado)
	for indice, nombre := range valores.Names {
		if nombre.Name == "_" {
			continue
		}
		objeto := estado.informacion.ObjectOf(nombre)
		if objeto == nil {
			continue
		}
		marca := marcasTipoIdentidad(objeto.Type())
		if indice < len(marcas) {
			marca |= marcas[indice]
		}
		if marca&marcaCanalIdentidad == 0 {
			continue
		}
		if objeto.Parent() == estado.paquete.Scope() {
			falla(valores, "almacenamiento de paquete transporta canal")
			return
		}
		if indice < len(marcas) && fuenteCanalInmediataIdentidad(
			valores.Values, len(valores.Names), indice, objeto.Type(), estado,
		) {
			continue
		}
		falla(valores, "declaracion local difiere canal")
		return
	}
}

func validarAsignacionIdentidad(
	asignacion *ast.AssignStmt,
	estado *estadoAnalisisFronterasIdentidad,
	falla func(ast.Node, string),
) {
	marcas := marcasDerechaIdentidad(asignacion.Rhs, len(asignacion.Lhs), estado)
	for indice, izquierda := range asignacion.Lhs {
		if indice >= len(marcas) || marcas[indice]&marcaCanalIdentidad == 0 {
			continue
		}
		identificador, esIdentificador := izquierda.(*ast.Ident)
		if esIdentificador && identificador.Name == "_" {
			continue
		}
		objeto := estado.informacion.ObjectOf(identificador)
		if esIdentificador && objeto != nil && objeto.Parent() == estado.paquete.Scope() {
			falla(asignacion, "almacenamiento de paquete transporta canal")
			return
		}
		if esIdentificador && objeto != nil && fuenteCanalInmediataIdentidad(
			asignacion.Rhs, len(asignacion.Lhs), indice, objeto.Type(), estado,
		) {
			continue
		}
		if esIdentificador && objeto != nil {
			if tipoFuncionIdentidad(objeto.Type()) {
				falla(asignacion, "callback difiere canal")
			} else if tipoCanalExactoIdentidad(objeto.Type()) {
				falla(asignacion, "alias local difiere canal")
			} else {
				falla(asignacion, "interfaz o contenedor difiere canal")
			}
			return
		}
		falla(asignacion, "almacenamiento compuesto transporta canal")
		return
	}
}

func fuenteCanalInmediataIdentidad(
	derecha []ast.Expr,
	cantidad, indice int,
	tipoDestino types.Type,
	estado *estadoAnalisisFronterasIdentidad,
) bool {
	if !tipoCanalExactoIdentidad(tipoDestino) {
		return false
	}
	var expresion ast.Expr
	indiceResultado := 0
	if len(derecha) == 1 && cantidad > 1 {
		expresion = derecha[0]
		indiceResultado = indice
	} else if indice < len(derecha) {
		expresion = derecha[indice]
	} else {
		return false
	}
	llamada, ok := expresion.(*ast.CallExpr)
	if !ok || !llamadaConcretaExternaIdentidad(llamada, estado.informacion) ||
		marcasArgumentosIdentidad(llamada, estado)&marcaCanalIdentidad != 0 {
		return false
	}
	if receptor := receptorLlamadaIdentidad(llamada, estado.informacion); receptor != nil &&
		marcasExpresionIdentidad(receptor, estado)&marcaCanalIdentidad != 0 {
		return false
	}
	tipoResultado := estado.informacion.TypeOf(llamada)
	if tupla, ok := types.Unalias(tipoResultado).(*types.Tuple); ok {
		return indiceResultado < tupla.Len() &&
			tipoCanalExactoIdentidad(tupla.At(indiceResultado).Type())
	}
	return indiceResultado == 0 && tipoCanalExactoIdentidad(tipoResultado)
}

func validarLlamadaIdentidad(
	llamada *ast.CallExpr,
	estado *estadoAnalisisFronterasIdentidad,
	falla func(ast.Node, string),
) {
	info := estado.informacion
	if tipo, ok := info.Types[llamada.Fun]; ok && tipo.IsType() {
		if marcasArgumentosIdentidad(llamada, estado)&marcaCanalIdentidad != 0 {
			falla(llamada, "conversion transporta canal")
		}
		return
	}
	concreta := llamadaConcretaExternaIdentidad(llamada, info)
	objeto := objetoFuncionIdentidad(llamada.Fun, info)
	if _, esFuncion := objeto.(*types.Func); !esFuncion &&
		marcasExpresionIdentidad(llamada.Fun, estado)&marcaCanalIdentidad != 0 {
		falla(llamada, "callback diferido transporta canal")
		return
	}
	firma, ok := types.Unalias(info.TypeOf(llamada.Fun)).(*types.Signature)
	if !ok {
		return
	}
	for indice, argumento := range llamada.Args {
		marcas := marcasExpresionIdentidad(argumento, estado)
		if marcas&marcaCanalIdentidad == 0 {
			continue
		}
		if tipoFuncionIdentidad(info.TypeOf(argumento)) {
			falla(llamada, "callback transporta canal")
			return
		}
		parametro := parametroLlamadaIdentidad(firma, indice)
		if !concreta || parametro == nil || !tipoCanalExactoIdentidad(parametro) {
			falla(llamada, "generico, interfaz o callback recibe canal")
			return
		}
	}
	if receptor := receptorLlamadaIdentidad(llamada, info); receptor != nil &&
		marcasExpresionIdentidad(receptor, estado)&marcaCanalIdentidad != 0 {
		funcion, _ := objeto.(*types.Func)
		firmaOriginal, _ := types.Unalias(funcion.Type()).(*types.Signature)
		if !concreta || firmaOriginal == nil || firmaOriginal.Recv() == nil ||
			!tipoCanalExactoIdentidad(firmaOriginal.Recv().Type()) {
			falla(llamada, "receptor o selector difiere canal")
		}
	}
}

func parametroLlamadaIdentidad(firma *types.Signature, indice int) types.Type {
	if firma.Params().Len() == 0 {
		return nil
	}
	parametro := indice
	if parametro >= firma.Params().Len() {
		if !firma.Variadic() {
			return nil
		}
		parametro = firma.Params().Len() - 1
	}
	tipo := firma.Params().At(parametro).Type()
	if firma.Variadic() && parametro == firma.Params().Len()-1 {
		if slice, ok := types.Unalias(tipo).(*types.Slice); ok {
			return slice.Elem()
		}
	}
	return tipo
}

func marcasArgumentosIdentidad(
	llamada *ast.CallExpr,
	estado *estadoAnalisisFronterasIdentidad,
) marcasFronteraIdentidad {
	var marcas marcasFronteraIdentidad
	for _, argumento := range llamada.Args {
		marcas |= marcasExpresionIdentidad(argumento, estado)
	}
	return marcas
}

func marcasCanalLlamadaIdentidad(
	llamada *ast.CallExpr,
	estado *estadoAnalisisFronterasIdentidad,
) bool {
	marcas := marcasArgumentosIdentidad(llamada, estado)
	if receptor := receptorLlamadaIdentidad(llamada, estado.informacion); receptor != nil {
		marcas |= marcasExpresionIdentidad(receptor, estado)
	}
	objeto := objetoFuncionIdentidad(llamada.Fun, estado.informacion)
	if _, esFuncion := objeto.(*types.Func); !esFuncion {
		marcas |= marcasExpresionIdentidad(llamada.Fun, estado)
	}
	return marcas&marcaCanalIdentidad != 0
}

func tipoCanalExactoIdentidad(tipo types.Type) bool {
	tipo = types.Unalias(tipo)
	nombrado, ok := tipo.(*types.Named)
	if !ok || nombrado.Obj().Pkg() == nil {
		return false
	}
	return nombrado.Obj().Pkg().Path() ==
		"vec-diputacion-granada/internal/vec/adapters/httpseguridad" &&
		nombrado.Obj().Name() == "CanalProxyAutenticado"
}

func tipoFuncionIdentidad(tipo types.Type) bool {
	tipo = types.Unalias(tipo)
	if nombrado, ok := tipo.(*types.Named); ok {
		tipo = nombrado.Underlying()
	}
	_, ok := tipo.(*types.Signature)
	return ok
}
