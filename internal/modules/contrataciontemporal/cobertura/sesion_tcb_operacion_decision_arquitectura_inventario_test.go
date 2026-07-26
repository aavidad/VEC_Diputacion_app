package cobertura

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"
)

type entradaInventarioFronteraNominal struct {
	posicion                  token.Pos
	origen                    string
	tipo                      types.Type
	referenciaDirectaEsBypass bool
}

func tipoContieneReferenciaFronteraNominal(
	tipo types.Type,
	frontera fronteraNominalArquitectura,
	visitados map[types.Type]bool,
) bool {
	if tipo == nil {
		return false
	}
	if tipoEsExactamente(tipo, frontera.tipo) ||
		tipoEsImplementacionNominalHomologada(tipo, frontera) {
		return true
	}
	if visitados[tipo] {
		return false
	}
	visitados[tipo] = true

	switch concreto := tipo.(type) {
	case *types.Basic:
		return false
	case *types.Alias:
		if parametrosTipoContienenReferenciaFronteraNominal(
			concreto.TypeParams(),
			frontera,
			visitados,
		) {
			return true
		}
		if argumentos := concreto.TypeArgs(); argumentos != nil {
			for indice := 0; indice < argumentos.Len(); indice++ {
				if tipoContieneReferenciaFronteraNominal(
					argumentos.At(indice),
					frontera,
					visitados,
				) {
					return true
				}
			}
		}
		return tipoContieneReferenciaFronteraNominal(
			concreto.Rhs(),
			frontera,
			visitados,
		)
	case *types.Named:
		if parametrosTipoContienenReferenciaFronteraNominal(
			concreto.TypeParams(),
			frontera,
			visitados,
		) {
			return true
		}
		if argumentos := concreto.TypeArgs(); argumentos != nil {
			for indice := 0; indice < argumentos.Len(); indice++ {
				if tipoContieneReferenciaFronteraNominal(
					argumentos.At(indice),
					frontera,
					visitados,
				) {
					return true
				}
			}
		}
		return tipoContieneReferenciaFronteraNominal(
			concreto.Underlying(),
			frontera,
			visitados,
		)
	case *types.Pointer:
		return tipoContieneReferenciaFronteraNominal(
			concreto.Elem(),
			frontera,
			visitados,
		)
	case *types.Array:
		return tipoContieneReferenciaFronteraNominal(
			concreto.Elem(),
			frontera,
			visitados,
		)
	case *types.Slice:
		return tipoContieneReferenciaFronteraNominal(
			concreto.Elem(),
			frontera,
			visitados,
		)
	case *types.Map:
		return tipoContieneReferenciaFronteraNominal(
			concreto.Key(),
			frontera,
			visitados,
		) || tipoContieneReferenciaFronteraNominal(
			concreto.Elem(),
			frontera,
			visitados,
		)
	case *types.Chan:
		return tipoContieneReferenciaFronteraNominal(
			concreto.Elem(),
			frontera,
			visitados,
		)
	case *types.Struct:
		for indice := 0; indice < concreto.NumFields(); indice++ {
			if tipoContieneReferenciaFronteraNominal(
				concreto.Field(indice).Type(),
				frontera,
				visitados,
			) {
				return true
			}
		}
		return false
	case *types.Signature:
		if parametrosTipoContienenReferenciaFronteraNominal(
			concreto.RecvTypeParams(),
			frontera,
			visitados,
		) || parametrosTipoContienenReferenciaFronteraNominal(
			concreto.TypeParams(),
			frontera,
			visitados,
		) {
			return true
		}
		return tuplaContieneReferenciaFronteraNominal(
			concreto.Params(),
			frontera,
			visitados,
		) || tuplaContieneReferenciaFronteraNominal(
			concreto.Results(),
			frontera,
			visitados,
		)
	case *types.Interface:
		concreto.Complete()
		for indice := 0; indice < concreto.NumEmbeddeds(); indice++ {
			if tipoContieneReferenciaFronteraNominal(
				concreto.EmbeddedType(indice),
				frontera,
				visitados,
			) {
				return true
			}
		}
		for indice := 0; indice < concreto.NumExplicitMethods(); indice++ {
			if tipoContieneReferenciaFronteraNominal(
				concreto.ExplicitMethod(indice).Type(),
				frontera,
				visitados,
			) {
				return true
			}
		}
		return false
	case *types.Tuple:
		return tuplaContieneReferenciaFronteraNominal(
			concreto,
			frontera,
			visitados,
		)
	case *types.TypeParam:
		return tipoContieneReferenciaFronteraNominal(
			concreto.Constraint(),
			frontera,
			visitados,
		)
	case *types.Union:
		for indice := 0; indice < concreto.Len(); indice++ {
			if tipoContieneReferenciaFronteraNominal(
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

func parametrosTipoContienenReferenciaFronteraNominal(
	parametros *types.TypeParamList,
	frontera fronteraNominalArquitectura,
	visitados map[types.Type]bool,
) bool {
	if parametros == nil {
		return false
	}
	for indice := 0; indice < parametros.Len(); indice++ {
		if tipoContieneReferenciaFronteraNominal(
			parametros.At(indice),
			frontera,
			visitados,
		) {
			return true
		}
	}
	return false
}

func tuplaContieneReferenciaFronteraNominal(
	tupla *types.Tuple,
	frontera fronteraNominalArquitectura,
	visitados map[types.Type]bool,
) bool {
	if tupla == nil {
		return false
	}
	for indice := 0; indice < tupla.Len(); indice++ {
		if tipoContieneReferenciaFronteraNominal(
			tupla.At(indice).Type(),
			frontera,
			visitados,
		) {
			return true
		}
	}
	return false
}

func origenObjetoInventariadoFronteraNominal(
	objeto types.Object,
) (string, bool, bool) {
	switch concreto := objeto.(type) {
	case *types.TypeName:
		if concreto.IsAlias() {
			return "alias o reexportación nominal " + concreto.Name(),
				true, true
		}
		return "TypeName nominal " + concreto.Name(), false, true
	case *types.Const:
		return "constante " + concreto.Name(), false, true
	case *types.Func:
		return "función " + concreto.Name(), false, true
	case *types.Var:
		if concreto.IsField() {
			if concreto.Embedded() {
				return "campo embebido " + concreto.Name(), true, true
			}
			return "campo " + concreto.Name(), false, true
		}
		return "variable " + concreto.Name() +
			" (" + concreto.Kind().String() + ")", false, true
	default:
		return "", false, false
	}
}

func origenTipoAnonimoFronteraNominal(
	expresion ast.Expr,
) (string, bool) {
	switch expresion.(type) {
	case *ast.StructType:
		return "tipo struct anónimo", true
	case *ast.InterfaceType:
		return "tipo interface anónimo", true
	case *ast.FuncType:
		return "tipo función anónimo", true
	case *ast.ArrayType:
		return "tipo array o slice anónimo", true
	case *ast.MapType:
		return "tipo map anónimo", true
	case *ast.ChanType:
		return "tipo canal anónimo", true
	default:
		return "", false
	}
}

func expresionesDefinicionesLegitimasFronteraNominal(
	paquete paqueteAnalizadoSesionTCB,
) map[ast.Expr]struct{} {
	legitimas := make(map[ast.Expr]struct{})
	if paquete.metadatos.ImportPath != rutaPaqueteCoberturaSesionTCB {
		return legitimas
	}
	for _, fichero := range paquete.ficheros {
		ast.Inspect(fichero, func(nodo ast.Node) bool {
			tipo, correcto := nodo.(*ast.TypeSpec)
			if !correcto {
				return true
			}
			if tipo.Name.Name == "TransaccionOperacionDecisionCobertura" ||
				tipo.Name.Name == implementacionNominalSesionTCB {
				legitimas[tipo.Type] = struct{}{}
			}
			return true
		})
	}
	return legitimas
}

func inventariarFronteraNominal(
	paquete paqueteAnalizadoSesionTCB,
) []entradaInventarioFronteraNominal {
	entradas := make([]entradaInventarioFronteraNominal, 0)
	objetosVistos := make(map[types.Object]struct{})
	registrarObjeto := func(objeto types.Object) {
		if objeto == nil {
			return
		}
		if _, visto := objetosVistos[objeto]; visto {
			return
		}
		origen, referenciaDirectaEsBypass, pertinente :=
			origenObjetoInventariadoFronteraNominal(objeto)
		if !pertinente {
			return
		}
		objetosVistos[objeto] = struct{}{}
		entradas = append(entradas, entradaInventarioFronteraNominal{
			posicion:                  objeto.Pos(),
			origen:                    origen,
			tipo:                      objeto.Type(),
			referenciaDirectaEsBypass: referenciaDirectaEsBypass,
		})
	}

	// El scope garantiza que ninguna reexportación nominal de paquete quede
	// condicionada al nombre elegido ni a cómo go/types pobló Info.Defs.
	for _, nombre := range paquete.tipos.Scope().Names() {
		if objeto, correcto := paquete.tipos.Scope().Lookup(nombre).(*types.TypeName); correcto {
			registrarObjeto(objeto)
		}
	}
	for _, objeto := range paquete.info.Defs {
		registrarObjeto(objeto)
	}

	legitimas := expresionesDefinicionesLegitimasFronteraNominal(paquete)
	for expresion, valor := range paquete.info.Types {
		if _, legitima := legitimas[expresion]; legitima {
			continue
		}
		origen, anonimo := origenTipoAnonimoFronteraNominal(expresion)
		if !anonimo || valor.Type == nil {
			continue
		}
		entradas = append(entradas, entradaInventarioFronteraNominal{
			posicion: expresion.Pos(),
			origen:   origen,
			tipo:     valor.Type,
		})
	}

	// Los identificadores de un embedding pueden ser simultáneamente usos de
	// un TypeName y declaraciones de campo. Se recorren también desde el AST
	// para que ni una interfaz ni un struct anónimos dependan de ese detalle.
	for _, fichero := range paquete.ficheros {
		ast.Inspect(fichero, func(nodo ast.Node) bool {
			var campos []*ast.Field
			var origen string
			switch concreto := nodo.(type) {
			case *ast.InterfaceType:
				campos = concreto.Methods.List
				origen = "interfaz embebida"
			case *ast.StructType:
				campos = concreto.Fields.List
				origen = "campo embebido de struct"
			default:
				return true
			}
			for _, campo := range campos {
				if len(campo.Names) != 0 {
					continue
				}
				valor, existe := paquete.info.Types[campo.Type]
				if !existe || valor.Type == nil {
					continue
				}
				entradas = append(
					entradas,
					entradaInventarioFronteraNominal{
						posicion:                  campo.Type.Pos(),
						origen:                    origen,
						tipo:                      valor.Type,
						referenciaDirectaEsBypass: true,
					},
				)
			}
			return true
		})
	}

	sort.Slice(entradas, func(i, j int) bool {
		if entradas[i].posicion != entradas[j].posicion {
			return entradas[i].posicion < entradas[j].posicion
		}
		return entradas[i].origen < entradas[j].origen
	})
	return entradas
}
