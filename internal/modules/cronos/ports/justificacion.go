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
var ErrPoliticaJustificacionNoVigente = errors.New("cronos: politica de justificacion no vigente")
var ErrJustificacionNoEncontrada = errors.New("cronos: justificacion no encontrada")

type ProveedorMaterialJustificacion interface {
	ProveerMaterialJustificacion(context.Context, domain.MaterialJustificacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// Las dos lecturas piden decisiones nuevas, distintas de las escrituras.
type ProveedorLecturaJustificacion interface {
	ProveerMaterialConsultaJustificacion(context.Context, domain.MaterialConsultaJustificacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	ProveerMaterialReciboJustificacion(context.Context, domain.MaterialReciboJustificacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	ProveerMaterialReciboPorClaveJustificacion(context.Context, domain.MaterialReciboPorClaveJustificacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
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
	Solicitud         domain.SolicitudJustificable `json:"solicitud"`
	Politica          domain.PoliticaJustificacion `json:"politica"`
	PoliticaVigente   bool                         `json:"politica_vigente"`
	PoliticaSintetica *bool                        `json:"politica_sintetica"`
	Actual            *domain.Justificacion        `json:"actual"`
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
	Documento            domain.DocumentoJustificacion `json:"documento"`
	ModuloID             string                        `json:"modulo_id"`
	ExpedienteRef        string                        `json:"expediente_ref"`
	TipoRef              string                        `json:"tipo_ref"`
	NumeroVEC            string                        `json:"numero_vec"`
	CreadoEnUTC          time.Time                     `json:"creado_en_utc"`
	PoliticaRef          string                        `json:"politica_ref"`
	PoliticaVersion      uint64                        `json:"politica_version"`
	PoliticaSHA256       string                        `json:"politica_sha256"`
	ConservacionHastaUTC time.Time                     `json:"conservacion_hasta_utc"`
	Proteccion           string                        `json:"proteccion"`
	EstadoPolitica       string                        `json:"estado_politica"`
}
type ReciboJustificacion struct {
	Justificacion  domain.Justificacion          `json:"justificacion"`
	Registro       *RegistroDocumentalConfirmado `json:"registro"`
	HuellaMaterial string                        `json:"huella_material"`
	ReciboRef      string                        `json:"recibo_ref"`
	FechaUTC       time.Time                     `json:"fecha_utc"`
	Replay         bool                          `json:"replay"`
}

// Recuperar reautoriza lectura en la petición actual; una misma clave con
// material distinto es conflicto. Confirmar revalida autorización exacta,
// catálogo, solicitud concedida, CAS e idempotencia bajo lock, en la misma
// transacción que historia append-only, auditoría común, recibo y outbox.
// Documentos tiene una transacción independiente: este contrato no la deshace.
type RepositorioJustificacion interface {
	RecuperarJustificacion(context.Context, OrdenJustificacion, domain.MaterialJustificacion) (ReciboJustificacion, bool, error)
	RecuperarRevisionPorClave(context.Context, OrdenJustificacion, domain.MaterialReciboPorClaveJustificacion) (ReciboJustificacion, bool, error)
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

type PeticionRecuperacionRevisionJustificacion struct {
	SolicitudRef, ClaveOperacion string
	VersionEsperada              int64
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
