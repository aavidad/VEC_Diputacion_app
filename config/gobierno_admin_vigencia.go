package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

// EnvGobiernoAdminValidezHoras sustituye, si está presente, las horas del JSON
// privado de vec-gobierno-usuarios-admin. La ausencia conserva ese contrato.
const EnvGobiernoAdminValidezHoras = "VEC_GOBIERNO_ADMIN_VALIDEZ_HORAS"

var ErrGobiernoAdminValidez = errors.New("config: vigencia de gobierno ADMIN invalida")

// EntornoGobiernoAdmin usa las mismas tres llaves explícitas del servidor.
// Una selección desconocida o incompleta nunca habilita el plazo de desarrollo.
func EntornoGobiernoAdmin() Config {
	return Config{
		ExecutionProfile: os.Getenv(EnvExecutionProfile),
		AuthMode:         os.Getenv(EnvAuthMode),
		DevelopmentGuard: os.Getenv(EnvDevelopmentGuard),
	}.Normalize()
}

// ValidezGobiernoAdmin valida antes de convertir a Duration, sin desbordamiento.
// Cero en el JSON privado significa el valor anterior por defecto: 24 horas.
func (c Config) ValidezGobiernoAdmin(horas int, override string) (time.Duration, error) {
	c = c.Normalize()
	switch c.ExecutionProfile {
	case ExecutionProfileProduction, ExecutionProfileDevelopment, ExecutionProfileRRHHPresentation:
	default:
		return 0, ErrGobiernoAdminValidez
	}
	if override != "" {
		var err error
		horas, err = strconv.Atoi(override)
		if err != nil {
			return 0, errors.Join(ErrGobiernoAdminValidez, err)
		}
	}
	if horas == 0 && override == "" {
		horas = 24
	}
	maximo := 24
	if c.DevelopmentEnabledByDoubleKey() {
		maximo = 400 * 24
	}
	if horas < 1 || horas > maximo {
		return 0, ErrGobiernoAdminValidez
	}
	return time.Duration(horas) * time.Hour, nil
}

// ValidarValidezGobiernoAdmin protege también la llamada directa a bootstrap.
func (c Config) ValidarValidezGobiernoAdmin(v time.Duration) error {
	if v < time.Hour || v%time.Hour != 0 {
		return ErrGobiernoAdminValidez
	}
	_, err := c.ValidezGobiernoAdmin(int(v/time.Hour), "")
	return err
}
