package cobertura

import (
	"fmt"
	"go/token"
	"go/types"
	"sort"
	"strings"
	"testing"
)

const implementacionNominalSesionTCB = "transaccionOperacionDecisionCoberturaTCB"

type fronteraNominalArquitectura struct {
	tipo           types.Type
	interfaz       *types.Interface
	implementacion types.Type
}

type hallazgoFronteraNominal struct {
	posicion token.Pos
	origen   string
	tipo     string
}

func (a analisisArquitecturaSesionTCB) implementaEjecutor(
	tipo types.Type,
) bool {
	tipo = types.Unalias(tipo)
	return types.Implements(tipo, a.interfazExec) ||
		implementaConPunteroSesionTCB(tipo, a.interfazExec)
}

func (a analisisArquitecturaSesionTCB) implementaSesion(
	tipo types.Type,
) bool {
	tipo = types.Unalias(tipo)
	return types.Implements(tipo, a.interfazTx) ||
		implementaConPunteroSesionTCB(tipo, a.interfazTx)
}

func implementaConPunteroSesionTCB(
	tipo types.Type,
	interfaz *types.Interface,
) bool {
	if _, esPuntero := tipo.(*types.Pointer); esPuntero {
		return false
	}
	return types.Implements(types.NewPointer(tipo), interfaz)
}

func fronteraNominalParaPaquete(
	paquete *types.Package,
	coberturaImportada *types.Package,
) (fronteraNominalArquitectura, error) {
	propietario := coberturaImportada
	if paquete.Path() == rutaPaqueteCoberturaSesionTCB {
		propietario = paquete
	}
	objeto := propietario.Scope().Lookup(
		"TransaccionOperacionDecisionCobertura",
	)
	if objeto == nil {
		return fronteraNominalArquitectura{},
			fmt.Errorf("frontera nominal ausente en %s", propietario.Path())
	}
	interfaz, correcta :=
		types.Unalias(objeto.Type()).Underlying().(*types.Interface)
	if !correcta {
		return fronteraNominalArquitectura{},
			fmt.Errorf("frontera nominal no es interfaz en %s", propietario.Path())
	}
	interfaz.Complete()
	frontera := fronteraNominalArquitectura{
		tipo:     objeto.Type(),
		interfaz: interfaz,
	}
	if paquete.Path() == rutaPaqueteCoberturaSesionTCB {
		implementacion := paquete.Scope().Lookup(
			implementacionNominalSesionTCB,
		)
		if implementacion == nil {
			return fronteraNominalArquitectura{},
				fmt.Errorf("implementación nominal homologada ausente")
		}
		frontera.implementacion = implementacion.Type()
	}
	return frontera, nil
}

func tipoEsExactamente(
	tipo types.Type,
	esperado types.Type,
) bool {
	return tipo != nil &&
		esperado != nil &&
		types.Identical(types.Unalias(tipo), types.Unalias(esperado))
}

func tipoEsImplementacionNominalHomologada(
	tipo types.Type,
	frontera fronteraNominalArquitectura,
) bool {
	if tipo == nil || frontera.implementacion == nil {
		return false
	}
	tipo = types.Unalias(tipo)
	if puntero, correcto := tipo.(*types.Pointer); correcto {
		tipo = types.Unalias(puntero.Elem())
	}
	return types.Identical(
		tipo,
		types.Unalias(frontera.implementacion),
	)
}

// tipoIncorporaFronteraNominal conserva los alias antes de desplegarlos y
// recorre todas las formas de tipo de go/types. Una referencia directa a la
// dependencia sellada es válida; crear otro nombre, interfaz o valor que la
// implemente por embedding no lo es.
func tipoIncorporaFronteraNominal(
	tipo types.Type,
	frontera fronteraNominalArquitectura,
	visitados map[types.Type]bool,
) bool {
	if tipo == nil {
		return false
	}
	if alias, correcto := tipo.(*types.Alias); correcto {
		desplegado := types.Unalias(alias)
		return tipoEsExactamente(desplegado, frontera.tipo) ||
			tipoEsImplementacionNominalHomologada(desplegado, frontera) ||
			tipoIncorporaFronteraNominal(
				desplegado,
				frontera,
				visitados,
			)
	}
	if tipoEsExactamente(tipo, frontera.tipo) ||
		tipoEsImplementacionNominalHomologada(tipo, frontera) {
		return false
	}
	if _, basico := tipo.(*types.Basic); basico {
		return false
	}
	if visitados[tipo] {
		return false
	}
	visitados[tipo] = true

	if types.Implements(tipo, frontera.interfaz) ||
		implementaConPunteroSesionTCB(tipo, frontera.interfaz) {
		return true
	}

	switch concreto := tipo.(type) {
	case *types.Named:
		if argumentos := concreto.TypeArgs(); argumentos != nil {
			for indice := 0; indice < argumentos.Len(); indice++ {
				if tipoIncorporaFronteraNominal(
					argumentos.At(indice),
					frontera,
					visitados,
				) {
					return true
				}
			}
		}
		return tipoIncorporaFronteraNominal(
			concreto.Underlying(),
			frontera,
			visitados,
		)
	case *types.Pointer:
		return tipoIncorporaFronteraNominal(
			concreto.Elem(),
			frontera,
			visitados,
		)
	case *types.Array:
		return tipoIncorporaFronteraNominal(
			concreto.Elem(),
			frontera,
			visitados,
		)
	case *types.Slice:
		return tipoIncorporaFronteraNominal(
			concreto.Elem(),
			frontera,
			visitados,
		)
	case *types.Map:
		return tipoIncorporaFronteraNominal(
			concreto.Key(),
			frontera,
			visitados,
		) || tipoIncorporaFronteraNominal(
			concreto.Elem(),
			frontera,
			visitados,
		)
	case *types.Chan:
		return tipoIncorporaFronteraNominal(
			concreto.Elem(),
			frontera,
			visitados,
		)
	case *types.Struct:
		for indice := 0; indice < concreto.NumFields(); indice++ {
			if tipoIncorporaFronteraNominal(
				concreto.Field(indice).Type(),
				frontera,
				visitados,
			) {
				return true
			}
		}
		return false
	case *types.Signature:
		if parametrosTipoIncorporanFronteraNominal(
			concreto.RecvTypeParams(),
			frontera,
			visitados,
		) || parametrosTipoIncorporanFronteraNominal(
			concreto.TypeParams(),
			frontera,
			visitados,
		) {
			return true
		}
		return tuplaIncorporaFronteraNominal(
			concreto.Params(),
			frontera,
			visitados,
		) || tuplaIncorporaFronteraNominal(
			concreto.Results(),
			frontera,
			visitados,
		)
	case *types.Interface:
		concreto.Complete()
		for indice := 0; indice < concreto.NumEmbeddeds(); indice++ {
			if tipoIncorporaFronteraNominal(
				concreto.EmbeddedType(indice),
				frontera,
				visitados,
			) {
				return true
			}
		}
		for indice := 0; indice < concreto.NumExplicitMethods(); indice++ {
			if tipoIncorporaFronteraNominal(
				concreto.ExplicitMethod(indice).Type(),
				frontera,
				visitados,
			) {
				return true
			}
		}
		return false
	case *types.Tuple:
		return tuplaIncorporaFronteraNominal(
			concreto,
			frontera,
			visitados,
		)
	case *types.TypeParam:
		return tipoIncorporaFronteraNominal(
			concreto.Constraint(),
			frontera,
			visitados,
		)
	case *types.Union:
		for indice := 0; indice < concreto.Len(); indice++ {
			if tipoIncorporaFronteraNominal(
				concreto.Term(indice).Type(),
				frontera,
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

func parametrosTipoIncorporanFronteraNominal(
	parametros *types.TypeParamList,
	frontera fronteraNominalArquitectura,
	visitados map[types.Type]bool,
) bool {
	if parametros == nil {
		return false
	}
	for indice := 0; indice < parametros.Len(); indice++ {
		if tipoIncorporaFronteraNominal(
			parametros.At(indice),
			frontera,
			visitados,
		) {
			return true
		}
	}
	return false
}

func tuplaIncorporaFronteraNominal(
	tupla *types.Tuple,
	frontera fronteraNominalArquitectura,
	visitados map[types.Type]bool,
) bool {
	if tupla == nil {
		return false
	}
	for indice := 0; indice < tupla.Len(); indice++ {
		if tipoIncorporaFronteraNominal(
			tupla.At(indice).Type(),
			frontera,
			visitados,
		) {
			return true
		}
	}
	return false
}

func hallarIncorporacionesFronteraNominal(
	paquete paqueteAnalizadoSesionTCB,
	coberturaImportada *types.Package,
) ([]hallazgoFronteraNominal, error) {
	frontera, err := fronteraNominalParaPaquete(
		paquete.tipos,
		coberturaImportada,
	)
	if err != nil {
		return nil, err
	}
	hallazgos := make([]hallazgoFronteraNominal, 0)
	for _, entrada := range inventariarFronteraNominal(paquete) {
		incorpora := tipoIncorporaFronteraNominal(
			entrada.tipo,
			frontera,
			make(map[types.Type]bool),
		)
		if !incorpora && entrada.referenciaDirectaEsBypass {
			incorpora = tipoContieneReferenciaFronteraNominal(
				entrada.tipo,
				frontera,
				make(map[types.Type]bool),
			)
		}
		if !incorpora {
			continue
		}
		hallazgos = append(hallazgos, hallazgoFronteraNominal{
			posicion: entrada.posicion,
			origen:   entrada.origen,
			tipo: types.TypeString(entrada.tipo, func(*types.Package) string {
				return ""
			}),
		})
	}
	sort.Slice(hallazgos, func(i, j int) bool {
		if hallazgos[i].posicion != hallazgos[j].posicion {
			return hallazgos[i].posicion < hallazgos[j].posicion
		}
		return hallazgos[i].origen < hallazgos[j].origen
	})
	return hallazgos, nil
}

func TestArquitecturaFronteraNominalSelladaEInventarioExhaustivo(
	t *testing.T,
) {
	analisis := obtenerAnalisisArquitecturaSesionTCB(t)
	if analisis.interfazNominal.NumExplicitMethods() != 1 ||
		analisis.interfazNominal.ExplicitMethod(0).Exported() {
		t.Fatal("TransaccionOperacionDecisionCobertura dejó de estar sellada")
	}

	encontradas := 0
	for _, paquete := range analisis.paquetes {
		hallazgos, err := hallarIncorporacionesFronteraNominal(
			paquete,
			analisis.cobertura,
		)
		if err != nil {
			t.Fatal(err)
		}
		for _, hallazgo := range hallazgos {
			t.Errorf(
				"%s incorpora o reexporta la frontera nominal mediante %s (%s) en %s",
				paquete.metadatos.ImportPath,
				hallazgo.origen,
				hallazgo.tipo,
				analisis.conjunto.Position(hallazgo.posicion),
			)
		}
		if paquete.metadatos.ImportPath != rutaPaqueteCoberturaSesionTCB {
			continue
		}
		objeto, esTipo := paquete.tipos.Scope().Lookup(
			implementacionNominalSesionTCB,
		).(*types.TypeName)
		frontera, err := fronteraNominalParaPaquete(
			paquete.tipos,
			analisis.cobertura,
		)
		if err != nil {
			t.Fatal(err)
		}
		if esTipo &&
			!objeto.IsAlias() &&
			(types.Implements(objeto.Type(), frontera.interfaz) ||
				implementaConPunteroSesionTCB(
					objeto.Type(),
					frontera.interfaz,
				)) {
			encontradas++
		}
	}
	if encontradas != 1 {
		t.Fatalf(
			"implementaciones nominales homologadas: %d; se esperaba 1",
			encontradas,
		)
	}
}

func TestArquitecturaImplementadoresTCBProductivosQuedanAcotados(t *testing.T) {
	analisis := obtenerAnalisisArquitecturaSesionTCB(t)
	for _, paquete := range analisis.paquetes {
		for _, nombre := range paquete.tipos.Scope().Names() {
			objeto, esTipo :=
				paquete.tipos.Scope().Lookup(nombre).(*types.TypeName)
			if !esTipo {
				continue
			}
			tipo := types.Unalias(objeto.Type())
			if _, esInterfaz := tipo.Underlying().(*types.Interface); esInterfaz {
				continue
			}
			ejecutor := analisis.implementaEjecutor(objeto.Type())
			sesion := analisis.implementaSesion(objeto.Type())
			if !ejecutor && !sesion {
				continue
			}
			if paquete.metadatos.ImportPath ==
				rutaPaquetePostgreSQLSesionTCB {
				continue
			}
			capacidades := make([]string, 0, 2)
			if ejecutor {
				capacidades = append(capacidades, "EjecutorSesionTCB")
			}
			if sesion {
				capacidades = append(capacidades, "SesionTCB")
			}
			t.Errorf(
				"implementador productivo de %s fuera de allowlist: %s.%s",
				strings.Join(capacidades, "+"),
				paquete.metadatos.ImportPath,
				nombre,
			)
		}
	}
}
