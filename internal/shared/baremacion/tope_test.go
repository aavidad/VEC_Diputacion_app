package baremacion

import (
	"errors"
	"testing"
)

func TestAplicarTopeConservaDesgloseExacto(t *testing.T) {
	t.Parallel()
	for _, caso := range []struct {
		nombre                string
		bruto, tope, esperado int64
	}{
		{"inferior", 100, 200, 100},
		{"igual", 200, 200, 200},
		{"superior", 300, 200, 200},
		{"tope_cero", 300, 0, 0},
		{"bruto_cero", 0, 200, 0},
		{"ambos_cero", 0, 0, 0},
		{"maximo", MaximoMicropuntos, 1, 1},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			bruto, _ := PuntosDesdeMicropuntos(caso.bruto)
			tope, _ := PuntosDesdeMicropuntos(caso.tope)
			d, err := AplicarTope(bruto, tope)
			if err != nil {
				t.Fatal(err)
			}
			if d.Bruto() != bruto || d.Tope() != tope || d.Resultado().Micropuntos() != caso.esperado ||
				d.Exceso().Micropuntos() != caso.bruto-caso.esperado || d.Aplicado() != (caso.bruto > caso.tope) {
				t.Fatalf("desglose inesperado: %+v", d)
			}
			reconstruido, err := d.Resultado().Sumar(d.Exceso())
			if err != nil || reconstruido != bruto {
				t.Fatalf("perdida de bruto: %v, %v", reconstruido, err)
			}
			segunda, err := AplicarTope(d.Resultado(), tope)
			if err != nil || segunda.Resultado() != d.Resultado() || segunda.Aplicado() {
				t.Fatalf("tope no idempotente: %+v, %v", segunda, err)
			}
		})
	}
}

func TestAplicarTopeRechazaValoresInvalidosSinDesglose(t *testing.T) {
	t.Parallel()
	for _, invalido := range []Puntos{{micropuntos: -1}, {micropuntos: MaximoMicropuntos + 1}} {
		for _, pareja := range [][2]Puntos{{invalido, {}}, {{}, invalido}} {
			d, err := AplicarTope(pareja[0], pareja[1])
			var valor *ErrorValor
			if !errors.Is(err, ErrValorInvalido) || !errors.As(err, &valor) || valor.Tipo() != "tope" || d != (DesgloseTope{}) {
				t.Fatalf("rechazo incorrecto: %+v, %v", d, err)
			}
		}
	}
}
