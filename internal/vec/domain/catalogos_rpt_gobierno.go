package domain

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

var ErrGobiernoCategoriaRPTInvalido = errors.New("vec: gobierno de categoria RPT invalido")

const (
	AccionGobiernoCategoriaRPTPublicar     = "publicar"
	AccionGobiernoCategoriaRPTDeshabilitar = "deshabilitar"
	EstadoGobiernoCategoriaRPTPropuesta    = "propuesta"
	EstadoGobiernoCategoriaRPTAprobada     = "aprobada"
	EstadoGobiernoCategoriaRPTConfirmada   = "confirmada"
)

var identificadorGobiernoRPT = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{2,127}$`)

// PreimagenControlGobiernoCategoriaRPT identifica el control observado, no
// una autorizacion para reemplazarlo. PostgreSQL vuelve a comparar bajo CAS.
type PreimagenControlGobiernoCategoriaRPT struct {
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
	Revision     int64  `json:"revision"`
	Estado       string `json:"estado"`
}

// ContenidoGobiernoCategoriaRPT es la propuesta inmutable. Las huellas del
// objeto JSONB y de sus preimagenes pertenecen a PostgreSQL; Go no reproduce
// jsonb::text. El documento canonico, cuando existe, conserva sus bytes.
type ContenidoGobiernoCategoriaRPT struct {
	Accion                  string                                          `json:"accion"`
	CatalogoID              string                                          `json:"catalogo_id"`
	ModuloID                string                                          `json:"modulo_id"`
	Version                 int                                             `json:"version"`
	DocumentoCanonico       *string                                         `json:"documento_canonico"`
	DocumentoHuellaSHA256   *string                                         `json:"documento_huella_sha256"`
	PreimagenesControl      map[string]PreimagenControlGobiernoCategoriaRPT `json:"preimagenes_control"`
	PreimagenesHuellaSHA256 string                                          `json:"preimagenes_huella_sha256"`
	CategoriaID             *string                                         `json:"categoria_id"`
	RevisionEsperada        *int64                                          `json:"revision_esperada"`
	MotivoRef               string                                          `json:"motivo_ref"`
	FuenteRef               string                                          `json:"fuente_ref"`
}

// PrepararBorradorParaEditor calcula únicamente la huella de los bytes del
// documento, que no depende de JSONB. La huella de preimagenes y la de toda
// la propuesta se completan en PostgreSQL antes de solicitar V3.
func (c ContenidoGobiernoCategoriaRPT) PrepararBorradorParaEditor(editor string) (ContenidoGobiernoCategoriaRPT, error) {
	if c.PreimagenesHuellaSHA256 != "" || c.DocumentoHuellaSHA256 != nil {
		return ContenidoGobiernoCategoriaRPT{}, ErrGobiernoCategoriaRPTInvalido
	}
	if c.Accion == AccionGobiernoCategoriaRPTPublicar || c.Accion == AccionGobiernoCategoriaRPTDeshabilitar {
		if c.DocumentoCanonico == nil {
			return ContenidoGobiernoCategoriaRPT{}, ErrGobiernoCategoriaRPTInvalido
		}
		suma := sha256.Sum256([]byte(*c.DocumentoCanonico))
		huella := hex.EncodeToString(suma[:])
		c.DocumentoHuellaSHA256 = &huella
	}
	validacion := c
	validacion.PreimagenesHuellaSHA256 = strings.Repeat("0", 64)
	if validacion.ValidarParaEditor(editor) != nil {
		return ContenidoGobiernoCategoriaRPT{}, ErrGobiernoCategoriaRPTInvalido
	}
	return c, nil
}

func (c ContenidoGobiernoCategoriaRPT) ValidarParaEditor(editor string) error {
	if !identificadorGobiernoRPT.MatchString(c.CatalogoID) ||
		!identificadorGobiernoRPT.MatchString(c.ModuloID) ||
		c.Version < 1 || c.Version > 1<<31-1 ||
		!huellaGobiernoRPTValida(c.PreimagenesHuellaSHA256) ||
		c.PreimagenesControl == nil || len(c.PreimagenesControl) > maximoEntradasCatalogo ||
		len(c.FuenteRef) < 3 || len(c.FuenteRef) > 320 ||
		len(c.MotivoRef) < 3 || len(c.MotivoRef) > 320 ||
		strings.TrimSpace(c.FuenteRef) != c.FuenteRef ||
		strings.TrimSpace(c.MotivoRef) != c.MotivoRef {
		return ErrGobiernoCategoriaRPTInvalido
	}
	for id, p := range c.PreimagenesControl {
		if !identificadorGobiernoRPT.MatchString(id) || p.Version < 1 ||
			p.Revision < 1 || !huellaGobiernoRPTValida(p.HuellaSHA256) ||
			(p.Estado != "habilitada" && p.Estado != "deshabilitada") {
			return ErrGobiernoCategoriaRPTInvalido
		}
	}
	if c.DocumentoCanonico == nil || c.DocumentoHuellaSHA256 == nil ||
		len(*c.DocumentoCanonico) < 2 || len(*c.DocumentoCanonico) > maximoBytesCatalogo ||
		!huellaGobiernoRPTValida(*c.DocumentoHuellaSHA256) {
		return ErrGobiernoCategoriaRPTInvalido
	}
	suma := sha256.Sum256([]byte(*c.DocumentoCanonico))
	esperada, _ := hex.DecodeString(*c.DocumentoHuellaSHA256)
	if subtle.ConstantTimeCompare(suma[:], esperada) != 1 {
		return ErrGobiernoCategoriaRPTInvalido
	}
	var catalogo CatalogoConfigurable
	if json.Unmarshal([]byte(*c.DocumentoCanonico), &catalogo) != nil ||
		catalogo.Validar() != nil || catalogo.Estado != EstadoCatalogoPublicado ||
		catalogo.ID != c.CatalogoID || catalogo.ModuloID != c.ModuloID ||
		catalogo.Version != c.Version || catalogo.FuenteRef != c.FuenteRef ||
		catalogo.CreadoPor != editor || catalogo.PublicadoPor == editor ||
		(catalogo.UltimaModificacionPor != "" && catalogo.UltimaModificacionPor != editor) {
		return ErrGobiernoCategoriaRPTInvalido
	}
	canonico, err := catalogo.ClonarCanonico()
	if err != nil {
		return ErrGobiernoCategoriaRPTInvalido
	}
	bytesCanonicos, err := json.Marshal(canonico)
	if err != nil || !bytes.Equal(bytesCanonicos, []byte(*c.DocumentoCanonico)) {
		return ErrGobiernoCategoriaRPTInvalido
	}
	entradas := make(map[string]EntradaCatalogoConfigurable, len(catalogo.Entradas))
	for _, entrada := range catalogo.Entradas {
		estado := entrada.Atributos["estado"]
		if estado != "habilitada" && estado != "deshabilitada" {
			return ErrGobiernoCategoriaRPTInvalido
		}
		entradas[entrada.Clave] = entrada
	}
	// El documento preserva todas las categorías controladas. Sus controles
	// se vuelven a comparar en SQL antes del efecto; ningún atributo concede.
	for id, preimagen := range c.PreimagenesControl {
		entrada, existe := entradas[id]
		if !existe || preimagen.Version != c.Version-1 {
			return ErrGobiernoCategoriaRPTInvalido
		}
		if c.CategoriaID == nil || id != *c.CategoriaID {
			if entrada.Atributos["estado"] != preimagen.Estado {
				return ErrGobiernoCategoriaRPTInvalido
			}
		}
	}
	switch c.Accion {
	case AccionGobiernoCategoriaRPTPublicar:
		if c.CategoriaID != nil || c.RevisionEsperada != nil {
			return ErrGobiernoCategoriaRPTInvalido
		}
		for id, entrada := range entradas {
			if _, existe := c.PreimagenesControl[id]; !existe && entrada.Atributos["estado"] != "habilitada" {
				return ErrGobiernoCategoriaRPTInvalido
			}
		}
	case AccionGobiernoCategoriaRPTDeshabilitar:
		if c.CategoriaID == nil || !identificadorGobiernoRPT.MatchString(*c.CategoriaID) ||
			c.RevisionEsperada == nil || *c.RevisionEsperada < 1 ||
			len(c.PreimagenesControl) != len(entradas) {
			return ErrGobiernoCategoriaRPTInvalido
		}
		p, ok := c.PreimagenesControl[*c.CategoriaID]
		entrada, existe := entradas[*c.CategoriaID]
		if !ok || !existe || p.Version != c.Version-1 || p.Revision != *c.RevisionEsperada ||
			p.Estado != "habilitada" || entrada.Atributos["estado"] != "deshabilitada" {
			return ErrGobiernoCategoriaRPTInvalido
		}
	default:
		return ErrGobiernoCategoriaRPTInvalido
	}
	return nil
}

func huellaGobiernoRPTValida(s string) bool {
	if len(s) != sha256.Size*2 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			if c < 'a' || c > 'f' {
				return false
			}
		}
	}
	return true
}
