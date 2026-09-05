package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// JustificanteRespuestaRecibida conserva los dos recibos originales. La
// consulta nominal acredita su procedencia de CT, no verifica el correo,
// recalcula su firma ni convierte la declaración en una aceptación o plazo.
type JustificanteRespuestaRecibida struct {
	Respuesta RespuestaRecibidaRegistrada
	Seleccion ReciboSolicitudLlamamientoBolsa
}

func (j JustificanteRespuestaRecibida) ValidarPara(s SolicitudResolverLlamamiento) error {
	r, seleccion := j.Respuesta, j.Seleccion
	if s.Validar() != nil || s.VersionEsperada != 2 ||
		(s.Respuesta != RespuestaLlamamientoAceptada && s.Respuesta != RespuestaLlamamientoRenunciada) ||
		r.ValidarPara(r.Solicitud) != nil || r.Estado != EstadoRespuestaRecibidaRegistrada ||
		r.JustificanteRef != s.PruebaRespuestaRef || r.Solicitud.Respuesta != s.Respuesta ||
		r.Solicitud.OrganizacionRef != s.OrganizacionRef || r.Solicitud.ExpedienteRef != s.ExpedienteRef ||
		r.Solicitud.LlamamientoRef != s.LlamamientoRef || r.Solicitud.ComunicacionRef != s.ComunicacionRef ||
		r.Solicitud.VersionComunicacionEsperada != s.VersionEsperada ||
		!seleccion.PropuestaGenerada || seleccion.VersionExpediente != 6 ||
		seleccion.OrganizacionRef != s.OrganizacionRef || seleccion.ExpedienteRef != s.ExpedienteRef ||
		seleccion.LlamamientoRef != s.LlamamientoRef || seleccion.SeleccionRef.Validar() != nil ||
		seleccion.OrdenSeleccionado == 0 || seleccion.OrdenSeleccionado > MaximoElementosIntegracionBolsa ||
		!instanteBolsaCanonico(seleccion.ConfirmadaEn) || seleccion.ConfirmadaEn.After(r.RegistradaEn) ||
		!seleccion.Procedencia.validarNominal() || seleccion.ConfirmadaEn.After(seleccion.Procedencia.Evidencia.EmitidaEn) {
		return ErrResultadoRespuestaRecibidaNoConfiable
	}
	for _, ref := range []string{seleccion.OperacionRef, seleccion.CorrelacionRef,
		seleccion.ReciboRef, seleccion.AuditoriaRef, seleccion.EventoRef} {
		if !domain.ReferenciaOpacaValida(ref) {
			return ErrResultadoRespuestaRecibidaNoConfiable
		}
	}
	for _, ref := range []ReferenciaVersionadaIntegracionBolsa{seleccion.Necesidad, seleccion.Bolsa,
		seleccion.Orden, seleccion.Politica, seleccion.Resultado, seleccion.Propuesta,
		seleccion.AccionEvento, seleccion.RetencionSeleccion} {
		if ref.Validar() != nil {
			return ErrResultadoRespuestaRecibidaNoConfiable
		}
	}
	return nil
}

// LectorJustificantesRespuestaRecibida exige permiso fresco sobre la solicitud
// completa en cada consulta. Recupera el antecedente original, sin registrar
// otra respuesta ni resolver el llamamiento. Las referencias no son autoridad.
type LectorJustificantesRespuestaRecibida interface {
	ConsultarJustificanteRespuestaRecibida(context.Context, SolicitudResolverLlamamiento) (JustificanteRespuestaRecibida, error)
}
