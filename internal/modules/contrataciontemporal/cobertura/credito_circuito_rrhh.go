package cobertura

import (
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func clonarEvidenciaCreditoCircuito(
	e *ports.EvidenciaCreditoCircuitoRRHH,
) *ports.EvidenciaCreditoCircuitoRRHH {
	if e == nil {
		return nil
	}
	copia := e.Clonar()
	return &copia
}

// adjuntarCreditoCircuitoDecisionCobertura conserva la misma versión, recibo
// y actuación de la decisión. El adaptador confirma agregado y auditoría en
// la transacción existente; no hay una escritura posterior del hito.
func adjuntarCreditoCircuitoDecisionCobertura(
	anterior, siguiente domain.Expediente,
	perfilRef string,
	evidencia *ports.EvidenciaCreditoCircuitoRRHH,
) (domain.Expediente, error) {
	if anterior.Circuito == nil {
		if evidencia != nil {
			return domain.Expediente{}, ErrOrdenOperacionDecisionCoberturaInvalida
		}
		return siguiente, nil
	}
	if len(siguiente.Actuaciones) == 0 {
		return domain.Expediente{}, ErrOrdenOperacionDecisionCoberturaInvalida
	}
	actuacion := siguiente.Actuaciones[len(siguiente.Actuaciones)-1]
	credito, err := anterior.DatosCreditoCircuitoRRHH()
	if err != nil {
		return domain.Expediente{}, ErrOrdenOperacionDecisionCoberturaInvalida
	}
	solicitud, err := ports.NuevaSolicitudEvidenciaCreditoCircuitoRRHH(
		credito, actuacion.ActorRef, perfilRef,
	)
	if err != nil || evidencia == nil || evidencia.ValidarPara(solicitud) != nil ||
		siguiente.Circuito == nil || siguiente.Version != anterior.Version+1 ||
		!domain.ReferenciaOpacaValida(perfilRef) {
		return domain.Expediente{}, ErrOrdenOperacionDecisionCoberturaInvalida
	}
	var transicion domain.TransicionCircuitoRRHH
	coincidencias := 0
	for _, candidata := range evidencia.Definicion.Transiciones {
		if candidata.Tipo == domain.HitoCreditoComprobado &&
			candidata.Origen == anterior.Circuito.EstadoActual {
			transicion = candidata
			coincidencias++
		}
	}
	if coincidencias != 1 || !transicion.RequiereDocumento ||
		transicion.RequiereFirma || transicion.AutorizanteCargoClave != "" {
		return domain.Expediente{}, ErrOrdenOperacionDecisionCoberturaInvalida
	}
	hito := domain.HitoCircuitoRRHH{
		Clave: transicion.Clave, ActuacionClave: actuacion.AccionClave,
		ActorRef: actuacion.ActorRef, PerfilClave: transicion.PerfilClave,
		PerfilRef: perfilRef, UnidadRef: actuacion.UnidadRef,
		DocumentoRef:          solicitud.DocumentoRef,
		DocumentoVersion:      evidencia.DocumentoVersion,
		HuellaDocumentoSHA256: evidencia.HuellaDocumentoSHA256,
		CreditoRef:            solicitud.CreditoRef,
		ReciboRef:             actuacion.ReciboRef, RegistradoEn: actuacion.RealizadaEn,
	}
	conHito, err := siguiente.AdjuntarHitosCircuito(
		evidencia.Definicion, siguiente.Version, []domain.HitoCircuitoRRHH{hito},
	)
	if err != nil {
		return domain.Expediente{}, ErrOrdenOperacionDecisionCoberturaInvalida
	}
	return conHito, nil
}
