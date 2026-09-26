package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// RutaNoIncorporaciones registra la no incorporación de la persona aceptada
// (CT124): el expediente vuelve a fiscalización, Bolsa aplica la baja y el
// llamamiento continúa con el siguiente candidato.
const RutaNoIncorporaciones = "/api/vec/contratacion-temporal/no-incorporaciones"

// EjecutorNoIncorporacion es opcional: solo con la no incorporación
// compuesta existe su ruta.
type EjecutorNoIncorporacion interface {
	RegistrarNoIncorporacion(context.Context, application.SolicitudRegistrarNoIncorporacion) (ports.ReciboOperacionSeguimiento, error)
}

func (h *manejadorSeguimiento) noIncorporacion(ctx context.Context, canal application.ContextoCanalSeguimiento, contenido []byte) (ports.ReciboOperacionSeguimiento, error) {
	ejecutor, ok := h.ejecutor.(EjecutorNoIncorporacion)
	if !ok {
		return ports.ReciboOperacionSeguimiento{}, ports.ErrOperacionSeguimientoNoDisponible
	}
	var in struct {
		ExpedienteRef     string `json:"expediente_ref"`
		VersionEsperada   uint64 `json:"version_esperada"`
		ClaveIdempotencia string `json:"clave_idempotencia"`
		MotivoClave       string `json:"motivo_clave"`
		ResolucionRef     string `json:"resolucion_ref"`
		ResolucionSHA256  string `json:"resolucion_sha256"`
		ResueltaPor       string `json:"resuelta_por"`
		FechaNotificacion string `json:"fecha_notificacion"`
		Observaciones     string `json:"observaciones"`
	}
	if decodificarCuerpoSeguimiento(contenido, &in) != nil {
		return ports.ReciboOperacionSeguimiento{}, errContenidoSeguimiento
	}
	fecha, err := fechaCivilSeguimiento(in.FechaNotificacion)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	return ejecutor.RegistrarNoIncorporacion(ctx, application.SolicitudRegistrarNoIncorporacion{Canal: canal, ExpedienteRef: in.ExpedienteRef,
		VersionEsperada: in.VersionEsperada, ClaveIdempotencia: in.ClaveIdempotencia, MotivoClave: in.MotivoClave,
		ResolucionRef: in.ResolucionRef, ResolucionSHA256: in.ResolucionSHA256, ResueltaPor: in.ResueltaPor,
		FechaNotificacion: fecha, Observaciones: in.Observaciones})
}

// estadoErrorNoIncorporacion traduce los rechazos propios a 409 con código.
func estadoErrorNoIncorporacion(err error) (int, string, bool) {
	switch {
	case errors.Is(err, ports.ErrSinAceptacion):
		return http.StatusConflict, "sin_aceptacion", true
	case errors.Is(err, ports.ErrIncorporacionExistente):
		return http.StatusConflict, "incorporacion_existente", true
	case errors.Is(err, ports.ErrNoIncorporacionExistente):
		return http.StatusConflict, "no_incorporacion_existente", true
	case errors.Is(err, ports.ErrFechaNoIncorporacionNoAdmitida):
		return http.StatusConflict, "fecha_no_admitida", true
	}
	return 0, "", false
}

func opcionesNoIncorporacionJSON(r *ports.ReglaNoIncorporacion) map[string]any {
	motivos := make([]map[string]string, 0, len(r.Motivos))
	for _, m := range r.Motivos {
		motivos = append(motivos, map[string]string{"clave": m.Clave, "etiqueta": m.Etiqueta})
	}
	return map[string]any{"motivos": motivos, "segunda_persona": r.SegundaPersona}
}

func estadoNoIncorporacionJSON(n *ports.EstadoNoIncorporacion) map[string]string {
	return map[string]string{"motivo_clave": n.MotivoClave, "resolucion_ref": n.ResolucionRef, "resolucion_sha256": n.ResolucionSHA256,
		"resuelta_por": n.ResueltaPor, "fecha_notificacion": n.FechaNotificacion, "recibo_ref": n.ReciboRef,
		"registrada_en": n.RegistradaEn.UTC().Format(time.RFC3339Nano)}
}
