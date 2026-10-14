package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ProveedorMaterialDecisionCorreccion interface {
	ProveerMaterialBandejaCorrecciones(context.Context, domain.MaterialBandejaCorrecciones) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	ProveerMaterialActuacionCorreccion(context.Context, domain.MaterialActuacionCorreccion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	ProveerMaterialHistoriaCorreccion(context.Context, domain.MaterialHistoriaCorreccion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type PeticionHistoriaCorreccion struct{ SolicitudRef, Vista, Paso string }

type HistoriaActuacionCorreccion struct {
	Paso        domain.PasoCorreccion      `json:"paso"`
	Resultado   domain.ResultadoCorreccion `json:"resultado"`
	Motivo      string                     `json:"motivo"`
	ActorRef    string                     `json:"actor_ref,omitempty"`
	PerfilRef   string                     `json:"perfil_ref,omitempty"`
	Estado      domain.EstadoCorreccion    `json:"estado"`
	Version     uint64                     `json:"version"`
	ReciboRef   string                     `json:"recibo_ref"`
	InstanteUTC time.Time                  `json:"instante_utc"`
}

type HistoriaAsientoCorreccion struct {
	Tipo               string           `json:"tipo"`
	MarcajeOriginalRef string           `json:"marcaje_original_ref"`
	Movimiento         domain.PunchKind `json:"movimiento"`
	InstanteUTC        *time.Time       `json:"instante_utc"`
	FuenteVersionRef   string           `json:"fuente_version_ref"`
}

type HistoriaCorreccion struct {
	SolicitudRef   string                        `json:"solicitud_ref"`
	EmpleadoRef    string                        `json:"empleado_ref"`
	Estado         domain.EstadoCorreccion       `json:"estado"`
	Version        uint64                        `json:"version"`
	FechaCivil     string                        `json:"fecha_civil"`
	HoraPretendida string                        `json:"hora_pretendida"`
	Movimiento     domain.PunchKind              `json:"movimiento"`
	MotivoCodigo   string                        `json:"motivo_codigo"`
	Actuaciones    []HistoriaActuacionCorreccion `json:"actuaciones"`
	Asientos       []HistoriaAsientoCorreccion   `json:"asientos"`
}

type CorreccionPendiente struct {
	SolicitudRef       string                  `json:"solicitud_ref"`
	EmpleadoRef        string                  `json:"empleado_ref"`
	EmpleadoEtiqueta   string                  `json:"empleado_etiqueta"`
	MarcajeOriginalRef string                  `json:"marcaje_original_ref"`
	HuecoDeclarado     bool                    `json:"hueco_declarado"`
	Movimiento         domain.PunchKind        `json:"movimiento"`
	FechaCivil         string                  `json:"fecha_civil"`
	HoraPretendida     string                  `json:"hora_pretendida"`
	MotivoCodigo       string                  `json:"motivo_codigo"`
	Estado             domain.EstadoCorreccion `json:"estado"`
	Version            uint64                  `json:"version"`
	SolicitadaEnUTC    time.Time               `json:"solicitada_en"`
}

type BandejaCorrecciones struct {
	Paso       string                `json:"paso"`
	Pendientes []CorreccionPendiente `json:"pendientes"`
}

type PeticionActuacionCorreccion struct {
	SolicitudRef, ClaveOperacion string
	Paso                         domain.PasoCorreccion
	Resultado                    domain.ResultadoCorreccion
	Motivo                       string
	VersionEsperada              uint64
}

type RepositorioDecisionCorrecciones interface {
	ConsultarBandejaCorrecciones(context.Context, OrdenConsumoCorreccion, domain.MaterialBandejaCorrecciones) (BandejaCorrecciones, error)
	RegistrarDecisionCorreccion(context.Context, OrdenConsumoCorreccion, domain.MaterialActuacionCorreccion) (ReciboCorreccion, error)
	ConsultarHistoriaCorreccion(context.Context, OrdenConsumoCorreccion, domain.MaterialHistoriaCorreccion) (HistoriaCorreccion, error)
}

type CasoUsoDecisionCorrecciones interface {
	ConsultarBandejaCorrecciones(context.Context, OrdenConsumoCorreccion, string) (BandejaCorrecciones, error)
	RegistrarDecisionCorreccion(context.Context, OrdenConsumoCorreccion, PeticionActuacionCorreccion) (ReciboCorreccion, error)
	ConsultarHistoriaCorreccion(context.Context, OrdenConsumoCorreccion, PeticionHistoriaCorreccion) (HistoriaCorreccion, error)
}
