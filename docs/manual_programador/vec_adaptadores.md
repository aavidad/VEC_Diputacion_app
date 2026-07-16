# Nucleo VEC: adaptadores

Parte del [Manual del programador](LEEME.md). Fichero generado con
`scripts/generar_manual_programador.py`; no editar a mano.

## Paquete `internal/vec/adapters/almacen`

> Package almacen contiene adaptadores que componen el puerto documental con conectores de objetos.

Package almacen contiene adaptadores que componen el puerto documental con
conectores de objetos. El nucleo no conoce el proveedor seleccionado.

### Variables

```go
var (
	ErrIdentificadorConectorInvalido  = errors.New("vec: identificador de conector de almacen invalido")
	ErrConectorAlmacenYaRegistrado    = errors.New("vec: conector de almacen ya registrado")
	ErrConectorAlmacenNoRegistrado    = errors.New("vec: conector de almacen no registrado")
	ErrFabricaConectorAlmacenInvalida = errors.New("vec: fabrica de conector de almacen invalida")
	ErrRegistroConectoresInvalido     = errors.New("vec: registro de conectores de almacen invalido")
)
```

### Funciones

```go
func RegistrarS3Compatible(registro *RegistroConectoresAlmacen, identificador string) error
```

RegistrarS3Compatible instala la fabrica, pero no crea conexiones ni lee
credenciales. La sonda de capacidades se ejecuta al seleccionar el conector
mediante RegistroConectoresAlmacen.Crear.

```go
func RegistrarS3CompatiblePredeterminado(registro *RegistroConectoresAlmacen) error
```

### Tipos

```go
type ConfiguracionConectorAlmacen map[string]string
```

ConfiguracionConectorAlmacen solo circula por la raiz de composicion.
El nucleo y los modulos no reciben direcciones, buckets ni credenciales.

```go
type ContenidoDocumental struct {
	// Has unexported fields.
}
```

ContenidoDocumental adapta el contrato historico de generacion al puerto
transversal de objetos. Es una fachada temporal: las nuevas cargas deben
utilizar AlmacenObjetos y su ciclo de cuarentena directamente.

```go
func NuevoContenidoDocumental(
	ctx context.Context,
	objetos ports.AlmacenObjetos,
	requisitos ports.RequisitosAlmacenObjetos,
) (*ContenidoDocumental, error)

func (a *ContenidoDocumental) GuardarContenido(
	ctx context.Context,
	solicitud ports.SolicitudGuardarContenido,
) (ports.ContenidoDocumentoGuardado, error)

func (a *ContenidoDocumental) LeerContenido(
	ctx context.Context,
	solicitud ports.SolicitudLeerContenido,
) (ports.ContenidoDocumentoLeido, error)

type FabricaConectorAlmacen func(context.Context, ConfiguracionConectorAlmacen) (ports.AlmacenObjetos, error)

type RegistroConectoresAlmacen struct {
	// Has unexported fields.
}
```

RegistroConectoresAlmacen permite seleccionar por configuracion cualquiera
de los conectores instalados. Incorporar otro producto anade una fabrica,
no condicionales en el nucleo ni cambios en los casos de uso.

```go
func NuevoRegistroConectoresAlmacen() *RegistroConectoresAlmacen

func (r *RegistroConectoresAlmacen) Crear(
	ctx context.Context,
	identificador string,
	configuracion ConfiguracionConectorAlmacen,
	requisitos ports.RequisitosAlmacenObjetos,
) (ports.AlmacenObjetos, error)

func (r *RegistroConectoresAlmacen) Listar() []string

func (r *RegistroConectoresAlmacen) Registrar(identificador string, fabrica FabricaConectorAlmacen) error
```

## Paquete `internal/vec/adapters/almacen/s3`

> Package s3 implementa un conector de objetos compatible con la API S3.

Package s3 implementa un conector de objetos compatible con la API S3.
El paquete pertenece a la capa de adaptadores: ningun tipo del SDK cruza hacia
el nucleo ni hacia los modulos de negocio.

### Constantes

```go
const (
	IdentificadorPredeterminado = "s3_compatible"
)
```

### Variables

```go
var (
	ErrConfiguracionInvalida = errors.New("vec: configuracion del conector s3 invalida")
	ErrOperacionS3           = errors.New("vec: operacion del conector s3 no disponible")
	ErrSondaS3NoSuperada     = errors.New("vec: sonda fuerte del conector s3 no superada")
)
```

### Tipos

```go
type Almacen struct {
	// Has unexported fields.
}
```

Almacen es un adaptador, no un gestor de permisos. Solo ejecuta una
capacidad opaca ya emitida por el nucleo y la revalida inmediatamente antes
de cada efecto remoto.

```go
func Nuevo(ctx context.Context, configuracion Configuracion) (*Almacen, error)

func NuevoConCliente(
	ctx context.Context,
	configuracion Configuracion,
	cliente clienteSDK,
	presignador presignadorSDK,
	reloj ports.Reloj,
) (*Almacen, error)
```

NuevoConCliente es el punto de inyeccion para pruebas contractuales y para
cabinas que proporcionen un cliente S3 instrumentado por la organizacion.

```go
func (a *Almacen) AbandonarCargaDirecta(
	ctx context.Context,
	contexto ports.ContextoOperacionAlmacen,
	sesionRef string,
) error

func (a *Almacen) Abrir(ctx context.Context, solicitud ports.SolicitudAbrirObjeto) (ports.LecturaObjetoAlmacen, error)

func (a *Almacen) AplicarRetencion(ctx context.Context, solicitud ports.SolicitudRetenerObjeto) (ports.ResultadoOperacionObjeto, error)

func (a *Almacen) Capacidades(ctx context.Context) (ports.CapacidadesAlmacenObjetos, error)

func (a *Almacen) ConfirmarCargaDirecta(
	ctx context.Context,
	solicitud ports.SolicitudConfirmarCargaDirecta,
) (ports.ResultadoOperacionObjeto, error)

func (a *Almacen) Eliminar(ctx context.Context, solicitud ports.SolicitudEliminarObjeto) (ports.EvidenciaOperacionAlmacen, error)

func (a *Almacen) Escribir(ctx context.Context, solicitud ports.SolicitudEscribirObjeto) (ports.ResultadoOperacionObjeto, error)

func (a *Almacen) Inmovilizar(ctx context.Context, solicitud ports.SolicitudInmovilizarObjeto) (ports.ResultadoOperacionObjeto, error)

func (a *Almacen) LevantarInmovilizacion(ctx context.Context, solicitud ports.SolicitudLevantarInmovilizacionObjeto) (ports.ResultadoOperacionObjeto, error)

func (a *Almacen) PrepararCargaDirecta(
	ctx context.Context,
	solicitud ports.SolicitudPrepararCargaDirecta,
) (ports.InstruccionesCargaDirecta, error)

func (a *Almacen) Promover(ctx context.Context, solicitud ports.SolicitudPromoverObjeto) (ports.ResultadoOperacionObjeto, error)

type Configuracion struct {
	ConectorID              string
	Endpoint                string
	Region                  string
	BucketCuarentena        string
	BucketAdmitida          string
	AccessKeyID             string
	SecretAccessKey         string
	SessionToken            string
	RutaCA                  string
	RedesPermitidas         []netip.Prefix
	PathStyle               bool
	TamanoMaximo            int64
	DuracionCargaDirecta    time.Duration
	RetencionMinimaAdmitida time.Duration
	ClaveDerivacion         []byte
	Cifrado                 types.ServerSideEncryption
	ClaveKMS                string
	UsarBucketKeyKMS        bool
	PerfilFuerte            bool
	ProbarCapacidades       bool
	PermitirEliminacion     bool
	ModoRetencion           types.ObjectLockRetentionMode
}
```

Configuracion contiene exclusivamente parametros de composicion. Las
credenciales no se incluyen nunca en errores, capacidades ni evidencias.
Si no se proporcionan credenciales estaticas se usa la cadena segura del SDK
(identidad de carga, fichero protegido o proveedor externo).

```go
func ConfiguracionDesdeMapa(valores map[string]string) (Configuracion, error)

func (Configuracion) Format(estado fmt.State, _ rune)

func (Configuracion) GoString() string

func (Configuracion) LogValue() slog.Value

func (Configuracion) MarshalJSON() ([]byte, error)

func (Configuracion) MarshalText() ([]byte, error)

func (Configuracion) String() string
```

Configuracion contiene credenciales y material de derivacion. Sus
representaciones genericas se cierran para que un log, traza o respuesta de
diagnostico no pueda volcarlos accidentalmente.

```go
func (c Configuracion) Validar() error
```

## Paquete `internal/vec/adapters/documentos/docx`

> Package docx genera documentos Word Open XML sin macros ni recursos externos.

Package docx genera documentos Word Open XML sin macros ni recursos externos.

### Variables

```go
var ErrSalidaDOCXInvalida = errors.New("docx: salida generada invalida")
var ErrTextoInvalido = errors.New("docx: texto no válido para XML 1.0")
```

ErrTextoInvalido indica que el título o un párrafo contiene caracteres que
no se pueden representar en un documento XML 1.0 válido.

### Funciones

```go
func Renderizar(titulo string, parrafos []string) ([]byte, error)
```

Renderizar genera un DOCX editable con un título y una secuencia de
párrafos. El resultado contiene únicamente partes internas Open XML.

### Tipos

```go
type Renderizador struct{}
```

Renderizador adapta la generacion DOCX al puerto documental del nucleo.

```go
func (Renderizador) Formato() domain.FormatoDocumento

func (Renderizador) Renderizar(ctx context.Context, contenido domain.ContenidoDocumento) ([]byte, error)

func (Renderizador) ValidarSalida(ctx context.Context, contenido []byte) error
```

ValidarSalida rechaza partes inesperadas, macros, relaciones externas,
cifrado y expansiones ZIP desproporcionadas.

## Paquete `internal/vec/adapters/documentos/pdf`

> Package pdf genera la representacion PDF de trabajo mediante un adaptador reemplazable.

Package pdf genera la representacion PDF de trabajo mediante un adaptador
reemplazable. Firma, sello de tiempo, CSV y registro son pasos posteriores.

### Variables

```go
var ErrSalidaPDFInvalida = errors.New("pdf: salida generada invalida")
var ErrTextoInvalido = errors.New("pdf: texto invalido")
```

### Tipos

```go
type Renderizador struct{}
```

Renderizador implementa el puerto PDF sin introducir la libreria en el
dominio ni en los casos de uso.

```go
func (Renderizador) Formato() domain.FormatoDocumento

func (Renderizador) Renderizar(ctx context.Context, contenido domain.ContenidoDocumento) ([]byte, error)

func (Renderizador) ValidarSalida(ctx context.Context, contenido []byte) error
```

ValidarSalida aplica una comprobacion estructural independiente antes de
custodiar el artefacto. No sustituye la futura validacion PDF/A y PDF/UA.

## Paquete `internal/vec/adapters/documentos/seguridad`

> Package seguridad contiene adaptadores criptograficos y de infraestructura local.

Package seguridad contiene adaptadores criptograficos y de infraestructura
local. En produccion, la clave HMAC debe proceder de un gestor de secretos o KMS
y nunca de una imagen, repositorio o variable mostrada en logs.

### Variables

```go
var ErrConfiguracionSeguridadInvalida = errors.New("documentos: configuracion de seguridad invalida")
```

### Tipos

```go
type GeneradorID struct{}

func (GeneradorID) NuevoIDDocumento() (string, error)

type RelojSistema struct{}

func (RelojSistema) Ahora() time.Time

type SelladorHMAC struct {
	// Has unexported fields.
}

func NuevoSelladorHMAC(idClave string, clave []byte) (*SelladorHMAC, error)

func (s *SelladorHMAC) SellarDatos(ctx context.Context, datos []byte) (string, error)

func (s *SelladorHMAC) SellarSolicitudDocumento(ctx context.Context, datos []byte) (string, error)
```

SellarSolicitudDocumento permite usar otra instancia de SelladorHMAC, con
otra clave e identificador, para la huella idempotente. El metodo separado
evita que el ensamblado de produccion reutilice una clave por accidente.

```go
func (s *SelladorHMAC) SeudonimizarSujetoAlmacen(
	ctx context.Context,
	solicitud ports.SolicitudSeudonimizarSujetoAlmacen,
) (string, error)
```

SeudonimizarSujetoAlmacen permite dedicar una instancia y una clave
exclusivas a la seudonimizacion tecnica del almacen. El ensamblado no debe
reutilizar aqui el sellador de datos ni el de idempotencia.

## Paquete `internal/vec/adapters/httpapi`

> Adaptador HTTP del shell VEC: rutas publicas y privadas.

### Tipos

```go
type DemoIdentityResolver interface {
	ResolveDemoIdentity(context.Context, *http.Request) (domain.Principal, error)
}
```

DemoIdentityResolver es el unico origen admitido para el modo fake.
La implementacion productiva de composicion resuelve un Bearer opaco contra
un fichero local y no consume identidad, roles ni garantia desde cabeceras.

```go
type Handler struct {
	// Has unexported fields.
}

func NewHandler(service *application.Service) (*Handler, error)

func NewHandlerWithOptions(service *application.Service, options HandlerOptions) (*Handler, error)

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request)

type HandlerOptions struct {
	InternalOperations      *application.InternalOperations
	PersonalCatalogPath     string
	OSRMBaseURL             string
	OSRMScopeName           string
	OSRMScopeBounds         string
	OSRMAllowedCIDRs        []string
	AllowDemoIdentity       bool
	DemoIdentityResolver    DemoIdentityResolver
	TrustIdentityHeaders    bool
	TrustedProxyCIDRs       []string
	IdentitySubjectHeader   string
	IdentityRolesHeader     string
	IdentityMechanismHeader string
}
```

## Paquete `internal/vec/adapters/httpseguridad`

> Package httpseguridad define la frontera de seguridad HTTP entre las superficies publica, personal, interna y de administracion.

Package httpseguridad define la frontera de seguridad HTTP entre las superficies
publica, personal, interna y de administracion. No contiene un servidor ni
implementa protocolos de identidad: proporciona invariantes y contratos para que
el arranque componga listeners y adaptadores separados.

### Variables

```go
var (
	ErrCanalProxyNoAutenticado  = errors.New("canal del proxy no autenticado")
	ErrAsercionAusente          = errors.New("asercion de identidad ausente")
	ErrAsercionDemasiadoGrande  = errors.New("asercion de identidad demasiado grande")
	ErrAsercionNoValida         = errors.New("asercion de identidad no valida")
	ErrSesionNoValida           = errors.New("sesion de identidad no valida o revocada")
	ErrVerificadorAusente       = errors.New("verificador de aserciones ausente")
	ErrEvaluadorGarantiaAusente = errors.New("evaluador de garantia ausente")
	ErrRegistroSesionesAusente  = errors.New("registro de sesiones ausente")
	ErrCredencialNoSerializable = errors.New("la credencial de identidad no se puede reconstruir desde una serializacion")
	ErrInicializacionIdentidad  = errors.New("no se pudo inicializar el servicio de identidad")
)
var (
	ErrConfiguracionSuperficie = errors.New("configuracion de superficie no valida")
	ErrSuperficiesCompartidas  = errors.New("las superficies comparten un limite de seguridad")
)
var ErrRedNoAutorizada = errors.New("red no autorizada")
```

### Funciones

```go
func ValidarArquitecturaCompleta(configuraciones []ConfiguracionSuperficie) error
```

ValidarArquitecturaCompleta exige el despliegue declarado de las cuatro
superficies. ValidarConjuntoSuperficies sigue siendo util para validar un
proceso aislado, pero no sustituye esta comprobacion global de arranque.

```go
func ValidarConjuntoSuperficies(configuraciones []ConfiguracionSuperficie) error
```

ValidarConjuntoSuperficies impide compartir audiencia o cookie. Solo
las dos clases del portal exterior pueden compartir listener; interna y
administracion siempre utilizan entradas diferentes.

### Tipos

```go
type AltaSesionAtomica struct {
	AsercionID         string
	SesionID           string
	SujetoID           string
	CuentaID           string
	CuentaOrdinariaID  string
	CuentaPrivilegiada bool
	Superficie         Superficie
	EmitidaEn          time.Time
	ExpiraEn           time.Time
	PoliticaRef        string
	HuellaPolitica     string
}
```

AltaSesionAtomica es el dato que recibe el registro de reproduccion.
No contiene el mensaje protegido.

```go
type AsercionProxyIdentidad struct {
	ID                string
	Emisor            string
	Audiencia         string
	Superficie        Superficie
	SujetoID          string
	Cuenta            CuentaAcceso
	SesionID          string
	CanalVinculadoRef string
	EmitidaEn         time.Time
	NoAntesDe         time.Time
	ExpiraEn          time.Time
	MetodoPrimario    MetodoAutenticacion
	ACRVerificado     string
	Factores          []FactorAutenticacion
}
```

AsercionProxyIdentidad solo puede proceder de VerificadorAsercionProtegida.
ACRVerificado es la referencia autenticada del proveedor; nunca se acepta un
nivel de garantia numerico declarado libremente por la asercion.

```go
type CanalProxyAutenticado struct {
	// Has unexported fields.
}
```

CanalProxyAutenticado no se puede fabricar mediante literales desde otros
paquetes. Solo ServicioIdentidad.AutenticarCanalTLSMutuo puede emitirlo y lo
liga a una instancia concreta del servicio.

```go
func (c CanalProxyAutenticado) IdentidadPar() string

func (c CanalProxyAutenticado) ReferenciaVinculacion() string

func (c CanalProxyAutenticado) Superficie() Superficie

func (c CanalProxyAutenticado) Tipo() TipoCanalProxy

type ConfiguracionCookie struct {
	Nombre   string
	Ruta     string
	Dominio  string
	Segura   bool
	SoloHTTP bool
	SameSite ModoSameSite
}
```

ConfiguracionCookie describe exclusivamente la cookie de sesion de una
superficie. No se reutiliza entre portales.

```go
type ConfiguracionSuperficie struct {
	Superficie                          Superficie
	ZonaRed                             ZonaRed
	DireccionEscucha                    string
	Audiencia                           string
	Cookie                              ConfiguracionCookie
	EmisorIdentidad                     string
	RedesPermitidas                     []string
	HuellasProxyTLSPermitidas           []string
	IdentidadesSANProxyPermitidas       []string
	DuracionMaximaAsercion              time.Duration
	ToleranciaReloj                     time.Duration
	PermiteAnonimo                      bool
	MetodosAdmitidos                    []MetodoAutenticacion
	FactoresRequeridos                  []MetodoAutenticacion
	MinimoFactoresVerificados           int
	MinimoGruposCriptograficosDistintos int
	GarantiaMinima                      dominiovec.AuthAssurance
	RequiereCuentaPrivilegiada          bool
}
```

ConfiguracionSuperficie es la configuracion independiente de un listener.
RedesPermitidas siempre es explicita; incluso la red publica debe declarar
0.0.0.0/0 y/o ::/0 cuando quiera aceptar Internet.

```go
func (c ConfiguracionSuperficie) Validar() error
```

Validar aplica invariantes de una superficie individual. Ante cualquier
omision la configuracion se rechaza; no se completan valores de seguridad.

```go
type ConsultaSesionActiva struct {
	AsercionID         string
	SesionID           string
	SujetoID           string
	CuentaID           string
	CuentaOrdinariaID  string
	CuentaPrivilegiada bool
	Superficie         Superficie
	ExpiraEn           time.Time
}
```

ConsultaSesionActiva identifica de forma completa la sesion que se proyecta.

```go
type ContextoAuditoriaAutenticada struct {
	// Has unexported fields.
}
```

ContextoAuditoriaAutenticada conserva la identidad humana, la cuenta, la
sesion, la superficie y la politica verificadas. Deliberadamente no contiene
roles, permisos ni atributos de autorizacion.

```go
func (c ContextoAuditoriaAutenticada) AsercionID() string

func (c ContextoAuditoriaAutenticada) Audiencia() string

func (c ContextoAuditoriaAutenticada) CanalVinculadoRef() string

func (c ContextoAuditoriaAutenticada) CuentaID() string

func (c ContextoAuditoriaAutenticada) CuentaOrdinariaID() string

func (c ContextoAuditoriaAutenticada) CuentaPrivilegiada() bool

func (c ContextoAuditoriaAutenticada) Emisor() string

func (c ContextoAuditoriaAutenticada) EmitidaEn() time.Time

func (c ContextoAuditoriaAutenticada) ExpiraEn() time.Time

func (c ContextoAuditoriaAutenticada) Factores() []ResumenFactorAuditoria

func (c ContextoAuditoriaAutenticada) Garantia() dominiovec.AuthAssurance

func (c ContextoAuditoriaAutenticada) HuellaConfiguracion() string

func (c ContextoAuditoriaAutenticada) HuellaPolitica() string

func (c ContextoAuditoriaAutenticada) MetodoPrimario() MetodoAutenticacion

func (c ContextoAuditoriaAutenticada) NoAntesDe() time.Time

func (c ContextoAuditoriaAutenticada) PoliticaGarantiaRef() string

func (c ContextoAuditoriaAutenticada) SesionID() string

func (c ContextoAuditoriaAutenticada) SujetoID() string

func (c ContextoAuditoriaAutenticada) Superficie() Superficie

type CredencialProxy struct {
	// Has unexported fields.
}
```

CredencialProxy conserva la asercion en memoria privada y copia la entrada.
No existe un getter de los bytes; solo ServicioIdentidad puede entregarlos
al verificador mediante una copia adicional.

```go
func NuevaCredencialProxy(asercionProtegida []byte, canal CanalProxyAutenticado) (CredencialProxy, error)
```

NuevaCredencialProxy es la unica forma publica de aportar una credencial.

```go
func (CredencialProxy) Format(estado fmt.State, _ rune)

func (CredencialProxy) GoString() string

func (*CredencialProxy) GobDecode([]byte) error

func (CredencialProxy) GobEncode() ([]byte, error)

func (CredencialProxy) LogValue() slog.Value

func (CredencialProxy) MarshalBinary() ([]byte, error)

func (CredencialProxy) MarshalJSON() ([]byte, error)

func (CredencialProxy) MarshalText() ([]byte, error)

func (CredencialProxy) String() string

type CuentaAcceso struct {
	ID                string
	SujetoVinculadoID string
	CuentaOrdinariaID string
	Privilegiada      bool
}
```

CuentaAcceso separa la persona de su cuenta tecnica. Los identificadores de
cuenta se canonicalizan como identificadores ASCII sin distinguir caja.

```go
type EntradaEvaluacionGarantia struct {
	ACRVerificado  string
	Emisor         string
	Superficie     Superficie
	SujetoID       string
	CuentaID       string
	MetodoPrimario MetodoAutenticacion
	Factores       []FactorAutenticacion
}
```

EntradaEvaluacionGarantia contiene exclusivamente valores ya verificados y
copias defensivas de los factores.

```go
type EvaluadorGarantia interface {
	Evaluar(context.Context, EntradaEvaluacionGarantia) (ResultadoEvaluacionGarantia, error)
}
```

EvaluadorGarantia calcula la garantia desde ACR/AMR y grupos criptograficos
confiables. Es obligatorio y debe fallar cerrado ante combinaciones que no
reconozca.

```go
type FactorAutenticacion struct {
	Metodo                MetodoAutenticacion
	SujetoVinculadoID     string
	Principal             string
	CredencialRef         string
	EvidenciaRef          string
	GrupoCriptograficoRef string
	VerificadoEn          time.Time
}
```

FactorAutenticacion conserva evidencias verificadas por el adaptador de
identidad. GrupoCriptograficoRef identifica la credencial raiz real:
un ticket Kerberos obtenido por PKINIT y el certificado de la misma tarjeta
deben declarar el mismo grupo y, por tanto, no cuentan como dos factores
independientes.

```go
type IdentidadSesion struct {
	// Has unexported fields.
}
```

IdentidadSesion es un vale opaco e inmutable. Solo puede proyectarlo la
misma instancia y politica de ServicioIdentidad que lo emitio.

```go
func (IdentidadSesion) Format(estado fmt.State, _ rune)

func (IdentidadSesion) GoString() string

func (IdentidadSesion) String() string

type MetodoAutenticacion string
```

MetodoAutenticacion es un catalogo cerrado de mecanismos de identidad.
Los roles y permisos no proceden de la asercion ni de este catalogo tecnico.

```go
const (
	MetodoKerberos    MetodoAutenticacion = "kerberos"
	MetodoCertificado MetodoAutenticacion = "certificado"
	MetodoDNIe        MetodoAutenticacion = "dnie"
	MetodoClave       MetodoAutenticacion = "clave"
	MetodoSSO         MetodoAutenticacion = "sso"
)
func (m MetodoAutenticacion) Valido() bool

type ModoSameSite string
```

ModoSameSite evita que la configuracion utilice valores libres.

```go
const (
	SameSiteLaxo     ModoSameSite = "lax"
	SameSiteEstricto ModoSameSite = "strict"
)
type PoliticaRed struct {
	// Has unexported fields.
}
```

PoliticaRed es inmutable despues de su construccion.

```go
func NuevaPoliticaRed(configuracion ConfiguracionSuperficie) (PoliticaRed, error)
```

NuevaPoliticaRed construye una lista explicita. Una entrada mal formada no
se ignora: invalida toda la politica para conservar el cierre por defecto.

```go
func (p PoliticaRed) Autorizar(direccionPar netip.Addr) error
```

Autorizar acepta exclusivamente la direccion observada en la conexion de
transporte. La zona pertenece a la politica fijada al arrancar y nunca puede
ser declarada por una peticion o por una cabecera de proxy.

```go
type RegistroSesiones interface {
	ConsumirAsercionYRegistrar(context.Context, AltaSesionAtomica) error
	ComprobarSesionYCuentaActivas(context.Context, ConsultaSesionActiva) error
}
```

RegistroSesiones consume el identificador de asercion, comprueba que la
cuenta de acceso y, en administracion, su cuenta ordinaria esten activas, y
registra la sesion en una unica operacion atomica. Al proyectar debe volver
a comprobar ambas cuentas activas y sesion no revocada. Una implementacion
que separase esas operaciones abriria una carrera TOCTOU.

```go
type Reloj interface {
	Ahora() time.Time
}

type ResultadoEvaluacionGarantia struct {
	Garantia       dominiovec.AuthAssurance
	PoliticaRef    string
	HuellaPolitica string
}
```

ResultadoEvaluacionGarantia identifica tanto el nivel calculado como la
version exacta de la politica que lo calculo.

```go
type ResumenFactorAuditoria struct {
	Metodo                MetodoAutenticacion
	EvidenciaRef          string
	GrupoCriptograficoRef string
	VerificadoEn          time.Time
}
```

ResumenFactorAuditoria no contiene secretos ni concede autorizacion.

```go
type ServicioIdentidad struct {
	// Has unexported fields.
}
```

ServicioIdentidad es la unica autoridad que crea y proyecta identidades.

```go
func NuevoServicioIdentidad(
	configuracion ConfiguracionSuperficie,
	verificador VerificadorAsercionProtegida,
	evaluadorGarantia EvaluadorGarantia,
	registroSesiones RegistroSesiones,
	reloj Reloj,
) (*ServicioIdentidad, error)

func (s *ServicioIdentidad) AutenticarCanalTLSMutuo(estado tls.ConnectionState) (CanalProxyAutenticado, error)
```

AutenticarCanalTLSMutuo crea un canal solo desde un handshake mTLS terminado
y una cadena que la biblioteca TLS ya haya verificado.

```go
func (s *ServicioIdentidad) ProyectarPrincipal(
	ctx context.Context,
	identidad IdentidadSesion,
) (dominiovec.Principal, ContextoAuditoriaAutenticada, error)
```

ProyectarPrincipal es la unica conversion al principal del nucleo. Revalida
politica, vigencia, version de garantia, sesion y cuenta activa.

```go
func (s *ServicioIdentidad) Resolver(ctx context.Context, credencial CredencialProxy) (IdentidadSesion, error)
```

Resolver verifica canal, asercion, factores y garantia antes de consumir el
identificador de asercion y registrar la sesion atomicamente.

```go
type Superficie string
```

Superficie es un conjunto cerrado de clases de ruta. La zona anonima y
el area personal comparten el portal exterior, pero la primera no crea ni
consume sesion. Interna y administracion son fronteras de despliegue aparte.

```go
const (
	SuperficiePublicaAnonima             Superficie = "publica_anonima"
	SuperficieExternaPersonal            Superficie = "externa_personal"
	SuperficieInternaCorporativa         Superficie = "interna_corporativa"
	SuperficieAdministracionPrivilegiada Superficie = "administracion_privilegiada"
)
func (s Superficie) Valida() bool
```

Valida informa de si la superficie pertenece al conjunto cerrado.

```go
type TipoCanalProxy string
```

TipoCanalProxy es cerrado. Por ahora solo mTLS tiene un constructor seguro;
el socket Unix permanece cerrado hasta disponer de credenciales de par
verificadas por plataforma.

```go
const CanalProxyTLSMutuo TipoCanalProxy = "tls_mutuo"
type VerificadorAsercionProtegida interface {
	Verificar(context.Context, []byte) (AsercionProxyIdentidad, error)
}
```

VerificadorAsercionProtegida verifica firma, algoritmo, claves, revocacion y
formato antes de devolver datos tipados.

```go
type ZonaRed string
```

ZonaRed expresa la zona que debe alcanzar fisicamente un listener. No es una
etiqueta procedente de una peticion: la fija el despliegue del proceso.

```go
const (
	ZonaRedPublica        ZonaRed = "publica"
	ZonaRedInterna        ZonaRed = "interna"
	ZonaRedAdministracion ZonaRed = "administracion"
)
func (z ZonaRed) Valida() bool
```

Valida informa de si la zona pertenece al conjunto cerrado.

## Paquete `internal/vec/adapters/memory`

> Adaptadores en memoria del nucleo VEC para pruebas y arranque local.

### Tipos

```go
type AlmacenAutorizacionMemoria struct {
	// Has unexported fields.
}
```

AlmacenAutorizacionMemoria es un adaptador de desarrollo y pruebas. Mantiene
instantaneas inmutables y el registro de decisiones protegido para acceso
concurrente; no sustituye al registro duradero y sellado de produccion.

```go
func NuevoAlmacenAutorizacionMemoria() *AlmacenAutorizacionMemoria

func (a *AlmacenAutorizacionMemoria) ObtenerDecision(ctx context.Context, referencia string) (domain.DecisionAutorizacion, error)

func (a *AlmacenAutorizacionMemoria) ObtenerDenegacion(
	ctx context.Context,
	referencia string,
) (domain.DecisionAutorizacion, error)
```

ObtenerDenegacion existe solo en este adaptador de pruebas para comprobar la
separacion fisica entre trazas negativas y capacidades ejecutables.

```go
func (a *AlmacenAutorizacionMemoria) ObtenerInstantaneaAutorizacion(
	ctx context.Context,
	principalID, perfilActivoRef string,
) (domain.InstantaneaAutorizacion, error)

func (a *AlmacenAutorizacionMemoria) RegistrarDecisionSiInstantaneaVigente(
	ctx context.Context,
	decision domain.DecisionAutorizacion,
) error

func (a *AlmacenAutorizacionMemoria) RegistrarDenegacionAutorizacion(
	ctx context.Context,
	decision domain.DecisionAutorizacion,
) error

func (a *AlmacenAutorizacionMemoria) SembrarAsignacionPerfil(asignacion domain.AsignacionPerfil) error
```

SembrarAsignacionPerfil conserva todas las instantaneas y avanza el puntero
vigente de cada perfil. Una revocacion se expresa mediante otra version.

```go
func (a *AlmacenAutorizacionMemoria) SembrarControlVigenciaVersionRol(control domain.ControlVigenciaVersionRol) error
```

SembrarControlVigenciaVersionRol aplica una retirada global explicita
sobre una version exacta. Una version retirada distinta nunca afecta por
inferencia a las asignaciones que referencian otra version.

```go
func (a *AlmacenAutorizacionMemoria) SembrarPoliticaRestrictiva(politica domain.PoliticaRestrictiva) error
```

SembrarPoliticaRestrictiva retiene cada instantanea y solo avanza la version
actual con una posterior. Una version retirada deja de aplicarse.

```go
func (a *AlmacenAutorizacionMemoria) SembrarVersionRol(version domain.VersionRol) error
```

SembrarVersionRol carga una instantanea de configuracion. No modifica una
version ya existente, ni siquiera si el contenido coincide.

```go
type AlmacenContextoActor struct {
	// Has unexported fields.
}
```

AlmacenContextoActor es un adaptador inmutable para pruebas y composiciones
locales. Conserva copias privadas; el mutex protege tambien futuras lecturas
concurrentes frente a cambios accidentales en la implementacion.

```go
func NuevoAlmacenContextoActor(
	instantaneas ...domain.InstantaneaContextoActor,
) (*AlmacenContextoActor, error)

func (a *AlmacenContextoActor) BuscarInstantaneasContextoActor(
	ctx context.Context,
	solicitud domain.SolicitudContextoActor,
) ([]domain.InstantaneaContextoActor, error)

type AlmacenObjetosMemoria struct {
	// Has unexported fields.
}
```

AlmacenObjetosMemoria ejercita el contrato del puerto sin fingir aptitud
productiva. No cifra, no ofrece URL temporales y pierde todo al reiniciar.

```go
func NuevoAlmacenObjetosMemoria(conectorID string, tamanoMaximo int64, reloj ports.Reloj) (*AlmacenObjetosMemoria, error)

func (a *AlmacenObjetosMemoria) Abrir(ctx context.Context, solicitud ports.SolicitudAbrirObjeto) (ports.LecturaObjetoAlmacen, error)

func (a *AlmacenObjetosMemoria) AplicarRetencion(ctx context.Context, solicitud ports.SolicitudRetenerObjeto) (ports.ResultadoOperacionObjeto, error)

func (a *AlmacenObjetosMemoria) Capacidades(ctx context.Context) (ports.CapacidadesAlmacenObjetos, error)

func (a *AlmacenObjetosMemoria) Eliminar(ctx context.Context, solicitud ports.SolicitudEliminarObjeto) (ports.EvidenciaOperacionAlmacen, error)

func (a *AlmacenObjetosMemoria) Escribir(ctx context.Context, solicitud ports.SolicitudEscribirObjeto) (ports.ResultadoOperacionObjeto, error)

func (a *AlmacenObjetosMemoria) Inmovilizar(ctx context.Context, solicitud ports.SolicitudInmovilizarObjeto) (ports.ResultadoOperacionObjeto, error)

func (a *AlmacenObjetosMemoria) LevantarInmovilizacion(
	ctx context.Context,
	solicitud ports.SolicitudLevantarInmovilizacionObjeto,
) (ports.ResultadoOperacionObjeto, error)

func (a *AlmacenObjetosMemoria) Promover(ctx context.Context, solicitud ports.SolicitudPromoverObjeto) (ports.ResultadoOperacionObjeto, error)

type RepositorioCargasDocumentalesMemoria struct {
	// Has unexported fields.
}
```

RepositorioCargasDocumentalesMemoria verifica la semantica transaccional
del puerto, pero no es durable ni apto para produccion. En particular,
ConfirmarPreparacion conserva bajo el mismo bloqueo la transicion, el
manifiesto, la auditoria y el evento; nunca simula esa atomicidad mediante
escrituras sucesivas.

```go
func NuevoRepositorioCargasDocumentalesMemoria(
	reloj ports.Reloj,
) (*RepositorioCargasDocumentalesMemoria, error)

func (r *RepositorioCargasDocumentalesMemoria) AbandonarReserva(
	ctx context.Context,
	token ports.TokenReservaCargaDocumental,
) error

func (r *RepositorioCargasDocumentalesMemoria) ConfirmarPreparacion(
	ctx context.Context,
	solicitud ports.SolicitudConfirmarPreparacionCargaDocumental,
) error

func (r *RepositorioCargasDocumentalesMemoria) ConfirmarTransicion(
	ctx context.Context,
	confirmacion ports.ConfirmacionTransicionCargaDocumental,
) error

func (r *RepositorioCargasDocumentalesMemoria) Obtener(
	ctx context.Context,
	id string,
) (domain.CargaDocumental, error)

func (r *RepositorioCargasDocumentalesMemoria) ObtenerPreparacion(
	ctx context.Context,
	id string,
) (ports.PreparacionCargaDocumentalPersistida, error)

func (r *RepositorioCargasDocumentalesMemoria) Reservar(
	ctx context.Context,
	solicitud ports.SolicitudReservarCargaDocumental,
) (ports.ReservaCargaDocumental, error)

type Store struct {
	// Has unexported fields.
}

func NewStore() *Store

func (s *Store) AbandonarGeneracion(ctx context.Context, token ports.TokenReservaGeneracionDocumento) error

func (s *Store) AbandonarReservaCodigoCotejo(ctx context.Context, token ports.TokenReservaEmisionCodigoCotejo) error

func (s *Store) AppendAudit(ctx context.Context, entry domain.AuditEntry) (domain.AuditEntry, error)

func (s *Store) BuscarCodigoCotejoPorIndices(ctx context.Context, indices []string) (domain.CodigoCotejo, error)

func (s *Store) ConfirmarActivacionCodigoCotejo(ctx context.Context, huellaAnterior string, codigo domain.CodigoCotejo, traza domain.AuditEntry, evento domain.Event) error

func (s *Store) ConfirmarActualizacionBorradorCatalogo(
	ctx context.Context,
	huellaAnterior string,
	catalogo domain.CatalogoConfigurable,
	traza domain.AuditEntry,
	evento domain.Event,
	evidencia ports.EvidenciaUsoDecisionAutorizacion,
) error

func (s *Store) ConfirmarActualizacionBorradorFlujo(
	ctx context.Context,
	huellaAnterior string,
	definicion domain.DefinicionFlujo,
	traza domain.AuditEntry,
	evento domain.Event,
) error

func (s *Store) ConfirmarActualizacionBorradorPoliticaCotejo(
	ctx context.Context,
	huellaAnterior string,
	politica domain.PoliticaCotejo,
	traza domain.AuditEntry,
	evento domain.Event,
) error

func (s *Store) ConfirmarAltaBorradorCatalogo(
	ctx context.Context,
	catalogo domain.CatalogoConfigurable,
	traza domain.AuditEntry,
	evento domain.Event,
	evidencia ports.EvidenciaUsoDecisionAutorizacion,
) error

func (s *Store) ConfirmarAltaBorradorFlujo(
	ctx context.Context,
	definicion domain.DefinicionFlujo,
	traza domain.AuditEntry,
	evento domain.Event,
) error

func (s *Store) ConfirmarAltaBorradorPlantilla(ctx context.Context, plantilla domain.PlantillaDocumento, traza domain.AuditEntry, evento domain.Event) error

func (s *Store) ConfirmarAltaBorradorPoliticaCotejo(
	ctx context.Context,
	politica domain.PoliticaCotejo,
	traza domain.AuditEntry,
	evento domain.Event,
) error

func (s *Store) ConfirmarGeneracion(
	ctx context.Context,
	documento domain.DocumentoGenerado,
	traza domain.AuditEntry,
	evento domain.Event,
	evidencia ports.EvidenciaUsoDecisionAutorizacion,
) error

func (s *Store) ConfirmarGeneracionLogica(
	ctx context.Context,
	token ports.TokenReservaGeneracionDocumento,
	huellaSolicitudHMAC string,
	confirmadaEn time.Time,
	resultado domain.ResultadoGeneracionDocumento,
	traza domain.AuditEntry,
	evento domain.Event,
) error

func (s *Store) ConfirmarInicioInstanciaFlujo(
	ctx context.Context,
	instancia domain.InstanciaFlujo,
	traza domain.AuditEntry,
	evento domain.Event,
) error

func (s *Store) ConfirmarPublicacionCatalogo(
	ctx context.Context,
	huellaBorrador string,
	catalogo domain.CatalogoConfigurable,
	traza domain.AuditEntry,
	evento domain.Event,
	evidencia ports.EvidenciaUsoDecisionAutorizacion,
) error

func (s *Store) ConfirmarPublicacionFlujo(
	ctx context.Context,
	huellaBorrador string,
	definicion domain.DefinicionFlujo,
	traza domain.AuditEntry,
	evento domain.Event,
) error

func (s *Store) ConfirmarPublicacionPlantilla(ctx context.Context, huellaBorradorEsperada string, publicada domain.PlantillaDocumento, traza domain.AuditEntry, evento domain.Event) error

func (s *Store) ConfirmarPublicacionPoliticaCotejo(
	ctx context.Context,
	huellaAnterior string,
	politica domain.PoliticaCotejo,
	traza domain.AuditEntry,
	evento domain.Event,
) error

func (s *Store) ConfirmarReservaCodigoCotejo(
	ctx context.Context,
	token ports.TokenReservaEmisionCodigoCotejo,
	huellaSolicitudHMAC string,
	confirmadaEn time.Time,
	codigo domain.CodigoCotejo,
	traza domain.AuditEntry,
	evento domain.Event,
) error

func (s *Store) ConfirmarRetiradaCatalogo(
	ctx context.Context,
	huellaPublicada string,
	catalogo domain.CatalogoConfigurable,
	traza domain.AuditEntry,
	evento domain.Event,
	evidencia ports.EvidenciaUsoDecisionAutorizacion,
) error

func (s *Store) ConfirmarRetiradaCodigoCotejo(ctx context.Context, huellaAnterior string, codigo domain.CodigoCotejo, traza domain.AuditEntry, evento domain.Event) error

func (s *Store) ConfirmarRetiradaFlujo(
	ctx context.Context,
	huellaPublicada string,
	definicion domain.DefinicionFlujo,
	traza domain.AuditEntry,
	evento domain.Event,
) error

func (s *Store) ConfirmarRetiradaPoliticaCotejo(
	ctx context.Context,
	huellaAnterior string,
	politica domain.PoliticaCotejo,
	traza domain.AuditEntry,
	evento domain.Event,
) error

func (s *Store) ConfirmarSustitucionCodigoCotejo(ctx context.Context, huellaAnterior string, codigo domain.CodigoCotejo, traza domain.AuditEntry, evento domain.Event) error

func (s *Store) ConfirmarTransicionInstanciaFlujo(
	ctx context.Context,
	huellaAnterior string,
	actualizada domain.InstanciaFlujo,
	cambio domain.CambioEstadoFlujo,
	decision domain.DecisionReglaFlujo,
	aprobacion *domain.EvidenciaAprobacionFlujo,
	traza domain.AuditEntry,
	evento domain.Event,
) error

func (s *Store) GuardarContenido(ctx context.Context, solicitud ports.SolicitudGuardarContenido) (ports.ContenidoDocumentoGuardado, error)

func (s *Store) LeerContenido(ctx context.Context, solicitud ports.SolicitudLeerContenido) (ports.ContenidoDocumentoLeido, error)

func (s *Store) ListAudit(ctx context.Context, subjectRef string) ([]domain.AuditEntry, error)

func (s *Store) ListEvents(ctx context.Context, types []string) ([]domain.Event, error)

func (s *Store) ListModules(ctx context.Context) ([]domain.ModuleManifest, error)

func (s *Store) ListarDocumentosExpediente(ctx context.Context, expedienteRef string) ([]domain.DocumentoGenerado, error)

func (s *Store) ListarPlantillas(ctx context.Context, moduloID string) ([]domain.PlantillaDocumento, error)

func (s *Store) ListarRepresentacionesDocumento(ctx context.Context, referencia domain.ReferenciaDocumento) ([]domain.RepresentacionDocumento, error)

func (s *Store) ListarVersionesCatalogo(ctx context.Context, id string) ([]domain.CatalogoConfigurable, error)

func (s *Store) ListarVersionesDefinicionFlujo(ctx context.Context, id string) ([]domain.DefinicionFlujo, error)

func (s *Store) ListarVersionesPoliticaCotejo(ctx context.Context, id string) ([]domain.PoliticaCotejo, error)

func (s *Store) NuevoIDInstanciaFlujo() (string, error)

func (s *Store) ObtenerCatalogo(ctx context.Context, id string, version int) (domain.CatalogoConfigurable, error)

func (s *Store) ObtenerCodigoCotejo(ctx context.Context, id string) (domain.CodigoCotejo, error)

func (s *Store) ObtenerCodigoCotejoPorDocumento(ctx context.Context, referencia domain.ReferenciaDocumento) (domain.CodigoCotejo, error)

func (s *Store) ObtenerDecisionReglaFlujo(ctx context.Context, referencia string) (domain.DecisionReglaFlujo, error)

func (s *Store) ObtenerDefinicionFlujo(ctx context.Context, id string, version int) (domain.DefinicionFlujo, error)

func (s *Store) ObtenerDefinicionFlujoPorReferencia(ctx context.Context, referencia string) (domain.DefinicionFlujo, error)

func (s *Store) ObtenerDocumento(ctx context.Context, id string) (domain.DocumentoGenerado, error)

func (s *Store) ObtenerDocumentoLogico(ctx context.Context, referencia domain.ReferenciaDocumento) (domain.DocumentoLogico, error)

func (s *Store) ObtenerInstanciaFlujo(ctx context.Context, id string) (domain.InstanciaFlujo, error)

func (s *Store) ObtenerPlantilla(ctx context.Context, id string, version int) (domain.PlantillaDocumento, error)

func (s *Store) ObtenerPoliticaCotejo(ctx context.Context, id string, version int) (domain.PoliticaCotejo, error)

func (s *Store) PublishEvent(ctx context.Context, event domain.Event) error

func (s *Store) RegistrarConsultaCotejo(ctx context.Context, traza domain.AuditEntry, evento domain.Event) error

func (s *Store) RegistrarDecisionReglaFlujo(
	ctx context.Context,
	decision domain.DecisionReglaFlujo,
	traza domain.AuditEntry,
	evento domain.Event,
) error

func (s *Store) ReservarEmisionCodigoCotejo(
	ctx context.Context,
	solicitud ports.SolicitudReservarEmisionCodigoCotejo,
) (ports.ReservaEmisionCodigoCotejo, error)

func (s *Store) ReservarGeneracion(ctx context.Context, solicitud ports.SolicitudReservarGeneracionDocumento) (ports.ReservaGeneracionDocumento, error)

func (s *Store) SaveModule(ctx context.Context, manifest domain.ModuleManifest) error
```

## Paquete `internal/vec/adapters/postgres`

> Package postgres contiene adaptadores duraderos del nucleo para PostgreSQL.

Package postgres contiene adaptadores duraderos del nucleo para PostgreSQL.
No ejecuta migraciones ni abre conexiones: composicion y ciclo de vida del pool
pertenecen al borde de infraestructura.

### Tipos

```go
type AlmacenAutorizacion struct {
	// Has unexported fields.
}
```

AlmacenAutorizacion implementa la fuente coherente y el registro CAS de
decisiones. La cuenta del pool no recibe permisos sobre tablas: solo puede
ejecutar las funciones cerradas de su grupo tecnico NOLOGIN. La composicion
debe crear una instancia y un pool distintos para fuente y registro; ninguna
identidad de ejecucion hereda ambos grupos.

```go
func NuevoAlmacenAutorizacion(pool *pgxpool.Pool) (*AlmacenAutorizacion, error)
```

NuevoAlmacenAutorizacion no toma un DSN para evitar que el adaptador lo
conserve o lo incluya accidentalmente en errores. El llamador crea y prueba
el pool con su gestor de secretos y conserva su ciclo de vida.

```go
func (a *AlmacenAutorizacion) ObtenerInstantaneaAutorizacion(
	ctx context.Context,
	principalID, perfilActivoRef string,
) (domain.InstantaneaAutorizacion, error)

func (a *AlmacenAutorizacion) RegistrarDecisionSiInstantaneaVigente(
	ctx context.Context,
	decision domain.DecisionAutorizacion,
) error
```

## Paquete `internal/vec/adapters/postgres/confianzadocumental`

> Package confianzadocumental implementa el conector PostgreSQL de ejecucion documental atestada V4.

Package confianzadocumental implementa el conector PostgreSQL de ejecucion
documental atestada V4. El nucleo solo conoce el puerto neutral de ports; pgx,
SQL, el socket Unix y el material criptografico permanecen en este adaptador.
Deliberadamente no se exponen fabricas de Servicio, RaizPublicaFijada,
ConfiguracionConfianzaFijada ni de la autoridad opaca.

La composicion PostgreSQL carga la confianza durable desde una identidad emisora
aislada y entrega al proceso ejecutor una capacidad HMAC efimera por socket
Unix. Ningun llamador puede aportar raices, claves, verificadores o repositorios
arbitrarios. Su apertura productiva sigue condicionada a que Sistemas mantenga
segregadas ambas credenciales y el secreto operativo. Un conector Oracle futuro
implementara el mismo puerto sin modificar application.

### Variables

```go
var (
	// ErrAutoridadInternaEjecucionDocumentalV4Invalida oculta la causa concreta
	// y conserva la politica de denegacion por defecto ante cualquier ausencia,
	// discrepancia, caducidad, ambiguedad o manipulacion.
	ErrAutoridadInternaEjecucionDocumentalV4Invalida = errors.New("vec: autoridad interna de ejecucion documental v4 invalida")
	// ErrSerializacionAutoridadEjecucionDocumentalV4Prohibida impide convertir
	// una autoridad local en una credencial transportable o persistida.
	ErrSerializacionAutoridadEjecucionDocumentalV4Prohibida = errors.New("vec: serializacion de autoridad interna de ejecucion documental v4 prohibida")
)
var (
	ErrConfiguracionConfianzaDocumentalInvalida = errors.New("vec: configuracion de confianza documental invalida")
	ErrSolicitudVerificacionCOSESign1Invalida   = errors.New("vec: solicitud de verificacion COSE Sign1 invalida")
	ErrVerificacionCOSESign1Fallida             = errors.New("vec: verificacion COSE Sign1 documental fallida")
	ErrPruebaCOSESign1VerificadaInvalida        = errors.New("vec: prueba COSE Sign1 documental verificada invalida")
	ErrSerializacionAutoridadCOSESign1Prohibida = errors.New("vec: serializacion de autoridad COSE Sign1 documental prohibida")
)
var (
	ErrEvidenciaDurableAtestacionPDPV4Invalida                      = errors.New("vec: evidencia durable de atestacion PDP v4 invalida")
	ErrSerializacionGeneralEvidenciaDurableAtestacionPDPV4Prohibida = errors.New("vec: serializacion general de evidencia durable de atestacion PDP v4 prohibida")
)
```

### Funciones

```go
func NuevoManejadorHTTPEmisorCapacidadDocumentalV4(
	ctx context.Context,
	pool *pgxpool.Pool,
) (http.Handler, error)
```

NuevoManejadorHTTPEmisorCapacidadDocumentalV4 es el unico ensamblado del
proceso emisor. Debe ejecutarse en un binario aislado que solo disponga
de la credencial emisor_capacidad. El handler no contiene repositorio de
efectos ni puede invocar ejecutar_plan_atestado.

### Tipos

```go
type AlgoritmoCOSEDocumental string
```

AlgoritmoCOSEDocumental es la lista positiva inicial del nucleo. Una clave,
una cabecera o un adaptador no pueden ampliarla durante la ejecucion.

```go
const (
	AlgoritmoCOSEDocumentalEdDSA AlgoritmoCOSEDocumental = "EdDSA"
	AlgoritmoCOSEDocumentalES256 AlgoritmoCOSEDocumental = "ES256"
)
type AudienciaCOSEDocumental string
```

AudienciaCOSEDocumental es cerrada porque cada valor separa un protocolo de
seguridad distinto mediante external_aad. No se aceptan cadenas libres.

```go
const (
	AudienciaCOSEReciboComponenteDocumental AudienciaCOSEDocumental = "recibo_componente"
	AudienciaCOSETokenCercadoDocumental     AudienciaCOSEDocumental = "token_cercado"
	AudienciaCOSEEvidenciaDocumental        AudienciaCOSEDocumental = "evidencia"
	AudienciaCOSEReconciliacionDocumental   AudienciaCOSEDocumental = "reconciliacion"
	AudienciaCOSEEscrituraAlmacenDocumental AudienciaCOSEDocumental = "escritura_almacen"
	AudienciaCOSEAtestacionAutorizacionPDP  AudienciaCOSEDocumental = "atestacion_autorizacion_pdp"
)
type AutoridadInternaEjecucionDocumentalV4 struct {
	// Has unexported fields.
}
```

AutoridadInternaEjecucionDocumentalV4 es la autoridad opaca emitida dentro
del perimetro compilable de application. Liga exactamente DecisionRef,
plan, efecto y la solicitud estructural que comprobo actor, accion, recurso,
finalidad, ambitos, tiempo, campos y obligaciones.

Este tipo reduce la superficie capaz de emitir autoridad y conserva la
prueba criptografica PDP verificada por Servicio sobre el mensaje exacto
de domain.SerializarMensajeAtestacionAutorizacionV1. Esta autoridad local
no es la credencial de la ruta productiva: esa ruta exige la capacidad
efimera del emisor aislado y su consumo durable UNIQUE(DecisionRef)+efecto
en el mismo COMMIT PostgreSQL. Una evidencia estructural o
ports.AtestacionAutorizacionV1 nunca sustituyen esa composicion.

```go
func (a AutoridadInternaEjecucionDocumentalV4) Format(estado fmt.State, _ rune)

func (a AutoridadInternaEjecucionDocumentalV4) GoString() string

func (a AutoridadInternaEjecucionDocumentalV4) LogValue() slog.Value

func (AutoridadInternaEjecucionDocumentalV4) MarshalBinary() ([]byte, error)

func (AutoridadInternaEjecucionDocumentalV4) MarshalJSON() ([]byte, error)

func (AutoridadInternaEjecucionDocumentalV4) MarshalText() ([]byte, error)

func (a AutoridadInternaEjecucionDocumentalV4) PrepararAplicacionExactaConEvidenciaEn(
	decisionRef, huellaPlanSHA256, efectoRef string,
	instante time.Time,
) (
	ports.SolicitudAplicacionAutorizacionEjecucionDocumentalV4,
	EvidenciaDurableAtestacionAutorizacionPDPV4,
	error,
)
```

PrepararAplicacionExactaConEvidenciaEn entrega conjuntamente la solicitud no
autoritativa y la prueba durable que el repositorio debera confirmar en el
mismo COMMIT. Entregarla no consume la decision ni acredita persistencia.

```go
func (a AutoridadInternaEjecucionDocumentalV4) PrepararAplicacionExactaEn(
	decisionRef, huellaPlanSHA256, efectoRef string,
	instante time.Time,
) (ports.SolicitudAplicacionAutorizacionEjecucionDocumentalV4, error)
```

PrepararAplicacionExactaEn produce una solicitud no autoritativa ligada
a la terna que el caso de uso pretende aplicar. La ruta V4 productiva no
consume esta proyeccion local: revalida el COSE en el emisor aislado y exige
UNIQUE(DecisionRef)+efecto en el mismo COMMIT PostgreSQL. Prepararla no
consume nada.

```go
func (AutoridadInternaEjecucionDocumentalV4) String() string

func (*AutoridadInternaEjecucionDocumentalV4) UnmarshalBinary([]byte) error

func (*AutoridadInternaEjecucionDocumentalV4) UnmarshalJSON([]byte) error

func (*AutoridadInternaEjecucionDocumentalV4) UnmarshalText([]byte) error

func (a AutoridadInternaEjecucionDocumentalV4) ValidarEn(instante time.Time) error
```

ValidarEn revalida la autoridad, la evidencia estructural y su ventana.
El limite superior es exclusivo y un reloj no canonico siempre deniega.

```go
type ConfiguracionConfianzaFijada struct {
	// Has unexported fields.
}
```

ConfiguracionConfianzaFijada es una instantanea de revocacion con revision y
caducidad obligatorias. Al caducar, el servicio falla cerrado hasta recibir
una configuracion nueva durante un arranque controlado.

```go
func (c ConfiguracionConfianzaFijada) Format(estado fmt.State, _ rune)

func (c ConfiguracionConfianzaFijada) GoString() string

func (c ConfiguracionConfianzaFijada) LogValue() slog.Value

func (ConfiguracionConfianzaFijada) MarshalBinary() ([]byte, error)

func (ConfiguracionConfianzaFijada) MarshalJSON() ([]byte, error)

func (ConfiguracionConfianzaFijada) MarshalText() ([]byte, error)

func (ConfiguracionConfianzaFijada) String() string

func (*ConfiguracionConfianzaFijada) UnmarshalBinary([]byte) error

func (*ConfiguracionConfianzaFijada) UnmarshalJSON([]byte) error

func (*ConfiguracionConfianzaFijada) UnmarshalText([]byte) error

type EjecutorDocumentalPostgreSQLV4 struct {
	// Has unexported fields.
}
```

EjecutorDocumentalPostgreSQLV4 es la composicion del proceso web.
Solo contiene la credencial ejecutora y un cliente concreto de socket Unix.
No carga raices, no conoce el secreto HMAC y no puede emitir capacidades.

```go
func NuevoEjecutorDocumentalPostgreSQLV4(
	ctx context.Context,
	pool *pgxpool.Pool,
	rutaSocketEmisor string,
) (*EjecutorDocumentalPostgreSQLV4, error)
```

NuevoEjecutorDocumentalPostgreSQLV4 exige un pool concreto con la identidad
ejecutor_atestado y la ruta absoluta al socket del verificador aislado. No
acepta una interfaz emisora, una raiz, una clave ni un repositorio externo.

```go
func (e *EjecutorDocumentalPostgreSQLV4) EjecutarDocumentalAtestadoV4(
	ctx context.Context,
	vinculo ports.SolicitudVinculadaAutorizacionEjecucionDocumentalV4,
	cabecera domain.CabeceraAtestacionAutorizacionV1,
	sobre ports.SobreCriptograficoDocumentalCrudoV4,
) (ports.ResultadoConectorEjecucionDocumentalAtestadaV4, error)
```

EjecutarDocumentalAtestadoV4 implementa el puerto neutral del nucleo.
El resultado que cruza la frontera no contiene el paquete, la capacidad ni
material criptografico reutilizable.

```go
func (e *EjecutorDocumentalPostgreSQLV4) Format(estado fmt.State, _ rune)

func (e *EjecutorDocumentalPostgreSQLV4) GoString() string

func (e *EjecutorDocumentalPostgreSQLV4) LogValue() slog.Value

func (*EjecutorDocumentalPostgreSQLV4) MarshalJSON() ([]byte, error)

func (*EjecutorDocumentalPostgreSQLV4) String() string

type EstadoConfianzaClaveDocumental string

const (
	EstadoConfianzaClaveDocumentalActiva   EstadoConfianzaClaveDocumental = "activa"
	EstadoConfianzaClaveDocumentalRevocada EstadoConfianzaClaveDocumental = "revocada"
)
type EvidenciaDurableAtestacionAutorizacionPDPV4 struct {
	// Has unexported fields.
}
```

EvidenciaDurableAtestacionAutorizacionPDPV4 conserva payload y sobre para
que un COMMIT futuro pueda volver a verificar COSE de manera independiente.
No concede autoridad. Solo una AutoridadInterna ya atestada puede
entregarla.

La clave publica no se autocertifica dentro de esta evidencia: recuperarla
del mismo registro no demostraria confianza. Operacion debe conservar el
catalogo historico autoritativo de raices y configuraciones, con material
publico, revisiones y actos. Sin ese registro, la reverificacion futura y el
gate de produccion deben fallar cerrados.

VEC-AD-1 firma la huella del contexto de recurso, no plan y efecto como
campos separados. Por eso esta evidencia conserva tambien la preimagen
canonica completa del recurso. Su validacion exige que los dos atributos
exactos de plan y efecto recompongan la huella incluida en la decision
VEC-AD-1 parseada. Aun asi, ese enlace solo se vuelve autentico despues de
reverificar COSE contra el registro historico confiable.

HuellaSobreSHA256 solo identifica los bytes persistidos y detecta cambios.
Nunca es clave de replay o idempotencia: incluso una firma valida puede
repetirse con otros bytes. Esa exclusion se cierra en COMMIT mediante la
terna semantica DecisionRef, HuellaPlanSHA256 y EfectoRef.

```go
func (e EvidenciaDurableAtestacionAutorizacionPDPV4) Format(estado fmt.State, _ rune)

func (e EvidenciaDurableAtestacionAutorizacionPDPV4) GoString() string

func (e EvidenciaDurableAtestacionAutorizacionPDPV4) HuellaSHA256() (string, error)

func (e EvidenciaDurableAtestacionAutorizacionPDPV4) LogValue() slog.Value

func (EvidenciaDurableAtestacionAutorizacionPDPV4) MarshalBinary() ([]byte, error)

func (EvidenciaDurableAtestacionAutorizacionPDPV4) MarshalJSON() ([]byte, error)

func (EvidenciaDurableAtestacionAutorizacionPDPV4) MarshalText() ([]byte, error)

func (e EvidenciaDurableAtestacionAutorizacionPDPV4) Metadatos() (
	MetadatosEvidenciaDurableAtestacionPDPV4,
	error,
)

func (e EvidenciaDurableAtestacionAutorizacionPDPV4) PayloadVECAD1() ([]byte, error)

func (e EvidenciaDurableAtestacionAutorizacionPDPV4) PreimagenRecursoCanonica() (
	[]byte,
	error,
)

func (e EvidenciaDurableAtestacionAutorizacionPDPV4) SerializacionCanonicaParaPersistencia() (
	[]byte,
	error,
)
```

SerializacionCanonicaParaPersistencia es la unica salida binaria general
permitida. Es versionada, cerrada y devuelve una copia defensiva. No debe
enviarse a clientes ni registrarse en logs porque contiene la decision.

```go
func (e EvidenciaDurableAtestacionAutorizacionPDPV4) SobreCOSESign1() ([]byte, error)

func (EvidenciaDurableAtestacionAutorizacionPDPV4) String() string

func (*EvidenciaDurableAtestacionAutorizacionPDPV4) UnmarshalBinary([]byte) error

func (*EvidenciaDurableAtestacionAutorizacionPDPV4) UnmarshalJSON([]byte) error

func (*EvidenciaDurableAtestacionAutorizacionPDPV4) UnmarshalText([]byte) error

func (e EvidenciaDurableAtestacionAutorizacionPDPV4) Validar() error
```

Validar comprueba integridad y completitud historicas. No comprueba vigencia
actual ni revocaciones posteriores y, por tanto, nunca concede autoridad.

```go
type MetadatosEvidenciaDurableAtestacionPDPV4 struct {
	Esquema                        string
	Version                        uint16
	DecisionRef                    string
	HuellaPlanSHA256               string
	EfectoRef                      string
	HuellaSolicitudVinculadaSHA256 string
	FormatoVECADVersion            uint16
	Suite                          string
	ClaveID                        string
	AudienciaDespliegue            string
	AlgoritmoCOSE                  AlgoritmoCOSEDocumental
	AudienciaCOSE                  AudienciaCOSEDocumental
	EstadoConfianza                EstadoConfianzaClaveDocumental
	HuellaClaveSHA256              string
	HuellaPayloadSHA256            string
	HuellaSobreSHA256              string
	VerificadaEn                   time.Time
	RaizValidaDesde                time.Time
	RaizValidaHasta                time.Time
	RevisionConfianza              string
	HuellaConfiguracionSHA256      string
	ConfiguracionPublicadaEn       time.Time
	ConfiguracionExpiraEn          time.Time
	HuellaPreimagenRecursoSHA256   string
	HuellaContextoRecursoSHA256    string
	HuellaAmbitosRecursoSHA256     string
	HuellaEvidenciaDurableSHA256   string
}
```

MetadatosEvidenciaDurableAtestacionPDPV4 es la proyeccion deliberada que un
adaptador duradero puede mapear a columnas. Es evidencia descriptiva, no una
autorizacion ni un sustituto del sobre y payload que deben persistirse.

```go
func (m MetadatosEvidenciaDurableAtestacionPDPV4) Format(estado fmt.State, _ rune)

func (m MetadatosEvidenciaDurableAtestacionPDPV4) GoString() string

func (m MetadatosEvidenciaDurableAtestacionPDPV4) LogValue() slog.Value

func (MetadatosEvidenciaDurableAtestacionPDPV4) MarshalBinary() ([]byte, error)

func (MetadatosEvidenciaDurableAtestacionPDPV4) MarshalJSON() ([]byte, error)

func (MetadatosEvidenciaDurableAtestacionPDPV4) MarshalText() ([]byte, error)

func (MetadatosEvidenciaDurableAtestacionPDPV4) String() string

func (*MetadatosEvidenciaDurableAtestacionPDPV4) UnmarshalBinary([]byte) error

func (*MetadatosEvidenciaDurableAtestacionPDPV4) UnmarshalJSON([]byte) error

func (*MetadatosEvidenciaDurableAtestacionPDPV4) UnmarshalText([]byte) error

type PruebaCOSESign1DocumentalVerificada struct {
	// Has unexported fields.
}
```

PruebaCOSESign1DocumentalVerificada es autoridad local opaca. No existe
constructor publico y nunca conserva el payload ni el sobre firmado.

```go
func (p PruebaCOSESign1DocumentalVerificada) Algoritmo() (AlgoritmoCOSEDocumental, error)

func (p PruebaCOSESign1DocumentalVerificada) Audiencia() (AudienciaCOSEDocumental, error)

func (p PruebaCOSESign1DocumentalVerificada) ClaveID() ([]byte, error)

func (p PruebaCOSESign1DocumentalVerificada) Format(estado fmt.State, _ rune)

func (p PruebaCOSESign1DocumentalVerificada) GoString() string

func (p PruebaCOSESign1DocumentalVerificada) HuellaConfiguracionConfianzaSHA256() (string, error)

func (p PruebaCOSESign1DocumentalVerificada) HuellaPayloadSHA256() (string, error)

func (p PruebaCOSESign1DocumentalVerificada) HuellaSobreSHA256() (string, error)

func (p PruebaCOSESign1DocumentalVerificada) LogValue() slog.Value

func (PruebaCOSESign1DocumentalVerificada) MarshalBinary() ([]byte, error)

func (PruebaCOSESign1DocumentalVerificada) MarshalJSON() ([]byte, error)

func (PruebaCOSESign1DocumentalVerificada) MarshalText() ([]byte, error)

func (p PruebaCOSESign1DocumentalVerificada) RevisionConfianza() (string, error)

func (PruebaCOSESign1DocumentalVerificada) String() string

func (*PruebaCOSESign1DocumentalVerificada) UnmarshalBinary([]byte) error

func (*PruebaCOSESign1DocumentalVerificada) UnmarshalJSON([]byte) error

func (*PruebaCOSESign1DocumentalVerificada) UnmarshalText([]byte) error

func (p PruebaCOSESign1DocumentalVerificada) Validar() error

func (p PruebaCOSESign1DocumentalVerificada) ValidarPara(
	solicitud SolicitudVerificacionCOSESign1,
	sobre ports.SobreCriptograficoDocumentalCrudoV4,
) error

func (p PruebaCOSESign1DocumentalVerificada) VerificadaEn() (time.Time, error)

type RaizPublicaFijada struct {
	// Has unexported fields.
}
```

RaizPublicaFijada liga kid, algoritmo, clave, una unica audiencia, ventana
y estado de revocacion. Es configuracion local, nunca una afirmacion de un
adaptador.

```go
func (r RaizPublicaFijada) Format(estado fmt.State, _ rune)

func (r RaizPublicaFijada) GoString() string

func (r RaizPublicaFijada) LogValue() slog.Value

func (RaizPublicaFijada) MarshalBinary() ([]byte, error)

func (RaizPublicaFijada) MarshalJSON() ([]byte, error)

func (RaizPublicaFijada) MarshalText() ([]byte, error)

func (RaizPublicaFijada) String() string

func (*RaizPublicaFijada) UnmarshalBinary([]byte) error

func (*RaizPublicaFijada) UnmarshalJSON([]byte) error

func (*RaizPublicaFijada) UnmarshalText([]byte) error

type ResultadoEjecucionPlanDocumentalV4 struct {
	OrdenRef        string
	Estado          string
	AuditoriaRef    string
	EventoOutboxRef string
	RegistradaEn    time.Time
}
```

ResultadoEjecucionPlanDocumentalV4 solo expone referencias operativas. No
contiene payload, COSE, identidad personal ni material capaz de reejecutar.

```go
func (r ResultadoEjecucionPlanDocumentalV4) Format(estado fmt.State, _ rune)

func (r ResultadoEjecucionPlanDocumentalV4) GoString() string

func (r ResultadoEjecucionPlanDocumentalV4) LogValue() slog.Value

func (ResultadoEjecucionPlanDocumentalV4) String() string

type Servicio struct {
	// Has unexported fields.
}
```

Servicio contiene una instantanea defensiva de la lista positiva. El reloj y
la configuracion son internos; el solicitante no puede proponer una fecha.

```go
func (s *Servicio) EjecutarPlanDocumentalV4(
	ctx context.Context,
	autoridad AutoridadInternaEjecucionDocumentalV4,
) (ResultadoEjecucionPlanDocumentalV4, error)
```

EjecutarPlanDocumentalV4 es la unica salida de alto nivel hacia PostgreSQL.
Revalida el COSE completo en el instante del reloj interno y entrega una
prueba privada al repositorio fijado durante el ensamblado. El repositorio
debe registrar/confirmar atestacion, consumir DecisionRef, crear la orden
documental, auditoria y outbox en un unico COMMIT.

```go
func (s *Servicio) EmitirAutoridadInternaEjecucionDocumentalV4(
	ctx context.Context,
	vinculo ports.SolicitudVinculadaAutorizacionEjecucionDocumentalV4,
	cabecera domain.CabeceraAtestacionAutorizacionV1,
	sobre ports.SobreCriptograficoDocumentalCrudoV4,
) (AutoridadInternaEjecucionDocumentalV4, error)
```

EmitirAutoridadInternaEjecucionDocumentalV4 es la unica composicion publica
capaz de devolver la autoridad V4. La solicitud vinculada sigue sin ser
una credencial: el metodo reconstruye desde ella la decision completa,
produce exactamente VEC-AD-1 y exige una firma COSE de una raiz PDP fijada.

cabecera es seleccion de configuracion, no autoridad. Suite, kid y audiencia
se cotejan de nuevo contra la raiz usada realmente por COSE. El llamador no
aporta ningun instante. Una lectura interna preliminar permite reconstruir
el mensaje; la lectura realizada por VerificarCOSESign1 fija el instante de
emision y provoca una revalidacion completa posterior. Un retroceso entre
ambas lecturas se deniega.

```go
func (s *Servicio) Format(estado fmt.State, _ rune)

func (s *Servicio) GoString() string

func (s *Servicio) LogValue() slog.Value

func (*Servicio) MarshalBinary() ([]byte, error)

func (*Servicio) MarshalJSON() ([]byte, error)

func (*Servicio) MarshalText() ([]byte, error)

func (*Servicio) String() string

func (*Servicio) UnmarshalBinary([]byte) error

func (*Servicio) UnmarshalJSON([]byte) error

func (*Servicio) UnmarshalText([]byte) error

func (s *Servicio) VerificarCOSESign1(
	ctx context.Context,
	solicitud SolicitudVerificacionCOSESign1,
	sobre ports.SobreCriptograficoDocumentalCrudoV4,
) (PruebaCOSESign1DocumentalVerificada, error)
```

VerificarCOSESign1 interpreta y verifica localmente el sobre. Solo acepta
las cabeceras protegidas alg y kid, ninguna cabecera no protegida,
el payload exacto y la audiencia cerrada ligada mediante external_aad.

```go
type SolicitudVerificacionCOSESign1 struct {
	// Has unexported fields.
}
```

SolicitudVerificacionCOSESign1 fija los bytes exactos que deben estar
firmados y una audiencia cerrada. No contiene un instante aportado por el
llamador: Servicio lo obtiene de su reloj interno confiable.

```go
func NuevaSolicitudVerificacionCOSESign1(
	payloadEsperado []byte,
	audiencia AudienciaCOSEDocumental,
) (SolicitudVerificacionCOSESign1, error)

func (s SolicitudVerificacionCOSESign1) AADExterno() ([]byte, error)
```

AADExterno devuelve la vinculacion canonica que debe usar el firmante.

```go
func (s SolicitudVerificacionCOSESign1) Audiencia() (AudienciaCOSEDocumental, error)

func (s SolicitudVerificacionCOSESign1) Format(estado fmt.State, _ rune)

func (s SolicitudVerificacionCOSESign1) GoString() string

func (s SolicitudVerificacionCOSESign1) LogValue() slog.Value

func (SolicitudVerificacionCOSESign1) MarshalBinary() ([]byte, error)

func (SolicitudVerificacionCOSESign1) MarshalJSON() ([]byte, error)

func (SolicitudVerificacionCOSESign1) MarshalText() ([]byte, error)

func (s SolicitudVerificacionCOSESign1) PayloadEsperado() ([]byte, error)

func (SolicitudVerificacionCOSESign1) String() string

func (*SolicitudVerificacionCOSESign1) UnmarshalBinary([]byte) error

func (*SolicitudVerificacionCOSESign1) UnmarshalJSON([]byte) error

func (*SolicitudVerificacionCOSESign1) UnmarshalText([]byte) error

func (s SolicitudVerificacionCOSESign1) Validar() error
```

## Paquete `internal/vec/adapters/seguridad`

> Adaptadores criptograficos del nucleo: HMAC, AEAD y atestacion.

### Variables

```go
var (
	ErrConfiguracionCriptografiaCargaDirectaInvalida = errors.New("seguridad: configuracion criptografica de carga directa invalida")
	ErrMaterialCriptograficoCargaDirectaInvalido     = errors.New("seguridad: material criptografico de carga directa invalido")
	ErrCriptografiaCargaDirectaCerrada               = errors.New("seguridad: criptografia de carga directa cerrada")
	ErrEntropiaCargaDirectaNoDisponible              = errors.New("seguridad: entropia de carga directa no disponible")
)
var (
	ErrConfiguracionCriptografiaCotejoInvalida = errors.New("seguridad: configuracion criptografica de cotejo invalida")
	ErrMaterialCriptograficoCotejoInvalido     = errors.New("seguridad: material criptografico de cotejo invalido")
	ErrCriptografiaCotejoCerrada               = errors.New("seguridad: criptografia de cotejo cerrada")
	ErrEntropiaCotejoNoDisponible              = errors.New("seguridad: entropia de cotejo no disponible")
)
```

### Tipos

```go
type AdaptadorCriptograficoCargaDirecta struct {
	// Has unexported fields.
}
```

AdaptadorCriptograficoCargaDirecta no conserva recibos. Emite 256 bits
aleatorios, persiste exclusivamente su indice HMAC y delega el uso unico en
un repositorio durable con consumo condicional atomico.

```go
func NuevoAdaptadorCriptograficoCargaDirecta(
	configuracion ConfiguracionCriptografiaCargaDirecta,
	repositorio ports.RepositorioRecibosCargaDirecta,
	reloj ports.Reloj,
) (*AdaptadorCriptograficoCargaDirecta, error)

func (a *AdaptadorCriptograficoCargaDirecta) Cerrar()

func (a *AdaptadorCriptograficoCargaDirecta) ConsumirReciboCargaDirecta(
	ctx context.Context,
	solicitud ports.SolicitudConsumirReciboCargaDirecta,
) (ports.ComprobanteConsumoReciboCargaDirecta, error)

func (a *AdaptadorCriptograficoCargaDirecta) EmitirReciboCargaDirecta(
	ctx context.Context,
	solicitud ports.SolicitudEmitirReciboCargaDirecta,
) (ports.ReciboCargaDirecta, error)

func (a *AdaptadorCriptograficoCargaDirecta) Format(estado fmt.State, _ rune)

func (*AdaptadorCriptograficoCargaDirecta) GoString() string

func (a *AdaptadorCriptograficoCargaDirecta) LogValue() slog.Value

func (*AdaptadorCriptograficoCargaDirecta) MarshalJSON() ([]byte, error)

func (*AdaptadorCriptograficoCargaDirecta) MarshalText() ([]byte, error)

func (a *AdaptadorCriptograficoCargaDirecta) SeudonimizarSujetoAlmacen(
	ctx context.Context,
	solicitud ports.SolicitudSeudonimizarSujetoAlmacen,
) (string, error)

func (*AdaptadorCriptograficoCargaDirecta) String() string

func (a *AdaptadorCriptograficoCargaDirecta) VerificarAtestacionConsumoReciboCargaDirecta(
	ctx context.Context,
	contexto ports.ContextoOperacionAlmacen,
	sesionRef string,
	comprobante ports.ComprobanteConsumoReciboCargaDirecta,
) error
```

VerificarAtestacionConsumoReciboCargaDirecta es la segunda fase obligatoria
antes de que el nucleo construya una solicitud para el conector. Verifica
con una cuarta clave exclusiva el contexto completo de confirmacion,
incluidos AutorizacionRef, Accion, sesion y la fecha durable.

```go
type AdaptadorCriptograficoCotejo struct {
	// Has unexported fields.
}
```

AdaptadorCriptograficoCotejo implementa generacion, indexacion e
idempotencia sin custodiar el CSV. Para custodia se requiere otro adaptador
respaldado por KMS, HSM o un gestor de secretos.

```go
func NuevoAdaptadorCriptograficoCotejo(
	configuracion ConfiguracionCriptografiaCotejo,
) (*AdaptadorCriptograficoCotejo, error)

func (a *AdaptadorCriptograficoCotejo) Cerrar()
```

Cerrar realiza un borrado logico de las copias de claves conservadas.
Go no permite garantizar la eliminacion de copias internas del recolector o
de la primitiva HMAC, por lo que esto no sustituye la custodia en KMS/HSM.

```go
func (a *AdaptadorCriptograficoCotejo) Format(estado fmt.State, _ rune)

func (a *AdaptadorCriptograficoCotejo) GenerarValorCodigoCotejo(
	ctx context.Context,
) (ports.ValorCodigoCotejoGenerado, error)
```

GenerarValorCodigoCotejo obtiene 26 simbolos uniformes de un alfabeto de 32
elementos. Cada simbolo conserva cinco bits independientes: 130 bits reales
de entropia, por encima del minimo contractual de 128.

```go
func (*AdaptadorCriptograficoCotejo) GoString() string

func (a *AdaptadorCriptograficoCotejo) LogValue() slog.Value

func (*AdaptadorCriptograficoCotejo) MarshalJSON() ([]byte, error)

func (*AdaptadorCriptograficoCotejo) MarshalText() ([]byte, error)

func (a *AdaptadorCriptograficoCotejo) NuevoIDCodigoCotejo() (string, error)
```

NuevoIDCodigoCotejo crea una referencia opaca de 160 bits sin datos de
persona, expediente, documento ni secuencias predecibles.

```go
func (a *AdaptadorCriptograficoCotejo) SellarIndiceCodigoCotejo(
	ctx context.Context,
	secreto ports.SecretoCodigoCotejo,
) (string, error)

func (a *AdaptadorCriptograficoCotejo) SellarIndicesConsultaCodigoCotejo(
	ctx context.Context,
	secreto ports.SecretoCodigoCotejo,
) ([]string, error)

func (a *AdaptadorCriptograficoCotejo) SellarSolicitudCotejo(
	ctx context.Context,
	datos []byte,
) (string, error)

func (*AdaptadorCriptograficoCotejo) String() string

type ConfiguracionClaveHMACCargaDirecta struct {
	Identificador string `json:"identificador"`
	Material      []byte `json:"-"`
}
```

ConfiguracionClaveHMACCargaDirecta transporta material desde el ensamblado.
Sus representaciones nunca muestran la clave y el constructor conserva
una copia propia. En produccion el origen debe ser KMS, HSM o gestor de
secretos.

```go
func (c ConfiguracionClaveHMACCargaDirecta) Format(estado fmt.State, _ rune)

func (ConfiguracionClaveHMACCargaDirecta) GoString() string

func (c ConfiguracionClaveHMACCargaDirecta) LogValue() slog.Value

func (ConfiguracionClaveHMACCargaDirecta) MarshalJSON() ([]byte, error)

func (ConfiguracionClaveHMACCargaDirecta) MarshalText() ([]byte, error)

func (ConfiguracionClaveHMACCargaDirecta) String() string

type ConfiguracionClaveHMACCotejo struct {
	Identificador string `json:"identificador"`
	Material      []byte `json:"-"`
}
```

ConfiguracionClaveHMACCotejo transporta una clave desde el ensamblado.
Sus representaciones textuales y JSON siempre ocultan el material.
El constructor del adaptador vuelve a copiarlo antes de conservarlo.

```go
func (c ConfiguracionClaveHMACCotejo) Format(estado fmt.State, _ rune)

func (ConfiguracionClaveHMACCotejo) GoString() string

func (c ConfiguracionClaveHMACCotejo) LogValue() slog.Value

func (ConfiguracionClaveHMACCotejo) MarshalJSON() ([]byte, error)

func (ConfiguracionClaveHMACCotejo) MarshalText() ([]byte, error)

func (ConfiguracionClaveHMACCotejo) String() string

type ConfiguracionCriptografiaCargaDirecta struct {
	ClaveSeudonimizacion ConfiguracionClaveHMACCargaDirecta `json:"clave_seudonimizacion"`
	ClaveIndiceRecibo    ConfiguracionClaveHMACCargaDirecta `json:"clave_indice_recibo"`
	ClaveVinculoRecibo   ConfiguracionClaveHMACCargaDirecta `json:"clave_vinculo_recibo"`
	ClaveAtestacion      ConfiguracionClaveHMACCargaDirecta `json:"clave_atestacion"`
}
```

ConfiguracionCriptografiaCargaDirecta separa cuatro finalidades que no
pueden compartir identificador ni material: seudonimizar sujetos, indexar
recibos autenticar el vinculo inmutable y atestar el consumo durable.

```go
func (c ConfiguracionCriptografiaCargaDirecta) Format(estado fmt.State, _ rune)

func (ConfiguracionCriptografiaCargaDirecta) GoString() string

func (c ConfiguracionCriptografiaCargaDirecta) LogValue() slog.Value

func (ConfiguracionCriptografiaCargaDirecta) MarshalJSON() ([]byte, error)

func (ConfiguracionCriptografiaCargaDirecta) MarshalText() ([]byte, error)

func (ConfiguracionCriptografiaCargaDirecta) String() string

type ConfiguracionCriptografiaCotejo struct {
	VersionGenerador       string                         `json:"version_generador"`
	ClaveIndiceActual      ConfiguracionClaveHMACCotejo   `json:"clave_indice_actual"`
	ClavesIndiceHistoricas []ConfiguracionClaveHMACCotejo `json:"claves_indice_historicas,omitempty"`
	ClaveSolicitud         ConfiguracionClaveHMACCotejo   `json:"clave_solicitud"`
}
```

ConfiguracionCriptografiaCotejo separa expresamente la finalidad de indice
de la de idempotencia. La primera clave historica se consulta antes que la
segunda, pero solo ClaveIndiceActual se usa al emitir codigos nuevos.

```go
func (c ConfiguracionCriptografiaCotejo) Format(estado fmt.State, _ rune)

func (ConfiguracionCriptografiaCotejo) GoString() string

func (c ConfiguracionCriptografiaCotejo) LogValue() slog.Value

func (ConfiguracionCriptografiaCotejo) MarshalJSON() ([]byte, error)

func (ConfiguracionCriptografiaCotejo) MarshalText() ([]byte, error)

func (ConfiguracionCriptografiaCotejo) String() string

type GeneradorReferenciasCriptograficas struct{}
```

GeneradorReferenciasCriptograficas crea identificadores opacos sin incluir
DNI, nombre, correo ni ninguna clave de negocio.

```go
func (GeneradorReferenciasCriptograficas) NuevaReferenciaDecisionAutorizacion() (string, error)
```
