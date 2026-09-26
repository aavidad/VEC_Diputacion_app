package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"time"
)

// EsquemaEventoNoIncorporacion es el contrato del evento que publica
// Contratación temporal (CT124) cuando la persona aceptada no se incorpora y
// que Bolsa recibe en su bandeja (000042).
const EsquemaEventoNoIncorporacion = "vec.contratacion-temporal.no-incorporacion-bolsa.v1"

const prefijoEventoNoIncorporacion = "evento:ct:no-incorporacion-bolsa:"

var (
	patronMotivoNoIncorporacion       = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
	patronConsecuenciaNoIncorporacion = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
	patronResolucionNoIncorporacion   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/#-]*$`)
	patronFechaCivil                  = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
)

// EventoNoIncorporacion es el hecho publicado por CT: referencias opacas, el
// motivo y la consecuencia del catálogo, la resolución por referencia y
// huella, quién la registró y quién la resolvió. Nunca datos de la persona.
type EventoNoIncorporacion struct {
	EventoRef         string `json:"evento_ref"`
	Tipo              string `json:"tipo"`
	OrigenRef         string `json:"origen_ref"`
	OrganizacionRef   string `json:"organizacion_ref"`
	ExpedienteRef     string `json:"expediente_ref"`
	LlamamientoRef    string `json:"llamamiento_ref"`
	MotivoClave       string `json:"motivo_clave"`
	ConsecuenciaClave string `json:"consecuencia_clave"`
	ResolucionRef     string `json:"resolucion_ref"`
	ResolucionSHA256  string `json:"resolucion_sha256"`
	ResueltaPor       string `json:"resuelta_por"`
	ActorRef          string `json:"actor_ref"`
	FechaNotificacion string `json:"fecha_notificacion"`
	OcurridoEn        string `json:"ocurrido_en"`
}

// DecodificarEventoNoIncorporacion valida el contenido exacto y su huella
// antes de que la bandeja lo toque: quince claves, una vez cada una.
func DecodificarEventoNoIncorporacion(contenido []byte, huellaSHA256 string) (EventoNoIncorporacion, error) {
	var vacio EventoNoIncorporacion
	suma := sha256.Sum256(contenido)
	if len(contenido) == 0 || len(contenido) > maximoEventoContratoParticipacion ||
		!patronHuellaSHA256.MatchString(huellaSHA256) || hex.EncodeToString(suma[:]) != huellaSHA256 {
		return vacio, ErrEventoContratoParticipacionInvalido
	}
	var crudo map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &crudo); err != nil || len(crudo) != 15 {
		return vacio, ErrEventoContratoParticipacionInvalido
	}
	var j struct {
		Esquema string `json:"esquema"`
		EventoNoIncorporacion
	}
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&j); err != nil {
		return vacio, fmt.Errorf("%w: %w", ErrEventoContratoParticipacionInvalido, err)
	}
	if dec.Decode(&struct{}{}) != io.EOF || j.Esquema != EsquemaEventoNoIncorporacion {
		return vacio, ErrEventoContratoParticipacionInvalido
	}
	if _, ok := crudo["esquema"]; !ok {
		return vacio, ErrEventoContratoParticipacionInvalido
	}
	if err := j.EventoNoIncorporacion.Validar(); err != nil {
		return vacio, err
	}
	return j.EventoNoIncorporacion, nil
}

// Validar comprueba las invariantes del hecho, también al construirlo en Go.
func (e EventoNoIncorporacion) Validar() error {
	derivada := sha256.Sum256([]byte("no_incorporacion\x1f" + e.OrigenRef))
	if e.Tipo != "no_incorporacion" || e.EventoRef != prefijoEventoNoIncorporacion+hex.EncodeToString(derivada[:]) ||
		!referenciaOpacaContrato(e.OrigenRef) || !referenciaOpacaContrato(e.OrganizacionRef) ||
		!referenciaOpacaContrato(e.ExpedienteRef) || !referenciaOpacaContrato(e.LlamamientoRef) ||
		!referenciaOpacaContrato(e.ActorRef) || !referenciaOpacaContrato(e.ResueltaPor) || len(e.ResueltaPor) > 256 ||
		!patronResolucionNoIncorporacion.MatchString(e.ResolucionRef) || len(e.ResolucionRef) > 256 ||
		!patronHuellaSHA256.MatchString(e.ResolucionSHA256) ||
		!patronMotivoNoIncorporacion.MatchString(e.MotivoClave) || !patronConsecuenciaNoIncorporacion.MatchString(e.ConsecuenciaClave) ||
		!patronInstanteCanonico.MatchString(e.OcurridoEn) || !patronFechaCivil.MatchString(e.FechaNotificacion) {
		return ErrEventoContratoParticipacionInvalido
	}
	if _, err := e.Notificacion(); err != nil {
		return err
	}
	return nil
}

// Notificacion es la fecha civil de notificación de la resolución.
func (e EventoNoIncorporacion) Notificacion() (time.Time, error) {
	f, err := time.Parse(time.DateOnly, e.FechaNotificacion)
	if err != nil || f.Format(time.DateOnly) != e.FechaNotificacion {
		return time.Time{}, ErrEventoContratoParticipacionInvalido
	}
	return f.UTC(), nil
}
