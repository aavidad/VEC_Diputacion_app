// Package canonico define representaciones públicas deterministas que pueden
// producir tanto el publicador interno como el proceso anónimo sin compartir
// datos de gobierno, actores ni expedientes.
package canonico

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const EsquemaCatalogoCategoriasV1 = "vec.bolsa.catalogo-categorias-publico.v1"

var (
	ErrCatalogoCategoriasInvalido = errors.New("bolsa publica: catalogo de categorias invalido")
	patronIDCatalogo              = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,127}$`)
	patronClaveCategoria          = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	patronClaveSemantica          = regexp.MustCompile(`^[a-z][a-z0-9_]{0,79}$`)
)

// CategoriaCatalogoV1 contiene exclusivamente los atributos que la audiencia
// pública puede observar. Las fechas completas, incluidas las no vigentes en
// este momento, forman parte de la huella para impedir reinterpretaciones.
type CategoriaCatalogoV1 struct {
	Clave        string     `json:"clave"`
	Etiqueta     string     `json:"etiqueta"`
	Descripcion  string     `json:"descripcion"`
	Semantica    string     `json:"semantica"`
	Orden        int        `json:"orden"`
	Area         string     `json:"area"`
	AreaEtiqueta string     `json:"area_etiqueta"`
	Suscribible  bool       `json:"suscribible"`
	VigenteDesde time.Time  `json:"vigente_desde"`
	VigenteHasta *time.Time `json:"vigente_hasta"`
}

// CatalogoCategoriasV1 es la preimagen exacta de la huella que se fija en la
// configuración del proceso público y en cada convocatoria publicada.
type CatalogoCategoriasV1 struct {
	Esquema    string                `json:"esquema"`
	CatalogoID string                `json:"catalogo_id"`
	Version    int                   `json:"version"`
	Categorias []CategoriaCatalogoV1 `json:"categorias"`
}

func NuevoCatalogoCategoriasV1(
	id string,
	version int,
	categorias []CategoriaCatalogoV1,
) (CatalogoCategoriasV1, error) {
	catalogo := CatalogoCategoriasV1{
		Esquema: EsquemaCatalogoCategoriasV1, CatalogoID: strings.TrimSpace(id),
		Version: version, Categorias: clonarCategorias(categorias),
	}
	if err := catalogo.Validar(); err != nil {
		return CatalogoCategoriasV1{}, err
	}
	return catalogo, nil
}

func (c CatalogoCategoriasV1) Validar() error {
	if c.Esquema != EsquemaCatalogoCategoriasV1 ||
		!patronIDCatalogo.MatchString(c.CatalogoID) || c.Version < 1 ||
		len(c.Categorias) == 0 || len(c.Categorias) > 1_024 {
		return ErrCatalogoCategoriasInvalido
	}
	claves := make(map[string]struct{}, len(c.Categorias))
	ordenes := make(map[int]struct{}, len(c.Categorias))
	for _, categoria := range c.Categorias {
		if !patronClaveCategoria.MatchString(categoria.Clave) ||
			!textoCanonico(categoria.Etiqueta, 120, false) ||
			!textoCanonico(categoria.Descripcion, 600, true) ||
			!patronClaveSemantica.MatchString(categoria.Semantica) || categoria.Orden < 1 ||
			!patronClaveSemantica.MatchString(categoria.Area) ||
			!textoCanonico(categoria.AreaEtiqueta, 120, false) ||
			!instanteCanonico(categoria.VigenteDesde) ||
			(categoria.VigenteHasta != nil && (!instanteCanonico(*categoria.VigenteHasta) ||
				!categoria.VigenteHasta.After(categoria.VigenteDesde))) {
			return ErrCatalogoCategoriasInvalido
		}
		if _, existe := claves[categoria.Clave]; existe {
			return ErrCatalogoCategoriasInvalido
		}
		if _, existe := ordenes[categoria.Orden]; existe {
			return ErrCatalogoCategoriasInvalido
		}
		claves[categoria.Clave] = struct{}{}
		ordenes[categoria.Orden] = struct{}{}
	}
	return nil
}

func (c CatalogoCategoriasV1) HuellaSHA256() (string, error) {
	if err := c.Validar(); err != nil {
		return "", err
	}
	canonico := c
	canonico.Categorias = clonarCategorias(c.Categorias)
	sort.Slice(canonico.Categorias, func(i, j int) bool {
		if canonico.Categorias[i].Orden != canonico.Categorias[j].Orden {
			return canonico.Categorias[i].Orden < canonico.Categorias[j].Orden
		}
		return canonico.Categorias[i].Clave < canonico.Categorias[j].Clave
	})
	contenido, err := json.Marshal(canonico)
	if err != nil {
		return "", ErrCatalogoCategoriasInvalido
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func clonarCategorias(origen []CategoriaCatalogoV1) []CategoriaCatalogoV1 {
	resultado := make([]CategoriaCatalogoV1, len(origen))
	copy(resultado, origen)
	for indice := range resultado {
		resultado[indice].VigenteDesde = resultado[indice].VigenteDesde.UTC().Truncate(time.Microsecond)
		if resultado[indice].VigenteHasta != nil {
			instante := resultado[indice].VigenteHasta.UTC().Truncate(time.Microsecond)
			resultado[indice].VigenteHasta = &instante
		}
	}
	return resultado
}

func instanteCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC && instante.Nanosecond()%1_000 == 0
}

func textoCanonico(valor string, maximo int, admiteVacio bool) bool {
	if valor != strings.TrimSpace(valor) || !utf8.ValidString(valor) ||
		!norm.NFC.IsNormalString(valor) || utf8.RuneCountInString(valor) > maximo ||
		(!admiteVacio && valor == "") {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || unicode.Is(unicode.Cf, caracter) {
			return false
		}
	}
	return true
}
