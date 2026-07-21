package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	cbor "github.com/fxamacker/cbor/v2"
	"gopkg.in/yaml.v3"
)

type codificadorCBORGobiernoConvocatoria interface {
	MarshalCBOR() ([]byte, error)
}

type decodificadorCBORGobiernoConvocatoria interface {
	UnmarshalCBOR([]byte) error
}

type codificadorYAMLGobiernoConvocatoria interface {
	MarshalYAML() (any, error)
}

type decodificadorYAMLGobiernoConvocatoria interface {
	UnmarshalYAML(func(any) error) error
}

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
		{"solicitud_semantica_motivo", SolicitudSemanticaMotivoGobiernoConvocatoria{}, &SolicitudSemanticaMotivoGobiernoConvocatoria{}},
		{"solicitud_hsm_motivo", SolicitudComprometerMotivoGobiernoConvocatoria{}, &SolicitudComprometerMotivoGobiernoConvocatoria{}},
		{"hmac_motivo", HMACMotivoGobiernoConvocatoria{}, &HMACMotivoGobiernoConvocatoria{}},
		{"proyeccion_hmac_motivo", ProyeccionHMACMotivoGobiernoConvocatoriaDurable{}, &ProyeccionHMACMotivoGobiernoConvocatoriaDurable{}},
		{"datos_compromiso_motivo", DatosCompromisoMotivoGobiernoConvocatoria{}, &DatosCompromisoMotivoGobiernoConvocatoria{}},
		{"compromiso_motivo", CompromisoMotivoGobiernoConvocatoria{}, &CompromisoMotivoGobiernoConvocatoria{}},
		{"datos_solicitud_materializar_motivo", DatosSolicitudMaterializarSelladoMotivoGobiernoConvocatoria{}, &DatosSolicitudMaterializarSelladoMotivoGobiernoConvocatoria{}},
		{"solicitud_materializar_motivo", SolicitudMaterializarSelladoMotivoGobiernoConvocatoria{}, &SolicitudMaterializarSelladoMotivoGobiernoConvocatoria{}},
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
	compromiso := compromisoMotivoConvocatoriaPrueba(
		t, AccionCrearBorradorConvocatoria, version,
		version.CreadaPor, version.MotivoCreacion, 'a',
	)
	material, err := MaterialAltaBorradorConvocatoria(
		version, nil, nil, compromiso,
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
	datosCompromiso, err := compromiso.DatosParaMaterial()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(contenido, []byte(datosCompromiso.HuellaSolicitudSHA256)) {
		t.Fatal("el material V2 filtro la huella semantica cruda")
	}
	const esperado = `{"esquema":"bolsa.convocatoria.intencion.v2",` +
		`"accion":"bolsa.convocatoria.borrador.crear",` +
		`"estado_principal_nuevo":{"referencia":"proceso:bolsa:auxiliar-2026#1",` +
		`"revision":1,"huella_estado_sha256":"0622db0daf0af07fb9381417c983ec64a3ec248880cd58b583dd1ff92e759cac"},` +
		`"dominio_criptografico_motivo":"bolsa.convocatoria.motivo.v1",` +
		`"generacion_clave_motivo":3,` +
		`"huella_motivo_hmac_sha256":"hmac-sha256:motivo-gobierno-v3:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
	if string(contenido) != esperado {
		t.Fatalf("golden V2 distinto:\n%s", contenido)
	}
	const huellaEsperada = "1998d703597dcebc199daed2634cba504b9a01f665176001c0332a8c5638a702"
	if huella := fmt.Sprintf("%x", sha256.Sum256(contenido)); huella != huellaEsperada {
		t.Fatalf("huella golden V2 distinta: %s", huella)
	}
}

func TestMaterialV2RechazaV1YNoDeclaraHuellasCrudas(t *testing.T) {
	tiposYCampos := []struct {
		tipo  reflect.Type
		campo string
	}{
		{reflect.TypeOf(MaterialIntencionGobiernoConvocatoria{}), "HuellaSolicitudMotivoSHA256"},
		{reflect.TypeOf(ProyeccionHMACMotivoGobiernoConvocatoriaDurable{}), "HuellaEntradaSHA256"},
		{reflect.TypeOf(DatosAtestacionSelladoMotivoConvocatoria{}), "HuellaSolicitudSHA256"},
	}
	for _, caso := range tiposYCampos {
		if _, existe := caso.tipo.FieldByName(caso.campo); existe {
			t.Fatalf("%s conserva el campo crudo %s", caso.tipo.Name(), caso.campo)
		}
	}
	version := versionGobernadaPuertosPrueba(t)
	material, err := MaterialAltaBorradorConvocatoria(
		version, nil, nil, compromisoMotivoConvocatoriaPrueba(
			t, AccionCrearBorradorConvocatoria, version,
			version.CreadaPor, version.MotivoCreacion, 'a',
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	materialV1 := material
	materialV1.Esquema = "bolsa.convocatoria.intencion.v1"
	if err := materialV1.Validar(); !errors.Is(err, ErrMaterialIntencionConvocatoriaInvalido) {
		t.Fatalf("material V1 aceptado: %v", err)
	}
	contenidoV1, err := json.Marshal(materialV1)
	if err != nil {
		t.Fatal(err)
	}
	var reconstruidoV1 MaterialIntencionGobiernoConvocatoria
	if err := json.Unmarshal(contenidoV1, &reconstruidoV1); err != nil {
		t.Fatal(err)
	}
	if reconstruidoV1.Esquema != "bolsa.convocatoria.intencion.v1" {
		t.Fatal("la lectura reinterpreto silenciosamente el esquema V1")
	}
	if err := reconstruidoV1.Validar(); !errors.Is(err, ErrMaterialIntencionConvocatoriaInvalido) {
		t.Fatalf("material V1 reconstruido aceptado: %v", err)
	}
	contenido, err := json.Marshal(material)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(contenido), "huella_solicitud_motivo_sha256") {
		t.Fatalf("JSON V2 conserva clave cruda: %s", contenido)
	}
}

func TestBloqueoComunCierraCBORYAMLRealesSinFiltrarMotivo(t *testing.T) {
	const motivo = "MOTIVO-PRIVADO-CBOR-YAML-NO-FILTRAR"
	valor := SolicitudSemanticaMotivoGobiernoConvocatoria{Motivo: motivo}
	modoSinBinario, err := (cbor.EncOptions{
		BinaryMarshaler: cbor.BinaryMarshalerNone,
	}).EncMode()
	if err != nil {
		t.Fatal(err)
	}
	pruebas := []struct {
		nombre string
		fn     func() ([]byte, error)
	}{
		{"cbor", func() ([]byte, error) { return cbor.Marshal(valor) }},
		{"cbor_sin_binary_marshaler", func() ([]byte, error) { return modoSinBinario.Marshal(valor) }},
		{"cbor_decode", func() ([]byte, error) {
			destino := SolicitudSemanticaMotivoGobiernoConvocatoria{Motivo: motivo}
			return nil, cbor.Unmarshal([]byte{0xa0}, &destino)
		}},
		{"yaml", func() ([]byte, error) { return yaml.Marshal(valor) }},
		{"yaml_decode", func() ([]byte, error) {
			destino := SolicitudSemanticaMotivoGobiernoConvocatoria{Motivo: motivo}
			return nil, yaml.Unmarshal([]byte("{}"), &destino)
		}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			salida, err := prueba.fn()
			if !esErrorSerializacionGobierno(err) {
				t.Fatalf("serializador permitio valor interno: %v", err)
			}
			if strings.Contains(string(salida)+err.Error(), motivo) {
				t.Fatalf("serializador filtro el motivo: %q / %v", salida, err)
			}
		})
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

	codificadorCBOR, ok := valor.(codificadorCBORGobiernoConvocatoria)
	if !ok {
		t.Fatal("falta MarshalCBOR")
	}
	if _, err := codificadorCBOR.MarshalCBOR(); !esErrorSerializacionGobierno(err) {
		t.Fatalf("MarshalCBOR no esta cerrado: %v", err)
	}
	decodificadorCBOR, ok := destino.(decodificadorCBORGobiernoConvocatoria)
	if !ok {
		t.Fatal("falta UnmarshalCBOR")
	}
	if err := decodificadorCBOR.UnmarshalCBOR(nil); !esErrorSerializacionGobierno(err) {
		t.Fatalf("UnmarshalCBOR no esta cerrado: %v", err)
	}

	codificadorYAML, ok := valor.(codificadorYAMLGobiernoConvocatoria)
	if !ok {
		t.Fatal("falta MarshalYAML")
	}
	if _, err := codificadorYAML.MarshalYAML(); !esErrorSerializacionGobierno(err) {
		t.Fatalf("MarshalYAML no esta cerrado: %v", err)
	}
	decodificadorYAML, ok := destino.(decodificadorYAMLGobiernoConvocatoria)
	if !ok {
		t.Fatal("falta UnmarshalYAML")
	}
	if err := decodificadorYAML.UnmarshalYAML(nil); !esErrorSerializacionGobierno(err) {
		t.Fatalf("UnmarshalYAML no esta cerrado: %v", err)
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
