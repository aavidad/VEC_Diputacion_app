package ports

import (
	"time"

	cal "vec-diputacion-granada/internal/modules/calendarios/domain"
)

// Estos DTO son exclusivamente de ensayo. No atestan adscripciones de Personal.
type FuenteEnsayoCalendario struct {
	Referencia string `json:"referencia"`
	SHA256     string `json:"sha256"`
}

type AdscripcionEnsayoCalendario struct {
	Ref       string                 `json:"ref"`
	Version   int                    `json:"version"`
	CentroRef string                 `json:"centro_ref"`
	Desde     cal.FechaCivil         `json:"desde"`
	Hasta     cal.FechaCivil         `json:"hasta"`
	Fuente    FuenteEnsayoCalendario `json:"fuente"`
}

type SnapshotEnsayoCalendario struct {
	Sintetico     bool                          `json:"sintetico"`
	Ref           string                        `json:"ref"`
	PersonaRef    string                        `json:"persona_ref"`
	ConocidoEn    time.Time                     `json:"conocido_en"`
	Fuente        FuenteEnsayoCalendario        `json:"fuente"`
	Adscripciones []AdscripcionEnsayoCalendario `json:"adscripciones"`
}

type SolicitudEnsayoCalendario struct {
	Desde      cal.FechaCivil           `json:"desde"`
	Hasta      cal.FechaCivil           `json:"hasta"`
	ConocidoEn time.Time                `json:"conocido_en"`
	Snapshot   SnapshotEnsayoCalendario `json:"snapshot"`
}

type CalendarioTramoEnsayo struct {
	CentroRef    string                  `json:"centro_ref"`
	Anio         int                     `json:"anio"`
	MunicipioRef string                  `json:"municipio_ref"`
	Zona         string                  `json:"zona"`
	Versiones    []cal.VersionCalendario `json:"versiones"`
	// Huella del resultado anual canónico, no de un documento oficial.
	HuellaResultadoAnual string              `json:"huella_resultado_anual"`
	Dias                 []cal.Clasificacion `json:"dias"`
}

type TramoEnsayoCalendario struct {
	Desde                cal.FechaCivil                `json:"desde"`
	Hasta                cal.FechaCivil                `json:"hasta"`
	Estado               string                        `json:"estado_determinacion"`
	Adscripciones        []AdscripcionEnsayoCalendario `json:"adscripciones"`
	Carencias            []string                      `json:"carencias"`
	AmbitosSinCalendario []cal.Ambito                  `json:"ambitos_sin_calendario,omitempty"`
	Calendario           *CalendarioTramoEnsayo        `json:"calendario,omitempty"`
}

type ResultadoEnsayoCalendario struct {
	Demostracion   bool                    `json:"demostracion"`
	PersonaRef     string                  `json:"persona_ref"`
	SnapshotRef    string                  `json:"snapshot_ref"`
	FuenteSnapshot FuenteEnsayoCalendario  `json:"fuente_snapshot"`
	HuellaSnapshot string                  `json:"huella_snapshot"`
	Desde          cal.FechaCivil          `json:"desde"`
	Hasta          cal.FechaCivil          `json:"hasta"`
	ConocidoEn     time.Time               `json:"conocido_en"`
	Estado         string                  `json:"estado_determinacion"`
	Tramos         []TramoEnsayoCalendario `json:"tramos"`
	// Ausencia explícita de programación individual, no cero minutos.
	PersonaProgramada *bool    `json:"persona_programada"`
	MinutosTeoricos   *int     `json:"minutos_teoricos"`
	Carencias         []string `json:"carencias"`
}
