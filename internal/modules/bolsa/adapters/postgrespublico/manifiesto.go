package postgrespublico

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	canonicopublico "vec-diputacion-granada/internal/modules/bolsa/publico/canonico"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/publico/dominio"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/publico/puertos"
)

const claveBloqueoPublicacion = "vec_bolsa_publica:publicacion:v1"

type huellasConvocatoriaPublica struct {
	completa string
	resumen  string
}

// cacheManifiestoPublico se construye una sola vez antes de exponer rutas. Sus
// mapas no se devuelven ni se modifican después de publicarlos atómicamente.
type cacheManifiestoPublico struct {
	porIdentificador map[string]huellasConvocatoriaPublica
}

func (f *Fuente) iniciarLectura(
	ctx context.Context,
	exigirCache bool,
) (pgx.Tx, error) {
	if ctx == nil || !f.configuracionValida() || (exigirCache && f.cacheManifiesto.Load() == nil) {
		return nil, ErrConfiguracionPostgreSQLPublicaInvalida
	}
	tx, err := f.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	if _, err := tx.Exec(ctx, `SELECT pg_catalog.pg_advisory_xact_lock_shared(
		pg_catalog.hashtextextended($1, 0)
	)`, claveBloqueoPublicacion); err != nil {
		rollbackPostgreSQLPublico(tx)
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	var ancla string
	if err := tx.QueryRow(ctx, `
		SELECT manifiesto_sha256
		  FROM vec_bolsa_publica_lectura.fuente_publica_v1
		 WHERE control_id IS TRUE`,
	).Scan(&ancla); err != nil {
		rollbackPostgreSQLPublico(tx)
		return nil, errorFilaPublica(ctx, err)
	}
	if !huellasIguales(ancla, f.manifiestoSHA256) {
		rollbackPostgreSQLPublico(tx)
		return nil, ErrDatosPostgreSQLPublicosNoConfiables
	}
	if !exigirCache {
		if _, err := tx.Exec(ctx, `SET LOCAL statement_timeout = '60s'`); err != nil {
			rollbackPostgreSQLPublico(tx)
			return nil, errorPostgreSQLPublico(ctx, err)
		}
	}
	return tx, nil
}

func (f *Fuente) construirCacheManifiesto(
	ctx context.Context,
	tx pgx.Tx,
) (*cacheManifiestoPublico, error) {
	metadatos, err := leerMetadatosFuente(ctx, tx)
	if err != nil {
		return nil, err
	}
	catalogos, err := leerCatalogos(ctx, tx)
	if err != nil {
		return nil, err
	}
	categorias, err := f.leerCategoriasManifiesto(ctx, tx)
	if err != nil {
		return nil, err
	}
	entradas, err := leerEntradasManifiestoEnFlujo(ctx, tx)
	if err != nil {
		return nil, err
	}
	materialCatalogos, err := proyectarCatalogosManifiesto(catalogos)
	if err != nil {
		return nil, err
	}
	manifiesto := canonicopublico.ManifiestoPublicoV1{
		Esquema: canonicopublico.EsquemaManifiestoPublicoV1,
		Fuente: canonicopublico.FuenteManifiestoPublicoV1{
			Revision: metadatos.Revision, ActualizadaEn: metadatos.ActualizadaEn,
		},
		Catalogos: materialCatalogos, Categorias: categorias, Convocatorias: entradas,
	}
	huella, err := manifiesto.HuellaSHA256()
	if err != nil || !huellasIguales(huella, f.manifiestoSHA256) {
		return nil, ErrDatosPostgreSQLPublicosNoConfiables
	}
	cache := &cacheManifiestoPublico{
		porIdentificador: make(map[string]huellasConvocatoriaPublica, len(entradas)),
	}
	for _, entrada := range entradas {
		cache.porIdentificador[entrada.IdentificadorPublico] = huellasConvocatoriaPublica{
			completa: entrada.HuellaCompletaSHA256, resumen: entrada.HuellaResumenSHA256,
		}
	}
	return cache, nil
}

func proyectarCatalogosManifiesto(
	catalogos []puertosbolsa.CatalogoPublico,
) ([]canonicopublico.CatalogoManifiestoV1, error) {
	resultado := make([]canonicopublico.CatalogoManifiestoV1, len(catalogos))
	for indice, catalogo := range catalogos {
		resultado[indice] = canonicopublico.CatalogoManifiestoV1{
			Referencia: catalogo.Referencia, Version: catalogo.Version,
			Entradas: make([]canonicopublico.EntradaCatalogoManifiestoV1, len(catalogo.Entradas)),
		}
		for posicion, entrada := range catalogo.Entradas {
			if !entrada.Publicable {
				return nil, ErrDatosPostgreSQLPublicosNoConfiables
			}
			resultado[indice].Entradas[posicion] = canonicopublico.EntradaCatalogoManifiestoV1{
				Clave: entrada.Clave, Etiqueta: entrada.Etiqueta,
				Descripcion: entrada.Descripcion, Semantica: entrada.Semantica, Orden: entrada.Orden,
			}
		}
	}
	return resultado, nil
}

func (f *Fuente) leerCategoriasManifiesto(
	ctx context.Context,
	tx pgx.Tx,
) (canonicopublico.CategoriasManifiestoPublicoV1, error) {
	var id, huellaGobernada, huellaProyeccion, revision string
	var version int
	var actualizadaEn time.Time
	err := tx.QueryRow(ctx, `
		SELECT catalogo_id, version, huella_gobernada_sha256,
		       huella_proyeccion_publica_sha256, revision, actualizada_en
		  FROM vec_bolsa_publica_lectura.catalogos_categorias_publicos_v1
		 WHERE catalogo_id = $1 AND version = $2`, f.catalogoCategorias, f.versionCategorias,
	).Scan(&id, &version, &huellaGobernada, &huellaProyeccion, &revision, &actualizadaEn)
	if err != nil || id != f.catalogoCategorias || version != f.versionCategorias ||
		!huellasIguales(huellaGobernada, f.huellaCategoriasGobernadaHex) ||
		!huellasIguales(huellaProyeccion, f.huellaProyeccionCategoriasHex) {
		return canonicopublico.CategoriasManifiestoPublicoV1{},
			puertosbolsa.ErrCatalogoCategoriasNoDisponible
	}
	filas, err := tx.Query(ctx, `
		SELECT clave, etiqueta, descripcion, semantica, orden, area, area_etiqueta,
		       suscribible, vigente_desde, vigente_hasta
		  FROM vec_bolsa_publica_lectura.categorias_publicas_v1
		 WHERE catalogo_id = $1 AND version = $2
	 ORDER BY orden, clave
		 LIMIT 1025`, f.catalogoCategorias, f.versionCategorias)
	if err != nil {
		return canonicopublico.CategoriasManifiestoPublicoV1{}, errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	material := make([]canonicopublico.CategoriaCatalogoV1, 0, 64)
	for filas.Next() {
		var categoria canonicopublico.CategoriaCatalogoV1
		var vigenteHasta pgtype.Timestamptz
		if err := filas.Scan(
			&categoria.Clave, &categoria.Etiqueta, &categoria.Descripcion, &categoria.Semantica,
			&categoria.Orden, &categoria.Area, &categoria.AreaEtiqueta, &categoria.Suscribible,
			&categoria.VigenteDesde, &vigenteHasta,
		); err != nil {
			return canonicopublico.CategoriasManifiestoPublicoV1{}, errorPostgreSQLPublico(ctx, err)
		}
		categoria.VigenteDesde = instanteUTC(categoria.VigenteDesde)
		if vigenteHasta.Valid {
			fin := instanteUTC(vigenteHasta.Time)
			categoria.VigenteHasta = &fin
		}
		material = append(material, categoria)
	}
	if err := filas.Err(); err != nil || len(material) == 0 || len(material) > maximoCategoriasPublicas {
		return canonicopublico.CategoriasManifiestoPublicoV1{},
			puertosbolsa.ErrCatalogoCategoriasNoDisponible
	}
	catalogo, err := canonicopublico.NuevoCatalogoCategoriasV1(id, version, material)
	if err != nil {
		return canonicopublico.CategoriasManifiestoPublicoV1{},
			puertosbolsa.ErrCatalogoCategoriasNoDisponible
	}
	return canonicopublico.CategoriasManifiestoPublicoV1{
		HuellaGobernadaSHA256: huellaGobernada, HuellaProyeccionSHA256: huellaProyeccion,
		Revision: revision, ActualizadaEn: instanteUTC(actualizadaEn),
		Catalogo: catalogo,
	}, nil
}

func leerHuellasAlmacenadas(
	ctx context.Context,
	tx pgx.Tx,
	identificadores []string,
) (map[string]huellasConvocatoriaPublica, error) {
	filas, err := tx.Query(ctx, `
		SELECT identificador_publico, huella_publica_sha256,
		       huella_resumen_publico_sha256
		  FROM vec_bolsa_publica_lectura.convocatorias_publicadas_v1
		 WHERE identificador_publico = ANY($1::text[])
	 ORDER BY identificador_publico`, identificadores)
	if err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	resultado := make(map[string]huellasConvocatoriaPublica, len(identificadores))
	for filas.Next() {
		var id string
		var huellas huellasConvocatoriaPublica
		if err := filas.Scan(&id, &huellas.completa, &huellas.resumen); err != nil {
			return nil, errorPostgreSQLPublico(ctx, err)
		}
		if _, duplicada := resultado[id]; duplicada {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
		resultado[id] = huellas
	}
	if err := filas.Err(); err != nil || len(resultado) != len(identificadores) {
		return nil, ErrDatosPostgreSQLPublicosNoConfiables
	}
	return resultado, nil
}

func validarDetallesYObtenerEntradas(
	ctx context.Context,
	tx pgx.Tx,
	convocatorias []dominiobolsa.Convocatoria,
	cache *cacheManifiestoPublico,
) ([]canonicopublico.ConvocatoriaManifiestoPublicoV1, error) {
	identificadores := make([]string, len(convocatorias))
	for indice, convocatoria := range convocatorias {
		if convocatoria.DatosPublicos == nil {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
		identificadores[indice] = convocatoria.DatosPublicos.IdentificadorPublico
	}
	almacenadas, err := leerHuellasAlmacenadas(ctx, tx, identificadores)
	if err != nil {
		return nil, err
	}
	resultado := make([]canonicopublico.ConvocatoriaManifiestoPublicoV1, len(convocatorias))
	for indice, convocatoria := range convocatorias {
		id := identificadores[indice]
		completa, err := canonicopublico.HuellaConvocatoriaV1(convocatoria)
		if err != nil {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
		resumen, err := canonicopublico.ResumenDesdeConvocatoriaV1(convocatoria)
		if err != nil {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
		huellaResumen, err := canonicopublico.HuellaResumenConvocatoriaV1(resumen)
		if err != nil {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
		esperadas, existe := almacenadas[id]
		if !existe || !huellasIguales(completa, esperadas.completa) ||
			!huellasIguales(huellaResumen, esperadas.resumen) {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
		if cache != nil {
			fijadas, existe := cache.porIdentificador[id]
			if !existe || !huellasIguales(completa, fijadas.completa) ||
				!huellasIguales(huellaResumen, fijadas.resumen) {
				return nil, ErrDatosPostgreSQLPublicosNoConfiables
			}
		}
		resultado[indice] = canonicopublico.ConvocatoriaManifiestoPublicoV1{
			IdentificadorPublico: id, HuellaCompletaSHA256: completa,
			HuellaResumenSHA256: huellaResumen,
		}
	}
	return resultado, nil
}

func validarResumenesContraManifiesto(
	ctx context.Context,
	tx pgx.Tx,
	resumenes []dominiobolsa.ResumenConvocatoria,
	cache *cacheManifiestoPublico,
) error {
	if cache == nil {
		return ErrDatosPostgreSQLPublicosNoConfiables
	}
	identificadores := make([]string, len(resumenes))
	for indice, resumen := range resumenes {
		if resumen.DatosPublicos == nil {
			return ErrDatosPostgreSQLPublicosNoConfiables
		}
		identificadores[indice] = resumen.DatosPublicos.IdentificadorPublico
	}
	almacenadas, err := leerHuellasAlmacenadas(ctx, tx, identificadores)
	if err != nil {
		return err
	}
	for indice, resumen := range resumenes {
		id := identificadores[indice]
		huellaResumen, err := canonicopublico.HuellaResumenConvocatoriaV1(resumen)
		if err != nil {
			return ErrDatosPostgreSQLPublicosNoConfiables
		}
		almacenada, existeAlmacenada := almacenadas[id]
		fijada, existeFijada := cache.porIdentificador[id]
		if !existeAlmacenada || !existeFijada ||
			!huellasIguales(resumen.HuellaSHA256, almacenada.completa) ||
			!huellasIguales(huellaResumen, almacenada.resumen) ||
			!huellasIguales(resumen.HuellaSHA256, fijada.completa) ||
			!huellasIguales(huellaResumen, fijada.resumen) {
			return ErrDatosPostgreSQLPublicosNoConfiables
		}
	}
	return nil
}

func instalarCacheUnaVez(f *Fuente, cache *cacheManifiestoPublico) error {
	if f == nil || cache == nil || cache.porIdentificador == nil {
		return ErrDatosPostgreSQLPublicosNoConfiables
	}
	if f.cacheManifiesto.CompareAndSwap(nil, cache) {
		return nil
	}
	return errors.New("bolsa publica: el manifiesto ya fue inicializado; reinicie para cambiarlo")
}
