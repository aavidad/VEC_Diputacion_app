package ports

import (
	"context"
	"errors"
	"time"

	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	DominioFinalizacionCT      = "bolsa.finalizacion.v1"
	AccionConsultarOrigenCT    = "bolsa.origen_ct.consultar"
	FinalidadConsultarOrigenCT = "gestionar_contratacion_temporal"
)

var (
	ErrOrigenLlamamientoCTNoDisponible = errors.New("bolsa: origen CT no disponible")
	ErrOrigenLlamamientoCTNoEncontrado = errors.New("bolsa: origen CT no encontrado")
	ErrOrigenLlamamientoCTAmbiguo      = errors.New("bolsa: origen CT ambiguo")
	ErrOrigenLlamamientoCTConflicto    = errors.New("bolsa: origen CT divergente")
)

// EntradaOrigenLlamamientoCT expresa el orden decidido por Bolsa en un corte.
// Un orden vigente ausente significa que Bolsa no permite llamar a la persona.
type EntradaOrigenLlamamientoCT struct {
	OrdenActa        uint64  `json:"orden_acta"`
	OrdenVigente     *uint64 `json:"orden_vigente"`
	ParticipacionRef string  `json:"participacion_ref"`
	CandidatoRef     string  `json:"candidato_ref"`
	Situacion        string  `json:"situacion"`
	Razon            string  `json:"razon"`
	Elegible         bool    `json:"elegible"`
}

// OrigenLlamamientoCT conserva los bytes canónicos originales y la decisión
// de orden de Bolsa. Contratación no interpreta situaciones ni reglas.
type OrigenLlamamientoCT struct {
	ActaRef                 string                       `json:"acta_ref"`
	CategoriaRef            string                       `json:"categoria_ref"`
	BolsaRef                string                       `json:"bolsa_ref"`
	VersionBolsa            uint64                       `json:"version_bolsa"`
	HuellaBolsaSHA256       string                       `json:"huella_bolsa_sha256"`
	BolsaCanonica           []byte                       `json:"bolsa_canonica"`
	InstantaneaRef          string                       `json:"instantanea_ref"`
	VersionInstantanea      uint64                       `json:"version_instantanea"`
	HuellaInstantaneaSHA256 string                       `json:"huella_instantanea_sha256"`
	InstantaneaCanonica     []byte                       `json:"instantanea_canonica"`
	PoliticaRef             string                       `json:"politica_ref"`
	VersionPolitica         uint64                       `json:"version_politica"`
	HuellaPoliticaSHA256    string                       `json:"huella_politica_sha256"`
	PoliticaProvisional     bool                         `json:"politica_provisional"`
	Corte                   time.Time                    `json:"corte"`
	BolsaVigenteHasta       *time.Time                   `json:"bolsa_vigente_hasta"`
	PoliticaVigenteHasta    *time.Time                   `json:"politica_vigente_hasta"`
	Entradas                []EntradaOrigenLlamamientoCT `json:"entradas"`
}

// FuenteOrigenLlamamientoCT devuelve exactamente una constitución vigente
// de la categoría. Ausencia, ambigüedad o historia inconsistente deniegan.
type FuenteOrigenLlamamientoCT interface {
	ConsultarOrigenLlamamientoCT(context.Context, string) (OrigenLlamamientoCT, error)
}

// OrigenEmisionCT fija la causa CT a una reserva B7 nativa. El repositorio
// coteja estos campos con su constitución y emisión en la misma transacción.
type OrigenEmisionCT struct {
	OrganizacionRef         string              `json:"organizacion_ref"`
	UnidadRef               string              `json:"unidad_ref"`
	ExpedienteRef           string              `json:"expediente_ref"`
	VersionExpediente       uint64              `json:"version_expediente"`
	ClaveCT                 string              `json:"clave_ct"`
	NecesidadRef            string              `json:"necesidad_ref"`
	ActaRef                 string              `json:"acta_ref"`
	CategoriaRef            string              `json:"categoria_ref"`
	BolsaRef                string              `json:"bolsa_ref"`
	VersionBolsa            uint64              `json:"version_bolsa"`
	HuellaBolsaSHA256       string              `json:"huella_bolsa_sha256"`
	InstantaneaRef          string              `json:"instantanea_ref"`
	VersionInstantanea      uint64              `json:"version_instantanea"`
	HuellaInstantaneaSHA256 string              `json:"huella_instantanea_sha256"`
	PoliticaRef             string              `json:"politica_ref"`
	VersionPolitica         uint64              `json:"version_politica"`
	HuellaPoliticaSHA256    string              `json:"huella_politica_sha256"`
	ParticipacionRef        string              `json:"participacion_ref"`
	CandidatoRef            string              `json:"candidato_ref"`
	OrdenSeleccionado       uint64              `json:"orden_seleccionado"`
	Lectura                 OrigenLlamamientoCT `json:"lectura"`
	VersionToken            string              `json:"version_token"`
}

// DatosDerivacionTokenFinalizacionCT liga la finalización B7 a una reserva CT.
// El derivador usa un secreto privado persistente, HMAC-SHA256 y el dominio
// separado bolsa.finalizacion.v1. VersionClave vacía pide la versión actual;
// en replay se exige la versión guardada en el origen.
type DatosDerivacionTokenFinalizacionCT struct {
	BolsaRef           string
	ClaveB7            string
	ActorRef           string
	HuellaOrigenSHA256 string
	VersionClave       string
}

type DerivadorTokenFinalizacionCT interface {
	DerivarTokenFinalizacionCT(context.Context, DatosDerivacionTokenFinalizacionCT) ([]byte, string, error)
}

type ReciboOrigenEmisionCT struct {
	OrigenEmisionCT
	LlamamientoRef string    `json:"llamamiento_ref"`
	ReciboRef      string    `json:"recibo_ref"`
	DecisionRef    string    `json:"decision_ref"`
	AuditoriaRef   string    `json:"auditoria_ref"`
	EventoRef      string    `json:"evento_ref"`
	EmitidoEn      time.Time `json:"emitido_en"`
	Reutilizada    bool      `json:"reutilizada"`
}

type LectorOrigenEmisionCT interface {
	RecuperarOrigenEmisionCT(context.Context, string, string) (ReciboOrigenEmisionCT, error)
	RecuperarLecturaOrigenCT(context.Context, string, string) (OrigenLlamamientoCT, error)
	RecuperarReservaLlamamientoCT(context.Context, string, string) (EmisionLlamamiento, error)
}

// ReservaOrigenCT proyecta sólo la identidad de la reserva necesaria para
// cotejar el replay y los diez campos de configuración concedidos para B7.
// La lectura de origen viaja una sola vez en OrigenLlamamientoCT.
type ReservaOrigenCT struct {
	ExpedienteRef     string                   `json:"expediente_ref"`
	ClaveCT           string                   `json:"clave_ct"`
	UnidadRef         string                   `json:"unidad_ref"`
	ParticipacionRef  string                   `json:"participacion_ref"`
	CandidatoRef      string                   `json:"candidato_ref"`
	OrdenSeleccionado uint64                   `json:"orden_seleccionado"`
	Configuracion     ConfiguracionLlamamiento `json:"configuracion"`
}

// CrearCapturaOrigenCT construye la orden positiva a partir de la referencia
// opaca de Bolsa que SQL resolvió y del contexto nominal actual.
type CapturaOrigenCT struct {
	Permiso   vecports.CapturaLecturaNominalBolsa
	Auditoria vecports.OrdenIntentoAuditoria
}

type CrearCapturaOrigenCT func(string) (CapturaOrigenCT, error)

// FuenteOrigenLlamamientoCTAuditada entrega datos sólo con acuse positivo
// confirmado en la misma transacción PostgreSQL. La reserva es nil cuando
// aún no hubo emisión B7 para expediente y clave.
type FuenteOrigenLlamamientoCTAuditada interface {
	ConsultarOrigenCTAuditado(context.Context, string, string, string, string, []byte, []byte, CrearCapturaOrigenCT) (
		OrigenLlamamientoCT, *ReservaOrigenCT, vecports.AcuseIntentoAuditoria, error)
}
