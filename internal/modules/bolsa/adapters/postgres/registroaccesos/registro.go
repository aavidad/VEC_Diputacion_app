package registroaccesos

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	registroaplicacion "vec-diputacion-granada/internal/modules/bolsa/application/registroaccesos"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	nombrePreparadaConsultar = "vec_bolsa_registro_accesos_consultar_v1"
	sqlConsultar             = `
SELECT vec_bolsa_registro_accesos.consultar_accesos_administrativos_v1($1::jsonb)`
)

type iniciadorTransacciones interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// RegistroPostgreSQL solo usa funciones SECURITY DEFINER. La cuenta runtime
// no necesita privilegios directos sobre tablas, secuencias o funciones
// auxiliares.
type RegistroPostgreSQL struct {
	pool  iniciadorTransacciones
	ahora func() time.Time
}

var (
	_ vecports.AuditStore                              = (*RegistroPostgreSQL)(nil)
	_ registroaplicacion.RegistroAccesosAdministrativo = (*RegistroPostgreSQL)(nil)
)

func NuevoRegistroPostgreSQL(
	pool *pgxpool.Pool,
) (*RegistroPostgreSQL, error) {
	if pool == nil {
		return nil, registroaplicacion.ErrRegistroAccesosNoDisponible
	}
	return &RegistroPostgreSQL{
		pool:  pool,
		ahora: func() time.Time { return time.Now().UTC().Truncate(time.Microsecond) },
	}, nil
}

func (r *RegistroPostgreSQL) AppendAudit(
	ctx context.Context,
	entrada vecdomain.AuditEntry,
) (vecdomain.AuditEntry, error) {
	if err := validarContexto(ctx, r); err != nil ||
		registroaplicacion.ValidarEntradaRegistroAcceso(entrada) != nil {
		return vecdomain.AuditEntry{}, registroaplicacion.ErrRegistroAccesosInvalido
	}
	// AuditEntry no contiene una capacidad VEC ligada al efecto ni puede
	// demostrar que el registro se confirma atómicamente con la operación
	// gobernada. El contrato común queda deliberadamente cerrado: cada
	// vertical debe aportar un wrapper específico como el de consulta T13.
	return vecdomain.AuditEntry{}, vecdomain.ErrPermissionDenied
}

// ListAudit falla siempre cerrado: el contrato común no liga identidad,
// capacidad PDP, finalidad, filtros ni auditoría previa de la lectura.
func (*RegistroPostgreSQL) ListAudit(
	context.Context,
	string,
) ([]vecdomain.AuditEntry, error) {
	return nil, vecdomain.ErrPermissionDenied
}

func (r *RegistroPostgreSQL) ConsultarAccesosAdministrativos(
	ctx context.Context,
	solicitud registroaplicacion.SolicitudConsultaAdministrativaAccesos,
) (registroaplicacion.PaginaConsultaAdministrativaAccesos, error) {
	if err := validarContexto(ctx, r); err != nil {
		if ctx != nil && ctx.Err() != nil {
			return registroaplicacion.PaginaConsultaAdministrativaAccesos{},
				ctx.Err()
		}
		return registroaplicacion.PaginaConsultaAdministrativaAccesos{},
			registroaplicacion.ErrConsultaAdministrativaAccesosDenegada
	}
	instante := r.ahora()
	datosAutorizacion, err := solicitud.RevalidarAutorizacion(instante)
	if err != nil {
		return registroaplicacion.PaginaConsultaAdministrativaAccesos{},
			registroaplicacion.ErrConsultaAdministrativaAccesosDenegada
	}
	carga, err := serializarConsulta(solicitud, datosAutorizacion, instante)
	if err != nil {
		return registroaplicacion.PaginaConsultaAdministrativaAccesos{}, err
	}
	tx, err := r.iniciar(ctx)
	if err != nil {
		return registroaplicacion.PaginaConsultaAdministrativaAccesos{},
			normalizarError(ctx, err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err = tx.Prepare(
		ctx, nombrePreparadaConsultar, sqlConsultar,
	); err != nil {
		return registroaplicacion.PaginaConsultaAdministrativaAccesos{},
			normalizarError(ctx, err)
	}
	var respuesta []byte
	if err = tx.QueryRow(
		ctx, nombrePreparadaConsultar, carga,
	).Scan(&respuesta); err != nil {
		return registroaplicacion.PaginaConsultaAdministrativaAccesos{},
			normalizarError(ctx, err)
	}
	pagina, err := restaurarPagina(solicitud, respuesta)
	if err != nil {
		return registroaplicacion.PaginaConsultaAdministrativaAccesos{}, err
	}
	// El consumo de la capacidad, la auditoría y la lectura se confirman juntos.
	// Ningún dato personal sale del adaptador antes de este COMMIT.
	if err = tx.Commit(ctx); err != nil {
		return registroaplicacion.PaginaConsultaAdministrativaAccesos{},
			normalizarError(ctx, err)
	}
	return pagina, nil
}

func (r *RegistroPostgreSQL) iniciar(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return nil, err
	}
	ajustes := [...]string{
		"SET LOCAL search_path = pg_catalog",
		"SET LOCAL row_security = on",
		"SET LOCAL TIME ZONE 'UTC'",
		"SET LOCAL lock_timeout = '3s'",
		"SET LOCAL statement_timeout = '15s'",
		"SET LOCAL idle_in_transaction_session_timeout = '20s'",
	}
	for _, ajuste := range ajustes {
		if _, err := tx.Exec(ctx, ajuste); err != nil {
			_ = tx.Rollback(context.Background())
			return nil, err
		}
	}
	return tx, nil
}

func validarContexto(ctx context.Context, r *RegistroPostgreSQL) error {
	if ctx == nil || r == nil || r.pool == nil || r.ahora == nil {
		return registroaplicacion.ErrRegistroAccesosNoDisponible
	}
	return ctx.Err()
}

func normalizarError(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "42501", "28000":
			return registroaplicacion.ErrConsultaAdministrativaAccesosDenegada
		case "22023", "23514":
			return registroaplicacion.ErrRegistroAccesosInvalido
		}
	}
	return registroaplicacion.ErrRegistroAccesosNoDisponible
}
