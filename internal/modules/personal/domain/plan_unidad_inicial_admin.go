package domain

import "time"

// PlanUnidadInicialAdminV1 prepara un único nodo de una fuente declarada
// sintética. No identifica personas ni crea cuentas, perfiles o permisos.
// La autoridad de Personal coteja LOGIN, configuración, aprobación y preimagen.
type PlanUnidadInicialAdminV1 struct {
	Version        uint64                      `json:"version"`
	OperacionRef   string                      `json:"operacion_ref"`
	PreparadoEn    time.Time                   `json:"preparado_en"`
	CaducaEn       time.Time                   `json:"caduca_en"`
	Entorno        string                      `json:"entorno"`
	AlcanceFuente  string                      `json:"alcance_fuente"`
	Unidad         UnidadInicialAdmin          `json:"unidad"`
	ActoTecnicoRef string                      `json:"acto_tecnico_ref"`
	Fuente         EvidenciaUnidadInicialAdmin `json:"fuente"`
}
type UnidadInicialAdmin struct {
	NodoRef              string     `json:"nodo_ref"`
	OrganizacionRef      string     `json:"organizacion_ref"`
	UnidadRef            string     `json:"unidad_ref"`
	Clase                string     `json:"clase"`
	Denominacion         string     `json:"denominacion"`
	CatalogoRef          string     `json:"catalogo_ref"`
	CatalogoVersion      uint64     `json:"catalogo_version"`
	CatalogoRevision     uint64     `json:"catalogo_revision"`
	CatalogoEntradaClave string     `json:"catalogo_entrada_clave"`
	RevisionEsperada     uint64     `json:"revision_esperada"`
	VigenteDesde         FechaCivil `json:"vigente_desde"`
	VigenteHasta         FechaCivil `json:"vigente_hasta"`
}
type EvidenciaUnidadInicialAdmin struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

// FuenteUnidadInicialAdminV1 contiene los metadatos exactos de la unidad. Su
// canon no incluye su propia huella, para evitar compromisos circulares.
type FuenteUnidadInicialAdminV1 struct {
	Version        uint64             `json:"version"`
	Referencia     string             `json:"referencia"`
	Entorno        string             `json:"entorno"`
	AlcanceFuente  string             `json:"alcance_fuente"`
	Unidad         UnidadInicialAdmin `json:"unidad"`
	ActoTecnicoRef string             `json:"acto_tecnico_ref"`
}
