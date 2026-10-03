package cargosct

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

type lectorFallido struct{ causa error }

func (l lectorFallido) Read([]byte) (int, error) { return 0, l.causa }

func TestClavesUnicasPropagaFalloDelLector(t *testing.T) {
	causa := errors.New("fallo_lector_sintetico")
	for _, prefijo := range []string{"", `{"`, `{"campo":`, `{"campo":1`} {
		t.Run(prefijo, func(t *testing.T) {
			lector := io.MultiReader(strings.NewReader(prefijo), lectorFallido{causa})
			if err := clavesUnicas(json.NewDecoder(lector)); !errors.Is(err, causa) {
				t.Fatal("el error del lector no llegó al clasificador")
			}
		})
	}
}

func TestClavesUnicasPropagaErrorSintactico(t *testing.T) {
	var sintaxis *json.SyntaxError
	if err := clavesUnicas(json.NewDecoder(strings.NewReader(`?`))); !errors.As(err, &sintaxis) {
		t.Fatal("el error sintáctico no llegó al clasificador")
	}
}

func TestClavesUnicasRechazaDuplicadasAnidadas(t *testing.T) {
	for _, material := range []string{`{"campo":1,"campo":2}`, `[{"campo":{"dato":1,"dato":2}}]`} {
		if err := clavesUnicas(json.NewDecoder(strings.NewReader(material))); !errors.Is(err, ErrRechazada) {
			t.Fatal("aceptó una clave duplicada")
		}
	}
}

func TestDecodificarMantieneRechazoOpaco(t *testing.T) {
	for _, material := range []string{`{"campo_ajeno":"valor_sintetico"}`, `{"campo":"valor_sintetico"`, `{"campo":"a","campo":"b"}`} {
		var destino struct {
			Campo string `json:"campo"`
		}
		if err := decodificar([]byte(material), &destino); err != ErrRechazada || err.Error() != "cargo_ct_rechazado" {
			t.Fatal("el clasificador no conservó el rechazo cerrado")
		}
	}
}
