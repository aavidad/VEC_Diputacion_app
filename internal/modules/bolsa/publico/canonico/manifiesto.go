package canonico

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"time"
)

const EsquemaManifiestoPublicoV1 = "vec.bolsa.manifiesto-publico.canonico.v1"

var (
	ErrManifiestoPublicoInvalido  = errors.New("bolsa publica: manifiesto canonico invalido")
	patronReferenciaCatalogo      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$`)
	patronIdentificadorManifiesto = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,79}$`)
	patronRevisionManifiesto      = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,79}$`)
	patronHuellaManifiesto        = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type FuenteManifiestoPublicoV1 struct {
	Revision      string    `json:"revision"`
	ActualizadaEn time.Time `json:"actualizada_en"`
}

type EntradaCatalogoManifiestoV1 struct {
	Clave       string `json:"clave"`
	Etiqueta    string `json:"etiqueta"`
	Descripcion string `json:"descripcion"`
	Semantica   string `json:"semantica"`
	Orden       int    `json:"orden"`
}

type CatalogoManifiestoV1 struct {
	Referencia string                        `json:"referencia"`
	Version    int                           `json:"version"`
	Entradas   []EntradaCatalogoManifiestoV1 `json:"entradas"`
}

type CategoriasManifiestoPublicoV1 struct {
	HuellaGobernadaSHA256  string               `json:"huella_gobernada_sha256"`
	HuellaProyeccionSHA256 string               `json:"huella_proyeccion_sha256"`
	Revision               string               `json:"revision"`
	ActualizadaEn          time.Time            `json:"actualizada_en"`
	Catalogo               CatalogoCategoriasV1 `json:"catalogo"`
}

type ConvocatoriaManifiestoPublicoV1 struct {
	IdentificadorPublico string `json:"identificador_publico"`
	HuellaCompletaSHA256 string `json:"huella_completa_sha256"`
	HuellaResumenSHA256  string `json:"huella_resumen_sha256"`
}

// ManifiestoPublicoV1 fija toda la proyeccion visible, no solo el numero de
// filas. Un cambio de fuente, catalogo, categoria, identificador, detalle o
// resumen produce otra ancla externa.
type ManifiestoPublicoV1 struct {
	Esquema       string                            `json:"esquema"`
	Fuente        FuenteManifiestoPublicoV1         `json:"fuente"`
	Catalogos     []CatalogoManifiestoV1            `json:"catalogos"`
	Categorias    CategoriasManifiestoPublicoV1     `json:"categorias"`
	Convocatorias []ConvocatoriaManifiestoPublicoV1 `json:"convocatorias"`
}

func (m ManifiestoPublicoV1) HuellaSHA256() (string, error) {
	canonico, err := m.canonico()
	if err != nil {
		return "", err
	}
	contenido, err := json.Marshal(canonico)
	if err != nil {
		return "", ErrManifiestoPublicoInvalido
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func (m ManifiestoPublicoV1) Validar() error {
	_, err := m.canonico()
	return err
}

func (m ManifiestoPublicoV1) canonico() (ManifiestoPublicoV1, error) {
	if m.Esquema != EsquemaManifiestoPublicoV1 ||
		!patronRevisionManifiesto.MatchString(m.Fuente.Revision) ||
		!instanteCanonico(m.Fuente.ActualizadaEn) || len(m.Catalogos) == 0 ||
		len(m.Catalogos) > 1_024 || len(m.Convocatorias) > 12_000 ||
		!patronHuellaManifiesto.MatchString(m.Categorias.HuellaGobernadaSHA256) ||
		!patronHuellaManifiesto.MatchString(m.Categorias.HuellaProyeccionSHA256) ||
		!patronRevisionManifiesto.MatchString(m.Categorias.Revision) ||
		!instanteCanonico(m.Categorias.ActualizadaEn) ||
		m.Categorias.Catalogo.Validar() != nil {
		return ManifiestoPublicoV1{}, ErrManifiestoPublicoInvalido
	}
	huellaCategorias, err := m.Categorias.Catalogo.HuellaSHA256()
	if err != nil || huellaCategorias != m.Categorias.HuellaProyeccionSHA256 {
		return ManifiestoPublicoV1{}, ErrManifiestoPublicoInvalido
	}
	resultado := m
	resultado.Fuente.ActualizadaEn = m.Fuente.ActualizadaEn.UTC().Truncate(time.Microsecond)
	resultado.Catalogos = clonarCatalogosManifiesto(m.Catalogos)
	resultado.Categorias.Catalogo.Categorias = clonarCategorias(m.Categorias.Catalogo.Categorias)
	resultado.Categorias.ActualizadaEn = m.Categorias.ActualizadaEn.UTC().Truncate(time.Microsecond)
	sort.Slice(resultado.Categorias.Catalogo.Categorias, func(i, j int) bool {
		if resultado.Categorias.Catalogo.Categorias[i].Orden != resultado.Categorias.Catalogo.Categorias[j].Orden {
			return resultado.Categorias.Catalogo.Categorias[i].Orden < resultado.Categorias.Catalogo.Categorias[j].Orden
		}
		return resultado.Categorias.Catalogo.Categorias[i].Clave < resultado.Categorias.Catalogo.Categorias[j].Clave
	})
	resultado.Convocatorias = append([]ConvocatoriaManifiestoPublicoV1(nil), m.Convocatorias...)

	referencias := make(map[string]struct{}, len(resultado.Catalogos))
	totalEntradas := 0
	for indice := range resultado.Catalogos {
		catalogo := &resultado.Catalogos[indice]
		if !patronReferenciaCatalogo.MatchString(catalogo.Referencia) || catalogo.Version < 1 ||
			len(catalogo.Entradas) == 0 || len(catalogo.Entradas) > 256 {
			return ManifiestoPublicoV1{}, ErrManifiestoPublicoInvalido
		}
		if _, duplicada := referencias[catalogo.Referencia]; duplicada {
			return ManifiestoPublicoV1{}, ErrManifiestoPublicoInvalido
		}
		referencias[catalogo.Referencia] = struct{}{}
		claves, ordenes := make(map[string]struct{}), make(map[int]struct{})
		for _, entrada := range catalogo.Entradas {
			if !patronReferenciaCatalogo.MatchString(entrada.Clave) || entrada.Orden < 1 ||
				!textoCanonico(entrada.Etiqueta, 120, false) ||
				!textoCanonico(entrada.Descripcion, 600, true) ||
				!patronReferenciaCatalogo.MatchString(entrada.Semantica) {
				return ManifiestoPublicoV1{}, ErrManifiestoPublicoInvalido
			}
			if _, duplicada := claves[entrada.Clave]; duplicada {
				return ManifiestoPublicoV1{}, ErrManifiestoPublicoInvalido
			}
			if _, duplicado := ordenes[entrada.Orden]; duplicado {
				return ManifiestoPublicoV1{}, ErrManifiestoPublicoInvalido
			}
			claves[entrada.Clave], ordenes[entrada.Orden] = struct{}{}, struct{}{}
		}
		totalEntradas += len(catalogo.Entradas)
		sort.Slice(catalogo.Entradas, func(i, j int) bool {
			if catalogo.Entradas[i].Orden != catalogo.Entradas[j].Orden {
				return catalogo.Entradas[i].Orden < catalogo.Entradas[j].Orden
			}
			return catalogo.Entradas[i].Clave < catalogo.Entradas[j].Clave
		})
	}
	if totalEntradas > 1_024 {
		return ManifiestoPublicoV1{}, ErrManifiestoPublicoInvalido
	}
	sort.Slice(resultado.Catalogos, func(i, j int) bool {
		return resultado.Catalogos[i].Referencia < resultado.Catalogos[j].Referencia
	})

	identificadores := make(map[string]struct{}, len(resultado.Convocatorias))
	for _, convocatoria := range resultado.Convocatorias {
		if !patronIdentificadorManifiesto.MatchString(convocatoria.IdentificadorPublico) ||
			!patronHuellaManifiesto.MatchString(convocatoria.HuellaCompletaSHA256) ||
			!patronHuellaManifiesto.MatchString(convocatoria.HuellaResumenSHA256) {
			return ManifiestoPublicoV1{}, ErrManifiestoPublicoInvalido
		}
		if _, duplicada := identificadores[convocatoria.IdentificadorPublico]; duplicada {
			return ManifiestoPublicoV1{}, ErrManifiestoPublicoInvalido
		}
		identificadores[convocatoria.IdentificadorPublico] = struct{}{}
	}
	sort.Slice(resultado.Convocatorias, func(i, j int) bool {
		return resultado.Convocatorias[i].IdentificadorPublico < resultado.Convocatorias[j].IdentificadorPublico
	})
	return resultado, nil
}

func clonarCatalogosManifiesto(origen []CatalogoManifiestoV1) []CatalogoManifiestoV1 {
	resultado := make([]CatalogoManifiestoV1, len(origen))
	for indice, catalogo := range origen {
		resultado[indice] = catalogo
		resultado[indice].Entradas = append([]EntradaCatalogoManifiestoV1(nil), catalogo.Entradas...)
	}
	return resultado
}
