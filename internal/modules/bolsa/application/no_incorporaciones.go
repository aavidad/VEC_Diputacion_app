package application

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// ServicioRecepcionNoIncorporaciones es el consumidor de las no
// incorporaciones de Contratación temporal (CT124): valida el evento, le
// añade la consecuencia que fija el catálogo de Bolsa y lo entrega a la
// bandeja, que aplica la baja una sola vez y habilita el siguiente llamamiento.
type ServicioRecepcionNoIncorporaciones struct {
	buzon    ports.BuzonNoIncorporaciones
	catalogo ports.ResolvedorSancionNoIncorporacion
}

func NuevoServicioRecepcionNoIncorporaciones(buzon ports.BuzonNoIncorporaciones, catalogo ports.ResolvedorSancionNoIncorporacion) (*ServicioRecepcionNoIncorporaciones, error) {
	if buzon == nil || catalogo == nil {
		return nil, ports.ErrContratosParticipacionNoDisponible
	}
	return &ServicioRecepcionNoIncorporaciones{buzon: buzon, catalogo: catalogo}, nil
}

// Cursor devuelve el último origen recibido; false si la bandeja está vacía.
func (s *ServicioRecepcionNoIncorporaciones) Cursor(ctx context.Context) (ports.CursorContratosParticipacion, bool, error) {
	if s == nil || ctx == nil {
		return ports.CursorContratosParticipacion{}, false, ports.ErrContratosParticipacionNoDisponible
	}
	return s.buzon.CursorNoIncorporaciones(ctx)
}

// Recibir registra una sola vez el evento. Si el catálogo no reconoce la
// consecuencia, el evento se conserva sin efecto; si el catálogo no está
// disponible, no se registra nada y el relevo reintentará.
func (s *ServicioRecepcionNoIncorporaciones) Recibir(ctx context.Context, contenido []byte, huellaSHA256 string, origenCreadaEn time.Time, origenPosicion int64) (ports.ResultadoRegistroNoIncorporacion, error) {
	if s == nil || ctx == nil || origenCreadaEn.IsZero() || origenPosicion < 0 {
		return ports.ResultadoRegistroNoIncorporacion{}, ports.ErrContratosParticipacionNoDisponible
	}
	evento, err := dominiobolsa.DecodificarEventoNoIncorporacion(contenido, huellaSHA256)
	if err != nil {
		return ports.ResultadoRegistroNoIncorporacion{}, err
	}
	notificada, err := evento.Notificacion()
	if err != nil {
		return ports.ResultadoRegistroNoIncorporacion{}, err
	}
	var consecuencia *ports.ConsecuenciaNoIncorporacion
	r, err := s.catalogo.ResolverSancion(ctx, evento.ConsecuenciaClave, notificada)
	switch {
	case errors.Is(err, dominiobolsa.ErrSancionParticipacionInvalida):
		// Clave desconocida: se conserva sin efecto para revisión.
	case err != nil:
		return ports.ResultadoRegistroNoIncorporacion{}, err
	default:
		c := r.Consecuencia
		consecuencia = &ports.ConsecuenciaNoIncorporacion{Clave: c.Clave, Etiqueta: c.Etiqueta, Efecto: c.Efecto, ReglaRef: c.ReglaRef,
			ReglaHuellaSHA256: c.Huella, RecursoVence: r.Recurso.UltimoDia, RecursoReglaRef: r.Recurso.ReglaRef,
			RecursoReglaHuellaSHA256: r.Recurso.Huella, OrdenFinal: c.OrdenFinal, FinAutomatico: c.FinAutomatico}
		if r.SuspensionHasta != "" {
			hasta := r.SuspensionHasta
			consecuencia.SuspensionHasta = &hasta
		}
	}
	copia := append([]byte(nil), contenido...)
	return s.buzon.RegistrarNoIncorporacion(ctx, ports.EventoNoIncorporacionRecibido{Evento: evento, Contenido: copia, HuellaSHA256: huellaSHA256,
		OrigenCreadaEn: origenCreadaEn.UTC(), OrigenPosicion: origenPosicion, Consecuencia: consecuencia})
}
