package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	"vec-diputacion-granada/internal/vec/ports"
)

type ConfiguracionFronteraTecnica struct {
	Proceso, Canal string
	Plazo          time.Duration
}
type RegistradorFronteraTecnica struct {
	pool    conexion
	login   string
	config  ConfiguracionFronteraTecnica
	codigos map[string]string
}

var _ ports.RegistradorFronteraAdminTecnica = (*RegistradorFronteraTecnica)(nil)

// La configuración y el catálogo vienen del proveedor técnico propio AD189.
// No se concede pertenencia de rol ni se registra una identidad humana.
func NuevoRegistradorFronteraTecnica(ctx context.Context, pool *pgxpool.Pool, c ConfiguracionFronteraTecnica) (*RegistradorFronteraTecnica, error) {
	return nuevoRegistradorFronteraTecnica(ctx, pool, c)
}
func nuevoRegistradorFronteraTecnica(ctx context.Context, pool conexion, c ConfiguracionFronteraTecnica) (*RegistradorFronteraTecnica, error) {
	if ctx == nil || ausente(pool) || c.Plazo <= 0 || c.Plazo > 2*time.Second || c.Proceso == "" || c.Canal != "administracion_privilegiada" {
		return nil, ports.ErrFronteraAdminTecnicaNoDisponible
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || ausente(tx) {
		return nil, falloFronteraTecnica(err)
	}
	defer cerrarTxFronteraTecnica(tx, c.Plazo)
	var b []byte
	if err := tx.QueryRow(ctx, `SELECT vec_autorizacion_atestada_v3.acreditar_frontera_admin_tecnica_v1()`).Scan(&b); err != nil {
		return nil, falloFronteraTecnica(err)
	}
	var r struct {
		Login   string `json:"operador_login"`
		Proceso string `json:"proceso"`
		Canal   string `json:"canal"`
		Codigos []struct {
			Codigo    string `json:"codigo_ref"`
			Resultado string `json:"resultado"`
		} `json:"codigos"`
	}
	if err := decodificar(b, &r); err != nil {
		return nil, falloFronteraTecnica(err)
	}
	if r.Login == "" || r.Proceso != c.Proceso || r.Canal != c.Canal || len(r.Codigos) == 0 {
		return nil, ports.ErrFronteraAdminTecnicaNoDisponible
	}
	codigos := make(map[string]string, len(r.Codigos))
	for _, d := range r.Codigos {
		if d.Codigo == "" || (d.Resultado != "denegado" && d.Resultado != "error") || codigos[d.Codigo] != "" {
			return nil, ports.ErrFronteraAdminTecnicaNoDisponible
		}
		codigos[d.Codigo] = d.Resultado
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, falloFronteraTecnica(err)
	}
	return &RegistradorFronteraTecnica{pool: pool, login: r.Login, config: c, codigos: codigos}, nil
}
func falloFronteraTecnica(err error) error {
	if err != nil {
		return ports.ErrFronteraAdminTecnicaNoDisponible
	}
	return ports.ErrFronteraAdminTecnicaNoDisponible
}
func cerrarTxFronteraTecnica(tx pgx.Tx, plazo time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(plazo))
	defer cancel()
	_ = tx.Rollback(ctx)
}

func (r *RegistradorFronteraTecnica) AppendFronteraAdminTecnica(ctx context.Context, e ports.EventoFronteraAdminTecnica) (ports.AcuseFronteraAdminTecnica, error) {
	var vacio ports.AcuseFronteraAdminTecnica
	if r == nil || ctx == nil || ausente(r.pool) || e.Validar() != nil || e.OperadorLogin != r.login || e.Proceso != r.config.Proceso || e.Canal != r.config.Canal || r.codigos[e.CodigoRef] != e.Resultado {
		return vacio, ports.ErrFronteraAdminTecnicaNoDisponible
	}
	b, err := json.Marshal(e)
	if err != nil {
		return vacio, falloFronteraTecnica(err)
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || ausente(tx) {
		return vacio, falloFronteraTecnica(err)
	}
	defer cerrarTxFronteraTecnica(tx, r.config.Plazo)
	var raw []byte
	if err := tx.QueryRow(ctx, `SELECT to_jsonb(a) FROM vec_autorizacion_atestada_v3.registrar_frontera_admin_tecnica_v1($1::jsonb) a`, b).Scan(&raw); err != nil {
		return vacio, falloFronteraTecnica(err)
	}
	var acuse ports.AcuseFronteraAdminTecnica
	if err := decodificar(raw, &acuse); err != nil {
		return vacio, falloFronteraTecnica(err)
	}
	if acuse.ValidarPara(e) != nil || ctx.Err() != nil {
		return vacio, ports.ErrFronteraAdminTecnicaNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, ports.ErrFronteraAdminTecnicaCommitIncierto
	}
	return acuse, nil
}
