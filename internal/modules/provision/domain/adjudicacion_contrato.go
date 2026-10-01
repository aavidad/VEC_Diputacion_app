package domain

// VersionAdjudicacion identifica un contrato de ensayo sin efectos administrativos.
const VersionAdjudicacion = "provision.adjudicacion.v1"
const MetodoAdjudicacionEnsayo = "aceptacion_diferida_personas_proponen_ensayo_v1"

// DesempateAdjudicacion consume un desglose ya calculado por el motor común.
// El orden de la colección es la cadena explícita de la política de ensayo.
type DesempateAdjudicacion struct {
	ReglaID string `json:"regla_id"`
	Sentido string `json:"sentido"` // mayor, menor
}

type ConfiguracionAdjudicacion struct {
	SchemaVersion           string                  `json:"schema_version"`
	ProcesoRef              string                  `json:"proceso_ref"`
	Version                 string                  `json:"version"`
	BasesRef                string                  `json:"bases_ref"`
	HuellaBases             string                  `json:"huella_bases"`
	PoliticaRef             string                  `json:"politica_ref"`
	PoliticaVersion         string                  `json:"politica_version"`
	Metodo                  string                  `json:"metodo"`
	Prioridad               string                  `json:"prioridad"`        // total_descendente
	Incompatibilidad        string                  `json:"incompatibilidad"` // un_puesto_por_persona
	Desempates              []DesempateAdjudicacion `json:"desempates"`
	VersionMotorValoracion  string                  `json:"version_motor_valoracion"`
	VersionReglasValoracion string                  `json:"version_reglas_valoracion"`
	HuellaReglasValoracion  string                  `json:"huella_reglas_valoracion"`
}

type VacanteAdjudicacion struct {
	VacanteRef string `json:"vacante_ref"`
	PuestoRef  string `json:"puesto_ref"`
}

// PreferenciaAdjudicacion referencia el resultado propio de Provisión. Su
// huella permite comprobar integridad reproducible, nunca autenticidad o firma.
// Exclusión y renuncia son declaraciones sintéticas de entrada sin efectos.
type PreferenciaAdjudicacion struct {
	VacanteRef string     `json:"vacante_ref"`
	Admision   string     `json:"admision"` // admitida, excluida, pendiente, renunciada
	Valoracion *Resultado `json:"valoracion"`
}

type SolicitudAdjudicacion struct {
	SolicitudRef   string                    `json:"solicitud_ref"`
	Version        string                    `json:"version"`
	PersonaRef     string                    `json:"persona_ref"`
	InstantaneaRef string                    `json:"instantanea_ref"`
	Preferencias   []PreferenciaAdjudicacion `json:"preferencias"`
}

// EntradaAdjudicacion declara un universo cerrado para un ensayo sintético.
// La secuencia de preferencias sí tiene significado; los otros órdenes no.
type EntradaAdjudicacion struct {
	UniversoRef     string                  `json:"universo_ref"`
	UniversoVersion string                  `json:"universo_version"`
	Cerrado         bool                    `json:"cerrado"`
	Sintetico       bool                    `json:"sintetico"`
	Vacantes        []VacanteAdjudicacion   `json:"vacantes"`
	Solicitudes     []SolicitudAdjudicacion `json:"solicitudes"`
}

type AsignacionSimulada struct {
	PersonaRef       string `json:"persona_ref"`
	SolicitudRef     string `json:"solicitud_ref"`
	SolicitudVersion string `json:"solicitud_version"`
	VacanteRef       string `json:"vacante_ref"`
	PuestoRef        string `json:"puesto_ref"`
	Preferencia      int    `json:"preferencia"`
	HuellaValoracion string `json:"huella_valoracion"`
}

type IncidenciaAdjudicacion struct {
	Codigo       string `json:"codigo"`
	SolicitudRef string `json:"solicitud_ref,omitempty"`
	VacanteRef   string `json:"vacante_ref,omitempty"`
}

type ResultadoAdjudicacion struct {
	SchemaVersion         string                   `json:"schema_version"`
	Alcance               string                   `json:"alcance"`
	Estado                string                   `json:"estado"`
	ProcesoRef            string                   `json:"proceso_ref"`
	Version               string                   `json:"version"`
	PoliticaRef           string                   `json:"politica_ref"`
	PoliticaVersion       string                   `json:"politica_version"`
	Metodo                string                   `json:"metodo"`
	UniversoRef           string                   `json:"universo_ref"`
	UniversoVersion       string                   `json:"universo_version"`
	HuellaConfiguracion   string                   `json:"huella_configuracion"`
	HuellaEntrada         string                   `json:"huella_entrada"`
	HuellaResultado       string                   `json:"huella_resultado"`
	Asignaciones          []AsignacionSimulada     `json:"asignaciones"`
	PersonasSinAsignacion []string                 `json:"personas_sin_asignacion"`
	VacantesSinAsignacion []string                 `json:"vacantes_sin_asignacion"`
	Incidencias           []IncidenciaAdjudicacion `json:"incidencias"`
}
