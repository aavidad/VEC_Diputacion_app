# Nucleo VEC: aplicacion y dobles de prueba

Parte del [Manual del programador](LEEME.md). Fichero generado con
`scripts/generar_manual_programador.py`; no editar a mano.

## Paquete `internal/vec/application`

> Casos de uso del shell VEC: modulos, auditoria, documentos, flujos y cotejo.

### Constantes

```go
const (
	AccionPrepararCargaDocumental  = "vec.documentos.carga.preparar"
	AccionConfirmarCargaDocumental = "vec.documentos.carga.confirmar"
	AccionAnalizarCargaDocumental  = "vec.documentos.carga.analizar"
	AccionPromoverCargaDocumental  = "vec.documentos.carga.promover"

	CampoCargaClasificacion       = "clasificacion"
	CampoCargaContenido           = "contenido"
	CampoCargaHuellaSHA256        = "huella_sha256"
	CampoCargaMIME                = "mime"
	CampoCargaTamano              = "tamano"
	CampoCargaContenidoCuarentena = "contenido_cuarentena"
	CampoCargaAnalisisSeguridad   = "analisis_seguridad"
	CampoCargaContenidoAdmitido   = "contenido_admitido"
	CampoCargaEstado              = "estado"
)
const (
	AccionCrearCatalogoConfigurable      = ports.AccionCrearCatalogoConfigurable
	AccionActualizarCatalogoConfigurable = ports.AccionActualizarCatalogoConfigurable
	AccionPublicarCatalogoConfigurable   = ports.AccionPublicarCatalogoConfigurable
	AccionRetirarCatalogoConfigurable    = ports.AccionRetirarCatalogoConfigurable
)
const (
	AccionCrearBorradorPoliticaCotejo      = "vec.documentos.cotejo.politicas.crear"
	AccionActualizarBorradorPoliticaCotejo = "vec.documentos.cotejo.politicas.actualizar"
	AccionPublicarPoliticaCotejo           = "vec.documentos.cotejo.politicas.publicar"
	AccionRetirarPoliticaCotejo            = "vec.documentos.cotejo.politicas.retirar"
	AccionReservarCodigoCotejo             = "vec.documentos.cotejo.codigos.reservar"
	AccionActivarCodigoCotejo              = "vec.documentos.cotejo.codigos.activar"
	AccionRetirarCodigoCotejo              = "vec.documentos.cotejo.codigos.retirar"
	AccionSustituirCodigoCotejo            = "vec.documentos.cotejo.codigos.sustituir"
	AccionConsultaPublicaCotejo            = "vec.documentos.cotejo.consultar_publico"
	AccionConsultaProtegidaCotejo          = "vec.documentos.cotejo.consultar_protegido"
	AccionRevisionInternaCotejo            = "vec.documentos.cotejo.revisar"
)
const (
	AccionCrearBorradorPlantillaDocumento = "vec.documentos.plantillas.crear"
	AccionPublicarPlantillaDocumento      = "vec.documentos.plantillas.publicar"
)
const (
	AccionDocumentoLogicoGenerado = "vec.documento.logico.generado"
)
const (
	AccionCrearDefinicionFlujo      = "vec.flujos.definicion.crear"
	AccionActualizarDefinicionFlujo = "vec.flujos.definicion.actualizar"
	AccionPublicarDefinicionFlujo   = "vec.flujos.definicion.publicar"
	AccionRetirarDefinicionFlujo    = "vec.flujos.definicion.retirar"
)
const FinalidadConsultaInternaFuenteAutoridad = ports.FinalidadConsultaInternaFuenteAutoridad
```

### Variables

```go
var (
	ErrDependenciaConsultaInternaFuenteAutoridadRequerida = errors.New("vec: dependencia de consulta interna de fuente de autoridad requerida")
	ErrOrdenConsultaInternaFuenteAutoridadInvalida        = errors.New("vec: orden de consulta interna de fuente de autoridad invalida")
	ErrResultadoConsultaInternaFuenteAutoridadInvalido    = errors.New("vec: resultado de consulta interna de fuente de autoridad invalido")
	ErrSerializacionOrdenConsultaInternaAutoridad         = errors.New("vec: serializacion de orden interna de consulta de autoridad prohibida")
	ErrSerializacionResultadoConsultaInternaAutoridad     = errors.New("vec: serializacion de resultado interno de consulta de autoridad prohibida")
)
var (
	ErrDependenciaCargaDocumentalRequerida = errors.New("vec: dependencia de carga documental requerida")
	ErrOrdenCargaDocumentalInvalida        = errors.New("vec: orden de carga documental invalida")
	ErrCargaDocumentalNoCorresponde        = errors.New("vec: carga documental no corresponde al contexto")
	ErrCargaDocumentalNoPreparada          = errors.New("vec: carga documental no preparada")
	ErrResultadoCargaDocumentalInvalido    = errors.New("vec: resultado de carga documental invalido")
	ErrCargaDocumentalYaProcesada          = errors.New("vec: carga documental ya procesada")
)
var (
	ErrDependenciaCatalogosRequerida = errors.New("vec: dependencia de catalogos requerida")
	ErrOrdenCatalogoInvalida         = errors.New("vec: orden de catalogo invalida")
	ErrSerializacionOrdenCatalogo    = errors.New("vec: serializacion de orden interna de catalogo prohibida")
)
var (
	ErrDependenciaCotejoRequerida = errors.New("vec: dependencia de cotejo requerida")
	ErrOrdenCotejoInvalida        = errors.New("vec: orden de cotejo invalida")
	ErrResultadoCotejoInvalido    = errors.New("vec: resultado de cotejo invalido")
	ErrCotejoNoDisponible         = errors.New("vec: cotejo no disponible")
)
var (
	ErrDependenciaDocumentalRequerida = errors.New("vec: dependencia documental requerida")
	ErrRenderizadorNoDisponible       = errors.New("vec: renderizador documental no disponible")
	ErrOrdenDocumentalInvalida        = errors.New("vec: orden documental invalida")
	ErrDocumentoDemasiadoGrande       = errors.New("vec: documento generado demasiado grande")
	ErrConfirmacionContenidoInvalida  = errors.New("vec: confirmacion de contenido invalida")
)
var (
	// ErrPoliticaUsoDecisionAutorizacionInvalida indica que el caso de uso no
	// ha declarado de forma cerrada como consumira la decision del PDP.
	ErrPoliticaUsoDecisionAutorizacionInvalida = errors.New("vec: politica de uso de decision de autorizacion invalida")
	// ErrSerializacionPoliticaUsoAutorizacionProhibida impide convertir una
	// configuracion interna en un DTO de entrada o salida.
	ErrSerializacionPoliticaUsoAutorizacionProhibida = errors.New("vec: serializacion de politica de uso de autorizacion prohibida")
)
var (
	ErrDependenciaEjecucionFlujosRequerida = errors.New("vec: dependencia de ejecucion de flujos requerida")
	ErrOrdenEjecucionFlujoInvalida         = errors.New("vec: orden de ejecucion de flujo invalida")
)
var (
	ErrDependenciaGobiernoFlujosRequerida = errors.New("vec: dependencia de gobierno de flujos requerida")
	ErrOrdenGobiernoFlujoInvalida         = errors.New("vec: orden de gobierno de flujo invalida")
	ErrReferenciaCatalogoFlujoInvalida    = errors.New("vec: referencia de catalogo de flujo invalida")
)
var (
	// Un unico error externo evita revelar si falta un perfil, existen dos
	// entradas contradictorias, fue retirado o no hay conector homologado.
	ErrResolucionFormatoDocumentalCerrada = errors.New("vec: resolucion de formato documental cerrada")
	ErrMetadatoInstitucionalNoIncorporado = errors.New("vec: metadato institucional no incorporado")
)
var (
	ErrDependenciaAltaCobroRequerida   = errors.New("vec: dependencia de alta de cobro requerida")
	ErrSolicitudAltaCobroInvalida      = errors.New("vec: solicitud de alta de cobro invalida")
	ErrLiquidacionCobroNoConfiable     = errors.New("vec: liquidacion de cobro no confiable")
	ErrLiquidacionCobroNoExigible      = errors.New("vec: liquidacion de cobro no exigible")
	ErrResultadoAltaCobroNoConfiable   = errors.New("vec: resultado de alta de cobro no confiable")
	ErrPersistenciaAltaCobroIncompleta = errors.New("vec: persistencia atomica de alta de cobro incompleta")
	ErrSerializacionAltaCobroProhibida = errors.New("vec: serializacion directa de alta de cobro prohibida")
)
var ErrEjecucionDocumentalAtestadaV4NoDisponible = errors.New(
	"vec: ejecucion documental atestada v4 no disponible",
)
var ErrInternalOperationsMismatch = errors.New("vec internal operations do not belong to service")
var ErrServiceDependencyRequired = errors.New("vec service dependency required")
```

### Funciones

```go
func CalcularHuellaLiquidacionCobroAutoritativa(
	datos DatosLiquidacionCobroAutoritativa,
) (string, error)
```

CalcularHuellaLiquidacionCobroAutoritativa ofrece al adaptador oficial
el mismo esquema canonico que valida el caso de uso. La huella acredita
integridad de la instantanea, no autoridad por si sola.

```go
func NewServiceWithInternalOperations(
	modules ports.ModuleRegistryStore,
	audit ports.AuditStore,
	events ports.EventStore,
) (*Service, *InternalOperations, error)
```

NewServiceWithInternalOperations es el constructor reservado a la raiz de
composicion. Mantiene separada la superficie externa de las operaciones de
arranque e infraestructura y liga estas ultimas a una unica instancia.

### Tipos

```go
type AltaOrdenCobroCompletada struct {
	// Has unexported fields.
}
```

AltaOrdenCobroCompletada solo expone la proyeccion minima del titular,
nunca el agregado, su historial, la decision ni las huellas HMAC internas.

```go
func (a AltaOrdenCobroCompletada) Datos() (DatosAltaOrdenCobroCompletada, error)

func (AltaOrdenCobroCompletada) MarshalJSON() ([]byte, error)

func (AltaOrdenCobroCompletada) String() string

func (*AltaOrdenCobroCompletada) UnmarshalJSON([]byte) error

type AuditCommand struct {
	Principal            domain.Principal
	ActorProfile         string
	RepresentedSubjectID string
	AuthorizationRef     string
	Purpose              string
	Action               string
	ModuleID             string
	SubjectRef           string
	ObjectVersion        int
	ExpedienteRef        string
	DocumentRef          string
	RuleRef              string
	Reason               string
	Result               string
	BeforeHash           string
	AfterHash            string
	CorrelationRef       string
	Metadata             map[string]string
}

type AuditQuery struct {
	// Has unexported fields.
}
```

AuditQuery fija el principal y una unica referencia exacta antes de tocar el
almacen. La lista positiva de permisos contiene solo vec.audit.read.

```go
func NewAuditQuery(principal domain.Principal, subjectRef string) (AuditQuery, error)

type AuditReceipt struct {
	// Has unexported fields.
}
```

AuditReceipt es la evidencia opaca de una escritura ya confirmada. Un evento
solo puede publicarse si fue declarado al crear la capacidad y queda ligado
exactamente al actor, modulo, sujeto e identificador de esa traza.

```go
func (r AuditReceipt) Entry() domain.AuditEntry

type AuthorizedAuditCommand struct {
	// Has unexported fields.
}
```

AuthorizedAuditCommand es una capacidad positiva, concreta e inmutable.
Solo se crea cuando el principal contiene exactamente el permiso exigido. No
es un PDP ni crea permisos: consume la evidencia autenticada que ya resolvio
la frontera y falla cerrado ante ausencia, comodines o valores no canonicos.

```go
func NewAuthorizedAuditCommand(
	command AuditCommand,
	requiredPermission string,
	expectedEventType string,
) (AuthorizedAuditCommand, error)

type ConfiguracionPoliticaCotejo struct {
	Nombre                   string
	Descripcion              string
	Modulos                  []string
	TiposDocumentales        []string
	Clasificaciones          []string
	ClaseAcceso              domain.ClaseAccesoCotejo
	CamposPublicos           []domain.CampoPublicoCotejo
	PermiteDescargaDocumento bool
	RequiereTitularidad      bool
	RolesTitularidad         []string
	RequiereFirma            bool
	RequiereSelloTiempo      bool
	RequiereRegistro         bool
	GarantiaMinima           domain.AuthAssurance
	DiasPlazoActivacion      int
	DiasDisponibilidad       int
	FuenteRef                string
}

type ConfiguracionServicioAutorizacion struct {
	VigenciaDecision time.Duration
}

type ConsultaLiquidacionCobro struct {
	LiquidacionRef string
}
```

ConsultaLiquidacionCobro contiene exclusivamente la referencia opaca
declarada por el iniciador. La fuente debe buscar todas las coincidencias;
el servicio deniega cero o mas de una y nunca elige la primera.

```go
type CredencialesGobiernoCatalogo struct {
	ContextoActor             domain.ContextoActor
	VinculoAutenticacionActor domain.VinculoAutenticacionActorV1
}
```

CredencialesGobiernoCatalogo agrupa las capacidades opacas resueltas por el
middleware interno. No contiene roles ni permisos declarados por el cliente.

```go
type DatosAltaOrdenCobroCompletada struct {
	Vista    domain.VistaTitularOrdenCobro
	Repetida bool
}

type DatosEvidenciaResolucionFormatoDocumental struct {
	Consulta     ports.ConsultaFormatoDocumental
	Descriptor   ports.DescriptorFormatoDocumental
	HuellaSHA256 string
}
```

DatosEvidenciaResolucionFormatoDocumental es una instantanea tipada y
autoconsistente para un futuro puerto duradero. Sus campos son valores
inmutables o copias; modificarlos no altera la evidencia original.

```go
type DatosLiquidacionCobroAutoritativa struct {
	LiquidacionRef    string
	Revision          uint64
	HuellaSHA256      string
	ExpedienteRef     string
	SolicitudRef      string
	Tarifa            domain.ReferenciaTarifaCobro
	SujetoRef         string
	RepresentacionRef string
	Importe           domain.DineroCobro
	Concepto          string
	Finalidad         string
	Estado            EstadoLiquidacionCobro
	ExigibleDesde     time.Time
	ExigibleHasta     time.Time
}
```

DatosLiquidacionCobroAutoritativa es la instantanea completa que debe
obtener un adaptador de liquidaciones desde su registro oficial. No es un
DTO de entrada: importe, tarifa, sujeto, estado y vigencia nunca proceden de
la peticion que inicia el caso de uso.

```go
type EjecutorDocumentalAtestadoV4 struct {
	// Has unexported fields.
}
```

EjecutorDocumentalAtestadoV4 es el caso de uso neutral del nucleo.
Solo conoce el puerto de ejecucion atestada y no conoce controladores,
motores de datos, sockets, claves ni repositorios concretos.

```go
func NuevoEjecutorDocumentalAtestadoV4(
	conector ports.ConectorEjecucionDocumentalAtestadaV4,
) (*EjecutorDocumentalAtestadoV4, error)
```

NuevoEjecutorDocumentalAtestadoV4 recibe un puerto fijado en la raiz de
composicion. Sustituir el motor exige otro conector homologado, no modificar
ni recompilar este caso de uso.

```go
func (e *EjecutorDocumentalAtestadoV4) Ejecutar(
	ctx context.Context,
	vinculo ports.SolicitudVinculadaAutorizacionEjecucionDocumentalV4,
	cabecera domain.CabeceraAtestacionAutorizacionV1,
	sobre ports.SobreCriptograficoDocumentalCrudoV4,
) (ResultadoEjecucionDocumentalAtestadaV4, error)
```

Ejecutar es la unica operacion de la fachada. Recibe el vinculo estructural
y el COSE del PDP, pero nunca un DTO persistible, un verificador, una raiz,
un repositorio ni un instante elegido por el llamador.

```go
func (e *EjecutorDocumentalAtestadoV4) Format(estado fmt.State, _ rune)

func (e *EjecutorDocumentalAtestadoV4) GoString() string

func (e *EjecutorDocumentalAtestadoV4) LogValue() slog.Value

func (*EjecutorDocumentalAtestadoV4) MarshalJSON() ([]byte, error)

func (*EjecutorDocumentalAtestadoV4) String() string

type EstadoConsultaCotejo string

const (
	EstadoConsultaCotejoNoDisponible           EstadoConsultaCotejo = "no_disponible"
	EstadoConsultaCotejoDisponible             EstadoConsultaCotejo = "disponible"
	EstadoConsultaCotejoRequiereIdentificacion EstadoConsultaCotejo = "requiere_identificacion"
)
type EstadoLiquidacionCobro string
```

EstadoLiquidacionCobro es una lista cerrada. Solo Exigible permite crear
una orden; los demas estados existen para que una fuente pueda expresar una
negativa explicita sin convertir una omision o un valor nuevo en permiso.

```go
const (
	EstadoLiquidacionCobroEmitida    EstadoLiquidacionCobro = "emitida"
	EstadoLiquidacionCobroExigible   EstadoLiquidacionCobro = "exigible"
	EstadoLiquidacionCobroSuspendida EstadoLiquidacionCobro = "suspendida"
	EstadoLiquidacionCobroAnulada    EstadoLiquidacionCobro = "anulada"
	EstadoLiquidacionCobroPagada     EstadoLiquidacionCobro = "pagada"
	EstadoLiquidacionCobroCaducada   EstadoLiquidacionCobro = "caducada"
)
type EvidenciaResolucionFormatoDocumental struct {
	// Has unexported fields.
}
```

EvidenciaResolucionFormatoDocumental compromete la consulta y la unica
respuesta aceptada, incluido el digest del artefacto instalado. Es inmutable
en memoria; un puerto duradero de auditoria se incorporara al integrar el
puente, sin reconstruir esta evidencia desde logs o mapas.

```go
func RestaurarEvidenciaResolucionFormatoDocumental(
	datos DatosEvidenciaResolucionFormatoDocumental,
) (EvidenciaResolucionFormatoDocumental, error)

func (e EvidenciaResolucionFormatoDocumental) Datos() (
	DatosEvidenciaResolucionFormatoDocumental,
	error,
)

func (e EvidenciaResolucionFormatoDocumental) HuellaSHA256() (string, error)

func (e EvidenciaResolucionFormatoDocumental) Validar() error

type ExigidorEvidenciaUsoDecisionAutorizacion interface {
	ExigirEvidencia(
		ctx context.Context,
		actor domain.ContextoActor,
		vinculo domain.VinculoAutenticacionActorV1,
		recurso domain.RecursoAutorizable,
		correlacionRef string,
		motivo string,
		politica PoliticaUsoDecisionAutorizacion,
	) (ports.EvidenciaUsoDecisionAutorizacion, error)
}
```

ExigidorEvidenciaUsoDecisionAutorizacion es el contrato minimo que deben
inyectar los modulos. Depender de esta interfaz evita acoplar sus casos de
uso al servicio concreto y permite dobles contractuales sin otro PEP.

```go
type ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2 interface {
	ExigirEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
		ctx context.Context,
		actor domain.ContextoActor,
		vinculo domain.VinculoAutenticacionActorV1,
		recurso domain.RecursoAutorizable,
		correlacion domain.ReferenciaCorrelacionAutorizacionV2,
		motivo domain.ReferenciaEntradaCatalogo,
		politica PoliticaUsoDecisionAutorizacion,
	) (ports.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2, error)
}
```

ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2 es un contrato
distinto y no sustituible por V1. Los flujos nuevos que necesitan demostrar
solicitud y motivo exactos deben depender expresamente de esta interfaz.

```go
type FachadaUsoDecisionAutorizacion struct {
	// Has unexported fields.
}
```

FachadaUsoDecisionAutorizacion ofrece a los modulos la unica operacion
general necesaria: exigir una decision vinculada y convertirla en evidencia
opaca. No expone domain.DecisionAutorizacion al llamador.

```go
func NuevaFachadaUsoDecisionAutorizacion(
	autorizador ports.Autorizador,
	reloj ports.Reloj,
) (*FachadaUsoDecisionAutorizacion, error)
```

NuevaFachadaUsoDecisionAutorizacion fija el PEP y el reloj confiable en la
composicion. Las dependencias no se seleccionan desde una peticion.

```go
func (f *FachadaUsoDecisionAutorizacion) ExigirEvidencia(
	ctx context.Context,
	actor domain.ContextoActor,
	vinculo domain.VinculoAutenticacionActorV1,
	recurso domain.RecursoAutorizable,
	correlacionRef string,
	motivo string,
	politica PoliticaUsoDecisionAutorizacion,
) (ports.EvidenciaUsoDecisionAutorizacion, error)
```

ExigirEvidencia ejecuta el PEP comun, comprueba la politica cerrada del caso
de uso y devuelve solo la capacidad que el adaptador duradero puede consumir
junto al efecto. Actor, vinculo y recurso deben proceder de fronteras
confiables ya resueltas; esta operacion nunca completa ni normaliza
entradas.

```go
type FachadaUsoDecisionAutorizacionSolicitudLigadaV2 struct {
	// Has unexported fields.
}
```

FachadaUsoDecisionAutorizacionSolicitudLigadaV2 es la composicion explicita
para modulos nuevos. No implementa la interfaz V1 ni acepta su autorizador.

```go
func NuevaFachadaUsoDecisionAutorizacionSolicitudLigadaV2(
	autorizador ports.AutorizadorSolicitudLigadaV2,
	reloj ports.Reloj,
) (*FachadaUsoDecisionAutorizacionSolicitudLigadaV2, error)

func (f *FachadaUsoDecisionAutorizacionSolicitudLigadaV2) ExigirEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
	ctx context.Context,
	actor domain.ContextoActor,
	vinculo domain.VinculoAutenticacionActorV1,
	recurso domain.RecursoAutorizable,
	correlacion domain.ReferenciaCorrelacionAutorizacionV2,
	motivo domain.ReferenciaEntradaCatalogo,
	politica PoliticaUsoDecisionAutorizacion,
) (ports.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2, error)

type FormatoDocumentalResuelto struct {
	// Has unexported fields.
}

func (r FormatoDocumentalResuelto) Descriptor() (ports.DescriptorFormatoDocumental, error)

func (r FormatoDocumentalResuelto) Evidencia() (EvidenciaResolucionFormatoDocumental, error)

func (r FormatoDocumentalResuelto) Renderizador() (ports.RenderizadorDocumentalPorPerfil, error)

type FuenteLiquidacionesCobro interface {
	BuscarLiquidacionesCobro(context.Context, ConsultaLiquidacionCobro) ([]LiquidacionCobroAutoritativa, error)
}
```

FuenteLiquidacionesCobro es una frontera de confianza pendiente de adaptador
productivo. Debe leer liquidacion y version exacta de tarifa en una unica
instantanea coherente del registro oficial. Una cache no revalidada,
un DTO o una tabla editable por el usuario no satisfacen este contrato. Esta
lectura permite decidir y construir la propuesta, pero nunca se considera
atomica con su persistencia: ConfirmarCreacion debe volver a comparar el
control oficial dentro de su propia transaccion.

```go
type InternalOperations struct {
	// Has unexported fields.
}
```

InternalOperations agrupa exclusivamente operaciones de composicion y de
infraestructura. No se obtiene desde Service: una frontera externa que solo
reciba *Service no puede registrar modulos, escribir trazas/eventos ni leer
auditoria. La raiz de composicion debe entregarlo de forma deliberada.

```go
func (o *InternalOperations) Audit(ctx context.Context, query AuditQuery) ([]domain.AuditEntry, error)

func (o *InternalOperations) Matches(service *Service) bool
```

Matches impide mezclar por error un Service con operaciones internas de otra
composicion (y, por tanto, con otros almacenes).

```go
func (o *InternalOperations) PublishEvent(ctx context.Context, receipt AuditReceipt, event domain.Event) error

func (o *InternalOperations) RecordAudit(
	ctx context.Context,
	authorized AuthorizedAuditCommand,
) (AuditReceipt, error)

func (o *InternalOperations) RegisterModule(ctx context.Context, manifest domain.ModuleManifest) error

type LiquidacionCobroAutoritativa struct {
	// Has unexported fields.
}
```

LiquidacionCobroAutoritativa es opaca y no serializable para impedir que un
adaptador HTTP la reconstruya accidentalmente. Su constructor comprueba que
la huella publicada liga exactamente todos los datos funcionales.

```go
func NuevaLiquidacionCobroAutoritativa(
	datos DatosLiquidacionCobroAutoritativa,
) (LiquidacionCobroAutoritativa, error)

func (l LiquidacionCobroAutoritativa) Datos() (DatosLiquidacionCobroAutoritativa, error)

func (l LiquidacionCobroAutoritativa) Format(estado fmt.State, _ rune)

func (l LiquidacionCobroAutoritativa) GoString() string

func (LiquidacionCobroAutoritativa) MarshalJSON() ([]byte, error)

func (LiquidacionCobroAutoritativa) MarshalText() ([]byte, error)

func (LiquidacionCobroAutoritativa) String() string

func (*LiquidacionCobroAutoritativa) UnmarshalJSON([]byte) error

type OpcionesServicioCargaDocumental struct {
	VigenciaInstrucciones       time.Duration
	TamanoMaximo                int64
	ConectorAlmacenPermitido    string
	ConectorAnalizadorPermitido string
	VersionAnalizadorPermitida  int
}

type OpcionesServicioDocumental struct {
	OrganoENI   string
	LimiteBytes int64
}

type OrdenActivarCodigoCotejo struct {
	Principal        domain.Principal
	PerfilActivo     string
	RepresentadoRef  string
	Finalidad        string
	CodigoID         string
	RepresentacionID string
	ActivacionRef    string
	Motivo           string
	CorrelacionRef   string
}

type OrdenActualizarBorradorCatalogo struct {
	Credenciales     CredencialesGobiernoCatalogo
	Finalidad        string
	ID               string
	Version          int
	RevisionEsperada int
	Nombre           string
	Descripcion      string
	FuenteRef        string
	Entradas         []domain.EntradaCatalogoConfigurable
	Motivo           string
	CorrelacionRef   string
}

func (OrdenActualizarBorradorCatalogo) MarshalJSON() ([]byte, error)

func (OrdenActualizarBorradorCatalogo) String() string

func (*OrdenActualizarBorradorCatalogo) UnmarshalJSON([]byte) error

type OrdenActualizarBorradorFlujo struct {
	Principal        domain.Principal
	PerfilActivo     string
	Finalidad        string
	ID               string
	Version          int
	RevisionEsperada int
	Configuracion    domain.ConfiguracionBorradorFlujo
	Motivo           string
	CorrelacionRef   string
}

type OrdenActualizarBorradorPoliticaCotejo struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	ID             string
	Version        int
	Configuracion  ConfiguracionPoliticaCotejo
	Motivo         string
	CorrelacionRef string
}

type OrdenAnalizarCargaDocumental struct {
	Principal      domain.Principal
	PerfilActivo   string
	Recurso        domain.RecursoAutorizable
	CargaID        string
	Finalidad      string
	Motivo         string
	CorrelacionRef string
}

type OrdenAplicarTransicionFlujo struct {
	Principal       domain.Principal
	PerfilActivo    string
	Finalidad       string
	InstanciaID     string
	TransicionClave string
	AprobacionRef   string
	Motivo          string
	CorrelacionRef  string
}

type OrdenConfirmarCargaDocumental struct {
	Principal      domain.Principal
	PerfilActivo   string
	Recurso        domain.RecursoAutorizable
	CargaID        string
	SesionRef      string
	Recibo         ports.ReciboCargaDirecta
	Finalidad      string
	Motivo         string
	CorrelacionRef string
}

type OrdenConsultaInternaExactaFuenteAutoridad struct {
	ContextoActor             domain.ContextoActor
	VinculoAutenticacionActor domain.VinculoAutenticacionActorV1
	Selector                  ports.SelectorVersionFuenteAutoridad
	MotivoCatalogo            domain.ReferenciaEntradaCatalogo
	Correlacion               domain.ReferenciaCorrelacionAutorizacionV2
	// Has unexported fields.
}
```

OrdenConsultaInternaExactaFuenteAutoridad contiene capacidades resueltas por
la frontera interna. No admite roles, permisos, finalidad ni atributos de
recurso declarados por el cliente.

```go
func (o OrdenConsultaInternaExactaFuenteAutoridad) Format(estado fmt.State, _ rune)

func (o OrdenConsultaInternaExactaFuenteAutoridad) GoString() string

func (*OrdenConsultaInternaExactaFuenteAutoridad) GobDecode([]byte) error

func (OrdenConsultaInternaExactaFuenteAutoridad) GobEncode() ([]byte, error)

func (o OrdenConsultaInternaExactaFuenteAutoridad) LogValue() slog.Value

func (OrdenConsultaInternaExactaFuenteAutoridad) MarshalBinary() ([]byte, error)

func (OrdenConsultaInternaExactaFuenteAutoridad) MarshalCBOR() ([]byte, error)

func (OrdenConsultaInternaExactaFuenteAutoridad) MarshalJSON() ([]byte, error)

func (OrdenConsultaInternaExactaFuenteAutoridad) MarshalText() ([]byte, error)

func (OrdenConsultaInternaExactaFuenteAutoridad) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (OrdenConsultaInternaExactaFuenteAutoridad) MarshalYAML() (any, error)

func (OrdenConsultaInternaExactaFuenteAutoridad) String() string

func (*OrdenConsultaInternaExactaFuenteAutoridad) UnmarshalBinary([]byte) error

func (*OrdenConsultaInternaExactaFuenteAutoridad) UnmarshalCBOR([]byte) error

func (*OrdenConsultaInternaExactaFuenteAutoridad) UnmarshalJSON([]byte) error

func (*OrdenConsultaInternaExactaFuenteAutoridad) UnmarshalText([]byte) error

func (*OrdenConsultaInternaExactaFuenteAutoridad) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (*OrdenConsultaInternaExactaFuenteAutoridad) UnmarshalYAML(func(any) error) error

type OrdenConsultaProtegidaCotejo struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	Secreto        ports.SecretoCodigoCotejo
	Motivo         string
	CorrelacionRef string
}

type OrdenConsultaPublicaCotejo struct {
	Secreto          ports.SecretoCodigoCotejo
	CorrelacionRef   string
	OrigenTecnicoRef string
}

type OrdenCrearBorradorCatalogo struct {
	Credenciales   CredencialesGobiernoCatalogo
	Finalidad      string
	ID             string
	Version        int
	ModuloID       string
	Nombre         string
	Descripcion    string
	FuenteRef      string
	Entradas       []domain.EntradaCatalogoConfigurable
	Motivo         string
	CorrelacionRef string
}

func (OrdenCrearBorradorCatalogo) MarshalJSON() ([]byte, error)

func (OrdenCrearBorradorCatalogo) String() string

func (*OrdenCrearBorradorCatalogo) UnmarshalJSON([]byte) error

type OrdenCrearBorradorFlujo struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	ID             string
	Version        int
	ModuloID       string
	TipoEntidad    string
	Configuracion  domain.ConfiguracionBorradorFlujo
	Motivo         string
	CorrelacionRef string
}

type OrdenCrearBorradorPlantilla struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	ID             string
	Version        int
	ModuloID       string
	TipoDocumental string
	Nombre         string
	Titulo         string
	Parrafos       []string
	Campos         []domain.CampoPlantillaDocumento
	Formatos       []domain.FormatoDocumento
	PermisoGenerar string
	GarantiaMinima domain.AuthAssurance
	Motivo         string
	CorrelacionRef string
}

type OrdenCrearBorradorPoliticaCotejo struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	ID             string
	Version        int
	Configuracion  ConfiguracionPoliticaCotejo
	Motivo         string
	CorrelacionRef string
}

type OrdenGenerarDocumento struct {
	Principal        domain.Principal
	PerfilActivo     string
	RepresentadoRef  string
	Finalidad        string
	Clasificacion    string
	PlantillaID      string
	PlantillaVersion int
	Formato          domain.FormatoDocumento
	ExpedienteRef    string
	Datos            map[string]string
	Motivo           string
	CorrelacionRef   string
}
```

OrdenGenerarDocumento contiene contexto administrativo, no solo datos de
presentacion. Ninguna generacion queda huerfana de expediente o decision de
autorizacion.

```go
type OrdenGenerarDocumentoLogico struct {
	Principal         domain.Principal
	PerfilActivo      string
	RepresentadoRef   string
	Finalidad         string
	Clasificacion     string
	ClaveIdempotencia string
	PlantillaID       string
	PlantillaVersion  int
	Relaciones        []domain.RelacionDocumento
	Representaciones  []domain.SolicitudRepresentacionDocumento
	Datos             map[string]string
	Motivo            string
	CorrelacionRef    string
}
```

OrdenGenerarDocumentoLogico produce una unica version administrativa con
una o varias representaciones tecnicas. ClaveIdempotencia debe ser opaca,
aleatoria y estable en todos los reintentos de la misma operacion.

```go
type OrdenIniciarInstanciaFlujo struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	DefinicionID   string
	Version        int
	EntidadRef     string
	Motivo         string
	CorrelacionRef string
}

type OrdenPrepararCargaDocumental struct {
	Principal                domain.Principal
	PerfilActivo             string
	Recurso                  domain.RecursoAutorizable
	OperacionRef             string
	Finalidad                string
	Clasificacion            string
	MIME                     string
	Tamano                   int64
	HuellaSHA256             string
	ClaveIdempotenciaCliente string
	Motivo                   string
	CorrelacionRef           string
}

type OrdenPublicarCatalogo struct {
	Credenciales   CredencialesGobiernoCatalogo
	Finalidad      string
	ID             string
	Version        int
	AprobacionRef  string
	Motivo         string
	CorrelacionRef string
}

func (OrdenPublicarCatalogo) MarshalJSON() ([]byte, error)

func (OrdenPublicarCatalogo) String() string

func (*OrdenPublicarCatalogo) UnmarshalJSON([]byte) error

type OrdenPublicarFlujo struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	ID             string
	Version        int
	AprobacionRef  string
	Motivo         string
	CorrelacionRef string
}

type OrdenPublicarPlantilla struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	PlantillaID    string
	Version        int
	AprobacionRef  string
	Motivo         string
	CorrelacionRef string
}

type OrdenPublicarPoliticaCotejo struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	ID             string
	Version        int
	AprobacionRef  string
	Motivo         string
	CorrelacionRef string
}

type OrdenReservarCodigoCotejo struct {
	Principal         domain.Principal
	PerfilActivo      string
	RepresentadoRef   string
	Finalidad         string
	ClaveIdempotencia string
	Documento         domain.ReferenciaDocumento
	PoliticaID        string
	PoliticaVersion   int
	Motivo            string
	CorrelacionRef    string
}

type OrdenRetirarCatalogo struct {
	Credenciales   CredencialesGobiernoCatalogo
	Finalidad      string
	ID             string
	Version        int
	AprobacionRef  string
	Motivo         string
	CorrelacionRef string
}

func (OrdenRetirarCatalogo) MarshalJSON() ([]byte, error)

func (OrdenRetirarCatalogo) String() string

func (*OrdenRetirarCatalogo) UnmarshalJSON([]byte) error

type OrdenRetirarCodigoCotejo struct {
	Principal       domain.Principal
	PerfilActivo    string
	RepresentadoRef string
	Finalidad       string
	CodigoID        string
	RetiradaRef     string
	Motivo          string
	CorrelacionRef  string
}

type OrdenRetirarFlujo struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	ID             string
	Version        int
	AprobacionRef  string
	Motivo         string
	CorrelacionRef string
}

type OrdenRetirarPoliticaCotejo struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	ID             string
	Version        int
	AprobacionRef  string
	Motivo         string
	CorrelacionRef string
}

type OrdenSustituirCodigoCotejo struct {
	Principal       domain.Principal
	PerfilActivo    string
	RepresentadoRef string
	Finalidad       string
	CodigoID        string
	SustitutoID     string
	SustitucionRef  string
	Motivo          string
	CorrelacionRef  string
}

type PerfilProteccionUsoAutorizacion uint8
```

PerfilProteccionUsoAutorizacion selecciona las comprobaciones locales que
el caso de uso aplica ademas de la decision exacta del PDP. El valor cero no
tiene significado para que una ampliacion no herede un perfil por omision.

```go
const (
	PerfilProteccionUsoAutorizacionNoDeclarado PerfilProteccionUsoAutorizacion = iota
	// PerfilProteccionUsoAutorizacionOrdinario admite una superficie personal
	// externa; la garantia efectiva sigue teniendo que cumplir la exigida por
	// la decision del PDP.
	PerfilProteccionUsoAutorizacionOrdinario
	// PerfilProteccionUsoAutorizacionInternoAlto exige una sesion de garantia
	// alta en una superficie corporativa o de administracion privilegiada. La
	// propia decision debe conservar tambien garantia minima alta.
	PerfilProteccionUsoAutorizacionInternoAlto
)
type PoliticaUsoDecisionAutorizacion struct {
	// Has unexported fields.
}
```

PoliticaUsoDecisionAutorizacion es una configuracion nominal cerrada e
inmutable para consumidores externos al paquete. No es una capacidad
criptografica: se construye deliberadamente con su fabrica, pero no
representa datos aportados por HTTP ni admite codecs de transporte.

```go
func NuevaPoliticaUsoDecisionAutorizacion(
	accion string,
	moduloID string,
	tipoRecurso string,
	finalidad string,
	camposEsperados []string,
	perfil PerfilProteccionUsoAutorizacion,
) (PoliticaUsoDecisionAutorizacion, error)
```

NuevaPoliticaUsoDecisionAutorizacion fija la operacion nominal exacta que el
caso de uso puede solicitar y el conjunto cerrado de campos que interpreta.
Una lista vacia declara una operacion atomica sin restricciones por campo;
no significa "cualquier campo".

```go
func (p PoliticaUsoDecisionAutorizacion) Format(estado fmt.State, _ rune)

func (p PoliticaUsoDecisionAutorizacion) GoString() string

func (*PoliticaUsoDecisionAutorizacion) GobDecode([]byte) error

func (PoliticaUsoDecisionAutorizacion) GobEncode() ([]byte, error)

func (p PoliticaUsoDecisionAutorizacion) LogValue() slog.Value

func (PoliticaUsoDecisionAutorizacion) MarshalBinary() ([]byte, error)

func (PoliticaUsoDecisionAutorizacion) MarshalJSON() ([]byte, error)

func (PoliticaUsoDecisionAutorizacion) MarshalText() ([]byte, error)

func (PoliticaUsoDecisionAutorizacion) MarshalXML(*xml.Encoder, xml.StartElement) error

func (PoliticaUsoDecisionAutorizacion) String() string

func (*PoliticaUsoDecisionAutorizacion) UnmarshalBinary([]byte) error

func (*PoliticaUsoDecisionAutorizacion) UnmarshalJSON([]byte) error

func (*PoliticaUsoDecisionAutorizacion) UnmarshalText([]byte) error

func (*PoliticaUsoDecisionAutorizacion) UnmarshalXML(*xml.Decoder, xml.StartElement) error

type ResultadoConsultaInternaExactaFuenteAutoridad struct {
	Encontrada   bool
	Fuente       domain.FuenteAutoridadVersionada
	EstadoExacto ports.ReferenciaEstadoFuenteAutoridad
	Recibo       ports.ReciboConsultaInternaFuenteAutoridad
	// Has unexported fields.
}

func (r ResultadoConsultaInternaExactaFuenteAutoridad) Clonar() (
	ResultadoConsultaInternaExactaFuenteAutoridad,
	error,
)

func (r ResultadoConsultaInternaExactaFuenteAutoridad) Format(estado fmt.State, _ rune)

func (r ResultadoConsultaInternaExactaFuenteAutoridad) GoString() string

func (*ResultadoConsultaInternaExactaFuenteAutoridad) GobDecode([]byte) error

func (ResultadoConsultaInternaExactaFuenteAutoridad) GobEncode() ([]byte, error)

func (r ResultadoConsultaInternaExactaFuenteAutoridad) LogValue() slog.Value

func (ResultadoConsultaInternaExactaFuenteAutoridad) MarshalBinary() ([]byte, error)

func (ResultadoConsultaInternaExactaFuenteAutoridad) MarshalCBOR() ([]byte, error)

func (ResultadoConsultaInternaExactaFuenteAutoridad) MarshalJSON() ([]byte, error)

func (ResultadoConsultaInternaExactaFuenteAutoridad) MarshalText() ([]byte, error)

func (ResultadoConsultaInternaExactaFuenteAutoridad) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (ResultadoConsultaInternaExactaFuenteAutoridad) MarshalYAML() (any, error)

func (ResultadoConsultaInternaExactaFuenteAutoridad) String() string

func (*ResultadoConsultaInternaExactaFuenteAutoridad) UnmarshalBinary([]byte) error

func (*ResultadoConsultaInternaExactaFuenteAutoridad) UnmarshalCBOR([]byte) error

func (*ResultadoConsultaInternaExactaFuenteAutoridad) UnmarshalJSON([]byte) error

func (*ResultadoConsultaInternaExactaFuenteAutoridad) UnmarshalText([]byte) error

func (*ResultadoConsultaInternaExactaFuenteAutoridad) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (*ResultadoConsultaInternaExactaFuenteAutoridad) UnmarshalYAML(func(any) error) error

type ResultadoConsultaProtegidaCotejo struct {
	Estado             EstadoConsultaCotejo        `json:"estado,omitempty"`
	CodigoRef          string                      `json:"codigo_ref,omitempty"`
	Documento          *domain.ReferenciaDocumento `json:"documento,omitempty"`
	ModuloID           string                      `json:"modulo_id,omitempty"`
	TipoDocumental     string                      `json:"tipo_documental,omitempty"`
	Clasificacion      string                      `json:"clasificacion,omitempty"`
	Organo             string                      `json:"organo,omitempty"`
	ExpedienteRef      string                      `json:"expediente_ref,omitempty"`
	FechaEmision       *time.Time                  `json:"fecha_emision,omitempty"`
	HuellaSHA256       string                      `json:"huella_sha256,omitempty"`
	FirmaRefs          []string                    `json:"firma_refs,omitempty"`
	SelloTiempoRefs    []string                    `json:"sello_tiempo_refs,omitempty"`
	ValidacionFirmaRef string                      `json:"validacion_firma_ref,omitempty"`
	RegistroRef        string                      `json:"registro_ref,omitempty"`
	// PermiteDescarga es un puntero para distinguir una denegacion expresa de
	// la ausencia de permiso para revelar siquiera esta capacidad. Solo se
	// proyecta cuando la decision concede el campo permite_descarga.
	PermiteDescarga *bool `json:"permite_descarga,omitempty"`
}

type ResultadoConsultaPublicaCotejo struct {
	Estado          EstadoConsultaCotejo `json:"estado"`
	Organo          string               `json:"organo,omitempty"`
	TipoDocumental  string               `json:"tipo_documental,omitempty"`
	FechaEmision    *time.Time           `json:"fecha_emision,omitempty"`
	HuellaSHA256    string               `json:"huella_sha256,omitempty"`
	PermiteDescarga bool                 `json:"permite_descarga"`
}
```

ResultadoConsultaPublicaCotejo contiene una lista cerrada de campos que no
pueden identificar por si mismos a una persona ni revelar un expediente.

```go
type ResultadoEjecucionDocumentalAtestadaV4 struct {
	OrdenRef        string
	Estado          string
	AuditoriaRef    string
	EventoOutboxRef string
	RegistradaEn    time.Time
}

type ResultadoPrepararCargaDocumental struct {
	Carga         domain.CargaDocumental
	Instrucciones ports.InstruccionesCargaDirecta
	Recibo        ports.ReciboCargaDirecta
	Repetida      bool
}

type ResultadoReservaCodigoCotejo struct {
	Codigo   domain.CodigoCotejo       `json:"codigo"`
	Secreto  ports.SecretoCodigoCotejo `json:"-"`
	Repetida bool                      `json:"repetida"`
}
```

ResultadoReservaCodigoCotejo mantiene el secreto fuera de cualquier JSON.
Solo el adaptador que prepara el sello PDF/QR puede llamar a Revelar.

```go
type Service struct {
	// Has unexported fields.
}

func NewService(
	modules ports.ModuleRegistryStore,
	audit ports.AuditStore,
	events ports.EventStore,
) (*Service, error)

func (s *Service) BuildMenu(ctx context.Context, principal domain.Principal) ([]domain.MenuEntry, error)

func (s *Service) Modules(ctx context.Context, principal domain.Principal) ([]domain.ModuleManifest, error)

type ServicioAltaOrdenCobro struct {
	// Has unexported fields.
}
```

ServicioAltaOrdenCobro es deliberadamente no exponible: no existe adaptador
HTTP/CLI/MCP ni composicion de produccion. Solo sera habilitable cuando
la fuente autoritativa y RepositorioOrdenesCobro duradero satisfagan sus
contratos. El repositorio confirma agregado, auditoria y outbox en una unica
transaccion; no se ofrece una ruta degradada de tres escrituras.

La confirmacion exige CAS autoritativo de liquidacion y reloj transaccional.
Sigue siendo puerta de despliegue disponer de adaptadores duraderos que
cumplan realmente ambos contratos; mientras no existan, este servicio no
debe cablearse a entradas.

```go
func NuevoServicioAltaOrdenCobro(
	repositorio ports.RepositorioOrdenesCobro,
	liquidaciones FuenteLiquidacionesCobro,
	autorizador ports.Autorizador,
	verificador domain.VerificadorAutenticacionCobro,
	sellador ports.SelladorSolicitudCobro,
	generador ports.GeneradorIDOrdenCobro,
	reloj ports.Reloj,
) (*ServicioAltaOrdenCobro, error)

func (s *ServicioAltaOrdenCobro) Crear(
	ctx context.Context,
	solicitud SolicitudAltaOrdenCobro,
) (AltaOrdenCobroCompletada, error)
```

Crear crea como maximo una orden para la instantanea autoritativa exacta.
Una repeticion semanticamente identica devuelve la orden ya existente.
Toda ausencia, ambiguedad, estado desconocido, representacion no acreditada
o resultado inconsistente falla cerrado antes de persistir.

```go
type ServicioAtestacionesAutorizacionV1 struct {
	// Has unexported fields.
}
```

ServicioAtestacionesAutorizacionV1 fija cabecera y firmante al construir
la composicion. Ninguna peticion de usuario selecciona suite, clave o
audiencia.

```go
func NuevoServicioAtestacionesAutorizacionV1(
	cabecera domain.CabeceraAtestacionAutorizacionV1,
	firmante ports.FirmanteAtestacionesAutorizacionV1,
) (*ServicioAtestacionesAutorizacionV1, error)

func (s *ServicioAtestacionesAutorizacionV1) Atestar(
	ctx context.Context,
	decision domain.DecisionAutorizacion,
) (ports.AtestacionAutorizacionV1, error)

type ServicioAutorizacion struct {
	// Has unexported fields.
}
```

ServicioAutorizacion evalua RBAC seguido de restricciones ABAC sin conocer
PostgreSQL, un IdP ni el generador criptografico concreto. Un fallo en
cualquier dependencia termina siempre en denegacion.

```go
func NuevoServicioAutorizacion(
	fuente ports.FuenteAutorizacion,
	registroConcesiones ports.RegistroDecisionesAutorizacion,
	registroDenegaciones ports.RegistroDenegacionesAutorizacion,
	reloj ports.Reloj,
	generador ports.GeneradorReferenciaDecisionAutorizacion,
	configuracion ConfiguracionServicioAutorizacion,
) (*ServicioAutorizacion, error)

func (s *ServicioAutorizacion) Exigir(ctx context.Context, solicitud domain.SolicitudAutorizacion) (domain.DecisionAutorizacion, error)

type ServicioAutorizacionSolicitudLigadaV2 struct {
	// Has unexported fields.
}
```

ServicioAutorizacionSolicitudLigadaV2 comparte el evaluador RBAC/ABAC, pero
usa puertos de registro distintos y solo expone el metodo V2. No implementa
ports.Autorizador ni puede registrar una decision historica por accidente.

```go
func NuevoServicioAutorizacionSolicitudLigadaV2(
	fuente ports.FuenteAutorizacion,
	registroConcesiones ports.RegistroDecisionesAutorizacionSolicitudLigadaV2,
	registroDenegaciones ports.RegistroDenegacionesAutorizacionSolicitudLigadaV2,
	validadorMotivos ports.ValidadorReferenciaMotivoAutorizacionV2,
	reloj ports.Reloj,
	generador ports.GeneradorReferenciaDecisionAutorizacion,
	configuracion ConfiguracionServicioAutorizacion,
) (*ServicioAutorizacionSolicitudLigadaV2, error)

func (s *ServicioAutorizacionSolicitudLigadaV2) ExigirSolicitudLigadaV2(
	ctx context.Context,
	solicitud domain.SolicitudAutorizacionLigadaV2,
) (domain.DecisionAutorizacion, error)

type ServicioCargaDocumental struct {
	// Has unexported fields.
}
```

ServicioCargaDocumental coordina autorizacion, carga directa, cuarentena,
analisis y promocion sin conocer S3, MinIO, ICAP, ClamAV ni PostgreSQL.
Todos los errores de dependencias o respuestas ambiguas fallan cerrados.

```go
func NuevoServicioCargaDocumental(
	repositorio ports.RepositorioCargasDocumentales,
	autorizador ports.Autorizador,
	almacen ports.AlmacenObjetos,
	gestorCargaDirecta ports.GestorCargaDirecta,
	analizador ports.AnalizadorContenido,
	selladorIdempotencia ports.SelladorIdempotenciaCarga,
	selladorSolicitud ports.SelladorSolicitudCargaDocumental,
	selladorSesion ports.SelladorVinculoSesionCarga,
	seudonimizadorSujeto ports.SeudonimizadorSujetoAlmacen,
	emisorRecibo ports.EmisorReciboCargaDirecta,
	consumidorRecibo ports.ConsumidorReciboCargaDirecta,
	verificadorRecibo ports.VerificadorAtestacionConsumoReciboCargaDirecta,
	generadorID ports.GeneradorIDCargaDocumental,
	reloj ports.Reloj,
	opciones OpcionesServicioCargaDocumental,
) (*ServicioCargaDocumental, error)

func (s *ServicioCargaDocumental) AnalizarYPromover(
	ctx context.Context,
	orden OrdenAnalizarCargaDocumental,
) (domain.CargaDocumental, error)
```

AnalizarYPromover conserva primero el resultado del analisis. Solo despues
obtiene una segunda autorizacion independiente para promocionar contenido
limpio; error, sospecha o falta de conclusion permanecen en cuarentena.

```go
func (s *ServicioCargaDocumental) Confirmar(
	ctx context.Context,
	orden OrdenConfirmarCargaDocumental,
) (domain.CargaDocumental, error)

func (s *ServicioCargaDocumental) Preparar(
	ctx context.Context,
	orden OrdenPrepararCargaDocumental,
) (ResultadoPrepararCargaDocumental, error)

type ServicioCatalogos struct {
	// Has unexported fields.
}

func NuevoServicioCatalogos(
	consulta ports.ConsultaCatalogosConfigurables,
	gobierno ports.RepositorioGobiernoCatalogos,
	autorizador ports.Autorizador,
	reloj ports.Reloj,
) (*ServicioCatalogos, error)

func (s *ServicioCatalogos) ActualizarBorrador(ctx context.Context, orden OrdenActualizarBorradorCatalogo) (domain.CatalogoConfigurable, error)

func (s *ServicioCatalogos) CrearBorrador(ctx context.Context, orden OrdenCrearBorradorCatalogo) (domain.CatalogoConfigurable, error)

func (s *ServicioCatalogos) Publicar(ctx context.Context, orden OrdenPublicarCatalogo) (domain.CatalogoConfigurable, error)

func (s *ServicioCatalogos) Retirar(ctx context.Context, orden OrdenRetirarCatalogo) (domain.CatalogoConfigurable, error)

type ServicioConsultaInternaFuentesAutoridad struct {
	// Has unexported fields.
}
```

ServicioConsultaInternaFuentesAutoridad gobierna la revelacion de una
version exacta. La decision se obtiene antes de invocar el repositorio y se
entrega a este para que sea consumida junto con el recibo de lectura.

```go
func NuevoServicioConsultaInternaFuentesAutoridad(
	consulta ports.ConsultaInternaGobernadaFuentesAutoridad,
	exigidor ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	reloj ports.Reloj,
) (*ServicioConsultaInternaFuentesAutoridad, error)

func (s *ServicioConsultaInternaFuentesAutoridad) ConsultarExacta(
	ctx context.Context,
	orden OrdenConsultaInternaExactaFuenteAutoridad,
) (ResultadoConsultaInternaExactaFuenteAutoridad, error)

type ServicioContextoActor struct {
	// Has unexported fields.
}
```

ServicioContextoActor resuelve la cuenta tecnica autenticada a una unica
persona canonica para el perfil solicitado expresamente. No autentica, no
autoriza y no infiere perfiles; produce el contexto cerrado que consumiran
despues los casos de uso y el PDP.

```go
func NuevoServicioContextoActor(
	fuente ports.FuenteContextoActor,
	reloj ports.Reloj,
) (*ServicioContextoActor, error)

func (s *ServicioContextoActor) Resolver(
	ctx context.Context,
	solicitud domain.SolicitudContextoActor,
) (domain.ContextoActor, error)

type ServicioCotejo struct {
	// Has unexported fields.
}
```

ServicioCotejo coordina gobierno, emision y consulta sin conocer el motor de
base de datos, el KMS/Vault, el generador QR, la firma ni el registro.

```go
func NuevoServicioCotejo(
	politicas ports.CatalogoPoliticasCotejo,
	gobiernoPoliticas ports.RepositorioGobiernoPoliticasCotejo,
	codigos ports.RepositorioCodigosCotejo,
	documentos ports.RepositorioDocumentosLogicos,
	autorizador ports.Autorizador,
	generadorValor ports.GeneradorValorCodigoCotejo,
	generadorID ports.GeneradorIDCodigoCotejo,
	selladorIndice ports.SelladorIndiceCodigoCotejo,
	selladorSolicitud ports.SelladorSolicitudCotejo,
	protector ports.ProtectorCodigoCotejo,
	evidenciasEmision ports.FuenteEvidenciaEmisionDocumento,
	reloj ports.Reloj,
) (*ServicioCotejo, error)

func (s *ServicioCotejo) ActivarCodigoCotejo(ctx context.Context, orden OrdenActivarCodigoCotejo) (domain.CodigoCotejo, error)

func (s *ServicioCotejo) ActualizarBorradorPoliticaCotejo(ctx context.Context, orden OrdenActualizarBorradorPoliticaCotejo) (domain.PoliticaCotejo, error)

func (s *ServicioCotejo) ConsultarCotejoProtegido(ctx context.Context, orden OrdenConsultaProtegidaCotejo) (ResultadoConsultaProtegidaCotejo, error)

func (s *ServicioCotejo) ConsultarCotejoPublico(ctx context.Context, orden OrdenConsultaPublicaCotejo) (ResultadoConsultaPublicaCotejo, error)

func (s *ServicioCotejo) CrearBorradorPoliticaCotejo(ctx context.Context, orden OrdenCrearBorradorPoliticaCotejo) (domain.PoliticaCotejo, error)

func (s *ServicioCotejo) PublicarPoliticaCotejo(ctx context.Context, orden OrdenPublicarPoliticaCotejo) (domain.PoliticaCotejo, error)

func (s *ServicioCotejo) ReservarCodigoCotejo(ctx context.Context, orden OrdenReservarCodigoCotejo) (resultado ResultadoReservaCodigoCotejo, err error)

func (s *ServicioCotejo) RetirarCodigoCotejo(ctx context.Context, orden OrdenRetirarCodigoCotejo) (domain.CodigoCotejo, error)

func (s *ServicioCotejo) RetirarPoliticaCotejo(ctx context.Context, orden OrdenRetirarPoliticaCotejo) (domain.PoliticaCotejo, error)

func (s *ServicioCotejo) SustituirCodigoCotejo(ctx context.Context, orden OrdenSustituirCodigoCotejo) (domain.CodigoCotejo, error)

type ServicioDocumental struct {
	// Has unexported fields.
}
```

ServicioDocumental orquesta la generacion sin conocer PDF, DOCX, S3,
PostgreSQL ni el proveedor de sellado. Todos esos detalles viven detras de
puertos intercambiables.

```go
func NuevoServicioDocumental(
	catalogo ports.CatalogoPlantillasDocumento,
	gobierno ports.RepositorioGobiernoPlantillasDocumento,
	autorizador ports.Autorizador,
	almacen ports.AlmacenContenidoDocumento,
	registroEfectos ports.RegistroEfectosGeneracionDocumental,
	repositorio ports.RepositorioDocumentos,
	repositorioLogico ports.RepositorioDocumentosLogicos,
	selladorDatos ports.SelladorDatosDocumento,
	selladorSolicitud ports.SelladorSolicitudDocumento,
	seudonimizador ports.SeudonimizadorSujetoAlmacen,
	generadorID ports.GeneradorIDDocumento,
	reloj ports.Reloj,
	opciones OpcionesServicioDocumental,
	renderizadores ...ports.RenderizadorDocumento,
) (*ServicioDocumental, error)

func (s *ServicioDocumental) CrearBorradorPlantilla(ctx context.Context, orden OrdenCrearBorradorPlantilla) (domain.PlantillaDocumento, error)

func (s *ServicioDocumental) Generar(ctx context.Context, orden OrdenGenerarDocumento) (domain.DocumentoGenerado, error)

func (s *ServicioDocumental) GenerarDocumentoLogico(ctx context.Context, orden OrdenGenerarDocumentoLogico) (resultado domain.ResultadoGeneracionDocumento, err error)

func (s *ServicioDocumental) PublicarPlantilla(ctx context.Context, orden OrdenPublicarPlantilla) (domain.PlantillaDocumento, error)

type ServicioEjecucionFlujos struct {
	// Has unexported fields.
}

func NuevoServicioEjecucionFlujos(
	definiciones ports.ConsultaDefinicionesFlujo,
	instancias ports.ConsultaInstanciasFlujo,
	repositorio ports.RepositorioInstanciasFlujo,
	evaluador ports.EvaluadorReglasFlujo,
	decisiones ports.RegistroDecisionesReglaFlujo,
	aprobaciones ports.VerificadorAprobacionesFlujo,
	autorizador ports.Autorizador,
	identificadores ports.GeneradorIDInstanciaFlujo,
	reloj ports.Reloj,
) (*ServicioEjecucionFlujos, error)

func (s *ServicioEjecucionFlujos) AplicarTransicion(
	ctx context.Context,
	orden OrdenAplicarTransicionFlujo,
) (domain.InstanciaFlujo, error)

func (s *ServicioEjecucionFlujos) IniciarInstancia(
	ctx context.Context,
	orden OrdenIniciarInstanciaFlujo,
) (domain.InstanciaFlujo, error)

type ServicioGobiernoFlujos struct {
	// Has unexported fields.
}

func NuevoServicioGobiernoFlujos(
	consulta ports.ConsultaDefinicionesFlujo,
	gobierno ports.RepositorioGobiernoFlujos,
	catalogos ports.ConsultaCatalogosConfigurables,
	autorizador ports.Autorizador,
	reloj ports.Reloj,
) (*ServicioGobiernoFlujos, error)

func (s *ServicioGobiernoFlujos) ActualizarBorrador(
	ctx context.Context,
	orden OrdenActualizarBorradorFlujo,
) (domain.DefinicionFlujo, error)

func (s *ServicioGobiernoFlujos) CrearBorrador(
	ctx context.Context,
	orden OrdenCrearBorradorFlujo,
) (domain.DefinicionFlujo, error)

func (s *ServicioGobiernoFlujos) Publicar(ctx context.Context, orden OrdenPublicarFlujo) (domain.DefinicionFlujo, error)

func (s *ServicioGobiernoFlujos) Retirar(ctx context.Context, orden OrdenRetirarFlujo) (domain.DefinicionFlujo, error)

type ServicioMetadatoInstitucionalDocumental struct {
	// Has unexported fields.
}

func NuevoServicioMetadatoInstitucionalDocumental(
	marcadores ports.RegistroMarcadoresMetadatoInstitucional,
	verificador ports.VerificadorEquivalenciaSemanticaDocumental,
) (*ServicioMetadatoInstitucionalDocumental, error)
```

Este servicio es un contrato en sombra. Separar interfaces evita la
autocertificacion directa, pero no prueba por si solo que sean componentes
independientes. El despliegue productivo debe permanecer cerrado hasta que
el verificador tenga identidad, digest y homologacion atestados, distintos
de los del marcador, y el bootstrap coteje esa segregacion.

```go
func (s *ServicioMetadatoInstitucionalDocumental) Incorporar(
	ctx context.Context,
	solicitud ports.SolicitudIncorporarMetadatoInstitucional,
) (ports.ResultadoMetadatoInstitucional, error)

type ServicioResolucionFormatoDocumental struct {
	// Has unexported fields.
}

func NuevoServicioResolucionFormatoDocumental(
	catalogo ports.CatalogoFormatosDocumentales,
	renderizadores ports.RegistroRenderizadoresDocumentales,
) (*ServicioResolucionFormatoDocumental, error)

func (s *ServicioResolucionFormatoDocumental) Resolver(
	ctx context.Context,
	consulta ports.ConsultaFormatoDocumental,
) (FormatoDocumentalResuelto, error)

type ServicioVinculoAutenticacionActorV1 struct {
	// Has unexported fields.
}
```

ServicioVinculoAutenticacionActorV1 es la unica ruta de aplicacion prevista
para construir el bloque que consumira el PDP. Exige un resultado obtenido
de la autoridad de sesion y un ContextoActor resuelto por su propio
servicio.

```go
func NuevoServicioVinculoAutenticacionActorV1(
	revalidador ports.RevalidadorAutenticacionActorV1,
	reloj ports.Reloj,
) (*ServicioVinculoAutenticacionActorV1, error)

func (s *ServicioVinculoAutenticacionActorV1) Crear(
	ctx context.Context,
	solicitud domain.SolicitudRevalidacionAutenticacionActorV1,
	actor domain.ContextoActor,
) (domain.VinculoAutenticacionActorV1, error)

type SolicitudAltaOrdenCobro struct {
	ContextoActor             domain.ContextoActor
	VinculoAutenticacionActor domain.VinculoAutenticacionActorV1
	SesionRef                 string
	HuellaSesionHMAC          string
	LiquidacionRef            string
	CorrelacionRef            string
}
```

SolicitudAltaOrdenCobro no contiene importe, moneda, tarifa, concepto,
sujeto, estado ni caducidad. ContextoActor y Vinculo son capacidades opacas
resueltas por la frontera de identidad. SesionRef y su HMAC deben obtenerse
del contexto confiable del adaptador, nunca del cuerpo de la peticion.

```go
func (SolicitudAltaOrdenCobro) MarshalJSON() ([]byte, error)

func (SolicitudAltaOrdenCobro) MarshalText() ([]byte, error)

func (SolicitudAltaOrdenCobro) String() string

func (*SolicitudAltaOrdenCobro) UnmarshalJSON([]byte) error

type ValidadorReferenciaMotivoCatalogoV2 struct {
	// Has unexported fields.
}
```

ValidadorReferenciaMotivoCatalogoV2 relee la version exacta del catalogo
configurado. Una ReferenciaEntradaCatalogo rellenada por el llamador nunca
basta: deben coincidir documento publicado, huella y entrada vigente.

```go
func NuevoValidadorReferenciaMotivoCatalogoV2(
	consulta ports.ConsultaCatalogosConfigurables,
	catalogoID string,
) (*ValidadorReferenciaMotivoCatalogoV2, error)

func (v *ValidadorReferenciaMotivoCatalogoV2) ValidarReferenciaMotivoAutorizacionV2(
	ctx context.Context,
	referencia domain.ReferenciaEntradaCatalogo,
	instante time.Time,
) error
```

## Paquete `internal/vec/pruebas`

> Package pruebas contiene fabricas exclusivas para dobles automatizados.

Package pruebas contiene fabricas exclusivas para dobles automatizados. No debe
importarse desde composicion productiva.

Package pruebas contiene fabricas exclusivas para dobles automatizados. No forma
parte de ninguna composicion productiva ni ofrece un modo degradado.

### Funciones

```go
func NuevoContextoAlmacen(
	instante time.Time,
	sufijo string,
	accionTecnica string,
	objeto ports.ReferenciaObjetoAlmacen,
) (ports.ContextoOperacionAlmacen, error)
```

NuevoContextoAlmacen crea una capacidad real de la lista positiva para una
prueba externa al paquete ports. No existe modo generico: una accion sin
fabrica productiva devuelve error. El objeto es obligatorio para lectura,
promocion y retencion, y debe ser cero para el resto.

```go
func NuevoContextoYVinculo(
	instante time.Time,
	personaRef string,
	perfilRef string,
	metodo domain.AuthMethod,
	garantia domain.AuthAssurance,
) (domain.ContextoActor, domain.VinculoAutenticacionActorV1, error)
```

NuevoContextoYVinculo crea una sesion autoritativa simulada y la cruza por
la misma fabrica sellada usada por el dominio. Persona y perfil deben ser
referencias opacas validas; la funcion no los corrige ni inventa defaults.

```go
func NuevoVinculoGenerico(instante time.Time) (domain.VinculoAutenticacionActorV1, error)
```

NuevoVinculoGenerico sirve para decisiones aisladas que no ejercitan el
cruce con una solicitud. Sigue pasando por revalidacion y fabrica sellada.
