package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const consultaRegistroPropio = `SELECT vec_identidad_sesiones_v1.registrar_propio_v1($1::bytea,$2::bytea,$3::text,$4::bytea,$5::bytea,$6::bytea,$7::bytea,$8::numeric,$9::numeric,$10::bytea,$11::bytea,$12::bytea,$13::bytea)`

type iniciadorRegistroPropio interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type RepositorioRegistroPropioPostgreSQL struct{ pool iniciadorRegistroPropio }

func NuevoRepositorioRegistroPropioPostgreSQL(pool *pgxpool.Pool) (*RepositorioRegistroPropioPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrRegistroPropioNoDisponible
	}
	return &RepositorioRegistroPropioPostgreSQL{pool: pool}, nil
}

func (r *RepositorioRegistroPropioPostgreSQL) RegistrarPropio(ctx context.Context, sujetoRef string, preparar func(context.Context) (ports.OrdenRegistroPropioV1, error)) (domain.ReciboRegistroPropioV1, error) {
	if r == nil || nuloRegistroPropioPostgres(r.pool) || ctx == nil || ctx.Err() != nil ||
		preparar == nil || !domain.ReferenciaSujetoRegistroPropioValida(sujetoRef) {
		return domain.ReciboRegistroPropioV1{}, ports.ErrRegistroPropioNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return domain.ReciboRegistroPropioV1{}, errorRegistroPropio(err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true)`); err != nil {
		return domain.ReciboRegistroPropioV1{}, errorRegistroPropio(err)
	}
	// Mismo lock que toma la función SQL. Se retiene durante la revalidación
	// institucional y V3 y hasta el commit; otra alta del sujeto no puede pasar.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('vec:registro-propio:sujeto:'||$1,0))`, sujetoRef); err != nil {
		return domain.ReciboRegistroPropioV1{}, errorRegistroPropio(err)
	}
	orden, err := preparar(ctx)
	if err != nil || orden.Acreditacion.SujetoRef != sujetoRef || orden.Equivalencia.SujetoRef != sujetoRef ||
		orden.OperacionRef == "" || orden.ActorRef == "" || orden.Acreditacion.ValidarEn(orden.Acreditacion.VigenteDesde) != nil ||
		len(orden.EntradaCanonica) == 0 || len(orden.RecursoCanonico) == 0 || orden.Material.ValidarEstructura() != nil ||
		orden.Decision.ValidarPara(orden.Solicitud) != nil {
		return domain.ReciboRegistroPropioV1{}, ports.ErrRegistroPropioNoDisponible
	}
	data, err := orden.Solicitud.Datos()
	if err != nil || data.Accion != ports.AccionRegistroPropioV1 || data.Finalidad != ports.FinalidadRegistroPropioV1 || data.Recurso.Referencia != orden.OperacionRef {
		return domain.ReciboRegistroPropioV1{}, ports.ErrRegistroPropioNoDisponible
	}
	vinculo, err := data.VinculoAutenticacionActor.Datos()
	if err != nil || vinculo.PrincipalID != orden.ActorRef {
		return domain.ReciboRegistroPropioV1{}, ports.ErrRegistroPropioNoDisponible
	}
	m := orden.Material
	var raw []byte
	err = tx.QueryRow(ctx, consultaRegistroPropio, orden.EntradaCanonica, orden.RecursoCanonico, orden.ActorRef,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&raw)
	if err != nil {
		return domain.ReciboRegistroPropioV1{}, errorRegistroPropio(err)
	}
	var recibo domain.ReciboRegistroPropioV1
	if json.Unmarshal(raw, &recibo) != nil || recibo.ValidarPendiente() != nil || recibo.OperacionRef != orden.OperacionRef {
		return domain.ReciboRegistroPropioV1{}, ports.ErrRegistroPropioNoDisponible
	}
	if err = ctx.Err(); err != nil {
		return domain.ReciboRegistroPropioV1{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.ReciboRegistroPropioV1{}, errorRegistroPropio(err)
	}
	return recibo, nil
}

func nuloRegistroPropioPostgres(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return r.IsNil()
	}
	return false
}

func errorRegistroPropio(err error) error {
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.Code {
		case "42501":
			return domain.ErrAutorizacionDenegada
		case "23505":
			return domain.ErrRegistroPropioInvalido
		}
	}
	return ports.ErrRegistroPropioNoDisponible
}
