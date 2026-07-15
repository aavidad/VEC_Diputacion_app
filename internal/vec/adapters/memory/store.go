package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type Store struct {
	mu                     sync.RWMutex
	modules                map[string]domain.ModuleManifest
	audit                  []domain.AuditEntry
	events                 []domain.Event
	plantillas             map[string]domain.PlantillaDocumento
	catalogos              map[string]domain.CatalogoConfigurable
	definicionesFlujo      map[string]domain.DefinicionFlujo
	instanciasFlujo        map[string]domain.InstanciaFlujo
	instanciasPorEntidad   map[string]string
	decisionesReglaFlujo   map[string]domain.DecisionReglaFlujo
	documentos             map[string]domain.DocumentoGenerado
	usosAutorizacionDoc    map[string]usoAutorizacionDocumento
	usosAutorizacionCat    map[string]usoAutorizacionCatalogo
	contenidos             map[string]objetoContenidoDocumento
	idempotenciasContenido map[string]idempotenciaContenidoDocumento
	documentosLogicos      map[string]domain.DocumentoLogico
	representaciones       map[string]domain.RepresentacionDocumento
	politicasCotejo        map[string]domain.PoliticaCotejo
	codigosCotejo          map[string]domain.CodigoCotejo
	cotejoPorDocumento     map[string]string
	cotejoPorIndice        map[string]string
	reservasDocumentales   map[string]reservaGeneracionDocumento
	reservasPorToken       map[string]string
	reservasCotejo         map[string]reservaEmisionCodigoCotejo
	reservasCotejoToken    map[string]string
	secuenciaReservas      uint64
	secuenciaCotejo        uint64
	secuenciaInstancias    uint64
}

type objetoContenidoDocumento struct {
	MIME  string
	Zona  ports.ZonaAlmacen
	Datos []byte
}

// idempotenciaContenidoDocumento liga una clave a la huella de la solicitud
// completa y a una unica identidad fisica. Solo conserva valores opacos; los
// bytes se custodian en contenidos mediante una copia defensiva separada.
type idempotenciaContenidoDocumento struct {
	HuellaSolicitudSHA256 string
	Referencia            string
}

type estadoReservaGeneracionDocumento string

const (
	estadoReservaDocumentalActiva     estadoReservaGeneracionDocumento = "activa"
	estadoReservaDocumentalConfirmada estadoReservaGeneracionDocumento = "confirmada"
	estadoReservaDocumentalAbandonada estadoReservaGeneracionDocumento = "abandonada"
)

type reservaGeneracionDocumento struct {
	ClaveAmbito         string
	PrincipalID         string
	HuellaSolicitudHMAC string
	Token               string
	Estado              estadoReservaGeneracionDocumento
	ExpiraEn            time.Time
	Resultado           domain.ResultadoGeneracionDocumento
}

type estadoReservaCotejo string

const (
	estadoReservaCotejoActiva     estadoReservaCotejo = "activa"
	estadoReservaCotejoConfirmada estadoReservaCotejo = "confirmada"
	estadoReservaCotejoAbandonada estadoReservaCotejo = "abandonada"
)

type reservaEmisionCodigoCotejo struct {
	ClaveAmbito         string
	PrincipalID         string
	HuellaSolicitudHMAC string
	Documento           domain.ReferenciaDocumento
	Politica            domain.ReferenciaPoliticaCotejo
	Token               string
	Estado              estadoReservaCotejo
	SolicitadaEn        time.Time
	ExpiraEn            time.Time
	Codigo              domain.CodigoCotejo
}

func NewStore() *Store {
	return &Store{
		modules:                map[string]domain.ModuleManifest{},
		plantillas:             map[string]domain.PlantillaDocumento{},
		catalogos:              map[string]domain.CatalogoConfigurable{},
		definicionesFlujo:      map[string]domain.DefinicionFlujo{},
		instanciasFlujo:        map[string]domain.InstanciaFlujo{},
		instanciasPorEntidad:   map[string]string{},
		decisionesReglaFlujo:   map[string]domain.DecisionReglaFlujo{},
		documentos:             map[string]domain.DocumentoGenerado{},
		usosAutorizacionDoc:    map[string]usoAutorizacionDocumento{},
		usosAutorizacionCat:    map[string]usoAutorizacionCatalogo{},
		contenidos:             map[string]objetoContenidoDocumento{},
		idempotenciasContenido: map[string]idempotenciaContenidoDocumento{},
		documentosLogicos:      map[string]domain.DocumentoLogico{},
		representaciones:       map[string]domain.RepresentacionDocumento{},
		politicasCotejo:        map[string]domain.PoliticaCotejo{},
		codigosCotejo:          map[string]domain.CodigoCotejo{},
		cotejoPorDocumento:     map[string]string{},
		cotejoPorIndice:        map[string]string{},
		reservasDocumentales:   map[string]reservaGeneracionDocumento{},
		reservasPorToken:       map[string]string{},
		reservasCotejo:         map[string]reservaEmisionCodigoCotejo{},
		reservasCotejoToken:    map[string]string{},
	}
}

func (s *Store) SaveModule(ctx context.Context, manifest domain.ModuleManifest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := manifest.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	s.modules[manifest.ID] = cloneModuleManifest(manifest)
	return nil
}

func (s *Store) ListModules(ctx context.Context) ([]domain.ModuleManifest, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	modules := make([]domain.ModuleManifest, 0, len(s.modules))
	for _, module := range s.modules {
		modules = append(modules, cloneModuleManifest(module))
	}
	sort.SliceStable(modules, func(i, j int) bool { return modules[i].ID < modules[j].ID })
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return modules, nil
}

func cloneModuleManifest(manifest domain.ModuleManifest) domain.ModuleManifest {
	manifest.Permissions = append([]domain.Permission(nil), manifest.Permissions...)
	manifest.Menu = append([]domain.MenuEntry(nil), manifest.Menu...)
	for indice := range manifest.Menu {
		manifest.Menu[indice].RequiredPermissions = append(
			[]string(nil),
			manifest.Menu[indice].RequiredPermissions...,
		)
	}
	return manifest
}

func (s *Store) AppendAudit(ctx context.Context, entry domain.AuditEntry) (domain.AuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return domain.AuditEntry{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return domain.AuditEntry{}, err
	}
	return s.appendAuditLocked(entry), nil
}

func (s *Store) appendAuditLocked(entry domain.AuditEntry) domain.AuditEntry {
	entry = cloneAuditEntry(entry)
	entry.Seq = int64(len(s.audit) + 1)
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("audit-%06d", entry.Seq)
	}
	if entry.OccurredAt.IsZero() {
		entry.OccurredAt = time.Now().UTC()
	}
	if len(s.audit) > 0 {
		entry.PrevSignature = s.audit[len(s.audit)-1].Signature
	}
	entry.IntegrityAlgorithm = "sha256-chain-v1"
	entry.Signature = auditSignature(entry)
	s.audit = append(s.audit, cloneAuditEntry(entry))
	return cloneAuditEntry(entry)
}

func (s *Store) ListAudit(ctx context.Context, subjectRef string) ([]domain.AuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if subjectRef == "" || subjectRef != strings.TrimSpace(subjectRef) {
		return nil, domain.ErrPermissionDenied
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result := make([]domain.AuditEntry, 0, len(s.audit))
	for _, entry := range s.audit {
		if entry.SubjectRef == subjectRef {
			result = append(result, cloneAuditEntry(entry))
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Store) PublishEvent(ctx context.Context, event domain.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	s.appendEventLocked(event)
	return nil
}

func (s *Store) appendEventLocked(event domain.Event) domain.Event {
	event = cloneEvent(event)
	if event.ID == "" {
		event.ID = fmt.Sprintf("event-%06d", len(s.events)+1)
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	s.events = append(s.events, cloneEvent(event))
	return cloneEvent(event)
}

func (s *Store) ListEvents(ctx context.Context, types []string) ([]domain.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// El conjunto vacio no es el comodin "todos". Los consumidores enumeran
	// positivamente cada tipo y una consulta global debera tener otro contrato.
	if len(types) == 0 {
		return nil, domain.ErrPermissionDenied
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	allowed := map[string]bool{}
	for _, eventType := range types {
		if eventType == "" || eventType != strings.TrimSpace(eventType) || strings.ContainsRune(eventType, '*') || allowed[eventType] {
			return nil, domain.ErrPermissionDenied
		}
		allowed[eventType] = true
	}
	result := make([]domain.Event, 0, len(s.events))
	for _, event := range s.events {
		if allowed[event.Type] {
			result = append(result, cloneEvent(event))
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func auditSignature(entry domain.AuditEntry) string {
	entry.Signature = ""
	canonico, _ := json.Marshal(entry)
	sum := sha256.Sum256(canonico)
	return hex.EncodeToString(sum[:])
}

func cloneAuditEntry(entry domain.AuditEntry) domain.AuditEntry {
	entry.ActorRoles = append([]string(nil), entry.ActorRoles...)
	entry.Metadata = cloneStringMap(entry.Metadata)
	return entry
}

func cloneEvent(event domain.Event) domain.Event {
	event.Payload = cloneStringMap(event.Payload)
	return event
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	destination := make(map[string]string, len(source))
	for key, value := range source {
		destination[key] = value
	}
	return destination
}
