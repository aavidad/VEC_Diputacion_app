package postgres

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestConstruirVersionAceptaSoloRepresentacionCanonicaSellada(t *testing.T) {
	t.Parallel()
	agregado := baremacionPostgreSQLPrueba(t)
	representacion, err := json.Marshal(agregado)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := agregado.HuellaEstadoSHA256()
	if err != nil {
		t.Fatal(err)
	}

	version, err := construirVersion(
		agregado.ID, "1", huella, representacion, instantePostgreSQLPrueba,
	)
	if err != nil {
		t.Fatalf("representacion canonica rechazada: %v", err)
	}
	if version.Referencia.HuellaEstadoSHA256 != huella || version.Agregado.ID != agregado.ID {
		t.Fatal("la version leida no conserva el agregado sellado")
	}
}

func TestDecodificarAgregadoCanonicoRechazaRepresentacionesJSONAlternativas(t *testing.T) {
	t.Parallel()
	agregado := baremacionPostgreSQLPrueba(t)
	representacion, err := json.Marshal(agregado)
	if err != nil {
		t.Fatal(err)
	}

	duplicada := append([]byte(`{"id":"`+agregado.ID+`",`), representacion[1:]...)
	conEspacio := append([]byte{'{', ' '}, representacion[1:]...)
	ordenAlterado := []byte(strings.Replace(
		string(representacion),
		`{"id":"`+agregado.ID+`","proceso_ref":"`+agregado.ProcesoRef+`"`,
		`{"proceso_ref":"`+agregado.ProcesoRef+`","id":"`+agregado.ID+`"`,
		1,
	))
	escapeEquivalente := []byte(strings.Replace(
		string(representacion), "baremacion:postgresql:prueba",
		`baremacion\u003apostgresql:prueba`, 1,
	))
	utf8Invalido := append([]byte(nil), representacion...)
	posicion := bytes.Index(utf8Invalido, []byte("postgresql"))
	if posicion < 0 {
		t.Fatal("la precondicion de la prueba UTF-8 no se cumple")
	}
	utf8Invalido[posicion] = 0xff
	conTrailing := append(append([]byte(nil), representacion...), ' ')
	desconocida := append([]byte(`{"propiedad_desconocida":true,`), representacion[1:]...)

	casos := map[string][]byte{
		"clave duplicada":     duplicada,
		"espacio":             conEspacio,
		"orden alterado":      ordenAlterado,
		"escape equivalente":  escapeEquivalente,
		"utf8 invalido":       utf8Invalido,
		"contenido posterior": conTrailing,
		"clave desconocida":   desconocida,
	}
	for nombre, contenido := range casos {
		nombre, contenido := nombre, contenido
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			_, err := decodificarAgregadoCanonico(contenido)
			if !errors.Is(err, puertosbolsa.ErrEvidenciaBaremacionNoConfiable) {
				t.Fatalf("representacion no canonica admitida; error=%v", err)
			}
		})
	}
}

func TestDecodificarAgregadoCanonicoRechazaExcesoAntesDeDecodificar(t *testing.T) {
	t.Parallel()
	contenido := bytes.Repeat([]byte{' '}, maximoBytesAgregadoCanonico+1)
	_, err := decodificarAgregadoCanonico(contenido)
	if !errors.Is(err, puertosbolsa.ErrEvidenciaBaremacionNoConfiable) {
		t.Fatalf("agregado sobredimensionado admitido; error=%v", err)
	}
}
