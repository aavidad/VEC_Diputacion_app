package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	bolsaapplication "vec-diputacion-granada/internal/modules/bolsa/application"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

type consultaAvisosBootstrapPrueba struct{ avisos []dominiobolsa.AvisoRRHH }

func (c consultaAvisosBootstrapPrueba) ContarAvisosRRHH(context.Context, time.Time) (map[string]int, error) {
	return map[string]int{dominiobolsa.AvisoSaltoOrden: len(c.avisos), dominiobolsa.AvisoTresAnos: 0}, nil
}

func (c consultaAvisosBootstrapPrueba) ListarAvisosRRHH(context.Context, time.Time, int, int) ([]dominiobolsa.AvisoRRHH, error) {
	return append([]dominiobolsa.AvisoRRHH(nil), c.avisos...), nil
}

func TestAvisosBolsaRRHHContratoMinimizado(t *testing.T) {
	ahora := time.Date(2026, 9, 23, 9, 30, 0, 0, time.UTC)
	servicio, err := bolsaapplication.NuevoServicioAvisosRRHH(consultaAvisosBootstrapPrueba{avisos: []dominiobolsa.AvisoRRHH{{
		Tipo: dominiobolsa.AvisoSaltoOrden, BolsaRef: "bolsa:opaca", Referencia: "aviso:opaco", Fecha: ahora,
		Detalle: map[string]any{"llamamiento_ref": "llamamiento:opaco", "participacion_ref": "participacion:opaca", "orden": 1, "orden_primero_llamado": 2},
	}}}, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	manejador := nuevoManejadorBolsasRRHHDesarrollo(nil)
	manejador.avisos = servicio
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, rutaAvisosBolsaRRHHDesarrollo+"?limite=20", nil))
	if respuesta.Code != http.StatusOK {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	if strings.Contains(respuesta.Body.String(), "nombre") || strings.Contains(respuesta.Body.String(), "documento") {
		t.Fatalf("el contrato filtra datos personales: %s", respuesta.Body.String())
	}
	var sobre struct {
		Data struct {
			Esquema string `json:"esquema"`
			Items   []struct {
				Tipo, Bolsa, Referencia string
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &sobre); err != nil {
		t.Fatal(err)
	}
	if sobre.Data.Esquema != "vec.bolsa.rrhh.avisos.v1" || len(sobre.Data.Items) != 1 {
		t.Fatalf("contrato inesperado: %#v", sobre.Data)
	}
}

func TestAvisosBolsaRRHHRechazaPaginacionInvalida(t *testing.T) {
	manejador := nuevoManejadorBolsasRRHHDesarrollo(nil)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, rutaAvisosBolsaRRHHDesarrollo+"?limite=101", nil))
	if respuesta.Code != http.StatusBadRequest {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
}
