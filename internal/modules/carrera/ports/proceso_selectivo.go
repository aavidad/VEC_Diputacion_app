package ports

import "context"

// Contrato de preparación sintética. El lector nominal de Selección, Personal,
// RUM y la autorización H08 siguen pendientes; no es un permiso de acceso.
type RequisitoPromocionSintetico struct {
	Referencia     string `json:"referencia"`
	Version        string `json:"version"`
	Hito           string `json:"hito"`
	HitoFecha      string `json:"hito_fecha"`
	Representacion string `json:"representacion"`
	ReglaRef       string `json:"regla_ref"`
	Fuente         string `json:"fuente"`
	FuenteVersion  string `json:"fuente_version"`
}

type HechoPromocionSintetico struct {
	Referencia     string `json:"referencia"`
	Version        string `json:"version"`
	EstadoAportado string `json:"estado_aportado"`
	Fuente         string `json:"fuente"`
	FuenteVersion  string `json:"fuente_version"`
	Evidencia      string `json:"evidencia"`
	VigenteDesde   string `json:"vigente_desde"`
	VigenteHasta   string `json:"vigente_hasta"`
}

type ConsultaPromocionSintetica struct {
	Alcance                  string                        `json:"alcance"`
	CasoRef                  string                        `json:"caso_ref"`
	PersonaRef               string                        `json:"persona_ref"`
	ProcesoRef               string                        `json:"proceso_ref"`
	ProcesoVersion           string                        `json:"proceso_version"`
	BasesRef                 string                        `json:"bases_ref"`
	BasesVersion             string                        `json:"bases_version"`
	HuellaBasesAportada      string                        `json:"huella_bases_aportada"`
	InstantaneaHechosRef     string                        `json:"instantanea_hechos_ref"`
	InstantaneaHechosVersion string                        `json:"instantanea_hechos_version"`
	InstanteReferencia       string                        `json:"instante_referencia"`
	Requisitos               []RequisitoPromocionSintetico `json:"requisitos"`
	Hechos                   []HechoPromocionSintetico     `json:"hechos"`
}

type ComprobacionPromocionSintetica struct {
	RequisitoRef      string   `json:"requisito_ref"`
	RequisitoVersion  string   `json:"requisito_version"`
	Hito              string   `json:"hito"`
	HitoFecha         string   `json:"hito_fecha"`
	EstadoAportado    string   `json:"estado_aportado"`
	MotivoClave       string   `json:"motivo_clave"`
	Fuente            string   `json:"fuente"`
	FuenteVersion     string   `json:"fuente_version"`
	HechosReferencias []string `json:"hechos_referencias"`
}

type DictamenPromocionSintetico struct {
	Alcance              string                           `json:"alcance"`
	HuellaConsultaSHA256 string                           `json:"huella_consulta_sha256"`
	EvaluadorRef         string                           `json:"evaluador_ref"`
	EvaluadorVersion     string                           `json:"evaluador_version"`
	Referencia           string                           `json:"referencia"`
	Version              string                           `json:"version"`
	EvaluadoEn           string                           `json:"evaluado_en"`
	Comprobaciones       []ComprobacionPromocionSintetica `json:"comprobaciones"`
}

type LectorCotejoProcesoSelectivoSintetico interface {
	ConsultarCotejoPromocionSintetico(context.Context, ConsultaPromocionSintetica) (DictamenPromocionSintetico, error)
}
