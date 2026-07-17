package calculoexperienciaoficial

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

func TestReciboConsumoFuenteBloqueaCadaCodecEnAmbosSentidos(t *testing.T) {
	recibo := reciboConsumoFuentePrueba(t, datosReciboConsumoFuentePrueba{})
	var destino ReciboConsumoAutorizacionFuenteV1
	casos := []struct {
		nombre string
		usar   func() error
	}{
		{"json_marshal", func() error { _, err := json.Marshal(recibo); return err }},
		{"json_unmarshal", func() error { return json.Unmarshal([]byte(`{}`), &destino) }},
		{"xml_marshal", func() error { _, err := xml.Marshal(recibo); return err }},
		{"xml_unmarshal", func() error { return xml.Unmarshal([]byte(`<recibo/>`), &destino) }},
		{"texto_marshal", func() error { _, err := recibo.MarshalText(); return err }},
		{"texto_unmarshal", func() error { return destino.UnmarshalText([]byte("recibo")) }},
		{"binario_marshal", func() error { _, err := recibo.MarshalBinary(); return err }},
		{"binario_unmarshal", func() error { return destino.UnmarshalBinary([]byte{1}) }},
		{"gob_marshal", func() error {
			var salida bytes.Buffer
			return gob.NewEncoder(&salida).Encode(recibo)
		}},
		{"gob_unmarshal", func() error { return destino.GobDecode([]byte{1}) }},
		{"cbor_marshal", func() error { _, err := recibo.MarshalCBOR(); return err }},
		{"cbor_unmarshal", func() error { return destino.UnmarshalCBOR([]byte{1}) }},
		{"yaml_marshal", func() error { _, err := recibo.MarshalYAML(); return err }},
		{"yaml_unmarshal", func() error {
			return destino.UnmarshalYAML(func(any) error { return nil })
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if err := caso.usar(); !errors.Is(err, ErrEntradaNoPermitida) {
				t.Fatalf("codec directo permitido: %v", err)
			}
		})
	}
}

func TestReciboConsumoFuenteBloqueaValorCeroYPunteroNulo(t *testing.T) {
	var cero ReciboConsumoAutorizacionFuenteV1
	var nulo *ReciboConsumoAutorizacionFuenteV1
	marshalsCero := []func() error{
		func() error { _, err := cero.MarshalJSON(); return err },
		func() error { return cero.MarshalXML(nil, xml.StartElement{}) },
		func() error { _, err := cero.MarshalText(); return err },
		func() error { _, err := cero.MarshalBinary(); return err },
		func() error { _, err := cero.GobEncode(); return err },
		func() error { _, err := cero.MarshalCBOR(); return err },
		func() error { _, err := cero.MarshalYAML(); return err },
	}
	unmarshalsNulo := []func() error{
		func() error { return nulo.UnmarshalJSON([]byte(`{}`)) },
		func() error { return nulo.UnmarshalXML(nil, xml.StartElement{}) },
		func() error { return nulo.UnmarshalText([]byte("x")) },
		func() error { return nulo.UnmarshalBinary([]byte{1}) },
		func() error { return nulo.GobDecode([]byte{1}) },
		func() error { return nulo.UnmarshalCBOR([]byte{1}) },
		func() error { return nulo.UnmarshalYAML(func(any) error { return nil }) },
	}
	for indice, usar := range append(marshalsCero, unmarshalsNulo...) {
		if err := usar(); !errors.Is(err, ErrEntradaNoPermitida) {
			t.Fatalf("frontera %d no fallo cerrada: %v", indice, err)
		}
	}
	if cero.Validar() == nil || nulo != nil {
		t.Fatal("el valor cero adquirio validez")
	}
}

func TestReciboConsumoFuenteRedactaFmtYLogsSinFugas(t *testing.T) {
	recibo := reciboConsumoFuentePrueba(t, datosReciboConsumoFuentePrueba{})
	var registro bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&registro, nil))
	logger.Info("prueba", "recibo", recibo)
	textos := []string{
		recibo.String(), recibo.GoString(), recibo.LogValue().Resolve().String(),
		fmt.Sprint(recibo), fmt.Sprintf("%s", recibo), fmt.Sprintf("%q", recibo),
		fmt.Sprintf("%v", recibo), fmt.Sprintf("%+v", recibo), fmt.Sprintf("%#v", recibo),
		registro.String(),
	}
	secretos := []string{
		"decision:fuente:1", "fuente:recurso:1", "correlacion:lectura:1",
		"fuente:exacta:1", hashPrueba("a"), hashPrueba("c"),
	}
	for _, texto := range textos {
		if texto == "" || !strings.Contains(texto, textoReciboConsumoFuenteOculto) {
			t.Fatalf("salida no redactada: %q", texto)
		}
		for _, secreto := range secretos {
			if strings.Contains(texto, secreto) {
				t.Fatalf("se filtro material del recibo: %q", texto)
			}
		}
	}
}
