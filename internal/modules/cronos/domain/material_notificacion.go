package domain

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Material de las notificaciones de la persona empleada a RRHH (C9,
// cronos_v1 000010). Lo construye el servidor con el contexto registrado de
// quien actúa; el cliente sólo aporta tipo, fecha, texto, adjunto, la
// notificación que se atiende y su clave. Canonico produce los bytes exactos
// que V3 liga por huella y que SQL vuelve a resumir. Puede contener texto
// personal: nunca se registra en logs.

var ErrNotificacionInvalida = errors.New("cronos notificacion invalida")

// MaximoTextoNotificacion es el límite, en caracteres, del texto; la función
// durable aplica el mismo.
const MaximoTextoNotificacion = 512

var (
	tipoNotificacionVersionPatron = regexp.MustCompile(`^notificacion:cronos:tipo:[a-z0-9-]{1,64}:[A-Za-z0-9_.-]{1,64}$`)
	tipoNotificacionPatron        = regexp.MustCompile(`^notificacion:cronos:tipo:[a-z0-9-]{1,64}$`)
	notificacionRefPatron         = regexp.MustCompile(`^notificacion:cronos:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	atencionRefPatron             = regexp.MustCompile(`^notificacion:cronos:atencion:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	adjuntoRefPatron              = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9:_-]{2,127}$`)
	huellaAdjuntoPatron           = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// TipoNotificacionVersionValido comprueba la referencia de una versión del
// catálogo de tipos.
func TipoNotificacionVersionValido(ref string) bool {
	return tipoNotificacionVersionPatron.MatchString(ref)
}

// TipoNotificacionValido comprueba la referencia estable de un tipo.
func TipoNotificacionValido(ref string) bool { return tipoNotificacionPatron.MatchString(ref) }

// NotificacionRefValida comprueba la referencia durable de una notificación.
func NotificacionRefValida(ref string) bool { return notificacionRefPatron.MatchString(ref) }

// AtencionRefValida comprueba la referencia durable de una atención.
func AtencionRefValida(ref string) bool { return atencionRefPatron.MatchString(ref) }

// AdjuntoNotificacionValido: sin adjunto (ambos vacíos) o la referencia de
// custodia y la huella SHA-256 no nula del documento, que no se sube.
func AdjuntoNotificacionValido(ref, huella string) bool {
	if ref == "" && huella == "" {
		return true
	}
	return adjuntoRefPatron.MatchString(ref) && huellaAdjuntoPatron.MatchString(huella) && huella != strings.Repeat("0", 64)
}

// TextoNotificacionValido: de 1 a MaximoTextoNotificacion caracteres, no en
// blanco, sin caracteres de control salvo salto de línea y tabulador.
func TextoNotificacionValido(texto string) bool {
	if !utf8.ValidString(texto) || utf8.RuneCountInString(texto) > MaximoTextoNotificacion || strings.Trim(texto, " \n\t") == "" {
		return false
	}
	for _, r := range texto {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return false
		}
	}
	return true
}

// NotificacionPropiaRecursoRef es el recurso autorizable del registro.
func NotificacionPropiaRecursoRef(clave string) string { return "notificacion:cronos:" + clave }

// AtencionNotificacionRecursoRef es el recurso autorizable de la atención.
func AtencionNotificacionRecursoRef(clave string) string {
	return "notificacion:cronos:atencion:" + clave
}

// BandejaNotificacionesRef es el recurso autorizable de la bandeja de RRHH.
const BandejaNotificacionesRef = "bandeja:cronos:notificaciones"

// MaterialRegistroNotificacion: la persona comunica algo a RRHH.
type MaterialRegistroNotificacion struct {
	ActorRef, PerfilRef, EmpleadoRef string
	ClaveOperacion, TipoVersionRef   string
	FechaReferida, Texto             string
	AdjuntoRef, AdjuntoSHA256        string
	ZonaHoraria                      string
}

func (m MaterialRegistroNotificacion) Validar() error {
	if _, ok := fechaCivilMaterial(m.FechaReferida); !ok || !identidadEmpleadoValida(m.ActorRef, m.PerfilRef, m.EmpleadoRef, m.ZonaHoraria) ||
		!claveOperacionMarcaje(m.ClaveOperacion) || !TipoNotificacionVersionValido(m.TipoVersionRef) || !TextoNotificacionValido(m.Texto) ||
		!AdjuntoNotificacionValido(m.AdjuntoRef, m.AdjuntoSHA256) {
		return ErrNotificacionInvalida
	}
	return nil
}

func (m MaterialRegistroNotificacion) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrNotificacionInvalida
	}
	return json.Marshal(map[string]string{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef,
		"clave_operacion": m.ClaveOperacion, "tipo_version_ref": m.TipoVersionRef, "fecha_referida": m.FechaReferida,
		"texto": m.Texto, "adjunto_ref": m.AdjuntoRef, "adjunto_sha256": m.AdjuntoSHA256, "zona_horaria": m.ZonaHoraria,
	})
}

// MaterialConsultaNotificaciones: lectura de las propias (persona) o de la
// bandeja (RRHH). EmpleadoRef es siempre el empleado propio de quien consulta;
// la bandeja lo excluye.
type MaterialConsultaNotificaciones struct {
	ActorRef, PerfilRef, EmpleadoRef, ZonaHoraria string
}

func (m MaterialConsultaNotificaciones) Validar() error {
	if !identidadEmpleadoValida(m.ActorRef, m.PerfilRef, m.EmpleadoRef, m.ZonaHoraria) {
		return ErrNotificacionInvalida
	}
	return nil
}

func (m MaterialConsultaNotificaciones) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrNotificacionInvalida
	}
	return json.Marshal(map[string]string{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef, "zona_horaria": m.ZonaHoraria,
	})
}

// MaterialAtencionNotificacion: RRHH marca atendida una notificación ajena.
type MaterialAtencionNotificacion struct {
	ActorRef, PerfilRef, EmpleadoRef string
	ClaveOperacion, NotificacionRef  string
	ZonaHoraria                      string
}

func (m MaterialAtencionNotificacion) Validar() error {
	if !identidadEmpleadoValida(m.ActorRef, m.PerfilRef, m.EmpleadoRef, m.ZonaHoraria) || !claveOperacionMarcaje(m.ClaveOperacion) ||
		!NotificacionRefValida(m.NotificacionRef) {
		return ErrNotificacionInvalida
	}
	return nil
}

func (m MaterialAtencionNotificacion) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrNotificacionInvalida
	}
	return json.Marshal(map[string]string{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef,
		"clave_operacion": m.ClaveOperacion, "notificacion_ref": m.NotificacionRef, "zona_horaria": m.ZonaHoraria,
	})
}
