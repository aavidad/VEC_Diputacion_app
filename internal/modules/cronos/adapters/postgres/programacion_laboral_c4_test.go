package postgres

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func TestDecodificarProgramacionC4NoConfundeAusenciaYTurnoNocturno(t *testing.T) {
	empleado, fecha := "emp_0123456789abcdefghijkl", "2026-06-15"
	sinFuente := []byte(`{"estado":"no_determinable","fecha":"2026-06-15"}`)
	if got, err := decodificarProgramacionC4(sinFuente, empleado, fecha); got != nil || err != nil {
		t.Fatalf("sin fuente: %+v, %v", got, err)
	}
	positivo := map[string]any{
		"estado": "programado", "fecha": fecha, "empleado_ref": empleado,
		"relacion_ref": "rel_0123456789abcdefghijkl", "centro_ref": "centro:demo", "regimen_ref": "regimen:demo",
		"programacion_ref": "programacion:cronos:demo", "turno_ref": "turno_noche", "politica_version_ref": "politica:demo",
		"fuente_ref": "fuente:cronos:demo", "version": 1, "fuente_version": 1, "minutos_programados": 480,
		"inicio_utc": "2026-06-15T18:00:00Z", "fin_utc": "2026-06-16T02:00:00Z", "publicada_en": "2026-06-01T10:00:00Z",
	}
	bruto, err := json.Marshal(positivo)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodificarProgramacionC4(bruto, empleado, fecha)
	if err != nil || got == nil || got.TurnoRef != "turno_noche" || got.FinUTC.Day() != 16 || got.InicioUTC.Location() != time.UTC {
		t.Fatalf("turno nocturno: %+v, %v", got, err)
	}
	positivo["empleado_ref"] = "emp_ajenoajenoajenoajenoajeno"
	bruto, _ = json.Marshal(positivo)
	if _, err = decodificarProgramacionC4(bruto, empleado, fecha); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatalf("empleado ajeno: %v", err)
	}
}
