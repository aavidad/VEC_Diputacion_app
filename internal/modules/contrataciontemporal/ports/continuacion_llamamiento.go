package ports

import (
	"context"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var (
	ErrOperacionContinuacionInvalida     = errors.New("contratacion temporal: solicitud de continuacion invalida")
	ErrOperacionContinuacionConflicto    = errors.New("contratacion temporal: continuacion en conflicto")
	ErrOperacionContinuacionDenegada     = errors.New("contratacion temporal: continuacion denegada")
	ErrOperacionContinuacionNoDisponible = errors.New("contratacion temporal: continuacion no disponible")
)

// SolicitudContinuarLlamamiento no contiene actor, candidato, orden ni permisos.
// Su JSON de cinco campos forma parte del material autorizado, sin etiquetas.
type SolicitudContinuarLlamamiento struct {
	ClaveIdempotencia string
	OrganizacionRef   string
	ExpedienteRef     string
	ResolucionRef     string
	IntencionRef      string
}

func (s SolicitudContinuarLlamamiento) Validar() error {
	if !ClaveIdempotenciaValida(s.ClaveIdempotencia) || len(s.ClaveIdempotencia) != 36 ||
		s.ClaveIdempotencia[14] != '4' || !strings.ContainsRune("89ab", rune(s.ClaveIdempotencia[19])) {
		return ErrOperacionContinuacionInvalida
	}
	for _, ref := range []string{s.OrganizacionRef, s.ExpedienteRef, s.ResolucionRef, s.IntencionRef} {
		if !domain.ReferenciaOpacaValida(ref) {
			return ErrOperacionContinuacionInvalida
		}
	}
	return nil
}

// ComandoSiguienteLlamamiento es la carga durable CT59, no una capacidad Bolsa.
// Se abre con permiso propio; la composición reacredita los antecedentes.
type ComandoSiguienteLlamamiento struct {
	Esquema         string `json:"esquema"`
	ComandoRef      string `json:"comando_ref"`
	IntencionRef    string `json:"intencion_ref"`
	OrganizacionRef string `json:"organizacion_ref"`
	ExpedienteRef   string `json:"expediente_ref"`
	LlamamientoRef  string `json:"llamamiento_ref"`
	JustificanteRef string `json:"justificante_ref"`
	SeleccionClave  string `json:"seleccion_clave"`
}

// Seleccion solo acompaña a una expiración confirmada: sin justificante, CT119
// devuelve el recibo de selección original de la misma comunicación. En una
// renuncia se consulta con el justificante y aquí debe venir vacía.
// NoIncorporacion solo acompaña a una aceptación seguida de no incorporación
// (CT124): la intención y el comando son los de la no incorporación.
type AntecedenteContinuacionLlamamiento struct {
	Resolucion          ResultadoResolucionLlamamiento
	ComandoSiguienteRef string
	ComandoSiguiente    ComandoSiguienteLlamamiento
	Seleccion           *ReciboSolicitudLlamamientoBolsa `json:",omitempty"`
	NoIncorporacion     *AntecedenteNoIncorporacion      `json:",omitempty"`
}

// EsExpiracion indica que el antecedente es la expiración confirmada por RRHH.
func (a AntecedenteContinuacionLlamamiento) EsExpiracion() bool {
	return a.Resolucion.Solicitud.Respuesta == RespuestaLlamamientoExpirada
}

// EsNoIncorporacion indica que el antecedente es una aceptación seguida de
// una no incorporación registrada por RRHH.
func (a AntecedenteContinuacionLlamamiento) EsNoIncorporacion() bool {
	return a.Resolucion.Solicitud.Respuesta == RespuestaLlamamientoAceptada && a.NoIncorporacion != nil
}

func (a AntecedenteContinuacionLlamamiento) ValidarPara(s SolicitudContinuarLlamamiento) error {
	r, c := a.Resolucion, a.ComandoSiguiente
	intencion, comando := r.IntencionSiguiente.IntencionRef, r.IntencionSiguiente.ComandoOpacoRef
	expiracion := a.EsExpiracion()
	if a.NoIncorporacion != nil {
		n := a.NoIncorporacion
		if !a.EsNoIncorporacion() || a.Seleccion != nil || !n.Valido() || r.EstadoPlazo != PlazoLlamamientoVigente ||
			r.IntencionSiguiente != (IntencionOutboxSiguienteCandidato{}) || n.RegistradaEn.Before(r.ResueltaEn) {
			return ErrOperacionContinuacionNoDisponible
		}
		intencion, comando = n.IntencionRef, n.ComandoRef
	} else if expiracion {
		if r.EstadoPlazo != PlazoLlamamientoExpirado || a.Seleccion == nil ||
			SeleccionOriginalValidaPara(*a.Seleccion, r.Solicitud.OrganizacionRef, r.Solicitud.ExpedienteRef, r.Solicitud.LlamamientoRef) != nil {
			return ErrOperacionContinuacionNoDisponible
		}
	} else if r.Solicitud.Respuesta != RespuestaLlamamientoRenunciada || a.Seleccion != nil {
		return ErrOperacionContinuacionNoDisponible
	}
	if s.Validar() != nil || r.ValidarPara(r.Solicitud) != nil ||
		r.Estado != ResultadoComunicacionLlamamientoConfirmado || !r.Solicitud.RevisionManualConfirmada() ||
		r.Solicitud.OrganizacionRef != s.OrganizacionRef || r.Solicitud.ExpedienteRef != s.ExpedienteRef ||
		r.ResolucionRef != s.ResolucionRef || intencion != s.IntencionRef ||
		c.Esquema != "vec.contratacion-temporal.siguiente-candidato.intencion.v1" ||
		c.ComandoRef != a.ComandoSiguienteRef || c.ComandoRef != comando ||
		c.IntencionRef != s.IntencionRef || c.OrganizacionRef != s.OrganizacionRef || c.ExpedienteRef != s.ExpedienteRef ||
		c.LlamamientoRef != r.Solicitud.LlamamientoRef || c.JustificanteRef != r.Solicitud.PruebaRespuestaRef ||
		!ClaveIdempotenciaValida(c.SeleccionClave) {
		return ErrOperacionContinuacionNoDisponible
	}
	return nil
}

// SeleccionOriginalValidaPara comprueba la forma del recibo de selección
// original ligado a organización, expediente y llamamiento. No acredita por sí
// mismo la apertura en Bolsa: la composición la coteja con el registro durable.
func SeleccionOriginalValidaPara(seleccion ReciboSolicitudLlamamientoBolsa, organizacion, expediente, llamamiento string) error {
	if !seleccion.PropuestaGenerada || seleccion.VersionExpediente < 6 ||
		seleccion.VersionExpediente > MaximoEnteroSeguroIntegracionBolsa ||
		seleccion.OrganizacionRef != organizacion || seleccion.ExpedienteRef != expediente ||
		seleccion.LlamamientoRef != llamamiento || seleccion.SeleccionRef.Validar() != nil ||
		seleccion.OrdenSeleccionado == 0 || seleccion.OrdenSeleccionado > MaximoElementosIntegracionBolsa ||
		!instanteBolsaCanonico(seleccion.ConfirmadaEn) || !seleccion.Procedencia.validarNominal() ||
		seleccion.ConfirmadaEn.After(seleccion.Procedencia.Evidencia.EmitidaEn) {
		return ErrOperacionContinuacionNoDisponible
	}
	for _, ref := range []string{seleccion.OperacionRef, seleccion.CorrelacionRef,
		seleccion.ReciboRef, seleccion.AuditoriaRef, seleccion.EventoRef} {
		if !domain.ReferenciaOpacaValida(ref) {
			return ErrOperacionContinuacionNoDisponible
		}
	}
	for _, ref := range []ReferenciaVersionadaIntegracionBolsa{seleccion.Necesidad, seleccion.Bolsa,
		seleccion.Orden, seleccion.Politica, seleccion.Resultado, seleccion.Propuesta,
		seleccion.AccionEvento, seleccion.RetencionSeleccion} {
		if ref.Validar() != nil {
			return ErrOperacionContinuacionNoDisponible
		}
	}
	return nil
}

// ReciboBolsaContinuacion es una proyección interna de un recibo ya confirmado
// y comprobado por el puente propietario. Referencias solas no prueban efectos;
// no se acepta desde HTTP ni se usa para decidir elegibilidad o plazos.
type ReciboBolsaContinuacion struct {
	IntencionRef         string
	TerminalOperacionRef string
	OperacionRef         string
	LlamamientoRef       string
	PropuestaRef         string
	ReciboRef            string
	AuditoriaRef         string
	EventoRef            string
	RegistroSHA256       string
	ConfirmadaEn         time.Time
}

func (r ReciboBolsaContinuacion) ValidarPara(s SolicitudContinuarLlamamiento) error {
	if s.Validar() != nil || r.IntencionRef != s.IntencionRef || r.OperacionRef == r.TerminalOperacionRef ||
		!patronHuellaEfectoAlta.MatchString(r.RegistroSHA256) || r.RegistroSHA256 == strings.Repeat("0", 64) ||
		!domain.InstanteUTCCanonico(r.ConfirmadaEn) {
		return ErrOperacionContinuacionNoDisponible
	}
	for _, ref := range []string{r.TerminalOperacionRef, r.OperacionRef, r.LlamamientoRef, r.PropuestaRef, r.ReciboRef, r.AuditoriaRef, r.EventoRef} {
		if !domain.ReferenciaOpacaValida(ref) {
			return ErrOperacionContinuacionNoDisponible
		}
	}
	return nil
}

type MaterialContinuacionLlamamiento struct {
	Etapa       string
	Solicitud   SolicitudContinuarLlamamiento
	ReciboBolsa *ReciboBolsaContinuacion `json:",omitempty"`
}

func (m MaterialContinuacionLlamamiento) Validar() error {
	if m.Solicitud.Validar() != nil {
		return ErrOperacionContinuacionInvalida
	}
	switch m.Etapa {
	case "consulta":
		if m.ReciboBolsa == nil {
			return nil
		}
	case "confirmacion":
		if m.ReciboBolsa != nil {
			return m.ReciboBolsa.ValidarPara(m.Solicitud)
		}
	}
	return ErrOperacionContinuacionInvalida
}

// ResultadoContinuacionLlamamiento acredita exclusivamente la confirmación
// local del recibo Bolsa. Al recuperar solo cambia Estado; no es correo
// enviado ni cambia el recibo original de la renuncia o la expiración.
type ResultadoContinuacionLlamamiento struct {
	Solicitud              SolicitudContinuarLlamamiento
	LlamamientoAnteriorRef string
	ReciboBolsa            ReciboBolsaContinuacion
	ReciboRef              string
	AuditoriaRef           string
	ConfirmadaEn           time.Time
	Estado                 string
}

func (r ResultadoContinuacionLlamamiento) ValidarPara(s SolicitudContinuarLlamamiento) error {
	if s.Validar() != nil || r.Solicitud != s || r.ReciboBolsa.ValidarPara(s) != nil ||
		!domain.ReferenciaOpacaValida(r.LlamamientoAnteriorRef) || r.LlamamientoAnteriorRef == r.ReciboBolsa.LlamamientoRef ||
		!domain.ReferenciaOpacaValida(r.ReciboRef) || !domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		!domain.InstanteUTCCanonico(r.ConfirmadaEn) || r.ConfirmadaEn.Before(r.ReciboBolsa.ConfirmadaEn) || (r.Estado != "confirmado" && r.Estado != "replay_confirmado") {
		return ErrOperacionContinuacionNoDisponible
	}
	return nil
}

type RegistroContinuacionLlamamiento interface {
	LeerAntecedente(context.Context, SolicitudContinuarLlamamiento) (AntecedenteContinuacionLlamamiento, error)
	Confirmar(context.Context, SolicitudContinuarLlamamiento, ReciboBolsaContinuacion) (ResultadoContinuacionLlamamiento, error)
}
