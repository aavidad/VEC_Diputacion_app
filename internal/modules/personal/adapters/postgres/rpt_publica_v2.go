package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"reflect"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const consultaRPTPublicaV2SQL = `SELECT vec_personal.consumir_consulta_rpt_publica_v2($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
const ajustesRPTPublicaV2SQL = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`
const maxRespuestaRPTPublicaV2 = 8 << 10

var patronHuellaLecturaRPTV2 = regexp.MustCompile(`^[a-f0-9]{64}$`)

type iniciadorRPTPublicaV2 interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type ConsumidorRPTPublicaV2PostgreSQL struct{ pool iniciadorRPTPublicaV2 }

func NuevoConsumidorRPTPublicaV2PostgreSQL(pool *pgxpool.Pool) (*ConsumidorRPTPublicaV2PostgreSQL, error) {
	if pool == nil {
		return nil, domain.ErrRPTPublicaV2NoDisponible
	}
	return &ConsumidorRPTPublicaV2PostgreSQL{pool: pool}, nil
}

func (r *ConsumidorRPTPublicaV2PostgreSQL) ConsumirLecturaRPTPublicaV2(ctx context.Context, o ports.OrdenLecturaRPTPublicaV2) (salida ports.EvidenciaLecturaRPTPublicaV2, errorSalida error) {
	var vacia ports.EvidenciaLecturaRPTPublicaV2
	if ctx == nil || r == nil || nuloRPTPublicaV2Postgres(r.pool) {
		return vacia, domain.ErrRPTPublicaV2NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	m := o.Material
	reconstruido, err := domain.NuevoMaterialConsultaRPTPublicaV2(m.Solicitud())
	if err != nil || !bytes.Equal(reconstruido.Canonico(), m.Canonico()) || !reflect.DeepEqual(reconstruido.Recurso(), m.Recurso()) ||
		!autorizacionRPTV2PostgresValida(m, o.Autorizacion) ||
		o.Autorizacion.PersonaVersion() > math.MaxInt64 || o.Autorizacion.PerfilVersion() > math.MaxInt64 {
		return vacia, domain.ErrRPTPublicaV2Invalida
	}
	a := o.Autorizacion
	parametros := []any{string(m.Canonico()), a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(), int64(a.PersonaVersion()), int64(a.PerfilVersion()), a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer func() {
		for _, p := range parametros {
			if b, ok := p.([]byte); ok {
				clear(b)
			}
		}
	}()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacia, errorRPTPublicaV2Postgres(ctx, err)
	}
	if tx == nil {
		return vacia, domain.ErrRPTPublicaV2NoDisponible
	}
	confirmada := false
	defer func() {
		if !confirmada {
			ctxCierre, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(2*time.Second))
			defer cancelar()
			if err := tx.Rollback(ctxCierre); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				salida, errorSalida = vacia, domain.ErrRPTPublicaV2NoDisponible
			}
		}
	}()
	if _, err := tx.Exec(ctx, ajustesRPTPublicaV2SQL); err != nil {
		return vacia, errorRPTPublicaV2Postgres(ctx, err)
	}
	var bruto []byte
	if err := tx.QueryRow(ctx, consultaRPTPublicaV2SQL, parametros...).Scan(&bruto); err != nil {
		return vacia, errorRPTPublicaV2Postgres(ctx, err)
	}
	if len(bruto) == 0 || len(bruto) > maxRespuestaRPTPublicaV2 {
		return vacia, domain.ErrRPTPublicaV2NoDisponible
	}
	evidencia, err := decodificarEvidenciaRPTV2(bruto)
	if err != nil || !evidenciaRPTV2PostgresValida(m, a, evidencia) {
		return vacia, domain.ErrRPTPublicaV2NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	if err := tx.Commit(ctx); err != nil {
		return vacia, errorRPTPublicaV2Postgres(ctx, err)
	}
	confirmada = true
	return evidencia, nil
}

func autorizacionRPTV2PostgresValida(m domain.MaterialConsultaRPTPublicaV2, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	if a.ValidarEstructura() != nil {
		return false
	}
	s := m.Solicitud()
	h, err := m.HuellaSHA256()
	x := a.ResumenCapacidad()
	return err == nil && a.PersonaVersion() == s.Actor.Instantanea.PersonaVersion && a.PerfilVersion() == s.Actor.Instantanea.PerfilVersion &&
		x.Operacion() == domain.AccionConsultaRPTPublicaV2 && x.AudienciaConsumo() == domain.AudienciaConsultaRPTPublicaV2 &&
		x.EfectoRef() == m.Recurso().Referencia && x.EfectoHuellaSHA256() == h
}

func decodificarEvidenciaRPTV2(bruto []byte) (ports.EvidenciaLecturaRPTPublicaV2, error) {
	var vacia ports.EvidenciaLecturaRPTPublicaV2
	var raiz map[string]json.RawMessage
	if json.Unmarshal(bruto, &raiz) != nil || len(raiz) != 1 || len(raiz["evidencia"]) == 0 {
		return vacia, domain.ErrRPTPublicaV2NoDisponible
	}
	decodificador := json.NewDecoder(bytes.NewReader(bruto))
	decodificador.DisallowUnknownFields()
	var respuesta struct {
		Evidencia ports.EvidenciaLecturaRPTPublicaV2 `json:"evidencia"`
	}
	if decodificador.Decode(&respuesta) != nil || decodificador.Decode(new(any)) != io.EOF {
		return vacia, domain.ErrRPTPublicaV2NoDisponible
	}
	return respuesta.Evidencia, nil
}

func evidenciaRPTV2PostgresValida(m domain.MaterialConsultaRPTPublicaV2, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, e ports.EvidenciaLecturaRPTPublicaV2) bool {
	x := a.ResumenCapacidad()
	_, zona := e.ConsultadaEn.Zone()
	return e.ReciboRef != "" && len(e.ReciboRef) <= 160 && e.DecisionRef == x.DecisionRef() && e.EfectoRef == x.EfectoRef() &&
		e.EfectoRef == m.Recurso().Referencia && e.AuditoriaRef != "" && len(e.AuditoriaRef) <= 160 &&
		patronHuellaLecturaRPTV2.MatchString(e.ConsumoHuellaSHA256) && !e.ConsultadaEn.IsZero() && zona == 0 &&
		e.ConsultadaEn.Nanosecond()%1000 == 0 && !e.ConsultadaEn.Before(x.EmitidaEn()) && e.ConsultadaEn.Before(x.ExpiraEn())
}

func errorRPTPublicaV2Postgres(ctx context.Context, err error) error {
	if err == nil {
		return domain.ErrRPTPublicaV2NoDisponible
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501", "P0002":
			return domain.ErrRPTPublicaV2Denegada
		case "22023":
			return domain.ErrRPTPublicaV2Invalida
		}
	}
	return domain.ErrRPTPublicaV2NoDisponible
}

func nuloRPTPublicaV2Postgres(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	return r.Kind() == reflect.Pointer && r.IsNil()
}

var _ ports.ConsumidorLecturaRPTPublicaV2 = (*ConsumidorRPTPublicaV2PostgreSQL)(nil)
