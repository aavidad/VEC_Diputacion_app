package ports_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestDetalleRRHHIndicaCantidadSingularLegadaYPluralDeclarada(t *testing.T) {
	b, err := os.ReadFile("../domain/testdata/expediente_solicitud_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var e domain.Expediente
	if err := json.Unmarshal(b, &e); err != nil {
		t.Fatal(err)
	}
	registrada := e.ActualizadoEn.Add(time.Minute)
	recibo := reciboDetalleMinimizadoPrueba(t, e.Referencia, e.Version, registrada)
	detalle, err := ports.NuevoDetalleExpedienteRRHH(e, recibo)
	if err != nil || detalle.Solicitud.NumeroPersonas != 1 {
		t.Fatalf("legado singular sin cantidad de presentación: %d %v", detalle.Solicitud.NumeroPersonas, err)
	}
	e.Solicitud.NumeroPersonas, e.Solicitud.JornadaMinutos, e.Solicitud.PuestoSolicitado = 2, 1500, "9"
	detalle, err = ports.NuevoDetalleExpedienteRRHH(e, recibo)
	if err != nil || detalle.Solicitud.NumeroPersonas != 2 {
		t.Fatalf("plural declarado perdió cantidad: %d %v", detalle.Solicitud.NumeroPersonas, err)
	}
}
