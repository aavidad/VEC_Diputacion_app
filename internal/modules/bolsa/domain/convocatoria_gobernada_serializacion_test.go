package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func TestDecodificarVersionConvocatoriaGobernadaCanonicaRecuperaBytesExactos(t *testing.T) {
	original := versionConvocatoriaGobernadaPrueba(t)
	contenido, err := original.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}

	recuperada, err := DecodificarVersionConvocatoriaGobernadaCanonica(contenido)
	if err != nil {
		t.Fatalf("decodificar representacion canonica: %v", err)
	}
	representacionRecuperada, err := recuperada.RepresentacionCanonica()
	if err != nil || !bytes.Equal(representacionRecuperada, contenido) {
		t.Fatalf("la recuperacion no conserva la representacion exacta: %v", err)
	}
}

func TestDecodificarVersionConvocatoriaGobernadaCanonicaRechazaJSONMaleable(t *testing.T) {
	original := versionConvocatoriaGobernadaPrueba(t)
	contenido, err := original.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}

	var material map[string]any
	if err := json.Unmarshal(contenido, &material); err != nil {
		t.Fatal(err)
	}
	conDesconocido := append([]byte(`{"desconocido":true,`), contenido[1:]...)
	conDuplicado := append([]byte(`{"esquema":"bolsa.version-convocatoria.estado.v2",`), contenido[1:]...)
	conEspacio := append(append([]byte(nil), contenido...), ' ')
	conEsquemaDistinto := bytes.Replace(
		contenido,
		[]byte(`"bolsa.version-convocatoria.estado.v2"`),
		[]byte(`"bolsa.version-convocatoria.estado.v9"`),
		1,
	)
	conEsquemaHistorico := bytes.Replace(
		contenido,
		[]byte(`"bolsa.version-convocatoria.estado.v2"`),
		[]byte(`"bolsa.version-convocatoria.estado.v1"`),
		1,
	)

	casos := map[string][]byte{
		"vacio":               nil,
		"desconocido":         conDesconocido,
		"duplicado":           conDuplicado,
		"espacio final":       conEspacio,
		"esquema alternativo": conEsquemaDistinto,
		"esquema historico":   conEsquemaHistorico,
		"utf8 invalido":       {0xff, 0xfe},
	}
	for nombre, candidata := range casos {
		t.Run(nombre, func(t *testing.T) {
			if _, err := DecodificarVersionConvocatoriaGobernadaCanonica(candidata); !errors.Is(
				err, ErrVersionConvocatoriaGobernadaInvalida,
			) {
				t.Fatalf("se esperaba rechazo uniforme, recibido %v", err)
			}
		})
	}
}
