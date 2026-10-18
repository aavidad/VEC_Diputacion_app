package httpinterno

import (
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type reciboDescarteOfertaJSON struct {
	Esquema              string `json:"esquema"`
	Estado               string `json:"estado"`
	OrganizacionRef      string `json:"organizacion_ref"`
	ExpedienteRef        string `json:"expediente_ref"`
	ReciboCierreBolsaRef string `json:"recibo_cierre_bolsa_ref"`
	ReciboDescarteRef    string `json:"recibo_descarte_ref"`
	DescartadaEn         string `json:"descartada_en"`
}

type envoltorioReciboDescarteOferta struct {
	Data reciboDescarteOfertaJSON `json:"data"`
}

func proyectarReciboDescarteOferta(r application.DatosReciboSeleccionLlamamientoParaAdaptador,
	expedienteRef string,
) (reciboDescarteOfertaJSON, bool) {
	d := r.Descarte
	if d == nil || r.Oferta != nil || r.ReciboRef != "" || !r.ConfirmadaEn.IsZero() ||
		r.OrganizacionRef != "" || r.LlamamientoRef != "" || r.VersionLlamamiento != 0 || r.PoliticaProvisional != nil ||
		d.Esquema != "vec.contratacion-temporal.recibo-descarte-oferta.v1" || d.Estado != "descartada" ||
		!domain.ReferenciaOpacaValida(d.OrganizacionRef) || !domain.ReferenciaOpacaValida(d.ExpedienteRef) ||
		d.ExpedienteRef != expedienteRef || !domain.ReferenciaOpacaValida(d.ReciboCierreBolsaRef) ||
		!domain.ReferenciaOpacaValida(d.ReciboDescarteRef) ||
		!strings.HasPrefix(d.ReciboCierreBolsaRef, "recibo:") || !strings.HasPrefix(d.ReciboDescarteRef, "recibo:") ||
		!domain.InstanteUTCCanonico(d.DescartadaEn) || d.DescartadaEn.Nanosecond()%1000 != 0 {
		return reciboDescarteOfertaJSON{}, false
	}
	return reciboDescarteOfertaJSON{Esquema: d.Esquema, Estado: d.Estado,
		OrganizacionRef: d.OrganizacionRef, ExpedienteRef: d.ExpedienteRef,
		ReciboCierreBolsaRef: d.ReciboCierreBolsaRef, ReciboDescarteRef: d.ReciboDescarteRef,
		DescartadaEn: d.DescartadaEn.UTC().Format(time.RFC3339Nano)}, true
}
