package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"

	core "vec-diputacion-granada/internal/vec/domain"
)

var ErrSolicitudRectificacionDietasInvalida = errors.New("personal: solicitud de rectificacion de dietas invalida")

type OperacionRectificacionDietas string

const (
	SolicitarRectificacionDietas OperacionRectificacionDietas = "solicitar"
	ConsultarRectificacionDietas OperacionRectificacionDietas = "consultar"
	ConfirmarRectificacionDietas OperacionRectificacionDietas = "confirmar"
	RechazarRectificacionDietas  OperacionRectificacionDietas = "rechazar"
)

var referenciaSolicitudRectificacion = regexp.MustCompile(`^srd_[0-9a-f]{32}$`)
var referenciaPersonaEnTextoRectificacion = regexp.MustCompile(`per_[A-Za-z0-9_-]{22,128}`)

// CamposARevisar expresa la petición del empleado. Ninguna referencia nueva de
// validador se acepta aquí: sólo la fachada gobernada de asignación puede fijarla.
type SolicitudRectificacionDietas struct {
	Actor             core.ContextoActor
	Operacion         OperacionRectificacionDietas
	PersonaRef        string
	EmpleadoRef       string
	RelacionRef       string
	UnidadRef         string
	AsignacionRef     string
	VersionEsperada   int64
	FechaReferencia   FechaCivil
	ClaveIdempotencia string
	SolicitudRef      string
	CamposARevisar    []string
	MotivoRevision    string
	DetalleSolicitado string
	Correccion        *SolicitudAsignacionDietas
}

type MaterialRectificacionDietas struct {
	solicitud SolicitudRectificacionDietas
	canonico  []byte
	recurso   core.RecursoAutorizable
}

func (m MaterialRectificacionDietas) Canonico() []byte { return append([]byte(nil), m.canonico...) }
func (m MaterialRectificacionDietas) Solicitud() SolicitudRectificacionDietas {
	s := m.solicitud
	a, err := s.Actor.Clonar()
	if err != nil {
		return SolicitudRectificacionDietas{}
	}
	s.Actor = a
	s.CamposARevisar = append([]string(nil), s.CamposARevisar...)
	if s.Correccion != nil {
		c := *s.Correccion
		c.Actor, err = c.Actor.Clonar()
		if err != nil {
			return SolicitudRectificacionDietas{}
		}
		s.Correccion = &c
	}
	return s
}
func (m MaterialRectificacionDietas) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = copiarMapaRelacion(r.Ambitos)
	r.Atributos = copiarMapaRelacion(r.Atributos)
	return r
}
func (m MaterialRectificacionDietas) HuellaSHA256() (string, error) {
	return m.recurso.HuellaContextoAutorizacionSHA256()
}

type rectificacionDietasCanonica struct {
	Esquema           string                                  `json:"esquema"`
	Identidad         identidadConsultaRelacionPropiaCanonica `json:"identidad"`
	Operacion         OperacionRectificacionDietas            `json:"operacion"`
	PersonaRef        string                                  `json:"persona_ref"`
	EmpleadoRef       string                                  `json:"empleado_ref"`
	RelacionRef       string                                  `json:"relacion_ref"`
	UnidadRef         string                                  `json:"unidad_ref"`
	AsignacionRef     string                                  `json:"asignacion_ref"`
	VersionEsperada   int64                                   `json:"version_esperada"`
	FechaReferencia   FechaCivil                              `json:"fecha_referencia"`
	ClaveIdempotencia string                                  `json:"clave_idempotencia"`
	SolicitudRef      string                                  `json:"solicitud_ref"`
	CamposARevisar    []string                                `json:"campos_a_revisar"`
	MotivoRevision    string                                  `json:"motivo_revision"`
	DetalleSolicitado string                                  `json:"detalle_solicitado"`
}

func NuevoMaterialRectificacionDietas(s SolicitudRectificacionDietas) (MaterialRectificacionDietas, error) {
	if s.Actor.Validar() != nil || !ReferenciaRelacionValida(s.RelacionRef) ||
		!textoAsignacionValido(s.UnidadRef, 256, 1) || s.FechaReferencia.Validar() != nil {
		return MaterialRectificacionDietas{}, ErrSolicitudRectificacionDietasInvalida
	}
	empleados, err := s.Actor.Referencias("empleado")
	if err != nil || len(empleados) != 1 {
		return MaterialRectificacionDietas{}, ErrSolicitudRectificacionDietasInvalida
	}
	persona, empleado, actorEmpleado := s.Actor.PersonaRef, empleados[0], empleados[0]
	if s.Operacion == ConsultarRectificacionDietas {
		if s.PersonaRef != "" || s.EmpleadoRef != "" {
			return MaterialRectificacionDietas{}, ErrSolicitudRectificacionDietasInvalida
		}
		if s.AsignacionRef != "" || s.VersionEsperada != 0 || s.ClaveIdempotencia != "" ||
			s.SolicitudRef != "" || len(s.CamposARevisar) != 0 || s.MotivoRevision != "" || s.DetalleSolicitado != "" {
			return MaterialRectificacionDietas{}, ErrSolicitudRectificacionDietasInvalida
		}
	} else if s.Operacion == SolicitarRectificacionDietas {
		if s.PersonaRef != "" || s.EmpleadoRef != "" {
			return MaterialRectificacionDietas{}, ErrSolicitudRectificacionDietasInvalida
		}
		s.CamposARevisar = append([]string(nil), s.CamposARevisar...)
		sort.Strings(s.CamposARevisar)
		if !referenciaAsignacionDietas.MatchString(s.AsignacionRef) || s.VersionEsperada < 1 ||
			!claveAsignacionValida(s.ClaveIdempotencia) || s.SolicitudRef != "" ||
			!textoAsignacionValido(s.MotivoRevision, 500, 3) || referenciaPersonaEnTextoRectificacion.MatchString(s.MotivoRevision) ||
			!detalleRectificacionValido(s.DetalleSolicitado) ||
			!camposRectificacionValidos(s.CamposARevisar) {
			return MaterialRectificacionDietas{}, ErrSolicitudRectificacionDietasInvalida
		}
	} else if s.Operacion == ConfirmarRectificacionDietas || s.Operacion == RechazarRectificacionDietas {
		if !ReferenciaPersonaValida(s.PersonaRef) || !ReferenciaEmpleadoValida(s.EmpleadoRef) ||
			s.PersonaRef == persona || s.EmpleadoRef == empleado {
			return MaterialRectificacionDietas{}, ErrSolicitudRectificacionDietasInvalida
		}
		persona, empleado = s.PersonaRef, s.EmpleadoRef
		if !referenciaSolicitudRectificacion.MatchString(s.SolicitudRef) || !claveAsignacionValida(s.ClaveIdempotencia) ||
			!textoAsignacionValido(s.MotivoRevision, 500, 3) || referenciaPersonaEnTextoRectificacion.MatchString(s.MotivoRevision) ||
			s.DetalleSolicitado != "" || len(s.CamposARevisar) != 0 {
			return MaterialRectificacionDietas{}, ErrSolicitudRectificacionDietasInvalida
		}
		if s.Operacion == ConfirmarRectificacionDietas {
			if !referenciaAsignacionDietas.MatchString(s.AsignacionRef) || s.VersionEsperada < 1 ||
				s.Correccion == nil || !correccionRectificacionLigada(s) {
				return MaterialRectificacionDietas{}, ErrSolicitudRectificacionDietasInvalida
			}
		} else if s.AsignacionRef != "" || s.VersionEsperada != 0 || s.Correccion != nil {
			return MaterialRectificacionDietas{}, ErrSolicitudRectificacionDietasInvalida
		}
	} else {
		return MaterialRectificacionDietas{}, ErrSolicitudRectificacionDietasInvalida
	}
	a := s.Actor
	i := identidadConsultaRelacionPropiaCanonica{ActorRef: a.Principal.ID, ContextoActorRef: a.Instantanea.VinculoRef,
		ContextoVersion: a.Instantanea.VinculoVersion, CuentaRef: a.Instantanea.CuentaRef,
		CuentaVersion: a.Instantanea.CuentaVersion, EmpleadoRef: actorEmpleado,
		PerfilRef: a.PerfilActivoRef, PerfilVersion: a.Instantanea.PerfilVersion,
		PersonaRef: persona, PersonaVersion: a.Instantanea.PersonaVersion}
	campos := append([]string(nil), s.CamposARevisar...)
	if campos == nil {
		campos = []string{}
	}
	c := rectificacionDietasCanonica{Esquema: "vec.personal.rectificacion-dietas.v1", Identidad: i,
		Operacion: s.Operacion, PersonaRef: persona, EmpleadoRef: empleado, RelacionRef: s.RelacionRef,
		UnidadRef: s.UnidadRef, AsignacionRef: s.AsignacionRef, VersionEsperada: s.VersionEsperada,
		FechaReferencia: s.FechaReferencia, ClaveIdempotencia: s.ClaveIdempotencia, SolicitudRef: s.SolicitudRef,
		CamposARevisar: campos, MotivoRevision: s.MotivoRevision, DetalleSolicitado: s.DetalleSolicitado}
	b, err := json.Marshal(c)
	if err != nil {
		return MaterialRectificacionDietas{}, ErrSolicitudRectificacionDietasInvalida
	}
	h := sha256.Sum256(b)
	r := core.RecursoAutorizable{Referencia: s.RelacionRef, ModuloID: "personal", Tipo: "rectificacion_asignacion_dietas",
		Ambitos:   map[string]string{"persona_ref": persona, "empleado_ref": empleado, "relacion_ref": s.RelacionRef, "unidad_ref": s.UnidadRef},
		Atributos: map[string]string{"fecha_referencia": s.FechaReferencia.Texto(), "material_sha256": hex.EncodeToString(h[:]), "operacion": string(s.Operacion)}}
	if _, err = r.HuellaContextoAutorizacionSHA256(); err != nil {
		return MaterialRectificacionDietas{}, ErrSolicitudRectificacionDietasInvalida
	}
	actor, err := a.Clonar()
	if err != nil {
		return MaterialRectificacionDietas{}, ErrSolicitudRectificacionDietasInvalida
	}
	s.Actor = actor
	s.CamposARevisar = campos
	if s.Correccion != nil {
		c := *s.Correccion
		c.Actor, err = c.Actor.Clonar()
		if err != nil {
			return MaterialRectificacionDietas{}, ErrSolicitudRectificacionDietasInvalida
		}
		s.Correccion = &c
	}
	return MaterialRectificacionDietas{solicitud: s, canonico: b, recurso: r}, nil
}

func correccionRectificacionLigada(s SolicitudRectificacionDietas) bool {
	c := s.Correccion
	if c == nil || c.Actor.Validar() != nil {
		return false
	}
	actor, e1 := c.Actor.RepresentacionCanonicaVinculadaV2()
	propio, e2 := s.Actor.RepresentacionCanonicaVinculadaV2()
	return e1 == nil && e2 == nil && string(actor) == string(propio) &&
		c.Operacion == CorregirAsignacionDietas && c.PersonaRef == s.PersonaRef &&
		c.EmpleadoRef == s.EmpleadoRef && c.RelacionRef == s.RelacionRef &&
		c.UnidadRef == s.UnidadRef && c.FechaReferencia == s.FechaReferencia &&
		c.VersionEsperada == s.VersionEsperada && c.ClaveIdempotencia == s.ClaveIdempotencia &&
		c.MotivoRevision == s.MotivoRevision && c.ProcedenciaActoRef == s.SolicitudRef
}

func camposRectificacionValidos(campos []string) bool {
	if len(campos) < 1 || len(campos) > 4 || !sort.StringsAreSorted(campos) {
		return false
	}
	previo := ""
	for _, campo := range campos {
		if campo == previo || (campo != "centro_ref" && campo != "unidad_ref" && campo != "administrativo_persona_ref" && campo != "responsable_persona_ref") {
			return false
		}
		previo = campo
	}
	return true
}

func detalleRectificacionValido(s string) bool {
	if len(s) > 500 || (len(s) > 0 && len(s) < 3) || strings.TrimSpace(s) != s || referenciaPersonaEnTextoRectificacion.MatchString(s) {
		return false
	}
	for _, r := range s {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}
