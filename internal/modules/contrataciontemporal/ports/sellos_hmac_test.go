package ports

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

const dominioColeccionPrueba = "vec.contratacion-temporal.huella-peticion"

func TestColeccionSellosHMACConservaOrdenYCopiasDefensivas(t *testing.T) {
	activo := selloGeneracionalPrueba(dominioColeccionPrueba, 3, "c")
	retenidos := []string{
		selloGeneracionalPrueba(dominioColeccionPrueba, 2, "b"),
		selloGeneracionalPrueba(dominioColeccionPrueba, 1, "a"),
	}
	coleccion, err := NuevaColeccionSellosHMAC(activo, retenidos)
	if err != nil {
		t.Fatal(err)
	}
	retenidos[0] = selloGeneracionalPrueba(dominioColeccionPrueba, 2, "d")
	datos, err := coleccion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datos.Retenidos[0].Valor = "adulterado"
	releidos, err := coleccion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if releidos.Activo.Generacion != 3 ||
		releidos.Retenidos[0].Generacion != 2 ||
		releidos.Retenidos[1].Generacion != 1 ||
		!coleccion.Contiene(
			selloGeneracionalPrueba(dominioColeccionPrueba, 1, "a"),
		) || coleccion.Contiene("adulterado") {
		t.Fatalf("colección mutable o desordenada: %#v", releidos)
	}
}

func TestColeccionSellosHMACRechazaMatrizInsegura(t *testing.T) {
	activoV2 := selloGeneracionalPrueba(dominioColeccionPrueba, 2, "b")
	casos := map[string][]string{
		"generacion ascendente": {
			selloGeneracionalPrueba(dominioColeccionPrueba, 3, "c"),
		},
		"generacion repetida": {
			selloGeneracionalPrueba(dominioColeccionPrueba, 2, "c"),
		},
		"dominio cruzado": {
			selloGeneracionalPrueba(
				"vec.contratacion-temporal.ambito-idempotencia",
				1,
				"a",
			),
		},
		"demasiadas generaciones": {
			selloGeneracionalPrueba(dominioColeccionPrueba, 1, "a"),
			selloGeneracionalPrueba(dominioColeccionPrueba, 1, "b"),
			selloGeneracionalPrueba(dominioColeccionPrueba, 1, "c"),
			selloGeneracionalPrueba(dominioColeccionPrueba, 1, "d"),
		},
	}
	for nombre, retenidos := range casos {
		t.Run(nombre, func(t *testing.T) {
			_, err := NuevaColeccionSellosHMAC(activoV2, retenidos)
			if !errors.Is(err, ErrColeccionSellosHMACInvalida) {
				t.Fatalf("matriz insegura aceptada: %v", err)
			}
		})
	}
}

func selloGeneracionalPrueba(
	dominio string,
	generacion uint32,
	caracter string,
) string {
	return "hmac-sha256:" + dominio + "/v" +
		strconv.FormatUint(uint64(generacion), 10) + ":" +
		strings.Repeat(caracter, 64)
}
