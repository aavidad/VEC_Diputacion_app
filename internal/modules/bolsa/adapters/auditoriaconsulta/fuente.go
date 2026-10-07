package auditoriaconsulta

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/shared/plazoarranque"
	"vec-diputacion-granada/internal/vec/auditoria"
)

const consultaSQL = `SELECT id,ocurrido_en,accion,actor_ref,resultado,expediente_ref,recibo_ref,motivo,campo,valor_anterior,valor_nuevo
FROM vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(
$1::text,$2::text,$3::timestamptz,$4::timestamptz,$5::timestamptz,$6::text,$7::text,$8::integer,
$9::text,$10::text,$11::text,$12::text,$13::bytea,$14::bytea,$15::bytea,$16::bytea,
$17::numeric,$18::numeric,$19::bytea,$20::bytea,$21::bytea,$22::bytea)`

type Fuente struct{ pool *pgxpool.Pool }

var _ auditoria.FuenteAuditoria = (*Fuente)(nil)

func NuevaFuente(pool *pgxpool.Pool) (*Fuente, error) {
	if pool == nil {
		return nil, auditoria.ErrNoDisponible
	}
	return &Fuente{pool: pool}, nil
}

// ConsultarAuditoria consume una decisión nominal de lectura dentro de la
// transacción de Bolsa. SQL sólo puede devolver la participación autorizada.
func (f *Fuente) ConsultarAuditoria(ctx context.Context, q auditoria.ConsultaAutorizada) (pagina auditoria.PaginaFuente, fallo error) {
	vacio := auditoria.PaginaFuente{}
	if f == nil || f.pool == nil || ctx == nil {
		return vacio, auditoria.ErrNoDisponible
	}
	if q.Filtro.Fuente != "bolsa" {
		return vacio, auditoria.ErrDenegada
	}
	if err := auditoria.ValidarConsultaAutorizada(q); err != nil {
		return vacio, err
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}

	tx, err := f.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, errorConsulta(ctx, err)
	}
	defer cerrarTransaccionConsulta(ctx, tx, &pagina, &fallo)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true), set_config('row_security','on',true), set_config('timezone','UTC',true), set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return vacio, errorConsulta(ctx, err)
	}

	fil := q.Filtro
	h, _ := auditoria.HuellaFiltro(fil)
	datos, _ := q.Solicitud.Datos()
	vinculo, _ := datos.VinculoAutenticacionActor.Datos()
	var actorFiltro, antesFuente, antesID *string
	var antesInstante *time.Time
	if fil.ActorRef != "" {
		actorFiltro = &fil.ActorRef
	}
	if !fil.Antes.OcurridoEn.IsZero() {
		antesFuente, antesID = &fil.Antes.Fuente, &fil.Antes.ID
		antes := fil.Antes.OcurridoEn.UTC()
		antesInstante = &antes
	}
	m := q.Material
	rows, err := tx.Query(ctx, consultaSQL,
		fil.ExpedienteRef, actorFiltro, fil.Desde.UTC(), fil.Hasta.UTC(), antesInstante, antesFuente, antesID, int(fil.Limite),
		vinculo.PrincipalID, fil.FinalidadRef, fil.MotivoRef, h,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		return vacio, errorConsulta(ctx, err)
	}
	salida := auditoria.PaginaFuente{Registros: make([]auditoria.Registro, 0, int(fil.Limite)+1)}
	for rows.Next() {
		var fila filaSQL
		if err = rows.Scan(&fila.ID, &fila.OcurridoEn, &fila.Accion, &fila.ActorRef, &fila.Resultado,
			&fila.ExpedienteRef, &fila.ReciboRef, &fila.Motivo, &fila.Campo, &fila.Anterior, &fila.Nuevo); err != nil {
			return vacio, errorConsulta(ctx, err)
		}
		registro, ok := proyectar(fila, fil)
		if !ok {
			return vacio, auditoria.ErrFuenteInvalida
		}
		salida.Registros = append(salida.Registros, registro)
		if len(salida.Registros) > int(fil.Limite)+1 {
			return vacio, auditoria.ErrFuenteInvalida
		}
	}
	if err = rows.Err(); err != nil {
		return vacio, errorConsulta(ctx, err)
	}
	rows.Close()
	if err = tx.Commit(ctx); err != nil {
		return vacio, errorConsulta(ctx, err)
	}
	return salida, nil
}

func cerrarTransaccionConsulta(ctx context.Context, tx pgx.Tx, pagina *auditoria.PaginaFuente, fallo *error) {
	// Esperar el rollback antes de devolver el fallo a la auditoría común.
	cierreCtx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(3*time.Second))
	defer cancelar()
	if err := tx.Rollback(cierreCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		*pagina = auditoria.PaginaFuente{}
		*fallo = auditoria.ErrNoDisponible
	}
}

type filaSQL struct {
	ID, Accion, ActorRef, Resultado, ExpedienteRef, ReciboRef string
	OcurridoEn                                                time.Time
	Motivo, Campo, Anterior, Nuevo                            *string
}

func proyectar(f filaSQL, filtro auditoria.Filtro) (auditoria.Registro, bool) {
	if f.ID == "" || f.Accion == "" || f.ActorRef == "" || f.Resultado != "confirmado" ||
		f.ExpedienteRef != filtro.ExpedienteRef || f.ReciboRef == "" || f.OcurridoEn.IsZero() ||
		f.OcurridoEn.Before(filtro.Desde) || !f.OcurridoEn.Before(filtro.Hasta) ||
		(filtro.ActorRef != "" && f.ActorRef != filtro.ActorRef) {
		return auditoria.Registro{}, false
	}
	r := auditoria.Registro{ID: f.ID, Fuente: "bolsa", ModuloID: "bolsa", Accion: f.Accion,
		ActorRef: f.ActorRef, OcurridoEn: f.OcurridoEn.UTC(), Resultado: f.Resultado,
		ExpedienteRef: f.ExpedienteRef, ReciboRef: f.ReciboRef}
	if f.Motivo != nil {
		if *f.Motivo != "Constitución de bolsa" && *f.Motivo != "Motivo reservado en Bolsa" {
			return auditoria.Registro{}, false
		}
		r.Motivo = *f.Motivo
	}
	if f.Campo != nil {
		if !campoMinimizado(*f.Campo, f.Anterior, f.Nuevo) {
			return auditoria.Registro{}, false
		}
		r.Antes = map[string]string{}
		r.Despues = map[string]string{}
		if f.Anterior != nil {
			r.Antes[*f.Campo] = *f.Anterior
		}
		if f.Nuevo != nil {
			r.Despues[*f.Campo] = *f.Nuevo
		}
		r.DatosDisponibles = true
	}
	return r, true
}

func campoMinimizado(campo string, anterior, nuevo *string) bool {
	if anterior == nil && nuevo == nil {
		return false
	}
	switch campo {
	case "situacion":
		return (anterior == nil || situacionValida(*anterior)) && (nuevo == nil || situacionValida(*nuevo))
	case "fecha_disponible":
		return (anterior == nil || fechaTrazaValida(*anterior)) && (nuevo == nil || fechaTrazaValida(*nuevo))
	case "datos_contacto", "correo", "telefono_1", "telefono_2":
		return (anterior == nil || versionCifradaValida(*anterior)) && (nuevo == nil || versionCifradaValida(*nuevo))
	default:
		return false
	}
}

func situacionValida(s string) bool {
	switch s {
	case "disponible", "no_disponible", "trabajando", "pendiente_incorporacion", "renuncia", "excluido", "disponible_desde":
		return true
	default:
		return false
	}
}

func fechaTrazaValida(s string) bool {
	t, err := time.Parse("2006-01-02T15:04:05.000000Z", s)
	return err == nil && t.UTC().Format("2006-01-02T15:04:05.000000Z") == s
}

func versionCifradaValida(s string) bool {
	if len(s) < len("version:1") || len(s) > len("version:")+19 || s[:len("version:")] != "version:" || s[len("version:")] == '0' {
		return false
	}
	for _, c := range s[len("version:"):] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func errorConsulta(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42501" {
		return auditoria.ErrDenegada
	}
	return auditoria.ErrNoDisponible
}
