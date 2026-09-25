package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type consultorCambiosRRHHPrueba struct {
	consultorDetalleRRHHPrueba
	llamadasCambios int
}

func (c *consultorCambiosRRHHPrueba) ConsultarCambios(_ context.Context, solicitud ports.SolicitudDetalleRRHH) (ports.ResultadoConsultaCambiosRRHH, error) {
	c.llamadasCambios++
	c.solicitud = solicitud
	anterior, nuevo := "C2", "C1"
	return ports.ResultadoConsultaCambiosRRHH{ExpedienteRef: solicitud.ExpedienteRef(), VersionExpediente: 3,
		Cambios:             []ports.CambioExpedienteRRHH{{VersionExpediente: 2, RegistradaEn: time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC), OrigenVersion: "analisis_o3", OperacionRef: "op:ct:2", Ruta: "solicitud.grupo_subgrupo", ValorAnterior: &anterior, ValorNuevo: &nuevo}},
		ConsumoHuellaSHA256: strings.Repeat("b", 64), AuditoriaRef: "auditoria:rrhh:001", AuditoriaHuellaSHA256: strings.Repeat("a", 64)}, nil
}

// Petición RRHH p.4: el Accept de cambios usa la misma ruta y consulta
// autorizada que el detalle y no toca el detalle completo.
func TestManejadorConsultaDetalleRRHHSirveCambiosConSuAccept(t *testing.T) {
	consultor := &consultorCambiosRRHHPrueba{}
	manejador, err := NuevoManejadorConsultaDetalleRRHH(consultor)
	if err != nil {
		t.Fatal(err)
	}
	peticion := nuevaPeticionConsultaRRHHPrueba(RutaConsultaDetalleRRHH, cuerpoDetalleRRHHPrueba())
	peticion.Header.Set("Accept", AcceptCambiosExpedienteRRHH)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusOK || consultor.llamadasCambios != 1 || consultor.llamadas != 0 {
		t.Fatalf("estado=%d cambios=%d detalle=%d cuerpo=%s", respuesta.Code, consultor.llamadasCambios, consultor.llamadas, respuesta.Body)
	}
	var salida struct {
		Data struct {
			Esquema string           `json:"esquema"`
			Cambios []map[string]any `json:"cambios"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &salida); err != nil || salida.Data.Esquema != EsquemaCambiosExpedienteRRHH || len(salida.Data.Cambios) != 1 ||
		salida.Data.Cambios[0]["valor_anterior"] != "C2" || salida.Data.Cambios[0]["valor_nuevo"] != "C1" || salida.Data.Cambios[0]["ruta"] != "solicitud.grupo_subgrupo" {
		t.Fatalf("contrato inesperado: %s", respuesta.Body)
	}
	for _, prohibido := range []string{"auditoria", "consumo", "operacion_ref", "op:ct:2"} {
		if strings.Contains(respuesta.Body.String(), prohibido) {
			t.Fatalf("se publicó %q: %s", prohibido, respuesta.Body)
		}
	}
	comprobarCabecerasConsultaRRHH(t, respuesta)
}

func TestManejadorConsultaDetalleRRHHCambiosSinLectorNoDisponible(t *testing.T) {
	consultor := &consultorDetalleRRHHPrueba{detalle: detalleRRHHPrueba()}
	manejador, _ := NuevoManejadorConsultaDetalleRRHH(consultor)
	peticion := nuevaPeticionConsultaRRHHPrueba(RutaConsultaDetalleRRHH, cuerpoDetalleRRHHPrueba())
	peticion.Header.Set("Accept", AcceptCambiosExpedienteRRHH)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusServiceUnavailable || consultor.llamadas != 0 {
		t.Fatalf("estado=%d llamadas=%d", respuesta.Code, consultor.llamadas)
	}
	sinTipo := nuevaPeticionConsultaRRHHPrueba(RutaConsultaDetalleRRHH, cuerpoDetalleRRHHPrueba())
	sinTipo.Header.Set("Accept", AcceptCambiosExpedienteRRHH)
	sinTipo.Header.Set("Content-Type", "text/plain")
	respuesta = httptest.NewRecorder()
	cambios := &consultorCambiosRRHHPrueba{}
	manejadorCambios, _ := NuevoManejadorConsultaDetalleRRHH(cambios)
	manejadorCambios.ServeHTTP(respuesta, sinTipo)
	if respuesta.Code == http.StatusOK || cambios.llamadasCambios != 0 {
		t.Fatalf("tipo no admitido aceptado: %d", respuesta.Code)
	}
}
