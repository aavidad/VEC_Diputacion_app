package httpinterno

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestDetalleRRHHDisponibilidadFichaVinculadaALaLectura(t *testing.T) {
	base := detalleRRHHPrueba()
	if proyectarDetalleRRHH(base).CapacidadesFicha != nil {
		t.Fatal("una lectura sin metadata no debe fabricar disponibilidad")
	}
	for _, estado := range []ports.EstadoDisponibilidadBorradoresRRHH{
		ports.BorradoresRRHHSinMontaje, ports.BorradoresRRHHIndisponible,
	} {
		detalle := base
		detalle.EstadoBorradoresPublicados = estado
		salida := proyectarDetalleRRHH(detalle)
		if salida.CapacidadesFicha == nil || salida.CapacidadesFicha.BorradoresPublicados.Estado != string(estado) ||
			salida.CapacidadesFicha.BorradoresPublicados.ExpedienteRef != base.Resumen.ExpedienteRef ||
			salida.CapacidadesFicha.BorradoresPublicados.VersionObservada != base.Resumen.Version {
			t.Fatalf("metadata de otra lectura o versión: %+v", salida.CapacidadesFicha)
		}
	}
}

func TestDetalleRRHHHTTPIncluyeDisponibilidadSoloTrasConsultaValida(t *testing.T) {
	detalle := detalleRRHHPrueba()
	detalle.EstadoBorradoresPublicados = ports.BorradoresRRHHSinMontaje
	lector := &consultorDetalleRRHHPrueba{detalle: detalle}
	h, err := NuevoManejadorConsultaDetalleRRHH(lector)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, nuevaPeticionConsultaRRHHPrueba(RutaConsultaDetalleRRHH, cuerpoDetalleRRHHPrueba()))
	if w.Code != http.StatusOK || lector.llamadas != 1 {
		t.Fatalf("lectura=%d, estado=%d, cuerpo=%s", lector.llamadas, w.Code, w.Body.String())
	}
	var salida struct {
		Data struct {
			CapacidadesFicha capacidadesFichaRRHHJSON `json:"capacidades_ficha"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &salida); err != nil {
		t.Fatal(err)
	}
	capacidad := salida.Data.CapacidadesFicha.BorradoresPublicados
	if capacidad.Estado != "sin_montaje" || capacidad.ExpedienteRef != detalle.Resumen.ExpedienteRef || capacidad.VersionObservada != detalle.Resumen.Version {
		t.Fatalf("disponibilidad sin anclaje a la lectura: %+v", capacidad)
	}
}
