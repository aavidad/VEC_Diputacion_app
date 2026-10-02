package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionVinculoPropioCRN11      = "personal.vinculo_propio.crn11.consultar"
	AudienciaVinculoPropioCRN11   = "vec_personal.vinculo_propio.crn11.v1"
	FinalidadVinculoPropioCRN11   = "acreditar_vinculo_historico_propio_crn11"
	TipoRecursoVinculoPropioCRN11 = "vinculo_historico_propio_crn11"
)

var (
	ErrVinculoCRN11Invalido     = errors.New("personal: vinculo CRN11 invalido")
	ErrVinculoCRN11Denegado     = errors.New("personal: vinculo CRN11 denegado")
	ErrVinculoCRN11NoDisponible = errors.New("personal: vinculo CRN11 no disponible")
	patronProyeccionCRN11       = regexp.MustCompile(`^pep_[A-Za-z0-9_-]{22,128}$`)
	patronProcedenciaCRN11      = regexp.MustCompile(`^prc_[A-Za-z0-9_-]{22,128}$`)
)

type SolicitudVinculoPropioCRN11 struct {
	Actor       core.ContextoActor
	EmpleadoRef string
}

// MaterialVinculoPropioCRN11 liga la lectura mínima a la identidad registrada.
// No fija la fecha del olvido ni representa un hecho histórico de Personal.
type MaterialVinculoPropioCRN11 struct {
	actor             core.ContextoActor
	empleadoRef       string
	proyeccionRef     string
	proyeccionVersion uint64
	canonico          []byte
	recurso           core.RecursoAutorizable
}

func NuevoMaterialVinculoPropioCRN11(s SolicitudVinculoPropioCRN11) (MaterialVinculoPropioCRN11, error) {
	if s.Actor.Validar() != nil || !ReferenciaEmpleadoValida(s.EmpleadoRef) || !ReferenciaPersonaValida(s.Actor.PersonaRef) || s.Actor.Principal.ID != s.Actor.PersonaRef {
		return MaterialVinculoPropioCRN11{}, ErrVinculoCRN11Invalido
	}
	empleados, err := s.Actor.Referencias(core.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 || empleados[0] != s.EmpleadoRef || !s.Actor.AlcanceProyecciones().IncluyeEmpleado() {
		return MaterialVinculoPropioCRN11{}, ErrVinculoCRN11Denegado
	}
	var proyeccion core.VinculoReferenciaContextoActor
	for _, v := range s.Actor.Instantanea.Vinculos {
		if v.Tipo == core.TipoReferenciaContextoActorEmpleado && v.Referencia == s.EmpleadoRef {
			proyeccion = v
		}
	}
	if !patronProyeccionCRN11.MatchString(proyeccion.VinculoRef) || proyeccion.Version == 0 || proyeccion.Version > math.MaxInt64 {
		return MaterialVinculoPropioCRN11{}, ErrVinculoCRN11Denegado
	}
	actor, err := s.Actor.Clonar()
	if err != nil {
		return MaterialVinculoPropioCRN11{}, ErrVinculoCRN11Invalido
	}
	canonico, err := json.Marshal(struct {
		Esquema          string `json:"esquema"`
		EmpleadoRef      string `json:"empleado_ref"`
		ActorRef         string `json:"actor_ref"`
		ContextoActorRef string `json:"contexto_actor_ref"`
		ContextoVersion  uint64 `json:"contexto_version"`
		CuentaRef        string `json:"cuenta_ref"`
		CuentaVersion    uint64 `json:"cuenta_version"`
		PerfilRef        string `json:"perfil_ref"`
		PerfilVersion    uint64 `json:"perfil_version"`
		PersonaRef       string `json:"persona_ref"`
		PersonaVersion   uint64 `json:"persona_version"`
		VinculoRef       string `json:"vinculo_ref"`
		VinculoVersion   uint64 `json:"vinculo_version"`
	}{"vec.personal.vinculo-propio-crn11.consulta.v1", s.EmpleadoRef, actor.Principal.ID,
		actor.Instantanea.VinculoRef, actor.Instantanea.VinculoVersion, actor.Instantanea.CuentaRef,
		actor.Instantanea.CuentaVersion, actor.PerfilActivoRef, actor.Instantanea.PerfilVersion,
		actor.PersonaRef, actor.Instantanea.PersonaVersion, proyeccion.VinculoRef, proyeccion.Version})
	if err != nil {
		return MaterialVinculoPropioCRN11{}, ErrVinculoCRN11Invalido
	}
	suma := sha256.Sum256(canonico)
	recurso := core.RecursoAutorizable{Referencia: s.EmpleadoRef, ModuloID: "personal", Tipo: TipoRecursoVinculoPropioCRN11,
		Ambitos:   map[string]string{"empleado_ref": s.EmpleadoRef},
		Atributos: map[string]string{"operacion": "vinculo_propio_historico_crn11", "material_sha256": hex.EncodeToString(suma[:])}}
	if _, err := recurso.HuellaContextoAutorizacionSHA256(); err != nil {
		return MaterialVinculoPropioCRN11{}, ErrVinculoCRN11Invalido
	}
	return MaterialVinculoPropioCRN11{actor: actor, empleadoRef: s.EmpleadoRef, proyeccionRef: proyeccion.VinculoRef, proyeccionVersion: proyeccion.Version, canonico: canonico, recurso: recurso}, nil
}

// Actor devuelve una copia del actor ya validado al construir el material.
func (m MaterialVinculoPropioCRN11) Actor() core.ContextoActor {
	actor := m.actor
	if m.actor.Instantanea.Vinculos != nil {
		actor.Instantanea.Vinculos = append([]core.VinculoReferenciaContextoActor{}, m.actor.Instantanea.Vinculos...)
	}
	if m.actor.Principal.Roles != nil {
		actor.Principal.Roles = append([]string{}, m.actor.Principal.Roles...)
	}
	if m.actor.Principal.Permissions != nil {
		actor.Principal.Permissions = append([]string{}, m.actor.Principal.Permissions...)
	}
	if m.actor.Principal.Attributes != nil {
		actor.Principal.Attributes = copiarMapaRelacion(m.actor.Principal.Attributes)
	}
	return actor
}
func (m MaterialVinculoPropioCRN11) EmpleadoRef() string    { return m.empleadoRef }
func (m MaterialVinculoPropioCRN11) VinculoRef() string     { return m.proyeccionRef }
func (m MaterialVinculoPropioCRN11) VinculoVersion() uint64 { return m.proyeccionVersion }
func (m MaterialVinculoPropioCRN11) Canonico() []byte       { return append([]byte(nil), m.canonico...) }
func (m MaterialVinculoPropioCRN11) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = copiarMapaRelacion(r.Ambitos)
	r.Atributos = copiarMapaRelacion(r.Atributos)
	return r
}
func (m MaterialVinculoPropioCRN11) HuellaSHA256() (string, error) {
	return m.recurso.HuellaContextoAutorizacionSHA256()
}

// VigenteParaLectura comprueba la ventana del contexto actual, no la vigencia
// laboral. La autoridad V3 y su consumo se revalidan en los adaptadores comunes.
func (m MaterialVinculoPropioCRN11) VigenteParaLectura(instante time.Time) bool {
	if instante.IsZero() || instante.Before(m.actor.ResueltoEn) || !m.actor.Instantanea.VigenteEn(instante) {
		return false
	}
	for _, v := range m.actor.Instantanea.Vinculos {
		if v.Tipo == core.TipoReferenciaContextoActorEmpleado && v.Referencia == m.empleadoRef {
			return v.VigenteEn(instante)
		}
	}
	return false
}

// VinculoHistoricoCRN11 procede de proyeccion_empleado_persona_historia.
// Su versión y procedencia acreditan la pareja permanente; no empleo vigente.
type VinculoHistoricoCRN11 struct {
	PersonaRef  string `json:"persona_ref"`
	EmpleadoRef string `json:"empleado_ref"`
	VinculoRef  string `json:"vinculo_ref"`
	FuenteRef   string `json:"fuente_ref"`
	Version     uint64 `json:"version"`
}

func (v VinculoHistoricoCRN11) ValidarPara(m MaterialVinculoPropioCRN11) error {
	if v.VinculoRef != m.proyeccionRef || v.Version != m.proyeccionVersion || v.PersonaRef != m.actor.PersonaRef || v.EmpleadoRef != m.empleadoRef || !ReferenciaPersonaValida(v.PersonaRef) || !ReferenciaEmpleadoValida(v.EmpleadoRef) ||
		!patronProyeccionCRN11.MatchString(v.VinculoRef) || !patronProcedenciaCRN11.MatchString(v.FuenteRef) || v.Version == 0 || v.Version > math.MaxInt64 {
		return ErrVinculoCRN11Invalido
	}
	return nil
}
