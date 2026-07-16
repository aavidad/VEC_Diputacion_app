package baremacion

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalFallidoNoMutaReceptores(t *testing.T) {
	t.Parallel()

	t.Run("puntos", func(t *testing.T) {
		original, _ := PuntosDesdeMicropuntos(3_000_000)
		destino := original
		if err := json.Unmarshal([]byte(`"03"`), &destino); err == nil {
			t.Fatal("se aceptaron puntos no canonicos")
		}
		if destino != original {
			t.Fatalf("receptor mutado: %v != %v", destino, original)
		}
	})

	t.Run("racional", func(t *testing.T) {
		original, _ := NuevoRacional(1, 3)
		destino := original
		if err := json.Unmarshal([]byte(`"2/6"`), &destino); err == nil {
			t.Fatal("se acepto un racional no canonico")
		}
		if destino != original {
			t.Fatalf("receptor mutado: %v != %v", destino, original)
		}
	})

	t.Run("fraccion jornada", func(t *testing.T) {
		original, _ := NuevaFraccionJornada(1, 2)
		destino := original
		if err := json.Unmarshal([]byte(`"2/1"`), &destino); err == nil {
			t.Fatal("se acepto una jornada fuera de limites")
		}
		if destino != original {
			t.Fatalf("receptor mutado: %v != %v", destino, original)
		}
	})

	t.Run("fecha civil", func(t *testing.T) {
		original, _ := NuevaFechaCivil(2024, 2, 29)
		destino := original
		if err := json.Unmarshal([]byte(`"2023-02-29"`), &destino); err == nil {
			t.Fatal("se acepto una fecha inexistente")
		}
		if destino != original {
			t.Fatalf("receptor mutado: %v != %v", destino, original)
		}
	})

	t.Run("intervalo civil", func(t *testing.T) {
		desde, _ := NuevaFechaCivil(2026, 1, 1)
		hasta, _ := NuevaFechaCivil(2026, 2, 1)
		original, _ := NuevoIntervaloCivil(desde, hasta)
		destino := original
		duplicado := `{"desde":"2026-01-01","desde":"2026-01-02","hasta":"2026-02-01"}`
		if err := json.Unmarshal([]byte(duplicado), &destino); err == nil {
			t.Fatal("se acepto una clave duplicada")
		}
		if destino != original {
			t.Fatalf("receptor mutado: %v != %v", destino, original)
		}
	})
}
