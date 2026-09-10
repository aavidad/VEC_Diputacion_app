package incorporacionejercicio

import (
	"context"
	hist "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/historiaincorporacion"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/modules/personal/adapters/fuenteejercicio"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	pgpersonal "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	core "vec-diputacion-granada/internal/vec/domain"
)

// PlanPreparacionDurableV2 es configuración administrativa inmutable del
// servidor. RelacionRef selecciona una raíz propietaria: NO acredita un alta.
// No contiene autoridad ni datos generados durante una consulta/reintento.
type PlanPreparacionDurableV2 struct {
	// Sólo una definición ya publicada, fijada por el mismo documento sellado.
	// Sin RelacionRef: ésta procede exclusivamente del alta Personal real.
	PublicacionInicial         dom.PublicacionDefinicionSeguimiento `json:"publicacion_inicial,omitzero"`
	OrganizacionRef            string                               `json:"organizacion_ref"`
	UnidadRef                  string                               `json:"unidad_ref"`
	SolicitudPersonal          ct.SolicitudAltaPersonalRPT          `json:"solicitud_personal"`
	FuentePersonal             fuenteejercicio.TernaEsperada        `json:"fuente_personal"`
	SeguimientoRef             string                               `json:"seguimiento_ref"`
	RelacionRef                string                               `json:"relacion_ref"`
	Definicion                 dom.ReferenciaDefinicionSeguimiento  `json:"definicion"`
	VersionExpedienteRaiz      uint64                               `json:"version_expediente_raiz"`
	VersionSeguimientoEsperada uint64                               `json:"version_seguimiento_esperada"`
	Periodo                    dom.IntervaloSeguimiento             `json:"periodo"`
	MotivoClave                dom.ClaveCatalogo                    `json:"motivo_clave"`
	Documentos                 []dom.DocumentoSeguimiento           `json:"documentos"`
	MotivoV3                   core.ReferenciaEntradaCatalogo       `json:"motivo_v3"`
}

func (p PlanPreparacionDurableV2) Copia() PlanPreparacionDurableV2 {
	p.Documentos = append([]dom.DocumentoSeguimiento(nil), p.Documentos...)
	if d, err := dom.RestaurarDefinicionSeguimiento(p.PublicacionInicial); err == nil {
		p.PublicacionInicial = d.Publicacion()
	}
	return p
}

type FuentePlanesPreparacionV2 interface {
	ResolverPlan(context.Context, string, string) (PlanPreparacionDurableV2, error)
}

// ConsultaDetallePreparacionV2 se conecta al servicio RRHH ya autorizado;
// jamás a una proyección recibida del navegador ni a un lector técnico suelto.
type ConsultaDetallePreparacionV2 interface {
	Consultar(context.Context, ct.SolicitudDetalleRRHH) (ct.DetalleExpedienteRRHH, error)
}
type LectorInicialPreparacionV2 interface {
	LeerPreparacionInicial(context.Context, string, string, string) (dom.PublicacionDefinicionSeguimiento, dom.EstadoPersistidoSeguimiento, uint64, error)
}
type LocalizadorOriginalPreparacionV2 interface {
	Localizar(context.Context, string, string, string) (hist.Selector, bool, error)
}
type LocalizadorPersonalPreparacionV2 interface {
	Localizar(context.Context, string, string, string) (pgpersonal.SolicitudLocalizadaAltaEjercicio, bool, error)
}
type RestauradorOriginalPreparacionV2 interface {
	Restaurar(context.Context, hist.Selector) (hist.Restauracion, error)
}
type LectorPersonalPreparacionV2 interface {
	Leer(context.Context, lector.Selector, string, ct.ContextoAutorizacionAltaV3) (lector.Resultado, error)
}

// Dependencias por petición, sin HTTP ni efectos de alta/confirmación. La
// autoridad actual y consulta nominal son independientes de la historia CT77.
type ConfiguracionPreparacionDurableV2 struct {
	Autoridad           *AutoridadAplicacion
	Detalle             ConsultaDetallePreparacionV2
	Planes              FuentePlanesPreparacionV2
	FuentePersonal      []byte
	TernaPersonal       fuenteejercicio.TernaEsperada
	Inicial             LectorInicialPreparacionV2
	LocalizadorCT       LocalizadorOriginalPreparacionV2
	Restaurador         RestauradorOriginalPreparacionV2
	LocalizadorPersonal LocalizadorPersonalPreparacionV2
	LectorPersonal      LectorPersonalPreparacionV2
	Reloj               ct.Reloj
}
