package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// El cuadro y las estadísticas no descifran el acta protegida y leen lo mismo
// por bolsa; los recuentos por estado coinciden con la carga completa.
func TestFuenteConstituidaRRHHResumenNoDescifraActaNiCreceConLasPersonas(t *testing.T) {
	const bolsas, participaciones = 4, 300
	completa, _ := fuenteVariasBolsasRRHHPrueba(t, bolsas, participaciones, true)
	esperado, err := completa.cargar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	f, c := fuenteVariasBolsasRRHHPrueba(t, bolsas, participaciones, true)
	resumen, err := f.cargarResumen(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if c.recuperar.Load() != 0 {
		t.Fatalf("el resumen descifró %d actas protegidas", c.recuperar.Load())
	}
	if c.orden.Load() != bolsas || c.situacionLote.Load() != bolsas || c.ceseLote.Load() != bolsas || c.situacionIndividual.Load() != 0 || c.ceseIndividual.Load() != 0 {
		t.Fatalf("lecturas del resumen: orden=%d situaciones=%d/%d ceses=%d/%d", c.orden.Load(), c.situacionLote.Load(), c.situacionIndividual.Load(), c.ceseLote.Load(), c.ceseIndividual.Load())
	}
	for _, candidata := range resumen.Candidaturas {
		if candidata.Nombre != "" || candidata.Documento != "" {
			t.Fatal("el resumen contiene datos personales")
		}
	}
	a, _ := json.Marshal((&bolsasRRHHDesarrolloDatos{datos: esperado}).respuestaEstadisticas())
	b, _ := json.Marshal((&bolsasRRHHDesarrolloDatos{datos: resumen}).respuestaEstadisticas())
	if string(a) != string(b) {
		t.Fatalf("estadísticas distintas:\n%s\n%s", a, b)
	}
	a, _ = json.Marshal((&bolsasRRHHDesarrolloDatos{datos: esperado}).respuestaBolsas())
	b, _ = json.Marshal((&bolsasRRHHDesarrolloDatos{datos: resumen}).respuestaBolsas())
	if string(a) != string(b) {
		t.Fatalf("cuadro distinto:\n%s\n%s", a, b)
	}
}

// El camino previo conserva la lectura acotada a una bolsa para consumidores
// históricos; la ruta RRHH exige el lector de conjunto B92.
func TestFuenteConstituidaRRHHCandidatosLeeSoloSuBolsa(t *testing.T) {
	f, c := fuenteVariasBolsasRRHHPrueba(t, 5, 50, true)
	datos, err := f.cargarAlcance(context.Background(), alcanceCargaBolsasRRHH{bolsa: "bolsa:prueba:03", detalle: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(datos.Bolsas) != 1 || datos.Bolsas[0].Referencia != "bolsa:prueba:03" || len(datos.Candidaturas) != 50 {
		t.Fatalf("bolsas=%d candidaturas=%d", len(datos.Bolsas), len(datos.Candidaturas))
	}
	if c.orden.Load() != 1 || c.recuperar.Load() != 1 || c.situacionLote.Load() != 1 || c.ceseLote.Load() != 1 {
		t.Fatalf("lecturas: orden=%d actas=%d situaciones=%d ceses=%d", c.orden.Load(), c.recuperar.Load(), c.situacionLote.Load(), c.ceseLote.Load())
	}
	if datos.Candidaturas[0].Nombre == "" {
		t.Fatal("la lista de candidatos perdió el nombre visible")
	}
	if _, err := f.cargarBolsa(context.Background(), ""); err == nil {
		t.Fatal("se aceptó una bolsa vacía")
	}
}

func TestBolsasRRHHDesarrolloSinB92FallaSoloLaLista(t *testing.T) {
	f, _ := fuenteVariasBolsasRRHHPrueba(t, 1, 2, true)
	h := nuevoManejadorBolsasRRHHDesarrollo(f.cargar)
	h.cargarBolsa = f.cargarBolsa
	lista := httptest.NewRecorder()
	h.ServeHTTP(lista, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"/bolsa:prueba:01/candidatos", nil))
	if lista.Code != http.StatusServiceUnavailable {
		t.Fatalf("lista sin B92: status=%d body=%s", lista.Code, lista.Body.String())
	}
	cuadro := httptest.NewRecorder()
	h.ServeHTTP(cuadro, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo, nil))
	if cuadro.Code != http.StatusOK {
		t.Fatalf("cuadro sin B92: status=%d body=%s", cuadro.Code, cuadro.Body.String())
	}
}

// El manejador usa el resumen para cuadro y estadísticas y la carga de una
// sola bolsa para su lista de candidatos.
func TestBolsasRRHHDesarrolloUsaResumenYBolsaConcreta(t *testing.T) {
	manejador := manejadorBolsasRRHHPrueba()
	var completas, resumenes atomic.Int64
	var pedida string
	cargar := manejador.cargar
	manejador.cargar = func(ctx context.Context) (datasetBolsasRRHHDesarrollo, error) {
		completas.Add(1)
		return cargar(ctx)
	}
	manejador.resumen = func(ctx context.Context) (datasetBolsasRRHHDesarrollo, error) {
		resumenes.Add(1)
		return cargar(ctx)
	}
	manejador.cargarBolsa = func(ctx context.Context, ref string) (datasetBolsasRRHHDesarrollo, error) {
		pedida = ref
		return cargar(ctx)
	}
	for _, ruta := range []string{rutaBolsasRRHHDesarrollo, rutaEstadisticasBolsaRRHHDesarrollo} {
		rec := httptest.NewRecorder()
		manejador.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ruta, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", ruta, rec.Code, rec.Body.String())
		}
	}
	rec := httptest.NewRecorder()
	manejador.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"/bolsa:constituida:administrativo/candidatos", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("candidatos: %d %s", rec.Code, rec.Body.String())
	}
	if resumenes.Load() != 2 || completas.Load() != 0 || pedida != "bolsa:constituida:administrativo" {
		t.Fatalf("resumenes=%d completas=%d bolsa=%q", resumenes.Load(), completas.Load(), pedida)
	}
}
