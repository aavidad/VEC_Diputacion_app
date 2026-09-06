package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type manejadorEntregaPeticionDesarrollo struct {
	proveedor   *proveedorEntregaPeticionDesarrollo
	repositorio ports.RepositorioEntregasPeticionCentro
	servicio    *application.ServicioEntregaPeticionCentro
}

func (m *manejadorEntregaPeticionDesarrollo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	prepararCabecerasCatalogosAltaContratacionTemporalDesarrollo(w)
	fallo := func(estado int, codigo string) {
		w.WriteHeader(estado)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"codigo": codigo, "clave_i18n": "api.contratacion_temporal.entrega_peticion.error." + codigo}})
	}
	if m == nil || m.proveedor == nil || m.repositorio == nil || m.servicio == nil || r == nil || r.URL == nil {
		fallo(503, "servicio_no_disponible")
		return
	}
	if r.URL.Path != rutaEntregaPeticionCentro || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 || cabeceraCatalogosAltaContratacionTemporalDesarrolloProhibida(r.Header) {
		fallo(400, "solicitud_invalida")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		fallo(405, "metodo_no_permitido")
		return
	}
	if _, _, err := m.proveedor.ActorEntregaPeticionCentro(r.Context()); err != nil {
		fallo(403, "operacion_denegada")
		return
	}
	responder := func(data any) { _ = json.NewEncoder(w).Encode(map[string]any{"data": data}) }
	if r.Method == http.MethodGet {
		if r.ContentLength != 0 {
			fallo(400, "solicitud_invalida")
			return
		}
		filas, err := m.repositorio.ListarPeticionesRRHH(r.Context())
		if err != nil {
			fallo(503, "servicio_no_disponible")
			return
		}
		responder(map[string]any{"peticiones": filas, "limite": 50})
		return
	}
	tipo, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || tipo != "application/json" || len(params) > 0 && (len(params) != 1 || !strings.EqualFold(params["charset"], "utf-8")) || r.ContentLength > 1024 || r.Body == nil {
		fallo(400, "solicitud_invalida")
		return
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1024))
	if err != nil || validarClavesJSONUnicas(b) != nil {
		fallo(400, "solicitud_invalida")
		return
	}
	defer clear(b)
	var c ports.ComandoEntregarPeticionCentro
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil || d.Decode(&struct{}{}) != io.EOF || c.Validar() != nil {
		fallo(400, "solicitud_invalida")
		return
	}
	e, err := m.servicio.Entregar(r.Context(), c)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPeticionCentroInvalida):
			fallo(400, "solicitud_invalida")
		case errors.Is(err, ports.ErrAutorizacionDenegada):
			fallo(403, "operacion_denegada")
		case errors.Is(err, ports.ErrEntregaPeticionEnConflicto):
			fallo(409, "peticion_en_conflicto")
		default:
			fallo(503, "servicio_no_disponible")
		}
		return
	}
	// La clave de alta y el dueño de la reserva no salen al navegador. El
	// recibo sí conserva la trazabilidad original y permite retomar el análisis.
	e.ClaveAlta, e.AmbitoAltaHMAC, e.ActorRef, e.PerfilRef = "", "", "", ""
	responder(e)
}
