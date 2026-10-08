package bootstrap

import (
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/catalogoalta"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const esquemaCatalogosAltaContratacionTemporalV2 = "vec.contratacion_temporal.catalogos_alta.v2"

type respuestaCatalogosAltaV2 struct {
	Data datosCatalogosAltaV2 `json:"data"`
}

type datosCatalogosAltaV2 struct {
	Esquema              string                                                        `json:"esquema"`
	NumeroExpedienteMOAD *domain.PoliticaNumeroExpediente                              `json:"numero_expediente_moad,omitempty"`
	Centros              []centroCatalogosAltaContratacionTemporalDesarrollo           `json:"centros"`
	Categorias           []categoriaCatalogosAltaContratacionTemporalDesarrollo        `json:"categorias"`
	Documentos           []opcionReferenciaCatalogosAltaContratacionTemporalDesarrollo `json:"documentos"`
	PreparacionVias      *preparacionViasCatalogosAltaJSON                             `json:"preparacion_vias,omitempty"`
	Necesidades          catalogoNecesidadesAltaV2JSON                                 `json:"necesidades"`
}

func datosCatalogosAltaV2Desde(actual catalogosAltaContratacionTemporalDesarrollo) (datosCatalogosAltaV2, error) {
	necesidades, err := catalogoNecesidadesAltaDesarrollo(actual.rutaNecesidades)
	if err != nil {
		return datosCatalogosAltaV2{}, err
	}
	return datosCatalogosAltaV2{
		Esquema:              esquemaCatalogosAltaContratacionTemporalV2,
		NumeroExpedienteMOAD: actual.NumeroExpedienteMOAD,
		Centros:              actual.Centros, Categorias: actual.Categorias, Documentos: actual.Documentos,
		PreparacionVias: preparacionViasCatalogosAlta(&actual), Necesidades: necesidades,
	}, nil
}

// catalogoNecesidadesAltaV2JSON expone la declaración del centro y su fuente.
// No publica modalidades de nombramiento ni acredita vacancia o financiación.
type catalogoNecesidadesAltaV2JSON struct {
	Referencia               string                     `json:"referencia"`
	Version                  uint64                     `json:"version"`
	HuellaSHA256             string                     `json:"huella_sha256"`
	EsEjemplo                bool                       `json:"es_ejemplo"`
	FuenteRef                string                     `json:"fuente_ref"`
	FuenteURL                string                     `json:"fuente_url"`
	JornadaReferenciaMinutos uint16                     `json:"jornada_referencia_minutos"`
	JornadaFuenteRef         string                     `json:"jornada_fuente_ref"`
	Causas                   []causaNecesidadAltaV2JSON `json:"causas"`
}

type causaNecesidadAltaV2JSON struct {
	Clave              string     `json:"clave"`
	EtiquetaClave      string     `json:"etiqueta_clave"`
	FuenteRef          string     `json:"fuente_ref"`
	FuenteURL          string     `json:"fuente_url"`
	ReglaRef           string     `json:"regla_ref"`
	FechaFin           string     `json:"fecha_fin"`
	CausaFin           string     `json:"causa_fin,omitempty"`
	MaximoMeses        uint8      `json:"maximo_meses"`
	VentanaMeses       uint8      `json:"ventana_meses,omitempty"`
	CamposPermitidos   []string   `json:"campos_permitidos"`
	CamposObligatorios []string   `json:"campos_obligatorios"`
	UnoDe              [][]string `json:"uno_de,omitempty"`
}

func catalogoNecesidadesAltaDesarrollo(rutas ...string) (catalogoNecesidadesAltaV2JSON, error) {
	if len(rutas) > 1 {
		return catalogoNecesidadesAltaV2JSON{}, errCatalogosAltaContratacionTemporalDesarrolloNoDisponibles
	}
	ruta := ""
	if len(rutas) == 1 {
		ruta = rutas[0]
	}
	c, err := catalogoalta.CargarNecesidades(ruta)
	if err != nil {
		return catalogoNecesidadesAltaV2JSON{}, err
	}
	resultado := catalogoNecesidadesAltaV2JSON{
		Referencia: c.Referencia, Version: c.Version, HuellaSHA256: c.HuellaSHA256,
		EsEjemplo: c.EsEjemplo, FuenteRef: c.FuenteRef, FuenteURL: c.FuenteURL,
		JornadaReferenciaMinutos: c.JornadaReferenciaMinutos, JornadaFuenteRef: c.JornadaFuenteRef,
		Causas: make([]causaNecesidadAltaV2JSON, 0, len(c.Causas)),
	}
	for _, causa := range c.Causas {
		opcion := causaNecesidadAltaV2JSON{
			Clave: string(causa.Clave), EtiquetaClave: causa.EtiquetaClave,
			FuenteRef: causa.FuenteRef, FuenteURL: causa.FuenteURL,
			ReglaRef: causa.ReglaRef, FechaFin: causa.FechaFin,
			MaximoMeses: causa.MaximoMeses, VentanaMeses: causa.VentanaMeses,
			CamposPermitidos:   append([]string(nil), causa.CamposPermitidos...),
			CamposObligatorios: append([]string(nil), causa.CamposObligatorios...),
			UnoDe:              make([][]string, len(causa.UnoDe)),
		}
		if causa.CausaFin != domain.ClaveCatalogo("") {
			opcion.CausaFin = string(causa.CausaFin)
		}
		for i, grupo := range causa.UnoDe {
			opcion.UnoDe[i] = append([]string(nil), grupo...)
		}
		resultado.Causas = append(resultado.Causas, opcion)
	}
	return resultado, nil
}
