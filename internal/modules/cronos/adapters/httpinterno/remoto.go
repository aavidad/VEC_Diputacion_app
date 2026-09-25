package httpinterno

import (
	"encoding/json"
	"errors"
	"net/http"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

const RutaRegistrarMarcajeRemoto = "/api/interna/cronos/marcajes/remoto"
const RutaDisponibilidadMarcajeRemoto = "/api/interna/cronos/marcajes/remoto/disponibilidad"

// ResolverMarcajeRemoto obtiene identidad, canal y orden V3 de las autoridades
// ya compuestas. El cuerpo HTTP no aporta ninguno de esos datos.
type ResolverMarcajeRemoto interface {
	ResolverMarcajeRemoto(*http.Request) (ports.ContextoMarcajePropio, error)
}

type ManejadorMarcajeRemoto struct {
	resolver ResolverMarcajeRemoto
	casoUso  ports.CasoUsoMarcajesRemotos
}

func NuevoManejadorMarcajeRemoto(casoUso ports.CasoUsoMarcajesRemotos, resolver ResolverMarcajeRemoto) (*ManejadorMarcajeRemoto, error) {
	if dependenciaCronosHTTPNula(casoUso) || dependenciaCronosHTTPNula(resolver) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ManejadorMarcajeRemoto{resolver: resolver, casoUso: casoUso}, nil
}

func (m *ManejadorMarcajeRemoto) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.URL == nil || (r.URL.Path != RutaRegistrarMarcajeRemoto && r.URL.Path != RutaDisponibilidadMarcajeRemoto) || r.URL.RawPath != "" || r.URL.RawQuery != "" {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	if r.URL.Path == RutaDisponibilidadMarcajeRemoto {
		m.consultarDisponibilidad(w, r)
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
	contexto, err := m.resolver.ResolverMarcajeRemoto(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	if contexto.CanalAcreditado.Validar() != nil || contexto.CanalAcreditado.OrigenRef() != domain.OrigenMarcajeRemoto {
		responderErrorAccesoCronos(w, ports.ErrDependenciaNoDisponible)
		return
	}
	recibo, err := m.casoUso.RegistrarMarcajeRemoto(r.Context(), contexto, ports.SolicitudMarcajePropio{
		Movimiento: entrada.Movimiento, ClaveOperacion: entrada.ClaveOperacion,
	})
	if err != nil {
		switch {
		case errors.Is(err, ports.ErrClaveOperacionEnConflicto):
			errorJSON(w, http.StatusConflict, "conflicto")
		case errors.Is(err, ports.ErrTeletrabajoNoAutorizado):
			errorJSON(w, http.StatusForbidden, "teletrabajo_no_autorizado")
		case errors.Is(err, ports.ErrContinuidadMarcajeNoConfirmada):
			errorJSON(w, http.StatusServiceUnavailable, "continuidad_no_confirmada")
		case errors.Is(err, ports.ErrMovimientoRemotoNoPermitido):
			errorJSON(w, http.StatusConflict, "secuencia_no_permitida")
		default:
			responderErrorAccesoCronos(w, err)
		}
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"recibo": recibo})
}

func (m *ManejadorMarcajeRemoto) consultarDisponibilidad(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		errorJSON(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	contexto, err := m.resolver.ResolverMarcajeRemoto(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	if contexto.CanalAcreditado.Validar() != nil || contexto.CanalAcreditado.OrigenRef() != domain.OrigenMarcajeRemoto {
		responderErrorAccesoCronos(w, ports.ErrDependenciaNoDisponible)
		return
	}
	disponibilidad, err := m.casoUso.ConsultarDisponibilidadMarcajeRemoto(r.Context(), contexto)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	if !movimientosRemotosValidos(disponibilidad.MovimientosPermitidos) ||
		(disponibilidad.Autorizado && disponibilidad.Periodo == nil) ||
		(disponibilidad.Autorizado && disponibilidad.ContinuidadConfirmada && len(disponibilidad.MovimientosPermitidos) > 0 && disponibilidad.Motivo != "autorizado") ||
		(disponibilidad.Autorizado && disponibilidad.ContinuidadConfirmada && len(disponibilidad.MovimientosPermitidos) == 0 && disponibilidad.Motivo != "secuencia_no_permitida") ||
		(disponibilidad.Autorizado && !disponibilidad.ContinuidadConfirmada && (len(disponibilidad.MovimientosPermitidos) != 0 || disponibilidad.Motivo != "continuidad_no_confirmada")) ||
		(!disponibilidad.Autorizado && (disponibilidad.ContinuidadConfirmada || len(disponibilidad.MovimientosPermitidos) != 0 || disponibilidad.Periodo != nil || disponibilidad.Motivo != "teletrabajo_no_autorizado")) {
		errorJSON(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	if disponibilidad.MovimientosPermitidos == nil {
		disponibilidad.MovimientosPermitidos = []domain.PunchKind{}
	}
	_ = json.NewEncoder(w).Encode(disponibilidad)
}

func movimientosRemotosValidos(movimientos []domain.PunchKind) bool {
	if len(movimientos) > 4 {
		return false
	}
	vistos := make(map[domain.PunchKind]bool, len(movimientos))
	for _, movimiento := range movimientos {
		switch movimiento {
		case domain.PunchEntry, domain.PunchExit, domain.PunchPauseStart, domain.PunchPauseEnd:
		default:
			return false
		}
		if vistos[movimiento] {
			return false
		}
		vistos[movimiento] = true
	}
	return true
}
