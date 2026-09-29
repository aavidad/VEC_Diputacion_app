package httpinterno

import (
	"context"
	"net/http"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

const RutaPreparacionCoberturaVigente = "/api/vec/contratacion-temporal/cobertura/preparacion-vigente"

type ConsultorPreparacionCoberturaVigente interface {
	ConsultarParaAdaptador(
		context.Context,
		application.SolicitudConsultarPreparacionCoberturaVigente,
	) (application.ResultadoConsultaPreparacionCoberturaVigenteParaAdaptador, error)
}

type manejadorPreparacionCoberturaVigente struct {
	autoridad AutoridadContextoCanalCobertura
	consultor ConsultorPreparacionCoberturaVigente
}

func NuevoManejadorPreparacionCoberturaVigente(
	autoridad AutoridadContextoCanalCobertura,
	consultor ConsultorPreparacionCoberturaVigente,
) (http.Handler, error) {
	if dependenciaCoberturaNula(autoridad) || dependenciaCoberturaNula(consultor) {
		return nil, ErrManejadorCoberturaInvalido
	}
	return &manejadorPreparacionCoberturaVigente{
		autoridad: autoridad, consultor: consultor,
	}, nil
}

type envoltorioPreparacionCoberturaVigente struct {
	Data preparacionCoberturaVigenteJSON `json:"data"`
}

type preparacionCoberturaVigenteJSON struct {
	Esquema  string                           `json:"esquema"`
	Catalogo catalogoPreparacionCoberturaJSON `json:"catalogo"`
}

func (h *manejadorPreparacionCoberturaVigente) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || dependenciaCoberturaNula(h.autoridad) || dependenciaCoberturaNula(h.consultor) {
		responderErrorCobertura(w, r, errorServicioCoberturaNoDisponible)
		return
	}
	if !rutaPreparacionCoberturaVigenteExacta(r) {
		responderErrorCobertura(w, r, errorRecursoCoberturaNoEncontrado)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		responderErrorCobertura(w, r, errorMetodoCoberturaNoPermitido)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorCobertura(w, r, clasificarErrorCobertura(err), err)
		return
	}
	if r.ContentLength != 0 || (r.Body != nil && r.Body != http.NoBody) ||
		len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 ||
		r.Header.Get("Content-Type") != "" || cabeceraCoberturaProhibida(r.Header) {
		responderErrorCobertura(w, r, errorPeticionCoberturaNoPermitida)
		return
	}
	if !acceptCompatibleJSON(r.Header) {
		responderErrorCobertura(w, r, errorRepresentacionCoberturaNoAceptable)
		return
	}
	canal, err := h.autoridad.ResolverContextoCanalCobertura(r.Context())
	if err != nil {
		responderErrorCobertura(w, r, clasificarErrorCobertura(err), err)
		return
	}
	if !canal.valido() {
		responderErrorCobertura(w, r, errorServicioCoberturaNoDisponible)
		return
	}
	resultado, err := h.consultor.ConsultarParaAdaptador(
		r.Context(), application.SolicitudConsultarPreparacionCoberturaVigente{
			AutenticacionRef: canal.AutenticacionRef,
			SesionRef:        canal.SesionRef,
			PerfilRef:        canal.PerfilRef,
			OrganizacionRef:  canal.OrganizacionRef,
		},
	)
	if err != nil {
		responderErrorCobertura(w, r, clasificarErrorCobertura(err), err)
		return
	}
	datos, ok := resultado.DatosParaAdaptador()
	if !ok {
		responderErrorCobertura(w, r, errorServicioCoberturaNoDisponible)
		return
	}
	catalogo := proyectarPreparacionCatalogoCobertura(&datos)
	if catalogo == nil {
		responderErrorCobertura(w, r, errorServicioCoberturaNoDisponible)
		return
	}
	responderJSONCobertura(w, r, http.StatusOK, envoltorioPreparacionCoberturaVigente{
		Data: preparacionCoberturaVigenteJSON{
			Esquema:  "vec.contratacion-temporal.preparacion-cobertura.v1",
			Catalogo: *catalogo,
		},
	})
}

func rutaPreparacionCoberturaVigenteExacta(r *http.Request) bool {
	if r == nil || r.URL == nil || r.URL.Path != RutaPreparacionCoberturaVigente ||
		r.URL.RawQuery != "" || r.URL.ForceQuery || r.URL.RawPath != "" ||
		r.URL.Scheme != "" || r.URL.Host != "" || r.URL.User != nil || r.URL.Fragment != "" {
		return false
	}
	return r.URL.EscapedPath() == RutaPreparacionCoberturaVigente
}
