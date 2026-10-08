package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"
)

// Canales de aviso de un llamamiento. Son complementarios: el correo se envía
// al emitir y, después, el mismo llamamiento puede seguirse por teléfono (y
// por SMS o Telegram cuando se activen). Cada aviso o intento se registra como
// un contacto neutral al canal; ningún canal resuelve por sí solo la oferta.
const (
	CanalAvisoCorreo   = "correo"
	CanalAvisoTelefono = "telefono"
	CanalAvisoSMS      = "sms"
	CanalAvisoTelegram = "telegram"

	ModoCanalAutomatico = "automatico"
	ModoCanalManual     = "manual"
)

var (
	ErrCatalogoCanalesLlamamiento = errors.New("bolsa: catalogo de canales de llamamiento invalido")
	patronVersionCatalogoCanales  = regexp.MustCompile(`^bolsa-canales-llamamiento-v[1-9][0-9]{0,3}$`)
)

// CanalLlamamiento declara lo que un canal sabe hacer. El catálogo versionado
// gobierna su lista y su activación; el adaptador dice si su infraestructura
// real existe. Solo se publica un canal activo en el catálogo y disponible.
type CanalLlamamiento struct {
	Canal string `json:"canal"`
	Modo  string `json:"modo"`
	// AlEmitir: el aviso sale al emitir el llamamiento (hoy, solo el correo).
	AlEmitir bool `json:"al_emitir"`
	// Seguimiento: RRHH puede seguir el llamamiento ya emitido por este canal,
	// persona a persona en el orden de la lista.
	Seguimiento bool `json:"seguimiento"`
	// Acuse: el canal puede confirmar la entrega o la lectura.
	Acuse bool `json:"acuse"`
	// AdmiteRespuesta: la persona puede contestar por el mismo canal. La
	// respuesta se anota; la aceptación o la renuncia siguen su propio acto.
	AdmiteRespuesta bool     `json:"admite_respuesta"`
	Resultados      []string `json:"resultados"`
	// ResultadosCierre marcan a una persona como atendida en el seguimiento.
	ResultadosCierre []string `json:"resultados_cierre,omitempty"`
	// RequiereAltaVoluntaria: solo se usa con la persona que se ha dado de
	// alta (y puede darse de baja) en el canal. Obligatorio en Telegram.
	RequiereAltaVoluntaria bool `json:"requiere_alta_voluntaria,omitempty"`
	// LimiteCaracteres acota el texto de los canales de mensaje corto.
	LimiteCaracteres int  `json:"limite_caracteres,omitempty"`
	Activo           bool `json:"activo"`
}

// CatalogoCanalesLlamamiento es la lista versionada de canales.
type CatalogoCanalesLlamamiento struct {
	Version string             `json:"version"`
	Canales []CanalLlamamiento `json:"canales"`
}

// ParsearCatalogoCanalesLlamamiento lee el catálogo de forma estricta. Un
// catálogo inválido impide componer los canales: no se adivina ninguno.
func ParsearCatalogoCanalesLlamamiento(datos []byte) (CatalogoCanalesLlamamiento, error) {
	var c CatalogoCanalesLlamamiento
	d := json.NewDecoder(bytes.NewReader(datos))
	d.DisallowUnknownFields()
	if len(datos) == 0 || len(datos) > 64<<10 || d.Decode(&c) != nil || d.More() {
		return CatalogoCanalesLlamamiento{}, ErrCatalogoCanalesLlamamiento
	}
	if err := c.Validar(); err != nil {
		return CatalogoCanalesLlamamiento{}, err
	}
	return c, nil
}

// Validar exige canales conocidos, sin repetir, con resultados del registro
// de contactos y las garantías de cada canal (correo al emitir, teléfono
// manual, Telegram con alta voluntaria, mensajes cortos con límite).
func (c CatalogoCanalesLlamamiento) Validar() error {
	if !patronVersionCatalogoCanales.MatchString(c.Version) || len(c.Canales) == 0 || len(c.Canales) > 8 {
		return ErrCatalogoCanalesLlamamiento
	}
	vistos := map[string]bool{}
	for _, canal := range c.Canales {
		if vistos[canal.Canal] || !canal.valido() {
			return ErrCatalogoCanalesLlamamiento
		}
		vistos[canal.Canal] = true
	}
	if !vistos[CanalAvisoCorreo] {
		return ErrCatalogoCanalesLlamamiento
	}
	return nil
}

func (c CanalLlamamiento) valido() bool {
	if (c.Modo != ModoCanalAutomatico && c.Modo != ModoCanalManual) || len(c.Resultados) == 0 || len(c.Resultados) > 16 || len(c.ResultadosCierre) > len(c.Resultados) || c.LimiteCaracteres < 0 || c.LimiteCaracteres > 4096 {
		return false
	}
	resultados := map[string]bool{}
	for _, r := range c.Resultados {
		if _, ok := resultadosContacto[r]; !ok || resultados[r] {
			return false
		}
		resultados[r] = true
	}
	for _, r := range c.ResultadosCierre {
		if !resultados[r] {
			return false
		}
	}
	switch c.Canal {
	case CanalAvisoCorreo:
		return c.Modo == ModoCanalAutomatico && c.AlEmitir && !c.RequiereAltaVoluntaria
	case CanalAvisoTelefono:
		return c.Modo == ModoCanalManual && !c.AlEmitir && c.Seguimiento && !c.RequiereAltaVoluntaria
	case CanalAvisoSMS:
		return c.Modo == ModoCanalAutomatico && c.LimiteCaracteres > 0
	case CanalAvisoTelegram:
		return c.Modo == ModoCanalAutomatico && c.LimiteCaracteres > 0 && c.RequiereAltaVoluntaria
	}
	return false
}

// Copia devuelve el canal sin compartir sus listas.
func (c CanalLlamamiento) Copia() CanalLlamamiento {
	c.Resultados = append([]string(nil), c.Resultados...)
	c.ResultadosCierre = append([]string(nil), c.ResultadosCierre...)
	return c
}
