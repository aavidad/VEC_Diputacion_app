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
var _ ports.LectorExpedienteAvisoLlamamiento = (*LectorExpedienteSeleccionLlamamientoPostgreSQL)(nil)

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
		!domain.ReferenciaOpacaValida(referencia) || (version < 6 || version > ports.MaximoEnteroSeguroIntegracionBolsa) {
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
	lector := "vec_contratacion_temporal.leer_expediente_seleccion_v1"
	if version != 6 {
		lector = "vec_contratacion_temporal.leer_expediente_seleccion_v2"
	}
	err = tx.QueryRow(ctx, `SELECT expediente_json, version_actual FROM `+lector+`($1,$2,$3)`,
		organizacion, referencia, int64(version)).Scan(&contenido, &actual)
	defer borrarBytes(contenido)
	var expediente domain.Expediente
	if err != nil || actual < int64(version) ||
		actual > int64(ports.MaximoEnteroSeguroIntegracionBolsa) ||
		len(contenido) > 3*1024*1024 ||
		decodificarJSONEstricto(contenido, &expediente) != nil ||
		expediente.Validar() != nil || expediente.Referencia != referencia ||
		expediente.OrganizacionRef != organizacion || expediente.Version != version ||
		expediente.FaseActual != domain.FaseFiscalizacion ||
		expediente.EstadoActual != domain.EstadoEnCurso || expediente.Fiscalizacion == nil ||
		expediente.Fiscalizacion.Resultado == domain.FiscalizacionDesfavorable {
		return vacio, ports.ErrEstadoExpedienteNoSeleccionable
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, errorLecturaSeleccion(ctx)
	}
	return ports.ExpedienteParaSeleccion{Fiscalizado: expediente, VersionActual: uint64(actual)}, nil
}

// LeerExpedienteParaAvisoConfirmado deriva la versión fiscalizada desde la
// ejecución CT ya confirmada y ligada al llamamiento. Así la recuperación de
// aviso/respuesta/resolución conserva su snapshot original aunque CT avance.
func (l *LectorExpedienteSeleccionLlamamientoPostgreSQL) LeerExpedienteParaAvisoConfirmado(
	ctx context.Context, organizacion, referencia, llamamiento string,
) (ports.ExpedienteParaSeleccion, error) {
	vacio := ports.ExpedienteParaSeleccion{}
	if ctx == nil || l == nil || dependenciaNula(l.pool) ||
		!domain.ReferenciaOpacaValida(organizacion) || !domain.ReferenciaOpacaValida(referencia) ||
		!domain.ReferenciaOpacaValida(llamamiento) {
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
	err = tx.QueryRow(ctx, `SELECT expediente_json, version_actual FROM vec_contratacion_temporal.leer_expediente_aviso_confirmado_v1($1,$2,$3)`,
		organizacion, referencia, llamamiento).Scan(&contenido, &actual)
	defer borrarBytes(contenido)
	var expediente domain.Expediente
	if err != nil || actual < 6 || actual > int64(ports.MaximoEnteroSeguroIntegracionBolsa) ||
		len(contenido) > 3*1024*1024 || decodificarJSONEstricto(contenido, &expediente) != nil ||
		expediente.Validar() != nil || expediente.Referencia != referencia || expediente.OrganizacionRef != organizacion ||
		expediente.Version < 6 || expediente.Version > uint64(actual) ||
		expediente.FaseActual != domain.FaseFiscalizacion || expediente.EstadoActual != domain.EstadoEnCurso ||
		expediente.Fiscalizacion == nil || expediente.Fiscalizacion.Resultado == domain.FiscalizacionDesfavorable {
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
