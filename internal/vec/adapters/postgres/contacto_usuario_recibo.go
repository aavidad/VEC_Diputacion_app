package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	vecapp "vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/ports"
)

// ConsultorReciboContactoUsuarioPostgreSQL sólo recupera evidencia histórica.
// El pool usa el perfil nominal de recibos autorizado por almacén2; no recibe
// protector, clave, sobre ni capacidad para descifrar el correo.
type ConsultorReciboContactoUsuarioPostgreSQL struct{ pool iniciadorTransacciones }

func NuevoConsultorReciboContactoUsuarioPostgreSQL(pool *pgxpool.Pool) (*ConsultorReciboContactoUsuarioPostgreSQL, error) {
	return nuevoConsultorReciboContactoUsuarioPostgreSQL(pool)
}
func nuevoConsultorReciboContactoUsuarioPostgreSQL(pool iniciadorTransacciones) (*ConsultorReciboContactoUsuarioPostgreSQL, error) {
	if valorNuloPostgreSQL(pool) {
		return nil, vecapp.ErrContactoUsuarioNoDisponible
	}
	return &ConsultorReciboContactoUsuarioPostgreSQL{pool: pool}, nil
}

func (r *ConsultorReciboContactoUsuarioPostgreSQL) ConsultarReciboContactoUsuario(ctx context.Context, o ports.OrdenConsultaReciboContactoUsuario) (ports.ResultadoConsultaReciboContactoUsuario, error) {
	vacio := ports.ResultadoConsultaReciboContactoUsuario{}
	if r == nil || valorNuloPostgreSQL(r.pool) || ctx == nil || ctx.Err() != nil || vecapp.ValidarOrdenConsultaReciboContactoUsuario(o, time.Now().UTC().Truncate(time.Microsecond)) != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	a := o.Acceso
	recurso, auditoria, err := contactoRecursoAuditoria(a.Recurso, a.Auditoria)
	if err != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
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
	var original, consulta []byte
	var consumoOriginalRef, consumoOriginalHuella, consumoConsultaRef, consumoConsultaHuella string
	err = tx.QueryRow(ctx, `SELECT encontrado, sujeto_ref, version, recibo_original, consumo_original_ref, consumo_original_huella_sha256, auditoria_consulta, consumo_consulta_ref, consumo_consulta_huella_sha256 FROM vec_contacto_usuario_v1.consultar_recibo_contacto_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11,$12,$13,$14)`, argumentosContacto(vecapp.AccionConsultarContactoUsuario, a.Material, a.PayloadNegocio, recurso, auditoria)...).Scan(&encontrado, &sujeto, &version, &original, &consumoOriginalRef, &consumoOriginalHuella, &consulta, &consumoConsultaRef, &consumoConsultaHuella)
	if err != nil {
		return vacio, contactoError(ctx, err)
	}
	resultado, err := vecapp.DecodificarResultadoConsultaReciboContactoUsuario(o, encontrado, sujeto, version, original, consumoOriginalRef, consumoOriginalHuella, consulta, consumoConsultaRef, consumoConsultaHuella)
	if err != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, contactoError(ctx, err)
	}
	// La auditoría de acceso se confirma incluso si la versión no se encontró.
	// No hay descifrado ni callback posterior en esta capacidad.
	return resultado, nil
}

var _ ports.ConsultorReciboContactoUsuario = (*ConsultorReciboContactoUsuarioPostgreSQL)(nil)
