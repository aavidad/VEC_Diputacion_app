package bootstrap

import (
	"context"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// PrepararSolicitudConsultarDatosContacto revalida el contexto Bolsa. La
// consulta completa (`?ver=completo`) lleva además el vínculo nominal, el
// motivo propio de la consulta y la correlación de la petición, que comparten
// la decisión V3, su consumo y, si falla, el intento AD169.
func (p *preparadorBorradorLlamamientoDesarrollo) PrepararSolicitudConsultarDatosContacto(ctx context.Context, bolsaRef, participacionRef string, completo bool) (puertosbolsa.SolicitudConsultarDatosContactoParticipacion, error) {
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return puertosbolsa.SolicitudConsultarDatosContactoParticipacion{}, err
	}
	solicitud := puertosbolsa.SolicitudConsultarDatosContactoParticipacion{ContextoActor: contexto.Resultado.Contexto, BolsaRef: bolsaRef, ParticipacionRef: participacionRef}
	correlacion, err := puertosvec.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return puertosbolsa.SolicitudConsultarDatosContactoParticipacion{}, err
	}
	solicitud.Vinculo, solicitud.ResultadoContexto, solicitud.Completo = contexto.Vinculo, contexto.Resultado, completo
	solicitud.Correlacion, solicitud.MotivoAutorizacion = correlacion, motivoConsultarDatosContactoParticipacionBolsaDesarrollo()
	if !completo {
		if p.fuenteRespuestas == nil {
			return solicitud, puertosbolsa.ErrDatosContactoParticipacionNoDisponibles
		}
		resuelto, err := p.ResolverContextoSituacionParticipacion(ctx, solicitud.ContextoActor, bolsaRef, participacionRef)
		if err != nil {
			return solicitud, err
		}
		contrato := contratoLecturaBolsa{
			accion:    puertosbolsa.AccionConsultarOrigenDatosContactoParticipacion,
			finalidad: puertosbolsa.FinalidadConsultarOrigenDatosContactoParticipacion,
			tipo:      puertosbolsa.TipoRecursoSituacionParticipacion,
			motivo:    "lectura_origen_datos_contacto_correcta",
			campos:    puertosbolsa.CamposConsultarOrigenDatosContacto(),
		}
		solicitud.Captura, err = capturarLecturaBolsa(ctx, p.fuenteRespuestas,
			contexto.Vinculo, contexto.Resultado, resuelto.UnidadRef, resuelto.AmbitoRef,
			bolsaRef, participacionRef, contrato)
		if err != nil {
			return solicitud, err
		}
	}
	return solicitud, nil
}

// motivoConsultarDatosContactoParticipacionBolsaDesarrollo es la entrada
// propia de la consulta completa, en un catálogo versionado de motivos aparte:
// así no se publica otra versión del catálogo de registro ya en uso.
func motivoConsultarDatosContactoParticipacionBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_consulta_datos_contacto_participacion_bolsa", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-bolsa-b4-consulta-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-b4-datos-contacto-consultar")}
}
