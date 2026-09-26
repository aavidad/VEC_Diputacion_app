package httpinterno

import (
	"encoding/json"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestPropuestasExpedienteJSON(t *testing.T) {
	if b, _ := json.Marshal(propuestasExpedienteJSON(nil)); string(b) != "[]" {
		t.Fatalf("sin propuestas: %s", b)
	}
	t0 := time.Date(2026, 9, 6, 1, 28, 30, 697897000, time.UTC)
	b, err := json.Marshal(estadoSeguimientoJSON(ports.EstadoSeguimientoExpediente{ExpedienteRef: "expediente:ct:a",
		Acreditada: &ports.EstadoIncorporacionAcreditada{Propuestas: []ports.EstadoPropuestaExpediente{
			{Orden: 1, VersionResultante: 7, ConfirmadaEn: t0, ReciboRef: "recibo:a",
				Sustitucion: &ports.SustitucionPropuestaExpediente{NoIncorporacionReciboRef: "recibo:ni", MotivoClave: "no_presentado", RegistradaEn: t0}},
			{Orden: 2, VersionResultante: 9, ConfirmadaEn: t0, ReciboRef: "recibo:b", Vigente: true}}}}))
	if err != nil {
		t.Fatal(err)
	}
	var salida struct {
		Propuestas []struct {
			Orden       int                `json:"orden"`
			Version     int                `json:"version_resultante"`
			Vigente     bool               `json:"vigente"`
			ReciboRef   string             `json:"recibo_ref"`
			Sustitucion *map[string]string `json:"sustitucion"`
		} `json:"propuestas"`
	}
	if json.Unmarshal(b, &salida) != nil || len(salida.Propuestas) != 2 || salida.Propuestas[0].Vigente ||
		salida.Propuestas[0].Sustitucion == nil || (*salida.Propuestas[0].Sustitucion)["motivo_clave"] != "no_presentado" ||
		!salida.Propuestas[1].Vigente || salida.Propuestas[1].Sustitucion != nil || salida.Propuestas[1].Version != 9 {
		t.Fatalf("JSON del seguimiento: %s", b)
	}
}
