package httpinterno

import (
	"context"
	"net/http"
	"net/url"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const RutaConsultaEstadoSeleccionOferta = "/api/vec/contratacion-temporal/llamamientos/seleccion/estado"

type ConsultorEstadoSeleccionOferta interface {
	ConsultarEstadoSeleccionOfertaParaAdaptador(context.Context, string) (application.DatosEstadoSeleccionOfertaParaAdaptador, error)
}

type manejadorConsultaEstadoSeleccionOferta struct {
	consultor ConsultorEstadoSeleccionOferta
}

func NuevoManejadorConsultaEstadoSeleccionOferta(c ConsultorEstadoSeleccionOferta) (http.Handler, error) {
	if dependenciaSeleccionLlamamientoNula(c) {
		return nil, ErrManejadorSeleccionLlamamientoInvalido
	}
	return &manejadorConsultaEstadoSeleccionOferta{consultor: c}, nil
}

func (h *manejadorConsultaEstadoSeleccionOferta) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || dependenciaSeleccionLlamamientoNula(h.consultor) {
		responderErrorSeleccionLlamamiento(w, r, errorServicioSeleccionLlamamientoNoDisponible)
		return
	}
	if r == nil || r.URL == nil || r.URL.Path != RutaConsultaEstadoSeleccionOferta ||
		r.URL.RawPath != "" || r.URL.ForceQuery || r.URL.Scheme != "" ||
		r.URL.Host != "" || r.URL.User != nil || r.URL.Opaque != "" ||
		r.URL.Fragment != "" || r.URL.RawFragment != "" ||
		r.URL.EscapedPath() != r.URL.Path {
		responderErrorSeleccionLlamamiento(w, r, errorRecursoSeleccionLlamamientoNoEncontrado)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		responderErrorSeleccionLlamamiento(w, r, errorMetodoSeleccionLlamamientoNoPermitido)
		return
	}
	if r.Body != nil && r.ContentLength != 0 || r.Header.Get("Content-Type") != "" {
		responderErrorSeleccionLlamamiento(w, r, errorPeticionSeleccionLlamamientoNoValida)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorSeleccionLlamamiento(w, r, clasificarErrorSeleccionLlamamiento(err), err)
		return
	}
	valores := r.URL.Query()
	refs := valores["expediente_ref"]
	if len(valores) != 1 || len(refs) != 1 || !domain.ReferenciaOpacaValida(refs[0]) ||
		r.URL.RawQuery != "expediente_ref="+url.QueryEscape(refs[0]) {
		responderErrorSeleccionLlamamiento(w, r, errorPeticionSeleccionLlamamientoNoValida)
		return
	}
	resultado, err := h.consultor.ConsultarEstadoSeleccionOfertaParaAdaptador(r.Context(), refs[0])
	if err != nil {
		problema, esOferta := clasificarErrorSeleccionOferta(err)
		if !esOferta {
			problema = clasificarErrorSeleccionLlamamiento(err)
		}
		responderErrorSeleccionLlamamiento(w, r, problema, err)
		return
	}
	proyeccion, ok := proyectarEstadoSeleccionOferta(resultado, refs[0])
	if !ok {
		responderErrorSeleccionLlamamiento(w, r, errorResultadoSeleccionLlamamientoNoConfiable)
		return
	}
	responderJSONCobertura(w, r, http.StatusOK, struct {
		Data estadoSeleccionOfertaJSON `json:"data"`
	}{proyeccion})
}

type estadoSeleccionOfertaJSON struct {
	Esquema           string                     `json:"esquema"`
	Estado            string                     `json:"estado"`
	ExpedienteRef     string                     `json:"expediente_ref"`
	VersionEsperada   uint64                     `json:"version_esperada,omitempty"`
	ClaveIdempotencia string                     `json:"clave_idempotencia,omitempty"`
	VentanaAbierta    bool                       `json:"ventana_abierta,omitempty"`
	Oferta            *reciboSeleccionOfertaJSON `json:"oferta,omitempty"`
	Descarte          *reciboDescarteOfertaJSON  `json:"descarte,omitempty"`
}

func proyectarEstadoSeleccionOferta(d application.DatosEstadoSeleccionOfertaParaAdaptador, ref string) (estadoSeleccionOfertaJSON, bool) {
	if d.ExpedienteRef != ref || !domain.ReferenciaOpacaValida(ref) {
		return estadoSeleccionOfertaJSON{}, false
	}
	switch d.Estado {
	case "ausente":
		if d.VersionEsperada != 0 || d.ClaveIdempotencia != "" || d.VentanaAbierta ||
			d.Oferta != nil || d.Descarte != nil {
			return estadoSeleccionOfertaJSON{}, false
		}
	case "pendiente", "confirmada", "descartada":
		if d.VersionEsperada == 0 || d.VersionEsperada > 1<<53-1 ||
			!ports.ClaveIdempotenciaValida(d.ClaveIdempotencia) ||
			(d.Estado == "pendiente" && (d.Oferta != nil || d.Descarte != nil)) ||
			(d.Estado == "confirmada" && (d.Oferta == nil || d.Descarte != nil || d.VentanaAbierta)) ||
			(d.Estado == "descartada" && (d.Descarte == nil || d.Oferta != nil || d.VentanaAbierta)) {
			return estadoSeleccionOfertaJSON{}, false
		}
	default:
		return estadoSeleccionOfertaJSON{}, false
	}
	r := estadoSeleccionOfertaJSON{Esquema: "vec.contratacion-temporal.estado-seleccion-oferta.v1",
		Estado: d.Estado, ExpedienteRef: ref, VersionEsperada: d.VersionEsperada,
		ClaveIdempotencia: d.ClaveIdempotencia, VentanaAbierta: d.VentanaAbierta}
	if d.Oferta != nil {
		o, ok := proyectarReciboSeleccionOferta(application.DatosReciboSeleccionLlamamientoParaAdaptador{Oferta: d.Oferta}, ref)
		if !ok {
			return estadoSeleccionOfertaJSON{}, false
		}
		r.Oferta = &o
	}
	if d.Descarte != nil {
		x, ok := proyectarReciboDescarteOferta(application.DatosReciboSeleccionLlamamientoParaAdaptador{Descarte: d.Descarte}, ref)
		if !ok {
			return estadoSeleccionOfertaJSON{}, false
		}
		r.Descarte = &x
	}
	return r, true
}
