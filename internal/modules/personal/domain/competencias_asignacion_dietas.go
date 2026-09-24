package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	core "vec-diputacion-granada/internal/vec/domain"
)

var ErrCompetenciasAsignacionDietasInvalidas = errors.New("personal: competencias de asignacion de dietas invalidas")

// La selección de asignaciones la hace Personal a partir de la identidad V2.
// La solicitud no admite referencias de otras personas ni unidades del cliente.
type SolicitudCompetenciasAsignacionDietas struct {
	Actor           core.ContextoActor
	FechaReferencia FechaCivil
}

type CompetenciaAsignacionDietas struct {
	AsignacionRef string     `json:"asignacion_ref"`
	RelacionRef   string     `json:"relacion_ref"`
	UnidadRef     string     `json:"unidad_ref"`
	Rol           string     `json:"rol"`
	VigenteDesde  FechaCivil `json:"vigente_desde"`
	Version       int64      `json:"version"`
}

func (c CompetenciaAsignacionDietas) Validar(fecha FechaCivil) error {
	if !referenciaAsignacionDietas.MatchString(c.AsignacionRef) || !ReferenciaRelacionValida(c.RelacionRef) ||
		!textoAsignacionValido(c.UnidadRef, 256, 1) || (c.Rol != "administrativo" && c.Rol != "responsable") ||
		c.VigenteDesde.Validar() != nil || fecha.Validar() != nil || fecha.AntesDe(c.VigenteDesde) || c.Version < 1 {
		return ErrCompetenciasAsignacionDietasInvalidas
	}
	return nil
}

type MaterialCompetenciasAsignacionDietas struct {
	solicitud SolicitudCompetenciasAsignacionDietas
	canonico  []byte
	recurso   core.RecursoAutorizable
}

func (m MaterialCompetenciasAsignacionDietas) Canonico() []byte {
	return append([]byte(nil), m.canonico...)
}
func (m MaterialCompetenciasAsignacionDietas) Solicitud() SolicitudCompetenciasAsignacionDietas {
	s := m.solicitud
	a, err := s.Actor.Clonar()
	if err != nil {
		return SolicitudCompetenciasAsignacionDietas{}
	}
	s.Actor = a
	return s
}
func (m MaterialCompetenciasAsignacionDietas) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = copiarMapaRelacion(r.Ambitos)
	r.Atributos = copiarMapaRelacion(r.Atributos)
	return r
}
func (m MaterialCompetenciasAsignacionDietas) HuellaSHA256() (string, error) {
	return m.recurso.HuellaContextoAutorizacionSHA256()
}

type materialCompetenciasAsignacionDietasCanonico struct {
	Esquema         string                                  `json:"esquema"`
	FechaReferencia FechaCivil                              `json:"fecha_referencia"`
	Identidad       identidadConsultaRelacionPropiaCanonica `json:"identidad"`
}

func NuevoMaterialCompetenciasAsignacionDietas(s SolicitudCompetenciasAsignacionDietas) (MaterialCompetenciasAsignacionDietas, error) {
	if s.Actor.Validar() != nil || s.FechaReferencia.Validar() != nil {
		return MaterialCompetenciasAsignacionDietas{}, ErrCompetenciasAsignacionDietasInvalidas
	}
	empleados, err := s.Actor.Referencias("empleado")
	if err != nil || len(empleados) != 1 {
		return MaterialCompetenciasAsignacionDietas{}, ErrCompetenciasAsignacionDietasInvalidas
	}
	i := identidadConsultaRelacionPropiaCanonica{
		ActorRef: s.Actor.Principal.ID, ContextoActorRef: s.Actor.Instantanea.VinculoRef,
		ContextoVersion: s.Actor.Instantanea.VinculoVersion, CuentaRef: s.Actor.Instantanea.CuentaRef,
		CuentaVersion: s.Actor.Instantanea.CuentaVersion, EmpleadoRef: empleados[0],
		PerfilRef: s.Actor.PerfilActivoRef, PerfilVersion: s.Actor.Instantanea.PerfilVersion,
		PersonaRef: s.Actor.PersonaRef, PersonaVersion: s.Actor.Instantanea.PersonaVersion,
	}
	b, err := json.Marshal(materialCompetenciasAsignacionDietasCanonico{
		Esquema: "vec.personal.asignacion-dietas.competencias.v1", FechaReferencia: s.FechaReferencia, Identidad: i,
	})
	if err != nil {
		return MaterialCompetenciasAsignacionDietas{}, ErrCompetenciasAsignacionDietasInvalidas
	}
	h := sha256.Sum256(b)
	r := core.RecursoAutorizable{
		Referencia: s.Actor.PersonaRef, ModuloID: "personal", Tipo: "asignacion_dietas_competencias",
		Ambitos:   map[string]string{"persona_ref": s.Actor.PersonaRef},
		Atributos: map[string]string{"fecha_referencia": s.FechaReferencia.Texto(), "material_sha256": hex.EncodeToString(h[:]), "operacion": "lista"},
	}
	if _, err = r.HuellaContextoAutorizacionSHA256(); err != nil {
		return MaterialCompetenciasAsignacionDietas{}, ErrCompetenciasAsignacionDietasInvalidas
	}
	a, err := s.Actor.Clonar()
	if err != nil {
		return MaterialCompetenciasAsignacionDietas{}, ErrCompetenciasAsignacionDietasInvalidas
	}
	s.Actor = a
	return MaterialCompetenciasAsignacionDietas{solicitud: s, canonico: b, recurso: r}, nil
}
