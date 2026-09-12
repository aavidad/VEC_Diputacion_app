package httpinterno

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const RutaPreparacionCierreSinCese = "/api/vec/contratacion-temporal/seguimiento/cerrar-sin-cese/preparacion"

func NuevoManejadorPreparacionCierreSinCese(a AutoridadServidorCierreAdministrativo, e ports.LectorPreparacionCierreAdministrativo) (http.Handler, error) {
	if dependenciaNula(a) || dependenciaNula(e) {
		return nil, ports.ErrConsultaPreparacionCierreAdministrativoNoDisponible
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rutaPreparacionCierreSinCeseExacta(r) {
			responderErrorCierreAdministrativo(w, nuevoErrorCierreAdministrativo(http.StatusNotFound, "recurso_no_encontrado"))
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			responderErrorCierreAdministrativo(w, nuevoErrorCierreAdministrativo(http.StatusMethodNotAllowed, "metodo_no_permitido"))
			return
		}
		expediente, seguimiento, ok := consultaPreparacionCierreSinCese(r)
		if !ok {
			responderErrorCierreAdministrativo(w, nuevoErrorCierreAdministrativo(http.StatusBadRequest, "peticion_no_valida"))
			return
		}
		organizacion, err := a.ResolverOrganizacionCierreAdministrativo(r.Context())
		if err != nil {
			responderErrorPreparacionCierreSinCese(w, err)
			return
		}
		solicitud := ports.SolicitudPreparacionCierreAdministrativo{OrganizacionRef: organizacion, ExpedienteRef: expediente, SeguimientoRef: seguimiento}
		if solicitud.Validar() != nil {
			responderErrorCierreAdministrativo(w, errorServicioCierreAdministrativoNoDisponible)
			return
		}
		preparacion, err := e.ConsultarPreparacionCierreAdministrativo(r.Context(), solicitud)
		if err != nil {
			responderErrorPreparacionCierreSinCese(w, err)
			return
		}
		if preparacion.ValidarPara(solicitud) != nil {
			responderErrorCierreAdministrativo(w, errorServicioCierreAdministrativoNoDisponible)
			return
		}
		responderJSONCobertura(w, http.StatusOK, struct {
			Data ports.PreparacionCierreAdministrativo `json:"data"`
		}{preparacion})
	}), nil
}

func rutaPreparacionCierreSinCeseExacta(r *http.Request) bool {
	return r != nil && r.URL != nil && r.URL.Path == RutaPreparacionCierreSinCese && r.URL.RawPath == "" && r.URL.Scheme == "" && r.URL.Host == "" && r.URL.User == nil && r.URL.Opaque == "" && r.URL.Fragment == "" && r.URL.RawFragment == "" && !r.URL.ForceQuery && r.URL.EscapedPath() == r.URL.Path
}
func consultaPreparacionCierreSinCese(r *http.Request) (string, string, bool) {
	if r == nil || r.URL == nil || len(r.URL.RawQuery) > 600 || r.ContentLength != 0 || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 {
		return "", "", false
	}
	if r.Body != nil && r.Body != http.NoBody {
		b, err := io.ReadAll(io.LimitReader(r.Body, 1))
		if err != nil || len(b) != 0 {
			return "", "", false
		}
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(q) != 2 || len(q["expediente_ref"]) != 1 || len(q["seguimiento_ref"]) != 1 {
		return "", "", false
	}
	e, s := q.Get("expediente_ref"), q.Get("seguimiento_ref")
	return e, s, domain.ReferenciaOpacaValida(e) && domain.ReferenciaOpacaValida(s)
}
func responderErrorPreparacionCierreSinCese(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrContextoCanalAusente) || errors.Is(err, ErrContextoCanalCaducado) {
		responderErrorCierreAdministrativo(w, nuevoErrorCierreAdministrativo(http.StatusUnauthorized, "autenticacion_requerida"))
		return
	}
	if errors.Is(err, ErrContextoCanalOrganizacionDenegada) || errors.Is(err, ports.ErrAutorizacionDenegada) || errors.Is(err, ports.ErrCierreAdministrativoDenegado) {
		responderErrorCierreAdministrativo(w, errorAccesoCierreAdministrativoDenegado)
		return
	}
	responderErrorCierreAdministrativo(w, errorServicioCierreAdministrativoNoDisponible)
}

var _ context.Context
