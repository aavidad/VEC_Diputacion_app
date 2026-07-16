# Modulo Bolsa

Parte del [Manual del programador](LEEME.md). Fichero generado con
`scripts/generar_manual_programador.py`; no editar a mano.

## Paquete `internal/modules/bolsa`

> Manifiesto del modulo Bolsa: identidad, permisos y menus para el shell VEC.

### Constantes

```go
const (
	ModuleID               = "vec.module.bolsa"
	PermissionRead         = "bolsa.solicitud.read"
	PermissionManage       = "bolsa.solicitud.manage"
	PermissionDocument     = "bolsa.document.read"
	PermissionClaim        = "bolsa.claim.read"
	PermissionNotification = "bolsa.notification.read"
	PermissionDemoAction   = "bolsa.demo.action"
	PermissionAudit        = "bolsa.audit.read"
	ActionDemoIntegration  = "bolsa.demo.integration"
)
```

### Funciones

```go
func Manifest() domain.ModuleManifest
```

### Tipos

```go
type AdminCapabilities struct {
	ModuleRef         string              `json:"module_ref"`
	Capabilities      []Capability        `json:"capabilities"`
	HTTPRoutes        []ModuleHTTPRoute   `json:"http_routes"`
	LegalIntegrations []IntegrationStatus `json:"legal_integrations"`
}

func AdminCapabilitiesContract() AdminCapabilities

type Capability struct {
	CapabilityRef string `json:"capability_ref"`
	Mode          string `json:"mode"`
	Method        string `json:"method,omitempty"`
	Route         string `json:"route,omitempty"`
	LabelI18nKey  string `json:"label_i18n_key"`
}

func AdminCapabilityList() []Capability

type IntegrationStatus struct {
	IntegrationRef string `json:"integration_ref"`
	Status         string `json:"status"`
	Mode           string `json:"mode"`
}

func LegalIntegrations() []IntegrationStatus

type MenuEntry struct {
	EntryRef            string   `json:"entry_ref"`
	LabelI18nKey        string   `json:"label_i18n_key"`
	Route               string   `json:"route"`
	RequiredPermissions []string `json:"required_permissions"`
}

type ModuleHTTPRoute struct {
	Method string `json:"method"`
	Route  string `json:"route"`
	Mode   string `json:"mode"`
}

func AdminRoutes() []ModuleHTTPRoute

type ModuleManifestContract struct {
	ModuleRef      string `json:"module_ref"`
	Version        string `json:"version"`
	TitleI18nKey   string `json:"title_i18n_key"`
	DescriptionKey string `json:"description_i18n_key"`
	CategoryRef    string `json:"category_ref"`
	BaseRoute      string `json:"base_route"`
	APIPrefix      string `json:"api_prefix"`
	PrototypeAPI   string `json:"prototype_api_prefix"`
	// AuthorizationPolicySource declara de donde debe obtenerse la decision,
	// pero nunca concede acceso. Los roles y concesiones son datos publicados y
	// versionados; no se incrustan en el manifiesto del modulo.
	AuthorizationPolicySource string            `json:"authorization_policy_source"`
	MenuEntries               []MenuEntry       `json:"menu_entries"`
	Capabilities              []Capability      `json:"capabilities"`
	EventsPublished           []string          `json:"events_published"`
	HealthRoute               string            `json:"health_route"`
	HTTPRoutes                []ModuleHTTPRoute `json:"http_routes"`
}

func ModuleManifestForCandidatePortal() ModuleManifestContract

type OperationalStatus struct {
	ModuleRef            string              `json:"module_ref"`
	RuntimeMode          string              `json:"runtime_mode"`
	Status               string              `json:"status"`
	AuthMode             string              `json:"auth_mode"`
	PersistenceMode      string              `json:"persistence_mode"`
	DemoEnabled          bool                `json:"demo_enabled"`
	LegalProductionReady bool                `json:"legal_production_ready"`
	AdminRoutes          []ModuleHTTPRoute   `json:"admin_routes"`
	LegalIntegrations    []IntegrationStatus `json:"legal_integrations"`
}

func OperationalStatusDefault(demoEnabled bool) OperationalStatus

func OperationalStatusForModes(demoEnabled bool, authMode, persistenceMode string) OperationalStatus
```

## Paquete `internal/modules/bolsa/adapters/fichero`

> Package fichero aporta únicamente una fuente local de demostración.

Package fichero aporta únicamente una fuente local de demostración. No es un
repositorio productivo ni persiste escrituras.

### Tipos

```go
type ConsultaConvocatorias struct {
	// Has unexported fields.
}
```

ConsultaConvocatorias mantiene una instantánea inmutable cargada al inicio.
Cambiar a PostgreSQL/Oracle requiere otro adaptador del mismo puerto,
no cambios en aplicación, HTTP o UI.

```go
func NuevaConsultaConvocatorias(ruta string) (*ConsultaConvocatorias, error)

func (c *ConsultaConvocatorias) BuscarPublicadas(ctx context.Context, filtro puertosbolsa.FiltroConvocatoriasPublicas) (puertosbolsa.PaginaConvocatorias, error)

func (c *ConsultaConvocatorias) ObtenerPublicada(ctx context.Context, identificador string) (puertosbolsa.DetalleConvocatoria, error)
```

## Paquete `internal/modules/bolsa/adapters/httppublico`

> Package httppublico expone únicamente proyecciones públicas minimizadas.

Package httppublico expone únicamente proyecciones públicas minimizadas.
No contiene rutas personales, internas ni de administración.

### Constantes

```go
const RutaConvocatorias = "/api/publico/bolsa/convocatorias"
```

### Funciones

```go
func NuevoHandler(servicio *aplicacionbolsa.ServicioConsultaPublica) (http.Handler, error)
```

### Tipos

```go
type Handler struct {
	// Has unexported fields.
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request)
```

## Paquete `internal/modules/bolsa/adapters/memory`

> Package memory contiene adaptadores efimeros y defensivos del modulo de bolsas.

Package memory contiene adaptadores efimeros y defensivos del modulo de bolsas.
Solo son apropiados para pruebas: no sustituyen una transaccion durable ni una
outbox persistente.

Package memory ofrece adaptadores efimeros y exclusivos para pruebas.

### Tipos

```go
type GeneradorReferenciasFlujoFirmaBaremacion struct{}

func (GeneradorReferenciasFlujoFirmaBaremacion) NuevaReferenciaEfectoFirmaBaremacion(
	paso puertosbolsa.PasoFlujoFirmaBaremacion,
) (string, error)

func (GeneradorReferenciasFlujoFirmaBaremacion) NuevaReferenciaFlujoFirmaBaremacion() (string, error)

func (GeneradorReferenciasFlujoFirmaBaremacion) NuevaReferenciaPropietarioArrendamientoFirmaBaremacion() (string, error)

type PerfilUsoRegistroPropuestasMemoria struct {
	// Has unexported fields.
}
```

PerfilUsoRegistroPropuestasMemoria es una capacidad deliberadamente opaca.
Evita cablear por accidente este adaptador efimero en una composicion real.

```go
type PerfilUsoRepositorioBaremacionesMemoria struct {
	// Has unexported fields.
}
```

PerfilUsoRepositorioBaremacionesMemoria es una capacidad deliberadamente
opaca. Solo el constructor de pruebas puede emitirla: evita seleccionar este
adaptador efimero por accidente en una composicion productiva.

```go
type ProtectorEstadoFlujoFirmaBaremacion struct {
	// Has unexported fields.
}
```

ProtectorEstadoFlujoFirmaBaremacion usa AES-256-GCM con AAD de esquema y
referencia de clave. La clave se inyecta para que dos instancias puedan
reanudar el mismo expediente; un despliegue productivo debe obtenerla de un
KMS/HSM y aplicar rotacion/versionado.

```go
func NuevoProtectorEstadoFlujoFirmaBaremacion(
	claveRef string,
	clave []byte,
) (*ProtectorEstadoFlujoFirmaBaremacion, error)

func (p *ProtectorEstadoFlujoFirmaBaremacion) DesprotegerEstadoFlujoFirmaBaremacion(
	ctx context.Context,
	estado puertosbolsa.EstadoProtegidoFlujoFirmaBaremacion,
) (puertosbolsa.CargaProtegida, error)

func (p *ProtectorEstadoFlujoFirmaBaremacion) ProtegerEstadoFlujoFirmaBaremacion(
	ctx context.Context,
	carga puertosbolsa.CargaProtegida,
) (puertosbolsa.EstadoProtegidoFlujoFirmaBaremacion, error)

type RegistroPropuestasLlamamiento struct {
	// Has unexported fields.
}
```

RegistroPropuestasLlamamiento simula una unica transaccion bajo un mutex. No
es un repositorio productivo: no aporta durabilidad, outbox, hora de base de
datos, bloqueo de la necesidad ni verificacion criptografica de atestacion.

```go
func NuevoRegistroPropuestasLlamamiento(
	reloj puertosbolsa.RelojLlamamientos,
	perfil PerfilUsoRegistroPropuestasMemoria,
) (*RegistroPropuestasLlamamiento, error)

func (r *RegistroPropuestasLlamamiento) GuardarPropuestaLlamamiento(
	ctx context.Context,
	propuesta dominiobolsa.PropuestaLlamamiento,
	evidencia puertosvec.EvidenciaUsoDecisionAutorizacion,
) error

func (r *RegistroPropuestasLlamamiento) NumeroPropuestasParaPruebas() int

func (r *RegistroPropuestasLlamamiento) ObtenerPropuestaParaPruebas(
	ctx context.Context,
	referencia string,
) (dominiobolsa.PropuestaLlamamiento, error)
```

ObtenerPropuestaParaPruebas no forma parte del puerto productivo. Devuelve
una copia profunda para comprobar aislamiento en tests de adaptadores.

```go
type RepositorioBaremaciones struct {
	// Has unexported fields.
}
```

RepositorioBaremaciones conserva versiones, tombstones de idempotencia,
auditoria y outbox bajo el mismo mutex. Una clave o token abandonados,
expirados o invalidados nunca vuelven a habilitarse.

```go
func NuevoRepositorioBaremaciones(
	reloj puertosbolsa.Reloj,
	verificador puertosbolsa.VerificadorSellosBaremacion,
	perfil PerfilUsoRepositorioBaremacionesMemoria,
) (*RepositorioBaremaciones, error)

func (r *RepositorioBaremaciones) AbandonarReserva(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudAbandonarReservaBaremacion,
) error

func (r *RepositorioBaremaciones) ConfirmarCambio(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
) (puertosbolsa.ResultadoConfirmarCambioBaremacion, error)

func (r *RepositorioBaremaciones) ObtenerEvidenciaTransaccion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudObtenerEvidenciaTransaccionBaremacion,
) (puertosbolsa.EvidenciaTransaccionBaremacionRecuperada, error)

func (r *RepositorioBaremaciones) ObtenerVersion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudObtenerVersionBaremacion,
) (puertosbolsa.VersionBaremacion, error)

func (r *RepositorioBaremaciones) ObtenerVersionVigente(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudObtenerBaremacionVigente,
) (puertosbolsa.VersionBaremacion, error)

func (r *RepositorioBaremaciones) ReservarCambio(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudReservarCambioBaremacion,
) (puertosbolsa.ReservaCambioBaremacion, error)

type RepositorioFlujosFirmaBaremacion struct {
	// Has unexported fields.
}
```

RepositorioFlujosFirmaBaremacion simula CAS, fencing, sellado e indices
unicos bajo un mutex. Es util para pruebas de reinicio de la capa de
aplicacion, pero no es almacenamiento durable productivo.

```go
func NuevoRepositorioFlujosFirmaBaremacion(
	reloj puertosbolsa.Reloj,
	verificador puertosbolsa.VerificadorEstadoFlujoFirmaBaremacion,
) (*RepositorioFlujosFirmaBaremacion, error)

func (r *RepositorioFlujosFirmaBaremacion) AdquirirArrendamientoFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudAdquirirArrendamientoFlujoFirmaBaremacion,
) (puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion, error)

func (r *RepositorioFlujosFirmaBaremacion) CrearORecuperarFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudCrearORecuperarFlujoFirmaBaremacion,
) (puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion, error)

func (r *RepositorioFlujosFirmaBaremacion) Format(estado fmt.State, _ rune)

func (r *RepositorioFlujosFirmaBaremacion) GoString() string

func (r *RepositorioFlujosFirmaBaremacion) GuardarFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudGuardarFlujoFirmaBaremacion,
) (puertosbolsa.ExpedienteFlujoFirmaBaremacion, error)

func (r *RepositorioFlujosFirmaBaremacion) LiberarArrendamientoFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudLiberarArrendamientoFlujoFirmaBaremacion,
) error

func (r *RepositorioFlujosFirmaBaremacion) LogValue() slog.Value

func (r *RepositorioFlujosFirmaBaremacion) ObtenerFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion,
) (puertosbolsa.ExpedienteFlujoFirmaBaremacion, error)

func (*RepositorioFlujosFirmaBaremacion) String() string
```

## Paquete `internal/modules/bolsa/adapters/postgres`

> Package postgres implementa la persistencia durable del agregado de baremacion.

Package postgres implementa la persistencia durable del agregado de baremacion.
La cuenta de ejecucion no recibe permisos sobre tablas: este adaptador solo
invoca funciones SECURITY DEFINER de contrato cerrado.

### Tipos

```go
type RepositorioBaremaciones struct {
	// Has unexported fields.
}
```

RepositorioBaremaciones mantiene las dependencias criptograficas fuera de
PostgreSQL. El ensamblado productivo debe aportar un verificador HMAC real,
con claves fuera del proceso y comparacion constante; una implementacion
que acepte siempre no constituye una frontera de seguridad. PostgreSQL
no confia en esa dependencia: vuelve a validar decision, sesion, rol,
catalogo y una atestacion COSE durable justo antes de aplicar cada efecto,
y falla cerrado si falta cualquiera de esas evidencias.

```go
func NuevoRepositorioBaremaciones(
	pool *pgxpool.Pool,
	reloj puertosbolsa.Reloj,
	verificador puertosbolsa.VerificadorSellosBaremacion,
) (*RepositorioBaremaciones, error)

func (r *RepositorioBaremaciones) AbandonarReserva(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudAbandonarReservaBaremacion,
) error

func (r *RepositorioBaremaciones) ConfirmarCambio(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
) (puertosbolsa.ResultadoConfirmarCambioBaremacion, error)

func (r *RepositorioBaremaciones) ObtenerEvidenciaTransaccion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudObtenerEvidenciaTransaccionBaremacion,
) (puertosbolsa.EvidenciaTransaccionBaremacionRecuperada, error)

func (r *RepositorioBaremaciones) ObtenerVersion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudObtenerVersionBaremacion,
) (puertosbolsa.VersionBaremacion, error)

func (r *RepositorioBaremaciones) ObtenerVersionVigente(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudObtenerBaremacionVigente,
) (puertosbolsa.VersionBaremacion, error)

func (r *RepositorioBaremaciones) ReservarCambio(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudReservarCambioBaremacion,
) (puertosbolsa.ReservaCambioBaremacion, error)
```

## Paquete `internal/modules/bolsa/application`

> Package application contiene casos de uso del modulo de bolsa.

Package application contiene casos de uso del modulo de bolsa.

### Constantes

```go
const (
	TamanoPaginaPublicaPredeterminado = 12
	TamanoPaginaPublicaMaximo         = 24
	PaginaPublicaMaxima               = 500
	LongitudTextoPublicoMaxima        = 100
)
```

### Variables

```go
var (
	ErrDependenciaBaremacionRequerida = errors.New("bolsa: dependencia de baremacion requerida")
	ErrOrdenBaremacionInvalida        = errors.New("bolsa: orden de baremacion invalida")
	ErrResultadoBaremacionNoConfiable = errors.New("bolsa: resultado de baremacion no confiable")
	ErrFirmaBaremacionNoCompletada    = errors.New("bolsa: firma de baremacion no completada")
)
var (
	ErrServicioConsultaPublicaInvalido = errors.New("bolsa: servicio de consulta publica invalido")
	ErrFiltroPublicoInvalido           = errors.New("bolsa: filtro publico invalido")
	ErrDatosPublicosNoConfiables       = errors.New("bolsa: datos publicos no confiables")
)
var ErrAbandonoReservaBaremacionNoAcreditado = errors.New(
	"bolsa: abandono de reserva de baremacion no acreditado",
)
```

### Tipos

```go
type ActorBaremacion struct {
	Motivo string
}
```

ActorBaremacion contiene exclusivamente la motivacion administrativa.
La identidad, el perfil y la garantia se resuelven siempre desde la
sesion autoritativa; repetirlos en una orden ampliaria innecesariamente la
superficie de datos personales y nunca podria conceder acceso.

```go
type AyudaPublica struct {
	Referencia string               `json:"referencia"`
	Categoria  ValorCatalogoPublico `json:"categoria"`
	Orden      int                  `json:"orden"`
	Pregunta   string               `json:"pregunta"`
	Respuesta  string               `json:"respuesta"`
}

type DecisionBaremacionCodificada struct {
	// Has unexported fields.
}
```

DecisionBaremacionCodificada enlaza contenido, politica y bytes canonicos
sin exponer estos ultimos en formatos de registro.

```go
func (d DecisionBaremacionCodificada) Format(estado fmt.State, _ rune)

func (d DecisionBaremacionCodificada) GoString() string

func (d DecisionBaremacionCodificada) LogValue() slog.Value

func (DecisionBaremacionCodificada) MarshalBinary() ([]byte, error)

func (DecisionBaremacionCodificada) MarshalJSON() ([]byte, error)

func (DecisionBaremacionCodificada) MarshalText() ([]byte, error)

func (DecisionBaremacionCodificada) String() string

func (*DecisionBaremacionCodificada) UnmarshalJSON([]byte) error

type DecisionBaremacionCustodiada struct {
	// Has unexported fields.
}
```

DecisionBaremacionCustodiada conserva el recibo tecnico exacto del almacen
cifrado que recibio el documento firmable.

```go
func (d DecisionBaremacionCustodiada) Format(estado fmt.State, _ rune)

func (d DecisionBaremacionCustodiada) GoString() string

func (d DecisionBaremacionCustodiada) LogValue() slog.Value

func (DecisionBaremacionCustodiada) MarshalBinary() ([]byte, error)

func (DecisionBaremacionCustodiada) MarshalJSON() ([]byte, error)

func (DecisionBaremacionCustodiada) MarshalText() ([]byte, error)

func (DecisionBaremacionCustodiada) String() string

func (*DecisionBaremacionCustodiada) UnmarshalJSON([]byte) error

type DetalleConvocatoriaPublica struct {
	Esquema      string                     `json:"esquema"`
	Fuente       FuentePublica              `json:"fuente"`
	Convocatoria ResumenConvocatoriaPublica `json:"convocatoria"`
	Descripcion  string                     `json:"descripcion"`
	Plazos       []PlazoPublico             `json:"plazos"`
	Requisitos   []RequisitoPublico         `json:"requisitos"`
	Documentos   []DocumentoPublico         `json:"documentos"`
	Ayuda        []AyudaPublica             `json:"ayuda"`
}

type DocumentoPublico struct {
	Referencia  string               `json:"referencia"`
	Tipo        ValorCatalogoPublico `json:"tipo"`
	Orden       int                  `json:"orden"`
	Titulo      string               `json:"titulo"`
	Descripcion string               `json:"descripcion"`
	Formato     string               `json:"formato"`
	URL         string               `json:"url"`
	PublicadoEn time.Time            `json:"publicado_en"`
}

type ErrorCustodiaBaremacionIncompleta struct {
	DecisionRef  string
	DocumentoRef string
	Escritura    puertosvec.ResultadoOperacionObjeto
	Retencion    *puertosvec.ResultadoOperacionObjeto
	Causa        error
}
```

ErrorCustodiaBaremacionIncompleta conserva el recibo de cualquier objeto que
el almacen haya creado antes de fallar su validacion, retencion o enlace
con el expediente. Nunca se descarta esa referencia: un reconciliador debe
completar la retencion o inmovilizar el objeto con intervencion registrada.

```go
func (*ErrorCustodiaBaremacionIncompleta) Error() string

func (e *ErrorCustodiaBaremacionIncompleta) Format(estado fmt.State, _ rune)

func (e *ErrorCustodiaBaremacionIncompleta) GoString() string

func (e *ErrorCustodiaBaremacionIncompleta) LogValue() slog.Value

func (e *ErrorCustodiaBaremacionIncompleta) MarshalBinary() ([]byte, error)

func (e *ErrorCustodiaBaremacionIncompleta) MarshalJSON() ([]byte, error)

func (e *ErrorCustodiaBaremacionIncompleta) MarshalText() ([]byte, error)

func (e *ErrorCustodiaBaremacionIncompleta) String() string

func (e *ErrorCustodiaBaremacionIncompleta) Unwrap() error

type ErrorDocumentoFirmadoHuerfano struct {
	DecisionRef string
	Documento   puertosbolsa.DocumentoFirmadoCustodiado
	Causa       error
}
```

ErrorDocumentoFirmadoHuerfano conserva la referencia institucional del
documento ya firmado y retenido cuando OCC impide convertirlo en decision
eficaz. El reconciliador puede inventariarlo sin volver a firmar ni
borrarlo.

Sus campos son deliberadamente accesibles al reconciliador, pero todas las
representaciones genericas se reducen a un mensaje fijo. Asi fmt, slog y los
codificadores habituales no convierten por accidente recibos o referencias
internas en datos de una respuesta HTTP o de un registro operacional.

```go
func (*ErrorDocumentoFirmadoHuerfano) Error() string

func (e *ErrorDocumentoFirmadoHuerfano) Format(estado fmt.State, _ rune)

func (e *ErrorDocumentoFirmadoHuerfano) GoString() string

func (e *ErrorDocumentoFirmadoHuerfano) LogValue() slog.Value

func (e *ErrorDocumentoFirmadoHuerfano) MarshalBinary() ([]byte, error)

func (e *ErrorDocumentoFirmadoHuerfano) MarshalJSON() ([]byte, error)

func (e *ErrorDocumentoFirmadoHuerfano) MarshalText() ([]byte, error)

func (e *ErrorDocumentoFirmadoHuerfano) String() string

func (e *ErrorDocumentoFirmadoHuerfano) Unwrap() error

type FacetasConvocatorias struct {
	Tipos      []ValorCatalogoPublico `json:"tipos"`
	Categorias []ValorCatalogoPublico `json:"categorias"`
	Estados    []ValorCatalogoPublico `json:"estados"`
}

type FachadaFirmaBaremacionDurable struct {
	// Has unexported fields.
}
```

FachadaFirmaBaremacionDurable separa el ciclo HTTP de las capacidades en
memoria del motor. El repositorio conserva una saga sellada y el protector
cifra el estado de trabajo; cada llamada vuelve a derivar la identidad desde
la sesion autoritativa.

Estado: infraestructura interna sin cableado productivo. No acredita
persistencia tras reinicio, KMS/HSM ni ejecutores reales; esas garantias
dependen de los adaptadores fijados por la composicion homologada.

```go
func NuevaFachadaFirmaBaremacionDurable(
	repositorio puertosbolsa.RepositorioFlujosFirmaBaremacion,
	protector puertosbolsa.ProtectorEstadoFlujoFirmaBaremacion,
	ejecutor puertosbolsa.EjecutorPasosFirmaBaremacion,
	generador puertosbolsa.GeneradorReferenciasFlujoFirmaBaremacion,
	sellador puertosbolsa.SelladorSolicitudBaremacion,
	verificador puertosbolsa.VerificadorEstadoFlujoFirmaBaremacion,
	sesiones FuenteSesionAutenticadaBaremacion,
	reloj puertosbolsa.Reloj,
	opciones OpcionesFachadaFirmaBaremacionDurable,
) (*FachadaFirmaBaremacionDurable, error)

func (f *FachadaFirmaBaremacionDurable) Consultar(
	ctx context.Context,
	orden OrdenReanudarFlujoFirmaBaremacion,
) (ProyeccionEstadoFlujoFirmaBaremacion, error)

func (f *FachadaFirmaBaremacionDurable) Finalizar(
	ctx context.Context,
	orden OrdenReanudarFlujoFirmaBaremacion,
) (puertosbolsa.ResultadoFinalFlujoFirmaBaremacion, error)

func (f *FachadaFirmaBaremacionDurable) Preparar(
	ctx context.Context,
	orden OrdenPrepararFlujoFirmaBaremacion,
) (puertosbolsa.ProyeccionLanzamientoFirmaBaremacion, error)

type FirmaBaremacionPreparada struct {
	// Has unexported fields.
}
```

FirmaBaremacionPreparada contiene la sesion opaca del firmador y todos los
enlaces probatorios previos necesarios para comprobar su resultado.

```go
func (f FirmaBaremacionPreparada) Format(estado fmt.State, _ rune)

func (f FirmaBaremacionPreparada) GoString() string

func (f FirmaBaremacionPreparada) LogValue() slog.Value

func (FirmaBaremacionPreparada) MarshalBinary() ([]byte, error)

func (FirmaBaremacionPreparada) MarshalJSON() ([]byte, error)

func (FirmaBaremacionPreparada) MarshalText() ([]byte, error)

func (FirmaBaremacionPreparada) String() string

func (*FirmaBaremacionPreparada) UnmarshalJSON([]byte) error

type FuentePublica struct {
	Revision      string    `json:"revision"`
	ActualizadaEn time.Time `json:"actualizada_en"`
	Demostracion  bool      `json:"demostracion"`
	Aviso         string    `json:"aviso,omitempty"`
}

type FuenteSesionAutenticadaBaremacion interface {
	BuscarSesionesAutenticadasBaremacion(context.Context) ([]SesionAutenticadaBaremacion, error)
}
```

FuenteSesionAutenticadaBaremacion debe resolver la sesion a partir del
contexto confiable del adaptador de entrada, no de cabeceras aportadas por
el cliente. Devuelve todas las coincidencias para que la aplicacion pueda
denegar tanto la ausencia como la ambiguedad sin elegir la primera.

```go
type ListadoConvocatoriasPublicas struct {
	Esquema       string                       `json:"esquema"`
	Fuente        FuentePublica                `json:"fuente"`
	Facetas       FacetasConvocatorias         `json:"facetas"`
	Paginacion    PaginacionPublica            `json:"paginacion"`
	Convocatorias []ResumenConvocatoriaPublica `json:"convocatorias"`
}

type OpcionesFachadaFirmaBaremacionDurable struct {
	DuracionArrendamiento time.Duration
}

type OpcionesServicioBaremacion struct {
	DuracionReserva          time.Duration
	DuracionFirma            time.Duration
	ClasificacionDocumental  string
	ConectorAlmacenPermitido string
	PoliticaRetencionRef     string
	DuracionRetencion        time.Duration
}
```

OpcionesServicioBaremacion fija exclusivamente limites tecnicos y el
conector de almacenamiento admitido por el despliegue.

```go
type OrdenAdoptarDecisionBaremacion struct {
	Actor                  ActorBaremacion
	Revision               RevisionBaremacionIniciada
	Clase                  dominiobolsa.ClaseDecisionTecnica
	CalculoRef             string
	HuellaResultadoCalculo string
	PuntosReconocidos      dominiobolsa.Puntos
	Resultado              dominiobolsa.ResultadoDecisionTecnica
	ValoracionesEvidencia  []dominiobolsa.ValoracionEvidencia
	MotivoClave            string
	MotivoDecision         string
	FuentesNormativasRefs  []string
}
```

OrdenAdoptarDecisionBaremacion expresa el juicio tecnico; actor, numero,
calculo oficial y referencias administrativas se obtienen o verifican en el
servidor.

```go
type OrdenCodificarDecisionBaremacion struct {
	Actor                ActorBaremacion
	Revision             RevisionBaremacionAdoptada
	PoliticaFirmaRef     string
	PoliticaFirmaVersion int
	HuellaPoliticaSHA256 string
}
```

OrdenCodificarDecisionBaremacion selecciona una politica publicada por
referencia, version y huella exactas.

```go
type OrdenCustodiarDecisionBaremacion struct {
	Actor             ActorBaremacion
	Decision          DecisionBaremacionCodificada
	OperacionRef      string
	ClaveIdempotencia string
	CargaRef          string
}
```

OrdenCustodiarDecisionBaremacion aporta referencias operativas opacas;
la huella de solicitud y el seudonimo se generan siempre en servidor.

```go
type OrdenFinalizarFirmaBaremacion struct {
	Actor                     ActorBaremacion
	Firma                     FirmaBaremacionPreparada
	OperacionRef              string
	ClaveIdempotenciaSello    string
	ClaveIdempotenciaAumento  string
	MotivoClaveConfirmacion   string
	MotivoConfirmacion        string
	OperacionCustodiaRef      string
	ClaveIdempotenciaCustodia string
	CargaDocumentoFirmadoRef  string
	ClaveIdempotenciaReserva  string
}
```

OrdenFinalizarFirmaBaremacion aporta solo claves de reintento y motivacion;
no acepta artefactos, validaciones ni atestaciones declaradas por el
cliente.

```go
type OrdenIniciarRevisionBaremacion struct {
	Actor               ActorBaremacion
	BaremacionMeritoRef string
	SujetoRef           string
	Finalidad           string
	CorrelacionRef      string
}
```

OrdenIniciarRevisionBaremacion identifica el merito y sujeto cuya version se
fija como base inmutable para la revision; no solicita una reserva.

```go
type OrdenPrepararFirmaBaremacion struct {
	Actor             ActorBaremacion
	Decision          DecisionBaremacionCustodiada
	OperacionRef      string
	ClaveIdempotencia string
}
```

OrdenPrepararFirmaBaremacion solicita una sesion interactiva acotada por
la politica de firma vigente; la reserva OCC se adquiere tras custodiar la
firma.

```go
type OrdenPrepararFlujoFirmaBaremacion struct {
	ClaveIdempotencia    string
	ProcesoRef           string
	SolicitudRef         string
	BaremacionMeritoRef  string
	DecisionRef          string
	EstadoTrabajoInicial puertosbolsa.CargaProtegida
}
```

OrdenPrepararFlujoFirmaBaremacion solo admite referencias de negocio y un
estado de trabajo interno sin capacidades. Un adaptador HTTP no debe aceptar
EstadoTrabajoInicial del cliente: lo construye el caso de uso tecnico que ha
fijado la decision y sus huellas. La entrada productiva permanece cerrada
hasta imponer esta restriccion por construccion y probarla en arquitectura.

```go
type OrdenReanudarFlujoFirmaBaremacion struct {
	FlujoRef          string
	ClaveIdempotencia string
}

type PaginacionPublica struct {
	Pagina  int `json:"pagina"`
	Tamano  int `json:"tamano"`
	Total   int `json:"total"`
	Paginas int `json:"paginas"`
}

type PlazoPublico struct {
	Referencia  string               `json:"referencia"`
	Tipo        ValorCatalogoPublico `json:"tipo"`
	Titulo      string               `json:"titulo"`
	Descripcion string               `json:"descripcion,omitempty"`
	AbreEn      time.Time            `json:"abre_en"`
	CierraEn    time.Time            `json:"cierra_en"`
	Situacion   string               `json:"situacion"`
	Etiqueta    string               `json:"etiqueta_situacion"`
	Semantica   string               `json:"semantica_situacion"`
}

type ProyeccionEstadoFlujoFirmaBaremacion struct {
	FlujoRef      string
	Estado        puertosbolsa.EstadoExpedienteFlujoFirmaBaremacion
	Lanzamiento   *puertosbolsa.ProyeccionLanzamientoFirmaBaremacion
	Resultado     *puertosbolsa.ResultadoFinalFlujoFirmaBaremacion
	ActualizadoEn time.Time
}

type RelojConsultaPublica interface {
	Ahora() time.Time
}

type RelojSistemaConsultaPublica struct{}

func (RelojSistemaConsultaPublica) Ahora() time.Time

type RequisitoPublico struct {
	Referencia  string `json:"referencia"`
	Orden       int    `json:"orden"`
	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`
	Obligatorio bool   `json:"obligatorio"`
}

type ResultadoFinalizarFirmaBaremacion struct {
	Decision            dominiobolsa.DecisionTecnica
	ValidacionInicial   puertosbolsa.ValidacionFirmaServidor
	SelloTiempo         *puertosbolsa.SelloTiempoFirma
	ValidacionTrasSello *puertosbolsa.ValidacionFirmaServidor
	Aumento             *puertosbolsa.ResultadoAumentoFirma
	ValidacionFinal     puertosbolsa.ValidacionFirmaServidor
	DocumentoFirmado    puertosbolsa.DocumentoFirmadoCustodiado
	Confirmacion        puertosbolsa.ResultadoConfirmarCambioBaremacion
}
```

ResultadoFinalizarFirmaBaremacion expone la decision inmutable y todas las
capas verificadas que condujeron a su confirmacion transaccional.

```go
type ResumenConvocatoriaPublica struct {
	IdentificadorPublico string                 `json:"identificador_publico"`
	Version              string                 `json:"version"`
	HuellaSHA256         string                 `json:"huella_sha256"`
	Titulo               string                 `json:"titulo"`
	Resumen              string                 `json:"resumen"`
	Tipo                 ValorCatalogoPublico   `json:"tipo"`
	Estado               ValorCatalogoPublico   `json:"estado"`
	Categorias           []ValorCatalogoPublico `json:"categorias"`
	PlazoDestacado       *PlazoPublico          `json:"plazo_destacado,omitempty"`
	NumeroRequisitos     int                    `json:"numero_requisitos"`
	NumeroDocumentos     int                    `json:"numero_documentos"`
	NumeroAyudas         int                    `json:"numero_ayudas"`
	PublicadaEn          time.Time              `json:"publicada_en"`
	ActualizadaEn        time.Time              `json:"actualizada_en"`
}

type RevisionBaremacionAdoptada struct {
	// Has unexported fields.
}
```

RevisionBaremacionAdoptada contiene el documento administrativo aun no
firmado y la capacidad opaca exacta que autorizo su adopcion.

```go
func (r RevisionBaremacionAdoptada) Format(estado fmt.State, _ rune)

func (r RevisionBaremacionAdoptada) GoString() string

func (r RevisionBaremacionAdoptada) LogValue() slog.Value

func (RevisionBaremacionAdoptada) MarshalBinary() ([]byte, error)

func (RevisionBaremacionAdoptada) MarshalJSON() ([]byte, error)

func (RevisionBaremacionAdoptada) MarshalText() ([]byte, error)

func (RevisionBaremacionAdoptada) String() string

func (*RevisionBaremacionAdoptada) UnmarshalJSON([]byte) error

type RevisionBaremacionIniciada struct {
	// Has unexported fields.
}
```

RevisionBaremacionIniciada conserva la version exacta y la autorizacion de
consulta que la obtuvo. Su contenido no es un DTO transportable.

```go
func (r RevisionBaremacionIniciada) Format(estado fmt.State, _ rune)

func (r RevisionBaremacionIniciada) GoString() string

func (r RevisionBaremacionIniciada) LogValue() slog.Value

func (RevisionBaremacionIniciada) MarshalBinary() ([]byte, error)

func (RevisionBaremacionIniciada) MarshalJSON() ([]byte, error)

func (RevisionBaremacionIniciada) MarshalText() ([]byte, error)

func (RevisionBaremacionIniciada) String() string

func (*RevisionBaremacionIniciada) UnmarshalJSON([]byte) error

type ServicioBaremacion struct {
	// Has unexported fields.
}
```

ServicioBaremacion coordina el corte probatorio de una revision tecnica.
Cada paso obtiene una decision de autorizacion nueva, exacta y no
serializable; ningun contexto de autorizacion llega en las ordenes.

```go
func NuevoServicioBaremacion(
	repositorio puertosbolsa.RepositorioBaremaciones,
	fuenteDatos puertosbolsa.FuenteDatosBaremacion,
	calculador puertosbolsa.CalculadorOficialBaremacion,
	catalogoFirma puertosbolsa.CatalogoPoliticasFirmaBaremacion,
	codificador puertosbolsa.CodificadorCanonicoDecision,
	almacen puertosbolsa.AlmacenDocumentosFirmables,
	firmador puertosbolsa.FirmadorInteractivo,
	recuperadorBinario puertosbolsa.RecuperadorBinarioFirmado,
	validadorFirma puertosbolsa.ValidadorFirmaServidor,
	selladorTiempo puertosbolsa.SelladorTiempoFirma,
	aumentadorFirma puertosbolsa.AumentadorFirmaLongeva,
	selladorSolicitud puertosbolsa.SelladorServicioBaremacion,
	seudonimizador puertosvec.SeudonimizadorSujetoAlmacen,
	generador puertosbolsa.GeneradorReferenciasOpacasBaremacion,
	autorizador puertosvec.Autorizador,
	sesiones FuenteSesionAutenticadaBaremacion,
	reloj puertosbolsa.Reloj,
	opciones OpcionesServicioBaremacion,
) (*ServicioBaremacion, error)
```

NuevoServicioBaremacion exige todos los conectores de seguridad al arrancar;
una composicion parcial no crea un servicio degradado.

```go
func (s *ServicioBaremacion) AdoptarDecision(
	ctx context.Context,
	orden OrdenAdoptarDecisionBaremacion,
) (RevisionBaremacionAdoptada, error)
```

AdoptarDecision recupera el calculo oficial, revalida todas sus fuentes y
construye el contenido mediante las invariantes del agregado.

```go
func (s *ServicioBaremacion) CodificarDecision(
	ctx context.Context,
	orden OrdenCodificarDecisionBaremacion,
) (DecisionBaremacionCodificada, error)
```

CodificarDecision obtiene una politica vigente y delega la representacion
canonica en el conector especializado.

```go
func (s *ServicioBaremacion) CustodiarDecision(
	ctx context.Context,
	orden OrdenCustodiarDecisionBaremacion,
) (DecisionBaremacionCustodiada, error)
```

CustodiarDecision conserva el PDF canónico todavía no firmado con cifrado,
integridad, versionado y referencias opacas. También comprueba que el mismo
conector dispone de retención y bloqueo legal para la copia firmada final;
no afirma que este artefacto temporal esté ya retenido.

```go
func (s *ServicioBaremacion) FinalizarFirma(
	ctx context.Context,
	orden OrdenFinalizarFirmaBaremacion,
) (resultadoRetorno ResultadoFinalizarFirmaBaremacion, errRetorno error)
```

FinalizarFirma consulta el firmador, valida en servidor, aplica las capas
exigidas por politica y confirma agregado, auditoria y outbox mediante el
repositorio transaccional.

BLOQUEANTE PRODUCTIVO: esta primera vertical exige que adopcion y firma
pertenezcan al mismo actor. El doble control maker-checker necesita un
flujo persistente separado, con asignacion, caducidad, sustitucion y firma
propias; no debe simularse relajando esta vinculacion ni reutilizando
autorizaciones.

```go
func (s *ServicioBaremacion) IniciarRevision(
	ctx context.Context,
	orden OrdenIniciarRevisionBaremacion,
) (RevisionBaremacionIniciada, error)
```

IniciarRevision consulta y fija una instantanea defensiva de la version
vigente. No adquiere ninguna reserva durante la revision humana.

```go
func (s *ServicioBaremacion) PrepararFirma(
	ctx context.Context,
	orden OrdenPrepararFirmaBaremacion,
) (FirmaBaremacionPreparada, error)
```

PrepararFirma crea una sesion vinculada al documento, firmante, perfil,
politica y autorizacion exactos.

```go
type ServicioConsultaPublica struct {
	// Has unexported fields.
}

func NuevoServicioConsultaPublica(fuente puertosbolsa.ConsultaConvocatoriasPublicas, reloj RelojConsultaPublica) (*ServicioConsultaPublica, error)

func (s *ServicioConsultaPublica) Listar(ctx context.Context, solicitud SolicitudListadoPublico) (ListadoConvocatoriasPublicas, error)

func (s *ServicioConsultaPublica) Obtener(ctx context.Context, identificador string) (DetalleConvocatoriaPublica, error)

type ServicioLlamamientos struct {
	// Has unexported fields.
}
```

ServicioLlamamientos tiene todas sus dependencias obligatorias y privadas.
No existe constructor reducido: omitir PDP, fuentes, motor, reloj,
generadores o transaccion deja el servicio deliberadamente inutilizable.

```go
func NuevoServicioLlamamientos(
	resolutor puertosbolsa.ResolutorRecursoNecesidad,
	vinculador puertosbolsa.CreadorVinculoAutenticacionActor,
	autorizador puertosvec.Autorizador,
	fuente puertosbolsa.FuenteDatosLlamamiento,
	motor puertosbolsa.MotorElegibilidadLlamamiento,
	reloj puertosbolsa.RelojLlamamientos,
	generador puertosbolsa.GeneradorReferenciasLlamamiento,
	transaccion puertosbolsa.TransaccionPropuestasLlamamiento,
) (*ServicioLlamamientos, error)

func (s *ServicioLlamamientos) ProponerPrimerLlamamiento(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudProponerLlamamiento,
) (dominiobolsa.PropuestaLlamamiento, error)
```

ProponerPrimerLlamamiento ejecuta el recorrido autoritativo del orden. Solo
persiste si existe una primera persona elegible y la concesion sigue vigente
en el instante final. Cualquier dato desconocido, ambiguedad o capacidad
parcial falla cerrado antes de la transaccion de negocio.

```go
type SesionAutenticadaBaremacion struct {
	// Has unexported fields.
}
```

SesionAutenticadaBaremacion es una capacidad opaca emitida por la frontera
autoritativa de identidad. Liga la persona y el perfil resueltos con la
sesion revalidada mediante VinculoAutenticacionActorV1; no contiene hechos
de autenticacion copiables desde una orden o un DTO.

```go
func NuevaSesionAutenticadaBaremacion(
	contextoActor dominiovec.ContextoActor,
	vinculo dominiovec.VinculoAutenticacionActorV1,
) (SesionAutenticadaBaremacion, error)

func (s SesionAutenticadaBaremacion) Format(estado fmt.State, _ rune)

func (s SesionAutenticadaBaremacion) GoString() string

func (s SesionAutenticadaBaremacion) LogValue() slog.Value

func (SesionAutenticadaBaremacion) MarshalBinary() ([]byte, error)

func (SesionAutenticadaBaremacion) MarshalJSON() ([]byte, error)

func (SesionAutenticadaBaremacion) MarshalText() ([]byte, error)

func (SesionAutenticadaBaremacion) String() string

func (*SesionAutenticadaBaremacion) UnmarshalJSON([]byte) error

type SolicitudListadoPublico struct {
	Texto            string
	Tipo             string
	Categoria        string
	Estado           string
	SoloPlazoAbierto bool
	Pagina           int
	Tamano           int
}

type ValorCatalogoPublico struct {
	Clave       string `json:"clave"`
	Version     int    `json:"version"`
	Etiqueta    string `json:"etiqueta"`
	Descripcion string `json:"descripcion,omitempty"`
	Semantica   string `json:"semantica"`
}
```

## Paquete `internal/modules/bolsa/domain`

> Package domain contiene las reglas puras del modulo de bolsas.

Package domain contiene las reglas puras del modulo de bolsas.

### Constantes

```go
const (
	// UnidadesPorPunto fija seis decimales sin recurrir nunca a coma flotante.
	// Es una invariante tecnica, no un tipo de merito ni una regla de baremo.
	UnidadesPorPunto Puntos = 1_000_000
)
```

### Variables

```go
var (
	ErrBaremacionInvalida          = errors.New("bolsa: baremacion de merito invalida")
	ErrCriterioInvalido            = errors.New("bolsa: referencia de criterio invalida")
	ErrReglaCalculoInvalida        = errors.New("bolsa: referencia de regla de calculo invalida")
	ErrCalculoOficialInvalido      = errors.New("bolsa: calculo oficial invalido")
	ErrEvidenciaInvalida           = errors.New("bolsa: referencia de evidencia invalida")
	ErrValoracionEvidenciaInvalida = errors.New("bolsa: valoracion de evidencia invalida")
	ErrContenidoDecisionInvalido   = errors.New("bolsa: contenido de decision tecnica invalido")
	ErrFirmaDecisionInvalida       = errors.New("bolsa: evidencia de firma de decision invalida")
	ErrDecisionTecnicaInvalida     = errors.New("bolsa: decision tecnica invalida")
	ErrHistorialDecisionesInvalido = errors.New("bolsa: historial de decisiones invalido")
	ErrTransicionDecisionInvalida  = errors.New("bolsa: transicion de decision invalida")
	ErrDecisionSinCambios          = errors.New("bolsa: la rectificacion no cambia la valoracion")
)
var (
	ErrBolsaConstituidaInvalida      = errors.New("bolsa: bolsa constituida invalida")
	ErrSituacionBolsaInvalida        = errors.New("bolsa: situacion de participacion invalida")
	ErrParticipacionBolsaInvalida    = errors.New("bolsa: participacion invalida")
	ErrNecesidadCoberturaInvalida    = errors.New("bolsa: necesidad de cobertura invalida")
	ErrPoliticaLlamamientoInvalida   = errors.New("bolsa: politica de llamamiento invalida")
	ErrInstantaneaOrdenBolsaInvalida = errors.New("bolsa: instantanea de orden invalida")
	ErrEvaluacionLlamamientoInvalida = errors.New("bolsa: evaluacion de llamamiento invalida")
	ErrPropuestaLlamamientoInvalida  = errors.New("bolsa: propuesta de llamamiento invalida")
	ErrSinParticipacionElegible      = errors.New("bolsa: no existe participacion elegible evaluada")
)
var ErrConvocatoriaInvalida = errors.New("bolsa: convocatoria invalida")
```

### Tipos

```go
type AltaBolsaConstituida = BolsaConstituida

type AltaInstantaneaOrdenBolsa struct {
	InstantaneaRef string
	Version        uint64
	Bolsa          BolsaConstituida
	ReferidaEn     time.Time
	GeneradaEn     time.Time
	Entradas       []EntradaOrdenBolsa
}

type AltaMeritoBaremable struct {
	ID                  string                   `json:"id"`
	ProcesoRef          string                   `json:"proceso_ref"`
	SolicitudRef        string                   `json:"solicitud_ref"`
	SujetoRef           string                   `json:"sujeto_ref"`
	Criterio            ReferenciaCriterio       `json:"criterio"`
	EvidenciasIniciales []EvidenciaMerito        `json:"evidencias_iniciales"`
	PuntosDeclarados    Puntos                   `json:"puntos_declarados"`
	CalculoOficial      CalculoOficialBaremacion `json:"calculo_oficial"`
	CreadaEn            time.Time                `json:"creada_en"`
}
```

AltaMeritoBaremable contiene las referencias minimas para crear un merito
atomico. EvidenciasIniciales puede contener varios documentos que, solo en
conjunto, acreditan el mismo merito y no deben puntuar por separado.

```go
type AltaNecesidadCobertura = NecesidadCobertura

type AltaParticipacionBolsa = ParticipacionBolsa

type AyudaConvocatoria struct {
	Referencia string `json:"referencia"`
	Categoria  string `json:"categoria"`
	Orden      int    `json:"orden"`
	Pregunta   string `json:"pregunta"`
	Respuesta  string `json:"respuesta"`
}

type BaremacionMerito struct {
	ID                  string                   `json:"id"`
	ProcesoRef          string                   `json:"proceso_ref"`
	SolicitudRef        string                   `json:"solicitud_ref"`
	SujetoRef           string                   `json:"sujeto_ref"`
	Criterio            ReferenciaCriterio       `json:"criterio"`
	EvidenciasIniciales []EvidenciaMerito        `json:"evidencias_iniciales"`
	PuntosDeclarados    Puntos                   `json:"puntos_declarados"`
	CalculoInicial      CalculoOficialBaremacion `json:"calculo_inicial"`
	CreadaEn            time.Time                `json:"creada_en"`
	Decisiones          []DecisionTecnica        `json:"decisiones"`
}
```

BaremacionMerito es el historial de un unico merito bajo un unico criterio.
Puede reunir varias evidencias y Decisiones solo crece por incorporacion.

```go
func NuevaBaremacionMerito(alta AltaMeritoBaremable) (BaremacionMerito, error)

func (b BaremacionMerito) ClonarCanonica() (BaremacionMerito, error)

func (b BaremacionMerito) HistorialDecisiones() ([]DecisionTecnica, error)

func (b BaremacionMerito) HuellaEstadoAdministrativoSHA256() (string, error)
```

HuellaEstadoAdministrativoSHA256 encadena el estado que debe quedar cubierto
por la firma sin crear una autorreferencia criptografica. La base cubre
todos los datos de alta; cada eslabon cubre la huella anterior y el nuevo
contenido administrativo salvo sus dos campos de enlace. La validacion es
lineal.

```go
func (b BaremacionMerito) HuellaEstadoSHA256() (string, error)

func (b BaremacionMerito) IncorporarDecision(decision DecisionTecnica) (BaremacionMerito, error)
```

IncorporarDecision es la unica transicion del agregado. Solo incorpora una
decision ya firmada y devuelve una copia nueva con un asiento adicional.

```go
func (b BaremacionMerito) PrepararDecisionInicial(propuesta PropuestaDecisionTecnica) (ContenidoDecisionTecnica, error)

func (b BaremacionMerito) PrepararRectificacion(propuesta PropuestaDecisionTecnica) (ContenidoDecisionTecnica, error)
```

PrepararRectificacion corrige puntos, motivacion, fuentes o valoraciones
documentales. Los cambios que retiran o conceden una aceptacion se tipifican
expresamente mediante PrepararRevocacion o PrepararRehabilitacion.

```go
func (b BaremacionMerito) PrepararRehabilitacion(propuesta PropuestaDecisionTecnica) (ContenidoDecisionTecnica, error)
```

PrepararRehabilitacion acepta un merito antes desestimado o pendiente.
Los puntos y valoraciones se aportan completos para que la firma cubra el
nuevo juicio tecnico, incluida una posible evidencia de subsanacion.

```go
func (b BaremacionMerito) PrepararRevocacion(propuesta PropuestaDecisionTecnica) (ContenidoDecisionTecnica, error)
```

PrepararRevocacion retira una aceptacion previa sin borrar la decision que
la concedio. La propuesta debe incluir la nueva valoracion de cada evidencia
y puede dejar el merito desestimado o pendiente de subsanacion.

```go
func (b BaremacionMerito) UltimaDecision() (DecisionTecnica, bool)

func (b BaremacionMerito) Validar() error

type BolsaConstituida struct {
	BolsaRef                  string     `json:"bolsa_ref"`
	Version                   uint64     `json:"version"`
	ProcesoRef                string     `json:"proceso_ref"`
	CategoriaRef              string     `json:"categoria_ref"`
	ListadoDefinitivoRef      string     `json:"listado_definitivo_ref"`
	VersionListado            uint64     `json:"version_listado"`
	HuellaListadoSHA256       string     `json:"huella_listado_sha256"`
	ResolucionConstitucionRef string     `json:"resolucion_constitucion_ref"`
	HuellaResolucionSHA256    string     `json:"huella_resolucion_sha256"`
	ConstituidaEn             time.Time  `json:"constituida_en"`
	VigenteDesde              time.Time  `json:"vigente_desde"`
	VigenteHasta              *time.Time `json:"vigente_hasta,omitempty"`
}
```

BolsaConstituida fija la version juridicamente eficaz desde la que puede
operar una lista. El listado y la resolucion se enlazan por referencia y
huella; este agregado no acepta una coleccion de candidatos reenviada por un
adaptador de entrada.

```go
func NuevaBolsaConstituida(alta AltaBolsaConstituida) (BolsaConstituida, error)

func (b BolsaConstituida) ClonarCanonica() (BolsaConstituida, error)

func (b BolsaConstituida) HuellaCanonicaSHA256() (string, error)

func (b BolsaConstituida) Validar() error

func (b BolsaConstituida) VigenteEn(instante time.Time) bool

type CalculoOficialBaremacion struct {
	CalculoRef            string                 `json:"calculo_ref"`
	ProcesoRef            string                 `json:"proceso_ref"`
	SolicitudRef          string                 `json:"solicitud_ref"`
	SujetoRef             string                 `json:"sujeto_ref"`
	BaremacionMeritoRef   string                 `json:"baremacion_merito_ref"`
	Criterio              ReferenciaCriterio     `json:"criterio"`
	Regla                 ReferenciaReglaCalculo `json:"regla"`
	Evidencias            []EvidenciaMerito      `json:"evidencias"`
	EntradaRef            string                 `json:"entrada_ref"`
	HuellaEntradaSHA256   string                 `json:"huella_entrada_sha256"`
	PuntosCalculados      Puntos                 `json:"puntos_calculados"`
	DesgloseRef           string                 `json:"desglose_ref"`
	HuellaDesgloseSHA256  string                 `json:"huella_desglose_sha256"`
	ResultadoRef          string                 `json:"resultado_ref"`
	HuellaResultadoSHA256 string                 `json:"huella_resultado_sha256"`
	MotorCalculoRef       string                 `json:"motor_calculo_ref"`
	VersionMotorCalculo   string                 `json:"version_motor_calculo"`
	EvidenciaEjecucionRef string                 `json:"evidencia_ejecucion_ref"`
	HuellaEjecucionSHA256 string                 `json:"huella_ejecucion_sha256"`
	CalculadoEn           time.Time              `json:"calculado_en"`
}
```

CalculoOficialBaremacion es el recibo inmutable de un calculador gobernado.
EntradaRef/HuellaEntrada permiten reproducir exactamente el calculo sin
confiar en parametros reenviados por HTTP. ResultadoRef/HuellaResultado
identifican el recibo completo conservado por el conector.

```go
func (c CalculoOficialBaremacion) ClonarCanonico() (CalculoOficialBaremacion, error)

func (c CalculoOficialBaremacion) CoincideCon(otro CalculoOficialBaremacion) bool
```

CoincideCon compara recibos canonicos completos, incluida regla, huellas y
tiempo, sin confundir dos localizaciones horarias del mismo instante.

```go
func (c CalculoOficialBaremacion) EvidenciasCanonicas() ([]EvidenciaMerito, error)

func (c CalculoOficialBaremacion) Validar() error

type ClaseDecisionTecnica string
```

ClaseDecisionTecnica hace explicito por que se crea cada asiento del
historial. Una revocacion y una rehabilitacion nunca sobrescriben la
decision anterior: la sustituyen conservandola integra.

```go
const (
	ClaseDecisionInicial        ClaseDecisionTecnica = "inicial"
	ClaseDecisionRectificacion  ClaseDecisionTecnica = "rectificacion"
	ClaseDecisionRevocacion     ClaseDecisionTecnica = "revocacion"
	ClaseDecisionRehabilitacion ClaseDecisionTecnica = "rehabilitacion"
)
func (c ClaseDecisionTecnica) Valida() bool

type ContenidoDecisionTecnica struct {
	ID                           string                   `json:"id"`
	Numero                       int                      `json:"numero"`
	Clase                        ClaseDecisionTecnica     `json:"clase"`
	ProcesoRef                   string                   `json:"proceso_ref"`
	SolicitudRef                 string                   `json:"solicitud_ref"`
	SujetoRef                    string                   `json:"sujeto_ref"`
	BaremacionMeritoRef          string                   `json:"baremacion_merito_ref"`
	VersionAnteriorBaremacion    uint64                   `json:"version_anterior_baremacion"`
	VersionBaremacion            uint64                   `json:"version_baremacion"`
	HuellaEstadoAnteriorSHA256   string                   `json:"huella_estado_anterior_sha256"`
	HuellaEstadoResultanteSHA256 string                   `json:"huella_estado_resultante_sha256"`
	Criterio                     ReferenciaCriterio       `json:"criterio"`
	CalculoOficial               CalculoOficialBaremacion `json:"calculo_oficial"`
	ValoracionesEvidencia        []ValoracionEvidencia    `json:"valoraciones_evidencia"`
	PuntosDeclarados             Puntos                   `json:"puntos_declarados"`
	PuntosReconocidos            Puntos                   `json:"puntos_reconocidos"`
	Resultado                    ResultadoDecisionTecnica `json:"resultado"`
	DecisorRef                   string                   `json:"decisor_ref"`
	PerfilDecisorClave           string                   `json:"perfil_decisor_clave"`
	MotivoClave                  string                   `json:"motivo_clave"`
	Motivo                       string                   `json:"motivo"`
	FuentesNormativasRefs        []string                 `json:"fuentes_normativas_refs"`
	AutorizacionRef              string                   `json:"autorizacion_ref"`
	FinalidadClave               string                   `json:"finalidad_clave"`
	CorrelacionRef               string                   `json:"correlacion_ref"`
	DecididaEn                   time.Time                `json:"decidida_en"`
	Sustituye                    *ReferenciaDecision      `json:"sustituye,omitempty"`
}
```

ContenidoDecisionTecnica es el contenido administrativo que se firma.
Para obtener una firma valida se calcula primero HuellaContenidoSHA256 y se
entrega esa huella al conector de firma.

```go
func (c ContenidoDecisionTecnica) ClonarCanonico() (ContenidoDecisionTecnica, error)

func (c ContenidoDecisionTecnica) HuellaContenidoSHA256() (string, error)

func (c ContenidoDecisionTecnica) Validar() error

type Convocatoria struct {
	ID            string                     `json:"id"`
	Version       string                     `json:"version"`
	Estado        EstadoConvocatoria         `json:"estado"`
	DatosPublicos *DatosPublicosConvocatoria `json:"datos_publicos,omitempty"`
}
```

Convocatoria es el agregado canónico compartido por publicación pública,
solicitudes y baremación. DatosPublicos es opcional mientras el expediente
está en preparación, pero una consulta pública solo admite agregados cuya
publicación sea válida.

```go
func NuevaConvocatoria(id, version string) (Convocatoria, error)

func (c Convocatoria) Clonar() Convocatoria
```

Clonar devuelve una copia profunda para que ningún adaptador pueda modificar
el agregado compartido después de validarlo.

```go
func (c Convocatoria) HuellaPublicaSHA256() (string, error)

func (c Convocatoria) NewVersion(version string) (Convocatoria, error)

func (c Convocatoria) ValidarPublicacion() error

func (c Convocatoria) Validate() error
```

Validate conserva el nombre usado por el prototipo durante la migración.

```go
type DatosPublicosConvocatoria struct {
	IdentificadorPublico string                  `json:"identificador_publico"`
	Tipo                 string                  `json:"tipo"`
	Categorias           []string                `json:"categorias"`
	Titulo               string                  `json:"titulo"`
	Resumen              string                  `json:"resumen"`
	Descripcion          string                  `json:"descripcion"`
	PublicadaEn          time.Time               `json:"publicada_en"`
	ActualizadaEn        time.Time               `json:"actualizada_en"`
	Plazos               []PlazoConvocatoria     `json:"plazos"`
	Requisitos           []RequisitoConvocatoria `json:"requisitos"`
	Documentos           []DocumentoConvocatoria `json:"documentos"`
	Ayuda                []AyudaConvocatoria     `json:"ayuda"`
}

type DecisionTecnica struct {
	Contenido    ContenidoDecisionTecnica `json:"contenido"`
	Firma        FirmaDecisionTecnica     `json:"firma"`
	HuellaSHA256 string                   `json:"huella_sha256"`
}
```

DecisionTecnica es un asiento firmado e inmutable por huella. Cualquier
cambio posterior invalida Validar y debe expresarse como un asiento nuevo.

```go
func ConstituirDecisionFirmada(contenido ContenidoDecisionTecnica, firma FirmaDecisionTecnica) (DecisionTecnica, error)

func (d DecisionTecnica) Referencia() ReferenciaDecision

func (d DecisionTecnica) Validar() error

type DocumentoConvocatoria struct {
	Referencia  string    `json:"referencia"`
	Tipo        string    `json:"tipo"`
	Orden       int       `json:"orden"`
	Titulo      string    `json:"titulo"`
	Descripcion string    `json:"descripcion"`
	Formato     string    `json:"formato"`
	URL         string    `json:"url"`
	PublicadoEn time.Time `json:"publicado_en"`
}

type EntradaOrdenBolsa struct {
	Orden         uint64             `json:"orden"`
	Participacion ParticipacionBolsa `json:"participacion"`
}

type EstadoConvocatoria string
```

EstadoConvocatoria contiene una clave gobernada por el catalogo de estados.
IsValid comprueba solo la sintaxis tecnica; los valores permitidos,
etiquetas y semantica se publican como datos versionados por el adaptador de
catalogos.

```go
const (
	EstadoConvocatoriaBorrador    EstadoConvocatoria = "Borrador"
	EstadoConvocatoriaInscripcion EstadoConvocatoria = "Inscripcion"
	EstadoConvocatoriaSubsanacion EstadoConvocatoria = "Subsanacion"
	EstadoConvocatoriaAlegaciones EstadoConvocatoria = "Alegaciones"
	EstadoConvocatoriaDefinitiva  EstadoConvocatoria = "Definitiva"
	EstadoConvocatoriaCerrada     EstadoConvocatoria = "Cerrada"
)
```

Estas constantes mantienen la compatibilidad temporal del prototipo
candidate. No constituyen la lista de estados permitidos por el modulo
definitivo: cualquier clave valida debe existir en el catalogo gobernado.

```go
func (e EstadoConvocatoria) IsValid() bool

type EstadoValoracionEvidencia string

const (
	EstadoEvidenciaApta       EstadoValoracionEvidencia = "apta"
	EstadoEvidenciaNoApta     EstadoValoracionEvidencia = "no_apta"
	EstadoEvidenciaSubsanable EstadoValoracionEvidencia = "subsanable"
)
func (e EstadoValoracionEvidencia) Valido() bool

type EvaluacionParticipacionLlamamiento struct {
	ParticipacionRef        string                           `json:"participacion_ref"`
	SujetoRef               string                           `json:"sujeto_ref"`
	Orden                   uint64                           `json:"orden"`
	SituacionSecuencia      uint64                           `json:"situacion_secuencia"`
	EstadoClave             string                           `json:"estado_clave"`
	EstadoVersion           uint64                           `json:"estado_version"`
	HuellaEstadoSHA256      string                           `json:"huella_estado_sha256"`
	NecesidadRef            string                           `json:"necesidad_ref"`
	VersionNecesidad        uint64                           `json:"version_necesidad"`
	HuellaNecesidadSHA256   string                           `json:"huella_necesidad_sha256"`
	InstantaneaRef          string                           `json:"instantanea_ref"`
	VersionInstantanea      uint64                           `json:"version_instantanea"`
	HuellaInstantaneaSHA256 string                           `json:"huella_instantanea_sha256"`
	PoliticaRef             string                           `json:"politica_ref"`
	VersionPolitica         uint64                           `json:"version_politica"`
	HuellaPoliticaSHA256    string                           `json:"huella_politica_sha256"`
	Resultado               ResultadoElegibilidadLlamamiento `json:"resultado"`
	Motivos                 []MotivoEvaluacionLlamamiento    `json:"motivos"`
	EntradaEvaluacionRef    string                           `json:"entrada_evaluacion_ref"`
	HuellaEntradaSHA256     string                           `json:"huella_entrada_sha256"`
	ResultadoEvaluacionRef  string                           `json:"resultado_evaluacion_ref"`
	HuellaResultadoSHA256   string                           `json:"huella_resultado_sha256"`
	EvaluadaEn              time.Time                        `json:"evaluada_en"`
}
```

EvaluacionParticipacionLlamamiento es un recibo del motor de politicas. Sus
referencias y huellas enlazan necesidad, instantanea, politica y situacion;
EvaluadaEn registra la ejecucion real, posterior a la generacion de la
instantanea. Un booleano suelto nunca basta para proponer un llamamiento.

```go
func (e EvaluacionParticipacionLlamamiento) Validar() error

type EvidenciaMerito struct {
	Referencia    ReferenciaEvidencia  `json:"referencia"`
	SubsanacionDe *ReferenciaEvidencia `json:"subsanacion_de,omitempty"`
}
```

EvidenciaMerito incorpora, cuando procede, el vinculo exacto con el
documento al que subsana. Un merito puede necesitar varias evidencias
conjuntas y estas no se convierten artificialmente en meritos distintos.

```go
func (e EvidenciaMerito) Validar() error

type FirmaDecisionTecnica struct {
	FirmanteRef                            string    `json:"firmante_ref"`
	PerfilFirmanteClave                    string    `json:"perfil_firmante_clave"`
	PoliticaFirmaRef                       string    `json:"politica_firma_ref"`
	PoliticaFirmaVersion                   int       `json:"politica_firma_version"`
	HuellaPoliticaFirmaSHA256              string    `json:"huella_politica_firma_sha256"`
	PerfilFirmaAlcanzadoClave              string    `json:"perfil_firma_alcanzado_clave"`
	RequiereFirmaInteractiva               bool      `json:"requiere_firma_interactiva"`
	RequiereValidacionServidor             bool      `json:"requiere_validacion_servidor"`
	RequiereSelloTiempo                    bool      `json:"requiere_sello_tiempo"`
	RequiereAumentoLongevidad              bool      `json:"requiere_aumento_longevidad"`
	SesionFirmaInteractivaRef              string    `json:"sesion_firma_interactiva_ref"`
	HuellaEvidenciaFirmaInteractivaSHA256  string    `json:"huella_evidencia_firma_interactiva_sha256"`
	DocumentoFirmableRef                   string    `json:"documento_firmable_ref"`
	VersionDocumentoFirmable               string    `json:"version_documento_firmable"`
	HuellaDocumentoFirmableSHA256          string    `json:"huella_documento_firmable_sha256"`
	EvidenciaCustodiaRef                   string    `json:"evidencia_custodia_ref"`
	FirmaRef                               string    `json:"firma_ref"`
	HuellaFirmaSHA256                      string    `json:"huella_firma_sha256"`
	DocumentoFirmadoRef                    string    `json:"documento_firmado_ref"`
	HuellaDocumentoSHA256                  string    `json:"huella_documento_sha256"`
	DocumentoFirmadoCustodiadoRef          string    `json:"documento_firmado_custodiado_ref"`
	VersionDocumentoFirmadoCustodiado      string    `json:"version_documento_firmado_custodiado"`
	EvidenciaRecuperacionFirmadoRef        string    `json:"evidencia_recuperacion_firmado_ref"`
	HuellaEvidenciaRecuperacionSHA256      string    `json:"huella_evidencia_recuperacion_sha256"`
	EvidenciaCustodiaDocumentoFirmadoRef   string    `json:"evidencia_custodia_documento_firmado_ref"`
	EvidenciaRetencionDocumentoFirmadoRef  string    `json:"evidencia_retencion_documento_firmado_ref"`
	PoliticaRetencionDocumentoFirmadoRef   string    `json:"politica_retencion_documento_firmado_ref"`
	DocumentoFirmadoRetenidoHasta          time.Time `json:"documento_firmado_retenido_hasta"`
	ManifiestoProbatorioRef                string    `json:"manifiesto_probatorio_ref"`
	HuellaManifiestoProbatorioSHA256       string    `json:"huella_manifiesto_probatorio_sha256"`
	SelloManifiestoProbatorioHMACSHA256    string    `json:"sello_manifiesto_probatorio_hmac_sha256"`
	HuellaContenidoSHA256                  string    `json:"huella_contenido_sha256"`
	ValidacionInicialFirmaRef              string    `json:"validacion_inicial_firma_ref"`
	HuellaValidacionInicialSHA256          string    `json:"huella_validacion_inicial_sha256"`
	ValidadaInicialEn                      time.Time `json:"validada_inicial_en"`
	ValidacionFirmaRef                     string    `json:"validacion_firma_ref"`
	HuellaValidacionSHA256                 string    `json:"huella_validacion_sha256"`
	ValidadaEn                             time.Time `json:"validada_en"`
	SelloTiempoRef                         string    `json:"sello_tiempo_ref,omitempty"`
	HuellaSelloTiempoSHA256                string    `json:"huella_sello_tiempo_sha256,omitempty"`
	VinculoRevisionSelladaRef              string    `json:"vinculo_revision_sellada_ref,omitempty"`
	HuellaVinculoRevisionSelladaSHA256     string    `json:"huella_vinculo_revision_sellada_sha256,omitempty"`
	PoliticaSelloTiempoRef                 string    `json:"politica_sello_tiempo_ref,omitempty"`
	PoliticaSelloTiempoVersion             int       `json:"politica_sello_tiempo_version,omitempty"`
	HuellaPoliticaSelloTiempoSHA256        string    `json:"huella_politica_sello_tiempo_sha256,omitempty"`
	ValidacionSelloTiempoRef               string    `json:"validacion_sello_tiempo_ref,omitempty"`
	HuellaValidacionSelloTiempoSHA256      string    `json:"huella_validacion_sello_tiempo_sha256,omitempty"`
	SelladaEn                              time.Time `json:"sellada_en,omitempty"`
	ValidacionDocumentoSelladoRef          string    `json:"validacion_documento_sellado_ref,omitempty"`
	HuellaValidacionDocumentoSelladoSHA256 string    `json:"huella_validacion_documento_sellado_sha256,omitempty"`
	ValidadoDocumentoSelladoEn             time.Time `json:"validado_documento_sellado_en,omitempty"`
	NivelLongevidadClave                   string    `json:"nivel_longevidad_clave,omitempty"`
	AumentoLongevidadRef                   string    `json:"aumento_longevidad_ref,omitempty"`
	HuellaAumentoLongevidadSHA256          string    `json:"huella_aumento_longevidad_sha256,omitempty"`
	VinculoRevisionLongevaRef              string    `json:"vinculo_revision_longeva_ref,omitempty"`
	HuellaVinculoRevisionLongevaSHA256     string    `json:"huella_vinculo_revision_longeva_sha256,omitempty"`
	PoliticaLongevidadRef                  string    `json:"politica_longevidad_ref,omitempty"`
	PoliticaLongevidadVersion              int       `json:"politica_longevidad_version,omitempty"`
	HuellaPoliticaLongevidadSHA256         string    `json:"huella_politica_longevidad_sha256,omitempty"`
	ValidacionLongevidadRef                string    `json:"validacion_longevidad_ref,omitempty"`
	HuellaValidacionLongevidadSHA256       string    `json:"huella_validacion_longevidad_sha256,omitempty"`
	AumentadaEn                            time.Time `json:"aumentada_en,omitempty"`
	FirmadaEn                              time.Time `json:"firmada_en"`
}
```

FirmaDecisionTecnica conserva la evidencia verificable de la firma.
La validacion criptografica real pertenece al conector de firma; el dominio
exige sus referencias y vincula el resultado con la huella del contenido.

```go
type InstantaneaOrdenBolsa struct {
	InstantaneaRef        string              `json:"instantanea_ref"`
	Version               uint64              `json:"version"`
	BolsaRef              string              `json:"bolsa_ref"`
	VersionBolsa          uint64              `json:"version_bolsa"`
	HuellaBolsaSHA256     string              `json:"huella_bolsa_sha256"`
	ListadoDefinitivoRef  string              `json:"listado_definitivo_ref"`
	VersionListado        uint64              `json:"version_listado"`
	HuellaListadoSHA256   string              `json:"huella_listado_sha256"`
	ReferidaEn            time.Time           `json:"referida_en"`
	GeneradaEn            time.Time           `json:"generada_en"`
	Entradas              []EntradaOrdenBolsa `json:"entradas"`
	HuellaContenidoSHA256 string              `json:"huella_contenido_sha256"`
}
```

InstantaneaOrdenBolsa congela el orden completo y la situacion vigente de
cada participacion en un instante de referencia. La huella se calcula sobre
una representacion JSON de campos y orden fijos, nunca sobre un map.

```go
func NuevaInstantaneaOrdenBolsa(alta AltaInstantaneaOrdenBolsa) (InstantaneaOrdenBolsa, error)

func (i InstantaneaOrdenBolsa) ClonarCanonica() (InstantaneaOrdenBolsa, error)

func (i InstantaneaOrdenBolsa) Validar() error

type MotivoEvaluacionLlamamiento struct {
	Clave             string `json:"clave"`
	ReglaRef          string `json:"regla_ref"`
	VersionRegla      uint64 `json:"version_regla"`
	HuellaReglaSHA256 string `json:"huella_regla_sha256"`
}

func (m MotivoEvaluacionLlamamiento) Validar() error

type NecesidadCobertura struct {
	NecesidadRef      string               `json:"necesidad_ref"`
	Version           uint64               `json:"version"`
	BolsaRef          string               `json:"bolsa_ref"`
	VersionBolsa      uint64               `json:"version_bolsa"`
	HuellaBolsaSHA256 string               `json:"huella_bolsa_sha256"`
	CategoriaRef      string               `json:"categoria_ref"`
	PuestoRef         string               `json:"puesto_ref"`
	UnidadRef         string               `json:"unidad_ref"`
	TipoCoberturaRef  string               `json:"tipo_cobertura_ref"`
	NumeroPuestos     uint64               `json:"numero_puestos"`
	InicioPrevisto    time.Time            `json:"inicio_previsto"`
	FinPrevisto       time.Time            `json:"fin_previsto"`
	CreadaEn          time.Time            `json:"creada_en"`
	Requisitos        []RequisitoCobertura `json:"requisitos"`
}
```

NecesidadCobertura contiene solo datos exactos y referencias gobernadas.
La version y huella de bolsa impiden reutilizarla con otra constitucion. La
duracion se representa mediante instantes, no float ni una regla codificada.

```go
func NuevaNecesidadCobertura(alta AltaNecesidadCobertura) (NecesidadCobertura, error)

func (n NecesidadCobertura) ClonarCanonica() (NecesidadCobertura, error)

func (n NecesidadCobertura) HuellaCanonicaSHA256() (string, error)

func (n NecesidadCobertura) Validar() error

type OrdenProponerPrimerLlamamiento struct {
	PropuestaRef string
	Bolsa        BolsaConstituida
	Necesidad    NecesidadCobertura
	Instantanea  InstantaneaOrdenBolsa
	Politica     ReferenciaPoliticaLlamamiento
	Evaluaciones []EvaluacionParticipacionLlamamiento
	GeneradaEn   time.Time
}

type ParticipacionBolsa struct {
	ParticipacionRef string                        `json:"participacion_ref"`
	BolsaRef         string                        `json:"bolsa_ref"`
	SujetoRef        string                        `json:"sujeto_ref"`
	Version          uint64                        `json:"version"`
	AltaEn           time.Time                     `json:"alta_en"`
	Situaciones      []SituacionParticipacionBolsa `json:"situaciones"`
}
```

ParticipacionBolsa mantiene una historia completa y sin solapes. La posicion
no vive aqui: pertenece a cada InstantaneaOrdenBolsa versionada.

```go
func NuevaParticipacionBolsa(alta AltaParticipacionBolsa) (ParticipacionBolsa, error)

func (p ParticipacionBolsa) ClonarCanonica() (ParticipacionBolsa, error)

func (p ParticipacionBolsa) HuellaCanonicaSHA256() (string, error)

func (p ParticipacionBolsa) SituacionVigenteEn(instante time.Time) (SituacionParticipacionBolsa, bool)

func (p ParticipacionBolsa) Validar() error

type PlazoConvocatoria struct {
	Referencia  string    `json:"referencia"`
	Tipo        string    `json:"tipo"`
	Titulo      string    `json:"titulo"`
	Descripcion string    `json:"descripcion"`
	AbreEn      time.Time `json:"abre_en"`
	CierraEn    time.Time `json:"cierra_en"`
}

type PropuestaDecisionTecnica struct {
	ID                    string                   `json:"id"`
	CalculoOficial        CalculoOficialBaremacion `json:"calculo_oficial"`
	PuntosReconocidos     Puntos                   `json:"puntos_reconocidos"`
	Resultado             ResultadoDecisionTecnica `json:"resultado"`
	DecisorRef            string                   `json:"decisor_ref"`
	PerfilDecisorClave    string                   `json:"perfil_decisor_clave"`
	ValoracionesEvidencia []ValoracionEvidencia    `json:"valoraciones_evidencia"`
	MotivoClave           string                   `json:"motivo_clave"`
	Motivo                string                   `json:"motivo"`
	FuentesNormativasRefs []string                 `json:"fuentes_normativas_refs"`
	AutorizacionRef       string                   `json:"autorizacion_ref"`
	FinalidadClave        string                   `json:"finalidad_clave"`
	CorrelacionRef        string                   `json:"correlacion_ref"`
	DecididaEn            time.Time                `json:"decidida_en"`
}
```

PropuestaDecisionTecnica expresa una valoracion antes de ser firmada.
La clase, numero, merito y referencia sustituida los determina el agregado.

```go
type PropuestaLlamamiento struct {
	PropuestaRef                    string                               `json:"propuesta_ref"`
	BolsaRef                        string                               `json:"bolsa_ref"`
	VersionBolsa                    uint64                               `json:"version_bolsa"`
	HuellaBolsaSHA256               string                               `json:"huella_bolsa_sha256"`
	NecesidadRef                    string                               `json:"necesidad_ref"`
	VersionNecesidad                uint64                               `json:"version_necesidad"`
	HuellaNecesidadSHA256           string                               `json:"huella_necesidad_sha256"`
	InstantaneaRef                  string                               `json:"instantanea_ref"`
	VersionInstantanea              uint64                               `json:"version_instantanea"`
	HuellaInstantaneaSHA256         string                               `json:"huella_instantanea_sha256"`
	PoliticaRef                     string                               `json:"politica_ref"`
	VersionPolitica                 uint64                               `json:"version_politica"`
	HuellaPoliticaSHA256            string                               `json:"huella_politica_sha256"`
	InstanteReferencia              time.Time                            `json:"instante_referencia"`
	InstantaneaGeneradaEn           time.Time                            `json:"instantanea_generada_en"`
	TotalParticipacionesInstantanea uint64                               `json:"total_participaciones_instantanea"`
	Evaluaciones                    []EvaluacionParticipacionLlamamiento `json:"evaluaciones"`
	ParticipacionSeleccionadaRef    string                               `json:"participacion_seleccionada_ref"`
	SujetoSeleccionadoRef           string                               `json:"sujeto_seleccionado_ref"`
	OrdenSeleccionado               uint64                               `json:"orden_seleccionado"`
	GeneradaEn                      time.Time                            `json:"generada_en"`
	HuellaContenidoSHA256           string                               `json:"huella_contenido_sha256"`
}
```

PropuestaLlamamiento conserva el prefijo completo del orden hasta la primera
participacion elegible y la cronologia causal de sus recibos. Asi demuestra
que no se ha omitido a nadie anterior sin tratar innecesariamente a quienes
se encuentran despues.

```go
func ProponerPrimerLlamamiento(orden OrdenProponerPrimerLlamamiento) (PropuestaLlamamiento, error)

func (p PropuestaLlamamiento) ClonarCanonica() (PropuestaLlamamiento, error)

func (p PropuestaLlamamiento) EvaluacionesCanonicas() ([]EvaluacionParticipacionLlamamiento, error)

func (p PropuestaLlamamiento) Validar() error

type Puntos int64
```

Puntos almacena micropuntos. Por ejemplo, 2,75 puntos se representan como
2_750_000. El dominio no admite float32 ni float64 para evitar redondeos.

```go
func (p Puntos) Validos() bool

type ReferenciaCriterio struct {
	ProcesoRef    string                 `json:"proceso_ref"`
	Clave         string                 `json:"clave"`
	Version       int                    `json:"version"`
	HuellaSHA256  string                 `json:"huella_sha256"`
	PuntosMaximos Puntos                 `json:"puntos_maximos"`
	ReglaCalculo  ReferenciaReglaCalculo `json:"regla_calculo"`
}
```

ReferenciaCriterio fija la configuracion exacta aplicada y el proceso que
la publica. Clave y ReglaCalculo son datos gobernados: incorporar un tipo de
merito o una formula nueva no obliga a recompilar.

```go
func (r ReferenciaCriterio) Validar() error

type ReferenciaDecision struct {
	ID           string `json:"id"`
	Numero       int    `json:"numero"`
	HuellaSHA256 string `json:"huella_sha256"`
}
```

ReferenciaDecision enlaza una rectificacion con la decision exacta que
sustituye. La huella impide que una referencia estable apunte despues a un
contenido distinto.

```go
func (r ReferenciaDecision) Validar() error

type ReferenciaEvidencia struct {
	DocumentoRef      string `json:"documento_ref"`
	VersionDocumento  int    `json:"version_documento"`
	RepresentacionRef string `json:"representacion_ref"`
	HuellaSHA256      string `json:"huella_sha256"`
}
```

ReferenciaEvidencia identifica los bytes exactos del documento evaluado.
DocumentoRef y RepresentacionRef deben ser referencias internas opacas,
sin DNI, nombre, correo u otros datos personales embebidos.

```go
func (r ReferenciaEvidencia) Validar() error

type ReferenciaPoliticaLlamamiento struct {
	PoliticaRef  string     `json:"politica_ref"`
	Clave        string     `json:"clave"`
	Version      uint64     `json:"version"`
	HuellaSHA256 string     `json:"huella_sha256"`
	PublicadaEn  time.Time  `json:"publicada_en"`
	VigenteDesde time.Time  `json:"vigente_desde"`
	VigenteHasta *time.Time `json:"vigente_hasta,omitempty"`
}
```

ReferenciaPoliticaLlamamiento fija los bytes y la vigencia de la politica
usada. Su contenido ejecutable vive fuera de este nucleo puro.

```go
func NuevaReferenciaPoliticaLlamamiento(datos ReferenciaPoliticaLlamamiento) (ReferenciaPoliticaLlamamiento, error)

func (p ReferenciaPoliticaLlamamiento) ClonarCanonica() (ReferenciaPoliticaLlamamiento, error)

func (p ReferenciaPoliticaLlamamiento) Validar() error

func (p ReferenciaPoliticaLlamamiento) VigenteEn(instante time.Time) bool

type ReferenciaReglaCalculo struct {
	Clave        string `json:"clave"`
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}
```

ReferenciaReglaCalculo identifica la regla ejecutable exacta que produjo la
puntuacion oficial. La regla es configuracion gobernada, no codigo elegido
por el cliente ni una constante compilada en este paquete.

```go
func (r ReferenciaReglaCalculo) Validar() error

type RequisitoCobertura struct {
	Clave        string `json:"clave"`
	ValorRef     string `json:"valor_ref"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}
```

RequisitoCobertura referencia una condicion gobernada. Clave y valor no son
texto libre; la interpretacion corresponde a la politica versionada.

```go
func (r RequisitoCobertura) Validar() error

type RequisitoConvocatoria struct {
	Referencia  string `json:"referencia"`
	Orden       int    `json:"orden"`
	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`
	Obligatorio bool   `json:"obligatorio"`
}

type ResultadoDecisionTecnica string
```

ResultadoDecisionTecnica es una invariante del procedimiento, no un catalogo
de tipos de merito. Las categorias de meritos viven en criterios versionados
mediante ReferenciaCriterio.

```go
const (
	ResultadoAceptado             ResultadoDecisionTecnica = "aceptado"
	ResultadoDesestimado          ResultadoDecisionTecnica = "desestimado"
	ResultadoPendienteSubsanacion ResultadoDecisionTecnica = "pendiente_subsanacion"
)
func (r ResultadoDecisionTecnica) Valido() bool

type ResultadoElegibilidadLlamamiento string

const (
	ResultadoElegible   ResultadoElegibilidadLlamamiento = "elegible"
	ResultadoNoElegible ResultadoElegibilidadLlamamiento = "no_elegible"
)
func (r ResultadoElegibilidadLlamamiento) Valido() bool

type ResultadoSubsanacion string

const (
	ResultadoSubsanacionNoAplica  ResultadoSubsanacion = "no_aplica"
	ResultadoSubsanacionPendiente ResultadoSubsanacion = "pendiente"
	ResultadoSubsanacionAceptada  ResultadoSubsanacion = "aceptada"
	ResultadoSubsanacionRechazada ResultadoSubsanacion = "rechazada"
)
func (r ResultadoSubsanacion) Valido() bool

type SituacionParticipacionBolsa struct {
	Secuencia            uint64     `json:"secuencia"`
	EstadoClave          string     `json:"estado_clave"`
	EstadoVersion        uint64     `json:"estado_version"`
	HuellaEstadoSHA256   string     `json:"huella_estado_sha256"`
	CausaClave           string     `json:"causa_clave"`
	CausaVersion         uint64     `json:"causa_version"`
	HuellaCausaSHA256    string     `json:"huella_causa_sha256"`
	DecisionRef          string     `json:"decision_ref"`
	HuellaDecisionSHA256 string     `json:"huella_decision_sha256"`
	Desde                time.Time  `json:"desde"`
	Hasta                *time.Time `json:"hasta,omitempty"`
}
```

SituacionParticipacionBolsa representa un periodo semicerrado [Desde,
Hasta). Estado y causa son entradas gobernadas y versionadas; el nucleo no
incorpora una lista estatica de situaciones ni reglas temporales concretas.

```go
func (s SituacionParticipacionBolsa) Validar() error

type ValoracionEvidencia struct {
	Evidencia            EvidenciaMerito           `json:"evidencia"`
	Estado               EstadoValoracionEvidencia `json:"estado"`
	ResultadoSubsanacion ResultadoSubsanacion      `json:"resultado_subsanacion"`
	MotivoClave          string                    `json:"motivo_clave"`
	Motivo               string                    `json:"motivo"`
}
```

ValoracionEvidencia deja constancia separada del juicio tecnico sobre cada
documento, aunque el resultado y los puntos se decidan globalmente para el
merito atomico.

```go
func (v ValoracionEvidencia) Validar() error
```

## Paquete `internal/modules/bolsa/internal/transaccion`

> Package transaccion concentra la derivacion canonica de la evidencia probatoria del modulo Bolsa.

Package transaccion concentra la derivacion canonica de la evidencia probatoria
del modulo Bolsa. Los adaptadores duraderos y efimeros comparten asi exactamente
las mismas huellas, referencias y reglas de encadenado.

La ruta internal impide que un cliente HTTP pueda presentar auditorias o eventos
ya construidos: solo los componentes del modulo Bolsa pueden derivarlos a partir
de una solicitud de confirmacion validada.

### Funciones

```go
func DerivarEvidencia(
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
	versionAnterior, versionNueva uint64,
	huellaAnterior, huellaNueva, huellaAuditoriaAnterior, huellaEventoAnterior string,
	secuenciaAuditoria, secuenciaEvento uint64,
	registradaEn time.Time,
) (
	puertosbolsa.RegistroAuditoriaBaremacion,
	puertosbolsa.EventoOutboxBaremacion,
	puertosbolsa.EvidenciaTransaccionBaremacion,
	error,
)
```

DerivarEvidencia crea el registro de auditoria, el evento outbox y el recibo
que deben persistirse junto a la nueva version bajo una unica transaccion.
Las referencias se generan con 256 bits aleatorios y no contienen datos
personales ni referencias de negocio.

```go
func GenerarTokenReserva() (puertosbolsa.TokenReservaBaremacion, error)
```

GenerarTokenReserva crea una capacidad temporal apta para el contrato del
puerto, sin exponer el material aleatorio en logs o serializadores.

```go
func HuellaAuditoria(a puertosbolsa.RegistroAuditoriaBaremacion) string
```

HuellaAuditoria calcula la cadena canonica sin incluir la propia huella.

```go
func HuellaCanonica(partes ...string) string
```

HuellaCanonica usa longitudes binarias para evitar concatenaciones ambiguas.

```go
func HuellaEfectoAbandono(solicitud puertosbolsa.SolicitudAbandonarReservaBaremacion) (string, error)
```

HuellaEfectoAbandono liga el abandono al token, clase y baremacion exactos.

```go
func HuellaEfectoConfirmacionV2(solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion) (string, error)
```

HuellaEfectoConfirmacionV2 cubre la version, el agregado, la trazabilidad y
las dos autorizaciones usadas por la mutacion definitiva.

```go
func HuellaEfectoPrevalidacionArchivoProbatorio(
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
) (string, error)
```

HuellaEfectoPrevalidacionArchivoProbatorio liga el permiso consumible de
prevalidacion al efecto completo que se confirmara. Una autorizacion no
puede trasladarse a otra version, manifiesto, actor ni autorizacion de
confirmacion.

```go
func HuellaEfectoReserva(solicitud puertosbolsa.SolicitudReservarCambioBaremacion) (string, error)
```

HuellaEfectoReserva cubre la representacion canonica completa, incluida la
decision de autorizacion y el vinculo de autenticacion.

```go
func HuellaEvento(e puertosbolsa.EventoOutboxBaremacion) string
```

HuellaEvento calcula la cadena canonica sin incluir la propia huella.

```go
func HuellaTokenReserva(token puertosbolsa.TokenReservaBaremacion) string
```

HuellaTokenReserva permite localizar una reserva sin guardar la capacidad en
claro. El token original solo se devuelve una vez al llamador.

```go
func MismoUso(a, b UsoAutorizacion) bool
```

MismoUso comprueba en tiempo constante las huellas antes de tratar un
reintento como la misma operacion.

```go
func NuevaReferenciaOpaca() (string, error)
```

NuevaReferenciaOpaca devuelve una referencia Base64URL de 256 bits.

### Tipos

```go
type PruebaDecisionAutorizacion struct {
	Uso                    UsoAutorizacion
	EsquemaHuella          string
	Decision               dominiovec.DecisionAutorizacion
	RepresentacionCanonica []byte
	VerificadaEn           time.Time
}
```

PruebaDecisionAutorizacion es la proyeccion privada que un adaptador
duradero necesita para comparar la fila de autorizacion y su representacion
reforzada. Los bytes son una copia de la codificacion que produjo la huella;
el adaptador no conoce ni replica ese serializador.

```go
func ExtraerPruebaDecisionAutorizacion(
	contexto puertosbolsa.ContextoOperacionBaremacion,
	instante time.Time,
	huellaEfecto string,
) (PruebaDecisionAutorizacion, error)
```

ExtraerPruebaDecisionAutorizacion solo acepta una capacidad viva del nucleo.
Devuelve copias independientes para que la serializacion SQL no pueda mutar
la evidencia mantenida en memoria.

```go
func (p PruebaDecisionAutorizacion) Validar() error

type UsoAutorizacion struct {
	DecisionRef          string
	HuellaDecisionSHA256 string
	HuellaEfectoSHA256   string
}
```

UsoAutorizacion liga de manera durable una decision exacta con un unico
efecto. Un adaptador puede persistir estos tres valores, pero solo esta
fabrica interna puede extraerlos de la capacidad opaca del nucleo.

```go
func NuevoUsoAutorizacion(
	contexto puertosbolsa.ContextoOperacionBaremacion,
	instante time.Time,
	huellaEfecto string,
) (UsoAutorizacion, error)
```

NuevoUsoAutorizacion obtiene la decision y su huella desde la evidencia
opaca creada por el nucleo. Nunca acepta esos datos como argumentos sueltos.

```go
func (u UsoAutorizacion) Validar() error
```

## Paquete `internal/modules/bolsa/ports`

> Package ports declara contratos hexagonales del modulo de bolsas.

Package ports declara contratos hexagonales del modulo de bolsas. Ninguno
depende de un motor de datos, proveedor de firma o producto de almacenamiento.

Package ports define las fronteras hexagonales del modulo de bolsa.

### Constantes

```go
const (
	// MaximoTamanoCargaProtegida permite aplicar el mismo limite antes de
	// decodificar entradas no confiables en los adaptadores.
	MaximoTamanoCargaProtegida = 64 << 20

	VentanaMaximaReservaBaremacion      = 10 * time.Minute
	VentanaMaximaSesionFirmaInteractiva = 15 * time.Minute
)
const (
	FormatoFirmaPDFCanonico       = "pdf_canonico"
	PerfilFirmaPAdESBaselineB     = "pades_baseline_b"
	PerfilFirmaPAdESBaselineT     = "pades_baseline_t"
	PerfilFirmaPAdESBaselineLTA   = "pades_baseline_lta"
	AlgoritmoHuellaFirmaSHA256    = "sha256"
	ComprobacionIntegridadFirma   = "integridad_criptografica"
	ComprobacionCadenaConfianza   = "cadena_confianza"
	ComprobacionRevocacionFirma   = "revocacion_instante_firma"
	ComprobacionIdentidadFirmante = "identidad_firmante"
	ComprobacionPoliticaFirma     = "politica_firma"
	ComprobacionDigestDocumento   = "digest_documento"
	ComprobacionAlgoritmosFirma   = "algoritmos_permitidos"
	ComprobacionFormatoPAdES      = "formato_pades"
	ComprobacionPerfilPAdES       = "perfil_pades"
)
const (
	CatalogoTiposConvocatoria      = "tipos_convocatoria"
	CatalogoEstadosConvocatoria    = "estados_convocatoria"
	CatalogoCategoriasConvocatoria = "categorias_convocatoria"
	CatalogoTiposPlazo             = "tipos_plazo"
	CatalogoTiposDocumento         = "tipos_documento"
	CatalogoCategoriasAyuda        = "categorias_ayuda"
)
const (
	EsquemaEstadoProtegidoFlujoFirmaBaremacion = "bolsa.firma.estado-protegido.v1"
	AlgoritmoProteccionEstadoAES256GCM         = "aes-256-gcm"
	DuracionMaximaArrendamientoFlujoFirma      = 5 * time.Minute
)
const (
	// VersionIndiceIdempotenciaBaremacionV1 identifica la derivacion HMAC
	// adoptada por DEC-045. Una rotacion de clave cambia ClaveHMACRef, no esta
	// version del esquema.
	VersionIndiceIdempotenciaBaremacionV1 uint16 = 1
	VersionPrincipalEstableBaremacionV1   uint16 = 1
	VersionSeudonimoSujetoBaremacionV1    uint16 = 1

	// VersionIntencionCambioBaremacionV1 cubre la incorporacion de una decision
	// tecnica ya firmada, custodiada y retenida. El alta inicial necesitara una
	// proyeccion distinta: no se admite rellenar sus campos con valores ficticios.
	VersionIntencionCambioBaremacionV1 uint16 = 1

	// VersionHMACIntencionCambioBaremacionV1 versiona el sobre criptografico,
	// no la clave. ClaveHMACRef identifica la clave concreta del llavero y
	// permite verificar sellos anteriores despues de una rotacion.
	VersionHMACIntencionCambioBaremacionV1 uint16 = 1

	// VersionHMACMotivoBaremacionV1 pertenece a un dominio criptografico
	// separado de indice, sujeto e intencion.
	VersionHMACMotivoBaremacionV1 uint16 = 1

	// VersionHMACManifiestoMaterialBaremacionV2 identifica el sello nominal
	// del manifiesto estable V2; no admite el sobre textual generico de V1.
	VersionHMACManifiestoMaterialBaremacionV2 uint16 = 2
)
const (
	// EsquemaCanonicoPrincipalEstableBaremacionV1 y
	// EsquemaCanonicoIndiceIdempotenciaBaremacionV1 fijan las dos formulas de
	// DEC-045. PoliticaDerivacionIdempotenciaBaremacionDEC045V1 forma parte de
	// las preimagenes y del material atestado; no es un texto declarativo libre.
	EsquemaCanonicoSeudonimoSujetoBaremacionV1       = "vec.bolsa.seudonimo-sujeto.v1"
	EsquemaCanonicoPrincipalEstableBaremacionV1      = "vec.bolsa.principal-estable.v1"
	EsquemaCanonicoIndiceIdempotenciaBaremacionV1    = "vec.bolsa.indice-idempotencia.v1"
	PoliticaDerivacionIdempotenciaBaremacionDEC045V1 = "vec.bolsa.politica-derivacion.dec-045.v1"
)
const (
	EsquemaManifiestoMaterialEstableBaremacionV2 EsquemaMaterialEstableBaremacion = "vec.bolsa.manifiesto-material-estable.v2"
	EsquemaPlanFirmaDurableBaremacionV2          EsquemaMaterialEstableBaremacion = "vec.bolsa.plan-firma-durable-material.v2"
	EsquemaReciboRecuperacionBaremacionV2        EsquemaMaterialEstableBaremacion = "vec.bolsa.recibo-recuperacion-firmado.v2"
	EsquemaReciboCustodiaBaremacionV2            EsquemaMaterialEstableBaremacion = "vec.bolsa.recibo-custodia-firmado.v2"
	EsquemaReciboRetencionBaremacionV2           EsquemaMaterialEstableBaremacion = "vec.bolsa.recibo-retencion-firmado.v2"

	VersionManifiestoMaterialEstableBaremacionV2 uint16 = 2
	VersionPlanFirmaDurableBaremacionV2          uint16 = 2
	VersionReciboRecuperacionBaremacionV2        uint16 = 2
	VersionReciboCustodiaBaremacionV2            uint16 = 2
	VersionReciboRetencionBaremacionV2           uint16 = 2
)
const (
	AccionProponerLlamamiento    = "bolsa.llamamiento.proponer"
	FinalidadProponerLlamamiento = "gestion_propuestas_llamamiento"
	ModuloLlamamientos           = "bolsa"
	TipoRecursoNecesidad         = "necesidad_cobertura"
)
const (
	EsquemaManifiestoProbatorioBaremacion   = "vec.bolsa.manifiesto_probatorio"
	FinalidadManifiestoProbatorioBaremacion = "decision_tecnica_baremacion"
	// VersionManifiestoProbatorioBaremacionV2 queda congelada para identificar
	// archivos preproductivos antiguos. El productor vigente solo emite V3.
	VersionManifiestoProbatorioBaremacionV2 = 2
	VersionManifiestoProbatorioBaremacion   = 3
)
const VentanaMaximaUsoAutorizacionBaremacion = 30 * time.Second
```

VentanaMaximaUsoAutorizacionBaremacion limita el tiempo durante el que una
decision ya evaluada puede viajar hasta el punto de aplicacion.

### Variables

```go
var (
	ErrSolicitudBaremacionInvalida             = errors.New("bolsa: solicitud de persistencia invalida")
	ErrBaremacionNoEncontrada                  = errors.New("bolsa: baremacion no encontrada")
	ErrVersionBaremacionNoEncontrada           = errors.New("bolsa: version de baremacion no encontrada")
	ErrBaremacionYaExiste                      = errors.New("bolsa: baremacion ya existente")
	ErrVersionBaremacionConflicto              = errors.New("bolsa: version de baremacion en conflicto")
	ErrHistorialBaremacionNoAnexable           = errors.New("bolsa: el historial no puede anexarse")
	ErrClaveIdempotenciaBaremacionInvalida     = errors.New("bolsa: clave de idempotencia invalida")
	ErrClaveIdempotenciaBaremacionReutilizada  = errors.New("bolsa: clave de idempotencia reutilizada")
	ErrCambioBaremacionEnCurso                 = errors.New("bolsa: cambio de baremacion en curso")
	ErrReservaBaremacionNoValida               = errors.New("bolsa: reserva de baremacion no valida")
	ErrFuenteBaremacionNoDisponible            = errors.New("bolsa: fuente fiable no disponible")
	ErrCriterioBaremacionNoEncontrado          = errors.New("bolsa: criterio de baremacion no encontrado")
	ErrCriterioBaremacionNoVigente             = errors.New("bolsa: criterio de baremacion no vigente")
	ErrEvidenciaBaremacionNoEncontrada         = errors.New("bolsa: evidencia de baremacion no encontrada")
	ErrEvidenciaBaremacionNoConfiable          = errors.New("bolsa: evidencia de baremacion no confiable")
	ErrRepresentacionBaremacionNoEncontrada    = errors.New("bolsa: representacion documental no encontrada")
	ErrRepresentacionBaremacionNoConfiable     = errors.New("bolsa: representacion documental no confiable")
	ErrCalculoOficialNoDisponible              = errors.New("bolsa: calculo oficial no disponible")
	ErrCalculoOficialNoReproducible            = errors.New("bolsa: calculo oficial no reproducible")
	ErrPoliticaFirmaNoEncontrada               = errors.New("bolsa: politica de firma no encontrada")
	ErrPoliticaFirmaNoVigente                  = errors.New("bolsa: politica de firma no vigente")
	ErrPoliticaFirmaInsegura                   = errors.New("bolsa: politica de firma no cumple los minimos")
	ErrCodificacionCanonicaNoDisponible        = errors.New("bolsa: codificacion canonica no disponible")
	ErrCustodiaDocumentoFirmableInvalida       = errors.New("bolsa: custodia de documento firmable invalida")
	ErrCargaProtegidaInvalida                  = errors.New("bolsa: carga protegida invalida")
	ErrSerializacionCargaProtegidaProhibida    = errors.New("bolsa: serializacion de carga protegida prohibida")
	ErrTokenReservaBaremacionInvalido          = errors.New("bolsa: token de reserva invalido")
	ErrSerializacionTokenReservaProhibida      = errors.New("bolsa: serializacion de token de reserva prohibida")
	ErrFirmaInteractivaNoDisponible            = errors.New("bolsa: firma interactiva no disponible")
	ErrSesionFirmaNoEncontrada                 = errors.New("bolsa: sesion de firma no encontrada")
	ErrSesionFirmaExpirada                     = errors.New("bolsa: sesion de firma expirada")
	ErrFirmaInteractivaNoCompletada            = errors.New("bolsa: firma interactiva no completada")
	ErrValidacionFirmaNoDisponible             = errors.New("bolsa: validacion de firma no disponible")
	ErrFirmaServidorNoValida                   = errors.New("bolsa: firma no valida")
	ErrValidacionFirmaNoConcluyente            = errors.New("bolsa: validacion de firma no concluyente")
	ErrRevisionPDFFirmaNoConfiable             = errors.New("bolsa: revision PDF de firma no confiable")
	ErrSelloTiempoNoDisponible                 = errors.New("bolsa: sello de tiempo no disponible")
	ErrAumentoFirmaNoDisponible                = errors.New("bolsa: aumento de firma no disponible")
	ErrEvidenciaFirmaNoEncontrada              = errors.New("bolsa: evidencia historica de firma no encontrada")
	ErrGeneracionReferenciaNoDisponible        = errors.New("bolsa: generacion de referencia no disponible")
	ErrAutorizacionBaremacionInvalida          = errors.New("bolsa: autorizacion de operacion invalida")
	ErrAutorizacionBaremacionReutilizada       = errors.New("bolsa: decision de autorizacion reutilizada para otro efecto")
	ErrSerializacionAutorizacionProhibida      = errors.New("bolsa: serializacion de autorizacion prohibida")
	ErrVerificacionSelloBaremacionNoDisponible = errors.New("bolsa: verificacion de sello no disponible")
	ErrSelloBaremacionNoAutentico              = errors.New("bolsa: sello de operacion no autentico")
)
var (
	ErrConsultaConvocatoriasInvalida = errors.New("bolsa: consulta publica de convocatorias invalida")
	ErrConvocatoriaNoEncontrada      = errors.New("bolsa: convocatoria publica no encontrada")
	ErrFuenteConvocatoriasInvalida   = errors.New("bolsa: fuente publica de convocatorias invalida")
)
var (
	ErrSolicitudFlujoFirmaBaremacionInvalida = errors.New("bolsa: solicitud de flujo de firma invalida")
	ErrFlujoFirmaBaremacionNoEncontrado      = errors.New("bolsa: flujo de firma no encontrado")
	ErrClaveFlujoFirmaBaremacionReutilizada  = errors.New("bolsa: clave de flujo reutilizada con otros datos")
	ErrConflictoFlujoFirmaBaremacion         = errors.New("bolsa: conflicto de version del flujo de firma")
	ErrFlujoFirmaBaremacionOcupado           = errors.New("bolsa: flujo de firma ocupado")
	ErrArrendamientoFlujoFirmaInvalido       = errors.New("bolsa: arrendamiento de flujo de firma invalido")
	ErrSerializacionArrendamientoProhibida   = errors.New("bolsa: serializacion de arrendamiento de flujo de firma prohibida")
	ErrEstadoFlujoFirmaAlterado              = errors.New("bolsa: estado protegido del flujo de firma alterado")
	ErrPasoFlujoFirmaNoPermitido             = errors.New("bolsa: paso de flujo de firma no permitido")
	ErrSerializacionEstadoFlujoProhibida     = errors.New("bolsa: serializacion generica de estado de flujo prohibida")
)
var (
	ErrSeudonimoSujetoBaremacionInvalido        = errors.New("bolsa: seudonimo HMAC de sujeto invalido")
	ErrPrincipalEstableBaremacionInvalido       = errors.New("bolsa: principal estable HMAC invalido")
	ErrHMACIntencionCambioBaremacionInvalido    = errors.New("bolsa: HMAC de intencion de cambio invalido")
	ErrHMACMotivoBaremacionInvalido             = errors.New("bolsa: HMAC de motivo de baremacion invalido")
	ErrHMACManifiestoMaterialBaremacionInvalido = errors.New("bolsa: HMAC de manifiesto material invalido")
	ErrSerializacionIdempotenciaBaremacion      = errors.New("bolsa: serializacion de idempotencia semantica prohibida")
	ErrCoincidenciaIdempotenciaAmbigua          = errors.New("bolsa: coincidencia de idempotencia ausente, multiple o ajena")
	ErrSeparacionDominiosClaveBaremacion        = errors.New("bolsa: separacion de dominios de clave no acreditada")
)
var (
	ErrSolicitudPropuestaLlamamientoInvalida      = errors.New("bolsa: solicitud de propuesta de llamamiento invalida")
	ErrRecursoNecesidadNoEncontrado               = errors.New("bolsa: recurso de necesidad no encontrado")
	ErrRecursoNecesidadAmbiguo                    = errors.New("bolsa: recurso de necesidad ambiguo")
	ErrRecursoNecesidadNoConfiable                = errors.New("bolsa: recurso de necesidad no confiable")
	ErrDatosLlamamientoNoEncontrados              = errors.New("bolsa: datos de llamamiento no encontrados")
	ErrDatosLlamamientoAmbiguos                   = errors.New("bolsa: datos de llamamiento ambiguos")
	ErrDatosLlamamientoNoConfiables               = errors.New("bolsa: datos de llamamiento no confiables")
	ErrMotorElegibilidadNoDisponible              = errors.New("bolsa: motor de elegibilidad no disponible")
	ErrEvaluacionMotorNoConfiable                 = errors.New("bolsa: evaluacion del motor no confiable")
	ErrGeneracionReferenciaLlamamiento            = errors.New("bolsa: no se pudo generar una referencia de llamamiento")
	ErrPersistenciaPropuestaNoDisponible          = errors.New("bolsa: persistencia de propuesta no disponible")
	ErrPropuestaLlamamientoYaExiste               = errors.New("bolsa: propuesta de llamamiento ya existe")
	ErrNecesidadLlamamientoYaPropuesta            = errors.New("bolsa: la version de la necesidad ya tiene propuesta de llamamiento")
	ErrReferenciaLlamamientoYaUtilizada           = errors.New("bolsa: referencia de llamamiento ya utilizada")
	ErrDecisionAutorizacionLlamamientoUsada       = errors.New("bolsa: decision de autorizacion ya consumida")
	ErrCapacidadMemoriaLlamamientosAgotada        = errors.New("bolsa: capacidad del adaptador de memoria agotada")
	ErrSerializacionSolicitudLlamamientoProhibida = errors.New("bolsa: serializacion de solicitud interna de llamamiento prohibida")
)
var (
	// ErrResultadoTransaccionalBaremacionInvalido impide convertir un valor
	// incompleto o fabricado en una afirmacion sobre el resultado del COMMIT.
	ErrResultadoTransaccionalBaremacionInvalido = errors.New("bolsa: resultado transaccional invalido")

	// ErrTransaccionBaremacionNoAplicada solo identifica el desenlace negativo
	// que lleva una prueba autenticada de no aplicacion.
	ErrTransaccionBaremacionNoAplicada = errors.New("bolsa: transaccion no aplicada acreditada")

	// ErrResultadoTransaccionalBaremacionIndeterminado significa que el COMMIT
	// pudo haber surtido efecto. No equivale a rollback ni concede un reintento.
	ErrResultadoTransaccionalBaremacionIndeterminado = errors.New("bolsa: resultado transaccional indeterminado")

	// ErrReconciliacionTransaccionalBaremacionRequerida permite enrutar el fallo
	// a una cola de reconciliacion sin inspeccionar textos de error.
	ErrReconciliacionTransaccionalBaremacionRequerida = errors.New("bolsa: reconciliacion transaccional requerida")

	// ErrSerializacionResultadoTransaccionalBaremacionProhibida evita que las
	// referencias y sellos probatorios crucen accidentalmente logs o DTO.
	ErrSerializacionResultadoTransaccionalBaremacionProhibida = errors.New("bolsa: serializacion generica de resultado transaccional prohibida")

	// ErrVerificadorNoAplicacionBaremacionRequerido impide convertir una mera
	// declaracion con forma valida en una prueba de no aplicacion.
	ErrVerificadorNoAplicacionBaremacionRequerido = errors.New("bolsa: verificador de no aplicacion requerido")

	// ErrEvidenciaNoAplicacionBaremacionNoVerificada no envuelve el fallo
	// tecnico del verificador para evitar conservar datos o credenciales.
	ErrEvidenciaNoAplicacionBaremacionNoVerificada = errors.New("bolsa: evidencia de no aplicacion no verificada")

	ErrContextoVerificacionNoAplicacionBaremacionInvalido = errors.New("bolsa: contexto de verificacion de no aplicacion invalido")
)
var ErrSerializacionConfirmacionNominalBaremacionV2Prohibida = errors.New(
	"bolsa: serializacion generica de confirmacion nominal V2 prohibida",
)
var ErrSerializacionConfirmacionNominalBaremacionV3Prohibida = errors.New(
	"bolsa: serializacion generica de confirmacion nominal V3 prohibida",
)
```

### Funciones

```go
func CalcularHuellaSnapshotResolucionIdentidadInternaEstableBaremacion(
	ambito SolicitudResolverSeudonimoSujetoBaremacion,
	seudonimo SeudonimoSujetoBaremacionHMAC,
	snapshotRef string,
	revision uint64,
	ancla []byte,
) (string, error)
```

CalcularHuellaSnapshotResolucionIdentidadInternaEstableBaremacion fija
HuellaSHA256 = SHA-256(esquema + ambito + seudonimo esperado + snapshotRef +
revision + ancla binaria). La huella compromete el ancla sin publicarla.

```go
func CamposRequeridosOperacionBaremacion(accion AccionOperacionBaremacion) ([]string, bool)
```

CamposRequeridosOperacionBaremacion permite configurar politicas sin
duplicar cadenas. Devuelve una copia y nunca una concesion.

```go
func ComprobacionesFirmaObligatorias() []string
func ConstituirFirmaDecisionConfiable(
	contenido dominiobolsa.ContenidoDecisionTecnica,
	politica PoliticaFirmaBaremacion,
	artefacto ArtefactoFirma,
	validacionInicial ValidacionFirmaServidor,
	sello *SelloTiempoFirma,
	validacionTrasSello *ValidacionFirmaServidor,
	aumento *ResultadoAumentoFirma,
	validacionFinal ValidacionFirmaServidor,
	documentoCustodiado DocumentoFirmadoCustodiado,
	manifiesto ManifiestoProbatorioBaremacion,
) (dominiobolsa.FirmaDecisionTecnica, error)
```

ConstituirFirmaDecisionConfiable es el unico ensamblador recomendado. Exige
firma interactiva, validacion servidor concluyente y las capas marcadas por
la politica exacta. Si hay aumento, exige validacion posterior de sus bytes.

```go
func ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
	ctx context.Context,
	solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	fronteraIdentidad FronteraIdentidadInternaEstableIdempotenciaBaremacion,
	productor ProductorTestimonioAtomicoIdempotenciaBaremacion,
	verificador VerificadorIndependienteTestimonioIdempotenciaBaremacion,
	raiz VerificadorIndependienteTestimonioIdempotenciaBaremacion,
	consumidor ConsumidorEfimeroProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion,
) error
```

ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion
es el unico camino publico del testimonio nominal. Resuelve la identidad
una sola vez y usa copias internas del mismo lote inmutable para productor,
verificador independiente y raiz. El consumidor recibe solo una vista
completa y efimera; no existe producto persistible ni puente reutilizable.

La clave cliente tiene propietario compartido y queda revocada al salir,
tambien ante error o panico. Como todas las dependencias siguen
siendo argumentos del llamador, un retorno nil NO concede autoridad,
CAS ni efectos. Se exige diversidad de tipo concreto entre productor,
verificador y raiz; es solo una barrera nominal y no acredita procesos,
operadores ni claves fisicas independientes. Esa acreditacion de DEC-048/049
sigue pendiente. El flujo permanece NO-GO hasta que el servicio privado de
aplicacion fije la composicion, verifique historicamente motivo/material y
persista de forma atomica.

```go
func ReferenciaOpacaLlamamientoValida(valor string) bool
```

ReferenciaOpacaLlamamientoValida evita documentos personales evidentes,
comodines, controles, espacios no canonicos y texto Unicode ambiguo.
No pretende sustituir el emisor criptograficamente aleatorio de referencias.

```go
func VisitarMaterialCanonicoAtestacionResolucionIdentidadInternaEstableBaremacion(
	ambito SolicitudResolverSeudonimoSujetoBaremacion,
	seudonimo SeudonimoSujetoBaremacionHMAC,
	snapshotRef string,
	revision uint64,
	huellaSHA256 string,
	visita func(MaterialCanonicoEfimeroBaremacion) error,
) error
```

VisitarMaterialCanonicoAtestacionResolucionIdentidadInternaEstableBaremacion
liga la evidencia de la frontera al ambito, seudonimo esperado y snapshot.
El hash del snapshot compromete la relacion interna; el ancla clara nunca
forma parte del testimonio ni de este material compartible. La carga
propietaria se borra al volver de la visita.

```go
func VisitarMaterialCanonicoParaDerivarIndiceIdempotenciaBaremacion(
	solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	versionPrincipal uint16,
	generacionPrincipal uint32,
	clavePrincipalRef, valorPrincipalHMAC string,
	claveClienteEfimera []byte,
	visita func(MaterialCanonicoEfimeroBaremacion) error,
) error
```

VisitarMaterialCanonicoParaDerivarIndiceIdempotenciaBaremacion fija indice
= HMAC(llavero_indice[g], esquema+politica+despliegue+modulo+accion+
principal_estable+clave_cliente). La clave solo puede proceder de la fuente
efimera del lote y debe coincidir con la solicitud; nunca se obtiene por un
getter publico. La preimagen propietaria se borra al volver del callback.

```go
func VisitarMaterialCanonicoParaDerivarPrincipalEstableBaremacion(
	s SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	identidadInternaEstableEfimera []byte,
	visita func(MaterialCanonicoEfimeroBaremacion) error,
) error
```

VisitarMaterialCanonicoParaDerivarPrincipalEstableBaremacion
fija la formula principal = HMAC(llavero_principal[g],
esquema+politica+identidad_interna_estable). La identidad es un
identificador binario opaco de 32 bytes que la frontera privada entrega solo
durante la llamada: nunca es DNI, seudonimo rotatorio, referencia libre,
DTO, contexto, log ni dato persistible. La clave cliente tampoco forma esta
preimagen.

El material visitado sigue siendo sensible. Su propietario interno se borra
al terminar el callback. El futuro aislamiento reforzado de DEC-047 debe
ejecutar este limite en otro proceso si un adaptador del mismo proceso se
considera malicioso.

```go
func VisitarMaterialCanonicoParaDerivarSeudonimoSujetoBaremacion(
	s SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	identidadInternaEstableEfimera []byte,
	visita func(MaterialCanonicoEfimeroBaremacion) error,
) error
```

VisitarMaterialCanonicoParaDerivarSeudonimoSujetoBaremacion
fija la formula seudonimo = HMAC(clave_sujeto_actual,
esquema+politica+version+identidad_interna_estable). Seudonimo y
principal parten de la misma identidad efimera pero de dominios de clave
independientes; rotar la clave de sujeto cambia solo el seudonimo.
El propietario interno se borra al volver del callback, tambien ante error o
panico; el adaptador no recibe un getter ni una carga reutilizable.

```go
func VisitarSolicitudDerivarHMACMotivoBaremacion(
	motivoClave string,
	motivo []byte,
	visita func(SolicitudDerivarHMACMotivoBaremacion) error,
) error
```

VisitarSolicitudDerivarHMACMotivoBaremacion evita almacenar el motivo en
una solicitud copiable. La solicitud y todas sus copias quedan revocadas al
volver; el adaptador criptografico debe consumir el motivo sincronicamente.

```go
func VisitarSolicitudVerificarHMACMotivoBaremacion(
	motivoClave string,
	motivo []byte,
	sello HMACMotivoBaremacion,
	visita func(SolicitudVerificarHMACMotivoBaremacion) error,
) error
```

### Tipos

```go
type AccionAuditoriaBaremacion string

const (
	AccionAuditoriaCrearBaremacion    AccionAuditoriaBaremacion = "crear_baremacion"
	AccionAuditoriaIncorporarDecision AccionAuditoriaBaremacion = "incorporar_decision_baremacion"
)
type AccionOperacionBaremacion string
```

AccionOperacionBaremacion es una accion cerrada. Una decision concedida para
una accion nunca habilita otra, aunque ambas actuen sobre el mismo recurso.

```go
const (
	AccionReservarAltaBaremacion                  AccionOperacionBaremacion = "bolsa.baremacion.alta.reservar"
	AccionConfirmarAltaBaremacion                 AccionOperacionBaremacion = "bolsa.baremacion.alta.confirmar"
	AccionAbandonarAltaBaremacion                 AccionOperacionBaremacion = "bolsa.baremacion.alta.abandonar"
	AccionReservarDecisionBaremacion              AccionOperacionBaremacion = "bolsa.baremacion.decision.reservar"
	AccionPrevalidarArchivoProbatorioBaremacion   AccionOperacionBaremacion = "bolsa.baremacion.archivo.prevalidar"
	AccionConfirmarDecisionBaremacion             AccionOperacionBaremacion = "bolsa.baremacion.decision.confirmar"
	AccionAdoptarDecisionInicialBaremacion        AccionOperacionBaremacion = "bolsa.baremacion.decision.inicial.adoptar"
	AccionRectificarDecisionBaremacion            AccionOperacionBaremacion = "bolsa.baremacion.decision.rectificar"
	AccionRevocarDecisionBaremacion               AccionOperacionBaremacion = "bolsa.baremacion.decision.revocar"
	AccionRehabilitarDecisionBaremacion           AccionOperacionBaremacion = "bolsa.baremacion.decision.rehabilitar"
	AccionAbandonarDecisionBaremacion             AccionOperacionBaremacion = "bolsa.baremacion.decision.abandonar"
	AccionConsultarBaremacionVigente              AccionOperacionBaremacion = "bolsa.baremacion.vigente.consultar"
	AccionConsultarVersionBaremacion              AccionOperacionBaremacion = "bolsa.baremacion.version.consultar"
	AccionConsultarCriterioBaremacion             AccionOperacionBaremacion = "bolsa.criterio.consultar"
	AccionConsultarEvidenciaBaremacion            AccionOperacionBaremacion = "bolsa.evidencia.consultar"
	AccionConsultarRepresentacionBaremacion       AccionOperacionBaremacion = "bolsa.representacion.consultar"
	AccionCalcularPuntuacionBaremacion            AccionOperacionBaremacion = "bolsa.puntuacion.calcular"
	AccionRecuperarCalculoBaremacion              AccionOperacionBaremacion = "bolsa.puntuacion.calculo.recuperar"
	AccionConsultarPoliticaFirmaBaremacion        AccionOperacionBaremacion = "bolsa.firma.politica.consultar"
	AccionCodificarDecisionBaremacion             AccionOperacionBaremacion = "bolsa.decision.codificar"
	AccionCustodiarDecisionBaremacion             AccionOperacionBaremacion = "bolsa.decision.custodiar"
	AccionPrepararFirmaDecisionBaremacion         AccionOperacionBaremacion = "bolsa.decision.firma.preparar"
	AccionConsultarFirmaDecisionBaremacion        AccionOperacionBaremacion = "bolsa.decision.firma.consultar"
	AccionValidarFirmaDecisionBaremacion          AccionOperacionBaremacion = "bolsa.decision.firma.validar"
	AccionSellarTiempoDecisionBaremacion          AccionOperacionBaremacion = "bolsa.decision.firma.sellar_tiempo"
	AccionAumentarFirmaDecisionBaremacion         AccionOperacionBaremacion = "bolsa.decision.firma.aumentar"
	AccionRecuperarBinarioFirmadoBaremacion       AccionOperacionBaremacion = "bolsa.decision.firma.binario.recuperar"
	AccionCustodiarDocumentoFirmadoBaremacion     AccionOperacionBaremacion = "bolsa.decision.firma.documento.custodiar"
	AccionRetenerDocumentoFirmadoBaremacion       AccionOperacionBaremacion = "bolsa.decision.firma.documento.retener"
	AccionRecuperarArtefactoFirmaBaremacion       AccionOperacionBaremacion = "bolsa.decision.firma.artefacto.recuperar"
	AccionRecuperarValidacionFirmaBaremacion      AccionOperacionBaremacion = "bolsa.decision.firma.validacion.recuperar"
	AccionRecuperarSelloTiempoFirmaBaremacion     AccionOperacionBaremacion = "bolsa.decision.firma.sello_tiempo.recuperar"
	AccionRecuperarAumentoFirmaBaremacion         AccionOperacionBaremacion = "bolsa.decision.firma.aumento.recuperar"
	AccionConsultarEvidenciaTransaccionBaremacion AccionOperacionBaremacion = "bolsa.baremacion.transaccion.consultar"
)
func AccionAdopcionParaClase(clase dominiobolsa.ClaseDecisionTecnica) (AccionOperacionBaremacion, bool)
```

AccionAdopcionParaClase devuelve la unica accion positiva que puede adoptar
cada transicion del historial. No existe una accion generica que herede o
amplie permisos entre una decision ordinaria y una actuacion inspectora.

```go
type AlmacenDocumentosFirmables interface {
	Capacidades(context.Context) (puertosvec.CapacidadesAlmacenObjetos, error)
	Escribir(context.Context, puertosvec.SolicitudEscribirObjeto) (puertosvec.ResultadoOperacionObjeto, error)
	AplicarRetencion(context.Context, puertosvec.SolicitudRetenerObjeto) (puertosvec.ResultadoOperacionObjeto, error)
}
```

AlmacenDocumentosFirmables limita la custodia de decisiones firmadas a
la lista positiva de operaciones que necesita este flujo. En particular,
no concede lectura, promocion, inmovilizacion ni eliminacion de objetos.

```go
type ArchivoEvidenciasFirmaBaremacion interface {
	RecuperarArtefactoFirma(context.Context, SolicitudRecuperarArtefactoFirma) (ArtefactoFirma, error)
	RecuperarValidacionFirma(context.Context, SolicitudRecuperarValidacionFirma) (ValidacionFirmaServidor, error)
	RecuperarSelloTiempo(context.Context, SolicitudRecuperarSelloTiempo) (SelloTiempoFirma, error)
	RecuperarAumentoFirma(context.Context, SolicitudRecuperarAumentoFirma) (ResultadoAumentoFirma, error)
}
```

ArchivoEvidenciasFirmaBaremacion permite revalidar historicamente cada
capa por referencia y huella exactas, sin depender del estado actual del
proveedor.

```go
type ArrendamientoFlujoFirmaBaremacion struct {
	FlujoRef         string
	PropietarioRef   string
	SecuenciaCercado uint64
	ExpiraEn         time.Time
	Token            TokenArrendamientoFlujoFirmaBaremacion
}

func (a ArrendamientoFlujoFirmaBaremacion) Format(estado fmt.State, _ rune)

func (a ArrendamientoFlujoFirmaBaremacion) GoString() string

func (*ArrendamientoFlujoFirmaBaremacion) GobDecode([]byte) error

func (ArrendamientoFlujoFirmaBaremacion) GobEncode() ([]byte, error)

func (a ArrendamientoFlujoFirmaBaremacion) LogValue() slog.Value

func (ArrendamientoFlujoFirmaBaremacion) MarshalBinary() ([]byte, error)

func (ArrendamientoFlujoFirmaBaremacion) MarshalJSON() ([]byte, error)

func (ArrendamientoFlujoFirmaBaremacion) MarshalText() ([]byte, error)

func (ArrendamientoFlujoFirmaBaremacion) MarshalXML(*xml.Encoder, xml.StartElement) error

func (ArrendamientoFlujoFirmaBaremacion) String() string

func (*ArrendamientoFlujoFirmaBaremacion) UnmarshalBinary([]byte) error

func (*ArrendamientoFlujoFirmaBaremacion) UnmarshalJSON([]byte) error

func (*ArrendamientoFlujoFirmaBaremacion) UnmarshalText([]byte) error

func (*ArrendamientoFlujoFirmaBaremacion) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (a ArrendamientoFlujoFirmaBaremacion) Validar() error
```

Validar solo acredita la forma nominal del sobre. La autoridad procede de
verificar Token contra la huella HMAC que conserva el repositorio.

```go
type ArtefactoFirma struct {
	ProcesoRef                       string
	SolicitudRef                     string
	SujetoRef                        string
	BaremacionMeritoRef              string
	DecisionRef                      string
	VersionBaremacion                uint64
	SesionFirmaRef                   string
	EvidenciaFirmaInteractivaRef     string
	HuellaEvidenciaInteractivaSHA256 string
	DocumentoFirmable                puertosvec.ReferenciaObjetoAlmacen
	HuellaDocumentoFirmableSHA256    string
	EvidenciaCustodiaRef             string
	FirmaRef                         string
	HuellaFirmaSHA256                string
	DocumentoFirmadoRef              string
	HuellaDocumentoSHA256            string
	HuellaContenidoSHA256            string
	PoliticaFirmaRef                 string
	PoliticaFirmaVersion             int
	HuellaPoliticaFirmaSHA256        string
	FirmanteRef                      string
	PerfilFirmanteClave              string
	FirmadaEn                        time.Time
}

func (a ArtefactoFirma) Validar() error

func (a ArtefactoFirma) ValidarPara(s SolicitudPrepararFirmaInteractiva, sesion SesionFirmaInteractiva) error

func (a ArtefactoFirma) ValidarRecuperacion(s SolicitudRecuperarArtefactoFirma) error

func (a ArtefactoFirma) ValidarRevisionPAdESDe(origen ArtefactoFirma) error
```

ValidarRevisionPAdESDe exige una nueva revision fisica del mismo PDF
firmado. FirmaRef y HuellaFirmaSHA256 identifican la firma criptografica
base y permanecen estables; la referencia y la huella del contenedor PDF
tienen que cambiar al incorporar atributos PAdES no firmados.

```go
type ArtefactosCanonicosManifiestoProbatorioBaremacionV3 struct {
	ContenidoSinHuella    CargaProtegida
	RepresentacionSellada CargaProtegida
	PreimagenHMAC         CargaProtegida
}
```

ArtefactosCanonicosManifiestoProbatorioBaremacionV3 contiene las tres
representaciones binarias que el archivo probatorio durable debe conservar
y contrastar. CargaProtegida impide su serializacion accidental y entrega
siempre copias defensivas mediante Revelar.

```go
func ArtefactosCanonicosManifiestoProbatorioBaremacion(
	manifiesto ManifiestoProbatorioBaremacion,
) (ArtefactosCanonicosManifiestoProbatorioBaremacionV3, error)
```

ArtefactosCanonicosManifiestoProbatorioBaremacion reconstruye, sin aceptar
bytes aportados por el cliente, el contenido sin huella, la representacion
canonica que incluye la huella y la preimagen exacta del HMAC. De este modo
PostgreSQL puede cotejar byte a byte su archivo sin poseer claves.

```go
type AumentadorFirmaLongeva interface {
	AumentarFirma(context.Context, SolicitudAumentarFirma) (ResultadoAumentoFirma, error)
}

type AutorizacionProbatoriaBaremacion struct {
	Secuencia       uint32
	Accion          AccionOperacionBaremacion
	ClaseRecurso    ClaseRecursoOperacionBaremacion
	RecursoRef      string
	AutorizacionRef string
}
```

AutorizacionProbatoriaBaremacion conserva el permiso positivo exacto usado
por cada PEP. La secuencia impide reordenar o omitir pasos silenciosamente.

```go
func (a AutorizacionProbatoriaBaremacion) Validar() error

type BinarioFirmadoRecuperado struct {
	DocumentoFirmadoRef      string
	HuellaDocumentoSHA256    string
	MIME                     string
	Tamano                   int64
	Contenido                io.ReadCloser
	EvidenciaRecuperacionRef string
	HuellaEvidenciaSHA256    string
	RecuperadoEn             time.Time
}
```

BinarioFirmadoRecuperado transporta el PDF final como flujo. El consumidor
debe comprobar cantidad y SHA-256 al agotarlo y cerrarlo siempre.

```go
func (b BinarioFirmadoRecuperado) ValidarPara(s SolicitudRecuperarBinarioFirmado) error

type CalculadorOficialBaremacion interface {
	CalcularPuntuacionOficial(context.Context, SolicitudCalcularPuntuacionOficial) (ResultadoCalculoOficial, error)
	RecuperarCalculoOficial(context.Context, SolicitudRecuperarCalculoOficial) (ResultadoCalculoOficial, error)
}

type CargaProtegida struct {
	// Has unexported fields.
}
```

CargaProtegida copia los bytes y bloquea su serializacion o formateo.

```go
func ContenidoCanonicoManifiestoProbatorioBaremacionV3(
	manifiesto ManifiestoProbatorioBaremacion,
) (CargaProtegida, error)
```

ContenidoCanonicoManifiestoProbatorioBaremacionV3 reconstruye exactamente
los bytes usados como preimagen SHA-256 por PrepararSellado, es decir,
materialCanonico(false). Solo admite un manifiesto completo con estructura,
huella y formato coherentes; la autenticidad del HMAC se verifica en la
frontera criptografica externa.

```go
func NuevaCargaProtegida(valor []byte) (CargaProtegida, error)

func RepresentacionCanonicaConfirmacionBaremacion(s SolicitudConfirmarCambioBaremacion) (CargaProtegida, error)
```

RepresentacionCanonicaConfirmacionBaremacion cubre token, agregado exacto,
version, trazabilidad, instante y autorizacion. Por ello alterar un solo
dato exige un sello nuevo y autentico.

```go
func RepresentacionCanonicaExpedienteFlujoFirmaBaremacion(
	e ExpedienteFlujoFirmaBaremacion,
) (CargaProtegida, error)

func RepresentacionCanonicaManifiestoProbatorioBaremacion(
	manifiesto ManifiestoProbatorioBaremacion,
) (CargaProtegida, error)
```

RepresentacionCanonicaManifiestoProbatorioBaremacion reconstruye los bytes
exactos que autentica el sellador. Admite el manifiesto preparado o ya
sellado porque el propio sello nunca forma parte de la carga autenticada.
La finalidad criptografica exclusiva encierra el material funcional y evita
reutilizar el HMAC valido de otro contrato aunque comparta campos.

```go
func RepresentacionCanonicaReservaBaremacion(s SolicitudReservarCambioBaremacion) (CargaProtegida, error)
```

RepresentacionCanonicaReservaBaremacion cubre todos los datos que fijan la
reserva, salvo el propio sello. Usa longitudes binarias para que no existan
concatenaciones ambiguas.

```go
func RepresentacionCanonicaSobreProbatorioConfirmacionBaremacionV2(
	s IntentoNominalConfirmacionBaremacionV2,
) (CargaProtegida, error)
```

RepresentacionCanonicaSobreProbatorioConfirmacionBaremacionV2 liga el
identificador opaco previo al COMMIT y el indice estable con todos los
datos exactos del intento. Es material nominal del sobre probatorio,
no el fingerprint semantico de DEC-045 ni una prueba de persistencia.

```go
func RepresentacionCanonicaSobreProbatorioConfirmacionBaremacionV3(
	s IntentoNominalConfirmacionBaremacionV3,
) (CargaProtegida, error)
```

RepresentacionCanonicaSobreProbatorioConfirmacionBaremacionV3 liga el
identificador nominal con ambos contextos de autorizacion. V2 se conserva
solo para reproducir su disposicion historica y no debe usarse en producto.

```go
func (c CargaProtegida) Format(estado fmt.State, _ rune)

func (CargaProtegida) GoString() string

func (CargaProtegida) MarshalJSON() ([]byte, error)

func (CargaProtegida) MarshalText() ([]byte, error)

func (c CargaProtegida) Revelar() []byte

func (CargaProtegida) String() string

func (c CargaProtegida) Tamano() int

func (c CargaProtegida) Validar() error

type CatalogoPoliticasFirmaBaremacion interface {
	ObtenerPoliticaFirma(context.Context, SolicitudObtenerPoliticaFirma) (PoliticaFirmaBaremacion, error)
}

type CatalogoPublico struct {
	Referencia string                   `json:"referencia"`
	Version    int                      `json:"version"`
	Entradas   []EntradaCatalogoPublico `json:"entradas"`
}

type ClaseCambioBaremacion string

const (
	ClaseCambioAltaBaremacion     ClaseCambioBaremacion = "alta"
	ClaseCambioIncorporarDecision ClaseCambioBaremacion = "incorporar_decision"
)
func (c ClaseCambioBaremacion) Valida() bool

type ClaseRecursoOperacionBaremacion string
```

ClaseRecursoOperacionBaremacion impide reutilizar accidentalmente una
capacidad entre clases con identificadores de aspecto similar.

```go
const (
	ClaseRecursoBaremacion       ClaseRecursoOperacionBaremacion = "baremacion"
	ClaseRecursoProceso          ClaseRecursoOperacionBaremacion = "proceso"
	ClaseRecursoEvidencia        ClaseRecursoOperacionBaremacion = "evidencia"
	ClaseRecursoRepresentacion   ClaseRecursoOperacionBaremacion = "representacion"
	ClaseRecursoCalculo          ClaseRecursoOperacionBaremacion = "calculo"
	ClaseRecursoPoliticaFirma    ClaseRecursoOperacionBaremacion = "politica_firma"
	ClaseRecursoDecision         ClaseRecursoOperacionBaremacion = "decision"
	ClaseRecursoSesionFirma      ClaseRecursoOperacionBaremacion = "sesion_firma"
	ClaseRecursoArtefactoFirma   ClaseRecursoOperacionBaremacion = "artefacto_firma"
	ClaseRecursoValidacionFirma  ClaseRecursoOperacionBaremacion = "validacion_firma"
	ClaseRecursoSelloTiempo      ClaseRecursoOperacionBaremacion = "sello_tiempo"
	ClaseRecursoAumentoFirma     ClaseRecursoOperacionBaremacion = "aumento_firma"
	ClaseRecursoDocumentoFirmado ClaseRecursoOperacionBaremacion = "documento_firmado"
	ClaseRecursoTransaccion      ClaseRecursoOperacionBaremacion = "transaccion"
)
func ClaseRecursoRequeridaOperacionBaremacion(
	accion AccionOperacionBaremacion,
) (ClaseRecursoOperacionBaremacion, bool)
```

ClaseRecursoRequeridaOperacionBaremacion expone la clase exacta ligada a
la accion para que el PEP y los adaptadores construyan el mismo recurso
cerrado.

```go
type ClaveClasificacionDocumentoBaremacion string
```

ClaveClasificacionDocumentoBaremacion referencia un catalogo administrable;
no fija clasificaciones en codigo ni acepta texto libre en la intencion.

```go
func (c ClaveClasificacionDocumentoBaremacion) Valida() bool

type ClaveClienteIdempotenciaBaremacion struct {
	// Has unexported fields.
}
```

ClaveClienteIdempotenciaBaremacion admite solo UUIDv4 canonico lowercase
o base64url sin relleno que decodifique 32..64 bytes no textuales y con
diversidad minima. La forma reduce DNI/NIE, correo, rutas y texto humano;
la entropia real debe generarla el cliente con CSPRNG.

```go
func NuevaClaveClienteIdempotenciaBaremacion(valor string) (ClaveClienteIdempotenciaBaremacion, error)

func (c ClaveClienteIdempotenciaBaremacion) Format(estado fmt.State, _ rune)

func (ClaveClienteIdempotenciaBaremacion) GoString() string

func (c ClaveClienteIdempotenciaBaremacion) LogValue() slog.Value

func (ClaveClienteIdempotenciaBaremacion) MarshalBinary() ([]byte, error)

func (ClaveClienteIdempotenciaBaremacion) MarshalJSON() ([]byte, error)

func (ClaveClienteIdempotenciaBaremacion) MarshalText() ([]byte, error)

func (ClaveClienteIdempotenciaBaremacion) String() string

type ClaveFormatoDocumentoBaremacion string
```

ClaveFormatoDocumentoBaremacion selecciona una entrada administrable del
catalogo historico de formatos, no un tipo fijado al compilar el nucleo.

```go
func (c ClaveFormatoDocumentoBaremacion) Valida() bool

type CodificacionCanonicaDecision struct {
	Carga                       CargaProtegida
	ProcesoRef                  string
	SolicitudRef                string
	SujetoRef                   string
	BaremacionMeritoRef         string
	DecisionRef                 string
	VersionBaremacion           uint64
	PrincipalRef                string
	PerfilActorClave            string
	AutorizacionDecisionRef     string
	AutorizacionCodificacionRef string
	FinalidadClave              string
	CorrelacionRef              string
	FormatoClave                string
	MIME                        string
	HuellaContenidoSHA256       string
	HuellaDocumentoSHA256       string
	VersionCodificador          string
}

func (c CodificacionCanonicaDecision) Validar() error

func (c CodificacionCanonicaDecision) ValidarPara(s SolicitudCodificarDecisionCanonica) error

type CodificadorCanonicoDecision interface {
	CodificarDecision(context.Context, SolicitudCodificarDecisionCanonica) (CodificacionCanonicaDecision, error)
}

type ComprobacionFirma struct {
	Clave                 string
	Estado                EstadoComprobacionFirma
	EvidenciaRef          string
	HuellaEvidenciaSHA256 string
}

func (c ComprobacionFirma) Validar() error

type ConsultaConvocatoriasPublicas interface {
	BuscarPublicadas(context.Context, FiltroConvocatoriasPublicas) (PaginaConvocatorias, error)
	ObtenerPublicada(context.Context, string) (DetalleConvocatoria, error)
}
```

ConsultaConvocatoriasPublicas es el puerto autoritativo de lectura del mismo
agregado Convocatoria que consumen solicitudes y baremación. Un adaptador
PostgreSQL u Oracle puede sustituir al fichero local sin alterar este
contrato.

```go
type ConsultaFirmaInteractiva struct {
	SesionRef             string
	Estado                EstadoSesionFirmaInteractiva
	Artefacto             *ArtefactoFirma
	EvidenciaConsultaRef  string
	HuellaEvidenciaSHA256 string
	ConsultadaEn          time.Time
}

func (c ConsultaFirmaInteractiva) Clonar() (ConsultaFirmaInteractiva, error)

func (c ConsultaFirmaInteractiva) Validar() error

func (c ConsultaFirmaInteractiva) ValidarPara(s SolicitudConsultarFirmaInteractiva) error

type ConsumidorClaveClienteLoteIdempotenciaBaremacion interface {
	ConsumirClaveClienteLoteIdempotenciaBaremacion(context.Context, []byte) error
}
```

ConsumidorClaveClienteLoteIdempotenciaBaremacion pertenece exclusivamente al
adaptador HSM/KMS. La fuente se crea dentro de la fabrica, permite una sola
entrega y se destruye al terminar; el llamador nunca recibe la fuente.

```go
type ConsumidorEfimeroProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion interface {
	ConsumirProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion(
		context.Context,
		SolicitudTestimonioAtomicoIdempotenciaBaremacion,
		VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion,
	) error
}
```

ConsumidorEfimeroProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion
es un punto de copia estructural, no un repositorio, CAS ni puerto de
efecto. El servicio de aplicacion pendiente debe implementarlo con un tipo
privado y limitarlo a crear su capacidad privada tras la reverificacion
independiente.

```go
type ConsumidorIdentidadInternaEstableIdempotenciaBaremacion interface {
	ConsumirIdentidadInternaEstableIdempotenciaBaremacion(context.Context, []byte) error
}
```

ConsumidorIdentidadInternaEstableIdempotenciaBaremacion solo puede recibir
el identificador interno opaco dentro de un callback sincrono. El valor no
debe copiarse a DTO, contexto, registro ni persistencia.

```go
type ContextoConsultaBaremacion = ContextoOperacionBaremacion
```

ContextoConsultaBaremacion mantiene el nombre semantico sin crear una via de
acceso menos exigente.

```go
type ContextoOperacionBaremacion struct {
	// Has unexported fields.
}
```

ContextoOperacionBaremacion es una capacidad opaca e inmutable. Su valor
cero deniega y no existe un literal publico que pueda rellenar sus datos.

```go
func NuevaAutorizacionOperacionAlmacenBaremacion(
	decision dominiovec.DecisionAutorizacion,
	recurso dominiovec.RecursoAutorizable,
	vinculo VinculoAutenticacionBaremacion,
	instante time.Time,
) (ContextoOperacionBaremacion, error)
```

NuevaAutorizacionOperacionAlmacenBaremacion es la unica variante que admite
las tres acciones de almacen del modulo. El recurso debe ser exactamente el
que el servidor entrego al PDP, incluidos todos los vinculos tecnicos. Se
conserva una copia defensiva para impedir que esos atributos se reconstruyan
o amplien despues de emitir la decision.

```go
func NuevaAutorizacionOperacionBaremacion(
	decision dominiovec.DecisionAutorizacion,
	vinculo VinculoAutenticacionBaremacion,
	instante time.Time,
) (ContextoOperacionBaremacion, error)
```

NuevaAutorizacionOperacionBaremacion deriva la capacidad exclusivamente de
una decision positiva y vigente. Las obligaciones se rechazan porque este
contrato no implementa ninguna de forma implicita. La lista de campos debe
coincidir exactamente con la definida para la accion.

```go
func (c ContextoOperacionBaremacion) CoincideExactamenteCon(
	otro ContextoOperacionBaremacion,
) bool
```

CoincideExactamenteCon distingue un reintento de la reutilizacion de una
capacidad para otra decision o efecto. La huella reforzada incluye todos los
campos de la decision y el vinculo V1; el sujeto se cruza aparte porque es
una relacion de negocio resuelta por el servidor.

```go
func (c ContextoOperacionBaremacion) CrearContextoAlmacenCustodiarDecision(
	vinculos puertosvec.VinculosOperacionAlmacen,
) (puertosvec.ContextoOperacionAlmacen, error)
```

CrearContextoAlmacenCustodiarDecision es el unico puente desde la decision
positiva de custodia canonica a una escritura tecnica. Los vinculos deben
coincidir con los atributos que ya evaluo el PDP.

```go
func (c ContextoOperacionBaremacion) CrearContextoAlmacenCustodiarDocumentoFirmado(
	vinculos puertosvec.VinculosOperacionAlmacen,
) (puertosvec.ContextoOperacionAlmacen, error)
```

CrearContextoAlmacenCustodiarDocumentoFirmado deriva exclusivamente la
escritura de la copia institucional firmada; nunca habilita retencion,
lectura ni eliminacion.

```go
func (c ContextoOperacionBaremacion) CrearContextoAlmacenRetenerDocumentoFirmado(
	vinculos puertosvec.VinculosOperacionAlmacen,
) (puertosvec.ContextoOperacionAlmacen, error)
```

CrearContextoAlmacenRetenerDocumentoFirmado deriva solo la retencion de la
referencia y version exactas incluidas en el recurso evaluado.

```go
func (c ContextoOperacionBaremacion) EsNulo() bool
```

EsNulo distingue la ausencia contractual de una capacidad invalida o ya no
vigente sin exponer ninguno de sus datos internos. Es la unica comprobacion
admisible para campos que deben estar exactamente ausentes.

```go
func (c ContextoOperacionBaremacion) EvidenciaUsoAutorizacion() (
	puertosvec.EvidenciaUsoDecisionAutorizacion,
	error,
)
```

EvidenciaUsoAutorizacion entrega al adaptador duradero la capacidad opaca
que debe revalidar y consumir en la misma transaccion que el efecto.
La copia conserva su inmutabilidad: no puede serializarse ni reconstruirse.

```go
func (c ContextoOperacionBaremacion) Format(estado fmt.State, _ rune)

func (ContextoOperacionBaremacion) GoString() string

func (ContextoOperacionBaremacion) MarshalBinary() ([]byte, error)

func (ContextoOperacionBaremacion) MarshalJSON() ([]byte, error)

func (ContextoOperacionBaremacion) MarshalText() ([]byte, error)

func (c ContextoOperacionBaremacion) MismoVinculoAutenticacionQue(
	otro ContextoOperacionBaremacion,
) bool
```

MismoVinculoAutenticacionQue exige la misma sesion, controles y documento
de actor V1. Las decisiones y acciones pueden ser distintas —por ejemplo,
reservar y confirmar—, pero nunca se mezclan actores o sesiones.

```go
func (c ContextoOperacionBaremacion) Proyeccion() ProyeccionAutorizacionBaremacion

func (ContextoOperacionBaremacion) String() string

func (c ContextoOperacionBaremacion) Validar() error

func (c ContextoOperacionBaremacion) ValidarPara(
	accion AccionOperacionBaremacion,
	clase ClaseRecursoOperacionBaremacion,
	recursoRef string,
) error

func (c ContextoOperacionBaremacion) ValidarVigentePara(
	accion AccionOperacionBaremacion,
	clase ClaseRecursoOperacionBaremacion,
	recursoRef string,
	instante time.Time,
) error

type ContextoOperacionFirma struct {
	ContextoOperacionBaremacion
	OperacionRef string
}

func (c ContextoOperacionFirma) Validar() error

type CreadorVinculoAutenticacionActor interface {
	Crear(
		context.Context,
		dominiovec.SolicitudRevalidacionAutenticacionActorV1,
		dominiovec.ContextoActor,
	) (dominiovec.VinculoAutenticacionActorV1, error)
}
```

CreadorVinculoAutenticacionActor revalida sesion, cuenta, superficie y
garantia mediante el servicio del nucleo. La solicitud externa solo aporta
dos referencias opacas; nunca puede rellenar el bloque de hechos resultante.

```go
type CriptografiaMotivoBaremacion interface {
	DerivarHMACMotivoBaremacion(
		context.Context,
		SolicitudDerivarHMACMotivoBaremacion,
	) (HMACMotivoBaremacion, error)
	VerificarHMACMotivoBaremacion(
		context.Context,
		SolicitudVerificarHMACMotivoBaremacion,
	) error
}
```

CriptografiaMotivoBaremacion impide aceptar una cadena hexadecimal como
prueba del motivo. La derivacion y verificacion usan un dominio y llavero
distintos de sujeto, principal, indice e intencion.

```go
type CriterioBaremacionConfiable struct {
	Referencia              dominiobolsa.ReferenciaCriterio
	PublicacionRef          string
	HuellaPublicacionSHA256 string
	EvidenciaConsultaRef    string
	HuellaEvidenciaSHA256   string
	ConsultadoEn            time.Time
}

func (c CriterioBaremacionConfiable) Validar() error

func (c CriterioBaremacionConfiable) ValidarPara(s SolicitudObtenerCriterioBaremacion) error

type DatosAutoritativosLlamamiento struct {
	Bolsa     dominiobolsa.BolsaConstituida
	Necesidad dominiobolsa.NecesidadCobertura
	Politica  dominiobolsa.ReferenciaPoliticaLlamamiento
	Entradas  []dominiobolsa.EntradaOrdenBolsa
}
```

DatosAutoritativosLlamamiento agrupa la version juridica exacta de todos los
insumos. Entradas procede del repositorio, nunca de la solicitud externa.
La aplicacion es quien crea la instantanea de orden y liga su contenido
mediante una huella. La huella no autentica ni constituye una firma:
esa garantia exige una atestacion criptografica en el adaptador duradero.

```go
func (d DatosAutoritativosLlamamiento) Clonar() (DatosAutoritativosLlamamiento, error)

type DatosPersistenciaEstadoProtegidoFlujoFirmaBaremacion struct {
	Esquema      string
	Algoritmo    string
	ClaveRef     string
	Nonce        []byte
	Cifrado      []byte
	HuellaSHA256 string
}

type DetalleConvocatoria struct {
	Convocatoria dominiobolsa.Convocatoria
	Catalogos    []CatalogoPublico
	Fuente       MetadatosFuenteConvocatorias
}

type DocumentoFirmableCustodiado struct {
	ProcesoRef                  string
	SolicitudRef                string
	SujetoRef                   string
	BaremacionMeritoRef         string
	DecisionRef                 string
	VersionBaremacion           uint64
	PrincipalRef                string
	PerfilActorClave            string
	AutorizacionDecisionRef     string
	AutorizacionCodificacionRef string
	AutorizacionCustodiaRef     string
	Objeto                      puertosvec.ObjetoAlmacenado
	EvidenciaCustodia           puertosvec.EvidenciaOperacionAlmacen
	FormatoClave                string
	MIME                        string
	Tamano                      int64
	HuellaContenidoSHA256       string
	HuellaDocumentoSHA256       string
	VersionCodificador          string
}

func NuevoDocumentoFirmableCustodiado(s SolicitudCustodiarDocumentoFirmable, r puertosvec.ResultadoOperacionObjeto) (DocumentoFirmableCustodiado, error)

func (d DocumentoFirmableCustodiado) Validar() error

func (d DocumentoFirmableCustodiado) ValidarPara(s SolicitudCustodiarDocumentoFirmable) error

type DocumentoFirmadoCustodiado struct {
	DocumentoFirmadoRef               string
	FirmaRef                          string
	HuellaDocumentoSHA256             string
	Objeto                            puertosvec.ObjetoAlmacenado
	EvidenciaEscritura                puertosvec.EvidenciaOperacionAlmacen
	EvidenciaRetencion                puertosvec.EvidenciaOperacionAlmacen
	EvidenciaRecuperacionRef          string
	HuellaEvidenciaRecuperacionSHA256 string
	PoliticaRetencionRef              string
	RetenidoHasta                     time.Time
}
```

DocumentoFirmadoCustodiado acredita la copia institucional del PDF final y
la retencion aplicada sobre exactamente la misma version y huella.

```go
func (d DocumentoFirmadoCustodiado) ValidarPara(
	artefacto ArtefactoFirma,
	escritura puertosvec.ResultadoOperacionObjeto,
	retencion puertosvec.ResultadoOperacionObjeto,
) error

type DominioClaveHMACBaremacion string
```

DominioClaveHMACBaremacion es un catalogo cerrado de usos criptograficos.
Una referencia distinta no demuestra una clave distinta: el verificador
autoritativo debe resolver cada alias contra HSM/KMS y acreditar claves
fisicas y politicas de uso separadas.

```go
const (
	DominioClavePrincipalBaremacion  DominioClaveHMACBaremacion = "principal"
	DominioClaveIndiceBaremacion     DominioClaveHMACBaremacion = "indice"
	DominioClaveSujetoBaremacion     DominioClaveHMACBaremacion = "sujeto"
	DominioClaveMotivoBaremacion     DominioClaveHMACBaremacion = "motivo"
	DominioClaveManifiestoBaremacion DominioClaveHMACBaremacion = "manifiesto"
	DominioClaveIntencionBaremacion  DominioClaveHMACBaremacion = "intencion"
)
func (d DominioClaveHMACBaremacion) Valido() bool

type EjecutorPasosFirmaBaremacion interface {
	EjecutarPasoFirmaBaremacion(context.Context, SolicitudEjecutarPasoFirmaBaremacion) (ResultadoEjecutarPasoFirmaBaremacion, error)
}

type EntradaCatalogoPublico struct {
	Clave       string `json:"clave"`
	Version     int    `json:"version"`
	Etiqueta    string `json:"etiqueta"`
	Descripcion string `json:"descripcion,omitempty"`
	Semantica   string `json:"semantica"`
	Orden       int    `json:"orden"`
	Publicable  bool   `json:"publicable"`
}
```

EntradaCatalogoPublico es configuración gobernada y versionada. Añadir un
valor no exige recompilar el núcleo ni la interfaz.

```go
type ErrorResultadoTransaccionalBaremacion struct {
	// Los campos exportados son envoltorios protegidos, no cadenas. Asi incluso
	// una desreferenciacion deliberada usa sus formateadores seguros. Validar
	// comprueba siempre la coherencia entre los tres.
	IdentificadorOperacion       IdentificadorOperacionTransaccionalBaremacion
	EstadoAplicacion             EstadoResultadoTransaccionalBaremacion
	PruebaNoAplicacionVerificada PruebaNoAplicacionVerificadaBaremacion
}
```

ErrorResultadoTransaccionalBaremacion representa exclusivamente un fracaso
cuyo efecto es negativo acreditado o desconocido. No conserva causa tecnica,
contexto, identidad, sesion, autorizacion ni capacidades temporales.

Un valor cero o alterado se interpreta en cerrado: podria haberse aplicado y
requiere reconciliacion. Este error nunca concede permiso para reintentar.

```go
func NuevoErrorResultadoTransaccionalIndeterminadoBaremacion(
	identificador IdentificadorOperacionTransaccionalBaremacion,
) (*ErrorResultadoTransaccionalBaremacion, error)

func NuevoErrorTransaccionBaremacionNoAplicada(
	prueba PruebaNoAplicacionVerificadaBaremacion,
) (*ErrorResultadoTransaccionalBaremacion, error)

func (e *ErrorResultadoTransaccionalBaremacion) Clonar() (*ErrorResultadoTransaccionalBaremacion, error)

func (e *ErrorResultadoTransaccionalBaremacion) Error() string

func (e *ErrorResultadoTransaccionalBaremacion) Estado() EstadoResultadoTransaccionalBaremacion

func (e *ErrorResultadoTransaccionalBaremacion) Format(estado fmt.State, _ rune)

func (e *ErrorResultadoTransaccionalBaremacion) GoString() string

func (e *ErrorResultadoTransaccionalBaremacion) Identificador() (
	IdentificadorOperacionTransaccionalBaremacion,
	error,
)
```

Identificador devuelve una copia protegida para que el reconciliador pueda
localizar la operacion sin acceder a ningun dato personal.

```go
func (e *ErrorResultadoTransaccionalBaremacion) Is(objetivo error) bool
```

Is ofrece clasificacion estable sin desenvolver causas tecnicas que pudieran
contener datos o credenciales. Un valor invalido conserva la clasificacion
mas restrictiva: indeterminado y pendiente de reconciliacion.

```go
func (e *ErrorResultadoTransaccionalBaremacion) LogValue() slog.Value

func (*ErrorResultadoTransaccionalBaremacion) MarshalBinary() ([]byte, error)

func (*ErrorResultadoTransaccionalBaremacion) MarshalJSON() ([]byte, error)

func (*ErrorResultadoTransaccionalBaremacion) MarshalText() ([]byte, error)

func (e *ErrorResultadoTransaccionalBaremacion) NoAplicadaVerificada() bool
```

NoAplicadaVerificada solo devuelve true ante una prueba validada por
la frontera de confianza y enlazada a la misma operacion. Ausencia,
manipulacion o estado desconocido devuelven false.

```go
func (e *ErrorResultadoTransaccionalBaremacion) PruebaNoAplicacion() (
	PruebaNoAplicacionVerificadaBaremacion,
	bool,
)

func (e *ErrorResultadoTransaccionalBaremacion) RequiereReconciliacion() bool
```

RequiereReconciliacion falla en cerrado: cualquier valor invalido se trata
como posiblemente aplicado.

```go
func (e *ErrorResultadoTransaccionalBaremacion) String() string

func (*ErrorResultadoTransaccionalBaremacion) UnmarshalBinary([]byte) error

func (*ErrorResultadoTransaccionalBaremacion) UnmarshalJSON([]byte) error

func (*ErrorResultadoTransaccionalBaremacion) UnmarshalText([]byte) error

func (e *ErrorResultadoTransaccionalBaremacion) Validar() error

type EsquemaMaterialEstableBaremacion string
```

EsquemaMaterialEstableBaremacion cierra los artefactos admitidos por la
intencion. Las constantes V2 son reservas de contrato: los productores y
verificadores V2 aun deben existir antes de abrir este flujo. Manifiesto V1,
que incorpora autorizaciones efimeras, queda explicitamente excluido.

```go
type EstadoComprobacionFirma string

const (
	EstadoComprobacionSuperada      EstadoComprobacionFirma = "superada"
	EstadoComprobacionNoSuperada    EstadoComprobacionFirma = "no_superada"
	EstadoComprobacionIndeterminada EstadoComprobacionFirma = "indeterminada"
)
func (e EstadoComprobacionFirma) Valido() bool

type EstadoDisponibilidadObjetoBaremacion string
```

EstadoDisponibilidadObjetoBaremacion prueba que el objeto incorporado sigue
disponible. El valor cero, eliminado, pendiente o desconocido falla cerrado.

```go
const EstadoDisponibilidadObjetoActivoNoEliminado EstadoDisponibilidadObjetoBaremacion = "activo_no_eliminado"
func (e EstadoDisponibilidadObjetoBaremacion) Valido() bool

type EstadoEventoOutboxBaremacion string

const EstadoEventoOutboxBaremacionPendiente EstadoEventoOutboxBaremacion = "pendiente"
type EstadoExpedienteFlujoFirmaBaremacion string

const (
	EstadoExpedienteFirmaPreparando           EstadoExpedienteFlujoFirmaBaremacion = "preparando"
	EstadoExpedienteFirmaPendienteInteraccion EstadoExpedienteFlujoFirmaBaremacion = "pendiente_interaccion"
	EstadoExpedienteFirmaFinalizando          EstadoExpedienteFlujoFirmaBaremacion = "finalizando"
	EstadoExpedienteFirmaCompletado           EstadoExpedienteFlujoFirmaBaremacion = "completado"
)
func (e EstadoExpedienteFlujoFirmaBaremacion) Valido() bool

type EstadoInmovilizacionObjetoBaremacion string
```

EstadoInmovilizacionObjetoBaremacion evita que el valor cero se interprete
como false. Ambos estados son materiales y deben proceder del recibo V2.

```go
const (
	EstadoInmovilizacionNoAplicada EstadoInmovilizacionObjetoBaremacion = "no_aplicada"
	EstadoInmovilizacionAplicada   EstadoInmovilizacionObjetoBaremacion = "aplicada"
)
func (e EstadoInmovilizacionObjetoBaremacion) Valido() bool

type EstadoOperacionIdempotenteBaremacion string
```

EstadoOperacionIdempotenteBaremacion es un catalogo cerrado. Ausente no
es una fila persistida: es la proyeccion fail-closed de una busqueda sin
resultado. En curso nunca autoriza a repetir efectos ya iniciados.

```go
const (
	EstadoOperacionIdempotenteAusente    EstadoOperacionIdempotenteBaremacion = "ausente"
	EstadoOperacionIdempotenteEnCurso    EstadoOperacionIdempotenteBaremacion = "en_curso"
	EstadoOperacionIdempotenteConfirmada EstadoOperacionIdempotenteBaremacion = "confirmada"
)
func (e EstadoOperacionIdempotenteBaremacion) Valido() bool

type EstadoPlanFirmaDurableBaremacion string

const EstadoPlanFirmaDurableCompletado EstadoPlanFirmaDurableBaremacion = "completado"
func (e EstadoPlanFirmaDurableBaremacion) Valido() bool

type EstadoProtegidoFlujoFirmaBaremacion struct {
	// Has unexported fields.
}
```

EstadoProtegidoFlujoFirmaBaremacion contiene exclusivamente un sobre AEAD.
Nunca contiene una autorizacion reconstruible en claro. Los adaptadores
duraderos deben usar DatosPersistencia e ImportarEstadoProtegido de forma
deliberada; los codificadores genericos fallan cerrados.

```go
func ImportarEstadoProtegidoFlujoFirmaBaremacion(
	datos DatosPersistenciaEstadoProtegidoFlujoFirmaBaremacion,
) (EstadoProtegidoFlujoFirmaBaremacion, error)

func NuevoEstadoProtegidoFlujoFirmaBaremacion(
	algoritmo, claveRef string,
	nonce, cifrado []byte,
) (EstadoProtegidoFlujoFirmaBaremacion, error)

func (e EstadoProtegidoFlujoFirmaBaremacion) Clonar() (EstadoProtegidoFlujoFirmaBaremacion, error)

func (e EstadoProtegidoFlujoFirmaBaremacion) DatosPersistencia() (
	DatosPersistenciaEstadoProtegidoFlujoFirmaBaremacion,
	error,
)

func (e EstadoProtegidoFlujoFirmaBaremacion) Format(estado fmt.State, _ rune)

func (e EstadoProtegidoFlujoFirmaBaremacion) GoString() string

func (EstadoProtegidoFlujoFirmaBaremacion) MarshalBinary() ([]byte, error)

func (EstadoProtegidoFlujoFirmaBaremacion) MarshalJSON() ([]byte, error)

func (EstadoProtegidoFlujoFirmaBaremacion) MarshalText() ([]byte, error)

func (EstadoProtegidoFlujoFirmaBaremacion) String() string

func (e EstadoProtegidoFlujoFirmaBaremacion) Validar() error

type EstadoPuntoControlFirmaBaremacion string

const (
	EstadoPuntoControlFirmaDeclarado  EstadoPuntoControlFirmaBaremacion = "declarado"
	EstadoPuntoControlFirmaCompletado EstadoPuntoControlFirmaBaremacion = "completado"
)
type EstadoResultadoTransaccionalBaremacion string
```

EstadoResultadoTransaccionalBaremacion es un catalogo cerrado. No existe un
estado "aplicada": una operacion aplicada debe recuperarse como el resultado
ordinario completo, no fabricarse desde este contrato de error.

```go
const (
	EstadoResultadoTransaccionalNoAplicadaVerificada  EstadoResultadoTransaccionalBaremacion = "no_aplicada_verificada"
	EstadoResultadoTransaccionalPodriaHaberseAplicado EstadoResultadoTransaccionalBaremacion = "podria_haberse_aplicado"
)
type EstadoSesionFirmaInteractiva string

const (
	EstadoSesionFirmaPreparada  EstadoSesionFirmaInteractiva = "preparada"
	EstadoSesionFirmaPendiente  EstadoSesionFirmaInteractiva = "pendiente"
	EstadoSesionFirmaCompletada EstadoSesionFirmaInteractiva = "completada"
	EstadoSesionFirmaRechazada  EstadoSesionFirmaInteractiva = "rechazada"
	EstadoSesionFirmaCancelada  EstadoSesionFirmaInteractiva = "cancelada"
	EstadoSesionFirmaExpirada   EstadoSesionFirmaInteractiva = "expirada"
	EstadoSesionFirmaFallida    EstadoSesionFirmaInteractiva = "fallida"
)
func (e EstadoSesionFirmaInteractiva) Valido() bool

type EstadoValidacionFirma string

const (
	EstadoValidacionFirmaValida        EstadoValidacionFirma = "valida"
	EstadoValidacionFirmaInvalida      EstadoValidacionFirma = "invalida"
	EstadoValidacionFirmaIndeterminada EstadoValidacionFirma = "indeterminada"
)
func (e EstadoValidacionFirma) Valido() bool

type EventoOutboxBaremacion struct {
	Referencia                   string
	Secuencia                    uint64
	Tipo                         TipoEventoOutboxBaremacion
	Estado                       EstadoEventoOutboxBaremacion
	Modulo                       string
	ProcesoRef                   string
	SolicitudRef                 string
	BaremacionMeritoRef          string
	DecisionRef                  string
	ManifiestoProbatorioRef      string
	HuellaManifiestoSHA256       string
	DocumentoFirmadoRef          string
	EvidenciaCustodiaFirmadoRef  string
	EvidenciaRetencionFirmadoRef string
	SujetoRef                    string
	PrincipalRef                 string
	VersionNueva                 uint64
	HuellaNuevaSHA256            string
	AuditoriaRef                 string
	HuellaAuditoriaSHA256        string
	CorrelacionRef               string
	RegistradoEn                 time.Time
	HuellaEventoAnteriorSHA256   string
	HuellaRegistroSHA256         string
}

func (e EventoOutboxBaremacion) Validar() error

type EvidenciaBaremacionConfiable struct {
	Evidencia                  dominiobolsa.EvidenciaMerito
	Documento                  dominiovec.DocumentoLogico
	VerificacionPertenenciaRef string
	HuellaVerificacionSHA256   string
	VerificadaEn               time.Time
}

func (e EvidenciaBaremacionConfiable) Clonar() (EvidenciaBaremacionConfiable, error)

func (e EvidenciaBaremacionConfiable) Validar() error

func (e EvidenciaBaremacionConfiable) ValidarPara(s SolicitudObtenerEvidenciaBaremacion) error

type EvidenciaNoAplicacionBaremacion struct {
	// Has unexported fields.
}
```

EvidenciaNoAplicacionBaremacion es material candidato: su forma valida
no demuestra por si sola que la operacion no se aplico. El sello HMAC
debe cubrir, mediante un esquema canonico versionado, el identificador,
la consulta autoritativa y la conclusion "no aplicada".

```go
func NuevaEvidenciaNoAplicacionBaremacion(
	identificador IdentificadorOperacionTransaccionalBaremacion,
	referenciaEvidencia string,
	selloEvidenciaHMAC string,
) (EvidenciaNoAplicacionBaremacion, error)

func (e EvidenciaNoAplicacionBaremacion) Clonar() (EvidenciaNoAplicacionBaremacion, error)

func (e EvidenciaNoAplicacionBaremacion) DatosVerificacion() (
	identificador IdentificadorOperacionTransaccionalBaremacion,
	referenciaEvidencia string,
	selloEvidenciaHMAC string,
	err error,
)
```

DatosVerificacion abre la evidencia solo para su verificador explicito.

```go
func (e EvidenciaNoAplicacionBaremacion) Format(estado fmt.State, _ rune)

func (EvidenciaNoAplicacionBaremacion) GoString() string

func (EvidenciaNoAplicacionBaremacion) LogValue() slog.Value

func (EvidenciaNoAplicacionBaremacion) MarshalBinary() ([]byte, error)

func (EvidenciaNoAplicacionBaremacion) MarshalJSON() ([]byte, error)

func (EvidenciaNoAplicacionBaremacion) MarshalText() ([]byte, error)

func (EvidenciaNoAplicacionBaremacion) String() string

func (*EvidenciaNoAplicacionBaremacion) UnmarshalBinary([]byte) error

func (*EvidenciaNoAplicacionBaremacion) UnmarshalJSON([]byte) error

func (*EvidenciaNoAplicacionBaremacion) UnmarshalText([]byte) error

func (e EvidenciaNoAplicacionBaremacion) Validar() error

type EvidenciaProbatoriaBaremacion struct {
	Secuencia             uint32
	Tipo                  TipoEvidenciaProbatoriaBaremacion
	Referencia            string
	HuellaEvidenciaSHA256 string
}

func (e EvidenciaProbatoriaBaremacion) Validar() error

type EvidenciaTransaccionBaremacion struct {
	AuditoriaRef             string
	HuellaAuditoriaSHA256    string
	EventoOutboxRef          string
	HuellaEventoOutboxSHA256 string
	ConfirmadaEn             time.Time
}

func (e EvidenciaTransaccionBaremacion) Validar() error

type EvidenciaTransaccionBaremacionRecuperada struct {
	Version    VersionBaremacion
	Auditoria  RegistroAuditoriaBaremacion
	Evento     EventoOutboxBaremacion
	Evidencia  EvidenciaTransaccionBaremacion
	Manifiesto *ManifiestoProbatorioBaremacion
}

func (r EvidenciaTransaccionBaremacionRecuperada) Validar() error

func (r EvidenciaTransaccionBaremacionRecuperada) ValidarPara(s SolicitudObtenerEvidenciaTransaccionBaremacion) error

type ExpedienteFlujoFirmaBaremacion struct {
	FlujoRef               string
	Version                uint64
	IndiceIdempotenciaHMAC string
	HuellaSolicitudHMAC    string
	VinculoActorHMAC       string
	PerfilActorClave       string
	ProcesoRef             string
	SolicitudRef           string
	BaremacionMeritoRef    string
	DecisionRef            string
	Estado                 EstadoExpedienteFlujoFirmaBaremacion
	EstadoProtegido        EstadoProtegidoFlujoFirmaBaremacion
	PuntosControl          []PuntoControlFirmaBaremacion
	ProyeccionLanzamiento  *ProyeccionLanzamientoFirmaBaremacion
	Resultado              *ResultadoFinalFlujoFirmaBaremacion
	CreadoEn               time.Time
	ActualizadoEn          time.Time
	SelloEstadoHMAC        string
}
```

ExpedienteFlujoFirmaBaremacion es la saga durable. Solo persiste
referencias, huellas, recibos y un sobre AEAD; las capacidades de
autorizacion se obtienen de nuevo para cada efecto y nunca forman parte de
esta estructura.

```go
func (e ExpedienteFlujoFirmaBaremacion) Clonar() (ExpedienteFlujoFirmaBaremacion, error)

func (e ExpedienteFlujoFirmaBaremacion) IncorporarSello(sello string) (
	ExpedienteFlujoFirmaBaremacion,
	error,
)

func (e ExpedienteFlujoFirmaBaremacion) PrepararSellado() (
	ExpedienteFlujoFirmaBaremacion,
	CargaProtegida,
	error,
)

func (e ExpedienteFlujoFirmaBaremacion) Validar() error

type FiltroConvocatoriasPublicas struct {
	Texto            string
	Tipo             string
	Categoria        string
	Estado           string
	SoloPlazoAbierto bool
	Instante         time.Time
	Limite           int
	Desplazamiento   int
}
```

FiltroConvocatoriasPublicas contiene únicamente filtros públicos. Instante
es fijado por el servidor para que un cliente no pueda decidir qué plazo se
considera vigente.

```go
type FinalidadSelloBaremacion string
```

FinalidadSelloBaremacion separa criptograficamente reservas y
confirmaciones. Un sello autentico de una finalidad no vale para la otra.

```go
const (
	FinalidadSelloReservaBaremacion FinalidadSelloBaremacion = "reserva_baremacion_v1"
	// FinalidadSelloConfirmacionBaremacion queda como alias historico V1.
	// Ningun productor vigente debe usarla tras incorporar la prevalidacion.
	FinalidadSelloConfirmacionBaremacion                  FinalidadSelloBaremacion = "confirmacion_baremacion_v1"
	FinalidadSelloConfirmacionBaremacionV2                FinalidadSelloBaremacion = "confirmacion_baremacion_v2"
	FinalidadSelloSobreProbatorioConfirmacionBaremacionV2 FinalidadSelloBaremacion = "sobre_probatorio_confirmacion_baremacion_v2"
	FinalidadSelloSobreProbatorioConfirmacionBaremacionV3 FinalidadSelloBaremacion = "sobre_probatorio_confirmacion_baremacion_v3"
	FinalidadSelloManifiestoProbatorioBaremacionV2        FinalidadSelloBaremacion = "manifiesto_probatorio_baremacion_v2"
	FinalidadSelloManifiestoProbatorioBaremacionV3        FinalidadSelloBaremacion = "manifiesto_probatorio_baremacion_v3"
)
type FirmadorInteractivo interface {
	PrepararFirmaInteractiva(context.Context, SolicitudPrepararFirmaInteractiva) (SesionFirmaInteractiva, error)
	ConsultarFirmaInteractiva(context.Context, SolicitudConsultarFirmaInteractiva) (ConsultaFirmaInteractiva, error)
}

type FronteraIdentidadInternaEstableIdempotenciaBaremacion interface {
	ResolverYEntregarIdentidadInternaEstableIdempotenciaBaremacion(
		context.Context,
		SolicitudResolverSeudonimoSujetoBaremacion,
		SeudonimoSujetoBaremacionHMAC,
		ReceptorEfimeroResolucionIdentidadInternaEstableBaremacion,
	) error
}
```

FronteraIdentidadInternaEstableIdempotenciaBaremacion debe resolver de
forma atomica las referencias opacas y el seudonimo esperado exacto. Una
implementacion no puede aceptar referencias de A junto al seudonimo de B.
Debe entregar una unica instantanea y una ancla CSPRNG inmutable de 256
bits. Al seguir siendo inyectable aqui, esta frontera solo produce un
testimonio nominal; el servicio de aplicacion pendiente debe fijarla en
privado.

```go
type FronteraIdentidadesEstablesBaremacion interface {
	ResolverSeudonimoSujetoBaremacion(
		context.Context,
		SolicitudResolverSeudonimoSujetoBaremacion,
	) (SeudonimoSujetoBaremacionHMAC, error)
	VerificarSeudonimoSujetoBaremacion(
		context.Context,
		SolicitudResolverSeudonimoSujetoBaremacion,
		SeudonimoSujetoBaremacionHMAC,
	) error
}
```

FronteraIdentidadesEstablesBaremacion queda limitada al seudonimo del
sujeto. Los principales de idempotencia solo nacen en la operacion atomica
combinada; no existe resolucion ni verificacion individual susceptible de
TOCTOU.

```go
type FuenteDatosBaremacion interface {
	ObtenerCriterio(context.Context, SolicitudObtenerCriterioBaremacion) (CriterioBaremacionConfiable, error)
	ObtenerEvidencia(context.Context, SolicitudObtenerEvidenciaBaremacion) (EvidenciaBaremacionConfiable, error)
	ObtenerRepresentacion(context.Context, SolicitudObtenerRepresentacionBaremacion) (RepresentacionBaremacionConfiable, error)
}

type FuenteDatosLlamamiento interface {
	CargarDatosAutoritativosLlamamiento(context.Context, string) ([]DatosAutoritativosLlamamiento, error)
}
```

FuenteDatosLlamamiento se consulta solo despues de una concesion exacta.
Al devolver todas las coincidencias permite que la aplicacion deniegue cero
o multiples versiones sin revelar cual de ellas existe.

```go
type FuenteEfimeraClaveClienteIdempotenciaBaremacion interface {
	EntregarClaveClienteLoteIdempotenciaBaremacion(
		context.Context,
		ConsumidorClaveClienteLoteIdempotenciaBaremacion,
	) error
}

type FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion interface {
	EntregarIdentidadInternaEstableIdempotenciaBaremacion(
		context.Context,
		ConsumidorIdentidadInternaEstableIdempotenciaBaremacion,
	) error
}
```

FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion la crea la
frontera privada de identidad/HSM despues de resolver el sujeto desde el
expediente y cotejar su seudonimo actual. Entrega exactamente una identidad
binaria estable de 32 bytes. La fabrica nominal crea entregas internas
separadas y de un solo uso para productor, verificador y raiz, y borra el
valor. El futuro servicio autoritativo debe fijar la implementacion como
dependencia privada; una fuente elegida por el llamador nunca concede
autoridad.

```go
type GeneradorReferenciasFlujoFirmaBaremacion interface {
	NuevaReferenciaFlujoFirmaBaremacion() (string, error)
	NuevaReferenciaPropietarioArrendamientoFirmaBaremacion() (string, error)
	NuevaReferenciaEfectoFirmaBaremacion(PasoFlujoFirmaBaremacion) (string, error)
}

type GeneradorReferenciasLlamamiento interface {
	NuevaReferenciaInstantaneaOrdenBolsa() (string, error)
	NuevaReferenciaPropuestaLlamamiento() (string, error)
}
```

GeneradorReferenciasLlamamiento es la unica autoridad para identificadores
nuevos. Version 1 pertenece a cada instantanea nueva de referencia unica;
un futuro repositorio que versione la misma referencia debe aportar otro
puerto y CAS, no inferir una version en este caso de uso.

```go
type GeneradorReferenciasOpacasBaremacion interface {
	NuevoIDBaremacion() (string, error)
	NuevoIDDecisionTecnica() (string, error)
	NuevaReferenciaManifiestoProbatorio() (string, error)
	NuevaReferenciaCorrelacion() (string, error)
	NuevaReferenciaEfectoAlmacen() (string, error)
}

type HMACIntencionCambioBaremacion struct {
	Version      uint16
	ClaveHMACRef string
	ValorHMAC    string
}
```

HMACIntencionCambioBaremacion es el sello persistible del fingerprint
semantico estable, no del sobre probatorio rotatorio. Solo puede calcularse
despues de verificar el MotivoHMAC historico contra el motivo efimero.
Los sobres exactos de seudonimo, manifiesto y motivo se conservan en la
auditoria del intento, pero nunca deciden igualdad semantica.

```go
func (h HMACIntencionCambioBaremacion) Clonar() (HMACIntencionCambioBaremacion, error)

func (h HMACIntencionCambioBaremacion) Format(estado fmt.State, _ rune)

func (HMACIntencionCambioBaremacion) GoString() string

func (h HMACIntencionCambioBaremacion) IgualConstante(otro HMACIntencionCambioBaremacion) bool

func (h HMACIntencionCambioBaremacion) LogValue() slog.Value

func (HMACIntencionCambioBaremacion) MarshalBinary() ([]byte, error)

func (HMACIntencionCambioBaremacion) MarshalJSON() ([]byte, error)

func (HMACIntencionCambioBaremacion) MarshalText() ([]byte, error)

func (HMACIntencionCambioBaremacion) String() string

func (h HMACIntencionCambioBaremacion) Validar() error

type HMACManifiestoMaterialBaremacionV2 struct {
	Version      uint16
	ClaveHMACRef string
	ValorHMAC    string
}
```

HMACManifiestoMaterialBaremacionV2 sustituye el sobre textual generico. La
forma del valor no acredita autenticidad. Solo la composicion TCB homologada
puede confiar en la aceptacion conjunta del verificador de dominios y del
verificador material V2 que mantiene como dependencias privadas.

```go
func (h HMACManifiestoMaterialBaremacionV2) Format(estado fmt.State, _ rune)

func (HMACManifiestoMaterialBaremacionV2) GoString() string

func (h HMACManifiestoMaterialBaremacionV2) IgualConstante(
	otro HMACManifiestoMaterialBaremacionV2,
) bool

func (h HMACManifiestoMaterialBaremacionV2) LogValue() slog.Value

func (HMACManifiestoMaterialBaremacionV2) MarshalBinary() ([]byte, error)

func (HMACManifiestoMaterialBaremacionV2) MarshalJSON() ([]byte, error)

func (HMACManifiestoMaterialBaremacionV2) MarshalText() ([]byte, error)

func (HMACManifiestoMaterialBaremacionV2) String() string

func (h HMACManifiestoMaterialBaremacionV2) Validar() error

type HMACMotivoBaremacion struct {
	Version      uint16
	ClaveHMACRef string
	ValorHMAC    string
}
```

HMACMotivoBaremacion compromete el motivo exacto sin persistirlo ni exponer
una huella SHA-256 susceptible a ataques de diccionario. Su clave debe ser
exclusiva de este dominio y mantenerse en el llavero historico.

```go
func (h HMACMotivoBaremacion) Clonar() (HMACMotivoBaremacion, error)

func (h HMACMotivoBaremacion) Format(estado fmt.State, _ rune)

func (HMACMotivoBaremacion) GoString() string

func (h HMACMotivoBaremacion) IgualConstante(otro HMACMotivoBaremacion) bool

func (h HMACMotivoBaremacion) LogValue() slog.Value

func (HMACMotivoBaremacion) MarshalBinary() ([]byte, error)

func (HMACMotivoBaremacion) MarshalJSON() ([]byte, error)

func (HMACMotivoBaremacion) MarshalText() ([]byte, error)

func (HMACMotivoBaremacion) String() string

func (h HMACMotivoBaremacion) Validar() error

type IdentificadorOperacionTransaccionalBaremacion struct {
	// Has unexported fields.
}
```

IdentificadorOperacionTransaccionalBaremacion enlaza la referencia opaca
creada antes de la transaccion con un indice HMAC estable. No es una
capacidad de acceso y no debe derivarse de DNI, correo, expediente ni otras
referencias de negocio. El adaptador debe generar la referencia con un
CSPRNG.

Sus campos son privados para impedir que un DTO generico los publique. Solo
DatosReconciliacion los abre de forma explicita al adaptador reconciliador.

```go
func NuevoIdentificadorOperacionTransaccionalBaremacion(
	referenciaOpaca string,
	indiceOperacionHMAC string,
) (IdentificadorOperacionTransaccionalBaremacion, error)
```

NuevoIdentificadorOperacionTransaccionalBaremacion valida la forma opaca de
la referencia y el indice autenticado. La validacion de forma no sustituye
la obligacion del adaptador de usar aleatoriedad criptografica.

```go
func (i IdentificadorOperacionTransaccionalBaremacion) Clonar() (IdentificadorOperacionTransaccionalBaremacion, error)

func (i IdentificadorOperacionTransaccionalBaremacion) CoincideExactamenteCon(
	otro IdentificadorOperacionTransaccionalBaremacion,
) bool
```

CoincideExactamenteCon comprueba las dos partes protegidas del identificador
sin abrirlas al llamador. Un valor cero, alterado o de otro esquema nunca
coincide, aunque comparta una de las dos partes.

```go
func (i IdentificadorOperacionTransaccionalBaremacion) DatosReconciliacion() (
	referenciaOpaca string,
	indiceOperacionHMAC string,
	err error,
)
```

DatosReconciliacion es la unica apertura deliberada del identificador.
El llamador debe tratar los valores como material probatorio, no como texto
de observabilidad.

```go
func (i IdentificadorOperacionTransaccionalBaremacion) Format(estado fmt.State, _ rune)

func (IdentificadorOperacionTransaccionalBaremacion) GoString() string

func (IdentificadorOperacionTransaccionalBaremacion) LogValue() slog.Value

func (IdentificadorOperacionTransaccionalBaremacion) MarshalBinary() ([]byte, error)

func (IdentificadorOperacionTransaccionalBaremacion) MarshalJSON() ([]byte, error)

func (IdentificadorOperacionTransaccionalBaremacion) MarshalText() ([]byte, error)

func (IdentificadorOperacionTransaccionalBaremacion) String() string

func (*IdentificadorOperacionTransaccionalBaremacion) UnmarshalBinary([]byte) error

func (*IdentificadorOperacionTransaccionalBaremacion) UnmarshalJSON([]byte) error

func (*IdentificadorOperacionTransaccionalBaremacion) UnmarshalText([]byte) error

func (i IdentificadorOperacionTransaccionalBaremacion) Validar() error

type InstantaneaCatalogoClasificacionDocumentoBaremacion struct {
	CatalogoRef          string
	CatalogoVersion      uint32
	HuellaCatalogoSHA256 string
	ClasificacionClave   ClaveClasificacionDocumentoBaremacion
}
```

InstantaneaCatalogoClasificacionDocumentoBaremacion fija por referencia,
version y huella la clasificacion aplicada al documento custodiado.

```go
func (i InstantaneaCatalogoClasificacionDocumentoBaremacion) Format(estado fmt.State, _ rune)

func (InstantaneaCatalogoClasificacionDocumentoBaremacion) GoString() string

func (i InstantaneaCatalogoClasificacionDocumentoBaremacion) LogValue() slog.Value

func (InstantaneaCatalogoClasificacionDocumentoBaremacion) MarshalBinary() ([]byte, error)

func (InstantaneaCatalogoClasificacionDocumentoBaremacion) MarshalJSON() ([]byte, error)

func (InstantaneaCatalogoClasificacionDocumentoBaremacion) MarshalText() ([]byte, error)

func (InstantaneaCatalogoClasificacionDocumentoBaremacion) String() string

func (i InstantaneaCatalogoClasificacionDocumentoBaremacion) Validar() error

type InstantaneaCatalogoFormatoDocumentoBaremacion struct {
	CatalogoRef          string
	CatalogoVersion      uint32
	HuellaCatalogoSHA256 string
	FormatoClave         ClaveFormatoDocumentoBaremacion
	MIMECanonico         MIMECanonicoDocumentoBaremacion
}
```

InstantaneaCatalogoFormatoDocumentoBaremacion fija la entrada historica que
eligio el flujo. Su forma no prueba que exista: el verificador material de
la TCB debe resolver CatalogoRef+Version+Huella y cotejar clave y MIME.

```go
func (i InstantaneaCatalogoFormatoDocumentoBaremacion) Format(estado fmt.State, _ rune)

func (InstantaneaCatalogoFormatoDocumentoBaremacion) GoString() string

func (i InstantaneaCatalogoFormatoDocumentoBaremacion) LogValue() slog.Value

func (InstantaneaCatalogoFormatoDocumentoBaremacion) MarshalBinary() ([]byte, error)

func (InstantaneaCatalogoFormatoDocumentoBaremacion) MarshalJSON() ([]byte, error)

func (InstantaneaCatalogoFormatoDocumentoBaremacion) MarshalText() ([]byte, error)

func (InstantaneaCatalogoFormatoDocumentoBaremacion) String() string

func (i InstantaneaCatalogoFormatoDocumentoBaremacion) Validar() error

type IntencionCambioBaremacion struct {
	Version uint16
	Clase   ClaseCambioBaremacion

	ProcesoRef          string
	SolicitudRef        string
	SujetoSeudonimoHMAC SeudonimoSujetoBaremacionHMAC
	BaremacionMeritoRef string
	VersionBase         ReferenciaVersionBaremacion
	VersionObjetivo     uint64

	DecisionRef                          string
	NumeroDecision                       uint64
	ClaseDecision                        dominiobolsa.ClaseDecisionTecnica
	ResultadoDecision                    dominiobolsa.ResultadoDecisionTecnica
	HuellaContenidoDecisionSHA256        string
	HuellaEstadoResultanteDecisionSHA256 string

	PoliticaFirmaRef             string
	PoliticaFirmaVersion         uint32
	HuellaPoliticaFirmaSHA256    string
	EsquemaPlanFirmaDurable      EsquemaMaterialEstableBaremacion
	VersionPlanFirmaDurable      uint16
	PlanFirmaDurableRef          string
	HuellaPlanFirmaDurableSHA256 string
	EstadoPlanFirmaDurable       EstadoPlanFirmaDurableBaremacion

	DocumentoFirmableRef          string
	VersionDocumentoFirmable      string
	HuellaDocumentoFirmableSHA256 string
	FirmaRef                      string
	HuellaFirmaSHA256             string
	DocumentoFirmadoRef           string
	HuellaDocumentoFirmadoSHA256  string

	EsquemaManifiestoProbatorio      EsquemaMaterialEstableBaremacion
	VersionManifiestoProbatorio      uint16
	ManifiestoProbatorioRef          string
	HuellaManifiestoProbatorioSHA256 string
	SelloManifiestoProbatorioHMAC    HMACManifiestoMaterialBaremacionV2

	ObjetoCustodiadoRef               string
	VersionObjetoCustodiado           string
	ConectorCustodiaID                string
	ZonaCustodia                      puertosvec.ZonaAlmacen
	HuellaObjetoCustodiadoSHA256      string
	FormatoDocumento                  InstantaneaCatalogoFormatoDocumentoBaremacion
	ClasificacionDocumento            InstantaneaCatalogoClasificacionDocumentoBaremacion
	TamanoDocumentoFirmado            uint64
	EstadoInmovilizacionObjeto        EstadoInmovilizacionObjetoBaremacion
	EstadoDisponibilidadObjeto        EstadoDisponibilidadObjetoBaremacion
	EsquemaEvidenciaRecuperacion      EsquemaMaterialEstableBaremacion
	VersionEvidenciaRecuperacion      uint16
	EvidenciaRecuperacionFirmadoRef   string
	HuellaEvidenciaRecuperacionSHA256 string
	EsquemaEvidenciaCustodia          EsquemaMaterialEstableBaremacion
	VersionEvidenciaCustodia          uint16
	EvidenciaCustodiaFirmadoRef       string
	HuellaEvidenciaCustodiaSHA256     string
	EsquemaEvidenciaRetencion         EsquemaMaterialEstableBaremacion
	VersionEvidenciaRetencion         uint16
	EvidenciaRetencionFirmadoRef      string
	HuellaEvidenciaRetencionSHA256    string
	PoliticaRetencionRef              string
	PoliticaRetencionVersion          uint32
	HuellaPoliticaRetencionSHA256     string
	RetenidoHasta                     time.Time

	HuellaAgregadoObjetivoSHA256 string
	MotivoClave                  string
	MotivoHMAC                   HMACMotivoBaremacion
}
```

IntencionCambioBaremacion es la proyeccion material estable de una
incorporacion firmada. Excluye deliberadamente contexto, autenticacion,
sesion, autorizacion del intento, tokens, correlaciones, auditoria, outbox
y tiempos del intento. Motivo se compromete mediante HMAC de dominio propio
para no duplicar texto que pueda contener datos personales. Los campos V2
de manifiesto y recibos son puertas cerradas: Validar comprueba forma, pero
solo la composicion TCB con su verificador material privado puede confiar en
el contenido y en las instantaneas historicas de catalogo.

No contiene mapas, punteros ni colecciones: una copia del valor no comparte
memoria mutable con el original.

```go
func (i IntencionCambioBaremacion) Clonar() (IntencionCambioBaremacion, error)

func (i IntencionCambioBaremacion) Format(estado fmt.State, _ rune)

func (IntencionCambioBaremacion) GoString() string

func (i IntencionCambioBaremacion) LogValue() slog.Value

func (IntencionCambioBaremacion) MarshalBinary() ([]byte, error)

func (IntencionCambioBaremacion) MarshalJSON() ([]byte, error)

func (IntencionCambioBaremacion) MarshalText() ([]byte, error)

func (IntencionCambioBaremacion) String() string

func (i IntencionCambioBaremacion) Validar() error

type IntentoNominalConfirmacionBaremacionV2 struct {
	IdentificadorOperacion IdentificadorOperacionTransaccionalBaremacion
	Confirmacion           SolicitudConfirmarCambioBaremacion
}
```

IntentoNominalConfirmacionBaremacionV2 conserva exclusivamente el canonico
historico anterior a la prevalidacion de archivo. Ningun productor vigente
debe emitirlo; V3 es el unico contrato nominal habilitado para nuevo codigo.
El identificador debia existir durablemente antes de construir el sobre y su
sello cubrirlo junto al efecto.

Este tipo es nominal: solo acredita forma y permite producir el canonico.
No acredita autenticidad, preparacion durable, persistencia ni resultado de
COMMIT y no habilita ningun efecto. Se mantiene para reproducir y verificar
vectores historicos, no como ruta de compatibilidad productiva.

```go
func (s IntentoNominalConfirmacionBaremacionV2) Clonar() (IntentoNominalConfirmacionBaremacionV2, error)

func (s IntentoNominalConfirmacionBaremacionV2) Format(estado fmt.State, _ rune)

func (IntentoNominalConfirmacionBaremacionV2) GoString() string

func (IntentoNominalConfirmacionBaremacionV2) LogValue() slog.Value

func (IntentoNominalConfirmacionBaremacionV2) MarshalBinary() ([]byte, error)

func (IntentoNominalConfirmacionBaremacionV2) MarshalJSON() ([]byte, error)

func (IntentoNominalConfirmacionBaremacionV2) MarshalText() ([]byte, error)

func (IntentoNominalConfirmacionBaremacionV2) String() string

func (*IntentoNominalConfirmacionBaremacionV2) UnmarshalBinary([]byte) error

func (*IntentoNominalConfirmacionBaremacionV2) UnmarshalJSON([]byte) error

func (*IntentoNominalConfirmacionBaremacionV2) UnmarshalText([]byte) error

func (s IntentoNominalConfirmacionBaremacionV2) ValidarForma() error

type IntentoNominalConfirmacionBaremacionV3 struct {
	IdentificadorOperacion IdentificadorOperacionTransaccionalBaremacion
	Confirmacion           SolicitudConfirmarCambioBaremacion
}
```

IntentoNominalConfirmacionBaremacionV3 sustituye al sobre V2 retirado e
incorpora la autorizacion de prevalidacion dentro del canonico autenticado.
Sigue siendo un contrato nominal: no acredita persistencia ni resultado.

```go
func (s IntentoNominalConfirmacionBaremacionV3) Clonar() (IntentoNominalConfirmacionBaremacionV3, error)

func (s IntentoNominalConfirmacionBaremacionV3) Format(estado fmt.State, _ rune)

func (IntentoNominalConfirmacionBaremacionV3) GoString() string

func (IntentoNominalConfirmacionBaremacionV3) LogValue() slog.Value

func (IntentoNominalConfirmacionBaremacionV3) MarshalBinary() ([]byte, error)

func (IntentoNominalConfirmacionBaremacionV3) MarshalJSON() ([]byte, error)

func (IntentoNominalConfirmacionBaremacionV3) MarshalText() ([]byte, error)

func (IntentoNominalConfirmacionBaremacionV3) String() string

func (*IntentoNominalConfirmacionBaremacionV3) UnmarshalBinary([]byte) error

func (*IntentoNominalConfirmacionBaremacionV3) UnmarshalJSON([]byte) error

func (*IntentoNominalConfirmacionBaremacionV3) UnmarshalText([]byte) error

func (s IntentoNominalConfirmacionBaremacionV3) ValidarForma() error

type MIMECanonicoDocumentoBaremacion string
```

MIMECanonicoDocumentoBaremacion conserva el media type concreto resuelto por
el catalogo. No admite parametros, comodines ni mayusculas canonicas.

```go
func (m MIMECanonicoDocumentoBaremacion) Valido() bool

type ManifiestoProbatorioBaremacion struct {
	Esquema                   string
	Finalidad                 string
	VersionEsquema            int
	Referencia                string
	ProcesoRef                string
	SolicitudRef              string
	SujetoRef                 string
	BaremacionMeritoRef       string
	DecisionRef               string
	VersionBase               uint64
	HuellaVersionBaseSHA256   string
	Autorizaciones            []AutorizacionProbatoriaBaremacion
	Evidencias                []EvidenciaProbatoriaBaremacion
	CreadoEn                  time.Time
	HuellaManifiestoSHA256    string
	SelloManifiestoHMACSHA256 string
}
```

ManifiestoProbatorioBaremacion es el indice sellado de capacidades y
evidencias que sostienen una decision. No acepta mapas ni extensiones
libres.

```go
func (m ManifiestoProbatorioBaremacion) Clonar() ManifiestoProbatorioBaremacion

func (m ManifiestoProbatorioBaremacion) IncorporarSello(sello string) (ManifiestoProbatorioBaremacion, error)

func (m ManifiestoProbatorioBaremacion) PrepararSellado() (ManifiestoProbatorioBaremacion, CargaProtegida, error)
```

PrepararSellado calcula la huella cerrada y devuelve exactamente los bytes
que debe autenticar el sellador institucional.

```go
func (m ManifiestoProbatorioBaremacion) Validar() error

func (m ManifiestoProbatorioBaremacion) ValidarCoberturaFirmaPara(
	version ReferenciaVersionBaremacion,
	contenido dominiobolsa.ContenidoDecisionTecnica,
	firma dominiobolsa.FirmaDecisionTecnica,
) error
```

ValidarCoberturaFirmaPara prueba que el manifiesto contiene exactamente las
capacidades y recibos que sostienen la firma incorporada a la decision. No
basta con que cada entrada sea valida de forma aislada: recursos, huellas,
capas opcionales y orden deben coincidir con esa decision concreta.

```go
func (m ManifiestoProbatorioBaremacion) ValidarPara(
	version ReferenciaVersionBaremacion,
	contenido dominiobolsa.ContenidoDecisionTecnica,
) error

type MaterialCanonicoEfimeroBaremacion struct {
	// Has unexported fields.
}
```

MaterialCanonicoEfimeroBaremacion es una vista revocable de una preimagen
sensible. Todas sus copias comparten propietario: al terminar el callback
exterior quedan invalidas y VisitarBytes falla. VisitarBytes entrega una
unica copia sincrona y la borra al volver, incluso ante error o panico.

Esto elimina las copias propias controlables, pero Go no permite garantizar
zeroization absoluta frente a copias deliberadas del adaptador o del
runtime. Si ese actor esta en el modelo de amenaza, DEC-047 exige otro
proceso.

```go
func (m MaterialCanonicoEfimeroBaremacion) Format(estado fmt.State, _ rune)

func (MaterialCanonicoEfimeroBaremacion) GoString() string

func (m MaterialCanonicoEfimeroBaremacion) LogValue() slog.Value

func (MaterialCanonicoEfimeroBaremacion) MarshalBinary() ([]byte, error)

func (MaterialCanonicoEfimeroBaremacion) MarshalJSON() ([]byte, error)

func (MaterialCanonicoEfimeroBaremacion) MarshalText() ([]byte, error)

func (MaterialCanonicoEfimeroBaremacion) String() string

func (m MaterialCanonicoEfimeroBaremacion) Validar() error

func (m MaterialCanonicoEfimeroBaremacion) VisitarBytes(
	visita func([]byte) error,
) (errRetorno error)

type MetadatosFuenteConvocatorias struct {
	Revision      string    `json:"revision"`
	ActualizadaEn time.Time `json:"actualizada_en"`
	Demostracion  bool      `json:"demostracion"`
	Aviso         string    `json:"aviso,omitempty"`
}

type ModuloIdempotenciaBaremacion string

const ModuloIdempotenciaBolsa ModuloIdempotenciaBaremacion = "bolsa"
func (m ModuloIdempotenciaBaremacion) Valido() bool

type MotorElegibilidadLlamamiento interface {
	EvaluarParticipacion(context.Context, SolicitudEvaluarParticipacionLlamamiento) (dominiobolsa.EvaluacionParticipacionLlamamiento, error)
}

type PaginaConvocatorias struct {
	Convocatorias []dominiobolsa.Convocatoria
	Total         int
	Catalogos     []CatalogoPublico
	Fuente        MetadatosFuenteConvocatorias
}

type PasoFlujoFirmaBaremacion string

const (
	PasoPrepararFirmaBaremacion  PasoFlujoFirmaBaremacion = "preparar_firma"
	PasoCompletarFirmaBaremacion PasoFlujoFirmaBaremacion = "completar_firma"
	PasoCustodiarFirmaBaremacion PasoFlujoFirmaBaremacion = "custodiar_firma"
	PasoRetenerFirmaBaremacion   PasoFlujoFirmaBaremacion = "retener_firma"
	PasoReservarFirmaBaremacion  PasoFlujoFirmaBaremacion = "reservar_cambio"
	PasoConfirmarFirmaBaremacion PasoFlujoFirmaBaremacion = "confirmar_cambio"
)
func PasosFlujoFirmaBaremacion() []PasoFlujoFirmaBaremacion

func (p PasoFlujoFirmaBaremacion) Valido() bool

type PoliticaFirmaBaremacion struct {
	Referencia                      string
	Version                         int
	HuellaSHA256                    string
	FormatoFirmaClave               string
	PerfilFirmaClave                string
	AlgoritmoHuellaClave            string
	ComprobacionesObligatorias      []string
	RequiereFirmaInteractiva        bool
	RequiereValidacionServidor      bool
	RequiereSelloTiempo             bool
	PoliticaSelloTiempoRef          string
	PoliticaSelloTiempoVersion      int
	HuellaPoliticaSelloTiempoSHA256 string
	RequiereAumentoLongevidad       bool
	NivelAumentoClave               string
	PoliticaLongevidadRef           string
	PoliticaLongevidadVersion       int
	HuellaPoliticaLongevidadSHA256  string
	AprobacionRef                   string
	HuellaAprobacionSHA256          string
	VigenteDesde                    time.Time
	VigenteHasta                    time.Time
}

func (p PoliticaFirmaBaremacion) Validar() error

func (p PoliticaFirmaBaremacion) ValidarPara(s SolicitudObtenerPoliticaFirma) error

func (p PoliticaFirmaBaremacion) VigenteEn(instante time.Time) bool

type ProductorTestimonioAtomicoIdempotenciaBaremacion interface {
	ProducirTestimonioAtomicoIdempotenciaBaremacion(
		context.Context,
		SolicitudTestimonioAtomicoIdempotenciaBaremacion,
		FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion,
		FuenteEfimeraClaveClienteIdempotenciaBaremacion,
		ReceptorEfimeroTestimonioAtomicoIdempotenciaBaremacion,
	) error
}
```

ProductorTestimonioAtomicoIdempotenciaBaremacion debe hacer, en una sola
operacion del HSM/KMS, las dos instantaneas, la matriz completa y la
atestacion ligada al contenido. No se admite resolver/verificar por sujeto.

```go
type ProtectorEstadoFlujoFirmaBaremacion interface {
	ProtegerEstadoFlujoFirmaBaremacion(context.Context, CargaProtegida) (EstadoProtegidoFlujoFirmaBaremacion, error)
	DesprotegerEstadoFlujoFirmaBaremacion(context.Context, EstadoProtegidoFlujoFirmaBaremacion) (CargaProtegida, error)
}

type ProyeccionAutorizacionBaremacion struct {
	PrincipalRef        string
	SujetoRef           string
	PerfilActorClave    string
	MetodoAutenticacion dominiovec.AuthMethod
	NivelAutenticacion  dominiovec.AuthAssurance
	GarantiaMinima      dominiovec.AuthAssurance
	AutenticacionRef    string
	SesionRef           string
	SesionEmitidaEn     time.Time
	SesionValidaHasta   time.Time
	AutorizacionRef     string
	Accion              AccionOperacionBaremacion
	ClaseRecurso        ClaseRecursoOperacionBaremacion
	RecursoRef          string
	FinalidadClave      string
	CorrelacionRef      string
	CamposPermitidos    []string
	EmitidaEn           time.Time
	ValidaHasta         time.Time
}
```

ProyeccionAutorizacionBaremacion es una copia solo de lectura para construir
trazabilidad. Modificarla nunca modifica ni concede la capacidad original.

```go
type ProyeccionLanzamientoFirmaBaremacion struct {
	FlujoRef              string
	SesionFirmaRef        string
	LanzamientoRef        string
	CanalLanzamientoClave string
	PreparadaEn           time.Time
	ExpiraEn              time.Time
}
```

ProyeccionLanzamientoFirmaBaremacion no transporta bytes, URL firmada,
autorizacion ni estado del expediente. LanzamientoRef es una referencia
opaca que el adaptador interno debe resolver tras reautenticar la peticion.

```go
func (p ProyeccionLanzamientoFirmaBaremacion) Validar() error

type PruebaNoAplicacionVerificadaBaremacion struct {
	// Has unexported fields.
}
```

PruebaNoAplicacionVerificadaBaremacion es una capacidad opaca. Sus campos
privados impiden construirla fuera de este paquete sin pasar por Verificar.

```go
func VerificarEvidenciaNoAplicacionBaremacion(
	ctx context.Context,
	verificador VerificadorNoAplicacionBaremacion,
	evidencia EvidenciaNoAplicacionBaremacion,
) (PruebaNoAplicacionVerificadaBaremacion, error)
```

VerificarEvidenciaNoAplicacionBaremacion descarta deliberadamente la causa
tecnica del verificador. La causa puede registrarse en su propio limite de
seguridad, pero nunca queda incorporada al resultado transaccional.

```go
func (p PruebaNoAplicacionVerificadaBaremacion) Clonar() (PruebaNoAplicacionVerificadaBaremacion, error)

func (p PruebaNoAplicacionVerificadaBaremacion) Evidencia() (EvidenciaNoAplicacionBaremacion, error)

func (p PruebaNoAplicacionVerificadaBaremacion) Format(estado fmt.State, _ rune)

func (PruebaNoAplicacionVerificadaBaremacion) GoString() string

func (PruebaNoAplicacionVerificadaBaremacion) LogValue() slog.Value

func (PruebaNoAplicacionVerificadaBaremacion) MarshalBinary() ([]byte, error)

func (PruebaNoAplicacionVerificadaBaremacion) MarshalJSON() ([]byte, error)

func (PruebaNoAplicacionVerificadaBaremacion) MarshalText() ([]byte, error)

func (PruebaNoAplicacionVerificadaBaremacion) String() string

func (*PruebaNoAplicacionVerificadaBaremacion) UnmarshalBinary([]byte) error

func (*PruebaNoAplicacionVerificadaBaremacion) UnmarshalJSON([]byte) error

func (*PruebaNoAplicacionVerificadaBaremacion) UnmarshalText([]byte) error

func (p PruebaNoAplicacionVerificadaBaremacion) Validar() error

type PuntoControlFirmaBaremacion struct {
	Paso                  PasoFlujoFirmaBaremacion
	Estado                EstadoPuntoControlFirmaBaremacion
	EfectoRef             string
	ClaveIdempotenciaHMAC string
	ResultadoRef          string
	HuellaResultadoSHA256 string
	DeclaradoEn           time.Time
	CompletadoEn          time.Time
}

func (p PuntoControlFirmaBaremacion) Validar() error

type ReceptorEfimeroResolucionIdentidadInternaEstableBaremacion interface {
	RegistrarResolucionIdentidadInternaEstableBaremacion(
		context.Context,
		[]byte,
		string,
		uint64,
		string,
		string,
		string,
		string,
		[]byte,
	) error
}
```

ReceptorEfimeroResolucionIdentidadInternaEstableBaremacion solo vive durante
una llamada a la frontera. La instantanea identifica la version inmutable de
la relacion expediente+seudonimo esperado -> ancla interna; la atestacion
permite que la raiz independiente compruebe esa resolucion sin ver el ancla.

```go
type ReceptorEfimeroTestimonioAtomicoIdempotenciaBaremacion interface {
	InmovilizarLlaveroIdentidadesBaremacion(string, uint64, uint8, string) error
	RegistrarPrincipalEstableBaremacion(int, uint16, uint32, string, string) error
	InmovilizarLlaveroIndicesBaremacion(string, uint64, uint8, string) error
	RegistrarIndiceIdempotenciaBaremacion(int, int, uint16, uint32, string, string) error
	VisitarMaterialCanonicoParaAtestacionBaremacion(func(MaterialCanonicoEfimeroBaremacion) error) error
	RegistrarEvidenciaAtestacionBaremacion(string, string, string, uint64, string, []byte) error
}
```

ReceptorEfimeroTestimonioAtomicoIdempotenciaBaremacion es la unica via de
construccion para un adaptador externo. El receptor lo crea la fabrica, se
cierra tras una sola llamada al productor y no devuelve tipos individuales.

```go
type RecuperadorBinarioFirmado interface {
	RecuperarBinarioFirmado(context.Context, SolicitudRecuperarBinarioFirmado) (BinarioFirmadoRecuperado, error)
}

type ReferenciaDespliegueIdempotenciaBaremacion string

func (r ReferenciaDespliegueIdempotenciaBaremacion) Valida() bool

type ReferenciaGeneracionClaveHMACNominalBaremacion struct {
	// Has unexported fields.
}
```

ReferenciaGeneracionClaveHMACNominalBaremacion es un descriptor opaco para
solicitar al KMS la comprobacion conjunta. La referencia no acredita por si
sola una clave fisica distinta ni concede autoridad.

```go
func NuevaReferenciaGeneracionClaveHMACNominalBaremacion(
	dominio DominioClaveHMACBaremacion,
	generacion uint32,
	claveHMACRef string,
) (ReferenciaGeneracionClaveHMACNominalBaremacion, error)

func (u ReferenciaGeneracionClaveHMACNominalBaremacion) Format(estado fmt.State, _ rune)

func (ReferenciaGeneracionClaveHMACNominalBaremacion) GoString() string

func (u ReferenciaGeneracionClaveHMACNominalBaremacion) LogValue() slog.Value

func (ReferenciaGeneracionClaveHMACNominalBaremacion) MarshalBinary() ([]byte, error)

func (ReferenciaGeneracionClaveHMACNominalBaremacion) MarshalJSON() ([]byte, error)

func (ReferenciaGeneracionClaveHMACNominalBaremacion) MarshalText() ([]byte, error)

func (ReferenciaGeneracionClaveHMACNominalBaremacion) String() string

type ReferenciaVersionBaremacion struct {
	BaremacionMeritoRef string
	Numero              uint64
	HuellaEstadoSHA256  string
}

func (r ReferenciaVersionBaremacion) Validar() error

type RegistroAuditoriaBaremacion struct {
	Referencia                    string
	Secuencia                     uint64
	PrincipalRef                  string
	SujetoRef                     string
	PerfilActorClave              string
	MetodoAutenticacion           dominiovec.AuthMethod
	NivelAutenticacion            dominiovec.AuthAssurance
	GarantiaMinima                dominiovec.AuthAssurance
	AutenticacionRef              string
	AutorizacionRef               string
	AccionAutorizada              AccionOperacionBaremacion
	ClaseRecursoAutorizada        ClaseRecursoOperacionBaremacion
	RecursoAutorizadoRef          string
	CamposPermitidos              []string
	FinalidadClave                string
	CorrelacionRef                string
	Modulo                        string
	Accion                        AccionAuditoriaBaremacion
	ClaseCambio                   ClaseCambioBaremacion
	ProcesoRef                    string
	SolicitudRef                  string
	BaremacionMeritoRef           string
	DecisionRef                   string
	ManifiestoProbatorioRef       string
	HuellaManifiestoSHA256        string
	DocumentoFirmadoCustodiadoRef string
	EvidenciaCustodiaFirmadoRef   string
	EvidenciaRetencionFirmadoRef  string
	VersionAnterior               uint64
	VersionNueva                  uint64
	HuellaAnteriorSHA256          string
	HuellaNuevaSHA256             string
	MotivoClave                   string
	Motivo                        string
	HuellaSolicitudHMAC           string
	Resultado                     string
	SolicitadaConfirmacionEn      time.Time
	RegistradaEn                  time.Time
	HuellaAnteriorAuditoriaSHA256 string
	HuellaRegistroSHA256          string
}
```

RegistroAuditoriaBaremacion es una proyeccion probatoria cerrada, no una
entrada que el repositorio acepte al escribir.

```go
func (r RegistroAuditoriaBaremacion) Validar() error

type Reloj = puertosvec.Reloj

type RelojLlamamientos interface {
	Ahora() time.Time
}

type RepositorioBaremaciones interface {
	ReservarCambio(context.Context, SolicitudReservarCambioBaremacion) (ReservaCambioBaremacion, error)
	ConfirmarCambio(context.Context, SolicitudConfirmarCambioBaremacion) (ResultadoConfirmarCambioBaremacion, error)
	AbandonarReserva(context.Context, SolicitudAbandonarReservaBaremacion) error
	ObtenerVersionVigente(context.Context, SolicitudObtenerBaremacionVigente) (VersionBaremacion, error)
	ObtenerVersion(context.Context, SolicitudObtenerVersionBaremacion) (VersionBaremacion, error)
	ObtenerEvidenciaTransaccion(context.Context, SolicitudObtenerEvidenciaTransaccionBaremacion) (EvidenciaTransaccionBaremacionRecuperada, error)
}
```

RepositorioBaremaciones debe derivar y confirmar agregado, auditoria tipada
y un unico evento outbox en la misma transaccion. Alta crea solo version 1;
cada incorporacion anexa exactamente una decision con OCC exacto.

Un adaptador duradero no puede fiarse solo de la proyeccion del contexto:
dentro de esa misma transaccion y con su reloj fiable debe validar la
EvidenciaUsoAutorizacion, releer la decision registrada por DecisionRef y
exigir coincidencia exacta de su huella y vinculo V1. Tambien debe comprobar
que siguen vigentes sesion y contexto de actor, asignacion, rol, control de
revision y catalogo de politicas. Ausencia, ambiguedad, cambio, revocacion,
caducidad o error deniegan y revierten reserva/efecto/auditoria/outbox.

Cada mutacion consume de forma unica DecisionRef -> huella del efecto.
Solo un reintento de la misma decision y del mismo efecto exacto puede
recuperar el resultado anterior; reutilizarla para otro efecto se deniega.
Las lecturas sensibles deben realizar la misma revalidacion dentro de su
transaccion de lectura. Ningun adaptador productivo cumple el puerto si
omite estas barreras.

```go
type RepositorioFlujosFirmaBaremacion interface {
	CrearORecuperarFlujoFirmaBaremacion(context.Context, SolicitudCrearORecuperarFlujoFirmaBaremacion) (ResultadoCrearORecuperarFlujoFirmaBaremacion, error)
	ObtenerFlujoFirmaBaremacion(context.Context, SolicitudObtenerFlujoFirmaBaremacion) (ExpedienteFlujoFirmaBaremacion, error)
	AdquirirArrendamientoFlujoFirmaBaremacion(context.Context, SolicitudAdquirirArrendamientoFlujoFirmaBaremacion) (ResultadoAdquirirArrendamientoFlujoFirmaBaremacion, error)
	GuardarFlujoFirmaBaremacion(context.Context, SolicitudGuardarFlujoFirmaBaremacion) (ExpedienteFlujoFirmaBaremacion, error)
	LiberarArrendamientoFlujoFirmaBaremacion(context.Context, SolicitudLiberarArrendamientoFlujoFirmaBaremacion) error
}

type RepresentacionBaremacionConfiable struct {
	Representacion        dominiovec.RepresentacionDocumento
	EvidenciaConsultaRef  string
	HuellaEvidenciaSHA256 string
	ConsultadaEn          time.Time
}

func (r RepresentacionBaremacionConfiable) Validar() error

func (r RepresentacionBaremacionConfiable) ValidarPara(s SolicitudObtenerRepresentacionBaremacion) error

type ReservaCambioBaremacion struct {
	Token               TokenReservaBaremacion
	Repetida            bool
	VersionConfirmada   *VersionBaremacion
	BaremacionMeritoRef string
	Clase               ClaseCambioBaremacion
	VersionEsperada     *ReferenciaVersionBaremacion
	HuellaSolicitudHMAC string
	ExpiraEn            time.Time
}

func (r ReservaCambioBaremacion) Clonar() (ReservaCambioBaremacion, error)

func (r ReservaCambioBaremacion) Validar() error

func (r ReservaCambioBaremacion) ValidarPara(s SolicitudReservarCambioBaremacion) error
```

ValidarPara impide aceptar una reserva o una repeticion perteneciente a otra
solicitud, aun cuando la respuesta sea internamente valida.

```go
type ResolutorRecursoNecesidad interface {
	ResolverRecursosNecesidad(context.Context, string) ([]dominiovec.RecursoAutorizable, error)
}
```

ResolutorRecursoNecesidad devuelve todas las coincidencias autoritativas. El
caso de uso exige exactamente una; el puerto no puede ocultar ambiguedades
mediante LIMIT 1 ni escoger la primera fila.

```go
type ResultadoAdquirirArrendamientoFlujoFirmaBaremacion struct {
	Expediente    ExpedienteFlujoFirmaBaremacion
	Arrendamiento ArrendamientoFlujoFirmaBaremacion
}

type ResultadoAumentoFirma struct {
	ArtefactoOrigen                ArtefactoFirma
	Artefacto                      ArtefactoFirma
	NivelAlcanzadoClave            string
	PoliticaLongevidadRef          string
	PoliticaLongevidadVersion      int
	HuellaPoliticaLongevidadSHA256 string
	EvidenciaAumentoRef            string
	HuellaEvidenciaSHA256          string
	AumentadaEn                    time.Time
}

func (r ResultadoAumentoFirma) Validar() error

func (r ResultadoAumentoFirma) ValidarPara(s SolicitudAumentarFirma) error

func (r ResultadoAumentoFirma) ValidarRecuperacion(s SolicitudRecuperarAumentoFirma) error

type ResultadoCalculoOficial struct {
	Calculo               dominiobolsa.CalculoOficialBaremacion
	EvidenciaGobiernoRef  string
	HuellaEvidenciaSHA256 string
}

func (r ResultadoCalculoOficial) Clonar() (ResultadoCalculoOficial, error)

func (r ResultadoCalculoOficial) Validar() error

func (r ResultadoCalculoOficial) ValidarPara(s SolicitudCalcularPuntuacionOficial) error

type ResultadoConfirmarCambioBaremacion struct {
	Version   VersionBaremacion
	Evidencia EvidenciaTransaccionBaremacion
}

func (r ResultadoConfirmarCambioBaremacion) Clonar() (ResultadoConfirmarCambioBaremacion, error)

func (r ResultadoConfirmarCambioBaremacion) Validar() error

func (r ResultadoConfirmarCambioBaremacion) ValidarPara(s SolicitudConfirmarCambioBaremacion) error
```

ValidarPara liga el resultado a la mutacion exacta solicitada. Una respuesta
valida de otra baremacion, version o agregado nunca se acepta por semejanza.

```go
type ResultadoCrearORecuperarFlujoFirmaBaremacion struct {
	Expediente ExpedienteFlujoFirmaBaremacion
	Creado     bool
}

type ResultadoEjecutarPasoFirmaBaremacion struct {
	Paso                  PasoFlujoFirmaBaremacion
	EfectoRef             string
	ResultadoRef          string
	HuellaResultadoSHA256 string
	EstadoTrabajo         CargaProtegida
	ProyeccionLanzamiento *ProyeccionLanzamientoFirmaBaremacion
	ResultadoFinal        *ResultadoFinalFlujoFirmaBaremacion
	EjecutadoEn           time.Time
}

func (r ResultadoEjecutarPasoFirmaBaremacion) ValidarPara(s SolicitudEjecutarPasoFirmaBaremacion) error

type ResultadoFinalFlujoFirmaBaremacion struct {
	FlujoRef                     string
	DecisionRef                  string
	DocumentoFirmadoRef          string
	HuellaDocumentoFirmadoSHA256 string
	VersionBaremacion            uint64
	EvidenciaConfirmacionRef     string
	HuellaResultadoSHA256        string
	CompletadoEn                 time.Time
}

func (r ResultadoFinalFlujoFirmaBaremacion) Validar() error

type ResultadoNominalConfirmacionBaremacionV2 struct {
	IdentificadorOperacion IdentificadorOperacionTransaccionalBaremacion
	Resultado              ResultadoConfirmarCambioBaremacion
}
```

ResultadoNominalConfirmacionBaremacionV2 coteja solo la forma del eco de un
adaptador nominal. La coincidencia sintactica del identificador no demuestra
que el COMMIT lo persistiera ni que version, auditoria y evento nacieran
atomicamente. Esa atribucion requerira el resultado canonico autenticado V2.

```go
func (r ResultadoNominalConfirmacionBaremacionV2) ClonarFormaPara(
	s IntentoNominalConfirmacionBaremacionV2,
) (ResultadoNominalConfirmacionBaremacionV2, error)

func (r ResultadoNominalConfirmacionBaremacionV2) Format(estado fmt.State, _ rune)

func (ResultadoNominalConfirmacionBaremacionV2) GoString() string

func (r ResultadoNominalConfirmacionBaremacionV2) LogValue() slog.Value

func (ResultadoNominalConfirmacionBaremacionV2) MarshalBinary() ([]byte, error)

func (ResultadoNominalConfirmacionBaremacionV2) MarshalJSON() ([]byte, error)

func (ResultadoNominalConfirmacionBaremacionV2) MarshalText() ([]byte, error)

func (ResultadoNominalConfirmacionBaremacionV2) String() string

func (*ResultadoNominalConfirmacionBaremacionV2) UnmarshalBinary([]byte) error

func (*ResultadoNominalConfirmacionBaremacionV2) UnmarshalJSON([]byte) error

func (*ResultadoNominalConfirmacionBaremacionV2) UnmarshalText([]byte) error

func (r ResultadoNominalConfirmacionBaremacionV2) ValidarFormaPara(
	s IntentoNominalConfirmacionBaremacionV2,
) error

type ResultadoNominalConfirmacionBaremacionV3 struct {
	IdentificadorOperacion IdentificadorOperacionTransaccionalBaremacion
	Resultado              ResultadoConfirmarCambioBaremacion
}
```

ResultadoNominalConfirmacionBaremacionV3 valida solo el eco nominal;
no eleva el sobre a prueba de COMMIT ni relaja el fail-closed transaccional.

```go
func (r ResultadoNominalConfirmacionBaremacionV3) ClonarFormaPara(
	s IntentoNominalConfirmacionBaremacionV3,
) (ResultadoNominalConfirmacionBaremacionV3, error)

func (r ResultadoNominalConfirmacionBaremacionV3) Format(estado fmt.State, _ rune)

func (ResultadoNominalConfirmacionBaremacionV3) GoString() string

func (r ResultadoNominalConfirmacionBaremacionV3) LogValue() slog.Value

func (ResultadoNominalConfirmacionBaremacionV3) MarshalBinary() ([]byte, error)

func (ResultadoNominalConfirmacionBaremacionV3) MarshalJSON() ([]byte, error)

func (ResultadoNominalConfirmacionBaremacionV3) MarshalText() ([]byte, error)

func (ResultadoNominalConfirmacionBaremacionV3) String() string

func (*ResultadoNominalConfirmacionBaremacionV3) UnmarshalBinary([]byte) error

func (*ResultadoNominalConfirmacionBaremacionV3) UnmarshalJSON([]byte) error

func (*ResultadoNominalConfirmacionBaremacionV3) UnmarshalText([]byte) error

func (r ResultadoNominalConfirmacionBaremacionV3) ValidarFormaPara(
	s IntentoNominalConfirmacionBaremacionV3,
) error

type SelladorSellosBaremacion interface {
	SellarSelloBaremacion(context.Context, SolicitudSellarSelloBaremacion) (string, error)
}
```

SelladorSellosBaremacion produce sellos con una finalidad explicita.
La implementacion debe resolver la clave activa de esa finalidad y devolver
el identificador de clave en el formato hmac-sha256:<clave>:<hex>.

```go
type SelladorServicioBaremacion interface {
	SelladorSolicitudBaremacion
	SelladorSellosBaremacion
}
```

SelladorServicioBaremacion es el compuesto requerido por ServicioBaremacion.
La fachada durable de firma conserva deliberadamente el puerto generico
SelladorSolicitudBaremacion y no queda acoplada a este contrato
transaccional.

```go
type SelladorSolicitudBaremacion interface {
	SellarSolicitudBaremacion(context.Context, CargaProtegida) (string, error)
}

type SelladorTiempoFirma interface {
	SellarTiempoFirma(context.Context, SolicitudSellarTiempoFirma) (SelloTiempoFirma, error)
}

type SelloTiempoFirma struct {
	SelloTiempoRef                  string
	HuellaSelloTiempoSHA256         string
	ArtefactoOrigen                 ArtefactoFirma
	ArtefactoSellado                ArtefactoFirma
	PoliticaSelloTiempoRef          string
	PoliticaSelloTiempoVersion      int
	HuellaPoliticaSelloTiempoSHA256 string
	ValidacionSelloRef              string
	HuellaValidacionSHA256          string
	SelladoEn                       time.Time
}

func (s SelloTiempoFirma) Validar() error

func (s SelloTiempoFirma) ValidarPara(sol SolicitudSellarTiempoFirma) error

func (sello SelloTiempoFirma) ValidarRecuperacion(s SolicitudRecuperarSelloTiempo) error

type SesionFirmaInteractiva struct {
	SesionRef               string
	Estado                  EstadoSesionFirmaInteractiva
	CargaLanzamiento        CargaProtegida
	MIMELanzamiento         string
	Documento               DocumentoFirmableCustodiado
	PoliticaFirmaRef        string
	PoliticaFirmaVersion    int
	HuellaPoliticaSHA256    string
	EvidenciaPreparacionRef string
	HuellaEvidenciaSHA256   string
	PreparadaEn             time.Time
	ExpiraEn                time.Time
}

func (s SesionFirmaInteractiva) Validar() error

func (s SesionFirmaInteractiva) ValidarPara(solicitud SolicitudPrepararFirmaInteractiva) error

type SeudonimoSujetoBaremacionHMAC struct {
	Version      uint16
	ClaveHMACRef string
	ValorHMAC    string
}
```

SeudonimoSujetoBaremacionHMAC sustituye cualquier DNI, nombre o referencia
libre en la intencion persistible. Su clave tiene dominio exclusivo y debe
conservarse en el llavero mientras existan operaciones recuperables.

```go
func (s SeudonimoSujetoBaremacionHMAC) Clonar() (SeudonimoSujetoBaremacionHMAC, error)

func (s SeudonimoSujetoBaremacionHMAC) Format(estado fmt.State, _ rune)

func (SeudonimoSujetoBaremacionHMAC) GoString() string

func (s SeudonimoSujetoBaremacionHMAC) IgualConstante(otro SeudonimoSujetoBaremacionHMAC) bool

func (s SeudonimoSujetoBaremacionHMAC) LogValue() slog.Value

func (SeudonimoSujetoBaremacionHMAC) MarshalBinary() ([]byte, error)

func (SeudonimoSujetoBaremacionHMAC) MarshalJSON() ([]byte, error)

func (SeudonimoSujetoBaremacionHMAC) MarshalText() ([]byte, error)

func (SeudonimoSujetoBaremacionHMAC) String() string

func (s SeudonimoSujetoBaremacionHMAC) Validar() error

type SolicitudAbandonarReservaBaremacion struct {
	Contexto            ContextoOperacionBaremacion
	Token               TokenReservaBaremacion
	Clase               ClaseCambioBaremacion
	BaremacionMeritoRef string
}

func (s SolicitudAbandonarReservaBaremacion) Validar() error

type SolicitudAdquirirArrendamientoFlujoFirmaBaremacion struct {
	Consulta        SolicitudObtenerFlujoFirmaBaremacion
	VersionEsperada uint64
	PropietarioRef  string
	Duracion        time.Duration
}

func (s SolicitudAdquirirArrendamientoFlujoFirmaBaremacion) Validar() error

type SolicitudAumentarFirma struct {
	Contexto          ContextoOperacionFirma
	ClaveIdempotencia string
	Artefacto         ArtefactoFirma
	Validacion        ValidacionFirmaServidor
	SelloTiempo       *SelloTiempoFirma
	Politica          PoliticaFirmaBaremacion
	SolicitadaEn      time.Time
}

func (s SolicitudAumentarFirma) Clonar() (SolicitudAumentarFirma, error)

func (s SolicitudAumentarFirma) Validar() error

type SolicitudCalcularPuntuacionOficial struct {
	Contexto            ContextoOperacionBaremacion
	BaremacionMeritoRef string
	ProcesoRef          string
	SolicitudRef        string
	SujetoRef           string
	Criterio            dominiobolsa.ReferenciaCriterio
	Evidencias          []EvidenciaBaremacionConfiable
	PuntosDeclarados    dominiobolsa.Puntos
	SolicitadaEn        time.Time
}
```

SolicitudCalcularPuntuacionOficial no contiene PuntosCalculados. Solo admite
criterio gobernado y evidencias que ya han pasado por FuenteDatosBaremacion.

```go
func (s SolicitudCalcularPuntuacionOficial) Clonar() (SolicitudCalcularPuntuacionOficial, error)

func (s SolicitudCalcularPuntuacionOficial) Validar() error

type SolicitudCodificarDecisionCanonica struct {
	Contexto             ContextoOperacionBaremacion
	AutorizacionDecision ContextoOperacionBaremacion
	Contenido            dominiobolsa.ContenidoDecisionTecnica
	Politica             PoliticaFirmaBaremacion
}

func (s SolicitudCodificarDecisionCanonica) Clonar() (SolicitudCodificarDecisionCanonica, error)

func (s SolicitudCodificarDecisionCanonica) Validar() error

type SolicitudConfirmarCambioBaremacion struct {
	Contexto                     ContextoOperacionBaremacion
	ContextoPrevalidacionArchivo ContextoOperacionBaremacion
	Token                        TokenReservaBaremacion
	Clase                        ClaseCambioBaremacion
	VersionEsperada              *ReferenciaVersionBaremacion
	HuellaSolicitudHMAC          string
	Agregado                     dominiobolsa.BaremacionMerito
	Manifiesto                   *ManifiestoProbatorioBaremacion
	Trazabilidad                 TrazabilidadCambioBaremacion
	ConfirmadaEn                 time.Time
}

func (s SolicitudConfirmarCambioBaremacion) Clonar() (SolicitudConfirmarCambioBaremacion, error)

func (s SolicitudConfirmarCambioBaremacion) Validar() error

type SolicitudConsultarFirmaInteractiva struct {
	Contexto              ContextoOperacionFirma
	SesionRef             string
	Documento             DocumentoFirmableCustodiado
	HuellaContenidoSHA256 string
	PoliticaFirmaRef      string
	PoliticaFirmaVersion  int
	HuellaPoliticaSHA256  string
	FirmanteRef           string
	PerfilFirmanteClave   string
}

func (s SolicitudConsultarFirmaInteractiva) Validar() error

type SolicitudCrearORecuperarFlujoFirmaBaremacion struct {
	Expediente ExpedienteFlujoFirmaBaremacion
}

type SolicitudCustodiarDocumentoFirmable struct {
	Contexto            ContextoOperacionBaremacion
	OperacionRef        string
	ClaveIdempotencia   string
	CargaRef            string
	SujetoSeudonimoHMAC string
	HuellaAlmacenHMAC   string
	EfectoRef           string
	ProcesoRef          string
	SolicitudRef        string
	BaremacionMeritoRef string
	DecisionRef         string
	ClasificacionClave  string
	Codificacion        CodificacionCanonicaDecision
}

func (s SolicitudCustodiarDocumentoFirmable) PrepararEscritura() (puertosvec.SolicitudEscribirObjeto, error)
```

PrepararEscritura crea el puente exacto hacia el almacen VEC. La carga se
vuelve a copiar y la zona admitida evita que un artefacto no confiable
llegue al firmador. El conector VEC verifica tamano y SHA-256 al escribir.

```go
func (s SolicitudCustodiarDocumentoFirmable) Validar() error

type SolicitudDerivarHMACMotivoBaremacion struct {
	// Has unexported fields.
}

func (s SolicitudDerivarHMACMotivoBaremacion) Format(estado fmt.State, _ rune)

func (SolicitudDerivarHMACMotivoBaremacion) GoString() string

func (s SolicitudDerivarHMACMotivoBaremacion) LogValue() slog.Value

func (SolicitudDerivarHMACMotivoBaremacion) MarshalBinary() ([]byte, error)

func (SolicitudDerivarHMACMotivoBaremacion) MarshalJSON() ([]byte, error)

func (SolicitudDerivarHMACMotivoBaremacion) MarshalText() ([]byte, error)

func (s SolicitudDerivarHMACMotivoBaremacion) MotivoClave() (string, error)

func (SolicitudDerivarHMACMotivoBaremacion) String() string

func (s SolicitudDerivarHMACMotivoBaremacion) Validar() error

func (s SolicitudDerivarHMACMotivoBaremacion) VisitarMotivo(
	visita func([]byte) error,
) error

type SolicitudEjecutarPasoFirmaBaremacion struct {
	FlujoRef              string
	Paso                  PasoFlujoFirmaBaremacion
	EfectoRef             string
	ClaveIdempotenciaHMAC string
	VinculoActorHMAC      string
	PerfilActorClave      string
	ProcesoRef            string
	SolicitudRef          string
	BaremacionMeritoRef   string
	DecisionRef           string
	EstadoTrabajo         CargaProtegida
	PuntosPrevios         []PuntoControlFirmaBaremacion
}
```

SolicitudEjecutarPasoFirmaBaremacion porta un estado de trabajo en claro
solo dentro del proceso. El ejecutor productivo debe rederivar identidad y
autorizacion desde ctx y recuperar por (EfectoRef, ClaveIdempotenciaHMAC)
antes de iniciar un efecto remoto.

```go
func (s SolicitudEjecutarPasoFirmaBaremacion) Validar() error

type SolicitudEvaluarParticipacionLlamamiento struct {
	Necesidad               dominiobolsa.NecesidadCobertura
	InstantaneaRef          string
	VersionInstantanea      uint64
	HuellaInstantaneaSHA256 string
	InstanteReferencia      time.Time
	InstantaneaGeneradaEn   time.Time
	Politica                dominiobolsa.ReferenciaPoliticaLlamamiento
	Entrada                 dominiobolsa.EntradaOrdenBolsa
	EvaluadaEn              time.Time
}
```

SolicitudEvaluarParticipacionLlamamiento contiene una sola posicion. El
motor no recibe el resto del listado y por tanto no puede saltar posiciones.
Estado y criterios siguen siendo referencias versionadas, no enumeraciones
favorables codificadas en la aplicacion.

```go
func (s SolicitudEvaluarParticipacionLlamamiento) Validar() error

type SolicitudGuardarFlujoFirmaBaremacion struct {
	VersionEsperada uint64
	Arrendamiento   ArrendamientoFlujoFirmaBaremacion
	Siguiente       ExpedienteFlujoFirmaBaremacion
}

func (s SolicitudGuardarFlujoFirmaBaremacion) Validar() error

type SolicitudLiberarArrendamientoFlujoFirmaBaremacion struct {
	Arrendamiento ArrendamientoFlujoFirmaBaremacion
}

type SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion struct {
	// Has unexported fields.
}
```

SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion representa
los seis dominios completos y todas sus generaciones recuperables (1..8 por
dominio, hasta 48). El KMS debe resolver cada alias y acreditar separacion
de clave fisica y politica; esta forma solo elimina omisiones y alias
logicos.

```go
func ConstruirSolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion(
	referencias []ReferenciaGeneracionClaveHMACNominalBaremacion,
) (SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion, error)

func (s SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion) Format(estado fmt.State, _ rune)

func (SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion) GoString() string

func (s SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion) LogValue() slog.Value

func (SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion) MarshalBinary() ([]byte, error)

func (SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion) MarshalJSON() ([]byte, error)

func (SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion) MarshalText() ([]byte, error)

func (SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion) String() string

func (s SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion) VisitarReferencias(
	visita func(DominioClaveHMACBaremacion, uint32, string) error,
) error
```

VisitarReferencias entrega el lote canonico solo durante la llamada.

```go
type SolicitudObtenerBaremacionVigente struct {
	Contexto            ContextoOperacionBaremacion
	BaremacionMeritoRef string
}

func (s SolicitudObtenerBaremacionVigente) Validar() error

type SolicitudObtenerCriterioBaremacion struct {
	Contexto             ContextoOperacionBaremacion
	ProcesoRef           string
	Clave                string
	Version              int
	HuellaEsperadaSHA256 string
}

func (s SolicitudObtenerCriterioBaremacion) Validar() error

type SolicitudObtenerEvidenciaBaremacion struct {
	Contexto     ContextoOperacionBaremacion
	ProcesoRef   string
	SolicitudRef string
	Evidencia    dominiobolsa.EvidenciaMerito
}

func (s SolicitudObtenerEvidenciaBaremacion) Validar() error

type SolicitudObtenerEvidenciaTransaccionBaremacion struct {
	Contexto            ContextoOperacionBaremacion
	BaremacionMeritoRef string
	NumeroVersion       uint64
	AuditoriaRef        string
	EventoOutboxRef     string
}

func (s SolicitudObtenerEvidenciaTransaccionBaremacion) Validar() error

type SolicitudObtenerFlujoFirmaBaremacion struct {
	FlujoRef               string
	IndiceIdempotenciaHMAC string
	VinculoActorHMAC       string
}

func (s SolicitudObtenerFlujoFirmaBaremacion) Validar() error

type SolicitudObtenerPoliticaFirma struct {
	Contexto             ContextoOperacionBaremacion
	Referencia           string
	Version              int
	HuellaEsperadaSHA256 string
	VigenteEn            time.Time
}

func (s SolicitudObtenerPoliticaFirma) Validar() error

type SolicitudObtenerRepresentacionBaremacion struct {
	Contexto   ContextoOperacionBaremacion
	Referencia dominiobolsa.ReferenciaEvidencia
}

func (s SolicitudObtenerRepresentacionBaremacion) Validar() error

type SolicitudObtenerVersionBaremacion struct {
	Contexto            ContextoOperacionBaremacion
	BaremacionMeritoRef string
	Numero              uint64
}

func (s SolicitudObtenerVersionBaremacion) Validar() error

type SolicitudPrepararFirmaInteractiva struct {
	Contexto            ContextoOperacionFirma
	ClaveIdempotencia   string
	ProcesoRef          string
	SolicitudRef        string
	BaremacionMeritoRef string
	DecisionRef         string
	Documento           DocumentoFirmableCustodiado
	FirmanteRef         string
	PerfilFirmanteClave string
	Politica            PoliticaFirmaBaremacion
	SolicitadaEn        time.Time
	ExpiraEn            time.Time
}

func (s SolicitudPrepararFirmaInteractiva) Validar() error

type SolicitudProponerLlamamiento struct {
	Actor            dominiovec.ContextoActor
	PerfilActivoRef  string
	AutenticacionRef string
	SesionRef        string
	NecesidadRef     string
	CorrelacionRef   string
}
```

SolicitudProponerLlamamiento es el unico dato de entrada del caso de uso.
Actor procede del resolutor canonico del nucleo: no se acepta identidad,
perfiles, roles ni atributos reconstruidos por un adaptador HTTP. El perfil
se repite deliberadamente para exigir una seleccion expresa y coincidente.
No hay campos para listados, participaciones, estados ni evaluaciones.

```go
func (s SolicitudProponerLlamamiento) Clonar() (SolicitudProponerLlamamiento, error)

func (SolicitudProponerLlamamiento) MarshalJSON() ([]byte, error)
```

El comando es interno y no puede usarse como DTO HTTP. La frontera de
entrada debe resolver ContextoActor en middleware confiable y construir el
comando de forma deliberada a partir de referencias ya validadas.

```go
func (*SolicitudProponerLlamamiento) UnmarshalJSON([]byte) error

func (s SolicitudProponerLlamamiento) Validar() error

type SolicitudRecuperarArtefactoFirma struct {
	Contexto              ContextoOperacionFirma
	FirmaRef              string
	HuellaFirmaSHA256     string
	DocumentoFirmadoRef   string
	HuellaDocumentoSHA256 string
}

func (s SolicitudRecuperarArtefactoFirma) Validar() error

type SolicitudRecuperarAumentoFirma struct {
	Contexto            ContextoOperacionFirma
	EvidenciaAumentoRef string
	HuellaAumentoSHA256 string
}

func (s SolicitudRecuperarAumentoFirma) Validar() error

type SolicitudRecuperarBinarioFirmado struct {
	Contexto              ContextoOperacionFirma
	DocumentoFirmadoRef   string
	HuellaDocumentoSHA256 string
	LimiteBytes           int64
}

func (s SolicitudRecuperarBinarioFirmado) Validar() error

type SolicitudRecuperarCalculoOficial struct {
	Contexto        ContextoOperacionBaremacion
	CalculoRef      string
	HuellaResultado string
}

func (s SolicitudRecuperarCalculoOficial) Validar() error

type SolicitudRecuperarSelloTiempo struct {
	Contexto          ContextoOperacionFirma
	SelloTiempoRef    string
	HuellaSelloSHA256 string
}

func (s SolicitudRecuperarSelloTiempo) Validar() error

type SolicitudRecuperarValidacionFirma struct {
	Contexto               ContextoOperacionFirma
	ValidacionRef          string
	HuellaValidacionSHA256 string
}

func (s SolicitudRecuperarValidacionFirma) Validar() error

type SolicitudReservarCambioBaremacion struct {
	Contexto            ContextoOperacionBaremacion
	Clase               ClaseCambioBaremacion
	ClaveIdempotencia   string
	BaremacionMeritoRef string
	VersionEsperada     *ReferenciaVersionBaremacion
	HuellaSolicitudHMAC string
	SolicitadaEn        time.Time
	ExpiraEn            time.Time
}

func (s SolicitudReservarCambioBaremacion) Clonar() SolicitudReservarCambioBaremacion

func (s SolicitudReservarCambioBaremacion) Validar() error

type SolicitudResolverSeudonimoSujetoBaremacion struct {
	// Has unexported fields.
}
```

SolicitudResolverSeudonimoSujetoBaremacion solo identifica el expediente.
La frontera resuelve el sujeto autoritativo desde datos internos; el cliente
no puede presentar DNI, nombre ni una referencia de persona.

```go
func NuevaSolicitudResolverSeudonimoSujetoBaremacion(
	procesoRef, solicitudRef, baremacionMeritoRef string,
) (SolicitudResolverSeudonimoSujetoBaremacion, error)

func (s SolicitudResolverSeudonimoSujetoBaremacion) Format(estado fmt.State, _ rune)

func (SolicitudResolverSeudonimoSujetoBaremacion) GoString() string

func (s SolicitudResolverSeudonimoSujetoBaremacion) LogValue() slog.Value

func (SolicitudResolverSeudonimoSujetoBaremacion) MarshalBinary() ([]byte, error)

func (SolicitudResolverSeudonimoSujetoBaremacion) MarshalJSON() ([]byte, error)

func (SolicitudResolverSeudonimoSujetoBaremacion) MarshalText() ([]byte, error)

func (SolicitudResolverSeudonimoSujetoBaremacion) String() string

func (s SolicitudResolverSeudonimoSujetoBaremacion) VisitarReferencias(
	visita func(procesoRef, solicitudRef, baremacionMeritoRef string) error,
) error
```

VisitarReferencias entrega las referencias opacas solo durante la llamada al
adaptador de identidad; evita getters y colecciones reutilizables.

```go
type SolicitudSellarSelloBaremacion struct {
	Finalidad              FinalidadSelloBaremacion
	RepresentacionCanonica CargaProtegida
}
```

SolicitudSellarSelloBaremacion obliga al productor a declarar el dominio
criptografico que el verificador recibira despues. No debe sustituirse
por un sellador generico: la finalidad permite seleccionar una clave y
un llavero historico independientes sin interpretar bytes opacos en el
conector.

```go
func (s SolicitudSellarSelloBaremacion) MaterialCanonicoHMAC() (CargaProtegida, error)
```

MaterialCanonicoHMAC devuelve la preimagen contractual exacta:

    HMAC(K_finalidad, finalidad || 0x00 || representacion_canonica)

La finalidad cerrada no puede contener NUL. La representacion conserva
ademas su dominio funcional para que el artefacto siga siendo autocontenido;
esta doble declaracion es deliberada y queda congelada por vector de prueba.

```go
func (s SolicitudSellarSelloBaremacion) Validar() error

type SolicitudSellarTiempoFirma struct {
	Contexto          ContextoOperacionFirma
	ClaveIdempotencia string
	ArtefactoOrigen   ArtefactoFirma
	ValidacionOrigen  ValidacionFirmaServidor
	Politica          PoliticaFirmaBaremacion
	SolicitadaEn      time.Time
}

func (s SolicitudSellarTiempoFirma) Validar() error

type SolicitudTestimonioAtomicoIdempotenciaBaremacion struct {
	// Has unexported fields.
}
```

SolicitudTestimonioAtomicoIdempotenciaBaremacion posee campos privados y no
expone la clave. El productor solo recibe el contexto estable por getters y
una fuente efimera creada dentro de la fabrica nominal.

```go
func NuevaSolicitudTestimonioAtomicoIdempotenciaBaremacion(
	despliegue ReferenciaDespliegueIdempotenciaBaremacion,
	modulo ModuloIdempotenciaBaremacion,
	clase ClaseCambioBaremacion,
	ambitoSujeto SolicitudResolverSeudonimoSujetoBaremacion,
	seudonimo SeudonimoSujetoBaremacionHMAC,
	clave ClaveClienteIdempotenciaBaremacion,
) (SolicitudTestimonioAtomicoIdempotenciaBaremacion, error)

func (s SolicitudTestimonioAtomicoIdempotenciaBaremacion) Clase() ClaseCambioBaremacion

func (s SolicitudTestimonioAtomicoIdempotenciaBaremacion) DespliegueRef() ReferenciaDespliegueIdempotenciaBaremacion

func (s SolicitudTestimonioAtomicoIdempotenciaBaremacion) Format(estado fmt.State, _ rune)

func (SolicitudTestimonioAtomicoIdempotenciaBaremacion) GoString() string

func (s SolicitudTestimonioAtomicoIdempotenciaBaremacion) LogValue() slog.Value

func (SolicitudTestimonioAtomicoIdempotenciaBaremacion) MarshalBinary() ([]byte, error)

func (SolicitudTestimonioAtomicoIdempotenciaBaremacion) MarshalJSON() ([]byte, error)

func (SolicitudTestimonioAtomicoIdempotenciaBaremacion) MarshalText() ([]byte, error)

func (s SolicitudTestimonioAtomicoIdempotenciaBaremacion) Modulo() ModuloIdempotenciaBaremacion

func (SolicitudTestimonioAtomicoIdempotenciaBaremacion) String() string

func (s SolicitudTestimonioAtomicoIdempotenciaBaremacion) VisitarAmbitoSujetoBaremacion(
	visita func(
		procesoRef, solicitudRef, baremacionMeritoRef string,
		versionSeudonimo uint16, claveSeudonimoRef, valorSeudonimoHMAC string,
	) error,
) error
```

VisitarAmbitoSujetoBaremacion entrega el expediente opaco y el seudonimo
ya resuelto exclusivamente para ligar solicitud y atestacion. El seudonimo
es rotatorio y, por tanto, nunca forma la preimagen del principal estable.
No expone DNI, nombre ni un getter reutilizable. En este puerto nominal
la forma no acredita origen: el futuro servicio privado debe resolver y
verificar el seudonimo con FronteraIdentidadesEstablesBaremacion antes de
construirla.

```go
type SolicitudValidarFirmaServidor struct {
	Contexto                              ContextoOperacionFirma
	Artefacto                             ArtefactoFirma
	Politica                              PoliticaFirmaBaremacion
	FirmanteEsperadoRef                   string
	PerfilEsperadoClave                   string
	PerfilFirmaEsperadoClave              string
	SelloTiempoEsperadoRef                string
	HuellaSelloTiempoEsperadaSHA256       string
	AumentoLongevidadEsperadoRef          string
	HuellaAumentoLongevidadEsperadaSHA256 string
	SolicitadaEn                          time.Time
}
```

SolicitudValidarFirmaServidor exige que el conector ateste tanto la revision
exacta del PDF como las evidencias embebidas del perfil solicitado.

```go
func (s SolicitudValidarFirmaServidor) Validar() error

type SolicitudVerificarEstadoFlujoFirmaBaremacion struct {
	RepresentacionCanonica CargaProtegida
	SelloHMAC              string
}

func (s SolicitudVerificarEstadoFlujoFirmaBaremacion) Validar() error

type SolicitudVerificarHMACMotivoBaremacion struct {
	Solicitud SolicitudDerivarHMACMotivoBaremacion
	Sello     HMACMotivoBaremacion
}

func (s SolicitudVerificarHMACMotivoBaremacion) Format(estado fmt.State, _ rune)

func (SolicitudVerificarHMACMotivoBaremacion) GoString() string

func (s SolicitudVerificarHMACMotivoBaremacion) LogValue() slog.Value

func (SolicitudVerificarHMACMotivoBaremacion) MarshalBinary() ([]byte, error)

func (SolicitudVerificarHMACMotivoBaremacion) MarshalJSON() ([]byte, error)

func (SolicitudVerificarHMACMotivoBaremacion) MarshalText() ([]byte, error)

func (SolicitudVerificarHMACMotivoBaremacion) String() string

func (s SolicitudVerificarHMACMotivoBaremacion) Validar() error

type SolicitudVerificarSelloBaremacion struct {
	Finalidad              FinalidadSelloBaremacion
	RepresentacionCanonica CargaProtegida
	SelloHMAC              string
}
```

SolicitudVerificarSelloBaremacion entrega al componente criptografico la
representacion canonica completa. El repositorio nunca recibe ni conserva
material de clave.

```go
func (s SolicitudVerificarSelloBaremacion) MaterialCanonicoHMAC() (CargaProtegida, error)
```

MaterialCanonicoHMAC reutiliza sin variantes la misma preimagen que el
productor. El sello solo autoriza la conversion; nunca entra en su propia
preimagen.

```go
func (s SolicitudVerificarSelloBaremacion) Validar() error

type TipoEventoOutboxBaremacion string

const (
	TipoEventoBaremacionCreada    TipoEventoOutboxBaremacion = "bolsa.baremacion_creada.v1"
	TipoEventoDecisionIncorporada TipoEventoOutboxBaremacion = "bolsa.decision_baremacion_incorporada.v1"
)
type TipoEvidenciaProbatoriaBaremacion string
```

TipoEvidenciaProbatoriaBaremacion es un catalogo cerrado. Una evidencia no
puede incorporarse al manifiesto bajo una etiqueta libre o ambigua.

```go
const (
	EvidenciaEstadoBaseBaremacion                 TipoEvidenciaProbatoriaBaremacion = "estado_base"
	EvidenciaCalculoOficialBaremacion             TipoEvidenciaProbatoriaBaremacion = "calculo_oficial"
	EvidenciaCriterioPublicadoBaremacion          TipoEvidenciaProbatoriaBaremacion = "criterio_publicado"
	EvidenciaDocumentoMeritoBaremacion            TipoEvidenciaProbatoriaBaremacion = "documento_merito"
	EvidenciaRepresentacionBaremacion             TipoEvidenciaProbatoriaBaremacion = "representacion_documento"
	EvidenciaContenidoDecisionBaremacion          TipoEvidenciaProbatoriaBaremacion = "contenido_decision"
	EvidenciaPoliticaFirmaBaremacion              TipoEvidenciaProbatoriaBaremacion = "politica_firma"
	EvidenciaDocumentoCanonicoBaremacion          TipoEvidenciaProbatoriaBaremacion = "documento_canonico"
	EvidenciaCustodiaFirmableBaremacion           TipoEvidenciaProbatoriaBaremacion = "custodia_firmable"
	EvidenciaPreparacionFirmaBaremacion           TipoEvidenciaProbatoriaBaremacion = "preparacion_firma"
	EvidenciaConsultaFirmaBaremacion              TipoEvidenciaProbatoriaBaremacion = "consulta_firma"
	EvidenciaValidacionInicialBaremacion          TipoEvidenciaProbatoriaBaremacion = "validacion_firma_inicial"
	EvidenciaSelloTiempoBaremacion                TipoEvidenciaProbatoriaBaremacion = "sello_tiempo"
	EvidenciaVinculoRevisionSelladaBaremacion     TipoEvidenciaProbatoriaBaremacion = "vinculo_revision_sellada"
	EvidenciaValidacionDocumentoSelladoBaremacion TipoEvidenciaProbatoriaBaremacion = "validacion_documento_sellado"
	EvidenciaAumentoLongevidadBaremacion          TipoEvidenciaProbatoriaBaremacion = "aumento_longevidad"
	EvidenciaVinculoRevisionLongevaBaremacion     TipoEvidenciaProbatoriaBaremacion = "vinculo_revision_longeva"
	EvidenciaValidacionFinalBaremacion            TipoEvidenciaProbatoriaBaremacion = "validacion_firma_final"
	EvidenciaRecuperacionFirmadoBaremacion        TipoEvidenciaProbatoriaBaremacion = "recuperacion_documento_firmado"
	EvidenciaCustodiaFirmadoBaremacion            TipoEvidenciaProbatoriaBaremacion = "custodia_documento_firmado"
	EvidenciaRetencionFirmadoBaremacion           TipoEvidenciaProbatoriaBaremacion = "retencion_documento_firmado"
)
type TokenArrendamientoFlujoFirmaBaremacion struct {
	// Has unexported fields.
}
```

TokenArrendamientoFlujoFirmaBaremacion es la capacidad aleatoria que
autoriza a usar un arrendamiento. Su valor solo vive en el cierre privado
e inmutable operarHMAC: no tiene representacion generica ni ofrece una
operacion para revelarlo. Los adaptadores conservan exclusivamente una
huella HMAC y la verifican antes de aceptar cualquier cambio o liberacion.

```go
func NuevoTokenArrendamientoFlujoFirmaBaremacion() (TokenArrendamientoFlujoFirmaBaremacion, error)
```

NuevoTokenArrendamientoFlujoFirmaBaremacion crea una capacidad de 256 bits
mediante el generador criptografico del sistema operativo.

```go
func (t TokenArrendamientoFlujoFirmaBaremacion) CoincideHuellaHMACSHA256(
	clave, esperada []byte,
) bool
```

CoincideHuellaHMACSHA256 autentica la capacidad mediante comparacion en
tiempo constante. Una capacidad nula, una clave invalida o una huella con
longitud incorrecta fallan cerradas.

```go
func (t TokenArrendamientoFlujoFirmaBaremacion) Format(estado fmt.State, _ rune)

func (t TokenArrendamientoFlujoFirmaBaremacion) GoString() string

func (*TokenArrendamientoFlujoFirmaBaremacion) GobDecode([]byte) error

func (TokenArrendamientoFlujoFirmaBaremacion) GobEncode() ([]byte, error)

func (t TokenArrendamientoFlujoFirmaBaremacion) HuellaHMACSHA256(clave []byte) ([]byte, error)
```

HuellaHMACSHA256 calcula el comprobante almacenable de la capacidad.
Nunca devuelve el token y exige una clave de, al menos, 256 bits.

```go
func (t TokenArrendamientoFlujoFirmaBaremacion) LogValue() slog.Value

func (TokenArrendamientoFlujoFirmaBaremacion) MarshalBinary() ([]byte, error)

func (TokenArrendamientoFlujoFirmaBaremacion) MarshalJSON() ([]byte, error)

func (TokenArrendamientoFlujoFirmaBaremacion) MarshalText() ([]byte, error)

func (TokenArrendamientoFlujoFirmaBaremacion) MarshalXML(*xml.Encoder, xml.StartElement) error

func (TokenArrendamientoFlujoFirmaBaremacion) String() string

func (*TokenArrendamientoFlujoFirmaBaremacion) UnmarshalBinary([]byte) error

func (*TokenArrendamientoFlujoFirmaBaremacion) UnmarshalJSON([]byte) error

func (*TokenArrendamientoFlujoFirmaBaremacion) UnmarshalText([]byte) error

func (*TokenArrendamientoFlujoFirmaBaremacion) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (t TokenArrendamientoFlujoFirmaBaremacion) Validar() error

type TokenReservaBaremacion struct {
	// Has unexported fields.
}
```

TokenReservaBaremacion es una capacidad temporal, nunca un identificador
de negocio. Su representacion Base64URL solo vive capturada por un cierre
privado: el valor no puede recuperarse mediante la API de reflexion segura.

La huella durable conserva deliberadamente el contrato historico exacto
SHA-256(Base64URL), sin decodificar ni anadir un dominio. Cambiar esa
formula impediria localizar reservas V1/V3 y alteraria los vectores
probatorios.

```go
func NuevoTokenReservaBaremacion(valor string) (TokenReservaBaremacion, error)
```

NuevoTokenReservaBaremacion valida e importa la representacion Base64URL
canonica empleada por el contrato historico. El tipo resultante no ofrece
ninguna operacion publica para volver a obtenerla.

```go
func (t TokenReservaBaremacion) CoincideConHuellaSHA256(huella string) bool
```

CoincideConHuellaSHA256 compara los 32 bytes en tiempo constante y rechaza
representaciones hexadecimales no canonicas.

```go
func (t TokenReservaBaremacion) Format(estado fmt.State, _ rune)

func (TokenReservaBaremacion) GoString() string

func (*TokenReservaBaremacion) GobDecode([]byte) error

func (TokenReservaBaremacion) GobEncode() ([]byte, error)

func (t TokenReservaBaremacion) HuellaSHA256() (string, error)
```

HuellaSHA256 devuelve exclusivamente el selector durable historico.
El material de la capacidad no forma parte del valor devuelto.

```go
func (t TokenReservaBaremacion) LogValue() slog.Value

func (TokenReservaBaremacion) MarshalBinary() ([]byte, error)

func (TokenReservaBaremacion) MarshalJSON() ([]byte, error)

func (TokenReservaBaremacion) MarshalText() ([]byte, error)

func (TokenReservaBaremacion) MarshalXML(*xml.Encoder, xml.StartElement) error

func (TokenReservaBaremacion) String() string

func (*TokenReservaBaremacion) UnmarshalBinary([]byte) error

func (*TokenReservaBaremacion) UnmarshalJSON([]byte) error

func (*TokenReservaBaremacion) UnmarshalText([]byte) error

func (*TokenReservaBaremacion) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (t TokenReservaBaremacion) Validar() error

type TransaccionPropuestasLlamamiento interface {
	GuardarPropuestaLlamamiento(
		context.Context,
		dominiobolsa.PropuestaLlamamiento,
		puertosvec.EvidenciaUsoDecisionAutorizacion,
	) error
}
```

TransaccionPropuestasLlamamiento consume la concesion y confirma el
efecto de negocio en una unica operacion. Un adaptador duradero debe
bloquear/releer necesidad y vigencia, validar la atestacion del PDP y
hacer COMMIT de propuesta, unicidad de la version inmutable de necesidad,
uso unico de decision, auditoria y outbox de forma indivisible. La clave
gobernada minima es (necesidad_ref, version_necesidad, huella_necesidad);
una reapertura debe ser explicita y producir otra version, nunca reemplazar
esta propuesta. La implementacion en memoria solo cubre semantica e
idempotencia para tests.

```go
type TrazabilidadCambioBaremacion struct {
	MotivoClave string
	Motivo      string
}
```

TrazabilidadCambioBaremacion solo admite la motivacion de negocio. Actor,
accion, modulo, sujeto, versiones y huellas los deriva el repositorio; no se
aceptan AuditEntry, Event ni mapas libres proporcionados por un cliente.

```go
func (t TrazabilidadCambioBaremacion) Validar() error

type ValidacionFirmaServidor struct {
	Estado                                  EstadoValidacionFirma
	Artefacto                               ArtefactoFirma
	ValidacionRef                           string
	HuellaValidacionSHA256                  string
	FirmanteVerificadoRef                   string
	PerfilVerificadoClave                   string
	PerfilFirmaVerificadoClave              string
	SelloTiempoVerificadoRef                string
	HuellaSelloTiempoVerificadaSHA256       string
	AumentoLongevidadVerificadoRef          string
	HuellaAumentoLongevidadVerificadaSHA256 string
	Comprobaciones                          []ComprobacionFirma
	ValidadaEn                              time.Time
}
```

ValidacionFirmaServidor conserva la atestacion del conector sobre una
revision PAdES y las referencias exactas de las evidencias que encontro.

```go
func (v ValidacionFirmaServidor) AptaParaDecision() bool

func (v ValidacionFirmaServidor) AptaParaPerfil(p PoliticaFirmaBaremacion, perfil string) bool
```

AptaParaPerfil permite verificar cada etapa material del flujo B -> T -> LTA
sin confundir el perfil intermedio con el objetivo final de la politica.

```go
func (v ValidacionFirmaServidor) AptaParaPolitica(p PoliticaFirmaBaremacion) bool

func (v ValidacionFirmaServidor) Clonar() (ValidacionFirmaServidor, error)

func (v ValidacionFirmaServidor) Validar() error

func (v ValidacionFirmaServidor) ValidarPara(s SolicitudValidarFirmaServidor) error

func (v ValidacionFirmaServidor) ValidarRecuperacion(s SolicitudRecuperarValidacionFirma) error

type ValidadorFirmaServidor interface {
	ValidarFirmaServidor(context.Context, SolicitudValidarFirmaServidor) (ValidacionFirmaServidor, error)
}

type VerificadorEstadoFlujoFirmaBaremacion interface {
	VerificarEstadoFlujoFirmaBaremacion(context.Context, SolicitudVerificarEstadoFlujoFirmaBaremacion) error
}

type VerificadorIndependienteTestimonioIdempotenciaBaremacion interface {
	VerificarTestimonioAtomicoIdempotenciaBaremacion(
		context.Context,
		SolicitudTestimonioAtomicoIdempotenciaBaremacion,
		FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion,
		FuenteEfimeraClaveClienteIdempotenciaBaremacion,
		VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion,
	) error
}

type VerificadorMaterialEstableIntencionBaremacion interface {
	VerificarMaterialEstableIntencionBaremacion(
		context.Context,
		IntencionCambioBaremacion,
	) error
}
```

VerificadorMaterialEstableIntencionBaremacion, cuando esta fijado como
dependencia privada de la composicion TCB homologada, debe recuperar y
comprobar la representacion V2 del plan durable de firma, manifiesto y
recibos. Tambien debe resolver historicamente ambas instantaneas de catalogo
por referencia+version+huella y acreditar:
  - formato clave <-> MIME canonico <-> perfil/politica de firma <->
    conector;
  - clasificacion <-> reglas de custodia y politica de retencion.

La interfaz es sustituible y no es autoridad por si sola. No existe aun el
servicio de aplicacion con dependencias privadas ni productor V2 homologado:
hasta implementarlos y superar sus pruebas, el flujo permanece en NO-GO.

```go
type VerificadorNoAplicacionBaremacion interface {
	VerificarNoAplicacionBaremacion(context.Context, EvidenciaNoAplicacionBaremacion) error
}
```

VerificadorNoAplicacionBaremacion es la frontera de confianza que comprueba
el sello y contrasta la consulta con la fuente transaccional autoritativa.
Un error, ausencia o implementacion nula nunca produce una conclusion
negativa.

```go
type VerificadorSellosBaremacion interface {
	VerificarSelloBaremacion(context.Context, SolicitudVerificarSelloBaremacion) error
}
```

VerificadorSellosBaremacion debe comparar en tiempo constante y devolver
error ante clave desconocida, indisponibilidad o sello no autentico.

```go
type VerificadorSeparacionDominiosClaveBaremacion interface {
	VerificarSeparacionDominiosClaveBaremacion(
		context.Context,
		SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion,
	) error
}

type VersionBaremacion struct {
	Referencia   ReferenciaVersionBaremacion
	Agregado     dominiobolsa.BaremacionMerito
	ConfirmadaEn time.Time
}

func (v VersionBaremacion) Clonar() (VersionBaremacion, error)

func (v VersionBaremacion) Validar() error

type VinculoAutenticacionBaremacion struct {
	SujetoRef                 string
	Metodo                    dominiovec.AuthMethod
	Garantia                  dominiovec.AuthAssurance
	AutenticacionRef          string
	SesionRef                 string
	SesionEmitidaEn           time.Time
	SesionValidaHasta         time.Time
	VinculoAutenticacionActor dominiovec.VinculoAutenticacionActorV1
}
```

VinculoAutenticacionBaremacion contiene hechos ya verificados de la sesion.
No concede acceso: la unica fuente de concesion aceptada por el constructor
es DecisionAutorizacion.

```go
type VinculoTransicionPAdES struct {
	Referencia   string
	HuellaSHA256 string
}
```

VinculoTransicionPAdES identifica un recibo canónico calculado por el
núcleo. Su huella se incorpora al manifiesto HMAC para impedir que una
evidencia válida se combine con una revisión PDF distinta.

```go
func NuevoVinculoRevisionLongevaPAdES(
	sello SelloTiempoFirma,
	atestacionT ValidacionFirmaServidor,
	aumento ResultadoAumentoFirma,
	atestacionLTA ValidacionFirmaServidor,
) (VinculoTransicionPAdES, error)
```

NuevoVinculoRevisionLongevaPAdES vincula la transición T→LTA con la
atestación que prueba tanto el sello como el aumento embebidos en LTA.

```go
func NuevoVinculoRevisionSelladaPAdES(
	sello SelloTiempoFirma,
	atestacion ValidacionFirmaServidor,
) (VinculoTransicionPAdES, error)
```

NuevoVinculoRevisionSelladaPAdES vincula la transición B→T con la atestación
que prueba el token TSA embebido en esa misma revisión T.

```go
func (v VinculoTransicionPAdES) ValidarParaTipo(tipo string) error

func (v VinculoTransicionPAdES) ValidarRevisionLongevaPara(
	sello SelloTiempoFirma,
	atestacionT ValidacionFirmaServidor,
	aumento ResultadoAumentoFirma,
	atestacionLTA ValidacionFirmaServidor,
) error
```

ValidarRevisionLongevaPara recompone el recibo desde el material original y
falla cerrado ante cualquier sustitución de revisión o evidencia.

```go
func (v VinculoTransicionPAdES) ValidarRevisionSelladaPara(
	sello SelloTiempoFirma,
	atestacion ValidacionFirmaServidor,
) error
```

ValidarRevisionSelladaPara recompone el recibo desde el material original;
no acepta una pareja referencia/huella válida pero perteneciente a otra
transición.

```go
type VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion interface {
	VisitarMaterialCanonicoAtestadoBaremacion(func(MaterialCanonicoEfimeroBaremacion) error) error
	VisitarResolucionIdentidadInternaEstableBaremacion(
		func(string, uint64, string, string, string, string, []byte) error,
	) error
	ResumenLlaveroIdentidadesBaremacion() (string, uint64, uint8, string, error)
	VisitarTopologiaIdentidadesBaremacion(func(int, uint16, uint32, string) error) error
	ResumenLlaveroIndicesBaremacion() (string, uint64, uint8, string, error)
	VisitarTopologiaIndicesBaremacion(func(int, uint16, uint32, string) error) error
	VisitarPrincipalesBaremacion(func(int, uint16, uint32, string, string) error) error
	VisitarMatrizIndicesBaremacion(func(int, int, uint16, uint32, string, string) error) error
	VisitarRepresentacionesCanonicasIntencionBaremacion(
		SolicitudTestimonioAtomicoIdempotenciaBaremacion,
		IntencionCambioBaremacion,
		[]byte,
		func(int, int, MaterialCanonicoEfimeroBaremacion, MaterialCanonicoEfimeroBaremacion) error,
	) error
	VisitarEvidenciaAtestacionBaremacion(func(string, string, string, uint64, string, []byte) error) error
}
```

VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion
permite a una raiz independiente cotejar la topologia esperada y verificar
la evidencia. Solo vive durante la llamada; no ofrece selectores ni devuelve
el producto. La efimeridad limita el ciclo de vida, pero codigo malicioso
en el mismo proceso puede copiar bytes o cadenas dentro del callback. Si ese
actor forma parte del modelo de amenaza, DEC-047 exige adaptador aislado en
otro proceso.
