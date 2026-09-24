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
	ErrRegistroEmpleadoB2Invalido      = errors.New("personal: registro de empleado invalido")
	ErrRegistroEmpleadoB2Denegado      = errors.New("personal: consulta de registro denegada")
	ErrRegistroEmpleadoB2NoEncontrado  = errors.New("personal: empleado no encontrado")
	ErrRegistroEmpleadoB2NoDisponible  = errors.New("personal: registro de empleado no disponible")
	ErrRegistroEmpleadoB2Conflicto     = errors.New("personal: conflicto de version o idempotencia")
	ErrCoberturaVacantesB2NoAcreditada = errors.New("personal: cobertura de vacantes no acreditada")
	patronReferenciaB2                 = regexp.MustCompile(`^[a-z][a-z0-9_:-]{2,159}$`)
	patronUUIDRegistroB2               = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	patronCursorB2                     = regexp.MustCompile(`^[A-Za-z0-9_-]{1,256}$`)
)

const (
	AccionFichaEmpleadoB2    = "personal.registro_empleado.ficha.consultar"
	AccionVacantesB2         = "personal.registro_empleado.vacantes.consultar"
	AudienciaFichaEmpleadoB2 = "vec_personal.registro_empleado.ficha.v1"
	AudienciaVacantesB2      = "vec_personal.registro_empleado.vacantes.v1"
	LimiteVacantesB2         = 100
)

// CorteEmpleadoB2 separa la fecha administrativa del instante en que el hecho
// era conocido. Un hecho posterior no modifica retrospectivamente esa foto.
type CorteEmpleadoB2 struct {
	VigenteEn  FechaCivil `json:"vigente_en"`
	ConocidoEn time.Time  `json:"conocido_en"`
}

func (c CorteEmpleadoB2) Validar() error {
	if c.VigenteEn.Validar() != nil || !instanteRegistroB2Valido(c.ConocidoEn) {
		return ErrRegistroEmpleadoB2Invalido
	}
	return nil
}

type SolicitudFichaEmpleadoB2 struct {
	EmpleadoRef string
	Corte       CorteEmpleadoB2
	Actor       core.ContextoActor
}

type SolicitudVacantesB2 struct {
	OrganismoRef string
	Corte        CorteEmpleadoB2
	Limite       int
	Cursor       string
	Actor        core.ContextoActor
}

// MaterialConsultaRegistroEmpleadoB2 liga selector, perfil y contexto a la
// concesión. La identidad viene de la frontera servidor, nunca del navegador.
type MaterialConsultaRegistroEmpleadoB2 struct {
	operacion    string
	empleadoRef  string
	organismoRef string
	corte        CorteEmpleadoB2
	limite       int
	cursor       string
	actor        core.ContextoActor
	canonico     []byte
	recurso      core.RecursoAutorizable
}

func NuevoMaterialFichaEmpleadoB2(s SolicitudFichaEmpleadoB2) (MaterialConsultaRegistroEmpleadoB2, error) {
	if !ReferenciaEmpleadoValida(s.EmpleadoRef) || s.Corte.Validar() != nil || s.Actor.Validar() != nil {
		return MaterialConsultaRegistroEmpleadoB2{}, ErrRegistroEmpleadoB2Invalido
	}
	return nuevoMaterialConsultaB2("ficha", s.EmpleadoRef, "", s.Corte, 0, "", s.Actor)
}

func NuevoMaterialVacantesB2(s SolicitudVacantesB2) (MaterialConsultaRegistroEmpleadoB2, error) {
	if !patronReferenciaB2.MatchString(s.OrganismoRef) || s.Corte.Validar() != nil || s.Limite < 1 || s.Limite > LimiteVacantesB2 || (s.Cursor != "" && !patronCursorB2.MatchString(s.Cursor)) || s.Actor.Validar() != nil {
		return MaterialConsultaRegistroEmpleadoB2{}, ErrRegistroEmpleadoB2Invalido
	}
	return nuevoMaterialConsultaB2("vacantes", "", s.OrganismoRef, s.Corte, s.Limite, s.Cursor, s.Actor)
}

func nuevoMaterialConsultaB2(operacion, empleado, organismo string, corte CorteEmpleadoB2, limite int, cursor string, actor core.ContextoActor) (MaterialConsultaRegistroEmpleadoB2, error) {
	actor, err := actor.Clonar()
	if err != nil {
		return MaterialConsultaRegistroEmpleadoB2{}, ErrRegistroEmpleadoB2Invalido
	}
	material := struct {
		Esquema          string `json:"esquema"`
		Operacion        string `json:"operacion"`
		EmpleadoRef      string `json:"empleado_ref"`
		OrganismoRef     string `json:"organismo_ref"`
		VigenteEn        string `json:"vigente_en"`
		ConocidoEn       string `json:"conocido_en"`
		Limite           int    `json:"limite"`
		Cursor           string `json:"cursor"`
		ActorRef         string `json:"actor_ref"`
		ContextoActorRef string `json:"contexto_actor_ref"`
		ContextoVersion  uint64 `json:"contexto_version"`
		CuentaRef        string `json:"cuenta_ref"`
		CuentaVersion    uint64 `json:"cuenta_version"`
		PerfilRef        string `json:"perfil_ref"`
		PerfilVersion    uint64 `json:"perfil_version"`
		PersonaRef       string `json:"persona_ref"`
		PersonaVersion   uint64 `json:"persona_version"`
	}{
		"vec.personal.registro-empleado-b2.consulta.v1", operacion, empleado, organismo,
		corte.VigenteEn.Texto(), corte.ConocidoEn.UTC().Format("2006-01-02T15:04:05.000000Z"), limite, cursor,
		actor.Principal.ID, actor.Instantanea.VinculoRef, actor.Instantanea.VinculoVersion,
		actor.Instantanea.CuentaRef, actor.Instantanea.CuentaVersion, actor.PerfilActivoRef,
		actor.Instantanea.PerfilVersion, actor.PersonaRef, actor.Instantanea.PersonaVersion,
	}
	canonico, err := json.Marshal(material)
	if err != nil {
		return MaterialConsultaRegistroEmpleadoB2{}, ErrRegistroEmpleadoB2Invalido
	}
	suma := sha256.Sum256(canonico)
	referencia, tipo := empleado, "registro_empleado_rrhh"
	if operacion == "vacantes" {
		referencia, tipo = organismo, "vacantes_rrhh"
	}
	ambitos := map[string]string{}
	if empleado != "" {
		ambitos["empleado_ref"] = empleado
	}
	if organismo != "" {
		ambitos["organismo_ref"] = organismo
	}
	recurso := core.RecursoAutorizable{
		Referencia: referencia, ModuloID: "personal", Tipo: tipo,
		Ambitos:   ambitos,
		Atributos: map[string]string{"operacion": operacion, "vigente_en": corte.VigenteEn.Texto(), "conocido_en": material.ConocidoEn, "material_sha256": hex.EncodeToString(suma[:])},
	}
	if _, err = recurso.HuellaContextoAutorizacionSHA256(); err != nil {
		return MaterialConsultaRegistroEmpleadoB2{}, ErrRegistroEmpleadoB2Invalido
	}
	return MaterialConsultaRegistroEmpleadoB2{operacion, empleado, organismo, corte, limite, cursor, actor, canonico, recurso}, nil
}

func (m MaterialConsultaRegistroEmpleadoB2) Operacion() string      { return m.operacion }
func (m MaterialConsultaRegistroEmpleadoB2) EmpleadoRef() string    { return m.empleadoRef }
func (m MaterialConsultaRegistroEmpleadoB2) OrganismoRef() string   { return m.organismoRef }
func (m MaterialConsultaRegistroEmpleadoB2) Corte() CorteEmpleadoB2 { return m.corte }
func (m MaterialConsultaRegistroEmpleadoB2) Limite() int            { return m.limite }
func (m MaterialConsultaRegistroEmpleadoB2) Cursor() string         { return m.cursor }
func (m MaterialConsultaRegistroEmpleadoB2) Canonico() []byte {
	return append([]byte(nil), m.canonico...)
}
func (m MaterialConsultaRegistroEmpleadoB2) Actor() core.ContextoActor {
	a, _ := m.actor.Clonar()
	return a
}
func (m MaterialConsultaRegistroEmpleadoB2) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = copiarMapaRelacion(r.Ambitos)
	r.Atributos = copiarMapaRelacion(r.Atributos)
	return r
}
func (m MaterialConsultaRegistroEmpleadoB2) HuellaSHA256() (string, error) {
	return m.recurso.HuellaContextoAutorizacionSHA256()
}

// TrazaEmpleadoB2 identifica una versión inmutable de un hecho. Los intervalos
// son semiabiertos [Desde,Hasta); RegistradaEn pertenece al eje de conocimiento.
type TrazaEmpleadoB2 struct {
	Desde         FechaCivil `json:"desde"`
	Hasta         FechaCivil `json:"hasta,omitempty"`
	RegistradaEn  time.Time  `json:"registrada_en"`
	Version       int64      `json:"version"`
	ActoRef       string     `json:"acto_ref"`
	FuenteRef     string     `json:"fuente_ref"`
	FuenteVersion int64      `json:"fuente_version"`
}

func (t TrazaEmpleadoB2) ValidarEn(c CorteEmpleadoB2) error {
	if t.Desde.Validar() != nil || (t.Hasta != "" && (t.Hasta.Validar() != nil || !t.Desde.AntesDe(t.Hasta))) ||
		!instanteRegistroB2Valido(t.RegistradaEn) || t.RegistradaEn.After(c.ConocidoEn) || t.Version < 1 ||
		!patronReferenciaB2.MatchString(t.ActoRef) || !patronReferenciaB2.MatchString(t.FuenteRef) || t.FuenteVersion < 1 {
		return ErrRegistroEmpleadoB2Invalido
	}
	return nil
}

type RelacionRegistroEmpleadoB2 struct {
	RelacionRef        string          `json:"relacion_ref"`
	UnidadRef          string          `json:"unidad_ref"`
	OrganismoRef       string          `json:"organismo_ref"`
	RegimenRef         string          `json:"regimen_ref"`
	ModalidadRef       string          `json:"modalidad_ref"`
	UnidadDenominacion string          `json:"unidad_denominacion,omitempty"`
	Estado             string          `json:"estado"`
	Traza              TrazaEmpleadoB2 `json:"traza"`
}
type OcupacionEmpleadoB2 struct {
	OcupacionRef       string          `json:"ocupacion_ref"`
	RelacionRef        string          `json:"relacion_ref"`
	PlazaRef           string          `json:"plaza_ref"`
	PuestoRef          string          `json:"puesto_ref"`
	UnidadRef          string          `json:"unidad_ref"`
	UnidadDenominacion string          `json:"unidad_denominacion,omitempty"`
	PuestoDenominacion string          `json:"puesto_denominacion,omitempty"`
	CodigoPlazaFuente  string          `json:"codigo_plaza_fuente,omitempty"`
	ModalidadRef       string          `json:"modalidad_ref"`
	Clase              string          `json:"clase"`
	Estado             string          `json:"estado"`
	Traza              TrazaEmpleadoB2 `json:"traza"`
}
type SituacionEmpleadoB2 struct {
	SituacionRef string          `json:"situacion_ref"`
	RelacionRef  string          `json:"relacion_ref"`
	CodigoRef    string          `json:"codigo_ref"`
	Estado       string          `json:"estado"`
	Traza        TrazaEmpleadoB2 `json:"traza"`
}
type ServicioReconocidoB2 struct {
	ServicioRef     string          `json:"servicio_ref"`
	RelacionRef     string          `json:"relacion_ref"`
	Estado          string          `json:"estado"`
	ClaseRef        string          `json:"clase_ref"`
	PeriodoDesde    FechaCivil      `json:"periodo_desde"`
	PeriodoHasta    FechaCivil      `json:"periodo_hasta"`
	DiasReconocidos int64           `json:"dias_reconocidos"`
	Traza           TrazaEmpleadoB2 `json:"traza"`
}
type FichaEmpleadoB2 struct {
	EmpleadoRef            string                       `json:"empleado_ref"`
	PersonaRef             string                       `json:"persona_ref"`
	EficaciaAdministrativa bool                         `json:"eficacia_administrativa"`
	FirmaOficial           bool                         `json:"firma_oficial"`
	Corte                  CorteEmpleadoB2              `json:"corte"`
	Version                int64                        `json:"version"`
	Relaciones             []RelacionRegistroEmpleadoB2 `json:"relaciones"`
	Ocupaciones            []OcupacionEmpleadoB2        `json:"ocupaciones"`
	Situaciones            []SituacionEmpleadoB2        `json:"situaciones"`
	Servicios              []ServicioReconocidoB2       `json:"servicios"`
}

func (f FichaEmpleadoB2) ValidarPara(m MaterialConsultaRegistroEmpleadoB2) error {
	if m.Operacion() != "ficha" || f.EmpleadoRef != m.EmpleadoRef() || !ReferenciaPersonaValida(f.PersonaRef) || f.EficaciaAdministrativa || f.FirmaOficial || !corteRegistroB2Igual(f.Corte, m.Corte()) || f.Version < 1 ||
		len(f.Relaciones) > 200 || len(f.Ocupaciones) > 200 || len(f.Situaciones) > 200 || len(f.Servicios) > 200 {
		return ErrRegistroEmpleadoB2Invalido
	}
	ids := map[string]struct{}{}
	for _, r := range f.Relaciones {
		if !ReferenciaRelacionValida(r.RelacionRef) || !patronReferenciaB2.MatchString(r.UnidadRef) || !patronReferenciaB2.MatchString(r.OrganismoRef) || !patronReferenciaB2.MatchString(r.RegimenRef) || !patronReferenciaB2.MatchString(r.ModalidadRef) || !estadoRelacionB2Valido(r.Estado) || r.Traza.ValidarEn(f.Corte) != nil || repetidoB2(ids, r.RelacionRef+":"+strconv.FormatInt(r.Traza.Version, 10)) {
			return ErrRegistroEmpleadoB2Invalido
		}
	}
	for _, o := range f.Ocupaciones {
		if !patronReferenciaB2.MatchString(o.OcupacionRef) || !ReferenciaRelacionValida(o.RelacionRef) || !referenciaOrganizacionB2Valida(o.PlazaRef) || (o.PuestoRef != "" && !referenciaOrganizacionB2Valida(o.PuestoRef)) || !patronReferenciaB2.MatchString(o.UnidadRef) || !patronReferenciaB2.MatchString(o.ModalidadRef) || (o.Clase != "titular" && o.Clase != "provisional" && o.Clase != "temporal" && o.Clase != "reserva") || (o.Estado != "vigente" && o.Estado != "finalizada") || o.Traza.ValidarEn(f.Corte) != nil {
			return ErrRegistroEmpleadoB2Invalido
		}
	}
	for _, s := range f.Situaciones {
		if !patronReferenciaB2.MatchString(s.SituacionRef) || !ReferenciaRelacionValida(s.RelacionRef) || !patronReferenciaB2.MatchString(s.CodigoRef) || (s.Estado != "vigente" && s.Estado != "finalizada" && s.Estado != "rectificada") || s.Traza.ValidarEn(f.Corte) != nil {
			return ErrRegistroEmpleadoB2Invalido
		}
	}
	for _, s := range f.Servicios {
		if !patronReferenciaB2.MatchString(s.ServicioRef) || !ReferenciaRelacionValida(s.RelacionRef) || (s.Estado != "declarado" && s.Estado != "comprobado" && s.Estado != "reconocido") || !patronReferenciaB2.MatchString(s.ClaseRef) || s.PeriodoDesde.Validar() != nil || s.PeriodoHasta.Validar() != nil || s.PeriodoHasta.AntesDe(s.PeriodoDesde) || s.DiasReconocidos < 0 || s.Traza.ValidarEn(f.Corte) != nil {
			return ErrRegistroEmpleadoB2Invalido
		}
	}
	return nil
}

type VacantePlazaB2 struct {
	PlazaRef            string         `json:"plaza_ref"`
	PuestoRef           string         `json:"puesto_ref,omitempty"`
	UnidadRef           string         `json:"unidad_ref"`
	UnidadDenominacion  string         `json:"unidad_denominacion,omitempty"`
	PuestoDenominacion  string         `json:"puesto_denominacion,omitempty"`
	CodigoPlazaFuente   string         `json:"codigo_plaza_fuente,omitempty"`
	EstadoCobertura     string         `json:"estado_cobertura"`
	VersionPlantillaRef string         `json:"version_plantilla_ref"`
	VersionRPTRef       string         `json:"version_rpt_ref,omitempty"`
	Traza               TrazaVacanteB2 `json:"traza"`
}

// La plaza pertenece a B3: su revisión estructural no es una versión de la
// fuente de Personal. La huella procede del hecho organizativo publicado.
type TrazaVacanteB2 struct {
	Desde               FechaCivil `json:"desde"`
	Hasta               FechaCivil `json:"hasta,omitempty"`
	RegistradaEn        time.Time  `json:"registrada_en"`
	RevisionEstructural int64      `json:"revision_estructural"`
	VersionPlantillaRef string     `json:"version_plantilla_ref"`
	ActoRef             string     `json:"acto_ref"`
	FuenteRef           string     `json:"fuente_ref"`
	FuenteHuellaSHA256  string     `json:"fuente_huella_sha256"`
}

func (t TrazaVacanteB2) ValidarEn(c CorteEmpleadoB2) error {
	if t.Desde.Validar() != nil || (t.Hasta != "" && (t.Hasta.Validar() != nil || !t.Desde.AntesDe(t.Hasta))) || !instanteRegistroB2Valido(t.RegistradaEn) || t.RegistradaEn.After(c.ConocidoEn) || t.RevisionEstructural < 1 || !patronReferenciaB2.MatchString(t.VersionPlantillaRef) || !patronReferenciaB2.MatchString(t.ActoRef) || !patronReferenciaB2.MatchString(t.FuenteRef) || !huellaRegistroDominioB2Valida(t.FuenteHuellaSHA256) {
		return ErrRegistroEmpleadoB2Invalido
	}
	return nil
}

type PaginaVacantesB2 struct {
	OrganismoRef    string           `json:"organismo_ref"`
	Corte           CorteEmpleadoB2  `json:"corte"`
	Limite          int              `json:"limite"`
	Cursor          string           `json:"cursor"`
	CursorSiguiente string           `json:"cursor_siguiente,omitempty"`
	Cobertura       string           `json:"cobertura"`
	Vacantes        []VacantePlazaB2 `json:"vacantes"`
}

func (p PaginaVacantesB2) ValidarPara(m MaterialConsultaRegistroEmpleadoB2) error {
	if m.Operacion() != "vacantes" || p.OrganismoRef != m.OrganismoRef() || !corteRegistroB2Igual(p.Corte, m.Corte()) || p.Limite != m.Limite() || p.Cursor != m.Cursor() || p.Cobertura != "completa" || len(p.Vacantes) > p.Limite || (p.CursorSiguiente != "" && (!patronCursorB2.MatchString(p.CursorSiguiente) || p.CursorSiguiente == p.Cursor)) {
		return ErrRegistroEmpleadoB2Invalido
	}
	ids := map[string]struct{}{}
	for _, v := range p.Vacantes {
		if !referenciaOrganizacionB2Valida(v.PlazaRef) || (v.PuestoRef != "" && !referenciaOrganizacionB2Valida(v.PuestoRef)) || !patronReferenciaB2.MatchString(v.UnidadRef) || !patronReferenciaB2.MatchString(v.VersionPlantillaRef) || (v.VersionRPTRef != "" && !patronReferenciaB2.MatchString(v.VersionRPTRef)) || v.Traza.VersionPlantillaRef != v.VersionPlantillaRef || v.EstadoCobertura != "vacante_sin_ocupacion" || v.Traza.ValidarEn(p.Corte) != nil || repetidoB2(ids, v.PlazaRef) {
			return ErrRegistroEmpleadoB2Invalido
		}
	}
	return nil
}

func estadoRelacionB2Valido(s string) bool {
	return s == "vigente" || s == "finalizada" || s == "suspendida"
}
func repetidoB2(ids map[string]struct{}, id string) bool {
	if _, ok := ids[id]; ok {
		return true
	}
	ids[id] = struct{}{}
	return false
}
func instanteRegistroB2Valido(t time.Time) bool {
	_, offset := t.Zone()
	return !t.IsZero() && offset == 0 && t.Nanosecond()%1000 == 0
}
func corteRegistroB2Igual(a, b CorteEmpleadoB2) bool {
	return a.VigenteEn == b.VigenteEn && instanteRegistroB2Valido(a.ConocidoEn) && a.ConocidoEn.Equal(b.ConocidoEn)
}
func referenciaOrganizacionB2Valida(s string) bool {
	return patronReferenciaB2.MatchString(s) || patronUUIDRegistroB2.MatchString(s)
}
