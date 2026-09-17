package bootstrap

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
)

// El catálogo público de la RPT (`data/catalogos/rpt/v1.rpt-2026.json`, generado por
// `scripts/generar_catalogo_rpt.py`) aporta las categorías reales de la Diputación
// con sus grupos. Cuando está configurado, sustituye a las categorías sintéticas en
// el catálogo de alta; las sintéticas siguen valiendo para los expedientes que ya las
// usan. No contiene ocupantes ni datos personales.

const esquemaCatalogoRPT = "vec.catalogo.rpt.v1"

var errCatalogoRPTDesarrolloInvalido = errors.New(
	"contratacion temporal: catalogo RPT de desarrollo invalido",
)

type catalogoRPTDesarrollo struct {
	Esquema    string                   `json:"esquema"`
	Categorias []categoriaRPTDesarrollo `json:"categorias"`
}

type categoriaRPTDesarrollo struct {
	Clave        string   `json:"clave"`
	Denominacion string   `json:"denominacion"`
	Grupos       []string `json:"grupos"`
	Dotacion     int      `json:"dotacion"`
}

// cargarCategoriasRPTDesarrollo lee el catálogo público de la RPT y devuelve las
// categorías como opciones del catálogo de alta: referencia `categoria:rpt:<clave>`,
// etiqueta la denominación de la RPT y un grupo por cada grupo admitido. Se ordenan
// por denominación para que el formulario sea legible.
func cargarCategoriasRPTDesarrollo(ruta string) ([]categoriaCatalogosAltaContratacionTemporalDesarrollo, error) {
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		return nil, errCatalogoRPTDesarrolloInvalido
	}
	var catalogo catalogoRPTDesarrollo
	if err := json.Unmarshal(contenido, &catalogo); err != nil || catalogo.Esquema != esquemaCatalogoRPT {
		return nil, errCatalogoRPTDesarrolloInvalido
	}
	vistas := make(map[string]struct{}, len(catalogo.Categorias))
	categorias := make([]categoriaCatalogosAltaContratacionTemporalDesarrollo, 0, len(catalogo.Categorias))
	for _, c := range catalogo.Categorias {
		clave := strings.TrimSpace(c.Clave)
		denominacion := strings.TrimSpace(c.Denominacion)
		if clave == "" || denominacion == "" || len(c.Grupos) == 0 {
			continue
		}
		if _, repetida := vistas[clave]; repetida {
			return nil, errCatalogoRPTDesarrolloInvalido
		}
		vistas[clave] = struct{}{}
		grupos := make([]opcionClaveCatalogosAltaContratacionTemporalDesarrollo, 0, len(c.Grupos))
		for _, g := range c.Grupos {
			g = strings.TrimSpace(g)
			if g == "" {
				continue
			}
			grupos = append(grupos, opcionClaveCatalogosAltaContratacionTemporalDesarrollo{Clave: g, Etiqueta: "Grupo " + g})
		}
		if len(grupos) == 0 {
			continue
		}
		categorias = append(categorias, categoriaCatalogosAltaContratacionTemporalDesarrollo{
			Referencia:      "categoria:rpt:" + clave,
			Etiqueta:        denominacion,
			GruposSubgrupos: grupos,
		})
	}
	if len(categorias) == 0 {
		return nil, errCatalogoRPTDesarrolloInvalido
	}
	sort.SliceStable(categorias, func(i, j int) bool { return categorias[i].Etiqueta < categorias[j].Etiqueta })
	return categorias, nil
}
