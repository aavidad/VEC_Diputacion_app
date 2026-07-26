package ports

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

type codecsMarshalConsultaResultadoCobertura interface {
	MarshalText() ([]byte, error)
	MarshalBinary() ([]byte, error)
	GobEncode() ([]byte, error)
	MarshalCBOR() ([]byte, error)
	MarshalYAML() (any, error)
}

type codecsUnmarshalConsultaResultadoCobertura interface {
	UnmarshalText([]byte) error
	UnmarshalBinary([]byte) error
	GobDecode([]byte) error
	UnmarshalCBOR([]byte) error
	UnmarshalYAML(func(any) error) error
}

func TestCapacidadesConsultaResultadoCoberturaBloqueanCodecsValorYPuntero(
	t *testing.T,
) {
	contexto := ContextoRecuperacionResultadoCobertura{}
	datos := DatosSolicitudLecturaResultadoCobertura{
		OrganizacionRef: "organizacion_serializacion_resultado_01",
		ExpedienteRef:   "expediente_serializacion_resultado_01",
		Accion:          AccionConsultarResultadoCobertura,
		Finalidad:       FinalidadRecuperarResultadoCobertura,
	}
	solicitud := SolicitudLecturaResultadoCobertura{}
	valores := map[string]codecsMarshalConsultaResultadoCobertura{
		"contexto_valor":    contexto,
		"contexto_puntero":  &contexto,
		"datos_valor":       datos,
		"datos_puntero":     &datos,
		"solicitud_valor":   solicitud,
		"solicitud_puntero": &solicitud,
	}
	for nombre, valor := range valores {
		t.Run(nombre, func(t *testing.T) {
			comprobarBloqueoMarshalConsultaResultadoCobertura(t, valor)
			formato := fmt.Sprintf("%+v", valor)
			var registro bytes.Buffer
			slog.New(slog.NewJSONHandler(&registro, nil)).Info(
				"capacidad_consulta",
				"valor",
				valor,
			)
			if !strings.Contains(formato, "REDACTAD") ||
				!strings.Contains(registro.String(), "REDACTAD") {
				t.Fatalf("%s perdió redacción", nombre)
			}
			for _, sensible := range []string{
				datos.OrganizacionRef,
				datos.ExpedienteRef,
				string(datos.Accion),
				string(datos.Finalidad),
			} {
				if strings.Contains(formato, sensible) ||
					strings.Contains(registro.String(), sensible) {
					t.Fatalf("%s filtró %q", nombre, sensible)
				}
			}
		})
	}

	destinos := map[string]any{
		"contexto":  &contexto,
		"datos":     &datos,
		"solicitud": &solicitud,
	}
	for nombre, destino := range destinos {
		t.Run(nombre+"_decode", func(t *testing.T) {
			comprobarErrorSerializacionConsultaResultadoCobertura(
				t,
				"json_decode",
				json.Unmarshal([]byte(`{}`), destino),
			)
			comprobarErrorSerializacionConsultaResultadoCobertura(
				t,
				"xml_decode",
				xml.Unmarshal([]byte(`<capacidad/>`), destino),
			)
			codecs, ok := destino.(codecsUnmarshalConsultaResultadoCobertura)
			if !ok {
				t.Fatalf("%T no expone bloqueos de reconstrucción", destino)
			}
			comprobarBloqueoUnmarshalConsultaResultadoCobertura(t, codecs)
		})
	}
}

func comprobarBloqueoMarshalConsultaResultadoCobertura(
	t *testing.T,
	valor codecsMarshalConsultaResultadoCobertura,
) {
	t.Helper()
	_, err := json.Marshal(valor)
	comprobarErrorSerializacionConsultaResultadoCobertura(t, "json", err)
	_, err = xml.Marshal(valor)
	comprobarErrorSerializacionConsultaResultadoCobertura(t, "xml", err)
	_, err = valor.MarshalText()
	comprobarErrorSerializacionConsultaResultadoCobertura(t, "text", err)
	_, err = valor.MarshalBinary()
	comprobarErrorSerializacionConsultaResultadoCobertura(t, "binary", err)
	var destino bytes.Buffer
	comprobarErrorSerializacionConsultaResultadoCobertura(
		t,
		"gob",
		gob.NewEncoder(&destino).Encode(valor),
	)
	_, err = valor.GobEncode()
	comprobarErrorSerializacionConsultaResultadoCobertura(
		t,
		"gob_directo",
		err,
	)
	_, err = valor.MarshalCBOR()
	comprobarErrorSerializacionConsultaResultadoCobertura(t, "cbor", err)
	_, err = valor.MarshalYAML()
	comprobarErrorSerializacionConsultaResultadoCobertura(t, "yaml", err)
}

func comprobarBloqueoUnmarshalConsultaResultadoCobertura(
	t *testing.T,
	valor codecsUnmarshalConsultaResultadoCobertura,
) {
	t.Helper()
	comprobarErrorSerializacionConsultaResultadoCobertura(
		t,
		"text_decode",
		valor.UnmarshalText([]byte("adulterado")),
	)
	comprobarErrorSerializacionConsultaResultadoCobertura(
		t,
		"binary_decode",
		valor.UnmarshalBinary([]byte("adulterado")),
	)
	comprobarErrorSerializacionConsultaResultadoCobertura(
		t,
		"gob_decode",
		valor.GobDecode([]byte("adulterado")),
	)
	comprobarErrorSerializacionConsultaResultadoCobertura(
		t,
		"cbor_decode",
		valor.UnmarshalCBOR([]byte("adulterado")),
	)
	comprobarErrorSerializacionConsultaResultadoCobertura(
		t,
		"yaml_decode",
		valor.UnmarshalYAML(func(any) error { return nil }),
	)
}

func comprobarErrorSerializacionConsultaResultadoCobertura(
	t *testing.T,
	nombre string,
	err error,
) {
	t.Helper()
	if !errors.Is(err, ErrSerializacionConsultaResultadoCoberturaProhibida) {
		t.Fatalf("%s no quedó bloqueado: %v", nombre, err)
	}
}
