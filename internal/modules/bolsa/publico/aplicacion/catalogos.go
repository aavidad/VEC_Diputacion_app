package aplicacion

import (
	"sort"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/publico/dominio"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/publico/puertos"
)

type indiceCatalogos struct {
	porCatalogo             map[string]map[string]puertosbolsa.EntradaCatalogoPublico
	ordenados               map[string][]puertosbolsa.EntradaCatalogoPublico
	referenciaCategorias    dominiobolsa.ReferenciaCatalogoCategorias
	categoriasPorReferencia map[dominiobolsa.ReferenciaCatalogoCategorias]map[string]puertosbolsa.EntradaCatalogoPublico
	referenciasPorVersion   map[int]dominiobolsa.ReferenciaCatalogoCategorias
}

func nuevoIndiceCatalogos(
	catalogos []puertosbolsa.CatalogoPublico,
	categorias puertosbolsa.CatalogoCategoriasPublicas,
	snapshots []puertosbolsa.CatalogoCategoriasPublicas,
) (indiceCatalogos, error) {
	if !patronIDCatalogoCategorias.MatchString(categorias.ID) || categorias.Version < 1 ||
		!patronHuellaSHA256.MatchString(categorias.HuellaGobernadaSHA256) ||
		!patronHuellaSHA256.MatchString(categorias.HuellaProyeccionSHA256) ||
		len(categorias.Categorias) > 1_024 ||
		validarFuenteCategorias(categorias.Fuente) != nil || len(snapshots) == 0 || len(snapshots) > 64 {
		return indiceCatalogos{}, ErrDatosPublicosNoConfiables
	}
	catalogosCompletos := make([]puertosbolsa.CatalogoPublico, 0, len(catalogos)+1)
	catalogosCompletos = append(catalogosCompletos, catalogos...)
	catalogoProfesional := puertosbolsa.CatalogoPublico{
		Referencia: puertosbolsa.CatalogoCategoriasConvocatoria,
		Version:    categorias.Version,
		Entradas:   make([]puertosbolsa.EntradaCatalogoPublico, 0, len(categorias.Categorias)),
	}
	for _, categoria := range categorias.Categorias {
		if categoria.Version != categorias.Version || !patronFiltroCatalogo.MatchString(categoria.Area) ||
			!textoPublicoCanonico(categoria.AreaEtiqueta, 120, false) {
			return indiceCatalogos{}, ErrDatosPublicosNoConfiables
		}
		catalogoProfesional.Entradas = append(catalogoProfesional.Entradas, puertosbolsa.EntradaCatalogoPublico{
			Clave:       categoria.Clave,
			Version:     categoria.Version,
			Etiqueta:    categoria.Etiqueta,
			Descripcion: categoria.Descripcion,
			Semantica:   categoria.Semantica,
			Orden:       categoria.Orden,
			Publicable:  true,
		})
	}
	catalogosCompletos = append(catalogosCompletos, catalogoProfesional)
	indice := indiceCatalogos{
		porCatalogo:             map[string]map[string]puertosbolsa.EntradaCatalogoPublico{},
		ordenados:               map[string][]puertosbolsa.EntradaCatalogoPublico{},
		categoriasPorReferencia: map[dominiobolsa.ReferenciaCatalogoCategorias]map[string]puertosbolsa.EntradaCatalogoPublico{},
		referenciasPorVersion:   map[int]dominiobolsa.ReferenciaCatalogoCategorias{},
		referenciaCategorias: dominiobolsa.ReferenciaCatalogoCategorias{
			CatalogoID: categorias.ID, CatalogoVersion: categorias.Version,
			CatalogoHuellaSHA256:           categorias.HuellaGobernadaSHA256,
			CatalogoHuellaProyeccionSHA256: categorias.HuellaProyeccionSHA256,
		},
	}
	actualEncontrado := false
	totalCategorias := 0
	for _, snapshot := range snapshots {
		referencia := dominiobolsa.ReferenciaCatalogoCategorias{
			CatalogoID: snapshot.ID, CatalogoVersion: snapshot.Version,
			CatalogoHuellaSHA256:           snapshot.HuellaGobernadaSHA256,
			CatalogoHuellaProyeccionSHA256: snapshot.HuellaProyeccionSHA256,
		}
		if !referencia.Valida() ||
			!patronIDCatalogoCategorias.MatchString(snapshot.ID) ||
			snapshot.ID != categorias.ID ||
			validarFuenteCategorias(snapshot.Fuente) != nil ||
			!fuentesCategoriasCoinciden(snapshot.Fuente, categorias.Fuente) ||
			len(snapshot.Categorias) == 0 || len(snapshot.Categorias) > 1_024 ||
			indice.categoriasPorReferencia[referencia] != nil ||
			indice.referenciasPorVersion[snapshot.Version].Valida() {
			return indiceCatalogos{}, ErrDatosPublicosNoConfiables
		}
		entradas := make(map[string]puertosbolsa.EntradaCatalogoPublico, len(snapshot.Categorias))
		for _, categoria := range snapshot.Categorias {
			if categoria.Version != snapshot.Version || !patronFiltroCatalogo.MatchString(categoria.Clave) ||
				!patronFiltroCatalogo.MatchString(categoria.Area) ||
				!textoPublicoCanonico(categoria.Etiqueta, 120, false) ||
				(categoria.Descripcion != "" && !textoPublicoCanonico(categoria.Descripcion, 600, true)) ||
				!patronFiltroCatalogo.MatchString(categoria.Semantica) || categoria.Orden < 1 ||
				!textoPublicoCanonico(categoria.AreaEtiqueta, 120, false) ||
				entradas[categoria.Clave].Clave != "" {
				return indiceCatalogos{}, ErrDatosPublicosNoConfiables
			}
			entradas[categoria.Clave] = puertosbolsa.EntradaCatalogoPublico{
				Clave: categoria.Clave, Version: categoria.Version, Etiqueta: categoria.Etiqueta,
				Descripcion: categoria.Descripcion, Semantica: categoria.Semantica,
				Orden: categoria.Orden, Publicable: true,
			}
		}
		indice.categoriasPorReferencia[referencia] = entradas
		indice.referenciasPorVersion[snapshot.Version] = referencia
		totalCategorias += len(snapshot.Categorias)
		if referencia == indice.referenciaCategorias {
			if actualEncontrado || !categoriasVigentesIncluidasExactamente(
				categorias.Categorias, snapshot.Categorias,
			) {
				return indiceCatalogos{}, ErrDatosPublicosNoConfiables
			}
			actualEncontrado = true
		}
	}
	if !actualEncontrado || totalCategorias > 4_096 {
		return indiceCatalogos{}, ErrDatosPublicosNoConfiables
	}
	for _, catalogo := range catalogosCompletos {
		if !patronFiltroCatalogo.MatchString(catalogo.Referencia) || catalogo.Version < 1 ||
			(len(catalogo.Entradas) == 0 && catalogo.Referencia != puertosbolsa.CatalogoCategoriasConvocatoria) ||
			indice.porCatalogo[catalogo.Referencia] != nil {
			return indiceCatalogos{}, ErrDatosPublicosNoConfiables
		}
		indice.porCatalogo[catalogo.Referencia] = map[string]puertosbolsa.EntradaCatalogoPublico{}
		for _, entrada := range catalogo.Entradas {
			if !patronFiltroCatalogo.MatchString(entrada.Clave) || entrada.Version != catalogo.Version ||
				!textoPublicoCanonico(entrada.Etiqueta, 120, false) ||
				(entrada.Descripcion != "" && !textoPublicoCanonico(entrada.Descripcion, 600, true)) ||
				!patronFiltroCatalogo.MatchString(entrada.Semantica) || entrada.Orden < 1 || indice.porCatalogo[catalogo.Referencia][entrada.Clave].Clave != "" {
				return indiceCatalogos{}, ErrDatosPublicosNoConfiables
			}
			indice.porCatalogo[catalogo.Referencia][entrada.Clave] = entrada
			indice.ordenados[catalogo.Referencia] = append(indice.ordenados[catalogo.Referencia], entrada)
		}
		sort.Slice(indice.ordenados[catalogo.Referencia], func(i, j int) bool {
			a, b := indice.ordenados[catalogo.Referencia][i], indice.ordenados[catalogo.Referencia][j]
			if a.Orden == b.Orden {
				return a.Clave < b.Clave
			}
			return a.Orden < b.Orden
		})
	}
	for _, requerido := range []string{puertosbolsa.CatalogoTiposConvocatoria, puertosbolsa.CatalogoEstadosConvocatoria, puertosbolsa.CatalogoCategoriasConvocatoria, puertosbolsa.CatalogoTiposPlazo, puertosbolsa.CatalogoTiposDocumento, puertosbolsa.CatalogoCategoriasAyuda} {
		if indice.porCatalogo[requerido] == nil {
			return indiceCatalogos{}, ErrDatosPublicosNoConfiables
		}
	}
	return indice, nil
}

func categoriasVigentesIncluidasExactamente(
	vigentes []puertosbolsa.CategoriaPublica,
	snapshot []puertosbolsa.CategoriaPublica,
) bool {
	porClave := make(map[string]puertosbolsa.CategoriaPublica, len(snapshot))
	for _, categoria := range snapshot {
		if categoria.Clave == "" {
			return false
		}
		if _, duplicada := porClave[categoria.Clave]; duplicada {
			return false
		}
		porClave[categoria.Clave] = categoria
	}
	vistas := make(map[string]struct{}, len(vigentes))
	for _, categoria := range vigentes {
		if _, duplicada := vistas[categoria.Clave]; duplicada || porClave[categoria.Clave] != categoria {
			return false
		}
		vistas[categoria.Clave] = struct{}{}
	}
	return true
}

func (i indiceCatalogos) resolver(catalogo, clave string) (ValorCatalogoPublico, error) {
	entrada, existe := i.porCatalogo[catalogo][clave]
	if !existe || !entrada.Publicable {
		return ValorCatalogoPublico{}, ErrDatosPublicosNoConfiables
	}
	return ValorCatalogoPublico{Clave: entrada.Clave, Version: entrada.Version, Etiqueta: entrada.Etiqueta, Descripcion: entrada.Descripcion, Semantica: entrada.Semantica}, nil
}

func (i indiceCatalogos) resolverCategoria(
	referencia dominiobolsa.ReferenciaCatalogoCategorias,
	clave string,
) (ValorCatalogoPublico, error) {
	entrada, existe := i.categoriasPorReferencia[referencia][clave]
	if !existe || !entrada.Publicable {
		return ValorCatalogoPublico{}, ErrDatosPublicosNoConfiables
	}
	return ValorCatalogoPublico{
		Clave: entrada.Clave, Version: entrada.Version, Etiqueta: entrada.Etiqueta,
		Descripcion: entrada.Descripcion, Semantica: entrada.Semantica,
	}, nil
}

func (i indiceCatalogos) facetas(conteos map[string]puertosbolsa.ConteoCategoriaConvocatorias) FacetasConvocatorias {
	categorias := make([]FacetaCategoriaPublica, 0, len(conteos))
	for _, entrada := range i.ordenados[puertosbolsa.CatalogoCategoriasConvocatoria] {
		conteo := conteos[entrada.Clave]
		if !entrada.Publicable || conteo.NumeroConvocatorias < 1 {
			continue
		}
		categorias = append(categorias, FacetaCategoriaPublica{
			Clave:            entrada.Clave,
			Version:          entrada.Version,
			Etiqueta:         entrada.Etiqueta,
			Descripcion:      entrada.Descripcion,
			Semantica:        entrada.Semantica,
			NumeroResultados: conteo.NumeroConvocatorias,
		})
	}
	return FacetasConvocatorias{
		Tipos:      i.valoresPublicables(puertosbolsa.CatalogoTiposConvocatoria),
		Categorias: categorias,
		Estados:    i.valoresPublicables(puertosbolsa.CatalogoEstadosConvocatoria),
	}
}

func (i indiceCatalogos) valoresPublicables(catalogo string) []ValorCatalogoPublico {
	valores := make([]ValorCatalogoPublico, 0, len(i.ordenados[catalogo]))
	for _, entrada := range i.ordenados[catalogo] {
		if entrada.Publicable {
			valores = append(valores, ValorCatalogoPublico{Clave: entrada.Clave, Version: entrada.Version, Etiqueta: entrada.Etiqueta, Descripcion: entrada.Descripcion, Semantica: entrada.Semantica})
		}
	}
	return valores
}

func (i indiceCatalogos) diccionarioCategorias(
	referencias []ReferenciaCategoriaPublica,
) ([]CategoriaDiccionarioPublico, error) {
	if len(referencias) > MaximoReferenciasCategoriasPagina {
		return nil, ErrDatosPublicosNoConfiables
	}
	resultado := make([]CategoriaDiccionarioPublico, 0, len(referencias))
	vistas := make(map[ReferenciaCategoriaPublica]struct{}, len(referencias))
	for _, referencia := range referencias {
		if referencia.Clave == "" || referencia.Version < 1 {
			return nil, ErrDatosPublicosNoConfiables
		}
		if _, duplicada := vistas[referencia]; duplicada {
			continue
		}
		catalogo, existe := i.referenciasPorVersion[referencia.Version]
		if !existe {
			return nil, ErrDatosPublicosNoConfiables
		}
		valor, err := i.resolverCategoria(catalogo, referencia.Clave)
		if err != nil || valor.Version != referencia.Version {
			return nil, ErrDatosPublicosNoConfiables
		}
		vistas[referencia] = struct{}{}
		resultado = append(resultado, CategoriaDiccionarioPublico{
			CatalogoCategorias: ReferenciaCatalogoCategoriasConvocatoriaPublica{
				CatalogoID: catalogo.CatalogoID, Version: catalogo.CatalogoVersion,
				HuellaSHA256:           catalogo.CatalogoHuellaSHA256,
				HuellaProyeccionSHA256: catalogo.CatalogoHuellaProyeccionSHA256,
			},
			Clave: valor.Clave, Version: valor.Version, Etiqueta: valor.Etiqueta,
			Descripcion: valor.Descripcion, Semantica: valor.Semantica,
		})
	}
	sort.Slice(resultado, func(a, b int) bool {
		if resultado[a].Version != resultado[b].Version {
			return resultado[a].Version < resultado[b].Version
		}
		return resultado[a].Clave < resultado[b].Clave
	})
	return resultado, nil
}

func validarCoberturaCategoriasResumenes(
	resumenes []ResumenConvocatoriaPublica,
	diccionarioCategorias []CategoriaDiccionarioPublico,
) error {
	type referenciaResuelta struct {
		catalogo  ReferenciaCatalogoCategoriasConvocatoriaPublica
		categoria ReferenciaCategoriaPublica
	}
	diccionario := make(map[referenciaResuelta]struct{}, len(diccionarioCategorias))
	for _, categoria := range diccionarioCategorias {
		referencia := referenciaResuelta{
			catalogo:  categoria.CatalogoCategorias,
			categoria: ReferenciaCategoriaPublica{Clave: categoria.Clave, Version: categoria.Version},
		}
		if referencia.categoria.Clave == "" || referencia.categoria.Version < 1 ||
			referencia.catalogo.CatalogoID == "" || referencia.catalogo.Version < 1 ||
			!patronHuellaSHA256.MatchString(referencia.catalogo.HuellaSHA256) ||
			!patronHuellaSHA256.MatchString(referencia.catalogo.HuellaProyeccionSHA256) {
			return ErrDatosPublicosNoConfiables
		}
		if _, duplicada := diccionario[referencia]; duplicada {
			return ErrDatosPublicosNoConfiables
		}
		diccionario[referencia] = struct{}{}
	}
	for _, resumen := range resumenes {
		vistas := make(map[ReferenciaCategoriaPublica]struct{}, len(resumen.Categorias))
		for _, referencia := range resumen.Categorias {
			if _, duplicada := vistas[referencia]; duplicada {
				return ErrDatosPublicosNoConfiables
			}
			if _, existe := diccionario[referenciaResuelta{
				catalogo: resumen.CatalogoCategorias, categoria: referencia,
			}]; !existe {
				return ErrDatosPublicosNoConfiables
			}
			vistas[referencia] = struct{}{}
		}
	}
	return nil
}
