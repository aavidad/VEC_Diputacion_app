package postgresimportacionconvoca

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	aplicacion "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

const (
	funcionGuardarLoteV1     = "vec_bolsa_importacion_convoca.guardar_lote_v1"
	funcionConsultarEstadoV1 = "vec_bolsa_importacion_convoca.consultar_estado_v1"
	funcionRecuperarPaginaV1 = "vec_bolsa_importacion_convoca.recuperar_lote_pagina_v1"
	maximoReintentos         = 4
	maximoFilasPagina        = 512
)

type iniciadorTransacciones interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

var (
	_ aplicacion.RepositorioImportaciones      = (*RepositorioPostgreSQL)(nil)
	_ aplicacion.ConsultaImportacionesDurables = (*RepositorioRecuperacionPostgreSQL)(nil)
)

// RepositorioPostgreSQL usa una identidad dedicada de importación. No puede
// conciliar ni ejecutar retención y nunca recibe acceso directo a tablas.
type RepositorioPostgreSQL struct {
	pool      iniciadorTransacciones
	protector ProtectorStagingConvoca
}

func NuevoRepositorioPostgreSQL(
	pool *pgxpool.Pool,
	protector ProtectorStagingConvoca,
) (*RepositorioPostgreSQL, error) {
	return nuevoRepositorioPostgreSQL(pool, protector)
}

func nuevoRepositorioPostgreSQL(
	pool iniciadorTransacciones,
	protector ProtectorStagingConvoca,
) (*RepositorioPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, ErrRepositorioNoDisponible
	}
	if valorNulo(protector) {
		return nil, ErrProtectorRequerido
	}
	return &RepositorioPostgreSQL{pool: pool, protector: protector}, nil
}

// RepositorioRecuperacionPostgreSQL exige una identidad VEC/T13 separada de
// la importadora. El rol de importación no recibe EXECUTE sobre recuperación.
type RepositorioRecuperacionPostgreSQL struct {
	pool      iniciadorTransacciones
	protector ProtectorStagingConvoca
}

func NuevoRepositorioRecuperacionPostgreSQL(
	pool *pgxpool.Pool,
	protector ProtectorStagingConvoca,
) (*RepositorioRecuperacionPostgreSQL, error) {
	return nuevoRepositorioRecuperacionPostgreSQL(pool, protector)
}

func nuevoRepositorioRecuperacionPostgreSQL(
	pool iniciadorTransacciones,
	protector ProtectorStagingConvoca,
) (*RepositorioRecuperacionPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, ErrRepositorioNoDisponible
	}
	if valorNulo(protector) {
		return nil, ErrProtectorRequerido
	}
	return &RepositorioRecuperacionPostgreSQL{pool: pool, protector: protector}, nil
}

func (r *RepositorioPostgreSQL) GuardarSiAusente(
	ctx context.Context,
	lote dominio.LoteValidado,
) (dominio.ActaImportacion, bool, error) {
	if ctx == nil || r == nil || valorNulo(r.pool) || valorNulo(r.protector) {
		return dominio.ActaImportacion{}, false, ErrRepositorioNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return dominio.ActaImportacion{}, false, err
	}
	if lote.Validar() != nil {
		return dominio.ActaImportacion{}, false, ErrLoteNoConfiable
	}
	protegido, err := r.protector.ProtegerStaging(ctx, SolicitudProteccionStaging{
		ImportacionRef:      lote.Acta.ImportacionRef,
		HuellaFicheroSHA256: lote.Acta.HuellaFicheroSHA256,
		Esquema:             lote.Acta.Esquema, Filas: clonarFilasDominio(lote.Aceptadas),
	})
	if err != nil {
		borrarFilasProtegidas(protegido.Filas)
		if ctx.Err() != nil {
			return dominio.ActaImportacion{}, false, ctx.Err()
		}
		return dominio.ActaImportacion{}, false, ErrProteccionNoDisponible
	}
	filasOriginales := protegido.Filas
	protegido.Filas = clonarFilasProtegidas(filasOriginales)
	borrarFilasProtegidas(filasOriginales)
	defer borrarFilasProtegidas(protegido.Filas)
	if validarCorrespondenciaProteccion(lote.Aceptadas, protegido.Filas) != nil {
		return dominio.ActaImportacion{}, false, ErrMaterialNoConfiable
	}
	actaJSON, err := serializarActa(lote.Acta)
	if err != nil {
		return dominio.ActaImportacion{}, false, err
	}
	filasJSON, err := serializarFilasProtegidas(protegido.Filas)
	if err != nil {
		return dominio.ActaImportacion{}, false, ErrMaterialNoConfiable
	}
	defer borrarBytes(actaJSON, filasJSON)

	var ultimo error
	for intento := 0; intento < maximoReintentos; intento++ {
		acta, reutilizada, err := r.guardarUnaVez(ctx, actaJSON, filasJSON)
		if err == nil {
			return acta, reutilizada, nil
		}
		ultimo = err
		if ctx.Err() != nil {
			return dominio.ActaImportacion{}, false, ctx.Err()
		}
		if !esReintentable(err) && !errors.Is(err, ErrRepositorioNoDisponible) {
			return dominio.ActaImportacion{}, false, errorPostgreSQL(ctx, err)
		}
	}
	return dominio.ActaImportacion{}, false, errorPostgreSQL(ctx, ultimo)
}

func (r *RepositorioPostgreSQL) guardarUnaVez(
	ctx context.Context,
	actaJSON []byte,
	filasJSON []byte,
) (dominio.ActaImportacion, bool, error) {
	tx, err := iniciarTransaccion(ctx, r.pool, pgx.ReadWrite)
	if err != nil {
		return dominio.ActaImportacion{}, false, err
	}
	defer revertir(tx)
	var actaGuardadaJSON []byte
	var reutilizada bool
	err = tx.QueryRow(ctx, `
		SELECT acta_canonica, reutilizada
		  FROM `+funcionGuardarLoteV1+`($1::jsonb, $2::jsonb)`,
		json.RawMessage(actaJSON), json.RawMessage(filasJSON),
	).Scan(&actaGuardadaJSON, &reutilizada)
	if err != nil {
		return dominio.ActaImportacion{}, false, err
	}
	defer borrarBytes(actaGuardadaJSON)
	acta, err := deserializarActa(actaGuardadaJSON)
	if err != nil {
		return dominio.ActaImportacion{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return dominio.ActaImportacion{}, false, err
	}
	return acta, reutilizada, nil
}

func (r *RepositorioRecuperacionPostgreSQL) ConsultarEstado(
	ctx context.Context,
	huella string,
) (aplicacion.EstadoImportacion, bool, error) {
	contenido, existe, err := r.consultarJSON(ctx, funcionConsultarEstadoV1, huella)
	if err != nil || !existe {
		return aplicacion.EstadoImportacion{}, existe, err
	}
	defer borrarBytes(contenido)
	var datos estadoPostgreSQL
	if decodificarJSONExacto(contenido, &datos) != nil {
		return aplicacion.EstadoImportacion{}, false, ErrResultadoNoConfiable
	}
	estado, err := restaurarEstado(datos)
	if err != nil || estado.Acta.HuellaFicheroSHA256 != huella {
		return aplicacion.EstadoImportacion{}, false, ErrResultadoNoConfiable
	}
	return estado, true, nil
}

func (r *RepositorioRecuperacionPostgreSQL) RecuperarLote(
	ctx context.Context,
	huella string,
) (dominio.LoteValidado, aplicacion.EstadoImportacion, bool, error) {
	if ctx == nil || r == nil || valorNulo(r.pool) || valorNulo(r.protector) {
		return dominio.LoteValidado{}, aplicacion.EstadoImportacion{}, false,
			ErrRepositorioNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return dominio.LoteValidado{}, aplicacion.EstadoImportacion{}, false, err
	}
	tx, err := iniciarTransaccion(ctx, r.pool, pgx.ReadOnly)
	if err != nil {
		return dominio.LoteValidado{}, aplicacion.EstadoImportacion{}, false, err
	}
	defer revertir(tx)
	var estadoJSON []byte
	if err := tx.QueryRow(
		ctx, `SELECT `+funcionConsultarEstadoV1+`($1::text)`, huella,
	).Scan(&estadoJSON); err != nil {
		return dominio.LoteValidado{}, aplicacion.EstadoImportacion{}, false,
			errorPostgreSQL(ctx, err)
	}
	if len(estadoJSON) == 0 {
		return dominio.LoteValidado{}, aplicacion.EstadoImportacion{}, false, nil
	}
	defer borrarBytes(estadoJSON)
	var estadoDatos estadoPostgreSQL
	if decodificarJSONExacto(estadoJSON, &estadoDatos) != nil {
		return dominio.LoteValidado{}, aplicacion.EstadoImportacion{}, false,
			ErrResultadoNoConfiable
	}
	estado, err := restaurarEstado(estadoDatos)
	if err != nil || estado.Acta.HuellaFicheroSHA256 != huella {
		return dominio.LoteValidado{}, aplicacion.EstadoImportacion{}, false, ErrResultadoNoConfiable
	}
	if estado.EstadoStaging == aplicacion.EstadoStagingExpurgado {
		return dominio.LoteValidado{}, estado, true, aplicacion.ErrStagingExpurgado
	}
	protegidas := make([]FilaStagingProtegida, 0, estado.Acta.FilasAceptadas)
	defer func() {
		borrarFilasProtegidas(protegidas)
	}()
	desde := 2
	numeroAnterior := 1
	totalBytes := 0
	for {
		var paginaJSON []byte
		err := tx.QueryRow(ctx, `
			SELECT `+funcionRecuperarPaginaV1+`(
			    $1::text, $2::integer, $3::integer
			)`, huella, desde, maximoFilasPagina,
		).Scan(&paginaJSON)
		if err != nil {
			return dominio.LoteValidado{}, aplicacion.EstadoImportacion{}, true,
				errorPostgreSQL(ctx, err)
		}
		if len(paginaJSON) == 0 {
			return dominio.LoteValidado{}, aplicacion.EstadoImportacion{}, false,
				ErrResultadoNoConfiable
		}
		var pagina loteRecuperadoPostgreSQL
		if decodificarJSONExacto(paginaJSON, &pagina) != nil ||
			!reflect.DeepEqual(pagina.Estado, estadoDatos) {
			borrarBytes(paginaJSON)
			return dominio.LoteValidado{}, aplicacion.EstadoImportacion{}, false,
				ErrResultadoNoConfiable
		}
		borrarBytes(paginaJSON)
		filasPagina, err := restaurarFilasProtegidas(pagina.Filas)
		if err != nil || validarFilasProtegidas(filasPagina) != nil ||
			len(filasPagina) > estado.Acta.FilasAceptadas-len(protegidas) {
			borrarFilasProtegidas(filasPagina)
			return dominio.LoteValidado{}, aplicacion.EstadoImportacion{}, false,
				ErrResultadoNoConfiable
		}
		for i := range filasPagina {
			bytesFila := len(filasPagina[i].Nonce) +
				len(filasPagina[i].ContenidoCifrado) +
				len(filasPagina[i].DerivacionDocumentoHMACSHA256) +
				len(filasPagina[i].AtestacionFilaHMACSHA256)
			if filasPagina[i].Numero <= numeroAnterior ||
				bytesFila > maximoBytesProtegidosLote-totalBytes {
				borrarFilasProtegidas(filasPagina)
				return dominio.LoteValidado{}, aplicacion.EstadoImportacion{},
					false, ErrResultadoNoConfiable
			}
			numeroAnterior = filasPagina[i].Numero
			totalBytes += bytesFila
		}
		protegidas = append(protegidas, filasPagina...)
		if pagina.SiguienteNumero == nil {
			break
		}
		if len(filasPagina) == 0 ||
			*pagina.SiguienteNumero <= numeroAnterior {
			return dominio.LoteValidado{}, aplicacion.EstadoImportacion{}, false,
				ErrResultadoNoConfiable
		}
		desde = *pagina.SiguienteNumero
	}
	if len(protegidas) != estado.Acta.FilasAceptadas {
		return dominio.LoteValidado{}, aplicacion.EstadoImportacion{}, false,
			ErrResultadoNoConfiable
	}
	if err := tx.Commit(ctx); err != nil {
		return dominio.LoteValidado{}, aplicacion.EstadoImportacion{}, false,
			errorPostgreSQL(ctx, err)
	}
	filasParaProtector := clonarFilasProtegidas(protegidas)
	recuperadas, err := r.protector.RecuperarStaging(ctx, SolicitudRecuperacionStaging{
		ImportacionRef:      estado.Acta.ImportacionRef,
		HuellaFicheroSHA256: estado.Acta.HuellaFicheroSHA256,
		Esquema:             estado.Acta.Esquema,
		Filas:               filasParaProtector,
	})
	borrarFilasProtegidas(filasParaProtector)
	if err != nil {
		if ctx.Err() != nil {
			return dominio.LoteValidado{}, aplicacion.EstadoImportacion{}, false, ctx.Err()
		}
		if errors.Is(err, ErrMaterialNoConfiable) {
			return dominio.LoteValidado{}, aplicacion.EstadoImportacion{}, false,
				ErrMaterialNoConfiable
		}
		return dominio.LoteValidado{}, aplicacion.EstadoImportacion{}, false, ErrProteccionNoDisponible
	}
	recuperadas = clonarFilasDominio(recuperadas)
	if validarCorrespondenciaRecuperada(protegidas, recuperadas, estado.Acta.Esquema) != nil {
		return dominio.LoteValidado{}, aplicacion.EstadoImportacion{}, false, ErrMaterialNoConfiable
	}
	lote := dominio.LoteValidado{Acta: estado.Acta, Aceptadas: recuperadas}
	if lote.Validar() != nil {
		return dominio.LoteValidado{}, aplicacion.EstadoImportacion{}, false, ErrResultadoNoConfiable
	}
	return lote, estado, true, nil
}

func (r *RepositorioRecuperacionPostgreSQL) consultarJSON(
	ctx context.Context,
	funcion string,
	huella string,
) ([]byte, bool, error) {
	if ctx == nil || r == nil || valorNulo(r.pool) || valorNulo(r.protector) {
		return nil, false, ErrRepositorioNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	tx, err := iniciarTransaccion(ctx, r.pool, pgx.ReadOnly)
	if err != nil {
		return nil, false, err
	}
	defer revertir(tx)
	var contenido []byte
	err = tx.QueryRow(ctx, `SELECT `+funcion+`($1::text)`, huella).Scan(&contenido)
	if err != nil {
		return nil, false, errorPostgreSQL(ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, errorPostgreSQL(ctx, err)
	}
	if len(contenido) == 0 {
		return nil, false, nil
	}
	return contenido, true, nil
}

func borrarFilasProtegidas(filas []FilaStagingProtegida) {
	for i := range filas {
		borrarBytes(filas[i].Nonce, filas[i].ContenidoCifrado,
			filas[i].DerivacionDocumentoHMACSHA256,
			filas[i].AtestacionFilaHMACSHA256)
	}
}

func borrarBytes(valores ...[]byte) {
	for _, valor := range valores {
		for i := range valor {
			valor[i] = 0
		}
	}
}
