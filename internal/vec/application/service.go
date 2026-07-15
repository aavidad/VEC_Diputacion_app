package application

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var ErrServiceDependencyRequired = errors.New("vec service dependency required")

var ErrInternalOperationsMismatch = errors.New("vec internal operations do not belong to service")

const (
	moduleReadPermission = "vec.modules.read"
	menuReadPermission   = "vec.menu.read"
	auditReadPermission  = "vec.audit.read"
)

type Service struct {
	modules ports.ModuleRegistryStore
}

// InternalOperations agrupa exclusivamente operaciones de composicion y de
// infraestructura. No se obtiene desde Service: una frontera externa que solo
// reciba *Service no puede registrar modulos, escribir trazas/eventos ni leer
// auditoria. La raiz de composicion debe entregarlo de forma deliberada.
type InternalOperations struct {
	owner   *Service
	modules ports.ModuleRegistryStore
	audit   ports.AuditStore
	events  ports.EventStore
	now     func() time.Time
}

func NewService(
	modules ports.ModuleRegistryStore,
	audit ports.AuditStore,
	events ports.EventStore,
) (*Service, error) {
	service, _, err := NewServiceWithInternalOperations(modules, audit, events)
	return service, err
}

// NewServiceWithInternalOperations es el constructor reservado a la raiz de
// composicion. Mantiene separada la superficie externa de las operaciones de
// arranque e infraestructura y liga estas ultimas a una unica instancia.
func NewServiceWithInternalOperations(
	modules ports.ModuleRegistryStore,
	audit ports.AuditStore,
	events ports.EventStore,
) (*Service, *InternalOperations, error) {
	if modules == nil || audit == nil || events == nil {
		return nil, nil, ErrServiceDependencyRequired
	}
	service := &Service{modules: modules}
	internal := &InternalOperations{
		owner: service, modules: modules, audit: audit, events: events, now: time.Now,
	}
	return service, internal, nil
}

// Matches impide mezclar por error un Service con operaciones internas de otra
// composicion (y, por tanto, con otros almacenes).
func (o *InternalOperations) Matches(service *Service) bool {
	return o != nil && service != nil && o.owner == service
}

func (o *InternalOperations) RegisterModule(ctx context.Context, manifest domain.ModuleManifest) error {
	if o == nil || o.owner == nil || o.modules == nil {
		return ErrInternalOperationsMismatch
	}
	if err := manifest.Validate(); err != nil {
		return err
	}
	return o.modules.SaveModule(ctx, cloneModuleManifest(manifest))
}

func (s *Service) Modules(ctx context.Context, principal domain.Principal) ([]domain.ModuleManifest, error) {
	if s == nil || s.modules == nil || principal.Validate() != nil ||
		!principal.HasPermission(moduleReadPermission) {
		return nil, domain.ErrPermissionDenied
	}
	modules, err := s.modules.ListModules(ctx)
	if err != nil {
		return nil, err
	}
	resultado := make([]domain.ModuleManifest, len(modules))
	for indice, module := range modules {
		if err := module.Validate(); err != nil {
			return nil, err
		}
		resultado[indice] = cloneModuleManifest(module)
	}
	return resultado, nil
}

func (s *Service) BuildMenu(ctx context.Context, principal domain.Principal) ([]domain.MenuEntry, error) {
	if s == nil || s.modules == nil || principal.Validate() != nil ||
		!principal.HasPermission(menuReadPermission) {
		return nil, domain.ErrPermissionDenied
	}
	modules, err := s.modules.ListModules(ctx)
	if err != nil {
		return nil, err
	}
	entries := make([]domain.MenuEntry, 0)
	for _, module := range modules {
		// El adaptador no es una frontera de confianza: una configuracion
		// incompleta o alterada deniega la construccion de todo el menu.
		if err := module.Validate(); err != nil {
			return nil, err
		}
		for _, entry := range module.Menu {
			if principal.HasAllPermissions(entry.RequiredPermissions) {
				entries = append(entries, cloneMenuEntry(entry))
			}
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Order != entries[j].Order {
			return entries[i].Order < entries[j].Order
		}
		if entries[i].ModuleID != entries[j].ModuleID {
			return entries[i].ModuleID < entries[j].ModuleID
		}
		return entries[i].ID < entries[j].ID
	})
	return entries, nil
}

func cloneModuleManifest(manifest domain.ModuleManifest) domain.ModuleManifest {
	manifest.Permissions = append([]domain.Permission(nil), manifest.Permissions...)
	manifest.Menu = append([]domain.MenuEntry(nil), manifest.Menu...)
	for indice := range manifest.Menu {
		manifest.Menu[indice] = cloneMenuEntry(manifest.Menu[indice])
	}
	return manifest
}

func cloneMenuEntry(entry domain.MenuEntry) domain.MenuEntry {
	entry.RequiredPermissions = append([]string(nil), entry.RequiredPermissions...)
	return entry
}

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

// AuthorizedAuditCommand es una capacidad positiva, concreta e inmutable. Solo
// se crea cuando el principal contiene exactamente el permiso exigido. No es
// un PDP ni crea permisos: consume la evidencia autenticada que ya resolvio la
// frontera y falla cerrado ante ausencia, comodines o valores no canonicos.
type AuthorizedAuditCommand struct {
	command            AuditCommand
	requiredPermission string
	expectedEventType  string
}

func NewAuthorizedAuditCommand(
	command AuditCommand,
	requiredPermission string,
	expectedEventType string,
) (AuthorizedAuditCommand, error) {
	if err := command.Principal.Validate(); err != nil {
		return AuthorizedAuditCommand{}, domain.ErrPermissionDenied
	}
	if !command.Principal.HasPermission(requiredPermission) ||
		strings.TrimSpace(command.Action) == "" || command.Action != strings.TrimSpace(command.Action) ||
		(expectedEventType != "" && expectedEventType != strings.TrimSpace(expectedEventType)) {
		return AuthorizedAuditCommand{}, domain.ErrPermissionDenied
	}
	command.Principal = clonePrincipal(command.Principal)
	command.Metadata = cloneStringMap(command.Metadata)
	return AuthorizedAuditCommand{
		command: command, requiredPermission: requiredPermission, expectedEventType: expectedEventType,
	}, nil
}

// AuditReceipt es la evidencia opaca de una escritura ya confirmada. Un evento
// solo puede publicarse si fue declarado al crear la capacidad y queda ligado
// exactamente al actor, modulo, sujeto e identificador de esa traza.
type AuditReceipt struct {
	owner             *InternalOperations
	entry             domain.AuditEntry
	expectedEventType string
}

func (r AuditReceipt) Entry() domain.AuditEntry {
	return cloneAuditEntry(r.entry)
}

func (o *InternalOperations) RecordAudit(
	ctx context.Context,
	authorized AuthorizedAuditCommand,
) (AuditReceipt, error) {
	if o == nil || o.owner == nil || o.audit == nil || o.now == nil {
		return AuditReceipt{}, ErrInternalOperationsMismatch
	}
	command := authorized.command
	if err := command.Principal.Validate(); err != nil ||
		!command.Principal.HasPermission(authorized.requiredPermission) ||
		authorized.requiredPermission == "" || strings.TrimSpace(command.Action) == "" ||
		command.Action != strings.TrimSpace(command.Action) {
		return AuditReceipt{}, domain.ErrPermissionDenied
	}
	entry := domain.AuditEntry{
		ActorID:              command.Principal.ID,
		ActorProfile:         command.ActorProfile,
		ActorRoles:           append([]string(nil), command.Principal.Roles...),
		RepresentedSubjectID: command.RepresentedSubjectID,
		AuthMethod:           command.Principal.AuthMethod,
		AuthAssurance:        command.Principal.AuthAssurance,
		AuthorizationRef:     command.AuthorizationRef,
		Purpose:              command.Purpose,
		Action:               command.Action,
		ModuleID:             command.ModuleID,
		SubjectRef:           command.SubjectRef,
		ObjectVersion:        command.ObjectVersion,
		ExpedienteRef:        command.ExpedienteRef,
		DocumentRef:          command.DocumentRef,
		RuleRef:              command.RuleRef,
		Reason:               command.Reason,
		Result:               command.Result,
		BeforeHash:           command.BeforeHash,
		AfterHash:            command.AfterHash,
		CorrelationRef:       command.CorrelationRef,
		Metadata:             command.Metadata,
		OccurredAt:           o.now().UTC(),
	}
	stored, err := o.audit.AppendAudit(ctx, entry)
	if err != nil {
		return AuditReceipt{}, err
	}
	return AuditReceipt{
		owner: o, entry: cloneAuditEntry(stored), expectedEventType: authorized.expectedEventType,
	}, nil
}

func (o *InternalOperations) PublishEvent(ctx context.Context, receipt AuditReceipt, event domain.Event) error {
	entry := receipt.entry
	if o == nil || o.owner == nil || o.events == nil || o.now == nil || receipt.owner != o ||
		receipt.expectedEventType == "" || event.Type != receipt.expectedEventType ||
		entry.ID == "" || entry.Signature == "" || event.ModuleID != entry.ModuleID ||
		event.SubjectRef != entry.SubjectRef || event.ActorID != entry.ActorID ||
		event.Payload == nil || event.Payload["audit_id"] != entry.ID {
		return domain.ErrPermissionDenied
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = o.now().UTC()
	}
	return o.events.PublishEvent(ctx, event)
}

// AuditQuery fija el principal y una unica referencia exacta antes de tocar el
// almacen. La lista positiva de permisos contiene solo vec.audit.read.
type AuditQuery struct {
	principal  domain.Principal
	subjectRef string
}

func NewAuditQuery(principal domain.Principal, subjectRef string) (AuditQuery, error) {
	// Una referencia vacia nunca significa "toda la auditoria". Una futura
	// consulta global necesitara un caso de uso y una concesion diferentes, con
	// sus propios ambitos, finalidad, limites y trazabilidad.
	if err := principal.Validate(); err != nil || !principal.HasPermission(auditReadPermission) ||
		!validAuditSubjectRef(subjectRef) {
		return AuditQuery{}, domain.ErrPermissionDenied
	}
	return AuditQuery{principal: clonePrincipal(principal), subjectRef: subjectRef}, nil
}

func (o *InternalOperations) Audit(ctx context.Context, query AuditQuery) ([]domain.AuditEntry, error) {
	if o == nil || o.owner == nil || o.audit == nil ||
		query.principal.Validate() != nil || !query.principal.HasPermission(auditReadPermission) ||
		!validAuditSubjectRef(query.subjectRef) {
		return nil, domain.ErrPermissionDenied
	}
	entries, err := o.audit.ListAudit(ctx, query.subjectRef)
	if err != nil {
		return nil, err
	}
	result := make([]domain.AuditEntry, len(entries))
	for index, entry := range entries {
		result[index] = cloneAuditEntry(entry)
	}
	return result, nil
}

func validAuditSubjectRef(reference string) bool {
	if reference == "" || reference != strings.TrimSpace(reference) || len(reference) > 512 ||
		!utf8.ValidString(reference) || strings.ContainsRune(reference, '*') {
		return false
	}
	for _, character := range reference {
		if unicode.IsControl(character) || unicode.IsSpace(character) {
			return false
		}
	}
	return true
}

func clonePrincipal(principal domain.Principal) domain.Principal {
	principal.Roles = append([]string(nil), principal.Roles...)
	principal.Permissions = append([]string(nil), principal.Permissions...)
	principal.Attributes = cloneStringMap(principal.Attributes)
	return principal
}

func cloneAuditEntry(entry domain.AuditEntry) domain.AuditEntry {
	entry.ActorRoles = append([]string(nil), entry.ActorRoles...)
	entry.Metadata = cloneStringMap(entry.Metadata)
	return entry
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
