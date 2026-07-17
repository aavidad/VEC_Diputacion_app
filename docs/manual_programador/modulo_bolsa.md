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

## Paquete `internal/modules/bolsa/adapters/catalogosvec`

> Package catalogosvec adapta catalogos configurables gobernados por el nucleo a las proyecciones publicas minimizadas del modulo Bolsa.

Package catalogosvec adapta catalogos configurables gobernados por el nucleo a
las proyecciones publicas minimizadas del modulo Bolsa.

### Tipos

```go
type ConsultaCategorias struct {
	// Has unexported fields.
}
```

ConsultaCategorias fija el ID y la version al construirse. Cambiar de
version es una decision de configuracion explicita; nunca se consulta "la
ultima" de forma implicita.

```go
func NuevaConsultaCategorias(
	fuente fuenteCatalogosPublicos,
	catalogoID string,
	version int,
) (*ConsultaCategorias, error)

func (c *ConsultaCategorias) ObtenerPublicadas(
	ctx context.Context,
	instante time.Time,
) (puertosbolsa.CatalogoCategoriasPublicas, error)
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
const (
	RutaConvocatorias = "/api/publico/bolsa/convocatorias"
	RutaCategorias    = "/api/publico/bolsa/categorias"
)
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
	comando puertosbolsa.ComandoGuardarPropuestaLlamamiento,
) error

func (r *RegistroPropuestasLlamamiento) NumeroPropuestasParaPruebas() int

func (r *RegistroPropuestasLlamamiento) ObtenerInstantaneaParaPruebas(
	ctx context.Context,
	referenciaPropuesta string,
) (dominiobolsa.InstantaneaOrdenBolsa, error)
```

ObtenerInstantaneaParaPruebas no forma parte del puerto productivo. Permite
demostrar que el adaptador conserva el orden completo, incluido el tramo no
evaluado que PropuestaLlamamiento no contiene.

```go
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

La consulta de gobierno de convocatorias es deliberadamente transaccional:
incluso una lectura consume una decision de autorizacion y deja auditoria. La
cuenta de ejecucion solo puede invocar la funcion SECURITY DEFINER de contrato
cerrado; no recibe permisos directos sobre las tablas.

La persistencia de propuestas de llamamiento es deliberadamente una sola
transaccion durable. La cuenta runtime no toca tablas: invoca una funcion
SECURITY DEFINER de contrato cerrado que vuelve a comprobar toda autoridad.

La cuenta de ejecucion del panel solo puede invocar la funcion SECURITY DEFINER
de contrato cerrado. No recibe SELECT sobre tablas de convocatorias, bolsas,
llamamientos, auditoria ni identidad.

### Variables

```go
var (
	ErrFuentePanelInternoPostgreSQLNoDisponible = errors.New(
		"bolsa: fuente PostgreSQL del panel interno no disponible",
	)
	ErrConsultaPanelInternoPostgreSQLEnCurso = errors.New(
		"bolsa: consulta PostgreSQL del panel interno en curso",
	)
)
```

### Tipos

```go
type ConsultaGobiernoConvocatoriasPostgreSQL struct {
	// Has unexported fields.
}
```

ConsultaGobiernoConvocatoriasPostgreSQL recupera una version exacta.
No contiene busquedas por "ultima", listados amplios ni rutas degradadas.

```go
func NuevaConsultaGobiernoConvocatoriasPostgreSQL(
	pool *pgxpool.Pool,
) (*ConsultaGobiernoConvocatoriasPostgreSQL, error)

func (r *ConsultaGobiernoConvocatoriasPostgreSQL) ObtenerVersionExacta(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudConsultaVersionConvocatoriaAutorizada,
) (puertosbolsa.ResultadoConsultaVersionConvocatoria, error)

type ConsultaPanelInternoPostgreSQL struct {
	// Has unexported fields.
}
```

ConsultaPanelInternoPostgreSQL consume una autorizacion V2 y obtiene una
instantanea agregada dentro de la misma transaccion durable.

```go
func NuevaConsultaPanelInternoPostgreSQL(
	pool *pgxpool.Pool,
) (*ConsultaPanelInternoPostgreSQL, error)

func (r *ConsultaPanelInternoPostgreSQL) ConsultarPanel(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudConsultaPanelInterno,
) (puertosbolsa.InstantaneaPanelInterno, error)

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

type TransaccionPropuestasLlamamientoPostgreSQL struct {
	// Has unexported fields.
}
```

TransaccionPropuestasLlamamientoPostgreSQL permanece cerrada aunque
exista la funcion guardar_propuesta_v1: ese contrato antiguo no recibe ni
confirma la instantanea completa generada. Conceder EXECUTE no basta para
habilitarla.

```go
func NuevaTransaccionPropuestasLlamamientoPostgreSQL(
	pool *pgxpool.Pool,
	reloj puertosbolsa.RelojLlamamientos,
) (*TransaccionPropuestasLlamamientoPostgreSQL, error)

func (r *TransaccionPropuestasLlamamientoPostgreSQL) GuardarPropuestaLlamamiento(
	ctx context.Context,
	comando puertosbolsa.ComandoGuardarPropuestaLlamamiento,
) error
```

GuardarPropuestaLlamamiento valida el nuevo comando indivisible y falla
cerrado antes de iniciar una transaccion. TODO(produccion): sustituir
guardar_propuesta_v1 por un contrato SQL nuevo que inserte la instantanea
completa, todas sus entradas, el prefijo de evaluaciones, la propuesta,
el consumo de autorizacion, la atestacion COSE, auditoria y outbox en un
unico COMMIT; solo entonces podra retirarse este cierre explicito.

## Paquete `internal/modules/bolsa/adapters/referencias`

> Package referencias contiene emisores productivos de identificadores opacos del modulo de Bolsa.

Package referencias contiene emisores productivos de identificadores opacos
del modulo de Bolsa. No incorpora identidad, datos del expediente ni claves de
negocio a las referencias.

### Tipos

```go
type GeneradorCriptograficoLlamamientos struct {
	// Has unexported fields.
}
```

GeneradorCriptograficoLlamamientos implementa el puerto de referencias
con el CSPRNG del sistema. El lector queda privado para impedir que la
composicion productiva inyecte identificadores previsibles.

La codificacion base64url sin relleno conserva 256 bits de entropia.
La validacion posterior evita devolver, incluso por coincidencia accidental,
texto que el dominio considere un documento personal evidente.

```go
func NuevoGeneradorCriptograficoLlamamientos() GeneradorCriptograficoLlamamientos
```

NuevoGeneradorCriptograficoLlamamientos fija crypto/rand.Reader como unica
fuente productiva de entropia. El valor cero falla cerrado.

```go
func (g GeneradorCriptograficoLlamamientos) NuevaReferenciaInstantaneaOrdenBolsa() (string, error)

func (g GeneradorCriptograficoLlamamientos) NuevaReferenciaPropuestaLlamamiento() (string, error)
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
var (
	ErrServicioConsultaConvocatoriaInvalido  = errors.New("bolsa: servicio de consulta interna de convocatoria invalido")
	ErrOrdenConsultaConvocatoriaInvalida     = errors.New("bolsa: orden de consulta interna de convocatoria invalida")
	ErrResultadoConsultaConvocatoriaInseguro = errors.New("bolsa: resultado de consulta interna de convocatoria no confiable")
)
var (
	ErrServicioPanelInternoInvalido  = errors.New("bolsa: servicio de panel interno invalido")
	ErrOrdenPanelInternoInvalida     = errors.New("bolsa: orden de panel interno invalida")
	ErrDatosPanelInternoNoConfiables = errors.New("bolsa: datos de panel interno no confiables")
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

type CategoriaDirectorioPublico struct {
	Clave                string `json:"clave"`
	Version              int    `json:"version"`
	Etiqueta             string `json:"etiqueta"`
	Descripcion          string `json:"descripcion,omitempty"`
	Semantica            string `json:"semantica"`
	Orden                int    `json:"orden"`
	Area                 string `json:"area"`
	AreaEtiqueta         string `json:"area_etiqueta"`
	Suscribible          bool   `json:"suscribible"`
	NumeroConvocatorias  int    `json:"numero_convocatorias"`
	NumeroPlazosAbiertos int    `json:"numero_plazos_abiertos"`
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

type DirectorioCategoriasPublicas struct {
	Esquema    string                              `json:"esquema"`
	Fuente     FuentePublica                       `json:"fuente"`
	Catalogo   ReferenciaCatalogoCategoriasPublico `json:"catalogo"`
	Categorias []CategoriaDirectorioPublico        `json:"categorias"`
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

type FacetaCategoriaPublica struct {
	Clave            string `json:"clave"`
	Version          int    `json:"version"`
	Etiqueta         string `json:"etiqueta"`
	Descripcion      string `json:"descripcion,omitempty"`
	Semantica        string `json:"semantica"`
	NumeroResultados int    `json:"numero_resultados"`
}

type FacetasConvocatorias struct {
	Tipos      []ValorCatalogoPublico   `json:"tipos"`
	Categorias []FacetaCategoriaPublica `json:"categorias"`
	Estados    []ValorCatalogoPublico   `json:"estados"`
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
type OrdenConsultaPanelInterno struct {
	ContextoActor             dominiovec.ContextoActor
	VinculoAutenticacionActor dominiovec.VinculoAutenticacionActorV1
	Selector                  puertosbolsa.SelectorPanelInterno
	MotivoCatalogo            dominiovec.ReferenciaEntradaCatalogo
	Correlacion               dominiovec.ReferenciaCorrelacionAutorizacionV2
}
```

OrdenConsultaPanelInterno recibe capacidades resueltas por la frontera
interna. No admite roles, permisos ni garantia declarados por el cliente.

```go
type OrdenConsultaVersionConvocatoria struct {
	ContextoActor             dominiovec.ContextoActor
	VinculoAutenticacionActor dominiovec.VinculoAutenticacionActorV1
	Selector                  puertosbolsa.SelectorVersionConvocatoriaExacta
	IncluirInstanciaFlujo     bool
	CorrelacionRef            string
}
```

OrdenConsultaVersionConvocatoria contiene capacidades resueltas por la
frontera interna. Ninguno de estos campos debe reconstruirse desde JSON.

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

type ReferenciaCatalogoCategoriasPublico struct {
	Referencia   string `json:"referencia"`
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
	Total        int    `json:"total"`
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
type ServicioConsultaPanelInterno struct {
	// Has unexported fields.
}
```

ServicioConsultaPanelInterno es el PEP del cuadro operativo de Bolsa.
Solo acepta sesion interna de garantia alta y una concesion PDP ligada V2.

```go
func NuevoServicioConsultaPanelInterno(
	consulta puertosbolsa.ConsultaPanelInterno,
	exigidor aplicacionvec.ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	reloj puertosvec.Reloj,
) (*ServicioConsultaPanelInterno, error)

func (s *ServicioConsultaPanelInterno) Consultar(
	ctx context.Context,
	orden OrdenConsultaPanelInterno,
) (puertosbolsa.InstantaneaPanelInterno, error)

type ServicioConsultaPublica struct {
	// Has unexported fields.
}

func NuevoServicioConsultaPublica(
	fuente puertosbolsa.ConsultaConvocatoriasPublicas,
	categorias puertosbolsa.ConsultaCategoriasPublicas,
	reloj RelojConsultaPublica,
) (*ServicioConsultaPublica, error)

func (s *ServicioConsultaPublica) Listar(ctx context.Context, solicitud SolicitudListadoPublico) (ListadoConvocatoriasPublicas, error)

func (s *ServicioConsultaPublica) ListarCategorias(ctx context.Context) (DirectorioCategoriasPublicas, error)

func (s *ServicioConsultaPublica) Obtener(ctx context.Context, identificador string) (DetalleConvocatoriaPublica, error)

func (s *ServicioConsultaPublica) ValidarConfiguracion(ctx context.Context) error
```

ValidarConfiguracion coteja de forma anticipada la instantanea profesional
con todas las convocatorias publicadas. El bootstrap debe ejecutarlo antes
de montar las rutas: una referencia desconocida o una version/huella
diferente impide arrancar en vez de convertirse despues en un error 500.

```go
type ServicioConsultaVersionConvocatoria struct {
	// Has unexported fields.
}
```

ServicioConsultaVersionConvocatoria compone identidad, PEP, PDP y lectura
durable. El adaptador PostgreSQL vuelve a revalidar y consumir la decision
en la misma transaccion que deja la auditoria.

```go
func NuevoServicioConsultaVersionConvocatoria(
	consulta puertosbolsa.ConsultaGobiernoConvocatorias,
	exigidor aplicacionvec.ExigidorEvidenciaUsoDecisionAutorizacion,
	reloj puertosvec.Reloj,
) (*ServicioConsultaVersionConvocatoria, error)

func (s *ServicioConsultaVersionConvocatoria) ObtenerExacta(
	ctx context.Context,
	orden OrdenConsultaVersionConvocatoria,
) (puertosbolsa.ResultadoConsultaVersionConvocatoria, error)

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

## Paquete `internal/modules/bolsa/application/calculoexperienciaoficial`

### Variables

```go
var (
	ErrServicioInvalido = errors.New(
		"bolsa: servicio de calculo oficial de experiencia invalido",
	)
	ErrOrdenInvalida = errors.New(
		"bolsa: orden confiable de calculo oficial de experiencia invalida",
	)
	ErrSesionNoApta = errors.New(
		"bolsa: sesion no apta para calculo oficial de experiencia",
	)
	ErrFuenteNoConfiable = errors.New(
		"bolsa: fuente de calculo oficial de experiencia no confiable",
	)
	ErrMotorNoCoincide = errors.New(
		"bolsa: motor de calculo oficial de experiencia no coincide",
	)
	ErrResultadoNoConfiable = errors.New(
		"bolsa: resultado de calculo oficial de experiencia no confiable",
	)
	ErrConfirmacionInvalida = errors.New(
		"bolsa: confirmacion durable de calculo oficial de experiencia invalida",
	)
	ErrReciboNoConfiable = errors.New(
		"bolsa: recibo de calculo oficial de experiencia no confiable",
	)
	ErrResultadoConfirmacionIndeterminado = errors.New(
		"bolsa: resultado de confirmacion de calculo oficial indeterminado",
	)
	ErrReconciliacionRequerida = errors.New(
		"bolsa: reconciliacion de calculo oficial requerida",
	)
	ErrSerializacionProhibida = errors.New(
		"bolsa: serializacion de capacidad de calculo oficial prohibida",
	)
)
```

### Tipos

```go
type ConfirmadorDuradero interface {
	Confirmar(
		context.Context,
		SolicitudConfirmacionDuradera,
	) (ResultadoConfirmacionDuradera, error)
}

type DatosConfirmacionDuradera struct {
	Perfil                PerfilConfirmacionDuradera
	ReferenciaIntento     string
	Selector              puertosbolsa.SelectorFuenteExactaCalculoReglasBaremo
	Fuente                puertosbolsa.FuenteExactaCalculoReglasBaremo
	Clave                 oficial.ClaveEfectoV1
	Intencion             oficial.IntencionResultadoV1
	Resultado             calculo.ResultadoExperienciaV1
	ResultadoCanonico     []byte
	HuellaResultadoSHA256 string
	AutorizacionLectura   puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	AutorizacionEscritura puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	CorrelacionLectura    dominiovec.ReferenciaCorrelacionAutorizacionV2
	CorrelacionEscritura  dominiovec.ReferenciaCorrelacionAutorizacionV2
	Motivo                dominiovec.ReferenciaEntradaCatalogo
	LecturaNoAntesDe      time.Time
	FuenteSolicitadaEn    time.Time
	EscrituraNoAntesDe    time.Time
	SolicitadaEn          time.Time
	// Has unexported fields.
}

func (b DatosConfirmacionDuradera) Format(estado fmt.State, _ rune)

func (b DatosConfirmacionDuradera) GoString() string

func (*DatosConfirmacionDuradera) GobDecode([]byte) error

func (DatosConfirmacionDuradera) GobEncode() ([]byte, error)

func (b DatosConfirmacionDuradera) LogValue() slog.Value

func (DatosConfirmacionDuradera) MarshalBinary() ([]byte, error)

func (DatosConfirmacionDuradera) MarshalCBOR() ([]byte, error)

func (DatosConfirmacionDuradera) MarshalJSON() ([]byte, error)

func (DatosConfirmacionDuradera) MarshalText() ([]byte, error)

func (DatosConfirmacionDuradera) MarshalXML(*xml.Encoder, xml.StartElement) error

func (DatosConfirmacionDuradera) MarshalYAML() (any, error)

func (DatosConfirmacionDuradera) String() string

func (*DatosConfirmacionDuradera) UnmarshalBinary([]byte) error

func (*DatosConfirmacionDuradera) UnmarshalCBOR([]byte) error

func (*DatosConfirmacionDuradera) UnmarshalJSON([]byte) error

func (*DatosConfirmacionDuradera) UnmarshalText([]byte) error

func (*DatosConfirmacionDuradera) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (*DatosConfirmacionDuradera) UnmarshalYAML(func(any) error) error

type DatosOrdenConfiable struct {
	ContextoActor             dominiovec.ContextoActor
	VinculoAutenticacionActor dominiovec.VinculoAutenticacionActorV1
	Selector                  puertosbolsa.SelectorFuenteExactaCalculoReglasBaremo
	Motivo                    dominiovec.ReferenciaEntradaCatalogo
	CorrelacionLectura        dominiovec.ReferenciaCorrelacionAutorizacionV2
	CorrelacionEscritura      dominiovec.ReferenciaCorrelacionAutorizacionV2
	Causa                     oficial.CausaGobernadaV1
	MotorEsperado             oficial.VinculoMotorV1
	TipoEfecto                oficial.TipoEfectoV1
	Predecesor                *oficial.VinculoPredecesorV1
	// Has unexported fields.
}
```

DatosOrdenConfiable solo admite capacidades obtenidas por la frontera de
identidad y referencias exactas ya resueltas. No contiene DNI ni permite
seleccionar el perfil de proteccion del servicio.

```go
func (b DatosOrdenConfiable) Format(estado fmt.State, _ rune)

func (b DatosOrdenConfiable) GoString() string

func (*DatosOrdenConfiable) GobDecode([]byte) error

func (DatosOrdenConfiable) GobEncode() ([]byte, error)

func (b DatosOrdenConfiable) LogValue() slog.Value

func (DatosOrdenConfiable) MarshalBinary() ([]byte, error)

func (DatosOrdenConfiable) MarshalCBOR() ([]byte, error)

func (DatosOrdenConfiable) MarshalJSON() ([]byte, error)

func (DatosOrdenConfiable) MarshalText() ([]byte, error)

func (DatosOrdenConfiable) MarshalXML(*xml.Encoder, xml.StartElement) error

func (DatosOrdenConfiable) MarshalYAML() (any, error)

func (DatosOrdenConfiable) String() string

func (*DatosOrdenConfiable) UnmarshalBinary([]byte) error

func (*DatosOrdenConfiable) UnmarshalCBOR([]byte) error

func (*DatosOrdenConfiable) UnmarshalJSON([]byte) error

func (*DatosOrdenConfiable) UnmarshalText([]byte) error

func (*DatosOrdenConfiable) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (*DatosOrdenConfiable) UnmarshalYAML(func(any) error) error

type DatosReconciliacionDuradera struct {
	ReferenciaIntento     string
	HuellaIntencionSHA256 string
	// Has unexported fields.
}

func (b DatosReconciliacionDuradera) Format(estado fmt.State, _ rune)

func (b DatosReconciliacionDuradera) GoString() string

func (*DatosReconciliacionDuradera) GobDecode([]byte) error

func (DatosReconciliacionDuradera) GobEncode() ([]byte, error)

func (b DatosReconciliacionDuradera) LogValue() slog.Value

func (DatosReconciliacionDuradera) MarshalBinary() ([]byte, error)

func (DatosReconciliacionDuradera) MarshalCBOR() ([]byte, error)

func (DatosReconciliacionDuradera) MarshalJSON() ([]byte, error)

func (DatosReconciliacionDuradera) MarshalText() ([]byte, error)

func (DatosReconciliacionDuradera) MarshalXML(*xml.Encoder, xml.StartElement) error

func (DatosReconciliacionDuradera) MarshalYAML() (any, error)

func (DatosReconciliacionDuradera) String() string

func (*DatosReconciliacionDuradera) UnmarshalBinary([]byte) error

func (*DatosReconciliacionDuradera) UnmarshalCBOR([]byte) error

func (*DatosReconciliacionDuradera) UnmarshalJSON([]byte) error

func (*DatosReconciliacionDuradera) UnmarshalText([]byte) error

func (*DatosReconciliacionDuradera) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (*DatosReconciliacionDuradera) UnmarshalYAML(func(any) error) error

type DesenlaceConfirmacionDuradera string

const (
	ConfirmacionCreada      DesenlaceConfirmacionDuradera = "creada"
	ConfirmacionReutilizada DesenlaceConfirmacionDuradera = "reutilizada"
)
type ErrorConfirmacionIndeterminada struct {
	// Has unexported fields.
}
```

ErrorConfirmacionIndeterminada evita propagar errores de driver tras una
frontera COMMIT ambigua y conserva la unica capacidad segura de consulta.

```go
func (*ErrorConfirmacionIndeterminada) Error() string

func (b ErrorConfirmacionIndeterminada) Format(estado fmt.State, _ rune)

func (b ErrorConfirmacionIndeterminada) GoString() string

func (*ErrorConfirmacionIndeterminada) GobDecode([]byte) error

func (ErrorConfirmacionIndeterminada) GobEncode() ([]byte, error)

func (e *ErrorConfirmacionIndeterminada) Intento() (
	IntentoReconciliacionCalculoOficial,
	error,
)

func (e *ErrorConfirmacionIndeterminada) LogValue() slog.Value

func (ErrorConfirmacionIndeterminada) MarshalBinary() ([]byte, error)

func (ErrorConfirmacionIndeterminada) MarshalCBOR() ([]byte, error)

func (ErrorConfirmacionIndeterminada) MarshalJSON() ([]byte, error)

func (ErrorConfirmacionIndeterminada) MarshalText() ([]byte, error)

func (ErrorConfirmacionIndeterminada) MarshalXML(*xml.Encoder, xml.StartElement) error

func (ErrorConfirmacionIndeterminada) MarshalYAML() (any, error)

func (ErrorConfirmacionIndeterminada) String() string

func (*ErrorConfirmacionIndeterminada) UnmarshalBinary([]byte) error

func (*ErrorConfirmacionIndeterminada) UnmarshalCBOR([]byte) error

func (*ErrorConfirmacionIndeterminada) UnmarshalJSON([]byte) error

func (*ErrorConfirmacionIndeterminada) UnmarshalText([]byte) error

func (*ErrorConfirmacionIndeterminada) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (*ErrorConfirmacionIndeterminada) UnmarshalYAML(func(any) error) error

func (*ErrorConfirmacionIndeterminada) Unwrap() []error

type IntentoReconciliacionCalculoOficial struct {
	// Has unexported fields.
}
```

IntentoReconciliacionCalculoOficial es una capacidad opaca creada antes de
cruzar la frontera que puede enviar COMMIT. No concede permiso para repetir
la confirmacion; solo permite consultar el mismo intento nominal.

NO-GO REANUDACION EXTERNA: esta capacidad es deliberadamente in-memory y no
puede serializarse ni reconstruirse solo desde ReferenciaOpaca. Produccion
necesita un worker/acuse durable autenticado que recupere el intento tras un
reinicio y vuelva a comprobar autoridad sin aceptar un identificador libre.

```go
func (b IntentoReconciliacionCalculoOficial) Format(estado fmt.State, _ rune)

func (b IntentoReconciliacionCalculoOficial) GoString() string

func (*IntentoReconciliacionCalculoOficial) GobDecode([]byte) error

func (IntentoReconciliacionCalculoOficial) GobEncode() ([]byte, error)

func (b IntentoReconciliacionCalculoOficial) LogValue() slog.Value

func (IntentoReconciliacionCalculoOficial) MarshalBinary() ([]byte, error)

func (IntentoReconciliacionCalculoOficial) MarshalCBOR() ([]byte, error)

func (IntentoReconciliacionCalculoOficial) MarshalJSON() ([]byte, error)

func (IntentoReconciliacionCalculoOficial) MarshalText() ([]byte, error)

func (IntentoReconciliacionCalculoOficial) MarshalXML(*xml.Encoder, xml.StartElement) error

func (IntentoReconciliacionCalculoOficial) MarshalYAML() (any, error)

func (i IntentoReconciliacionCalculoOficial) ReferenciaOpaca() (string, error)
```

ReferenciaOpaca devuelve el identificador nominal emitido antes del COMMIT.
No contiene sujeto, convocatoria ni datos personales.

```go
func (IntentoReconciliacionCalculoOficial) String() string

func (*IntentoReconciliacionCalculoOficial) UnmarshalBinary([]byte) error

func (*IntentoReconciliacionCalculoOficial) UnmarshalCBOR([]byte) error

func (*IntentoReconciliacionCalculoOficial) UnmarshalJSON([]byte) error

func (*IntentoReconciliacionCalculoOficial) UnmarshalText([]byte) error

func (*IntentoReconciliacionCalculoOficial) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (*IntentoReconciliacionCalculoOficial) UnmarshalYAML(func(any) error) error

type OrdenCalculoExperienciaOficial struct {
	// Has unexported fields.
}

func NuevaOrdenConfiable(
	datos DatosOrdenConfiable,
) (OrdenCalculoExperienciaOficial, error)

func (o OrdenCalculoExperienciaOficial) Format(estado fmt.State, _ rune)

func (o OrdenCalculoExperienciaOficial) GoString() string

func (*OrdenCalculoExperienciaOficial) GobDecode([]byte) error

func (OrdenCalculoExperienciaOficial) GobEncode() ([]byte, error)

func (o OrdenCalculoExperienciaOficial) LogValue() slog.Value

func (OrdenCalculoExperienciaOficial) MarshalBinary() ([]byte, error)

func (OrdenCalculoExperienciaOficial) MarshalCBOR() ([]byte, error)

func (OrdenCalculoExperienciaOficial) MarshalJSON() ([]byte, error)

func (OrdenCalculoExperienciaOficial) MarshalText() ([]byte, error)

func (OrdenCalculoExperienciaOficial) MarshalXML(*xml.Encoder, xml.StartElement) error

func (OrdenCalculoExperienciaOficial) MarshalYAML() (any, error)

func (OrdenCalculoExperienciaOficial) String() string

func (*OrdenCalculoExperienciaOficial) UnmarshalBinary([]byte) error

func (*OrdenCalculoExperienciaOficial) UnmarshalCBOR([]byte) error

func (*OrdenCalculoExperienciaOficial) UnmarshalJSON(datos []byte) error

func (*OrdenCalculoExperienciaOficial) UnmarshalText([]byte) error

func (*OrdenCalculoExperienciaOficial) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (*OrdenCalculoExperienciaOficial) UnmarshalYAML(func(any) error) error

type PerfilConfirmacionDuradera string

const (
	PerfilConfirmacionExternoOrdinario PerfilConfirmacionDuradera = "externo_ordinario"
	PerfilConfirmacionInternoAlto      PerfilConfirmacionDuradera = "interno_alto"
)
type ReconciliadorDuradero interface {
	Reconciliar(
		context.Context,
		SolicitudReconciliacionDuradera,
	) (ResultadoConfirmacionDuradera, error)
}

type ResultadoConfirmacionDuradera struct {
	// Has unexported fields.
}

func NuevoResultadoConfirmacionDuradera(
	referenciaIntento string,
	recibo oficial.ReciboV1,
	indiceEfectoHMACSHA256 string,
	huellaResultadoSHA256 string,
	desenlace DesenlaceConfirmacionDuradera,
) (ResultadoConfirmacionDuradera, error)

func (r ResultadoConfirmacionDuradera) Format(estado fmt.State, _ rune)

func (r ResultadoConfirmacionDuradera) GoString() string

func (*ResultadoConfirmacionDuradera) GobDecode([]byte) error

func (ResultadoConfirmacionDuradera) GobEncode() ([]byte, error)

func (r ResultadoConfirmacionDuradera) LogValue() slog.Value

func (ResultadoConfirmacionDuradera) MarshalBinary() ([]byte, error)

func (ResultadoConfirmacionDuradera) MarshalCBOR() ([]byte, error)

func (ResultadoConfirmacionDuradera) MarshalJSON() ([]byte, error)

func (ResultadoConfirmacionDuradera) MarshalText() ([]byte, error)

func (ResultadoConfirmacionDuradera) MarshalXML(*xml.Encoder, xml.StartElement) error

func (ResultadoConfirmacionDuradera) MarshalYAML() (any, error)

func (ResultadoConfirmacionDuradera) String() string

func (*ResultadoConfirmacionDuradera) UnmarshalBinary([]byte) error

func (*ResultadoConfirmacionDuradera) UnmarshalCBOR([]byte) error

func (*ResultadoConfirmacionDuradera) UnmarshalJSON([]byte) error

func (*ResultadoConfirmacionDuradera) UnmarshalText([]byte) error

func (*ResultadoConfirmacionDuradera) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (*ResultadoConfirmacionDuradera) UnmarshalYAML(func(any) error) error

func (r ResultadoConfirmacionDuradera) ValidarParaReconciliacion(
	solicitud SolicitudReconciliacionDuradera,
) error

type ResultadoEjecucion struct {
	// Has unexported fields.
}

func (r ResultadoEjecucion) Desenlace() (DesenlaceConfirmacionDuradera, error)

func (r ResultadoEjecucion) Format(estado fmt.State, _ rune)

func (r ResultadoEjecucion) GoString() string

func (*ResultadoEjecucion) GobDecode([]byte) error

func (ResultadoEjecucion) GobEncode() ([]byte, error)

func (r ResultadoEjecucion) LogValue() slog.Value

func (ResultadoEjecucion) MarshalBinary() ([]byte, error)

func (ResultadoEjecucion) MarshalCBOR() ([]byte, error)

func (ResultadoEjecucion) MarshalJSON() ([]byte, error)

func (ResultadoEjecucion) MarshalText() ([]byte, error)

func (ResultadoEjecucion) MarshalXML(*xml.Encoder, xml.StartElement) error

func (ResultadoEjecucion) MarshalYAML() (any, error)

func (r ResultadoEjecucion) Recibo() (oficial.ReciboV1, error)

func (r ResultadoEjecucion) Resultado() (calculo.ResultadoExperienciaV1, error)

func (ResultadoEjecucion) String() string

func (*ResultadoEjecucion) UnmarshalBinary([]byte) error

func (*ResultadoEjecucion) UnmarshalCBOR([]byte) error

func (*ResultadoEjecucion) UnmarshalJSON(datos []byte) error

func (*ResultadoEjecucion) UnmarshalText([]byte) error

func (*ResultadoEjecucion) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (*ResultadoEjecucion) UnmarshalYAML(func(any) error) error

type Servicio struct {
	// Has unexported fields.
}

func NuevoServicioExternoOrdinario(
	fuente puertosbolsa.FuenteReglasBaremoParaCalculo,
	exigidor aplicacionvec.ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	confirmador ConfirmadorDuradero,
	reconciliador ReconciliadorDuradero,
	reloj puertosvec.Reloj,
) (*Servicio, error)

func NuevoServicioInternoAlto(
	fuente puertosbolsa.FuenteReglasBaremoParaCalculo,
	exigidor aplicacionvec.ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	confirmador ConfirmadorDuradero,
	reconciliador ReconciliadorDuradero,
	reloj puertosvec.Reloj,
) (*Servicio, error)

func (s *Servicio) Ejecutar(
	ctx context.Context,
	orden OrdenCalculoExperienciaOficial,
) (ResultadoEjecucion, error)

func (s *Servicio) Reconciliar(
	ctx context.Context,
	intento IntentoReconciliacionCalculoOficial,
) (ResultadoEjecucion, error)
```

Reconciliar consulta un intento previo; nunca vuelve a llamar a Confirmar.

```go
type SolicitudConfirmacionDuradera struct {
	// Has unexported fields.
}
```

SolicitudConfirmacionDuradera es opaca para impedir que una frontera de
transporte reconstruya una escritura oficial sin pasar por el caso de uso.

```go
func (s SolicitudConfirmacionDuradera) Datos() (DatosConfirmacionDuradera, error)

func (s SolicitudConfirmacionDuradera) Format(estado fmt.State, _ rune)

func (s SolicitudConfirmacionDuradera) GoString() string

func (*SolicitudConfirmacionDuradera) GobDecode([]byte) error

func (SolicitudConfirmacionDuradera) GobEncode() ([]byte, error)

func (s SolicitudConfirmacionDuradera) LogValue() slog.Value

func (SolicitudConfirmacionDuradera) MarshalBinary() ([]byte, error)

func (SolicitudConfirmacionDuradera) MarshalCBOR() ([]byte, error)

func (SolicitudConfirmacionDuradera) MarshalJSON() ([]byte, error)

func (SolicitudConfirmacionDuradera) MarshalText() ([]byte, error)

func (SolicitudConfirmacionDuradera) MarshalXML(*xml.Encoder, xml.StartElement) error

func (SolicitudConfirmacionDuradera) MarshalYAML() (any, error)

func (SolicitudConfirmacionDuradera) String() string

func (*SolicitudConfirmacionDuradera) UnmarshalBinary([]byte) error

func (*SolicitudConfirmacionDuradera) UnmarshalCBOR([]byte) error

func (*SolicitudConfirmacionDuradera) UnmarshalJSON(datos []byte) error

func (*SolicitudConfirmacionDuradera) UnmarshalText([]byte) error

func (*SolicitudConfirmacionDuradera) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (*SolicitudConfirmacionDuradera) UnmarshalYAML(func(any) error) error

type SolicitudReconciliacionDuradera struct {
	// Has unexported fields.
}
```

SolicitudReconciliacionDuradera no admite un DTO libre: se deriva de la
correlacion V2 y la intencion oficial exactas que originaron el intento.

```go
func NuevaSolicitudReconciliacionDuradera(
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2,
	intencion oficial.IntencionResultadoV1,
) (SolicitudReconciliacionDuradera, error)

func (s SolicitudReconciliacionDuradera) Datos() (DatosReconciliacionDuradera, error)

func (s SolicitudReconciliacionDuradera) Format(estado fmt.State, _ rune)

func (s SolicitudReconciliacionDuradera) GoString() string

func (*SolicitudReconciliacionDuradera) GobDecode([]byte) error

func (SolicitudReconciliacionDuradera) GobEncode() ([]byte, error)

func (s SolicitudReconciliacionDuradera) LogValue() slog.Value

func (SolicitudReconciliacionDuradera) MarshalBinary() ([]byte, error)

func (SolicitudReconciliacionDuradera) MarshalCBOR() ([]byte, error)

func (SolicitudReconciliacionDuradera) MarshalJSON() ([]byte, error)

func (SolicitudReconciliacionDuradera) MarshalText() ([]byte, error)

func (SolicitudReconciliacionDuradera) MarshalXML(*xml.Encoder, xml.StartElement) error

func (SolicitudReconciliacionDuradera) MarshalYAML() (any, error)

func (SolicitudReconciliacionDuradera) String() string

func (*SolicitudReconciliacionDuradera) UnmarshalBinary([]byte) error

func (*SolicitudReconciliacionDuradera) UnmarshalCBOR([]byte) error

func (*SolicitudReconciliacionDuradera) UnmarshalJSON([]byte) error

func (*SolicitudReconciliacionDuradera) UnmarshalText([]byte) error

func (*SolicitudReconciliacionDuradera) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (*SolicitudReconciliacionDuradera) UnmarshalYAML(func(any) error) error
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
const (
	AccionBorradorConvocatoriaCreado      = "bolsa.convocatoria.borrador.creado"
	AccionBorradorConvocatoriaActualizado = "bolsa.convocatoria.borrador.actualizado"
	AccionConvocatoriaPublicada           = "bolsa.convocatoria.publicada"
	AccionConvocatoriaSustituida          = "bolsa.convocatoria.sustituida"
	AccionConvocatoriaRetirada            = "bolsa.convocatoria.retirada"
)
const TipoEntidadFlujoConvocatoriaBolsa = "convocatoria_bolsa"
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
	ErrVersionConvocatoriaGobernadaInvalida = errors.New("bolsa: version gobernada de convocatoria invalida")
	ErrTransicionGobiernoConvocatoria       = errors.New("bolsa: transicion de gobierno de convocatoria invalida")
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

type ConfiguracionFijadaConvocatoria struct {
	Catalogos        ReferenciaConfiguracionConvocatoria      `json:"catalogos"`
	Calendario       ReferenciaConfiguracionConvocatoria      `json:"calendario"`
	ReglasBaremacion ReferenciaConfiguracionConvocatoria      `json:"reglas_baremacion"`
	FlujoProceso     ReferenciaConfiguracionConvocatoria      `json:"flujo_proceso"`
	FlujoSolicitud   ReferenciaConfiguracionConvocatoria      `json:"flujo_solicitud"`
	Documentos       []ReferenciaDocumentoOficialConvocatoria `json:"documentos"`
}

func (c ConfiguracionFijadaConvocatoria) ClonarCanonicaPara(
	contenido ContenidoPublicableConvocatoria,
) (ConfiguracionFijadaConvocatoria, error)

func (c ConfiguracionFijadaConvocatoria) ValidarPara(contenido ContenidoPublicableConvocatoria) error

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

type ContenidoPublicableConvocatoria struct {
	IdentificadorPublico string                            `json:"identificador_publico"`
	Tipo                 string                            `json:"tipo"`
	CatalogoCategorias   ReferenciaCatalogoCategorias      `json:"catalogo_categorias"`
	Categorias           []string                          `json:"categorias"`
	Titulo               string                            `json:"titulo"`
	Resumen              string                            `json:"resumen"`
	Descripcion          string                            `json:"descripcion"`
	Plazos               []PlazoConvocatoria               `json:"plazos"`
	Requisitos           []RequisitoConvocatoria           `json:"requisitos"`
	Documentos           []DocumentoPublicableConvocatoria `json:"documentos"`
	Ayuda                []AyudaConvocatoria               `json:"ayuda"`
}
```

ContenidoPublicableConvocatoria no contiene fase ni marcas de publicacion.
Esas piezas proceden, respectivamente, del flujo y del acto de gobierno.

```go
func (c ContenidoPublicableConvocatoria) ClonarCanonico() (ContenidoPublicableConvocatoria, error)

func (c ContenidoPublicableConvocatoria) Validar() error

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
type DatosNuevaVersionConvocatoriaGobernada struct {
	ID                   string
	CodigoVersionPublica string
	InstanciaFlujoRef    string
	Contenido            ContenidoPublicableConvocatoria
	Configuracion        ConfiguracionFijadaConvocatoria
	ExpedienteRef        string
	Motivo               string
	ActorID              string
	Instante             time.Time
}

type DatosPublicosConvocatoria struct {
	IdentificadorPublico string                       `json:"identificador_publico"`
	Tipo                 string                       `json:"tipo"`
	CatalogoCategorias   ReferenciaCatalogoCategorias `json:"catalogo_categorias"`
	Categorias           []string                     `json:"categorias"`
	Titulo               string                       `json:"titulo"`
	Resumen              string                       `json:"resumen"`
	Descripcion          string                       `json:"descripcion"`
	PublicadaEn          time.Time                    `json:"publicada_en"`
	ActualizadaEn        time.Time                    `json:"actualizada_en"`
	Plazos               []PlazoConvocatoria          `json:"plazos"`
	Requisitos           []RequisitoConvocatoria      `json:"requisitos"`
	Documentos           []DocumentoConvocatoria      `json:"documentos"`
	Ayuda                []AyudaConvocatoria          `json:"ayuda"`
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

type DocumentoPublicableConvocatoria struct {
	Referencia  string `json:"referencia"`
	Tipo        string `json:"tipo"`
	Orden       int    `json:"orden"`
	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`
	Formato     string `json:"formato"`
	URL         string `json:"url"`
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
	EstadoConvocatoriaBorrador    EstadoConvocatoria = "borrador"
	EstadoConvocatoriaInscripcion EstadoConvocatoria = "inscripcion"
	EstadoConvocatoriaSubsanacion EstadoConvocatoria = "subsanacion"
	EstadoConvocatoriaAlegaciones EstadoConvocatoria = "alegaciones"
	EstadoConvocatoriaDefinitiva  EstadoConvocatoria = "definitiva"
	EstadoConvocatoriaCerrada     EstadoConvocatoria = "cerrada"
)
```

Estas constantes mantienen la compatibilidad temporal del prototipo
candidate con las claves canonicas del catalogo gobernado. No constituyen
la lista de estados permitidos: cualquier clave valida debe existir en la
definicion de flujo exacta que gobierna la convocatoria.

```go
func (e EstadoConvocatoria) IsValid() bool

type EstadoGobiernoConvocatoria string

const (
	EstadoGobiernoConvocatoriaBorrador   EstadoGobiernoConvocatoria = "borrador"
	EstadoGobiernoConvocatoriaPublicada  EstadoGobiernoConvocatoria = "publicada"
	EstadoGobiernoConvocatoriaSustituida EstadoGobiernoConvocatoria = "sustituida"
	EstadoGobiernoConvocatoriaRetirada   EstadoGobiernoConvocatoria = "retirada"
)
func (e EstadoGobiernoConvocatoria) Valido() bool

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

type EvidenciaAprobacionConvocatoria struct {
	Accion                string    `json:"accion"`
	Referencia            string    `json:"referencia"`
	HuellaEvidenciaSHA256 string    `json:"huella_evidencia_sha256"`
	ConvocatoriaRef       string    `json:"convocatoria_ref"`
	Revision              int       `json:"revision"`
	HuellaContenidoSHA256 string    `json:"huella_contenido_sha256"`
	HuellaEstadoSHA256    string    `json:"huella_estado_sha256"`
	AprobadaPor           string    `json:"aprobada_por"`
	AprobadaEn            time.Time `json:"aprobada_en"`
}

type EvidenciaDependenciasConvocatoria struct {
	Referencia            string    `json:"referencia"`
	HuellaEvidenciaSHA256 string    `json:"huella_evidencia_sha256"`
	ConvocatoriaRef       string    `json:"convocatoria_ref"`
	Revision              int       `json:"revision"`
	HuellaContenidoSHA256 string    `json:"huella_contenido_sha256"`
	HuellaEstadoSHA256    string    `json:"huella_estado_sha256"`
	VerificadaEn          time.Time `json:"verificada_en"`
}

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

type ReferenciaCatalogoCategorias struct {
	CatalogoID           string `json:"catalogo_id"`
	CatalogoVersion      int    `json:"catalogo_version"`
	CatalogoHuellaSHA256 string `json:"catalogo_huella_sha256"`
}
```

ReferenciaCatalogoCategorias inmoviliza la instantanea profesional usada al
publicar una convocatoria. La huella publica de la convocatoria incluye esta
referencia, por lo que otra version nunca puede reinterpretarla de forma
silenciosa.

```go
func (r ReferenciaCatalogoCategorias) Valida() bool

type ReferenciaConfiguracionConvocatoria struct {
	ID                    string `json:"id"`
	Version               int    `json:"version"`
	HuellaContenidoSHA256 string `json:"huella_contenido_sha256"`
}
```

ReferenciaConfiguracionConvocatoria fija identidad, version y contenido.
Nunca representa «la ultima version» de una dependencia.

```go
func (r ReferenciaConfiguracionConvocatoria) ReferenciaVersionada() string

func (r ReferenciaConfiguracionConvocatoria) Validar() error

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

type ReferenciaDocumentoOficialConvocatoria struct {
	Rol                   string `json:"rol"`
	PublicacionRef        string `json:"publicacion_ref"`
	DocumentoRef          string `json:"documento_ref"`
	VersionDocumento      int    `json:"version_documento"`
	RepresentacionRef     string `json:"representacion_ref"`
	HuellaContenidoSHA256 string `json:"huella_contenido_sha256"`
	FirmaValidadaRef      string `json:"firma_validada_ref"`
	ReciboCustodiaRef     string `json:"recibo_custodia_ref"`
}

func (r ReferenciaDocumentoOficialConvocatoria) Validar() error

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

type ResultadoPublicacionSucesoraConvocatoria struct {
	Publicada   VersionConvocatoriaGobernada
	Predecesora VersionConvocatoriaGobernada
}

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

type VersionConvocatoriaGobernada struct {
	ID                       string                             `json:"id"`
	Secuencia                int                                `json:"secuencia"`
	CodigoVersionPublica     string                             `json:"codigo_version_publica"`
	Revision                 int                                `json:"revision"`
	VersionAnteriorRef       string                             `json:"version_anterior_ref,omitempty"`
	InstanciaFlujoRef        string                             `json:"instancia_flujo_ref"`
	Contenido                ContenidoPublicableConvocatoria    `json:"contenido"`
	Configuracion            ConfiguracionFijadaConvocatoria    `json:"configuracion"`
	ExpedienteRef            string                             `json:"expediente_ref"`
	MotivoCreacion           string                             `json:"motivo_creacion"`
	EstadoGobierno           EstadoGobiernoConvocatoria         `json:"estado_gobierno"`
	CreadaPor                string                             `json:"creada_por"`
	CreadaEn                 time.Time                          `json:"creada_en"`
	UltimaModificacionPor    string                             `json:"ultima_modificacion_por,omitempty"`
	UltimaModificacionEn     time.Time                          `json:"ultima_modificacion_en,omitempty"`
	MotivoModificacion       string                             `json:"motivo_modificacion,omitempty"`
	PublicadaPor             string                             `json:"publicada_por,omitempty"`
	PublicadaEn              time.Time                          `json:"publicada_en,omitempty"`
	MotivoPublicacion        string                             `json:"motivo_publicacion,omitempty"`
	AprobacionPublicacion    *EvidenciaAprobacionConvocatoria   `json:"aprobacion_publicacion,omitempty"`
	ComprobacionDependencias *EvidenciaDependenciasConvocatoria `json:"comprobacion_dependencias,omitempty"`
	SustituidaPorRef         string                             `json:"sustituida_por_ref,omitempty"`
	SustituidaPor            string                             `json:"sustituida_por,omitempty"`
	SustituidaEn             time.Time                          `json:"sustituida_en,omitempty"`
	RetiradaPor              string                             `json:"retirada_por,omitempty"`
	RetiradaEn               time.Time                          `json:"retirada_en,omitempty"`
	MotivoRetirada           string                             `json:"motivo_retirada,omitempty"`
	AprobacionRetirada       *EvidenciaAprobacionConvocatoria   `json:"aprobacion_retirada,omitempty"`
}

func DecodificarVersionConvocatoriaGobernadaCanonica(
	contenido []byte,
) (VersionConvocatoriaGobernada, error)
```

DecodificarVersionConvocatoriaGobernadaCanonica reconstruye exclusivamente
los bytes producidos por RepresentacionCanonica. No basta con que el JSON
represente un agregado valido: se rechazan claves desconocidas o duplicadas,
orden alternativo, espacios, formatos temporales equivalentes y cualquier
otra codificacion maleable. Esta frontera se usa al recuperar evidencias
duraderas de un almacenamiento que el dominio no debe confiar a ciegas.

```go
func NuevaVersionConvocatoriaGobernada(datos DatosNuevaVersionConvocatoriaGobernada) (VersionConvocatoriaGobernada, error)

func (v VersionConvocatoriaGobernada) ActualizarBorrador(
	revisionEsperada int,
	contenido ContenidoPublicableConvocatoria,
	configuracion ConfiguracionFijadaConvocatoria,
	actorID, motivo string,
	instante time.Time,
) (VersionConvocatoriaGobernada, error)

func (v VersionConvocatoriaGobernada) ClonarCanonico() (VersionConvocatoriaGobernada, error)

func (v VersionConvocatoriaGobernada) HuellaContenidoSHA256() (string, error)

func (v VersionConvocatoriaGobernada) HuellaSHA256() (string, error)

func (v VersionConvocatoriaGobernada) NuevaVersion(
	codigoVersionPublica string,
	contenido ContenidoPublicableConvocatoria,
	configuracion ConfiguracionFijadaConvocatoria,
	expedienteRef, actorID, motivo string,
	instante time.Time,
) (VersionConvocatoriaGobernada, error)

func (v VersionConvocatoriaGobernada) ProyectarPublica(
	instancia dominiovec.InstanciaFlujo,
	definicion dominiovec.DefinicionFlujo,
) (Convocatoria, error)

func (v VersionConvocatoriaGobernada) PublicarInicial(
	actorID string,
	aprobacion EvidenciaAprobacionConvocatoria,
	dependencias EvidenciaDependenciasConvocatoria,
	motivo string,
	instante time.Time,
) (VersionConvocatoriaGobernada, error)

func (v VersionConvocatoriaGobernada) PublicarSucesora(
	predecesora VersionConvocatoriaGobernada,
	actorID string,
	aprobacion EvidenciaAprobacionConvocatoria,
	dependencias EvidenciaDependenciasConvocatoria,
	motivo string,
	instante time.Time,
) (ResultadoPublicacionSucesoraConvocatoria, error)
```

PublicarSucesora devuelve las dos instantaneas que el repositorio debe
confirmar de forma atomica. Toda la cadena iniciada conserva la misma
instancia y definicion de flujo; cambiar cualquiera exige una migracion
expresa, no una publicacion ordinaria.

```go
func (v VersionConvocatoriaGobernada) Referencia() string

func (v VersionConvocatoriaGobernada) RepresentacionCanonica() ([]byte, error)

func (v VersionConvocatoriaGobernada) RepresentacionContenidoCanonica() ([]byte, error)

func (v VersionConvocatoriaGobernada) Retirar(
	actorID string,
	aprobacion EvidenciaAprobacionConvocatoria,
	motivo string,
	instante time.Time,
) (VersionConvocatoriaGobernada, error)

func (v VersionConvocatoriaGobernada) SustituirPor(
	nueva VersionConvocatoriaGobernada,
) (VersionConvocatoriaGobernada, error)
```

SustituirPor prepara la mutacion de la version anterior. El repositorio debe
confirmar esta copia y la publicacion nueva en una unica transaccion.

```go
func (v VersionConvocatoriaGobernada) Validar() error
```

## Paquete `internal/modules/bolsa/domain/calculoexperiencia`

### Variables

```go
var (
	ErrSeleccionTemporalBloqueada   = &ErrorCalculo{codigo: CodigoSeleccionTemporalBloqueada}
	ErrSeleccionTemporalInvalida    = &ErrorCalculo{codigo: CodigoSeleccionTemporalInvalida}
	ErrLimiteAplicacionesTemporales = &ErrorCalculo{codigo: CodigoLimiteAplicacionesTemporales}
	ErrLimiteEventosTemporales      = &ErrorCalculo{codigo: CodigoLimiteEventosTemporales}
)
var (
	ErrCompilacionConjuntoInvalido  = &ErrorCalculo{codigo: CodigoCompilacionConjuntoInvalido}
	ErrCompilacionPlanInvalido      = &ErrorCalculo{codigo: CodigoCompilacionPlanInvalido}
	ErrUnidadBaseNoSoportada        = &ErrorCalculo{codigo: CodigoUnidadBaseNoSoportada}
	ErrJornadaNoSoportada           = &ErrorCalculo{codigo: CodigoJornadaNoSoportada}
	ErrRedondeoNoSoportado          = &ErrorCalculo{codigo: CodigoRedondeoNoSoportado}
	ErrMinimoSeccionNoSoportado     = &ErrorCalculo{codigo: CodigoMinimoSeccionNoSoportado}
	ErrRestosRedondeoNoSoportados   = &ErrorCalculo{codigo: CodigoRestosRedondeoNoSoportados}
	ErrCoincidenciaNoSoportada      = &ErrorCalculo{codigo: CodigoCoincidenciaNoSoportada}
	ErrSolapeNoSoportado            = &ErrorCalculo{codigo: CodigoSolapeNoSoportado}
	ErrTopeUnidadesNoSoportado      = &ErrorCalculo{codigo: CodigoTopeUnidadesNoSoportado}
	ErrCatalogoCriterioIncompatible = &ErrorCalculo{codigo: CodigoCatalogoCriterioIncompatible}
)
var (
	ErrValorInvalido        = &ErrorCalculo{codigo: CodigoValorInvalido}
	ErrValorNoCanonico      = &ErrorCalculo{codigo: CodigoValorNoCanonico}
	ErrFueraDeLimites       = &ErrorCalculo{codigo: CodigoFueraDeLimites}
	ErrValorDuplicado       = &ErrorCalculo{codigo: CodigoValorDuplicado}
	ErrResultadoNegativo    = &ErrorCalculo{codigo: CodigoResultadoNegativo}
	ErrDivisionPorCero      = &ErrorCalculo{codigo: CodigoDivisionPorCero}
	ErrResultadoNoExacto    = &ErrorCalculo{codigo: CodigoResultadoNoExacto}
	ErrDesbordamiento       = &ErrorCalculo{codigo: CodigoDesbordamiento}
	ErrLimiteOperaciones    = &ErrorCalculo{codigo: CodigoLimiteOperaciones}
	ErrContextoIncompatible = &ErrorCalculo{codigo: CodigoContextoIncompatible}
	ErrModoRedondeoInvalido = &ErrorCalculo{codigo: CodigoModoRedondeoInvalido}
	ErrEsquemaIncompatible  = &ErrorCalculo{codigo: CodigoEsquemaIncompatible}
	ErrHuellaNoCoincide     = &ErrorCalculo{codigo: CodigoHuellaNoCoincide}
)
```

### Tipos

```go
type AplicacionCalculadaResultadoExperienciaV1 struct {
	// Has unexported fields.
}

func (a AplicacionCalculadaResultadoExperienciaV1) Jornada() JornadaResultadoExperienciaV1

func (a AplicacionCalculadaResultadoExperienciaV1) Puntuacion() PuntuacionPeriodoResultadoExperienciaV1

func (a AplicacionCalculadaResultadoExperienciaV1) ReglaClave() string

func (a AplicacionCalculadaResultadoExperienciaV1) Tramo() reglasbaremo.ReferenciaVersionada

func (a AplicacionCalculadaResultadoExperienciaV1) Unidades() UnidadesAplicacionResultadoExperienciaV1

type AplicacionSeleccionResultadoExperienciaV1 struct {
	// Has unexported fields.
}
```

AplicacionSeleccionResultadoExperienciaV1 afirma que todos los criterios
gobernados de la regla ligada por el plan resultaron verdaderos. No repite
claves ni valores catalogados: el plan exacto conserva las primeras y
la huella de entrada liga los segundos sin ampliar datos laborales en la
salida.

```go
func (a AplicacionSeleccionResultadoExperienciaV1) GrupoClave() string

func (a AplicacionSeleccionResultadoExperienciaV1) Prioridad() uint32

func (a AplicacionSeleccionResultadoExperienciaV1) Razon() CodigoRazonResultadoExperienciaV1

func (a AplicacionSeleccionResultadoExperienciaV1) ReglaClave() string

func (a AplicacionSeleccionResultadoExperienciaV1) SeccionClave() string

func (a AplicacionSeleccionResultadoExperienciaV1) Tramo() reglasbaremo.ReferenciaVersionada

type AtributoCatalogado struct {
	// Has unexported fields.
}
```

AtributoCatalogado aporta una clave normalizada y un valor gobernado por una
version exacta de catalogo. No admite descripciones ni texto libre.

```go
func NuevoAtributoCatalogado(
	clave string,
	catalogo reglasbaremo.ReferenciaVersionada,
	valor string,
) (AtributoCatalogado, error)

func (a AtributoCatalogado) Catalogo() reglasbaremo.ReferenciaVersionada

func (a AtributoCatalogado) Clave() string

func (a AtributoCatalogado) Valor() string

type BloqueoResultadoExperienciaV1 struct {
	// Has unexported fields.
}

func (b BloqueoResultadoExperienciaV1) ClaveGobernada() string

func (b BloqueoResultadoExperienciaV1) Codigo() CodigoBloqueoResultadoExperienciaV1

func (b BloqueoResultadoExperienciaV1) GrupoClave() string

func (b BloqueoResultadoExperienciaV1) Reglas() []string

func (b BloqueoResultadoExperienciaV1) SeccionClave() string

func (b BloqueoResultadoExperienciaV1) Tramos() []reglasbaremo.ReferenciaVersionada

func (b BloqueoResultadoExperienciaV1) ValorExacto() (string, bool)

type CodigoBloqueoResultadoExperienciaV1 string

const (
	BloqueoResultadoCatalogoIncompatible  CodigoBloqueoResultadoExperienciaV1 = "catalogo_incompatible"
	BloqueoResultadoGruposDistintos       CodigoBloqueoResultadoExperienciaV1 = "reglas_en_grupos_distintos"
	BloqueoResultadoCoincidenciaRechazada CodigoBloqueoResultadoExperienciaV1 = "coincidencia_reglas_rechazada"
	BloqueoResultadoSolape                CodigoBloqueoResultadoExperienciaV1 = "solape_tramos"
	BloqueoResultadoRedondeoNoExacto      CodigoBloqueoResultadoExperienciaV1 = "redondeo_no_exacto"
)
type CodigoError string
```

CodigoError identifica de forma estable un fallo del calculo exacto. Los
errores nunca incorporan los valores operados ni referencias del expediente.

```go
const (
	CodigoSeleccionTemporalBloqueada   CodigoError = "seleccion_temporal_bloqueada"
	CodigoSeleccionTemporalInvalida    CodigoError = "seleccion_temporal_invalida"
	CodigoLimiteAplicacionesTemporales CodigoError = "limite_aplicaciones_temporales"
	CodigoLimiteEventosTemporales      CodigoError = "limite_eventos_temporales"
)
```

Los errores tecnicos de la fase temporal se separan de sus exclusiones y
bloqueos de negocio. Estos ultimos forman parte del resultado explicable.

```go
const (
	CodigoCompilacionConjuntoInvalido  CodigoError = "compilacion_conjunto_invalido"
	CodigoCompilacionPlanInvalido      CodigoError = "compilacion_plan_invalido"
	CodigoUnidadBaseNoSoportada        CodigoError = "unidad_base_no_soportada"
	CodigoJornadaNoSoportada           CodigoError = "jornada_no_soportada"
	CodigoRedondeoNoSoportado          CodigoError = "redondeo_no_soportado"
	CodigoMinimoSeccionNoSoportado     CodigoError = "minimo_seccion_no_soportado"
	CodigoRestosRedondeoNoSoportados   CodigoError = "restos_redondeo_no_soportados"
	CodigoCoincidenciaNoSoportada      CodigoError = "coincidencia_no_soportada"
	CodigoSolapeNoSoportado            CodigoError = "solape_no_soportado"
	CodigoTopeUnidadesNoSoportado      CodigoError = "tope_unidades_no_soportado"
	CodigoCatalogoCriterioIncompatible CodigoError = "catalogo_criterio_incompatible"
)
```

Los codigos de compilacion separan una configuracion valida pero no
ejecutable por V1 de un fallo aritmetico producido durante el calculo.

```go
const (
	CodigoValorInvalido        CodigoError = "valor_invalido"
	CodigoValorNoCanonico      CodigoError = "valor_no_canonico"
	CodigoFueraDeLimites       CodigoError = "fuera_de_limites"
	CodigoValorDuplicado       CodigoError = "valor_duplicado"
	CodigoResultadoNegativo    CodigoError = "resultado_negativo"
	CodigoDivisionPorCero      CodigoError = "division_por_cero"
	CodigoResultadoNoExacto    CodigoError = "resultado_no_exacto"
	CodigoDesbordamiento       CodigoError = "desbordamiento"
	CodigoLimiteOperaciones    CodigoError = "limite_operaciones"
	CodigoContextoIncompatible CodigoError = "contexto_incompatible"
	CodigoModoRedondeoInvalido CodigoError = "modo_redondeo_invalido"
	CodigoEsquemaIncompatible  CodigoError = "esquema_incompatible"
	CodigoHuellaNoCoincide     CodigoError = "huella_no_coincide"
)
type CodigoRazonResultadoExperienciaV1 string

const (
	RazonCoincidenciaUnica    CodigoRazonResultadoExperienciaV1 = "coincidencia_unica"
	RazonPrioridad            CodigoRazonResultadoExperienciaV1 = "prioridad"
	RazonAcumulacion          CodigoRazonResultadoExperienciaV1 = "acumulacion"
	RazonPrioridadInferior    CodigoRazonResultadoExperienciaV1 = "prioridad_inferior"
	RazonNingunaCoincidencia  CodigoRazonResultadoExperienciaV1 = "ninguna_regla_coincidente"
	RazonPosteriorCorte       CodigoRazonResultadoExperienciaV1 = "posterior_corte"
	RazonIntervaloVacio       CodigoRazonResultadoExperienciaV1 = "intervalo_vacio"
	RazonJornadaProporcional  CodigoRazonResultadoExperienciaV1 = "jornada_proporcional"
	RazonJornadaIntegra       CodigoRazonResultadoExperienciaV1 = "jornada_integra"
	RazonUmbralAlcanzado      CodigoRazonResultadoExperienciaV1 = "umbral_alcanzado"
	RazonUmbralNoAlcanzado    CodigoRazonResultadoExperienciaV1 = "umbral_no_alcanzado"
	RazonProteccionAtestada   CodigoRazonResultadoExperienciaV1 = "proteccion_atestada"
	RazonProteccionNoAtestada CodigoRazonResultadoExperienciaV1 = "proteccion_no_atestada"
)
type ComputoIntegroAtestado struct {
	// Has unexported fields.
}
```

ComputoIntegroAtestado solo informa de la consecuencia computable. Nunca
contiene la causa medica, familiar, sindical o de otra naturaleza.

```go
func NuevoComputoIntegroAtestado(
	referencia reglasbaremo.ReferenciaVersionada,
) (ComputoIntegroAtestado, error)

func SinComputoIntegroAtestado() ComputoIntegroAtestado

func (a ComputoIntegroAtestado) EstaAtestado() bool

func (a ComputoIntegroAtestado) Referencia() (reglasbaremo.ReferenciaVersionada, bool)

type DescarteSeleccionResultadoExperienciaV1 struct {
	// Has unexported fields.
}

func (d DescarteSeleccionResultadoExperienciaV1) GrupoClave() string

func (d DescarteSeleccionResultadoExperienciaV1) Razon() CodigoRazonResultadoExperienciaV1

func (d DescarteSeleccionResultadoExperienciaV1) ReglaClave() string

func (d DescarteSeleccionResultadoExperienciaV1) ReglaSeleccionada() string

func (d DescarteSeleccionResultadoExperienciaV1) Tramo() reglasbaremo.ReferenciaVersionada

type EntradaExperiencia struct {
	// Has unexported fields.
}
```

EntradaExperiencia es la instantanea inmutable y minimizada que consume el
futuro calculador. Una entrada vacia es valida y representa cero hechos.

```go
func NuevaEntradaExperiencia(
	instantanea reglasbaremo.ReferenciaVersionada,
	tramos []TramoExperiencia,
) (EntradaExperiencia, error)

func RestaurarEntradaExperiencia(contenido []byte) (EntradaExperiencia, error)
```

RestaurarEntradaExperiencia solo acepta los mismos bytes que produciria
RepresentacionCanonica. Rechaza campos, orden, espacios y claves duplicadas
alternativos aunque representen aparentemente los mismos datos. Restaurar
y comprobar la huella no autentica la fuente, pertenencia, catalogos ni
atestaciones: antes de calcular, la aplicacion debe obtener una atestacion
externa exacta desde el puerto confiable de servicios y evidencias.

```go
func RestaurarEntradaExperienciaConHuellaSHA256(
	contenido []byte,
	huellaEsperada string,
) (EntradaExperiencia, error)

func (e EntradaExperiencia) HuellaSHA256() (string, error)

func (e EntradaExperiencia) Instantanea() reglasbaremo.ReferenciaVersionada

func (e EntradaExperiencia) MarshalJSON() ([]byte, error)
```

MarshalJSON conserva exactamente la representacion canonica.

```go
func (e EntradaExperiencia) RepresentacionCanonica() ([]byte, error)
```

RepresentacionCanonica devuelve el unico JSON V1 admitido para la entrada.

```go
func (e EntradaExperiencia) Tramos() []TramoExperiencia

func (e EntradaExperiencia) Validar() error

type ErrorCalculo struct {
	// Has unexported fields.
}
```

ErrorCalculo permite clasificar un fallo sin depender de su texto. Campo es
siempre una etiqueta tecnica fija y nunca el valor de entrada rechazado.

```go
func (e *ErrorCalculo) Campo() string
```

Campo devuelve la etiqueta tecnica del elemento rechazado.

```go
func (e *ErrorCalculo) Codigo() CodigoError
```

Codigo devuelve la clasificacion estable del error.

```go
func (e *ErrorCalculo) Error() string

func (e *ErrorCalculo) Is(objetivo error) bool
```

Is permite clasificar el error mediante errors.Is.

```go
type EstadoResultadoExperienciaV1 string

const (
	ResultadoExperienciaCompletado EstadoResultadoExperienciaV1 = "completado"
	ResultadoExperienciaBloqueado  EstadoResultadoExperienciaV1 = "bloqueado"
)
type FaseResultadoExperienciaV1 string

const (
	FaseResultadoSeleccion  FaseResultadoExperienciaV1 = "seleccion"
	FaseResultadoIntervalos FaseResultadoExperienciaV1 = "intervalos"
	FaseResultadoPuntuacion FaseResultadoExperienciaV1 = "puntuacion"
	FaseResultadoCompletado FaseResultadoExperienciaV1 = "completado"
)
type FronteraRestosResultadoExperienciaV1 string

const (
	FronteraRestosResultadoExacta  FronteraRestosResultadoExperienciaV1 = "exacta"
	FronteraRestosResultadoPeriodo FronteraRestosResultadoExperienciaV1 = "periodo"
	FronteraRestosResultadoRegla   FronteraRestosResultadoExperienciaV1 = "regla"
)
type IntervaloAplicacionResultadoExperienciaV1 struct {
	// Has unexported fields.
}

func (i IntervaloAplicacionResultadoExperienciaV1) Dias() uint64

func (i IntervaloAplicacionResultadoExperienciaV1) Efectivo() (baremacion.IntervaloCivil, bool)

func (i IntervaloAplicacionResultadoExperienciaV1) Extremo() reglasbaremo.TratamientoExtremoFinal

func (i IntervaloAplicacionResultadoExperienciaV1) Periodo() PeriodoServicio

func (i IntervaloAplicacionResultadoExperienciaV1) Razon() CodigoRazonResultadoExperienciaV1

func (i IntervaloAplicacionResultadoExperienciaV1) ReglaClave() string

func (i IntervaloAplicacionResultadoExperienciaV1) Tramo() reglasbaremo.ReferenciaVersionada

type JornadaResultadoExperienciaV1 struct {
	// Has unexported fields.
}

func (j JornadaResultadoExperienciaV1) AtestacionPresente() bool

func (j JornadaResultadoExperienciaV1) AtestacionUsada() bool

func (j JornadaResultadoExperienciaV1) FactorExacto() string

func (j JornadaResultadoExperienciaV1) Modo() reglasbaremo.ModoJornada

func (j JornadaResultadoExperienciaV1) Origen() baremacion.FraccionJornada

func (j JornadaResultadoExperienciaV1) Razon() CodigoRazonResultadoExperienciaV1

type ModoPeriodoServicio string
```

ModoPeriodoServicio diferencia de forma expresa un periodo cerrado de uno
que seguia en curso al generar la instantanea.

```go
const (
	PeriodoServicioCerrado ModoPeriodoServicio = "cerrado"
	PeriodoServicioEnCurso ModoPeriodoServicio = "en_curso"
)
type PeriodoServicio struct {
	// Has unexported fields.
}
```

PeriodoServicio conserva las fechas informadas por la fuente. No decide si
el extremo final se incluye: esa decision pertenece a la regla publicada.

```go
func NuevoPeriodoServicioCerrado(
	desde baremacion.FechaCivil,
	finInformado baremacion.FechaCivil,
) (PeriodoServicio, error)
```

NuevoPeriodoServicioCerrado acepta fin igual a inicio porque una regla con
extremo inclusivo puede representar asi un servicio de un dia.

```go
func NuevoPeriodoServicioEnCurso(desde baremacion.FechaCivil) (PeriodoServicio, error)
```

NuevoPeriodoServicioEnCurso no inventa una fecha final provisional.

```go
func (p PeriodoServicio) Desde() baremacion.FechaCivil

func (p PeriodoServicio) EnCurso() bool

func (p PeriodoServicio) FinInformado() (baremacion.FechaCivil, bool)
```

FinInformado devuelve false cuando el servicio estaba en curso.

```go
func (p PeriodoServicio) Modo() ModoPeriodoServicio

type PlanExperiencia struct {
	// Has unexported fields.
}
```

PlanExperiencia es la instantanea cerrada que consumira el calculador V1.
Fija la version y huella exactas de las reglas y conserva solo el material
minimo necesario para rederivar ese vinculo, nunca referencias a las
colecciones recibidas.

El pipeline puntuable es unico: unidades y restos, tope de unidades,
coeficiente y redondeo, tope de puntos de regla, suma y tope de seccion.
Las comprobaciones de elegibilidad, coincidencia y ausencia de solapes
ocurren antes de ese pipeline y nunca cambian su orden.

```go
func Compilar(conjunto reglasbaremo.ConjuntoReglasBaremo) (PlanExperiencia, error)
```

Compilar convierte un conjunto valido en un plan ejecutable por V1. Una
politica que el modelo puede representar pero el calculador aun no gobierna
se rechaza expresamente; nunca se aproxima ni se sustituye por un valor por
defecto. Deliberadamente no comprueba el estado de gobierno: tambien compila
borradores para su simulacion y conformidad. El caso de uso administrativo
oficial debe aportar una FuenteExacta atestada en estado activo.

```go
func (p PlanExperiencia) Conjunto() reglasbaremo.ReferenciaVersionada
```

Conjunto devuelve la referencia, version y huella exactas fijadas al
compilar. No significa nunca "la version vigente".

```go
func (p PlanExperiencia) FechaCorte() baremacion.FechaCivil
```

FechaCorte devuelve el ultimo dia civil incluido por el conjunto.

```go
func (p PlanExperiencia) GruposConcurrencia() []reglasbaremo.GrupoConcurrenciaExperiencia
```

GruposConcurrencia devuelve una copia en el orden canonico del conjunto.

```go
func (p PlanExperiencia) Reglas() []reglasbaremo.ReglaExperiencia
```

Reglas devuelve una copia ordenada. Los criterios de cada regla son tambien
inmutables y sus propios accesores devuelven copias defensivas.

```go
func (p PlanExperiencia) Secciones() []reglasbaremo.SeccionBaremo
```

Secciones devuelve una copia en el orden canonico del conjunto.

```go
func (p PlanExperiencia) Validar() error
```

Validar vuelve a comprobar que el plan conserva las invariantes cerradas de
compilacion. Su valor cero es invalido.

```go
type PuntuacionPeriodoResultadoExperienciaV1 struct {
	// Has unexported fields.
}

func (p PuntuacionPeriodoResultadoExperienciaV1) BrutoExacto() string

func (p PuntuacionPeriodoResultadoExperienciaV1) RedondeadoExacto() (string, bool)

type RedondeoResultadoExperienciaV1 struct {
	// Has unexported fields.
}

func (r RedondeoResultadoExperienciaV1) EntradaExacta() string

func (r RedondeoResultadoExperienciaV1) Modo() baremacion.ModoRedondeo

func (r RedondeoResultadoExperienciaV1) Momento() reglasbaremo.MomentoRedondeo

func (r RedondeoResultadoExperienciaV1) SalidaExacta() string

type ResultadoExperienciaV1 struct {
	// Has unexported fields.
}
```

ResultadoExperienciaV1 es el recibo semantico inmutable del motor.
No contiene identidad directa, modo oficial/simulacion, actor ni instante de
ejecucion. Esos datos pertenecen al recibo administrativo que lo envuelva.
Un error tecnico nunca produce este agregado: devuelve su valor cero y
error.

```go
func CalcularExperienciaV1(
	plan PlanExperiencia,
	entrada EntradaExperiencia,
) (ResultadoExperienciaV1, error)
```

CalcularExperienciaV1 ejecuta de extremo a extremo el contrato puro del
motor. No decide si el plan esta administrativamente activo: esa garantia
corresponde al caso de uso oficial, mientras que una simulacion puede usar
un borrador compilable. Ante un fallo tecnico nunca devuelve material
parcial; los impedimentos de negocio forman un resultado bloqueado, canonico
y explicable.

```go
func RestaurarResultadoExperienciaV1(contenido []byte) (ResultadoExperienciaV1, error)
```

RestaurarResultadoExperienciaV1 acepta exclusivamente los mismos bytes que
produce RepresentacionCanonica. La restauracion estructural no autentica la
ejecucion; el caso oficial debe exigir ademas huella y prueba confiables.

```go
func RestaurarResultadoExperienciaV1ConHuellaSHA256(
	contenido []byte,
	huellaEsperada string,
) (ResultadoExperienciaV1, error)

func (r ResultadoExperienciaV1) Aplicaciones() []AplicacionCalculadaResultadoExperienciaV1

func (r ResultadoExperienciaV1) Bloqueos() []BloqueoResultadoExperienciaV1

func (r ResultadoExperienciaV1) Estado() EstadoResultadoExperienciaV1

func (r ResultadoExperienciaV1) Fase() FaseResultadoExperienciaV1

func (r ResultadoExperienciaV1) HuellaSHA256() (string, error)

func (r ResultadoExperienciaV1) Intervalos() []IntervaloAplicacionResultadoExperienciaV1

func (r ResultadoExperienciaV1) MarshalJSON() ([]byte, error)

func (r ResultadoExperienciaV1) Reglas() []ResultadoReglaExperienciaV1

func (r ResultadoExperienciaV1) RepresentacionCanonica() ([]byte, error)
```

RepresentacionCanonica devuelve el unico JSON V1 admitido. No incorpora
datos de ejecucion: simulacion y calculo oficial comparten bytes si sus dos
instantaneas semanticas son identicas.

```go
func (r ResultadoExperienciaV1) Secciones() []SubtotalSeccionResultadoExperienciaV1

func (r ResultadoExperienciaV1) Seleccion() SeleccionResultadoExperienciaV1

func (r ResultadoExperienciaV1) Total() (baremacion.Puntos, bool)

func (r ResultadoExperienciaV1) Validar() error
```

Validar comprueba estructura, presencia por fase y aritmetica registrada.
No reinterpreta criterios, divisor temporal ni umbral de jornada:
esa verificacion exige el PlanExperiencia y la EntradaExperiencia exactos
ligados y se realizara reejecutando el unico motor o comparando su prueba
confiable.

```go
func (r ResultadoExperienciaV1) Vinculos() VinculosResultadoExperienciaV1

type ResultadoReglaExperienciaV1 struct {
	// Has unexported fields.
}

func (r ResultadoReglaExperienciaV1) BrutoExacto() string

func (r ResultadoReglaExperienciaV1) Coeficiente() baremacion.Puntos

func (r ResultadoReglaExperienciaV1) PuntosFinalesExactos() string

func (r ResultadoReglaExperienciaV1) Redondeo() RedondeoResultadoExperienciaV1

func (r ResultadoReglaExperienciaV1) ReglaClave() string

func (r ResultadoReglaExperienciaV1) RestoRegla() string

func (r ResultadoReglaExperienciaV1) SeccionClave() string

func (r ResultadoReglaExperienciaV1) TopePuntos() TopeResultadoExperienciaV1

func (r ResultadoReglaExperienciaV1) TopeUnidades() TopeResultadoExperienciaV1

func (r ResultadoReglaExperienciaV1) UnidadesAgregadas() string

func (r ResultadoReglaExperienciaV1) UnidadesTrasRestos() string

type SeleccionResultadoExperienciaV1 struct {
	// Has unexported fields.
}

func (s SeleccionResultadoExperienciaV1) Aplicaciones() []AplicacionSeleccionResultadoExperienciaV1

func (s SeleccionResultadoExperienciaV1) Descartes() []DescarteSeleccionResultadoExperienciaV1

func (s SeleccionResultadoExperienciaV1) Evaluaciones() uint64

func (s SeleccionResultadoExperienciaV1) SinCoincidencia() []SinCoincidenciaResultadoExperienciaV1

type SinCoincidenciaResultadoExperienciaV1 struct {
	// Has unexported fields.
}

func (s SinCoincidenciaResultadoExperienciaV1) Razon() CodigoRazonResultadoExperienciaV1

func (s SinCoincidenciaResultadoExperienciaV1) Tramo() reglasbaremo.ReferenciaVersionada

type SubtotalSeccionResultadoExperienciaV1 struct {
	// Has unexported fields.
}

func (s SubtotalSeccionResultadoExperienciaV1) AntesTopeExacto() string

func (s SubtotalSeccionResultadoExperienciaV1) PuntosFinales() baremacion.Puntos

func (s SubtotalSeccionResultadoExperienciaV1) SeccionClave() string

func (s SubtotalSeccionResultadoExperienciaV1) Tope() TopeResultadoExperienciaV1

type TopeResultadoExperienciaV1 struct {
	// Has unexported fields.
}

func (t TopeResultadoExperienciaV1) Antes() string

func (t TopeResultadoExperienciaV1) Aplicado() bool

func (t TopeResultadoExperienciaV1) Despues() string

func (t TopeResultadoExperienciaV1) Limite() (string, bool)

type TramoExperiencia struct {
	// Has unexported fields.
}
```

TramoExperiencia es un hecho temporal minimo. servicioRef permite reconocer
tramos procedentes del mismo servicio sin revelar identidad ni empleador.
Sigue siendo un seudonimo sujeto a las mismas medidas de proteccion que el
resto del expediente; no convierte el dato en anonimo. Los prefijos y la
carga hexadecimal solo fijan el formato: el adaptador de fuente debe generar
el token en servidor con aleatoriedad o seudonimizacion institucional que no
permita probar identificadores por diccionario.

```go
func NuevoTramoExperiencia(
	referencia reglasbaremo.ReferenciaVersionada,
	servicioRef string,
	periodo PeriodoServicio,
	jornada baremacion.FraccionJornada,
	atestacion ComputoIntegroAtestado,
	atributos []AtributoCatalogado,
) (TramoExperiencia, error)

func (t TramoExperiencia) Atestacion() ComputoIntegroAtestado

func (t TramoExperiencia) Atributos() []AtributoCatalogado

func (t TramoExperiencia) Jornada() baremacion.FraccionJornada

func (t TramoExperiencia) Periodo() PeriodoServicio

func (t TramoExperiencia) Referencia() reglasbaremo.ReferenciaVersionada

func (t TramoExperiencia) ServicioRef() string

type UnidadesAplicacionResultadoExperienciaV1 struct {
	// Has unexported fields.
}

func (u UnidadesAplicacionResultadoExperienciaV1) Aportadas() string

func (u UnidadesAplicacionResultadoExperienciaV1) Exactas() string

func (u UnidadesAplicacionResultadoExperienciaV1) Frontera() FronteraRestosResultadoExperienciaV1

func (u UnidadesAplicacionResultadoExperienciaV1) Resto() string

type VinculoEntradaResultadoExperienciaV1 struct {
	// Has unexported fields.
}

func (v VinculoEntradaResultadoExperienciaV1) HuellaContenidoSHA256() string

func (v VinculoEntradaResultadoExperienciaV1) Instantanea() reglasbaremo.ReferenciaVersionada

type VinculoMotorResultadoExperienciaV1 struct {
	// Has unexported fields.
}

func (v VinculoMotorResultadoExperienciaV1) Contrato() string

func (v VinculoMotorResultadoExperienciaV1) HuellaContratoSHA256() string

func (v VinculoMotorResultadoExperienciaV1) Version() uint64

type VinculoPlanResultadoExperienciaV1 struct {
	// Has unexported fields.
}

func (v VinculoPlanResultadoExperienciaV1) Esquema() string

func (v VinculoPlanResultadoExperienciaV1) HuellaSHA256() string

type VinculosResultadoExperienciaV1 struct {
	// Has unexported fields.
}

func (v VinculosResultadoExperienciaV1) Conjunto() reglasbaremo.ReferenciaVersionada

func (v VinculosResultadoExperienciaV1) Entrada() VinculoEntradaResultadoExperienciaV1

func (v VinculosResultadoExperienciaV1) FechaCorte() baremacion.FechaCivil

func (v VinculosResultadoExperienciaV1) Motor() VinculoMotorResultadoExperienciaV1

func (v VinculosResultadoExperienciaV1) Plan() VinculoPlanResultadoExperienciaV1
```

## Paquete `internal/modules/bolsa/domain/calculoexperienciaoficial`

### Constantes

```go
const EsquemaSelectorFuenteExactaCalculoReglasBaremoV1 = "vec.bolsa.calculo-experiencia.selector-fuente-exacta.v1"
```

### Variables

```go
var (
	ErrValorInvalido       = &ErrorDominio{codigo: CodigoValorInvalido}
	ErrValorNoCanonico     = &ErrorDominio{codigo: CodigoValorNoCanonico}
	ErrFueraDeLimites      = &ErrorDominio{codigo: CodigoFueraDeLimites}
	ErrEsquemaIncompatible = &ErrorDominio{codigo: CodigoEsquemaIncompatible}
	ErrHuellaNoCoincide    = &ErrorDominio{codigo: CodigoHuellaNoCoincide}
	ErrEstadoIncompatible  = &ErrorDominio{codigo: CodigoEstadoIncompatible}
	ErrSecretoInvalido     = &ErrorDominio{codigo: CodigoSecretoInvalido}
	ErrEntradaNoPermitida  = &ErrorDominio{codigo: CodigoEntradaNoPermitida}
)
var ErrSelectorFuenteExactaCalculoReglasBaremoInvalido = errors.New("selector de fuente exacta para calculo de reglas de baremo invalido")
```

### Funciones

```go
func CalcularIndiceHMACSHA256(clave ClaveEfectoV1, secretoServidor []byte) (string, error)
```

CalcularIndiceHMACSHA256 deriva el índice durable sin exponer el pseudónimo
ni permitir que una clave elegida por el cliente controle la idempotencia.

### Tipos

```go
type CausaGobernadaV1 struct {
	Catalogo ReferenciaExactaV1 `json:"catalogo"`
	Clave    string             `json:"clave"`
}
```

CausaGobernadaV1 evita incorporar motivos libres a la identidad semántica.

```go
type ClaveEfectoV1 struct {
	// Has unexported fields.
}
```

ClaveEfectoV1 identifica un único efecto oficial por sus entradas
semánticas. Excluye actor, sesión, autorizaciones, correlaciones, tiempos,
auditoría y cualquier dato personal directo.

```go
func NuevaClaveEfectoV1(datos DatosClaveEfectoV1) (ClaveEfectoV1, error)

func RestaurarClaveEfectoV1(contenido []byte) (ClaveEfectoV1, error)

func RestaurarClaveEfectoV1ConHuellaSHA256(
	contenido []byte,
	huellaEsperada string,
) (ClaveEfectoV1, error)

func (c ClaveEfectoV1) Causa() CausaGobernadaV1

func (c ClaveEfectoV1) Convocatoria() ReferenciaExactaV1

func (c ClaveEfectoV1) Datos() (DatosClaveEfectoV1, error)

func (c ClaveEfectoV1) Entrada() VinculoEntradaV1

func (c ClaveEfectoV1) Format(estado fmt.State, _ rune)

func (c ClaveEfectoV1) GoString() string

func (c ClaveEfectoV1) HuellaPlanSHA256() string

func (c ClaveEfectoV1) HuellaSHA256() (string, error)

func (c ClaveEfectoV1) LogValue() slog.Value

func (c ClaveEfectoV1) MarshalJSON() ([]byte, error)

func (c ClaveEfectoV1) Motor() VinculoMotorV1

func (c ClaveEfectoV1) Predecesor() (VinculoPredecesorV1, bool)

func (c ClaveEfectoV1) Reglas() VinculoReglasV1

func (c ClaveEfectoV1) RepresentacionCanonica() ([]byte, error)

func (ClaveEfectoV1) String() string

func (c ClaveEfectoV1) SujetoPseudonimizado() ReferenciaExactaV1

func (c ClaveEfectoV1) Tipo() TipoEfectoV1

func (*ClaveEfectoV1) UnmarshalJSON([]byte) error
```

UnmarshalJSON impide crear un valor cero aparente saltándose la restauración
estricta, que es la única frontera de entrada admitida.

```go
func (c ClaveEfectoV1) Validar() error

type CodigoError string
```

CodigoError identifica de forma estable un rechazo del contrato oficial.
Los errores no incorporan valores recibidos para evitar filtraciones.

```go
const (
	CodigoValorInvalido       CodigoError = "valor_invalido"
	CodigoValorNoCanonico     CodigoError = "valor_no_canonico"
	CodigoFueraDeLimites      CodigoError = "fuera_de_limites"
	CodigoEsquemaIncompatible CodigoError = "esquema_incompatible"
	CodigoHuellaNoCoincide    CodigoError = "huella_no_coincide"
	CodigoEstadoIncompatible  CodigoError = "estado_incompatible"
	CodigoSecretoInvalido     CodigoError = "secreto_invalido"
	CodigoEntradaNoPermitida  CodigoError = "entrada_no_permitida"
)
type DatosClaveEfectoV1 struct {
	// SujetoPseudonimizado es la ReferenciaVersionada exacta emitida por la
	// fuente confiable; nunca un DNI, nombre, correo ni pseudónimo sin huella.
	SujetoPseudonimizado ReferenciaExactaV1
	Convocatoria         ReferenciaExactaV1
	Reglas               VinculoReglasV1
	Entrada              VinculoEntradaV1
	Motor                VinculoMotorV1
	HuellaPlanSHA256     string
	Causa                CausaGobernadaV1
	Tipo                 TipoEfectoV1
	Predecesor           *VinculoPredecesorV1
}

type ErrorDominio struct {
	// Has unexported fields.
}
```

ErrorDominio permite clasificar fallos sin depender de su texto.

```go
func (e *ErrorDominio) Campo() string
```

Campo es una etiqueta técnica fija, nunca el valor rechazado.

```go
func (e *ErrorDominio) Codigo() CodigoError

func (e *ErrorDominio) Error() string

func (e *ErrorDominio) Is(objetivo error) bool

type EstadoResultadoV1 string

const (
	ResultadoCompletado EstadoResultadoV1 = "completado"
	ResultadoBloqueado  EstadoResultadoV1 = "bloqueado"
)
type FaseResultadoV1 string

const (
	FaseSeleccion  FaseResultadoV1 = "seleccion"
	FaseIntervalos FaseResultadoV1 = "intervalos"
	FasePuntuacion FaseResultadoV1 = "puntuacion"
	FaseCompletado FaseResultadoV1 = "completado"
)
type IntencionResultadoV1 struct {
	// Has unexported fields.
}
```

IntencionResultadoV1 liga la clave semántica completa con el resultado
exacto que se pretende confirmar. No porta autoridad ni contexto de sesión.

```go
func NuevaIntencionResultadoV1(
	clave ClaveEfectoV1,
	huellaResultadoSHA256 string,
	estado EstadoResultadoV1,
	fase FaseResultadoV1,
) (IntencionResultadoV1, error)

func RestaurarIntencionResultadoV1(contenido []byte) (IntencionResultadoV1, error)

func RestaurarIntencionResultadoV1ConHuellaSHA256(
	contenido []byte,
	huellaEsperada string,
) (IntencionResultadoV1, error)

func (i IntencionResultadoV1) Clave() ClaveEfectoV1

func (i IntencionResultadoV1) Estado() EstadoResultadoV1

func (i IntencionResultadoV1) Fase() FaseResultadoV1

func (i IntencionResultadoV1) Format(estado fmt.State, _ rune)

func (i IntencionResultadoV1) GoString() string

func (i IntencionResultadoV1) HuellaResultadoSHA256() string

func (i IntencionResultadoV1) HuellaSHA256() (string, error)

func (i IntencionResultadoV1) LogValue() slog.Value

func (i IntencionResultadoV1) MarshalJSON() ([]byte, error)

func (i IntencionResultadoV1) RepresentacionCanonica() ([]byte, error)

func (IntencionResultadoV1) String() string

func (*IntencionResultadoV1) UnmarshalJSON([]byte) error

func (i IntencionResultadoV1) Validar() error

func (i IntencionResultadoV1) ValidarPara(
	clave ClaveEfectoV1,
	huellaResultadoSHA256 string,
	estado EstadoResultadoV1,
	fase FaseResultadoV1,
) error

type ReciboV1 struct {
	// Has unexported fields.
}
```

ReciboV1 es el comprobante mínimo e inmutable del efecto confirmado.
No duplica la intención ni incorpora actor, autoridad, auditoría o tiempos.

```go
func NuevoReciboV1(
	referencia string,
	generacionClaveHMAC uint32,
	indiceEfectoHMACSHA256 string,
	intencion IntencionResultadoV1,
) (ReciboV1, error)

func RestaurarReciboV1(contenido []byte) (ReciboV1, error)

func RestaurarReciboV1ConHuellaSHA256(
	contenido []byte,
	huellaEsperada string,
) (ReciboV1, error)

func (r ReciboV1) Estado() EstadoResultadoV1

func (r ReciboV1) Fase() FaseResultadoV1

func (r ReciboV1) Format(estado fmt.State, _ rune)

func (r ReciboV1) GeneracionClaveHMAC() uint32

func (r ReciboV1) GoString() string

func (r ReciboV1) HuellaClaveEfectoSHA256() string

func (r ReciboV1) HuellaIntencionSHA256() string

func (r ReciboV1) HuellaResultadoSHA256() string

func (r ReciboV1) HuellaSHA256() (string, error)

func (r ReciboV1) IndiceHMACSHA256() string

func (r ReciboV1) LogValue() slog.Value

func (r ReciboV1) MarshalJSON() ([]byte, error)

func (r ReciboV1) Referencia() string

func (r ReciboV1) RepresentacionCanonica() ([]byte, error)

func (ReciboV1) String() string

func (*ReciboV1) UnmarshalJSON([]byte) error

func (r ReciboV1) Validar() error

func (r ReciboV1) ValidarPara(
	indiceEfectoHMACSHA256 string,
	intencion IntencionResultadoV1,
) error

func (r ReciboV1) VinculoPredecesor() (VinculoPredecesorV1, error)

type ReferenciaExactaV1 struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}
```

ReferenciaExactaV1 nunca significa «la vigente»: fija identidad, versión y
contenido. Replica el contrato léxico de ReferenciaVersionada de reglas;
la capa confiable debe emitir referencias opacas, nunca datos personales.

```go
type SelectorFuenteExactaCalculoReglasBaremo struct {
	EstadoReglas       reglas.VinculoEstadoReglasBaremo
	InstantaneaEntrada reglas.ReferenciaVersionada
	SujetoPseudonimo   reglas.ReferenciaVersionada
	Convocatoria       reglas.ReferenciaVersionada
}
```

SelectorFuenteExactaCalculoReglasBaremo liga el calculo a reglas, entrada,
sujeto pseudonimizado y convocatoria exactos. No admite identificadores
personales directos ni resolucion temporal de versiones.

```go
func (s SelectorFuenteExactaCalculoReglasBaremo) HuellaSHA256V1() (string, error)
```

HuellaSHA256V1 identifica el esquema y todos los campos exactos del
selector.

```go
func (s SelectorFuenteExactaCalculoReglasBaremo) RepresentacionCanonicaV1() ([]byte, error)
```

RepresentacionCanonicaV1 es el unico material que deben compartir la capa de
aplicacion y los adaptadores al ligar una autorizacion a este selector.

```go
func (s SelectorFuenteExactaCalculoReglasBaremo) Validar() error
```

Validar comprueba que cada identidad versionada y el estado de reglas pueden
reconstruirse exactamente. No resuelve alias ni completa valores ausentes.

```go
type TipoEfectoV1 string

const (
	EfectoCalculoInicial TipoEfectoV1 = "calculo_inicial"
	EfectoRectificacion  TipoEfectoV1 = "rectificacion"
)
type VinculoEntradaV1 struct {
	Instantanea           ReferenciaExactaV1 `json:"instantanea"`
	HuellaContenidoSHA256 string             `json:"huella_contenido_sha256"`
}

type VinculoMotorV1 struct {
	Contrato             string `json:"contrato"`
	Version              uint64 `json:"version"`
	HuellaContratoSHA256 string `json:"huella_contrato_sha256"`
}

type VinculoPredecesorV1 struct {
	ReferenciaRecibo   string `json:"referencia_recibo"`
	HuellaReciboSHA256 string `json:"huella_recibo_sha256"`
}
```

VinculoPredecesorV1 identifica un recibo oficial inmutable por referencia y
huella. Solo puede aparecer en una rectificación.

```go
type VinculoReglasV1 struct {
	Contenido          ReferenciaExactaV1 `json:"contenido"`
	Revision           uint64             `json:"revision"`
	HuellaEstadoSHA256 string             `json:"huella_estado_sha256"`
}
```

VinculoReglasV1 fija tanto el contenido como el estado gobernado exacto.

## Paquete `internal/modules/bolsa/domain/reglasbaremo`

### Variables

```go
var (
	ErrValorInvalido       = &ErrorModelo{codigo: CodigoValorInvalido}
	ErrValorNoCanonico     = &ErrorModelo{codigo: CodigoValorNoCanonico}
	ErrFueraDeLimites      = &ErrorModelo{codigo: CodigoFueraDeLimites}
	ErrValorDuplicado      = &ErrorModelo{codigo: CodigoValorDuplicado}
	ErrPoliticaIncompleta  = &ErrorModelo{codigo: CodigoPoliticaIncompleta}
	ErrSeccionDesconocida  = &ErrorModelo{codigo: CodigoSeccionDesconocida}
	ErrGrupoDesconocido    = &ErrorModelo{codigo: CodigoGrupoDesconocido}
	ErrCoeficienteAusente  = &ErrorModelo{codigo: CodigoCoeficienteAusente}
	ErrInvarianteQuebrada  = &ErrorModelo{codigo: CodigoInvarianteQuebrada}
	ErrEsquemaIncompatible = &ErrorModelo{codigo: CodigoEsquemaIncompatible}
	ErrHuellaNoCoincide    = &ErrorModelo{codigo: CodigoHuellaNoCoincide}
)
var (
	ErrGobiernoValorInvalido       = &ErrorGobierno{codigo: CodigoGobiernoValorInvalido}
	ErrGobiernoEstadoInvalido      = &ErrorGobierno{codigo: CodigoGobiernoEstadoInvalido}
	ErrGobiernoRevisionConflicto   = &ErrorGobierno{codigo: CodigoGobiernoRevisionConflicto}
	ErrGobiernoTransicionProhibida = &ErrorGobierno{codigo: CodigoGobiernoTransicionProhibida}
	ErrGobiernoEvidenciaInvalida   = &ErrorGobierno{codigo: CodigoGobiernoEvidenciaInvalida}
	ErrGobiernoVinculoInexacto     = &ErrorGobierno{codigo: CodigoGobiernoVinculoInexacto}
	ErrGobiernoInstanteInvalido    = &ErrorGobierno{codigo: CodigoGobiernoInstanteInvalido}
	ErrGobiernoInvarianteQuebrada  = &ErrorGobierno{codigo: CodigoGobiernoInvarianteQuebrada}
)
```

### Tipos

```go
type AccionGobiernoReglasBaremo string
```

AccionGobiernoReglasBaremo forma parte del vinculo de autoridad. No es un
texto configurable ni permite inventar nuevas transiciones desde datos.

```go
const (
	AccionPublicarReglasBaremo  AccionGobiernoReglasBaremo = "publicar"
	AccionActivarReglasBaremo   AccionGobiernoReglasBaremo = "activar"
	AccionSustituirReglasBaremo AccionGobiernoReglasBaremo = "sustituir"
	AccionRetirarReglasBaremo   AccionGobiernoReglasBaremo = "retirar"
	AccionDescartarReglasBaremo AccionGobiernoReglasBaremo = "descartar"
)
type AtestacionAprobacionFirmadaReglasBaremo struct {
	// Has unexported fields.
}
```

AtestacionAprobacionFirmadaReglasBaremo es una afirmacion estructurada de
un verificador externo. El dominio liga referencias, huellas y tiempos,
pero no verifica certificados ni convierte una huella SHA-256 en una firma.

```go
func NuevaAtestacionAprobacionFirmadaReglasBaremo(
	datos DatosAtestacionAprobacionFirmadaReglasBaremo,
) (AtestacionAprobacionFirmadaReglasBaremo, error)

func (a AtestacionAprobacionFirmadaReglasBaremo) Atestacion() ReferenciaVersionada

func (a AtestacionAprobacionFirmadaReglasBaremo) Firma() ReferenciaVersionada

func (a AtestacionAprobacionFirmadaReglasBaremo) FirmadaEn() time.Time

func (a AtestacionAprobacionFirmadaReglasBaremo) Firmantes() []string

func (a AtestacionAprobacionFirmadaReglasBaremo) PoliticaFirma() ReferenciaVersionada

func (a AtestacionAprobacionFirmadaReglasBaremo) ValidaHasta() time.Time

func (a AtestacionAprobacionFirmadaReglasBaremo) VerificadaEn() time.Time

func (a AtestacionAprobacionFirmadaReglasBaremo) Vinculo() VinculoEstadoReglasBaremo

type AtestacionAutoridadReglasBaremo struct {
	// Has unexported fields.
}
```

AtestacionAutoridadReglasBaremo demuestra estructuralmente que una autoridad
externa autorizo una transicion terminal exacta. Su autenticidad pertenece
al adaptador de verificacion, no a este valor puro.

```go
func NuevaAtestacionAutoridadReglasBaremo(
	datos DatosAtestacionAutoridadReglasBaremo,
) (AtestacionAutoridadReglasBaremo, error)

func (a AtestacionAutoridadReglasBaremo) Accion() AccionGobiernoReglasBaremo

func (a AtestacionAutoridadReglasBaremo) Atestacion() ReferenciaVersionada

func (a AtestacionAutoridadReglasBaremo) EmitidaEn() time.Time

func (a AtestacionAutoridadReglasBaremo) PrincipalRef() string

func (a AtestacionAutoridadReglasBaremo) Relacionada() (ReferenciaVersionada, bool)

func (a AtestacionAutoridadReglasBaremo) ValidaHasta() time.Time

func (a AtestacionAutoridadReglasBaremo) Vinculo() VinculoEstadoReglasBaremo

type AtestacionDependenciasVigentesReglasBaremo struct {
	// Has unexported fields.
}
```

AtestacionDependenciasVigentesReglasBaremo liga la activacion a la version
exacta de convocatoria, bases y todas las referencias del contenido.

```go
func NuevaAtestacionDependenciasVigentesReglasBaremo(
	datos DatosAtestacionDependenciasVigentesReglasBaremo,
) (AtestacionDependenciasVigentesReglasBaremo, error)

func (a AtestacionDependenciasVigentesReglasBaremo) Atestacion() ReferenciaVersionada

func (a AtestacionDependenciasVigentesReglasBaremo) Bases() ReferenciaVersionada

func (a AtestacionDependenciasVigentesReglasBaremo) Convocatoria() ReferenciaVersionada

func (a AtestacionDependenciasVigentesReglasBaremo) Dependencias() []ReferenciaVersionada

func (a AtestacionDependenciasVigentesReglasBaremo) ValidaHasta() time.Time

func (a AtestacionDependenciasVigentesReglasBaremo) VerificadaEn() time.Time

func (a AtestacionDependenciasVigentesReglasBaremo) VerificadorRef() string

func (a AtestacionDependenciasVigentesReglasBaremo) Vinculo() VinculoEstadoReglasBaremo

type CodigoError string
```

CodigoError identifica de forma estable por que se rechazo una
configuracion. Los errores no incorporan los valores recibidos para evitar
que una referencia sensible termine accidentalmente en un registro.

```go
const (
	CodigoValorInvalido       CodigoError = "valor_invalido"
	CodigoValorNoCanonico     CodigoError = "valor_no_canonico"
	CodigoFueraDeLimites      CodigoError = "fuera_de_limites"
	CodigoValorDuplicado      CodigoError = "valor_duplicado"
	CodigoPoliticaIncompleta  CodigoError = "politica_incompleta"
	CodigoSeccionDesconocida  CodigoError = "seccion_desconocida"
	CodigoGrupoDesconocido    CodigoError = "grupo_desconocido"
	CodigoCoeficienteAusente  CodigoError = "coeficiente_ausente"
	CodigoInvarianteQuebrada  CodigoError = "invariante_quebrada"
	CodigoEsquemaIncompatible CodigoError = "esquema_incompatible"
	CodigoHuellaNoCoincide    CodigoError = "huella_no_coincide"
)
type CodigoErrorGobierno string
```

CodigoErrorGobierno clasifica rechazos del ciclo de gobierno sin incorporar
referencias, actores ni otros valores de entrada a los mensajes de error.

```go
const (
	CodigoGobiernoValorInvalido       CodigoErrorGobierno = "valor_invalido"
	CodigoGobiernoEstadoInvalido      CodigoErrorGobierno = "estado_invalido"
	CodigoGobiernoRevisionConflicto   CodigoErrorGobierno = "revision_conflicto"
	CodigoGobiernoTransicionProhibida CodigoErrorGobierno = "transicion_prohibida"
	CodigoGobiernoEvidenciaInvalida   CodigoErrorGobierno = "evidencia_invalida"
	CodigoGobiernoVinculoInexacto     CodigoErrorGobierno = "vinculo_inexacto"
	CodigoGobiernoInstanteInvalido    CodigoErrorGobierno = "instante_invalido"
	CodigoGobiernoInvarianteQuebrada  CodigoErrorGobierno = "invariante_quebrada"
)
type ConjuntoReglasBaremo struct {
	// Has unexported fields.
}
```

ConjuntoReglasBaremo es la version inmutable de las reglas de una
convocatoria. Este primer corte solo contiene reglas de experiencia;
no calcula ni publica resultados.

```go
func NuevoConjuntoReglasBaremo(
	identidad IdentidadConjuntoReglasBaremo,
	bases ReferenciaVersionada,
	fechaCorte baremacion.FechaCivil,
	secciones []SeccionBaremo,
	gruposConcurrencia []GrupoConcurrenciaExperiencia,
	reglasExperiencia []ReglaExperiencia,
) (ConjuntoReglasBaremo, error)
```

NuevoConjuntoReglasBaremo construye, valida y ordena una instantanea.
Las colecciones recibidas se copian y dejan de pertenecer al agregado.

```go
func RestaurarConjuntoReglasBaremo(contenido []byte) (ConjuntoReglasBaremo, error)
```

RestaurarConjuntoReglasBaremo reconstruye exclusivamente una
RepresentacionCanonica V1. Cualquier otra codificacion, aunque
json.Unmarshal pudiera interpretarla con el mismo resultado aparente,
se rechaza.

```go
func RestaurarConjuntoReglasBaremoConHuellaSHA256(
	contenido []byte,
	huellaEsperada string,
) (ConjuntoReglasBaremo, error)
```

RestaurarConjuntoReglasBaremoConHuellaSHA256 añade a la restauracion
canonica la comprobacion en tiempo constante de una huella esperada.

```go
func (c ConjuntoReglasBaremo) Bases() ReferenciaVersionada
```

Bases devuelve la version y huella exactas de las bases publicadas.

```go
func (c ConjuntoReglasBaremo) FechaCorte() baremacion.FechaCivil
```

FechaCorte devuelve el ultimo dia civil incluido en el computo. La
inclusividad forma parte de la invariante V1 y no depende de una convencion
del calculador.

```go
func (c ConjuntoReglasBaremo) GrupoConcurrenciaPorClave(
	clave string,
) (GrupoConcurrenciaExperiencia, bool)
```

GrupoConcurrenciaPorClave busca sin exponer la coleccion interna.

```go
func (c ConjuntoReglasBaremo) GruposConcurrenciaExperiencia() []GrupoConcurrenciaExperiencia
```

GruposConcurrenciaExperiencia devuelve una copia ordenada de las politicas
que resuelven coincidencias de reglas y solapes temporales.

```go
func (c ConjuntoReglasBaremo) HuellaSHA256() (string, error)
```

HuellaSHA256 calcula la huella hexadecimal minuscula de los bytes canonicos.

```go
func (c ConjuntoReglasBaremo) Identidad() IdentidadConjuntoReglasBaremo
```

Identidad devuelve la identidad por valor.

```go
func (c ConjuntoReglasBaremo) MarshalJSON() ([]byte, error)
```

MarshalJSON usa exactamente el contrato canonico, sin una segunda forma de
serializacion accidental.

```go
func (c ConjuntoReglasBaremo) ReferenciaVersionada() (ReferenciaVersionada, error)
```

ReferenciaVersionada devuelve la identidad del conjunto enlazada a su
contenido exacto.

```go
func (c ConjuntoReglasBaremo) ReglaExperienciaPorClave(clave string) (ReglaExperiencia, bool)
```

ReglaExperienciaPorClave busca y devuelve una copia profunda.

```go
func (c ConjuntoReglasBaremo) ReglasExperiencia() []ReglaExperiencia
```

ReglasExperiencia devuelve una copia profunda y ordenada.

```go
func (c ConjuntoReglasBaremo) RepresentacionCanonica() ([]byte, error)
```

RepresentacionCanonica devuelve un contrato JSON estable de esquema V1.
No serializa directamente los campos privados del agregado.

```go
func (c ConjuntoReglasBaremo) SeccionPorClave(clave string) (SeccionBaremo, bool)
```

SeccionPorClave busca sin exponer la coleccion interna.

```go
func (c ConjuntoReglasBaremo) Secciones() []SeccionBaremo
```

Secciones devuelve una copia ordenada.

```go
func (c ConjuntoReglasBaremo) Validar() error
```

Validar vuelve a comprobar las invariantes de la instantanea.

```go
type CriterioDesempateExceso string
```

CriterioDesempateExceso hace visible en el canon si interviene la prioridad.

```go
const (
	DesempateExcesoNoAplica              CriterioDesempateExceso = "no_aplica"
	DesempateExcesoPrioridadConcurrencia CriterioDesempateExceso = "prioridad_concurrencia"
)
type CriterioExperiencia struct {
	// Has unexported fields.
}
```

CriterioExperiencia enlaza un eje configurable con un catalogo versionado y
un conjunto cerrado de claves admitidas. No ejecuta expresiones libres.

```go
func NuevoCriterioExperiencia(
	clave string,
	catalogo ReferenciaVersionada,
	valores []string,
) (CriterioExperiencia, error)
```

NuevoCriterioExperiencia valida, deduplica mediante rechazo y ordena los
valores para que el mismo significado produzca los mismos bytes.

```go
func (c CriterioExperiencia) Catalogo() ReferenciaVersionada

func (c CriterioExperiencia) Clave() string

func (c CriterioExperiencia) Valores() []string

type DatosAtestacionAprobacionFirmadaReglasBaremo struct {
	Atestacion    ReferenciaVersionada
	Vinculo       VinculoEstadoReglasBaremo
	Firma         ReferenciaVersionada
	PoliticaFirma ReferenciaVersionada
	Firmantes     []string
	FirmadaEn     time.Time
	VerificadaEn  time.Time
	ValidaHasta   time.Time
}

type DatosAtestacionAutoridadReglasBaremo struct {
	Atestacion   ReferenciaVersionada
	Vinculo      VinculoEstadoReglasBaremo
	Accion       AccionGobiernoReglasBaremo
	PrincipalRef string
	Relacionada  *ReferenciaVersionada
	EmitidaEn    time.Time
	ValidaHasta  time.Time
}

type DatosAtestacionDependenciasVigentesReglasBaremo struct {
	Atestacion     ReferenciaVersionada
	Vinculo        VinculoEstadoReglasBaremo
	Convocatoria   ReferenciaVersionada
	Bases          ReferenciaVersionada
	Dependencias   []ReferenciaVersionada
	VerificadorRef string
	VerificadaEn   time.Time
	ValidaHasta    time.Time
}

type ErrorGobierno struct {
	// Has unexported fields.
}
```

ErrorGobierno es un error de dominio estable y deliberadamente exento de
valores potencialmente sensibles.

```go
func (e *ErrorGobierno) Codigo() CodigoErrorGobierno

func (e *ErrorGobierno) Error() string

func (e *ErrorGobierno) Is(objetivo error) bool

type ErrorModelo struct {
	// Has unexported fields.
}
```

ErrorModelo es un error de dominio clasificable con errors.Is.

```go
func (e *ErrorModelo) Campo() string
```

Campo devuelve el nombre tecnico del elemento rechazado, nunca su valor.

```go
func (e *ErrorModelo) Codigo() CodigoError
```

Codigo devuelve la causa estable del rechazo.

```go
func (e *ErrorModelo) Error() string

func (e *ErrorModelo) Is(objetivo error) bool
```

Is permite clasificar errores sin depender del texto mostrado.

```go
type EstadoGobiernoReglasBaremo string
```

EstadoGobiernoReglasBaremo describe exclusivamente el gobierno de una
version de contenido inmutable. No se modifica el conjunto al transicionar.

```go
const (
	EstadoReglasBaremoBorrador   EstadoGobiernoReglasBaremo = "borrador"
	EstadoReglasBaremoPublicada  EstadoGobiernoReglasBaremo = "publicada"
	EstadoReglasBaremoActiva     EstadoGobiernoReglasBaremo = "activa"
	EstadoReglasBaremoSustituida EstadoGobiernoReglasBaremo = "sustituida"
	EstadoReglasBaremoRetirada   EstadoGobiernoReglasBaremo = "retirada"
	EstadoReglasBaremoDescartada EstadoGobiernoReglasBaremo = "descartada"
)
func (e EstadoGobiernoReglasBaremo) Valido() bool

type GrupoConcurrenciaExperiencia struct {
	// Has unexported fields.
}
```

GrupoConcurrenciaExperiencia gobierna coincidencias multiples entre reglas,
incluso si pertenecen a secciones diferentes. La prioridad 1 de cada regla
es la maxima y no se admite empate dentro del grupo. Si un mismo tramo
coincide en grupos diferentes, V1 no inventa una prioridad entre grupos:
el calculador debe rechazar la entrada de forma cerrada.

```go
func NuevoGrupoConcurrenciaExperiencia(
	clave string,
	definicion ReferenciaVersionada,
	orden uint32,
	coincidenciaReglas PoliticaCoincidenciaReglas,
	solape PoliticaSolape,
	repartoExceso *PoliticaRepartoExceso,
) (GrupoConcurrenciaExperiencia, error)
```

NuevoGrupoConcurrenciaExperiencia exige la politica de exceso exactamente
cuando el solape acumula hasta un limite. El puntero se copia y no se
retiene.

```go
func (g GrupoConcurrenciaExperiencia) Clave() string

func (g GrupoConcurrenciaExperiencia) CoincidenciaReglas() PoliticaCoincidenciaReglas

func (g GrupoConcurrenciaExperiencia) Definicion() ReferenciaVersionada

func (g GrupoConcurrenciaExperiencia) Orden() uint32

func (g GrupoConcurrenciaExperiencia) RepartoExceso() (PoliticaRepartoExceso, bool)

func (g GrupoConcurrenciaExperiencia) Solape() PoliticaSolape

type IdentidadConjuntoReglasBaremo struct {
	// Has unexported fields.
}
```

IdentidadConjuntoReglasBaremo enlaza el conjunto con una convocatoria y un
expediente concretos. Todos los identificadores son referencias opacas.

```go
func NuevaIdentidadConjuntoReglasBaremo(
	referencia string,
	version uint64,
	convocatoriaRef string,
	expedienteRef string,
) (IdentidadConjuntoReglasBaremo, error)
```

NuevaIdentidadConjuntoReglasBaremo valida la identidad sin normalizar
silenciosamente ninguna referencia.

```go
func (i IdentidadConjuntoReglasBaremo) ConvocatoriaRef() string
```

ConvocatoriaRef devuelve la convocatoria a la que pertenece.

```go
func (i IdentidadConjuntoReglasBaremo) ExpedienteRef() string
```

ExpedienteRef devuelve el expediente administrativo enlazado.

```go
func (i IdentidadConjuntoReglasBaremo) Referencia() string
```

Referencia devuelve la referencia inmutable del conjunto.

```go
func (i IdentidadConjuntoReglasBaremo) Version() uint64
```

Version devuelve la version semantica del conjunto.

```go
type LimitePuntos struct {
	// Has unexported fields.
}
```

LimitePuntos distingue de forma expresa la ausencia de limite de un valor
cero. Su valor cero es invalido y no selecciona una politica implicita.

```go
func NuevoLimitePuntos(valor baremacion.Puntos) (LimitePuntos, error)
```

NuevoLimitePuntos declara un tope positivo exacto.

```go
func SinLimitePuntos() LimitePuntos
```

SinLimitePuntos declara explicitamente que la regla no tiene tope propio.

```go
func (l LimitePuntos) EstaLimitado() bool
```

EstaLimitado indica si existe un maximo propio.

```go
func (l LimitePuntos) Valor() (baremacion.Puntos, bool)
```

Valor devuelve el limite y si este fue configurado.

```go
type LimiteUnidades struct {
	// Has unexported fields.
}
```

LimiteUnidades distingue un limite racional positivo de la ausencia expresa
de tope temporal.

```go
func NuevoLimiteUnidades(valor baremacion.Racional) (LimiteUnidades, error)
```

NuevoLimiteUnidades construye un tope racional positivo y exacto.

```go
func SinLimiteUnidades() LimiteUnidades
```

SinLimiteUnidades declara explicitamente que no hay tope temporal propio.

```go
func (l LimiteUnidades) EstaLimitado() bool
```

EstaLimitado indica si hay un maximo de unidades.

```go
func (l LimiteUnidades) Valor() (baremacion.Racional, bool)
```

Valor devuelve el limite y si este fue configurado.

```go
type ModoCoincidenciaReglas string
```

ModoCoincidenciaReglas decide que ocurre cuando un mismo tramo satisface
simultaneamente los criterios de varias reglas del grupo.

```go
const (
	CoincidenciaReglasRechazar              ModoCoincidenciaReglas = "rechazar"
	CoincidenciaReglasElegirPrioridad       ModoCoincidenciaReglas = "elegir_prioridad"
	CoincidenciaReglasElegirMayorPuntuacion ModoCoincidenciaReglas = "elegir_mayor_puntuacion"
	CoincidenciaReglasAcumular              ModoCoincidenciaReglas = "acumular"
)
type ModoJornada string
```

ModoJornada selecciona una semantica revisada por el dominio.

```go
const (
	JornadaProporcional       ModoJornada = "proporcional"
	JornadaIntegra            ModoJornada = "integra"
	JornadaIntegraDesdeUmbral ModoJornada = "integra_desde_umbral"
	JornadaProtegidaIntegra   ModoJornada = "protegida_integra"
	JornadaPorHoras           ModoJornada = "por_horas"
)
type ModoRepartoDentroRegla string
```

ModoRepartoDentroRegla impide usar el orden de entrada como desempate.

```go
const (
	RepartoDentroReglaNoAplica           ModoRepartoDentroRegla = "no_aplica"
	RepartoDentroReglaProporcionalExacto ModoRepartoDentroRegla = "proporcional_exacto"
)
type ModoRepartoExceso string
```

ModoRepartoExceso decide que hacer con la dedicacion que supera el limite de
una politica de solape acumulable.

```go
const (
	RepartoExcesoRechazar                      ModoRepartoExceso = "rechazar"
	RepartoExcesoRecortarPorPrioridad          ModoRepartoExceso = "recortar_por_prioridad"
	RepartoExcesoProporcionalExacto            ModoRepartoExceso = "repartir_proporcional_exacto"
	RepartoExcesoElegirMayorPuntuacionMarginal ModoRepartoExceso = "elegir_mayor_puntuacion_marginal"
)
type ModoRestos string
```

ModoRestos fija en que frontera se conserva o descarta una fraccion
temporal.

```go
const (
	RestosConservarExactos    ModoRestos = "conservar_exactos"
	RestosAcumularPorRegla    ModoRestos = "acumular_por_regla"
	RestosDescartarPorPeriodo ModoRestos = "descartar_por_periodo"
	RestosDescartarPorRegla   ModoRestos = "descartar_por_regla"
)
type ModoSolape string
```

ModoSolape selecciona como resolver tramos distintos que concurren en el
tiempo dentro de un grupo. La coincidencia de un mismo tramo con varias
reglas se gobierna por PoliticaCoincidenciaReglas, nunca por este valor.

```go
const (
	SolapeRechazar              ModoSolape = "rechazar"
	SolapeAcumularHastaLimite   ModoSolape = "acumular_hasta_limite"
	SolapeElegirMayorPuntuacion ModoSolape = "elegir_mayor_puntuacion"
	SolapeElegirMayorDedicacion ModoSolape = "elegir_mayor_dedicacion"
)
type MomentoRedondeo string
```

MomentoRedondeo fija la unica frontera en la que se redondea.

```go
const (
	RedondearPorPeriodo MomentoRedondeo = "periodo"
	RedondearPorRegla   MomentoRedondeo = "regla"
	RedondearPorSeccion MomentoRedondeo = "seccion"
	RedondearEnTotal    MomentoRedondeo = "total"
)
type MotivoCatalogadoReglasBaremo struct {
	// Has unexported fields.
}
```

MotivoCatalogadoReglasBaremo fija tanto la version y huella del catalogo
como la clave elegida. El dominio no conserva un motivo libre en las trazas.

```go
func NuevoMotivoCatalogadoReglasBaremo(
	catalogo ReferenciaVersionada,
	clave string,
) (MotivoCatalogadoReglasBaremo, error)

func (m MotivoCatalogadoReglasBaremo) Catalogo() ReferenciaVersionada

func (m MotivoCatalogadoReglasBaremo) Clave() string

type PoliticaCoincidenciaReglas struct {
	// Has unexported fields.
}
```

PoliticaCoincidenciaReglas exige una eleccion explicita y no confunde una
coincidencia de criterios con el solape temporal de tramos distintos.
En las elecciones, una igualdad de puntuacion se resuelve por la prioridad
unica de regla: 1 es la maxima.

```go
func NuevaPoliticaCoincidenciaReglas(modo ModoCoincidenciaReglas) (PoliticaCoincidenciaReglas, error)
```

NuevaPoliticaCoincidenciaReglas valida un modo cerrado.

```go
func (p PoliticaCoincidenciaReglas) Modo() ModoCoincidenciaReglas

type PoliticaJornada struct {
	// Has unexported fields.
}
```

PoliticaJornada conserva el modo y, solo cuando corresponde, el umbral
exacto. El valor cero no representa ninguna politica.

```go
func NuevaPoliticaJornada(modo ModoJornada) (PoliticaJornada, error)
```

NuevaPoliticaJornada construye una politica sin umbral.

```go
func NuevaPoliticaJornadaDesdeUmbral(umbral baremacion.FraccionJornada) (PoliticaJornada, error)
```

NuevaPoliticaJornadaDesdeUmbral construye exclusivamente la politica de
computo integro a partir de una fraccion publicada.

```go
func (p PoliticaJornada) Modo() ModoJornada

func (p PoliticaJornada) Umbral() (baremacion.FraccionJornada, bool)

type PoliticaRedondeo struct {
	// Has unexported fields.
}
```

PoliticaRedondeo combina un momento explicito con uno de los modos exactos
del fundamento comun de baremacion.

```go
func NuevaPoliticaRedondeo(momento MomentoRedondeo, modo baremacion.ModoRedondeo) (PoliticaRedondeo, error)
```

NuevaPoliticaRedondeo no aplica ningun modo por defecto.

```go
func (p PoliticaRedondeo) Modo() baremacion.ModoRedondeo

func (p PoliticaRedondeo) Momento() MomentoRedondeo

type PoliticaRepartoExceso struct {
	// Has unexported fields.
}
```

PoliticaRepartoExceso existe exclusivamente con un solape acumulable.
Recortar por prioridad asigna capacidad por prioridad de regla; si varios
tramos de la misma regla comparten prioridad, distribuye el remanente de
forma proporcional exacta, nunca por orden de entrada. Elegir la mayor
puntuacion marginal desempata por prioridad y aplica el mismo reparto exacto
dentro de la regla elegida.

```go
func NuevaPoliticaRepartoExceso(modo ModoRepartoExceso) (PoliticaRepartoExceso, error)
```

NuevaPoliticaRepartoExceso valida un modo cerrado.

```go
func (p PoliticaRepartoExceso) DesempateEntreReglas() CriterioDesempateExceso

func (p PoliticaRepartoExceso) Modo() ModoRepartoExceso

func (p PoliticaRepartoExceso) RepartoDentroMismaRegla() ModoRepartoDentroRegla

type PoliticaRestos struct {
	// Has unexported fields.
}
```

PoliticaRestos exige una eleccion expresa.

```go
func NuevaPoliticaRestos(modo ModoRestos) (PoliticaRestos, error)
```

NuevaPoliticaRestos valida un modo conocido.

```go
func (p PoliticaRestos) Modo() ModoRestos

type PoliticaSolape struct {
	// Has unexported fields.
}
```

PoliticaSolape almacena el limite de dedicacion exclusivamente cuando se
acumulan periodos. El valor cero es invalido.

```go
func NuevaPoliticaSolape(modo ModoSolape) (PoliticaSolape, error)
```

NuevaPoliticaSolape construye una politica que no acumula fracciones.

```go
func NuevaPoliticaSolapeAcumulable(limite baremacion.FraccionJornada) (PoliticaSolape, error)
```

NuevaPoliticaSolapeAcumulable fija el limite exacto de acumulacion.

```go
func (p PoliticaSolape) Limite() (baremacion.FraccionJornada, bool)

func (p PoliticaSolape) Modo() ModoSolape

type PoliticaUnidadTemporal struct {
	// Has unexported fields.
}
```

PoliticaUnidadTemporal expresa una conversion exacta. Por ejemplo, una regla
por meses convencionales puede fijar dia -> mes y 30/1 unidades base.

```go
func NuevaPoliticaUnidadTemporal(
	unidadBase UnidadTemporal,
	unidadPuntuable UnidadTemporal,
	unidadesBasePorUnidad baremacion.Racional,
	extremoFinal TratamientoExtremoFinal,
) (PoliticaUnidadTemporal, error)
```

NuevaPoliticaUnidadTemporal exige todos sus parametros; no presupone 30 dias
por mes, 365 por anio ni la inclusion del extremo final.

```go
func (p PoliticaUnidadTemporal) ExtremoFinal() TratamientoExtremoFinal

func (p PoliticaUnidadTemporal) UnidadBase() UnidadTemporal

func (p PoliticaUnidadTemporal) UnidadPuntuable() UnidadTemporal

func (p PoliticaUnidadTemporal) UnidadesBasePorUnidad() baremacion.Racional

type ReferenciaVersionada struct {
	// Has unexported fields.
}
```

ReferenciaVersionada fija una dependencia por referencia opaca, version y
huella. Nunca significa "la version vigente".

```go
func NuevaReferenciaVersionada(referencia string, version uint64, huellaSHA256 string) (ReferenciaVersionada, error)
```

NuevaReferenciaVersionada construye una referencia cerrada y reproducible.

```go
func (r ReferenciaVersionada) HuellaSHA256() string
```

HuellaSHA256 devuelve la huella hexadecimal canonica.

```go
func (r ReferenciaVersionada) Referencia() string
```

Referencia devuelve el identificador opaco exacto.

```go
func (r ReferenciaVersionada) Version() uint64
```

Version devuelve la version positiva fijada.

```go
type ReglaExperiencia struct {
	// Has unexported fields.
}
```

ReglaExperiencia configura como transformar experiencia elegible en puntos.
Es solo modelo: no contiene ni ejecuta un calculador.

```go
func NuevaReglaExperiencia(
	clave string,
	definicion ReferenciaVersionada,
	seccionClave string,
	orden uint32,
	criterios []CriterioExperiencia,
	grupoConcurrenciaClave string,
	prioridadConcurrencia uint32,
	unidadTemporal PoliticaUnidadTemporal,
	jornada PoliticaJornada,
	restos PoliticaRestos,
	redondeo PoliticaRedondeo,
	puntosPorUnidad baremacion.Puntos,
	maximoUnidades LimiteUnidades,
	maximoPuntos LimitePuntos,
) (ReglaExperiencia, error)
```

NuevaReglaExperiencia exige coeficiente y politicas completos. No hay una
jornada, conversion, solape, resto o redondeo implicitos.

```go
func (r ReglaExperiencia) Clave() string

func (r ReglaExperiencia) Criterios() []CriterioExperiencia

func (r ReglaExperiencia) Definicion() ReferenciaVersionada

func (r ReglaExperiencia) GrupoConcurrenciaClave() string

func (r ReglaExperiencia) Jornada() PoliticaJornada

func (r ReglaExperiencia) MaximoPuntos() LimitePuntos

func (r ReglaExperiencia) MaximoUnidades() LimiteUnidades

func (r ReglaExperiencia) Orden() uint32

func (r ReglaExperiencia) PrioridadConcurrencia() uint32

func (r ReglaExperiencia) PuntosPorUnidad() baremacion.Puntos

func (r ReglaExperiencia) Redondeo() PoliticaRedondeo

func (r ReglaExperiencia) Restos() PoliticaRestos

func (r ReglaExperiencia) SeccionClave() string

func (r ReglaExperiencia) UnidadTemporal() PoliticaUnidadTemporal

type SeccionBaremo struct {
	// Has unexported fields.
}
```

SeccionBaremo es una seccion ordenada y acotada del baremo.

```go
func NuevaSeccionBaremo(
	clave string,
	definicion ReferenciaVersionada,
	orden uint32,
	puntosMinimos baremacion.Puntos,
	puntosMaximos baremacion.Puntos,
) (SeccionBaremo, error)
```

NuevaSeccionBaremo construye una seccion sin inferir sus limites.

```go
func (s SeccionBaremo) Clave() string

func (s SeccionBaremo) Definicion() ReferenciaVersionada

func (s SeccionBaremo) Orden() uint32

func (s SeccionBaremo) PuntosMaximos() baremacion.Puntos

func (s SeccionBaremo) PuntosMinimos() baremacion.Puntos

type TratamientoExtremoFinal string
```

TratamientoExtremoFinal fija si el ultimo dia u hora se incorpora antes de
convertir unidades.

```go
const (
	ExtremoFinalExclusivo TratamientoExtremoFinal = "exclusivo"
	ExtremoFinalInclusivo TratamientoExtremoFinal = "inclusivo"
)
type UnidadTemporal string
```

UnidadTemporal identifica la unidad de entrada o la unidad puntuable.

```go
const (
	UnidadTemporalDia  UnidadTemporal = "dia"
	UnidadTemporalMes  UnidadTemporal = "mes"
	UnidadTemporalAnio UnidadTemporal = "anio"
	UnidadTemporalHora UnidadTemporal = "hora"
)
type VersionGobernadaReglasBaremo struct {
	// Has unexported fields.
}
```

VersionGobernadaReglasBaremo envuelve un contenido inmutable. Todos sus
campos son privados; cada transicion devuelve una copia nueva y aumenta una
sola vez la revision usada para OCC.

```go
func NuevaVersionGobernadaReglasBaremo(
	conjunto ConjuntoReglasBaremo,
	actorRef string,
	motivo MotivoCatalogadoReglasBaremo,
	instante time.Time,
) (VersionGobernadaReglasBaremo, error)

func RestaurarVersionGobernadaReglasBaremo(
	contenido []byte,
) (VersionGobernadaReglasBaremo, error)
```

RestaurarVersionGobernadaReglasBaremo reconstruye exclusivamente la
representacion canonica V1. Ademas de validar el JSON, reproduce el ciclo de
gobierno mediante sus constructores y transiciones publicas.

```go
func RestaurarVersionGobernadaReglasBaremoConHuellaSHA256(
	contenido []byte,
	huellaEsperada string,
) (VersionGobernadaReglasBaremo, error)
```

RestaurarVersionGobernadaReglasBaremoConHuellaSHA256 exige tambien la huella
canonica esperada y la compara en tiempo constante.

```go
func (v VersionGobernadaReglasBaremo) Activar(
	revisionEsperada uint64,
	actorRef string,
	motivo MotivoCatalogadoReglasBaremo,
	dependencias AtestacionDependenciasVigentesReglasBaremo,
	instante time.Time,
) (VersionGobernadaReglasBaremo, error)

func (v VersionGobernadaReglasBaremo) Clonar() (VersionGobernadaReglasBaremo, error)

func (v VersionGobernadaReglasBaremo) Conjunto() (ConjuntoReglasBaremo, error)

func (v VersionGobernadaReglasBaremo) ConvocatoriaActivacion() (
	ReferenciaVersionada,
	bool,
	error,
)
```

ConvocatoriaActivacion devuelve la referencia exacta que el verificador de
dependencias ligo al activar la version. El segundo resultado distingue
una ausencia legitima de activacion en borradores, versiones publicadas y
descartadas. Las versiones sustituidas o retiradas conservan el vinculo para
permitir la reproduccion historica.

No expone el acto ni la atestacion internos. La referencia se devuelve por
valor y no comparte estado mutable con la version gobernada.

```go
func (v VersionGobernadaReglasBaremo) CreadaEn() time.Time

func (v VersionGobernadaReglasBaremo) CreadaPor() string

func (v VersionGobernadaReglasBaremo) DependenciasContenido() ([]ReferenciaVersionada, error)
```

DependenciasContenido devuelve bases, definiciones y catalogos exactos en
orden canonico para que el verificador externo pueda atestarlos.

```go
func (v VersionGobernadaReglasBaremo) Descartar(
	revisionEsperada uint64,
	actorRef string,
	motivo MotivoCatalogadoReglasBaremo,
	autoridad AtestacionAutoridadReglasBaremo,
	instante time.Time,
) (VersionGobernadaReglasBaremo, error)

func (v VersionGobernadaReglasBaremo) Estado() EstadoGobiernoReglasBaremo

func (v VersionGobernadaReglasBaremo) HuellaSHA256() (string, error)

func (v VersionGobernadaReglasBaremo) MotivoCreacion() MotivoCatalogadoReglasBaremo

func (v VersionGobernadaReglasBaremo) Publicar(
	revisionEsperada uint64,
	actorRef string,
	motivo MotivoCatalogadoReglasBaremo,
	aprobacion AtestacionAprobacionFirmadaReglasBaremo,
	instante time.Time,
) (VersionGobernadaReglasBaremo, error)

func (v VersionGobernadaReglasBaremo) ReferenciaContenido() (ReferenciaVersionada, error)

func (v VersionGobernadaReglasBaremo) RepresentacionCanonica() ([]byte, error)

func (v VersionGobernadaReglasBaremo) Retirar(
	revisionEsperada uint64,
	actorRef string,
	motivo MotivoCatalogadoReglasBaremo,
	autoridad AtestacionAutoridadReglasBaremo,
	instante time.Time,
) (VersionGobernadaReglasBaremo, error)

func (v VersionGobernadaReglasBaremo) Revision() uint64

func (v VersionGobernadaReglasBaremo) Sustituir(
	revisionEsperada uint64,
	actorRef string,
	motivo MotivoCatalogadoReglasBaremo,
	sucesora ReferenciaVersionada,
	autoridad AtestacionAutoridadReglasBaremo,
	instante time.Time,
) (VersionGobernadaReglasBaremo, error)

func (v VersionGobernadaReglasBaremo) Validar() error

func (v VersionGobernadaReglasBaremo) VinculoEstado() (VinculoEstadoReglasBaremo, error)

type VinculoEstadoReglasBaremo struct {
	// Has unexported fields.
}
```

VinculoEstadoReglasBaremo impide aplicar una atestacion a otra revision,
otro contenido o un estado materialmente distinto.

```go
func NuevoVinculoEstadoReglasBaremo(
	contenido ReferenciaVersionada,
	revision uint64,
	huellaEstadoSHA256 string,
) (VinculoEstadoReglasBaremo, error)

func (v VinculoEstadoReglasBaremo) Contenido() ReferenciaVersionada

func (v VinculoEstadoReglasBaremo) HuellaEstadoSHA256() string

func (v VinculoEstadoReglasBaremo) Revision() uint64
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

Package ports declara las fronteras hexagonales del modulo de Bolsa.

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
	AccionConsultarVersionConvocatoria         = "bolsa.convocatoria.version.consultar"
	AccionConsultarVersionConFlujoConvocatoria = "bolsa.convocatoria.version_con_flujo.consultar"
	AccionCrearBorradorConvocatoria            = "bolsa.convocatoria.borrador.crear"
	AccionActualizarBorradorConvocatoria       = "bolsa.convocatoria.borrador.actualizar"
	AccionPublicarVersionConvocatoria          = "bolsa.convocatoria.version.publicar"
	AccionPublicarYSustituirConvocatoria       = "bolsa.convocatoria.version.publicar_y_sustituir"
	AccionPublicarTrasRetiradaConvocatoria     = "bolsa.convocatoria.version.publicar_tras_retirada"
	AccionRetirarVersionConvocatoria           = "bolsa.convocatoria.version.retirar"
	ModuloGobiernoConvocatorias                = "bolsa"
	TipoRecursoVersionConvocatoriaGobernada    = "version_convocatoria_gobernada"
	FinalidadConsultaInternaConvocatorias      = "consulta_interna_convocatorias"
	FinalidadGobiernoConvocatorias             = "gobierno_convocatorias"
	AtributoHuellaIntencionConvocatoria        = "huella_intencion_sha256"
	VentanaMaximaUsoAutorizacionConvocatoria   = 30 * time.Second
)
const (
	VersionTestimonioIdempotenciaConvocatoriaV1 = 1
	VigenciaMaximaTestimonioConvocatoria        = 10 * time.Minute
)
const (
	VigenciaMaximaComprobacionDependenciasConvocatoria = 15 * time.Minute
	VigenciaMaximaAtestacionVerificacionConvocatoria   = 5 * time.Minute
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
const (
	EsquemaPanelInternoBolsaV1 = "vec.bolsa.panel.interno.v1"

	ModuloPanelInternoBolsa      = "bolsa"
	TipoRecursoPanelInternoBolsa = "panel_interno_agregado"
	AccionConsultarPanelInterno  = "bolsa.panel_interno.consultar"
	FinalidadPanelInternoBolsa   = "gestion_operativa_bolsa"
	CampoPanelInternoAgregado    = "panel_agregado_sin_datos_personales"
)
const DominioCriptograficoMotivoGobiernoConvocatoriaV1 = "bolsa.convocatoria.motivo.v1"
const VentanaMaximaUsoAutorizacionBaremacion = 30 * time.Second
```

VentanaMaximaUsoAutorizacionBaremacion limita el tiempo durante el que una
decision ya evaluada puede viajar hasta el punto de aplicacion.

```go
const VigenciaMaximaAtestacionMotivoGobiernoConvocatoria = 5 * time.Minute
```

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
	ErrAutorizacionGobiernoConvocatoriaInvalida = errors.New("bolsa: autorizacion de gobierno de convocatoria invalida")
	ErrConsultaGobiernoConvocatoriaInvalida     = errors.New("bolsa: consulta interna de convocatoria invalida")
	ErrVersionGobernadaConvocatoriaNoEncontrada = errors.New("bolsa: version gobernada de convocatoria no encontrada")
	ErrFuenteGobiernoConvocatoriasNoDisponible  = errors.New("bolsa: fuente de gobierno de convocatorias no disponible")
	ErrConsultaGobiernoConvocatoriaEnCurso      = errors.New("bolsa: consulta de convocatoria en curso")
	ErrEvidenciaConsultaConvocatoriaNoConfiable = errors.New("bolsa: evidencia de consulta de convocatoria no confiable")
)
var (
	ErrMaterialIntencionConvocatoriaInvalido = errors.New("bolsa: material de intencion de convocatoria invalido")
	ErrIdempotenciaConvocatoriaInvalida      = errors.New("bolsa: idempotencia semantica de convocatoria invalida")
	ErrClaveIdempotenciaConvocatoriaReusada  = errors.New("bolsa: clave de idempotencia reutilizada con otra intencion")
	ErrSerializacionIdempotenciaConvocatoria = errors.New("bolsa: serializacion de idempotencia de convocatoria prohibida")
)
var (
	ErrSelladoMotivoGobiernoConvocatoriaInvalido = errors.New("bolsa: sellado HMAC de motivo de convocatoria invalido")
	ErrSerializacionMotivoGobiernoConvocatoria   = errors.New("bolsa: serializacion de motivo de convocatoria prohibida")
)
var (
	ErrConfirmacionGobiernoConvocatoriaInvalida   = errors.New("bolsa: confirmacion de gobierno de convocatoria invalida")
	ErrVersionGobernadaConvocatoriaYaExiste       = errors.New("bolsa: version gobernada de convocatoria ya existe")
	ErrCASVersionConvocatoriaEnConflicto          = errors.New("bolsa: revision o huella de convocatoria en conflicto")
	ErrRamaVersionConvocatoriaEnConflicto         = errors.New("bolsa: la predecesora ya tiene otra rama")
	ErrUsoAutorizacionConvocatoriaConsumido       = errors.New("bolsa: uso de autorizacion de convocatoria ya consumido")
	ErrAtestacionVerificacionConsumida            = errors.New("bolsa: atestacion de verificacion de convocatoria ya consumida")
	ErrReciboGobiernoConvocatoriaInvalido         = errors.New("bolsa: recibo de gobierno de convocatoria invalido")
	ErrSerializacionGobiernoConvocatoriaProhibida = errors.New("bolsa: serializacion de orden de gobierno de convocatoria prohibida")
)
var (
	ErrComprobacionDependenciasConvocatoriaInvalida = errors.New("bolsa: comprobacion de dependencias de convocatoria invalida")
	ErrAprobacionConvocatoriaInvalida               = errors.New("bolsa: aprobacion de convocatoria invalida")
	ErrDependenciaConvocatoriaNoDisponible          = errors.New("bolsa: dependencia exacta de convocatoria no disponible")
	ErrAprobacionConvocatoriaNoDisponible           = errors.New("bolsa: aprobacion de convocatoria no disponible")
	ErrSerializacionVerificacionConvocatoria        = errors.New("bolsa: serializacion de verificacion de convocatoria prohibida")
)
var (
	ErrConsultaConvocatoriasInvalida  = errors.New("bolsa: consulta publica de convocatorias invalida")
	ErrConsultaCategoriasInvalida     = errors.New("bolsa: consulta publica de categorias invalida")
	ErrConvocatoriaNoEncontrada       = errors.New("bolsa: convocatoria publica no encontrada")
	ErrFuenteConvocatoriasInvalida    = errors.New("bolsa: fuente publica de convocatorias invalida")
	ErrCatalogoCategoriasNoDisponible = errors.New("bolsa: catalogo publico de categorias no disponible")
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
	ErrComandoGuardarPropuestaLlamamientoInvalido = errors.New("bolsa: comando para guardar propuesta de llamamiento invalido")
)
var (
	ErrSelectorPanelInternoInvalido  = errors.New("bolsa: selector de panel interno invalido")
	ErrConsultaPanelInternoInvalida  = errors.New("bolsa: consulta de panel interno invalida")
	ErrResultadoPanelInternoInvalido = errors.New("bolsa: resultado de panel interno invalido")
)
var (
	// ErrReglasBaremoNoEncontradas indica que no existe el estado exacto
	// solicitado. Ningun adaptador puede sustituirlo por «el vigente».
	ErrReglasBaremoNoEncontradas = errors.New("bolsa: estado exacto de reglas de baremo no encontrado")

	// ErrConflictoOCCReglasBaremo indica que revision o huella esperadas ya no
	// coinciden con el estado durable.
	ErrConflictoOCCReglasBaremo = errors.New("bolsa: conflicto OCC en reglas de baremo")

	// ErrClaveIdempotenciaReglasReutilizada corresponde al indice exacto de
	// intencion ya usado para otro material semantico. La derivacion de ese
	// indice pertenece a application/internal, nunca a este paquete.
	ErrClaveIdempotenciaReglasReutilizada = errors.New("bolsa: indice idempotente reutilizado con otra intencion")

	ErrConfirmacionReglasBaremoInvalida  = errors.New("bolsa: confirmacion transaccional de reglas de baremo invalida")
	ErrFuenteCalculoReglasBaremoInvalida = errors.New("bolsa: fuente exacta de calculo de reglas de baremo invalida")
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
func RecursoAutorizableConsultaVersionConvocatoria(
	selector SelectorVersionConvocatoriaExacta,
) (dominiovec.RecursoAutorizable, error)
```

RecursoAutorizableConsultaVersionConvocatoria construye el mismo recurso
exacto que debe evaluar el PDP. La consulta no admite ambitos ni atributos
declarados por el cliente.

```go
func RecursoAutorizableMutacionConvocatoria(
	material MaterialIntencionGobiernoConvocatoria,
) (dominiovec.RecursoAutorizable, error)
```

RecursoAutorizableMutacionConvocatoria liga la concesion a la preimagen
semantica completa; una concesion no puede aplicarse a otra mutacion.

```go
func RecursoAutorizablePanelInterno(
	selector SelectorPanelInterno,
	motivo dominiovec.ReferenciaEntradaCatalogo,
) (dominiovec.RecursoAutorizable, error)
```

RecursoAutorizablePanelInterno liga el alcance exacto y el motivo catalogado
a la decision V2. No acepta comodines ni ambitos solapados.

```go
func ReferenciaOpacaLlamamientoValida(valor string) bool
```

ReferenciaOpacaLlamamientoValida evita documentos personales evidentes,
comodines, controles, espacios no canonicos y texto Unicode ambiguo.
No pretende sustituir el emisor criptograficamente aleatorio de referencias.

```go
func VinculoAptoParaGestionLlamamientos(
	vinculo dominiovec.VinculoAutenticacionActorV1,
	actor dominiovec.ContextoActor,
	perfilActivoRef string,
) bool
```

VinculoAptoParaGestionLlamamientos comprueba la frontera de autenticacion
reforzada de una operacion interna de RRHH. No concede autorizacion:
exige que el vinculo opaco proceda de la superficie corporativa con
cuenta ordinaria o de administracion con cuenta privilegiada, y que
conserve exactamente el contexto y el perfil resueltos con garantia alta.
La superficie personal externa y el metodo de demostracion nunca habilitan
el acceso aunque un PDP defectuoso tratase de concederlo.

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
type AccionAprobacionConvocatoria string

const (
	AccionAprobacionPublicarConvocatoria AccionAprobacionConvocatoria = "publicar"
	AccionAprobacionRetirarConvocatoria  AccionAprobacionConvocatoria = "retirar"
)
func (a AccionAprobacionConvocatoria) Valida() bool

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
type ActuacionPendientePanelInterno struct {
	ActuacionRef    string    `json:"actuacion_ref"`
	RecursoRef      string    `json:"recurso_ref"`
	TipoClave       string    `json:"tipo_clave"`
	EstadoClave     string    `json:"estado_clave"`
	PrioridadClave  string    `json:"prioridad_clave"`
	FechaLimite     time.Time `json:"fecha_limite,omitempty"`
	NumeroElementos int       `json:"numero_elementos"`
}
```

ActuacionPendientePanelInterno describe trabajo administrativo sin actor ni
interesado. RecursoRef apunta al expediente o agregado autorizado.

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
type AtestacionAprobacionConvocatoria struct {
	// Has unexported fields.
}
```

AtestacionAprobacionConvocatoria es reconstruible y no autoritativa. El
repositorio debe verificarla y consumir su token dentro de la transaccion.

```go
func NuevaAtestacionAprobacionConvocatoria(
	solicitud SolicitudComprobarAprobacionConvocatoria,
	datos DatosAtestacionAprobacionConvocatoria,
) (AtestacionAprobacionConvocatoria, error)

func (c AtestacionAprobacionConvocatoria) DatosParaConsumo() (
	DatosAtestacionAprobacionConvocatoria,
	error,
)

func (c AtestacionAprobacionConvocatoria) Evidencia() (
	dominiobolsa.EvidenciaAprobacionConvocatoria,
	error,
)

func (c AtestacionAprobacionConvocatoria) Format(estado fmt.State, _ rune)

func (c AtestacionAprobacionConvocatoria) GoString() string

func (*AtestacionAprobacionConvocatoria) GobDecode([]byte) error

func (AtestacionAprobacionConvocatoria) GobEncode() ([]byte, error)

func (c AtestacionAprobacionConvocatoria) LogValue() slog.Value

func (AtestacionAprobacionConvocatoria) MarshalBinary() ([]byte, error)

func (AtestacionAprobacionConvocatoria) MarshalJSON() ([]byte, error)

func (AtestacionAprobacionConvocatoria) MarshalText() ([]byte, error)

func (AtestacionAprobacionConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (AtestacionAprobacionConvocatoria) String() string

func (*AtestacionAprobacionConvocatoria) UnmarshalBinary([]byte) error

func (*AtestacionAprobacionConvocatoria) UnmarshalJSON([]byte) error

func (*AtestacionAprobacionConvocatoria) UnmarshalText([]byte) error

func (*AtestacionAprobacionConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (c AtestacionAprobacionConvocatoria) ValidarPara(
	solicitud SolicitudComprobarAprobacionConvocatoria,
	instante time.Time,
) error

type AtestacionDependenciasConvocatoria struct {
	// Has unexported fields.
}
```

AtestacionDependenciasConvocatoria es un testimonio reconstruible para que
la barrera durable relea y verifique su procedencia. No es una capacidad ni
concede autoridad por si sola.

```go
func NuevaAtestacionDependenciasConvocatoria(
	solicitud SolicitudVerificarDependenciasConvocatoria,
	datos DatosAtestacionDependenciasConvocatoria,
) (AtestacionDependenciasConvocatoria, error)

func (c AtestacionDependenciasConvocatoria) DatosParaConsumo() (
	DatosAtestacionDependenciasConvocatoria,
	error,
)

func (c AtestacionDependenciasConvocatoria) Evidencia() (
	dominiobolsa.EvidenciaDependenciasConvocatoria,
	error,
)

func (c AtestacionDependenciasConvocatoria) Format(estado fmt.State, _ rune)

func (c AtestacionDependenciasConvocatoria) GoString() string

func (*AtestacionDependenciasConvocatoria) GobDecode([]byte) error

func (AtestacionDependenciasConvocatoria) GobEncode() ([]byte, error)

func (c AtestacionDependenciasConvocatoria) LogValue() slog.Value

func (AtestacionDependenciasConvocatoria) MarshalBinary() ([]byte, error)

func (AtestacionDependenciasConvocatoria) MarshalJSON() ([]byte, error)

func (AtestacionDependenciasConvocatoria) MarshalText() ([]byte, error)

func (AtestacionDependenciasConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (AtestacionDependenciasConvocatoria) String() string

func (*AtestacionDependenciasConvocatoria) UnmarshalBinary([]byte) error

func (*AtestacionDependenciasConvocatoria) UnmarshalJSON([]byte) error

func (*AtestacionDependenciasConvocatoria) UnmarshalText([]byte) error

func (*AtestacionDependenciasConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (c AtestacionDependenciasConvocatoria) ValidarPara(
	solicitud SolicitudVerificarDependenciasConvocatoria,
	instante time.Time,
) error

type AtestacionSelladoMotivoConvocatoria struct {
	// Has unexported fields.
}
```

AtestacionSelladoMotivoConvocatoria es reconstruible y no concede autoridad.
La barrera durable debe releerla desde el registro del HSM/KMS, comprobar su
huella y consumir TokenConsumoRef en la misma transaccion que la mutacion.

```go
func NuevaAtestacionSelladoMotivoConvocatoria(
	solicitud SolicitudSellarMotivoGobiernoConvocatoria,
	datos DatosAtestacionSelladoMotivoConvocatoria,
) (AtestacionSelladoMotivoConvocatoria, error)

func (a AtestacionSelladoMotivoConvocatoria) DatosParaConsumo() (
	DatosAtestacionSelladoMotivoConvocatoria,
	error,
)

func (a AtestacionSelladoMotivoConvocatoria) Format(estado fmt.State, _ rune)

func (a AtestacionSelladoMotivoConvocatoria) GoString() string

func (*AtestacionSelladoMotivoConvocatoria) GobDecode([]byte) error

func (AtestacionSelladoMotivoConvocatoria) GobEncode() ([]byte, error)

func (a AtestacionSelladoMotivoConvocatoria) LogValue() slog.Value

func (AtestacionSelladoMotivoConvocatoria) MarshalBinary() ([]byte, error)

func (AtestacionSelladoMotivoConvocatoria) MarshalJSON() ([]byte, error)

func (AtestacionSelladoMotivoConvocatoria) MarshalText() ([]byte, error)

func (AtestacionSelladoMotivoConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (AtestacionSelladoMotivoConvocatoria) String() string

func (*AtestacionSelladoMotivoConvocatoria) UnmarshalBinary([]byte) error

func (*AtestacionSelladoMotivoConvocatoria) UnmarshalJSON([]byte) error

func (*AtestacionSelladoMotivoConvocatoria) UnmarshalText([]byte) error

func (*AtestacionSelladoMotivoConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

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

type CatalogoCategoriasPublicas struct {
	ID           string
	Version      int
	HuellaSHA256 string
	Fuente       MetadatosFuenteCategorias
	Categorias   []CategoriaPublica
}
```

CatalogoCategoriasPublicas conserva la identidad y version exactas que
resolvio el adaptador. Ningun consumidor selecciona implicitamente la ultima
version disponible.

```go
type CatalogoPoliticasFirmaBaremacion interface {
	ObtenerPoliticaFirma(context.Context, SolicitudObtenerPoliticaFirma) (PoliticaFirmaBaremacion, error)
}

type CatalogoPublico struct {
	Referencia string                   `json:"referencia"`
	Version    int                      `json:"version"`
	Entradas   []EntradaCatalogoPublico `json:"entradas"`
}

type CategoriaPublica struct {
	Clave        string
	Version      int
	Etiqueta     string
	Descripcion  string
	Semantica    string
	Orden        int
	Area         string
	AreaEtiqueta string
	Suscribible  bool
}
```

CategoriaPublica es la proyeccion minimizada de una entrada del catalogo
gobernado del nucleo. Los metadatos de procedencia, gobierno y aprobacion no
forman parte de este contrato publico.

```go
type ClaseAmbitoPanelInterno string
```

ClaseAmbitoPanelInterno obliga a elegir un alcance exacto. El valor cero no
significa toda la organizacion y nunca se interpreta como valor por defecto.

```go
const (
	AmbitoPanelOrganizacion ClaseAmbitoPanelInterno = "organizacion"
	AmbitoPanelUnidad       ClaseAmbitoPanelInterno = "unidad_gestion"
)
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

type ComandoGuardarPropuestaLlamamiento struct {
	// Has unexported fields.
}
```

ComandoGuardarPropuestaLlamamiento conserva, como una unica capacidad opaca,
todos los datos que una persistencia duradera debe confirmar de forma
indivisible. La propuesta solo contiene el prefijo evaluado; por eso el
comando retiene ademas la instantanea completa que genero dicho prefijo.

Sus campos son privados y Datos devuelve copias profundas. Un adaptador no
puede completar la instantanea consultando de nuevo una fuente mutable ni
aceptar propuesta y evidencia por parametros independientes.

```go
func NuevoComandoGuardarPropuestaLlamamiento(
	instantanea dominiobolsa.InstantaneaOrdenBolsa,
	propuesta dominiobolsa.PropuestaLlamamiento,
	evidencia puertosvec.EvidenciaUsoDecisionAutorizacion,
) (ComandoGuardarPropuestaLlamamiento, error)

func (c ComandoGuardarPropuestaLlamamiento) Datos() (
	dominiobolsa.InstantaneaOrdenBolsa,
	dominiobolsa.PropuestaLlamamiento,
	puertosvec.EvidenciaUsoDecisionAutorizacion,
	error,
)
```

Datos devuelve una fotografia defensiva del conjunto indivisible.
La evidencia ya es una capacidad opaca e inmutable del nucleo; instantanea y
propuesta se clonan para no compartir slices con el adaptador.

```go
func (c ComandoGuardarPropuestaLlamamiento) ValidarEn(instante time.Time) error
```

ValidarEn revalida la capacidad en el reloj efectivo del adaptador. No
sustituye el bloqueo y la relectura autoritativa dentro de la transaccion.

```go
type ComprobacionFirma struct {
	Clave                 string
	Estado                EstadoComprobacionFirma
	EvidenciaRef          string
	HuellaEvidenciaSHA256 string
}

func (c ComprobacionFirma) Validar() error

type ConfirmacionActualizacionBorradorConvocatoria struct {
	Version     dominiobolsa.VersionConvocatoriaGobernada
	Esperada    ReferenciaEstadoVersionConvocatoria
	Transaccion PreparacionTransaccionGobiernoConvocatoria
	// Has unexported fields.
}

func (c ConfirmacionActualizacionBorradorConvocatoria) Format(estado fmt.State, _ rune)

func (c ConfirmacionActualizacionBorradorConvocatoria) GoString() string

func (*ConfirmacionActualizacionBorradorConvocatoria) GobDecode([]byte) error

func (ConfirmacionActualizacionBorradorConvocatoria) GobEncode() ([]byte, error)

func (c ConfirmacionActualizacionBorradorConvocatoria) LogValue() slog.Value

func (ConfirmacionActualizacionBorradorConvocatoria) MarshalBinary() ([]byte, error)

func (ConfirmacionActualizacionBorradorConvocatoria) MarshalJSON() ([]byte, error)

func (ConfirmacionActualizacionBorradorConvocatoria) MarshalText() ([]byte, error)

func (ConfirmacionActualizacionBorradorConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (ConfirmacionActualizacionBorradorConvocatoria) String() string

func (*ConfirmacionActualizacionBorradorConvocatoria) UnmarshalBinary([]byte) error

func (*ConfirmacionActualizacionBorradorConvocatoria) UnmarshalJSON([]byte) error

func (*ConfirmacionActualizacionBorradorConvocatoria) UnmarshalText([]byte) error

func (*ConfirmacionActualizacionBorradorConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (c ConfirmacionActualizacionBorradorConvocatoria) Validar() error

func (c ConfirmacionActualizacionBorradorConvocatoria) ValidarRecibo(
	recibo ReciboGobiernoConvocatoria,
) error

type ConfirmacionAltaBorradorConvocatoria struct {
	Version             dominiobolsa.VersionConvocatoriaGobernada
	PredecesoraEsperada *ReferenciaEstadoVersionConvocatoria
	Predecesora         *dominiobolsa.VersionConvocatoriaGobernada
	Transaccion         PreparacionTransaccionGobiernoConvocatoria
	// Has unexported fields.
}

func (c ConfirmacionAltaBorradorConvocatoria) Format(estado fmt.State, _ rune)

func (c ConfirmacionAltaBorradorConvocatoria) GoString() string

func (*ConfirmacionAltaBorradorConvocatoria) GobDecode([]byte) error

func (ConfirmacionAltaBorradorConvocatoria) GobEncode() ([]byte, error)

func (c ConfirmacionAltaBorradorConvocatoria) LogValue() slog.Value

func (ConfirmacionAltaBorradorConvocatoria) MarshalBinary() ([]byte, error)

func (ConfirmacionAltaBorradorConvocatoria) MarshalJSON() ([]byte, error)

func (ConfirmacionAltaBorradorConvocatoria) MarshalText() ([]byte, error)

func (ConfirmacionAltaBorradorConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (ConfirmacionAltaBorradorConvocatoria) String() string

func (*ConfirmacionAltaBorradorConvocatoria) UnmarshalBinary([]byte) error

func (*ConfirmacionAltaBorradorConvocatoria) UnmarshalJSON([]byte) error

func (*ConfirmacionAltaBorradorConvocatoria) UnmarshalText([]byte) error

func (*ConfirmacionAltaBorradorConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (c ConfirmacionAltaBorradorConvocatoria) Validar() error

func (c ConfirmacionAltaBorradorConvocatoria) ValidarRecibo(
	recibo ReciboGobiernoConvocatoria,
) error

type ConfirmacionPublicacionConvocatoria struct {
	VersionPublicada     dominiobolsa.VersionConvocatoriaGobernada
	PublicadaEsperada    ReferenciaEstadoVersionConvocatoria
	PredecesoraResultado *dominiobolsa.VersionConvocatoriaGobernada
	PredecesoraEsperada  *ReferenciaEstadoVersionConvocatoria
	Dependencias         AtestacionDependenciasConvocatoria
	Aprobacion           AtestacionAprobacionConvocatoria
	Transaccion          PreparacionTransaccionGobiernoConvocatoria
	// Has unexported fields.
}

func (c ConfirmacionPublicacionConvocatoria) Format(estado fmt.State, _ rune)

func (c ConfirmacionPublicacionConvocatoria) GoString() string

func (*ConfirmacionPublicacionConvocatoria) GobDecode([]byte) error

func (ConfirmacionPublicacionConvocatoria) GobEncode() ([]byte, error)

func (c ConfirmacionPublicacionConvocatoria) LogValue() slog.Value

func (ConfirmacionPublicacionConvocatoria) MarshalBinary() ([]byte, error)

func (ConfirmacionPublicacionConvocatoria) MarshalJSON() ([]byte, error)

func (ConfirmacionPublicacionConvocatoria) MarshalText() ([]byte, error)

func (ConfirmacionPublicacionConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (ConfirmacionPublicacionConvocatoria) String() string

func (*ConfirmacionPublicacionConvocatoria) UnmarshalBinary([]byte) error

func (*ConfirmacionPublicacionConvocatoria) UnmarshalJSON([]byte) error

func (*ConfirmacionPublicacionConvocatoria) UnmarshalText([]byte) error

func (*ConfirmacionPublicacionConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (c ConfirmacionPublicacionConvocatoria) Validar() error

func (c ConfirmacionPublicacionConvocatoria) ValidarRecibo(
	recibo ReciboGobiernoConvocatoria,
) error

type ConfirmacionRetiradaConvocatoria struct {
	Version     dominiobolsa.VersionConvocatoriaGobernada
	Esperada    ReferenciaEstadoVersionConvocatoria
	Aprobacion  AtestacionAprobacionConvocatoria
	Transaccion PreparacionTransaccionGobiernoConvocatoria
	// Has unexported fields.
}

func (c ConfirmacionRetiradaConvocatoria) Format(estado fmt.State, _ rune)

func (c ConfirmacionRetiradaConvocatoria) GoString() string

func (*ConfirmacionRetiradaConvocatoria) GobDecode([]byte) error

func (ConfirmacionRetiradaConvocatoria) GobEncode() ([]byte, error)

func (c ConfirmacionRetiradaConvocatoria) LogValue() slog.Value

func (ConfirmacionRetiradaConvocatoria) MarshalBinary() ([]byte, error)

func (ConfirmacionRetiradaConvocatoria) MarshalJSON() ([]byte, error)

func (ConfirmacionRetiradaConvocatoria) MarshalText() ([]byte, error)

func (ConfirmacionRetiradaConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (ConfirmacionRetiradaConvocatoria) String() string

func (*ConfirmacionRetiradaConvocatoria) UnmarshalBinary([]byte) error

func (*ConfirmacionRetiradaConvocatoria) UnmarshalJSON([]byte) error

func (*ConfirmacionRetiradaConvocatoria) UnmarshalText([]byte) error

func (*ConfirmacionRetiradaConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (c ConfirmacionRetiradaConvocatoria) Validar() error

func (c ConfirmacionRetiradaConvocatoria) ValidarRecibo(
	recibo ReciboGobiernoConvocatoria,
) error

type ConsultaAutorizadaReglasBaremo interface {
	ObtenerVersionExacta(
		context.Context,
		SolicitudConsultaExactaReglasBaremo,
	) (ResultadoConsultaExactaReglasBaremo, error)
}
```

ConsultaAutorizadaReglasBaremo no resuelve alias temporales ni selecciona
versiones por orden de insercion.

```go
type ConsultaCategoriasPublicas interface {
	ObtenerPublicadas(context.Context, time.Time) (CatalogoCategoriasPublicas, error)
}
```

ConsultaCategoriasPublicas separa el catalogo profesional de la fuente
de convocatorias. Su adaptador debe fijar ID y version al construirse y
devolver solo entradas publicadas, vigentes y publicables para el instante
indicado.

```go
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

type ConsultaGobiernoConvocatorias interface {
	ObtenerVersionExacta(
		context.Context,
		SolicitudConsultaVersionConvocatoriaAutorizada,
	) (ResultadoConsultaVersionConvocatoria, error)
}
```

ConsultaGobiernoConvocatorias debe leer exactamente una version y registrar
la lectura junto con el uso de autorizacion en la misma transaccion.
La preimagen auditada incluye HuellaVersionSHA256 y, cuando exista,
HuellaInstanciaFlujoSHA256; la huella del registro no es decorativa.

```go
type ConsultaPanelInterno interface {
	ConsultarPanel(
		context.Context,
		SolicitudConsultaPanelInterno,
	) (InstantaneaPanelInterno, error)
}
```

ConsultaPanelInterno no es un DAO libre. La implementacion productiva debe
consumir la evidencia una sola vez, auditar el acceso y devolver el panel
solo despues de confirmar la transaccion.

```go
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
type ConteoCategoriaConvocatorias struct {
	NumeroConvocatorias  int
	NumeroPlazosAbiertos int
}

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

type DatosAtestacionAprobacionConvocatoria struct {
	Evidencia                 dominiobolsa.EvidenciaAprobacionConvocatoria
	RevisionVersion           int
	HuellaEstadoVersionSHA256 string
	VerificadorRef            string
	AtestacionRef             string
	HuellaAtestacionSHA256    string
	TokenConsumoRef           string
	AtestacionEmitidaEn       time.Time
	AtestacionValidaHasta     time.Time
	// Has unexported fields.
}

func (d DatosAtestacionAprobacionConvocatoria) Format(estado fmt.State, _ rune)

func (d DatosAtestacionAprobacionConvocatoria) GoString() string

func (*DatosAtestacionAprobacionConvocatoria) GobDecode([]byte) error

func (DatosAtestacionAprobacionConvocatoria) GobEncode() ([]byte, error)

func (d DatosAtestacionAprobacionConvocatoria) LogValue() slog.Value

func (DatosAtestacionAprobacionConvocatoria) MarshalBinary() ([]byte, error)

func (DatosAtestacionAprobacionConvocatoria) MarshalJSON() ([]byte, error)

func (DatosAtestacionAprobacionConvocatoria) MarshalText() ([]byte, error)

func (DatosAtestacionAprobacionConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (DatosAtestacionAprobacionConvocatoria) String() string

func (*DatosAtestacionAprobacionConvocatoria) UnmarshalBinary([]byte) error

func (*DatosAtestacionAprobacionConvocatoria) UnmarshalJSON([]byte) error

func (*DatosAtestacionAprobacionConvocatoria) UnmarshalText([]byte) error

func (*DatosAtestacionAprobacionConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

type DatosAtestacionDependenciasConvocatoria struct {
	Evidencia                 dominiobolsa.EvidenciaDependenciasConvocatoria
	RevisionVersion           int
	HuellaEstadoVersionSHA256 string
	VerificadorRef            string
	AtestacionRef             string
	HuellaAtestacionSHA256    string
	TokenConsumoRef           string
	AtestacionEmitidaEn       time.Time
	AtestacionValidaHasta     time.Time
	// Has unexported fields.
}

func (d DatosAtestacionDependenciasConvocatoria) Format(estado fmt.State, _ rune)

func (d DatosAtestacionDependenciasConvocatoria) GoString() string

func (*DatosAtestacionDependenciasConvocatoria) GobDecode([]byte) error

func (DatosAtestacionDependenciasConvocatoria) GobEncode() ([]byte, error)

func (d DatosAtestacionDependenciasConvocatoria) LogValue() slog.Value

func (DatosAtestacionDependenciasConvocatoria) MarshalBinary() ([]byte, error)

func (DatosAtestacionDependenciasConvocatoria) MarshalJSON() ([]byte, error)

func (DatosAtestacionDependenciasConvocatoria) MarshalText() ([]byte, error)

func (DatosAtestacionDependenciasConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (DatosAtestacionDependenciasConvocatoria) String() string

func (*DatosAtestacionDependenciasConvocatoria) UnmarshalBinary([]byte) error

func (*DatosAtestacionDependenciasConvocatoria) UnmarshalJSON([]byte) error

func (*DatosAtestacionDependenciasConvocatoria) UnmarshalText([]byte) error

func (*DatosAtestacionDependenciasConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

type DatosAtestacionSelladoMotivoConvocatoria struct {
	HMAC                   HMACMotivoGobiernoConvocatoria
	Accion                 string
	ConvocatoriaRef        string
	PrincipalRef           string
	CorrelacionRef         string
	HuellaSolicitudSHA256  string
	SelladorRef            string
	AtestacionRef          string
	HuellaAtestacionSHA256 string
	TokenConsumoRef        string
	AtestacionEmitidaEn    time.Time
	AtestacionValidaHasta  time.Time
	// Has unexported fields.
}

func (b DatosAtestacionSelladoMotivoConvocatoria) Format(estado fmt.State, _ rune)

func (b DatosAtestacionSelladoMotivoConvocatoria) GoString() string

func (*DatosAtestacionSelladoMotivoConvocatoria) GobDecode([]byte) error

func (DatosAtestacionSelladoMotivoConvocatoria) GobEncode() ([]byte, error)

func (b DatosAtestacionSelladoMotivoConvocatoria) LogValue() slog.Value

func (DatosAtestacionSelladoMotivoConvocatoria) MarshalBinary() ([]byte, error)

func (DatosAtestacionSelladoMotivoConvocatoria) MarshalJSON() ([]byte, error)

func (DatosAtestacionSelladoMotivoConvocatoria) MarshalText() ([]byte, error)

func (DatosAtestacionSelladoMotivoConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (DatosAtestacionSelladoMotivoConvocatoria) String() string

func (*DatosAtestacionSelladoMotivoConvocatoria) UnmarshalBinary([]byte) error

func (*DatosAtestacionSelladoMotivoConvocatoria) UnmarshalJSON([]byte) error

func (*DatosAtestacionSelladoMotivoConvocatoria) UnmarshalText([]byte) error

func (*DatosAtestacionSelladoMotivoConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

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

type DatosTestimonioIdempotenciaConvocatoria struct {
	Version                   uint16
	GeneracionClave           uint32
	ClaveHMACRef              string
	ProtectorRef              string
	AtestacionRef             string
	HuellaAtestacionSHA256    string
	IndiceOperacionHMACSHA256 string
	PrincipalRef              string
	HuellaIntencionSHA256     string
	EmitidoEn                 time.Time
	ValidoHasta               time.Time
	// Has unexported fields.
}

func (b DatosTestimonioIdempotenciaConvocatoria) Format(estado fmt.State, _ rune)

func (b DatosTestimonioIdempotenciaConvocatoria) GoString() string

func (*DatosTestimonioIdempotenciaConvocatoria) GobDecode([]byte) error

func (DatosTestimonioIdempotenciaConvocatoria) GobEncode() ([]byte, error)

func (b DatosTestimonioIdempotenciaConvocatoria) LogValue() slog.Value

func (DatosTestimonioIdempotenciaConvocatoria) MarshalBinary() ([]byte, error)

func (DatosTestimonioIdempotenciaConvocatoria) MarshalJSON() ([]byte, error)

func (DatosTestimonioIdempotenciaConvocatoria) MarshalText() ([]byte, error)

func (DatosTestimonioIdempotenciaConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (DatosTestimonioIdempotenciaConvocatoria) String() string

func (*DatosTestimonioIdempotenciaConvocatoria) UnmarshalBinary([]byte) error

func (*DatosTestimonioIdempotenciaConvocatoria) UnmarshalJSON([]byte) error

func (*DatosTestimonioIdempotenciaConvocatoria) UnmarshalText([]byte) error

func (*DatosTestimonioIdempotenciaConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

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
type FuenteExactaCalculoReglasBaremo struct {
	Version             reglas.VersionGobernadaReglasBaremo
	Entrada             calculo.EntradaExperiencia
	Prueba              PruebaFuenteExactaCalculoReglasBaremo
	Auditoria           reglas.ReferenciaVersionada
	ConsumoAutorizacion reglas.ReferenciaVersionada
	ConsumoPrueba       reglas.ReferenciaVersionada
	ObtenidaEn          time.Time
}

type FuenteReglasBaremoParaCalculo interface {
	ObtenerFuenteExacta(
		context.Context,
		SolicitudFuenteExactaCalculoReglasBaremo,
	) (FuenteExactaCalculoReglasBaremo, error)
}
```

FuenteReglasBaremoParaCalculo obtiene y verifica la procedencia de una
instantanea exacta. El adaptador devuelve el consumo durable de la prueba;
una entrada restaurada localmente no satisface este contrato.

NO-GO PRODUCCION: el contrato actual no prueba todavia que
ConsumoAutorizacion ligue de forma durable decision_ref, huella de
decision V2, recurso y correlacion exactos. Ningun adaptador satisface
la autorizacion de produccion hasta incorporar y verificar esa atestacion
tipada.

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

type HMACMotivoGobiernoConvocatoria struct {
	DominioCriptografico string
	GeneracionClave      uint32
	ClaveHMACRef         string
	ValorHMACSHA256      string
	// Has unexported fields.
}
```

HMACMotivoGobiernoConvocatoria identifica dominio y generacion de clave.
No contiene la clave ni permite calcular otro HMAC.

```go
func (h HMACMotivoGobiernoConvocatoria) Format(estado fmt.State, _ rune)

func (h HMACMotivoGobiernoConvocatoria) GoString() string

func (*HMACMotivoGobiernoConvocatoria) GobDecode([]byte) error

func (HMACMotivoGobiernoConvocatoria) GobEncode() ([]byte, error)

func (h HMACMotivoGobiernoConvocatoria) LogValue() slog.Value

func (HMACMotivoGobiernoConvocatoria) MarshalBinary() ([]byte, error)

func (HMACMotivoGobiernoConvocatoria) MarshalJSON() ([]byte, error)

func (HMACMotivoGobiernoConvocatoria) MarshalText() ([]byte, error)

func (HMACMotivoGobiernoConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (HMACMotivoGobiernoConvocatoria) String() string

func (*HMACMotivoGobiernoConvocatoria) UnmarshalBinary([]byte) error

func (*HMACMotivoGobiernoConvocatoria) UnmarshalJSON([]byte) error

func (*HMACMotivoGobiernoConvocatoria) UnmarshalText([]byte) error

func (*HMACMotivoGobiernoConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (h HMACMotivoGobiernoConvocatoria) Validar() error

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

type IndicadoresPanelInterno struct {
	ConvocatoriasBorrador        int `json:"convocatorias_borrador"`
	ConvocatoriasRevision        int `json:"convocatorias_revision"`
	ConvocatoriasPendientesFirma int `json:"convocatorias_pendientes_firma"`
	ConvocatoriasPublicadas      int `json:"convocatorias_publicadas"`
	BolsasActivas                int `json:"bolsas_activas"`
	BolsasSuspendidas            int `json:"bolsas_suspendidas"`
	BolsasAgotadas               int `json:"bolsas_agotadas"`
	LlamamientosPendientes       int `json:"llamamientos_pendientes"`
	LlamamientosEnCurso          int `json:"llamamientos_en_curso"`
	LlamamientosVencenHoy        int `json:"llamamientos_vencen_hoy"`
	DocumentosPendientesFirma    int `json:"documentos_pendientes_firma"`
	IncidenciasAbiertas          int `json:"incidencias_abiertas"`
}
```

IndicadoresPanelInterno contiene exclusivamente magnitudes agregadas.
No transporta identidades ni permite reconstruir un listado de personas.

```go
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

type InstantaneaPanelInterno struct {
	Esquema               string                            `json:"esquema"`
	Selector              SelectorPanelInterno              `json:"selector"`
	Origen                OrigenPanelInterno                `json:"origen"`
	PruebaLectura         PruebaLecturaPanelInterno         `json:"prueba_lectura"`
	Indicadores           IndicadoresPanelInterno           `json:"indicadores"`
	Convocatorias         []ResumenConvocatoriaPanelInterno `json:"convocatorias"`
	ActuacionesPendientes []ActuacionPendientePanelInterno  `json:"actuaciones_pendientes"`
}
```

InstantaneaPanelInterno es el contrato minimo del cuadro operativo. No hay
campos de nombre, documento identificativo, correo, telefono, direccion ni
colecciones globales de candidatos.

```go
func (i InstantaneaPanelInterno) ClonarValidadaPara(
	solicitud SolicitudConsultaPanelInterno,
) (InstantaneaPanelInterno, error)
```

ClonarValidadaPara aplica copia defensiva y coteja el selector exacto. Un
origen de demostracion o una prueba de lectura incoherente fallan cerrado.

```go
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

type MaterialIntencionGobiernoConvocatoria struct {
	Esquema                     string                               `json:"esquema"`
	Accion                      string                               `json:"accion"`
	EstadoPrincipalEsperado     *ReferenciaEstadoVersionConvocatoria `json:"estado_principal_esperado,omitempty"`
	EstadoPrincipalNuevo        ReferenciaEstadoVersionConvocatoria  `json:"estado_principal_nuevo"`
	EstadoRelacionadoEsperado   *ReferenciaEstadoVersionConvocatoria `json:"estado_relacionado_esperado,omitempty"`
	EstadoRelacionadoNuevo      *ReferenciaEstadoVersionConvocatoria `json:"estado_relacionado_nuevo,omitempty"`
	DominioCriptograficoMotivo  string                               `json:"dominio_criptografico_motivo"`
	GeneracionClaveMotivo       uint32                               `json:"generacion_clave_motivo"`
	HuellaSolicitudMotivoSHA256 string                               `json:"huella_solicitud_motivo_sha256"`
	HuellaMotivoHMACSHA256      string                               `json:"huella_motivo_hmac_sha256"`
}
```

MaterialIntencionGobiernoConvocatoria es la preimagen semantica estable de
una mutacion. Solo contiene referencias y huellas; nunca motivos en claro.

```go
func MaterialActualizacionBorradorConvocatoria(
	esperada ReferenciaEstadoVersionConvocatoria,
	version dominiobolsa.VersionConvocatoriaGobernada,
	motivo AtestacionSelladoMotivoConvocatoria,
) (MaterialIntencionGobiernoConvocatoria, error)

func MaterialAltaBorradorConvocatoria(
	version dominiobolsa.VersionConvocatoriaGobernada,
	predecesora *ReferenciaEstadoVersionConvocatoria,
	versionPredecesora *dominiobolsa.VersionConvocatoriaGobernada,
	motivo AtestacionSelladoMotivoConvocatoria,
) (MaterialIntencionGobiernoConvocatoria, error)

func MaterialPublicacionConvocatoria(
	esperada ReferenciaEstadoVersionConvocatoria,
	version dominiobolsa.VersionConvocatoriaGobernada,
	predecesoraEsperada *ReferenciaEstadoVersionConvocatoria,
	predecesoraResultado *dominiobolsa.VersionConvocatoriaGobernada,
	motivo AtestacionSelladoMotivoConvocatoria,
) (MaterialIntencionGobiernoConvocatoria, error)
```

MaterialPublicacionConvocatoria exige la predecesora para toda secuencia
posterior a la primera. Publicada pasa a sustituida; retirada se relee y se
devuelve sin alteracion. En ambos casos el repositorio bloquea ambas filas.

```go
func MaterialRetiradaConvocatoria(
	esperada ReferenciaEstadoVersionConvocatoria,
	version dominiobolsa.VersionConvocatoriaGobernada,
	motivo AtestacionSelladoMotivoConvocatoria,
) (MaterialIntencionGobiernoConvocatoria, error)

func (m MaterialIntencionGobiernoConvocatoria) HuellaSHA256() (string, error)

func (m MaterialIntencionGobiernoConvocatoria) Validar() error

type MetadatosFuenteCategorias struct {
	Revision      string
	ActualizadaEn time.Time
	Demostracion  bool
	Aviso         string
}

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

type OperacionGobiernoReglasBaremo string
```

OperacionGobiernoReglasBaremo identifica el efecto durable ya preparado por
application. No ejecuta ni valida transiciones de dominio.

```go
const (
	OperacionAltaBorradorReglasBaremo OperacionGobiernoReglasBaremo = "alta_borrador"
	OperacionPublicarReglasBaremo     OperacionGobiernoReglasBaremo = "publicar"
	OperacionActivarReglasBaremo      OperacionGobiernoReglasBaremo = "activar"
	OperacionSustituirReglasBaremo    OperacionGobiernoReglasBaremo = "sustituir"
	OperacionRetirarReglasBaremo      OperacionGobiernoReglasBaremo = "retirar"
	OperacionDescartarReglasBaremo    OperacionGobiernoReglasBaremo = "descartar"
)
type OrdenConfirmacionReglasBaremo struct {
	Operacion        OperacionGobiernoReglasBaremo
	Intencion        reglas.ReferenciaVersionada
	EstadoEsperado   *reglas.VinculoEstadoReglasBaremo
	VersionResultado reglas.VersionGobernadaReglasBaremo
	Autorizacion     puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	PruebaTransicion *reglas.ReferenciaVersionada
	EfectuarEn       time.Time
}
```

OrdenConfirmacionReglasBaremo contiene exclusivamente material ya derivado
y validado por application. Intencion es una referencia exacta al material
idempotente canonico creado fuera de ports.

EstadoEsperado hace explicito el CAS: nil solo es admisible para el alta;
en el resto de operaciones contiene identificador, version, revision y
huellas exactas. El adaptador no calcula ninguna transicion.

```go
type OrigenPanelInterno struct {
	Revision      string    `json:"revision"`
	ActualizadaEn time.Time `json:"actualizada_en"`
	Demostracion  bool      `json:"demostracion"`
}
```

OrigenPanelInterno permite rechazar de forma positiva adaptadores de
demostracion. El servicio productivo solo acepta Demostracion=false.

```go
type PaginaConvocatorias struct {
	Convocatorias []dominiobolsa.Convocatoria
	Total         int
	Catalogos     []CatalogoPublico
	// ConteosCategorias aplica todos los filtros salvo Categoria. Permite
	// construir facetas navegables sin que la opcion seleccionada oculte las
	// restantes y sin convertir la fuente en autoridad del catalogo.
	ConteosCategorias map[string]ConteoCategoriaConvocatorias
	Fuente            MetadatosFuenteConvocatorias
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

type PreparacionTransaccionGobiernoConvocatoria struct {
	Material      MaterialIntencionGobiernoConvocatoria
	Idempotencia  TestimonioIdempotenciaConvocatoria
	Autorizacion  puertosvec.EvidenciaUsoDecisionAutorizacion
	SelladoMotivo AtestacionSelladoMotivoConvocatoria
	SolicitadaEn  time.Time
	// Has unexported fields.
}

func (b PreparacionTransaccionGobiernoConvocatoria) Format(estado fmt.State, _ rune)

func (b PreparacionTransaccionGobiernoConvocatoria) GoString() string

func (*PreparacionTransaccionGobiernoConvocatoria) GobDecode([]byte) error

func (PreparacionTransaccionGobiernoConvocatoria) GobEncode() ([]byte, error)

func (b PreparacionTransaccionGobiernoConvocatoria) LogValue() slog.Value

func (PreparacionTransaccionGobiernoConvocatoria) MarshalBinary() ([]byte, error)

func (PreparacionTransaccionGobiernoConvocatoria) MarshalJSON() ([]byte, error)

func (PreparacionTransaccionGobiernoConvocatoria) MarshalText() ([]byte, error)

func (PreparacionTransaccionGobiernoConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (PreparacionTransaccionGobiernoConvocatoria) String() string

func (*PreparacionTransaccionGobiernoConvocatoria) UnmarshalBinary([]byte) error

func (*PreparacionTransaccionGobiernoConvocatoria) UnmarshalJSON([]byte) error

func (*PreparacionTransaccionGobiernoConvocatoria) UnmarshalText([]byte) error

func (*PreparacionTransaccionGobiernoConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (p PreparacionTransaccionGobiernoConvocatoria) Validar() error

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

type ProtectorIdempotenciaConvocatorias interface {
	Proteger(
		context.Context,
		SolicitudProtegerIdempotenciaConvocatoria,
	) (TestimonioIdempotenciaConvocatoria, error)
}
```

ProtectorIdempotenciaConvocatorias registra antes de devolver el testimonio
la atestacion exacta, su generacion de clave y el indice HMAC. El
repositorio de gobierno no confia en la copia recibida: relee ese registro
durable.

```go
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

type PruebaFuenteExactaCalculoReglasBaremo struct {
	Evidencia           reglas.ReferenciaVersionada
	Verificador         reglas.ReferenciaVersionada
	EstadoReglas        reglas.VinculoEstadoReglasBaremo
	InstantaneaEntrada  reglas.ReferenciaVersionada
	HuellaEntradaSHA256 string
	SujetoPseudonimo    reglas.ReferenciaVersionada
	Convocatoria        reglas.ReferenciaVersionada
	EmitidaEn           time.Time
	ValidaHasta         time.Time
}
```

PruebaFuenteExactaCalculoReglasBaremo es compacta: liga la evidencia
verificable a la instantanea y su contenido, sin repetir tramos ni
catalogos.

```go
type PruebaLecturaPanelInterno struct {
	LecturaRef           string    `json:"lectura_ref"`
	AuditoriaRef         string    `json:"auditoria_ref"`
	AuditoriaSecuencia   uint64    `json:"auditoria_secuencia"`
	DecisionRef          string    `json:"decision_ref"`
	HuellaDecisionSHA256 string    `json:"huella_decision_sha256"`
	CorrelacionRef       string    `json:"correlacion_ref"`
	ConfirmadaEn         time.Time `json:"confirmada_en"`
}
```

PruebaLecturaPanelInterno acredita que la consulta y su auditoria quedaron
confirmadas. Solo contiene referencias tecnicas opacas.

```go
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
type ReciboConfirmacionReglasBaremo struct {
	Operacion               OperacionGobiernoReglasBaremo
	Intencion               reglas.ReferenciaVersionada
	EstadoResultado         reglas.VinculoEstadoReglasBaremo
	Transaccion             reglas.ReferenciaVersionada
	Auditoria               reglas.ReferenciaVersionada
	EventoOutbox            reglas.ReferenciaVersionada
	ConsumoAutorizacion     reglas.ReferenciaVersionada
	ConsumoPruebaTransicion *reglas.ReferenciaVersionada
	ConfirmadaEn            time.Time
}
```

ReciboConfirmacionReglasBaremo referencia los efectos confirmados por una
unica transaccion. No constituye por si mismo la validacion del resultado;
application coteja estos vinculos exactos con la orden original.

```go
type ReciboConsumoVerificacionConvocatoria struct {
	TokenConsumoRef        string
	AtestacionRef          string
	HuellaAtestacionSHA256 string
	// Has unexported fields.
}

func (b ReciboConsumoVerificacionConvocatoria) Format(estado fmt.State, _ rune)

func (b ReciboConsumoVerificacionConvocatoria) GoString() string

func (*ReciboConsumoVerificacionConvocatoria) GobDecode([]byte) error

func (ReciboConsumoVerificacionConvocatoria) GobEncode() ([]byte, error)

func (b ReciboConsumoVerificacionConvocatoria) LogValue() slog.Value

func (ReciboConsumoVerificacionConvocatoria) MarshalBinary() ([]byte, error)

func (ReciboConsumoVerificacionConvocatoria) MarshalJSON() ([]byte, error)

func (ReciboConsumoVerificacionConvocatoria) MarshalText() ([]byte, error)

func (ReciboConsumoVerificacionConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (ReciboConsumoVerificacionConvocatoria) String() string

func (*ReciboConsumoVerificacionConvocatoria) UnmarshalBinary([]byte) error

func (*ReciboConsumoVerificacionConvocatoria) UnmarshalJSON([]byte) error

func (*ReciboConsumoVerificacionConvocatoria) UnmarshalText([]byte) error

func (*ReciboConsumoVerificacionConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

type ReciboGobiernoConvocatoria struct {
	TransaccionRef                     string
	Accion                             string
	EstadoPrincipal                    ReferenciaEstadoVersionConvocatoria
	EstadoRelacionado                  *ReferenciaEstadoVersionConvocatoria
	PrincipalRef                       string
	AutorizacionRef                    string
	HuellaAutorizacionSHA256           string
	AtestacionAutorizacionRef          string
	HuellaAtestacionAutorizacionSHA256 string
	ConsumoAutorizacionRef             string
	IndiceIdempotenciaHMACSHA256       string
	AtestacionIdempotenciaRef          string
	HuellaAtestacionIdempotenciaSHA256 string
	HuellaIntencionSHA256              string
	AuditoriaRef                       string
	HuellaAuditoriaSHA256              string
	EventoOutboxRef                    string
	HuellaEventoOutboxSHA256           string
	ConsumoMotivo                      *ReciboConsumoVerificacionConvocatoria
	ConsumoDependencias                *ReciboConsumoVerificacionConvocatoria
	ConsumoAprobacion                  *ReciboConsumoVerificacionConvocatoria
	ConfirmadaEn                       time.Time
	// Has unexported fields.
}
```

ReciboGobiernoConvocatoria es la prueba minima devuelta tras COMMIT. Liga
el estado confirmado con la decision consumida, la intencion idempotente, el
registro de auditoria y el evento outbox. No contiene atributos personales
directos; es interno y PrincipalRef sigue siendo una referencia seudonima.

```go
func (r ReciboGobiernoConvocatoria) Format(estado fmt.State, _ rune)

func (r ReciboGobiernoConvocatoria) GoString() string

func (*ReciboGobiernoConvocatoria) GobDecode([]byte) error

func (ReciboGobiernoConvocatoria) GobEncode() ([]byte, error)

func (r ReciboGobiernoConvocatoria) LogValue() slog.Value

func (ReciboGobiernoConvocatoria) MarshalBinary() ([]byte, error)

func (ReciboGobiernoConvocatoria) MarshalJSON() ([]byte, error)

func (ReciboGobiernoConvocatoria) MarshalText() ([]byte, error)

func (ReciboGobiernoConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (ReciboGobiernoConvocatoria) String() string

func (*ReciboGobiernoConvocatoria) UnmarshalBinary([]byte) error

func (*ReciboGobiernoConvocatoria) UnmarshalJSON([]byte) error

func (*ReciboGobiernoConvocatoria) UnmarshalText([]byte) error

func (*ReciboGobiernoConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (r ReciboGobiernoConvocatoria) ValidarPara(
	preparacion PreparacionTransaccionGobiernoConvocatoria,
) error

type RecuperadorBinarioFirmado interface {
	RecuperarBinarioFirmado(context.Context, SolicitudRecuperarBinarioFirmado) (BinarioFirmadoRecuperado, error)
}

type ReferenciaDespliegueIdempotenciaBaremacion string

func (r ReferenciaDespliegueIdempotenciaBaremacion) Valida() bool

type ReferenciaEstadoVersionConvocatoria struct {
	Referencia         string `json:"referencia"`
	Revision           int    `json:"revision"`
	HuellaEstadoSHA256 string `json:"huella_estado_sha256"`
}
```

ReferenciaEstadoVersionConvocatoria fija referencia, revision y huella del
agregado completo. No es una referencia a contenido ni a «la ultima» fila.

```go
func EstadoVersionConvocatoria(
	version dominiobolsa.VersionConvocatoriaGobernada,
) (ReferenciaEstadoVersionConvocatoria, error)

func (r ReferenciaEstadoVersionConvocatoria) Validar() error

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

type RepositorioGobiernoConvocatorias interface {
	ConfirmarAltaBorrador(context.Context, ConfirmacionAltaBorradorConvocatoria) (ReciboGobiernoConvocatoria, error)
	ConfirmarActualizacionBorrador(context.Context, ConfirmacionActualizacionBorradorConvocatoria) (ReciboGobiernoConvocatoria, error)
	ConfirmarPublicacion(context.Context, ConfirmacionPublicacionConvocatoria) (ReciboGobiernoConvocatoria, error)
	ConfirmarRetirada(context.Context, ConfirmacionRetiradaConvocatoria) (ReciboGobiernoConvocatoria, error)
}
```

RepositorioGobiernoConvocatorias es la barrera durable, no un simple DAO.
Dentro de la MISMA transaccion debe bloquear y releer las filas afectadas,
comparar revision+huella, revalidar la decision registrada y su instantanea
de politicas y una atestacion PDP registrada/COSE cuya procedencia ya haya
sido verificada, releer el testimonio registrado del protector idempotente,
y verificar las atestaciones de sellado HSM/KMS, aprobacion y dependencias.
El indice (principal, HMAC idempotente) devuelve el recibo previo
si coincide la intencion y rechaza su reutilizacion si difiere.
Decision y tokens de sellado/verificacion se consumen una sola vez. Solo
despues confirma agregado(s), auditoria encadenada y outbox en un COMMIT
indivisible. Una validacion previa de la aplicacion nunca sustituye estas
comprobaciones. ConfirmarAlta bloquea y relee la predecesora de secuencia
>1; publicacion vuelve a bloquear ambas versiones y evita que dos ramas la
reclamen. La composicion productiva permanece NO-GO hasta disponer de ese
adaptador durable en el mismo TCB/BD; EvidenciaUsoDecisionAutorizacion por
si sola no acredita la procedencia del PDP.

```go
type RepositorioGobiernoReglasBaremo interface {
	Confirmar(
		context.Context,
		OrdenConfirmacionReglasBaremo,
	) (ReciboConfirmacionReglasBaremo, error)
}
```

RepositorioGobiernoReglasBaremo confirma en una sola transaccion durable la
idempotencia ya derivada, el CAS, la version resultante, el consumo de la
autorizacion V2, la prueba de transicion cuando exista, auditoria y outbox.
Ante cualquier fallo no confirma ningun efecto parcial.

```go
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
type ResultadoConsultaExactaReglasBaremo struct {
	Version             reglas.VersionGobernadaReglasBaremo
	Auditoria           reglas.ReferenciaVersionada
	ConsumoAutorizacion reglas.ReferenciaVersionada
	ConsultadaEn        time.Time
}

type ResultadoConsultaVersionConvocatoria struct {
	Version                            dominiobolsa.VersionConvocatoriaGobernada
	InstanciaFlujo                     *dominiovec.InstanciaFlujo
	HuellaVersionSHA256                string
	HuellaInstanciaFlujoSHA256         string
	AutorizacionRef                    string
	HuellaAutorizacionSHA256           string
	AtestacionAutorizacionRef          string
	HuellaAtestacionAutorizacionSHA256 string
	ConsumoAutorizacionRef             string
	AuditoriaRef                       string
	HuellaAuditoriaSHA256              string
	ConsultadaEn                       time.Time
	// Has unexported fields.
}

func (r ResultadoConsultaVersionConvocatoria) Clonar() (
	ResultadoConsultaVersionConvocatoria,
	error,
)

func (b ResultadoConsultaVersionConvocatoria) Format(estado fmt.State, _ rune)

func (b ResultadoConsultaVersionConvocatoria) GoString() string

func (*ResultadoConsultaVersionConvocatoria) GobDecode([]byte) error

func (ResultadoConsultaVersionConvocatoria) GobEncode() ([]byte, error)

func (b ResultadoConsultaVersionConvocatoria) LogValue() slog.Value

func (ResultadoConsultaVersionConvocatoria) MarshalBinary() ([]byte, error)

func (ResultadoConsultaVersionConvocatoria) MarshalJSON() ([]byte, error)

func (ResultadoConsultaVersionConvocatoria) MarshalText() ([]byte, error)

func (ResultadoConsultaVersionConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (ResultadoConsultaVersionConvocatoria) String() string

func (*ResultadoConsultaVersionConvocatoria) UnmarshalBinary([]byte) error

func (*ResultadoConsultaVersionConvocatoria) UnmarshalJSON([]byte) error

func (*ResultadoConsultaVersionConvocatoria) UnmarshalText([]byte) error

func (*ResultadoConsultaVersionConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (r ResultadoConsultaVersionConvocatoria) ValidarPara(
	s SolicitudConsultaVersionConvocatoriaAutorizada,
) error

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

type ResumenConvocatoriaPanelInterno struct {
	ConvocatoriaRef   string    `json:"convocatoria_ref"`
	CategoriaClave    string    `json:"categoria_clave"`
	EstadoClave       string    `json:"estado_clave"`
	PlazoCierraEn     time.Time `json:"plazo_cierra_en,omitempty"`
	NumeroSolicitudes int       `json:"numero_solicitudes"`
	NumeroPendientes  int       `json:"numero_pendientes"`
}
```

ResumenConvocatoriaPanelInterno usa claves de catalogo y referencias opacas.
Las etiquetas se resuelven aparte desde catalogos gobernados.

```go
type SelectorFuenteExactaCalculoReglasBaremo = oficial.SelectorFuenteExactaCalculoReglasBaremo
```

SelectorFuenteExactaCalculoReglasBaremo conserva en la frontera hexagonal
el contrato canonico versionado del dominio. Sus metodos Validar,
RepresentacionCanonicaV1 y HuellaSHA256V1 son la unica fuente de verdad para
aplicacion y adaptadores; el algoritmo no debe copiarse fuera del dominio.

```go
type SelectorPanelInterno struct {
	Clase            ClaseAmbitoPanelInterno `json:"clase"`
	OrganizacionRef  string                  `json:"organizacion_ref"`
	UnidadGestionRef string                  `json:"unidad_gestion_ref,omitempty"`
}
```

SelectorPanelInterno no contiene identidad de la persona operadora.
Sus referencias proceden de configuracion interna y quedan ligadas al PDP.

```go
func (s SelectorPanelInterno) Validar() error

type SelectorVersionConvocatoriaExacta struct {
	ID        string
	Secuencia int
}
```

SelectorVersionConvocatoriaExacta impide consultas ambiguas o a «la ultima».

```go
func (s SelectorVersionConvocatoriaExacta) Referencia() string

func (s SelectorVersionConvocatoriaExacta) Validar() error

type SelladorMotivoGobiernoConvocatoria interface {
	SellarMotivo(
		context.Context,
		SolicitudSellarMotivoGobiernoConvocatoria,
	) (AtestacionSelladoMotivoConvocatoria, error)
}
```

SelladorMotivoGobiernoConvocatoria debe usar un HSM/KMS o servicio de claves
versionadas. El contrato no admite una clave recibida por parametro.

```go
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

type SolicitudComprobarAprobacionConvocatoria struct {
	Version       dominiobolsa.VersionConvocatoriaGobernada
	Accion        AccionAprobacionConvocatoria
	AprobacionRef string
	ComprobarEn   time.Time
	// Has unexported fields.
}

func (b SolicitudComprobarAprobacionConvocatoria) Format(estado fmt.State, _ rune)

func (b SolicitudComprobarAprobacionConvocatoria) GoString() string

func (*SolicitudComprobarAprobacionConvocatoria) GobDecode([]byte) error

func (SolicitudComprobarAprobacionConvocatoria) GobEncode() ([]byte, error)

func (b SolicitudComprobarAprobacionConvocatoria) LogValue() slog.Value

func (SolicitudComprobarAprobacionConvocatoria) MarshalBinary() ([]byte, error)

func (SolicitudComprobarAprobacionConvocatoria) MarshalJSON() ([]byte, error)

func (SolicitudComprobarAprobacionConvocatoria) MarshalText() ([]byte, error)

func (SolicitudComprobarAprobacionConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (SolicitudComprobarAprobacionConvocatoria) String() string

func (*SolicitudComprobarAprobacionConvocatoria) UnmarshalBinary([]byte) error

func (*SolicitudComprobarAprobacionConvocatoria) UnmarshalJSON([]byte) error

func (*SolicitudComprobarAprobacionConvocatoria) UnmarshalText([]byte) error

func (*SolicitudComprobarAprobacionConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (s SolicitudComprobarAprobacionConvocatoria) Validar() error

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

type SolicitudConsultaExactaReglasBaremo struct {
	Selector     reglas.VinculoEstadoReglasBaremo
	Autorizacion puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	SolicitadaEn time.Time
}
```

SolicitudConsultaExactaReglasBaremo solo admite un estado identificado por
contenido, version, revision y huellas exactas.

```go
type SolicitudConsultaPanelInterno struct {
	// Has unexported fields.
}
```

SolicitudConsultaPanelInterno es una capacidad opaca para el adaptador
durable. Este debe revalidar y consumir la decision V2 en la misma
transaccion que calcula los agregados y confirma la auditoria de lectura.

```go
func NuevaSolicitudConsultaPanelInterno(
	selector SelectorPanelInterno,
	autorizacion puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2,
	consultadaEn time.Time,
) (SolicitudConsultaPanelInterno, error)

func (s SolicitudConsultaPanelInterno) Autorizacion() (
	puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	error,
)

func (s SolicitudConsultaPanelInterno) ConsultadaEn() (time.Time, error)

func (s SolicitudConsultaPanelInterno) Correlacion() (
	dominiovec.ReferenciaCorrelacionAutorizacionV2,
	error,
)

func (s SolicitudConsultaPanelInterno) Motivo() (dominiovec.ReferenciaEntradaCatalogo, error)

func (s SolicitudConsultaPanelInterno) Selector() (SelectorPanelInterno, error)

func (SolicitudConsultaPanelInterno) String() string

type SolicitudConsultaVersionConvocatoriaAutorizada struct {
	Selector              SelectorVersionConvocatoriaExacta
	IncluirInstanciaFlujo bool
	Autorizacion          puertosvec.EvidenciaUsoDecisionAutorizacion
	ConsultadaEn          time.Time
	// Has unexported fields.
}

func (b SolicitudConsultaVersionConvocatoriaAutorizada) Format(estado fmt.State, _ rune)

func (b SolicitudConsultaVersionConvocatoriaAutorizada) GoString() string

func (*SolicitudConsultaVersionConvocatoriaAutorizada) GobDecode([]byte) error

func (SolicitudConsultaVersionConvocatoriaAutorizada) GobEncode() ([]byte, error)

func (b SolicitudConsultaVersionConvocatoriaAutorizada) LogValue() slog.Value

func (SolicitudConsultaVersionConvocatoriaAutorizada) MarshalBinary() ([]byte, error)

func (SolicitudConsultaVersionConvocatoriaAutorizada) MarshalJSON() ([]byte, error)

func (SolicitudConsultaVersionConvocatoriaAutorizada) MarshalText() ([]byte, error)

func (SolicitudConsultaVersionConvocatoriaAutorizada) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (SolicitudConsultaVersionConvocatoriaAutorizada) String() string

func (*SolicitudConsultaVersionConvocatoriaAutorizada) UnmarshalBinary([]byte) error

func (*SolicitudConsultaVersionConvocatoriaAutorizada) UnmarshalJSON([]byte) error

func (*SolicitudConsultaVersionConvocatoriaAutorizada) UnmarshalText([]byte) error

func (*SolicitudConsultaVersionConvocatoriaAutorizada) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (s SolicitudConsultaVersionConvocatoriaAutorizada) Validar() error

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

type SolicitudFuenteExactaCalculoReglasBaremo struct {
	Selector     SelectorFuenteExactaCalculoReglasBaremo
	Autorizacion puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	SolicitadaEn time.Time
}

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

type SolicitudProtegerIdempotenciaConvocatoria struct {
	ClaveIdempotencia string
	PrincipalRef      string
	Material          MaterialIntencionGobiernoConvocatoria
	SolicitadaEn      time.Time
	// Has unexported fields.
}
```

SolicitudProtegerIdempotenciaConvocatoria no es un DTO HTTP. La clave se
recibe de una frontera limitada y se convierte en indice HMAC antes de
llegar al repositorio de gobierno.

```go
func (s SolicitudProtegerIdempotenciaConvocatoria) Format(estado fmt.State, _ rune)

func (s SolicitudProtegerIdempotenciaConvocatoria) GoString() string

func (*SolicitudProtegerIdempotenciaConvocatoria) GobDecode([]byte) error

func (SolicitudProtegerIdempotenciaConvocatoria) GobEncode() ([]byte, error)

func (s SolicitudProtegerIdempotenciaConvocatoria) LogValue() slog.Value

func (SolicitudProtegerIdempotenciaConvocatoria) MarshalBinary() ([]byte, error)

func (SolicitudProtegerIdempotenciaConvocatoria) MarshalJSON() ([]byte, error)

func (SolicitudProtegerIdempotenciaConvocatoria) MarshalText() ([]byte, error)

func (SolicitudProtegerIdempotenciaConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (SolicitudProtegerIdempotenciaConvocatoria) String() string

func (*SolicitudProtegerIdempotenciaConvocatoria) UnmarshalBinary([]byte) error

func (*SolicitudProtegerIdempotenciaConvocatoria) UnmarshalJSON([]byte) error

func (*SolicitudProtegerIdempotenciaConvocatoria) UnmarshalText([]byte) error

func (*SolicitudProtegerIdempotenciaConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (s SolicitudProtegerIdempotenciaConvocatoria) Validar() error

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
type SolicitudSellarMotivoGobiernoConvocatoria struct {
	DominioCriptografico string
	Accion               string
	ConvocatoriaRef      string
	PrincipalRef         string
	CorrelacionRef       string
	Motivo               string
	SolicitadaEn         time.Time
	// Has unexported fields.
}
```

SolicitudSellarMotivoGobiernoConvocatoria es una orden interna. El motivo en
claro solo cruza este puerto hacia un servicio de claves; nunca se guarda en
idempotencia ni se registra en trazas.

```go
func (s SolicitudSellarMotivoGobiernoConvocatoria) Format(estado fmt.State, _ rune)

func (s SolicitudSellarMotivoGobiernoConvocatoria) GoString() string

func (*SolicitudSellarMotivoGobiernoConvocatoria) GobDecode([]byte) error

func (SolicitudSellarMotivoGobiernoConvocatoria) GobEncode() ([]byte, error)

func (s SolicitudSellarMotivoGobiernoConvocatoria) HuellaSHA256() (string, error)
```

HuellaSHA256 fija la preimagen que el sellador debe autenticar. El HSM/KMS
calcula su HMAC sobre dominio || 0x00 || esta huella, nunca solo sobre el
motivo. Asi quedan ligados accion, version, principal y correlacion.

```go
func (s SolicitudSellarMotivoGobiernoConvocatoria) LogValue() slog.Value

func (SolicitudSellarMotivoGobiernoConvocatoria) MarshalBinary() ([]byte, error)

func (SolicitudSellarMotivoGobiernoConvocatoria) MarshalJSON() ([]byte, error)

func (SolicitudSellarMotivoGobiernoConvocatoria) MarshalText() ([]byte, error)

func (SolicitudSellarMotivoGobiernoConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (SolicitudSellarMotivoGobiernoConvocatoria) String() string

func (*SolicitudSellarMotivoGobiernoConvocatoria) UnmarshalBinary([]byte) error

func (*SolicitudSellarMotivoGobiernoConvocatoria) UnmarshalJSON([]byte) error

func (*SolicitudSellarMotivoGobiernoConvocatoria) UnmarshalText([]byte) error

func (*SolicitudSellarMotivoGobiernoConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (s SolicitudSellarMotivoGobiernoConvocatoria) Validar() error

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

type SolicitudVerificarDependenciasConvocatoria struct {
	Version     dominiobolsa.VersionConvocatoriaGobernada
	VerificarEn time.Time
	// Has unexported fields.
}

func (s SolicitudVerificarDependenciasConvocatoria) Clonar() (
	SolicitudVerificarDependenciasConvocatoria,
	error,
)

func (b SolicitudVerificarDependenciasConvocatoria) Format(estado fmt.State, _ rune)

func (b SolicitudVerificarDependenciasConvocatoria) GoString() string

func (*SolicitudVerificarDependenciasConvocatoria) GobDecode([]byte) error

func (SolicitudVerificarDependenciasConvocatoria) GobEncode() ([]byte, error)

func (b SolicitudVerificarDependenciasConvocatoria) LogValue() slog.Value

func (SolicitudVerificarDependenciasConvocatoria) MarshalBinary() ([]byte, error)

func (SolicitudVerificarDependenciasConvocatoria) MarshalJSON() ([]byte, error)

func (SolicitudVerificarDependenciasConvocatoria) MarshalText() ([]byte, error)

func (SolicitudVerificarDependenciasConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (SolicitudVerificarDependenciasConvocatoria) String() string

func (*SolicitudVerificarDependenciasConvocatoria) UnmarshalBinary([]byte) error

func (*SolicitudVerificarDependenciasConvocatoria) UnmarshalJSON([]byte) error

func (*SolicitudVerificarDependenciasConvocatoria) UnmarshalText([]byte) error

func (*SolicitudVerificarDependenciasConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (s SolicitudVerificarDependenciasConvocatoria) Validar() error

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

type TestimonioIdempotenciaConvocatoria struct {
	// Has unexported fields.
}
```

TestimonioIdempotenciaConvocatoria es reconstruible y no acredita por si
solo la procedencia del protector. La barrera durable relee AtestacionRef y
su huella antes de consultar o crear el indice semantico.

```go
func NuevoTestimonioIdempotenciaConvocatoria(
	datos DatosTestimonioIdempotenciaConvocatoria,
) (TestimonioIdempotenciaConvocatoria, error)

func (t TestimonioIdempotenciaConvocatoria) Datos() (DatosTestimonioIdempotenciaConvocatoria, error)

func (t TestimonioIdempotenciaConvocatoria) Format(estado fmt.State, _ rune)

func (t TestimonioIdempotenciaConvocatoria) GoString() string

func (*TestimonioIdempotenciaConvocatoria) GobDecode([]byte) error

func (TestimonioIdempotenciaConvocatoria) GobEncode() ([]byte, error)

func (t TestimonioIdempotenciaConvocatoria) LogValue() slog.Value

func (TestimonioIdempotenciaConvocatoria) MarshalBinary() ([]byte, error)

func (TestimonioIdempotenciaConvocatoria) MarshalJSON() ([]byte, error)

func (TestimonioIdempotenciaConvocatoria) MarshalText() ([]byte, error)

func (TestimonioIdempotenciaConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (TestimonioIdempotenciaConvocatoria) String() string

func (*TestimonioIdempotenciaConvocatoria) UnmarshalBinary([]byte) error

func (*TestimonioIdempotenciaConvocatoria) UnmarshalJSON([]byte) error

func (*TestimonioIdempotenciaConvocatoria) UnmarshalText([]byte) error

func (*TestimonioIdempotenciaConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (t TestimonioIdempotenciaConvocatoria) ValidarPara(
	material MaterialIntencionGobiernoConvocatoria,
	principalRef string,
) error

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
		ComandoGuardarPropuestaLlamamiento,
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

type VerificadorAprobacionConvocatoria interface {
	ComprobarAprobacion(
		context.Context,
		SolicitudComprobarAprobacionConvocatoria,
	) (AtestacionAprobacionConvocatoria, error)
}
```

El repositorio debe releer la atestacion y consumir TokenConsumoRef dentro
de la misma transaccion que el cambio de gobierno.

```go
type VerificadorDependenciasConvocatoria interface {
	VerificarDependencias(
		context.Context,
		SolicitudVerificarDependenciasConvocatoria,
	) (AtestacionDependenciasConvocatoria, error)
}
```

El repositorio debe releer la atestacion y consumir TokenConsumoRef dentro
de la misma transaccion que la publicacion.

```go
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
