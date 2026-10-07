package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"
	"time"

	rptpublica "vec-diputacion-granada/internal/modules/personal/adapters/rptpublica"
	"vec-diputacion-granada/internal/modules/personal/domain"
)

// Este fixture sólo entrega un catálogo en memoria al handler público. No crea
// conexiones de base de datos ni ejecuta consultas SQL.
func catalogoRPTBenchmark(b *testing.B, ampliado bool) domain.CatalogoRPTPublica {
	b.Helper()
	bytes, err := os.ReadFile("../../../../data/catalogos/rpt/v1.rpt-2026.json")
	if err != nil {
		b.Fatal(err)
	}
	huella := sha256.Sum256(bytes)
	if hex.EncodeToString(huella[:]) != rptpublica.HuellaRPT2026 {
		b.Fatal("fuente RPT distinta de la inmovilizada")
	}
	var catalogo domain.CatalogoRPTPublica
	if err := json.Unmarshal(bytes, &catalogo); err != nil {
		b.Fatal(err)
	}
	catalogo.Fuente.HuellaSHA256 = rptpublica.HuellaRPT2026
	if ampliado {
		originales := append([]domain.PuestoRPTPublico(nil), catalogo.Puestos...)
		for len(catalogo.Puestos) < 10_000 {
			puesto := originales[len(catalogo.Puestos)%len(originales)].Clonar()
			puesto.Codigo = fmt.Sprintf("S-%05d", len(catalogo.Puestos))
			catalogo.Puestos = append(catalogo.Puestos, puesto)
		}
		catalogo.Resumen.Puestos = len(catalogo.Puestos)
		catalogo.Resumen.Dotacion = 0
		for _, puesto := range catalogo.Puestos {
			catalogo.Resumen.Dotacion += puesto.Dotacion
		}
	}
	if err := catalogo.Validar(); err != nil {
		b.Fatal(err)
	}
	return catalogo
}

func BenchmarkRPTPublicaEnlaces(b *testing.B) {
	for _, caso := range []struct {
		nombre   string
		ampliado bool
		codigo   string
		total    int
	}{{"fuente_842", false, "", 842}, {"fuente_842_codigo_217", false, "217", 1}, {"sintetico_10000", true, "", 10_000}} {
		b.Run(caso.nombre, func(b *testing.B) {
			catalogo := catalogoRPTBenchmark(b, caso.ampliado)
			h, err := NewHandlerRPTPublica(consultaRPTPublicaPrueba{catalogo})
			if err != nil {
				b.Fatal(err)
			}
			url := RutaRPTPublicaPersonal + "?vista=puestos&q=&limit=100&offset=0&enlaces=1"
			if caso.codigo != "" {
				url += "&codigo_puesto=" + caso.codigo
			}
			previa := httptest.NewRecorder()
			h.ServeHTTP(previa, httptest.NewRequest(http.MethodGet, url, nil))
			var pagina struct {
				Data struct {
					RPT struct {
						Total int `json:"total"`
						Items []struct {
							Codigo string `json:"codigo"`
						} `json:"items"`
					} `json:"rpt"`
				} `json:"data"`
			}
			if previa.Code != 200 || json.Unmarshal(previa.Body.Bytes(), &pagina) != nil || pagina.Data.RPT.Total != caso.total ||
				caso.codigo != "" && (len(pagina.Data.RPT.Items) != 1 || pagina.Data.RPT.Items[0].Codigo != caso.codigo) {
				b.Fatalf("respuesta previa incompatible: HTTP %d", previa.Code)
			}
			muestras := make([]int64, 0, 2048)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				inicio := time.Now()
				w := httptest.NewRecorder()
				h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
				if w.Code != 200 {
					b.Fatalf("HTTP %d", w.Code)
				}
				if len(muestras) < cap(muestras) {
					muestras = append(muestras, time.Since(inicio).Nanoseconds())
				}
			}
			b.StopTimer()
			sort.Slice(muestras, func(i, j int) bool { return muestras[i] < muestras[j] })
			if len(muestras) > 0 {
				indice := (95*len(muestras)+99)/100 - 1
				b.ReportMetric(float64(muestras[indice]), "p95-ns/op")
				b.ReportMetric(float64(len(muestras)), "p95-samples")
			}
			b.ReportMetric(0, "sql-queries/op") // Sólo fixture en memoria.
		})
	}
}
