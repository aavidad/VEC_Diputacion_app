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

const (
	FinalidadLectorRelacionRPT   = "conciliar_relacion_laboral_para_rpt"
	TipoRecursoLectorRelacionRPT = "relacion_para_rpt"
)

var (
	ErrLectorRelacionRPTInvalido     = errors.New("personal: lectura de relacion RPT invalida")
	ErrLectorRelacionRPTDenegado     = errors.New("personal: lectura de relacion RPT denegada")
	ErrLectorRelacionRPTNoDisponible = errors.New("personal: lectura de relacion RPT no disponible")
	organismoLectorRelacionRPT       = regexp.MustCompile(`^[a-z][a-z0-9_:-]{2,127}$`)
)

// SolicitudLectorRelacionRPT conserva al actor efectivo. El servidor fija la
// relación seleccionada y su corte; el empleado objetivo puede ser otra persona.
// Esa selección nunca concede permiso: necesita una autorización RPT propia.
type SolicitudLectorRelacionRPT struct {
	Actor                                  core.ContextoActor
	EmpleadoRef, RelacionRef, OrganismoRef string
	VersionEsperada                        int64
	Corte                                  CorteEmpleadoB2
}

type MaterialLectorRelacionRPT struct {
	solicitud SolicitudLectorRelacionRPT
	canonico  []byte
	recurso   core.RecursoAutorizable
}

// El orden de estos 17 campos es el canon que reconstruye Personal27.
// ConocidoEn conserva el instante histórico; la vigencia del actor se revalida
// con el reloj actual, independientemente de ese corte solicitado.
type canonLectorRelacionRPT struct {
	Esquema          string `json:"esquema"`
	Operacion        string `json:"operacion"`
	EmpleadoRef      string `json:"empleado_ref"`
	RelacionRef      string `json:"relacion_ref"`
	OrganismoRef     string `json:"organismo_ref"`
	VersionEsperada  int64  `json:"version_esperada"`
	VigenteEn        string `json:"vigente_en"`
	ConocidoEn       string `json:"conocido_en"`
	ActorRef         string `json:"actor_ref"`
	ContextoActorRef string `json:"contexto_actor_ref"`
	ContextoVersion  uint64 `json:"contexto_version"`
	CuentaRef        string `json:"cuenta_ref"`
	CuentaVersion    uint64 `json:"cuenta_version"`
	PerfilRef        string `json:"perfil_ref"`
	PerfilVersion    uint64 `json:"perfil_version"`
	PersonaRef       string `json:"persona_ref"`
	PersonaVersion   uint64 `json:"persona_version"`
}

func NuevoMaterialLectorRelacionRPT(s SolicitudLectorRelacionRPT) (MaterialLectorRelacionRPT, error) {
	if s.Actor.Validar() != nil || !ReferenciaEmpleadoValida(s.EmpleadoRef) || !ReferenciaRelacionValida(s.RelacionRef) ||
		!organismoLectorRelacionRPT.MatchString(s.OrganismoRef) || s.VersionEsperada < 1 || s.Corte.Validar() != nil {
		return MaterialLectorRelacionRPT{}, ErrLectorRelacionRPTInvalido
	}
	actor, err := s.Actor.Clonar()
	if err != nil {
		return MaterialLectorRelacionRPT{}, ErrLectorRelacionRPTInvalido
	}
	s.Actor = actor
	conocido := s.Corte.ConocidoEn.UTC().Format("2006-01-02T15:04:05.000000Z")
	canonico, err := json.Marshal(canonLectorRelacionRPT{
		Esquema: "vec.personal.relacion-rpt.consulta.v1", Operacion: "relacion_para_rpt",
		EmpleadoRef: s.EmpleadoRef, RelacionRef: s.RelacionRef, OrganismoRef: s.OrganismoRef,
		VersionEsperada: s.VersionEsperada, VigenteEn: s.Corte.VigenteEn.Texto(), ConocidoEn: conocido,
		ActorRef: actor.Principal.ID, ContextoActorRef: actor.Instantanea.VinculoRef, ContextoVersion: actor.Instantanea.VinculoVersion,
		CuentaRef: actor.Instantanea.CuentaRef, CuentaVersion: actor.Instantanea.CuentaVersion,
		PerfilRef: actor.PerfilActivoRef, PerfilVersion: actor.Instantanea.PerfilVersion,
		PersonaRef: actor.PersonaRef, PersonaVersion: actor.Instantanea.PersonaVersion,
	})
	if err != nil {
		return MaterialLectorRelacionRPT{}, ErrLectorRelacionRPTInvalido
	}
	h := sha256.Sum256(canonico)
	recurso := core.RecursoAutorizable{Referencia: s.RelacionRef, ModuloID: "personal", Tipo: TipoRecursoLectorRelacionRPT,
		Ambitos:   map[string]string{"empleado_ref": s.EmpleadoRef, "organismo_ref": s.OrganismoRef, "relacion_ref": s.RelacionRef},
		Atributos: map[string]string{"conocido_en": conocido, "material_sha256": hex.EncodeToString(h[:]), "operacion": "relacion_para_rpt", "version_esperada": strconv.FormatInt(s.VersionEsperada, 10), "vigente_en": s.Corte.VigenteEn.Texto()},
	}
	if _, err := recurso.HuellaContextoAutorizacionSHA256(); err != nil {
		return MaterialLectorRelacionRPT{}, ErrLectorRelacionRPTInvalido
	}
	return MaterialLectorRelacionRPT{solicitud: s, canonico: canonico, recurso: recurso}, nil
}
func (m MaterialLectorRelacionRPT) Actor() core.ContextoActor {
	a := m.solicitud.Actor
	if a.Instantanea.Vinculos != nil {
		a.Instantanea.Vinculos = append([]core.VinculoReferenciaContextoActor{}, a.Instantanea.Vinculos...)
	}
	if a.Principal.Roles != nil {
		a.Principal.Roles = append([]string{}, a.Principal.Roles...)
	}
	if a.Principal.Permissions != nil {
		a.Principal.Permissions = append([]string{}, a.Principal.Permissions...)
	}
	if a.Principal.Attributes != nil {
		a.Principal.Attributes = copiarMapaRelacion(a.Principal.Attributes)
	}
	return a
}
func (m MaterialLectorRelacionRPT) Solicitud() SolicitudLectorRelacionRPT {
	s := m.solicitud
	s.Actor = m.Actor()
	return s
}
func (m MaterialLectorRelacionRPT) Corte() CorteEmpleadoB2 { return m.solicitud.Corte }
func (m MaterialLectorRelacionRPT) Canonico() []byte       { return append([]byte(nil), m.canonico...) }
func (m MaterialLectorRelacionRPT) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = copiarMapaRelacion(r.Ambitos)
	r.Atributos = copiarMapaRelacion(r.Atributos)
	return r
}
func (m MaterialLectorRelacionRPT) HuellaSHA256() (string, error) {
	return m.recurso.HuellaContextoAutorizacionSHA256()
}
func (m MaterialLectorRelacionRPT) CoincideObjetivo(empleado, relacion, organismo string, version int64) bool {
	s := m.solicitud
	return empleado == s.EmpleadoRef && relacion == s.RelacionRef && organismo == s.OrganismoRef && version == s.VersionEsperada
}
func (m MaterialLectorRelacionRPT) CoincideCorte(c CorteEmpleadoB2) bool {
	return corteRegistroB2Igual(m.Corte(), c)
}
func (m MaterialLectorRelacionRPT) ActorVigenteEn(instante time.Time) bool {
	a := m.solicitud.Actor
	if instante.IsZero() || instante.Before(a.ResueltoEn) || !a.Instantanea.VigenteEn(instante) {
		return false
	}
	for _, v := range a.Instantanea.Vinculos {
		if !v.VigenteEn(instante) {
			return false
		}
	}
	return true
}

func ValidarEstadoPeriodoLectorRelacionRPT(estado string, desde, hasta FechaCivil) error {
	if !estadoRelacionB2Valido(estado) || desde.Validar() != nil || (hasta != "" && (hasta.Validar() != nil || !desde.AntesDe(hasta))) {
		return ErrLectorRelacionRPTInvalido
	}
	return nil
}
func ValidarProcedenciaLectorRelacionRPT(acto, fuente, version string) error {
	v, err := strconv.ParseInt(version, 10, 64)
	if !patronReferenciaB2.MatchString(acto) || !patronReferenciaB2.MatchString(fuente) || err != nil || v < 1 || strconv.FormatInt(v, 10) != version {
		return ErrLectorRelacionRPTInvalido
	}
	return nil
}
