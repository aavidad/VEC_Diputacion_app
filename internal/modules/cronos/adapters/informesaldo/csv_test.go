package informesaldo

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func ejemploSaldoCSV(t *testing.T) (ports.SaldoExportable, string) {
	t.Helper()
	var e struct {
		Nombre string
		Saldo  ports.SaldoExportable
	}
	raw, err := os.ReadFile("escenario_sintetico.json")
	if err != nil || json.Unmarshal(raw, &e) != nil {
		t.Fatal("fixture", err)
	}
	return e.Saldo, e.Nombre
}

func catalogoSaldoCSV(t *testing.T, idioma string) []byte {
	t.Helper()
	raw, err := os.ReadFile("../../../../../web/static/textos/" + idioma + "/cronos-informe-saldo-csv.json")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestSaldoCSVMinutosNegativosDesconocidosYSinReferencias(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			s, nombre := ejemploSaldoCSV(t)
			raw := catalogoSaldoCSV(t, idioma)
			var catalogo CatalogoCSV
			if err := json.Unmarshal(raw, &catalogo); err != nil {
				t.Fatal(err)
			}
			for _, desconocido := range []bool{false, true} {
				if desconocido {
					s.Resumen.Estado = ports.EstadoSaldoNoDisponible
					s.Resumen.PrevistosMinutos = nil
					s.Resumen.SaldoMinutos = nil
				}
				contenido, err := PrepararCSVEjemploSintetico(context.Background(), bytes.NewReader(raw), s, nombre)
				if err != nil {
					t.Fatal(err)
				}
				filas, err := csv.NewReader(bytes.NewReader(contenido)).ReadAll()
				if err != nil || len(filas) != 2 || len(filas[0]) != 8 || len(filas[1]) != 8 {
					t.Fatal("tabla irregular", err)
				}
				if filas[1][0] != nombre || filas[1][4] != "2220" {
					t.Fatal(filas)
				}
				if desconocido {
					if filas[1][3] != catalogo.Desconocido || filas[1][5] != catalogo.Desconocido {
						t.Fatal("nil no es cero", filas)
					}
				} else if filas[1][3] != "2250" || filas[1][5] != "-30" {
					t.Fatal("minutos numéricos", filas)
				}
				if strings.Contains(string(contenido), s.EmpleadoRef) || strings.Contains(string(contenido), s.FuenteRef) {
					t.Fatal("referencias expuestas")
				}
				if filas[0][5] != catalogo.Cabeceras["saldo_minutos"] || filas[1][7] != catalogo.Sintetico {
					t.Fatal("texto o aviso perdido", filas)
				}
			}
		})
	}
}

func TestSaldoCSVNeutralizaTodasLasCeldasTextuales(t *testing.T) {
	s, _ := ejemploSaldoCSV(t)
	var c CatalogoCSV
	if err := json.Unmarshal(catalogoSaldoCSV(t, "es"), &c); err != nil {
		t.Fatal(err)
	}
	for _, prefijo := range []string{"=", "+", "-", "@", " \u200b="} {
		for k := range c.Cabeceras {
			c.Cabeceras[k] = prefijo + "cabecera"
		}
		c.FormatoFecha = prefijo + "2006"
		c.Estados[s.Resumen.Estado] = prefijo + "estado"
		c.Sintetico = prefijo + "aviso"
		raw, err := json.Marshal(c)
		if err != nil {
			t.Fatal(err)
		}
		contenido, err := PrepararCSVEjemploSintetico(context.Background(), bytes.NewReader(raw), s, prefijo+"Carmen")
		if err != nil {
			t.Fatal(err)
		}
		filas, err := csv.NewReader(bytes.NewReader(contenido)).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		for _, valor := range filas[0] {
			if !strings.HasPrefix(valor, "'") {
				t.Fatal("cabecera fórmula", valor)
			}
		}
		for _, i := range []int{0, 1, 2, 6, 7} {
			if !strings.HasPrefix(filas[1][i], "'") {
				t.Fatal("celda fórmula", i, filas)
			}
		}
		if filas[1][5] != "-30" {
			t.Fatal("número convertido en texto")
		}
	}
}

func TestSaldoCSVFalloSinBytes(t *testing.T) {
	s, nombre := ejemploSaldoCSV(t)
	raw := catalogoSaldoCSV(t, "es")
	casos := []ports.SaldoExportable{s, s, s}
	casos[0].Periodo.Hasta = "2026-02-30"
	casos[1].Resumen.Estado = ports.EstadoSaldoNoDisponible
	casos[2].Resumen.TrabajadosMinutos = -1
	for _, invalido := range casos {
		contenido, err := PrepararCSVEjemploSintetico(context.Background(), bytes.NewReader(raw), invalido, nombre)
		if err == nil || len(contenido) != 0 {
			t.Fatal("saldo inválido aceptado")
		}
	}
	for _, invalido := range [][]byte{[]byte(`{}`), append(raw, []byte(`{}`)...), bytes.Replace(raw, []byte("cronos-saldo-csv-v1"), []byte("otro"), 1)} {
		contenido, err := PrepararCSVEjemploSintetico(context.Background(), bytes.NewReader(invalido), s, nombre)
		if err == nil || len(contenido) != 0 {
			t.Fatal("catálogo inválido aceptado")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	contenido, err := PrepararCSVEjemploSintetico(ctx, bytes.NewReader(raw), s, nombre)
	if !errors.Is(err, context.Canceled) || len(contenido) != 0 {
		t.Fatal("cancelación", err)
	}
}
