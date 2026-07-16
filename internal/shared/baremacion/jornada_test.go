package baremacion

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
)

func TestFraccionJornadaEsExactaYAcotada(t *testing.T) {
	t.Parallel()

	unTercio, err := NuevaFraccionJornada(1, 3)
	if err != nil || unTercio.String() != "1/3" {
		t.Fatalf("un tercio = %s, %v", unTercio, err)
	}
	if unTercio.Numerador() != 1 || unTercio.Denominador() != 3 {
		t.Fatalf("componentes = %d/%d", unTercio.Numerador(), unTercio.Denominador())
	}
	aproximacion, err := NuevaFraccionJornada(3333, 10000)
	if err != nil {
		t.Fatalf("aproximacion: %v", err)
	}
	comparacion, err := unTercio.Racional().Comparar(aproximacion.Racional())
	if err != nil || comparacion <= 0 {
		t.Fatalf("1/3 debe ser mayor que 3333/10000: comparacion %d, error %v", comparacion, err)
	}
	if unTercio == aproximacion {
		t.Fatal("1/3 no puede ser igual a 3333/10000")
	}

	if !JornadaCompleta().EsCompleta() {
		t.Fatal("JornadaCompleta no es completa")
	}
	for _, caso := range [][2]int64{{0, 1}, {-1, 2}, {3, 2}} {
		if _, err := NuevaFraccionJornada(caso[0], caso[1]); !errors.Is(err, ErrFueraDeLimites) {
			t.Errorf("jornada %d/%d: error = %v", caso[0], caso[1], err)
		}
	}
	if _, err := NuevaFraccionJornada(1, 0); !errors.Is(err, ErrDenominadorCero) {
		t.Fatalf("denominador cero: error = %v", err)
	}
}

func TestFraccionJornadaAtribuyeSusErroresAlTipoPublico(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nombre      string
		numerador   int64
		denominador int64
		centinela   error
	}{
		{"denominador cero", 1, 0, ErrDenominadorCero},
		{"componente excesivo", math.MinInt64, 1, ErrFueraDeLimites},
		{"fuera del intervalo", 2, 1, ErrFueraDeLimites},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			_, err := NuevaFraccionJornada(caso.numerador, caso.denominador)
			if !errors.Is(err, caso.centinela) {
				t.Fatalf("error = %v", err)
			}
			var errorValor *ErrorValor
			if !errors.As(err, &errorValor) || errorValor.Tipo() != "fraccion_jornada" {
				t.Fatalf("tipo de error = %v", err)
			}
		})
	}

	var destino FraccionJornada
	err := json.Unmarshal([]byte(`"1/0"`), &destino)
	var errorValor *ErrorValor
	if !errors.Is(err, ErrDenominadorCero) || !errors.As(err, &errorValor) ||
		errorValor.Tipo() != "fraccion_jornada" {
		t.Fatalf("error al decodificar = %v", err)
	}
}

func TestFraccionJornadaRoundtripJSON(t *testing.T) {
	t.Parallel()

	original, _ := NuevaFraccionJornada(1, 3)
	datos, err := json.Marshal(original)
	if err != nil || string(datos) != `"1/3"` {
		t.Fatalf("Marshal = %s, %v", datos, err)
	}
	var recuperada FraccionJornada
	if err := json.Unmarshal(datos, &recuperada); err != nil || recuperada != original {
		t.Fatalf("roundtrip = %s, %v", recuperada, err)
	}
	for _, invalida := range []string{`"0/1"`, `"2/1"`, `"2/6"`, `0.5`} {
		var destino FraccionJornada
		if err := json.Unmarshal([]byte(invalida), &destino); err == nil {
			t.Errorf("Unmarshal(%s) no rechazo", invalida)
		}
	}
}
