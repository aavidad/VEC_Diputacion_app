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
	if err != nil || resultado.Total != 2 || !resultado.Fuente.Demostracion {
		t.Fatalf("resultado = %#v, error = %v", resultado, err)
	}
	resultado.Convocatorias[0].DatosPublicos.Titulo = "alterado"
	detalle, err := consulta.ObtenerPublicada(context.Background(), resultado.Convocatorias[0].DatosPublicos.IdentificadorPublico)
	if err != nil || detalle.Convocatoria.DatosPublicos.Titulo == "alterado" {
		t.Fatalf("la instantanea fue mutable: %#v, %v", detalle, err)
	}
}

func TestFuenteDemoFiltraPlazoAbiertoConCierreInclusivo(t *testing.T) {
	consulta, err := NuevaConsultaConvocatorias(rutaDemoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	cierre := time.Date(2026, 8, 15, 21, 59, 59, 0, time.UTC)
	for _, prueba := range []struct {
		nombre   string
		instante time.Time
		total    int
	}{
		{nombre: "instante exacto", instante: cierre, total: 1},
		{nombre: "microsegundo posterior", instante: cierre.Add(time.Microsecond), total: 0},
	} {
		t.Run(prueba.nombre, func(t *testing.T) {
			resultado, err := consulta.BuscarPublicadas(context.Background(), puertosbolsa.FiltroConvocatoriasPublicas{Instante: prueba.instante, Limite: 24, SoloPlazoAbierto: true})
			if err != nil || resultado.Total != prueba.total {
				t.Fatalf("total = %d, error = %v", resultado.Total, err)
			}
		})
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
		"espacio no canonico":          strings.Replace(string(base), `"titulo": "Bolsa temporal`, `"titulo": " Bolsa temporal`, 1),
		"marca bidi":                   strings.Replace(string(base), `"titulo": "Bolsa temporal`, `"titulo": "Bolsa \u202etemporal`, 1),
		"version catalogo incoherente": strings.Replace(string(base), `"clave": "bolsa_temporal", "version": 1`, `"clave": "bolsa_temporal", "version": 2`, 1),
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
