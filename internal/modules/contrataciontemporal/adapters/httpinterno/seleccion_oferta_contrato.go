package httpinterno

import (
	"errors"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var referenciaOfertaSeleccionCT = regexp.MustCompile(`^oferta:[0-9a-f]{64}$`)
var reciboOfertaSeleccionCT = regexp.MustCompile(`^recibo:oferta:[0-9a-f]{64}$`)

type envoltorioReciboSeleccionOferta struct {
	Data reciboSeleccionOfertaJSON `json:"data"`
}

// La salida ofrece únicamente el vínculo CT y el recibo de publicación de
// Bolsa. El comando reservado, candidatos, capturas y material V3 no cruzan HTTP.
type reciboSeleccionOfertaJSON struct {
	Esquema          string `json:"esquema"`
	Estado           string `json:"estado"`
	OrganizacionRef  string `json:"organizacion_ref"`
	ExpedienteRef    string `json:"expediente_ref"`
	BolsaRef         string `json:"bolsa_ref"`
	OfertaRef        string `json:"oferta_ref"`
	ReciboOfertaRef  string `json:"recibo_oferta_ref"`
	ReciboVinculoRef string `json:"recibo_vinculo_ref"`
	NumeroPlazas     int    `json:"numero_plazas"`
	PublicadaEn      string `json:"publicada_en"`
	VenceAntesDe     string `json:"vence_antes_de"`
}

func proyectarReciboSeleccionOferta(
	recibo application.DatosReciboSeleccionLlamamientoParaAdaptador,
	expedienteRef string,
) (reciboSeleccionOfertaJSON, bool) {
	o := recibo.Oferta
	if o == nil || recibo.ReciboRef != "" || !recibo.ConfirmadaEn.IsZero() ||
		recibo.OrganizacionRef != "" || recibo.LlamamientoRef != "" ||
		recibo.VersionLlamamiento != 0 || recibo.PoliticaProvisional != nil ||
		o.Esquema != ports.EsquemaReciboSeleccionOfertaCTV1 ||
		o.Estado != ports.EstadoReciboSeleccionOfertaCTV1 ||
		!domain.ReferenciaOpacaValida(o.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(o.ExpedienteRef) || o.ExpedienteRef != expedienteRef ||
		!domain.ReferenciaOpacaValida(o.BolsaRef) ||
		!referenciaOfertaSeleccionCT.MatchString(o.OfertaRef) ||
		!reciboOfertaSeleccionCT.MatchString(o.ReciboOfertaRef) ||
		!domain.ReferenciaOpacaValida(o.ReciboVinculoRef) ||
		o.NumeroPlazas < 2 || o.NumeroPlazas > 100 ||
		!domain.InstanteUTCCanonico(o.PublicadaEn) ||
		!domain.InstanteUTCCanonico(o.VenceAntesDe) ||
		!o.VenceAntesDe.After(o.PublicadaEn) {
		return reciboSeleccionOfertaJSON{}, false
	}
	return reciboSeleccionOfertaJSON{
		Esquema: o.Esquema, Estado: o.Estado,
		OrganizacionRef: o.OrganizacionRef, ExpedienteRef: o.ExpedienteRef,
		BolsaRef: o.BolsaRef, OfertaRef: o.OfertaRef,
		ReciboOfertaRef: o.ReciboOfertaRef, ReciboVinculoRef: o.ReciboVinculoRef,
		NumeroPlazas: o.NumeroPlazas,
		PublicadaEn:  o.PublicadaEn.Format(time.RFC3339Nano),
		VenceAntesDe: o.VenceAntesDe.Format(time.RFC3339Nano),
	}, true
}

// Los fallos propios de oferta conservan la semántica pública de esta ruta.
// Los fallos compartidos (denegación, cancelación y plazo) siguen pasando por
// el clasificador singular, sin crear una segunda autoridad HTTP.
func clasificarErrorSeleccionOferta(err error) (errorPublicoCobertura, bool) {
	switch {
	case errors.Is(err, application.ErrClaveSeleccionOfertaEnColision):
		return errorConflictoSeleccionLlamamientoNoReintentable, true
	case errors.Is(err, application.ErrSeleccionOfertaConcurrente):
		return errorConflictoEstadoSeleccionLlamamiento, true
	case errors.Is(err, application.ErrSolicitudSeleccionOfertaInvalida):
		return errorContenidoSeleccionLlamamientoInvalido, true
	case errors.Is(err, application.ErrResultadoSeleccionOfertaNoConfiable):
		return errorResultadoSeleccionLlamamientoNoConfiable, true
	case errors.Is(err, application.ErrSeleccionOfertaIndeterminada),
		errors.Is(err, application.ErrSeleccionOfertaNoDisponible),
		errors.Is(err, application.ErrServicioSeleccionOfertaInvalido):
		return errorServicioSeleccionLlamamientoNoDisponible, true
	default:
		return errorPublicoCobertura{}, false
	}
}
