package ports

import (
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var ErrResultadoOperacionAnalisisNoConfiable = errors.New(
	"contratacion temporal: resultado de operacion de analisis no confiable",
)

type ReciboOperacionAnalisis struct {
	Operacion              TipoOperacionAnalisis `json:"operacion"`
	OrganizacionRef        string                `json:"organizacion_ref"`
	ExpedienteRef          string                `json:"expediente_ref"`
	VersionAnterior        uint64                `json:"version_anterior"`
	VersionResultante      uint64                `json:"version_resultante"`
	SecuenciaActuacion     uint64                `json:"secuencia_actuacion"`
	ArtefactoRef           string                `json:"artefacto_ref"`
	ArtefactoHuellaSHA256  string                `json:"artefacto_huella_sha256"`
	ReciboRef              string                `json:"recibo_ref"`
	AuditoriaRef           string                `json:"auditoria_ref"`
	EventoRef              string                `json:"evento_ref"`
	ConsumoFuentesRef      string                `json:"consumo_fuentes_ref"`
	HuellaConsumoFuentes   string                `json:"huella_consumo_fuentes_sha256"`
	ConcesionV3DecisionRef string                `json:"concesion_v3_decision_ref"`
	HuellaSemanticaHMAC    string                `json:"huella_semantica_hmac"`
	ConfirmadaEn           time.Time             `json:"confirmada_en"`
}

func (r ReciboOperacionAnalisis) ValidarParaPreparacion(
	preparacion DatosPreparacionOperacionAnalisis,
) error {
	if !r.Operacion.Valida() ||
		r.Operacion != preparacion.Operacion ||
		r.OrganizacionRef != preparacion.OrganizacionRef ||
		r.ExpedienteRef != preparacion.ExpedienteRef ||
		r.VersionAnterior != preparacion.VersionExpediente ||
		!VersionOperacionAnalisisConIncrementoValida(r.VersionAnterior) ||
		r.VersionResultante != r.VersionAnterior+1 ||
		r.VersionResultante > MaximoEnteroSeguroOperacionAnalisis ||
		r.SecuenciaActuacion != r.VersionResultante ||
		r.ArtefactoRef != preparacion.ArtefactoRef ||
		r.ArtefactoHuellaSHA256 != preparacion.ArtefactoHuellaSHA256 ||
		r.ReciboRef != preparacion.ReciboRef ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(r.EventoRef) ||
		!domain.ReferenciaOpacaValida(r.ConsumoFuentesRef) ||
		!huellaSHA256OperacionAnalisisValida(
			r.HuellaConsumoFuentes,
		) ||
		!domain.ReferenciaOpacaValida(r.ConcesionV3DecisionRef) ||
		!sellosHMACIguales(
			r.HuellaSemanticaHMAC,
			preparacion.HuellaSemanticaHMAC,
		) ||
		!instanteSeguroOperacionAnalisis(r.ConfirmadaEn) {
		return ErrResultadoOperacionAnalisisNoConfiable
	}
	return nil
}

func (r ReciboOperacionAnalisis) ValidarParaConsulta(
	solicitud SolicitudConsultarOperacionAnalisisConfirmada,
) error {
	if solicitud.Validar() != nil ||
		r.Operacion != solicitud.Operacion ||
		r.OrganizacionRef != solicitud.OrganizacionRef ||
		r.ExpedienteRef != solicitud.ExpedienteRef ||
		r.VersionAnterior != solicitud.VersionExpediente ||
		r.ArtefactoRef != solicitud.ArtefactoRef ||
		!huellaSHA256OperacionAnalisisValida(
			r.ArtefactoHuellaSHA256,
		) ||
		!domain.ReferenciaOpacaValida(r.ReciboRef) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(r.EventoRef) ||
		!domain.ReferenciaOpacaValida(r.ConsumoFuentesRef) ||
		!huellaSHA256OperacionAnalisisValida(
			r.HuellaConsumoFuentes,
		) ||
		!domain.ReferenciaOpacaValida(r.ConcesionV3DecisionRef) ||
		!SelloHMACSHA256Valido(r.HuellaSemanticaHMAC) ||
		!instanteSeguroOperacionAnalisis(r.ConfirmadaEn) ||
		r.VersionResultante != r.VersionAnterior+1 ||
		r.SecuenciaActuacion != r.VersionResultante {
		return ErrResultadoOperacionAnalisisNoConfiable
	}
	return nil
}

func (r ReciboOperacionAnalisis) ValidarParaOrdenDentroDeTransaccion(
	orden OrdenConfirmarOperacionAnalisis,
) error {
	evidencia, err := orden.Datos()
	if err != nil ||
		orden.ValidarConfirmacionDentroDeTransaccion(r.ConfirmadaEn) != nil {
		return ErrResultadoOperacionAnalisisNoConfiable
	}
	preparacion, err := evidencia.Preparacion.DatosPara(
		evidencia.SolicitudPreparacion,
	)
	datosConsumo, errConsumo := evidencia.OrdenConsumoFuentes.Datos()
	confirmacionV3, errConfirmacion := evidencia.ConfirmacionV3.Datos()
	if err != nil || errConsumo != nil || errConfirmacion != nil ||
		r.ValidarParaPreparacion(preparacion) != nil ||
		r.HuellaConsumoFuentes != datosConsumo.HuellaSHA256 ||
		r.ConcesionV3DecisionRef != confirmacionV3.DecisionRef {
		return ErrResultadoOperacionAnalisisNoConfiable
	}
	return nil
}
