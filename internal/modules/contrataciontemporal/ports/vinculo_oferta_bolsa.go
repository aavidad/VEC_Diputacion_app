package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	// La concesión, el recurso y la audiencia son propios de la oferta.
	// CT201 singular conserva íntegro su contrato anterior.
	AccionVincularOfertaBolsaCT     = "contratacion_temporal.bolsa.oferta.vincular"
	TipoRecursoVinculoOfertaBolsaCT = "vinculo_oferta_bolsa_expediente"
	AudienciaVincularOfertaBolsaCT  = "vec_contratacion_temporal.vinculo_oferta_bolsa.v1"
	FinalidadVincularOfertaBolsaCT  = "tramitacion_expediente_contratacion_temporal"
)

var ErrVinculoOfertaBolsaCTInvalido = errors.New("contratacion temporal: vinculo de oferta de bolsa invalido")

// OfertaPublicadaSeleccionCT minimiza el recibo autenticado de Bolsa. La
// composición verifica su correspondencia con el comando congelado y Bolsa
// vuelve a acreditar el origen ante CT mediante su función nominal B120.
type OfertaPublicadaSeleccionCT struct {
	ClaveBolsa      string                 `json:"clave_bolsa"`
	BolsaRef        string                 `json:"bolsa_ref"`
	OfertaRef       string                 `json:"oferta_ref"`
	ReciboOfertaRef string                 `json:"recibo_oferta_ref"`
	NumeroPlazas    int                    `json:"numero_plazas"`
	Datos           DatosOfertaSeleccionCT `json:"datos"`
	PublicadaEn     time.Time              `json:"publicada_en"`
	VenceAntesDe    time.Time              `json:"vence_antes_de"`
}

func (o OfertaPublicadaSeleccionCT) ValidarPara(c ComandoOfertaBolsaCT) error {
	if o.ClaveBolsa != c.ClaveBolsa || o.BolsaRef != c.BolsaRef ||
		!domain.ReferenciaOpacaValida(o.OfertaRef) ||
		!domain.ReferenciaOpacaValida(o.ReciboOfertaRef) ||
		o.NumeroPlazas != c.NumeroPlazas || o.Datos != c.Datos ||
		!domain.InstanteUTCCanonico(o.PublicadaEn) ||
		!domain.InstanteUTCCanonico(o.VenceAntesDe) ||
		!o.VenceAntesDe.After(o.PublicadaEn) {
		return ErrVinculoOfertaBolsaCTInvalido
	}
	return nil
}

// El publicador recibe siempre el comando original y una identidad
// revalidada; emite material V3 nuevo de Bolsa en cada llamada.
type PublicadorOfertaBolsaCT interface {
	PublicarORecuperar(context.Context, IntencionSeleccionOfertaCT, ComandoOfertaBolsaCT) (OfertaPublicadaSeleccionCT, error)
}

// ReciboSeleccionOfertaCT prueba el vínculo propio de CT con la oferta real.
// El recibo permanece estable al consultar una operación terminal histórica.
type ReciboSeleccionOfertaCT struct {
	Esquema           string    `json:"esquema"`
	Estado            string    `json:"estado"`
	OrganizacionRef   string    `json:"organizacion_ref"`
	ExpedienteRef     string    `json:"expediente_ref"`
	VersionExpediente uint64    `json:"version_expediente"`
	BolsaRef          string    `json:"bolsa_ref"`
	OfertaRef         string    `json:"oferta_ref"`
	ReciboOfertaRef   string    `json:"recibo_oferta_ref"`
	ReciboVinculoRef  string    `json:"recibo_vinculo_ref"`
	NumeroPlazas      int       `json:"numero_plazas"`
	PublicadaEn       time.Time `json:"publicada_en"`
	VenceAntesDe      time.Time `json:"vence_antes_de"`
}

func (r ReciboSeleccionOfertaCT) ValidarPara(i IntencionSeleccionOfertaCT, c ComandoOfertaBolsaCT) error {
	if i.Validar() != nil || c.ValidarPara(i) != nil ||
		r.Esquema != EsquemaReciboSeleccionOfertaCTV1 ||
		r.Estado != EstadoReciboSeleccionOfertaCTV1 ||
		r.OrganizacionRef != i.OrganizacionRef || r.ExpedienteRef != i.ExpedienteRef ||
		r.VersionExpediente != i.VersionExpediente || r.BolsaRef != c.BolsaRef ||
		!domain.ReferenciaOpacaValida(r.OfertaRef) ||
		!domain.ReferenciaOpacaValida(r.ReciboOfertaRef) ||
		!domain.ReferenciaOpacaValida(r.ReciboVinculoRef) ||
		r.NumeroPlazas != c.NumeroPlazas ||
		!domain.InstanteUTCCanonico(r.PublicadaEn) ||
		!domain.InstanteUTCCanonico(r.VenceAntesDe) ||
		!r.VenceAntesDe.After(r.PublicadaEn) {
		return ErrVinculoOfertaBolsaCTInvalido
	}
	return nil
}
