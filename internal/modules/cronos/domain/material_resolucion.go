package domain

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Material de la resolución de permisos (C7) y de los avisos de resolución
// de la persona empleada (C9), cronos_v1 000009. Lo construye el servidor a
// partir del contexto registrado de quien actúa; el cliente sólo aporta
// paso, solicitud, decisión, motivo, versión y clave. Canonico produce los
// bytes exactos que V3 liga por huella y que SQL vuelve a resumir.

// MaximoMotivoResolucion es el límite, en caracteres, del motivo que quien
// resuelve escribe; la función durable aplica el mismo.
const MaximoMotivoResolucion = 500

var (
	solicitudPermisoRefPatron = regexp.MustCompile(`^permiso:cronos:solicitud:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$`)
	avisoRefPatron            = regexp.MustCompile(`^aviso:cronos:[0-9a-f-]{36}$`)
)

// PasoResolucionValido admite sólo los dos pasos del circuito: jefatura
// (responsable) y RRHH (administración).
func PasoResolucionValido(p PasoPermiso) bool { return p == PasoResponsable || p == PasoAdministracion }

// SolicitudPermisoRefValida comprueba la forma de la referencia durable.
func SolicitudPermisoRefValida(ref string) bool { return solicitudPermisoRefPatron.MatchString(ref) }

// AvisoRefValida comprueba la forma de la referencia de un aviso.
func AvisoRefValida(ref string) bool { return avisoRefPatron.MatchString(ref) }

// MotivoResolucionValido: texto de una línea, sin caracteres de control ni
// espacios en los extremos, de hasta MaximoMotivoResolucion caracteres.
// Vacío sólo si la decisión no lo exige.
func MotivoResolucionValido(motivo string) bool {
	if !utf8.ValidString(motivo) || utf8.RuneCountInString(motivo) > MaximoMotivoResolucion || motivo != strings.TrimSpace(motivo) {
		return false
	}
	for _, r := range motivo {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// ResolucionPermisoRef es la referencia durable de una resolución.
func ResolucionPermisoRef(clave string) string { return "permiso:cronos:resolucion:" + clave }

// ArchivoAvisoRef es la referencia durable del archivo de un aviso.
func ArchivoAvisoRef(clave string) string { return "aviso:cronos:archivo:" + clave }

// BandejaPermisosRef es el recurso autorizable de la bandeja de un paso.
func BandejaPermisosRef(paso PasoPermiso) string { return "bandeja:cronos:permisos:" + string(paso) }

// MaterialBandejaPermisos: lectura de las solicitudes pendientes del paso
// para quien resuelve. EmpleadoRef es el empleado propio de quien consulta,
// que la función durable excluye de la bandeja.
type MaterialBandejaPermisos struct {
	ActorRef, PerfilRef, EmpleadoRef string
	Paso                             PasoPermiso
	ZonaHoraria                      string
}

func (m MaterialBandejaPermisos) Validar() error {
	if !identidadEmpleadoValida(m.ActorRef, m.PerfilRef, m.EmpleadoRef, m.ZonaHoraria) || !PasoResolucionValido(m.Paso) {
		return ErrSolicitudPermisoInvalida
	}
	return nil
}

func (m MaterialBandejaPermisos) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrSolicitudPermisoInvalida
	}
	return json.Marshal(map[string]string{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef,
		"paso": string(m.Paso), "zona_horaria": m.ZonaHoraria,
	})
}

// MaterialResolucionPermiso: resolución de un paso de una solicitud ajena.
// EmpleadoRef es el empleado propio de quien resuelve (nunca el de la
// solicitud): la función durable impide resolver lo propio.
type MaterialResolucionPermiso struct {
	ActorRef, PerfilRef, EmpleadoRef string
	ClaveOperacion, SolicitudRef     string
	Paso                             PasoPermiso
	Decision                         DecisionPermiso
	Motivo                           string
	VersionEsperada                  int
	ZonaHoraria                      string
}

func (m MaterialResolucionPermiso) Validar() error {
	if !identidadEmpleadoValida(m.ActorRef, m.PerfilRef, m.EmpleadoRef, m.ZonaHoraria) || !claveOperacionMarcaje(m.ClaveOperacion) ||
		!SolicitudPermisoRefValida(m.SolicitudRef) || !PasoResolucionValido(m.Paso) ||
		(m.Decision != DecisionAprobar && m.Decision != DecisionDenegar) || m.VersionEsperada < 1 || m.VersionEsperada > 999 ||
		!MotivoResolucionValido(m.Motivo) || (m.Decision == DecisionDenegar && m.Motivo == "") {
		return ErrSolicitudPermisoInvalida
	}
	return nil
}

func (m MaterialResolucionPermiso) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrSolicitudPermisoInvalida
	}
	return json.Marshal(map[string]string{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef,
		"clave_operacion": m.ClaveOperacion, "solicitud_ref": m.SolicitudRef, "paso": string(m.Paso),
		"decision": string(m.Decision), "motivo": m.Motivo, "version_esperada": strconv.Itoa(m.VersionEsperada),
		"zona_horaria": m.ZonaHoraria,
	})
}

// EstadoTrasResolucion es el estado que produce una decisión en un paso.
func EstadoTrasResolucion(paso PasoPermiso, decision DecisionPermiso) (EstadoSolicitudPermiso, bool) {
	switch {
	case decision == DecisionDenegar && PasoResolucionValido(paso):
		return EstadoPermisoDenegado, true
	case decision == DecisionAprobar && paso == PasoResponsable:
		return EstadoPermisoPendienteAdministracion, true
	case decision == DecisionAprobar && paso == PasoAdministracion:
		return EstadoPermisoConcedido, true
	}
	return "", false
}

// MaterialConsultaAvisosPropios: avisos de resolución de la persona.
type MaterialConsultaAvisosPropios struct {
	ActorRef, PerfilRef, EmpleadoRef, ZonaHoraria string
}

func (m MaterialConsultaAvisosPropios) Validar() error {
	if !identidadEmpleadoValida(m.ActorRef, m.PerfilRef, m.EmpleadoRef, m.ZonaHoraria) {
		return ErrMensajeInvalido
	}
	return nil
}

func (m MaterialConsultaAvisosPropios) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrMensajeInvalido
	}
	return json.Marshal(map[string]string{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef, "zona_horaria": m.ZonaHoraria,
	})
}

// MaterialArchivoAvisoPropio: archivar un aviso propio, una sola vez.
type MaterialArchivoAvisoPropio struct {
	ActorRef, PerfilRef, EmpleadoRef string
	ClaveOperacion, AvisoRef         string
	ZonaHoraria                      string
}

func (m MaterialArchivoAvisoPropio) Validar() error {
	if !identidadEmpleadoValida(m.ActorRef, m.PerfilRef, m.EmpleadoRef, m.ZonaHoraria) || !claveOperacionMarcaje(m.ClaveOperacion) || !AvisoRefValida(m.AvisoRef) {
		return ErrMensajeInvalido
	}
	return nil
}

func (m MaterialArchivoAvisoPropio) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrMensajeInvalido
	}
	return json.Marshal(map[string]string{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef,
		"clave_operacion": m.ClaveOperacion, "aviso_ref": m.AvisoRef, "zona_horaria": m.ZonaHoraria,
	})
}
