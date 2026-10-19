package bootstrap

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	appbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

// La audiencia nueva es opcional: su falta no interrumpe el resto de Bolsa.
func proveedorComprobacionOfrecimiento(ctx context.Context, gobierno *pgxpool.Pool,
	material materialAtestacionContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo,
) *proveedorMaterialAltaContratacionTemporalDesarrollo {
	d := descriptorMaterialConsumidorV3Desarrollo{Audiencia: ports.AudienciaComprobacionOfrecimiento,
		Dominio:          "vec.bolsa.comprobacion-ofrecimiento.capacidad-v3",
		Prefijo:          "clave:capacidad:bolsa-comprobacion-ofrecimiento:",
		ProveedorNominal: "proveedor-material-comprobacion-ofrecimiento-bolsa"}
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo([]descriptorMaterialConsumidorV3Desarrollo{d})
	if err != nil {
		slog.Warn("comprobacion_ofrecimiento_no_habilitada", "etapa", "catalogo")
		return nil
	}
	p, err := nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogo, d.Audiencia)
	if err != nil {
		slog.Warn("comprobacion_ofrecimiento_no_habilitada", "etapa", "material")
		return nil
	}
	return p
}

func motivoComprobacionOfrecimiento() vd.ReferenciaEntradaCatalogo {
	return vd.ReferenciaEntradaCatalogo{CatalogoID: "motivos_comprobacion_ofrecimiento_bolsa", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-bolsa-comprobacion-ofrecimiento-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-comprobacion-ofrecimiento-registrar")}
}

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararComprobacionOfrecimiento(ctx context.Context,
	e bolsahttp.EntradaComprobacionOfrecimiento,
) (ports.SolicitudComprobacion, error) {
	var cero ports.SolicitudComprobacion
	c, err := p.contextoRevalidado(ctx)
	if err != nil {
		return cero, err
	}
	correlacion, err := vd.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.generar)
	if err != nil {
		return cero, err
	}
	return ports.SolicitudComprobacion{Vinculo: c.Vinculo, ResultadoContexto: c.Resultado,
		Correlacion: correlacion, MotivoAutorizacion: motivoComprobacionOfrecimiento(),
		Contexto: e.Contexto, Hechos: e.Hechos, VersionEsperada: e.VersionEsperada, ClaveIdempotencia: e.ClaveIdempotencia}, nil
}

func montarComprobacionOfrecimiento(pool *pgxpool.Pool, p *preparadorBorradorLlamamientoDesarrollo,
	emisor *emisorBorradorLlamamientoDesarrollo,
) http.Handler {
	var servicio ports.ServicioComprobacion
	if emisor != nil && emisor.comprobacion != nil {
		r, err := postgresbolsa.NuevoRepositorioComprobacionPostgreSQL(pool)
		if err == nil {
			servicio, err = appbolsa.NuevoServicioComprobacionOfrecimiento(p, emisor, r)
		}
		if err != nil {
			slog.Warn("comprobacion_ofrecimiento_no_habilitada", "etapa", "composicion")
		}
	}
	return bolsahttp.NuevoHandlerComprobacionOfrecimiento(p, servicio)
}
