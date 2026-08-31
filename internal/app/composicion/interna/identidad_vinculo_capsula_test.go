package interna

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

func TestFachadaIdentidadOfflineAutenticarYVincularNoExponeCanalEnFronteras(t *testing.T) {
	contenido, err := os.ReadFile("identidad.go")
	if err != nil {
		t.Fatalf("leer identidad.go: %v", err)
	}
	if err := nuevoAnalizadorFronterasIdentidad(t).analizar("identidad.go", contenido); err != nil {
		t.Fatalf("identidad.go expone o difiere el canal: %v", err)
	}
}

func TestFachadaIdentidadOfflineAutenticarYVincularNoExponeCanalEnFronterasMutantes(t *testing.T) {
	analizador := nuevoAnalizadorFronterasIdentidad(t)
	mutantes := map[string][2]string{
		"helper con retorno directo": {`func mutante(c httpseguridad.CanalProxyAutenticado) httpseguridad.CanalProxyAutenticado { return c }`, "firma"},
		"alias de tipo":              {`type canalAliasMutante = httpseguridad.CanalProxyAutenticado`, "tipo"},
		"callback":                   {`func mutante(_ func(httpseguridad.CanalProxyAutenticado, context.Context)) {}`, "firma"},
		"clausura por captura": {`func mutante(d soporteMutante) func() {
	canal, _ := d.servicio.AutenticarCanalTLSMutuo(d.estado); var identidad httpseguridad.IdentidadSesion
	capsula, _ := d.servicio.ProyectarCapsulaIdentidadPeticion(d.contexto, identidad, canal)
	return func() { _, _ = canal, capsula }
}`, "clausura"},
		"struct anonimo": {`func mutante(d soporteMutante) any {
	canal, _ := d.servicio.AutenticarCanalTLSMutuo(d.estado); var capsula httpseguridad.CapsulaIdentidadPeticion
	return struct { canal httpseguridad.CanalProxyAutenticado; capsula httpseguridad.CapsulaIdentidadPeticion }{canal, capsula}
}`, "retorno"},
		"slice any": {`func mutante(d soporteMutante) {
	canal, _ := d.servicio.AutenticarCanalTLSMutuo(d.estado); var capsula httpseguridad.CapsulaIdentidadPeticion
	diferido := []any{canal, capsula}; _ = diferido
}`, "contenedor"},
		"retorno any": {`func mutante(d soporteMutante) any {
	canal, _ := d.servicio.AutenticarCanalTLSMutuo(d.estado); return canal
}`, "retorno"},
		"almacenamiento de paquete": {`var almacenMutante any
func mutante(d soporteMutante) { canal, _ := d.servicio.AutenticarCanalTLSMutuo(d.estado); almacenMutante = canal }`, "paquete"},
	}
	for nombre, mutante := range mutantes {
		t.Run(nombre, func(t *testing.T) {
			err := analizador.analizar("mutante.go", []byte(cabeceraMutantesIdentidad+mutante[0]))
			if err == nil || !strings.Contains(err.Error(), mutante[1]) {
				t.Fatalf("mutante no rechazado por %q: %v", mutante[1], err)
			}
		})
	}
	if err := analizador.analizar("homonimo.go", []byte(cabeceraMutantesIdentidad)); err != nil {
		t.Fatalf("selector homonimo inocuo produjo falso positivo: %v", err)
	}
}

const fuenteSoporteIdentidad = `package interna
import "crypto/tls"
type tokenServidorInterno struct{ marca byte }
type ServidorInterno struct { propietario *ServidorInterno; token *tokenServidorInterno }
type claveContextoCanalTLSInterno struct{}
type capacidadCanalTLSInterno struct{}
func (*capacidadCanalTLSInterno) consumir(*tokenServidorInterno) (tls.ConnectionState, bool) { return tls.ConnectionState{}, false }`

const cabeceraMutantesIdentidad = `package interna
import ("context"; "crypto/tls"; "vec-diputacion-granada/internal/vec/adapters/httpseguridad")
type soporteMutante struct { contexto context.Context; estado tls.ConnectionState; servicio *httpseguridad.ServicioIdentidad }
type tipoHomonimoMutante struct{ CanalProxyAutenticado, CapsulaIdentidadPeticion int }
func selectorHomonimoMutante(v tipoHomonimoMutante) int { return v.CanalProxyAutenticado + v.CapsulaIdentidadPeticion }
`

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

func nuevoAnalizadorFronterasIdentidad(t *testing.T) *analizadorFronterasIdentidad {
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
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	configuracion := types.Config{Importer: a.importador, GoVersion: "go1.26"}
	paquete, err := configuracion.Check("vec-diputacion-granada/internal/app/composicion/interna", a.archivos, []*ast.File{objetivo, soporte}, informacion)
	if err != nil {
		return fmt.Errorf("tipar %s: %w", nombre, err)
	}
	return validarFronterasIdentidad(a.archivos, objetivo, paquete, informacion)
}

func marcasTipoIdentidad(tipo types.Type) marcasFronteraIdentidad {
	return marcasTipoIdentidadVisitado(tipo, map[types.Type]bool{})
}

func marcasTipoIdentidadVisitado(tipo types.Type, vistos map[types.Type]bool) marcasFronteraIdentidad {
	if tipo == nil || vistos[tipo] {
		return 0
	}
	vistos[tipo] = true
	tipo = types.Unalias(tipo)
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
			if objeto.Pkg().Path() != "vec-diputacion-granada/internal/app/composicion/interna" {
				return 0
			}
		}
		return marcasTipoIdentidadVisitado(nombrado.Underlying(), vistos)
	}
	var marcas marcasFronteraIdentidad
	sumar := func(t types.Type) { marcas |= marcasTipoIdentidadVisitado(t, vistos) }
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
		sumar(actual.Params())
		sumar(actual.Results())
	case *types.Interface:
		marcas |= marcaAbiertaIdentidad
		for indice := range actual.NumExplicitMethods() {
			sumar(actual.ExplicitMethod(indice).Type())
		}
	}
	return marcas
}

func marcasDerechaIdentidad(expresiones []ast.Expr, cantidad int, info *types.Info) []marcasFronteraIdentidad {
	if len(expresiones) == 1 && cantidad > 1 {
		if tupla, ok := info.TypeOf(expresiones[0]).(*types.Tuple); ok && tupla.Len() == cantidad {
			resultado := make([]marcasFronteraIdentidad, cantidad)
			for indice := range cantidad {
				resultado[indice] = marcasTipoIdentidad(tupla.At(indice).Type())
			}
			return resultado
		}
	}
	resultado := make([]marcasFronteraIdentidad, len(expresiones))
	for indice, expresion := range expresiones {
		resultado[indice] = marcasExpresionIdentidad(expresion, info)
	}
	return resultado
}

func marcasExpresionIdentidad(expresion ast.Expr, info *types.Info) marcasFronteraIdentidad {
	if expresion == nil {
		return 0
	}
	marcas := marcasTipoIdentidad(info.TypeOf(expresion))
	switch actual := expresion.(type) {
	case *ast.ParenExpr:
		marcas |= marcasExpresionIdentidad(actual.X, info)
	case *ast.UnaryExpr:
		marcas |= marcasExpresionIdentidad(actual.X, info)
	case *ast.CompositeLit:
		for _, elemento := range actual.Elts {
			if par, ok := elemento.(*ast.KeyValueExpr); ok {
				marcas |= marcasExpresionIdentidad(par.Key, info)
				marcas |= marcasExpresionIdentidad(par.Value, info)
			} else if valor, ok := elemento.(ast.Expr); ok {
				marcas |= marcasExpresionIdentidad(valor, info)
			}
		}
	case *ast.CallExpr:
		tipo, conversion := info.Types[actual.Fun]
		identificador, appendBuiltin := actual.Fun.(*ast.Ident)
		if conversion && tipo.IsType() || appendBuiltin && identificador.Name == "append" {
			for _, argumento := range actual.Args {
				marcas |= marcasExpresionIdentidad(argumento, info)
			}
		}
	}
	return marcas
}

func validarFronterasIdentidad(archivos *token.FileSet, archivo *ast.File, paquete *types.Package, info *types.Info) error {
	var infraccion error
	falla := func(nodo ast.Node, clase string) {
		if infraccion == nil {
			infraccion = fmt.Errorf("%s en %s", clase, archivos.Position(nodo.Pos()))
		}
	}
	ast.Inspect(archivo, func(nodo ast.Node) bool {
		if nodo == nil || infraccion != nil {
			return infraccion == nil
		}
		switch actual := nodo.(type) {
		case *ast.FuncDecl:
			if objeto, ok := info.Defs[actual.Name].(*types.Func); ok && marcasTipoIdentidad(objeto.Type())&marcaCanalIdentidad != 0 {
				falla(actual, "firma o callback transporta canal")
			}
		case *ast.FuncLit:
			if marcasTipoIdentidad(info.TypeOf(actual))&marcaCanalIdentidad != 0 {
				falla(actual, "firma o callback transporta canal")
				break
			}
			ast.Inspect(actual.Body, func(interior ast.Node) bool {
				identificador, ok := interior.(*ast.Ident)
				if !ok {
					return true
				}
				objeto := info.Uses[identificador]
				if objeto != nil && (objeto.Pos() < actual.Pos() || objeto.Pos() > actual.End()) &&
					marcasTipoIdentidad(objeto.Type())&marcaCanalIdentidad != 0 {
					falla(actual, "clausura captura canal")
					return false
				}
				return true
			})
		case *ast.TypeSpec:
			if marcasTipoIdentidad(info.TypeOf(actual.Type))&marcaCanalIdentidad != 0 {
				falla(actual, "tipo o alias transporta canal")
			}
		case *ast.ValueSpec:
			for indice, nombre := range actual.Names {
				objeto := info.ObjectOf(nombre)
				if objeto == nil {
					continue
				}
				marcas := marcasTipoIdentidad(objeto.Type())
				if indice < len(actual.Values) {
					marcas |= marcasExpresionIdentidad(actual.Values[indice], info)
				}
				if marcas&marcaCanalIdentidad != 0 && objeto.Parent() == paquete.Scope() {
					falla(actual, "almacenamiento de paquete transporta canal")
				}
			}
		case *ast.CompositeLit:
			if marcasTipoIdentidad(info.TypeOf(actual.Type))&marcaCanalIdentidad != 0 {
				falla(actual, "tipo anonimo transporta canal")
			} else if marcas := marcasExpresionIdentidad(actual, info); marcas&marcaCanalIdentidad != 0 && marcas&(marcaCapsulaIdentidad|marcaContextoIdentidad|marcaAbiertaIdentidad) != 0 {
				falla(actual, "contenedor transporta canal")
			}
		case *ast.ReturnStmt:
			for _, resultado := range actual.Results {
				if marcasExpresionIdentidad(resultado, info)&marcaCanalIdentidad != 0 {
					falla(actual, "retorno transporta canal")
				}
			}
		case *ast.AssignStmt:
			marcas := marcasDerechaIdentidad(actual.Rhs, len(actual.Lhs), info)
			for indice, izquierda := range actual.Lhs {
				if indice >= len(marcas) || marcas[indice]&marcaCanalIdentidad == 0 {
					continue
				}
				identificador, esIdentificador := izquierda.(*ast.Ident)
				objeto := info.ObjectOf(identificador)
				if esIdentificador && objeto != nil && objeto.Parent() == paquete.Scope() {
					falla(actual, "almacenamiento de paquete transporta canal")
				} else if marcasTipoIdentidad(info.TypeOf(izquierda))&marcaAbiertaIdentidad != 0 {
					falla(actual, "interfaz o contenedor transporta canal")
				}
			}
		case *ast.CallExpr:
			validarLlamadaIdentidad(actual, info, falla)
		}
		return infraccion == nil
	})
	return infraccion
}

func validarLlamadaIdentidad(llamada *ast.CallExpr, info *types.Info, falla func(ast.Node, string)) {
	if tipo, ok := info.Types[llamada.Fun]; ok && tipo.IsType() {
		if marcasTipoIdentidad(tipo.Type)&marcaAbiertaIdentidad != 0 && marcasArgumentosIdentidad(llamada, info)&marcaCanalIdentidad != 0 {
			falla(llamada, "interfaz transporta canal")
		}
		return
	}
	firma, ok := types.Unalias(info.TypeOf(llamada.Fun)).(*types.Signature)
	if !ok {
		return
	}
	for indice, argumento := range llamada.Args {
		if marcasExpresionIdentidad(argumento, info)&marcaCanalIdentidad == 0 || firma.Params().Len() == 0 {
			continue
		}
		parametro := min(indice, firma.Params().Len()-1)
		tipoParametro := firma.Params().At(parametro).Type()
		if firma.Variadic() && parametro == firma.Params().Len()-1 {
			tipoParametro = tipoParametro.(*types.Slice).Elem()
		}
		if marcasTipoIdentidad(tipoParametro)&marcaCanalIdentidad == 0 {
			falla(llamada, "interfaz o callback recibe canal")
			return
		}
	}
}

func marcasArgumentosIdentidad(llamada *ast.CallExpr, info *types.Info) marcasFronteraIdentidad {
	var marcas marcasFronteraIdentidad
	for _, argumento := range llamada.Args {
		marcas |= marcasExpresionIdentidad(argumento, info)
	}
	return marcas
}

func TestFachadaIdentidadOfflineAutenticarYVincularExitoYConservaContexto(t *testing.T) {
	entorno := nuevoEntornoIdentidadOfflinePrueba(t)
	type claveContextoPrueba struct{}
	valor := &struct{ marca string }{marca: "valor-sintetico"}
	limite := time.Now().Add(time.Minute)
	var vinculado context.Context
	var cancelar context.CancelFunc
	var err error

	codigo := entorno.ejecutarEnC4(t, func(ctx context.Context) {
		ctx = context.WithValue(ctx, claveContextoPrueba{}, valor)
		ctx, cancelar = context.WithDeadline(ctx, limite)
		vinculado, err = entorno.fachada.AutenticarYVincular(
			ctx, []byte("asercion-corporativa-protegida"),
		)
	})
	if codigo != http.StatusNoContent || err != nil || vinculado == nil {
		t.Fatalf("autenticar y vincular: codigo=%d contexto_nulo=%t error=%v", codigo, vinculado == nil, err)
	}
	if vinculado.Value(claveContextoPrueba{}) != valor {
		t.Fatal("el contexto derivado no conserva los valores del contexto de peticion")
	}
	limiteObtenido, tieneLimite := vinculado.Deadline()
	if !tieneLimite || !limiteObtenido.Equal(limite) {
		t.Fatalf("deadline alterado: presente=%t", tieneLimite)
	}
	if efectos := entorno.efectos(); efectos != (efectosIdentidadOfflinePrueba{1, 3, 1, 2}) {
		t.Fatalf("secuencia de autenticacion y vinculo incompleta: %+v", efectos)
	}
	cuenta, auditoria, err := entorno.servicio.ExtraerCapsulaIdentidadPeticion(vinculado)
	if err != nil || cuenta.Validar() != nil ||
		auditoria.CanalVinculadoRef() != entorno.canal.ReferenciaVinculacion() {
		t.Fatalf("extraccion por la instancia emisora rechazada: %v", err)
	}
	otro := nuevoEntornoIdentidadOfflinePrueba(t)
	cuentaCruzada, auditoriaCruzada, err := otro.servicio.ExtraerCapsulaIdentidadPeticion(vinculado)
	if !errors.Is(err, httpseguridad.ErrSesionNoValida) ||
		cuentaCruzada.Validar() == nil || auditoriaCruzada.CanalVinculadoRef() != "" {
		t.Fatalf("extraccion por servicio cruzado admitida: %v", err)
	}

	cancelar()
	<-vinculado.Done()
	if !errors.Is(vinculado.Err(), context.Canceled) {
		t.Fatalf("cancelacion del padre no conservada: %v", vinculado.Err())
	}
}

func TestFachadaIdentidadOfflineAutenticarYVincularFallaCerrado(t *testing.T) {
	t.Run("fachada nula", func(t *testing.T) {
		var fachada *FachadaIdentidadOffline
		ctx, err := fachada.AutenticarYVincular(context.Background(), []byte("x"))
		exigirContextoVinculadoNulo(t, ctx, err, ErrIdentidadOfflineNoDisponible)
	})

	t.Run("contexto nulo", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		ctx, err := entorno.fachada.AutenticarYVincular(nil, []byte("x"))
		exigirContextoVinculadoNulo(t, ctx, err, ErrIdentidadOfflineNoDisponible)
		var contextoNulo *contextoIdentidadOfflineNulo
		ctx, err = entorno.fachada.AutenticarYVincular(contextoNulo, []byte("x"))
		exigirContextoVinculadoNulo(t, ctx, err, ErrIdentidadOfflineNoDisponible)
	})

	t.Run("canal ausente", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		ctx, err := entorno.fachada.AutenticarYVincular(context.Background(), []byte("x"))
		exigirContextoVinculadoNulo(t, ctx, err, httpseguridad.ErrCanalProxyNoAutenticado)
		if efectos := entorno.efectos(); efectos != (efectosIdentidadOfflinePrueba{}) {
			t.Fatalf("el canal ausente produjo efectos: %+v", efectos)
		}
	})

	t.Run("asercion invalida", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		var resultado context.Context
		var err error
		entorno.ejecutarEnC4(t, func(ctx context.Context) {
			resultado, err = entorno.fachada.AutenticarYVincular(ctx, nil)
		})
		exigirContextoVinculadoNulo(t, resultado, err, httpseguridad.ErrAsercionAusente)
	})

	t.Run("sesion revocada", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		entorno.registro.errorRevalidacion = errors.New("revocacion sintetica")
		var resultado context.Context
		var err error
		entorno.ejecutarEnC4(t, func(ctx context.Context) {
			resultado, err = entorno.fachada.AutenticarYVincular(ctx, []byte("asercion-revocada"))
		})
		exigirContextoVinculadoNulo(t, resultado, err, httpseguridad.ErrSesionNoValida)
	})
}

func TestFachadaIdentidadOfflineAutenticarYVincularRechazaContextoPrevioYCruces(t *testing.T) {
	t.Run("capsula previa de la misma instancia", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		capsula, err := entorno.autenticarEnC4(t, []byte("asercion-previa"))
		if err != nil {
			t.Fatalf("autenticar capsula previa: %v", err)
		}
		previo, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
			context.Background(), capsula, entorno.canal,
		)
		if err != nil {
			t.Fatalf("vincular capsula previa: %v", err)
		}
		var resultado context.Context
		entorno.ejecutarEnC4(t, func(ctx context.Context) {
			ctxConPrevio := context.WithValue(
				previo, claveContextoCanalTLSInterno{}, ctx.Value(claveContextoCanalTLSInterno{}),
			)
			resultado, err = entorno.fachada.AutenticarYVincular(
				ctxConPrevio, []byte("asercion-con-contexto-previo"),
			)
		})
		exigirContextoVinculadoNulo(t, resultado, err, httpseguridad.ErrSesionNoValida)
	})

	t.Run("capsula previa de otra instancia", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		ajeno := nuevoEntornoIdentidadOfflinePrueba(t)
		capsulaAjena, err := ajeno.autenticarEnC4(t, []byte("asercion-ajena"))
		if err != nil {
			t.Fatalf("autenticar capsula ajena: %v", err)
		}
		ctxAjeno, err := ajeno.servicio.VincularCapsulaIdentidadPeticion(
			context.Background(), capsulaAjena, ajeno.canal,
		)
		if err != nil {
			t.Fatalf("vincular capsula ajena: %v", err)
		}
		var resultado context.Context
		entorno.ejecutarEnC4(t, func(ctx context.Context) {
			ctxCruzado := context.WithValue(
				ctxAjeno, claveContextoCanalTLSInterno{}, ctx.Value(claveContextoCanalTLSInterno{}),
			)
			resultado, err = entorno.fachada.AutenticarYVincular(
				ctxCruzado, []byte("asercion-con-capsula-cruzada"),
			)
		})
		exigirContextoVinculadoNulo(t, resultado, err, httpseguridad.ErrSesionNoValida)
	})
}

func TestFachadaIdentidadOfflineAutenticarYVincularConsumoUnico(t *testing.T) {
	entorno := nuevoEntornoIdentidadOfflinePrueba(t)
	const contendientes = 16
	resultados := make([]context.Context, contendientes)
	errores := make([]error, contendientes)
	var resultadoReplay context.Context
	var errorReplay error

	codigo := entorno.ejecutarEnC4(t, func(ctx context.Context) {
		inicio := make(chan struct{})
		var grupo sync.WaitGroup
		grupo.Add(contendientes)
		for indice := range contendientes {
			go func(indice int) {
				defer grupo.Done()
				<-inicio
				resultados[indice], errores[indice] = entorno.fachada.AutenticarYVincular(
					ctx, []byte("asercion-concurrente"),
				)
			}(indice)
		}
		close(inicio)
		grupo.Wait()
		resultadoReplay, errorReplay = entorno.fachada.AutenticarYVincular(
			ctx, []byte("asercion-replay"),
		)
	})
	if codigo != http.StatusNoContent {
		t.Fatalf("respuesta C4 inesperada: %d", codigo)
	}
	ganadores, rechazados := 0, 0
	var ganador context.Context
	for indice, err := range errores {
		switch {
		case err == nil && resultados[indice] != nil:
			ganadores++
			ganador = resultados[indice]
		case errors.Is(err, httpseguridad.ErrCanalProxyNoAutenticado) && resultados[indice] == nil:
			rechazados++
		default:
			t.Fatalf("resultado concurrente inesperado %d: contexto_nulo=%t error=%v", indice, resultados[indice] == nil, err)
		}
	}
	if ganadores != 1 || rechazados != contendientes-1 {
		t.Fatalf("consumo concurrente: ganadores=%d rechazados=%d", ganadores, rechazados)
	}
	exigirContextoVinculadoNulo(
		t, resultadoReplay, errorReplay, httpseguridad.ErrCanalProxyNoAutenticado,
	)
	if efectos := entorno.efectos(); efectos != (efectosIdentidadOfflinePrueba{1, 3, 1, 2}) {
		t.Fatalf("los perdedores o el replay produjeron efectos: %+v", efectos)
	}
	if _, _, err := entorno.servicio.ExtraerCapsulaIdentidadPeticion(ganador); err != nil {
		t.Fatalf("el unico ganador no es extraible: %v", err)
	}
}

func TestFachadaIdentidadOfflineAutenticarYVincularCompiteConAutenticar(t *testing.T) {
	t.Run("Autenticar gana antes de AutenticarYVincular", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		var capsula httpseguridad.CapsulaIdentidadPeticion
		var vinculado context.Context
		var errorAutenticar, errorVincular error
		entorno.ejecutarEnC4(t, func(ctx context.Context) {
			capsula, errorAutenticar = entorno.fachada.Autenticar(ctx, []byte("asercion-autenticar-primero"))
			vinculado, errorVincular = entorno.fachada.AutenticarYVincular(ctx, []byte("asercion-vincular-despues"))
		})
		if errorAutenticar != nil || vinculado != nil ||
			!errors.Is(errorVincular, httpseguridad.ErrCanalProxyNoAutenticado) {
			t.Fatalf("resultado mixto inesperado: autenticar=%v contexto_nulo=%t vincular=%v",
				errorAutenticar, vinculado == nil, errorVincular)
		}
		ctxCapsula, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
			context.Background(), capsula, entorno.canal,
		)
		if err != nil {
			t.Fatalf("la unica capsula ganadora no se pudo vincular: %v", err)
		}
		if _, _, err = entorno.servicio.ExtraerCapsulaIdentidadPeticion(ctxCapsula); err != nil {
			t.Fatalf("la unica capsula ganadora no se pudo extraer: %v", err)
		}
	})

	t.Run("AutenticarYVincular gana antes de Autenticar", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		var capsula httpseguridad.CapsulaIdentidadPeticion
		var vinculado context.Context
		var errorAutenticar, errorVincular error
		entorno.ejecutarEnC4(t, func(ctx context.Context) {
			vinculado, errorVincular = entorno.fachada.AutenticarYVincular(ctx, []byte("asercion-vincular-primero"))
			capsula, errorAutenticar = entorno.fachada.Autenticar(ctx, []byte("asercion-autenticar-despues"))
		})
		if errorVincular != nil || vinculado == nil ||
			!errors.Is(errorAutenticar, httpseguridad.ErrCanalProxyNoAutenticado) {
			t.Fatalf("resultado mixto inesperado: vincular=%v contexto_nulo=%t autenticar=%v",
				errorVincular, vinculado == nil, errorAutenticar)
		}
		if _, _, err := entorno.servicio.ExtraerCapsulaIdentidadPeticion(vinculado); err != nil {
			t.Fatalf("el unico contexto ganador no se pudo extraer: %v", err)
		}
		exigirCapsulaIdentidadNoUtilizable(t, entorno, capsula)
	})

	t.Run("carrera mixta tiene un unico ganador", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		var capsula httpseguridad.CapsulaIdentidadPeticion
		var vinculado context.Context
		var errorAutenticar, errorVincular error
		entorno.ejecutarEnC4(t, func(ctx context.Context) {
			inicio := make(chan struct{})
			var grupo sync.WaitGroup
			grupo.Add(2)
			go func() {
				defer grupo.Done()
				<-inicio
				capsula, errorAutenticar = entorno.fachada.Autenticar(ctx, []byte("asercion-carrera-autenticar"))
			}()
			go func() {
				defer grupo.Done()
				<-inicio
				vinculado, errorVincular = entorno.fachada.AutenticarYVincular(ctx, []byte("asercion-carrera-vincular"))
			}()
			close(inicio)
			grupo.Wait()
		})

		ganadores := 0
		if errorAutenticar == nil {
			ganadores++
		}
		if errorVincular == nil {
			ganadores++
		}
		if ganadores != 1 {
			t.Fatalf("la carrera mixta no tuvo un unico ganador: autenticar=%v vincular=%v", errorAutenticar, errorVincular)
		}
		if errorAutenticar == nil {
			if vinculado != nil || !errors.Is(errorVincular, httpseguridad.ErrCanalProxyNoAutenticado) {
				t.Fatalf("Autenticar ganador dejo un segundo contexto: contexto_nulo=%t error=%v", vinculado == nil, errorVincular)
			}
			ctxCapsula, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
				context.Background(), capsula, entorno.canal,
			)
			if err != nil {
				t.Fatalf("capsula ganadora de carrera no vinculable: %v", err)
			}
			if _, _, err = entorno.servicio.ExtraerCapsulaIdentidadPeticion(ctxCapsula); err != nil {
				t.Fatalf("capsula ganadora de carrera no extraible: %v", err)
			}
			return
		}
		if vinculado == nil || !errors.Is(errorAutenticar, httpseguridad.ErrCanalProxyNoAutenticado) {
			t.Fatalf("AutenticarYVincular ganador dejo una segunda capsula: contexto_nulo=%t error=%v", vinculado == nil, errorAutenticar)
		}
		if _, _, err := entorno.servicio.ExtraerCapsulaIdentidadPeticion(vinculado); err != nil {
			t.Fatalf("contexto ganador de carrera no extraible: %v", err)
		}
		exigirCapsulaIdentidadNoUtilizable(t, entorno, capsula)
	})
}

func TestFachadaIdentidadOfflineAutenticarYVincularCancelacionAntesYDurante(t *testing.T) {
	t.Run("antes no consume el canal", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		var cancelado, valido context.Context
		var errorCancelado, errorValido error
		entorno.ejecutarEnC4(t, func(ctx context.Context) {
			ctxCancelado, cancelar := context.WithCancel(ctx)
			cancelar()
			cancelado, errorCancelado = entorno.fachada.AutenticarYVincular(
				ctxCancelado, []byte("asercion-cancelada"),
			)
			valido, errorValido = entorno.fachada.AutenticarYVincular(
				ctx, []byte("asercion-posterior"),
			)
		})
		exigirContextoVinculadoNulo(t, cancelado, errorCancelado, context.Canceled)
		if errorValido != nil || valido == nil {
			t.Fatalf("la cancelacion previa consumio el canal: contexto_nulo=%t error=%v", valido == nil, errorValido)
		}
	})

	t.Run("durante la verificacion", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		verificador := &verificadorCancelacionVinculoPrueba{iniciado: make(chan struct{})}
		servicio := debeServicioIdentidadOfflinePrueba(
			t, entorno.configuracion, verificador, entorno.evaluador, entorno.registro, entorno.ahora,
		)
		canal, err := servicio.AutenticarCanalTLSMutuo(entorno.intercambio.estadoServidor)
		if err != nil {
			t.Fatalf("autenticar canal del servicio bloqueante: %v", err)
		}
		verificador.asercion = asercionIdentidadOfflinePrueba(entorno.ahora, entorno.configuracion, canal)
		fachada, err := NuevaFachadaIdentidadOffline(servicio, entorno.propietario)
		if err != nil {
			t.Fatalf("crear fachada bloqueante: %v", err)
		}
		var resultado context.Context
		var errorResultado error
		entorno.ejecutarEnC4(t, func(ctx context.Context) {
			ctxCancelado, cancelar := context.WithCancel(ctx)
			terminado := make(chan struct{})
			go func() {
				resultado, errorResultado = fachada.AutenticarYVincular(
					ctxCancelado, []byte("asercion-cancelada-durante-verificacion"),
				)
				close(terminado)
			}()
			<-verificador.iniciado
			cancelar()
			<-terminado
		})
		exigirContextoVinculadoNulo(t, resultado, errorResultado, context.Canceled)
		if verificador.llamadas.Load() != 1 || entorno.registro.altas.Load() != 0 ||
			entorno.registro.revalidaciones.Load() != 0 {
			t.Fatalf("cancelacion durante produjo efectos: verificaciones=%d altas=%d revalidaciones=%d",
				verificador.llamadas.Load(), entorno.registro.altas.Load(), entorno.registro.revalidaciones.Load())
		}
	})
}

func TestFachadaIdentidadOfflineAutenticarYVincularConservaAutenticar(t *testing.T) {
	entorno := nuevoEntornoIdentidadOfflinePrueba(t)
	capsula, err := entorno.autenticarEnC4(t, []byte("regresion-autenticar"))
	if err != nil {
		t.Fatalf("Autenticar cambio su resultado: %v", err)
	}
	vinculado, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
		context.Background(), capsula, entorno.canal,
	)
	if err != nil {
		t.Fatalf("Autenticar dejo de emitir una capsula vinculable: %v", err)
	}
	if _, _, err = entorno.servicio.ExtraerCapsulaIdentidadPeticion(vinculado); err != nil {
		t.Fatalf("Autenticar dejo de emitir una capsula extraible: %v", err)
	}
}

type verificadorCancelacionVinculoPrueba struct {
	asercion httpseguridad.AsercionProxyIdentidad
	iniciado chan struct{}
	unaVez   sync.Once
	llamadas atomic.Int32
}

func (v *verificadorCancelacionVinculoPrueba) Verificar(
	ctx context.Context,
	_ []byte,
) (httpseguridad.AsercionProxyIdentidad, error) {
	v.llamadas.Add(1)
	v.unaVez.Do(func() { close(v.iniciado) })
	<-ctx.Done()
	resultado := v.asercion
	resultado.Factores = append([]httpseguridad.FactorAutenticacion(nil), v.asercion.Factores...)
	return resultado, nil
}

func exigirContextoVinculadoNulo(
	t *testing.T,
	resultado context.Context,
	err error,
	esperado error,
) {
	t.Helper()
	if resultado != nil || !errors.Is(err, esperado) {
		t.Fatalf("fallo no cerrado: contexto_nulo=%t error=%v esperado=%v", resultado == nil, err, esperado)
	}
}

func exigirCapsulaIdentidadNoUtilizable(
	t *testing.T,
	entorno *entornoIdentidadOfflinePrueba,
	capsula httpseguridad.CapsulaIdentidadPeticion,
) {
	t.Helper()
	ctx, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
		context.Background(), capsula, entorno.canal,
	)
	if ctx != nil || !errors.Is(err, httpseguridad.ErrSesionNoValida) {
		t.Fatalf("el perdedor devolvio una capsula utilizable: contexto_nulo=%t error=%v", ctx == nil, err)
	}
}
