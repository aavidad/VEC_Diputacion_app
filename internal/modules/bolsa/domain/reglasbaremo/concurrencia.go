package reglasbaremo

import "vec-diputacion-granada/internal/shared/baremacion"

// ModoSolape selecciona como resolver tramos distintos que concurren en el
// tiempo dentro de un grupo. La coincidencia de un mismo tramo con varias
// reglas se gobierna por PoliticaCoincidenciaReglas, nunca por este valor.
type ModoSolape string

const (
	SolapeRechazar              ModoSolape = "rechazar"
	SolapeAcumularHastaLimite   ModoSolape = "acumular_hasta_limite"
	SolapeElegirMayorPuntuacion ModoSolape = "elegir_mayor_puntuacion"
	SolapeElegirMayorDedicacion ModoSolape = "elegir_mayor_dedicacion"
)

func (m ModoSolape) valido() bool {
	switch m {
	case SolapeRechazar, SolapeAcumularHastaLimite,
		SolapeElegirMayorPuntuacion, SolapeElegirMayorDedicacion:
		return true
	default:
		return false
	}
}

// PoliticaSolape almacena el limite de dedicacion exclusivamente cuando se
// acumulan periodos. El valor cero es invalido.
type PoliticaSolape struct {
	modo        ModoSolape
	tieneLimite bool
	limite      baremacion.FraccionJornada
}

// NuevaPoliticaSolape construye una politica que no acumula fracciones.
func NuevaPoliticaSolape(modo ModoSolape) (PoliticaSolape, error) {
	politica := PoliticaSolape{modo: modo}
	if err := politica.validar(); err != nil {
		return PoliticaSolape{}, err
	}
	return politica, nil
}

// NuevaPoliticaSolapeAcumulable fija el limite exacto de acumulacion.
func NuevaPoliticaSolapeAcumulable(limite baremacion.FraccionJornada) (PoliticaSolape, error) {
	politica := PoliticaSolape{
		modo:        SolapeAcumularHastaLimite,
		tieneLimite: true,
		limite:      limite,
	}
	if err := politica.validar(); err != nil {
		return PoliticaSolape{}, err
	}
	return politica, nil
}

func (p PoliticaSolape) Modo() ModoSolape { return p.modo }
func (p PoliticaSolape) Limite() (baremacion.FraccionJornada, bool) {
	return p.limite, p.tieneLimite
}

func (p PoliticaSolape) validar() error {
	if !p.modo.valido() {
		return nuevoError("politica_solape.modo", CodigoPoliticaIncompleta)
	}
	if p.modo == SolapeAcumularHastaLimite {
		if !p.tieneLimite || !p.limite.EsValida() {
			return nuevoError("politica_solape.limite", CodigoPoliticaIncompleta)
		}
		return nil
	}
	if p.tieneLimite || p.limite.EsValida() {
		return nuevoError("politica_solape.limite", CodigoPoliticaIncompleta)
	}
	return nil
}

// ModoCoincidenciaReglas decide que ocurre cuando un mismo tramo satisface
// simultaneamente los criterios de varias reglas del grupo.
type ModoCoincidenciaReglas string

const (
	CoincidenciaReglasRechazar              ModoCoincidenciaReglas = "rechazar"
	CoincidenciaReglasElegirPrioridad       ModoCoincidenciaReglas = "elegir_prioridad"
	CoincidenciaReglasElegirMayorPuntuacion ModoCoincidenciaReglas = "elegir_mayor_puntuacion"
	CoincidenciaReglasAcumular              ModoCoincidenciaReglas = "acumular"
)

func (m ModoCoincidenciaReglas) valido() bool {
	switch m {
	case CoincidenciaReglasRechazar, CoincidenciaReglasElegirPrioridad,
		CoincidenciaReglasElegirMayorPuntuacion, CoincidenciaReglasAcumular:
		return true
	default:
		return false
	}
}

// PoliticaCoincidenciaReglas exige una eleccion explicita y no confunde una
// coincidencia de criterios con el solape temporal de tramos distintos. En las
// elecciones, una igualdad de puntuacion se resuelve por la prioridad unica de
// regla: 1 es la maxima.
type PoliticaCoincidenciaReglas struct{ modo ModoCoincidenciaReglas }

// NuevaPoliticaCoincidenciaReglas valida un modo cerrado.
func NuevaPoliticaCoincidenciaReglas(modo ModoCoincidenciaReglas) (PoliticaCoincidenciaReglas, error) {
	politica := PoliticaCoincidenciaReglas{modo: modo}
	if !politica.modo.valido() {
		return PoliticaCoincidenciaReglas{}, nuevoError("politica_coincidencia_reglas.modo", CodigoPoliticaIncompleta)
	}
	return politica, nil
}

func (p PoliticaCoincidenciaReglas) Modo() ModoCoincidenciaReglas { return p.modo }

func (p PoliticaCoincidenciaReglas) validar() error {
	if !p.modo.valido() {
		return nuevoError("politica_coincidencia_reglas.modo", CodigoPoliticaIncompleta)
	}
	return nil
}

// ModoRepartoExceso decide que hacer con la dedicacion que supera el limite
// de una politica de solape acumulable.
type ModoRepartoExceso string

const (
	RepartoExcesoRechazar                      ModoRepartoExceso = "rechazar"
	RepartoExcesoRecortarPorPrioridad          ModoRepartoExceso = "recortar_por_prioridad"
	RepartoExcesoProporcionalExacto            ModoRepartoExceso = "repartir_proporcional_exacto"
	RepartoExcesoElegirMayorPuntuacionMarginal ModoRepartoExceso = "elegir_mayor_puntuacion_marginal"
)

func (m ModoRepartoExceso) valido() bool {
	switch m {
	case RepartoExcesoRechazar, RepartoExcesoRecortarPorPrioridad,
		RepartoExcesoProporcionalExacto, RepartoExcesoElegirMayorPuntuacionMarginal:
		return true
	default:
		return false
	}
}

// PoliticaRepartoExceso existe exclusivamente con un solape acumulable.
// Recortar por prioridad asigna capacidad por prioridad de regla; si varios
// tramos de la misma regla comparten prioridad, distribuye el remanente de
// forma proporcional exacta, nunca por orden de entrada. Elegir la mayor
// puntuacion marginal desempata por prioridad y aplica el mismo reparto exacto
// dentro de la regla elegida.
type PoliticaRepartoExceso struct {
	modo                    ModoRepartoExceso
	desempateEntreReglas    CriterioDesempateExceso
	repartoDentroMismaRegla ModoRepartoDentroRegla
}

// CriterioDesempateExceso hace visible en el canon si interviene la prioridad.
type CriterioDesempateExceso string

const (
	DesempateExcesoNoAplica              CriterioDesempateExceso = "no_aplica"
	DesempateExcesoPrioridadConcurrencia CriterioDesempateExceso = "prioridad_concurrencia"
)

// ModoRepartoDentroRegla impide usar el orden de entrada como desempate.
type ModoRepartoDentroRegla string

const (
	RepartoDentroReglaNoAplica           ModoRepartoDentroRegla = "no_aplica"
	RepartoDentroReglaProporcionalExacto ModoRepartoDentroRegla = "proporcional_exacto"
)

// NuevaPoliticaRepartoExceso valida un modo cerrado.
func NuevaPoliticaRepartoExceso(modo ModoRepartoExceso) (PoliticaRepartoExceso, error) {
	politica := PoliticaRepartoExceso{modo: modo}
	switch modo {
	case RepartoExcesoRechazar:
		politica.desempateEntreReglas = DesempateExcesoNoAplica
		politica.repartoDentroMismaRegla = RepartoDentroReglaNoAplica
	case RepartoExcesoRecortarPorPrioridad, RepartoExcesoElegirMayorPuntuacionMarginal:
		politica.desempateEntreReglas = DesempateExcesoPrioridadConcurrencia
		politica.repartoDentroMismaRegla = RepartoDentroReglaProporcionalExacto
	case RepartoExcesoProporcionalExacto:
		politica.desempateEntreReglas = DesempateExcesoNoAplica
		politica.repartoDentroMismaRegla = RepartoDentroReglaProporcionalExacto
	}
	if !politica.modo.valido() {
		return PoliticaRepartoExceso{}, nuevoError("politica_reparto_exceso.modo", CodigoPoliticaIncompleta)
	}
	return politica, nil
}

func (p PoliticaRepartoExceso) Modo() ModoRepartoExceso { return p.modo }
func (p PoliticaRepartoExceso) DesempateEntreReglas() CriterioDesempateExceso {
	return p.desempateEntreReglas
}
func (p PoliticaRepartoExceso) RepartoDentroMismaRegla() ModoRepartoDentroRegla {
	return p.repartoDentroMismaRegla
}

func (p PoliticaRepartoExceso) validar() error {
	if !p.modo.valido() {
		return nuevoError("politica_reparto_exceso.modo", CodigoPoliticaIncompleta)
	}
	esperada, err := NuevaPoliticaRepartoExceso(p.modo)
	if err != nil || p.desempateEntreReglas != esperada.desempateEntreReglas ||
		p.repartoDentroMismaRegla != esperada.repartoDentroMismaRegla {
		return nuevoError("politica_reparto_exceso.semantica", CodigoPoliticaIncompleta)
	}
	return nil
}

// GrupoConcurrenciaExperiencia gobierna coincidencias multiples entre reglas,
// incluso si pertenecen a secciones diferentes. La prioridad 1 de cada regla
// es la maxima y no se admite empate dentro del grupo. Si un mismo tramo
// coincide en grupos diferentes, V1 no inventa una prioridad entre grupos: el
// calculador debe rechazar la entrada de forma cerrada.
type GrupoConcurrenciaExperiencia struct {
	clave              string
	definicion         ReferenciaVersionada
	orden              uint32
	coincidenciaReglas PoliticaCoincidenciaReglas
	solape             PoliticaSolape
	tieneRepartoExceso bool
	repartoExceso      PoliticaRepartoExceso
}

// NuevoGrupoConcurrenciaExperiencia exige la politica de exceso exactamente
// cuando el solape acumula hasta un limite. El puntero se copia y no se retiene.
func NuevoGrupoConcurrenciaExperiencia(
	clave string,
	definicion ReferenciaVersionada,
	orden uint32,
	coincidenciaReglas PoliticaCoincidenciaReglas,
	solape PoliticaSolape,
	repartoExceso *PoliticaRepartoExceso,
) (GrupoConcurrenciaExperiencia, error) {
	grupo := GrupoConcurrenciaExperiencia{
		clave:              clave,
		definicion:         definicion,
		orden:              orden,
		coincidenciaReglas: coincidenciaReglas,
		solape:             solape,
	}
	if repartoExceso != nil {
		grupo.tieneRepartoExceso = true
		grupo.repartoExceso = *repartoExceso
	}
	if err := grupo.validar(); err != nil {
		return GrupoConcurrenciaExperiencia{}, err
	}
	return grupo, nil
}

func (g GrupoConcurrenciaExperiencia) Clave() string                    { return g.clave }
func (g GrupoConcurrenciaExperiencia) Definicion() ReferenciaVersionada { return g.definicion }
func (g GrupoConcurrenciaExperiencia) Orden() uint32                    { return g.orden }
func (g GrupoConcurrenciaExperiencia) CoincidenciaReglas() PoliticaCoincidenciaReglas {
	return g.coincidenciaReglas
}
func (g GrupoConcurrenciaExperiencia) Solape() PoliticaSolape { return g.solape }
func (g GrupoConcurrenciaExperiencia) RepartoExceso() (PoliticaRepartoExceso, bool) {
	return g.repartoExceso, g.tieneRepartoExceso
}

func (g GrupoConcurrenciaExperiencia) validar() error {
	if !claveValida(g.clave) {
		return nuevoError("grupo_concurrencia.clave", CodigoValorNoCanonico)
	}
	if err := g.definicion.validar("grupo_concurrencia.definicion"); err != nil {
		return err
	}
	if g.orden == 0 || g.orden > maximoOrden {
		return nuevoError("grupo_concurrencia.orden", CodigoFueraDeLimites)
	}
	if err := g.coincidenciaReglas.validar(); err != nil {
		return err
	}
	if err := g.solape.validar(); err != nil {
		return err
	}
	if g.solape.modo == SolapeAcumularHastaLimite {
		if !g.tieneRepartoExceso || g.repartoExceso.validar() != nil {
			return nuevoError("grupo_concurrencia.reparto_exceso", CodigoPoliticaIncompleta)
		}
		return nil
	}
	if g.tieneRepartoExceso || g.repartoExceso != (PoliticaRepartoExceso{}) {
		return nuevoError("grupo_concurrencia.reparto_exceso", CodigoPoliticaIncompleta)
	}
	return nil
}
