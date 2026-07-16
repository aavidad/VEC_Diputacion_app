# Modulos Personal, Cronos, Dietas y Administracion

Parte del [Manual del programador](LEEME.md). Fichero generado con
`scripts/generar_manual_programador.py`; no editar a mano.

## Paquete `internal/modules/administracion`

> Manifiesto del modulo Administracion para el shell VEC.

### Constantes

```go
const (
	ModuleID                     = "vec.module.administracion"
	PermissionRolesManage        = "vec.roles.manage"
	PermissionCatalogsManage     = "vec.catalogs.manage"
	PermissionIntegrationsManage = "vec.integrations.manage"
	PermissionMonitoringRead     = "vec.monitoring.read"
	PermissionAuditRead          = "vec.audit.read"
	ActionAssignRole             = "vec.roles.assign"
	ActionPublishCatalog         = "vec.catalog.publish"
)
```

### Funciones

```go
func Manifest() domain.ModuleManifest
```

## Paquete `internal/modules/cronos`

> Manifiesto del modulo Cronos: fichajes, permisos, vacaciones y saldos.

### Constantes

```go
const (
	ModuleID                    = "vec.module.cronos"
	PermissionTimeRead          = "cronos.fichaje.read"
	PermissionTimeManage        = "cronos.fichaje.manage"
	PermissionScheduleRead      = "cronos.horario.read"
	PermissionScheduleManage    = "cronos.horario.manage"
	PermissionLeaveRead         = "cronos.permiso.read"
	PermissionLeaveManage       = "cronos.permiso.manage"
	PermissionApprovalManage    = "cronos.aprobacion.manage"
	PermissionAudit             = "cronos.audit.read"
	ActionReviewJustification   = "cronos.jornada.justificacion.review"
	ActionReviewLeaveAndHoliday = "cronos.permiso.vacacion.review"
)
```

### Funciones

```go
func Manifest() domain.ModuleManifest
```

## Paquete `internal/modules/cronos/adapters/memory`

> Adaptadores en memoria del modulo Cronos.

### Tipos

```go
type Store struct {
	// Has unexported fields.
}

func NewStore() *Store

func (s *Store) ListLeaveBalances(_ context.Context, employeeID string, year int) ([]domain.LeaveBalance, error)

func (s *Store) ListLeavePolicies(_ context.Context) ([]domain.LeavePolicy, error)

func (s *Store) ListLeaveRequests(_ context.Context, employeeID string, year int) ([]domain.LeaveRequest, error)

func (s *Store) ListProfiles(_ context.Context) ([]domain.ScheduleProfile, error)

func (s *Store) ListWorkdays(_ context.Context, employeeIDs []string, date time.Time) ([]domain.Workday, error)

func (s *Store) SaveLeaveBalance(_ context.Context, balance domain.LeaveBalance) error

func (s *Store) SaveLeavePolicy(_ context.Context, policy domain.LeavePolicy) error

func (s *Store) SaveLeaveRequest(_ context.Context, request domain.LeaveRequest) error

func (s *Store) SaveProfile(_ context.Context, profile domain.ScheduleProfile) error

func (s *Store) SaveWorkday(_ context.Context, workday domain.Workday) error
```

## Paquete `internal/modules/cronos/application`

> Casos de uso del modulo Cronos.

### Variables

```go
var ErrServiceDependencyRequired = errors.New("cronos service dependency required")
```

### Tipos

```go
type Service struct {
	// Has unexported fields.
}

func NewService(store ports.Store) (*Service, error)

func (s *Service) LeaveBalances(ctx context.Context, employeeID string, year int) ([]domain.LeaveBalance, error)

func (s *Service) LeavePolicies(ctx context.Context) ([]domain.LeavePolicy, error)

func (s *Service) LeaveRequests(ctx context.Context, employeeID string, year int) ([]domain.LeaveRequest, error)

func (s *Service) Profiles(ctx context.Context) ([]domain.ScheduleProfile, error)

func (s *Service) RegisterWorkday(ctx context.Context, workday domain.Workday) error

func (s *Service) RequestLeave(ctx context.Context, request domain.LeaveRequest) (domain.LeaveRequest, error)

func (s *Service) SaveLeaveBalance(ctx context.Context, balance domain.LeaveBalance) error

func (s *Service) SaveLeavePolicy(ctx context.Context, policy domain.LeavePolicy) error

func (s *Service) SaveProfile(ctx context.Context, profile domain.ScheduleProfile) error

func (s *Service) Snapshot(ctx context.Context, date time.Time, employeeIDs []string) (Snapshot, error)

type Snapshot struct {
	Date          time.Time
	Profiles      []domain.ScheduleProfile
	Workdays      []domain.Workday
	Results       []domain.DayResult
	LeavePolicies []domain.LeavePolicy
	LeaveBalances []domain.LeaveBalance
	LeaveRequests []domain.LeaveRequest
}
```

## Paquete `internal/modules/cronos/domain`

> Reglas puras del dominio Cronos.

### Variables

```go
var (
	ErrLeavePolicyInvalid   = errors.New("cronos leave policy invalid")
	ErrLeaveBalanceInvalid  = errors.New("cronos leave balance invalid")
	ErrLeaveRequestInvalid  = errors.New("cronos leave request invalid")
	ErrLeaveBalanceExceeded = errors.New("cronos leave balance exceeded")
)
var (
	ErrScheduleProfileInvalid = errors.New("cronos schedule profile invalid")
	ErrWorkdayInvalid         = errors.New("cronos workday invalid")
)
```

### Funciones

```go
func DailyReductionForAge(age int) time.Duration
func FormatHHMM(value time.Duration) string
func FormatLeaveAmount(amount int, unit LeaveUnit) string
func Minutes(hours, minutes int) time.Duration
```

### Tipos

```go
type AuthorizedAbsence struct {
	Ref      string
	Kind     string
	Duration time.Duration
	Paid     bool
}

type DayResult struct {
	EmployeeID          string
	Date                time.Time
	Age                 int
	ProfileID           string
	Theoretical         time.Duration
	Reduction           time.Duration
	AuthorizedAbsence   time.Duration
	Worked              time.Duration
	OnSiteWorked        time.Duration
	Telework            time.Duration
	Balance             time.Duration
	OpenEntry           bool
	FixedCoverageBreach bool
	Incidents           []Incident
}

func EvaluateWorkday(day Workday) (DayResult, error)

type Incident struct {
	Code     string
	Label    string
	Severity IncidentSeverity
}

type IncidentSeverity string

const (
	IncidentInfo     IncidentSeverity = "info"
	IncidentWarning  IncidentSeverity = "warning"
	IncidentBlocking IncidentSeverity = "blocking"
)
type LeaveBalance struct {
	EmployeeID string
	Year       int
	PolicyID   string
	Granted    int
	Requested  int
	Approved   int
	Consumed   int
}

func (b LeaveBalance) Remaining() int

func (b LeaveBalance) Validate() error

type LeavePolicy struct {
	ID               string
	Name             string
	Unit             LeaveUnit
	AnnualAllowance  int
	Requestable      bool
	RequiresDocument bool
	RequiresApproval bool
	MinRequest       int
	MaxRequest       int
	PayrollImpact    bool
}

func DefaultLeavePolicies() []LeavePolicy

func (p LeavePolicy) Validate() error

type LeaveRequest struct {
	ID          string
	EmployeeID  string
	PolicyID    string
	From        time.Time
	To          time.Time
	Amount      int
	Unit        LeaveUnit
	State       LeaveState
	Reason      string
	DocumentRef string
	CreatedAt   time.Time
}

func (r LeaveRequest) Validate(policy LeavePolicy, balance LeaveBalance) error

type LeaveState string

const (
	LeaveStateDraft     LeaveState = "borrador"
	LeaveStateRequested LeaveState = "solicitado"
	LeaveStateReview    LeaveState = "pendiente_responsable"
	LeaveStateApproved  LeaveState = "aprobado"
	LeaveStateDenied    LeaveState = "denegado"
	LeaveStateCancelled LeaveState = "cancelado"
	LeaveStateConsumed  LeaveState = "disfrutado"
)
type LeaveUnit string

const (
	LeaveUnitDay  LeaveUnit = "dia"
	LeaveUnitHour LeaveUnit = "hora"
)
type Punch struct {
	At      time.Time
	Kind    PunchKind
	Channel string
	Mode    WorkMode
}

type PunchKind string

const (
	PunchEntry      PunchKind = "entrada"
	PunchExit       PunchKind = "salida"
	PunchPauseStart PunchKind = "inicio_pausa"
	PunchPauseEnd   PunchKind = "fin_pausa"
)
type ScheduleProfile struct {
	ID               string
	Name             string
	Unit             string
	Flexible         bool
	AllowsTelework   bool
	RequiresCoverage bool
	DailyTarget      time.Duration
	WeeklyTarget     time.Duration
	EntryWindowStart time.Duration
	EntryWindowEnd   time.Duration
	CoreStart        time.Duration
	CoreEnd          time.Duration
	EffectiveFrom    time.Time
	EffectiveTo      time.Time
}

func (p ScheduleProfile) Validate() error

type WorkMode string

const (
	WorkModeOnSite   WorkMode = "presencial"
	WorkModeTelework WorkMode = "teletrabajo"
)
type Workday struct {
	EmployeeID       string
	Date             time.Time
	Age              int
	Profile          ScheduleProfile
	Punches          []Punch
	TeleworkDuration time.Duration
	Absences         []AuthorizedAbsence
}
```

## Paquete `internal/modules/cronos/ports`

> Contratos hexagonales del modulo Cronos.

### Tipos

```go
type Store interface {
	SaveProfile(context.Context, domain.ScheduleProfile) error
	ListProfiles(context.Context) ([]domain.ScheduleProfile, error)
	SaveWorkday(context.Context, domain.Workday) error
	// ListWorkdays exige una lista positiva no vacia de personas. Una fecha
	// concreta sin sujetos concretos no equivale a permiso para leer a todos.
	ListWorkdays(context.Context, []string, time.Time) ([]domain.Workday, error)
	SaveLeavePolicy(context.Context, domain.LeavePolicy) error
	ListLeavePolicies(context.Context) ([]domain.LeavePolicy, error)
	SaveLeaveBalance(context.Context, domain.LeaveBalance) error
	ListLeaveBalances(context.Context, string, int) ([]domain.LeaveBalance, error)
	SaveLeaveRequest(context.Context, domain.LeaveRequest) error
	ListLeaveRequests(context.Context, string, int) ([]domain.LeaveRequest, error)
}
```

## Paquete `internal/modules/dietas`

> Manifiesto del modulo Dietas: comisiones, kilometraje y liquidaciones.

### Constantes

```go
const (
	ModuleID                  = "vec.module.dietas"
	PermissionExpenseRead     = "dietas.gasto.read"
	PermissionExpenseManage   = "dietas.gasto.manage"
	PermissionRouteRead       = "dietas.ruta.read"
	PermissionRouteManage     = "dietas.ruta.manage"
	PermissionApprovalManage  = "dietas.aprobacion.manage"
	PermissionAudit           = "dietas.audit.read"
	ActionReviewTravelExpense = "dietas.comision.review"
	ActionReviewRouteKM       = "dietas.ruta.km.review"
)
```

### Funciones

```go
func Manifest() domain.ModuleManifest
func ProvinceLocalityMaps() []map[string]any
func ProvinceRouteItineraryExampleMaps() []map[string]any
func ProvinceRouteMatrixStatus() map[string]any
func ProvinceRoutePairMaps() []map[string]any
func ProvinceRoutePointMaps() []map[string]any
```

### Tipos

```go
type ProvinceLocality struct {
	INECode string
	Name    string
	Kind    string
	Source  string
}

func ProvinceLocalities() []ProvinceLocality

type RouteItinerary struct {
	ID                  string
	Label               string
	Stops               []string
	Legs                []RouteLeg
	TotalKM             float64
	TotalMinutes        int
	MileageRateEURKM    float64
	MileageAmountEUR    float64
	AllowanceSuggestion string
	MatrixVersion       string
	AuditState          string
}

func ProvinceRouteItineraryExamples() []RouteItinerary

type RouteLeg struct {
	From        string
	To          string
	DistanceKM  float64
	DurationMin int
	State       string
}

type RoutePair struct {
	ID            string
	From          string
	To            string
	FromCode      string
	ToCode        string
	DistanceKM    float64
	DurationMin   int
	Allowance     string
	MatrixVersion string
	Source        string
	State         string
}

func ProvinceRoutePairs() []RoutePair

type RoutePoint struct {
	Code             string
	Name             string
	Kind             string
	MunicipalityCode string
	MunicipalityName string
	Latitude         float64
	Longitude        float64
	Source           string
	State            string
}

func ProvinceRoutePoints() []RoutePoint
```

## Paquete `internal/modules/personal`

> Manifiesto del modulo Personal/Nominas.

### Constantes

```go
const (
	ModuleID                       = "vec.module.personal"
	PermissionEmployeeRead         = "personal.empleado.read"
	PermissionEmployeeManage       = "personal.empleado.manage"
	PermissionPositionRead         = "personal.puesto.read"
	PermissionPositionManage       = "personal.puesto.manage"
	PermissionPayrollRead          = "personal.nomina.read"
	PermissionPayrollManage        = "personal.nomina.manage"
	PermissionSeniorityRead        = "personal.antiguedad.read"
	PermissionCertificateManage    = "personal.certificado.manage"
	PermissionAdministrativeManage = "personal.situacion.manage"
	PermissionAudit                = "personal.audit.read"
	ActionReviewPayrollIncident    = "personal.nomina.incidencia.review"
	ActionIssueServiceCertificate  = "personal.certificado.servicios.issue"
)
```

### Funciones

```go
func Manifest() domain.ModuleManifest
```

## Paquete `internal/modules/personal/adapters/file`

> Adaptador de catalogo Personal sobre fichero local.

### Tipos

```go
type CatalogStore struct {
	// Has unexported fields.
}

func NewCatalogStore(path string) (*CatalogStore, error)

func (s *CatalogStore) DeleteCategory(ctx context.Context, slug string) (bool, error)

func (s *CatalogStore) DeletePosition(ctx context.Context, code string) (bool, error)

func (s *CatalogStore) GetCategory(ctx context.Context, slug string) (domain.ProfessionalCategory, bool, error)

func (s *CatalogStore) GetPosition(ctx context.Context, code string) (domain.RPTPosition, bool, error)

func (s *CatalogStore) ImportPositions(ctx context.Context, cmd domain.RPTImportCommand) (domain.RPTImportReceipt, error)

func (s *CatalogStore) ListCatalogEntries(ctx context.Context) ([]domain.CatalogEntry, error)

func (s *CatalogStore) ListCategories(ctx context.Context, filter domain.ProfessionalCategoryFilter) (domain.ProfessionalCategoryPage, error)

func (s *CatalogStore) ListPositions(ctx context.Context, filter domain.RPTPositionFilter) (domain.RPTPositionPage, error)

func (s *CatalogStore) Stats(ctx context.Context) (domain.CatalogStats, error)

func (s *CatalogStore) UpsertCatalogEntry(ctx context.Context, entry domain.CatalogEntry) error

func (s *CatalogStore) UpsertCategory(ctx context.Context, category domain.ProfessionalCategory) error

func (s *CatalogStore) UpsertPosition(ctx context.Context, position domain.RPTPosition) error
```

## Paquete `internal/modules/personal/adapters/memory`

> Adaptador de catalogo Personal en memoria.

### Tipos

```go
type CatalogStore struct {
	// Has unexported fields.
}

func NewCatalogStore() *CatalogStore

func (s *CatalogStore) DeleteCategory(ctx context.Context, slug string) (bool, error)

func (s *CatalogStore) DeletePosition(ctx context.Context, code string) (bool, error)

func (s *CatalogStore) GetCategory(ctx context.Context, slug string) (domain.ProfessionalCategory, bool, error)

func (s *CatalogStore) GetPosition(ctx context.Context, code string) (domain.RPTPosition, bool, error)

func (s *CatalogStore) ImportPositions(ctx context.Context, cmd domain.RPTImportCommand) (domain.RPTImportReceipt, error)

func (s *CatalogStore) ListCatalogEntries(ctx context.Context) ([]domain.CatalogEntry, error)

func (s *CatalogStore) ListCategories(ctx context.Context, filter domain.ProfessionalCategoryFilter) (domain.ProfessionalCategoryPage, error)

func (s *CatalogStore) ListPositions(ctx context.Context, filter domain.RPTPositionFilter) (domain.RPTPositionPage, error)

func (s *CatalogStore) Stats(ctx context.Context) (domain.CatalogStats, error)

func (s *CatalogStore) UpsertCatalogEntry(ctx context.Context, entry domain.CatalogEntry) error

func (s *CatalogStore) UpsertCategory(ctx context.Context, category domain.ProfessionalCategory) error

func (s *CatalogStore) UpsertPosition(ctx context.Context, position domain.RPTPosition) error
```

## Paquete `internal/modules/personal/application`

> Casos de uso del modulo Personal/Nominas.

### Variables

```go
var (
	ErrCatalogStoreRequired = errors.New("personal catalog store required")
	ErrRPTPositionNotFound  = errors.New("personal rpt position not found")
	ErrCategoryNotFound     = errors.New("personal professional category not found")
)
```

### Tipos

```go
type CatalogService struct {
	// Has unexported fields.
}

func NewCatalogService(store ports.CatalogStore) (*CatalogService, error)

func (s *CatalogService) DeleteCategory(ctx context.Context, slug string) (bool, error)

func (s *CatalogService) DeletePosition(ctx context.Context, code string) (bool, error)

func (s *CatalogService) GetCategory(ctx context.Context, slug string) (domain.ProfessionalCategory, error)

func (s *CatalogService) GetPosition(ctx context.Context, code string) (domain.RPTPosition, error)

func (s *CatalogService) ImportPositions(ctx context.Context, cmd domain.RPTImportCommand) (domain.RPTImportReceipt, error)

func (s *CatalogService) ListCatalogEntries(ctx context.Context) ([]domain.CatalogEntry, error)

func (s *CatalogService) ListCategories(ctx context.Context, filter domain.ProfessionalCategoryFilter) (domain.ProfessionalCategoryPage, error)

func (s *CatalogService) ListPositions(ctx context.Context, filter domain.RPTPositionFilter) (domain.RPTPositionPage, error)

func (s *CatalogService) Stats(ctx context.Context) (domain.CatalogStats, error)

func (s *CatalogService) UpsertCatalogEntry(ctx context.Context, entry domain.CatalogEntry) error

func (s *CatalogService) UpsertCategory(ctx context.Context, category domain.ProfessionalCategory) error

func (s *CatalogService) UpsertPosition(ctx context.Context, position domain.RPTPosition) (domain.RPTPosition, error)
```

## Paquete `internal/modules/personal/domain`

> Reglas puras del dominio Personal: RPT, puestos y categorias.

### Variables

```go
var (
	ErrRPTPositionInvalid          = errors.New("personal rpt position invalid")
	ErrProfessionalCategoryInvalid = errors.New("personal professional category invalid")
	ErrCatalogEntryInvalid         = errors.New("personal catalog entry invalid")
)
var (
	ErrPayrollProfileInvalid = errors.New("personal payroll profile invalid")
	ErrPayrollDraftInvalid   = errors.New("personal payroll draft invalid")
)
```

### Tipos

```go
type CatalogEntry struct {
	Catalog   string `json:"catalog"`
	Code      string `json:"code"`
	Label     string `json:"label"`
	Source    string `json:"source"`
	ModuleKey string `json:"module_key"`
	State     string `json:"state"`
	Usage     string `json:"usage"`
}

func (e CatalogEntry) Validate() error

type CatalogStats struct {
	Positions        int            `json:"positions"`
	Categories       int            `json:"categories"`
	CatalogEntries   int            `json:"catalog_entries"`
	PositionsByGroup map[string]int `json:"positions_by_group"`
	CategoriesByArea map[string]int `json:"categories_by_area"`
	PendingLegend    int            `json:"pending_legend"`
}

type CategoryAlias struct {
	Alias        string `json:"alias"`
	CategorySlug string `json:"category_slug"`
	Source       string `json:"source"`
}

type CategoryRule struct {
	ID             string  `json:"id"`
	Label          string  `json:"label"`
	Section        string  `json:"section"`
	PointsPerMonth float64 `json:"points_per_month"`
	Source         string  `json:"source"`
}

type EmploymentRegime string

const (
	RegimeCareerCivilServant  EmploymentRegime = "funcionario_carrera"
	RegimeInterimCivilServant EmploymentRegime = "funcionario_interino"
	RegimeLabourStaff         EmploymentRegime = "personal_laboral"
	RegimeTemporaryLabour     EmploymentRegime = "personal_laboral_temporal"
	RegimeSeniorManager       EmploymentRegime = "alto_cargo"
)
type PayrollConcept struct {
	Code                  string
	Name                  string
	Kind                  PayrollConceptKind
	AmountCents           int64
	CountsForContribution bool
	CountsForTax          bool
	SourceRef             string
}

type PayrollConceptKind string

const (
	ConceptBasicPay        PayrollConceptKind = "retribucion_basica"
	ConceptComplementary   PayrollConceptKind = "retribucion_complementaria"
	ConceptSeniority       PayrollConceptKind = "trienio_antiguedad"
	ConceptExtraPay        PayrollConceptKind = "paga_extraordinaria"
	ConceptVariable        PayrollConceptKind = "productividad_gratificacion"
	ConceptSocialSecurity  PayrollConceptKind = "cotizacion_seguridad_social"
	ConceptTaxWithholding  PayrollConceptKind = "retencion_irpf"
	ConceptAdvanceDeduct   PayrollConceptKind = "anticipo_o_reintegro"
	ConceptCourtAttachment PayrollConceptKind = "embargo"
)
type PayrollDraft struct {
	EmployeeID       string
	Period           PayrollPeriod
	Regime           EmploymentRegime
	Concepts         []PayrollConcept
	GrossCents       int64
	DeductionsCents  int64
	NetCents         int64
	ContributionBase int64
	TaxBase          int64
}

func BuildPayrollDraft(profile PayrollProfile, period PayrollPeriod, concepts []PayrollConcept) (PayrollDraft, error)

type PayrollPeriod struct {
	Year  int
	Month time.Month
	Extra bool
}

type PayrollProfile struct {
	EmployeeID        string
	Regime            EmploymentRegime
	Group             string
	Level             string
	PositionID        string
	SeniorityTriennia int
	IBAN              string
	ActiveFrom        time.Time
	ActiveTo          time.Time
}

func (p PayrollProfile) Validate() error

type ProfessionalCategory struct {
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	Area       string `json:"area"`
	Source     string `json:"source"`
	SourcePath string `json:"source_path"`
	ModuleKey  string `json:"module_key"`
	State      string `json:"state"`
	Usage      string `json:"usage"`
}

func (c ProfessionalCategory) Validate() error

type ProfessionalCategoryFilter struct {
	Query  string `json:"query,omitempty"`
	Area   string `json:"area,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

type ProfessionalCategoryPage struct {
	Items  []ProfessionalCategory `json:"items"`
	Total  int                    `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
}

type RPTImportCommand struct {
	Source    string        `json:"source"`
	Version   string        `json:"version"`
	Replace   bool          `json:"replace"`
	Positions []RPTPosition `json:"positions"`
}

type RPTImportReceipt struct {
	Source   string `json:"source"`
	Version  string `json:"version"`
	Imported int    `json:"imported"`
	Replaced bool   `json:"replaced"`
}

type RPTPosition struct {
	Code               string `json:"code"`
	Name               string `json:"name"`
	Dot                int    `json:"dot"`
	Type               string `json:"type"`
	Administration     string `json:"administration"`
	Provision          string `json:"provision"`
	Group              string `json:"group"`
	Area               string `json:"area"`
	Scale              string `json:"scale"`
	CategoryCode       string `json:"category_code"`
	CategorySlug       string `json:"category_slug"`
	Delegation         string `json:"delegation,omitempty"`
	CenterCode         string `json:"center_code,omitempty"`
	CenterName         string `json:"center_name,omitempty"`
	DestinationLevel   int    `json:"destination_level"`
	SpecificKind       string `json:"specific_kind,omitempty"`
	AnnualAmountCents  int64  `json:"annual_amount_cents,omitempty"`
	GCPCTLevel         string `json:"gcp_ct_level,omitempty"`
	SpecificComplement string `json:"specific_complement"`
	GeoDispersion      string `json:"geo_dispersion"`
	Telework           string `json:"telework"`
	Coverage           string `json:"coverage"`
	State              string `json:"state"`
	Source             string `json:"source"`
	Requirements       string `json:"requirements,omitempty"`
	Observations       string `json:"observations"`
	Page               int    `json:"page,omitempty"`
	Raw                string `json:"raw,omitempty"`
}

func (p RPTPosition) Validate() error

type RPTPositionFilter struct {
	Query      string `json:"query,omitempty"`
	Group      string `json:"group,omitempty"`
	CenterCode string `json:"center_code,omitempty"`
	Provision  string `json:"provision,omitempty"`
	State      string `json:"state,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty"`
}

type RPTPositionPage struct {
	Items  []RPTPosition `json:"items"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}
```

## Paquete `internal/modules/personal/ports`

> Contratos hexagonales del modulo Personal.

### Tipos

```go
type CatalogStore interface {
	UpsertPosition(context.Context, domain.RPTPosition) error
	GetPosition(context.Context, string) (domain.RPTPosition, bool, error)
	DeletePosition(context.Context, string) (bool, error)
	ListPositions(context.Context, domain.RPTPositionFilter) (domain.RPTPositionPage, error)
	ImportPositions(context.Context, domain.RPTImportCommand) (domain.RPTImportReceipt, error)

	UpsertCategory(context.Context, domain.ProfessionalCategory) error
	GetCategory(context.Context, string) (domain.ProfessionalCategory, bool, error)
	DeleteCategory(context.Context, string) (bool, error)
	ListCategories(context.Context, domain.ProfessionalCategoryFilter) (domain.ProfessionalCategoryPage, error)
	UpsertCatalogEntry(context.Context, domain.CatalogEntry) error
	ListCatalogEntries(context.Context) ([]domain.CatalogEntry, error)
	Stats(context.Context) (domain.CatalogStats, error)
}
```
