package domain

import (
	"errors"
	"strings"
	"time"
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
	if strings.TrimSpace(p.ID) == "" {
		return ErrPrincipalInvalid
	}
	if strings.TrimSpace(string(p.AuthMethod)) == "" {
		return ErrPrincipalInvalid
	}
	if strings.TrimSpace(string(p.AuthAssurance)) == "" {
		return ErrPrincipalInvalid
	}
	return nil
}

func (p Principal) HasPermission(permission string) bool {
	permission = strings.TrimSpace(permission)
	if permission == "" {
		return true
	}
	for _, candidate := range p.Permissions {
		if strings.TrimSpace(candidate) == permission {
			return true
		}
	}
	return false
}

func (p Principal) HasAllPermissions(permissions []string) bool {
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
	if strings.TrimSpace(m.ID) == "" || strings.TrimSpace(m.NameKey) == "" {
		return ErrModuleInvalid
	}
	for _, entry := range m.Menu {
		if err := entry.Validate(); err != nil {
			return err
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
	if strings.TrimSpace(m.ID) == "" || strings.TrimSpace(m.ModuleID) == "" ||
		strings.TrimSpace(m.LabelKey) == "" || strings.TrimSpace(m.Path) == "" {
		return ErrMenuEntryInvalid
	}
	return nil
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
	ID            string            `json:"id"`
	Seq           int64             `json:"seq"`
	ActorID       string            `json:"actor_id"`
	Action        string            `json:"action"`
	ModuleID      string            `json:"module_id"`
	SubjectRef    string            `json:"subject_ref"`
	Result        string            `json:"result"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	OccurredAt    time.Time         `json:"occurred_at"`
	PrevSignature string            `json:"prev_signature,omitempty"`
	Signature     string            `json:"signature"`
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
