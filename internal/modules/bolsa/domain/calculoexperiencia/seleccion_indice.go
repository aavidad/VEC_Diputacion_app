package calculoexperiencia

import "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"

type criterioSeleccionIndexado struct {
	clave   string
	valores []string
}

type reglaSeleccionIndexada struct {
	clave     string
	seccion   string
	grupo     string
	prioridad uint32
	criterios []criterioSeleccionIndexado
}

// El indice persiste como tablas ordenadas inmutables, no como mapas. Los
// mapas usados durante su construccion se descartan antes de evaluar tramos.
type indiceSeleccion struct {
	clavesGobernadas []claveGobernadaSeleccion
	anclas           []anclaSeleccion
	reglas           []reglaSeleccionIndexada
	grupos           []grupoSeleccionIndexado
}

type claveGobernadaSeleccion struct {
	clave    string
	catalogo reglasbaremo.ReferenciaVersionada
}

type anclaSeleccion struct {
	clave      string
	valor      string
	posiciones []int
}

type grupoSeleccionIndexado struct {
	clave string
	modo  reglasbaremo.ModoCoincidenciaReglas
}

func construirIndiceSeleccion(plan PlanExperiencia) (indiceSeleccion, error) {
	indice := indiceSeleccion{}
	for _, grupo := range plan.GruposConcurrencia() {
		indice.grupos = append(indice.grupos, grupoSeleccionIndexado{
			clave: grupo.Clave(),
			modo:  grupo.CoincidenciaReglas().Modo(),
		})
	}
	ordenarGruposSeleccion(indice.grupos)

	catalogosPorClave := make(map[string]reglasbaremo.ReferenciaVersionada)
	anclasPorClave := make(map[string]map[string][]int)
	reglas := plan.Reglas()
	indice.reglas = make([]reglaSeleccionIndexada, len(reglas))
	for posicion, regla := range reglas {
		criterios := regla.Criterios()
		ordenarCriteriosSeleccion(criterios)
		indexada := reglaSeleccionIndexada{
			clave:     regla.Clave(),
			seccion:   regla.SeccionClave(),
			grupo:     regla.GrupoConcurrenciaClave(),
			prioridad: regla.PrioridadConcurrencia(),
			criterios: make([]criterioSeleccionIndexado, len(criterios)),
		}
		for indiceCriterio, criterio := range criterios {
			catalogo, existe := catalogosPorClave[criterio.Clave()]
			if existe && !referenciasPlanIguales(catalogo, criterio.Catalogo()) {
				return indiceSeleccion{}, nuevoError(
					"regla.criterios.catalogo",
					CodigoCatalogoCriterioIncompatible,
				)
			}
			catalogosPorClave[criterio.Clave()] = criterio.Catalogo()
			valores := criterio.Valores()
			ordenarCadenasSeleccion(valores)
			indexada.criterios[indiceCriterio] = criterioSeleccionIndexado{
				clave:   criterio.Clave(),
				valores: valores,
			}
		}
		indice.reglas[posicion] = indexada

		// Toda regla valida tiene al menos un criterio. Elegir el primero tras
		// ordenar por clave hace que el indice no dependa del orden recibido.
		ancla := indexada.criterios[0]
		porValor := anclasPorClave[ancla.clave]
		if porValor == nil {
			porValor = make(map[string][]int)
			anclasPorClave[ancla.clave] = porValor
		}
		for _, valor := range ancla.valores {
			porValor[valor] = append(porValor[valor], posicion)
		}
	}
	indice.clavesGobernadas = make([]claveGobernadaSeleccion, 0, len(catalogosPorClave))
	for clave, catalogo := range catalogosPorClave {
		indice.clavesGobernadas = append(indice.clavesGobernadas, claveGobernadaSeleccion{
			clave:    clave,
			catalogo: catalogo,
		})
	}
	ordenarClavesGobernadasSeleccion(indice.clavesGobernadas)
	for clave, porValor := range anclasPorClave {
		for valor, posiciones := range porValor {
			indice.anclas = append(indice.anclas, anclaSeleccion{
				clave:      clave,
				valor:      valor,
				posiciones: append([]int(nil), posiciones...),
			})
		}
	}
	ordenarAnclasSeleccion(indice.anclas)
	return indice, nil
}

func buscarClaveGobernadaSeleccion(
	valores []claveGobernadaSeleccion,
	clave string,
) (claveGobernadaSeleccion, bool) {
	inicio, fin := 0, len(valores)
	for inicio < fin {
		medio := inicio + (fin-inicio)/2
		if valores[medio].clave < clave {
			inicio = medio + 1
		} else {
			fin = medio
		}
	}
	if inicio == len(valores) || valores[inicio].clave != clave {
		return claveGobernadaSeleccion{}, false
	}
	return valores[inicio], true
}

func buscarAnclaSeleccion(
	valores []anclaSeleccion,
	clave string,
	valor string,
) (anclaSeleccion, bool) {
	inicio, fin := 0, len(valores)
	for inicio < fin {
		medio := inicio + (fin-inicio)/2
		actual := valores[medio]
		if actual.clave < clave || (actual.clave == clave && actual.valor < valor) {
			inicio = medio + 1
		} else {
			fin = medio
		}
	}
	if inicio == len(valores) || valores[inicio].clave != clave || valores[inicio].valor != valor {
		return anclaSeleccion{}, false
	}
	return valores[inicio], true
}

func buscarGrupoSeleccion(
	valores []grupoSeleccionIndexado,
	clave string,
) (grupoSeleccionIndexado, bool) {
	inicio, fin := 0, len(valores)
	for inicio < fin {
		medio := inicio + (fin-inicio)/2
		if valores[medio].clave < clave {
			inicio = medio + 1
		} else {
			fin = medio
		}
	}
	if inicio == len(valores) || valores[inicio].clave != clave {
		return grupoSeleccionIndexado{}, false
	}
	return valores[inicio], true
}

func contieneCadenaSeleccion(valores []string, buscada string) bool {
	inicio, fin := 0, len(valores)
	for inicio < fin {
		medio := inicio + (fin-inicio)/2
		if valores[medio] < buscada {
			inicio = medio + 1
		} else {
			fin = medio
		}
	}
	return inicio < len(valores) && valores[inicio] == buscada
}
