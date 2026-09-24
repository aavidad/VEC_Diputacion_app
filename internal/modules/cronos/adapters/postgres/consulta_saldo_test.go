package postgres

import (
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func TestDecodificarFuenteSaldoInternaConOrigenNoAcreditado(t *testing.T) {
	bruto := []byte(`{"empleado_ref":"emp_test","desde":"2026-09-24","hasta":"2026-09-24","zona_horaria":"Europe/Madrid","completo":true,"jornadas":[{"programacion_ref":"p1","fecha":"2026-09-24","turno_ref":"turno1","politica_version_ref":"v1","fuente_ref":"f1","zona_horaria":"Europe/Madrid","minutos_previstos":0,"version":1}],"marcajes":[{"marcaje_ref":"m1","movimiento":"entrada","instante_utc":"2026-09-24T06:00:00Z","canal":{"politica_version_ref":"v1","canal_ref":"c1","origen_ref":"o1","calidad_ref":"q1"},"tipo_origen":null}],"movimientos_saldo":[{"fecha":"2026-09-24","tipo":"previsto","delta_microsegundos":0,"fuentes":["p1/v1"]}]}`)
	f, err := decodificarFuenteSaldoInterna(bruto)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Marcajes) != 1 || f.Marcajes[0].TipoOrigen != nil || f.Marcajes[0].Canal.OrigenRef != "o1" || len(f.MovimientosSaldo) != 1 {
		t.Fatalf("fuente=%+v", f)
	}
	if _, err := decodificarFuenteSaldoInterna(append(bruto, []byte(`{"otro":true}`)...)); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal(err)
	}
}
