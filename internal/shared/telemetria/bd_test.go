package telemetria

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOperacionSinValores(t *testing.T) {
	for sql, want := range map[string]string{
		"SELECT * FROM vec_bolsa.listar_participaciones($1, $2)": "vec_bolsa.listar_participaciones",
		"select vec_ct.consultar($1)":                            "vec_ct.consultar",
		"INSERT INTO vec_auditoria.intentos (a) VALUES ($1)":     "vec_auditoria.intentos",
		"SELECT 'juan.perez(' , x FROM s.t":                      "s.t",
		"/* datos.de_juan( */ begin":                             "begin",
		"COMMIT":                                                 "commit",
		"":                                                       "sql",
	} {
		if got := operacion(sql); got != want {
			t.Errorf("operacion(%q) = %q, se esperaba %q", sql, got, want)
		}
	}
}

func TestAccesoCuentaConsultasYMarcaLaPeticionN1(t *testing.T) {
	var b bytes.Buffer
	o := opciones(&b)
	o.Consultas = 20
	tr := trazador{}
	h := Middleware(o, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		c := tr.TraceAcquireStart(ctx, nil, pgxpool.TraceAcquireStartData{})
		tr.TraceAcquireEnd(c, nil, pgxpool.TraceAcquireEndData{})
		c = tr.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT * FROM vec_bolsa.listar($1)"})
		time.Sleep(2 * time.Millisecond)
		tr.TraceQueryEnd(c, nil, pgx.TraceQueryEndData{})
		for i := 0; i < 20; i++ {
			c = tr.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT vec_bolsa.una($1)"})
			tr.TraceQueryEnd(c, nil, pgx.TraceQueryEndData{})
		}
		c = tr.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT vec_bolsa.una($1)"})
		tr.TraceQueryEnd(c, nil, pgx.TraceQueryEndData{Err: &pgconn.PgError{Code: "57014", Message: "con datos de Juan"}})
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/vec/bolsa", nil))
	l := lineas(t, &b)[0]
	if l["bd_consultas"] != float64(22) || l["lenta"] != true || l["level"] != "WARN" || l["bd_error"] != "bd_57014" ||
		l["consulta_mas_lenta"] != "vec_bolsa.listar" {
		t.Errorf("linea = %v", l)
	}
	if strings.Contains(b.String(), "Juan") {
		t.Error("el texto del error llego al registro")
	}
}

func TestTrazadorFueraDePeticionNoMide(t *testing.T) {
	tr := trazador{}
	ctx := t.Context()
	if tr.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT 1"}) != ctx {
		t.Error("sin peticion se derivo otro contexto")
	}
	cfg, _ := pgxpool.ParseConfig("postgres://u@127.0.0.1:1/db")
	Instrumentar(cfg)
	if _, ok := cfg.ConnConfig.Tracer.(trazador); !ok {
		t.Error("no se instalo el trazador")
	}
	otro, _ := pgxpool.ParseConfig("postgres://u@127.0.0.1:1/db")
	otro.ConnConfig.Tracer = otroTrazador{}
	Instrumentar(otro)
	if _, ok := otro.ConnConfig.Tracer.(trazador); ok {
		t.Error("se sustituyo un trazador existente")
	}
}

func TestDiagnosticoSoloEnBucleLocal(t *testing.T) {
	var avisos bytes.Buffer
	for _, escucha := range []string{"0.0.0.0:9464", ":9464", "localhost:9464", "192.0.2.1:9464", "127.0.0.1:0"} {
		arrancarDiagnostico(escucha, &avisos)()
	}
	if strings.Count(avisos.String(), "no arrancado") != 5 {
		t.Errorf("avisos = %q", avisos.String())
	}
	libre, _ := net.Listen("tcp", "127.0.0.1:0")
	direccion := libre.Addr().String()
	_ = libre.Close()
	cerrar := arrancarDiagnostico(direccion, &avisos)
	defer cerrar()
	r, err := http.Get("http://" + direccion + "/debug/vars")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	cuerpo, _ := io.ReadAll(r.Body)
	var vars map[string]any
	if r.StatusCode != http.StatusOK || json.Unmarshal(cuerpo, &vars) != nil || vars["vec_bd_pools"] == nil || vars["cmdline"] != nil {
		t.Errorf("estado %d, variables %s", r.StatusCode, cuerpo)
	}
	w := httptest.NewRecorder()
	ajena := httptest.NewRequest(http.MethodGet, "/debug/vars", nil)
	ajena.RemoteAddr = "192.0.2.10:5000"
	soloLocal(http.NotFoundHandler()).ServeHTTP(w, ajena)
	if w.Code != http.StatusForbidden {
		t.Errorf("remota no local = %d", w.Code)
	}
}

type otroTrazador struct{}

func (otroTrazador) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	return ctx
}
func (otroTrazador) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}
