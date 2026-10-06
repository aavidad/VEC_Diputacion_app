package bootstrap

import (
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Coste de los plazos de una página de 100 filas, cada una con su entrada en
// fase propia como en el cuadro real: fila a fila (antes) y con las reglas
// leídas una vez (ahora).
func benchmarkPlazosFaseCTPagina100(b *testing.B, preparar bool) {
	t := &testing.T{}
	calculadora := calculadoraPlazoFaseCTPrueba(t, rutaReglasCTEjemploPrueba, calendariosPlazoFasePrueba(t))
	ahora := time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)
	fases := []domain.ClaveFase{"asignacion_unidad", "fiscalizacion", "solicitud", "subsanacion_unidad"}
	b.ReportAllocs()
	for range b.N {
		usada := calculadora
		if preparar {
			preparada, err := calculadora.(ports.PreparadorPlazosFaseRRHH).PrepararPlazosFase(b.Context())
			if err != nil {
				b.Fatal(err)
			}
			usada = preparada
		}
		for i := range 100 {
			desde := time.Date(2026, 9, 15, 9, 30, i, 0, time.UTC)
			_, _, _ = usada.CalcularPlazoFase(b.Context(), ports.SolicitudPlazoFaseRRHH{Fase: fases[i%len(fases)], Desde: desde, Ahora: ahora})
		}
	}
}

func BenchmarkPlazosFaseCTPagina100FilaAFila(b *testing.B) { benchmarkPlazosFaseCTPagina100(b, false) }
func BenchmarkPlazosFaseCTPagina100Preparada(b *testing.B) { benchmarkPlazosFaseCTPagina100(b, true) }
