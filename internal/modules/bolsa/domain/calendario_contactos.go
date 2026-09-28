package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"time"
)

const EsquemaCalendarioContactos = "vec.bolsa.calendario-contactos.v1"

const TipoCalendarioHabilSede = "habil_administrativo_sede"

var ErrCalendarioContactosInvalido = errors.New("bolsa: calendario de contactos invalido")

var referenciaCalendarioContacto = regexp.MustCompile(`^[a-z][a-z0-9:_-]{1,95}$`)
var huellaCalendarioContacto = regexp.MustCompile(`^[0-9a-f]{64}$`)

// VersionFuenteCalendario identifica una versión publicada de Calendarios.
// La instantánea de Bolsa contiene la composición exacta de las fuentes que
// dieron el resultado anual, sin consultar ni copiar las tablas del módulo.
type VersionFuenteCalendario struct {
	AmbitoTipo string `json:"ambito_tipo"`
	AmbitoRef  string `json:"ambito_ref"`
	VersionID  string `json:"version_id"`
	Numero     int    `json:"numero"`
}

type DiaCalendarioContactos struct {
	Fecha string `json:"fecha"`
	Habil bool   `json:"habil"`
}

// CalendarioContactos es la proyección completa de un año publicada por la
// autoridad Calendarios para una sede. Cada fecha civil figura exactamente
// una vez; una ausencia no se interpreta como día inhábil.
type CalendarioContactos struct {
	Esquema         string                    `json:"esquema"`
	Tipo            string                    `json:"tipo"`
	SedeRef         string                    `json:"sede_ref"`
	Anio            int                       `json:"anio"`
	Version         uint64                    `json:"version"`
	VersionAnterior string                    `json:"version_anterior,omitempty"`
	Fuentes         []VersionFuenteCalendario `json:"fuentes"`
	Dias            []DiaCalendarioContactos  `json:"dias"`
	HuellaSHA256    string                    `json:"huella_sha256"`
}

type materialCalendarioContactos struct {
	Esquema         string                    `json:"esquema"`
	Tipo            string                    `json:"tipo"`
	SedeRef         string                    `json:"sede_ref"`
	Anio            int                       `json:"anio"`
	Version         uint64                    `json:"version"`
	VersionAnterior string                    `json:"version_anterior,omitempty"`
	Fuentes         []VersionFuenteCalendario `json:"fuentes"`
	Dias            []DiaCalendarioContactos  `json:"dias"`
}

func (c CalendarioContactos) material() materialCalendarioContactos {
	return materialCalendarioContactos{
		Esquema: c.Esquema, Tipo: c.Tipo, SedeRef: c.SedeRef, Anio: c.Anio,
		Version: c.Version, VersionAnterior: c.VersionAnterior,
		Fuentes: c.Fuentes, Dias: c.Dias,
	}
}

// HuellaCanonica calcula la huella del material exacto antes de importarlo.
// El orden de fuentes y días es parte del contrato; Validar exige orden único.
func (c CalendarioContactos) HuellaCanonica() (string, error) {
	contenido, err := c.MaterialCanonico()
	if err != nil {
		return "", ErrCalendarioContactosInvalido
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

// MaterialCanonico es la preimagen exacta que el adaptador transmite a SQL;
// PostgreSQL vuelve a calcular su SHA-256 antes de guardarla.
func (c CalendarioContactos) MaterialCanonico() ([]byte, error) {
	if err := c.validarEstructura(); err != nil {
		return nil, err
	}
	contenido, err := json.Marshal(c.material())
	if err != nil {
		return nil, ErrCalendarioContactosInvalido
	}
	return contenido, nil
}

func (c CalendarioContactos) Validar() error {
	huella, err := c.HuellaCanonica()
	if err != nil || !huellaCalendarioContacto.MatchString(c.HuellaSHA256) || huella != c.HuellaSHA256 {
		return ErrCalendarioContactosInvalido
	}
	return nil
}

func (c CalendarioContactos) validarEstructura() error {
	if c.Esquema != EsquemaCalendarioContactos || c.Tipo != TipoCalendarioHabilSede ||
		!referenciaCalendarioContacto.MatchString(c.SedeRef) || c.Anio < 2000 || c.Anio > 2100 ||
		c.Version == 0 || c.Version > 10000 || len(c.Fuentes) != 3 {
		return ErrCalendarioContactosInvalido
	}
	if (c.Version == 1 && c.VersionAnterior != "") ||
		(c.Version > 1 && !huellaCalendarioContacto.MatchString(c.VersionAnterior)) {
		return ErrCalendarioContactosInvalido
	}
	for i, tipo := range []string{"nacional", "autonomico", "local"} {
		f := c.Fuentes[i]
		if f.AmbitoTipo != tipo || !referenciaCalendarioContacto.MatchString(f.AmbitoRef) ||
			!referenciaCalendarioContacto.MatchString(f.VersionID) || f.Numero < 1 || f.Numero > 10000 {
			return ErrCalendarioContactosInvalido
		}
	}
	primero := time.Date(c.Anio, 1, 1, 0, 0, 0, 0, time.UTC)
	siguiente := primero.AddDate(1, 0, 0)
	if len(c.Dias) != int(siguiente.Sub(primero).Hours()/24) {
		return ErrCalendarioContactosInvalido
	}
	for i, dia := range c.Dias {
		if dia.Fecha != primero.AddDate(0, 0, i).Format("2006-01-02") {
			return ErrCalendarioContactosInvalido
		}
	}
	return nil
}
