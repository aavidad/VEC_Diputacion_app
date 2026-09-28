package httpinterno

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type lectorErrorCapacidadPolitica struct{ err error }

func (l lectorErrorCapacidadPolitica) Read([]byte) (int, error) { return 0, l.err }

func TestLeerBolsaCapacidadPoliticaConservaErrorDeLectura(t *testing.T) {
	fallo := errors.New("lectura interrumpida")
	for _, tc := range []struct {
		nombre, prefijo string
	}{
		{"inicio", ""},
		{"clave", "{"},
		{"valor", `{"bolsa_ref":`},
		{"fin", `{"bolsa_ref":"bolsa:prueba"`},
		{"resto", `{"bolsa_ref":"bolsa:prueba"}`},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			lector := io.MultiReader(strings.NewReader(tc.prefijo), lectorErrorCapacidadPolitica{fallo})
			bolsa, err := leerBolsaCapacidadPoliticaOfertas(lector)
			if bolsa != "" || !errors.Is(err, fallo) {
				t.Fatalf("bolsa=%q error=%v, se esperaba conservar el fallo de lectura", bolsa, err)
			}
		})
	}
}
