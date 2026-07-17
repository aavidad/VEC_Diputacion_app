package application

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestResultadoConsultaInternaAutoridadNoCruzaFronterasDeSerializacion(t *testing.T) {
	fuente := fuenteConsultaInternaAutoridadPrueba(t)
	textoSensible := fuente.Contenido.Preceptos[0].Cita
	estado, err := ports.EstadoExactoFuenteAutoridad(fuente)
	if err != nil {
		t.Fatal(err)
	}
	resultado := ResultadoConsultaInternaExactaFuenteAutoridad{
		Encontrada:   true,
		Fuente:       fuente,
		EstadoExacto: estado,
	}

	serializaciones := []func() error{
		func() error { _, err := json.Marshal(resultado); return err },
		func() error { _, err := xml.Marshal(resultado); return err },
		func() error {
			var destino bytes.Buffer
			return gob.NewEncoder(&destino).Encode(resultado)
		},
		func() error { _, err := resultado.MarshalText(); return err },
		func() error { _, err := resultado.MarshalBinary(); return err },
		func() error { _, err := resultado.MarshalCBOR(); return err },
		func() error { _, err := resultado.MarshalYAML(); return err },
	}
	for indice, serializar := range serializaciones {
		if err := serializar(); !errors.Is(err, ErrSerializacionResultadoConsultaInternaAutoridad) {
			t.Fatalf("serializacion %d aceptada: %v", indice, err)
		}
	}

	var reconstruido ResultadoConsultaInternaExactaFuenteAutoridad
	if err := json.Unmarshal([]byte(`{}`), &reconstruido); !errors.Is(err, ErrSerializacionResultadoConsultaInternaAutoridad) {
		t.Fatalf("reconstruccion JSON aceptada: %v", err)
	}
	if err := xml.Unmarshal([]byte(`<resultado/>`), &reconstruido); !errors.Is(err, ErrSerializacionResultadoConsultaInternaAutoridad) {
		t.Fatalf("reconstruccion XML aceptada: %v", err)
	}
	if err := reconstruido.UnmarshalCBOR([]byte{0xa0}); !errors.Is(err, ErrSerializacionResultadoConsultaInternaAutoridad) {
		t.Fatalf("reconstruccion CBOR aceptada: %v", err)
	}
	if err := reconstruido.UnmarshalYAML(func(any) error { return nil }); !errors.Is(err, ErrSerializacionResultadoConsultaInternaAutoridad) {
		t.Fatalf("reconstruccion YAML aceptada: %v", err)
	}

	formateado := fmt.Sprintf("%v %+v %#v", resultado, resultado, resultado)
	for _, sensible := range []string{fuente.ID, fuente.Contenido.Documento.DocumentoID, textoSensible} {
		if strings.Contains(formateado, sensible) {
			t.Fatalf("formato expuso %q: %s", sensible, formateado)
		}
	}
}
