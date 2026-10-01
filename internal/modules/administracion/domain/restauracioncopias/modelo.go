// Package restauracioncopias gobierna propuestas; no ejecuta sustituciones.
package restauracioncopias

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"time"
)

var ErrInvalida = errors.New("copias_restauracion_propuesta_invalida")
var ErrAlterada = errors.New("copias_restauracion_propuesta_alterada")
var ErrCaducada = errors.New("copias_restauracion_caducada")
var ErrMismaPersona = errors.New("copias_restauracion_misma_persona")
var ErrConfiguracion = errors.New("copias_restauracion_configuracion_invalida")

var refRE = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9:_.-]{0,191}$`)
var shaRE = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Configuracion procede de composición. Omitir DobleControl mantiene dos personas.
// La excepción explícita solo se admite en el ejercicio sintético offline.
type Configuracion struct {
	DobleControl *bool  `json:"doble_control"`
	Entorno      string `json:"entorno"`
}

func (c Configuracion) ExigirDobleControl() (bool, error) {
	if c.Entorno != "operativo" && c.Entorno != "sintetico_offline" {
		return false, ErrConfiguracion
	}
	if c.DobleControl == nil || *c.DobleControl {
		return true, nil
	}
	if c.Entorno != "sintetico_offline" {
		return false, ErrConfiguracion
	}
	return false, nil
}

type Propuesta struct {
	FormatoVersion       int       `json:"formato_version"`
	Ref                  string    `json:"ref"`
	ConjuntoRef          string    `json:"conjunto_ref"`
	ConjuntoSHA256       string    `json:"conjunto_sha256"`
	DestinoRef           string    `json:"destino_ref"`
	PreimagenSHA256      string    `json:"preimagen_sha256"`
	MotivoRef            string    `json:"motivo_ref"`
	VentanaRef           string    `json:"ventana_ref"`
	VentanaInicio        time.Time `json:"ventana_inicio"`
	VentanaFin           time.Time `json:"ventana_fin"`
	PoliticaRef          string    `json:"politica_ref"`
	PoliticaSHA256       string    `json:"politica_sha256"`
	ProponentePersonaRef string    `json:"proponente_persona_ref"`
	Creada               time.Time `json:"creada"`
	Caduca               time.Time `json:"caduca"`
	DobleControl         bool      `json:"doble_control"`
	Entorno              string    `json:"entorno"`
}
type Sellada struct {
	Propuesta Propuesta `json:"propuesta"`
	SHA256    string    `json:"sha256"`
}
type Revision struct {
	PropuestaSHA256 string    `json:"propuesta_sha256"`
	PersonaRef      string    `json:"persona_ref"`
	Fecha           time.Time `json:"fecha"`
}

// Huella encuadra el tipo de documento; acredita integridad, nunca identidad.
// Un documento nulo o no serializable devuelve el motivo nominal ErrInvalida.
// Ese motivo nunca es una SHA256 válida ni puede aceptarse como sello.
func Huella(tipo string, documento any) string {
	b, err := json.Marshal(documento)
	if err != nil {
		return ErrInvalida.Error()
	}
	if string(b) == "null" {
		return ErrInvalida.Error()
	}
	sum := sha256.Sum256(append([]byte(tipo+"\x00"), b...))
	return hex.EncodeToString(sum[:])
}
func Sellar(p Propuesta) (Sellada, error) {
	if !Valida(p) {
		return Sellada{}, ErrInvalida
	}
	sello := Huella("vec-restauracion-propuesta-v1", p)
	if !SHA256Valida(sello) {
		return Sellada{}, ErrInvalida
	}
	return Sellada{p, sello}, nil
}

// SHA256Valida distingue un sello canónico de un motivo nominal o vacío.
func SHA256Valida(sello string) bool { return shaRE.MatchString(sello) }

func UTC(t time.Time) bool { _, off := t.Zone(); return !t.IsZero() && off == 0 }
func Valida(p Propuesta) bool {
	if p.FormatoVersion != 1 {
		return false
	}
	for _, r := range []string{p.Ref, p.ConjuntoRef, p.DestinoRef, p.MotivoRef, p.VentanaRef, p.PoliticaRef, p.ProponentePersonaRef} {
		if !refRE.MatchString(r) {
			return false
		}
	}
	for _, h := range []string{p.ConjuntoSHA256, p.PreimagenSHA256, p.PoliticaSHA256} {
		if !SHA256Valida(h) {
			return false
		}
	}
	if !UTC(p.Creada) || !UTC(p.Caduca) || !UTC(p.VentanaInicio) || !UTC(p.VentanaFin) {
		return false
	}
	if !p.Creada.Before(p.Caduca) || !p.VentanaInicio.Before(p.VentanaFin) || p.Caduca.After(p.VentanaFin) || !p.Creada.Before(p.VentanaFin) || !p.VentanaInicio.Before(p.Caduca) {
		return false
	}
	return p.Entorno == "operativo" && p.DobleControl || p.Entorno == "sintetico_offline"
}
func (s Sellada) Comprobar(ahora time.Time) error {
	if !Valida(s.Propuesta) || !SHA256Valida(s.SHA256) || s.SHA256 != Huella("vec-restauracion-propuesta-v1", s.Propuesta) {
		return ErrAlterada
	}
	if !UTC(ahora) || ahora.Before(s.Propuesta.Creada) || !ahora.Before(s.Propuesta.Caduca) {
		return ErrCaducada
	}
	return nil
}
func (s Sellada) Revisar(persona string, ahora time.Time) (Revision, error) {
	if err := s.Comprobar(ahora); err != nil {
		return Revision{}, err
	}
	if !refRE.MatchString(persona) {
		return Revision{}, ErrInvalida
	}
	if s.Propuesta.DobleControl && persona == s.Propuesta.ProponentePersonaRef {
		return Revision{}, ErrMismaPersona
	}
	return Revision{s.SHA256, persona, ahora}, nil
}
func (s Sellada) ComprobarRevision(r Revision, ahora time.Time) error {
	if err := s.Comprobar(ahora); err != nil {
		return err
	}
	if r.PropuestaSHA256 != s.SHA256 || !UTC(r.Fecha) || r.Fecha.Before(s.Propuesta.Creada) || r.Fecha.After(ahora) || !r.Fecha.Before(s.Propuesta.Caduca) {
		return ErrAlterada
	}
	_, err := s.Revisar(r.PersonaRef, ahora)
	return err
}
