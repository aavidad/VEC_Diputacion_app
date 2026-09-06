package ports

import "context"

const EsquemaEstructuraOrganizativa = "personal.estructura_organizativa.v1"

type UnidadEstructuraOrganizativa struct {
	Clave                string `json:"clave"`
	Etiqueta             string `json:"etiqueta"`
	Tipo                 string `json:"tipo"`
	AdscripcionClave     string `json:"adscripcion_clave,omitempty"`
	CodigoFuente         string `json:"codigo_fuente,omitempty"`
	PaginaFuente         int    `json:"pagina_fuente,omitempty"`
	ModificadaLocalmente bool   `json:"modificada_localmente,omitempty"`
}

type EstructuraOrganizativaConsultable struct {
	Esquema              string                         `json:"esquema"`
	CatalogoID           string                         `json:"catalogo_id"`
	CatalogoVersion      int                            `json:"catalogo_version"`
	CatalogoRevision     int                            `json:"catalogo_revision"`
	CatalogoHuellaSHA256 string                         `json:"catalogo_huella_sha256"`
	Estado               string                         `json:"estado"`
	FuenteRef            string                         `json:"fuente_ref"`
	Descripcion          string                         `json:"descripcion,omitempty"`
	Unidades             []UnidadEstructuraOrganizativa `json:"unidades"`
}

type ConsultaEstructuraOrganizativa interface {
	Obtener(context.Context) (EstructuraOrganizativaConsultable, error)
}
