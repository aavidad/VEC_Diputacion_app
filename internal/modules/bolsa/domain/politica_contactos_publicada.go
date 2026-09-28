package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"time"
)

const EsquemaPoliticaContactos = "vec.bolsa.politica-contactos.v1"

var ErrPoliticaContactosPublicadaInvalida = errors.New("bolsa: politica publicada de contactos invalida")

var bolsaPoliticaContactos = regexp.MustCompile(`^bolsa:[A-Za-z0-9:_-]{1,250}$`)
var referenciaReglaContactos = regexp.MustCompile(`^[a-z][a-z0-9:._-]{2,255}$`)

// PoliticaContactosPublicada es la regla versionada de Bolsa. Su huella y
// versión son independientes de la instantánea anual de Calendarios.
type PoliticaContactosPublicada struct {
	Esquema               string   `json:"esquema"`
	BolsaRef              string   `json:"bolsa_ref"`
	Version               uint64   `json:"version"`
	VersionAnterior       string   `json:"version_anterior,omitempty"`
	CatalogoRef           string   `json:"catalogo_ref"`
	CatalogoHuellaSHA256  string   `json:"catalogo_huella_sha256"`
	TipoDia               string   `json:"tipo_dia"`
	SedeRef               string   `json:"sede_ref"`
	Zona                  string   `json:"zona"`
	DesdeMinuto           int      `json:"desde_minuto"`
	HastaMinuto           int      `json:"hasta_minuto"`
	ControlFranja         string   `json:"control_franja"`
	IntentosPorCiclo      int      `json:"intentos_por_ciclo"`
	Ciclos                int      `json:"ciclos"`
	SeparacionSegundos    int      `json:"separacion_segundos"`
	ControlSeparacion     string   `json:"control_separacion"`
	ResultadosSinContacto []string `json:"resultados_sin_contacto"`
	HuellaSHA256          string   `json:"huella_sha256"`
}

type materialPoliticaContactos struct {
	Esquema               string   `json:"esquema"`
	BolsaRef              string   `json:"bolsa_ref"`
	Version               uint64   `json:"version"`
	VersionAnterior       string   `json:"version_anterior,omitempty"`
	CatalogoRef           string   `json:"catalogo_ref"`
	CatalogoHuellaSHA256  string   `json:"catalogo_huella_sha256"`
	TipoDia               string   `json:"tipo_dia"`
	SedeRef               string   `json:"sede_ref"`
	Zona                  string   `json:"zona"`
	DesdeMinuto           int      `json:"desde_minuto"`
	HastaMinuto           int      `json:"hasta_minuto"`
	ControlFranja         string   `json:"control_franja"`
	IntentosPorCiclo      int      `json:"intentos_por_ciclo"`
	Ciclos                int      `json:"ciclos"`
	SeparacionSegundos    int      `json:"separacion_segundos"`
	ControlSeparacion     string   `json:"control_separacion"`
	ResultadosSinContacto []string `json:"resultados_sin_contacto"`
}

func (p PoliticaContactosPublicada) material() materialPoliticaContactos {
	return materialPoliticaContactos{
		Esquema: p.Esquema, BolsaRef: p.BolsaRef, Version: p.Version, VersionAnterior: p.VersionAnterior,
		CatalogoRef: p.CatalogoRef, CatalogoHuellaSHA256: p.CatalogoHuellaSHA256,
		TipoDia: p.TipoDia, SedeRef: p.SedeRef, Zona: p.Zona,
		DesdeMinuto: p.DesdeMinuto, HastaMinuto: p.HastaMinuto, ControlFranja: p.ControlFranja,
		IntentosPorCiclo: p.IntentosPorCiclo, Ciclos: p.Ciclos, SeparacionSegundos: p.SeparacionSegundos,
		ControlSeparacion: p.ControlSeparacion, ResultadosSinContacto: p.ResultadosSinContacto,
	}
}

func (p PoliticaContactosPublicada) MaterialCanonico() ([]byte, error) {
	if p.validarEstructura() != nil {
		return nil, ErrPoliticaContactosPublicadaInvalida
	}
	material, err := json.Marshal(p.material())
	if err != nil {
		return nil, ErrPoliticaContactosPublicadaInvalida
	}
	return material, nil
}

func (p PoliticaContactosPublicada) HuellaCanonica() (string, error) {
	material, err := p.MaterialCanonico()
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(material)
	return hex.EncodeToString(suma[:]), nil
}

func (p PoliticaContactosPublicada) Validar() error {
	huella, err := p.HuellaCanonica()
	if err != nil || p.HuellaSHA256 != huella {
		return ErrPoliticaContactosPublicadaInvalida
	}
	return nil
}

func (p PoliticaContactosPublicada) validarEstructura() error {
	if p.Esquema != EsquemaPoliticaContactos || !bolsaPoliticaContactos.MatchString(p.BolsaRef) ||
		p.Version == 0 || p.Version > 10000 || !referenciaReglaContactos.MatchString(p.CatalogoRef) ||
		!huellaCalendarioContacto.MatchString(p.CatalogoHuellaSHA256) ||
		p.TipoDia != TipoCalendarioHabilSede || !referenciaCalendarioContacto.MatchString(p.SedeRef) ||
		p.Zona != "Europe/Madrid" || p.ControlFranja != ControlReglaImpedir ||
		p.ControlSeparacion != ControlReglaImpedir ||
		p.DesdeMinuto < 0 || p.DesdeMinuto >= p.HastaMinuto || p.HastaMinuto > 1440 ||
		p.IntentosPorCiclo < 1 || p.IntentosPorCiclo > maximoIntentosPorProceso ||
		p.Ciclos < 2 || p.Ciclos > maximoProcesos || p.SeparacionSegundos < 0 ||
		p.SeparacionSegundos > int(maximaSeparacion/time.Second) ||
		len(p.ResultadosSinContacto) == 0 || len(p.ResultadosSinContacto) > maximoResultadosSin {
		return ErrPoliticaContactosPublicadaInvalida
	}
	if (p.Version == 1 && p.VersionAnterior != "") ||
		(p.Version > 1 && !huellaCalendarioContacto.MatchString(p.VersionAnterior)) {
		return ErrPoliticaContactosPublicadaInvalida
	}
	for i, resultado := range p.ResultadosSinContacto {
		if _, existe := resultadosContacto[resultado]; !existe || slices.Contains(p.ResultadosSinContacto[:i], resultado) {
			return ErrPoliticaContactosPublicadaInvalida
		}
	}
	return nil
}
