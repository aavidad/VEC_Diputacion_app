package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrRelacionEmpleadoInvalida = errors.New("personal: relacion de empleado invalida")
	ErrFechaCivilInvalida       = errors.New("personal: fecha civil invalida")
)

// Los prefijos y límites de persona replican exactamente el contrato canónico
// de ContextoActor. Personal no intenta reinterpretar una cuenta ni ampliar
// ese espacio de referencias; empleado y relación son referencias propias.
var (
	patronPersonaEmpleado  = regexp.MustCompile(`^per_[A-Za-z0-9_-]{22,128}$`)
	patronEmpleado         = regexp.MustCompile(`^emp_[A-Za-z0-9_-]{22,128}$`)
	patronRelacionEmpleado = regexp.MustCompile(`^rel_[A-Za-z0-9_-]{22,128}$`)
	patronReferenciaFuente = regexp.MustCompile(`^[a-z][a-z0-9_:-]{2,159}$`)
)

// FechaCivil representa una fecha administrativa sin zona horaria. Las
// relaciones usan el intervalo semiabierto [Desde, Hasta): Hasta vacía es
// abierta y no se convierte implícitamente a UTC.
type FechaCivil string

func NuevaFechaCivil(valor string) (FechaCivil, error) {
	fecha, err := time.Parse("2006-01-02", valor)
	if err != nil || fecha.Format("2006-01-02") != valor {
		return "", ErrFechaCivilInvalida
	}
	return FechaCivil(valor), nil
}

// OperacionConsultaRelacionPropia y SolicitudConsultaRelacionPropia son la
// intención de negocio de una consulta propia. No son DTOs de PostgreSQL ni de
// autorización: todos los canales construyen el mismo material inmutable.
type OperacionConsultaRelacionPropia string

const (
	OperacionListaRelacionPropia   OperacionConsultaRelacionPropia = "lista"
	OperacionDetalleRelacionPropia OperacionConsultaRelacionPropia = "detalle"
)

type SolicitudConsultaRelacionPropia struct {
	FechaReferencia FechaCivil
	Operacion       OperacionConsultaRelacionPropia
	RelacionRef     string
	Actor           core.ContextoActor
}

// MaterialConsultaRelacionPropia conserva la representación exacta que se
// autoriza. Sus accesores clonan los valores mutables para que proveedor,
// aplicación y adaptador no puedan alterar la decisión ya construida.
type MaterialConsultaRelacionPropia struct {
	solicitud SolicitudConsultaRelacionPropia
	canonico  []byte
	recurso   core.RecursoAutorizable
}

func (m MaterialConsultaRelacionPropia) Solicitud() SolicitudConsultaRelacionPropia {
	c := m.solicitud
	actor, err := c.Actor.Clonar()
	if err != nil {
		return SolicitudConsultaRelacionPropia{}
	}
	c.Actor = actor
	return c
}

func (m MaterialConsultaRelacionPropia) Canonico() []byte { return append([]byte(nil), m.canonico...) }

func (m MaterialConsultaRelacionPropia) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = copiarMapaRelacion(r.Ambitos)
	r.Atributos = copiarMapaRelacion(r.Atributos)
	return r
}

func (m MaterialConsultaRelacionPropia) HuellaSHA256() (string, error) {
	return m.recurso.HuellaContextoAutorizacionSHA256()
}

type materialConsultaRelacionPropiaCanonico struct {
	Esquema     string                                  `json:"esquema"`
	Fecha       string                                  `json:"fecha_referencia"`
	Identidad   identidadConsultaRelacionPropiaCanonica `json:"identidad"`
	Operacion   string                                  `json:"operacion"`
	RelacionRef *string                                 `json:"relacion_ref"`
}

type identidadConsultaRelacionPropiaCanonica struct {
	ActorRef         string `json:"actor_ref"`
	ContextoActorRef string `json:"contexto_actor_ref"`
	ContextoVersion  uint64 `json:"contexto_version"`
	CuentaRef        string `json:"cuenta_ref"`
	CuentaVersion    uint64 `json:"cuenta_version"`
	EmpleadoRef      string `json:"empleado_ref"`
	PerfilRef        string `json:"perfil_ref"`
	PerfilVersion    uint64 `json:"perfil_version"`
	PersonaRef       string `json:"persona_ref"`
	PersonaVersion   uint64 `json:"persona_version"`
}

func NuevoMaterialConsultaRelacionPropia(c SolicitudConsultaRelacionPropia) (MaterialConsultaRelacionPropia, error) {
	if c.FechaReferencia.Validar() != nil || c.Actor.Validar() != nil {
		return MaterialConsultaRelacionPropia{}, ErrRelacionEmpleadoInvalida
	}
	empleados, err := c.Actor.Referencias("empleado")
	if err != nil || len(empleados) != 1 {
		return MaterialConsultaRelacionPropia{}, ErrRelacionEmpleadoInvalida
	}
	m := materialConsultaRelacionPropiaCanonico{
		Esquema: "vec.personal.relacion-propia-dietas.v1",
		Fecha:   c.FechaReferencia.Texto(),
		Identidad: identidadConsultaRelacionPropiaCanonica{
			ActorRef: c.Actor.Principal.ID, ContextoActorRef: c.Actor.Instantanea.VinculoRef,
			ContextoVersion: c.Actor.Instantanea.VinculoVersion, CuentaRef: c.Actor.Instantanea.CuentaRef,
			CuentaVersion: c.Actor.Instantanea.CuentaVersion, EmpleadoRef: empleados[0],
			PerfilRef: c.Actor.PerfilActivoRef, PerfilVersion: c.Actor.Instantanea.PerfilVersion,
			PersonaRef: c.Actor.PersonaRef, PersonaVersion: c.Actor.Instantanea.PersonaVersion,
		},
	}
	switch c.Operacion {
	case OperacionListaRelacionPropia:
		if c.RelacionRef != "" {
			return MaterialConsultaRelacionPropia{}, ErrRelacionEmpleadoInvalida
		}
		m.Operacion = string(OperacionListaRelacionPropia)
	case OperacionDetalleRelacionPropia:
		if !ReferenciaRelacionValida(c.RelacionRef) {
			return MaterialConsultaRelacionPropia{}, ErrRelacionEmpleadoInvalida
		}
		m.Operacion = string(OperacionDetalleRelacionPropia)
		relacion := c.RelacionRef
		m.RelacionRef = &relacion
	default:
		return MaterialConsultaRelacionPropia{}, ErrRelacionEmpleadoInvalida
	}
	canonico, err := json.Marshal(m)
	if err != nil {
		return MaterialConsultaRelacionPropia{}, ErrRelacionEmpleadoInvalida
	}
	suma := sha256.Sum256(canonico)
	relacion := c.RelacionRef
	if relacion == "" {
		relacion = "sin_seleccion"
	}
	recurso := core.RecursoAutorizable{
		Referencia: empleados[0], ModuloID: "personal", Tipo: "relacion_empleado_dietas",
		Ambitos:   map[string]string{"persona_ref": c.Actor.PersonaRef, "empleado_ref": empleados[0]},
		Atributos: map[string]string{"fecha_referencia": c.FechaReferencia.Texto(), "material_sha256": hex.EncodeToString(suma[:]), "operacion": string(c.Operacion), "relacion_ref": relacion},
	}
	if _, err = recurso.HuellaContextoAutorizacionSHA256(); err != nil {
		return MaterialConsultaRelacionPropia{}, ErrRelacionEmpleadoInvalida
	}
	actor, err := c.Actor.Clonar()
	if err != nil {
		return MaterialConsultaRelacionPropia{}, ErrRelacionEmpleadoInvalida
	}
	c.Actor = actor
	return MaterialConsultaRelacionPropia{solicitud: c, canonico: append([]byte(nil), canonico...), recurso: recurso}, nil
}

func copiarMapaRelacion(origen map[string]string) map[string]string {
	destino := make(map[string]string, len(origen))
	for clave, valor := range origen {
		destino[clave] = valor
	}
	return destino
}

func (f FechaCivil) Validar() error {
	_, err := NuevaFechaCivil(string(f))
	return err
}

func (f FechaCivil) Texto() string { return string(f) }

func (f FechaCivil) AntesDe(otra FechaCivil) bool { return string(f) < string(otra) }

// Las referencias sólo validan forma. No prueban existencia, vigencia ni
// autorización, que son responsabilidad de la consulta nominal de Personal.
func ReferenciaPersonaValida(valor string) bool  { return patronPersonaEmpleado.MatchString(valor) }
func ReferenciaEmpleadoValida(valor string) bool { return patronEmpleado.MatchString(valor) }
func ReferenciaRelacionValida(valor string) bool { return patronRelacionEmpleado.MatchString(valor) }

// ConsultaRelacionVigente expresa la selección nominal que un consumidor ya
// autorizado pide a Personal. La validación reside en dominio, no en el
// puerto: éste sólo declara la dependencia tecnológica mínima.
type ConsultaRelacionVigente struct {
	PersonaRef, EmpleadoRef, RelacionRef string
	En                                   FechaCivil
}

func (c ConsultaRelacionVigente) Validar() error {
	if c.En.Validar() != nil || !ReferenciaPersonaValida(c.PersonaRef) ||
		!ReferenciaEmpleadoValida(c.EmpleadoRef) || !ReferenciaRelacionValida(c.RelacionRef) {
		return ErrRelacionEmpleadoInvalida
	}
	return nil
}

// RelacionEmpleado es el sello mínimo completo que Personal entrega a un
// consumidor autorizado. No contiene identidad de cuenta, DNI ni contacto.
// La resolución previa no acredita por sí misma un efecto posterior: quien
// vaya a escribir deberá revalidar este sello en su propia transacción.
type RelacionEmpleado struct {
	PersonaRef, EmpleadoRef, RelacionRef string
	UnidadRef, Estado                    string
	Desde, Hasta                         FechaCivil
	Version                              int64
	ProcedenciaActoRef, FuenteRef        string
	FuenteVersion                        int64
}

func (r RelacionEmpleado) VigenteEn(fecha FechaCivil) bool {
	return r.Estado == "activa" && !fecha.AntesDe(r.Desde) && (r.Hasta == "" || fecha.AntesDe(r.Hasta))
}

func (r RelacionEmpleado) Validar() error {
	if !ReferenciaPersonaValida(r.PersonaRef) || !ReferenciaEmpleadoValida(r.EmpleadoRef) ||
		!ReferenciaRelacionValida(r.RelacionRef) || !patronReferenciaFuente.MatchString(r.UnidadRef) ||
		r.Estado != "activa" || r.Desde.Validar() != nil || (r.Hasta != "" && r.Hasta.Validar() != nil) ||
		(r.Hasta != "" && !r.Desde.AntesDe(r.Hasta)) || r.Version < 1 ||
		!patronReferenciaFuente.MatchString(r.ProcedenciaActoRef) || !patronReferenciaFuente.MatchString(r.FuenteRef) || r.FuenteVersion < 1 {
		return ErrRelacionEmpleadoInvalida
	}
	return nil
}
