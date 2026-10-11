package ports

import (
	"context"
	"errors"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionConsultarPreferenciasBolsas  = "bolsa.preferencias_bolsas.consultar_propias"
	AccionOrdenarPreferenciasBolsas    = "bolsa.preferencias_bolsas.ordenar_propias"
	AccionConsultarPreferenciaRRHH     = "bolsa.preferencias_bolsas.consultar_rrhh"
	AudienciaOrdenarPreferenciasBolsas = "vec.bolsa.preferencias-bolsas.ordenar.v1"
	FinalidadPreferenciasBolsas        = "gestion_preferencias_bolsas_propias"
	FinalidadPreferenciaRRHH           = "seleccion_simultanea_bolsas"
	CampoPreferenciasBolsas            = "orden_preferencias_bolsas"
)

var (
	ErrOrdenPreferenciasInvalido   = errors.New("bolsa: orden de preferencias invalido")
	ErrPreferenciasDesactualizadas = errors.New("bolsa: preferencias desactualizadas")
	ErrPreferenciasNoDisponibles   = errors.New("bolsa: preferencias no disponibles")
)

// EstadoPreferenciasBolsas conserva el orden escrito por la persona. Altas y
// bajas de participaciones no lo reescriben: exigen una nueva confirmacion.
type EstadoPreferenciasBolsas struct {
	Version           uint64
	OrdenGuardado     []string
	BolsasActuales    []string
	BolsasNuevas      []string
	BolsasRetiradas   []string
	RevisionRequerida bool
	ReciboRef         string
	RegistradoEn      *time.Time
}

type SolicitudPreferenciasBolsas struct {
	CandidatoRef string
	ConsultadaEn time.Time
}

type OrdenarPreferenciasBolsas struct {
	SolicitudPreferenciasBolsas
	Material          puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
	Orden             []string
	VersionEsperada   uint64
	ClaveIdempotencia string
}

type ReciboPreferenciasBolsas struct {
	ReciboRef    string
	Version      uint64
	Orden        []string
	RegistradoEn time.Time
	Reutilizado  bool
}

// La escritura consume V3 dentro de la transaccion SQL.
type EscrituraPreferenciasBolsas interface {
	OrdenarPreferenciasBolsas(context.Context, OrdenarPreferenciasBolsas) (ReciboPreferenciasBolsas, error)
}

// Esta lectura privada solo se invoca tras la autoridad nominal de sesion,
// permiso y auditoria. Por sí sola no se monta en HTTP.
type ConsultaPreferenciasBolsas interface {
	ConsultarPreferenciasBolsas(context.Context, SolicitudPreferenciasBolsas) (EstadoPreferenciasBolsas, error)
}

// El proveedor atesta una decisión nominal para el único efecto de ordenar.
type ProveedorMaterialPreferenciasBolsas interface {
	EmitirMaterialPreferenciasBolsas(context.Context, dominiovec.SolicitudAutorizacionLigadaV3,
		dominiovec.ResultadoContextoActorRegistradoV2, dominiovec.DecisionAutorizacionLigadaV3,
		puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3) (puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

// SeleccionSimultaneaRRHH contiene solo bolsas donde la persona ya es elegible
// segun el circuito de seleccion. La preferencia no altera ese resultado.
type SeleccionSimultaneaRRHH struct {
	CandidatoRef    string
	BolsasElegibles []string
	ConsultadaEn    time.Time
}

type PreferenciaSeleccionRRHH struct {
	Version           uint64
	BolsaPreferida    string
	RevisionRequerida bool
	BolsasElegibles   []string
	MotivoCodigo      string
}

type ConsultaPreferenciaSeleccionRRHH interface {
	ConsultarPreferenciaSeleccionRRHH(context.Context, SeleccionSimultaneaRRHH) (PreferenciaSeleccionRRHH, error)
}
