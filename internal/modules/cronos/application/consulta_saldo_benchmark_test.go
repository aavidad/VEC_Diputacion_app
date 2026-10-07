package application

import (
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

var resultadoBenchmarkSaldo ports.ConsultaSaldo
var marcajesBenchmarkSaldo [][]ports.MarcajeDia

func fuenteAnualBenchmarkSaldo(b *testing.B) (time.Time, time.Time, *time.Location, ports.FuenteSaldo) {
	b.Helper()
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		b.Fatal(err)
	}
	desde := time.Date(2026, 1, 1, 0, 0, 0, 0, zona)
	hasta := time.Date(2026, 12, 31, 0, 0, 0, 0, zona)
	fuente := ports.FuenteSaldo{Completo: true, Marcajes: make([]ports.MarcajeSaldo, 0, 10000)}
	for dia := 0; dia < 365; dia++ {
		fecha := desde.AddDate(0, 0, dia)
		clave := fecha.Format("2006-01-02")
		pares := 13
		if dia < 250 {
			pares = 14
		}
		if dia == 0 {
			pares += 5
		}
		inicio := time.Date(fecha.Year(), fecha.Month(), fecha.Day(), 8, 0, 0, 0, zona)
		for i := 0; i < pares; i++ {
			entrada := inicio.Add(time.Duration(i*20) * time.Minute).UTC()
			salida := entrada.Add(10 * time.Minute)
			fuente.Marcajes = append(fuente.Marcajes,
				ports.MarcajeSaldo{MarcajeRef: clave + "-e", Movimiento: domain.PunchEntry, InstanteUTC: entrada, Canal: canalSaldoPrueba(), OrigenRef: "terminal_1"},
				ports.MarcajeSaldo{MarcajeRef: clave + "-s", Movimiento: domain.PunchExit, InstanteUTC: salida, Canal: canalSaldoPrueba(), OrigenRef: "terminal_1"})
		}
		previsto := int64(pares * 10)
		fuente.Jornadas = append(fuente.Jornadas, ports.JornadaPrevista{Fecha: clave, TurnoRef: "turno", PoliticaVersionRef: "v1", MinutosPrevistos: previsto})
		fuente.MovimientosSaldo = append(fuente.MovimientosSaldo,
			ports.MovimientoSaldo{Fecha: clave, Tipo: "previsto", DeltaMicrosegundos: -previsto * 60 * 1000000, Fuentes: []string{"turno"}},
			ports.MovimientoSaldo{Fecha: clave, Tipo: "trabajado", DeltaMicrosegundos: previsto * 60 * 1000000, Fuentes: []string{"marcajes"}})
	}
	if len(fuente.Marcajes) != 10000 {
		b.Fatalf("marcajes=%d", len(fuente.Marcajes))
	}
	return desde, hasta, zona, fuente
}

func BenchmarkConstruirConsultaSaldoAnio10k(b *testing.B) {
	desde, hasta, zona, fuente := fuenteAnualBenchmarkSaldo(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultado, err := construirConsultaSaldo(ports.PeriodoSaldoAnio, desde, hasta, fuente, zona)
		if err != nil {
			b.Fatal(err)
		}
		resultadoBenchmarkSaldo = resultado
	}
	b.StopTimer()
	if len(resultadoBenchmarkSaldo.Detalle) != 365 || resultadoBenchmarkSaldo.Resumen.SaldoMinutos == nil || *resultadoBenchmarkSaldo.Resumen.SaldoMinutos != 0 || resultadoBenchmarkSaldo.Resumen.TrabajadosMinutos != 50000 {
		b.Fatalf("libro incoherente: %+v", resultadoBenchmarkSaldo.Resumen)
	}
}

// Compara únicamente la agrupación que antes recorría todos los marcajes por día.
func BenchmarkMarcajesSaldoAnio10k(b *testing.B) {
	desde, _, zona, fuente := fuenteAnualBenchmarkSaldo(b)
	fechas := make([]string, 365)
	for i := range fechas {
		fechas[i] = desde.AddDate(0, 0, i).Format("2006-01-02")
	}
	b.Run("busqueda_anterior", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			porDia := make([][]ports.MarcajeDia, 0, len(fechas))
			for _, clave := range fechas {
				marcajes := make([]ports.MarcajeDia, 0)
				for _, m := range fuente.Marcajes {
					if m.InstanteUTC.In(zona).Format("2006-01-02") == clave {
						marcajes = append(marcajes, ports.MarcajeDia{InstanteUTC: m.InstanteUTC, Movimiento: m.Movimiento, Origen: m.TipoOrigen})
					}
				}
				porDia = append(porDia, marcajes)
			}
			marcajesBenchmarkSaldo = porDia
		}
	})
	b.Run("agrupacion_actual", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			grupos := make(map[string][]ports.MarcajeDia)
			for _, m := range fuente.Marcajes {
				clave := m.InstanteUTC.In(zona).Format("2006-01-02")
				grupos[clave] = append(grupos[clave], ports.MarcajeDia{InstanteUTC: m.InstanteUTC, Movimiento: m.Movimiento, Origen: m.TipoOrigen})
			}
			porDia := make([][]ports.MarcajeDia, 0, len(fechas))
			for _, clave := range fechas {
				marcajes := make([]ports.MarcajeDia, 0)
				if grupo := grupos[clave]; len(grupo) != 0 {
					marcajes = grupo
				}
				porDia = append(porDia, marcajes)
			}
			marcajesBenchmarkSaldo = porDia
		}
	})
}
