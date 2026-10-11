package ports

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/domain"
)

const (
	AccionPrepararIngestaMayan  = "documentos.ingesta_mayan.preparar"
	AccionDespacharIngestaMayan = "documentos.ingesta_mayan.despachar"
	AccionResolverIngestaMayan  = "documentos.ingesta_mayan.resolver"
	AccionConciliarIngestaMayan = "documentos.ingesta_mayan.conciliar"
	AccionConsultarIngestaMayan = "documentos.ingesta_mayan.consultar"
	FinalidadIngestaMayan       = "custodiar_evidencia_meritos"
	EstadoIngestaPreparada      = "preparado"
	EstadoIngestaDespachada     = "despachado"
	EstadoIngestaConfirmada     = "confirmado"
	EstadoIngestaIndeterminada  = "indeterminado"
	EstadoIngestaRechazada      = "rechazado"
)

var ErrIngestaIndeterminada = errors.New("documentos: ingesta externa pendiente de conciliacion")
var ErrIngestaNoReenviable = errors.New("documentos: despacho externo ya registrado")

// PlanIngestaMayan conserva metadatos y referencias, nunca contenido. CargaID
// identifica una carga previamente admitida por Documentos y no concede por si
// solo acceso al original.
type PlanIngestaMayan struct {
	PlanRef               string    `json:"plan_ref"`
	CargaID               string    `json:"carga_id"`
	PersonaRef            string    `json:"persona_ref"`
	ExpedienteRef         string    `json:"expediente_ref"`
	EvidenciaRef          string    `json:"evidencia_ref"`
	ClaveIdempotencia     string    `json:"clave_idempotencia"`
	HuellaSolicitudSHA256 string    `json:"huella_solicitud_sha256"`
	HuellaSHA256          string    `json:"huella_sha256"`
	MIME                  string    `json:"mime"`
	Tamano                int64     `json:"tamano"`
	CargaVersion          int64     `json:"carga_version"`
	Version               int64     `json:"version"`
	Estado                string    `json:"estado"`
	DocumentoID           string    `json:"documento_id"`
	ArchivoID             string    `json:"archivo_id"`
	VersionID             string    `json:"version_id"`
	CreadoEn              time.Time `json:"creado_en"`
	ActualizadoEn         time.Time `json:"actualizado_en"`
}

func (p PlanIngestaMayan) ValidarBase() error {
	if !domain.ReferenciaOpacaValida(p.PlanRef) || !domain.ReferenciaOpacaValida(p.CargaID) ||
		!domain.ReferenciaOpacaValida(p.PersonaRef) || !domain.ReferenciaExpedienteModuloValida("meritos", p.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(p.EvidenciaRef) || !domain.ReferenciaOpacaValida(p.ClaveIdempotencia) ||
		!huellaHexIngesta(p.HuellaSolicitudSHA256) || !huellaHexIngesta(p.HuellaSHA256) ||
		p.Tamano < 1 || p.Tamano > 64<<20 || p.CargaVersion != 5 ||
		p.MIME == "" || len(p.MIME) > 255 {
		return ErrSolicitudInvalida
	}
	return nil
}

func huellaHexIngesta(v string) bool {
	if len(v) != 64 {
		return false
	}
	for _, c := range v {
		if c < '0' || c > '9' && c < 'a' || c > 'f' {
			return false
		}
	}
	return true
}

type DocumentoMayan struct {
	DocumentoID  string `json:"documento_id"`
	ArchivoID    string `json:"archivo_id"`
	VersionID    string `json:"version_id"`
	HuellaSHA256 string `json:"huella_sha256"`
	MIME         string `json:"mime"`
	Tamano       int64  `json:"tamano"`
}

func (d DocumentoMayan) ValidarContra(p PlanIngestaMayan) error {
	if !idMayanValido(d.DocumentoID) || !idMayanValido(d.ArchivoID) ||
		!idMayanValido(d.VersionID) || d.HuellaSHA256 != p.HuellaSHA256 ||
		d.Tamano != p.Tamano || d.MIME != p.MIME {
		return ErrSolicitudInvalida
	}
	return nil
}

func idMayanValido(s string) bool {
	n, err := strconv.ParseUint(s, 10, 64)
	return err == nil && n > 0 && strconv.FormatUint(n, 10) == s
}

type OrdenPrepararIngestaMayan struct {
	Plan         PlanIngestaMayan
	Autorizacion AutorizacionV3
}

type OrdenCambiarIngestaMayan struct {
	PlanRef, ExpedienteRef, Estado string
	VersionEsperada                int64
	Documento                      DocumentoMayan
	Autorizacion                   AutorizacionV3
}

type OrdenConsultarIngestaMayan struct {
	PlanRef, ExpedienteRef string
	Autorizacion           AutorizacionV3
}

// PreimagenIngestaMayan usa campos ordenados y exactos. SQL vuelve a cotejar
// accion, recurso y ambito contra el material V3 antes de cada efecto.
func PreimagenIngestaMayan(accion string, dato any) ([]byte, error) {
	if accion != AccionPrepararIngestaMayan && accion != AccionDespacharIngestaMayan &&
		accion != AccionResolverIngestaMayan && accion != AccionConciliarIngestaMayan &&
		accion != AccionConsultarIngestaMayan {
		return nil, ErrSolicitudInvalida
	}
	b, err := json.Marshal(struct {
		Accion string `json:"accion"`
		Dato   any    `json:"dato"`
	}{accion, dato})
	if err != nil {
		return nil, ErrSolicitudInvalida
	}
	return b, nil
}

func (a AutorizacionV3) ValidarParaIngestaMayan(accion, planRef, expedienteRef string, ahora time.Time) error {
	if !domain.ReferenciaOpacaValida(planRef) || !domain.ReferenciaExpedienteModuloValida("meritos", expedienteRef) ||
		a.Accion != accion || a.Finalidad != FinalidadIngestaMayan || a.RecursoRef != planRef ||
		a.AmbitoRef != expedienteRef || !domain.ReferenciaValida(a.PrincipalID) ||
		!domain.ReferenciaValida(a.PerfilActivoRef) || !domain.ReferenciaValida(a.CorrelacionRef) ||
		a.Material.ValidarEstructura() != nil || ahora.IsZero() {
		return ErrSolicitudInvalida
	}
	r := a.Material.ResumenCapacidad()
	if r.ValidarEstructura() != nil || r.AudienciaConsumo() != AudienciaV3 ||
		r.Operacion() != accion || r.EfectoRef() != planRef ||
		ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) {
		return ErrSolicitudInvalida
	}
	return nil
}

// RepositorioIngestaMayan consume V3 y escribe estado, CAS, auditoria y outbox
// en la misma transaccion. Despachar devuelve nuevo=false en cualquier replay:
// la aplicacion solo puede llamar al proveedor cuando nuevo=true.
type RepositorioIngestaMayan interface {
	PrepararIngesta(context.Context, OrdenPrepararIngestaMayan) (PlanIngestaMayan, error)
	DespacharIngesta(context.Context, OrdenCambiarIngestaMayan) (PlanIngestaMayan, bool, error)
	ResolverIngesta(context.Context, OrdenCambiarIngestaMayan) (PlanIngestaMayan, error)
	ConsultarIngesta(context.Context, OrdenConsultarIngestaMayan) (PlanIngestaMayan, error)
}

// CustodioMayan se implementa en el adaptador del proveedor. Reconciliar solo
// recibe un ID conocido y nunca repite POST.
type CustodioMayan interface {
	IngerirOriginal(context.Context, PlanIngestaMayan, io.Reader) (DocumentoMayan, error)
	ReconciliarOriginal(context.Context, DocumentoMayan) (DocumentoMayan, error)
}

// AutoridadIngestaMayan obtiene una decisión nueva para la preimagen exacta.
// El repositorio vuelve a consumir y revalidar el material en su transacción.
type AutoridadIngestaMayan interface {
	AutorizarIngestaMayan(context.Context, string, []byte, string, string) (AutorizacionV3, error)
}
