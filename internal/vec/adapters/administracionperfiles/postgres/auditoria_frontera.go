package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/ports"
)

type AuditorFrontera struct{ pool conexion }

func NuevoAuditorFrontera(pool *pgxpool.Pool) (*AuditorFrontera, error) {
	if pool == nil {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return &AuditorFrontera{pool: pool}, nil
}
func (a *AuditorFrontera) RegistrarDenegacionADMIN(ctx context.Context, d api.DenegacionADMIN) error {
	if ctx == nil || a == nil || ausente(a.pool) {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(d.Codigo) == 0 || len(d.Codigo) > 128 || len(d.Accion) > 128 || len(d.RecursoRef) > 512 || len(d.ActorPersonaRef) > 512 || len(d.PerfilActivoRef) > 512 || len(d.CorrelacionRef) > 256 {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	var entropia [16]byte
	if _, err := rand.Read(entropia[:]); err != nil {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	material, err := json.Marshal(struct {
		OperacionRef    string `json:"operacion_ref"`
		Codigo          string `json:"codigo"`
		Accion          string `json:"accion"`
		RecursoRef      string `json:"recurso_ref"`
		ActorPersonaRef string `json:"actor_persona_ref,omitempty"`
		PerfilActivoRef string `json:"perfil_activo_ref,omitempty"`
		CorrelacionRef  string `json:"correlacion_ref,omitempty"`
	}{"frontera_admin:" + hex.EncodeToString(entropia[:]), d.Codigo, d.Accion, d.RecursoRef, d.ActorPersonaRef, d.PerfilActivoRef, d.CorrelacionRef})
	if err != nil {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || ausente(tx) {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(2*time.Second))
		defer cancel()
		_ = tx.Rollback(c)
	}()
	var bruto []byte
	if err := tx.QueryRow(ctx, `SELECT vec_autorizacion.registrar_denegacion_frontera_admin_v1($1::text)`, string(material)).Scan(&bruto); err != nil {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	var confirmada struct {
		OperacionRef string    `json:"operacion_ref"`
		AuditoriaRef string    `json:"auditoria_ref"`
		RegistradaEn time.Time `json:"registrada_en"`
	}
	if decodificar(bruto, &confirmada) != nil || confirmada.OperacionRef != "frontera_admin:"+hex.EncodeToString(entropia[:]) || !strings.HasPrefix(confirmada.AuditoriaRef, "auditoria:") || !instantePersistible(confirmada.RegistradaEn) {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return traducir(ctx, tx.Commit(ctx))
}

var _ api.AuditorFrontera = (*AuditorFrontera)(nil)
