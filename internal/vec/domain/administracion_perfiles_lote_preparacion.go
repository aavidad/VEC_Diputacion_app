package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"time"
)

// MaximoOpcionesPreparacionLote es el tope de altas y de bajas que devuelve la
// preparación (AUT50); si hay más, la respuesta lo indica con Truncado.
const MaximoOpcionesPreparacionLote = 64

// Misma forma que exige AUT50 para la referencia de asignación del actor.
var asignacionPreparacionLote = regexp.MustCompile(`^asignacion:[A-Za-z0-9_:-]{1,200}:v[1-9][0-9]{0,9}$`)

// SolicitudPreparacionLoteAdministracionPerfiles pide, para una persona y una
// unidad, los datos que hacen falta para construir un lote: cuenta, versiones,
// procedencia y huella de cada alta o baja posible. No cambia nada; consume una
// decisión propia (acción del lote, atributo preparacion_sha256) y deja registro.
type SolicitudPreparacionLoteAdministracionPerfiles struct {
	OperacionRef            string
	OrganizacionRef         string
	UnidadRef               string
	PersonaRef              string
	Actor                   ContextoActor
	Evidencia               EvidenciaSesionAdministracionPerfiles `json:"-"`
	InstantaneaAutorizacion InstantaneaAutorizacion
	CorrelacionRef          string
}

// ReferenciaPreparacionLoteValida acepta sólo «prep_admin:» y 32 hexadecimales.
func ReferenciaPreparacionLoteValida(ref string) bool {
	return referenciaHexAdministracionPerfiles(ref, "prep_admin:")
}

func (s SolicitudPreparacionLoteAdministracionPerfiles) validarEstructura() error {
	asignacion := s.InstantaneaAutorizacion.AsignacionPerfil
	if !ReferenciaPreparacionLoteValida(s.OperacionRef) ||
		!organizacionLoteAdministracion.MatchString(s.OrganizacionRef) ||
		!organizacionLoteAdministracion.MatchString(s.UnidadRef) ||
		!referenciaOpacaAdministracionPerfiles(s.PersonaRef, "per_") ||
		s.Actor.Validar() != nil || s.InstantaneaAutorizacion.Validar() != nil ||
		!referenciaOpacaAdministracionPerfiles(s.Actor.PersonaRef, "per_") ||
		!referenciaOpacaAdministracionPerfiles(s.Actor.PerfilActivoRef, "prf_") ||
		!asignacionPreparacionLote.MatchString(asignacion.Referencia()) ||
		s.PersonaRef == s.Actor.PersonaRef ||
		s.Actor.PersonaRef != asignacion.PrincipalID || s.Actor.PerfilActivoRef != asignacion.PerfilActivoRef ||
		!ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef) {
		return ErrActoAdministracionPerfilesInvalido
	}
	return nil
}

// CanonicoYHuella produce el material de ocho claves de texto que AUT50 valida
// (validar_material_preparacion_lote_admin_v1) y su SHA-256, que va como
// atributo preparacion_sha256 del recurso autorizado.
func (s SolicitudPreparacionLoteAdministracionPerfiles) CanonicoYHuella() ([]byte, string, error) {
	if err := s.validarEstructura(); err != nil {
		return nil, "", err
	}
	canonico := struct {
		Esquema         string
		OperacionRef    string
		ActorPersonaRef string
		PerfilActivoRef string
		AsignacionRef   string
		OrganizacionRef string
		UnidadRef       string
		PersonaRef      string
	}{"administracion_perfiles_lote_preparacion:v1", s.OperacionRef, s.Actor.PersonaRef, s.Actor.PerfilActivoRef,
		s.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), s.OrganizacionRef, s.UnidadRef, s.PersonaRef}
	b, err := json.Marshal(canonico)
	if err != nil || len(b) > 4096 {
		return nil, "", ErrActoAdministracionPerfilesInvalido
	}
	h := sha256.Sum256(b)
	return b, hex.EncodeToString(h[:]), nil
}

func (s SolicitudPreparacionLoteAdministracionPerfiles) Validar() error {
	_, _, err := s.CanonicoYHuella()
	return err
}

// AltaPosibleLoteAdministracion es un perfil registrado para el lote que la
// persona aún no tiene en la unidad, con referencias nuevas y la huella de la
// preimagen que el lote debe repetir. Nombre es el del documento del rol.
type AltaPosibleLoteAdministracion struct {
	RolVersionRef      string
	Nombre             string
	UnidadRequerida    bool
	VigenteHastaMaxima time.Time
	DuracionPropuesta  time.Duration
	PerfilRef          string
	VinculoRef         string
	HuellaSHA256       string
}

// BajaPosibleLoteAdministracion es una asignación ordinaria activa de la
// persona en la organización y unidad, con las versiones que el lote compara.
type BajaPosibleLoteAdministracion struct {
	RolVersionRef  string
	Nombre         string
	PerfilRef      string
	VinculoRef     string
	PerfilVersion  uint64
	VinculoVersion uint64
	VigenteDesde   time.Time
	VigenteHasta   time.Time
	HuellaSHA256   string
}

// PreparacionLoteAdministracionPerfiles es la respuesta de AUT50 tras COMMIT.
type PreparacionLoteAdministracionPerfiles struct {
	OperacionRef            string
	AuditoriaRef            string
	PreparadaEn             time.Time
	PersonaRef              string
	PersonaVersion          uint64
	CuentaRef               string
	CuentaVersion           uint64
	ProcedenciaRef          string
	ProcedenciaVersion      uint64
	ProcedenciaHuellaSHA256 string
	OrganizacionRef         string
	UnidadRef               string
	Altas                   []AltaPosibleLoteAdministracion
	Bajas                   []BajaPosibleLoteAdministracion
	Truncado                bool
}

// ValidarPara comprueba que la respuesta corresponde a la solicitud y que cada
// opción trae lo necesario para construir el cambio del lote sin inventar nada.
func (p PreparacionLoteAdministracionPerfiles) ValidarPara(s SolicitudPreparacionLoteAdministracionPerfiles) error {
	if s.Validar() != nil || p.OperacionRef != s.OperacionRef || p.PersonaRef != s.PersonaRef ||
		p.OrganizacionRef != s.OrganizacionRef || p.UnidadRef != s.UnidadRef ||
		!referenciaAuditoriaAdministracionPerfiles(p.AuditoriaRef) || !instanteContextoActorCanonico(p.PreparadaEn) ||
		p.PersonaVersion == 0 || p.CuentaVersion == 0 || p.ProcedenciaVersion == 0 ||
		!referenciaOpacaAdministracionPerfiles(p.CuentaRef, "cta_") || !procedenciaAdministracionPerfiles(p.ProcedenciaRef) ||
		!huellaAdministracionPerfiles(p.ProcedenciaHuellaSHA256) ||
		len(p.Altas) > MaximoOpcionesPreparacionLote || len(p.Bajas) > MaximoOpcionesPreparacionLote {
		return ErrActoAdministracionPerfilesInvalido
	}
	perfiles, vinculos := map[string]bool{}, map[string]bool{}
	for _, a := range p.Altas {
		if !rolVersionAdministracionPerfiles(a.RolVersionRef) || len(a.Nombre) > 256 ||
			!referenciaOpacaAdministracionPerfiles(a.PerfilRef, "prf_") || !referenciaOpacaAdministracionPerfiles(a.VinculoRef, "vca_") ||
			!huellaAdministracionPerfiles(a.HuellaSHA256) || !a.VigenteHastaMaxima.After(p.PreparadaEn) ||
			a.DuracionPropuesta <= 0 || perfiles[a.PerfilRef] || vinculos[a.VinculoRef] {
			return ErrActoAdministracionPerfilesInvalido
		}
		perfiles[a.PerfilRef], vinculos[a.VinculoRef] = true, true
	}
	for _, b := range p.Bajas {
		if !rolVersionAdministracionPerfiles(b.RolVersionRef) || len(b.Nombre) > 256 ||
			!referenciaOpacaAdministracionPerfiles(b.PerfilRef, "prf_") || !referenciaOpacaAdministracionPerfiles(b.VinculoRef, "vca_") ||
			b.PerfilVersion == 0 || b.VinculoVersion == 0 || !huellaAdministracionPerfiles(b.HuellaSHA256) ||
			b.VigenteDesde.IsZero() || !b.VigenteHasta.After(b.VigenteDesde) ||
			perfiles[b.PerfilRef] || vinculos[b.VinculoRef] {
			return ErrActoAdministracionPerfilesInvalido
		}
		perfiles[b.PerfilRef], vinculos[b.VinculoRef] = true, true
	}
	return nil
}
