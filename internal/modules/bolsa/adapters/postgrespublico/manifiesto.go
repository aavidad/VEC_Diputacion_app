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

type huellasConvocatoriaPublica struct {
	completa string
	resumen  string
}

// cacheManifiestoPublico se construye una sola vez antes de exponer rutas. Sus
// mapas no se devuelven ni se modifican después de publicarlos atómicamente.
type cacheManifiestoPublico struct {
	porIdentificador    map[string]huellasConvocatoriaPublica
	snapshotsCategorias []puertosbolsa.CatalogoCategoriasPublicas
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
	var ancla string
	if err := tx.QueryRow(ctx, `
		SELECT manifiesto_sha256
		  FROM vec_bolsa_publica_lectura.fuente_publica_v2
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
	manifiesto, metadatos, err := f.leerManifiestoPublico(ctx, tx)
	if err != nil {
		return nil, err
	}
	huella, err := manifiesto.HuellaSHA256()
	if err != nil || !huellasIguales(huella, f.manifiestoSHA256) {
		return nil, ErrDatosPostgreSQLPublicosNoConfiables
	}
	cache := &cacheManifiestoPublico{
		porIdentificador: make(
			map[string]huellasConvocatoriaPublica, len(manifiesto.Convocatorias),
		),
		snapshotsCategorias: snapshotsPublicosDesdeManifiesto(manifiesto.Categorias, metadatos),
	}
	for _, entrada := range manifiesto.Convocatorias {
		cache.porIdentificador[entrada.IdentificadorPublico] = huellasConvocatoriaPublica{
			completa: entrada.HuellaCompletaSHA256, resumen: entrada.HuellaResumenSHA256,
		}
	}
	return cache, nil
}

func (f *Fuente) leerManifiestoPublico(
	ctx context.Context,
	tx pgx.Tx,
) (canonicopublico.ManifiestoPublicoV2, puertosbolsa.MetadatosFuenteConvocatorias, error) {
	metadatos, err := leerMetadatosFuente(ctx, tx)
	if err != nil {
		return canonicopublico.ManifiestoPublicoV2{}, puertosbolsa.MetadatosFuenteConvocatorias{}, err
	}
	catalogos, err := leerCatalogos(ctx, tx)
	if err != nil {
		return canonicopublico.ManifiestoPublicoV2{}, puertosbolsa.MetadatosFuenteConvocatorias{}, err
	}
	categorias, err := f.leerCategoriasManifiesto(ctx, tx)
	if err != nil {
		return canonicopublico.ManifiestoPublicoV2{}, puertosbolsa.MetadatosFuenteConvocatorias{}, err
	}
	entradas, err := leerEntradasManifiestoEnFlujo(ctx, tx)
	if err != nil {
		return canonicopublico.ManifiestoPublicoV2{}, puertosbolsa.MetadatosFuenteConvocatorias{}, err
	}
	materialCatalogos, err := proyectarCatalogosManifiesto(catalogos)
	if err != nil {
		return canonicopublico.ManifiestoPublicoV2{}, puertosbolsa.MetadatosFuenteConvocatorias{}, err
	}
	manifiesto := canonicopublico.ManifiestoPublicoV2{
		Esquema: canonicopublico.EsquemaManifiestoPublicoV2,
		Fuente: canonicopublico.FuenteManifiestoPublicoV2{
			Revision: metadatos.Revision, ActualizadaEn: metadatos.ActualizadaEn,
		},
		Catalogos: materialCatalogos, Categorias: categorias, Convocatorias: entradas,
	}
	return manifiesto, metadatos, nil
}

func proyectarCatalogosManifiesto(
	catalogos []puertosbolsa.CatalogoPublico,
) ([]canonicopublico.CatalogoManifiestoV2, error) {
	resultado := make([]canonicopublico.CatalogoManifiestoV2, len(catalogos))
	for indice, catalogo := range catalogos {
		resultado[indice] = canonicopublico.CatalogoManifiestoV2{
			Referencia: catalogo.Referencia, Version: catalogo.Version,
			Entradas: make([]canonicopublico.EntradaCatalogoManifiestoV2, len(catalogo.Entradas)),
		}
		for posicion, entrada := range catalogo.Entradas {
			if !entrada.Publicable {
				return nil, ErrDatosPostgreSQLPublicosNoConfiables
			}
			resultado[indice].Entradas[posicion] = canonicopublico.EntradaCatalogoManifiestoV2{
				Clave: entrada.Clave, Etiqueta: entrada.Etiqueta,
				Descripcion: entrada.Descripcion, Semantica: entrada.Semantica, Orden: entrada.Orden,
			}
		}
	}
	return resultado, nil
}

type snapshotCategoriasLeido struct {
	actual                 bool
	huellaGobernadaSHA256  string
	huellaProyeccionSHA256 string
	revision               string
	actualizadaEn          time.Time
	catalogo               canonicopublico.CatalogoCategoriasV1
}

func (f *Fuente) leerCategoriasManifiesto(
	ctx context.Context,
	tx pgx.Tx,
) (canonicopublico.CategoriasManifiestoPublicoV2, error) {
	snapshots, err := f.leerSnapshotsCategorias(ctx, tx)
	if err != nil {
		return canonicopublico.CategoriasManifiestoPublicoV2{}, err
	}
	resultado := canonicopublico.CategoriasManifiestoPublicoV2{
		Snapshots: make([]canonicopublico.SnapshotCategoriasManifiestoV2, len(snapshots)),
	}
	for indice, snapshot := range snapshots {
		resultado.Snapshots[indice] = canonicopublico.SnapshotCategoriasManifiestoV2{
			HuellaGobernadaSHA256:  snapshot.huellaGobernadaSHA256,
			HuellaProyeccionSHA256: snapshot.huellaProyeccionSHA256,
			Catalogo:               snapshot.catalogo,
		}
		if snapshot.actual {
			resultado.Actual = canonicopublico.ReferenciaCatalogoCategoriasManifiestoV2{
				CatalogoID:                     snapshot.catalogo.CatalogoID,
				CatalogoVersion:                snapshot.catalogo.Version,
				CatalogoHuellaSHA256:           snapshot.huellaGobernadaSHA256,
				CatalogoHuellaProyeccionSHA256: snapshot.huellaProyeccionSHA256,
			}
		}
	}
	return resultado, nil
}

func (f *Fuente) leerSnapshotsCategorias(
	ctx context.Context,
	tx pgx.Tx,
) ([]snapshotCategoriasLeido, error) {
	filas, err := tx.Query(ctx, `
		SELECT catalogo.catalogo_id, catalogo.version,
		       catalogo.huella_gobernada_sha256,
		       catalogo.huella_proyeccion_publica_sha256,
		       catalogo.actual, catalogo.revision, catalogo.actualizada_en,
		       categoria.clave, categoria.etiqueta, categoria.descripcion,
		       categoria.semantica, categoria.orden, categoria.area,
		       categoria.area_etiqueta, categoria.suscribible,
		       categoria.vigente_desde, categoria.vigente_hasta
		  FROM vec_bolsa_publica_lectura.catalogos_categorias_publicos_v2 AS catalogo
		  JOIN vec_bolsa_publica_lectura.categorias_publicas_v2 AS categoria
		    ON categoria.catalogo_id = catalogo.catalogo_id
		   AND categoria.version = catalogo.version
	 ORDER BY catalogo.catalogo_id, catalogo.version, categoria.orden, categoria.clave
	 LIMIT 4097`)
	if err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	snapshots := make([]snapshotCategoriasLeido, 0, 8)
	materiales := make([][]canonicopublico.CategoriaCatalogoV1, 0, 8)
	actuales := 0
	for filas.Next() {
		var id, gobernada, proyeccion, revision string
		var version int
		var actual bool
		var actualizada time.Time
		var categoria canonicopublico.CategoriaCatalogoV1
		var vigenteHasta pgtype.Timestamptz
		if err := filas.Scan(
			&id, &version, &gobernada, &proyeccion, &actual, &revision, &actualizada,
			&categoria.Clave, &categoria.Etiqueta, &categoria.Descripcion, &categoria.Semantica,
			&categoria.Orden, &categoria.Area, &categoria.AreaEtiqueta, &categoria.Suscribible,
			&categoria.VigenteDesde, &vigenteHasta,
		); err != nil {
			return nil, errorPostgreSQLPublico(ctx, err)
		}
		if id != f.catalogoCategorias || !patronHuellaPublicaPostgreSQL.MatchString(gobernada) ||
			!patronHuellaPublicaPostgreSQL.MatchString(proyeccion) {
			return nil, puertosbolsa.ErrCatalogoCategoriasNoDisponible
		}
		posicion := len(snapshots) - 1
		if posicion < 0 || snapshots[posicion].catalogo.CatalogoID != id ||
			snapshots[posicion].catalogo.Version != version {
			if len(snapshots) >= 64 {
				return nil, puertosbolsa.ErrCatalogoCategoriasNoDisponible
			}
			snapshots = append(snapshots, snapshotCategoriasLeido{
				actual: actual, huellaGobernadaSHA256: gobernada,
				huellaProyeccionSHA256: proyeccion, revision: revision,
				actualizadaEn: instanteUTC(actualizada),
				catalogo: canonicopublico.CatalogoCategoriasV1{
					Esquema:    canonicopublico.EsquemaCatalogoCategoriasV1,
					CatalogoID: id, Version: version,
				},
			})
			materiales = append(materiales, nil)
			posicion++
			if actual {
				actuales++
			}
		} else if snapshots[posicion].actual != actual ||
			!huellasIguales(snapshots[posicion].huellaGobernadaSHA256, gobernada) ||
			!huellasIguales(snapshots[posicion].huellaProyeccionSHA256, proyeccion) ||
			snapshots[posicion].revision != revision ||
			!snapshots[posicion].actualizadaEn.Equal(instanteUTC(actualizada)) {
			return nil, puertosbolsa.ErrCatalogoCategoriasNoDisponible
		}
		categoria.VigenteDesde = instanteUTC(categoria.VigenteDesde)
		if vigenteHasta.Valid {
			fin := instanteUTC(vigenteHasta.Time)
			categoria.VigenteHasta = &fin
		}
		materiales[posicion] = append(materiales[posicion], categoria)
	}
	if err := filas.Err(); err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	if len(snapshots) == 0 || actuales != 1 {
		return nil, puertosbolsa.ErrCatalogoCategoriasNoDisponible
	}
	total := 0
	actualValido := false
	for indice := range snapshots {
		total += len(materiales[indice])
		catalogo, err := canonicopublico.NuevoCatalogoCategoriasV1(
			snapshots[indice].catalogo.CatalogoID,
			snapshots[indice].catalogo.Version,
			materiales[indice],
		)
		if err != nil {
			return nil, puertosbolsa.ErrCatalogoCategoriasNoDisponible
		}
		huella, err := catalogo.HuellaSHA256()
		if err != nil || !huellasIguales(huella, snapshots[indice].huellaProyeccionSHA256) {
			return nil, puertosbolsa.ErrCatalogoCategoriasNoDisponible
		}
		snapshots[indice].catalogo = catalogo
		if snapshots[indice].actual &&
			catalogo.Version == f.versionCategorias &&
			huellasIguales(snapshots[indice].huellaGobernadaSHA256, f.huellaCategoriasGobernadaHex) &&
			huellasIguales(snapshots[indice].huellaProyeccionSHA256, f.huellaProyeccionCategoriasHex) {
			actualValido = true
		}
	}
	if total > 4_096 || !actualValido {
		return nil, puertosbolsa.ErrCatalogoCategoriasNoDisponible
	}
	return snapshots, nil
}

func snapshotsPublicosDesdeManifiesto(
	categorias canonicopublico.CategoriasManifiestoPublicoV2,
	metadatos puertosbolsa.MetadatosFuenteConvocatorias,
) []puertosbolsa.CatalogoCategoriasPublicas {
	resultado := make([]puertosbolsa.CatalogoCategoriasPublicas, len(categorias.Snapshots))
	for indice, snapshot := range categorias.Snapshots {
		resultado[indice] = puertosbolsa.CatalogoCategoriasPublicas{
			ID: snapshot.Catalogo.CatalogoID, Version: snapshot.Catalogo.Version,
			HuellaGobernadaSHA256:  snapshot.HuellaGobernadaSHA256,
			HuellaProyeccionSHA256: snapshot.HuellaProyeccionSHA256,
			Fuente: puertosbolsa.MetadatosFuenteCategorias{
				Revision: metadatos.Revision, ActualizadaEn: metadatos.ActualizadaEn,
				Demostracion: metadatos.Demostracion, Aviso: metadatos.Aviso,
			},
			Categorias: make([]puertosbolsa.CategoriaPublica, len(snapshot.Catalogo.Categorias)),
		}
		for posicion, categoria := range snapshot.Catalogo.Categorias {
			resultado[indice].Categorias[posicion] = puertosbolsa.CategoriaPublica{
				Clave: categoria.Clave, Version: snapshot.Catalogo.Version,
				Etiqueta: categoria.Etiqueta, Descripcion: categoria.Descripcion,
				Semantica: categoria.Semantica, Orden: categoria.Orden,
				Area: categoria.Area, AreaEtiqueta: categoria.AreaEtiqueta,
				Suscribible: categoria.Suscribible,
			}
		}
	}
	return resultado
}

func leerHuellasAlmacenadas(
	ctx context.Context,
	tx pgx.Tx,
	identificadores []string,
) (map[string]huellasConvocatoriaPublica, error) {
	filas, err := tx.Query(ctx, `
		SELECT identificador_publico, huella_publica_sha256,
		       huella_resumen_publico_sha256
		  FROM vec_bolsa_publica_lectura.convocatorias_publicadas_v2
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
) ([]canonicopublico.ConvocatoriaManifiestoPublicoV2, error) {
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
	resultado := make([]canonicopublico.ConvocatoriaManifiestoPublicoV2, len(convocatorias))
	for indice, convocatoria := range convocatorias {
		id := identificadores[indice]
		completa, err := canonicopublico.HuellaConvocatoriaV2(convocatoria)
		if err != nil {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
		resumen, err := canonicopublico.ResumenDesdeConvocatoriaV2(convocatoria)
		if err != nil {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
		huellaResumen, err := canonicopublico.HuellaResumenConvocatoriaV2(resumen)
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
		resultado[indice] = canonicopublico.ConvocatoriaManifiestoPublicoV2{
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
		huellaResumen, err := canonicopublico.HuellaResumenConvocatoriaV2(resumen)
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
	if f == nil || cache == nil || cache.porIdentificador == nil || len(cache.snapshotsCategorias) == 0 {
		return ErrDatosPostgreSQLPublicosNoConfiables
	}
	if f.cacheManifiesto.CompareAndSwap(nil, cache) {
		f.iniciarVigilanciaIntegridad()
		return nil
	}
	return errors.New("bolsa publica: el manifiesto ya fue inicializado; reinicie para cambiarlo")
}

func clonarSnapshotsCategorias(
	cache *cacheManifiestoPublico,
) ([]puertosbolsa.CatalogoCategoriasPublicas, error) {
	if cache == nil || len(cache.snapshotsCategorias) == 0 {
		return nil, ErrDatosPostgreSQLPublicosNoConfiables
	}
	resultado := make([]puertosbolsa.CatalogoCategoriasPublicas, len(cache.snapshotsCategorias))
	for indice, snapshot := range cache.snapshotsCategorias {
		resultado[indice] = snapshot
		resultado[indice].Categorias = append(
			[]puertosbolsa.CategoriaPublica(nil), snapshot.Categorias...,
		)
	}
	return resultado, nil
}
