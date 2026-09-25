package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"unicode/utf8"

	core "vec-diputacion-granada/internal/vec/domain"
)

// Ficha propia de la persona empleada («mis datos»). El empleado no llega
// nunca del cliente: es el único empleado canónico que ContextoActor, con el
// alcance {empleado}, proyecta desde Personal para la persona autenticada.
const (
	AccionFichaPropia      = "personal.registro_empleado.ficha_propia.consultar"
	AudienciaFichaPropia   = "vec_personal.registro_empleado.ficha_propia.v1"
	FinalidadFichaPropia   = "consultar_ficha_propia"
	TipoRecursoFichaPropia = "ficha_propia_empleado"
	// LimiteFilasFichaPropia acota cada apartado; la función SQL rechaza más.
	LimiteFilasFichaPropia = 200
	maximoTextoFichaPropia = 300
)

var (
	ErrFichaPropiaInvalida     = errors.New("personal: ficha propia inválida")
	ErrFichaPropiaSinEmpleado  = errors.New("personal: la persona no tiene empleado acreditado")
	ErrFichaPropiaAmbigua      = errors.New("personal: la persona tiene varios empleados")
	ErrFichaPropiaDenegada     = errors.New("personal: ficha propia denegada")
	ErrFichaPropiaNoDisponible = errors.New("personal: ficha propia no disponible")
)

type SolicitudFichaPropia struct {
	Corte CorteEmpleadoB2
	Actor core.ContextoActor
}

// MaterialFichaPropia liga la persona, el perfil, el contexto y el empleado
// canónico a la concesión V3; su forma JSON es la que reconstruye el SQL.
type MaterialFichaPropia struct {
	empleadoRef string
	corte       CorteEmpleadoB2
	actor       core.ContextoActor
	canonico    []byte
	recurso     core.RecursoAutorizable
}

func NuevoMaterialFichaPropia(s SolicitudFichaPropia) (MaterialFichaPropia, error) {
	if s.Corte.Validar() != nil || s.Actor.Validar() != nil {
		return MaterialFichaPropia{}, ErrFichaPropiaInvalida
	}
	empleados, err := s.Actor.Referencias(core.TipoReferenciaContextoActorEmpleado)
	switch {
	case err != nil:
		return MaterialFichaPropia{}, ErrFichaPropiaInvalida
	case len(empleados) == 0:
		return MaterialFichaPropia{}, ErrFichaPropiaSinEmpleado
	case len(empleados) > 1:
		return MaterialFichaPropia{}, ErrFichaPropiaAmbigua
	case !s.Actor.AlcanceProyecciones().IncluyeEmpleado():
		// Un puntero heredado del núcleo no es la proyección de Personal.
		return MaterialFichaPropia{}, ErrFichaPropiaSinEmpleado
	}
	empleado := empleados[0]
	actor, err := s.Actor.Clonar()
	if err != nil || !ReferenciaEmpleadoValida(empleado) || !ReferenciaPersonaValida(actor.PersonaRef) || actor.Principal.ID != actor.PersonaRef {
		return MaterialFichaPropia{}, ErrFichaPropiaInvalida
	}
	material := struct {
		Esquema          string `json:"esquema"`
		EmpleadoRef      string `json:"empleado_ref"`
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
	}{
		"vec.personal.ficha-propia.consulta.v1", empleado, s.Corte.VigenteEn.Texto(),
		s.Corte.ConocidoEn.UTC().Format("2006-01-02T15:04:05.000000Z"),
		actor.Principal.ID, actor.Instantanea.VinculoRef, actor.Instantanea.VinculoVersion,
		actor.Instantanea.CuentaRef, actor.Instantanea.CuentaVersion, actor.PerfilActivoRef,
		actor.Instantanea.PerfilVersion, actor.PersonaRef, actor.Instantanea.PersonaVersion,
	}
	canonico, err := json.Marshal(material)
	if err != nil {
		return MaterialFichaPropia{}, ErrFichaPropiaInvalida
	}
	suma := sha256.Sum256(canonico)
	recurso := core.RecursoAutorizable{
		Referencia: empleado, ModuloID: "personal", Tipo: TipoRecursoFichaPropia,
		Ambitos: map[string]string{"empleado_ref": empleado},
		Atributos: map[string]string{"operacion": "ficha_propia", "vigente_en": material.VigenteEn,
			"conocido_en": material.ConocidoEn, "material_sha256": hex.EncodeToString(suma[:])},
	}
	if _, err = recurso.HuellaContextoAutorizacionSHA256(); err != nil {
		return MaterialFichaPropia{}, ErrFichaPropiaInvalida
	}
	return MaterialFichaPropia{empleado, s.Corte, actor, canonico, recurso}, nil
}

func (m MaterialFichaPropia) EmpleadoRef() string    { return m.empleadoRef }
func (m MaterialFichaPropia) Corte() CorteEmpleadoB2 { return m.corte }
func (m MaterialFichaPropia) Canonico() []byte       { return append([]byte(nil), m.canonico...) }
func (m MaterialFichaPropia) Actor() core.ContextoActor {
	a, _ := m.actor.Clonar()
	return a
}
func (m MaterialFichaPropia) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = copiarMapaRelacion(r.Ambitos)
	r.Atributos = copiarMapaRelacion(r.Atributos)
	return r
}
func (m MaterialFichaPropia) HuellaSHA256() (string, error) {
	return m.recurso.HuellaContextoAutorizacionSHA256()
}

// RelacionFichaPropia describe una relación de servicio en su fecha de
// referencia con las denominaciones publicadas; Fin es el último día
// inclusive y vacío si sigue abierta. Sin referencias internas.
type RelacionFichaPropia struct {
	Inicio    FechaCivil `json:"inicio"`
	Fin       FechaCivil `json:"fin"`
	Estado    string     `json:"estado"`
	Regimen   string     `json:"regimen"`
	Modalidad string     `json:"modalidad"`
	Unidad    string     `json:"unidad"`
	Puesto    string     `json:"puesto"`
	Situacion string     `json:"situacion"`
}

// ServicioFichaPropia es un servicio reconocido tal como consta en su acto.
type ServicioFichaPropia struct {
	Inicio FechaCivil `json:"inicio"`
	Fin    FechaCivil `json:"fin"`
	Clase  string     `json:"clase"`
	Dias   int64      `json:"dias"`
	Estado string     `json:"estado"`
}

type FichaPropia struct {
	Corte      CorteEmpleadoB2       `json:"corte"`
	Relaciones []RelacionFichaPropia `json:"relaciones"`
	Servicios  []ServicioFichaPropia `json:"servicios"`
}

func (f FichaPropia) ValidarPara(m MaterialFichaPropia) error {
	if !corteRegistroB2Igual(f.Corte, m.Corte()) || f.Relaciones == nil || f.Servicios == nil ||
		len(f.Relaciones) > LimiteFilasFichaPropia || len(f.Servicios) > LimiteFilasFichaPropia {
		return ErrFichaPropiaInvalida
	}
	for _, r := range f.Relaciones {
		if r.Inicio.Validar() != nil || (r.Fin != "" && (r.Fin.Validar() != nil || r.Fin.AntesDe(r.Inicio))) ||
			!estadoRelacionB2Valido(r.Estado) || !textoFichaPropiaValido(r.Regimen) || !textoFichaPropiaValido(r.Modalidad) ||
			!textoFichaPropiaValido(r.Unidad) || !textoFichaPropiaValido(r.Puesto) || !textoFichaPropiaValido(r.Situacion) {
			return ErrFichaPropiaInvalida
		}
	}
	for _, s := range f.Servicios {
		if s.Inicio.Validar() != nil || s.Fin.Validar() != nil || s.Fin.AntesDe(s.Inicio) || s.Dias < 0 ||
			(s.Estado != "declarado" && s.Estado != "comprobado" && s.Estado != "reconocido") || !textoFichaPropiaValido(s.Clase) {
			return ErrFichaPropiaInvalida
		}
	}
	return nil
}

// Denominación visible: vacía si la fuente no la tiene; nunca controles.
func textoFichaPropiaValido(s string) bool {
	if !utf8.ValidString(s) || utf8.RuneCountInString(s) > maximoTextoFichaPropia {
		return false
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}
