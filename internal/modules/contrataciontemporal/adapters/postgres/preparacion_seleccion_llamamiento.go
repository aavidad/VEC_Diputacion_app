package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type LectorExpedienteSeleccionLlamamientoPostgreSQL struct {
	pool iniciadorTransacciones
}

var _ ports.LectorExpedienteSeleccionLlamamiento = (*LectorExpedienteSeleccionLlamamientoPostgreSQL)(nil)

func NuevoLectorExpedienteSeleccionLlamamientoPostgreSQL(pool *pgxpool.Pool) (*LectorExpedienteSeleccionLlamamientoPostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil, ports.ErrIntegracionBolsaNoDisponible
	}
	return &LectorExpedienteSeleccionLlamamientoPostgreSQL{pool: pool}, nil
}

func (l *LectorExpedienteSeleccionLlamamientoPostgreSQL) LeerExpedienteParaSeleccion(
	ctx context.Context, organizacion, referencia string, version uint64,
) (ports.ExpedienteParaSeleccion, error) {
	vacio := ports.ExpedienteParaSeleccion{}
	if ctx == nil || l == nil || dependenciaNula(l.pool) ||
		!domain.ReferenciaOpacaValida(organizacion) ||
		!domain.ReferenciaOpacaValida(referencia) || version != 6 {
		return vacio, ports.ErrPeticionIntegracionBolsaInvalida
	}
	ctx, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return vacio, errorLecturaSeleccion(ctx)
	}
	defer revertirTransaccion(tx)
	var contenido []byte
	var actual int64
	err = tx.QueryRow(ctx, `SELECT expediente_json, version_actual FROM
		vec_contratacion_temporal.leer_expediente_seleccion_v1($1,$2,$3)`,
		organizacion, referencia, int64(version)).Scan(&contenido, &actual)
	defer borrarBytes(contenido)
	var expediente domain.Expediente
	if err != nil || actual < 6 ||
		actual > int64(ports.MaximoEnteroSeguroIntegracionBolsa) ||
		len(contenido) > 3*1024*1024 ||
		decodificarJSONEstricto(contenido, &expediente) != nil ||
		expediente.Validar() != nil || expediente.Referencia != referencia ||
		expediente.OrganizacionRef != organizacion || expediente.Version != version ||
		expediente.FaseActual != domain.FaseFiscalizacion ||
		expediente.EstadoActual != domain.EstadoEnCurso || expediente.Fiscalizacion == nil ||
		expediente.Fiscalizacion.Resultado == domain.FiscalizacionDesfavorable {
		return vacio, errorLecturaSeleccion(ctx)
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, errorLecturaSeleccion(ctx)
	}
	return ports.ExpedienteParaSeleccion{Fiscalizado: expediente, VersionActual: uint64(actual)}, nil
}

func errorLecturaSeleccion(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return ports.ErrIntegracionBolsaNoDisponible
}
