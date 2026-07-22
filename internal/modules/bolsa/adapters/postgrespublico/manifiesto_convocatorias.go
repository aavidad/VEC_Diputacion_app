package postgrespublico

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"

	canonicopublico "vec-diputacion-granada/internal/modules/bolsa/publico/canonico"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/publico/dominio"
)

const (
	maximoFilasManifiestoArranque = 12_000
	// El techo abarca el material JSON recibido y los escalares de las 12.000
	// convocatorias. Se procesa una fila cada vez y solo se conservan 3 hashes
	// cortos por convocatoria. Rebasarlo exige publicar una nueva version del
	// contrato y revisar memoria/tiempo de arranque.
	maximoBytesManifiestoArranque    = 256 << 20
	duracionMaximaManifiestoArranque = 60 * time.Second
)

const consultaEntradasManifiestoEnFlujo = `
SELECT convocatoria.version_publica, convocatoria.estado,
       convocatoria.identificador_publico, convocatoria.tipo,
       convocatoria.catalogo_categorias_id,
       convocatoria.catalogo_categorias_version,
       convocatoria.catalogo_categorias_huella_sha256,
       convocatoria.catalogo_categorias_huella_proyeccion_sha256,
       convocatoria.huella_publica_sha256,
       convocatoria.huella_resumen_publico_sha256,
       convocatoria.titulo, convocatoria.resumen, convocatoria.descripcion,
       convocatoria.publicada_en, convocatoria.actualizada_en,
       COALESCE((
           SELECT jsonb_agg(categoria.categoria_clave ORDER BY categoria.categoria_clave)
             FROM vec_bolsa_publica_lectura.categorias_convocatorias_publicas_v2 AS categoria
            WHERE categoria.identificador_publico = convocatoria.identificador_publico
       ), '[]'::jsonb),
       COALESCE((
           SELECT jsonb_agg(jsonb_build_object(
               'referencia', plazo.referencia, 'tipo', plazo.tipo,
               'titulo', plazo.titulo, 'descripcion', plazo.descripcion,
               'abre_en', plazo.abre_en, 'cierra_en', plazo.cierra_en
           ) ORDER BY plazo.referencia)
             FROM vec_bolsa_publica_lectura.plazos_convocatorias_publicas_v2 AS plazo
            WHERE plazo.identificador_publico = convocatoria.identificador_publico
       ), '[]'::jsonb),
       COALESCE((
           SELECT jsonb_agg(jsonb_build_object(
               'referencia', requisito.referencia, 'orden', requisito.orden,
               'titulo', requisito.titulo, 'descripcion', requisito.descripcion,
               'obligatorio', requisito.obligatorio
           ) ORDER BY requisito.orden, requisito.referencia)
             FROM vec_bolsa_publica_lectura.requisitos_convocatorias_publicas_v2 AS requisito
            WHERE requisito.identificador_publico = convocatoria.identificador_publico
       ), '[]'::jsonb),
       COALESCE((
           SELECT jsonb_agg(jsonb_build_object(
               'referencia', documento.referencia, 'tipo', documento.tipo,
               'orden', documento.orden, 'titulo', documento.titulo,
               'descripcion', documento.descripcion, 'formato', documento.formato,
               'url', documento.url, 'publicado_en', documento.publicado_en
           ) ORDER BY documento.orden, documento.referencia)
             FROM vec_bolsa_publica_lectura.documentos_convocatorias_publicas_v2 AS documento
            WHERE documento.identificador_publico = convocatoria.identificador_publico
       ), '[]'::jsonb),
       COALESCE((
           SELECT jsonb_agg(jsonb_build_object(
               'referencia', ayuda.referencia, 'categoria', ayuda.categoria,
               'orden', ayuda.orden, 'pregunta', ayuda.pregunta,
               'respuesta', ayuda.respuesta
           ) ORDER BY ayuda.orden, ayuda.referencia)
             FROM vec_bolsa_publica_lectura.ayuda_convocatorias_publicas_v2 AS ayuda
            WHERE ayuda.identificador_publico = convocatoria.identificador_publico
       ), '[]'::jsonb)
  FROM vec_bolsa_publica_lectura.convocatorias_publicadas_v2 AS convocatoria
 ORDER BY convocatoria.identificador_publico
 LIMIT 12001`

func leerEntradasManifiestoEnFlujo(
	ctx context.Context,
	tx pgx.Tx,
) ([]canonicopublico.ConvocatoriaManifiestoPublicoV2, error) {
	filas, err := tx.Query(ctx, consultaEntradasManifiestoEnFlujo)
	if err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	resultado := make([]canonicopublico.ConvocatoriaManifiestoPublicoV2, 0, 256)
	bytesLeidos := 0
	for filas.Next() {
		convocatoria, huellaResumenAlmacenada, bytesFila, err := escanearConvocatoriaManifiesto(filas)
		if err != nil {
			return nil, errorPostgreSQLPublico(ctx, err)
		}
		bytesLeidos += bytesFila
		if len(resultado) >= maximoFilasManifiestoArranque || bytesLeidos > maximoBytesManifiestoArranque {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
		completa, err := canonicopublico.HuellaConvocatoriaV2(convocatoria)
		if err != nil || !huellasIguales(completa, convocatoria.HuellaSHA256) {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
		resumen, err := canonicopublico.ResumenDesdeConvocatoriaV2(convocatoria)
		if err != nil {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
		huellaResumen, err := canonicopublico.HuellaResumenConvocatoriaV2(resumen)
		if err != nil || !huellasIguales(huellaResumen, huellaResumenAlmacenada) {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
		resultado = append(resultado, canonicopublico.ConvocatoriaManifiestoPublicoV2{
			IdentificadorPublico: convocatoria.DatosPublicos.IdentificadorPublico,
			HuellaCompletaSHA256: completa, HuellaResumenSHA256: huellaResumen,
		})
	}
	if err := filas.Err(); err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	return resultado, nil
}

func escanearConvocatoriaManifiesto(
	fila pgx.Row,
) (dominiobolsa.Convocatoria, string, int, error) {
	var convocatoria dominiobolsa.Convocatoria
	datos := &dominiobolsa.DatosPublicosConvocatoria{}
	var estado, huellaResumen string
	var categorias, plazos, requisitos, documentos, ayuda []byte
	err := fila.Scan(
		&convocatoria.Version, &estado, &datos.IdentificadorPublico, &datos.Tipo,
		&datos.CatalogoCategorias.CatalogoID, &datos.CatalogoCategorias.CatalogoVersion,
		&datos.CatalogoCategorias.CatalogoHuellaSHA256,
		&datos.CatalogoCategorias.CatalogoHuellaProyeccionSHA256,
		&convocatoria.HuellaSHA256,
		&huellaResumen, &datos.Titulo, &datos.Resumen, &datos.Descripcion,
		&datos.PublicadaEn, &datos.ActualizadaEn,
		&categorias, &plazos, &requisitos, &documentos, &ayuda,
	)
	if err != nil {
		return dominiobolsa.Convocatoria{}, "", 0, err
	}
	convocatoria.Estado = dominiobolsa.EstadoConvocatoria(estado)
	datos.PublicadaEn, datos.ActualizadaEn = instanteUTC(datos.PublicadaEn), instanteUTC(datos.ActualizadaEn)
	if json.Unmarshal(categorias, &datos.Categorias) != nil ||
		json.Unmarshal(plazos, &datos.Plazos) != nil ||
		json.Unmarshal(requisitos, &datos.Requisitos) != nil ||
		json.Unmarshal(documentos, &datos.Documentos) != nil ||
		json.Unmarshal(ayuda, &datos.Ayuda) != nil {
		return dominiobolsa.Convocatoria{}, "", 0, ErrDatosPostgreSQLPublicosNoConfiables
	}
	normalizarInstantesHijos(datos)
	convocatoria.DatosPublicos = datos
	bytesEscalares := len(convocatoria.Version) + len(estado) + len(datos.IdentificadorPublico) +
		len(datos.Tipo) + len(datos.CatalogoCategorias.CatalogoID) +
		len(datos.CatalogoCategorias.CatalogoHuellaSHA256) +
		len(datos.CatalogoCategorias.CatalogoHuellaProyeccionSHA256) + len(convocatoria.HuellaSHA256) +
		len(huellaResumen) + len(datos.Titulo) + len(datos.Resumen) + len(datos.Descripcion) + 32
	return convocatoria, huellaResumen, bytesEscalares + len(categorias) + len(plazos) +
		len(requisitos) + len(documentos) + len(ayuda), nil
}

func normalizarInstantesHijos(datos *dominiobolsa.DatosPublicosConvocatoria) {
	for indice := range datos.Plazos {
		datos.Plazos[indice].AbreEn = instanteUTC(datos.Plazos[indice].AbreEn)
		datos.Plazos[indice].CierraEn = instanteUTC(datos.Plazos[indice].CierraEn)
	}
	for indice := range datos.Documentos {
		datos.Documentos[indice].PublicadoEn = instanteUTC(datos.Documentos[indice].PublicadoEn)
	}
}
