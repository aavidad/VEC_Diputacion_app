package domain

import "time"

type ResultadoComprobacion string

const (
	ComprobacionAfirmativa ResultadoComprobacion = "afirmativa"
	ComprobacionNegativa   ResultadoComprobacion = "negativa"
	ComprobacionNoAplica   ResultadoComprobacion = "no_aplica"
)

func (r ResultadoComprobacion) valido() bool {
	return r == ComprobacionAfirmativa || r == ComprobacionNegativa ||
		r == ComprobacionNoAplica
}

type ComprobacionCobertura struct {
	Clave      ClaveCatalogo         `json:"clave"`
	Resultado  ResultadoComprobacion `json:"resultado"`
	FuenteRef  string                `json:"fuente_ref"`
	ReciboRef  string                `json:"recibo_ref"`
	EvaluadaEn time.Time             `json:"evaluada_en"`
	Detalle    string                `json:"detalle,omitempty"`
}

func (c ComprobacionCobertura) Validar() error {
	if !c.Clave.Valida() || !c.Resultado.valido() ||
		!referenciaValida(c.FuenteRef) || !referenciaValida(c.ReciboRef) ||
		!instanteCanonico(c.EvaluadaEn) || !textoValido(c.Detalle, 1000, true) {
		return ErrDatoInvalido
	}
	return nil
}

type DecisionViaCobertura struct {
	ViaClave         ClaveCatalogo           `json:"via_clave"`
	ProcedimientoRef string                  `json:"procedimiento_ref"`
	BolsaRef         string                  `json:"bolsa_ref,omitempty"`
	Comprobaciones   []ComprobacionCobertura `json:"comprobaciones"`
	Motivacion       string                  `json:"motivacion"`
}

func (d DecisionViaCobertura) Validar() error {
	if !d.ViaClave.Valida() || !referenciaValida(d.ProcedimientoRef) ||
		(d.BolsaRef != "" && !referenciaValida(d.BolsaRef)) ||
		len(d.Comprobaciones) == 0 || len(d.Comprobaciones) > 32 ||
		!textoValido(d.Motivacion, 2000, false) {
		return ErrDatoInvalido
	}
	vistas := make(map[ClaveCatalogo]struct{}, len(d.Comprobaciones))
	for _, comprobacion := range d.Comprobaciones {
		if comprobacion.Validar() != nil {
			return ErrDatoInvalido
		}
		if _, repetida := vistas[comprobacion.Clave]; repetida {
			return ErrDatoInvalido
		}
		vistas[comprobacion.Clave] = struct{}{}
	}
	return nil
}

func (d DecisionViaCobertura) clonar() DecisionViaCobertura {
	d.Comprobaciones = append([]ComprobacionCobertura(nil), d.Comprobaciones...)
	return d
}

type AsignacionUnidad struct {
	UnidadRef       string    `json:"unidad_ref"`
	ResponsableRef  string    `json:"responsable_ref"`
	NotificacionRef string    `json:"notificacion_ref"`
	AsignadaEn      time.Time `json:"asignada_en"`
	Observaciones   string    `json:"observaciones,omitempty"`
}

func (a AsignacionUnidad) Validar() error {
	if !referenciaValida(a.UnidadRef) || !referenciaValida(a.ResponsableRef) ||
		!referenciaValida(a.NotificacionRef) || !instanteCanonico(a.AsignadaEn) ||
		!textoValido(a.Observaciones, 1000, true) {
		return ErrDatoInvalido
	}
	return nil
}
