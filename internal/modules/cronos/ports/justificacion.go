package ports

import (
	"context"
	"errors"
	"time"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrJustificacionNoDisponible = errors.New("cronos: justificacion no disponible")
var ErrEnlaceJustificacionPendiente = errors.New("cronos: enlace documental pendiente")

type ProveedorMaterialJustificacion interface {
	ProveerMaterialJustificacion(context.Context, domain.MaterialJustificacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
type OrdenJustificacion struct {
	actor     vecdomain.ContextoActor
	proveedor ProveedorMaterialJustificacion
}

func NuevaOrdenJustificacion(a vecdomain.ContextoActor, p ProveedorMaterialJustificacion) (OrdenJustificacion, error) {
	if a.Validar() != nil || dependenciaRemotaNula(p) {
		return OrdenJustificacion{}, ErrJustificacionNoDisponible
	}
	c, e := a.Clonar()
	if e != nil {
		return OrdenJustificacion{}, ErrJustificacionNoDisponible
	}
	return OrdenJustificacion{c, p}, nil
}
func (o OrdenJustificacion) ContextoActor() (vecdomain.ContextoActor, error) {
	if dependenciaRemotaNula(o.proveedor) || o.actor.Validar() != nil {
		return vecdomain.ContextoActor{}, ErrJustificacionNoDisponible
	}
	return o.actor.Clonar()
}
func (o OrdenJustificacion) ProveedorMaterial() ProveedorMaterialJustificacion { return o.proveedor }

type PreparacionJustificacion struct {
	Solicitud domain.SolicitudJustificable
	Politica  domain.PoliticaJustificacion
	Actual    *domain.Justificacion
}

// Preparar debe acreditar enclave interno, Persona/Personal y permiso nominal
// de lectura del objeto exacto; audita acceso/denegación y coteja el catálogo
// original sin mutarlo. No hay implementación de esta autoridad en el ensayo.
type FuenteJustificacion interface {
	PrepararJustificacion(context.Context, OrdenJustificacion, string) (PreparacionJustificacion, error)
}
type DocumentosJustificacion interface {
	// Sin efectos: exige política común y autorizador nominal de la petición.
	PrepararRegistro(context.Context, OrdenJustificacion, domain.SolicitudJustificable, domain.PoliticaJustificacion) error
	RegistrarJustificante(context.Context, OrdenJustificacion, domain.SolicitudJustificable, domain.PoliticaJustificacion, domain.DocumentoJustificacion, string) (domain.DocumentoJustificacion, error)
}
type ReciboJustificacion struct {
	Justificacion  domain.Justificacion
	HuellaMaterial string
	ReciboRef      string
	FechaUTC       time.Time
	Replay         bool
}

// Recuperar reautoriza lectura en la petición actual; una misma clave con
// material distinto es conflicto. Confirmar revalida autorización exacta,
// catálogo, solicitud concedida, CAS e idempotencia bajo lock, en la misma
// transacción que historia append-only, auditoría común, recibo y outbox.
// Documentos tiene una transacción independiente: este contrato no la deshace.
type RepositorioJustificacion interface {
	RecuperarJustificacion(context.Context, OrdenJustificacion, domain.MaterialJustificacion) (ReciboJustificacion, bool, error)
	ConfirmarJustificacion(context.Context, domain.MaterialJustificacion, domain.Justificacion, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboJustificacion, error)
}
type PeticionAnexoJustificacion struct {
	SolicitudRef, ClaveOperacion string
	VersionEsperada              int64
	Documento                    domain.DocumentoJustificacion
}
type PeticionRevisionJustificacion struct {
	SolicitudRef, ClaveOperacion string
	VersionEsperada              int64
	Vinculo                      domain.VinculoJustificacion
	Decision                     domain.EstadoJustificacion
	MotivoRef                    string
}

// Sin recibo conjunto. Documento puede estar confirmado mientras el enlace
// espera recuperación; ReciboCronos sólo aparece tras confirmación coherente.
type ResultadoAnexoJustificacion struct {
	Documento       *domain.DocumentoJustificacion
	ReciboCronos    *ReciboJustificacion
	EnlacePendiente bool
}
