package documental

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

func TestSerializarCamposV3FijaLongitudEnBytesYCamposVacios(t *testing.T) {
	t.Parallel()

	valores := []string{"a", "", "á"}
	serializado := SerializarCamposV3(valores)
	if esperado := "1:a\n0:\n2:á\n"; string(serializado) != esperado {
		t.Fatalf("preimagen distinta: %q", serializado)
	}
	if huella := HuellaCamposSHA256V3(valores); huella != "2d1b86e71f5cbfc96191d995e576f716d2380281ff8ed4916a36f67d957776ca" {
		t.Fatalf("huella canonica distinta: %q", huella)
	}
	if HuellaBytesSHA256(serializado) != HuellaCamposSHA256V3(valores) {
		t.Fatal("la huella de campos no comparte la preimagen documentada")
	}
	if !BytesIguales(serializado, append([]byte(nil), serializado...)) || BytesIguales(serializado, nil) {
		t.Fatal("la comparacion de preimagenes no conserva igualdad exacta")
	}
}

func TestSerializarCamposV3ConservaFronterasDelCodecMaterialV4(t *testing.T) {
	t.Parallel()

	if SerializarCamposV3(nil) != nil || SerializarCamposV3([]string{}) != nil {
		t.Fatal("la coleccion vacia dejo de producir una preimagen nil")
	}
	valores := []string{"x:y\n", "\x00", "á"}
	esperada := []byte("4:x:y\n\n1:\x00\n2:á\n")
	obtenida := SerializarCamposV3(valores)
	if !bytes.Equal(obtenida, esperada) {
		t.Fatalf("preimagen de frontera distinta: %q", obtenida)
	}
	if HuellaBytesSHA256(obtenida) != HuellaCamposSHA256V3(valores) {
		t.Fatal("la huella de bytes y la de campos dejaron de compartir preimagen")
	}
	if HuellaBytesSHA256(nil) != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatal("la frontera SHA-256 de la preimagen vacia cambio")
	}
	malformado := string([]byte{0xff, 0xfe})
	if obtenido := SerializarCamposV3([]string{malformado}); !bytes.Equal(obtenido, []byte{'2', ':', 0xff, 0xfe, '\n'}) {
		t.Fatalf("se alteraron bytes UTF-8 malformados: %x", obtenido)
	}
}

func TestHuellasSHA256DistintasConservaFormaCardinalidadYFronteras(t *testing.T) {
	t.Parallel()

	a := strings.Repeat("a", 64)
	b := strings.Repeat("b", 64)
	if !HuellasSHA256Distintas() || !HuellasSHA256Distintas(a) ||
		!HuellasSHA256Distintas(a, b) {
		t.Fatal("se rechazo una coleccion canonica de huellas distintas")
	}
	for nombre, huellas := range map[string][]string{
		"repetida":  {a, a},
		"mayuscula": {a, strings.ToUpper(b)},
		"corta":     {a, strings.Repeat("b", 63)},
		"no hex":    {a, strings.Repeat("g", 64)},
		"espacio":   {a, b + " "},
	} {
		if HuellasSHA256Distintas(huellas...) {
			t.Errorf("%s: se acepto una coleccion no canonica", nombre)
		}
	}
}

func TestSHA256HexadecimalValidoEquivaleAlContratoHistorico(t *testing.T) {
	t.Parallel()

	referencia := func(valor string) bool {
		if len(valor) != sha256.Size*2 || valor != strings.TrimSpace(valor) ||
			valor != strings.ToLower(valor) {
			return false
		}
		decodificado, err := hex.DecodeString(valor)
		return err == nil && len(decodificado) == sha256.Size
	}
	comprobar := func(valor string) {
		t.Helper()
		if obtenido, esperado := SHA256HexadecimalValido(valor), referencia(valor); obtenido != esperado {
			t.Fatalf("validacion SHA-256 no equivalente: %q obtenido=%v esperado=%v",
				valor, obtenido, esperado)
		}
	}

	for longitud := 0; longitud <= sha256.Size*2+2; longitud++ {
		comprobar(strings.Repeat("a", longitud))
	}
	for _, base := range []string{
		strings.Repeat("0", sha256.Size*2),
		strings.Repeat("9", sha256.Size*2),
		strings.Repeat("a", sha256.Size*2),
		strings.Repeat("f", sha256.Size*2),
	} {
		for posicion := range len(base) {
			for octeto := 0; octeto <= 255; octeto++ {
				candidato := []byte(base)
				candidato[posicion] = byte(octeto)
				comprobar(string(candidato))
			}
		}
	}
	for _, valor := range []string{
		strings.Repeat("A", sha256.Size*2),
		strings.Repeat("F", sha256.Size*2),
		" " + strings.Repeat("a", sha256.Size*2-1),
		strings.Repeat("a", sha256.Size*2-1) + "\n",
		"á" + strings.Repeat("a", sha256.Size*2-2),
		"Ａ" + strings.Repeat("a", sha256.Size*2-3),
		"İ" + strings.Repeat("a", sha256.Size*2-2),
		string([]byte{0xff}) + strings.Repeat("a", sha256.Size*2-1),
	} {
		comprobar(valor)
	}
}

func TestReferenciaEjecucionV3ValidaSoloFormaOpaca(t *testing.T) {
	t.Parallel()

	for _, valor := range []string{
		"reserva:abc-123", "a", "objeto_1.version-2", strings.Repeat("a", 256),
	} {
		if !ReferenciaEjecucionV3Valida(valor) {
			t.Errorf("se rechazo referencia valida %q", valor)
		}
	}
	for _, valor := range []string{
		"", "A", "1abc", "reserva/abc", "reserva*", "https://ejemplo.test",
		"dni:12345678z", "nif:abc", "nie:x", "email:persona", "mailto:persona",
		"reserva con espacio", "reserva@dominio", strings.Repeat("a", 257), "referencia:á",
	} {
		if ReferenciaEjecucionV3Valida(valor) {
			t.Errorf("se acepto referencia no opaca %q", valor)
		}
	}
}

func TestInstanteV3ValidoExigeUTCYMicrosegundos(t *testing.T) {
	t.Parallel()

	valido := time.Date(2026, time.July, 18, 12, 30, 0, 123_456_000, time.UTC)
	if !InstanteV3Valido(valido) {
		t.Fatal("se rechazo instante UTC con precision de microsegundo")
	}
	for nombre, instante := range map[string]time.Time{
		"cero":        {},
		"otra zona":   valido.In(time.FixedZone("CEST", 2*60*60)),
		"nanosegundo": valido.Add(time.Nanosecond),
	} {
		if InstanteV3Valido(instante) {
			t.Errorf("%s: se acepto instante no canonico", nombre)
		}
	}
}

func TestClavesHMACSHA256V3DistingueDominios(t *testing.T) {
	t.Parallel()

	a := "hmac-sha256:clave-a:" + strings.Repeat("a", 64)
	b := "hmac-sha256:clave-b:" + strings.Repeat("b", 64)
	if ClaveHMACSHA256V3(a) != "clave-a" || !ClavesHMACSHA256V3Distintas(a, b) {
		t.Fatal("no se proyectaron claves HMAC validas y distintas")
	}
	for nombre, valores := range map[string][]string{
		"repetida":  {a, "hmac-sha256:clave-a:" + strings.Repeat("c", 64)},
		"vacia":     {a, "hmac-sha256::" + strings.Repeat("b", 64)},
		"algoritmo": {a, "sha256:clave-b:" + strings.Repeat("b", 64)},
		"segmentos": {a, "hmac-sha256:clave-b:uno:dos"},
	} {
		if ClavesHMACSHA256V3Distintas(valores...) {
			t.Errorf("%s: se acepto separacion de claves invalida", nombre)
		}
	}
	if Uint64Decimal(^uint64(0)) != "18446744073709551615" {
		t.Fatal("la representacion decimal uint64 cambio")
	}
}
