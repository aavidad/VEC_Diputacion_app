package telemetria

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
		"SELECT * FROM vec_usuarios.catalogo_vigente_preferencias_v1($1)": "vec_usuarios.catalogo_vigente_preferencias_v1",
		"select vec_usuarios.consultar_preferencias_propias_v1($1)":       "vec_usuarios.consultar_preferencias_propias_v1",
		"INSERT INTO vec_auditoria.intentos (a) VALUES ($1)":              "insert",
		"SELECT 'juan.perez(' , x FROM s.t":                               "select",
		"SELECT vec_persona_juan.perez($1)":                               "select",
		"/* datos.de_juan( */ begin":                                      "begin",
		"COMMIT":                                                          "commit",
		"":                                                                "sql",
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
		for _, control := range []string{"begin", "BEGIN ISOLATION LEVEL SERIALIZABLE", "select set_config('vec.x', $1, true)", "SET LOCAL statement_timeout = 1000", "commit"} {
			c = tr.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: control})
			time.Sleep(3 * time.Millisecond) // más lenta que las de datos: no debe nombrarse
			tr.TraceQueryEnd(c, nil, pgx.TraceQueryEndData{})
		}
		c = tr.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT * FROM vec_usuarios.catalogo_vigente_preferencias_v1($1)"})
		time.Sleep(2 * time.Millisecond)
		tr.TraceQueryEnd(c, nil, pgx.TraceQueryEndData{})
		for i := 0; i < 20; i++ {
			c = tr.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT vec_usuarios.consultar_preferencias_propias_v1($1)"})
			tr.TraceQueryEnd(c, nil, pgx.TraceQueryEndData{})
		}
		c = tr.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT vec_usuarios.consultar_preferencias_propias_v1($1)"})
		tr.TraceQueryEnd(c, nil, pgx.TraceQueryEndData{Err: &pgconn.PgError{Code: "57014", Message: "con datos de Juan"}})
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/vec/bolsa", nil))
	l := lineas(t, &b)[0]
	if l["vec.bd.consultas"] != float64(22) || l["vec.lenta"] != true || l["level"] != "WARN" || l["vec.bd.error"] != "bd_57014" ||
		l["vec.bd.consulta_mas_lenta"] != "vec_usuarios.catalogo_vigente_preferencias_v1" {
		t.Errorf("linea = %v", l)
	}
	resumen, ok := l["vec.bd.operaciones"].([]any)
	if !ok || len(resumen) != 2 {
		t.Errorf("resumen por operación = %v", l["vec.bd.operaciones"])
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
	for _, c := range []struct{ remota, host string }{
		{"192.0.2.10:5000", "127.0.0.1:9464"},       // remota no local
		{"127.0.0.1:5000", "atacante.example:9464"}, // rebinding de DNS
		{"127.0.0.1:5000", "localhost:9464"},
	} {
		w := httptest.NewRecorder()
		ajena := httptest.NewRequest(http.MethodGet, "/debug/vars", nil)
		ajena.RemoteAddr, ajena.Host = c.remota, c.host
		soloLocal(http.NotFoundHandler()).ServeHTTP(w, ajena)
		if w.Code != http.StatusForbidden {
			t.Errorf("%+v = %d", c, w.Code)
		}
	}
}

type otroTrazador struct{}

func (otroTrazador) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	return ctx
}
func (otroTrazador) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func TestErrorTypeSoloDeLaOperacionQueFallo(t *testing.T) {
	var b bytes.Buffer
	tr := trazador{}
	h := Middleware(opciones(&b), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// Un conflicto ya manejado (reintento) seguido de una consulta correcta.
		c := tr.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT vec_ct.f($1)"})
		tr.TraceQueryEnd(c, nil, pgx.TraceQueryEndData{Err: &pgconn.PgError{Code: "40001"}})
		c = tr.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT vec_ct.f($1)"})
		tr.TraceQueryEnd(c, nil, pgx.TraceQueryEndData{})
		w.WriteHeader(http.StatusInternalServerError)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	l := lineas(t, &b)[0]
	if l["error.type"] != "500" || l["vec.bd.error"] != nil {
		t.Errorf("linea = %v", l)
	}
}

func TestEsControl(t *testing.T) {
	for sql, control := range map[string]bool{
		"begin": true, " COMMIT": true, "rollback to savepoint x": true, "SET LOCAL x = 1": true,
		"select set_config('a', $1, true)": true, "SELECT pg_catalog.set_config('a','b',false)": true,
		"select vec.f($1)": false, "settings": false, "SELECT * FROM t": false, "update t set a=1": false,
	} {
		if esControl(sql) != control {
			t.Errorf("esControl(%q) = %t", sql, !control)
		}
	}
}

func TestErrorTypeConservaElFalloTrasElRollback(t *testing.T) {
	var b bytes.Buffer
	tr := trazador{}
	h := Middleware(opciones(&b), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		for _, paso := range []struct {
			sql string
			err error
		}{
			{"begin", nil},
			{"SELECT vec_ct.f($1)", &pgconn.PgError{Code: "57014"}},
			{"rollback", nil},
		} {
			c := tr.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: paso.sql})
			tr.TraceQueryEnd(c, nil, pgx.TraceQueryEndData{Err: paso.err})
		}
		c := tr.TraceAcquireStart(ctx, nil, pgxpool.TraceAcquireStartData{})
		tr.TraceAcquireEnd(c, nil, pgxpool.TraceAcquireEndData{})
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	l := lineas(t, &b)[0]
	if l["error.type"] != "bd_57014" || l["vec.bd.error"] != "bd_57014" || l["vec.bd.consultas"] != float64(1) {
		t.Errorf("linea = %v", l)
	}
}

func TestBatchCuentaResultadosObservadosSinAtribuirDuracionAlPrimerSQL(t *testing.T) {
	var b bytes.Buffer
	o := opciones(&b)
	o.Lenta = time.Nanosecond
	tr := trazador{}
	h := Middleware(o, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lote := &pgx.Batch{}
		lote.Queue("SELECT vec_bolsa.primera($1)", "dato privado")
		lote.Queue("SELECT vec_bolsa.segunda($1)", "dato privado")
		lote.Queue("SELECT vec_bolsa.tercera($1)", "dato privado")
		ctx := tr.TraceBatchStart(r.Context(), nil, pgx.TraceBatchStartData{Batch: lote})
		tr.TraceBatchQuery(ctx, nil, pgx.TraceBatchQueryData{SQL: lote.QueuedQueries[0].SQL})
		err := &pgconn.PgError{Code: "57014", Message: "dato privado"}
		tr.TraceBatchEnd(ctx, nil, pgx.TraceBatchEndData{Err: err})
		tr.TraceBatchEnd(ctx, nil, pgx.TraceBatchEndData{Err: err}) // pgx puede llamar End dos veces en un fallo temprano
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	l := lineas(t, &b)[0]
	if l["vec.bd.consultas"] != float64(1) || l["vec.bd.lotes"] != float64(1) ||
		l["vec.bd.lote.resultados_observados"] != float64(1) || l["vec.bd.lote.errores"] != float64(1) ||
		l["vec.bd.error"] != "bd_57014" || l["vec.bd.consulta_mas_lenta"] != nil || l["vec.bd.operaciones"] != nil {
		t.Errorf("lote = %v", l)
	}
	if l["vec.bd.lote.duracion_hasta_cierre"].(float64) < 0 || strings.Contains(b.String(), "dato privado") {
		t.Errorf("duración o privacidad del lote: %s", b.String())
	}
}

func TestResumenOperacionesAcotadoYErroresCerrados(t *testing.T) {
	m := &medida{}
	m.operaciones = make(map[string]*resumenConsulta)
	for n := 0; n < maxOperacionesPeticion-1; n++ {
		m.operaciones[fmt.Sprintf("operacion_estatica_%d", n)] = &resumenConsulta{n: 1}
	}
	ctx := context.WithValue(context.Background(), claveMedida{}, m)
	tr := trazador{}
	for n := 0; n < 6; n++ {
		q := tr.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT vec_usuarios.consultar_preferencias_propias_v1($1)", Args: []any{"dato privado"}})
		tr.TraceQueryEnd(q, nil, pgx.TraceQueryEndData{})
	}
	for n := 0; n < 5; n++ {
		q := tr.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: fmt.Sprintf("SELECT vec_persona_%d.valor($1)", n)})
		tr.TraceQueryEnd(q, nil, pgx.TraceQueryEndData{})
	}
	for n, codigo := range []string{"23505", "40001", "57014", "55P03", "42P01"} {
		q := tr.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT vec_usuarios.consultar_preferencias_propias_v1($1)"})
		tr.TraceQueryEnd(q, nil, pgx.TraceQueryEndData{Err: &pgconn.PgError{Code: codigo, Message: fmt.Sprintf("dato privado %d", n)}})
	}
	resumen := m.resumenOperaciones()
	if len(resumen) != maxOperacionesPeticion+1 {
		t.Fatalf("operaciones = %d, se esperaban %d", len(resumen), maxOperacionesPeticion+1)
	}
	var otras, primera *operacionMedida
	for i := range resumen {
		if resumen[i].Nombre == "otras" {
			otras = &resumen[i]
		}
		if resumen[i].Nombre == "vec_usuarios.consultar_preferencias_propias_v1" {
			primera = &resumen[i]
		}
	}
	if otras == nil || otras.N != 5 || primera == nil || primera.N != 11 || len(primera.Errores) != maxClasesErrorOperacion+1 || primera.Errores["otras"] != 1 || m.desconocidas.Load() != 5 {
		t.Errorf("resumen acotado incorrecto: %+v", resumen)
	}
}

func BenchmarkTrazadorConsulta(b *testing.B) {
	tr := trazador{}
	sql := "SELECT vec_usuarios.consultar_preferencias_propias_v1($1)"
	b.Run("sin_trazador", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			_ = context.Background()
		}
	})
	b.Run("trazador_peticion", func(b *testing.B) {
		ctx := context.WithValue(context.Background(), claveMedida{}, &medida{})
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			q := tr.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: sql})
			tr.TraceQueryEnd(q, nil, pgx.TraceQueryEndData{})
		}
	})
}
