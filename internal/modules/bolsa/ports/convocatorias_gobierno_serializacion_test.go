package ports

import (
	"bytes"
	"encoding"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"testing"
)

func TestValoresInternosCierranTodosLosSerializadoresEstandar(t *testing.T) {
	casos := []struct {
		nombre  string
		valor   any
		destino any
	}{
		{"solicitud_consulta", SolicitudConsultaVersionConvocatoriaAutorizada{}, &SolicitudConsultaVersionConvocatoriaAutorizada{}},
		{"resultado_consulta", ResultadoConsultaVersionConvocatoria{}, &ResultadoConsultaVersionConvocatoria{}},
		{"solicitud_idempotencia", SolicitudProtegerIdempotenciaConvocatoria{}, &SolicitudProtegerIdempotenciaConvocatoria{}},
		{"datos_idempotencia", DatosTestimonioIdempotenciaConvocatoria{}, &DatosTestimonioIdempotenciaConvocatoria{}},
		{"testimonio_idempotencia", TestimonioIdempotenciaConvocatoria{}, &TestimonioIdempotenciaConvocatoria{}},
		{"solicitud_motivo", SolicitudSellarMotivoGobiernoConvocatoria{}, &SolicitudSellarMotivoGobiernoConvocatoria{}},
		{"hmac_motivo", HMACMotivoGobiernoConvocatoria{}, &HMACMotivoGobiernoConvocatoria{}},
		{"datos_sellado_motivo", DatosAtestacionSelladoMotivoConvocatoria{}, &DatosAtestacionSelladoMotivoConvocatoria{}},
		{"atestacion_sellado_motivo", AtestacionSelladoMotivoConvocatoria{}, &AtestacionSelladoMotivoConvocatoria{}},
		{"solicitud_dependencias", SolicitudVerificarDependenciasConvocatoria{}, &SolicitudVerificarDependenciasConvocatoria{}},
		{"datos_dependencias", DatosAtestacionDependenciasConvocatoria{}, &DatosAtestacionDependenciasConvocatoria{}},
		{"atestacion_dependencias", AtestacionDependenciasConvocatoria{}, &AtestacionDependenciasConvocatoria{}},
		{"solicitud_aprobacion", SolicitudComprobarAprobacionConvocatoria{}, &SolicitudComprobarAprobacionConvocatoria{}},
		{"datos_aprobacion", DatosAtestacionAprobacionConvocatoria{}, &DatosAtestacionAprobacionConvocatoria{}},
		{"atestacion_aprobacion", AtestacionAprobacionConvocatoria{}, &AtestacionAprobacionConvocatoria{}},
		{"consumo_recibo", ReciboConsumoVerificacionConvocatoria{}, &ReciboConsumoVerificacionConvocatoria{}},
		{"recibo", ReciboGobiernoConvocatoria{}, &ReciboGobiernoConvocatoria{}},
		{"preparacion", PreparacionTransaccionGobiernoConvocatoria{}, &PreparacionTransaccionGobiernoConvocatoria{}},
		{"confirmacion_alta", ConfirmacionAltaBorradorConvocatoria{}, &ConfirmacionAltaBorradorConvocatoria{}},
		{"confirmacion_actualizacion", ConfirmacionActualizacionBorradorConvocatoria{}, &ConfirmacionActualizacionBorradorConvocatoria{}},
		{"confirmacion_publicacion", ConfirmacionPublicacionConvocatoria{}, &ConfirmacionPublicacionConvocatoria{}},
		{"confirmacion_retirada", ConfirmacionRetiradaConvocatoria{}, &ConfirmacionRetiradaConvocatoria{}},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			comprobarInterfacesSerializacionCerradas(t, caso.valor, caso.destino)
			if _, err := json.Marshal(caso.valor); !esErrorSerializacionGobierno(err) {
				t.Fatalf("JSON permitio codificar el valor interno: %v", err)
			}
			if err := json.Unmarshal([]byte(`{}`), caso.destino); !esErrorSerializacionGobierno(err) {
				t.Fatalf("JSON permitio reconstruir el valor interno: %v", err)
			}
			if _, err := xml.Marshal(caso.valor); !esErrorSerializacionGobierno(err) {
				t.Fatalf("XML permitio codificar el valor interno: %v", err)
			}
			if err := xml.Unmarshal([]byte(`<interno/>`), caso.destino); !esErrorSerializacionGobierno(err) {
				t.Fatalf("XML permitio reconstruir el valor interno: %v", err)
			}
			var salida bytes.Buffer
			if err := gob.NewEncoder(&salida).Encode(caso.valor); !esErrorSerializacionGobierno(err) {
				t.Fatalf("gob permitio codificar el valor interno: %v", err)
			}
		})
	}
}

func TestMaterialIntencionMantieneRepresentacionCanonicaSerializable(t *testing.T) {
	version := versionGobernadaPuertosPrueba(t)
	material, err := MaterialAltaBorradorConvocatoria(
		version, nil, nil, atestacionMotivoConvocatoriaPrueba(
			t, AccionCrearBorradorConvocatoria, version,
			version.CreadaPor, version.MotivoCreacion, 'a',
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := json.Marshal(material)
	if err != nil {
		t.Fatalf("material canonico no serializable: %v", err)
	}
	var reconstruido MaterialIntencionGobiernoConvocatoria
	if err := json.Unmarshal(contenido, &reconstruido); err != nil {
		t.Fatalf("material canonico no reconstruible: %v", err)
	}
	huellaOriginal, errOriginal := material.HuellaSHA256()
	huellaReconstruida, errReconstruida := reconstruido.HuellaSHA256()
	if errOriginal != nil || errReconstruida != nil || huellaOriginal != huellaReconstruida {
		t.Fatal("la representacion canonica altero la intencion")
	}
}

func comprobarInterfacesSerializacionCerradas(t *testing.T, valor, destino any) {
	t.Helper()
	codificadorJSON, ok := valor.(json.Marshaler)
	if !ok {
		t.Fatal("falta json.Marshaler")
	}
	if _, err := codificadorJSON.MarshalJSON(); !esErrorSerializacionGobierno(err) {
		t.Fatalf("MarshalJSON no esta cerrado: %v", err)
	}
	decodificadorJSON, ok := destino.(json.Unmarshaler)
	if !ok {
		t.Fatal("falta json.Unmarshaler")
	}
	if err := decodificadorJSON.UnmarshalJSON(nil); !esErrorSerializacionGobierno(err) {
		t.Fatalf("UnmarshalJSON no esta cerrado: %v", err)
	}

	codificadorTexto, ok := valor.(encoding.TextMarshaler)
	if !ok {
		t.Fatal("falta encoding.TextMarshaler")
	}
	if _, err := codificadorTexto.MarshalText(); !esErrorSerializacionGobierno(err) {
		t.Fatalf("MarshalText no esta cerrado: %v", err)
	}
	decodificadorTexto, ok := destino.(encoding.TextUnmarshaler)
	if !ok {
		t.Fatal("falta encoding.TextUnmarshaler")
	}
	if err := decodificadorTexto.UnmarshalText(nil); !esErrorSerializacionGobierno(err) {
		t.Fatalf("UnmarshalText no esta cerrado: %v", err)
	}

	codificadorBinario, ok := valor.(encoding.BinaryMarshaler)
	if !ok {
		t.Fatal("falta encoding.BinaryMarshaler")
	}
	if _, err := codificadorBinario.MarshalBinary(); !esErrorSerializacionGobierno(err) {
		t.Fatalf("MarshalBinary no esta cerrado: %v", err)
	}
	decodificadorBinario, ok := destino.(encoding.BinaryUnmarshaler)
	if !ok {
		t.Fatal("falta encoding.BinaryUnmarshaler")
	}
	if err := decodificadorBinario.UnmarshalBinary(nil); !esErrorSerializacionGobierno(err) {
		t.Fatalf("UnmarshalBinary no esta cerrado: %v", err)
	}

	codificadorGob, ok := valor.(gob.GobEncoder)
	if !ok {
		t.Fatal("falta gob.GobEncoder")
	}
	if _, err := codificadorGob.GobEncode(); !esErrorSerializacionGobierno(err) {
		t.Fatalf("GobEncode no esta cerrado: %v", err)
	}
	decodificadorGob, ok := destino.(gob.GobDecoder)
	if !ok {
		t.Fatal("falta gob.GobDecoder")
	}
	if err := decodificadorGob.GobDecode(nil); !esErrorSerializacionGobierno(err) {
		t.Fatalf("GobDecode no esta cerrado: %v", err)
	}

	codificadorXML, ok := valor.(xml.Marshaler)
	if !ok {
		t.Fatal("falta xml.Marshaler")
	}
	if err := codificadorXML.MarshalXML(xml.NewEncoder(&bytes.Buffer{}), xml.StartElement{}); !esErrorSerializacionGobierno(err) {
		t.Fatalf("MarshalXML no esta cerrado: %v", err)
	}
	decodificadorXML, ok := destino.(xml.Unmarshaler)
	if !ok {
		t.Fatal("falta xml.Unmarshaler")
	}
	if err := decodificadorXML.UnmarshalXML(xml.NewDecoder(bytes.NewReader(nil)), xml.StartElement{}); !esErrorSerializacionGobierno(err) {
		t.Fatalf("UnmarshalXML no esta cerrado: %v", err)
	}
}

func esErrorSerializacionGobierno(err error) bool {
	return errors.Is(err, ErrSerializacionGobiernoConvocatoriaProhibida) ||
		errors.Is(err, ErrSerializacionIdempotenciaConvocatoria) ||
		errors.Is(err, ErrSerializacionMotivoGobiernoConvocatoria) ||
		errors.Is(err, ErrSerializacionVerificacionConvocatoria)
}
