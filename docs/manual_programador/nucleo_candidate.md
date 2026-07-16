# Nucleo heredado de candidatos (Bolsa)

Parte del [Manual del programador](LEEME.md). Fichero generado con
`scripts/generar_manual_programador.py`; no editar a mano.

## Paquete `internal/candidate/adapters/auth`

> Autenticadores del nucleo heredado de Bolsa, incluido el fake local de pruebas.

### Tipos

```go
type FakeAuthenticator struct {
	// Has unexported fields.
}

func NewFakeAuthenticator() (*FakeAuthenticator, error)
```

NewFakeAuthenticator crea un doble vacio para pruebas unitarias. No deriva
nunca un token del sujeto ni precarga identidades; la composicion fake real
usa exclusivamente el fichero seguro de bootstrap.

```go
func (f *FakeAuthenticator) Authenticate(
	ctx context.Context,
	credentials ports.AuthCredentials,
) (ports.AuthPrincipal, error)

func (f *FakeAuthenticator) Register(principal ports.AuthPrincipal, token string) error

type TrustedHeadersAuthenticator struct {
	// Has unexported fields.
}

func NewTrustedHeadersAuthenticator(cfg config.Config) (*TrustedHeadersAuthenticator, error)

func (a *TrustedHeadersAuthenticator) Authenticate(
	ctx context.Context,
	_ ports.AuthCredentials,
) (ports.AuthPrincipal, error)

func (a *TrustedHeadersAuthenticator) AuthenticateRequest(
	ctx context.Context,
	r *http.Request,
) (ports.AuthPrincipal, error)
```

## Paquete `internal/candidate/adapters/handler`

> Handlers HTTP de la API Bolsa heredada.

### Tipos

```go
type AddMeritCommand = application.AddMeritCommand

type AdministrativeFlowService interface {
	RegisterCandidateDocument(context.Context, string, ports.AuthPrincipal, administrativeDocumentRequest) (administrativeDocumentView, error)
	ListCandidateDocuments(context.Context, string) ([]administrativeDocumentView, error)
	PresentCandidateClaim(context.Context, string, ports.AuthPrincipal, administrativeClaimRequest) (administrativeClaimView, error)
	ListCandidateClaims(context.Context, string, string) ([]administrativeClaimView, error)
	CreateCandidateNotification(context.Context, string, ports.AuthPrincipal, administrativeNotificationRequest) (administrativeNotificationView, error)
	ListCandidateNotifications(context.Context, string) ([]administrativeNotificationView, error)
	SendNotification(context.Context, ports.AuthPrincipal, administrativeNotificationReceiptRequest) (administrativeNotificationView, error)
	MarkNotificationRead(context.Context, ports.AuthPrincipal, administrativeNotificationReceiptRequest) (administrativeNotificationView, error)
	ListCandidateAudit(context.Context, string) ([]administrativeAuditView, error)
	ListAuditByScope(context.Context, string) ([]administrativeAuditView, error)
}

func NewAdministrativeFlowService(
	documents ports.CandidateDocumentRepository,
	usecase *usecases.AdministrativeFlowUseCase,
) AdministrativeFlowService

type BaremoDetailView = application.BaremoDetailView

type BaremoView = application.BaremoView

type CandidateApplicationService = application.CandidateApplicationService

func NewCandidateApplicationService(
	candidates ports.CandidateRepository,
	merits ports.MeritRepository,
	baremo baremoCalculator,
	ruleSet domain.BaremoRuleSet,
) (*CandidateApplicationService, error)

type CandidateView = application.CandidateView

type ConvocatoriaView struct {
	ID      string                `json:"id"`
	Version string                `json:"version"`
	Estado  domain.ProcedureState `json:"estado"`
}

type CreateCandidateCommand = application.CreateCandidateCommand

type ExpedienteView = application.ExpedienteView

type Handler struct {
	// Has unexported fields.
}

func NewHTTPHandler(
	service Service,
	authenticator ports.Authenticator,
	messages *i18n.Catalog,
) (*Handler, error)

func NewHTTPHandlerWithDemoRunner(
	service Service,
	demoRunner ProcedureDemoRunner,
	authenticator ports.Authenticator,
	messages *i18n.Catalog,
) (*Handler, error)

func NewHTTPHandlerWithModules(
	service Service,
	demoRunner ProcedureDemoRunner,
	administrative AdministrativeFlowService,
	authenticator ports.Authenticator,
	messages *i18n.Catalog,
) (*Handler, error)

func NewHTTPHandlerWithModulesAndStatus(
	service Service,
	demoRunner ProcedureDemoRunner,
	administrative AdministrativeFlowService,
	authenticator ports.Authenticator,
	messages *i18n.Catalog,
	status bolsamodule.OperationalStatus,
) (*Handler, error)

func NewHTTPHandlerWithProcedure(
	service Service,
	procedure ProcedureUseCase,
	authenticator ports.Authenticator,
	messages *i18n.Catalog,
) (*Handler, error)

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request)

type ListadoItemView struct {
	ConvocatoriaID string                `json:"convocatoria_id"`
	SolicitudID    string                `json:"solicitud_id"`
	CandidateID    string                `json:"candidate_id"`
	Estado         domain.SolicitudState `json:"estado"`
	TotalPoints    float64               `json:"total_points"`
	SectionPoints  map[string]float64    `json:"section_points,omitempty"`
	RuleSetID      string                `json:"rule_set_id,omitempty"`
	RuleSetVersion string                `json:"rule_set_version,omitempty"`
	Details        []BaremoDetailView    `json:"details,omitempty"`
	Rank           int                   `json:"rank,omitempty"`
}

type ListadoView struct {
	ConvocatoriaID string            `json:"convocatoria_id"`
	Version        string            `json:"version"`
	Items          []ListadoItemView `json:"items"`
}

type MeritDataCommand = application.MeritDataCommand

type MeritView = application.MeritView

type ProcedureDemoRunner interface {
	Run(ctx context.Context) (ProcedureDemoView, error)
}

func NewProcedureDemoRunner(usecase ProcedureUseCase) ProcedureDemoRunner

type ProcedureDemoView struct {
	Convocatoria ConvocatoriaView `json:"convocatoria"`
	Provisional  ListadoView      `json:"provisional"`
	Definitivo   ListadoView      `json:"definitivo"`
}

type ProcedureUseCase = *usecases.ProcedureUseCase

type Service = application.Service
```

## Paquete `internal/candidate/adapters/repository`

> Repositorios en memoria y durables del nucleo heredado de Bolsa.

### Funciones

```go
func NewProcedureRepositories() (*ProcedureConvocatoriaRepository, *ProcedureSolicitudRepository)
func NewRepositories() (*CandidateRepository, *MeritRepository)
```

### Tipos

```go
type AdministrativeAuditTrail struct {
	// Has unexported fields.
}

func NewAdministrativeAuditTrail(store *AdministrativeFlowMemoryStore) *AdministrativeAuditTrail

func (a *AdministrativeAuditTrail) Append(ctx context.Context, scope string, envelope domain.AuditEnvelope) (domain.AuditEntry, error)

func (a *AdministrativeAuditTrail) ListByScope(ctx context.Context, scope string) ([]domain.AuditEntry, error)

type AdministrativeCandidateDocumentRepository struct {
	// Has unexported fields.
}

func NewAdministrativeCandidateDocumentRepository(store *AdministrativeFlowMemoryStore) *AdministrativeCandidateDocumentRepository

func (r *AdministrativeCandidateDocumentRepository) GetByID(ctx context.Context, id string) (domain.CandidateDocument, error)

func (r *AdministrativeCandidateDocumentRepository) ListByCandidate(ctx context.Context, candidateID string) ([]domain.CandidateDocument, error)

func (r *AdministrativeCandidateDocumentRepository) Save(ctx context.Context, document domain.CandidateDocument) error

type AdministrativeClaimRepository struct {
	// Has unexported fields.
}

func NewAdministrativeClaimRepository(store *AdministrativeFlowMemoryStore) *AdministrativeClaimRepository

func (r *AdministrativeClaimRepository) GetByID(ctx context.Context, id string) (domain.Claim, error)

func (r *AdministrativeClaimRepository) ListBySolicitud(ctx context.Context, solicitudID string) ([]domain.Claim, error)

func (r *AdministrativeClaimRepository) Save(ctx context.Context, claim domain.Claim) error

type AdministrativeFlowMemoryStore struct {
	// Has unexported fields.
}

func NewAdministrativeFlowMemoryStore() *AdministrativeFlowMemoryStore

type AdministrativeNotificationRepository struct {
	// Has unexported fields.
}

func NewAdministrativeNotificationRepository(store *AdministrativeFlowMemoryStore) *AdministrativeNotificationRepository

func (r *AdministrativeNotificationRepository) GetByID(ctx context.Context, id string) (domain.Notification, error)

func (r *AdministrativeNotificationRepository) ListByCandidate(ctx context.Context, candidateID string) ([]domain.Notification, error)

func (r *AdministrativeNotificationRepository) Save(ctx context.Context, notification domain.Notification) error

type BaremoResultRepository struct {
	// Has unexported fields.
}

func NewBaremoResultRepository() *BaremoResultRepository

func (r *BaremoResultRepository) GetByCandidate(
	ctx context.Context,
	candidateID string,
) (domain.BaremoResult, bool, error)

func (r *BaremoResultRepository) Save(
	ctx context.Context,
	candidateID string,
	result domain.BaremoResult,
) error

type CandidateRepository struct {
	// Has unexported fields.
}

func NewCandidateRepository(store *MemoryStore) *CandidateRepository

func (r *CandidateRepository) GetByID(
	ctx context.Context,
	id string,
) (domain.Candidate, string, error)

func (r *CandidateRepository) ListByCall(
	ctx context.Context,
	callID string,
) ([]domain.Candidate, error)

func (r *CandidateRepository) Save(
	ctx context.Context,
	callID string,
	candidate domain.Candidate,
) error

type DurableAdministrativeAuditTrail struct {
	*AdministrativeAuditTrail
	// Has unexported fields.
}

func (r *DurableAdministrativeAuditTrail) Append(ctx context.Context, scope string, envelope domain.AuditEnvelope) (domain.AuditEntry, error)

func (r *DurableAdministrativeAuditTrail) ListByScope(ctx context.Context, scope string) ([]domain.AuditEntry, error)

type DurableBaremoResultRepository struct {
	*BaremoResultRepository
	// Has unexported fields.
}

func (r *DurableBaremoResultRepository) Save(ctx context.Context, candidateID string, result domain.BaremoResult) error

type DurableCandidateDocumentRepository struct {
	*AdministrativeCandidateDocumentRepository
	// Has unexported fields.
}

func (r *DurableCandidateDocumentRepository) Save(ctx context.Context, document domain.CandidateDocument) error

type DurableCandidateRepository struct {
	*CandidateRepository
	// Has unexported fields.
}

func (r *DurableCandidateRepository) Save(ctx context.Context, callID string, candidate domain.Candidate) error

type DurableClaimRepository struct {
	*AdministrativeClaimRepository
	// Has unexported fields.
}

func (r *DurableClaimRepository) Save(ctx context.Context, claim domain.Claim) error

type DurableFileStore struct {
	// Has unexported fields.
}

func NewDurableFileStore(path string) (*DurableFileStore, error)

func (s *DurableFileStore) AdministrativeAuditTrail() *DurableAdministrativeAuditTrail

func (s *DurableFileStore) BaremoResultRepository() *DurableBaremoResultRepository

func (s *DurableFileStore) CandidateDocumentRepository() *DurableCandidateDocumentRepository

func (s *DurableFileStore) CandidateRepository() *DurableCandidateRepository

func (s *DurableFileStore) ClaimRepository() *DurableClaimRepository

func (s *DurableFileStore) MeritRepository() *DurableMeritRepository

func (s *DurableFileStore) NotificationRepository() *DurableNotificationRepository

func (s *DurableFileStore) ProcedureConvocatoriaRepository() *DurableProcedureConvocatoriaRepository

func (s *DurableFileStore) ProcedureSolicitudRepository() *DurableProcedureSolicitudRepository

type DurableMeritRepository struct {
	*MeritRepository
	// Has unexported fields.
}

func (r *DurableMeritRepository) Save(ctx context.Context, candidateID string, merit domain.Merit) error

type DurableNotificationRepository struct {
	*AdministrativeNotificationRepository
	// Has unexported fields.
}

func (r *DurableNotificationRepository) Save(ctx context.Context, notification domain.Notification) error

type DurableProcedureConvocatoriaRepository struct {
	*ProcedureConvocatoriaRepository
	// Has unexported fields.
}

func (r *DurableProcedureConvocatoriaRepository) GetByID(ctx context.Context, id string) (ports.ConvocatoriaRecord, error)

func (r *DurableProcedureConvocatoriaRepository) Save(ctx context.Context, convocatoria ports.ConvocatoriaRecord) error

type DurableProcedureSolicitudRepository struct {
	*ProcedureSolicitudRepository
	// Has unexported fields.
}

func (r *DurableProcedureSolicitudRepository) GetByID(ctx context.Context, id string) (ports.SolicitudRecord, error)

func (r *DurableProcedureSolicitudRepository) ListByConvocatoria(ctx context.Context, convocatoriaID string) ([]ports.SolicitudRecord, error)

func (r *DurableProcedureSolicitudRepository) Save(ctx context.Context, solicitud ports.SolicitudRecord) error

type MemoryStore struct {
	// Has unexported fields.
}

func NewMemoryStore() *MemoryStore

type MeritRepository struct {
	// Has unexported fields.
}

func NewMeritRepository(store *MemoryStore) *MeritRepository

func (r *MeritRepository) ListByCandidate(
	ctx context.Context,
	candidateID string,
) ([]domain.Merit, error)

func (r *MeritRepository) Save(
	ctx context.Context,
	candidateID string,
	merit domain.Merit,
) error

type ProcedureConvocatoriaRepository struct {
	// Has unexported fields.
}

func NewProcedureConvocatoriaRepository(store *ProcedureMemoryStore) *ProcedureConvocatoriaRepository

func (r *ProcedureConvocatoriaRepository) GetByID(
	ctx context.Context,
	id string,
) (ports.ConvocatoriaRecord, error)

func (r *ProcedureConvocatoriaRepository) Save(
	ctx context.Context,
	convocatoria ports.ConvocatoriaRecord,
) error

type ProcedureMemoryStore struct {
	// Has unexported fields.
}

func NewProcedureMemoryStore() *ProcedureMemoryStore

type ProcedureSolicitudRepository struct {
	// Has unexported fields.
}

func NewProcedureSolicitudRepository(store *ProcedureMemoryStore) *ProcedureSolicitudRepository

func (r *ProcedureSolicitudRepository) GetByID(
	ctx context.Context,
	id string,
) (ports.SolicitudRecord, error)

func (r *ProcedureSolicitudRepository) ListByConvocatoria(
	ctx context.Context,
	convocatoriaID string,
) ([]ports.SolicitudRecord, error)

func (r *ProcedureSolicitudRepository) Save(
	ctx context.Context,
	solicitud ports.SolicitudRecord,
) error
```

## Paquete `internal/candidate/application`

> Casos de uso heredados de candidatos de Bolsa.

### Variables

```go
var (
	ErrServiceDependenciesRequired = errors.New("candidate application service: repositories and baremo usecase are required")
	ErrCallIDRequired              = errors.New("candidate application service: an explicit call is required")
	ErrCallNotConfigured           = errors.New("candidate application service: call is not configured")
	ErrCandidateCallBindingInvalid = errors.New("candidate application service: candidate call binding is invalid")
)
```

### Tipos

```go
type AddMeritCommand struct {
	ID     string            `json:"id"`
	Tipo   domain.MeritType  `json:"tipo"`
	Datos  MeritDataCommand  `json:"datos"`
	Estado domain.MeritState `json:"estado,omitempty"`
}

type BaremoCalculator interface {
	CalcularAutobaremo(context.Context, string, domain.BaremoRuleSet) (domain.BaremoResult, error)
}

type BaremoDetailView struct {
	MeritID       string  `json:"merit_id"`
	MeritType     string  `json:"merit_type"`
	Section       string  `json:"section"`
	RawPoints     float64 `json:"raw_points"`
	AppliedPoints float64 `json:"applied_points"`
	Capped        bool    `json:"capped"`
}

type BaremoView struct {
	TotalPoints    float64            `json:"total_points"`
	SectionPoints  map[string]float64 `json:"section_points"`
	RuleSetID      string             `json:"rule_set_id"`
	RuleSetVersion string             `json:"rule_set_version"`
	Details        []BaremoDetailView `json:"details"`
}

type CandidateApplicationService struct {
	// Has unexported fields.
}

func NewCandidateApplicationService(
	candidates ports.CandidateRepository,
	merits ports.MeritRepository,
	baremo BaremoCalculator,
	ruleSet domain.BaremoRuleSet,
) (*CandidateApplicationService, error)

func (s *CandidateApplicationService) AddMerit(
	ctx context.Context,
	candidateID string,
	command AddMeritCommand,
) (MeritView, error)

func (s *CandidateApplicationService) CalculateBaremo(
	ctx context.Context,
	candidateID string,
) (BaremoView, error)

func (s *CandidateApplicationService) CreateCandidate(
	ctx context.Context,
	command CreateCandidateCommand,
) (CandidateView, error)

func (s *CandidateApplicationService) ExportExpediente(
	ctx context.Context,
	candidateID string,
) (ExpedienteView, error)

type CandidateView struct {
	ID     string `json:"id"`
	DNI    string `json:"dni"`
	Nombre string `json:"nombre"`
	Email  string `json:"email"`
	CallID string `json:"call_id"`
}

type CreateCandidateCommand struct {
	ID     string `json:"id"`
	DNI    string `json:"dni"`
	Nombre string `json:"nombre"`
	Email  string `json:"email"`
	CallID string `json:"call_id,omitempty"`
}

type ExpedienteView struct {
	Candidate CandidateView `json:"candidate"`
	Merits    []MeritView   `json:"merits"`
	Baremo    BaremoView    `json:"baremo"`
}

type MeritDataCommand struct {
	Meses       int     `json:"meses,omitempty"`
	Horas       int     `json:"horas,omitempty"`
	PuntosFijos float64 `json:"puntos_fijos,omitempty"`
}

type MeritView struct {
	ID     string            `json:"id"`
	Tipo   domain.MeritType  `json:"tipo"`
	Datos  MeritDataCommand  `json:"datos"`
	Estado domain.MeritState `json:"estado"`
}

type ProfessionalActionView struct {
	ID          string `json:"id"`
	LabelKey    string `json:"label_key"`
	Priority    string `json:"priority"`
	ModuleID    string `json:"module_id"`
	DeadlineKey string `json:"deadline_key"`
}

type ProfessionalAuditView struct {
	ReceiptKey     string `json:"receipt_key"`
	TimelineKey    string `json:"timeline_key"`
	LastActorKey   string `json:"last_actor_key"`
	LastChangedKey string `json:"last_changed_key"`
}

type ProfessionalAutobaremoView struct {
	TotalPoints    float64                         `json:"total_points"`
	RuleSetID      string                          `json:"rule_set_id"`
	RuleSetVersion string                          `json:"rule_set_version"`
	Sections       []ProfessionalBaremoSectionView `json:"sections"`
	Details        []BaremoDetailView              `json:"details"`
	Warnings       []ProfessionalWarningView       `json:"warnings"`
	Explanation    []ProfessionalExplanationView   `json:"explanation"`
}

type ProfessionalBaremoSectionView struct {
	ID        string  `json:"id"`
	LabelKey  string  `json:"label_key"`
	Points    float64 `json:"points"`
	StatusKey string  `json:"status_key"`
}

type ProfessionalCapabilityView struct {
	ID       string `json:"id"`
	Mode     string `json:"mode"`
	Route    string `json:"route,omitempty"`
	Method   string `json:"method,omitempty"`
	LabelKey string `json:"label_key"`
}

type ProfessionalExplanationView struct {
	ID         string `json:"id"`
	MessageKey string `json:"message_key"`
	ReceiptKey string `json:"receipt_key"`
}

type ProfessionalMetricView struct {
	ID       string  `json:"id"`
	LabelKey string  `json:"label_key"`
	Value    float64 `json:"value"`
	UnitKey  string  `json:"unit_key"`
	StateKey string  `json:"state_key"`
}

type ProfessionalModuleView struct {
	ID               string `json:"id"`
	Group            string `json:"group"`
	Accent           string `json:"accent"`
	Icon             string `json:"icon"`
	LabelKey         string `json:"label_key"`
	DescriptionKey   string `json:"description_key"`
	StatusKey        string `json:"status_key"`
	PrimaryActionKey string `json:"primary_action_key"`
	EmptyStateKey    string `json:"empty_state_key"`
	Count            int    `json:"count"`
	AlertCount       int    `json:"alert_count"`
}

type ProfessionalPortalHeaderView struct {
	TitleKey       string `json:"title_key"`
	StateKey       string `json:"state_key"`
	CallID         string `json:"call_id"`
	NextActionKey  string `json:"next_action_key"`
	DeadlineKey    string `json:"deadline_key"`
	LastUpdatedKey string `json:"last_updated_key"`
}

type ProfessionalPortalView struct {
	Locale         string                       `json:"locale"`
	Candidate      CandidateView                `json:"candidate"`
	Header         ProfessionalPortalHeaderView `json:"header"`
	Modules        []ProfessionalModuleView     `json:"modules"`
	Capabilities   []ProfessionalCapabilityView `json:"capabilities"`
	Summary        []ProfessionalMetricView     `json:"summary"`
	PendingActions []ProfessionalActionView     `json:"pending_actions"`
	Autobaremo     ProfessionalAutobaremoView   `json:"autobaremo"`
	Audit          ProfessionalAuditView        `json:"audit"`
}

func NewProfessionalPortalView(expediente ExpedienteView) ProfessionalPortalView

type ProfessionalWarningView struct {
	Code       string `json:"code"`
	MessageKey string `json:"message_key"`
	ModuleID   string `json:"module_id"`
	Severity   string `json:"severity"`
}

type Service interface {
	CreateCandidate(context.Context, CreateCandidateCommand) (CandidateView, error)
	AddMerit(context.Context, string, AddMeritCommand) (MeritView, error)
	CalculateBaremo(context.Context, string) (BaremoView, error)
	ExportExpediente(context.Context, string) (ExpedienteView, error)
}
```

## Paquete `internal/candidate/domain`

> Tipos y reglas puras del dominio heredado de candidatos.

### Constantes

```go
const (
	MeritTypeExperienciaMismaCategoria MeritType = "experiencia_misma_categoria"
	MeritTypeExperienciaOtraCategoria  MeritType = "experiencia_otra_categoria"
	MeritTypeFormacionTitulo           MeritType = "formacion_titulo"
	MeritTypeFormacionCurso            MeritType = "formacion_curso"
	MeritTypeOtros                     MeritType = "otros"

	MeritTypeExperienceSameCategory  = MeritTypeExperienciaMismaCategoria
	MeritTypeExperienceOtherCategory = MeritTypeExperienciaOtraCategoria
	MeritTypeTrainingDegree          = MeritTypeFormacionTitulo
	MeritTypeTrainingCourse          = MeritTypeFormacionCurso
	MeritTypeOther                   = MeritTypeOtros
)
const (
	MeritStateBorrador    MeritState = "Borrador"
	MeritStatePresentado  MeritState = "Presentado"
	MeritStateValidado    MeritState = "Validado"
	MeritStateRechazado   MeritState = "Rechazado"
	MeritStateSubsanacion MeritState = "Subsanacion"

	MeritStateDraft      = MeritStateBorrador
	MeritStateSubmitted  = MeritStatePresentado
	MeritStateValidated  = MeritStateValidado
	MeritStateRejected   = MeritStateRechazado
	MeritStateCorrection = MeritStateSubsanacion
)
const (
	ProcedureStateBorrador    = dominiobolsa.EstadoConvocatoriaBorrador
	ProcedureStateInscripcion = dominiobolsa.EstadoConvocatoriaInscripcion
	ProcedureStateSubsanacion = dominiobolsa.EstadoConvocatoriaSubsanacion
	ProcedureStateAlegaciones = dominiobolsa.EstadoConvocatoriaAlegaciones
	ProcedureStateDefinitiva  = dominiobolsa.EstadoConvocatoriaDefinitiva
	ProcedureStateCerrada     = dominiobolsa.EstadoConvocatoriaCerrada
)
```

### Variables

```go
var (
	ErrAuditInvalid      = errors.New("audit entry is invalid")
	ErrAuditChainInvalid = errors.New("audit chain is invalid")
)
var (
	ErrBaremoRuleSetInvalid = errors.New("baremo rule set is invalid")
	ErrBaremoMeritNoRule    = errors.New("baremo merit has no rule")
	ErrBaremoTieInvalid     = errors.New("baremo tie break is invalid")
)
var (
	ErrClaimInvalid    = errors.New("claim is invalid")
	ErrClaimTransition = errors.New("claim transition is invalid")
)
var (
	ErrDocumentInvalid     = errors.New("document evidence is invalid")
	ErrDocumentQuarantined = errors.New("document evidence is quarantined")
	ErrFileInvalid         = errors.New("electronic file is invalid")
)
var (
	ErrCandidateIDRequired     = errors.New("candidate id is required")
	ErrCandidateDNIRequired    = errors.New("candidate dni is required")
	ErrCandidateNombreRequired = errors.New("candidate nombre is required")
	ErrCandidateEmailRequired  = errors.New("candidate email is required")

	ErrMeritIDRequired   = errors.New("merit id is required")
	ErrMeritTypeInvalid  = errors.New("merit type is invalid")
	ErrMeritStateInvalid = errors.New("merit state is invalid")
	ErrMeritDataInvalid  = errors.New("merit data is invalid")
	ErrMeritTransition   = errors.New("merit transition is invalid")
)
var (
	ErrNotificationInvalid    = errors.New("notification is invalid")
	ErrNotificationTransition = errors.New("notification transition is invalid")
)
var (
	ErrProcedureInvalid    = dominiobolsa.ErrConvocatoriaInvalida
	ErrProcedureTransition = errors.New("procedure transition is invalid")
	ErrProcedureRanking    = errors.New("procedure ranking is invalid")
)
var ErrCandidateDocumentInvalid = errors.New("candidate document is invalid")
```

### Funciones

```go
func HashAuditPayload(payload []byte) string
func VerifyAuditChain(entries []AuditEntry, signingRef string) error
```

### Tipos

```go
type AVStatus string

const (
	AVStatusPending AVStatus = "PENDING"
	AVStatusClean   AVStatus = "CLEAN"
	AVStatusThreat  AVStatus = "THREAT"
)
func (s AVStatus) IsValid() bool

type AuditEntry struct {
	Sequence      int
	OccurredAt    time.Time
	Actor         string
	Action        string
	PayloadHash   string
	PrevSignature string
	Signature     string
}

func NewAuditEntry(previous *AuditEntry, envelope AuditEnvelope, signingRef string) (AuditEntry, error)

func (e AuditEntry) Validate() error

type AuditEnvelope struct {
	Actor      string
	Action     string
	OccurredAt time.Time
	Payload    []byte
}

func (e AuditEnvelope) Validate() error

type BaremoMeritRule struct {
	MeritType     MeritType
	Section       BaremoSection
	Unit          BaremoUnit
	PointsPerUnit float64
}

type BaremoMeritScore struct {
	MeritID                  string
	MeritType                MeritType
	Section                  BaremoSection
	RawPoints, AppliedPoints float64
	Capped                   bool
}

type BaremoResult struct {
	TotalPoints               float64
	SectionPoints             map[BaremoSection]float64
	RuleSetID, RuleSetVersion string
	Details                   []BaremoMeritScore
}

func CalcularAutobaremo(merits []Merit, ruleSet BaremoRuleSet) (BaremoResult, error)

func CalcularBaremoOficial(merits []Merit, ruleSet BaremoRuleSet) (BaremoResult, error)
```

CalcularBaremoOficial solo computa decisiones administrativas validadas. El
flujo productivo nuevo sustituira el estado mutable por decisiones firmadas
append-only; esta funcion cierra mientras tanto el fallo del prototipo
heredado que puntuaba rechazados y subsanaciones.

```go
type BaremoRuleSet struct {
	// Has unexported fields.
}

func NewBaremoRuleSet(config BaremoRuleSetConfig) (BaremoRuleSet, error)

func (r BaremoRuleSet) Config() BaremoRuleSetConfig

func (r BaremoRuleSet) Validate() error

type BaremoRuleSetConfig struct {
	ConvocatoriaID, Version, SorteoLetra string
	MeritRules                           []BaremoMeritRule
	SectionCaps                          []BaremoSectionCap
	TieBreakRules                        []BaremoTieBreakRule
}

type BaremoSection string

const (
	BaremoSectionExperiencia BaremoSection = "experiencia"
	BaremoSectionFormacion   BaremoSection = "formacion"
	BaremoSectionOtros       BaremoSection = "otros"
)
func (s BaremoSection) IsValid() bool

type BaremoSectionCap struct {
	Section   BaremoSection
	MaxPoints float64
}

type BaremoTieBreakDecision struct {
	Rule                             BaremoTieBreakRule
	WinnerID, AValue, BValue, Reason string
}

type BaremoTieBreakResult struct {
	WinnerID  string
	IsTie     bool
	Decisions []BaremoTieBreakDecision
}

func Desempate(a, b BaremoTieCandidate, ruleSet BaremoRuleSet) (BaremoTieBreakResult, error)

type BaremoTieBreakRule string

const (
	BaremoTieMayorExperiencia BaremoTieBreakRule = "mayor_experiencia"
	BaremoTieMayorFormacion   BaremoTieBreakRule = "mayor_formacion"
	BaremoTieLetraSorteo      BaremoTieBreakRule = "letra_sorteo"
	BaremoTieCandidateID      BaremoTieBreakRule = "candidate_id"
)
func (r BaremoTieBreakRule) IsValid() bool

type BaremoTieCandidate struct {
	CandidateID, SorteoKey string
	Result                 BaremoResult
}

type BaremoUnit string

const (
	BaremoUnitMeses           BaremoUnit = "meses"
	BaremoUnitHoras           BaremoUnit = "horas"
	BaremoUnitMerito          BaremoUnit = "merito"
	BaremoUnitPuntosDeclarado BaremoUnit = "puntos_declarado"
)
func (u BaremoUnit) IsValid() bool

type BolsaState string

const (
	BolsaStateSinConstituir BolsaState = "SinConstituir"
	BolsaStateProvisional   BolsaState = "Provisional"
	BolsaStateEnAlegaciones BolsaState = "EnAlegaciones"
	BolsaStateDefinitiva    BolsaState = "Definitiva"
	BolsaStateAgotada       BolsaState = "Agotada"
	BolsaStateCerrada       BolsaState = "Cerrada"
)
func (s BolsaState) CanTransition(to BolsaState) bool

func (s BolsaState) IsValid() bool

func (s BolsaState) Transition(to BolsaState) (BolsaState, error)

type CSV string

func NewCSV(value string) (CSV, error)

func (c CSV) Validate() error

type Candidate struct {
	ID     string
	DNI    string
	Nombre string
	Email  string
}
```

Candidate represents an applicant in the Diputacion de Granada pool.

```go
func NewCandidate(id, dni, nombre, email string) (Candidate, error)

func (c Candidate) Validate() error

type CandidateDocument struct {
	ID           string
	CandidateID  string
	SolicitudID  string
	ProcedureID  string
	Purpose      DocumentPurpose
	Evidence     DocumentEvidence
	RegisteredBy string
	RegisteredAt time.Time
}

func NewCandidateDocument(input CandidateDocumentInput) (CandidateDocument, error)

func (d CandidateDocument) AuditPayload() []byte

func (d CandidateDocument) EnsureExportable() error

func (d CandidateDocument) ExportManifestItem() (DocumentManifestItem, error)

func (d CandidateDocument) Validate() error

type CandidateDocumentInput struct {
	ID           string
	CandidateID  string
	SolicitudID  string
	ProcedureID  string
	Purpose      DocumentPurpose
	Evidence     DocumentEvidence
	RegisteredBy string
	RegisteredAt time.Time
}

type Claim struct {
	ID          string
	CandidateID string
	SolicitudID string
	Text        string
	Documents   []CandidateDocument
	State       ClaimState
	PresentedBy string
	PresentedAt time.Time
	Receipt     ClaimReceipt
}

func NewClaim(input ClaimInput) (Claim, error)

func (c Claim) AuditPayload() []byte

func (c Claim) CanTransition(to ClaimState) bool

func (c *Claim) Transition(to ClaimState) error

func (c Claim) Validate() error

type ClaimInput struct {
	ID          string
	CandidateID string
	SolicitudID string
	Text        string
	Documents   []CandidateDocument
	PresentedBy string
	PresentedAt time.Time
	ReceiptCSV  string
}

type ClaimReceipt struct {
	CSV         CSV
	Actor       string
	IssuedAt    time.Time
	PayloadHash string
}

func (r ClaimReceipt) Validate() error

type ClaimState string

const (
	ClaimStatePresentada  ClaimState = "Presentada"
	ClaimStateEnRevision  ClaimState = "EnRevision"
	ClaimStateEstimada    ClaimState = "Estimada"
	ClaimStateDesestimada ClaimState = "Desestimada"
	ClaimStateArchivada   ClaimState = "Archivada"
)
func (s ClaimState) IsValid() bool

type Convocatoria = dominiobolsa.Convocatoria

func NewConvocatoria(id, version string) (Convocatoria, error)

type DocumentEvidence struct {
	CSV          CSV
	DigestSHA256 string
	Refs         DocumentExternalRefs
	ENI          ENIMetadata
	AVStatus     AVStatus
	SubmittedBy  string
	SubmittedAt  time.Time
	Signatures   []SignatureEvidence
}

func NewDocumentEvidence(input DocumentEvidenceInput) (DocumentEvidence, error)

func (d DocumentEvidence) EnsureExportable() error

func (d DocumentEvidence) IsQuarantined() bool

func (d *DocumentEvidence) MarkAVStatus(status AVStatus) error

func (d DocumentEvidence) Validate() error

type DocumentEvidenceInput struct {
	CSV          string
	DigestSHA256 string
	Refs         DocumentExternalRefs
	ENI          ENIMetadata
	AVStatus     AVStatus
	SubmittedBy  string
	SubmittedAt  time.Time
	Signatures   []SignatureEvidence
}

type DocumentExternalRefs struct {
	// Opaque refs resolved by adapters such as MinIO, OpenBao or TSA clients.
	StorageObjectRef string
	VaultSecretRef   string
	TSAStampRef      string
}

func (r DocumentExternalRefs) Validate() error

type DocumentManifestItem struct {
	CSV           CSV
	Who           string
	What          string
	When          time.Time
	SignatureRefs []string
	TSAStampRef   string
}

type DocumentPurpose string

const (
	DocumentPurposeSolicitud    DocumentPurpose = "Solicitud"
	DocumentPurposeSubsanacion  DocumentPurpose = "Subsanacion"
	DocumentPurposeAlegacion    DocumentPurpose = "Alegacion"
	DocumentPurposeResolucion   DocumentPurpose = "Resolucion"
	DocumentPurposeNotificacion DocumentPurpose = "Notificacion"
)
func (p DocumentPurpose) IsValid() bool

type ENIMetadata struct {
	DocumentID   string
	Organ        string
	Procedure    string
	DocumentType string
	Title        string
	Format       string
	Language     string
	CreatedAt    time.Time
}

func (m ENIMetadata) Validate() error

type ElectronicFile struct {
	ID          string
	CandidateID string
	ProcedureID string
	CreatedBy   string
	CreatedAt   time.Time
	Documents   []DocumentEvidence
}

func (f ElectronicFile) ExportManifest() (ElectronicFileManifest, error)

func (f ElectronicFile) Validate() error

type ElectronicFileManifest struct {
	FileID      string
	CandidateID string
	ProcedureID string
	Items       []DocumentManifestItem
}

type Merit struct {
	ID     string
	Tipo   MeritType
	Datos  MeritData
	Estado MeritState
}

func NewMerit(id string, tipo MeritType, datos MeritData) (Merit, error)

func (m Merit) CanTransition(to MeritState) bool

func (m *Merit) Transition(to MeritState) error

func (m Merit) Validate() error

type MeritData struct {
	Meses       int
	Horas       int
	PuntosFijos float64
}

func (d MeritData) Validate() error

type MeritState string

func (s MeritState) IsValid() bool

type MeritType string

func (t MeritType) IsValid() bool

type Notification struct {
	ID          string
	CandidateID string
	SolicitudID string
	Type        string
	Subject     string
	Body        string
	State       NotificationState
	CreatedBy   string
	CreatedAt   time.Time
	Receipts    []NotificationReceipt
}

func NewNotification(input NotificationInput) (Notification, error)

func (n Notification) AuditPayload() []byte

func (n *Notification) MarkRead(receipt NotificationReceipt) error

func (n *Notification) Send(receipt NotificationReceipt) error

func (n Notification) Validate() error

type NotificationInput struct {
	ID          string
	CandidateID string
	SolicitudID string
	Type        string
	Subject     string
	Body        string
	CreatedBy   string
	CreatedAt   time.Time
}

type NotificationReceipt struct {
	CSV         CSV
	RecipientID string
	Channel     string
	IssuedAt    time.Time
	PayloadHash string
}

func NewNotificationReceipt(csv, recipientID, channel string, issuedAt time.Time, payload []byte) (NotificationReceipt, error)

func (r NotificationReceipt) Validate() error

type NotificationState string

const (
	NotificationStateCreada    NotificationState = "Creada"
	NotificationStateEnviada   NotificationState = "Enviada"
	NotificationStateLeida     NotificationState = "Leida"
	NotificationStateFallida   NotificationState = "Fallida"
	NotificationStateCancelada NotificationState = "Cancelada"
)
func (s NotificationState) IsValid() bool

type ProcedureState = dominiobolsa.EstadoConvocatoria

type RankedSolicitud struct {
	Position             int
	SolicitudID          string
	CandidateID          string
	TotalPoints          float64
	PreviousOrderReason  string
	PreviousTieDecisions []BaremoTieBreakDecision
}

func RankSolicitudes(entries []SolicitudRankingEntry, ruleSet BaremoRuleSet) ([]RankedSolicitud, error)

type SignatureEvidence struct {
	SignerID     string
	SignatureRef string
	SignedAt     time.Time
}

func (s SignatureEvidence) Validate() error

type SolicitudRankingEntry struct {
	SolicitudID string
	CandidateID string
	Estado      SolicitudState
	Result      BaremoResult
	SorteoKey   string
}

func (e SolicitudRankingEntry) Validate() error

type SolicitudState string

const (
	SolicitudStateBorrador             SolicitudState = "Borrador"
	SolicitudStateInscrita             SolicitudState = "Inscrita"
	SolicitudStateSubsanacionRequerida SolicitudState = "SubsanacionRequerida"
	SolicitudStateSubsanada            SolicitudState = "Subsanada"
	SolicitudStateAdmitidaProvisional  SolicitudState = "AdmitidaProvisional"
	SolicitudStateExcluidaProvisional  SolicitudState = "ExcluidaProvisional"
	SolicitudStateAlegacionPresentada  SolicitudState = "AlegacionPresentada"
	SolicitudStateAdmitidaDefinitiva   SolicitudState = "AdmitidaDefinitiva"
	SolicitudStateExcluidaDefinitiva   SolicitudState = "ExcluidaDefinitiva"
)
func (s SolicitudState) CanTransition(to SolicitudState) bool

func (s SolicitudState) IsValid() bool

func (s SolicitudState) Transition(to SolicitudState) (SolicitudState, error)
```

## Paquete `internal/candidate/ports`

> Contratos hexagonales del nucleo heredado de Bolsa.

### Variables

```go
var (
	ErrCandidateDocumentNotFound = errors.New("administrative flow repository: candidate document not found")
	ErrClaimNotFound             = errors.New("administrative flow repository: claim not found")
	ErrNotificationNotFound      = errors.New("administrative flow repository: notification not found")
)
var (
	ErrAuthMechanismRequired    = errors.New("auth mechanism is required")
	ErrAuthMechanismUnsupported = errors.New("auth mechanism is unsupported")
	ErrAuthSubjectRequired      = errors.New("auth subject is required")
	ErrAuthSubjectInvalid       = errors.New("auth subject is invalid")
	ErrAuthTokenRequired        = errors.New("auth token is required")
	ErrAuthTokenInvalid         = errors.New("auth token is invalid")
	ErrAuthRoleInvalid          = errors.New("auth role is invalid")
	ErrAuthPrincipalInvalid     = errors.New("auth principal is invalid")
	ErrAuthenticationFailed     = errors.New("authentication failed")
)
var (
	// ErrCandidateNotFound signals that no candidate exists for the requested ID.
	ErrCandidateNotFound = errors.New("candidate repository: candidate not found")
	// ErrCandidateCallInvalid impide interpretar una convocatoria ausente,
	// no canonica o con comodines como un ambito valido.
	ErrCandidateCallInvalid = errors.New("candidate repository: candidate call is invalid")
	// ErrMeritNotFound signals that no merit exists for the requested repository query.
	ErrMeritNotFound = errors.New("merit repository: merit not found")
)
var (
	ErrConvocatoriaNotFound = errors.New("procedure repository: convocatoria not found")
	ErrSolicitudNotFound    = errors.New("procedure repository: solicitud not found")
)
```

### Tipos

```go
type AdministrativeAuditTrail interface {
	Append(ctx context.Context, scope string, envelope domain.AuditEnvelope) (domain.AuditEntry, error)
	ListByScope(ctx context.Context, scope string) ([]domain.AuditEntry, error)
}

type AuthCredentials struct {
	Mechanism  AuthMechanism
	Subject    string
	Token      string
	Assertions map[string]string
}

func (c AuthCredentials) Validate() error

type AuthMechanism string

const (
	AuthMechanismKerberosAD AuthMechanism = "kerberos_ad"
	AuthMechanismDNIe       AuthMechanism = "dnie"
	AuthMechanismClave      AuthMechanism = "clave"
)
func (m AuthMechanism) IsValid() bool

type AuthPrincipal struct {
	Subject     string
	DisplayName string
	Email       string
	Role        AuthRole
	Roles       []AuthRole
	Mechanism   AuthMechanism
	Method      AuthMechanism
	Attributes  map[string]string
}

func (p AuthPrincipal) AllRoles() []AuthRole

func (p AuthPrincipal) AuthMethod() AuthMechanism

func (p AuthPrincipal) HasRole(role AuthRole) bool

func (p AuthPrincipal) PrimaryRole() AuthRole

func (p AuthPrincipal) Validate() error

type AuthRole string

const (
	AuthRoleCandidate   AuthRole = "candidate"
	AuthRoleValidatorL1 AuthRole = "validator_l1"
	AuthRoleValidatorL2 AuthRole = "validator_l2"
	AuthRoleSystemAdmin AuthRole = "system_admin"

	AuthRoleCiudadano       AuthRole = AuthRoleCandidate
	AuthRolePersonalInterno AuthRole = AuthRoleValidatorL1
)
func (r AuthRole) IsValid() bool

type Authenticator interface {
	Authenticate(context.Context, AuthCredentials) (Identity, error)
}

type CandidateDocumentRepository interface {
	Save(ctx context.Context, document domain.CandidateDocument) error
	GetByID(ctx context.Context, id string) (domain.CandidateDocument, error)
	ListByCandidate(ctx context.Context, candidateID string) ([]domain.CandidateDocument, error)
}

type CandidateRepository interface {
	Save(ctx context.Context, callID string, candidate domain.Candidate) error
	// GetByID devuelve tambien la convocatoria duradera exacta. El caso de uso
	// no puede reconstruirla mediante memoria de proceso ni un valor por defecto.
	GetByID(ctx context.Context, id string) (domain.Candidate, string, error)
	ListByCall(ctx context.Context, callID string) ([]domain.Candidate, error)
}
```

CandidateRepository is the outbound persistence port for candidates.

```go
type ClaimRepository interface {
	Save(ctx context.Context, claim domain.Claim) error
	GetByID(ctx context.Context, id string) (domain.Claim, error)
	ListBySolicitud(ctx context.Context, solicitudID string) ([]domain.Claim, error)
}

type ConvocatoriaRecord struct {
	Convocatoria domain.Convocatoria
	RuleSet      domain.BaremoRuleSet
}

type ConvocatoriaRepository interface {
	Save(ctx context.Context, convocatoria ConvocatoriaRecord) error
	GetByID(ctx context.Context, id string) (ConvocatoriaRecord, error)
}

type Identity = AuthPrincipal

type MeritRepository interface {
	Save(ctx context.Context, candidateID string, merit domain.Merit) error
	ListByCandidate(ctx context.Context, candidateID string) ([]domain.Merit, error)
}
```

MeritRepository is the outbound persistence port for candidate merits.

```go
type NotificationRepository interface {
	Save(ctx context.Context, notification domain.Notification) error
	GetByID(ctx context.Context, id string) (domain.Notification, error)
	ListByCandidate(ctx context.Context, candidateID string) ([]domain.Notification, error)
}

type SolicitudRecord struct {
	ID             string
	ConvocatoriaID string
	CandidateID    string
	SorteoKey      string
	Estado         domain.SolicitudState
	Result         domain.BaremoResult
}

type SolicitudRepository interface {
	Save(ctx context.Context, solicitud SolicitudRecord) error
	GetByID(ctx context.Context, id string) (SolicitudRecord, error)
	ListByConvocatoria(ctx context.Context, convocatoriaID string) ([]SolicitudRecord, error)
}
```

## Paquete `internal/candidate/usecases`

> Casos de uso del flujo administrativo heredado.

### Variables

```go
var (
	ErrAdministrativeFlowPortsRequired = errors.New("administrative flow usecase: ports are required")
	ErrAdministrativeDocumentRequired  = errors.New("administrative flow usecase: document is required")
	ErrAdministrativeClaimRequired     = errors.New("administrative flow usecase: claim is required")
	ErrAdministrativeNoticeRequired    = errors.New("administrative flow usecase: notification is required")
	ErrAdministrativeRecipientMismatch = errors.New("administrative flow usecase: notification recipient mismatch")
	ErrAdministrativeAuditRequired     = errors.New("administrative flow usecase: audit scope is required")
)
var (
	ErrBaremoMeritRepositoryRequired  = errors.New("baremo usecase: merit repository is required")
	ErrBaremoResultRepositoryRequired = errors.New("baremo usecase: result repository is required")
)
var (
	ErrProcedureRepositoryRequired   = errors.New("procedure usecase: repositories are required")
	ErrProcedureConvocatoriaRequired = errors.New("procedure usecase: convocatoria is required")
	ErrProcedureSolicitudRequired    = errors.New("procedure usecase: solicitud is required")
)
var ErrContextRequired = errors.New("candidate usecase: context is required")
```

ErrContextRequired impide sustituir silenciosamente la cancelacion o la
ausencia del contexto que delimita una operacion administrativa.

### Funciones

```go
func BaremoRuleSetFor(convocatoriaID, version string) (domain.BaremoRuleSet, error)
```

### Tipos

```go
type AdministrativeFlowUseCase struct {
	// Has unexported fields.
}

func NewAdministrativeFlowUseCase(
	documents ports.CandidateDocumentRepository,
	claims ports.ClaimRepository,
	notifications ports.NotificationRepository,
	audit ports.AdministrativeAuditTrail,
) (*AdministrativeFlowUseCase, error)

func (u *AdministrativeFlowUseCase) CreateNotification(
	ctx context.Context,
	command CreateNotificationCommand,
) (domain.Notification, domain.AuditEntry, error)

func (u *AdministrativeFlowUseCase) ListAuditByScope(ctx context.Context, scope string) ([]domain.AuditEntry, error)

func (u *AdministrativeFlowUseCase) ListClaimsBySolicitud(ctx context.Context, solicitudID string) ([]domain.Claim, error)

func (u *AdministrativeFlowUseCase) ListNotificationsByCandidate(ctx context.Context, candidateID string) ([]domain.Notification, error)

func (u *AdministrativeFlowUseCase) MarkNotificationRead(
	ctx context.Context,
	command ReceiptCommand,
) (domain.Notification, domain.AuditEntry, error)

func (u *AdministrativeFlowUseCase) PresentClaim(
	ctx context.Context,
	command PresentClaimCommand,
) (domain.Claim, domain.AuditEntry, error)

func (u *AdministrativeFlowUseCase) RegisterCandidateDocument(
	ctx context.Context,
	command RegisterCandidateDocumentCommand,
) (domain.CandidateDocument, domain.AuditEntry, error)

func (u *AdministrativeFlowUseCase) SendNotification(
	ctx context.Context,
	command ReceiptCommand,
) (domain.Notification, domain.AuditEntry, error)

type BaremoResultRepository interface {
	Save(ctx context.Context, candidateID string, result domain.BaremoResult) error
}

type BaremoUseCase struct {
	// Has unexported fields.
}

func NewBaremoUseCase(
	merits ports.MeritRepository,
	results BaremoResultRepository,
) (BaremoUseCase, error)

func (u BaremoUseCase) CalcularAutobaremo(
	ctx context.Context,
	candidateID string,
	ruleSet domain.BaremoRuleSet,
) (domain.BaremoResult, error)

func (u BaremoUseCase) PresentarSolicitud(
	ctx context.Context,
	candidateID string,
	ruleSet domain.BaremoRuleSet,
) (domain.BaremoResult, error)

func (u BaremoUseCase) PuntuacionProvisional(
	ctx context.Context,
	candidateID string,
	ruleSet domain.BaremoRuleSet,
) (float64, error)

type CrearConvocatoriaCommand struct {
	ID      string
	Version string
	RuleSet domain.BaremoRuleSet
}

type CreateNotificationCommand struct {
	ID          string
	CandidateID string
	SolicitudID string
	Type        string
	Subject     string
	Body        string
	CreatedBy   string
	CreatedAt   time.Time
}

type Listado struct {
	ConvocatoriaID string
	Version        string
	Items          []ListadoItem
}

type ListadoItem struct {
	SolicitudID string
	CandidateID string
	Estado      domain.SolicitudState
	Result      domain.BaremoResult
	Rank        int
}

type PresentClaimCommand struct {
	ID          string
	CandidateID string
	SolicitudID string
	Text        string
	Documents   []domain.CandidateDocument
	PresentedBy string
	PresentedAt time.Time
	ReceiptCSV  string
}

type ProcedureUseCase struct {
	// Has unexported fields.
}

func NewProcedureUseCase(
	convocatorias ports.ConvocatoriaRepository,
	solicitudes ports.SolicitudRepository,
) (*ProcedureUseCase, error)

func (u *ProcedureUseCase) CrearConvocatoria(
	ctx context.Context,
	command CrearConvocatoriaCommand,
) (ports.ConvocatoriaRecord, error)

func (u *ProcedureUseCase) EnsureConvocatoria(
	ctx context.Context,
	command CrearConvocatoriaCommand,
) (ports.ConvocatoriaRecord, error)

func (u *ProcedureUseCase) EnsureSolicitud(
	ctx context.Context,
	command RegistrarSolicitudCommand,
) (ports.SolicitudRecord, error)

func (u *ProcedureUseCase) ListadoActual(
	ctx context.Context,
	convocatoriaID string,
) (Listado, error)

func (u *ProcedureUseCase) PublicarListadoDefinitivo(
	ctx context.Context,
	convocatoriaID string,
	admitidas map[string]bool,
) (Listado, error)

func (u *ProcedureUseCase) PublicarListadoProvisional(
	ctx context.Context,
	convocatoriaID string,
	admitidas map[string]bool,
) (Listado, error)

func (u *ProcedureUseCase) RegistrarSolicitud(
	ctx context.Context,
	command RegistrarSolicitudCommand,
) (ports.SolicitudRecord, error)

type ReceiptCommand struct {
	NotificationID string
	CSV            string
	RecipientID    string
	Channel        string
	IssuedAt       time.Time
}

type RegisterCandidateDocumentCommand struct {
	ID           string
	CandidateID  string
	SolicitudID  string
	ProcedureID  string
	Purpose      domain.DocumentPurpose
	Evidence     domain.DocumentEvidence
	RegisteredBy string
	RegisteredAt time.Time
}

type RegistrarSolicitudCommand struct {
	ID             string
	ConvocatoriaID string
	CandidateID    string
	SorteoKey      string
	Merits         []domain.Merit
}
```
