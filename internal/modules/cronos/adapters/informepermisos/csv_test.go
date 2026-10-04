package informepermisos

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func catalogoCSVPrueba(t *testing.T, idioma string) []byte {
	t.Helper()
	raw, err := os.ReadFile("../../../../../web/static/textos/" + idioma + "/cronos-informe-permisos-csv.json")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func ejemploCSVPrueba() EjemploSinteticoCSV {
	concedidoDias, concedidoMinutos := int64(12), int64(120)
	return EjemploSinteticoCSV{Demo: true, Nombre: "Persona sintética", Resumen: ports.ResumenPermisosInforme{
		Ejercicio: 2026, CorteUTC: time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC),
		CamposPermitidos: []string{"etiqueta", "unidad", "computo", "pendiente_resolver", "concedido", "restante", "conciliacion"},
		Filas: []ports.FilaInformePermisos{
			{Etiqueta: "Vacaciones", Unidad: "dia", Computo: "laborables", Concedido: &concedidoDias, Conciliacion: ports.ConciliacionPermisosConfirmada},
			{Etiqueta: "Asuntos propios", Unidad: "hora", Computo: "laborables", Concedido: &concedidoMinutos, Conciliacion: ports.ConciliacionPermisosPendiente},
		},
	}}
}

func filasCSV(t *testing.T, b []byte) [][]string {
	t.Helper()
	filas, err := csv.NewReader(bytes.NewReader(b)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return filas
}

func TestCSVPermisosContextoYSubconjuntoSinColumnasExcluidas(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			e := ejemploCSVPrueba()
			e.Resumen.CamposPermitidos = []string{"etiqueta", "unidad", "concedido"}
			contenido, err := PrepararCSVEjemploSintetico(context.Background(), bytes.NewReader(catalogoCSVPrueba(t, idioma)), e)
			if err != nil {
				t.Fatal(err)
			}
			filas := filasCSV(t, contenido)
			if len(filas) != 4 {
				t.Fatal(filas)
			}
			for _, fila := range filas {
				if len(fila) != 8 {
					t.Fatal("tabla irregular", fila)
				}
			}
			if filas[1][1] != e.Nombre || filas[1][2] != "2026" || filas[1][3] == "" || filas[1][4] == "" || filas[2][7] != "12" || filas[3][7] != "120" {
				t.Fatal(filas)
			}
			if idioma == "es" && (filas[2][6] != "Días" || filas[3][6] != "Minutos") {
				t.Fatal("unidad equivocada", filas)
			}
			for _, excluido := range []string{"Pendiente de resolver", "Restante", "Conciliación", "Cómputo", "No disponible", "tipo_ref"} {
				if strings.Contains(string(contenido), excluido) {
					t.Fatal("campo excluido", excluido)
				}
			}
			e.Resumen.Filas = nil
			contenido, err = PrepararCSVEjemploSintetico(context.Background(), bytes.NewReader(catalogoCSVPrueba(t, idioma)), e)
			if err != nil || len(filasCSV(t, contenido)) != 2 {
				t.Fatal("contexto vacío", err)
			}
		})
	}
}

func TestCSVPermisosNuloCeroYFórmulas(t *testing.T) {
	e := ejemploCSVPrueba()
	e.Resumen.CamposPermitidos = []string{"etiqueta", "unidad", "concedido", "restante"}
	cero := int64(0)
	e.Resumen.Filas[0].Concedido = &cero
	var c CatalogoCSV
	if err := json.Unmarshal(catalogoCSVPrueba(t, "es"), &c); err != nil {
		t.Fatal(err)
	}
	for k := range c.Cabeceras {
		c.Cabeceras[k] = "=cabecera"
	}
	c.Contexto, c.Permiso, c.Sintetico, c.Desconocido = "=contexto", "+permiso", "@aviso", "-desconocido"
	c.FormatoCorte = "=2006-01-02"
	c.Unidades["dia"], c.Unidades["hora"] = "=días", "+minutos"
	e.Nombre = " \u200b=persona"
	e.Resumen.Filas[0].Etiqueta = " \u200b=vacaciones"
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := PrepararCSVEjemploSintetico(context.Background(), bytes.NewReader(raw), e)
	if err != nil {
		t.Fatal(err)
	}
	filas := filasCSV(t, contenido)
	for _, cabecera := range filas[0] {
		if !strings.HasPrefix(cabecera, "'") {
			t.Fatal("fórmula en cabecera", cabecera)
		}
	}
	for _, celda := range []string{filas[1][0], filas[1][1], filas[1][3], filas[1][4], filas[2][0], filas[2][5], filas[2][6], filas[3][8]} {
		if !strings.HasPrefix(celda, "'") {
			t.Fatal("fórmula textual", celda)
		}
	}
	if filas[2][7] != "0" || filas[2][8] != "'-desconocido" || filas[3][7] != "120" {
		t.Fatal("números o nulos alterados", filas)
	}
}

func TestCSVPermisosFallaCerrado(t *testing.T) {
	base := catalogoCSVPrueba(t, "es")
	for _, alterar := range []func(*EjemploSinteticoCSV){
		func(e *EjemploSinteticoCSV) { e.Demo = false },
		func(e *EjemploSinteticoCSV) { e.Resumen.CamposPermitidos = nil },
		func(e *EjemploSinteticoCSV) { e.Resumen.CamposPermitidos = []string{"etiqueta", "etiqueta"} },
		func(e *EjemploSinteticoCSV) { e.Resumen.CamposPermitidos = []string{"concedido"} },
		func(e *EjemploSinteticoCSV) { e.Resumen.CamposPermitidos = []string{"motivo"} },
		func(e *EjemploSinteticoCSV) { negativo := int64(-1); e.Resumen.Filas[0].Concedido = &negativo },
	} {
		e := ejemploCSVPrueba()
		alterar(&e)
		contenido, err := PrepararCSVEjemploSintetico(context.Background(), bytes.NewReader(base), e)
		if err == nil || len(contenido) != 0 {
			t.Fatal("entrada inválida admitida", err)
		}
	}
	e := ejemploCSVPrueba()
	for _, raw := range [][]byte{[]byte(`{}`), append(append([]byte(nil), base...), []byte(` {}`)...), bytes.Replace(base, []byte("cronos-permisos-csv-v1"), []byte("otro"), 1), bytes.Replace(base, []byte(`"version": "1"`), []byte(`"version": "1", "version": "2"`), 1)} {
		contenido, err := PrepararCSVEjemploSintetico(context.Background(), bytes.NewReader(raw), e)
		if err == nil || len(contenido) != 0 {
			t.Fatal("catálogo inválido admitido", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	contenido, err := PrepararCSVEjemploSintetico(ctx, bytes.NewReader(base), e)
	if !errors.Is(err, context.Canceled) || len(contenido) != 0 {
		t.Fatal("cancelación", err)
	}
}
