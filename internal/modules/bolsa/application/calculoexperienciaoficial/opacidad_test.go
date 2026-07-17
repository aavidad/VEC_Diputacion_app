package calculoexperienciaoficial

import (
	"context"
	"encoding"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
)

type codificadorGobPrueba interface{ GobEncode() ([]byte, error) }
type decodificadorGobPrueba interface{ GobDecode([]byte) error }
type codificadorCBORPrueba interface{ MarshalCBOR() ([]byte, error) }
type decodificadorCBORPrueba interface{ UnmarshalCBOR([]byte) error }
type codificadorYAMLPrueba interface{ MarshalYAML() (any, error) }
type decodificadorYAMLPrueba interface{ UnmarshalYAML(func(any) error) error }

func TestCapacidadesRedactanFormatoYLogsSinValoresInternos(t *testing.T) {
	valores, sensibles := valoresOpacosPrueba(t)
	var salida strings.Builder
	registrador := slog.New(slog.NewJSONHandler(&salida, nil))
	for indice, valor := range valores {
		_, _ = fmt.Fprintf(&salida, "%v|%+v|%#v|%s|%q|", valor, valor, valor, valor, valor)
		if stringer, ok := valor.(fmt.Stringer); ok {
			salida.WriteString(stringer.String())
		}
		if goStringer, ok := valor.(fmt.GoStringer); ok {
			salida.WriteString(goStringer.GoString())
		}
		if loggable, ok := valor.(slog.LogValuer); ok {
			registrador.Info("opaco", "indice", indice, "valor", loggable.LogValue())
		}
	}
	texto := salida.String()
	for _, sensible := range sensibles {
		if strings.Contains(texto, sensible) {
			t.Fatalf("formato o log filtro %q", sensible)
		}
	}
}

func TestCapacidadesBloqueanTodosLosCodificadores(t *testing.T) {
	valores, _ := valoresOpacosPrueba(t)
	for indice, valor := range valores {
		t.Run(fmt.Sprintf("valor_%02d", indice), func(t *testing.T) {
			if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionProhibida) {
				t.Fatalf("JSON permitido: %v", err)
			}
			_, errTexto := valor.(encoding.TextMarshaler).MarshalText()
			_, errBinario := valor.(encoding.BinaryMarshaler).MarshalBinary()
			_, errGob := valor.(codificadorGobPrueba).GobEncode()
			_, errCBOR := valor.(codificadorCBORPrueba).MarshalCBOR()
			exigirErrorSerializacionPrueba(t, errTexto)
			exigirErrorSerializacionPrueba(t, errBinario)
			exigirErrorSerializacionPrueba(t, errGob)
			exigirErrorSerializacionPrueba(t, errCBOR)
			_, errYAML := valor.(codificadorYAMLPrueba).MarshalYAML()
			if !errors.Is(errYAML, ErrSerializacionProhibida) {
				t.Fatalf("YAML permitido: %v", errYAML)
			}
			errXML := valor.(xml.Marshaler).MarshalXML(
				xml.NewEncoder(io.Discard), xml.StartElement{Name: xml.Name{Local: "valor"}},
			)
			if !errors.Is(errXML, ErrSerializacionProhibida) {
				t.Fatalf("XML permitido: %v", errXML)
			}
		})
	}
}

func TestCapacidadesBloqueanTodosLosDecodificadores(t *testing.T) {
	punteros := punterosOpacosPrueba(t)
	for indice, valor := range punteros {
		t.Run(fmt.Sprintf("valor_%02d", indice), func(t *testing.T) {
			if err := json.Unmarshal([]byte(`{}`), valor); !errors.Is(
				err, ErrSerializacionProhibida,
			) {
				t.Fatalf("JSON permitido: %v", err)
			}
			if err := valor.(encoding.TextUnmarshaler).UnmarshalText([]byte("x")); !errors.Is(
				err, ErrSerializacionProhibida,
			) {
				t.Fatalf("texto permitido: %v", err)
			}
			if err := valor.(encoding.BinaryUnmarshaler).UnmarshalBinary([]byte{1}); !errors.Is(
				err, ErrSerializacionProhibida,
			) {
				t.Fatalf("binario permitido: %v", err)
			}
			if err := valor.(decodificadorGobPrueba).GobDecode([]byte{1}); !errors.Is(
				err, ErrSerializacionProhibida,
			) {
				t.Fatalf("gob permitido: %v", err)
			}
			if err := valor.(decodificadorCBORPrueba).UnmarshalCBOR([]byte{1}); !errors.Is(
				err, ErrSerializacionProhibida,
			) {
				t.Fatalf("CBOR permitido: %v", err)
			}
			if err := valor.(decodificadorYAMLPrueba).UnmarshalYAML(
				func(any) error { return nil },
			); !errors.Is(err, ErrSerializacionProhibida) {
				t.Fatalf("YAML permitido: %v", err)
			}
			if err := valor.(xml.Unmarshaler).UnmarshalXML(
				xml.NewDecoder(strings.NewReader("<valor/>")),
				xml.StartElement{Name: xml.Name{Local: "valor"}},
			); !errors.Is(err, ErrSerializacionProhibida) {
				t.Fatalf("XML permitido: %v", err)
			}
		})
	}
}

func TestSolicitudConfirmacionNoExponeAliasDelResultadoCanonico(t *testing.T) {
	escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
	if _, err := escenario.servicio.Ejecutar(contextoFondoPrueba(), escenario.orden); err != nil {
		t.Fatal(err)
	}
	primera, err := escenario.confirmador.ultimaSolicitud.Datos()
	if err != nil || len(primera.ResultadoCanonico) == 0 {
		t.Fatal("solicitud durable incompleta")
	}
	original := primera.ResultadoCanonico[0]
	primera.ResultadoCanonico[0] ^= 0xff
	segunda, err := escenario.confirmador.ultimaSolicitud.Datos()
	if err != nil || segunda.ResultadoCanonico[0] != original {
		t.Fatal("la proyeccion permitio alterar el resultado durable")
	}
}

func valoresOpacosPrueba(t *testing.T) ([]any, []string) {
	t.Helper()
	escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
	salida := debePrueba(esquemaEjecutarPrueba(escenario))
	datosConfirmacion := debePrueba(escenario.confirmador.ultimaSolicitud.Datos())
	solicitudReconciliacion := debePrueba(NuevaSolicitudReconciliacionDuradera(
		escenario.datosOrden.CorrelacionEscritura, datosConfirmacion.Intencion,
	))
	datosReconciliacion := debePrueba(solicitudReconciliacion.Datos())
	intento := debePrueba(nuevoIntentoReconciliacion(
		perfilInternoAlto, escenario.datosOrden.CorrelacionEscritura,
		datosConfirmacion.Intencion, datosConfirmacion.Resultado,
	))
	indeterminado := *nuevoErrorConfirmacionIndeterminada(intento)
	valores := []any{
		bloqueoSerializacion{}, escenario.datosOrden, escenario.orden, salida,
		datosConfirmacion, escenario.confirmador.ultimaSolicitud,
		escenario.confirmador.ultimoResultado, datosReconciliacion,
		solicitudReconciliacion, intento, indeterminado,
	}
	correlacion, _ := escenario.datosOrden.CorrelacionEscritura.ValorCanonico()
	return valores, []string{
		escenario.datosOrden.Selector.SujetoPseudonimo.Referencia(),
		escenario.datosOrden.Selector.Convocatoria.Referencia(), correlacion,
		datosConfirmacion.HuellaResultadoSHA256,
	}
}

func punterosOpacosPrueba(t *testing.T) []any {
	t.Helper()
	valores, _ := valoresOpacosPrueba(t)
	resultado := make([]any, 0, len(valores))
	for _, valor := range valores {
		switch v := valor.(type) {
		case bloqueoSerializacion:
			resultado = append(resultado, &v)
		case DatosOrdenConfiable:
			resultado = append(resultado, &v)
		case OrdenCalculoExperienciaOficial:
			resultado = append(resultado, &v)
		case ResultadoEjecucion:
			resultado = append(resultado, &v)
		case DatosConfirmacionDuradera:
			resultado = append(resultado, &v)
		case SolicitudConfirmacionDuradera:
			resultado = append(resultado, &v)
		case ResultadoConfirmacionDuradera:
			resultado = append(resultado, &v)
		case DatosReconciliacionDuradera:
			resultado = append(resultado, &v)
		case SolicitudReconciliacionDuradera:
			resultado = append(resultado, &v)
		case IntentoReconciliacionCalculoOficial:
			resultado = append(resultado, &v)
		case ErrorConfirmacionIndeterminada:
			resultado = append(resultado, &v)
		}
	}
	return resultado
}

func exigirErrorSerializacionPrueba(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, ErrSerializacionProhibida) {
		t.Fatalf("codificador permitido: %v", err)
	}
}

func esquemaEjecutarPrueba(
	escenario escenarioServicioPrueba,
) (ResultadoEjecucion, error) {
	return escenario.servicio.Ejecutar(contextoFondoPrueba(), escenario.orden)
}

func contextoFondoPrueba() context.Context { return context.Background() }
