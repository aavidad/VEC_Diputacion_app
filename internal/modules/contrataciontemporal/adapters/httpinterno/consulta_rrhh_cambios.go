package httpinterno

import (
	"context"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Petición RRHH p.4: la misma ruta, método y autorización que el detalle
// devuelven, con este Accept, el histórico de cambios del expediente con
// valor anterior y nuevo. La representación no amplía la autoridad.
const (
	AcceptCambiosExpedienteRRHH  = "application/vnd.vec.contratacion-temporal.cambios-expediente+json"
	EsquemaCambiosExpedienteRRHH = "vec.contratacion_temporal.rrhh.cambios_expediente.v1"
)

// ConsultorCambiosRRHH lo satisface el caso de uso de detalle cuando su
// sesión ofrece la lectura de cambios.
type ConsultorCambiosRRHH interface {
	ConsultarCambios(context.Context, ports.SolicitudDetalleRRHH) (ports.ResultadoConsultaCambiosRRHH, error)
}

type cambioExpedienteRRHHJSON struct {
	VersionExpediente uint64  `json:"version_expediente"`
	RegistradaEn      string  `json:"registrada_en"`
	OrigenVersion     string  `json:"origen_version"`
	Ruta              string  `json:"ruta"`
	ValorAnterior     *string `json:"valor_anterior"`
	ValorNuevo        *string `json:"valor_nuevo"`
}

type envoltorioCambiosExpedienteRRHH struct {
	Data struct {
		Esquema           string                     `json:"esquema"`
		ExpedienteRef     string                     `json:"expediente_ref"`
		VersionExpediente uint64                     `json:"version_expediente"`
		Cambios           []cambioExpedienteRRHHJSON `json:"cambios"`
	} `json:"data"`
}

func cambiosExpedienteRRHHSolicitados(cabeceras http.Header) bool {
	valor, unico := cabeceraUnicaAlta(cabeceras, "Accept")
	return unico && valor == AcceptCambiosExpedienteRRHH
}

func (h *manejadorConsultaDetalleRRHH) responderCambios(w http.ResponseWriter, r *http.Request) {
	consultor, disponible := h.consultor.(ConsultorCambiosRRHH)
	if !disponible || dependenciaConsultaRRHHNula(consultor) {
		responderErrorConsultaRRHH(w, r, nil, errorServicioConsultaRRHHNoDisponible)
		return
	}
	if r.ContentLength > MaximoCuerpoConsultaDetalleRRHHBytes {
		responderErrorConsultaRRHH(w, r, nil, errorCuerpoConsultaRRHHDemasiadoGrande)
		return
	}
	if r.ContentLength == 0 || r.Body == nil || r.Body == http.NoBody || len(r.Trailer) != 0 ||
		!transferenciaAltaPermitida(r.TransferEncoding) {
		responderErrorConsultaRRHH(w, r, nil, errorPeticionConsultaRRHHNoValida)
		return
	}
	if !cabeceraJSONConsultaRRHHExacta(r.Header, "Content-Type") {
		responderErrorConsultaRRHH(w, r, nil, errorTipoConsultaRRHHNoAdmitido)
		return
	}
	if cabeceraCoberturaProhibida(r.Header) {
		responderErrorConsultaRRHH(w, r, nil, errorPeticionConsultaRRHHNoPermitida)
		return
	}
	solicitud, err := solicitudDetalleRRHHDesdePeticion(w, r)
	if err != nil {
		responderErrorConsultaRRHH(w, r, nil, errorEntradaConsultaRRHH(err))
		return
	}
	resultado, err := consultor.ConsultarCambios(r.Context(), solicitud)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorConsultaRRHH(w, r, errContexto, clasificarErrorConsultaRRHH(errContexto))
		return
	}
	if err != nil {
		responderErrorConsultaRRHH(w, r, err, clasificarErrorConsultaRRHH(err))
		return
	}
	var salida envoltorioCambiosExpedienteRRHH
	salida.Data.Esquema = EsquemaCambiosExpedienteRRHH
	salida.Data.ExpedienteRef = resultado.ExpedienteRef
	salida.Data.VersionExpediente = resultado.VersionExpediente
	salida.Data.Cambios = make([]cambioExpedienteRRHHJSON, 0, len(resultado.Cambios))
	for _, c := range resultado.Cambios {
		salida.Data.Cambios = append(salida.Data.Cambios, cambioExpedienteRRHHJSON{
			VersionExpediente: c.VersionExpediente, RegistradaEn: c.RegistradaEn.UTC().Format(time.RFC3339Nano),
			OrigenVersion: c.OrigenVersion, Ruta: c.Ruta, ValorAnterior: c.ValorAnterior, ValorNuevo: c.ValorNuevo,
		})
	}
	responderJSONConsultaRRHH(w, r, http.StatusOK, salida)
}
