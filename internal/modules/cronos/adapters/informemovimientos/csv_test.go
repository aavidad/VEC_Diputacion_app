package informemovimientos

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func leerFilasCSV(t *testing.T, datos []byte) [][]string {
	t.Helper()
	filas, err := csv.NewReader(bytes.NewReader(datos)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return filas
}

func TestCSVMovimientosConservaCambioHorarioYOrdenEstable(t *testing.T) {
	e := ejemploPrueba(t)
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			b, err := PrepararCSV(context.Background(), bytes.NewReader(catalogoCSVPrueba(t, idioma)), e)
			if err != nil {
				t.Fatal(err)
			}
			filas := leerFilasCSV(t, b)
			if len(filas) != 6 || filas[1][5] == "" || filas[1][6] == "" || filas[1][7] != "Europe/Madrid" || filas[1][8] == "" || filas[1][9] == "" {
				t.Fatal("falta contexto o un fichaje", filas)
			}
			if !strings.Contains(filas[2][2], "+02:00") || !strings.Contains(filas[5][2], "+01:00") || filas[2][2] != filas[3][2] {
				t.Fatal("se perdieron las horas repetidas o los empates", filas)
			}
			if !strings.Contains(filas[3][4], "verificar") && !strings.Contains(filas[3][4], "verified") {
				t.Fatal("origen nulo tratado como conocido", filas[3])
			}
			if bytes.Contains(b, []byte(e.Nombre)) || bytes.Contains(b, []byte("marcaje_ref")) || bytes.Contains(b, []byte("saldo_minutos")) {
				t.Fatal("datos no necesarios en CSV")
			}
			otra, err := PrepararCSV(context.Background(), bytes.NewReader(catalogoCSVPrueba(t, idioma)), e)
			if err != nil || !bytes.Equal(b, otra) {
				t.Fatal("exportación inestable", err)
			}
		})
	}
}

func catalogoCSVPrueba(t *testing.T, idioma string) []byte {
	t.Helper()
	ruta := "../../../../../web/static/textos/" + idioma + "/cronos-informe-movimientos-csv.json"
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCSVVacioConservaContextoYEscapaFormula(t *testing.T) {
	e := ejemploPrueba(t)
	e.Marcajes = []ports.MarcajeDia{}
	c := bytes.Replace(catalogoCSVPrueba(t, "es"), []byte("Contexto del ejemplo"), []byte("=SUM(1,1)"), 1)
	b, err := PrepararCSV(context.Background(), bytes.NewReader(c), e)
	if err != nil {
		t.Fatal(err)
	}
	filas := leerFilasCSV(t, b)
	if len(filas) != 2 || filas[1][0] != "'=SUM(1,1)" || !strings.Contains(filas[1][8], "incompleta") {
		t.Fatal("contexto vacío o fórmula insegura", filas)
	}
}

func TestCSVRechazaFuenteAjenaSinBytes(t *testing.T) {
	casos := map[string]func(*EjemploSintetico){
		"no_demo": func(e *EjemploSintetico) { e.Demo = false },
		"periodo": func(e *EjemploSintetico) { e.Periodo.Hasta = "2026-10-24" },
		"utc": func(e *EjemploSintetico) {
			e.Marcajes[0].InstanteUTC = e.Marcajes[0].InstanteUTC.In(time.FixedZone("otro", 3600))
		},
		"origen":        func(e *EjemploSintetico) { s := "ajeno"; e.Marcajes[0].Origen = &s },
		"movimiento":    func(e *EjemploSintetico) { e.Marcajes[0].Movimiento = "ajeno" },
		"lista_ausente": func(e *EjemploSintetico) { e.Marcajes = nil },
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			e := ejemploPrueba(t)
			cambiar(&e)
			b, err := PrepararCSV(context.Background(), bytes.NewReader(catalogoCSVPrueba(t, "es")), e)
			if err == nil || len(b) != 0 {
				t.Fatal("entrada inválida produjo bytes", err)
			}
		})
	}
	e := ejemploPrueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if b, err := PrepararCSV(ctx, bytes.NewReader(catalogoCSVPrueba(t, "es")), e); err == nil || len(b) != 0 {
		t.Fatal("cancelación produjo bytes", err)
	}
}

func TestCSVRechazaCatalogoAmbiguo(t *testing.T) {
	e := ejemploPrueba(t)
	catalogo := catalogoCSVPrueba(t, "es")
	for _, c := range [][]byte{
		bytes.Replace(catalogo, []byte(`"version": "1"`), []byte(`"version": "1", "version": "2"`), 1),
		bytes.Replace(catalogo, []byte(`"idioma": "es-ES"`), []byte(`"idioma": "en-GB"`), 1),
		bytes.Replace(catalogo, []byte("15:04:05 -07:00 MST"), []byte("15:04:05"), 1),
	} {
		b, err := PrepararCSV(context.Background(), bytes.NewReader(c), e)
		if err == nil || len(b) != 0 {
			t.Fatal("catálogo inválido produjo bytes", err)
		}
	}
}

func TestCSVRechazaCabecerasFueraDelCatalogoComun(t *testing.T) {
	e := ejemploPrueba(t)
	var base map[string]any
	if err := json.Unmarshal(catalogoCSVPrueba(t, "es"), &base); err != nil {
		t.Fatal(err)
	}
	for nombre, cambiar := range map[string]func(map[string]any){
		"array":   func(c map[string]any) { c["cabeceras"] = []string{"Registro", "Fecha"} },
		"ausente": func(c map[string]any) { delete(c["cabeceras"].(map[string]any), "registro") },
		"ajena":   func(c map[string]any) { c["cabeceras"].(map[string]any)["dato_ajeno"] = "Dato ajeno" },
	} {
		t.Run(nombre, func(t *testing.T) {
			c := map[string]any{}
			for k, v := range base {
				c[k] = v
			}
			cabeceras := map[string]any{}
			for k, v := range base["cabeceras"].(map[string]any) {
				cabeceras[k] = v
			}
			c["cabeceras"] = cabeceras
			cambiar(c)
			raw, err := json.Marshal(c)
			if err != nil {
				t.Fatal(err)
			}
			b, err := PrepararCSV(context.Background(), bytes.NewReader(raw), e)
			if err == nil || len(b) != 0 {
				t.Fatal("cabeceras inválidas produjeron bytes", err)
			}
		})
	}
}
