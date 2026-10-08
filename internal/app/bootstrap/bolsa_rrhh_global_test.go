package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	bolsapg "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	bolsaapp "vec-diputacion-granada/internal/modules/bolsa/application"
)

func TestBolsaGlobalConservaCorteTotalYFiltroSinVolverALeer(t *testing.T) {
	datos := datosBolsasRRHHPrueba()
	datos.ResumenConjunto = true
	datos.Candidaturas[1].Estado = "renuncia"
	for n := 3; n <= 205; n++ {
		p := datos.Candidaturas[0]
		p.Referencia, p.OrdenActa = "participacion:"+strconv.Itoa(n), n
		datos.Candidaturas = append(datos.Candidaturas, p)
	}
	lecturas := 0
	h := nuevoManejadorBolsasRRHHDesarrollo(func(context.Context) (datasetBolsasRRHHDesarrollo, error) { lecturas++; return datos, nil })
	get := func(query string) map[string]any {
		t.Helper()
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+query, nil))
		if w.Code != 200 {
			t.Fatalf("GET %s: %d %s", query, w.Code, w.Body.String())
		}
		var salida struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &salida); err != nil {
			t.Fatal(err)
		}
		return salida.Data
	}
	resumen := get("")
	corte := resumen["corte_ref"].(string)
	// Un cambio posterior no altera los recuentos ni las filas de ese corte.
	datos.Candidaturas[0].Estado = "trabajando"
	for filtro, esperado := range map[string]int{"todos": 205, "disponible": 204, "renuncia": 1} {
		cursor, vistos := "", 0
		for {
			q := "?filtro=" + filtro + "&corte=" + corte + "&limite=100"
			if cursor != "" {
				q += "&cursor=" + cursor
			}
			pagina := get(q)
			if int(pagina["total"].(float64)) != esperado || pagina["corte_ref"] != corte {
				t.Fatalf("corte/total distintos: %v", pagina)
			}
			items := pagina["items"].([]any)
			vistos += len(items)
			for _, item := range items {
				p := item.(map[string]any)
				if filtro != "todos" && p["estado_clave"] != filtro {
					t.Fatalf("predicado: %v", p)
				}
				if _, existe := p["nombre_visible"]; existe {
					t.Fatal("la lista mínima expone nombres")
				}
			}
			if !pagina["hay_mas"].(bool) {
				break
			}
			cursor = pagina["cursor_siguiente"].(string)
		}
		if vistos != esperado {
			t.Fatalf("filtro=%s total=%d filas=%d", filtro, esperado, vistos)
		}
	}
	if lecturas != 1 {
		t.Fatalf("lecturas: esperado=1 obtenido=%d", lecturas)
	}
}

func TestBolsaGlobalRechazaParametrosAntesDeLeer(t *testing.T) {
	lecturas := 0
	h := nuevoManejadorBolsasRRHHDesarrollo(func(context.Context) (datasetBolsasRRHHDesarrollo, error) {
		lecturas++
		return datosBolsasRRHHPrueba(), nil
	})
	for _, q := range []string{"?filtro=otro", "?filtro=todos&filtro=renuncia", "?filtro=todos&cursor=1", "?filtro=todos&limite=101", "?filtro=todos&corte=mal", "?filtro=todos&bolsa=bolsa:1", "?filtro=todos&actor=administrador", "?filtro=todos&cursor=-1", "?filtro=todos&limite=01"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+q, nil))
		if w.Code != 400 {
			t.Fatalf("%s: esperado=400 obtenido=%d", q, w.Code)
		}
	}
	if lecturas != 0 {
		t.Fatalf("entrada inválida lee: %d", lecturas)
	}
}

func TestBolsaGlobalCaducadoNoCambiaElCorteEnSilencio(t *testing.T) {
	var cache cacheGlobalBolsasRRHH
	ahora := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	ref, err := cache.guardar(bolsaapp.ConjuntoGlobalRRHH{GeneradoEn: ahora.Format(time.RFC3339)}, ahora)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cache.leer(ref, ahora.Add(vigenciaCorteGlobalBolsas-time.Nanosecond)); !ok {
		t.Fatal("caducó antes de tiempo")
	}
	if _, ok := cache.leer(ref, ahora.Add(vigenciaCorteGlobalBolsas)); ok {
		t.Fatal("corte caducado servido")
	}
	if len(cache.cortes) != 0 {
		t.Fatal("el corte caducado sigue retenido")
	}
	h := manejadorBolsasRRHHPrueba()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"?filtro=todos&corte="+ref, nil))
	if w.Code != 409 {
		t.Fatalf("esperado=409 obtenido=%d", w.Code)
	}
}

func TestBolsaGlobalRetieneEnlacesAunqueOtrosUsuariosRecarguen(t *testing.T) {
	var cache cacheGlobalBolsasRRHH
	ahora := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	conjunto := bolsaapp.ConjuntoGlobalRRHH{GeneradoEn: ahora.Format(time.RFC3339Nano), Participaciones: []bolsaapp.ParticipacionGlobalRRHH{{ParticipacionRef: "participacion:primera", Estado: "disponible"}}}
	original, err := cache.guardar(conjunto, ahora)
	if err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= 20; n++ {
		instante := ahora.Add(time.Duration(n) * time.Second)
		conjunto.GeneradoEn = instante.Format(time.RFC3339Nano)
		ref, err := cache.guardar(conjunto, instante)
		if err != nil || ref != original {
			t.Fatalf("recarga=%d corte esperado=%s observado=%s error=%v", n, original, ref, err)
		}
	}
	retenido, ok := cache.leer(original, ahora.Add(time.Minute))
	if !ok || retenido.GeneradoEn != ahora.Format(time.RFC3339Nano) {
		t.Fatal("la recarga altera el corte original")
	}
	// Cambios reales crean otros cortes; tampoco invalidan los enlaces emitidos.
	for n := 1; n <= 12; n++ {
		nuevo := bolsaapp.ConjuntoGlobalRRHH{Participaciones: []bolsaapp.ParticipacionGlobalRRHH{{ParticipacionRef: "participacion:" + strconv.Itoa(n), Estado: "renuncia"}}}
		if _, err := cache.guardar(nuevo, ahora.Add(time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := cache.leer(original, ahora.Add(vigenciaCorteGlobalBolsas-time.Nanosecond)); !ok {
		t.Fatal("otros usuarios expulsaron un enlace todavía vigente")
	}
	if _, ok := cache.leer(original, ahora.Add(vigenciaCorteGlobalBolsas)); ok {
		t.Fatal("el corte vencido sigue disponible")
	}
}

func TestBolsaGlobalLlamamientosConMismoTotalYPredicado(t *testing.T) {
	datos := datosBolsasRRHHPrueba()
	datos.ResumenConjunto = true
	datos.Bolsas[0].LlamamientosEnCurso = 2
	datos.LlamamientosGlobales = []bolsaapp.LlamamientoGlobalRRHH{
		{BolsaRef: datos.Bolsas[0].Referencia, LlamamientoRef: "llamamiento:01", Referencia: "2026/0001", EmitidoEn: datos.GeneradoEn, Participaciones: 2},
		{BolsaRef: datos.Bolsas[0].Referencia, LlamamientoRef: "llamamiento:02", Referencia: "2026/0002", EmitidoEn: datos.GeneradoEn, Participaciones: 1},
	}
	c, err := conjuntoGlobalBolsas(datos)
	if err != nil {
		t.Fatal(err)
	}
	p, err := c.Paginar(bolsaapp.ConsultaGlobalRRHH{Filtro: "llamamientos", BolsaRef: datos.Bolsas[0].Referencia, Limite: 1})
	if err != nil || p.Total != 2 || p.Hasta != 1 {
		t.Fatalf("pagina=%+v err=%v", p, err)
	}
	datos.Bolsas[0].LlamamientosEnCurso = 3
	if _, err := conjuntoGlobalBolsas(datos); err == nil {
		t.Fatal("lista distinta de contador aceptada")
	}
}

func BenchmarkBolsaGlobalPagina20000(b *testing.B) {
	c := bolsaapp.ConjuntoGlobalRRHH{}
	for n := 0; n < 20000; n++ {
		c.Participaciones = append(c.Participaciones, bolsaapp.ParticipacionGlobalRRHH{Estado: "disponible", OrdenActa: n + 1})
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if _, err := c.Paginar(bolsaapp.ConsultaGlobalRRHH{Filtro: "disponible", Limite: 50}); err != nil {
			b.Fatal(err)
		}
	}
}

// Usa el cargador y el manejador reales contra el clon causal. Mide HTTP de
// aplicación, sin atribuirle el coste de TLS o de la frontera nominal externa.
func TestBolsaGlobalHTTPP95Clon(t *testing.T) {
	dsn := os.Getenv("VEC_BOLSA_GLOBAL_PG_DSN")
	if dsn == "" {
		t.Skip("sin VEC_BOLSA_GLOBAL_PG_DSN")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal("no se pudo abrir el pool privado")
	}
	defer pool.Close()
	lector, err := bolsapg.NuevoLectorResumenBolsasConLlamamientosPostgreSQL(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	base, err := lector.LeerResumen(ctx, time.Now())
	if err != nil || len(base.Situaciones) < 2000 {
		t.Fatalf("volumen esperado>=2000 obtenido=%d error=%v", len(base.Situaciones), err)
	}
	categorias := map[string]string{}
	for _, fila := range base.Situaciones {
		categorias[fila.CategoriaRef] = "Administración"
	}
	fuente := &fuenteConstituidaRRHHDesarrollo{resumenConjunto: lector, ahora: time.Now, categorias: categorias}
	h := nuevoManejadorBolsasRRHHDesarrollo(nil)
	h.resumen = fuente.cargarResumenConjunto
	var tiempos []time.Duration
	var corte string
	for n := 0; n < 30; n++ {
		w := httptest.NewRecorder()
		inicio := time.Now()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo, nil))
		tiempos = append(tiempos, time.Since(inicio))
		if w.Code != 200 {
			t.Fatalf("resumen: HTTP=%d", w.Code)
		}
		var salida struct {
			Data struct {
				Corte string `json:"corte_ref"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &salida); err != nil {
			t.Fatal(err)
		}
		corte = salida.Data.Corte
	}
	sort.Slice(tiempos, func(i, j int) bool { return tiempos[i] < tiempos[j] })
	p95 := tiempos[28]
	if p95 >= 300*time.Millisecond {
		t.Fatalf("resumen p95 esperado<300ms obtenido=%v", p95)
	}
	t.Logf("participaciones=%d resumen_http_p95=%v", len(base.Situaciones), p95)
	for _, filtro := range []string{"todos", "disponible", "renuncia", "llamamientos"} {
		tiempos = nil
		for n := 0; n < 30; n++ {
			w := httptest.NewRecorder()
			inicio := time.Now()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"?filtro="+filtro+"&corte="+corte+"&limite=50", nil))
			tiempos = append(tiempos, time.Since(inicio))
			if w.Code != 200 {
				t.Fatalf("filtro=%s HTTP=%d", filtro, w.Code)
			}
		}
		sort.Slice(tiempos, func(i, j int) bool { return tiempos[i] < tiempos[j] })
		if tiempos[28] >= 300*time.Millisecond {
			t.Fatalf("%s p95 esperado<300ms obtenido=%v", filtro, tiempos[28])
		}
		t.Logf("filtro=%s pagina_http_p95=%v", filtro, tiempos[28])
	}
}
