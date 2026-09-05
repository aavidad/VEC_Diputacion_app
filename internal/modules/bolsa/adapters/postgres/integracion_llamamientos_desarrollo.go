package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// RepositorioIntegracionLlamamientosDesarrollo usa exclusivamente funciones del
// propietario Bolsa. Nunca consulta tablas CT ni concede permisos al runtime.
type RepositorioIntegracionLlamamientosDesarrollo struct{ pool iniciadorTransacciones }

var _ ports.RepositorioLlamamientoDesarrollo = (*RepositorioIntegracionLlamamientosDesarrollo)(nil)

func NuevoRepositorioIntegracionLlamamientosDesarrollo(pool *pgxpool.Pool) (*RepositorioIntegracionLlamamientosDesarrollo, error) {
	if pool == nil {
		return nil, ports.ErrIntegracionLlamamientoDesarrollo
	}
	return &RepositorioIntegracionLlamamientosDesarrollo{pool: pool}, nil
}

func (r *RepositorioIntegracionLlamamientosDesarrollo) iniciar(ctx context.Context) (pgx.Tx, error) {
	if ctx == nil || r == nil || valorNulo(r.pool) {
		return nil, ports.ErrIntegracionLlamamientoDesarrollo
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return nil, errorIntegracionDesarrollo(ctx, err)
	}
	_, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
  set_config('timezone','UTC',true), set_config('row_security','on',true),
  set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),
  set_config('idle_in_transaction_session_timeout','20s',true)`)
	if err != nil {
		revertir(tx)
		return nil, errorIntegracionDesarrollo(ctx, err)
	}
	return tx, nil
}

func (r *RepositorioIntegracionLlamamientosDesarrollo) BuscarOperacion(ctx context.Context, operacion string) (ports.RegistroLlamamientoDesarrollo, bool, error) {
	if !ports.ReferenciaOpacaLlamamientoValida(operacion) {
		return ports.RegistroLlamamientoDesarrollo{}, false, ports.ErrIntegracionLlamamientoDesarrollo
	}
	tx, err := r.iniciar(ctx)
	if err != nil {
		return ports.RegistroLlamamientoDesarrollo{}, false, err
	}
	defer revertir(tx)
	var b []byte
	err = tx.QueryRow(ctx, "SELECT registro_canonico FROM vec_bolsa_llamamientos.buscar_integracion_desarrollo_v1($1)", operacion).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.RegistroLlamamientoDesarrollo{}, false, nil
	}
	if err != nil {
		return ports.RegistroLlamamientoDesarrollo{}, false, errorIntegracionDesarrollo(ctx, err)
	}
	var registro ports.RegistroLlamamientoDesarrollo
	if len(b) > 4*1024*1024 || decodificarJSONExactoLlamamiento(b, &registro) != nil {
		return ports.RegistroLlamamientoDesarrollo{}, false, ports.ErrIntegracionLlamamientoDesarrollo
	}
	canonico, err := registro.Canonico()
	if err != nil || !bytes.Equal(b, canonico) || registro.OperacionRef != operacion {
		return ports.RegistroLlamamientoDesarrollo{}, false, ports.ErrIntegracionLlamamientoDesarrollo
	}
	return registro, true, nil
}

func (r *RepositorioIntegracionLlamamientosDesarrollo) Guardar(ctx context.Context, registro ports.RegistroLlamamientoDesarrollo, material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboLlamamientoDesarrollo, error) {
	b, err := registro.Canonico()
	if err != nil || len(b) > 4*1024*1024 || material.ValidarEstructura() != nil {
		return ports.ReciboLlamamientoDesarrollo{}, ports.ErrIntegracionLlamamientoDesarrollo
	}
	recurso, err := registro.RecursoAutorizable()
	if err != nil {
		return ports.ReciboLlamamientoDesarrollo{}, err
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	c := material.ResumenCapacidad()
	if err != nil || c.EfectoRef() != registro.OperacionRef || c.EfectoHuellaSHA256() != h ||
		c.Operacion() != registro.Accion() || c.AudienciaConsumo() != ports.AudienciaIntegracionLlamamientoDesarrollo {
		return ports.ReciboLlamamientoDesarrollo{}, ports.ErrIntegracionLlamamientoDesarrollo
	}
	tx, err := r.iniciar(ctx)
	if err != nil {
		return ports.ReciboLlamamientoDesarrollo{}, err
	}
	defer revertir(tx)
	var recibido []byte
	var recibo ports.ReciboLlamamientoDesarrollo
	err = tx.QueryRow(ctx, `SELECT registro_canonico,recibo_ref,auditoria_ref,evento_ref,confirmada_en
 FROM vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(
 $1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`,
		b, material.CapacidadCanonica(), material.DecisionCanonica(), material.MotivoCanonico(),
		material.ContextoActorCanonico(), material.PersonaVersion(), material.PerfilVersion(),
		material.PayloadVECAD3(), material.SobreCOSESign1(), material.EvidenciaVerificacion(), material.RaizPublicaSPKI(),
	).Scan(&recibido, &recibo.ReciboRef, &recibo.AuditoriaRef, &recibo.EventoRef, &recibo.ConfirmadaEn)
	if err != nil {
		return ports.ReciboLlamamientoDesarrollo{}, errorIntegracionDesarrollo(ctx, err)
	}
	recibo.ConfirmadaEn = recibo.ConfirmadaEn.UTC()
	if !bytes.Equal(recibido, b) || !ports.ReferenciaOpacaLlamamientoValida(recibo.ReciboRef) ||
		!ports.ReferenciaOpacaLlamamientoValida(recibo.AuditoriaRef) || !ports.ReferenciaOpacaLlamamientoValida(recibo.EventoRef) ||
		!instantePostgreSQLLlamamientoValido(recibo.ConfirmadaEn) || recibo.ConfirmadaEn.Before(registro.Instantanea.GeneradaEn) ||
		json.Unmarshal(recibido, &recibo.Registro) != nil {
		return ports.ReciboLlamamientoDesarrollo{}, ports.ErrIntegracionLlamamientoDesarrollo
	}
	if err := tx.Commit(ctx); err != nil {
		// Respuesta perdida no es éxito: el llamante repite la misma operación;
		// BuscarOperacion recuperará el contenido que PostgreSQL haya confirmado.
		return ports.ReciboLlamamientoDesarrollo{}, errorIntegracionDesarrollo(ctx, err)
	}
	return recibo, nil
}

func errorIntegracionDesarrollo(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	// No exportar errores PostgreSQL: pueden incluir documento, sujeto o DSN.
	return ports.ErrIntegracionLlamamientoDesarrollo
}
