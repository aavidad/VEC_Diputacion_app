package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"strconv"
	"time"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var ErrPreservacionCommitIndeterminado = errors.New("preservacion_auditoria_commit_indeterminado")

type FuentePreservacionAuditoriaPostgreSQL struct {
	pool poolCheckpointPeriodico
	rol  string
}

var _ ports.FuentePreservacionAuditoria = (*FuentePreservacionAuditoriaPostgreSQL)(nil)

// La operación elige una fachada fija; el LOGIN privado requiere su concesión
// propia. El selector no publica permisos ni presta el rol del sello periódico.
func NuevaFuentePreservacionAuditoriaPostgreSQL(pool *pgxpool.Pool, configurador bool) (*FuentePreservacionAuditoriaPostgreSQL, error) {
	if pool == nil {
		return nil, domain.ErrPreservacionAuditoriaNoDisponible
	}
	rol := "vec_auditoria_preservacion_consultor"
	if configurador {
		rol = "vec_auditoria_preservacion_configurador"
	}
	return &FuentePreservacionAuditoriaPostgreSQL{pool: pool, rol: rol}, nil
}

func (f *FuentePreservacionAuditoriaPostgreSQL) PublicarPreservacionAuditoria(ctx context.Context, s domain.SolicitudPreservacionAuditoria) (domain.ResultadoPreservacionAuditoria, error) {
	if s.Validar() != nil {
		return domain.ResultadoPreservacionAuditoria{}, domain.ErrPreservacionAuditoriaInvalida
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return domain.ResultadoPreservacionAuditoria{}, err
	}
	var r domain.ResultadoPreservacionAuditoria
	err = f.operar(ctx, "configurar_preservacion_auditoria_v1", `SELECT vec_autorizacion_atestada_v3.configurar_preservacion_auditoria_v1($2::text,$1::text)`, &r, string(raw))
	if err != nil {
		return domain.ResultadoPreservacionAuditoria{}, err
	}
	return r, nil
}
func (f *FuentePreservacionAuditoriaPostgreSQL) ConsultarPreservacionAuditoria(ctx context.Context, version uint64) (domain.ResultadoPreservacionAuditoria, error) {
	if version > domain.MaxVersionPreservacionAuditoria {
		return domain.ResultadoPreservacionAuditoria{}, domain.ErrPreservacionAuditoriaInvalida
	}
	var r domain.ResultadoPreservacionAuditoria
	err := f.operar(ctx, "consultar_preservacion_auditoria_v1", `SELECT vec_autorizacion_atestada_v3.consultar_preservacion_auditoria_v1($2::bigint,$1::text)`, &r, strconv.FormatUint(version, 10))
	if err != nil {
		return domain.ResultadoPreservacionAuditoria{}, err
	}
	return r, nil
}

func (f *FuentePreservacionAuditoriaPostgreSQL) operar(ctx context.Context, accion, consulta string, destino any, args ...any) error {
	correlacion, ok := ports.CorrelacionIncidenciasPeticion(ctx)
	if f == nil || valorNuloPostgreSQL(f.pool) || ctx == nil || !ok {
		return domain.ErrPreservacionAuditoriaNoDisponible
	}
	params := append([]any{"correlacion_" + correlacion}, args...)
	err := f.transaccion(ctx, consulta, destino, params...)
	if err == nil || errors.Is(err, ErrPreservacionCommitIndeterminado) {
		return err
	}
	resultado := "error"
	var p *pgconn.PgError
	if errors.As(err, &p) && p.Code == "42501" {
		resultado = "denegado"
	}
	if f.RegistrarFalloPreservacion(ctx, accion, resultado) != nil {
		return domain.ErrPreservacionAuditoriaNoDisponible
	}
	return domain.ErrPreservacionAuditoriaNoDisponible
}

// Se invoca una vez cerrado el intento original. No incluye mensajes del
// proveedor ni marca un COMMIT incierto como una operación sin efecto.
func (f *FuentePreservacionAuditoriaPostgreSQL) RegistrarFalloPreservacion(ctx context.Context, accion, resultado string) error {
	corr, ok := ports.CorrelacionIncidenciasPeticion(ctx)
	if ctx == nil || !ok || f == nil || valorNuloPostgreSQL(f.pool) {
		return domain.ErrPreservacionAuditoriaNoDisponible
	}
	c, cancel := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(10*time.Second))
	defer cancel()
	var a domain.AcusePreservacionAuditoria
	return f.transaccion(c, `SELECT vec_autorizacion_atestada_v3.registrar_intento_preservacion_v1($1::text,$2::text,$3::text)`, &a, accion, resultado, "correlacion_"+corr)
}

func (f *FuentePreservacionAuditoriaPostgreSQL) transaccion(ctx context.Context, consulta string, destino any, args ...any) error {
	tx, err := f.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return err
	}
	rollback := func() error {
		c, cancel := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(3*time.Second))
		defer cancel()
		return tx.Rollback(c)
	}
	defer func() { _ = rollback() }()
	// LOGIN propio NOINHERIT: activa únicamente la autoridad fija de esta
	// fachada, nunca un rol enviado por el cliente ni membresía de owner.
	rol := `SET LOCAL ROLE vec_auditoria_preservacion_consultor`
	if f.rol == "vec_auditoria_preservacion_configurador" {
		rol = `SET LOCAL ROLE vec_auditoria_preservacion_configurador`
	}
	if _, err = tx.Exec(ctx, rol); err != nil {
		if rollback() != nil {
			return ErrPreservacionCommitIndeterminado
		}
		return err
	}
	if _, err = tx.Exec(ctx, `SET LOCAL TIME ZONE 'UTC'`); err != nil {
		if rollback() != nil {
			return ErrPreservacionCommitIndeterminado
		}
		return err
	}
	var raw []byte
	if err = tx.QueryRow(ctx, consulta, args...).Scan(&raw); err != nil {
		if rollback() != nil {
			return ErrPreservacionCommitIndeterminado
		}
		return err
	}
	if len(raw) > 32768 || json.Unmarshal(raw, destino) != nil {
		if rollback() != nil {
			return ErrPreservacionCommitIndeterminado
		}
		return domain.ErrPreservacionAuditoriaNoDisponible
	}
	if tx.Commit(ctx) != nil {
		return ErrPreservacionCommitIndeterminado
	}
	return nil
}
