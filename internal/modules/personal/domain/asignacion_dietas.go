package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	core "vec-diputacion-granada/internal/vec/domain"
)

var ErrAsignacionDietasInvalida = errors.New("personal: asignacion de dietas invalida")

var referenciaAsignacionDietas = regexp.MustCompile(`^ads_[A-Za-z0-9_-]{22,128}$`)

type OperacionAsignacionDietas string

const (
	ConsultarAsignacionDietas        OperacionAsignacionDietas = "consultar"
	RegistrarInicialAsignacionDietas OperacionAsignacionDietas = "registrar_inicial"
	CorregirAsignacionDietas         OperacionAsignacionDietas = "corregir"
	CorregirGrupoAsignacionDietas    OperacionAsignacionDietas = "grupo_corregir"
)

// La identidad viene de la frontera autenticada. Una referencia enviada por el
// cliente sólo selecciona la relación; nunca acredita titularidad ni permiso.
// UnidadRef debe coincidir con la relación canónica de Personal. Mover a otra
// unidad requiere antes el circuito propio de cambio de relación.
type SolicitudAsignacionDietas struct {
	Actor                    core.ContextoActor
	PersonaRef               string
	EmpleadoRef              string
	RelacionRef              string
	FechaReferencia          FechaCivil
	Operacion                OperacionAsignacionDietas
	ClaveIdempotencia        string
	VersionEsperada          int64
	CentroRef                string
	UnidadRef                string
	AdministrativoPersonaRef string
	ResponsablePersonaRef    string
	GrupoDieta               int16
	VigenteDesde             FechaCivil
	MotivoRevision           string
	ProcedenciaActoRef       string
}

type AsignacionDietas struct {
	AsignacionRef            string     `json:"asignacion_ref"`
	RelacionRef              string     `json:"relacion_ref"`
	PersonaRef               string     `json:"persona_ref"`
	UnidadRef                string     `json:"unidad_ref"`
	CentroRef                string     `json:"centro_ref"`
	AdministrativoPersonaRef string     `json:"administrativo_persona_ref"`
	ResponsablePersonaRef    string     `json:"responsable_persona_ref"`
	GrupoDieta               int16      `json:"grupo_dieta"`
	VigenteDesde             FechaCivil `json:"vigente_desde"`
	Version                  int64      `json:"version"`
}

func (a AsignacionDietas) Validar() error {
	if !referenciaAsignacionDietas.MatchString(a.AsignacionRef) || !ReferenciaRelacionValida(a.RelacionRef) ||
		!ReferenciaPersonaValida(a.PersonaRef) || !ReferenciaPersonaValida(a.AdministrativoPersonaRef) ||
		!ReferenciaPersonaValida(a.ResponsablePersonaRef) || a.PersonaRef == a.AdministrativoPersonaRef ||
		a.PersonaRef == a.ResponsablePersonaRef || a.AdministrativoPersonaRef == a.ResponsablePersonaRef ||
		!textoAsignacionValido(a.UnidadRef, 256, 1) || !textoAsignacionValido(a.CentroRef, 160, 1) ||
		a.GrupoDieta < 1 || a.GrupoDieta > 3 || a.VigenteDesde.Validar() != nil || a.Version < 1 {
		return ErrAsignacionDietasInvalida
	}
	return nil
}

func textoAsignacionValido(s string, max, min int) bool {
	if len(s) < min || len(s) > max || strings.TrimSpace(s) != s {
		return false
	}
	for _, r := range s {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}

type MaterialAsignacionDietas struct {
	solicitud SolicitudAsignacionDietas
	canonico  []byte
	recurso   core.RecursoAutorizable
}

func (m MaterialAsignacionDietas) Canonico() []byte { return append([]byte(nil), m.canonico...) }
func (m MaterialAsignacionDietas) Solicitud() SolicitudAsignacionDietas {
	s := m.solicitud
	a, err := s.Actor.Clonar()
	if err != nil {
		return SolicitudAsignacionDietas{}
	}
	s.Actor = a
	return s
}
func (m MaterialAsignacionDietas) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = copiarMapaRelacion(r.Ambitos)
	r.Atributos = copiarMapaRelacion(r.Atributos)
	return r
}
func (m MaterialAsignacionDietas) HuellaSHA256() (string, error) {
	return m.recurso.HuellaContextoAutorizacionSHA256()
}

type materialAsignacionCanonico struct {
	Esquema                  string                                  `json:"esquema"`
	Identidad                identidadConsultaRelacionPropiaCanonica `json:"identidad"`
	PersonaRef               string                                  `json:"persona_ref"`
	EmpleadoRef              string                                  `json:"empleado_ref"`
	Operacion                OperacionAsignacionDietas               `json:"operacion"`
	RelacionRef              string                                  `json:"relacion_ref"`
	FechaReferencia          FechaCivil                              `json:"fecha_referencia"`
	ClaveIdempotencia        string                                  `json:"clave_idempotencia"`
	VersionEsperada          int64                                   `json:"version_esperada"`
	CentroRef                string                                  `json:"centro_ref"`
	UnidadRef                string                                  `json:"unidad_ref"`
	AdministrativoPersonaRef string                                  `json:"administrativo_persona_ref"`
	ResponsablePersonaRef    string                                  `json:"responsable_persona_ref"`
	GrupoDieta               int16                                   `json:"grupo_dieta"`
	VigenteDesde             FechaCivil                              `json:"vigente_desde"`
	MotivoRevision           string                                  `json:"motivo_revision"`
	ProcedenciaActoRef       string                                  `json:"procedencia_acto_ref"`
}

func NuevoMaterialAsignacionDietas(s SolicitudAsignacionDietas) (MaterialAsignacionDietas, error) {
	if s.Actor.Validar() != nil || !ReferenciaRelacionValida(s.RelacionRef) || s.FechaReferencia.Validar() != nil {
		return MaterialAsignacionDietas{}, ErrAsignacionDietasInvalida
	}
	empleados, e := s.Actor.Referencias("empleado")
	if e != nil || len(empleados) != 1 {
		return MaterialAsignacionDietas{}, ErrAsignacionDietasInvalida
	}
	if s.Operacion != ConsultarAsignacionDietas && s.Operacion != RegistrarInicialAsignacionDietas && s.Operacion != CorregirAsignacionDietas && s.Operacion != CorregirGrupoAsignacionDietas {
		return MaterialAsignacionDietas{}, ErrAsignacionDietasInvalida
	}
	if s.Operacion == CorregirGrupoAsignacionDietas || s.Operacion == RegistrarInicialAsignacionDietas || s.Operacion == CorregirAsignacionDietas {
		if !ReferenciaPersonaValida(s.PersonaRef) || !ReferenciaEmpleadoValida(s.EmpleadoRef) {
			return MaterialAsignacionDietas{}, ErrAsignacionDietasInvalida
		}
		if s.PersonaRef == s.Actor.PersonaRef {
			return MaterialAsignacionDietas{}, ErrAsignacionDietasInvalida
		}
	} else {
		if (s.PersonaRef != "" && s.PersonaRef != s.Actor.PersonaRef) || (s.EmpleadoRef != "" && s.EmpleadoRef != empleados[0]) {
			return MaterialAsignacionDietas{}, ErrAsignacionDietasInvalida
		}
		s.PersonaRef, s.EmpleadoRef = s.Actor.PersonaRef, empleados[0]
	}
	if !textoAsignacionValido(s.UnidadRef, 256, 1) {
		return MaterialAsignacionDietas{}, ErrAsignacionDietasInvalida
	}
	if s.Operacion == ConsultarAsignacionDietas {
		if s.ClaveIdempotencia != "" || s.VersionEsperada != 0 || s.CentroRef != "" || s.AdministrativoPersonaRef != "" || s.ResponsablePersonaRef != "" || s.GrupoDieta != 0 || s.VigenteDesde != "" || s.MotivoRevision != "" || s.ProcedenciaActoRef != "" {
			return MaterialAsignacionDietas{}, ErrAsignacionDietasInvalida
		}
	} else if !claveAsignacionValida(s.ClaveIdempotencia) || (s.Operacion == RegistrarInicialAsignacionDietas && s.VersionEsperada != 0) || (s.Operacion != RegistrarInicialAsignacionDietas && s.VersionEsperada < 1) || !textoAsignacionValido(s.CentroRef, 160, 1) || !textoAsignacionValido(s.UnidadRef, 256, 1) || !ReferenciaPersonaValida(s.AdministrativoPersonaRef) || !ReferenciaPersonaValida(s.ResponsablePersonaRef) || s.AdministrativoPersonaRef == s.ResponsablePersonaRef || s.AdministrativoPersonaRef == s.PersonaRef || s.ResponsablePersonaRef == s.PersonaRef || s.GrupoDieta < 1 || s.GrupoDieta > 3 || s.VigenteDesde.Validar() != nil || !textoAsignacionValido(s.MotivoRevision, 500, 3) || !textoAsignacionValido(s.ProcedenciaActoRef, 256, 1) {
		return MaterialAsignacionDietas{}, ErrAsignacionDietasInvalida
	}
	i := identidadConsultaRelacionPropiaCanonica{ActorRef: s.Actor.Principal.ID, ContextoActorRef: s.Actor.Instantanea.VinculoRef, ContextoVersion: s.Actor.Instantanea.VinculoVersion, CuentaRef: s.Actor.Instantanea.CuentaRef, CuentaVersion: s.Actor.Instantanea.CuentaVersion, EmpleadoRef: empleados[0], PerfilRef: s.Actor.PerfilActivoRef, PerfilVersion: s.Actor.Instantanea.PerfilVersion, PersonaRef: s.Actor.PersonaRef, PersonaVersion: s.Actor.Instantanea.PersonaVersion}
	c := materialAsignacionCanonico{Esquema: "vec.personal.asignacion-dietas.v1", Identidad: i, PersonaRef: s.PersonaRef, EmpleadoRef: s.EmpleadoRef, Operacion: s.Operacion, RelacionRef: s.RelacionRef, FechaReferencia: s.FechaReferencia, ClaveIdempotencia: s.ClaveIdempotencia, VersionEsperada: s.VersionEsperada, CentroRef: s.CentroRef, UnidadRef: s.UnidadRef, AdministrativoPersonaRef: s.AdministrativoPersonaRef, ResponsablePersonaRef: s.ResponsablePersonaRef, GrupoDieta: s.GrupoDieta, VigenteDesde: s.VigenteDesde, MotivoRevision: s.MotivoRevision, ProcedenciaActoRef: s.ProcedenciaActoRef}
	b, e := json.Marshal(c)
	if e != nil {
		return MaterialAsignacionDietas{}, ErrAsignacionDietasInvalida
	}
	h := sha256.Sum256(b)
	r := core.RecursoAutorizable{Referencia: s.RelacionRef, ModuloID: "personal", Tipo: "asignacion_dietas", Ambitos: map[string]string{"persona_ref": s.PersonaRef, "empleado_ref": s.EmpleadoRef, "relacion_ref": s.RelacionRef, "unidad_ref": s.UnidadRef}, Atributos: map[string]string{"fecha_referencia": s.FechaReferencia.Texto(), "material_sha256": hex.EncodeToString(h[:]), "operacion": string(s.Operacion)}}
	if _, e = r.HuellaContextoAutorizacionSHA256(); e != nil {
		return MaterialAsignacionDietas{}, ErrAsignacionDietasInvalida
	}
	actor, e := s.Actor.Clonar()
	if e != nil {
		return MaterialAsignacionDietas{}, ErrAsignacionDietasInvalida
	}
	s.Actor = actor
	return MaterialAsignacionDietas{solicitud: s, canonico: b, recurso: r}, nil
}

var patronClaveAsignacion = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func claveAsignacionValida(s string) bool { return patronClaveAsignacion.MatchString(s) }

// Vigencia y versión se comprueban de nuevo en PostgreSQL en la transacción
// que consume la capacidad. Este sello sólo valida la forma de la proyección.
func (a AsignacionDietas) VigenteEn(f FechaCivil) bool { return !f.AntesDe(a.VigenteDesde) }
