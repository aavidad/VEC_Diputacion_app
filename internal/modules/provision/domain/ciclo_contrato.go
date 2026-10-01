package domain

const VersionCiclo = "provision.ciclo.ensayo.v1"

// CatalogoCausas identifica causas aportadas por el proceso. Su presencia en
// un ensayo no acredita que RRHH haya aprobado el catálogo ni las bases.
type CatalogoCausas struct {
	Version string             `json:"version"`
	Causas  []CausaReclamacion `json:"causas"`
}

type CausaReclamacion struct {
	Codigo         string `json:"codigo"`
	ReferenciaBase string `json:"referencia_base"`
}

type Reclamacion struct {
	Referencia        string `json:"referencia"`
	VersionValoracion uint32 `json:"version_valoracion"`
	HuellaValoracion  string `json:"huella_valoracion"`
	VersionCatalogo   string `json:"version_catalogo"`
	CausaCodigo       string `json:"causa_codigo"`
	EvidenciaRef      string `json:"evidencia_ref"`
}

type TipoDecision string

const (
	MantenerValoracion   TipoDecision = "mantener"
	RectificarValoracion TipoDecision = "rectificar"
)

// MotivacionRef identifica un documento motivado; no incorpora su contenido.
// La entrada corregida es otra instantánea de ensayo, no una edición de RUM.
type DecisionRevision struct {
	Referencia       string       `json:"referencia"`
	ReclamacionRef   string       `json:"reclamacion_ref"`
	VersionEsperada  uint32       `json:"version_esperada"`
	Tipo             TipoDecision `json:"tipo"`
	MotivacionRef    string       `json:"motivacion_ref"`
	EvidenciaRef     string       `json:"evidencia_ref"`
	EntradaCorregida *Entrada     `json:"entrada_corregida,omitempty"`
}

type ValoracionCiclo struct {
	Version              uint32    `json:"version"`
	VersionAnterior      uint32    `json:"version_anterior"`
	HuellaAnterior       string    `json:"huella_anterior"`
	HuellaRevision       string    `json:"huella_revision"`
	HuellaCatalogoCausas string    `json:"huella_catalogo_causas"`
	RevisionInicialRef   string    `json:"revision_inicial_ref,omitempty"`
	HuellaReclamacion    string    `json:"huella_reclamacion,omitempty"`
	HuellaDecision       string    `json:"huella_decision,omitempty"`
	ReclamacionRef       string    `json:"reclamacion_ref,omitempty"`
	DecisionRef          string    `json:"decision_ref,omitempty"`
	Entrada              Entrada   `json:"entrada"`
	Resultado            Resultado `json:"resultado"`
}

type ResolucionBorrador struct {
	Estado                  string   `json:"estado"`
	VersionValoracion       uint32   `json:"version_valoracion"`
	HuellaValoracion        string   `json:"huella_valoracion"`
	ReclamacionesPendientes []string `json:"reclamaciones_pendientes"`
	Pendientes              []string `json:"pendientes"`
	Firmada                 bool     `json:"firmada"`
	Publicada               bool     `json:"publicada"`
	EfectoOficial           bool     `json:"efecto_oficial"`
}

type CicloEnsayado struct {
	SchemaVersion      string             `json:"schema_version"`
	Alcance            string             `json:"alcance"`
	Configuracion      Configuracion      `json:"configuracion"`
	CatalogoCausas     CatalogoCausas     `json:"catalogo_causas"`
	RevisionInicialRef string             `json:"revision_inicial_ref,omitempty"`
	Valoraciones       []ValoracionCiclo  `json:"valoraciones"`
	Reclamaciones      []Reclamacion      `json:"reclamaciones"`
	Decisiones         []DecisionRevision `json:"decisiones"`
	Resolucion         ResolucionBorrador `json:"resolucion"`
}
