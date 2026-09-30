package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type operadorEmisionAvisoPrueba struct{ emision ports.EmisionLlamamiento }

func (o operadorEmisionAvisoPrueba) EmitirLlamamiento(context.Context, ports.SolicitudEmitirLlamamiento) (ports.EmisionLlamamiento, error) {
	return o.emision, nil
}
func (o operadorEmisionAvisoPrueba) RecuperarLlamamiento(context.Context, ports.SolicitudRecuperarLlamamiento) (ports.EmisionLlamamiento, error) {
	return o.emision, nil
}

func TestHandlerEmisionAvisoConservaDTOYExcluyeEventoInterno(t *testing.T) {
	b, err := os.ReadFile("testdata/emision_aviso_pendiente.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Data ports.EmisionLlamamiento `json:"data"`
	}
	if err = json.Unmarshal(b, &fixture); err != nil {
		t.Fatal(err)
	}
	fixture.Data.AvisosExternos = []ports.AvisoExternoPendiente{{Evento: ports.EventoAvisoExterno{DestinatarioExternoRef: "can_interno_no_publicar"}}}
	for _, metodo := range []string{http.MethodPost, http.MethodGet} {
		t.Run(metodo, func(t *testing.T) {
			h, err := NuevoHandlerEmisionLlamamiento(preparadorEmisionPrueba{}, operadorEmisionAvisoPrueba{fixture.Data})
			if err != nil {
				t.Fatal(err)
			}
			cuerpo := `{"bolsa_ref":"bolsa:auxiliar:prueba","participaciones":["participacion:externa:prueba"],"configuracion":{"referencia":"NEC-AVISO-PRUEBA"}}`
			estado := http.StatusCreated
			ruta := RutaEmisionesLlamamiento
			if metodo == http.MethodGet {
				ruta += "?bolsa_ref=bolsa%3Aauxiliar%3Aprueba&clave_idempotencia=aviso-prueba-01"
				cuerpo = ""
				estado = http.StatusOK
			}
			r := httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
			r.Header.Set("Accept", "application/json")
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Idempotency-Key", "aviso-prueba-01")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != estado {
				t.Fatalf("estado: %d", w.Code)
			}
			var want, got any
			if json.Unmarshal(b, &want) != nil || json.Unmarshal(w.Body.Bytes(), &got) != nil || !reflect.DeepEqual(want, got) {
				t.Fatal("respuesta no conserva el contrato de emisión pública")
			}
			if strings.Contains(w.Body.String(), "avisos_externos") || strings.Contains(w.Body.String(), "can_interno_no_publicar") {
				t.Fatal("respuesta publica evento interno")
			}
		})
	}
}
