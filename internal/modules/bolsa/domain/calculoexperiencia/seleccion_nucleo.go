package calculoexperiencia

import "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"

// numeroEvaluacionesSeleccion hace visible la unidad del presupuesto. Una
// evaluacion es la comprobacion de un criterio de una regla candidata contra
// un tramo. La busqueda por el indice no consume el producto cartesiano.
type numeroEvaluacionesSeleccion uint64

const maximoEvaluacionesSeleccion numeroEvaluacionesSeleccion = 100_000

type presupuestoEvaluacionesSeleccion struct {
	limite numeroEvaluacionesSeleccion
	usadas numeroEvaluacionesSeleccion
}

func (p *presupuestoEvaluacionesSeleccion) consumir() error {
	if p == nil || p.usadas >= p.limite {
		return nuevoError("seleccion.evaluaciones", CodigoLimiteOperaciones)
	}
	p.usadas++
	return nil
}

// Los motivos son codigos cerrados para la futura explicacion. No contienen
// textos libres, atributos de la persona ni valores de los catalogos.
type motivoAplicacionSeleccion string

const (
	motivoAplicacionUnica     motivoAplicacionSeleccion = "coincidencia_unica"
	motivoAplicacionPrioridad motivoAplicacionSeleccion = "prioridad"
	motivoAplicacionAcumulada motivoAplicacionSeleccion = "acumulacion"
)

type motivoDescarteSeleccion string

const motivoDescartePrioridadInferior motivoDescarteSeleccion = "prioridad_inferior"

type motivoNoCoincidenciaSeleccion string

const motivoNingunaReglaCoincidente motivoNoCoincidenciaSeleccion = "ninguna_regla_coincidente"

type codigoBloqueoSeleccion string

const (
	bloqueoCatalogoIncompatible  codigoBloqueoSeleccion = "catalogo_incompatible"
	bloqueoGruposDistintos       codigoBloqueoSeleccion = "reglas_en_grupos_distintos"
	bloqueoCoincidenciaRechazada codigoBloqueoSeleccion = "coincidencia_reglas_rechazada"
)

// aplicacionSeleccion identifica el periodo logico que continuara hacia las
// fases temporales. Guarda solo referencias opacas y claves gobernadas.
type aplicacionSeleccion struct {
	tramo        reglasbaremo.ReferenciaVersionada
	reglaClave   string
	grupoClave   string
	seccionClave string
	prioridad    uint32
	razon        motivoAplicacionSeleccion
}

type descarteSeleccion struct {
	tramo             reglasbaremo.ReferenciaVersionada
	reglaClave        string
	grupoClave        string
	reglaSeleccionada string
	razon             motivoDescarteSeleccion
}

// noCoincidenciaSeleccion se registra por tramo, no por cada regla descartada
// por el indice. Generar el producto tramo-regla solo para explicar ceros
// anularia la proteccion de complejidad de esta fase.
type noCoincidenciaSeleccion struct {
	tramo reglasbaremo.ReferenciaVersionada
	razon motivoNoCoincidenciaSeleccion
}

type bloqueoSeleccion struct {
	codigo         codigoBloqueoSeleccion
	tramo          reglasbaremo.ReferenciaVersionada
	claveGobernada string
	reglas         []string
}

func (b bloqueoSeleccion) clonar() bloqueoSeleccion {
	b.reglas = append([]string(nil), b.reglas...)
	return b
}

// seleccionExperiencia es un resultado interno y canonico. Si contiene algun
// bloqueo, ninguna aplicacion puede avanzar a puntuacion. Se conservan las
// aplicaciones de otros tramos solo para que el futuro resultado explique el
// expediente de forma completa; el consumidor debe comprobar bloqueada antes
// de iniciar cualquier fase posterior.
type seleccionExperiencia struct {
	aplicaciones    []aplicacionSeleccion
	descartes       []descarteSeleccion
	noCoincidencias []noCoincidenciaSeleccion
	bloqueos        []bloqueoSeleccion
	evaluaciones    numeroEvaluacionesSeleccion
}

func (s seleccionExperiencia) bloqueada() bool { return len(s.bloqueos) > 0 }

func (s seleccionExperiencia) aplicacionesCopia() []aplicacionSeleccion {
	return append([]aplicacionSeleccion(nil), s.aplicaciones...)
}

func (s seleccionExperiencia) descartesCopia() []descarteSeleccion {
	return append([]descarteSeleccion(nil), s.descartes...)
}

func (s seleccionExperiencia) noCoincidenciasCopia() []noCoincidenciaSeleccion {
	return append([]noCoincidenciaSeleccion(nil), s.noCoincidencias...)
}

func (s seleccionExperiencia) bloqueosCopia() []bloqueoSeleccion {
	resultado := make([]bloqueoSeleccion, len(s.bloqueos))
	for indice := range s.bloqueos {
		resultado[indice] = s.bloqueos[indice].clonar()
	}
	return resultado
}

// seleccionarAplicaciones valida de nuevo sus dos instantaneas y aplica el
// presupuesto cerrado de produccion. Los tests de limite usan la variante
// interna para probar la misma ruta con una cota pequena.
func seleccionarAplicaciones(
	plan PlanExperiencia,
	entrada EntradaExperiencia,
) (seleccionExperiencia, error) {
	return seleccionarAplicacionesConLimite(
		plan,
		entrada,
		maximoEvaluacionesSeleccion,
	)
}

func seleccionarAplicacionesConLimite(
	plan PlanExperiencia,
	entrada EntradaExperiencia,
	limite numeroEvaluacionesSeleccion,
) (seleccionExperiencia, error) {
	if err := plan.Validar(); err != nil {
		return seleccionExperiencia{}, err
	}
	if err := entrada.Validar(); err != nil {
		return seleccionExperiencia{}, err
	}

	indice, err := construirIndiceSeleccion(plan)
	if err != nil {
		return seleccionExperiencia{}, err
	}
	presupuesto := presupuestoEvaluacionesSeleccion{limite: limite}
	resultado := seleccionExperiencia{}
	for _, tramo := range entrada.Tramos() {
		if err := seleccionarTramo(indice, tramo, &presupuesto, &resultado); err != nil {
			return seleccionExperiencia{}, err
		}
	}
	resultado.evaluaciones = presupuesto.usadas
	ordenarSeleccion(&resultado)
	return resultado, nil
}

func seleccionarTramo(
	indice indiceSeleccion,
	tramo TramoExperiencia,
	presupuesto *presupuestoEvaluacionesSeleccion,
	resultado *seleccionExperiencia,
) error {
	atributos := tramo.Atributos()
	porClave := make(map[string]AtributoCatalogado, len(atributos))
	incompatible := false
	for _, atributo := range atributos {
		porClave[atributo.Clave()] = atributo
		gobernada, existe := buscarClaveGobernadaSeleccion(
			indice.clavesGobernadas,
			atributo.Clave(),
		)
		if existe && !referenciasPlanIguales(gobernada.catalogo, atributo.Catalogo()) {
			incompatible = true
			resultado.bloqueos = append(resultado.bloqueos, bloqueoSeleccion{
				codigo:         bloqueoCatalogoIncompatible,
				tramo:          tramo.Referencia(),
				claveGobernada: atributo.Clave(),
			})
		}
	}
	if incompatible {
		return nil
	}

	candidatas := reglasCandidatas(indice, atributos)
	coincidentes := make([]int, 0, len(candidatas))
	for _, posicion := range candidatas {
		coincide, err := coincideRegla(indice.reglas[posicion], porClave, presupuesto)
		if err != nil {
			return err
		}
		if coincide {
			coincidentes = append(coincidentes, posicion)
		}
	}
	if len(coincidentes) == 0 {
		resultado.noCoincidencias = append(resultado.noCoincidencias, noCoincidenciaSeleccion{
			tramo: tramo.Referencia(),
			razon: motivoNingunaReglaCoincidente,
		})
		return nil
	}
	return resolverCoincidencia(indice, tramo.Referencia(), coincidentes, resultado)
}

func reglasCandidatas(indice indiceSeleccion, atributos []AtributoCatalogado) []int {
	candidatas := make([]int, 0)
	for _, atributo := range atributos {
		ancla, existe := buscarAnclaSeleccion(
			indice.anclas,
			atributo.Clave(),
			atributo.Valor(),
		)
		if !existe {
			continue
		}
		candidatas = append(candidatas, ancla.posiciones...)
	}
	ordenarEnterosSeleccion(candidatas)
	// Un tramo no puede repetir una clave y cada regla solo tiene un ancla,
	// pero deduplicar aqui mantiene cerrado el indice ante cambios internos.
	escritura := 0
	for _, posicion := range candidatas {
		if escritura > 0 && candidatas[escritura-1] == posicion {
			continue
		}
		candidatas[escritura] = posicion
		escritura++
	}
	return candidatas[:escritura]
}

func coincideRegla(
	regla reglaSeleccionIndexada,
	atributos map[string]AtributoCatalogado,
	presupuesto *presupuestoEvaluacionesSeleccion,
) (bool, error) {
	for _, criterio := range regla.criterios {
		if err := presupuesto.consumir(); err != nil {
			return false, err
		}
		atributo, existe := atributos[criterio.clave]
		if !existe {
			return false, nil
		}
		if !contieneCadenaSeleccion(criterio.valores, atributo.Valor()) {
			return false, nil
		}
	}
	return true, nil
}

func resolverCoincidencia(
	indice indiceSeleccion,
	tramo reglasbaremo.ReferenciaVersionada,
	coincidentes []int,
	resultado *seleccionExperiencia,
) error {
	grupos := make(map[string]struct{})
	for _, posicion := range coincidentes {
		grupos[indice.reglas[posicion].grupo] = struct{}{}
	}
	if len(grupos) > 1 {
		resultado.bloqueos = append(resultado.bloqueos, bloqueoSeleccion{
			codigo: bloqueoGruposDistintos,
			tramo:  tramo,
			reglas: clavesReglasCoincidentes(indice, coincidentes),
		})
		return nil
	}

	grupo := indice.reglas[coincidentes[0]].grupo
	grupoIndexado, existe := buscarGrupoSeleccion(indice.grupos, grupo)
	if !existe {
		return errorPlanInvalido()
	}
	modo := grupoIndexado.modo
	if len(coincidentes) == 1 {
		anadirAplicacion(resultado, tramo, indice.reglas[coincidentes[0]], motivoAplicacionUnica)
		return nil
	}

	switch modo {
	case reglasbaremo.CoincidenciaReglasRechazar:
		resultado.bloqueos = append(resultado.bloqueos, bloqueoSeleccion{
			codigo: bloqueoCoincidenciaRechazada,
			tramo:  tramo,
			reglas: clavesReglasCoincidentes(indice, coincidentes),
		})
	case reglasbaremo.CoincidenciaReglasElegirPrioridad:
		ordenadas := append([]int(nil), coincidentes...)
		ordenarReglasPorPrioridadSeleccion(ordenadas, indice.reglas)
		seleccionada := indice.reglas[ordenadas[0]]
		anadirAplicacion(resultado, tramo, seleccionada, motivoAplicacionPrioridad)
		for _, posicion := range ordenadas[1:] {
			descartada := indice.reglas[posicion]
			resultado.descartes = append(resultado.descartes, descarteSeleccion{
				tramo:             tramo,
				reglaClave:        descartada.clave,
				grupoClave:        descartada.grupo,
				reglaSeleccionada: seleccionada.clave,
				razon:             motivoDescartePrioridadInferior,
			})
		}
	case reglasbaremo.CoincidenciaReglasAcumular:
		for _, posicion := range coincidentes {
			anadirAplicacion(
				resultado,
				tramo,
				indice.reglas[posicion],
				motivoAplicacionAcumulada,
			)
		}
	default:
		return errorPlanInvalido()
	}
	return nil
}

func anadirAplicacion(
	resultado *seleccionExperiencia,
	tramo reglasbaremo.ReferenciaVersionada,
	regla reglaSeleccionIndexada,
	motivo motivoAplicacionSeleccion,
) {
	resultado.aplicaciones = append(resultado.aplicaciones, aplicacionSeleccion{
		tramo:        tramo,
		reglaClave:   regla.clave,
		grupoClave:   regla.grupo,
		seccionClave: regla.seccion,
		prioridad:    regla.prioridad,
		razon:        motivo,
	})
}

func clavesReglasCoincidentes(indice indiceSeleccion, posiciones []int) []string {
	claves := make([]string, len(posiciones))
	for posicion, indiceRegla := range posiciones {
		claves[posicion] = indice.reglas[indiceRegla].clave
	}
	ordenarCadenasSeleccion(claves)
	return claves
}
