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
	ErrCorreccionNoAutorizada = errors.New("cronos correccion no autorizada")
	ErrCorreccionEnConflicto  = errors.New("cronos correccion en conflicto")
)

type SolicitudOlvidoMarcaje struct {
	ClaveOperacion     string
	MarcajeOriginalRef string
	HuecoDeclarado     bool
	Movimiento         domain.PunchKind
	FechaCivil         string
	HoraPretendida     string
}

type DecisionResponsableCorreccion struct {
	SolicitudRef, ClaveOperacion string
	VersionEsperada              uint64
	Resultado                    domain.ResultadoCorreccion
}

type ResolucionRRHHCorreccion = DecisionResponsableCorreccion

type AplicacionCorreccion struct {
	SolicitudRef, ClaveOperacion string
	VersionEsperada              uint64
}

type ClaveRecuperacionCorreccion struct {
	SolicitudRef, ClaveOperacion string
	Paso                         domain.PasoCorreccion
}

type ReciboCorreccion struct {
	SolicitudRef, ActuacionRef, ReciboRef string
	Estado                                domain.EstadoCorreccion
	Version                               uint64
	InstanteUTC                           time.Time
	Replay                                bool
}

// ProveedorMaterialCorreccion es nominal para cada paso. Su implementación
// consulta la autoridad V3; no se deriva del rol, sesión ni identidad solos.
type ProveedorMaterialCorreccion interface {
	ProveerMaterialCorreccion(context.Context, domain.MaterialAutorizacionCorreccion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// OrdenConsumoCorreccion sólo puede crearse con contexto V2 y proveedor V3
// válidos. El adaptador durable debe usar ambos en el mismo paso.
type OrdenConsumoCorreccion struct {
	actor     vecdomain.ContextoActor
	proveedor ProveedorMaterialCorreccion
}

func NuevaOrdenConsumoCorreccion(actor vecdomain.ContextoActor, proveedor ProveedorMaterialCorreccion) (OrdenConsumoCorreccion, error) {
	if actor.Validar() != nil || proveedor == nil {
		return OrdenConsumoCorreccion{}, ErrDependenciaNoDisponible
	}
	copia, err := actor.Clonar()
	if err != nil {
		return OrdenConsumoCorreccion{}, ErrDependenciaNoDisponible
	}
	return OrdenConsumoCorreccion{actor: copia, proveedor: proveedor}, nil
}

func (o OrdenConsumoCorreccion) ContextoActor() (vecdomain.ContextoActor, error) {
	if o.proveedor == nil || o.actor.Validar() != nil {
		return vecdomain.ContextoActor{}, ErrDependenciaNoDisponible
	}
	return o.actor.Clonar()
}

func (o OrdenConsumoCorreccion) ProveedorMaterial() ProveedorMaterialCorreccion { return o.proveedor }

// RepositorioCorrecciones debe hacer append-only, versión optimista e
// idempotencia semántica. Para cada efecto: resolver empleado propio o sujeto
// gobernado, obtener V3 nominal, revalidar y consumirla en la transacción de
// historia+recibo+auditoría+outbox. La aplicación es la única que puede crear
// un asiento compensatorio y afectar la proyección de jornada; nunca muta el
// original. Sin puente durable entre bases, devolver ErrDependenciaNoDisponible.
type RepositorioCorrecciones interface {
	SolicitarOlvido(context.Context, domain.SolicitudCorreccion, OrdenConsumoCorreccion) (ReciboCorreccion, error)
	RegistrarActuacion(context.Context, domain.ActuacionCorreccion, OrdenConsumoCorreccion) (ReciboCorreccion, error)
	RecuperarRecibo(context.Context, ClaveRecuperacionCorreccion, OrdenConsumoCorreccion) (ReciboCorreccion, error)
}

type CasoUsoCorrecciones interface {
	SolicitarOlvido(context.Context, OrdenConsumoCorreccion, SolicitudOlvidoMarcaje) (ReciboCorreccion, error)
	DecidirResponsable(context.Context, OrdenConsumoCorreccion, DecisionResponsableCorreccion) (ReciboCorreccion, error)
	ResolverRRHH(context.Context, OrdenConsumoCorreccion, ResolucionRRHHCorreccion) (ReciboCorreccion, error)
	AplicarResolucion(context.Context, OrdenConsumoCorreccion, AplicacionCorreccion) (ReciboCorreccion, error)
	RecuperarRecibo(context.Context, OrdenConsumoCorreccion, ClaveRecuperacionCorreccion) (ReciboCorreccion, error)
}
