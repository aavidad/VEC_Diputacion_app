package ports

import (
	"context"
	"errors"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

const (
	ProtocoloIncorporacionPersonalB2 = "personal_b2_v1"
	AccionRegistrarPlanNominalB2     = "contratacion_temporal.incorporacion_personal.plan.registrar"
	AccionLeerPlanNominalB2          = "contratacion_temporal.incorporacion_personal.plan.consultar"
	AccionConfirmarOrigenB2          = "contratacion_temporal.incorporacion_personal.origen.confirmar"
	AudienciaRegistrarPlanNominalB2  = "vec_contratacion_temporal.incorporacion_personal.plan.registrar.v1"
	AudienciaLeerPlanNominalB2       = "vec_contratacion_temporal.incorporacion_personal.plan.consultar.v1"
	AudienciaConfirmarOrigenB2       = "vec_contratacion_temporal.incorporacion_personal.origen.confirmar.v1"
	TipoRecursoPlanNominalB2         = "incorporacion_personal_ct"
	FinalidadPlanNominalB2           = "incorporar_personal_desde_ct"
)

var (
	ErrPlanNominalB2Invalido     = domain.ErrPlanIncorporacionPersonalB2
	ErrPlanNominalB2Conflicto    = errors.New("contratacion temporal: conflicto de incorporacion personal")
	ErrPlanNominalB2Denegado     = errors.New("contratacion temporal: incorporacion personal denegada")
	ErrPlanNominalB2NoEncontrado = errors.New("contratacion temporal: plan nominal personal no encontrado")
	ErrPlanNominalB2NoDisponible = errors.New("contratacion temporal: incorporacion personal no disponible")
)

type ActorIncorporacionPersonalB2 = core.ContextoActor

// Selección humana: el servidor resuelve persona, recibos y autoridad.
type SolicitudPlanNominalB2 struct {
	OrganizacionRef     string                       `json:"organizacion_ref"`
	ExpedienteRef       string                       `json:"expediente_ref"`
	VersionExpediente   uint64                       `json:"version_expediente"`
	PuestoRef           string                       `json:"puesto_ref"`
	PlazaRef            string                       `json:"plaza_ref"`
	VersionPlantillaRef string                       `json:"version_plantilla_ref"`
	VersionRPTRef       string                       `json:"version_rpt_ref"`
	Regimen             domain.EntradaPlanPersonalB2 `json:"regimen"`
	Modalidad           domain.EntradaPlanPersonalB2 `json:"modalidad"`
	ClaseOcupacion      string                       `json:"clase_ocupacion"`
	Desde               string                       `json:"desde"`
	Hasta               string                       `json:"hasta"`
	MotivoClave         string                       `json:"motivo_clave"`
	DocumentoRef        string                       `json:"documento_ref"`
	DocumentoSHA256     string                       `json:"documento_sha256"`
	ClaveIdempotencia   string                       `json:"clave_idempotencia"`
}
type ContratoPlanNominalB2 struct {
	Protocolo                string                             `json:"protocolo"`
	PlanRef                  string                             `json:"plan_ref"`
	PlanVersion              uint64                             `json:"plan_version"`
	PlanReciboRef            string                             `json:"plan_recibo_ref"`
	PlanSHA256               string                             `json:"plan_sha256"`
	IntencionRef             string                             `json:"intencion_ref"`
	IntencionReciboRef       string                             `json:"intencion_recibo_ref"`
	IntencionVersion         uint64                             `json:"intencion_version"`
	SolicitudPersonalRef     string                             `json:"solicitud_personal_ref"`
	IdempotenciaPersonalUUID string                             `json:"idempotencia_personal_uuid"`
	Solicitud                SolicitudPlanNominalB2             `json:"solicitud"`
	Material                 domain.PlanIncorporacionPersonalB2 `json:"material"`
	RegistradoEn             time.Time                          `json:"registrado_en"`
}

// PrefijoReciboOrigenIncorporacionPersonalB2 es el prefijo con el que CT155/
// CT160 emiten el recibo del origen B2 y con el que su disparador distingue,
// en cese y GINPIX, una incorporación personal_b2_v1 de una ejercicio_v2.
const PrefijoReciboOrigenIncorporacionPersonalB2 = "recibo:incorporacion-personal-b2:"

// ReciboOrigenIncorporacionPersonalB2 dice si la incorporación que CT asocia
// al expediente es un origen B2, con la misma regla que aplica su SQL.
func ReciboOrigenIncorporacionPersonalB2(ref string) bool {
	return len(ref) > len(PrefijoReciboOrigenIncorporacionPersonalB2) && domain.ReferenciaOpacaValida(ref) &&
		ref[:len(PrefijoReciboOrigenIncorporacionPersonalB2)] == PrefijoReciboOrigenIncorporacionPersonalB2
}

// HechosPersonalIncorporacionB2 es una proyección del propietario, releída con
// permiso actual. No admite las referencias de recibos como prueba por sí solas.
type HechosPersonalIncorporacionB2 struct {
	ModoPersonal          string `json:"modo_personal"`
	PersonalPlanRef       string `json:"personal_plan_ref"`
	PersonalPlanVersion   uint64 `json:"personal_plan_version"`
	PersonalPlanReciboRef string `json:"personal_plan_recibo_ref"`
	PersonalPlanSHA256    string `json:"personal_plan_sha256"`
	EmpleadoRef           string `json:"empleado_ref"`
	AltaReciboRef         string `json:"alta_recibo_ref"`
	RelacionRef           string `json:"relacion_ref"`
	RelacionVersion       uint64 `json:"relacion_version"`
	RelacionReciboRef     string `json:"relacion_recibo_ref"`
	OcupacionRef          string `json:"ocupacion_ref"`
	OcupacionVersion      uint64 `json:"ocupacion_version"`
	OcupacionReciboRef    string `json:"ocupacion_recibo_ref"`
	RPTConfirmacionRef    string `json:"rpt_confirmacion_ref"`
	RPTReciboRef          string `json:"rpt_recibo_ref"`
	SeguimientoRef        string `json:"seguimiento_ref"`
}
type ConfirmacionOrigenIncorporacionB2 struct {
	OrganizacionRef string                        `json:"organizacion_ref"`
	UnidadCTRef     string                        `json:"unidad_ct_ref"`
	ExpedienteRef   string                        `json:"expediente_ref"`
	PlanRef         string                        `json:"plan_ref"`
	PlanVersion     uint64                        `json:"plan_version"`
	PlanSHA256      string                        `json:"plan_sha256"`
	Hechos          HechosPersonalIncorporacionB2 `json:"hechos"`
}
type OrigenIncorporacionPersonalB2 struct {
	Protocolo              string                            `json:"protocolo"`
	Confirmacion           ConfirmacionOrigenIncorporacionB2 `json:"confirmacion"`
	ReciboRef              string                            `json:"recibo_ref"`
	AuditoriaRef           string                            `json:"auditoria_ref"`
	OutboxRef              string                            `json:"outbox_ref"`
	RegistradoEn           time.Time                         `json:"registrado_en"`
	FirmaOficial           bool                              `json:"firma_oficial"`
	EficaciaAdministrativa bool                              `json:"eficacia_administrativa"`
}
type FuentePreparacionPlanNominalB2 interface {
	ResolverPlanNominalB2(context.Context, SolicitudPlanNominalB2, ActorIncorporacionPersonalB2) (domain.PlanIncorporacionPersonalB2, error)
	VerificarHechosPersonalB2(context.Context, ContratoPlanNominalB2, HechosPersonalIncorporacionB2, ActorIncorporacionPersonalB2) (HechosPersonalIncorporacionB2, error)
}
type AutoridadPlanNominalB2 interface {
	AutorizarPlanNominalB2(context.Context, string, []byte, ActorIncorporacionPersonalB2) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
type RegistroPlanNominalB2 struct {
	Solicitud SolicitudPlanNominalB2             `json:"solicitud"`
	Material  domain.PlanIncorporacionPersonalB2 `json:"material"`
}
type RepositorioPlanNominalB2 interface {
	RegistrarPlanNominalB2(context.Context, RegistroPlanNominalB2, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ContratoPlanNominalB2, error)
	LeerContratoPlanNominal(context.Context, string, string, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, ...string) (ContratoPlanNominalB2, error)
	ConfirmarOrigenIncorporacionB2(context.Context, ConfirmacionOrigenIncorporacionB2, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (OrigenIncorporacionPersonalB2, error)
}

// AntecedentesPlanNominalB2 sólo contiene referencias propias de CT.
type AntecedentesPlanNominalB2 struct {
	OrganizacionRef     string                     `json:"organizacion_ref"`
	ExpedienteRef       string                     `json:"expediente_ref"`
	VersionExpediente   uint64                     `json:"version_expediente"`
	AnalisisVersion     uint64                     `json:"analisis_version"`
	AnalisisReciboRef   string                     `json:"analisis_recibo_ref"`
	AnalisisSHA256      string                     `json:"analisis_sha256"`
	CategoriaRef        string                     `json:"categoria_ref"`
	PropuestaReciboRef  string                     `json:"propuesta_recibo_ref"`
	AceptacionRef       string                     `json:"aceptacion_ref"`
	AceptacionReciboRef string                     `json:"aceptacion_recibo_ref"`
	DocumentoRef        string                     `json:"documento_ref"`
	DocumentoSHA256     string                     `json:"documento_sha256"`
	DocumentoVersion    uint64                     `json:"documento_version"`
	Bolsa               domain.SelectorBolsaPlanB2 `json:"bolsa"`
	Vinculo             *EstadoVinculoCategoriaRPT `json:"vinculo"`
}
type LectorOrigenIncorporacionB2 interface {
	LeerOrigenIncorporacionB2(context.Context, string, string, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, ...string) (OrigenIncorporacionPersonalB2, bool, error)
}

// La unidad CT procede del detalle autorizado vigente, distinta de la unidad
// de destino de Personal. No se acepta como selector del canal.
type FuenteUnidadPlanNominalB2 interface {
	ResolverUnidadPlanNominalB2(context.Context, string, string) (string, error)
}
