package cobertura

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
)

const rutaPaqueteCoberturaSesionTCB = "vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
const rutaPaquetePostgreSQLSesionTCB = "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"

var simbolosNominalesProhibidosEnCanales = map[string]struct{}{
	"OrdenOperacionDecisionCobertura":       {},
	"TransaccionOperacionDecisionCobertura": {},
}

func raizRepositorioSesionTCBPrueba(t *testing.T) string {
	t.Helper()
	_, fichero, _, correcto := runtime.Caller(0)
	if !correcto {
		t.Fatal("no se localizó el fichero de prueba")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(fichero), "../../../.."))
}

func rutaEsCanalNoConfiableSesionTCB(ruta string) bool {
	normalizada := "/" + strings.ToLower(filepath.ToSlash(ruta)) + "/"
	for _, segmento := range []string{
		"/http", "/cli/", "/mcp/", "/desktop/", "/escritorio/", "/cmd/",
		"/internal/app/server/", "/adapters/handler/", "/web/", "/api/",
	} {
		if strings.Contains(normalizada, segmento) {
			return true
		}
	}
	return false
}

type paqueteProductivoSesionTCB struct {
	Dir             string
	ImportPath      string
	Export          string
	GoFiles         []string
	CompiledGoFiles []string
	Module          *struct {
		Path string
	}
}

type paqueteAnalizadoSesionTCB struct {
	metadatos paqueteProductivoSesionTCB
	tipos     *types.Package
	info      *types.Info
	ficheros  []*ast.File
}

type analisisArquitecturaSesionTCB struct {
	conjunto        *token.FileSet
	cobertura       *types.Package
	paquetes        []paqueteAnalizadoSesionTCB
	simbolosTCB     map[string]struct{}
	interfazExec    *types.Interface
	interfazTx      *types.Interface
	interfazNominal *types.Interface
}

var (
	analisisArquitecturaSesionTCBUnaVez sync.Once
	analisisArquitecturaSesionTCBCache  analisisArquitecturaSesionTCB
	errAnalisisArquitecturaSesionTCB    error
)

func obtenerAnalisisArquitecturaSesionTCB(
	t *testing.T,
) analisisArquitecturaSesionTCB {
	t.Helper()
	analisisArquitecturaSesionTCBUnaVez.Do(func() {
		analisisArquitecturaSesionTCBCache,
			errAnalisisArquitecturaSesionTCB =
			analizarArquitecturaProductivaSesionTCB(
				raizRepositorioSesionTCBPrueba(t),
			)
	})
	if errAnalisisArquitecturaSesionTCB != nil {
		t.Fatal(errAnalisisArquitecturaSesionTCB)
	}
	return analisisArquitecturaSesionTCBCache
}

func analizarArquitecturaProductivaSesionTCB(
	raiz string,
) (analisisArquitecturaSesionTCB, error) {
	paquetes, err := paquetesVersionablesSesionTCB(raiz)
	if err != nil {
		return analisisArquitecturaSesionTCB{}, err
	}
	argumentos := []string{
		"list",
		"-json",
		"-compiled",
		"-export",
		"-deps",
	}
	argumentos = append(argumentos, paquetes...)
	comando := exec.Command("go", argumentos...)
	comando.Dir = raiz
	salida, err := comando.CombinedOutput()
	if err != nil {
		return analisisArquitecturaSesionTCB{}, fmt.Errorf(
			"go list de arquitectura TCB: %w: %s",
			err,
			strings.TrimSpace(string(salida)),
		)
	}

	decodificador := json.NewDecoder(strings.NewReader(string(salida)))
	var metadatos []paqueteProductivoSesionTCB
	exportaciones := make(map[string]string)
	for {
		var paquete paqueteProductivoSesionTCB
		err = decodificador.Decode(&paquete)
		if err == io.EOF {
			break
		}
		if err != nil {
			return analisisArquitecturaSesionTCB{}, err
		}
		if paquete.Export != "" {
			exportaciones[paquete.ImportPath] = paquete.Export
		}
		if paquete.Module != nil &&
			paquete.Module.Path == "vec-diputacion-granada" {
			metadatos = append(metadatos, paquete)
		}
	}

	conjunto := token.NewFileSet()
	importador := importer.ForCompiler(
		conjunto,
		"gc",
		func(ruta string) (io.ReadCloser, error) {
			exportacion, existe := exportaciones[ruta]
			if !existe || exportacion == "" {
				return nil, fmt.Errorf(
					"exportación ausente para %s",
					ruta,
				)
			}
			return os.Open(exportacion)
		},
	)
	cobertura, err := importador.Import(rutaPaqueteCoberturaSesionTCB)
	if err != nil {
		return analisisArquitecturaSesionTCB{}, err
	}
	interfazExec, err := interfazExportadaSesionTCB(
		cobertura,
		"EjecutorSesionTCBOperacionDecisionCobertura",
	)
	if err != nil {
		return analisisArquitecturaSesionTCB{}, err
	}
	interfazTx, err := interfazExportadaSesionTCB(
		cobertura,
		"SesionTCBOperacionDecisionCobertura",
	)
	if err != nil {
		return analisisArquitecturaSesionTCB{}, err
	}
	interfazNominal, err := interfazExportadaSesionTCB(
		cobertura,
		"TransaccionOperacionDecisionCobertura",
	)
	if err != nil {
		return analisisArquitecturaSesionTCB{}, err
	}

	analisis := analisisArquitecturaSesionTCB{
		conjunto:        conjunto,
		cobertura:       cobertura,
		simbolosTCB:     inventariarSimbolosTCB(cobertura),
		interfazExec:    interfazExec,
		interfazTx:      interfazTx,
		interfazNominal: interfazNominal,
	}
	if len(analisis.simbolosTCB) == 0 {
		return analisisArquitecturaSesionTCB{},
			fmt.Errorf("inventario vacío de símbolos TCB")
	}
	for _, paquete := range metadatos {
		ficheros, errParseo := parsearPaqueteProductivoSesionTCB(
			conjunto,
			paquete,
		)
		if errParseo != nil {
			return analisisArquitecturaSesionTCB{}, errParseo
		}
		info := &types.Info{
			Defs:       make(map[*ast.Ident]types.Object),
			Uses:       make(map[*ast.Ident]types.Object),
			Selections: make(map[*ast.SelectorExpr]*types.Selection),
			Types:      make(map[ast.Expr]types.TypeAndValue),
		}
		configuracion := types.Config{
			Importer: importador,
			Sizes:    types.SizesFor("gc", runtime.GOARCH),
		}
		paqueteTipos, errTipos := configuracion.Check(
			paquete.ImportPath,
			conjunto,
			ficheros,
			info,
		)
		if errTipos != nil {
			return analisisArquitecturaSesionTCB{}, errTipos
		}
		analisis.paquetes = append(
			analisis.paquetes,
			paqueteAnalizadoSesionTCB{
				metadatos: paquete,
				tipos:     paqueteTipos,
				info:      info,
				ficheros:  ficheros,
			},
		)
	}
	return analisis, nil
}

func paquetesVersionablesSesionTCB(raiz string) ([]string, error) {
	comando := exec.Command(
		"git",
		"ls-files",
		"-co",
		"--exclude-standard",
		"--",
		"*.go",
	)
	comando.Dir = raiz
	salida, err := comando.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(
			"inventario Git de arquitectura TCB: %w: %s",
			err,
			strings.TrimSpace(string(salida)),
		)
	}
	directorios := make(map[string]struct{})
	for _, fichero := range strings.Fields(string(salida)) {
		directorio := filepath.ToSlash(filepath.Dir(fichero))
		if directorio == "." {
			directorios["."] = struct{}{}
			continue
		}
		directorios["./"+directorio] = struct{}{}
	}
	paquetes := make([]string, 0, len(directorios))
	for directorio := range directorios {
		paquetes = append(paquetes, directorio)
	}
	sort.Strings(paquetes)
	if len(paquetes) == 0 {
		return nil, fmt.Errorf("inventario vacío de paquetes Go versionables")
	}
	return paquetes, nil
}

func inventariarSimbolosTCB(paquete *types.Package) map[string]struct{} {
	inventario := make(map[string]struct{})
	for _, nombre := range paquete.Scope().Names() {
		if strings.Contains(nombre, "TCB") {
			inventario[nombre] = struct{}{}
		}
	}
	return inventario
}

func interfazExportadaSesionTCB(
	paquete *types.Package,
	nombre string,
) (*types.Interface, error) {
	objeto := paquete.Scope().Lookup(nombre)
	if objeto == nil {
		return nil, fmt.Errorf("no existe %s.%s", paquete.Path(), nombre)
	}
	interfaz, correcta := types.Unalias(objeto.Type()).Underlying().(*types.Interface)
	if !correcta {
		return nil, fmt.Errorf("%s.%s no es interfaz", paquete.Path(), nombre)
	}
	interfaz.Complete()
	return interfaz, nil
}

func parsearPaqueteProductivoSesionTCB(
	conjunto *token.FileSet,
	paquete paqueteProductivoSesionTCB,
) ([]*ast.File, error) {
	nombres := paquete.CompiledGoFiles
	if len(nombres) == 0 {
		nombres = paquete.GoFiles
	}
	ficheros := make([]*ast.File, 0, len(nombres))
	for _, nombre := range nombres {
		ruta := nombre
		if !filepath.IsAbs(ruta) {
			ruta = filepath.Join(paquete.Dir, nombre)
		}
		archivo, err := parser.ParseFile(conjunto, ruta, nil, 0)
		if err != nil {
			return nil, err
		}
		ficheros = append(ficheros, archivo)
	}
	return ficheros, nil
}

func objetoEsSimboloTCB(
	objeto types.Object,
	simbolos map[string]struct{},
) bool {
	if objeto == nil || objeto.Pkg() == nil ||
		objeto.Pkg().Path() != rutaPaqueteCoberturaSesionTCB {
		return false
	}
	_, existe := simbolos[objeto.Name()]
	return existe
}

func objetoEsCapacidadProhibidaEnCanal(
	objeto types.Object,
	simbolos map[string]struct{},
) bool {
	if objetoEsSimboloTCB(objeto, simbolos) {
		return true
	}
	if objeto == nil || objeto.Pkg() == nil ||
		objeto.Pkg().Path() != rutaPaqueteCoberturaSesionTCB {
		return false
	}
	_, existe := simbolosNominalesProhibidosEnCanales[objeto.Name()]
	return existe
}

func TestArquitecturaCanalesNoPoseenOrdenNiSesionTCB(t *testing.T) {
	analisis := obtenerAnalisisArquitecturaSesionTCB(t)
	for _, paquete := range analisis.paquetes {
		if !rutaEsCanalNoConfiableSesionTCB(paquete.metadatos.ImportPath) {
			continue
		}
		for identificador, objeto := range paquete.info.Uses {
			if !objetoEsCapacidadProhibidaEnCanal(
				objeto,
				analisis.simbolosTCB,
			) {
				continue
			}
			t.Errorf(
				"canal %s referencia capacidad %s en %s",
				paquete.metadatos.ImportPath,
				objeto.Name(),
				analisis.conjunto.Position(identificador.Pos()),
			)
		}
		for _, nombre := range paquete.tipos.Scope().Names() {
			objeto := paquete.tipos.Scope().Lookup(nombre)
			if !tipoExponeCapacidadSesionTCB(
				objeto.Type(),
				paquete.metadatos.ImportPath,
				analisis.simbolosTCB,
				make(map[types.Type]bool),
			) {
				continue
			}
			t.Errorf(
				"canal %s expone capacidad TCB mediante %s",
				paquete.metadatos.ImportPath,
				nombre,
			)
		}
	}
}

func TestArquitecturaNingunIntermediarioReexportaSimbolosTCB(t *testing.T) {
	analisis := obtenerAnalisisArquitecturaSesionTCB(t)
	for _, paquete := range analisis.paquetes {
		ruta := paquete.metadatos.ImportPath
		if ruta == rutaPaqueteCoberturaSesionTCB ||
			ruta == rutaPaquetePostgreSQLSesionTCB {
			continue
		}
		for identificador, objeto := range paquete.info.Uses {
			if !objetoEsSimboloTCB(objeto, analisis.simbolosTCB) {
				continue
			}
			t.Errorf(
				"%s usa el símbolo TCB %s fuera de la allowlist en %s",
				ruta,
				objeto.Name(),
				analisis.conjunto.Position(identificador.Pos()),
			)
		}
	}
}

func tipoExponeCapacidadSesionTCB(
	tipo types.Type,
	paquetePropietario string,
	simbolos map[string]struct{},
	visitados map[types.Type]bool,
) bool {
	if tipo == nil {
		return false
	}
	tipo = types.Unalias(tipo)
	if visitados[tipo] {
		return false
	}
	visitados[tipo] = true

	switch concreto := tipo.(type) {
	case *types.Basic:
		return false
	case *types.Named:
		objeto := concreto.Obj()
		if objetoEsCapacidadProhibidaEnCanal(objeto, simbolos) {
			return true
		}
		if argumentos := concreto.TypeArgs(); argumentos != nil {
			for indice := 0; indice < argumentos.Len(); indice++ {
				if tipoExponeCapacidadSesionTCB(
					argumentos.At(indice),
					paquetePropietario,
					simbolos,
					visitados,
				) {
					return true
				}
			}
		}
		_, esInterfaz := concreto.Underlying().(*types.Interface)
		if objeto.Pkg() != nil &&
			objeto.Pkg().Path() != paquetePropietario &&
			!esInterfaz {
			return false
		}
		if tipoExponeCapacidadSesionTCB(
			concreto.Underlying(),
			paquetePropietario,
			simbolos,
			visitados,
		) {
			return true
		}
		if objeto.Pkg() == nil ||
			objeto.Pkg().Path() == paquetePropietario {
			for indice := 0; indice < concreto.NumMethods(); indice++ {
				if tipoExponeCapacidadSesionTCB(
					concreto.Method(indice).Type(),
					paquetePropietario,
					simbolos,
					visitados,
				) {
					return true
				}
			}
		}
		return false
	case *types.Pointer:
		return tipoExponeCapacidadSesionTCB(
			concreto.Elem(),
			paquetePropietario,
			simbolos,
			visitados,
		)
	case *types.Array:
		return tipoExponeCapacidadSesionTCB(
			concreto.Elem(),
			paquetePropietario,
			simbolos,
			visitados,
		)
	case *types.Slice:
		return tipoExponeCapacidadSesionTCB(
			concreto.Elem(),
			paquetePropietario,
			simbolos,
			visitados,
		)
	case *types.Map:
		return tipoExponeCapacidadSesionTCB(
			concreto.Key(),
			paquetePropietario,
			simbolos,
			visitados,
		) || tipoExponeCapacidadSesionTCB(
			concreto.Elem(),
			paquetePropietario,
			simbolos,
			visitados,
		)
	case *types.Chan:
		return tipoExponeCapacidadSesionTCB(
			concreto.Elem(),
			paquetePropietario,
			simbolos,
			visitados,
		)
	case *types.Struct:
		for indice := 0; indice < concreto.NumFields(); indice++ {
			if tipoExponeCapacidadSesionTCB(
				concreto.Field(indice).Type(),
				paquetePropietario,
				simbolos,
				visitados,
			) {
				return true
			}
		}
		return false
	case *types.Signature:
		if parametrosTipoExponenCapacidadSesionTCB(
			concreto.RecvTypeParams(),
			paquetePropietario,
			simbolos,
			visitados,
		) || parametrosTipoExponenCapacidadSesionTCB(
			concreto.TypeParams(),
			paquetePropietario,
			simbolos,
			visitados,
		) {
			return true
		}
		return tuplaExponeCapacidadSesionTCB(
			concreto.Params(),
			paquetePropietario,
			simbolos,
			visitados,
		) || tuplaExponeCapacidadSesionTCB(
			concreto.Results(),
			paquetePropietario,
			simbolos,
			visitados,
		)
	case *types.Interface:
		concreto.Complete()
		for indice := 0; indice < concreto.NumEmbeddeds(); indice++ {
			if tipoExponeCapacidadSesionTCB(
				concreto.EmbeddedType(indice),
				paquetePropietario,
				simbolos,
				visitados,
			) {
				return true
			}
		}
		for indice := 0; indice < concreto.NumExplicitMethods(); indice++ {
			if tipoExponeCapacidadSesionTCB(
				concreto.ExplicitMethod(indice).Type(),
				paquetePropietario,
				simbolos,
				visitados,
			) {
				return true
			}
		}
		return false
	case *types.Tuple:
		return tuplaExponeCapacidadSesionTCB(
			concreto,
			paquetePropietario,
			simbolos,
			visitados,
		)
	case *types.TypeParam:
		return tipoExponeCapacidadSesionTCB(
			concreto.Constraint(),
			paquetePropietario,
			simbolos,
			visitados,
		)
	case *types.Union:
		for indice := 0; indice < concreto.Len(); indice++ {
			if tipoExponeCapacidadSesionTCB(
				concreto.Term(indice).Type(),
				paquetePropietario,
				simbolos,
				visitados,
			) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func parametrosTipoExponenCapacidadSesionTCB(
	parametros *types.TypeParamList,
	paquetePropietario string,
	simbolos map[string]struct{},
	visitados map[types.Type]bool,
) bool {
	if parametros == nil {
		return false
	}
	for indice := 0; indice < parametros.Len(); indice++ {
		if tipoExponeCapacidadSesionTCB(
			parametros.At(indice),
			paquetePropietario,
			simbolos,
			visitados,
		) {
			return true
		}
	}
	return false
}

func tuplaExponeCapacidadSesionTCB(
	tupla *types.Tuple,
	paquetePropietario string,
	simbolos map[string]struct{},
	visitados map[types.Type]bool,
) bool {
	if tupla == nil {
		return false
	}
	for indice := 0; indice < tupla.Len(); indice++ {
		if tipoExponeCapacidadSesionTCB(
			tupla.At(indice).Type(),
			paquetePropietario,
			simbolos,
			visitados,
		) {
			return true
		}
	}
	return false
}

func TestArquitecturaOrdenNoAdquiereDatosNiCodec(t *testing.T) {
	raiz := raizRepositorioSesionTCBPrueba(t)
	paquete := filepath.Join(
		raiz,
		"internal/modules/contrataciontemporal/cobertura",
	)
	ficheros, err := os.ReadDir(paquete)
	if err != nil {
		t.Fatal(err)
	}
	for _, fichero := range ficheros {
		if fichero.IsDir() ||
			filepath.Ext(fichero.Name()) != ".go" ||
			strings.HasSuffix(fichero.Name(), "_test.go") {
			continue
		}
		ruta := filepath.Join(paquete, fichero.Name())
		archivo, err := parser.ParseFile(
			token.NewFileSet(),
			ruta,
			nil,
			0,
		)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaracion := range archivo.Decls {
			funcion, ok := declaracion.(*ast.FuncDecl)
			if !ok || funcion.Recv == nil || len(funcion.Recv.List) != 1 {
				continue
			}
			receptor := funcion.Recv.List[0].Type
			if puntero, ok := receptor.(*ast.StarExpr); ok {
				receptor = puntero.X
			}
			tipo, ok := receptor.(*ast.Ident)
			if !ok || tipo.Name != "OrdenOperacionDecisionCobertura" {
				continue
			}
			nombre := strings.ToLower(funcion.Name.Name)
			if nombre == "datos" ||
				strings.HasPrefix(nombre, "marshal") ||
				strings.HasPrefix(nombre, "unmarshal") ||
				strings.HasPrefix(nombre, "gobencode") ||
				strings.HasPrefix(nombre, "gobdecode") {
				t.Fatalf(
					"la orden opaca adquirió el método prohibido %s en %s",
					funcion.Name.Name,
					fichero.Name(),
				)
			}
		}
	}
}

func TestArquitecturaSesionTCBNoContieneSQLNiPersistenciaSimulada(
	t *testing.T,
) {
	paquete := filepath.Join(
		raizRepositorioSesionTCBPrueba(t),
		"internal/modules/contrataciontemporal/cobertura",
	)
	ficheros, err := filepath.Glob(
		filepath.Join(paquete, "sesion_tcb_operacion_decision*.go"),
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, ruta := range ficheros {
		if strings.HasSuffix(ruta, "_test.go") {
			continue
		}
		archivo, err := parser.ParseFile(
			token.NewFileSet(),
			ruta,
			nil,
			0,
		)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(archivo, func(nodo ast.Node) bool {
			literal, correcto := nodo.(*ast.BasicLit)
			if !correcto || literal.Kind != token.STRING {
				return true
			}
			mayusculas := strings.ToUpper(literal.Value)
			for _, prohibido := range []string{
				"SELECT ", "INSERT ", "UPDATE ", "DELETE ", "PGX.",
				"DATABASE/SQL", "ADAPTERS/MEMORY",
			} {
				if strings.Contains(mayusculas, prohibido) {
					t.Errorf(
						"%s contiene persistencia impropia %q",
						filepath.Base(ruta),
						prohibido,
					)
				}
			}
			return true
		})
	}
}
