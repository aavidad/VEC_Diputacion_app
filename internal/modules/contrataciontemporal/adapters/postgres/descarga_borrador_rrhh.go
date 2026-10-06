package postgres

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// registrarDescargaBorradorRRHHPostgreSQL llama a CT177 con el LOGIN del
// consultor RRHH: consume AD199 y escribe la fila en la misma transacción.
const registrarDescargaBorradorRRHHPostgreSQL = `
SELECT descarga_ref, auditoria_ref, decision_ref, registrada_en
  FROM vec_contratacion_temporal.registrar_descarga_borrador_rrhh_v1(
       ROW($1::text, $2::text, $3::text)::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
       $4::text, $5::numeric, $6::text, $7::text, $8::text, $9::integer, $10::text,
       $11::bytea, $12::bytea, $13::bytea, $14::bytea, $15::numeric, $16::numeric,
       $17::bytea, $18::bytea, $19::bytea, $20::bytea)`

var (
	descargaRefValida  = regexp.MustCompile(`^descarga_borrador:[0-9a-f]{64}$`)
	auditoriaRefValida = regexp.MustCompile(`^aud_v3_[0-9a-f]{32}$`)
)

// RepositorioDescargaBorradorRRHHPostgreSQL usa el mismo pool acreditado que
// la consulta de detalle; nunca abre otra identidad.
type RepositorioDescargaBorradorRRHHPostgreSQL struct {
	pool iniciadorTransacciones
}

var _ ports.RepositorioDescargaBorradorRRHH = (*RepositorioDescargaBorradorRRHHPostgreSQL)(nil)

func NuevoRepositorioDescargaBorradorRRHHPostgreSQL(pool *PoolConsultasRRHHPostgreSQL) (*RepositorioDescargaBorradorRRHHPostgreSQL, error) {
	if pool == nil || pool.iniciador == nil {
		return nil, ports.ErrDescargaBorradorRRHHNoDisponible
	}
	return &RepositorioDescargaBorradorRRHHPostgreSQL{pool: pool.iniciador}, nil
}

func (r *RepositorioDescargaBorradorRRHHPostgreSQL) RegistrarDescargaBorrador(ctx context.Context, a ports.AlcanceDescargaBorradorRRHH,
	s ports.SolicitudDescargaBorradorRRHH, m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
) (ports.ReciboDescargaBorradorRRHH, error) {
	vacio := ports.ReciboDescargaBorradorRRHH{}
	if r == nil || dependenciaNula(r.pool) || ctx == nil || s.Validar() != nil || a.OrganizacionRef == "" ||
		a.ClaseAmbito == "" || a.AmbitoRef == "" || m.ValidarEstructura() != nil {
		return vacio, ports.ErrDescargaBorradorRRHHNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, errorDescargaBorradorRRHH(ctx, err)
	}
	defer revertirTransaccion(tx)
	var recibo ports.ReciboDescargaBorradorRRHH
	err = tx.QueryRow(ctx, registrarDescargaBorradorRRHHPostgreSQL,
		a.OrganizacionRef, string(a.ClaseAmbito), a.AmbitoRef,
		s.ExpedienteRef, strconv.FormatUint(s.VersionExpediente, 10), string(s.Tipo), s.Formato,
		s.DocumentoSHA256, s.TamanoBytes, s.ConsultaHuellaSHA256,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		strconv.FormatUint(m.PersonaVersion(), 10), strconv.FormatUint(m.PerfilVersion(), 10),
		m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI(),
	).Scan(&recibo.DescargaRef, &recibo.AuditoriaRef, &recibo.DecisionRef, &recibo.RegistradaEn)
	if err != nil {
		return vacio, errorDescargaBorradorRRHH(ctx, err)
	}
	recibo.RegistradaEn = recibo.RegistradaEn.UTC()
	if !descargaRefValida.MatchString(recibo.DescargaRef) || !auditoriaRefValida.MatchString(recibo.AuditoriaRef) ||
		recibo.DecisionRef == "" || recibo.RegistradaEn.IsZero() || recibo.RegistradaEn.After(time.Now().Add(time.Minute)) {
		return vacio, ports.ErrDescargaBorradorRRHHNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, errorDescargaBorradorRRHH(ctx, err)
	}
	return recibo, nil
}

// Solo 42501 es una denegación; P0002 (versión inexistente) y cualquier otro
// código, incluido 55000 de una transacción mal configurada, son fallos.
func errorDescargaBorradorRRHH(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "42501":
			return errors.Join(ports.ErrAutorizacionDenegada, dominiovec.ErrAutorizacionDenegada)
		case "P0002":
			return ports.ErrDescargaBorradorRRHHVersionAusente
		}
	}
	return ports.ErrDescargaBorradorRRHHNoDisponible
}
