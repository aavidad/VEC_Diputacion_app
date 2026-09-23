package postgres

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type txMarcajePrueba struct {
	pgx.Tx
	commitErr, rollbackErr                error
	commits, rollbacks                    int
	efectosProvisionales, efectosDurables int
}

func (t *txMarcajePrueba) Commit(context.Context) error {
	t.commits++
	if t.commitErr == nil {
		t.efectosDurables += t.efectosProvisionales
		t.efectosProvisionales = 0
	}
	return t.commitErr
}
func (t *txMarcajePrueba) Rollback(ctx context.Context) error {
	t.rollbacks++
	if ctx.Err() != nil {
		panic("rollback hereda cancelacion")
	}
	if t.rollbackErr == nil {
		t.efectosProvisionales = 0
	}
	return t.rollbackErr
}

type dbMarcajePrueba struct {
	tx  *txMarcajePrueba
	err error
}

func (d dbMarcajePrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	if o.IsoLevel != pgx.Serializable || o.AccessMode != pgx.ReadWrite {
		panic("aislamiento alterado")
	}
	return d.tx, d.err
}

type auditoriaMarcajePrueba struct {
	t       *testing.T
	tx      *txMarcajePrueba
	eventos []ports.ResultadoEjecucionMarcaje
	err     error
}

func (a *auditoriaMarcajePrueba) RegistrarResultadoEjecucionMarcaje(ctx context.Context, evento ports.ResultadoEjecucionMarcaje) error {
	a.t.Helper()
	if ctx.Err() != nil {
		a.t.Fatal("auditoria hereda cancelacion")
	}
	if _, ok := ctx.Deadline(); !ok {
		a.t.Fatal("auditoria sin plazo")
	}
	if a.tx != nil && a.tx.commits == 0 && a.tx.rollbacks == 0 {
		a.t.Fatal("auditoria antes de terminar negocio")
	}
	if evento.DecisionRef != "decision:cronos:prueba" || evento.ContextoRef != "contexto:cronos:prueba" ||
		evento.Accion != "cronos.marcaje.propio.registrar" || evento.RecursoRef != "marcaje:cronos:prueba-0001" ||
		evento.ActorRef != "per_0123456789abcdefghijkl" || evento.PerfilRef != "prf_0123456789abcdefghijkl" || evento.ObservadaEn.IsZero() {
		a.t.Fatal("correlacion perdida")
	}
	a.eventos = append(a.eventos, evento)
	return a.err
}
func eventoMarcajePrueba() ports.ResultadoEjecucionMarcaje {
	return ports.ResultadoEjecucionMarcaje{
		DecisionRef: "decision:cronos:prueba", ContextoRef: "contexto:cronos:prueba",
		ActorRef: "per_0123456789abcdefghijkl", PerfilRef: "prf_0123456789abcdefghijkl",
		Accion: "cronos.marcaje.propio.registrar", RecursoRef: "marcaje:cronos:prueba-0001",
	}
}

func TestFalloMarcajeAuditaDespuesDeRollbackSinEfectos(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		err      error
		causa    string
		esperado error
	}{
		{"conflicto", ports.ErrClaveOperacionEnConflicto, "conflicto", ports.ErrClaveOperacionEnConflicto},
		{"sql", ports.ErrDependenciaNoDisponible, "persistencia", ports.ErrDependenciaNoDisponible},
		{"recibo", errReciboMarcajeInvalido, "recibo_invalido", ports.ErrDependenciaNoDisponible},
		{"cancelacion", context.Canceled, "persistencia", context.Canceled},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &txMarcajePrueba{}
			audit := &auditoriaMarcajePrueba{t: t, tx: tx}
			r := &RepositorioMarcajes{db: dbMarcajePrueba{tx: tx}, auditoria: audit}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, err := r.ejecutarTransaccionMarcaje(ctx, eventoMarcajePrueba(), func(pgx.Tx) (ports.ReciboMarcajePropio, error) {
				tx.efectosProvisionales = 1
				if caso.nombre == "cancelacion" {
					cancel()
				}
				return ports.ReciboMarcajePropio{}, caso.err
			})
			if !errors.Is(err, caso.esperado) || tx.rollbacks != 1 || tx.commits != 0 || tx.efectosProvisionales != 0 || tx.efectosDurables != 0 {
				t.Fatalf("fallo no revierte: err=%v tx=%+v", err, tx)
			}
			if len(audit.eventos) != 1 || audit.eventos[0].Resultado != "fallo_confirmado" || audit.eventos[0].Causa != caso.causa {
				t.Fatal("fallo sin evidencia independiente")
			}
		})
	}
}
func TestMarcajeClasificaCommitYRollbackSinAfirmarRechazoAmbiguo(t *testing.T) {
	for _, caso := range []struct {
		nombre                    string
		commit, rollback, aplicar error
		resultado, causa          string
		esperado                  error
	}{
		{"commit sin respuesta", errors.New("respuesta perdida privada"), nil, nil, "resultado_indeterminado", "commit", ports.ErrResultadoMarcajeIndeterminado},
		{"commit cancelado", context.Canceled, nil, nil, "resultado_indeterminado", "commit", ports.ErrResultadoMarcajeIndeterminado},
		{"servidor caido", &pgconn.PgError{Code: "57P01", Severity: "FATAL", Message: "privado"}, nil, nil, "resultado_indeterminado", "commit", ports.ErrResultadoMarcajeIndeterminado},
		{"serializacion", &pgconn.PgError{Code: "40001", Severity: "ERROR", Message: "privado"}, nil, nil, "fallo_confirmado", "commit", ports.ErrDependenciaNoDisponible},
		{"commit convertido rollback", pgx.ErrTxCommitRollback, nil, nil, "fallo_confirmado", "commit", ports.ErrDependenciaNoDisponible},
		{"rollback sin respuesta", nil, errors.New("red privada"), ports.ErrClaveOperacionEnConflicto, "resultado_indeterminado", "rollback", ports.ErrResultadoMarcajeIndeterminado},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &txMarcajePrueba{commitErr: caso.commit, rollbackErr: caso.rollback}
			audit := &auditoriaMarcajePrueba{t: t, tx: tx}
			r := &RepositorioMarcajes{db: dbMarcajePrueba{tx: tx}, auditoria: audit}
			_, err := r.ejecutarTransaccionMarcaje(context.Background(), eventoMarcajePrueba(), func(pgx.Tx) (ports.ReciboMarcajePropio, error) { return ports.ReciboMarcajePropio{}, caso.aplicar })
			if !errors.Is(err, caso.esperado) || strings.Contains(err.Error(), "privad") {
				t.Fatalf("clasificacion o filtracion: %v", err)
			}
			if len(audit.eventos) != 1 || audit.eventos[0].Resultado != caso.resultado || audit.eventos[0].Causa != caso.causa {
				t.Fatal(audit.eventos)
			}
			if caso.aplicar == nil && tx.rollbacks != 0 {
				t.Fatal("rollback posterior a commit no acredita resultado")
			}
		})
	}
}
func TestFalloAuditoriaMarcajeNoSeOculta(t *testing.T) {
	for _, ambiguo := range []bool{false, true} {
		tx := &txMarcajePrueba{}
		if ambiguo {
			tx.commitErr = errors.New("red")
		}
		audit := &auditoriaMarcajePrueba{t: t, tx: tx, err: errors.New("destino privado")}
		r := &RepositorioMarcajes{db: dbMarcajePrueba{tx: tx}, auditoria: audit}
		_, err := r.ejecutarTransaccionMarcaje(context.Background(), eventoMarcajePrueba(), func(pgx.Tx) (ports.ReciboMarcajePropio, error) {
			if ambiguo {
				return ports.ReciboMarcajePropio{}, nil
			}
			return ports.ReciboMarcajePropio{}, ports.ErrClaveOperacionEnConflicto
		})
		if !errors.Is(err, ports.ErrAuditoriaMarcajeNoDisponible) || errors.Is(err, ports.ErrResultadoMarcajeIndeterminado) != ambiguo || strings.Contains(err.Error(), "privado") {
			t.Fatal(err)
		}
	}
}
func TestMarcajeExitosoYReplayConservanReciboSinAuditoriaDeFallo(t *testing.T) {
	for _, replay := range []bool{false, true} {
		tx := &txMarcajePrueba{}
		audit := &auditoriaMarcajePrueba{t: t, tx: tx}
		r := &RepositorioMarcajes{db: dbMarcajePrueba{tx: tx}, auditoria: audit}
		esperado := ports.ReciboMarcajePropio{Referencia: "recibo:cronos:original", MarcajeOriginalRef: "marcaje:cronos:prueba-0001", InstanteUTC: time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC), Replay: replay}
		recibido, err := r.ejecutarTransaccionMarcaje(context.Background(), eventoMarcajePrueba(), func(pgx.Tx) (ports.ReciboMarcajePropio, error) {
			if !replay {
				tx.efectosProvisionales = 1
			}
			return esperado, nil
		})
		if err != nil || recibido != esperado || tx.commits != 1 || tx.rollbacks != 0 || len(audit.eventos) != 0 {
			t.Fatal("recibo o resultado alterados")
		}
		if replay && tx.efectosDurables != 0 {
			t.Fatal("replay duplica efecto")
		}
	}
}
func TestMarcajeSinTransaccionYConstructorSinAuditoria(t *testing.T) {
	if _, err := NuevoRepositorioMarcajes(nil, nil); err == nil {
		t.Fatal("constructor abierto")
	}
	audit := &auditoriaMarcajePrueba{t: t}
	r := &RepositorioMarcajes{db: dbMarcajePrueba{err: errors.New("conexion privada")}, auditoria: audit}
	_, err := r.ejecutarTransaccionMarcaje(context.Background(), eventoMarcajePrueba(), func(pgx.Tx) (ports.ReciboMarcajePropio, error) {
		t.Fatal("aplica sin transaccion")
		return ports.ReciboMarcajePropio{}, nil
	})
	if !errors.Is(err, ports.ErrDependenciaNoDisponible) || len(audit.eventos) != 1 || audit.eventos[0].Resultado != "fallo_confirmado" {
		t.Fatal(err)
	}
}

// Guarda estructural, NO sustituye la campaña real de PostgreSQL con AD3/COSE.
func TestContratoSQLMarcajeEsperaAntesDeV3YRevalidaAmbosResultados(t *testing.T) {
	b, err := os.ReadFile("../../../../../deploy/postgresql/cronos_v1/migraciones/000002_registrar_marcaje_propio.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	bloqueo := strings.Index(s, "PERFORM pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:clave:'")
	consumo := strings.Index(s, "SELECT * INTO STRICT consumo")
	vigente := strings.Index(s, "-- El consumidor mantiene sus bloqueos")
	lectura := strings.Index(s, "SELECT * INTO previa")
	if bloqueo < 0 || !(bloqueo < consumo && consumo < vigente && vigente < lectura) {
		t.Fatal("espera o lectura fuera de la frontera V3")
	}
	if strings.Count(s, "IF clock_timestamp() >= vence_en THEN") != 2 {
		t.Fatal("alta o replay sin vigencia final")
	}
	for _, campo := range []string{"expira_en", "decision_valida_hasta", "configuracion_expira_en", "raiz_valida_hasta"} {
		if !strings.Contains(s, "ahora < (c->>'"+campo+"')::timestamptz") {
			t.Fatalf("falta vigencia %s", campo)
		}
	}
	if !strings.Contains(s, "ahora < (vinculo->>'vigente_hasta')::timestamptz) IS NOT TRUE") {
		t.Fatal("vinculo empleado no revalidado fail-closed")
	}
}
