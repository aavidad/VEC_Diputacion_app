package main

import (
	"time"
	bolsadomain "vec-diputacion-granada/internal/modules/bolsa/domain"
	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

// Provision belongs to clone direction. This command never installs functions,
// rewrites government, grants privileges or fabricates a context/decision record.
type configuration struct {
	OrganizacionFuente string                                             `json:"organizacion_fuente"`
	UnidadFuente       string                                             `json:"unidad_fuente"`
	VersionFuente      bolsadomain.DatosNuevaVersionConvocatoriaGobernada `json:"version_fuente"`
	Mode               string                                             `json:"mode"`
	Crypto             cryptoConfig                                       `json:"crypto"`
	Actors             map[string]actorConfig                             `json:"actors"`
	Operations         []operation                                        `json:"operations"`
	Keys               map[string]capabilityKey                           `json:"keys"`
	RBAC               []rbacConfig                                       `json:"rbac"`
}
type rbacConfig struct {
	Name       string                       `json:"name"`
	Role       vd.VersionRol                `json:"role"`
	Control    vd.ControlVigenciaVersionRol `json:"control"`
	Assignment vd.AsignacionPerfil          `json:"assignment"`
}

// Keys are a closed startup catalogue by consumption audience. Supplying them
// does not publish PostgreSQL government or change any profile or grant.
type capabilityKey struct {
	ID                 string    `json:"id"`
	Version            uint64    `json:"version"`
	HMAC               []byte    `json:"hmac"`
	Issuer             string    `json:"issuer"`
	GovernmentRevision uint64    `json:"government_revision"`
	GovernmentSHA      string    `json:"government_sha"`
	From               time.Time `json:"from"`
	Until              time.Time `json:"until"`
}
type cryptoConfig struct {
	Seed               []byte    `json:"seed"`
	HMAC               []byte    `json:"hmac"`
	RootID             string    `json:"root_id"`
	RootVersion        uint64    `json:"root_version"`
	Deployment         string    `json:"deployment"`
	Revision           string    `json:"revision"`
	Sequence           uint64    `json:"sequence"`
	Published          time.Time `json:"published"`
	Expires            time.Time `json:"expires"`
	RootFrom           time.Time `json:"root_from"`
	RootUntil          time.Time `json:"root_until"`
	KeyID              string    `json:"key_id"`
	KeyVersion         uint64    `json:"key_version"`
	Issuer             string    `json:"issuer"`
	GovernmentRevision uint64    `json:"government_revision"`
	GovernmentSHA      string    `json:"government_sha"`
	KeyFrom            time.Time `json:"key_from"`
	KeyUntil           time.Time `json:"key_until"`
}
type actorConfig struct {
	Account           string `json:"account"`
	Profile           string `json:"profile"`
	Authentication    string `json:"authentication"`
	Session           string `json:"session"`
	ContextLogin      string `json:"context_login"`
	RevalidationLogin string `json:"revalidation_login"`
	SourceLogin       string `json:"source_login"`
	RegisterLogin     string `json:"register_login"`
	ReasonLogin       string `json:"reason_login"`
	RuntimeLogin      string `json:"runtime_login"`
}
type operation struct {
	Name     string                                       `json:"name"`
	Actor    string                                       `json:"actor"`
	Selector bolsaports.SelectorVersionConvocatoriaExacta `json:"selector"`
	Motivo   vd.ReferenciaEntradaCatalogo                 `json:"motivo"`
	Expected string                                       `json:"expected"`
}
