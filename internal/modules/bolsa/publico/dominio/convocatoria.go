// Package dominio contiene el modelo nominal que puede cruzar la frontera anónima.
// No incluye referencias a agregados, expedientes ni actores internos.
package dominio

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var ErrConvocatoriaInvalida = errors.New("bolsa: convocatoria invalida")

const (
	// Una convocatoria puede abarcar ampliamente las 68 categorías actuales,
	// pero no materializar el catálogo profesional completo en cada tarjeta.
	// 128 mantiene el modelo dinámico y acota listado, detalle y manifiesto.
	maximoCategoriasConvocatoria = 128
	maximoPlazosConvocatoria     = 64
	maximoRequisitosConvocatoria = 256
	maximoDocumentosConvocatoria = 256
	maximoAyudasConvocatoria     = 128
)

var (
	patronClaveCatalogoConvocatoria = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$`)
	patronReferenciaConvocatoria    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/#-]{0,159}$`)
	patronIdentificadorPublico      = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,79}$`)
	patronHuellaCatalogoSHA256      = regexp.MustCompile(`^[a-f0-9]{64}$`)
	patronIdentificadorFlujo        = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,127}$`)
)

// EstadoConvocatoria contiene una clave gobernada por el catalogo de estados.
// IsValid comprueba solo la sintaxis tecnica; los valores permitidos, etiquetas
// y semantica se publican como datos versionados por el adaptador de catalogos.
type EstadoConvocatoria string

// Estas constantes mantienen la compatibilidad temporal del prototipo
// candidate con las claves canonicas del catalogo gobernado. No constituyen la
// lista de estados permitidos: cualquier clave valida debe existir en la
// definicion de flujo exacta que gobierna la convocatoria.
const (
	EstadoConvocatoriaBorrador    EstadoConvocatoria = "borrador"
	EstadoConvocatoriaInscripcion EstadoConvocatoria = "inscripcion"
	EstadoConvocatoriaSubsanacion EstadoConvocatoria = "subsanacion"
	EstadoConvocatoriaAlegaciones EstadoConvocatoria = "alegaciones"
	EstadoConvocatoriaDefinitiva  EstadoConvocatoria = "definitiva"
	EstadoConvocatoriaCerrada     EstadoConvocatoria = "cerrada"
)

func (e EstadoConvocatoria) IsValid() bool {
	return patronClaveCatalogoConvocatoria.MatchString(string(e))
}

// Convocatoria es la proyección nominal anónima. La identidad pública vive en
// DatosPublicos y no existe campo alguno para la referencia del agregado
// interno.
type Convocatoria struct {
	Version       string                     `json:"version"`
	Estado        EstadoConvocatoria         `json:"estado"`
	DatosPublicos *DatosPublicosConvocatoria `json:"datos_publicos,omitempty"`
	HuellaSHA256  string                     `json:"-"`
}

// ResumenConvocatoria contiene únicamente el material acotado que necesita
// un listado. Los textos extensos de requisitos, documentos y ayuda no se
// cargan por cada tarjeta; sus cantidades y la huella canónica vienen fijadas
// por el publicador.
type ResumenConvocatoria struct {
	Version          string
	Estado           EstadoConvocatoria
	DatosPublicos    *DatosPublicosResumenConvocatoria
	HuellaSHA256     string
	NumeroRequisitos int
	NumeroDocumentos int
	NumeroAyudas     int
}

type DatosPublicosResumenConvocatoria struct {
	IdentificadorPublico string
	Tipo                 string
	CatalogoCategorias   ReferenciaCatalogoCategorias
	Categorias           []string
	Titulo               string
	Resumen              string
	PublicadaEn          time.Time
	ActualizadaEn        time.Time
	Plazos               []PlazoConvocatoria
}

type DatosPublicosConvocatoria struct {
	IdentificadorPublico string                       `json:"identificador_publico"`
	Tipo                 string                       `json:"tipo"`
	CatalogoCategorias   ReferenciaCatalogoCategorias `json:"catalogo_categorias"`
	Categorias           []string                     `json:"categorias"`
	Titulo               string                       `json:"titulo"`
	Resumen              string                       `json:"resumen"`
	Descripcion          string                       `json:"descripcion"`
	PublicadaEn          time.Time                    `json:"publicada_en"`
	ActualizadaEn        time.Time                    `json:"actualizada_en"`
	Plazos               []PlazoConvocatoria          `json:"plazos"`
	Requisitos           []RequisitoConvocatoria      `json:"requisitos"`
	Documentos           []DocumentoConvocatoria      `json:"documentos"`
	Ayuda                []AyudaConvocatoria          `json:"ayuda"`
}

// ReferenciaCatalogoCategorias inmoviliza la instantanea profesional usada
// al publicar una convocatoria. La huella publica de la convocatoria incluye
// esta referencia, por lo que otra version nunca puede reinterpretarla de
// forma silenciosa.
type ReferenciaCatalogoCategorias struct {
	CatalogoID                     string `json:"catalogo_id"`
	CatalogoVersion                int    `json:"catalogo_version"`
	CatalogoHuellaSHA256           string `json:"catalogo_huella_sha256"`
	CatalogoHuellaProyeccionSHA256 string `json:"catalogo_huella_proyeccion_sha256"`
}

func (r ReferenciaCatalogoCategorias) Valida() bool {
	return patronClaveCatalogoConvocatoria.MatchString(r.CatalogoID) &&
		r.CatalogoVersion >= 1 &&
		patronHuellaCatalogoSHA256.MatchString(r.CatalogoHuellaSHA256) &&
		patronHuellaCatalogoSHA256.MatchString(r.CatalogoHuellaProyeccionSHA256)
}

type PlazoConvocatoria struct {
	Referencia  string    `json:"referencia"`
	Tipo        string    `json:"tipo"`
	Titulo      string    `json:"titulo"`
	Descripcion string    `json:"descripcion"`
	AbreEn      time.Time `json:"abre_en"`
	CierraEn    time.Time `json:"cierra_en"`
}

type RequisitoConvocatoria struct {
	Referencia  string `json:"referencia"`
	Orden       int    `json:"orden"`
	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`
	Obligatorio bool   `json:"obligatorio"`
}

type DocumentoConvocatoria struct {
	Referencia  string    `json:"referencia"`
	Tipo        string    `json:"tipo"`
	Orden       int       `json:"orden"`
	Titulo      string    `json:"titulo"`
	Descripcion string    `json:"descripcion"`
	Formato     string    `json:"formato"`
	URL         string    `json:"url"`
	PublicadoEn time.Time `json:"publicado_en"`
}

type AyudaConvocatoria struct {
	Referencia string `json:"referencia"`
	Categoria  string `json:"categoria"`
	Orden      int    `json:"orden"`
	Pregunta   string `json:"pregunta"`
	Respuesta  string `json:"respuesta"`
}

func (c Convocatoria) ValidarPublicacion() error {
	if !referenciaConvocatoriaValida(c.Version) || !c.Estado.IsValid() ||
		!patronHuellaCatalogoSHA256.MatchString(c.HuellaSHA256) || c.DatosPublicos == nil {
		return ErrConvocatoriaInvalida
	}
	d := c.DatosPublicos
	if !patronIdentificadorPublico.MatchString(d.IdentificadorPublico) ||
		!claveCatalogoConvocatoriaValida(d.Tipo) ||
		!d.CatalogoCategorias.Valida() ||
		!textoConvocatoriaValido(d.Titulo, 180, false) ||
		!textoConvocatoriaValido(d.Resumen, 500, false) ||
		!textoConvocatoriaValido(d.Descripcion, 12000, true) ||
		d.PublicadaEn.IsZero() || d.ActualizadaEn.IsZero() || d.ActualizadaEn.Before(d.PublicadaEn) ||
		len(d.Categorias) == 0 || len(d.Categorias) > maximoCategoriasConvocatoria ||
		len(d.Plazos) > maximoPlazosConvocatoria ||
		len(d.Requisitos) > maximoRequisitosConvocatoria || len(d.Documentos) > maximoDocumentosConvocatoria ||
		len(d.Ayuda) > maximoAyudasConvocatoria {
		return ErrConvocatoriaInvalida
	}
	if !instanteUTCCanonico(d.PublicadaEn) || !instanteUTCCanonico(d.ActualizadaEn) {
		return ErrConvocatoriaInvalida
	}
	// Un histórico cerrado puede no conservar un plazo estructurado fiable. En
	// ese caso se publica la fecha oficial de las bases y no se inventa un
	// intervalo. Cualquier estado operativo sigue exigiendo al menos un plazo.
	if (len(d.Plazos) == 0 && c.Estado != EstadoConvocatoriaCerrada) ||
		!clavesCatalogoUnicas(d.Categorias) || !plazosValidos(d.Plazos) || !requisitosValidos(d.Requisitos) ||
		!documentosValidos(d.Documentos) || !ayudasValidas(d.Ayuda) {
		return ErrConvocatoriaInvalida
	}
	return nil
}

func (c ResumenConvocatoria) ValidarPublicacion() error {
	if !referenciaConvocatoriaValida(c.Version) || !c.Estado.IsValid() ||
		!patronHuellaCatalogoSHA256.MatchString(c.HuellaSHA256) || c.DatosPublicos == nil ||
		c.NumeroRequisitos < 0 || c.NumeroRequisitos > maximoRequisitosConvocatoria ||
		c.NumeroDocumentos < 0 || c.NumeroDocumentos > maximoDocumentosConvocatoria ||
		c.NumeroAyudas < 0 || c.NumeroAyudas > maximoAyudasConvocatoria {
		return ErrConvocatoriaInvalida
	}
	d := c.DatosPublicos
	if !patronIdentificadorPublico.MatchString(d.IdentificadorPublico) ||
		!claveCatalogoConvocatoriaValida(d.Tipo) || !d.CatalogoCategorias.Valida() ||
		!textoConvocatoriaValido(d.Titulo, 180, false) ||
		!textoConvocatoriaValido(d.Resumen, 500, false) ||
		!instanteUTCCanonico(d.PublicadaEn) || !instanteUTCCanonico(d.ActualizadaEn) ||
		d.ActualizadaEn.Before(d.PublicadaEn) || len(d.Categorias) == 0 ||
		len(d.Categorias) > maximoCategoriasConvocatoria ||
		len(d.Plazos) > maximoPlazosConvocatoria ||
		(len(d.Plazos) == 0 && c.Estado != EstadoConvocatoriaCerrada) ||
		!clavesCatalogoUnicas(d.Categorias) || !plazosValidos(d.Plazos) {
		return ErrConvocatoriaInvalida
	}
	return nil
}

// Clonar devuelve una copia profunda para que ningún adaptador pueda modificar
// el agregado compartido después de validarlo.
func (c Convocatoria) Clonar() Convocatoria {
	clon := c
	if c.DatosPublicos == nil {
		return clon
	}
	d := *c.DatosPublicos
	d.Categorias = append([]string(nil), c.DatosPublicos.Categorias...)
	d.Plazos = append([]PlazoConvocatoria(nil), c.DatosPublicos.Plazos...)
	d.Requisitos = append([]RequisitoConvocatoria(nil), c.DatosPublicos.Requisitos...)
	d.Documentos = append([]DocumentoConvocatoria(nil), c.DatosPublicos.Documentos...)
	d.Ayuda = append([]AyudaConvocatoria(nil), c.DatosPublicos.Ayuda...)
	clon.DatosPublicos = &d
	return clon
}

func plazosValidos(plazos []PlazoConvocatoria) bool {
	vistos := make(map[string]struct{}, len(plazos))
	for _, plazo := range plazos {
		if !referenciaUnica(plazo.Referencia, vistos) || !claveCatalogoConvocatoriaValida(plazo.Tipo) ||
			!textoConvocatoriaValido(plazo.Titulo, 180, false) || !textoConvocatoriaValido(plazo.Descripcion, 1000, true) ||
			plazo.AbreEn.IsZero() || plazo.CierraEn.IsZero() || !plazo.AbreEn.Before(plazo.CierraEn) ||
			!instanteUTCCanonico(plazo.AbreEn) || !instanteUTCCanonico(plazo.CierraEn) {
			return false
		}
	}
	return true
}

func requisitosValidos(requisitos []RequisitoConvocatoria) bool {
	vistos := make(map[string]struct{}, len(requisitos))
	ordenes := make(map[int]struct{}, len(requisitos))
	for _, requisito := range requisitos {
		if !referenciaUnica(requisito.Referencia, vistos) || requisito.Orden < 1 ||
			!ordenUnico(requisito.Orden, ordenes) || !textoConvocatoriaValido(requisito.Titulo, 180, false) ||
			!textoConvocatoriaValido(requisito.Descripcion, 3000, true) {
			return false
		}
	}
	return true
}

func documentosValidos(documentos []DocumentoConvocatoria) bool {
	vistos := make(map[string]struct{}, len(documentos))
	ordenes := make(map[int]struct{}, len(documentos))
	for _, documento := range documentos {
		if !referenciaUnica(documento.Referencia, vistos) || documento.Orden < 1 || !ordenUnico(documento.Orden, ordenes) ||
			!claveCatalogoConvocatoriaValida(documento.Tipo) || !claveCatalogoConvocatoriaValida(documento.Formato) ||
			!textoConvocatoriaValido(documento.Titulo, 180, false) || !textoConvocatoriaValido(documento.Descripcion, 1000, true) ||
			documento.PublicadoEn.IsZero() || !instanteUTCCanonico(documento.PublicadoEn) ||
			!urlDocumentoPublicoValida(documento.URL) {
			return false
		}
	}
	return true
}

func ayudasValidas(ayudas []AyudaConvocatoria) bool {
	vistos := make(map[string]struct{}, len(ayudas))
	ordenes := make(map[int]struct{}, len(ayudas))
	for _, ayuda := range ayudas {
		if !referenciaUnica(ayuda.Referencia, vistos) || ayuda.Orden < 1 || !ordenUnico(ayuda.Orden, ordenes) ||
			!claveCatalogoConvocatoriaValida(ayuda.Categoria) || !textoConvocatoriaValido(ayuda.Pregunta, 300, false) ||
			!textoConvocatoriaValido(ayuda.Respuesta, 5000, true) {
			return false
		}
	}
	return true
}

func clavesCatalogoUnicas(claves []string) bool {
	vistas := make(map[string]struct{}, len(claves))
	for _, clave := range claves {
		if !claveCatalogoConvocatoriaValida(clave) {
			return false
		}
		if _, existe := vistas[clave]; existe {
			return false
		}
		vistas[clave] = struct{}{}
	}
	return true
}

func referenciaUnica(referencia string, vistas map[string]struct{}) bool {
	if !referenciaConvocatoriaValida(referencia) {
		return false
	}
	if _, existe := vistas[referencia]; existe {
		return false
	}
	vistas[referencia] = struct{}{}
	return true
}

func ordenUnico(orden int, vistos map[int]struct{}) bool {
	if _, existe := vistos[orden]; existe {
		return false
	}
	vistos[orden] = struct{}{}
	return true
}

func referenciaConvocatoriaValida(valor string) bool {
	return valor == strings.TrimSpace(valor) && patronReferenciaConvocatoria.MatchString(valor)
}

func claveCatalogoConvocatoriaValida(valor string) bool {
	return valor == strings.TrimSpace(valor) && patronClaveCatalogoConvocatoria.MatchString(valor)
}

func textoConvocatoriaValido(valor string, maximo int, multilinea bool) bool {
	if maximo < 1 || len(valor) > maximo*utf8.UTFMax || !utf8.ValidString(valor) ||
		!norm.NFC.IsNormalString(valor) || valor != strings.TrimSpace(valor) || valor == "" ||
		utf8.RuneCountInString(valor) > maximo {
		return false
	}
	for _, caracter := range valor {
		if unicode.Is(unicode.Cf, caracter) ||
			(unicode.IsControl(caracter) && (!multilinea || (caracter != '\n' && caracter != '\t'))) {
			return false
		}
	}
	return true
}

func instanteUTCCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC && instante.Nanosecond()%1000 == 0
}

func urlDocumentoPublicoValida(valor string) bool {
	if len(valor) > 240 || strings.TrimSpace(valor) != valor ||
		!strings.HasPrefix(valor, "/bolsa/documentos/") || strings.Contains(valor, "\\") {
		return false
	}
	u, err := url.Parse(valor)
	if err != nil || u.IsAbs() || u.Host != "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	for _, segmento := range strings.Split(u.Path, "/") {
		if segmento == ".." || segmento == "." {
			return false
		}
	}
	return u.Path == valor
}
