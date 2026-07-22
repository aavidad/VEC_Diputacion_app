package canonico

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	dominiopublico "vec-diputacion-granada/internal/modules/bolsa/publico/dominio"
)

const EsquemaConvocatoriaV2 = "vec.bolsa.convocatoria-publica.canonica.v2"

var ErrConvocatoriaPublicaInvalida = errors.New("bolsa publica: convocatoria canonica invalida")

type ReferenciaCatalogoCategoriasV2 struct {
	CatalogoID                     string `json:"catalogo_id"`
	CatalogoVersion                int    `json:"catalogo_version"`
	CatalogoHuellaSHA256           string `json:"catalogo_huella_sha256"`
	CatalogoHuellaProyeccionSHA256 string `json:"catalogo_huella_proyeccion_sha256"`
}

type PlazoConvocatoriaV1 struct {
	Referencia  string    `json:"referencia"`
	Tipo        string    `json:"tipo"`
	Titulo      string    `json:"titulo"`
	Descripcion string    `json:"descripcion"`
	AbreEn      time.Time `json:"abre_en"`
	CierraEn    time.Time `json:"cierra_en"`
}

type RequisitoConvocatoriaV1 struct {
	Referencia  string `json:"referencia"`
	Orden       int    `json:"orden"`
	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`
	Obligatorio bool   `json:"obligatorio"`
}

type DocumentoConvocatoriaV1 struct {
	Referencia  string    `json:"referencia"`
	Tipo        string    `json:"tipo"`
	Orden       int       `json:"orden"`
	Titulo      string    `json:"titulo"`
	Descripcion string    `json:"descripcion"`
	Formato     string    `json:"formato"`
	URL         string    `json:"url"`
	PublicadoEn time.Time `json:"publicado_en"`
}

type AyudaConvocatoriaV1 struct {
	Referencia string `json:"referencia"`
	Categoria  string `json:"categoria"`
	Orden      int    `json:"orden"`
	Pregunta   string `json:"pregunta"`
	Respuesta  string `json:"respuesta"`
}

// ConvocatoriaV2 excluye por diseño la referencia del agregado interno. La
// huella de esta estructura sí puede publicarse en una respuesta anónima.
type ConvocatoriaV2 struct {
	Esquema              string                         `json:"esquema"`
	IdentificadorPublico string                         `json:"identificador_publico"`
	Version              string                         `json:"version"`
	Estado               string                         `json:"estado"`
	Tipo                 string                         `json:"tipo"`
	CatalogoCategorias   ReferenciaCatalogoCategoriasV2 `json:"catalogo_categorias"`
	Categorias           []string                       `json:"categorias"`
	Titulo               string                         `json:"titulo"`
	Resumen              string                         `json:"resumen"`
	Descripcion          string                         `json:"descripcion"`
	PublicadaEn          time.Time                      `json:"publicada_en"`
	ActualizadaEn        time.Time                      `json:"actualizada_en"`
	Plazos               []PlazoConvocatoriaV1          `json:"plazos"`
	Requisitos           []RequisitoConvocatoriaV1      `json:"requisitos"`
	Documentos           []DocumentoConvocatoriaV1      `json:"documentos"`
	Ayuda                []AyudaConvocatoriaV1          `json:"ayuda"`
}

func (c ConvocatoriaV2) HuellaSHA256() (string, error) {
	if err := c.Validar(); err != nil {
		return "", ErrConvocatoriaPublicaInvalida
	}
	canonico := c
	canonico.Categorias = append([]string(nil), c.Categorias...)
	canonico.Plazos = append([]PlazoConvocatoriaV1(nil), c.Plazos...)
	canonico.Requisitos = append([]RequisitoConvocatoriaV1(nil), c.Requisitos...)
	canonico.Documentos = append([]DocumentoConvocatoriaV1(nil), c.Documentos...)
	canonico.Ayuda = append([]AyudaConvocatoriaV1(nil), c.Ayuda...)
	sort.Strings(canonico.Categorias)
	sort.Slice(canonico.Plazos, func(i, j int) bool {
		return canonico.Plazos[i].Referencia < canonico.Plazos[j].Referencia
	})
	sort.Slice(canonico.Requisitos, func(i, j int) bool {
		if canonico.Requisitos[i].Orden != canonico.Requisitos[j].Orden {
			return canonico.Requisitos[i].Orden < canonico.Requisitos[j].Orden
		}
		return canonico.Requisitos[i].Referencia < canonico.Requisitos[j].Referencia
	})
	sort.Slice(canonico.Documentos, func(i, j int) bool {
		if canonico.Documentos[i].Orden != canonico.Documentos[j].Orden {
			return canonico.Documentos[i].Orden < canonico.Documentos[j].Orden
		}
		return canonico.Documentos[i].Referencia < canonico.Documentos[j].Referencia
	})
	sort.Slice(canonico.Ayuda, func(i, j int) bool {
		if canonico.Ayuda[i].Orden != canonico.Ayuda[j].Orden {
			return canonico.Ayuda[i].Orden < canonico.Ayuda[j].Orden
		}
		return canonico.Ayuda[i].Referencia < canonico.Ayuda[j].Referencia
	})
	contenido, err := json.Marshal(canonico)
	if err != nil {
		return "", ErrConvocatoriaPublicaInvalida
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

// Validar aplica al material canónico exactamente las invariantes del modelo
// público nominal antes de que el publicador o el lector calculen una huella.
func (c ConvocatoriaV2) Validar() error {
	if c.Esquema != EsquemaConvocatoriaV2 {
		return ErrConvocatoriaPublicaInvalida
	}
	convocatoria := dominiopublico.Convocatoria{
		Version: c.Version,
		Estado:  dominiopublico.EstadoConvocatoria(c.Estado),
		// La huella todavía no existe; este valor sintáctico permite reutilizar
		// el resto de invariantes sin introducirla en su propia preimagen.
		HuellaSHA256: strings.Repeat("0", 64),
		DatosPublicos: &dominiopublico.DatosPublicosConvocatoria{
			IdentificadorPublico: c.IdentificadorPublico,
			Tipo:                 c.Tipo,
			CatalogoCategorias: dominiopublico.ReferenciaCatalogoCategorias{
				CatalogoID: c.CatalogoCategorias.CatalogoID, CatalogoVersion: c.CatalogoCategorias.CatalogoVersion,
				CatalogoHuellaSHA256:           c.CatalogoCategorias.CatalogoHuellaSHA256,
				CatalogoHuellaProyeccionSHA256: c.CatalogoCategorias.CatalogoHuellaProyeccionSHA256,
			},
			Categorias: append([]string(nil), c.Categorias...), Titulo: c.Titulo,
			Resumen: c.Resumen, Descripcion: c.Descripcion,
			PublicadaEn: c.PublicadaEn, ActualizadaEn: c.ActualizadaEn,
			Plazos:     make([]dominiopublico.PlazoConvocatoria, len(c.Plazos)),
			Requisitos: make([]dominiopublico.RequisitoConvocatoria, len(c.Requisitos)),
			Documentos: make([]dominiopublico.DocumentoConvocatoria, len(c.Documentos)),
			Ayuda:      make([]dominiopublico.AyudaConvocatoria, len(c.Ayuda)),
		},
	}
	for indice, plazo := range c.Plazos {
		convocatoria.DatosPublicos.Plazos[indice] = dominiopublico.PlazoConvocatoria(plazo)
	}
	for indice, requisito := range c.Requisitos {
		convocatoria.DatosPublicos.Requisitos[indice] = dominiopublico.RequisitoConvocatoria(requisito)
	}
	for indice, documento := range c.Documentos {
		convocatoria.DatosPublicos.Documentos[indice] = dominiopublico.DocumentoConvocatoria(documento)
	}
	for indice, ayuda := range c.Ayuda {
		convocatoria.DatosPublicos.Ayuda[indice] = dominiopublico.AyudaConvocatoria(ayuda)
	}
	if err := convocatoria.ValidarPublicacion(); err != nil {
		return ErrConvocatoriaPublicaInvalida
	}
	return nil
}

// HuellaConvocatoriaV2 proyecta el modelo público nominal a la preimagen V2.
// Es la única conversión autorizada para el publicador, el adaptador de
// lectura y la aplicación anónima.
func HuellaConvocatoriaV2(c dominiopublico.Convocatoria) (string, error) {
	if err := c.ValidarPublicacion(); err != nil || c.DatosPublicos == nil {
		return "", ErrConvocatoriaPublicaInvalida
	}
	datos := c.DatosPublicos
	material := ConvocatoriaV2{
		Esquema: EsquemaConvocatoriaV2, IdentificadorPublico: datos.IdentificadorPublico,
		Version: c.Version, Estado: string(c.Estado), Tipo: datos.Tipo,
		CatalogoCategorias: ReferenciaCatalogoCategoriasV2{
			CatalogoID:                     datos.CatalogoCategorias.CatalogoID,
			CatalogoVersion:                datos.CatalogoCategorias.CatalogoVersion,
			CatalogoHuellaSHA256:           datos.CatalogoCategorias.CatalogoHuellaSHA256,
			CatalogoHuellaProyeccionSHA256: datos.CatalogoCategorias.CatalogoHuellaProyeccionSHA256,
		},
		Categorias: append([]string(nil), datos.Categorias...),
		Titulo:     datos.Titulo, Resumen: datos.Resumen, Descripcion: datos.Descripcion,
		PublicadaEn: datos.PublicadaEn, ActualizadaEn: datos.ActualizadaEn,
		Plazos:     make([]PlazoConvocatoriaV1, len(datos.Plazos)),
		Requisitos: make([]RequisitoConvocatoriaV1, len(datos.Requisitos)),
		Documentos: make([]DocumentoConvocatoriaV1, len(datos.Documentos)),
		Ayuda:      make([]AyudaConvocatoriaV1, len(datos.Ayuda)),
	}
	for indice, plazo := range datos.Plazos {
		material.Plazos[indice] = PlazoConvocatoriaV1(plazo)
	}
	for indice, requisito := range datos.Requisitos {
		material.Requisitos[indice] = RequisitoConvocatoriaV1(requisito)
	}
	for indice, documento := range datos.Documentos {
		material.Documentos[indice] = DocumentoConvocatoriaV1(documento)
	}
	for indice, ayuda := range datos.Ayuda {
		material.Ayuda[indice] = AyudaConvocatoriaV1(ayuda)
	}
	return material.HuellaSHA256()
}
