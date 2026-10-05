package administracion

import (
	"context"
	"errors"
	"net/http"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/ports"
)

// Los activos contienen solo código/textos públicos. La fuente nominal debe
// confirmar lista propia y auditoría común en una sola transacción antes de
// elegir perfil. Servir HTML nunca concede una consulta de negocio.
func (h *handlerPerfilesADMIN) estadoActivosSelector(ctx context.Context, r *http.Request) int {
	o, err := h.observador.ObservarADMIN(ctx, r)
	if err == nil && (r.Host != h.host.autoridad || o.Host != h.host.nombre || o.Audiencia != h.audienciaSelector || !o.Valida(h.reloj.Ahora().UTC())) {
		err = api.ErrAutenticacionRequerida
	}
	if err == nil {
		if h.fuenteSeleccion == nil {
			err = ports.ErrAutoridadAdministracionPerfilesNoDisponible
		} else {
			resultado, e := h.fuenteSeleccion.ListarPropiosAuditadosADMIN(ctx, o)
			err = e
			if err == nil && (!resultado.Propios.Validos() || len(resultado.Propios.Perfiles) == 0) {
				err = api.ErrAccesoDenegado
			}
			if err == nil && !referenciaAuditoriaComun(resultado.AuditoriaComunRef) {
				err = ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
		}
	}
	if err == nil {
		return http.StatusOK
	}
	estado, codigo := http.StatusServiceUnavailable, "servicio_no_disponible"
	if errors.Is(err, api.ErrAutenticacionRequerida) {
		estado, codigo = http.StatusUnauthorized, "autenticacion_requerida"
	}
	if errors.Is(err, api.ErrAccesoDenegado) {
		estado, codigo = http.StatusForbidden, "acceso_denegado"
	}
	return h.denegarActivos(ctx, estado, codigo)
}
