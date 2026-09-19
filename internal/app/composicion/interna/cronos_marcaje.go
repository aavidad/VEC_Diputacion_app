package interna

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	cronoshttp "vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	cronospg "vec-diputacion-granada/internal/modules/cronos/adapters/postgres"
	cronosapp "vec-diputacion-granada/internal/modules/cronos/application"
	cronosdomain "vec-diputacion-granada/internal/modules/cronos/domain"
	cronosports "vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// ResolutorMarcajePropio is fed only by the authenticated server capsule.
// It deliberately has no header, employee, role, or client-clock arguments.
type ResolutorMarcajePropio interface {
	ResolverContextoActorMarcajePropio(*http.Request) (vecdomain.ContextoActor, error)
	AcreditarCanalMarcajePropio(*http.Request) (cronosdomain.AcreditacionCanalMarcaje, error)
}

type resolutorHTTPMarcajePropio struct {
	fuente    ResolutorMarcajePropio
	proveedor cronosports.ProveedorMaterialMarcajePropio
}

func (r resolutorHTTPMarcajePropio) ResolverMarcajePropio(peticion *http.Request) (cronosports.ContextoMarcajePropio, error) {
	if r.fuente == nil || r.proveedor == nil || peticion == nil {
		return cronosports.ContextoMarcajePropio{}, errors.New("cronos contexto no disponible")
	}
	actor, err := r.fuente.ResolverContextoActorMarcajePropio(peticion)
	if err != nil {
		return cronosports.ContextoMarcajePropio{}, errors.New("cronos contexto no disponible")
	}
	canal, err := r.fuente.AcreditarCanalMarcajePropio(peticion)
	if err != nil || canal.Validar() != nil {
		return cronosports.ContextoMarcajePropio{}, errors.New("cronos contexto no disponible")
	}
	orden, err := cronosports.NuevaOrdenConsumoAutorizacion(actor, r.proveedor)
	if err != nil {
		return cronosports.ContextoMarcajePropio{}, errors.New("cronos contexto no disponible")
	}
	return cronosports.ContextoMarcajePropio{CanalAcreditado: canal, OrdenConsumo: orden}, nil
}

// NuevoManejadorMarcajePropio wires the real consumer: HTTP -> trusted actor
// capsule -> exact V3 material -> PostgreSQL module function -> receipt.
// Root mounting/listening remains intentionally outside this module-specific file.
func NuevoManejadorMarcajePropio(pool *pgxpool.Pool, reloj cronosports.Reloj, fuente ResolutorMarcajePropio, proveedor cronosports.ProveedorMaterialMarcajePropio, auditoria cronosports.RegistroResultadoEjecucionMarcaje) (*cronoshttp.ManejadorMarcajes, error) {
	if pool == nil || reloj == nil || fuente == nil || proveedor == nil || auditoria == nil {
		return nil, cronosports.ErrDependenciaNoDisponible
	}
	repo, err := cronospg.NuevoRepositorioMarcajes(pool, auditoria)
	if err != nil {
		return nil, err
	}
	servicio, err := cronosapp.NuevoServicioMarcajes(repo, reloj)
	if err != nil {
		return nil, err
	}
	return cronoshttp.NuevoManejadorMarcajes(servicio, resolutorHTTPMarcajePropio{fuente: fuente, proveedor: proveedor})
}
