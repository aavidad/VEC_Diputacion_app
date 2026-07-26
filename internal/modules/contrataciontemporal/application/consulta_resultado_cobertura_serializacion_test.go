package application

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

type codecsMarshalSolicitudConsultaResultadoCobertura interface {
	MarshalText() ([]byte, error)
	MarshalBinary() ([]byte, error)
	GobEncode() ([]byte, error)
	MarshalCBOR() ([]byte, error)
	MarshalYAML() (any, error)
}

func TestSolicitudConsultaResultadoCoberturaBloqueaSerializacionValorYPuntero(
	t *testing.T,
) {
	solicitud := SolicitudConsultaResultadoCobertura{
		ClaveIdempotencia: "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
		ExpedienteRef:     "expediente_serializacion_resultado_01",
	}
	for nombre, valor := range map[string]codecsMarshalSolicitudConsultaResultadoCobertura{
		"valor":   solicitud,
		"puntero": &solicitud,
	} {
		t.Run(nombre, func(t *testing.T) {
			comprobarBloqueoMarshalSolicitudConsultaResultadoCobertura(
				t,
				valor,
			)
			formato := fmt.Sprintf("%+v", valor)
			var registro bytes.Buffer
			slog.New(slog.NewJSONHandler(&registro, nil)).Info(
				"solicitud_consulta",
				"valor",
				valor,
			)
			for _, sensible := range []string{
				solicitud.ClaveIdempotencia,
				solicitud.ExpedienteRef,
			} {
				if strings.Contains(formato, sensible) ||
					strings.Contains(registro.String(), sensible) {
					t.Fatalf("%s filtró %q", nombre, sensible)
				}
			}
			if !strings.Contains(formato, "REDACTAD") ||
				!strings.Contains(registro.String(), "REDACTAD") {
				t.Fatalf("%s perdió redacción", nombre)
			}
		})
	}

	var destino SolicitudConsultaResultadoCobertura
	comprobarErrorSerializacionSolicitudConsultaResultadoCobertura(
		t,
		"json_decode",
		json.Unmarshal([]byte(`{}`), &destino),
	)
	comprobarErrorSerializacionSolicitudConsultaResultadoCobertura(
		t,
		"xml_decode",
		xml.Unmarshal([]byte(`<solicitud/>`), &destino),
	)
	comprobarErrorSerializacionSolicitudConsultaResultadoCobertura(
		t,
		"text_decode",
		destino.UnmarshalText([]byte("adulterado")),
	)
	comprobarErrorSerializacionSolicitudConsultaResultadoCobertura(
		t,
		"binary_decode",
		destino.UnmarshalBinary([]byte("adulterado")),
	)
	comprobarErrorSerializacionSolicitudConsultaResultadoCobertura(
		t,
		"gob_decode",
		destino.GobDecode([]byte("adulterado")),
	)
	comprobarErrorSerializacionSolicitudConsultaResultadoCobertura(
		t,
		"cbor_decode",
		destino.UnmarshalCBOR([]byte("adulterado")),
	)
	comprobarErrorSerializacionSolicitudConsultaResultadoCobertura(
		t,
		"yaml_decode",
		destino.UnmarshalYAML(func(any) error { return nil }),
	)
}

func comprobarBloqueoMarshalSolicitudConsultaResultadoCobertura(
	t *testing.T,
	valor codecsMarshalSolicitudConsultaResultadoCobertura,
) {
	t.Helper()
	_, err := json.Marshal(valor)
	comprobarErrorSerializacionSolicitudConsultaResultadoCobertura(t, "json", err)
	_, err = xml.Marshal(valor)
	comprobarErrorSerializacionSolicitudConsultaResultadoCobertura(t, "xml", err)
	_, err = valor.MarshalText()
	comprobarErrorSerializacionSolicitudConsultaResultadoCobertura(t, "text", err)
	_, err = valor.MarshalBinary()
	comprobarErrorSerializacionSolicitudConsultaResultadoCobertura(t, "binary", err)
	var destino bytes.Buffer
	comprobarErrorSerializacionSolicitudConsultaResultadoCobertura(
		t,
		"gob",
		gob.NewEncoder(&destino).Encode(valor),
	)
	_, err = valor.GobEncode()
	comprobarErrorSerializacionSolicitudConsultaResultadoCobertura(
		t,
		"gob_directo",
		err,
	)
	_, err = valor.MarshalCBOR()
	comprobarErrorSerializacionSolicitudConsultaResultadoCobertura(t, "cbor", err)
	_, err = valor.MarshalYAML()
	comprobarErrorSerializacionSolicitudConsultaResultadoCobertura(t, "yaml", err)
}

func comprobarErrorSerializacionSolicitudConsultaResultadoCobertura(
	t *testing.T,
	nombre string,
	err error,
) {
	t.Helper()
	if !errors.Is(
		err,
		ErrSerializacionSolicitudConsultaResultadoCoberturaProhibida,
	) {
		t.Fatalf("%s no quedó bloqueado: %v", nombre, err)
	}
}

func TestDatosConsultaResultadoCoberturaParaAdaptadorSigueSerializable(
	t *testing.T,
) {
	salida := DatosConsultaResultadoCoberturaParaAdaptador{
		Estado: ResultadoCoberturaNoObservable,
	}
	if _, err := json.Marshal(salida); err != nil {
		t.Fatalf("DTO de salida dejó de ser JSON serializable: %v", err)
	}
	if _, err := xml.Marshal(salida); err != nil {
		t.Fatalf("DTO de salida dejó de ser XML serializable: %v", err)
	}
	var destino bytes.Buffer
	if err := gob.NewEncoder(&destino).Encode(salida); err != nil {
		t.Fatalf("DTO de salida dejó de ser gob serializable: %v", err)
	}
}
