package calculoexperiencia

import (
	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

// PlanExperiencia es la instantanea cerrada que consumira el calculador V1.
// Fija la version y huella exactas de las reglas y conserva solo el material
// minimo necesario para rederivar ese vinculo, nunca referencias a las
// colecciones recibidas.
//
// El pipeline puntuable es unico: unidades y restos, tope de unidades,
// coeficiente y redondeo, tope de puntos de regla, suma y tope de seccion. Las
// comprobaciones de elegibilidad, coincidencia y ausencia de solapes ocurren
// antes de ese pipeline y nunca cambian su orden.
type PlanExperiencia struct {
	identidad           reglasbaremo.IdentidadConjuntoReglasBaremo
	bases               reglasbaremo.ReferenciaVersionada
	conjunto            reglasbaremo.ReferenciaVersionada
	fechaCorteInclusiva baremacion.FechaCivil
	secciones           []reglasbaremo.SeccionBaremo
	gruposConcurrencia  []reglasbaremo.GrupoConcurrenciaExperiencia
	reglas              []reglasbaremo.ReglaExperiencia
}

// Compilar convierte un conjunto valido en un plan ejecutable por V1. Una
// politica que el modelo puede representar pero el calculador aun no gobierna
// se rechaza expresamente; nunca se aproxima ni se sustituye por un valor por
// defecto. Deliberadamente no comprueba el estado de gobierno: tambien compila
// borradores para su simulacion y conformidad. El caso de uso administrativo
// oficial debe aportar una FuenteExacta atestada en estado activo.
func Compilar(conjunto reglasbaremo.ConjuntoReglasBaremo) (PlanExperiencia, error) {
	if err := conjunto.Validar(); err != nil {
		return PlanExperiencia{}, nuevoError("conjunto", CodigoCompilacionConjuntoInvalido)
	}

	secciones := conjunto.Secciones()
	for _, seccion := range secciones {
		if seccion.PuntosMinimos().Micropuntos() > 0 {
			return PlanExperiencia{}, nuevoError(
				"seccion.puntos_minimos",
				CodigoMinimoSeccionNoSoportado,
			)
		}
	}

	reglas := conjunto.ReglasExperiencia()
	for _, regla := range reglas {
		if err := validarReglaCompilableV1(regla); err != nil {
			return PlanExperiencia{}, err
		}
	}
	if err := validarCatalogosCriterioCompatibles(reglas); err != nil {
		return PlanExperiencia{}, err
	}
	grupos := conjunto.GruposConcurrenciaExperiencia()
	for _, grupo := range grupos {
		if err := validarGrupoCompilableV1(grupo); err != nil {
			return PlanExperiencia{}, err
		}
	}

	referencia, err := conjunto.ReferenciaVersionada()
	if err != nil {
		return PlanExperiencia{}, nuevoError("conjunto", CodigoCompilacionConjuntoInvalido)
	}
	plan := PlanExperiencia{
		identidad:           conjunto.Identidad(),
		bases:               conjunto.Bases(),
		conjunto:            referencia,
		fechaCorteInclusiva: conjunto.FechaCorte(),
		secciones:           append([]reglasbaremo.SeccionBaremo(nil), secciones...),
		gruposConcurrencia:  append([]reglasbaremo.GrupoConcurrenciaExperiencia(nil), grupos...),
		reglas:              append([]reglasbaremo.ReglaExperiencia(nil), reglas...),
	}
	if err := plan.Validar(); err != nil {
		return PlanExperiencia{}, err
	}
	return plan, nil
}

func validarReglaCompilableV1(regla reglasbaremo.ReglaExperiencia) error {
	switch regla.Jornada().Modo() {
	case reglasbaremo.JornadaProporcional,
		reglasbaremo.JornadaIntegra,
		reglasbaremo.JornadaIntegraDesdeUmbral,
		reglasbaremo.JornadaProtegidaIntegra:
	case reglasbaremo.JornadaPorHoras:
		return nuevoError("regla.jornada", CodigoJornadaNoSoportada)
	default:
		return nuevoError("regla.jornada", CodigoJornadaNoSoportada)
	}

	if regla.UnidadTemporal().UnidadBase() != reglasbaremo.UnidadTemporalDia {
		return nuevoError("regla.unidad_temporal.unidad_base", CodigoUnidadBaseNoSoportada)
	}

	momento := regla.Redondeo().Momento()
	switch momento {
	case reglasbaremo.RedondearPorPeriodo, reglasbaremo.RedondearPorRegla:
	case reglasbaremo.RedondearPorSeccion, reglasbaremo.RedondearEnTotal:
		return nuevoError("regla.redondeo.momento", CodigoRedondeoNoSoportado)
	default:
		return nuevoError("regla.redondeo.momento", CodigoRedondeoNoSoportado)
	}

	semantica, err := semanticaRestosCompilableV1(regla.Restos().Modo())
	if err != nil {
		return err
	}
	if semantica.frontera == fronteraRestosReglaV1 &&
		momento == reglasbaremo.RedondearPorPeriodo {
		return nuevoError("regla.restos_redondeo", CodigoRestosRedondeoNoSoportados)
	}
	if regla.MaximoUnidades().EstaLimitado() &&
		momento == reglasbaremo.RedondearPorPeriodo {
		return nuevoError("regla.maximo_unidades", CodigoTopeUnidadesNoSoportado)
	}
	return nil
}

// validarCatalogosCriterioCompatibles impide que una misma clave tenga dos
// significados dentro de una instantanea. La identidad del catalogo comprende
// conjuntamente referencia, version y huella; no basta con que coincida una de
// ellas.
func validarCatalogosCriterioCompatibles(reglas []reglasbaremo.ReglaExperiencia) error {
	catalogos := make(map[string]reglasbaremo.ReferenciaVersionada)
	for _, regla := range reglas {
		for _, criterio := range regla.Criterios() {
			catalogo, existe := catalogos[criterio.Clave()]
			if existe && !referenciasPlanIguales(catalogo, criterio.Catalogo()) {
				return nuevoError(
					"regla.criterios.catalogo",
					CodigoCatalogoCriterioIncompatible,
				)
			}
			catalogos[criterio.Clave()] = criterio.Catalogo()
		}
	}
	return nil
}

// La semantica V1 de restos tiene dos ejes cerrados. La frontera determina
// donde se agrupan las fracciones temporales y el tratamiento si sobreviven:
//
//   - conservar exactos: sin descarte; cada contribucion permanece racional;
//   - acumular por regla: agrupa en la regla y conserva tambien el resto final;
//   - descartar por periodo: trunca cada periodo antes de agregarlo;
//   - descartar por regla: agrupa primero y trunca una sola vez en la regla.
//
// Una frontera de regla no puede convivir con redondeo por periodo porque
// obligaria a ejecutar primero dos operaciones incompatibles. No se elige un
// orden implicito.
type fronteraRestosV1 uint8

const (
	fronteraRestosExactaV1 fronteraRestosV1 = iota + 1
	fronteraRestosPeriodoV1
	fronteraRestosReglaV1
)

type tratamientoRestosV1 uint8

const (
	tratamientoRestosConservarV1 tratamientoRestosV1 = iota + 1
	tratamientoRestosDescartarV1
)

type semanticaRestosV1 struct {
	frontera    fronteraRestosV1
	tratamiento tratamientoRestosV1
}

func semanticaRestosCompilableV1(modo reglasbaremo.ModoRestos) (semanticaRestosV1, error) {
	switch modo {
	case reglasbaremo.RestosConservarExactos:
		return semanticaRestosV1{fronteraRestosExactaV1, tratamientoRestosConservarV1}, nil
	case reglasbaremo.RestosAcumularPorRegla:
		return semanticaRestosV1{fronteraRestosReglaV1, tratamientoRestosConservarV1}, nil
	case reglasbaremo.RestosDescartarPorPeriodo:
		return semanticaRestosV1{fronteraRestosPeriodoV1, tratamientoRestosDescartarV1}, nil
	case reglasbaremo.RestosDescartarPorRegla:
		return semanticaRestosV1{fronteraRestosReglaV1, tratamientoRestosDescartarV1}, nil
	default:
		return semanticaRestosV1{}, nuevoError("regla.restos", CodigoRestosRedondeoNoSoportados)
	}
}

func validarGrupoCompilableV1(grupo reglasbaremo.GrupoConcurrenciaExperiencia) error {
	switch grupo.CoincidenciaReglas().Modo() {
	case reglasbaremo.CoincidenciaReglasRechazar,
		reglasbaremo.CoincidenciaReglasElegirPrioridad,
		reglasbaremo.CoincidenciaReglasAcumular:
	case reglasbaremo.CoincidenciaReglasElegirMayorPuntuacion:
		return nuevoError("grupo.coincidencia_reglas", CodigoCoincidenciaNoSoportada)
	default:
		return nuevoError("grupo.coincidencia_reglas", CodigoCoincidenciaNoSoportada)
	}
	if grupo.Solape().Modo() != reglasbaremo.SolapeRechazar {
		return nuevoError("grupo.solape", CodigoSolapeNoSoportado)
	}
	return nil
}

// Conjunto devuelve la referencia, version y huella exactas fijadas al
// compilar. No significa nunca "la version vigente".
func (p PlanExperiencia) Conjunto() reglasbaremo.ReferenciaVersionada {
	return p.conjunto
}

// FechaCorte devuelve el ultimo dia civil incluido por el conjunto.
func (p PlanExperiencia) FechaCorte() baremacion.FechaCivil {
	return p.fechaCorteInclusiva
}

// Secciones devuelve una copia en el orden canonico del conjunto.
func (p PlanExperiencia) Secciones() []reglasbaremo.SeccionBaremo {
	return append([]reglasbaremo.SeccionBaremo(nil), p.secciones...)
}

// GruposConcurrencia devuelve una copia en el orden canonico del conjunto.
func (p PlanExperiencia) GruposConcurrencia() []reglasbaremo.GrupoConcurrenciaExperiencia {
	return append([]reglasbaremo.GrupoConcurrenciaExperiencia(nil), p.gruposConcurrencia...)
}

// Reglas devuelve una copia ordenada. Los criterios de cada regla son tambien
// inmutables y sus propios accesores devuelven copias defensivas.
func (p PlanExperiencia) Reglas() []reglasbaremo.ReglaExperiencia {
	return append([]reglasbaremo.ReglaExperiencia(nil), p.reglas...)
}

// Validar vuelve a comprobar que el plan conserva las invariantes cerradas de
// compilacion. Su valor cero es invalido.
func (p PlanExperiencia) Validar() error {
	if !referenciaPlanValida(p.conjunto) || !p.fechaCorteInclusiva.EsValida() {
		return errorPlanInvalido()
	}
	if _, err := p.fechaCorteInclusiva.Siguiente(); err != nil {
		return errorPlanInvalido()
	}
	if len(p.secciones) == 0 || len(p.gruposConcurrencia) == 0 || len(p.reglas) == 0 {
		return errorPlanInvalido()
	}
	if err := validarCatalogosCriterioCompatibles(p.reglas); err != nil {
		return err
	}
	// Reconstruir el agregado vuelve a aplicar limites de volumen, unicidad de
	// referencias, relaciones regla/seccion/grupo y todos los topes cruzados. Su
	// huella debe coincidir con la identidad exacta fijada al compilar.
	rederivado, err := reglasbaremo.NuevoConjuntoReglasBaremo(
		p.identidad,
		p.bases,
		p.fechaCorteInclusiva,
		p.secciones,
		p.gruposConcurrencia,
		p.reglas,
	)
	if err != nil {
		return errorPlanInvalido()
	}
	referenciaRederivada, err := rederivado.ReferenciaVersionada()
	if err != nil || !referenciasPlanIguales(p.conjunto, referenciaRederivada) {
		return errorPlanInvalido()
	}

	ordenSecciones := make(map[string]uint32, len(p.secciones))
	for indice, seccion := range p.secciones {
		if seccion.PuntosMinimos().Micropuntos() != 0 ||
			!seccionPlanValida(seccion) ||
			(indice > 0 && p.secciones[indice-1].Orden() >= seccion.Orden()) {
			return errorPlanInvalido()
		}
		if _, existe := ordenSecciones[seccion.Clave()]; existe {
			return errorPlanInvalido()
		}
		ordenSecciones[seccion.Clave()] = seccion.Orden()
	}

	grupos := make(map[string]struct{}, len(p.gruposConcurrencia))
	for indice, grupo := range p.gruposConcurrencia {
		if !grupoPlanValido(grupo) ||
			validarGrupoCompilableV1(grupo) != nil ||
			(indice > 0 && p.gruposConcurrencia[indice-1].Orden() >= grupo.Orden()) {
			return errorPlanInvalido()
		}
		if _, existe := grupos[grupo.Clave()]; existe {
			return errorPlanInvalido()
		}
		grupos[grupo.Clave()] = struct{}{}
	}

	clavesRegla := make(map[string]struct{}, len(p.reglas))
	prioridades := make(map[string]map[uint32]struct{}, len(grupos))
	seccionesUsadas := make(map[string]struct{}, len(ordenSecciones))
	gruposUsados := make(map[string]struct{}, len(grupos))
	for indice, regla := range p.reglas {
		if !reglaPlanValida(regla) || validarReglaCompilableV1(regla) != nil {
			return errorPlanInvalido()
		}
		ordenSeccion, existe := ordenSecciones[regla.SeccionClave()]
		if !existe {
			return errorPlanInvalido()
		}
		if _, existe := grupos[regla.GrupoConcurrenciaClave()]; !existe {
			return errorPlanInvalido()
		}
		if _, existe := clavesRegla[regla.Clave()]; existe {
			return errorPlanInvalido()
		}
		if indice > 0 {
			anterior := p.reglas[indice-1]
			ordenAnterior := ordenSecciones[anterior.SeccionClave()]
			if ordenAnterior > ordenSeccion ||
				(ordenAnterior == ordenSeccion && anterior.Orden() >= regla.Orden()) {
				return errorPlanInvalido()
			}
		}
		porGrupo := prioridades[regla.GrupoConcurrenciaClave()]
		if porGrupo == nil {
			porGrupo = make(map[uint32]struct{})
			prioridades[regla.GrupoConcurrenciaClave()] = porGrupo
		}
		if _, existe := porGrupo[regla.PrioridadConcurrencia()]; existe {
			return errorPlanInvalido()
		}
		porGrupo[regla.PrioridadConcurrencia()] = struct{}{}
		clavesRegla[regla.Clave()] = struct{}{}
		seccionesUsadas[regla.SeccionClave()] = struct{}{}
		gruposUsados[regla.GrupoConcurrenciaClave()] = struct{}{}
	}
	if len(seccionesUsadas) != len(ordenSecciones) || len(gruposUsados) != len(grupos) {
		return errorPlanInvalido()
	}
	return nil
}

func referenciaPlanValida(referencia reglasbaremo.ReferenciaVersionada) bool {
	_, err := reglasbaremo.NuevaReferenciaVersionada(
		referencia.Referencia(), referencia.Version(), referencia.HuellaSHA256(),
	)
	return err == nil
}

func referenciasPlanIguales(
	izquierda reglasbaremo.ReferenciaVersionada,
	derecha reglasbaremo.ReferenciaVersionada,
) bool {
	return izquierda.Referencia() == derecha.Referencia() &&
		izquierda.Version() == derecha.Version() &&
		izquierda.HuellaSHA256() == derecha.HuellaSHA256()
}

func seccionPlanValida(seccion reglasbaremo.SeccionBaremo) bool {
	_, err := reglasbaremo.NuevaSeccionBaremo(
		seccion.Clave(), seccion.Definicion(), seccion.Orden(),
		seccion.PuntosMinimos(), seccion.PuntosMaximos(),
	)
	return err == nil
}

func grupoPlanValido(grupo reglasbaremo.GrupoConcurrenciaExperiencia) bool {
	reparto, tieneReparto := grupo.RepartoExceso()
	var punteroReparto *reglasbaremo.PoliticaRepartoExceso
	if tieneReparto {
		punteroReparto = &reparto
	}
	_, err := reglasbaremo.NuevoGrupoConcurrenciaExperiencia(
		grupo.Clave(), grupo.Definicion(), grupo.Orden(),
		grupo.CoincidenciaReglas(), grupo.Solape(), punteroReparto,
	)
	return err == nil
}

func reglaPlanValida(regla reglasbaremo.ReglaExperiencia) bool {
	_, err := reglasbaremo.NuevaReglaExperiencia(
		regla.Clave(), regla.Definicion(), regla.SeccionClave(), regla.Orden(),
		regla.Criterios(), regla.GrupoConcurrenciaClave(), regla.PrioridadConcurrencia(),
		regla.UnidadTemporal(), regla.Jornada(), regla.Restos(), regla.Redondeo(),
		regla.PuntosPorUnidad(), regla.MaximoUnidades(), regla.MaximoPuntos(),
	)
	return err == nil
}

func errorPlanInvalido() error {
	return nuevoError("plan", CodigoCompilacionPlanInvalido)
}
