// Package httpinterno adapta solo la ruta del enclave Cronos; la ruta se monta fuera del artefacto publico.
package httpinterno

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

const RutaRegistrarMarcajePropio = "/api/interna/cronos/marcajes/propio"

type ResolverContextoMarcajePropio interface {
	ResolverMarcajePropio(*http.Request) (ports.ContextoMarcajePropio, error)
}
type ManejadorMarcajes struct {
	casoUso  ports.CasoUsoRegistrarMarcajePropio
	resolver ResolverContextoMarcajePropio
}

func NuevoManejadorMarcajes(casoUso ports.CasoUsoRegistrarMarcajePropio, resolver ResolverContextoMarcajePropio) (*ManejadorMarcajes, error) {
	if casoUso == nil || resolver == nil {
		return nil, errors.New("cronos handler requiere caso de uso y resolver")
	}
	return &ManejadorMarcajes{casoUso: casoUso, resolver: resolver}, nil
}

type solicitud struct {
	Movimiento     domain.PunchKind `json:"movimiento"`
	ClaveOperacion string           `json:"clave_operacion"`
}

func (m *ManejadorMarcajes) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.URL == nil || r.URL.Path != RutaRegistrarMarcajePropio || r.URL.RawPath != "" || r.URL.RawQuery != "" {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		errorJSON(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	defer r.Body.Close()
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	var entrada solicitud
	if decodificarSolicitud(dec, &entrada) != nil {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	contexto, err := m.resolver.ResolverMarcajePropio(r)
	if err != nil {
		errorJSON(w, http.StatusForbidden, "no_autorizado")
		return
	}
	recibo, err := m.casoUso.RegistrarMarcajePropio(r.Context(), contexto, ports.SolicitudMarcajePropio{Movimiento: entrada.Movimiento, ClaveOperacion: entrada.ClaveOperacion})
	if err != nil {
		if errors.Is(err, ports.ErrClaveOperacionEnConflicto) {
			errorJSON(w, http.StatusConflict, "conflicto")
			return
		}
		errorJSON(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"recibo": recibo})
}
func errorJSON(w http.ResponseWriter, status int, mensaje string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": mensaje})
}

func decodificarSolicitud(dec *json.Decoder, entrada *solicitud) error {
	apertura, err := dec.Token()
	if err != nil || apertura != json.Delim('{') {
		return errors.New("peticion_invalida")
	}
	vistos := make(map[string]bool, 2)
	for dec.More() {
		token, err := dec.Token()
		nombre, ok := token.(string)
		if err != nil || !ok || vistos[nombre] {
			return errors.New("peticion_invalida")
		}
		vistos[nombre] = true
		switch nombre {
		case "movimiento":
			err = dec.Decode(&entrada.Movimiento)
		case "clave_operacion":
			err = dec.Decode(&entrada.ClaveOperacion)
		default:
			return errors.New("peticion_invalida")
		}
		if err != nil {
			return errors.New("peticion_invalida")
		}
	}
	cierre, err := dec.Token()
	if err != nil || cierre != json.Delim('}') || len(vistos) != 2 ||
		dec.Decode(&struct{}{}) != io.EOF || entrada.ClaveOperacion == "" {
		return errors.New("peticion_invalida")
	}
	switch entrada.Movimiento {
	case domain.PunchEntry, domain.PunchExit, domain.PunchPauseStart, domain.PunchPauseEnd:
		return nil
	default:
		return errors.New("peticion_invalida")
	}
}
