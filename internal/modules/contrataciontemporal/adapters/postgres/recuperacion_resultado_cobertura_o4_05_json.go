package postgres

import (
	"encoding/json"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	esquemaResultadoRecuperacionResultadoCoberturaO405 = "" +
		"vec.contratacion-temporal.resultado-recuperacion-propia-" +
		"decision-cobertura.o4-05.v1"
	estadoNoObservableRecuperacionResultadoCoberturaO405   = "no_observable"
	estadoConfirmadoRecuperacionResultadoCoberturaO405     = "confirmado"
	maximoBytesResultadoRecuperacionResultadoCoberturaO405 = 256 * 1024
)

var (
	clavesResultadoNoObservableRecuperacionResultadoCoberturaO405 = []string{
		"esquema",
		"estado",
		"observada_en",
	}
	clavesResultadoConfirmadoRecuperacionResultadoCoberturaO405 = []string{
		"esquema",
		"estado",
		"organizacion_ref",
		"expediente_ref",
		"version_expediente",
		"reserva_ref",
		"recibo_ref",
		"actuacion_ref",
		"auditoria_ref",
		"evento_ref",
		"correlacion_vec_ref",
		"decision_vec_ref",
		"ambito_idempotencia_hmac",
		"huella_semantica_hmac",
		"revision_cercado",
		"observada_en_db",
		"recibo",
		"observada_en",
	}
)

type cabeceraResultadoRecuperacionResultadoCoberturaO405 struct {
	Estado string `json:"estado"`
}

type resultadoNoObservableRecuperacionResultadoCoberturaO405 struct {
	Esquema     string    `json:"esquema"`
	Estado      string    `json:"estado"`
	ObservadaEn time.Time `json:"observada_en"`
}

type resultadoConfirmadoRecuperacionResultadoCoberturaO405 struct {
	Esquema                string                       `json:"esquema"`
	Estado                 string                       `json:"estado"`
	OrganizacionRef        string                       `json:"organizacion_ref"`
	ExpedienteRef          string                       `json:"expediente_ref"`
	VersionExpediente      uint64                       `json:"version_expediente"`
	ReservaRef             string                       `json:"reserva_ref"`
	ReciboRef              string                       `json:"recibo_ref"`
	ActuacionRef           string                       `json:"actuacion_ref"`
	AuditoriaRef           string                       `json:"auditoria_ref"`
	EventoRef              string                       `json:"evento_ref"`
	CorrelacionVECRef      string                       `json:"correlacion_vec_ref"`
	DecisionVECRef         string                       `json:"decision_vec_ref"`
	AmbitoIdempotenciaHMAC string                       `json:"ambito_idempotencia_hmac"`
	HuellaSemanticaHMAC    string                       `json:"huella_semantica_hmac"`
	RevisionCercado        uint64                       `json:"revision_cercado"`
	ObservadaEnDB          time.Time                    `json:"observada_en_db"`
	Recibo                 reciboDecisionCoberturaO404E `json:"recibo"`
	ObservadaEn            time.Time                    `json:"observada_en"`
}

func decodificarResultadoRecuperacionResultadoCoberturaO405(
	contenido []byte,
) (
	cobertura.DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
	error,
) {
	if len(contenido) == 0 ||
		len(contenido) > maximoBytesResultadoRecuperacionResultadoCoberturaO405 ||
		validarJSONSinDuplicadosDecisionCoberturaO404E(
			contenido,
			maximaProfundidadJSONDecisionCoberturaO404E,
		) != nil {
		return resultadoNoConfiableRecuperacionResultadoCoberturaO405()
	}
	var objeto map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &objeto); err != nil || objeto == nil {
		return resultadoNoConfiableRecuperacionResultadoCoberturaO405()
	}
	var cabecera cabeceraResultadoRecuperacionResultadoCoberturaO405
	if err := json.Unmarshal(contenido, &cabecera); err != nil {
		return resultadoNoConfiableRecuperacionResultadoCoberturaO405()
	}
	switch cabecera.Estado {
	case estadoNoObservableRecuperacionResultadoCoberturaO405:
		return decodificarNoObservableRecuperacionResultadoCoberturaO405(
			contenido,
		)
	case estadoConfirmadoRecuperacionResultadoCoberturaO405:
		return decodificarConfirmadoRecuperacionResultadoCoberturaO405(
			contenido,
			objeto,
		)
	default:
		return resultadoNoConfiableRecuperacionResultadoCoberturaO405()
	}
}

func decodificarNoObservableRecuperacionResultadoCoberturaO405(
	contenido []byte,
) (
	cobertura.DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
	error,
) {
	var dto resultadoNoObservableRecuperacionResultadoCoberturaO405
	_, err := decodificarObjetoJSONExactoDecisionCoberturaO404E(
		contenido,
		clavesResultadoNoObservableRecuperacionResultadoCoberturaO405,
		&dto,
	)
	if err != nil ||
		dto.Esquema != esquemaResultadoRecuperacionResultadoCoberturaO405 ||
		dto.Estado != estadoNoObservableRecuperacionResultadoCoberturaO405 ||
		!domain.InstanteUTCCanonico(dto.ObservadaEn) {
		return resultadoNoConfiableRecuperacionResultadoCoberturaO405()
	}
	return cobertura.DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB{
		ObservadaEn: dto.ObservadaEn,
	}, nil
}

func decodificarConfirmadoRecuperacionResultadoCoberturaO405(
	contenido []byte,
	objeto map[string]json.RawMessage,
) (
	cobertura.DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
	error,
) {
	var dto resultadoConfirmadoRecuperacionResultadoCoberturaO405
	_, err := decodificarObjetoJSONExactoDecisionCoberturaO404E(
		contenido,
		clavesResultadoConfirmadoRecuperacionResultadoCoberturaO405,
		&dto,
	)
	if err != nil ||
		validarObjetoCrudoJSONExactoDecisionCoberturaO404E(
			objeto["recibo"],
			clavesReciboDecisionCoberturaO404E,
		) != nil ||
		!validarConfirmadoRecuperacionResultadoCoberturaO405(dto) {
		return resultadoNoConfiableRecuperacionResultadoCoberturaO405()
	}
	recibo, err := reciboNominalDecisionCoberturaO404E(dto.Recibo)
	if err != nil {
		return resultadoNoConfiableRecuperacionResultadoCoberturaO405()
	}
	reserva := cobertura.DatosReservaTerminalOperacionDecisionCobertura{
		OrganizacionRef:        dto.OrganizacionRef,
		ExpedienteRef:          dto.ExpedienteRef,
		VersionExpediente:      dto.VersionExpediente,
		ReservaRef:             dto.ReservaRef,
		ReciboRef:              dto.ReciboRef,
		ActuacionRef:           dto.ActuacionRef,
		AuditoriaRef:           dto.AuditoriaRef,
		EventoRef:              dto.EventoRef,
		CorrelacionVECRef:      dto.CorrelacionVECRef,
		DecisionVECRef:         dto.DecisionVECRef,
		AmbitoIdempotenciaHMAC: dto.AmbitoIdempotenciaHMAC,
		HuellaSemanticaHMAC:    dto.HuellaSemanticaHMAC,
		RevisionCercado:        dto.RevisionCercado,
		ObservadaEnDB:          dto.ObservadaEnDB,
	}
	return cobertura.DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB{
		Encontrado:  true,
		Reserva:     reserva,
		Recibo:      recibo,
		ObservadaEn: dto.ObservadaEn,
	}, nil
}

func validarConfirmadoRecuperacionResultadoCoberturaO405(
	dto resultadoConfirmadoRecuperacionResultadoCoberturaO405,
) bool {
	if dto.Esquema != esquemaResultadoRecuperacionResultadoCoberturaO405 ||
		dto.Estado != estadoConfirmadoRecuperacionResultadoCoberturaO405 ||
		!domain.ReferenciaOpacaValida(dto.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(dto.ExpedienteRef) ||
		dto.VersionExpediente < 2 ||
		dto.VersionExpediente >=
			cobertura.MaximoEnteroSeguroOperacionDecisionCobertura ||
		!domain.ReferenciaOpacaValida(dto.ReservaRef) ||
		!domain.ReferenciaOpacaValida(dto.ReciboRef) ||
		!domain.ReferenciaOpacaValida(dto.ActuacionRef) ||
		!domain.ReferenciaOpacaValida(dto.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(dto.EventoRef) ||
		!domain.ReferenciaOpacaValida(dto.CorrelacionVECRef) ||
		!domain.ReferenciaOpacaValida(dto.DecisionVECRef) ||
		!puertosct.SelloHMACSHA256Valido(dto.AmbitoIdempotenciaHMAC) ||
		!puertosct.SelloHMACSHA256Valido(dto.HuellaSemanticaHMAC) ||
		dto.RevisionCercado == 0 ||
		dto.RevisionCercado >
			cobertura.MaximoEnteroSeguroOperacionDecisionCobertura ||
		!domain.InstanteUTCCanonico(dto.ObservadaEnDB) ||
		!domain.InstanteUTCCanonico(dto.ObservadaEn) ||
		dto.ObservadaEn.Before(dto.ObservadaEnDB) ||
		!validarReciboDecisionCoberturaO404E(dto.Recibo) {
		return false
	}
	if dto.Recibo.ReciboRef != dto.ReciboRef ||
		dto.Recibo.ReservaRef != dto.ReservaRef ||
		dto.Recibo.AuditoriaRef != dto.AuditoriaRef ||
		dto.Recibo.CorrelacionVECRef != dto.CorrelacionVECRef ||
		dto.Recibo.DecisionVECRef != dto.DecisionVECRef ||
		dto.Recibo.AmbitoIdempotenciaHMAC != dto.AmbitoIdempotenciaHMAC ||
		dto.Recibo.HuellaSemanticaHMAC != dto.HuellaSemanticaHMAC ||
		dto.Recibo.RevisionCercado != dto.RevisionCercado ||
		dto.Recibo.ConfirmadaEn.Before(dto.ObservadaEnDB) ||
		dto.ObservadaEn.Before(dto.Recibo.ConfirmadaEn) {
		return false
	}
	if dto.Recibo.Aplicada {
		return dto.Recibo.VersionResultante == dto.VersionExpediente+1 &&
			dto.Recibo.EventoRef == dto.EventoRef &&
			dto.Recibo.ActuacionRef == dto.ActuacionRef
	}
	return true
}

func resultadoNoConfiableRecuperacionResultadoCoberturaO405() (
	cobertura.DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
	error,
) {
	return resultadoVacioRecuperacionResultadoCoberturaO405(),
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
}
