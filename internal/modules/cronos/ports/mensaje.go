package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrComunicacionNoAcreditada = errors.New("cronos comunicacion no acreditada")

// Los proveedores son nominales: la composición debe conectar V3 real.
type ProveedorAutorizacionComunicaciones interface {
	ProveerArchivoMensaje(context.Context, MaterialArchivoMensaje) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	ProveerMensajeResolucion(context.Context, MaterialMensajeResolucion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// OrdenComunicaciones solo se forma en el servidor tras resolver identidad y
// empleado propio. El proveedor y repositorio revalidan concesión en el efecto.
type OrdenComunicaciones struct {
	actor     vecdomain.ContextoActor
	proveedor ProveedorAutorizacionComunicaciones
}

func NuevaOrdenComunicaciones(actor vecdomain.ContextoActor, proveedor ProveedorAutorizacionComunicaciones) (OrdenComunicaciones, error) {
	if actor.Validar() != nil || proveedor == nil {
		return OrdenComunicaciones{}, ErrComunicacionNoAcreditada
	}
	copia, err := actor.Clonar()
	if err != nil {
		return OrdenComunicaciones{}, ErrComunicacionNoAcreditada
	}
	return OrdenComunicaciones{actor: copia, proveedor: proveedor}, nil
}
func (o OrdenComunicaciones) Actor() (vecdomain.ContextoActor, error) {
	if o.proveedor == nil || o.actor.Validar() != nil {
		return vecdomain.ContextoActor{}, ErrComunicacionNoAcreditada
	}
	return o.actor.Clonar()
}
func (o OrdenComunicaciones) Proveedor() ProveedorAutorizacionComunicaciones { return o.proveedor }

type SolicitudMensajeResolucion struct {
	ResolucionRef, ClaveOperacion string
	ResolucionVersion             int64
}

type MaterialMensajeResolucion struct {
	ActorRef, PerfilRef, ResolucionRef, ClaveOperacion string
	ResolucionVersion                                  int64
	InstanteUTC                                        time.Time
}

type MaterialArchivoMensaje struct {
	ActorRef, PerfilRef string
	Archivo             domain.ArchivoMensaje
}

type ReciboMensaje struct {
	Referencia, MensajeRef string
	Estado                 domain.EstadoMensaje
	Version                int64
	InstanteUTC            time.Time
	Replay                 bool
}

type EventoMensaje struct {
	MensajeRef  string
	Version     int64
	Estado      domain.EstadoMensaje
	InstanteUTC time.Time
	ReciboRef   string
}

// La creación consulta dentro de la transacción la resolución auténtica y su
// versión, destinatario, decisión y texto seguro. No recibe un "concedido" ni
// un texto libre del cliente. La falta de fuente impide crear el aviso.
type RepositorioMensajes interface {
	RegistrarDesdeResolucionAutorizada(context.Context, MaterialMensajeResolucion, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboMensaje, error)
	ArchivarAutorizado(context.Context, MaterialArchivoMensaje, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboMensaje, error)
	// Lecturas con concesión positiva, ámbito propio y auditoría real.
	ListarPropios(context.Context, vecdomain.ContextoActor, string, bool) ([]domain.MensajeResolucion, error)
	ConsultarPropio(context.Context, vecdomain.ContextoActor, string, string) (domain.MensajeResolucion, error)
	HistoriaPropia(context.Context, vecdomain.ContextoActor, string, string) ([]EventoMensaje, error)
}

type CasoUsoMensajes interface {
	RegistrarMensajeResolucion(context.Context, OrdenComunicaciones, SolicitudMensajeResolucion) (ReciboMensaje, error)
	ArchivarMensaje(context.Context, OrdenComunicaciones, domain.ArchivoMensaje) (ReciboMensaje, error)
	ListarMensajes(context.Context, OrdenComunicaciones, bool) ([]domain.MensajeResolucion, error)
	ConsultarMensaje(context.Context, OrdenComunicaciones, string) (domain.MensajeResolucion, error)
	ConsultarHistoriaMensaje(context.Context, OrdenComunicaciones, string) ([]EventoMensaje, error)
}
