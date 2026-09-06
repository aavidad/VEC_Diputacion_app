package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"vec-diputacion-granada/config"
	personalcatalogos "vec-diputacion-granada/internal/modules/personal/adapters/catalogosvec"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

const rutaOrganizacionContratacionTemporalDesarrollo = "/api/vec/contratacion-temporal/organizacion"

// La consulta reutiliza Personal y el catálogo versionado. No cambia el
// catálogo de altas ni concede permisos por adscripción o denominación.
func nuevaRutaOrganizacionContratacionTemporalDesarrollo(cfg config.Config) (vechttp.RutaExacta, error) {
	manejador := &manejadorOrganizacionContratacionTemporalDesarrollo{}
	if cfg.PersonalOrganizacionSourcePath != "" {
		fuente, err := fichero.NuevaConsultaCatalogos(cfg.PersonalOrganizacionSourcePath)
		if err != nil {
			return vechttp.RutaExacta{}, err
		}
		consulta, err := personalcatalogos.NuevaConsultaEstructuraOrganizativa(
			fuente, "estructura-organizativa-dipgra", cfg.PersonalOrganizacionVersion,
		)
		if err != nil {
			return vechttp.RutaExacta{}, err
		}
		if _, err := consulta.Obtener(context.Background()); err != nil {
			return vechttp.RutaExacta{}, err
		}
		manejador.consulta = consulta
	}
	return vechttp.RutaExacta{
		Ruta: rutaOrganizacionContratacionTemporalDesarrollo, Manejador: manejador,
	}, nil
}

type manejadorOrganizacionContratacionTemporalDesarrollo struct {
	consulta          personalports.ConsultaEstructuraOrganizativa
	edicionHabilitada bool
}

func (m *manejadorOrganizacionContratacionTemporalDesarrollo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	prepararCabecerasCatalogosAltaContratacionTemporalDesarrollo(w)
	if r == nil || r.URL == nil || r.URL.Path != rutaOrganizacionContratacionTemporalDesarrollo ||
		r.URL.RawQuery != "" || r.ContentLength != 0 || len(r.TransferEncoding) != 0 ||
		cabeceraCatalogosAltaContratacionTemporalDesarrolloProhibida(r.Header) {
		responderErrorCatalogosAltaContratacionTemporalDesarrollo(w, r, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		responderErrorCatalogosAltaContratacionTemporalDesarrollo(w, r, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if m == nil || m.consulta == nil {
		responderErrorCatalogosAltaContratacionTemporalDesarrollo(w, r, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	datos, err := m.consulta.Obtener(r.Context())
	if err != nil {
		responderErrorCatalogosAltaContratacionTemporalDesarrollo(w, r, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	contenido, err := json.Marshal(struct {
		Data struct {
			personalports.EstructuraOrganizativaConsultable
			EdicionHabilitada bool `json:"edicion_habilitada"`
		} `json:"data"`
	}{Data: struct {
		personalports.EstructuraOrganizativaConsultable
		EdicionHabilitada bool `json:"edicion_habilitada"`
	}{datos, m.edicionHabilitada}})
	if err != nil || len(contenido) > 512*1024 {
		responderErrorCatalogosAltaContratacionTemporalDesarrollo(w, r, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		_, _ = w.Write(contenido)
	}
}
