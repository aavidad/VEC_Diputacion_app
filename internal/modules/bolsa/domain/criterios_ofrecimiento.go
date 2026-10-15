package domain

import (
	"errors"
	"strings"
	"time"
)

var ErrCriteriosOfrecimientoInvalidos = errors.New("bolsa: criterios de ofrecimiento invalidos")

const (
	RechazoModalidadSinRenuncia = "no_renuncia"
	RechazoModalidadConRenuncia = "renuncia"

	MotivoOfrecimientoElegible                 = "elegible"
	MotivoOfrecimientoSituacionNoDisponible    = "situacion_no_disponible"
	MotivoOfrecimientoDisponibleDesde          = "disponible_desde"
	MotivoOfrecimientoRestriccionActiva        = "restriccion_activa"
	MotivoOfrecimientoCesePendiente            = "cese_pendiente"
	MotivoOfrecimientoRespuestaPendiente       = "respuesta_pendiente"
	MotivoOfrecimientoModalidadNoAdmitida      = "modalidad_no_admitida"
	MotivoOfrecimientoModalidadNoElegida       = "modalidad_no_elegida"
	MotivoOfrecimientoTrabajando               = "trabajando"
	MotivoOfrecimientoDatosRelacionIncompletos = "datos_relacion_incompletos"
	MotivoOfrecimientoDatosOfertaIncompletos   = "datos_oferta_incompletos"
	MotivoOfrecimientoCriteriosInvalidos       = "criterios_invalidos"
)

// CriteriosOfrecimiento es el valor opcional y versionado de la política de
// ofertas. La ausencia de valor conserva la exclusión de quien trabaja.
// Las claves de modalidad y causa proceden del catálogo de cada bolsa.
type CriteriosOfrecimiento struct {
	FuenteNormativa       string                   `json:"fuente_normativa,omitempty"`
	ModalidadesAceptables []string                 `json:"modalidades_aceptables,omitempty"`
	Mejora                *ReglaMejoraOfrecimiento `json:"mejora,omitempty"`
	ModalidadNoAceptada   string                   `json:"modalidad_no_aceptada,omitempty"`
}

// Una relación activa debe satisfacer alguna condición para que pueda
// ofrecerse una modalidad de destino a quien ya presta servicios.
type ReglaMejoraOfrecimiento struct {
	Habilitada          bool                      `json:"habilitada"`
	ModalidadesDestino  []string                  `json:"modalidades_destino"`
	CondicionesActuales []CondicionRelacionActual `json:"condiciones_actuales"`
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
	Modalidad      string     `json:"modalidad"`
	Causa          string     `json:"causa"`
	Inicio         time.Time  `json:"inicio"`
	FinInicial     *time.Time `json:"fin_inicial,omitempty"`
	BajaComprobada *bool      `json:"baja_comprobada,omitempty"`
}

// ModalidadesElegidas: nil significa que la persona no hizo una selección;
// una lista vacía expresa que no acepta ninguna modalidad.
type EvaluacionOfrecimiento struct {
	Ahora               time.Time        `json:"ahora"`
	Situacion           string           `json:"situacion"`
	DisponibleDesde     *time.Time       `json:"disponible_desde,omitempty"`
	RestriccionActiva   bool             `json:"restriccion_activa"`
	CesePendiente       bool             `json:"cese_pendiente"`
	RespuestaPendiente  bool             `json:"respuesta_pendiente"`
	ModalidadOfrecida   string           `json:"modalidad_ofrecida"`
	ModalidadesElegidas *[]string        `json:"modalidades_elegidas,omitempty"`
	RelacionesActuales  []RelacionActual `json:"relaciones_actuales,omitempty"`
}

type DecisionOfrecimiento struct {
	Elegible       bool   `json:"elegible"`
	Motivo         string `json:"motivo"`
	RenunciaCuenta bool   `json:"renuncia_cuenta"`
}

// Validar solo acepta una política vacía o una política con fuente y efecto
// explícito de la modalidad excluida. No incorpora modalidades jurídicas.
func (c CriteriosOfrecimiento) Validar() error {
	if c.FuenteNormativa == "" && c.ModalidadNoAceptada == "" &&
		len(c.ModalidadesAceptables) == 0 && c.Mejora == nil {
		return nil
	}
	if !fuenteNormativaOfrecimientoValida(c.FuenteNormativa) ||
		(c.ModalidadNoAceptada != RechazoModalidadSinRenuncia && c.ModalidadNoAceptada != RechazoModalidadConRenuncia) ||
		len(c.ModalidadesAceptables) > 100 || !clavesOfrecimientoValidas(c.ModalidadesAceptables) {
		return ErrCriteriosOfrecimientoInvalidos
	}
	if c.Mejora == nil {
		return nil
	}
	if len(c.Mejora.ModalidadesDestino) == 0 || len(c.Mejora.ModalidadesDestino) > 100 ||
		!clavesOfrecimientoValidas(c.Mejora.ModalidadesDestino) ||
		len(c.Mejora.CondicionesActuales) == 0 || len(c.Mejora.CondicionesActuales) > 100 {
		return ErrCriteriosOfrecimientoInvalidos
	}
	if len(c.ModalidadesAceptables) > 0 {
		for _, destino := range c.Mejora.ModalidadesDestino {
			if !contieneClave(c.ModalidadesAceptables, destino) {
				return ErrCriteriosOfrecimientoInvalidos
			}
		}
	}
	for _, condicion := range c.Mejora.CondicionesActuales {
		if !claveOfrecimientoValida(condicion.Modalidad) ||
			(condicion.Causa != "" && !claveOfrecimientoValida(condicion.Causa)) ||
			!limiteMesesValido(condicion.MesesTranscurridos) || !limiteMesesValido(condicion.MesesDuracionInicial) {
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

// EvaluarCriteriosOfrecimiento aplica primero los impedimentos de la
// participación y después los criterios de modalidad y relación actual.
func EvaluarCriteriosOfrecimiento(c CriteriosOfrecimiento, e EvaluacionOfrecimiento) DecisionOfrecimiento {
	excluir := func(motivo string) DecisionOfrecimiento { return DecisionOfrecimiento{Motivo: motivo} }
	if c.Validar() != nil {
		return excluir(MotivoOfrecimientoCriteriosInvalidos)
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
		return excluir(MotivoOfrecimientoDatosOfertaIncompletos)
	}
	switch e.Situacion {
	case SituacionDisponible:
	case SituacionDisponibleDesde:
		if e.DisponibleDesde == nil || e.DisponibleDesde.IsZero() {
			return excluir(MotivoOfrecimientoDatosOfertaIncompletos)
		}
		if e.Ahora.Before(*e.DisponibleDesde) {
			return excluir(MotivoOfrecimientoDisponibleDesde)
		}
	case SituacionTrabajando:
		if c.Mejora == nil || !c.Mejora.Habilitada || !contieneClave(c.Mejora.ModalidadesDestino, e.ModalidadOfrecida) {
			return excluir(MotivoOfrecimientoTrabajando)
		}
		if len(e.RelacionesActuales) == 0 {
			return excluir(MotivoOfrecimientoDatosRelacionIncompletos)
		}
		for _, relacion := range e.RelacionesActuales {
			if !relacionAdmiteMejora(c.Mejora.CondicionesActuales, relacion, e.Ahora) {
				if relacionIncompleta(c.Mejora.CondicionesActuales, relacion, e.Ahora) {
					return excluir(MotivoOfrecimientoDatosRelacionIncompletos)
				}
				return excluir(MotivoOfrecimientoTrabajando)
			}
		}
	default:
		return excluir(MotivoOfrecimientoSituacionNoDisponible)
	}
	if len(c.ModalidadesAceptables) > 0 && !contieneClave(c.ModalidadesAceptables, e.ModalidadOfrecida) {
		return excluir(MotivoOfrecimientoModalidadNoAdmitida)
	}
	if e.ModalidadesElegidas != nil && !contieneClave(*e.ModalidadesElegidas, e.ModalidadOfrecida) {
		return DecisionOfrecimiento{Motivo: MotivoOfrecimientoModalidadNoElegida,
			RenunciaCuenta: c.ModalidadNoAceptada == RechazoModalidadConRenuncia}
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
	if !claveOfrecimientoValida(r.Modalidad) || r.Causa != "" && !claveOfrecimientoValida(r.Causa) ||
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
		if condicion.RequiereBajaComprobada && r.BajaComprobada == nil ||
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
		if condicion.Modalidad != r.Modalidad || condicion.Causa != "" && condicion.Causa != r.Causa ||
			condicion.RequiereBajaComprobada && (r.BajaComprobada == nil || !*r.BajaComprobada) ||
			!cumpleMeses(r.Inicio, ahora, condicion.MesesTranscurridos) ||
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
