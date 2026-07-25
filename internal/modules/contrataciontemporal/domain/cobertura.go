package domain

import "time"

type ResultadoComprobacion string

const (
	ComprobacionAfirmativa ResultadoComprobacion = "afirmativa"
	ComprobacionNegativa   ResultadoComprobacion = "negativa"
	ComprobacionNoAplica   ResultadoComprobacion = "no_aplica"
	// ComprobacionNoConsta acredita que la fuente respondió sin disponer del
	// dato. No equivale a omisión, resultado negativo ni indisponibilidad.
	ComprobacionNoConsta ResultadoComprobacion = "no_consta"
)

func (r ResultadoComprobacion) valido() bool {
	return r == ComprobacionAfirmativa || r == ComprobacionNegativa ||
		r == ComprobacionNoAplica || r == ComprobacionNoConsta
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
	UnidadRef         string                      `json:"unidad_ref"`
	ResponsableRef    string                      `json:"responsable_ref"`
	NotificacionRef   string                      `json:"notificacion_ref"`
	AsignadaEn        time.Time                   `json:"asignada_en"`
	MotivoClave       ClaveCatalogo               `json:"motivo_clave,omitempty"`
	Observaciones     string                      `json:"observaciones,omitempty"`
	ActuacionRegistro *VinculoActuacionAsignacion `json:"actuacion_registro"`
}

func (a AsignacionUnidad) Validar() error {
	if a.validarDatos() != nil || a.ActuacionRegistro == nil ||
		a.ActuacionRegistro.validar() != nil {
		return ErrDatoInvalido
	}
	return nil
}

func (a AsignacionUnidad) validarEntrada() error {
	if a.validarDatos() != nil || a.MotivoClave != "" ||
		a.ActuacionRegistro != nil {
		return ErrDatoInvalido
	}
	return nil
}

func (a AsignacionUnidad) validarReasignacion() error {
	if a.validarDatos() != nil || !a.MotivoClave.Valida() ||
		a.ActuacionRegistro != nil {
		return ErrDatoInvalido
	}
	return nil
}

func (a AsignacionUnidad) validarDatos() error {
	if !referenciaValida(a.UnidadRef) || !referenciaValida(a.ResponsableRef) ||
		!referenciaValida(a.NotificacionRef) || !instanteCanonico(a.AsignadaEn) ||
		(a.MotivoClave != "" && !a.MotivoClave.Valida()) ||
		!textoValido(a.Observaciones, 1000, true) {
		return ErrDatoInvalido
	}
	return nil
}

// VinculoActuacionAsignacion impide que una proyección de asignación válida
// se adjunte a una actuación distinta durante la rehidratación.
type VinculoActuacionAsignacion struct {
	Secuencia              uint64        `json:"secuencia"`
	VersionExpediente      uint64        `json:"version_expediente"`
	AccionClave            ClaveCatalogo `json:"accion_clave"`
	FaseDestino            ClaveFase     `json:"fase_destino"`
	ReciboRef              string        `json:"recibo_ref"`
	UnidadAsignadaRef      string        `json:"unidad_asignada_ref"`
	ResponsableAsignadoRef string        `json:"responsable_asignado_ref"`
	NotificacionRef        string        `json:"notificacion_ref"`
	MotivoClave            ClaveCatalogo `json:"motivo_clave,omitempty"`
}

func (v VinculoActuacionAsignacion) validar() error {
	if v.Secuencia < 2 || v.VersionExpediente < 2 ||
		v.Secuencia != v.VersionExpediente ||
		!v.AccionClave.Valida() || !v.FaseDestino.Valida() ||
		!referenciaValida(v.ReciboRef) ||
		!referenciaValida(v.UnidadAsignadaRef) ||
		!referenciaValida(v.ResponsableAsignadoRef) ||
		!referenciaValida(v.NotificacionRef) ||
		(v.MotivoClave != "" && !v.MotivoClave.Valida()) {
		return ErrDatoInvalido
	}
	return nil
}

func (v VinculoActuacionAsignacion) correspondeA(
	actuacion Actuacion,
	asignacion AsignacionUnidad,
) bool {
	return v.validar() == nil &&
		v.Secuencia == actuacion.Secuencia &&
		v.VersionExpediente == actuacion.VersionExpediente &&
		v.AccionClave == actuacion.AccionClave &&
		v.FaseDestino == actuacion.FaseDestino &&
		v.ReciboRef == actuacion.ReciboRef &&
		v.UnidadAsignadaRef == asignacion.UnidadRef &&
		v.ResponsableAsignadoRef == asignacion.ResponsableRef &&
		v.NotificacionRef == asignacion.NotificacionRef &&
		v.MotivoClave == asignacion.MotivoClave
}

func nuevoVinculoActuacionAsignacion(
	versionExpediente uint64,
	secuencia uint64,
	actuacion DatosActuacion,
	asignacion AsignacionUnidad,
) VinculoActuacionAsignacion {
	return VinculoActuacionAsignacion{
		Secuencia: secuencia, VersionExpediente: versionExpediente,
		AccionClave: actuacion.AccionClave, FaseDestino: actuacion.FaseDestino,
		ReciboRef:              actuacion.ReciboRef,
		UnidadAsignadaRef:      asignacion.UnidadRef,
		ResponsableAsignadoRef: asignacion.ResponsableRef,
		NotificacionRef:        asignacion.NotificacionRef,
		MotivoClave:            asignacion.MotivoClave,
	}
}
