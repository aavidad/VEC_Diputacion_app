package postgrespublico

import (
	"context"

	"github.com/jackc/pgx/v5"

	canonicopublico "vec-diputacion-granada/internal/modules/bolsa/publico/canonico"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/publico/dominio"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/publico/puertos"
)

const predicadoFiltroConvocatorias = `
       ($1::text = '' OR convocatoria.busqueda @@ websearch_to_tsquery('spanish', $1))
   AND ($2::text = '' OR convocatoria.tipo = $2)
   AND ($3::text = '' OR convocatoria.estado = $3)
   AND (NOT $4::boolean OR EXISTS (
       SELECT 1
         FROM vec_bolsa_publica_lectura.plazos_convocatorias_publicas_v1 AS plazo
        WHERE plazo.identificador_publico = convocatoria.identificador_publico
          AND plazo.abre_en <= $5
          AND $5 <= plazo.cierra_en
   ))`

const (
	maximoCategoriasPorConvocatoria = 128
	maximoPlazosPorConvocatoria     = 64
	maximoRequisitosPorConvocatoria = 256
	maximoDocumentosPorConvocatoria = 256
	maximoAyudasPorConvocatoria     = 128
)

func leerResumenesConvocatorias(
	ctx context.Context,
	tx pgx.Tx,
	identificadores []string,
) ([]dominiobolsa.ResumenConvocatoria, error) {
	if len(identificadores) == 0 {
		return []dominiobolsa.ResumenConvocatoria{}, nil
	}
	indice := make(map[string]int, len(identificadores))
	for posicion, identificador := range identificadores {
		if _, duplicado := indice[identificador]; duplicado {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
		indice[identificador] = posicion
	}
	resultado := make([]dominiobolsa.ResumenConvocatoria, len(identificadores))
	presentes := make([]bool, len(identificadores))
	filas, err := tx.Query(ctx, `
		SELECT convocatoria.version_publica, convocatoria.estado,
		       convocatoria.identificador_publico, convocatoria.tipo,
		       convocatoria.catalogo_categorias_id,
		       convocatoria.catalogo_categorias_version,
		       convocatoria.catalogo_categorias_huella_sha256,
		       convocatoria.huella_publica_sha256,
		       convocatoria.titulo, convocatoria.resumen,
		       convocatoria.publicada_en, convocatoria.actualizada_en,
		       (SELECT count(*)::integer
		          FROM vec_bolsa_publica_lectura.requisitos_convocatorias_publicas_v1 AS requisito
		         WHERE requisito.identificador_publico = convocatoria.identificador_publico),
		       (SELECT count(*)::integer
		          FROM vec_bolsa_publica_lectura.documentos_convocatorias_publicas_v1 AS documento
		         WHERE documento.identificador_publico = convocatoria.identificador_publico),
		       (SELECT count(*)::integer
		          FROM vec_bolsa_publica_lectura.ayuda_convocatorias_publicas_v1 AS ayuda
		         WHERE ayuda.identificador_publico = convocatoria.identificador_publico)
		  FROM vec_bolsa_publica_lectura.convocatorias_publicadas_v1 AS convocatoria
		 WHERE convocatoria.identificador_publico = ANY($1::text[])
	 ORDER BY convocatoria.identificador_publico`, identificadores)
	if err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	for filas.Next() {
		var convocatoria dominiobolsa.ResumenConvocatoria
		datos := &dominiobolsa.DatosPublicosResumenConvocatoria{}
		var estado string
		if err := filas.Scan(
			&convocatoria.Version, &estado, &datos.IdentificadorPublico, &datos.Tipo,
			&datos.CatalogoCategorias.CatalogoID, &datos.CatalogoCategorias.CatalogoVersion,
			&datos.CatalogoCategorias.CatalogoHuellaSHA256, &convocatoria.HuellaSHA256,
			&datos.Titulo, &datos.Resumen, &datos.PublicadaEn, &datos.ActualizadaEn,
			&convocatoria.NumeroRequisitos, &convocatoria.NumeroDocumentos,
			&convocatoria.NumeroAyudas,
		); err != nil {
			filas.Close()
			return nil, errorPostgreSQLPublico(ctx, err)
		}
		posicion, esperada := indice[datos.IdentificadorPublico]
		if !esperada || presentes[posicion] {
			filas.Close()
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
		convocatoria.Estado = dominiobolsa.EstadoConvocatoria(estado)
		datos.PublicadaEn, datos.ActualizadaEn = instanteUTC(datos.PublicadaEn), instanteUTC(datos.ActualizadaEn)
		convocatoria.DatosPublicos = datos
		resultado[posicion], presentes[posicion] = convocatoria, true
	}
	if err := filas.Err(); err != nil {
		filas.Close()
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	filas.Close()
	for _, presente := range presentes {
		if !presente {
			return nil, puertosbolsa.ErrConvocatoriaNoEncontrada
		}
	}
	if err := leerCategoriasResumenesLote(ctx, tx, identificadores, resultado, indice); err != nil {
		return nil, err
	}
	if err := leerPlazosResumenesLote(ctx, tx, identificadores, resultado, indice); err != nil {
		return nil, err
	}
	for _, convocatoria := range resultado {
		if err := convocatoria.ValidarPublicacion(); err != nil {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
	}
	return resultado, nil
}

func leerCategoriasResumenesLote(
	ctx context.Context, tx pgx.Tx, ids []string,
	convocatorias []dominiobolsa.ResumenConvocatoria, indice map[string]int,
) error {
	filas, err := tx.Query(ctx, `
		SELECT identificador_publico, categoria_clave
		  FROM vec_bolsa_publica_lectura.categorias_convocatorias_publicas_v1
		 WHERE identificador_publico = ANY($1::text[])
	 ORDER BY identificador_publico, categoria_clave
		 LIMIT $2`, ids, len(ids)*(maximoCategoriasPorConvocatoria+1))
	if err != nil {
		return errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	for filas.Next() {
		var id, clave string
		if err := filas.Scan(&id, &clave); err != nil {
			return errorPostgreSQLPublico(ctx, err)
		}
		posicion, existe := indice[id]
		if !existe {
			return ErrDatosPostgreSQLPublicosNoConfiables
		}
		datos := convocatorias[posicion].DatosPublicos
		datos.Categorias = append(datos.Categorias, clave)
		if len(datos.Categorias) > maximoCategoriasPorConvocatoria {
			return ErrDatosPostgreSQLPublicosNoConfiables
		}
	}
	return errorFilasPublicas(ctx, filas)
}

func leerPlazosResumenesLote(
	ctx context.Context, tx pgx.Tx, ids []string,
	convocatorias []dominiobolsa.ResumenConvocatoria, indice map[string]int,
) error {
	filas, err := tx.Query(ctx, `
		SELECT identificador_publico, referencia, tipo, titulo, descripcion, abre_en, cierra_en
		  FROM vec_bolsa_publica_lectura.plazos_convocatorias_publicas_v1
		 WHERE identificador_publico = ANY($1::text[])
	 ORDER BY identificador_publico, referencia
		 LIMIT $2`, ids, len(ids)*(maximoPlazosPorConvocatoria+1))
	if err != nil {
		return errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	for filas.Next() {
		var id string
		var plazo dominiobolsa.PlazoConvocatoria
		if err := filas.Scan(&id, &plazo.Referencia, &plazo.Tipo, &plazo.Titulo, &plazo.Descripcion, &plazo.AbreEn, &plazo.CierraEn); err != nil {
			return errorPostgreSQLPublico(ctx, err)
		}
		posicion, existe := indice[id]
		if !existe {
			return ErrDatosPostgreSQLPublicosNoConfiables
		}
		plazo.AbreEn, plazo.CierraEn = instanteUTC(plazo.AbreEn), instanteUTC(plazo.CierraEn)
		datos := convocatorias[posicion].DatosPublicos
		datos.Plazos = append(datos.Plazos, plazo)
		if len(datos.Plazos) > maximoPlazosPorConvocatoria {
			return ErrDatosPostgreSQLPublicosNoConfiables
		}
	}
	return errorFilasPublicas(ctx, filas)
}

func buscarReferencias(
	ctx context.Context,
	tx pgx.Tx,
	filtro puertosbolsa.FiltroConvocatoriasPublicas,
) (int, []string, error) {
	argumentos := []any{
		filtro.Texto, filtro.Tipo, filtro.Estado, filtro.SoloPlazoAbierto, filtro.Instante,
	}
	var total int
	err := tx.QueryRow(ctx, `
		SELECT count(*)::integer
		  FROM vec_bolsa_publica_lectura.convocatorias_publicadas_v1 AS convocatoria
		 WHERE `+predicadoFiltroConvocatorias+`
		   AND ($6::text = '' OR EXISTS (
		       SELECT 1
		         FROM vec_bolsa_publica_lectura.categorias_convocatorias_publicas_v1 AS categoria
		        WHERE categoria.identificador_publico = convocatoria.identificador_publico
		          AND categoria.categoria_clave = $6
		   ))`, append(argumentos, filtro.Categoria)...,
	).Scan(&total)
	if err != nil {
		return 0, nil, errorPostgreSQLPublico(ctx, err)
	}
	filas, err := tx.Query(ctx, `
		SELECT convocatoria.identificador_publico
		  FROM vec_bolsa_publica_lectura.convocatorias_publicadas_v1 AS convocatoria
		 WHERE `+predicadoFiltroConvocatorias+`
		   AND ($6::text = '' OR EXISTS (
		       SELECT 1
		         FROM vec_bolsa_publica_lectura.categorias_convocatorias_publicas_v1 AS categoria
		        WHERE categoria.identificador_publico = convocatoria.identificador_publico
		          AND categoria.categoria_clave = $6
		   ))
	 ORDER BY convocatoria.actualizada_en DESC, convocatoria.identificador_publico
	 LIMIT $7 OFFSET $8`, append(argumentos, filtro.Categoria, filtro.Limite, filtro.Desplazamiento)...,
	)
	if err != nil {
		return 0, nil, errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	referencias := make([]string, 0, filtro.Limite)
	for filas.Next() {
		var referencia string
		if err := filas.Scan(&referencia); err != nil {
			return 0, nil, errorPostgreSQLPublico(ctx, err)
		}
		referencias = append(referencias, referencia)
	}
	if err := filas.Err(); err != nil {
		return 0, nil, errorPostgreSQLPublico(ctx, err)
	}
	return total, referencias, nil
}

func leerConteosCategorias(
	ctx context.Context,
	tx pgx.Tx,
	filtro puertosbolsa.FiltroConvocatoriasPublicas,
) (map[string]puertosbolsa.ConteoCategoriaConvocatorias, error) {
	filas, err := tx.Query(ctx, `
		SELECT categoria.categoria_clave,
		       count(*)::integer,
		       COALESCE(sum((
		           SELECT count(*)
		             FROM vec_bolsa_publica_lectura.plazos_convocatorias_publicas_v1 AS plazo_abierto
		            WHERE plazo_abierto.identificador_publico = convocatoria.identificador_publico
		              AND plazo_abierto.abre_en <= $5
		              AND $5 <= plazo_abierto.cierra_en
		       )), 0)::integer
		  FROM vec_bolsa_publica_lectura.convocatorias_publicadas_v1 AS convocatoria
		  JOIN vec_bolsa_publica_lectura.categorias_convocatorias_publicas_v1 AS categoria
		    ON categoria.identificador_publico = convocatoria.identificador_publico
		 WHERE `+predicadoFiltroConvocatorias+`
	 GROUP BY categoria.categoria_clave
	 ORDER BY categoria.categoria_clave
	 LIMIT 1025`,
		filtro.Texto, filtro.Tipo, filtro.Estado, filtro.SoloPlazoAbierto, filtro.Instante,
	)
	if err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	resultado := make(map[string]puertosbolsa.ConteoCategoriaConvocatorias)
	for filas.Next() {
		var clave string
		var conteo puertosbolsa.ConteoCategoriaConvocatorias
		if err := filas.Scan(&clave, &conteo.NumeroConvocatorias, &conteo.NumeroPlazosAbiertos); err != nil {
			return nil, errorPostgreSQLPublico(ctx, err)
		}
		resultado[clave] = conteo
	}
	if err := filas.Err(); err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	if len(resultado) > maximoCategoriasPorConvocatoria {
		return nil, ErrDatosPostgreSQLPublicosNoConfiables
	}
	return resultado, nil
}

func leerConvocatoria(
	ctx context.Context,
	tx pgx.Tx,
	identificador string,
) (dominiobolsa.Convocatoria, error) {
	convocatorias, err := leerConvocatorias(ctx, tx, []string{identificador})
	if err != nil {
		return dominiobolsa.Convocatoria{}, err
	}
	if len(convocatorias) != 1 {
		return dominiobolsa.Convocatoria{}, puertosbolsa.ErrConvocatoriaNoEncontrada
	}
	return convocatorias[0], nil
}

func leerConvocatorias(
	ctx context.Context,
	tx pgx.Tx,
	identificadores []string,
) ([]dominiobolsa.Convocatoria, error) {
	if len(identificadores) == 0 {
		return []dominiobolsa.Convocatoria{}, nil
	}
	indice := make(map[string]int, len(identificadores))
	for posicion, identificador := range identificadores {
		if _, duplicado := indice[identificador]; duplicado {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
		indice[identificador] = posicion
	}
	resultado := make([]dominiobolsa.Convocatoria, len(identificadores))
	presentes := make([]bool, len(identificadores))
	filas, err := tx.Query(ctx, `
		SELECT version_publica, estado, identificador_publico, tipo,
		       catalogo_categorias_id, catalogo_categorias_version,
		       catalogo_categorias_huella_sha256, huella_publica_sha256,
		       titulo, resumen, descripcion,
		       publicada_en, actualizada_en
		  FROM vec_bolsa_publica_lectura.convocatorias_publicadas_v1
		 WHERE identificador_publico = ANY($1::text[])
	 ORDER BY identificador_publico`, identificadores)
	if err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	for filas.Next() {
		var convocatoria dominiobolsa.Convocatoria
		datos := &dominiobolsa.DatosPublicosConvocatoria{}
		var estado string
		if err := filas.Scan(
			&convocatoria.Version, &estado, &datos.IdentificadorPublico,
			&datos.Tipo, &datos.CatalogoCategorias.CatalogoID,
			&datos.CatalogoCategorias.CatalogoVersion,
			&datos.CatalogoCategorias.CatalogoHuellaSHA256, &convocatoria.HuellaSHA256,
			&datos.Titulo, &datos.Resumen,
			&datos.Descripcion, &datos.PublicadaEn, &datos.ActualizadaEn,
		); err != nil {
			filas.Close()
			return nil, errorPostgreSQLPublico(ctx, err)
		}
		posicion, esperada := indice[datos.IdentificadorPublico]
		if !esperada || presentes[posicion] {
			filas.Close()
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
		convocatoria.Estado = dominiobolsa.EstadoConvocatoria(estado)
		datos.PublicadaEn = instanteUTC(datos.PublicadaEn)
		datos.ActualizadaEn = instanteUTC(datos.ActualizadaEn)
		convocatoria.DatosPublicos = datos
		resultado[posicion], presentes[posicion] = convocatoria, true
	}
	if err := filas.Err(); err != nil {
		filas.Close()
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	filas.Close()
	for _, presente := range presentes {
		if !presente {
			return nil, puertosbolsa.ErrConvocatoriaNoEncontrada
		}
	}
	if err := leerCategoriasConvocatoriasLote(ctx, tx, identificadores, resultado, indice); err != nil {
		return nil, err
	}
	if err := leerPlazosConvocatoriasLote(ctx, tx, identificadores, resultado, indice); err != nil {
		return nil, err
	}
	if err := leerRequisitosConvocatoriasLote(ctx, tx, identificadores, resultado, indice); err != nil {
		return nil, err
	}
	if err := leerDocumentosConvocatoriasLote(ctx, tx, identificadores, resultado, indice); err != nil {
		return nil, err
	}
	if err := leerAyudaConvocatoriasLote(ctx, tx, identificadores, resultado, indice); err != nil {
		return nil, err
	}
	for _, convocatoria := range resultado {
		huella, err := canonicopublico.HuellaConvocatoriaV1(convocatoria)
		if err != nil || !huellasIguales(huella, convocatoria.HuellaSHA256) {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
	}
	return resultado, nil
}

func leerCategoriasConvocatoriasLote(
	ctx context.Context, tx pgx.Tx, ids []string,
	convocatorias []dominiobolsa.Convocatoria, indice map[string]int,
) error {
	filas, err := tx.Query(ctx, `
		SELECT identificador_publico, categoria_clave
		  FROM vec_bolsa_publica_lectura.categorias_convocatorias_publicas_v1
		 WHERE identificador_publico = ANY($1::text[])
	 ORDER BY identificador_publico, categoria_clave
	 LIMIT $2`, ids, len(ids)*(maximoCategoriasPorConvocatoria+1))
	if err != nil {
		return errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	for filas.Next() {
		var id, clave string
		if err := filas.Scan(&id, &clave); err != nil {
			return errorPostgreSQLPublico(ctx, err)
		}
		posicion, existe := indice[id]
		if !existe {
			return ErrDatosPostgreSQLPublicosNoConfiables
		}
		datos := convocatorias[posicion].DatosPublicos
		datos.Categorias = append(datos.Categorias, clave)
		if len(datos.Categorias) > maximoCategoriasPorConvocatoria {
			return ErrDatosPostgreSQLPublicosNoConfiables
		}
	}
	return errorFilasPublicas(ctx, filas)
}

func leerPlazosConvocatoriasLote(
	ctx context.Context, tx pgx.Tx, ids []string,
	convocatorias []dominiobolsa.Convocatoria, indice map[string]int,
) error {
	filas, err := tx.Query(ctx, `
		SELECT identificador_publico, referencia, tipo, titulo, descripcion, abre_en, cierra_en
		  FROM vec_bolsa_publica_lectura.plazos_convocatorias_publicas_v1
		 WHERE identificador_publico = ANY($1::text[])
	 ORDER BY identificador_publico, referencia
	 LIMIT $2`, ids, len(ids)*(maximoPlazosPorConvocatoria+1))
	if err != nil {
		return errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	for filas.Next() {
		var id string
		var plazo dominiobolsa.PlazoConvocatoria
		if err := filas.Scan(&id, &plazo.Referencia, &plazo.Tipo, &plazo.Titulo, &plazo.Descripcion, &plazo.AbreEn, &plazo.CierraEn); err != nil {
			return errorPostgreSQLPublico(ctx, err)
		}
		posicion, existe := indice[id]
		if !existe {
			return ErrDatosPostgreSQLPublicosNoConfiables
		}
		plazo.AbreEn, plazo.CierraEn = instanteUTC(plazo.AbreEn), instanteUTC(plazo.CierraEn)
		datos := convocatorias[posicion].DatosPublicos
		datos.Plazos = append(datos.Plazos, plazo)
		if len(datos.Plazos) > maximoPlazosPorConvocatoria {
			return ErrDatosPostgreSQLPublicosNoConfiables
		}
	}
	return errorFilasPublicas(ctx, filas)
}

func leerRequisitosConvocatoriasLote(
	ctx context.Context, tx pgx.Tx, ids []string,
	convocatorias []dominiobolsa.Convocatoria, indice map[string]int,
) error {
	filas, err := tx.Query(ctx, `
		SELECT identificador_publico, referencia, orden, titulo, descripcion, obligatorio
		  FROM vec_bolsa_publica_lectura.requisitos_convocatorias_publicas_v1
		 WHERE identificador_publico = ANY($1::text[])
	 ORDER BY identificador_publico, orden, referencia
	 LIMIT $2`, ids, len(ids)*(maximoRequisitosPorConvocatoria+1))
	if err != nil {
		return errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	for filas.Next() {
		var id string
		var requisito dominiobolsa.RequisitoConvocatoria
		if err := filas.Scan(&id, &requisito.Referencia, &requisito.Orden, &requisito.Titulo, &requisito.Descripcion, &requisito.Obligatorio); err != nil {
			return errorPostgreSQLPublico(ctx, err)
		}
		posicion, existe := indice[id]
		if !existe {
			return ErrDatosPostgreSQLPublicosNoConfiables
		}
		datos := convocatorias[posicion].DatosPublicos
		datos.Requisitos = append(datos.Requisitos, requisito)
		if len(datos.Requisitos) > maximoRequisitosPorConvocatoria {
			return ErrDatosPostgreSQLPublicosNoConfiables
		}
	}
	return errorFilasPublicas(ctx, filas)
}

func leerDocumentosConvocatoriasLote(
	ctx context.Context, tx pgx.Tx, ids []string,
	convocatorias []dominiobolsa.Convocatoria, indice map[string]int,
) error {
	filas, err := tx.Query(ctx, `
		SELECT identificador_publico, referencia, tipo, orden, titulo, descripcion,
		       formato, url, publicado_en
		  FROM vec_bolsa_publica_lectura.documentos_convocatorias_publicas_v1
		 WHERE identificador_publico = ANY($1::text[])
	 ORDER BY identificador_publico, orden, referencia
	 LIMIT $2`, ids, len(ids)*(maximoDocumentosPorConvocatoria+1))
	if err != nil {
		return errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	for filas.Next() {
		var id string
		var documento dominiobolsa.DocumentoConvocatoria
		if err := filas.Scan(
			&id, &documento.Referencia, &documento.Tipo, &documento.Orden,
			&documento.Titulo, &documento.Descripcion, &documento.Formato,
			&documento.URL, &documento.PublicadoEn,
		); err != nil {
			return errorPostgreSQLPublico(ctx, err)
		}
		posicion, existe := indice[id]
		if !existe {
			return ErrDatosPostgreSQLPublicosNoConfiables
		}
		documento.PublicadoEn = instanteUTC(documento.PublicadoEn)
		datos := convocatorias[posicion].DatosPublicos
		datos.Documentos = append(datos.Documentos, documento)
		if len(datos.Documentos) > maximoDocumentosPorConvocatoria {
			return ErrDatosPostgreSQLPublicosNoConfiables
		}
	}
	return errorFilasPublicas(ctx, filas)
}

func leerAyudaConvocatoriasLote(
	ctx context.Context, tx pgx.Tx, ids []string,
	convocatorias []dominiobolsa.Convocatoria, indice map[string]int,
) error {
	filas, err := tx.Query(ctx, `
		SELECT identificador_publico, referencia, categoria, orden, pregunta, respuesta
		  FROM vec_bolsa_publica_lectura.ayuda_convocatorias_publicas_v1
		 WHERE identificador_publico = ANY($1::text[])
	 ORDER BY identificador_publico, orden, referencia
	 LIMIT $2`, ids, len(ids)*(maximoAyudasPorConvocatoria+1))
	if err != nil {
		return errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	for filas.Next() {
		var id string
		var ayuda dominiobolsa.AyudaConvocatoria
		if err := filas.Scan(&id, &ayuda.Referencia, &ayuda.Categoria, &ayuda.Orden, &ayuda.Pregunta, &ayuda.Respuesta); err != nil {
			return errorPostgreSQLPublico(ctx, err)
		}
		posicion, existe := indice[id]
		if !existe {
			return ErrDatosPostgreSQLPublicosNoConfiables
		}
		datos := convocatorias[posicion].DatosPublicos
		datos.Ayuda = append(datos.Ayuda, ayuda)
		if len(datos.Ayuda) > maximoAyudasPorConvocatoria {
			return ErrDatosPostgreSQLPublicosNoConfiables
		}
	}
	return errorFilasPublicas(ctx, filas)
}

func errorFilasPublicas(ctx context.Context, filas pgx.Rows) error {
	if err := filas.Err(); err != nil {
		return errorPostgreSQLPublico(ctx, err)
	}
	return nil
}

func leerCategoriasConvocatoria(ctx context.Context, tx pgx.Tx, identificador string) ([]string, error) {
	filas, err := tx.Query(ctx, `
		SELECT categoria_clave
		  FROM vec_bolsa_publica_lectura.categorias_convocatorias_publicas_v1
		 WHERE identificador_publico = $1
	 ORDER BY categoria_clave
	 LIMIT 1025`, identificador)
	if err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	resultado := make([]string, 0, 8)
	for filas.Next() {
		var clave string
		if err := filas.Scan(&clave); err != nil {
			return nil, errorPostgreSQLPublico(ctx, err)
		}
		resultado = append(resultado, clave)
	}
	if err := filas.Err(); err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	if len(resultado) > maximoCategoriasPorConvocatoria {
		return nil, ErrDatosPostgreSQLPublicosNoConfiables
	}
	return resultado, nil
}

func leerPlazosConvocatoria(ctx context.Context, tx pgx.Tx, identificador string) ([]dominiobolsa.PlazoConvocatoria, error) {
	filas, err := tx.Query(ctx, `
		SELECT referencia, tipo, titulo, descripcion, abre_en, cierra_en
		  FROM vec_bolsa_publica_lectura.plazos_convocatorias_publicas_v1
		 WHERE identificador_publico = $1
	 ORDER BY referencia
	 LIMIT 65`, identificador)
	if err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	resultado := make([]dominiobolsa.PlazoConvocatoria, 0, 8)
	for filas.Next() {
		var plazo dominiobolsa.PlazoConvocatoria
		if err := filas.Scan(&plazo.Referencia, &plazo.Tipo, &plazo.Titulo, &plazo.Descripcion, &plazo.AbreEn, &plazo.CierraEn); err != nil {
			return nil, errorPostgreSQLPublico(ctx, err)
		}
		plazo.AbreEn, plazo.CierraEn = instanteUTC(plazo.AbreEn), instanteUTC(plazo.CierraEn)
		resultado = append(resultado, plazo)
	}
	if err := filas.Err(); err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	if len(resultado) > maximoPlazosPorConvocatoria {
		return nil, ErrDatosPostgreSQLPublicosNoConfiables
	}
	return resultado, nil
}

func leerRequisitosConvocatoria(ctx context.Context, tx pgx.Tx, identificador string) ([]dominiobolsa.RequisitoConvocatoria, error) {
	filas, err := tx.Query(ctx, `
		SELECT referencia, orden, titulo, descripcion, obligatorio
		  FROM vec_bolsa_publica_lectura.requisitos_convocatorias_publicas_v1
		 WHERE identificador_publico = $1
	 ORDER BY orden, referencia
	 LIMIT 257`, identificador)
	if err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	resultado := make([]dominiobolsa.RequisitoConvocatoria, 0, 16)
	for filas.Next() {
		var requisito dominiobolsa.RequisitoConvocatoria
		if err := filas.Scan(&requisito.Referencia, &requisito.Orden, &requisito.Titulo, &requisito.Descripcion, &requisito.Obligatorio); err != nil {
			return nil, errorPostgreSQLPublico(ctx, err)
		}
		resultado = append(resultado, requisito)
	}
	if err := filas.Err(); err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	if len(resultado) > maximoRequisitosPorConvocatoria {
		return nil, ErrDatosPostgreSQLPublicosNoConfiables
	}
	return resultado, nil
}

func leerDocumentosConvocatoria(ctx context.Context, tx pgx.Tx, identificador string) ([]dominiobolsa.DocumentoConvocatoria, error) {
	filas, err := tx.Query(ctx, `
		SELECT referencia, tipo, orden, titulo, descripcion, formato, url, publicado_en
		  FROM vec_bolsa_publica_lectura.documentos_convocatorias_publicas_v1
		 WHERE identificador_publico = $1
	 ORDER BY orden, referencia
	 LIMIT 257`, identificador)
	if err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	resultado := make([]dominiobolsa.DocumentoConvocatoria, 0, 16)
	for filas.Next() {
		var documento dominiobolsa.DocumentoConvocatoria
		if err := filas.Scan(
			&documento.Referencia, &documento.Tipo, &documento.Orden, &documento.Titulo,
			&documento.Descripcion, &documento.Formato, &documento.URL, &documento.PublicadoEn,
		); err != nil {
			return nil, errorPostgreSQLPublico(ctx, err)
		}
		documento.PublicadoEn = instanteUTC(documento.PublicadoEn)
		resultado = append(resultado, documento)
	}
	if err := filas.Err(); err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	if len(resultado) > maximoDocumentosPorConvocatoria {
		return nil, ErrDatosPostgreSQLPublicosNoConfiables
	}
	return resultado, nil
}

func leerAyudaConvocatoria(ctx context.Context, tx pgx.Tx, identificador string) ([]dominiobolsa.AyudaConvocatoria, error) {
	filas, err := tx.Query(ctx, `
		SELECT referencia, categoria, orden, pregunta, respuesta
		  FROM vec_bolsa_publica_lectura.ayuda_convocatorias_publicas_v1
		 WHERE identificador_publico = $1
	 ORDER BY orden, referencia
	 LIMIT 129`, identificador)
	if err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	resultado := make([]dominiobolsa.AyudaConvocatoria, 0, 16)
	for filas.Next() {
		var ayuda dominiobolsa.AyudaConvocatoria
		if err := filas.Scan(&ayuda.Referencia, &ayuda.Categoria, &ayuda.Orden, &ayuda.Pregunta, &ayuda.Respuesta); err != nil {
			return nil, errorPostgreSQLPublico(ctx, err)
		}
		resultado = append(resultado, ayuda)
	}
	if err := filas.Err(); err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	if len(resultado) > maximoAyudasPorConvocatoria {
		return nil, ErrDatosPostgreSQLPublicosNoConfiables
	}
	return resultado, nil
}
