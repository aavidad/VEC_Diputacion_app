package ports

import (
	"context"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ModalidadPreferencia procede del catálogo gobernado común; su clave es la
// misma que conserva la oferta publicada, sin consultar el análisis actual.
type ModalidadPreferencia struct {
	Clave  string
	Fuente core.ReferenciaEntradaCatalogo
}

// EstadoPreferenciasModalidades tiene versión y recibo independientes del
// orden de bolsas. La versión cero expresa que todavía no hay una elección.
// Una selección nil muestra todas; una lista vacía no nula expresa ninguna.
type EstadoPreferenciasModalidades struct {
	Version                uint64
	ModalidadesGuardadas   []string
	ModalidadesDisponibles []ModalidadPreferencia
	ReciboRef              string
	RegistradoEn           *time.Time
}

// CatalogoPreferenciasModalidades aporta las opciones y sus referencias desde
// el catálogo común vigente. Nunca recibe claves ni fuentes del navegador.
type CatalogoPreferenciasModalidades interface {
	ConsultarModalidadesDisponibles(context.Context) ([]ModalidadPreferencia, error)
}

// GuardarPreferenciasModalidades recibe la selección de la persona y el
// catálogo capturado por el servidor. El material liga ambos a la decisión.
type GuardarPreferenciasModalidades struct {
	SolicitudPreferenciasBolsas
	Material          vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	Modalidades       []string
	Disponibles       []ModalidadPreferencia
	VersionEsperada   uint64
	ClaveIdempotencia string
}

type ReciboPreferenciasModalidades struct {
	ReciboRef    string
	Version      uint64
	Modalidades  []string
	RegistradoEn time.Time
	Reutilizado  bool
}

type EscrituraPreferenciasModalidades interface {
	GuardarPreferenciasModalidades(context.Context, GuardarPreferenciasModalidades) (ReciboPreferenciasModalidades, error)
}

// La misma captura nominal acredita una de las proyecciones fijas del
// descriptor de preferencias. El cliente HTTP no elige campos de lectura.
type LecturaPreferenciasModalidades interface {
	LeerPreferenciasModalidades(context.Context, CapturaLecturaPreferencias) (EstadoPreferenciasModalidades, error)
}
