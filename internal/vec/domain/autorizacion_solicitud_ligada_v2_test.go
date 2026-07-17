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
	datos.Correlacion = referenciaCorrelacionAutorizacionV2ParaPrueba(
		t,
		"correlacion_fedcba9876543210fedcba9876543210",
	)
	entregados, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	entregados.Recurso.Ambitos["unidad"] = "intervencion"
	entregados.Recurso.Atributos["estado"] = "eliminado"

	nuevos, err := solicitud.Datos()
	correlacionNueva, errCorrelacion := nuevos.Correlacion.ValorCanonico()
	nuevaHuella, errHuella := HuellaSHA256SolicitudAutorizacionV2(solicitud)
	if err != nil || errCorrelacion != nil || errHuella != nil || nuevaHuella != huella ||
		nuevos.Recurso.Ambitos["unidad"] != "seleccion" ||
		nuevos.Recurso.Atributos["estado"] != "presentado" ||
		nuevos.ReferenciaMotivo.EntradaClave != claveMotivoAutorizacionV2Prueba ||
		correlacionNueva != referenciaCorrelacionAutorizacionV2Prueba {
		t.Fatalf("la capacidad compartio estado mutable: datos=%+v err=%v huella=%q errHuella=%v", nuevos, err, nuevaHuella, errHuella)
	}
}

func TestSolicitudAutorizacionLigadaV2BloqueaCodecsYFormato(t *testing.T) {
	solicitud := solicitudHuellaAutorizacionV2Prueba(t)
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	correlacionRef, err := datos.Correlacion.ValorCanonico()
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
				strings.Contains(texto, correlacionRef) || strings.Contains(texto, datos.Recurso.Referencia) {
				t.Fatalf("formato filtro contenido: %q", texto)
			}
			valorLog := slog.AnyValue(valor).Resolve().String()
			if strings.Contains(valorLog, datos.ReferenciaMotivo.EntradaClave) ||
				strings.Contains(valorLog, correlacionRef) || strings.Contains(valorLog, datos.Recurso.Referencia) {
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

func TestSolicitudAutorizacionLigadaV2PrevalidaPresupuestoAntesDeClonar(t *testing.T) {
	base, err := solicitudHuellaAutorizacionV2Prueba(t).Datos()
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre string
		mutar  func(*DatosSolicitudAutorizacionLigadaV2)
	}{
		{"ambitos sobre limite", func(datos *DatosSolicitudAutorizacionLigadaV2) {
			datos.Recurso.Ambitos = make(map[string]string, maximoElementosAutorizacion+1)
			for indice := 0; indice <= maximoElementosAutorizacion; indice++ {
				datos.Recurso.Ambitos[fmt.Sprintf("ambito_%03d", indice)] = "valor"
			}
		}},
		{"atributo fuera de forma", func(datos *DatosSolicitudAutorizacionLigadaV2) {
			datos.Recurso.Atributos = map[string]string{"estado": strings.Repeat("x", 513)}
		}},
		{"roles de actor no permitidos", func(datos *DatosSolicitudAutorizacionLigadaV2) {
			datos.ContextoActor.Principal.Roles = make([]string, 100_000)
		}},
		{"vinculos sobre limite", func(datos *DatosSolicitudAutorizacionLigadaV2) {
			datos.ContextoActor.Instantanea.Vinculos = make(
				[]VinculoReferenciaContextoActor,
				maximoVinculosContextoActor+1,
			)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			datos := base
			caso.mutar(&datos)
			if err := prevalidarDatosSolicitudAutorizacionLigadaV2(datos); err == nil {
				t.Fatal("el preflight acepto una entrada fuera de presupuesto")
			}
			if _, err := NuevaSolicitudAutorizacionLigadaV2(datos); !errors.Is(err, ErrSolicitudAutorizacionLigadaV2Invalida) {
				t.Fatalf("el constructor acepto la entrada: %v", err)
			}
		})
	}
}
