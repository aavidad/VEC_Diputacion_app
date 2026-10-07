// Package domain contiene la valoración de méritos de Provisión. Los hechos
// pertenecen a Personal/RUM; este consumidor conserva solo su instantánea.
package domain

import b "vec-diputacion-granada/internal/shared/baremacion"

const VersionMotor = "provision.v1"

type Familia string

const (
	Grado             Familia = "grado"
	Antiguedad        Familia = "antiguedad"
	Permanencia       Familia = "permanencia"
	Cursos            Familia = "cursos"
	Titulaciones      Familia = "titulaciones"
	ValoracionTrabajo Familia = "valoracion_trabajo"
)

// Conversión siempre explícita: meses civiles completos; días/divisor exacto;
// o años enteros desde meses, añadiendo uno si el resto supera el umbral.
type Conversion struct {
	Metodo      string `json:"metodo"`
	Divisor     int64  `json:"divisor"`
	UmbralResto int64  `json:"umbral_resto"`
}

type Tramo struct {
	ID            string   `json:"id"`
	MinDiferencia int      `json:"min_diferencia"`
	MaxDiferencia int      `json:"max_diferencia"`
	Coeficiente   b.Puntos `json:"coeficiente"`
	Maximo        b.Puntos `json:"maximo"`
}

type Regla struct {
	ID                 string         `json:"id"`
	Familia            Familia        `json:"familia"`
	ReferenciaBase     string         `json:"referencia_base"`
	Coeficiente        b.Puntos       `json:"coeficiente"`
	Maximo             b.Puntos       `json:"maximo"`
	Redondeo           b.ModoRedondeo `json:"redondeo"`
	Diferencia         string         `json:"diferencia,omitempty"`
	MinDiferencia      int            `json:"min_diferencia,omitempty"`
	MaxDiferencia      int            `json:"max_diferencia,omitempty"`
	Agrupacion         string         `json:"agrupacion,omitempty"`
	Tramos             []Tramo        `json:"tramos,omitempty"`
	Tipos              []string       `json:"tipos,omitempty"`
	Conversion         *Conversion    `json:"conversion,omitempty"`
	Jornada            string         `json:"jornada,omitempty"` // integra, proporcional, protegida_integra
	Solapes            string         `json:"solapes,omitempty"` // rechazar
	HorasMinimas       *b.Racional    `json:"horas_minimas,omitempty"`
	ExcluirRequisito   bool           `json:"excluir_requisito"`
	SeleccionElementos string         `json:"seleccion_elementos,omitempty"`
	MaximoElementos    int            `json:"maximo_elementos,omitempty"`
	// Una política de permanencia distingue los meses provisionales sin
	// fijar en código el corrector ni la regla temporal de una convocatoria.
	PermanenciaPolitica         string `json:"permanencia_politica,omitempty"`
	TipoProvisional            string `json:"tipo_provisional,omitempty"`
	FactorProvisionalNumerador int64  `json:"factor_provisional_numerador,omitempty"`
	FactorProvisionalDenominador int64 `json:"factor_provisional_denominador,omitempty"`
}

type Configuracion struct {
	SchemaVersion   string       `json:"schema_version"`
	ConvocatoriaRef string       `json:"convocatoria_ref"`
	Version         string       `json:"version"`
	BasesRef        string       `json:"bases_ref"`
	// CoberturaRequerida obliga a declarar las seis familias aun cuando una
	// regla siga pendiente. Vacio conserva el contrato de ensayos anterior.
	CoberturaRequerida string       `json:"cobertura_requerida,omitempty"`
	FechaCorte      b.FechaCivil `json:"fecha_corte"` // extremo exclusivo explícito
	VentanaDesde    b.FechaCivil `json:"ventana_desde"`
	MaximoTotal     b.Puntos     `json:"maximo_total"`
	Reglas          []Regla      `json:"reglas"`
}

type Periodo struct {
	ID                     string            `json:"id"`
	EvidenciaRef           string            `json:"evidencia_ref"`
	Desde                  b.FechaCivil      `json:"desde"`
	Hasta                  *b.FechaCivil     `json:"hasta,omitempty"` // nil = abierto, corta en fecha_corte
	Nivel                  int               `json:"nivel"`
	Tipo                   string            `json:"tipo"`
	Familias               []Familia         `json:"familias"`
	Jornada                b.FraccionJornada `json:"jornada"`
	AtestacionProtegidaRef string            `json:"atestacion_protegida_ref,omitempty"`
}

type Curso struct {
	ID           string        `json:"id"`
	EvidenciaRef string        `json:"evidencia_ref"`
	Tipo         string        `json:"tipo"`
	Horas        b.Racional    `json:"horas"`
	Fecha        b.FechaCivil  `json:"fecha"`
	VigenteHasta *b.FechaCivil `json:"vigente_hasta,omitempty"`
	Relacionado  bool          `json:"relacionado"`
	Acreditado   bool          `json:"acreditado"`
}

type Titulo struct {
	ID             string       `json:"id"`
	EvidenciaRef   string       `json:"evidencia_ref"`
	Tipo           string       `json:"tipo"`
	Fecha          b.FechaCivil `json:"fecha"`
	Acreditado     bool         `json:"acreditado"`
	UsadoRequisito bool         `json:"usado_requisito"`
}

type Entrada struct {
	InstantaneaRef    string    `json:"instantanea_ref"`
	PuestoRef         string    `json:"puesto_ref"`
	NivelPuesto       int       `json:"nivel_puesto"`
	GradoPersonal     *int      `json:"grado_personal,omitempty"`
	GradoEvidenciaRef string    `json:"grado_evidencia_ref,omitempty"`
	Disponibles       []Familia `json:"disponibles"` // distingue fuente disponible sin méritos de fuente caída
	Periodos          []Periodo `json:"periodos"`
	Cursos            []Curso   `json:"cursos"`
	Titulaciones      []Titulo  `json:"titulaciones"`
}

type Detalle struct {
	HechoID       string     `json:"hecho_id"`
	EvidenciaRef  string     `json:"evidencia_ref"`
	TramoID       string     `json:"tramo_id,omitempty"`
	Motivo        string     `json:"motivo"`
	DiasBrutos    int64      `json:"dias_brutos,omitempty"`
	DiasElegibles int64      `json:"dias_elegibles,omitempty"`
	Unidades      b.Racional `json:"unidades"`
	FactorJornada b.Racional `json:"factor_jornada"`
	Coeficiente   b.Puntos   `json:"coeficiente"`
	Bruto         b.Puntos   `json:"bruto"`
	Maximo        b.Puntos   `json:"maximo"`
	Resultado     b.Puntos   `json:"resultado"`
}

type Desglose struct {
	ReglaID        string    `json:"regla_id"`
	Familia        Familia   `json:"familia"`
	ReferenciaBase string    `json:"referencia_base"`
	Estado         string    `json:"estado"`
	Detalles       []Detalle `json:"detalles"`
	Bruto          b.Puntos  `json:"bruto"`
	Maximo         b.Puntos  `json:"maximo"`
	Resultado      b.Puntos  `json:"resultado"`
}

type Resultado struct {
	Estado          string     `json:"estado"`
	VersionMotor    string     `json:"version_motor"`
	ConvocatoriaRef string     `json:"convocatoria_ref"`
	VersionReglas   string     `json:"version_reglas"`
	InstantaneaRef  string     `json:"instantanea_ref"`
	PuestoRef       string     `json:"puesto_ref"`
	HuellaReglas    string     `json:"huella_reglas"`
	HuellaEntrada   string     `json:"huella_entrada"`
	HuellaResultado string     `json:"huella_resultado"`
	Completo        bool       `json:"completo"`
	Incidencias     []string   `json:"incidencias"`
	Desglose        []Desglose `json:"desglose"`
	Bruto           b.Puntos   `json:"bruto"`
	MaximoTotal     b.Puntos   `json:"maximo_total"`
	Total           *b.Puntos  `json:"total"`
}
