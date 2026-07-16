package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding"
	"encoding/base64"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
)

func TestTokenReservaConservaHuellaHistoricaSinRevelarMaterial(t *testing.T) {
	secreto := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdefghijklmn"))
	token, err := NuevoTokenReservaBaremacion(secreto)
	if err != nil {
		t.Fatalf("NuevoTokenReservaBaremacion: %v", err)
	}
	esperada := sha256.Sum256([]byte(secreto))
	huellaEsperada := hex.EncodeToString(esperada[:])
	huella, err := token.HuellaSHA256()
	if err != nil || huella != huellaEsperada || !token.CoincideConHuellaSHA256(huellaEsperada) {
		t.Fatalf("formula SHA-256(Base64URL) alterada: obtenida=%q esperada=%q err=%v", huella, huellaEsperada, err)
	}
	for nombre, candidata := range map[string]string{
		"otro token":     strings.Repeat("0", sha256.Size*2),
		"mayusculas":     strings.ToUpper(huellaEsperada),
		"truncada":       huellaEsperada[:len(huellaEsperada)-1],
		"no hexadecimal": strings.Repeat("z", sha256.Size*2),
	} {
		if token.CoincideConHuellaSHA256(candidata) {
			t.Fatalf("huella %s admitida", nombre)
		}
	}
	copia := token
	huellaCopia, err := copia.HuellaSHA256()
	if err != nil || huellaCopia != huellaEsperada {
		t.Fatalf("la copia inmutable cambio de identidad: %q, %v", huellaCopia, err)
	}
	if (TokenReservaBaremacion{}).Validar() == nil ||
		(TokenReservaBaremacion{}).CoincideConHuellaSHA256(huellaEsperada) {
		t.Fatal("el valor cero concedio autoridad")
	}
}

func TestTokenReservaSoloBase64URLCanonico(t *testing.T) {
	secreto := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdefghijklmn"))
	invalidos := []string{
		"token con espacios", strings.Repeat("a", 31), strings.Repeat("a", 129),
		secreto + "=", "á" + secreto, "abc$" + secreto,
	}
	for _, invalido := range invalidos {
		if _, err := NuevoTokenReservaBaremacion(invalido); !errors.Is(err, ErrTokenReservaBaremacionInvalido) {
			t.Fatalf("token ambiguo admitido %q: %v", invalido, err)
		}
	}
}

func TestTokenReservaSoloExponeUnCierrePrivadoNoReflectible(t *testing.T) {
	secreto := base64.RawURLEncoding.EncodeToString([]byte("token-reflexion-segura-32-bytes"))
	token, err := NuevoTokenReservaBaremacion(secreto)
	if err != nil {
		t.Fatal(err)
	}
	tipo := reflect.TypeOf(token)
	if tipo.NumField() != 1 {
		t.Fatalf("TokenReservaBaremacion tiene %d campos; se esperaba un unico cierre", tipo.NumField())
	}
	campoTipo := tipo.Field(0)
	if campoTipo.Name != "operar" || campoTipo.IsExported() || campoTipo.Type.Kind() != reflect.Func {
		t.Fatalf("superficie reflectible inesperada: %+v", campoTipo)
	}
	if _, existe := tipo.MethodByName("Revelar"); existe {
		t.Fatal("TokenReservaBaremacion aun publica Revelar")
	}
	campo := reflect.ValueOf(token).Field(0)
	if campo.CanInterface() || campo.CanSet() {
		t.Fatal("el cierre privado puede convertirse o sustituirse mediante reflexion segura")
	}
	argumentos := make([]reflect.Value, campo.Type().NumIn())
	for indice := range argumentos {
		argumentos[indice] = reflect.Zero(campo.Type().In(indice))
	}
	exigirPanicoReflexionTokenReserva(t, func() { campo.Call(argumentos) })
	if salida := fmt.Sprintf("%v|%#v|%+v|%s|%q", token, token, token, token, token); strings.Contains(salida, secreto) ||
		strings.Count(salida, "[TOKEN-RESERVA-OCULTO]") != 5 {
		t.Fatalf("formateo no redactado: %q", salida)
	}
	var registro bytes.Buffer
	slog.New(slog.NewJSONHandler(&registro, nil)).Info("prueba", "token", token)
	if strings.Contains(registro.String(), secreto) || !strings.Contains(registro.String(), "TOKEN-RESERVA-OCULTO") {
		t.Fatalf("slog no redacto la capacidad: %s", registro.String())
	}
}

func exigirPanicoReflexionTokenReserva(t *testing.T, operacion func()) {
	t.Helper()
	detectado := false
	func() {
		defer func() { detectado = recover() != nil }()
		operacion()
	}()
	if !detectado {
		t.Fatal("la reflexion segura invoco el cierre privado")
	}
}

func TestTokenReservaBloqueaSerializacionYRestauracionGenericas(t *testing.T) {
	secreto := base64.RawURLEncoding.EncodeToString([]byte("token-serializacion-32-bytes"))
	token, err := NuevoTokenReservaBaremacion(secreto)
	if err != nil {
		t.Fatal(err)
	}
	envoltura := struct {
		XMLName xml.Name               `xml:"envoltura"`
		Token   TokenReservaBaremacion `xml:"token"`
	}{Token: token}
	for indice, valor := range []any{token, envoltura} {
		if contenido, err := json.Marshal(valor); contenido != nil ||
			!errors.Is(err, ErrSerializacionTokenReservaProhibida) {
			t.Fatalf("objeto %d admite JSON: contenido=%q error=%v", indice, contenido, err)
		}
		var binario bytes.Buffer
		if err := gob.NewEncoder(&binario).Encode(valor); err == nil ||
			!strings.Contains(err.Error(), ErrSerializacionTokenReservaProhibida.Error()) {
			t.Fatalf("objeto %d admite gob: contenido=%x error=%v", indice, binario.Bytes(), err)
		}
		if contenido, err := xml.Marshal(valor); contenido != nil ||
			!errors.Is(err, ErrSerializacionTokenReservaProhibida) {
			t.Fatalf("objeto %d admite XML: contenido=%q error=%v", indice, contenido, err)
		}
	}
	for nombre, serializar := range map[string]func() ([]byte, error){
		"texto":   token.MarshalText,
		"binario": token.MarshalBinary,
		"gob":     token.GobEncode,
	} {
		contenido, err := serializar()
		if contenido != nil || !errors.Is(err, ErrSerializacionTokenReservaProhibida) {
			t.Fatalf("%s admite serializacion: contenido=%q error=%v", nombre, contenido, err)
		}
	}

	var restaurado TokenReservaBaremacion
	for nombre, restaurar := range map[string]func() error{
		"json":    func() error { return json.Unmarshal([]byte(`"forjado"`), &restaurado) },
		"texto":   func() error { return restaurado.UnmarshalText([]byte("forjado")) },
		"binario": func() error { return restaurado.UnmarshalBinary([]byte("forjado")) },
		"gob":     func() error { return restaurado.GobDecode([]byte("forjado")) },
		"xml":     func() error { return xml.Unmarshal([]byte(`<token>forjado</token>`), &restaurado) },
	} {
		if err := restaurar(); !errors.Is(err, ErrSerializacionTokenReservaProhibida) {
			t.Fatalf("%s admite restauracion: %v", nombre, err)
		}
	}
	if restaurado.Validar() == nil {
		t.Fatal("una restauracion rechazada dejo una capacidad valida")
	}
	var _ encoding.TextMarshaler = token
	var _ encoding.TextUnmarshaler = &restaurado
	var _ encoding.BinaryMarshaler = token
	var _ encoding.BinaryUnmarshaler = &restaurado
}
