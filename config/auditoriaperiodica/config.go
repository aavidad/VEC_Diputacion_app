// Package auditoriaperiodica prepara los límites del ejecutor sin publicar
// política, conceder permisos o configurar una conexión a base de datos.
package auditoriaperiodica

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrConfiguracionInvalida = errors.New("auditoria_periodica_configuracion_invalida")

type errorLecturaConfiguracion struct{ causa error }

func (errorLecturaConfiguracion) Error() string     { return ErrConfiguracionInvalida.Error() }
func (e errorLecturaConfiguracion) Unwrap() error   { return e.causa }
func (errorLecturaConfiguracion) Is(err error) bool { return err == ErrConfiguracionInvalida }

var versionBinarioValida = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$`)

// Ejecutor no fija la periodicidad ni la política del sello. Ambas pertenecen
// a la fuente gobernada. El pin se coteja en composición antes de firmar.
type Ejecutor struct {
	Version         uint64 `json:"version"`
	MaxRegistros    uint64 `json:"max_registros"`
	VersionBinario  string `json:"version_binario"`
	PinSPKISHA256   string `json:"pin_spki_sha256"`
	TimeoutSegundos int64  `json:"timeout_segundos"`
}

func (c Ejecutor) Validar() error {
	if c.Version != 1 || c.MaxRegistros == 0 || c.MaxRegistros > 1000000 ||
		!versionBinarioValida.MatchString(c.VersionBinario) ||
		!domain.SHA256CheckpointValido(c.PinSPKISHA256) ||
		c.PinSPKISHA256 == strings.Repeat("0", 64) ||
		c.TimeoutSegundos <= 0 || c.TimeoutSegundos > 300 {
		return ErrConfiguracionInvalida
	}
	return nil
}

func (c Ejecutor) Timeout() time.Duration {
	if c.Validar() != nil {
		return 0
	}
	return time.Duration(c.TimeoutSegundos) * time.Second
}

// Decodificar exige un objeto cerrado y acotado. No admite DSN, credenciales,
// perfiles, política o valores de periodicidad en el archivo del ejecutor.
func Decodificar(raw []byte) (Ejecutor, error) {
	if len(raw) == 0 || len(raw) > 16*1024 {
		return Ejecutor{}, ErrConfiguracionInvalida
	}
	repetidas, err := clavesRepetidas(raw)
	if err != nil {
		return Ejecutor{}, errorLecturaConfiguracion{err}
	}
	if repetidas {
		return Ejecutor{}, ErrConfiguracionInvalida
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	var c Ejecutor
	var extra any
	if d.Decode(&c) != nil || !errors.Is(d.Decode(&extra), io.EOF) || c.Validar() != nil {
		return Ejecutor{}, ErrConfiguracionInvalida
	}
	return c, nil
}

func clavesRepetidas(raw []byte) (bool, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	primera, err := d.Token()
	if err != nil {
		return false, err
	}
	if primera != json.Delim('{') {
		return true, nil
	}
	vistas := make(map[string]bool)
	for d.More() {
		token, err := d.Token()
		clave, ok := token.(string)
		if err != nil {
			return false, err
		}
		if !ok || vistas[clave] {
			return true, nil
		}
		vistas[clave] = true
		var valor json.RawMessage
		if err := d.Decode(&valor); err != nil {
			return false, err
		}
	}
	return false, nil
}
