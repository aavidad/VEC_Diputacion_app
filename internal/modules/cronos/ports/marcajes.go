package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrClaveOperacionEnConflicto = errors.New("cronos clave de operacion con contenido distinto")
	ErrDependenciaNoDisponible   = errors.New("cronos persistencia transaccional no disponible")
)

// ProveedorMaterialMarcajePropio es la autoridad nominal V3 de este efecto.
// Recibe material creado por el servidor; nunca bytes de HTTP.
type ProveedorMaterialMarcajePropio interface {
	ProveerMaterialMarcajePropio(context.Context, domain.MaterialAutorizacionMarcajePropio) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// OrdenConsumoAutorizacion no es concesión ni referencia cruda: lleva el
// ContextoActor ya resuelto por el servidor y proveedor nominal opaco.
type OrdenConsumoAutorizacion struct {
	contexto  vecdomain.ContextoActor
	proveedor ProveedorMaterialMarcajePropio
}

func NuevaOrdenConsumoAutorizacion(contexto vecdomain.ContextoActor, proveedor ProveedorMaterialMarcajePropio) (OrdenConsumoAutorizacion, error) {
	if contexto.Validar() != nil || proveedor == nil {
		return OrdenConsumoAutorizacion{}, ErrDependenciaNoDisponible
	}
	copia, err := contexto.Clonar()
	if err != nil {
		return OrdenConsumoAutorizacion{}, ErrDependenciaNoDisponible
	}
	return OrdenConsumoAutorizacion{contexto: copia, proveedor: proveedor}, nil
}
func (o OrdenConsumoAutorizacion) ContextoActor() (vecdomain.ContextoActor, error) {
	if o.proveedor == nil || o.contexto.Validar() != nil {
		return vecdomain.ContextoActor{}, ErrDependenciaNoDisponible
	}
	return o.contexto.Clonar()
}
func (o OrdenConsumoAutorizacion) ProveedorMaterial() ProveedorMaterialMarcajePropio {
	return o.proveedor
}

type ContextoMarcajePropio struct {
	CanalAcreditado domain.AcreditacionCanalMarcaje
	OrdenConsumo    OrdenConsumoAutorizacion
}

type SolicitudMarcajePropio struct {
	Movimiento     domain.PunchKind
	ClaveOperacion string
}
type ReciboMarcajePropio struct {
	Referencia         string    `json:"referencia"`
	InstanteUTC        time.Time `json:"instante_utc"`
	MarcajeOriginalRef string    `json:"marcaje_original_ref"`
	Replay             bool      `json:"replay"`
}
type Reloj interface{ AhoraUTC() time.Time }

// Transaction boundary: revalidate+consume grant, append-only original, receipt, audit and outbox.
type RepositorioMarcajes interface {
	RegistrarOriginalAutorizado(context.Context, domain.MarcajeOriginal, domain.MaterialAutorizacionMarcajePropio, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboMarcajePropio, error)
}
type CasoUsoRegistrarMarcajePropio interface {
	RegistrarMarcajePropio(context.Context, ContextoMarcajePropio, SolicitudMarcajePropio) (ReciboMarcajePropio, error)
}
