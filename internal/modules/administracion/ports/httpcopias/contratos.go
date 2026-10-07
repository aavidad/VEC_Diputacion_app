// Package httpcopias defines the ADMIN backup boundary without transport or platform details.
package httpcopias

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrAutenticacion = errors.New("copias: autenticacion requerida")
	ErrDenegado      = errors.New("copias: acceso denegado")
	ErrNoDisponible  = errors.New("copias: dependencia no disponible")
	ErrConflicto     = errors.New("copias: conflicto de version")
	ErrNoEncontrado  = errors.New("copias: recurso no encontrado")
	ErrSolicitud     = errors.New("copias: solicitud invalida")
)

type Operacion string

const (
	Consultar            Operacion = "copias_consultar"
	Lanzar               Operacion = "copias_lanzar"
	ConfigurarCalendario Operacion = "copias_calendario_configurar"
	ConfigurarRetencion  Operacion = "copias_retencion_configurar"
	Proponer             Operacion = "copias_restauracion_proponer"
	Revisar              Operacion = "copias_restauracion_revisar"
	Ejecutar             Operacion = "copias_restauracion_sustituir"
)

// Sesion is produced by the existing ADMIN resolver. No identity is supplied by a client.
type Sesion struct {
	Actor          domain.ContextoActor
	Instantanea    domain.InstantaneaAutorizacion
	CorrelacionRef string
}

// Autorizador delegates to the current central authority, for one exact operation/resource.
// The backend must consume/revalidate that authority atomically with each durable effect.
type Autorizador interface {
	AutorizarCopias(context.Context, Sesion, Operacion, string) error
}

type Capacidades struct {
	Consultar            bool `json:"consultar"`
	Lanzar               bool `json:"lanzar"`
	ConfigurarCalendario bool `json:"configurar_calendario"`
	ConfigurarRetencion  bool `json:"configurar_retencion"`
	Proponer             bool `json:"proponer"`
	Revisar              bool `json:"revisar"`
	Ejecutar             bool `json:"ejecutar"`
}
type Compatibilidad struct {
	Estado    string `json:"estado"`
	ClaveI18N string `json:"clave_i18n"`
}
type Componente struct {
	Componente string `json:"componente"`
	Estado     string `json:"estado"`
}
type Copia struct {
	ReleaseRef     string         `json:"release_ref,omitempty"`
	CopiaRef       string         `json:"copia_ref"`
	Version        uint64         `json:"version"`
	Tipo           string         `json:"tipo"`
	Estado         string         `json:"estado"`
	IniciadaEn     time.Time      `json:"iniciada_en"`
	FinalizadaEn   *time.Time     `json:"finalizada_en,omitempty"`
	TamanoBytes    *uint64        `json:"tamano_bytes,omitempty"`
	Compatibilidad Compatibilidad `json:"compatibilidad"`
	HuellaSHA256   string         `json:"huella_sha256,omitempty"`
	Componentes    []Componente   `json:"componentes,omitempty"`
	RetencionHasta *time.Time     `json:"retencion_hasta,omitempty"`
}
type Pagina struct {
	Version         uint64  `json:"version"`
	Copias          []Copia `json:"copias"`
	CursorSiguiente string  `json:"cursor_siguiente,omitempty"`
}

// Politica is a minimized transport model; the policy service owns validation.
type Politica struct {
	Formato      uint64            `json:"formato"`
	Referencia   string            `json:"referencia"`
	Destino      string            `json:"destino"`
	ZonaHoraria  string            `json:"zona_horaria"`
	FechaInicial string            `json:"fecha_inicial"`
	CadaDias     int               `json:"cada_dias"`
	DiasSemana   []int             `json:"dias_semana,omitempty"`
	Ventana      VentanaPolitica   `json:"ventana"`
	Retencion    RetencionPolitica `json:"retencion"`
}
type VentanaPolitica struct {
	Inicio string `json:"inicio"`
	Fin    string `json:"fin"`
}
type RetencionPolitica struct {
	ConservarMinimo  int      `json:"conservar_minimo"`
	EdadMaximaDias   int      `json:"edad_maxima_dias"`
	Protegidas       []string `json:"protegidas,omitempty"`
	BorradoPermitido bool     `json:"borrado_permitido"`
	DobleControl     *bool    `json:"doble_control,omitempty"`
}
type Configuracion struct {
	Version  uint64   `json:"version"`
	Politica Politica `json:"politica"`
}
type Opcion struct {
	Ref       string `json:"ref"`
	ClaveI18N string `json:"clave_i18n"`
}
type OpcionesRestauracion struct {
	Destinos []Opcion `json:"destinos"`
	Motivos  []Opcion `json:"motivos"`
	Ventanas []Opcion `json:"ventanas"`
}

// FuenteOpciones is injected by configuration; it never accepts paths or grants from HTTP.
type FuenteOpciones interface {
	Opciones(context.Context, Sesion) (OpcionesRestauracion, error)
}
type Propuesta struct {
	MetadatosRevision       *MetadatosRevision `json:"metadatos_revision,omitempty"`
	ConjuntoHuellaSHA256    string             `json:"conjunto_huella_sha256"`
	PreimagenSHA256         string             `json:"preimagen_sha256"`
	PoliticaRef             string             `json:"politica_ref"`
	PoliticaHuellaSHA256    string             `json:"politica_huella_sha256"`
	MotivoRef               string             `json:"motivo_ref"`
	VentanaRef              string             `json:"ventana_ref"`
	DobleControl            bool               `json:"doble_control"`
	CopiaPreviaRequerida    bool               `json:"copia_previa_requerida"`
	PerdidaDesde            *time.Time         `json:"perdida_desde,omitempty"`
	PerdidaHasta            *time.Time         `json:"perdida_hasta,omitempty"`
	AlcancePerdidaClaveI18N string             `json:"alcance_perdida_clave_i18n,omitempty"`

	PropuestaRef  string    `json:"propuesta_ref"`
	ConjuntoRef   string    `json:"conjunto_ref"`
	DestinoRef    string    `json:"destino_ref"`
	Estado        string    `json:"estado"`
	Version       uint64    `json:"version"`
	HuellaSHA256  string    `json:"huella_sha256"`
	CaducaEn      time.Time `json:"caduca_en"`
	VentanaInicio time.Time `json:"ventana_inicio"`
	VentanaFin    time.Time `json:"ventana_fin"`
}
type Recibo struct {
	ReciboRef    string    `json:"recibo_ref"`
	OperacionRef string    `json:"operacion_ref"`
	RecursoRef   string    `json:"recurso_ref"`
	Version      uint64    `json:"version"`
	Estado       string    `json:"estado"`
	RegistradoEn time.Time `json:"registrado_en"`
}
type SolicitudLanzamiento struct {
	OperacionRef    string `json:"operacion_ref"`
	VersionEsperada uint64 `json:"version_esperada"`
	Tipo            string `json:"tipo"`
}
type SolicitudPolitica struct {
	OperacionRef    string   `json:"operacion_ref"`
	VersionEsperada uint64   `json:"version_esperada"`
	Politica        Politica `json:"politica"`
}
type SolicitudPropuesta struct {
	OperacionRef  string    `json:"operacion_ref"`
	ConjuntoRef   string    `json:"conjunto_ref"`
	DestinoRef    string    `json:"destino_ref"`
	MotivoRef     string    `json:"motivo_ref"`
	VentanaRef    string    `json:"ventana_ref"`
	VentanaInicio time.Time `json:"ventana_inicio"`
	VentanaFin    time.Time `json:"ventana_fin"`
	CaducaEn      time.Time `json:"caduca_en"`
}
type SolicitudControl struct {
	OperacionRef          string `json:"operacion_ref"`
	DestinoRef            string `json:"destino_ref"`
	PropuestaHuellaSHA256 string `json:"propuesta_huella_sha256"`
	VersionEsperada       uint64 `json:"version_esperada"`
}

// Consultas must authorize/audit every read and return only this minimized read model.
type Consultas interface {
	Listar(context.Context, Sesion, string, int) (Pagina, error)
	Detalle(context.Context, Sesion, string) (Copia, error)
	Calendario(context.Context, Sesion) (Configuracion, error)
	Retencion(context.Context, Sesion) (Configuracion, error)
	Propuestas(context.Context, Sesion) ([]Propuesta, error)
}

// Cambios preserves semantic idempotency, CAS, receipt and audit; no local success substitute.
type Cambios interface {
	Lanzar(context.Context, Sesion, SolicitudLanzamiento) (Recibo, error)
	ConfigurarCalendario(context.Context, Sesion, SolicitudPolitica) (Recibo, error)
	ConfigurarRetencion(context.Context, Sesion, SolicitudPolitica) (Recibo, error)
}
type Control interface {
	Proponer(context.Context, Sesion, SolicitudPropuesta) (Propuesta, error)
	Revisar(context.Context, Sesion, string, SolicitudControl) (Propuesta, error)
	Ejecutar(context.Context, Sesion, string, SolicitudControl) (Recibo, error)
}
