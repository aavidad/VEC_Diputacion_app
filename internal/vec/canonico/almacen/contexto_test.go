package almacen

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestProyeccionContextoEsInternaNoSerializableYRedactada(t *testing.T) {
	t.Parallel()

	proyeccion := ProyeccionContextoOperacionAlmacen{
		OperacionRef:    "operacion:confidencial:1",
		AutorizacionRef: "autorizacion:confidencial:1",
		PasoRef:         PasoConfirmarCargaDirecta,
	}
	if _, err := json.Marshal(proyeccion); !errors.Is(err, ErrSerializacionContextoAlmacenProhibida) {
		t.Fatalf("serializacion JSON permitida: %v", err)
	}
	if _, err := proyeccion.MarshalText(); !errors.Is(err, ErrSerializacionContextoAlmacenProhibida) {
		t.Fatalf("serializacion textual permitida: %v", err)
	}
	var destino ProyeccionContextoOperacionAlmacen
	if err := destino.UnmarshalJSON([]byte(`{}`)); !errors.Is(err, ErrSerializacionContextoAlmacenProhibida) {
		t.Fatalf("deserializacion JSON permitida: %v", err)
	}
	if err := destino.UnmarshalText([]byte("dato")); !errors.Is(err, ErrSerializacionContextoAlmacenProhibida) {
		t.Fatalf("deserializacion textual permitida: %v", err)
	}

	representaciones := []string{
		proyeccion.String(),
		proyeccion.GoString(),
		fmt.Sprintf("%v", proyeccion),
		fmt.Sprintf("%#v", proyeccion),
		proyeccion.LogValue().String(),
		slog.Any("contexto", proyeccion).Value.String(),
	}
	for _, representacion := range representaciones {
		if strings.Contains(representacion, proyeccion.OperacionRef) ||
			strings.Contains(representacion, proyeccion.AutorizacionRef) {
			t.Fatalf("la representacion revela referencias internas: %q", representacion)
		}
	}
}

func TestPasosOperacionAlmacenConservanIdentificadoresCanonicos(t *testing.T) {
	t.Parallel()

	casos := map[PasoOperacionAlmacen]string{
		PasoPrepararCargaDirecta:  "01_preparar_carga_directa",
		PasoAbandonarCargaDirecta: "02_abandonar_carga_directa",
		PasoConfirmarCargaDirecta: "01_confirmar_carga_directa",
		PasoLeerParaAnalisis:      "01_leer_para_analisis",
		PasoAnalizarContenido:     "02_analizar_contenido",
		PasoPromover:              "01_promover",
		PasoCustodiarDecision:     "01_custodiar_decision",
		PasoCustodiarFirmado:      "01_custodiar_documento_firmado",
		PasoRetenerFirmado:        "01_retener_documento_firmado",
	}
	if len(casos) != 9 {
		t.Fatalf("identificadores de paso duplicados: %d", len(casos))
	}
	for paso, esperado := range casos {
		if string(paso) != esperado {
			t.Errorf("paso %q; esperado %q", paso, esperado)
		}
	}
}
