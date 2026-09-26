package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type entradaMotivoPostgreSQLContratacionTemporalDesarrollo struct {
	Clave        string  `json:"clave"`
	VigenteDesde string  `json:"vigente_desde"`
	VigenteHasta *string `json:"vigente_hasta"`
}

func publicarMotivosPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	soporte *soporteAltaContratacionTemporalDesarrollo,
) error {
	motivos := []dominiovec.ReferenciaEntradaCatalogo{
		soporte.motivo,
		soporte.motivoRegistroAnalisis,
		soporte.motivoRectificacionAnalisis,
		soporte.motivoPropuestaCobertura,
		soporte.motivoDecisionCobertura,
		soporte.motivoRectificacionCobertura,
		soporte.motivoResultadoCobertura,
		soporte.motivoAsignacion,
		soporte.motivoInformeJuridico,
	}
	porCatalogo := make(map[string][]dominiovec.ReferenciaEntradaCatalogo)
	for _, motivo := range motivos {
		if !dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivo) {
			return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
		}
		clave := motivo.CatalogoID + "\x00" + motivo.CatalogoHuellaSHA256
		porCatalogo[clave] = append(porCatalogo[clave], motivo)
	}
	claves := make([]string, 0, len(porCatalogo))
	for clave := range porCatalogo {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(
		soporte.reloj.Ahora(),
	)
	if !vigente {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	for _, clave := range claves {
		if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(
			ctx, pool, porCatalogo[clave], desde,
		); err != nil {
			return err
		}
	}
	return nil
}

type entradaMotivoCoberturaPostgreSQLDesarrollo struct {
	Clave        string  `json:"clave"`
	ClaveI18n    string  `json:"clave_i18n"`
	VigenteDesde string  `json:"vigente_desde"`
	VigenteHasta *string `json:"vigente_hasta"`
}

func publicarMotivoEleccionProcedimientoRRHHPostgreSQL(ctx context.Context, pool *pgxpool.Pool, ahora time.Time) error {
	motivo := motivoEleccionProcedimientoRRHHDesarrollo()
	if ctx == nil || pool == nil || motivo.Validar() != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	_ = ahora
	publicadoEn := instantePublicacionMotivoEleccionProcedimientoRRHHDesarrollo()
	contenido, err := json.Marshal([]entradaMotivoCoberturaPostgreSQLDesarrollo{{
		Clave: motivo.EntradaClave, ClaveI18n: "contratacion_temporal.cobertura.motivo.eleccion_procedimiento_rrhh",
		VigenteDesde: publicadoEn.Format("2006-01-02T15:04:05.000000Z"),
	}})
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE `+rolPropietarioAutorizacionContratacionTemporalDesarrollo); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	var secuencia int64
	var huellaExistente string
	err = tx.QueryRow(ctx, `SELECT secuencia_origen, catalogo_huella_publicada_sha256 FROM vec_autorizacion.motivo_cobertura_v1_catalogo_publicado WHERE catalogo_id=$1 AND catalogo_version=$2`, motivo.CatalogoID, motivo.CatalogoVersion).Scan(&secuencia, &huellaExistente)
	if err == nil && huellaExistente != motivo.CatalogoHuellaSHA256 {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `SELECT ultima_secuencia+1 FROM vec_autorizacion.motivo_cobertura_v1_checkpoint_origen WHERE control_id FOR UPDATE`).Scan(&secuencia)
	}
	if err != nil || tx.Commit(ctx) != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	tx, err = pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE `+rolProyectorMotivosContratacionTemporalDesarrollo); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	var publicada bool
	err = tx.QueryRow(ctx, `SELECT vec_autorizacion.publicar_motivos_cobertura_v1($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb)`,
		referenciaAltaContratacionTemporalDesarrollo("evento_", "motivo-cobertura-eleccion-procedimiento-rrhh"), secuencia,
		huellaAltaContratacionTemporalDesarrollo("motivo-cobertura-eleccion-procedimiento-rrhh"), motivo.CatalogoID, motivo.CatalogoVersion,
		motivo.CatalogoHuellaSHA256, moduloMotivosDecisionCoberturaDesarrollo, publicadoEn, contenido).Scan(&publicada)
	if err != nil || !publicada || tx.Commit(ctx) != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return nil
}

func instantePublicacionMotivoEleccionProcedimientoRRHHDesarrollo() time.Time {
	// Es parte de la preimagen idempotente: no depende del instante de arranque.
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}

func motivoEleccionProcedimientoRRHHDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           catalogoMotivosDecisionCoberturaDesarrollo,
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("motivos-cobertura-eleccion-procedimiento-rrhh-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "eleccion-procedimiento-rrhh"),
	}
}

func publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	motivos []dominiovec.ReferenciaEntradaCatalogo,
	publicadoEn time.Time,
) error {
	diagnosticar := func(etapa string, causa error) {
		codigo := "sin_codigo_pg"
		var pg *pgconn.PgError
		if errors.As(causa, &pg) && len(pg.Code) == 5 {
			valido := true
			for _, c := range pg.Code {
				if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
					valido = false
				}
			}
			if valido {
				codigo = pg.Code
			}
		}
		log.Printf("contratacion temporal: motivo no disponible; etapa=%s sqlstate=%s", etapa, codigo)
	}
	if len(motivos) == 0 {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	sort.Slice(motivos, func(i, j int) bool {
		return motivos[i].EntradaClave < motivos[j].EntradaClave
	})
	primero := motivos[0]
	for _, motivo := range motivos {
		if motivo.CatalogoID != primero.CatalogoID ||
			motivo.CatalogoVersion != primero.CatalogoVersion ||
			motivo.CatalogoHuellaSHA256 != primero.CatalogoHuellaSHA256 {
			return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
		}
	}
	eventoRef := referenciaAltaContratacionTemporalDesarrollo(
		"evento_", "catalogo-motivos\x00"+primero.CatalogoID,
	)
	huellaEvento := huellaAltaContratacionTemporalDesarrollo(
		"catalogo-motivos\x00" + primero.CatalogoID,
	)
	var secuencia int64
	txConsulta, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		diagnosticar("consulta_transaccion", err)
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	defer txConsulta.Rollback(context.Background())
	if _, err = txConsulta.Exec(ctx, `SET LOCAL ROLE `+
		rolPropietarioAutorizacionContratacionTemporalDesarrollo); err != nil {
		diagnosticar("consulta_rol", err)
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	// Un arranque repetido reutiliza la publicación existente: con la misma
	// huella de catálogo se repite con el publicado_en ya registrado, de modo
	// que el replay de la función SQL (que exige contenido idéntico) sigue
	// comprobando las entradas y ningún instante distinto del arranque previo
	// rompe los siguientes. Otra huella para la misma versión se rechaza.
	var huellaExistente string
	var publicadoExistente time.Time
	err = txConsulta.QueryRow(ctx, `
		SELECT secuencia_origen, catalogo_huella_publicada_sha256, publicado_en
		  FROM vec_autorizacion.motivo_v2_catalogo_publicado
		 WHERE catalogo_id=$1 AND catalogo_version=$2`,
		primero.CatalogoID, primero.CatalogoVersion,
	).Scan(&secuencia, &huellaExistente, &publicadoExistente)
	if err == nil {
		if huellaExistente != primero.CatalogoHuellaSHA256 {
			diagnosticar("consulta_huella", nil)
			return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
		}
		publicadoEn = publicadoExistente
	}
	if errors.Is(err, pgx.ErrNoRows) {
		err = txConsulta.QueryRow(ctx, `
			SELECT ultima_secuencia+1
			  FROM vec_autorizacion.motivo_v2_checkpoint_origen
			 WHERE control_id=true FOR UPDATE`).Scan(&secuencia)
	}
	if err != nil {
		diagnosticar("consulta_secuencia", err)
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if err = txConsulta.Commit(ctx); err != nil {
		diagnosticar("consulta_commit", err)
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	contenido, err := contenidoCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(motivos, publicadoEn)
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		diagnosticar("publicacion_transaccion", err)
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE `+
		rolProyectorMotivosContratacionTemporalDesarrollo); err != nil {
		diagnosticar("publicacion_rol", err)
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	var publicada bool
	err = tx.QueryRow(ctx, `
		SELECT vec_autorizacion.publicar_motivos_autorizacion_v2(
		 $1,$2,$3,$4,$5,$6,$7,$8::jsonb)`,
		eventoRef, secuencia, huellaEvento, primero.CatalogoID,
		primero.CatalogoVersion, primero.CatalogoHuellaSHA256,
		publicadoEn, contenido,
	).Scan(&publicada)
	if err == nil && !publicada {
		// La función SQL rechaza sin error un replay cuyo contenido no es el
		// ya publicado (p. ej., entradas añadidas a una versión existente).
		// Se nombra el catálogo en el error de arranque; no es dato sensible.
		diagnosticar("publicacion_rechazada", nil)
		return errors.Join(errPostgreSQLContratacionTemporalDesarrolloNoDisponible,
			fmt.Errorf("catalogo de motivos %s v%d rechazado: su contenido difiere del ya publicado en esa version",
				primero.CatalogoID, primero.CatalogoVersion))
	}
	if err != nil {
		diagnosticar("publicacion_ejecucion", err)
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		diagnosticar("publicacion_commit", err)
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return nil
}

// contenidoCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo serializa
// las entradas con la vigencia del instante de publicación. Es determinista:
// la misma entrada produce los mismos bytes, que es lo que el replay exige.
func contenidoCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(
	motivos []dominiovec.ReferenciaEntradaCatalogo,
	publicadoEn time.Time,
) ([]byte, error) {
	entradas := make([]entradaMotivoPostgreSQLContratacionTemporalDesarrollo, len(motivos))
	for indice, motivo := range motivos {
		entradas[indice] = entradaMotivoPostgreSQLContratacionTemporalDesarrollo{
			Clave:        motivo.EntradaClave,
			VigenteDesde: publicadoEn.UTC().Format("2006-01-02T15:04:05.000000Z"),
		}
	}
	return json.Marshal(entradas)
}
