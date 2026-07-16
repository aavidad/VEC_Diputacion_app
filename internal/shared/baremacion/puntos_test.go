package baremacion

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestPuntosConstruccionYOperaciones(t *testing.T) {
	t.Parallel()

	cero, err := PuntosDesdeMicropuntos(0)
	if err != nil || cero.Micropuntos() != 0 {
		t.Fatalf("PuntosDesdeMicropuntos(0) = %v, %v", cero, err)
	}
	if _, err := PuntosDesdeMicropuntos(-1); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("puntos negativos: error = %v", err)
	}
	if _, err := PuntosDesdeMicropuntos(MaximoMicropuntos + 1); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("puntos sobre el maximo: error = %v", err)
	}

	dos, _ := PuntosDesdeMicropuntos(2_000_000)
	tres, _ := PuntosDesdeMicropuntos(3_000_000)
	for _, caso := range []struct {
		izquierda Puntos
		derecha   Puntos
		esperado  int
	}{{dos, tres, -1}, {tres, dos, 1}, {dos, dos, 0}} {
		comparacion, err := caso.izquierda.Comparar(caso.derecha)
		if err != nil || comparacion != caso.esperado {
			t.Fatalf("Comparar(%v,%v) = %d, %v", caso.izquierda, caso.derecha, comparacion, err)
		}
	}
	suma, err := dos.Sumar(tres)
	if err != nil || suma.Micropuntos() != 5_000_000 {
		t.Fatalf("sumar = %d, %v", suma.Micropuntos(), err)
	}
	resta, err := tres.Restar(dos)
	if err != nil || resta.Micropuntos() != 1_000_000 {
		t.Fatalf("restar = %d, %v", resta.Micropuntos(), err)
	}
	if _, err := dos.Restar(tres); !errors.Is(err, ErrResultadoNegativo) {
		t.Fatalf("resta negativa: error = %v", err)
	}
}

func TestPuntosCierranDesbordamientoYRedondeoImplicito(t *testing.T) {
	t.Parallel()

	maximos, _ := PuntosDesdeMicropuntos(MaximoMicropuntos)
	uno, _ := PuntosDesdeMicropuntos(1)
	if _, err := maximos.Sumar(uno); !errors.Is(err, ErrDesbordamiento) {
		t.Fatalf("suma desbordada: error = %v", err)
	}

	tresPuntos, _ := PuntosDesdeMicropuntos(3_000_000)
	unTercio, _ := NuevoRacional(1, 3)
	resultado, err := tresPuntos.MultiplicarExacto(unTercio)
	if err != nil || resultado.Micropuntos() != 1_000_000 {
		t.Fatalf("3 puntos por 1/3 = %d, %v", resultado.Micropuntos(), err)
	}
	if _, err := uno.MultiplicarExacto(unTercio); !errors.Is(err, ErrResultadoNoExacto) {
		t.Fatalf("fraccion de micropunto: error = %v", err)
	}
	dos, _ := NuevoRacional(2, 1)
	if _, err := maximos.MultiplicarExacto(dos); !errors.Is(err, ErrDesbordamiento) {
		t.Fatalf("producto desbordado: error = %v", err)
	}
	menosUno, _ := NuevoRacional(-1, 1)
	if _, err := uno.MultiplicarExacto(menosUno); !errors.Is(err, ErrResultadoNegativo) {
		t.Fatalf("producto negativo: error = %v", err)
	}
}

func TestPuntosJSONCanonicoYExacto(t *testing.T) {
	t.Parallel()

	original, _ := PuntosDesdeMicropuntos(MaximoMicropuntos)
	datos, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(datos) != `"9000000000000000"` {
		t.Fatalf("JSON = %s", datos)
	}
	var recuperados Puntos
	if err := json.Unmarshal(datos, &recuperados); err != nil || recuperados != original {
		t.Fatalf("roundtrip = %v, %v", recuperados, err)
	}

	for _, noCanonico := range []string{`1000000`, `"01"`, `"+1"`, `"-0"`, `null`} {
		var destino Puntos
		if err := json.Unmarshal([]byte(noCanonico), &destino); !errors.Is(err, ErrValorNoCanonico) {
			t.Errorf("Unmarshal(%s): error = %v", noCanonico, err)
		}
	}
}

func FuzzPuntosRoundtripJSON(f *testing.F) {
	for _, semilla := range []int64{0, 1, MicropuntosPorPunto, MaximoMicropuntos, -1, MaximoMicropuntos + 1} {
		f.Add(semilla)
	}
	f.Fuzz(func(t *testing.T, micropuntos int64) {
		original, err := PuntosDesdeMicropuntos(micropuntos)
		if err != nil {
			return
		}
		datos, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal(%d): %v", micropuntos, err)
		}
		var recuperados Puntos
		if err := json.Unmarshal(datos, &recuperados); err != nil {
			t.Fatalf("Unmarshal(%s): %v", datos, err)
		}
		if recuperados != original {
			t.Fatalf("roundtrip: %v != %v", recuperados, original)
		}
	})
}
