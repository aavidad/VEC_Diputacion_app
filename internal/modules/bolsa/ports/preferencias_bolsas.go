package ports

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionConsultarPreferenciasBolsas         = "bolsa.preferencias_bolsas.consultar_propias"
	AccionOrdenarPreferenciasBolsas           = "bolsa.preferencias_bolsas.ordenar_propias"
	AccionConsultarPreferenciaRRHH            = "bolsa.preferencias_bolsas.consultar_rrhh"
	AudienciaOrdenarPreferenciasBolsas        = "vec.bolsa.preferencias-bolsas.ordenar.v1"
	AudienciaOrdenarPreferenciasBolsasInterna = "vec.bolsa.preferencias-bolsas.ordenar.interna.v1"
	FinalidadPreferenciasBolsas               = "gestion_preferencias_bolsas_propias"
	FinalidadPreferenciaRRHH                  = "seleccion_simultanea_bolsas"
	CampoPreferenciasBolsas                   = "orden_preferencias_bolsas"
)

// La proyección B104 es cerrada; su orden coincide con la concesión nominal.
func CamposLecturaPreferenciasBolsas() []string {
	return []string{"version", "orden_guardado", "bolsas_actuales", "bolsas_nuevas", "bolsas_retiradas", "revision_requerida", "recibo_ref", "registrado_en"}
}

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

// CapturaLecturaPreferencias conserva una evaluación puntual y las dos
// representaciones canónicas. Sus bytes se copian al crear y al consumir.
type CapturaLecturaPreferencias struct {
	candidatoRef, correlacionRef string
	contexto, vinculo, orden     []byte
	validaHasta                  time.Time
}

func NuevaCapturaLecturaPreferencias(candidato, correlacion string, contexto, vinculo, orden []byte, validaHasta time.Time) CapturaLecturaPreferencias {
	return CapturaLecturaPreferencias{candidatoRef: candidato, correlacionRef: correlacion,
		contexto: slices.Clone(contexto), vinculo: slices.Clone(vinculo), orden: slices.Clone(orden), validaHasta: validaHasta}
}

func (c CapturaLecturaPreferencias) CandidatoRef() string       { return c.candidatoRef }
func (c CapturaLecturaPreferencias) CorrelacionRef() string     { return c.correlacionRef }
func (c CapturaLecturaPreferencias) ContextoCanonico() []byte   { return slices.Clone(c.contexto) }
func (c CapturaLecturaPreferencias) VinculoCanonico() []byte    { return slices.Clone(c.vinculo) }
func (c CapturaLecturaPreferencias) OrdenJSON() json.RawMessage { return slices.Clone(c.orden) }
func (c CapturaLecturaPreferencias) ValidaHasta() time.Time     { return c.validaHasta }

// LecturaPreferenciasBolsas solo recibe una captura del servicio de aplicación.
// El adaptador repite las guardas nominales en SQL, dentro de la lectura.
type LecturaPreferenciasBolsas interface {
	LeerPreferenciasBolsas(context.Context, CapturaLecturaPreferencias) (EstadoPreferenciasBolsas, error)
}

// El intento fallido conserva evidencia nominal sin transportar datos de Bolsa.
// La fachada SQL revalida el vínculo y el candidato antes de confirmar el asiento.
type AuditoriaIntentosPreferenciasBolsas interface {
	RegistrarIntentoLecturaPreferenciasBolsas(context.Context, CapturaLecturaPreferencias) error
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
