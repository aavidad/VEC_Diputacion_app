package gobiernoconvocatorias

import (
	"bytes"
	"encoding"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"testing"

	cbor "github.com/fxamacker/cbor/v2"
	"gopkg.in/yaml.v3"
)

func TestClaveClienteExigeFormaCanonicaSinAfirmarEntropia(t *testing.T) {
	material := make([]byte, 32)
	for indice := range material {
		material[indice] = byte(indice)
	}
	forma := base64.RawURLEncoding.EncodeToString(material)
	if len(forma) != 43 {
		t.Fatalf("longitud base64url inesperada: %d", len(forma))
	}
	clave, err := NuevaClaveClienteIdempotenciaConvocatoria(forma)
	if err != nil || !clave.Valida() {
		t.Fatalf("clave canonica rechazada: %v", err)
	}

	// Una muestra de 32 bytes cero tiene forma correcta. Aceptarla demuestra
	// que este constructor no pretende estimar entropia observando el valor.
	formaCero := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	claveCero, err := NuevaClaveClienteIdempotenciaConvocatoria(formaCero)
	if err != nil || !claveCero.Valida() {
		t.Fatalf("la validacion de forma infirio entropia: %v", err)
	}
	if (ClaveClienteIdempotenciaConvocatoria{}).Valida() {
		t.Fatal("el valor cero no construido se considero valido")
	}

	invalidas := []string{
		"", forma + "=", forma[:42], strings.Repeat("*", 43),
		strings.Repeat("A", 42) + "B", // bits finales no canonicos en modo estricto.
	}
	for _, invalida := range invalidas {
		if _, err := NuevaClaveClienteIdempotenciaConvocatoria(invalida); !errors.Is(
			err, ErrClaveClienteIdempotenciaInvalida,
		) {
			t.Errorf("forma invalida aceptada (%d caracteres): %v", len(invalida), err)
		}
	}
}

func TestLocalizadorYHuellaSonHMACNominalesSeparadosYVersionados(t *testing.T) {
	valorA := strings.Repeat("a", 64)
	valorB := strings.Repeat("b", 64)
	claveLocalizador, _ := NuevaReferenciaClaveHMACLocalizador(
		"clave:hmac:convocatorias:localizador:principal", 7,
	)
	claveLocalizadorOtraGeneracion, _ := NuevaReferenciaClaveHMACLocalizador(
		"clave:hmac:convocatorias:localizador:principal", 8,
	)
	claveLocalizadorOtraReferencia, _ := NuevaReferenciaClaveHMACLocalizador(
		"clave:hmac:convocatorias:localizador:rotada", 7,
	)
	claveHuella, _ := NuevaReferenciaClaveHMACHuellaSolicitud(
		"clave:hmac:convocatorias:huella:orden", 7,
	)
	localizador, err := NuevoLocalizadorOperacion(1, claveLocalizador, valorA)
	if err != nil || !localizador.Valido() {
		t.Fatalf("localizador valido rechazado: %v", err)
	}
	localizadorIgual, _ := NuevoLocalizadorOperacion(1, claveLocalizador, valorA)
	localizadorOtraVersion, _ := NuevoLocalizadorOperacion(2, claveLocalizador, valorA)
	localizadorOtraGeneracion, _ := NuevoLocalizadorOperacion(1, claveLocalizadorOtraGeneracion, valorA)
	localizadorOtraReferencia, _ := NuevoLocalizadorOperacion(1, claveLocalizadorOtraReferencia, valorA)
	localizadorOtroValor, _ := NuevoLocalizadorOperacion(1, claveLocalizador, valorB)
	if !localizador.CoincideExactamente(localizadorIgual) ||
		localizador.CoincideExactamente(localizadorOtraVersion) ||
		localizador.CoincideExactamente(localizadorOtraGeneracion) ||
		localizador.CoincideExactamente(localizadorOtraReferencia) ||
		localizador.CoincideExactamente(localizadorOtroValor) {
		t.Fatal("la coincidencia exacta omitio version, clave, generacion o valor")
	}
	huella, err := NuevaHuellaSolicitud(1, claveHuella, valorA)
	if err != nil || !huella.Valida() {
		t.Fatalf("huella valida rechazada: %v", err)
	}
	if reflect.TypeOf(localizador) == reflect.TypeOf(huella) {
		t.Fatal("L y F comparten el mismo tipo nominal")
	}
	if (LocalizadorOperacion{}).Valido() || (HuellaSolicitud{}).Valida() {
		t.Fatal("un HMAC nominal cero fue valido")
	}
	localizadorCero, err := NuevoLocalizadorOperacion(1, claveLocalizador, strings.Repeat("0", 64))
	if err != nil || !localizadorCero.Valido() {
		t.Fatalf("un MAC todo-cero con forma valida se confundio con el struct cero: %v", err)
	}
	for _, caso := range []struct {
		version uint16
		clave   ReferenciaClaveHMAC
		valor   string
	}{
		{0, claveLocalizador, valorA}, {1, ReferenciaClaveHMAC{}, valorA},
		{1, claveLocalizador, strings.Repeat("a", 63)},
		{1, claveLocalizador, strings.Repeat("A", 64)},
		{1, claveLocalizador, strings.Repeat("g", 64)},
	} {
		if _, err := NuevoLocalizadorOperacion(caso.version, caso.clave, caso.valor); !errors.Is(
			err, ErrHMACIdempotenciaInvalido,
		) {
			t.Errorf("HMAC invalido aceptado: %+v: %v", caso, err)
		}
	}
	if _, err := NuevoLocalizadorOperacion(1, claveHuella, valorA); !errors.Is(
		err, ErrHMACIdempotenciaInvalido,
	) {
		t.Fatalf("L acepto la clave nominal de F: %v", err)
	}
	if _, err := NuevaHuellaSolicitud(1, claveLocalizador, valorA); !errors.Is(
		err, ErrHMACIdempotenciaInvalido,
	) {
		t.Fatalf("F acepto la clave nominal de L: %v", err)
	}
}

func TestReferenciasClaveHMACSonEstrictasYVersionadas(t *testing.T) {
	validas := []func(string, uint32) (ReferenciaClaveHMAC, error){
		NuevaReferenciaClaveHMACLocalizador,
		NuevaReferenciaClaveHMACHuellaSolicitud,
	}
	referencias := []string{
		"clave:hmac:convocatorias:localizador:principal",
		"clave:hmac:convocatorias:huella:orden",
	}
	for indice, construir := range validas {
		if clave, err := construir(referencias[indice], 1); err != nil || !clave.valida() {
			t.Fatalf("referencia nominal valida rechazada: %v", err)
		}
	}
	prefijo := "clave:hmac:convocatorias:localizador:"
	referencia127 := prefijo + strings.Repeat("a", 127-len(prefijo))
	referencia128 := prefijo + strings.Repeat("b", 128-len(prefijo))
	for _, referencia := range []string{referencia127, referencia128} {
		if clave, err := NuevaReferenciaClaveHMACLocalizador(referencia, 1); err != nil || !clave.valida() {
			t.Errorf("referencia limite %d rechazada: %v", len(referencia), err)
		}
	}
	for _, caso := range []struct {
		construir  func(string, uint32) (ReferenciaClaveHMAC, error)
		referencia string
		generacion uint32
	}{
		{NuevaReferenciaClaveHMACLocalizador, "clave:hmac:convocatorias:huella:orden", 1},
		{NuevaReferenciaClaveHMACHuellaSolicitud, "clave:hmac:convocatorias:localizador:principal", 1},
		{NuevaReferenciaClaveHMACLocalizador, "CLAVE:hmac:convocatorias:localizador:x", 1},
		{NuevaReferenciaClaveHMACLocalizador, "clave:hmac:convocatorias:localizador:x/", 1},
		{NuevaReferenciaClaveHMACLocalizador, "clave:hmac:convocatorias:localizador:x", 0},
		{NuevaReferenciaClaveHMACLocalizador, prefijo + strings.Repeat("x", 129-len(prefijo)), 1},
	} {
		if _, err := caso.construir(caso.referencia, caso.generacion); !errors.Is(
			err, ErrHMACIdempotenciaInvalido,
		) {
			t.Errorf("referencia de clave invalida aceptada: %q g%d: %v", caso.referencia, caso.generacion, err)
		}
	}
	claveCorta, _ := NuevaReferenciaClaveHMACLocalizador(prefijo+"comun", 1)
	claveLarga, _ := NuevaReferenciaClaveHMACLocalizador(prefijo+"comunx", 1)
	localizadorCorto, _ := NuevoLocalizadorOperacion(1, claveCorta, strings.Repeat("c", 64))
	localizadorLargo, _ := NuevoLocalizadorOperacion(1, claveLarga, strings.Repeat("c", 64))
	if localizadorCorto.CoincideExactamente(localizadorLargo) {
		t.Fatal("referencias con prefijo comun y distinta longitud coincidieron")
	}
	forjada := ReferenciaClaveHMAC{
		dominio: dominioClaveHMACLocalizador, longitud: 1,
		generacionClave: 1, definida: true,
	}
	forjada.referencia[0] = '*'
	if forjada.valida() {
		t.Fatal("descriptor de clave forjado se considero valido")
	}
	if _, err := NuevoLocalizadorOperacion(1, forjada, strings.Repeat("d", 64)); !errors.Is(
		err, ErrHMACIdempotenciaInvalido,
	) {
		t.Fatalf("HMAC acepto descriptor forjado: %v", err)
	}
	if (ReferenciaClaveHMAC{}).valida() {
		t.Fatal("descriptor cero se considero valido")
	}
}

func TestValoresIdempotenciaCierranCodecsFormatoYLogs(t *testing.T) {
	forma := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x5a}, 32))
	clave, _ := NuevaClaveClienteIdempotenciaConvocatoria(forma)
	claveLocalizador, _ := NuevaReferenciaClaveHMACLocalizador(
		"clave:hmac:convocatorias:localizador:principal", 1,
	)
	claveHuella, _ := NuevaReferenciaClaveHMACHuellaSolicitud(
		"clave:hmac:convocatorias:huella:orden", 1,
	)
	localizador, _ := NuevoLocalizadorOperacion(1, claveLocalizador, strings.Repeat("a", 64))
	huella, _ := NuevaHuellaSolicitud(1, claveHuella, strings.Repeat("b", 64))
	casos := []struct {
		nombre  string
		valor   any
		destino any
		secreto string
	}{
		{"clave", clave, &ClaveClienteIdempotenciaConvocatoria{}, forma},
		{"clave_hmac", claveLocalizador, &ReferenciaClaveHMAC{}, "clave:hmac:convocatorias:localizador:principal"},
		{"localizador", localizador, &LocalizadorOperacion{}, strings.Repeat("a", 64)},
		{"huella", huella, &HuellaSolicitud{}, strings.Repeat("b", 64)},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			comprobarCodecsDiarioCerrados(t, caso.valor, caso.destino)
			for _, formato := range []string{"%v", "%+v", "%#v"} {
				salida := fmt.Sprintf(formato, caso.valor)
				if strings.Contains(salida, caso.secreto) || !strings.Contains(salida, "PROTEGIDO") {
					t.Fatalf("fmt %s filtro o no redacto el valor: %s", formato, salida)
				}
			}
			registro := slog.AnyValue(caso.valor).Resolve().String()
			if strings.Contains(registro, caso.secreto) || !strings.Contains(registro, "PROTEGIDO") {
				t.Fatalf("slog filtro o no redacto el valor: %s", registro)
			}
		})
	}
}

func TestComparacionConcurrenteNoMutaHMACNominal(t *testing.T) {
	clave, _ := NuevaReferenciaClaveHMACLocalizador(
		"clave:hmac:convocatorias:localizador:principal", 1,
	)
	localizador, _ := NuevoLocalizadorOperacion(1, clave, strings.Repeat("c", 64))
	const trabajadores = 100
	var espera sync.WaitGroup
	espera.Add(trabajadores)
	errores := make(chan error, trabajadores)
	for indice := 0; indice < trabajadores; indice++ {
		go func() {
			defer espera.Done()
			if !localizador.CoincideExactamente(localizador) {
				errores <- errors.New("comparacion exacta inestable")
			}
		}()
	}
	espera.Wait()
	close(errores)
	for err := range errores {
		t.Error(err)
	}
}

func comprobarCodecsDiarioCerrados(t *testing.T, valor, destino any) {
	t.Helper()
	if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionDiarioProhibida) {
		t.Fatalf("JSON de salida abierto: %v", err)
	}
	if err := json.Unmarshal([]byte(`{}`), destino); !errors.Is(err, ErrSerializacionDiarioProhibida) {
		t.Fatalf("JSON de entrada abierto: %v", err)
	}
	texto, ok := valor.(encoding.TextMarshaler)
	if !ok {
		t.Fatal("falta TextMarshaler")
	}
	if contenido, err := texto.MarshalText(); contenido != nil || !errors.Is(err, ErrSerializacionDiarioProhibida) {
		t.Fatalf("texto abierto: %q, %v", contenido, err)
	}
	destinoTexto, ok := destino.(encoding.TextUnmarshaler)
	if !ok {
		t.Fatal("falta TextUnmarshaler")
	}
	if err := destinoTexto.UnmarshalText([]byte("fabricado")); !errors.Is(err, ErrSerializacionDiarioProhibida) {
		t.Fatalf("texto de entrada abierto: %v", err)
	}
	binario, ok := valor.(encoding.BinaryMarshaler)
	if !ok {
		t.Fatal("falta BinaryMarshaler")
	}
	if contenido, err := binario.MarshalBinary(); contenido != nil || !errors.Is(err, ErrSerializacionDiarioProhibida) {
		t.Fatalf("binario abierto: %q, %v", contenido, err)
	}
	destinoBinario, ok := destino.(encoding.BinaryUnmarshaler)
	if !ok {
		t.Fatal("falta BinaryUnmarshaler")
	}
	if err := destinoBinario.UnmarshalBinary([]byte("fabricado")); !errors.Is(err, ErrSerializacionDiarioProhibida) {
		t.Fatalf("binario de entrada abierto: %v", err)
	}
	var destinoGob bytes.Buffer
	if err := gob.NewEncoder(&destinoGob).Encode(valor); !errors.Is(err, ErrSerializacionDiarioProhibida) {
		t.Fatalf("Gob de salida abierto: %v", err)
	}
	destinoDecodificadorGob, ok := destino.(gob.GobDecoder)
	if !ok {
		t.Fatal("falta GobDecoder")
	}
	if err := destinoDecodificadorGob.GobDecode([]byte("fabricado")); !errors.Is(
		err, ErrSerializacionDiarioProhibida,
	) {
		t.Fatalf("Gob de entrada abierto: %v", err)
	}
	if _, err := xml.Marshal(valor); !errors.Is(err, ErrSerializacionDiarioProhibida) {
		t.Fatalf("XML de salida abierto: %v", err)
	}
	if err := xml.Unmarshal([]byte(`<valor/>`), destino); !errors.Is(err, ErrSerializacionDiarioProhibida) {
		t.Fatalf("XML de entrada abierto: %v", err)
	}
	if _, err := cbor.Marshal(valor); !errors.Is(err, ErrSerializacionDiarioProhibida) {
		t.Fatalf("CBOR de salida abierto: %v", err)
	}
	if err := cbor.Unmarshal([]byte{0xa0}, destino); !errors.Is(err, ErrSerializacionDiarioProhibida) {
		t.Fatalf("CBOR de entrada abierto: %v", err)
	}
	if _, err := yaml.Marshal(valor); !errors.Is(err, ErrSerializacionDiarioProhibida) {
		t.Fatalf("YAML de salida abierto: %v", err)
	}
	if err := yaml.Unmarshal([]byte("{}"), destino); !errors.Is(err, ErrSerializacionDiarioProhibida) {
		t.Fatalf("YAML de entrada abierto: %v", err)
	}
}
