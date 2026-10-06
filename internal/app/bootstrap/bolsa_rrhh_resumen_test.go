package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

// La lista de candidatos de una bolsa lee y descifra solo esa bolsa.
func TestFuenteConstituidaRRHHCandidatosLeeSoloSuBolsa(t *testing.T) {
	f, c := fuenteVariasBolsasRRHHPrueba(t, 5, 50, true)
	datos, err := f.cargarBolsa(context.Background(), "bolsa:prueba:03")
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

// Peticiones simultáneas comparten una lectura; una mutación hace que las
// siguientes no se sumen a la lectura anterior.
func TestCargaCompartidaBolsasRRHHUneSimultaneasYRespetaMutaciones(t *testing.T) {
	var c cargaCompartidaBolsasRRHH
	var lecturas atomic.Int64
	liberar := make(chan struct{})
	empezada := make(chan struct{}, 8)
	cargar := func(context.Context) (datasetBolsasRRHHDesarrollo, error) {
		n := lecturas.Add(1)
		empezada <- struct{}{}
		<-liberar
		return datasetBolsasRRHHDesarrollo{GeneradoEn: time.Unix(n, 0).UTC().Format(time.RFC3339)}, nil
	}
	var wg sync.WaitGroup
	resultados := make([]string, 5)
	for i := range resultados {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			datos, err := c.obtener(context.Background(), cargar)
			if err != nil {
				t.Error(err)
			}
			resultados[i] = datos.GeneradoEn
		}(i)
		if i == 0 {
			<-empezada
		}
	}
	// Espera a que las otras cuatro se sumen a la lectura en curso.
	time.Sleep(100 * time.Millisecond)
	c.invalidar()
	wg.Add(1)
	var posterior string
	go func() {
		defer wg.Done()
		datos, _ := c.obtener(context.Background(), cargar)
		posterior = datos.GeneradoEn
	}()
	<-empezada
	close(liberar)
	wg.Wait()
	if lecturas.Load() != 2 {
		t.Fatalf("lecturas=%d; se esperaban 2 (una compartida y otra tras la mutación)", lecturas.Load())
	}
	for _, r := range resultados {
		if r != resultados[0] {
			t.Fatalf("resultados distintos en la lectura compartida: %v", resultados)
		}
	}
	if posterior == resultados[0] {
		t.Fatal("una petición posterior a la mutación reutilizó la lectura anterior")
	}
}
