package ports

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
)

const (
	dominioSeudonimoSeleccionBolsa = "vec.contratacion-temporal.seleccion"
	seudonimoSeleccionRedactado    = "[SEUDONIMO_SELECCION_REDACTADO]"
)

// SeudonimoSeleccionBolsa es una referencia personal seudonimizada mediante
// HMAC y separada por dominio. Su gramática cerrada impide introducir un DNI,
// correo, nombre o identificador directo donde el contrato exige seudonimia.
//
// El valor solo se expone mediante los codecs de transporte. Todas las
// representaciones destinadas a diagnóstico o logging están redactadas.
type SeudonimoSeleccionBolsa struct {
	valor string
}

// NuevoSeudonimoSeleccionBolsa valida la gramática HMAC y su dominio antes de
// crear el tipo nominal.
func NuevoSeudonimoSeleccionBolsa(valor string) (SeudonimoSeleccionBolsa, error) {
	seudonimo := SeudonimoSeleccionBolsa{valor: valor}
	if seudonimo.Validar() != nil {
		return SeudonimoSeleccionBolsa{}, ErrSeudonimoSeleccionBolsaInvalido
	}
	return seudonimo, nil
}

func (s SeudonimoSeleccionBolsa) Validar() error {
	referencia, _, valida := descomponerSelloHMACBolsa(
		s.valor,
		dominioSeudonimoSeleccionBolsa,
	)
	if !valida || referencia == "" {
		return ErrSeudonimoSeleccionBolsaInvalido
	}
	return nil
}

func (s SeudonimoSeleccionBolsa) valorCanonico() string {
	if s.Validar() != nil {
		return ""
	}
	return s.valor
}

func (SeudonimoSeleccionBolsa) String() string {
	return seudonimoSeleccionRedactado
}

func (SeudonimoSeleccionBolsa) GoString() string {
	return seudonimoSeleccionRedactado
}

func (SeudonimoSeleccionBolsa) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, seudonimoSeleccionRedactado)
}

func (SeudonimoSeleccionBolsa) LogValue() slog.Value {
	return slog.StringValue(seudonimoSeleccionRedactado)
}

func (s SeudonimoSeleccionBolsa) MarshalJSON() ([]byte, error) {
	if s.valor == "" {
		return []byte("null"), nil
	}
	if s.Validar() != nil {
		return nil, ErrSeudonimoSeleccionBolsaInvalido
	}
	return json.Marshal(s.valor)
}

func (s *SeudonimoSeleccionBolsa) UnmarshalJSON(contenido []byte) error {
	if s == nil {
		return ErrSeudonimoSeleccionBolsaInvalido
	}
	if bytes.Equal(contenido, []byte("null")) {
		*s = SeudonimoSeleccionBolsa{}
		return nil
	}
	var valor string
	if err := json.Unmarshal(contenido, &valor); err != nil {
		return ErrSeudonimoSeleccionBolsaInvalido
	}
	seudonimo, err := NuevoSeudonimoSeleccionBolsa(valor)
	if err != nil {
		return err
	}
	*s = seudonimo
	return nil
}

func (s SeudonimoSeleccionBolsa) MarshalText() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrSeudonimoSeleccionBolsaInvalido
	}
	return []byte(s.valor), nil
}

func (s *SeudonimoSeleccionBolsa) UnmarshalText(contenido []byte) error {
	if s == nil {
		return ErrSeudonimoSeleccionBolsaInvalido
	}
	seudonimo, err := NuevoSeudonimoSeleccionBolsa(string(contenido))
	if err != nil {
		return err
	}
	*s = seudonimo
	return nil
}
