package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
)

const (
	FinalidadUnidadTrabajoPropiaCronos   = "consultar_unidad_trabajo_propia_cronos"
	TipoRecursoUnidadTrabajoPropiaCronos = "unidad_trabajo_propia_cronos"
)

var (
	ErrUnidadTrabajoCronosInvalida     = errors.New("personal: consulta de unidad invalida")
	ErrUnidadTrabajoCronosDenegada     = errors.New("personal: consulta de unidad denegada")
	ErrUnidadTrabajoCronosNoDisponible = errors.New("personal: consulta de unidad no disponible")
)

// MaterialUnidadTrabajoCronos vincula los dos ejes temporales con el empleado
// que acredita el contexto registrado por el servidor.
type MaterialUnidadTrabajoCronos struct {
	actor       core.ContextoActor
	fecha       FechaCivil
	conocidoEn  time.Time
	empleadoRef string
	canonico    []byte
	recurso     core.RecursoAutorizable
}

func NuevoMaterialUnidadTrabajoCronos(actor core.ContextoActor, fecha FechaCivil, conocidoEn time.Time) (MaterialUnidadTrabajoCronos, error) {
	corte := CorteEmpleadoB2{VigenteEn: fecha, ConocidoEn: conocidoEn}
	base, err := NuevoMaterialFichaPropia(SolicitudFichaPropia{Actor: actor, Corte: corte})
	if err != nil {
		if errors.Is(err, ErrFichaPropiaSinEmpleado) || errors.Is(err, ErrFichaPropiaAmbigua) {
			return MaterialUnidadTrabajoCronos{}, ErrUnidadTrabajoCronosDenegada
		}
		return MaterialUnidadTrabajoCronos{}, ErrUnidadTrabajoCronosInvalida
	}
	actor = base.Actor()
	contexto, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		return MaterialUnidadTrabajoCronos{}, ErrUnidadTrabajoCronosInvalida
	}
	conocidoEn = conocidoEn.UTC().Truncate(time.Microsecond)
	contenido := struct {
		Esquema       string          `json:"esquema"`
		EmpleadoRef   string          `json:"empleado_ref"`
		Fecha         FechaCivil      `json:"fecha"`
		ConocidoEn    string          `json:"conocido_en"`
		ContextoActor json.RawMessage `json:"contexto_actor"`
	}{"vec.personal.unidad-trabajo-propia-cronos.consulta.v1", base.EmpleadoRef(), fecha, conocidoEn.Format("2006-01-02T15:04:05.000000Z"), contexto}
	canonico, err := json.Marshal(contenido)
	if err != nil {
		return MaterialUnidadTrabajoCronos{}, ErrUnidadTrabajoCronosInvalida
	}
	suma := sha256.Sum256(canonico)
	recurso := core.RecursoAutorizable{Referencia: base.EmpleadoRef(), ModuloID: "personal", Tipo: TipoRecursoUnidadTrabajoPropiaCronos,
		Ambitos:   map[string]string{"empleado_ref": base.EmpleadoRef()},
		Atributos: map[string]string{"operacion": "unidad_trabajo_propia_cronos", "fecha": fecha.Texto(), "conocido_en": contenido.ConocidoEn, "material_sha256": hex.EncodeToString(suma[:])}}
	if _, err = recurso.HuellaContextoAutorizacionSHA256(); err != nil {
		return MaterialUnidadTrabajoCronos{}, ErrUnidadTrabajoCronosInvalida
	}
	return MaterialUnidadTrabajoCronos{actor, fecha, conocidoEn, base.EmpleadoRef(), canonico, recurso}, nil
}

func (m MaterialUnidadTrabajoCronos) Actor() core.ContextoActor { a, _ := m.actor.Clonar(); return a }
func (m MaterialUnidadTrabajoCronos) Fecha() FechaCivil         { return m.fecha }
func (m MaterialUnidadTrabajoCronos) ConocidoEn() time.Time     { return m.conocidoEn }
func (m MaterialUnidadTrabajoCronos) EmpleadoRef() string       { return m.empleadoRef }
func (m MaterialUnidadTrabajoCronos) Canonico() []byte          { return append([]byte(nil), m.canonico...) }
func (m MaterialUnidadTrabajoCronos) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = copiarMapaRelacion(r.Ambitos)
	r.Atributos = copiarMapaRelacion(r.Atributos)
	return r
}
func (m MaterialUnidadTrabajoCronos) HuellaSHA256() (string, error) {
	return m.recurso.HuellaContextoAutorizacionSHA256()
}
