package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	core "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionAltaEmpleadoB2     = "personal.registro_empleado.alta.registrar"
	AccionHechoEmpleadoB2    = "personal.registro_empleado.hecho.registrar"
	AudienciaAltaEmpleadoB2  = "vec_personal.registro_empleado.alta.v1"
	AudienciaHechoEmpleadoB2 = "vec_personal.registro_empleado.hecho.v1"
)

type ProcedenciaActoEmpleadoB2 struct {
	ActoRef            string `json:"acto_ref"`
	FuenteRef          string `json:"fuente_ref"`
	FuenteVersion      int64  `json:"fuente_version"`
	FuenteHuellaSHA256 string `json:"fuente_huella_sha256"`
	IdempotenciaRef    string `json:"idempotencia_ref"`
}

func (p ProcedenciaActoEmpleadoB2) Validar() error {
	if !patronReferenciaB2.MatchString(p.ActoRef) || !patronReferenciaB2.MatchString(p.FuenteRef) || p.FuenteVersion < 1 ||
		!huellaRegistroDominioB2Valida(p.FuenteHuellaSHA256) || !patronUUIDRegistroB2.MatchString(p.IdempotenciaRef) {
		return ErrRegistroEmpleadoB2Invalido
	}
	return nil
}

type SolicitudAltaEmpleadoB2 struct {
	PersonaRef   string
	OrganismoRef string
	UnidadRef    string
	RegimenRef   string
	ModalidadRef string
	VigenteDesde FechaCivil
	VigenteHasta FechaCivil
	Procedencia  ProcedenciaActoEmpleadoB2
	Actor        core.ContextoActor
}

type SolicitudHechoEmpleadoB2 struct {
	Tipo                    string
	EmpleadoRef             string
	OrganismoRef            string
	RelacionRef             string
	RevisionEsperada        int64
	RelacionVersionEsperada int64
	UnidadRef               string
	RegimenRef              string
	ModalidadRef            string
	Estado                  string
	PlazaRef                string
	PuestoRef               string
	ClaseRef                string
	VersionPlazaRef         string
	VersionPuestoRef        string
	PeriodoDesde            FechaCivil
	PeriodoHasta            FechaCivil
	DiasReconocidos         int64
	VigenteDesde            FechaCivil
	VigenteHasta            FechaCivil
	Procedencia             ProcedenciaActoEmpleadoB2
	Actor                   core.ContextoActor
}

type MaterialActoRegistroEmpleadoB2 struct {
	tipo       string
	referencia string
	actor      core.ContextoActor
	canonico   []byte
	recurso    core.RecursoAutorizable
}

func NuevoMaterialAltaEmpleadoB2(s SolicitudAltaEmpleadoB2) (MaterialActoRegistroEmpleadoB2, error) {
	if !ReferenciaPersonaValida(s.PersonaRef) || !patronReferenciaB2.MatchString(s.OrganismoRef) ||
		!patronReferenciaB2.MatchString(s.UnidadRef) || !patronReferenciaB2.MatchString(s.RegimenRef) ||
		!patronReferenciaB2.MatchString(s.ModalidadRef) || !intervaloActoB2Valido(s.VigenteDesde, s.VigenteHasta) ||
		s.Procedencia.Validar() != nil || s.Actor.Validar() != nil {
		return MaterialActoRegistroEmpleadoB2{}, ErrRegistroEmpleadoB2Invalido
	}
	return nuevoMaterialActoB2("alta", s.PersonaRef, s.OrganismoRef, s.Actor, struct {
		Esquema         string                    `json:"esquema"`
		Operacion       string                    `json:"operacion"`
		PersonaRef      string                    `json:"persona_ref"`
		OrganismoRef    string                    `json:"organismo_ref"`
		UnidadRef       string                    `json:"unidad_ref"`
		RegimenRef      string                    `json:"regimen_ref"`
		ModalidadRef    string                    `json:"modalidad_ref"`
		VigenteDesde    string                    `json:"vigente_desde"`
		VigenteHasta    string                    `json:"vigente_hasta"`
		VersionEsperada int64                     `json:"version_esperada"`
		Procedencia     ProcedenciaActoEmpleadoB2 `json:"procedencia"`
		Actor           identidadActoB2           `json:"actor"`
	}{"vec.personal.registro-empleado-b2.alta.v1", "alta", s.PersonaRef, s.OrganismoRef, s.UnidadRef, s.RegimenRef, s.ModalidadRef, s.VigenteDesde.Texto(), s.VigenteHasta.Texto(), 0, s.Procedencia, identidadActoRegistroB2(s.Actor)})
}

func NuevoMaterialHechoEmpleadoB2(s SolicitudHechoEmpleadoB2) (MaterialActoRegistroEmpleadoB2, error) {
	if !ReferenciaEmpleadoValida(s.EmpleadoRef) || !patronReferenciaB2.MatchString(s.OrganismoRef) || s.RevisionEsperada < 1 || !intervaloActoB2Valido(s.VigenteDesde, s.VigenteHasta) ||
		s.Procedencia.Validar() != nil || s.Actor.Validar() != nil || !hechoRegistroB2Valido(s) {
		return MaterialActoRegistroEmpleadoB2{}, ErrRegistroEmpleadoB2Invalido
	}
	return nuevoMaterialActoB2("hecho", s.EmpleadoRef, s.OrganismoRef, s.Actor, struct {
		Esquema                 string                    `json:"esquema"`
		Operacion               string                    `json:"operacion"`
		Tipo                    string                    `json:"tipo"`
		EmpleadoRef             string                    `json:"empleado_ref"`
		OrganismoRef            string                    `json:"organismo_ref"`
		RelacionRef             string                    `json:"relacion_ref"`
		RevisionEsperada        int64                     `json:"revision_esperada"`
		RelacionVersionEsperada int64                     `json:"relacion_version_esperada"`
		UnidadRef               string                    `json:"unidad_ref"`
		RegimenRef              string                    `json:"regimen_ref"`
		ModalidadRef            string                    `json:"modalidad_ref"`
		Estado                  string                    `json:"estado"`
		PlazaRef                string                    `json:"plaza_ref"`
		PuestoRef               string                    `json:"puesto_ref"`
		ClaseRef                string                    `json:"clase_ref"`
		VersionPlazaRef         string                    `json:"version_plaza_ref"`
		VersionPuestoRef        string                    `json:"version_puesto_ref"`
		PeriodoDesde            string                    `json:"periodo_desde"`
		PeriodoHasta            string                    `json:"periodo_hasta"`
		DiasReconocidos         int64                     `json:"dias_reconocidos"`
		VigenteDesde            string                    `json:"vigente_desde"`
		VigenteHasta            string                    `json:"vigente_hasta"`
		Procedencia             ProcedenciaActoEmpleadoB2 `json:"procedencia"`
		Actor                   identidadActoB2           `json:"actor"`
	}{"vec.personal.registro-empleado-b2.hecho.v1", "hecho", s.Tipo, s.EmpleadoRef, s.OrganismoRef, s.RelacionRef, s.RevisionEsperada, s.RelacionVersionEsperada, s.UnidadRef, s.RegimenRef, s.ModalidadRef, s.Estado, s.PlazaRef, s.PuestoRef, s.ClaseRef, s.VersionPlazaRef, s.VersionPuestoRef, s.PeriodoDesde.Texto(), s.PeriodoHasta.Texto(), s.DiasReconocidos, s.VigenteDesde.Texto(), s.VigenteHasta.Texto(), s.Procedencia, identidadActoRegistroB2(s.Actor)})
}

type identidadActoB2 struct {
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

func identidadActoRegistroB2(a core.ContextoActor) identidadActoB2 {
	return identidadActoB2{a.Principal.ID, a.Instantanea.VinculoRef, a.Instantanea.VinculoVersion, a.Instantanea.CuentaRef, a.Instantanea.CuentaVersion, a.PerfilActivoRef, a.Instantanea.PerfilVersion, a.PersonaRef, a.Instantanea.PersonaVersion}
}
func nuevoMaterialActoB2(tipo, ref, organismo string, actor core.ContextoActor, obj any) (MaterialActoRegistroEmpleadoB2, error) {
	actor, err := actor.Clonar()
	if err != nil {
		return MaterialActoRegistroEmpleadoB2{}, ErrRegistroEmpleadoB2Invalido
	}
	canonico, err := json.Marshal(obj)
	if err != nil {
		return MaterialActoRegistroEmpleadoB2{}, ErrRegistroEmpleadoB2Invalido
	}
	suma := sha256.Sum256(canonico)
	accionTipo := "alta_empleado_rrhh"
	if tipo == "hecho" {
		accionTipo = "hecho_empleado_rrhh"
	}
	recurso := core.RecursoAutorizable{Referencia: ref, ModuloID: "personal", Tipo: accionTipo, Ambitos: map[string]string{"objetivo_ref": ref, "organismo_ref": organismo}, Atributos: map[string]string{"operacion": tipo, "material_sha256": hex.EncodeToString(suma[:])}}
	if _, err = recurso.HuellaContextoAutorizacionSHA256(); err != nil {
		return MaterialActoRegistroEmpleadoB2{}, ErrRegistroEmpleadoB2Invalido
	}
	return MaterialActoRegistroEmpleadoB2{tipo, ref, actor, canonico, recurso}, nil
}
func (m MaterialActoRegistroEmpleadoB2) Tipo() string       { return m.tipo }
func (m MaterialActoRegistroEmpleadoB2) Referencia() string { return m.referencia }
func (m MaterialActoRegistroEmpleadoB2) Canonico() []byte   { return append([]byte(nil), m.canonico...) }
func (m MaterialActoRegistroEmpleadoB2) Actor() core.ContextoActor {
	a, _ := m.actor.Clonar()
	return a
}
func (m MaterialActoRegistroEmpleadoB2) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = copiarMapaRelacion(r.Ambitos)
	r.Atributos = copiarMapaRelacion(r.Atributos)
	return r
}
func (m MaterialActoRegistroEmpleadoB2) HuellaSHA256() (string, error) {
	return m.recurso.HuellaContextoAutorizacionSHA256()
}

func intervaloActoB2Valido(desde, hasta FechaCivil) bool {
	return desde.Validar() == nil && (hasta == "" || (hasta.Validar() == nil && desde.AntesDe(hasta)))
}
func huellaRegistroDominioB2Valida(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
func hechoRegistroB2Valido(s SolicitudHechoEmpleadoB2) bool {
	if s.Tipo != "relacion" && s.Tipo != "ocupacion" && s.Tipo != "servicio" && s.Tipo != "situacion" {
		return false
	}
	if s.Tipo == "relacion" {
		return ((s.RelacionRef == "" && s.RelacionVersionEsperada == 0) || (ReferenciaRelacionValida(s.RelacionRef) && s.RelacionVersionEsperada >= 1)) && patronReferenciaB2.MatchString(s.UnidadRef) && patronReferenciaB2.MatchString(s.RegimenRef) && patronReferenciaB2.MatchString(s.ModalidadRef) && estadoRelacionB2Valido(s.Estado)
	}
	if !ReferenciaRelacionValida(s.RelacionRef) || s.RelacionVersionEsperada < 1 {
		return false
	}
	switch s.Tipo {
	case "ocupacion":
		return referenciaOrganizacionB2Valida(s.PlazaRef) && (s.PuestoRef == "" || referenciaOrganizacionB2Valida(s.PuestoRef)) && patronReferenciaB2.MatchString(s.UnidadRef) && patronReferenciaB2.MatchString(s.ModalidadRef) && (s.ClaseRef == "titular" || s.ClaseRef == "provisional" || s.ClaseRef == "temporal" || s.ClaseRef == "reserva") && patronReferenciaB2.MatchString(s.VersionPlazaRef)
	case "servicio":
		return patronReferenciaB2.MatchString(s.ClaseRef) && s.PeriodoDesde.Validar() == nil && s.PeriodoHasta.Validar() == nil && s.PeriodoDesde.AntesDe(s.PeriodoHasta) && s.DiasReconocidos >= 0 && (s.Estado == "declarado" || s.Estado == "comprobado" || s.Estado == "reconocido")
	case "situacion":
		return patronReferenciaB2.MatchString(s.ClaseRef)
	}
	return false
}
