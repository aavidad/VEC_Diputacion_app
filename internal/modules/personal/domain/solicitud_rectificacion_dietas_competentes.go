package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	core "vec-diputacion-granada/internal/vec/domain"
)

// Esta consulta enumera únicamente desde la competencia vigente que resuelve
// Personal; el cliente no indica persona, relación, unidad ni solicitud.
type SolicitudRectificacionesCompetentesDietas struct {
	Actor           core.ContextoActor
	FechaReferencia FechaCivil
}

type MaterialRectificacionesCompetentesDietas struct {
	solicitud SolicitudRectificacionesCompetentesDietas
	canonico  []byte
	recurso   core.RecursoAutorizable
}

func (m MaterialRectificacionesCompetentesDietas) Canonico() []byte {
	return append([]byte(nil), m.canonico...)
}
func (m MaterialRectificacionesCompetentesDietas) Solicitud() SolicitudRectificacionesCompetentesDietas {
	s := m.solicitud
	a, err := s.Actor.Clonar()
	if err != nil {
		return SolicitudRectificacionesCompetentesDietas{}
	}
	s.Actor = a
	return s
}
func (m MaterialRectificacionesCompetentesDietas) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = copiarMapaRelacion(r.Ambitos)
	r.Atributos = copiarMapaRelacion(r.Atributos)
	return r
}
func (m MaterialRectificacionesCompetentesDietas) HuellaSHA256() (string, error) {
	return m.recurso.HuellaContextoAutorizacionSHA256()
}

type rectificacionesCompetentesCanonico struct {
	Esquema         string                                  `json:"esquema"`
	FechaReferencia FechaCivil                              `json:"fecha_referencia"`
	Identidad       identidadConsultaRelacionPropiaCanonica `json:"identidad"`
}

func NuevoMaterialRectificacionesCompetentesDietas(s SolicitudRectificacionesCompetentesDietas) (MaterialRectificacionesCompetentesDietas, error) {
	if s.Actor.Validar() != nil || s.FechaReferencia.Validar() != nil {
		return MaterialRectificacionesCompetentesDietas{}, ErrSolicitudRectificacionDietasInvalida
	}
	empleados, err := s.Actor.Referencias("empleado")
	if err != nil || len(empleados) != 1 {
		return MaterialRectificacionesCompetentesDietas{}, ErrSolicitudRectificacionDietasInvalida
	}
	i := identidadConsultaRelacionPropiaCanonica{
		ActorRef: s.Actor.Principal.ID, ContextoActorRef: s.Actor.Instantanea.VinculoRef,
		ContextoVersion: s.Actor.Instantanea.VinculoVersion, CuentaRef: s.Actor.Instantanea.CuentaRef,
		CuentaVersion: s.Actor.Instantanea.CuentaVersion, EmpleadoRef: empleados[0],
		PerfilRef: s.Actor.PerfilActivoRef, PerfilVersion: s.Actor.Instantanea.PerfilVersion,
		PersonaRef: s.Actor.PersonaRef, PersonaVersion: s.Actor.Instantanea.PersonaVersion,
	}
	b, err := json.Marshal(rectificacionesCompetentesCanonico{
		Esquema: "vec.personal.rectificaciones-dietas.competentes.v1", FechaReferencia: s.FechaReferencia, Identidad: i,
	})
	if err != nil {
		return MaterialRectificacionesCompetentesDietas{}, ErrSolicitudRectificacionDietasInvalida
	}
	h := sha256.Sum256(b)
	r := core.RecursoAutorizable{
		Referencia: s.Actor.PersonaRef, ModuloID: "personal", Tipo: "rectificaciones_competentes_dietas",
		Ambitos:   map[string]string{"persona_ref": s.Actor.PersonaRef},
		Atributos: map[string]string{"fecha_referencia": s.FechaReferencia.Texto(), "material_sha256": hex.EncodeToString(h[:]), "operacion": "lista"},
	}
	if _, err = r.HuellaContextoAutorizacionSHA256(); err != nil {
		return MaterialRectificacionesCompetentesDietas{}, ErrSolicitudRectificacionDietasInvalida
	}
	actor, err := s.Actor.Clonar()
	if err != nil {
		return MaterialRectificacionesCompetentesDietas{}, ErrSolicitudRectificacionDietasInvalida
	}
	s.Actor = actor
	return MaterialRectificacionesCompetentesDietas{solicitud: s, canonico: b, recurso: r}, nil
}
