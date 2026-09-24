package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrConsultaOrganizacionHistoricaInvalida = errors.New("personal: consulta historica de organizacion invalida")
	ErrConsultaOrganizacionHistoricaDenegada = errors.New("personal: consulta historica de organizacion denegada")
	ErrOrganizacionHistoricaNoDisponible     = errors.New("personal: organizacion historica no disponible")
	patronIDOrganizacionHistorica            = regexp.MustCompile(`^[a-z][a-z0-9_:-]{2,159}$`)
	patronHuellaOrganizacionHistorica        = regexp.MustCompile(`^[a-f0-9]{64}$`)
	patronCursorOrganizacionHistorica        = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

const (
	AccionConsultaOrganizacionHistorica    = "personal.organizacion_historica.consultar"
	AudienciaConsultaOrganizacionHistorica = "vec_personal.organizacion_historica.consultar.v1"
	LimiteMaximoOrganizacionHistorica      = 100
)

// SelectorOrganizacionHistorica reconstruye una foto: efectos en fecha civil y
// conocimiento en instante UTC. Las versiones opcionales restringen la foto;
// nunca se toma la fecha de publicación como fecha de efectos.
type SelectorOrganizacionHistorica struct {
	OrganismoRef        string     `json:"organismo_ref"`
	UnidadClave         string     `json:"unidad_clave"`
	VigenteEn           FechaCivil `json:"vigente_en"`
	ConocidoEn          time.Time  `json:"conocido_en"`
	VersionRPTRef       string     `json:"version_rpt_ref"`
	VersionPlantillaRef string     `json:"version_plantilla_ref"`
	Limite              int        `json:"limite"`
	Cursor              string     `json:"cursor"`
}

func (s SelectorOrganizacionHistorica) Validar() error {
	if !patronIDOrganizacionHistorica.MatchString(s.OrganismoRef) ||
		!referenciaHistoricaOpcional(s.UnidadClave) ||
		s.VigenteEn.Validar() != nil || !instanteHistoricoValido(s.ConocidoEn) ||
		(s.VersionRPTRef != "" && !patronIDOrganizacionHistorica.MatchString(s.VersionRPTRef)) ||
		(s.VersionPlantillaRef != "" && !patronIDOrganizacionHistorica.MatchString(s.VersionPlantillaRef)) ||
		s.Limite < 1 || s.Limite > LimiteMaximoOrganizacionHistorica || !cursorHistoricoValido(s.Cursor) {
		return ErrConsultaOrganizacionHistoricaInvalida
	}
	return nil
}

// La identidad del actor procede del contexto servidor; ningún campo de
// selección del navegador puede sustituirla.
type SolicitudConsultaOrganizacionHistorica struct {
	Selector SelectorOrganizacionHistorica
	Actor    core.ContextoActor
}

type MaterialConsultaOrganizacionHistorica struct {
	solicitud SolicitudConsultaOrganizacionHistorica
	canonico  []byte
	recurso   core.RecursoAutorizable
}

func NuevoMaterialConsultaOrganizacionHistorica(s SolicitudConsultaOrganizacionHistorica) (MaterialConsultaOrganizacionHistorica, error) {
	if s.Selector.Validar() != nil || s.Actor.Validar() != nil {
		return MaterialConsultaOrganizacionHistorica{}, ErrConsultaOrganizacionHistoricaInvalida
	}
	actor, err := s.Actor.Clonar()
	if err != nil {
		return MaterialConsultaOrganizacionHistorica{}, ErrConsultaOrganizacionHistoricaInvalida
	}
	s.Actor = actor
	material := struct {
		Esquema             string `json:"esquema"`
		OrganismoRef        string `json:"organismo_ref"`
		UnidadClave         string `json:"unidad_clave"`
		VigenteEn           string `json:"vigente_en"`
		ConocidoEn          string `json:"conocido_en"`
		VersionRPTRef       string `json:"version_rpt_ref"`
		VersionPlantillaRef string `json:"version_plantilla_ref"`
		Limite              int    `json:"limite"`
		Cursor              string `json:"cursor"`
		ActorRef            string `json:"actor_ref"`
		ContextoActorRef    string `json:"contexto_actor_ref"`
		ContextoVersion     uint64 `json:"contexto_version"`
		PersonaVersion      uint64 `json:"persona_version"`
		PerfilRef           string `json:"perfil_ref"`
		PerfilVersion       uint64 `json:"perfil_version"`
	}{"vec.personal.organizacion-historica.v1", s.Selector.OrganismoRef, s.Selector.UnidadClave,
		s.Selector.VigenteEn.Texto(), s.Selector.ConocidoEn.Format("2006-01-02T15:04:05.000000Z"),
		s.Selector.VersionRPTRef, s.Selector.VersionPlantillaRef, s.Selector.Limite, s.Selector.Cursor,
		s.Actor.Principal.ID, s.Actor.Instantanea.VinculoRef, s.Actor.Instantanea.VinculoVersion,
		s.Actor.Instantanea.PersonaVersion, s.Actor.PerfilActivoRef, s.Actor.Instantanea.PerfilVersion}
	canonico, err := json.Marshal(material)
	if err != nil {
		return MaterialConsultaOrganizacionHistorica{}, ErrConsultaOrganizacionHistoricaInvalida
	}
	h := sha256.Sum256(canonico)
	recurso := core.RecursoAutorizable{
		Referencia: s.Selector.OrganismoRef, ModuloID: "personal", Tipo: "organizacion_historica",
		Ambitos: map[string]string{"organismo_ref": s.Selector.OrganismoRef, "unidad_clave": versionONinguna(s.Selector.UnidadClave)},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:]), "vigente_en": s.Selector.VigenteEn.Texto(), "conocido_en": material.ConocidoEn,
			"version_rpt_ref": versionONinguna(s.Selector.VersionRPTRef), "version_plantilla_ref": versionONinguna(s.Selector.VersionPlantillaRef), "limite": strconv.Itoa(s.Selector.Limite), "cursor": versionONinguna(s.Selector.Cursor)},
	}
	if _, err := recurso.HuellaContextoAutorizacionSHA256(); err != nil {
		return MaterialConsultaOrganizacionHistorica{}, ErrConsultaOrganizacionHistoricaInvalida
	}
	return MaterialConsultaOrganizacionHistorica{solicitud: s, canonico: canonico, recurso: recurso}, nil
}

func versionONinguna(v string) string {
	if v == "" {
		return "sin_seleccion"
	}
	return v
}
func (m MaterialConsultaOrganizacionHistorica) Solicitud() SolicitudConsultaOrganizacionHistorica {
	s := m.solicitud
	actor, err := s.Actor.Clonar()
	if err != nil {
		return SolicitudConsultaOrganizacionHistorica{}
	}
	s.Actor = actor
	return s
}
func (m MaterialConsultaOrganizacionHistorica) Canonico() []byte {
	return append([]byte(nil), m.canonico...)
}
func (m MaterialConsultaOrganizacionHistorica) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = copiarMapaRelacion(r.Ambitos)
	r.Atributos = copiarMapaRelacion(r.Atributos)
	return r
}
func (m MaterialConsultaOrganizacionHistorica) HuellaSHA256() (string, error) {
	return m.recurso.HuellaContextoAutorizacionSHA256()
}

// TrazaOrganizacionHistorica acompaña a cada hecho. Hasta es exclusivo en los
// dos ejes; un Hasta vacío representa intervalo abierto. Los códigos de la
// fuente son atributos, no identidades técnicas.
type TrazaOrganizacionHistorica struct {
	ID            string     `json:"id"`
	Version       int64      `json:"version"`
	FuenteRef     string     `json:"fuente_ref"`
	ActoRef       string     `json:"acto_ref"`
	HuellaSHA256  string     `json:"huella_sha256"`
	EfectosDesde  FechaCivil `json:"efectos_desde"`
	EfectosHasta  FechaCivil `json:"efectos_hasta,omitempty"`
	ConocidoDesde time.Time  `json:"conocido_desde"`
	ConocidoHasta time.Time  `json:"conocido_hasta,omitempty"`
}

func (t TrazaOrganizacionHistorica) ValidarEn(s SelectorOrganizacionHistorica) error {
	if !patronIDOrganizacionHistorica.MatchString(t.ID) || t.Version < 1 ||
		!patronIDOrganizacionHistorica.MatchString(t.FuenteRef) || !patronIDOrganizacionHistorica.MatchString(t.ActoRef) ||
		!patronHuellaOrganizacionHistorica.MatchString(t.HuellaSHA256) || t.EfectosDesde.Validar() != nil ||
		(t.EfectosHasta != "" && (t.EfectosHasta.Validar() != nil || !t.EfectosDesde.AntesDe(t.EfectosHasta))) ||
		!instanteHistoricoValido(t.ConocidoDesde) ||
		(!t.ConocidoHasta.IsZero() && (!instanteHistoricoValido(t.ConocidoHasta) || !t.ConocidoDesde.Before(t.ConocidoHasta))) ||
		s.VigenteEn.AntesDe(t.EfectosDesde) || (t.EfectosHasta != "" && !s.VigenteEn.AntesDe(t.EfectosHasta)) ||
		s.ConocidoEn.Before(t.ConocidoDesde) || (!t.ConocidoHasta.IsZero() && !s.ConocidoEn.Before(t.ConocidoHasta)) {
		return ErrOrganizacionHistoricaNoDisponible
	}
	return nil
}

type UnidadOrganizacionHistorica struct {
	Traza            TrazaOrganizacionHistorica `json:"traza"`
	CatalogoID       string                     `json:"catalogo_id"`
	CatalogoVersion  int                        `json:"catalogo_version"`
	CatalogoRevision int                        `json:"catalogo_revision"`
	ClaveCatalogo    string                     `json:"clave_catalogo"`
	PadreID          string                     `json:"padre_id,omitempty"`
	Tipo             string                     `json:"tipo"`
	Etiqueta         string                     `json:"etiqueta"`
}

type PuestoTipoOrganizacionHistorica struct {
	Traza            TrazaOrganizacionHistorica `json:"traza"`
	VersionRPTRef    string                     `json:"version_rpt_ref"`
	CodigoFuente     string                     `json:"codigo_fuente"`
	UnidadID         string                     `json:"unidad_id"`
	Denominacion     string                     `json:"denominacion"`
	ClasificacionRef string                     `json:"clasificacion_ref"`
}

type DotacionOrganizacionHistorica struct {
	Traza         TrazaOrganizacionHistorica `json:"traza"`
	VersionRPTRef string                     `json:"version_rpt_ref"`
	PuestoTipoID  string                     `json:"puesto_tipo_id"`
	Cantidad      int                        `json:"cantidad"`
}

type PlazaOrganizacionHistorica struct {
	Traza               TrazaOrganizacionHistorica `json:"traza"`
	VersionPlantillaRef string                     `json:"version_plantilla_ref"`
	CodigoFuente        string                     `json:"codigo_fuente"`
	ClasificacionRef    string                     `json:"clasificacion_ref"`
	UnidadID            string                     `json:"unidad_id"`
	EstadoEstructural   string                     `json:"estado_estructural"`
}

type PuestoIndividualOrganizacionHistorica struct {
	Traza             TrazaOrganizacionHistorica `json:"traza"`
	VersionRPTRef     string                     `json:"version_rpt_ref"`
	CodigoFuente      string                     `json:"codigo_fuente"`
	PuestoTipoID      string                     `json:"puesto_tipo_id"`
	UnidadID          string                     `json:"unidad_id"`
	EstadoEstructural string                     `json:"estado_estructural"`
}

type VinculoPlazaPuestoHistorico struct {
	Traza    TrazaOrganizacionHistorica `json:"traza"`
	PlazaID  string                     `json:"plaza_id"`
	PuestoID string                     `json:"puesto_id"`
}

func instanteHistoricoValido(t time.Time) bool {
	_, offset := t.Zone()
	return !t.IsZero() && offset == 0 && t.Nanosecond()%1000 == 0
}

func referenciaHistoricaOpcional(v string) bool {
	return v == "" || patronIDOrganizacionHistorica.MatchString(v)
}
func cursorHistoricoValido(v string) bool {
	return v == "" || (len(v) <= 256 && patronCursorOrganizacionHistorica.MatchString(v))
}
