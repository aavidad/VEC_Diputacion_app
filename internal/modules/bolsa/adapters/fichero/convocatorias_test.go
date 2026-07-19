package fichero

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const rutaDemoPrueba = "../../../../../data/demo/convocatorias_publicas.demo.json"

func TestFuenteDemoCargaAgregadoCanonicoSinPIIYDevuelveCopias(t *testing.T) {
	consulta, err := NuevaConsultaConvocatorias(rutaDemoPrueba)
	if err != nil {
		t.Fatalf("NuevaConsultaConvocatorias() error = %v", err)
	}
	resultado, err := consulta.BuscarPublicadas(context.Background(), puertosbolsa.FiltroConvocatoriasPublicas{
		Instante: time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC), Limite: 24,
	})
	if err != nil || resultado.Total != 36 || !resultado.Fuente.Demostracion {
		t.Fatalf("resultado = %#v, error = %v", resultado, err)
	}
	resultado.Convocatorias[0].DatosPublicos.Titulo = "alterado"
	detalle, err := consulta.ObtenerPublicada(context.Background(), resultado.Convocatorias[0].DatosPublicos.IdentificadorPublico)
	if err != nil || detalle.Convocatoria.DatosPublicos.Titulo == "alterado" {
		t.Fatalf("la instantanea fue mutable: %#v, %v", detalle, err)
	}
}

func TestFuenteDemoOfreceDosRecorridosAbiertosClaramenteRotulados(t *testing.T) {
	consulta, err := NuevaConsultaConvocatorias(rutaDemoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := consulta.BuscarPublicadas(context.Background(), puertosbolsa.FiltroConvocatoriasPublicas{
		Instante: time.Date(2026, 7, 19, 8, 0, 0, 0, time.UTC), Limite: 24, SoloPlazoAbierto: true,
	})
	if err != nil || resultado.Total != 2 {
		t.Fatalf("total = %d, error = %v", resultado.Total, err)
	}
	for _, convocatoria := range resultado.Convocatorias {
		if convocatoria.Estado != "inscripcion" || len(convocatoria.DatosPublicos.Plazos) != 1 ||
			!strings.Contains(convocatoria.DatosPublicos.Descripcion, "escenario sintético rotulado como DEMO") {
			t.Fatalf("recorrido abierto no queda inequívocamente rotulado: %#v", convocatoria)
		}
	}
}

func TestFuenteDemoCuentaCategoriasIgnorandoSoloEseFiltro(t *testing.T) {
	consulta, err := NuevaConsultaConvocatorias(rutaDemoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := consulta.BuscarPublicadas(context.Background(), puertosbolsa.FiltroConvocatoriasPublicas{
		Categoria: "operario",
		Instante:  time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC),
		Limite:    24,
	})
	if err != nil || resultado.Total != 2 ||
		resultado.ConteosCategorias["operario"].NumeroConvocatorias != 2 ||
		resultado.ConteosCategorias["tecnico-de-gestion"].NumeroConvocatorias != 1 {
		t.Fatalf("resultado=%#v error=%v", resultado, err)
	}
}

func TestFuenteDemoRechazaJSONLaxoDuplicadoYDatosPersonales(t *testing.T) {
	base, err := os.ReadFile(rutaDemoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	casos := map[string]string{
		"campo desconocido":            strings.Replace(string(base), `"version_esquema": 1,`, `"version_esquema": 1, "desconocido": true,`, 1),
		"clave duplicada":              strings.Replace(string(base), `"version_esquema": 1,`, `"version_esquema": 1, "version_esquema": 1,`, 1),
		"dni_sintetico":                strings.Replace(string(base), `"version_esquema": 1,`, `"version_esquema": 1, "dni": "00000000A",`, 1),
		"espacio no canonico":          strings.Replace(string(base), `"titulo": "Bolsa de empleo`, `"titulo": " Bolsa de empleo`, 1),
		"marca bidi":                   strings.Replace(string(base), `"titulo": "Bolsa de empleo`, `"titulo": "Bolsa \u202ede empleo`, 1),
		"version catalogo incoherente": strings.Replace(string(base), `"clave": "bolsa_temporal", "version": 1`, `"clave": "bolsa_temporal", "version": 2`, 1),
		"categorias embebidas":         strings.Replace(string(base), `"catalogos": [`, `"catalogos": [{"referencia":"categorias_convocatoria","version":1,"entradas":[{"clave":"auxiliar-administrativo","version":1,"etiqueta":"Auxiliar administrativo","semantica":"informacion","orden":1,"publicable":true}]},`, 1),
	}
	for nombre, contenido := range casos {
		t.Run(nombre, func(t *testing.T) {
			_, err := nuevaConsultaDesdeJSON([]byte(contenido))
			if !errors.Is(err, puertosbolsa.ErrFuenteConvocatoriasInvalida) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestFuenteDemoRechazaFiltrosNoCanonicosAunqueSeInvoqueElPuertoDirecto(t *testing.T) {
	consulta, err := NuevaConsultaConvocatorias(rutaDemoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	_, err = consulta.BuscarPublicadas(context.Background(), puertosbolsa.FiltroConvocatoriasPublicas{
		Tipo: " bolsa_temporal", Instante: time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC), Limite: 1,
	})
	if !errors.Is(err, puertosbolsa.ErrConsultaConvocatoriasInvalida) {
		t.Fatalf("error = %v", err)
	}
}
