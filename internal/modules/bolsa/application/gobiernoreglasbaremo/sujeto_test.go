package gobiernoreglasbaremo

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestSujetoSeudonimoHMACAceptaSoloGramaticaCerrada(t *testing.T) {
	t.Parallel()
	huella := strings.Repeat("a", 64)
	validos := []string{"hmac-sha256:reglas_baremo_v2:" + huella}
	for _, valor := range validos {
		sujeto, err := RestaurarSujetoSeudonimoHMAC(valor)
		if err != nil {
			t.Fatalf("valor valido rechazado: %q: %v", valor, err)
		}
		recuperado, err := sujeto.ValorCanonico()
		if err != nil || recuperado != valor {
			t.Fatalf("valor alterado: %q, %v", recuperado, err)
		}
	}
}

func TestSujetoSeudonimoHMACRechazaEntradasAdversariales(t *testing.T) {
	t.Parallel()
	huella := strings.Repeat("a", 64)
	invalidos := []string{
		"",
		"hmac-sha256:otro_modulo_v2:" + huella,
		"hmac-sha256::" + huella,
		"hmac-sha256:0a:" + huella,
		"hmac-sha256:A:" + huella,
		"hmac-SHA256:a:" + huella,
		"hmac-sha256:a/:" + huella,
		"hmac-sha256:a b:" + huella,
		"hmac-sha256:a\nb:" + huella,
		"hmac-sha256:á:" + huella,
		"hmac-sha256:a:b:" + huella,
		"hmac-sha256:reglas-baremo-v2:" + huella,
		"hmac-sha256:a:" + strings.Repeat("a", 63),
		"hmac-sha256:a:" + strings.Repeat("a", 65),
		"hmac-sha256:a:" + strings.Repeat("A", 64),
		"hmac-sha256:a:" + strings.Repeat("g", 64),
		"hmac-sha256:a:" + huella + "x",
		"dni:12345678z",
		actorPlanPrueba,
	}
	for _, valor := range invalidos {
		if _, err := RestaurarSujetoSeudonimoHMAC(valor); !errors.Is(
			err, ErrSujetoSeudonimoHMACInvalido,
		) {
			t.Fatalf("entrada adversarial aceptada: %q, %v", valor, err)
		}
	}
	if _, err := (SujetoSeudonimoHMAC{}).ValorCanonico(); !errors.Is(
		err, ErrSujetoSeudonimoHMACInvalido,
	) {
		t.Fatalf("valor cero aceptado: %v", err)
	}
}

func TestSujetoSeudonimoHMACBloqueaCodecsYRedactaRegistros(t *testing.T) {
	t.Parallel()
	sujeto, err := RestaurarSujetoSeudonimoHMAC(sujetoSeudonimoPlanPrueba)
	if err != nil {
		t.Fatal(err)
	}
	serializadores := []struct {
		nombre   string
		ejecutar func() ([]byte, error)
	}{
		{"json", func() ([]byte, error) { return json.Marshal(sujeto) }},
		{"xml", func() ([]byte, error) { return xml.Marshal(sujeto) }},
		{"texto", sujeto.MarshalText},
		{"binario", sujeto.MarshalBinary},
		{"cbor", sujeto.MarshalCBOR},
		{"gob", func() ([]byte, error) {
			var destino bytes.Buffer
			err := gob.NewEncoder(&destino).Encode(sujeto)
			return destino.Bytes(), err
		}},
	}
	for _, serializador := range serializadores {
		serializado, err := serializador.ejecutar()
		if err == nil || bytes.Contains(serializado, []byte(sujetoSeudonimoPlanPrueba)) {
			t.Fatalf("%s filtro o permitio el sujeto: %q, %v", serializador.nombre, serializado, err)
		}
	}
	if valor, err := sujeto.MarshalYAML(); err == nil || valor != nil {
		t.Fatalf("YAML permitido: %#v, %v", valor, err)
	}

	formateado := fmt.Sprintf("%v %#v %+v", sujeto, sujeto, sujeto)
	if strings.Contains(formateado, sujetoSeudonimoPlanPrueba) ||
		!strings.Contains(formateado, "[GOBIERNO-REGLAS-BAREMO-OPACO]") {
		t.Fatalf("formato no redactado: %s", formateado)
	}
	var registro bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&registro, nil))
	logger.Info("prueba", "sujeto", sujeto)
	if strings.Contains(registro.String(), sujetoSeudonimoPlanPrueba) ||
		!strings.Contains(registro.String(), "GOBIERNO-REGLAS-BAREMO-OPACO") {
		t.Fatalf("registro no redactado: %s", registro.String())
	}
}
