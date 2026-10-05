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
	FinalidadLectorServiciosCertificados   = "consultar_servicios_para_certificados"
	TipoRecursoLectorServiciosCertificados = "servicios_certificados"
	OperacionLectorServiciosCertificados   = "servicios_certificados_propios"
	// Límite técnico de la fuente; un exceso es error, nunca un truncado.
	LimiteLectorServiciosCertificados = 200
)

var (
	ErrLectorServiciosCertificadosInvalido     = errors.New("personal: lectura de servicios para certificados invalida")
	ErrLectorServiciosCertificadosDenegado     = errors.New("personal: lectura de servicios para certificados denegada")
	ErrLectorServiciosCertificadosNoDisponible = errors.New("personal: lectura de servicios para certificados no disponible")
	ErrLectorServiciosCertificadosExcedeLimite = errors.New("personal: lectura de servicios para certificados excede su limite")
	organismoLectorServiciosCertificados       = regexp.MustCompile(`^[a-z][a-z0-9_:-]{2,127}$`)
	servicioLectorServiciosCertificados        = regexp.MustCompile(`^srv_[A-Za-z0-9_-]{22,128}$`)
)

// SolicitudLectorServiciosCertificados conserva el actor efectivo acreditado
// por ContextoActor y el empleado/organismo fijados por el servidor. En este
// corte sólo hay autoservicio: el empleado debe ser el único empleado canónico
// del propio actor. La consulta de RRHH sobre otra persona no está disponible.
type SolicitudLectorServiciosCertificados struct {
	Actor                     core.ContextoActor
	EmpleadoRef, OrganismoRef string
	Corte                     CorteEmpleadoB2
}

type MaterialLectorServiciosCertificados struct {
	solicitud SolicitudLectorServiciosCertificados
	canonico  []byte
	recurso   core.RecursoAutorizable
}

// El orden de estos 15 campos es el canon que reconstruye Personal36.
type canonLectorServiciosCertificados struct {
	Esquema          string `json:"esquema"`
	Operacion        string `json:"operacion"`
	EmpleadoRef      string `json:"empleado_ref"`
	OrganismoRef     string `json:"organismo_ref"`
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

// NuevoMaterialLectorServiciosCertificados devuelve Denegado si el empleado no
// es el propio del actor (ausente, ambiguo, heredado o ajeno) e Invalido si la
// entrada está mal formada. Que el empleado sea el propio no concede permiso:
// la lectura sigue exigiendo una concesión central positiva y la revalidación
// de la proyección canónica en la fuente.
func NuevoMaterialLectorServiciosCertificados(s SolicitudLectorServiciosCertificados) (MaterialLectorServiciosCertificados, error) {
	var cero MaterialLectorServiciosCertificados
	if s.Actor.Validar() != nil || !ReferenciaEmpleadoValida(s.EmpleadoRef) || !organismoLectorServiciosCertificados.MatchString(s.OrganismoRef) || s.Corte.Validar() != nil {
		return cero, ErrLectorServiciosCertificadosInvalido
	}
	actor, err := s.Actor.Clonar()
	if err != nil || !ReferenciaPersonaValida(actor.PersonaRef) || actor.Principal.ID != actor.PersonaRef {
		return cero, ErrLectorServiciosCertificadosInvalido
	}
	empleados, err := actor.Referencias(core.TipoReferenciaContextoActorEmpleado)
	if err != nil {
		return cero, ErrLectorServiciosCertificadosInvalido
	}
	// Un puntero heredado del núcleo no es la proyección de Personal.
	if len(empleados) != 1 || !actor.AlcanceProyecciones().IncluyeEmpleado() || empleados[0] != s.EmpleadoRef {
		return cero, ErrLectorServiciosCertificadosDenegado
	}
	s.Actor = actor
	conocido := s.Corte.ConocidoEn.UTC().Format("2006-01-02T15:04:05.000000Z")
	canonico, err := json.Marshal(canonLectorServiciosCertificados{
		Esquema: "vec.personal.servicios-certificados.consulta.v1", Operacion: OperacionLectorServiciosCertificados,
		EmpleadoRef: s.EmpleadoRef, OrganismoRef: s.OrganismoRef, VigenteEn: s.Corte.VigenteEn.Texto(), ConocidoEn: conocido,
		ActorRef: actor.Principal.ID, ContextoActorRef: actor.Instantanea.VinculoRef, ContextoVersion: actor.Instantanea.VinculoVersion,
		CuentaRef: actor.Instantanea.CuentaRef, CuentaVersion: actor.Instantanea.CuentaVersion,
		PerfilRef: actor.PerfilActivoRef, PerfilVersion: actor.Instantanea.PerfilVersion,
		PersonaRef: actor.PersonaRef, PersonaVersion: actor.Instantanea.PersonaVersion,
	})
	if err != nil {
		return cero, ErrLectorServiciosCertificadosInvalido
	}
	h := sha256.Sum256(canonico)
	recurso := core.RecursoAutorizable{Referencia: s.EmpleadoRef, ModuloID: "personal", Tipo: TipoRecursoLectorServiciosCertificados,
		Ambitos:   map[string]string{"empleado_ref": s.EmpleadoRef, "organismo_ref": s.OrganismoRef},
		Atributos: map[string]string{"conocido_en": conocido, "material_sha256": hex.EncodeToString(h[:]), "operacion": OperacionLectorServiciosCertificados, "vigente_en": s.Corte.VigenteEn.Texto()},
	}
	if _, err := recurso.HuellaContextoAutorizacionSHA256(); err != nil {
		return cero, ErrLectorServiciosCertificadosInvalido
	}
	return MaterialLectorServiciosCertificados{solicitud: s, canonico: canonico, recurso: recurso}, nil
}

func (m MaterialLectorServiciosCertificados) Actor() core.ContextoActor {
	// Copia defensiva sin error posible: el actor ya se clonó al crear el material.
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
func (m MaterialLectorServiciosCertificados) Solicitud() SolicitudLectorServiciosCertificados {
	s := m.solicitud
	s.Actor = m.Actor()
	return s
}
func (m MaterialLectorServiciosCertificados) Corte() CorteEmpleadoB2 { return m.solicitud.Corte }
func (m MaterialLectorServiciosCertificados) EmpleadoRef() string    { return m.solicitud.EmpleadoRef }
func (m MaterialLectorServiciosCertificados) OrganismoRef() string   { return m.solicitud.OrganismoRef }
func (m MaterialLectorServiciosCertificados) Canonico() []byte {
	return append([]byte(nil), m.canonico...)
}
func (m MaterialLectorServiciosCertificados) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = copiarMapaRelacion(r.Ambitos)
	r.Atributos = copiarMapaRelacion(r.Atributos)
	return r
}
func (m MaterialLectorServiciosCertificados) HuellaSHA256() (string, error) {
	return m.recurso.HuellaContextoAutorizacionSHA256()
}
func (m MaterialLectorServiciosCertificados) CoincideCorte(c CorteEmpleadoB2) bool {
	return corteRegistroB2Igual(m.Corte(), c)
}

// ActorVigenteEn revalida con el reloj actual la vigencia del contexto y de
// sus vínculos; el corte solicitado no la sustituye.
func (m MaterialLectorServiciosCertificados) ActorVigenteEn(instante time.Time) bool {
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

// ServicioLeidoCertificados es la forma de un servicio devuelto por la fuente.
// Desde/Hasta forman el periodo [Desde,Hasta) tal como consta en Personal17.
type ServicioLeidoCertificados struct {
	ServicioRef, RelacionRef, Estado, ClaseRef string
	Version, DiasReconocidos, ClaseVersion     int64
	Desde, Hasta                               FechaCivil
	ActoRef, FuenteRef, FuenteVersion          string
}

// ValidarServicioLeidoCertificados comprueba forma y coherencia con el corte:
// el servicio ya había comenzado en la fecha del corte. No decide cómputo,
// antigüedad ni si el servicio sustenta un certificado.
func ValidarServicioLeidoCertificados(s ServicioLeidoCertificados, corte CorteEmpleadoB2) error {
	v, err := strconv.ParseInt(s.FuenteVersion, 10, 64)
	if !servicioLectorServiciosCertificados.MatchString(s.ServicioRef) || !ReferenciaRelacionValida(s.RelacionRef) ||
		(s.Estado != "declarado" && s.Estado != "comprobado" && s.Estado != "reconocido") || !patronReferenciaB2.MatchString(s.ClaseRef) ||
		s.Version < 1 || s.Version > 2147483647 || s.DiasReconocidos < 0 || s.DiasReconocidos > 2147483647 || s.ClaseVersion < 0 ||
		s.Desde.Validar() != nil || s.Hasta.Validar() != nil || !s.Desde.AntesDe(s.Hasta) || corte.VigenteEn.AntesDe(s.Desde) ||
		!patronReferenciaB2.MatchString(s.ActoRef) || !patronReferenciaB2.MatchString(s.FuenteRef) ||
		err != nil || v < 1 || strconv.FormatInt(v, 10) != s.FuenteVersion {
		return ErrLectorServiciosCertificadosInvalido
	}
	return nil
}
