package calculoexperiencia

import (
	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

type numeroAplicacionesTemporales uint64
type numeroEventosTemporales uint64

const (
	maximoAplicacionesTemporales numeroAplicacionesTemporales = 100_000
	maximoEventosTemporales      numeroEventosTemporales      = 200_000
)

type limitesAplicacionesTemporales struct {
	aplicaciones numeroAplicacionesTemporales
	eventos      numeroEventosTemporales
}

type presupuestoAplicacionesTemporales struct {
	limites                limitesAplicacionesTemporales
	aplicacionesConsumidas numeroAplicacionesTemporales
	eventosConsumidos      numeroEventosTemporales
}

func (p *presupuestoAplicacionesTemporales) consumirAplicacion() error {
	if p == nil || p.aplicacionesConsumidas >= p.limites.aplicaciones {
		return nuevoError(
			"aplicaciones_temporales.aplicaciones",
			CodigoLimiteAplicacionesTemporales,
		)
	}
	p.aplicacionesConsumidas++
	return nil
}

func (p *presupuestoAplicacionesTemporales) consumirEvento() error {
	if p == nil || p.eventosConsumidos >= p.limites.eventos {
		return nuevoError(
			"aplicaciones_temporales.eventos",
			CodigoLimiteEventosTemporales,
		)
	}
	p.eventosConsumidos++
	return nil
}

type razonExclusionTemporal string

const (
	exclusionTemporalIntervaloVacio razonExclusionTemporal = "intervalo_vacio"
	exclusionTemporalFueraDeCorte   razonExclusionTemporal = "fuera_de_corte"
)

type codigoBloqueoTemporal string

const bloqueoTemporalSolapeRechazado codigoBloqueoTemporal = "solape_rechazado"

// aplicacionTemporal conserva los dos hechos gobernados exactos que necesitara
// la fase aritmetica. El intervalo siempre es semiabierto y dias se obtiene por
// ordinales civiles, nunca recorriendo el calendario.
type aplicacionTemporal struct {
	tramo     TramoExperiencia
	regla     reglasbaremo.ReglaExperiencia
	intervalo baremacion.IntervaloCivil
	dias      int64
	razon     motivoAplicacionSeleccion
}

func (a aplicacionTemporal) clonar() aplicacionTemporal {
	a.tramo = a.tramo.clonar()
	return a
}

type exclusionAplicacionTemporal struct {
	tramo      reglasbaremo.ReferenciaVersionada
	reglaClave string
	grupoClave string
	razon      razonExclusionTemporal
}

// bloqueoAplicacionesTemporales contiene exclusivamente claves gobernadas y
// referencias opacas. Deliberadamente no conserva servicioRef, atributos ni
// ningun texto libre del expediente.
type bloqueoAplicacionesTemporales struct {
	codigo       codigoBloqueoTemporal
	grupoClave   string
	tramoPrimero reglasbaremo.ReferenciaVersionada
	tramoSegundo reglasbaremo.ReferenciaVersionada
}

// resultadoAplicacionesTemporales es canonico. Un bloqueo impide que sus
// aplicaciones avancen a puntuacion, aunque se conserven para explicar todo el
// expediente. Como maximo se emite un bloqueo por grupo.
type resultadoAplicacionesTemporales struct {
	aplicaciones           []aplicacionTemporal
	exclusiones            []exclusionAplicacionTemporal
	bloqueos               []bloqueoAplicacionesTemporales
	aplicacionesProcesadas numeroAplicacionesTemporales
	eventosProcesados      numeroEventosTemporales
}

func (r resultadoAplicacionesTemporales) bloqueada() bool {
	return len(r.bloqueos) > 0
}

func (r resultadoAplicacionesTemporales) aplicacionesCopia() []aplicacionTemporal {
	resultado := make([]aplicacionTemporal, len(r.aplicaciones))
	for indice := range r.aplicaciones {
		resultado[indice] = r.aplicaciones[indice].clonar()
	}
	return resultado
}

func (r resultadoAplicacionesTemporales) exclusionesCopia() []exclusionAplicacionTemporal {
	return append([]exclusionAplicacionTemporal(nil), r.exclusiones...)
}

func (r resultadoAplicacionesTemporales) bloqueosCopia() []bloqueoAplicacionesTemporales {
	return append([]bloqueoAplicacionesTemporales(nil), r.bloqueos...)
}

func resolverAplicacionesTemporales(
	plan PlanExperiencia,
	entrada EntradaExperiencia,
	seleccion seleccionExperiencia,
) (resultadoAplicacionesTemporales, error) {
	return resolverAplicacionesTemporalesConLimites(
		plan,
		entrada,
		seleccion,
		limitesAplicacionesTemporales{
			aplicaciones: maximoAplicacionesTemporales,
			eventos:      maximoEventosTemporales,
		},
	)
}

func resolverAplicacionesTemporalesConLimites(
	plan PlanExperiencia,
	entrada EntradaExperiencia,
	seleccion seleccionExperiencia,
	limites limitesAplicacionesTemporales,
) (resultadoAplicacionesTemporales, error) {
	if seleccion.bloqueada() {
		return resultadoAplicacionesTemporales{}, nuevoError(
			"aplicaciones_temporales.seleccion",
			CodigoSeleccionTemporalBloqueada,
		)
	}
	if err := plan.Validar(); err != nil {
		return resultadoAplicacionesTemporales{}, err
	}
	if err := entrada.Validar(); err != nil {
		return resultadoAplicacionesTemporales{}, err
	}

	tramosPorReferencia := make(map[string]TramoExperiencia)
	for _, tramo := range entrada.Tramos() {
		tramosPorReferencia[tramo.Referencia().Referencia()] = tramo
	}
	reglasPorClave := make(map[string]reglasbaremo.ReglaExperiencia)
	for _, regla := range plan.Reglas() {
		reglasPorClave[regla.Clave()] = regla
	}
	gruposPorClave := make(map[string]reglasbaremo.GrupoConcurrenciaExperiencia)
	for _, grupo := range plan.GruposConcurrencia() {
		gruposPorClave[grupo.Clave()] = grupo
	}

	presupuesto := presupuestoAplicacionesTemporales{limites: limites}
	resultado := resultadoAplicacionesTemporales{}
	aplicacionesVistas := make(map[claveAplicacionTemporal]struct{})
	for _, seleccionada := range seleccion.aplicacionesCopia() {
		if err := presupuesto.consumirAplicacion(); err != nil {
			return resultadoAplicacionesTemporales{}, err
		}
		tramo, regla, err := resolverAplicacionTemporal(
			seleccionada,
			tramosPorReferencia,
			reglasPorClave,
			gruposPorClave,
		)
		if err != nil {
			return resultadoAplicacionesTemporales{}, err
		}
		clave := claveAplicacionTemporal{
			tramoReferencia: tramo.Referencia().Referencia(),
			reglaClave:      regla.Clave(),
		}
		if _, existe := aplicacionesVistas[clave]; existe {
			return resultadoAplicacionesTemporales{}, errorSeleccionTemporalInvalida()
		}
		aplicacionesVistas[clave] = struct{}{}

		intervalo, efectivo, err := normalizarPeriodoEfectivo(
			tramo.Periodo(),
			plan.FechaCorte(),
			regla.UnidadTemporal().ExtremoFinal(),
		)
		if err != nil {
			return resultadoAplicacionesTemporales{}, err
		}
		if !efectivo {
			resultado.exclusiones = append(
				resultado.exclusiones,
				nuevaExclusionAplicacionTemporal(plan.FechaCorte(), tramo, regla),
			)
			continue
		}
		dias, err := intervalo.NumeroDias()
		if err != nil {
			return resultadoAplicacionesTemporales{}, errorSeleccionTemporalInvalida()
		}
		resultado.aplicaciones = append(resultado.aplicaciones, aplicacionTemporal{
			tramo:     tramo,
			regla:     regla,
			intervalo: intervalo,
			dias:      dias,
			razon:     seleccionada.razon,
		})
	}

	if err := detectarSolapesTemporales(
		resultado.aplicaciones,
		gruposPorClave,
		&presupuesto,
		&resultado,
	); err != nil {
		return resultadoAplicacionesTemporales{}, err
	}
	ordenarResultadoAplicacionesTemporales(&resultado)
	resultado.aplicacionesProcesadas = presupuesto.aplicacionesConsumidas
	resultado.eventosProcesados = presupuesto.eventosConsumidos
	return resultado, nil
}

type claveAplicacionTemporal struct {
	tramoReferencia string
	reglaClave      string
}

func resolverAplicacionTemporal(
	aplicacion aplicacionSeleccion,
	tramos map[string]TramoExperiencia,
	reglas map[string]reglasbaremo.ReglaExperiencia,
	grupos map[string]reglasbaremo.GrupoConcurrenciaExperiencia,
) (TramoExperiencia, reglasbaremo.ReglaExperiencia, error) {
	tramo, existe := tramos[aplicacion.tramo.Referencia()]
	if !existe || !referenciasTemporalesIguales(tramo.Referencia(), aplicacion.tramo) {
		return TramoExperiencia{}, reglasbaremo.ReglaExperiencia{},
			errorSeleccionTemporalInvalida()
	}
	regla, existe := reglas[aplicacion.reglaClave]
	if !existe || regla.GrupoConcurrenciaClave() != aplicacion.grupoClave ||
		regla.SeccionClave() != aplicacion.seccionClave ||
		regla.PrioridadConcurrencia() != aplicacion.prioridad ||
		!razonAplicacionTemporalValida(aplicacion.razon) {
		return TramoExperiencia{}, reglasbaremo.ReglaExperiencia{},
			errorSeleccionTemporalInvalida()
	}
	grupo, existe := grupos[aplicacion.grupoClave]
	if !existe || grupo.Solape().Modo() != reglasbaremo.SolapeRechazar {
		return TramoExperiencia{}, reglasbaremo.ReglaExperiencia{},
			errorSeleccionTemporalInvalida()
	}
	return tramo, regla, nil
}

func nuevaExclusionAplicacionTemporal(
	corte baremacion.FechaCivil,
	tramo TramoExperiencia,
	regla reglasbaremo.ReglaExperiencia,
) exclusionAplicacionTemporal {
	razon := exclusionTemporalIntervaloVacio
	comparacion, err := tramo.Periodo().Desde().Comparar(corte)
	if err == nil && comparacion > 0 {
		razon = exclusionTemporalFueraDeCorte
	}
	return exclusionAplicacionTemporal{
		tramo:      tramo.Referencia(),
		reglaClave: regla.Clave(),
		grupoClave: regla.GrupoConcurrenciaClave(),
		razon:      razon,
	}
}

func razonAplicacionTemporalValida(razon motivoAplicacionSeleccion) bool {
	switch razon {
	case motivoAplicacionUnica, motivoAplicacionPrioridad, motivoAplicacionAcumulada:
		return true
	default:
		return false
	}
}

func referenciasTemporalesIguales(
	izquierda reglasbaremo.ReferenciaVersionada,
	derecha reglasbaremo.ReferenciaVersionada,
) bool {
	return izquierda.Referencia() == derecha.Referencia() &&
		izquierda.Version() == derecha.Version() &&
		izquierda.HuellaSHA256() == derecha.HuellaSHA256()
}

func errorSeleccionTemporalInvalida() error {
	return nuevoError(
		"aplicaciones_temporales.seleccion",
		CodigoSeleccionTemporalInvalida,
	)
}
