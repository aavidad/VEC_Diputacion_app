package postgres

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/vec/ports"
)

func datosResultadoEjecucionPGPrueba() ports.DatosResultadoEjecucionAutorizada {
	return ports.DatosResultadoEjecucionAutorizada{
		InformeRef: "inf_ejec_" + strings.Repeat("a", 32), PerfilConsumidor: ports.ResultadoBorradorDietas,
		DecisionRef: "dec_" + strings.Repeat("b", 32), DecisionHuellaSHA256: strings.Repeat("c", 64),
		ContextoRef: "rca_" + strings.Repeat("d", 32), ContextoHuellaSHA256: strings.Repeat("e", 64),
		ActorRef: "per_" + strings.Repeat("f", 32), PerfilRef: "prf_" + strings.Repeat("g", 32),
		CorrelacionRef: "correlacion_" + strings.Repeat("1", 32), Accion: "dietas.borrador.crear_propio",
		RecursoRef: "dietas:borrador:" + strings.Repeat("h", 32), RecursoHuellaSHA256: strings.Repeat("2", 64),
		Resultado: "fallo_confirmado", Etapa: "transaccion", Causa: "persistencia",
	}
}

type filasResultadoEjecucionPrueba struct {
	pgx.Rows
	recibo           ports.ReciboResultadoEjecucionAutorizada
	indice, cantidad int
	err, errorScan   error
	cerradas         bool
}

func (f *filasResultadoEjecucionPrueba) Next() bool { f.indice++; return f.indice <= f.cantidad }
func (f *filasResultadoEjecucionPrueba) Err() error { return f.err }
func (f *filasResultadoEjecucionPrueba) Close()     { f.cerradas = true }
func (f *filasResultadoEjecucionPrueba) Scan(dest ...any) error {
	if f.errorScan != nil {
		return f.errorScan
	}
	*dest[0].(*string) = f.recibo.InformeRef
	*dest[1].(*string) = f.recibo.ReciboRef
	*dest[2].(*string) = f.recibo.HuellaSHA256
	*dest[3].(*time.Time) = f.recibo.ObservadoEn
	*dest[4].(*bool) = f.recibo.Repeticion
	return nil
}

type txResultadoEjecucionPrueba struct {
	pgx.Tx
	filas                                *filasResultadoEjecucionPrueba
	consulta, configuracion              string
	args                                 []any
	errorConfig, errorQuery, errorCommit error
	commits, rollbacks                   int
	alCommit                             func()
}

func (x *txResultadoEjecucionPrueba) Exec(_ context.Context, q string, _ ...any) (pgconn.CommandTag, error) {
	x.configuracion = q
	return pgconn.CommandTag{}, x.errorConfig
}
func (x *txResultadoEjecucionPrueba) Query(_ context.Context, q string, a ...any) (pgx.Rows, error) {
	x.consulta = q
	x.args = a
	return x.filas, x.errorQuery
}
func (x *txResultadoEjecucionPrueba) Commit(context.Context) error {
	x.commits++
	if x.alCommit != nil {
		x.alCommit()
	}
	return x.errorCommit
}
func (x *txResultadoEjecucionPrueba) Rollback(context.Context) error { x.rollbacks++; return nil }

type poolResultadoEjecucionPrueba struct {
	tx       *txResultadoEjecucionPrueba
	err      error
	inicio   int
	opciones pgx.TxOptions
}

func (p *poolResultadoEjecucionPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.inicio++
	p.opciones = o
	return p.tx, p.err
}
func escenarioResultadoEjecucionPrueba(t *testing.T) (*RegistroResultadoEjecucion, *poolResultadoEjecucionPrueba, ports.InformeResultadoEjecucionAutorizada) {
	t.Helper()
	d := datosResultadoEjecucionPGPrueba()
	i, err := ports.NuevoInformeResultadoEjecucionAutorizada(d)
	if err != nil {
		t.Fatal(err)
	}
	p := &poolResultadoEjecucionPrueba{tx: &txResultadoEjecucionPrueba{filas: &filasResultadoEjecucionPrueba{
		cantidad: 1, recibo: ports.ReciboResultadoEjecucionAutorizada{InformeRef: d.InformeRef, ReciboRef: "rec_ejec_" + strings.Repeat("a", 32), HuellaSHA256: strings.Repeat("b", 64), ObservadoEn: time.Date(2026, 9, 19, 14, 0, 0, 0, time.UTC)},
	}}}
	r, err := nuevoRegistroResultadoEjecucion(p, ports.ResultadoBorradorDietas)
	if err != nil {
		t.Fatal(err)
	}
	return r, p, i
}
func TestResultadoEjecucionPGTransaccionIndependienteYReplay(t *testing.T) {
	for _, replay := range []bool{false, true} {
		t.Run(map[bool]string{false: "nuevo", true: "replay"}[replay], func(t *testing.T) {
			r, p, i := escenarioResultadoEjecucionPrueba(t)
			p.tx.filas.recibo.Repeticion = replay
			got, err := r.RegistrarResultadoEjecucionAutorizada(context.Background(), i)
			if err != nil || got != p.tx.filas.recibo {
				t.Fatal("no confirmó recibo exacto", err)
			}
			if p.inicio != 1 || p.opciones.IsoLevel != pgx.ReadCommitted || p.tx.commits != 1 || !p.tx.filas.cerradas {
				t.Fatal("frontera transaccional incompleta")
			}
			if !strings.Contains(p.tx.consulta, "registrar_resultado_borrador_dietas_v1(") || len(p.tx.args) != 14 {
				t.Fatal("wrapper/contrato SQL incorrecto")
			}
			d, _ := i.Datos()
			esperado := []any{d.InformeRef, d.DecisionRef, d.DecisionHuellaSHA256, d.ContextoRef, d.ContextoHuellaSHA256, d.ActorRef, d.PerfilRef, d.CorrelacionRef, d.Accion, d.RecursoRef, d.RecursoHuellaSHA256, d.Resultado, d.Etapa, d.Causa}
			for n, v := range esperado {
				if p.tx.args[n] != v {
					t.Fatalf("campo %d cruzado", n)
				}
			}
			if !strings.Contains(p.tx.configuracion, "statement_timeout") {
				t.Fatal("sin límite temporal")
			}
		})
	}
}
func TestResultadoEjecucionPGNoEjecutaPerfilCruzadoNiZero(t *testing.T) {
	r, p, i := escenarioResultadoEjecucionPrueba(t)
	r.perfil = ports.ResultadoMarcajeCronos
	if _, err := r.RegistrarResultadoEjecucionAutorizada(context.Background(), i); !errors.Is(err, ports.ErrInformeResultadoEjecucionInvalido) || p.inicio != 0 {
		t.Fatal("perfil cruzado abrió transacción")
	}
	if _, err := r.RegistrarResultadoEjecucionAutorizada(context.Background(), ports.InformeResultadoEjecucionAutorizada{}); err == nil || p.inicio != 0 {
		t.Fatal("zero ejecutó")
	}
	if _, err := NuevoRegistroResultadoRutasDietas(nil); err == nil {
		t.Fatal("pool nil aceptado")
	}
	if _, err := nuevoRegistroResultadoEjecucion(p, "otro"); err == nil {
		t.Fatal("perfil arbitrario")
	}
	for _, perfil := range []ports.PerfilResultadoEjecucion{ports.ResultadoRutasDietas, ports.ResultadoBorradorDietas, ports.ResultadoMarcajeCronos} {
		if consultaResultadoEjecucion(perfil) == "" {
			t.Fatal("perfil sin wrapper")
		}
	}
}
func TestResultadoEjecucionPGFalloPrevioNoEntregaRecibo(t *testing.T) {
	for _, c := range []struct {
		nombre   string
		cambiar  func(*poolResultadoEjecucionPrueba)
		esperado error
	}{
		{"begin", func(p *poolResultadoEjecucionPrueba) { p.err = errors.New("DSN secreto") }, ports.ErrRegistroResultadoEjecucionNoDisponible},
		{"config", func(p *poolResultadoEjecucionPrueba) { p.tx.errorConfig = errors.New("secreto") }, ports.ErrRegistroResultadoEjecucionNoDisponible},
		{"query", func(p *poolResultadoEjecucionPrueba) { p.tx.errorQuery = errors.New("payload secreto") }, ports.ErrRegistroResultadoEjecucionNoDisponible},
		{"conflicto", func(p *poolResultadoEjecucionPrueba) {
			p.tx.errorQuery = &pgconn.PgError{Code: "P4201", Message: "contenido secreto"}
		}, ports.ErrInformeResultadoEjecucionEnConflicto},
		{"scan", func(p *poolResultadoEjecucionPrueba) { p.tx.filas.errorScan = errors.New("secreto") }, ports.ErrRegistroResultadoEjecucionNoDisponible},
		{"sin fila", func(p *poolResultadoEjecucionPrueba) { p.tx.filas.cantidad = 0 }, ports.ErrRegistroResultadoEjecucionNoDisponible},
		{"dos filas", func(p *poolResultadoEjecucionPrueba) { p.tx.filas.cantidad = 2 }, ports.ErrRegistroResultadoEjecucionNoDisponible},
		{"error tardio", func(p *poolResultadoEjecucionPrueba) { p.tx.filas.err = &pgconn.PgError{Code: "P4201"} }, ports.ErrInformeResultadoEjecucionEnConflicto},
		{"informe cruzado", func(p *poolResultadoEjecucionPrueba) { p.tx.filas.recibo.InformeRef = "otro" }, ports.ErrRegistroResultadoEjecucionNoDisponible},
		{"recibo libre", func(p *poolResultadoEjecucionPrueba) { p.tx.filas.recibo.ReciboRef = "persona@example.invalid" }, ports.ErrRegistroResultadoEjecucionNoDisponible},
		{"fecha cliente", func(p *poolResultadoEjecucionPrueba) { p.tx.filas.recibo.ObservadoEn = time.Time{} }, ports.ErrRegistroResultadoEjecucionNoDisponible},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			r, p, i := escenarioResultadoEjecucionPrueba(t)
			c.cambiar(p)
			got, err := r.RegistrarResultadoEjecucionAutorizada(context.Background(), i)
			if !errors.Is(err, c.esperado) || got != (ports.ReciboResultadoEjecucionAutorizada{}) || p.tx.commits != 0 {
				t.Fatal("fallo entregó recibo/commit", err)
			}
			if strings.Contains(err.Error(), "secreto") {
				t.Fatal("error sin redactar")
			}
		})
	}
}
func TestResultadoEjecucionPGCommitAmbiguoYCancelacion(t *testing.T) {
	for _, c := range []struct {
		nombre             string
		err, errorEsperado error
	}{
		{"red", errors.New("respuesta perdida"), ports.ErrRegistroResultadoEjecucionIndeterminado},
		{"cancelado", context.Canceled, ports.ErrRegistroResultadoEjecucionIndeterminado},
		{"fatal", &pgconn.PgError{Severity: "FATAL", Code: "57P01"}, ports.ErrRegistroResultadoEjecucionIndeterminado},
		{"rollback", pgx.ErrTxCommitRollback, ports.ErrRegistroResultadoEjecucionNoDisponible},
		{"sql confirmado", &pgconn.PgError{Severity: "ERROR", Code: "40001"}, ports.ErrRegistroResultadoEjecucionNoDisponible},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			r, p, i := escenarioResultadoEjecucionPrueba(t)
			p.tx.errorCommit = c.err
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			p.tx.alCommit = cancel
			got, err := r.RegistrarResultadoEjecucionAutorizada(ctx, i)
			if !errors.Is(err, c.errorEsperado) || got != (ports.ReciboResultadoEjecucionAutorizada{}) || p.tx.commits != 1 {
				t.Fatal("clasificación de COMMIT incorrecta", err)
			}
		})
	}
	r, p, i := escenarioResultadoEjecucionPrueba(t)
	ctx, cancel := context.WithCancel(context.Background())
	p.tx.alCommit = cancel
	defer cancel()
	if _, err := r.RegistrarResultadoEjecucionAutorizada(ctx, i); err != nil {
		t.Fatal("COMMIT confirmado reinterpretado por cancelación", err)
	}
}
func TestResultadoEjecucionSQLFronteraEstatica(t *testing.T) {
	// No acredita PostgreSQL. Protege únicamente el contrato de la migración
	// contra regresiones antes de las pruebas reales de roles/cadena/rollback.
	raiz := filepath.Join("..", "..", "..", "..", "deploy", "postgresql", "autorizacion_atestada_v3", "migraciones")
	up, err := os.ReadFile(filepath.Join(raiz, "000042_registro_resultado_ejecucion.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(up)
	for _, obligatorio := range []string{
		"decision_concedida_contexto_actor_v3", "concesion_resultado_ejecucion_v1", "SECURITY DEFINER", "SET search_path=pg_catalog",
		"FORCE ROW LEVEL SECURITY", "rechazar_mutacion()", "rechazar_truncado()", "p_huella_decision", "p_huella_contexto", "p_correlacion", "p_huella_recurso",
		"p_actor", "p_perfil", "p_accion", "p_recurso", "a.inherit_option AND NOT a.set_option AND NOT a.admin_option",
		"vec_resultado_rutas_dietas_registro", "vec_resultado_borrador_dietas_registro", "vec_resultado_marcaje_cronos_registro",
		"clock_timestamp()", "contenido_huella_sha256 IS DISTINCT FROM v_huella", "ERRCODE='P4201'",
	} {
		if !strings.Contains(s, obligatorio) {
			t.Fatalf("falta guardia %s", obligatorio)
		}
	}
	for _, prohibido := range []string{"REFERENCES vec_autorizacion_atestada_v3.consumo_decision_v3", "INSERT INTO vec_dietas", "INSERT INTO vec_cronos", "ALTER FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna", "DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check"} {
		if strings.Contains(s, prohibido) {
			t.Fatalf("amplía autoridad/efecto: %s", prohibido)
		}
	}
	down, err := os.ReadFile(filepath.Join(raiz, "000042_registro_resultado_ejecucion.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(down), "CASCADE") || !strings.Contains(string(down), "historia resultado conservada") {
		t.Fatal("DOWN puede destruir dependientes/historia")
	}
}
