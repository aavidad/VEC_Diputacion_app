package domain

import (
	"encoding/json"
	"regexp"
	"strconv"
	"time"
)

// Material de LECTURA y ESCRITURA del segundo corte de la persona empleada
// (cronos_v1 000008). Lo construye el servidor a partir del contexto
// registrado; el cliente sólo aporta fechas, horas, permiso y clave. Canonico
// produce los bytes exactos que V3 liga por huella y que SQL vuelve a resumir.

var (
	permisoRefPatron = regexp.MustCompile(`^permiso:cronos:[-A-Za-z0-9_.]{1,96}$`)
	horaCivilPatron  = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
)

func identidadEmpleadoValida(actor, perfil, empleado, zona string) bool {
	return referenciaIdentidadMarcaje(actor, "per_") && referenciaIdentidadMarcaje(perfil, "prf_") &&
		referenciaIdentidadMarcaje(empleado, "emp_") && (zona == ZonaSaldoPeninsula || zona == ZonaSaldoCanarias)
}

func fechaCivilMaterial(valor string) (time.Time, bool) {
	if len(valor) != 10 {
		return time.Time{}, false
	}
	f, err := time.Parse(time.DateOnly, valor)
	return f, err == nil && f.Format(time.DateOnly) == valor && f.Year() >= 2000 && f.Year() <= 2100
}

// MaterialConsultaMovimientosPropios: calendario, absentismos y correcciones
// de un periodo civil de, como mucho, un año.
type MaterialConsultaMovimientosPropios struct {
	ActorRef, PerfilRef, EmpleadoRef string
	Desde, Hasta, ZonaHoraria        string
}

func (m MaterialConsultaMovimientosPropios) Validar() error {
	desde, ok1 := fechaCivilMaterial(m.Desde)
	hasta, ok2 := fechaCivilMaterial(m.Hasta)
	if !identidadEmpleadoValida(m.ActorRef, m.PerfilRef, m.EmpleadoRef, m.ZonaHoraria) || !ok1 || !ok2 ||
		hasta.Before(desde) || hasta.Sub(desde) > 366*24*time.Hour {
		return ErrMarcajeInvalido
	}
	return nil
}

func (m MaterialConsultaMovimientosPropios) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrMarcajeInvalido
	}
	return json.Marshal(map[string]string{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef,
		"desde": m.Desde, "hasta": m.Hasta, "zona_horaria": m.ZonaHoraria,
	})
}

// MaterialConsultaPermisosPropios: catálogo del año y solicitudes propias.
type MaterialConsultaPermisosPropios struct {
	ActorRef, PerfilRef, EmpleadoRef string
	Anio                             int
	ZonaHoraria                      string
}

func (m MaterialConsultaPermisosPropios) Validar() error {
	if !identidadEmpleadoValida(m.ActorRef, m.PerfilRef, m.EmpleadoRef, m.ZonaHoraria) || m.Anio < 2000 || m.Anio > 2100 {
		return ErrMarcajeInvalido
	}
	return nil
}

func (m MaterialConsultaPermisosPropios) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrMarcajeInvalido
	}
	return json.Marshal(map[string]string{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef,
		"anio": strconv.Itoa(m.Anio), "zona_horaria": m.ZonaHoraria,
	})
}

// MaterialSolicitudPermisoPropio: la cantidad no viaja; la calcula la
// función durable con el catálogo vigente y el calendario publicado. Un
// permiso en horas lleva tramo de un solo día; uno en días, ninguno.
type MaterialSolicitudPermisoPropio struct {
	ActorRef, PerfilRef, EmpleadoRef string
	ClaveOperacion, PermisoRef       string
	Desde, Hasta                     string
	HoraInicio, HoraFin              string
	ZonaHoraria                      string
}

func (m MaterialSolicitudPermisoPropio) Validar() error {
	desde, ok1 := fechaCivilMaterial(m.Desde)
	hasta, ok2 := fechaCivilMaterial(m.Hasta)
	if !identidadEmpleadoValida(m.ActorRef, m.PerfilRef, m.EmpleadoRef, m.ZonaHoraria) || !claveOperacionMarcaje(m.ClaveOperacion) ||
		!permisoRefPatron.MatchString(m.PermisoRef) || !ok1 || !ok2 || hasta.Before(desde) || hasta.Year() != desde.Year() ||
		(m.HoraInicio == "") != (m.HoraFin == "") {
		return ErrSolicitudPermisoInvalida
	}
	if m.HoraInicio != "" && (!horaCivilPatron.MatchString(m.HoraInicio) || !horaCivilPatron.MatchString(m.HoraFin) ||
		m.HoraFin <= m.HoraInicio || m.Desde != m.Hasta) {
		return ErrSolicitudPermisoInvalida
	}
	return nil
}

func (m MaterialSolicitudPermisoPropio) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrSolicitudPermisoInvalida
	}
	return json.Marshal(map[string]string{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef,
		"clave_operacion": m.ClaveOperacion, "permiso_ref": m.PermisoRef, "desde": m.Desde, "hasta": m.Hasta,
		"hora_inicio": m.HoraInicio, "hora_fin": m.HoraFin, "zona_horaria": m.ZonaHoraria,
	})
}

// SolicitudPermisoPropioRef es la referencia durable de la solicitud.
func SolicitudPermisoPropioRef(clave string) string { return "permiso:cronos:solicitud:" + clave }

// MaterialSolicitudCorreccionPropia es el material exacto de una solicitud
// de corrección por olvido: el hecho declarado más la zona civil con la que
// la función durable comprueba que el día no es futuro.
func MaterialSolicitudCorreccionPropia(s SolicitudCorreccion, zona string) ([]byte, error) {
	if s.Validar() != nil || (zona != ZonaSaldoPeninsula && zona != ZonaSaldoCanarias) {
		return nil, ErrCorreccionInvalida
	}
	return json.Marshal(map[string]string{
		"actor_ref": s.ActorRef, "perfil_ref": s.PerfilRef, "empleado_ref": s.EmpleadoRef,
		"clave_operacion": s.ClaveOperacion, "marcaje_original_ref": s.MarcajeOriginalRef,
		"movimiento": string(s.Movimiento), "fecha_civil": s.FechaCivil, "hora_pretendida": s.HoraPretendida,
		"zona_horaria": zona,
	})
}
