package domain

const VersionProceso = "provision.proceso.v1"

// EstadoRequisito expresa una comprobación separada de la valoración de méritos.
// En el simulador se recibe como dato sintético; no acredita admisión administrativa.
type EstadoRequisito string

const (
	Cumple    EstadoRequisito = "cumple"
	NoCumple  EstadoRequisito = "no_cumple"
	Pendiente EstadoRequisito = "pendiente"
)

type RequisitoProvision struct {
	Referencia string `json:"referencia"`
	Version    string `json:"version"`
}

// PuestoOfertado conserva la referencia y versión exacta de la fuente RPT.
// Su presencia en una simulación no acredita vacancia ni publicación oficial.
type PuestoOfertado struct {
	Referencia string               `json:"referencia"`
	RPTRef     string               `json:"rpt_ref"`
	RPTVersion string               `json:"rpt_version"`
	Nivel      int                  `json:"nivel"`
	Requisitos []RequisitoProvision `json:"requisitos"`
}

type ProcesoProvision struct {
	SchemaVersion string           `json:"schema_version"`
	Referencia    string           `json:"referencia"`
	Version       string           `json:"version"`
	Estado        string           `json:"estado"`
	Configuracion Configuracion    `json:"configuracion"`
	Puestos       []PuestoOfertado `json:"puestos"`
}

// InstantaneaEmpleado identifica los datos suministrados al ejercicio. La
// condición interna declarada no sustituye una consulta autorizada a Personal.
type InstantaneaEmpleado struct {
	Referencia       string          `json:"referencia"`
	EmpleadoRef      string          `json:"empleado_ref"`
	Version          string          `json:"version"`
	FuenteRef        string          `json:"fuente_ref"`
	CondicionInterna EstadoRequisito `json:"condicion_interna"`
}

type PreferenciaProvision struct {
	Orden     int    `json:"orden"`
	PuestoRef string `json:"puesto_ref"`
}

type ComprobacionRequisito struct {
	RequisitoRef     string          `json:"requisito_ref"`
	RequisitoVersion string          `json:"requisito_version"`
	Estado           EstadoRequisito `json:"estado"`
	MotivoCodigo     string          `json:"motivo_codigo"`
	FuenteRef        string          `json:"fuente_ref"`
}

type EntradaValoracionPuesto struct {
	PuestoRef  string                  `json:"puesto_ref"`
	Requisitos []ComprobacionRequisito `json:"requisitos"`
	Entrada    Entrada                 `json:"entrada"`
}

type SolicitudProvision struct {
	Referencia     string                    `json:"referencia"`
	ProcesoRef     string                    `json:"proceso_ref"`
	ProcesoVersion string                    `json:"proceso_version"`
	EmpleadoRef    string                    `json:"empleado_ref"`
	VersionReglas  string                    `json:"version_reglas"`
	Instantanea    InstantaneaEmpleado       `json:"instantanea"`
	Preferencias   []PreferenciaProvision    `json:"preferencias"`
	Valoraciones   []EntradaValoracionPuesto `json:"valoraciones"`
}

type ValoracionPuesto struct {
	PuestoRef        string                  `json:"puesto_ref"`
	Orden            int                     `json:"orden"`
	RequisitosEstado EstadoRequisito         `json:"requisitos_estado"`
	Requisitos       []ComprobacionRequisito `json:"requisitos"`
	Resultado        Resultado               `json:"resultado"`
}

type ResultadoProceso struct {
	HuellaSimulacion string             `json:"huella_simulacion"`
	SchemaVersion    string             `json:"schema_version"`
	Alcance          string             `json:"alcance"`
	Estado           string             `json:"estado"`
	Proceso          ProcesoProvision   `json:"proceso"`
	Solicitud        SolicitudProvision `json:"solicitud"`
	Valoraciones     []ValoracionPuesto `json:"valoraciones"`
}

// HuellaSimulacionProceso vincula todo el ejercicio mediante el JSON tipado.
// Conserva el orden de las colecciones de entrada; no es firma ni recibo.
func HuellaSimulacionProceso(r ResultadoProceso) string {
	r.HuellaSimulacion = ""
	return huella(r)
}
