package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// TransaccionVinculoOfertaBolsaPostgreSQL ejecuta CT188: esa única llamada
// consume los permisos CT y Bolsa y confirma ambos efectos en la misma TX.
type TransaccionVinculoOfertaBolsaPostgreSQL struct{ pool *pgxpool.Pool }

var _ ports.TransaccionVinculoOfertaBolsa = (*TransaccionVinculoOfertaBolsaPostgreSQL)(nil)

func NuevaTransaccionVinculoOfertaBolsaPostgreSQL(pool *pgxpool.Pool) (*TransaccionVinculoOfertaBolsaPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrVinculoOfertaBolsaNoDisponible
	}
	return &TransaccionVinculoOfertaBolsaPostgreSQL{pool: pool}, nil
}

type materialVinculoOfertaBolsaSQL struct {
	OrganizacionRef   string `json:"organizacion_ref"`
	ExpedienteRef     string `json:"expediente_ref"`
	VersionEsperada   uint64 `json:"version_esperada"`
	BolsaRef          string `json:"bolsa_ref"`
	OfertaRef         string `json:"oferta_ref"`
	NumeroPlaza       int    `json:"numero_plaza"`
	ActorRef          string `json:"actor_ref"`
	PerfilRef         string `json:"perfil_ref"`
	UnidadRef         string `json:"unidad_ref"`
	AmbitoRef         string `json:"ambito_ref"`
	ClaveIdempotencia string `json:"clave_idempotencia"`
	CorrelacionRef    string `json:"correlacion_ref"`
	FlujoRef          string `json:"flujo_ref"`
	FlujoVersion      uint64 `json:"flujo_version"`
	FlujoHuellaSHA256 string `json:"flujo_huella_sha256"`
}

func (r *TransaccionVinculoOfertaBolsaPostgreSQL) VincularOfertaBolsa(ctx context.Context,
	p ports.PreparacionVinculoOfertaBolsa) (ports.ResultadoVinculoOfertaBolsa, error) {
	vacio := ports.ResultadoVinculoOfertaBolsa{}
	if r == nil || r.pool == nil || ctx == nil || p.ValidarPara(p.Solicitud) != nil {
		return vacio, ports.ErrVinculoOfertaBolsaNoDisponible
	}
	s := p.Solicitud
	material, err := json.Marshal(materialVinculoOfertaBolsaSQL{
		OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		VersionEsperada: s.VersionEsperada, BolsaRef: s.BolsaRef,
		OfertaRef: s.OfertaRef, NumeroPlaza: s.NumeroPlaza,
		ActorRef: p.ActorRef, PerfilRef: p.PerfilRef,
		UnidadRef: p.UnidadRef, AmbitoRef: p.AmbitoRef,
		ClaveIdempotencia: s.ClaveIdempotencia, CorrelacionRef: p.CorrelacionRef,
		FlujoRef:          p.Definicion.Flujo.DefinicionRef,
		FlujoVersion:      p.Definicion.Flujo.Version,
		FlujoHuellaSHA256: p.Definicion.Flujo.HuellaSHA256,
	})
	if err != nil || len(material) > 16*1024 {
		return vacio, ports.ErrVinculoOfertaBolsaInvalido
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, errorVinculoOfertaBolsaSQL(ctx, err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
	 set_config('timezone','UTC',true),set_config('row_security','on',true),
	 set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),
	 set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return vacio, errorVinculoOfertaBolsaSQL(ctx, err)
	}
	ct, bolsa := p.AutorizacionCT, p.AutorizacionBolsa
	var salida []byte
	err = tx.QueryRow(ctx, `SELECT vec_contratacion_temporal.vincular_oferta_bolsa_v1(
	 $1::jsonb,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11,
	 $12,$13,$14,$15,$16::numeric,$17::numeric,$18,$19,$20,$21)`,
		material,
		ct.CapacidadCanonica(), ct.DecisionCanonica(), ct.MotivoCanonico(), ct.ContextoActorCanonico(),
		ct.PersonaVersion(), ct.PerfilVersion(), ct.PayloadVECAD3(), ct.SobreCOSESign1(), ct.EvidenciaVerificacion(), ct.RaizPublicaSPKI(),
		bolsa.CapacidadCanonica(), bolsa.DecisionCanonica(), bolsa.MotivoCanonico(), bolsa.ContextoActorCanonico(),
		bolsa.PersonaVersion(), bolsa.PerfilVersion(), bolsa.PayloadVECAD3(), bolsa.SobreCOSESign1(), bolsa.EvidenciaVerificacion(), bolsa.RaizPublicaSPKI(),
	).Scan(&salida)
	if err != nil {
		return vacio, errorVinculoOfertaBolsaSQL(ctx, err)
	}
	if len(salida) < 2 || len(salida) > 8*1024*1024 {
		return vacio, ports.ErrVinculoOfertaBolsaNoConfiable
	}
	var recibo ports.ResultadoVinculoOfertaBolsa
	if decodificarJSONEstricto(salida, &recibo) != nil {
		return vacio, ports.ErrVinculoOfertaBolsaNoConfiable
	}
	recibo.PublicadaEn = normalizarInstantePostgreSQL(recibo.PublicadaEn)
	recibo.AsociadaEn = normalizarInstantePostgreSQL(recibo.AsociadaEn)
	recibo.RegistradaEn = normalizarInstantePostgreSQL(recibo.RegistradaEn)
	if recibo.Anterior.Validar() != nil || recibo.Siguiente.Validar() != nil ||
		recibo.Anterior.OrganizacionRef != s.OrganizacionRef ||
		recibo.Anterior.Referencia != s.ExpedienteRef || recibo.Anterior.Version != s.VersionEsperada ||
		recibo.OfertaRef != s.OfertaRef || recibo.BolsaRef != s.BolsaRef || recibo.NumeroPlaza != s.NumeroPlaza {
		return vacio, ports.ErrVinculoOfertaBolsaNoConfiable
	}
	if err = tx.Commit(ctx); err != nil {
		// Un COMMIT sin respuesta se recupera con la misma clave semántica.
		return vacio, errorVinculoOfertaBolsaSQL(ctx, err)
	}
	return recibo, nil
}

func errorVinculoOfertaBolsaSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return ports.ErrVinculoOfertaBolsaDenegado
		case "23505":
			return ports.ErrVinculoOfertaBolsaEnConflicto
		case "22023", "23503":
			return ports.ErrVinculoOfertaBolsaInvalido
		}
	}
	return ports.ErrVinculoOfertaBolsaNoDisponible
}
