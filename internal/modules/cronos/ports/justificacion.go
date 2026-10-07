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
	RegistrarJustificante(context.Context, OrdenJustificacion, domain.SolicitudJustificable, domain.PoliticaJustificacion, domain.DocumentoJustificacion, string) (RegistroDocumentalConfirmado, error)
}

// Sale exclusivamente del servicio común de Documentos después de registrar
// una referencia externa. Acredita ese registro VEC, no la custodia externa.
type RegistroDocumentalConfirmado struct {
	Documento            domain.DocumentoJustificacion
	ModuloID             string
	ExpedienteRef        string
	TipoRef              string
	NumeroVEC            string
	CreadoEnUTC          time.Time
	PoliticaRef          string
	PoliticaVersion      uint64
	PoliticaSHA256       string
	ConservacionHastaUTC time.Time
	Proteccion           string
	EstadoPolitica       string
}
type ReciboJustificacion struct {
	Justificacion  domain.Justificacion
	Registro       *RegistroDocumentalConfirmado
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
	ConfirmarJustificacion(context.Context, domain.MaterialJustificacion, domain.Justificacion, *RegistroDocumentalConfirmado, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboJustificacion, error)
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
	Registro        *RegistroDocumentalConfirmado
	ReciboCronos    *ReciboJustificacion
	EnlacePendiente bool
}
