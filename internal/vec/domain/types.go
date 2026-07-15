package domain

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrPrincipalInvalid = errors.New("vec principal invalid")
	ErrModuleInvalid    = errors.New("vec module invalid")
	ErrMenuEntryInvalid = errors.New("vec menu entry invalid")
	ErrPermissionDenied = errors.New("vec permission denied")
)

type AuthMethod string

const (
	AuthMethodCertificate AuthMethod = "certificado"
	AuthMethodDNIe        AuthMethod = "dnie"
	AuthMethodSSO         AuthMethod = "sso"
	AuthMethodClave       AuthMethod = "clave"
	AuthMethodKerberos    AuthMethod = "kerberos_ad"
	AuthMethodDemo        AuthMethod = "demo"
)

type AuthAssurance string

const (
	AuthAssuranceLow         AuthAssurance = "bajo"
	AuthAssuranceSubstantial AuthAssurance = "sustancial"
	AuthAssuranceHigh        AuthAssurance = "alto"
)

func (a AuthAssurance) Valida() bool {
	return a == AuthAssuranceLow || a == AuthAssuranceSubstantial || a == AuthAssuranceHigh
}

func (a AuthAssurance) Cumple(minima AuthAssurance) bool {
	nivel := map[AuthAssurance]int{
		AuthAssuranceLow:         1,
		AuthAssuranceSubstantial: 2,
		AuthAssuranceHigh:        3,
	}
	actual, actualValido := nivel[a]
	requerido, requeridoValido := nivel[minima]
	return actualValido && requeridoValido && actual >= requerido
}

// CumpleGarantiaAutenticacion conserva el contrato funcional usado por los
// adaptadores, pero pertenece al núcleo de identidad y no a un módulo de
// autorización o documentos concreto.
func CumpleGarantiaAutenticacion(actual, minima AuthAssurance) bool {
	return actual.Cumple(minima)
}

type Principal struct {
	ID            string            `json:"id"`
	DisplayName   string            `json:"display_name"`
	Email         string            `json:"email,omitempty"`
	Roles         []string          `json:"roles"`
	Permissions   []string          `json:"permissions"`
	AuthMethod    AuthMethod        `json:"auth_method"`
	AuthAssurance AuthAssurance     `json:"auth_assurance"`
	Attributes    map[string]string `json:"attributes,omitempty"`
}

func (p Principal) Validate() error {
	if !textoPrincipalSeguro(p.ID, 512, false) ||
		(p.DisplayName != "" && !textoPrincipalSeguro(p.DisplayName, 512, true)) ||
		(p.Email != "" && !textoPrincipalSeguro(p.Email, 512, false)) ||
		len(p.Roles) > 64 || len(p.Permissions) > 512 || len(p.Attributes) > 128 {
		return ErrPrincipalInvalid
	}
	if !p.AuthMethod.Valido() {
		return ErrPrincipalInvalid
	}
	if !p.AuthAssurance.Valida() {
		return ErrPrincipalInvalid
	}
	roles := make(map[string]struct{}, len(p.Roles))
	for _, rol := range p.Roles {
		if !identificadorManifestSeguro(rol, 128) || strings.ContainsRune(rol, '*') {
			return ErrPrincipalInvalid
		}
		if _, repetido := roles[rol]; repetido {
			return ErrPrincipalInvalid
		}
		roles[rol] = struct{}{}
	}
	permisos := make(map[string]struct{}, len(p.Permissions))
	for _, permiso := range p.Permissions {
		if !permisoManifestConcreto(permiso) {
			return ErrPrincipalInvalid
		}
		if _, repetido := permisos[permiso]; repetido {
			return ErrPrincipalInvalid
		}
		permisos[permiso] = struct{}{}
	}
	for clave, valor := range p.Attributes {
		if !identificadorManifestSeguro(clave, 128) || !textoPrincipalSeguro(valor, 1024, true) {
			return ErrPrincipalInvalid
		}
	}
	return nil
}

func (a AuthMethod) Valido() bool {
	switch a {
	case AuthMethodCertificate, AuthMethodDNIe, AuthMethodSSO, AuthMethodClave,
		AuthMethodKerberos, AuthMethodDemo:
		return true
	default:
		return false
	}
}

func (p Principal) HasPermission(permission string) bool {
	// Ni el principal ni el permiso solicitado se normalizan para conceder. Una
	// identidad o configuracion no canonica pierde todas sus capacidades.
	if p.Validate() != nil || !permisoManifestConcreto(permission) {
		return false
	}
	for _, candidate := range p.Permissions {
		if candidate == permission && permisoManifestConcreto(candidate) {
			return true
		}
	}
	return false
}

func (p Principal) HasAllPermissions(permissions []string) bool {
	// Una lista vacia nunca significa acceso publico. Cada superficie debe
	// declarar al menos un permiso positivo y concreto.
	if len(permissions) == 0 {
		return false
	}
	for _, permission := range permissions {
		if !p.HasPermission(permission) {
			return false
		}
	}
	return true
}

type Permission struct {
	Key         string `json:"key"`
	LabelKey    string `json:"label_key"`
	Description string `json:"description,omitempty"`
}

type ModuleManifest struct {
	ID             string       `json:"id"`
	NameKey        string       `json:"name_key"`
	DescriptionKey string       `json:"description_key"`
	Version        string       `json:"version"`
	Group          string       `json:"group"`
	BasePath       string       `json:"base_path"`
	Permissions    []Permission `json:"permissions"`
	Menu           []MenuEntry  `json:"menu"`
}

func (m ModuleManifest) Validate() error {
	if !identificadorManifestSeguro(m.ID, 128) || !identificadorManifestSeguro(m.NameKey, 256) ||
		len(m.Permissions) == 0 || len(m.Permissions) > 512 || len(m.Menu) > 512 {
		return ErrModuleInvalid
	}

	permisosDeclarados := make(map[string]struct{}, len(m.Permissions))
	for _, permiso := range m.Permissions {
		if !permisoManifestConcreto(permiso.Key) || !identificadorManifestSeguro(permiso.LabelKey, 256) {
			return ErrModuleInvalid
		}
		if _, repetido := permisosDeclarados[permiso.Key]; repetido {
			return ErrModuleInvalid
		}
		permisosDeclarados[permiso.Key] = struct{}{}
	}

	identificadoresMenu := make(map[string]struct{}, len(m.Menu))
	rutasMenu := make(map[string]struct{}, len(m.Menu))
	for _, entry := range m.Menu {
		if err := entry.Validate(); err != nil {
			return err
		}
		if entry.ModuleID != m.ID {
			return ErrMenuEntryInvalid
		}
		if _, repetido := identificadoresMenu[entry.ID]; repetido {
			return ErrMenuEntryInvalid
		}
		if _, repetida := rutasMenu[entry.Path]; repetida {
			return ErrMenuEntryInvalid
		}
		identificadoresMenu[entry.ID] = struct{}{}
		rutasMenu[entry.Path] = struct{}{}
		for _, permiso := range entry.RequiredPermissions {
			if _, declarado := permisosDeclarados[permiso]; !declarado {
				return ErrMenuEntryInvalid
			}
		}
	}
	return nil
}

type MenuEntry struct {
	ID                  string   `json:"id"`
	ModuleID            string   `json:"module_id"`
	LabelKey            string   `json:"label_key"`
	Path                string   `json:"path"`
	Icon                string   `json:"icon"`
	Group               string   `json:"group"`
	Order               int      `json:"order"`
	RequiredPermissions []string `json:"required_permissions,omitempty"`
}

func (m MenuEntry) Validate() error {
	if !identificadorManifestSeguro(m.ID, 128) || !identificadorManifestSeguro(m.ModuleID, 128) ||
		!identificadorManifestSeguro(m.LabelKey, 256) || !rutaMenuInternaSegura(m.Path) ||
		len(m.RequiredPermissions) == 0 || len(m.RequiredPermissions) > 32 {
		return ErrMenuEntryInvalid
	}
	permisos := make(map[string]struct{}, len(m.RequiredPermissions))
	for _, permiso := range m.RequiredPermissions {
		if !permisoManifestConcreto(permiso) {
			return ErrMenuEntryInvalid
		}
		if _, repetido := permisos[permiso]; repetido {
			return ErrMenuEntryInvalid
		}
		permisos[permiso] = struct{}{}
	}
	return nil
}

func identificadorManifestSeguro(valor string, maximoBytes int) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > maximoBytes || !utf8.ValidString(valor) {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || unicode.IsSpace(caracter) {
			return false
		}
	}
	return true
}

func permisoManifestConcreto(permiso string) bool {
	return identificadorManifestSeguro(permiso, 256) && !strings.ContainsRune(permiso, '*')
}

func textoPrincipalSeguro(valor string, maximoBytes int, permiteEspacios bool) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > maximoBytes || !utf8.ValidString(valor) {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || (!permiteEspacios && unicode.IsSpace(caracter)) {
			return false
		}
	}
	return true
}

func rutaMenuInternaSegura(ruta string) bool {
	if !identificadorManifestSeguro(ruta, 512) || !strings.HasPrefix(ruta, "/") || strings.HasPrefix(ruta, "//") {
		return false
	}
	return !strings.ContainsAny(ruta, "?#\\\\")
}

type Event struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	ModuleID   string            `json:"module_id"`
	SubjectRef string            `json:"subject_ref"`
	ActorID    string            `json:"actor_id"`
	Payload    map[string]string `json:"payload,omitempty"`
	OccurredAt time.Time         `json:"occurred_at"`
}

type AuditEntry struct {
	ID                   string            `json:"id"`
	Seq                  int64             `json:"seq"`
	ActorID              string            `json:"actor_id"`
	ActorProfile         string            `json:"actor_profile,omitempty"`
	ActorRoles           []string          `json:"actor_roles,omitempty"`
	RepresentedSubjectID string            `json:"represented_subject_id,omitempty"`
	AuthMethod           AuthMethod        `json:"auth_method,omitempty"`
	AuthAssurance        AuthAssurance     `json:"auth_assurance,omitempty"`
	AuthorizationRef     string            `json:"authorization_ref,omitempty"`
	Purpose              string            `json:"purpose,omitempty"`
	Action               string            `json:"action"`
	ModuleID             string            `json:"module_id"`
	SubjectRef           string            `json:"subject_ref"`
	ObjectVersion        int               `json:"object_version,omitempty"`
	ExpedienteRef        string            `json:"expediente_ref,omitempty"`
	DocumentRef          string            `json:"document_ref,omitempty"`
	RuleRef              string            `json:"rule_ref,omitempty"`
	Reason               string            `json:"reason,omitempty"`
	Result               string            `json:"result"`
	BeforeHash           string            `json:"before_hash,omitempty"`
	AfterHash            string            `json:"after_hash,omitempty"`
	CorrelationRef       string            `json:"correlation_ref,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
	OccurredAt           time.Time         `json:"occurred_at"`
	IntegrityAlgorithm   string            `json:"integrity_algorithm,omitempty"`
	PrevSignature        string            `json:"prev_signature,omitempty"`
	Signature            string            `json:"signature"`
}

type AuthChallenge struct {
	ID        string    `json:"id"`
	Nonce     string    `json:"nonce"`
	ExpiresAt time.Time `json:"expires_at"`
}

type AuthChallengeResponse struct {
	ChallengeID string `json:"challenge_id"`
	Certificate string `json:"certificate"`
	Signature   string `json:"signature"`
}

type SignRequest struct {
	DocumentRef string `json:"document_ref"`
	Purpose     string `json:"purpose"`
}

type SignReceipt struct {
	ReceiptRef  string    `json:"receipt_ref"`
	DocumentRef string    `json:"document_ref"`
	SignedAt    time.Time `json:"signed_at"`
}

type SignVerification struct {
	DocumentRef string    `json:"document_ref"`
	Valid       bool      `json:"valid"`
	CheckedAt   time.Time `json:"checked_at"`
}
