package auditoriaconsulta

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/shared/plazoarranque"
	"vec-diputacion-granada/internal/vec/auditoria"
)

// Iniciador recibe exclusivamente el pool nominal CT de consultas RRHH.
// La función SQL comprueba además el LOGIN, la pertenencia de grupo, la
// transacción y el consumo de AD3-91 antes de leer historia.
type Iniciador interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type Fuente struct{ pool Iniciador }

var _ auditoria.FuenteAuditoria = (*Fuente)(nil)

func NuevaFuente(pool Iniciador) (*Fuente, error) {
	if pool == nil || reflect.ValueOf(pool).Kind() == reflect.Ptr && reflect.ValueOf(pool).IsNil() {
		return nil, auditoria.ErrNoDisponible
	}
	return &Fuente{pool: pool}, nil
}

const consulta = `SELECT id,fuente,modulo_id,accion,actor_ref,resultado,
       expediente_ref,recibo_ref,antes_sha256,despues_sha256,motivo,
       ocurrido_en,antes,despues,datos_disponibles
  FROM vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(
       $1::text,$2::text,$3::text,$4::timestamptz,$5::timestamptz,$6::integer,
       $7::timestamptz,$8::text,$9::text,$10::text,$11::text,
       $12::bytea,$13::bytea,$14::bytea,$15::bytea,$16::numeric,$17::numeric,
       $18::bytea,$19::bytea,$20::bytea,$21::bytea)`

var patronSHA = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (f *Fuente) ConsultarAuditoria(ctx context.Context, q auditoria.ConsultaAutorizada) (auditoria.PaginaFuente, error) {
	var vacia auditoria.PaginaFuente
	if ctx == nil || f == nil || f.pool == nil {
		return vacia, auditoria.ErrNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	if q.Filtro.Fuente != "ct" || auditoria.ValidarConsultaAutorizada(q) != nil {
		return vacia, auditoria.ErrDenegada
	}
	m := q.Material
	antes := any(nil)
	if !q.Filtro.Antes.OcurridoEn.IsZero() {
		antes = q.Filtro.Antes.OcurridoEn
	}
	tx, err := f.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacia, normalizar(ctx, err)
	}
	defer func() {
		// El contexto HTTP puede quedar cancelado durante Scan; liberar la
		// transacción no depende de que el cliente siga conectado.
		cancelCtx, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(3*time.Second))
		defer cancel()
		_ = tx.Rollback(cancelCtx)
	}()
	rows, err := tx.Query(ctx, consulta,
		q.Filtro.Fuente, q.Filtro.ExpedienteRef, q.Filtro.ActorRef,
		q.Filtro.Desde, q.Filtro.Hasta, int32(q.Filtro.Limite),
		antes, q.Filtro.Antes.Fuente, q.Filtro.Antes.ID, q.Filtro.FinalidadRef, q.Filtro.MotivoRef,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		int64(m.PersonaVersion()), int64(m.PerfilVersion()), m.PayloadVECAD3(), m.SobreCOSESign1(),
		m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
	if err != nil {
		return vacia, normalizar(ctx, err)
	}
	resultado := auditoria.PaginaFuente{Registros: make([]auditoria.Registro, 0, int(q.Filtro.Limite)+1)}
	for rows.Next() {
		var r auditoria.Registro
		var anterior, nuevo []byte
		if err = rows.Scan(&r.ID, &r.Fuente, &r.ModuloID, &r.Accion, &r.ActorRef, &r.Resultado,
			&r.ExpedienteRef, &r.ReciboRef, &r.AntesSHA256, &r.DespuesSHA256, &r.Motivo,
			&r.OcurridoEn, &anterior, &nuevo, &r.DatosDisponibles); err != nil {
			rows.Close()
			return vacia, normalizar(ctx, err)
		}
		if len(anterior) > 0 && json.Unmarshal(anterior, &r.Antes) != nil ||
			len(nuevo) > 0 && json.Unmarshal(nuevo, &r.Despues) != nil ||
			!registroValido(r, q.Filtro, resultado.Registros) {
			rows.Close()
			return vacia, auditoria.ErrFuenteInvalida
		}
		resultado.Registros = append(resultado.Registros, r)
		if len(resultado.Registros) > int(q.Filtro.Limite)+1 {
			rows.Close()
			return vacia, auditoria.ErrFuenteInvalida
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return vacia, normalizar(ctx, err)
	}
	rows.Close()
	if err = tx.Commit(ctx); err != nil {
		return vacia, normalizar(ctx, err)
	}
	return resultado, nil
}

func registroValido(r auditoria.Registro, f auditoria.Filtro, previos []auditoria.Registro) bool {
	if f.Fuente != "ct" || r.Fuente != f.Fuente || r.ModuloID != "contratacion_temporal" ||
		r.ID == "" || r.Accion == "" || r.ExpedienteRef != f.ExpedienteRef ||
		r.Resultado == "" || r.OcurridoEn.Location() != time.UTC ||
		r.OcurridoEn.Nanosecond()%1000 != 0 || r.OcurridoEn.Before(f.Desde) ||
		!r.OcurridoEn.Before(f.Hasta) ||
		(f.ActorRef != "" && r.ActorRef != f.ActorRef) ||
		(r.AntesSHA256 != "" && !patronSHA.MatchString(r.AntesSHA256)) ||
		!patronSHA.MatchString(r.DespuesSHA256) || !r.DatosDisponibles ||
		(r.AntesSHA256 == "") != (r.Antes == nil) ||
		!estadoCTValido(r.Despues) || (r.Antes != nil && !estadoCTValido(r.Antes)) {
		return false
	}
	if len(previos) > 0 {
		prev := previos[len(previos)-1]
		if !r.OcurridoEn.Before(prev.OcurridoEn) &&
			!(r.OcurridoEn.Equal(prev.OcurridoEn) && r.ID < prev.ID) {
			return false
		}
	}
	if !f.Antes.OcurridoEn.IsZero() {
		if !r.OcurridoEn.Before(f.Antes.OcurridoEn) &&
			!(r.OcurridoEn.Equal(f.Antes.OcurridoEn) &&
				("ct" < f.Antes.Fuente || "ct" == f.Antes.Fuente && r.ID < f.Antes.ID)) {
			return false
		}
	}
	return true
}

// La proyección SQL tiene una lista cerrada; un JSON de agregado o un campo
// personal añadido por error debe invalidar toda la respuesta antes del COMMIT.
func estadoCTValido(m map[string]string) bool {
	if len(m) != 2 || m["fase"] == "" || m["estado"] == "" {
		return false
	}
	for k, v := range m {
		if k != "fase" && k != "estado" || len(v) > 80 || !regexpClave.MatchString(v) {
			return false
		}
	}
	return true
}

var regexpClave = regexp.MustCompile(`^[a-z][a-z0-9_]{0,79}$`)

func normalizar(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "42501" {
		return auditoria.ErrDenegada
	}
	return auditoria.ErrNoDisponible
}
