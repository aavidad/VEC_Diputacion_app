package domain

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

func TestSolicitudAutorizacionLigadaV2ClonaEntradaYDatos(t *testing.T) {
	original := solicitudHuellaAutorizacionV2Prueba(t)
	datos, err := original.Datos()
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := NuevaSolicitudAutorizacionLigadaV2(datos)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := HuellaSHA256SolicitudAutorizacionV2(solicitud)
	if err != nil {
		t.Fatal(err)
	}

	datos.Recurso.Ambitos["unidad"] = "nominas"
	datos.Recurso.Atributos["estado"] = "borrado"
	datos.ReferenciaMotivo.EntradaClave = claveMotivoAutorizacionV2Alternativa
	datos.CorrelacionRef = "correlacion_fedcba9876543210fedcba9876543210"
	entregados, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	entregados.Recurso.Ambitos["unidad"] = "intervencion"
	entregados.Recurso.Atributos["estado"] = "eliminado"

	nuevos, err := solicitud.Datos()
	nuevaHuella, errHuella := HuellaSHA256SolicitudAutorizacionV2(solicitud)
	if err != nil || errHuella != nil || nuevaHuella != huella ||
		nuevos.Recurso.Ambitos["unidad"] != "seleccion" ||
		nuevos.Recurso.Atributos["estado"] != "presentado" ||
		nuevos.ReferenciaMotivo.EntradaClave != claveMotivoAutorizacionV2Prueba ||
		nuevos.CorrelacionRef != referenciaCorrelacionAutorizacionV2Prueba {
		t.Fatalf("la capacidad compartio estado mutable: datos=%+v err=%v huella=%q errHuella=%v", nuevos, err, nuevaHuella, errHuella)
	}
}

func TestSolicitudAutorizacionLigadaV2BloqueaCodecsYFormato(t *testing.T) {
	solicitud := solicitudHuellaAutorizacionV2Prueba(t)
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	for nombre, valor := range map[string]any{"solicitud": solicitud, "datos": datos} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida) {
				t.Fatalf("JSON no bloqueado: %v", err)
			}
			if _, err := xml.Marshal(valor); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida) {
				t.Fatalf("XML no bloqueado: %v", err)
			}
			var destino bytes.Buffer
			if err := gob.NewEncoder(&destino).Encode(valor); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida) {
				t.Fatalf("Gob no bloqueado: %v", err)
			}
			if _, err := valor.(interface{ MarshalBinary() ([]byte, error) }).MarshalBinary(); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida) {
				t.Fatalf("binario no bloqueado: %v", err)
			}
			if _, err := valor.(interface{ MarshalText() ([]byte, error) }).MarshalText(); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida) {
				t.Fatalf("texto no bloqueado: %v", err)
			}
			if _, err := valor.(interface{ MarshalCBOR() ([]byte, error) }).MarshalCBOR(); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida) {
				t.Fatalf("CBOR no bloqueado: %v", err)
			}
			if _, err := valor.(interface{ MarshalYAML() (any, error) }).MarshalYAML(); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida) {
				t.Fatalf("YAML no bloqueado: %v", err)
			}
			texto := fmt.Sprintf("%v %+v %#v", valor, valor, valor)
			if strings.Contains(texto, datos.ReferenciaMotivo.EntradaClave) ||
				strings.Contains(texto, datos.CorrelacionRef) || strings.Contains(texto, datos.Recurso.Referencia) {
				t.Fatalf("formato filtro contenido: %q", texto)
			}
			valorLog := slog.AnyValue(valor).Resolve().String()
			if strings.Contains(valorLog, datos.ReferenciaMotivo.EntradaClave) ||
				strings.Contains(valorLog, datos.CorrelacionRef) || strings.Contains(valorLog, datos.Recurso.Referencia) {
				t.Fatalf("slog filtro contenido: %q", valorLog)
			}
		})
	}

	for nombre, destino := range map[string]any{
		"solicitud": &SolicitudAutorizacionLigadaV2{},
		"datos":     &DatosSolicitudAutorizacionLigadaV2{},
	} {
		t.Run("decodificar "+nombre, func(t *testing.T) {
			if err := json.Unmarshal([]byte(`{}`), destino); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida) {
				t.Fatalf("JSON no bloqueado: %v", err)
			}
			if err := xml.Unmarshal([]byte(`<solicitud/>`), destino); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida) {
				t.Fatalf("XML no bloqueado: %v", err)
			}
			if err := destino.(interface{ GobDecode([]byte) error }).GobDecode(nil); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida) {
				t.Fatalf("Gob no bloqueado: %v", err)
			}
			if err := destino.(interface{ UnmarshalBinary([]byte) error }).UnmarshalBinary(nil); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida) {
				t.Fatalf("binario no bloqueado: %v", err)
			}
			if err := destino.(interface{ UnmarshalText([]byte) error }).UnmarshalText(nil); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida) {
				t.Fatalf("texto no bloqueado: %v", err)
			}
			if err := destino.(interface{ UnmarshalCBOR([]byte) error }).UnmarshalCBOR([]byte{0xa0}); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida) {
				t.Fatalf("CBOR no bloqueado: %v", err)
			}
			if err := destino.(interface{ UnmarshalYAML(func(any) error) error }).UnmarshalYAML(
				func(any) error { return nil },
			); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida) {
				t.Fatalf("YAML no bloqueado: %v", err)
			}
		})
	}
}

func TestSolicitudAutorizacionLigadaV2ValorCeroFallaCerrado(t *testing.T) {
	var solicitud SolicitudAutorizacionLigadaV2
	if _, err := solicitud.Datos(); !errors.Is(err, ErrSolicitudAutorizacionLigadaV2Invalida) ||
		!errors.Is(err, ErrSolicitudAutorizacionInvalida) {
		t.Fatalf("valor cero aceptado: %v", err)
	}
}
