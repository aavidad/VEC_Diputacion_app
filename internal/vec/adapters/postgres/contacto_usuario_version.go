package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	vecapp "vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/ports"
)

type ConsultorVersionContactoUsuarioPostgreSQL struct{ pool iniciadorTransacciones }

func NuevoConsultorVersionContactoUsuarioPostgreSQL(pool *pgxpool.Pool) (*ConsultorVersionContactoUsuarioPostgreSQL, error) {
	return nuevoConsultorVersionContactoUsuarioPostgreSQL(pool)
}
func nuevoConsultorVersionContactoUsuarioPostgreSQL(pool iniciadorTransacciones) (*ConsultorVersionContactoUsuarioPostgreSQL, error) {
	if valorNuloPostgreSQL(pool) {
		return nil, vecapp.ErrContactoUsuarioNoDisponible
	}
	return &ConsultorVersionContactoUsuarioPostgreSQL{pool}, nil
}
func (c *ConsultorVersionContactoUsuarioPostgreSQL) ConsultarVersionContactoUsuario(ctx context.Context, o ports.OrdenVersionContactoUsuario) (ports.ResultadoVersionContactoUsuario, error) {
	vacio := ports.ResultadoVersionContactoUsuario{}
	if c == nil || valorNuloPostgreSQL(c.pool) || ctx == nil || ctx.Err() != nil || vecapp.ValidarOrdenVersionContactoUsuario(o, time.Now().UTC().Truncate(time.Microsecond)) != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	a := o.Acceso
	recurso, auditoria, err := contactoRecursoAuditoria(a.Recurso, a.Auditoria)
	if err != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	tx, err := c.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || valorNuloPostgreSQL(tx) {
		return vacio, contactoError(ctx, err)
	}
	defer revertirTransaccionPostgreSQL(tx)
	if err = configurarTransaccionContacto(ctx, tx); err != nil {
		return vacio, contactoError(ctx, err)
	}
	var encontrado bool
	var sujeto string
	var version uint64
	var evidencia []byte
	var consumoRef, consumoHuella string
	accion := vecapp.AccionVersionContactoPropia
	if o.Clase == ports.VersionContactoLlamamiento {
		accion = vecapp.AccionVersionContactoLlamamiento
	}
	err = tx.QueryRow(ctx, `SELECT encontrado,sujeto_ref,version_actual,auditoria_consulta,consumo_consulta_ref,consumo_consulta_huella_sha256 FROM vec_contacto_usuario_v1.consultar_version_contacto_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11,$12,$13,$14)`, argumentosContacto(accion, a.Material, a.PayloadNegocio, recurso, auditoria)...).Scan(&encontrado, &sujeto, &version, &evidencia, &consumoRef, &consumoHuella)
	if err != nil {
		return vacio, contactoError(ctx, err)
	}
	r, err := vecapp.DecodificarResultadoVersionContactoUsuario(o, encontrado, sujeto, version, evidencia, consumoRef, consumoHuella)
	if err != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, contactoError(ctx, err)
	}
	return r, nil
}

var _ ports.ConsultorVersionContactoUsuario = (*ConsultorVersionContactoUsuarioPostgreSQL)(nil)
