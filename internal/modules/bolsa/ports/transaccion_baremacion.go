package ports

import (
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type AccionAuditoriaBaremacion string

const (
	AccionAuditoriaCrearBaremacion    AccionAuditoriaBaremacion = "crear_baremacion"
	AccionAuditoriaIncorporarDecision AccionAuditoriaBaremacion = "incorporar_decision_baremacion"
)

func (a AccionAuditoriaBaremacion) valida() bool {
	return a == AccionAuditoriaCrearBaremacion || a == AccionAuditoriaIncorporarDecision
}

type TipoEventoOutboxBaremacion string

const (
	TipoEventoBaremacionCreada    TipoEventoOutboxBaremacion = "bolsa.baremacion_creada.v1"
	TipoEventoDecisionIncorporada TipoEventoOutboxBaremacion = "bolsa.decision_baremacion_incorporada.v1"
)

type EstadoEventoOutboxBaremacion string

const EstadoEventoOutboxBaremacionPendiente EstadoEventoOutboxBaremacion = "pendiente"

// RegistroAuditoriaBaremacion es una proyeccion probatoria cerrada, no una
// entrada que el repositorio acepte al escribir.
type RegistroAuditoriaBaremacion struct {
	Referencia                    string
	Secuencia                     uint64
	PrincipalRef                  string
	SujetoRef                     string
	PerfilActorClave              string
	MetodoAutenticacion           dominiovec.AuthMethod
	NivelAutenticacion            dominiovec.AuthAssurance
	GarantiaMinima                dominiovec.AuthAssurance
	AutenticacionRef              string
	AutorizacionRef               string
	AccionAutorizada              AccionOperacionBaremacion
	ClaseRecursoAutorizada        ClaseRecursoOperacionBaremacion
	RecursoAutorizadoRef          string
	CamposPermitidos              []string
	FinalidadClave                string
	CorrelacionRef                string
	Modulo                        string
	Accion                        AccionAuditoriaBaremacion
	ClaseCambio                   ClaseCambioBaremacion
	ProcesoRef                    string
	SolicitudRef                  string
	BaremacionMeritoRef           string
	DecisionRef                   string
	ManifiestoProbatorioRef       string
	HuellaManifiestoSHA256        string
	DocumentoFirmadoCustodiadoRef string
	EvidenciaCustodiaFirmadoRef   string
	EvidenciaRetencionFirmadoRef  string
	VersionAnterior               uint64
	VersionNueva                  uint64
	HuellaAnteriorSHA256          string
	HuellaNuevaSHA256             string
	MotivoClave                   string
	Motivo                        string
	HuellaSolicitudHMAC           string
	Resultado                     string
	SolicitadaConfirmacionEn      time.Time
	RegistradaEn                  time.Time
	HuellaAnteriorAuditoriaSHA256 string
	HuellaRegistroSHA256          string
}

func (r RegistroAuditoriaBaremacion) Validar() error {
	if !referenciaValida(r.Referencia, 512) || r.Secuencia < 1 || !referenciaValida(r.PrincipalRef, 512) ||
		!referenciaValida(r.SujetoRef, 512) || !referenciaValida(r.PerfilActorClave, 512) ||
		!r.MetodoAutenticacion.Valido() || r.MetodoAutenticacion == dominiovec.AuthMethodDemo ||
		!r.NivelAutenticacion.Valida() || !r.GarantiaMinima.Valida() ||
		!dominiovec.CumpleGarantiaAutenticacion(r.NivelAutenticacion, r.GarantiaMinima) ||
		!referenciaValida(r.AutenticacionRef, 512) ||
		!referenciaValida(r.AutorizacionRef, 512) || !claveValida(r.FinalidadClave) ||
		!referenciaValida(r.CorrelacionRef, 512) || r.Modulo != "bolsa" || !r.Accion.valida() ||
		!r.ClaseCambio.Valida() || !referenciaValida(r.ProcesoRef, 512) || !referenciaValida(r.SolicitudRef, 512) ||
		!referenciaValida(r.BaremacionMeritoRef, 512) || r.VersionNueva != r.VersionAnterior+1 ||
		!huellaSHA256Valida(r.HuellaNuevaSHA256) || !claveValida(r.MotivoClave) || !textoValido(r.Motivo, 8000) ||
		!huellaHMACSHA256Valida(r.HuellaSolicitudHMAC) || r.Resultado != "correcto" ||
		r.SolicitadaConfirmacionEn.IsZero() || r.RegistradaEn.IsZero() || r.RegistradaEn.Before(r.SolicitadaConfirmacionEn) ||
		!huellaSHA256Valida(r.HuellaRegistroSHA256) {
		return ErrSolicitudBaremacionInvalida
	}
	especificacion, existe := especificacionesAccionBaremacion[r.AccionAutorizada]
	if !existe || r.ClaseRecursoAutorizada != especificacion.clase ||
		!referenciaValida(r.RecursoAutorizadoRef, 512) || !mismosCamposExactos(r.CamposPermitidos, especificacion.campos) {
		return ErrSolicitudBaremacionInvalida
	}
	if r.VersionAnterior == 0 {
		if r.Accion != AccionAuditoriaCrearBaremacion || r.ClaseCambio != ClaseCambioAltaBaremacion ||
			r.AccionAutorizada != AccionConfirmarAltaBaremacion || r.DecisionRef != "" || r.HuellaAnteriorSHA256 != "" ||
			r.ManifiestoProbatorioRef != "" || r.HuellaManifiestoSHA256 != "" ||
			r.DocumentoFirmadoCustodiadoRef != "" || r.EvidenciaCustodiaFirmadoRef != "" ||
			r.EvidenciaRetencionFirmadoRef != "" {
			return ErrSolicitudBaremacionInvalida
		}
	} else if r.Accion != AccionAuditoriaIncorporarDecision || r.ClaseCambio != ClaseCambioIncorporarDecision ||
		r.AccionAutorizada != AccionConfirmarDecisionBaremacion ||
		!referenciaValida(r.DecisionRef, 512) || !huellaSHA256Valida(r.HuellaAnteriorSHA256) ||
		!referenciaValida(r.ManifiestoProbatorioRef, 512) || !huellaSHA256Valida(r.HuellaManifiestoSHA256) ||
		!referenciaValida(r.DocumentoFirmadoCustodiadoRef, 512) ||
		!referenciaValida(r.EvidenciaCustodiaFirmadoRef, 512) ||
		!referenciaValida(r.EvidenciaRetencionFirmadoRef, 512) {
		return ErrSolicitudBaremacionInvalida
	}
	if r.Secuencia == 1 {
		if r.HuellaAnteriorAuditoriaSHA256 != "" {
			return ErrSolicitudBaremacionInvalida
		}
	} else if !huellaSHA256Valida(r.HuellaAnteriorAuditoriaSHA256) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

type EventoOutboxBaremacion struct {
	Referencia                   string
	Secuencia                    uint64
	Tipo                         TipoEventoOutboxBaremacion
	Estado                       EstadoEventoOutboxBaremacion
	Modulo                       string
	ProcesoRef                   string
	SolicitudRef                 string
	BaremacionMeritoRef          string
	DecisionRef                  string
	ManifiestoProbatorioRef      string
	HuellaManifiestoSHA256       string
	DocumentoFirmadoRef          string
	EvidenciaCustodiaFirmadoRef  string
	EvidenciaRetencionFirmadoRef string
	SujetoRef                    string
	PrincipalRef                 string
	VersionNueva                 uint64
	HuellaNuevaSHA256            string
	AuditoriaRef                 string
	HuellaAuditoriaSHA256        string
	CorrelacionRef               string
	RegistradoEn                 time.Time
	HuellaEventoAnteriorSHA256   string
	HuellaRegistroSHA256         string
}

func (e EventoOutboxBaremacion) Validar() error {
	if !referenciaValida(e.Referencia, 512) || e.Secuencia < 1 ||
		(e.Tipo != TipoEventoBaremacionCreada && e.Tipo != TipoEventoDecisionIncorporada) ||
		e.Estado != EstadoEventoOutboxBaremacionPendiente || e.Modulo != "bolsa" ||
		!referenciaValida(e.ProcesoRef, 512) || !referenciaValida(e.SolicitudRef, 512) ||
		!referenciaValida(e.BaremacionMeritoRef, 512) || !referenciaValida(e.SujetoRef, 512) ||
		!referenciaValida(e.PrincipalRef, 512) || e.VersionNueva < 1 || !huellaSHA256Valida(e.HuellaNuevaSHA256) ||
		!referenciaValida(e.AuditoriaRef, 512) || !huellaSHA256Valida(e.HuellaAuditoriaSHA256) ||
		!referenciaValida(e.CorrelacionRef, 512) || e.RegistradoEn.IsZero() ||
		!huellaSHA256Valida(e.HuellaRegistroSHA256) {
		return ErrSolicitudBaremacionInvalida
	}
	if e.Tipo == TipoEventoBaremacionCreada {
		if e.VersionNueva != 1 || e.DecisionRef != "" || e.ManifiestoProbatorioRef != "" ||
			e.HuellaManifiestoSHA256 != "" || e.DocumentoFirmadoRef != "" ||
			e.EvidenciaCustodiaFirmadoRef != "" || e.EvidenciaRetencionFirmadoRef != "" {
			return ErrSolicitudBaremacionInvalida
		}
	} else if e.VersionNueva < 2 || !referenciaValida(e.DecisionRef, 512) ||
		!referenciaValida(e.ManifiestoProbatorioRef, 512) || !huellaSHA256Valida(e.HuellaManifiestoSHA256) ||
		!referenciaValida(e.DocumentoFirmadoRef, 512) ||
		!referenciaValida(e.EvidenciaCustodiaFirmadoRef, 512) || !referenciaValida(e.EvidenciaRetencionFirmadoRef, 512) {
		return ErrSolicitudBaremacionInvalida
	}
	if e.Secuencia == 1 {
		if e.HuellaEventoAnteriorSHA256 != "" {
			return ErrSolicitudBaremacionInvalida
		}
	} else if !huellaSHA256Valida(e.HuellaEventoAnteriorSHA256) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

type SolicitudObtenerEvidenciaTransaccionBaremacion struct {
	Contexto            ContextoOperacionBaremacion
	BaremacionMeritoRef string
	NumeroVersion       uint64
	AuditoriaRef        string
	EventoOutboxRef     string
}

func (s SolicitudObtenerEvidenciaTransaccionBaremacion) Validar() error {
	if s.Contexto.ValidarPara(AccionConsultarEvidenciaTransaccionBaremacion, ClaseRecursoTransaccion, s.AuditoriaRef) != nil ||
		!referenciaValida(s.BaremacionMeritoRef, 512) || s.NumeroVersion < 1 ||
		!referenciaValida(s.AuditoriaRef, 512) || !referenciaValida(s.EventoOutboxRef, 512) ||
		s.AuditoriaRef == s.EventoOutboxRef {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

type EvidenciaTransaccionBaremacionRecuperada struct {
	Version   VersionBaremacion
	Auditoria RegistroAuditoriaBaremacion
	Evento    EventoOutboxBaremacion
	Evidencia EvidenciaTransaccionBaremacion
}

func (r EvidenciaTransaccionBaremacionRecuperada) Validar() error {
	if r.Version.Validar() != nil || r.Auditoria.Validar() != nil || r.Evento.Validar() != nil ||
		r.Evidencia.Validar() != nil || r.Auditoria.BaremacionMeritoRef != r.Version.Referencia.BaremacionMeritoRef ||
		r.Auditoria.VersionNueva != r.Version.Referencia.Numero ||
		r.Auditoria.HuellaNuevaSHA256 != r.Version.Referencia.HuellaEstadoSHA256 ||
		r.Evento.BaremacionMeritoRef != r.Auditoria.BaremacionMeritoRef ||
		r.Evento.VersionNueva != r.Auditoria.VersionNueva || r.Evento.HuellaNuevaSHA256 != r.Auditoria.HuellaNuevaSHA256 ||
		r.Evento.AuditoriaRef != r.Auditoria.Referencia || r.Evento.HuellaAuditoriaSHA256 != r.Auditoria.HuellaRegistroSHA256 ||
		r.Evento.SujetoRef != r.Auditoria.SujetoRef || r.Evento.PrincipalRef != r.Auditoria.PrincipalRef ||
		r.Evento.ManifiestoProbatorioRef != r.Auditoria.ManifiestoProbatorioRef ||
		r.Evento.HuellaManifiestoSHA256 != r.Auditoria.HuellaManifiestoSHA256 ||
		r.Evento.DocumentoFirmadoRef != r.Auditoria.DocumentoFirmadoCustodiadoRef ||
		r.Evento.EvidenciaCustodiaFirmadoRef != r.Auditoria.EvidenciaCustodiaFirmadoRef ||
		r.Evento.EvidenciaRetencionFirmadoRef != r.Auditoria.EvidenciaRetencionFirmadoRef ||
		r.Evento.CorrelacionRef != r.Auditoria.CorrelacionRef || !r.Evento.RegistradoEn.Equal(r.Auditoria.RegistradaEn) ||
		r.Evidencia.AuditoriaRef != r.Auditoria.Referencia ||
		r.Evidencia.HuellaAuditoriaSHA256 != r.Auditoria.HuellaRegistroSHA256 ||
		r.Evidencia.EventoOutboxRef != r.Evento.Referencia ||
		r.Evidencia.HuellaEventoOutboxSHA256 != r.Evento.HuellaRegistroSHA256 ||
		!r.Evidencia.ConfirmadaEn.Equal(r.Auditoria.RegistradaEn) ||
		!r.Version.ConfirmadaEn.Equal(r.Auditoria.RegistradaEn) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (r EvidenciaTransaccionBaremacionRecuperada) ValidarPara(s SolicitudObtenerEvidenciaTransaccionBaremacion) error {
	if s.Validar() != nil || r.Validar() != nil || r.Version.Referencia.BaremacionMeritoRef != s.BaremacionMeritoRef ||
		r.Version.Referencia.Numero != s.NumeroVersion || r.Auditoria.Referencia != s.AuditoriaRef ||
		r.Evento.Referencia != s.EventoOutboxRef || r.Auditoria.SujetoRef != s.Contexto.Proyeccion().SujetoRef {
		return ErrEvidenciaBaremacionNoEncontrada
	}
	return nil
}
