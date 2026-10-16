package bootstrap

import (
	"context"
	"errors"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type lectorExclusionesOfrecimientoBolsaDesarrollo interface {
	ConsultarExclusionesOfrecimientoRRHH(context.Context, string, string, int64) (puertosbolsa.PaginaExclusionesOfrecimientoRRHH, error)
}

func (h *bolsasRRHHDesarrollo) consultarExclusionesOrigen(ctx context.Context, bolsaRef, llamamientoRef string, desde int64) map[string]any {
	if h.exclusionesOrigen == nil {
		return map[string]any{"estado": "no_disponible", "error_estado": "lectura_no_disponible"}
	}
	pagina, err := h.exclusionesOrigen.ConsultarExclusionesOfrecimientoRRHH(ctx, bolsaRef, llamamientoRef, desde)
	if errors.Is(err, vecdomain.ErrAutorizacionDenegada) || errors.Is(err, vecdomain.ErrPermissionDenied) {
		return map[string]any{"estado": "sin_permiso"}
	}
	if err != nil || !pagina.ValidaPara(puertosbolsa.ConsultaExclusionesOfrecimientoRRHH{
		BolsaRef: bolsaRef, LlamamientoRef: llamamientoRef, DesdeOrdenActa: desde,
		Limite: puertosbolsa.LimitePaginaExclusionesOfrecimientoRRHH,
	}) {
		return map[string]any{"estado": "no_disponible", "error_estado": "lectura_no_disponible"}
	}
	return map[string]any{"estado": "disponible", "pagina": pagina}
}
