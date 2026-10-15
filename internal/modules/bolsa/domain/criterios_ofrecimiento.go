package domain

import (
	"errors"
	"strings"
	"time"
)

var ErrCriteriosOfrecimientoInvalidos = errors.New("bolsa: criterios de ofrecimiento invalidos")

const (
	MotivoOfrecimientoElegible                 = "elegible"
	MotivoOfrecimientoSituacionNoDisponible    = "situacion_no_disponible"
	MotivoOfrecimientoDisponibleDesde          = "disponible_desde"
	MotivoOfrecimientoRestriccionActiva        = "restriccion_activa"
	MotivoOfrecimientoCesePendiente            = "cese_pendiente"
	MotivoOfrecimientoRespuestaPendiente       = "respuesta_pendiente"
	MotivoOfrecimientoTrabajando               = "trabajando"
	MotivoOfrecimientoPendienteComprobacion    = "pendiente_comprobacion_rrhh"
	MotivoOfrecimientoDatosRelacionIncompletos = MotivoOfrecimientoPendienteComprobacion
	MotivoOfrecimientoDatosOfertaIncompletos   = "datos_oferta_incompletos"
	MotivoOfrecimientoCriteriosInvalidos       = "criterios_invalidos"
)

// CriteriosOfrecimiento es el valor versionado de la política de ofertas.
type CriteriosOfrecimiento struct {
	FuenteNormativa string                   `json:"fuente_normativa,omitempty"`
	Mejora          *ReglaMejoraOfrecimiento `json:"mejora,omitempty"`
}

// Una relación activa debe satisfacer alguna condición para que pueda
// ofrecerse una modalidad de destino a quien ya presta servicios.
type ReglaMejoraOfrecimiento struct {
	Habilitada                     bool                      `json:"habilitada"`
	ModalidadesDestino             []string                  `json:"modalidades_destino"`
	CausasDestino                  []string                  `json:"causas_destino"`
	CondicionesActuales            []CondicionRelacionActual `json:"condiciones_actuales"`
	MaximoVacanteMeses             int                       `json:"maximo_vacante_meses"`
	SupuestosReduccionVacante      []string                  `json:"supuestos_reduccion_vacante,omitempty"`
	ReduccionPorModalidadesPrevias []string                  `json:"reduccion_por_modalidades_previas"`
}

type CondicionRelacionActual struct {
	Modalidad              string       `json:"modalidad"`
	Causa                  string       `json:"causa,omitempty"`
	RequiereBajaComprobada bool         `json:"requiere_baja_comprobada,omitempty"`
	MesesTranscurridos     *LimiteMeses `json:"meses_transcurridos,omitempty"`
	MesesDuracionInicial   *LimiteMeses `json:"meses_duracion_inicial,omitempty"`
}

// IncluyeLimite distingue «hasta nueve meses» de «menos de nueve meses».
type LimiteMeses struct {
	Meses         int  `json:"meses"`
	IncluyeLimite bool `json:"incluye_limite"`
}

type RelacionActual struct {
	EventoRef      string     `json:"evento_ref"`
	Modalidad      string     `json:"modalidad"`
	Causa          string     `json:"causa"`
	Inicio         time.Time  `json:"inicio"`
	FinInicial     *time.Time `json:"fin_inicial,omitempty"`
	BajaComprobada *bool      `json:"baja_comprobada,omitempty"`
	BajaInicio     *time.Time `json:"baja_inicio,omitempty"`
	BajaEventoRef  string     `json:"baja_evento_ref,omitempty"`
}

type EvaluacionOfrecimiento struct {
	Ahora              time.Time        `json:"ahora"`
	Situacion          string           `json:"situacion"`
	DisponibleDesde    *time.Time       `json:"disponible_desde,omitempty"`
	RestriccionActiva  bool             `json:"restriccion_activa"`
	CesePendiente      bool             `json:"cese_pendiente"`
	RespuestaPendiente bool             `json:"respuesta_pendiente"`
	ModalidadOfrecida  string           `json:"modalidad_ofrecida"`
	CausaOfrecida      string           `json:"causa_ofrecida,omitempty"`
	RelacionesActuales []RelacionActual `json:"relaciones_actuales,omitempty"`
}

type DecisionOfrecimiento struct {
	Elegible                 bool                      `json:"elegible"`
	Motivo                   string                    `json:"motivo"`
	PendienteComprobacion    bool                      `json:"pendiente_comprobacion"`
	MaximoVacanteMeses       int                       `json:"maximo_vacante_meses,omitempty"`
	PeriodosReduccionVacante []PeriodoReduccionVacante `json:"periodos_reduccion_vacante,omitempty"`
}

// PeriodoReduccionVacante se descuenta del plazo máximo al formalizar la
// vacante con su fecha real de inicio. El evento permite comprobar el hecho.
type PeriodoReduccionVacante struct {
	EventoRef string    `json:"evento_ref"`
	Desde     time.Time `json:"desde"`
	Hasta     time.Time `json:"hasta"`
}

// Validar acepta la política vacía anterior o una regla con fuente y límites.
func (c CriteriosOfrecimiento) Validar() error {
	if c.FuenteNormativa == "" && c.Mejora == nil {
		return nil
	}
	if !fuenteNormativaOfrecimientoValida(c.FuenteNormativa) {
		return ErrCriteriosOfrecimientoInvalidos
	}
	if c.Mejora == nil {
		return nil
	}
	if len(c.Mejora.ModalidadesDestino) == 0 || len(c.Mejora.ModalidadesDestino) > 100 ||
		!clavesOfrecimientoValidas(c.Mejora.ModalidadesDestino) ||
		len(c.Mejora.CausasDestino) == 0 || len(c.Mejora.CausasDestino) > 100 ||
		!clavesOfrecimientoValidas(c.Mejora.CausasDestino) ||
		len(c.Mejora.CondicionesActuales) == 0 || len(c.Mejora.CondicionesActuales) > 100 ||
		c.Mejora.MaximoVacanteMeses < 1 || c.Mejora.MaximoVacanteMeses > 1200 ||
		len(c.Mejora.ReduccionPorModalidadesPrevias) == 0 ||
		!clavesOfrecimientoValidas(c.Mejora.ReduccionPorModalidadesPrevias) ||
		(c.Mejora.SupuestosReduccionVacante != nil &&
			(len(c.Mejora.SupuestosReduccionVacante) == 0 || len(c.Mejora.SupuestosReduccionVacante) > 100 ||
				!clavesOfrecimientoValidas(c.Mejora.SupuestosReduccionVacante))) {
		return ErrCriteriosOfrecimientoInvalidos
	}
	for _, condicion := range c.Mejora.CondicionesActuales {
		if !claveOfrecimientoValida(condicion.Modalidad) ||
			(condicion.Causa != "" && !claveOfrecimientoValida(condicion.Causa)) ||
			!limiteMesesValido(condicion.MesesTranscurridos) || !limiteMesesValido(condicion.MesesDuracionInicial) {
			return ErrCriteriosOfrecimientoInvalidos
		}
	}
	for _, modalidad := range c.Mejora.ReduccionPorModalidadesPrevias {
		presente := false
		for _, condicion := range c.Mejora.CondicionesActuales {
			presente = presente || condicion.Modalidad == modalidad
		}
		if !presente {
			return ErrCriteriosOfrecimientoInvalidos
		}
	}
	for _, supuesto := range c.Mejora.SupuestosReduccionVacante {
		presente := false
		for _, condicion := range c.Mejora.CondicionesActuales {
			presente = presente || condicion.Modalidad == supuesto
		}
		if !presente {
			return ErrCriteriosOfrecimientoInvalidos
		}
	}
	return nil
}

func claveOfrecimientoValida(s string) bool {
	return patronClaveContrato.MatchString(s)
}

func fuenteNormativaOfrecimientoValida(s string) bool {
	return len(s) >= 1 && len(s) <= 2000 && s == strings.TrimSpace(s)
}

func clavesOfrecimientoValidas(claves []string) bool {
	vistas := make(map[string]bool, len(claves))
	for _, clave := range claves {
		if !claveOfrecimientoValida(clave) || vistas[clave] {
			return false
		}
		vistas[clave] = true
	}
	return true
}

func limiteMesesValido(l *LimiteMeses) bool { return l == nil || l.Meses >= 1 && l.Meses <= 1200 }

// EvaluarCriteriosOfrecimiento no recibe preferencias de modalidad: solo
// decide con la norma publicada, la situación y hechos acreditados.
func EvaluarCriteriosOfrecimiento(c CriteriosOfrecimiento, e EvaluacionOfrecimiento) DecisionOfrecimiento {
	excluir := func(motivo string) DecisionOfrecimiento { return DecisionOfrecimiento{Motivo: motivo} }
	pendiente := func(motivo string) DecisionOfrecimiento {
		return DecisionOfrecimiento{Motivo: motivo, PendienteComprobacion: true}
	}
	if c.Validar() != nil {
		return pendiente(MotivoOfrecimientoCriteriosInvalidos)
	}
	if e.RestriccionActiva {
		return excluir(MotivoOfrecimientoRestriccionActiva)
	}
	if e.CesePendiente {
		return excluir(MotivoOfrecimientoCesePendiente)
	}
	if e.RespuestaPendiente {
		return excluir(MotivoOfrecimientoRespuestaPendiente)
	}
	if e.Ahora.IsZero() || !claveOfrecimientoValida(e.ModalidadOfrecida) {
		return pendiente(MotivoOfrecimientoDatosOfertaIncompletos)
	}
	var periodos []PeriodoReduccionVacante
	switch e.Situacion {
	case SituacionDisponible:
	case SituacionDisponibleDesde:
		if e.DisponibleDesde == nil || e.DisponibleDesde.IsZero() {
			return pendiente(MotivoOfrecimientoDatosOfertaIncompletos)
		}
		if e.Ahora.Before(*e.DisponibleDesde) {
			return excluir(MotivoOfrecimientoDisponibleDesde)
		}
	case SituacionTrabajando:
		if c.Mejora == nil || !c.Mejora.Habilitada || !contieneClave(c.Mejora.ModalidadesDestino, e.ModalidadOfrecida) {
			return excluir(MotivoOfrecimientoTrabajando)
		}
		if !claveOfrecimientoValida(e.CausaOfrecida) {
			return pendiente(MotivoOfrecimientoDatosOfertaIncompletos)
		}
		if !contieneClave(c.Mejora.CausasDestino, e.CausaOfrecida) {
			return excluir(MotivoOfrecimientoTrabajando)
		}
		if len(e.RelacionesActuales) == 0 {
			return pendiente(MotivoOfrecimientoDatosRelacionIncompletos)
		}
		noAdmitida := false
		// Sin este campo no se presupone el alcance de la reducción.
		aplicaReduccion := false
		for _, relacion := range e.RelacionesActuales {
			aplicaReduccion = aplicaReduccion || contieneClave(c.Mejora.SupuestosReduccionVacante, relacion.Modalidad)
		}
		for _, relacion := range e.RelacionesActuales {
			if relacionIncompleta(c.Mejora.CondicionesActuales, relacion, e.Ahora) {
				return pendiente(MotivoOfrecimientoDatosRelacionIncompletos)
			}
			if !relacionAdmiteMejora(c.Mejora.CondicionesActuales, relacion, e.Ahora) {
				noAdmitida = true
			}
			if aplicaReduccion && contieneClave(c.Mejora.ReduccionPorModalidadesPrevias, relacion.Modalidad) {
				periodos = append(periodos, PeriodoReduccionVacante{
					EventoRef: relacion.EventoRef, Desde: relacion.Inicio, Hasta: e.Ahora,
				})
			}
		}
		if noAdmitida {
			return excluir(MotivoOfrecimientoTrabajando)
		}
	default:
		if e.Situacion == "" {
			return pendiente(MotivoOfrecimientoPendienteComprobacion)
		}
		return excluir(MotivoOfrecimientoSituacionNoDisponible)
	}
	if e.Situacion == SituacionTrabajando {
		// R1a solo acredita que la excepción puede ser revisada. La mejora
		// requiere el acto R1b y la comprobación del plazo resultante.
		return DecisionOfrecimiento{
			Motivo: MotivoOfrecimientoPendienteComprobacion, PendienteComprobacion: true,
			MaximoVacanteMeses:       c.Mejora.MaximoVacanteMeses,
			PeriodosReduccionVacante: periodos,
		}
	}
	return DecisionOfrecimiento{Elegible: true, Motivo: MotivoOfrecimientoElegible}
}

func contieneClave(claves []string, buscada string) bool {
	for _, clave := range claves {
		if clave == buscada {
			return true
		}
	}
	return false
}

func relacionIncompleta(condiciones []CondicionRelacionActual, r RelacionActual, ahora time.Time) bool {
	if !referenciaOpacaContrato(r.EventoRef) || !claveOfrecimientoValida(r.Modalidad) ||
		r.Causa != "" && !claveOfrecimientoValida(r.Causa) ||
		r.Inicio.IsZero() || r.Inicio.After(ahora) {
		return true
	}
	for _, condicion := range condiciones {
		if condicion.Modalidad != r.Modalidad {
			continue
		}
		if condicion.Causa != "" && r.Causa == "" {
			return true
		}
		if condicion.Causa != "" && condicion.Causa != r.Causa {
			continue
		}
		if condicion.RequiereBajaComprobada && (r.BajaComprobada == nil || !referenciaOpacaContrato(r.BajaEventoRef) || r.BajaInicio == nil ||
			r.BajaInicio.IsZero() || r.BajaInicio.After(ahora) || r.BajaInicio.After(r.Inicio)) ||
			condicion.MesesDuracionInicial != nil && (r.FinInicial == nil || r.FinInicial.IsZero() || !r.FinInicial.After(r.Inicio)) {
			return true
		}
	}
	return false
}

func relacionAdmiteMejora(condiciones []CondicionRelacionActual, r RelacionActual, ahora time.Time) bool {
	if !claveOfrecimientoValida(r.Modalidad) || r.Causa != "" && !claveOfrecimientoValida(r.Causa) ||
		r.Inicio.IsZero() || r.Inicio.After(ahora) {
		return false
	}
	for _, condicion := range condiciones {
		inicioPlazo := r.Inicio
		if condicion.RequiereBajaComprobada && r.BajaInicio != nil {
			inicioPlazo = *r.BajaInicio
		}
		if condicion.Modalidad != r.Modalidad || condicion.Causa != "" && condicion.Causa != r.Causa ||
			condicion.RequiereBajaComprobada && (r.BajaComprobada == nil || !*r.BajaComprobada) ||
			!cumpleMeses(inicioPlazo, ahora, condicion.MesesTranscurridos) ||
			condicion.MesesDuracionInicial != nil && (r.FinInicial == nil || !r.FinInicial.After(r.Inicio) ||
				!cumpleMeses(r.Inicio, *r.FinInicial, condicion.MesesDuracionInicial)) {
			continue
		}
		return true
	}
	return false
}

func cumpleMeses(inicio, fin time.Time, limite *LimiteMeses) bool {
	if limite == nil {
		return true
	}
	// AddDate desborda el final de un mes (31/05 + 9 meses sería 03/03).
	// El plazo natural conserva el día o usa el último del mes de destino.
	anio, mes, dia := inicio.Date()
	ultimo := time.Date(anio, mes+time.Month(limite.Meses)+1, 0, 0, 0, 0, 0, inicio.Location()).Day()
	if dia > ultimo {
		dia = ultimo
	}
	frontera := time.Date(anio, mes+time.Month(limite.Meses), dia,
		inicio.Hour(), inicio.Minute(), inicio.Second(), inicio.Nanosecond(), inicio.Location())
	return fin.Before(frontera) || limite.IncluyeLimite && fin.Equal(frontera)
}
