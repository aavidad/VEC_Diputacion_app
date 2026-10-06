package plazoarranque

import (
	"testing"
	"time"
)

func TestAnalizar(t *testing.T) {
	casos := map[string]time.Duration{"": 0, " ": 0, "1": time.Second, "300": 300 * time.Second, "60s": time.Minute, "600": 600 * time.Second}
	for v, esperado := range casos {
		if d, err := Analizar(v); err != nil || d != esperado {
			t.Fatalf("Analizar(%q)=%v,%v; esperado %v", v, d, err, esperado)
		}
	}
	for _, v := range []string{"0", "-1", "601", "1.5", "abc", "1m", "9999999999999999999", "18446744074", "10000000000"} {
		if _, err := Analizar(v); err == nil {
			t.Fatalf("Analizar(%q) debía fallar", v)
		}
	}
}

func TestAmpliarSoloDuranteElArranque(t *testing.T) {
	if Ampliar(3*time.Second) != 3*time.Second {
		t.Fatal("sin fijar no se amplía")
	}
	if err := Fijar(MaximoPlazo + time.Second); err == nil {
		t.Fatal("plazo excesivo admitido")
	}
	if err := Fijar(300 * time.Second); err != nil {
		t.Fatal(err)
	}
	_ = Fijar(time.Second) // la segunda vez no cambia nada
	if Ampliar(3*time.Second) != 300*time.Second || Ampliar(900*time.Second) != 900*time.Second {
		t.Fatal("durante el arranque se toma el mayor")
	}
	Terminar()
	if Ampliar(3*time.Second) != 3*time.Second {
		t.Fatal("tras el arranque el plazo es el declarado")
	}
	_ = Fijar(500 * time.Second)
	if Ampliar(2*time.Second) != 2*time.Second {
		t.Fatal("Fijar tras Terminar no tiene efecto")
	}
}
