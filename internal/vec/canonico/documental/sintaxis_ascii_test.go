package documental

import (
	"math/rand"
	"regexp"
	"strings"
	"testing"
)

var (
	referenciaASCIIRegularPrueba = regexp.MustCompile(`^[a-z][a-z0-9._:-]{0,255}$`)
	idClaveASCIIRegularPrueba    = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,127}$`)
	dniNIEASCIIRegularPrueba     = regexp.MustCompile(`^(?:[0-9]{8}[a-z]|[xyz][0-9]{7}[a-z])$`)
)

func TestEscaneresASCIIEquivalenAExpresionesRegularesEnFronteras(t *testing.T) {
	comprobarEquivalenciaASCII(t, referenciaASCIIRegularPrueba, ReferenciaASCIIBasicaValida, 256, true)
	comprobarEquivalenciaASCII(t, idClaveASCIIRegularPrueba, IDClaveHMACASCIIBasicoValido, 128, false)

	for longitud := 0; longitud <= 12; longitud++ {
		base := strings.Repeat("0", longitud)
		for octeto := 0; octeto <= 255; octeto++ {
			for posicion := 0; posicion <= len(base); posicion++ {
				candidato := base[:posicion] + string([]byte{byte(octeto)}) + base[posicion:]
				if obtenido, esperado := DNINIEASCIIMinusculoEvidente(candidato),
					dniNIEASCIIRegularPrueba.MatchString(candidato); obtenido != esperado {
					t.Fatalf("DNI/NIE no equivalente: %q obtenido=%v esperado=%v", candidato, obtenido, esperado)
				}
			}
		}
	}
}

func TestDNINIEASCIIMinusculoEquivaleAlMutarCadaPosicion(t *testing.T) {
	t.Parallel()

	for _, base := range []string{"12345678a", "x1234567a", "y1234567a", "z1234567a"} {
		for posicion := range len(base) {
			for octeto := 0; octeto <= 255; octeto++ {
				candidato := []byte(base)
				candidato[posicion] = byte(octeto)
				valor := string(candidato)
				if obtenido, esperado := DNINIEASCIIMinusculoEvidente(valor),
					dniNIEASCIIRegularPrueba.MatchString(valor); obtenido != esperado {
					t.Fatalf("DNI/NIE mutado no equivalente: %q obtenido=%v esperado=%v",
						valor, obtenido, esperado)
				}
			}
		}
	}
}

func comprobarEquivalenciaASCII(
	t *testing.T,
	regular *regexp.Regexp,
	escaner func(string) bool,
	maximo int,
	admiteDosPuntos bool,
) {
	t.Helper()
	for _, longitud := range []int{0, 1, 2, maximo - 1, maximo, maximo + 1} {
		base := strings.Repeat("a", longitud)
		if escaner(base) != regular.MatchString(base) {
			t.Fatalf("frontera no equivalente: longitud=%d", longitud)
		}
		if longitud == 0 {
			continue
		}
		for octeto := 0; octeto <= 255; octeto++ {
			for _, posicion := range []int{0, longitud / 2, longitud - 1} {
				copia := []byte(base)
				copia[posicion] = byte(octeto)
				candidato := string(copia)
				if obtenido, esperado := escaner(candidato), regular.MatchString(candidato); obtenido != esperado {
					t.Fatalf("escaner no equivalente: %q dos_puntos=%v obtenido=%v esperado=%v",
						candidato, admiteDosPuntos, obtenido, esperado)
				}
			}
		}
	}
}

func TestEscaneresASCIIEquivalenciaDeterministaAmplia(t *testing.T) {
	aleatorio := rand.New(rand.NewSource(0x564543))
	for iteracion := 0; iteracion < 100_000; iteracion++ {
		longitud := aleatorio.Intn(300)
		contenido := make([]byte, longitud)
		for indice := range contenido {
			contenido[indice] = byte(aleatorio.Intn(256))
		}
		valor := string(contenido)
		if ReferenciaASCIIBasicaValida(valor) != referenciaASCIIRegularPrueba.MatchString(valor) ||
			IDClaveHMACASCIIBasicoValido(valor) != idClaveASCIIRegularPrueba.MatchString(valor) ||
			DNINIEASCIIMinusculoEvidente(valor) != dniNIEASCIIRegularPrueba.MatchString(valor) {
			t.Fatalf("corpus determinista no equivalente en iteracion %d: %q", iteracion, valor)
		}
	}
}

func FuzzEscaneresASCIIEquivalenARegulares(f *testing.F) {
	for _, semilla := range []string{"", "a", "a:1", strings.Repeat("a", 256), "12345678z", "x1234567l", "á"} {
		f.Add(semilla)
	}
	f.Fuzz(func(t *testing.T, valor string) {
		if ReferenciaASCIIBasicaValida(valor) != referenciaASCIIRegularPrueba.MatchString(valor) ||
			IDClaveHMACASCIIBasicoValido(valor) != idClaveASCIIRegularPrueba.MatchString(valor) ||
			DNINIEASCIIMinusculoEvidente(valor) != dniNIEASCIIRegularPrueba.MatchString(valor) {
			t.Fatalf("entrada no equivalente: %q", valor)
		}
	})
}
