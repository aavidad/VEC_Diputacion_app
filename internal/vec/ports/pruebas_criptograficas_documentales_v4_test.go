package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestSobreCriptograficoDocumentalCrudoV4CortaAliasYCalculaHuella(t *testing.T) {
	original := bytes.Repeat([]byte{0xa5}, minimoBytesSobreCriptograficoDocumentalCrudoV4)
	esperado := append([]byte(nil), original...)
	sobre, err := NuevoSobreCriptograficoDocumentalCrudoV4(original)
	if err != nil {
		t.Fatalf("crear sobre crudo: %v", err)
	}
	original[0] ^= 0xff

	primera, err := sobre.COSESign1()
	if err != nil || !bytes.Equal(primera, esperado) {
		t.Fatalf("el constructor conservo un alias de entrada: %x, %v", primera, err)
	}
	primera[1] ^= 0xff
	segunda, err := sobre.COSESign1()
	if err != nil || !bytes.Equal(segunda, esperado) {
		t.Fatalf("el accessor expuso un alias mutable: %x, %v", segunda, err)
	}

	huella := sha256.Sum256(esperado)
	huellaObtenida, err := sobre.HuellaSHA256()
	if err != nil || huellaObtenida != hex.EncodeToString(huella[:]) {
		t.Fatalf("huella inesperada: %q, %v", huellaObtenida, err)
	}
}

func TestSobreCriptograficoDocumentalCrudoV4AplicaLimitesSinInterpretarAutoridad(t *testing.T) {
	casos := [][]byte{
		nil,
		bytes.Repeat([]byte{0}, minimoBytesSobreCriptograficoDocumentalCrudoV4),
		bytes.Repeat([]byte{1}, minimoBytesSobreCriptograficoDocumentalCrudoV4-1),
		bytes.Repeat([]byte{1}, maximoBytesSobreCriptograficoDocumentalCrudoV4+1),
	}
	if _, err := NuevoSobreCriptograficoDocumentalCrudoV4(
		bytes.Repeat([]byte{1}, maximoBytesSobreCriptograficoDocumentalCrudoV4),
	); err != nil {
		t.Fatalf("el maximo global estricto fue rechazado: %v", err)
	}
	for _, contenido := range casos {
		if _, err := NuevoSobreCriptograficoDocumentalCrudoV4(contenido); !errors.Is(
			err, ErrSobreCriptograficoDocumentalCrudoV4Invalido,
		) {
			t.Fatalf("contenido invalido aceptado, longitud=%d: %v", len(contenido), err)
		}
	}

	// El puerto acepta bytes no nulos de tamano acotado aunque no sean COSE:
	// interpretar y verificar pertenece exclusivamente al nucleo local.
	if _, err := NuevoSobreCriptograficoDocumentalCrudoV4(
		bytes.Repeat([]byte{0x7f}, minimoBytesSobreCriptograficoDocumentalCrudoV4),
	); err != nil {
		t.Fatalf("el transporte crudo intento conceder o denegar autoridad: %v", err)
	}
}

func TestPruebasCriptograficasDocumentalesCrudasV4SonNominalesYNoAutoritativas(t *testing.T) {
	sobre := nuevoSobreCriptograficoDocumentalCrudoV4Prueba(t)
	recibo, err := NuevaPruebaCrudaReciboComponenteDocumentalV4(sobre)
	if err != nil {
		t.Fatalf("crear prueba de recibo: %v", err)
	}
	token, err := NuevaPruebaCrudaTokenCercadoDocumentalV4(sobre)
	if err != nil {
		t.Fatalf("crear prueba de token: %v", err)
	}
	firma, err := NuevaFirmaCrudaEvidenciaDocumentalV4(sobre)
	if err != nil {
		t.Fatalf("crear firma de evidencia: %v", err)
	}
	reconciliacion, err := NuevaAtestacionCrudaReconciliacionDocumentalV4(sobre)
	if err != nil {
		t.Fatalf("crear atestacion de reconciliacion: %v", err)
	}

	valores := []any{recibo, token, firma, reconciliacion}
	tipos := make(map[reflect.Type]struct{}, len(valores))
	for _, valor := range valores {
		tipo := reflect.TypeOf(valor)
		tipos[tipo] = struct{}{}
		for _, nombreProhibido := range []string{
			"Autoriza", "EsAutoritativa", "EstaVerificada", "ValidarPara", "Verificada",
		} {
			if _, existe := tipo.MethodByName(nombreProhibido); existe {
				t.Fatalf("%s concede autoridad mediante %s", tipo, nombreProhibido)
			}
		}
	}
	if len(tipos) != len(valores) {
		t.Fatal("dos protocolos crudos comparten el mismo tipo nominal")
	}
}

func TestPruebasCriptograficasDocumentalesCrudasV4RechazanValorCero(t *testing.T) {
	var sobre SobreCriptograficoDocumentalCrudoV4
	if sobre.ValidarSintaxis() == nil {
		t.Fatal("el sobre cero fue valido")
	}
	for nombre, validar := range map[string]func() error{
		"recibo":         func() error { return (PruebaCrudaReciboComponenteDocumentalV4{}).ValidarSintaxis() },
		"token":          func() error { return (PruebaCrudaTokenCercadoDocumentalV4{}).ValidarSintaxis() },
		"evidencia":      func() error { return (FirmaCrudaEvidenciaDocumentalV4{}).ValidarSintaxis() },
		"reconciliacion": func() error { return (AtestacionCrudaReconciliacionDocumentalV4{}).ValidarSintaxis() },
	} {
		if err := validar(); !errors.Is(err, ErrPruebaCriptograficaDocumentalCrudaV4Invalida) {
			t.Fatalf("%s cero fue valido: %v", nombre, err)
		}
	}
}

func TestPruebasCriptograficasDocumentalesCrudasV4SeRedactanYNoSeSerializan(t *testing.T) {
	sobre := nuevoSobreCriptograficoDocumentalCrudoV4Prueba(t)
	recibo, _ := NuevaPruebaCrudaReciboComponenteDocumentalV4(sobre)
	token, _ := NuevaPruebaCrudaTokenCercadoDocumentalV4(sobre)
	firma, _ := NuevaFirmaCrudaEvidenciaDocumentalV4(sobre)
	reconciliacion, _ := NuevaAtestacionCrudaReconciliacionDocumentalV4(sobre)

	for nombre, valor := range map[string]any{
		"sobre": sobre, "recibo": recibo, "token": token,
		"firma": firma, "reconciliacion": reconciliacion,
	} {
		texto := fmt.Sprintf("%v|%+v|%#v|%s", valor, valor, valor, valor)
		if strings.Contains(strings.ToLower(texto), "a5a5a5") {
			t.Fatalf("%s filtro bytes: %s", nombre, texto)
		}
		if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionPruebaCriptograficaCrudaV4) {
			t.Fatalf("%s se serializo por JSON: %v", nombre, err)
		}
		marshalerTexto, ok := valor.(interface{ MarshalText() ([]byte, error) })
		if !ok {
			t.Fatalf("%s no bloquea serializacion textual", nombre)
		}
		if _, err := marshalerTexto.MarshalText(); !errors.Is(
			err, ErrSerializacionPruebaCriptograficaCrudaV4,
		) {
			t.Fatalf("%s se serializo como texto: %v", nombre, err)
		}
		marshalerBinario, ok := valor.(interface{ MarshalBinary() ([]byte, error) })
		if !ok {
			t.Fatalf("%s no bloquea serializacion binaria", nombre)
		}
		if _, err := marshalerBinario.MarshalBinary(); !errors.Is(
			err, ErrSerializacionPruebaCriptograficaCrudaV4,
		) {
			t.Fatalf("%s se serializo como binario: %v", nombre, err)
		}
	}

	var restaurado SobreCriptograficoDocumentalCrudoV4
	if err := json.Unmarshal([]byte(`{}`), &restaurado); !errors.Is(
		err, ErrSerializacionPruebaCriptograficaCrudaV4,
	) {
		t.Fatalf("el sobre se restauro mediante JSON: %v", err)
	}
}

func nuevoSobreCriptograficoDocumentalCrudoV4Prueba(
	t *testing.T,
) SobreCriptograficoDocumentalCrudoV4 {
	t.Helper()
	sobre, err := NuevoSobreCriptograficoDocumentalCrudoV4(
		bytes.Repeat([]byte{0xa5}, minimoBytesSobreCriptograficoDocumentalCrudoV4),
	)
	if err != nil {
		t.Fatalf("crear sobre criptografico de prueba: %v", err)
	}
	return sobre
}
