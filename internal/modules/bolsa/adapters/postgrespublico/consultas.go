package postgrespublico

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	canonicopublico "vec-diputacion-granada/internal/modules/bolsa/publico/canonico"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/publico/dominio"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/publico/puertos"
	postgresqlcompartido "vec-diputacion-granada/internal/shared/postgresql"
)

const (
	maximoResultadosPostgreSQLPublico     = 100
	maximoDesplazamientoPostgreSQLPublico = 12_000
	maximoEntradasCatalogosPublicos       = 1_024
	maximoEntradasPorCatalogoPublico      = 256
	maximoCategoriasPublicas              = 1_024
)

func (f *Fuente) BuscarPublicadas(
	ctx context.Context,
	filtro puertosbolsa.FiltroConvocatoriasPublicas,
) (puertosbolsa.PaginaConvocatorias, error) {
	if ctx == nil || !f.valida() || !filtroConvocatoriasValido(filtro) {
		return puertosbolsa.PaginaConvocatorias{}, puertosbolsa.ErrConsultaConvocatoriasInvalida
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.PaginaConvocatorias{}, err
	}
	tx, err := f.iniciarLectura(ctx, true)
	if err != nil {
		return puertosbolsa.PaginaConvocatorias{}, errorPostgreSQLPublico(ctx, err)
	}
	defer rollbackPostgreSQLPublico(tx)

	resultado, err := f.buscarPublicadasTx(ctx, tx, filtro)
	if err != nil {
		return puertosbolsa.PaginaConvocatorias{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return puertosbolsa.PaginaConvocatorias{}, errorPostgreSQLPublico(ctx, err)
	}
	return resultado, nil
}

func (f *Fuente) ObtenerPublicada(
	ctx context.Context,
	identificador string,
) (puertosbolsa.DetalleConvocatoria, error) {
	if ctx == nil || !f.valida() || identificador != strings.TrimSpace(identificador) ||
		!patronIdentificadorPostgreSQL.MatchString(identificador) {
		return puertosbolsa.DetalleConvocatoria{}, puertosbolsa.ErrConsultaConvocatoriasInvalida
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.DetalleConvocatoria{}, err
	}
	tx, err := f.iniciarLectura(ctx, true)
	if err != nil {
		return puertosbolsa.DetalleConvocatoria{}, errorPostgreSQLPublico(ctx, err)
	}
	defer rollbackPostgreSQLPublico(tx)
	resultado, err := f.obtenerPublicadaTx(ctx, tx, identificador)
	if err != nil {
		return puertosbolsa.DetalleConvocatoria{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return puertosbolsa.DetalleConvocatoria{}, errorPostgreSQLPublico(ctx, err)
	}
	return resultado, nil
}

func (f *Fuente) BuscarPublicadasConCategorias(
	ctx context.Context,
	filtro puertosbolsa.FiltroConvocatoriasPublicas,
) (puertosbolsa.LecturaListadoPublicoConsistente, error) {
	if ctx == nil || !f.valida() || !filtroConvocatoriasValido(filtro) {
		return puertosbolsa.LecturaListadoPublicoConsistente{}, puertosbolsa.ErrConsultaConvocatoriasInvalida
	}
	tx, err := f.iniciarLectura(ctx, true)
	if err != nil {
		return puertosbolsa.LecturaListadoPublicoConsistente{}, errorPostgreSQLPublico(ctx, err)
	}
	defer rollbackPostgreSQLPublico(tx)
	categorias, err := f.leerCategorias(ctx, tx, filtro.Instante)
	if err != nil {
		return puertosbolsa.LecturaListadoPublicoConsistente{}, err
	}
	pagina, err := f.buscarPublicadasTx(ctx, tx, filtro)
	if err != nil {
		return puertosbolsa.LecturaListadoPublicoConsistente{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return puertosbolsa.LecturaListadoPublicoConsistente{}, errorPostgreSQLPublico(ctx, err)
	}
	return puertosbolsa.LecturaListadoPublicoConsistente{Pagina: pagina, Categorias: categorias}, nil
}

func (f *Fuente) ObtenerPublicadaConCategorias(
	ctx context.Context,
	identificador string,
	instante time.Time,
) (puertosbolsa.LecturaDetallePublicoConsistente, error) {
	instante = instante.UTC().Truncate(time.Microsecond)
	if ctx == nil || !f.valida() || instante.IsZero() ||
		identificador != strings.TrimSpace(identificador) ||
		!patronIdentificadorPostgreSQL.MatchString(identificador) {
		return puertosbolsa.LecturaDetallePublicoConsistente{}, puertosbolsa.ErrConsultaConvocatoriasInvalida
	}
	tx, err := f.iniciarLectura(ctx, true)
	if err != nil {
		return puertosbolsa.LecturaDetallePublicoConsistente{}, errorPostgreSQLPublico(ctx, err)
	}
	defer rollbackPostgreSQLPublico(tx)
	categorias, err := f.leerCategorias(ctx, tx, instante)
	if err != nil {
		return puertosbolsa.LecturaDetallePublicoConsistente{}, err
	}
	detalle, err := f.obtenerPublicadaTx(ctx, tx, identificador)
	if err != nil {
		return puertosbolsa.LecturaDetallePublicoConsistente{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return puertosbolsa.LecturaDetallePublicoConsistente{}, errorPostgreSQLPublico(ctx, err)
	}
	return puertosbolsa.LecturaDetallePublicoConsistente{Detalle: detalle, Categorias: categorias}, nil
}

func (f *Fuente) ConsultarCategoriasConConteos(
	ctx context.Context,
	instante time.Time,
) (puertosbolsa.LecturaCategoriasPublicasConsistente, error) {
	instante = instante.UTC().Truncate(time.Microsecond)
	if ctx == nil || !f.valida() || instante.IsZero() {
		return puertosbolsa.LecturaCategoriasPublicasConsistente{}, puertosbolsa.ErrConsultaCategoriasInvalida
	}
	tx, err := f.iniciarLectura(ctx, true)
	if err != nil {
		return puertosbolsa.LecturaCategoriasPublicasConsistente{}, errorPostgreSQLPublico(ctx, err)
	}
	defer rollbackPostgreSQLPublico(tx)
	categorias, err := f.leerCategorias(ctx, tx, instante)
	if err != nil {
		return puertosbolsa.LecturaCategoriasPublicasConsistente{}, err
	}
	pagina, err := f.buscarPublicadasTx(ctx, tx, puertosbolsa.FiltroConvocatoriasPublicas{
		Instante: instante, Limite: 1,
	})
	if err != nil {
		return puertosbolsa.LecturaCategoriasPublicasConsistente{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return puertosbolsa.LecturaCategoriasPublicasConsistente{}, errorPostgreSQLPublico(ctx, err)
	}
	return puertosbolsa.LecturaCategoriasPublicasConsistente{Pagina: pagina, Categorias: categorias}, nil
}

func (f *Fuente) ValidarConfiguracionPublica(ctx context.Context, instante time.Time) error {
	instante = instante.UTC().Truncate(time.Microsecond)
	if ctx == nil || !f.configuracionValida() || instante.IsZero() {
		return puertosbolsa.ErrConsultaConvocatoriasInvalida
	}
	ctx, cancelar := context.WithTimeout(ctx, duracionMaximaManifiestoArranque)
	defer cancelar()
	tx, err := f.iniciarLectura(ctx, false)
	if err != nil {
		return errorPostgreSQLPublico(ctx, err)
	}
	defer rollbackPostgreSQLPublico(tx)
	if _, err := f.leerCategorias(ctx, tx, instante); err != nil {
		return err
	}
	if _, err := leerMetadatosFuente(ctx, tx); err != nil {
		return err
	}
	if _, err := leerCatalogos(ctx, tx); err != nil {
		return err
	}
	var valida bool
	err = tx.QueryRow(ctx, `
		SELECT
		    (SELECT count(*) <= 12000
		       FROM vec_bolsa_publica_lectura.convocatorias_publicadas_v1)
		AND NOT EXISTS (
		    SELECT requerido.referencia
		      FROM (VALUES
		          ('tipos_convocatoria'), ('estados_convocatoria'),
		          ('tipos_plazo'), ('tipos_documento'), ('categorias_ayuda')
		      ) AS requerido(referencia)
		     WHERE NOT EXISTS (
		         SELECT 1
		           FROM vec_bolsa_publica_lectura.entradas_catalogos_publicos_v1 AS entrada
		          WHERE entrada.referencia = requerido.referencia
		     )
		)
		AND NOT EXISTS (
		    SELECT 1
		      FROM vec_bolsa_publica_lectura.convocatorias_publicadas_v1 AS convocatoria
		     WHERE convocatoria.catalogo_categorias_id <> $1
		        OR convocatoria.catalogo_categorias_version <> $2
		        OR convocatoria.catalogo_categorias_huella_sha256 <> $3
		)
		AND NOT EXISTS (
		    SELECT 1
		      FROM vec_bolsa_publica_lectura.categorias_convocatorias_publicas_v1 AS vinculo
		      LEFT JOIN vec_bolsa_publica_lectura.categorias_publicas_v1 AS categoria
		        ON categoria.catalogo_id = $1 AND categoria.version = $2
		       AND categoria.clave = vinculo.categoria_clave
		       AND categoria.vigente_desde <= $4
		       AND (categoria.vigente_hasta IS NULL OR $4 < categoria.vigente_hasta)
		     WHERE categoria.clave IS NULL
		)
		AND NOT EXISTS (
		    SELECT 1
		      FROM vec_bolsa_publica_lectura.convocatorias_publicadas_v1 AS convocatoria
		      LEFT JOIN vec_bolsa_publica_lectura.entradas_catalogos_publicos_v1 AS tipo
		        ON tipo.referencia = 'tipos_convocatoria' AND tipo.clave = convocatoria.tipo
		      LEFT JOIN vec_bolsa_publica_lectura.entradas_catalogos_publicos_v1 AS estado
		        ON estado.referencia = 'estados_convocatoria' AND estado.clave = convocatoria.estado
		     WHERE tipo.clave IS NULL OR estado.clave IS NULL
		)
		AND NOT EXISTS (
		    SELECT 1
		      FROM vec_bolsa_publica_lectura.plazos_convocatorias_publicas_v1 AS plazo
		      LEFT JOIN vec_bolsa_publica_lectura.entradas_catalogos_publicos_v1 AS tipo
		        ON tipo.referencia = 'tipos_plazo' AND tipo.clave = plazo.tipo
		     WHERE tipo.clave IS NULL
		)
		AND NOT EXISTS (
		    SELECT 1
		      FROM vec_bolsa_publica_lectura.documentos_convocatorias_publicas_v1 AS documento
		      LEFT JOIN vec_bolsa_publica_lectura.entradas_catalogos_publicos_v1 AS tipo
		        ON tipo.referencia = 'tipos_documento' AND tipo.clave = documento.tipo
		     WHERE tipo.clave IS NULL
		)
		AND NOT EXISTS (
		    SELECT 1
		      FROM vec_bolsa_publica_lectura.ayuda_convocatorias_publicas_v1 AS ayuda
		      LEFT JOIN vec_bolsa_publica_lectura.entradas_catalogos_publicos_v1 AS categoria
		        ON categoria.referencia = 'categorias_ayuda' AND categoria.clave = ayuda.categoria
		     WHERE categoria.clave IS NULL
		)`, f.catalogoCategorias, f.versionCategorias, f.huellaCategoriasGobernadaHex, instante,
	).Scan(&valida)
	if err != nil {
		return errorPostgreSQLPublico(ctx, err)
	}
	if !valida {
		return ErrDatosPostgreSQLPublicosNoConfiables
	}
	cache, err := f.construirCacheManifiesto(ctx, tx)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return errorPostgreSQLPublico(ctx, err)
	}
	return instalarCacheUnaVez(f, cache)
}

func (f *Fuente) buscarPublicadasTx(
	ctx context.Context,
	tx pgx.Tx,
	filtro puertosbolsa.FiltroConvocatoriasPublicas,
) (puertosbolsa.PaginaConvocatorias, error) {
	fuente, err := leerMetadatosFuente(ctx, tx)
	if err != nil {
		return puertosbolsa.PaginaConvocatorias{}, err
	}
	catalogos, err := leerCatalogos(ctx, tx)
	if err != nil {
		return puertosbolsa.PaginaConvocatorias{}, err
	}
	conteos, err := leerConteosCategorias(ctx, tx, filtro)
	if err != nil {
		return puertosbolsa.PaginaConvocatorias{}, err
	}
	total, referencias, err := buscarReferencias(ctx, tx, filtro)
	if err != nil {
		return puertosbolsa.PaginaConvocatorias{}, err
	}
	convocatorias, err := leerResumenesConvocatorias(ctx, tx, referencias)
	if err != nil {
		return puertosbolsa.PaginaConvocatorias{}, err
	}
	if err := validarResumenesContraManifiesto(ctx, tx, convocatorias, f.cacheManifiesto.Load()); err != nil {
		return puertosbolsa.PaginaConvocatorias{}, err
	}
	return puertosbolsa.PaginaConvocatorias{
		Convocatorias: convocatorias, Total: total, Catalogos: catalogos,
		ConteosCategorias: conteos, Fuente: fuente,
	}, nil
}

func (f *Fuente) obtenerPublicadaTx(
	ctx context.Context,
	tx pgx.Tx,
	identificador string,
) (puertosbolsa.DetalleConvocatoria, error) {
	fuente, err := leerMetadatosFuente(ctx, tx)
	if err != nil {
		return puertosbolsa.DetalleConvocatoria{}, err
	}
	catalogos, err := leerCatalogos(ctx, tx)
	if err != nil {
		return puertosbolsa.DetalleConvocatoria{}, err
	}
	convocatoria, err := leerConvocatoria(ctx, tx, identificador)
	if err != nil {
		return puertosbolsa.DetalleConvocatoria{}, err
	}
	if _, err := validarDetallesYObtenerEntradas(
		ctx, tx, []dominiobolsa.Convocatoria{convocatoria}, f.cacheManifiesto.Load(),
	); err != nil {
		return puertosbolsa.DetalleConvocatoria{}, err
	}
	return puertosbolsa.DetalleConvocatoria{
		Convocatoria: convocatoria, Catalogos: catalogos, Fuente: fuente,
	}, nil
}

func (f *Fuente) ObtenerPublicadas(
	ctx context.Context,
	instante time.Time,
) (puertosbolsa.CatalogoCategoriasPublicas, error) {
	instante = instante.UTC().Truncate(time.Microsecond)
	if ctx == nil || !f.valida() || instante.IsZero() {
		return puertosbolsa.CatalogoCategoriasPublicas{}, puertosbolsa.ErrConsultaCategoriasInvalida
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.CatalogoCategoriasPublicas{}, err
	}
	tx, err := f.iniciarLectura(ctx, true)
	if err != nil {
		return puertosbolsa.CatalogoCategoriasPublicas{}, errorPostgreSQLPublico(ctx, err)
	}
	defer rollbackPostgreSQLPublico(tx)
	resultado, err := f.leerCategorias(ctx, tx, instante)
	if err != nil {
		return puertosbolsa.CatalogoCategoriasPublicas{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return puertosbolsa.CatalogoCategoriasPublicas{}, errorPostgreSQLPublico(ctx, err)
	}
	return resultado, nil
}

func (f *Fuente) configuracionValida() bool {
	return f != nil && f.pool != nil &&
		patronIDCatalogoPublicoPostgreSQL.MatchString(f.catalogoCategorias) &&
		f.versionCategorias > 0 &&
		patronHuellaPublicaPostgreSQL.MatchString(f.huellaCategoriasGobernadaHex) &&
		patronHuellaPublicaPostgreSQL.MatchString(f.huellaProyeccionCategoriasHex) &&
		patronHuellaPublicaPostgreSQL.MatchString(f.manifiestoSHA256)
}

func (f *Fuente) valida() bool {
	return f.configuracionValida() && f.cacheManifiesto.Load() != nil
}

func filtroConvocatoriasValido(f puertosbolsa.FiltroConvocatoriasPublicas) bool {
	return f.Texto == strings.TrimSpace(f.Texto) && len([]rune(f.Texto)) <= 100 &&
		f.Limite >= 1 && f.Limite <= maximoResultadosPostgreSQLPublico &&
		f.Desplazamiento >= 0 && f.Desplazamiento <= maximoDesplazamientoPostgreSQLPublico &&
		!f.Instante.IsZero() && f.Instante.Location() == time.UTC && f.Instante.Nanosecond()%1000 == 0 &&
		claveFiltroPostgreSQLValida(f.Tipo) && claveFiltroPostgreSQLValida(f.Categoria) &&
		claveFiltroPostgreSQLValida(f.Estado)
}

func claveFiltroPostgreSQLValida(valor string) bool {
	if valor == "" {
		return true
	}
	if valor != strings.TrimSpace(valor) || len(valor) > 80 {
		return false
	}
	for indice, caracter := range valor {
		if caracter >= 'a' && caracter <= 'z' || caracter >= 'A' && caracter <= 'Z' ||
			caracter >= '0' && caracter <= '9' || indice > 0 && strings.ContainsRune("._-", caracter) {
			continue
		}
		return false
	}
	return true
}

func leerMetadatosFuente(
	ctx context.Context,
	tx pgx.Tx,
) (puertosbolsa.MetadatosFuenteConvocatorias, error) {
	var revision string
	var actualizada time.Time
	err := tx.QueryRow(ctx, `
		SELECT revision, actualizada_en
		  FROM vec_bolsa_publica_lectura.fuente_publica_v1
		 WHERE control_id IS TRUE`,
	).Scan(&revision, &actualizada)
	if err != nil {
		return puertosbolsa.MetadatosFuenteConvocatorias{}, errorFilaPublica(ctx, err)
	}
	return puertosbolsa.MetadatosFuenteConvocatorias{
		Revision: revision, ActualizadaEn: instanteUTC(actualizada), Demostracion: false,
	}, nil
}

func leerCatalogos(ctx context.Context, tx pgx.Tx) ([]puertosbolsa.CatalogoPublico, error) {
	filas, err := tx.Query(ctx, `
		SELECT referencia, version, clave, etiqueta, descripcion, semantica, orden, publicable
		  FROM vec_bolsa_publica_lectura.entradas_catalogos_publicos_v1
	 ORDER BY referencia, version, orden, clave
	 LIMIT 1025`)
	if err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	resultado := make([]puertosbolsa.CatalogoPublico, 0, 8)
	indice := make(map[string]int)
	for filas.Next() {
		var referencia, clave, etiqueta, descripcion, semantica string
		var version, orden int
		var publicable bool
		if err := filas.Scan(&referencia, &version, &clave, &etiqueta, &descripcion, &semantica, &orden, &publicable); err != nil {
			return nil, errorPostgreSQLPublico(ctx, err)
		}
		posicion, existe := indice[referencia]
		if !existe {
			posicion = len(resultado)
			indice[referencia] = posicion
			resultado = append(resultado, puertosbolsa.CatalogoPublico{Referencia: referencia, Version: version})
		} else if resultado[posicion].Version != version {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
		resultado[posicion].Entradas = append(resultado[posicion].Entradas, puertosbolsa.EntradaCatalogoPublico{
			Clave: clave, Version: version, Etiqueta: etiqueta, Descripcion: descripcion,
			Semantica: semantica, Orden: orden, Publicable: publicable,
		})
		if len(resultado[posicion].Entradas) > maximoEntradasPorCatalogoPublico {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
	}
	if err := filas.Err(); err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	totalEntradas := 0
	for _, catalogo := range resultado {
		totalEntradas += len(catalogo.Entradas)
	}
	if len(resultado) == 0 || totalEntradas > maximoEntradasCatalogosPublicos {
		return nil, ErrDatosPostgreSQLPublicosNoConfiables
	}
	return resultado, nil
}

func (f *Fuente) leerCategorias(
	ctx context.Context,
	tx pgx.Tx,
	instante time.Time,
) (puertosbolsa.CatalogoCategoriasPublicas, error) {
	var id, huellaGobernada, huellaProyeccion, revision string
	var version int
	var actualizada time.Time
	err := tx.QueryRow(ctx, `
		SELECT catalogo_id, version, huella_gobernada_sha256,
		       huella_proyeccion_publica_sha256, revision, actualizada_en
		  FROM vec_bolsa_publica_lectura.catalogos_categorias_publicos_v1
		 WHERE catalogo_id = $1 AND version = $2`, f.catalogoCategorias, f.versionCategorias,
	).Scan(&id, &version, &huellaGobernada, &huellaProyeccion, &revision, &actualizada)
	if err != nil || id != f.catalogoCategorias || version != f.versionCategorias ||
		!huellasIguales(huellaGobernada, f.huellaCategoriasGobernadaHex) ||
		!huellasIguales(huellaProyeccion, f.huellaProyeccionCategoriasHex) {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return puertosbolsa.CatalogoCategoriasPublicas{}, err
		}
		return puertosbolsa.CatalogoCategoriasPublicas{}, puertosbolsa.ErrCatalogoCategoriasNoDisponible
	}
	filas, err := tx.Query(ctx, `
		SELECT clave, etiqueta, descripcion, semantica, orden, area, area_etiqueta,
		       suscribible, vigente_desde, vigente_hasta
		  FROM vec_bolsa_publica_lectura.categorias_publicas_v1
		 WHERE catalogo_id = $1 AND version = $2
	 ORDER BY orden, clave
	 LIMIT 1025`, f.catalogoCategorias, f.versionCategorias,
	)
	if err != nil {
		return puertosbolsa.CatalogoCategoriasPublicas{}, errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	resultado := puertosbolsa.CatalogoCategoriasPublicas{
		ID: id, Version: version, HuellaSHA256: huellaGobernada,
		Fuente: puertosbolsa.MetadatosFuenteCategorias{
			Revision: revision, ActualizadaEn: instanteUTC(actualizada), Demostracion: false,
		},
	}
	materialCategorias := make([]canonicopublico.CategoriaCatalogoV1, 0, 64)
	for filas.Next() {
		var categoria puertosbolsa.CategoriaPublica
		var material canonicopublico.CategoriaCatalogoV1
		var vigenteHasta pgtype.Timestamptz
		if err := filas.Scan(
			&material.Clave, &material.Etiqueta, &material.Descripcion, &material.Semantica,
			&material.Orden, &material.Area, &material.AreaEtiqueta, &material.Suscribible,
			&material.VigenteDesde, &vigenteHasta,
		); err != nil {
			return puertosbolsa.CatalogoCategoriasPublicas{}, errorPostgreSQLPublico(ctx, err)
		}
		material.VigenteDesde = instanteUTC(material.VigenteDesde)
		if vigenteHasta.Valid {
			fin := instanteUTC(vigenteHasta.Time)
			material.VigenteHasta = &fin
		}
		materialCategorias = append(materialCategorias, material)
		if !instante.Before(material.VigenteDesde) &&
			(material.VigenteHasta == nil || instante.Before(*material.VigenteHasta)) {
			categoria = puertosbolsa.CategoriaPublica{
				Clave: material.Clave, Version: version, Etiqueta: material.Etiqueta,
				Descripcion: material.Descripcion, Semantica: material.Semantica,
				Orden: material.Orden, Area: material.Area, AreaEtiqueta: material.AreaEtiqueta,
				Suscribible: material.Suscribible,
			}
			resultado.Categorias = append(resultado.Categorias, categoria)
		}
	}
	if err := filas.Err(); err != nil {
		return puertosbolsa.CatalogoCategoriasPublicas{}, errorPostgreSQLPublico(ctx, err)
	}
	if len(materialCategorias) == 0 || len(materialCategorias) > maximoCategoriasPublicas ||
		len(resultado.Categorias) == 0 {
		return puertosbolsa.CatalogoCategoriasPublicas{}, puertosbolsa.ErrCatalogoCategoriasNoDisponible
	}
	catalogoCanonico, err := canonicopublico.NuevoCatalogoCategoriasV1(id, version, materialCategorias)
	if err != nil {
		return puertosbolsa.CatalogoCategoriasPublicas{}, puertosbolsa.ErrCatalogoCategoriasNoDisponible
	}
	huellaCalculada, err := catalogoCanonico.HuellaSHA256()
	if err != nil || !huellasIguales(huellaCalculada, huellaProyeccion) ||
		!huellasIguales(huellaCalculada, f.huellaProyeccionCategoriasHex) {
		return puertosbolsa.CatalogoCategoriasPublicas{}, puertosbolsa.ErrCatalogoCategoriasNoDisponible
	}
	return resultado, nil
}

func instanteUTC(instante time.Time) time.Time {
	return instante.UTC().Truncate(time.Microsecond)
}

func errorFilaPublica(ctx context.Context, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrDatosPostgreSQLPublicosNoConfiables
	}
	return errorPostgreSQLPublico(ctx, err)
}

func rollbackPostgreSQLPublico(tx pgx.Tx) {
	postgresqlcompartido.RevertirAcotado(tx)
}
