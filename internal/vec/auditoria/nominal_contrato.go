package auditoria

import (
	"context"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	RutaConsultaNominal           = "/api/vec/auditoria/nominal/consultas"
	AccionConsultarNominal        = "vec.auditoria.nominal.consultar"
	TipoRecursoNominal            = "auditoria_nominal"
	AudienciaConsumoNominal       = "vec_auditoria.consulta_nominal.v1"
	TipoRegistroConsumoConfirmado = "consumo_confirmado"
)

// FiltroNominal scopes a query to an exact anchor and an explicit time range.
// AntesSecuencia is supplied only by the service's authenticated cursor.
type FiltroNominal struct {
	ActorRef       string
	RecursoRef     string
	Accion         string
	Desde          time.Time
	Hasta          time.Time
	Limite         uint16
	AntesSecuencia uint64
	FinalidadRef   string
	MotivoRef      string
}

type ConsultaNominalAutorizada struct {
	Filtro            FiltroNominal
	Material          vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	Solicitud         vecdomain.SolicitudAutorizacionLigadaV3
	Decision          vecdomain.DecisionAutorizacionLigadaV3
	Confirmacion      vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	ResultadoContexto vecdomain.ResultadoContextoActorRegistradoV2
}

// RegistroNominal projects the committed AD3 consumption and its canonical
// decision. Consumption confirmation does not assert business success.
type RegistroNominal struct {
	AuditoriaRef    string    `json:"auditoria_ref"`
	Secuencia       uint64    `json:"secuencia"`
	ActorRef        string    `json:"actor_ref"`
	PerfilActivoRef string    `json:"perfil_activo_ref"`
	AsignacionRef   string    `json:"asignacion_ref"`
	VersionRolRef   string    `json:"version_rol_ref"`
	ModuloID        string    `json:"modulo_id"`
	Accion          string    `json:"accion"`
	RecursoRef      string    `json:"recurso_ref"`
	FinalidadRef    string    `json:"finalidad_ref"`
	CorrelacionRef  string    `json:"correlacion_ref"`
	Canal           string    `json:"canal"`
	RegistradaEn    time.Time `json:"registrada_en"`
	TipoRegistro    string    `json:"tipo_registro"`
}

// ReciboConsultaNominal must be produced by the same committed transaction
// that consumes authorization, reads rows and audits the query itself.
type ReciboConsultaNominal struct {
	AuditoriaRef       string    `json:"auditoria_ref"`
	DecisionRef        string    `json:"decision_ref"`
	EfectoRef          string    `json:"efecto_ref"`
	HuellaFiltroSHA256 string    `json:"huella_filtro_sha256"`
	RegistradaEn       time.Time `json:"registrada_en"`
	Resultado          string    `json:"resultado"`
}

type PaginaFuenteNominal struct {
	Registros []RegistroNominal
	Recibo    ReciboConsultaNominal
}

// FuenteAuditoriaNominal reads only the common internal AD3 chain. It must
// revalidate and consume the V3 capability within its own transaction and
// return no rows until COMMIT succeeds. It cannot provide a generic append.
type FuenteAuditoriaNominal interface {
	ConsultarAuditoriaNominal(context.Context, ConsultaNominalAutorizada) (PaginaFuenteNominal, error)
}

type PeticionNominal struct {
	Filtro   FiltroNominal
	Cursor   string
	Contexto ContextoConsulta
}

type PaginaNominal struct {
	Registros       []RegistroNominal     `json:"registros"`
	SiguienteCursor string                `json:"siguiente_cursor"`
	Recibo          ReciboConsultaNominal `json:"recibo"`
}

func CamposPermitidosNominal() []string {
	return []string{"accion", "actor_ref", "asignacion_ref", "auditoria_ref", "canal",
		"correlacion_ref", "finalidad_ref", "modulo_id", "perfil_activo_ref", "recurso_ref",
		"registrada_en", "secuencia", "tipo_registro", "version_rol_ref"}
}
