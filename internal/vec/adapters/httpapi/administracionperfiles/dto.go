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

// ConsultaPersonas se aplica en la fuente autorizada antes de paginar.
// La consulta vacía requiere permiso propio para el conjunto gobernado.
type ConsultaPersonas struct {
	Texto     string
	Cursor    string
	PerfilRef string
	UnidadRef string
	Estado    string
}

type Unidad struct {
	UnidadRef string `json:"unidad_ref"`
	Nombre    string `json:"nombre"`
	ClaveI18N string `json:"clave_i18n"`
}

type AmbitoPerfil struct {
	Dimension  string `json:"dimension"`
	Referencia string `json:"referencia"`
	Nombre     string `json:"nombre"`
	ClaveI18N  string `json:"clave_i18n"`
}

type PerfilResumen struct {
	PerfilRef     string    `json:"perfil_ref"`
	RolVersionRef string    `json:"rol_version_ref"`
	RolClaveI18N  string    `json:"rol_clave_i18n"`
	RolEtiqueta   string    `json:"rol_etiqueta"`
	Estado        string    `json:"estado"`
	VigenteDesde  time.Time `json:"vigente_desde"`
	VigenteHasta  time.Time `json:"vigente_hasta"`
}

type Persona struct {
	PersonaRef      string          `json:"persona_ref"`
	UnidadRef       string          `json:"unidad_ref"`
	Perfiles        []PerfilResumen `json:"perfiles"`
	Nombre          string          `json:"nombre"`
	UnidadNombre    string          `json:"unidad_nombre"`
	UnidadClaveI18N string          `json:"unidad_clave_i18n"`
}

type PaginaPersonas struct {
	Metadatos       *PaginaPersonasMetadatos `json:"-"`
	Personas        []Persona                `json:"personas"`
	SiguienteCursor string                   `json:"siguiente_cursor,omitempty"`
}

type Perfil struct {
	RolClaveI18N   string         `json:"rol_clave_i18n"`
	RolEtiqueta    string         `json:"rol_etiqueta"`
	Ambitos        []AmbitoPerfil `json:"ambitos"`
	AmbitoEtiqueta string         `json:"ambito_etiqueta"`
	VigenteDesde   time.Time      `json:"vigente_desde"`
	PerfilRef      string         `json:"perfil_ref"`
	RolVersionRef  string         `json:"rol_version_ref"`
	Estado         string         `json:"estado"`
	Version        uint64         `json:"version"`
	VigenteHasta   time.Time      `json:"vigente_hasta"`
}

type Rol struct {
	Fijo         bool   `json:"fijo"`
	VersionRef   string `json:"version_ref"`
	Clase        string `json:"clase"`
	ClaveI18N    string `json:"clave_i18n"`
	Etiqueta     string `json:"etiqueta"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type Roles struct {
	Roles    []Rol    `json:"roles"`
	Unidades []Unidad `json:"unidades"`
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
	CentroRef               string    `json:"centro_ref,omitempty"`
	VigenteDesde            time.Time `json:"vigente_desde"`
	UnidadRef               string    `json:"unidad_ref,omitempty"`
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
	return domain.PreimagenAdministracionPerfiles{CentroRef: o.CentroRef, VigenteDesde: o.VigenteDesde, UnidadRef: o.UnidadRef, CuentaRef: o.CuentaRef, CuentaVersion: o.CuentaVersion,
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

// IdentidadHistoria es una proyección minimizada, resuelta por la fuente
// central; no contiene cuentas, certificados ni datos de contacto.
type IdentidadHistoria struct {
	PerfilActivoNombre    string `json:"perfil_activo_nombre"`
	PerfilActivoClaveI18N string `json:"perfil_activo_clave_i18n"`
	PersonaRef            string `json:"persona_ref"`
	Nombre                string `json:"nombre"`
	PerfilActivoRef       string `json:"perfil_activo_ref"`
	AsignacionRef         string `json:"asignacion_ref"`
}

type Historia struct {
	Actor              IdentidadHistoria  `json:"actor"`
	Proponente         *IdentidadHistoria `json:"proponente,omitempty"`
	Aprobador          *IdentidadHistoria `json:"aprobador,omitempty"`
	ObjetivoPersonaRef string             `json:"objetivo_persona_ref"`
	ObjetivoNombre     string             `json:"objetivo_nombre"`
	PerfilRef          string             `json:"perfil_ref"`
	RolVersionRef      string             `json:"rol_version_ref"`
	Ambitos            []AmbitoPerfil     `json:"ambitos"`
	VigenteDesde       time.Time          `json:"vigente_desde"`
	VigenteHasta       time.Time          `json:"vigente_hasta"`
	ReferenciaActo     string             `json:"referencia_acto,omitempty"`
	Motivo             MotivoLectura      `json:"motivo"`
	ReciboRef          string             `json:"recibo_ref"`
	ActoRef            string             `json:"acto_ref"`
	Operacion          string             `json:"operacion"`
	Estado             string             `json:"estado"`
	ConfirmadoEn       time.Time          `json:"confirmado_en"`
}

type FichaPersona struct {
	Metadatos        *FichaPersonaMetadatos `json:"-"`
	UnidadRef        string                 `json:"unidad_ref"`
	UnidadClaveI18N  string                 `json:"unidad_clave_i18n"`
	PersonaRef       string                 `json:"persona_ref"`
	Nombre           string                 `json:"nombre"`
	UnidadNombre     string                 `json:"unidad_nombre"`
	Perfiles         []Perfil               `json:"perfiles"`
	ActosDisponibles []ActoDisponible       `json:"actos_disponibles"`
	Historia         []Historia             `json:"historia"`
}

type Propuesta struct {
	// Estos datos opcionales solo los resuelve FuenteLecturas desde la propuesta
	// conservada y el catálogo autorizado. Su ausencia no acredita el material.
	ProponenteNombre       string          `json:"proponente_nombre,omitempty"`
	ProponentePerfilNombre string          `json:"proponente_perfil_nombre,omitempty"`
	Ambitos                []AmbitoPerfil  `json:"ambitos,omitempty"`
	VigenteDesde           time.Time       `json:"vigente_desde,omitzero"`
	VigenteHasta           time.Time       `json:"vigente_hasta,omitzero"`
	Motivo                 *MotivoLectura  `json:"motivo,omitempty"`
	PropuestaRef           string          `json:"propuesta_ref"`
	ProponentePersonaRef   string          `json:"proponente_persona_ref"`
	ObjetivoPersonaRef     string          `json:"objetivo_persona_ref"`
	ObjetivoNombre         string          `json:"objetivo_nombre"`
	RolVersionRef          string          `json:"rol_version_ref"`
	Operacion              string          `json:"operacion"`
	HuellaSHA256           string          `json:"huella_sha256"`
	CaducaEn               time.Time       `json:"caduca_en"`
	PuedeCerrar            bool            `json:"puede_cerrar"`
	MotivosCierre          []MotivoLectura `json:"motivos_cierre"`
}

type PaginaPropuestas struct {
	Propuestas []Propuesta `json:"propuestas"`
}

type Recibo struct {
	CentroRef           string    `json:"centro_ref,omitempty"`
	ActorPersonaRef     string    `json:"actor_persona_ref"`
	PerfilActivoRef     string    `json:"perfil_activo_ref"`
	AsignacionPerfilRef string    `json:"asignacion_perfil_ref"`
	CorrelacionRef      string    `json:"correlacion_ref"`
	RolVersionRef       string    `json:"rol_version_ref"`
	VigenteDesde        time.Time `json:"vigente_desde"`
	VigenteHasta        time.Time `json:"vigente_hasta"`
	Motivo              Motivo    `json:"motivo"`
	UnidadRef           string    `json:"unidad_ref,omitempty"`
	ReferenciaActo      string    `json:"referencia_acto,omitempty"`
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
	return Recibo{CentroRef: r.CentroRef, ActorPersonaRef: r.ActorPersonaRef, PerfilActivoRef: r.PerfilActivoRef, AsignacionPerfilRef: r.AsignacionPerfilRef, CorrelacionRef: r.CorrelacionRef, RolVersionRef: r.RolVersionRef, VigenteDesde: r.VigenteDesde, VigenteHasta: r.VigenteHasta, Motivo: Motivo{CatalogoID: r.Motivo.CatalogoID, CatalogoVersion: r.Motivo.CatalogoVersion, CatalogoHuellaSHA256: r.Motivo.CatalogoHuellaSHA256, EntradaClave: r.Motivo.EntradaClave}, UnidadRef: r.UnidadRef, ReferenciaActo: r.ReferenciaActo, OperacionRef: r.OperacionRef, ActoRef: r.ActoRef, ReciboRef: r.ReciboRef,
		PropuestaRef: r.PropuestaRef, AuditoriaRef: r.AuditoriaRef,
		ObjetivoPersonaRef: r.ObjetivoPersonaRef, PerfilRef: r.PerfilRef, VinculoRef: r.VinculoRef,
		EstadoPosterior: string(r.EstadoPosterior), VersionPosterior: r.VersionPosterior,
		HuellaAntesSHA256: r.HuellaAntesSHA256, HuellaDespuesSHA256: r.HuellaDespuesSHA256,
		ConfirmadoEn: r.ConfirmadoEn}
}

type SolicitudActo struct {
	ReferenciaActo string   `json:"referencia_acto,omitempty"`
	OperacionRef   string   `json:"operacion_ref"`
	Operacion      string   `json:"operacion"`
	RolVersionRef  string   `json:"rol_version_ref"`
	Objetivo       Objetivo `json:"objetivo"`
	Motivo         Motivo   `json:"motivo"`
}

type SolicitudCierre struct {
	OperacionRef          string `json:"operacion_ref"`
	PropuestaHuellaSHA256 string `json:"propuesta_huella_sha256"`
	Decision              string `json:"decision"`
	Motivo                Motivo `json:"motivo"`
}

// SolicitudLote acepta solo el material del efecto. Identidad, evidencia,
// correlación, instantánea vigente y huella se reconstruyen en el servidor.
type SolicitudLote struct {
	OperacionRef   string         `json:"operacion_ref"`
	Cambios        []CambioPerfil `json:"cambios"`
	Motivo         Motivo         `json:"motivo"`
	ReferenciaActo string         `json:"referencia_acto,omitempty"`
}

type CambioPerfil struct {
	Operacion      string   `json:"operacion"`
	InicioVigencia string   `json:"inicio_vigencia,omitempty"`
	RolVersionRef  string   `json:"rol_version_ref"`
	Objetivo       Objetivo `json:"objetivo"`
}

type InicioEfectivoLote struct {
	Modo         string     `json:"modo,omitempty"`
	VigenteDesde *time.Time `json:"vigente_desde,omitempty"`
}

type ReciboLote struct {
	OperacionRef          string               `json:"operacion_ref"`
	ActoRef               string               `json:"acto_ref"`
	ReciboRef             string               `json:"recibo_ref"`
	AuditoriaRef          string               `json:"auditoria_ref"`
	HuellaSolicitudSHA256 string               `json:"huella_solicitud_sha256"`
	FuentesSHA256         string               `json:"fuentes_sha256"`
	ConfirmadoEn          time.Time            `json:"confirmado_en"`
	Cambios               []Recibo             `json:"cambios"`
	Inicios               []InicioEfectivoLote `json:"inicios"`
}
