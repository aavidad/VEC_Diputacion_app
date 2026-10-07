package ports

import (
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	TipoEventoOrigenRelacionUnidadCT = "contratacion_temporal.asignacion_confirmada"
	maximaVersionRelacionUnidadCT    = uint64(9007199254740991)
)

var (
	ErrRelacionUnidadExpedienteCTInvalida     = errors.New("contratacion_temporal.relacion_unidad_expediente.invalida")
	ErrRelacionUnidadExpedienteCTNoDisponible = errors.New("contratacion_temporal.relacion_unidad_expediente.no_disponible")
	ErrRelacionUnidadExpedienteCTDivergente   = errors.New("contratacion_temporal.relacion_unidad_expediente.divergente")
)

// La unidad esperada sólo permite comparar con la fuente CT. No procede del
// actor ni concede autoridad. La versión solicitada identifica el snapshot
// histórico usado por el documento; puede ser anterior a la versión actual.
type SolicitudRelacionUnidadExpedienteCT struct {
	OrganizacionRef             string
	ExpedienteRef               string
	UnidadRefEsperada           string
	VersionExpedienteSolicitada uint64
}

func (q SolicitudRelacionUnidadExpedienteCT) Validar() error {
	if !domain.ReferenciaOpacaValida(q.OrganizacionRef) || !domain.ReferenciaOpacaValida(q.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(q.UnidadRefEsperada) || !versionRelacionUnidadCTValida(q.VersionExpedienteSolicitada) {
		return ErrRelacionUnidadExpedienteCTInvalida
	}
	return nil
}

// RelacionUnidadExpedienteCT transporta hechos contrastados por el propietario
// CT desde CT050. La unidad actual y la histórica proceden de asignacion.unidad_ref.
// OperacionOrigenRef y ReservaAsignacionRef conservan sus semánticas separadas;
// CT050 guarda la referencia de reserva como operación de origen.
//
// PruebaSnapshotOrigenHuellaSHA256 es la huella ya persistida de la prueba del
// snapshot que confirmó la asignación. No es una huella dedicada del par
// expediente/unidad. La fecha es reserva.confirmada_en, contrastada con terminal.
// Esta estructura no acredita autorización, auditoría, bloqueo ni ejecución K.
type RelacionUnidadExpedienteCT struct {
	Solicitud                        SolicitudRelacionUnidadExpedienteCT
	UnidadRef                        string
	UnidadSnapshotSolicitadoRef      string
	VersionExpedienteActual          uint64
	VersionOrigenVinculo             uint64
	OperacionOrigenRef               string
	ReservaAsignacionRef             string
	ReciboAsignacionRef              string
	PruebaSnapshotOrigenHuellaSHA256 string
	TipoEventoOrigen                 string
	EventoAsignacionRef              string
	EventoPayloadHuellaSHA256        string
	AsignacionConfirmadaEn           time.Time
}

// ValidarPara comprueba consistencia estructural y eco del recurso exacto.
// La fuente SQL debe contrastar reserva/terminal confirmados, snapshot de
// origen, actuación y evento, con V3 y auditoría dentro del efecto real.
func (r RelacionUnidadExpedienteCT) ValidarPara(q SolicitudRelacionUnidadExpedienteCT) error {
	if q.Validar() != nil || r.Solicitud != q ||
		!domain.ReferenciaOpacaValida(r.UnidadRef) || !domain.ReferenciaOpacaValida(r.UnidadSnapshotSolicitadoRef) ||
		r.UnidadRef != q.UnidadRefEsperada || r.UnidadSnapshotSolicitadoRef != r.UnidadRef ||
		!versionRelacionUnidadCTValida(r.VersionOrigenVinculo) || !versionRelacionUnidadCTValida(r.VersionExpedienteActual) ||
		r.VersionOrigenVinculo > q.VersionExpedienteSolicitada || q.VersionExpedienteSolicitada > r.VersionExpedienteActual ||
		!domain.ReferenciaOpacaValida(r.OperacionOrigenRef) || !domain.ReferenciaOpacaValida(r.ReservaAsignacionRef) ||
		r.OperacionOrigenRef != r.ReservaAsignacionRef || !domain.ReferenciaOpacaValida(r.ReciboAsignacionRef) ||
		!domain.HuellaSHA256FirmaValida(r.PruebaSnapshotOrigenHuellaSHA256) ||
		r.TipoEventoOrigen != TipoEventoOrigenRelacionUnidadCT || !domain.ReferenciaOpacaValida(r.EventoAsignacionRef) ||
		!domain.HuellaSHA256FirmaValida(r.EventoPayloadHuellaSHA256) || !domain.InstanteUTCCanonico(r.AsignacionConfirmadaEn) {
		return ErrRelacionUnidadExpedienteCTInvalida
	}
	return nil
}

// ValidarRevalidacion conserva la procedencia y unidad del vínculo mientras
// admite que el expediente haya avanzado. La decisión de replay y el CAS de
// un efecto nuevo pertenecen al consumidor SQL, no a este DTO.
func (r RelacionUnidadExpedienteCT) ValidarRevalidacion(esperada RelacionUnidadExpedienteCT) error {
	if esperada.ValidarPara(esperada.Solicitud) != nil || r.ValidarPara(esperada.Solicitud) != nil {
		return ErrRelacionUnidadExpedienteCTInvalida
	}
	if r.VersionExpedienteActual < esperada.VersionExpedienteActual ||
		r.VersionOrigenVinculo != esperada.VersionOrigenVinculo || r.OperacionOrigenRef != esperada.OperacionOrigenRef ||
		r.ReservaAsignacionRef != esperada.ReservaAsignacionRef || r.ReciboAsignacionRef != esperada.ReciboAsignacionRef ||
		r.PruebaSnapshotOrigenHuellaSHA256 != esperada.PruebaSnapshotOrigenHuellaSHA256 ||
		r.TipoEventoOrigen != esperada.TipoEventoOrigen || r.EventoAsignacionRef != esperada.EventoAsignacionRef ||
		r.EventoPayloadHuellaSHA256 != esperada.EventoPayloadHuellaSHA256 || !r.AsignacionConfirmadaEn.Equal(esperada.AsignacionConfirmadaEn) {
		return ErrRelacionUnidadExpedienteCTDivergente
	}
	return nil
}

func versionRelacionUnidadCTValida(v uint64) bool {
	return v > 0 && v <= maximaVersionRelacionUnidadCT
}
