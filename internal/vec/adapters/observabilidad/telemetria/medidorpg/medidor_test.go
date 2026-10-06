package medidorpg

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/adapters/observabilidad/telemetria"
)

func TestOperacionSQL(t *testing.T) {
	casos := map[string]string{
		"SELECT * FROM vec_bolsa.listar_participaciones($1, $2)":    "vec_bolsa.listar_participaciones",
		"select vec_ct.consultar_expediente($1)":                    "vec_ct.consultar_expediente",
		"SELECT t.col FROM esquema.tabla t WHERE t.id = $1":         "esquema.tabla",
		"INSERT INTO vec_auditoria.intentos (a, b) VALUES ($1, $2)": "vec_auditoria.intentos",
		"UPDATE s.t SET x = 'juan.perez' WHERE id = $1":             "s.t",
		"WITH x AS (SELECT a FROM s.base) SELECT * FROM x":          "s.base",
		"begin":                                      "begin",
		"COMMIT":                                     "commit",
		"SET LOCAL statement_timeout = '5s'":         "set",
		"SELECT 'Juan Pérez', $1":                    "select",
		"SELECT \"Juan\".\"x\" FROM \"Pérez\"":       "select",
		"-- comentario con datos.de_juan(\nSELECT 1": "select",
		"/* datos.de_juan( */ SELECT $$ a.b( $$":     "select",
		"SELECT count(*) FROM tabla_simple":          "tabla_simple",
		"CALL vec.procedimiento($1)":                 "vec.procedimiento",
		"":                                           "sql",
		"12345678Z":                                  "sql",
	}
	for sql, esperada := range casos {
		if got := operacionSQL(sql); got != esperada {
			t.Errorf("operacionSQL(%q) = %q, se esperaba %q", sql, got, esperada)
		}
	}
}

type relojPaso struct{ t time.Time }

func (r *relojPaso) ahora() time.Time {
	r.t = r.t.Add(10 * time.Millisecond)
	return r.t
}

func TestTrazadorAnotaConsultasEsperasYErroresEnLaFicha(t *testing.T) {
	r := &relojPaso{t: time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC)}
	tr := &Trazador{reloj: r.ahora}
	ctx, f := telemetria.IniciarFicha(context.Background())

	c := tr.TraceAcquireStart(ctx, nil, pgxpool.TraceAcquireStartData{})
	tr.TraceAcquireEnd(c, nil, pgxpool.TraceAcquireEndData{})
	for i := 0; i < 3; i++ {
		c = tr.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT * FROM vec_bolsa.f($1)"})
		tr.TraceQueryEnd(c, nil, pgx.TraceQueryEndData{})
	}
	c = tr.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT vec_ct.g($1)"})
	tr.TraceQueryEnd(c, nil, pgx.TraceQueryEndData{Err: &pgconn.PgError{Code: "57014", Message: "con datos"}})
	b := &pgx.Batch{}
	b.Queue("INSERT INTO s.t VALUES ($1)", 1)
	b.Queue("INSERT INTO s.t VALUES ($1)", 2)
	c = tr.TraceBatchStart(ctx, nil, pgx.TraceBatchStartData{Batch: b})
	tr.TraceBatchEnd(c, nil, pgx.TraceBatchEndData{})

	m := f.ResumenBD()
	if m.Consultas != 6 || m.Esperas != 1 || m.Error != "bd_57014" {
		t.Fatalf("medida = %+v", m)
	}
	if m.Operaciones["vec_bolsa.f"] != 3 || m.Operaciones["vec_ct.g"] != 1 || m.Operaciones["s.t"] != 2 {
		t.Errorf("operaciones = %v", m.Operaciones)
	}
	if m.Total != 50*time.Millisecond || m.Espera != 10*time.Millisecond {
		t.Errorf("tiempos = %v espera %v", m.Total, m.Espera)
	}
}

func TestTrazadorSinFichaNiMetricasNoFalla(t *testing.T) {
	tr := &Trazador{reloj: time.Now}
	ctx := context.Background()
	c := tr.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT 1"})
	tr.TraceQueryEnd(c, nil, pgx.TraceQueryEndData{Err: errors.New("x")})
	tr.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})
	if got := tr.TraceAcquireStart(ctx, nil, pgxpool.TraceAcquireStartData{}); got != ctx {
		t.Error("sin ficha la espera derivo otro contexto")
	}
}

func TestInstrumentarNoSustituyeOtroTrazador(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://u@localhost/db")
	if err != nil {
		t.Fatal(err)
	}
	if !Instrumentar(cfg) || cfg.ConnConfig.Tracer != trazadorComun {
		t.Fatal("no se instalo el trazador")
	}
	otro, _ := pgxpool.ParseConfig("postgres://u@localhost/db")
	otro.ConnConfig.Tracer = &Trazador{}
	if Instrumentar(otro) {
		t.Error("se sustituyo un trazador existente")
	}
	if Instrumentar(nil) {
		t.Error("nil instrumentado")
	}
}

func TestMetricasBDPorOperacionErrorYPool(t *testing.T) {
	m := &metricasBD{}
	tr := &Trazador{reloj: (&relojPaso{t: time.Unix(0, 0)}).ahora, metricas: m}
	ctx := context.Background() // sin petición: también se mide
	for i := 0; i < 2; i++ {
		c := tr.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT vec_bolsa.f($1)"})
		tr.TraceQueryEnd(c, nil, pgx.TraceQueryEndData{Err: &pgconn.PgError{Code: "40001"}})
	}
	cfg, _ := pgxpool.ParseConfig("postgres://vec_bolsa_ejecutor@127.0.0.1:1/db")
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if tr.nombrePool(pool) != "vec_bolsa_ejecutor" {
		t.Fatal("nombre de pool")
	}
	var b strings.Builder
	m.escribir(&b, "vec-server")
	for _, esperado := range []string{
		`vec_bd_consultas_total{servicio="vec-server",operacion="vec_bolsa.f"} 2`,
		`vec_bd_consultas_segundos_total{servicio="vec-server",operacion="vec_bolsa.f"} 0.02`,
		`vec_bd_errores_total{servicio="vec-server",clase="bd_40001"} 2`,
		`vec_pool_conexiones_en_uso{servicio="vec-server",pool="vec_bolsa_ejecutor"} 0`,
		`vec_pool_prestamos_con_espera_total{servicio="vec-server",pool="vec_bolsa_ejecutor"} 0`,
	} {
		if !strings.Contains(b.String(), esperado) {
			t.Errorf("falta %q en\n%s", esperado, b.String())
		}
	}
}
