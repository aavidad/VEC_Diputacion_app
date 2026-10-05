package postgres

import (
	"context"
	"encoding/json"
	"log/slog"

	"vec-diputacion-granada/internal/vec/domain"
)

type materialFuentesLote struct {
	Esquema          string                  `json:"esquema"`
	Version          int                     `json:"version"`
	SolicitudSHA256  string                  `json:"solicitud_sha256"`
	OrganizacionRef  string                  `json:"organizacion_ref"`
	UnidadRef        string                  `json:"unidad_ref"`
	AmbitosPorCambio [][]DimensionFuenteLote `json:"ambitos_por_cambio"`
}

// Un fallo de fuente devuelve nil y el adaptador no emite decisión: la orden
// (también su repetición) responde no disponible hasta que vuelva la fuente.
func materialFuentesPrivadasLote(ctx context.Context, proveedor ProveedorAmbitosLote,
	organizacion string, s domain.SolicitudLoteAdministracionPerfiles) []byte {
	if ctx == nil || ctx.Err() != nil || ausente(proveedor) || len(s.Cambios) == 0 {
		return nil
	}
	unidad := s.Cambios[0].Objetivo.UnidadRef
	for _, cambio := range s.Cambios {
		if cambio.Objetivo.UnidadRef != unidad || cambio.Objetivo.CentroRef != "" {
			return nil
		}
	}
	ambitos, err := proveedor.ResolverUnidadLote(ctx, organizacion, unidad)
	if err != nil || !ambitosFuenteLoteValidos(ambitos, organizacion, unidad) {
		slog.Warn("vec.admin.lote.fuente_no_disponible")
		return nil
	}
	material := materialFuentesLote{
		Esquema: "vec.admin.perfiles.lote.fuentes.v1", Version: 1,
		SolicitudSHA256: s.HuellaSolicitudSHA256, OrganizacionRef: organizacion,
		UnidadRef: unidad, AmbitosPorCambio: make([][]DimensionFuenteLote, 0, len(s.Cambios)),
	}
	for range s.Cambios {
		material.AmbitosPorCambio = append(material.AmbitosPorCambio,
			clonarAmbitosFuenteLote(ambitos).Descriptores)
	}
	b, err := json.Marshal(material)
	if err != nil || len(b) > 65536 {
		slog.Warn("vec.admin.lote.material_fuentes_no_disponible")
		return nil
	}
	return b
}
