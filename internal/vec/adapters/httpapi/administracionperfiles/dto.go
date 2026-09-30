package administracionperfiles

import (
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

const PrefijoV1 = "/api/admin/perfiles/v1"

// Los modelos de lectura son proyecciones mínimas de una fuente ya autorizada.
// Ninguno contiene concesiones o definiciones de permiso.
type Capacidades struct {
	Version         string   `json:"version"`
	ActorPersonaRef string   `json:"actor_persona_ref"`
	Acciones        []string `json:"acciones"`
}

type Persona struct {
	PersonaRef      string `json:"persona_ref"`
	Nombre          string `json:"nombre"`
	UnidadNombre    string `json:"unidad_nombre"`
	UnidadClaveI18N string `json:"unidad_clave_i18n"`
}

type PaginaPersonas struct {
	Personas        []Persona `json:"personas"`
	SiguienteCursor string    `json:"siguiente_cursor,omitempty"`
}

type Perfil struct {
	PerfilRef     string    `json:"perfil_ref"`
	RolVersionRef string    `json:"rol_version_ref"`
	Estado        string    `json:"estado"`
	Version       uint64    `json:"version"`
	VigenteHasta  time.Time `json:"vigente_hasta"`
}

type Rol struct {
	VersionRef   string `json:"version_ref"`
	Clase        string `json:"clase"`
	ClaveI18N    string `json:"clave_i18n"`
	Etiqueta     string `json:"etiqueta"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type Roles struct {
	Roles []Rol `json:"roles"`
}

type Motivo struct {
	CatalogoID           string `json:"catalogo_id"`
	CatalogoVersion      int    `json:"catalogo_version"`
	CatalogoHuellaSHA256 string `json:"catalogo_huella_sha256"`
	EntradaClave         string `json:"entrada_clave"`
}

// Etiqueta procede del catálogo autorizado y localizado, solo en lecturas.
// El POST acepta únicamente las cuatro referencias de Motivo.
type MotivoLectura struct {
	Motivo
	Etiqueta string `json:"etiqueta"`
}

func (m Motivo) dominio() domain.ReferenciaEntradaCatalogo {
	return domain.ReferenciaEntradaCatalogo{CatalogoID: m.CatalogoID, CatalogoVersion: m.CatalogoVersion,
		CatalogoHuellaSHA256: m.CatalogoHuellaSHA256, EntradaClave: m.EntradaClave}
}

type Objetivo struct {
	CuentaRef               string    `json:"cuenta_ref"`
	CuentaVersion           uint64    `json:"cuenta_version"`
	PersonaRef              string    `json:"persona_ref"`
	PersonaVersion          uint64    `json:"persona_version"`
	PerfilRef               string    `json:"perfil_ref"`
	PerfilVersion           uint64    `json:"perfil_version"`
	VinculoRef              string    `json:"vinculo_ref"`
	VinculoVersion          uint64    `json:"vinculo_version"`
	HuellaSHA256            string    `json:"huella_sha256"`
	RevisionContinuidad     uint64    `json:"revision_continuidad"`
	ProcedenciaRef          string    `json:"procedencia_ref"`
	ProcedenciaVersion      uint64    `json:"procedencia_version"`
	ProcedenciaHuellaSHA256 string    `json:"procedencia_huella_sha256"`
	VigenteHasta            time.Time `json:"vigente_hasta"`
}

func (o Objetivo) dominio() domain.PreimagenAdministracionPerfiles {
	return domain.PreimagenAdministracionPerfiles{CuentaRef: o.CuentaRef, CuentaVersion: o.CuentaVersion,
		PersonaRef: o.PersonaRef, PersonaVersion: o.PersonaVersion, PerfilRef: o.PerfilRef,
		PerfilVersion: o.PerfilVersion, VinculoRef: o.VinculoRef, VinculoVersion: o.VinculoVersion,
		HuellaSHA256: o.HuellaSHA256, RevisionContinuidad: o.RevisionContinuidad,
		ProcedenciaRef: o.ProcedenciaRef, ProcedenciaVersion: o.ProcedenciaVersion,
		ProcedenciaHuellaSHA256: o.ProcedenciaHuellaSHA256, VigenteHasta: o.VigenteHasta}
}

type ActoDisponible struct {
	Operacion     string          `json:"operacion"`
	RolVersionRef string          `json:"rol_version_ref"`
	Objetivo      Objetivo        `json:"objetivo"`
	Motivos       []MotivoLectura `json:"motivos"`
}

type Historia struct {
	ActoRef      string    `json:"acto_ref"`
	Operacion    string    `json:"operacion"`
	Estado       string    `json:"estado"`
	ConfirmadoEn time.Time `json:"confirmado_en"`
}

type FichaPersona struct {
	PersonaRef       string           `json:"persona_ref"`
	Nombre           string           `json:"nombre"`
	UnidadNombre     string           `json:"unidad_nombre"`
	Perfiles         []Perfil         `json:"perfiles"`
	ActosDisponibles []ActoDisponible `json:"actos_disponibles"`
	Historia         []Historia       `json:"historia"`
}

type Propuesta struct {
	PropuestaRef         string          `json:"propuesta_ref"`
	ProponentePersonaRef string          `json:"proponente_persona_ref"`
	ObjetivoPersonaRef   string          `json:"objetivo_persona_ref"`
	ObjetivoNombre       string          `json:"objetivo_nombre"`
	RolVersionRef        string          `json:"rol_version_ref"`
	Operacion            string          `json:"operacion"`
	HuellaSHA256         string          `json:"huella_sha256"`
	CaducaEn             time.Time       `json:"caduca_en"`
	PuedeCerrar          bool            `json:"puede_cerrar"`
	MotivosCierre        []MotivoLectura `json:"motivos_cierre"`
}

type PaginaPropuestas struct {
	Propuestas []Propuesta `json:"propuestas"`
}

type Recibo struct {
	OperacionRef        string    `json:"operacion_ref"`
	ActoRef             string    `json:"acto_ref"`
	ReciboRef           string    `json:"recibo_ref"`
	PropuestaRef        string    `json:"propuesta_ref,omitempty"`
	AuditoriaRef        string    `json:"auditoria_ref"`
	ObjetivoPersonaRef  string    `json:"objetivo_persona_ref"`
	PerfilRef           string    `json:"perfil_ref"`
	VinculoRef          string    `json:"vinculo_ref"`
	EstadoPosterior     string    `json:"estado_posterior"`
	VersionPosterior    uint64    `json:"version_posterior"`
	HuellaAntesSHA256   string    `json:"huella_antes_sha256"`
	HuellaDespuesSHA256 string    `json:"huella_despues_sha256"`
	ConfirmadoEn        time.Time `json:"confirmado_en"`
}

func reciboDTO(r domain.ReciboAdministracionPerfiles) Recibo {
	return Recibo{OperacionRef: r.OperacionRef, ActoRef: r.ActoRef, ReciboRef: r.ReciboRef,
		PropuestaRef: r.PropuestaRef, AuditoriaRef: r.AuditoriaRef,
		ObjetivoPersonaRef: r.ObjetivoPersonaRef, PerfilRef: r.PerfilRef, VinculoRef: r.VinculoRef,
		EstadoPosterior: string(r.EstadoPosterior), VersionPosterior: r.VersionPosterior,
		HuellaAntesSHA256: r.HuellaAntesSHA256, HuellaDespuesSHA256: r.HuellaDespuesSHA256,
		ConfirmadoEn: r.ConfirmadoEn}
}

type SolicitudActo struct {
	OperacionRef  string   `json:"operacion_ref"`
	Operacion     string   `json:"operacion"`
	RolVersionRef string   `json:"rol_version_ref"`
	Objetivo      Objetivo `json:"objetivo"`
	Motivo        Motivo   `json:"motivo"`
}

type SolicitudCierre struct {
	OperacionRef          string `json:"operacion_ref"`
	PropuestaHuellaSHA256 string `json:"propuesta_huella_sha256"`
	Decision              string `json:"decision"`
	Motivo                Motivo `json:"motivo"`
}
