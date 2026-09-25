package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"time"
)

// EsquemaEventoContratoParticipacion es el contrato del evento que publica
// Contratación temporal (CT113) y Bolsa recibe en su inbox B13.
const EsquemaEventoContratoParticipacion = "vec.contratacion-temporal.contrato-bolsa.v1"

const prefijoEventoContratoParticipacion = "evento:ct:contrato-bolsa:"

// maximoEventoContratoParticipacion coincide con la guarda SQL de B13.
const maximoEventoContratoParticipacion = 16384

var ErrEventoContratoParticipacionInvalido = errors.New("bolsa: evento de contrato invalido")

var (
	patronTipoContrato       = regexp.MustCompile(`^[a-z][a-z0-9_]{1,39}$`)
	patronClaveContrato      = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,79}$`)
	patronReferenciaContrato = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:._/-]*$`)
	patronInstanteCanonico   = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$`)
	patronHuellaSHA256       = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

const longitudMaximaReferencia = 512

// EventoContratoParticipacion es un hecho de contrato de un expediente de
// CT. No lleva datos personales: Bolsa resuelve la participación con su
// propio llamamiento. Tipo es una clave abierta (hoy solo «incorporacion»);
// su rótulo pertenece al catálogo i18n de la interfaz.
type EventoContratoParticipacion struct {
	EventoRef       string
	Tipo            string
	OrigenRef       string
	OrganizacionRef string
	ExpedienteRef   string
	LlamamientoRef  string
	Inicio          *time.Time
	FinPrevisto     *time.Time
	ModalidadClave  string
	CategoriaRef    string
	CausaClave      string
	OcurridoEn      time.Time
}

type eventoContratoParticipacionJSON struct {
	Esquema         string  `json:"esquema"`
	EventoRef       string  `json:"evento_ref"`
	Tipo            string  `json:"tipo"`
	OrigenRef       string  `json:"origen_ref"`
	OrganizacionRef string  `json:"organizacion_ref"`
	ExpedienteRef   string  `json:"expediente_ref"`
	LlamamientoRef  string  `json:"llamamiento_ref"`
	Inicio          *string `json:"inicio"`
	FinPrevisto     *string `json:"fin_previsto"`
	ModalidadClave  *string `json:"modalidad_clave"`
	CategoriaRef    *string `json:"categoria_ref"`
	CausaClave      *string `json:"causa_clave"`
	OcurridoEn      string  `json:"ocurrido_en"`
}

// DecodificarEventoContratoParticipacion valida el contenido exacto recibido
// y su huella antes de que el inbox lo toque. Rechaza campos desconocidos o
// ausentes, referencias no opacas y fechas no canónicas o invertidas.
func DecodificarEventoContratoParticipacion(contenido []byte, huellaSHA256 string) (EventoContratoParticipacion, error) {
	var vacio EventoContratoParticipacion
	suma := sha256.Sum256(contenido)
	if len(contenido) == 0 || len(contenido) > maximoEventoContratoParticipacion ||
		!patronHuellaSHA256.MatchString(huellaSHA256) || hex.EncodeToString(suma[:]) != huellaSHA256 {
		return vacio, ErrEventoContratoParticipacionInvalido
	}
	if err := camposExactosEventoContrato(contenido); err != nil {
		return vacio, err
	}
	var j eventoContratoParticipacionJSON
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&j); err != nil {
		return vacio, fmt.Errorf("%w: %w", ErrEventoContratoParticipacionInvalido, err)
	}
	if dec.Decode(&struct{}{}) != io.EOF || j.Esquema != EsquemaEventoContratoParticipacion {
		return vacio, ErrEventoContratoParticipacionInvalido
	}
	e := EventoContratoParticipacion{EventoRef: j.EventoRef, Tipo: j.Tipo, OrigenRef: j.OrigenRef, OrganizacionRef: j.OrganizacionRef,
		ExpedienteRef: j.ExpedienteRef, LlamamientoRef: j.LlamamientoRef, ModalidadClave: textoOpcional(j.ModalidadClave),
		CategoriaRef: textoOpcional(j.CategoriaRef), CausaClave: textoOpcional(j.CausaClave)}
	var err error
	if e.Inicio, err = instanteOpcional(j.Inicio); err != nil {
		return vacio, err
	}
	if e.FinPrevisto, err = instanteOpcional(j.FinPrevisto); err != nil {
		return vacio, err
	}
	ocurrido, err := instanteOpcional(&j.OcurridoEn)
	if err != nil {
		return vacio, err
	}
	e.OcurridoEn = *ocurrido
	if e.Validar() != nil || (j.ModalidadClave != nil && e.ModalidadClave == "") ||
		(j.CausaClave != nil && e.CausaClave == "") || (j.CategoriaRef != nil && e.CategoriaRef == "") {
		return vacio, ErrEventoContratoParticipacionInvalido
	}
	return e, nil
}

// Validar comprueba las invariantes del hecho, también al construirlo en Go.
func (e EventoContratoParticipacion) Validar() error {
	derivada := sha256.Sum256([]byte(e.Tipo + "\x1f" + e.OrigenRef))
	switch {
	case !patronTipoContrato.MatchString(e.Tipo),
		!referenciaOpacaContrato(e.OrigenRef), !referenciaOpacaContrato(e.OrganizacionRef),
		!referenciaOpacaContrato(e.ExpedienteRef), !referenciaOpacaContrato(e.LlamamientoRef),
		e.EventoRef != prefijoEventoContratoParticipacion+hex.EncodeToString(derivada[:]),
		e.ModalidadClave != "" && !patronClaveContrato.MatchString(e.ModalidadClave),
		e.CausaClave != "" && !patronClaveContrato.MatchString(e.CausaClave),
		e.CategoriaRef != "" && !referenciaOpacaContrato(e.CategoriaRef),
		e.OcurridoEn.IsZero(),
		e.Inicio != nil && e.FinPrevisto != nil && e.FinPrevisto.Before(*e.Inicio):
		return ErrEventoContratoParticipacionInvalido
	}
	return nil
}

func referenciaOpacaContrato(v string) bool {
	return len(v) <= longitudMaximaReferencia && patronReferenciaContrato.MatchString(v)
}

func textoOpcional(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func instanteOpcional(v *string) (*time.Time, error) {
	if v == nil {
		return nil, nil
	}
	if !patronInstanteCanonico.MatchString(*v) {
		return nil, ErrEventoContratoParticipacionInvalido
	}
	t, err := time.Parse("2006-01-02T15:04:05.000000Z", *v)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEventoContratoParticipacionInvalido, err)
	}
	t = t.UTC()
	return &t, nil
}

// camposExactosEventoContrato exige las trece claves una sola vez:
// DisallowUnknownFields no detecta ausencias ni duplicados.
func camposExactosEventoContrato(contenido []byte) error {
	var crudo map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &crudo); err != nil {
		return fmt.Errorf("%w: %w", ErrEventoContratoParticipacionInvalido, err)
	}
	if len(crudo) != 13 {
		return ErrEventoContratoParticipacionInvalido
	}
	for _, clave := range []string{"esquema", "evento_ref", "tipo", "origen_ref", "organizacion_ref", "expediente_ref",
		"llamamiento_ref", "inicio", "fin_previsto", "modalidad_clave", "categoria_ref", "causa_clave", "ocurrido_en"} {
		if _, ok := crudo[clave]; !ok {
			return ErrEventoContratoParticipacionInvalido
		}
	}
	dec := json.NewDecoder(bytes.NewReader(contenido))
	inicio, err := dec.Token()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrEventoContratoParticipacionInvalido, err)
	}
	if inicio != json.Delim('{') {
		return ErrEventoContratoParticipacionInvalido
	}
	vistos := map[string]bool{}
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return fmt.Errorf("%w: %w", ErrEventoContratoParticipacionInvalido, err)
		}
		clave, ok := t.(string)
		if !ok || vistos[clave] {
			return ErrEventoContratoParticipacionInvalido
		}
		vistos[clave] = true
		var descartar json.RawMessage
		if err := dec.Decode(&descartar); err != nil {
			return fmt.Errorf("%w: %w", ErrEventoContratoParticipacionInvalido, err)
		}
	}
	return nil
}
