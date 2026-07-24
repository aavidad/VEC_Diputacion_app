package canonico

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"time"

	dominiopublico "vec-diputacion-granada/internal/modules/bolsa/publico/dominio"
)

const EsquemaResumenConvocatoriaV2 = "vec.bolsa.resumen-convocatoria-publica.canonico.v2"

var ErrResumenConvocatoriaPublicaInvalido = errors.New("bolsa publica: resumen canonico invalido")

// ResumenConvocatoriaV2 fija exactamente el material que puede aparecer en
// listados. No depende del instante de consulta: conserva todos los plazos y
// deja que la aplicacion elija el destacado.
type ResumenConvocatoriaV2 struct {
	Esquema              string                         `json:"esquema"`
	IdentificadorPublico string                         `json:"identificador_publico"`
	Version              string                         `json:"version"`
	Estado               string                         `json:"estado"`
	Tipo                 string                         `json:"tipo"`
	CatalogoCategorias   ReferenciaCatalogoCategoriasV2 `json:"catalogo_categorias"`
	Categorias           []string                       `json:"categorias"`
	Titulo               string                         `json:"titulo"`
	Resumen              string                         `json:"resumen"`
	PublicadaEn          time.Time                      `json:"publicada_en"`
	ActualizadaEn        time.Time                      `json:"actualizada_en"`
	Plazos               []PlazoConvocatoriaV1          `json:"plazos"`
	NumeroRequisitos     int                            `json:"numero_requisitos"`
	NumeroDocumentos     int                            `json:"numero_documentos"`
	NumeroAyudas         int                            `json:"numero_ayudas"`
	HuellaCompletaSHA256 string                         `json:"huella_completa_sha256"`
}

func (r ResumenConvocatoriaV2) HuellaSHA256() (string, error) {
	if err := r.Validar(); err != nil {
		return "", err
	}
	canonico := r
	canonico.Categorias = append([]string(nil), r.Categorias...)
	canonico.Plazos = append([]PlazoConvocatoriaV1(nil), r.Plazos...)
	sort.Strings(canonico.Categorias)
	sort.Slice(canonico.Plazos, func(i, j int) bool {
		return canonico.Plazos[i].Referencia < canonico.Plazos[j].Referencia
	})
	contenido, err := json.Marshal(canonico)
	if err != nil {
		return "", ErrResumenConvocatoriaPublicaInvalido
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func (r ResumenConvocatoriaV2) Validar() error {
	if r.Esquema != EsquemaResumenConvocatoriaV2 {
		return ErrResumenConvocatoriaPublicaInvalido
	}
	resumen := dominiopublico.ResumenConvocatoria{
		Version: r.Version, Estado: dominiopublico.EstadoConvocatoria(r.Estado),
		HuellaSHA256:     r.HuellaCompletaSHA256,
		NumeroRequisitos: r.NumeroRequisitos, NumeroDocumentos: r.NumeroDocumentos,
		NumeroAyudas: r.NumeroAyudas,
		DatosPublicos: &dominiopublico.DatosPublicosResumenConvocatoria{
			IdentificadorPublico: r.IdentificadorPublico, Tipo: r.Tipo,
			CatalogoCategorias: dominiopublico.ReferenciaCatalogoCategorias{
				CatalogoID:                     r.CatalogoCategorias.CatalogoID,
				CatalogoVersion:                r.CatalogoCategorias.CatalogoVersion,
				CatalogoHuellaSHA256:           r.CatalogoCategorias.CatalogoHuellaSHA256,
				CatalogoHuellaProyeccionSHA256: r.CatalogoCategorias.CatalogoHuellaProyeccionSHA256,
			},
			Categorias: append([]string(nil), r.Categorias...), Titulo: r.Titulo,
			Resumen: r.Resumen, PublicadaEn: r.PublicadaEn, ActualizadaEn: r.ActualizadaEn,
			Plazos: make([]dominiopublico.PlazoConvocatoria, len(r.Plazos)),
		},
	}
	for indice, plazo := range r.Plazos {
		resumen.DatosPublicos.Plazos[indice] = dominiopublico.PlazoConvocatoria(plazo)
	}
	if err := resumen.ValidarPublicacion(); err != nil {
		return ErrResumenConvocatoriaPublicaInvalido
	}
	return nil
}

// ResumenDesdeConvocatoriaV2 evita que publicador y lector discrepen al
// derivar cantidades o material de listado desde el detalle completo.
func ResumenDesdeConvocatoriaV2(c dominiopublico.Convocatoria) (dominiopublico.ResumenConvocatoria, error) {
	if err := c.ValidarPublicacion(); err != nil || c.DatosPublicos == nil {
		return dominiopublico.ResumenConvocatoria{}, ErrResumenConvocatoriaPublicaInvalido
	}
	datos := c.DatosPublicos
	return dominiopublico.ResumenConvocatoria{
		Version: c.Version, Estado: c.Estado, HuellaSHA256: c.HuellaSHA256,
		NumeroRequisitos: len(datos.Requisitos), NumeroDocumentos: len(datos.Documentos),
		NumeroAyudas: len(datos.Ayuda),
		DatosPublicos: &dominiopublico.DatosPublicosResumenConvocatoria{
			IdentificadorPublico: datos.IdentificadorPublico, Tipo: datos.Tipo,
			CatalogoCategorias: datos.CatalogoCategorias,
			Categorias:         append([]string(nil), datos.Categorias...), Titulo: datos.Titulo,
			Resumen: datos.Resumen, PublicadaEn: datos.PublicadaEn,
			ActualizadaEn: datos.ActualizadaEn,
			Plazos:        append([]dominiopublico.PlazoConvocatoria(nil), datos.Plazos...),
		},
	}, nil
}

func HuellaResumenConvocatoriaV2(r dominiopublico.ResumenConvocatoria) (string, error) {
	if err := r.ValidarPublicacion(); err != nil || r.DatosPublicos == nil {
		return "", ErrResumenConvocatoriaPublicaInvalido
	}
	datos := r.DatosPublicos
	material := ResumenConvocatoriaV2{
		Esquema: EsquemaResumenConvocatoriaV2, IdentificadorPublico: datos.IdentificadorPublico,
		Version: r.Version, Estado: string(r.Estado), Tipo: datos.Tipo,
		CatalogoCategorias: ReferenciaCatalogoCategoriasV2{
			CatalogoID:                     datos.CatalogoCategorias.CatalogoID,
			CatalogoVersion:                datos.CatalogoCategorias.CatalogoVersion,
			CatalogoHuellaSHA256:           datos.CatalogoCategorias.CatalogoHuellaSHA256,
			CatalogoHuellaProyeccionSHA256: datos.CatalogoCategorias.CatalogoHuellaProyeccionSHA256,
		},
		Categorias: append([]string(nil), datos.Categorias...), Titulo: datos.Titulo,
		Resumen: datos.Resumen, PublicadaEn: datos.PublicadaEn,
		ActualizadaEn: datos.ActualizadaEn, Plazos: make([]PlazoConvocatoriaV1, len(datos.Plazos)),
		NumeroRequisitos: r.NumeroRequisitos, NumeroDocumentos: r.NumeroDocumentos,
		NumeroAyudas: r.NumeroAyudas, HuellaCompletaSHA256: r.HuellaSHA256,
	}
	for indice, plazo := range datos.Plazos {
		material.Plazos[indice] = PlazoConvocatoriaV1(plazo)
	}
	return material.HuellaSHA256()
}
