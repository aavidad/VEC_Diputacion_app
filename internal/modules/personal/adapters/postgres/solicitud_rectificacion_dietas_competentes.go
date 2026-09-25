package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

const consultarRectificacionesCompetentesDietasSQL = `SELECT recibo_ref,decision_ref,efecto_ref,consumo_huella_sha256,auditoria_ad3_ref,consultada_en,cardinalidad,solicitudes FROM vec_personal.consultar_rectificaciones_competentes_dietas_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`

type RepositorioRectificacionesCompetentesDietasPostgreSQL struct{ pool iniciadorConsultaRelacion }

var _ personalports.RepositorioRectificacionesCompetentesDietas = (*RepositorioRectificacionesCompetentesDietasPostgreSQL)(nil)

func NuevoRepositorioRectificacionesCompetentesDietasPostgreSQL(pool *pgxpool.Pool) (*RepositorioRectificacionesCompetentesDietasPostgreSQL, error) {
	if nuloRelacion(pool) {
		return nil, personalports.ErrRectificacionDietasNoDisponible
	}
	return &RepositorioRectificacionesCompetentesDietasPostgreSQL{pool: pool}, nil
}

func (r *RepositorioRectificacionesCompetentesDietasPostgreSQL) ConsultarRectificacionesCompetentesDietas(ctx context.Context, orden personalports.OrdenConsultaRectificacionesCompetentesDietas) (personalports.ResultadoConsultaRectificacionesCompetentesDietas, error) {
	var vacio personalports.ResultadoConsultaRectificacionesCompetentesDietas
	if r == nil || ctx == nil || nuloRelacion(r.pool) {
		return vacio, personalports.ErrRectificacionDietasNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	material := orden.Material.Canonico()
	if len(material) == 0 || orden.Autorizacion.ValidarEstructura() != nil {
		return vacio, personalports.ErrRectificacionDietasInvalida
	}
	args, secretos, err := parametrosConsultaRelacionPropia(material, orden.Autorizacion)
	if err != nil {
		return vacio, personalports.ErrRectificacionDietasInvalida
	}
	defer borrarRelacion(secretos[:])
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || tx == nil {
		return vacio, normalizarErrorRectificacionDietas(ctx, err)
	}
	confirmada := false
	defer func() {
		if !confirmada {
			_ = tx.Rollback(context.Background())
		}
	}()
	if _, err = tx.Exec(ctx, ajustesAsignacionDietas); err != nil {
		return vacio, normalizarErrorRectificacionDietas(ctx, err)
	}
	var resultado personalports.ResultadoConsultaRectificacionesCompetentesDietas
	var bruto []byte
	err = tx.QueryRow(ctx, consultarRectificacionesCompetentesDietasSQL, args...).Scan(
		&resultado.ReciboRef, &resultado.DecisionRef, &resultado.EfectoRef, &resultado.ConsumoHuellaSHA256,
		&resultado.AuditoriaAD3Ref, &resultado.ConsultadaEn, &resultado.Cardinalidad, &bruto,
	)
	if err != nil {
		return vacio, normalizarErrorRectificacionDietas(ctx, err)
	}
	if len(bruto) == 0 || len(bruto) > 128<<10 || bytes.Equal(bytes.TrimSpace(bruto), []byte("null")) || resultado.Cardinalidad < 0 || resultado.Cardinalidad > 50 {
		return vacio, personalports.ErrRectificacionDietasNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(bruto))
	d.DisallowUnknownFields()
	if d.Decode(&resultado.Solicitudes) != nil || d.Decode(new(any)) != io.EOF ||
		resultado.Solicitudes == nil || len(resultado.Solicitudes) != resultado.Cardinalidad ||
		!competentesRectificacionSQLValidas(resultado.Solicitudes, orden.Material.Solicitud().Actor.PersonaRef) {
		return vacio, personalports.ErrRectificacionDietasNoDisponible
	}
	resultado.ConsultadaEn = resultado.ConsultadaEn.UTC()
	for i := range resultado.Solicitudes {
		resultado.Solicitudes[i].RegistradaEn = resultado.Solicitudes[i].RegistradaEn.UTC()
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, normalizarErrorRectificacionDietas(ctx, err)
	}
	confirmada = true
	return resultado, nil
}

func competentesRectificacionSQLValidas(xs []personalports.SolicitudCompetenteRectificacionDietas, actor string) bool {
	for _, x := range xs {
		if x.Estado != "pendiente" || !personaldomain.ReferenciaPersonaValida(x.PersonaRef) ||
			!personaldomain.ReferenciaEmpleadoValida(x.EmpleadoRef) || !personaldomain.ReferenciaRelacionValida(x.RelacionRef) ||
			x.AsignacionActual.AdministrativoPersonaRef != actor || x.VersionOrigen < 1 ||
			x.AsignacionActual.Version < x.VersionOrigen || x.FechaReferencia.Validar() != nil ||
			x.RegistradaEn.IsZero() {
			return false
		}
	}
	return true
}
