package httpcopias

import (
	"context"
	"time"
)

// VersionObservada describes authenticated observed bytes, not package forecasts
// or the current repository HEAD. Unknown values must stay unknown.
type VersionObservada struct {
	ReleaseRef             string `json:"release_ref"`
	AppVersion             string `json:"app_version"`
	PostgreSQLVersion      string `json:"postgresql_version"`
	EsquemaRef             string `json:"esquema_ref"`
	DescriptorHuellaSHA256 string `json:"descriptor_huella_sha256"`
}
type CompatibilidadRevision struct {
	Estado  string   `json:"estado"`
	Razones []string `json:"razones"`
}

// MetadatosRevision comes only from the verified manifest and the current
// observed destination. No metadata, version, date or reason is accepted from HTTP.
type MetadatosRevision struct {
	PropuestaRef          string                 `json:"propuesta_ref"`
	PropuestaHuellaSHA256 string                 `json:"propuesta_huella_sha256"`
	ConjuntoRef           string                 `json:"conjunto_ref"`
	ConjuntoHuellaSHA256  string                 `json:"conjunto_huella_sha256"`
	DestinoRef            string                 `json:"destino_ref"`
	PreimagenSHA256       string                 `json:"preimagen_sha256"`
	FechaCopia            time.Time              `json:"fecha_copia"`
	PerdidaDesde          time.Time              `json:"perdida_desde"`
	Actual                VersionObservada       `json:"actual"`
	Resultante            VersionObservada       `json:"resultante"`
	Compatibilidad        CompatibilidadRevision `json:"compatibilidad"`
	ObservadaEn           time.Time              `json:"observada_en"`
}

// FuenteRevision rechecks authenticated manifest provenance, current preimage,
// authorization and read audit on every call. It must not fill absent values
// from defaults or client data. Control must recheck those bindings at its effect.
type FuenteRevision interface {
	PropuestaParaRevision(context.Context, Sesion, string) (Propuesta, error)
}
