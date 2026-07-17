package ports

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var (
	patronReferenciaPanel = regexp.MustCompile(`^[a-z]{3}_[a-z0-9]{16,80}$`)
	patronClavePanel      = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,79}$`)
)

func referenciaOpacaPanelValida(referencia, prefijo string) bool {
	return strings.HasPrefix(referencia, prefijo) && patronReferenciaPanel.MatchString(referencia)
}

func clavePanelValida(clave string) bool {
	return patronClavePanel.MatchString(clave)
}

func selectorClaseCanonica(clase ClaseAmbitoPanelInterno) string { return string(clase) }

func enteroDecimal(valor int) string { return strconv.Itoa(valor) }

func instantePanelCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Equal(instante.Truncate(time.Microsecond))
}

func validarSolicitudConsultaPanelInterno(datos datosSolicitudConsultaPanelInterno) error {
	recurso, errRecurso := RecursoAutorizablePanelInterno(datos.selector, datos.motivo)
	datosAutorizacion, errAutorizacion := datos.autorizacion.Datos()
	correlacionRef, errCorrelacion := datos.correlacion.ValorCanonico()
	if errRecurso != nil || errAutorizacion != nil || errCorrelacion != nil ||
		!instantePanelCanonico(datos.consultadaEn) ||
		datos.autorizacion.ValidarEn(datos.consultadaEn) != nil ||
		datos.autorizacion.ValidarMotivo(datos.motivo) != nil {
		return ErrConsultaPanelInternoInvalida
	}
	decision := datosAutorizacion.Decision
	huellaContexto, errHuella := recurso.HuellaContextoAutorizacionSHA256()
	datosVinculo, errVinculo := decision.VinculoAutenticacionActor.Datos()
	if errHuella != nil || errVinculo != nil ||
		decision.Accion != AccionConsultarPanelInterno ||
		decision.RecursoRef != recurso.Referencia ||
		decision.ModuloID != recurso.ModuloID || decision.TipoRecurso != recurso.Tipo ||
		decision.ContextoRecursoHuellaSHA256 != huellaContexto ||
		decision.Finalidad != FinalidadPanelInternoBolsa ||
		decision.CorrelacionRef != correlacionRef ||
		decision.GarantiaMinima != dominiovec.AuthAssuranceHigh ||
		datosVinculo.GarantiaObservada != dominiovec.AuthAssuranceHigh ||
		!superficiePanelInternoValida(datosVinculo.Superficie) ||
		len(decision.Obligaciones) != 0 ||
		len(decision.CamposPermitidos) != 1 ||
		decision.CamposPermitidos[0] != CampoPanelInternoAgregado {
		return ErrConsultaPanelInternoInvalida
	}
	return nil
}

func superficiePanelInternoValida(superficie dominiovec.SuperficieAutenticacionActorV1) bool {
	return superficie == dominiovec.SuperficieAutenticacionInternaCorporativaV1 ||
		superficie == dominiovec.SuperficieAutenticacionAdministracionPrivilegiadaV1
}

func validarInstantaneaPanelInterno(
	instantanea InstantaneaPanelInterno,
	solicitud SolicitudConsultaPanelInterno,
) error {
	selector, errSelector := solicitud.Selector()
	autorizacion, errAutorizacion := solicitud.Autorizacion()
	datosAutorizacion, errDatosAutorizacion := autorizacion.Datos()
	correlacion, errCorrelacion := solicitud.Correlacion()
	correlacionRef, errCorrelacionRef := correlacion.ValorCanonico()
	consultadaEn, errInstante := solicitud.ConsultadaEn()
	if errSelector != nil || errAutorizacion != nil || errDatosAutorizacion != nil ||
		errCorrelacion != nil || errCorrelacionRef != nil || errInstante != nil {
		return ErrResultadoPanelInternoInvalido
	}
	if selector.Validar() != nil || instantanea.Selector != selector ||
		instantanea.Esquema != EsquemaPanelInternoBolsaV1 ||
		instantanea.Origen.Demostracion ||
		!referenciaOpacaPanelValida(instantanea.Origen.Revision, "rev_") ||
		!instantePanelCanonico(instantanea.Origen.ActualizadaEn) ||
		instantanea.Origen.ActualizadaEn.After(consultadaEn) ||
		!referenciaOpacaPanelValida(instantanea.PruebaLectura.LecturaRef, "lec_") ||
		!referenciaOpacaPanelValida(instantanea.PruebaLectura.AuditoriaRef, "aud_") ||
		instantanea.PruebaLectura.AuditoriaSecuencia == 0 ||
		instantanea.PruebaLectura.DecisionRef != datosAutorizacion.Decision.DecisionRef ||
		instantanea.PruebaLectura.HuellaDecisionSHA256 != datosAutorizacion.HuellaDecisionSHA256 ||
		instantanea.PruebaLectura.CorrelacionRef != correlacionRef ||
		!instantePanelCanonico(instantanea.PruebaLectura.ConfirmadaEn) ||
		instantanea.PruebaLectura.ConfirmadaEn.Before(consultadaEn) ||
		instantanea.PruebaLectura.ConfirmadaEn.After(consultadaEn.Add(5*time.Minute)) ||
		len(instantanea.Convocatorias) > maximoConvocatoriasPanel ||
		len(instantanea.ActuacionesPendientes) > maximoActuacionesPanel ||
		!indicadoresPanelValidos(instantanea.Indicadores) {
		return ErrResultadoPanelInternoInvalido
	}
	convocatoriasVistas := make(map[string]struct{}, len(instantanea.Convocatorias))
	for _, convocatoria := range instantanea.Convocatorias {
		if !referenciaOpacaPanelValida(convocatoria.ConvocatoriaRef, "cnv_") ||
			!clavePanelValida(convocatoria.CategoriaClave) ||
			!clavePanelValida(convocatoria.EstadoClave) ||
			(!convocatoria.PlazoCierraEn.IsZero() && !instantePanelCanonico(convocatoria.PlazoCierraEn)) ||
			!contadorPanelValido(convocatoria.NumeroSolicitudes) ||
			!contadorPanelValido(convocatoria.NumeroPendientes) ||
			convocatoria.NumeroPendientes > convocatoria.NumeroSolicitudes {
			return ErrResultadoPanelInternoInvalido
		}
		if _, repetida := convocatoriasVistas[convocatoria.ConvocatoriaRef]; repetida {
			return ErrResultadoPanelInternoInvalido
		}
		convocatoriasVistas[convocatoria.ConvocatoriaRef] = struct{}{}
	}
	actuacionesVistas := make(map[string]struct{}, len(instantanea.ActuacionesPendientes))
	for _, actuacion := range instantanea.ActuacionesPendientes {
		if !referenciaOpacaPanelValida(actuacion.ActuacionRef, "act_") ||
			!patronReferenciaPanel.MatchString(actuacion.RecursoRef) ||
			!clavePanelValida(actuacion.TipoClave) || !clavePanelValida(actuacion.EstadoClave) ||
			!clavePanelValida(actuacion.PrioridadClave) ||
			(!actuacion.FechaLimite.IsZero() && !instantePanelCanonico(actuacion.FechaLimite)) ||
			!contadorPanelValido(actuacion.NumeroElementos) {
			return ErrResultadoPanelInternoInvalido
		}
		if _, repetida := actuacionesVistas[actuacion.ActuacionRef]; repetida {
			return ErrResultadoPanelInternoInvalido
		}
		actuacionesVistas[actuacion.ActuacionRef] = struct{}{}
	}
	return nil
}

func contadorPanelValido(valor int) bool { return valor >= 0 && valor <= maximoContadorPanel }

func indicadoresPanelValidos(i IndicadoresPanelInterno) bool {
	valores := [...]int{
		i.ConvocatoriasBorrador, i.ConvocatoriasRevision, i.ConvocatoriasPendientesFirma,
		i.ConvocatoriasPublicadas, i.BolsasActivas, i.BolsasSuspendidas, i.BolsasAgotadas,
		i.LlamamientosPendientes, i.LlamamientosEnCurso, i.LlamamientosVencenHoy,
		i.DocumentosPendientesFirma, i.IncidenciasAbiertas,
	}
	for _, valor := range valores {
		if !contadorPanelValido(valor) {
			return false
		}
	}
	return true
}
