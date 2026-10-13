package postgres

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/auditoria"
)

// FuenteAuditoriaRRHHPostgreSQL usa el LOGIN de consultas RRHH segregado.
// Una lectura se confirma sólo junto con su asiento nominal de autorización.
type FuenteAuditoriaRRHHPostgreSQL struct{ pool *pgxpool.Pool }

var _ auditoria.FuenteLecturaSesion = (*FuenteAuditoriaRRHHPostgreSQL)(nil)

func NuevaFuenteAuditoriaRRHHPostgreSQL(pool *pgxpool.Pool) (*FuenteAuditoriaRRHHPostgreSQL, error) {
	if pool == nil {
		return nil, auditoria.ErrNoDisponible
	}
	return &FuenteAuditoriaRRHHPostgreSQL{pool: pool}, nil
}

func errorLecturaAuditoriaRRHH(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) {
		slog.Error("auditoria: lectura PostgreSQL rechazada", "sqlstate", pgerr.Code,
			"restriccion", pgerr.ConstraintName, "rutina", pgerr.Routine)
	} else if err != nil {
		slog.Error("auditoria: conexion PostgreSQL no disponible", "causa", "conexion_o_transaccion")
	}
	if errors.As(err, &pgerr) && (pgerr.Code == "42501" || pgerr.Code == "22023") {
		return auditoria.ErrDenegada
	}
	return auditoria.ErrNoDisponible
}

func (f *FuenteAuditoriaRRHHPostgreSQL) ejecutar(
	ctx context.Context, q auditoria.ConsultaSesion,
	lector func(context.Context, pgx.Tx, string) error,
) error {
	if f == nil || f.pool == nil || ctx == nil || ctx.Err() != nil ||
		q.Filtro.Validar() != nil || len(q.Captura.ContextoCanonico) == 0 ||
		len(q.Captura.VinculoCanonico) == 0 || len(q.Captura.Orden) == 0 ||
		q.Captura.OrganizacionRef == "" {
		return errorLecturaAuditoriaRRHH(ctx, nil)
	}
	tx, err := f.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return errorLecturaAuditoriaRRHH(ctx, err)
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := tx.Rollback(c); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.Error("auditoria: no se pudo cerrar la transaccion", "causa", "rollback_no_disponible")
		}
	}()
	if _, err = tx.Exec(ctx, "SET LOCAL TimeZone='UTC'"); err != nil {
		return errorLecturaAuditoriaRRHH(ctx, err)
	}
	var ref, huella, correlacion string
	var secuencia string
	var registrada time.Time
	err = tx.QueryRow(ctx, `SELECT auditoria_ref,secuencia::text,huella_sha256,correlacion_ref,registrada_en
		FROM vec_autorizacion_atestada_v3.registrar_lectura_auditoria_rrhh_v1($1::bytea,$2::bytea,$3::jsonb)`,
		q.Captura.ContextoCanonico, q.Captura.VinculoCanonico, string(q.Captura.Orden)).Scan(
		&ref, &secuencia, &huella, &correlacion, &registrada)
	if err != nil || ref == "" || secuencia == "" || len(huella) != 64 || correlacion == "" || registrada.IsZero() {
		return errorLecturaAuditoriaRRHH(ctx, err)
	}
	if lector != nil {
		if err = lector(ctx, tx, ref); err != nil {
			return errorLecturaAuditoriaRRHH(ctx, err)
		}
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return errorLecturaAuditoriaRRHH(ctx, err)
	}
	return nil
}

func (f *FuenteAuditoriaRRHHPostgreSQL) RegistrarOpciones(ctx context.Context, q auditoria.ConsultaSesion) error {
	return f.ejecutar(ctx, q, nil)
}

func (f *FuenteAuditoriaRRHHPostgreSQL) BuscarObjetivos(
	ctx context.Context, q auditoria.ConsultaSesion, b auditoria.Busqueda,
) ([]auditoria.Objetivo, error) {
	objetivos := make([]auditoria.Objetivo, 0)
	err := f.ejecutar(ctx, q, func(ctx context.Context, tx pgx.Tx, ref string) error {
		rows, err := tx.Query(ctx, `SELECT referencia,etiqueta
			FROM vec_autorizacion_atestada_v3.buscar_objetivos_auditoria_rrhh_v1(
			$1::text,$2::text,$3::text,$4::text,$5::integer,$6::text)`,
			q.Filtro.Fuente, q.Captura.OrganizacionRef, b.Texto, b.DespuesRef,
			int(q.Filtro.Limite)+1, ref)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var o auditoria.Objetivo
			if err := rows.Scan(&o.Referencia, &o.Etiqueta); err != nil {
				return err
			}
			objetivos = append(objetivos, o)
			if len(objetivos) > int(q.Filtro.Limite)+1 {
				return auditoria.ErrFuenteInvalida
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
		return auditoria.ValidarObjetivosFuenteSesion(q.Filtro.Limite, b.DespuesRef, objetivos)
	})
	if err != nil {
		return nil, err
	}
	return objetivos, nil
}

func (f *FuenteAuditoriaRRHHPostgreSQL) ConsultarAuditoriaSesion(
	ctx context.Context, q auditoria.ConsultaSesion,
) (auditoria.PaginaFuente, error) {
	pagina := auditoria.PaginaFuente{Registros: make([]auditoria.Registro, 0)}
	err := f.ejecutar(ctx, q, func(ctx context.Context, tx pgx.Tx, ref string) error {
		var antesEn *time.Time
		if !q.Filtro.Antes.OcurridoEn.IsZero() {
			antesEn = &q.Filtro.Antes.OcurridoEn
		}
		rows, err := tx.Query(ctx, `SELECT id,fuente,modulo_id,accion,actor_ref,ocurrido_en,resultado,
			expediente_ref,recibo_ref,antes_sha256,despues_sha256,motivo,antes,despues,
			datos_disponibles,perfil_ref,finalidad_ref,canal
			FROM vec_autorizacion_atestada_v3.consultar_registros_auditoria_rrhh_v1(
			$1::text,$2::text,$3::text,$4::timestamptz,$5::timestamptz,$6::timestamptz,
			$7::text,$8::integer,$9::text,$10::text)`,
			q.Filtro.Fuente, q.Captura.OrganizacionRef, q.Filtro.ExpedienteRef,
			q.Filtro.Desde, q.Filtro.Hasta, antesEn, q.Filtro.Antes.ID,
			int(q.Filtro.Limite)+1, q.Filtro.ActorRef, ref)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var r auditoria.Registro
			if err := rows.Scan(&r.ID, &r.Fuente, &r.ModuloID, &r.Accion, &r.ActorRef,
				&r.OcurridoEn, &r.Resultado, &r.ExpedienteRef, &r.ReciboRef,
				&r.AntesSHA256, &r.DespuesSHA256, &r.Motivo, &r.Antes, &r.Despues,
				&r.DatosDisponibles, &r.PerfilRef, &r.FinalidadRef, &r.Canal); err != nil {
				return err
			}
			r.OcurridoEn = r.OcurridoEn.UTC().Truncate(time.Microsecond)
			pagina.Registros = append(pagina.Registros, r)
			if len(pagina.Registros) > int(q.Filtro.Limite)+1 {
				return auditoria.ErrFuenteInvalida
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		for _, r := range pagina.Registros {
			if r.ExpedienteRef != q.Filtro.ExpedienteRef {
				return auditoria.ErrFuenteInvalida
			}
		}
		if err := auditoria.ValidarRegistrosFuenteSesion(q.Filtro, pagina.Registros); err != nil {
			return err
		}
		pagina.AuditoriaRef = ref
		return nil
	})
	if err != nil {
		return auditoria.PaginaFuente{}, err
	}
	return pagina, nil
}
