package firmasct

import (
	"context"
	"errors"
	"strings"

	personalcatalogos "vec-diputacion-granada/internal/modules/personal/adapters/catalogosvec"
	"vec-diputacion-granada/internal/modules/personal/adapters/rptpublica"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
)

var ErrEtiquetasOriginalNoDisponibles = errors.New("composicion CT: etiquetas publicadas no disponibles")

// UnidadPublicada conserva solamente la proyección pública del catálogo
// organizativo que ya consume el alta CT. Ningún nombre procede de la ruta
// HTTP ni de la cuenta del actor.
type UnidadPublicada struct {
	Clave        string
	CodigoFuente string
	Etiqueta     string
	Tipo         string
}

type ReferenciaEtiqueta struct {
	Referencia string
	Etiqueta   string
}

// NuevoEtiquetadorReferencias es la proyección común que consume el PDF de
// bootstrap y el original R5. No admite duplicados divergentes ni valores
// vacíos; una referencia desconocida conserva resultado vacío.
func NuevoEtiquetadorReferencias(entradas []ReferenciaEtiqueta) (func(string) string, error) {
	if len(entradas) == 0 {
		return nil, ErrEtiquetasOriginalNoDisponibles
	}
	nombres := make(map[string]string, len(entradas))
	for _, e := range entradas {
		if strings.TrimSpace(e.Referencia) == "" || strings.TrimSpace(e.Etiqueta) == "" ||
			e.Referencia != strings.TrimSpace(e.Referencia) || e.Etiqueta != strings.TrimSpace(e.Etiqueta) {
			return nil, ErrEtiquetasOriginalNoDisponibles
		}
		if anterior, existe := nombres[e.Referencia]; existe && anterior != e.Etiqueta {
			return nil, ErrEtiquetasOriginalNoDisponibles
		}
		nombres[e.Referencia] = e.Etiqueta
	}
	return func(ref string) string { return nombres[ref] }, nil
}

// CargarUnidadesPublicadas reutiliza el lector canónico del catálogo de
// estructura organizativa. Lo consume bootstrap para el alta y R5 para el PDF.
func CargarUnidadesPublicadas(ctx context.Context, ruta string) ([]UnidadPublicada, error) {
	if ctx == nil || ctx.Err() != nil || strings.TrimSpace(ruta) == "" || ruta != strings.TrimSpace(ruta) {
		return nil, ErrEtiquetasOriginalNoDisponibles
	}
	fuente, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		return nil, errors.Join(ErrEtiquetasOriginalNoDisponibles, err)
	}
	consulta, err := personalcatalogos.NuevaConsultaEstructuraOrganizativa(fuente, "estructura-organizativa-dipgra", 1)
	if err != nil {
		return nil, errors.Join(ErrEtiquetasOriginalNoDisponibles, err)
	}
	datos, err := consulta.Obtener(ctx)
	if err != nil || ctx.Err() != nil || len(datos.Unidades) == 0 {
		return nil, errors.Join(ErrEtiquetasOriginalNoDisponibles, err, ctx.Err())
	}
	salida := make([]UnidadPublicada, 0, len(datos.Unidades))
	for _, u := range datos.Unidades {
		if strings.TrimSpace(u.Clave) == "" || strings.TrimSpace(u.Etiqueta) == "" ||
			u.Clave != strings.TrimSpace(u.Clave) || u.Etiqueta != strings.TrimSpace(u.Etiqueta) {
			return nil, ErrEtiquetasOriginalNoDisponibles
		}
		salida = append(salida, UnidadPublicada{Clave: u.Clave, CodigoFuente: u.CodigoFuente, Etiqueta: u.Etiqueta, Tipo: u.Tipo})
	}
	return salida, nil
}

// CargarEtiquetasOriginalR5 compone los nombres publicados que usan el alta
// CT y la RPT de Personal. Cada clave se resuelve exactamente; una referencia
// desconocida devuelve vacío y el renderizador R5 deniega el PDF.
func CargarEtiquetasOriginalR5(ctx context.Context, rutaOrganizacion, rutaRPT string) (func(string) string, error) {
	unidades, err := CargarUnidadesPublicadas(ctx, rutaOrganizacion)
	if err != nil {
		return nil, err
	}
	fuenteRPT, err := rptpublica.NuevaFuente(rutaRPT)
	if err != nil {
		return nil, ErrEtiquetasOriginalNoDisponibles
	}
	catalogo, err := fuenteRPT.ObtenerRPTPublica(ctx)
	if err != nil || ctx.Err() != nil {
		return nil, ErrEtiquetasOriginalNoDisponibles
	}
	entradas := make([]ReferenciaEtiqueta, 0, len(unidades)*2+len(catalogo.Categorias))
	for _, u := range unidades {
		entradas = append(entradas, ReferenciaEtiqueta{u.Clave, u.Etiqueta})
		if u.Tipo == "centro" {
			codigo := u.CodigoFuente
			if codigo == "" {
				codigo = u.Clave
			}
			entradas = append(entradas, ReferenciaEtiqueta{"centro:rpt:" + codigo, u.Etiqueta})
		}
	}
	for _, c := range catalogo.Categorias {
		entradas = append(entradas, ReferenciaEtiqueta{"categoria:rpt:" + c.Clave, c.Denominacion})
	}
	return NuevoEtiquetadorReferencias(entradas)
}
